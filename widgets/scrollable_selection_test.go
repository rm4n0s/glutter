package widgets_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/scrollable_selection_test.dart.

func selectableList(controller ScrollController) Widget {
	return NewListViewBuilder(func(_ BuildContext, i int) Widget {
		return ltrText(fmt.Sprintf("Item %d", i))
	}, ListViewOpts{ItemCount: foundation.Some(100), Controller: controller})
}

func itemParagraph(ts *gluttertest.Tester, i int) *rendering.RenderParagraph {
	return paragraphOf(ts, fmt.Sprintf("Item %d", i))
}

func TestScrollableSelectionMouseCanSelectMultipleWidgets(t *testing.T) {
	ts := gluttertest.New(t)
	pumpSelectable(ts, selectableList(nil))
	ts.PumpAndSettle()
	p1 := itemParagraph(ts, 0)
	g := ts.StartGesture(paragraphOffsetToPosition(p1, 2), mouseKind)
	ts.Pump()
	g.MoveTo(paragraphOffsetToPosition(p1, 4))
	ts.Pump()
	requireSelection(t, p1, sel(2, 4))
	p2 := itemParagraph(ts, 1)
	g.MoveTo(paragraphOffsetToPosition(p2, 5))
	// Should select the rest of paragraph 1.
	requireSelection(t, p1, sel(2, 6))
	requireSelection(t, p2, sel(0, 5))
	p3 := itemParagraph(ts, 3)
	g.MoveTo(paragraphOffsetToPosition(p3, 3))
	requireSelection(t, p1, sel(2, 6))
	requireSelection(t, p2, sel(0, 6))
	requireSelection(t, p3, sel(0, 3))
	g.Up()
}

func TestScrollableSelectionSelectToScrollForward(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewScrollController()
	t.Cleanup(controller.Dispose)
	pumpSelectable(ts, selectableList(controller))
	ts.PumpAndSettle()
	p1 := itemParagraph(ts, 0)
	g := ts.StartGesture(paragraphOffsetToPosition(p1, 2), mouseKind)
	ts.Pump()
	if controller.Offset() != 0 {
		t.Fatalf("offset = %v, want 0", controller.Offset())
	}
	previous := controller.Offset()
	// The scrollable only auto scrolls if the drag passes the boundary.
	g.MoveTo(ts.BottomRight(gluttertest.ByType[*ListView]()).Add(ui.Offset{Dy: 20}))
	ts.Pump()
	ts.Pump(time.Second)
	if controller.Offset() <= previous {
		t.Fatalf("offset = %v, did not scroll past %v", controller.Offset(), previous)
	}
	previous = controller.Offset()
	ts.Pump()
	ts.Pump(time.Second)
	if controller.Offset() <= previous {
		t.Fatalf("offset = %v, stopped scrolling at %v", controller.Offset(), previous)
	}
	// Scroll to the end.
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: time.Second})
	maxExtent := controller.Position().MaxScrollExtent()
	if controller.Offset() != maxExtent {
		t.Fatalf("offset = %v, want the end %v", controller.Offset(), maxExtent)
	}
	for i := 99; i >= 96; i-- {
		requireSelection(t, itemParagraph(ts, i), sel(0, 7))
	}
	g.Up()
}

func TestScrollableSelectionPreservesTheSelectionWhenOutOfView(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewScrollController()
	t.Cleanup(controller.Dispose)
	pumpSelectable(ts, selectableList(controller))
	ts.PumpAndSettle()
	// The Dart test jumps to 2000 where the items are 40 tall: Item 50 is in view.
	itemHeight := ts.Size(gluttertest.ByText("Item 0")).Height
	near, far := 50*itemHeight-100, 90*itemHeight
	controller.JumpTo(near)
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Item 50")) {
		t.Fatal("Item 50 is not in view")
	}
	p50 := itemParagraph(ts, 50)
	g := ts.StartGesture(paragraphOffsetToPosition(p50, 2), mouseKind)
	ts.Pump()
	g.MoveTo(paragraphOffsetToPosition(p50, 4))
	g.Up()
	requireSelection(t, p50, sel(2, 4))

	controller.JumpTo(0)
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("Item 50")) {
		t.Fatal("Item 50 is still built")
	}
	for _, offset := range []float64{near, far, near} {
		controller.JumpTo(offset)
		ts.PumpAndSettle()
		if offset == near {
			requireSelection(t, itemParagraph(ts, 50), sel(2, 4))
		} else if ts.Exists(gluttertest.ByText("Item 50")) {
			t.Fatal("Item 50 is built far from the view")
		}
	}
}
