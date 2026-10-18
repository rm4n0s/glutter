package ui

import (
	"math"
	"testing"
)

func TestPathFlattenLines(t *testing.T) {
	p := NewPath()
	p.MoveTo(0, 0)
	p.LineTo(10, 0)
	p.LineTo(10, 10)
	p.Close()
	p.MoveTo(20, 20)
	p.LineTo(30, 20)
	p.MoveTo(50, 50) // a lone move
	p.MoveTo(60, 60)
	p.Close() // closed lone move
	got := p.Flatten(nil, 0)
	if len(got) != 3 {
		t.Fatalf("%d contours %+v, want 3", len(got), got)
	}
	pointsNear(t, "triangle", got[0].Points, []Offset{{0, 0}, {10, 0}, {10, 10}}, 0)
	if !got[0].Closed || got[1].Closed || !got[2].Closed {
		t.Errorf("closed flags: %v %v %v", got[0].Closed, got[1].Closed, got[2].Closed)
	}
	pointsNear(t, "open line", got[1].Points, []Offset{{20, 20}, {30, 20}}, 0)
	pointsNear(t, "closed point", got[2].Points, []Offset{{60, 60}}, 0)
}

func TestPathFlattenTransform(t *testing.T) {
	p := NewPath()
	p.MoveTo(1, 1)
	p.LineTo(2, 1)
	got := p.Flatten(NewMatrix4Diagonal3Values(10, 20, 1), 0)
	pointsNear(t, "scaled", got[0].Points, []Offset{{10, 20}, {20, 20}}, 1e-6)
	// The identity is the same as no matrix.
	pointsNear(t, "identity", p.Flatten(NewMatrix4Identity(), 0)[0].Points, []Offset{{1, 1}, {2, 1}}, 0)
}

// maxDeviationFromCircle is the largest distance of the vertices and the middles
// of the edges of the polyline from the circle.
func maxDeviationFromCircle(pts []Offset, center Offset, r float64) float64 {
	worst := 0.0
	for i, a := range pts {
		worst = math.Max(worst, math.Abs(math.Hypot(a.Dx-center.Dx, a.Dy-center.Dy)-r))
		b := pts[(i+1)%len(pts)]
		mid := lerpPt(a, b, 0.5)
		worst = math.Max(worst, math.Abs(math.Hypot(mid.Dx-center.Dx, mid.Dy-center.Dy)-r))
	}
	return worst
}

func TestPathFlattenCurvesStayWithinTolerance(t *testing.T) {
	circle := NewPath()
	circle.AddOval(Rect{-50, -50, 50, 50})
	for _, tol := range []float64{1, 0.25, 0.01, 0.001} {
		got := circle.Flatten(nil, tol)
		if len(got) != 1 || !got[0].Closed {
			t.Fatalf("tol %v: %+v", tol, got)
		}
		// The vertices are on the circle; the edges sag inwards by at most tol.
		dev := maxDeviationFromCircle(got[0].Points, Offset{}, 50)
		if dev > tol*1.01 {
			t.Errorf("tolerance %v: deviation %v with %d points", tol, dev, len(got[0].Points))
		}
	}
	// A finer tolerance gives more points.
	coarse := len(circle.Flatten(nil, 1)[0].Points)
	fine := len(circle.Flatten(nil, 0.001)[0].Points)
	if fine <= coarse*5 {
		t.Errorf("points: %d at 1, %d at 0.001", coarse, fine)
	}
	// The tolerance is in the units after the transform: a path scaled up needs
	// more points to stay as close.
	small := len(circle.Flatten(NewMatrix4Diagonal3Values(0.1, 0.1, 1), 0.1)[0].Points)
	big := len(circle.Flatten(NewMatrix4Diagonal3Values(10, 10, 1), 0.1)[0].Points)
	if big <= small {
		t.Errorf("scaled up %d points, scaled down %d", big, small)
	}
}

