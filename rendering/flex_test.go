package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// square100 is BoxConstraints.tightFor(width: 100, height: 100).
var square100 = rendering.BoxConstraints{MinWidth: 100, MaxWidth: 100, MinHeight: 100, MaxHeight: 100}

func newSquare() *rendering.RenderConstrainedBox {
	return rendering.NewRenderConstrainedBox(square100, nil)
}

func flexData(b rendering.Box) *rendering.FlexParentData {
	return b.ParentData().(*rendering.FlexParentData)
}

func offsetOf(b rendering.Box) ui.Offset { return rendering.BoxParentDataOf(b).Offset }

func off(x, y float64) ui.Offset { return ui.Offset{Dx: x, Dy: y} }

func global(b rendering.Box) ui.Offset { return b.LocalToGlobal(ui.OffsetZero, nil) }

func newTextFlex(opts rendering.RenderFlexOpts, children ...rendering.Box) *rendering.RenderFlex {
	if opts.TextDirection == ui.TextDirectionUnspecified {
		opts.TextDirection = ui.TextDirectionLTR
	}
	return rendering.NewRenderFlex(children, opts)
}

func TestFlexOverconstrained(t *testing.T) {
	flex := newTextFlex(rendering.RenderFlexOpts{}, rendering.NewRenderProxyBox(nil))
	layout(t, newHarness(t), flex, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 200, Height: 200})))
	if flex.Size() != (ui.Size{Width: 200, Height: 200}) {
		t.Errorf("flex size = %v, want 200x200", flex.Size())
	}
}

func TestFlexInconsequentialOverflowIsIgnored(t *testing.T) {
	const slightlyLarger, slightlySmaller = 438.8571428571429, 438.85714285714283
	box := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{
		Width: foundation.Some(slightlyLarger), Height: foundation.Some(100.0)}), nil)
	flex := newTextFlex(rendering.RenderFlexOpts{MainAxisSize: rendering.MainAxisSizeMin})
	flex.Add(box)
	parent := rendering.NewRenderConstrainedOverflowBox(flex, rendering.RenderConstrainedOverflowBoxOpts{
		MinWidth: some(0), MaxWidth: some(slightlySmaller), MinHeight: some(0), MaxHeight: some(400)})
	h := newHarness(t)
	layout(t, h, parent)
	if flex.Size() != (ui.Size{Width: slightlySmaller, Height: 100}) {
		t.Errorf("flex size = %v", flex.Size())
	}
	pumpFrame(h, phasePaint)
	if len(h.Errors) != 0 {
		t.Errorf("errors = %v, want none", h.Errors)
	}
}

func TestFlexVerticalAndHorizontalOverflowIntrinsics(t *testing.T) {
	tests := []struct {
		name        string
		vertical    bool
		spacing     float64
		minH, minW  float64
		flexedSize  float64
		wantMinMaxH float64
		wantMinMaxW float64
	}{
		{name: "vertical", vertical: true, wantMinMaxH: 200, wantMinMaxW: 0},
		{name: "vertical with spacing", vertical: true, spacing: 16, wantMinMaxH: 216, wantMinMaxW: 0},
		{name: "horizontal", wantMinMaxH: 0, wantMinMaxW: 200},
		{name: "horizontal with spacing", spacing: 12, wantMinMaxH: 0, wantMinMaxW: 212},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flexible := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsExpand(), nil)
			var fixed rendering.Box
			opts := rendering.RenderFlexOpts{Spacing: tc.spacing}
			if tc.vertical {
				opts.Direction = painting.AxisVertical
				fixed = rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Height: foundation.Some(200.0)}), nil)
			} else {
				fixed = rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(200.0)}), nil)
			}
			flex := newTextFlex(opts, fixed, flexible)
			flexData(flexible).Flex = foundation.Some(1)
			layout(t, newHarness(t), flex, withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}))
			if tc.vertical && flexible.Size().Height != 0 || !tc.vertical && flexible.Size().Width != 0 {
				t.Errorf("flexible size = %v, want 0 along the main axis", flexible.Size())
			}
			if got := flex.GetMinIntrinsicHeight(100); got != tc.wantMinMaxH {
				t.Errorf("min intrinsic height = %v, want %v", got, tc.wantMinMaxH)
			}
			if got := flex.GetMaxIntrinsicHeight(100); got != tc.wantMinMaxH {
				t.Errorf("max intrinsic height = %v, want %v", got, tc.wantMinMaxH)
			}
			if got := flex.GetMinIntrinsicWidth(100); got != tc.wantMinMaxW {
				t.Errorf("min intrinsic width = %v, want %v", got, tc.wantMinMaxW)
			}
			if got := flex.GetMaxIntrinsicWidth(100); got != tc.wantMinMaxW {
				t.Errorf("max intrinsic width = %v, want %v", got, tc.wantMinMaxW)
			}
		})
	}
}

