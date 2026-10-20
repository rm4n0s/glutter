package widgets_test

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// slotBox is a render object with children in slots: it puts them side by side,
// in the order of its slots.
type slotBox struct {
	rendering.RenderBox
	rendering.SlottedContainer
	order []any
}

func newSlotBox(order []any) *slotBox {
	r := &slotBox{order: order}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitSlottedContainer(&r.SlottedContainer, r)
	return r
}

func (r *slotBox) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*rendering.BoxParentData); !ok {
		child.AsRenderObject().SetParentData(&rendering.BoxParentData{})
	}
}

func (r *slotBox) PerformLayout() {
	x, h := 0.0, 0.0
	for _, slot := range r.order {
		child, ok := r.SlotChild(slot).(rendering.Box)
		if !ok {
			continue
		}
		child.Layout(r.BoxConstraints().Loosen(), rendering.LayoutOpts{ParentUsesSize: true})
		child.ParentData().(*rendering.BoxParentData).Offset = ui.Offset{Dx: x}
		x += child.Size().Width
		h = max(h, child.Size().Height)
	}
	r.SetSize(r.BoxConstraints().Constrain(ui.Size{Width: x, Height: h}))
}

func (r *slotBox) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	for _, slot := range r.order {
		if child, ok := r.SlotChild(slot).(rendering.Box); ok {
			context.PaintChild(child, child.ParentData().(*rendering.BoxParentData).Offset.Add(offset))
		}
	}
}

func (r *slotBox) HitTestChildren(*rendering.BoxHitTestResult, ui.Offset) bool { return false }

// slotWidget has a widget for each of its slots.
type slotWidget struct {
	SlottedMultiChildRenderObjectWidget
	slots    []any
	children map[any]Widget
}

func newSlotWidget(slots []any, children map[any]Widget) *slotWidget {
	return &slotWidget{SlottedMultiChildRenderObjectWidget: NewSlottedMultiChildRenderObjectWidgetBase(), slots: slots, children: children}
}

func (w *slotWidget) Slots() []any                 { return w.slots }
func (w *slotWidget) ChildForSlot(slot any) Widget { return w.children[slot] }
func (w *slotWidget) CreateRenderObject(BuildContext) rendering.RenderObject {
	return newSlotBox(w.slots)
}
func (w *slotWidget) UpdateRenderObject(_ BuildContext, ro rendering.RenderObject) {
	ro.(*slotBox).order = w.slots
	ro.MarkNeedsLayout()
}

func slotKeyedBox(key string, width float64) Widget {
	return NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey(key), Width: foundation.Some(width), Height: foundation.Some(10.0)})
}

func slotApp(w Widget) Widget {
	return testWidgetsApp(testWidgetsAppOpts{home: NewAlign(w)})
}

func TestSlottedRenderObjectLaysOutTheChildrenInTheirSlots(t *testing.T) {
	ts := gluttertest.New(t)
	slots := []any{"a", "b"}
	ts.PumpWidget(slotApp(newSlotWidget(slots, map[any]Widget{"a": slotKeyedBox("x", 30), "b": slotKeyedBox("y", 50)})))
	box := ts.RenderObject(gluttertest.ByType[*slotWidget]()).(rendering.Box)
	if got := box.Size(); got != (ui.Size{Width: 80, Height: 10}) {
		t.Errorf("size = %v, want 80x10", got)
	}
	x, y := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("x"))), ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("y")))
	if y.Dx-x.Dx != 30 {
		t.Errorf("the second child is %v after the first, want 30", y.Dx-x.Dx)
	}
	if n := ts.Count(gluttertest.ByType[*SizedBox]()); n < 2 {
		t.Errorf("%d children", n)
	}
}

