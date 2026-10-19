package widgets_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Route tests of the tags of the failures of step 9.3: each case triggers the
// failure through the public API and asserts the tag that ends the route.

func TestAnimatedScrollViewFailures(t *testing.T) {
	skipUnlessDebug(t)
	item := func(BuildContext, int, animation.Animation[float64]) Widget {
		return NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(10.0)})
	}
	removed := func(BuildContext, animation.Animation[float64]) Widget {
		return NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(10.0)})
	}
	t.Run("of without a list", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(ltr(NewBuilder(func(bc BuildContext) Widget {
			requireTag(t, "AnimatedGridMissing", func() { AnimatedGridOf(bc) })
			requireTag(t, "SliverAnimatedGridMissing", func() { SliverAnimatedGridOf(bc) })
			requireTag(t, "SliverAnimatedListMissing", func() { SliverAnimatedListOf(bc) })
			requireTag(t, "AnimatedListMissing", func() { AnimatedListOf(bc) })
			return NewSizedBox(nil)
		})))
	})
	t.Run("negative initial count", func(t *testing.T) {
		requireTag(t, "AnimatedItemCountNegative", func() { NewAnimatedList(item, AnimatedScrollViewOpts{InitialItemCount: -1}) })
		requireTag(t, "AnimatedItemCountNegative", func() { NewSliverAnimatedList(item, SliverAnimatedListOpts{InitialItemCount: -1}) })
	})
	t.Run("separator removal builder is required", func(t *testing.T) {
		requireTag(t, "AnimatedRemovedSeparatorBuilderNil", func() { NewAnimatedListSeparated(item, item, nil) })
	})
	t.Run("not built", func(t *testing.T) {
		state := NewAnimatedList(item).CreateState().(*AnimatedListState)
		requireTag(t, "AnimatedScrollViewNotBuilt", func() { state.InsertItem(0) })
	})
	t.Run("index and double removal", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		key := NewGlobalKey[*AnimatedListState]()
		ts.PumpWidget(ltr(NewAnimatedList(item, AnimatedScrollViewOpts{Key: key, InitialItemCount: 2})))
		state, _ := key.CurrentState()
		requireTag(t, "AnimatedItemIndexInvalid", func() { state.InsertItem(5) })
		requireTag(t, "AnimatedItemIndexInvalid", func() { state.RemoveItem(5, removed) })
		state.RemoveItem(0, removed)
	})
}

func TestDragBoundaryFailures(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	key := NewGlobalKey[State]()
	ts.PumpWidget(dragBoundaryApp(key, true))
	boundary := DragBoundaryForRectOf(key.CurrentContext())
	requireTag(t, "DragBoundaryTooSmall", func() { boundary.NearestPositionWithinBoundary(ui.NewRectFromLTWH(0, 0, 200, 200)) })
	// The boundary has no size before the layout.
	ts.PumpWidget(ltr(NewDragBoundary(NewBuilder(func(bc BuildContext) Widget {
		DragBoundaryForRectOf(bc)
		return NewSizedBox(nil)
	}))))
	_ = requireErrorTag(t, takeFirst(t, ts), "DragBoundaryUnavailable")
}

func TestListWheelFailures(t *testing.T) {
	skipUnlessDebug(t)
	child := []Widget{NewSizedBox(nil)}
	t.Run("viewport arguments", func(t *testing.T) {
		tests := []ListWheelScrollViewOpts{
			{DiameterRatio: foundation.Some(0.0)},
			{Perspective: foundation.Some(0.5)},
			{Magnification: foundation.Some(0.0)},
			{OverAndUnderCenterOpacity: foundation.Some(2.0)},
			{Squeeze: foundation.Some(0.0)},
			{RenderChildrenOutsideViewport: true},
		}
		for _, o := range tests {
			requireTag(t, "ListWheelViewportInvalid", func() { NewListWheelScrollView(10, child, o) })
		}
		requireTag(t, "ListWheelViewportInvalid", func() { NewListWheelScrollView(0, child) })
	})
	t.Run("controller", func(t *testing.T) {
		controller := NewFixedExtentScrollController()
		defer controller.Dispose()
		requireTag(t, "FixedExtentControllerNotAttached", func() { controller.SelectedItem() })
	})
	t.Run("controller on a plain list", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewFixedExtentScrollController()
		defer controller.Dispose()
		ts.PumpWidget(ltr(NewListView(child, ListViewOpts{Controller: controller})))
		_ = requireErrorTag(t, takeFirst(t, ts), "FixedExtentControllerWrongView")
	})
	t.Run("controller on two views", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewFixedExtentScrollController()
		defer controller.Dispose()
		wheel := func() Widget {
			return NewListWheelScrollView(10, child, ListWheelScrollViewOpts{Controller: controller})
		}
		ts.PumpWidget(ltr(NewColumn([]Widget{NewSizedBox(wheel(), SizedBoxOpts{Height: foundation.Some(100.0)}), NewSizedBox(wheel(), SizedBoxOpts{Height: foundation.Some(100.0)})})))
		requireTag(t, "FixedExtentControllerMultipleViews", func() { controller.SelectedItem() })
	})
}

