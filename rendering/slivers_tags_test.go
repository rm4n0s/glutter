package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// validSliverConstraints are valid constraints for a vertical sliver in a viewport
// of 100x600.
func validSliverConstraints() rendering.SliverConstraints {
	return rendering.SliverConstraints{
		AxisDirection: painting.AxisDirectionDown, GrowthDirection: rendering.GrowthDirectionForward,
		UserScrollDirection: rendering.ScrollDirectionIdle, RemainingPaintExtent: 600, CrossAxisExtent: 100,
		CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 600, RemainingCacheExtent: 600,
	}
}

func geometryOf(o rendering.SliverGeometryOpts) rendering.SliverGeometry {
	return rendering.NewSliverGeometry(o)
}

// TestSliverFailureTags checks the route of every failure of the sliver protocol,
// of the viewports and of the sliver render objects.
func TestSliverFailureTags(t *testing.T) {
	skipUnlessDebug(t)
	manager := func() *testSliverBoxChildManager { return newTestChildManager() }
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"sliver without geometry", "SliverNotLaidOut", func() { newTestRenderSliver(rendering.SliverGeometryZero).Geometry() }},
		{"geometry set outside layout", "SliverGeometrySetterInvalid", func() { newTestRenderSliver(rendering.SliverGeometryZero).SetGeometry(rendering.SliverGeometryZero) }},
		{"geometry outside the remaining paint extent", "SliverGeometryViolatesConstraints", func() {
			c := validSliverConstraints()
			c.RemainingPaintExtent = 50
			newTestRenderSliver(geometryOf(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 100, MaxPaintExtent: 100})).Layout(c)
		}},
		{"paint offset of an inverted range", "SliverRangeInvalid", func() {
			newTestRenderSliver(rendering.SliverGeometryZero).CalculatePaintOffset(validSliverConstraints(), 10, 5)
		}},
		{"cache offset of an inverted range", "SliverRangeInvalid", func() {
			newTestRenderSliver(rendering.SliverGeometryZero).CalculateCacheOffset(validSliverConstraints(), 10, 5)
		}},
		{"child position of a sliver without it", "SliverChildPositionMissing", func() { newTestRenderSliver(rendering.SliverGeometryZero).ChildMainAxisPosition(nil) }},
		{"paint transform of a sliver without it", "SliverPaintTransformMissing", func() {
			newTestRenderSliver(rendering.SliverGeometryZero).ApplyPaintTransform(nil, ui.NewMatrix4Identity())
		}},
		{"sliver sized by the parent", "SliverPerformResize", func() { newTestRenderSliver(rendering.SliverGeometryZero).PerformResize() }},
		{"geometry with a zero correction", "SliverGeometryInvalid", func() {
			rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollOffsetCorrection: foundation.Some(0.0)})
		}},
		{"sliver constraints that are not valid", "SliverConstraintsInvalid", func() {
			c := validSliverConstraints()
			c.ScrollOffset = -1
			c.DebugAssertIsValid()
		}},
		{"painted child size that is not finite", "SliverPaintExtentInvalid", func() {
			c := validSliverConstraints()
			c.RemainingPaintExtent = math.Inf(1)
			rendering.NewRenderSliverFillRemainingWithScrollable(nil).Layout(c)
		}},
		{"fill remaining with an infinite extent", "SliverFillRemainingExtentInfinite", func() {
			c := validSliverConstraints()
			c.ViewportMainAxisExtent = math.Inf(1)
			rendering.NewRenderSliverFillRemaining(nil).Layout(c)
		}},
		{"fill viewport fraction that is not positive", "SliverViewportFractionInvalid", func() {
			rendering.NewRenderSliverFillViewport(manager(), rendering.RenderSliverFillViewportOpts{ViewportFraction: foundation.Some(0.0)})
		}},
		{"proxy sliver without a child", "ProxySliverChildMissing", func() { rendering.NewRenderSliverOpacity(nil).PerformLayout() }},
		{"constrained cross axis with a negative maximum", "SliverMaxExtentNegative", func() { rendering.NewRenderSliverConstrainedCrossAxis(-1, nil) }},
		{"show on screen configuration with an inverted range", "PersistentHeaderShowOnScreenInvalid", func() {
			rendering.NewPersistentHeaderShowOnScreenConfiguration(rendering.PersistentHeaderShowOnScreenConfigurationOpts{
				MinShowOnScreenExtent: foundation.Some(10.0), MaxShowOnScreenExtent: foundation.Some(5.0)})
		}},
		{"grid delegate without columns", "SliverGridDelegateInvalid", func() { rendering.NewSliverGridDelegateWithFixedCrossAxisCount(0) }},
		{"grid delegate with a negative spacing", "SliverGridDelegateInvalid", func() {
			rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{MainAxisSpacing: -1})
		}},
		{"grid delegate without a maximum extent", "SliverGridDelegateInvalid", func() { rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(0) }},
		{"max extent grid layout in no extent", "SliverGridDelegateInvalid", func() {
			rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(100).GetLayout(rendering.SliverConstraints{})
		}},
		{"tile layout without columns", "SliverGridLayoutInvalid", func() { rendering.NewSliverGridRegularTileLayout(0, 1, 1, 1, 1, false) }},
		{"fixed extent list with a negative extent", "SliverItemExtentInvalid", func() { newTestFixedExtentAdaptor(-1).PerformLayout() }},
		{"fixed extent list without an extent", "SliverItemExtentInvalid", func() {
			a := &noExtentAdaptor{}
			rendering.InitRenderSliverFixedExtentBoxAdaptor(&a.RenderSliverFixedExtentBoxAdaptor, a, manager())
			a.PerformLayout()
		}},
		{"collecting more garbage than children", "SliverGarbageInvalid", func() {
			rendering.NewRenderSliverList(manager()).CollectGarbage(1, 1)
		}},
		{"empty child list", "SliverChildListInvalid", func() {
			rendering.NewRenderSliverList(manager()).DebugAssertChildListIsNonEmptyAndContiguous()
		}},
		{"viewport axes along the same axis", "ViewportAxesInvalid", func() {
			rendering.NewRenderViewport(painting.AxisDirectionUp, rendering.NewViewportOffsetZero())
		}},
		{"viewport anchor above one", "ViewportAnchorInvalid", func() {
			rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Anchor: 2})
		}},
		{"viewport anchor set below zero", "ViewportAnchorInvalid", func() {
			rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero()).SetAnchor(-1)
		}},
		{"viewport scroll offset that is negative", "ViewportScrollOffsetInvalid", func() {
			rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero()).LayoutChildSequence(rendering.LayoutChildSequenceOpts{ScrollOffset: -1})
		}},
		{"intrinsic width of a viewport", "ViewportIntrinsicsUnsupported", func() {
			rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero()).GetMinIntrinsicWidth(100)
		}},
		{"viewport not found", "ViewportNotFound", func() { rendering.RenderAbstractViewportOf(newRenderSizedBox(1, 1)) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { requireTag(t, tc.tag, tc.call) })
	}
}

