package gpu

import (
	"image"
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// Culling (SPEC §9.3, "Culling"): a drawing command whose pixels all lie outside the part of the
// surface its clip reaches does no work in the GPU renderers: no paint, no hash of its geometry, no
// request to the coverage atlas, no instance. The test is conservative, so a command that touches
// the visible area is drawn as it always was.

// OutsideClip reports whether the drawing operation op, drawn under the transform m, certainly
// changes no pixel of visible (the pixels the command's clip reaches inside the target). It takes
// the operation's bounds in its own coordinates (the geometry, widened by the stroke and its
// joins as ShapeBounds does, or the destination rectangle of an image, or the reach of a
// shadow), maps them to the surface, and widens the result by what no bounds can tell: a pixel of
// anti-aliasing on each side, and for a stroke the least width a stroke is drawn with (a hairline
// is a pixel wide) times its miter. It costs one pass over the points of a path and does not hash
// them.
//
// It reports false, so that the renderer goes on as before, for an operation whose bounds are not
// cheap or not known: clips, transforms, paints that cover the whole clip, vertices, atlases,
// anything not finite, and a transform with perspective. A command whose paint has an image filter
// or a mask filter reaches past its bounds, so it reports false for those too.
//
// Flutter: no counterpart (Impeller's Entity::GetCoverage is the same idea)
func OutsideClip(op ui.DisplayOp, m *ui.Matrix4, visible image.Rectangle) bool {
	if visible.Empty() {
		return true
	}
	st := m.Storage()
	if st[3] != 0 || st[7] != 0 || st[15] != 1 {
		return false
	}
	local, pad, ok := cullBounds(op)
	if !ok {
		return false
	}
	dev := m.TransformRect(local)
	if !dev.IsFinite() {
		return false
	}
	dev = ui.Rect{Left: dev.Left - pad, Top: dev.Top - pad, Right: dev.Right + pad, Bottom: dev.Bottom + pad}
	if !dev.IsFinite() {
		return false
	}
	return !PixelRect(dev).Overlaps(visible)
}

// cullBounds returns the bounds of the pixels that op can change, in its own coordinates, and how
// many pixels of the surface to add around them once they are mapped to it. It is false for an
// operation whose bounds are not known or whose paint reaches past them.
func cullBounds(op ui.DisplayOp) (local ui.Rect, pad float64, ok bool) {
	switch v := op.(type) {
	case ui.OpDrawImage:
		if v.Image == nil || v.Image.IsDisposed() || imageFiltered(&v.Paint) {
			return ui.Rect{}, 0, false // the draw reports a disposed image
		}
		w, h := float64(v.Image.Width()), float64(v.Image.Height())
		return ui.NewRectFromLTWH(v.Offset.Dx, v.Offset.Dy, w, h), imagePad, true
	case ui.OpDrawImageRect:
		if v.Image == nil || v.Image.IsDisposed() || imageFiltered(&v.Paint) {
			return ui.Rect{}, 0, false
		}
		return normalized(v.Dst), imagePad, true
	case ui.OpDrawImageNine:
		if v.Image == nil || v.Image.IsDisposed() || imageFiltered(&v.Paint) {
			return ui.Rect{}, 0, false
		}
		return normalized(v.Dst), imagePad, true
	case ui.OpDrawShadow:
		r, ok := ShadowBounds(v)
		return r, shapePad, ok
	}
	r, facts, ok := shapeBounds(op)
	if !ok || facts.filtered {
		return ui.Rect{}, 0, false
	}
	pad = shapePad
	if facts.stroked {
		// A stroke thinner than a pixel is drawn one pixel wide (ShapeEncoder.StrokeHalfWidth),
		// whatever the scale: half a pixel, times a miter, beyond the geometry.
		pad += 0.5 * max(math.Sqrt2, facts.miter)
	}
	return r, pad, true
}

// imageFiltered says that an image's paint has a filter, which the draw handles as a layer.
func imageFiltered(p *ui.Paint) bool { return p.ImageFilter() != nil || p.MaskFilter() != nil }

const (
	// shapePad is the pixels that the anti-aliased edge of a shape reaches beyond its bounds,
	// and imagePad the same for the filtered edge of an image.
	shapePad = 1.5
	imagePad = 2
)

// normalized returns the rectangle with its edges in order.
func normalized(r ui.Rect) ui.Rect {
	return ui.Rect{Left: min(r.Left, r.Right), Top: min(r.Top, r.Bottom), Right: max(r.Left, r.Right), Bottom: max(r.Top, r.Bottom)}
}
