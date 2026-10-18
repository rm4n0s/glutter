package ui

import (
	"math"
	"testing"
)

// onEllipse reports whether p is on the ellipse with the given center and radii.
func onEllipse(p, center Offset, rx, ry, eps float64) bool {
	dx, dy := (p.Dx-center.Dx)/rx, (p.Dy-center.Dy)/ry
	return math.Abs(math.Hypot(dx, dy)-1) <= eps
}

// pathEndPoints returns the point each segment ends at.
func pathEndPoints(p *Path) []Offset {
	var out []Offset
	p.ForEachSegment(func(s PathSegment) {
		switch s.Verb {
		case PathVerbMove:
			out = append(out, s.P0)
		case PathVerbLine:
			out = append(out, s.P1)
		case PathVerbQuad, PathVerbConic:
			out = append(out, s.P2)
		case PathVerbCubic:
			out = append(out, s.P3)
		}
	})
	return out
}

func TestPathAddArc(t *testing.T) {
	oval := Rect{0, 0, 20, 10}
	center := Offset{10, 5}
	tests := []struct {
		name         string
		start, sweep float64 // radians
		verbs        string
		first, last  Offset
	}{
		{"a quarter clockwise from the right", 0, math.Pi / 2, "MO", Offset{20, 5}, Offset{10, 10}},
		{"a quarter counter-clockwise from the right", 0, -math.Pi / 2, "MO", Offset{20, 5}, Offset{10, 0}},
		{"a half", 0, math.Pi, "MOO", Offset{20, 5}, Offset{0, 5}},
		{"three quarters", 0, 3 * math.Pi / 2, "MOOO", Offset{20, 5}, Offset{10, 0}},
		{"an eighth makes one short conic", 0, math.Pi / 4, "MO", Offset{20, 5}, Offset{10 + 10*math.Cos(math.Pi/4), 5 + 5*math.Sin(math.Pi/4)}},
		{"from the bottom", math.Pi / 2, math.Pi / 2, "MO", Offset{10, 10}, Offset{0, 5}},
		{"a quarter across quadrants is one conic", math.Pi / 4, math.Pi / 2, "MO", Offset{10 + 10*math.Cos(math.Pi/4), 5 + 5*math.Sin(math.Pi/4)}, Offset{10 + 10*math.Cos(3*math.Pi/4), 5 + 5*math.Sin(3*math.Pi/4)}},
	}
	for _, tt := range tests {
		p := NewPath()
		p.AddArc(oval, tt.start, tt.sweep)
		if got := verbString(p); got != tt.verbs {
			t.Errorf("%s: verbs %s, want %s", tt.name, got, tt.verbs)
			continue
		}
		ends := pathEndPoints(p)
		pointsNear(t, tt.name, []Offset{ends[0], ends[len(ends)-1]}, []Offset{tt.first, tt.last}, 1e-6)
		for i, pt := range ends {
			if !onEllipse(pt, center, 10, 5, 1e-6) {
				t.Errorf("%s: end point %d %v is not on the ellipse", tt.name, i, pt)
			}
		}
	}

	t.Run("a full turn from a quadrant start is an oval", func(t *testing.T) {
		p := NewPath()
		p.AddArc(oval, 0, 2*math.Pi)
		o := NewPath()
		o.AddOval(oval)
		if verbString(p) != verbString(o) {
			t.Errorf("verbs %s, want %s", verbString(p), verbString(o))
		}
		pointsNear(t, "full turn", p.points, o.points, 1e-9)
		// Counter-clockwise from the bottom (index 2): the oval goes the other way.
		q := NewPath()
		q.AddArc(oval, math.Pi/2, -2*math.Pi)
		if verbString(q) != "MOOOOZ" || !near(q.points[0].Dx, 10, 1e-9) || !near(q.points[0].Dy, 10, 1e-9) {
			t.Errorf("counter-clockwise full turn: %s %v", verbString(q), q.points[:1])
		}
	})
	t.Run("a full turn from another start is an arc that closes on itself", func(t *testing.T) {
		p := NewPath()
		p.AddArc(oval, 0.3, 2*math.Pi)
		if verbString(p) == "MOOOOZ" {
			t.Error("an arc from 0.3 radians is not an oval path")
		}
		ends := pathEndPoints(p)
		if !near(ends[0].Dx, ends[len(ends)-1].Dx, 1e-3) || !near(ends[0].Dy, ends[len(ends)-1].Dy, 1e-3) {
			t.Errorf("start %v and end %v should meet", ends[0], ends[len(ends)-1])
		}
	})
	t.Run("nothing is added for an empty oval or no sweep", func(t *testing.T) {
		p := NewPath()
		p.AddArc(Rect{0, 0, 0, 10}, 0, 1)
		p.AddArc(oval, 0, 0)
		if !p.IsEmpty() {
			t.Errorf("verbs %s", verbString(p))
		}
	})
}

