package layout

import (
	"math"
	"sort"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype/tables"
	"github.com/rm4n0s/glutter/ui"
)

// maxPaintDepth bounds the nesting of a COLR paint graph, which a broken font could make
// endless.
const maxPaintDepth = 64

// colorGlyph returns the picture of a glyph that the COLR table of the face paints, in font
// units with y growing up (it is drawn under a scale of size/upem and -size/upem), or nil if
// the glyph is not a COLR glyph. fg is the color the paint with the palette index 0xFFFF
// has: the color of the text.
//
// Flutter: no counterpart (Skia draws COLR through FreeType's color glyph support)
func (e *Engine) colorGlyph(face *font.Face, gid font.GID, fg ui.Color) *ui.Picture {
	if face.COLR == nil {
		return nil
	}
	key := colorKey{face, gid, fg}
	if pic, ok := e.pictures[key]; ok {
		return pic
	}
	if len(e.pictures) >= maxCachedGlyphs {
		clear(e.pictures)
	}
	var pic *ui.Picture
	if paint, ok := face.COLR.Search(tables.GlyphID(gid)); ok {
		rec := ui.NewPictureRecorder()
		upem := float64(face.Upem())
		cp := &colrPainter{face: face, fg: fg, c: ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.Rect{Left: -upem, Top: -upem, Right: 2 * upem, Bottom: 2 * upem})}), e: e}
		cp.c.ClipRect(ui.Rect{Left: -upem, Top: -upem, Right: 2 * upem, Bottom: 2 * upem})
		cp.paint(paint, 0)
		pic = rec.EndRecording()
	}
	e.pictures[key] = pic
	return pic
}

// colrPainter records the drawing of a COLR paint graph on a canvas.
type colrPainter struct {
	face *font.Face
	fg   ui.Color
	c    *ui.Canvas
	e    *Engine
}

// color returns a palette color with an alpha factor.
func (cp *colrPainter) color(index uint16, alpha float64) ui.Color {
	c := cp.fg
	if index != 0xFFFF {
		var rec tables.ColorRecord
		if len(cp.face.CPAL) > 0 && int(index) < len(cp.face.CPAL[0]) {
			rec = cp.face.CPAL[0][index]
		}
		c = ui.NewColorFromARGB(int(rec.Alpha), int(rec.Red), int(rec.Green), int(rec.Blue))
		return c.WithValues(ui.ColorWithValuesOpts{Alpha: ui.Ptr(float64(c.A) * alpha)})
	}
	return c.WithValues(ui.ColorWithValuesOpts{Alpha: ui.Ptr(float64(c.A) * alpha)})
}

func fixed214(c tables.Fixed214) float64 { return float64(c) / (1 << 14) }

// affine makes the matrix of x' = xx x + xy y + dx, y' = yx x + yy y + dy.
func affine(xx, yx, xy, yy, dx, dy float64) *ui.Matrix4 {
	return ui.NewMatrix4FromList([16]float32{
		float32(xx), float32(yx), 0, 0,
		float32(xy), float32(yy), 0, 0,
		0, 0, 1, 0,
		float32(dx), float32(dy), 0, 1,
	})
}

// around makes the transformation m (a matrix about the origin) about the point (cx, cy).
func around(m *ui.Matrix4, cx, cy float64) *ui.Matrix4 {
	out := ui.NewMatrix4TranslationValues(float32(cx), float32(cy), 0)
	out.MultiplyBy(m)
	out.Translate(float32(-cx), float32(-cy), 0)
	return out
}

func (cp *colrPainter) transformed(m *ui.Matrix4, inner tables.PaintTable, depth int) {
	cp.c.Save()
	cp.c.Transform(m)
	cp.paint(inner, depth+1)
	cp.c.Restore()
}

