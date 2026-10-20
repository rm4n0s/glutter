// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package animation

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/physics"
)

// Ported from packages/flutter/test/animation/animation_controller_test.dart.

func dur(d time.Duration) AnimationControllerOpts {
	return AnimationControllerOpts{Duration: foundation.Some(d)}
}

type statusLog []AnimationStatus

func logStatuses(c Animation[float64]) *statusLog {
	l := &statusLog{}
	c.AddStatusListener(func(s AnimationStatus) { *l = append(*l, s) })
	return l
}

func (l *statusLog) clear() { *l = nil }

func (l *statusLog) want(t *testing.T, name string, want ...AnimationStatus) {
	t.Helper()
	if len(*l) != len(want) {
		t.Errorf("%s: status log = %v, want %v", name, *l, want)
		return
	}
	for i := range want {
		if (*l)[i] != want[i] {
			t.Errorf("%s: status log = %v, want %v", name, *l, want)
			return
		}
	}
}

func logValues(c Animation[float64]) *[]float64 {
	l := &[]float64{}
	c.AddListener(func() { *l = append(*l, c.Value()) })
	return l
}

const (
	dismissed = AnimationStatusDismissed
	forward   = AnimationStatusForward
	reverse   = AnimationStatusReverse
	completed = AnimationStatusCompleted
)

func TestCanStartTheAnimationWhenResyncIsCalled(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(ms(100)))
	controller.Forward()
	controller.Resync(newTester())
	tt.tick(time.Second)
	controller.Dispose()
}

func TestCanSetValueDuringStatusCallback(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(ms(100)))
	didComplete, didDismiss := false, false
	controller.AddStatusListener(func(s AnimationStatus) {
		switch s {
		case completed:
			didComplete = true
			controller.SetValue(0)
			controller.Forward()
		case dismissed:
			didDismiss = true
			controller.SetValue(0)
			controller.Forward()
		}
	})
	controller.Forward()
	if didComplete || didDismiss {
		t.Fatal("callbacks ran too early")
	}
	tt.tick(1 * time.Second)
	if didComplete || didDismiss {
		t.Fatal("callbacks ran too early after the first frame")
	}
	tt.tick(2 * time.Second)
	if !didComplete || !didDismiss {
		t.Errorf("didComplete = %v, didDismiss = %v, want both", didComplete, didDismiss)
	}
	controller.Stop()
	controller.Dispose()
}

func TestReceivesStatusCallbacksForForwardAndReverse(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(ms(100)))
	statuses := logStatuses(controller)
	values := logValues(controller)
	statuses.want(t, "initial")
	controller.Forward()
	statuses.want(t, "forward", forward)
	controller.Reverse()
	statuses.want(t, "reverse", forward, dismissed)
	controller.Reverse()
	statuses.want(t, "reverse again", forward, dismissed)
	statuses.clear()
	controller.Forward()
	statuses.want(t, "forward again", forward)
	controller.Forward()
	statuses.want(t, "forward twice", forward)
	controller.Reverse()
	statuses.clear()
	for _, s := range []int{10, 20, 30, 40} {
		tt.tick(time.Duration(s) * time.Second)
		statuses.want(t, "ticks")
	}
	if len(*values) != 0 {
		t.Errorf("value log = %v, want empty", *values)
	}
	controller.Stop()
	controller.Dispose()
}

func TestForwardAndReverseFromValues(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(ms(100)))
	statuses := logStatuses(controller)
	values := logValues(controller)
	controller.Reverse(0.2)
	statuses.want(t, "reverse from", reverse)
	approxSlice(t, "values", *values, []float64{0.2})
	if controller.Value() != 0.2 {
		t.Errorf("value = %v", controller.Value())
	}
	statuses.clear()
	*values = nil
	controller.Forward(0.0)
	statuses.want(t, "forward from", dismissed, forward)
	approxSlice(t, "values", *values, []float64{0})
	if controller.Value() != 0 {
		t.Errorf("value = %v", controller.Value())
	}
	controller.Dispose()
}

type tickValue struct {
	at   int
	want float64
}

func runTicks(t *testing.T, tt *tester, c *AnimationController, name string, ticks []tickValue) {
	t.Helper()
	for _, tv := range ticks {
		tt.tick(ms(tv.at))
		if tv.want >= 0 {
			moreOrLess(t, name, c.Value(), tv.want)
		}
	}
}