func TestPathArcTo(t *testing.T) {
	oval := Rect{0, 0, 20, 20}
	t.Run("force move to", func(t *testing.T) {
		p := NewPath()
		p.MoveTo(100, 100)
		p.ArcTo(oval, 0, math.Pi/2, true)
		if verbString(p) != "MMO" || p.points[1] != (Offset{20, 10}) {
			t.Errorf("%s %v", verbString(p), p.points)
		}
	})
	t.Run("joined with a line", func(t *testing.T) {
		p := NewPath()
		p.MoveTo(100, 100)
		p.ArcTo(oval, 0, math.Pi/2, false)
		if verbString(p) != "MLO" || p.points[1] != (Offset{20, 10}) {
			t.Errorf("%s %v", verbString(p), p.points)
		}
	})
	t.Run("an empty path always starts with a move", func(t *testing.T) {
		p := NewPath()
		p.ArcTo(oval, 0, math.Pi/2, false)
		if verbString(p) != "MO" {
			t.Errorf("%s", verbString(p))
		}
	})
	t.Run("contiguous arcs from the same oval are not joined by lines", func(t *testing.T) {
		p := NewPath()
		p.ArcTo(oval, 0, math.Pi/2, true)
		p.ArcTo(oval, math.Pi/2, math.Pi/2, false)
		if verbString(p) != "MOO" {
			t.Errorf("%s", verbString(p))
		}
	})
	t.Run("a lone point", func(t *testing.T) {
		p := NewPath()
		p.MoveTo(1, 1)
		p.ArcTo(oval, 0, 0, false)
		if verbString(p) != "ML" || p.points[1] != (Offset{20, 10}) {
			t.Errorf("%s %v", verbString(p), p.points)
		}
	})
	t.Run("a sweep too small to see", func(t *testing.T) {
		p := NewPath()
		p.ArcTo(oval, 0, 1e-9, true)
		if verbString(p) != "M" {
			t.Errorf("%s", verbString(p))
		}
	})
	t.Run("a sweep a hair short of a full turn still draws", func(t *testing.T) {
		p := NewPath()
		p.ArcTo(oval, 0, 2*math.Pi-1e-7, true)
		if len(p.verbs) < 5 {
			t.Errorf("%s", verbString(p))
		}
	})
	t.Run("a negative size draws nothing", func(t *testing.T) {
		p := NewPath()
		p.ArcTo(Rect{10, 10, 0, 0}, 0, 1, true)
		if !p.IsEmpty() {
			t.Error("a negative oval adds nothing")
		}
	})
}

