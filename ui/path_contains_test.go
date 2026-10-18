package ui

import (
	"math"
	"math/rand"
	"testing"
)

func TestPathContains(t *testing.T) {
	rect := func() *Path { p := NewPath(); p.AddRect(Rect{0, 0, 10, 10}); return p }
	tests := []struct {
		name  string
		p     func() *Path
		point Offset
		want  bool
	}{
		{"empty path", NewPath, Offset{1, 1}, false},
		{"rect inside", rect, Offset{5, 5}, true},
		{"rect outside", rect, Offset{15, 5}, false},
		{"rect left of it", rect, Offset{-1, 5}, false},
		{"rect above it", rect, Offset{5, -1}, false},
		{"rect below it", rect, Offset{5, 11}, false},
		{"the left edge is inside", rect, Offset{0, 5}, true},
		{"the right edge is outside", rect, Offset{10, 5}, false},
		{"the top edge is inside", rect, Offset{5, 0}, true},
		{"the bottom edge is outside", rect, Offset{5, 10}, false},
		{"the top left corner is inside", rect, Offset{0, 0}, true},
		{"the bottom right corner is outside", rect, Offset{10, 10}, false},
		{"NaN", rect, Offset{math.NaN(), 5}, false},
		{"an open triangle is closed implicitly", func() *Path {
			p := NewPath()
			p.MoveTo(0, 0)
			p.LineTo(10, 0)
			p.LineTo(0, 10)
			return p
		}, Offset{2, 2}, true},
		{"outside the diagonal of that triangle", func() *Path {
			p := NewPath()
			p.MoveTo(0, 0)
			p.LineTo(10, 0)
			p.LineTo(0, 10)
			return p
		}, Offset{8, 8}, false},
		{"a counter-clockwise rectangle", func() *Path {
			p := NewPath()
			p.AddPolygon([]Offset{{0, 0}, {0, 10}, {10, 10}, {10, 0}}, true)
			return p
		}, Offset{5, 5}, true},
		{"a lone move contains nothing", func() *Path { p := NewPath(); p.MoveTo(1, 1); return p }, Offset{1, 1}, false},
		{"a line contains nothing", func() *Path { p := NewPath(); p.MoveTo(0, 0); p.LineTo(10, 10); return p }, Offset{5, 5}, false},
		{"inside a circle", func() *Path { p := NewPath(); p.AddOval(Rect{0, 0, 20, 20}); return p }, Offset{10, 10}, true},
		{"just inside a circle", func() *Path { p := NewPath(); p.AddOval(Rect{0, 0, 20, 20}); return p }, Offset{10 + 9.99, 10}, true},
		{"just outside a circle", func() *Path { p := NewPath(); p.AddOval(Rect{0, 0, 20, 20}); return p }, Offset{10 + 10.01, 10}, false},
		{"in the corner of the circle's box", func() *Path { p := NewPath(); p.AddOval(Rect{0, 0, 20, 20}); return p }, Offset{1, 1}, false},
		{"inside a quad", func() *Path {
			p := NewPath()
			p.MoveTo(0, 0)
			p.QuadraticBezierTo(5, 20, 10, 0)
			return p
		}, Offset{5, 5}, true},
		{"above the top of a quad", func() *Path {
			p := NewPath()
			p.MoveTo(0, 0)
			p.QuadraticBezierTo(5, 20, 10, 0)
			return p
		}, Offset{5, 11}, false},
		{"inside a cubic loop", func() *Path {
			p := NewPath()
			p.MoveTo(0, 0)
			p.CubicTo(0, 20, 10, 20, 10, 0)
			p.Close()
			return p
		}, Offset{5, 8}, true},
		{"outside a cubic loop", func() *Path {
			p := NewPath()
			p.MoveTo(0, 0)
			p.CubicTo(0, 20, 10, 20, 10, 0)
			p.Close()
			return p
		}, Offset{1, 12}, false},
	}
	for _, tt := range tests {
		if got := tt.p().Contains(tt.point); got != tt.want {
			t.Errorf("%s: Contains(%v) = %v, want %v", tt.name, tt.point, got, tt.want)
		}
	}
}

