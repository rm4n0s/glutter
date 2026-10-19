package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
)

// singleBoxSliver is a sliver with one box child (RenderSliverSingleBoxAdapter).
type singleBoxSliver interface {
	rendering.Sliver
	Child() rendering.Box
}

func expectSliverConstraints(t *testing.T, sliver rendering.Sliver, cacheOrigin, remainingPaintExtent, remainingCacheExtent, scrollOffset float64) {
	t.Helper()
	c := sliver.SliverConstraints()
	if c.CacheOrigin != cacheOrigin {
		t.Errorf("cacheOrigin = %v, want %v", c.CacheOrigin, cacheOrigin)
	}
	if c.RemainingPaintExtent != remainingPaintExtent {
		t.Errorf("remainingPaintExtent = %v, want %v", c.RemainingPaintExtent, remainingPaintExtent)
	}
	if c.RemainingCacheExtent != remainingCacheExtent {
		t.Errorf("remainingCacheExtent = %v, want %v", c.RemainingCacheExtent, remainingCacheExtent)
	}
	if c.ScrollOffset != scrollOffset {
		t.Errorf("scrollOffset = %v, want %v", c.ScrollOffset, scrollOffset)
	}
}

func expectSliverGeometry(t *testing.T, sliver rendering.Sliver, paintExtent, cacheExtent float64, visible bool) {
	t.Helper()
	g := sliver.Geometry()
	if g.PaintExtent != paintExtent {
		t.Errorf("paintExtent = %v, want %v", g.PaintExtent, paintExtent)
	}
	if g.CacheExtent != cacheExtent {
		t.Errorf("cacheExtent = %v, want %v", g.CacheExtent, cacheExtent)
	}
	if g.Visible != visible {
		t.Errorf("visible = %v, want %v", g.Visible, visible)
	}
}

func expectSliverChildConstraints(t *testing.T, sliver singleBoxSliver, maxWidth, maxHeight, minWidth, minHeight float64) {
	t.Helper()
	c := sliver.Child().BoxConstraints()
	if c.MaxWidth != maxWidth || c.MaxHeight != maxHeight || c.MinWidth != minWidth || c.MinHeight != minHeight {
		t.Errorf("child constraints = %v, want w=%v..%v h=%v..%v", c, minWidth, maxWidth, minHeight, maxHeight)
	}
}

// allAttached checks that all the boxes are attached (want true), and anyAttached
// that some box is (want false: none is).
func allAttached(t *testing.T, boxes []rendering.Box, want bool, what string) {
	t.Helper()
	got := true
	for _, b := range boxes {
		got = got && b.Attached()
	}
	if got != want {
		t.Errorf("%s: all attached = %v, want %v", what, got, want)
	}
}

func anyAttached(t *testing.T, boxes []rendering.Box, want bool, what string) {
	t.Helper()
	got := false
	for _, b := range boxes {
		got = got || b.Attached()
	}
	if got != want {
		t.Errorf("%s: any attached = %v, want %v", what, got, want)
	}
}