// paint records a paint table.
func (cp *colrPainter) paint(pt tables.PaintTable, depth int) {
	if depth > maxPaintDepth || pt == nil {
		return
	}
	c := cp.c
	switch v := pt.(type) {
	case tables.PaintColrLayersResolved:
		for _, l := range v {
			cp.glyphFill(uint16(l.GlyphID), func() { c.DrawPaint(fillPaint(cp.color(l.PaletteIndex, 1))) })
		}
	case tables.PaintColrLayers:
		layers, err := cp.face.COLR.LayerList.Resolve(v)
		if err != nil {
			return
		}
		for _, l := range layers {
			cp.paint(l, depth+1)
		}
	case tables.PaintSolid:
		c.DrawPaint(fillPaint(cp.color(v.PaletteIndex, fixed214(v.Alpha))))
	case tables.PaintVarSolid:
		c.DrawPaint(fillPaint(cp.color(v.PaletteIndex, fixed214(v.Alpha))))
	case tables.PaintLinearGradient:
		cp.linear(v.ColorLine.ColorStops, v.ColorLine.Extend, v.X0, v.Y0, v.X1, v.Y1, v.X2, v.Y2)
	case tables.PaintVarLinearGradient:
		stops := make([]tables.ColorStop, len(v.ColorLine.ColorStops))
		for i, s := range v.ColorLine.ColorStops {
			stops[i] = tables.ColorStop{StopOffset: s.StopOffset, PaletteIndex: s.PaletteIndex, Alpha: s.Alpha}
		}
		cp.linear(stops, v.ColorLine.Extend, v.X0, v.Y0, v.X1, v.Y1, v.X2, v.Y2)
	case tables.PaintRadialGradient:
		cp.radial(v.ColorLine.ColorStops, v.ColorLine.Extend, v.X0, v.Y0, float64(v.Radius0), v.X1, v.Y1, float64(v.Radius1))
	case tables.PaintVarRadialGradient:
		cp.radial(varStops(v.ColorLine.ColorStops), v.ColorLine.Extend, v.X0, v.Y0, float64(v.Radius0), v.X1, v.Y1, float64(v.Radius1))
	case tables.PaintSweepGradient:
		cp.sweep(v.ColorLine.ColorStops, v.ColorLine.Extend, v.CenterX, v.CenterY, v.StartAngle, v.EndAngle)
	case tables.PaintVarSweepGradient:
		cp.sweep(varStops(v.ColorLine.ColorStops), v.ColorLine.Extend, v.CenterX, v.CenterY, v.StartAngle, v.EndAngle)
	case tables.PaintGlyph:
		cp.glyphFill(v.GlyphID, func() { cp.paint(v.Paint, depth+1) })
	case tables.PaintColrGlyph:
		if inner, ok := cp.face.COLR.Search(tables.GlyphID(v.GlyphID)); ok {
			cp.paint(inner, depth+1)
		}
	case tables.PaintTransform:
		cp.transformed(affine(float64(v.Transform.Xx), float64(v.Transform.Yx), float64(v.Transform.Xy), float64(v.Transform.Yy), float64(v.Transform.Dx), float64(v.Transform.Dy)), v.Paint, depth)
	case tables.PaintVarTransform:
		cp.transformed(affine(float64(v.Transform.Xx), float64(v.Transform.Yx), float64(v.Transform.Xy), float64(v.Transform.Yy), float64(v.Transform.Dx), float64(v.Transform.Dy)), v.Paint, depth)
	case tables.PaintTranslate:
		cp.transformed(affine(1, 0, 0, 1, float64(v.Dx), float64(v.Dy)), v.Paint, depth)
	case tables.PaintVarTranslate:
		cp.transformed(affine(1, 0, 0, 1, float64(v.Dx), float64(v.Dy)), v.Paint, depth)
	case tables.PaintScale:
		cp.transformed(affine(fixed214(v.ScaleX), 0, 0, fixed214(v.ScaleY), 0, 0), v.Paint, depth)
	case tables.PaintVarScale:
		cp.transformed(affine(fixed214(v.ScaleX), 0, 0, fixed214(v.ScaleY), 0, 0), v.Paint, depth)
	case tables.PaintScaleAroundCenter:
		cp.transformed(around(affine(fixed214(v.ScaleX), 0, 0, fixed214(v.ScaleY), 0, 0), float64(v.CenterX), float64(v.CenterY)), v.Paint, depth)
	case tables.PaintVarScaleAroundCenter:
		cp.transformed(around(affine(fixed214(v.ScaleX), 0, 0, fixed214(v.ScaleY), 0, 0), float64(v.CenterX), float64(v.CenterY)), v.Paint, depth)
	case tables.PaintScaleUniform:
		cp.transformed(affine(fixed214(v.Scale), 0, 0, fixed214(v.Scale), 0, 0), v.Paint, depth)
	case tables.PaintVarScaleUniform:
		cp.transformed(affine(fixed214(v.Scale), 0, 0, fixed214(v.Scale), 0, 0), v.Paint, depth)
	case tables.PaintScaleUniformAroundCenter:
		cp.transformed(around(affine(fixed214(v.Scale), 0, 0, fixed214(v.Scale), 0, 0), float64(v.CenterX), float64(v.CenterY)), v.Paint, depth)
	case tables.PaintVarScaleUniformAroundCenter:
		cp.transformed(around(affine(fixed214(v.Scale), 0, 0, fixed214(v.Scale), 0, 0), float64(v.CenterX), float64(v.CenterY)), v.Paint, depth)
	case tables.PaintRotate:
		cp.transformed(rotation(fixed214(v.Angle)), v.Paint, depth)
	case tables.PaintVarRotate:
		cp.transformed(rotation(fixed214(v.Angle)), v.Paint, depth)
	case tables.PaintRotateAroundCenter:
		cp.transformed(around(rotation(fixed214(v.Angle)), float64(v.CenterX), float64(v.CenterY)), v.Paint, depth)
	case tables.PaintVarRotateAroundCenter:
		cp.transformed(around(rotation(fixed214(v.Angle)), float64(v.CenterX), float64(v.CenterY)), v.Paint, depth)
	case tables.PaintSkew:
		cp.transformed(skew(fixed214(v.XSkewAngle), fixed214(v.YSkewAngle)), v.Paint, depth)
	case tables.PaintVarSkew:
		cp.transformed(skew(fixed214(v.XSkewAngle), fixed214(v.YSkewAngle)), v.Paint, depth)
	case tables.PaintSkewAroundCenter:
		cp.transformed(around(skew(fixed214(v.XSkewAngle), fixed214(v.YSkewAngle)), float64(v.CenterX), float64(v.CenterY)), v.Paint, depth)
	case tables.PaintVarSkewAroundCenter:
		cp.transformed(around(skew(fixed214(v.XSkewAngle), fixed214(v.YSkewAngle)), float64(v.CenterX), float64(v.CenterY)), v.Paint, depth)
	case tables.PaintComposite:
		// The source is blended onto what the backdrop painted.
		c.SaveLayer(nil, ui.NewPaint())
		cp.paint(v.BackdropPaint, depth+1)
		sp := ui.NewPaint()
		sp.SetBlendMode(compositeBlend(v.CompositeMode))
		c.SaveLayer(nil, sp)
		cp.paint(v.SourcePaint, depth+1)
		c.Restore()
		c.Restore()
	}
}

