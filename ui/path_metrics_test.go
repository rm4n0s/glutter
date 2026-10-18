package ui

import (
	"math"
	"testing"
)

func TestPathMetricsLengthsAndFlags(t *testing.T) {
	p := NewPath()
	p.MoveTo(0, 0)
	p.LineTo(30, 40) // 50
	p.MoveTo(100, 100)
	p.LineTo(110, 100)
	p.LineTo(110, 110)
	p.Close() // 10 + 10 + sqrt(200)
	p.MoveTo(200, 200)
	p.MoveTo(300, 300) // lone moves have no length
	p.MoveTo(400, 400)
	p.LineTo(400, 400) // a line of no length is dropped
	m := p.ComputeMetrics()
	if len(m) != 2 {
		t.Fatalf("%d metrics, want 2: %v", len(m), m)
	}
	if !near(m[0].Length, 50, 1e-12) || m[0].IsClosed || m[0].ContourIndex != 0 {
		t.Errorf("first = %v", m[0])
	}
	if !near(m[1].Length, 20+math.Sqrt(200), 1e-12) || !m[1].IsClosed || m[1].ContourIndex != 1 {
		t.Errorf("second = %v", m[1])
	}
	if got := m[1].String(); got != "PathMetric(length: 34.14213562373095, isClosed: true, contourIndex: 1)" {
		t.Errorf("String = %q", got)
	}
	// ForceClosed closes the first contour too.
	f := p.ComputeMetrics(ComputePathMetricsOpts{ForceClosed: true})
	if !f[0].IsClosed || !near(f[0].Length, 100, 1e-12) || !near(f[1].Length, m[1].Length, 1e-12) {
		t.Errorf("forced: %v %v", f[0], f[1])
	}
	// An empty path has none.
	if len(NewPath().ComputeMetrics()) != 0 {
		t.Error("an empty path has no contours")
	}
	// A path that is only moves too.
	only := NewPath()
	only.MoveTo(1, 1)
	if len(only.ComputeMetrics()) != 0 {
		t.Error("a lone move has no length")
	}
	// A closed contour that is already closed by a line to its start has no extra
	// closing segment.
	sq := NewPath()
	sq.AddRect(Rect{0, 0, 10, 10})
	if got := sq.ComputeMetrics()[0].Length; !near(got, 40, 1e-12) {
		t.Errorf("a square: %v", got)
	}
	sq2 := NewPath()
	sq2.MoveTo(0, 0)
	sq2.LineTo(10, 0)
	sq2.LineTo(10, 10)
	sq2.LineTo(0, 10)
	sq2.LineTo(0, 0)
	sq2.Close()
	if got := sq2.ComputeMetrics()[0].Length; !near(got, 40, 1e-12) {
		t.Errorf("a square closed by a line: %v", got)
	}
}

// curveLength integrates the speed of a curve with Simpson's rule.
func curveLength(speed func(t float64) float64) float64 {
	const n = 20000
	h := 1.0 / n
	sum := speed(0) + speed(1)
	for i := 1; i < n; i++ {
		w := 2.0
		if i%2 == 1 {
			w = 4
		}
		sum += w * speed(float64(i)*h)
	}
	return sum * h / 3
}

