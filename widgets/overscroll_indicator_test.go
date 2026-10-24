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
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/overscroll_indicator_test.dart. The
// `paints` matcher of flutter_test is replaced by recordForeground, which
// records what the foreground painter of a CustomPaint draws, and by the small
// matcher paintsInOrder (below): the expected calls must appear in the order
// given, other calls are skipped, as with `paints`.

// paintCall is a call to the canvas, as the `paints` matcher sees it.
type paintCall struct {
	kind  string // save, restore, translate, scale, rotate, clipRect, circle
	a, b  float64
	color ui.Color
}

// recordForeground records what the foreground painter of the one CustomPaint
// the finder finds draws, within the save and the restore that RenderCustomPaint
// puts around it.
func recordForeground(t *testing.T, ts *gluttertest.Tester, f gluttertest.Finder) []paintCall {
	t.Helper()
	renderObject, ok := ts.RenderObject(f).(*rendering.RenderCustomPaint)
	if !ok {
		t.Fatalf("%s is not a CustomPaint", f.Describe())
	}
	recorder := ui.NewPictureRecorder()
	canvas := ui.NewCanvas(recorder)
	canvas.Save()
	renderObject.ForegroundPainter().Paint(canvas, ts.Size(f))
	canvas.Restore()
	var calls []paintCall
	for _, op := range recorder.EndRecording().DisplayList().Ops() {
		switch op := op.(type) {
		case ui.OpSave:
			calls = append(calls, paintCall{kind: "save"})
		case ui.OpRestore:
			calls = append(calls, paintCall{kind: "restore"})
		case ui.OpTransform:
			m := op.Matrix
			switch {
			case m.Entry(0, 0) == 1 && m.Entry(1, 1) == 1 && m.Entry(0, 1) == 0 && m.Entry(1, 0) == 0:
				calls = append(calls, paintCall{kind: "translate", a: float64(m.Entry(0, 3)), b: float64(m.Entry(1, 3))})
			case m.Entry(0, 1) == 0 && m.Entry(1, 0) == 0 && m.Entry(0, 3) == 0 && m.Entry(1, 3) == 0:
				calls = append(calls, paintCall{kind: "scale", a: float64(m.Entry(0, 0)), b: float64(m.Entry(1, 1))})
			default:
				calls = append(calls, paintCall{kind: "rotate", a: math.Atan2(float64(m.Entry(1, 0)), float64(m.Entry(0, 0)))})
			}
		case ui.OpClipRect:
			calls = append(calls, paintCall{kind: "clipRect"})
		case ui.OpDrawCircle:
			calls = append(calls, paintCall{kind: "circle", a: op.Center.Dx, b: op.Center.Dy, color: op.Paint.Color()})
		}
	}
	return calls
}

// want is an expected call, or a save and its restore (saveRestore).
type want struct {
	match func(paintCall) bool
	pair  bool
}

func kindOf(kind string) want { return want{match: func(c paintCall) bool { return c.kind == kind }} }

func wantSave() want    { return kindOf("save") }
func wantRestore() want { return kindOf("restore") }
func wantScale() want   { return kindOf("scale") }

// wantSaveRestore is `saveRestore()`: a save and the restore that closes it.
func wantSaveRestore() want { return want{pair: true} }

// wantScaleY is `scale(y: y)`.
func wantScaleY(y float64) want {
	return want{match: func(c paintCall) bool { return c.kind == "scale" && c.b == y }}
}

// wantTranslateY is `translate(y: y)`.
func wantTranslateY(y float64) want {
	return want{match: func(c paintCall) bool { return c.kind == "translate" && math.Abs(c.b-y) < 1e-4 }}
}

// wantRotate is `rotate(angle: angle)`.
func wantRotate(angle float64) want {
	return want{match: func(c paintCall) bool { return c.kind == "rotate" && math.Abs(c.a-angle) < 1e-6 }}
}

// wantCircle is `circle(...)`, with the conditions given.
func wantCircle(conditions ...func(paintCall) bool) want {
	return want{match: func(c paintCall) bool {
		if c.kind != "circle" {
			return false
		}
		for _, condition := range conditions {
			if !condition(c) {
				return false
			}
		}
		return true
	}}
}

