// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/scrollable_test.dart. The cases that
// need CustomScrollView, SingleChildScrollView, ListView, PageView, nested
// scrollables of the sliver widgets or the semantics tester wait for step 9.3;
// the engine notification of pointer signals
// (onRespond) is not ported.

var allDeviceKinds = []ui.PointerDeviceKind{
	ui.PointerDeviceKindTouch, ui.PointerDeviceKindMouse, ui.PointerDeviceKindStylus,
	ui.PointerDeviceKindInvertedStylus, ui.PointerDeviceKindTrackpad, ui.PointerDeviceKindUnknown,
}

func usePlatform(t *testing.T, p foundation.TargetPlatform) {
	t.Helper()
	skipUnlessDebug(t) // the platform override is a debug-only feature
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(p))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}

var applePlatforms = []foundation.TargetPlatform{foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS}

// pumpTestOpts are the parameters of pumpTest.
type pumpTestOpts struct {
	notScrollable bool
	reverse       bool
	horizontal    bool
	controller    ScrollController
	noMouseDrag   bool
	// axisModifiers replaces the keys that flip the axis of the wheel.
	axisModifiers []services.LogicalKeyboardKey
}

// pumpTest pumps a scrollable with a sliver of 2000 pixels and no scrollbar, as
// the pumpTest of the Dart test does with a CustomScrollView.
func pumpTest(ts *gluttertest.Tester, o pumpTestOpts) {
	behavior := NewScrollBehavior().CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(false), PointerAxisModifiers: o.axisModifiers})
	if !o.noMouseDrag {
		behavior = behavior.CopyWith(ScrollBehaviorCopy{DragDevices: allDeviceKinds})
	}
	axis := painting.AxisDirectionDown
	if o.horizontal {
		axis = painting.AxisDirectionRight
	}
	if o.reverse {
		axis = painting.AxisDirectionUp
	}
	var physics ScrollPhysics
	if o.notScrollable {
		physics = NewNeverScrollableScrollPhysics(nil)
	}
	var box Widget = NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(2000.0)})
	if o.horizontal {
		box = NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(2000.0)})
	}
	ts.PumpWidget(ltr(NewScrollConfiguration(behavior, NewScrollable(func(_ BuildContext, position rendering.ViewportOffset) Widget {
		return NewViewport(position, []Widget{newSliverBox(box)}, ViewportOpts{AxisDirection: axis})
	}, ScrollableOpts{AxisDirection: axis, Controller: o.controller, Physics: physics}))))
}

const dragOffset = 200.0

func scrollOffset(ts *gluttertest.Tester) float64 {
	vp := ts.RenderObject(gluttertest.Last(gluttertest.ByType[*Viewport]())).(*rendering.RenderViewport)
	return vp.Offset().Pixels()
}

func scrollVelocity(ts *gluttertest.Tester) float64 {
	vp := ts.RenderObject(gluttertest.ByType[*Viewport]()).(*rendering.RenderViewport)
	return vp.Offset().(ScrollPosition).Activity().Velocity()
}

func TestScrollableHoldingScroll(t *testing.T) {
	for _, platform := range applePlatforms {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			pumpTest(ts, pumpTestOpts{})
			ts.Drag(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: 200}, gluttertest.GestureOpts{TouchSlopY: foundation.Some(0.0)})
			if got := scrollOffset(ts); got != -200 {
				t.Fatalf("offset = %v, want -200", got)
			}
			ts.Pump() // trigger ballistic
			ts.Pump(10 * time.Millisecond)
			if got := scrollOffset(ts); got <= -200 || got >= 0 {
				t.Fatalf("offset = %v, want between -200 and 0", got)
			}
			held := scrollOffset(ts)
			// Hold and let go while in overscroll.
			g := ts.StartGesture(ts.Center(gluttertest.ByType[*Scrollable]()))
			if frames := ts.PumpAndSettle(); frames != 1 {
				t.Errorf("settled in %d frames, want 1", frames)
			}
			if got := scrollOffset(ts); got != held {
				t.Errorf("offset = %v, want the held %v", got, held)
			}
			g.Up()
			// Once the hold is let go, it should still snap back to origin.
			// Flutter's test expects 3 frames: a spring-back takes one frame less in
			// Glutter (see the follow-ups of step 9.2 in docs/IMPLEMENTED.md).
			if frames := ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: time.Minute}); frames != 2 {
				t.Errorf("settled in %d frames, want 2", frames)
			}
			if got := scrollOffset(ts); got != 0 {
				t.Errorf("offset = %v, want 0", got)
			}
		})
	}
}

