package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func TestViewportCacheExtent(t *testing.T) {
	const width, height = 800.0, 600.0
	expandedOnAxis := func(v float64) ui.Rect { return ui.NewRectFromLTRB(0, 0-v, width, height+v) }
	tests := []struct {
		name  string
		cache foundation.Opt[rendering.ScrollCacheExtent]
		want  float64
	}{
		{"null, pixels", foundation.None[rendering.ScrollCacheExtent](), rendering.RenderAbstractViewportDefaultCacheExtent},
		{"0, pixels", foundation.Some(rendering.NewScrollCacheExtentPixels(0)), 0},
		{"500, pixels", foundation.Some(rendering.NewScrollCacheExtentPixels(500)), 500},
		{"0x viewport", foundation.Some(rendering.NewScrollCacheExtentViewport(0)), 0},
		{".5x viewport", foundation.Some(rendering.NewScrollCacheExtentViewport(0.5)), height / 2},
		{"1x viewport", foundation.Some(rendering.NewScrollCacheExtentViewport(1)), height},
		{"2.5x viewport", foundation.Some(rendering.NewScrollCacheExtentViewport(2.5)), height * 2.5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			children := []rendering.Sliver{rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(800, 400))}
			viewport := rendering.NewRenderViewport(painting.AxisDirectionLeft, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{
				RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{ScrollCacheExtent: tc.cache},
				Children:               children,
			})
			layout(t, newHarness(t), viewport)
			if got := viewport.DescribeSemanticsClip(nil); got != foundation.Some(expandedOnAxis(tc.want)) {
				t.Errorf("DescribeSemanticsClip = %v, want %v", got, expandedOnAxis(tc.want))
			}
		})
	}
}

func TestScrollCacheExtent(t *testing.T) {
	tests := []struct {
		name string
		e    rendering.ScrollCacheExtent
		main float64
		want float64
		str  string
	}{
		{"pixels", rendering.NewScrollCacheExtentPixels(250), 600, 250, "ScrollCacheExtent.pixels(250.0)"},
		{"viewport", rendering.NewScrollCacheExtentViewport(0.5), 600, 300, "ScrollCacheExtent.viewport(0.5)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.e.CalculateCacheOffset(tc.main); got != tc.want {
				t.Errorf("CalculateCacheOffset = %v, want %v", got, tc.want)
			}
			if got := tc.e.String(); got != tc.str {
				t.Errorf("String = %q, want %q", got, tc.str)
			}
		})
	}
}

// customConstraintsSliver is CustomConstraintsRenderSliver.
type customConstraintsSliver struct {
	rendering.RenderSliver
	constraints rendering.SliverConstraints
}

func newCustomConstraintsSliver(c rendering.SliverConstraints) *customConstraintsSliver {
	s := &customConstraintsSliver{constraints: c}
	rendering.InitRenderSliver(&s.RenderSliver, s)
	return s
}

func (s *customConstraintsSliver) SliverConstraints() rendering.SliverConstraints {
	return s.constraints
}
func (s *customConstraintsSliver) Geometry() rendering.SliverGeometry {
	return rendering.SliverGeometryZero
}
func (s *customConstraintsSliver) PerformLayout() {}

func TestShrinkWrappingViewportApproximatePaintClipInfiniteExtent(t *testing.T) {
	child := newCustomConstraintsSliver(rendering.SliverConstraints{
		AxisDirection:          painting.AxisDirectionDown,
		CrossAxisDirection:     painting.AxisDirectionLeft,
		CrossAxisExtent:        400,
		GrowthDirection:        rendering.GrowthDirectionForward,
		Overlap:                1, // must not equal 0 for this test
		RemainingPaintExtent:   math.Inf(1),
		UserScrollDirection:    rendering.ScrollDirectionIdle,
		ViewportMainAxisExtent: math.Inf(1), // must == infinity
	})
	viewport := rendering.NewRenderShrinkWrappingViewport(painting.AxisDirectionLeft, rendering.NewViewportOffsetZero(),
		rendering.RenderShrinkWrappingViewportOpts{Children: []rendering.Sliver{child}})
	layout(t, newHarness(t), viewport)
	want := foundation.Some(ui.NewRectFromLTRB(0, 0, 800, 600))
	if got := viewport.DescribeApproximatePaintClip(child); got != want {
		t.Errorf("DescribeApproximatePaintClip = %v, want %v", got, want)
	}
}