func TestSliverCacheViewportAndToBoxAdapter(t *testing.T) {
	var children []rendering.Sliver
	for i := 0; i < 30; i++ {
		children = append(children, rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(400, 100)))
	}
	// Viewport is 800x600, can show 6 children at a time.
	root := cachedViewport(children...)
	h := newHarness(t)
	layout(t, h, root)

	firstVisible := children[0]
	expectSliverConstraints(t, firstVisible, 0.0, 600.0, 600.0+250.0, 0.0)
	expectSliverGeometry(t, firstVisible, 100.0, 100.0, true)

	lastVisible := children[5]
	expectSliverConstraints(t, lastVisible, 0.0, 100.0, 350.0, 0.0)
	expectSliverGeometry(t, lastVisible, 100.0, 100.0, true)

	firstInCache := children[6]
	expectSliverConstraints(t, firstInCache, 0.0, 0.0, 250.0, 0.0)
	expectSliverGeometry(t, firstInCache, 0.0, 100.0, false)

	lastInCache := children[8]
	expectSliverConstraints(t, lastInCache, 0.0, 0.0, 50.0, 0.0)
	expectSliverGeometry(t, lastInCache, 0.0, 50.0, false)

	outsideCache := children[9]
	expectSliverConstraints(t, outsideCache, 0.0, 0.0, 0.0, 0.0)
	expectSliverGeometry(t, outsideCache, 0.0, 0.0, false)

	// scroll down half a sliver
	root.SetOffset(rendering.NewViewportOffsetFixed(50.0))
	pumpFrame(h, phaseLayout)

	firstVisible = children[0]
	expectSliverConstraints(t, firstVisible, -50.0, 600.0, 50.0+600.0+250.0, 50.0)
	expectSliverGeometry(t, firstVisible, 50.0, 100.0, true)

	lastVisible = children[6]
	expectSliverConstraints(t, lastVisible, 0.0, 50.0, 300.0, 0.0)
	expectSliverGeometry(t, lastVisible, 50.0, 100.0, true)

	firstInCache = children[7]
	expectSliverConstraints(t, firstInCache, 0.0, 0.0, 200.0, 0.0)
	expectSliverGeometry(t, firstInCache, 0.0, 100.0, false)

	lastInCache = children[8]
	expectSliverConstraints(t, lastInCache, 0.0, 0.0, 100.0, 0.0)
	expectSliverGeometry(t, lastInCache, 0.0, 100.0, false)

	outsideCache = children[9]
	expectSliverConstraints(t, outsideCache, 0.0, 0.0, 0.0, 0.0)
	expectSliverGeometry(t, outsideCache, 0.0, 0.0, false)

	// scroll down 1.5 slivers
	root.SetOffset(rendering.NewViewportOffsetFixed(150.0))
	pumpFrame(h, phaseLayout)

	firstInPreCache := children[0]
	expectSliverConstraints(t, firstInPreCache, -150.0, 600.0, 150.0+600.0+250.0, 150.0)
	expectSliverGeometry(t, firstInPreCache, 0.0, 100.0, false)

	firstVisible = children[1]
	expectSliverConstraints(t, firstVisible, -50.0, 600.0, 50.0+600.0+250.0, 50.0)
	expectSliverGeometry(t, firstVisible, 50.0, 100.0, true)

	// scroll down 10 slivers
	root.SetOffset(rendering.NewViewportOffsetFixed(1000.0))
	pumpFrame(h, phaseLayout)

	first := children[0]
	expectSliverConstraints(t, first, -250.0, 600.0, 250.0+600.0+250.0, 1000.0)
	expectSliverGeometry(t, first, 0.0, 0.0, false)

	firstInPreCache = children[7]
	expectSliverConstraints(t, firstInPreCache, -250.0, 600.0, 250.0+600.0+250.0, 300.0)
	expectSliverGeometry(t, firstInPreCache, 0.0, 50.0, false)

	lastInPreCache := children[9]
	expectSliverConstraints(t, lastInPreCache, -100.0, 600.0, 100.0+600.0+250.0, 100.0)
	expectSliverGeometry(t, lastInPreCache, 0.0, 100.0, false)

	firstVisible = children[10]
	expectSliverConstraints(t, firstVisible, 0.0, 600.0, 600.0+250.0, 0.0)
}

func TestSliverCacheFixedExtentList(t *testing.T) {
	// Viewport is 800x600, can show 6 full children at a time
	boxes := sizedBoxes(30, 400, 100)
	children := asBoxes(boxes)
	manager := newTestChildManager(children...)
	inner := manager.createFixedExtentList(100)
	root := cachedViewport(inner)
	h := newHarness(t)
	layout(t, h, root)

	expectSliverConstraints(t, inner, 0.0, 600.0, 600.0+250.0, 0.0)
	expectSliverGeometry(t, inner, 600.0, 850.0, true)
	allAttached(t, children[0:9], true, "children[0:9]")
	anyAttached(t, children[9:30], false, "children[9:30]")

	// scroll half an item down
	root.SetOffset(rendering.NewViewportOffsetFixed(50.0))
	pumpFrame(h, phaseLayout)

	expectSliverConstraints(t, inner, -50.0, 600.0, 50.0+600.0+250.0, 50.0)
	expectSliverGeometry(t, inner, 600.0, 900.0, true)
	allAttached(t, children[0:9], true, "children[0:9]")
	anyAttached(t, children[9:30], false, "children[9:30]")

	// scroll to the middle
	root.SetOffset(rendering.NewViewportOffsetFixed(1500.0))
	pumpFrame(h, phaseLayout)

	expectSliverConstraints(t, inner, -250.0, 600.0, 250.0+600.0+250.0, 1500.0)
	expectSliverGeometry(t, inner, 600.0, 1100.0, true)

	anyAttached(t, children[0:12], false, "children[0:12]")
	allAttached(t, children[12:24], true, "children[12:24]")
	anyAttached(t, children[24:30], false, "children[24:30]")

	// scroll to the end
	root.SetOffset(rendering.NewViewportOffsetFixed(2400.0))
	pumpFrame(h, phaseLayout)

	expectSliverConstraints(t, inner, -250.0, 600.0, 250.0+600.0+250.0, 2400.0)
	expectSliverGeometry(t, inner, 600.0, 850.0, true)
	anyAttached(t, children[0:21], false, "children[0:21]")
	allAttached(t, children[21:30], true, "children[21:30]")
}

