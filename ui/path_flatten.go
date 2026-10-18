package ui

import "math"

// FlatContour is a contour of a path turned into straight lines.
//
// Flutter: no counterpart
type FlatContour struct {
	// Points are the vertices of the polyline. For a closed contour the first
	// point is not repeated at the end.
	Points []Offset
	// Closed says that the contour was closed: the last point connects to the
	// first.
	Closed bool
}

// defaultFlattenTolerance is the distance, in the units of the points after the
// transform, that Flatten allows between a curve and its lines when it is given
// no tolerance.
const defaultFlattenTolerance = 0.25

// maxFlattenSplits bounds how many lines a single curve is cut into, and
// maxConicFlattenDepth how many times a conic is halved to get there.
const (
	maxFlattenSplits     = 1 << 12
	maxConicFlattenDepth = 12
)

// Flatten turns the path into polylines: each contour becomes a list of points
// that stay within tolerance of the curves they replace. The points are
// transformed by matrix (which may be nil) first, so the tolerance is in the
// units of the result, such as device pixels. A tolerance that is not positive
// is 0.25. A contour that is only a move is dropped, unless it was closed.
//
// Flutter: no counterpart
func (p *Path) Flatten(matrix *Matrix4, tolerance float64) []FlatContour {
	if !(tolerance > 0) {
		tolerance = defaultFlattenTolerance
	}
	if matrix != nil && matrix.IsIdentity() {
		matrix = nil
	}
	mapPt := func(pt Offset) Offset {
		if matrix == nil {
			return pt
		}
		return matrix.mapPoint(pt)
	}
	var out []FlatContour
	var cur []Offset
	hasSegment := false
	flush := func(closed bool) {
		if len(cur) > 0 && (hasSegment || closed) {
			out = append(out, FlatContour{Points: cur, Closed: closed})
		}
		cur = nil
		hasSegment = false
	}
	p.ForEachSegment(func(s PathSegment) {
		switch s.Verb {
		case PathVerbMove:
			flush(false)
			cur = []Offset{mapPt(s.P0)}
		case PathVerbLine:
			cur = append(cur, mapPt(s.P1))
			hasSegment = true
		case PathVerbQuad:
			cur = flattenQuad(cur, mapPt(s.P0), mapPt(s.P1), mapPt(s.P2), tolerance)
			hasSegment = true
		case PathVerbConic:
			w := s.Weight
			if matrix != nil && matrix.hasPerspective() {
				w = perspectiveConicWeight(s.P0, s.P1, s.P2, w, matrix)
			}
			cur = flattenConic(cur, [3]Offset{mapPt(s.P0), mapPt(s.P1), mapPt(s.P2)}, w, tolerance, 0)
			hasSegment = true
		case PathVerbCubic:
			cur = flattenCubic(cur, mapPt(s.P0), mapPt(s.P1), mapPt(s.P2), mapPt(s.P3), tolerance)
			hasSegment = true
		case PathVerbClose:
			if len(cur) > 0 {
				flush(true)
				// A segment after the close starts at the contour's start, with its
				// own move that was injected.
			}
		}
	})
	flush(false)
	return out
}

// flattenQuad appends the points of the quadratic after its start (which is the
// last of dst): uniform in t, with as many lines as the tolerance needs.
func flattenQuad(dst []Offset, p0, p1, p2 Offset, tol float64) []Offset {
	d := p0.Sub(p1.Mul(2)).Add(p2)
	n := splitCount(math.Hypot(d.Dx, d.Dy) / (4 * tol))
	for i := 1; i < n; i++ {
		dst = append(dst, quadAt(p0, p1, p2, float64(i)/float64(n)))
	}
	return append(dst, p2)
}

// flattenCubic appends the points of the cubic after its start.
func flattenCubic(dst []Offset, p0, p1, p2, p3 Offset, tol float64) []Offset {
	d1 := p0.Sub(p1.Mul(2)).Add(p2)
	d2 := p1.Sub(p2.Mul(2)).Add(p3)
	m := math.Max(math.Hypot(d1.Dx, d1.Dy), math.Hypot(d2.Dx, d2.Dy))
	n := splitCount(3 * m / (4 * tol))
	for i := 1; i < n; i++ {
		dst = append(dst, cubicAt(p0, p1, p2, p3, float64(i)/float64(n)))
	}
	return append(dst, p3)
}

// flattenConic appends the points of the conic after its start, by cutting it in
// halves until each is within the tolerance of its chord.
func flattenConic(dst []Offset, p [3]Offset, w, tol float64, depth int) []Offset {
	// The middle of a conic is between the middle of the chord and the control
	// point, w/(1+w) of the way to it.
	chordMid := lerpPt(p[0], p[2], 0.5)
	dev := p[1].Sub(chordMid)
	err := math.Hypot(dev.Dx, dev.Dy) * w / (1 + w)
	if err <= tol || depth >= maxConicFlattenDepth || math.IsNaN(err) {
		return append(dst, p[2])
	}
	a, wa, b, wb := chopConicAt(p, w, 0.5)
	dst = flattenConic(dst, a, wa, tol, depth+1)
	return flattenConic(dst, b, wb, tol, depth+1)
}

// splitCount turns an estimate of n^2 into the number of lines: its square root
// rounded up, from 1 to maxFlattenSplits.
func splitCount(squared float64) int {
	if !(squared > 0) {
		return 1
	}
	n := math.Ceil(math.Sqrt(squared))
	if n < 1 {
		return 1
	}
	if n > maxFlattenSplits || math.IsNaN(n) {
		return maxFlattenSplits
	}
	return int(n)
}