func varStops(in []tables.VarColorStop) []tables.ColorStop {
	out := make([]tables.ColorStop, len(in))
	for i, s := range in {
		out[i] = tables.ColorStop{StopOffset: s.StopOffset, PaletteIndex: s.PaletteIndex, Alpha: s.Alpha}
	}
	return out
}

// rotation is the counter-clockwise rotation of angle (in units of 180 degrees); the glyphs
// are drawn with y up, where the canvas rotation is counter-clockwise too.
func rotation(angle float64) *ui.Matrix4 {
	return ui.NewMatrix4RotationZ(angle * math.Pi)
}

// skew is the skew of the COLR table (the angles are counter-clockwise, in units of 180
// degrees): x' = x - tan(xAngle) y and y' = y + tan(yAngle) x, with y up.
func skew(xAngle, yAngle float64) *ui.Matrix4 {
	return affine(1, math.Tan(yAngle*math.Pi), -math.Tan(xAngle*math.Pi), 1, 0, 0)
}

// glyphFill clips to the outline of a glyph (in font units, y up) and runs fill.
func (cp *colrPainter) glyphFill(gid uint16, fill func()) {
	out, ok := cp.face.GlyphDataOutline(tables.GlyphID(gid))
	if !ok {
		return
	}
	cp.c.Save()
	cp.c.ClipPath(outlinePath(out.Segments, 1, 1))
	fill()
	cp.c.Restore()
}