func TestInteractiveViewerFailures(t *testing.T) {
	skipUnlessDebug(t)
	child := NewSizedBox(nil)
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"minScale zero", "InteractiveViewerScaleInvalid", func() { NewInteractiveViewer(child, InteractiveViewerOpts{MinScale: foundation.Some(0.0)}) }},
		{"maxScale zero", "InteractiveViewerScaleInvalid", func() { NewInteractiveViewer(child, InteractiveViewerOpts{MaxScale: foundation.Some(0.0)}) }},
		{"maxScale below minScale", "InteractiveViewerScaleInvalid", func() {
			NewInteractiveViewer(child, InteractiveViewerOpts{MinScale: foundation.Some(2.0), MaxScale: foundation.Some(1.0)})
		}},
		{"friction zero", "InteractiveViewerFrictionInvalid", func() {
			NewInteractiveViewer(child, InteractiveViewerOpts{InteractionEndFrictionCoefficient: foundation.Some(0.0)})
		}},
		{"boundary margin", "InteractiveViewerBoundaryMarginInvalid", func() {
			NewInteractiveViewer(child, InteractiveViewerOpts{BoundaryMargin: painting.EdgeInsets{Left: math.Inf(1)}})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { requireTag(t, tc.tag, tc.call) })
	}
	t.Run("a builder viewer can not be constrained", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(NewInteractiveViewerBuilder(nil, InteractiveViewerOpts{}))
		_ = requireErrorTag(t, takeFirst(t, ts), "InteractiveViewerBuilderInvalid")
	})
}

type noWantLeaf struct {
	StatefulWidget
}

func (w *noWantLeaf) CreateState() State { return &noWantLeafState{} }

// noWantLeafState embeds AutomaticKeepAliveClient but has no WantKeepAlive method.
type noWantLeafState struct {
	StateOf[*noWantLeaf]
	AutomaticKeepAliveClient
}

func (s *noWantLeafState) Build(BuildContext) Widget { return NewSizedBox(nil) }

func TestKeepAliveFailures(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("the same handle twice", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		handle := NewKeepAliveHandle()
		defer handle.Dispose()
		ts.PumpWidget(ltr(NewListView([]Widget{NewAutomaticKeepAlive(NewBuilder(func(bc BuildContext) Widget {
			DispatchNotification(bc, KeepAliveNotification{Handle: handle})
			DispatchNotification(bc, KeepAliveNotification{Handle: handle})
			return NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(10.0)})
		}))}, ListViewOpts{AddAutomaticKeepAlives: foundation.Some(false)})))
		_ = requireErrorTag(t, takeFirst(t, ts), "KeepAliveHandleAdded")
	})
	t.Run("a state without WantKeepAlive", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(ltr(&noWantLeaf{NewStatefulWidgetBase()}))
		_ = requireErrorTag(t, takeFirst(t, ts), "WantKeepAliveMissing")
	})
}