func TestPathArcToPoint(t *testing.T) {
	cw, ccw := Ptr(true), Ptr(false)
	t.Run("zero radii make a line", func(t *testing.T) {
		p := NewPath()
		p.MoveTo(0, 0)
		p.ArcToPoint(Offset{10, 10})
		if verbString(p) != "ML" || p.points[1] != (Offset{10, 10}) {
			t.Errorf("%s %v", verbString(p), p.points)
		}
	})
	t.Run("the same point makes a line", func(t *testing.T) {
		p := NewPath()
		p.MoveTo(5, 5)
		p.ArcToPoint(Offset{5, 5}, ArcToPointOpts{Radius: Radius{3, 3}})
		if verbString(p) != "ML" {
			t.Errorf("%s", verbString(p))
		}
	})
	t.Run("a quarter circle", func(t *testing.T) {
		// From (10, 0) to (0, 10) with radius 10, clockwise and small: the center
		// is the origin.
		p := NewPath()
		p.MoveTo(10, 0)
		p.ArcToPoint(Offset{0, 10}, ArcToPointOpts{Radius: Radius{10, 10}})
		if verbString(p) != "MO" {
			t.Fatalf("verbs %s", verbString(p))
		}
		end := p.points[len(p.points)-1]
		if end != (Offset{0, 10}) {
			t.Errorf("the arc must end exactly at the requested point, got %v", end)
		}
		// The control point is the corner (10, 10) and the weight cos(45 deg).
		pointsNear(t, "quarter", p.points, []Offset{{10, 0}, {10, 10}, {0, 10}}, 1e-9)
		if !near(p.weights[0], math.Sqrt2/2, 1e-12) {
			t.Errorf("weight = %v", p.weights[0])
		}
	})
	t.Run("the direction picks the center", func(t *testing.T) {
		// The other direction goes around the other way: the center is (10, 10).
		p := NewPath()
		p.MoveTo(10, 0)
		p.ArcToPoint(Offset{0, 10}, ArcToPointOpts{Radius: Radius{10, 10}, Clockwise: ccw})
		if verbString(p) != "MO" {
			t.Fatalf("verbs %s", verbString(p))
		}
		pointsNear(t, "ccw", p.points, []Offset{{10, 0}, {0, 0}, {0, 10}}, 1e-9)
	})
	t.Run("large arcs", func(t *testing.T) {
		// A large clockwise arc from (10, 0) to (0, 10) goes around the center
		// (10, 10) for 270 degrees; a large counter-clockwise one around the origin.
		p := NewPath()
		p.MoveTo(10, 0)
		p.ArcToPoint(Offset{0, 10}, ArcToPointOpts{Radius: Radius{10, 10}, LargeArc: true, Clockwise: cw})
		if verbString(p) != "MOOO" {
			t.Fatalf("verbs %s", verbString(p))
		}
		for i, pt := range pathEndPoints(p) {
			if !onEllipse(pt, Offset{10, 10}, 10, 10, 1e-6) {
				t.Errorf("end point %d %v is not on the circle around (10, 10)", i, pt)
			}
		}
		q := NewPath()
		q.MoveTo(10, 0)
		q.ArcToPoint(Offset{0, 10}, ArcToPointOpts{Radius: Radius{10, 10}, LargeArc: true, Clockwise: ccw})
		for i, pt := range pathEndPoints(q) {
			if !onEllipse(pt, Offset{0, 0}, 10, 10, 1e-6) {
				t.Errorf("end point %d %v is not on the circle around the origin", i, pt)
			}
		}
	})
	t.Run("radii that are too small are scaled up", func(t *testing.T) {
		p := NewPath()
		p.MoveTo(0, 0)
		p.ArcToPoint(Offset{20, 0}, ArcToPointOpts{Radius: Radius{1, 1}})
		// The smallest circle through the points has the radius 10, around (10, 0).
		for i, pt := range pathEndPoints(p) {
			if !onEllipse(pt, Offset{10, 0}, 10, 10, 1e-6) {
				t.Errorf("end point %d %v is not on the scaled circle", i, pt)
			}
		}
	})
	t.Run("an elliptical arc stays on its ellipse", func(t *testing.T) {
		p := NewPath()
		p.MoveTo(20, 10)
		p.ArcToPoint(Offset{0, 10}, ArcToPointOpts{Radius: Radius{10, 5}, LargeArc: false, Clockwise: cw})
		// The ellipse with the radii 10 by 5 through both points has its center at
		// (10, 10); a half ellipse clockwise from the right goes through the bottom.
		for i, pt := range pathEndPoints(p) {
			if !onEllipse(pt, Offset{10, 10}, 10, 5, 1e-6) {
				t.Errorf("end point %d %v is not on the ellipse", i, pt)
			}
		}
		mid := pathEndPoints(p)
		bottom := false
		for _, pt := range mid {
			if near(pt.Dy, 15, 1e-6) {
				bottom = true
			}
		}
		if !bottom {
			t.Errorf("the clockwise arc from the right should pass the bottom: %v", mid)
		}
	})
	t.Run("a rotated ellipse stays on its ellipse", func(t *testing.T) {
		rot := math.Pi / 6
		p := NewPath()
		// Points on the ellipse rx=10, ry=4 rotated by 30 degrees around (5, 5).
		pt := func(a float64) Offset {
			x, y := 10*math.Cos(a), 4*math.Sin(a)
			return Offset{5 + x*math.Cos(rot) - y*math.Sin(rot), 5 + x*math.Sin(rot) + y*math.Cos(rot)}
		}
		start, end := pt(0.5), pt(2.0)
		p.MoveTo(start.Dx, start.Dy)
		p.ArcToPoint(end, ArcToPointOpts{Radius: Radius{10, 4}, Rotation: rot})
		inv := func(q Offset) Offset {
			d := q.Sub(Offset{5, 5})
			return Offset{d.Dx*math.Cos(rot) + d.Dy*math.Sin(rot), -d.Dx*math.Sin(rot) + d.Dy*math.Cos(rot)}
		}
		for i, q := range pathEndPoints(p) {
			if !onEllipse(inv(q), Offset{}, 10, 4, 1e-6) {
				t.Errorf("end point %d %v is not on the rotated ellipse", i, q)
			}
		}
	})
	t.Run("relative", func(t *testing.T) {
		p := NewPath()
		p.MoveTo(10, 0)
		p.RelativeArcToPoint(Offset{-10, 10}, ArcToPointOpts{Radius: Radius{10, 10}})
		if end := p.points[len(p.points)-1]; end != (Offset{0, 10}) {
			t.Errorf("end = %v", end)
		}
		q := NewPath()
		q.RelativeArcToPoint(Offset{5, 5}, ArcToPointOpts{Radius: Radius{5, 5}})
		if q.points[0] != (Offset{}) || q.points[len(q.points)-1] != (Offset{5, 5}) {
			t.Errorf("relative from an empty path: %v", q.points)
		}
	})
	t.Run("a tiny arc is a line", func(t *testing.T) {
		p := NewPath()
		p.MoveTo(0, 0)
		p.ArcToPoint(Offset{1e-4, 0}, ArcToPointOpts{Radius: Radius{1e9, 1e9}})
		if verbString(p) != "ML" {
			t.Errorf("%s", verbString(p))
		}
	})
}

