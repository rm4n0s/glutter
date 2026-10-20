// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package gestures

import (
	"slices"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// GestureDoubleTapCallback is called when a double tap happens.
//
// Flutter: gestures/multitap.dart GestureDoubleTapCallback
type GestureDoubleTapCallback func()

// GestureMultiTapDownCallback is called when a pointer that might tap touches
// the screen, with the pointer.
//
// Flutter: gestures/multitap.dart GestureMultiTapDownCallback
type GestureMultiTapDownCallback func(pointer int, details TapDownDetails)

// GestureMultiTapUpCallback is called when a pointer that tapped is lifted.
//
// Flutter: gestures/multitap.dart GestureMultiTapUpCallback
type GestureMultiTapUpCallback func(pointer int, details TapUpDetails)

// GestureMultiTapCallback is called when a pointer taps.
//
// Flutter: gestures/multitap.dart GestureMultiTapCallback
type GestureMultiTapCallback func(pointer int)

// GestureMultiTapCancelCallback is called when the pointer that sent a tap down
// will not tap.
//
// Flutter: gestures/multitap.dart GestureMultiTapCancelCallback
type GestureMultiTapCancelCallback func(pointer int)

// tapTracker tracks one pointer of a tap, for the recognizers of several
// taps.
type tapTracker struct {
	binding               *GestureBinding
	gestureSettings       foundation.Opt[DeviceGestureSettings]
	pointer               int
	entry                 GestureArenaEntry
	initialGlobalPosition ui.Offset
	initialButtons        int
	minTimeElapsed        bool
	isTrackingPointer     bool
	handle                PointerRouteHandle
}

func newTapTracker(binding *GestureBinding, event *PointerDownEvent, entry GestureArenaEntry, doubleTapMinTime time.Duration,
	settings foundation.Opt[DeviceGestureSettings]) *tapTracker {
	t := &tapTracker{
		binding: binding, gestureSettings: settings, pointer: event.Pointer(), entry: entry,
		initialGlobalPosition: event.Position(), initialButtons: event.Buttons(),
	}
	binding.NewTimer(doubleTapMinTime, func() { t.minTimeElapsed = true })
	return t
}

func (t *tapTracker) startTrackingPointer(route PointerRoute, transform *ui.Matrix4) {
	if !t.isTrackingPointer {
		t.isTrackingPointer = true
		t.handle = t.binding.PointerRouter.AddRoute(t.pointer, route, transform)
	}
}

func (t *tapTracker) stopTrackingPointer() {
	if t.isTrackingPointer {
		t.isTrackingPointer = false
		t.binding.PointerRouter.RemoveRoute(t.pointer, t.handle)
	}
}

func (t *tapTracker) isWithinGlobalTolerance(event PointerEvent, tolerance float64) bool {
	return event.Position().Sub(t.initialGlobalPosition).Distance() <= tolerance
}

func (t *tapTracker) hasElapsedMinTime() bool { return t.minTimeElapsed }

func (t *tapTracker) hasSameButton(event *PointerDownEvent) bool {
	return event.Buttons() == t.initialButtons
}

// DoubleTapGestureRecognizerOpts are the optional parameters of
// NewDoubleTapGestureRecognizer.
//
// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer.new
type DoubleTapGestureRecognizerOpts struct {
	// DebugOwner is the object that owns the recognizer.
	DebugOwner any
	// SupportedDevices are the kinds of pointer the recognizer accepts; nil for
	// all.
	SupportedDevices []ui.PointerDeviceKind
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed;
	// the default allows the primary button only.
	AllowedButtonsFilter AllowedButtonsFilter
}

// DoubleTapGestureRecognizer recognizes two taps in quick succession, near each
// other, with the same button.
//
// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer
type DoubleTapGestureRecognizer struct {
	GestureRecognizerBase

	// OnDoubleTapDown is called when the second pointer touches the screen.
	//
	// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer.onDoubleTapDown
	OnDoubleTapDown GestureTapDownCallback
	// OnDoubleTap is called when the double tap happens.
	//
	// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer.onDoubleTap
	OnDoubleTap GestureDoubleTapCallback
	// OnDoubleTapCancel is called when a pointer that sent OnDoubleTapDown will
	// not complete a double tap.
	//
	// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer.onDoubleTapCancel
	OnDoubleTapCancel GestureTapCancelCallback

	doubleTapTimer *Timer
	firstTap       *tapTracker
	trackers       map[int]*tapTracker
}

// NewDoubleTapGestureRecognizer creates a recognizer that works with the
// binding.
//
// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer.new
func NewDoubleTapGestureRecognizer(binding *GestureBinding, opts ...DoubleTapGestureRecognizerOpts) *DoubleTapGestureRecognizer {
	o := oneOpts(opts)
	if o.AllowedButtonsFilter == nil {
		o.AllowedButtonsFilter = func(b int) bool { return b == PrimaryButton }
	}
	r := &DoubleTapGestureRecognizer{trackers: map[int]*tapTracker{}}
	r.Init(binding, r, GestureRecognizerOpts(o))
	return r
}

// IsPointerAllowed allows a pointer if a callback is set; a pointer that is not
// allowed resets the recognizer.
//
// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer.isPointerAllowed
func (r *DoubleTapGestureRecognizer) IsPointerAllowed(event *PointerDownEvent) bool {
	if r.firstTap == nil && r.OnDoubleTapDown == nil && r.OnDoubleTap == nil && r.OnDoubleTapCancel == nil {
		return false
	}
	allowed := r.GestureRecognizerBase.IsPointerAllowed(event)
	if !allowed {
		r.reset()
	}
	return allowed
}

// AddAllowedPointer tracks the tap, or starts over if it is too early or has
// another button.
//
// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer.addAllowedPointer
func (r *DoubleTapGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	if r.firstTap != nil {
		if !r.firstTap.isWithinGlobalTolerance(event, DoubleTapSlop) {
			// Ignore out-of-bounds second taps.
			return
		} else if !r.firstTap.hasElapsedMinTime() || !r.firstTap.hasSameButton(event) {
			// Restart when the second tap is too close to the first, or has
			// another button.
			r.reset()
			r.trackTap(event)
			return
		} else if r.OnDoubleTapDown != nil {
			details := NewTapDownDetails(TapDownOpts{
				GlobalPosition: event.Position(), LocalPosition: foundation.Some(event.LocalPosition()),
				Kind: foundation.Some(r.GetKindForPointer(event.Pointer())),
			})
			r.InvokeCallback("onDoubleTapDown", func() { r.OnDoubleTapDown(details) })
		}
	}
	r.trackTap(event)
}

func (r *DoubleTapGestureRecognizer) trackTap(event *PointerDownEvent) {
	r.stopDoubleTapTimer()
	tracker := newTapTracker(r.binding, event, r.binding.GestureArena.Add(event.Pointer(), r), DoubleTapMinTime, r.GestureSettings)
	r.trackers[event.Pointer()] = tracker
	tracker.startTrackingPointer(r.handleEvent, event.Transform())
}

func (r *DoubleTapGestureRecognizer) handleEvent(event PointerEvent) {
	tracker := r.trackers[event.Pointer()]
	switch event.(type) {
	case *PointerUpEvent:
		if r.firstTap == nil {
			r.registerFirstTap(tracker)
		} else {
			r.registerSecondTap(tracker)
		}
	case *PointerMoveEvent:
		if !tracker.isWithinGlobalTolerance(event, DoubleTapTouchSlop) {
			r.reject(tracker)
		}
	case *PointerCancelEvent:
		r.reject(tracker)
	}
}

// AcceptGesture does nothing.
//
// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer.acceptGesture
func (r *DoubleTapGestureRecognizer) AcceptGesture(int) {}

// RejectGesture rejects the tracker of the pointer.
//
// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer.rejectGesture
func (r *DoubleTapGestureRecognizer) RejectGesture(pointer int) {
	tracker := r.trackers[pointer]
	// If tracker isn't in the list, check if this is the first tap tracker.
	if tracker == nil && r.firstTap != nil && r.firstTap.pointer == pointer {
		tracker = r.firstTap
	}
	// If tracker is still nil, we rejected ourselves already.
	if tracker != nil {
		r.reject(tracker)
	}
}

func (r *DoubleTapGestureRecognizer) reject(tracker *tapTracker) {
	delete(r.trackers, tracker.pointer)
	tracker.entry.Resolve(GestureRejected)
	r.freezeTracker(tracker)
	// If the first tap is in progress, the double tap recognizer is still
	// possible.
	if r.firstTap != nil {
		if tracker == r.firstTap {
			r.reset()
		} else {
			r.checkCancel()
			if len(r.trackers) == 0 {
				r.reset()
			}
		}
	}
}

// Dispose resets the recognizer.
//
// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer.dispose
func (r *DoubleTapGestureRecognizer) Dispose() {
	r.reset()
	r.GestureRecognizerBase.Dispose()
}

func (r *DoubleTapGestureRecognizer) reset() {
	r.stopDoubleTapTimer()
	if r.firstTap != nil {
		if len(r.trackers) != 0 {
			r.checkCancel()
		}
		// Note, order is important below in order for the resolve -> reject ->
		// reset chain to work properly.
		tracker := r.firstTap
		r.firstTap = nil
		r.reject(tracker)
		r.binding.GestureArena.Release(tracker.pointer)
	}
	r.clearTrackers()
}

func (r *DoubleTapGestureRecognizer) registerFirstTap(tracker *tapTracker) {
	r.startDoubleTapTimer()
	r.binding.GestureArena.Hold(tracker.pointer)
	// Note, order is important below in order for the clear -> reject chain to
	// work properly.
	r.freezeTracker(tracker)
	delete(r.trackers, tracker.pointer)
	r.clearTrackers()
	r.firstTap = tracker
}

func (r *DoubleTapGestureRecognizer) registerSecondTap(tracker *tapTracker) {
	r.firstTap.entry.Resolve(GestureAccepted)
	tracker.entry.Resolve(GestureAccepted)
	r.freezeTracker(tracker)
	delete(r.trackers, tracker.pointer)
	r.checkUp()
	r.reset()
}

func (r *DoubleTapGestureRecognizer) clearTrackers() {
	for _, tracker := range r.trackerList() {
		r.reject(tracker)
	}
}

func (r *DoubleTapGestureRecognizer) trackerList() []*tapTracker {
	pointers := make([]int, 0, len(r.trackers))
	for p := range r.trackers {
		pointers = append(pointers, p)
	}
	slices.Sort(pointers)
	list := make([]*tapTracker, len(pointers))
	for i, p := range pointers {
		list[i] = r.trackers[p]
	}
	return list
}

func (r *DoubleTapGestureRecognizer) freezeTracker(tracker *tapTracker) {
	tracker.stopTrackingPointer()
}

func (r *DoubleTapGestureRecognizer) startDoubleTapTimer() {
	if r.doubleTapTimer == nil {
		r.doubleTapTimer = r.binding.NewTimer(DoubleTapTimeout, r.reset)
	}
}

func (r *DoubleTapGestureRecognizer) stopDoubleTapTimer() {
	if r.doubleTapTimer != nil {
		r.doubleTapTimer.Cancel()
		r.doubleTapTimer = nil
	}
}

func (r *DoubleTapGestureRecognizer) checkUp() {
	if r.OnDoubleTap != nil {
		r.InvokeCallback("onDoubleTap", r.OnDoubleTap)
	}
}

func (r *DoubleTapGestureRecognizer) checkCancel() {
	if r.OnDoubleTapCancel != nil {
		r.InvokeCallback("onDoubleTapCancel", r.OnDoubleTapCancel)
	}
}

// DebugDescription is "double tap".
//
// Flutter: gestures/multitap.dart DoubleTapGestureRecognizer.debugDescription
func (r *DoubleTapGestureRecognizer) DebugDescription() string { return "double tap" }

// tapGesture tracks one tap of a MultiTapGestureRecognizer.
type tapGesture struct {
	*tapTracker
	recognizer    *MultiTapGestureRecognizer
	wonArena      bool
	timer         *Timer
	lastPosition  OffsetPair
	finalPosition foundation.Opt[OffsetPair]
}

func newTapGesture(r *MultiTapGestureRecognizer, event *PointerDownEvent, longTapDelay time.Duration,
	settings foundation.Opt[DeviceGestureSettings]) *tapGesture {
	g := &tapGesture{recognizer: r, lastPosition: NewOffsetPairFromEventPosition(event)}
	g.tapTracker = newTapTracker(r.binding, event, r.binding.GestureArena.Add(event.Pointer(), r), DoubleTapMinTime, settings)
	g.startTrackingPointer(g.handleEvent, event.Transform())
	if longTapDelay > 0 {
		g.timer = r.binding.NewTimer(longTapDelay, func() {
			g.timer = nil
			r.dispatchLongTap(event.Pointer(), g.lastPosition)
		})
	}
	return g
}

func (g *tapGesture) handleEvent(event PointerEvent) {
	switch event.(type) {
	case *PointerMoveEvent:
		if !g.isWithinGlobalTolerance(event, ComputeHitSlop(event.Kind(), g.gestureSettings)) {
			g.cancel()
		} else {
			g.lastPosition = NewOffsetPairFromEventPosition(event)
		}
	case *PointerCancelEvent:
		g.cancel()
	case *PointerUpEvent:
		g.stopTracking()
		g.finalPosition = foundation.Some(NewOffsetPairFromEventPosition(event))
		g.check()
	}
}

// stopTracking is the stopTrackingPointer of the tap gesture, which also stops
// the long tap timer.
func (g *tapGesture) stopTracking() {
	if g.timer != nil {
		g.timer.Cancel()
		g.timer = nil
	}
	g.stopTrackingPointer()
}

func (g *tapGesture) accept() {
	g.wonArena = true
	g.check()
}

func (g *tapGesture) reject() {
	g.stopTracking()
	g.recognizer.dispatchCancel(g.pointer)
}

func (g *tapGesture) cancel() {
	// If we won the arena already, then entry is resolved, so resolving again
	// is a no-op. But we still need to clean up and call changeState.
	if g.wonArena {
		g.reject()
	} else {
		g.entry.Resolve(GestureRejected) // eventually calls reject()
	}
}

func (g *tapGesture) check() {
	if pos, ok := g.finalPosition.Get(); g.wonArena && ok {
		g.recognizer.dispatchTap(g.pointer, pos)
	}
}

// MultiTapGestureRecognizerOpts are the optional parameters of
// NewMultiTapGestureRecognizer.
//
// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.new
type MultiTapGestureRecognizerOpts struct {
	// LongTapDelay is how long a pointer must stay down for OnLongTapDown; 0
	// disables it.
	LongTapDelay time.Duration
	// DebugOwner is the object that owns the recognizer.
	DebugOwner any
	// SupportedDevices are the kinds of pointer the recognizer accepts; nil for
	// all.
	SupportedDevices []ui.PointerDeviceKind
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed.
	AllowedButtonsFilter AllowedButtonsFilter
}

// MultiTapGestureRecognizer recognizes taps of several pointers at once, each
// on its own, without competing with the others.
//
// Flutter: gestures/multitap.dart MultiTapGestureRecognizer
type MultiTapGestureRecognizer struct {
	GestureRecognizerBase

	// OnTapDown is called when a pointer that might tap touches the screen.
	//
	// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.onTapDown
	OnTapDown GestureMultiTapDownCallback
	// OnTapUp is called when a pointer that tapped is lifted.
	//
	// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.onTapUp
	OnTapUp GestureMultiTapUpCallback
	// OnTap is called after OnTapUp, when a pointer taps.
	//
	// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.onTap
	OnTap GestureMultiTapCallback
	// OnTapCancel is called when a pointer that sent OnTapDown will not tap.
	//
	// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.onTapCancel
	OnTapCancel GestureMultiTapCancelCallback
	// LongTapDelay is how long a pointer must stay down for OnLongTapDown.
	//
	// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.longTapDelay
	LongTapDelay time.Duration
	// OnLongTapDown is called when a pointer stays down for LongTapDelay.
	//
	// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.onLongTapDown
	OnLongTapDown GestureMultiTapDownCallback

	gestureMap map[int]*tapGesture
}

// NewMultiTapGestureRecognizer creates a recognizer that works with the
// binding.
//
// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.new
func NewMultiTapGestureRecognizer(binding *GestureBinding, opts ...MultiTapGestureRecognizerOpts) *MultiTapGestureRecognizer {
	o := oneOpts(opts)
	r := &MultiTapGestureRecognizer{LongTapDelay: o.LongTapDelay, gestureMap: map[int]*tapGesture{}}
	r.Init(binding, r, GestureRecognizerOpts{DebugOwner: o.DebugOwner, SupportedDevices: o.SupportedDevices, AllowedButtonsFilter: o.AllowedButtonsFilter})
	return r
}

// AddAllowedPointer starts tracking the tap of the pointer.
//
// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.addAllowedPointer
func (r *MultiTapGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	if foundation.DebugMode && r.gestureMap[event.Pointer()] != nil {
		panic(errors.New("MultiTapPointerAddedTwice", "the pointer is already tracked", "pointer", event.Pointer()))
	}
	r.gestureMap[event.Pointer()] = newTapGesture(r, event, r.LongTapDelay, r.GestureSettings)
	if r.OnTapDown != nil {
		r.InvokeCallback("onTapDown", func() {
			r.OnTapDown(event.Pointer(), NewTapDownDetails(TapDownOpts{
				GlobalPosition: event.Position(), LocalPosition: foundation.Some(event.LocalPosition()),
				Kind: foundation.Some(event.Kind()),
			}))
		})
	}
}

// AcceptGesture accepts the tap of the pointer.
//
// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.acceptGesture
func (r *MultiTapGestureRecognizer) AcceptGesture(pointer int) {
	r.gestureMap[pointer].accept()
}

// RejectGesture rejects the tap of the pointer.
//
// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.rejectGesture
func (r *MultiTapGestureRecognizer) RejectGesture(pointer int) {
	r.gestureMap[pointer].reject()
}

func (r *MultiTapGestureRecognizer) dispatchCancel(pointer int) {
	delete(r.gestureMap, pointer)
	if r.OnTapCancel != nil {
		r.InvokeCallback("onTapCancel", func() { r.OnTapCancel(pointer) })
	}
}

func (r *MultiTapGestureRecognizer) dispatchTap(pointer int, position OffsetPair) {
	delete(r.gestureMap, pointer)
	if r.OnTapUp != nil {
		r.InvokeCallback("onTapUp", func() {
			r.OnTapUp(pointer, NewTapUpDetails(r.GetKindForPointer(pointer), TapUpOpts{
				LocalPosition: foundation.Some(position.Local), GlobalPosition: position.Global,
			}))
		})
	}
	if r.OnTap != nil {
		r.InvokeCallback("onTap", func() { r.OnTap(pointer) })
	}
}

func (r *MultiTapGestureRecognizer) dispatchLongTap(pointer int, lastPosition OffsetPair) {
	if r.OnLongTapDown != nil {
		r.InvokeCallback("onLongTapDown", func() {
			r.OnLongTapDown(pointer, NewTapDownDetails(TapDownOpts{
				GlobalPosition: lastPosition.Global, LocalPosition: foundation.Some(lastPosition.Local),
				Kind: foundation.Some(r.GetKindForPointer(pointer)),
			}))
		})
	}
}

// Dispose cancels the taps.
//
// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.dispose
func (r *MultiTapGestureRecognizer) Dispose() {
	pointers := make([]int, 0, len(r.gestureMap))
	for p := range r.gestureMap {
		pointers = append(pointers, p)
	}
	slices.Sort(pointers)
	for _, p := range pointers {
		if g := r.gestureMap[p]; g != nil {
			g.cancel()
		}
	}
	r.GestureRecognizerBase.Dispose()
}

// DebugDescription is "multitap".
//
// Flutter: gestures/multitap.dart MultiTapGestureRecognizer.debugDescription
func (r *MultiTapGestureRecognizer) DebugDescription() string { return "multitap" }

// SerialTapDownDetails is the data of a pointer that touched the screen and
// might be the n-th tap of a series.
//
// Flutter: gestures/multitap.dart SerialTapDownDetails
type SerialTapDownDetails struct {
	// GlobalPosition is where the pointer touched the screen.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// Kind is the kind of device.
	Kind ui.PointerDeviceKind
	// Buttons are the buttons pressed.
	Buttons int
	// Count is the number of the tap in the series, starting at 1.
	Count int
}

// SerialTapDownOpts are the optional parameters of NewSerialTapDownDetails.
//
// Flutter: gestures/multitap.dart SerialTapDownDetails.new
type SerialTapDownOpts struct {
	// GlobalPosition is where the pointer touched the screen.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target; the
	// default is GlobalPosition.
	LocalPosition foundation.Opt[ui.Offset]
	// Buttons are the buttons pressed.
	Buttons int
	// Count is the number of the tap in the series; the default is 1.
	Count int
}

// NewSerialTapDownDetails creates the details for a device of the kind. A count
// that is not positive panics with the tag SerialTapCountInvalid in debug
// builds.
//
// Flutter: gestures/multitap.dart SerialTapDownDetails.new
func NewSerialTapDownDetails(kind ui.PointerDeviceKind, opts ...SerialTapDownOpts) SerialTapDownDetails {
	o := oneOpts(opts)
	return SerialTapDownDetails{
		GlobalPosition: o.GlobalPosition, LocalPosition: localOrGlobal(o.LocalPosition, o.GlobalPosition),
		Kind: kind, Buttons: o.Buttons, Count: serialCount(o.Count),
	}
}

func serialCount(count int) int {
	if count == 0 {
		return 1
	}
	if foundation.DebugMode && count < 0 {
		panic(errors.New("SerialTapCountInvalid", "the count must be positive", "count", count))
	}
	return count
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/multitap.dart SerialTapDownDetails.globalPosition
func (d SerialTapDownDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/multitap.dart SerialTapDownDetails.localPosition
func (d SerialTapDownDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/multitap.dart SerialTapDownDetails.debugFillProperties
func (d SerialTapDownDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewEnumProperty("kind", foundation.Some(d.Kind)))
	p.Add(foundation.NewIntProperty("buttons", foundation.Some(d.Buttons)))
	p.Add(foundation.NewIntProperty("count", foundation.Some(d.Count)))
}

// GestureSerialTapDownCallback is called when a pointer that might be a tap of a
// series touches the screen.
//
// Flutter: gestures/multitap.dart GestureSerialTapDownCallback
type GestureSerialTapDownCallback func(details SerialTapDownDetails)

// SerialTapCancelDetails is the data of a series tap that was cancelled.
//
// Flutter: gestures/multitap.dart SerialTapCancelDetails
type SerialTapCancelDetails struct {
	// Count is the number of the tap in the series that was cancelled.
	Count int
}

// NewSerialTapCancelDetails creates the details; a count of 0 is 1.
//
// Flutter: gestures/multitap.dart SerialTapCancelDetails.new
func NewSerialTapCancelDetails(count int) SerialTapCancelDetails {
	return SerialTapCancelDetails{Count: serialCount(count)}
}

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/multitap.dart SerialTapCancelDetails.debugFillProperties
func (d SerialTapCancelDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewIntProperty("count", foundation.Some(d.Count)))
}

// GestureSerialTapCancelCallback is called when a tap of a series is cancelled.
//
// Flutter: gestures/multitap.dart GestureSerialTapCancelCallback
type GestureSerialTapCancelCallback func(details SerialTapCancelDetails)

// SerialTapUpDetails is the data of a tap of a series that ended.
//
// Flutter: gestures/multitap.dart SerialTapUpDetails
type SerialTapUpDetails struct {
	// GlobalPosition is where the pointer was lifted.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// Kind is the kind of device.
	Kind foundation.Opt[ui.PointerDeviceKind]
	// Count is the number of the tap in the series, starting at 1.
	Count int
}

// SerialTapUpOpts are the optional parameters of NewSerialTapUpDetails.
//
// Flutter: gestures/multitap.dart SerialTapUpDetails.new
type SerialTapUpOpts struct {
	// GlobalPosition is where the pointer was lifted.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target; the
	// default is GlobalPosition.
	LocalPosition foundation.Opt[ui.Offset]
	// Kind is the kind of device.
	Kind foundation.Opt[ui.PointerDeviceKind]
	// Count is the number of the tap in the series; the default is 1.
	Count int
}

// NewSerialTapUpDetails creates the details.
//
// Flutter: gestures/multitap.dart SerialTapUpDetails.new
func NewSerialTapUpDetails(opts ...SerialTapUpOpts) SerialTapUpDetails {
	o := oneOpts(opts)
	return SerialTapUpDetails{
		GlobalPosition: o.GlobalPosition, LocalPosition: localOrGlobal(o.LocalPosition, o.GlobalPosition),
		Kind: o.Kind, Count: serialCount(o.Count),
	}
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/multitap.dart SerialTapUpDetails.globalPosition
func (d SerialTapUpDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/multitap.dart SerialTapUpDetails.localPosition
func (d SerialTapUpDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/multitap.dart SerialTapUpDetails.debugFillProperties
func (d SerialTapUpDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewEnumProperty("kind", d.Kind))
	p.Add(foundation.NewIntProperty("count", foundation.Some(d.Count)))
}

// GestureSerialTapUpCallback is called when a tap of a series ends.
//
// Flutter: gestures/multitap.dart GestureSerialTapUpCallback
type GestureSerialTapUpCallback func(details SerialTapUpDetails)

// SerialTapGestureRecognizerOpts are the optional parameters of
// NewSerialTapGestureRecognizer.
//
// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.new
type SerialTapGestureRecognizerOpts struct {
	// DebugOwner is the object that owns the recognizer.
	DebugOwner any
	// SupportedDevices are the kinds of pointer the recognizer accepts; nil for
	// all.
	SupportedDevices []ui.PointerDeviceKind
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed.
	AllowedButtonsFilter AllowedButtonsFilter
}

// SerialTapGestureRecognizer recognizes a series of taps in quick succession, and
// reports the number of each tap.
//
// Flutter: gestures/multitap.dart SerialTapGestureRecognizer
type SerialTapGestureRecognizer struct {
	GestureRecognizerBase

	// OnSerialTapDown is called when a pointer that might be a tap of a series
	// touches the screen.
	//
	// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.onSerialTapDown
	OnSerialTapDown GestureSerialTapDownCallback
	// OnSerialTapCancel is called when a tap of a series is cancelled.
	//
	// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.onSerialTapCancel
	OnSerialTapCancel GestureSerialTapCancelCallback
	// OnSerialTapUp is called when a tap of a series ends.
	//
	// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.onSerialTapUp
	OnSerialTapUp GestureSerialTapUpCallback

	serialTapTimer    *Timer
	completedTaps     []*tapTracker
	gestureResolution map[int]GestureDisposition
	pendingTap        *tapTracker
}

// NewSerialTapGestureRecognizer creates a recognizer that works with the
// binding.
//
// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.new
func NewSerialTapGestureRecognizer(binding *GestureBinding, opts ...SerialTapGestureRecognizerOpts) *SerialTapGestureRecognizer {
	o := oneOpts(opts)
	r := &SerialTapGestureRecognizer{gestureResolution: map[int]GestureDisposition{}}
	r.Init(binding, r, GestureRecognizerOpts(o))
	return r
}

// IsTrackingPointer reports whether the recognizer tracks a pointer.
//
// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.isTrackingPointer
func (r *SerialTapGestureRecognizer) IsTrackingPointer() bool { return r.pendingTap != nil }

// IsPointerAllowed allows a pointer if a callback is set.
//
// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.isPointerAllowed
func (r *SerialTapGestureRecognizer) IsPointerAllowed(event *PointerDownEvent) bool {
	if r.OnSerialTapDown == nil && r.OnSerialTapCancel == nil && r.OnSerialTapUp == nil {
		return false
	}
	return r.GestureRecognizerBase.IsPointerAllowed(event)
}

// AddAllowedPointer tracks the tap, resetting the series if the tap is not
// part of it.
//
// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.addAllowedPointer
func (r *SerialTapGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	if (len(r.completedTaps) > 0 && !r.representsSameSeries(r.completedTaps[len(r.completedTaps)-1], event)) || r.pendingTap != nil {
		r.reset()
	}
	r.trackTap(event)
}

func (r *SerialTapGestureRecognizer) representsSameSeries(tap *tapTracker, event *PointerDownEvent) bool {
	// touch screens often detect touches intermittently
	return tap.hasElapsedMinTime() && tap.hasSameButton(event) && tap.isWithinGlobalTolerance(event, DoubleTapSlop)
}

func (r *SerialTapGestureRecognizer) trackTap(event *PointerDownEvent) {
	r.stopSerialTapTimer()
	if r.OnSerialTapDown != nil {
		details := NewSerialTapDownDetails(r.GetKindForPointer(event.Pointer()), SerialTapDownOpts{
			GlobalPosition: event.Position(), LocalPosition: foundation.Some(event.LocalPosition()),
			Buttons: event.Buttons(), Count: len(r.completedTaps) + 1,
		})
		r.InvokeCallback("onSerialTapDown", func() { r.OnSerialTapDown(details) })
	}
	tracker := newTapTracker(r.binding, event, r.binding.GestureArena.Add(event.Pointer(), r), DoubleTapMinTime, r.GestureSettings)
	r.pendingTap = tracker
	tracker.startTrackingPointer(r.handleEvent, event.Transform())
}

func (r *SerialTapGestureRecognizer) handleEvent(event PointerEvent) {
	tracker := r.pendingTap
	switch e := event.(type) {
	case *PointerUpEvent:
		r.registerTap(e, tracker)
	case *PointerMoveEvent:
		if !tracker.isWithinGlobalTolerance(event, DoubleTapTouchSlop) {
			r.reset()
		}
	case *PointerCancelEvent:
		r.reset()
	}
}

// AcceptGesture records that the pending tap won.
//
// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.acceptGesture
func (r *SerialTapGestureRecognizer) AcceptGesture(pointer int) {
	r.gestureResolution[pointer] = GestureAccepted
}

// RejectGesture records that the pending tap lost, and resets the series.
//
// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.rejectGesture
func (r *SerialTapGestureRecognizer) RejectGesture(pointer int) {
	r.gestureResolution[pointer] = GestureRejected
	r.reset()
}

func (r *SerialTapGestureRecognizer) rejectPendingTap() {
	tracker := r.pendingTap
	r.pendingTap = nil
	r.checkCancel(len(r.completedTaps) + 1)
	if _, ok := r.gestureResolution[tracker.pointer]; !ok {
		tracker.entry.Resolve(GestureRejected)
	}
	tracker.stopTrackingPointer()
}

// Dispose resets the series.
//
// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.dispose
func (r *SerialTapGestureRecognizer) Dispose() {
	r.reset()
	r.GestureRecognizerBase.Dispose()
}

func (r *SerialTapGestureRecognizer) reset() {
	if r.pendingTap != nil {
		r.rejectPendingTap()
	}
	r.pendingTap = nil
	r.completedTaps = nil
	r.gestureResolution = map[int]GestureDisposition{}
	r.stopSerialTapTimer()
}

func (r *SerialTapGestureRecognizer) registerTap(event *PointerUpEvent, tracker *tapTracker) {
	r.startSerialTapTimer()
	if _, ok := r.gestureResolution[event.Pointer()]; !ok {
		tracker.entry.Resolve(GestureAccepted)
	}
	tracker.stopTrackingPointer()
	r.pendingTap = nil
	r.checkUp(event, tracker)
	r.completedTaps = append(r.completedTaps, tracker)
}

func (r *SerialTapGestureRecognizer) startSerialTapTimer() {
	if r.serialTapTimer == nil {
		r.serialTapTimer = r.binding.NewTimer(DoubleTapTimeout, r.reset)
	}
}

func (r *SerialTapGestureRecognizer) stopSerialTapTimer() {
	if r.serialTapTimer != nil {
		r.serialTapTimer.Cancel()
		r.serialTapTimer = nil
	}
}

func (r *SerialTapGestureRecognizer) checkUp(event *PointerUpEvent, tracker *tapTracker) {
	if r.OnSerialTapUp != nil {
		details := NewSerialTapUpDetails(SerialTapUpOpts{
			GlobalPosition: event.Position(), LocalPosition: foundation.Some(event.LocalPosition()),
			Kind: foundation.Some(r.GetKindForPointer(tracker.pointer)), Count: len(r.completedTaps) + 1,
		})
		r.InvokeCallback("onSerialTapUp", func() { r.OnSerialTapUp(details) })
	}
}

func (r *SerialTapGestureRecognizer) checkCancel(count int) {
	if r.OnSerialTapCancel != nil {
		details := NewSerialTapCancelDetails(count)
		r.InvokeCallback("onSerialTapCancel", func() { r.OnSerialTapCancel(details) })
	}
}

// DebugDescription is "serial tap".
//
// Flutter: gestures/multitap.dart SerialTapGestureRecognizer.debugDescription
func (r *SerialTapGestureRecognizer) DebugDescription() string { return "serial tap" }