func TestForwardAndReverseWithDifferentDurations(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), ReverseDuration: foundation.Some(ms(50))})
	controller.Forward()
	runTicks(t, tt, controller, "forward", []tickValue{{10, -1}, {30, 0.2}, {60, 0.5}, {90, 0.8}, {120, 1.0}})
	controller.Stop()
	controller.Reverse()
	runTicks(t, tt, controller, "reverse", []tickValue{{210, -1}, {220, 0.8}, {230, 0.6}, {240, 0.4}, {260, 0.0}})
	controller.Stop()
	controller.Dispose()

	controller = NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(50)), ReverseDuration: foundation.Some(ms(100))})
	controller.Forward()
	runTicks(t, tt, controller, "forward", []tickValue{{10, -1}, {30, 0.4}, {60, 1.0}, {90, 1.0}})
	controller.Stop()
	controller.Reverse()
	runTicks(t, tt, controller, "reverse", []tickValue{{210, -1}, {220, 0.9}, {230, 0.8}, {240, 0.7}, {260, 0.5}, {310, 0.0}})
	controller.Stop()
	controller.Dispose()
}

func TestToggleWithDifferentDurations(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), ReverseDuration: foundation.Some(ms(50))})
	controller.Toggle()
	runTicks(t, tt, controller, "forward", []tickValue{{10, -1}, {30, 0.2}, {60, 0.5}, {90, 0.8}, {120, 1.0}})
	controller.Stop()
	controller.Toggle()
	runTicks(t, tt, controller, "reverse", []tickValue{{210, -1}, {220, 0.8}, {230, 0.6}, {240, 0.4}, {260, 0.0}})
	controller.Stop()
	controller.Dispose()

	controller = NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(50)), ReverseDuration: foundation.Some(ms(100))})
	controller.Toggle()
	runTicks(t, tt, controller, "forward", []tickValue{{10, -1}, {30, 0.4}, {60, 1.0}, {90, 1.0}})
	controller.Stop()
	controller.Toggle()
	runTicks(t, tt, controller, "reverse", []tickValue{{210, -1}, {220, 0.9}, {230, 0.8}, {240, 0.7}, {260, 0.5}, {310, 0.0}})
	controller.Stop()
	controller.Dispose()
}

func TestToggleActsCorrectlyBasedOnTheAnimationState(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), ReverseDuration: foundation.Some(ms(100))})
	controller.Forward()
	if controller.Status() != forward || !IsForwardOrCompleted[float64](controller) {
		t.Fatalf("status = %v", controller.Status())
	}
	tt.tick(ms(10))
	tt.tick(ms(60))
	moreOrLess(t, "value", controller.Value(), 0.5)
	controller.Toggle()
	tt.tick(ms(10))
	if controller.Status() != reverse || IsForwardOrCompleted[float64](controller) {
		t.Errorf("status = %v, want reverse", controller.Status())
	}
	tt.tick(ms(110))
	moreOrLess(t, "value", controller.Value(), 0)
	if controller.Status() != dismissed || IsForwardOrCompleted[float64](controller) {
		t.Errorf("status = %v, want dismissed", controller.Status())
	}
	controller.Dispose()
}

func TestForwardOnlyFromValue(t *testing.T) {
	controller := NewAnimationController(newTester(), dur(ms(100)))
	statuses := logStatuses(controller)
	values := logValues(controller)
	controller.Forward(0.2)
	statuses.want(t, "forward from", forward)
	approxSlice(t, "values", *values, []float64{0.2})
	if controller.Value() != 0.2 {
		t.Errorf("value = %v", controller.Value())
	}
	controller.Dispose()
}

func TestCanFlingToUpperAndLowerBounds(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(ms(100)))
	controller.Fling()
	tt.tick(1 * time.Second)
	tt.tick(2 * time.Second)
	if controller.Value() != 1.0 {
		t.Errorf("value = %v, want 1", controller.Value())
	}
	controller.Stop()

	large := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), LowerBound: -30, UpperBound: foundation.Some(45.0)})
	large.Fling()
	tt.tick(3 * time.Second)
	tt.tick(4 * time.Second)
	if large.Value() != 45 {
		t.Errorf("value = %v, want 45", large.Value())
	}
	large.Fling(FlingOpts{Velocity: foundation.Some(-1.0)})
	tt.tick(5 * time.Second)
	tt.tick(6 * time.Second)
	if large.Value() != -30 {
		t.Errorf("value = %v, want -30", large.Value())
	}
	large.Stop()
	controller.Dispose()
	large.Dispose()
}

