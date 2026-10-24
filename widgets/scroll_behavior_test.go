package widgets_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/scroll_behavior_test.dart. The
// scrolling cases scroll a Scrollable of one sliver instead of a ListView or
// SingleChildScrollView (step 9.3); the overscroll indicator cases (step 26.1) use
// a ListView and a SingleChildScrollView, as the Dart test does.

var desktopAndMobile = []foundation.TargetPlatform{
	foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformIOS,
	foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows,
}

// testScrollBehavior is TestScrollBehavior of the Dart test.
type testScrollBehavior struct {
	ScrollBehaviorBase
	flag bool
	last *gestures.GestureVelocityTrackerBuilder
}

func newTestScrollBehavior(flag bool, last *gestures.GestureVelocityTrackerBuilder) *testScrollBehavior {
	b := &testScrollBehavior{flag: flag, last: last}
	InitScrollBehaviorBase(&b.ScrollBehaviorBase, b)
	return b
}

func (b *testScrollBehavior) GetScrollPhysics(BuildContext) ScrollPhysics {
	if b.flag {
		return NewClampingScrollPhysics(nil)
	}
	return NewBouncingScrollPhysics()
}

func (b *testScrollBehavior) ShouldNotify(old ScrollBehavior) bool {
	return b.flag != old.(*testScrollBehavior).flag
}

func (b *testScrollBehavior) VelocityTrackerBuilder(bc BuildContext) gestures.GestureVelocityTrackerBuilder {
	var clock gestures.SamplingClock
	if binding := MaybeWidgetsBindingOf(bc); binding != nil {
		clock = binding.Gestures().Clock()
	}
	builder := func(ev gestures.PointerEvent) *gestures.VelocityTracker {
		return gestures.NewVelocityTrackerWithKind(clock, ev.Kind())
	}
	if !b.flag {
		builder = func(ev gestures.PointerEvent) *gestures.VelocityTracker {
			return gestures.NewIOSScrollViewFlingVelocityTracker(clock, ev.Kind())
		}
	}
	*b.last = builder
	return builder
}

func TestScrollBehaviorBuildScrollbarNeedsAControllerWhenUsingIt(t *testing.T) {
	skipUnlessDebug(t)
	for _, platform := range desktopAndMobile {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			var captured BuildContext
			ts.PumpWidget(NewScrollConfiguration(NewScrollBehavior().CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(false)}),
				NewBuilder(func(bc BuildContext) Widget {
					captured = bc
					return NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(1000.0)})
				})))
			details := NewScrollableDetails(painting.AxisDirectionDown)
			call := func() { NewScrollBehavior().BuildScrollbar(captured, NewContainer(nil), details) }
			switch platform {
			case foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows:
				requireTag(t, "ScrollBehaviorBase.BuildScrollbar.ScrollbarControllerMissing", call)
			default:
				call() // does not throw if we aren't using it
			}
		})
	}
}

func TestWrappedScrollBehaviorShouldNotify(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/89681
	b1, b2 := NewScrollBehavior().CopyWith(), NewScrollBehavior().CopyWith()
	if b1.ShouldNotify(b2) {
		t.Error("two copies of the default behavior should not notify")
	}
}

