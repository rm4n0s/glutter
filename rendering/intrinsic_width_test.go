package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// testBox is RenderTestBox of intrinsic_width_test.dart: a box that reports the
// given intrinsic dimensions and is sized by its parent to the middle of them.
type testBox struct {
	rendering.RenderBox
	dims rendering.BoxConstraints
}

func newTestBox(dims rendering.BoxConstraints) *testBox {
	b := &testBox{dims: dims}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

func (b *testBox) ComputeMinIntrinsicWidth(float64) float64  { return b.dims.MinWidth }
func (b *testBox) ComputeMaxIntrinsicWidth(float64) float64  { return b.dims.MaxWidth }
func (b *testBox) ComputeMinIntrinsicHeight(float64) float64 { return b.dims.MinHeight }
func (b *testBox) ComputeMaxIntrinsicHeight(float64) float64 { return b.dims.MaxHeight }
func (b *testBox) SizedByParent() bool                       { return true }
func (b *testBox) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Constrain(ui.Size{
		Width:  b.dims.MinWidth + (b.dims.MaxWidth-b.dims.MinWidth)/2,
		Height: b.dims.MinHeight + (b.dims.MaxHeight-b.dims.MinHeight)/2})
}

var intrinsicArgs = []float64{0, 10, 80, inf}

