package ui

import (
	"math"
	"sort"
)

// PathOperation is how PathCombine combines two paths.
//
// Flutter: ui/painting.dart PathOperation
type PathOperation int

// The values of PathOperation follow Flutter's declaration order.
const (
	// PathOperationUnspecified is the zero value: PathOperationDifference.
	//
	// Flutter: no counterpart
	PathOperationUnspecified PathOperation = iota
	// PathOperationDifference subtracts the second path from the first.
	//
	// Flutter: ui/painting.dart PathOperation.difference
	PathOperationDifference
	// PathOperationIntersect keeps what the two paths have in common.
	//
	// Flutter: ui/painting.dart PathOperation.intersect
	PathOperationIntersect
	// PathOperationUnion keeps what is in either of the paths.
	//
	// Flutter: ui/painting.dart PathOperation.union
	PathOperationUnion
	// PathOperationXor keeps what is in one of the paths and not in both.
	//
	// Flutter: ui/painting.dart PathOperation.xor
	PathOperationXor
	// PathOperationReverseDifference subtracts the first path from the second.
	//
	// Flutter: ui/painting.dart PathOperation.reverseDifference
	PathOperationReverseDifference
)

var pathOperationNames = [...]string{"unspecified", "difference", "intersect", "union", "xor", "reverseDifference"}

// String returns the name Dart prints for the value, such as "PathOperation.union".
func (o PathOperation) String() string {
	return dartEnumName(pathOperationNames[:], int(o), "PathOperation")
}

// inside says whether a point is in the result, given whether it is in each path.
func (o PathOperation) inside(a, b bool) bool {
	switch o {
	case PathOperationIntersect:
		return a && b
	case PathOperationUnion:
		return a || b
	case PathOperationXor:
		return a != b
	case PathOperationReverseDifference:
		return b && !a
	}
	return a && !b // difference, and the zero value
}

// maxCombineSegments is how many straight lines the two paths may be cut into. When the
// curves need more for the tolerance, the tolerance is made coarser until they fit,
// which keeps the time of a combination bounded.
const maxCombineSegments = 1 << 15

// PathCombine applies the operation to the two paths and returns the result as a new
// path: the region that is inside the paths (each by its own fill type) the way the
// operation says. The contours of the result do not cross each other or themselves,
// holes have the opposite direction of what they are in, and the fill type is
// PathFillTypeNonZero. Open contours are taken as closed, as when a path is filled.
//
// The way it works (a deliberate choice, documented in SPEC §7): the curves of both
// paths are cut into straight lines, in steps of the same parameter, with a tolerance of
// a ten-thousandth of the size of the paths; every crossing of two lines, every touch
// and every overlap becomes a vertex (the coordinates are snapped to a grid a
// billionth of that size, and vertices that fall in neighbouring cells are welded, so that
// rounding does not leave gaps); this gives a planar graph whose edges each have a
// winding number of both paths on their two sides, found by casting a ray; an edge is on
// the border of the result when the operation says differently for its two sides; the
// edges are chained into contours. Because no assumption is made on the paths, the
// inputs may cross themselves, overlap each other, touch along an edge, or be open.
// Where the border of the result follows a stretch of one of the original curves, the
// curve is put back (cut at the crossings, with its end points on the vertices), so a
// result made of pieces of circles still has curves, not lines; only the stretches
// between two crossings of a curve with another curve are exact up to the tolerance.
//
// Deviation: Skia's pathops works with exact curve intersections and can fail (Flutter's
// Path.combine then returns an empty path, or throws); this one always returns a path.
// The vertices of the crossings of curves are found on the lines that replace them, so
// they are off the true crossing by at most the tolerance.
//
// Flutter: ui/painting.dart Path.combine
func PathCombine(op PathOperation, path1, path2 *Path) *Path {
	c := &combiner{op: op}
	return c.run(path1, path2)
}

// combineCurve is a piece of a path that was cut into lines: a line, or a curve that
// the lines replace.
type combineCurve struct {
	kind PathVerb // PathVerbLine, PathVerbQuad, PathVerbConic or PathVerbCubic
	pts  [4]Offset
	w    float64
}

// combineSeg is one straight line of a curve, going the way the path goes.
type combineSeg struct {
	a, b   Offset
	curve  int     // the index of the curve it comes from
	t0, t1 float64 // the part of the curve that the line replaces
	src    int     // 0 for the first path, 1 for the second
}

