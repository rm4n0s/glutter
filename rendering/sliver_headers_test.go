package rendering_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// persistentHeader gives the extents to the variants.
type persistentHeader struct{ min, max float64 }

func (p persistentHeader) MaxExtent() float64 { return p.max }
func (p persistentHeader) MinExtent() float64 { return p.min }

type scrollingHeader struct {
	rendering.RenderSliverScrollingPersistentHeader
	persistentHeader
	updates []float64
}

func newScrollingHeader(child rendering.Box, min, max float64, stretch *rendering.OverScrollHeaderStretchConfiguration) *scrollingHeader {
	h := &scrollingHeader{persistentHeader: persistentHeader{min, max}}
	rendering.InitRenderSliverScrollingPersistentHeader(&h.RenderSliverScrollingPersistentHeader, h, child, stretch)
	return h
}
func (h *scrollingHeader) UpdateChild(shrinkOffset float64, _ bool) {
	h.updates = append(h.updates, shrinkOffset)
}
func (h *scrollingHeader) Paint(*rendering.PaintingContext, ui.Offset) {}

type pinnedHeader struct {
	rendering.RenderSliverPinnedPersistentHeader
	persistentHeader
	overlaps []bool
}

func newPinnedHeader(child rendering.Box, min, max float64) *pinnedHeader {
	h := &pinnedHeader{persistentHeader: persistentHeader{min, max}}
	rendering.InitRenderSliverPinnedPersistentHeader(&h.RenderSliverPinnedPersistentHeader, h, child, nil, rendering.NewPersistentHeaderShowOnScreenConfiguration())
	return h
}
func (h *pinnedHeader) UpdateChild(_ float64, overlapsContent bool) {
	h.overlaps = append(h.overlaps, overlapsContent)
}
func (h *pinnedHeader) Paint(*rendering.PaintingContext, ui.Offset) {}

// originY is the vertical position of the origin of the object in the root.
func originY(o rendering.RenderObject) float64 {
	return o.GetTransformTo(nil).TransformRect(ui.Rect{}).Top
}

// headerViewport lays out the header, then a list of 20 items of 100 below it.
func headerViewport(t *testing.T, header rendering.Sliver, offset float64) (*rendering.RenderViewport, func(float64), []*RenderSizedBox) {
	t.Helper()
	list, boxes := listOf(20, 100)
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetFixed(offset), header, list)
	h := newHarness(t)
	layout(t, h, root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 600})))
	return root, func(o float64) {
		root.SetOffset(rendering.NewViewportOffsetFixed(o))
		pumpFrame(h, phaseLayout)
	}, boxes
}

func TestSliverScrollingPersistentHeader(t *testing.T) {
	child := newRenderSizedBox(300, 10000)
	header := newScrollingHeader(child, 100, 200, nil)
	_, scrollTo, boxes := headerViewport(t, header, 0)
	tests := []struct {
		offset                         float64
		wantChild                      float64
		wantPaint, wantLayout, wantPos float64
	}{
		{0, 200, 200, 200, 0},
		{50, 150, 150, 150, 0},
		{150, 100, 50, 50, -50},
		{300, 100, 0, 0, -200},
	}
	for _, tc := range tests {
		scrollTo(tc.offset)
		g := header.Geometry()
		if g.ScrollExtent != 200 || g.PaintExtent != tc.wantPaint || g.LayoutExtent != tc.wantLayout {
			t.Errorf("offset %v: geometry = %v, want scrollExtent 200, paintExtent %v, layoutExtent %v", tc.offset, g, tc.wantPaint, tc.wantLayout)
		}
		if got := child.Size().Height; got != tc.wantChild {
			t.Errorf("offset %v: child extent = %v, want %v", tc.offset, got, tc.wantChild)
		}
		if got := header.ChildMainAxisPosition(child); got != tc.wantPos {
			t.Errorf("offset %v: child position = %v, want %v", tc.offset, got, tc.wantPos)
		}
	}
	// the list starts where the header ends
	scrollTo(50)
	if got := boxes[0].LocalToGlobal(ui.Offset{}, nil).Dy - originY(header); got != 150 {
		t.Errorf("the first item is %v below the header, want 150", got)
	}
	// the child was told about the shrink offsets
	if len(header.updates) == 0 || header.updates[0] != 0 {
		t.Errorf("updates = %v, want to start with 0", header.updates)
	}
	if header.LastShrinkOffset() != 50 || header.LastOverlapsContent() {
		t.Errorf("last shrink offset %v and overlap %v, want 50 and false", header.LastShrinkOffset(), header.LastOverlapsContent())
	}
}

