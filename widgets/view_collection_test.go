// Ported from packages/flutter/test/widgets/view_test.dart: the ViewCollection and
// ViewAnchor tests, the render object getter group, and the view focus tests. The
// FakeView of Flutter's tests is a view that the window controllers make.
package widgets_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// extraView makes a window, whose view stands in for Flutter's FakeView.
func extraView(t *testing.T, ts *gluttertest.Tester) *ui.View {
	t.Helper()
	c, err := NewRegularWindowController(ts.Element(gluttertest.ByType[*Directionality]()), ui.Size{Width: 200, Height: 100})
	if err != nil {
		t.Fatal(err)
	}
	return c.RootView()
}

// viewTester is a tester with a first widget, so that there is a context to make
// windows from.
func viewTester(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t)
	ts.PumpWidget(text("start"))
	return ts
}

func TestViewCollectionMayStartWithZeroViews(t *testing.T) {
	ts := viewTester(t)
	ts.PumpWidget(NewViewCollection(nil), gluttertest.PumpWidgetOpts{NoView: true})
	if errs := ts.Errors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

func TestViewAnchorChildDoesNotSeeTheSurroundingView(t *testing.T) {
	ts := viewTester(t)
	other := extraView(t, ts)
	var inside, outside *ui.View
	ts.PumpWidget(NewBuilder(func(bc BuildContext) Widget {
		outside = MaybeViewOf(bc)
		return NewViewAnchor(NewSizedBox(nil), ViewAnchorOpts{View: NewBuilder(func(bc BuildContext) Widget {
			inside = MaybeViewOf(bc)
			return NewView(other, NewSizedBox(nil))
		})})
	}))
	if inside != nil {
		t.Errorf("the view slot sees the view %d", inside.ViewID())
	}
	if outside == nil {
		t.Error("the surrounding view is not there")
	}
}

// spyBox is a render box that logs when it lays out.
type spyBox struct {
	rendering.RenderConstrainedBox
	label int
	log   *[]string
}

func (b *spyBox) PerformLayout() {
	*b.log = append(*b.log, fmt.Sprintf("layout %d", b.label))
	b.RenderConstrainedBox.PerformLayout()
}

// spyWidget is a SizedBox with a spy render object.
type spyWidget struct {
	SingleChildRenderObjectWidget
	label int
	log   *[]string
}

func newSpy(label int, log *[]string, child Widget) *spyWidget {
	return &spyWidget{NewSingleChildRenderObjectWidgetBase(child), label, log}
}

func (w *spyWidget) CreateRenderObject(BuildContext) rendering.RenderObject {
	b := &spyBox{label: w.label, log: w.log}
	rendering.InitRenderConstrainedBox(&b.RenderConstrainedBox, b, rendering.NewBoxConstraints(), nil)
	return b
}

func TestViewAnchorLayoutOrder(t *testing.T) {
	ts := viewTester(t)
	other := extraView(t, ts)
	var log []string
	ts.PumpWidget(newSpy(1, &log, NewViewAnchor(newSpy(3, &log, nil),
		ViewAnchorOpts{View: NewView(other, newSpy(2, &log, nil))})))
	log = nil
	spy := func(label int) rendering.RenderObject {
		return ts.RenderObject(gluttertest.ByPredicate("a spy", func(w Widget) bool {
			s, ok := w.(*spyWidget)
			return ok && s.label == label
		}))
	}
	spy(3).MarkNeedsLayout()
	spy(2).MarkNeedsLayout()
	spy(1).MarkNeedsLayout()
	ts.Pump()
	if want := []string{"layout 1", "layout 3", "layout 2"}; !slices.Equal(log, want) {
		t.Errorf("layout order = %q, want %q", log, want)
	}
}

// componentElements are the elements of the widget behind ViewCollection and
// ViewAnchor.
func componentElements(ts *gluttertest.Tester) []Element {
	return ts.Elements(gluttertest.ByElementPredicate("a view component", func(e Element) bool {
		name := fmt.Sprintf("%T", e.Widget())
		return name == "*widgets.multiChildComponent" || name == "*widgets.ViewCollection"
	}))
}

func visited(e Element) int {
	n := 0
	e.VisitChildren(func(Element) { n++ })
	return n
}

func TestVisitChildrenOfViewAnchorVisitsBothChildren(t *testing.T) {
	ts := viewTester(t)
	other := extraView(t, ts)
	ts.PumpWidget(NewViewAnchor(NewSizedBox(nil), ViewAnchorOpts{View: NewView(other, NewSizedBox(nil))}))
	anchor := componentElements(ts)
	if len(anchor) != 1 || visited(anchor[0]) != 2 {
		t.Fatalf("components %d, children %d, want one with two", len(anchor), visited(anchor[0]))
	}
	ts.PumpWidget(NewViewAnchor(NewSizedBox(nil)))
	if got := visited(anchor[0]); got != 1 {
		t.Errorf("children = %d after the view left, want 1", got)
	}
}

func TestVisitChildrenOfViewCollectionVisitsAllChildren(t *testing.T) {
	ts := viewTester(t)
	v2, v3 := extraView(t, ts), extraView(t, ts)
	ts.PumpWidget(NewViewCollection([]Widget{
		NewView(ts.View, NewSizedBox(nil)), NewView(v2, NewSizedBox(nil)), NewView(v3, NewSizedBox(nil)),
	}), gluttertest.PumpWidgetOpts{NoView: true})
	collection := componentElements(ts)
	if len(collection) != 1 || visited(collection[0]) != 3 {
		t.Fatalf("components %d, want one with three children", len(collection))
	}
	ts.PumpWidget(NewViewCollection([]Widget{NewView(ts.View, NewSizedBox(nil))}), gluttertest.PumpWidgetOpts{NoView: true})
	if got := visited(collection[0]); got != 1 {
		t.Errorf("children = %d, want 1", got)
	}
}

func TestRenderObjectGetter(t *testing.T) {
	t.Run("ancestors of a view see the RenderView as renderObject", func(t *testing.T) {
		ts := viewTester(t)
		var builderContext BuildContext
		ts.PumpWidget(NewBuilder(func(bc BuildContext) Widget {
			builderContext = bc
			return NewView(ts.View, NewSizedBox(nil))
		}), gluttertest.PumpWidgetOpts{NoView: true})
		ro := builderContext.(Element).FindRenderObject()
		if _, ok := ro.(*rendering.RenderView); !ok {
			t.Fatalf("render object = %T, want a RenderView", ro)
		}
		if ro != ts.RenderObject(gluttertest.ByType[*View]()) {
			t.Error("it is not the render view of the View")
		}
		if got := builderContext.(Element).AsElement().RenderObject(); got != ro {
			t.Errorf("the element's renderObject = %v", got)
		}
	})
	t.Run("ancestors of a ViewCollection get nil for renderObject", func(t *testing.T) {
		ts := viewTester(t)
		other := extraView(t, ts)
		var builderContext BuildContext
		ts.PumpWidget(NewBuilder(func(bc BuildContext) Widget {
			builderContext = bc
			return NewViewCollection([]Widget{NewView(ts.View, NewSizedBox(nil)), NewView(other, NewSizedBox(nil))})
		}), gluttertest.PumpWidgetOpts{NoView: true})
		if ro := builderContext.(Element).FindRenderObject(); ro != nil {
			t.Errorf("render object = %v, want nil", ro)
		}
		if ro := builderContext.(Element).AsElement().RenderObject(); ro != nil {
			t.Errorf("the element's renderObject = %v, want nil", ro)
		}
	})
	t.Run("ancestors of a ViewAnchor see the right render object", func(t *testing.T) {
		ts := viewTester(t)
		other := extraView(t, ts)
		var builderContext BuildContext
		ts.PumpWidget(NewBuilder(func(bc BuildContext) Widget {
			builderContext = bc
			return NewViewAnchor(NewSizedBox(nil), ViewAnchorOpts{View: NewView(other, NewSizedBox(nil))})
		}))
		ro := builderContext.(Element).FindRenderObject()
		if _, ok := ro.(*rendering.RenderConstrainedBox); !ok {
			t.Fatalf("render object = %T, want a RenderConstrainedBox", ro)
		}
		if ro != ts.RenderObject(gluttertest.First(gluttertest.ByType[*SizedBox]())) {
			t.Error("it is not the render object of the anchored SizedBox")
		}
	})
}

func TestViewFocusEventsCauseUnfocusingAndRefocusing(t *testing.T) {
	ts := gluttertest.New(t)
	node := NewFocusNode()
	t.Cleanup(node.Dispose)
	ts.PumpWidget(NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: node}))
	root := ts.Binding.BuildOwner().FocusManager().RootScope()

	node.RequestFocus()
	ts.Pump()
	if !node.HasPrimaryFocus() || root.HasPrimaryFocus() {
		t.Fatalf("the node has the primary focus: %v, the root scope: %v", node.HasPrimaryFocus(), root.HasPrimaryFocus())
	}
	ts.SetViewFocus(ui.ImplicitViewID, false, ui.ViewFocusDirectionForward)
	ts.Pump()
	if node.HasPrimaryFocus() || !root.HasPrimaryFocus() {
		t.Errorf("after the unfocus event: node %v, root scope %v", node.HasPrimaryFocus(), root.HasPrimaryFocus())
	}
	ts.SetViewFocus(ui.ImplicitViewID, true, ui.ViewFocusDirectionBackward)
	ts.Pump()
	if !node.HasPrimaryFocus() || root.HasPrimaryFocus() {
		t.Errorf("after the focus event: node %v, root scope %v", node.HasPrimaryFocus(), root.HasPrimaryFocus())
	}
}

