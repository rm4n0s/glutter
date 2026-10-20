// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/animated_size_test.dart. The checks
// of the paint calls (pushClipRect or paintChild on a mock PaintingContext)
// are not ported: Go has no mock for the context, and the clip is the canvas
// clip of a layer that does not need compositing (SPEC section 12.10).

func animatedSizeOf(w, h float64, opts ...AnimatedSizeOpts) Widget {
	return NewCenter(NewAnimatedSize(200*time.Millisecond, sizedBox(w, h), opts...))
}

func sizeOfAnimatedSize(ts *gluttertest.Tester) ui.Size {
	return ts.Size(gluttertest.ByType[*AnimatedSize]())
}

func TestAnimatedSizeAnimatesForwardsThenBackwardsWithStableSizedChildren(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(animatedSizeOf(100, 100))
	if got := sizeOfAnimatedSize(ts); got != (ui.Size{Width: 100, Height: 100}) {
		t.Fatalf("size = %v", got)
	}
	ts.PumpWidget(animatedSizeOf(200, 200))
	for _, want := range []float64{150, 200} {
		ts.Pump(100 * time.Millisecond)
		if got := sizeOfAnimatedSize(ts); got != (ui.Size{Width: want, Height: want}) {
			t.Errorf("size = %v, want %v", got, want)
		}
	}
	ts.PumpWidget(animatedSizeOf(100, 100))
	for _, want := range []float64{150, 100} {
		ts.Pump(100 * time.Millisecond)
		if got := sizeOfAnimatedSize(ts); got != (ui.Size{Width: want, Height: want}) {
			t.Errorf("size = %v, want %v", got, want)
		}
	}
}

