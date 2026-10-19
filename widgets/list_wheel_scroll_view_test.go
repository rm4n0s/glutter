package widgets_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/list_wheel_scroll_view_test.dart.

// callbackPainter is TestCallbackPainter of the Dart tests: it calls a function
// every time it paints.
type callbackPainter struct {
	rendering.CustomPainterBase
	onPaint func()
}

func newCallbackPainter(onPaint func()) *callbackPainter {
	return &callbackPainter{rendering.NewCustomPainterBase(nil), onPaint}
}

func (p *callbackPainter) Paint(*ui.Canvas, ui.Size)                  { p.onPaint() }
func (p *callbackPainter) ShouldRepaint(rendering.CustomPainter) bool { return true }
func (p *callbackPainter) HitTest(ui.Offset) foundation.Opt[bool]     { return foundation.None[bool]() }

func paintingChildren(n int, painted *[]int) []Widget {
	out := make([]Widget, n)
	for i := range out {
		i := i
		out[i] = NewCustomPaint(nil, CustomPaintOpts{Painter: newCallbackPainter(func() { *painted = append(*painted, i) })})
	}
	return out
}

func requireInts(t *testing.T, got []int, want ...int) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestListWheelScrollViewConstructionChecks(t *testing.T) {
	t.Run("needs positive diameter ratio", func(t *testing.T) {
		requirePanic(t, func() {
			NewListWheelScrollView(20, nil, ListWheelScrollViewOpts{DiameterRatio: foundation.Some(-2.0)})
		})
	})
	t.Run("needs positive magnification", func(t *testing.T) {
		requirePanic(t, func() {
			NewListWheelScrollView(20, []Widget{NewContainer(nil)}, ListWheelScrollViewOpts{UseMagnifier: true, Magnification: foundation.Some(-1.0)})
		})
	})
	t.Run("needs valid overAndUnderCenterOpacity", func(t *testing.T) {
		skipUnlessDebug(t)
		for _, v := range []float64{-1, 2} {
			requirePanic(t, func() {
				NewListWheelScrollView(20, []Widget{NewContainer(nil)}, ListWheelScrollViewOpts{OverAndUnderCenterOpacity: foundation.Some(v)})
			})
		}
		for _, o := range []ListWheelScrollViewOpts{{}, {OverAndUnderCenterOpacity: foundation.Some(0.0)}} {
			if panicOf(func() { NewListWheelScrollView(20, []Widget{NewContainer(nil)}, o) }) != nil {
				t.Errorf("%+v panicked", o)
			}
		}
	})
	t.Run("can have zero child", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(ltr(NewListWheelScrollView(50, nil)))
		if got := ts.Size(gluttertest.ByType[*ListWheelScrollView]()); got != (ui.Size{Width: 800, Height: 600}) {
			t.Errorf("size = %v", got)
		}
	})
}

func TestFixedExtentScrollControllerOnAttachAndOnDetach(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	attach, detach := 0, 0
	controller := NewFixedExtentScrollController(FixedExtentScrollControllerOpts{
		OnAttach: func(ScrollPosition) { attach++ }, OnDetach: func(ScrollPosition) { detach++ }})
	defer controller.Dispose()
	ts.PumpWidget(ltr(NewListWheelScrollView(100, textItems(10), ListWheelScrollViewOpts{Controller: controller})))
	if attach != 1 || detach != 0 {
		t.Errorf("attach %d detach %d, want 1 0", attach, detach)
	}
	ts.PumpWidget(ltr(NewSizedBox(nil)))
	if attach != 1 || detach != 1 {
		t.Errorf("attach %d detach %d, want 1 1", attach, detach)
	}
}

func wheelOfSizedText(delegate ListWheelChildDelegate, controller ScrollController) Widget {
	return ltr(NewListWheelScrollViewUseDelegate(100, delegate, ListWheelScrollViewOpts{Controller: controller, OnSelectedItemChanged: func(int) {}}))
}

func sizedText(index int) Widget {
	return NewSizedBox(NewText(fmt.Sprint(index)), SizedBoxOpts{Width: foundation.Some(400.0), Height: foundation.Some(100.0)})
}

func sizedTextTopLeft(ts *gluttertest.Tester, text string) ui.Offset {
	return ts.TopLeft(gluttertest.Ancestor(gluttertest.ByText(text), gluttertest.ByType[*SizedBox]()))
}

func requireOffsetNear(t *testing.T, got, want ui.Offset, epsilon float64) {
	t.Helper()
	if !(moreOrLessEqualsEps(got.Dx, want.Dx, epsilon) && moreOrLessEqualsEps(got.Dy, want.Dy, epsilon)) {
		t.Errorf("offset = %v, want %v (±%v)", got, want, epsilon)
	}
}

func moreOrLessEqualsEps(a, b, eps float64) bool { return a-b <= eps && b-a <= eps }