// combineVertex is a point of the graph.
type combineVertex = Offset

// combineEdge is an edge of the graph: a line between two vertices u and v (u < v), the
// net number of times each path goes along it from u to v (against the way it goes counts
// negative), and the curve it comes from.
type combineEdge struct {
	u, v   int
	d      [2]int
	curve  int
	t0, t1 float64 // the curve parameters at u and at v
	left   [2]int  // winding numbers of both paths on the left of u -> v
}

// piece is a part of a combineSeg between two vertices.
type piece struct {
	a, b   int
	curve  int
	t0, t1 float64 // curve parameters at a and at b
	src    int
}

type combiner struct {
	op   PathOperation
	fill [2]PathFillType

	curves []combineCurve
	segs   []combineSeg
	grid   float64
	tol    float64

	verts []combineVertex
	edges []combineEdge
}

// splitPoint is a place where a segment is cut: the parameter along it and the vertex.
type splitPoint struct {
	s float64
	p Offset
}

func (c *combiner) run(path1, path2 *Path) *Path {
	out := NewPath()
	if path1 == nil {
		path1 = NewPath()
	}
	if path2 == nil {
		path2 = NewPath()
	}
	c.fill = [2]PathFillType{path1.FillType(), path2.FillType()}
	bounds, ok := combineBounds(path1, path2)
	if !ok {
		return out
	}
	extent := math.Max(bounds.Right-bounds.Left, bounds.Bottom-bounds.Top)
	maxAbs := math.Max(math.Max(math.Abs(bounds.Left), math.Abs(bounds.Right)), math.Max(math.Abs(bounds.Top), math.Abs(bounds.Bottom)))
	if !(extent > 0) {
		return out
	}
	c.grid = math.Max(extent*1e-9, maxAbs*1e-13)
	c.tol = extent * 1e-4
	for attempt := 0; ; attempt++ {
		c.curves, c.segs = c.curves[:0], c.segs[:0]
		c.collect(path1, 0)
		c.collect(path2, 1)
		if len(c.segs) <= maxCombineSegments || attempt >= 6 {
			break
		}
		c.tol *= 4
	}
	if len(c.segs) > maxCombineSegments {
		c.segs = c.segs[:maxCombineSegments]
	}
	c.buildGraph()
	c.classify()
	contours := c.chain()
	c.emit(out, contours)
	return out
}

// combineBounds are the bounds of the points of both paths, and whether they are all
// finite and there is at least one segment.
func combineBounds(paths ...*Path) (Rect, bool) {
	first := true
	var r Rect
	for _, p := range paths {
		for _, pt := range p.points {
			if !pt.IsFinite() {
				return Rect{}, false
			}
			if first {
				r = Rect{pt.Dx, pt.Dy, pt.Dx, pt.Dy}
				first = false
				continue
			}
			r.Left, r.Right = math.Min(r.Left, pt.Dx), math.Max(r.Right, pt.Dx)
			r.Top, r.Bottom = math.Min(r.Top, pt.Dy), math.Max(r.Bottom, pt.Dy)
		}
	}
	return r, !first
}

func (c *combiner) snap(p Offset) Offset {
	return Offset{math.Round(p.Dx/c.grid) * c.grid, math.Round(p.Dy/c.grid) * c.grid}
}

// collect cuts the contours of a path into lines. A contour that is open is closed with a
// line.
func (c *combiner) collect(p *Path, src int) {
	var start, cur Offset
	open := false
	closeContour := func() {
		if open && cur != start {
			c.addCurve(src, combineCurve{kind: PathVerbLine, pts: [4]Offset{cur, start}})
		}
		open = false
	}
	p.ForEachSegment(func(s PathSegment) {
		switch s.Verb {
		case PathVerbMove:
			closeContour()
			start, cur, open = s.P0, s.P0, true
		case PathVerbLine:
			c.addCurve(src, combineCurve{kind: PathVerbLine, pts: [4]Offset{s.P0, s.P1}})
			cur = s.P1
		case PathVerbQuad:
			c.addCurve(src, combineCurve{kind: PathVerbQuad, pts: [4]Offset{s.P0, s.P1, s.P2}})
			cur = s.P2
		case PathVerbConic:
			c.addCurve(src, combineCurve{kind: PathVerbConic, pts: [4]Offset{s.P0, s.P1, s.P2}, w: s.Weight})
			cur = s.P2
		case PathVerbCubic:
			c.addCurve(src, combineCurve{kind: PathVerbCubic, pts: [4]Offset{s.P0, s.P1, s.P2, s.P3}})
			cur = s.P3
		case PathVerbClose:
			closeContour()
		}
	})
	closeContour()
}