func TestFlexVerticalFlippedConstraints(t *testing.T) {
	flex := rendering.NewRenderFlex([]rendering.Box{rendering.NewRenderAspectRatio(1, nil)}, rendering.RenderFlexOpts{Direction: painting.AxisVertical})
	layout(t, newHarness(t), flex, withConstraints(rendering.BoxConstraints{MaxWidth: 1000, MaxHeight: 200}))
	if got := flex.GetMaxIntrinsicWidth(200); got != 0 {
		t.Errorf("max intrinsic width = %v, want 0", got)
	}
}

func TestFlexDefaults(t *testing.T) {
	flex := rendering.NewRenderFlex(nil)
	if flex.CrossAxisAlignment() != rendering.CrossAxisAlignmentCenter || flex.Direction() != painting.AxisHorizontal {
		t.Errorf("defaults = %v / %v", flex.CrossAxisAlignment(), flex.Direction())
	}
	skipUnlessDebug(t)
	want := "RenderFlex#00000 NEEDS-LAYOUT NEEDS-PAINT DETACHED\n" +
		"   parentData: MISSING\n" +
		"   constraints: MISSING\n" +
		"   size: MISSING\n" +
		"   direction: horizontal\n" +
		"   mainAxisAlignment: start\n" +
		"   mainAxisSize: max\n" +
		"   crossAxisAlignment: center\n" +
		"   verticalDirection: down\n" +
		"   spacing: 0.0\n"
	if got := deepInfo(flex); got != want {
		t.Errorf("toStringDeep:\n%s\nwant:\n%s", got, want)
	}
}

func TestFlexParentData(t *testing.T) {
	box1, box2 := rendering.NewRenderProxyBox(nil), rendering.NewRenderProxyBox(nil)
	flex := newTextFlex(rendering.RenderFlexOpts{}, box1, box2)
	h := newHarness(t)
	layout(t, h, flex, withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}))
	if box1.Size() != (ui.Size{}) || box2.Size() != (ui.Size{}) {
		t.Errorf("sizes = %v, %v, want zero", box1.Size(), box2.Size())
	}
	flexData(box2).Flex = foundation.Some(1)
	flex.MarkNeedsLayout()
	pumpFrame(h, phaseLayout)
	if box1.Size() != (ui.Size{}) || box2.Size() != (ui.Size{Width: 100}) {
		t.Errorf("sizes = %v, %v, want 0x0 and 100x0", box1.Size(), box2.Size())
	}
	if got := flexData(box2).String(); got != "offset=Offset(0.0, 0.0); flex=1; fit=null" {
		t.Errorf("parent data = %q", got)
	}
}

func TestFlexStretch(t *testing.T) {
	box1, box2 := rendering.NewRenderProxyBox(nil), rendering.NewRenderProxyBox(nil)
	flex := newTextFlex(rendering.RenderFlexOpts{})
	flex.SetupParentData(box2)
	flexData(box2).Flex = foundation.Some(2)
	flex.AddAll([]rendering.Box{box1, box2})
	h := newHarness(t)
	layout(t, h, flex, withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}))
	check := func(step string, w1, h1, w2, h2 float64) {
		t.Helper()
		if box1.Size() != (ui.Size{Width: w1, Height: h1}) || box2.Size() != (ui.Size{Width: w2, Height: h2}) {
			t.Errorf("%s: sizes = %v, %v, want %vx%v and %vx%v", step, box1.Size(), box2.Size(), w1, h1, w2, h2)
		}
	}
	check("start", 0, 0, 100, 0)
	flex.SetCrossAxisAlignment(rendering.CrossAxisAlignmentStretch)
	pumpFrame(h, phaseLayout)
	check("stretch", 0, 100, 100, 100)
	flex.SetDirection(painting.AxisVertical)
	pumpFrame(h, phaseLayout)
	check("vertical", 100, 0, 100, 100)
}