func TestSlottedRenderObjectKeepsTheElementsWhenTheWidgetsSwapSlots(t *testing.T) {
	ts := gluttertest.New(t)
	slots := []any{"a", "b"}
	ts.PumpWidget(slotApp(newSlotWidget(slots, map[any]Widget{"a": slotKeyedBox("x", 30), "b": slotKeyedBox("y", 50)})))
	x := ts.Element(gluttertest.ByKey(foundation.NewValueKey("x")))
	y := ts.Element(gluttertest.ByKey(foundation.NewValueKey("y")))
	ts.PumpWidget(slotApp(newSlotWidget(slots, map[any]Widget{"a": slotKeyedBox("y", 50), "b": slotKeyedBox("x", 30)})))
	if ts.Element(gluttertest.ByKey(foundation.NewValueKey("x"))) != x || ts.Element(gluttertest.ByKey(foundation.NewValueKey("y"))) != y {
		t.Error("the elements of the keyed children were not kept")
	}
	if got := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("x"))).Dx - ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("y"))).Dx; got != 50 {
		t.Errorf("x is %v after y, want 50: the children swapped slots", got)
	}
}

func TestSlottedRenderObjectUpdatesAndRemovesChildren(t *testing.T) {
	ts := gluttertest.New(t)
	slots := []any{"a", "b"}
	ts.PumpWidget(slotApp(newSlotWidget(slots, map[any]Widget{"a": NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(30.0), Height: foundation.Some(10.0)}), "b": slotKeyedBox("y", 50)})))
	a := ts.Element(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*slotWidget](), gluttertest.ByType[*SizedBox]())))
	// The unkeyed child of a slot is updated, not replaced.
	ts.PumpWidget(slotApp(newSlotWidget(slots, map[any]Widget{"a": NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(40.0), Height: foundation.Some(10.0)}), "b": slotKeyedBox("y", 50)})))
	if ts.Element(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*slotWidget](), gluttertest.ByType[*SizedBox]()))) != a {
		t.Error("the element of the slot was replaced")
	}
	if got := ts.RenderObject(gluttertest.ByType[*slotWidget]()).(rendering.Box).Size().Width; got != 90 {
		t.Errorf("width = %v, want 90", got)
	}
	ts.PumpWidget(slotApp(newSlotWidget(slots, map[any]Widget{"b": slotKeyedBox("y", 50)})))
	if got := ts.RenderObject(gluttertest.ByType[*slotWidget]()).(rendering.Box).Size().Width; got != 50 {
		t.Errorf("width after removing the child of a = %v, want 50", got)
	}
}

func TestSlottedRenderObjectErrors(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("needs the debug-only checks of the framework")
	}
	tests := []struct {
		name  string
		build func(ts *gluttertest.Tester)
		tag   string
	}{
		{"slots that are not unique", func(ts *gluttertest.Tester) {
			ts.PumpWidget(slotApp(newSlotWidget([]any{"a", "a"}, map[any]Widget{"a": slotKeyedBox("x", 10)})))
		}, "SlottedSlotsNotUnique"},
		{"slots that change", func(ts *gluttertest.Tester) {
			ts.PumpWidget(slotApp(newSlotWidget([]any{"a", "b"}, map[any]Widget{"a": slotKeyedBox("x", 10)})))
			ts.PumpWidget(slotApp(newSlotWidget([]any{"a", "c"}, map[any]Widget{"a": slotKeyedBox("x", 10)})))
		}, "SlottedSlotsChanged"},
		{"a key that is used twice", func(ts *gluttertest.Tester) {
			ts.PumpWidget(slotApp(newSlotWidget([]any{"a", "b"}, map[any]Widget{"a": slotKeyedBox("x", 10), "b": slotKeyedBox("x", 10)})))
		}, "SlottedDuplicateKeys"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			tc.build(ts)
			errs := ts.TakeErrors()
			if len(errs) == 0 {
				t.Fatalf("no error, want %s", tc.tag)
			}
			appErr, ok := errors.FromError(errs[0])
			if !ok || !appErr.HasRoute(tc.tag) {
				t.Errorf("error = %v, want the tag %s", errs[0], tc.tag)
			}
		})
	}
}