func TestCustomSpringDescriptionCanBeApplied(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt)
	custom := NewAnimationController(tt)
	controller.Fling()
	custom.Fling(FlingOpts{SpringDescription: foundation.Some(physics.NewSpringDescriptionWithDampingRatio(0.01, 10.0, physics.SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(2.0)}))})
	tt.tick(0)
	tt.tick(ms(50))
	if !(custom.Value() < controller.Value()) {
		t.Errorf("custom spring value %v should be below the default %v", custom.Value(), controller.Value())
	}
	controller.Dispose()
	custom.Dispose()
}

func TestFlingWithAnUnderdampedSpringPanics(t *testing.T) {
	controller := NewAnimationController(newTester())
	wantPanic(t, "AnimationControllerFlingUnderdamped", func() {
		controller.Fling(FlingOpts{SpringDescription: foundation.Some(physics.NewSpringDescriptionWithDampingRatio(1, 500, physics.SpringDescriptionWithDampingRatioOpts{Ratio: foundation.Some(0.5)}))})
	})
}

func TestLastElapsedDurationControlTest(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(ms(100)))
	if controller.LastElapsedDuration().IsSome() {
		t.Error("no last elapsed duration before the animation runs")
	}
	controller.Forward()
	tt.tick(ms(20))
	tt.tick(ms(30))
	tt.tick(ms(40))
	if got := controller.LastElapsedDuration().OrElse(-1); got != ms(20) {
		t.Errorf("LastElapsedDuration = %v, want 20ms", got)
	}
	controller.Stop()
	controller.Dispose()
}

func TestAnimationControllerToStringControlTest(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(ms(100)))
	oneLine(t, "controller", controller)
	controller.Forward()
	tt.tick(ms(10))
	tt.tick(ms(20))
	oneLine(t, "forward", controller)
	tt.tick(ms(30))
	oneLine(t, "forward 30", controller)
	controller.Reverse()
	tt.tick(ms(40))
	tt.tick(ms(50))
	oneLine(t, "reverse", controller)
	controller.Stop()
	controller.Dispose()
}

func within(t *testing.T, name string, got, lo, hi float64) {
	t.Helper()
	if got < lo || got > hi {
		t.Errorf("%s = %v, want in [%v, %v]", name, got, lo, hi)
	}
}

func TestVelocityLinear(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(ms(1000)))
	controller.Forward()
	tt.tick(0)
	tt.tick(ms(500))
	within(t, "velocity", controller.Velocity(), 0.9, 1.1)

	controller.Forward()
	within(t, "velocity", controller.Velocity(), 0.4, 0.6)
	tt.tick(0)
	within(t, "velocity", controller.Velocity(), 0.4, 0.6)
	tt.tick(ms(5))
	within(t, "velocity", controller.Velocity(), 0.9, 1.1)

	controller.Forward(0.5)
	within(t, "velocity", controller.Velocity(), 0.4, 0.6)
	tt.tick(0)
	within(t, "velocity", controller.Velocity(), 0.4, 0.6)
	tt.tick(ms(5))
	within(t, "velocity", controller.Velocity(), 0.9, 1.1)

	controller.Forward(1.0)
	if controller.Velocity() != 0 {
		t.Errorf("velocity = %v, want 0", controller.Velocity())
	}
	tt.tick(0)
	tt.tick(ms(500))
	if controller.Velocity() != 0 {
		t.Errorf("velocity = %v, want 0", controller.Velocity())
	}

	controller.Forward()
	tt.tick(0)
	tt.tick(ms(1000))
	if controller.Velocity() != 0 {
		t.Errorf("velocity = %v, want 0", controller.Velocity())
	}
	controller.Stop()
	controller.Dispose()
}

func TestDisposedAnimationControllerToStringWorks(t *testing.T) {
	controller := NewAnimationController(newTester(), dur(ms(100)))
	controller.Dispose()
	oneLine(t, "disposed", controller)
	if !strings.Contains(controller.String(), "DISPOSED") {
		t.Errorf("String = %q, want DISPOSED in it", controller.String())
	}
}