func TestFlexMainAxisAlignmentWithSpacing(t *testing.T) {
	tests := []struct {
		name       string
		alignment  rendering.MainAxisAlignment
		spacing    float64
		wantX      [3]float64
		wantY      [3]float64
		xTolerance float64
	}{
		{"spaceEvenly", rendering.MainAxisAlignmentSpaceEvenly, 0, [3]float64{50, 200, 350}, [3]float64{25, 150, 275}, 0},
		{"start with spacing", rendering.MainAxisAlignmentStart, 14, [3]float64{0, 114, 228}, [3]float64{0, 114, 228}, 0},
		{"end with spacing", rendering.MainAxisAlignmentEnd, 14, [3]float64{172, 286, 400}, [3]float64{72, 186, 300}, 0},
		{"center with spacing", rendering.MainAxisAlignmentCenter, 14, [3]float64{86, 200, 314}, [3]float64{36, 150, 264}, 0},
		{"spaceEvenly with spacing", rendering.MainAxisAlignmentSpaceEvenly, 14, [3]float64{43, 200, 357}, [3]float64{18, 150, 282}, 0},
		{"spaceAround with spacing", rendering.MainAxisAlignmentSpaceAround, 14, [3]float64{28.6, 200, 371.3}, [3]float64{12, 150, 288}, 0.1},
		{"spaceBetween with spacing", rendering.MainAxisAlignmentSpaceBetween, 14, [3]float64{0, 200, 400}, [3]float64{0, 150, 300}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			boxes := []rendering.Box{newSquare(), newSquare(), newSquare()}
			flex := newTextFlex(rendering.RenderFlexOpts{MainAxisAlignment: tc.alignment, Spacing: tc.spacing}, boxes...)
			h := newHarness(t)
			layout(t, h, flex, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 400}))
			for i, b := range boxes {
				if math.Abs(offsetOf(b).Dx-tc.wantX[i]) > math.Max(tc.xTolerance, 1e-9) || b.Size().Width != 100 {
					t.Errorf("horizontal box %d: offset %v size %v, want dx %v", i, offsetOf(b), b.Size(), tc.wantX[i])
				}
			}
			flex.SetDirection(painting.AxisVertical)
			pumpFrame(h, phaseLayout)
			for i, b := range boxes {
				if offsetOf(b).Dy != tc.wantY[i] || b.Size().Height != 100 {
					t.Errorf("vertical box %d: offset %v size %v, want dy %v", i, offsetOf(b), b.Size(), tc.wantY[i])
				}
			}
		})
	}
}

func TestFlexFitLoose(t *testing.T) {
	box1, box2, box3 := newSquare(), newSquare(), newSquare()
	flex := newTextFlex(rendering.RenderFlexOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween}, box1, box2, box3)
	h := newHarness(t)
	layout(t, h, flex, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 400}))
	check := func(step string, dx, w [3]float64) {
		t.Helper()
		for i, b := range []rendering.Box{box1, box2, box3} {
			if offsetOf(b).Dx != dx[i] || b.Size().Width != w[i] {
				t.Errorf("%s: box %d offset %v size %v, want dx %v width %v", step, i+1, offsetOf(b), b.Size(), dx[i], w[i])
			}
		}
	}
	check("fixed", [3]float64{0, 200, 400}, [3]float64{100, 100, 100})
	d := flexData(box1)
	d.Flex, d.Fit = foundation.Some(1), rendering.FlexFitLoose
	flex.MarkNeedsLayout()
	pumpFrame(h, phaseLayout)
	check("loose", [3]float64{0, 200, 400}, [3]float64{100, 100, 100})
	box1.SetAdditionalConstraints(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(1000.0), Height: foundation.Some(100.0)}))
	pumpFrame(h, phaseLayout)
	check("loose, big child", [3]float64{0, 300, 400}, [3]float64{300, 100, 100})
}

func TestFlexFlexibleWithMainAxisSizeMin(t *testing.T) {
	box1, box2, box3 := newSquare(), newSquare(), newSquare()
	flex := newTextFlex(rendering.RenderFlexOpts{MainAxisSize: rendering.MainAxisSizeMin, MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween}, box1, box2, box3)
	h := newHarness(t)
	layout(t, h, flex, withConstraints(rendering.BoxConstraints{MaxWidth: 500, MaxHeight: 400}))
	check := func(step string, dx, w [3]float64, total float64) {
		t.Helper()
		for i, b := range []rendering.Box{box1, box2, box3} {
			if offsetOf(b).Dx != dx[i] || b.Size().Width != w[i] {
				t.Errorf("%s: box %d offset %v size %v, want dx %v width %v", step, i+1, offsetOf(b), b.Size(), dx[i], w[i])
			}
		}
		if flex.Size().Width != total {
			t.Errorf("%s: flex width = %v, want %v", step, flex.Size().Width, total)
		}
	}
	check("fixed", [3]float64{0, 100, 200}, [3]float64{100, 100, 100}, 300)
	d := flexData(box1)
	d.Flex, d.Fit = foundation.Some(1), rendering.FlexFitTight
	flex.MarkNeedsLayout()
	pumpFrame(h, phaseLayout)
	check("tight", [3]float64{0, 300, 400}, [3]float64{300, 100, 100}, 500)
	d.Fit = rendering.FlexFitLoose
	flex.MarkNeedsLayout()
	pumpFrame(h, phaseLayout)
	check("loose", [3]float64{0, 100, 200}, [3]float64{100, 100, 100}, 300)
}

