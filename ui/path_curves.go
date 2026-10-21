package ui

import "math"

// scalarNearlyZero is Skia's SK_ScalarNearlyZero, 1/4096.
const scalarNearlyZero = 1.0 / (1 << 12)

func nearlyEqual(a, b float64) bool { return math.Abs(a-b) <= scalarNearlyZero }

func nearlyEqualPoints(a, b Offset) bool { return nearlyEqual(a.Dx, b.Dx) && nearlyEqual(a.Dy, b.Dy) }

func lerpPt(a, b Offset, t float64) Offset {
	return Offset{a.Dx + (b.Dx-a.Dx)*t, a.Dy + (b.Dy-a.Dy)*t}
}

// validUnitDivide computes numer/denom and reports whether it is strictly
// between 0 and 1.
func validUnitDivide(numer, denom float64) (float64, bool) {
	if numer < 0 {
		numer, denom = -numer, -denom
	}
	if denom == 0 || numer == 0 || numer >= denom {
		return 0, false
	}
	r := numer / denom
	if math.IsNaN(r) || r == 0 {
		return 0, false
	}
	return r, true
}

// findUnitQuadRoots returns the roots of a*t^2 + b*t + c that are strictly
// inside (0, 1), in increasing order.
//
// Flutter: skia SkFindUnitQuadRoots
func findUnitQuadRoots(a, b, c float64) []float64 {
	var roots []float64
	if a == 0 {
		if r, ok := validUnitDivide(-c, b); ok {
			roots = append(roots, r)
		}
		return roots
	}
	dr := b*b - 4*a*c
	if dr < 0 {
		return nil
	}
	dr = math.Sqrt(dr)
	if math.IsInf(dr, 0) || math.IsNaN(dr) {
		return nil
	}
	var q float64
	if b < 0 {
		q = -(b - dr) / 2
	} else {
		q = -(b + dr) / 2
	}
	if r, ok := validUnitDivide(q, a); ok {
		roots = append(roots, r)
	}
	if r, ok := validUnitDivide(c, q); ok {
		roots = append(roots, r)
	}
	if len(roots) == 2 {
		if roots[0] > roots[1] {
			roots[0], roots[1] = roots[1], roots[0]
		} else if roots[0] == roots[1] {
			roots = roots[:1]
		}
	}
	return roots
}

// quadAt evaluates the quadratic Bézier curve at t.
func quadAt(p0, p1, p2 Offset, t float64) Offset {
	a := lerpPt(p0, p1, t)
	b := lerpPt(p1, p2, t)
	return lerpPt(a, b, t)
}

// quadTangentAt is the derivative of the quadratic at t, which is not a unit
// vector. At an end with a coincident control point it uses the other end.
func quadTangentAt(p0, p1, p2 Offset, t float64) Offset {
	if (t == 0 && p0 == p1) || (t == 1 && p1 == p2) {
		return p2.Sub(p0)
	}
	b := p1.Sub(p0)
	a := p2.Sub(p1).Sub(b)
	return a.Mul(t).Add(b)
}

// cubicAt evaluates the cubic Bézier curve at t.
func cubicAt(p0, p1, p2, p3 Offset, t float64) Offset {
	ab := lerpPt(p0, p1, t)
	bc := lerpPt(p1, p2, t)
	cd := lerpPt(p2, p3, t)
	return lerpPt(lerpPt(ab, bc, t), lerpPt(bc, cd, t), t)
}

// cubicTangentAt is the derivative of the cubic at t, which is not a unit
// vector. Where a control point is on an end it looks at the next one.
func cubicTangentAt(p0, p1, p2, p3 Offset, t float64) Offset {
	if (t == 0 && p0 == p1) || (t == 1 && p2 == p3) {
		var tangent Offset
		if t == 0 {
			tangent = p2.Sub(p0)
		} else {
			tangent = p3.Sub(p1)
		}
		if tangent.Dx == 0 && tangent.Dy == 0 {
			tangent = p3.Sub(p0)
		}
		return tangent
	}
	// 3 * ((p1-p0)(1-t)^2 + 2(p2-p1)t(1-t) + (p3-p2)t^2)
	a := p1.Sub(p0)
	b := p2.Sub(p1)
	c := p3.Sub(p2)
	u := 1 - t
	return a.Mul(u * u).Add(b.Mul(2 * u * t)).Add(c.Mul(t * t)).Mul(3)
}