func TestListViewCountFailures(t *testing.T) {
	skipUnlessDebug(t)
	builder := func(BuildContext, int) Widget { return NewSizedBox(nil) }
	requireTag(t, "ListViewItemCountInvalid", func() { NewListViewBuilder(builder, ListViewOpts{ItemCount: foundation.Some(-1)}) })
	requireTag(t, "ListViewItemCountInvalid", func() { NewListViewSeparated(builder, builder, -1) })
	requireTag(t, "ListViewSemanticChildCountInvalid", func() {
		NewListViewBuilder(builder, ListViewOpts{ItemCount: foundation.Some(2), SemanticChildCount: foundation.Some(3)})
	})
}

func TestScrollViewArgumentFailures(t *testing.T) {
	skipUnlessDebug(t)
	controller := NewScrollController()
	defer controller.Dispose()
	tests := []struct {
		name string
		tag  string
		opts ScrollViewOpts
	}{
		{"primary with a controller", "ScrollViewPrimaryWithController", ScrollViewOpts{Primary: foundation.Some(true), Controller: controller}},
		{"center with shrink wrap", "ScrollViewCenterShrinkWrap", ScrollViewOpts{ShrinkWrap: true, Center: foundation.NewValueKey("center")}},
		{"anchor out of range", "ScrollViewAnchorInvalid", ScrollViewOpts{Anchor: 2}},
		{"negative semantic child count", "ScrollViewSemanticChildCountInvalid", ScrollViewOpts{SemanticChildCount: foundation.Some(-1)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { requireTag(t, tc.tag, func() { NewCustomScrollView(nil, tc.opts) }) })
	}
}

func TestSeparatorBuilderMustReturnAWidget(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewListViewSeparated(func(BuildContext, int) Widget { return NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(10.0)}) },
		func(BuildContext, int) Widget { return nil }, 3)))
	_ = requireErrorTag(t, takeFirst(t, ts), "SeparatorBuilderNil")
}

func TestReorderableListFailures(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("of without a list", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(ltr(NewBuilder(func(bc BuildContext) Widget {
			requireTag(t, "ReorderableListMissing", func() { ReorderableListOf(bc) })
			requireTag(t, "SliverReorderableListMissing", func() { SliverReorderableListOf(bc) })
			return NewSizedBox(nil)
		})))
	})
	t.Run("items without a key", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(reorderableApp(NewReorderableList(func(BuildContext, int) Widget { return NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(10.0)}) },
			2, ReorderableListOpts{OnReorderItem: func(int, int) {}})))
		_ = requireErrorTag(t, takeFirst(t, ts), "ReorderableItemKeyMissing")
	})
	t.Run("the index to drag", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		key := NewGlobalKey[*SliverReorderableListState]()
		ts.PumpWidget(reorderableApp(NewCustomScrollView([]Widget{NewSliverReorderableList(func(_ BuildContext, i int) Widget {
			return NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey(i), Height: foundation.Some(100.0)})
		}, 20, SliverReorderableListOpts{Key: key, OnReorderItem: func(int, int) {}})})))
		state, _ := key.CurrentState()
		requireTag(t, "ReorderableIndexInvalid", func() { state.StartItemDragReorder(99, nil, nil) })
	})
}

func TestSliverCrossAxisExpandedFlexMustBePositive(t *testing.T) {
	skipUnlessDebug(t)
	requireTag(t, "SliverCrossAxisExpandedFlexInvalid", func() { NewSliverCrossAxisExpanded(0, NewSliverToBoxAdapter(nil)) })
}

func TestSliverOverlapAbsorberNeedsItsOwnHandle(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	handle := NewSliverOverlapAbsorberHandle()
	defer handle.Dispose()
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{
		NewSliverOverlapAbsorber(handle, NewSliverToBoxAdapter(NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(10.0)}))),
		NewSliverOverlapAbsorber(handle, NewSliverToBoxAdapter(NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(10.0)})))})))
	_ = requireErrorTag(t, takeFirst(t, ts), "SliverOverlapAbsorberHandleWriters")
}

func TestOverlayFailures(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("a duplicated entry", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		overlay := pumpOverlay(ts, NewGlobalKey[*OverlayState]())
		entry := NewOverlayEntry(func(BuildContext) Widget { return NewSizedBox(nil) })
		requireTag(t, "OverlayEntryDuplicate", func() { overlay.Rearrange([]*OverlayEntry{entry, entry}) })
	})
	t.Run("always size to content without a sizing entry", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{AlwaysSizeToContent: true})))
		_ = requireErrorTag(t, takeFirst(t, ts), "OverlaySizeToContentInvalid")
	})
}