func atX(x float64) func(paintCall) bool { return func(c paintCall) bool { return c.a == x } }
func withColor(color ui.Color) func(paintCall) bool {
	return func(c paintCall) bool { return c.color == color }
}

// paintsInOrder is the `paints` matcher: whether the calls contain the expected
// ones, in the order given.
func paintsInOrder(calls []paintCall, wants ...want) bool {
	i := 0
	for _, w := range wants {
		if w.pair {
			depth := 0
			found := false
			for ; i < len(calls) && !found; i++ {
				switch {
				case depth == 0 && calls[i].kind == "save":
					depth = 1
				case depth > 0 && calls[i].kind == "save":
					depth++
				case depth > 0 && calls[i].kind == "restore":
					depth--
					found = depth == 0
				}
			}
			if !found {
				return false
			}
			continue
		}
		for i < len(calls) && !w.match(calls[i]) {
			i++
		}
		if i == len(calls) {
			return false
		}
		i++
	}
	return true
}

func hasCircle(calls []paintCall) bool { return paintsInOrder(calls, wantCircle()) }

func expectPaints(t *testing.T, calls []paintCall, why string, wants ...want) {
	t.Helper()
	if !paintsInOrder(calls, wants...) {
		t.Errorf("%s: the painter did not paint the expected calls, it painted %v", why, calls)
	}
}

func expectNotPaints(t *testing.T, calls []paintCall, why string, wants ...want) {
	t.Helper()
	if paintsInOrder(calls, wants...) {
		t.Errorf("%s: the painter painted the calls it must not paint: %v", why, calls)
	}
}

// slowDrag is the helper of the Dart test: a drag in ten moves, 20 ms apart.
func slowDrag(ts *gluttertest.Tester, start, offset ui.Offset) {
	gesture := ts.StartGesture(start)
	for index := 0; index < 10; index++ {
		gesture.MoveBy(offset)
		ts.Pump(20 * time.Millisecond)
	}
	gesture.Up()
}

func settleSeconds(ts *gluttertest.Tester) {
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: time.Second})
}

// glowBehavior is TestScrollBehavior1, TestScrollBehavior2 and
// TestScrollBehaviorNoGlow of the Dart test. They are different types in Dart, and
// a ScrollConfiguration notifies its dependents when the type of the behavior
// changes, so glowBehavior2 stands for TestScrollBehavior2.
type glowBehavior struct {
	ScrollBehaviorBase
	color  ui.Color
	noGlow bool
}

func newGlowBehavior(color ui.Color, noGlow bool) *glowBehavior {
	b := &glowBehavior{color: color, noGlow: noGlow}
	InitScrollBehaviorBase(&b.ScrollBehaviorBase, b)
	return b
}

func (b *glowBehavior) BuildOverscrollIndicator(_ BuildContext, child Widget, details ScrollableDetails) Widget {
	if b.noGlow {
		return child
	}
	return NewGlowingOverscrollIndicator(details.Direction, b.color, child)
}

type glowBehavior2 struct{ glowBehavior }

func newGlowBehavior2(color ui.Color) *glowBehavior2 {
	b := &glowBehavior2{glowBehavior{color: color}}
	InitScrollBehaviorBase(&b.ScrollBehaviorBase, b)
	return b
}

func tallSliver(height float64) []Widget {
	return []Widget{NewSliverToBoxAdapter(NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(height)}))}
}

func theCustomPaint() gluttertest.Finder { return gluttertest.ByType[*CustomPaint]() }

