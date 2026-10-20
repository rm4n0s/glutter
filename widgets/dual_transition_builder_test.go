// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/dual_transition_builder_test.dart.

func dualApp(controller *animation.AnimationController, child Widget) Widget {
	return NewDirectionality(ui.TextDirectionLTR, NewCenter(NewDualTransitionBuilder(controller,
		func(_ BuildContext, a animation.Animation[float64], child Widget) Widget {
			return NewScaleTransition(a, child)
		},
		func(_ BuildContext, a animation.Animation[float64], child Widget) Widget {
			return NewFadeTransition(animation.NewTween(1.0, 0.0).Animate(a), child)
		}, child)))
}

func dualScale(ts *gluttertest.Tester) float64 {
	return ts.Widget(gluttertest.ByType[*ScaleTransition]()).(*ScaleTransition).Scale().Value()
}

func dualOpacity(ts *gluttertest.Tester) float64 {
	return ts.Widget(gluttertest.ByType[*FadeTransition]()).(*FadeTransition).Opacity().Value()
}

func newDualController(ts *gluttertest.Tester, value float64) *animation.AnimationController {
	return newVsyncController(ts, animation.AnimationControllerOpts{Value: foundation.Some(value), Duration: foundation.Some(300 * time.Millisecond)})
}

type dualStep struct {
	pump           time.Duration
	scale, opacity float64
}

func checkDual(t *testing.T, ts *gluttertest.Tester, steps []dualStep) {
	t.Helper()
	for i, s := range steps {
		if s.pump > 0 {
			ts.Pump(s.pump)
		}
		if got := dualScale(ts); got != s.scale {
			t.Errorf("step %d: scale = %v, want %v", i, got, s.scale)
		}
		if got := dualOpacity(ts); got != s.opacity {
			t.Errorf("step %d: opacity = %v, want %v", i, got, s.opacity)
		}
	}
}

func TestDualTransitionBuilderRunsAnimations(t *testing.T) {
	ts := gluttertest.New(t)
	controller := newDualController(ts, 0)
	defer controller.Dispose()
	ts.PumpWidget(dualApp(controller, NewContainer(nil, ContainerOpts{Color: foundation.Some(ui.NewColor(0xFF00FF00))})))
	checkDual(t, ts, []dualStep{{0, 0, 1}})

	controller.Forward()
	ts.Pump()
	checkDual(t, ts, []dualStep{{150 * time.Millisecond, 0.5, 1}, {150 * time.Millisecond, 1, 1}})
	ts.PumpAndSettle()
	checkDual(t, ts, []dualStep{{0, 1, 1}})

	controller.Reverse()
	ts.Pump()
	checkDual(t, ts, []dualStep{{150 * time.Millisecond, 1, 0.5}, {150 * time.Millisecond, 1, 0}})
	ts.PumpAndSettle()
	checkDual(t, ts, []dualStep{{0, 0, 1}})
}

func TestDualTransitionBuilderKeepsState(t *testing.T) {
	ts := gluttertest.New(t)
	controller := newDualController(ts, 0)
	defer controller.Dispose()
	ts.PumpWidget(dualApp(controller, NewBuilder(func(BuildContext) Widget { return NewText("Foo") })))
	element := ts.Element(gluttertest.ByText("Foo"))
	same := func(step string) {
		t.Helper()
		if ts.Element(gluttertest.ByText("Foo")) != element {
			t.Errorf("%s: the child lost its element", step)
		}
	}
	controller.Forward()
	ts.Pump()
	same("forward start")
	ts.Pump(150 * time.Millisecond)
	same("forward half")
	ts.Pump(150 * time.Millisecond)
	same("forward end")
	ts.PumpAndSettle()
	same("forward settled")
	controller.Reverse()
	ts.Pump()
	same("reverse start")
	ts.Pump(150 * time.Millisecond)
	same("reverse half")
	ts.Pump(150 * time.Millisecond)
	same("reverse end")
	ts.PumpAndSettle()
	same("reverse settled")
}

func TestDualTransitionBuilderDoesNotJumpWhenInterrupted(t *testing.T) {
	t.Run("forward", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newDualController(ts, 0)
		defer controller.Dispose()
		ts.PumpWidget(dualApp(controller, NewContainer(nil)))
		checkDual(t, ts, []dualStep{{0, 0, 1}})
		controller.Forward()
		ts.Pump()
		checkDual(t, ts, []dualStep{{150 * time.Millisecond, 0.5, 1}})
		controller.Reverse()
		checkDual(t, ts, []dualStep{{0, 0.5, 1}})
		ts.Pump()
		checkDual(t, ts, []dualStep{{0, 0.5, 1}, {75 * time.Millisecond, 0.25, 1}, {75 * time.Millisecond, 0, 1}})
		ts.PumpAndSettle()
		checkDual(t, ts, []dualStep{{0, 0, 1}})
	})

	t.Run("reverse", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newDualController(ts, 1)
		defer controller.Dispose()
		ts.PumpWidget(dualApp(controller, NewContainer(nil)))
		checkDual(t, ts, []dualStep{{0, 1, 1}})
		controller.Reverse()
		ts.Pump()
		checkDual(t, ts, []dualStep{{150 * time.Millisecond, 1, 0.5}})
		controller.Forward()
		checkDual(t, ts, []dualStep{{0, 1, 0.5}})
		ts.Pump()
		checkDual(t, ts, []dualStep{{0, 1, 0.5}, {75 * time.Millisecond, 1, 0.75}, {75 * time.Millisecond, 1, 1}})
		ts.PumpAndSettle()
		checkDual(t, ts, []dualStep{{0, 1, 1}})
	})
}

func TestDualTransitionBuilderChangesAnimation(t *testing.T) {
	ts := gluttertest.New(t)
	first := newDualController(ts, 0)
	defer first.Dispose()
	second := newDualController(ts, 1)
	defer second.Dispose()
	ts.PumpWidget(dualApp(first, NewContainer(nil)))
	checkDual(t, ts, []dualStep{{0, 0, 1}})
	ts.PumpWidget(dualApp(second, NewContainer(nil)))
	checkDual(t, ts, []dualStep{{0, 1, 1}})
	// The old animation no longer drives the widget.
	first.Forward()
	ts.Pump()
	ts.Pump(150 * time.Millisecond)
	checkDual(t, ts, []dualStep{{0, 1, 1}})
	second.Reverse()
	ts.Pump()
	checkDual(t, ts, []dualStep{{150 * time.Millisecond, 1, 0.5}})
	first.Stop()
}