func TestListWheelScrollViewInfiniteLoopingList(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewFixedExtentScrollController()
	defer controller.Dispose()
	children := make([]Widget, 10)
	for i := range children {
		children[i] = sizedText(i)
	}
	ts.PumpWidget(wheelOfSizedText(NewListWheelChildLoopingListDelegate(children), controller))
	// The first item is at the center of the viewport.
	requireOffsetNear(t, sizedTextTopLeft(ts, "0"), ui.Offset{Dx: 200, Dy: 250}, 0.001)
	// The last item is just before the first item.
	requireOffsetNear(t, sizedTextTopLeft(ts, "9"), ui.Offset{Dx: 200, Dy: 150}, 15)
	controller.JumpTo(1000)
	ts.Pump()
	// The list has looped back past the end.
	requireOffsetNear(t, sizedTextTopLeft(ts, "0"), ui.Offset{Dx: 200, Dy: 250}, 0.001)
}

func TestListWheelScrollViewInfiniteChildBuilder(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewFixedExtentScrollController()
	defer controller.Dispose()
	ts.PumpWidget(wheelOfSizedText(NewListWheelChildBuilderDelegate(func(_ BuildContext, i int) Widget { return sizedText(i) }), controller))
	controller.JumpTo(-100000)
	ts.Pump()
	requireOffsetNear(t, sizedTextTopLeft(ts, "-1000"), ui.Offset{Dx: 200, Dy: 250}, 0.001)
	controller.JumpTo(100000)
	ts.Pump()
	requireOffsetNear(t, sizedTextTopLeft(ts, "1000"), ui.Offset{Dx: 200, Dy: 250}, 0.001)
}

func TestListWheelScrollViewChildBuilderWithLowerAndUpperLimits(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var painted []int
	controller := NewFixedExtentScrollController(FixedExtentScrollControllerOpts{InitialItem: -10})
	defer controller.Dispose()
	ts.PumpWidget(wheelOfSizedText(NewListWheelChildBuilderDelegate(func(_ BuildContext, i int) Widget {
		if i < -15 || i > -5 {
			return nil
		}
		return NewSizedBox(NewCustomPaint(nil, CustomPaintOpts{Painter: newCallbackPainter(func() { painted = append(painted, i) })}),
			SizedBoxOpts{Width: foundation.Some(400.0), Height: foundation.Some(100.0)})
	}), controller))
	requireInts(t, painted, -13, -12, -11, -10, -9, -8, -7)
	// A fling with a high velocity stops at the lower limit.
	ts.Fling(gluttertest.ByType[*ListWheelScrollView](), ui.Offset{Dy: 1000}, 1000)
	ts.PumpAndSettle()
	if controller.SelectedItem() != -15 {
		t.Errorf("selected = %d, want -15", controller.SelectedItem())
	}
	// And at the upper limit.
	ts.Fling(gluttertest.ByType[*ListWheelScrollView](), ui.Offset{Dy: -1000}, 1000)
	ts.PumpAndSettle()
	if controller.SelectedItem() != -5 {
		t.Errorf("selected = %d, want -5", controller.SelectedItem())
	}
}