func TestSliverCacheList(t *testing.T) {
	// Viewport is 800x600, can show 6 full children at a time
	boxes := sizedBoxes(30, 400, 100)
	children := asBoxes(boxes)
	manager := newTestChildManager(children...)
	inner := manager.createList()
	root := cachedViewport(inner)
	h := newHarness(t)
	layout(t, h, root)

	expectSliverConstraints(t, inner, 0.0, 600.0, 600.0+250.0, 0.0)
	expectSliverGeometry(t, inner, 600.0, 850.0, true)
	allAttached(t, children[0:9], true, "children[0:9]")
	anyAttached(t, children[9:30], false, "children[9:30]")

	// scroll half an item down
	root.SetOffset(rendering.NewViewportOffsetFixed(50.0))
	pumpFrame(h, phaseLayout)

	expectSliverConstraints(t, inner, -50.0, 600.0, 50.0+600.0+250.0, 50.0)
	expectSliverGeometry(t, inner, 600.0, 900.0, true)
	allAttached(t, children[0:9], true, "children[0:9]")
	anyAttached(t, children[9:30], false, "children[9:30]")

	// scroll to the middle
	root.SetOffset(rendering.NewViewportOffsetFixed(1500.0))
	pumpFrame(h, phaseLayout)

	expectSliverConstraints(t, inner, -250.0, 600.0, 250.0+600.0+250.0, 1500.0)
	expectSliverGeometry(t, inner, 600.0, 1100.0, true)

	anyAttached(t, children[0:12], false, "children[0:12]")
	allAttached(t, children[12:24], true, "children[12:24]")
	anyAttached(t, children[24:30], false, "children[24:30]")

	// scroll to the end
	root.SetOffset(rendering.NewViewportOffsetFixed(2400.0))
	pumpFrame(h, phaseLayout)

	expectSliverConstraints(t, inner, -250.0, 600.0, 250.0+600.0+250.0, 2400.0)
	expectSliverGeometry(t, inner, 600.0, 850.0, true)
	anyAttached(t, children[0:21], false, "children[0:21]")
	allAttached(t, children[21:30], true, "children[21:30]")
}

func TestSliverCacheGrid(t *testing.T) {
	// Viewport is 800x600, can show 6 full children at a time
	boxes := sizedBoxes(60, 400, 100)
	children := asBoxes(boxes)
	manager := newTestChildManager(children...)
	inner := manager.createGrid(rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{ChildAspectRatio: foundation.Some(4.0)}))
	root := cachedViewport(inner)
	h := newHarness(t)
	layout(t, h, root)

	expectSliverConstraints(t, inner, 0.0, 600.0, 600.0+250.0, 0.0)
	expectSliverGeometry(t, inner, 600.0, 850.0, true)
	allAttached(t, children[0:18], true, "children[0:18]")
	anyAttached(t, children[18:60], false, "children[18:60]")

	// scroll half an item down
	root.SetOffset(rendering.NewViewportOffsetFixed(50.0))
	pumpFrame(h, phaseLayout)

	expectSliverConstraints(t, inner, -50.0, 600.0, 50.0+600.0+250.0, 50.0)
	expectSliverGeometry(t, inner, 600.0, 900.0, true)
	allAttached(t, children[0:18], true, "children[0:18]")
	anyAttached(t, children[18:60], false, "children[18:60]")

	// scroll to the middle
	root.SetOffset(rendering.NewViewportOffsetFixed(1500.0))
	pumpFrame(h, phaseLayout)

	expectSliverConstraints(t, inner, -250.0, 600.0, 250.0+600.0+250.0, 1500.0)
	expectSliverGeometry(t, inner, 600.0, 1100.0, true)

	anyAttached(t, children[0:24], false, "children[0:24]")
	allAttached(t, children[24:48], true, "children[24:48]")
	anyAttached(t, children[48:60], false, "children[48:60]")

	// scroll to the end
	root.SetOffset(rendering.NewViewportOffsetFixed(2400.0))
	pumpFrame(h, phaseLayout)

	expectSliverConstraints(t, inner, -250.0, 600.0, 250.0+600.0+250.0, 2400.0)
	expectSliverGeometry(t, inner, 600.0, 850.0, true)
	anyAttached(t, children[0:42], false, "children[0:42]")
	allAttached(t, children[42:60], true, "children[42:60]")
}