func fillPaint(c ui.Color) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(c)
	return p
}

func tileOf(e tables.Extend) ui.TileMode {
	switch e {
	case tables.ExtendRepeat:
		return ui.TileModeRepeated
	case tables.ExtendReflect:
		return ui.TileModeMirror
	}
	return ui.TileModeClamp
}

// colorLine makes the colors and stops of a color line, sorted, with the offsets scaled to the
// 0 to 1 range; the returned lo and hi are the original first and last offsets, so that the
// geometry of the gradient can be moved to them.
func (cp *colrPainter) colorLine(stops []tables.ColorStop) (colors []ui.Color, offsets []float64, lo, hi float64) {
	if len(stops) == 0 {
		return nil, nil, 0, 1
	}
	sorted := append([]tables.ColorStop(nil), stops...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].StopOffset < sorted[j].StopOffset })
	lo, hi = fixed214(sorted[0].StopOffset), fixed214(sorted[len(sorted)-1].StopOffset)
	for _, s := range sorted {
		colors = append(colors, cp.color(s.PaletteIndex, fixed214(s.Alpha)))
		if hi == lo {
			offsets = append(offsets, 0)
			continue
		}
		offsets = append(offsets, (fixed214(s.StopOffset)-lo)/(hi-lo))
	}
	if hi == lo {
		offsets[len(offsets)-1] = 1
	}
	if len(colors) == 1 {
		colors, offsets = append(colors, colors[0]), []float64{0, 1}
	}
	return colors, offsets, lo, hi
}

func (cp *colrPainter) shaderPaint(g ui.Shader) *ui.Paint {
	p := ui.NewPaint()
	p.SetShader(g)
	return p
}

// linear paints a linear gradient: p0 to p1, with p2 giving the rotation of the gradient line.
func (cp *colrPainter) linear(stops []tables.ColorStop, ext tables.Extend, x0, y0, x1, y1, x2, y2 int16) {
	colors, offsets, lo, hi := cp.colorLine(stops)
	if colors == nil {
		return
	}
	p0 := ui.Offset{Dx: float64(x0), Dy: float64(y0)}
	p1 := ui.Offset{Dx: float64(x1), Dy: float64(y1)}
	p2 := ui.Offset{Dx: float64(x2), Dy: float64(y2)}
	// The gradient line is p0 to the projection of p1 on the line through p0 perpendicular
	// to p0p2.
	v02 := ui.Offset{Dx: p2.Dx - p0.Dx, Dy: p2.Dy - p0.Dy}
	perp := ui.Offset{Dx: -v02.Dy, Dy: v02.Dx}
	v01 := ui.Offset{Dx: p1.Dx - p0.Dx, Dy: p1.Dy - p0.Dy}
	den := perp.Dx*perp.Dx + perp.Dy*perp.Dy
	end := p1
	if den != 0 {
		k := (v01.Dx*perp.Dx + v01.Dy*perp.Dy) / den
		end = ui.Offset{Dx: p0.Dx + k*perp.Dx, Dy: p0.Dy + k*perp.Dy}
	}
	from := ui.Offset{Dx: p0.Dx + lo*(end.Dx-p0.Dx), Dy: p0.Dy + lo*(end.Dy-p0.Dy)}
	to := ui.Offset{Dx: p0.Dx + hi*(end.Dx-p0.Dx), Dy: p0.Dy + hi*(end.Dy-p0.Dy)}
	if from == to {
		return
	}
	g := ui.NewGradientLinear(from, to, colors, ui.GradientLinearOpts{ColorStops: offsets, TileMode: tileOf(ext)})
	cp.c.DrawPaint(cp.shaderPaint(g))
}

