package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func TestSliverVariedExtentList(t *testing.T) {
	skipUnlessDebug(t)
	extents := []float64{50, 100, 150, 200, 250}
	builder := func(index int, _ rendering.SliverLayoutDimensions) foundation.Opt[float64] {
		if index < len(extents) {
			return foundation.Some(extents[index])
		}
		return foundation.None[float64]()
	}
	boxes := sizedBoxes(len(extents), 300, 10)
	manager := newTestChildManager(asBoxes(boxes)...)
	sliver := rendering.NewRenderSliverVariedExtentList(manager, builder)
	manager.insert, manager.remove = sliver.Insert, sliver.Remove
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetFixed(120), sliver)
	h := newHarness(t)
	layout(t, h, root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 300})))
	// the viewport shows 120..420: item 1 (50..150) to item 3 (300..500)
	want := map[int]float64{1: -70, 2: 30, 3: 180}
	for i, b := range boxes {
		pos, built := want[i]
		if b.Attached() != built {
			t.Errorf("item %d attached = %v, want %v", i, b.Attached(), built)
			continue
		}
		if !built {
			continue
		}
		if got := b.LocalToGlobal(ui.Offset{}, root).Dy; got != pos {
			t.Errorf("item %d at %v, want %v", i, got, pos)
		}
		if got := b.Size().Height; got != extents[i] {
			t.Errorf("item %d has height %v, want %v", i, got, extents[i])
		}
	}
	if sliver.ItemExtent().IsSome() {
		t.Error("a varied extent list has no item extent")
	}
	if got := sliver.IndexToLayoutOffset(3); got != 300 {
		t.Errorf("IndexToLayoutOffset(3) = %v, want 300", got)
	}
	if got := sliver.GetMinChildIndexForScrollOffset(120); got != 1 {
		t.Errorf("GetMinChildIndexForScrollOffset(120) = %d, want 1", got)
	}
	if got := sliver.GetMaxChildIndexForScrollOffset(420); got != 3 {
		t.Errorf("GetMaxChildIndexForScrollOffset(420) = %d, want 3", got)
	}
	if got := sliver.ComputeMaxScrollOffset(sliver.SliverConstraints()); got != 750 {
		t.Errorf("ComputeMaxScrollOffset = %v, want 750", got)
	}
	if got := sliver.PaintExtentOf(boxes[2]); got != 150 {
		t.Errorf("PaintExtentOf(item 2) = %v, want 150", got)
	}
	if d := sliver.LayoutDimensions(); d.ScrollOffset != 120 || d.ViewportMainAxisExtent != 300 || d.CrossAxisExtent != 300 {
		t.Errorf("LayoutDimensions = %v", d)
	}
	sliver.SetItemExtentBuilder(builder)
	if !sliver.DebugNeedsLayout() {
		t.Error("setting the builder did not mark the sliver as needing layout")
	}
}

func TestSliverFixedExtentListSetItemExtent(t *testing.T) {
	skipUnlessDebug(t)
	manager := newTestChildManager(asBoxes(sizedBoxes(10, 300, 10))...)
	sliver := manager.createFixedExtentList(50)
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), sliver)
	h := newHarness(t)
	layout(t, h, root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 200})))
	if got := sliver.Geometry().ScrollExtent; got != 500 {
		t.Errorf("scrollExtent = %v, want 500", got)
	}
	sliver.SetItemExtent(50)
	if sliver.DebugNeedsLayout() {
		t.Error("setting the same extent marked the sliver as needing layout")
	}
	sliver.SetItemExtent(100)
	pumpFrame(h, phaseLayout)
	if got := sliver.Geometry().ScrollExtent; got != 1000 {
		t.Errorf("scrollExtent = %v, want 1000", got)
	}
}

func TestSliverFillViewportSemanticsChildren(t *testing.T) {
	tests := []struct {
		name     string
		implicit bool
		want     int
	}{
		{"implicit scrolling allowed", true, 3},
		{"implicit scrolling not allowed", false, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager := newTestChildManager(asBoxes(sizedBoxes(5, 300, 10))...)
			sliver := rendering.NewRenderSliverFillViewport(manager, rendering.RenderSliverFillViewportOpts{AllowImplicitScrolling: foundation.Some(tc.implicit)})
			manager.insert, manager.remove = sliver.Insert, sliver.Remove
			root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetFixed(0), rendering.RenderViewportOpts{
				RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentViewport(1))},
				Children:               []rendering.Sliver{sliver},
			})
			layout(t, newHarness(t), root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 200})))
			// the cache extent is the viewport: the items 0 and 1 are built, and item 2 starts at the edge
			count := 0
			sliver.VisitChildrenForSemantics(func(rendering.RenderObject) { count++ })
			want := tc.want
			if tc.implicit {
				want = 0
				sliver.VisitChildren(func(rendering.RenderObject) { want++ })
			}
			if count != want {
				t.Errorf("visited %d children for semantics, want %d", count, want)
			}
			if sliver.AllowImplicitScrolling() != tc.implicit || sliver.ViewportFraction() != 1 {
				t.Errorf("AllowImplicitScrolling %v, ViewportFraction %v", sliver.AllowImplicitScrolling(), sliver.ViewportFraction())
			}
		})
	}
}

func TestSliverFillViewportFraction(t *testing.T) {
	manager := newTestChildManager(asBoxes(sizedBoxes(10, 300, 10))...)
	sliver := rendering.NewRenderSliverFillViewport(manager, rendering.RenderSliverFillViewportOpts{ViewportFraction: foundation.Some(0.5)})
	manager.insert, manager.remove = sliver.Insert, sliver.Remove
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), sliver)
	h := newHarness(t)
	layout(t, h, root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 200})))
	if got := sliver.ItemExtent().Value(); got != 100 {
		t.Errorf("itemExtent = %v, want 100", got)
	}
	sliver.SetViewportFraction(0.25)
	pumpFrame(h, phaseLayout)
	if got := sliver.ItemExtent().Value(); got != 50 {
		t.Errorf("itemExtent = %v, want 50", got)
	}
	sliver.SetAllowImplicitScrolling(false)
	sliver.SetViewportFraction(0.25)
}
