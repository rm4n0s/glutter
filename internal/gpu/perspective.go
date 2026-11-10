package gpu

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// AffinePerspective returns the affine transform that agrees with the transform m, which has a perspective, at
// the center of the bounds of the shape that op draws, and has the same first derivatives there: the first order
// approximation of the projective map. ok is false when the center is at or behind the camera (the weight is not
// positive), when the transform is not finite there, and for an operation that has no bounds (ShapeBounds).
//
// The GPU renderers draw their shapes through coverage and glyph atlases that are keyed by a linear transform, so
// they cannot draw a perspective exactly; a ListWheelScrollView, which paints its children with a perspective of
// 0.003, is what needs it (SPEC §9.12). Over a line of text the difference of the two maps is of the second
// order. `soft` draws the perspective exactly.
//
// Flutter: no counterpart
func AffinePerspective(op ui.DisplayOp, m *ui.Matrix4) (ui.Matrix4, bool) {
	bounds, ok := ShapeBounds(op)
	if !ok {
		return ui.Matrix4{}, false
	}
	s := m.Storage()
	x0, y0 := (bounds.Left+bounds.Right)/2, (bounds.Top+bounds.Bottom)/2
	w := float64(s[3])*x0 + float64(s[7])*y0 + float64(s[15])
	if !(w > 1e-6) || math.IsInf(w, 0) {
		return ui.Matrix4{}, false
	}
	x := float64(s[0])*x0 + float64(s[4])*y0 + float64(s[12])
	y := float64(s[1])*x0 + float64(s[5])*y0 + float64(s[13])
	w2 := w * w
	a := (float64(s[0])*w - x*float64(s[3])) / w2
	c := (float64(s[4])*w - x*float64(s[7])) / w2
	b := (float64(s[1])*w - y*float64(s[3])) / w2
	d := (float64(s[5])*w - y*float64(s[7])) / w2
	tx := x/w - (a*x0 + c*y0)
	ty := y/w - (b*x0 + d*y0)
	for _, v := range []float64{a, b, c, d, tx, ty} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return ui.Matrix4{}, false
		}
	}
	return *ui.NewMatrix4FromList([16]float32{
		float32(a), float32(b), 0, 0,
		float32(c), float32(d), 0, 0,
		0, 0, 1, 0,
		float32(tx), float32(ty), 0, 1,
	}), true
}

// HasPerspective says whether the transform changes the weight of a point of the plane.
//
// Flutter: no counterpart
func HasPerspective(m *ui.Matrix4) bool {
	s := m.Storage()
	return s[3] != 0 || s[7] != 0 || s[15] != 1
}
