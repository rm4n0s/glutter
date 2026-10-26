package svg

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// maxDashes is how many dashes one stroke may be cut into; a pattern that would make more (a tiny dash on a long
// path) draws the stroke solid.
const maxDashes = 200_000

// drawShape fills and strokes a shape in the order paint-order says. alpha is the opacity that the shape's parents
// handed down.
//
// SVG 1.1: painting.html#Introduction
func (p *painter) drawShape(n *Node, alpha float64) {
	st := n.Style
	if !st.Visible || n.Path == nil || n.Path.IsEmpty() {
		return
	}
	var done [3]bool
	for _, layer := range st.PaintOrder {
		if int(layer) >= len(done) || done[layer] {
			continue
		}
		done[layer] = true
		switch layer {
		case PaintLayerFill:
			p.fillShape(n, alpha)
		case PaintLayerStroke:
			p.strokeShape(n, alpha)
		case PaintLayerMarkers:
			// 29.3: markers.
		}
	}
}

// fillShape fills the geometry of a shape with its fill.
func (p *painter) fillShape(n *Node, alpha float64) {
	st := n.Style
	if st.Fill.Kind == PaintKindNone {
		return
	}
	paint := ui.NewPaint()
	paint.SetStyle(ui.PaintingStyleFill)
	paint.SetIsAntiAlias(st.ShapeRendering != ShapeRenderingCrispEdges)
	if !p.setPaint(paint, st.Fill, st.FillOpacity*alpha, n, Identity) {
		return
	}
	p.c.DrawPath(n.Path, paint)
}

// strokeShape strokes the geometry of a shape: dashed if stroke-dasharray says so, and with a width in device pixels
// under vector-effect: non-scaling-stroke.
func (p *painter) strokeShape(n *Node, alpha float64) {
	st := n.Style
	if st.Stroke.Kind == PaintKindNone || !(st.StrokeWidth > 0) {
		return
	}
	path := n.Path
	paint := ui.NewPaint()
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetIsAntiAlias(st.ShapeRendering != ShapeRenderingCrispEdges)
	paint.SetStrokeWidth(st.StrokeWidth)
	paint.SetStrokeCap(strokeCap(st.LineCap))
	paint.SetStrokeJoin(strokeJoin(st.LineJoin))
	if st.MiterLimit >= 1 {
		paint.SetStrokeMiterLimit(st.MiterLimit)
	}
	toLocal := Identity
	if n.VectorEffect == VectorEffectNonScalingStroke {
		// The stroke is made in device pixels: the canvas goes to the device's space, the path with it, and the paint
		// servers follow.
		toDevice := Scale(p.scale, p.scale).Mul(p.ctm)
		inv, ok := p.ctm.Invert()
		if !ok {
			return
		}
		p.save()
		defer p.restore()
		p.concat(inv.Mul(Scale(1/p.scale, 1/p.scale)))
		path = path.Transform(toDevice.Matrix4())
		toLocal = toDevice
	}
	// The dashes are cut from the path in the space where the stroke is made: lengths of the user space, or device
	// pixels under non-scaling-stroke.
	if len(st.DashArray) > 0 {
		path = dashPath(path, st.DashArray, st.DashOffset)
	}
	if !p.setPaint(paint, st.Stroke, st.StrokeOpacity*alpha, n, toLocal) {
		return
	}
	p.c.DrawPath(path, paint)
}

func strokeCap(c LineCap) ui.StrokeCap {
	switch c {
	case LineCapRound:
		return ui.StrokeCapRound
	case LineCapSquare:
		return ui.StrokeCapSquare
	}
	return ui.StrokeCapButt
}

func strokeJoin(j LineJoin) ui.StrokeJoin {
	switch j {
	case LineJoinRound:
		return ui.StrokeJoinRound
	case LineJoinBevel:
		return ui.StrokeJoinBevel
	}
	return ui.StrokeJoinMiter
}