func viewWithNode(t *testing.T) (*gluttertest.Tester, *FocusNode) {
	t.Helper()
	ts := gluttertest.New(t)
	node := NewFocusNode()
	t.Cleanup(node.Dispose)
	ts.PumpWidget(NewDirectionality(ui.TextDirectionRTL, NewColumn([]Widget{
		NewFocus(NewText("a"), FocusOpts{FocusNode: node}),
	})))
	ts.ResetFocusEvents()
	return ts, node
}

func TestViewNotifiesTheEngineWhenAWidgetFocusChanges(t *testing.T) {
	ts, node := viewWithNode(t)
	notified := 0
	handle := ts.Binding.BuildOwner().FocusManager().AddListener(func() { notified++ })
	t.Cleanup(func() { ts.Binding.BuildOwner().FocusManager().RemoveListener(handle) })

	node.RequestFocus()
	ts.Pump()
	want := []ui.ViewFocusEvent{{ViewID: ui.ImplicitViewID, State: ui.ViewFocusStateFocused, Direction: ui.ViewFocusDirectionForward}}
	if got := ts.FocusEvents(); !slices.Equal(got, want) {
		t.Errorf("focus events = %+v, want %+v", got, want)
	}
	if !node.HasPrimaryFocus() || notified != 1 {
		t.Errorf("primary focus %v, notified %d times, want true and once", node.HasPrimaryFocus(), notified)
	}
}

