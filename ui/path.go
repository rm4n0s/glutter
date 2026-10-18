package ui

import (
	"math"
)

// PathFillType is how the interior of a Path is decided.
//
// Flutter: ui/painting.dart PathFillType
type PathFillType int

// The values of PathFillType follow Flutter's declaration order.
const (
	// PathFillTypeUnspecified is the zero value: PathFillTypeNonZero.
	//
	// Flutter: no counterpart
	PathFillTypeUnspecified PathFillType = iota
	// PathFillTypeNonZero is Flutter's PathFillType.nonZero. A point is inside
	// the path if a ray from it crosses the path a different number of times
	// going one way than the other. This is the default.
	//
	// Flutter: ui/painting.dart PathFillType.nonZero
	PathFillTypeNonZero
	// PathFillTypeEvenOdd is Flutter's PathFillType.evenOdd. A point is inside
	// the path if a ray from it crosses the path an odd number of times.
	//
	// Flutter: ui/painting.dart PathFillType.evenOdd
	PathFillTypeEvenOdd
)

var pathFillTypeNames = [...]string{"unspecified", "nonZero", "evenOdd"}

// String returns the name Dart prints for the value, such as
// "PathFillType.nonZero".
func (e PathFillType) String() string {
	return dartEnumName(pathFillTypeNames[:], int(e), "PathFillType")
}

// PathVerb is the kind of one segment of a Path.
//
// Flutter: skia SkPath::Verb
type PathVerb uint8

// The verbs of a Path.
const (
	// PathVerbMove starts a new contour at a point.
	//
	// Flutter: skia SkPath::kMove_Verb
	PathVerbMove PathVerb = iota
	// PathVerbLine is a straight line to a point.
	//
	// Flutter: skia SkPath::kLine_Verb
	PathVerbLine
	// PathVerbQuad is a quadratic Bézier curve with one control point.
	//
	// Flutter: skia SkPath::kQuad_Verb
	PathVerbQuad
	// PathVerbConic is a conic section (a rational quadratic) with one control
	// point and a weight.
	//
	// Flutter: skia SkPath::kConic_Verb
	PathVerbConic
	// PathVerbCubic is a cubic Bézier curve with two control points.
	//
	// Flutter: skia SkPath::kCubic_Verb
	PathVerbCubic
	// PathVerbClose closes the contour with a line back to its start.
	//
	// Flutter: skia SkPath::kClose_Verb
	PathVerbClose
)

// pointsPerVerb is how many points each verb adds to a path.
var pointsPerVerb = [...]int{PathVerbMove: 1, PathVerbLine: 1, PathVerbQuad: 2, PathVerbConic: 2, PathVerbCubic: 3, PathVerbClose: 0}

// Path is a complex two-dimensional shape: any number of contours, each made of
// straight lines, quadratic and cubic Bézier curves, and conic sections.
// Contours can be open or closed. It follows Skia's SkPath, which is what Flutter
// wraps, including the way a segment added after a close starts a new contour
// at the start of the closed one.
//
// Unlike most of package ui, a Path is mutable: the methods that build it change
// it. NewPathFrom copies one.
//
// Deviation: the points are float64, where Skia keeps float32.
//
// Flutter: ui/painting.dart Path
type Path struct {
	verbs    []PathVerb
	points   []Offset
	weights  []float64
	fillType PathFillType
	// lastMoveTo is the index in points of the point of the last moveTo, or its
	// bitwise complement (negative) when the next segment must start with a
	// moveTo of its own: at the start, and after a close. As in Skia.
	lastMoveTo int
}

// NewPath creates an empty path.
//
// Flutter: ui/painting.dart Path
func NewPath() *Path { return &Path{lastMoveTo: ^0} }

// NewPathFrom creates a copy of source.
//
// Flutter: ui/painting.dart Path.from
func NewPathFrom(source *Path) *Path {
	return &Path{
		verbs:      append([]PathVerb(nil), source.verbs...),
		points:     append([]Offset(nil), source.points...),
		weights:    append([]float64(nil), source.weights...),
		fillType:   source.fillType,
		lastMoveTo: source.lastMoveTo,
	}
}

// FillType is how the interior of the path is decided. It defaults to
// PathFillTypeNonZero.
//
// Flutter: ui/painting.dart Path.fillType
func (p *Path) FillType() PathFillType {
	if p.fillType == PathFillTypeUnspecified {
		return PathFillTypeNonZero
	}
	return p.fillType
}