// addCurve cuts a curve into lines of equal steps of the parameter and adds them.
func (c *combiner) addCurve(src int, cv combineCurve) {
	for _, pt := range cv.pts[:curvePoints(cv.kind)] {
		if !pt.IsFinite() {
			return
		}
	}
	n := 1
	switch cv.kind {
	case PathVerbQuad, PathVerbConic:
		d := cv.pts[0].Sub(cv.pts[1].Mul(2)).Add(cv.pts[2])
		n = splitCount(math.Hypot(d.Dx, d.Dy) / (4 * c.tol))
	case PathVerbCubic:
		d1 := cv.pts[0].Sub(cv.pts[1].Mul(2)).Add(cv.pts[2])
		d2 := cv.pts[1].Sub(cv.pts[2].Mul(2)).Add(cv.pts[3])
		n = splitCount(3 * math.Max(math.Hypot(d1.Dx, d1.Dy), math.Hypot(d2.Dx, d2.Dy)) / (4 * c.tol))
	}
	idx := len(c.curves)
	c.curves = append(c.curves, cv)
	prev := c.snap(cv.pts[0])
	for i := 1; i <= n; i++ {
		t1 := float64(i) / float64(n)
		var q Offset
		switch {
		case i == n:
			q = cv.pts[curvePoints(cv.kind)-1]
		case cv.kind == PathVerbQuad:
			q = quadAt(cv.pts[0], cv.pts[1], cv.pts[2], t1)
		case cv.kind == PathVerbConic:
			q = conicAt(cv.pts[0], cv.pts[1], cv.pts[2], cv.w, t1)
		default:
			q = cubicAt(cv.pts[0], cv.pts[1], cv.pts[2], cv.pts[3], t1)
		}
		q = c.snap(q)
		if q != prev {
			c.segs = append(c.segs, combineSeg{a: prev, b: q, curve: idx, t0: float64(i-1) / float64(n), t1: t1, src: src})
			prev = q
		}
	}
}

// curvePoints is how many points the curve of the verb has.
func curvePoints(kind PathVerb) int {
	switch kind {
	case PathVerbLine:
		return 2
	case PathVerbQuad, PathVerbConic:
		return 3
	}
	return 4
}

// buildGraph finds where the lines meet and makes the vertices and the edges.
func (c *combiner) buildGraph() {
	n := len(c.segs)
	splits := make([][]splitPoint, n)
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	minX := func(i int) float64 { return math.Min(c.segs[i].a.Dx, c.segs[i].b.Dx) }
	maxX := func(i int) float64 { return math.Max(c.segs[i].a.Dx, c.segs[i].b.Dx) }
	sort.Slice(order, func(x, y int) bool { return minX(order[x]) < minX(order[y]) })
	slack := 4 * c.grid
	for oi, i := range order {
		si := &c.segs[i]
		ylo, yhi := math.Min(si.a.Dy, si.b.Dy)-slack, math.Max(si.a.Dy, si.b.Dy)+slack
		xhi := maxX(i) + slack
		for oj := oi + 1; oj < n; oj++ {
			j := order[oj]
			if minX(j) > xhi {
				break
			}
			sj := &c.segs[j]
			if math.Max(sj.a.Dy, sj.b.Dy) < ylo || math.Min(sj.a.Dy, sj.b.Dy) > yhi {
				continue
			}
			c.intersect(i, j, splits)
		}
	}

	// Weld the vertices: the endpoints and the splits that are in neighbouring cells of
	// the grid become one.
	weld := newWelder(c.grid)
	vertexOf := func(p Offset) int { return weld.id(p, &c.verts) }
	pieces := make([]piece, 0, n+n/2)
	for i := range c.segs {
		s := &c.segs[i]
		pts := append([]splitPoint{{0, s.a}}, splits[i]...)
		pts = append(pts, splitPoint{1, s.b})
		sort.SliceStable(pts, func(x, y int) bool { return pts[x].s < pts[y].s })
		prev := vertexOf(pts[0].p)
		prevS := pts[0].s
		for _, sp := range pts[1:] {
			v := vertexOf(sp.p)
			if v == prev {
				continue
			}
			pieces = append(pieces, piece{a: prev, b: v, curve: s.curve,
				t0: s.t0 + (s.t1-s.t0)*prevS, t1: s.t0 + (s.t1-s.t0)*sp.s, src: s.src})
			prev, prevS = v, sp.s
		}
	}

	// Edges: pieces on the same pair of vertices are one edge.
	index := map[[2]int]int{}
	for _, pc := range pieces {
		u, v, sign := pc.a, pc.b, 1
		t0, t1 := pc.t0, pc.t1
		if u > v {
			u, v, sign = v, u, -1
			t0, t1 = t1, t0
		}
		key := [2]int{u, v}
		ei, ok := index[key]
		if !ok {
			ei = len(c.edges)
			index[key] = ei
			c.edges = append(c.edges, combineEdge{u: u, v: v, curve: pc.curve, t0: t0, t1: t1})
		}
		c.edges[ei].d[pc.src] += sign
	}
}

