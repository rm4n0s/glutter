package rendering_test

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func TestSliverFixedExtentListRoundingError(t *testing.T) {
	children := sizedBoxes(3, 400, 100)
	manager := newTestChildManager(asBoxes(children)...)
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), manager.createFillViewport())
	h := newHarness(t)
	layout(t, h, root)
	if !children[0].Attached() || children[1].Attached() {
		t.Fatalf("attached = %v %v, want true false", children[0].Attached(), children[1].Attached())
	}

	root.SetOffset(rendering.NewViewportOffsetFixed(600))
	pumpFrame(h, phaseLayout)
	if children[0].Attached() || !children[1].Attached() {
		t.Fatalf("attached = %v %v, want false true", children[0].Attached(), children[1].Attached())
	}

	// Simulate double precision error.
	root.SetOffset(rendering.NewViewportOffsetFixed(1199.999999999998))
	pumpFrame(h, phaseLayout)
	if children[1].Attached() || !children[2].Attached() {
		t.Fatalf("attached = %v %v, want false true", children[1].Attached(), children[2].Attached())
	}
}

// testFixedExtentAdaptor is TestRenderSliverFixedExtentBoxAdaptor.
type testFixedExtentAdaptor struct {
	rendering.RenderSliverFixedExtentBoxAdaptor
	extent float64
	// totalExtent, when it is some, replaces the scroll extent that the layout
	// computed (NonMultipleFixedExtentList)
	totalExtent foundation.Opt[float64]
}

func newTestFixedExtentAdaptor(extent float64) *testFixedExtentAdaptor {
	return newTestFixedExtentAdaptorWith(extent, newTestChildManager())
}

func newTestFixedExtentAdaptorWith(extent float64, manager rendering.RenderSliverBoxChildManager) *testFixedExtentAdaptor {
	a := &testFixedExtentAdaptor{extent: extent}
	rendering.InitRenderSliverFixedExtentBoxAdaptor(&a.RenderSliverFixedExtentBoxAdaptor, a, manager)
	return a
}

func (a *testFixedExtentAdaptor) PerformLayout() {
	a.RenderSliverFixedExtentBoxAdaptor.PerformLayout()
	if total, ok := a.totalExtent.Get(); ok {
		a.SetGeometry(a.Geometry().CopyWith(rendering.SliverGeometryCopy{ScrollExtent: foundation.Some(total), MaxPaintExtent: foundation.Some(total)}))
	}
}

func (a *testFixedExtentAdaptor) ItemExtent() foundation.Opt[float64] {
	return foundation.Some(a.extent)
}
func (a *testFixedExtentAdaptor) ItemExtentBuilder() rendering.ItemExtentBuilder { return nil }
func (a *testFixedExtentAdaptor) Paint(*rendering.PaintingContext, ui.Offset)    {}

func TestGetMaxChildIndexForScrollOffset(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/68182
	const (
		genericItemExtent                   = 600.0
		extraValueToNotHaveRoundingIssues   = 1e-10
		extraValueToHaveRoundingIssues      = 1e-11
		itemExtentForAProblematicScreenSize = 411.42857142857144
		anotherGenericItemExtent            = 414.0
		offsetValueWhichDoesNotCare         = 1234.0
	)
	tests := []struct {
		name       string
		offset     float64
		itemExtent float64
		want       int
	}{
		{"0 when item extent is 0", offsetValueWhichDoesNotCare, 0, 0},
		{"0 when offset is 0", 0, genericItemExtent, 0},
		{"0 when offset is equal to item extent", genericItemExtent, genericItemExtent, 0},
		{"1 when offset is greater than item extent", genericItemExtent + 1, genericItemExtent, 1},
		{"1 when offset is slightly greater than item extent", genericItemExtent + extraValueToNotHaveRoundingIssues, genericItemExtent, 1},
		{"4 when offset is four times and a half greater than item extent", genericItemExtent * 4.5, genericItemExtent, 4},
		{"5 when offset is 6 times greater than item extent", anotherGenericItemExtent * 6, anotherGenericItemExtent, 5},
		{"5 when offset is 6 times greater than an item extent where the division returns more than 13 zero decimals",
			itemExtentForAProblematicScreenSize*6 + extraValueToHaveRoundingIssues, itemExtentForAProblematicScreenSize, 5},
		{"0 when offset is a bit greater than item extent", genericItemExtent + extraValueToHaveRoundingIssues, genericItemExtent, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := newTestFixedExtentAdaptor(tc.itemExtent).GetMaxChildIndexForScrollOffset(tc.offset); got != tc.want {
				t.Errorf("GetMaxChildIndexForScrollOffset(%v) = %d, want %d", tc.offset, got, tc.want)
			}
		})
	}
}