func TestAnimationControllerErrorHandling(t *testing.T) {
	skipUnlessDebug(t)
	controller := NewAnimationController(newTester())
	for name, fn := range map[string]func(){
		"forward":   func() { controller.Forward() },
		"reverse":   func() { controller.Reverse() },
		"animateTo": func() { controller.AnimateTo(0.5) },
		"repeat":    func() { controller.Repeat() },
	} {
		if tag := panicTag(t, fn); tag != "AnimationControllerNoDuration" {
			t.Errorf("%s: panic tag = %q, want AnimationControllerNoDuration", name, tag)
		}
	}
	controller.Dispose()
	var got string
	tag := panicTag(t, func() {
		defer func() {
			if r := recover(); r != nil {
				got = r.(error).Error()
				panic(r)
			}
		}()
		controller.Dispose()
	})
	if tag != "AnimationControllerDisposedTwice" {
		t.Errorf("panic tag = %q, want AnimationControllerDisposedTwice", tag)
	}
	if !strings.Contains(got, "disposed") && !strings.Contains(got, "Dispose") {
		t.Errorf("message = %q", got)
	}
}

func TestAnimationControllerBoundsInvalid(t *testing.T) {
	wantPanic(t, "AnimationControllerBoundsInvalid", func() {
		NewAnimationController(newTester(), AnimationControllerOpts{LowerBound: 2, UpperBound: foundation.Some(1.0)})
	})
}

func TestRepeatThrowsIfPeriodIsNotSpecified(t *testing.T) {
	controller := NewAnimationController(newTester())
	wantPanic(t, "AnimationControllerNoDuration", func() { controller.Repeat() })
	wantPanic(t, "AnimationControllerNoDuration", func() { controller.Repeat() })
	controller.Dispose()
}

func TestDoNotAnimateIfAlreadyAtTarget(t *testing.T) {
	for name, value := range map[string]float64{"middle": 0.5, "upperBound": 1.0, "lowerBound": 0.0} {
		controller := NewAnimationController(newTester(), AnimationControllerOpts{Value: foundation.Some(value)})
		statuses := logStatuses(controller)
		controller.AnimateTo(value, AnimateOpts{Duration: foundation.Some(ms(100))})
		// A controller at the lower bound is already dismissed, so only the
		// others report the completed status of the animateTo.
		statuses.want(t, name, completed)
		if controller.Value() != value {
			t.Errorf("%s: value = %v, want %v", name, controller.Value(), value)
		}
		controller.Dispose()
	}
}

func TestDoNotAnimateIfAlreadyAtTargetMidFlight(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start float64
		run   func(*AnimationController)
		first AnimationStatus
	}{
		{"forward", 0, func(c *AnimationController) { c.Forward() }, forward},
		{"reverse", 1, func(c *AnimationController) { c.Reverse() }, reverse},
	} {
		tt := newTester()
		controller := NewAnimationController(tt, AnimationControllerOpts{Value: foundation.Some(tc.start), Duration: foundation.Some(ms(1000))})
		statuses := logStatuses(controller)
		tc.run(controller)
		tt.tick(0)
		tt.tick(ms(500))
		within(t, tc.name, controller.Value(), 0.4, 0.6)
		statuses.want(t, tc.name, tc.first)
		current := controller.Value()
		controller.AnimateTo(current, AnimateOpts{Duration: foundation.Some(ms(100))})
		statuses.want(t, tc.name, tc.first, completed)
		if controller.Value() != current {
			t.Errorf("%s: value = %v, want %v", tc.name, controller.Value(), current)
		}
		controller.Dispose()
	}
}

func TestAnimateToCanDealWithDurationZero(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(ms(100)))
	controller.Forward(0.2)
	if controller.Value() != 0.2 {
		t.Fatalf("value = %v", controller.Value())
	}
	controller.AnimateTo(1.0, AnimateOpts{Duration: foundation.Some(time.Duration(0))})
	if n := tt.TransientCallbackCount(); n != 0 {
		t.Errorf("transient callbacks = %d, want none (no animation)", n)
	}
	if controller.Value() != 1.0 {
		t.Errorf("value = %v, want 1", controller.Value())
	}
	controller.Dispose()
}