func TestNestedScrollViewFailures(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("the handle without a NestedScrollView", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(ltr(NewBuilder(func(bc BuildContext) Widget {
			requireTag(t, "NestedScrollViewMissing", func() { NestedScrollViewSliverOverlapAbsorberHandleFor(bc) })
			return NewSizedBox(nil)
		})))
	})
}

func TestPageViewFailures(t *testing.T) {
	skipUnlessDebug(t)
	requireTag(t, "ViewportFractionInvalid", func() { NewPageController(PageControllerOpts{ViewportFraction: foundation.Some(0.0)}) })
	t.Run("a controller without a view", func(t *testing.T) {
		controller := NewPageController()
		defer controller.Dispose()
		requireTag(t, "PageControllerNotAttached", func() { controller.Page() })
		requireTag(t, "PageControllerNotAttached", func() { controller.JumpToPage(1) })
	})
	t.Run("a controller with two views", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewPageController()
		defer controller.Dispose()
		page := func() Widget {
			return NewSizedBox(NewPageView([]Widget{NewSizedBox(nil)}, PageViewOpts{Controller: controller}), SizedBoxOpts{Height: foundation.Some(100.0)})
		}
		ts.PumpWidget(ltr(NewColumn([]Widget{page(), page()})))
		requireTag(t, "PageControllerMultipleViews", func() { controller.Page() })
		requireTag(t, "PageControllerMultipleViews", func() { controller.JumpToPage(1) })
	})
	t.Run("a cache extent that contradicts allowImplicitScrolling", func(t *testing.T) {
		requireTag(t, "PageViewScrollCacheExtentInconsistent", func() {
			NewPageView(nil, PageViewOpts{ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(10))})
		})
	})
}

func TestTwoDimensionalFailures(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("the maximum index", func(t *testing.T) {
		requireTag(t, "TwoDimensionalIndexInvalid", func() {
			NewTwoDimensionalChildBuilderDelegate(func(BuildContext, ChildVicinity) Widget { return nil },
				TwoDimensionalChildBuilderDelegateOpts{MaxXIndex: foundation.Some(-2)})
		})
	})
	t.Run("the scrollable above", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(ltr(NewBuilder(func(bc BuildContext) Widget {
			requireTag(t, "TwoDimensionalScrollableMissing", func() { TwoDimensionalScrollableOf(bc) })
			return NewSizedBox(nil)
		})))
	})
	t.Run("the viewport directions", func(t *testing.T) {
		offset := rendering.NewViewportOffsetZero()
		delegate := tableDelegate(1, 1)
		requireTag(t, "TwoDimensionalViewportAxisInvalid", func() {
			NewTwoDimensionalViewportBase(offset, painting.AxisDirectionLeft, offset, painting.AxisDirectionRight, delegate, painting.AxisVertical)
		})
		requireTag(t, "TwoDimensionalViewportAxisInvalid", func() {
			NewTwoDimensionalViewportBase(offset, painting.AxisDirectionDown, offset, painting.AxisDirectionUp, delegate, painting.AxisVertical)
		})
		r := &renderTable{}
		requireTag(t, "TwoDimensionalViewportAxisInvalid", func() {
			rendering.InitRenderTwoDimensionalViewport(&r.RenderTwoDimensionalViewport, r, offset, painting.AxisDirectionDown, offset, painting.AxisDirectionDown, delegate, painting.AxisVertical, nil)
		})
	})
	t.Run("the scroll view conflicts", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewScrollController()
		defer controller.Dispose()
		ts.PumpWidget(ltr(newTableView(tableDelegate(1, 1), tableOpts{TwoDimensionalScrollViewOpts: TwoDimensionalScrollViewOpts{
			Primary:         foundation.Some(true),
			VerticalDetails: foundation.Some(NewScrollableDetailsVertical(ScrollableDetailsOpts{Controller: controller}))}})))
		_ = requireErrorTag(t, takeFirst(t, ts), "TwoDimensionalScrollViewPrimaryConflict")
	})
	t.Run("the paint extent before the layout", func(t *testing.T) {
		data := &TwoDimensionalViewportParentData{}
		requireTag(t, "ParentDataPaintExtentUnknown", func() { data.IsVisible() })
	})
	t.Run("building outside of the layout", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(tableApp(tableDelegate(1, 1), tableOpts{}))
		r := tableViewportOf(ts)
		requireTag(t, "TwoDimensionalViewportBuildOutsideLayout", func() { r.BuildOrObtainChildFor(NewChildVicinity(0, 0)) })
		stranger := rendering.NewRenderLimitedBox(nil)
		requireTag(t, "TwoDimensionalViewportChildUnknown", func() { r.ParentDataOf(stranger) })
	})
}