func TestInheritedScrollConfigurationChanged(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	var lastBuilder gestures.GestureVelocityTrackerBuilder
	var behavior *testScrollBehavior
	var position *ScrollPositionWithSingleContext
	scrollView := NewScrollable(func(_ BuildContext, p rendering.ViewportOffset) Widget {
		return NewViewport(p, []Widget{newSliverBox(NewBuilder(func(bc BuildContext) Widget {
			behavior = ScrollConfigurationOf(bc).(*testScrollBehavior)
			position = ScrollableOf(bc).Position().(*ScrollPositionWithSingleContext)
			return NewContainer(nil, ContainerOpts{Height: foundation.Some(1000.0)})
		}))}, ViewportOpts{})
	}, ScrollableOpts{Key: NewGlobalKey[*ScrollableState](GlobalKeyOpts{DebugLabel: "scrollable"})})

	ts.PumpWidget(ltr(NewScrollConfiguration(newTestScrollBehavior(true, &lastBuilder), scrollView)))
	if behavior == nil || !behavior.flag {
		t.Fatalf("behavior = %v", behavior)
	}
	if _, ok := position.Physics().(*ClampingScrollPhysics); !ok {
		t.Errorf("physics = %T, want clamping", position.Physics())
	}
	if lastBuilder == nil || lastBuilder(gestures.NewPointerDownEvent()) == nil {
		t.Error("no velocity tracker builder")
	}
	metrics := position.CopyWith()
	if metrics.ExtentAfter() != 400 || metrics.ViewportDimension() != 600 {
		t.Errorf("extentAfter = %v, viewportDimension = %v", metrics.ExtentAfter(), metrics.ViewportDimension())
	}

	// The same Scrollable, a different ScrollConfiguration.
	ts.PumpWidget(ltr(NewScrollConfiguration(newTestScrollBehavior(false, &lastBuilder), scrollView)))
	if behavior == nil || behavior.flag {
		t.Fatalf("behavior = %v", behavior)
	}
	if _, ok := position.Physics().(*BouncingScrollPhysics); !ok {
		t.Errorf("physics = %T, want bouncing", position.Physics())
	}
	// Regression test for https://github.com/flutter/flutter/issues/5856
	metrics = position.CopyWith()
	if metrics.ExtentAfter() != 400 || metrics.ViewportDimension() != 600 {
		t.Errorf("extentAfter = %v, viewportDimension = %v", metrics.ExtentAfter(), metrics.ViewportDimension())
	}
}

func multitouchApp(behavior ScrollBehavior, controller ScrollController, capture *BuildContext) Widget {
	return ltr(NewScrollConfiguration(behavior, NewBuilder(func(bc BuildContext) Widget {
		if capture != nil {
			*capture = bc
		}
		return testList([]Widget{NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(1000.0), Height: foundation.Some(1000.0)})},
			listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}})
	})))
}

func TestScrollBehaviorMultitouchDragStrategySumsAllPointers(t *testing.T) {
	for _, platform := range desktopAndMobile {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			controller := NewScrollController()
			defer controller.Dispose()
			behavior1 := NewScrollBehavior().CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(false)})
			behavior2 := behavior1.CopyWith(ScrollBehaviorCopy{MultitouchDragStrategy: gestures.MultitouchDragStrategySumAllPointers})
			ts.PumpWidget(multitouchApp(behavior1, controller, nil))
			if controller.Position().Pixels() != 0 {
				t.Fatalf("pixels = %v", controller.Position().Pixels())
			}
			location := ts.Center(gluttertest.ByType[*Scrollable]())
			g1 := ts.StartGesture(location, gluttertest.GestureOpts{Pointer: 1})
			ts.Pump()
			g2 := ts.StartGesture(location, gluttertest.GestureOpts{Pointer: 2})
			ts.Pump()
			g1.MoveBy(ui.Offset{Dy: -50})
			ts.Pump()
			g2.MoveBy(ui.Offset{Dy: -50})
			ts.Pump()
			// The default multitouchDragStrategy is latestPointer or averageBoundaryPointers:
			// the received delta should be 50.
			if got := controller.Position().Pixels(); got != 50 {
				t.Fatalf("pixels = %v, want 50", got)
			}
			ts.PumpWidget(multitouchApp(behavior2, controller, nil)) // changes to sumAllPointers
			g1.MoveBy(ui.Offset{Dy: -50})
			ts.Pump()
			g2.MoveBy(ui.Offset{Dy: -50})
			ts.Pump()
			// All active pointers are tracked.
			if got := controller.Position().Pixels(); got != 150 {
				t.Errorf("pixels = %v, want 150", got)
			}
			g1.Up()
			g2.Up()
		})
	}
}