func TestRenderIntrinsicSizing(t *testing.T) {
	dims := rendering.BoxConstraints{MinWidth: 10, MaxWidth: 100, MinHeight: 20, MaxHeight: 200}
	loose := rendering.BoxConstraints{MinWidth: 5, MinHeight: 8, MaxWidth: 500, MaxHeight: 800}
	rect := func(minW, minH, maxW, maxH float64) rendering.BoxConstraints {
		return rendering.BoxConstraints{MinWidth: minW, MinHeight: minH, MaxWidth: maxW, MaxHeight: maxH}
	}
	type result struct{ minW, maxW, minH, maxH float64 }
	tests := []struct {
		name        string
		make        func(child rendering.Box) rendering.Box
		child       bool
		constraints rendering.BoxConstraints
		size        ui.Size
		childSize   ui.Size
		intrinsics  result
	}{
		{"shrink-wrapping width", func(c rendering.Box) rendering.Box { return rendering.NewRenderIntrinsicWidth(c) }, true, loose,
			ui.Size{Width: 100, Height: 110}, ui.Size{Width: 100, Height: 110}, result{100, 100, 20, 200}},
		{"IntrinsicWidth without a child", func(rendering.Box) rendering.Box { return rendering.NewRenderIntrinsicWidth(nil) }, false, loose,
			ui.Size{Width: 5, Height: 8}, ui.Size{}, result{0, 0, 0, 0}},
		{"stepped width", func(c rendering.Box) rendering.Box {
			return rendering.NewRenderIntrinsicWidth(c, rendering.RenderIntrinsicWidthOpts{StepWidth: some(47)})
		}, true, loose, ui.Size{Width: 3 * 47, Height: 110}, ui.Size{Width: 3 * 47, Height: 110}, result{3 * 47, 3 * 47, 20, 200}},
		{"stepped height", func(c rendering.Box) rendering.Box {
			return rendering.NewRenderIntrinsicWidth(c, rendering.RenderIntrinsicWidthOpts{StepHeight: some(47)})
		}, true, loose, ui.Size{Width: 100, Height: 235}, ui.Size{Width: 100, Height: 235}, result{100, 100, 47, 5 * 47}},
		{"stepped everything", func(c rendering.Box) rendering.Box {
			return rendering.NewRenderIntrinsicWidth(c, rendering.RenderIntrinsicWidthOpts{StepHeight: some(47), StepWidth: some(37)})
		}, true, loose, ui.Size{Width: 3 * 37, Height: 235}, ui.Size{Width: 3 * 37, Height: 235}, result{3 * 37, 3 * 37, 47, 5 * 47}},
		{"IntrinsicWidth, loose constraints smaller than the child", func(c rendering.Box) rendering.Box { return rendering.NewRenderIntrinsicWidth(c) },
			true, rect(50, 8, 70, 800), ui.Size{Width: 70, Height: 110}, ui.Size{Width: 70, Height: 110}, result{100, 100, 20, 200}},
		{"IntrinsicWidth, tight constraints larger than the child", func(c rendering.Box) rendering.Box { return rendering.NewRenderIntrinsicWidth(c) },
			true, rect(500, 8, 500, 800), ui.Size{Width: 500, Height: 110}, ui.Size{Width: 500, Height: 110}, result{100, 100, 20, 200}},
		{"IntrinsicWidth, tight constraints smaller than the child", func(c rendering.Box) rendering.Box { return rendering.NewRenderIntrinsicWidth(c) },
			true, rect(50, 8, 50, 800), ui.Size{Width: 50, Height: 110}, ui.Size{Width: 50, Height: 110}, result{100, 100, 20, 200}},
		{"shrink-wrapping height", func(c rendering.Box) rendering.Box { return rendering.NewRenderIntrinsicHeight(c) }, true, loose,
			ui.Size{Width: 55, Height: 200}, ui.Size{Width: 55, Height: 200}, result{10, 100, 200, 200}},
		{"IntrinsicHeight without a child", func(rendering.Box) rendering.Box { return rendering.NewRenderIntrinsicHeight(nil) }, false, loose,
			ui.Size{Width: 5, Height: 8}, ui.Size{}, result{0, 0, 0, 0}},
		{"IntrinsicHeight, loose constraints smaller than the child", func(c rendering.Box) rendering.Box { return rendering.NewRenderIntrinsicHeight(c) },
			true, rect(5, 8, 500, 80), ui.Size{Width: 55, Height: 80}, ui.Size{Width: 55, Height: 80}, result{10, 100, 200, 200}},
		{"IntrinsicHeight, tight constraints larger than the child", func(c rendering.Box) rendering.Box { return rendering.NewRenderIntrinsicHeight(c) },
			true, rect(5, 400, 500, 400), ui.Size{Width: 55, Height: 400}, ui.Size{Width: 55, Height: 400}, result{10, 100, 200, 200}},
		{"IntrinsicHeight, tight constraints smaller than the child", func(c rendering.Box) rendering.Box { return rendering.NewRenderIntrinsicHeight(c) },
			true, rect(5, 80, 500, 80), ui.Size{Width: 55, Height: 80}, ui.Size{Width: 55, Height: 80}, result{10, 100, 200, 200}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var child rendering.Box
			if tc.child {
				child = newTestBox(dims)
			}
			parent := tc.make(child)
			layout(t, newHarness(t), parent, withConstraints(tc.constraints))
			if parent.Size() != tc.size {
				t.Errorf("parent size = %v, want %v", parent.Size(), tc.size)
			}
			if tc.child && child.Size() != tc.childSize {
				t.Errorf("child size = %v, want %v", child.Size(), tc.childSize)
			}
			for _, arg := range intrinsicArgs {
				got := result{parent.GetMinIntrinsicWidth(arg), parent.GetMaxIntrinsicWidth(arg), parent.GetMinIntrinsicHeight(arg), parent.GetMaxIntrinsicHeight(arg)}
				if got != tc.intrinsics {
					t.Errorf("intrinsics at %v = %+v, want %+v", arg, got, tc.intrinsics)
				}
			}
		})
	}
}

func TestRenderPaddingIntrinsics(t *testing.T) {
	tests := []struct {
		name  string
		child func() rendering.Box
		want  map[float64]float64 // by argument; the others are `def`
		def   float64
	}{
		{"boring child", func() rendering.Box { return newSizedBox(20, 20) }, nil, 50},
		{"aspect ratio child", func() rendering.Box { return rendering.NewRenderAspectRatio(1, nil) }, map[float64]float64{80: 80}, 30},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			box := rendering.NewRenderPadding(painting.NewEdgeInsetsAll(15), tc.child())
			for _, arg := range intrinsicArgs {
				want := tc.def
				if v, ok := tc.want[arg]; ok {
					want = v
				}
				for name, got := range map[string]float64{
					"minWidth": box.GetMinIntrinsicWidth(arg), "maxWidth": box.GetMaxIntrinsicWidth(arg),
					"minHeight": box.GetMinIntrinsicHeight(arg), "maxHeight": box.GetMaxIntrinsicHeight(arg)} {
					if got != want {
						t.Errorf("%s at %v = %v, want %v", name, arg, got, want)
					}
				}
			}
			// also a smoke test
			layout(t, newHarness(t), box, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 10, Height: 10})))
		})
	}
}
