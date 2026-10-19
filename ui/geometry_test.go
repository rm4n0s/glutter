package ui

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestToStringAsFixed1(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0.0"},
		{1, "1.0"},
		{10.5, "10.5"},
		{0.25, "0.3"},   // an exact tie rounds away from zero, as in Dart
		{-0.25, "-0.3"}, // ... for negative values too
		{0.75, "0.8"},
		{2.25, "2.3"},
		{0.15, "0.1"}, // not a tie: 0.15 is a little below
		{0.05, "0.1"}, // not a tie: 0.05 is a little above
		{-0.04, "-0.0"},
		{1234567.89, "1234567.9"},
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
	}
	for _, tt := range tests {
		if got := toStringAsFixed1(tt.in); got != tt.want {
			t.Errorf("toStringAsFixed1(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestToStringAsFixedDigits(t *testing.T) {
	tests := []struct {
		in     float64
		digits int
		want   string
	}{
		{0.125, 2, "0.13"}, // exact tie: away from zero
		{-0.125, 2, "-0.13"},
		{0.375, 2, "0.38"},
		{0.12345, 4, "0.1235"}, // 0.12345 is a little above in binary
		{1, 4, "1.0000"},
		{0.99995, 4, "1.0000"},
		{9.5, 0, "10"}, // a tie that carries into a new digit
		{0.5, 0, "1"},
		{1.5, 0, "2"},
		{2.5, 0, "3"},
		{99.95, 1, "100.0"},
		{0.9999999, 4, "1.0000"},
	}
	for _, tt := range tests {
		if got := toStringAsFixed(tt.in, tt.digits); got != tt.want {
			t.Errorf("toStringAsFixed(%v, %d) = %q, want %q", tt.in, tt.digits, got, tt.want)
		}
	}
}

func TestDartDouble(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0.0"},
		{math.Copysign(0, -1), "-0.0"},
		{1, "1.0"},
		{-2, "-2.0"},
		{0.5, "0.5"},
		{0.2126, "0.2126"},
		{123456.789, "123456.789"},
		{0.0001, "0.0001"},
		{0.00001, "1e-5"},
		{1.5e-7, "1.5e-7"},
		{1e21, "1e+21"},
		{1.5e22, "1.5e+22"},
		{1e20, "100000000000000000000.0"},
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
	}
	for _, tt := range tests {
		if got := dartDouble(tt.in); got != tt.want {
			t.Errorf("dartDouble(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestClampDoubleAndLerpDouble(t *testing.T) {
	nan := math.NaN()
	inf := math.Inf(1)
	clamp := []struct {
		x, min, max, want float64
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{11, 0, 10, 10},
		{nan, 0, 10, 10}, // a NaN gives the maximum
		{inf, 0, 10, 10},
		{3, math.Inf(-1), inf, 3},
	}
	for _, tt := range clamp {
		if got := ClampDouble(tt.x, tt.min, tt.max); got != tt.want {
			t.Errorf("ClampDouble(%v, %v, %v) = %v, want %v", tt.x, tt.min, tt.max, got, tt.want)
		}
	}

	lerp := []struct {
		a, b, t, want float64
	}{
		{0, 10, 0, 0},
		{0, 10, 1, 10},
		{0, 10, 0.25, 2.5},
		{10, 0, 0.25, 7.5},
		{-5, 5, 0.5, 0},
		{3, 3, 0.7, 3},       // equal values give the value, whatever t is
		{inf, inf, 0.5, inf}, // ... infinities included
		{0, 10, 2, 20},       // t is not clamped
	}
	for _, tt := range lerp {
		if got := LerpDouble(tt.a, tt.b, tt.t); got != tt.want {
			t.Errorf("LerpDouble(%v, %v, %v) = %v, want %v", tt.a, tt.b, tt.t, got, tt.want)
		}
	}
	if got := LerpDouble(nan, nan, 0.5); !math.IsNaN(got) {
		t.Errorf("LerpDouble(NaN, NaN) = %v, want NaN", got)
	}

	one, ten := 1.0, 10.0
	opt := []struct {
		name string
		a, b *float64
		want *float64
	}{
		{"both nil", nil, nil, nil},
		{"a nil counts as zero", nil, &ten, ptr(5.0)},
		{"b nil counts as zero", &ten, nil, ptr(5.0)},
		{"both set", &one, &ten, ptr(5.5)},
	}
	for _, tt := range opt {
		got := LerpDoubleOpt(tt.a, tt.b, 0.5)
		if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, deref(got), deref(tt.want))
		}
	}
}

func ptr[T any](v T) *T { return &v }

func deref(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestOffset(t *testing.T) {
	a, b := Offset{3, 4}, Offset{1, -2}
	tests := []struct {
		name string
		got  Offset
		want Offset
	}{
		{"zero", OffsetZero, Offset{0, 0}},
		{"add", a.Add(b), Offset{4, 2}},
		{"sub", a.Sub(b), Offset{2, 6}},
		{"neg", a.Neg(), Offset{-3, -4}},
		{"mul", a.Mul(2), Offset{6, 8}},
		{"div", a.Div(2), Offset{1.5, 2}},
		{"trunc div", Offset{7, -7}.TruncDiv(2), Offset{3, -3}},
		{"mod", Offset{7, -7}.Mod(3), Offset{1, 2}}, // Dart's % is never negative
		{"mod of a negative divisor", Offset{7, -7}.Mod(-3), Offset{1, 2}},
		{"scale", a.Scale(2, 3), Offset{6, 12}},
		{"translate", a.Translate(10, 20), Offset{13, 24}},
		{"lerp", a.Lerp(b, 0.5), Offset{2, 1}},
		{"from direction, default distance", NewOffsetFromDirection(0, 1), Offset{1, 0}},
		{"from direction, distance", NewOffsetFromDirection(math.Pi, 3), Offset{-3, 3 * math.Sin(math.Pi)}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	if d := a.Distance(); d != 5 {
		t.Errorf("Distance = %v, want 5", d)
	}
	if d := a.DistanceSquared(); d != 25 {
		t.Errorf("DistanceSquared = %v, want 25", d)
	}
	if d := (Offset{0, 1}).Direction(); d != math.Pi/2 {
		t.Errorf("Direction = %v, want pi/2", d)
	}
	if got := a.And(Size{10, 20}); got != (Rect{3, 4, 13, 24}) {
		t.Errorf("And = %v", got)
	}
	// Mod by zero is NaN, as in Dart.
	if got := (Offset{1, 1}).Mod(0); !math.IsNaN(got.Dx) || !math.IsNaN(got.Dy) {
		t.Errorf("Mod(0) = %v, want NaN", got)
	}
}

func TestOffsetPredicatesAndComparison(t *testing.T) {
	nan := math.NaN()
	tests := []struct {
		name             string
		o                Offset
		infinite, finite bool
		lt, le, gt, ge   bool // compared to {2, 2}
	}{
		{"smaller", Offset{1, 1}, false, true, true, true, false, false},
		{"equal", Offset{2, 2}, false, true, false, true, false, true},
		{"mixed", Offset{1, 3}, false, true, false, false, false, false},
		{"greater", Offset{3, 3}, false, true, false, false, true, true},
		{"infinite", OffsetInfinite, true, false, false, false, true, true},
		{"one infinite", Offset{1, math.Inf(1)}, true, false, false, false, false, false},
		{"negative infinity is not IsInfinite", Offset{math.Inf(-1), 0}, false, false, true, true, false, false},
		{"NaN", Offset{nan, 1}, false, false, false, false, false, false},
	}
	two := Offset{2, 2}
	for _, tt := range tests {
		if tt.o.IsInfinite() != tt.infinite || tt.o.IsFinite() != tt.finite {
			t.Errorf("%s: IsInfinite/IsFinite = %v/%v, want %v/%v", tt.name, tt.o.IsInfinite(), tt.o.IsFinite(), tt.infinite, tt.finite)
		}
		if tt.o.Lt(two) != tt.lt || tt.o.Le(two) != tt.le || tt.o.Gt(two) != tt.gt || tt.o.Ge(two) != tt.ge {
			t.Errorf("%s: comparisons = %v %v %v %v, want %v %v %v %v", tt.name,
				tt.o.Lt(two), tt.o.Le(two), tt.o.Gt(two), tt.o.Ge(two), tt.lt, tt.le, tt.gt, tt.ge)
		}
	}
	if (Offset{nan, 0}) == (Offset{nan, 0}) {
		t.Error("offsets with NaN must not be equal, as in Dart")
	}
}

func TestOffsetLerpWithNil(t *testing.T) {
	a, b := Offset{10, 20}, Offset{30, 60}
	tests := []struct {
		name string
		a, b *Offset
		t    float64
		want *Offset
	}{
		{"both nil", nil, nil, 0.5, nil},
		{"a nil scales b by t", nil, &b, 0.5, &Offset{15, 30}},
		{"b nil scales a by 1-t", &a, nil, 0.25, &Offset{7.5, 15}},
		{"both", &a, &b, 0.5, &Offset{20, 40}},
		{"t = 0", &a, &b, 0, &a},
		{"t = 1", &a, &b, 1, &b},
	}
	for _, tt := range tests {
		got := OffsetLerp(tt.a, tt.b, tt.t)
		if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestSize(t *testing.T) {
	s := Size{4, 2}
	tests := []struct {
		name string
		got  Size
		want Size
	}{
		{"square", NewSizeSquare(3), Size{3, 3}},
		{"from radius", NewSizeFromRadius(5), Size{10, 10}},
		{"from width", NewSizeFromWidth(7), Size{7, math.Inf(1)}},
		{"from height", NewSizeFromHeight(7), Size{math.Inf(1), 7}},
		{"add offset", s.Add(Offset{1, 2}), Size{5, 4}},
		{"sub offset", s.SubOffset(Offset{1, 2}), Size{3, 0}},
		{"mul", s.Mul(2), Size{8, 4}},
		{"div", s.Div(2), Size{2, 1}},
		{"trunc div", Size{7, 5}.TruncDiv(2), Size{3, 2}},
		{"mod", Size{7, -5}.Mod(3), Size{1, 1}},
		{"flipped", s.Flipped(), Size{2, 4}},
		{"lerp", s.Lerp(Size{8, 6}, 0.5), Size{6, 4}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	if got := s.Sub(Size{1, 5}); got != (Offset{3, -3}) {
		t.Errorf("Sub(Size) = %v, want Offset(3, -3)", got)
	}
	if s.ShortestSide() != 2 || s.LongestSide() != 4 || (Size{-5, 3}).LongestSide() != 5 {
		t.Error("ShortestSide/LongestSide wrong")
	}
	if !s.Lt(Size{5, 3}) || s.Lt(Size{5, 2}) || !s.Le(Size{4, 2}) || !s.Gt(Size{3, 1}) || !s.Ge(Size{4, 2}) {
		t.Error("comparisons wrong")
	}
	if !SizeInfinite.IsInfinite() || SizeInfinite.IsFinite() || !s.IsFinite() || s.IsInfinite() {
		t.Error("IsInfinite/IsFinite wrong")
	}
}

func TestSizeAspectRatio(t *testing.T) {
	tests := []struct {
		size Size
		want float64
	}{
		{Size{4, 2}, 2},
		{Size{2, 4}, 0.5},
		{Size{3, 0}, math.Inf(1)},
		{Size{-3, 0}, math.Inf(-1)},
		{Size{0, 0}, 0},
		{Size{-4, 2}, -2},
	}
	for _, tt := range tests {
		if got := tt.size.AspectRatio(); got != tt.want {
			t.Errorf("%v.AspectRatio() = %v, want %v", tt.size, got, tt.want)
		}
	}
}

func TestSizeEmptyAndContains(t *testing.T) {
	tests := []struct {
		name  string
		s     Size
		empty bool
	}{
		{"normal", Size{1, 1}, false},
		{"zero width", Size{0, 5}, true},
		{"zero height", Size{5, 0}, true},
		{"negative", Size{-1, 5}, true},
		{"zero", SizeZero, true},
	}
	for _, tt := range tests {
		if got := tt.s.IsEmpty(); got != tt.empty {
			t.Errorf("%s: IsEmpty = %v, want %v", tt.name, got, tt.empty)
		}
	}
	s := Size{10, 20}
	contains := []struct {
		p    Offset
		want bool
	}{
		{Offset{0, 0}, true},
		{Offset{9.9, 19.9}, true},
		{Offset{10, 5}, false}, // the right edge is outside
		{Offset{5, 20}, false}, // ... and so is the bottom
		{Offset{-0.1, 5}, false},
		{Offset{5, -0.1}, false},
	}
	for _, tt := range contains {
		if got := s.Contains(tt.p); got != tt.want {
			t.Errorf("Contains(%v) = %v, want %v", tt.p, got, tt.want)
		}
	}
}

func TestSizeCorners(t *testing.T) {
	s := Size{10, 20}
	o := Offset{100, 200}
	tests := []struct {
		name string
		got  Offset
		want Offset
	}{
		{"topLeft", s.TopLeft(o), Offset{100, 200}},
		{"topCenter", s.TopCenter(o), Offset{105, 200}},
		{"topRight", s.TopRight(o), Offset{110, 200}},
		{"centerLeft", s.CenterLeft(o), Offset{100, 210}},
		{"center", s.Center(o), Offset{105, 210}},
		{"centerRight", s.CenterRight(o), Offset{110, 210}},
		{"bottomLeft", s.BottomLeft(o), Offset{100, 220}},
		{"bottomCenter", s.BottomCenter(o), Offset{105, 220}},
		{"bottomRight", s.BottomRight(o), Offset{110, 220}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
		// The same as the corner of the rectangle with this origin and size.
	}
	r := o.And(s)
	if r.TopCenter() != s.TopCenter(o) || r.Center() != s.Center(o) || r.BottomRight() != s.BottomRight(o) {
		t.Error("Size corners disagree with Rect corners")
	}
}

func TestSizeLerpWithNil(t *testing.T) {
	a, b := Size{10, 20}, Size{30, 60}
	if SizeLerp(nil, nil, 0.5) != nil {
		t.Error("nil, nil must give nil")
	}
	if got := SizeLerp(nil, &b, 0.5); got == nil || *got != (Size{15, 30}) {
		t.Errorf("nil, b = %v", got)
	}
	if got := SizeLerp(&a, nil, 0.25); got == nil || *got != (Size{7.5, 15}) {
		t.Errorf("a, nil = %v", got)
	}
	if got := SizeLerp(&a, &b, 0.5); got == nil || *got != (Size{20, 40}) {
		t.Errorf("a, b = %v", got)
	}
}

func TestRectConstructors(t *testing.T) {
	tests := []struct {
		name string
		got  Rect
		want Rect
	}{
		{"LTRB", NewRectFromLTRB(1, 2, 3, 4), Rect{1, 2, 3, 4}},
		{"LTWH", NewRectFromLTWH(1, 2, 10, 20), Rect{1, 2, 11, 22}},
		{"circle", NewRectFromCircle(Offset{10, 10}, 5), Rect{5, 5, 15, 15}},
		{"center", NewRectFromCenter(Offset{10, 20}, 4, 6), Rect{8, 17, 12, 23}},
		{"points, in order", NewRectFromPoints(Offset{1, 2}, Offset{3, 4}), Rect{1, 2, 3, 4}},
		{"points, reversed", NewRectFromPoints(Offset{3, 4}, Offset{1, 2}), Rect{1, 2, 3, 4}},
		{"points, crossed", NewRectFromPoints(Offset{3, 2}, Offset{1, 4}), Rect{1, 2, 3, 4}},
		{"zero", RectZero, Rect{}},
		{"largest", RectLargest, Rect{-1e9, -1e9, 1e9, 1e9}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestRectMeasures(t *testing.T) {
	r := Rect{10, 20, 40, 100}
	if r.Width() != 30 || r.Height() != 80 || r.Size() != (Size{30, 80}) {
		t.Errorf("Width/Height/Size = %v %v %v", r.Width(), r.Height(), r.Size())
	}
	if r.ShortestSide() != 30 || r.LongestSide() != 80 {
		t.Errorf("sides = %v %v", r.ShortestSide(), r.LongestSide())
	}
	corners := []struct {
		name      string
		got, want Offset
	}{
		{"topLeft", r.TopLeft(), Offset{10, 20}},
		{"topCenter", r.TopCenter(), Offset{25, 20}},
		{"topRight", r.TopRight(), Offset{40, 20}},
		{"centerLeft", r.CenterLeft(), Offset{10, 60}},
		{"center", r.Center(), Offset{25, 60}},
		{"centerRight", r.CenterRight(), Offset{40, 60}},
		{"bottomLeft", r.BottomLeft(), Offset{10, 100}},
		{"bottomCenter", r.BottomCenter(), Offset{25, 100}},
		{"bottomRight", r.BottomRight(), Offset{40, 100}},
	}
	for _, tt := range corners {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestRectPredicates(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	tests := []struct {
		name                       string
		r                          Rect
		empty, hasNaN, inf_, finit bool
	}{
		{"normal", Rect{0, 0, 10, 10}, false, false, false, true},
		{"zero width", Rect{5, 0, 5, 10}, true, false, false, true},
		{"inverted", Rect{10, 0, 0, 10}, true, false, false, true},
		{"zero height", Rect{0, 5, 10, 5}, true, false, false, true},
		{"NaN", Rect{0, 0, nan, 10}, false, true, false, false}, // comparisons with NaN are false
		{"infinite right", Rect{0, 0, inf, 10}, false, false, true, false},
		{"negative infinity is finite-false but not infinite", Rect{math.Inf(-1), 0, 10, 10}, false, false, false, false},
		{"zero", RectZero, true, false, false, true},
	}
	for _, tt := range tests {
		if tt.r.IsEmpty() != tt.empty || tt.r.HasNaN() != tt.hasNaN || tt.r.IsInfinite() != tt.inf_ || tt.r.IsFinite() != tt.finit {
			t.Errorf("%s: empty/NaN/infinite/finite = %v/%v/%v/%v, want %v/%v/%v/%v", tt.name,
				tt.r.IsEmpty(), tt.r.HasNaN(), tt.r.IsInfinite(), tt.r.IsFinite(), tt.empty, tt.hasNaN, tt.inf_, tt.finit)
		}
	}
}

func TestRectTransformations(t *testing.T) {
	r := Rect{10, 20, 30, 60}
	tests := []struct {
		name string
		got  Rect
		want Rect
	}{
		{"shift", r.Shift(Offset{5, -5}), Rect{15, 15, 35, 55}},
		{"translate", r.Translate(1, 2), Rect{11, 22, 31, 62}},
		{"inflate", r.Inflate(5), Rect{5, 15, 35, 65}},
		{"deflate", r.Deflate(5), Rect{15, 25, 25, 55}},
		{"deflate past the center inverts", r.Deflate(15), Rect{25, 35, 15, 45}},
		{"intersect", r.Intersect(Rect{20, 30, 50, 50}), Rect{20, 30, 30, 50}},
		{"intersect, disjoint, is empty and inverted", r.Intersect(Rect{40, 0, 50, 10}), Rect{40, 20, 30, 10}},
		{"expand to include", r.ExpandToInclude(Rect{0, 30, 15, 100}), Rect{0, 20, 30, 100}},
		{"expand with itself", r.ExpandToInclude(r), r},
		{"lerp", r.Lerp(Rect{30, 40, 50, 100}, 0.5), Rect{20, 30, 40, 80}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	if !r.Intersect(Rect{40, 0, 50, 10}).IsEmpty() {
		t.Error("the intersection of disjoint rectangles must be empty")
	}
}

func TestRectOverlapsAndContains(t *testing.T) {
	r := Rect{0, 0, 10, 10}
	overlaps := []struct {
		name string
		o    Rect
		want bool
	}{
		{"inside", Rect{2, 2, 4, 4}, true},
		{"partial", Rect{5, 5, 15, 15}, true},
		{"touching on the right does not overlap", Rect{10, 0, 20, 10}, false},
		{"touching below does not overlap", Rect{0, 10, 10, 20}, false},
		{"touching on the left does not overlap", Rect{-10, 0, 0, 10}, false},
		{"far", Rect{50, 50, 60, 60}, false},
		{"covering", Rect{-5, -5, 15, 15}, true},
	}
	for _, tt := range overlaps {
		if got := r.Overlaps(tt.o); got != tt.want {
			t.Errorf("%s: Overlaps = %v, want %v", tt.name, got, tt.want)
		}
		if got := tt.o.Overlaps(r); got != tt.want {
			t.Errorf("%s: Overlaps is not symmetric", tt.name)
		}
	}
	contains := []struct {
		p    Offset
		want bool
	}{
		{Offset{0, 0}, true},
		{Offset{5, 5}, true},
		{Offset{10, 5}, false},
		{Offset{5, 10}, false},
		{Offset{-1, 5}, false},
		{Offset{9.999, 9.999}, true},
	}
	for _, tt := range contains {
		if got := r.Contains(tt.p); got != tt.want {
			t.Errorf("Contains(%v) = %v, want %v", tt.p, got, tt.want)
		}
	}
}

func TestRectLerpWithNil(t *testing.T) {
	a, b := Rect{10, 20, 30, 40}, Rect{30, 60, 50, 80}
	if RectLerp(nil, nil, 0.5) != nil {
		t.Error("nil, nil must give nil")
	}
	if got := RectLerp(nil, &b, 0.5); got == nil || *got != (Rect{15, 30, 25, 40}) {
		t.Errorf("nil, b = %v", got)
	}
	if got := RectLerp(&a, nil, 0.25); got == nil || *got != (Rect{7.5, 15, 22.5, 30}) {
		t.Errorf("a, nil = %v", got)
	}
	if got := RectLerp(&a, &b, 0.5); got == nil || *got != (Rect{20, 40, 40, 60}) {
		t.Errorf("a, b = %v", got)
	}
}

func TestRadius(t *testing.T) {
	r := Radius{4, 8}
	tests := []struct {
		name string
		got  Radius
		want Radius
	}{
		{"circular", NewRadiusCircular(3), Radius{3, 3}},
		{"elliptical", NewRadiusElliptical(3, 5), Radius{3, 5}},
		{"zero", RadiusZero, Radius{}},
		{"neg", r.Neg(), Radius{-4, -8}},
		{"add", r.Add(Radius{1, 2}), Radius{5, 10}},
		{"sub", r.Sub(Radius{1, 2}), Radius{3, 6}},
		{"mul", r.Mul(2), Radius{8, 16}},
		{"div", r.Div(4), Radius{1, 2}},
		{"trunc div", r.TruncDiv(3), Radius{1, 2}},
		{"mod", Radius{-4, 8}.Mod(3), Radius{2, 2}},
		{"lerp", r.Lerp(Radius{8, 0}, 0.5), Radius{6, 4}},
		{"clamp, no bounds", r.Clamp(), r},
		{"clamp, both", r.Clamp(RadiusClampOpts{Minimum: ptr(Radius{5, 0}), Maximum: ptr(Radius{10, 6})}), Radius{5, 6}},
		{"clamp, minimum only", r.Clamp(RadiusClampOpts{Minimum: ptr(Radius{5, 10})}), Radius{5, 10}},
		{"clamp, maximum only", r.Clamp(RadiusClampOpts{Maximum: ptr(Radius{1, 1})}), Radius{1, 1}},
		{"clamp values", r.ClampValues(RadiusClampValuesOpts{MinimumX: ptr(5.0), MaximumY: ptr(6.0)}), Radius{5, 6}},
		{"clamp values, none", r.ClampValues(), r},
		{"clamp values, y bounds", r.ClampValues(RadiusClampValuesOpts{MinimumY: ptr(9.0), MaximumX: ptr(3.0)}), Radius{3, 9}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestRadiusLerpWithNil(t *testing.T) {
	a, b := Radius{10, 20}, Radius{30, 60}
	if RadiusLerp(nil, nil, 0.5) != nil {
		t.Error("nil, nil must give nil")
	}
	if got := RadiusLerp(nil, &b, 0.5); got == nil || *got != (Radius{15, 30}) {
		t.Errorf("nil, b = %v", got)
	}
	if got := RadiusLerp(&a, nil, 0.25); got == nil || *got != (Radius{7.5, 15}) {
		t.Errorf("a, nil = %v", got)
	}
	if got := RadiusLerp(&a, &b, 0.5); got == nil || *got != (Radius{20, 40}) {
		t.Errorf("a, b = %v", got)
	}
}

func TestStrings(t *testing.T) {
	tests := []struct {
		got, want string
	}{
		{Offset{10.5, 20}.String(), "Offset(10.5, 20.0)"},
		{OffsetInfinite.String(), "Offset(Infinity, Infinity)"},
		{Size{1.25, 2}.String(), "Size(1.3, 2.0)"},
		{Rect{1, 2, 3, 4}.String(), "Rect.fromLTRB(1.0, 2.0, 3.0, 4.0)"},
		{Radius{4, 4}.String(), "Radius.circular(4.0)"},
		{Radius{4, 5}.String(), "Radius.elliptical(4.0, 5.0)"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

func TestMultipleOptionsPanics(t *testing.T) {
	defer func() {
		r := recover()
		err, ok := r.(error)
		if !ok {
			t.Fatalf("panic value %v (%T) is not an error", r, r)
		}
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T", err)
		}
		if appErr.Tag != "MultipleOptions" || !appErr.HasRoute("Radius.Clamp->oneOpts[...]->First[...].MultipleOptions") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
	}()
	Radius{1, 1}.Clamp(RadiusClampOpts{}, RadiusClampOpts{})
	t.Fatal("two options values must panic")
}