// SetFillType sets how the interior of the path is decided.
//
// Flutter: ui/painting.dart Path.fillType
func (p *Path) SetFillType(t PathFillType) {
	if t == PathFillTypeNonZero {
		t = PathFillTypeUnspecified
	}
	p.fillType = t
}

// IsEmpty reports whether the path has no segments at all.
//
// Flutter: skia SkPath::isEmpty
func (p *Path) IsEmpty() bool { return len(p.verbs) == 0 }

// lastPt is the last point of the path, or the origin when there is none.
func (p *Path) lastPt() Offset {
	if len(p.points) == 0 {
		return Offset{}
	}
	return p.points[len(p.points)-1]
}

// injectMoveToIfNeeded starts a contour at the start of the last one if the last
// segment closed it, or at the origin if there was none.
func (p *Path) injectMoveToIfNeeded() {
	if p.lastMoveTo < 0 {
		var at Offset
		if len(p.points) != 0 {
			at = p.points[^p.lastMoveTo]
		}
		p.MoveTo(at.Dx, at.Dy)
	}
}

// MoveTo starts a new sub-path at the given coordinate.
//
// Flutter: ui/painting.dart Path.moveTo
func (p *Path) MoveTo(x, y float64) {
	p.lastMoveTo = len(p.points)
	p.verbs = append(p.verbs, PathVerbMove)
	p.points = append(p.points, Offset{x, y})
}

// RelativeMoveTo starts a new sub-path at the given offset from the last point.
//
// Flutter: ui/painting.dart Path.relativeMoveTo
func (p *Path) RelativeMoveTo(dx, dy float64) {
	last := p.lastPt()
	p.MoveTo(last.Dx+dx, last.Dy+dy)
}

// LineTo adds a straight line segment from the current point to the given point.
//
// Flutter: ui/painting.dart Path.lineTo
func (p *Path) LineTo(x, y float64) {
	p.injectMoveToIfNeeded()
	p.verbs = append(p.verbs, PathVerbLine)
	p.points = append(p.points, Offset{x, y})
}

// RelativeLineTo adds a straight line segment from the current point to the
// point at the given offset from it.
//
// Flutter: ui/painting.dart Path.relativeLineTo
func (p *Path) RelativeLineTo(dx, dy float64) {
	p.injectMoveToIfNeeded()
	last := p.lastPt()
	p.LineTo(last.Dx+dx, last.Dy+dy)
}

// QuadraticBezierTo adds a quadratic Bézier segment that curves from the
// current point to the given point (x2, y2), using the control point (x1, y1).
//
// Flutter: ui/painting.dart Path.quadraticBezierTo
func (p *Path) QuadraticBezierTo(x1, y1, x2, y2 float64) {
	p.injectMoveToIfNeeded()
	p.verbs = append(p.verbs, PathVerbQuad)
	p.points = append(p.points, Offset{x1, y1}, Offset{x2, y2})
}

// RelativeQuadraticBezierTo is QuadraticBezierTo with the points given as
// offsets from the current point.
//
// Flutter: ui/painting.dart Path.relativeQuadraticBezierTo
func (p *Path) RelativeQuadraticBezierTo(x1, y1, x2, y2 float64) {
	p.injectMoveToIfNeeded()
	last := p.lastPt()
	p.QuadraticBezierTo(last.Dx+x1, last.Dy+y1, last.Dx+x2, last.Dy+y2)
}

// CubicTo adds a cubic Bézier segment that curves from the current point to the
// given point (x3, y3), using the control points (x1, y1) and (x2, y2).
//
// Flutter: ui/painting.dart Path.cubicTo
func (p *Path) CubicTo(x1, y1, x2, y2, x3, y3 float64) {
	p.injectMoveToIfNeeded()
	p.verbs = append(p.verbs, PathVerbCubic)
	p.points = append(p.points, Offset{x1, y1}, Offset{x2, y2}, Offset{x3, y3})
}

// RelativeCubicTo is CubicTo with the points given as offsets from the current
// point.
//
// Flutter: ui/painting.dart Path.relativeCubicTo
func (p *Path) RelativeCubicTo(x1, y1, x2, y2, x3, y3 float64) {
	p.injectMoveToIfNeeded()
	last := p.lastPt()
	p.CubicTo(last.Dx+x1, last.Dy+y1, last.Dx+x2, last.Dy+y2, last.Dx+x3, last.Dy+y3)
}

