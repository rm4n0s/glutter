package gles

import (
	"fmt"
	"image"
	"math"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// Layers (SPEC §9.1): a SaveLayer, an opacity or color filter layer, and a clip
// layer that smooths its edge as a whole, are drawn into an offscreen framebuffer
// that covers the rectangle of the surface they can touch, and composited into
// the layer under them with their alpha, color filter and blend mode. The
// framebuffers are pooled by the device. A layer can also be filtered when it is
// composited (image filters, effects.go), start as a blurred copy of what is under
// it (a backdrop filter), or get a shader painted over it (a shader mask).

// effect is what a layer needs besides compositing: passes over its pixels (filters.go),
// how its alpha is read, and where it starts.
type effect struct {
	// passes filter the layer's pixels before they are composited; for a layer that
	// starts as a backdrop, they filter the copy of the pixels under it.
	passes []filterPass
	// backdrop says that the layer starts as the filtered copy of the pixels under it.
	backdrop bool
	// mask is -1 when the layer's pixels are composited, else the blur style (0 to 3)
	// by which the layer's alpha, a shape blurred by the passes, is the coverage
	// that the paint fills.
	mask int32
	// drawnOnly composites only the pixels that have something in them, after the
	// filter.
	drawnOnly bool
}

// maskBlur is the blur of a shape that a mask filter, or a shadow, draws.
type maskBlur struct {
	sigma float64
	style int32
}

// layerRec is a layer that is being drawn.
type layerRec struct {
	// rect is the pixels of the surface that the layer covers.
	rect image.Rectangle
	// cmd is the command that began the layer: the transform and clip it is
	// composited with.
	cmd gpu.Command
	// info is the paint the layer is composited with.
	info paintInfo
	// filter filters the layer's pixels when it is composited, under the transform
	// ctm; tile is the mode a blur without one gets.
	filter ui.ImageFilter
	ctm    affine2D
	tile   ui.TileMode
	// blur makes the layer's alpha the coverage of a blurred shape.
	blur *maskBlur
	// drawnOnly, backdrop and shaderMask are the effect fields of the same name.
	drawnOnly  bool
	backdrop   []filterPass
	isBackdrop bool
	shaderMask *ui.ShaderMaskLayer
}

// deviceBounds returns the pixels that a rectangle covers under the transform.
func deviceBounds(m *ui.Matrix4, r ui.Rect) image.Rectangle {
	b := m.TransformRect(r)
	if !b.IsFinite() {
		return image.Rectangle{}
	}
	clamp := func(v float64) int { return int(math.Max(-1e9, math.Min(1e9, v))) }
	return image.Rect(clamp(math.Floor(b.Left)), clamp(math.Floor(b.Top)), clamp(math.Ceil(b.Right)), clamp(math.Ceil(b.Bottom)))
}

// rectOf is a pixel rectangle as the bounds of a clip.
func rectOf(r image.Rectangle) ui.Rect {
	return ui.Rect{Left: float64(r.Min.X), Top: float64(r.Min.Y), Right: float64(r.Max.X), Bottom: float64(r.Max.Y)}
}

// target is the rectangle of the surface that is being drawn into: the whole
// surface, or the innermost layer.
func (b *frameBuilder) target() image.Rectangle {
	if n := len(b.layers); n > 0 {
		return b.layers[n-1].rect
	}
	return image.Rect(0, 0, b.w, b.h)
}

// pushLayer starts drawing a layer.
func (b *frameBuilder) pushLayer(rec *layerRec) {
	b.layers = append(b.layers, rec)
	var eff *effect
	if rec.isBackdrop {
		eff = &effect{backdrop: true, passes: rec.backdrop, mask: -1}
	}
	b.frame.beginLayer(rec.rect, eff)
}

// beginLayer starts a group. A group that cannot be drawn is skipped (skipping is
// set, and unsupported has been called), and one with nothing to draw is culled.
func (b *frameBuilder) beginLayer(c gpu.Command) error {
	ctx := &shapeContext{b: b, c: c, t: affineOf(&c.Transform)}
	rec := &layerRec{cmd: c}
	rect := outerPixels(c.Clip.Bounds, b.w, b.h).Intersect(b.target())
	var sp paintSpec
	switch v := c.Op.(type) {
	case ui.OpSaveLayer:
		info, status, err := ctx.paintWith(&v.Paint, false, ctx.t)
		switch status {
		case paintUnsupported:
			b.skipping = 1
			return err
		case paintNothing:
			b.culled = 1
			return nil
		}
		rec.info = info
		if v.HasBounds {
			rect = rect.Intersect(deviceBounds(&c.Transform, v.Bounds))
		}
		if f := v.Paint.ImageFilter(); f != nil {
			rec.filter, rec.ctm, rec.tile = f, ctx.t, ui.TileModeClamp
		}
	default:
		switch v := c.Layer.(type) {
		case *ui.OpacityLayer:
			a := float32(v.Alpha) / 255
			rec.info.color = [4]float32{a, a, a, a}
		case *ui.ColorFilterLayer:
			filterSpec(&sp, v.Filter, false)
			rec.info.color = [4]float32{1, 1, 1, 1}
		case *ui.ImageFilterLayer:
			rec.info.color = [4]float32{1, 1, 1, 1}
			if v.Filter != nil {
				rec.filter, rec.ctm, rec.tile = v.Filter, translation(v.Offset.Dx, v.Offset.Dy).then(ctx.t), ui.TileModeDecal
			}
		case *ui.BackdropFilterLayer:
			rec.info.color = [4]float32{1, 1, 1, 1}
			if !b.setBlend(&rec.info, &sp, v.BlendMode) {
				b.skipping = 1
				return b.unsupported("blend mode " + v.BlendMode.String())
			}
			rec.isBackdrop = true
			if v.Filter != nil {
				rec.filter, rec.ctm, rec.tile = v.Filter, ctx.t, ui.TileModeMirror
			}
		case *ui.ShaderMaskLayer:
			rec.info.color = [4]float32{1, 1, 1, 1}
			rec.shaderMask = v
		case *ui.ClipRectLayer, *ui.ClipRRectLayer, *ui.ClipRSuperellipseLayer, *ui.ClipPathLayer:
			rec.info.color = [4]float32{1, 1, 1, 1}
		default:
			b.skipping = 1
			return b.unsupported("layer " + fmt.Sprintf("%T", c.Layer))
		}
		if rec.info.blend == (blendFunc{}) {
			rec.info.blend = blendFuncs[ui.BlendModeSrcOver]
		}
		rec.info.aa = true
		rec.info.spec = b.intern(sp)
	}
	if rec.filter != nil {
		if _, ok := filterPasses(rec.filter, rec.ctm, rec.tile, 1, 1); !ok {
			b.skipping = 1
			return b.unsupported(fmt.Sprintf("image filter %T", rec.filter))
		}
	}
	if rect.Empty() {
		b.culled = 1
		return nil
	}
	rec.rect = rect
	if rec.isBackdrop {
		// The backdrop is the copy of the pixels under the layer, filtered.
		rec.backdrop, _ = filterPasses(rec.filter, rec.ctm, rec.tile, rect.Dx(), rect.Dy())
		rec.filter = nil
	}
	b.pushLayer(rec)
	return nil
}

// setBlend sets how a blend mode is drawn into the paint info and spec: through the
// fixed-function blend if it can, else through the shader that reads the pixels under
// the draw. false says that the mode is not known.
func (b *frameBuilder) setBlend(info *paintInfo, sp *paintSpec, mode ui.BlendMode) bool {
	if mode == ui.BlendModeUnspecified {
		mode = ui.BlendModeSrcOver
	}
	if blend, ok := blendFuncs[mode]; ok {
		info.blend = blend
		return true
	}
	idx, ok := blendIndex(mode)
	if !ok {
		return false
	}
	info.blend = replaceBlend
	sp.readBlend = int32(idx + 1)
	return true
}

// endLayer ends the innermost layer, and composites it into the one under it.
func (b *frameBuilder) endLayer() error {
	rec := b.layers[len(b.layers)-1]
	if rec.shaderMask != nil {
		if err := b.shaderMaskFill(rec); err != nil {
			return err
		}
	}
	b.layers = b.layers[:len(b.layers)-1]
	c := rec.cmd
	var in [12]float32
	region := rec.rect
	var mask *clipMask
	if !c.Clip.IsRect() {
		if b.res == nil {
			region = image.Rectangle{}
			if err := b.unsupported("clip shape"); err != nil {
				return err
			}
		} else {
			m, empty, err := b.res.clipMask(c.Clip, b.w, b.h)
			if err != nil {
				return err
			}
			if empty {
				region = image.Rectangle{}
			}
			mask = m
		}
	}
	region = region.Intersect(outerPixels(c.Clip.Bounds, b.w, b.h)).Intersect(b.target())
	key := callKey{blend: rec.info.blend, scissor: [4]int{region.Min.X, region.Min.Y, region.Dx(), region.Dy()}, clip: mask, spec: rec.info.spec}
	if region.Empty() {
		// Nothing of the layer shows: it is only taken off the stack.
		b.frame.composite(key, in[:0], rec.rect)
		b.frame.calls[len(b.frame.calls)-1].count = 0
		return nil
	}
	if eff := b.layerEffect(rec); eff != nil {
		key.eff = eff
	}
	in[0], in[1], in[2], in[3] = float32(region.Min.X), float32(region.Min.Y), float32(region.Max.X), float32(region.Max.Y)
	copy(in[4:8], rec.info.color[:])
	cb := c.Clip.Bounds
	in[8], in[9], in[10], in[11] = float32(cb.Left), float32(cb.Top), float32(cb.Right), float32(cb.Bottom)
	if rec.info.spec.readBlend != 0 {
		b.seq++
		key.seq = b.seq
	}
	b.frame.composite(key, in[:], rec.rect)
	return nil
}

// layerEffect is what the composite of a layer does besides drawing its pixels.
func (b *frameBuilder) layerEffect(rec *layerRec) *effect {
	w, h := rec.rect.Dx(), rec.rect.Dy()
	eff := &effect{mask: -1, drawnOnly: rec.drawnOnly}
	switch {
	case rec.blur != nil:
		eff.passes = blurPasses(rec.blur.sigma, rec.blur.sigma, ui.TileModeDecal, w, h)
		eff.mask = rec.blur.style
	case rec.filter != nil:
		eff.passes, _ = filterPasses(rec.filter, rec.ctm, rec.tile, w, h)
	}
	if len(eff.passes) == 0 && eff.mask < 0 && !eff.drawnOnly {
		return nil
	}
	return eff
}

// shaderMaskFill paints the shader of a shader mask layer over its mask rectangle,
// into the layer, with the layer's blend mode. The shader's coordinates start at the
// top left of the rectangle.
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