func TestResettingAnimationWorksAtAllPhases(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), Value: foundation.Some(0.0)})
	statuses := logStatuses(controller)
	if controller.Value() != 0 || controller.Status() != dismissed {
		t.Fatalf("value, status = %v, %v", controller.Value(), controller.Status())
	}
	controller.Reset()
	if controller.Value() != 0 || controller.Status() != dismissed {
		t.Fatalf("value, status = %v, %v", controller.Value(), controller.Status())
	}

	statuses.clear()
	controller.Forward()
	tt.tick(0)
	tt.tick(ms(50))
	if controller.Status() != forward {
		t.Errorf("status = %v", controller.Status())
	}
	controller.Reset()
	if controller.Value() != 0 || controller.Status() != dismissed {
		t.Errorf("value, status = %v, %v", controller.Value(), controller.Status())
	}
	statuses.want(t, "reset while forward", forward, dismissed)

	controller.SetValue(1)
	statuses.clear()
	controller.Reverse()
	tt.tick(0)
	tt.tick(ms(50))
	if controller.Status() != reverse {
		t.Errorf("status = %v", controller.Status())
	}
	controller.Reset()
	if controller.Value() != 0 || controller.Status() != dismissed {
		t.Errorf("value, status = %v, %v", controller.Value(), controller.Status())
	}
	statuses.want(t, "reset while reverse", reverse, dismissed)

	statuses.clear()
	controller.Forward()
	tt.tick(0)
	tt.tick(ms(150))
	if controller.Status() != completed {
		t.Errorf("status = %v", controller.Status())
	}
	controller.Reset()
	if controller.Value() != 0 || controller.Status() != dismissed {
		t.Errorf("value, status = %v, %v", controller.Value(), controller.Status())
	}
	statuses.want(t, "reset when completed", forward, completed, dismissed)
	controller.Dispose()
}

func TestSettingValueDirectlySetsCorrectStatus(t *testing.T) {
	controller := NewAnimationController(newTester(), AnimationControllerOpts{Value: foundation.Some(0.0)})
	for _, tc := range []struct {
		value  float64
		status AnimationStatus
	}{{0, dismissed}, {0.5, forward}, {1, completed}, {0.5, forward}, {0, dismissed}} {
		controller.SetValue(tc.value)
		if controller.Value() != tc.value || controller.Status() != tc.status {
			t.Errorf("SetValue(%v): value, status = %v, %v, want %v", tc.value, controller.Value(), controller.Status(), tc.status)
		}
	}
	controller.Dispose()
}

func TestAnimateToSetsCorrectStatus(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), Value: foundation.Some(0.0)})
	statuses := logStatuses(controller)
	for _, target := range []float64{0.5, 1.0, 0.5, 0.0} {
		controller.AnimateTo(target)
		tt.tick(0)
		tt.tick(ms(150))
		if controller.Value() != target {
			t.Errorf("value = %v, want %v", controller.Value(), target)
		}
		statuses.want(t, "animateTo", forward, completed)
		statuses.clear()
	}
	controller.Dispose()
}

func TestAfterAReverseCallAnimateToSetsCorrectStatus(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), Value: foundation.Some(1.0)})
	statuses := logStatuses(controller)
	if controller.Status() != completed {
		t.Fatalf("status = %v", controller.Status())
	}
	controller.Reverse()
	tt.tick(0)
	tt.tick(ms(150))
	if controller.Value() != 0 {
		t.Errorf("value = %v, want 0", controller.Value())
	}
	statuses.want(t, "reverse", reverse, dismissed)
	statuses.clear()
	controller.AnimateTo(0.5)
	tt.tick(0)
	tt.tick(ms(150))
	if controller.Value() != 0.5 {
		t.Errorf("value = %v, want 0.5", controller.Value())
	}
	statuses.want(t, "animateTo", forward, completed)
	controller.Dispose()
}

func TestAfterAForwardCallAnimateToSetsCorrectStatus(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), Value: foundation.Some(0.0)})
	statuses := logStatuses(controller)
	controller.Forward()
	tt.tick(0)
	tt.tick(ms(150))
	if controller.Value() != 1 {
		t.Errorf("value = %v, want 1", controller.Value())
	}
	statuses.want(t, "forward", forward, completed)
	statuses.clear()
	controller.AnimateTo(0.5)
	tt.tick(0)
	tt.tick(ms(150))
	if controller.Value() != 0.5 {
		t.Errorf("value = %v, want 0.5", controller.Value())
	}
	statuses.want(t, "animateTo", forward, completed)
	controller.Dispose()
}

func TestRepeatWithReverseAlternatesBetweenTheBounds(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), Value: foundation.Some(0.0)})
	controller.Repeat(RepeatOpts{Reverse: true})
	tt.tick(0)
	tt.tick(ms(25))
	moreOrLess(t, "value", controller.Value(), 0.25)
	tt.tick(0)
	tt.tick(ms(125))
	moreOrLess(t, "value", controller.Value(), 0.75)

	controller.Reset()
	controller.SetValue(1.0)
	controller.Repeat(RepeatOpts{Reverse: true})
	tt.tick(0)
	tt.tick(ms(25))
	moreOrLess(t, "value", controller.Value(), 0.75)
	tt.tick(0)
	tt.tick(ms(125))
	moreOrLess(t, "value", controller.Value(), 0.25)

	controller.Reset()
	controller.SetValue(0.5)
	controller.Repeat(RepeatOpts{Reverse: true})
	tt.tick(0)
	tt.tick(ms(50))
	moreOrLess(t, "value", controller.Value(), 1.0)
	tt.tick(0)
	tt.tick(ms(150))
	moreOrLess(t, "value", controller.Value(), 0.0)
	controller.Dispose()
}

