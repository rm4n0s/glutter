package ui

import (
	"math"
	"testing"
)

func TestRRectConstructors(t *testing.T) {
	r := Rect{10, 20, 110, 220}
	round := RRect{10, 20, 110, 220, 5, 6, 5, 6, 5, 6, 5, 6}
	corners := Corners{
		TopLeft: Radius{1, 2}, TopRight: Radius{3, 4}, BottomRight: Radius{5, 6}, BottomLeft: Radius{7, 8},
	}
	cornered := RRect{
		Left: 10, Top: 20, Right: 110, Bottom: 220,
		TlRadiusX: 1, TlRadiusY: 2, TrRadiusX: 3, TrRadiusY: 4, BrRadiusX: 5, BrRadiusY: 6, BlRadiusX: 7, BlRadiusY: 8,
	}
	tests := []struct {
		name string
		got  RRect
		want RRect
	}{
		{"LTRBXY", NewRRectFromLTRBXY(10, 20, 110, 220, 5, 6), round},
		{"LTRBR", NewRRectFromLTRBR(10, 20, 110, 220, Radius{5, 6}), round},
		{"RectXY", NewRRectFromRectXY(r, 5, 6), round},
		{"RectAndRadius", NewRRectFromRectAndRadius(r, Radius{5, 6}), round},
		{"LTRBAndCorners", NewRRectFromLTRBAndCorners(10, 20, 110, 220, corners), cornered},
		{"RectAndCorners", NewRRectFromRectAndCorners(r, corners), cornered},
		{"LTRBAndCorners without corners is a rectangle", NewRRectFromLTRBAndCorners(10, 20, 110, 220), RRect{Left: 10, Top: 20, Right: 110, Bottom: 220}},
		{"zero", RRectZero, RRect{}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	if cornered.TlRadius() != (Radius{1, 2}) || cornered.TrRadius() != (Radius{3, 4}) ||
		cornered.BrRadius() != (Radius{5, 6}) || cornered.BlRadius() != (Radius{7, 8}) {
		t.Error("corner radii read from the wrong fields")
	}
}

func TestRRectMeasures(t *testing.T) {
	rr := NewRRectFromLTRBXY(0, 0, 100, 60, 10, 20)
	if rr.Width() != 100 || rr.Height() != 60 || rr.OuterRect() != (Rect{0, 0, 100, 60}) || rr.Center() != (Offset{50, 30}) {
		t.Errorf("Width/Height/OuterRect/Center = %v %v %v %v", rr.Width(), rr.Height(), rr.OuterRect(), rr.Center())
	}
	if rr.ShortestSide() != 60 || rr.LongestSide() != 100 {
		t.Errorf("sides = %v %v", rr.ShortestSide(), rr.LongestSide())
	}
	tests := []struct {
		name string
		got  Rect
		want Rect
	}{
		{"middleRect", rr.MiddleRect(), Rect{10, 20, 90, 40}},
		{"wideMiddleRect", rr.WideMiddleRect(), Rect{0, 20, 100, 40}},
		{"tallMiddleRect", rr.TallMiddleRect(), Rect{10, 0, 90, 60}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	inner := rr.SafeInnerRect()
	want := Rect{10 * 0.29289321881, 20 * 0.29289321881, 100 - 10*0.29289321881, 60 - 20*0.29289321881}
	if inner != want {
		t.Errorf("SafeInnerRect = %v, want %v", inner, want)
	}
	// The corners differ: the largest radius on each side counts.
	odd := NewRRectFromLTRBAndCorners(0, 0, 100, 100, Corners{
		TopLeft: Radius{10, 5}, TopRight: Radius{20, 30}, BottomRight: Radius{15, 8}, BottomLeft: Radius{12, 2},
	})
	if got, want := odd.MiddleRect(), (Rect{12, 30, 80, 92}); got != want {
		t.Errorf("MiddleRect of mixed corners = %v, want %v", got, want)
	}
}

func TestRRectShapes(t *testing.T) {
	tests := []struct {
		name                                          string
		rr                                            RRect
		isRect, isStadium, isEllipse, isCircle, empty bool
	}{
		{"plain rectangle", NewRRectFromLTRBXY(0, 0, 100, 50, 0, 0), true, false, false, false, false},
		{"one zero radius axis is a rectangle", NewRRectFromLTRBXY(0, 0, 100, 50, 0, 10), true, false, false, false, false},
		{"rounded rectangle", NewRRectFromLTRBXY(0, 0, 100, 50, 10, 10), false, false, false, false, false},
		{"stadium", NewRRectFromLTRBXY(0, 0, 100, 50, 25, 25), false, true, false, false, false},
		{"ellipse", NewRRectFromLTRBXY(0, 0, 100, 50, 50, 25), false, true, true, false, false},
		{"circle", NewRRectFromLTRBXY(0, 0, 50, 50, 25, 25), false, true, true, true, false},
		{"different corners", NewRRectFromLTRBAndCorners(0, 0, 100, 50, Corners{TopLeft: Radius{25, 25}}), false, false, false, false, false},
		{"empty (zero width fits inside the radii)", NewRRectFromLTRBXY(10, 10, 10, 20, 1, 1), false, true, false, false, true},
	}
	for _, tt := range tests {
		if tt.rr.IsRect() != tt.isRect || tt.rr.IsStadium() != tt.isStadium || tt.rr.IsEllipse() != tt.isEllipse ||
			tt.rr.IsCircle() != tt.isCircle || tt.rr.IsEmpty() != tt.empty {
			t.Errorf("%s: rect/stadium/ellipse/circle/empty = %v/%v/%v/%v/%v, want %v/%v/%v/%v/%v", tt.name,
				tt.rr.IsRect(), tt.rr.IsStadium(), tt.rr.IsEllipse(), tt.rr.IsCircle(), tt.rr.IsEmpty(),
				tt.isRect, tt.isStadium, tt.isEllipse, tt.isCircle, tt.empty)
		}
	}
	nan := math.NaN()
	if !(RRect{TlRadiusX: nan}).HasNaN() || (RRect{}).HasNaN() || (RRect{Right: math.Inf(1)}).IsFinite() || !(RRect{}).IsFinite() {
		t.Error("HasNaN/IsFinite wrong")
	}
}

func TestRRectShiftInflateDeflate(t *testing.T) {
	rr := NewRRectFromLTRBXY(10, 20, 110, 70, 5, 8)
	tests := []struct {
		name string
		got  RRect
		want RRect
	}{
		{"shift", rr.Shift(Offset{1, 2}), NewRRectFromLTRBXY(11, 22, 111, 72, 5, 8)},
		{"inflate", rr.Inflate(2), NewRRectFromLTRBXY(8, 18, 112, 72, 7, 10)},
		{"deflate", rr.Deflate(2), NewRRectFromLTRBXY(12, 22, 108, 68, 3, 6)},
		{"deflate clamps the radii at zero", rr.Deflate(6), NewRRectFromLTRBXY(16, 26, 104, 64, 0, 2)},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestRRectScaleRadii(t *testing.T) {
	tests := []struct {
		name string
		rr   RRect
		want RRect
	}{
		{"fits", NewRRectFromLTRBXY(0, 0, 100, 100, 10, 10), NewRRectFromLTRBXY(0, 0, 100, 100, 10, 10)},
		{"x radii too wide", NewRRectFromLTRBXY(0, 0, 100, 100, 100, 10), NewRRectFromLTRBXY(0, 0, 100, 100, 50, 5)},
		{"y radii too tall", NewRRectFromLTRBXY(0, 0, 100, 40, 10, 40), NewRRectFromLTRBXY(0, 0, 100, 40, 5, 20)},
		{"only one pair too big", NewRRectFromLTRBAndCorners(0, 0, 100, 100, Corners{TopLeft: Radius{80, 0}, TopRight: Radius{80, 0}}),
			NewRRectFromLTRBAndCorners(0, 0, 100, 100, Corners{TopLeft: Radius{50, 0}, TopRight: Radius{50, 0}})},
		{"zero radii", NewRRectFromLTRBXY(0, 0, 0, 0, 0, 0), NewRRectFromLTRBXY(0, 0, 0, 0, 0, 0)},
	}
	for _, tt := range tests {
		if got := tt.rr.ScaleRadii(); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestRRectContains(t *testing.T) {
	rr := NewRRectFromLTRBXY(0, 0, 100, 100, 20, 20)
	tests := []struct {
		name string
		p    Offset
		want bool
	}{
		{"center", Offset{50, 50}, true},
		{"edge midpoint", Offset{0, 50}, true},
		{"outside left", Offset{-1, 50}, false},
		{"right edge is outside", Offset{100, 50}, false},
		{"bottom edge is outside", Offset{50, 100}, false},
		{"top left corner point is cut", Offset{1, 1}, false},
		{"just inside the top left arc", Offset{20 - 20*math.Sqrt2/2 + 0.1, 20 - 20*math.Sqrt2/2 + 0.1}, true},
		{"just outside the top left arc", Offset{20 - 20*math.Sqrt2/2 - 0.1, 20 - 20*math.Sqrt2/2 - 0.1}, false},
		{"top right corner point is cut", Offset{99, 1}, false},
		{"bottom right corner point is cut", Offset{99, 99}, false},
		{"bottom left corner point is cut", Offset{1, 99}, false},
		{"on the arc's own box edge", Offset{20, 0}, true},
		{"inside a corner box and the arc", Offset{19, 19}, true},
	}
	for _, tt := range tests {
		if got := rr.Contains(tt.p); got != tt.want {
			t.Errorf("%s: Contains(%v) = %v, want %v", tt.name, tt.p, got, tt.want)
		}
	}
	// Radii that do not fit are scaled before testing, so the shape is a stadium.
	big := NewRRectFromLTRBXY(0, 0, 100, 20, 1000, 1000)
	if !big.Contains(Offset{50, 10}) || big.Contains(Offset{1, 1}) || big.Contains(Offset{99, 19}) {
		t.Error("Contains must scale the radii first")
	}
	// A plain rectangle contains everything but its far edges.
	plain := NewRRectFromLTRBXY(0, 0, 10, 10, 0, 0)
	if !plain.Contains(Offset{0, 0}) || plain.Contains(Offset{10, 10}) || !plain.Contains(Offset{9.99, 9.99}) {
		t.Error("a rectangle contains its near corner and not its far one")
	}
}

func TestRRectLerp(t *testing.T) {
	a := NewRRectFromLTRBXY(0, 0, 100, 100, 10, 20)
	b := NewRRectFromLTRBXY(20, 40, 60, 140, 30, 0)
	tests := []struct {
		name string
		a, b *RRect
		t    float64
		want *RRect
	}{
		{"both nil", nil, nil, 0.5, nil},
		{"a only: shrinks to the origin", &a, nil, 0.5, ptr(NewRRectFromLTRBXY(0, 0, 50, 50, 5, 10))},
		{"b only: grows from the origin", nil, &b, 0.5, ptr(NewRRectFromLTRBXY(10, 20, 30, 70, 15, 0))},
		{"both", &a, &b, 0.5, ptr(NewRRectFromLTRBXY(10, 20, 80, 120, 20, 10))},
		{"negative radii clamp at zero", &a, &b, 2, ptr(NewRRectFromLTRBXY(40, 80, 20, 180, 50, 0))},
	}
	for _, tt := range tests {
		got := RRectLerp(tt.a, tt.b, tt.t)
		if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
	if got := a.Lerp(b, 0.5); got != *tests[3].want {
		t.Errorf("Lerp = %v", got)
	}
}

func TestRRectString(t *testing.T) {
	tests := []struct {
		rr   RRect
		want string
	}{
		{NewRRectFromLTRBXY(1, 2, 3, 4, 5, 5), "RRect.fromLTRBR(1.0, 2.0, 3.0, 4.0, 5.0)"},
		{NewRRectFromLTRBXY(1, 2, 3, 4, 5, 6), "RRect.fromLTRBXY(1.0, 2.0, 3.0, 4.0, 5.0, 6.0)"},
		{
			NewRRectFromLTRBAndCorners(1, 2, 3, 4, Corners{TopLeft: Radius{1, 1}, TopRight: Radius{2, 3}}),
			"RRect.fromLTRBAndCorners(1.0, 2.0, 3.0, 4.0, topLeft: Radius.circular(1.0), topRight: Radius.elliptical(2.0, 3.0), bottomRight: Radius.circular(0.0), bottomLeft: Radius.circular(0.0))",
		},
	}
	for _, tt := range tests {
		if got := tt.rr.String(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
	if got := RSuperellipse(NewRRectFromLTRBXY(1, 2, 3, 4, 5, 5)).String(); got != "RSuperellipse.fromLTRBR(1.0, 2.0, 3.0, 4.0, 5.0)" {
		t.Errorf("RSuperellipse string = %q", got)
	}
}

func TestRSuperellipse(t *testing.T) {
	rs := NewRSuperellipseFromRectAndRadius(Rect{0, 0, 100, 100}, Radius{30, 30})
	rr := NewRRectFromRectAndRadius(Rect{0, 0, 100, 100}, Radius{30, 30})
	tests := []struct {
		name string
		p    Offset
		want bool
	}{
		{"center", Offset{50, 50}, true},
		{"edge midpoint", Offset{0, 50}, true},
		{"outside", Offset{-1, 50}, false},
		{"right edge is outside", Offset{100, 50}, false},
		{"corner tip", Offset{1, 1}, false},
		{"near the corner it is fuller than a circular corner", Offset{6, 6}, true},
		{"but still cut", Offset{3, 3}, false},
	}
	for _, tt := range tests {
		if got := rs.Contains(tt.p); got != tt.want {
			t.Errorf("%s: Contains(%v) = %v, want %v", tt.name, tt.p, got, tt.want)
		}
	}
	if rr.Contains(Offset{6, 6}) {
		t.Error("the circular corner must not contain (6, 6): that is what makes the squircle different")
	}
	// Radii of zero make a rectangle.
	plain := NewRSuperellipseFromLTRBXY(0, 0, 10, 10, 0, 0)
	if !plain.Contains(Offset{0, 0}) || plain.Contains(Offset{10, 5}) {
		t.Error("a superellipse with no radius is a rectangle")
	}
	// A stadium's corners cover the whole short side.
	pill := NewRSuperellipseFromRectAndRadius(Rect{0, 0, 100, 20}, Radius{50, 50})
	if !pill.Contains(Offset{50, 10}) || pill.Contains(Offset{0, 0}) {
		t.Error("a pill contains its middle and not its corner")
	}
	// The shared methods work on RSuperellipse too.
	if rs.Shift(Offset{5, 5}).Left != 5 || rs.Inflate(1).Left != -1 || rs.IsEmpty() || rs.ScaleRadii() != rs {
		t.Error("shared methods wrong on RSuperellipse")
	}
	if got := RSuperellipseLerp(nil, nil, 0.5); got != nil {
		t.Errorf("Lerp(nil, nil) = %v", got)
	}
	half := RSuperellipseLerp(&rs, nil, 0.5)
	if half == nil || half.Right != 50 || half.TlRadiusX != 15 {
		t.Errorf("Lerp(rs, nil, 0.5) = %v", half)
	}
	if got := rs.Lerp(plain, 0.5); got.TlRadiusX != 15 || got.Right != 55 {
		t.Errorf("Lerp = %v", got)
	}
}

func TestRSTransform(t *testing.T) {
	tests := []struct {
		name string
		got  RSTransform
		want RSTransform
	}{
		{"raw", NewRSTransform(1, 2, 3, 4), RSTransform{1, 2, 3, 4}},
		{"identity", NewRSTransformFromComponents(0, 1, 0, 0, 0, 0), RSTransform{1, 0, 0, 0}},
		{"scale 2, anchor moves to the translation", NewRSTransformFromComponents(0, 2, 5, 5, 100, 100), RSTransform{2, 0, 90, 90}},
		{"quarter turn", NewRSTransformFromComponents(math.Pi/2, 1, 10, 0, 0, 0), RSTransform{float32(math.Cos(math.Pi / 2)), 1, float32(-math.Cos(math.Pi/2) * 10), float32(-10)}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
