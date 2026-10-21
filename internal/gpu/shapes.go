package gpu

import (
	"image"
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// The analytic shapes (SPEC §9.1): rectangles, rounded rectangles, ovals, circles,
// rounded superellipses, the region between two rounded rectangles, lines and points.
// Each shape is one instance of a quad whose fragment shader finds the coverage of every
// pixel from a signed distance field, with no multisampling. A ShapeEncoder turns the
// shapes of a drawing command into instances; the GLES and the explicit renderers draw
// them with the same shader logic (gles/shaders/shape.*.glsl, explicit/shaders/shape.wgsl).

// The layout of an instance: twelve vec4 attributes, the same in both shaders.
const (
	// ShapeVec4s is the number of vec4 attributes of a shape instance.
	//
	// Flutter: no counterpart
	ShapeVec4s = 12
	// ShapeFloats is the size of a shape instance in floats.
	//
	// Flutter: no counterpart
	ShapeFloats = ShapeVec4s * 4
	// ShapeStride is the size of a shape instance in bytes.
	//
	// Flutter: no counterpart
	ShapeStride = ShapeFloats * 4
)

// The kinds of shape of the shader (instance float 10).
const (
	// ShapeKindRRect is a rounded rectangle, minus shape B when the flags say so.
	//
	// Flutter: no counterpart
	ShapeKindRRect = 0
	// ShapeKindSuper is a rounded superellipse.
	//
	// Flutter: no counterpart
	ShapeKindSuper = 1
	// ShapeKindSegment is a line segment with caps, or a dot.
	//
	// Flutter: no counterpart
	ShapeKindSegment = 2
	// ShapeKindFill covers every pixel of the quad.
	//
	// Flutter: no counterpart
	ShapeKindFill = 3
	// ShapeKindBox is an axis-aligned rectangle in pixels, minus shape B, by exact area.
	//
	// Flutter: no counterpart
	ShapeKindBox = 4
	// ShapeKindOutline is every point within a half width of a rounded rectangle's
	// outline.
	//
	// Flutter: no counterpart
	ShapeKindOutline = 5
	// ShapeKindShadow is a rounded rectangle in pixels blurred by a Gaussian.
	//
	// Flutter: no counterpart
	ShapeKindShadow = 6
)

// The flags of the shader (instance float 11).
const (
	// ShapeFlagHasB says that shape B is cut out of shape A.
	//
	// Flutter: no counterpart
	ShapeFlagHasB = 1
	// ShapeFlagAntialias says that the edge is smoothed.
	//
	// Flutter: no counterpart
	ShapeFlagAntialias = 2
	// ShapeFlagSimilarity says that the transform only rotates, flips and scales evenly.
	//
	// Flutter: no counterpart
	ShapeFlagSimilarity = 4
)

// The constants of ui.Path.AddRSuperellipse, which the software renderer draws: a corner
// box is this much times the radius, filled by |x|^n + |y|^n = 1.
const (
	// SuperellipseExtent is the size of a corner box in radii.
	//
	// Flutter: no counterpart
	SuperellipseExtent = 1.2924
	// SuperellipseExponent is the exponent n of the corner curve.
	//
	// Flutter: no counterpart
	SuperellipseExponent = 5
)

// ShapeInstance is the attributes of one shape, in the order of the shaders: the quad in
// pixels (0-3), the inverse transform (4-9), the kind and flags (10, 11), the premultiplied
// color (12-15), the clip rectangle (16-19), shape A (20-31), shape B (32-43), the
// parameters (44-47: half stroke width, cap, exponent, pixels per local unit).
//
// Flutter: no counterpart
type ShapeInstance = [ShapeFloats]float32

// ShapePaint is what a paint gives the shape encoder.
//
// Flutter: no counterpart
type ShapePaint struct {
	// Color is premultiplied; for a shader, its alpha in all four components.
	Color [4]float32
	// Antialias says whether the edge is smoothed.
	Antialias bool
}

// ShapeEncoder turns the shapes of one drawing command into instances. The renderer
// fills in the command's state and Emit, which receives each instance with the pixels it
// covers; the methods return the name of a feature when the shape has no analytic form
// and the coverage atlas has to draw it, and "" otherwise (including when the shape
// draws nothing).
//
// Flutter: no counterpart
type ShapeEncoder struct {
	// Transform maps the shape's coordinates to the surface's pixels, and T is its 2D part.
	Transform *ui.Matrix4
	T         Affine2D
	// Outer is the pixels the command's clip reaches, inside the target.
	Outer image.Rectangle
	// Clip is the clip rectangle, which the shader cuts to by exact coverage.
	Clip [4]float32
	// ClipBounds is the clip's bounds, which Fill covers.
	ClipBounds ui.Rect
	// Paint is the color and antialiasing for the instances.
	Paint ShapePaint
	// Emit is called with each finished instance and the pixels its quad covers.
	Emit func(in *ShapeInstance, covers image.Rectangle)
}

// Scale is the factor that lengths in the shape's units grow by.
//
// Flutter: no counterpart
func (e *ShapeEncoder) Scale() float64 { return math.Sqrt(math.Abs(e.T.Det())) }

// StrokeHalfWidth returns the half width of a stroke in the shape's units for a paint,
// and the factor to scale its alpha by: a stroke thinner than a pixel is drawn one pixel
// wide with less alpha, as Skia does; width 0 is a hairline, one full pixel.
//
// Flutter: no counterpart
func (e *ShapeEncoder) StrokeHalfWidth(p *ui.Paint) (hw float64, alpha float32) {
	scale := e.Scale()
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

// Place finishes an instance: the placement of the quad around the local bounds, the
// inverse transform, the color and the clip. It hands the instance to Emit.
//
// Flutter: no counterpart
func (e *ShapeEncoder) Place(in *ShapeInstance, local ui.Rect, kind, flags int, alpha float32) {
	e.PlacePadded(in, local, kind, flags, alpha, 1)
}

// PlacePadded is Place with the quad grown by pad pixels around the shape's bounds.
//
// Flutter: no counterpart
func (e *ShapeEncoder) PlacePadded(in *ShapeInstance, local ui.Rect, kind, flags int, alpha float32, pad float64) {
	inv, ok := e.T.Invert()
	if !ok {
		return // a flattened shape covers nothing
	}
	if !local.IsFinite() {
		return
	}
	dev := e.Transform.TransformRect(local)
	if !dev.IsFinite() {
		return
	}
	// One pixel more on each side for the anti-aliased edge, but not beyond the clip.
	o := e.Outer
	x0, y0 := math.Max(math.Floor(dev.Left)-pad, float64(o.Min.X)), math.Max(math.Floor(dev.Top)-pad, float64(o.Min.Y))
	x1, y1 := math.Min(math.Ceil(dev.Right)+pad, float64(o.Max.X)), math.Min(math.Ceil(dev.Bottom)+pad, float64(o.Max.Y))
	if x1 <= x0 || y1 <= y0 {
		return
	}
	in[0], in[1], in[2], in[3] = float32(x0), float32(y0), float32(x1), float32(y1)
	in[4], in[5], in[6], in[7] = float32(inv.A), float32(inv.B), float32(inv.C), float32(inv.D)
	if e.T.Similarity() {
		flags |= ShapeFlagSimilarity
	}
	if e.Paint.Antialias {
		flags |= ShapeFlagAntialias
	}
	in[8], in[9], in[10], in[11] = float32(inv.E), float32(inv.F), float32(kind), float32(flags)
	for i := 0; i < 4; i++ {
		in[12+i] = e.Paint.Color[i] * alpha
	}
	copy(in[16:20], e.Clip[:])
	in[47] = float32(e.Scale())
	e.Emit(in, image.Rect(int(x0), int(y0), int(x1), int(y1)))
}

// RRectParts is a rounded rectangle ready for the shader: its center, half extents and the
// radius of each corner, clockwise from the top left.
//
// Flutter: no counterpart
type RRectParts struct {
	CX, CY, HX, HY float64
	R              [4]ui.Radius
}

// Write puts the parts in the instance at the float index at: center and half extents,
// then the radii of the four corners.
//
// Flutter: no counterpart
func (p RRectParts) Write(in *ShapeInstance, at int) {
	in[at], in[at+1], in[at+2], in[at+3] = float32(p.CX), float32(p.CY), float32(p.HX), float32(p.HY)
	for i, r := range p.R {
		in[at+4+2*i], in[at+5+2*i] = float32(r.X), float32(r.Y)
	}
}

// Rect returns the rectangle of the parts.
//
// Flutter: no counterpart
func (p RRectParts) Rect() ui.Rect {
	return ui.Rect{Left: p.CX - p.HX, Top: p.CY - p.HY, Right: p.CX + p.HX, Bottom: p.CY + p.HY}
}

// PartsOf normalizes a rectangle with corner radii as Skia does: edges in order, a corner
// with a missing radius square, and the radii scaled down together when two on one side
// are longer than the side. It reports false for a rectangle that is not finite.
//
// Flutter: no counterpart
func PartsOf(r ui.Rect, radii [4]ui.Radius) (RRectParts, bool) {
	left, right := math.Min(r.Left, r.Right), math.Max(r.Left, r.Right)
	top, bottom := math.Min(r.Top, r.Bottom), math.Max(r.Top, r.Bottom)
	w, h := right-left, bottom-top
	if !(w >= 0 && h >= 0) || math.IsInf(w, 0) || math.IsInf(h, 0) {
		return RRectParts{}, false
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
	return RRectParts{CX: (left + right) / 2, CY: (top + bottom) / 2, HX: w / 2, HY: h / 2, R: radii}, true
}

// RectOfRRect returns the bounds of a rounded rectangle.
//
// Flutter: no counterpart
func RectOfRRect(r ui.RRect) ui.Rect {
	return ui.Rect{Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Bottom}
}

// RadiiOfRRect returns the corner radii of a rounded rectangle, clockwise from the top
// left.
//
// Flutter: no counterpart
func RadiiOfRRect(r ui.RRect) [4]ui.Radius {
	return [4]ui.Radius{{X: r.TlRadiusX, Y: r.TlRadiusY}, {X: r.TrRadiusX, Y: r.TrRadiusY}, {X: r.BrRadiusX, Y: r.BrRadiusY}, {X: r.BlRadiusX, Y: r.BlRadiusY}}
}

// OvalRadii returns the radii that make a rounded rectangle of the rectangle an oval.
//
// Flutter: no counterpart
func OvalRadii(r ui.Rect) [4]ui.Radius {
	x, y := math.Abs(r.Right-r.Left)/2, math.Abs(r.Bottom-r.Top)/2
	return [4]ui.Radius{{X: x, Y: y}, {X: x, Y: y}, {X: x, Y: y}, {X: x, Y: y}}
}

// PixelRect returns the pixels that a rectangle touches.
//
// Flutter: no counterpart
func PixelRect(r ui.Rect) image.Rectangle {
	if !r.IsFinite() {
		return image.Rectangle{}
	}
	clamp := func(v float64) int { return int(math.Max(-1e9, math.Min(1e9, v))) }
	return image.Rect(clamp(math.Floor(r.Left)), clamp(math.Floor(r.Top)), clamp(math.Ceil(r.Right)), clamp(math.Ceil(r.Bottom)))
}

// RRect draws a rounded rectangle, filled or stroked.
//
// Flutter: no counterpart
func (e *ShapeEncoder) RRect(r ui.Rect, radii [4]ui.Radius, p *ui.Paint) (needsMask string) {
	parts, valid := PartsOf(r, radii)
	if !valid {
		return ""
	}
	var in ShapeInstance
	if p.Style() != ui.PaintingStyleStroke {
		if parts.HX <= 0 || parts.HY <= 0 {
			return "" // a fill of an empty rectangle draws nothing
		}
		local := parts.Rect()
		if e.boxable(parts) {
			e.box(&in, local, nil, 1)
			return ""
		}
		parts.Write(&in, 20)
		e.Place(&in, local, ShapeKindRRect, 0, 1)
		return ""
	}

	hw, alpha := e.StrokeHalfWidth(p)
	if alpha == 0 {
		return ""
	}
	sharp := false
	for _, c := range parts.R {
		if c.X <= 0 {
			sharp = true
		}
	}
	join := p.StrokeJoin()
	round := join == ui.StrokeJoinRound
	if sharp && !round {
		switch {
		case join == ui.StrokeJoinBevel:
			return "bevel stroke join"
		case p.StrokeMiterLimit() < math.Sqrt2:
			return "stroke miter limit below the corner's miter"
		}
	}
	if !sharp || round {
		// The stroke is every point within hw of the outline: exact for circular
		// corners, and for the round joins of square ones.
		parts.Write(&in, 20)
		in[44] = float32(hw)
		local := ui.Rect{Left: parts.CX - parts.HX - hw, Top: parts.CY - parts.HY - hw, Right: parts.CX + parts.HX + hw, Bottom: parts.CY + parts.HY + hw}
		e.Place(&in, local, ShapeKindOutline, 0, alpha)
		return ""
	}
	// Miter joins at square corners: the outer rectangle minus the inner one.
	outer := RRectParts{CX: parts.CX, CY: parts.CY, HX: parts.HX + hw, HY: parts.HY + hw}
	inner := RRectParts{CX: parts.CX, CY: parts.CY, HX: parts.HX - hw, HY: parts.HY - hw}
	for i, c := range parts.R {
		if c.X > 0 {
			outer.R[i] = ui.Radius{X: c.X + hw, Y: c.Y + hw}
			if c.X > hw && c.Y > hw {
				inner.R[i] = ui.Radius{X: c.X - hw, Y: c.Y - hw}
			}
		}
	}
	local := outer.Rect()
	hasInner := inner.HX > 0 && inner.HY > 0
	if e.boxable(parts) {
		var hole *ui.Rect
		if hasInner {
			r := inner.Rect()
			hole = &r
		}
		e.box(&in, local, hole, alpha)
		return ""
	}
	outer.Write(&in, 20)
	flags := 0
	if hasInner {
		inner.Write(&in, 32)
		flags |= ShapeFlagHasB
	}
	e.Place(&in, local, ShapeKindRRect, flags, alpha)
	return ""
}

// boxable says that the rectangle has square corners and the transform keeps it
// axis-aligned, so that the exact coverage of a box can be used.
func (e *ShapeEncoder) boxable(p RRectParts) bool {
	for _, r := range p.R {
		if r.X > 0 {
			return false
		}
	}
	return IsAxisAligned(e.Transform)
}

// box draws an axis-aligned rectangle, with a rectangular hole when inner is not nil, by
// the exact area of each pixel it covers. The rectangles are given in the shape's units and
// put in the instance in pixels.
func (e *ShapeEncoder) box(in *ShapeInstance, outer ui.Rect, inner *ui.Rect, alpha float32) {
	d := e.Transform.TransformRect(outer)
	in[20], in[21], in[22], in[23] = float32(d.Left), float32(d.Top), float32(d.Right), float32(d.Bottom)
	flags := 0
	if inner != nil {
		di := e.Transform.TransformRect(*inner)
		in[32], in[33], in[34], in[35] = float32(di.Left), float32(di.Top), float32(di.Right), float32(di.Bottom)
		flags |= ShapeFlagHasB
	}
	e.Place(in, outer, ShapeKindBox, flags, alpha)
}

// DRRect draws the region between two rounded rectangles.
//
// Flutter: no counterpart
func (e *ShapeEncoder) DRRect(outer, inner ui.RRect, p *ui.Paint) (needsMask string) {
	if p.Style() == ui.PaintingStyleStroke {
		return "stroked DRRect"
	}
	o, ok1 := PartsOf(RectOfRRect(outer), RadiiOfRRect(outer))
	i, ok2 := PartsOf(RectOfRRect(inner), RadiiOfRRect(inner))
	if !ok1 || !ok2 {
		return ""
	}
	var in ShapeInstance
	o.Write(&in, 20)
	flags := 0
	if i.HX > 0 && i.HY > 0 {
		i.Write(&in, 32)
		flags |= ShapeFlagHasB
	}
	e.Place(&in, o.Rect(), ShapeKindRRect, flags, 1)
	return ""
}

// Superellipse draws a filled rounded superellipse.
//
// Flutter: no counterpart
func (e *ShapeEncoder) Superellipse(rs ui.RSuperellipse, p *ui.Paint) (needsMask string) {
	if p.Style() == ui.PaintingStyleStroke {
		return "stroked superellipse"
	}
	parts, valid := PartsOf(RectOfRRect(ui.RRect(rs)), RadiiOfRRect(ui.RRect(rs)))
	if !valid || parts.HX <= 0 || parts.HY <= 0 {
		return ""
	}
	// The corner boxes: the radius times the extent, as far as it fits.
	for i, c := range parts.R {
		parts.R[i] = ui.Radius{X: math.Min(c.X*SuperellipseExtent, parts.HX), Y: math.Min(c.Y*SuperellipseExtent, parts.HY)}
	}
	var in ShapeInstance
	parts.Write(&in, 20)
	in[44+2] = SuperellipseExponent
	e.Place(&in, parts.Rect(), ShapeKindSuper, 0, 1)
	return ""
}

// Segments draws lines and points with the paint's width and cap. For a line, pts are its
// two ends.
//
// Flutter: no counterpart
func (e *ShapeEncoder) Segments(pts []ui.Offset, p *ui.Paint, line bool, mode ui.PointMode) (needsMask string) {
	if mode == ui.PointModePolygon {
		return "polygon points"
	}
	hw, alpha := e.StrokeHalfWidth(p)
	if alpha == 0 || len(pts) == 0 {
		return ""
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
	draw := func(a, b ui.Offset) {
		if a == b && cap == 0 {
			return // a butt cap on no length draws nothing
		}
		var in ShapeInstance
		in[20], in[21], in[22], in[23] = float32(a.Dx), float32(a.Dy), float32(b.Dx), float32(b.Dy)
		in[44], in[45] = float32(hw), cap
		reach := hw * math.Sqrt2
		local := ui.Rect{
			Left: math.Min(a.Dx, b.Dx) - reach, Top: math.Min(a.Dy, b.Dy) - reach,
			Right: math.Max(a.Dx, b.Dx) + reach, Bottom: math.Max(a.Dy, b.Dy) + reach,
		}
		e.Place(&in, local, ShapeKindSegment, 0, alpha)
	}
	switch mode {
	case ui.PointModeLines:
		if line {
			draw(pts[0], pts[1])
			return ""
		}
		for i := 0; i+1 < len(pts); i += 2 {
			draw(pts[i], pts[i+1])
		}
	default:
		for _, pt := range pts {
			draw(pt, pt)
		}
	}
	return ""
}

// Fill paints the whole clip with the paint's color.
//
// Flutter: no counterpart
func (e *ShapeEncoder) Fill() {
	// The quad is the clip, whatever the transform: the identity places it.
	id := *e
	id.T = Affine2D{A: 1, D: 1}
	id.Transform = ui.NewMatrix4Identity()
	var in ShapeInstance
	id.Place(&in, e.ClipBounds, ShapeKindFill, 0, 1)
}
