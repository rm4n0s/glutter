package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func min80() rendering.Box {
	return rendering.NewRenderConstrainedBox(rendering.BoxConstraints{MinWidth: 80, MinHeight: 80, MaxWidth: inf, MaxHeight: inf}, nil)
}

func TestWrapToStringDeep(t *testing.T) {
	skipUnlessDebug(t)
	wrap := rendering.NewRenderWrap(nil)
	want := "RenderWrap#00000 NEEDS-LAYOUT NEEDS-PAINT DETACHED\n" +
		"   parentData: MISSING\n" +
		"   constraints: MISSING\n" +
		"   size: MISSING\n" +
		"   direction: horizontal\n" +
		"   alignment: start\n" +
		"   spacing: 0.0\n" +
		"   runAlignment: start\n" +
		"   runSpacing: 0.0\n" +
		"   crossAxisAlignment: 0.0\n"
	if got := deepInfo(wrap); got != want {
		t.Errorf("toStringDeep:\n%s\nwant:\n%s", got, want)
	}
}

func TestWrapIntrinsics(t *testing.T) {
	tests := []struct {
		name     string
		children int
		vertical bool
		extents  []float64
		want     []float64
	}{
		{"height, three boxes", 3, false, []float64{245, 250, 80, 79}, []float64{165, 80, 250, 250}},
		{"width, three boxes", 3, true, []float64{245, 250, 80, 79}, []float64{165, 80, 250, 250}},
		{"height, one run", 1, false, []float64{100, 79, 80}, []float64{80, 80, 80}},
		{"width, one run", 1, true, []float64{100, 79, 80}, []float64{80, 80, 80}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wrap := rendering.NewRenderWrap(nil, rendering.RenderWrapOpts{Spacing: 5, RunSpacing: 5})
			for i := 0; i < tc.children; i++ {
				wrap.Add(min80())
			}
			if tc.vertical {
				wrap.SetDirection(painting.AxisVertical)
			}
			for i, extent := range tc.extents {
				var maxV, minV float64
				if tc.vertical {
					maxV, minV = wrap.ComputeMaxIntrinsicWidth(extent), wrap.ComputeMinIntrinsicWidth(extent)
				} else {
					maxV, minV = wrap.ComputeMaxIntrinsicHeight(extent), wrap.ComputeMinIntrinsicHeight(extent)
				}
				if maxV != tc.want[i] || minV != tc.want[i] {
					t.Errorf("extent %v: max %v min %v, want %v", extent, maxV, minV, tc.want[i])
				}
			}
		})
	}
}

func TestWrapRespectsClipBehavior(t *testing.T) {
	for _, tc := range clipCases(ui.ClipNone) {
		t.Run(tc.clip.String(), func(t *testing.T) {
			wrap := rendering.NewRenderWrap([]rendering.Box{newSizedBox(200, 200)},
				rendering.RenderWrapOpts{TextDirection: ui.TextDirectionLTR, ClipBehavior: tc.clip})
			h := newHarness(t)
			layout(t, h, wrap, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}), phaseComposite))
			if len(h.Errors) != 0 {
				t.Errorf("errors = %v", h.Errors)
			}
			if got := observedClip(t, h, wrap); got != tc.want {
				t.Errorf("clip = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWrapLayout(t *testing.T) {
	tests := []struct {
		name string
		opts rendering.RenderWrapOpts
		want []ui.Offset // offsets of five 40x20 boxes in a 100 wide wrap
		size ui.Size
	}{
		{"start", rendering.RenderWrapOpts{}, []ui.Offset{off(0, 0), off(40, 0), off(0, 20), off(40, 20), off(0, 40)}, ui.Size{Width: 80, Height: 60}},
		{"spacing", rendering.RenderWrapOpts{Spacing: 10, RunSpacing: 5}, []ui.Offset{off(0, 0), off(50, 0), off(0, 25), off(50, 25), off(0, 50)}, ui.Size{Width: 90, Height: 70}},
		{"end", rendering.RenderWrapOpts{Alignment: rendering.WrapAlignmentEnd}, []ui.Offset{off(0, 0), off(40, 0), off(0, 20), off(40, 20), off(40, 40)}, ui.Size{Width: 80, Height: 60}},
		{"rtl", rendering.RenderWrapOpts{TextDirection: ui.TextDirectionRTL}, []ui.Offset{off(40, 0), off(0, 0), off(40, 20), off(0, 20), off(40, 40)}, ui.Size{Width: 80, Height: 60}},
		{"vertical direction up", rendering.RenderWrapOpts{VerticalDirection: painting.VerticalDirectionUp, RunAlignment: rendering.WrapAlignmentEnd}, []ui.Offset{off(0, 40), off(40, 40), off(0, 20), off(40, 20), off(0, 0)}, ui.Size{Width: 80, Height: 60}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.opts
			if opts.TextDirection == ui.TextDirectionUnspecified {
				opts.TextDirection = ui.TextDirectionLTR
			}
			var boxes []rendering.Box
			for i := 0; i < 5; i++ {
				boxes = append(boxes, newSizedBox(40, 20))
			}
			// Each box gets loose constraints so that it keeps its size.
			wrap := rendering.NewRenderWrap(boxes, opts)
			for _, b := range boxes {
				b.(*tBox).sizedByParent = false
			}
			layout(t, newHarness(t), wrap, withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 300}))
			if wrap.Size() != tc.size {
				t.Errorf("size = %v, want %v", wrap.Size(), tc.size)
			}
			for i, b := range boxes {
				if got := offsetOf(b); got != tc.want[i] {
					t.Errorf("box %d offset = %v, want %v", i, got, tc.want[i])
				}
			}
		})
	}
}