func TestSwitchingFocusBetweenViewsYieldsTheCorrectEvents(t *testing.T) {
	ts, node := viewWithNode(t)
	notified := 0
	handle := ts.Binding.BuildOwner().FocusManager().AddListener(func() { notified++ })
	t.Cleanup(func() { ts.Binding.BuildOwner().FocusManager().RemoveListener(handle) })
	focusEvent := ui.ViewFocusEvent{ViewID: ui.ImplicitViewID, State: ui.ViewFocusStateFocused, Direction: ui.ViewFocusDirectionForward}

	// Focus, and the engine is told.
	node.RequestFocus()
	ts.Pump()
	if got := ts.FocusEvents(); !slices.Equal(got, []ui.ViewFocusEvent{focusEvent}) || !node.HasPrimaryFocus() || notified != 1 {
		t.Fatalf("events %+v, primary %v, notified %d", got, node.HasPrimaryFocus(), notified)
	}
	notified = 0
	ts.ResetFocusEvents()

	// Unfocus all the views.
	ts.SetViewFocus(ui.ImplicitViewID, false, ui.ViewFocusDirectionForward)
	ts.Pump()
	if node.HasFocus() || len(ts.FocusEvents()) != 0 || notified != 1 {
		t.Errorf("after the unfocus: focus %v, events %v, notified %d", node.HasFocus(), ts.FocusEvents(), notified)
	}
	notified = 0
	ts.ResetFocusEvents()

	// Focus another view: this node stays unfocused, and the engine is not told.
	ts.SetViewFocus(100, true, ui.ViewFocusDirectionForward)
	ts.Pump()
	if node.HasFocus() || len(ts.FocusEvents()) != 0 || notified != 0 {
		t.Errorf("after another view got the focus: focus %v, events %v, notified %d", node.HasFocus(), ts.FocusEvents(), notified)
	}

	// Focusing the node again tells the engine that this view has the focus.
	node.RequestFocus()
	ts.Pump()
	if got := ts.FocusEvents(); !slices.Equal(got, []ui.ViewFocusEvent{focusEvent}) || !node.HasPrimaryFocus() || notified != 1 {
		t.Errorf("after refocusing: events %+v, primary %v, notified %d", got, node.HasPrimaryFocus(), notified)
	}
}

func TestANestedViewDoesNotMakeItsParentRequestFocus(t *testing.T) {
	ts := viewTester(t)
	childNode, parentNode := NewFocusNode(), NewFocusNode()
	t.Cleanup(func() { childNode.Dispose(); parentNode.Dispose() })
	child := extraView(t, ts)
	ts.PumpWidget(NewViewAnchor(NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: parentNode}),
		ViewAnchorOpts{View: NewView(child, NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: childNode}))}))
	ts.ResetFocusEvents()

	// Only the nested view is asked to take the focus; the surrounding one must
	// not: a native window would be pulled to the front.
	childNode.RequestFocus()
	ts.Pump()
	if !childNode.HasPrimaryFocus() {
		t.Fatal("the child node has no focus")
	}
	for _, e := range ts.FocusEvents() {
		if e.ViewID != child.ViewID() {
			t.Errorf("view %d was asked for the focus, want only the nested view %d", e.ViewID, child.ViewID())
		}
	}
	ts.ResetFocusEvents()

	// Moving the focus to the surrounding view asks that view.
	parentNode.RequestFocus()
	ts.Pump()
	if !parentNode.HasPrimaryFocus() {
		t.Fatal("the parent node has no focus")
	}
	asked := map[ui.ViewID]bool{}
	for _, e := range ts.FocusEvents() {
		asked[e.ViewID] = true
	}
	if !asked[ui.ImplicitViewID] || asked[child.ViewID()] {
		t.Errorf("views asked for the focus = %v, want the surrounding view only", asked)
	}
}
