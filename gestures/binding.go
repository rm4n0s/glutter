// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package gestures

import (
	"fmt"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// SamplingTimer is a function waiting to run on a SamplingClock.
//
// Flutter: dart:async Timer
type SamplingTimer interface {
	// Stop prevents the function from running.
	Stop() bool
}

// SamplingClock is the clock the event resampler reads. The default is the
// system clock; tests replace it through GestureBindingOpts.Clock.
//
// Flutter: gestures/binding.dart SamplingClock
type SamplingClock interface {
	// Now returns the time.
	//
	// Flutter: gestures/binding.dart SamplingClock.now
	Now() time.Time
	// AfterFunc calls f once after d, on any goroutine. It stands in for the
	// Timer.periodic of Flutter's resampler, which re-arms it.
	AfterFunc(d time.Duration, f func()) SamplingTimer
}

type systemSamplingClock struct{}

func (systemSamplingClock) Now() time.Time { return time.Now() }
func (systemSamplingClock) AfterFunc(d time.Duration, f func()) SamplingTimer {
	return time.AfterFunc(d, f)
}

// GestureScheduler is what the GestureBinding needs of the scheduler binding.
// *scheduler.SchedulerBinding implements it.
type GestureScheduler interface {
	// Locked reports whether events are held back.
	Locked() bool
	// AddUnlockedCallback adds a function that runs when events are unlocked.
	AddUnlockedCallback(fn func())
	// AddPostFrameCallback runs the callback after the current frame.
	AddPostFrameCallback(callback scheduler.FrameCallback)
	// CurrentSystemFrameTimeStamp is the time of the current frame, as the
	// platform gave it.
	CurrentSystemFrameTimeStamp() time.Duration
}

var _ GestureScheduler = (*scheduler.SchedulerBinding)(nil)

// periodicTimer is Timer.periodic on a SamplingClock: the clock's one-shot timer
// re-armed after each tick, with the tick run on the UI thread.
type periodicTimer struct {
	stopped bool
	timer   SamplingTimer
}

func (p *periodicTimer) cancel() {
	if p == nil {
		return
	}
	p.stopped = true
	if p.timer != nil {
		p.timer.Stop()
	}
}

func (p *periodicTimer) isActive() bool { return p != nil && !p.stopped }

const (
	defaultSamplingOffset = -38 * time.Millisecond
	samplingInterval      = 16667 * time.Microsecond
)

// resampler drives the PointerEventResampler of each touch device from the
// frame times.
type resampler struct {
	b                      *GestureBinding
	resamplers             map[int]*PointerEventResampler
	frameCallbackScheduled bool
	frameTime              time.Duration
	frameTimeAgeStart      time.Time
	lastSampleTime         time.Duration
	lastEventTime          time.Duration
	timer                  *periodicTimer
}

func newResampler(b *GestureBinding) *resampler {
	return &resampler{b: b, resamplers: map[int]*PointerEventResampler{}}
}

func (r *resampler) addOrDispatch(event PointerEvent) {
	if event.Kind() == ui.PointerDeviceKindTouch {
		// Add the event to the resampler of its device.
		r.lastEventTime = event.TimeStamp()
		res, ok := r.resamplers[event.Device()]
		if !ok {
			res = NewPointerEventResampler()
			r.resamplers[event.Device()] = res
		}
		res.AddEvent(event)
	} else {
		// Dispatch the event right away if it is not a touch event.
		r.b.handlePointerEventImmediately(event)
	}
}

func (r *resampler) startTimer() {
	r.timer = r.b.startPeriodic(samplingInterval, r.onSampleTimeChanged)
}

func (r *resampler) sample(samplingOffset time.Duration, clock SamplingClock) {
	if r.frameTime == 0 {
		// Initialize the frame time and its age from the clock.
		r.frameTime = time.Duration(clock.Now().UnixMilli()) * time.Millisecond
		r.frameTimeAgeStart = clock.Now()
	}
	// Schedule periodic resampling if it is not already running.
	if !r.timer.isActive() {
		r.startTimer()
	}
	// Calculate the effective frame time by taking the number of sampling
	// intervals since the frame time into account.
	elapsedIntervals := clock.Now().Sub(r.frameTimeAgeStart).Microseconds() / samplingInterval.Microseconds()
	elapsed := time.Duration(elapsedIntervals) * time.Duration(samplingInterval.Microseconds()) * time.Microsecond
	frameTime := r.frameTime + elapsed
	sampleTime := frameTime + samplingOffset
	nextSampleTime := sampleTime + samplingInterval

	// Resample and dispatch the events of each device.
	for _, res := range r.resamplers {
		res.Sample(sampleTime, nextSampleTime, r.b.handlePointerEventImmediately)
	}
	// Remove the resamplers that have nothing to do.
	for device, res := range r.resamplers {
		if !res.HasPendingEvents() && !res.IsDown() {
			delete(r.resamplers, device)
		}
	}
	r.lastSampleTime = sampleTime

	// Stop the timer when nothing is left to resample.
	if len(r.resamplers) == 0 {
		r.timer.cancel()
		return
	}

	// Schedule a frame callback if one is not scheduled yet: when it runs, the
	// frame time is the one of the frame, and the sampling timer is reset to it.
	if !r.frameCallbackScheduled {
		r.frameCallbackScheduled = true
		r.b.scheduler.AddPostFrameCallback(func(time.Duration) {
			r.frameCallbackScheduled = false
			// We use the frame timestamp as the original frame time, since the
			// clock's own time may be a bit late.
			r.frameTime = r.b.scheduler.CurrentSystemFrameTimeStamp()
			r.frameTimeAgeStart = r.b.Clock().Now()
			// Reset the timer so that it is in phase with the frames.
			r.timer.cancel()
			r.startTimer()
			r.onSampleTimeChanged()
		})
	}
}

func (r *resampler) stop() {
	for _, res := range r.resamplers {
		res.Stop(r.b.handlePointerEventImmediately)
	}
	r.resamplers = map[int]*PointerEventResampler{}
	r.frameTime = 0
	r.timer.cancel()
}

func (r *resampler) onSampleTimeChanged() {
	if foundation.DebugMode && r.b.Debug.PrintResamplingMargin {
		r.b.Debug.print((r.lastEventTime - r.lastSampleTime).String())
	}
	r.b.handleSampleTimeChanged()
}

// GestureBindingOpts are the optional parameters of NewGestureBinding.
//
// Flutter: gestures/binding.dart GestureBinding
type GestureBindingOpts struct {
	// Debug holds the diagnostics flags of the gestures library.
	Debug DebugOptions
	// Clock is the clock of the resampler, the recognizers' timers and the
	// velocity trackers; the default is the system clock. Tests give a fake
	// one.
	//
	// Flutter: gestures/binding.dart GestureBinding.debugSamplingClock
	Clock SamplingClock
	// ErrorReporter receives the panics the binding recovers from; see
	// SetErrorReporter.
	ErrorReporter foundation.ErrorReporter
}

// GestureBinding is the glue between the engine's pointer data and the gestures
// library: it converts the data to events, hit tests them, dispatches them to
// the targets that were hit, routes them and resolves the gesture arena.
//
// The renderer binding plugs in the hit test with OnHitTestInView and the mouse
// tracking with OnDispatchEvent.
//
// Deviation: Flutter's GestureBinding is a mixin on BindingBase with a global
// instance; here it is a struct that is made from the dispatcher and the
// scheduler (SPEC §12.10).
//
// Flutter: gestures/binding.dart GestureBinding
type GestureBinding struct {
	d         *ui.PlatformDispatcher
	scheduler GestureScheduler

	// PointerRouter routes the events of a pointer to the routes registered
	// for it.
	//
	// Flutter: gestures/binding.dart GestureBinding.pointerRouter
	PointerRouter *PointerRouter
	// GestureArena decides which gesture of a pointer wins.
	//
	// Flutter: gestures/binding.dart GestureBinding.gestureArena
	GestureArena *GestureArenaManager
	// PointerSignalResolver decides who handles a signal event.
	//
	// Flutter: gestures/binding.dart GestureBinding.pointerSignalResolver
	PointerSignalResolver *PointerSignalResolver

	// Debug holds the diagnostics flags.
	Debug *DebugOptions

	errorReporter foundation.ErrorReporter

	// OnHitTestInView, if set, is called by HitTestInView before the binding
	// adds itself to the result. The renderer binding uses it to hit test the
	// render views.
	//
	// Flutter: rendering/binding.dart RendererBinding.hitTestInView
	OnHitTestInView func(result *HitTestResult, position ui.Offset, viewID ui.ViewID)
	// OnDispatchEvent, if set, is called by DispatchEvent before it sends the
	// event to the targets. The renderer binding uses it to update the mouse
	// tracker.
	//
	// Flutter: rendering/binding.dart RendererBinding.dispatchEvent
	OnDispatchEvent func(event PointerEvent, hitTestResult *HitTestResult)

	// ResamplingEnabled is whether touch events are resampled to the frame
	// times. It is off by default.
	//
	// Flutter: gestures/binding.dart GestureBinding.resamplingEnabled
	ResamplingEnabled bool
	// SamplingOffset is the time between the frame time and the sample time
	// when resampling. The default is -38ms.
	//
	// Flutter: gestures/binding.dart GestureBinding.samplingOffset
	SamplingOffset time.Duration

	clock                SamplingClock
	pendingPointerEvents []PointerEvent
	hitTests             map[int]*HitTestResult
	resampler            *resampler
}

var (
	_ HitTestable       = (*GestureBinding)(nil)
	_ HitTestDispatcher = (*GestureBinding)(nil)
	_ HitTestTarget     = (*GestureBinding)(nil)
)

// NewGestureBinding creates the binding and sets it as the receiver of the
// dispatcher's pointer data.
//
// Flutter: gestures/binding.dart GestureBinding.initInstances
func NewGestureBinding(d *ui.PlatformDispatcher, s GestureScheduler, opts ...GestureBindingOpts) *GestureBinding {
	o := oneOpts(opts)
	b := &GestureBinding{
		d:              d,
		scheduler:      s,
		Debug:          &o.Debug,
		clock:          o.Clock,
		SamplingOffset: defaultSamplingOffset,
		hitTests:       map[int]*HitTestResult{},
	}
	b.PointerRouter = NewPointerRouter()
	b.GestureArena = NewGestureArenaManager(d.Post, GestureArenaManagerOpts{Debug: b.Debug})
	b.PointerSignalResolver = NewPointerSignalResolver()
	b.SetErrorReporter(o.ErrorReporter)
	b.resampler = newResampler(b)
	d.OnPointerDataPacket = b.handlePointerDataPacket
	s.AddUnlockedCallback(b.flushPointerEventQueue)
	return b
}

func (b *GestureBinding) devicePixelRatioForView(viewID ui.ViewID) foundation.Opt[float64] {
	v := b.d.View(viewID)
	if v == nil {
		return foundation.None[float64]()
	}
	return foundation.Some(v.DevicePixelRatio())
}

// SetErrorReporter sets the function that receives the panics the binding
// recovers from: in the conversion of pointer data, in the targets of a hit test,
// in the routes of the pointer router and in the signal resolver. Without one a
// panic goes on (SPEC §14.2).
//
// Flutter: foundation/assertions.dart FlutterError.onError
func (b *GestureBinding) SetErrorReporter(r foundation.ErrorReporter) {
	b.errorReporter = r
	b.PointerRouter.errorReporter = r
	b.PointerSignalResolver.errorReporter = r
}

func (b *GestureBinding) handlePointerDataPacket(packet ui.PointerDataPacket) {
	foundation.Guard(b.errorReporter, foundation.CallbackPanicked, "while handling a pointer data packet", nil, func() {
		b.pendingPointerEvents = append(b.pendingPointerEvents,
			PointerEventConverterExpand(packet.Data, b.devicePixelRatioForView)...)
		if !b.scheduler.Locked() {
			b.flushPointerEventQueue()
		}
	})
}

// CancelPointer dispatches a cancel event for the pointer, before the events
// that are waiting.
//
// Flutter: gestures/binding.dart GestureBinding.cancelPointer
func (b *GestureBinding) CancelPointer(pointer int) {
	if len(b.pendingPointerEvents) == 0 && !b.scheduler.Locked() {
		b.d.Post(b.flushPointerEventQueue)
	}
	b.pendingPointerEvents = append([]PointerEvent{NewPointerCancelEvent(PointerEventOpts{Pointer: pointer})},
		b.pendingPointerEvents...)
}

func (b *GestureBinding) flushPointerEventQueue() {
	if foundation.DebugMode && b.scheduler.Locked() {
		panic(errors.New("GestureBindingLocked", "pointer events cannot be flushed while the binding is locked"))
	}
	for len(b.pendingPointerEvents) > 0 {
		event := b.pendingPointerEvents[0]
		b.pendingPointerEvents = b.pendingPointerEvents[1:]
		b.HandlePointerEvent(event)
	}
}

// HandlePointerEvent dispatches the event to the targets it hits, after
// resampling it if resampling is enabled. It must not be called while the
// binding is locked.
//
// Flutter: gestures/binding.dart GestureBinding.handlePointerEvent
func (b *GestureBinding) HandlePointerEvent(event PointerEvent) {
	if foundation.DebugMode && b.scheduler.Locked() {
		panic(errors.New("GestureBindingLocked", "pointer events cannot be handled while the binding is locked"))
	}
	if b.ResamplingEnabled {
		b.resampler.addOrDispatch(event)
		b.resampler.sample(b.SamplingOffset, b.Clock())
		return
	}
	// Stop resampling if it was enabled before.
	b.resampler.stop()
	b.handlePointerEventImmediately(event)
}

func (b *GestureBinding) handlePointerEventImmediately(event PointerEvent) {
	var hitTestResult *HitTestResult
	switch e := event.(type) {
	case *PointerDownEvent, *PointerHoverEvent, *PointerPanZoomStartEvent, PointerSignalEvent:
		if foundation.DebugMode {
			if _, ok := b.hitTests[event.Pointer()]; ok {
				panic(errors.New("PointerHitTestExists",
					"the pointer of the event unexpectedly has a hit test result associated with it",
					"event", foundation.DiagnosticableString(event, foundation.DiagnosticLevelDebug)))
			}
		}
		hitTestResult = NewHitTestResult()
		b.HitTestInView(hitTestResult, event.Position(), event.ViewID())
		switch e.(type) {
		case *PointerDownEvent, *PointerPanZoomStartEvent:
			b.hitTests[event.Pointer()] = hitTestResult
		}
		if foundation.DebugMode && b.Debug.PrintHitTestResults {
			b.Debug.print(fmt.Sprintf("%s: %s",
				foundation.DiagnosticableString(event, foundation.DiagnosticLevelDebug), hitTestResult))
		}
	case *PointerUpEvent, *PointerCancelEvent, *PointerPanZoomEndEvent:
		hitTestResult = b.hitTests[event.Pointer()]
		delete(b.hitTests, event.Pointer())
	default:
		_, isUpdate := event.(*PointerPanZoomUpdateEvent)
		if event.Down() || isUpdate {
			hitTestResult = b.hitTests[event.Pointer()]
		}
	}
	if foundation.DebugMode && b.Debug.PrintMouseHoverEvents {
		if _, ok := event.(*PointerHoverEvent); ok {
			b.Debug.print(event.String())
		}
	}
	_, added := event.(*PointerAddedEvent)
	_, removed := event.(*PointerRemovedEvent)
	if hitTestResult != nil || added || removed {
		b.DispatchEvent(event, hitTestResult)
	}
}

// HitTestInView adds the binding to the result, after OnHitTestInView had its
// turn. The binding is the last target of every hit test, so it routes the
// event and resolves the arena after the objects in front of it handled it.
//
// Flutter: gestures/binding.dart GestureBinding.hitTestInView
func (b *GestureBinding) HitTestInView(result *HitTestResult, position ui.Offset, viewID ui.ViewID) {
	if b.OnHitTestInView != nil {
		b.OnHitTestInView(result, position, viewID)
	}
	result.Add(NewHitTestEntry(b))
}

// DispatchEvent sends the event to each target in the hit test result, front
// first, transformed to the target's coordinates. An event without a result
// (added and removed events) is only routed.
//
// Flutter: gestures/binding.dart GestureBinding.dispatchEvent
func (b *GestureBinding) DispatchEvent(event PointerEvent, hitTestResult *HitTestResult) {
	if foundation.DebugMode && b.scheduler.Locked() {
		panic(errors.New("GestureBindingLocked", "pointer events cannot be dispatched while the binding is locked"))
	}
	if b.OnDispatchEvent != nil {
		b.OnDispatchEvent(event, hitTestResult)
	}
	if hitTestResult == nil {
		if foundation.DebugMode {
			switch event.(type) {
			case *PointerAddedEvent, *PointerRemovedEvent:
			default:
				panic(errors.New("PointerEventNotHitTested",
					"only added and removed events are dispatched without a hit test result",
					"event", event.String()))
			}
		}
		foundation.Guard(b.errorReporter, foundation.CallbackPanicked, "while routing a pointer event", nil, func() { b.PointerRouter.Route(event) })
		return
	}
	for _, entry := range hitTestResult.Path() {
		// A target that panics is reported and the next one still gets the event.
		foundation.Guard(b.errorReporter, foundation.CallbackPanicked, "while dispatching a pointer event to "+foundation.DescribeIdentity(entry.Target()),
			func() []any { return []any{"event", event.String()} },
			func() { entry.Target().HandleEvent(event.Transformed(entry.Transform()), entry) })
	}
}

// HandleEvent routes the event, and manages the gesture arena and the signal
// resolver. It is the binding's part as the last target of a hit test.
//
// Flutter: gestures/binding.dart GestureBinding.handleEvent
func (b *GestureBinding) HandleEvent(event PointerEvent, _ HitTestEntry) {
	b.PointerRouter.Route(event)
	switch e := event.(type) {
	case *PointerDownEvent, *PointerPanZoomStartEvent:
		b.GestureArena.Close(event.Pointer())
	case *PointerUpEvent, *PointerPanZoomEndEvent:
		b.GestureArena.Sweep(event.Pointer())
	case PointerSignalEvent:
		b.PointerSignalResolver.Resolve(e)
	}
}

// ResetGestureBinding forgets the hit tests of the pointers that are down. It
// is for tests.
//
// Flutter: gestures/binding.dart GestureBinding.resetGestureBinding
func (b *GestureBinding) ResetGestureBinding() { b.hitTests = map[int]*HitTestResult{} }

// Clock is the clock of the binding: the one given in the options, or the
// system clock.
//
// Flutter: gestures/binding.dart GestureBinding.samplingClock
func (b *GestureBinding) Clock() SamplingClock {
	if b.clock != nil {
		return b.clock
	}
	return systemSamplingClock{}
}

// Timer is a function that runs once, a while from now, on the UI thread.
//
// Flutter: dart:async Timer
type Timer struct {
	stopped bool
	timer   SamplingTimer
}

// NewTimer runs fn on the UI thread after d, unless the timer is cancelled
// first. The binding's clock says when.
//
// Flutter: dart:async Timer.new
func (b *GestureBinding) NewTimer(d time.Duration, fn func()) *Timer {
	t := &Timer{}
	t.timer = b.Clock().AfterFunc(d, func() {
		b.d.Post(func() {
			if t.stopped {
				return
			}
			t.stopped = true
			fn()
		})
	})
	return t
}

// Cancel stops the timer if it did not run yet.
//
// Flutter: dart:async Timer.cancel
func (t *Timer) Cancel() {
	t.stopped = true
	if t.timer != nil {
		t.timer.Stop()
	}
}

// IsActive reports whether the timer has not run or been cancelled.
//
// Flutter: dart:async Timer.isActive
func (t *Timer) IsActive() bool { return !t.stopped }

// Scheduler is the scheduler the binding was made with.
func (b *GestureBinding) Scheduler() GestureScheduler { return b.scheduler }

// Post runs fn on the UI thread, after the work that is running now: the
// microtask of Dart.
//
// Flutter: dart:async scheduleMicrotask
func (b *GestureBinding) Post(fn func()) { b.d.Post(fn) }

func (b *GestureBinding) startPeriodic(interval time.Duration, tick func()) *periodicTimer {
	p := &periodicTimer{}
	clock := b.Clock()
	var arm func()
	arm = func() {
		p.timer = clock.AfterFunc(interval, func() {
			b.d.Post(func() {
				if p.stopped {
					return
				}
				arm()
				tick()
			})
		})
	}
	arm()
	return p
}

func (b *GestureBinding) handleSampleTimeChanged() {
	if !b.scheduler.Locked() {
		if b.ResamplingEnabled {
			b.resampler.sample(b.SamplingOffset, b.Clock())
		} else {
			b.resampler.stop()
		}
	}
}
