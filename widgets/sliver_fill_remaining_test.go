package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/sliver_fill_remaining_test.dart.
// Not ported: the semantics cases and the cases that use TestButton, Material's
// visual density or FlutterLogo.

var debugRed = ui.NewColor(0xFFFF0000)

func fillSliverBox() Widget {
	return NewSliverToBoxAdapter(NewContainer(nil, ContainerOpts{Color: foundation.Some(debugRed), Height: foundation.Some(150.0), Width: foundation.Some(150.0)}))
}

func fillApp(slivers []Widget, controller ScrollController, axis painting.Axis) Widget {
	return ltr(NewCustomScrollView(slivers, CustomScrollViewOpts{Controller: controller, ScrollDirection: axis}))
}

func TestSliverFillRemainingWithScrollBody(t *testing.T) {
	heightOf := func(ts *gluttertest.Tester) float64 { return ts.Size(gluttertest.ByType[*Container]()).Height }
	t.Run("no siblings", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewScrollController()
		defer controller.Dispose()
		ts.PumpWidget(fillApp([]Widget{NewSliverFillRemaining(NewContainer(nil))}, controller, painting.AxisVertical))
		for _, offset := range []float64{0, 50, -100, 0} {
			controller.JumpTo(offset)
			ts.Pump()
			if h := heightOf(ts); h != 600 {
				t.Errorf("offset %v: height = %v, want 600", offset, h)
			}
		}
	})
	t.Run("one sibling", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewScrollController()
		defer controller.Dispose()
		ts.PumpWidget(fillApp([]Widget{NewSliverToBoxAdapter(NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(100.0)})),
			NewSliverFillRemaining(NewContainer(nil))}, controller, painting.AxisVertical))
		if h := heightOf(ts); h != 500 {
			t.Errorf("height = %v, want 500", h)
		}
		for _, step := range []struct{ offset, height float64 }{{50, 550}, {-100, 400}, {0, 500}} {
			controller.JumpTo(step.offset)
			ts.Pump()
			if h := heightOf(ts); h != step.height {
				t.Errorf("offset %v: height = %v, want %v", step.offset, h, step.height)
			}
		}
	})
	t.Run("scrolls beyond viewportMainAxisExtent", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewScrollController()
		defer controller.Dispose()
		fill := foundation.NewValueKey("fill")
		ts.PumpWidget(fillApp([]Widget{fillSliverBox(), NewSliverFillRemaining(NewColoredBox(debugRed, nil, ColoredBoxOpts{Key: fill}))}, controller, painting.AxisVertical))
		if controller.Offset() != 0 || ts.Count(gluttertest.ByType[*Container]()) != 1 || ts.Count(gluttertest.ByKey(fill)) != 1 {
			t.Fatal("initial state")
		}
		controller.JumpTo(150)
		ts.PumpAndSettle()
		if controller.Offset() != 150 || ts.Count(gluttertest.ByType[*Container]()) != 0 || ts.Count(gluttertest.ByKey(fill)) != 1 {
			t.Errorf("offset %v, containers %d", controller.Offset(), ts.Count(gluttertest.ByType[*Container]()))
		}
	})
}

