package explicit

import (
	"fmt"
	"image"
	"math"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// Blurs and shadows (SPEC §9.1). A paint with a mask filter blurs the coverage of its shape: a
// rectangle or rounded rectangle under a transform that only scales and moves is drawn by the
// closed form of its blur in one pass (shaders/shadow.wgsl.inc, the shape of every BoxShadow);
// any other shape is drawn white into a layer, which is blurred, and whose alpha then fills the
// paint. A paint with an image filter draws into a layer that is filtered. Canvas.drawShadow
// draws the two blurred shadows of a path into a layer from which an opaque occluder is cut out.
// Ported from gles/effects.go.

// maskBlur is the blur of a shape that a mask filter, or a shadow, draws.
type maskBlur struct {
	sigma float64
	style int32
}

// effect is what a layer needs besides compositing: passes over its pixels (gpu.FilterPass), how
// its alpha is read, and where it starts.
type effect struct {
	// passes filter the layer's pixels before they are composited; for a layer that starts as a
	// backdrop, they filter the copy of the pixels under it.
	passes []gpu.FilterPass
	// backdrop says that the layer starts as the filtered copy of the pixels under it.
	backdrop bool
	// src is the pixels of the surface that a backdrop is copied from, when they are not the
	// layer's (a filter that moves pixels reads them from elsewhere); empty for the layer's.
	src image.Rectangle
	// mask is -1 when the layer's pixels are composited, else the blur style (0 to 3) by which
	// the layer's alpha, a shape blurred by the passes, is the coverage that the paint fills.
	mask int32
	// drawnOnly composites only the pixels that have something in them, after the filter.
	drawnOnly bool
}

// filteredDraw draws an operation whose paint has an image filter: into a layer without the filter
// and the blend mode, which is then filtered and composited with the blend mode where it has
// drawn something, as Skia does.
func (b *frameBuilder) filteredDraw(c gpu.Command, p *ui.Paint, with func(*ui.Paint) ui.DisplayOp) (bool, error) {
	rect := gpu.PixelRect(c.Clip.Bounds).Intersect(image.Rect(0, 0, b.w, b.h)).Intersect(b.target())
	if rect.Empty() {
		return true, nil
	}
	mode := p.BlendMode()
	if mode == ui.BlendModeDst {
		return true, nil
	}
	filter := p.ImageFilter()
	ctm := gpu.AffineOf(&c.Transform)
	if _, ok := gpu.FilterPasses(filter, ctm, ui.TileModeClamp, 1, 1); !ok {
		return false, b.unsupported(fmt.Sprintf("image filter %T", filter))
	}
	rec := &layerRec{cmd: c, rect: rect, filter: filter, ctm: ctm, tile: ui.TileModeClamp, drawnOnly: true}
	rec.info = paintInfo{color: [4]float32{1, 1, 1, 1}, aa: true}
	var sp gpu.PaintSpec
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

// rectOf is a pixel rectangle as the bounds of a clip.
func rectOf(r image.Rectangle) ui.Rect {
	return ui.Rect{Left: float64(r.Min.X), Top: float64(r.Min.Y), Right: float64(r.Max.X), Bottom: float64(r.Max.Y)}
}

// maskFiltered draws an operation whose paint has a mask filter. handled is false for an operation
// the mask filter does not apply to (images and the like), which is then drawn without it.
func (s *shapeContext) maskFiltered(p *ui.Paint, with func(*ui.Paint) ui.DisplayOp) (handled, ok bool, err error) {
	bounds, isShape := gpu.ShapeBounds(s.c.Op)
	if !isShape {
		return false, false, nil
	}
	mf := p.MaskFilter()
	sigma := mf.Sigma() * s.scaleOf()
	if !(sigma > gpu.MinSigma) || math.IsInf(sigma, 0) {
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
			r, radii = gpu.RectOfRRect(v.RRect), gpu.RadiiOfRRect(v.RRect)
		case ui.OpDrawOval:
			r, radii = v.Rect, gpu.OvalRadii(v.Rect)
		case ui.OpDrawCircle:
			r = ui.NewRectFromCircle(v.Center, v.Radius)
			radii = gpu.OvalRadii(r)
		default:
			analytic = false
		}
		if analytic {
			ok, err := s.blurredRRect(r, radii, p, sigma, gpu.StyleCode(mf.Style()))
			return true, ok, err
		}
	}
	margin := int(math.Ceil(3 * sigma))
	rect := deviceBounds(&s.c.Transform, bounds).Inset(-margin).Intersect(s.outer.Inset(-margin))
	if rect.Empty() {
		return true, true, nil
	}
	info, status, err := s.paint(p, true)
	if status != paintDraw {
		return true, status == paintNothing, err
	}
	rec := &layerRec{cmd: s.c, rect: rect, info: info, blur: &maskBlur{sigma: sigma, style: gpu.StyleCode(mf.Style())}}
	ic := s.c
	ic.Op = with(gpu.WhiteFill(p))
	ic.Clip = gpu.Clip{Bounds: rectOf(rect)}
	s.b.pushLayer(rec)
	ok, err = s.b.draw(ic)
	if err != nil {
		return true, false, err
	}
	return true, ok, s.b.endLayer()
}

// blurredRRect draws a filled rectangle or rounded rectangle blurred by a mask filter, by the
// closed form of its blur. The transform only scales and moves, so the shape in pixels is a
// rounded rectangle too, and the blur is the same in every direction.
func (s *shapeContext) blurredRRect(r ui.Rect, radii [4]ui.Radius, p *ui.Paint, sigma float64, style int32) (bool, error) {
	info, status, err := s.paint(p, true)
	if status != paintDraw {
		return status == paintNothing, err
	}
	parts, valid := gpu.PartsOf(r, radii)
	if !valid || parts.HX <= 0 || parts.HY <= 0 {
		return true, nil
	}
	t := s.t
	sx, sy := math.Abs(t.A), math.Abs(t.D)
	// The corners in pixels, in order from the top left; a flip swaps them.
	dev := gpu.RRectParts{CX: t.A*parts.CX + t.E, CY: t.D*parts.CY + t.F, HX: parts.HX * sx, HY: parts.HY * sy}
	for i, c := range parts.R {
		j := i
		if t.A < 0 {
			j = [4]int{1, 0, 3, 2}[j]
		}
		if t.D < 0 {
			j = [4]int{3, 2, 1, 0}[j]
		}
		dev.R[j] = ui.Radius{X: c.X * sx, Y: c.Y * sy}
	}
	var in gpu.ShapeInstance
	dev.Write(&in, 20)
	in[44], in[45] = float32(sigma), float32(style)
	enc := &gpu.ShapeEncoder{
		Transform: &s.c.Transform, T: s.t, Outer: s.outer, Clip: s.clip, ClipBounds: s.c.Clip.Bounds,
		Paint: gpu.ShapePaint{Color: info.color, Antialias: info.aa},
	}
	enc.Emit = func(in *gpu.ShapeInstance, covers image.Rectangle) {
		if key, ok := s.key(pipeShape, info, covers); ok {
			s.b.add(key, in[:])
		}
	}
	enc.PlacePadded(&in, parts.Rect(), gpu.ShapeKindShadow, 0, 1, math.Ceil(3*sigma)+1)
	return true, nil
}

// shadow draws Canvas.drawShadow: an ambient shadow, the path blurred in place, and a spot
// shadow, the path moved away from a light above it and blurred. The two are drawn into a layer,
// and unless the occluder is transparent the path itself is cut out of the layer.
//
// Deviation: Skia tessellates the shadow geometry; this blurs the path's coverage, as the
// software renderer does, which has the same extent and falloff but not the same pixels.
func (s *shapeContext) shadow(v ui.OpDrawShadow) (bool, error) {
	m := &s.c.Transform
	plan, ok := gpu.PlanShadow(v.Elevation, m, s.scaleOf(), deviceBounds(m, v.Path.GetBounds()))
	if !ok {
		return true, nil
	}
	vis := s.outer
	fixed, _ := gpu.FixedBlendOf(ui.BlendModeSrcOver)
	group := &layerRec{cmd: s.c, rect: vis, info: paintInfo{color: [4]float32{1, 1, 1, 1}, blend: fixed, aa: true, spec: s.b.intern(gpu.PaintSpec{})}}
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
			blend: fixed, aa: true, spec: s.b.intern(gpu.PaintSpec{}),
		}
		rec := &layerRec{cmd: gpu.Command{Kind: gpu.CommandBeginLayer, Transform: *mat, Clip: whole}, rect: rect, info: info, blur: &maskBlur{sigma: sigma}}
		s.b.pushLayer(rec)
		_, err := s.b.draw(gpu.Command{Kind: gpu.CommandDraw, Op: ui.OpDrawPath{Path: v.Path, Paint: *gpu.WhiteFill(ui.NewPaint())}, Transform: *mat, Clip: gpu.Clip{Bounds: rectOf(rect)}})
		if err != nil {
			return err
		}
		return s.b.endLayer()
	}
	if err := paintShadow(m, plan.AmbientSigma, gpu.ShadowAmbientAlpha); err != nil {
		return false, err
	}
	if err := paintShadow(plan.Spot, plan.SpotSigma, gpu.ShadowSpotAlpha); err != nil {
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

// shaderMaskFill paints the shader of a shader mask layer over its mask rectangle, into the layer,
// with the layer's blend mode. The shader's coordinates start at the top left of the rectangle.
func (b *frameBuilder) shaderMaskFill(rec *layerRec) error {
	v := rec.shaderMask
	p := ui.NewPaint()
	p.SetShader(v.Shader)
	p.SetBlendMode(v.BlendMode)
	m := rec.cmd.Transform
	m.Translate(float32(v.MaskRect.Left), float32(v.MaskRect.Top), 0)
	_, err := b.draw(gpu.Command{
		Kind:      gpu.CommandDraw,
		Op:        ui.OpDrawRect{Rect: ui.NewRectFromLTWH(0, 0, v.MaskRect.Width(), v.MaskRect.Height()), Paint: *p},
		Transform: m,
		Clip:      gpu.Clip{Bounds: rectOf(rec.rect)},
	})
	return err
}