func TestSliverFixedExtentPaintsChild(t *testing.T) {
	children := sizedBoxes(3, 400, 100)
	manager := newTestChildManager(asBoxes(children)...)
	root := noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), manager.createFillViewport())
	h := newHarness(t)
	layout(t, h, root)
	parent, ok := children[0].Parent().(interface {
		PaintsChild(rendering.RenderObject) bool
	})
	if _, isAdaptor := children[0].Parent().(*rendering.RenderSliverFillViewport); !ok || !isAdaptor {
		t.Fatalf("parent is %T, want a RenderSliverMultiBoxAdaptor", children[0].Parent())
	}
	steps := []struct {
		offset float64
		want   [3]bool
	}{
		{0, [3]bool{true, false, false}},
		{600, [3]bool{false, true, false}},
		{1200, [3]bool{false, false, true}},
	}
	for i, s := range steps {
		if i > 0 {
			root.SetOffset(rendering.NewViewportOffsetFixed(s.offset))
			pumpFrame(h, phaseLayout)
		}
		for j, c := range children {
			if got := parent.PaintsChild(c); got != s.want[j] {
				t.Errorf("offset %v: PaintsChild(children[%d]) = %v, want %v", s.offset, j, got, s.want[j])
			}
		}
	}
}

func TestSliverFillViewportReferencesItemExtent(t *testing.T) {
	tests := []struct {
		name             string
		offset           float64
		wantMin, wantMax int
	}{
		{"non-zero offset", 1200, 2, 1},
		{"zero offset", 0, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager := newTestChildManager(asBoxes(sizedBoxes(3, 400, 100))...)
			sliver := manager.createFillViewport()
			root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetFixed(tc.offset), rendering.RenderViewportOpts{
				RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(100))},
				Children:               []rendering.Sliver{sliver},
			})
			layout(t, newHarness(t), root)
			// The true item extent is 600 to fill the viewport.
			if got := sliver.SliverConstraints().ScrollOffset; got != tc.offset {
				t.Errorf("scrollOffset = %v, want %v", got, tc.offset)
			}
			if got := sliver.ItemExtent().Value(); got != 600 {
				t.Errorf("itemExtent = %v, want 600", got)
			}
			if got := sliver.IndexToLayoutOffset(10); got != 6000 {
				t.Errorf("IndexToLayoutOffset(10) = %v, want 6000", got)
			}
			if got := sliver.GetMinChildIndexForScrollOffset(sliver.SliverConstraints().ScrollOffset); got != tc.wantMin {
				t.Errorf("GetMinChildIndexForScrollOffset = %d, want %d", got, tc.wantMin)
			}
			if got := sliver.GetMaxChildIndexForScrollOffset(sliver.SliverConstraints().ScrollOffset); got != tc.wantMax {
				t.Errorf("GetMaxChildIndexForScrollOffset = %d, want %d", got, tc.wantMax)
			}
			if got := sliver.ComputeMaxScrollOffset(sliver.SliverConstraints()); got != 1800 {
				t.Errorf("ComputeMaxScrollOffset = %v, want 1800", got)
			}
		})
	}
}