func TestFlexMainAxisSizeMinInsideUnconstrained(t *testing.T) {
	box1, box2, box3 := newSquare(), newSquare(), newSquare()
	flex := newTextFlex(rendering.RenderFlexOpts{MainAxisSize: rendering.MainAxisSizeMin}, box1, box2, box3)
	parent := rendering.NewRenderConstrainedOverflowBox(flex, rendering.RenderConstrainedOverflowBoxOpts{
		MinWidth: some(0), MaxWidth: some(inf), MinHeight: some(0), MaxHeight: some(400)})
	h := newHarness(t)
	layout(t, h, parent)
	want := func(step string, w float64) {
		t.Helper()
		if flex.Size() != (ui.Size{Width: w, Height: 100}) {
			t.Errorf("%s: flex size = %v, want %vx100", step, flex.Size(), w)
		}
	}
	want("start", 300)
	d := flexData(box2)
	d.Flex, d.Fit = foundation.Some(1), rendering.FlexFitLoose
	flex.MarkNeedsLayout()
	pumpFrame(h, phaseLayout)
	want("loose flexible", 300)
	parent.SetMaxWidth(some(500))
	pumpFrame(h, phaseLayout)
	want("constrained boundaries", 300)
	flex.SetMainAxisSize(rendering.MainAxisSizeMax)
	pumpFrame(h, phaseLayout)
	want("max", 500)
	flex.SetMainAxisSize(rendering.MainAxisSizeMin)
	d.Fit = rendering.FlexFitTight
	flex.MarkNeedsLayout()
	pumpFrame(h, phaseLayout)
	want("tight", 500)
	parent.SetMaxWidth(some(505))
	pumpFrame(h, phaseLayout)
	want("wider", 505)
}

func TestFlexUnboundedFlexIsAnError(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		size rendering.MainAxisSize
		fit  rendering.FlexFit
	}{
		{"max size, tight flexible", rendering.MainAxisSizeMax, rendering.FlexFitUnspecified},
		{"max size, loose flexible", rendering.MainAxisSizeMax, rendering.FlexFitLoose},
		{"min size, tight flexible", rendering.MainAxisSizeMin, rendering.FlexFitTight},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			box1, box2, box3 := newSquare(), newSquare(), newSquare()
			flex := newTextFlex(rendering.RenderFlexOpts{MainAxisSize: tc.size}, box1, box2, box3)
			parent := rendering.NewRenderConstrainedOverflowBox(flex, rendering.RenderConstrainedOverflowBoxOpts{
				MinWidth: some(0), MaxWidth: some(inf), MinHeight: some(0), MaxHeight: some(400)})
			d := flexData(box2)
			d.Flex, d.Fit = foundation.Some(1), tc.fit
			requireTag(t, "RenderFlexUnboundedFlex", func() { layout(t, newHarness(t), parent) })
		})
	}
	t.Run("min size, loose flexible is fine", func(t *testing.T) {
		box1, box2 := newSquare(), newSquare()
		flex := newTextFlex(rendering.RenderFlexOpts{MainAxisSize: rendering.MainAxisSizeMin}, box1, box2)
		parent := rendering.NewRenderConstrainedOverflowBox(flex, rendering.RenderConstrainedOverflowBoxOpts{
			MinWidth: some(0), MaxWidth: some(inf), MinHeight: some(0), MaxHeight: some(400)})
		d := flexData(box2)
		d.Flex, d.Fit = foundation.Some(1), rendering.FlexFitLoose
		h := newHarness(t)
		layout(t, h, parent)
		if len(h.Errors) != 0 {
			t.Errorf("errors = %v", h.Errors)
		}
	})
}

func TestFlexTightlyConstrainedRTL(t *testing.T) {
	boxes := []rendering.Box{newSquare(), newSquare(), newSquare()}
	flex := newTextFlex(rendering.RenderFlexOpts{TextDirection: ui.TextDirectionRTL, MainAxisSize: rendering.MainAxisSizeMin}, boxes...)
	layout(t, newHarness(t), flex)
	if !flex.BoxConstraints().HasTightWidth() {
		t.Error("the flex does not have tight width")
	}
	for i, want := range []ui.Offset{{Dx: 700, Dy: 250}, {Dx: 600, Dy: 250}, {Dx: 500, Dy: 250}} {
		if got := global(boxes[i]); got != want {
			t.Errorf("box %d at %v, want %v", i+1, got, want)
		}
		if boxes[i].Size() != (ui.Size{Width: 100, Height: 100}) {
			t.Errorf("box %d size = %v", i+1, boxes[i].Size())
		}
	}
}