func TestAnimatedSizeCallsOnEndWhenTheAnimationIsCompleted(t *testing.T) {
	ts := gluttertest.New(t)
	calls := 0
	build := func(size float64) Widget {
		return NewCenter(NewAnimatedSize(200*time.Millisecond, sizedBox(size, size), AnimatedSizeOpts{OnEnd: func() { calls++ }}))
	}
	ts.PumpWidget(build(100))
	if calls != 0 {
		t.Fatalf("calls = %d", calls)
	}
	ts.PumpWidget(build(200))
	if calls != 0 {
		t.Fatalf("calls = %d", calls)
	}
	ts.PumpAndSettle()
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	ts.PumpWidget(build(100))
	ts.PumpAndSettle()
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestAnimatedSizeClampsAnimatedSizeToConstraints(t *testing.T) {
	ts := gluttertest.New(t)
	build := func(child float64) Widget {
		return NewCenter(NewSizedBoxSquare(foundation.Some(100.0), NewAnimatedSize(200*time.Millisecond, NewSizedBoxSquare(foundation.Some(child), nil))))
	}
	ts.PumpWidget(build(100))
	if got := sizeOfAnimatedSize(ts); got != (ui.Size{Width: 100, Height: 100}) {
		t.Fatalf("size = %v", got)
	}
	ts.PumpWidget(build(200))
	ts.Pump(100 * time.Millisecond)
	if got := sizeOfAnimatedSize(ts); got != (ui.Size{Width: 100, Height: 100}) {
		t.Errorf("size = %v, want 100x100", got)
	}
}

func TestAnimatedSizeTracksAnUnstableChildThenResumesWhenItStabilizes(t *testing.T) {
	ts := gluttertest.New(t)
	pumpMillis := func(ms int) { ts.Pump(time.Duration(ms) * time.Millisecond) }
	verify := func(size float64, state rendering.RenderAnimatedSizeState) {
		t.Helper()
		box := ts.RenderObject(gluttertest.ByType[*AnimatedSize]()).(*rendering.RenderAnimatedSize)
		if size >= 0 && box.Size() != (ui.Size{Width: size, Height: size}) {
			t.Errorf("size = %v, want %v", box.Size(), size)
		}
		if box.State() != state {
			t.Errorf("state = %v, want %v", box.State(), state)
		}
	}
	build := func(size float64, d time.Duration) Widget {
		return NewCenter(NewAnimatedSize(200*time.Millisecond, NewAnimatedContainer(d, nil, AnimatedContainerOpts{
			Width: foundation.Some(size), Height: foundation.Some(size)})))
	}
	ts.PumpWidget(build(100, 100*time.Millisecond))
	verify(100, rendering.RenderAnimatedSizeStateStable)
	ts.PumpWidget(build(200, 100*time.Millisecond))
	verify(-1, rendering.RenderAnimatedSizeStateStable)
	pumpMillis(1) // register change
	verify(-1, rendering.RenderAnimatedSizeStateChanged)
	pumpMillis(49)
	verify(150, rendering.RenderAnimatedSizeStateUnstable)
	pumpMillis(50)
	verify(200, rendering.RenderAnimatedSizeStateUnstable)
	pumpMillis(50)
	verify(200, rendering.RenderAnimatedSizeStateStable)
	ts.PumpWidget(build(100, time.Millisecond))
	verify(200, rendering.RenderAnimatedSizeStateStable)
	pumpMillis(1) // register change
	verify(-1, rendering.RenderAnimatedSizeStateChanged)
	pumpMillis(100)
	verify(150, rendering.RenderAnimatedSizeStateStable)
	pumpMillis(100)
	verify(100, rendering.RenderAnimatedSizeStateStable)
}

func TestAnimatedSizeResyncsItsAnimationController(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(animatedSizeOf(100, 100))
	ts.PumpWidget(animatedSizeOf(200, 100))
	ts.Pump(100 * time.Millisecond)
	if got := sizeOfAnimatedSize(ts); got.Width != 150 {
		t.Errorf("width = %v, want 150", got.Width)
	}
}

func TestAnimatedSizeDoesNotRunAnimationUnnecessarily(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(animatedSizeOf(100, 100))
	for i := 0; i < 20; i++ {
		box := ts.RenderObject(gluttertest.ByType[*AnimatedSize]()).(*rendering.RenderAnimatedSize)
		if box.Size() != (ui.Size{Width: 100, Height: 100}) || box.State() != rendering.RenderAnimatedSizeStateStable || box.IsAnimating() {
			t.Fatalf("pump %d: size %v, state %v, animating %v", i, box.Size(), box.State(), box.IsAnimating())
		}
		ts.Pump(10 * time.Millisecond)
	}
}

func TestAnimatedSizeCanSetAndUpdateClipBehavior(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(animatedSizeOf(100, 100))
	box := ts.RenderObject(gluttertest.ByType[*AnimatedSize]()).(*rendering.RenderAnimatedSize)
	if box.ClipBehavior() != ui.ClipHardEdge {
		t.Errorf("clipBehavior = %v, want hardEdge", box.ClipBehavior())
	}
	for _, clip := range []ui.Clip{ui.ClipNone, ui.ClipHardEdge, ui.ClipAntiAlias, ui.ClipAntiAliasWithSaveLayer} {
		ts.PumpWidget(animatedSizeOf(100, 100, AnimatedSizeOpts{ClipBehavior: clip}))
		if box.ClipBehavior() != clip {
			t.Errorf("clipBehavior = %v, want %v", box.ClipBehavior(), clip)
		}
	}
}

func TestAnimatedSizeWorksWrappedInIntrinsicHeightAndWrap(t *testing.T) {
	ts := gluttertest.New(t)
	pump := func(size ui.Size, d ...time.Duration) {
		w := NewCenter(NewIntrinsicHeight(NewWrap([]Widget{
			NewAnimatedSize(200*time.Millisecond, sizedBox(size.Width, size.Height), AnimatedSizeOpts{Curve: animation.Curves.EaseInOutBack}),
		}, WrapOpts{TextDirection: ui.TextDirectionLTR})))
		if len(d) > 0 {
			ts.PumpWidget(w, gluttertest.PumpWidgetOpts{Duration: d[0]})
		} else {
			ts.PumpWidget(w)
		}
	}
	sizeOf := func() ui.Size { return ts.Size(gluttertest.ByType[*IntrinsicHeight]()) }
	pump(ui.Size{Width: 100, Height: 100})
	if got := sizeOf(); got != (ui.Size{Width: 100, Height: 100}) {
		t.Fatalf("size = %v", got)
	}
	pump(ui.Size{Width: 150, Height: 200})
	if got := sizeOf(); got != (ui.Size{Width: 100, Height: 100}) {
		t.Fatalf("size = %v, want the old size", got)
	}
	for total := 0; total < 200; total += 10 {
		ts.Pump(10 * time.Millisecond)
	}
	if got := sizeOf(); got != (ui.Size{Width: 150, Height: 200}) {
		t.Fatalf("size = %v, want 150x200", got)
	}
	pump(ui.Size{Width: 100, Height: 100})
	if got := sizeOf(); got != (ui.Size{Width: 150, Height: 200}) {
		t.Fatalf("size = %v, want the old size", got)
	}
	pump(ui.Size{Width: 111, Height: 111}, 10*time.Millisecond)
	if got := sizeOf(); got != (ui.Size{Width: 111, Height: 111}) {
		t.Fatalf("size = %v, want 111x111", got)
	}
	pump(ui.Size{Width: 222, Height: 222}, 10*time.Millisecond)
	if got := sizeOf(); got != (ui.Size{Width: 222, Height: 222}) {
		t.Fatalf("size = %v, want 222x222", got)
	}
}

func TestAnimatedSizeReattachWithInterruptedAnimation(t *testing.T) {
	ts := gluttertest.New(t)
	key1, key2 := foundation.NewKey("key1"), foundation.NewKey("key2")
	var setState StateSetter
	childSize := 100.0
	animatedSize := NewCenter(
		NewSizedBoxFromSize(ui.Size{Width: 200, Height: 200}, NewCenter(
			NewAnimatedSize(time.Second, NewStatefulBuilder(func(_ BuildContext, s StateSetter) Widget {
				setState = s
				return NewSizedBoxSquare(foundation.Some(childSize), nil)
			})))),
		CenterOpts{Key: NewGlobalKey[State]()})
	h200 := foundation.Some(200.0)
	ts.PumpWidget(ltr(NewRow([]Widget{
		NewSizedBox(animatedSize, SizedBoxOpts{Key: key1, Height: h200}),
		NewSizedBox(nil, SizedBoxOpts{Key: key2, Height: h200}),
	})))
	setState(func() { childSize = 150 })
	ts.Pump()
	ts.PumpWidget(ltr(NewRow([]Widget{
		NewSizedBox(nil, SizedBoxOpts{Key: key1, Height: h200}),
		NewSizedBox(animatedSize, SizedBoxOpts{Key: key2, Height: h200}),
	})))
	if got := sizeOfAnimatedSize(ts); got != (ui.Size{Width: 100, Height: 100}) {
		t.Errorf("size = %v, want 100x100", got)
	}
	ts.PumpAndSettle()
	if got := sizeOfAnimatedSize(ts); got != (ui.Size{Width: 150, Height: 150}) {
		t.Errorf("size = %v, want 150x150", got)
	}
}

func TestAnimatedSizeDisposesAnimationAndController(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(animatedSizeOf(100, 100))
	box := ts.RenderObject(gluttertest.ByType[*AnimatedSize]()).(*rendering.RenderAnimatedSize)
	ts.PumpWidget(NewCenter(nil))
	if box.DebugAnimation() == nil || !box.DebugAnimation().IsDisposed {
		t.Error("the animation is not disposed")
	}
	if box.DebugController() == nil {
		t.Fatal("no controller")
	}
	err := panicOf(func() { box.DebugController().Dispose() })
	appErr, ok := err.(*errors.Error)
	if !ok || !appErr.HasRoute("AnimationControllerDisposedTwice") {
		t.Errorf("second dispose panicked with %v, want AnimationControllerDisposedTwice", err)
	}
}

func TestAnimatedSizeDoesNotCrashAtZeroSize(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxShrink(NewAnimatedSize(300*time.Millisecond, NewText("X"))))))
	if got := sizeOfAnimatedSize(ts); got != (ui.Size{}) {
		t.Errorf("size = %v", got)
	}
}
