package gpu_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// project is the projective map of the transform on a point of the plane.
func project(m *ui.Matrix4, x, y float64) (float64, float64) {
	s := m.Storage()
	w := float64(s[3])*x + float64(s[7])*y + float64(s[15])
	return (float64(s[0])*x + float64(s[4])*y + float64(s[12])) / w, (float64(s[1])*x + float64(s[5])*y + float64(s[13])) / w
}

func TestAffinePerspectiveTouchesTheProjectiveMapAtTheCenter(t *testing.T) {
	// The transform of a row of a wheel: the cylinder's rotation about the x axis, then the perspective of 0.003.
	m := ui.NewMatrix4Identity()
	m.SetEntry(3, 2, 0.003)
	m.Translate(0, 40, -200)
	m.MultiplyBy(ui.NewMatrix4RotationX(0.5))
	m.Translate(0, 0, 200)
	rect := ui.NewRectFromLTWH(20, 30, 40, 16)
	op := ui.OpDrawRect{Rect: rect}
	// What a 2D drawing sees of the 4x4 matrix is its x and y columns: take the plane z = 0.
	s := m.Storage()
	flat := ui.NewMatrix4FromList([16]float32{s[0], s[1], 0, s[3], s[4], s[5], 0, s[7], 0, 0, 1, 0, s[12], s[13], 0, s[15]})
	if !gpu.HasPerspective(flat) {
		t.Fatal("the test transform has no perspective")
	}
	got, ok := gpu.AffinePerspective(op, flat)
	if !ok {
		t.Fatal("no approximation")
	}
	if gpu.HasPerspective(&got) {
		t.Errorf("the approximation has a perspective: %v", got.Storage())
	}
	cx, cy := (rect.Left+rect.Right)/2, (rect.Top+rect.Bottom)/2
	wantX, wantY := project(flat, cx, cy)
	gotX, gotY := project(&got, cx, cy)
	if math.Abs(gotX-wantX) > 1e-3 || math.Abs(gotY-wantY) > 1e-3 {
		t.Errorf("the center maps to (%v, %v), want (%v, %v)", gotX, gotY, wantX, wantY)
	}
	// Near the center the two maps differ by less than a pixel over a glyph's width and a row's height.
	for _, p := range [][2]float64{{rect.Left, rect.Top}, {rect.Right, rect.Top}, {rect.Left, rect.Bottom}, {rect.Right, rect.Bottom}} {
		wx, wy := project(flat, p[0], p[1])
		gx, gy := project(&got, p[0], p[1])
		if math.Hypot(gx-wx, gy-wy) > 1 {
			t.Errorf("the corner %v maps to (%v, %v), the projective map says (%v, %v)", p, gx, gy, wx, wy)
		}
	}
}

func TestAffinePerspectiveRefuses(t *testing.T) {
	m := ui.NewMatrix4Identity()
	m.SetEntry(3, 0, 0.001)
	behind := ui.NewMatrix4Identity()
	behind.SetEntry(3, 0, -1)
	tests := []struct {
		name string
		op   ui.DisplayOp
		m    *ui.Matrix4
	}{
		{"an operation without bounds", ui.OpDrawColor{Color: ui.NewColor(0xFF000000)}, m},
		{"a center behind the camera", ui.OpDrawRect{Rect: ui.NewRectFromLTWH(10, 0, 10, 10)}, behind},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := gpu.AffinePerspective(tt.op, tt.m); ok {
				t.Error("an approximation was made")
			}
		})
	}
}