func TestOverscrollIndicatorColor(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCustomScrollView(tallSliver(2000))))
	painter := func() []paintCall { return recordForeground(t, ts, theCustomPaint()) }

	expectNotPaints(t, painter(), "at the start", wantCircle())

	// the scroll gesture from tester.scroll happens in zero time, so nothing should appear:
	ts.Drag(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: 100})
	expectNotPaints(t, painter(), "after the drag", wantCircle())
	ts.Pump() // allow the ticker to register itself
	expectNotPaints(t, painter(), "after the first pump", wantCircle())
	ts.Pump(100 * time.Millisecond) // animate
	expectNotPaints(t, painter(), "after animating", wantCircle())

	gesture := ts.StartGesture(ui.Offset{Dx: 200, Dy: 200})
	ts.Pump(100 * time.Millisecond) // animate
	expectNotPaints(t, painter(), "while holding", wantCircle())
	gesture.Up()
	expectNotPaints(t, painter(), "after releasing", wantCircle())

	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 5})
	expectPaints(t, painter(), "after the slow drag", wantCircle(withColor(ui.NewColor(0x0DFFFFFF))))

	settleSeconds(ts)
	expectNotPaints(t, painter(), "after settling", wantCircle())
}

func TestOverscrollIndicatorNestedScrollable(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewGlowingOverscrollIndicator(painting.AxisDirectionDown, ui.NewColor(0x0DFFFFFF),
		NewSingleChildScrollView(
			NewSizedBox(NewCustomScrollView(tallSliver(2000)), SizedBoxOpts{Width: foundation.Some(600.0)}),
			SingleChildScrollViewOpts{ScrollDirection: painting.AxisHorizontal}),
		GlowingOverscrollIndicatorOpts{NotificationPredicate: func(notification ScrollNotification) bool { return notification.Depth() == 1 }})))

	outer := gluttertest.First(theCustomPaint())
	inner := gluttertest.Last(theCustomPaint())

	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 5})
	expectPaints(t, recordForeground(t, ts, outer), "the outer painter", wantCircle())
	expectPaints(t, recordForeground(t, ts, inner), "the inner painter", wantCircle())
}

func TestOverscrollIndicatorChangesSideWhenYouDragOnTheOtherSide(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCustomScrollView(tallSliver(2000))))
	painter := func() []paintCall { return recordForeground(t, ts, theCustomPaint()) }

	slowDrag(ts, ui.Offset{Dx: 400, Dy: 200}, ui.Offset{Dy: 10})
	expectPaints(t, painter(), "dragging in the middle", wantCircle(atX(400)))
	firstCircle := func(calls []paintCall) paintCall {
		for _, c := range calls {
			if c.kind == "circle" {
				return c
			}
		}
		t.Fatal("no circle was painted")
		return paintCall{}
	}
	slowDrag(ts, ui.Offset{Dx: 100, Dy: 200}, ui.Offset{Dy: 10})
	if c := firstCircle(painter()); c.a >= 400 {
		t.Errorf("Dragging on left hand side did not overscroll on left hand side: the circle is at %v", c.a)
	}
	slowDrag(ts, ui.Offset{Dx: 700, Dy: 200}, ui.Offset{Dy: 10})
	if c := firstCircle(painter()); c.a <= 400 {
		t.Errorf("Dragging on right hand side did not overscroll on right hand side: the circle is at %v", c.a)
	}

	settleSeconds(ts)
	expectNotPaints(t, painter(), "after settling", wantCircle())
}

func TestOverscrollIndicatorChangesSideWhenYouShiftSides(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCustomScrollView(tallSliver(2000))))
	painter := func() []paintCall { return recordForeground(t, ts, theCustomPaint()) }
	gesture := ts.StartGesture(ui.Offset{Dx: 300, Dy: 200})
	gesture.MoveBy(ui.Offset{Dy: 10})
	ts.Pump(20 * time.Millisecond)
	oldX := 0.0
	for index := 0; index < 10; index++ {
		gesture.MoveBy(ui.Offset{Dx: 50, Dy: 50})
		ts.Pump(20 * time.Millisecond)
		calls := painter()
		if !hasCircle(calls) {
			t.Fatalf("move %d: no circle was painted", index)
		}
		for _, c := range calls {
			if c.kind != "circle" {
				continue
			}
			if c.a <= oldX {
				t.Errorf("move %d: Sliding to the right did not make the center of the radius slide to the right (%v after %v)", index, c.a, oldX)
			}
			oldX = c.a
			break
		}
	}
	gesture.Up()

	settleSeconds(ts)
	expectNotPaints(t, painter(), "after settling", wantCircle())
}

