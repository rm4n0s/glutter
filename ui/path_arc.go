// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

import "math"

// sinSnapToZero and cosSnapToZero are sin and cos with results that are nearly
// zero snapped to exactly zero.
func sinSnapToZero(rad float64) float64 { return snapToZero(math.Sin(rad)) }
func cosSnapToZero(rad float64) float64 { return snapToZero(math.Cos(rad)) }

func snapToZero(v float64) float64 {
	if math.Abs(v) <= scalarNearlyZero {
		return 0
	}
	return v
}

// ArcTo appends a conic-built arc of the oval bounded by rect, starting at
// startAngle radians around the oval and going sweepAngle radians (positive is
// clockwise). The angle 0 is at the right of the oval. If forceMoveTo is false,
// a line from the current point to the arc's start is added first (unless they
// are the same point); if it is true, a new sub-path starts at the arc.
//
// Flutter: ui/painting.dart Path.arcTo
func (p *Path) ArcTo(rect Rect, startAngle, sweepAngle float64, forceMoveTo bool) {
	p.arcToDegrees(rect, startAngle*180/math.Pi, sweepAngle*180/math.Pi, forceMoveTo)
}

// arcToDegrees is skia's SkPath::arcTo(oval, startAngle, sweepAngle,
// forceMoveTo), with the angles in degrees.
func (p *Path) arcToDegrees(oval Rect, startAngle, sweepAngle float64, forceMoveTo bool) {
	if oval.Width() < 0 || oval.Height() < 0 {
		return
	}
	if len(p.verbs) == 0 {
		forceMoveTo = true
	}
	if lone, ok := arcIsLonePoint(oval, startAngle, sweepAngle); ok {
		if forceMoveTo {
			p.MoveTo(lone.Dx, lone.Dy)
		} else {
			p.LineTo(lone.Dx, lone.Dy)
		}
		return
	}
	startV, stopV, cw := anglesToUnitVectors(startAngle, sweepAngle)
	// Adds a move to pt if forceMoveTo is true. Otherwise a line, unless we are
	// sufficiently close to pt already, which prevents spurious lines when adding
	// a series of contiguous arcs from the same oval.
	addPt := func(pt Offset) {
		if forceMoveTo {
			p.MoveTo(pt.Dx, pt.Dy)
		} else if !nearlyEqualPoints(p.lastPt(), pt) {
			p.LineTo(pt.Dx, pt.Dy)
		}
	}
	// The arc is not a lone point, but equal vectors mean the sweep is too small
	// for angles_to_unit_vectors to tell apart.
	if sameUnitVector(startV, stopV) {
		endAngle := (startAngle + sweepAngle) * math.Pi / 180
		rx, ry := oval.Width()/2, oval.Height()/2
		addPt(Offset{oval.Left + rx + rx*math.Cos(endAngle), oval.Top + ry + ry*math.Sin(endAngle)})
		return
	}
	conics, single := buildArcConics(oval, startV, stopV, cw)
	if len(conics) == 0 {
		addPt(single)
		return
	}
	addPt(conics[0].p[0])
	for _, c := range conics {
		p.ConicTo(c.p[1].Dx, c.p[1].Dy, c.p[2].Dx, c.p[2].Dy, c.w)
	}
}

// arcIsLonePoint reports whether the arc is a single point and where.
func arcIsLonePoint(oval Rect, startAngle, sweepAngle float64) (Offset, bool) {
	if sweepAngle == 0 && (startAngle == 0 || startAngle == 360) {
		// This path is a single point at the right of the oval.
		return Offset{oval.Right, oval.Top + oval.Height()/2}, true
	}
	if oval.Width() == 0 && oval.Height() == 0 {
		// Round rects with zero radius: a degenerate quad would keep the path from
		// being recognized as a rect.
		return Offset{oval.Right, oval.Bottom}, true
	}
	return Offset{}, false
}

