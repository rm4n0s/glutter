package ui

import (
	"math"
	"math/rand"
	"testing"
)

func circlePath(cx, cy, r float64) *Path {
	p := NewPath()
	p.AddOval(NewRectFromLTRB(cx-r, cy-r, cx+r, cy+r))
	return p
}

func rectPath(l, t, r, b float64) *Path {
	p := NewPath()
	p.AddRect(NewRectFromLTRB(l, t, r, b))
	return p
}

// starPath is a star with n points; step 2 on 5 points is the pentagram, whose lines
// cross themselves.
func starPath(cx, cy, outer, inner float64, n int) *Path {
	var pts []Offset
	for i := 0; i < 2*n; i++ {
		r := outer
		if i%2 == 1 {
			r = inner
		}
		a := -math.Pi/2 + float64(i)*math.Pi/float64(n)
		pts = append(pts, Offset{cx + r*math.Cos(a), cy + r*math.Sin(a)})
	}
	p := NewPath()
	p.AddPolygon(pts, true)
	return p
}

func pentagram(cx, cy, r float64, fill PathFillType) *Path {
	p := NewPath()
	var pts []Offset
	for i := 0; i < 5; i++ {
		a := -math.Pi/2 + float64(i*2)*2*math.Pi/5
		pts = append(pts, Offset{cx + r*math.Cos(a), cy + r*math.Sin(a)})
	}
	p.AddPolygon(pts, true)
	p.SetFillType(fill)
	return p
}

func ringPath(cx, cy, outer, inner float64) *Path {
	p := circlePath(cx, cy, outer)
	p.AddPath(circlePath(cx, cy, inner), Offset{})
	p.SetFillType(PathFillTypeEvenOdd)
	return p
}

// boundaryDistance is how far the point is from the lines of the paths.
func boundaryDistance(pt Offset, paths ...*Path) float64 {
	best := math.Inf(1)
	for _, p := range paths {
		for _, c := range p.Flatten(nil, 0.01) {
			pts := c.Points
			for i := range pts {
				a, b := pts[i], pts[(i+1)%len(pts)]
				if i == len(pts)-1 && !c.Closed {
					break
				}
				best = math.Min(best, segmentDistance(pt, a, b))
			}
		}
	}
	return best
}

func segmentDistance(p, a, b Offset) float64 {
	e := b.Sub(a)
	l2 := e.Dx*e.Dx + e.Dy*e.Dy
	t := 0.0
	if l2 > 0 {
		t = math.Max(0, math.Min(1, ((p.Dx-a.Dx)*e.Dx+(p.Dy-a.Dy)*e.Dy)/l2))
	}
	return p.Sub(a.Add(e.Mul(t))).Distance()
}

// contourAreas are the signed areas of the contours of a path, by the polyline.
func contourAreas(p *Path) []float64 { return contourAreasTol(p, 0.01) }

func contourAreasTol(p *Path, tol float64) []float64 {
	var out []float64
	for _, c := range p.Flatten(nil, tol) {
		a := 0.0
		for i := range c.Points {
			q, r := c.Points[i], c.Points[(i+1)%len(c.Points)]
			a += q.Dx*r.Dy - r.Dx*q.Dy
		}
		out = append(out, a/2)
	}
	return out
}

// totalArea is the area of the region that a path from PathCombine fills: the signed
// areas of its contours added up (the holes turn the other way).
func totalArea(p *Path) float64 { return totalAreaTol(p, 0.01) }

func totalAreaTol(p *Path, tol float64) float64 {
	sum := 0.0
	for _, a := range contourAreasTol(p, tol) {
		sum += a
	}
	return math.Abs(sum)
}

var allOps = []PathOperation{PathOperationDifference, PathOperationIntersect, PathOperationUnion, PathOperationXor, PathOperationReverseDifference}