func TestListWheelScrollViewSelectedItemNotifications(t *testing.T) {
	placeholders := func(n int) []Widget {
		out := make([]Widget, n)
		for i := range out {
			out[i] = NewPlaceholder(nil)
		}
		return out
	}
	t.Run("no callback on the first build", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		called := false
		ts.PumpWidget(ltr(NewListWheelScrollView(100, []Widget{NewSizedBox(NewCenter(NewText("blah")), SizedBoxOpts{Width: foundation.Some(200.0)})},
			ListWheelScrollViewOpts{OnSelectedItemChanged: func(int) { called = true }})))
		if called {
			t.Error("onSelectedItemChanged was called")
		}
	})
	t.Run("a new item is closest to the center", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var selected []int
		ts.PumpWidget(ltr(NewListWheelScrollView(100, placeholders(10), ListWheelScrollViewOpts{OnSelectedItemChanged: func(i int) { selected = append(selected, i) }})))
		g := ts.StartGesture(ui.Offset{Dx: 10, Dy: 10})
		g.MoveBy(ui.Offset{Dy: -49})
		requireInts(t, selected)
		g.MoveBy(ui.Offset{Dy: -1})
		requireInts(t, selected, 1)
		g.MoveBy(ui.Offset{Dy: -99})
		requireInts(t, selected, 1)
		g.MoveBy(ui.Offset{Dy: -1})
		requireInts(t, selected, 1, 2)
		g.MoveBy(ui.Offset{Dy: 50})
		requireInts(t, selected, 1, 2, 1)
	})
	t.Run("on scroll end", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var selected []int
		ts.PumpWidget(ltr(NewListWheelScrollView(100, placeholders(10), ListWheelScrollViewOpts{
			OnSelectedItemChanged: func(i int) { selected = append(selected, i) }, ChangeReportingBehavior: ChangeReportingBehaviorOnScrollEnd})))
		g := ts.StartGesture(ui.Offset{Dx: 10, Dy: 10})
		g.MoveBy(ui.Offset{Dy: -49})
		requireInts(t, selected)
		g.MoveBy(ui.Offset{Dy: -1})
		requireInts(t, selected)
		g.MoveBy(ui.Offset{Dy: -99})
		requireInts(t, selected)
		g.MoveBy(ui.Offset{Dy: -1})
		g.Up()
		requireInts(t, selected, 2)
		g = ts.StartGesture(ui.Offset{Dx: 10, Dy: 10})
		g.MoveBy(ui.Offset{Dy: 100})
		requireInts(t, selected, 2)
		g.Up()
		requireInts(t, selected, 2, 1)
	})
	t.Run("reports only in the valid range", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var selected []int
		ts.PumpWidget(ltr(NewListWheelScrollView(100, placeholders(10), ListWheelScrollViewOpts{OnSelectedItemChanged: func(i int) { selected = append(selected, i) }})))
		g := ts.StartGesture(ui.Offset{Dx: 10, Dy: 10})
		// First move back past the beginning.
		g.MoveBy(ui.Offset{Dy: 70})
		for v := 0.0; v > -2000; v -= 10 {
			// Then gradually move down by a total vertical extent much higher than
			// the scrollable extent.
			g.MoveTo(ui.Offset{Dy: v})
		}
		requireInts(t, selected, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	})
}

func TestListWheelScrollViewController(t *testing.T) {
	build := func(controller *FixedExtentScrollController, painted *[]int, onChange func(int)) Widget {
		return ltr(NewListWheelScrollView(100, paintingChildren(100, painted), ListWheelScrollViewOpts{Controller: controller, OnSelectedItemChanged: onChange}))
	}
	t.Run("initialItem", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewFixedExtentScrollController(FixedExtentScrollControllerOpts{InitialItem: 10})
		defer controller.Dispose()
		var painted []int
		ts.PumpWidget(build(controller, &painted, nil))
		// The screen is 600 high, item 10 is in the center and each item is 100 high.
		requireInts(t, painted, 7, 8, 9, 10, 11, 12, 13)
	})
	t.Run("jump", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewFixedExtentScrollController(FixedExtentScrollControllerOpts{InitialItem: 10})
		defer controller.Dispose()
		var painted []int
		ts.PumpWidget(build(controller, &painted, nil))
		requireInts(t, painted, 7, 8, 9, 10, 11, 12, 13)
		painted = nil
		controller.JumpToItem(0)
		ts.Pump()
		requireInts(t, painted, 0, 1, 2, 3)
		if controller.SelectedItem() != 0 {
			t.Errorf("selected = %d", controller.SelectedItem())
		}
	})
	t.Run("animateToItem", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewFixedExtentScrollController(FixedExtentScrollControllerOpts{InitialItem: 10})
		defer controller.Dispose()
		var painted []int
		ts.PumpWidget(build(controller, &painted, nil))
		requireInts(t, painted, 7, 8, 9, 10, 11, 12, 13)
		painted = nil
		controller.AnimateToItem(0, time.Second, animation.Curves.Linear)
		ts.Pump()
		ts.Pump(time.Second)
		requireInts(t, painted, 0, 1, 2, 3)
		if controller.SelectedItem() != 0 {
			t.Errorf("selected = %d", controller.SelectedItem())
		}
	})
	t.Run("onSelectedItemChanged and the controller are in sync", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var selected []int
		controller := NewFixedExtentScrollController(FixedExtentScrollControllerOpts{InitialItem: 10})
		defer controller.Dispose()
		var painted []int
		ts.PumpWidget(build(controller, &painted, func(i int) { selected = append(selected, i) }))
		g := ts.StartGesture(ui.Offset{Dx: 10, Dy: 10})
		g.MoveBy(ui.Offset{Dy: -49})
		ts.Pump()
		requireInts(t, selected)
		if controller.SelectedItem() != 10 {
			t.Errorf("selected = %d, want 10", controller.SelectedItem())
		}
		g.MoveBy(ui.Offset{Dy: -1})
		ts.Pump()
		requireInts(t, selected, 11)
		if controller.SelectedItem() != 11 {
			t.Errorf("selected = %d, want 11", controller.SelectedItem())
		}
		g.MoveBy(ui.Offset{Dy: 70})
		ts.Pump()
		requireInts(t, selected, 11, 10)
		if controller.SelectedItem() != 10 {
			t.Errorf("selected = %d, want 10", controller.SelectedItem())
		}
	})
}
