package explicit

import (
	"fmt"
	"image"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// Layers (SPEC §9.1): a SaveLayer, an opacity or color filter layer, and a clip layer that smooths
// its edge as a whole, are drawn into an offscreen texture that covers the rectangle of the surface
// they can touch, and composited into the layer under them with their alpha, color filter and blend
// mode. The textures are pooled. A layer can also be filtered when it is composited (image filters,
// effects.go), start as a blurred copy of what is under it (a backdrop filter), or get a shader
// painted over it (a shader mask). Ported from gles/layers.go.

// layerRec is a layer that is being drawn.
type layerRec struct {
	// rect is the pixels of the surface that the layer covers.
	rect image.Rectangle
	// cmd is the command that began the layer: the transform and clip it is composited with.
	cmd gpu.Command
	// info is the paint the layer is composited with.
	info paintInfo
	// filter filters the layer's pixels when it is composited, under the transform ctm; tile is
	// the mode a blur without one gets.
	filter ui.ImageFilter
	ctm    gpu.Affine2D
	tile   ui.TileMode
	// blur makes the layer's alpha the coverage of a blurred shape.
	blur *maskBlur
	// drawnOnly, backdrop and shaderMask are the effect fields of the same name.
	drawnOnly   bool
	backdrop    []gpu.FilterPass
	backdropSrc image.Rectangle
	isBackdrop  bool
	shaderMask  *ui.ShaderMaskLayer
	// boundary is the layer of a group that the raster cache may keep (a layer that
	// gpu.FlattenOpts.Boundary made a group), and cacheKey its key at the top level.
	boundary ui.Layer
	cacheKey gpu.LayerKey
	keyed    bool
}

// layerKeep says that a layer's pixels are kept in the raster cache for the layer under the key.
type layerKeep struct {
	layer ui.Layer
	key   gpu.LayerKey
}

// layerEntry is a layer of the raster cache.
type layerEntry = gpu.LayerEntry[Texture]

// pushLayer starts drawing a layer.
func (b *frameBuilder) pushLayer(rec *layerRec) {
	b.layers = append(b.layers, rec)
	var eff *effect
	if rec.isBackdrop {
		eff = &effect{backdrop: true, passes: rec.backdrop, src: rec.backdropSrc, mask: -1}
	}
	b.calls = append(b.calls, drawCall{callKey: callKey{op: opBeginLayer, eff: eff}, rect: rec.rect})
}

// beginLayer starts a group. A group that cannot be drawn is skipped (skipping is set, and
// unsupported has been called), and one with nothing to draw is culled.
func (b *frameBuilder) beginLayer(c gpu.Command) error {
	ctx := &shapeContext{b: b, c: c, t: gpu.AffineOf(&c.Transform)}
	rec := &layerRec{cmd: c}
	rect := gpu.PixelRect(c.Clip.Bounds).Intersect(image.Rect(0, 0, b.w, b.h)).Intersect(b.target())
	var sp gpu.PaintSpec
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
			sp.SetColorFilter(v.Filter, false)
			rec.info.color = [4]float32{1, 1, 1, 1}
		case *ui.ImageFilterLayer:
			rec.info.color = [4]float32{1, 1, 1, 1}
			if v.Filter != nil {
				rec.filter, rec.ctm, rec.tile = v.Filter, gpu.Translation(v.Offset.Dx, v.Offset.Dy).Then(ctx.t), ui.TileModeDecal
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
		case *ui.TransformLayer, *ui.ContainerLayer:
			// A group that the raster cache asked for.
			rec.info.color = [4]float32{1, 1, 1, 1}
			rec.boundary = c.Layer
		default:
			b.skipping = 1
			return b.unsupported("layer " + fmt.Sprintf("%T", c.Layer))
		}
		if rec.info.blend == (gpu.FixedBlend{}) {
			rec.info.blend, _ = gpu.FixedBlendOf(ui.BlendModeSrcOver)
		}
		rec.info.aa = true
		rec.info.spec = b.intern(&sp)
	}
	if rec.filter != nil {
		if _, ok := gpu.FilterPasses(rec.filter, rec.ctm, rec.tile, 1, 1); !ok {
			b.skipping = 1
			return b.unsupported(fmt.Sprintf("image filter %T", rec.filter))
		}
	}
	if rect.Empty() {
		b.culled = 1
		return nil
	}
	rec.rect = rect
	if rec.boundary != nil && b.cache != nil && len(b.layers) == 0 {
		k := gpu.LayerKeyOf(&c.Transform, rect)
		if e := b.cache.Lookup(rec.boundary, k); e != nil {
			// The layer's pixels are there: draw them, and drop what is inside.
			b.stats.CachedLayersDrawn++
			b.culled = 1
			return b.composite(rec, e)
		}
		rec.cacheKey, rec.keyed = k, true
	}
	if rec.isBackdrop {
		// The backdrop is the copy of the pixels under the layer, filtered.
		if src, pass, ok := gpu.BackdropMatrix(rec.filter, rec.ctm, rect, b.target()); ok {
			// A filter that moves pixels reads them from where it moves them from.
			rec.backdrop, rec.backdropSrc = []gpu.FilterPass{pass}, src
		} else {
			rec.backdrop, _ = gpu.FilterPasses(rec.filter, rec.ctm, rec.tile, rect.Dx(), rect.Dy())
		}
		if len(rec.backdrop) == 0 {
			rec.backdrop = []gpu.FilterPass{{Kind: gpu.PassCopy, OutW: rect.Dx(), OutH: rect.Dy()}}
		}
		rec.filter = nil
	}
	b.pushLayer(rec)
	return nil
}

