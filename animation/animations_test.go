// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package animation

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

// Ported from packages/flutter/test/animation/animations_test.dart.

// bogusCurve maps every value to 100.
type bogusCurve struct{}

func (bogusCurve) Transform(float64) float64 { return 100 }
func (bogusCurve) Flipped() Curve            { return FlippedCurve{bogusCurve{}} }
func (bogusCurve) String() string            { return "bogusCurve" }

func TestAnimationsToString(t *testing.T) {
	tt := newTester()
	oneLine(t, "kAlwaysComplete", AlwaysComplete)
	oneLine(t, "kAlwaysDismissed", AlwaysDismissed)
	oneLine(t, "AlwaysStopped", NewAlwaysStoppedAnimation(0.5))
	curved := NewCurvedAnimation(AlwaysDismissed, Curves.Ease)
	oneLine(t, "curved", curved)
	curved.ReverseCurve = Curves.ElasticOut
	oneLine(t, "curved with reverse", curved)
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(500))})
	controller.SetValue(0.5)
	controller.Reverse()
	curved = NewCurvedAnimation(controller, Curves.Ease, Curves.ElasticOut)
	oneLine(t, "curved on a controller", curved)
	if !strings.Contains(curved.String(), "/") {
		t.Errorf("String = %q, want both curves", curved.String())
	}
	controller.Stop()
	controller.Dispose()
}

func TestProxyAnimationToString(t *testing.T) {
	animation := NewProxyAnimation(nil)
	if animation.Value() != 0 || animation.Status() != AnimationStatusDismissed {
		t.Errorf("value, status = %v, %v", animation.Value(), animation.Status())
	}
	oneLine(t, "proxy", animation)
	animation.SetParent(AlwaysDismissed)
	oneLine(t, "proxy with parent", animation)
}

func TestProxyAnimationSetParentGeneratesValueChanged(t *testing.T) {
	controller := NewAnimationController(newTester())
	controller.SetValue(0.5)
	called := false
	animation := NewProxyAnimation(nil)
	animation.AddListener(func() { called = true })
	if called {
		t.Fatal("the listener was called too early")
	}
	animation.SetParent(controller)
	if !called {
		t.Error("setting the parent should call the listener")
	}
	called = false
	controller.SetValue(0.6)
	if !called {
		t.Error("the listener should follow the new parent")
	}
}

func TestProxyAnimationMovesStatusListenersToTheNewParent(t *testing.T) {
	tt := newTester()
	a := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100))})
	b := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100))})
	b.SetValue(1)
	proxy := NewProxyAnimation(a)
	var log []AnimationStatus
	h := proxy.AddStatusListener(func(s AnimationStatus) { log = append(log, s) })
	proxy.SetParent(b)
	if len(log) != 1 || log[0] != AnimationStatusCompleted {
		t.Errorf("status log = %v, want [completed]", log)
	}
	b.SetValue(0)
	if len(log) != 2 || log[1] != AnimationStatusDismissed {
		t.Errorf("status log = %v, want [completed dismissed]", log)
	}
	proxy.RemoveStatusListener(h)
	b.SetValue(1)
	if len(log) != 2 {
		t.Errorf("a removed listener was called: %v", log)
	}
}

func TestReverseAnimationCallsListeners(t *testing.T) {
	controller := NewAnimationController(newTester())
	controller.SetValue(0.5)
	called := false
	listener := func() { called = true }
	animation := NewReverseAnimation(controller)
	h := animation.AddListener(listener)
	if called {
		t.Fatal("called too early")
	}
	controller.SetValue(0.6)
	if !called {
		t.Error("the listener was not called")
	}
	called = false
	animation.RemoveListener(h)
	controller.SetValue(0.7)
	if called {
		t.Error("a removed listener was called")
	}
	oneLine(t, "reverse", animation)
}

func TestReverseAnimationReversesTheStatus(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100))})
	reversed := NewReverseAnimation(controller)
	var log []AnimationStatus
	reversed.AddStatusListener(func(s AnimationStatus) { log = append(log, s) })
	controller.Forward()
	if len(log) != 1 || log[0] != AnimationStatusReverse || reversed.Status() != AnimationStatusReverse {
		t.Errorf("log = %v, status = %v, want reverse", log, reversed.Status())
	}
}

