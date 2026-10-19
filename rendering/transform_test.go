package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func roundOffset(o ui.Offset) ui.Offset {
	return ui.Offset{Dx: math.Round(o.Dx) + 0, Dy: math.Round(o.Dy) + 0}
}

// rotateAroundXAxis is the 3D rotation transform with alpha a.
func rotateAroundXAxis(a float64) *ui.Matrix4 {
	const x, y, z = 1.0, 0.0, 0.0
	sc := math.Sin(a/2) * math.Cos(a/2)
	sq := math.Sin(a/2) * math.Sin(a/2)
	f := func(v float64) float32 { return float32(v) }
	return ui.NewMatrix4FromList([16]float32{
		f(1 - 2*(y*y+z*z)*sq), f(2 * (x*y*sq + z*sc)), f(2 * (x*z*sq - y*sc)), 0,
		f(2 * (x*y*sq - z*sc)), f(1 - 2*(x*x+z*z)*sq), f(2 * (y*z*sq + x*sc)), 0,
		f(2 * (x*z*sq + y*sc)), f(2 * (y*z*sq - x*sc)), f(1 - 2*(x*x+z*z)*sq), 0,
		0, 0, 0, 1,
	})
}

func rotateAroundXAxis90Degrees() *ui.Matrix4 {
	return ui.NewMatrix4FromList([16]float32{1, 0, 0, 0, 0, 0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1})
}

type pt struct{ in, out ui.Offset }

func pts(pairs ...float64) []pt {
	var out []pt
	for i := 0; i < len(pairs); i += 4 {
		out = append(out, pt{ui.Offset{Dx: pairs[i], Dy: pairs[i+1]}, ui.Offset{Dx: pairs[i+2], Dy: pairs[i+3]}})
	}
	return out
}

// layoutTransform lays out the transform in 100x100 at the top left, like the
// tests of transform_test.dart.
func layoutTransform(t *testing.T, transform *ui.Matrix4, leaf rendering.Box, padding float64) {
	t.Helper()
	var child rendering.Box = leaf //nolint:staticcheck
	if padding != 0 {
		child = rendering.NewRenderPadding(painting.EdgeInsets{Left: padding}, leaf)
	}
	sizer := rendering.NewRenderTransform(transform, child, rendering.RenderTransformOpts{Alignment: painting.AlignmentCenter})
	layout(t, newHarness(t), sizer, layoutOpts{
		constraints: foundation.Some(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 100})),
		alignment:   painting.AlignmentTopLeft})
}