func TestSliverCachePadding(t *testing.T) {
	// Viewport is 800x600, each item is 100px high with 50px before and after = 200px
	var adapters []*rendering.RenderSliverToBoxAdapter
	var paddings []*rendering.RenderSliverPadding
	var slivers []rendering.Sliver
	for i := 0; i < 30; i++ {
		adapter := rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(400, 100))
		padding := rendering.NewRenderSliverPadding(painting.NewEdgeInsetsSymmetric(50, 0), adapter)
		adapters = append(adapters, adapter)
		paddings = append(paddings, padding)
		slivers = append(slivers, padding)
	}
	root := cachedViewport(slivers...)
	h := newHarness(t)
	layout(t, h, root)

	firstVisiblePadding := paddings[0]
	expectSliverConstraints(t, firstVisiblePadding, 0.0, 600.0, 600.0+250.0, 0.0)
	expectSliverGeometry(t, firstVisiblePadding, 200.0, 200.0, true)
	firstVisiblePadded := adapters[0]
	expectSliverConstraints(t, firstVisiblePadded, 0.0, 550.0, 600.0+250.0-50.0, 0.0)
	expectSliverGeometry(t, firstVisiblePadded, 100.0, 100.0, true)

	lastVisiblePadding := paddings[2]
	expectSliverConstraints(t, lastVisiblePadding, 0.0, 200.0, 200.0+250.0, 0.0)
	expectSliverGeometry(t, lastVisiblePadding, 200.0, 200.0, true)
	lastVisiblePadded := adapters[2]
	expectSliverConstraints(t, lastVisiblePadded, 0.0, 150.0, 200.0+250.0-50.0, 0.0)
	expectSliverGeometry(t, lastVisiblePadded, 100.0, 100.0, true)

	firstCachePadding := paddings[3]
	expectSliverConstraints(t, firstCachePadding, 0.0, 0.0, 250.0, 0.0)
	expectSliverGeometry(t, firstCachePadding, 0.0, 200.0, false)
	firstCachePadded := adapters[3]
	expectSliverConstraints(t, firstCachePadded, 0.0, 0.0, 250.0-50.0, 0.0)
	expectSliverGeometry(t, firstCachePadded, 0.0, 100.0, false)

	lastCachePadding := paddings[4]
	expectSliverConstraints(t, lastCachePadding, 0.0, 0.0, 50.0, 0.0)
	expectSliverGeometry(t, lastCachePadding, 0.0, 50.0, false)
	lastCachePadded := adapters[4]
	expectSliverConstraints(t, lastCachePadded, 0.0, 0.0, 0.0, 0.0)
	expectSliverGeometry(t, lastCachePadded, 0.0, 0.0, false)

	// scroll first padding off screen
	root.SetOffset(rendering.NewViewportOffsetFixed(50.0))
	pumpFrame(h, phaseLayout)

	firstVisiblePadding = paddings[0]
	expectSliverConstraints(t, firstVisiblePadding, -50.0, 600.0, 50.0+600.0+250.0, 50.0)
	expectSliverGeometry(t, firstVisiblePadding, 150.0, 200.0, true)
	firstVisiblePadded = adapters[0]
	expectSliverConstraints(t, firstVisiblePadded, 0.0, 600.0, 600.0+250.0, 0.0)
	expectSliverGeometry(t, firstVisiblePadded, 100.0, 100.0, true)

	// scroll to the end
	root.SetOffset(rendering.NewViewportOffsetFixed(5400.0))
	pumpFrame(h, phaseLayout)

	firstPadding := paddings[0]
	expectSliverConstraints(t, firstPadding, -250.0, 600.0, 250.0+600.0+250.0, 5400.0)
	expectSliverGeometry(t, firstPadding, 0.0, 0.0, false)
	firstPadded := adapters[0]
	expectSliverConstraints(t, firstPadded, -200.0, 600.0, 250.0+600.0+250.0, 5350.0)
	expectSliverGeometry(t, firstPadded, 0.0, 0.0, false)

	firstPreCachePadding := paddings[25]
	expectSliverConstraints(t, firstPreCachePadding, -250.0, 600.0, 250.0+600.0+250.0, 400.0)
	expectSliverGeometry(t, firstPreCachePadding, 0.0, 50.0, false)
	firstPreCachePadded := adapters[25]
	expectSliverConstraints(t, firstPreCachePadded, -200.0, 600.0, 250.0+600.0+250.0, 350.0)
	expectSliverGeometry(t, firstPreCachePadded, 0.0, 0.0, false)

	lastPreCachePadding := paddings[26]
	expectSliverConstraints(t, lastPreCachePadding, -200.0, 600.0, 200.0+600.0+250.0, 200.0)
	expectSliverGeometry(t, lastPreCachePadding, 0.0, 200.0, false)
	lastPreCachePadded := adapters[26]
	expectSliverConstraints(t, lastPreCachePadded, -150.0, 600.0, 150.0+600.0+250.0, 150.0)
	expectSliverGeometry(t, lastPreCachePadded, 0.0, 100.0, false)

	lastVisiblePadding = paddings[29]
	expectSliverConstraints(t, lastVisiblePadding, 0.0, 200.0, 200.0+250.0, 0.0)
	expectSliverGeometry(t, lastVisiblePadding, 200.0, 200.0, true)
	lastVisiblePadded = adapters[29]
	expectSliverConstraints(t, lastVisiblePadded, 0.0, 150.0, 150.0+250.0, 0.0)
	expectSliverGeometry(t, lastVisiblePadded, 100.0, 100.0, true)
}

