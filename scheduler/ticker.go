// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package scheduler

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// TickerCallback is called by a Ticker once per frame, with the time since the
// ticker started (with the time dilation).
//
// Flutter: scheduler/ticker.dart TickerCallback
type TickerCallback func(elapsed time.Duration)

// TickerProvider is an object that makes Tickers. A State that needs one
// embeds SingleTickerProvider or TickerProviderMixin (steps 8.2 and 7.1).
//
// Flutter: scheduler/ticker.dart TickerProvider
type TickerProvider interface {
	// CreateTicker makes a ticker that calls onTick. The caller must Dispose
	// it.
	//
	// Flutter: scheduler/ticker.dart TickerProvider.createTicker
	CreateTicker(onTick TickerCallback) *Ticker
}

// Ticker calls its callback once per frame, from the frame after it is started
// until it is stopped. The time it passes is relative to the first frame it
// ticked in, so that animations start at zero.
//
// A ticker stops ticking (but stays active) while it is muted, and while frames
// are disabled (the app is hidden or paused).
//
// Flutter: scheduler/ticker.dart Ticker
type Ticker struct {
	binding *SchedulerBinding
	onTick  TickerCallback

	// ForceFrames makes the ticker schedule frames with ScheduleForcedFrame,
	// which works while frames are disabled.
	//
	// Flutter: scheduler/ticker.dart Ticker.forceFrames
	ForceFrames bool

	// DebugLabel names the ticker in debug output.
	//
	// Flutter: scheduler/ticker.dart Ticker.debugLabel
	DebugLabel string

	// OnDispose, if set, is called when the ticker is disposed, before it stops.
	// A TickerProvider uses it to forget the tickers it made.
	//
	// Deviation: Flutter's _WidgetTicker overrides dispose in a subclass.
	OnDispose func()

	future      *TickerFuture
	muted       bool
	startTime   foundation.Opt[time.Duration]
	animationID foundation.Opt[int]
	disposed    bool
}

// TickerOpts are the optional parameters of NewTicker.
//
// Flutter: scheduler/ticker.dart Ticker.new
type TickerOpts struct {
	// DebugLabel names the ticker in debug output.
	DebugLabel string
}

// NewTicker creates a ticker that calls onTick, on the binding's frames. It is
// not started; call Start. Dispose it when it is no longer needed.
//
// Flutter: scheduler/ticker.dart Ticker.new
func NewTicker(binding *SchedulerBinding, onTick TickerCallback, opts ...TickerOpts) *Ticker {
	o := firstOpts(opts)
	return &Ticker{binding: binding, onTick: onTick, DebugLabel: o.DebugLabel}
}

// Muted reports whether the ticker is muted: it stays active but does not tick.
//
// Flutter: scheduler/ticker.dart Ticker.muted
func (t *Ticker) Muted() bool { return t.muted }

// SetMuted mutes or unmutes the ticker. A muted ticker does not tick and
// schedules no frames; an unmuted one that is active starts again, with its
// start time unchanged, so that time passes for it while it was muted.
//
// Flutter: scheduler/ticker.dart Ticker.muted=
func (t *Ticker) SetMuted(value bool) {
	if value == t.muted {
		return
	}
	t.muted = value
	if value {
		t.unscheduleTick()
	} else if t.shouldScheduleTick() {
		t.scheduleTick(false)
	}
}

// IsTicking reports whether the ticker will call its callback on frames: it is
// active, not muted, and frames are enabled (or a frame is being built).
//
// Flutter: scheduler/ticker.dart Ticker.isTicking
func (t *Ticker) IsTicking() bool {
	if t.future == nil || t.muted {
		return false
	}
	if t.binding.FramesEnabled() {
		return true
	}
	// For example, in a forced frame.
	return t.binding.SchedulerPhase() != SchedulerPhaseIdle
}

// IsActive reports whether the ticker was started and not stopped. A muted
// ticker is active.
//
// Flutter: scheduler/ticker.dart Ticker.isActive
func (t *Ticker) IsActive() bool { return t.future != nil }

// Start starts the ticker, and returns the TickerFuture that completes when it
// is stopped. Starting a ticker that is active panics with the tag
// TickerStartedTwice in debug builds, and so does starting a disposed one with
// TickerUsedAfterDispose.
//
// Flutter: scheduler/ticker.dart Ticker.start
func (t *Ticker) Start() *TickerFuture {
	if foundation.DebugMode {
		if t.disposed {
			panic(errors.New("TickerUsedAfterDispose", "A ticker was started after it was disposed.",
				"ticker", t.String()))
		}
		if t.IsActive() {
			panic(errors.New("TickerStartedTwice",
				"A ticker that is already active cannot be started again without first stopping it.",
				"ticker", t.String()))
		}
	}
	t.future = newTickerFuture()
	if t.shouldScheduleTick() {
		t.scheduleTick(false)
	}
	if phase := t.binding.SchedulerPhase(); phase > SchedulerPhaseIdle && phase < SchedulerPhasePostFrameCallbacks {
		t.startTime = foundation.Some(t.binding.CurrentFrameTimeStamp())
	}
	return t.future
}