// setPaint sets the color or the shader of a paint from an SVG paint, with the opacity. n is the element that is
// painted (its bounding box gives the units of the paint servers) and toLocal maps its user space to the canvas's.
// It reports whether anything is to be drawn.
//
// SVG 1.1: painting.html#SpecifyingPaint
func (p *painter) setPaint(paint *ui.Paint, pt Paint, opacity float64, n *Node, toLocal Affine) bool {
	switch pt.Kind {
	case PaintKindColor:
		paint.SetColor(withOpacity(pt.Color, opacity))
		return true
	case PaintKindServer:
		bbox, hasBBox := objectBBox(n)
		switch s := pt.Server.(type) {
		case *Gradient:
			shader, color, ok := p.gradientShader(s, bbox, hasBBox, toLocal)
			if !ok {
				return false
			}
			if shader == nil {
				paint.SetColor(withOpacity(color, opacity))
				return true
			}
			paint.SetShader(shader)
			paint.SetColor(ui.NewColorFromARGB(alpha8(opacity), 0, 0, 0))
			return true
		case *Pattern:
			shader, ok := p.patternShader(s, bbox, hasBBox, toLocal)
			if !ok {
				return false
			}
			paint.SetShader(shader)
			paint.SetColor(ui.NewColorFromARGB(alpha8(opacity), 0, 0, 0))
			return true
		}
	}
	return false
}

// withOpacity multiplies the alpha of a color by an opacity.
func withOpacity(c ui.Color, opacity float64) ui.Color {
	if opacity >= 1 {
		return c
	}
	return ui.NewColorFromARGB(int(math.Round(float64(c.Alpha())*math.Max(0, opacity))), c.Red(), c.Green(), c.Blue())
}

// gradientShader makes the shader of a gradient for a shape with the bounding box. If the gradient is one color (a
// single stop, or a degenerate vector or radius), the shader is nil and the color is what to paint. It reports
// false when nothing is to be painted: objectBoundingBox units with a box that has no area.
//
// SVG 1.1: pservers.html#Gradients
func (p *painter) gradientShader(g *Gradient, bbox ui.Rect, hasBBox bool, toLocal Affine) (ui.Shader, ui.Color, bool) {
	if len(g.Stops) == 0 {
		return nil, ui.Color{}, false
	}
	last := g.Stops[len(g.Stops)-1].Color
	if len(g.Stops) == 1 {
		return nil, last, true
	}
	m := toLocal
	// refs are what a percentage refers to: 1 for a fraction of the box, the viewport otherwise.
	refX, refY, refR := 1.0, 1.0, 1.0
	if g.Units == UnitsObjectBoundingBox {
		if !hasBBox {
			return nil, ui.Color{}, false
		}
		m = m.Mul(Affine{A: bbox.Width(), D: bbox.Height(), E: bbox.Left, F: bbox.Top})
	} else {
		refX, refY = p.vw, p.vh
		refR = math.Sqrt((p.vw*p.vw + p.vh*p.vh) / 2)
	}
	m = m.Mul(g.Transform)
	if _, ok := m.Invert(); !ok {
		return nil, last, true
	}
	colors := make([]ui.Color, len(g.Stops))
	offsets := make([]float64, len(g.Stops))
	for i, s := range g.Stops {
		colors[i], offsets[i] = s.Color, s.Offset
	}
	tile := ui.TileModeClamp
	switch g.Spread {
	case SpreadReflect:
		tile = ui.TileModeMirror
	case SpreadRepeat:
		tile = ui.TileModeRepeated
	}
	if !g.Radial {
		x1, y1 := g.X1.Resolve(refX), g.Y1.Resolve(refY)
		x2, y2 := g.X2.Resolve(refX), g.Y2.Resolve(refY)
		if x1 == x2 && y1 == y2 {
			return nil, last, true
		}
		return ui.NewGradientLinear(ui.Offset{Dx: x1, Dy: y1}, ui.Offset{Dx: x2, Dy: y2}, colors,
			ui.GradientLinearOpts{ColorStops: offsets, TileMode: tile, Matrix4: m.Matrix4()}), ui.Color{}, true
	}
	cx, cy, r := g.CX.Resolve(refX), g.CY.Resolve(refY), g.R.Resolve(refR)
	fx, fy, fr := g.FX.Resolve(refX), g.FY.Resolve(refY), g.FR.Resolve(refR)
	if !(r > 0) {
		return nil, last, true
	}
	opts := ui.GradientRadialOpts{ColorStops: offsets, TileMode: tile, Matrix4: m.Matrix4()}
	if fx != cx || fy != cy || fr > 0 {
		opts.Focal, opts.FocalRadius = &ui.Offset{Dx: fx, Dy: fy}, fr
	}
	return ui.NewGradientRadial(ui.Offset{Dx: cx, Dy: cy}, r, colors, opts), ui.Color{}, true
}