func fling(ts *gluttertest.Tester, dy, speed float64) {
	ts.Fling(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: dy}, speed)
}

func TestScrollableRepeatedFlingsBuildMomentum(t *testing.T) {
	for _, platform := range applePlatforms {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			pumpTest(ts, pumpTestOpts{})
			fling(ts, -dragOffset, 1000)
			ts.Pump() // trigger fling
			ts.Pump(10 * time.Millisecond)
			// Repeat the exact same motion.
			fling(ts, -dragOffset, 1000)
			ts.Pump()
			// On iOS, the velocity will be larger than the velocity of the last fling by a
			// non-trivial amount.
			if v := scrollVelocity(ts); v <= 1100 {
				t.Errorf("velocity = %v, want more than 1100", v)
			}
		})
	}
}

func TestScrollableRepeatedFlingsDoNotBuildMomentumOnAndroid(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformAndroid)
	ts := gluttertest.New(t)
	pumpTest(ts, pumpTestOpts{})
	fling(ts, -dragOffset, 1000)
	ts.Pump() // trigger fling
	ts.Pump(10 * time.Millisecond)
	// Repeat the exact same motion.
	fling(ts, -dragOffset, 1000)
	ts.Pump()
	// On Android, there is no momentum build. The final velocity is the same as the
	// velocity of the last fling.
	if v := scrollVelocity(ts); !moreOrLessEquals(v, 1000) {
		t.Errorf("velocity = %v, want 1000", v)
	}
}

func TestScrollableSlowerFinalFlingDoesNotApplyCarriedMomentum(t *testing.T) {
	for _, platform := range applePlatforms {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			pumpTest(ts, pumpTestOpts{})
			fling(ts, -dragOffset, 1000)
			ts.Pump() // trigger fling
			ts.Pump(10 * time.Millisecond)
			// Repeat the exact same motion to build momentum.
			fling(ts, -dragOffset, 1000)
			ts.Pump() // trigger the second fling
			ts.Pump(10 * time.Millisecond)
			// Make a final fling that is much slower.
			fling(ts, -dragOffset, 200)
			ts.Pump() // trigger the third fling
			ts.Pump(10 * time.Millisecond)
			// expect that there is no carried velocity
			if v := scrollVelocity(ts); v >= 200 {
				t.Errorf("velocity = %v, want less than 200", v)
			}
		})
	}
}

func TestScrollableNoMomentumBuildWithFlingsInOppositeDirections(t *testing.T) {
	for _, platform := range applePlatforms {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			pumpTest(ts, pumpTestOpts{})
			fling(ts, -dragOffset, 1000)
			ts.Pump() // trigger fling
			ts.Pump(10 * time.Millisecond)
			// Repeat the exact same motion in the opposite direction.
			fling(ts, dragOffset, 1000)
			ts.Pump()
			// The only applied velocity to the scrollable is the second fling that was in the
			// opposite direction.
			if v := scrollVelocity(ts); v != -1000 {
				t.Errorf("velocity = %v, want -1000", v)
			}
		})
	}
}

func TestScrollableNoMomentumKeptOnHoldGestures(t *testing.T) {
	for _, platform := range applePlatforms {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			pumpTest(ts, pumpTestOpts{})
			fling(ts, -dragOffset, 1000)
			ts.Pump() // trigger fling
			ts.Pump(10 * time.Millisecond)
			if v := scrollVelocity(ts); v <= 0 {
				t.Fatalf("velocity = %v, want more than 0", v)
			}
			g := ts.StartGesture(ts.Center(gluttertest.ByType[*Scrollable]()))
			ts.Pump(40 * time.Millisecond)
			g.Up()
			// After a hold longer than 2 frames, previous velocity is lost.
			if v := scrollVelocity(ts); v != 0 {
				t.Errorf("velocity = %v, want 0", v)
			}
		})
	}
}

