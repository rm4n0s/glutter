package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func TestSliverPaddingLayout(t *testing.T) {
	tests := []struct {
		name         string
		axis         painting.AxisDirection
		cross        painting.AxisDirection
		padding      painting.EdgeInsetsGeometry
		direction    ui.TextDirection
		viewport     ui.Size
		offset       float64
		wantOffset   ui.Offset // of the child, in its parent
		wantChild    ui.Size
		wantScroll   float64
		wantPaint    float64
		wantChildPos float64
	}{
		{"down", painting.AxisDirectionDown, painting.AxisDirectionRight, painting.NewEdgeInsetsFromLTRB(10, 20, 30, 40), ui.TextDirectionUnspecified,
			ui.Size{Width: 300, Height: 600}, 0, off(10, 20), ui.Size{Width: 260, Height: 100}, 160, 160, 20},
		{"down scrolled into the padding", painting.AxisDirectionDown, painting.AxisDirectionRight, painting.NewEdgeInsetsFromLTRB(10, 20, 30, 40), ui.TextDirectionUnspecified,
			ui.Size{Width: 300, Height: 600}, 10, off(10, 10), ui.Size{Width: 260, Height: 100}, 160, 150, 10},
		{"up", painting.AxisDirectionUp, painting.AxisDirectionRight, painting.NewEdgeInsetsFromLTRB(10, 20, 30, 40), ui.TextDirectionUnspecified,
			ui.Size{Width: 300, Height: 600}, 0, off(10, 20), ui.Size{Width: 260, Height: 100}, 160, 160, 40},
		{"right", painting.AxisDirectionRight, painting.AxisDirectionDown, painting.NewEdgeInsetsFromLTRB(10, 20, 30, 40), ui.TextDirectionUnspecified,
			ui.Size{Width: 600, Height: 300}, 0, off(10, 20), ui.Size{Width: 100, Height: 240}, 140, 140, 10},
		{"left", painting.AxisDirectionLeft, painting.AxisDirectionDown, painting.NewEdgeInsetsFromLTRB(10, 20, 30, 40), ui.TextDirectionUnspecified,
			ui.Size{Width: 600, Height: 300}, 0, off(10, 20), ui.Size{Width: 100, Height: 240}, 140, 140, 30},
		{"directional padding in a right to left text", painting.AxisDirectionDown, painting.AxisDirectionLeft,
			painting.NewEdgeInsetsDirectionalFromSTEB(5, 0, 25, 0), ui.TextDirectionRTL,
			ui.Size{Width: 300, Height: 600}, 0, off(25, 0), ui.Size{Width: 270, Height: 100}, 100, 100, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			box := newRenderSizedBox(1000, 100)
			if tc.axis == painting.AxisDirectionLeft || tc.axis == painting.AxisDirectionRight {
				box = newRenderSizedBox(100, 1000)
			}
			adapter := rendering.NewRenderSliverToBoxAdapter(box)
			padding := rendering.NewRenderSliverPadding(tc.padding, adapter, rendering.RenderSliverPaddingOpts{TextDirection: tc.direction})
			root := noCacheViewport(tc.cross, tc.axis, rendering.NewViewportOffsetFixed(tc.offset), padding)
			layout(t, newHarness(t), root, withConstraints(rendering.NewBoxConstraintsTight(tc.viewport)))
			if got := rendering.SliverPhysicalParentDataOf(adapter).PaintOffset; got != tc.wantOffset {
				t.Errorf("paint offset of the child = %v, want %v", got, tc.wantOffset)
			}
			if box.Size() != tc.wantChild {
				t.Errorf("box size = %v, want %v", box.Size(), tc.wantChild)
			}
			g := padding.Geometry()
			if g.ScrollExtent != tc.wantScroll || g.PaintExtent != tc.wantPaint {
				t.Errorf("geometry = %v, want scrollExtent %v and paintExtent %v", g, tc.wantScroll, tc.wantPaint)
			}
			if got := padding.ChildScrollOffset(adapter).Value(); got != padding.BeforePadding() {
				t.Errorf("ChildScrollOffset = %v, want %v", got, padding.BeforePadding())
			}
			if got := padding.ChildMainAxisPosition(adapter); got != tc.wantChildPos {
				t.Errorf("ChildMainAxisPosition = %v, want %v", got, tc.wantChildPos)
			}
		})
	}
}

func TestSliverPaddingWithoutChild(t *testing.T) {
	padding := rendering.NewRenderSliverPadding(painting.NewEdgeInsetsSymmetric(10, 5), nil)
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), padding)
	layout(t, newHarness(t), root)
	g := padding.Geometry()
	if g.ScrollExtent != 20 || g.PaintExtent != 20 || g.MaxPaintExtent != 20 {
		t.Errorf("geometry = %v, want 20 for the scroll, paint and maximum paint extent", g)
	}
	if got := padding.MainAxisPadding(); got != 20 {
		t.Errorf("MainAxisPadding = %v, want 20", got)
	}
	if got := padding.CrossAxisPadding(); got != 10 {
		t.Errorf("CrossAxisPadding = %v, want 10", got)
	}
	if got := padding.AfterPadding(); got != 10 {
		t.Errorf("AfterPadding = %v, want 10", got)
	}
}

func TestSliverPaddingSetters(t *testing.T) {
	skipUnlessDebug(t)
	padding := rendering.NewRenderSliverPadding(painting.NewEdgeInsetsAll(10), nil)
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), padding)
	h := newHarness(t)
	layout(t, h, root)
	padding.SetPadding(painting.NewEdgeInsetsAll(10))
	padding.SetTextDirection(ui.TextDirectionUnspecified)
	if padding.DebugNeedsLayout() {
		t.Error("setting the same values marked the sliver as needing layout")
	}
	padding.SetPadding(painting.NewEdgeInsetsAll(20))
	if !padding.DebugNeedsLayout() {
		t.Error("changing the padding did not mark the sliver as needing layout")
	}
	pumpFrame(h, phaseLayout)
	padding.SetTextDirection(ui.TextDirectionRTL)
	if !padding.DebugNeedsLayout() {
		t.Error("changing the text direction did not mark the sliver as needing layout")
	}
	requireTag(t, "PaddingNegative", func() { padding.SetPadding(painting.NewEdgeInsetsAll(-1)) })
	if !padding.ResolvedPadding().IsSome() && padding.Padding() == nil {
		t.Error("the padding was lost")
	}
}