func TestSliverCacheFillRemainingInitiallyInView(t *testing.T) {
	// Viewport is 800x600
	const viewportHeight float64 = 600
	const viewportWidth float64 = 800
	const cacheExtent float64 = 250
	const beginningViewportCacheExtent float64 = viewportHeight + cacheExtent
	const firstSliverHeight float64 = 400
	const sliverFillRemainingChildHeight float64 = 100

	slivers := []rendering.Sliver{
		rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(400, firstSliverHeight)),
		rendering.NewRenderSliverFillRemaining(newRenderSizedBox(100, sliverFillRemainingChildHeight)),
	}
	root := cachedViewport(slivers...)
	h := newHarness(t)
	layout(t, h, root)

	firstVisibleSliver := slivers[0]
	expectSliverConstraints(t, firstVisibleSliver, 0.0, viewportHeight, beginningViewportCacheExtent, 0.0)
	expectSliverGeometry(t, firstVisibleSliver, firstSliverHeight, firstSliverHeight, true)

	// With RenderSliverFillRemaining:
	// * The child has a minExtent and maxExtent of the remaining space of the
	// viewportMainAxisExtent or the height of the child - whichever is larger.
	// * The sliver has a paintExtent of the child's minExtent/maxExtent or the
	// remainingPaintExtent - whichever is smaller.
	// * The sliver has a cacheExtent of the child's minExtent/maxExtent or the
	// remainingCacheExtent - whichever is smaller.
	sliverFillRemaining := slivers[1].(singleBoxSliver)
	const extentOfChild float64 = viewportHeight - firstSliverHeight
	remainingPaintExtent := viewportHeight - firstSliverHeight
	remainingCacheExtent := beginningViewportCacheExtent - firstSliverHeight
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, extentOfChild, viewportWidth, extentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, extentOfChild, extentOfChild, true)

	// Overscroll
	const scrollOffset float64 = 50
	root.SetOffset(rendering.NewViewportOffsetFixed(scrollOffset))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = viewportHeight - firstSliverHeight + scrollOffset
	remainingCacheExtent = beginningViewportCacheExtent - firstSliverHeight + scrollOffset

	// With RenderSliverFillRemaining, when you overscroll, the extent of the
	// child does not change and therefore neither does paintExtent or cacheExtent.
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, extentOfChild, viewportWidth, extentOfChild)
}

func TestSliverCacheFillRemainingScrolledIntoView(t *testing.T) {
	// Viewport is 800x600
	const viewportHeight float64 = 600
	const viewportWidth float64 = 800
	const cacheExtent float64 = 250
	const beginningViewportCacheExtent float64 = viewportHeight + cacheExtent
	const firstSliverHeight float64 = beginningViewportCacheExtent
	const sliverFillRemainingChildHeight float64 = 100

	slivers := []rendering.Sliver{
		rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(400, firstSliverHeight)),
		rendering.NewRenderSliverFillRemaining(newRenderSizedBox(100, sliverFillRemainingChildHeight)),
	}
	root := cachedViewport(slivers...)
	h := newHarness(t)
	layout(t, h, root)

	firstVisibleSliver := slivers[0]
	expectSliverConstraints(t, firstVisibleSliver, 0.0, viewportHeight, beginningViewportCacheExtent, 0.0)
	expectSliverGeometry(t, firstVisibleSliver, viewportHeight, firstSliverHeight, true)

	// With RenderSliverFillRemaining:
	// * The child has a minExtent and maxExtent of the remaining space of the
	// viewportMainAxisExtent or the height of the child - whichever is larger.
	// * The sliver has a paintExtent of the child's minExtent/maxExtent or the
	// remainingPaintExtent - whichever is smaller.
	// * The sliver has a cacheExtent of the child's minExtent/maxExtent or the
	// remainingCacheExtent - whichever is smaller.
	sliverFillRemaining := slivers[1].(singleBoxSliver)
	const extentOfChild float64 = sliverFillRemainingChildHeight
	var remainingPaintExtent float64 = 0
	var remainingCacheExtent float64 = 0
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, extentOfChild, viewportWidth, extentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, remainingPaintExtent, remainingCacheExtent, false)

	// Scroll so RenderSliverFillRemaining is not within viewport, but is
	// within remainingCacheExtent.
	root.SetOffset(rendering.NewViewportOffsetFixed(cacheExtent))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = 0
	remainingCacheExtent = cacheExtent

	// When within the remainingCacheExtent, the sliver will have a cacheExtent
	// of the child's extent.
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, extentOfChild, viewportWidth, extentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, remainingPaintExtent, extentOfChild, false)

	// Scroll so RenderSliverFillRemaining is partially within viewport.
	root.SetOffset(rendering.NewViewportOffsetFixed(cacheExtent + 50))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = 50
	remainingCacheExtent = cacheExtent + 50

	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, extentOfChild, viewportWidth, extentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, remainingPaintExtent, extentOfChild, true)

	// Scroll so RenderSliverFillRemaining is completely within viewport.
	root.SetOffset(rendering.NewViewportOffsetFixed(cacheExtent + sliverFillRemainingChildHeight))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = sliverFillRemainingChildHeight
	remainingCacheExtent = cacheExtent + sliverFillRemainingChildHeight

	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, extentOfChild, viewportWidth, extentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, extentOfChild, extentOfChild, true)

	// Overscroll
	root.SetOffset(rendering.NewViewportOffsetFixed(cacheExtent + sliverFillRemainingChildHeight + 50))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = sliverFillRemainingChildHeight + 50
	remainingCacheExtent = cacheExtent + sliverFillRemainingChildHeight + 50

	// With RenderSliverFillRemaining, when you overscroll, the extent of the
	// child does not change and therefore neither does paintExtent or cacheExtent.
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, extentOfChild, viewportWidth, extentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, extentOfChild, extentOfChild, true)
}