// ConicTo adds a conic section that curves from the current point to the given
// point (x2, y2), using the control point (x1, y1) and the weight w. If the
// weight is greater than 1 the curve is a hyperbola, if it is 1 a parabola, and
// if it is less than 1 an ellipse. A weight that is not positive makes a line to
// (x2, y2), and one that is infinite makes two lines through (x1, y1).
//
// Flutter: ui/painting.dart Path.conicTo
func (p *Path) ConicTo(x1, y1, x2, y2, w float64) {
	switch {
	case !(w > 0):
		p.LineTo(x2, y2)
	case math.IsInf(w, 1) || math.IsNaN(w):
		p.LineTo(x1, y1)
		p.LineTo(x2, y2)
	case w == 1:
		p.QuadraticBezierTo(x1, y1, x2, y2)
	default:
		p.injectMoveToIfNeeded()
		p.verbs = append(p.verbs, PathVerbConic)
		p.points = append(p.points, Offset{x1, y1}, Offset{x2, y2})
		p.weights = append(p.weights, w)
	}
}

// RelativeConicTo is ConicTo with the points given as offsets from the current
// point.
//
// Flutter: ui/painting.dart Path.relativeConicTo
func (p *Path) RelativeConicTo(x1, y1, x2, y2, w float64) {
	p.injectMoveToIfNeeded()
	last := p.lastPt()
	p.ConicTo(last.Dx+x1, last.Dy+y1, last.Dx+x2, last.Dy+y2, w)
}

// Close closes the last sub-path, as if a straight line had been drawn from the
// current point to the first point of the sub-path.
//
// Flutter: ui/painting.dart Path.close
func (p *Path) Close() {
	if n := len(p.verbs); n > 0 && p.verbs[n-1] != PathVerbClose {
		p.verbs = append(p.verbs, PathVerbClose)
	}
	// Signal that a segment that follows needs a moveTo of its own.
	if p.lastMoveTo >= 0 {
		p.lastMoveTo = ^p.lastMoveTo
	}
}

// Reset clears the path, removing every sub-path. The fill type is kept.
//
// Flutter: ui/painting.dart Path.reset
func (p *Path) Reset() {
	p.verbs = p.verbs[:0]
	p.points = p.points[:0]
	p.weights = p.weights[:0]
	p.lastMoveTo = ^0
}

// AddRect adds a new sub-path that consists of four lines that outline the given
// rectangle, clockwise from its top left corner.
//
// Flutter: ui/painting.dart Path.addRect
func (p *Path) AddRect(rect Rect) { p.addRectDir(rect, true, 0) }

// rectCorner is corner i of the rectangle: 0 top left, 1 top right, 2 bottom
// right, 3 bottom left.
func rectCorner(r Rect, i int) Offset {
	switch i & 3 {
	case 0:
		return Offset{r.Left, r.Top}
	case 1:
		return Offset{r.Right, r.Top}
	case 2:
		return Offset{r.Right, r.Bottom}
	}
	return Offset{r.Left, r.Bottom}
}

// stepIndex moves a corner or point index one step in a direction, wrapping at n.
func stepIndex(i, n int, cw bool) int {
	if cw {
		return (i + 1) % n
	}
	return (i + n - 1) % n
}

// addRectDir adds the rectangle starting at the given corner, in the direction.
//
// Flutter: skia SkPath::addRect
func (p *Path) addRectDir(rect Rect, cw bool, start int) {
	i := start & 3
	at := rectCorner(rect, i)
	p.MoveTo(at.Dx, at.Dy)
	for k := 0; k < 3; k++ {
		i = stepIndex(i, 4, cw)
		at = rectCorner(rect, i)
		p.LineTo(at.Dx, at.Dy)
	}
	p.Close()
}

// ovalPoint is point i of the oval: 0 top center, 1 right center, 2 bottom
// center, 3 left center.
func ovalPoint(r Rect, i int) Offset {
	cx, cy := r.Left+r.Width()/2, r.Top+r.Height()/2
	switch i & 3 {
	case 0:
		return Offset{cx, r.Top}
	case 1:
		return Offset{r.Right, cy}
	case 2:
		return Offset{cx, r.Bottom}
	}
	return Offset{r.Left, cy}
}

