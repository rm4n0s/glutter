package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func TestSliverGridDelegates(t *testing.T) {
	constraints := func(crossExtent float64, crossDirection painting.AxisDirection) rendering.SliverConstraints {
		c := validSliverConstraints()
		c.CrossAxisExtent = crossExtent
		c.CrossAxisDirection = crossDirection
		return c
	}
	tests := []struct {
		name     string
		delegate rendering.SliverGridDelegate
		c        rendering.SliverConstraints
		want     rendering.SliverGridRegularTileLayout
	}{
		{"fixed count", rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{MainAxisSpacing: 10, CrossAxisSpacing: 20}),
			constraints(300, painting.AxisDirectionRight),
			rendering.SliverGridRegularTileLayout{CrossAxisCount: 2, MainAxisStride: 150, CrossAxisStride: 160, ChildMainAxisExtent: 140, ChildCrossAxisExtent: 140}},
		{"fixed count with an aspect ratio", rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{ChildAspectRatio: foundation.Some(4.0)}),
			constraints(400, painting.AxisDirectionRight),
			rendering.SliverGridRegularTileLayout{CrossAxisCount: 2, MainAxisStride: 50, CrossAxisStride: 200, ChildMainAxisExtent: 50, ChildCrossAxisExtent: 200}},
		{"fixed count with a main axis extent", rendering.NewSliverGridDelegateWithFixedCrossAxisCount(3, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{MainAxisExtent: foundation.Some(70.0)}),
			constraints(300, painting.AxisDirectionLeft),
			rendering.SliverGridRegularTileLayout{CrossAxisCount: 3, MainAxisStride: 70, CrossAxisStride: 100, ChildMainAxisExtent: 70, ChildCrossAxisExtent: 100, ReverseCrossAxis: true}},
		{"max cross axis extent", rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(100),
			constraints(300, painting.AxisDirectionRight),
			rendering.SliverGridRegularTileLayout{CrossAxisCount: 3, MainAxisStride: 100, CrossAxisStride: 100, ChildMainAxisExtent: 100, ChildCrossAxisExtent: 100}},
		{"max cross axis extent that does not divide", rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(120, rendering.SliverGridDelegateWithMaxCrossAxisExtentOpts{CrossAxisSpacing: 10, MainAxisSpacing: 5}),
			constraints(300, painting.AxisDirectionRight),
			rendering.SliverGridRegularTileLayout{CrossAxisCount: 3, MainAxisStride: 280.0/3 + 5, CrossAxisStride: 280.0/3 + 10, ChildMainAxisExtent: 280.0 / 3, ChildCrossAxisExtent: 280.0 / 3}},
		{"max cross axis extent larger than the viewport", rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(1000),
			constraints(300, painting.AxisDirectionRight),
			rendering.SliverGridRegularTileLayout{CrossAxisCount: 1, MainAxisStride: 300, CrossAxisStride: 300, ChildMainAxisExtent: 300, ChildCrossAxisExtent: 300}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.delegate.GetLayout(tc.c).(*rendering.SliverGridRegularTileLayout)
			if *got != tc.want {
				t.Errorf("layout = %+v, want %+v", *got, tc.want)
			}
		})
	}
}

func TestSliverGridDelegateShouldRelayout(t *testing.T) {
	base := rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2)
	tests := []struct {
		name  string
		other rendering.SliverGridDelegate
		want  bool
	}{
		{"same values", rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2), false},
		{"count", rendering.NewSliverGridDelegateWithFixedCrossAxisCount(3), true},
		{"main axis spacing", rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{MainAxisSpacing: 1}), true},
		{"cross axis spacing", rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{CrossAxisSpacing: 1}), true},
		{"aspect ratio", rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{ChildAspectRatio: foundation.Some(2.0)}), true},
		{"main axis extent", rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{MainAxisExtent: foundation.Some(2.0)}), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.other.ShouldRelayout(base); got != tc.want {
				t.Errorf("ShouldRelayout = %v, want %v", got, tc.want)
			}
		})
	}
	maxBase := rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(100)
	if rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(100).ShouldRelayout(maxBase) {
		t.Error("equal max extent delegates should not relayout")
	}
	if !rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(101).ShouldRelayout(maxBase) {
		t.Error("different max extent delegates should relayout")
	}
}

