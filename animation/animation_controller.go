// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package animation

import (
	"fmt"
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

type animationDirection int

const (
	directionForward animationDirection = iota
	directionReverse
)

var flingSpringDescription = physics.NewSpringDescriptionWithDampingRatio(1.0, 500.0)

var flingTolerance = physics.Tolerance{Velocity: math.Inf(1), Distance: 0.01, Time: 1e-3}

// AnimationBehavior is how an AnimationController behaves when animations are
// disabled in the system (the accessibility setting).
//
// Flutter: animation/animation_controller.dart AnimationBehavior
type AnimationBehavior int

const (
	// AnimationBehaviorUnspecified is the zero value: the controller's default,
	// which is AnimationBehaviorNormal (AnimationBehaviorPreserve for an
	// unbounded controller).
	AnimationBehaviorUnspecified AnimationBehavior = iota
	// AnimationBehaviorNormal makes the animations run much faster (a twentieth
	// of the duration) when the system disables animations.
	//
	// Flutter: animation/animation_controller.dart AnimationBehavior.normal
	AnimationBehaviorNormal
	// AnimationBehaviorPreserve keeps the animation running at the same speed
	// even if the system disables animations.
	//
	// Flutter: animation/animation_controller.dart AnimationBehavior.preserve
	AnimationBehaviorPreserve
)

// String is the name of the behavior.
//
// Flutter: animation/animation_controller.dart AnimationBehavior
func (b AnimationBehavior) String() string {
	switch b {
	case AnimationBehaviorNormal:
		return "AnimationBehavior.normal"
	case AnimationBehaviorPreserve:
		return "AnimationBehavior.preserve"
	}
	return "AnimationBehavior.unspecified"
}

// Environment is what an AnimationController asks of its vsync (the
// TickerProvider) besides the tickers, in place of Flutter's global
// SemanticsBinding and FlutterError.onError. A TickerProvider that implements it
// is asked for them; one that does not gets "animations are enabled" and no
// error reporter.
//
// Flutter: no counterpart (Flutter reads SemanticsBinding.instance)
type Environment interface {
	// DisableAnimations is whether the system asks for animations to be off
	// (SemanticsBinding.disableAnimations).
	DisableAnimations() bool
	// ErrorReporter receives the errors of the listeners of the controller.
	ErrorReporter() foundation.ErrorReporter
}

// AnimationControllerOpts are the optional parameters of NewAnimationController
// and NewAnimationControllerUnbounded.
//
// Flutter: animation/animation_controller.dart AnimationController.new
type AnimationControllerOpts struct {
	// Value is the initial value; the default is LowerBound (0 for unbounded).
	Value foundation.Opt[float64]
	// Duration is the length of the animation in the forward direction.
	Duration foundation.Opt[time.Duration]
	// ReverseDuration is the length of the animation in the reverse direction;
	// without it, Duration is used.
	ReverseDuration foundation.Opt[time.Duration]
	// DebugLabel names the controller in debug output.
	DebugLabel string
	// LowerBound is the value at which the animation is dismissed; the default
	// is 0. Not used by NewAnimationControllerUnbounded.
	LowerBound float64
	// UpperBound is the value at which the animation is completed; the default
	// is 1. Not used by NewAnimationControllerUnbounded.
	UpperBound foundation.Opt[float64]
	// AnimationBehavior is how the controller reacts to the system disabling
	// animations; the default is AnimationBehaviorNormal, and
	// AnimationBehaviorPreserve for an unbounded controller.
	AnimationBehavior AnimationBehavior
	// ErrorReporter receives the errors of the listeners; the default is the
	// vsync's Environment.ErrorReporter, if it has one.
	ErrorReporter foundation.ErrorReporter
}

// AnimationController is an animation that is controlled: it plays forward or
// backward, can be stopped, set to a value, or run with a simulation. By default
// it goes linearly from 0 to 1 in its duration, once per frame while the app
// runs. It calls its listeners when the value changes and its status listeners
// when the status changes.
//
// The controller asks its vsync (a TickerProvider) for a ticker, so that it
// runs only while its widget is shown. Call Dispose when you are done with it.
//
// Flutter: animation/animation_controller.dart AnimationController
type AnimationController struct {
	LocalListeners
	LocalStatusListeners

	// LowerBound is the value at which the animation is dismissed.
	//
	// Flutter: animation/animation_controller.dart AnimationController.lowerBound
	LowerBound float64
	// UpperBound is the value at which the animation is completed.
	//
	// Flutter: animation/animation_controller.dart AnimationController.upperBound
	UpperBound float64
	// DebugLabel names the controller in debug output.
	//
	// Flutter: animation/animation_controller.dart AnimationController.debugLabel
	DebugLabel string
	// AnimationBehavior is how the controller reacts to the system disabling
	// animations.
	//
	// Flutter: animation/animation_controller.dart AnimationController.animationBehavior
	AnimationBehavior AnimationBehavior
	// Duration is the length of the animation in the forward direction.
	//
	// Flutter: animation/animation_controller.dart AnimationController.duration
	Duration foundation.Opt[time.Duration]
	// ReverseDuration is the length of the animation in the reverse direction.
	//
	// Flutter: animation/animation_controller.dart AnimationController.reverseDuration
	ReverseDuration foundation.Opt[time.Duration]

	env                Environment
	ticker             *scheduler.Ticker
	simulation         physics.Simulation
	value              float64
	status             AnimationStatus
	direction          animationDirection
	lastElapsed        foundation.Opt[time.Duration]
	lastReportedStatus AnimationStatus
	disposed           bool
}

func newAnimationController(vsync scheduler.TickerProvider, o AnimationControllerOpts, lower, upper float64, behavior AnimationBehavior, initial float64) *AnimationController {
	if foundation.DebugMode && !(upper >= lower) {
		panic(errors.New("AnimationControllerBoundsInvalid", "the upper bound of an AnimationController must not be below its lower bound", "lowerBound", lower, "upperBound", upper))
	}
	c := &AnimationController{
		LowerBound: lower, UpperBound: upper, DebugLabel: o.DebugLabel, AnimationBehavior: behavior,
		Duration: o.Duration, ReverseDuration: o.ReverseDuration,
		lastReportedStatus: AnimationStatusDismissed,
	}
	if env, ok := vsync.(Environment); ok {
		c.env = env
	}
	reporter := o.ErrorReporter
	if reporter == nil && c.env != nil {
		reporter = c.env.ErrorReporter()
	}
	c.LocalListeners.SetErrorReporter(reporter)
	c.LocalStatusListeners.SetErrorReporter(reporter)
	c.ticker = vsync.CreateTicker(c.tick)
	c.internalSetValue(initial)
	return c
}

// NewAnimationController creates a controller with the bounds 0 and 1 (see the
// options). The vsync gives it its ticker; a widget State embeds a ticker
// provider for it. In debug builds it panics with the tag
// AnimationControllerBoundsInvalid when the upper bound is below the lower one.
//
// Flutter: animation/animation_controller.dart AnimationController.new
func NewAnimationController(vsync scheduler.TickerProvider, opts ...AnimationControllerOpts) *AnimationController {
	o := firstOpts(opts)
	upper := o.UpperBound.OrElse(1.0)
	behavior := o.AnimationBehavior
	if behavior == AnimationBehaviorUnspecified {
		behavior = AnimationBehaviorNormal
	}
	return newAnimationController(vsync, o, o.LowerBound, upper, behavior, o.Value.OrElse(o.LowerBound))
}

// NewAnimationControllerUnbounded creates a controller that is not bounded:
// its value can be any number, with the bounds at minus and plus infinity. The
// default AnimationBehavior is AnimationBehaviorPreserve, and the initial value
// is 0.
//
// Flutter: animation/animation_controller.dart AnimationController.unbounded
func NewAnimationControllerUnbounded(vsync scheduler.TickerProvider, opts ...AnimationControllerOpts) *AnimationController {
	o := firstOpts(opts)
	behavior := o.AnimationBehavior
	if behavior == AnimationBehaviorUnspecified {
		behavior = AnimationBehaviorPreserve
	}
	return newAnimationController(vsync, o, math.Inf(-1), math.Inf(1), behavior, o.Value.OrElse(0))
}

func (c *AnimationController) enableAnimations() bool {
	if c.AnimationBehavior == AnimationBehaviorPreserve {
		return true
	}
	return c.env == nil || !c.env.DisableAnimations()
}

// View is the controller as a plain animation: it hides the methods that
// control it.
//
// Flutter: animation/animation_controller.dart AnimationController.view
func (c *AnimationController) View() Animation[float64] { return c }

// Resync replaces the ticker with one from the new vsync, carrying over the
// running animation (see Ticker.AbsorbTicker).
//
// Flutter: animation/animation_controller.dart AnimationController.resync
func (c *AnimationController) Resync(vsync scheduler.TickerProvider) {
	old := c.ticker
	c.ticker = vsync.CreateTicker(c.tick)
	c.ticker.AbsorbTicker(old)
}

// Value is the current value of the animation, from LowerBound to UpperBound.
//
// Flutter: animation/animation_controller.dart AnimationController.value
func (c *AnimationController) Value() float64 { return c.value }

// SetValue stops the animation and sets its value (clamped to the bounds). The
// listeners are called, and the status listeners if the status changed.
//
// Flutter: animation/animation_controller.dart AnimationController.value=
func (c *AnimationController) SetValue(newValue float64) {
	c.Stop()
	c.internalSetValue(newValue)
	c.NotifyListeners()
	c.checkStatusChanged()
}

// Reset sets the value to the lower bound, which stops the animation and makes
// its status dismissed.
//
// Flutter: animation/animation_controller.dart AnimationController.reset
func (c *AnimationController) Reset() { c.SetValue(c.LowerBound) }

// Velocity is the rate of change of the value, per second; 0 when the animation
// is not running.
//
// Flutter: animation/animation_controller.dart AnimationController.velocity
func (c *AnimationController) Velocity() float64 {
	if !c.IsAnimating() {
		return 0
	}
	elapsed, _ := c.lastElapsed.Get()
	return c.simulation.Dx(float64(elapsed.Microseconds()) / 1e6)
}

func (c *AnimationController) internalSetValue(newValue float64) {
	c.value = ui.ClampDouble(newValue, c.LowerBound, c.UpperBound)
	switch {
	case c.value == c.LowerBound:
		c.status = AnimationStatusDismissed
	case c.value == c.UpperBound:
		c.status = AnimationStatusCompleted
	case c.direction == directionForward:
		c.status = AnimationStatusForward
	default:
		c.status = AnimationStatusReverse
	}
}

// LastElapsedDuration is the amount of time the ticker has been running for the
// current animation, as of the last frame; none if the animation is not running.
//
// Flutter: animation/animation_controller.dart AnimationController.lastElapsedDuration
func (c *AnimationController) LastElapsedDuration() foundation.Opt[time.Duration] {
	return c.lastElapsed
}

// IsAnimating is whether the animation is running (its ticker is active), even
// when the ticker is muted.
//
// Flutter: animation/animation_controller.dart AnimationController.isAnimating
func (c *AnimationController) IsAnimating() bool { return c.ticker != nil && c.ticker.IsActive() }

// Status is the current status of the animation.
//
// Flutter: animation/animation_controller.dart AnimationController.status
func (c *AnimationController) Status() AnimationStatus { return c.status }

// IsDisposed is whether Dispose was called.
//
// Flutter: no counterpart (Flutter reads the ticker, which is null after dispose)
func (c *AnimationController) IsDisposed() bool { return c.disposed }

func (c *AnimationController) assertUsable(method string) bool {
	if c.ticker != nil {
		return true
	}
	if foundation.DebugMode {
		panic(errors.New("AnimationControllerDisposed", fmt.Sprintf("AnimationController.%s() called after AnimationController.dispose(); AnimationController methods should not be used after calling dispose", method),
			"method", method))
	}
	return false
}

func (c *AnimationController) assertDuration(method string, ok bool, hint string) {
	if foundation.DebugMode && !ok {
		panic(errors.New("AnimationControllerNoDuration", fmt.Sprintf("AnimationController.%s() called with no duration: %s", method, hint), "method", method))
	}
}

// Forward starts running the animation forward, towards UpperBound, from the
// current value or from the value given. The returned future completes when the
// animation ends, and is canceled if it is stopped. In debug builds it panics
// with the tag AnimationControllerNoDuration when there is no Duration, and
// AnimationControllerDisposed after Dispose.
//
// Flutter: animation/animation_controller.dart AnimationController.forward
func (c *AnimationController) Forward(from ...float64) *scheduler.TickerFuture {
	c.assertDuration("forward", c.Duration.IsSome(), "the Duration should be set, in the constructor or later, before calling forward()")
	if !c.assertUsable("forward") {
		return scheduler.NewTickerFutureComplete()
	}
	c.direction = directionForward
	if len(from) > 0 {
		c.SetValue(from[0])
	}
	return c.animateToInternal(c.UpperBound, foundation.None[time.Duration](), Curves.Linear)
}

// Reverse starts running the animation in reverse, towards LowerBound, from the
// current value or from the value given. In debug builds it panics with the tag
// AnimationControllerNoDuration when there is neither Duration nor
// ReverseDuration.
//
// Flutter: animation/animation_controller.dart AnimationController.reverse
func (c *AnimationController) Reverse(from ...float64) *scheduler.TickerFuture {
	c.assertDuration("reverse", c.Duration.IsSome() || c.ReverseDuration.IsSome(), "the Duration or ReverseDuration should be set before calling reverse()")
	if !c.assertUsable("reverse") {
		return scheduler.NewTickerFutureComplete()
	}
	c.direction = directionReverse
	if len(from) > 0 {
		c.SetValue(from[0])
	}
	return c.animateToInternal(c.LowerBound, foundation.None[time.Duration](), Curves.Linear)
}

// Toggle runs the animation in reverse if it is going forward or completed,
// and forward otherwise.
//
// Flutter: animation/animation_controller.dart AnimationController.toggle
func (c *AnimationController) Toggle(from ...float64) *scheduler.TickerFuture {
	d := c.Duration
	if c.status.IsForwardOrCompleted() && d.IsNone() {
		d = c.ReverseDuration
	}
	c.assertDuration("toggle", d.IsSome(), "the Duration should be set before calling toggle()")
	if !c.assertUsable("toggle") {
		return scheduler.NewTickerFutureComplete()
	}
	if c.status.IsForwardOrCompleted() {
		c.direction = directionReverse
	} else {
		c.direction = directionForward
	}
	if len(from) > 0 {
		c.SetValue(from[0])
	}
	target := c.UpperBound
	if c.direction == directionReverse {
		target = c.LowerBound
	}
	return c.animateToInternal(target, foundation.None[time.Duration](), Curves.Linear)
}

// AnimateOpts are the optional parameters of AnimateTo and AnimateBack.
//
// Flutter: animation/animation_controller.dart AnimationController.animateTo
type AnimateOpts struct {
	// Duration is the length of the animation; the default is the controller's
	// Duration (ReverseDuration for AnimateBack when it is set), scaled by the
	// distance left.
	Duration foundation.Opt[time.Duration]
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve Curve
}

// AnimateTo drives the animation from its current value to the target, in the
// forward direction.
//
// Flutter: animation/animation_controller.dart AnimationController.animateTo
func (c *AnimationController) AnimateTo(target float64, opts ...AnimateOpts) *scheduler.TickerFuture {
	o := firstOpts(opts)
	c.assertDuration("animateTo", c.Duration.IsSome() || o.Duration.IsSome(), "pass a duration, or set the Duration of the controller")
	if !c.assertUsable("animateTo") {
		return scheduler.NewTickerFutureComplete()
	}
	c.direction = directionForward
	return c.animateToInternal(target, o.Duration, o.Curve)
}

// AnimateBack drives the animation from its current value to the target, in the
// reverse direction.
//
// Flutter: animation/animation_controller.dart AnimationController.animateBack
func (c *AnimationController) AnimateBack(target float64, opts ...AnimateOpts) *scheduler.TickerFuture {
	o := firstOpts(opts)
	c.assertDuration("animateBack", c.Duration.IsSome() || c.ReverseDuration.IsSome() || o.Duration.IsSome(), "pass a duration, or set the Duration or ReverseDuration of the controller")
	if !c.assertUsable("animateBack") {
		return scheduler.NewTickerFutureComplete()
	}
	c.direction = directionReverse
	return c.animateToInternal(target, o.Duration, o.Curve)
}

// scaleDuration is Dart's Duration * double: the microseconds scaled and rounded.
func scaleDuration(d time.Duration, factor float64) time.Duration {
	return time.Duration(math.Round(float64(d.Microseconds())*factor)) * time.Microsecond
}

func (c *AnimationController) animateToInternal(target float64, duration foundation.Opt[time.Duration], curve Curve) *scheduler.TickerFuture {
	if curve == nil {
		curve = Curves.Linear
	}
	scale := 1.0
	if !c.enableAnimations() {
		scale = 0.05
	}
	var simulationDuration time.Duration
	if d, ok := duration.Get(); ok {
		simulationDuration = d
		if target == c.value {
			simulationDuration = 0
		}
	} else {
		span := c.UpperBound - c.LowerBound
		remaining := 1.0
		if !math.IsInf(span, 0) && !math.IsNaN(span) {
			remaining = math.Abs(target-c.value) / span
		}
		directionDuration := c.Duration.OrElse(0)
		if c.direction == directionReverse && c.ReverseDuration.IsSome() {
			directionDuration = c.ReverseDuration.OrElse(0)
		}
		simulationDuration = scaleDuration(directionDuration, remaining)
	}
	c.Stop()
	if simulationDuration == 0 {
		if c.value != target {
			c.value = ui.ClampDouble(target, c.LowerBound, c.UpperBound)
			c.NotifyListeners()
		}
		if c.direction == directionForward {
			c.status = AnimationStatusCompleted
		} else {
			c.status = AnimationStatusDismissed
		}
		c.checkStatusChanged()
		return scheduler.NewTickerFutureComplete()
	}
	return c.startSimulation(newInterpolationSimulation(c.value, target, simulationDuration, curve, scale))
}

// RepeatOpts are the optional parameters of Repeat.
//
// Flutter: animation/animation_controller.dart AnimationController.repeat
type RepeatOpts struct {
	// Min is the lower value of the repetition; the default is LowerBound.
	Min foundation.Opt[float64]
	// Max is the upper value of the repetition; the default is UpperBound.
	Max foundation.Opt[float64]
	// Reverse makes the animation go back and forth instead of jumping from the
	// end to the start.
	Reverse bool
	// Period is the length of one repetition; the default is the controller's
	// Duration.
	Period foundation.Opt[time.Duration]
	// Count is the number of repetitions; the default is to repeat forever. It
	// must be greater than zero.
	Count foundation.Opt[int]
}

// Repeat starts running the animation in a loop, between Min and Max. The
// returned future never completes while the animation repeats (when Count is
// set it completes after that many repetitions). In debug builds it panics with
// the tag AnimationControllerRepeatInvalid when the range or the count is not
// valid, and AnimationControllerNoDuration without a period or Duration.
//
// Flutter: animation/animation_controller.dart AnimationController.repeat
func (c *AnimationController) Repeat(opts ...RepeatOpts) *scheduler.TickerFuture {
	o := firstOpts(opts)
	lo := o.Min.OrElse(c.LowerBound)
	hi := o.Max.OrElse(c.UpperBound)
	period, hasPeriod := o.Period.Get()
	if !hasPeriod {
		period, hasPeriod = c.Duration.Get()
	}
	c.assertDuration("repeat", hasPeriod, "pass a period, or set the Duration of the controller")
	if foundation.DebugMode {
		if !(hi >= lo && hi <= c.UpperBound && lo >= c.LowerBound) {
			panic(errors.New("AnimationControllerRepeatInvalid", "the min and max of repeat() must be within the bounds of the controller, with the max not below the min", "min", lo, "max", hi))
		}
		if n, ok := o.Count.Get(); ok && n <= 0 {
			panic(errors.New("AnimationControllerRepeatInvalid", "the count of repeat() must be greater than zero", "count", n))
		}
	}
	if !c.assertUsable("repeat") {
		return scheduler.NewTickerFutureComplete()
	}
	c.Stop()
	return c.startSimulation(newRepeatingSimulation(c.value, lo, hi, o.Reverse, period, c.setDirection, o.Count))
}

func (c *AnimationController) setDirection(d animationDirection) {
	c.direction = d
	if d == directionForward {
		c.status = AnimationStatusForward
	} else {
		c.status = AnimationStatusReverse
	}
	c.checkStatusChanged()
}

// FlingOpts are the optional parameters of Fling.
//
// Flutter: animation/animation_controller.dart AnimationController.fling
type FlingOpts struct {
	// Velocity is the velocity in the animation's range per second; the default
	// is 1. A negative velocity flings towards LowerBound.
	Velocity foundation.Opt[float64]
	// SpringDescription is the spring that drives the fling; the default is a
	// critically damped one (mass 1, stiffness 500).
	SpringDescription foundation.Opt[physics.SpringDescription]
	// AnimationBehavior overrides the controller's behavior for this fling.
	AnimationBehavior AnimationBehavior
}

// Fling drives the animation with a spring (the default is critically damped)
// towards the upper bound, or the lower bound if the velocity is negative. In
// debug builds it panics with the tag AnimationControllerFlingUnderdamped when
// the spring is underdamped.
//
// Flutter: animation/animation_controller.dart AnimationController.fling
func (c *AnimationController) Fling(opts ...FlingOpts) *scheduler.TickerFuture {
	o := firstOpts(opts)
	velocity := o.Velocity.OrElse(1.0)
	spring := o.SpringDescription.OrElse(flingSpringDescription)
	var target float64
	if velocity < 0 {
		c.direction = directionReverse
		target = c.LowerBound - flingTolerance.Distance
	} else {
		c.direction = directionForward
		target = c.UpperBound + flingTolerance.Distance
	}
	behavior := o.AnimationBehavior
	if behavior == AnimationBehaviorUnspecified {
		behavior = c.AnimationBehavior
	}
	scale := 200.0
	if behavior == AnimationBehaviorPreserve || c.env == nil || !c.env.DisableAnimations() {
		scale = 1.0
	}
	sim := physics.NewSpringSimulation(spring, c.value, target, velocity*scale)
	sim.SetTolerance(flingTolerance)
	if foundation.DebugMode && sim.Type() == physics.SpringTypeUnderDamped {
		panic(errors.New("AnimationControllerFlingUnderdamped", "the specified spring simulation is underdamped, which oscillates instead of flinging; use a different spring description, or AnimateWith with an explicit SpringSimulation",
			"spring", spring.String()))
	}
	c.Stop()
	return c.startSimulation(sim)
}

// AnimateWith drives the animation with a simulation, in the forward direction.
// The value is clamped to the bounds. The animation is done when the simulation
// is.
//
// Flutter: animation/animation_controller.dart AnimationController.animateWith
func (c *AnimationController) AnimateWith(simulation physics.Simulation) *scheduler.TickerFuture {
	if !c.assertUsable("animateWith") {
		return scheduler.NewTickerFutureComplete()
	}
	c.Stop()
	c.direction = directionForward
	return c.startSimulation(simulation)
}

// AnimateBackWith drives the animation with a simulation, in the reverse
// direction.
//
// Flutter: animation/animation_controller.dart AnimationController.animateBackWith
func (c *AnimationController) AnimateBackWith(simulation physics.Simulation) *scheduler.TickerFuture {
	if !c.assertUsable("animateBackWith") {
		return scheduler.NewTickerFutureComplete()
	}
	c.Stop()
	c.direction = directionReverse
	return c.startSimulation(simulation)
}

func (c *AnimationController) startSimulation(simulation physics.Simulation) *scheduler.TickerFuture {
	c.simulation = simulation
	c.lastElapsed = foundation.Some(time.Duration(0))
	c.value = ui.ClampDouble(simulation.X(0.0), c.LowerBound, c.UpperBound)
	result := c.ticker.Start()
	if c.direction == directionForward {
		c.status = AnimationStatusForward
	} else {
		c.status = AnimationStatusReverse
	}
	c.checkStatusChanged()
	return result
}

// StopOpts are the optional parameters of Stop.
//
// Flutter: animation/animation_controller.dart AnimationController.stop
type StopOpts struct {
	// Canceled is whether the future of the animation is canceled instead of
	// completed; the default is true.
	Canceled foundation.Opt[bool]
}

// Stop stops running the animation. It does not change the value; the future
// that the animation returned is canceled (or completed, with Canceled false).
//
// Flutter: animation/animation_controller.dart AnimationController.stop
func (c *AnimationController) Stop(opts ...StopOpts) {
	o := firstOpts(opts)
	if !c.assertUsable("stop") {
		return
	}
	c.simulation = nil
	c.lastElapsed = foundation.None[time.Duration]()
	c.ticker.Stop(scheduler.TickerStopOpts{Canceled: o.Canceled.OrElse(true)})
}

// Dispose releases the resources of the controller: its ticker and listeners.
// It must be called once, and the controller can't be used afterwards. In debug
// builds a second call panics with the tag AnimationControllerDisposedTwice.
//
// Flutter: animation/animation_controller.dart AnimationController.dispose
func (c *AnimationController) Dispose() {
	if c.ticker == nil {
		if foundation.DebugMode {
			panic(errors.New("AnimationControllerDisposedTwice", "AnimationController.Dispose() called more than once; a given AnimationController cannot be disposed more than once",
				"controller", c.String()))
		}
		return
	}
	c.ticker.Dispose()
	c.ticker = nil
	c.disposed = true
	c.ClearStatusListeners()
	c.ClearListeners()
}

func (c *AnimationController) checkStatusChanged() {
	if newStatus := c.status; c.lastReportedStatus != newStatus {
		c.lastReportedStatus = newStatus
		c.NotifyStatusListeners(newStatus)
	}
}

func (c *AnimationController) tick(elapsed time.Duration) {
	c.lastElapsed = foundation.Some(elapsed)
	seconds := float64(elapsed.Microseconds()) / 1e6
	c.value = ui.ClampDouble(c.simulation.X(seconds), c.LowerBound, c.UpperBound)
	if c.simulation.IsDone(seconds) {
		if c.direction == directionForward {
			c.status = AnimationStatusCompleted
		} else {
			c.status = AnimationStatusDismissed
		}
		c.Stop(StopOpts{Canceled: foundation.Some(false)})
	}
	c.NotifyListeners()
	c.checkStatusChanged()
}

// String describes the controller.
//
// Flutter: animation/animation_controller.dart AnimationController.toStringDetails
func (c *AnimationController) String() string {
	paused := ""
	if !c.IsAnimating() {
		paused = "; paused"
	}
	ticker := ""
	if c.ticker == nil {
		ticker = "; DISPOSED"
	} else if c.ticker.Muted() {
		ticker = "; silenced"
	}
	label := ""
	if foundation.DebugMode && c.DebugLabel != "" {
		label = "; for " + c.DebugLabel
	}
	return fmt.Sprintf("%s(%s %.3f%s%s%s)", foundation.DescribeIdentity(c), statusSymbol(c.status), c.value, paused, ticker, label)
}

type interpolationSimulation struct {
	physics.SimulationBase
	durationInSeconds float64
	begin, end        float64
	curve             Curve
}

func newInterpolationSimulation(begin, end float64, duration time.Duration, curve Curve, scale float64) *interpolationSimulation {
	return &interpolationSimulation{
		SimulationBase:    physics.NewSimulationBase(foundation.None[physics.Tolerance]()),
		durationInSeconds: float64(duration.Microseconds()) * scale / 1e6,
		begin:             begin, end: end, curve: curve,
	}
}

func (s *interpolationSimulation) X(timeInSeconds float64) float64 {
	t := ui.ClampDouble(timeInSeconds/s.durationInSeconds, 0, 1)
	switch t {
	case 0:
		return s.begin
	case 1:
		return s.end
	}
	return s.begin + (s.end-s.begin)*s.curve.Transform(t)
}

func (s *interpolationSimulation) Dx(timeInSeconds float64) float64 {
	epsilon := s.Tolerance().Time
	return (s.X(timeInSeconds+epsilon) - s.X(timeInSeconds-epsilon)) / (2 * epsilon)
}

func (s *interpolationSimulation) IsDone(timeInSeconds float64) bool {
	return timeInSeconds > s.durationInSeconds
}

type repeatingSimulation struct {
	physics.SimulationBase
	min, max        float64
	reverse         bool
	count           foundation.Opt[int]
	directionSetter func(animationDirection)

	periodInSeconds float64
	initialT        float64
	exitTime        float64
}

func newRepeatingSimulation(initialValue, min, max float64, reverse bool, period time.Duration, setter func(animationDirection), count foundation.Opt[int]) *repeatingSimulation {
	periodSeconds := float64(period.Microseconds()) / 1e6
	s := &repeatingSimulation{
		SimulationBase: physics.NewSimulationBase(foundation.None[physics.Tolerance]()),
		min:            min, max: max, reverse: reverse, count: count, directionSetter: setter,
		periodInSeconds: periodSeconds,
	}
	if max != min {
		s.initialT = ((ui.ClampDouble(initialValue, min, max) - min) / (max - min)) * periodSeconds
	}
	if n, ok := count.Get(); ok {
		s.exitTime = float64(n)*periodSeconds - s.initialT
	}
	return s
}

func (s *repeatingSimulation) X(timeInSeconds float64) float64 {
	total := timeInSeconds + s.initialT
	t := math.Mod(total/s.periodInSeconds, 1.0)
	playingReverse := int64(total/s.periodInSeconds)%2 != 0
	if s.reverse && playingReverse {
		s.directionSetter(directionReverse)
		return ui.LerpDouble(s.max, s.min, t)
	}
	s.directionSetter(directionForward)
	return ui.LerpDouble(s.min, s.max, t)
}

func (s *repeatingSimulation) Dx(float64) float64 { return (s.max - s.min) / s.periodInSeconds }

func (s *repeatingSimulation) IsDone(timeInSeconds float64) bool {
	return s.count.IsSome() && timeInSeconds >= s.exitTime
}