func TestTwoDimensionalViewportManagerFailures(t *testing.T) {
	skipUnlessDebug(t)
	pump := func(t *testing.T, delegate *TwoDimensionalChildBuilderDelegate) (*gluttertest.Tester, rendering.TwoDimensionalChildManager) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(tableApp(delegate, tableOpts{}))
		return ts, ts.Element(gluttertest.ByType[*tableViewport]()).(rendering.TwoDimensionalChildManager)
	}
	t.Run("outside of the layout", func(t *testing.T) {
		_, manager := pump(t, tableDelegate(1, 1))
		requireTag(t, "TwoDimensionalViewportLayoutNotStarted", func() { manager.BuildChild(NewChildVicinity(0, 0)) })
		requireTag(t, "TwoDimensionalViewportLayoutNotStarted", func() { manager.ReuseChild(NewChildVicinity(0, 0)) })
		requireTag(t, "TwoDimensionalViewportLayoutNotStarted", func() { manager.EndLayout() })
	})
	t.Run("nested layouts", func(t *testing.T) {
		_, manager := pump(t, tableDelegate(1, 1))
		manager.StartLayout()
		requireTag(t, "TwoDimensionalViewportLayoutNested", func() { manager.StartLayout() })
		manager.EndLayout()
	})
	t.Run("reusing a child that does not exist", func(t *testing.T) {
		_, manager := pump(t, tableDelegate(1, 1))
		manager.StartLayout()
		requireTag(t, "TwoDimensionalViewportReuseMissing", func() { manager.ReuseChild(NewChildVicinity(9, 9)) })
		manager.EndLayout()
	})
	t.Run("building a vicinity twice", func(t *testing.T) {
		_, manager := pump(t, tableDelegate(1, 1))
		manager.StartLayout()
		manager.BuildChild(NewChildVicinity(0, 0))
		requireTag(t, "TwoDimensionalViewportChildDuplicate", func() { manager.BuildChild(NewChildVicinity(0, 0)) })
		manager.EndLayout()
	})
	t.Run("two children with the same key", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		same := foundation.NewValueKey("same")
		ts.PumpWidget(tableApp(NewTwoDimensionalChildBuilderDelegate(func(BuildContext, ChildVicinity) Widget {
			return NewSizedBox(nil, SizedBoxOpts{Key: same, Width: foundation.Some(200.0), Height: foundation.Some(200.0)})
		}, TwoDimensionalChildBuilderDelegateOpts{MaxXIndex: foundation.Some(1), MaxYIndex: foundation.Some(1),
			AddRepaintBoundaries: foundation.Some(false), AddAutomaticKeepAlives: foundation.Some(false)}), tableOpts{}))
		_ = requireErrorTag(t, takeFirst(t, ts), "TwoDimensionalViewportKeyDuplicate")
	})
}

func TestTwoDimensionalScrollableFailures(t *testing.T) {
	skipUnlessDebug(t)
	builder := func(BuildContext, rendering.ViewportOffset, rendering.ViewportOffset) Widget { return NewSizedBox(nil) }
	t.Run("the axes", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(ltr(NewTwoDimensionalScrollable(NewScrollableDetailsVertical(), NewScrollableDetailsHorizontal(), builder)))
		_ = requireErrorTag(t, takeFirst(t, ts), "TwoDimensionalScrollableAxisInvalid")
	})
	t.Run("the scrollables before the build", func(t *testing.T) {
		state := NewTwoDimensionalScrollable(NewScrollableDetailsHorizontal(), NewScrollableDetailsVertical(), builder).CreateState().(*TwoDimensionalScrollableState)
		requireTag(t, "TwoDimensionalScrollableNotBuilt", func() { state.VerticalScrollable() })
		requireTag(t, "TwoDimensionalScrollableNotBuilt", func() { state.HorizontalScrollable() })
	})
}