// conicAt evaluates the conic at t.
func conicAt(p0, p1, p2 Offset, w, t float64) Offset {
	u := 1 - t
	n0, n1, n2 := u*u, 2*w*t*u, t*t
	d := n0 + n1 + n2
	return Offset{
		(p0.Dx*n0 + p1.Dx*n1 + p2.Dx*n2) / d,
		(p0.Dy*n0 + p1.Dy*n1 + p2.Dy*n2) / d,
	}
}

// conicTangentAt is the direction of the conic at t, not a unit vector. It is
// the numerator of the derivative, which is enough for the direction.
func conicTangentAt(p0, p1, p2 Offset, w, t float64) Offset {
	// A degenerate conic has no tangent at an end with a coincident control
	// point: look at the other end.
	if (t == 0 && p0 == p1) || (t == 1 && p1 == p2) {
		return p2.Sub(p0)
	}
	p20 := p2.Sub(p0)
	p10 := p1.Sub(p0)
	c := p10.Mul(w)
	a := p20.Mul(w).Sub(p20)
	b := p20.Sub(c).Sub(c)
	return a.Mul(t).Add(b).Mul(t).Add(c)
}

// chopQuadAt splits the quadratic at t into two.
func chopQuadAt(p [3]Offset, t float64) (a, b [3]Offset) {
	p01 := lerpPt(p[0], p[1], t)
	p12 := lerpPt(p[1], p[2], t)
	m := lerpPt(p01, p12, t)
	return [3]Offset{p[0], p01, m}, [3]Offset{m, p12, p[2]}
}

// chopCubicAt splits the cubic at t into two.
func chopCubicAt(p [4]Offset, t float64) (a, b [4]Offset) {
	ab := lerpPt(p[0], p[1], t)
	bc := lerpPt(p[1], p[2], t)
	cd := lerpPt(p[2], p[3], t)
	abc := lerpPt(ab, bc, t)
	bcd := lerpPt(bc, cd, t)
	m := lerpPt(abc, bcd, t)
	return [4]Offset{p[0], ab, abc, m}, [4]Offset{m, bcd, cd, p[3]}
}

// chopConicAt splits the conic at t into two, each in standard form (end
// weights of 1).
//
// Flutter: skia SkConic::chopAt
func chopConicAt(p [3]Offset, w, t float64) (a [3]Offset, wa float64, b [3]Offset, wb float64) {
	// Lift to homogeneous coordinates (x*w, y*w, w), split them like a quadratic,
	// and project back.
	type p3 struct{ x, y, z float64 }
	h := [3]p3{{p[0].Dx, p[0].Dy, 1}, {p[1].Dx * w, p[1].Dy * w, w}, {p[2].Dx, p[2].Dy, 1}}
	mix := func(u, v p3) p3 { return p3{u.x + (v.x-u.x)*t, u.y + (v.y-u.y)*t, u.z + (v.z-u.z)*t} }
	ab := mix(h[0], h[1])
	bc := mix(h[1], h[2])
	m := mix(ab, bc)
	proj := func(q p3) Offset { return Offset{q.x / q.z, q.y / q.z} }
	a = [3]Offset{p[0], proj(ab), proj(m)}
	b = [3]Offset{proj(m), proj(bc), p[2]}
	root := math.Sqrt(m.z)
	return a, ab.z / root, b, bc.z / root
}

// subQuad returns the part of the quadratic between t0 and t1.
func subQuad(p [3]Offset, t0, t1 float64) [3]Offset {
	if t0 > 0 {
		_, p = chopQuadAt(p, t0)
		t1 = (t1 - t0) / (1 - t0)
	}
	if t1 < 1 {
		p, _ = chopQuadAt(p, t1)
	}
	return p
}

