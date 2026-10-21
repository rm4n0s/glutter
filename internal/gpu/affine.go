package gpu

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// Affine2D is the 2D part of a transform: x' = A*x + C*y + E, y' = B*x + D*y + F. The
// shape and image pipelines of every GPU renderer work with it, because a shader needs
// the inverse of the transform, not the 4x4 matrix.
//
// Flutter: no counterpart
type Affine2D struct{ A, B, C, D, E, F float64 }

// AffineOf returns the 2D part of a matrix.
//
// Flutter: no counterpart
func AffineOf(m *ui.Matrix4) Affine2D {
	s := m.Storage()
	return Affine2D{float64(s[0]), float64(s[1]), float64(s[4]), float64(s[5]), float64(s[12]), float64(s[13])}
}

// Det returns the determinant of the linear part.
//
// Flutter: no counterpart
func (t Affine2D) Det() float64 { return t.A*t.D - t.B*t.C }

// Apply returns the point (x, y) transformed.
//
// Flutter: no counterpart
func (t Affine2D) Apply(x, y float64) ui.Offset {
	return ui.Offset{Dx: t.A*x + t.C*y + t.E, Dy: t.B*x + t.D*y + t.F}
}

// Then returns the transform that applies t first and u after it.
//
// Flutter: no counterpart
func (t Affine2D) Then(u Affine2D) Affine2D {
	return Affine2D{
		A: u.A*t.A + u.C*t.B, B: u.B*t.A + u.D*t.B,
		C: u.A*t.C + u.C*t.D, D: u.B*t.C + u.D*t.D,
		E: u.A*t.E + u.C*t.F + u.E, F: u.B*t.E + u.D*t.F + u.F,
	}
}

// Translation returns the transform that moves by (x, y).
//
// Flutter: no counterpart
func Translation(x, y float64) Affine2D { return Affine2D{A: 1, D: 1, E: x, F: y} }

// Scaling returns the transform that scales by x and y.
//
// Flutter: no counterpart
func Scaling(x, y float64) Affine2D { return Affine2D{A: x, D: y} }

// AxisAligned says that the transform keeps rectangles axis-aligned (it may flip them or
// turn them by a quarter).
//
// Flutter: no counterpart
func (t Affine2D) AxisAligned() bool {
	const eps = 1e-6
	near := func(v, with float64) bool { return math.Abs(v) <= eps*math.Abs(with)+1e-12 }
	return (near(t.B, t.A) && near(t.C, t.D)) || (near(t.A, t.B) && near(t.D, t.C))
}

// Invert returns the inverse transform; false when the transform has none (its
// determinant is 0 or not a number).
//
// Flutter: no counterpart
func (t Affine2D) Invert() (Affine2D, bool) {
	det := t.Det()
	if det == 0 || math.IsNaN(det) || math.IsInf(det, 0) {
		return Affine2D{}, false
	}
	return Affine2D{
		A: t.D / det, B: -t.B / det, C: -t.C / det, D: t.A / det,
		E: (t.C*t.F - t.D*t.E) / det, F: (t.B*t.E - t.A*t.F) / det,
	}, true
}

// Similarity says that the transform only rotates, flips and scales evenly, so that a
// distance in the shape's units is the same number of pixels in every direction.
//
// Flutter: no counterpart
func (t Affine2D) Similarity() bool {
	const eps = 1e-4
	n1, n2 := math.Hypot(t.A, t.B), math.Hypot(t.C, t.D)
	return math.Abs(n1-n2) <= eps*math.Max(n1, n2) && math.Abs(t.A*t.C+t.B*t.D) <= eps*n1*n2
}