func TestPathContainsFillTypes(t *testing.T) {
	nested := func(sameDirection bool) *Path {
		p := NewPath()
		p.AddRect(Rect{0, 0, 30, 30})
		if sameDirection {
			p.AddRect(Rect{10, 10, 20, 20})
		} else {
			p.AddPolygon([]Offset{{10, 10}, {10, 20}, {20, 20}, {20, 10}}, true)
		}
		return p
	}
	tests := []struct {
		name string
		p    *Path
		fill PathFillType
		pt   Offset
		want bool
	}{
		{"same direction, non-zero: the inner area has winding 2", nested(true), PathFillTypeNonZero, Offset{15, 15}, true},
		{"same direction, even-odd: the inner area is a hole", nested(true), PathFillTypeEvenOdd, Offset{15, 15}, false},
		{"opposite directions, non-zero: a hole", nested(false), PathFillTypeNonZero, Offset{15, 15}, false},
		{"opposite directions, even-odd: a hole", nested(false), PathFillTypeEvenOdd, Offset{15, 15}, false},
		{"the ring itself, non-zero", nested(true), PathFillTypeNonZero, Offset{5, 5}, true},
		{"the ring itself, even-odd", nested(true), PathFillTypeEvenOdd, Offset{5, 5}, true},
		{"outside both", nested(true), PathFillTypeEvenOdd, Offset{40, 15}, false},
	}
	for _, tt := range tests {
		tt.p.SetFillType(tt.fill)
		if got := tt.p.Contains(tt.pt); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
	// A pentagram: the center is inside with non-zero and outside with even-odd.
	star := NewPath()
	var pts []Offset
	for i := 0; i < 5; i++ {
		a := -math.Pi/2 + float64(i)*4*math.Pi/5
		pts = append(pts, Offset{50 + 40*math.Cos(a), 50 + 40*math.Sin(a)})
	}
	star.AddPolygon(pts, true)
	if !star.Contains(Offset{50, 50}) {
		t.Error("the center of a pentagram is inside with non-zero")
	}
	star.SetFillType(PathFillTypeEvenOdd)
	if star.Contains(Offset{50, 50}) {
		t.Error("the center of a pentagram is outside with even-odd")
	}
	if !star.Contains(Offset{50, 20}) {
		t.Error("a point of the star is inside with even-odd")
	}
}

func TestPathContainsAgreesWithTheShapes(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	oval := Rect{10, 20, 90, 60} // center (50, 40), radii 40 and 20
	op := NewPath()
	op.AddOval(oval)
	for i := 0; i < 3000; i++ {
		pt := Offset{rng.Float64() * 100, rng.Float64() * 80}
		d := math.Hypot((pt.Dx-50)/40, (pt.Dy-40)/20)
		if math.Abs(d-1) < 1e-6 {
			continue
		}
		if got, want := op.Contains(pt), d < 1; got != want {
			t.Fatalf("oval Contains(%v) = %v, want %v (d = %v)", pt, got, want, d)
		}
	}
	// A circle made of an arc and one made by addArc.
	ap := NewPath()
	ap.MoveTo(100, 50)
	ap.ArcToPoint(Offset{0, 50}, ArcToPointOpts{Radius: Radius{50, 50}})
	ap.ArcToPoint(Offset{100, 50}, ArcToPointOpts{Radius: Radius{50, 50}})
	for i := 0; i < 2000; i++ {
		pt := Offset{rng.Float64() * 110, rng.Float64() * 110}
		d := math.Hypot(pt.Dx-50, pt.Dy-50) / 50
		if math.Abs(d-1) < 1e-6 {
			continue
		}
		if got, want := ap.Contains(pt), d < 1; got != want {
			t.Fatalf("arc circle Contains(%v) = %v, want %v", pt, got, want)
		}
	}
	// A cubic and a quad against a polyline of the same curve.
	cp := NewPath()
	cp.MoveTo(0, 50)
	cp.CubicTo(30, -40, 70, 140, 100, 50)
	cp.Close()
	poly := cp.Flatten(nil, 1e-4)[0].Points
	pp := NewPath()
	pp.AddPolygon(poly, true)
	for i := 0; i < 3000; i++ {
		pt := Offset{rng.Float64() * 100, rng.Float64() * 100}
		if cp.Contains(pt) != pp.Contains(pt) {
			// Allow only points that are within the flattening error of the outline.
			if !nearOutline(poly, pt, 1e-3) {
				t.Fatalf("cubic Contains(%v) = %v, polyline says %v", pt, cp.Contains(pt), pp.Contains(pt))
			}
		}
	}
}

// nearOutline reports whether the point is within eps of a segment of the
// closed polyline.
func nearOutline(poly []Offset, pt Offset, eps float64) bool {
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		ab := b.Sub(a)
		l2 := ab.Dx*ab.Dx + ab.Dy*ab.Dy
		tt := 0.0
		if l2 > 0 {
			tt = math.Max(0, math.Min(1, ((pt.Dx-a.Dx)*ab.Dx+(pt.Dy-a.Dy)*ab.Dy)/l2))
		}
		c := a.Add(ab.Mul(tt))
		if math.Hypot(pt.Dx-c.Dx, pt.Dy-c.Dy) <= eps {
			return true
		}
	}
	return false
}
