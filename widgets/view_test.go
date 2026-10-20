// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
	"testing"

	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/widgets/view_test.dart: the tests of View
// (finding the view, the pipeline owner and the media query from inside, and
// attaching and detaching the render view). ViewCollection, ViewAnchor, the
// LookupBoundary test and the view focus event tests wait for the windows (step
// 14.2).

func TestWidgetsRunWithTheDefaultViewCanFindTheView(t *testing.T) {
	ts := gluttertest.New(t)
	var viewOf, viewMaybeOf *ui.View
	ts.PumpWidget(builderOf(func(bc BuildContext) Widget {
		viewOf = ViewOf(bc)
		viewMaybeOf = MaybeViewOf(bc)
		return newLeaf()
	}))
	if viewOf == nil || viewOf != ts.View || viewMaybeOf != ts.View {
		t.Errorf("ViewOf = %v, MaybeViewOf = %v, want the view %v", viewOf, viewMaybeOf, ts.View)
	}
}

func TestViewOfWithoutView(t *testing.T) {
	ts := gluttertest.New(t)
	var maybe *ui.View
	pumpUnwrapped(ts, builderOf(func(bc BuildContext) Widget {
		maybe = MaybeViewOf(bc)
		ViewOf(bc)
		return newLeaf()
	}))
	if maybe != nil {
		t.Errorf("MaybeViewOf = %v, want nil", maybe)
	}
	_ = requireErrorTag(t, takeFirst(t, ts), "ViewOfWithoutView")
}

func TestChildOfViewFindsViewPipelineOwnerAndMediaQuery(t *testing.T) {
	ts := gluttertest.New(t)
	var outsideView, insideView *ui.View
	var outsideOwner, insideOwner *rendering.PipelineOwner
	var data MediaQueryData
	var hasMediaQuery bool
	pumpUnwrapped(ts, builderOf(func(bc BuildContext) Widget {
		outsideView = MaybeViewOf(bc)
		outsideOwner = PipelineOwnerOf(bc)
		return NewView(ts.View, builderOf(func(bc BuildContext) Widget {
			insideView = MaybeViewOf(bc)
			insideOwner = PipelineOwnerOf(bc)
			data, hasMediaQuery = MediaQueryMaybeOf(bc)
			return newBox(10, 10)
		}))
	}))
	ts.Pump()
	if outsideView != nil || insideView != ts.View {
		t.Errorf("outside view = %v, inside view = %v; want nil and %v", outsideView, insideView, ts.View)
	}
	if outsideOwner != ts.Binding.Renderer().RootPipelineOwner() {
		t.Error("outside the View, PipelineOwnerOf should be the root pipeline owner of the renderer binding")
	}
	if insideOwner == nil || insideOwner == outsideOwner {
		t.Error("inside the View, PipelineOwnerOf should be the owner of the render view")
	}
	if !hasMediaQuery || data.Size != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("inside the View, the media query = %v (found %v), want 800x600", data.Size, hasMediaQuery)
	}
	boxes := ts.Elements(gluttertest.ByPredicate("test widget", func(w Widget) bool { _, ok := w.(*box); return ok }))
	if len(boxes) != 1 || boxes[0].AsElement().RenderObject().Owner() != insideOwner {
		t.Error("the render object of the child does not belong to the pipeline owner it found")
	}
	var owners []*rendering.PipelineOwner
	ts.Binding.Renderer().RootPipelineOwner().VisitChildren(func(c *rendering.PipelineOwner) { owners = append(owners, c) })
	if len(owners) != 1 || owners[0] != insideOwner {
		t.Errorf("the root pipeline owner has %d children, want the one of the view", len(owners))
	}
}

func TestViewAttachesAndDetachesItsRenderView(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(newBox(10, 10))
	views := ts.Binding.Renderer().RenderViews()
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
	pumpUnwrapped(ts, newNothing())
	if n := len(ts.Binding.Renderer().RenderViews()); n != 0 {
		t.Errorf("%d render views after the View left the tree, want 0", n)
	}
	if renderView.Owner() != nil {
		t.Error("the render view still has an owner")
	}
	children := 0
	ts.Binding.Renderer().RootPipelineOwner().VisitChildren(func(*rendering.PipelineOwner) { children++ })
	if children != 0 {
		t.Errorf("the root pipeline owner has %d children, want 0", children)
	}
}

func TestViewRenderViewDrawsTheChild(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(newBox(100, 50))
	if len(ts.Scenes()) != 1 {
		t.Fatalf("%d scenes, want 1", len(ts.Scenes()))
	}
	boxes := ts.Elements(gluttertest.ByPredicate("test widget", func(w Widget) bool { _, ok := w.(*box); return ok }))
	size, ok := boxes[0].Size()
	if !ok || size.Width != 800 || size.Height != 600 {
		// The render view gives tight constraints, which the box obeys.
		t.Errorf("size = %v (%v), want the 800x600 of the view", size, ok)
	}
}

func TestViewMetricsChangeRelayoutsTheChild(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(newBox(10, 10))
	ts.SetViewConfiguration(func(c *ui.ViewConfiguration) { c.PhysicalSize = ui.Size{Width: 400, Height: 300} })
	if !ts.Binding.Scheduler().HasScheduledFrame() {
		t.Fatal("a metrics change did not schedule a frame")
	}
	ts.Pump()
	boxes := ts.Elements(gluttertest.ByPredicate("test widget", func(w Widget) bool { _, ok := w.(*box); return ok }))
	if size, _ := boxes[0].Size(); size.Width != 400 || size.Height != 300 {
		t.Errorf("size = %v, want 400x300", size)
	}
}