// AddOval adds a new sub-path that consists of a curve that forms the ellipse
// that fills the given rectangle. The oval is drawn clockwise, starting from the
// center of its right edge, as four conics.
//
// Flutter: ui/painting.dart Path.addOval
func (p *Path) AddOval(oval Rect) { p.addOvalDir(oval, true, 1) }

// addOvalDir adds the oval starting at the given point index, in the direction.
//
// Flutter: skia SkPath::addOval
func (p *Path) addOvalDir(oval Rect, cw bool, start int) {
	const weight = math.Sqrt2 / 2
	oi := start & 3
	// The corner iterator tracks "behind" the oval points.
	ri := start & 3
	if !cw {
		ri = (ri + 1) & 3
	}
	at := ovalPoint(oval, oi)
	p.MoveTo(at.Dx, at.Dy)
	for k := 0; k < 4; k++ {
		ri = stepIndex(ri, 4, cw)
		oi = stepIndex(oi, 4, cw)
		c, e := rectCorner(oval, ri), ovalPoint(oval, oi)
		p.ConicTo(c.Dx, c.Dy, e.Dx, e.Dy, weight)
	}
	p.Close()
}

// AddPolygon adds a new sub-path with a sequence of line segments that connect
// the given points. If close is true, the last point is joined to the first one.
//
// Flutter: ui/painting.dart Path.addPolygon
func (p *Path) AddPolygon(points []Offset, close bool) {
	if len(points) == 0 {
		return
	}
	p.MoveTo(points[0].Dx, points[0].Dy)
	for _, pt := range points[1:] {
		p.LineTo(pt.Dx, pt.Dy)
	}
	if close {
		p.Close()
	}
}

// AddPathOpts are the optional arguments of AddPath and ExtendWithPath.
//
// Flutter: ui/painting.dart Path.addPath
type AddPathOpts struct {
	// Matrix4 transforms the added path before it is moved by the offset; nil
	// means none.
	Matrix4 *Matrix4
}

// AddPath adds the sub-paths of path to this path, translated by offset (after
// the optional transform).
//
// Flutter: ui/painting.dart Path.addPath
func (p *Path) AddPath(path *Path, offset Offset, opts ...AddPathOpts) {
	p.addPath(path, offset, oneOpts(opts).Matrix4, false)
}

// ExtendWithPath adds the sub-paths of path to this path, translated by offset
// (after the optional transform), like AddPath, except that the first sub-path
// is connected to the current point with a line instead of starting a new one.
//
// Flutter: ui/painting.dart Path.extendWithPath
func (p *Path) ExtendWithPath(path *Path, offset Offset, opts ...AddPathOpts) {
	p.addPath(path, offset, oneOpts(opts).Matrix4, true)
}

// addPath is skia's SkPath::addPath with a matrix, in the append or the extend
// mode.
func (p *Path) addPath(src *Path, offset Offset, matrix *Matrix4, extend bool) {
	if len(src.verbs) == 0 {
		return
	}
	if src == p {
		src = NewPathFrom(src)
	}
	m := Matrix4{}
	if matrix == nil {
		m = *NewMatrix4Identity()
	} else {
		m = *matrix
	}
	// The offset is added to the translation of the matrix, as Flutter does. When
	// there is no perspective that is the same as adding it to the mapped points,
	// which keeps the precision of a float64 offset.
	persp := m.hasPerspective()
	if persp {
		m.storage[12] += float32(offset.Dx)
		m.storage[13] += float32(offset.Dy)
	}
	mapped := src.mappedPoints(&m)
	if !persp {
		for i := range mapped {
			mapped[i] = mapped[i].Add(offset)
		}
	}
	if !extend && !m.hasPerspective() {
		// Append: the verbs and points are copied as they are.
		n := len(p.points)
		if src.lastMoveTo >= 0 {
			p.lastMoveTo = src.lastMoveTo + n
		} else {
			p.lastMoveTo = ^(^src.lastMoveTo + n)
		}
		p.verbs = append(p.verbs, src.verbs...)
		p.points = append(p.points, mapped...)
		p.weights = append(p.weights, src.weights...)
		return
	}
	pi, wi := 0, 0
	first := true
	for _, v := range src.verbs {
		switch v {
		case PathVerbMove:
			pt := mapped[pi]
			pi++
			if first && extend && len(p.verbs) != 0 {
				p.injectMoveToIfNeeded() // In case the last contour is closed
				if last := p.lastPt(); len(p.points) == 0 || last != pt {
					p.LineTo(pt.Dx, pt.Dy)
				}
			} else {
				p.MoveTo(pt.Dx, pt.Dy)
			}
		case PathVerbLine:
			p.LineTo(mapped[pi].Dx, mapped[pi].Dy)
			pi++
		case PathVerbQuad:
			p.QuadraticBezierTo(mapped[pi].Dx, mapped[pi].Dy, mapped[pi+1].Dx, mapped[pi+1].Dy)
			pi += 2
		case PathVerbConic:
			w := src.weights[wi]
			wi++
			p.conicToMapped(src, pi, mapped, w, &m)
			pi += 2
		case PathVerbCubic:
			p.CubicTo(mapped[pi].Dx, mapped[pi].Dy, mapped[pi+1].Dx, mapped[pi+1].Dy, mapped[pi+2].Dx, mapped[pi+2].Dy)
			pi += 3
		case PathVerbClose:
			p.Close()
		}
		first = false
	}
}

