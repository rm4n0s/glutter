package gles

import (
	"fmt"
	"math"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// Blurs and shadows (SPEC §9.1). A paint with a mask filter blurs the coverage of its
// shape: a rectangle or rounded rectangle under a transform that only scales and
// moves is drawn by the closed form of its blur in one pass (shaders/shadow.glsl, the
// shape of every BoxShadow); any other shape is drawn white into a layer, which is
// blurred, and whose alpha then fills the paint. A paint with an image filter draws
// into a layer that is filtered. Canvas.drawShadow draws the two blurred shadows of
// a path into a layer from which an opaque occluder is cut out.

// paintAccess returns the paint of a drawing operation and the function that makes
// the same operation with another paint; nil for the operations without one.
func paintAccess(op ui.DisplayOp) (*ui.Paint, func(*ui.Paint) ui.DisplayOp) {
	switch v := op.(type) {
	case ui.OpDrawPath:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawRect:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawRRect:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawRSuperellipse:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawDRRect:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawOval:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawCircle:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawArc:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawLine:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawPoints:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawImage:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawImageRect:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawImageNine:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawAtlas:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawVertices:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	}
	return nil, nil
}

// whiteFill is the paint that draws only the coverage of a shape: opaque white,
// source-over, with the geometry of p (style, stroke, anti-aliasing).
func whiteFill(p *ui.Paint) *ui.Paint {
	q := *p
	q.SetMaskFilter(nil)
	q.SetImageFilter(nil)
	q.SetShader(nil)
	q.SetColorFilter(nil)
	q.SetInvertColors(false)
	q.SetColor(ui.NewColor(0xFFFFFFFF))
	q.SetBlendMode(ui.BlendModeSrcOver)
	return &q
}

// filteredDraw draws an operation whose paint has an image filter: into a layer
// without the filter and the blend mode, which is then filtered and composited with
// the blend mode where it has drawn something, as Skia does.
func (b *frameBuilder) filteredDraw(c gpu.Command, p *ui.Paint, with func(*ui.Paint) ui.DisplayOp) (bool, error) {
	rect := outerPixels(c.Clip.Bounds, b.w, b.h).Intersect(b.target())
	if rect.Empty() {
		return true, nil
	}
	mode := p.BlendMode()
	if mode == ui.BlendModeDst {
		return true, nil
	}
	filter := p.ImageFilter()
	ctm := affineOf(&c.Transform)
	if _, ok := filterPasses(filter, ctm, ui.TileModeClamp, 1, 1); !ok {
		return false, b.unsupported(fmt.Sprintf("image filter %T", filter))
	}
	rec := &layerRec{cmd: c, rect: rect, filter: filter, ctm: ctm, tile: ui.TileModeClamp, drawnOnly: true}
	rec.info = paintInfo{color: [4]float32{1, 1, 1, 1}, aa: true}
	var sp paintSpec
	if !b.setBlend(&rec.info, &sp, mode) {
		return false, b.unsupported("blend mode " + mode.String())
	}
	rec.info.spec = b.intern(sp)
	inner := *p
	inner.SetImageFilter(nil)
	inner.SetBlendMode(ui.BlendModeSrcOver)
	ic := c
	ic.Op = with(&inner)
	ic.Clip = gpu.Clip{Bounds: rectOf(rect)}
	b.pushLayer(rec)
	ok, err := b.draw(ic)
	if err != nil {
		return false, err
	}
	return ok, b.endLayer()
}

// opBounds returns the bounds of what a shape-drawing operation covers in its own
// coordinates, strokes included.
func opBounds(op ui.DisplayOp) (ui.Rect, bool) { return gpu.ShapeBounds(op) }

// styleCode is the number of a blur style in the shaders.
func styleCode(s ui.BlurStyle) int32 {
	switch s {
	case ui.BlurStyleSolid:
		return 1
	case ui.BlurStyleOuter:
		return 2
	case ui.BlurStyleInner:
		return 3
	}
	return 0
}

// maskFiltered draws an operation whose paint has a mask filter. handled is false
// for an operation the mask filter does not apply to (images and the like), which is
// then drawn without it.
func (s *shapeContext) maskFiltered(p *ui.Paint, with func(*ui.Paint) ui.DisplayOp) (handled, ok bool, err error) {
	bounds, isShape := opBounds(s.c.Op)
	if !isShape {
		return false, false, nil
	}
	mf := p.MaskFilter()
	sigma := mf.Sigma() * s.scale()
	if !(sigma > minSigma) || math.IsInf(sigma, 0) {
		return false, false, nil
	}
	if s.c.Transform.Storage()[1] == 0 && s.c.Transform.Storage()[4] == 0 && p.Style() != ui.PaintingStyleStroke {
		var r ui.Rect
		var radii [4]ui.Radius
		analytic := true
		switch v := s.c.Op.(type) {
		case ui.OpDrawRect:
			r = v.Rect
		case ui.OpDrawRRect:
			r, radii = rectOfRRect(v.RRect), radiiOf(v.RRect)
		case ui.OpDrawOval:
			r, radii = v.Rect, ovalRadii(v.Rect)
		case ui.OpDrawCircle:
			r = ui.NewRectFromCircle(v.Center, v.Radius)
			radii = ovalRadii(r)
		default:
			analytic = false
		}
		if analytic {
			ok, err := s.blurredRRect(r, radii, p, sigma, styleCode(mf.Style()))
			return true, ok, err
		}
	}
	margin := int(math.Ceil(3 * sigma))
	vis := s.outer
	rect := deviceBounds(&s.c.Transform, bounds).Inset(-margin).Intersect(vis.Inset(-margin))
	if rect.Empty() {
		return true, true, nil
	}
	info, status, err := s.paint(p)
	if status != paintDraw {
		return true, status == paintNothing, err
	}
	rec := &layerRec{cmd: s.c, rect: rect, info: info, blur: &maskBlur{sigma: sigma, style: styleCode(mf.Style())}}
	ic := s.c
	ic.Op = with(whiteFill(p))
	ic.Clip = gpu.Clip{Bounds: rectOf(rect)}
	s.b.pushLayer(rec)
	ok, err = s.b.draw(ic)
	if err != nil {
		return true, false, err
	}
	return true, ok, s.b.endLayer()
}

// blurredRRect draws a filled rectangle or rounded rectangle blurred by a mask
// filter, by the closed form of its blur. The transform only scales and moves, so
// the shape in pixels is a rounded rectangle too, and the blur is the same in every
// direction.
func (s *shapeContext) blurredRRect(r ui.Rect, radii [4]ui.Radius, p *ui.Paint, sigma float64, style int32) (bool, error) {
	info, status, err := s.paint(p)
	if status != paintDraw {
		return status == paintNothing, err
	}
	parts, valid := partsOf(r, radii)
	if !valid || parts.hx <= 0 || parts.hy <= 0 {
		return true, nil
	}
	t := s.t
	sx, sy := math.Abs(t.a), math.Abs(t.d)
	// The corners in pixels, in order from the top left; a flip swaps them.
	dev := rrectParts{cx: t.a*parts.cx + t.e, cy: t.d*parts.cy + t.f, hx: parts.hx * sx, hy: parts.hy * sy}
	for i, c := range parts.r {
		j := i
		if t.a < 0 {
			j = [4]int{1, 0, 3, 2}[j]
		}
		if t.d < 0 {
			j = [4]int{3, 2, 1, 0}[j]
		}
		dev.r[j] = ui.Radius{X: c.X * sx, Y: c.Y * sy}
	}
	var in instance
	dev.write(&in, 20)
	in[44], in[45] = float32(sigma), float32(style)
	local := ui.Rect{Left: parts.cx - parts.hx, Top: parts.cy - parts.hy, Right: parts.cx + parts.hx, Bottom: parts.cy + parts.hy}
	return s.placePadded(&in, local, kindShadow, 0, info, 1, math.Ceil(3*sigma)+1)
}

// blurRadiusToSigma is Skia's conversion of a blur radius to a deviation.
func blurRadiusToSigma(radius float64) float64 {
	if radius > 0 {
		return 0.57735*radius + 0.5
	}
	return 0
}

// Skia's constants for the shadows of Flutter's physical shapes, as in the software
// renderer.
const (
	shadowAmbientAlpha  = 0.039
	shadowSpotAlpha     = 0.25
	shadowLightHeight   = 600.0
	shadowLightRadius   = 800.0
	shadowAmbientFactor = 0.5
	shadowMaxAmbient    = 300.0
)

// shadow draws Canvas.drawShadow: an ambient shadow, the path blurred in place, and a
// spot shadow, the path moved away from a light above it and blurred. The two are
// drawn into a layer, and unless the occluder is transparent the path itself is cut
// out of the layer.
//
// Deviation: Skia tessellates the shadow geometry; this blurs the path's coverage,
// as the software renderer does, which has the same extent and falloff but not the
// same pixels.
func (s *shapeContext) shadow(v ui.OpDrawShadow) (bool, error) {
	m := &s.c.Transform
	scale := s.scale()
	z := v.Elevation * scale
	lightZ := shadowLightHeight * scale
	if !(z > 0) || z >= lightZ {
		return true, nil
	}
	devBounds := deviceBounds(m, v.Path.GetBounds())
	if devBounds.Empty() {
		return true, nil
	}
	ambientSigma := blurRadiusToSigma(math.Min(z*shadowAmbientFactor, shadowMaxAmbient))
	spotSigma := blurRadiusToSigma(shadowLightRadius * scale * z / (lightZ - z))
	// The spot shadow moves down and grows, away from a light above the shape.
	spotScale := lightZ / (lightZ - z)
	dy := z * (shadowLightHeight*scale + float64(devBounds.Dy())/2) / (lightZ - z)
	cx, cy := float64(devBounds.Min.X+devBounds.Max.X)/2, float64(devBounds.Min.Y+devBounds.Max.Y)/2
	spot := ui.NewMatrix4TranslationValues(float32(cx*(1-spotScale)), float32(cy*(1-spotScale)+dy), 0)
	spot.Scale(float32(spotScale), float32(spotScale), 1)
	spotM := spot.Multiply(m)

	vis := s.outer
	group := &layerRec{cmd: s.c, rect: vis, info: paintInfo{color: [4]float32{1, 1, 1, 1}, blend: blendFuncs[ui.BlendModeSrcOver], aa: true, spec: s.b.intern(paintSpec{})}}
	s.b.pushLayer(group)
	whole := gpu.Clip{Bounds: rectOf(vis)}
	paintShadow := func(mat *ui.Matrix4, sigma, alpha float64) error {
		margin := int(math.Ceil(3 * sigma))
		rect := deviceBounds(mat, v.Path.GetBounds()).Inset(-margin).Intersect(vis.Inset(-margin))
		if rect.Empty() {
			return nil
		}
		col := v.Color
		col.A = float32(float64(col.A) * alpha)
		a := float32(col.Alpha()) / 255
		info := paintInfo{
			color: [4]float32{float32(col.Red()) / 255 * a, float32(col.Green()) / 255 * a, float32(col.Blue()) / 255 * a, a},
			blend: blendFuncs[ui.BlendModeSrcOver], aa: true, spec: s.b.intern(paintSpec{}),
		}
		rec := &layerRec{cmd: gpu.Command{Kind: gpu.CommandBeginLayer, Transform: *mat, Clip: whole}, rect: rect, info: info, blur: &maskBlur{sigma: sigma}}
		s.b.pushLayer(rec)
		_, err := s.b.draw(gpu.Command{Kind: gpu.CommandDraw, Op: ui.OpDrawPath{Path: v.Path, Paint: *whiteFill(ui.NewPaint())}, Transform: *mat, Clip: gpu.Clip{Bounds: rectOf(rect)}})
		if err != nil {
			return err
		}
		return s.b.endLayer()
	}
	if err := paintShadow(m, ambientSigma, shadowAmbientAlpha); err != nil {
		return false, err
	}
	if err := paintShadow(spotM, spotSigma, shadowSpotAlpha); err != nil {
		return false, err
	}
	if !v.TransparentOccluder {
		cut := ui.NewPaint()
		cut.SetColor(ui.NewColor(0xFFFFFFFF))
		cut.SetBlendMode(ui.BlendModeDstOut)
		if _, err := s.b.draw(gpu.Command{Kind: gpu.CommandDraw, Op: ui.OpDrawPath{Path: v.Path, Paint: *cut}, Transform: *m, Clip: whole}); err != nil {
			return false, err
		}
	}
	return true, s.b.endLayer()
}