func TestRepeatWithMinAndMaxBetween0And1(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), Value: foundation.Some(0.0)})
	controller.Repeat(RepeatOpts{Reverse: true, Min: foundation.Some(0.5), Max: foundation.Some(1.0)})
	tt.tick(0)
	tt.tick(ms(50))
	moreOrLess(t, "value", controller.Value(), 0.75)
	tt.tick(0)
	tt.tick(ms(100))
	moreOrLess(t, "value", controller.Value(), 1.0)
	tt.tick(0)
	tt.tick(ms(200))
	moreOrLess(t, "value", controller.Value(), 0.5)

	controller.Reset()
	controller.SetValue(0)
	controller.Repeat(RepeatOpts{Reverse: true, Min: foundation.Some(1.0), Max: foundation.Some(1.0)})
	tt.tick(0)
	tt.tick(ms(25))
	moreOrLess(t, "value", controller.Value(), 1.0)
	tt.tick(0)
	tt.tick(ms(125))
	moreOrLess(t, "value", controller.Value(), 1.0)

	controller.Reset()
	controller.SetValue(0.2)
	controller.Repeat(RepeatOpts{Reverse: true, Min: foundation.Some(0.2), Max: foundation.Some(0.6)})
	tt.tick(0)
	tt.tick(ms(50))
	moreOrLess(t, "value", controller.Value(), 0.4)
	controller.Dispose()
}

func TestRepeatWithNegativeMinAndPositiveMax(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), Value: foundation.Some(1.0), LowerBound: -1, UpperBound: foundation.Some(3.0)})
	controller.Repeat(RepeatOpts{Min: foundation.Some(1.0), Max: foundation.Some(3.0)})
	tt.tick(0)
	moreOrLess(t, "value", controller.Value(), 1)
	tt.tick(ms(50))
	moreOrLess(t, "value", controller.Value(), 2)

	controller.Reset()
	controller.SetValue(0)
	controller.Repeat(RepeatOpts{Min: foundation.Some(-1.0), Max: foundation.Some(3.0)})
	tt.tick(0)
	moreOrLess(t, "value", controller.Value(), 0)
	tt.tick(ms(25))
	moreOrLess(t, "value", controller.Value(), 1)
	controller.Dispose()
}

func TestAnimationBehaviorDefaultValues(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt)
	if controller.AnimationBehavior != AnimationBehaviorNormal {
		t.Errorf("behavior = %v, want normal", controller.AnimationBehavior)
	}
	repeating := NewAnimationControllerUnbounded(tt)
	if repeating.AnimationBehavior != AnimationBehaviorPreserve {
		t.Errorf("unbounded behavior = %v, want preserve", repeating.AnimationBehavior)
	}
	if !math.IsInf(repeating.LowerBound, -1) || !math.IsInf(repeating.UpperBound, 1) {
		t.Errorf("unbounded bounds = %v, %v", repeating.LowerBound, repeating.UpperBound)
	}
	controller.Dispose()
	repeating.Dispose()
}

func TestAnimationBehaviorPreserveRunsAtNormalSpeedWhenAnimatingTo(t *testing.T) {
	tt := newTester()
	tt.disableAnimations = true
	controller := NewAnimationController(envTester{tt}, AnimationControllerOpts{AnimationBehavior: AnimationBehaviorPreserve})
	controller.AnimateTo(1.0, AnimateOpts{Duration: foundation.Some(ms(100))})
	tt.tick(0)
	tt.tick(ms(50))
	if controller.Value() != 0.5 || controller.Status() != forward {
		t.Errorf("value, status = %v, %v", controller.Value(), controller.Status())
	}
	tt.tick(0)
	tt.tick(ms(150))
	if controller.Value() != 1 || controller.Status() != completed {
		t.Errorf("value, status = %v, %v", controller.Value(), controller.Status())
	}
	controller.Dispose()
}