// intersect finds where the lines i and j meet and records the places where they have to be cut.
func (c *combiner) intersect(i, j int, splits [][]splitPoint) {
	a, b := &c.segs[i], &c.segs[j]
	p, q := a.a, a.b
	r, s := b.a, b.b
	e1, e2 := q.Sub(p), s.Sub(r)
	l1, l2 := math.Hypot(e1.Dx, e1.Dy), math.Hypot(e2.Dx, e2.Dy)
	if l1 == 0 || l2 == 0 {
		return
	}
	denom := cross(e1, e2)
	near := 4 * c.grid
	if math.Abs(denom) <= 1e-12*l1*l2 {
		// Parallel: they only meet where they overlap on one line.
		if math.Abs(cross(r.Sub(p), e1))/l1 > near {
			return
		}
		c.addOnto(i, r, splits)
		c.addOnto(i, s, splits)
		c.addOnto(j, p, splits)
		c.addOnto(j, q, splits)
		return
	}
	t := cross(r.Sub(p), e2) / denom
	u := cross(r.Sub(p), e1) / denom
	tol1, tol2 := near/l1, near/l2
	if t < -tol1 || t > 1+tol1 || u < -tol2 || u > 1+tol2 {
		return
	}
	// The vertex: an end point when the crossing is at one, else the crossing.
	var pt Offset
	switch {
	case t <= tol1:
		pt = p
	case t >= 1-tol1:
		pt = q
	case u <= tol2:
		pt = r
	case u >= 1-tol2:
		pt = s
	default:
		pt = c.snap(p.Add(e1.Mul(t)))
	}
	c.cut(i, math.Min(math.Max(t, 0), 1), pt, splits)
	c.cut(j, math.Min(math.Max(u, 0), 1), pt, splits)
}

// addOnto cuts the line i at the point, when the point is on it (an end point of a line that
// overlaps it).
func (c *combiner) addOnto(i int, pt Offset, splits [][]splitPoint) {
	s := &c.segs[i]
	e := s.b.Sub(s.a)
	l2 := e.Dx*e.Dx + e.Dy*e.Dy
	t := (pt.Dx-s.a.Dx)*e.Dx/l2 + (pt.Dy-s.a.Dy)*e.Dy/l2
	if t <= 0 || t >= 1 {
		return
	}
	c.cut(i, t, pt, splits)
}

// cut adds a place where the line i is cut. The ends of the line are vertices already.
func (c *combiner) cut(i int, s float64, pt Offset, splits [][]splitPoint) {
	seg := &c.segs[i]
	if pt == seg.a || pt == seg.b || s <= 0 || s >= 1 {
		return
	}
	splits[i] = append(splits[i], splitPoint{s, pt})
}

func cross(a, b Offset) float64 { return a.Dx*b.Dy - a.Dy*b.Dx }

// welder gives each point of a grid an ID, the same one to points in neighbouring cells.
type welder struct {
	cell  float64
	cells map[[2]int64]int
}

func newWelder(grid float64) *welder { return &welder{cell: 4 * grid, cells: map[[2]int64]int{}} }

