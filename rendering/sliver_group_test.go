package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// listOf is a sliver list of n boxes of the height.
func listOf(n int, height float64) (*rendering.RenderSliverList, []*RenderSizedBox) {
	boxes := sizedBoxes(n, 300, height)
	return newTestChildManager(asBoxes(boxes)...).createList(), boxes
}

// groupViewport lays out the sliver in a viewport of 300x600.
func groupViewport(t *testing.T, offset float64, sliver rendering.Sliver) (*rendering.RenderViewport, func(float64)) {
	t.Helper()
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetFixed(offset), sliver)
	h := newHarness(t)
	layout(t, h, root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 600})))
	return root, func(o float64) {
		root.SetOffset(rendering.NewViewportOffsetFixed(o))
		pumpFrame(h, phaseLayout)
	}
}

func TestSliverCrossAxisGroupLayout(t *testing.T) {
	tests := []struct {
		name         string
		flex         [2]int // 0 for a sliver without flex
		maxExtent    [2]float64
		wantCross    [2]float64
		wantOffsetsX [2]float64
	}{
		{"equal flex", [2]int{1, 1}, [2]float64{}, [2]float64{150, 150}, [2]float64{0, 150}},
		{"expanded 3 and 2", [2]int{3, 2}, [2]float64{}, [2]float64{180, 120}, [2]float64{0, 180}},
		{"constrained then expanded", [2]int{0, 1}, [2]float64{60, 0}, [2]float64{60, 240}, [2]float64{0, 60}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			first, _ := listOf(20, 300)
			second, _ := listOf(20, 200)
			children := [2]rendering.Sliver{first, second}
			if tc.maxExtent[0] > 0 {
				children[0] = rendering.NewRenderSliverConstrainedCrossAxis(tc.maxExtent[0], first)
			}
			group := rendering.NewRenderSliverCrossAxisGroup(children[0], children[1])
			for i, child := range children {
				if tc.flex[i] > 0 {
					rendering.SliverPhysicalParentDataOf(child).CrossAxisFlex = foundation.Some(tc.flex[i])
				} else {
					rendering.SliverPhysicalParentDataOf(child).CrossAxisFlex = foundation.None[int]()
				}
			}
			groupViewport(t, 0, group)
			for i, s := range []*rendering.RenderSliverList{first, second} {
				if got := s.SliverConstraints().CrossAxisExtent; got != tc.wantCross[i] {
					t.Errorf("sliver %d cross axis extent = %v, want %v", i, got, tc.wantCross[i])
				}
				if got := rendering.SliverPhysicalParentDataOf(children[i]).PaintOffset.Dx; got != tc.wantOffsetsX[i] {
					t.Errorf("sliver %d paint offset = %v, want %v", i, got, tc.wantOffsetsX[i])
				}
			}
			// The group is as long as its longest child: 20 items of 300
			if got := group.Geometry().ScrollExtent; got != 300*20 {
				t.Errorf("scrollExtent = %v, want %v", got, 300*20)
			}
		})
	}
}

func TestSliverCrossAxisGroupScrolls(t *testing.T) {
	first, firstBoxes := listOf(20, 300)
	second, secondBoxes := listOf(20, 200)
	group := rendering.NewRenderSliverCrossAxisGroup(first, second)
	_, scrollTo := groupViewport(t, 0, group)
	if !firstBoxes[0].Attached() || !firstBoxes[1].Attached() || firstBoxes[2].Attached() {
		t.Error("the first list builds items 0 and 1")
	}
	if !secondBoxes[0].Attached() || !secondBoxes[2].Attached() || secondBoxes[3].Attached() {
		t.Error("the second list builds items 0 to 2")
	}
	scrollTo(18 * 300)
	// Item 17 ends exactly at the scroll offset, so the list keeps it.
	if firstBoxes[16].Attached() || !firstBoxes[17].Attached() || !firstBoxes[18].Attached() || !firstBoxes[19].Attached() {
		t.Error("the first list builds items 17 to 19")
	}
	// The second list is scrolled past its end: it keeps only the last item, to
	// know where its content ends.
	for i, b := range secondBoxes {
		if b.Attached() != (i == 19) {
			t.Errorf("second list: item %d attached = %v", i, b.Attached())
		}
	}
}

