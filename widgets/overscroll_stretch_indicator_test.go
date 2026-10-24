package widgets_test

import (
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/overscroll_stretch_indicator_test.dart.
//
// StretchingOverscrollIndicator uses a different algorithm when shaders are
// available (ui.ImageFilter.isShaderFilterSupported), so the Dart tests are
// different depending on shader support. Glutter has no shader image filters
// (SPEC §2), so shaderSupported is false: the tests that Dart skips with `skip:
// shaderSupported` run, and there is no test that needs the shader (the Dart
// file has none). The goldens are made by Glutter's software renderer.

// stretchTestOpts are the optional parameters of buildTest of the Dart test.
type stretchTestOpts struct {
	axis          painting.Axis
	reverse       bool
	textDirection ui.TextDirection
	boxHeight     float64
	boxWidth      float64
	physics       ScrollPhysics
}

type stretchKeys [3]foundation.Key

func newStretchKeys() stretchKeys {
	return stretchKeys{NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State]()}
}

func buildStretchTest(keys stretchKeys, controller ScrollController, o stretchTestOpts) Widget {
	if o.axis == painting.AxisUnspecified {
		o.axis = painting.AxisVertical
	}
	if o.textDirection == ui.TextDirectionUnspecified {
		o.textDirection = ui.TextDirectionLTR
	}
	if o.boxHeight == 0 {
		o.boxHeight = 250.0
	}
	if o.boxWidth == 0 {
		o.boxWidth = 300.0
	}
	var axisDirection painting.AxisDirection
	switch o.axis {
	case painting.AxisHorizontal:
		switch {
		case o.textDirection == ui.TextDirectionRTL && o.reverse:
			axisDirection = painting.AxisDirectionRight
		case o.textDirection == ui.TextDirectionRTL:
			axisDirection = painting.AxisDirectionLeft
		case o.reverse:
			axisDirection = painting.AxisDirectionLeft
		default:
			axisDirection = painting.AxisDirectionRight
		}
	case painting.AxisVertical:
		axisDirection = painting.AxisDirectionDown
		if o.reverse {
			axisDirection = painting.AxisDirectionUp
		}
	}
	box := func(color uint32, key foundation.Key) Widget {
		return NewSliverToBoxAdapter(NewContainer(nil, ContainerOpts{Key: key, Color: foundation.Some(ui.NewColor(color)),
			Height: foundation.Some(o.boxHeight), Width: foundation.Some(o.boxWidth)}))
	}
	return NewDirectionality(o.textDirection,
		NewMediaQuery(NewMediaQueryData(MediaQueryDataOpts{Size: ui.Size{Width: 800, Height: 600}}),
			NewScrollConfiguration(noOverscroll(),
				NewStretchingOverscrollIndicator(axisDirection,
					NewCustomScrollView([]Widget{
						box(0xD0FF0000, keys[0]),
						box(0xFFFFFF00, keys[1]),
						box(0xFF6200EA, keys[2]),
					}, CustomScrollViewOpts{Physics: o.physics, Reverse: o.reverse, ScrollDirection: o.axis, Controller: controller})))))
}

// noOverscroll is `const ScrollBehavior().copyWith(overscroll: false)`.
func noOverscroll() ScrollBehavior {
	return NewScrollBehavior().CopyWith(ScrollBehaviorCopy{Overscroll: foundation.Some(false)})
}

func findStretchEffect(ts *gluttertest.Tester) *StretchEffect {
	return ts.Widget(gluttertest.ByType[*StretchEffect]()).(*StretchEffect)
}

// topLeftOf is `box.localToGlobal(Offset.zero)` of the box with the key.
func topLeftOf(ts *gluttertest.Tester, key foundation.Key) ui.Offset {
	return ts.RenderObject(gluttertest.ByKey(key)).(rendering.Box).LocalToGlobal(ui.Offset{}, nil)
}

func expectAt(t *testing.T, ts *gluttertest.Tester, keys stretchKeys, why string, want ...ui.Offset) {
	t.Helper()
	for i, w := range want {
		if got := topLeftOf(ts, keys[i]); got != w {
			t.Errorf("%s: box%d is at %v, want %v", why, i+1, got, w)
		}
	}
}

// index1Dy is where the center of the text "Index 1" is before the stretch. Flutter's
// tests draw text with the Ahem font, whose line is 14 pixels high, so the text is at
// 51.0 (10 + 14 + 10 + 10 + 7); Roboto at 14 pixels has a line of 16 pixels, so the
// text is 3 pixels lower. The locations of the stretched text are the ones of Flutter's
// scaled by the same factor.
const index1Dy = 54.0

func stretchEffectStrength(ts *gluttertest.Tester) float64 {
	return findStretchEffect(ts).StretchStrength()
}

func TestStretchOverscrollWillDoNothingWhenAxesDoNotMatch(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	box1, box2 := NewGlobalKey[State](), NewGlobalKey[State]()
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR,
		NewMediaQuery(NewMediaQueryData(MediaQueryDataOpts{Size: ui.Size{Width: 800, Height: 600}}),
			NewScrollConfiguration(noOverscroll(),
				NewStretchingOverscrollIndicator(painting.AxisDirectionRight,
					NewCustomScrollView([]Widget{
						NewSliverToBoxAdapter(NewContainer(nil, ContainerOpts{Key: box1, Color: foundation.Some(ui.NewColor(0xD0FF0000)), Height: foundation.Some(250.0)})),
						NewSliverToBoxAdapter(NewContainer(nil, ContainerOpts{Key: box2, Color: foundation.Some(ui.NewColor(0xFFFFFF00)),
							Height: foundation.Some(250.0), Width: foundation.Some(300.0)})),
					}, CustomScrollViewOpts{Controller: controller}))))))

	if ts.Count(gluttertest.ByType[*StretchingOverscrollIndicator]()) != 1 || ts.Exists(gluttertest.ByType[*GlowingOverscrollIndicator]()) {
		t.Fatal("expected the stretching indicator alone")
	}
	if controller.Offset() != 0.0 || topLeftOf(ts, box1) != (ui.Offset{}) || topLeftOf(ts, box2) != (ui.Offset{Dy: 250}) {
		t.Fatalf("the start: offset %v, box1 %v, box2 %v", controller.Offset(), topLeftOf(ts, box1), topLeftOf(ts, box2))
	}

	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll the start, no stretching occurs.
	gesture.MoveBy(ui.Offset{Dy: 200})
	ts.PumpAndSettle()
	if topLeftOf(ts, box1) != (ui.Offset{}) || topLeftOf(ts, box2).Dy != 250.0 {
		t.Errorf("stretched: box1 %v, box2 %v", topLeftOf(ts, box1), topLeftOf(ts, box2))
	}

	gesture.Up()
	ts.PumpAndSettle()

	// Overscroll released
	if topLeftOf(ts, box1) != (ui.Offset{}) || topLeftOf(ts, box2) != (ui.Offset{Dy: 250}) {
		t.Errorf("released: box1 %v, box2 %v", topLeftOf(ts, box1), topLeftOf(ts, box2))
	}
}