func TestFlexRTLAndDirections(t *testing.T) {
	box1, box2, box3 := newSquare(), newSquare(), newSquare()
	flex := newTextFlex(rendering.RenderFlexOpts{}, box1, box2, box3)
	h := newHarness(t)
	layout(t, h, flex)
	steps := []struct {
		name  string
		apply func()
		pos   [3]ui.Offset
		size  ui.Size
	}{
		{"ltr start", func() {}, [3]ui.Offset{off(0, 250), off(100, 250), off(200, 250)}, ui.Size{Width: 100, Height: 100}},
		{"ltr end", func() { flex.SetMainAxisAlignment(rendering.MainAxisAlignmentEnd) }, [3]ui.Offset{off(500, 250), off(600, 250), off(700, 250)}, ui.Size{Width: 100, Height: 100}},
		{"rtl end", func() { flex.SetTextDirection(ui.TextDirectionRTL) }, [3]ui.Offset{off(200, 250), off(100, 250), off(0, 250)}, ui.Size{Width: 100, Height: 100}},
		{"rtl start", func() { flex.SetMainAxisAlignment(rendering.MainAxisAlignmentStart) }, [3]ui.Offset{off(700, 250), off(600, 250), off(500, 250)}, ui.Size{Width: 100, Height: 100}},
		{"cross start", func() { flex.SetCrossAxisAlignment(rendering.CrossAxisAlignmentStart) }, [3]ui.Offset{off(700, 0), off(600, 0), off(500, 0)}, ui.Size{Width: 100, Height: 100}},
		{"cross end", func() { flex.SetCrossAxisAlignment(rendering.CrossAxisAlignmentEnd) }, [3]ui.Offset{off(700, 500), off(600, 500), off(500, 500)}, ui.Size{Width: 100, Height: 100}},
		{"up, cross end", func() { flex.SetVerticalDirection(painting.VerticalDirectionUp) }, [3]ui.Offset{off(700, 0), off(600, 0), off(500, 0)}, ui.Size{Width: 100, Height: 100}},
		{"up, cross start", func() { flex.SetCrossAxisAlignment(rendering.CrossAxisAlignmentStart) }, [3]ui.Offset{off(700, 500), off(600, 500), off(500, 500)}, ui.Size{Width: 100, Height: 100}},
		{"vertical, up, rtl", func() { flex.SetDirection(painting.AxisVertical) }, [3]ui.Offset{off(700, 500), off(700, 400), off(700, 300)}, ui.Size{Width: 100, Height: 100}},
		{"vertical, cross end", func() { flex.SetCrossAxisAlignment(rendering.CrossAxisAlignmentEnd) }, [3]ui.Offset{off(0, 500), off(0, 400), off(0, 300)}, ui.Size{Width: 100, Height: 100}},
		{"vertical, stretch", func() { flex.SetCrossAxisAlignment(rendering.CrossAxisAlignmentStretch) }, [3]ui.Offset{off(0, 500), off(0, 400), off(0, 300)}, ui.Size{Width: 800, Height: 100}},
		{"vertical, stretch, ltr", func() { flex.SetTextDirection(ui.TextDirectionLTR) }, [3]ui.Offset{off(0, 500), off(0, 400), off(0, 300)}, ui.Size{Width: 800, Height: 100}},
		{"vertical, ltr, cross start", func() { flex.SetCrossAxisAlignment(rendering.CrossAxisAlignmentStart) }, [3]ui.Offset{off(0, 500), off(0, 400), off(0, 300)}, ui.Size{Width: 100, Height: 100}},
		{"vertical, ltr, cross end", func() { flex.SetCrossAxisAlignment(rendering.CrossAxisAlignmentEnd) }, [3]ui.Offset{off(700, 500), off(700, 400), off(700, 300)}, ui.Size{Width: 100, Height: 100}},
		{"vertical, down", func() { flex.SetVerticalDirection(painting.VerticalDirectionDown) }, [3]ui.Offset{off(700, 0), off(700, 100), off(700, 200)}, ui.Size{Width: 100, Height: 100}},
		{"vertical, down, end", func() { flex.SetMainAxisAlignment(rendering.MainAxisAlignmentEnd) }, [3]ui.Offset{off(700, 300), off(700, 400), off(700, 500)}, ui.Size{Width: 100, Height: 100}},
	}
	for _, s := range steps {
		s.apply()
		pumpFrame(h, phaseLayout)
		for i, b := range []rendering.Box{box1, box2, box3} {
			if got := global(b); got != s.pos[i] {
				t.Errorf("%s: box %d at %v, want %v", s.name, i+1, got, s.pos[i])
			}
			if b.Size() != s.size {
				t.Errorf("%s: box %d size = %v, want %v", s.name, i+1, b.Size(), s.size)
			}
		}
	}
}

