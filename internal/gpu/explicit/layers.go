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
// mode. The textures are pooled. Ported from gles/layers.go; the layers that are filtered when
// composited (image filters, backdrop filters, shader masks and blurred masks) come with step 20.6.

// layerRec is a layer that is being drawn.
type layerRec struct {
	// rect is the pixels of the surface that the layer covers.
	rect image.Rectangle
	// cmd is the command that began the layer: the transform and clip it is composited with.
	cmd gpu.Command
	// info is the paint the layer is composited with.
	info paintInfo
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
		if v.Paint.ImageFilter() != nil {
			b.skipping = 1
			return b.unsupported("image filter")
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
			b.skipping = 1
			return b.unsupported("image filter")
		case *ui.BackdropFilterLayer:
			b.skipping = 1
			return b.unsupported("backdrop filter")
		case *ui.ShaderMaskLayer:
			b.skipping = 1
			return b.unsupported("shader mask")
		case *ui.ClipRectLayer, *ui.ClipRRectLayer, *ui.ClipRSuperellipseLayer, *ui.ClipPathLayer, *ui.TransformLayer, *ui.ContainerLayer:
			rec.info.color = [4]float32{1, 1, 1, 1}
		default:
			b.skipping = 1
			return b.unsupported("layer " + fmt.Sprintf("%T", c.Layer))
		}
		if rec.info.blend == (gpu.FixedBlend{}) {
			rec.info.blend, _ = gpu.FixedBlendOf(ui.BlendModeSrcOver)
		}
		rec.info.aa = true
		rec.info.spec = b.intern(sp)
	}
	if rect.Empty() {
		b.culled = 1
		return nil
	}
	rec.rect = rect
	b.layers = append(b.layers, rec)
	b.calls = append(b.calls, drawCall{callKey: callKey{op: opBeginLayer}, rect: rect})
	return nil
}

// deviceBounds returns the pixels that a rectangle covers under the transform.
func deviceBounds(m *ui.Matrix4, r ui.Rect) image.Rectangle {
	b := m.TransformRect(r)
	return gpu.PixelRect(b)
}

// endLayer ends the innermost layer, and composites it into the one under it.
func (b *frameBuilder) endLayer() error {
	rec := b.layers[len(b.layers)-1]
	b.layers = b.layers[:len(b.layers)-1]
	return b.composite(rec)
}

// composite draws the layer into the one under it.
func (b *frameBuilder) composite(rec *layerRec) error {
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
		// Nothing of the layer shows: it is only taken off the stack.
		b.calls = append(b.calls, drawCall{callKey: key, count: 0, rect: rec.rect})
		return nil
	}
	in[0], in[1], in[2], in[3] = float32(region.Min.X), float32(region.Min.Y), float32(region.Max.X), float32(region.Max.Y)
	copy(in[4:8], rec.info.color[:])
	cb := c.Clip.Bounds
	in[8], in[9], in[10], in[11] = float32(cb.Left), float32(cb.Top), float32(cb.Right), float32(cb.Bottom)
	if rec.info.spec.ReadBlend != 0 {
		b.seq++
		key.seq = b.seq
	}
	first := b.items(pipeLayer)
	b.calls = append(b.calls, drawCall{callKey: key, first: first, count: 1, rect: rec.rect})
	b.data[pipeLayer] = append(b.data[pipeLayer], in[:]...)
	return nil
}
