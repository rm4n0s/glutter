package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func some(v float64) foundation.Opt[float64] { return foundation.Some(v) }

func TestRenderFractionallySizedOverflowBoxConstraints(t *testing.T) {
	leaf := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsExpand(), nil)
	test := rendering.NewRenderFractionallySizedOverflowBox(leaf, rendering.RenderFractionallySizedOverflowBoxOpts{
		WidthFactor: some(2), HeightFactor: some(0.5)})
	root := rendering.NewRenderPositionedBox(rendering.NewRenderConstrainedBox(
		rendering.NewBoxConstraintsTight(ui.Size{Width: 200, Height: 200}), test))
	layout(t, newHarness(t), root)
	for _, c := range []struct {
		name string
		got  ui.Size
		want ui.Size
	}{
		{"root", root.Size(), ui.Size{Width: 800, Height: 600}},
		{"test", test.Size(), ui.Size{Width: 200, Height: 200}},
		{"leaf", leaf.Size(), ui.Size{Width: 400, Height: 100}},
	} {
		if c.got != c.want {
			t.Errorf("%s size = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestRenderAspectRatioIntrinsicSizing(t *testing.T) {
	type intrinsics struct{ minW, maxW, minH, maxH float64 }
	tests := []struct {
		name  string
		ratio float64
		child rendering.Box
		// at 200 and 400, and at infinity
		at200, at400, atInf intrinsics
	}{
		{"2.0 without child", 2, nil, intrinsics{400, 400, 100, 100}, intrinsics{800, 800, 200, 200}, intrinsics{0, 0, 0, 0}},
		{"0.5 without child", 0.5, nil, intrinsics{100, 100, 400, 400}, intrinsics{200, 200, 800, 800}, intrinsics{0, 0, 0, 0}},
		{"2.0 with child", 2, newSizedBox(90, 70), intrinsics{400, 400, 100, 100}, intrinsics{800, 800, 200, 200}, intrinsics{90, 90, 70, 70}},
		{"0.5 with child", 0.5, newSizedBox(90, 70), intrinsics{100, 100, 400, 400}, intrinsics{200, 200, 800, 800}, intrinsics{90, 90, 70, 70}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			box := rendering.NewRenderAspectRatio(tc.ratio, tc.child)
			check := func(arg float64, want intrinsics) {
				got := intrinsics{box.GetMinIntrinsicWidth(arg), box.GetMaxIntrinsicWidth(arg), box.GetMinIntrinsicHeight(arg), box.GetMaxIntrinsicHeight(arg)}
				if got != want {
					t.Errorf("intrinsics at %v = %+v, want %+v", arg, got, want)
				}
			}
			check(200, tc.at200)
			check(400, tc.at400)
			check(inf, tc.atInf)
		})
	}
}

func TestRenderAspectRatioUnbounded(t *testing.T) {
	skipUnlessDebug(t)
	box := rendering.NewRenderConstrainedOverflowBox(rendering.NewRenderAspectRatio(0.5, newSizedBox(90, 70)),
		rendering.RenderConstrainedOverflowBoxOpts{MaxWidth: some(inf), MaxHeight: some(inf)})
	requireTag(t, "AspectRatioUnbounded", func() { layout(t, newHarness(t), box) })
}

func TestRenderAspectRatioSizing(t *testing.T) {
	inside := rendering.NewRenderAspectRatio(1, nil)
	outside := rendering.NewRenderConstrainedOverflowBox(inside)
	h := newHarness(t)
	layout(t, h, outside)
	if inside.Size() != (ui.Size{Width: 800, Height: 600}) {
		t.Fatalf("initial size %v, want 800x600", inside.Size())
	}
	outside.SetMinWidth(some(0))
	outside.SetMinHeight(some(0))
	steps := []struct {
		ratio                  float64
		minW, minH             float64
		maxW, maxH             float64
		want                   ui.Size
		setMin, setRatioBefore bool
	}{
		{1, 0, 0, 100, 90, ui.Size{Width: 90, Height: 90}, false, false},
		{1, 0, 0, 90, 100, ui.Size{Width: 90, Height: 90}, false, false},
		{1, 0, 0, inf, 90, ui.Size{Width: 90, Height: 90}, false, false},
		{1, 0, 0, 90, inf, ui.Size{Width: 90, Height: 90}, false, false},
		{2, 0, 0, 100, 90, ui.Size{Width: 100, Height: 50}, false, true},
		{2, 0, 0, 90, 100, ui.Size{Width: 90, Height: 45}, false, false},
		{2, 0, 0, inf, 90, ui.Size{Width: 180, Height: 90}, false, false},
		{2, 0, 0, 90, inf, ui.Size{Width: 90, Height: 45}, false, false},
		{2, 80, 80, 100, 90, ui.Size{Width: 100, Height: 80}, true, false},
		{2, 80, 80, 90, 100, ui.Size{Width: 90, Height: 80}, true, false},
		{2, 80, 80, inf, 90, ui.Size{Width: 180, Height: 90}, true, false},
		{2, 80, 80, 90, inf, ui.Size{Width: 90, Height: 80}, true, false},
	}
	for i, s := range steps {
		if s.setRatioBefore {
			inside.SetAspectRatio(s.ratio)
		}
		if s.setMin {
			outside.SetMinWidth(some(s.minW))
			outside.SetMinHeight(some(s.minH))
		}
		outside.SetMaxWidth(some(s.maxW))
		outside.SetMaxHeight(some(s.maxH))
		pumpFrame(h, phaseLayout)
		if inside.Size() != s.want {
			t.Errorf("step %d (max %vx%v): size %v, want %v", i, s.maxW, s.maxH, inside.Size(), s.want)
		}
	}
}

func TestRenderLimitedBox(t *testing.T) {
	newChild := func() rendering.Box {
		return rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(
			rendering.TightForOpts{Width: some(300), Height: some(400)}), nil)
	}
	tests := []struct {
		name   string
		parent func(child rendering.Box) (rendering.Box, rendering.Box)
		want   ui.Size
	}{
		{"parent max size is unconstrained", func(c rendering.Box) (rendering.Box, rendering.Box) {
			return rendering.NewRenderConstrainedOverflowBox(rendering.NewRenderLimitedBox(c, rendering.RenderLimitedBoxOpts{MaxWidth: some(100), MaxHeight: some(200)}),
				rendering.RenderConstrainedOverflowBoxOpts{MinWidth: some(0), MaxWidth: some(inf), MinHeight: some(0), MaxHeight: some(inf)}), c
		}, ui.Size{Width: 100, Height: 200}},
		{"parent maxWidth is unconstrained", func(c rendering.Box) (rendering.Box, rendering.Box) {
			return rendering.NewRenderConstrainedOverflowBox(rendering.NewRenderLimitedBox(c, rendering.RenderLimitedBoxOpts{MaxWidth: some(100), MaxHeight: some(200)}),
				rendering.RenderConstrainedOverflowBoxOpts{MinWidth: some(0), MaxWidth: some(inf), MinHeight: some(500), MaxHeight: some(500)}), c
		}, ui.Size{Width: 100, Height: 500}},
		{"parent maxHeight is unconstrained", func(c rendering.Box) (rendering.Box, rendering.Box) {
			return rendering.NewRenderConstrainedOverflowBox(rendering.NewRenderLimitedBox(c, rendering.RenderLimitedBoxOpts{MaxWidth: some(100), MaxHeight: some(200)}),
				rendering.RenderConstrainedOverflowBoxOpts{MinWidth: some(500), MaxWidth: some(500), MinHeight: some(0), MaxHeight: some(inf)}), c
		}, ui.Size{Width: 500, Height: 200}},
		{"no child", func(rendering.Box) (rendering.Box, rendering.Box) {
			box := rendering.NewRenderLimitedBox(nil, rendering.RenderLimitedBoxOpts{MaxWidth: some(100), MaxHeight: some(200)})
			return rendering.NewRenderConstrainedOverflowBox(box, rendering.RenderConstrainedOverflowBoxOpts{
				MinWidth: some(10), MaxWidth: some(500), MinHeight: some(0), MaxHeight: some(inf)}), box
		}, ui.Size{Width: 10, Height: 0}},
		{"no child use parent", func(rendering.Box) (rendering.Box, rendering.Box) {
			box := rendering.NewRenderLimitedBox(nil, rendering.RenderLimitedBoxOpts{MaxWidth: some(100), MaxHeight: some(200)})
			return rendering.NewRenderConstrainedOverflowBox(box, rendering.RenderConstrainedOverflowBoxOpts{MinWidth: some(10)}), box
		}, ui.Size{Width: 10, Height: 600}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parent, measured := tc.parent(newChild())
			layout(t, newHarness(t), parent)
			if measured.Size() != tc.want {
				t.Errorf("size = %v, want %v", measured.Size(), tc.want)
			}
		})
	}
}

