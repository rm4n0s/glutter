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
// framebuffers are pooled by the device. Layers that need an image filter (blurs,
// backdrops, shader masks) arrive with step 3.5.

// layerRec is a layer that is being drawn.
type layerRec struct {
	// rect is the pixels of the surface that the layer covers, inside the layer
	// under it.
	rect image.Rectangle
	// cmd is the command that began the layer: the transform and clip it is
	// composited with.
	cmd gpu.Command
	// info is the paint the layer is composited with.
	info paintInfo
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

// target is the rectangle of the surface that is being drawn into: the whole
// surface, or the innermost layer.
func (b *frameBuilder) target() image.Rectangle {
	if n := len(b.layers); n > 0 {
		return b.layers[n-1].rect
	}
	return image.Rect(0, 0, b.w, b.h)
}

// beginLayer starts a group. A group that cannot be drawn is skipped (skipping is
// set, and unsupported has been called), and one with nothing to draw is culled.
func (b *frameBuilder) beginLayer(c gpu.Command) error {
	ctx := &shapeContext{b: b, c: c, t: affineOf(&c.Transform)}
	var info paintInfo
	rect := outerPixels(c.Clip.Bounds, b.w, b.h).Intersect(b.target())
	switch v := c.Op.(type) {
	case ui.OpSaveLayer:
		var status paintStatus
		var err error
		info, status, err = ctx.paintWith(&v.Paint, false, ctx.t)
		switch status {
		case paintUnsupported:
			b.skipping = 1
			return err
		case paintNothing:
			b.culled = 1
			return nil
		}
		if v.HasBounds {
			rect = rect.Intersect(deviceBounds(&c.Transform, v.Bounds))
		}
	default:
		var sp paintSpec
		switch v := c.Layer.(type) {
		case *ui.OpacityLayer:
			a := float32(v.Alpha) / 255
			info.color = [4]float32{a, a, a, a}
		case *ui.ColorFilterLayer:
			filterSpec(&sp, v.Filter, false)
			info.color = [4]float32{1, 1, 1, 1}
		case *ui.ClipRectLayer, *ui.ClipRRectLayer, *ui.ClipRSuperellipseLayer, *ui.ClipPathLayer:
			info.color = [4]float32{1, 1, 1, 1}
		default:
			b.skipping = 1
			return b.unsupported("layer " + fmt.Sprintf("%T", c.Layer))
		}
		info.blend = blendFuncs[ui.BlendModeSrcOver]
		info.aa = true
		info.spec = b.intern(sp)
	}
	if rect.Empty() {
		b.culled = 1
		return nil
	}
	b.layers = append(b.layers, &layerRec{rect: rect, cmd: c, info: info})
	b.frame.beginLayer(rect)
	return nil
}

// endLayer ends the innermost layer, and composites it into the one under it.
func (b *frameBuilder) endLayer() error {
	rec := b.layers[len(b.layers)-1]
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