// sameUnitVector reports whether two unit vectors are the same as far as Skia,
// which computes them in float32, could tell: they differ by less than the
// precision of a float32.
func sameUnitVector(a, b Offset) bool {
	const eps = 1.2e-7
	return math.Abs(a.Dx-b.Dx) <= eps && math.Abs(a.Dy-b.Dy) <= eps
}

// anglesToUnitVectors returns the unit vectors of the start and the end of the
// arc, and whether it goes clockwise.
func anglesToUnitVectors(startAngle, sweepAngle float64) (startV, stopV Offset, cw bool) {
	startRad := startAngle * math.Pi / 180
	stopRad := (startAngle + sweepAngle) * math.Pi / 180
	startV = Offset{cosSnapToZero(startRad), sinSnapToZero(startRad)}
	stopV = Offset{cosSnapToZero(stopRad), sinSnapToZero(stopRad)}
	// If the sweep angle is nearly (but less than) 360, then because of precision
	// loss the vectors may be the same, which would draw nothing instead of an
	// almost complete circle. Tweak the stop vector.
	if sameUnitVector(startV, stopV) {
		sw := math.Abs(sweepAngle)
		if sw < 360 && sw > 359 {
			delta := math.Copysign(1.0/512, sweepAngle)
			for sameUnitVector(startV, stopV) {
				stopRad -= delta
				stopV = Offset{cosSnapToZero(stopRad), sinSnapToZero(stopRad)}
			}
		}
	}
	return startV, stopV, sweepAngle > 0
}

// conicArc is one conic of an arc.
type conicArc struct {
	p [3]Offset
	w float64
}

// buildArcConics is skia's build_arc_conics: the conics of the arc of the oval
// between the unit vectors, or the single end point when the arc is too small.
func buildArcConics(oval Rect, start, stop Offset, cw bool) ([]conicArc, Offset) {
	sx, sy := oval.Width()/2, oval.Height()/2
	cx, cy := oval.Left+sx, oval.Top+sy
	conics := buildUnitArc(start, stop, cw)
	if len(conics) == 0 {
		return nil, Offset{stop.Dx*sx + cx, stop.Dy*sy + cy}
	}
	for i := range conics {
		for j := range conics[i].p {
			q := conics[i].p[j]
			conics[i].p[j] = Offset{q.Dx*sx + cx, q.Dy*sy + cy}
		}
	}
	return conics, Offset{}
}

// buildUnitArc is skia's SkConic::BuildUnitArc: up to five conics, one per
// quadrant, for the arc of the unit circle from uStart to uStop.
func buildUnitArc(uStart, uStop Offset, cw bool) []conicArc {
	// Rotate by x, y so that uStart is (1, 0).
	x := uStart.Dx*uStop.Dx + uStart.Dy*uStop.Dy
	y := uStart.Dx*uStop.Dy - uStart.Dy*uStop.Dx
	absY := math.Abs(y)
	// Check for effectively coincident vectors; this happens if the angle is
	// nearly 0 or nearly 180 (y == 0). The dot product tells them apart (x > 0).
	if absY <= scalarNearlyZero && x > 0 && ((y >= 0 && cw) || (y <= 0 && !cw)) {
		return nil
	}
	if !cw {
		y = -y
	}
	// One conic per quadrant of a circle. Which quadrant does (x, y) lie in?
	quadrant := 0
	switch {
	case y == 0:
		quadrant = 2 // 180
	case x == 0:
		if y > 0 {
			quadrant = 1 // 90
		} else {
			quadrant = 3 // 270
		}
	default:
		if y < 0 {
			quadrant += 2
		}
		if (x < 0) != (y < 0) {
			quadrant++
		}
	}
	quadrantPts := [8]Offset{{1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}}
	const quadrantWeight = math.Sqrt2 / 2
	var dst []conicArc
	for i := 0; i < quadrant; i++ {
		dst = append(dst, conicArc{[3]Offset{quadrantPts[i*2], quadrantPts[i*2+1], quadrantPts[(i*2+2)%8]}, quadrantWeight})
	}
	// Now the remaining (sub-90-degree) arc for the last conic.
	finalP := Offset{x, y}
	lastQ := quadrantPts[(quadrant*2)%8] // a unit vector
	dot := lastQ.Dx*finalP.Dx + lastQ.Dy*finalP.Dy
	if dot < 1 {
		offCurve := Offset{lastQ.Dx + x, lastQ.Dy + y}
		// The bisector, rescaled to be the off-curve point: its length comes from
		// cos(theta/2) = length / 1, and the weight is cos(theta/2) as well.
		cosThetaOver2 := math.Sqrt((1 + dot) / 2)
		if l := math.Hypot(offCurve.Dx, offCurve.Dy); l > 0 {
			offCurve = offCurve.Mul(1 / cosThetaOver2 / l)
		}
		if !(nearlyEqualPoints(lastQ, offCurve)) {
			dst = append(dst, conicArc{[3]Offset{lastQ, offCurve, finalP}, cosThetaOver2})
		}
	}
	// Counter-clockwise and the rotation of the initial uStart.
	for i := range dst {
		for j := range dst[i].p {
			q := dst[i].p[j]
			if !cw {
				q.Dy = -q.Dy
			}
			// Rotate by the angle of uStart: (c, s) = (uStart.x, uStart.y).
			dst[i].p[j] = Offset{q.Dx*uStart.Dx - q.Dy*uStart.Dy, q.Dx*uStart.Dy + q.Dy*uStart.Dx}
		}
	}
	return dst
}