func TestFlippingDirectionOfScrollableDoesNotChangeOverscrollBehavior(t *testing.T) {
	tests := []struct {
		name    string
		reverse bool
		wants   []want
	}{
		{"down", false, []want{wantSave(), wantCircle(), wantRestore(), wantSave(), wantScaleY(-1.0), wantRestore(), wantRestore()}},
		{"up", true, []want{wantSave(), wantScaleY(-1.0), wantRestore(), wantSave(), wantCircle(), wantRestore(), wantRestore()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useAndroid(t)
			ts := gluttertest.New(t)
			ts.PumpWidget(ltr(NewCustomScrollView(tallSliver(20), CustomScrollViewOpts{
				Reverse: tt.reverse, Physics: NewAlwaysScrollableScrollPhysics(nil)})))
			painter := func() []paintCall { return recordForeground(t, ts, theCustomPaint()) }
			slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 5})
			expectPaints(t, painter(), "after the slow drag", tt.wants...)

			settleSeconds(ts)
			expectNotPaints(t, painter(), "after settling", wantCircle())
		})
	}
}

func TestOverscrollInBothDirections(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCustomScrollView(tallSliver(20), CustomScrollViewOpts{Physics: NewAlwaysScrollableScrollPhysics(nil)})))
	painter := func() []paintCall { return recordForeground(t, ts, theCustomPaint()) }
	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 5})
	expectPaints(t, painter(), "overscrolling the leading edge", wantCircle())
	expectNotPaints(t, painter(), "overscrolling the leading edge", wantCircle(), wantCircle())
	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: -5})
	expectPaints(t, painter(), "overscrolling both edges", wantCircle(), wantCircle())

	settleSeconds(ts)
	expectNotPaints(t, painter(), "after settling", wantCircle())
}

func TestOverscrollIgnoredFromAlternateAxis(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewScrollConfiguration(newGlowBehavior(ui.NewColor(0), true),
		NewGlowingOverscrollIndicator(painting.AxisDirectionRight, ui.NewColor(0xFF0000FF),
			NewCustomScrollView(tallSliver(20), CustomScrollViewOpts{Physics: NewAlwaysScrollableScrollPhysics(nil)})))))
	painter := func() []paintCall { return recordForeground(t, ts, theCustomPaint()) }
	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 5})
	expectNotPaints(t, painter(), "after the first drag", wantCircle())
	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: -5})
	expectNotPaints(t, painter(), "after the second drag", wantCircle())

	settleSeconds(ts)
	expectNotPaints(t, painter(), "after settling", wantCircle())
}

func TestOverscrollHorizontally(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCustomScrollView(tallSliver(20), CustomScrollViewOpts{
		ScrollDirection: painting.AxisHorizontal, Physics: NewAlwaysScrollableScrollPhysics(nil)})))
	painter := func() []paintCall { return recordForeground(t, ts, theCustomPaint()) }
	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dx: 5})
	expectPaints(t, painter(), "overscrolling the leading edge", wantRotate(math.Pi/2.0), wantCircle(), wantSaveRestore())
	expectNotPaints(t, painter(), "overscrolling the leading edge", wantCircle(), wantCircle())
	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dx: -5})
	expectPaints(t, painter(), "overscrolling both edges", wantRotate(math.Pi/2.0), wantCircle(), wantRotate(math.Pi/2.0), wantCircle())

	settleSeconds(ts)
	expectNotPaints(t, painter(), "after settling", wantCircle())
}

func TestNestedOverscrollsDoNotThrowExceptions(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewPageView([]Widget{
		NewListView([]Widget{NewContainer(nil, ContainerOpts{
			Width: foundation.Some(2000.0), Height: foundation.Some(2000.0), Color: foundation.Some(ui.NewColor(0xFF00FF00))})}),
	})))

	ts.DragFrom(ui.Offset{Dx: 100, Dy: 100}, ui.Offset{Dy: 2000})
	ts.PumpAndSettle()
}

