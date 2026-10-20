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

// monoPiece is a part of a curve in which y changes in one direction only.
type monoPiece struct {
	verb PathVerb
	p    [4]Offset
	w    float64
}

func (m *monoPiece) at(t float64) Offset {
	switch m.verb {
	case PathVerbLine:
		return lerpPt(m.p[0], m.p[1], t)
	case PathVerbQuad:
		return quadAt(m.p[0], m.p[1], m.p[2], t)
	case PathVerbConic:
		return conicAt(m.p[0], m.p[1], m.p[2], m.w, t)
	}
	return cubicAt(m.p[0], m.p[1], m.p[2], m.p[3], t)
}

func (m *monoPiece) end() Offset {
	switch m.verb {
	case PathVerbLine:
		return m.p[1]
	case PathVerbQuad, PathVerbConic:
		return m.p[2]
	}
	return m.p[3]
}

// splitMonotone cuts a segment at the points where y turns around.
func splitMonotone(s PathSegment, emit func(monoPiece)) {
	switch s.Verb {
	case PathVerbLine:
		emit(monoPiece{verb: PathVerbLine, p: [4]Offset{s.P0, s.P1}})
	case PathVerbQuad:
		q := [3]Offset{s.P0, s.P1, s.P2}
		if t, ok := quadExtrema(s.P0.Dy, s.P1.Dy, s.P2.Dy); ok {
			a, b := chopQuadAt(q, t)
			// At the turn the tangent is horizontal, so each half is monotone.
			emit(monoPiece{verb: PathVerbQuad, p: [4]Offset{a[0], a[1], a[2]}})
			emit(monoPiece{verb: PathVerbQuad, p: [4]Offset{b[0], b[1], b[2]}})
			return
		}
		emit(monoPiece{verb: PathVerbQuad, p: [4]Offset{q[0], q[1], q[2]}})
	case PathVerbConic:
		q := [3]Offset{s.P0, s.P1, s.P2}
		if t, ok := conicExtrema(s.P0.Dy, s.P1.Dy, s.P2.Dy, s.Weight); ok {
			a, wa, b, wb := chopConicAt(q, s.Weight, t)
			emit(monoPiece{verb: PathVerbConic, p: [4]Offset{a[0], a[1], a[2]}, w: wa})
			emit(monoPiece{verb: PathVerbConic, p: [4]Offset{b[0], b[1], b[2]}, w: wb})
			return
		}
		emit(monoPiece{verb: PathVerbConic, p: [4]Offset{q[0], q[1], q[2]}, w: s.Weight})
	case PathVerbCubic:
		c := [4]Offset{s.P0, s.P1, s.P2, s.P3}
		ts := cubicExtrema(s.P0.Dy, s.P1.Dy, s.P2.Dy, s.P3.Dy)
		sort.Float64s(ts)
		prev := 0.0
		for _, t := range ts {
			// c is the rest of the curve from prev; cut it where the original has t.
			local := (t - prev) / (1 - prev)
			a, b := chopCubicAt(c, local)
			emit(monoPiece{verb: PathVerbCubic, p: a})
			c, prev = b, t
		}
		emit(monoPiece{verb: PathVerbCubic, p: c})
	}
}

// crossings adds to the winding number and the crossing count the effect of a
// ray from (x, y) towards +x on the piece.
func (m *monoPiece) crossings(x, y float64, winding, count *int) {
	a, b := m.p[0], m.end()
	if (a.Dy <= y) == (b.Dy <= y) {
		return
	}
	var cx float64
	if m.verb == PathVerbLine {
		cx = a.Dx + (y-a.Dy)*(b.Dx-a.Dx)/(b.Dy-a.Dy)
	} else {
		// Find t where the curve is at y by bisection: y is monotone in t.
		lo, hi := 0.0, 1.0
		up := b.Dy > a.Dy
		for i := 0; i < 64; i++ {
			mid := (lo + hi) / 2
			if (m.at(mid).Dy < y) == up {
				lo = mid
			} else {
				hi = mid
			}
		}
		cx = m.at((lo + hi) / 2).Dx
	}
	if cx > x {
		*count++
		if b.Dy > a.Dy {
			*winding++
		} else {
			*winding--
		}
	}
}

// Contains reports whether the point is inside the path, as the fill type
// decides. Every contour counts as closed. A point on the outline is inside when
// it is on a left or top edge, as for Rect.Contains.
//
// Flutter: ui/painting.dart Path.contains
func (p *Path) Contains(point Offset) bool {
	if math.IsNaN(point.Dx) || math.IsNaN(point.Dy) {
		return false
	}
	winding, count := 0, 0
	addPiece := func(m monoPiece) { m.crossings(point.Dx, point.Dy, &winding, &count) }
	var start, cur Offset
	open := false
	closeContour := func() {
		if open && cur != start {
			splitMonotone(PathSegment{Verb: PathVerbLine, P0: cur, P1: start}, addPiece)
		}
		cur = start
	}
	p.ForEachSegment(func(s PathSegment) {
		switch s.Verb {
		case PathVerbMove:
			closeContour()
			start, cur, open = s.P0, s.P0, true
		case PathVerbClose:
			closeContour()
		default:
			splitMonotone(s, addPiece)
			switch s.Verb {
			case PathVerbLine:
				cur = s.P1
			case PathVerbQuad, PathVerbConic:
				cur = s.P2
			case PathVerbCubic:
				cur = s.P3
			}
		}
	})
	closeContour()
	if p.FillType() == PathFillTypeEvenOdd {
		return count%2 == 1
	}
	return winding != 0
}