func TestRenderTransformPoints(t *testing.T) {
	tests := []struct {
		name      string
		transform *ui.Matrix4
		padding   float64
		size      ui.Size
		toLocal   []pt // globalToLocal
		toGlobal  []pt // localToGlobal
	}{
		{"identity", ui.NewMatrix4Identity(), 0, ui.Size{Width: 100, Height: 100},
			pts(0, 0, 0, 0, 100, 100, 100, 100, 25, 75, 25, 75, 50, 50, 50, 50),
			pts(0, 0, 0, 0, 100, 100, 100, 100, 25, 75, 25, 75, 50, 50, 50, 50)},
		{"identity with internal offset", ui.NewMatrix4Identity(), 20, ui.Size{Width: 80, Height: 100},
			pts(0, 0, -20, 0, 100, 100, 80, 100, 25, 75, 5, 75, 50, 50, 30, 50),
			pts(0, 0, 20, 0, 100, 100, 120, 100, 25, 75, 45, 75, 50, 50, 70, 50)},
		{"translation", ui.NewMatrix4TranslationValues(50, 200, 0), 0, ui.Size{Width: 100, Height: 100},
			pts(0, 0, -50, -200, 100, 100, 50, -100, 25, 75, -25, -125, 50, 50, 0, -150),
			pts(0, 0, 50, 200, 100, 100, 150, 300, 25, 75, 75, 275, 50, 50, 100, 250)},
		{"translation with internal offset", ui.NewMatrix4TranslationValues(50, 200, 0), 20, ui.Size{Width: 80, Height: 100},
			pts(0, 0, -70, -200, 100, 100, 30, -100, 25, 75, -45, -125, 50, 50, -20, -150),
			pts(0, 0, 70, 200, 100, 100, 170, 300, 25, 75, 95, 275, 50, 50, 120, 250)},
		{"rotation", ui.NewMatrix4RotationZ(math.Pi), 0, ui.Size{Width: 100, Height: 100},
			pts(0, 0, 100, 100, 100, 100, 0, 0, 25, 75, 75, 25, 50, 50, 50, 50),
			pts(0, 0, 100, 100, 100, 100, 0, 0, 25, 75, 75, 25, 50, 50, 50, 50)},
		{"rotation with internal offset", ui.NewMatrix4RotationZ(math.Pi), 20, ui.Size{Width: 80, Height: 100},
			pts(0, 0, 80, 100, 100, 100, -20, 0, 25, 75, 55, 25, 50, 50, 30, 50),
			pts(0, 0, 80, 100, 100, 100, -20, 0, 25, 75, 55, 25, 50, 50, 30, 50)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			leaf := newSizedBox(tc.size.Width, tc.size.Height)
			layoutTransform(t, tc.transform, leaf, tc.padding)
			for _, p := range tc.toLocal {
				if got := roundOffset(leaf.GlobalToLocal(p.in, nil)); got != p.out {
					t.Errorf("globalToLocal(%v) = %v, want %v", p.in, got, p.out)
				}
			}
			for _, p := range tc.toGlobal {
				if got := roundOffset(leaf.LocalToGlobal(p.in, nil)); got != p.out {
					t.Errorf("localToGlobal(%v) = %v, want %v", p.in, got, p.out)
				}
			}
		})
	}
}

func TestRenderTransformPerspectiveGlobalToLocal(t *testing.T) {
	leaf := newSizedBox(100, 100)
	layoutTransform(t, rotateAroundXAxis(math.Pi*0.25), leaf, 0) // at pi/4 the box is about 70 pixels high
	if got := roundOffset(leaf.GlobalToLocal(ui.Offset{Dx: 25, Dy: 50}, nil)); got != (ui.Offset{Dx: 25, Dy: 50}) {
		t.Errorf("globalToLocal(25, 50) = %v", got)
	}
	top := leaf.GlobalToLocal(ui.Offset{Dx: 25, Dy: 17}, nil)
	bottom := leaf.GlobalToLocal(ui.Offset{Dx: 25, Dy: 83}, nil)
	if !(top.Dy > 0 && top.Dy < 10) {
		t.Errorf("globalToLocal(25, 17).dy = %v, want 0..10", top.Dy)
	}
	if !(bottom.Dy > 90 && bottom.Dy < 100) {
		t.Errorf("globalToLocal(25, 83).dy = %v, want 90..100", bottom.Dy)
	}
	if math.Round(top.Dy) != 100-math.Round(bottom.Dy) {
		t.Errorf("rounded dy %v and %v are not mirrored", top.Dy, bottom.Dy)
	}
}

func TestRenderTransformParallelViewDirectionIsZero(t *testing.T) {
	leaf := newSizedBox(100, 100)
	layoutTransform(t, rotateAroundXAxis90Degrees(), leaf, 0)
	if got := leaf.GlobalToLocal(ui.Offset{Dx: 25, Dy: 50}, nil); got != ui.OffsetZero {
		t.Errorf("globalToLocal = %v, want zero", got)
	}
}

func TestRenderTransformPerspectiveLocalToGlobal(t *testing.T) {
	leaf := newSizedBox(100, 100)
	layoutTransform(t, rotateAroundXAxis(math.Pi*0.4999), leaf, 0) // the box is seen on its edge
	// The box is about half a pixel high, so everything ends up around the middle.
	// Matrices hold float32 values here, so the first point is rounded too.
	for _, y := range []float64{50, 75, 100} {
		if got := roundOffset(leaf.LocalToGlobal(ui.Offset{Dx: 25, Dy: y}, nil)); got != (ui.Offset{Dx: 25, Dy: 50}) {
			t.Errorf("localToGlobal(25, %v) = %v, want (25, 50)", y, got)
		}
	}
}