func TestSliverGridRegularTileLayout(t *testing.T) {
	l := rendering.NewSliverGridRegularTileLayout(3, 50, 100, 40, 90, false)
	tests := []struct {
		offset           float64
		wantMin, wantMax int
	}{
		{0, 0, 0},
		{49, 0, 2},
		{50, 3, 2},
		{51, 3, 5},
		{175, 9, 11},
	}
	for _, tc := range tests {
		if got := l.GetMinChildIndexForScrollOffset(tc.offset); got != tc.wantMin {
			t.Errorf("GetMinChildIndexForScrollOffset(%v) = %d, want %d", tc.offset, got, tc.wantMin)
		}
		if got := l.GetMaxChildIndexForScrollOffset(tc.offset); got != tc.wantMax {
			t.Errorf("GetMaxChildIndexForScrollOffset(%v) = %d, want %d", tc.offset, got, tc.wantMax)
		}
	}
	if got := l.GetGeometryForChildIndex(7); got != (rendering.SliverGridGeometry{ScrollOffset: 100, CrossAxisOffset: 100, MainAxisExtent: 40, CrossAxisExtent: 90}) {
		t.Errorf("geometry of child 7 = %v", got)
	}
	if got := l.GetGeometryForChildIndex(7).TrailingScrollOffset(); got != 140 {
		t.Errorf("trailing scroll offset = %v, want 140", got)
	}
	// 7 children make 3 rows: 3 strides minus the spacing after the last row
	if got := l.ComputeMaxScrollOffset(7); got != 140 {
		t.Errorf("ComputeMaxScrollOffset(7) = %v, want 140", got)
	}
	if got := l.ComputeMaxScrollOffset(0); got != 0 {
		t.Errorf("ComputeMaxScrollOffset(0) = %v, want 0", got)
	}
	reversed := rendering.NewSliverGridRegularTileLayout(3, 50, 100, 40, 90, true)
	if got := reversed.GetGeometryForChildIndex(1).CrossAxisOffset; got != 100 {
		t.Errorf("reversed cross axis offset of child 1 = %v, want 100", got)
	}
	if got := reversed.GetGeometryForChildIndex(0).CrossAxisOffset; got != 200 {
		t.Errorf("reversed cross axis offset of child 0 = %v, want 200", got)
	}
	if got := (rendering.SliverGridGeometry{ScrollOffset: 1, CrossAxisOffset: 2, MainAxisExtent: 3, CrossAxisExtent: 4}).String(); got !=
		"SliverGridGeometry(scrollOffset: 1.0, crossAxisOffset: 2.0, mainAxisExtent: 3.0, crossAxisExtent: 4.0)" {
		t.Errorf("String() = %q", got)
	}
}

func TestSliverGridLayout(t *testing.T) {
	tests := []struct {
		name  string
		cross painting.AxisDirection
		want  [6]ui.Offset
	}{
		{"right", painting.AxisDirectionRight, [6]ui.Offset{off(0, 0), off(160, 0), off(0, 150), off(160, 150), off(0, 300), off(160, 300)}},
		{"left", painting.AxisDirectionLeft, [6]ui.Offset{off(160, 0), off(0, 0), off(160, 150), off(0, 150), off(160, 300), off(0, 300)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			boxes := sizedBoxes(6, 10, 10)
			manager := newTestChildManager(asBoxes(boxes)...)
			delegate := rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{MainAxisSpacing: 10, CrossAxisSpacing: 20})
			grid := manager.createGrid(delegate)
			root := rendering.NewRenderViewport(tc.cross, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Children: []rendering.Sliver{grid}})
			layout(t, newHarness(t), root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 600})))
			for i, b := range boxes {
				if got := b.LocalToGlobal(ui.Offset{}, root); got != tc.want[i] {
					t.Errorf("child %d at %v, want %v", i, got, tc.want[i])
				}
				if b.Size() != (ui.Size{Width: 140, Height: 140}) {
					t.Errorf("child %d has size %v, want 140x140", i, b.Size())
				}
			}
			// 3 rows of 140 with 10 between them
			if got := grid.Geometry().ScrollExtent; got != 440 {
				t.Errorf("scrollExtent = %v, want 440", got)
			}
			if got := rendering.SliverMultiBoxAdaptorParentDataOf(boxes[1]).LayoutOffset.Value(); got != 0 {
				t.Errorf("layout offset of child 1 = %v, want 0", got)
			}
			if got := grid.ChildCrossAxisPosition(boxes[1]); got != tc.want[1].Dx {
				t.Errorf("cross axis position of child 1 = %v, want %v", got, tc.want[1].Dx)
			}
		})
	}
}

func TestSliverGridSetGridDelegate(t *testing.T) {
	manager := newTestChildManager(asBoxes(sizedBoxes(4, 10, 10))...)
	delegate := rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2)
	grid := manager.createGrid(delegate)
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Children: []rendering.Sliver{grid}})
	h := newHarness(t)
	layout(t, h, root)
	if grid.GridDelegate() != rendering.SliverGridDelegate(delegate) {
		t.Fatal("GridDelegate() is not the delegate")
	}
	skipUnlessDebug(t)
	grid.SetGridDelegate(delegate)
	grid.SetGridDelegate(rendering.NewSliverGridDelegateWithFixedCrossAxisCount(2))
	if grid.DebugNeedsLayout() {
		t.Error("a delegate that does not say it should relayout marked the grid as needing layout")
	}
	grid.SetGridDelegate(rendering.NewSliverGridDelegateWithFixedCrossAxisCount(3))
	if !grid.DebugNeedsLayout() {
		t.Error("a delegate that says it should relayout did not mark the grid as needing layout")
	}
	pumpFrame(h, phaseLayout)
	grid.SetGridDelegate(rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(100))
	if !grid.DebugNeedsLayout() {
		t.Error("a delegate of another type did not mark the grid as needing layout")
	}
}