func TestFlexChildrenWithoutBaselinesAreTopAligned(t *testing.T) {
	box1, box2 := newSquare(), newSquare()
	flex := newTextFlex(rendering.RenderFlexOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentBaseline,
		TextBaseline: ui.TextBaselineAlphabetic, VerticalDirection: painting.VerticalDirectionUp}, box1, box2)
	h := newHarness(t)
	layout(t, h, flex)
	for _, dir := range []painting.VerticalDirection{painting.VerticalDirectionUp, painting.VerticalDirectionDown} {
		flex.SetVerticalDirection(dir)
		pumpFrame(h, phaseLayout)
		if global(box1).Dy != 0 || global(box2).Dy != 0 {
			t.Errorf("%v: boxes at %v and %v, want y 0", dir, global(box1), global(box2))
		}
	}
}

// baselineGridBox is RenderFlowBaselineTestBox of flex_test.dart: a grid of
// 10x10 cells that wraps, with a baseline wherever the placer says.
type baselineGridBox struct {
	rendering.RenderBox
	gridCount int
	placer    func(height float64) foundation.Opt[float64]
}

func newBaselineGridBox(gridCount int, placer func(height float64) foundation.Opt[float64]) *baselineGridBox {
	b := &baselineGridBox{gridCount: gridCount, placer: placer}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

func (b *baselineGridBox) lineGridCount(width float64) int {
	perLine := b.gridCount
	if width < float64(b.gridCount)*10 {
		perLine = int(width / 10)
	}
	return max(1, perLine)
}

func (b *baselineGridBox) lineCount(width float64) int {
	return (b.gridCount + b.lineGridCount(width) - 1) / b.lineGridCount(width)
}

func (b *baselineGridBox) ComputeMinIntrinsicWidth(float64) float64 { return 10 }
func (b *baselineGridBox) ComputeMaxIntrinsicWidth(float64) float64 { return float64(10 * b.gridCount) }
func (b *baselineGridBox) ComputeMinIntrinsicHeight(w float64) float64 {
	return float64(10 * b.lineCount(w))
}
func (b *baselineGridBox) ComputeMaxIntrinsicHeight(w float64) float64 {
	return b.ComputeMinIntrinsicHeight(w)
}
func (b *baselineGridBox) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Constrain(ui.Size{Width: float64(10 * b.lineGridCount(c.MaxWidth)), Height: float64(10 * b.lineCount(c.MaxWidth))})
}
func (b *baselineGridBox) ComputeDryBaseline(c rendering.BoxConstraints, _ ui.TextBaseline) foundation.Opt[float64] {
	return b.placer(b.GetDryLayout(c).Height)
}
func (b *baselineGridBox) ComputeDistanceToActualBaseline(ui.TextBaseline) foundation.Opt[float64] {
	return b.placer(b.Size().Height)
}
func (b *baselineGridBox) PerformLayout() { b.SetSize(b.ComputeDryLayout(b.BoxConstraints())) }

func atTop(float64) foundation.Opt[float64]      { return foundation.Some(0.0) }
func atBottom(h float64) foundation.Opt[float64] { return foundation.Some(h) }
func noBaseline(float64) foundation.Opt[float64] { return foundation.None[float64]() }
func atTen(float64) foundation.Opt[float64]      { return foundation.Some(10.0) }
func baselineFlex(a, b rendering.Box) *rendering.RenderFlex {
	flex := newTextFlex(rendering.RenderFlexOpts{TextBaseline: ui.TextBaselineAlphabetic,
		CrossAxisAlignment: rendering.CrossAxisAlignmentBaseline}, a, b)
	d1, d2 := flexData(a), flexData(b)
	d1.Flex, d1.Fit = foundation.Some(2), rendering.FlexFitTight
	d2.Flex, d2.Fit = foundation.Some(1), rendering.FlexFitLoose
	return flex
}

