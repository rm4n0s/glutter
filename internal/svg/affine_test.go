package svg

import (
	"math"
	"testing"
)

func affEq(a, b Affine) bool {
	d := func(x, y float64) bool { return math.Abs(x-y) < 1e-9 }
	return d(a.A, b.A) && d(a.B, b.B) && d(a.C, b.C) && d(a.D, b.D) && d(a.E, b.E) && d(a.F, b.F)
}

func TestParseTransform(t *testing.T) {
	tests := []struct {
		in   string
		want Affine
		ok   bool
	}{
		{"", Identity, true},
		{"matrix(1 2 3 4 5 6)", Affine{1, 2, 3, 4, 5, 6}, true},
		{"matrix(1,2,3,4,5,6)", Affine{1, 2, 3, 4, 5, 6}, true},
		{"translate(10)", Translate(10, 0), true},
		{"translate(10 20)", Translate(10, 20), true},
		{"translate(10,-20)", Translate(10, -20), true},
		{"scale(2)", Scale(2, 2), true},
		{"scale(2 3)", Scale(2, 3), true},
		{"rotate(90)", Affine{0, 1, -1, 0, 0, 0}, true},
		{"rotate(180)", Affine{-1, 0, 0, -1, 0, 0}, true},
		{"rotate(90 10 10)", Affine{0, 1, -1, 0, 20, 0}, true},
		{"skewX(45)", Affine{1, 0, 1, 1, 0, 0}, true},
		{"skewY(45)", Affine{1, 1, 0, 1, 0, 0}, true},
		{"translate(10 0) scale(2)", Affine{2, 0, 0, 2, 10, 0}, true},
		{"translate(10 0),scale(2)", Affine{2, 0, 0, 2, 10, 0}, true},
		{"scale(2)translate(10 0)", Affine{2, 0, 0, 2, 20, 0}, true},
		{"  rotate( 90 )  ", Affine{0, 1, -1, 0, 0, 0}, true},
		{"translate(1e1 -.5)", Translate(10, -0.5), true},
		{"matrix(1 2 3)", Identity, false},
		{"translate()", Identity, false},
		{"translate(1 2 3)", Identity, false},
		{"foo(1)", Identity, false},
		{"translate(10px)", Identity, false},
		{"translate(10", Identity, false},
		{"rotate(1 2)", Identity, false},
	}
	for _, tt := range tests {
		got, ok := ParseTransform(tt.in)
		if ok != tt.ok || (ok && !affEq(got, tt.want)) {
			t.Errorf("ParseTransform(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestParseCSSTransform(t *testing.T) {
	tests := []struct {
		in   string
		want Affine
		ok   bool
	}{
		{"none", Identity, true},
		{"translate(10px, 20px)", Translate(10, 20), true},
		{"rotate(90deg)", Affine{0, 1, -1, 0, 0, 0}, true},
		{"rotate(0.25turn)", Affine{0, 1, -1, 0, 0, 0}, true},
		{"rotate(100grad)", Affine{0, 1, -1, 0, 0, 0}, true},
		{"translateX(5px) translateY(6px)", Translate(5, 6), true},
		{"scaleX(2) scaleY(3)", Scale(2, 3), true},
		{"rotate(1deg 2 3)", Identity, false},
		{"translate(1em)", Identity, false},
	}
	for _, tt := range tests {
		got, ok := parseCSSTransform(tt.in)
		if ok != tt.ok || (ok && !affEq(got, tt.want)) {
			t.Errorf("parseCSSTransform(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestAffineAlgebra(t *testing.T) {
	m := Translate(10, 20).Mul(Rotate(30)).Mul(Scale(2, 3))
	inv, ok := m.Invert()
	if !ok {
		t.Fatal("no inverse")
	}
	if p := m.Mul(inv); !affEq(p, Identity) {
		t.Errorf("m * m^-1 = %v", p)
	}
	x, y := m.Apply(1, 1)
	ix, iy := inv.Apply(x, y)
	if math.Abs(ix-1) > 1e-9 || math.Abs(iy-1) > 1e-9 {
		t.Errorf("round trip = %v, %v", ix, iy)
	}
	if _, ok := Scale(0, 1).Invert(); ok {
		t.Error("a singular matrix has an inverse")
	}
	m4 := Affine{1, 2, 3, 4, 5, 6}.Matrix4()
	for _, c := range []struct {
		row, col int
		want     float32
	}{{0, 0, 1}, {1, 0, 2}, {0, 1, 3}, {1, 1, 4}, {0, 3, 5}, {1, 3, 6}, {2, 2, 1}, {3, 3, 1}} {
		if got := m4.Entry(c.row, c.col); got != c.want {
			t.Errorf("Matrix4[%d][%d] = %v, want %v", c.row, c.col, got, c.want)
		}
	}
}