func TestTrainHoppingAnimation(t *testing.T) {
	tt := newTester()
	currentTrain := NewAnimationController(tt)
	defer currentTrain.Dispose()
	nextTrain := NewAnimationController(tt)
	defer nextTrain.Dispose()
	currentTrain.SetValue(0.5)
	nextTrain.SetValue(0.75)
	switched := false
	animation := NewTrainHoppingAnimation(currentTrain, nextTrain, func() { switched = true })
	if switched {
		t.Error("switched too early")
	}
	if animation.Value() != 0.5 {
		t.Errorf("value = %v, want 0.5", animation.Value())
	}
	oneLine(t, "train", animation)
	nextTrain.SetValue(0.25)
	if !switched {
		t.Error("the animation should have switched trains")
	}
	if animation.Value() != 0.25 {
		t.Errorf("value = %v, want 0.25", animation.Value())
	}
	oneLine(t, "train after", animation)
	if !strings.Contains(animation.String(), "no next") {
		t.Errorf("String = %q, want it to contain 'no next'", animation.String())
	}
}

func TestTrainHoppingAnimationStartingAtTheSameValueUsesTheSecond(t *testing.T) {
	a := NewAlwaysStoppedAnimation(1.0)
	b := NewAlwaysStoppedAnimation(1.0)
	switched := false
	train := NewTrainHoppingAnimation(a, b, func() { switched = true })
	if train.CurrentTrain() != b {
		t.Error("the second animation should be used from the start")
	}
	if switched {
		t.Error("OnSwitchedTrain must not be called")
	}
	train.Dispose()
}

func TestTrainHoppingAnimationNotifiesStatusListeners(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100))})
	defer controller.Dispose()
	animation := NewTrainHoppingAnimation(
		NewTween(1.0, -1.0).Animate(controller),
		NewTween(-1.0, 1.0).Animate(controller), nil)
	defer animation.Dispose()
	var log []AnimationStatus
	animation.AddStatusListener(func(s AnimationStatus) { log = append(log, s) })
	if len(log) != 0 {
		t.Fatalf("log = %v, want empty", log)
	}
	controller.Forward()
	if len(log) != 1 || log[0] != AnimationStatusForward {
		t.Errorf("log = %v, want [forward]", log)
	}
	log = nil
	controller.Reverse()
	if len(log) != 1 || log[0] != AnimationStatusDismissed {
		t.Errorf("log = %v, want [dismissed]", log)
	}
}

func compoundLog(t *testing.T, a Animation[float64]) (log *[]float64, listener func()) {
	l := &[]float64{}
	return l, func() { *l = append(*l, a.Value()) }
}

func TestAnimationMeanControlTest(t *testing.T) {
	tt := newTester()
	left := NewAnimationController(tt, AnimationControllerOpts{Value: foundation.Some(0.5)})
	right := NewAnimationController(tt)
	mean := NewAnimationMean(left, right)
	oneLine(t, "mean", mean)
	if mean.Value() != 0.25 {
		t.Errorf("value = %v, want 0.25", mean.Value())
	}
	log, logValue := compoundLog(t, mean)
	h := mean.AddListener(logValue)
	right.SetValue(1)
	if mean.Value() != 0.75 {
		t.Errorf("value = %v, want 0.75", mean.Value())
	}
	approxSlice(t, "log", *log, []float64{0.75})
	*log = nil
	mean.RemoveListener(h)
	left.SetValue(0)
	if mean.Value() != 0.5 || len(*log) != 0 {
		t.Errorf("value = %v, log = %v, want 0.5 and no calls", mean.Value(), *log)
	}
}