// id returns the ID of the vertex at the point, adding it to verts when there is none
// within a cell of it.
func (w *welder) id(p Offset, verts *[]Offset) int {
	cx, cy := int64(math.Floor(p.Dx/w.cell)), int64(math.Floor(p.Dy/w.cell))
	for dx := int64(-1); dx <= 1; dx++ {
		for dy := int64(-1); dy <= 1; dy++ {
			if id, ok := w.cells[[2]int64{cx + dx, cy + dy}]; ok {
				v := (*verts)[id]
				if math.Abs(v.Dx-p.Dx) <= w.cell && math.Abs(v.Dy-p.Dy) <= w.cell {
					return id
				}
			}
		}
	}
	id := len(*verts)
	*verts = append(*verts, p)
	w.cells[[2]int64{cx, cy}] = id
	return id
}

// inside says whether a winding number is inside by the fill type.
func inside(w int, fill PathFillType) bool {
	if fill == PathFillTypeEvenOdd {
		return w%2 != 0
	}
	return w != 0
}

// classify finds the winding numbers of both paths on the left of each edge, by casting a
// ray to the right from a point just left of the middle of the edge.
func (c *combiner) classify() {
	n := len(c.edges)
	if n == 0 {
		return
	}
	// The edges by the rows they cover, so that a ray looks at the edges of its own row only.
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, v := range c.verts {
		minY, maxY = math.Min(minY, v.Dy), math.Max(maxY, v.Dy)
	}
	rows := int(math.Ceil(math.Sqrt(float64(n))))
	if rows < 1 {
		rows = 1
	}
	height := maxY - minY
	rowOf := func(y float64) int {
		if !(height > 0) {
			return 0
		}
		r := int((y - minY) / height * float64(rows))
		return min(max(r, 0), rows-1)
	}
	buckets := make([][]int32, rows)
	for i, e := range c.edges {
		y0, y1 := c.verts[e.u].Dy, c.verts[e.v].Dy
		if y0 > y1 {
			y0, y1 = y1, y0
		}
		for r := rowOf(y0); r <= rowOf(y1); r++ {
			buckets[r] = append(buckets[r], int32(i))
		}
	}
	for i := range c.edges {
		e := &c.edges[i]
		u, v := c.verts[e.u], c.verts[e.v]
		dir := v.Sub(u)
		l := math.Hypot(dir.Dx, dir.Dy)
		mid := u.Add(dir.Mul(0.5))
		delta := math.Min(2*c.grid, l/4)
		left := Offset{-dir.Dy / l, dir.Dx / l}
		pt := mid.Add(left.Mul(delta))
		var w [2]int
		for _, k := range buckets[rowOf(pt.Dy)] {
			o := &c.edges[k]
			a, b := c.verts[o.u], c.verts[o.v]
			if (a.Dy <= pt.Dy) == (b.Dy <= pt.Dy) {
				continue
			}
			x := a.Dx + (pt.Dy-a.Dy)*(b.Dx-a.Dx)/(b.Dy-a.Dy)
			if x <= pt.Dx {
				continue
			}
			sign := 1
			if b.Dy < a.Dy {
				sign = -1
			}
			w[0] += sign * o.d[0]
			w[1] += sign * o.d[1]
		}
		e.left = w
	}
}

// directedEdge is an edge of the border of the result, with the inside on its left.
type directedEdge struct {
	from, to int
	curve    int
	t0, t1   float64 // the curve parameters at from and at to
	used     bool
}

// chain picks the edges on the border of the result and joins them into contours.
func (c *combiner) chain() [][]directedEdge {
	var border []directedEdge
	for _, e := range c.edges {
		l := c.op.inside(inside(e.left[0], c.fill[0]), inside(e.left[1], c.fill[1]))
		r := c.op.inside(inside(e.left[0]-e.d[0], c.fill[0]), inside(e.left[1]-e.d[1], c.fill[1]))
		switch {
		case l == r:
		case l:
			border = append(border, directedEdge{from: e.u, to: e.v, curve: e.curve, t0: e.t0, t1: e.t1})
		default:
			border = append(border, directedEdge{from: e.v, to: e.u, curve: e.curve, t0: e.t1, t1: e.t0})
		}
	}
	out := map[int][]int{}
	for i, e := range border {
		out[e.from] = append(out[e.from], i)
	}
	// Start at the edges in a fixed order, so that the result does not depend on the map.
	var contours [][]directedEdge
	for start := range border {
		if border[start].used {
			continue
		}
		var contour []directedEdge
		cur := start
		for cur >= 0 {
			border[cur].used = true
			contour = append(contour, border[cur])
			if border[cur].to == border[start].from {
				break
			}
			cur = c.next(border, out, cur)
		}
		contours = append(contours, contour)
	}
	return contours
}