// creep is the moves of the creeping tests: an offset and the time stamp in
// milliseconds, and the offset the scroll is expected at.
type creep struct {
	dy   float64
	ms   int
	want float64
}

func runCreep(t *testing.T, platform foundation.TargetPlatform, moves []creep) {
	t.Helper()
	usePlatform(t, platform)
	ts := gluttertest.New(t)
	pumpTest(ts, pumpTestOpts{})
	g := ts.StartGesture(ts.Center(gluttertest.ByType[*Scrollable]()))
	for i, m := range moves {
		g.MoveByAt(ui.Offset{Dy: m.dy}, time.Duration(m.ms)*time.Millisecond)
		if want := m.want; want >= 0 {
			if got := scrollOffset(ts); !moreOrLessEquals(got, want) {
				t.Errorf("move %d: offset = %v, want %v", i, got, want)
			}
		}
	}
}

func TestScrollableDragsCreepingUnaffectedOnAndroid(t *testing.T) {
	runCreep(t, foundation.TargetPlatformAndroid, []creep{{-0.5, 0, 0.5}, {-0.5, 10, 1.0}, {-0.5, 20, 1.5}})
}

func TestScrollableDragsCreepingMustBreakThresholdOnApple(t *testing.T) {
	for _, platform := range applePlatforms {
		t.Run(platform.String(), func(t *testing.T) {
			runCreep(t, platform, []creep{{-0.5, 0, 0}, {-0.5, 10, 0}, {-0.5, 20, 0}, {-1, 30, 0}, // Now -2.5 in total.
				{-1, 40, 0},      // Now -3.5, just reached threshold.
				{-0.5, 50, 0.5}}) // -0.5 over threshold transferred.
		})
	}
}

func TestScrollableBigDragOverThresholdMagnitudePreservedOnApple(t *testing.T) {
	for _, platform := range applePlatforms {
		t.Run(platform.String(), func(t *testing.T) {
			runCreep(t, platform, []creep{{-30, 0, 30}}) // No offset lost from threshold.
		})
	}
}

func TestScrollableSlowThresholdBreaksAreAttenuatedOnApple(t *testing.T) {
	for _, platform := range applePlatforms {
		t.Run(platform.String(), func(t *testing.T) {
			// This is a typical 'hesitant' iOS scroll start.
			runCreep(t, platform, []creep{{-10, 0, 1.1666666666666667},
				{-10, 20, 11.16666666666666673}}) // Subsequent motions unaffected.
		})
	}
}

func TestScrollableSmallContinuingMotionPreservedOnApple(t *testing.T) {
	for _, platform := range applePlatforms {
		t.Run(platform.String(), func(t *testing.T) {
			runCreep(t, platform, []creep{{-30, 0, 30}, // Break threshold.
				{-0.5, 20, 30.5}, {-0.5, 40, 31}, {-0.5, 60, 31.5}})
		})
	}
}

func TestScrollableMotionStopResetsThresholdOnApple(t *testing.T) {
	for _, platform := range applePlatforms {
		t.Run(platform.String(), func(t *testing.T) {
			runCreep(t, platform, []creep{{-30, 0, 30}, // Break threshold.
				{-0.5, 20, 30.5}, {0, 21, -1},
				{0, 120, -1}, // Stationary too long, threshold reset.
				{-1, 140, 30.5}, {-1, 150, 30.5}, {-1, 160, 30.5},
				{-1, 170, 31.5}, // New threshold broken.
				{-1, 180, 32.5}})
		})
	}
}

func mouseWheel(ts *gluttertest.Tester, dy float64) {
	ts.ScrollAt(ts.Center(gluttertest.ByType[*Viewport]()), ui.Offset{Dy: dy})
}

func TestScrollablePointerSignalsAreHandledOnFuchsia(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformFuchsia)
	ts := gluttertest.New(t)
	pumpTest(ts, pumpTestOpts{})
	mouseWheel(ts, 20)
	if got := scrollOffset(ts); got != 20 {
		t.Fatalf("offset = %v, want 20", got)
	}
	// Pointer signals should not cause overscroll.
	mouseWheel(ts, -30)
	if got := scrollOffset(ts); got != 0 {
		t.Errorf("offset = %v, want 0", got)
	}
}