func TestSliverMainAxisGroupLayout(t *testing.T) {
	first, firstBoxes := listOf(2, 100)
	second, secondBoxes := listOf(3, 100)
	group := rendering.NewRenderSliverMainAxisGroup(first, second)
	_, scrollTo := groupViewport(t, 0, group)
	g := group.Geometry()
	if g.ScrollExtent != 500 || g.PaintExtent != 500 {
		t.Errorf("geometry = %v, want scrollExtent and paintExtent 500", g)
	}
	// want maps the index of an item to its position; the items that are not in it
	// are not built
	check := func(step string, boxes []*RenderSizedBox, want map[int]float64) {
		t.Helper()
		for i, b := range boxes {
			pos, built := want[i]
			if b.Attached() != built {
				t.Errorf("%s: item %d attached = %v, want %v", step, i, b.Attached(), built)
				continue
			}
			if built {
				if got := b.LocalToGlobal(ui.Offset{}, nil).Dy; got != pos {
					t.Errorf("%s: item %d at %v, want %v", step, i, got, pos)
				}
			}
		}
	}
	check("start, first", firstBoxes, map[int]float64{0: 0, 1: 100})
	check("start, second", secondBoxes, map[int]float64{0: 200, 1: 300, 2: 400})
	scrollTo(150)
	check("scrolled, first", firstBoxes, map[int]float64{1: -50})
	check("scrolled, second", secondBoxes, map[int]float64{0: 50, 1: 150, 2: 250})
	if got := group.ChildScrollOffset(second).Value(); got != 200 {
		t.Errorf("ChildScrollOffset(second) = %v, want 200", got)
	}
	if got := group.ChildMainAxisPosition(second); got != 50 {
		t.Errorf("ChildMainAxisPosition(second) = %v, want 50", got)
	}
}

func TestSliverMainAxisGroupFailsForSliverAfterInfiniteOne(t *testing.T) {
	skipUnlessDebug(t)
	infinite := newTestRenderSliver(geometryOf(rendering.SliverGeometryOpts{ScrollExtent: math.Inf(1), PaintExtent: 600, MaxPaintExtent: math.Inf(1)}))
	next := newTestRenderSliver(rendering.SliverGeometryZero)
	group := rendering.NewRenderSliverMainAxisGroup(infinite, next)
	requireTag(t, "SliverUnreachableSliver", func() { group.Layout(validSliverConstraints()) })
}

func TestSliverCrossAxisGroupFailures(t *testing.T) {
	skipUnlessDebug(t)
	extent := func(e float64) rendering.SliverGeometry {
		return geometryOf(rendering.SliverGeometryOpts{ScrollExtent: 10, PaintExtent: 10, MaxPaintExtent: 10, CrossAxisExtent: foundation.Some(e)})
	}
	noFlex := func(s rendering.Sliver) rendering.Sliver {
		rendering.SliverPhysicalParentDataOf(s).CrossAxisFlex = foundation.None[int]()
		return s
	}
	tests := []struct {
		name string
		tag  string
		make func() (*rendering.RenderSliverCrossAxisGroup, rendering.SliverConstraints)
	}{
		{"infinite cross axis extent", "SliverCrossAxisGroupExtentInfinite", func() (*rendering.RenderSliverCrossAxisGroup, rendering.SliverConstraints) {
			c := validSliverConstraints()
			c.CrossAxisExtent = math.Inf(1)
			return rendering.NewRenderSliverCrossAxisGroup(newTestRenderSliver(extent(10))), c
		}},
		{"child without flex and without an extent", "SliverCrossAxisGroupChildExtentMissing", func() (*rendering.RenderSliverCrossAxisGroup, rendering.SliverConstraints) {
			g := rendering.NewRenderSliverCrossAxisGroup()
			child := newTestRenderSliver(rendering.SliverGeometryZero)
			g.Add(child)
			noFlex(child)
			return g, validSliverConstraints()
		}},
		{"children use all the extent", "SliverCrossAxisGroupOutOfExtent", func() (*rendering.RenderSliverCrossAxisGroup, rendering.SliverConstraints) {
			g := rendering.NewRenderSliverCrossAxisGroup()
			a, b := newTestRenderSliver(extent(100)), newTestRenderSliver(extent(100))
			g.Add(a)
			g.Add(b)
			noFlex(a)
			noFlex(b)
			return g, validSliverConstraints()
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, c := tc.make()
			requireTag(t, tc.tag, func() { g.Layout(c) })
		})
	}
}