// next picks the unused edge that leaves the end of the edge cur and turns the most to the
// left (so that contours that touch do not cross), or -1.
func (c *combiner) next(border []directedEdge, out map[int][]int, cur int) int {
	e := border[cur]
	in := c.verts[e.to].Sub(c.verts[e.from])
	best, bestTurn := -1, math.Inf(-1)
	for _, k := range out[e.to] {
		if border[k].used {
			continue
		}
		o := c.verts[border[k].to].Sub(c.verts[border[k].from])
		turn := math.Atan2(cross(in, o), in.Dx*o.Dx+in.Dy*o.Dy)
		if turn > bestTurn {
			best, bestTurn = k, turn
		}
	}
	return best
}

// emit writes the contours into a path, putting back the curves.
func (c *combiner) emit(out *Path, contours [][]directedEdge) {
	for _, contour := range contours {
		if len(contour) < 2 {
			continue
		}
		// Start where one piece of curve ends and another begins, so that a curve is not cut
		// in two by the start of the contour.
		contour = c.rotate(contour)
		first := c.verts[contour[0].from]
		out.MoveTo(first.Dx, first.Dy)
		for i := 0; i < len(contour); {
			j := i + 1
			for j < len(contour) && c.joins(contour[j-1], contour[j]) {
				j++
			}
			c.emitRun(out, contour[i:j])
			i = j
		}
		out.Close()
	}
}

// joins says whether the edge b goes on along the curve that the edge a ends on.
func (c *combiner) joins(a, b directedEdge) bool {
	if a.curve != b.curve {
		return false
	}
	if c.curves[a.curve].kind == PathVerbLine {
		return true
	}
	const eps = 1e-9
	fwdA, fwdB := a.t1 > a.t0, b.t1 > b.t0
	return fwdA == fwdB && math.Abs(a.t1-b.t0) <= eps
}

// rotate moves the start of the contour to a place where two runs meet, when there is one.
func (c *combiner) rotate(contour []directedEdge) []directedEdge {
	n := len(contour)
	for k := 0; k < n; k++ {
		if !c.joins(contour[(k+n-1)%n], contour[k]) {
			if k == 0 {
				return contour
			}
			return append(append([]directedEdge{}, contour[k:]...), contour[:k]...)
		}
	}
	return contour
}

// emitRun adds a run of edges that follow one curve: a line, or the part of the curve
// between the first and the last parameter.
func (c *combiner) emitRun(out *Path, run []directedEdge) {
	cv := c.curves[run[0].curve]
	end := c.verts[run[len(run)-1].to]
	t0, t1 := run[0].t0, run[len(run)-1].t1
	if cv.kind == PathVerbLine || math.Abs(t1-t0) < 1e-9 || !(isFinite(t0) && isFinite(t1)) {
		out.LineTo(end.Dx, end.Dy)
		return
	}
	reversed := t1 < t0
	lo, hi := t0, t1
	if reversed {
		lo, hi = t1, t0
	}
	lo, hi = math.Max(lo, 0), math.Min(hi, 1)
	switch cv.kind {
	case PathVerbQuad:
		p := subQuad([3]Offset{cv.pts[0], cv.pts[1], cv.pts[2]}, lo, hi)
		if reversed {
			p[0], p[2] = p[2], p[0]
		}
		out.QuadraticBezierTo(p[1].Dx, p[1].Dy, end.Dx, end.Dy)
	case PathVerbConic:
		p, w := subConic([3]Offset{cv.pts[0], cv.pts[1], cv.pts[2]}, cv.w, lo, hi)
		if reversed {
			p[0], p[2] = p[2], p[0]
		}
		out.ConicTo(p[1].Dx, p[1].Dy, end.Dx, end.Dy, w)
	default:
		p := subCubic(cv.pts, lo, hi)
		if reversed {
			p[0], p[1], p[2], p[3] = p[3], p[2], p[1], p[0]
		}
		out.CubicTo(p[1].Dx, p[1].Dy, p[2].Dx, p[2].Dy, end.Dx, end.Dy)
	}
}