// checkAgainstPoints compares the result with the operation on the points: at every point
// that is not near a border, the result has what the operation says of the two paths.
func checkAgainstPoints(t *testing.T, op PathOperation, a, b *Path, bounds Rect, step float64) {
	t.Helper()
	res := PathCombine(op, a, b)
	checked, wrong := 0, 0
	for y := bounds.Top; y < bounds.Bottom; y += step {
		for x := bounds.Left; x < bounds.Right; x += step {
			pt := Offset{x + 0.123, y + 0.071}
			if boundaryDistance(pt, a, b, res) < 0.4 {
				continue
			}
			checked++
			want := op.inside(a.Contains(pt), b.Contains(pt))
			if got := res.Contains(pt); got != want {
				wrong++
				if wrong <= 3 {
					t.Errorf("%v at (%.2f, %.2f): result says %v, want %v", op, pt.Dx, pt.Dy, got, want)
				}
			}
		}
	}
	if checked < 50 {
		t.Errorf("only %d points were checked", checked)
	}
}

func TestPathCombineAgainstPoints(t *testing.T) {
	shapes := []struct {
		name string
		a, b *Path
	}{
		{"overlapping circles", circlePath(60, 60, 40), circlePath(100, 70, 40)},
		{"circle and star", circlePath(80, 80, 45), starPath(95, 85, 55, 22, 5)},
		{"two stars", starPath(70, 80, 60, 25, 5), starPath(110, 90, 60, 25, 6)},
		{"rectangles", rectPath(10, 10, 100, 90), rectPath(60, 50, 150, 130)},
		{"rectangles sharing an edge", rectPath(10, 10, 80, 80), rectPath(80, 10, 150, 80)},
		{"rectangles sharing part of an edge", rectPath(10, 10, 80, 80), rectPath(80, 40, 150, 120)},
		{"nested circles", circlePath(80, 80, 60), circlePath(80, 80, 25)},
		{"a circle in a rectangle", rectPath(20, 20, 140, 140), circlePath(80, 80, 50)},
		{"disjoint", rectPath(10, 10, 60, 60), circlePath(130, 130, 30)},
		{"identical", circlePath(80, 80, 50), circlePath(80, 80, 50)},
		{"identical rectangles", rectPath(20, 20, 120, 100), rectPath(20, 20, 120, 100)},
		{"a ring and a rectangle", ringPath(80, 80, 60, 30), rectPath(10, 70, 150, 90)},
		{"a pentagram, nonzero and even-odd", pentagram(80, 80, 70, PathFillTypeNonZero), pentagram(100, 90, 70, PathFillTypeEvenOdd)},
		{"touching at a corner", rectPath(10, 10, 60, 60), rectPath(60, 60, 110, 110)},
		{"tangent circles", circlePath(50, 80, 40), circlePath(130, 80, 40)},
	}
	for _, tt := range shapes {
		for _, op := range allOps {
			t.Run(tt.name+"/"+op.String(), func(t *testing.T) {
				checkAgainstPoints(t, op, tt.a, tt.b, NewRectFromLTRB(0, 0, 180, 180), 3.7)
			})
		}
	}
}

func TestPathCombineShapes(t *testing.T) {
	a, b := rectPath(0, 0, 100, 100), rectPath(50, 50, 150, 150)
	tests := []struct {
		op       PathOperation
		area     float64
		contours int
	}{
		{PathOperationUnion, 100*100 + 100*100 - 50*50, 1},
		{PathOperationIntersect, 50 * 50, 1},
		{PathOperationDifference, 100*100 - 50*50, 1},
		{PathOperationReverseDifference, 100*100 - 50*50, 1},
		{PathOperationXor, 2 * (100*100 - 50*50), 2},
	}
	for _, tt := range tests {
		t.Run(tt.op.String(), func(t *testing.T) {
			res := PathCombine(tt.op, a, b)
			if got := totalArea(res); math.Abs(got-tt.area) > 1 {
				t.Errorf("area = %v, want %v", got, tt.area)
			}
			if got := len(contourAreas(res)); got != tt.contours {
				t.Errorf("%d contours, want %d", got, tt.contours)
			}
			if res.FillType() != PathFillTypeNonZero {
				t.Errorf("fill type = %v", res.FillType())
			}
		})
	}
}

