package soft

import (
	"image"

	"github.com/rm4n0s/glutter/ui"
)

// layerRec is a layer that is being drawn: the surface that drawing went to
// before it, which the layer is composited into when it ends.
type layerRec struct {
	prev *image.RGBA
}

// beginLayer starts drawing into a new, transparent surface that covers the
// pixels of rect. Drawing is clipped to the layer's area, which is never larger
// than the surface it goes back to.
func (r *renderer) beginLayer(rect image.Rectangle) *layerRec {
	rect = rect.Intersect(r.img.Rect)
	rec := &layerRec{prev: r.img}
	r.img = image.NewRGBA(rect)
	return rec
}

// endLayer ends the layer, and composites it into the surface it came from,
// inside the clip, with the parameters: filtered by the image filter, scaled by
// the alpha, color filtered, and blended with the mode over every pixel of the
// layer.
func (r *renderer) endLayer(rec *layerRec, cl clip, c compositeParams) error {
	layer := r.img
	r.img = rec.prev
	if c.imageFilter != nil {
		filtered, err := applyImageFilter(layer, c.imageFilter, c.ctm, c.tile)
		if err != nil {
			return err
		}
		layer = filtered
	}
	area := cl.limit(layer.Rect)
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			cov := cl.at(x, y)
			if cov == 0 {
				continue
			}
			s := load(layer, x, y)
			if c.drawnOnly && s[3] == 0 {
				continue
			}
			for k := range s {
				s[k] *= c.alpha
			}
			if c.filter != nil {
				s = c.filter(s)
			}
			r.mix(x, y, s, c.mode, cov, 0)
		}
	}
	return nil
}

// layer draws a layer of the scene and everything under it, in the state st,
// which it does not change.
func (r *renderer) layer(l ui.Layer, st *state) error {
	switch v := l.(type) {
	case *ui.ContainerLayer:
	case *ui.TransformLayer:
		st = &state{m: st.m.Multiply(&v.Matrix), clip: st.clip}
	case *ui.PictureLayer:
		inner := st.clone()
		inner.m.Translate(float32(v.Offset.Dx), float32(v.Offset.Dy), 0)
		return r.picture(v.Picture, inner)
	case *ui.ClipRectLayer:
		return r.clipLayer(l, st, rectPath(v.Rect), v.ClipBehavior)
	case *ui.ClipRRectLayer:
		return r.clipLayer(l, st, rrectPath(v.RRect), v.ClipBehavior)
	case *ui.ClipRSuperellipseLayer:
		return r.clipLayer(l, st, rsuperellipsePath(v.RSuperellipse), v.ClipBehavior)
	case *ui.ClipPathLayer:
		return r.clipLayer(l, st, v.Path, v.ClipBehavior)
	case *ui.OpacityLayer:
		inner := st.clone()
		inner.m.Translate(float32(v.Offset.Dx), float32(v.Offset.Dy), 0)
		return r.groupLayer(l, st, inner, compositeParams{alpha: float32(v.Alpha) / 255, mode: ui.BlendModeSrcOver})
	case *ui.ColorFilterLayer:
		return r.groupLayer(l, st, st, compositeParams{alpha: 1, filter: colorFilterFunc(v.Filter), mode: ui.BlendModeSrcOver})
	case *ui.ImageFilterLayer:
		inner := st.clone()
		inner.m.Translate(float32(v.Offset.Dx), float32(v.Offset.Dy), 0)
		return r.groupLayer(l, st, inner, compositeParams{
			alpha: 1, mode: ui.BlendModeSrcOver, imageFilter: v.Filter, ctm: affineOf(inner.m), tile: ui.TileModeDecal,
		})
	case *ui.BackdropFilterLayer:
		return r.backdropLayer(v, st)
	case *ui.ShaderMaskLayer:
		return r.shaderMaskLayer(v, st)
	default:
		return notImplemented("layer", l)
	}
	return r.children(l, st)
}

// children draws the layers under l.
func (r *renderer) children(l ui.Layer, st *state) error {
	for _, c := range l.Children() {
		if err := r.layer(c, st); err != nil {
			return err
		}
	}
	return nil
}