func TestScrollBehaviorMultitouchDragStrategyOnNonApplePlatforms(t *testing.T) {
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformLinux,
		foundation.TargetPlatformFuchsia, foundation.TargetPlatformWindows} {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			controller := NewScrollController()
			defer controller.Dispose()
			var captured BuildContext
			behavior1 := NewScrollBehavior().CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(false)})
			behavior2 := behavior1.CopyWith(ScrollBehaviorCopy{MultitouchDragStrategy: gestures.MultitouchDragStrategyAverageBoundaryPointers})
			ts.PumpWidget(multitouchApp(behavior1, controller, &captured))
			location := ts.Center(gluttertest.ByType[*Scrollable]())
			g1 := ts.StartGesture(location, gluttertest.GestureOpts{Pointer: 1})
			ts.Pump()
			g2 := ts.StartGesture(location, gluttertest.GestureOpts{Pointer: 2})
			ts.Pump()
			g1.MoveBy(ui.Offset{Dy: -50})
			ts.Pump()
			g2.MoveBy(ui.Offset{Dy: -40})
			ts.Pump()
			// The default multitouchDragStrategy is latestPointer: only the latest active pointer is tracked.
			if got := ScrollConfigurationOf(captured).GetMultitouchDragStrategy(captured); got != gestures.MultitouchDragStrategyLatestPointer {
				t.Errorf("strategy = %v", got)
			}
			if got := controller.Position().Pixels(); got != 40 {
				t.Fatalf("pixels = %v, want 40", got)
			}
			ts.PumpWidget(multitouchApp(behavior2, controller, &captured)) // changes to averageBoundaryPointers
			g1.MoveBy(ui.Offset{Dy: -70})
			ts.Pump()
			g2.MoveBy(ui.Offset{Dy: -60})
			ts.Pump()
			if got := controller.Position().Pixels(); got != 40+70 {
				t.Errorf("pixels = %v, want 110", got)
			}
			g1.Up()
			g2.Up()
		})
	}
}

func TestScrollBehaviorMultitouchDragStrategyOnApplePlatforms(t *testing.T) {
	for _, platform := range applePlatforms {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			controller := NewScrollController()
			defer controller.Dispose()
			var captured BuildContext
			behavior1 := NewScrollBehavior().CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(false)})
			behavior2 := behavior1.CopyWith(ScrollBehaviorCopy{MultitouchDragStrategy: gestures.MultitouchDragStrategyLatestPointer})
			ts.PumpWidget(multitouchApp(behavior1, controller, &captured))
			location := ts.Center(gluttertest.ByType[*Scrollable]())
			g1 := ts.StartGesture(location, gluttertest.GestureOpts{Pointer: 1})
			ts.Pump()
			g2 := ts.StartGesture(location, gluttertest.GestureOpts{Pointer: 2})
			ts.Pump()
			g1.MoveBy(ui.Offset{Dy: -40})
			ts.Pump()
			g2.MoveBy(ui.Offset{Dy: -50})
			ts.Pump()
			// The default multitouchDragStrategy is averageBoundaryPointers.
			if got := ScrollConfigurationOf(captured).GetMultitouchDragStrategy(captured); got != gestures.MultitouchDragStrategyAverageBoundaryPointers {
				t.Errorf("strategy = %v", got)
			}
			if got := controller.Position().Pixels(); got != 50 {
				t.Fatalf("pixels = %v, want 50", got)
			}
			ts.PumpWidget(multitouchApp(behavior2, controller, &captured)) // changes to latestPointer
			g1.MoveBy(ui.Offset{Dy: -50})
			ts.Pump()
			g2.MoveBy(ui.Offset{Dy: -40})
			ts.Pump()
			if got := controller.Position().Pixels(); got != 50+40 {
				t.Errorf("pixels = %v, want 90", got)
			}
			g1.Up()
			g2.Up()
		})
	}
}

