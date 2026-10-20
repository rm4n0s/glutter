// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package soft

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// StrokeStyle describes how a path is stroked. The units are those of the path.
//
// Flutter: ui/painting.dart Paint.strokeWidth
type StrokeStyle struct {
	// Width is the width of the stroke; it must be greater than 0.
	Width float64
	// Cap is the shape of the ends of open contours; StrokeCapUnspecified is
	// StrokeCapButt.
	Cap ui.StrokeCap
	// Join is the shape of the corners; StrokeJoinUnspecified is
	// StrokeJoinMiter.
	Join ui.StrokeJoin
	// MiterLimit is the longest miter, as a multiple of half the width, that a
	// miter join may have before it becomes a bevel; 0 is Flutter's default, 4.
	MiterLimit float64
	// Scale is how much a transform applied to the result magnifies lengths. It
	// only sets the number of segments used for round joins and caps; 0 is 1.
	Scale float64
}

// arcTolerance is how far, in pixels, a round join or cap may be from a circle.
const arcTolerance = 0.05

// Stroke turns flattened contours into the polygons that cover their stroke. The
// polygons all wind the same way, so they are filled together with FillNonZero.
//
// A contour with a single point, after repeated points are removed, draws a dot
// for round and square caps and nothing for butt caps.
//
// Flutter: no counterpart (Skia's SkStroke)
func Stroke(contours []ui.FlatContour, st StrokeStyle) [][]ui.Offset {
	if !(st.Width > 0) || math.IsInf(st.Width, 0) {
		return nil
	}
	s := stroker{hw: st.Width / 2, cap: st.Cap, join: st.Join, miter: st.MiterLimit}
	if s.miter == 0 {
		s.miter = 4
	}
	if s.cap == ui.StrokeCapUnspecified {
		s.cap = ui.StrokeCapButt
	}
	if s.join == ui.StrokeJoinUnspecified {
		s.join = ui.StrokeJoinMiter
	}
	scale := st.Scale
	if !(scale > 0) {
		scale = 1
	}
	r := s.hw * scale
	s.step = 2 * math.Acos(math.Max(-1, 1-arcTolerance/math.Max(r, arcTolerance)))
	s.step = math.Max(s.step, math.Pi/256)
	for _, c := range contours {
		s.contour(c)
	}
	return s.out
}

type stroker struct {
	hw    float64
	cap   ui.StrokeCap
	join  ui.StrokeJoin
	miter float64
	step  float64 // the angle of one segment of a round join or cap
	out   [][]ui.Offset
}

func (s *stroker) contour(c ui.FlatContour) {
	pts := dedupe(c.Points, c.Closed)
	switch {
	case len(pts) == 0:
		return
	case len(pts) == 1:
		s.dot(pts[0])
		return
	}
	rev := reversed(pts)
	if c.Closed {
		s.out = append(s.out, s.side(pts, true, nil), s.side(rev, true, nil))
		return
	}
	poly := s.side(pts, false, nil)
	poly = s.endCap(poly, pts[len(pts)-1], unit(pts[len(pts)-1], pts[len(pts)-2]))
	poly = s.side(rev, false, poly)
	dStart := unit(pts[1], pts[0])
	dStart = ui.Offset{Dx: -dStart.Dx, Dy: -dStart.Dy}
	poly = s.endCap(poly, pts[0], dStart)
	s.out = append(s.out, poly)
}

// dot draws the stroke of a contour that is a single point.
func (s *stroker) dot(p ui.Offset) {
	switch s.cap {
	case ui.StrokeCapRound:
		poly := make([]ui.Offset, 0, 8)
		n := s.arcSteps(2 * math.Pi)
		for i := 0; i < n; i++ {
			a := -2 * math.Pi * float64(i) / float64(n)
			poly = append(poly, ui.Offset{Dx: p.Dx + s.hw*math.Cos(a), Dy: p.Dy + s.hw*math.Sin(a)})
		}
		s.out = append(s.out, poly)
	case ui.StrokeCapSquare:
		s.out = append(s.out, []ui.Offset{
			{Dx: p.Dx - s.hw, Dy: p.Dy - s.hw}, {Dx: p.Dx - s.hw, Dy: p.Dy + s.hw},
			{Dx: p.Dx + s.hw, Dy: p.Dy + s.hw}, {Dx: p.Dx + s.hw, Dy: p.Dy - s.hw},
		})
	}
}

// arcSteps returns the number of segments for an arc of the given angle.
func (s *stroker) arcSteps(angle float64) int {
	return max(int(math.Ceil(math.Abs(angle)/s.step)), 1)
}

