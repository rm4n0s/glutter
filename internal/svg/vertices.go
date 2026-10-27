package svg

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// Vertex is a vertex of a shape, where a marker may be drawn: its position and the directions of the path that arrives
// at it and leaves it.
//
// SVG 1.1: painting.html#MarkerOrientAttribute
type Vertex struct {
	X, Y float64
	// In is the direction of the path arriving at the vertex and Out the direction leaving it; HasIn and HasOut say
	// whether there is one (the first vertex has no In, the last no Out, and a segment of no length has none).
	In, Out       [2]float64
	HasIn, HasOut bool
}

// vertexCollector gathers the vertices of path data command by command, so that an arc (which the path stores as
// several conics) is one segment.
type vertexCollector struct {
	verts []Vertex
	// first is the index of the first vertex of the current subpath, or -1 before the first moveto.
	first int
	// cx and cy are the current point.
	cx, cy float64
}

func newVertexCollector() *vertexCollector { return &vertexCollector{first: -1} }

// maxVertices bounds the vertices that one shape keeps: a path with more draws no more markers than that.
const maxVertices = 100_000

func (c *vertexCollector) move(x, y float64) {
	if len(c.verts) >= maxVertices {
		return
	}
	c.first = len(c.verts)
	c.verts = append(c.verts, Vertex{X: x, Y: y})
	c.cx, c.cy = x, y
}

// segment adds a segment that goes from the current point to (x, y), leaving in direction d0 and arriving in
// direction d1.
func (c *vertexCollector) segment(x, y float64, d0, d1 [2]float64) {
	if c.first < 0 || len(c.verts) >= maxVertices {
		c.cx, c.cy = x, y
		return
	}
	prev := &c.verts[len(c.verts)-1]
	if !prev.HasOut && nonZero(d0) {
		prev.Out, prev.HasOut = d0, true
	}
	v := Vertex{X: x, Y: y}
	if nonZero(d1) {
		v.In, v.HasIn = d1, true
	}
	c.verts = append(c.verts, v)
	c.cx, c.cy = x, y
}

// closePath closes the current subpath: a segment back to its start, whose vertex has the first segment's direction
// going out and the closing one coming in at the first vertex too.
func (c *vertexCollector) closePath() {
	if c.first < 0 || len(c.verts) >= maxVertices {
		return
	}
	start := c.verts[c.first]
	d := [2]float64{start.X - c.cx, start.Y - c.cy}
	if !nonZero(d) {
		last := c.verts[len(c.verts)-1]
		d = last.In
	}
	c.segment(start.X, start.Y, d, d)
	end := &c.verts[len(c.verts)-1]
	if first := &c.verts[c.first]; first.HasOut {
		end.Out, end.HasOut = first.Out, true
	}
	if nonZero(d) {
		f := &c.verts[c.first]
		f.In, f.HasIn = d, true
	}
	// A command after a closepath that is not a moveto starts at the start of the closed subpath.
	c.cx, c.cy = start.X, start.Y
	c.first = len(c.verts) - 1
}

func nonZero(d [2]float64) bool { return d[0] != 0 || d[1] != 0 }

func dirOf(x0, y0, x1, y1 float64) [2]float64 { return [2]float64{x1 - x0, y1 - y0} }

// firstNonZero returns the first of the directions that has a length.
func firstNonZero(ds ...[2]float64) [2]float64 {
	for _, d := range ds {
		if nonZero(d) {
			return d
		}
	}
	return [2]float64{}
}

// line adds a line segment to (x, y).
func (c *vertexCollector) line(x, y float64) {
	d := dirOf(c.cx, c.cy, x, y)
	c.segment(x, y, d, d)
}

// cubic adds a cubic segment.
func (c *vertexCollector) cubic(x1, y1, x2, y2, x3, y3 float64) {
	d0 := firstNonZero(dirOf(c.cx, c.cy, x1, y1), dirOf(c.cx, c.cy, x2, y2), dirOf(c.cx, c.cy, x3, y3))
	d1 := firstNonZero(dirOf(x2, y2, x3, y3), dirOf(x1, y1, x3, y3), dirOf(c.cx, c.cy, x3, y3))
	c.segment(x3, y3, d0, d1)
}