func TestScrollBehaviorConfigurationIsMaintainedOverMultipleCopies(t *testing.T) {
	t.Run("dragDevices", func(t *testing.T) {
		// Regression test for https://github.com/flutter/flutter/issues/91673
		def := NewScrollBehavior()
		want := []ui.PointerDeviceKind{ui.PointerDeviceKindTouch, ui.PointerDeviceKindStylus, ui.PointerDeviceKindInvertedStylus,
			ui.PointerDeviceKindTrackpad, ui.PointerDeviceKindUnknown}
		if got := def.DragDevices(); !sameSet(got, want) {
			t.Fatalf("default drag devices = %v, want %v", got, want)
		}
		once := def.CopyWith(ScrollBehaviorCopy{DragDevices: allDeviceKinds}) // use CopyWith to modify the drag devices
		if got := once.DragDevices(); !sameSet(got, allDeviceKinds) {
			t.Fatalf("once copied = %v", got)
		}
		twice := once.CopyWith() // copy again: the previously modified drag devices carry over
		if got := twice.DragDevices(); !sameSet(got, allDeviceKinds) {
			t.Errorf("twice copied = %v", got)
		}
	})
	t.Run("physics and platform", func(t *testing.T) {
		// Regression test for https://github.com/flutter/flutter/issues/91673
		useAndroid(t)
		ts := gluttertest.New(t)
		var defaultPhysics, oncePhysics, twicePhysics ScrollPhysics
		var defaultPlatform, oncePlatform, twicePlatform foundation.TargetPlatform
		ts.PumpWidget(NewScrollConfiguration(NewScrollBehavior(), NewBuilder(func(bc BuildContext) Widget {
			def := ScrollConfigurationOf(bc)
			defaultPhysics, defaultPlatform = def.GetScrollPhysics(bc), def.GetPlatform(bc)
			return NewScrollConfiguration(def.CopyWith(ScrollBehaviorCopy{Physics: NewBouncingScrollPhysics(), Platform: foundation.TargetPlatformFuchsia}),
				NewBuilder(func(bc BuildContext) Widget {
					once := ScrollConfigurationOf(bc)
					oncePhysics, oncePlatform = once.GetScrollPhysics(bc), once.GetPlatform(bc)
					return NewScrollConfiguration(once.CopyWith(), NewBuilder(func(bc BuildContext) Widget { // copy again, physics should follow
						twice := ScrollConfigurationOf(bc)
						twicePhysics, twicePlatform = twice.GetScrollPhysics(bc), twice.GetPlatform(bc)
						return NewSizedBox(nil)
					}))
				}))
		})))
		if want := NewClampingScrollPhysics(NewRangeMaintainingScrollPhysics(nil)); !reflect.DeepEqual(defaultPhysics, want) {
			t.Errorf("default physics = %v, want %v", defaultPhysics, want)
		}
		if oncePhysics.String() != NewBouncingScrollPhysics().String() || twicePhysics.String() != NewBouncingScrollPhysics().String() {
			t.Errorf("physics = %v and %v, want bouncing", oncePhysics, twicePhysics)
		}
		if defaultPlatform != foundation.TargetPlatformAndroid || oncePlatform != foundation.TargetPlatformFuchsia || twicePlatform != foundation.TargetPlatformFuchsia {
			t.Errorf("platforms = %v, %v and %v", defaultPlatform, oncePlatform, twicePlatform)
		}
	})
	t.Run("scrollbar", func(t *testing.T) {
		// Regression test for https://github.com/flutter/flutter/issues/91673
		for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows} {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			wrap := func(b ScrollBehavior) Widget {
				return ltr(NewMediaQuery(MediaQueryData{Size: ui.Size{Width: 500, Height: 500}},
					NewScrollConfiguration(b, testList([]Widget{NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(1000.0)})}))))
			}
			def := NewScrollBehavior()
			ts.PumpWidget(wrap(def))
			if ts.Count(gluttertest.ByType[*RawScrollbar]()) != 1 { // the default adds a scrollbar
				t.Fatalf("%v: no scrollbar by default", platform)
			}
			once := def.CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(false)})
			ts.PumpWidget(wrap(once))
			if ts.Count(gluttertest.ByType[*RawScrollbar]()) != 0 { // the copy does not add a scrollbar
				t.Fatalf("%v: the copy has a scrollbar", platform)
			}
			ts.PumpWidget(wrap(once.CopyWith()))
			if ts.Count(gluttertest.ByType[*RawScrollbar]()) != 0 { // the second copy maintains the setting
				t.Errorf("%v: the second copy has a scrollbar", platform)
			}
		}
	})
}