func TestFlexVerticalBaseline(t *testing.T) {
	skipUnlessDebug(t)
	mk := func() *rendering.RenderConstrainedBox {
		return rendering.NewRenderConstrainedBox(square100, newBaselineGridBox(1, atTen))
	}
	filler := newSquare
	flex := newTextFlex(rendering.RenderFlexOpts{Direction: painting.AxisVertical}, filler(), mk(), filler(), mk(), filler())
	h := newHarness(t)
	layout(t, h, flex, atPhase(layoutOpts{}, phasePaint))
	flexHeight := flex.Size().Height
	dry := func() float64 {
		return flex.GetDryBaseline(flex.BoxConstraints(), ui.TextBaselineAlphabetic).Value()
	}
	if got := dry(); got != 110 {
		t.Errorf("start baseline = %v, want 110", got)
	}
	flex.SetMainAxisAlignment(rendering.MainAxisAlignmentEnd)
	pumpFrame(h, phasePaint)
	if got, want := dry(), flexHeight-400+10; got != want {
		t.Errorf("end baseline = %v, want %v", got, want)
	}
	flex.SetVerticalDirection(painting.VerticalDirectionUp)
	pumpFrame(h, phasePaint)
	if got := dry(); got != 310 {
		t.Errorf("end, up baseline = %v, want 310", got)
	}
	flex.SetMainAxisAlignment(rendering.MainAxisAlignmentStart)
	pumpFrame(h, phasePaint)
	if got, want := dry(), flexHeight-200+10; got != want {
		t.Errorf("start, up baseline = %v, want %v", got, want)
	}
}

func TestFlexIntrinsics(t *testing.T) {
	t.Run("main axis with aspect ratio", func(t *testing.T) {
		box1, box2 := newSquare(), newSquare()
		box3 := rendering.NewRenderAspectRatio(1, newSquare())
		flex := newTextFlex(rendering.RenderFlexOpts{}, box1, box2, box3)
		d := flexData(box2)
		d.Flex, d.Fit = foundation.Some(1), rendering.FlexFitTight
		tests := []struct{ extent, want float64 }{{inf, 300}, {300, 500}, {500, 700}}
		for _, tc := range tests {
			if got := flex.GetMinIntrinsicWidth(tc.extent); got != tc.want {
				t.Errorf("min intrinsic width(%v) = %v, want %v", tc.extent, got, tc.want)
			}
			if got := flex.GetMaxIntrinsicWidth(tc.extent); got != tc.want {
				t.Errorf("max intrinsic width(%v) = %v, want %v", tc.extent, got, tc.want)
			}
		}
	})
	for _, vertical := range []bool{false, true} {
		name := "horizontal with spacing"
		if vertical {
			name = "vertical with spacing"
		}
		t.Run(name, func(t *testing.T) {
			opts := rendering.RenderFlexOpts{Spacing: 16}
			if vertical {
				opts.Direction = painting.AxisVertical
			}
			flex := newTextFlex(opts, newSquare(), newSquare(), newSquare())
			for _, extent := range []float64{inf, 300, 500} {
				main, cross := 332.0, 100.0
				gotMinW, gotMaxW := flex.GetMinIntrinsicWidth(extent), flex.GetMaxIntrinsicWidth(extent)
				gotMinH, gotMaxH := flex.GetMinIntrinsicHeight(extent), flex.GetMaxIntrinsicHeight(extent)
				wantW, wantH := main, cross
				if vertical {
					wantW, wantH = cross, main
				}
				if gotMinW != wantW || gotMaxW != wantW || gotMinH != wantH || gotMaxH != wantH {
					t.Errorf("extent %v: widths %v/%v heights %v/%v, want %v and %v", extent, gotMinW, gotMaxW, gotMinH, gotMaxH, wantW, wantH)
				}
			}
		})
	}
	t.Run("baseline aligned flex computeDryLayout", func(t *testing.T) {
		skipUnlessDebug(t)
		box1, box2 := newBaselineGridBox(10, atTop), newBaselineGridBox(10, atBottom)
		flex := baselineFlex(box1, box2)
		tests := []struct {
			size     ui.Size
			wantSize ui.Size
			wantBase float64
		}{
			{ui.Size{Width: 200, Height: 100}, ui.Size{Width: 200, Height: 30}, 20},
			{ui.Size{Width: 300, Height: 100}, ui.Size{Width: 300, Height: 20}, 10},
		}
		for _, tc := range tests {
			c := rendering.NewBoxConstraintsLoose(tc.size)
			if got := flex.GetDryLayout(c); got != tc.wantSize {
				t.Errorf("dry layout(%v) = %v, want %v", tc.size, got, tc.wantSize)
			}
			if got := flex.GetDryBaseline(c, ui.TextBaselineAlphabetic).Value(); got != tc.wantBase {
				t.Errorf("dry baseline(%v) = %v, want %v", tc.size, got, tc.wantBase)
			}
		}
	})
	t.Run("baseline aligned children cross intrinsic size", func(t *testing.T) {
		flex := baselineFlex(newBaselineGridBox(10, atTop), newBaselineGridBox(10, atBottom))
		for _, tc := range []struct{ width, want float64 }{{200, 30}, {300, 20}} {
			if got := flex.GetMaxIntrinsicHeight(tc.width); got != tc.want {
				t.Errorf("max intrinsic height(%v) = %v, want %v", tc.width, got, tc.want)
			}
			if got := flex.GetMinIntrinsicHeight(tc.width); got != tc.want {
				t.Errorf("min intrinsic height(%v) = %v, want %v", tc.width, got, tc.want)
			}
		}
	})
	t.Run("children with no baselines do not affect the baseline location", func(t *testing.T) {
		skipUnlessDebug(t)
		flex := baselineFlex(newBaselineGridBox(10, atBottom), newBaselineGridBox(10, noBaseline))
		tests := []struct {
			size     ui.Size
			wantSize ui.Size
		}{
			{ui.Size{Width: 200, Height: 100}, ui.Size{Width: 200, Height: 20}},
			{ui.Size{Width: 300, Height: 100}, ui.Size{Width: 300, Height: 10}},
		}
		for _, tc := range tests {
			c := rendering.NewBoxConstraintsLoose(tc.size)
			if got := flex.GetDryLayout(c); got != tc.wantSize {
				t.Errorf("dry layout(%v) = %v, want %v", tc.size, got, tc.wantSize)
			}
			if got := flex.GetDryBaseline(c, ui.TextBaselineAlphabetic).Value(); got != 10 {
				t.Errorf("dry baseline(%v) = %v, want 10", tc.size, got)
			}
		}
	})
}