func TestAnimationBehaviorNormalRuns20xFasterWhenAnimatingTo(t *testing.T) {
	tt := newTester()
	tt.disableAnimations = true
	controller := NewAnimationController(envTester{tt})
	controller.AnimateTo(1.0, AnimateOpts{Duration: foundation.Some(ms(100))})
	tt.tick(0)
	tt.tick(2500 * time.Microsecond)
	if controller.Value() != 0.5 || controller.Status() != forward {
		t.Errorf("value, status = %v, %v", controller.Value(), controller.Status())
	}
	tt.tick(0)
	tt.tick(5*time.Millisecond + 1000*time.Microsecond)
	if controller.Value() != 1 || controller.Status() != completed {
		t.Errorf("value, status = %v, %v", controller.Value(), controller.Status())
	}
	controller.Dispose()
}

func TestAnimationBehaviorNormalRunsFasterThanPreserve(t *testing.T) {
	tt := newTester()
	tt.disableAnimations = true
	controller := NewAnimationController(envTester{tt})
	fast := NewAnimationController(envTester{tt})
	controller.Fling(FlingOpts{AnimationBehavior: AnimationBehaviorPreserve})
	fast.Fling(FlingOpts{AnimationBehavior: AnimationBehaviorNormal})
	tt.tick(0)
	tt.tick(ms(50))
	if !(controller.Value() < fast.Value()) {
		t.Errorf("preserve value %v should be below normal value %v", controller.Value(), fast.Value())
	}
	controller.Dispose()
	fast.Dispose()
}

func TestAnimationControllerMethodsAssertTheTickerIsNotNull(t *testing.T) {
	skipUnlessDebug(t)
	controller := NewAnimationController(newTester())
	controller.Dispose()
	for name, fn := range map[string]func(){
		"animateBack": func() { controller.AnimateBack(0) },
		"animateTo":   func() { controller.AnimateTo(0) },
		"animateWith": func() { controller.AnimateWith(physics.NewGravitySimulation(0, 0, 0, 0)) },
		"stop":        func() { controller.Stop() },
		"forward":     func() { controller.Forward() },
		"reverse":     func() { controller.Reverse() },
	} {
		tag := panicTag(t, fn)
		// The calls without a duration panic about the duration first, as in
		// Flutter, where both are assertion errors.
		if tag != "AnimationControllerDisposed" && tag != "AnimationControllerNoDuration" {
			t.Errorf("%s: panic tag = %q", name, tag)
		}
	}
	d := NewAnimationController(newTester(), dur(ms(100)))
	d.Dispose()
	for name, fn := range map[string]func(){
		"animateTo": func() { d.AnimateTo(0) }, "stop": func() { d.Stop() }, "forward": func() { d.Forward() },
		"reverse": func() { d.Reverse() }, "animateWith": func() { d.AnimateWith(physics.NewGravitySimulation(0, 0, 0, 0)) },
	} {
		if tag := panicTag(t, fn); tag != "AnimationControllerDisposed" {
			t.Errorf("%s: panic tag = %q, want AnimationControllerDisposed", name, tag)
		}
	}
}

// testSimulation is x(t) = t, forever.
type testSimulation struct{ physics.SimulationBase }

func (testSimulation) X(time float64) float64  { return time }
func (testSimulation) Dx(time float64) float64 { return time }
func (testSimulation) IsDone(float64) bool     { return false }

func TestSimulationsRunForward(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(time.Second))
	statuses := logStatuses(controller)
	controller.AnimateWith(&testSimulation{})
	tt.tick(0)
	tt.tick(2 * time.Second)
	statuses.want(t, "animateWith", forward)
	controller.Dispose()
}

func TestSimulationsRunForwardEvenAfterAReverseRun(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(time.Second))
	statuses := logStatuses(controller)
	controller.Reverse(1.0)
	tt.tick(0)
	tt.tick(2 * time.Second)
	statuses.want(t, "reverse", completed, reverse, dismissed)
	statuses.clear()
	controller.AnimateWith(&testSimulation{})
	tt.tick(0)
	tt.tick(2 * time.Second)
	statuses.want(t, "animateWith", forward)
	controller.Dispose()
}

func TestAnimateBackWithRunsInReverse(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(time.Second))
	controller.SetValue(1)
	statuses := logStatuses(controller)
	controller.AnimateBackWith(physics.NewSpringSimulation(physics.NewSpringDescriptionWithDampingRatio(1, 500), 1, 0, 0))
	statuses.want(t, "animateBackWith", reverse)
	tt.tick(0)
	tt.tick(2 * time.Second)
	if math.Abs(controller.Value()) > 1e-9 || controller.Status() != dismissed {
		t.Errorf("value, status = %v, %v, want 0, dismissed", controller.Value(), controller.Status())
	}
	controller.Dispose()
}