// side appends to out the outline that runs along the left of the points, with a
// join at every vertex.
func (s *stroker) side(pts []ui.Offset, closed bool, out []ui.Offset) []ui.Offset {
	n := len(pts)
	segs := n - 1
	if closed {
		segs = n
	}
	dir := func(i int) ui.Offset { return unit(pts[(i+1)%n], pts[i]) }
	for i := 0; i < segs; i++ {
		d := dir(i)
		if i == 0 && !closed {
			out = append(out, s.offset(pts[0], d))
		}
		if i > 0 || closed {
			prev := i - 1
			if prev < 0 {
				prev = segs - 1
			}
			out = s.joinAt(out, pts[i], dir(prev), d)
		}
		if i == segs-1 && !closed {
			out = append(out, s.offset(pts[n-1], d))
		}
	}
	return out
}

// offset returns the point at half a width to the left of p, looking along d.
func (s *stroker) offset(p, d ui.Offset) ui.Offset {
	return ui.Offset{Dx: p.Dx - d.Dy*s.hw, Dy: p.Dy + d.Dx*s.hw}
}

// joinAt appends the left side of the corner at p, between the directions d0 and
// d1.
func (s *stroker) joinAt(out []ui.Offset, p, d0, d1 ui.Offset) []ui.Offset {
	const eps = 1e-9
	cross := d0.Dx*d1.Dy - d0.Dy*d1.Dx
	dot := d0.Dx*d1.Dx + d0.Dy*d1.Dy
	a, b := s.offset(p, d0), s.offset(p, d1)
	if math.Abs(cross) < eps && dot > 0 {
		return append(out, a)
	}
	outer := cross < 0 || (math.Abs(cross) < eps && dot < 0)
	if !outer {
		return append(out, a, p, b)
	}
	switch s.join {
	case ui.StrokeJoinRound:
		sweep := math.Atan2(cross, dot)
		if math.Abs(cross) < eps {
			sweep = -math.Pi
		}
		n := s.arcSteps(sweep)
		out = append(out, a)
		for i := 1; i < n; i++ {
			out = append(out, rotate(p, a, sweep*float64(i)/float64(n)))
		}
		return append(out, b)
	case ui.StrokeJoinMiter:
		// The miter is as long as 1 / cos(φ/2), φ being the angle the path turns.
		if k := 1 + dot; k > eps && 1/math.Sqrt(k/2) <= s.miter {
			return append(out, a, ui.Offset{Dx: p.Dx + (a.Dx-p.Dx+b.Dx-p.Dx)/k, Dy: p.Dy + (a.Dy-p.Dy+b.Dy-p.Dy)/k}, b)
		}
	}
	return append(out, a, b)
}

// endCap appends the cap at the end p of an open contour, for the outline that
// has just arrived at p+offset(d) and goes on at p-offset(d); d points away from
// the contour.
func (s *stroker) endCap(out []ui.Offset, p, d ui.Offset) []ui.Offset {
	switch s.cap {
	case ui.StrokeCapSquare:
		a := s.offset(p, d)
		ext := ui.Offset{Dx: d.Dx * s.hw, Dy: d.Dy * s.hw}
		return append(out,
			ui.Offset{Dx: a.Dx + ext.Dx, Dy: a.Dy + ext.Dy},
			ui.Offset{Dx: 2*p.Dx - a.Dx + ext.Dx, Dy: 2*p.Dy - a.Dy + ext.Dy})
	case ui.StrokeCapRound:
		a := s.offset(p, d)
		n := s.arcSteps(math.Pi)
		for i := 1; i < n; i++ {
			out = append(out, rotate(p, a, -math.Pi*float64(i)/float64(n)))
		}
	}
	return out
}

// rotate turns the point a around c by the angle, counterclockwise for a
// positive angle in a plane whose y points up.
func rotate(c, a ui.Offset, angle float64) ui.Offset {
	sin, cos := math.Sincos(angle)
	x, y := a.Dx-c.Dx, a.Dy-c.Dy
	return ui.Offset{Dx: c.Dx + x*cos - y*sin, Dy: c.Dy + x*sin + y*cos}
}

// unit returns the direction from b to a, of length 1.
func unit(a, b ui.Offset) ui.Offset {
	dx, dy := a.Dx-b.Dx, a.Dy-b.Dy
	l := math.Hypot(dx, dy)
	return ui.Offset{Dx: dx / l, Dy: dy / l}
}

// dedupe removes points that repeat the one before them, and for a closed
// contour a last point equal to the first.
func dedupe(pts []ui.Offset, closed bool) []ui.Offset {
	out := make([]ui.Offset, 0, len(pts))
	for _, p := range pts {
		if len(out) > 0 && same(out[len(out)-1], p) {
			continue
		}
		out = append(out, p)
	}
	if closed && len(out) > 1 && same(out[0], out[len(out)-1]) {
		out = out[:len(out)-1]
	}
	return out
}

func same(a, b ui.Offset) bool { return math.Hypot(a.Dx-b.Dx, a.Dy-b.Dy) < 1e-9 }

func reversed(pts []ui.Offset) []ui.Offset {
	out := make([]ui.Offset, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}