func TestPathMetricsCurveLengths(t *testing.T) {
	r := 37.0
	circle := NewPath()
	circle.AddOval(Rect{-r, -r, r, r})
	if got, want := circle.ComputeMetrics()[0].Length, 2*math.Pi*r; !near(got, want, want*2e-6) {
		t.Errorf("circle length %v, want %v", got, want)
	}
	arc := NewPath()
	arc.AddArc(Rect{-r, -r, r, r}, 0.3, 2.0)
	if got, want := arc.ComputeMetrics()[0].Length, r*2.0; !near(got, want, want*2e-6) {
		t.Errorf("arc length %v, want %v", got, want)
	}
	q := NewPath()
	q.MoveTo(0, 0)
	q.QuadraticBezierTo(30, 80, 100, 10)
	qWant := curveLength(func(t float64) float64 {
		d := quadTangentAt(Offset{0, 0}, Offset{30, 80}, Offset{100, 10}, t).Mul(2)
		return math.Hypot(d.Dx, d.Dy)
	})
	if got := q.ComputeMetrics()[0].Length; !near(got, qWant, qWant*2e-6) {
		t.Errorf("quad length %v, want %v", got, qWant)
	}
	c := NewPath()
	c.MoveTo(0, 0)
	c.CubicTo(0, 100, 100, 100, 100, 0)
	cWant := curveLength(func(t float64) float64 {
		d := cubicTangentAt(Offset{0, 0}, Offset{0, 100}, Offset{100, 100}, Offset{100, 0}, t)
		return math.Hypot(d.Dx, d.Dy)
	})
	if got := c.ComputeMetrics()[0].Length; !near(got, cWant, cWant*2e-6) {
		t.Errorf("cubic length %v, want %v", got, cWant)
	}
	// A conic of weight 0.5 (an ellipse arc? no: a flatter curve), against its own derivative.
	k := NewPath()
	k.MoveTo(0, 0)
	k.ConicTo(50, 80, 100, 0, 2)
	kWant := curveLength(func(t float64) float64 {
		// numerical derivative of conicAt
		const h = 1e-6
		a, b := conicAt(Offset{0, 0}, Offset{50, 80}, Offset{100, 0}, 2, math.Max(0, t-h)), conicAt(Offset{0, 0}, Offset{50, 80}, Offset{100, 0}, 2, math.Min(1, t+h))
		return math.Hypot(b.Dx-a.Dx, b.Dy-a.Dy) / (math.Min(1, t+h) - math.Max(0, t-h))
	})
	if got := k.ComputeMetrics()[0].Length; !near(got, kWant, kWant*1e-5) {
		t.Errorf("conic length %v, want %v", got, kWant)
	}
}

func TestPathMetricsTangents(t *testing.T) {
	rect := NewPath()
	rect.AddRect(Rect{0, 0, 10, 20})
	m := rect.ComputeMetrics()[0] // 60 long, clockwise from the top left
	tests := []struct {
		d         float64
		pos, vec  Offset
		available bool
	}{
		{0, Offset{0, 0}, Offset{1, 0}, true},
		{5, Offset{5, 0}, Offset{1, 0}, true},
		{10, Offset{10, 0}, Offset{1, 0}, true}, // the end of a segment keeps the direction it arrives with
		{15, Offset{10, 5}, Offset{0, 1}, true},
		{30, Offset{10, 20}, Offset{0, 1}, true},
		{35, Offset{5, 20}, Offset{-1, 0}, true},
		{50, Offset{0, 10}, Offset{0, -1}, true},
		{60, Offset{0, 0}, Offset{0, -1}, true},
		{-5, Offset{0, 0}, Offset{1, 0}, true},    // clamped
		{1000, Offset{0, 0}, Offset{0, -1}, true}, // clamped
		{math.NaN(), Offset{}, Offset{}, false},
	}
	for _, tt := range tests {
		got, ok := m.GetTangentForOffset(tt.d)
		if ok != tt.available {
			t.Errorf("distance %v: ok = %v", tt.d, ok)
			continue
		}
		if !ok {
			continue
		}
		if !near(got.Position.Dx, tt.pos.Dx, 1e-9) || !near(got.Position.Dy, tt.pos.Dy, 1e-9) {
			// At a corner either direction is fine; position must match.
			t.Errorf("distance %v: position %v, want %v", tt.d, got.Position, tt.pos)
		}
		if tt.d != 10 && tt.d != 30 && tt.d != 60 && tt.d != 0 {
			if !near(got.Vector.Dx, tt.vec.Dx, 1e-9) || !near(got.Vector.Dy, tt.vec.Dy, 1e-9) {
				t.Errorf("distance %v: vector %v, want %v", tt.d, got.Vector, tt.vec)
			}
		}
		if l := math.Hypot(got.Vector.Dx, got.Vector.Dy); !near(l, 1, 1e-12) {
			t.Errorf("distance %v: vector %v is not a unit vector", tt.d, got.Vector)
		}
	}

	// On a circle the position is at the right radius and the tangent is
	// perpendicular to the radius and clockwise.
	r := 40.0
	circle := NewPath()
	circle.AddOval(Rect{-r, -r, r, r})
	cm := circle.ComputeMetrics()[0]
	for i := 0; i <= 50; i++ {
		d := cm.Length * float64(i) / 50
		tan, ok := cm.GetTangentForOffset(d)
		if !ok {
			t.Fatal("no tangent")
		}
		if !near(math.Hypot(tan.Position.Dx, tan.Position.Dy), r, 1e-4) {
			t.Errorf("distance %v: position %v is off the circle", d, tan.Position)
		}
		if dot := tan.Position.Dx*tan.Vector.Dx + tan.Position.Dy*tan.Vector.Dy; !near(dot, 0, 1e-4*r) {
			t.Errorf("distance %v: the tangent %v is not perpendicular to %v (dot %v)", d, tan.Vector, tan.Position, dot)
		}
		// Clockwise on screen: the cross product of position and direction is positive.
		if cross := tan.Position.Dx*tan.Vector.Dy - tan.Position.Dy*tan.Vector.Dx; cross <= 0 {
			t.Errorf("distance %v: the circle is not clockwise (cross %v)", d, cross)
		}
		// The arc length from the start to here matches the angle.
		angle := math.Atan2(tan.Position.Dy, tan.Position.Dx)
		if angle < 0 {
			angle += 2 * math.Pi
		}
		if want := d / r; !near(angle, math.Mod(want, 2*math.Pi), 1e-3) && !near(angle, 2*math.Pi, 1e-3) && want < 2*math.Pi-1e-3 {
			t.Errorf("distance %v: at angle %v, want %v", d, angle, want)
		}
	}
	// The vector of a degenerate cubic at its end looks past the coincident control point.
	deg := NewPath()
	deg.MoveTo(0, 0)
	deg.CubicTo(0, 0, 10, 0, 10, 0)
	dm := deg.ComputeMetrics()[0]
	if tan, ok := dm.GetTangentForOffset(0); !ok || !near(tan.Vector.Dx, 1, 1e-9) {
		t.Errorf("tangent at the start of a degenerate cubic: %v %v", tan, ok)
	}
}