// TickerStopOpts are the optional parameters of Ticker.Stop.
//
// Flutter: scheduler/ticker.dart Ticker.stop
type TickerStopOpts struct {
	// Canceled makes the TickerFuture end as canceled (its OrCancel channel
	// gets a TickerCanceled error) instead of completing.
	Canceled bool
}

// Stop stops the ticker, if it is active. The TickerFuture it returned from
// Start completes, or ends as canceled with Canceled.
//
// Flutter: scheduler/ticker.dart Ticker.stop
func (t *Ticker) Stop(opts ...TickerStopOpts) {
	o := firstOpts(opts)
	if !t.IsActive() {
		return
	}
	// The future is taken into a local variable so that IsTicking is false when
	// it completes: IsTicking looks at the future.
	local := t.future
	t.future = nil
	t.startTime = foundation.None[time.Duration]()
	t.unscheduleTick()
	if o.Canceled {
		local.cancel(t)
	} else {
		local.complete()
	}
}

func (t *Ticker) scheduled() bool { return t.animationID.IsSome() }

func (t *Ticker) shouldScheduleTick() bool { return !t.muted && t.IsActive() && !t.scheduled() }

func (t *Ticker) tick(timeStamp time.Duration) {
	t.animationID = foundation.None[int]()
	if !t.startTime.IsSome() {
		t.startTime = foundation.Some(timeStamp)
	}
	t.onTick(timeStamp - t.startTime.Value())
	// The callback may have scheduled another tick already, for example by
	// calling Stop and then Start again.
	if t.shouldScheduleTick() {
		t.scheduleTick(true)
	}
}

func (t *Ticker) scheduleTick(rescheduling bool) {
	if t.ForceFrames {
		t.binding.ScheduleForcedFrame()
	} else {
		t.binding.ScheduleFrame()
	}
	id := t.binding.ScheduleFrameCallback(t.tick, ScheduleFrameCallbackOpts{Rescheduling: rescheduling, NoNewFrame: true})
	t.animationID = foundation.Some(id)
}

func (t *Ticker) unscheduleTick() {
	if id, ok := t.animationID.Get(); ok {
		t.binding.CancelFrameCallbackWithID(id)
		t.animationID = foundation.None[int]()
	}
}

// AbsorbTicker makes this ticker take over the state of another: if the other
// was started, this one is active with the same TickerFuture and start time,
// and keeps ticking from where it was. The other is disposed. This ticker must
// not be active. A State uses it when its TickerProvider changes.
//
// Flutter: scheduler/ticker.dart Ticker.absorbTicker
func (t *Ticker) AbsorbTicker(original *Ticker) {
	if original.future != nil {
		t.future = original.future
		t.startTime = original.startTime
		if t.shouldScheduleTick() {
			t.scheduleTick(false)
		}
		original.future = nil // so that disposing the original does not cancel it
		original.unscheduleTick()
	}
	original.Dispose()
}

// Dispose stops the ticker and cancels its TickerFuture, if it is active. A
// disposed ticker cannot be started.
//
// Flutter: scheduler/ticker.dart Ticker.dispose
func (t *Ticker) Dispose() {
	if t.OnDispose != nil {
		t.OnDispose()
	}
	if t.future != nil {
		local := t.future
		t.future = nil
		t.unscheduleTick()
		local.cancel(t)
	}
	t.disposed = true
}

// String describes the ticker, such as "Ticker(my label)".
//
// Flutter: scheduler/ticker.dart Ticker.toString
func (t *Ticker) String() string {
	label := ""
	if foundation.DebugMode {
		label = t.DebugLabel
	}
	return foundation.RuntimeTypeName(t) + "(" + label + ")"
}

// TickerFuture is what a Ticker's Start returns: it completes when the ticker
// is stopped. Flutter's is a Future; Glutter's has a channel and callbacks
// (SPEC §12.6).
//
// A canceled ticker never completes its Done channel; its OrCancel channel gets
// a TickerCanceled error instead.
//
// Callbacks run at once, on the goroutine that stops the ticker (the UI
// thread), not in a later microtask as Dart's futures do.
//
// Flutter: scheduler/ticker.dart TickerFuture
type TickerFuture struct {
	// completed: None while the ticker runs, Some(true) when it completed and
	// Some(false) when it was canceled.
	completed foundation.Opt[bool]
	done      chan struct{}
	onDone    []func()
	onEnd     []func(err error)
}