func TestSliverFillRemainingWithoutScrollBody(t *testing.T) {
	noBody := foundation.Some(false)
	t.Run("does not extend past viewportMainAxisExtent", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewScrollController()
		defer controller.Dispose()
		fill := foundation.NewValueKey("fill")
		ts.PumpWidget(fillApp([]Widget{fillSliverBox(), NewSliverFillRemaining(NewColoredBox(debugRed, nil, ColoredBoxOpts{Key: fill}),
			SliverFillRemainingOpts{HasScrollBody: noBody})}, controller, painting.AxisVertical))
		controller.JumpTo(150)
		ts.PumpAndSettle()
		if controller.Offset() != 0 || ts.Count(gluttertest.ByType[*Container]()) != 1 || ts.Count(gluttertest.ByKey(fill)) != 1 {
			t.Errorf("offset = %v", controller.Offset())
		}
	})
	t.Run("a child without size is sized by the extent", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		fill := foundation.NewValueKey("fill")
		slivers := []Widget{fillSliverBox(), NewSliverFillRemaining(NewColoredBox(debugRed, nil, ColoredBoxOpts{Key: fill}), SliverFillRemainingOpts{HasScrollBody: noBody})}
		ts.PumpWidget(fillApp(slivers, nil, painting.AxisVertical))
		if h := ts.Size(gluttertest.ByKey(fill)).Height; h != 450 {
			t.Errorf("height = %v, want 450", h)
		}
		ts.PumpWidget(fillApp(slivers, nil, painting.AxisHorizontal))
		if w := ts.Size(gluttertest.ByKey(fill)).Width; w != 650 {
			t.Errorf("width = %v, want 650", w)
		}
	})
	t.Run("a child with a smaller size is sized by the extent", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		key := NewGlobalKey[State]()
		button := foundation.NewValueKey("button")
		slivers := []Widget{fillSliverBox(), NewSliverFillRemaining(NewColoredBox(debugRed,
			NewAlign(NewSizedBox(nil, SizedBoxOpts{Key: button, Width: foundation.Some(116.0), Height: foundation.Some(48.0)}), AlignOpts{Alignment: painting.AlignmentBottomCenter}),
			ColoredBoxOpts{Key: key}), SliverFillRemainingOpts{HasScrollBody: noBody})}
		ts.PumpWidget(fillApp(slivers, nil, painting.AxisVertical))
		if h := ts.Size(gluttertest.ByKey(key)).Height; h != 450 {
			t.Errorf("height = %v, want 450", h)
		}
		if ts.BottomRight(gluttertest.ByKey(button)).Dy != 600 || ts.Center(gluttertest.ByKey(button)).Dx != 400 {
			t.Errorf("button at %v", ts.Center(gluttertest.ByKey(button)))
		}
		ts.PumpWidget(fillApp(slivers, nil, painting.AxisHorizontal))
		if w := ts.Size(gluttertest.ByKey(key)).Width; w != 650 {
			t.Errorf("width = %v, want 650", w)
		}
	})
	t.Run("the extent is overridden by a child with a larger size", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		slivers := []Widget{fillSliverBox(), NewSliverFillRemaining(NewContainer(nil, ContainerOpts{Color: foundation.Some(debugRed),
			Height: foundation.Some(600.0), Width: foundation.Some(1000.0)}), SliverFillRemainingOpts{HasScrollBody: noBody})}
		ts.PumpWidget(fillApp(slivers, nil, painting.AxisVertical))
		last := gluttertest.Last(gluttertest.ByType[*Container]())
		if h := ts.Size(last).Height; h != 600 {
			t.Errorf("height = %v, want 600", h)
		}
		ts.PumpWidget(fillApp(slivers, nil, painting.AxisHorizontal))
		if w := ts.Size(last).Width; w != 1000 {
			t.Errorf("width = %v, want 1000", w)
		}
	})
	t.Run("the extent is overridden by the size of the child if precedingScrollExtent > viewportMainAxisExtent", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t, gluttertest.Opts{})
		key := NewGlobalKey[State]()
		button := foundation.NewValueKey("button")
		slivers := []Widget{
			NewSliverFixedExtentListBuilder(func(BuildContext, int) Widget { return NewColoredBox(debugRed, nil) }, 150, SliverBuilderOpts{ItemCount: foundation.Some(5)}),
			NewSliverFillRemaining(NewColoredBox(debugRed, NewAlign(NewPadding(painting.NewEdgeInsetsAll(50),
				NewSizedBox(nil, SizedBoxOpts{Key: button, Width: foundation.Some(116.0), Height: foundation.Some(48.0)}))), ColoredBoxOpts{Key: key}),
				SliverFillRemainingOpts{HasScrollBody: noBody}),
		}
		ts.PumpWidget(fillApp(slivers, nil, painting.AxisVertical))
		ts.Drag(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: -750})
		ts.Pump()
		if h := ts.Size(gluttertest.ByKey(key)).Height; h != 148 {
			t.Errorf("height = %v, want 148", h)
		}
		if ts.BottomRight(gluttertest.ByKey(button)).Dy != 550 || ts.Center(gluttertest.ByKey(button)).Dx != 400 {
			t.Errorf("button at %v", ts.Center(gluttertest.ByKey(button)))
		}
	})
}