func TestSliverFixedExtentListReferencesItemExtent(t *testing.T) {
	tests := []struct {
		name             string
		offset           float64
		wantMin, wantMax int
	}{
		{"non-zero offset", 45, 1, 1},
		{"zero offset", 0, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager := newTestChildManager(asBoxes(sizedBoxes(3, 400, 100))...)
			sliver := manager.createFixedExtentList(30)
			root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetFixed(tc.offset), rendering.RenderViewportOpts{
				RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(100))},
				Children:               []rendering.Sliver{sliver},
			})
			layout(t, newHarness(t), root)
			// The true item extent is 30.
			c := sliver.SliverConstraints()
			if c.ScrollOffset != tc.offset || c.ViewportMainAxisExtent != 600 {
				t.Errorf("scrollOffset %v and viewportMainAxisExtent %v, want %v and 600", c.ScrollOffset, c.ViewportMainAxisExtent, tc.offset)
			}
			if got := sliver.ItemExtent().Value(); got != 30 {
				t.Errorf("itemExtent = %v, want 30", got)
			}
			if got := sliver.IndexToLayoutOffset(10); got != 300 {
				t.Errorf("IndexToLayoutOffset(10) = %v, want 300", got)
			}
			if got := sliver.GetMinChildIndexForScrollOffset(c.ScrollOffset); got != tc.wantMin {
				t.Errorf("GetMinChildIndexForScrollOffset = %d, want %d", got, tc.wantMin)
			}
			if got := sliver.GetMaxChildIndexForScrollOffset(c.ScrollOffset); got != tc.wantMax {
				t.Errorf("GetMaxChildIndexForScrollOffset = %d, want %d", got, tc.wantMax)
			}
			if got := sliver.ComputeMaxScrollOffset(c); got != 90 {
				t.Errorf("ComputeMaxScrollOffset = %v, want 90", got)
			}
		})
	}
}

func TestSliverMultiBoxAdaptorGarbage(t *testing.T) {
	manager := newTestChildManager(asBoxes(sizedBoxes(3, 400, 100))...)
	sliver := manager.createFixedExtentList(30)
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{
		RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(100))},
		Children:               []rendering.Sliver{sliver},
	})
	layout(t, newHarness(t), root)
	// There are 3 children. If I want to garbage collect based on keeping only
	// the middle child, then I should get 1 for leading and 1 for trailing.
	if got := sliver.CalculateLeadingGarbage(1); got != 1 {
		t.Errorf("CalculateLeadingGarbage(1) = %d, want 1", got)
	}
	if got := sliver.CalculateTrailingGarbage(1); got != 1 {
		t.Errorf("CalculateTrailingGarbage(1) = %d, want 1", got)
	}
}

// testChildManagerSimple is TestChildManagerSimple.
type testChildManagerSimple struct {
	rendering.RenderSliverBoxChildManagerDefaults
	insert func(child, after rendering.Box)
	child  rendering.Box
}

func (m *testChildManagerSimple) CreateChild(index int, after rendering.Box) {
	if index == 0 && m.child != nil {
		m.insert(m.child, after)
	}
}
func (m *testChildManagerSimple) RemoveChild(rendering.Box) {}
func (m *testChildManagerSimple) EstimateMaxScrollOffset(rendering.SliverConstraints, rendering.EstimateMaxScrollOffsetOpts) float64 {
	return 0
}
func (m *testChildManagerSimple) ChildCount() int { return 1 }
func (m *testChildManagerSimple) DidAdoptChild(child rendering.Box) {
	rendering.SliverMultiBoxAdaptorParentDataOf(child).Index = foundation.Some(0)
}
func (m *testChildManagerSimple) SetDidUnderflow(bool) {}

// layoutNonMultiple lays out a fixed extent list whose scroll extent is
// totalExtent and returns the errors that the layout reported.
func layoutNonMultiple(t *testing.T, totalExtent float64) []error {
	t.Helper()
	manager := &testChildManagerSimple{}
	sliver := newTestFixedExtentAdaptorWith(100, manager)
	sliver.totalExtent = foundation.Some(totalExtent)
	manager.insert = sliver.Insert
	manager.child = newRenderSizedBox(400, 100)
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Children: []rendering.Sliver{sliver}})
	h := renderingtest.New(t, renderingtest.Opts{Size: ui.Size{Width: 800, Height: 600}, CollectPanics: true})
	layout(t, h, root)
	return h.Errors
}

func TestSliverFixedExtentBoxAdaptorAssertion(t *testing.T) {
	skipUnlessDebug(t)
	errs := layoutNonMultiple(t, 250) // Not a multiple of 100
	if len(errs) == 0 {
		t.Fatal("no error was reported")
	}
	appErr, ok := errors.FromError(errs[0])
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", errs[0])
	}
	if appErr.Tag != "SliverFixedExtentNotMultiple" {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestSliverFixedExtentBoxAdaptorRoundingTolerance(t *testing.T) {
	if errs := layoutNonMultiple(t, 200.0000000000001); len(errs) != 0 {
		t.Errorf("errors reported: %v", errs)
	}
}