func TestFlexMethodsThatCheckOverflowWorkWithoutLayout(t *testing.T) {
	skipUnlessDebug(t)
	flex := rendering.NewRenderFlex(nil)
	if s := flex.ToStringShort(); len(s) == 0 || containsStr(s, "OVERFLOWING") {
		t.Errorf("ToStringShort() = %q", s)
	}
	// Layout panics, which leaves the overflow unset; ToStringShort still works.
	requireTag(t, "FlexDirectionMissing", func() { layout(t, newHarness(t), flex, atPhase(layoutOpts{}, phasePaint)) })
	if s := flex.ToStringShort(); containsStr(s, "OVERFLOWING") {
		t.Errorf("ToStringShort() = %q", s)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestFlexNegativeSpacingPanics(t *testing.T) {
	skipUnlessDebug(t)
	requireTag(t, "FlexSpacingNegative", func() {
		rendering.NewRenderFlex([]rendering.Box{rendering.NewRenderProxyBox(nil)},
			rendering.RenderFlexOpts{TextDirection: ui.TextDirectionLTR, Spacing: -15})
	})
}

func TestFlexBaselineMissingPanics(t *testing.T) {
	skipUnlessDebug(t)
	flex := newTextFlex(rendering.RenderFlexOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentBaseline,
		TextBaseline: ui.TextBaselineAlphabetic}, newSquare())
	requireTag(t, "FlexBaselineMissing", func() { flex.SetTextBaseline(ui.TextBaselineUnspecified) })
	noBase := newTextFlex(rendering.RenderFlexOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentBaseline}, newSquare())
	requireTag(t, "FlexBaselineMissing", func() { layout(t, newHarness(t), noBase) })
}

func TestFlexClipBehaviorAndOverflow(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		clip ui.Clip
		want ui.Clip
	}{
		{"default", ui.ClipUnspecified, ui.ClipNone},
		{"none", ui.ClipNone, ui.ClipNone},
		{"hardEdge", ui.ClipHardEdge, ui.ClipHardEdge},
		{"antiAlias", ui.ClipAntiAlias, ui.ClipAntiAlias},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			box := newSizedBox(200, 200)
			flex := rendering.NewRenderFlex([]rendering.Box{box}, rendering.RenderFlexOpts{Direction: painting.AxisVertical, ClipBehavior: tc.clip})
			h := newHarness(t)
			layout(t, h, flex, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}), phaseComposite))
			if flex.ClipBehavior() != tc.want {
				t.Errorf("clip behavior = %v, want %v", flex.ClipBehavior(), tc.want)
			}
			if len(h.Errors) != 1 {
				t.Fatalf("errors = %v, want 1 overflow error", h.Errors)
			}
			requireRoute(t, h.Errors[0], "RenderOverflow")
			if got := flex.ToStringShort(); !containsStr(got, "OVERFLOWING") {
				t.Errorf("ToStringShort() = %q, want OVERFLOWING", got)
			}
			clip := flex.DescribeApproximatePaintClip(box)
			if (tc.want != ui.ClipNone) != clip.IsSome() {
				t.Errorf("approximate paint clip = %v, want set only if clipping", clip)
			}
		})
	}
}
