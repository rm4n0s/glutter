package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// testFloatingHeader is TestRenderSliverFloatingPersistentHeader.
type testFloatingHeader struct {
	rendering.RenderSliverFloatingPersistentHeader
}

func newTestFloatingHeader(child rendering.Box) *testFloatingHeader {
	h := &testFloatingHeader{}
	rendering.InitRenderSliverFloatingPersistentHeader(&h.RenderSliverFloatingPersistentHeader, h, child, nil, nil, nil, nil)
	return h
}
func (h *testFloatingHeader) MaxExtent() float64                          { return 200 }
func (h *testFloatingHeader) MinExtent() float64                          { return 100 }
func (h *testFloatingHeader) Paint(*rendering.PaintingContext, ui.Offset) {}

// testFloatingPinnedHeader is TestRenderSliverFloatingPinnedPersistentHeader.
type testFloatingPinnedHeader struct {
	rendering.RenderSliverFloatingPinnedPersistentHeader
}

func newTestFloatingPinnedHeader(child rendering.Box) *testFloatingPinnedHeader {
	h := &testFloatingPinnedHeader{}
	rendering.InitRenderSliverFloatingPinnedPersistentHeader(&h.RenderSliverFloatingPinnedPersistentHeader, h, child, nil, nil, nil, nil)
	return h
}
func (h *testFloatingPinnedHeader) MaxExtent() float64                          { return 200 }
func (h *testFloatingPinnedHeader) MinExtent() float64                          { return 100 }
func (h *testFloatingPinnedHeader) Paint(*rendering.PaintingContext, ui.Offset) {}

func TestSliverPersistentHeaderMaxScrollObstructionExtent(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/35426.
	tests := []struct {
		name   string
		header func() rendering.Sliver
		want   float64
	}{
		{"RenderSliverFloatingPersistentHeader is 0", func() rendering.Sliver { return newTestFloatingHeader(newRenderSizedBox(400, 100)) }, 0},
		{"RenderSliverFloatingPinnedPersistentHeader is minExtent", func() rendering.Sliver { return newTestFloatingPinnedHeader(newRenderSizedBox(400, 100)) }, 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			header := tc.header()
			root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), header)
			layout(t, newHarness(t), root)
			if got := header.Geometry().MaxScrollObstructionExtent; got != tc.want {
				t.Errorf("maxScrollObstructionExtent = %v, want %v", got, tc.want)
			}
		})
	}
}