// ArcToPointOpts are the optional arguments of ArcToPoint and
// RelativeArcToPoint.
//
// Flutter: ui/painting.dart Path.arcToPoint
type ArcToPointOpts struct {
	// Radius is the radii of the ellipse the arc is on. The zero value (Flutter's
	// Radius.zero) makes a straight line; radii that are too small to reach the
	// end are scaled up.
	Radius Radius
	// Rotation is the rotation of the ellipse, in radians.
	Rotation float64
	// LargeArc picks the arc of more than half of the ellipse, among the two
	// that reach the end point in the same direction.
	LargeArc bool
	// Clockwise is the direction of the arc; nil means true, Flutter's default.
	Clockwise *bool
}

// ArcToPoint appends up to four conic curves weighted to describe an oval arc
// that starts at the current point and ends at arcEnd, as an SVG elliptical arc
// does. If the radii are zero or the points are the same, a line is added.
//
// Flutter: ui/painting.dart Path.arcToPoint
func (p *Path) ArcToPoint(arcEnd Offset, opts ...ArcToPointOpts) {
	o := oneOpts(opts)
	p.arcToPoint(o.Radius.X, o.Radius.Y, o.Rotation*180/math.Pi, o.LargeArc, o.Clockwise == nil || *o.Clockwise, arcEnd.Dx, arcEnd.Dy)
}

// RelativeArcToPoint is ArcToPoint with the end given as an offset from the
// current point.
//
// Flutter: ui/painting.dart Path.relativeArcToPoint
func (p *Path) RelativeArcToPoint(arcEndDelta Offset, opts ...ArcToPointOpts) {
	o := oneOpts(opts)
	p.injectMoveToIfNeeded()
	last := p.lastPt()
	p.arcToPoint(o.Radius.X, o.Radius.Y, o.Rotation*180/math.Pi, o.LargeArc, o.Clockwise == nil || *o.Clockwise,
		last.Dx+arcEndDelta.Dx, last.Dy+arcEndDelta.Dy)
}