// conicToMapped adds the conic whose control and end points are mapped[pi] and
// mapped[pi+1]. Under a perspective transform the weight changes.
func (p *Path) conicToMapped(src *Path, pi int, mapped []Offset, w float64, m *Matrix4) {
	if !m.hasPerspective() {
		p.ConicTo(mapped[pi].Dx, mapped[pi].Dy, mapped[pi+1].Dx, mapped[pi+1].Dy, w)
		return
	}
	p.ConicTo(mapped[pi].Dx, mapped[pi].Dy, mapped[pi+1].Dx, mapped[pi+1].Dy, perspectiveConicWeight(src.points[pi-1], src.points[pi], src.points[pi+1], w, m))
}

// hasPerspective reports whether the matrix changes w.
func (m *Matrix4) hasPerspective() bool {
	return m.storage[3] != 0 || m.storage[7] != 0 || m.storage[15] != 1
}

// perspectiveConicWeight is the weight of the conic (p0, p1, p2, w) after the
// perspective transform m: the homogeneous weights of the three points scale the
// middle one.
func perspectiveConicWeight(p0, p1, p2 Offset, w float64, m *Matrix4) float64 {
	hw := func(p Offset) float64 {
		return float64(m.storage[3])*p.Dx + float64(m.storage[7])*p.Dy + float64(m.storage[15])
	}
	w0, w1, w2 := hw(p0), hw(p1), hw(p2)
	return w * w1 / math.Sqrt(w0*w2)
}

// mapPoint transforms a point of the plane by the matrix, dividing by w.
func (m *Matrix4) mapPoint(p Offset) Offset { return m.TransformPoint(p) }

// mappedPoints returns the points of the path transformed by the matrix.
func (p *Path) mappedPoints(m *Matrix4) []Offset {
	out := make([]Offset, len(p.points))
	if m.IsIdentity() {
		copy(out, p.points)
		return out
	}
	for i, pt := range p.points {
		out[i] = m.mapPoint(pt)
	}
	return out
}

// Shift returns a new path that is this path translated by the given offset.
//
// Flutter: ui/painting.dart Path.shift
func (p *Path) Shift(offset Offset) *Path {
	c := NewPathFrom(p)
	for i := range c.points {
		c.points[i] = c.points[i].Add(offset)
	}
	return c
}

// Transform returns a new path with the transformation matrix applied to this
// one. A matrix with perspective turns the conics into conics with other
// weights.
//
// Flutter: ui/painting.dart Path.transform
func (p *Path) Transform(matrix *Matrix4) *Path {
	c := NewPathFrom(p)
	if matrix.IsIdentity() {
		return c
	}
	c.points = p.mappedPoints(matrix)
	if matrix.hasPerspective() {
		pi, wi := 0, 0
		for _, v := range p.verbs {
			if v == PathVerbConic {
				c.weights[wi] = perspectiveConicWeight(p.points[pi-1], p.points[pi], p.points[pi+1], p.weights[wi], matrix)
				wi++
			}
			pi += pointsPerVerb[v]
		}
	}
	return c
}