func TestOverscrollChangingSettings(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)

	ts.PumpWidget(ltr(NewScrollConfiguration(newGlowBehavior(ui.NewColor(0xFF00FF00), false),
		NewCustomScrollView(tallSliver(20), CustomScrollViewOpts{
			ScrollDirection: painting.AxisHorizontal, Physics: NewAlwaysScrollableScrollPhysics(nil), Reverse: true}))))
	painter := func() []paintCall { return recordForeground(t, ts, theCustomPaint()) }
	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dx: 5})
	expectPaints(t, painter(), "the green glow", wantRotate(math.Pi/2.0), wantCircle(withColor(ui.NewColor(0x0A00FF00))))
	expectNotPaints(t, painter(), "the green glow", wantCircle(), wantCircle())

	settleSeconds(ts)
	ts.PumpWidget(ltr(NewScrollConfiguration(newGlowBehavior2(ui.NewColor(0xFF0000FF)),
		NewCustomScrollView(tallSliver(20), CustomScrollViewOpts{
			ScrollDirection: painting.AxisHorizontal, Physics: NewAlwaysScrollableScrollPhysics(nil)}))))
	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dx: 5})
	expectPaints(t, painter(), "the blue glow", wantRotate(math.Pi/2.0), wantCircle(withColor(ui.NewColor(0x0A0000FF))), wantSaveRestore())
	expectNotPaints(t, painter(), "the blue glow", wantCircle(), wantCircle())
}

// sliversAroundCenter are the slivers of the two tests with a CustomScrollView
// that has a sliver before its center.
func sliversAroundCenter(centerKey foundation.Key) []Widget {
	return []Widget{
		NewSliverListBuilder(func(_ BuildContext, index int) Widget { return NewText("First sliver " + strconv.Itoa(index)) },
			SliverBuilderOpts{ItemCount: foundation.Some(2)}),
		NewSliverListBuilder(func(_ BuildContext, index int) Widget { return NewText("Second sliver " + strconv.Itoa(index)) },
			SliverBuilderOpts{Key: centerKey, ItemCount: foundation.Some(5)}),
	}
}

func TestCustomScrollViewOverscrollIndicatorWorksIfThereIsSliverBeforeCenter(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	centerKey := foundation.NewUniqueKey()
	ts.PumpWidget(ltr(NewScrollConfiguration(newGlowBehavior2(ui.NewColor(0xFF0000FF)),
		NewCustomScrollView(sliversAroundCenter(centerKey), CustomScrollViewOpts{
			Center: centerKey, Physics: NewAlwaysScrollableScrollPhysics(nil)}))))

	if ts.Exists(gluttertest.ByText("First sliver 1")) {
		t.Fatal("the first sliver is visible")
	}

	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 300})
	if !ts.Exists(gluttertest.ByText("First sliver 1")) {
		t.Fatal("the first sliver is not visible")
	}
	// The scroll offset and paint extend should cancel out each other. (A
	// translation by zero is not recorded by the canvas.)
	calls := recordForeground(t, ts, theCustomPaint())
	expectPaints(t, calls, "the translation cancels out", wantSave(), wantScale(), wantCircle())
	for i, c := range calls {
		if c.kind != "circle" {
			continue
		}
		scale := i - 1 // the scale and the clip are before the circle
		for calls[scale].kind != "scale" {
			scale--
		}
		if before := calls[scale-1]; before.kind == "translate" && math.Abs(before.b) > 1e-4 {
			t.Errorf("the glow is translated by %v, want 0", before.b)
		}
	}
}

func TestCustomScrollViewOverscrollIndicatorWorksWithCenterAndPaintOffset(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	centerKey := foundation.NewUniqueKey()
	ts.PumpWidget(ltr(NewScrollConfiguration(newGlowBehavior2(ui.NewColor(0xFF0000FF)),
		NewNotificationListener(func(notification *OverscrollIndicatorNotification) bool {
			if notification.Leading {
				notification.PaintOffset = 50.0
			}
			return false
		}, NewCustomScrollView(sliversAroundCenter(centerKey), CustomScrollViewOpts{
			Center: centerKey, Physics: NewAlwaysScrollableScrollPhysics(nil)})))))

	if ts.Exists(gluttertest.ByText("First sliver 1")) {
		t.Fatal("the first sliver is visible")
	}

	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 5}) // offset will be magnified ten times
	if !ts.Exists(gluttertest.ByText("First sliver 1")) {
		t.Fatal("the first sliver is not visible")
	}
	// The OverscrollIndicator should respect the OverscrollIndicatorNotification.paintOffset setting.
	expectPaints(t, recordForeground(t, ts, theCustomPaint()), "the paint offset",
		wantSave(), wantTranslateY(50.0), wantScale(), wantCircle())
}

