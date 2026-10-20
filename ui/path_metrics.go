// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

import (
	"math"
	"sort"
)

// Tangent is the position and the direction of a path at some distance along it.
//
// Flutter: ui/painting.dart Tangent
type Tangent struct {
	// Position is the point on the path.
	Position Offset
	// Vector is the unit vector pointing the way the path goes there.
	Vector Offset
}

// NewTangent creates a Tangent from a position and a direction vector.
//
// Flutter: ui/painting.dart Tangent
func NewTangent(position, vector Offset) Tangent { return Tangent{position, vector} }

// NewTangentFromAngle creates a Tangent from a position and the angle of its
// direction, in radians, measured clockwise from the positive x axis.
//
// Flutter: ui/painting.dart Tangent.fromAngle
func NewTangentFromAngle(position Offset, angle float64) Tangent {
	return Tangent{position, Offset{math.Cos(angle), math.Sin(angle)}}
}

// Angle is the angle of the direction in radians, counter-clockwise from the
// positive x axis, to be consistent with the sweepAngle of Path.ArcTo.
//
// Flutter: ui/painting.dart Tangent.angle
func (t Tangent) Angle() float64 { return -math.Atan2(t.Vector.Dy, t.Vector.Dx) }

// PathMetric describes one contour of a path: how long it is, and what lies at
// each distance along it.
//
// Flutter: ui/painting.dart PathMetric
type PathMetric struct {
	// Length is the total length of the contour, in the units of the path.
	Length float64
	// IsClosed says whether the contour is closed, or was made so.
	IsClosed bool
	// ContourIndex is the position of this contour among the path's contours that
	// have a length.
	ContourIndex int

	curves []measureCurve
	// segs are the pieces of the curves the length was measured with, in order.
	segs []measureSeg
}

// PathMetrics is the list of the contours of a path, as PathMetric values.
//
// Flutter: ui/painting.dart PathMetrics
type PathMetrics []*PathMetric

// measureCurve is a curve of a contour, as it was in the path.
type measureCurve struct {
	verb PathVerb
	p    [4]Offset
	w    float64
}

// measureSeg is the end of a piece of a curve: the curve, the parameter at the
// end of the piece, and the distance along the contour there.
type measureSeg struct {
	curve int
	t     float64
	dist  float64
}

// measureTolerance is how far a polyline may be from a curve while measuring it.
// It is much finer than Skia's, so that lengths are close to exact.
const measureTolerance = 1e-4

// ComputePathMetricsOpts are the optional arguments of ComputeMetrics.
//
// Flutter: ui/painting.dart Path.computeMetrics
type ComputePathMetricsOpts struct {
	// ForceClosed measures every contour as if it were closed.
	ForceClosed bool
}

// ComputeMetrics returns the metrics of the contours of the path, in order.
// Contours with no length are left out. The path is measured as it is when this
// is called.
//
// Flutter: ui/painting.dart Path.computeMetrics
func (p *Path) ComputeMetrics(opts ...ComputePathMetricsOpts) PathMetrics {
	forceClosed := oneOpts(opts).ForceClosed
	var out PathMetrics
	var curves []measureCurve
	var start, cur Offset
	haveMove := false
	closed := forceClosed
	finish := func() {
		if !haveMove {
			return
		}
		if m := buildMetric(curves, start, cur, closed); m != nil {
			m.ContourIndex = len(out)
			out = append(out, m)
		}
		curves = nil
		closed = forceClosed
	}
	p.ForEachSegment(func(s PathSegment) {
		switch s.Verb {
		case PathVerbMove:
			finish()
			start, cur, haveMove = s.P0, s.P0, true
		case PathVerbLine:
			curves = append(curves, measureCurve{verb: s.Verb, p: [4]Offset{s.P0, s.P1}})
			cur = s.P1
		case PathVerbQuad:
			curves = append(curves, measureCurve{verb: s.Verb, p: [4]Offset{s.P0, s.P1, s.P2}})
			cur = s.P2
		case PathVerbConic:
			curves = append(curves, measureCurve{verb: s.Verb, p: [4]Offset{s.P0, s.P1, s.P2}, w: s.Weight})
			cur = s.P2
		case PathVerbCubic:
			curves = append(curves, measureCurve{verb: s.Verb, p: [4]Offset{s.P0, s.P1, s.P2, s.P3}})
			cur = s.P3
		case PathVerbClose:
			closed = true
		}
	})
	finish()
	return out
}

