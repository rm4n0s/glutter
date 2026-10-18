package gles

import (
	"image"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// General paths (SPEC §9.1): the software rasterizer turns the shape into an
// 8-bit coverage mask, which is cached in the coverage atlas by the shape, the
// transform's linear part and its fractional translation, and the paint's stroke.
// The shape is cut into tiles of at most gpu.MaxAtlasTile pixels, anchored at the
// origin of the transformed shape's space, so that a shape drawn again at another
// whole-pixel offset, or only partly visible, finds its tiles. The GPU composites
// each tile with the paint's color.

// maskKey identifies a tile of the coverage of a shape.
type maskKey struct {
	geom   uint64     // a hash of the shape's geometry
	style  uint64     // a hash of the paint's stroke and anti-aliasing
	lin    [4]float32 // the linear part of the transform
	fx, fy float32    // the fractional part of its translation
	tx, ty int32      // the tile, in tiles from the origin of the shape's space
}

// maskTile is a tile of coverage to draw: where it goes on the surface and where
// its texels are.
type maskTile struct {
	tex            *Texture // the atlas page
	x0, y0, x1, y1 int      // the pixels of the surface it covers
	ax, ay         int      // its top left texel on the page
}

// opShape describes what an operation draws, for the atlas: a hash of its
// geometry, its bounds in its own coordinates, and its paint.
type opShape struct {
	geom   uint64
	bounds ui.Rect
	paint  *ui.Paint
	stroke bool // the outline is stroked, not filled
}

// shapeOf describes a shape-drawing operation; false for any other.
func shapeOf(op ui.DisplayOp) (opShape, bool) {
	h := newHasher()
	rr := func(left, top, right, bottom float64, radii ...float64) {
		h.f64(left)
		h.f64(top)
		h.f64(right)
		h.f64(bottom)
		for _, r := range radii {
			h.f64(r)
		}
	}
	switch v := op.(type) {
	case ui.OpDrawPath:
		h.i(1)
		h.u64(v.Path.Fingerprint())
		return opShape{h.sum(), v.Path.GetBounds(), &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawRect:
		h.i(2)
		rr(v.Rect.Left, v.Rect.Top, v.Rect.Right, v.Rect.Bottom)
		return opShape{h.sum(), v.Rect, &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawRRect:
		h.i(3)
		r := v.RRect
		rr(r.Left, r.Top, r.Right, r.Bottom, r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY, r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY)
		return opShape{h.sum(), v.RRect.OuterRect(), &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawRSuperellipse:
		h.i(4)
		r := v.RSuperellipse
		rr(r.Left, r.Top, r.Right, r.Bottom, r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY, r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY)
		return opShape{h.sum(), v.RSuperellipse.OuterRect(), &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawDRRect:
		h.i(5)
		o, in := v.Outer, v.Inner
		rr(o.Left, o.Top, o.Right, o.Bottom, o.TlRadiusX, o.TlRadiusY, o.TrRadiusX, o.TrRadiusY, o.BrRadiusX, o.BrRadiusY, o.BlRadiusX, o.BlRadiusY)
		rr(in.Left, in.Top, in.Right, in.Bottom, in.TlRadiusX, in.TlRadiusY, in.TrRadiusX, in.TrRadiusY, in.BrRadiusX, in.BrRadiusY, in.BlRadiusX, in.BlRadiusY)
		return opShape{h.sum(), v.Outer.OuterRect(), &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawOval:
		h.i(6)
		rr(v.Rect.Left, v.Rect.Top, v.Rect.Right, v.Rect.Bottom)
		return opShape{h.sum(), v.Rect, &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawCircle:
		h.i(7)
		rr(v.Center.Dx, v.Center.Dy, v.Radius, 0)
		return opShape{h.sum(), ui.NewRectFromCircle(v.Center, v.Radius), &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawArc:
		h.i(8)
		rr(v.Rect.Left, v.Rect.Top, v.Rect.Right, v.Rect.Bottom, v.StartAngle, v.SweepAngle)
		h.b(v.UseCenter)
		return opShape{h.sum(), v.Rect, &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawLine:
		h.i(9)
		rr(v.P1.Dx, v.P1.Dy, v.P2.Dx, v.P2.Dy)
		b := ui.Rect{Left: math.Min(v.P1.Dx, v.P2.Dx), Top: math.Min(v.P1.Dy, v.P2.Dy), Right: math.Max(v.P1.Dx, v.P2.Dx), Bottom: math.Max(v.P1.Dy, v.P2.Dy)}
		return opShape{h.sum(), b, &v.Paint, true}, true
	case ui.OpDrawPoints:
		h.i(10)
		h.i(int(v.Mode))
		b := ui.Rect{Left: math.Inf(1), Top: math.Inf(1), Right: math.Inf(-1), Bottom: math.Inf(-1)}
		for _, p := range v.Points {
			h.f64(p.Dx)
			h.f64(p.Dy)
			b.Left, b.Top = math.Min(b.Left, p.Dx), math.Min(b.Top, p.Dy)
			b.Right, b.Bottom = math.Max(b.Right, p.Dx), math.Max(b.Bottom, p.Dy)
		}
		return opShape{h.sum(), b, &v.Paint, true}, true
	}
	return opShape{}, false
}

func (h *hasher) sum() uint64 { return uint64(*h) }

// styleKey hashes the parts of a paint that change the shape's coverage.
func styleKey(p *ui.Paint, stroke bool) uint64 {
	h := newHasher()
	h.b(stroke)
	h.b(p.IsAntiAlias())
	if stroke {
		h.f64(p.StrokeWidth())
		h.i(int(p.StrokeCap()))
		h.i(int(p.StrokeJoin()))
		h.f64(p.StrokeMiterLimit())
	}
	return h.sum()
}

// pathMask draws an operation's shape from the coverage atlas, with the paint's
// color. It is for the shapes the analytic pipeline cannot draw: paths, arcs, and
// the strokes and shapes that have no distance field here.
func (s *shapeContext) pathMask(op ui.DisplayOp, p *ui.Paint) (bool, error) {
	info, status, err := s.paint(p)
	if status != paintDraw {
		return status == paintNothing, err
	}
	sh, ok := shapeOf(op)
	if !ok {
		return false, s.b.unsupported("operation " + typeName(op))
	}
	scale := s.scale()
	if !(scale > 0) {
		return true, nil
	}
	// The shape's bounds, widened by what its stroke and its edge can reach.
	margin := 1 / scale
	if sh.stroke {
		hw := sh.paint.StrokeWidth() / 2
		if sh.paint.StrokeWidth() <= 0 {
			hw = 0.5 / scale
		}
		margin += hw * math.Max(sh.paint.StrokeMiterLimit(), 1.5)
	}
	tiles, err := s.b.res.maskTiles(op, sh, &s.c.Transform, margin, s.outer)
	if err != nil {
		return false, s.maskError(err)
	}
	for _, t := range tiles {
		var in [16]float32
		in[0], in[1], in[2], in[3] = float32(t.x0), float32(t.y0), float32(t.x1), float32(t.y1)
		in[4], in[5] = float32(t.ax), float32(t.ay)
		for i := 0; i < 4; i++ {
			in[8+i] = info.color[i]
		}
		copy(in[12:16], s.clip[:])
		s.b.frame.add(callKey{pipe: pipeMask, blend: info.blend, scissor: s.scissor, tex: t.tex, clip: s.mask}, in[:], 1)
	}
	return true, nil
}

// maskError reports an atlas that is full as an unsupported feature when frames
// skip them; otherwise, and for other errors, it returns the error as it is.
func (s *shapeContext) maskError(err error) error {
	if appErr, ok := errors.FromError(err); ok && appErr.Tag == "AtlasFull" && s.b.skip {
		return s.b.unsupported("coverage atlas full")
	}
	return err
}

// maskTiles returns the tiles of coverage to draw for the shape under the
// transform, within the visible pixels, rasterizing and uploading the ones the
// atlas does not have. margin is how far the shape's coverage can reach beyond
// its bounds, in its own units.
func (r *resources) maskTiles(op ui.DisplayOp, sh opShape, m *ui.Matrix4, margin float64, visible image.Rectangle) ([]maskTile, error) {
	st := m.Storage()
	e, f := float64(st[12]), float64(st[13])
	ox, oy := math.Floor(e), math.Floor(f)
	// The mask is made under the transform with only the fractional translation: it
	// is the same wherever on the surface a whole number of pixels takes it.
	mm := m.Clone()
	mm.SetEntry(0, 3, float32(e-ox))
	mm.SetEntry(1, 3, float32(f-oy))
	local := sh.bounds.Inflate(margin)
	dev := mm.TransformRect(local)
	if !dev.IsFinite() {
		return nil, nil
	}
	bounds := image.Rect(int(math.Floor(dev.Left)), int(math.Floor(dev.Top)), int(math.Ceil(dev.Right)), int(math.Ceil(dev.Bottom)))
	// The part of the shape's space that shows.
	view := visible.Sub(image.Pt(int(ox), int(oy))).Intersect(bounds)
	if view.Empty() {
		return nil, nil
	}
	key := maskKey{
		geom: sh.geom, style: styleKey(sh.paint, sh.stroke),
		lin: [4]float32{st[0], st[1], st[4], st[5]},
		fx:  float32(e - ox), fy: float32(f - oy),
	}
	const t = gpu.MaxAtlasTile
	var tiles []maskTile
	for ty := floorDiv(view.Min.Y, t); ty <= floorDiv(view.Max.Y-1, t); ty++ {
		for tx := floorDiv(view.Min.X, t); tx <= floorDiv(view.Max.X-1, t); tx++ {
			key.tx, key.ty = int32(tx), int32(ty)
			entry := r.atlas.Get(key)
			if entry == nil {
				var err error
				if entry, err = r.rasterizeTile(op, mm, key, image.Rect(tx*t, ty*t, (tx+1)*t, (ty+1)*t).Intersect(bounds)); err != nil {
					return nil, err
				}
			}
			if entry.Empty {
				continue
			}
			tiles = append(tiles, maskTile{
				tex: r.pages[entry.Page],
				x0:  int(ox) + entry.OX, y0: int(oy) + entry.OY, x1: int(ox) + entry.OX + entry.W, y1: int(oy) + entry.OY + entry.H,
				ax: entry.X, ay: entry.Y,
			})
		}
	}
	return tiles, nil
}

// rasterizeTile makes the coverage of the area of the shape, stores it in the
// atlas and uploads it.
func (r *resources) rasterizeTile(op ui.DisplayOp, m *ui.Matrix4, key maskKey, area image.Rectangle) (*gpu.AtlasEntry[maskKey], error) {
	var mask *soft.Mask
	if !area.Empty() {
		mask, _ = soft.OpMask(op, m, area)
	}
	r.tilesRasterized++
	if mask == nil {
		return r.atlas.Put(key, 0, 0)
	}
	w, h := mask.Rect.Dx(), mask.Rect.Dy()
	entry, err := r.atlas.Put(key, w, h)
	if err != nil {
		return nil, err
	}
	for len(r.pages) < r.atlas.Pages() {
		page, err := r.dev.AcquireTexture(atlasPageSize, atlasPageSize, TextureFormatR8)
		if err != nil {
			return nil, err
		}
		r.dev.SetTextureParams(page, glNearest, glNearest, glClampToEdge, glClampToEdge)
		r.pages = append(r.pages, page)
	}
	bytes := make([]byte, len(mask.Cov))
	for i, c := range mask.Cov {
		bytes[i] = byte(math.Min(math.Max(float64(c)*255+0.5, 0), 255))
	}
	if err := r.dev.UploadTexture(r.pages[entry.Page], entry.X, entry.Y, w, h, bytes); err != nil {
		return nil, err
	}
	entry.OX, entry.OY = mask.Rect.Min.X, mask.Rect.Min.Y
	return entry, nil
}

// floorDiv divides rounding toward negative infinity.
func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}