func TestSliverPinnedPersistentHeader(t *testing.T) {
	child := newRenderSizedBox(300, 10000)
	header := newPinnedHeader(child, 100, 200)
	_, scrollTo, _ := headerViewport(t, header, 0)
	tests := []struct {
		offset                 float64
		wantChild              float64
		wantPaint, wantLayout  float64
		wantOrigin             float64
		wantOverlapsAfterFrame bool
	}{
		{0, 200, 200, 200, 0, false},
		{150, 100, 100, 50, 0, false},
		{500, 100, 100, 0, 0, false},
	}
	for _, tc := range tests {
		scrollTo(tc.offset)
		g := header.Geometry()
		if g.ScrollExtent != 200 || g.PaintExtent != tc.wantPaint || g.LayoutExtent != tc.wantLayout || g.PaintOrigin != tc.wantOrigin {
			t.Errorf("offset %v: geometry = %v, want paintExtent %v, layoutExtent %v, paintOrigin %v", tc.offset, g, tc.wantPaint, tc.wantLayout, tc.wantOrigin)
		}
		if g.MaxScrollObstructionExtent != 100 {
			t.Errorf("offset %v: maxScrollObstructionExtent = %v, want 100", tc.offset, g.MaxScrollObstructionExtent)
		}
		if got := child.Size().Height; got != tc.wantChild {
			t.Errorf("offset %v: child extent = %v, want %v", tc.offset, got, tc.wantChild)
		}
		if got := header.ChildMainAxisPosition(child); got != 0 {
			t.Errorf("offset %v: child position = %v, want 0", tc.offset, got)
		}
	}
	// the pinned header stays at the top of the viewport
	if got := originY(header); got != 0 {
		t.Errorf("the pinned header is at %v, want the top (0 in the root)", got)
	}
}

func TestSliverPersistentHeaderStretch(t *testing.T) {
	triggered := 0
	stretch := &rendering.OverScrollHeaderStretchConfiguration{StretchTriggerOffset: 50, OnStretchTrigger: func() { triggered++ }}
	child := newRenderSizedBox(300, 10000)
	header := newScrollingHeader(child, 100, 200, stretch)
	if stretch2 := rendering.NewOverScrollHeaderStretchConfiguration(); stretch2.StretchTriggerOffset != 100 || stretch2.OnStretchTrigger != nil {
		t.Errorf("default stretch configuration = %+v", *stretch2)
	}
	_, scrollTo, _ := headerViewport(t, header, 0)
	if triggered != 0 {
		t.Fatalf("triggered %d times without overscroll", triggered)
	}
	scrollTo(-30)
	if triggered != 0 {
		t.Errorf("triggered %d times for an overscroll of 30", triggered)
	}
	if got := child.Size().Height; got != 230 {
		t.Errorf("stretched child extent = %v, want 230", got)
	}
	scrollTo(-60)
	if triggered != 1 {
		t.Errorf("triggered %d times for an overscroll of 60, want 1", triggered)
	}
	scrollTo(-70)
	if triggered != 1 {
		t.Errorf("triggered %d times while it stays over the trigger offset, want 1", triggered)
	}
	if header.StretchConfiguration() != stretch {
		t.Error("StretchConfiguration() is not the configuration")
	}
}

func TestSliverPersistentHeaderMinAboveMax(t *testing.T) {
	skipUnlessDebug(t)
	header := newScrollingHeader(newRenderSizedBox(300, 100), 300, 200, nil)
	requireTag(t, "SliverPersistentHeaderExtentsInvalid", func() { headerViewport(t, header, 0) })
}