// GetBounds computes the bounding rectangle for this path: the smallest
// rectangle that contains every point of the path, control points included. A
// path with no points, or with a point that is not finite, has the empty
// rectangle at the origin. A path containing only axis-aligned points on the
// same straight line has no area, and therefore Rect.IsEmpty is true for it:
// consider checking rect.Width() + rect.Height() > 0 instead.
//
// This is what Flutter returns (SkPath::computeFiniteBounds). GetTightBounds is
// the exact box of the curves.
//
// Flutter: ui/painting.dart Path.getBounds
func (p *Path) GetBounds() Rect {
	var acc boundsAcc
	for _, pt := range p.points {
		if !pt.IsFinite() {
			return Rect{}
		}
		acc.add(pt)
	}
	return acc.rect()
}

// GetTightBounds computes the smallest rectangle that contains the path itself:
// unlike GetBounds it ignores control points that the curves do not reach.
//
// Flutter: skia SkPath::computeTightBounds
func (p *Path) GetTightBounds() Rect {
	var acc boundsAcc
	p.ForEachSegment(func(s PathSegment) {
		switch s.Verb {
		case PathVerbMove:
			acc.add(s.P0)
		case PathVerbLine:
			acc.add(s.P0)
			acc.add(s.P1)
		case PathVerbQuad:
			acc.addQuadTight(s.P0, s.P1, s.P2)
		case PathVerbConic:
			acc.addConicTight(s.P0, s.P1, s.P2, s.Weight)
		case PathVerbCubic:
			acc.addCubicTight(s.P0, s.P1, s.P2, s.P3)
		}
	})
	for _, pt := range p.points {
		if !pt.IsFinite() {
			return Rect{}
		}
	}
	return acc.rect()
}

// PathSegment is one segment of a path, with the point it starts from. P0 is
// where the segment starts; P1 to P3 are the points it adds, as many as its verb
// needs: a move has only P0 (the point it moves to), a line P1, a quad or a conic
// P1 (control) and P2 (end), a cubic P1 and P2 (control) and P3 (end). A close
// has P0, the last point, and P1, the point the contour started at.
//
// Flutter: skia SkPath::Iter
type PathSegment struct {
	// Verb says what kind of segment this is.
	Verb PathVerb
	// P0 is the start of the segment; for a move, the point moved to.
	P0 Offset
	// P1 is the first point added: the end of a line, the control point of a
	// quad or conic, the first control point of a cubic, the start of the contour
	// for a close.
	P1 Offset
	// P2 is the end of a quad or conic, and the second control point of a cubic.
	P2 Offset
	// P3 is the end of a cubic.
	P3 Offset
	// Weight is the weight of a conic.
	Weight float64
}

// ForEachSegment calls fn for each segment of the path, in order.
//
// Flutter: skia SkPath::Iter
func (p *Path) ForEachSegment(fn func(PathSegment)) {
	var cur, start Offset
	pi, wi := 0, 0
	for _, v := range p.verbs {
		switch v {
		case PathVerbMove:
			cur = p.points[pi]
			start = cur
			pi++
			fn(PathSegment{Verb: v, P0: cur})
		case PathVerbLine:
			s := PathSegment{Verb: v, P0: cur, P1: p.points[pi]}
			cur = s.P1
			pi++
			fn(s)
		case PathVerbQuad:
			s := PathSegment{Verb: v, P0: cur, P1: p.points[pi], P2: p.points[pi+1]}
			cur = s.P2
			pi += 2
			fn(s)
		case PathVerbConic:
			s := PathSegment{Verb: v, P0: cur, P1: p.points[pi], P2: p.points[pi+1], Weight: p.weights[wi]}
			cur = s.P2
			pi += 2
			wi++
			fn(s)
		case PathVerbCubic:
			s := PathSegment{Verb: v, P0: cur, P1: p.points[pi], P2: p.points[pi+1], P3: p.points[pi+2]}
			cur = s.P3
			pi += 3
			fn(s)
		case PathVerbClose:
			fn(PathSegment{Verb: v, P0: cur, P1: start})
			cur = start
		}
	}
}

// Segments returns the segments of the path as a list.
//
// Flutter: skia SkPath::Iter
func (p *Path) Segments() []PathSegment {
	out := make([]PathSegment, 0, len(p.verbs))
	p.ForEachSegment(func(s PathSegment) { out = append(out, s) })
	return out
}

// String describes the path, as Dart's toString does.
//
// Flutter: ui/painting.dart Path.toString
func (p *Path) String() string { return "Path" }
