package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// testRenderSliver is a sliver that sets its geometry directly.
type testRenderSliver struct {
	rendering.RenderSliver
	geometry rendering.SliverGeometry
}

func newTestRenderSliver(g rendering.SliverGeometry) *testRenderSliver {
	s := &testRenderSliver{geometry: g}
	rendering.InitRenderSliver(&s.RenderSliver, s)
	return s
}

func (s *testRenderSliver) PerformLayout()                              { s.SetGeometry(s.geometry) }
func (s *testRenderSliver) Paint(*rendering.PaintingContext, ui.Offset) {}

func TestSliverGetMaxPaintRect(t *testing.T) {
	tests := []struct {
		name        string
		geometry    rendering.SliverGeometry
		constraints rendering.SliverConstraints
		want        ui.Rect
	}{
		{"RenderSliver.getMaxPaintRect AxisDirection.down", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 100, MaxPaintExtent: 100}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionDown, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 0, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 300, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0)},
		{"RenderSliver.getMaxPaintRect AxisDirection.down and GrowthDirection.reverse", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 100, MaxPaintExtent: 100}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionDown, GrowthDirection: rendering.GrowthDirectionReverse, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 0, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 300, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0)},
		{"RenderSliver.getMaxPaintRect AxisDirection.down scrolled", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 50, MaxPaintExtent: 100}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionDown, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 50, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 50, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(0.0, -50.0, 100.0, 100.0)},
		{"RenderSliver.getMaxPaintRect AxisDirection.up", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 100, MaxPaintExtent: 100}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionUp, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 0, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 300, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0)},
		{"RenderSliver.getMaxPaintRect AxisDirection.up scrolled", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 50, MaxPaintExtent: 100}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionUp, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 50, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 50, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0)},
		{"RenderSliver.getMaxPaintRect AxisDirection.right", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 100, MaxPaintExtent: 100}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionRight, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 0, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 300, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionDown, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0)},
		{"RenderSliver.getMaxPaintRect AxisDirection.right scrolled", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 50, MaxPaintExtent: 100}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionRight, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 50, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 50, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionDown, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(-50.0, 0.0, 100.0, 100.0)},
		{"RenderSliver.getMaxPaintRect AxisDirection.left", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 100, MaxPaintExtent: 100}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionLeft, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 0, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 300, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionDown, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0)},
		{"RenderSliver.getMaxPaintRect AxisDirection.left scrolled", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 50, MaxPaintExtent: 100}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionLeft, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 50, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 50, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionDown, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0)},
		{"RenderSliver.getMaxPaintRect pinned", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 10, MaxPaintExtent: 100, MaxScrollObstructionExtent: 10}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionDown, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 95, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 10, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(0.0, -90.0, 100.0, 100.0)},
		{"RenderSliver.getMaxPaintRect infinite maxPaintExtent", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: inf, PaintExtent: 100, MaxPaintExtent: inf, CacheExtent: foundation.Some(150.0)}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionDown, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 50, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 300, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 100, RemainingCacheExtent: 200, CacheOrigin: -50},
			ui.NewRectFromLTWH(0.0, -50.0, 100.0, 150.0)},
		{"RenderSliver.getMaxPaintRect with crossAxisExtent set (e.g. SliverCrossAxisGroup)", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 100, MaxPaintExtent: 100, CrossAxisExtent: foundation.Some(50.0)}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionDown, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 0, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 300, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(0.0, 0.0, 50.0, 100.0)},
		{"RenderSliver.getMaxPaintRect maxPaintExtent > remainingPaintExtent", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 200, PaintExtent: 100, MaxPaintExtent: 150}),
			rendering.SliverConstraints{AxisDirection: painting.AxisDirectionDown, GrowthDirection: rendering.GrowthDirectionForward, UserScrollDirection: rendering.ScrollDirectionIdle, ScrollOffset: 0, PrecedingScrollExtent: 0, Overlap: 0, RemainingPaintExtent: 100, CrossAxisExtent: 100, CrossAxisDirection: painting.AxisDirectionRight, ViewportMainAxisExtent: 100, RemainingCacheExtent: 100, CacheOrigin: 0},
			ui.NewRectFromLTWH(0.0, 0.0, 100.0, 150.0)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sliver := newTestRenderSliver(tc.geometry)
			sliver.Layout(tc.constraints)
			if got := sliver.GetMaxPaintRect(); got != tc.want {
				t.Errorf("GetMaxPaintRect() = %v, want %v", got, tc.want)
			}
		})
	}
}
