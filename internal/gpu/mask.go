package gpu

import (
	"image"
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// General paths (SPEC §9.1): a software rasterizer turns the shape into an 8-bit coverage
// mask, which is cached in the coverage atlas by the shape, the transform's linear part and
// its fractional translation, and the paint's stroke. The shape is cut into tiles of at most
// MaxAtlasTile pixels, anchored at the origin of the transformed shape's space, so that a
// shape drawn again at another whole-pixel offset, or only partly visible, finds its tiles.
// The GPU composites each tile with the paint's color. MaskCache is that cache for both GPU
// renderers: they give it the rasterizer and the way to put pixels in a page.

// The sizes of the coverage atlas.
const (
	// AtlasPageSize is the side of a page of the coverage atlas, and AtlasMaxPages the number
	// of pages it may grow to: 8-bit pages, so 32 MB at most.
	//
	// Flutter: no counterpart
	AtlasPageSize = 2048
	// AtlasMaxPages is the number of pages the coverage atlas may grow to.
	//
	// Flutter: no counterpart
	AtlasMaxPages = 8
)

// MaskKey identifies a tile of the coverage of a shape.
//
// Flutter: no counterpart
type MaskKey struct {
	Geom   uint64     // a hash of the shape's geometry
	Style  uint64     // a hash of the paint's stroke and anti-aliasing
	Lin    [4]float32 // the linear part of the transform
	FX, FY float32    // the fractional part of its translation
	TX, TY int32      // the tile, in tiles from the origin of the shape's space
}

// MaskTile is a tile of coverage to draw: where it goes on the surface and where its texels
// are.
//
// Flutter: no counterpart
type MaskTile struct {
	Page           int // the atlas page
	X0, Y0, X1, Y1 int // the pixels of the surface it covers
	AX, AY         int // its top left texel on the page
}

// OpShape describes what an operation draws, for the atlas: a hash of its geometry, its
// bounds in its own coordinates, and its paint.
//
// Flutter: no counterpart
type OpShape struct {
	Geom   uint64
	Bounds ui.Rect
	Paint  *ui.Paint
	Stroke bool // the outline is stroked, not filled
}

// ShapeOf describes a shape-drawing operation; false for any other.
//
// Flutter: no counterpart
func ShapeOf(op ui.DisplayOp) (OpShape, bool) {
	h := NewHasher()
	rr := func(left, top, right, bottom float64, radii ...float64) {
		h.F64(left)
		h.F64(top)
		h.F64(right)
		h.F64(bottom)
		for _, r := range radii {
			h.F64(r)
		}
	}
	switch v := op.(type) {
	case ui.OpDrawPath:
		h.I(1)
		h.U64(v.Path.Fingerprint())
		return OpShape{h.Sum(), v.Path.GetBounds(), &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawRect:
		h.I(2)
		rr(v.Rect.Left, v.Rect.Top, v.Rect.Right, v.Rect.Bottom)
		return OpShape{h.Sum(), v.Rect, &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawRRect:
		h.I(3)
		r := v.RRect
		rr(r.Left, r.Top, r.Right, r.Bottom, r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY, r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY)
		return OpShape{h.Sum(), v.RRect.OuterRect(), &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawRSuperellipse:
		h.I(4)
		r := v.RSuperellipse
		rr(r.Left, r.Top, r.Right, r.Bottom, r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY, r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY)
		return OpShape{h.Sum(), v.RSuperellipse.OuterRect(), &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawDRRect:
		h.I(5)
		o, in := v.Outer, v.Inner
		rr(o.Left, o.Top, o.Right, o.Bottom, o.TlRadiusX, o.TlRadiusY, o.TrRadiusX, o.TrRadiusY, o.BrRadiusX, o.BrRadiusY, o.BlRadiusX, o.BlRadiusY)
		rr(in.Left, in.Top, in.Right, in.Bottom, in.TlRadiusX, in.TlRadiusY, in.TrRadiusX, in.TrRadiusY, in.BrRadiusX, in.BrRadiusY, in.BlRadiusX, in.BlRadiusY)
		return OpShape{h.Sum(), v.Outer.OuterRect(), &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawOval:
		h.I(6)
		rr(v.Rect.Left, v.Rect.Top, v.Rect.Right, v.Rect.Bottom)
		return OpShape{h.Sum(), v.Rect, &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawCircle:
		h.I(7)
		rr(v.Center.Dx, v.Center.Dy, v.Radius, 0)
		return OpShape{h.Sum(), ui.NewRectFromCircle(v.Center, v.Radius), &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawArc:
		h.I(8)
		rr(v.Rect.Left, v.Rect.Top, v.Rect.Right, v.Rect.Bottom, v.StartAngle, v.SweepAngle)
		h.B(v.UseCenter)
		return OpShape{h.Sum(), v.Rect, &v.Paint, v.Paint.Style() == ui.PaintingStyleStroke}, true
	case ui.OpDrawLine:
		h.I(9)
		rr(v.P1.Dx, v.P1.Dy, v.P2.Dx, v.P2.Dy)
		b := ui.Rect{Left: min(v.P1.Dx, v.P2.Dx), Top: min(v.P1.Dy, v.P2.Dy), Right: max(v.P1.Dx, v.P2.Dx), Bottom: max(v.P1.Dy, v.P2.Dy)}
		return OpShape{h.Sum(), b, &v.Paint, true}, true
	case ui.OpDrawPoints:
		h.I(10)
		h.I(int(v.Mode))
		b := ui.Rect{Left: math.Inf(1), Top: math.Inf(1), Right: math.Inf(-1), Bottom: math.Inf(-1)}
		for _, p := range v.Points {
			h.F64(p.Dx)
			h.F64(p.Dy)
			b.Left, b.Top = min(b.Left, p.Dx), min(b.Top, p.Dy)
			b.Right, b.Bottom = max(b.Right, p.Dx), max(b.Bottom, p.Dy)
		}
		return OpShape{h.Sum(), b, &v.Paint, true}, true
	}
	return OpShape{}, false
}

// StyleKey hashes the parts of a paint that change the shape's coverage.
//
// Flutter: no counterpart
func StyleKey(p *ui.Paint, stroke bool) uint64 {
	h := NewHasher()
	h.B(stroke)
	h.B(p.IsAntiAlias())
	if stroke {
		h.F64(p.StrokeWidth())
		h.I(int(p.StrokeCap()))
		h.I(int(p.StrokeJoin()))
		h.F64(p.StrokeMiterLimit())
	}
	return h.Sum()
}

// MaskMargin is how far the coverage of the shape can reach beyond its bounds, in its own
// units, under a transform that scales lengths by scale: an edge pixel, and for a stroke
// half its width times its miter.
//
// Flutter: no counterpart
func MaskMargin(sh OpShape, scale float64) float64 {
	margin := 1 / scale
	if sh.Stroke {
		hw := sh.Paint.StrokeWidth() / 2
		if sh.Paint.StrokeWidth() <= 0 {
			hw = 0.5 / scale
		}
		margin += hw * max(sh.Paint.StrokeMiterLimit(), 1.5)
	}
	return margin
}

// MaskCache is the coverage atlas of a renderer and the tiling of shapes into it.
//
// Flutter: no counterpart
type MaskCache struct {
	atlas *Atlas[MaskKey]
	pages int
	// Rasterize makes the coverage of op under m over area: the rectangle it covers and one
	// coverage from 0 to 1 per pixel, row by row; nil when the shape covers nothing there.
	Rasterize func(op ui.DisplayOp, m *ui.Matrix4, area image.Rectangle) (rect image.Rectangle, cov []float32)
	// NewPage is called when the atlas grows by a page, which the renderer must make: an 8-bit
	// texture of PageSize pixels on each side.
	NewPage func() error
	// Upload copies the bytes of a tile, w x h, row by row, to the page at (x, y).
	Upload func(page, x, y, w, h int, data []byte) error
	// Rasterized counts the tiles that were rasterized.
	Rasterized int
}

// NewMaskCache returns an empty cache whose atlas has pages of pageSize pixels, at most
// maxPages of them.
//
// Flutter: no counterpart
func NewMaskCache(pageSize, maxPages int) *MaskCache {
	return &MaskCache{atlas: NewAtlas[MaskKey](pageSize, maxPages)}
}

// SetAtlas replaces the atlas, for tests that need a small one.
//
// Flutter: no counterpart
func (c *MaskCache) SetAtlas(a *Atlas[MaskKey]) { c.atlas = a }

// Atlas returns the atlas that holds the tiles.
//
// Flutter: no counterpart
func (c *MaskCache) Atlas() *Atlas[MaskKey] { return c.atlas }

// BeginFrame starts a frame: what is used from now on survives until the next.
//
// Flutter: no counterpart
func (c *MaskCache) BeginFrame() { c.atlas.BeginFrame() }

// Clear forgets every tile.
//
// Flutter: no counterpart
func (c *MaskCache) Clear() { c.atlas.Clear(); c.pages = 0 }

// Tiles returns the tiles of coverage to draw for the shape under the transform, within the
// visible pixels, rasterizing and uploading the ones the atlas does not have. margin is how
// far the shape's coverage can reach beyond its bounds, in its own units. It fails with
// AtlasFull when a tile fits no page.
//
// Flutter: no counterpart
func (c *MaskCache) Tiles(op ui.DisplayOp, sh OpShape, m *ui.Matrix4, margin float64, visible image.Rectangle) ([]MaskTile, error) {
	st := m.Storage()
	e, f := float64(st[12]), float64(st[13])
	ox, oy := math.Floor(e), math.Floor(f)
	// The mask is made under the transform with only the fractional translation: it
	// is the same wherever on the surface a whole number of pixels takes it.
	mm := m.Clone()
	mm.SetEntry(0, 3, float32(e-ox))
	mm.SetEntry(1, 3, float32(f-oy))
	local := sh.Bounds.Inflate(margin)
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
	key := MaskKey{
		Geom: sh.Geom, Style: StyleKey(sh.Paint, sh.Stroke),
		Lin: [4]float32{st[0], st[1], st[4], st[5]},
		FX:  float32(e - ox), FY: float32(f - oy),
	}
	const t = MaxAtlasTile
	var tiles []MaskTile
	for ty := FloorDiv(view.Min.Y, t); ty <= FloorDiv(view.Max.Y-1, t); ty++ {
		for tx := FloorDiv(view.Min.X, t); tx <= FloorDiv(view.Max.X-1, t); tx++ {
			key.TX, key.TY = int32(tx), int32(ty)
			entry := c.atlas.Get(key)
			if entry == nil {
				var err error
				if entry, err = c.rasterizeTile(op, mm, key, image.Rect(tx*t, ty*t, (tx+1)*t, (ty+1)*t).Intersect(bounds)); err != nil {
					return nil, err
				}
			}
			if entry.Empty {
				continue
			}
			tiles = append(tiles, MaskTile{
				Page: entry.Page,
				X0:   int(ox) + entry.OX, Y0: int(oy) + entry.OY, X1: int(ox) + entry.OX + entry.W, Y1: int(oy) + entry.OY + entry.H,
				AX: entry.X, AY: entry.Y,
			})
		}
	}
	return tiles, nil
}

// rasterizeTile makes the coverage of the area of the shape, stores it in the atlas and
// uploads it.
func (c *MaskCache) rasterizeTile(op ui.DisplayOp, m *ui.Matrix4, key MaskKey, area image.Rectangle) (*AtlasEntry[MaskKey], error) {
	var rect image.Rectangle
	var cov []float32
	if !area.Empty() {
		rect, cov = c.Rasterize(op, m, area)
	}
	c.Rasterized++
	if cov == nil {
		return c.atlas.Put(key, 0, 0)
	}
	w, h := rect.Dx(), rect.Dy()
	entry, err := c.atlas.Put(key, w, h)
	if err != nil {
		return nil, err
	}
	for c.pages < c.atlas.Pages() {
		if err := c.NewPage(); err != nil {
			return nil, err
		}
		c.pages++
	}
	if err := c.Upload(entry.Page, entry.X, entry.Y, w, h, CoverageBytes(cov)); err != nil {
		return nil, err
	}
	entry.OX, entry.OY = rect.Min.X, rect.Min.Y
	return entry, nil
}

// CoverageBytes converts coverages from 0 to 1 to the bytes of an 8-bit texture.
//
// Flutter: no counterpart
func CoverageBytes(cov []float32) []byte {
	b := make([]byte, len(cov))
	for i, c := range cov {
		b[i] = byte(min(max(float64(c)*255+0.5, 0), 255))
	}
	return b
}

// FloorDiv divides rounding toward negative infinity.
//
// Flutter: no counterpart
func FloorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}