// buildMetric measures a contour. It returns nil for a contour with no length.
func buildMetric(curves []measureCurve, start, last Offset, closed bool) *PathMetric {
	m := &PathMetric{IsClosed: closed}
	var dist float64
	for i, c := range curves {
		dist = m.measureCurve(c, i, dist)
	}
	if len(m.segs) == 0 || math.IsNaN(dist) || math.IsInf(dist, 0) {
		return nil
	}
	if closed {
		// The closing line, when it has a length.
		before := len(m.segs)
		line := measureCurve{verb: PathVerbLine, p: [4]Offset{last, start}}
		dist = m.measureCurve(line, len(curves), dist)
		m.curves = append(m.curves, curves...)
		if len(m.segs) > before {
			m.curves = append(m.curves, line)
		}
	} else {
		m.curves = append(m.curves, curves...)
	}
	m.Length = dist
	return m
}

// measureCurve adds the pieces of curve c, which is curve number index, to the
// segments, with the contour at distance dist before it. It returns the distance
// after it. Zero-length pieces are dropped.
func (m *PathMetric) measureCurve(c measureCurve, index int, dist float64) float64 {
	add := func(t float64, from, to Offset) {
		d := math.Hypot(to.Dx-from.Dx, to.Dy-from.Dy)
		if d > 0 {
			dist += d
			m.segs = append(m.segs, measureSeg{curve: index, t: t, dist: dist})
		}
	}
	switch c.verb {
	case PathVerbLine:
		add(1, c.p[0], c.p[1])
	default:
		const maxDepth = 14
		var rec func(t0, t1 float64, p0, pm, p1 Offset, depth int)
		pt := func(t float64) Offset {
			switch c.verb {
			case PathVerbQuad:
				return quadAt(c.p[0], c.p[1], c.p[2], t)
			case PathVerbConic:
				return conicAt(c.p[0], c.p[1], c.p[2], c.w, t)
			}
			return cubicAt(c.p[0], c.p[1], c.p[2], c.p[3], t)
		}
		// A piece is kept when its middle is within the tolerance of its chord.
		rec = func(t0, t1 float64, p0, pm, p1 Offset, depth int) {
			chordMid := lerpPt(p0, p1, 0.5)
			dev := math.Hypot(pm.Dx-chordMid.Dx, pm.Dy-chordMid.Dy)
			if dev <= measureTolerance || depth >= maxDepth {
				add(t1, p0, p1)
				return
			}
			tm := (t0 + t1) / 2
			rec(t0, tm, p0, pt((t0+tm)/2), pt(tm), depth+1)
			rec(tm, t1, pt(tm), pt((tm+t1)/2), p1, depth+1)
		}
		end := pt(1)
		rec(0, 1, c.p[0], pt(0.5), end, 0)
	}
	return dist
}

// distanceToSeg finds the segment that holds the distance, and the parameter of
// the curve there. The distance is clamped to the contour.
func (m *PathMetric) distanceToSeg(distance float64) (seg int, t float64, ok bool) {
	if math.IsNaN(distance) || len(m.segs) == 0 {
		return 0, 0, false
	}
	if distance < 0 {
		distance = 0
	} else if distance > m.Length {
		distance = m.Length
	}
	seg = sort.Search(len(m.segs), func(i int) bool { return m.segs[i].dist >= distance })
	if seg == len(m.segs) {
		seg = len(m.segs) - 1
	}
	s := m.segs[seg]
	var startT, startD float64
	if seg > 0 {
		prev := m.segs[seg-1]
		startD = prev.dist
		if prev.curve == s.curve {
			startT = prev.t
		}
	}
	t = startT + (s.t-startT)*(distance-startD)/(s.dist-startD)
	return seg, t, true
}

func (c *measureCurve) posTan(t float64) (pos, tan Offset) {
	switch c.verb {
	case PathVerbLine:
		return lerpPt(c.p[0], c.p[1], t), c.p[1].Sub(c.p[0])
	case PathVerbQuad:
		return quadAt(c.p[0], c.p[1], c.p[2], t), quadTangentAt(c.p[0], c.p[1], c.p[2], t)
	case PathVerbConic:
		return conicAt(c.p[0], c.p[1], c.p[2], c.w, t), conicTangentAt(c.p[0], c.p[1], c.p[2], c.w, t)
	}
	return cubicAt(c.p[0], c.p[1], c.p[2], c.p[3], t), cubicTangentAt(c.p[0], c.p[1], c.p[2], c.p[3], t)
}