// retained says that the raster cache has the pixels of the group of the layer l, so that
// beginLayer draws them and drops what is inside: the flattener then does not walk the
// layer. It is asked for the groups at the top level only, with the key beginLayer makes.
func (b *frameBuilder) retained(l ui.Layer, m *ui.Matrix4, clip gpu.Clip) bool {
	rect := gpu.PixelRect(clip.Bounds).Intersect(image.Rect(0, 0, b.w, b.h))
	if rect.Empty() {
		return true // beginLayer culls the group
	}
	return b.cache.Lookup(l, gpu.LayerKeyOf(m, rect)) != nil
}

// deviceBounds returns the pixels that a rectangle covers under the transform.
func deviceBounds(m *ui.Matrix4, r ui.Rect) image.Rectangle {
	b := m.TransformRect(r)
	return gpu.PixelRect(b)
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
	return b.composite(rec, nil)
}

// composite draws the layer into the one under it: from the texture it was drawn into, or from
// the pixels of the raster cache's entry.
func (b *frameBuilder) composite(rec *layerRec, cached *layerEntry) error {
	c := rec.cmd
	var in [12]float32
	region := rec.rect
	var mask *clipMask
	if !c.Clip.IsRect() {
		m, empty, err := b.res.clipMask(c.Clip, b.w, b.h)
		if err != nil {
			return err
		}
		if empty {
			region = image.Rectangle{}
		}
		mask = m
	}
	region = region.Intersect(gpu.PixelRect(c.Clip.Bounds)).Intersect(image.Rect(0, 0, b.w, b.h)).Intersect(b.target())
	key := callKey{op: opComposite, pipe: pipeLayer, blend: rec.info.blend, scissor: region, clip: mask, spec: rec.info.spec}
	if region.Empty() {
		if cached != nil {
			return nil
		}
		// Nothing of the layer shows: it is only taken off the stack.
		b.calls = append(b.calls, drawCall{callKey: key, count: 0, rect: rec.rect})
		return nil
	}
	key.eff = b.layerEffect(rec)
	if rec.keyed && cached == nil && len(b.layers) == 0 {
		key.keep = &layerKeep{layer: rec.boundary, key: rec.cacheKey}
		b.stats.CachedLayersMade++
	}
	in[0], in[1], in[2], in[3] = float32(region.Min.X), float32(region.Min.Y), float32(region.Max.X), float32(region.Max.Y)
	copy(in[4:8], rec.info.color[:])
	cb := c.Clip.Bounds
	in[8], in[9], in[10], in[11] = float32(cb.Left), float32(cb.Top), float32(cb.Right), float32(cb.Bottom)
	if rec.info.spec.ReadBlend != 0 {
		b.seq++
		key.seq = b.seq
	}
	if cached != nil {
		key.op, key.cache = opDrawCached, cached
		rect := cached.Rect
		first := b.items(pipeLayer)
		b.calls = append(b.calls, drawCall{callKey: key, first: first, count: 1, rect: rect})
		b.data[pipeLayer] = append(b.data[pipeLayer], in[:]...)
		return nil
	}
	first := b.items(pipeLayer)
	b.calls = append(b.calls, drawCall{callKey: key, first: first, count: 1, rect: rec.rect})
	b.data[pipeLayer] = append(b.data[pipeLayer], in[:]...)
	return nil
}

// layerEffect is what the composite of a layer does besides drawing its pixels.
func (b *frameBuilder) layerEffect(rec *layerRec) *effect {
	w, h := rec.rect.Dx(), rec.rect.Dy()
	eff := &effect{mask: -1, drawnOnly: rec.drawnOnly}
	switch {
	case rec.blur != nil:
		eff.passes = gpu.BlurPasses(rec.blur.sigma, rec.blur.sigma, ui.TileModeDecal, w, h)
		eff.mask = rec.blur.style
	case rec.filter != nil:
		eff.passes, _ = gpu.FilterPasses(rec.filter, rec.ctm, rec.tile, w, h)
	}
	if len(eff.passes) == 0 && eff.mask < 0 && !eff.drawnOnly {
		return nil
	}
	return eff
}