func newTickerFuture() *TickerFuture { return &TickerFuture{done: make(chan struct{})} }

// NewTickerFutureComplete creates a future that is complete already, for an
// animation that does not run.
//
// Flutter: scheduler/ticker.dart TickerFuture.complete
func NewTickerFutureComplete() *TickerFuture {
	f := newTickerFuture()
	f.complete()
	return f
}

// NewTickerFuturePending creates a future that is not complete, and the function
// that completes it. Code that has no ticker (a scroll activity that completes
// when it is disposed) uses it where Dart uses a Completer<void>. Calling the
// function again does nothing.
//
// Flutter: no counterpart (dart:async Completer<void>)
func NewTickerFuturePending() (future *TickerFuture, complete func()) {
	f := newTickerFuture()
	return f, func() {
		if !f.completed.IsSome() {
			f.complete()
		}
	}
}

// WaitTickerFutures returns a future that completes when all the futures have
// completed (immediately if there are none). A future that is canceled never
// completes it.
//
// Flutter: no counterpart (dart:async Future.wait)
func WaitTickerFutures(futures ...*TickerFuture) *TickerFuture {
	result, complete := NewTickerFuturePending()
	remaining := len(futures)
	if remaining == 0 {
		complete()
		return result
	}
	for _, f := range futures {
		f.Then(func() {
			remaining--
			if remaining == 0 {
				complete()
			}
		})
	}
	return result
}

func (f *TickerFuture) complete() {
	f.completed = foundation.Some(true)
	close(f.done)
	for _, cb := range f.onDone {
		cb()
	}
	for _, cb := range f.onEnd {
		cb(nil)
	}
	f.onDone, f.onEnd = nil, nil
}

func (f *TickerFuture) cancel(ticker *Ticker) {
	f.completed = foundation.Some(false)
	f.onDone = nil
	for _, cb := range f.onEnd {
		cb(newTickerCanceled(ticker))
	}
	f.onEnd = nil
}

// Done returns a channel that is closed when the ticker completes (is stopped
// without being canceled). It is never closed if the ticker is canceled; use
// OrCancel to learn of both.
//
// Flutter: scheduler/ticker.dart TickerFuture.then
func (f *TickerFuture) Done() <-chan struct{} { return f.done }

// Then calls fn when the ticker completes, at once if it already did. It is
// never called if the ticker is canceled.
//
// Flutter: scheduler/ticker.dart TickerFuture.then
func (f *TickerFuture) Then(fn func()) {
	switch {
	case !f.completed.IsSome():
		f.onDone = append(f.onDone, fn)
	case f.completed.Value():
		fn()
	}
}

// OrCancel returns a channel that gets nil when the ticker completes and a
// TickerCanceled error when it is canceled; the value is there at once if that
// already happened. Each call returns a new channel.
//
// Flutter: scheduler/ticker.dart TickerFuture.orCancel
func (f *TickerFuture) OrCancel() <-chan error {
	ch := make(chan error, 1)
	f.whenEnded(func(err error) { ch <- err })
	return ch
}

// whenEnded calls fn when the future completes (with nil) or is canceled (with
// the error), at once if it already ended.
func (f *TickerFuture) whenEnded(fn func(err error)) {
	switch {
	case !f.completed.IsSome():
		f.onEnd = append(f.onEnd, fn)
	case f.completed.Value():
		fn(nil)
	default:
		// Canceled before anyone asked: the error does not name the ticker.
		fn(newTickerCanceled(nil))
	}
}

// WhenCompleteOrCancel calls fn when the ticker completes or is canceled, at
// once if it already ended.
//
// Flutter: scheduler/ticker.dart TickerFuture.whenCompleteOrCancel
func (f *TickerFuture) WhenCompleteOrCancel(fn func()) {
	f.whenEnded(func(error) { fn() })
}

// Completed reports whether the ticker completed (true) or was canceled
// (false); the second result is false while it runs.
func (f *TickerFuture) Completed() (completed, ended bool) {
	v, ok := f.completed.Get()
	return v, ok
}

// String describes the future, such as "TickerFuture#1a2b3(active)".
//
// Flutter: scheduler/ticker.dart TickerFuture.toString
func (f *TickerFuture) String() string {
	state := "active"
	if v, ok := f.completed.Get(); ok {
		state = "canceled"
		if v {
			state = "complete"
		}
	}
	return foundation.DescribeIdentity(f) + "(" + state + ")"
}

// newTickerCanceled makes the error of a canceled ticker.
func newTickerCanceled(ticker *Ticker) error {
	if ticker != nil {
		return errors.New("TickerCanceled", "This ticker was canceled: "+ticker.String())
	}
	return errors.New("TickerCanceled", "The ticker was canceled before the \"orCancel\" property was first used.")
}