// GetTangentForOffset computes the position of the contour at the given
// distance along it, and the unit vector of its direction there. The distance is
// clamped to the contour. It reports false when there is nothing to compute, as
// for a NaN distance.
//
// Flutter: ui/painting.dart PathMetric.getTangentForOffset
func (m *PathMetric) GetTangentForOffset(distance float64) (Tangent, bool) {
	seg, t, ok := m.distanceToSeg(distance)
	if !ok || math.IsNaN(t) {
		return Tangent{}, false
	}
	c := m.curves[m.segs[seg].curve]
	pos, tan := c.posTan(t)
	if l := math.Hypot(tan.Dx, tan.Dy); l > 0 {
		tan = tan.Div(l)
	}
	return Tangent{pos, tan}, true
}

// ExtractPathOpts are the optional arguments of PathMetric.ExtractPath.
//
// Flutter: ui/painting.dart PathMetric.extractPath
type ExtractPathOpts struct {
	// StartWithMoveTo says whether the new path starts with a move to the start of
	// the extracted part; nil means true, Flutter's default. When false the part
	// continues the path being built, if it is added with ExtendWithPath.
	StartWithMoveTo *bool
}

// ExtractPath returns the part of the contour between the distances start and
// end, as a new path. The distances are clamped to the contour. When start is
// after end, or there is nothing to extract, the path is empty.
//
// Flutter: ui/painting.dart PathMetric.extractPath
func (m *PathMetric) ExtractPath(start, end float64, opts ...ExtractPathOpts) *Path {
	o := oneOpts(opts)
	startWithMoveTo := o.StartWithMoveTo == nil || *o.StartWithMoveTo
	dst := NewPath()
	if start < 0 {
		start = 0
	}
	if end > m.Length {
		end = m.Length
	}
	if !(start <= end) || len(m.segs) == 0 {
		return dst
	}
	seg, startT, ok := m.distanceToSeg(start)
	if !ok || math.IsNaN(startT) {
		return dst
	}
	stopSeg, stopT, ok := m.distanceToSeg(end)
	if !ok || math.IsNaN(stopT) {
		return dst
	}
	c := m.segs[seg].curve
	if startWithMoveTo {
		pos, _ := m.curves[c].posTan(startT)
		dst.MoveTo(pos.Dx, pos.Dy)
	}
	stopCurve := m.segs[stopSeg].curve
	if c == stopCurve {
		m.curves[c].segTo(startT, stopT, dst)
		return dst
	}
	used := make([]bool, len(m.curves))
	for _, sg := range m.segs {
		used[sg.curve] = true
	}
	for ; c < stopCurve; c++ {
		// A curve with no length has no piece to extract.
		if used[c] {
			m.curves[c].segTo(startT, 1, dst)
		}
		startT = 0
	}
	m.curves[stopCurve].segTo(0, stopT, dst)
	return dst
}

// segTo appends to dst the part of the curve between t0 and t1.
func (c *measureCurve) segTo(t0, t1 float64, dst *Path) {
	if t0 == t1 {
		// A zero-length part, such as a dash of no length, becomes a line of no
		// length, so that stroking can give it caps.
		last := dst.lastPt()
		dst.LineTo(last.Dx, last.Dy)
		return
	}
	switch c.verb {
	case PathVerbLine:
		if t1 == 1 {
			dst.LineTo(c.p[1].Dx, c.p[1].Dy)
		} else {
			e := lerpPt(c.p[0], c.p[1], t1)
			dst.LineTo(e.Dx, e.Dy)
		}
	case PathVerbQuad:
		q := subQuad([3]Offset{c.p[0], c.p[1], c.p[2]}, t0, t1)
		dst.QuadraticBezierTo(q[1].Dx, q[1].Dy, q[2].Dx, q[2].Dy)
	case PathVerbConic:
		q, w := subConic([3]Offset{c.p[0], c.p[1], c.p[2]}, c.w, t0, t1)
		dst.ConicTo(q[1].Dx, q[1].Dy, q[2].Dx, q[2].Dy, w)
	case PathVerbCubic:
		q := subCubic(c.p, t0, t1)
		dst.CubicTo(q[1].Dx, q[1].Dy, q[2].Dx, q[2].Dy, q[3].Dx, q[3].Dy)
	}
}

// String describes the metric, as Dart's toString does.
//
// Flutter: ui/painting.dart PathMetric.toString
func (m *PathMetric) String() string {
	return "PathMetric(length: " + dartDouble(m.Length) + ", isClosed: " + boolString(m.IsClosed) + ", contourIndex: " + itoa(m.ContourIndex) + ")"
}