func TestRenderLimitedBoxToStringDeep(t *testing.T) {
	skipUnlessDebug(t)
	child := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(
		rendering.TightForOpts{Width: some(300), Height: some(400)}), nil)
	parent := rendering.NewRenderConstrainedOverflowBox(rendering.NewRenderLimitedBox(child, rendering.RenderLimitedBoxOpts{MaxWidth: some(100), MaxHeight: some(200)}),
		rendering.RenderConstrainedOverflowBoxOpts{MinWidth: some(0), MaxWidth: some(inf), MinHeight: some(0), MaxHeight: some(inf)})
	layout(t, newHarness(t), parent)
	want := `RenderConstrainedOverflowBox#00000 NEEDS-PAINT NEEDS-COMPOSITING-BITS-UPDATE
 │ parentData: <none>
 │ constraints: BoxConstraints(w=800.0, h=600.0)
 │ size: Size(800.0, 600.0)
 │ alignment: Alignment.center
 │ minWidth: 0.0
 │ maxWidth: Infinity
 │ minHeight: 0.0
 │ maxHeight: Infinity
 │ fit: max
 │
 └─child: RenderLimitedBox#00000 relayoutBoundary=up1 NEEDS-PAINT NEEDS-COMPOSITING-BITS-UPDATE
   │ parentData: offset=Offset(350.0, 200.0) (can use size)
   │ constraints: BoxConstraints(unconstrained)
   │ size: Size(100.0, 200.0)
   │ maxWidth: 100.0
   │ maxHeight: 200.0
   │
   └─child: RenderConstrainedBox#00000 relayoutBoundary=up2 NEEDS-PAINT
       parentData: <none> (can use size)
       constraints: BoxConstraints(0.0<=w<=100.0, 0.0<=h<=200.0)
       size: Size(100.0, 200.0)
       additionalConstraints: BoxConstraints(w=300.0, h=400.0)
`
	if got := deepInfo(parent); got != want {
		t.Errorf("toStringDeep:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderOffstage(t *testing.T) {
	painted := false
	child := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(
		rendering.TightForOpts{Height: some(10), Width: some(10)}), nil)
	root := rendering.NewRenderPositionedBox(rendering.NewRenderConstrainedBox(
		rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: some(800)}),
		rendering.NewRenderOffstage(rendering.NewRenderCustomPaint(child, rendering.RenderCustomPaintOpts{
			Painter: newCallbackPainter(func() { painted = true })}))))
	if child.HasSize() || painted {
		t.Fatal("the child has a size or was painted before layout")
	}
	layout(t, newHarness(t), root, layoutOpts{phase: phasePaint})
	if !child.HasSize() {
		t.Error("an offstage child is not laid out")
	}
	if painted {
		t.Error("an offstage child was painted")
	}
	if child.Size() != (ui.Size{Width: 800, Height: 10}) {
		t.Errorf("child size = %v, want 800x10", child.Size())
	}
}

