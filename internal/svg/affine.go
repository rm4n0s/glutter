package svg

import (
	"math"
	"strings"

	"github.com/rm4n0s/glutter/ui"
)

// Affine is a 2D affine transform in the order of SVG's matrix(a b c d e f): a point (x, y) goes to
// (A·x + C·y + E, B·x + D·y + F).
//
// SVG 1.1: coords.html#TransformMatrixDefined
type Affine struct {
	A, B, C, D, E, F float64
}

// Identity is the transform that moves nothing.
var Identity = Affine{A: 1, D: 1}

// IsIdentity reports whether the transform moves nothing.
func (m Affine) IsIdentity() bool { return m == Identity }

// Mul returns the transform that applies n first and then m (the matrix product m·n). A transform list is the
// product of its items from left to right.
//
// SVG 1.1: coords.html#TransformAttribute
func (m Affine) Mul(n Affine) Affine {
	return Affine{
		A: m.A*n.A + m.C*n.B,
		B: m.B*n.A + m.D*n.B,
		C: m.A*n.C + m.C*n.D,
		D: m.B*n.C + m.D*n.D,
		E: m.A*n.E + m.C*n.F + m.E,
		F: m.B*n.E + m.D*n.F + m.F,
	}
}

// Apply maps a point.
func (m Affine) Apply(x, y float64) (float64, float64) {
	return m.A*x + m.C*y + m.E, m.B*x + m.D*y + m.F
}

// Det returns the determinant of the linear part.
func (m Affine) Det() float64 { return m.A*m.D - m.B*m.C }

// Invert returns the inverse transform, or false if the transform has none (a zero determinant).
func (m Affine) Invert() (Affine, bool) {
	det := m.Det()
	if det == 0 || math.IsNaN(det) || math.IsInf(det, 0) {
		return Identity, false
	}
	return Affine{
		A: m.D / det,
		B: -m.B / det,
		C: -m.C / det,
		D: m.A / det,
		E: (m.C*m.F - m.D*m.E) / det,
		F: (m.B*m.E - m.A*m.F) / det,
	}, true
}

// Translate returns a translation.
func Translate(tx, ty float64) Affine { return Affine{A: 1, D: 1, E: tx, F: ty} }

// Scale returns a scale.
func Scale(sx, sy float64) Affine { return Affine{A: sx, D: sy} }

// Rotate returns a rotation by deg degrees about the origin.
func Rotate(deg float64) Affine {
	s, c := sinCosDeg(deg)
	return Affine{A: c, B: s, C: -s, D: c}
}

// sinCosDeg is sin and cos of an angle in degrees, exact for multiples of 90.
func sinCosDeg(deg float64) (s, c float64) {
	switch r := math.Mod(deg, 360); r {
	case 0:
		return 0, 1
	case 90, -270:
		return 1, 0
	case 180, -180:
		return 0, -1
	case 270, -90:
		return -1, 0
	}
	rad := deg * math.Pi / 180
	return math.Sin(rad), math.Cos(rad)
}

// Matrix4 returns the transform as the 4x4 matrix that ui.Canvas.Transform takes.
func (m Affine) Matrix4() *ui.Matrix4 {
	return ui.NewMatrix4FromList([16]float32{
		float32(m.A), float32(m.B), 0, 0,
		float32(m.C), float32(m.D), 0, 0,
		0, 0, 1, 0,
		float32(m.E), float32(m.F), 0, 1,
	})
}

// ParseTransform parses a transform list (matrix, translate, scale, rotate, skewX and skewY, separated by white
// space and commas). A list with an error is not valid as a whole, and the result is false.
//
// SVG 1.1: coords.html#TransformAttribute
func ParseTransform(s string) (Affine, bool) { return parseTransformList(s, false) }

// parseCSSTransform parses the transform property of CSS, whose items also have units (px, deg, rad, grad, turn),
// the functions translateX/translateY/scaleX/scaleY, and may not be written with commas between the items.
//
// SVG 2: coords.html#TransformProperty
func parseCSSTransform(s string) (Affine, bool) { return parseTransformList(s, true) }

func parseTransformList(s string, css bool) (Affine, bool) {
	m := Identity
	i := skipSpace(s, 0)
	if css && strings.TrimSpace(s) == "none" {
		return Identity, true
	}
	for i < len(s) {
		start := i
		for i < len(s) && (s[i] >= 'a' && s[i] <= 'z' || s[i] >= 'A' && s[i] <= 'Z') {
			i++
		}
		name := s[start:i]
		i = skipSpace(s, i)
		if name == "" || i >= len(s) || s[i] != '(' {
			return Identity, false
		}
		end := strings.IndexByte(s[i:], ')')
		if end < 0 {
			return Identity, false
		}
		args, ok := parseTransformArgs(s[i+1:i+end], name, css)
		if !ok {
			return Identity, false
		}
		t, ok := transformItem(name, args, css)
		if !ok {
			return Identity, false
		}
		m = m.Mul(t)
		i = skipCommaSpace(s, i+end+1)
	}
	return m, true
}

// parseTransformArgs reads the arguments of one transform function. In CSS mode a number may have px, deg, rad,
// grad or turn after it; angles become degrees.
func parseTransformArgs(s, fn string, css bool) ([]float64, bool) {
	var out []float64
	i := skipSpace(s, 0)
	for i < len(s) {
		v, j, ok := scanNumber(s, i)
		if !ok {
			return nil, false
		}
		if css {
			k := j
			for k < len(s) && s[k] >= 'a' && s[k] <= 'z' {
				k++
			}
			switch s[j:k] {
			case "":
			case "px":
			case "deg":
			case "rad":
				v = v * 180 / math.Pi
			case "grad":
				v = v * 360 / 400
			case "turn":
				v *= 360
			default:
				return nil, false
			}
			j = k
		}
		out = append(out, v)
		i = skipCommaSpace(s, j)
	}
	return out, true
}

func transformItem(name string, a []float64, css bool) (Affine, bool) {
	switch name {
	case "matrix":
		if len(a) != 6 {
			return Identity, false
		}
		return Affine{a[0], a[1], a[2], a[3], a[4], a[5]}, true
	case "translate":
		switch len(a) {
		case 1:
			return Translate(a[0], 0), true
		case 2:
			return Translate(a[0], a[1]), true
		}
	case "scale":
		switch len(a) {
		case 1:
			return Scale(a[0], a[0]), true
		case 2:
			return Scale(a[0], a[1]), true
		}
	case "rotate":
		switch len(a) {
		case 1:
			return Rotate(a[0]), true
		case 3:
			if css {
				return Identity, false
			}
			return Translate(a[1], a[2]).Mul(Rotate(a[0])).Mul(Translate(-a[1], -a[2])), true
		}
	case "skewX":
		if len(a) == 1 {
			return Affine{A: 1, C: math.Tan(a[0] * math.Pi / 180), D: 1}, true
		}
	case "skewY":
		if len(a) == 1 {
			return Affine{A: 1, B: math.Tan(a[0] * math.Pi / 180), D: 1}, true
		}
	case "translateX":
		if css && len(a) == 1 {
			return Translate(a[0], 0), true
		}
	case "translateY":
		if css && len(a) == 1 {
			return Translate(0, a[0]), true
		}
	case "scaleX":
		if css && len(a) == 1 {
			return Scale(a[0], 1), true
		}
	case "scaleY":
		if css && len(a) == 1 {
			return Scale(1, a[0]), true
		}
	}
	return Identity, false
}
