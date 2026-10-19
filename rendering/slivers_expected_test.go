package rendering_test

// Expected descriptions of slivers_test.dart, copied from the Dart test.
const (
	viewportNoChildrenBeforeLayout = `RenderViewport#00000 NEEDS-LAYOUT NEEDS-PAINT DETACHED
   needs compositing
   parentData: MISSING
   constraints: MISSING
   size: MISSING
   axisDirection: down
   crossAxisDirection: right
   offset: fixedViewportOffset#00000(offset: 0.0)
   anchor: 0.0
`
	viewportNoChildrenAfterLayout = `RenderViewport#00000 NEEDS-LAYOUT NEEDS-PAINT
   needs compositing
   parentData: <none>
   constraints: BoxConstraints(w=800.0, h=600.0)
   size: Size(800.0, 600.0)
   axisDirection: down
   crossAxisDirection: right
   offset: fixedViewportOffset#00000(offset: 900.0)
   anchor: 0.0
`
	viewportDownDescription = `RenderViewport#00000 NEEDS-PAINT NEEDS-COMPOSITING-BITS-UPDATE
 │ needs compositing
 │ parentData: <none>
 │ constraints: BoxConstraints(w=800.0, h=600.0)
 │ size: Size(800.0, 600.0)
 │ axisDirection: down
 │ crossAxisDirection: right
 │ offset: fixedViewportOffset#00000(offset: 0.0)
 │ anchor: 0.0
 │
 ├─center child: RenderSliverToBoxAdapter#00000 relayoutBoundary=up1 NEEDS-PAINT NEEDS-COMPOSITING-BITS-UPDATE
 │ │ parentData: paintOffset=Offset(0.0, 0.0) (can use size)
 │ │ constraints: SliverConstraints(AxisDirection.down,
 │ │   GrowthDirection.forward, ScrollDirection.idle, scrollOffset:
 │ │   0.0, precedingScrollExtent: 0.0, remainingPaintExtent: 600.0,
 │ │   crossAxisExtent: 800.0, crossAxisDirection:
 │ │   AxisDirection.right, viewportMainAxisExtent: 600.0,
 │ │   remainingCacheExtent: 850.0, cacheOrigin: 0.0)
 │ │ geometry: SliverGeometry(scrollExtent: 400.0, paintExtent: 400.0,
 │ │   maxPaintExtent: 400.0, cacheExtent: 400.0)
 │ │
 │ └─child: RenderSizedBox#00000 NEEDS-PAINT
 │     parentData: paintOffset=Offset(0.0, -0.0) (can use size)
 │     constraints: BoxConstraints(w=800.0, 0.0<=h<=Infinity)
 │     size: Size(800.0, 400.0)
 │
 ├─child 1: RenderSliverToBoxAdapter#00000 relayoutBoundary=up1 NEEDS-PAINT NEEDS-COMPOSITING-BITS-UPDATE
 │ │ parentData: paintOffset=Offset(0.0, 400.0) (can use size)
 │ │ constraints: SliverConstraints(AxisDirection.down,
 │ │   GrowthDirection.forward, ScrollDirection.idle, scrollOffset:
 │ │   0.0, precedingScrollExtent: 400.0, remainingPaintExtent: 200.0,
 │ │   crossAxisExtent: 800.0, crossAxisDirection:
 │ │   AxisDirection.right, viewportMainAxisExtent: 600.0,
 │ │   remainingCacheExtent: 450.0, cacheOrigin: 0.0)
 │ │ geometry: SliverGeometry(scrollExtent: 400.0, paintExtent: 200.0,
 │ │   maxPaintExtent: 400.0, hasVisualOverflow: true, cacheExtent:
 │ │   400.0)
 │ │
 │ └─child: RenderSizedBox#00000 NEEDS-PAINT
 │     parentData: paintOffset=Offset(0.0, -0.0) (can use size)
 │     constraints: BoxConstraints(w=800.0, 0.0<=h<=Infinity)
 │     size: Size(800.0, 400.0)
 │
 ├─child 2: RenderSliverToBoxAdapter#00000 relayoutBoundary=up1 NEEDS-PAINT NEEDS-COMPOSITING-BITS-UPDATE
 │ │ parentData: paintOffset=Offset(0.0, 800.0) (can use size)
 │ │ constraints: SliverConstraints(AxisDirection.down,
 │ │   GrowthDirection.forward, ScrollDirection.idle, scrollOffset:
 │ │   0.0, precedingScrollExtent: 800.0, remainingPaintExtent: 0.0,
 │ │   crossAxisExtent: 800.0, crossAxisDirection:
 │ │   AxisDirection.right, viewportMainAxisExtent: 600.0,
 │ │   remainingCacheExtent: 50.0, cacheOrigin: 0.0)
 │ │ geometry: SliverGeometry(scrollExtent: 400.0, hidden,
 │ │   maxPaintExtent: 400.0, hasVisualOverflow: true, cacheExtent:
 │ │   50.0)
 │ │
 │ └─child: RenderSizedBox#00000 NEEDS-PAINT
 │     parentData: paintOffset=Offset(0.0, -0.0) (can use size)
 │     constraints: BoxConstraints(w=800.0, 0.0<=h<=Infinity)
 │     size: Size(800.0, 400.0)
 │
 ├─child 3: RenderSliverToBoxAdapter#00000 relayoutBoundary=up1 NEEDS-PAINT NEEDS-COMPOSITING-BITS-UPDATE
 │ │ parentData: paintOffset=Offset(0.0, 1200.0) (can use size)
 │ │ constraints: SliverConstraints(AxisDirection.down,
 │ │   GrowthDirection.forward, ScrollDirection.idle, scrollOffset:
 │ │   0.0, precedingScrollExtent: 1200.0, remainingPaintExtent: 0.0,
 │ │   crossAxisExtent: 800.0, crossAxisDirection:
 │ │   AxisDirection.right, viewportMainAxisExtent: 600.0,
 │ │   remainingCacheExtent: 0.0, cacheOrigin: 0.0)
 │ │ geometry: SliverGeometry(scrollExtent: 400.0, hidden,
 │ │   maxPaintExtent: 400.0, hasVisualOverflow: true)
 │ │
 │ └─child: RenderSizedBox#00000 NEEDS-PAINT
 │     parentData: paintOffset=Offset(0.0, -0.0) (can use size)
 │     constraints: BoxConstraints(w=800.0, 0.0<=h<=Infinity)
 │     size: Size(800.0, 400.0)
 │
 └─child 4: RenderSliverToBoxAdapter#00000 relayoutBoundary=up1 NEEDS-PAINT NEEDS-COMPOSITING-BITS-UPDATE
   │ parentData: paintOffset=Offset(0.0, 1600.0) (can use size)
   │ constraints: SliverConstraints(AxisDirection.down,
   │   GrowthDirection.forward, ScrollDirection.idle, scrollOffset:
   │   0.0, precedingScrollExtent: 1600.0, remainingPaintExtent: 0.0,
   │   crossAxisExtent: 800.0, crossAxisDirection:
   │   AxisDirection.right, viewportMainAxisExtent: 600.0,
   │   remainingCacheExtent: 0.0, cacheOrigin: 0.0)
   │ geometry: SliverGeometry(scrollExtent: 400.0, hidden,
   │   maxPaintExtent: 400.0, hasVisualOverflow: true)
   │
   └─child: RenderSizedBox#00000 NEEDS-PAINT
       parentData: paintOffset=Offset(0.0, -0.0) (can use size)
       constraints: BoxConstraints(w=800.0, 0.0<=h<=Infinity)
       size: Size(800.0, 400.0)
`
)