// baselineTester is _RenderBaselineTester.
type baselineTester struct {
	rendering.RenderBox
	boxSize                                             ui.Size
	alphabeticBaselineOffset, ideographicBaselineOffset foundation.Opt[float64]
}

func newBaselineTester() *baselineTester {
	b := &baselineTester{}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

func (b *baselineTester) offset(baseline ui.TextBaseline) foundation.Opt[float64] {
	if baseline == ui.TextBaselineAlphabetic {
		return b.alphabeticBaselineOffset
	}
	return b.ideographicBaselineOffset
}
func (b *baselineTester) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	return b.offset(baseline)
}
func (b *baselineTester) ComputeDryBaseline(_ rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	return b.offset(baseline)
}
func (b *baselineTester) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Constrain(b.boxSize)
}
func (b *baselineTester) PerformLayout() { b.SetSize(b.ComputeDryLayout(b.BoxConstraints())) }

func TestRenderBaseline(t *testing.T) {
	child := newSizedBox(100, 100)
	parent := rendering.NewRenderBaseline(0, ui.TextBaselineAlphabetic, child)
	root := rendering.NewRenderPositionedBox(parent, rendering.RenderPositionedBoxOpts{Alignment: alignmentTopLeft})
	h := newHarness(t)
	layout(t, h, root)
	steps := []struct {
		baseline float64
		dy       float64
		size     ui.Size
	}{
		{0, -100, ui.Size{Width: 100, Height: 0}},
		{25, -75, ui.Size{Width: 100, Height: 25}},
		{90, -10, ui.Size{Width: 100, Height: 90}},
		{100, 0, ui.Size{Width: 100, Height: 100}},
		{110, 10, ui.Size{Width: 100, Height: 110}},
	}
	for _, s := range steps {
		parent.SetBaseline(s.baseline)
		pumpFrame(h, phaseLayout)
		if off := rendering.BoxParentDataOf(child).Offset; off != (ui.Offset{Dy: s.dy}) {
			t.Errorf("baseline %v: offset %v, want (0, %v)", s.baseline, off, s.dy)
		}
		if parent.Size() != s.size {
			t.Errorf("baseline %v: size %v, want %v", s.baseline, parent.Size(), s.size)
		}
	}
}