func TestTangent(t *testing.T) {
	tan := NewTangent(Offset{1, 2}, Offset{0, 1})
	if tan.Position != (Offset{1, 2}) || tan.Vector != (Offset{0, 1}) {
		t.Errorf("%+v", tan)
	}
	// The angle is counter-clockwise, to match ArcTo's sweep: the vector (0, 1)
	// points down the screen, which is -90 degrees.
	if !near(tan.Angle(), -math.Pi/2, 1e-12) {
		t.Errorf("Angle = %v", tan.Angle())
	}
	f := NewTangentFromAngle(Offset{}, math.Pi/3)
	if !near(f.Vector.Dx, 0.5, 1e-12) || !near(f.Vector.Dy, math.Sqrt(3)/2, 1e-12) {
		t.Errorf("FromAngle vector = %v", f.Vector)
	}
}

func TestPathMetricsExtractPath(t *testing.T) {
	line := NewPath()
	line.MoveTo(0, 0)
	line.LineTo(100, 0)
	m := line.ComputeMetrics()[0]

	t.Run("part of a line", func(t *testing.T) {
		e := m.ExtractPath(20, 70)
		if verbString(e) != "ML" {
			t.Fatalf("verbs %s", verbString(e))
		}
		pointsNear(t, "extract", e.points, []Offset{{20, 0}, {70, 0}}, 1e-9)
	})
	t.Run("the whole line, and more than it", func(t *testing.T) {
		pointsNear(t, "whole", m.ExtractPath(0, 100).points, []Offset{{0, 0}, {100, 0}}, 0)
		pointsNear(t, "clamped", m.ExtractPath(-50, 500).points, []Offset{{0, 0}, {100, 0}}, 0)
	})
	t.Run("without a move", func(t *testing.T) {
		e := m.ExtractPath(20, 70, ExtractPathOpts{StartWithMoveTo: Ptr(false)})
		// A line continues from the current point, which is the origin.
		if verbString(e) != "ML" || e.points[1] != (Offset{70, 0}) {
			t.Errorf("verbs %s points %v", verbString(e), e.points)
		}
		dst := NewPath()
		dst.MoveTo(5, 5)
		dst.ExtendWithPath(m.ExtractPath(20, 70, ExtractPathOpts{StartWithMoveTo: Ptr(true)}), Offset{})
		if verbString(dst) != "MLL" {
			t.Errorf("extended: %s", verbString(dst))
		}
	})
	t.Run("start after end is empty", func(t *testing.T) {
		if e := m.ExtractPath(70, 20); !e.IsEmpty() {
			t.Errorf("verbs %s", verbString(e))
		}
		if e := m.ExtractPath(math.NaN(), 20); !e.IsEmpty() {
			t.Errorf("NaN verbs %s", verbString(e))
		}
	})
	t.Run("a zero length part is a point with a line of no length", func(t *testing.T) {
		e := m.ExtractPath(40, 40)
		if verbString(e) != "ML" || e.points[0] != (Offset{40, 0}) || e.points[1] != (Offset{40, 0}) {
			t.Errorf("verbs %s points %v", verbString(e), e.points)
		}
	})
	t.Run("across the corners of a polygon", func(t *testing.T) {
		rect := NewPath()
		rect.AddRect(Rect{0, 0, 10, 20})
		rm := rect.ComputeMetrics()[0]
		e := rm.ExtractPath(5, 45)
		if verbString(e) != "MLLLL" {
			t.Fatalf("verbs %s", verbString(e))
		}
		pointsNear(t, "polygon", e.points, []Offset{{5, 0}, {10, 0}, {10, 20}, {0, 20}, {0, 15}}, 1e-9)
		if got := e.ComputeMetrics()[0].Length; !near(got, 40, 1e-9) {
			t.Errorf("length %v, want 40", got)
		}
	})
	t.Run("curves", func(t *testing.T) {
		r := 50.0
		circle := NewPath()
		circle.AddOval(Rect{-r, -r, r, r})
		cm := circle.ComputeMetrics()[0]
		// The second quarter: from the bottom to the left (the oval starts at the
		// right and goes clockwise on screen).
		q := cm.Length / 4
		e := cm.ExtractPath(q, 2*q)
		if got := e.ComputeMetrics()[0].Length; !near(got, q, q*2e-6) {
			t.Errorf("quarter length %v, want %v", got, q)
		}
		em := e.ComputeMetrics()[0]
		start, _ := em.GetTangentForOffset(0)
		end, _ := em.GetTangentForOffset(em.Length)
		pointsNear(t, "quarter ends", []Offset{start.Position, end.Position}, []Offset{{0, 50}, {-50, 0}}, 1e-6)
		// Every extracted point is on the circle.
		for _, p := range e.Flatten(nil, 0.001)[0].Points {
			if !near(math.Hypot(p.Dx, p.Dy), r, 0.01) {
				t.Fatalf("%v is off the circle", p)
			}
		}
		// A piece starting inside a conic and ending inside another.
		e2 := cm.ExtractPath(q*0.5, q*2.5)
		if got := e2.ComputeMetrics()[0].Length; !near(got, 2*q, 2*q*2e-6) {
			t.Errorf("two quarters length %v, want %v", got, 2*q)
		}
		// Quads and cubics too.
		cu := NewPath()
		cu.MoveTo(0, 0)
		cu.CubicTo(0, 100, 100, 100, 100, 0)
		cum := cu.ComputeMetrics()[0]
		part := cum.ExtractPath(cum.Length*0.25, cum.Length*0.75)
		if got := part.ComputeMetrics()[0].Length; !near(got, cum.Length/2, cum.Length*2e-6) {
			t.Errorf("cubic half length %v, want %v", got, cum.Length/2)
		}
		qu := NewPath()
		qu.MoveTo(0, 0)
		qu.QuadraticBezierTo(50, 100, 100, 0)
		qm := qu.ComputeMetrics()[0]
		qpart := qm.ExtractPath(qm.Length*0.1, qm.Length*0.6)
		if got := qpart.ComputeMetrics()[0].Length; !near(got, qm.Length*0.5, qm.Length*2e-6) {
			t.Errorf("quad part length %v, want %v", got, qm.Length*0.5)
		}
	})
	t.Run("a closed contour extracts the closing line too", func(t *testing.T) {
		tri := NewPath()
		tri.MoveTo(0, 0)
		tri.LineTo(10, 0)
		tri.LineTo(0, 10)
		tri.Close()
		tm := tri.ComputeMetrics()[0]
		e := tm.ExtractPath(0, tm.Length)
		if verbString(e) != "MLLL" {
			t.Errorf("verbs %s", verbString(e))
		}
		pointsNear(t, "triangle", e.points, []Offset{{0, 0}, {10, 0}, {0, 10}, {0, 0}}, 1e-9)
	})
}