func TestPathAddRRect(t *testing.T) {
	t.Run("a standard one starts on the left edge and goes clockwise", func(t *testing.T) {
		p := NewPath()
		p.AddRRect(NewRRectFromRectAndRadius(Rect{0, 0, 100, 50}, Radius{10, 10}))
		if verbString(p) != "MLOLOLOLOZ" {
			t.Fatalf("verbs %s", verbString(p))
		}
		pointsNear(t, "rrect", p.points, []Offset{
			{0, 40},         // start, above the bottom left corner
			{0, 10},         // up the left edge
			{0, 0}, {10, 0}, // top left corner
			{90, 0}, // along the top
			{100, 0}, {100, 10},
			{100, 40},
			{100, 50}, {90, 50},
			{10, 50},
			{0, 50}, {0, 40},
		}, 1e-9)
	})
	t.Run("different radii", func(t *testing.T) {
		p := NewPath()
		p.AddRRect(NewRRectFromRectAndCorners(Rect{0, 0, 100, 50}, Corners{TopLeft: Radius{10, 20}, TopRight: Radius{30, 5}, BottomRight: Radius{15, 15}, BottomLeft: Radius{5, 25}}))
		ends := pathEndPoints(p)
		want := []Offset{{0, 25}, {0, 20}, {10, 0}, {70, 0}, {100, 5}, {100, 35}, {85, 50}, {5, 50}, {0, 25}}
		pointsNear(t, "ends", ends, want, 1e-9)
	})
	t.Run("zero radii are a rectangle starting at the bottom left", func(t *testing.T) {
		p := NewPath()
		p.AddRRect(NewRRectFromRectAndRadius(Rect{0, 0, 100, 50}, Radius{}))
		if verbString(p) != "MLLLZ" {
			t.Fatalf("verbs %s", verbString(p))
		}
		pointsNear(t, "rect", p.points, []Offset{{0, 50}, {0, 0}, {100, 0}, {100, 50}}, 0)
	})
	t.Run("an axis with a zero radius is a rectangle", func(t *testing.T) {
		p := NewPath()
		p.AddRRect(NewRRectFromRectXY(Rect{0, 0, 100, 50}, 10, 0))
		if verbString(p) != "MLLLZ" {
			t.Errorf("verbs %s", verbString(p))
		}
	})
	t.Run("radii of half the size are an oval", func(t *testing.T) {
		p := NewPath()
		p.AddRRect(NewRRectFromRectXY(Rect{0, 0, 100, 50}, 50, 25))
		if verbString(p) != "MOOOOZ" || p.points[0] != (Offset{0, 25}) {
			t.Errorf("verbs %s start %v", verbString(p), p.points[0])
		}
	})
	t.Run("radii that do not fit are scaled", func(t *testing.T) {
		p := NewPath()
		p.AddRRect(NewRRectFromRectXY(Rect{0, 0, 100, 40}, 100, 40))
		// The scaled radii are 50 by 20: an oval.
		if verbString(p) != "MOOOOZ" {
			t.Errorf("verbs %s", verbString(p))
		}
	})
	t.Run("an empty one is a degenerate rectangle", func(t *testing.T) {
		p := NewPath()
		p.AddRRect(NewRRectFromRectXY(Rect{5, 5, 5, 5}, 1, 1))
		if verbString(p) != "MLLLZ" {
			t.Errorf("verbs %s", verbString(p))
		}
	})
	t.Run("the path contains what the rrect contains", func(t *testing.T) {
		rr := NewRRectFromRectAndCorners(Rect{0, 0, 100, 50}, Corners{TopLeft: Radius{10, 20}, TopRight: Radius{30, 5}, BottomRight: Radius{15, 15}, BottomLeft: Radius{5, 25}})
		p := NewPath()
		p.AddRRect(rr)
		for x := 0.5; x < 100; x += 3.7 {
			for y := 0.5; y < 50; y += 2.9 {
				pt := Offset{x, y}
				if got, want := p.Contains(pt), rr.Contains(pt); got != want {
					t.Errorf("Contains(%v) = %v, RRect.Contains = %v", pt, got, want)
				}
			}
		}
	})
}