// noExtentAdaptor is a fixed extent adaptor that has neither an extent nor a
// builder.
type noExtentAdaptor struct {
	rendering.RenderSliverFixedExtentBoxAdaptor
}

func (a *noExtentAdaptor) ItemExtent() foundation.Opt[float64]            { return foundation.None[float64]() }
func (a *noExtentAdaptor) ItemExtentBuilder() rendering.ItemExtentBuilder { return nil }

// correctingSliver always asks the viewport for a scroll offset correction.
type correctingSliver struct{ rendering.RenderSliver }

func newCorrectingSliver() *correctingSliver {
	s := &correctingSliver{}
	rendering.InitRenderSliver(&s.RenderSliver, s)
	return s
}

func (s *correctingSliver) PerformLayout() {
	s.SetGeometry(rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollOffsetCorrection: foundation.Some(1.0)}))
}
func (s *correctingSliver) Paint(*rendering.PaintingContext, ui.Offset) {}

// TestViewportLayoutFailureTags checks the failures that happen while a viewport
// is laid out in a frame.
func TestViewportLayoutFailureTags(t *testing.T) {
	skipUnlessDebug(t)
	// unbounded gives the box the constraints of the overflow box
	unbounded := func(box rendering.Box, maxWidth, maxHeight float64) rendering.Box {
		return rendering.NewRenderConstrainedOverflowBox(box, rendering.RenderConstrainedOverflowBoxOpts{
			Alignment: painting.AlignmentTopLeft,
			MinWidth:  some(0), MaxWidth: some(maxWidth), MinHeight: some(0), MaxHeight: some(maxHeight),
		})
	}
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"viewport offset that is infinite", "ViewportExtentInvalid", func() {
			layout(t, newHarness(t), rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetFixed(math.Inf(1)),
				rendering.RenderViewportOpts{Children: []rendering.Sliver{rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(10, 10))}}))
		}},
		{"layout cycles that never end", "ViewportLayoutCyclesExceeded", func() {
			layout(t, newHarness(t), rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(),
				rendering.RenderViewportOpts{Children: []rendering.Sliver{newCorrectingSliver()}}))
		}},
		{"shrink wrapping viewport in unbounded width", "ViewportUnboundedCrossAxis", func() {
			layout(t, newHarness(t), unbounded(rendering.NewRenderShrinkWrappingViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero()), math.Inf(1), 600))
		}},
		{"viewport in unbounded height", "ViewportUnboundedMainAxis", func() {
			layout(t, newHarness(t), unbounded(rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero()), 800, math.Inf(1)))
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { requireTag(t, tc.tag, tc.call) })
	}
}

func TestViewportMaxLayoutCyclesDependOnChildren(t *testing.T) {
	skipUnlessDebug(t)
	// Each child gets 10 layout cycles: a viewport with N children corrects its
	// offset 10*N times before it gives up.
	for _, n := range []int{1, 3} {
		offset := newViewportOffsetSpy(0)
		var slivers []rendering.Sliver
		for i := 0; i < n; i++ {
			slivers = append(slivers, newCorrectingSliver())
		}
		root := rendering.NewRenderViewport(painting.AxisDirectionRight, offset, rendering.RenderViewportOpts{Children: slivers})
		requireTag(t, "ViewportLayoutCyclesExceeded", func() { layout(t, newHarness(t), root) })
		if want := 10 * n; int(offset.pixels) != want {
			t.Errorf("%d children: the offset was corrected %v times, want %d", n, offset.pixels, want)
		}
	}
}