func TestSliverFloatingPersistentHeader(t *testing.T) {
	child := newRenderSizedBox(300, 10000)
	header := newTestFloatingHeader(child) // min 100, max 200
	_, scrollTo, _ := headerViewport(t, header, 0)
	// Scrolling down hides the header; scrolling back up shows it again as soon
	// as the user scrolls forward.
	header.UpdateScrollStartDirection(rendering.ScrollDirectionForward)
	steps := []struct {
		offset    float64
		wantPaint float64
	}{
		{300, 0},
		{250, 50}, // the header was off screen: it comes back from its far edge
		{150, 150},
		{0, 200},
	}
	for _, s := range steps {
		scrollTo(s.offset)
		if g := header.Geometry(); g.PaintExtent != s.wantPaint {
			t.Errorf("at %v: paintExtent = %v, want %v", s.offset, g.PaintExtent, s.wantPaint)
		}
	}
}

func TestSliverFloatingPersistentHeaderStaysHiddenWhenScrollingBackIsNotAllowed(t *testing.T) {
	header := newTestFloatingHeader(newRenderSizedBox(300, 10000))
	_, scrollTo, _ := headerViewport(t, header, 0)
	scrollTo(300)
	scrollTo(100)
	// The user never scrolled forward (the viewport offset reports idle), so the
	// header does not expand while the content scrolls back.
	if g := header.Geometry(); g.PaintExtent != 100 {
		// the content is back at 100: the header is 200 - 100 = 100 on screen
		// because the sliver itself is scrolled by 100
		t.Errorf("paintExtent = %v, want 100", g.PaintExtent)
	}
}

func TestSliverFloatingPersistentHeaderSnap(t *testing.T) {
	h := newHarness(t)
	header := &snapHeader{}
	rendering.InitRenderSliverFloatingPersistentHeader(&header.RenderSliverFloatingPersistentHeader, header, newRenderSizedBox(300, 10000),
		tickerProvider{h}, rendering.NewFloatingHeaderSnapConfiguration(rendering.FloatingHeaderSnapConfigurationOpts{
			Duration: foundation.Some(100 * time.Millisecond), Curve: animation.Curves.Linear}), nil, nil)
	list, _ := listOf(20, 100)
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetFixed(0), header, list)
	layout(t, h, root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 600})))
	if header.SnapConfiguration() == nil || header.Vsync() == nil {
		t.Fatal("the snap configuration and the vsync were lost")
	}
	// Fully visible: nothing to snap forward; snapping back starts the animation.
	header.MaybeStartSnapAnimation(rendering.ScrollDirectionForward)
	header.MaybeStartSnapAnimation(rendering.ScrollDirectionReverse)
	header.MaybeStopSnapAnimation(rendering.ScrollDirectionIdle)
	// Without a snap configuration there is nothing to do, even without a vsync.
	header.SetSnapConfiguration(nil)
	header.SetVsync(nil)
	header.MaybeStartSnapAnimation(rendering.ScrollDirectionReverse)
}

type snapHeader struct {
	rendering.RenderSliverFloatingPersistentHeader
}

func (*snapHeader) MaxExtent() float64                          { return 200 }
func (*snapHeader) MinExtent() float64                          { return 100 }
func (*snapHeader) Paint(*rendering.PaintingContext, ui.Offset) {}

func TestSliverFloatingPersistentHeaderFailures(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("snapping without a vsync", func(t *testing.T) {
		header := &snapHeader{}
		rendering.InitRenderSliverFloatingPersistentHeader(&header.RenderSliverFloatingPersistentHeader, header, newRenderSizedBox(300, 10000),
			nil, rendering.NewFloatingHeaderSnapConfiguration(), nil, nil)
		list, _ := listOf(20, 100)
		root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetFixed(0), header, list)
		layout(t, newHarness(t), root)
		requireTag(t, "SliverFloatingHeaderVsyncMissing", func() { header.MaybeStartSnapAnimation(rendering.ScrollDirectionReverse) })
	})
	t.Run("showing a descendant of a header without a child", func(t *testing.T) {
		header := &snapHeader{}
		rendering.InitRenderSliverFloatingPersistentHeader(&header.RenderSliverFloatingPersistentHeader, header, nil, nil, nil, nil,
			rendering.NewPersistentHeaderShowOnScreenConfiguration())
		requireTag(t, "SliverFloatingHeaderChildMissing", func() {
			header.ShowOnScreen(rendering.ShowOnScreenOpts{Descendant: newRenderSizedBox(1, 1)})
		})
	})
}
