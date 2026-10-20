// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package gestures

import (
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// GestureLongPressDownCallback is called when a pointer that might long press
// touches the screen.
//
// Flutter: gestures/long_press.dart GestureLongPressDownCallback
type GestureLongPressDownCallback func(details LongPressDownDetails)

// GestureLongPressCancelCallback is called when the pointer of a long press is
// cancelled.
//
// Flutter: gestures/long_press.dart GestureLongPressCancelCallback
type GestureLongPressCancelCallback func()

// GestureLongPressCallback is called when a long press is recognized.
//
// Flutter: gestures/long_press.dart GestureLongPressCallback
type GestureLongPressCallback func()

// GestureLongPressUpCallback is called when the pointer of a long press is
// lifted.
//
// Flutter: gestures/long_press.dart GestureLongPressUpCallback
type GestureLongPressUpCallback func()

// GestureLongPressStartCallback is called when a long press is recognized.
//
// Flutter: gestures/long_press.dart GestureLongPressStartCallback
type GestureLongPressStartCallback func(details LongPressStartDetails)

// GestureLongPressMoveUpdateCallback is called when the pointer moves during a
// long press.
//
// Flutter: gestures/long_press.dart GestureLongPressMoveUpdateCallback
type GestureLongPressMoveUpdateCallback func(details LongPressMoveUpdateDetails)

// GestureLongPressEndCallback is called when the pointer of a long press is
// lifted.
//
// Flutter: gestures/long_press.dart GestureLongPressEndCallback
type GestureLongPressEndCallback func(details LongPressEndDetails)

// LongPressDownDetails is the data of a pointer that touched the screen and
// might long press.
//
// Flutter: gestures/long_press.dart LongPressDownDetails
type LongPressDownDetails struct {
	// GlobalPosition is where the pointer touched the screen.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// Kind is the kind of device.
	Kind foundation.Opt[ui.PointerDeviceKind]
}

// NewLongPressDownDetails creates the details; the local position is the global
// one unless given.
//
// Flutter: gestures/long_press.dart LongPressDownDetails.new
func NewLongPressDownDetails(opts ...TapDownOpts) LongPressDownDetails {
	o := oneOpts(opts)
	return LongPressDownDetails{GlobalPosition: o.GlobalPosition, LocalPosition: localOrGlobal(o.LocalPosition, o.GlobalPosition), Kind: o.Kind}
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/long_press.dart LongPressDownDetails.globalPosition
func (d LongPressDownDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/long_press.dart LongPressDownDetails.localPosition
func (d LongPressDownDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/long_press.dart LongPressDownDetails.debugFillProperties
func (d LongPressDownDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewEnumProperty("kind", d.Kind))
}

// LongPressStartDetails is the data of a long press that started.
//
// Flutter: gestures/long_press.dart LongPressStartDetails
type LongPressStartDetails struct {
	// GlobalPosition is where the long press started.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
}

// NewLongPressStartDetails creates the details; the local position is the global
// one unless given.
//
// Flutter: gestures/long_press.dart LongPressStartDetails.new
func NewLongPressStartDetails(opts ...TapUpOpts) LongPressStartDetails {
	o := oneOpts(opts)
	return LongPressStartDetails{GlobalPosition: o.GlobalPosition, LocalPosition: localOrGlobal(o.LocalPosition, o.GlobalPosition)}
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/long_press.dart LongPressStartDetails.globalPosition
func (d LongPressStartDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/long_press.dart LongPressStartDetails.localPosition
func (d LongPressStartDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/long_press.dart LongPressStartDetails.debugFillProperties
func (d LongPressStartDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
}

// LongPressMoveUpdateDetails is the data of a pointer that moved during a long
// press.
//
// Flutter: gestures/long_press.dart LongPressMoveUpdateDetails
type LongPressMoveUpdateDetails struct {
	// GlobalPosition is the global position of the pointer.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// OffsetFromOrigin is how far the pointer is from where the long press
	// started, in global coordinates.
	OffsetFromOrigin ui.Offset
	// LocalOffsetFromOrigin is OffsetFromOrigin in the coordinates of the
	// target.
	LocalOffsetFromOrigin ui.Offset
}

// LongPressMoveUpdateOpts are the optional parameters of
// NewLongPressMoveUpdateDetails.
//
// Flutter: gestures/long_press.dart LongPressMoveUpdateDetails.new
type LongPressMoveUpdateOpts struct {
	// GlobalPosition is the global position of the pointer.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target; the
	// default is GlobalPosition.
	LocalPosition foundation.Opt[ui.Offset]
	// OffsetFromOrigin is how far the pointer is from where the long press
	// started.
	OffsetFromOrigin ui.Offset
	// LocalOffsetFromOrigin is OffsetFromOrigin in local coordinates; the
	// default is OffsetFromOrigin.
	LocalOffsetFromOrigin foundation.Opt[ui.Offset]
}

// NewLongPressMoveUpdateDetails creates the details.
//
// Flutter: gestures/long_press.dart LongPressMoveUpdateDetails.new
func NewLongPressMoveUpdateDetails(opts ...LongPressMoveUpdateOpts) LongPressMoveUpdateDetails {
	o := oneOpts(opts)
	return LongPressMoveUpdateDetails{
		GlobalPosition: o.GlobalPosition, LocalPosition: localOrGlobal(o.LocalPosition, o.GlobalPosition),
		OffsetFromOrigin: o.OffsetFromOrigin, LocalOffsetFromOrigin: localOrGlobal(o.LocalOffsetFromOrigin, o.OffsetFromOrigin),
	}
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/long_press.dart LongPressMoveUpdateDetails.globalPosition
func (d LongPressMoveUpdateDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/long_press.dart LongPressMoveUpdateDetails.localPosition
func (d LongPressMoveUpdateDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/long_press.dart LongPressMoveUpdateDetails.debugFillProperties
func (d LongPressMoveUpdateDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewDiagnosticsProperty("offsetFromOrigin", d.OffsetFromOrigin))
	p.Add(foundation.NewDiagnosticsProperty("localOffsetFromOrigin", d.LocalOffsetFromOrigin))
}

// LongPressEndDetails is the data of a long press that ended.
//
// Flutter: gestures/long_press.dart LongPressEndDetails
type LongPressEndDetails struct {
	// GlobalPosition is where the pointer was lifted.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// Velocity is the velocity of the pointer when it was lifted.
	Velocity Velocity
}

// LongPressEndOpts are the optional parameters of NewLongPressEndDetails.
//
// Flutter: gestures/long_press.dart LongPressEndDetails.new
type LongPressEndOpts struct {
	// GlobalPosition is where the pointer was lifted.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target; the
	// default is GlobalPosition.
	LocalPosition foundation.Opt[ui.Offset]
	// Velocity is the velocity of the pointer.
	Velocity Velocity
}

// NewLongPressEndDetails creates the details.
//
// Flutter: gestures/long_press.dart LongPressEndDetails.new
func NewLongPressEndDetails(opts ...LongPressEndOpts) LongPressEndDetails {
	o := oneOpts(opts)
	return LongPressEndDetails{GlobalPosition: o.GlobalPosition, LocalPosition: localOrGlobal(o.LocalPosition, o.GlobalPosition), Velocity: o.Velocity}
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/long_press.dart LongPressEndDetails.globalPosition
func (d LongPressEndDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/long_press.dart LongPressEndDetails.localPosition
func (d LongPressEndDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/long_press.dart LongPressEndDetails.debugFillProperties
func (d LongPressEndDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewDiagnosticsProperty("velocity", d.Velocity))
}

// LongPressGestureRecognizerOpts are the optional parameters of
// NewLongPressGestureRecognizer.
//
// Flutter: gestures/long_press.dart LongPressGestureRecognizer.new
type LongPressGestureRecognizerOpts struct {
	// Duration is how long the pointer must stay down; the default is
	// LongPressTimeout.
	Duration foundation.Opt[time.Duration]
	// PostAcceptSlopTolerance is how far the pointer may move after the long
	// press was recognized; the default (None) is any distance.
	PostAcceptSlopTolerance foundation.Opt[float64]
	// SupportedDevices are the kinds of pointer the recognizer accepts; nil for
	// all.
	SupportedDevices []ui.PointerDeviceKind
	// DebugOwner is the object that owns the recognizer.
	DebugOwner any
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed;
	// the default allows the primary, secondary and tertiary buttons.
	AllowedButtonsFilter AllowedButtonsFilter
}

// LongPressGestureRecognizer recognizes a pointer that stays down for a time
// (LongPressTimeout by default) without moving past the slop.
//
// Flutter: gestures/long_press.dart LongPressGestureRecognizer
type LongPressGestureRecognizer struct {
	PrimaryPointerGestureRecognizer

	// OnLongPressDown is called when a pointer that might long press with the
	// primary button touches the screen.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onLongPressDown
	OnLongPressDown GestureLongPressDownCallback
	// OnLongPressCancel is called when the pointer that sent OnLongPressDown
	// will not long press.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onLongPressCancel
	OnLongPressCancel GestureLongPressCancelCallback
	// OnLongPress is called when a long press with the primary button is
	// recognized.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onLongPress
	OnLongPress GestureLongPressCallback
	// OnLongPressStart is called when a long press with the primary button is
	// recognized.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onLongPressStart
	OnLongPressStart GestureLongPressStartCallback
	// OnLongPressMoveUpdate is called when the pointer moves during a long
	// press with the primary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onLongPressMoveUpdate
	OnLongPressMoveUpdate GestureLongPressMoveUpdateCallback
	// OnLongPressUp is called when the pointer of a long press with the primary
	// button is lifted.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onLongPressUp
	OnLongPressUp GestureLongPressUpCallback
	// OnLongPressEnd is called when the pointer of a long press with the
	// primary button is lifted.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onLongPressEnd
	OnLongPressEnd GestureLongPressEndCallback
	// OnSecondaryLongPressDown is OnLongPressDown for the secondary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onSecondaryLongPressDown
	OnSecondaryLongPressDown GestureLongPressDownCallback
	// OnSecondaryLongPressCancel is OnLongPressCancel for the secondary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onSecondaryLongPressCancel
	OnSecondaryLongPressCancel GestureLongPressCancelCallback
	// OnSecondaryLongPress is OnLongPress for the secondary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onSecondaryLongPress
	OnSecondaryLongPress GestureLongPressCallback
	// OnSecondaryLongPressStart is OnLongPressStart for the secondary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onSecondaryLongPressStart
	OnSecondaryLongPressStart GestureLongPressStartCallback
	// OnSecondaryLongPressMoveUpdate is OnLongPressMoveUpdate for the secondary
	// button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onSecondaryLongPressMoveUpdate
	OnSecondaryLongPressMoveUpdate GestureLongPressMoveUpdateCallback
	// OnSecondaryLongPressUp is OnLongPressUp for the secondary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onSecondaryLongPressUp
	OnSecondaryLongPressUp GestureLongPressUpCallback
	// OnSecondaryLongPressEnd is OnLongPressEnd for the secondary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onSecondaryLongPressEnd
	OnSecondaryLongPressEnd GestureLongPressEndCallback
	// OnTertiaryLongPressDown is OnLongPressDown for the tertiary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onTertiaryLongPressDown
	OnTertiaryLongPressDown GestureLongPressDownCallback
	// OnTertiaryLongPressCancel is OnLongPressCancel for the tertiary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onTertiaryLongPressCancel
	OnTertiaryLongPressCancel GestureLongPressCancelCallback
	// OnTertiaryLongPress is OnLongPress for the tertiary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onTertiaryLongPress
	OnTertiaryLongPress GestureLongPressCallback
	// OnTertiaryLongPressStart is OnLongPressStart for the tertiary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onTertiaryLongPressStart
	OnTertiaryLongPressStart GestureLongPressStartCallback
	// OnTertiaryLongPressMoveUpdate is OnLongPressMoveUpdate for the tertiary
	// button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onTertiaryLongPressMoveUpdate
	OnTertiaryLongPressMoveUpdate GestureLongPressMoveUpdateCallback
	// OnTertiaryLongPressUp is OnLongPressUp for the tertiary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onTertiaryLongPressUp
	OnTertiaryLongPressUp GestureLongPressUpCallback
	// OnTertiaryLongPressEnd is OnLongPressEnd for the tertiary button.
	//
	// Flutter: gestures/long_press.dart LongPressGestureRecognizer.onTertiaryLongPressEnd
	OnTertiaryLongPressEnd GestureLongPressEndCallback

	longPressAccepted bool
	longPressOrigin   foundation.Opt[OffsetPair]
	initialButtons    foundation.Opt[int]
	velocityTracker   *VelocityTracker
}

// NewLongPressGestureRecognizer creates a recognizer that works with the
// binding.
//
// Flutter: gestures/long_press.dart LongPressGestureRecognizer.new
func NewLongPressGestureRecognizer(binding *GestureBinding, opts ...LongPressGestureRecognizerOpts) *LongPressGestureRecognizer {
	o := oneOpts(opts)
	filter := o.AllowedButtonsFilter
	if filter == nil {
		filter = func(b int) bool { return b == PrimaryButton || b == SecondaryButton || b == TertiaryButton }
	}
	post := SlopToleranceNone
	if v, ok := o.PostAcceptSlopTolerance.Get(); ok {
		post = NewSlopTolerance(v)
	}
	r := &LongPressGestureRecognizer{}
	r.InitPrimaryPointer(binding, r, PrimaryPointerOpts{
		GestureRecognizerOpts:   GestureRecognizerOpts{DebugOwner: o.DebugOwner, SupportedDevices: o.SupportedDevices, AllowedButtonsFilter: filter},
		Deadline:                foundation.Some(o.Duration.OrElse(LongPressTimeout)),
		PostAcceptSlopTolerance: post,
	})
	return r
}

// longPressButtonCallbacks are the callbacks of one button of a long press.
type longPressButtonCallbacks struct {
	prefix     string
	down       GestureLongPressDownCallback
	cancel     GestureLongPressCancelCallback
	press      GestureLongPressCallback
	start      GestureLongPressStartCallback
	moveUpdate GestureLongPressMoveUpdateCallback
	up         GestureLongPressUpCallback
	end        GestureLongPressEndCallback
}

func (c longPressButtonCallbacks) none() bool {
	return c.down == nil && c.cancel == nil && c.start == nil && c.press == nil && c.moveUpdate == nil && c.end == nil && c.up == nil
}

// callbacks returns the callbacks of the button.
func (r *LongPressGestureRecognizer) callbacks(buttons int) (longPressButtonCallbacks, bool) {
	switch buttons {
	case PrimaryButton:
		return longPressButtonCallbacks{"", r.OnLongPressDown, r.OnLongPressCancel, r.OnLongPress, r.OnLongPressStart,
			r.OnLongPressMoveUpdate, r.OnLongPressUp, r.OnLongPressEnd}, true
	case SecondaryButton:
		return longPressButtonCallbacks{"Secondary", r.OnSecondaryLongPressDown, r.OnSecondaryLongPressCancel, r.OnSecondaryLongPress,
			r.OnSecondaryLongPressStart, r.OnSecondaryLongPressMoveUpdate, r.OnSecondaryLongPressUp, r.OnSecondaryLongPressEnd}, true
	case TertiaryButton:
		return longPressButtonCallbacks{"Tertiary", r.OnTertiaryLongPressDown, r.OnTertiaryLongPressCancel, r.OnTertiaryLongPress,
			r.OnTertiaryLongPressStart, r.OnTertiaryLongPressMoveUpdate, r.OnTertiaryLongPressUp, r.OnTertiaryLongPressEnd}, true
	}
	return longPressButtonCallbacks{}, false
}

// currentCallbacks returns the callbacks of the button that started the long
// press. An unhandled button panics with the tag LongPressButtonUnhandled in
// debug builds.
func (r *LongPressGestureRecognizer) currentCallbacks() (longPressButtonCallbacks, bool) {
	c, ok := r.callbacks(r.initialButtons.Value())
	if !ok && foundation.DebugMode {
		panic(errors.New("LongPressButtonUnhandled", "unhandled button", "buttons", r.initialButtons.Value()))
	}
	return c, ok
}

// IsPointerAllowed allows a pointer whose button has a callback.
//
// Flutter: gestures/long_press.dart LongPressGestureRecognizer.isPointerAllowed
func (r *LongPressGestureRecognizer) IsPointerAllowed(event *PointerDownEvent) bool {
	c, ok := r.callbacks(event.Buttons())
	if !ok || c.none() {
		return false
	}
	return r.PrimaryPointerGestureRecognizer.IsPointerAllowed(event)
}

// DidExceedDeadline accepts the long press.
//
// Flutter: gestures/long_press.dart LongPressGestureRecognizer.didExceedDeadline
func (r *LongPressGestureRecognizer) DidExceedDeadline() {
	// Exceeding the deadline puts the gesture in the accepted state.
	r.Resolve(GestureAccepted)
	r.longPressAccepted = true
	r.PrimaryPointerGestureRecognizer.AcceptGesture(r.PrimaryPointer().Value())
	r.checkLongPressStart()
}

// HandlePrimaryPointer follows the primary pointer.
//
// Flutter: gestures/long_press.dart LongPressGestureRecognizer.handlePrimaryPointer
func (r *LongPressGestureRecognizer) HandlePrimaryPointer(event PointerEvent) {
	if !event.Synthesized() {
		if down, ok := event.(*PointerDownEvent); ok {
			r.velocityTracker = NewVelocityTrackerWithKind(r.binding.Clock(), down.Kind())
			r.velocityTracker.AddPosition(down.TimeStamp(), down.LocalPosition())
		}
		if move, ok := event.(*PointerMoveEvent); ok && r.velocityTracker != nil {
			r.velocityTracker.AddPosition(move.TimeStamp(), move.LocalPosition())
		}
	}
	switch e := event.(type) {
	case *PointerUpEvent:
		if r.longPressAccepted {
			r.checkLongPressEnd(e)
		} else {
			// Because the gesture recognizer doesn't received the up event if
			// it is rejected, the long press is cancelled.
			r.Resolve(GestureRejected)
		}
		r.reset()
	case *PointerCancelEvent:
		r.checkLongPressCancel()
		r.reset()
	case *PointerDownEvent:
		// The first touch.
		r.longPressOrigin = foundation.Some(NewOffsetPairFromEventPosition(e))
		r.initialButtons = foundation.Some(e.Buttons())
		r.checkLongPressDown(e)
	case *PointerMoveEvent:
		if e.Buttons() != r.initialButtons.Value() && !r.longPressAccepted {
			r.Resolve(GestureRejected)
			r.StopTrackingPointer(r.PrimaryPointer().Value())
		} else if r.longPressAccepted {
			r.checkLongPressMoveUpdate(e)
		}
	}
}

func (r *LongPressGestureRecognizer) checkLongPressDown(event *PointerDownEvent) {
	origin := r.longPressOrigin.Value()
	details := NewLongPressDownDetails(TapDownOpts{
		GlobalPosition: origin.Global, LocalPosition: foundation.Some(origin.Local),
		Kind: foundation.Some(r.GetKindForPointer(event.Pointer())),
	})
	if c, ok := r.currentCallbacks(); ok && c.down != nil {
		r.InvokeCallback("on"+c.prefix+"LongPressDown", func() { c.down(details) })
	}
}

func (r *LongPressGestureRecognizer) checkLongPressCancel() {
	if r.State() == GestureRecognizerStatePossible {
		if c, ok := r.currentCallbacks(); ok && c.cancel != nil {
			r.InvokeCallback("on"+c.prefix+"LongPressCancel", c.cancel)
		}
	}
}

func (r *LongPressGestureRecognizer) checkLongPressStart() {
	c, ok := r.currentCallbacks()
	if !ok {
		return
	}
	if c.start != nil {
		origin := r.longPressOrigin.Value()
		details := NewLongPressStartDetails(TapUpOpts{GlobalPosition: origin.Global, LocalPosition: foundation.Some(origin.Local)})
		r.InvokeCallback("on"+c.prefix+"LongPressStart", func() { c.start(details) })
	}
	if c.press != nil {
		r.InvokeCallback("on"+c.prefix+"LongPress", c.press)
	}
}

func (r *LongPressGestureRecognizer) checkLongPressMoveUpdate(event PointerEvent) {
	origin := r.longPressOrigin.Value()
	details := NewLongPressMoveUpdateDetails(LongPressMoveUpdateOpts{
		GlobalPosition: event.Position(), LocalPosition: foundation.Some(event.LocalPosition()),
		OffsetFromOrigin:      event.Position().Sub(origin.Global),
		LocalOffsetFromOrigin: foundation.Some(event.LocalPosition().Sub(origin.Local)),
	})
	if c, ok := r.currentCallbacks(); ok && c.moveUpdate != nil {
		r.InvokeCallback("on"+c.prefix+"LongPressMoveUpdate", func() { c.moveUpdate(details) })
	}
}

func (r *LongPressGestureRecognizer) checkLongPressEnd(event PointerEvent) {
	velocity := VelocityZero
	if estimate, ok := r.velocityTracker.GetVelocityEstimate().Get(); ok {
		velocity = Velocity{estimate.PixelsPerSecond}
	}
	details := NewLongPressEndDetails(LongPressEndOpts{
		GlobalPosition: event.Position(), LocalPosition: foundation.Some(event.LocalPosition()), Velocity: velocity,
	})
	r.velocityTracker = nil
	c, ok := r.currentCallbacks()
	if !ok {
		return
	}
	if c.end != nil {
		r.InvokeCallback("on"+c.prefix+"LongPressEnd", func() { c.end(details) })
	}
	if c.up != nil {
		r.InvokeCallback("on"+c.prefix+"LongPressUp", c.up)
	}
}

func (r *LongPressGestureRecognizer) reset() {
	r.longPressAccepted = false
	r.longPressOrigin = foundation.None[OffsetPair]()
	r.initialButtons = foundation.None[int]()
	r.velocityTracker = nil
}

// Resolve cancels or resets the long press when it is rejected, then resolves.
//
// Flutter: gestures/long_press.dart LongPressGestureRecognizer.resolve
func (r *LongPressGestureRecognizer) Resolve(disposition GestureDisposition) {
	if disposition == GestureRejected {
		if r.longPressAccepted {
			// This can happen if the gesture has been canceled. For example when
			// the buttons have changed.
			r.reset()
		} else {
			r.checkLongPressCancel()
		}
	}
	r.PrimaryPointerGestureRecognizer.Resolve(disposition)
}

// AcceptGesture does nothing: the recognizer accepts itself when the deadline
// passes.
//
// Flutter: gestures/long_press.dart LongPressGestureRecognizer.acceptGesture
func (r *LongPressGestureRecognizer) AcceptGesture(int) {
	// Winning the arena isn't important here, since it is already accepted by
	// the deadline.
}

// DebugDescription is "long press".
//
// Flutter: gestures/long_press.dart LongPressGestureRecognizer.debugDescription
func (r *LongPressGestureRecognizer) DebugDescription() string { return "long press" }