// quad adds a quadratic segment.
func (c *vertexCollector) quad(x1, y1, x2, y2 float64) {
	d0 := firstNonZero(dirOf(c.cx, c.cy, x1, y1), dirOf(c.cx, c.cy, x2, y2))
	d1 := firstNonZero(dirOf(x1, y1, x2, y2), dirOf(c.cx, c.cy, x2, y2))
	c.segment(x2, y2, d0, d1)
}

// arc adds an arc segment to (x, y), with the directions of the arc's ends, which are read off the path that the arc
// makes.
func (c *vertexCollector) arc(x, y float64, opts ui.ArcToPointOpts) {
	tmp := ui.NewPath()
	tmp.MoveTo(c.cx, c.cy)
	tmp.ArcToPoint(ui.Offset{Dx: x, Dy: y}, opts)
	segs := tmp.Segments()
	var d0, d1 [2]float64
	if len(segs) > 1 {
		d0, d1 = segmentStartDir(segs[1]), segmentEndDir(segs[len(segs)-1])
	}
	c.segment(x, y, d0, d1)
}

func segmentStartDir(s ui.PathSegment) [2]float64 {
	switch s.Verb {
	case ui.PathVerbLine:
		return dirOf(s.P0.Dx, s.P0.Dy, s.P1.Dx, s.P1.Dy)
	case ui.PathVerbQuad, ui.PathVerbConic:
		return firstNonZero(dirOf(s.P0.Dx, s.P0.Dy, s.P1.Dx, s.P1.Dy), dirOf(s.P0.Dx, s.P0.Dy, s.P2.Dx, s.P2.Dy))
	case ui.PathVerbCubic:
		return firstNonZero(dirOf(s.P0.Dx, s.P0.Dy, s.P1.Dx, s.P1.Dy), dirOf(s.P0.Dx, s.P0.Dy, s.P2.Dx, s.P2.Dy),
			dirOf(s.P0.Dx, s.P0.Dy, s.P3.Dx, s.P3.Dy))
	}
	return [2]float64{}
}

func segmentEndDir(s ui.PathSegment) [2]float64 {
	switch s.Verb {
	case ui.PathVerbLine:
		return dirOf(s.P0.Dx, s.P0.Dy, s.P1.Dx, s.P1.Dy)
	case ui.PathVerbQuad, ui.PathVerbConic:
		return firstNonZero(dirOf(s.P1.Dx, s.P1.Dy, s.P2.Dx, s.P2.Dy), dirOf(s.P0.Dx, s.P0.Dy, s.P2.Dx, s.P2.Dy))
	case ui.PathVerbCubic:
		return firstNonZero(dirOf(s.P2.Dx, s.P2.Dy, s.P3.Dx, s.P3.Dy), dirOf(s.P1.Dx, s.P1.Dy, s.P3.Dx, s.P3.Dy),
			dirOf(s.P0.Dx, s.P0.Dy, s.P3.Dx, s.P3.Dy))
	}
	return [2]float64{}
}

// verticesOfPath reads the vertices off a path that has only lines, as the line, polyline and polygon elements make.
func verticesOfPath(p *ui.Path) []Vertex {
	c := newVertexCollector()
	for _, s := range p.Segments() {
		switch s.Verb {
		case ui.PathVerbMove:
			c.move(s.P0.Dx, s.P0.Dy)
		case ui.PathVerbLine:
			c.line(s.P1.Dx, s.P1.Dy)
		case ui.PathVerbClose:
			c.closePath()
		}
	}
	return c.verts
}

// bisect returns the angle in degrees of a marker at a vertex with orient="auto": the direction of the path through
// the vertex, which at a corner is the bisector of the directions in and out.
//
// SVG 1.1: painting.html#OrientAttribute
func (v Vertex) bisect() float64 {
	ang := func(d [2]float64) float64 { return math.Atan2(d[1], d[0]) }
	switch {
	case v.HasIn && v.HasOut:
		a0, a1 := ang(v.In), ang(v.Out)
		d := a1 - a0
		for d > math.Pi {
			d -= 2 * math.Pi
		}
		for d < -math.Pi {
			d += 2 * math.Pi
		}
		return (a0 + d/2) * 180 / math.Pi
	case v.HasIn:
		return ang(v.In) * 180 / math.Pi
	case v.HasOut:
		return ang(v.Out) * 180 / math.Pi
	}
	return 0
}