func TestRepeatingAnimationWithReverseReportsForwardAndReverse(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(time.Second))
	statuses := logStatuses(controller)
	controller.Repeat(RepeatOpts{Reverse: true})
	tt.tick(0)
	tt.tick(ms(999))
	statuses.want(t, "first period", forward)
	statuses.clear()
	tt.tick(time.Second)
	statuses.want(t, "second period", reverse)
	controller.Dispose()
}

func TestAnimateBackRunsWithJustReverseDurationSet(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{ReverseDuration: foundation.Some(2 * time.Second)})
	statuses := logStatuses(controller)
	controller.AnimateBack(0.8)
	statuses.want(t, "animateBack", reverse)
	statuses.clear()
	tt.tick(0)
	tt.tick(2 * time.Second)
	statuses.want(t, "end", dismissed)
	controller.Dispose()
}

func TestRepeatCountMustBePositive(t *testing.T) {
	for _, count := range []int{0, -1} {
		controller := NewAnimationController(newTester(), AnimationControllerOpts{Duration: foundation.Some(ms(100)), Value: foundation.Some(0.0)})
		wantPanic(t, "AnimationControllerRepeatInvalid", func() { controller.Repeat(RepeatOpts{Reverse: true, Count: foundation.Some(count)}) })
		controller.Dispose()
	}
}

func TestRepeatWithCountAndNoReverse(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), Value: foundation.Some(0.0)})
	controller.Repeat(RepeatOpts{Count: foundation.Some(1)})
	tt.tick(0)
	for _, tv := range []tickValue{{25, 0.25}, {50, 0.5}, {99, 0.99}, {100, 0}} {
		tt.tick(ms(tv.at))
		moreOrLess(t, "count 1", controller.Value(), tv.want)
	}
	controller.Reset()
	controller.Repeat(RepeatOpts{Count: foundation.Some(2)})
	tt.tick(0)
	for _, tv := range []tickValue{{25, 0.25}, {50, 0.5}, {200, 0}} {
		tt.tick(ms(tv.at))
		moreOrLess(t, "count 2", controller.Value(), tv.want)
	}
	controller.Reset()
	controller.Dispose()
}

func TestRepeatWithCountAndReverse(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), Value: foundation.Some(0.0)})
	controller.Repeat(RepeatOpts{Reverse: true, Count: foundation.Some(4)})
	tt.tick(0)
	for _, tv := range []tickValue{{25, 0.25}, {50, 0.5}, {99, 0.99}, {100, 1}, {60, 0.6}} {
		tt.tick(ms(tv.at))
		near(t, "count 4 reverse", controller.Value(), tv.want, 0.05)
	}
	controller.Reset()
	controller.Dispose()
}

func TestAnimationControllerListenersAreClearedOnDispose(t *testing.T) {
	// Ported from animation_controller_listener_test.dart.
	controller := NewAnimationController(newTester(), dur(ms(100)))
	statusCalls, calls := 0, 0
	controller.AddListener(func() { calls++ })
	controller.AddStatusListener(func(AnimationStatus) { statusCalls++ })
	controller.NotifyListeners()
	controller.NotifyStatusListeners(completed)
	if statusCalls != 1 || calls != 1 {
		t.Fatalf("calls = %d, statusCalls = %d, want 1 and 1", calls, statusCalls)
	}
	controller.Dispose()
	controller.NotifyListeners()
	controller.NotifyStatusListeners(completed)
	if statusCalls != 1 || calls != 1 {
		t.Errorf("after dispose: calls = %d, statusCalls = %d, want 1 and 1", calls, statusCalls)
	}
}

func TestAnimationControllerReportsListenerPanics(t *testing.T) {
	tt := newTester()
	var reported []error
	tt.reporter = func(err error) { reported = append(reported, err) }
	controller := NewAnimationController(envTester{tt}, dur(ms(100)))
	called := false
	controller.AddListener(func() { panic("boom") })
	controller.AddListener(func() { called = true })
	controller.AddStatusListener(func(AnimationStatus) { panic("boom status") })
	controller.SetValue(0.5)
	if !called || len(reported) != 2 {
		t.Errorf("the second listener ran = %v, reported %d errors, want true and 2", called, len(reported))
	}
	controller.Dispose()
}

func TestAnimationControllerViewHidesTheController(t *testing.T) {
	controller := NewAnimationController(newTester())
	view := controller.View()
	controller.SetValue(0.25)
	if view.Value() != 0.25 {
		t.Errorf("view value = %v", view.Value())
	}
}