func TestRenderBaselineDifferentBaselineTypes(t *testing.T) {
	child := newBaselineTester()
	child.boxSize = ui.Size{Width: 100, Height: 100}
	child.alphabeticBaselineOffset = some(50)
	child.ideographicBaselineOffset = some(60)
	box := rendering.NewRenderBaseline(1, ui.TextBaselineAlphabetic, child)
	h := newHarness(t)
	layout(t, h, box, layoutOpts{phase: phasePaint})
	loose := rendering.BoxConstraints{MaxWidth: inf, MaxHeight: inf}
	if got := box.GetDryBaseline(loose, ui.TextBaselineAlphabetic); got != some(1) {
		t.Errorf("alphabetic dry baseline = %v, want 1", got)
	}
	if got := box.GetDryBaseline(loose, ui.TextBaselineIdeographic); got != some(11) {
		t.Errorf("ideographic dry baseline = %v, want 11", got)
	}
	child.alphabeticBaselineOffset = foundation.None[float64]()
	child.ideographicBaselineOffset = foundation.None[float64]()
	child.MarkNeedsLayout() // clears the baseline cache
	box.MarkNeedsLayout()
	pumpFrame(h, phasePaint)
	for _, bl := range []ui.TextBaseline{ui.TextBaselineAlphabetic, ui.TextBaselineIdeographic} {
		if got := box.GetDryBaseline(loose, bl); got.IsSome() {
			t.Errorf("%v dry baseline = %v, want none", bl, got)
		}
	}
}