func TestWrapMainAxisAlignment(t *testing.T) {
	tests := []struct {
		name      string
		alignment rendering.WrapAlignment
		want      [2]float64
	}{
		{"start", rendering.WrapAlignmentStart, [2]float64{0, 40}},
		{"end", rendering.WrapAlignmentEnd, [2]float64{20, 60}},
		{"center", rendering.WrapAlignmentCenter, [2]float64{10, 50}},
		{"spaceBetween", rendering.WrapAlignmentSpaceBetween, [2]float64{0, 60}},
		{"spaceAround", rendering.WrapAlignmentSpaceAround, [2]float64{5, 55}},
		{"spaceEvenly", rendering.WrapAlignmentSpaceEvenly, [2]float64{6.666666666666667, 53.333333333333336}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, b := newTBox(40, 20), newTBox(40, 20)
			wrap := rendering.NewRenderWrap([]rendering.Box{a, b}, rendering.RenderWrapOpts{Alignment: tc.alignment, TextDirection: ui.TextDirectionLTR})
			layout(t, newHarness(t), wrap, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 20})))
			if math.Abs(offsetOf(a).Dx-tc.want[0]) > 1e-9 || math.Abs(offsetOf(b).Dx-tc.want[1]) > 1e-9 {
				t.Errorf("offsets = %v, %v, want x %v and %v", offsetOf(a), offsetOf(b), tc.want[0], tc.want[1])
			}
		})
	}
}

func TestWrapCrossAxisAlignment(t *testing.T) {
	tests := []struct {
		alignment rendering.WrapCrossAlignment
		want      float64
	}{
		{rendering.WrapCrossAlignmentUnspecified, 0},
		{rendering.WrapCrossAlignmentStart, 0},
		{rendering.WrapCrossAlignmentEnd, 20},
		{rendering.WrapCrossAlignmentCenter, 10},
	}
	for _, tc := range tests {
		t.Run(tc.alignment.String(), func(t *testing.T) {
			tall, short := newTBox(40, 40), newTBox(40, 20)
			wrap := rendering.NewRenderWrap([]rendering.Box{tall, short},
				rendering.RenderWrapOpts{CrossAxisAlignment: tc.alignment, TextDirection: ui.TextDirectionLTR})
			layout(t, newHarness(t), wrap, withConstraints(rendering.BoxConstraints{MaxWidth: 100, MaxHeight: 100}))
			if got := offsetOf(short).Dy; got != tc.want {
				t.Errorf("short box dy = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWrapDryLayoutAndBaseline(t *testing.T) {
	skipUnlessDebug(t)
	boxes := []rendering.Box{newBaselineGridBox(3, atBottom), newBaselineGridBox(3, atTop)}
	wrap := rendering.NewRenderWrap(boxes, rendering.RenderWrapOpts{TextDirection: ui.TextDirectionLTR})
	c := rendering.BoxConstraints{MaxWidth: 50, MaxHeight: 100}
	if got := wrap.GetDryLayout(c); got != (ui.Size{Width: 30, Height: 20}) {
		t.Errorf("dry layout = %v, want 30x20", got)
	}
	if got := wrap.GetDryBaseline(c, ui.TextBaselineAlphabetic); got != foundation.Some(10.0) {
		t.Errorf("dry baseline = %v, want 10", got)
	}
	empty := rendering.NewRenderWrap(nil, rendering.RenderWrapOpts{TextDirection: ui.TextDirectionLTR})
	if empty.GetDryBaseline(c, ui.TextBaselineAlphabetic).IsSome() {
		t.Error("an empty wrap has a baseline")
	}
}

func TestWrapNoChildrenAndMissingDirection(t *testing.T) {
	wrap := rendering.NewRenderWrap(nil, rendering.RenderWrapOpts{TextDirection: ui.TextDirectionLTR})
	layout(t, newHarness(t), wrap, withConstraints(rendering.BoxConstraints{MinWidth: 5, MinHeight: 6, MaxWidth: 50, MaxHeight: 60}))
	if wrap.Size() != (ui.Size{Width: 5, Height: 6}) {
		t.Errorf("empty size = %v, want 5x6", wrap.Size())
	}
	skipUnlessDebug(t)
	noDir := rendering.NewRenderWrap([]rendering.Box{newTBox(1, 1), newTBox(1, 1)})
	requireTag(t, "WrapDirectionMissing", func() { layout(t, newHarness(t), noDir) })
}