func TestPathAddRSuperellipse(t *testing.T) {
	rs := NewRSuperellipseFromRectAndRadius(Rect{0, 0, 100, 60}, Radius{20, 20})
	p := NewPath()
	p.AddRSuperellipse(rs)
	if p.verbs[0] != PathVerbMove || p.verbs[len(p.verbs)-1] != PathVerbClose {
		t.Fatalf("verbs %s", verbString(p))
	}
	rectNear(t, p.GetTightBounds(), Rect{0, 0, 100, 60}, 1e-3)
	// The outline agrees with RSuperellipse.Contains away from the boundary.
	disagree := 0
	for x := 0.25; x < 100; x += 1.9 {
		for y := 0.25; y < 60; y += 1.7 {
			pt := Offset{x, y}
			if p.Contains(pt) != rs.Contains(pt) {
				disagree++
			}
		}
	}
	if disagree > 6 {
		t.Errorf("%d sample points disagree between the path and Contains", disagree)
	}
	// Square corners are plain rectangles.
	sq := NewPath()
	sq.AddRSuperellipse(NewRSuperellipseFromRectXY(Rect{0, 0, 10, 10}, 0, 0))
	if verbString(sq) != "MLLLZ" {
		t.Errorf("verbs %s", verbString(sq))
	}
	// A pill has the same extent on both sides.
	pill := NewPath()
	pill.AddRSuperellipse(NewRSuperellipseFromRectXY(Rect{0, 0, 100, 20}, 10, 10))
	rectNear(t, pill.GetTightBounds(), Rect{0, 0, 100, 20}, 1e-3)
}