// subCubic returns the part of the cubic between t0 and t1.
func subCubic(p [4]Offset, t0, t1 float64) [4]Offset {
	if t0 > 0 {
		_, p = chopCubicAt(p, t0)
		t1 = (t1 - t0) / (1 - t0)
	}
	if t1 < 1 {
		p, _ = chopCubicAt(p, t1)
	}
	return p
}

// subConic returns the part of the conic between t0 and t1.
func subConic(p [3]Offset, w, t0, t1 float64) ([3]Offset, float64) {
	if t0 > 0 {
		_, _, p, w = chopConicAt(p, w, t0)
		t1 = (t1 - t0) / (1 - t0)
	}
	if t1 < 1 {
		p, w, _, _ = chopConicAt(p, w, t1)
	}
	return p, w
}

// boundsAcc accumulates a bounding box.
type boundsAcc struct {
	l, t, r, b float64
	any        bool
}

func (a *boundsAcc) add(p Offset) {
	if !a.any {
		a.l, a.r, a.t, a.b, a.any = p.Dx, p.Dx, p.Dy, p.Dy, true
		return
	}
	a.l = min(a.l, p.Dx)
	a.r = max(a.r, p.Dx)
	a.t = min(a.t, p.Dy)
	a.b = max(a.b, p.Dy)
}

func (a *boundsAcc) rect() Rect {
	if !a.any {
		return Rect{}
	}
	return Rect{a.l, a.t, a.r, a.b}
}

// quadExtrema returns the t in (0, 1) where a coordinate of the quadratic is at
// an extreme.
func quadExtrema(a, b, c float64) (float64, bool) { return validUnitDivide(a-b, a-b-b+c) }

// cubicExtrema returns the t in (0, 1) where a coordinate of the cubic is at an
// extreme.
func cubicExtrema(a, b, c, d float64) []float64 {
	return findUnitQuadRoots(d-a+3*(b-c), 2*(a-b-b+c), b-a)
}

// conicExtrema returns the t in (0, 1) where a coordinate of the conic is at an
// extreme.
func conicExtrema(p0, p1, p2, w float64) (float64, bool) {
	p20 := p2 - p0
	p10 := p1 - p0
	wp10 := w * p10
	roots := findUnitQuadRoots(w*p20-p20, p20-2*wp10, wp10)
	if len(roots) == 1 {
		return roots[0], true
	}
	return 0, false
}

// addQuadTight adds the extreme points of the quadratic to the bounds.
func (a *boundsAcc) addQuadTight(p0, p1, p2 Offset) {
	a.add(p0)
	a.add(p2)
	if t, ok := quadExtrema(p0.Dx, p1.Dx, p2.Dx); ok {
		a.add(quadAt(p0, p1, p2, t))
	}
	if t, ok := quadExtrema(p0.Dy, p1.Dy, p2.Dy); ok {
		a.add(quadAt(p0, p1, p2, t))
	}
}

func (a *boundsAcc) addCubicTight(p0, p1, p2, p3 Offset) {
	a.add(p0)
	a.add(p3)
	for _, t := range cubicExtrema(p0.Dx, p1.Dx, p2.Dx, p3.Dx) {
		a.add(cubicAt(p0, p1, p2, p3, t))
	}
	for _, t := range cubicExtrema(p0.Dy, p1.Dy, p2.Dy, p3.Dy) {
		a.add(cubicAt(p0, p1, p2, p3, t))
	}
}

func (a *boundsAcc) addConicTight(p0, p1, p2 Offset, w float64) {
	a.add(p0)
	a.add(p2)
	if t, ok := conicExtrema(p0.Dx, p1.Dx, p2.Dx, w); ok {
		a.add(conicAt(p0, p1, p2, w, t))
	}
	if t, ok := conicExtrema(p0.Dy, p1.Dy, p2.Dy, w); ok {
		a.add(conicAt(p0, p1, p2, w, t))
	}
}