func TestAnimationMaxControlTest(t *testing.T) {
	tt := newTester()
	first := NewAnimationController(tt, AnimationControllerOpts{Value: foundation.Some(0.5)})
	second := NewAnimationController(tt)
	max := NewAnimationMax[float64](first, second)
	oneLine(t, "max", max)
	if max.Value() != 0.5 {
		t.Errorf("value = %v, want 0.5", max.Value())
	}
	log, logValue := compoundLog(t, max)
	h := max.AddListener(logValue)
	second.SetValue(1)
	approxSlice(t, "log", *log, []float64{1})
	*log = nil
	max.RemoveListener(h)
	first.SetValue(0)
	if max.Value() != 1 || len(*log) != 0 {
		t.Errorf("value = %v, log = %v", max.Value(), *log)
	}
}

func TestAnimationMinControlTest(t *testing.T) {
	tt := newTester()
	first := NewAnimationController(tt, AnimationControllerOpts{Value: foundation.Some(0.5)})
	second := NewAnimationController(tt)
	min := NewAnimationMin[float64](first, second)
	oneLine(t, "min", min)
	if min.Value() != 0 {
		t.Errorf("value = %v, want 0", min.Value())
	}
	log, logValue := compoundLog(t, min)
	h := min.AddListener(logValue)
	second.SetValue(1)
	if min.Value() != 0.5 {
		t.Errorf("value = %v, want 0.5", min.Value())
	}
	approxSlice(t, "log", *log, []float64{0.5})
	*log = nil
	min.RemoveListener(h)
	first.SetValue(0.25)
	if min.Value() != 0.25 || len(*log) != 0 {
		t.Errorf("value = %v, log = %v", min.Value(), *log)
	}
}

func TestCompoundAnimationStatus(t *testing.T) {
	tt := newTester()
	first := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100))})
	second := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100))})
	mean := NewAnimationMean(first, second)
	var log []AnimationStatus
	mean.AddStatusListener(func(s AnimationStatus) { log = append(log, s) })
	if mean.Status() != AnimationStatusDismissed {
		t.Errorf("status = %v, want dismissed", mean.Status())
	}
	second.Forward()
	if mean.Status() != AnimationStatusForward || len(log) != 1 || log[0] != AnimationStatusForward {
		t.Errorf("status = %v, log = %v, want forward", mean.Status(), log)
	}
}

func TestCurvedAnimationWithBogusCurve(t *testing.T) {
	skipUnlessDebug(t)
	controller := NewAnimationController(newTester())
	curved := NewCurvedAnimation(controller, bogusCurve{})
	if tag := panicTag(t, func() { curved.Value() }); tag != "CurveEndpointInvalid" {
		t.Errorf("panic tag = %q, want CurveEndpointInvalid", tag)
	}
	controller.SetValue(0.3)
	if got := curved.Value(); got != 100 {
		t.Errorf("a value inside the range goes through the curve: %v", got)
	}
}

func TestCurvedAnimationRunningWithDifferentForwardAndReverseDurations(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), ReverseDuration: foundation.Some(ms(50))})
	curved := NewCurvedAnimation(controller, Curves.Linear, Curves.Linear)
	controller.Forward()
	tt.tick(0)
	for i := 1; i <= 10; i++ {
		tt.tick(ms(10 * i))
		moreOrLess(t, "forward", curved.Value(), float64(i)/10)
	}
	controller.Reverse()
	tt.tick(ms(110))
	moreOrLess(t, "reverse start", curved.Value(), 1.0)
	for i, want := range []float64{0.8, 0.6, 0.4, 0.2, 0} {
		tt.tick(ms(120 + 10*i))
		moreOrLess(t, "reverse", curved.Value(), want)
	}
}