// groupLayer draws the children of l, with the state inner, into a layer that
// is composited over the surface with the parameters; outer is the state it is
// composited in.
func (r *renderer) groupLayer(l ui.Layer, outer, inner *state, c compositeParams) error {
	rec := r.beginLayer(outer.clip.bounds)
	err := r.children(l, inner)
	if ferr := r.endLayer(rec, outer.clip, c); err == nil {
		err = ferr
	}
	return err
}

// clipLayer draws the children of l clipped to the shape. With the behavior
// ClipNone nothing is clipped; with ClipAntiAliasWithSaveLayer the children are
// drawn into a layer, and the clip is applied to the layer as a whole, so that
// the edge of the clip does not show the background through anti-aliasing.
func (r *renderer) clipLayer(l ui.Layer, st *state, shape *ui.Path, behavior ui.Clip) error {
	switch behavior {
	case ui.ClipNone:
		return r.children(l, st)
	case ui.ClipAntiAliasWithSaveLayer:
		clipped := &state{m: st.m, clip: shapeClip(st, shape, true, false)}
		rec := r.beginLayer(st.clip.bounds)
		err := r.children(l, st)
		if ferr := r.endLayer(rec, clipped.clip, compositeParams{alpha: 1, mode: ui.BlendModeSrcOver}); err == nil {
			err = ferr
		}
		return err
	}
	inner := &state{m: st.m, clip: shapeClip(st, shape, behavior != ui.ClipHardEdge, false)}
	return r.children(l, inner)
}

// backdropLayer filters what has been drawn so far, inside the clip, and draws
// the children over the filtered image. The layer starts as the filtered
// backdrop; it is composited with the layer's blend mode, which is source-over
// by default, so where the filter leaves the backdrop transparent the original
// shows. A blur without a tile mode mirrors the backdrop at its edges, as
// Flutter's backdrop filters do.
func (r *renderer) backdropLayer(v *ui.BackdropFilterLayer, st *state) error {
	rec := r.beginLayer(st.clip.bounds)
	backdrop := image.NewRGBA(r.img.Rect)
	for y := backdrop.Rect.Min.Y; y < backdrop.Rect.Max.Y; y++ {
		for x := backdrop.Rect.Min.X; x < backdrop.Rect.Max.X; x++ {
			i, j := backdrop.PixOffset(x, y), rec.prev.PixOffset(x, y)
			copy(backdrop.Pix[i:i+4], rec.prev.Pix[j:j+4])
		}
	}
	filtered, err := applyImageFilter(backdrop, v.Filter, affineOf(st.m), ui.TileModeMirror)
	if err != nil {
		r.img = rec.prev
		return err
	}
	copy(r.img.Pix, filtered.Pix)
	err = r.children(v, st)
	if ferr := r.endLayer(rec, st.clip, compositeParams{alpha: 1, mode: v.BlendMode}); err == nil {
		err = ferr
	}
	return err
}

// shaderMaskLayer draws the children into a layer, paints the shader over the
// mask rectangle with the blend mode, and composites the layer. The shader's
// coordinates start at the top left of the mask rectangle.
func (r *renderer) shaderMaskLayer(v *ui.ShaderMaskLayer, st *state) error {
	rec := r.beginLayer(st.clip.bounds)
	err := r.children(v, st)
	if err == nil {
		mask := &state{m: st.m.Clone(), clip: newClip(r.img.Rect)}
		mask.m.Translate(float32(v.MaskRect.Left), float32(v.MaskRect.Top), 0)
		p := ui.NewPaint()
		p.SetShader(v.Shader)
		p.SetBlendMode(v.BlendMode)
		err = r.drawPath(rectPath(ui.NewRectFromLTWH(0, 0, v.MaskRect.Width(), v.MaskRect.Height())), p, mask, false)
	}
	if ferr := r.endLayer(rec, st.clip, compositeParams{alpha: 1, mode: ui.BlendModeSrcOver}); err == nil {
		err = ferr
	}
	return err
}