func TestStretchOverscrollVertically(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(buildStretchTest(keys, controller, stretchTestOpts{}))

	expectIndicators(t, ts)
	if controller.Offset() != 0.0 {
		t.Errorf("offset = %v", controller.Offset())
	}
	expectAt(t, ts, keys, "the start", ui.Offset{}, ui.Offset{Dy: 250}, ui.Offset{Dy: 500})
	ts.MatchesGolden("overscroll_stretch_vertical_start")

	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll the start
	gesture.MoveBy(ui.Offset{Dy: 200})
	ts.PumpAndSettle()
	if topLeftOf(ts, keys[0]) != (ui.Offset{}) || topLeftOf(ts, keys[1]).Dy <= 255.0 || topLeftOf(ts, keys[2]).Dy <= 510.0 {
		t.Errorf("overscrolled the start: %v %v %v", topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}
	ts.MatchesGolden("overscroll_stretch_vertical_start_stretched")

	gesture.Up()
	ts.PumpAndSettle()

	// Stretch released back to the start
	expectAt(t, ts, keys, "released at the start", ui.Offset{}, ui.Offset{Dy: 250}, ui.Offset{Dy: 500})

	// Jump to end of the list
	controller.JumpTo(controller.Position().MaxScrollExtent())
	ts.PumpAndSettle()
	if controller.Offset() != 150.0 {
		t.Errorf("offset = %v, want 150", controller.Offset())
	}
	expectAt(t, ts, keys, "the end", ui.Offset{Dy: -150}, ui.Offset{Dy: 100}, ui.Offset{Dy: 350})
	ts.MatchesGolden("overscroll_stretch_vertical_end")

	gesture = ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll the end
	gesture.MoveBy(ui.Offset{Dy: -200})
	ts.PumpAndSettle()
	if topLeftOf(ts, keys[0]).Dy >= -165 || topLeftOf(ts, keys[1]).Dy >= 90.0 || topLeftOf(ts, keys[2]).Dy >= 350.0 {
		t.Errorf("overscrolled the end: %v %v %v", topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}
	ts.MatchesGolden("overscroll_stretch_vertical_end_stretched")

	gesture.Up()
	ts.PumpAndSettle()

	// Stretch released back
	expectAt(t, ts, keys, "released at the end", ui.Offset{Dy: -150}, ui.Offset{Dy: 100}, ui.Offset{Dy: 350})
}

// expectIndicators is the `expect` pair that opens most of the Dart tests: the
// stretching indicator is there, the glowing one is not.
func expectIndicators(t *testing.T, ts *gluttertest.Tester) {
	t.Helper()
	if ts.Count(gluttertest.ByType[*StretchingOverscrollIndicator]()) != 1 {
		t.Error("expected one StretchingOverscrollIndicator")
	}
	if ts.Exists(gluttertest.ByType[*GlowingOverscrollIndicator]()) {
		t.Error("expected no GlowingOverscrollIndicator")
	}
}

func TestStretchOverscrollWorksInReverseVertical(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(buildStretchTest(keys, controller, stretchTestOpts{reverse: true}))

	expectIndicators(t, ts)
	if controller.Offset() != 0.0 {
		t.Errorf("offset = %v", controller.Offset())
	}
	expectAt(t, ts, keys, "the start", ui.Offset{Dy: 350}, ui.Offset{Dy: 100}, ui.Offset{Dy: -150})

	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll
	gesture.MoveBy(ui.Offset{Dy: -200})
	ts.PumpAndSettle()
	if topLeftOf(ts, keys[0]).Dy >= 350.0 || topLeftOf(ts, keys[1]).Dy >= 100.0 || topLeftOf(ts, keys[2]).Dy >= -150.0 {
		t.Errorf("overscrolled: %v %v %v", topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}
	ts.MatchesGolden("overscroll_stretch_vertical_reverse")
	gesture.Up()
	ts.PumpAndSettle()
}

func TestStretchOverscrollWorksInReverseHorizontal(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(buildStretchTest(keys, controller, stretchTestOpts{axis: painting.AxisHorizontal, reverse: true}))

	expectIndicators(t, ts)
	if controller.Offset() != 0.0 {
		t.Errorf("offset = %v", controller.Offset())
	}
	expectAt(t, ts, keys, "the start", ui.Offset{Dx: 500}, ui.Offset{Dx: 200}, ui.Offset{Dx: -100})

	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll
	gesture.MoveBy(ui.Offset{Dx: -200})
	ts.PumpAndSettle()
	if topLeftOf(ts, keys[0]).Dx >= 500.0 || topLeftOf(ts, keys[1]).Dx >= 200.0 || topLeftOf(ts, keys[2]).Dx >= -100.0 {
		t.Errorf("overscrolled: %v %v %v", topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}
	ts.MatchesGolden("overscroll_stretch_horizontal_reverse")
	gesture.Up()
	ts.PumpAndSettle()
}

func TestStretchOverscrollWorksInReverseHorizontalRTL(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(buildStretchTest(keys, controller, stretchTestOpts{axis: painting.AxisHorizontal, reverse: true, textDirection: ui.TextDirectionRTL}))

	expectIndicators(t, ts)
	if controller.Offset() != 0.0 {
		t.Errorf("offset = %v", controller.Offset())
	}
	expectAt(t, ts, keys, "the start", ui.Offset{}, ui.Offset{Dx: 300}, ui.Offset{Dx: 600})
	ts.MatchesGolden("overscroll_stretch_horizontal_reverse_rtl_start")

	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll the start
	gesture.MoveBy(ui.Offset{Dx: 200})
	ts.PumpAndSettle()

	if topLeftOf(ts, keys[0]) != (ui.Offset{}) || topLeftOf(ts, keys[1]).Dx <= 305.0 || topLeftOf(ts, keys[2]).Dx <= 610.0 {
		t.Errorf("overscrolled the start: %v %v %v", topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}
	ts.MatchesGolden("overscroll_stretch_horizontal_reverse_rtl_start_stretched")

	gesture.Up()
	ts.PumpAndSettle()

	// Stretch released back to the start
	expectAt(t, ts, keys, "released at the start", ui.Offset{}, ui.Offset{Dx: 300}, ui.Offset{Dx: 600})

	// Jump to end of the list
	controller.JumpTo(controller.Position().MaxScrollExtent())
	ts.PumpAndSettle()
	if controller.Offset() != 100.0 {
		t.Errorf("offset = %v, want 100", controller.Offset())
	}
	expectAt(t, ts, keys, "the end", ui.Offset{Dx: -100}, ui.Offset{Dx: 200}, ui.Offset{Dx: 500})
	ts.MatchesGolden("overscroll_stretch_horizontal_reverse_rtl_end")

	gesture = ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll the end
	gesture.MoveBy(ui.Offset{Dx: -200})
	ts.PumpAndSettle()
	if topLeftOf(ts, keys[0]).Dx >= -116.0 || topLeftOf(ts, keys[1]).Dx >= 190.0 || topLeftOf(ts, keys[2]).Dx >= 500.0 {
		t.Errorf("overscrolled the end: %v %v %v", topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}
	ts.MatchesGolden("overscroll_stretch_horizontal_reverse_rtl_end_stretched")

	gesture.Up()
	ts.PumpAndSettle()

	// Stretch released back
	expectAt(t, ts, keys, "released at the end", ui.Offset{Dx: -100}, ui.Offset{Dx: 200}, ui.Offset{Dx: 500})
}

func TestStretchOverscrollHorizontally(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(buildStretchTest(keys, controller, stretchTestOpts{axis: painting.AxisHorizontal}))

	expectIndicators(t, ts)
	if controller.Offset() != 0.0 {
		t.Errorf("offset = %v", controller.Offset())
	}
	expectAt(t, ts, keys, "the start", ui.Offset{}, ui.Offset{Dx: 300}, ui.Offset{Dx: 600})
	ts.MatchesGolden("overscroll_stretch_horizontal_start")

	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll the start
	gesture.MoveBy(ui.Offset{Dx: 200})
	ts.PumpAndSettle()
	if topLeftOf(ts, keys[0]) != (ui.Offset{}) || topLeftOf(ts, keys[1]).Dx <= 305.0 || topLeftOf(ts, keys[2]).Dx <= 610.0 {
		t.Errorf("overscrolled the start: %v %v %v", topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}
	ts.MatchesGolden("overscroll_stretch_horizontal_start_stretched")

	gesture.Up()
	ts.PumpAndSettle()

	// Stretch released back to the start
	expectAt(t, ts, keys, "released at the start", ui.Offset{}, ui.Offset{Dx: 300}, ui.Offset{Dx: 600})

	// Jump to end of the list
	controller.JumpTo(controller.Position().MaxScrollExtent())
	ts.PumpAndSettle()
	if controller.Offset() != 100.0 {
		t.Errorf("offset = %v, want 100", controller.Offset())
	}
	expectAt(t, ts, keys, "the end", ui.Offset{Dx: -100}, ui.Offset{Dx: 200}, ui.Offset{Dx: 500})
	ts.MatchesGolden("overscroll_stretch_horizontal_end")

	gesture = ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll the end
	gesture.MoveBy(ui.Offset{Dx: -200})
	ts.PumpAndSettle()
	if topLeftOf(ts, keys[0]).Dx >= -116.0 || topLeftOf(ts, keys[1]).Dx >= 190.0 || topLeftOf(ts, keys[2]).Dx >= 500.0 {
		t.Errorf("overscrolled the end: %v %v %v", topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}
	ts.MatchesGolden("overscroll_stretch_horizontal_end_stretched")

	gesture.Up()
	ts.PumpAndSettle()

	// Stretch released back
	expectAt(t, ts, keys, "released at the end", ui.Offset{Dx: -100}, ui.Offset{Dx: 200}, ui.Offset{Dx: 500})
}

func TestStretchOverscrollHorizontallyRTL(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(buildStretchTest(keys, controller, stretchTestOpts{axis: painting.AxisHorizontal, textDirection: ui.TextDirectionRTL}))

	expectIndicators(t, ts)
	if controller.Offset() != 0.0 {
		t.Errorf("offset = %v", controller.Offset())
	}
	expectAt(t, ts, keys, "the start", ui.Offset{Dx: 500}, ui.Offset{Dx: 200}, ui.Offset{Dx: -100})

	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll
	gesture.MoveBy(ui.Offset{Dx: -200})
	ts.PumpAndSettle()
	if topLeftOf(ts, keys[0]).Dx >= 500.0 || topLeftOf(ts, keys[1]).Dx >= 200.0 || topLeftOf(ts, keys[2]).Dx >= -100.0 {
		t.Errorf("overscrolled: %v %v %v", topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}
	ts.MatchesGolden("overscroll_stretch_horizontal_rtl")
	gesture.Up()
	ts.PumpAndSettle()
}

// disallowingApp is the app of the three tests that disallow the indicator: the
// notification listener counts the notifications and disallows the indicator
// when allow says so.
func disallowingApp(keys stretchKeys, controller ScrollController, count *int, disallow func() bool) Widget {
	return NewNotificationListener(func(notification *OverscrollIndicatorNotification) bool {
		if disallow() {
			notification.DisallowIndicator()
		}
		*count++
		return false
	}, buildStretchTest(keys, controller, stretchTestOpts{}))
}

func TestDisallowStretchingOverscroll(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	indicatorNotification := 0
	ts.PumpWidget(disallowingApp(keys, controller, &indicatorNotification, func() bool { return true }))

	expectIndicators(t, ts)
	if indicatorNotification != 0 || controller.Offset() != 0.0 {
		t.Errorf("notifications %d, offset %v", indicatorNotification, controller.Offset())
	}
	expectAt(t, ts, keys, "the start", ui.Offset{}, ui.Offset{Dy: 250}, ui.Offset{Dy: 500})

	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll the start, should not stretch
	gesture.MoveBy(ui.Offset{Dy: 200})
	ts.PumpAndSettle()
	if indicatorNotification != 1 {
		t.Errorf("notifications = %d, want 1", indicatorNotification)
	}
	expectAt(t, ts, keys, "overscrolled", ui.Offset{}, ui.Offset{Dy: 250}, ui.Offset{Dy: 500})

	gesture.Up()
	ts.PumpAndSettle()
}

func TestDisallowStretchingOverscrollDuringAFling(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	indicatorNotification := 0
	ts.PumpWidget(disallowingApp(keys, controller, &indicatorNotification, func() bool { return true }))

	expectIndicators(t, ts)
	if indicatorNotification != 0 {
		t.Errorf("notifications = %d", indicatorNotification)
	}

	controller.JumpTo(controller.Position().MaxScrollExtent())
	ts.Pump()

	drag := controller.Position().Drag(gestures.NewDragStartDetails(), func() {})
	drag.Update(gestures.DragUpdateDetails{GlobalPosition: ui.Offset{}, Delta: ui.Offset{Dy: -5000}, PrimaryDelta: foundation.Some(-5000.0)})
	drag.End(gestures.NewDragEndDetails(gestures.DragEndOpts{
		Velocity: gestures.Velocity{PixelsPerSecond: ui.Offset{Dy: -5000}}, PrimaryVelocity: foundation.Some(-5000.0)}))
	ts.Pump()
	ts.Pump(16 * time.Millisecond)

	if indicatorNotification != 1 {
		t.Errorf("notifications = %d, want 1", indicatorNotification)
	}
	if got := stretchEffectStrength(ts); got != 0.0 {
		t.Errorf("stretchStrength = %v, want 0", got)
	}
}

func TestDisallowedStretchDoesNotPersistIntoANewScrollSequence(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	disallowStretch := true
	indicatorNotification := 0
	ts.PumpWidget(disallowingApp(keys, controller, &indicatorNotification, func() bool { return disallowStretch }))

	expectIndicators(t, ts)
	if indicatorNotification != 0 {
		t.Errorf("notifications = %d", indicatorNotification)
	}

	disallowedGesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	disallowedGesture.MoveBy(ui.Offset{Dy: 200})
	ts.PumpAndSettle()
	disallowedGesture.Up()
	ts.PumpAndSettle()

	if indicatorNotification != 1 || stretchEffectStrength(ts) != 0.0 {
		t.Errorf("notifications %d, stretchStrength %v", indicatorNotification, stretchEffectStrength(ts))
	}

	disallowStretch = false
	controller.JumpTo(controller.Position().MaxScrollExtent() / 2.0)
	ts.Pump()

	ts.Fling(gluttertest.ByType[*CustomScrollView](), ui.Offset{Dy: -50}, 10000.0)
	ts.Pump(100 * time.Millisecond)
	ts.Pump(100 * time.Millisecond)
	ts.Pump(100 * time.Millisecond)

	if indicatorNotification != 2 {
		t.Errorf("notifications = %d, want 2", indicatorNotification)
	}
	if stretchEffectStrength(ts) == 0.0 {
		t.Error("stretchStrength is 0 after the new scroll sequence")
	}
}

// listWithRedBox is the app of the tests about the overflow and the clip of the
// stretch: a list of 300 pixels over a translucent red box.
func listWithRedBox(indicator func(list Widget) Widget) Widget {
	return NewDirectionality(ui.TextDirectionLTR,
		NewMediaQuery(NewMediaQueryData(MediaQueryDataOpts{Size: ui.Size{Width: 800, Height: 600}}),
			NewScrollConfiguration(noOverscroll(),
				NewDefaultTextStyle(testTextStyle(),
					NewColumn([]Widget{
						indicator(NewSizedBox(NewListViewBuilder(func(_ BuildContext, index int) Widget {
							return NewPadding(painting.NewEdgeInsetsAll(10.0), NewText("Index "+strconv.Itoa(index)))
						}, ListViewOpts{ItemCount: foundation.Some(20)}), SizedBoxOpts{Height: foundation.Some(300.0)})),
						NewOpacity(0.5, NewContainer(nil, ContainerOpts{Color: foundation.Some(ui.NewColor(0xD0FF0000)), Height: foundation.Some(100.0)})),
					})))))
}

// testTextStyle is the style of the text of the tests: the text of a test of
// gluttertest has no glyphs unless it has a font family of the bundled fonts, and
// Flutter's default is black.
func testTextStyle() *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto", FontSize: foundation.Some(14.0), Color: foundation.Some(ui.NewColor(0xFF000000))})
}

func TestStretchDoesNotOverflowBoundsOfContainer(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/90197
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(listWithRedBox(func(list Widget) Widget {
		return NewStretchingOverscrollIndicator(painting.AxisDirectionDown, list)
	}))

	index1 := gluttertest.ByText("Index 1")
	if !ts.Exists(index1) {
		t.Fatal("Index 1 is missing")
	}
	if got := ts.Center(index1).Dy; got != index1Dy {
		t.Fatalf("Index 1 is at %v, want %v", got, index1Dy)
	}

	gesture := ts.StartGesture(ts.Center(index1))
	// Overscroll the start.
	gesture.MoveBy(ui.Offset{Dy: 200})
	ts.PumpAndSettle()
	if !ts.Exists(index1) || ts.Center(index1).Dy <= 0 {
		t.Errorf("Index 1 is not stretched down: %v", ts.Center(index1))
	}
	// Image should not show the text overlapping the red area below the list.
	ts.MatchesGolden("overscroll_stretch_no_overflow")

	gesture.Up()
	ts.PumpAndSettle()
}

// clipRects are the RenderClipRects of the tree, in tree order.
func clipRects(ts *gluttertest.Tester) []*rendering.RenderClipRect {
	var out []*rendering.RenderClipRect
	for _, e := range ts.Elements(gluttertest.Everything()) {
		if r, ok := e.RenderObject().(*rendering.RenderClipRect); ok && (len(out) == 0 || out[len(out)-1] != r) {
			out = append(out, r)
		}
	}
	return out
}

func TestClipBehaviorIsUpdatedAsNeeded(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/97867
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(listWithRedBox(func(list Widget) Widget {
		return NewStretchingOverscrollIndicator(painting.AxisDirectionDown, list)
	}))

	index1 := gluttertest.ByText("Index 1")
	if !ts.Exists(index1) || ts.Center(index1).Dy != index1Dy {
		t.Fatalf("Index 1 is at %v", ts.Center(index1))
	}
	// Currently not clipping
	if got := clipRects(ts)[0].ClipBehavior(); got != ui.ClipNone {
		t.Errorf("clip behavior = %v, want Clip.none", got)
	}

	gesture := ts.StartGesture(ts.Center(index1))
	// Overscroll the start.
	gesture.MoveBy(ui.Offset{Dy: 200})
	ts.PumpAndSettle()
	if !ts.Exists(index1) || ts.Center(index1).Dy <= 0 {
		t.Errorf("Index 1 is at %v", ts.Center(index1))
	}
	// Now clipping
	if got := clipRects(ts)[0].ClipBehavior(); got != ui.ClipHardEdge {
		t.Errorf("clip behavior = %v, want Clip.hardEdge", got)
	}

	gesture.Up()
	ts.PumpAndSettle()
}

func TestClipBehaviorParameterUpdatesOverscrollClippingBehavior(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/103491
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(listWithRedBox(func(list Widget) Widget {
		return NewStretchingOverscrollIndicator(painting.AxisDirectionDown, list, StretchingOverscrollIndicatorOpts{ClipBehavior: ui.ClipNone})
	}))

	index1 := gluttertest.ByText("Index 1")
	if !ts.Exists(index1) || ts.Center(index1).Dy != index1Dy {
		t.Fatalf("Index 1 is at %v", ts.Center(index1))
	}
	// Currently not clipping
	if got := clipRects(ts)[0].ClipBehavior(); got != ui.ClipNone {
		t.Errorf("clip behavior = %v, want Clip.none", got)
	}

	gesture := ts.StartGesture(ts.Center(index1))
	// Overscroll the start.
	gesture.MoveBy(ui.Offset{Dy: 200})
	ts.PumpAndSettle()
	if !ts.Exists(index1) || ts.Center(index1).Dy <= 0 {
		t.Errorf("Index 1 is at %v", ts.Center(index1))
	}
	// Now clipping
	if got := clipRects(ts)[0].ClipBehavior(); got != ui.ClipNone {
		t.Errorf("clip behavior = %v, want Clip.none", got)
	}

	gesture.Up()
	ts.PumpAndSettle()
}

// shortList is the app of the tests about the stretch limit.
func shortList() Widget {
	return NewDirectionality(ui.TextDirectionLTR,
		NewMediaQuery(NewMediaQueryData(),
			NewScrollConfiguration(noOverscroll(),
				NewDefaultTextStyle(testTextStyle(),
					NewStretchingOverscrollIndicator(painting.AxisDirectionDown,
						NewSizedBox(NewListViewBuilder(func(_ BuildContext, index int) Widget {
							return NewPadding(painting.NewEdgeInsetsAll(10.0), NewText("Index "+strconv.Itoa(index)))
						}, ListViewOpts{ItemCount: foundation.Some(20)}), SizedBoxOpts{Height: foundation.Some(300.0)}))))))
}

func TestStretchLimit(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/99264
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(shortList())
	// The location of the text is Flutter's 52.63178407049861 for the text at 51.0
	// scaled to the text of gluttertest (see index1Dy). Flutter's Matrix4 has
	// 64-bit entries, Glutter's has 32-bit ones, so it is compared to a tolerance
	// of 1e-4.
	const maxStretchLocation = 52.63178407049861 / 51.0 * index1Dy

	index1 := gluttertest.ByText("Index 1")
	if !ts.Exists(index1) || ts.Center(index1).Dy != index1Dy {
		t.Fatalf("Index 1 is at %v", ts.Center(index1))
	}

	pointer := ts.StartGesture(ts.Center(index1))
	// Overscroll beyond the limit (the viewport is 600.0).
	pointer.MoveBy(ui.Offset{Dy: 610})
	ts.PumpAndSettle()
	if got := ts.Center(index1).Dy; !ts.Exists(index1) || math.Abs(got-maxStretchLocation) > 1e-4 {
		t.Errorf("Index 1 is at %v, want %v", got, maxStretchLocation)
	}

	pointer = ts.StartGesture(ts.Center(index1))
	// Overscroll way way beyond the limit
	pointer.MoveBy(ui.Offset{Dy: 1000})
	ts.PumpAndSettle()
	if got := ts.Center(index1).Dy; !ts.Exists(index1) || math.Abs(got-maxStretchLocation) > 1e-4 {
		t.Errorf("Index 1 is at %v, want %v", got, maxStretchLocation)
	}

	pointer.Up()
	ts.PumpAndSettle()
}

func TestMultiplePointersWillNotExceedStretchLimit(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/99264
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(shortList())
	index1 := gluttertest.ByText("Index 1")
	if !ts.Exists(index1) || ts.Center(index1).Dy != index1Dy {
		t.Fatalf("Index 1 is at %v", ts.Center(index1))
	}

	pointer1 := ts.StartGesture(ts.Center(index1))
	// Overscroll the start.
	pointer1.MoveBy(ui.Offset{Dy: 210})
	ts.PumpAndSettle()
	lastStretchedLocation := ts.Center(index1).Dy
	if !ts.Exists(index1) || lastStretchedLocation <= index1Dy {
		t.Fatalf("Index 1 is at %v", lastStretchedLocation)
	}

	pointer2 := ts.StartGesture(ts.Center(index1))
	// Add overscroll from an additional pointer
	pointer2.MoveBy(ui.Offset{Dy: 210})
	ts.PumpAndSettle()
	if !ts.Exists(index1) || ts.Center(index1).Dy <= lastStretchedLocation {
		t.Fatalf("Index 1 is at %v, after %v", ts.Center(index1).Dy, lastStretchedLocation)
	}
	lastStretchedLocation = ts.Center(index1).Dy

	pointer3 := ts.StartGesture(ts.Center(index1))
	// Add overscroll from an additional pointer, exceeding the max stretch (600)
	pointer3.MoveBy(ui.Offset{Dy: 210})
	ts.PumpAndSettle()
	if !ts.Exists(index1) || ts.Center(index1).Dy <= lastStretchedLocation {
		t.Fatalf("Index 1 is at %v, after %v", ts.Center(index1).Dy, lastStretchedLocation)
	}
	lastStretchedLocation = ts.Center(index1).Dy

	pointer4 := ts.StartGesture(ts.Center(index1))
	// Since we have maxed out the overscroll, it should not have stretched
	// further, regardless of the number of pointers.
	pointer4.MoveBy(ui.Offset{Dy: 210})
	ts.PumpAndSettle()
	if !ts.Exists(index1) || ts.Center(index1).Dy != lastStretchedLocation {
		t.Errorf("Index 1 is at %v, want %v", ts.Center(index1).Dy, lastStretchedLocation)
	}

	pointer1.Up()
	pointer2.Up()
	pointer3.Up()
	pointer4.Up()
	ts.PumpAndSettle()
}

func TestStretchOverscrollChangeDirectionMidScroll(t *testing.T) {
	tests := []struct {
		name string
		axis painting.Axis
		// the boxes fit in the viewport; the thresholds are the ones of the
		// Dart tests
		opts        stretchTestOpts
		start       [3]float64
		delta       func(d float64) ui.Offset
		overscroll  [3]float64 // greater than, after overscrolling the start
		back        [3]float64 // lower than, after overscrolling the end
		startThresh func(o ui.Offset) float64
	}{
		{
			name: "vertically", axis: painting.AxisVertical,
			opts:  stretchTestOpts{boxHeight: 100, physics: NewAlwaysScrollableScrollPhysics(nil)},
			start: [3]float64{0, 100, 200}, delta: func(d float64) ui.Offset { return ui.Offset{Dy: d} },
			overscroll: [3]float64{0, 103, 206}, back: [3]float64{-19, 85, 188},
			startThresh: func(o ui.Offset) float64 { return o.Dy },
		},
		{
			name: "horizontally", axis: painting.AxisHorizontal,
			opts:  stretchTestOpts{boxWidth: 100, physics: NewAlwaysScrollableScrollPhysics(nil), axis: painting.AxisHorizontal},
			start: [3]float64{0, 100, 200}, delta: func(d float64) ui.Offset { return ui.Offset{Dx: d} },
			overscroll: [3]float64{0, 102, 205}, back: [3]float64{-19, 85, 188},
			startThresh: func(o ui.Offset) float64 { return o.Dx },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useAndroid(t)
			ts := gluttertest.New(t)
			keys := newStretchKeys()
			controller := NewScrollController()
			defer controller.Dispose()

			ts.PumpWidget(buildStretchTest(keys, controller, tt.opts))

			expectIndicators(t, ts)
			if controller.Offset() != 0.0 {
				t.Errorf("offset = %v", controller.Offset())
			}
			original := [3]ui.Offset{tt.delta(tt.start[0]), tt.delta(tt.start[1]), tt.delta(tt.start[2])}
			expectAt(t, ts, keys, "the start", original[0], original[1], original[2])

			gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
			// Overscroll the start
			gesture.MoveBy(tt.delta(600))
			ts.PumpAndSettle()

			// The boxes should now be at different locations because of the scaling.
			check := func(why string) {
				t.Helper()
				if topLeftOf(ts, keys[0]) != (ui.Offset{}) || tt.startThresh(topLeftOf(ts, keys[1])) <= tt.overscroll[1] || tt.startThresh(topLeftOf(ts, keys[2])) <= tt.overscroll[2] {
					t.Errorf("%s: %v %v %v", why, topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
				}
			}
			check("overscrolled the start")

			// Move the pointer up a miniscule amount to trigger a directional change.
			gesture.MoveBy(tt.delta(-20))
			ts.PumpAndSettle()

			// The boxes should remain roughly at the same locations, since the pointer
			// didn't move far.
			check("after the directional change")

			// Now make the pointer overscroll to the end
			gesture.MoveBy(tt.delta(-1200))
			ts.PumpAndSettle()

			if tt.startThresh(topLeftOf(ts, keys[0])) >= tt.back[0] || tt.startThresh(topLeftOf(ts, keys[1])) >= tt.back[1] || tt.startThresh(topLeftOf(ts, keys[2])) >= tt.back[2] {
				t.Errorf("overscrolled the end: %v %v %v", topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
			}

			// Release the pointer
			gesture.Up()
			ts.PumpAndSettle()

			// Now the boxes should be back to their original locations.
			expectAt(t, ts, keys, "released", original[0], original[1], original[2])
		})
	}
}

func TestFlingTowardTheTrailingEdgeCausesStretchTowardTheLeadingEdge(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(buildStretchTest(keys, controller, stretchTestOpts{}))

	expectIndicators(t, ts)
	if controller.Offset() != 0.0 {
		t.Errorf("offset = %v", controller.Offset())
	}
	expectAt(t, ts, keys, "the start", ui.Offset{}, ui.Offset{Dy: 250}, ui.Offset{Dy: 500})

	ts.Fling(gluttertest.ByType[*CustomScrollView](), ui.Offset{Dy: -50}, 10000.0)
	ts.Pump(100 * time.Millisecond)
	ts.Pump(100 * time.Millisecond)
	ts.Pump(100 * time.Millisecond)

	// The boxes should now be at different locations because of the scaling.
	if controller.Offset() != 150.0 || topLeftOf(ts, keys[0]).Dy >= -160.0 || topLeftOf(ts, keys[1]).Dy >= 93.0 || topLeftOf(ts, keys[2]).Dy >= 347.0 {
		t.Errorf("offset %v, boxes %v %v %v", controller.Offset(), topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}

	ts.PumpAndSettle()

	// The boxes should now be at their final position.
	if controller.Offset() != 150.0 {
		t.Errorf("offset = %v, want 150", controller.Offset())
	}
	expectAt(t, ts, keys, "the final position", ui.Offset{Dy: -150}, ui.Offset{Dy: 100}, ui.Offset{Dy: 350})
}

func TestFlingTowardTheLeadingEdgeCausesStretchTowardTheTrailingEdge(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(buildStretchTest(keys, controller, stretchTestOpts{}))

	expectIndicators(t, ts)
	if controller.Offset() != 0.0 {
		t.Errorf("offset = %v", controller.Offset())
	}
	expectAt(t, ts, keys, "the start", ui.Offset{}, ui.Offset{Dy: 250}, ui.Offset{Dy: 500})

	// We fling to the trailing edge and let it settle.
	ts.Fling(gluttertest.ByType[*CustomScrollView](), ui.Offset{Dy: -50}, 10000.0)
	ts.PumpAndSettle()

	// We are now at the trailing edge
	if controller.Offset() != 150.0 {
		t.Errorf("offset = %v, want 150", controller.Offset())
	}
	expectAt(t, ts, keys, "the trailing edge", ui.Offset{Dy: -150}, ui.Offset{Dy: 100}, ui.Offset{Dy: 350})

	// Now fling to the leading edge
	ts.Fling(gluttertest.ByType[*CustomScrollView](), ui.Offset{Dy: 50}, 10000.0)
	ts.Pump(100 * time.Millisecond)
	ts.Pump(100 * time.Millisecond)
	ts.Pump(100 * time.Millisecond)
	ts.Pump(100 * time.Millisecond)

	// The boxes should now be at different locations because of the scaling.
	if controller.Offset() != 0.0 || topLeftOf(ts, keys[0]).Dy != 0.0 || topLeftOf(ts, keys[1]).Dy <= 250.9 || topLeftOf(ts, keys[2]).Dy <= 500.9 {
		t.Errorf("offset %v, boxes %v %v %v", controller.Offset(), topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}

	ts.PumpAndSettle()

	// The boxes should now be at their final position.
	if controller.Offset() != 0.0 {
		t.Errorf("offset = %v, want 0", controller.Offset())
	}
	expectAt(t, ts, keys, "the final position", ui.Offset{}, ui.Offset{Dy: 250}, ui.Offset{Dy: 500})
}

func TestChangingScrollDirectionDuringRecedeAnimationWillNotChangeTheStretchDirection(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(buildStretchTest(keys, controller, stretchTestOpts{boxHeight: 205.0}))

	expectIndicators(t, ts)

	// Fling to the trailing edge
	ts.Fling(gluttertest.ByType[*CustomScrollView](), ui.Offset{Dy: -50}, 10000.0)
	ts.PumpAndSettle()

	expectAt(t, ts, keys, "the trailing edge", ui.Offset{Dy: -15}, ui.Offset{Dy: 190}, ui.Offset{Dy: 395})

	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll to the trailing edge
	gesture.MoveBy(ui.Offset{Dy: -200})
	ts.PumpAndSettle()

	if topLeftOf(ts, keys[0]).Dy >= -25.0 || topLeftOf(ts, keys[1]).Dy >= 185.0 || topLeftOf(ts, keys[2]).Dy >= 392.0 {
		t.Errorf("overscrolled the trailing edge: %v %v %v", topLeftOf(ts, keys[0]), topLeftOf(ts, keys[1]), topLeftOf(ts, keys[2]))
	}

	// This will trigger the recede animation
	// The y offset of the boxes should be increasing, since the boxes were stretched
	// toward the leading edge.
	gesture.MoveBy(ui.Offset{Dy: 150})
	ts.Pump(100 * time.Millisecond)

	// Explicitly check that the box1 offset is not 0.0, since this would probably mean that
	// the stretch direction is wrong.
	if stretchEffectStrength(ts) == 0.0 {
		t.Error("the stretch strength is 0 while receding")
	}

	gesture.Up()
	ts.Pump(100 * time.Millisecond)

	// At this point, the stretchStrength should be close to 0.
	if got := math.Abs(stretchEffectStrength(ts)); got >= 0.01 {
		t.Errorf("stretchStrength = %v, want close to 0", got)
	}

	ts.PumpAndSettle()

	// The recede animation is done now, we should now be at the leading edge.
	expectAt(t, ts, keys, "the leading edge", ui.Offset{}, ui.Offset{Dy: 205}, ui.Offset{Dy: 410})
}

// scenesHaveImageFilter is whether the last frame has an ImageFilterLayer.
func sceneHasImageFilter(ts *gluttertest.Tester) bool {
	scenes := ts.Scenes()
	var visit func(layer ui.Layer) bool
	visit = func(layer ui.Layer) bool {
		if _, ok := layer.(*ui.ImageFilterLayer); ok {
			return true
		}
		for _, child := range layer.Children() {
			if visit(child) {
				return true
			}
		}
		return false
	}
	return visit(scenes[len(scenes)-1].Root())
}

func TestStretchOverscrollOnlyUsesImageFilterDuringStretchEffect(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(buildStretchTest(keys, controller, stretchTestOpts{axis: painting.AxisHorizontal}))

	if sceneHasImageFilter(ts) {
		t.Error("the scene has an ImageFilterLayer before the stretch")
	}

	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*CustomScrollView]()))
	// Overscroll
	gesture.MoveBy(ui.Offset{Dx: 200})
	ts.PumpAndSettle()

	if !sceneHasImageFilter(ts) {
		t.Error("the scene has no ImageFilterLayer during the stretch")
	}
	gesture.Up()
	ts.PumpAndSettle()
}

// highFrictionClampingScrollPhysics is _HighFrictionClampingScrollPhysics of
// the Dart test.
type highFrictionClampingScrollPhysics struct{ ScrollPhysicsBase }

func newHighFrictionClampingScrollPhysics(parent ScrollPhysics) *highFrictionClampingScrollPhysics {
	p := &highFrictionClampingScrollPhysics{}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, parent)
	return p
}

func (p *highFrictionClampingScrollPhysics) ApplyTo(ancestor ScrollPhysics) ScrollPhysics {
	return newHighFrictionClampingScrollPhysics(p.BuildParent(ancestor))
}

func (p *highFrictionClampingScrollPhysics) CreateBallisticSimulation(position ScrollMetrics, velocity float64) physics.Simulation {
	return physics.NewClampingScrollSimulation(position.Pixels(), velocity, physics.ClampingScrollSimulationOpts{
		Friction: foundation.Some(0.94), Tolerance: foundation.Some(p.ToleranceFor(position))})
}

func TestStretchingAnimationCompletesAfterFlingUnderScrollPhysicsWithHighFriction(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/146277
	useAndroid(t)
	ts := gluttertest.New(t)
	keys := newStretchKeys()
	controller := NewScrollController()
	defer controller.Dispose()
	var overscrollNotification *OverscrollNotification

	ts.PumpWidget(NewNotificationListener(func(notification *OverscrollNotification) bool {
		overscrollNotification = notification
		return false
	}, buildStretchTest(keys, controller, stretchTestOpts{physics: newHighFrictionClampingScrollPhysics(nil)})))

	expectIndicators(t, ts)
	if controller.Offset() != 0.0 {
		t.Errorf("offset = %v", controller.Offset())
	}
	expectAt(t, ts, keys, "the start", ui.Offset{}, ui.Offset{Dy: 250}, ui.Offset{Dy: 500})

	// We fling to the trailing edge and let it settle.
	ts.Fling(gluttertest.ByType[*CustomScrollView](), ui.Offset{Dy: -50}, 10000.0)
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 300 * time.Millisecond})

	// We are now at the trailing edge
	if overscrollNotification == nil || overscrollNotification.Velocity >= 25 {
		t.Errorf("the last overscroll notification is %v", overscrollNotification)
	}
	if controller.Offset() != 150.0 {
		t.Errorf("offset = %v, want 150", controller.Offset())
	}
	expectAt(t, ts, keys, "the trailing edge", ui.Offset{Dy: -150}, ui.Offset{Dy: 100}, ui.Offset{Dy: 350})
}

func TestFlingOverscrollTriggersStretchingAnimationCorrectly(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/169659
	tests := []struct {
		name    string
		reverse bool
		start   [3]ui.Offset
		fling   float64
		greater bool
	}{
		{"forward", false, [3]ui.Offset{{}, {Dy: 250}, {Dy: 500}}, 50, true},
		{"in reverse", true, [3]ui.Offset{{Dy: 350}, {Dy: 100}, {Dy: -150}}, -50, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useAndroid(t)
			ts := gluttertest.New(t)
			keys := newStretchKeys()
			controller := NewScrollController()
			defer controller.Dispose()

			ts.PumpWidget(buildStretchTest(keys, controller, stretchTestOpts{reverse: tt.reverse}))

			expectIndicators(t, ts)
			if controller.Offset() != 0.0 {
				t.Errorf("offset = %v", controller.Offset())
			}
			expectAt(t, ts, keys, "the start", tt.start[0], tt.start[1], tt.start[2])

			// Simulates a fast and short user fling.
			ts.Fling(gluttertest.ByType[*CustomScrollView](), ui.Offset{Dy: tt.fling}, 1000.0)

			// Records the current stretch strength of the StretchEffect widget
			// before pumping additional frames, so it can be compared later
			// to verify that the stretch increases after the fling.
			progressStrength := stretchEffectStrength(ts)

			ts.PumpFrames(ts.Widget(gluttertest.ByType[*Directionality]()), 50*time.Millisecond)

			// Verifies that the stretchStrength has increased compared to the
			// previously recorded value, ensuring the fling triggers stretching.
			got := stretchEffectStrength(ts)
			if (tt.greater && got <= progressStrength) || (!tt.greater && got >= progressStrength) {
				t.Errorf("stretchStrength = %v after %v", got, progressStrength)
			}

			ts.PumpAndSettle()
			if got := math.Abs(stretchEffectStrength(ts)); got != 0.0 {
				t.Errorf("stretchStrength = %v after settling, want 0", got)
			}
		})
	}
}
