package widgets

import (
	"testing"

	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/widgets/view_test.dart: the tests of View
// (finding the view, the pipeline owner and the media query from inside, and
// attaching and detaching the render view). ViewCollection, ViewAnchor, the
// LookupBoundary test and the focus tests wait for the windows (step 14.2) and
// the focus system (step 10.5).

func TestWidgetsRunWithTheDefaultViewCanFindTheView(t *testing.T) {
	bt := newBindingTree(t)
	var viewOf, viewMaybeOf *ui.View
	bt.pump(builderOf(func(bc BuildContext) Widget {
		viewOf = ViewOf(bc)
		viewMaybeOf = MaybeViewOf(bc)
		return newLeaf()
	}))
	if viewOf == nil || viewOf != bt.view || viewMaybeOf != bt.view {
		t.Errorf("ViewOf = %v, MaybeViewOf = %v, want the view %v", viewOf, viewMaybeOf, bt.view)
	}
}

func TestViewOfWithoutView(t *testing.T) {
	tr := newTree(t)
	tr.collectPanics = true
	var maybe *ui.View
	tr.pump(builderOf(func(bc BuildContext) Widget {
		maybe = MaybeViewOf(bc)
		ViewOf(bc)
		return newLeaf()
	}))
	if maybe != nil {
		t.Errorf("MaybeViewOf = %v, want nil", maybe)
	}
	_ = requireErrorTag(t, tr.errs[0], "ViewOfWithoutView")
}

func TestChildOfViewFindsViewPipelineOwnerAndMediaQuery(t *testing.T) {
	bt := newBindingTree(t)
	var outsideView, insideView *ui.View
	var outsideOwner, insideOwner *rendering.PipelineOwner
	var data MediaQueryData
	var hasMediaQuery bool
	bt.attachUnwrapped(builderOf(func(bc BuildContext) Widget {
		outsideView = MaybeViewOf(bc)
		outsideOwner = PipelineOwnerOf(bc)
		return NewView(bt.view, builderOf(func(bc BuildContext) Widget {
			insideView = MaybeViewOf(bc)
			insideOwner = PipelineOwnerOf(bc)
			data, hasMediaQuery = MediaQueryMaybeOf(bc)
			return newBox(10, 10)
		}))
	}))
	bt.frame()
	if outsideView != nil || insideView != bt.view {
		t.Errorf("outside view = %v, inside view = %v; want nil and %v", outsideView, insideView, bt.view)
	}
	if outsideOwner != bt.b.Renderer().RootPipelineOwner() {
		t.Error("outside the View, PipelineOwnerOf should be the root pipeline owner of the renderer binding")
	}
	if insideOwner == nil || insideOwner == outsideOwner {
		t.Error("inside the View, PipelineOwnerOf should be the owner of the render view")
	}
	if !hasMediaQuery || data.Size != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("inside the View, the media query = %v (found %v), want 800x600", data.Size, hasMediaQuery)
	}
	boxes := bt.find(func(w Widget) bool { _, ok := w.(*box); return ok })
	if len(boxes) != 1 || boxes[0].AsElement().RenderObject().Owner() != insideOwner {
		t.Error("the render object of the child does not belong to the pipeline owner it found")
	}
	var owners []*rendering.PipelineOwner
	bt.b.Renderer().RootPipelineOwner().VisitChildren(func(c *rendering.PipelineOwner) { owners = append(owners, c) })
	if len(owners) != 1 || owners[0] != insideOwner {
		t.Errorf("the root pipeline owner has %d children, want the one of the view", len(owners))
	}
}

func TestViewAttachesAndDetachesItsRenderView(t *testing.T) {
	bt := newBindingTree(t)
	bt.pump(newBox(10, 10))
	views := bt.b.Renderer().RenderViews()
	if len(views) != 1 {
		t.Fatalf("%d render views after attaching, want 1", len(views))
	}
	renderView := views[0]
	if renderView.Owner() == nil || renderView.Owner().RootNode() != rendering.RenderObject(renderView) {
		t.Error("the render view is not the root node of its pipeline owner")
	}
	if renderView.Child() == nil {
		t.Error("the render view has no child")
	}
	// Remove the View from the tree.
	bt.b.AttachRootWidget(newLeaf())
	bt.frame()
	if n := len(bt.b.Renderer().RenderViews()); n != 0 {
		t.Errorf("%d render views after the View left the tree, want 0", n)
	}
	if renderView.Owner() != nil {
		t.Error("the render view still has an owner")
	}
	children := 0
	bt.b.Renderer().RootPipelineOwner().VisitChildren(func(*rendering.PipelineOwner) { children++ })
	if children != 0 {
		t.Errorf("the root pipeline owner has %d children, want 0", children)
	}
}

func TestViewRenderViewDrawsTheChild(t *testing.T) {
	bt := newBindingTree(t)
	bt.pump(newBox(100, 50))
	if len(bt.host.Scenes()) != 1 {
		t.Fatalf("%d scenes, want 1", len(bt.host.Scenes()))
	}
	boxes := bt.find(func(w Widget) bool { _, ok := w.(*box); return ok })
	size, ok := boxes[0].Size()
	if !ok || size.Width != 800 || size.Height != 600 {
		// The render view gives tight constraints, which the box obeys.
		t.Errorf("size = %v (%v), want the 800x600 of the view", size, ok)
	}
}

func TestViewMetricsChangeRelayoutsTheChild(t *testing.T) {
	bt := newBindingTree(t)
	bt.pump(newBox(10, 10))
	bt.setView(func(c *ui.ViewConfiguration) { c.PhysicalSize = ui.Size{Width: 400, Height: 300} })
	if !bt.b.Scheduler().HasScheduledFrame() {
		t.Fatal("a metrics change did not schedule a frame")
	}
	bt.frame()
	boxes := bt.find(func(w Widget) bool { _, ok := w.(*box); return ok })
	if size, _ := boxes[0].Size(); size.Width != 400 || size.Height != 300 {
		t.Errorf("size = %v, want 400x300", size)
	}
}

func TestViewInternalChecks(t *testing.T) {
	skipUnlessDebug(t)
	bt := newBindingTree(t)
	bt.pump(newBox(10, 10))
	raw := bt.find(func(w Widget) bool { _, ok := w.(*rawViewInternal); return ok })
	if len(raw) != 1 {
		t.Fatalf("%d raw views, want 1", len(raw))
	}
	requireTag(t, "RawViewChildMoved", func() { raw[0].(*rawViewElement).MoveRenderObjectChild(nil, nil, nil) })

	// A binding whose dispatcher has no implicit view cannot wrap the app.
	noView := newBindingTree(t)
	noView.d.RemoveView(ui.ImplicitViewID)
	requireTag(t, "ImplicitViewMissing", func() { noView.b.WrapWithDefaultView(newLeaf()) })
}