func TestSliverCacheFillRemainingAndOverscrollInitiallyInView(t *testing.T) {
	// Viewport is 800x600
	const viewportHeight float64 = 600
	const viewportWidth float64 = 800
	const cacheExtent float64 = 250
	const beginningViewportCacheExtent float64 = viewportHeight + cacheExtent
	const firstSliverHeight float64 = 400
	const sliverFillRemainingChildHeight float64 = 100

	slivers := []rendering.Sliver{
		rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(400, firstSliverHeight)),
		rendering.NewRenderSliverFillRemainingAndOverscroll(newRenderSizedBox(100, sliverFillRemainingChildHeight)),
	}
	root := cachedViewport(slivers...)
	h := newHarness(t)
	layout(t, h, root)

	firstVisibleSliver := slivers[0]
	expectSliverConstraints(t, firstVisibleSliver, 0.0, viewportHeight, beginningViewportCacheExtent, 0.0)
	expectSliverGeometry(t, firstVisibleSliver, firstSliverHeight, firstSliverHeight, true)

	// With RenderSliverFillRemainingAndOverscroll:
	// * The child has a minExtent of the remaining space of the viewportMainAxisExtent
	// or the height of the child - whichever is larger.
	// * The child has a maxExtent of the remaining space of the viewportMainAxisExtent,
	// the height of the child, or the remainingPaintExtent - whichever is larger.
	// * The sliver has a paintExtent of the child's maxExtent or the
	// remainingPaintExtent - whichever is smaller.
	// * The sliver has a cacheExtent of the child's minExtent or the
	// remainingCacheExtent - whichever is smaller.
	sliverFillRemaining := slivers[1].(singleBoxSliver)
	const minExtentOfChild float64 = viewportHeight - firstSliverHeight
	maxExtentOfChild := viewportHeight - firstSliverHeight
	remainingPaintExtent := viewportHeight - firstSliverHeight
	remainingCacheExtent := beginningViewportCacheExtent - firstSliverHeight
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, maxExtentOfChild, minExtentOfChild, true)

	// Overscroll
	const scrollOffset float64 = 50
	root.SetOffset(rendering.NewViewportOffsetFixed(scrollOffset))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = viewportHeight - firstSliverHeight + scrollOffset
	remainingCacheExtent = beginningViewportCacheExtent - firstSliverHeight + scrollOffset

	// When you overscroll, the child's maxExtent is the
	// remainingPaintExtent, since it's the higher value.
	maxExtentOfChild = remainingPaintExtent
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
}