func TestPathCombineHolesTurnTheOtherWay(t *testing.T) {
	res := PathCombine(PathOperationDifference, rectPath(0, 0, 100, 100), rectPath(30, 30, 70, 70))
	areas := contourAreas(res)
	if len(areas) != 2 || areas[0]*areas[1] >= 0 {
		t.Fatalf("areas = %v, want two with opposite signs", areas)
	}
	if got := math.Abs(areas[0] + areas[1]); math.Abs(got-(100*100-40*40)) > 1 {
		t.Errorf("area = %v", got)
	}
	// The hole is a hole for the nonzero fill type as much as for the even-odd one.
	for _, fill := range []PathFillType{PathFillTypeNonZero, PathFillTypeEvenOdd} {
		res.SetFillType(fill)
		if res.Contains(Offset{50, 50}) || !res.Contains(Offset{10, 10}) {
			t.Errorf("fill type %v: the hole is not a hole", fill)
		}
	}
}

func TestPathCombineEmptyResults(t *testing.T) {
	a, b := rectPath(0, 0, 10, 10), rectPath(20, 20, 30, 30)
	tests := []struct {
		name string
		op   PathOperation
		a, b *Path
		want bool // empty
	}{
		{"disjoint intersect", PathOperationIntersect, a, b, true},
		{"identical difference", PathOperationDifference, a, rectPath(0, 0, 10, 10), true},
		{"identical xor", PathOperationXor, a, rectPath(0, 0, 10, 10), true},
		{"difference of nothing", PathOperationDifference, NewPath(), a, true},
		{"reverse difference of nothing", PathOperationReverseDifference, a, NewPath(), true},
		{"union of nothing", PathOperationUnion, NewPath(), NewPath(), true},
		{"nil paths", PathOperationUnion, nil, nil, true},
		{"a point", PathOperationUnion, pointPath(5, 5), pointPath(5, 5), true},
		{"a line is no area", PathOperationUnion, linePath(0, 0, 10, 10), linePath(0, 10, 10, 0), true},
		{"disjoint union", PathOperationUnion, a, b, false},
		{"difference with nothing", PathOperationDifference, a, nil, false},
		{"union with nothing", PathOperationUnion, nil, a, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PathCombine(tt.op, tt.a, tt.b).IsEmpty(); got != tt.want {
				t.Errorf("empty = %v, want %v", got, tt.want)
			}
		})
	}
	if got := totalArea(PathCombine(PathOperationUnion, a, b)); math.Abs(got-200) > 0.01 {
		t.Errorf("area of the union of two squares = %v", got)
	}
	if got := totalArea(PathCombine(PathOperationDifference, a, nil)); math.Abs(got-100) > 0.01 {
		t.Errorf("area of a square without nothing = %v", got)
	}
}

func pointPath(x, y float64) *Path {
	p := NewPath()
	p.MoveTo(x, y)
	return p
}

func linePath(x0, y0, x1, y1 float64) *Path {
	p := NewPath()
	p.MoveTo(x0, y0)
	p.LineTo(x1, y1)
	return p
}

func TestPathCombineBadInput(t *testing.T) {
	good := rectPath(0, 0, 10, 10)
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		p := NewPath()
		p.MoveTo(0, 0)
		p.LineTo(bad, 5)
		p.LineTo(5, 10)
		for _, op := range allOps {
			if got := PathCombine(op, p, good); !got.IsEmpty() {
				t.Errorf("%v with %v: %d verbs, want none", op, bad, len(got.verbs))
			}
		}
	}
	// The zero operation is a difference.
	if got := totalArea(PathCombine(PathOperationUnspecified, good, rectPath(5, 0, 10, 10))); math.Abs(got-50) > 0.01 {
		t.Errorf("the zero operation gave an area of %v", got)
	}
}