func TestScrollablePointerSignalsAreIgnoredWhenScrollingIsDisabled(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformFuchsia)
	ts := gluttertest.New(t)
	pumpTest(ts, pumpTestOpts{notScrollable: true})
	mouseWheel(ts, 20)
	if got := scrollOffset(ts); got != 0 {
		t.Errorf("offset = %v, want 0", got)
	}
}

func TestScrollableScrollsInCorrectDirectionWhenAxisIsReversed(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformFuchsia)
	ts := gluttertest.New(t)
	pumpTest(ts, pumpTestOpts{reverse: true})
	mouseWheel(ts, -20)
	if got := scrollOffset(ts); got != 20 {
		t.Errorf("offset = %v, want 20", got)
	}
}

func TestScrollableHoldingScrollAndPointerSignalUpdateScrollDirection(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformFuchsia)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	pumpTest(ts, pumpTestOpts{controller: controller})
	last := rendering.ScrollDirectionIdle
	controller.AddListener(func() {
		if d := controller.Position().UserScrollDirection(); d != rendering.ScrollDirectionIdle {
			last = d
		}
	})
	slop := gluttertest.GestureOpts{TouchSlopY: foundation.Some(0.0)}
	ts.Drag(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: -20}, slop)
	if last != rendering.ScrollDirectionReverse {
		t.Fatalf("direction = %v, want reverse", last)
	}
	mouseWheel(ts, 20)
	if last != rendering.ScrollDirectionReverse {
		t.Fatalf("direction = %v, want reverse", last)
	}
	ts.Drag(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: 20}, slop)
	if last != rendering.ScrollDirectionForward {
		t.Fatalf("direction = %v, want forward", last)
	}
	mouseWheel(ts, -20)
	if last != rendering.ScrollDirectionForward {
		t.Errorf("direction = %v, want forward", last)
	}
}

func TestScrollableSetCanDragFalseWithActiveDragGesture(t *testing.T) {
	pump := func(ts *gluttertest.Tester, canDrag bool) {
		var physics ScrollPhysics = NewNeverScrollableScrollPhysics(nil)
		if canDrag {
			physics = NewAlwaysScrollableScrollPhysics(nil)
		}
		behavior := NewScrollBehavior().CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(false)})
		ts.PumpWidget(ltr(NewScrollConfiguration(behavior, NewScrollable(func(_ BuildContext, position rendering.ViewportOffset) Widget {
			return NewViewport(position, []Widget{newSliverBox(NewSizedBox(NewGestureDetector(nil, GestureDetectorOpts{OnTap: func() {}}),
				SizedBoxOpts{Height: foundation.Some(2000.0)}))}, ViewportOpts{})
		}, ScrollableOpts{Physics: physics}))))
	}
	ignoring := func(ts *gluttertest.Tester) *rendering.RenderIgnorePointer {
		return ts.RenderObject(gluttertest.Descendant(gluttertest.ByType[*Scrollable](), gluttertest.ByType[*IgnorePointer]())).(*rendering.RenderIgnorePointer)
	}
	t.Run("Hold does not disable user interaction", func(t *testing.T) {
		// Regression test for https://github.com/flutter/flutter/issues/66816.
		usePlatform(t, foundation.TargetPlatformAndroid)
		ts := gluttertest.New(t)
		pump(ts, true)
		r := ignoring(ts)
		if r.Ignoring() {
			t.Fatal("ignoring before the hold")
		}
		g := ts.StartGesture(ts.Center(gluttertest.ByType[*Viewport]()))
		// The lone drag recognizer wins by default (see StartGesture), so what
		// Flutter's test calls a hold is a drag whose activity has not moved.
		pump(ts, false)
		if r.Ignoring() {
			t.Error("ignoring after the physics changed")
		}
		g.Up()
		if r.Ignoring() {
			t.Error("ignoring after the up")
		}
	})
	t.Run("Drag disables user interaction when recognized", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformAndroid)
		ts := gluttertest.New(t)
		pump(ts, true)
		r := ignoring(ts)
		g := ts.StartGesture(ts.Center(gluttertest.ByType[*Viewport]()))
		g.MoveBy(ui.Offset{Dy: -100})
		// Starts ignoring when the drag is recognized.
		if !r.Ignoring() {
			t.Error("not ignoring after the drag was recognized")
		}
		pump(ts, false)
		if r.Ignoring() {
			t.Error("ignoring after the physics changed")
		}
		g.Up()
		if r.Ignoring() {
			t.Error("ignoring after the up")
		}
	})
	t.Run("Ballistic disables user interaction until it stops", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformAndroid)
		ts := gluttertest.New(t)
		pump(ts, true)
		r := ignoring(ts)
		// Starts ignoring when the drag is recognized.
		fling(ts, -100, 1000)
		if !r.Ignoring() {
			t.Error("not ignoring after the fling")
		}
		ts.Pump()
		// When the activity ends we should stop ignoring pointers.
		ts.PumpAndSettle()
		if r.Ignoring() {
			t.Error("still ignoring after the activity ended")
		}
	})
}