// radial paints a gradient between two circles.
func (cp *colrPainter) radial(stops []tables.ColorStop, ext tables.Extend, x0, y0 int16, r0 float64, x1, y1 int16, r1 float64) {
	colors, offsets, lo, hi := cp.colorLine(stops)
	if colors == nil {
		return
	}
	c0 := ui.Offset{Dx: float64(x0), Dy: float64(y0)}
	c1 := ui.Offset{Dx: float64(x1), Dy: float64(y1)}
	// Move the two circles to the first and last stop.
	lerp := func(a, b, t float64) float64 { return a + (b-a)*t }
	s := ui.Offset{Dx: lerp(c0.Dx, c1.Dx, lo), Dy: lerp(c0.Dy, c1.Dy, lo)}
	e := ui.Offset{Dx: lerp(c0.Dx, c1.Dx, hi), Dy: lerp(c0.Dy, c1.Dy, hi)}
	sr, er := lerp(r0, r1, lo), lerp(r0, r1, hi)
	g := ui.NewGradientRadial(e, er, colors, ui.GradientRadialOpts{ColorStops: offsets, TileMode: tileOf(ext), Focal: &s, FocalRadius: sr})
	cp.c.DrawPaint(cp.shaderPaint(g))
}

// sweep paints a sweep gradient; the angles are counter-clockwise in units of 180 degrees from
// the positive x axis, with y up.
func (cp *colrPainter) sweep(stops []tables.ColorStop, ext tables.Extend, cx, cy int16, start, end tables.Fixed214) {
	colors, offsets, lo, hi := cp.colorLine(stops)
	if colors == nil {
		return
	}
	a0 := (fixed214(start) + 1) * math.Pi
	a1 := (fixed214(end) + 1) * math.Pi
	s, e := a0+lo*(a1-a0), a0+hi*(a1-a0)
	g := ui.NewGradientSweep(ui.Offset{Dx: float64(cx), Dy: float64(cy)}, colors, ui.GradientSweepOpts{ColorStops: offsets, TileMode: tileOf(ext), StartAngle: s, EndAngle: e})
	cp.c.DrawPaint(cp.shaderPaint(g))
}

// compositeBlend maps a COLR composite mode to a blend mode.
func compositeBlend(m tables.CompositeMode) ui.BlendMode {
	switch m {
	case tables.CompositeClear:
		return ui.BlendModeClear
	case tables.CompositeSrc:
		return ui.BlendModeSrc
	case tables.CompositeDest:
		return ui.BlendModeDst
	case tables.CompositeDestOver:
		return ui.BlendModeDstOver
	case tables.CompositeSrcIn:
		return ui.BlendModeSrcIn
	case tables.CompositeDestIn:
		return ui.BlendModeDstIn
	case tables.CompositeSrcOut:
		return ui.BlendModeSrcOut
	case tables.CompositeDestOut:
		return ui.BlendModeDstOut
	case tables.CompositeSrcAtop:
		return ui.BlendModeSrcATop
	case tables.CompositeDestAtop:
		return ui.BlendModeDstATop
	case tables.CompositeXor:
		return ui.BlendModeXor
	case tables.CompositePlus:
		return ui.BlendModePlus
	case tables.CompositeScreen:
		return ui.BlendModeScreen
	case tables.CompositeOverlay:
		return ui.BlendModeOverlay
	case tables.CompositeDarken:
		return ui.BlendModeDarken
	case tables.CompositeLighten:
		return ui.BlendModeLighten
	case tables.CompositeColorDodge:
		return ui.BlendModeColorDodge
	case tables.CompositeColorBurn:
		return ui.BlendModeColorBurn
	case tables.CompositeHardLight:
		return ui.BlendModeHardLight
	case tables.CompositeSoftLight:
		return ui.BlendModeSoftLight
	case tables.CompositeDifference:
		return ui.BlendModeDifference
	case tables.CompositeExclusion:
		return ui.BlendModeExclusion
	case tables.CompositeMultiply:
		return ui.BlendModeMultiply
	case tables.CompositeHslHue:
		return ui.BlendModeHue
	case tables.CompositeHslSaturation:
		return ui.BlendModeSaturation
	case tables.CompositeHslColor:
		return ui.BlendModeColor
	case tables.CompositeHslLuminosity:
		return ui.BlendModeLuminosity
	}
	return ui.BlendModeSrcOver
}