func TestPathCombineKeepsCurves(t *testing.T) {
	countVerbs := func(p *Path) map[PathVerb]int {
		m := map[PathVerb]int{}
		for _, v := range p.verbs {
			m[v]++
		}
		return m
	}
	// A circle inside a rectangle: the intersection is the circle itself, with its four conics.
	res := PathCombine(PathOperationIntersect, circlePath(80, 80, 40), rectPath(0, 0, 200, 200))
	if m := countVerbs(res); m[PathVerbConic] != 4 || m[PathVerbLine] != 0 || m[PathVerbCubic] != 0 {
		t.Errorf("the circle was not kept: %v", m)
	}
	// Two circles that cross: the border is arcs, a few conics each, and no more lines.
	res = PathCombine(PathOperationUnion, circlePath(60, 60, 40), circlePath(100, 70, 40))
	m := countVerbs(res)
	if m[PathVerbConic] < 4 || m[PathVerbConic] > 12 || m[PathVerbLine] > 2 {
		t.Errorf("a union of circles has %v", m)
	}
	// The conics are on the circles: the points of the result are at the radius of one of them.
	for _, c := range res.Flatten(nil, 0.01) {
		for _, p := range c.Points {
			d1 := p.Sub(Offset{60, 60}).Distance()
			d2 := p.Sub(Offset{100, 70}).Distance()
			if math.Abs(d1-40) > 0.2 && math.Abs(d2-40) > 0.2 {
				t.Fatalf("the point %v of the union is on neither circle (%v, %v)", p, d1, d2)
			}
		}
	}
	// A rectangle cut by a circle: lines and arcs.
	res = PathCombine(PathOperationDifference, rectPath(0, 0, 100, 100), circlePath(100, 50, 30))
	m = countVerbs(res)
	if m[PathVerbConic] == 0 || m[PathVerbLine] < 3 {
		t.Errorf("a rectangle without a bite is %v", m)
	}
	// A quadratic and a cubic are kept as such.
	q := NewPath()
	q.MoveTo(0, 0)
	q.QuadraticBezierTo(50, 100, 100, 0)
	q.Close()
	res = PathCombine(PathOperationIntersect, q, rectPath(-10, -10, 110, 60))
	if m := countVerbs(res); m[PathVerbQuad] == 0 {
		t.Errorf("the quad was lost: %v", m)
	}
	cu := NewPath()
	cu.MoveTo(0, 0)
	cu.CubicTo(0, 100, 100, 100, 100, 0)
	cu.Close()
	res = PathCombine(PathOperationIntersect, cu, rectPath(-10, -10, 110, 60))
	if m := countVerbs(res); m[PathVerbCubic] == 0 {
		t.Errorf("the cubic was lost: %v", m)
	}
}

func TestPathCombineIsDeterministic(t *testing.T) {
	a, b := starPath(70, 80, 60, 25, 7), circlePath(90, 90, 50)
	first := PathCombine(PathOperationXor, a, b)
	for i := 0; i < 5; i++ {
		again := PathCombine(PathOperationXor, a, b)
		if again.Fingerprint() != first.Fingerprint() {
			t.Fatal("the same paths gave different results")
		}
	}
}

func TestPathCombineDoesNotChangeItsInputs(t *testing.T) {
	a, b := circlePath(60, 60, 40), starPath(80, 80, 50, 20, 5)
	fa, fb := a.Fingerprint(), b.Fingerprint()
	PathCombine(PathOperationUnion, a, b)
	if a.Fingerprint() != fa || b.Fingerprint() != fb {
		t.Error("an input changed")
	}
}

func TestPathCombineAtOtherScales(t *testing.T) {
	// The same shapes far from the origin and very small or very large: the answer scales.
	for _, tt := range []struct{ scale, shift float64 }{{1e-4, 0}, {1e-3, 5}, {1, 1e5}, {1e3, 0}, {1e6, -1e7}} {
		a := scaledPath(circlePath(60, 60, 40), tt.scale, tt.shift)
		b := scaledPath(circlePath(100, 70, 40), tt.scale, tt.shift)
		union := PathCombine(PathOperationUnion, a, b)
		inter := PathCombine(PathOperationIntersect, a, b)
		want := PathCombine(PathOperationUnion, circlePath(60, 60, 40), circlePath(100, 70, 40))
		wantI := PathCombine(PathOperationIntersect, circlePath(60, 60, 40), circlePath(100, 70, 40))
		k := tt.scale * tt.scale
		tol := 0.01 * tt.scale
		if got, w := totalAreaTol(union, tol), totalArea(want)*k; math.Abs(got-w) > w*0.01 {
			t.Errorf("scale %v shift %v: union area %v, want about %v", tt.scale, tt.shift, got, w)
		}
		if got, w := totalAreaTol(inter, tol), totalArea(wantI)*k; math.Abs(got-w) > w*0.01 {
			t.Errorf("scale %v shift %v: intersection area %v, want about %v", tt.scale, tt.shift, got, w)
		}
	}
}