func TestScrollableTrackpadPanZoomScrolls(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformAndroid)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	pumpTest(ts, pumpTestOpts{controller: controller})
	g := ts.StartTrackpadGesture(ts.Center(gluttertest.ByType[*Scrollable]()))
	g.PanBy(ui.Offset{Dy: -100})
	if got := scrollOffset(ts); got <= 0 {
		t.Fatalf("offset = %v, want a scroll", got)
	}
	g.End()
	ts.PumpAndSettle()
	if got := scrollOffset(ts); got < 100 {
		t.Errorf("offset after the pan = %v, want at least 100", got)
	}
	if _, ok := controller.Position().Activity().(*IdleScrollActivity); !ok {
		t.Errorf("activity is %T after settling, want an idle one", controller.Position().Activity())
	}
}

func TestScrollableTrackpadFlingKeepsScrolling(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformAndroid)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	pumpTest(ts, pumpTestOpts{controller: controller})
	ts.TrackpadFling(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: -dragOffset}, 1000)
	ts.Pump()
	if _, ok := controller.Position().Activity().(*BallisticScrollActivity); !ok {
		t.Fatalf("activity is %T, want a ballistic one", controller.Position().Activity())
	}
	if v := scrollVelocity(ts); !moreOrLessEquals(v, 1000) {
		t.Errorf("velocity = %v, want 1000", v)
	}
	before := controller.Position().Pixels()
	ts.PumpAndSettle()
	if after := controller.Position().Pixels(); after <= before {
		t.Errorf("the trackpad fling did not carry on: %v -> %v", before, after)
	}
}

func TestScrollableSupportsUpdatingDragDevicesAtRuntime(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformAndroid)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	pumpTest(ts, pumpTestOpts{controller: controller, noMouseDrag: true})
	mouse := gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse}
	ts.Drag(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: -100}, mouse)
	if got := controller.Position().Pixels(); got != 0 {
		t.Fatalf("a mouse dragged before it was allowed: %v", got)
	}
	pumpTest(ts, pumpTestOpts{controller: controller})
	ts.Drag(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: -100}, mouse)
	if got := controller.Position().Pixels(); got <= 0 {
		t.Errorf("the mouse did not drag after it was allowed: %v", got)
	}
}

func TestScrollableHitTestBehaviorIsRespected(t *testing.T) {
	useAndroid(t) // no scrollbar, which has a gesture detector too
	for _, behavior := range []rendering.HitTestBehavior{rendering.HitTestBehaviorOpaque, rendering.HitTestBehaviorTranslucent, rendering.HitTestBehaviorDeferToChild} {
		ts := gluttertest.New(t)
		ts.PumpWidget(ltr(NewScrollable(func(_ BuildContext, position rendering.ViewportOffset) Widget {
			return NewViewport(position, nil, ViewportOpts{})
		}, ScrollableOpts{HitTestBehavior: behavior})))
		raw := ts.Widget(gluttertest.Descendant(gluttertest.ByType[*Scrollable](), gluttertest.ByType[*RawGestureDetector]())).(*RawGestureDetector)
		if raw.Behavior() != behavior {
			t.Errorf("behavior = %v, want %v", raw.Behavior(), behavior)
		}
	}
}

