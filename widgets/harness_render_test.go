package widgets

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// This file adds render objects to the test harness: a render tree whose root
// is the render view of the fake engine, and a few widgets that create render
// objects (Flutter's SizedBox, Column and Flexible, and a leaf box).

// viewRootWidget stands for RenderObjectToWidgetAdapter: the widget at the top
// of the tree, whose render object is the render view.
type viewRootWidget struct {
	SingleChildRenderObjectWidget
	view *rendering.RenderView
}

func (w *viewRootWidget) CreateRenderObject(BuildContext) rendering.RenderObject { return w.view }

// viewRootElement stands for RenderObjectToWidgetElement.
type viewRootElement struct {
	RenderTreeRootElement
	child Element
}

func (e *viewRootElement) VisitChildren(v ElementVisitor) {
	if e.child != nil {
		v(e.child)
	}
}

func (e *viewRootElement) ForgetChild(child Element) { e.child = nil }

func (e *viewRootElement) Mount(parent Element, slot any) {
	e.RenderObjectElement.Mount(parent, slot)
	e.child = e.UpdateChild(e.child, e.widget.(*viewRootWidget).Child(), nil)
}

func (e *viewRootElement) Update(w Widget) {
	e.RenderObjectElement.Update(w)
	e.child = e.UpdateChild(e.child, e.widget.(*viewRootWidget).Child(), nil)
}

func (e *viewRootElement) InsertRenderObjectChild(child rendering.RenderObject, _ any) {
	e.renderObject.(singleChildHolder).SetChildObject(child)
}

func (e *viewRootElement) MoveRenderObjectChild(rendering.RenderObject, any, any) {}

func (e *viewRootElement) RemoveRenderObjectChild(rendering.RenderObject, any) {
	e.renderObject.(singleChildHolder).SetChildObject(nil)
}

func (w *viewRootWidget) CreateElement() Element {
	e := &viewRootElement{}
	InitRenderTreeRootElement(&e.RenderTreeRootElement, e, w)
	return e
}

// renderTree is a widget tree whose render objects live in a render view of the
// fake engine.
type renderTree struct {
	t     *testing.T
	stack *renderhost.Stack
	owner *BuildOwner
	root  *viewRootElement
	errs  []error
}

func newRenderTree(t *testing.T) *renderTree {
	t.Helper()
	rt := &renderTree{t: t, stack: renderhost.New()}
	rt.owner = NewBuildOwner(BuildOwnerOpts{OnError: func(err error) { rt.errs = append(rt.errs, err) }})
	return rt
}

// pump replaces the widget of the tree, builds, finalizes and lays out.
func (rt *renderTree) pump(w Widget) {
	rt.t.Helper()
	rw := &viewRootWidget{
		SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(w),
		view:                          rt.stack.RenderView,
	}
	if rt.root == nil {
		rt.root = rw.CreateElement().(*viewRootElement)
		rt.root.AssignOwner(rt.owner)
		rt.owner.BuildScope(rt.root, func() { rt.root.Mount(nil, nil) })
	} else {
		rt.owner.BuildScope(rt.root, func() { rt.root.Update(rw) })
	}
	rt.owner.FinalizeTree()
	rt.stack.FlushPipeline()
}

// pumpFrame builds the dirty elements and lays out.
func (rt *renderTree) pumpFrame() {
	rt.t.Helper()
	rt.owner.BuildScope(rt.root, nil)
	rt.owner.FinalizeTree()
	rt.stack.FlushPipeline()
}

func (rt *renderTree) takeError() error {
	rt.t.Helper()
	return takeError(rt.t, &rt.errs)
}

func (rt *renderTree) find(pred func(Widget) bool) []Element {
	var found []Element
	var visit func(Element)
	visit = func(e Element) {
		if pred(e.AsElement().widget) {
			found = append(found, e)
		}
		e.VisitChildren(visit)
	}
	rt.root.VisitChildren(visit)
	return found
}

func renderElementOfType[W Widget](t *testing.T, rt *renderTree) Element {
	t.Helper()
	es := rt.find(func(w Widget) bool { _, ok := w.(W); return ok })
	if len(es) != 1 {
		t.Fatalf("found %d elements for %T, want 1", len(es), *new(W))
	}
	return es[0]
}

// box is a leaf render object widget: a render box of a given size.
type box struct {
	LeafRenderObjectWidget
	width, height float64
	created       *int
}

func newBox(w, h float64, opts ...WidgetOpts) *box {
	return &box{LeafRenderObjectWidget: NewLeafRenderObjectWidgetBase(opts...), width: w, height: h}
}

func (w *box) CreateRenderObject(BuildContext) rendering.RenderObject {
	if w.created != nil {
		*w.created++
	}
	return rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: w.width, Height: w.height}), nil)
}

func (w *box) UpdateRenderObject(_ BuildContext, ro rendering.RenderObject) {
	ro.(*rendering.RenderConstrainedBox).SetAdditionalConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: w.width, Height: w.height}))
}

// sized is SizedBox: a render object with one child and tight constraints.
type sized struct {
	SingleChildRenderObjectWidget
	width, height float64
	unmounted     *[]rendering.RenderObject
}

func newSized(w, h float64, child Widget, opts ...WidgetOpts) *sized {
	return &sized{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, opts...), width: w, height: h}
}

func (w *sized) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: w.width, Height: w.height}), nil)
}

func (w *sized) UpdateRenderObject(_ BuildContext, ro rendering.RenderObject) {
	ro.(*rendering.RenderConstrainedBox).SetAdditionalConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: w.width, Height: w.height}))
}

func (w *sized) DidUnmountRenderObject(ro rendering.RenderObject) {
	if w.unmounted != nil {
		*w.unmounted = append(*w.unmounted, ro)
	}
}

// column is Column: a vertical flex.
type column struct {
	MultiChildRenderObjectWidget
}

func newColumn(children ...Widget) *column {
	return &column{NewMultiChildRenderObjectWidgetBase(children)}
}

func (w *column) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderFlex(nil, rendering.RenderFlexOpts{Direction: painting.AxisVertical})
}

// center is Center: it gives its child loose constraints.
type center struct{ SingleChildRenderObjectWidget }

func newCenter(child Widget) *center {
	return &center{NewSingleChildRenderObjectWidgetBase(child)}
}

func (w *center) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderPositionedBox(nil)
}