func TestSliverCacheFillRemainingAndOverscrollScrolledIntoView(t *testing.T) {
	// Viewport is 800x600
	const viewportHeight float64 = 600
	const viewportWidth float64 = 800
	const cacheExtent float64 = 250
	const beginningViewportCacheExtent float64 = viewportHeight + cacheExtent
	const firstSliverHeight float64 = beginningViewportCacheExtent
	const sliverFillRemainingChildHeight float64 = 100

	slivers := []rendering.Sliver{
		rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(400, firstSliverHeight)),
		rendering.NewRenderSliverFillRemainingAndOverscroll(newRenderSizedBox(100, sliverFillRemainingChildHeight)),
	}
	root := cachedViewport(slivers...)
	h := newHarness(t)
	layout(t, h, root)

	firstVisibleSliver := slivers[0]
	expectSliverConstraints(t, firstVisibleSliver, 0.0, viewportHeight, beginningViewportCacheExtent, 0.0)
	expectSliverGeometry(t, firstVisibleSliver, viewportHeight, firstSliverHeight, true)

	// With RenderSliverFillRemainingAndOverscroll:
	// * The child has a minExtent of the remaining space of the viewportMainAxisExtent
	// or the height of the child - whichever is larger.
	// * The child has a maxExtent of the remaining space of the viewportMainAxisExtent,
	// the height of the child, or the remainingPaintExtent - whichever is larger.
	// * The sliver has a paintExtent of the child's maxExtent or the
	// remainingPaintExtent - whichever is smaller.
	// * The sliver has a cacheExtent of the child's minExtent or the
	// remainingCacheExtent - whichever is smaller.
	sliverFillRemaining := slivers[1].(singleBoxSliver)
	const minExtentOfChild float64 = sliverFillRemainingChildHeight
	maxExtentOfChild := sliverFillRemainingChildHeight
	var remainingPaintExtent float64 = 0
	var remainingCacheExtent float64 = 0
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, remainingPaintExtent, remainingCacheExtent, false)

	// Scroll so RenderSliverFillRemainingAndOverscroll is not within viewport,
	// but is within remainingCacheExtent.
	root.SetOffset(rendering.NewViewportOffsetFixed(cacheExtent))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = 0
	remainingCacheExtent = cacheExtent

	// When within the remainingCacheExtent, the sliver will have a cacheExtent
	// of the child's minExtent.
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, remainingPaintExtent, minExtentOfChild, false)

	// Scroll so RenderSliverFillRemainingAndOverscroll is partially within
	// the viewport.
	root.SetOffset(rendering.NewViewportOffsetFixed(cacheExtent + 50))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = 50
	remainingCacheExtent = cacheExtent + 50

	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, remainingPaintExtent, minExtentOfChild, true)

	// Scroll so RenderSliverFillRemainingAndOverscroll is completely within
	// the viewport.
	root.SetOffset(rendering.NewViewportOffsetFixed(cacheExtent + sliverFillRemainingChildHeight))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = sliverFillRemainingChildHeight
	remainingCacheExtent = cacheExtent + sliverFillRemainingChildHeight

	// When completely in view, the slivers's paintExtent is the child's
	// maxExtentOfChild, since it's the smaller value.
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, maxExtentOfChild, minExtentOfChild, true)

	// Overscroll
	root.SetOffset(rendering.NewViewportOffsetFixed(cacheExtent + sliverFillRemainingChildHeight + 50))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = sliverFillRemainingChildHeight + 50
	remainingCacheExtent = cacheExtent + sliverFillRemainingChildHeight + 50

	// When you overscroll, the child's maxExtent is the remainingPaintExtent,
	// since it's the higher value.
	maxExtentOfChild = remainingPaintExtent
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, maxExtentOfChild, minExtentOfChild, true)
}

func TestSliverCacheFillRemainingWithScrollableInitiallyInView(t *testing.T) {
	// Viewport is 800x600
	const viewportHeight float64 = 600
	const viewportWidth float64 = 800
	const cacheExtent float64 = 250
	const beginningViewportCacheExtent float64 = viewportHeight + cacheExtent
	const firstSliverHeight float64 = 400
	const sliverFillRemainingChildHeight float64 = 100

	slivers := []rendering.Sliver{
		rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(400, firstSliverHeight)),
		rendering.NewRenderSliverFillRemainingWithScrollable(newRenderSizedBox(100, sliverFillRemainingChildHeight)),
	}
	root := cachedViewport(slivers...)
	h := newHarness(t)
	layout(t, h, root)

	firstVisibleSliver := slivers[0]
	expectSliverConstraints(t, firstVisibleSliver, 0.0, viewportHeight, beginningViewportCacheExtent, 0.0)
	expectSliverGeometry(t, firstVisibleSliver, firstSliverHeight, firstSliverHeight, true)

	// With RenderSliverFillRemainingWithScrollable:
	// * The child has a minExtent of the remainingPaintExtent.
	// * If not within the viewport but within the cacheExtent, the child has
	// a maxExtent of the sliver's cacheExtent. Otherwise, the child has a
	// maxExtent of the remainingPaintExtent
	// * The sliver has a paintExtent of the child's minExtent or the
	// remainingPaintExtent - whichever is smaller.
	// * The sliver has a cacheExtent of either the viewportMainAxisExtent or
	// the remainingCacheExtent - whichever is smaller.
	sliverFillRemaining := slivers[1].(singleBoxSliver)
	remainingPaintExtent := viewportHeight - firstSliverHeight
	remainingCacheExtent := beginningViewportCacheExtent - firstSliverHeight
	minExtentOfChild := remainingPaintExtent
	maxExtentOfChild := remainingPaintExtent

	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, minExtentOfChild, remainingCacheExtent, true)

	// Overscroll so first sliver is partially out of view.
	root.SetOffset(rendering.NewViewportOffsetFixed(50))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = viewportHeight - firstSliverHeight + 50
	remainingCacheExtent = beginningViewportCacheExtent - firstSliverHeight + 50
	minExtentOfChild = remainingPaintExtent
	maxExtentOfChild = remainingPaintExtent

	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, minExtentOfChild, remainingCacheExtent, true)

	// Overscroll so only RenderSliverFillRemainingWithScrollable is visible.
	root.SetOffset(rendering.NewViewportOffsetFixed(firstSliverHeight))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = viewportHeight
	remainingCacheExtent = beginningViewportCacheExtent
	minExtentOfChild = remainingPaintExtent
	maxExtentOfChild = remainingPaintExtent

	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, minExtentOfChild, viewportHeight, true)
}