func TestOverscrollIndicatorShouldNotOverflowTheScrollableViewEdge(t *testing.T) {
	// Regressing test for https://github.com/flutter/flutter/issues/64149
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewNotificationListener(func(notification *OverscrollIndicatorNotification) bool {
		notification.PaintOffset = 50.0 // both the leading and trailing indicator have a 50.0 pixels offset.
		return false
	}, NewCustomScrollView(tallSliver(2000)))))
	painter := func() []paintCall { return recordForeground(t, ts, theCustomPaint()) }
	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 5})
	expectPaints(t, painter(), "the leading indicator", wantSave(), wantTranslateY(50.0), wantScale(), wantCircle())
	// Reverse scroll (30 pixels), and the offset < notification.paintOffset.
	ts.DragFrom(ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: -30})
	ts.Pump()
	// The OverscrollIndicator should move with the CustomScrollView.
	expectPaints(t, painter(), "reverse 30", wantSave(), wantTranslateY(50.0-30.0), wantScale(), wantCircle())

	// Reverse scroll (30+20 pixels) and offset == notification.paintOffset.
	ts.DragFrom(ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: -20})
	ts.Pump()
	// A translation by zero is not recorded by the canvas.
	expectPaints(t, painter(), "reverse 50", wantSave(), wantScale(), wantCircle())
	expectNotPaints(t, painter(), "reverse 50", wantTranslateY(0.0))

	// Reverse scroll (30+20+10 pixels) and offset > notification.paintOffset.
	ts.DragFrom(ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: -10})
	ts.Pump()
	// The OverscrollIndicator should not overflow the CustomScrollView's edge.
	expectPaints(t, painter(), "reverse 60", wantSave(), wantScale(), wantCircle())
	expectNotPaints(t, painter(), "reverse 60", wantTranslateY(0.0))

	ts.PumpAndSettle() // Finish the leading indicator.

	// trigger the trailing indicator
	slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: -200})
	expectPaints(t, painter(), "the trailing indicator", wantScaleY(-1.0), wantSave(), wantTranslateY(50.0), wantScale(), wantCircle())

	// Reverse scroll (30 pixels), and the offset < notification.paintOffset.
	ts.DragFrom(ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 30})
	ts.Pump()
	// The OverscrollIndicator should move with the CustomScrollView.
	expectPaints(t, painter(), "trailing reverse 30", wantScaleY(-1.0), wantSave(), wantTranslateY(50.0-30.0), wantScale(), wantCircle())

	// Reverse scroll (30+20 pixels) and offset == notification.paintOffset.
	ts.DragFrom(ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 20})
	ts.Pump()
	expectPaints(t, painter(), "trailing reverse 50", wantScaleY(-1.0), wantSave(), wantScale(), wantCircle())

	// Reverse scroll (30+20+10 pixels) and offset > notification.paintOffset.
	ts.DragFrom(ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 10})
	ts.Pump()
	// The OverscrollIndicator should not overflow the CustomScrollView's edge.
	expectPaints(t, painter(), "trailing reverse 60", wantScaleY(-1.0), wantSave(), wantScale(), wantCircle())
	expectNotPaints(t, painter(), "trailing reverse 60", wantTranslateY(0.0))
}