func TestScrollableHitTestBehaviorTranslucentLetsWidgetsUnderneathCatchTheHit(t *testing.T) {
	ts := gluttertest.New(t)
	key := foundation.NewUniqueKey()
	tapped := false
	ts.PumpWidget(ltr(NewStack([]Widget{
		NewPositionedFill(NewGestureDetector(NewSizedBox(nil, SizedBoxOpts{Key: key, Height: foundation.Some(300.0)}),
			GestureDetectorOpts{Behavior: rendering.HitTestBehaviorOpaque, OnTap: func() { tapped = true }})),
		// The test platform is Linux, whose scrollbar has an opaque mouse region;
		// the Dart test runs on Android, which has none.
		NewScrollConfiguration(NewScrollBehavior().CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(false)}),
			NewScrollable(func(_ BuildContext, position rendering.ViewportOffset) Widget {
				return NewViewport(position, nil, ViewportOpts{})
			}, ScrollableOpts{HitTestBehavior: rendering.HitTestBehaviorTranslucent})),
	})))
	ts.TapAt(ts.Center(gluttertest.ByKey(key)))
	if !tapped {
		t.Error("the widget underneath did not get the tap")
	}
}

// wheelAt is the horizontal and vertical offset of the scrollable after a wheel
// notch of dy pixels from a device of the kind.
func wheelFrom(ts *gluttertest.Tester, kind ui.PointerDeviceKind, dy float64) {
	ts.ScrollKindAt(kind, ts.Center(gluttertest.ByType[*Viewport]()), ui.Offset{Dy: dy})
}

// The four cases of scrollable_test.dart about pointerAxisModifiers.
func TestScrollablePointerAxisModifiers(t *testing.T) {
	tests := []struct {
		name      string
		kind      ui.PointerDeviceKind
		modifiers []services.LogicalKeyboardKey
		down      []services.LogicalKeyboardKey
		// flips is whether the vertical wheel scrolls the horizontal scrollable
		// while the keys are down.
		flips bool
	}{
		{name: "Scrolls horizontally when shift is pressed by default", kind: ui.PointerDeviceKindMouse,
			down: []services.LogicalKeyboardKey{services.LogicalKeyboardKeyShift}, flips: true},
		{name: "Scroll axis is not flipped for trackpad", kind: ui.PointerDeviceKindTrackpad,
			down: []services.LogicalKeyboardKey{services.LogicalKeyboardKeyShift}},
		{name: "Scrolls horizontally when custom key is pressed", kind: ui.PointerDeviceKindMouse,
			modifiers: []services.LogicalKeyboardKey{services.LogicalKeyboardKeyAltLeft},
			down:      []services.LogicalKeyboardKey{services.LogicalKeyboardKeyAltLeft}, flips: true},
		{name: "Still scrolls horizontally when other keys are pressed at the same time", kind: ui.PointerDeviceKindMouse,
			modifiers: []services.LogicalKeyboardKey{services.LogicalKeyboardKeyAltLeft},
			down:      []services.LogicalKeyboardKey{services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeySpace}, flips: true},
		{name: "Shift is not the modifier when a custom key is set", kind: ui.PointerDeviceKindMouse,
			modifiers: []services.LogicalKeyboardKey{services.LogicalKeyboardKeyAltLeft},
			down:      []services.LogicalKeyboardKey{services.LogicalKeyboardKeyShift}},
	}
	for _, tt := range tests {
		for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS} {
			t.Run(tt.name+"/"+platform.String(), func(t *testing.T) {
				usePlatform(t, platform)
				ts := gluttertest.New(t)
				pumpTest(ts, pumpTestOpts{horizontal: true, axisModifiers: tt.modifiers})
				wheelFrom(ts, tt.kind, 20)
				if got := scrollOffset(ts); got != 0 {
					t.Fatalf("vertical input was accepted: offset = %v", got)
				}
				for _, key := range tt.down {
					ts.SendKeyDownEvent(key)
				}
				wheelFrom(ts, tt.kind, 20)
				want := 0.0
				if tt.flips {
					want = 20
				}
				if got := scrollOffset(ts); got != want {
					t.Errorf("offset with the keys down = %v, want %v", got, want)
				}
				for _, key := range tt.down {
					ts.SendKeyUpEvent(key)
				}
				ts.Pump()
				wheelFrom(ts, tt.kind, 20)
				if got := scrollOffset(ts); got != want {
					t.Errorf("offset after the keys are up = %v, want %v", got, want)
				}
			})
		}
	}
}