func TestSliverCacheFillRemainingWithScrollableScrolledIntoView(t *testing.T) {
	// Viewport is 800x600
	const viewportHeight float64 = 600
	const viewportWidth float64 = 800
	const cacheExtent float64 = 250
	const beginningViewportCacheExtent float64 = viewportHeight + cacheExtent
	const firstSliverHeight float64 = beginningViewportCacheExtent
	const sliverFillRemainingChildHeight float64 = 100

	slivers := []rendering.Sliver{
		rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(400, firstSliverHeight)),
		rendering.NewRenderSliverFillRemainingWithScrollable(newRenderSizedBox(100, sliverFillRemainingChildHeight)),
	}
	root := cachedViewport(slivers...)
	h := newHarness(t)
	layout(t, h, root)

	firstVisibleSliver := slivers[0]
	expectSliverConstraints(t, firstVisibleSliver, 0.0, viewportHeight, beginningViewportCacheExtent, 0.0)
	expectSliverGeometry(t, firstVisibleSliver, viewportHeight, firstSliverHeight, true)

	// With RenderSliverFillRemainingWithScrollable:
	// * The child has a minExtent of the remainingPaintExtent.
	// * If not within the viewport but within the cacheExtent, the child has
	// a maxExtent of the sliver's cacheExtent. Otherwise, the child has a
	// maxExtent of the remainingPaintExtent
	// * The sliver has a paintExtent of the child's minExtent or the
	// remainingPaintExtent - whichever is smaller.
	// * The sliver has a cacheExtent of either the viewportMainAxisExtent or
	// the remainingCacheExtent - whichever is smaller.
	sliverFillRemaining := slivers[1].(singleBoxSliver)
	var remainingPaintExtent float64 = 0
	var remainingCacheExtent float64 = 0
	minExtentOfChild := remainingPaintExtent
	maxExtentOfChild := remainingPaintExtent
	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, minExtentOfChild, remainingCacheExtent, false)

	// Scroll so RenderSliverFillRemainingWithScrollable is not within
	// viewport, but is within remainingCacheExtent.
	root.SetOffset(rendering.NewViewportOffsetFixed(cacheExtent))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = 0
	remainingCacheExtent = cacheExtent
	minExtentOfChild = remainingPaintExtent
	// When RenderSliverFillRemainingWithScrollable is completely outside the
	// viewport, but is within the remainingCacheExtent, the child has a
	// maxExtent of the slivers's cacheExtent.
	maxExtentOfChild = remainingCacheExtent

	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, minExtentOfChild, remainingCacheExtent, false)

	// Scroll so RenderSliverFillRemainingWithScrollable is partially within
	// viewport.
	root.SetOffset(rendering.NewViewportOffsetFixed(cacheExtent + 50))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = 50
	remainingCacheExtent = cacheExtent + 50
	minExtentOfChild = remainingPaintExtent
	maxExtentOfChild = remainingPaintExtent

	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, minExtentOfChild, remainingCacheExtent, true)

	// Scroll so RenderSliverFillRemainingWithScrollable takes the entire
	// viewport.
	root.SetOffset(rendering.NewViewportOffsetFixed(firstSliverHeight))
	pumpFrame(h, phaseLayout)
	remainingPaintExtent = viewportHeight
	remainingCacheExtent = beginningViewportCacheExtent
	minExtentOfChild = remainingPaintExtent
	maxExtentOfChild = remainingPaintExtent

	expectSliverConstraints(t, sliverFillRemaining, 0.0, remainingPaintExtent, remainingCacheExtent, 0.0)
	expectSliverChildConstraints(t, sliverFillRemaining, viewportWidth, maxExtentOfChild, viewportWidth, minExtentOfChild)
	expectSliverGeometry(t, sliverFillRemaining, minExtentOfChild, viewportHeight, true)
}
