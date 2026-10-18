package gles

import (
	"fmt"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// The shape pipeline draws rectangles, rounded rectangles, ovals, circles,
// rounded superellipses, the region between two rounded rectangles, lines and
// points, and fills with a color. Each shape is one instance of a quad whose
// fragment shader finds the coverage of every pixel from a signed distance
// field (shaders/shape.frag.glsl). There is no multisampling.

// Instance layout: twelve vec4 attributes.
const (
	shapeVec4s  = 12
	shapeFloats = shapeVec4s * 4
	shapeStride = shapeFloats * 4
)

// The kinds of shape of the shader.
const (
	kindRRect   = 0
	kindSuper   = 1
	kindSegment = 2
	kindFill    = 3
	kindBox     = 4
	kindOutline = 5
)

// The flags of the shader.
const (
	flagHasB       = 1
	flagAntialias  = 2
	flagSimilarity = 4
)

// superellipseExtent and superellipseExponent are the constants of
// ui.Path.AddRSuperellipse, which the software renderer draws: a corner box is
// this much times the radius, filled by |x|^n + |y|^n = 1.
const (
	superellipseExtent   = 1.2924
	superellipseExponent = 5
)

// blendFunc is a pair of glBlendFuncSeparate factors.
type blendFunc struct{ srcRGB, dstRGB, srcA, dstA uint32 }

// blendFuncs are the blend modes that glBlendFunc makes exactly when the shape's
// coverage scales the source. The others need the destination in the shader or
// a layer (step 3.4).
var blendFuncs = map[ui.BlendMode]blendFunc{
	ui.BlendModeSrcOver: {glOne, glOneMinusSrcAlpha, glOne, glOneMinusSrcAlpha},
	ui.BlendModeDstOver: {glOneMinusDstAlpha, glOne, glOneMinusDstAlpha, glOne},
	ui.BlendModePlus:    {glOne, glOne, glOne, glOne},
}

// batch is a run of instances drawn with the same state in one call.
type batch struct {
	first, count int
	blend        blendFunc
	scissor      [4]int // x, y, width, height from the top left of the surface
}

// shapeFrame collects the instances of a frame and the batches that draw them.
type shapeFrame struct {
	data    []float32
	batches []batch
}

func (f *shapeFrame) instances() int { return len(f.data) / shapeFloats }

// add appends one instance to the batch it belongs to.
func (f *shapeFrame) add(inst *[shapeFloats]float32, blend blendFunc, scissor [4]int) {
	n := f.instances()
	if k := len(f.batches); k > 0 && f.batches[k-1].blend == blend && f.batches[k-1].scissor == scissor {
		f.batches[k-1].count++
	} else {
		f.batches = append(f.batches, batch{first: n, count: 1, blend: blend, scissor: scissor})
	}
	f.data = append(f.data, inst[:]...)
}

// affine2D is the 2D part of a transform: x' = a*x + c*y + e, y' = b*x + d*y + f.
type affine2D struct{ a, b, c, d, e, f float64 }

func affineOf(m *ui.Matrix4) affine2D {
	s := m.Storage()
	return affine2D{float64(s[0]), float64(s[1]), float64(s[4]), float64(s[5]), float64(s[12]), float64(s[13])}
}

func (t affine2D) det() float64 { return t.a*t.d - t.b*t.c }

func (t affine2D) invert() (affine2D, bool) {
	det := t.det()
	if det == 0 || math.IsNaN(det) || math.IsInf(det, 0) {
		return affine2D{}, false
	}
	return affine2D{
		a: t.d / det, b: -t.b / det, c: -t.c / det, d: t.a / det,
		e: (t.c*t.f - t.d*t.e) / det, f: (t.b*t.e - t.a*t.f) / det,
	}, true
}

// similarity says that the transform only rotates, flips and scales evenly, so
// that a distance in the shape's units is the same number of pixels in every
// direction.
func (t affine2D) similarity() bool {
	const eps = 1e-4
	n1, n2 := math.Hypot(t.a, t.b), math.Hypot(t.c, t.d)
	return math.Abs(n1-n2) <= eps*math.Max(n1, n2) && math.Abs(t.a*t.c+t.b*t.d) <= eps*n1*n2
}

// frameBuilder turns the commands of a draw list into instances. It does not
// call GL.
type frameBuilder struct {
	w, h  int
	skip  bool // unsupported commands are skipped and counted instead of failing
	shape shapeFrame
	stats FrameStats
	// skipping is the depth of the unsupported group being skipped; 0 when none.
	skipping int
	// reasons are the unsupported features seen, to count each once.
	reasons map[string]bool
}

// FrameStats says what the last frame did.
//
// Flutter: no counterpart
type FrameStats struct {
	// Commands is the number of drawing commands in the scene.
	Commands int
	// Drawn is the number of commands drawn as shapes.
	Drawn int
	// Skipped is the number of commands left out because the renderer cannot draw
	// them yet; only a renderer with SkipUnsupported skips.
	Skipped int
	// Instances is the number of shapes sent to the GPU, Batches the number of draw
	// calls that sent them.
	Instances, Batches int
	// Unsupported lists the features that caused skipped commands, once each.
	Unsupported []string
}

// unsupported is called for a feature the renderer cannot draw. In strict mode it
// returns GLNotImplemented; when skipping it records the feature and returns nil
// and the caller leaves the command out.
func (b *frameBuilder) unsupported(what string) error {
	if !b.skip {
		return errors.New("GLNotImplemented", "the GPU renderer cannot draw this yet", "feature", what)
	}
	if !b.reasons[what] {
		if b.reasons == nil {
			b.reasons = map[string]bool{}
		}
		b.reasons[what] = true
		b.stats.Unsupported = append(b.stats.Unsupported, what)
	}
	return nil
}

// command adds one command of the draw list.
func (b *frameBuilder) command(c gpu.Command) error {
	switch c.Kind {
	case gpu.CommandBeginLayer:
		if b.skipping > 0 {
			b.skipping++
			return nil
		}
		// Groups arrive with step 3.4.
		if err := b.unsupported("layer"); err != nil {
			return err
		}
		b.skipping = 1
		return nil
	case gpu.CommandEndLayer:
		if b.skipping > 0 {
			b.skipping--
		}
		return nil
	}
	if b.skipping > 0 {
		b.stats.Commands++
		b.stats.Skipped++
		return nil
	}
	b.stats.Commands++
	drawn, err := b.draw(c)
	if err != nil {
		return err
	}
	if drawn {
		b.stats.Drawn++
	} else {
		b.stats.Skipped++
	}
	return nil
}

// draw makes the instances of a drawing command. It reports whether the command
// was drawn, or at least needed no drawing; false means it is unsupported and
// was skipped.
func (b *frameBuilder) draw(c gpu.Command) (bool, error) {
	if c.Clip.IsEmpty() {
		return true, nil
	}
	if !c.Clip.IsRect() {
		return false, b.unsupported("clip shape")
	}
	m := c.Transform.Storage()
	if m[3] != 0 || m[7] != 0 || m[15] != 1 {
		return false, b.unsupported("perspective transform")
	}
	ctx := &shapeContext{b: b, c: c, t: affineOf(&c.Transform)}
	var (
		ok  bool
		err error
	)
	switch v := c.Op.(type) {
	case ui.OpDrawRect:
		ok, err = ctx.rrect(v.Rect, [4]ui.Radius{}, &v.Paint)
	case ui.OpDrawRRect:
		ok, err = ctx.rrect(rectOfRRect(v.RRect), radiiOf(v.RRect), &v.Paint)
	case ui.OpDrawOval:
		ok, err = ctx.rrect(v.Rect, ovalRadii(v.Rect), &v.Paint)
	case ui.OpDrawCircle:
		r := ui.NewRectFromCircle(v.Center, v.Radius)
		ok, err = ctx.rrect(r, ovalRadii(r), &v.Paint)
	case ui.OpDrawDRRect:
		ok, err = ctx.drrect(v)
	case ui.OpDrawRSuperellipse:
		ok, err = ctx.superellipse(v)
	case ui.OpDrawLine:
		ok, err = ctx.segments([]ui.Offset{v.P1, v.P2}, &v.Paint, c.Op, ui.PointModeLines)
	case ui.OpDrawPoints:
		ok, err = ctx.segments(v.Points, &v.Paint, c.Op, v.Mode)
	case ui.OpDrawColor:
		p := ui.NewPaint()
		p.SetColor(v.Color)
		p.SetBlendMode(v.BlendMode)
		ok, err = ctx.fill(p)
	case ui.OpDrawPaint:
		ok, err = ctx.fill(&v.Paint)
	default:
		return false, b.unsupported("operation " + typeName(c.Op))
	}
	return ok, err
}

// shapeContext is a drawing command being turned into an instance.
type shapeContext struct {
	b *frameBuilder
	c gpu.Command
	t affine2D
}

// scale is the factor that lengths in the shape's units grow by.
func (s *shapeContext) scale() float64 { return math.Sqrt(math.Abs(s.t.det())) }

// paintInfo is what a paint gives the shape pipeline.
type paintInfo struct {
	color [4]float32 // premultiplied
	blend blendFunc
	aa    bool
}

// paintStatus says what to do with a paint.
type paintStatus int

const (
	paintDraw        paintStatus = iota // draw with the info
	paintNothing                        // the paint draws nothing: the command is done
	paintUnsupported                    // the command is skipped, or failed with the error
)

// paint reads the parts of a paint that the shape pipeline draws. For
// paintUnsupported, unsupported has been called, and err is its result.
func (s *shapeContext) paint(p *ui.Paint) (info paintInfo, status paintStatus, err error) {
	switch {
	case p.Shader() != nil:
		return info, paintUnsupported, s.b.unsupported("paint shader")
	case p.ColorFilter() != nil || p.InvertColors():
		return info, paintUnsupported, s.b.unsupported("paint color filter")
	case p.ImageFilter() != nil:
		return info, paintUnsupported, s.b.unsupported("paint image filter")
	case p.MaskFilter() != nil:
		return info, paintUnsupported, s.b.unsupported("paint mask filter")
	}
	mode := p.BlendMode()
	if mode == ui.BlendModeDst {
		return info, paintNothing, nil
	}
	blend, supported := blendFuncs[mode]
	if !supported {
		return info, paintUnsupported, s.b.unsupported("blend mode " + mode.String())
	}
	col := p.Color()
	a := col.A
	info = paintInfo{color: [4]float32{col.R * a, col.G * a, col.B * a, a}, blend: blend, aa: p.IsAntiAlias()}
	return info, paintDraw, nil
}

// stroke returns the half width of a stroke in the shape's units for a paint, and
// the factor to scale its alpha by: a stroke thinner than a pixel is drawn one
// pixel wide with less alpha, as Skia does; width 0 is a hairline, one full pixel.
func (s *shapeContext) stroke(p *ui.Paint) (hw float64, alpha float32) {
	scale := s.scale()
	if scale == 0 {
		return 0, 0
	}
	w := p.StrokeWidth() * scale // in pixels
	switch {
	case w <= 0:
		return 0.5 / scale, 1
	case w < 1:
		return 0.5 / scale, float32(w)
	}
	return p.StrokeWidth() / 2, 1
}

// instance is the attributes of one shape, in the order of the shader.
type instance = [shapeFloats]float32

// place finishes an instance: the placement of the quad around local bounds, the
// inverse transform, the color and the clip. It adds the instance to the frame.
func (s *shapeContext) place(in *instance, local ui.Rect, kind, flags int, info paintInfo, alpha float32) (bool, error) {
	inv, ok := s.t.invert()
	if !ok {
		return true, nil // a flattened shape covers nothing
	}
	if !local.IsFinite() {
		return true, nil
	}
	dev := s.c.Transform.TransformRect(local)
	if !dev.IsFinite() {
		return true, nil
	}
	clip := s.c.Clip.Bounds
	outer := [4]int{int(math.Floor(clip.Left)), int(math.Floor(clip.Top)), int(math.Ceil(clip.Right)), int(math.Ceil(clip.Bottom))}
	outer[0], outer[1] = max(outer[0], 0), max(outer[1], 0)
	outer[2], outer[3] = min(outer[2], s.b.w), min(outer[3], s.b.h)
	if outer[2] <= outer[0] || outer[3] <= outer[1] {
		return true, nil
	}
	// One pixel more on each side for the anti-aliased edge, but not beyond the clip.
	x0, y0 := math.Max(math.Floor(dev.Left)-1, float64(outer[0])), math.Max(math.Floor(dev.Top)-1, float64(outer[1]))
	x1, y1 := math.Min(math.Ceil(dev.Right)+1, float64(outer[2])), math.Min(math.Ceil(dev.Bottom)+1, float64(outer[3]))
	if x1 <= x0 || y1 <= y0 {
		return true, nil
	}
	in[0], in[1], in[2], in[3] = float32(x0), float32(y0), float32(x1), float32(y1)
	in[4], in[5], in[6], in[7] = float32(inv.a), float32(inv.b), float32(inv.c), float32(inv.d)
	if s.t.similarity() {
		flags |= flagSimilarity
	}
	if info.aa {
		flags |= flagAntialias
	}
	in[8], in[9], in[10], in[11] = float32(inv.e), float32(inv.f), float32(kind), float32(flags)
	for i := 0; i < 4; i++ {
		in[12+i] = info.color[i] * alpha
	}
	in[16], in[17], in[18], in[19] = float32(clip.Left), float32(clip.Top), float32(clip.Right), float32(clip.Bottom)
	in[47] = float32(s.scale())
	scissor := [4]int{outer[0], outer[1], outer[2] - outer[0], outer[3] - outer[1]}
	s.b.shape.add(in, info.blend, scissor)
	return true, nil
}

// rrectParts is a rounded rectangle ready for the shader: its center, half
// extents and the radius of each corner, clockwise from the top left.
type rrectParts struct {
	cx, cy, hx, hy float64
	r              [4]ui.Radius
}

func (p rrectParts) write(in *instance, at int) {
	in[at], in[at+1], in[at+2], in[at+3] = float32(p.cx), float32(p.cy), float32(p.hx), float32(p.hy)
	for i, r := range p.r {
		in[at+4+2*i], in[at+5+2*i] = float32(r.X), float32(r.Y)
	}
}

// partsOf normalizes a rectangle with corner radii as Skia does: edges in order,
// a corner with a missing radius square, and the radii scaled down together when
// two on one side are longer than the side.
func partsOf(r ui.Rect, radii [4]ui.Radius) (rrectParts, bool) {
	left, right := math.Min(r.Left, r.Right), math.Max(r.Left, r.Right)
	top, bottom := math.Min(r.Top, r.Bottom), math.Max(r.Top, r.Bottom)
	w, h := right-left, bottom-top
	if !(w >= 0 && h >= 0) || math.IsInf(w, 0) || math.IsInf(h, 0) {
		return rrectParts{}, false
	}
	for i := range radii {
		if !(radii[i].X > 0 && radii[i].Y > 0) {
			radii[i] = ui.Radius{}
		}
	}
	scale := 1.0
	fit := func(a, b, limit float64) {
		if sum := a + b; sum > limit && sum > 0 {
			scale = math.Min(scale, limit/sum)
		}
	}
	fit(radii[3].Y, radii[0].Y, h) // left side
	fit(radii[0].X, radii[1].X, w) // top
	fit(radii[1].Y, radii[2].Y, h) // right
	fit(radii[2].X, radii[3].X, w) // bottom
	if scale < 1 {
		for i := range radii {
			radii[i].X *= scale
			radii[i].Y *= scale
		}
	}
	return rrectParts{cx: (left + right) / 2, cy: (top + bottom) / 2, hx: w / 2, hy: h / 2, r: radii}, true
}

func rectOfRRect(r ui.RRect) ui.Rect {
	return ui.Rect{Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Bottom}
}

func radiiOf(r ui.RRect) [4]ui.Radius {
	return [4]ui.Radius{{X: r.TlRadiusX, Y: r.TlRadiusY}, {X: r.TrRadiusX, Y: r.TrRadiusY}, {X: r.BrRadiusX, Y: r.BrRadiusY}, {X: r.BlRadiusX, Y: r.BlRadiusY}}
}

func ovalRadii(r ui.Rect) [4]ui.Radius {
	x, y := math.Abs(r.Right-r.Left)/2, math.Abs(r.Bottom-r.Top)/2
	return [4]ui.Radius{{X: x, Y: y}, {X: x, Y: y}, {X: x, Y: y}, {X: x, Y: y}}
}

// rrect draws a rounded rectangle, filled or stroked.
func (s *shapeContext) rrect(r ui.Rect, radii [4]ui.Radius, p *ui.Paint) (bool, error) {
	info, status, err := s.paint(p)
	if status != paintDraw {
		return status == paintNothing, err
	}
	parts, valid := partsOf(r, radii)
	if !valid {
		return true, nil
	}
	var in instance
	if p.Style() != ui.PaintingStyleStroke {
		if parts.hx <= 0 || parts.hy <= 0 {
			return true, nil // a fill of an empty rectangle draws nothing
		}
		local := ui.Rect{Left: parts.cx - parts.hx, Top: parts.cy - parts.hy, Right: parts.cx + parts.hx, Bottom: parts.cy + parts.hy}
		if s.boxable(parts) {
			return s.box(&in, local, nil, info, 1)
		}
		parts.write(&in, 20)
		return s.place(&in, local, kindRRect, 0, info, 1)
	}

	hw, alpha := s.stroke(p)
	if alpha == 0 {
		return true, nil
	}
	sharp := false
	for _, c := range parts.r {
		if c.X <= 0 {
			sharp = true
		}
	}
	join := p.StrokeJoin()
	round := join == ui.StrokeJoinRound
	if sharp && !round {
		switch {
		case join == ui.StrokeJoinBevel:
			return false, s.b.unsupported("bevel stroke join")
		case p.StrokeMiterLimit() < math.Sqrt2:
			return false, s.b.unsupported("stroke miter limit below the corner's miter")
		}
	}
	if !sharp || round {
		// The stroke is every point within hw of the outline: exact for circular
		// corners, and for the round joins of square ones.
		parts.write(&in, 20)
		in[44] = float32(hw)
		local := ui.Rect{Left: parts.cx - parts.hx - hw, Top: parts.cy - parts.hy - hw, Right: parts.cx + parts.hx + hw, Bottom: parts.cy + parts.hy + hw}
		return s.place(&in, local, kindOutline, 0, info, alpha)
	}
	// Miter joins at square corners: the outer rectangle minus the inner one.
	outer := rrectParts{cx: parts.cx, cy: parts.cy, hx: parts.hx + hw, hy: parts.hy + hw}
	inner := rrectParts{cx: parts.cx, cy: parts.cy, hx: parts.hx - hw, hy: parts.hy - hw}
	for i, c := range parts.r {
		if c.X > 0 {
			outer.r[i] = ui.Radius{X: c.X + hw, Y: c.Y + hw}
			if c.X > hw && c.Y > hw {
				inner.r[i] = ui.Radius{X: c.X - hw, Y: c.Y - hw}
			}
		}
	}
	local := ui.Rect{Left: outer.cx - outer.hx, Top: outer.cy - outer.hy, Right: outer.cx + outer.hx, Bottom: outer.cy + outer.hy}
	hasInner := inner.hx > 0 && inner.hy > 0
	if s.boxable(parts) {
		return s.box(&in, local, innerRect(inner, hasInner), info, alpha)
	}
	outer.write(&in, 20)
	flags := 0
	if hasInner {
		inner.write(&in, 32)
		flags |= flagHasB
	}
	return s.place(&in, local, kindRRect, flags, info, alpha)
}

// boxable says that the rectangle has square corners and the transform keeps it
// axis-aligned, so that the exact coverage of a box can be used.
func (s *shapeContext) boxable(p rrectParts) bool {
	for _, r := range p.r {
		if r.X > 0 {
			return false
		}
	}
	return gpu.IsAxisAligned(&s.c.Transform)
}

// innerRect is the rectangle of the hole of a stroke, or nil when it has none.
func innerRect(p rrectParts, ok bool) *ui.Rect {
	if !ok {
		return nil
	}
	return &ui.Rect{Left: p.cx - p.hx, Top: p.cy - p.hy, Right: p.cx + p.hx, Bottom: p.cy + p.hy}
}

// box draws an axis-aligned rectangle, with a rectangular hole when inner is not
// nil, by the exact area of each pixel it covers. The rectangles are given in the
// shape's units and put in the instance in pixels.
func (s *shapeContext) box(in *instance, outer ui.Rect, inner *ui.Rect, info paintInfo, alpha float32) (bool, error) {
	d := s.c.Transform.TransformRect(outer)
	in[20], in[21], in[22], in[23] = float32(d.Left), float32(d.Top), float32(d.Right), float32(d.Bottom)
	flags := 0
	if inner != nil {
		di := s.c.Transform.TransformRect(*inner)
		in[32], in[33], in[34], in[35] = float32(di.Left), float32(di.Top), float32(di.Right), float32(di.Bottom)
		flags |= flagHasB
	}
	return s.place(in, outer, kindBox, flags, info, alpha)
}

// drrect draws the region between two rounded rectangles.
func (s *shapeContext) drrect(v ui.OpDrawDRRect) (bool, error) {
	info, status, err := s.paint(&v.Paint)
	if status != paintDraw {
		return status == paintNothing, err
	}
	if v.Paint.Style() == ui.PaintingStyleStroke {
		return false, s.b.unsupported("stroked DRRect")
	}
	outer, ok1 := partsOf(rectOfRRect(v.Outer), radiiOf(v.Outer))
	inner, ok2 := partsOf(rectOfRRect(v.Inner), radiiOf(v.Inner))
	if !ok1 || !ok2 {
		return true, nil
	}
	var in instance
	outer.write(&in, 20)
	flags := 0
	if inner.hx > 0 && inner.hy > 0 {
		inner.write(&in, 32)
		flags |= flagHasB
	}
	local := ui.Rect{Left: outer.cx - outer.hx, Top: outer.cy - outer.hy, Right: outer.cx + outer.hx, Bottom: outer.cy + outer.hy}
	return s.place(&in, local, kindRRect, flags, info, 1)
}

// superellipse draws a filled rounded superellipse.
func (s *shapeContext) superellipse(v ui.OpDrawRSuperellipse) (bool, error) {
	info, status, err := s.paint(&v.Paint)
	if status != paintDraw {
		return status == paintNothing, err
	}
	if v.Paint.Style() == ui.PaintingStyleStroke {
		return false, s.b.unsupported("stroked superellipse")
	}
	rs := v.RSuperellipse
	parts, valid := partsOf(rectOfRRect(ui.RRect(rs)), radiiOf(ui.RRect(rs)))
	if !valid || parts.hx <= 0 || parts.hy <= 0 {
		return true, nil
	}
	// The corner boxes: the radius times the extent, as far as it fits.
	for i, c := range parts.r {
		parts.r[i] = ui.Radius{X: math.Min(c.X*superellipseExtent, parts.hx), Y: math.Min(c.Y*superellipseExtent, parts.hy)}
	}
	var in instance
	parts.write(&in, 20)
	in[44+2] = superellipseExponent
	local := ui.Rect{Left: parts.cx - parts.hx, Top: parts.cy - parts.hy, Right: parts.cx + parts.hx, Bottom: parts.cy + parts.hy}
	return s.place(&in, local, kindSuper, 0, info, 1)
}

// segments draws lines and points with the paint's width and cap.
func (s *shapeContext) segments(pts []ui.Offset, p *ui.Paint, op ui.DisplayOp, mode ui.PointMode) (bool, error) {
	info, status, err := s.paint(p)
	if status != paintDraw {
		return status == paintNothing, err
	}
	if mode == ui.PointModePolygon {
		return false, s.b.unsupported("polygon points")
	}
	hw, alpha := s.stroke(p)
	if alpha == 0 || len(pts) == 0 {
		return true, nil
	}
	cap := float32(0)
	switch p.StrokeCap() {
	case ui.StrokeCapRound:
		cap = 1
	case ui.StrokeCapSquare:
		cap = 2
	}
	if mode == ui.PointModePoints && cap == 0 {
		cap = 2 // a point is a square unless the cap is round
	}
	draw := func(a, b ui.Offset) (bool, error) {
		if a == b && cap == 0 {
			return true, nil // a butt cap on no length draws nothing
		}
		var in instance
		in[20], in[21], in[22], in[23] = float32(a.Dx), float32(a.Dy), float32(b.Dx), float32(b.Dy)
		in[44], in[45] = float32(hw), cap
		reach := hw * math.Sqrt2
		local := ui.Rect{
			Left: math.Min(a.Dx, b.Dx) - reach, Top: math.Min(a.Dy, b.Dy) - reach,
			Right: math.Max(a.Dx, b.Dx) + reach, Bottom: math.Max(a.Dy, b.Dy) + reach,
		}
		return s.place(&in, local, kindSegment, 0, info, alpha)
	}
	switch mode {
	case ui.PointModeLines:
		if _, isLine := op.(ui.OpDrawLine); isLine {
			return draw(pts[0], pts[1])
		}
		for i := 0; i+1 < len(pts); i += 2 {
			if _, err := draw(pts[i], pts[i+1]); err != nil {
				return true, err
			}
		}
	default:
		for _, pt := range pts {
			if _, err := draw(pt, pt); err != nil {
				return true, err
			}
		}
	}
	return true, nil
}

// fill paints the whole clip with the paint's color.
func (s *shapeContext) fill(p *ui.Paint) (bool, error) {
	info, status, err := s.paint(p)
	if status != paintDraw {
		return status == paintNothing, err
	}
	// The quad is the clip, whatever the transform: the identity places it.
	id := shapeContext{b: s.b, c: s.c, t: affine2D{a: 1, d: 1}}
	id.c.Transform = *ui.NewMatrix4Identity()
	var in instance
	return id.place(&in, s.c.Clip.Bounds, kindFill, 0, info, 1)
}

// typeName is the name of the dynamic type of an operation, for the list of
// unsupported features.
func typeName(op ui.DisplayOp) string { return fmt.Sprintf("%T", op) }