func TestListWheelElementFailures(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	children := make([]Widget, 10)
	for i := range children {
		children[i] = NewSizedBox(nil)
	}
	ts.PumpWidget(ltr(NewListWheelScrollView(30, children)))
	element := ts.Element(gluttertest.ByType[*ListWheelViewport]()).(*ListWheelElement)
	requireTag(t, "ListWheelChildMoved", func() { element.MoveRenderObjectChild(nil, nil, nil) })
	requireTag(t, "ListWheelChildBeforeMissing", func() { element.CreateChild(50, rendering.NewRenderLimitedBox(nil)) })
	viewport := ts.RenderObject(gluttertest.ByType[*ListWheelViewport]()).(*rendering.RenderListWheelViewport)
	stranger := rendering.NewRenderLimitedBox(nil)
	stranger.SetParentData(&rendering.ListWheelParentData{})
	requireTag(t, "ListWheelChildIndexMissing", func() { viewport.IndexOf(stranger) })
}

func TestFixedExtentPhysicsNeedsAFixedExtentPosition(t *testing.T) {
	skipUnlessDebug(t)
	metrics := NewFixedScrollMetrics(foundation.Some(0.0), foundation.Some(100.0), foundation.Some(10.0), foundation.Some(50.0), painting.AxisDirectionDown, 1)
	requireTag(t, "FixedExtentPhysicsWrongPosition", func() { NewFixedExtentScrollPhysics(nil).CreateBallisticSimulation(metrics, 500) })
}

func TestSliverMultiBoxAdaptorFailures(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverListList([]Widget{NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(10.0)}), NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(10.0)})})})))
	element := ts.Element(gluttertest.ByType[*SliverList]()).(*SliverMultiBoxAdaptorElement)
	box := rendering.NewRenderLimitedBox(nil)
	requireTag(t, "SliverChildAdoptedOutsideUpdate", func() { element.DidAdoptChild(box) })
	requireTag(t, "SliverChildRemovedOutsideUpdate", func() { element.RemoveRenderObjectChild(box, 0) })
	requireTag(t, "SliverChildSlotMismatch", func() { element.InsertRenderObjectChild(box, 3) })
	requireTag(t, "SliverChildSlotMismatch", func() { element.MoveRenderObjectChild(box, 0, 3) })
	requireTag(t, "SliverChildBeforeMissing", func() { element.CreateChild(5, box) })
	requireTag(t, "SliverLastIndexBeforeFirst", func() {
		NewSliverList(NewSliverChildListDelegate(nil)).EstimateMaxScrollOffset(rendering.SliverConstraints{}, 5, 3, 0, 0)
	})
}

func TestReorderableItemNotVisible(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	key := NewGlobalKey[*SliverReorderableListState]()
	ts.PumpWidget(reorderableApp(NewCustomScrollView([]Widget{NewSliverReorderableList(func(_ BuildContext, i int) Widget {
		return NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey(i), Height: foundation.Some(100.0)})
	}, 20, SliverReorderableListOpts{Key: key, OnReorderItem: func(int, int) {}})})))
	state, _ := key.CurrentState()
	requireTag(t, "ReorderableItemNotVisible", func() { state.StartItemDragReorder(15, nil, nil) })
}

func TestNestedScrollPositionFailures(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	key := NewGlobalKey[*NestedScrollViewState]()
	ts.PumpWidget(buildNestedTest(nestedTestOpts{key: key}))
	state, _ := key.CurrentState()
	outer := state.OuterController().Position()
	requireTag(t, "NestedScrollPositionApplyUserOffset", func() { outer.(interface{ ApplyUserOffset(float64) }).ApplyUserOffset(1) })
	requireTag(t, "NestedScrollPositionJumpToWithoutSettling", func() { outer.JumpToWithoutSettling(1) })
}