// arcToPoint is skia's SkPath::arcTo(rx, ry, xAxisRotate, arcSize, sweep, x, y),
// which follows the SVG implementation notes for elliptical arcs. The rotation
// is in degrees.
func (p *Path) arcToPoint(rx, ry, angle float64, largeArc, cw bool, x, y float64) {
	p.injectMoveToIfNeeded()
	start := p.lastPt()
	// If rx = 0 or ry = 0 then this arc is treated as a straight line segment
	// (https://www.w3.org/TR/SVG/implnote.html#ArcOutOfRangeParameters).
	if rx == 0 || ry == 0 {
		p.LineTo(x, y)
		return
	}
	// If the current point and the target point are the same, it is a zero length
	// path, which keeps animations continuous.
	end := Offset{x, y}
	if start == end {
		p.LineTo(x, y)
		return
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	phi := angle * math.Pi / 180
	cosPhi, sinPhi := math.Cos(phi), math.Sin(phi)
	// Step 1: the mid point, in the frame of the ellipse (rotated by -phi).
	mid := start.Sub(end).Mul(0.5)
	x1p := cosPhi*mid.Dx + sinPhi*mid.Dy
	y1p := -sinPhi*mid.Dx + cosPhi*mid.Dy
	// If the radii are not big enough to draw the arc, scale them
	// (https://www.w3.org/TR/SVG/implnote.html#ArcCorrectionOutOfRangeRadii).
	radiiScale := (x1p*x1p)/(rx*rx) + (y1p*y1p)/(ry*ry)
	if radiiScale > 1 {
		radiiScale = math.Sqrt(radiiScale)
		rx *= radiiScale
		ry *= radiiScale
	}
	// Work on the unit circle: scale by 1/r after rotating by -phi.
	toUnit := func(q Offset) Offset {
		rxq := cosPhi*q.Dx + sinPhi*q.Dy
		ryq := -sinPhi*q.Dx + cosPhi*q.Dy
		return Offset{rxq / rx, ryq / ry}
	}
	fromUnit := func(q Offset) Offset {
		sx, sy := q.Dx*rx, q.Dy*ry
		return Offset{cosPhi*sx - sinPhi*sy, sinPhi*sx + cosPhi*sy}
	}
	unit0, unit1 := toUnit(start), toUnit(end)
	delta := unit1.Sub(unit0)
	d := delta.Dx*delta.Dx + delta.Dy*delta.Dy
	scaleFactor := math.Sqrt(math.Max(1/d-0.25, 0))
	// The center is on the perpendicular bisector; which side depends on the
	// arc size and the direction.
	if largeArc != !cw { // SVG: the sign is + when the large-arc and sweep flags differ
		scaleFactor = -scaleFactor
	}
	delta = delta.Mul(scaleFactor)
	center := unit0.Add(unit1).Mul(0.5)
	center = Offset{center.Dx - delta.Dy, center.Dy + delta.Dx}
	u0, u1 := unit0.Sub(center), unit1.Sub(center)
	theta1 := math.Atan2(u0.Dy, u0.Dx)
	theta2 := math.Atan2(u1.Dy, u1.Dx)
	thetaArc := theta2 - theta1
	if thetaArc < 0 && cw {
		thetaArc += math.Pi * 2
	} else if thetaArc > 0 && !cw {
		thetaArc -= math.Pi * 2
	}
	// Very tiny angles make the math below go wrong (skbug.com/9272).
	if math.Abs(thetaArc) < math.Pi/(1000*1000) {
		p.LineTo(x, y)
		return
	}
	// The arc may be slightly bigger than 1/4 circle, so allow up to 1/3.
	segments := int(math.Ceil(math.Abs(thetaArc / (2 * math.Pi / 3))))
	thetaWidth := thetaArc / float64(segments)
	t := math.Tan(0.5 * thetaWidth)
	if math.IsInf(t, 0) || math.IsNaN(t) {
		return
	}
	startTheta := theta1
	w := math.Sqrt(0.5 + math.Cos(thetaWidth)*0.5)
	isInteger := func(v float64) bool { return v == math.Floor(v) }
	expectIntegers := nearlyEqual(math.Pi/2, math.Abs(thetaWidth)) &&
		isInteger(rx) && isInteger(ry) && isInteger(x) && isInteger(y)
	// The conics are in the frame where the ellipse is the unit circle around
	// center; map them back to the plane.
	for i := 0; i < segments; i++ {
		endTheta := startTheta + thetaWidth
		sinEnd, cosEnd := sinSnapToZero(endTheta), cosSnapToZero(endTheta)
		end := Offset{cosEnd, sinEnd}.Add(center)
		ctrl := end.Add(Offset{t * sinEnd, -t * cosEnd})
		mappedCtrl, mappedEnd := fromUnit(ctrl), fromUnit(end)
		if expectIntegers {
			// Computing the arc width introduces rounding errors that make arcs start
			// outside their marks and a round rect lose convexity. With integer inputs
			// put the conic on integers too.
			mappedCtrl = Offset{math.Round(mappedCtrl.Dx), math.Round(mappedCtrl.Dy)}
			mappedEnd = Offset{math.Round(mappedEnd.Dx), math.Round(mappedEnd.Dy)}
		}
		p.ConicTo(mappedCtrl.Dx, mappedCtrl.Dy, mappedEnd.Dx, mappedEnd.Dy, w)
		startTheta = endTheta
	}
	// The final point must match the input point by definition; replace it so
	// that rounding errors above cannot matter.
	p.points[len(p.points)-1] = Offset{x, y}
}

// AddArc adds a new sub-path with one arc segment that consists of the arc that
// follows the edge of the oval bounded by the given rectangle, from startAngle
// radians around the oval up to startAngle + sweepAngle radians around the
// oval, with zero radians being the point on the right hand side of the oval
// that crosses the horizontal line that intersects the center of the rectangle
// and with positive angles going clockwise around the oval.
//
// Flutter: ui/painting.dart Path.addArc
func (p *Path) AddArc(oval Rect, startAngle, sweepAngle float64) {
	p.addArcDegrees(oval, startAngle*180/math.Pi, sweepAngle*180/math.Pi)
}

func (p *Path) addArcDegrees(oval Rect, startAngle, sweepAngle float64) {
	if oval.IsEmpty() || sweepAngle == 0 {
		return
	}
	const fullCircleAngle = 360.0
	if sweepAngle >= fullCircleAngle || sweepAngle <= -fullCircleAngle {
		// The arc can be treated as an oval if it begins at one of the legal
		// starting positions of addOval.
		startOver90 := startAngle / 90
		startOver90I := math.Round(startOver90)
		if nearlyEqual(startOver90-startOver90I, 0) {
			// Index 1 is at startAngle == 0.
			startIndex := startOver90I + 1
			startIndex -= math.Floor(startIndex/4) * 4
			p.addOvalDir(oval, sweepAngle > 0, int(startIndex))
			return
		}
	}
	p.arcToDegrees(oval, startAngle, sweepAngle, true)
}

// skRRect is a rounded rectangle normalized the way Skia's SkRRect does it: a
// sorted rectangle, radii that fit, and zero in both axes of a corner when
// either is zero.
type skRRect struct {
	rect  Rect
	radii [4]Radius // top left, top right, bottom right, bottom left
	kind  skRRectKind
}

type skRRectKind int

const (
	rrectEmpty skRRectKind = iota
	rrectRect
	rrectOval
	rrectOther
)

func newSkRRect(rr rrectLike) skRRect {
	r := Rect{math.Min(rr.Left, rr.Right), math.Min(rr.Top, rr.Bottom), math.Max(rr.Left, rr.Right), math.Max(rr.Top, rr.Bottom)}
	out := skRRect{rect: r}
	if r.IsEmpty() || !r.IsFinite() {
		out.kind = rrectEmpty
		return out
	}
	radii := [4]Radius{{rr.TlRadiusX, rr.TlRadiusY}, {rr.TrRadiusX, rr.TrRadiusY}, {rr.BrRadiusX, rr.BrRadiusY}, {rr.BlRadiusX, rr.BlRadiusY}}
	allZero := true
	for i := range radii {
		if radii[i].X <= 0 || radii[i].Y <= 0 {
			radii[i] = Radius{}
		} else {
			allZero = false
		}
	}
	if allZero {
		out.kind = rrectRect
		return out
	}
	scaled := rrectLike{
		Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Bottom,
		TlRadiusX: radii[0].X, TlRadiusY: radii[0].Y, TrRadiusX: radii[1].X, TrRadiusY: radii[1].Y,
		BrRadiusX: radii[2].X, BrRadiusY: radii[2].Y, BlRadiusX: radii[3].X, BlRadiusY: radii[3].Y,
	}.scaleRadii()
	out.radii = [4]Radius{scaled.tlRadius(), scaled.trRadius(), scaled.brRadius(), scaled.blRadius()}
	same := out.radii[0] == out.radii[1] && out.radii[1] == out.radii[2] && out.radii[2] == out.radii[3]
	if same && nearlyEqual(out.radii[0].X, r.Width()/2) && nearlyEqual(out.radii[0].Y, r.Height()/2) {
		out.kind = rrectOval
	} else {
		out.kind = rrectOther
	}
	return out
}

// point is point i of the rounded rectangle's outline, clockwise from the top
// edge: 0 and 1 on the top edge, 2 and 3 on the right, 4 and 5 on the bottom,
// 6 and 7 on the left.
func (r skRRect) point(i int) Offset {
	b := r.rect
	switch i & 7 {
	case 0:
		return Offset{b.Left + r.radii[0].X, b.Top}
	case 1:
		return Offset{b.Right - r.radii[1].X, b.Top}
	case 2:
		return Offset{b.Right, b.Top + r.radii[1].Y}
	case 3:
		return Offset{b.Right, b.Bottom - r.radii[2].Y}
	case 4:
		return Offset{b.Right - r.radii[2].X, b.Bottom}
	case 5:
		return Offset{b.Left + r.radii[3].X, b.Bottom}
	case 6:
		return Offset{b.Left, b.Bottom - r.radii[3].Y}
	}
	return Offset{b.Left, b.Top + r.radii[0].Y}
}

// AddRRect adds a new sub-path that consists of the straight lines and
// curves needed to form the rounded rectangle described by the argument. It
// starts on the left edge, above the bottom left corner, and goes clockwise.
//
// Flutter: ui/painting.dart Path.addRRect
func (p *Path) AddRRect(rrect RRect) {
	p.addRRectDir(newSkRRect(rrectLike(rrect)), true, 6)
}

// addRRectDir is skia's SkPath::addRRect(rrect, dir, startIndex).
func (p *Path) addRRectDir(rr skRRect, cw bool, startIndex int) {
	switch rr.kind {
	case rrectRect, rrectEmpty:
		// Degenerate to a rect: the radius points are collapsing.
		p.addRectDir(rr.rect, cw, (startIndex+1)/2)
		return
	case rrectOval:
		// Degenerate to an oval: the line points are collapsing.
		p.addOvalDir(rr.rect, cw, startIndex/2)
		return
	}
	const weight = math.Sqrt2 / 2
	// Start with a conic on odd indices when moving clockwise and on even indices
	// when moving counter-clockwise.
	startsWithConic := (startIndex&1 == 1) == cw
	ri := startIndex
	// The corner iterator follows the collapsed radii model, adjusted so that the
	// start point is "behind" the radii start point.
	ci := startIndex / 2
	if !cw {
		ci++
	}
	at := rr.point(ri)
	p.MoveTo(at.Dx, at.Dy)
	conic := func() {
		ci = stepIndex(ci, 4, cw)
		ri = stepIndex(ri, 8, cw)
		c, e := rectCorner(rr.rect, ci), rr.point(ri)
		p.ConicTo(c.Dx, c.Dy, e.Dx, e.Dy, weight)
	}
	line := func() {
		ri = stepIndex(ri, 8, cw)
		e := rr.point(ri)
		p.LineTo(e.Dx, e.Dy)
	}
	if startsWithConic {
		for i := 0; i < 3; i++ {
			conic()
			line()
		}
		conic()
		// The final line is done by the close.
	} else {
		for i := 0; i < 4; i++ {
			line()
			conic()
		}
	}
	p.Close()
}

// AddRSuperellipse adds a new sub-path that consists of the straight lines and
// curves needed to form the rounded superellipse described by the argument.
//
// Deviation: the corners follow the approximation of RSuperellipse.Contains, not
// Impeller's exact shape; see SPEC §12.10.
//
// Flutter: ui/painting.dart Path.addRSuperellipse
func (p *Path) AddRSuperellipse(rs RSuperellipse) {
	rr := newSkRRect(rrectLike(rs))
	if rr.kind == rrectRect || rr.kind == rrectEmpty {
		p.addRectDir(rr.rect, true, 3)
		return
	}
	b := rr.rect
	// The box each corner occupies: its radius times the extent, as far as it fits.
	box := func(r Radius) (float64, float64) {
		return math.Min(r.X*superellipseExtent, b.Width()/2), math.Min(r.Y*superellipseExtent, b.Height()/2)
	}
	tlx, tly := box(rr.radii[0])
	trx, try := box(rr.radii[1])
	brx, bry := box(rr.radii[2])
	blx, bly := box(rr.radii[3])
	p.MoveTo(b.Left, b.Bottom-bly)
	p.LineTo(b.Left, b.Top+tly)
	// Each corner goes from the unit curve (1, 0) to (0, 1) mapped to the box.
	p.superellipseCorner(func(x, y float64) Offset { return Offset{b.Left + tlx - tlx*x, b.Top + tly - tly*y} })
	p.LineTo(b.Right-trx, b.Top)
	p.superellipseCorner(func(x, y float64) Offset { return Offset{b.Right - trx + trx*y, b.Top + try - try*x} })
	p.LineTo(b.Right, b.Bottom-bry)
	p.superellipseCorner(func(x, y float64) Offset { return Offset{b.Right - brx + brx*x, b.Bottom - bry + bry*y} })
	p.LineTo(b.Left+blx, b.Bottom)
	p.superellipseCorner(func(x, y float64) Offset { return Offset{b.Left + blx - blx*y, b.Bottom - bly + bly*x} })
	p.Close()
}

// superellipseCorner adds, as cubic curves, the quarter of the unit curve
// |x|^n + |y|^n = 1 from (1, 0) to (0, 1), with its points mapped by place. The
// curve is in polar form r(a) so that its tangent is smooth at the ends.
func (p *Path) superellipseCorner(place func(x, y float64) Offset) {
	const n = superellipseExponent
	const segments = 8
	// point and tangent at angle a, in the unit square.
	eval := func(a float64) (pt, d Offset) {
		c, s := math.Cos(a), math.Sin(a)
		sum := math.Pow(c, n) + math.Pow(s, n)
		r := math.Pow(sum, -1.0/n)
		// dr/da = -sum^(-1/n-1) * (-c^(n-1) s + s^(n-1) c)
		dr := -math.Pow(sum, -1.0/n-1) * (-math.Pow(c, n-1)*s + math.Pow(s, n-1)*c)
		return Offset{r * c, r * s}, Offset{dr*c - r*s, dr*s + r*c}
	}
	step := math.Pi / 2 / segments
	prev, dprev := eval(0)
	for i := 1; i <= segments; i++ {
		cur, dcur := eval(step * float64(i))
		c1 := prev.Add(dprev.Mul(step / 3))
		c2 := cur.Sub(dcur.Mul(step / 3))
		// The polar angle runs from the x axis (1, 0) to the y axis (0, 1).
		a, b, c := place(c1.Dx, c1.Dy), place(c2.Dx, c2.Dy), place(cur.Dx, cur.Dy)
		p.CubicTo(a.Dx, a.Dy, b.Dx, b.Dy, c.Dx, c.Dy)
		prev, dprev = cur, dcur
	}
}