func TestScrollBehaviorDefaultAndroidOverscrollIndicator(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewScrollConfiguration(NewScrollBehavior(),
		NewListView([]Widget{NewSizedBox(NewText("Test"), SizedBoxOpts{Height: foundation.Some(1000.0), Width: foundation.Some(1000.0)})}))))

	if ts.Exists(gluttertest.ByType[*StretchingOverscrollIndicator]()) {
		t.Error("the default behavior built a StretchingOverscrollIndicator")
	}
	if got := ts.Count(gluttertest.ByType[*GlowingOverscrollIndicator]()); got != 1 {
		t.Errorf("the default behavior built %d GlowingOverscrollIndicators, want 1", got)
	}
}

func TestScrollBehaviorOverscrollIndicatorPerPlatform(t *testing.T) {
	for _, platform := range desktopAndMobile {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			ts.PumpWidget(ltr(NewScrollConfiguration(NewScrollBehavior(),
				NewListView([]Widget{NewSizedBox(NewText("Test"), SizedBoxOpts{Height: foundation.Some(1000.0), Width: foundation.Some(1000.0)})}))))
			want := platform == foundation.TargetPlatformAndroid || platform == foundation.TargetPlatformFuchsia
			if got := ts.Exists(gluttertest.ByType[*GlowingOverscrollIndicator]()); got != want {
				t.Errorf("glowing indicator is %v, want %v", got, want)
			}
			if ts.Exists(gluttertest.ByType[*StretchingOverscrollIndicator]()) {
				t.Error("the default behavior built a StretchingOverscrollIndicator")
			}
		})
	}
}

func TestScrollBehaviorOverscrollConfigurationIsMaintainedOverMultipleCopies(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/91673
	useAndroid(t)
	ts := gluttertest.New(t)
	wrap := func(b ScrollBehavior) Widget {
		return ltr(NewMediaQuery(NewMediaQueryData(MediaQueryDataOpts{Size: ui.Size{Width: 500, Height: 500}}),
			NewScrollConfiguration(b, NewBuilder(func(BuildContext) Widget {
				return NewSingleChildScrollView(NewContainer(nil, ContainerOpts{Height: foundation.Some(1000.0)}))
			}))))
	}
	def := NewScrollBehavior()
	ts.PumpWidget(wrap(def))
	// Default adds a glowing overscroll indicator
	if got := ts.Count(gluttertest.ByType[*GlowingOverscrollIndicator]()); got != 1 {
		t.Fatalf("the default has %d glowing indicators, want 1", got)
	}
	once := def.CopyWith(ScrollBehaviorCopy{Overscroll: foundation.Some(false)})
	ts.PumpWidget(wrap(once))
	// Copy does not add indicator
	if ts.Exists(gluttertest.ByType[*GlowingOverscrollIndicator]()) {
		t.Fatal("the copy has a glowing indicator")
	}
	twice := once.CopyWith()
	ts.PumpWidget(wrap(twice))
	// Second copy maintains overscroll setting
	if ts.Exists(gluttertest.ByType[*GlowingOverscrollIndicator]()) {
		t.Error("the second copy has a glowing indicator")
	}
}

func sameSet(a, b []ui.PointerDeviceKind) bool {
	if len(a) != len(b) {
		return false
	}
	for _, k := range a {
		if !slices.Contains(b, k) {
			return false
		}
	}
	return true
}