func TestOverscrollIndicatorNotificationPaintOffset(t *testing.T) {
	t.Run("Leading", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t)
		ts.PumpWidget(ltr(NewNotificationListener(func(notification *OverscrollIndicatorNotification) bool {
			if notification.Leading {
				notification.PaintOffset = 50.0
			}
			return false
		}, NewCustomScrollView(tallSliver(2000)))))
		painter := func() []paintCall { return recordForeground(t, ts, theCustomPaint()) }
		slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 5})
		// The OverscrollIndicator should respect the OverscrollIndicatorNotification.paintOffset setting.
		expectPaints(t, painter(), "the leading offset", wantSave(), wantTranslateY(50.0), wantScale(), wantCircle())
		// Reverse scroll direction.
		ts.DragFrom(ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: -30})
		ts.Pump()
		// The OverscrollIndicator should move with the CustomScrollView.
		expectPaints(t, painter(), "the leading offset after reversing", wantSave(), wantTranslateY(50.0-30.0), wantScale(), wantCircle())
	})
	t.Run("Trailing", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t)
		ts.PumpWidget(ltr(NewNotificationListener(func(notification *OverscrollIndicatorNotification) bool {
			if !notification.Leading {
				notification.PaintOffset = 50.0
			}
			return false
		}, NewCustomScrollView(tallSliver(2000)))))
		painter := func() []paintCall { return recordForeground(t, ts, theCustomPaint()) }
		ts.DragFrom(ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dx: 200, Dy: -10000})
		ts.Pump()
		slowDrag(ts, ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: -5})
		// The OverscrollIndicator should respect the OverscrollIndicatorNotification.paintOffset setting.
		expectPaints(t, painter(), "the trailing offset", wantScaleY(-1.0), wantSave(), wantTranslateY(50.0), wantScale(), wantCircle())
		// Reverse scroll direction.
		ts.DragFrom(ui.Offset{Dx: 200, Dy: 200}, ui.Offset{Dy: 30})
		ts.Pump()
		// The OverscrollIndicator should move with the CustomScrollView.
		expectPaints(t, painter(), "the trailing offset after reversing", wantScaleY(-1.0), wantSave(), wantTranslateY(50.0-30.0), wantScale(), wantCircle())
	})
}

func TestStretchingOverscrollIndicatorDoesNotCrashWhenChildIsNull(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewStretchingOverscrollIndicator(painting.AxisDirectionDown, nil)))
	if err := ts.TakeError(); err != nil {
		t.Errorf("the indicator reported %v", err)
	}
}

func TestGlowingOverscrollIndicatorDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxShrink(
		NewGlowingOverscrollIndicator(painting.AxisDirectionDown, ui.NewColor(0xFF112233), nil)))))
	if got := ts.Size(gluttertest.ByType[*GlowingOverscrollIndicator]()); got != (ui.Size{}) {
		t.Errorf("size = %v, want zero", got)
	}
}

func TestStretchingOverscrollIndicatorDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxShrink(
		NewStretchingOverscrollIndicator(painting.AxisDirectionDown, NewListView(nil))))))
	if got := ts.Size(gluttertest.ByType[*StretchingOverscrollIndicator]()); got != (ui.Size{}) {
		t.Errorf("size = %v, want zero", got)
	}
}

// The assertions of the indicators about the notifications they get (the tags
// of overscroll_indicator.go).
func TestOverscrollIndicatorsRejectAVelocityWithDragDetails(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	tests := []struct {
		name    string
		tag     string
		route   string
		indicat func(child Widget) Widget
	}{
		{"glow", "GlowingOverscrollVelocityWithDrag", "glowingOverscrollIndicatorState.handleScrollNotification.GlowingOverscrollVelocityWithDrag",
			func(child Widget) Widget {
				return NewGlowingOverscrollIndicator(painting.AxisDirectionDown, ui.NewColor(0xFFFFFFFF), child)
			}},
		{"stretch", "StretchingOverscrollVelocityWithDrag", "stretchingOverscrollIndicatorState.handleScrollNotification.StretchingOverscrollVelocityWithDrag",
			func(child Widget) Widget { return NewStretchingOverscrollIndicator(painting.AxisDirectionDown, child) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(ltr(tt.indicat(NewScrollConfiguration(newGlowBehavior(ui.NewColor(0), true),
				NewCustomScrollView(tallSliver(2000))))))
			state := scrollableOf(ts)
			details := foundation.Some(gestures.DragUpdateDetails{})
			notification := NewOverscrollNotification(state.Position(), state.Context(), -5, OverscrollNotificationOpts{DragDetails: details, Velocity: 10})
			v := panicOf(func() { DispatchNotification(state.Context(), notification) })
			err, _ := v.(error)
			appErr := requireErrorTag(t, err, tt.tag)
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}