func TestPathFlattenQuadAndCubic(t *testing.T) {
	q := NewPath()
	q.MoveTo(0, 0)
	q.QuadraticBezierTo(50, 100, 100, 0)
	pts := q.Flatten(nil, 0.01)[0].Points
	// Every point is on the quad y = 100*t*(1-t)*2, x = 100*t: y = 2x(100-x)/100... check the curve.
	for _, pt := range pts {
		tt := pt.Dx / 100
		if !near(pt.Dy, 100*2*tt*(1-tt), 1e-9) {
			t.Fatalf("%v is not on the quad", pt)
		}
	}
	if pts[0] != (Offset{0, 0}) || pts[len(pts)-1] != (Offset{100, 0}) {
		t.Errorf("ends: %v %v", pts[0], pts[len(pts)-1])
	}
	c := NewPath()
	c.MoveTo(0, 0)
	c.CubicTo(0, 100, 100, 100, 100, 0)
	cp := c.Flatten(nil, 0.01)[0].Points
	if cp[0] != (Offset{0, 0}) || cp[len(cp)-1] != (Offset{100, 0}) || len(cp) < 10 {
		t.Errorf("cubic points: %d, ends %v %v", len(cp), cp[0], cp[len(cp)-1])
	}
	// The midpoints of the edges are within the tolerance of the curve.
	for i := 0; i+1 < len(cp); i++ {
		mid := lerpPt(cp[i], cp[i+1], 0.5)
		best := math.Inf(1)
		for k := 0; k <= 40000; k++ {
			pt := cubicAt(Offset{0, 0}, Offset{0, 100}, Offset{100, 100}, Offset{100, 0}, float64(k)/40000)
			best = math.Min(best, math.Hypot(pt.Dx-mid.Dx, pt.Dy-mid.Dy))
		}
		if best > 0.0101 {
			t.Fatalf("edge %d is %v from the curve", i, best)
		}
	}
}

func TestPathFlattenIsBounded(t *testing.T) {
	// A huge curve with a tiny tolerance does not make an enormous list.
	p := NewPath()
	p.MoveTo(0, 0)
	p.CubicTo(1e9, 1e9, -1e9, 1e9, 0, 0)
	pts := p.Flatten(nil, 1e-9)[0].Points
	if len(pts) > maxFlattenSplits+2 {
		t.Errorf("%d points", len(pts))
	}
	// And NaN in a curve does not hang or explode.
	n := NewPath()
	n.MoveTo(0, 0)
	n.QuadraticBezierTo(math.NaN(), 0, 1, 1)
	if got := n.Flatten(nil, 0.1); len(got) != 1 {
		t.Errorf("%+v", got)
	}
}

func TestPathFingerprint(t *testing.T) {
	build := func(fn func(p *Path)) uint64 {
		p := NewPath()
		fn(p)
		return p.Fingerprint()
	}
	square := func(p *Path) { p.AddRect(NewRectFromLTWH(0, 0, 10, 10)) }
	base := build(square)
	tests := []struct {
		name string
		fn   func(p *Path)
		same bool
	}{
		{"the same contents", square, true},
		{"built by hand with the same verbs and points", func(p *Path) {
			p.MoveTo(0, 0)
			p.LineTo(10, 0)
			p.LineTo(10, 10)
			p.LineTo(0, 10)
			p.Close()
		}, true}, // the fingerprint is of the contents, not of how the path was built
		{"the same square with another start", func(p *Path) {
			p.MoveTo(10, 0)
			p.LineTo(10, 10)
			p.LineTo(0, 10)
			p.LineTo(0, 0)
			p.Close()
		}, false},
		{"moved", func(p *Path) { p.AddRect(NewRectFromLTWH(1, 0, 10, 10)) }, false},
		{"a different size", func(p *Path) { p.AddRect(NewRectFromLTWH(0, 0, 10, 11)) }, false},
		{"another fill type", func(p *Path) { square(p); p.SetFillType(PathFillTypeEvenOdd) }, false},
		{"a second contour", func(p *Path) { square(p); p.AddRect(NewRectFromLTWH(20, 0, 5, 5)) }, false},
		{"a conic weight", func(p *Path) { p.MoveTo(0, 0); p.ConicTo(5, 0, 5, 5, 0.5) }, false},
	}
	for _, tt := range tests {
		got := build(tt.fn)
		if (got == base) != tt.same {
			t.Errorf("%s: fingerprint equal to the square's = %v, want %v", tt.name, got == base, tt.same)
		}
	}
	if build(func(p *Path) { p.MoveTo(0, 0); p.ConicTo(5, 0, 5, 5, 0.5) }) == build(func(p *Path) { p.MoveTo(0, 0); p.ConicTo(5, 0, 5, 5, 0.7) }) {
		t.Error("conics with different weights have the same fingerprint")
	}
	e1, e2 := NewPath(), NewPath()
	if e1.Fingerprint() != e2.Fingerprint() {
		t.Error("empty paths differ")
	}
	// A fill type left unspecified is the default one.
	a, b := NewPath(), NewPath()
	b.SetFillType(PathFillTypeNonZero)
	if a.Fingerprint() != b.Fingerprint() {
		t.Error("the default fill type and nonZero differ")
	}
}
