package soft

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// affine is a 2D affine transform: x' = a*x + c*y + e, y' = b*x + d*y + f.
type affine struct{ a, b, c, d, e, f float64 }

func translation(x, y float64) affine { return affine{a: 1, d: 1, e: x, f: y} }

func scaling(x, y float64) affine { return affine{a: x, d: y} }

// affineOf takes the 2D part of a transform; perspective is ignored.
func affineOf(m *ui.Matrix4) affine {
	s := m.Storage()
	return affine{float64(s[0]), float64(s[1]), float64(s[4]), float64(s[5]), float64(s[12]), float64(s[13])}
}

func (t affine) apply(x, y float64) ui.Offset {
	return ui.Offset{Dx: t.a*x + t.c*y + t.e, Dy: t.b*x + t.d*y + t.f}
}

// then returns the transform that applies t first and u after it.
func (t affine) then(u affine) affine {
	return affine{
		a: u.a*t.a + u.c*t.b, b: u.b*t.a + u.d*t.b,
		c: u.a*t.c + u.c*t.d, d: u.b*t.c + u.d*t.d,
		e: u.a*t.e + u.c*t.f + u.e, f: u.b*t.e + u.d*t.f + u.f,
	}
}

func (t affine) det() float64 { return t.a*t.d - t.b*t.c }

// invert returns the inverse, and false if there is none.
func (t affine) invert() (affine, bool) {
	det := t.det()
	if det == 0 || math.IsNaN(det) || math.IsInf(det, 0) {
		return affine{}, false
	}
	a, b, c, d := t.d/det, -t.b/det, -t.c/det, t.a/det
	return affine{a, b, c, d, -(a*t.e + c*t.f), -(b*t.e + d*t.f)}, true
}