func TestCurvedAnimationStopsListeningToParentWhenDisposed(t *testing.T) {
	tt := newTester()
	forwardCurve := NewInterval(0, 0.5)
	reverseCurve := NewInterval(0.5, 1)
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), ReverseDuration: foundation.Some(ms(100))})
	curved := NewCurvedAnimation(controller, forwardCurve, reverseCurve)
	if forwardCurve.Transform(0.5) != 1.0 || reverseCurve.Transform(0.5) != 0.0 {
		t.Fatal("the intervals are not what the test expects")
	}
	controller.Forward(0.5)
	if controller.Status() != AnimationStatusForward || curved.Value() != 1.0 {
		t.Errorf("status = %v, value = %v", controller.Status(), curved.Value())
	}
	controller.SetValue(1)
	if controller.Status() != AnimationStatusCompleted {
		t.Errorf("status = %v, want completed", controller.Status())
	}
	controller.Reverse(0.5)
	if controller.Status() != AnimationStatusReverse || curved.Value() != 0.0 {
		t.Errorf("status = %v, value = %v", controller.Status(), curved.Value())
	}
	if curved.IsDisposed {
		t.Error("disposed too early")
	}
	curved.Dispose()
	if !curved.IsDisposed {
		t.Error("not disposed")
	}
	controller.SetValue(0)
	if controller.Status() != AnimationStatusDismissed {
		t.Errorf("status = %v, want dismissed", controller.Status())
	}
	controller.Forward(0.5)
	if controller.Status() != AnimationStatusForward || curved.Value() != 0.0 {
		t.Errorf("after dispose: status = %v, value = %v (the curve direction no longer follows the parent)", controller.Status(), curved.Value())
	}
}

func TestReverseAnimationRunningWithDifferentForwardAndReverseDurations(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, AnimationControllerOpts{Duration: foundation.Some(ms(100)), ReverseDuration: foundation.Some(ms(50))})
	reversed := NewReverseAnimation(NewCurvedAnimation(controller, Curves.Linear, Curves.Linear))
	controller.Forward()
	tt.tick(0)
	for i := 1; i <= 10; i++ {
		tt.tick(ms(10 * i))
		moreOrLess(t, "forward", reversed.Value(), 1-float64(i)/10)
	}
	controller.Reverse()
	tt.tick(ms(110))
	moreOrLess(t, "reverse start", reversed.Value(), 0)
	for i, want := range []float64{0.2, 0.4, 0.6, 0.8, 1.0} {
		tt.tick(ms(120 + 10*i))
		moreOrLess(t, "reverse", reversed.Value(), want)
	}
}

func TestTweenSequenceAnimations(t *testing.T) {
	controller := NewAnimationController(newTester())
	animation := NewTweenSequence([]TweenSequenceItem[float64]{
		{Tween: NewTween(5.0, 10.0), Weight: 4},
		{Tween: NewConstantTween(10.0), Weight: 2},
		{Tween: NewTween(10.0, 5.0), Weight: 4},
	}).Animate(controller)
	for _, tt := range []struct{ at, want float64 }{{0, 5}, {0.2, 7.5}, {0.4, 10}, {0.6, 10}, {0.8, 7.5}, {1, 5}} {
		controller.SetValue(tt.at)
		moreOrLess(t, "sequence", animation.Value(), tt.want)
	}
}

func TestTweenSequenceWithCurves(t *testing.T) {
	controller := NewAnimationController(newTester())
	animation := NewTweenSequence([]TweenSequenceItem[float64]{
		{Tween: NewTween(5.0, 10.0).Chain(NewCurveTween(NewInterval(0.5, 1))), Weight: 4},
		{Tween: NewConstantTween(10.0).Chain(NewCurveTween(Curves.Linear)), Weight: 2},
		{Tween: NewTween(10.0, 5.0).Chain(NewCurveTween(NewInterval(0, 0.5))), Weight: 4},
	}).Animate(controller)
	for _, tt := range []struct{ at, want float64 }{{0, 5}, {0.2, 5}, {0.4, 10}, {0.6, 10}, {0.8, 5}, {1, 5}} {
		controller.SetValue(tt.at)
		moreOrLess(t, "sequence", animation.Value(), tt.want)
	}
}

func TestTweenSequenceOneTween(t *testing.T) {
	controller := NewAnimationController(newTester())
	animation := NewTweenSequence([]TweenSequenceItem[float64]{{Tween: NewTween(5.0, 10.0), Weight: 1}}).Animate(controller)
	for _, tt := range []struct{ at, want float64 }{{0, 5}, {0.5, 7.5}, {1, 10}} {
		controller.SetValue(tt.at)
		moreOrLess(t, "sequence", animation.Value(), tt.want)
	}
}