// scaledPath is the path with every point scaled and then moved.
func scaledPath(p *Path, k, shift float64) *Path {
	m := func(o Offset) (float64, float64) { return o.Dx*k + shift, o.Dy*k + shift }
	out := NewPath()
	p.ForEachSegment(func(s PathSegment) {
		switch s.Verb {
		case PathVerbMove:
			out.MoveTo(m(s.P0))
		case PathVerbLine:
			out.LineTo(m(s.P1))
		case PathVerbQuad:
			x1, y1 := m(s.P1)
			x2, y2 := m(s.P2)
			out.QuadraticBezierTo(x1, y1, x2, y2)
		case PathVerbConic:
			x1, y1 := m(s.P1)
			x2, y2 := m(s.P2)
			out.ConicTo(x1, y1, x2, y2, s.Weight)
		case PathVerbCubic:
			x1, y1 := m(s.P1)
			x2, y2 := m(s.P2)
			x3, y3 := m(s.P3)
			out.CubicTo(x1, y1, x2, y2, x3, y3)
		case PathVerbClose:
			out.Close()
		}
	})
	return out
}

func TestPathCombineRandomRectangles(t *testing.T) {
	// Rectangles with edges that meet, overlap and touch (the coordinates are on a coarse
	// grid): the areas obey inclusion and exclusion.
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 300; i++ {
		mk := func() *Path {
			x, y := float64(rng.Intn(6)), float64(rng.Intn(6))
			return rectPath(x*10, y*10, x*10+float64(1+rng.Intn(5))*10, y*10+float64(1+rng.Intn(5))*10)
		}
		a, b := mk(), mk()
		aa, ab := totalArea(a), totalArea(b)
		inter := totalArea(PathCombine(PathOperationIntersect, a, b))
		union := totalArea(PathCombine(PathOperationUnion, a, b))
		diff := totalArea(PathCombine(PathOperationDifference, a, b))
		rdiff := totalArea(PathCombine(PathOperationReverseDifference, a, b))
		xor := totalArea(PathCombine(PathOperationXor, a, b))
		if math.Abs(union-(aa+ab-inter)) > 0.1 || math.Abs(diff-(aa-inter)) > 0.1 || math.Abs(rdiff-(ab-inter)) > 0.1 ||
			math.Abs(xor-(aa+ab-2*inter)) > 0.1 {
			t.Fatalf("case %d (%v, %v): areas a %v b %v inter %v union %v diff %v rdiff %v xor %v",
				i, a.GetBounds(), b.GetBounds(), aa, ab, inter, union, diff, rdiff, xor)
		}
	}
}

// TestPathCombineRandomPolygons combines polygons that cross themselves and each other, with
// vertices on a coarse grid so that edges overlap, touch and pass through each other's
// vertices, and compares with the operation on points.
func TestPathCombineRandomPolygons(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	poly := func() *Path {
		n := 3 + rng.Intn(6)
		pts := make([]Offset, n)
		for i := range pts {
			pts[i] = Offset{float64(1+rng.Intn(8)) * 10, float64(1+rng.Intn(8)) * 10}
		}
		p := NewPath()
		p.AddPolygon(pts, true)
		if rng.Intn(2) == 0 {
			p.SetFillType(PathFillTypeEvenOdd)
		}
		return p
	}
	bad := 0
	for i := 0; i < 400; i++ {
		a, b := poly(), poly()
		op := allOps[i%len(allOps)]
		res := PathCombine(op, a, b)
		wrong := 0
		for y := 0.0; y < 100; y += 4.3 {
			for x := 0.0; x < 100; x += 4.3 {
				pt := Offset{x + 0.37, y + 0.19}
				if boundaryDistance(pt, a, b, res) < 0.5 {
					continue
				}
				if res.Contains(pt) != op.inside(a.Contains(pt), b.Contains(pt)) {
					wrong++
				}
			}
		}
		if wrong > 0 {
			bad++
			if bad <= 3 {
				t.Errorf("case %d (%v): %d points are wrong; a %v b %v", i, op, wrong, a.Segments(), b.Segments())
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d of 400 combinations were wrong", bad)
	}
}