// dashPath cuts a path into the dashes of stroke-dasharray, starting offset along each contour (a negative offset
// starts the pattern later). The dashes are subpaths of their own, with the caps of the stroke at both ends; a dash of
// length 0 is a point that round and square caps make visible. A pattern that would make more than maxDashes dashes
// leaves the path as it is.
//
// SVG 1.1: painting.html#StrokeDasharrayProperty
func dashPath(src *ui.Path, dash []float64, offset float64) *ui.Path {
	total := 0.0
	for _, d := range dash {
		if d < 0 || math.IsNaN(d) || math.IsInf(d, 0) {
			return src
		}
		total += d
	}
	if !(total > 0) || math.IsNaN(offset) || math.IsInf(offset, 0) {
		return src
	}
	metrics := src.ComputeMetrics()
	length := 0.0
	for _, m := range metrics {
		length += m.Length
	}
	if length/total*float64(len(dash)) > maxDashes {
		return src
	}
	out := ui.NewPath()
	for _, m := range metrics {
		// Walk the pattern to the offset: idx is the dash that the contour starts in and rest what is left of it.
		off := math.Mod(offset, total)
		if off < 0 {
			off += total
		}
		idx, rest := 0, dash[0]
		for off > 0 {
			if off >= rest {
				off -= rest
				idx = (idx + 1) % len(dash)
				rest = dash[idx]
			} else {
				rest -= off
				off = 0
			}
		}
		startIdx, startRest := idx, rest
		// The dashes of this contour. On a closed contour the dash that starts at the start of the contour and the one that
		// ends at its end are one dash, joined at the corner, as cairo and Skia do.
		var segs []*ui.Path
		endsAtEnd := false
		for pos := 0.0; pos < m.Length; {
			end := math.Min(pos+rest, m.Length)
			if idx%2 == 0 {
				segs = append(segs, dropTinyLines(m.ExtractPath(pos, end)))
				endsAtEnd = end >= m.Length
			} else {
				endsAtEnd = false
			}
			pos += rest
			idx = (idx + 1) % len(dash)
			rest = dash[idx]
		}
		startsOn := startIdx%2 == 0 && startRest > 0
		if m.IsClosed && startsOn && endsAtEnd && len(segs) > 1 {
			last := segs[len(segs)-1]
			last.ExtendWithPath(segs[0], ui.Offset{})
			segs = append(segs[1:len(segs)-1], last)
		}
		for _, sg := range segs {
			out.AddPath(sg, ui.Offset{})
		}
	}
	return out
}

// tinyLine is the length under which a line is a rounding error.
const tinyLine = 1e-6

// dropTinyLines removes the lines of no length from a piece of a path, unless that is all there is (a dash of length
// 0). A dash that ends where the path turns would otherwise end with a line of length 1e-13 in the direction of the
// next side, and the join between the two would stick out.
func dropTinyLines(p *ui.Path) *ui.Path {
	segs := p.Segments()
	tiny := func(s ui.PathSegment) bool {
		return s.Verb == ui.PathVerbLine && math.Hypot(s.P1.Dx-s.P0.Dx, s.P1.Dy-s.P0.Dy) < tinyLine
	}
	real := 0
	for _, s := range segs {
		if s.Verb != ui.PathVerbMove && !tiny(s) {
			real++
		}
	}
	if real == 0 || real == len(segs)-1 {
		return p
	}
	out := ui.NewPath()
	for _, s := range segs {
		switch s.Verb {
		case ui.PathVerbMove:
			out.MoveTo(s.P0.Dx, s.P0.Dy)
		case ui.PathVerbLine:
			if !tiny(s) {
				out.LineTo(s.P1.Dx, s.P1.Dy)
			}
		case ui.PathVerbQuad:
			out.QuadraticBezierTo(s.P1.Dx, s.P1.Dy, s.P2.Dx, s.P2.Dy)
		case ui.PathVerbConic:
			out.ConicTo(s.P1.Dx, s.P1.Dy, s.P2.Dx, s.P2.Dy, s.Weight)
		case ui.PathVerbCubic:
			out.CubicTo(s.P1.Dx, s.P1.Dy, s.P2.Dx, s.P2.Dy, s.P3.Dx, s.P3.Dy)
		case ui.PathVerbClose:
			out.Close()
		}
	}
	return out
}
