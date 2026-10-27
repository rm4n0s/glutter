package svg

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// maxTilePixels is the largest width or height of the image of a pattern tile; a bigger tile is made smaller and
// scaled up.
const maxTilePixels = 4096

// tileKey names a tile image that was made: the pattern, its size in pixels and the box it was made for (a pattern in
// objectBoundingBox units depends on the box).
type tileKey struct {
	pattern *Pattern
	w, h    int
	box     ui.Rect
}

// patternShader makes the shader of a pattern for a shape with the bounding box: the tile is drawn once into an image
// with as many pixels as the device will show, and the shader repeats it. It reports false if nothing is to be
// painted: a box with no area in objectBoundingBox units, a tile with no size, or a tile that could not be made.
//
// SVG 1.1: pservers.html#Patterns
func (p *painter) patternShader(pt *Pattern, bbox ui.Rect, hasBBox bool, toLocal Affine) (ui.Shader, bool) {
	var x, y, w, h float64
	if pt.Units == UnitsObjectBoundingBox {
		if !hasBBox {
			return nil, false
		}
		x = bbox.Left + pt.X.Resolve(1)*bbox.Width()
		y = bbox.Top + pt.Y.Resolve(1)*bbox.Height()
		w, h = pt.Width.Resolve(1)*bbox.Width(), pt.Height.Resolve(1)*bbox.Height()
	} else {
		x, y = pt.X.Resolve(p.vw), pt.Y.Resolve(p.vh)
		w, h = pt.Width.Resolve(p.vw), pt.Height.Resolve(p.vh)
	}
	if !(w > 0 && h > 0) || math.IsInf(w, 0) || math.IsInf(h, 0) {
		return nil, false
	}
	content := Identity
	switch {
	case pt.ViewBox != nil:
		content = ViewBoxTransform(*pt.ViewBox, pt.PreserveAspectRatio, w, h)
	case pt.ContentUnits == UnitsObjectBoundingBox:
		if !hasBBox {
			return nil, false
		}
		content = Scale(bbox.Width(), bbox.Height())
	}
	// The tile in the user space, and the device's pixels it covers.
	tileToLocal := toLocal.Mul(pt.Transform).Mul(Translate(x, y))
	toDevice := Scale(p.scale, p.scale).Mul(p.ctm).Mul(tileToLocal)
	if _, ok := tileToLocal.Invert(); !ok {
		return nil, false
	}
	pw := int(math.Ceil(w*math.Hypot(toDevice.A, toDevice.B) - 1e-6))
	ph := int(math.Ceil(h*math.Hypot(toDevice.C, toDevice.D) - 1e-6))
	pw, ph = max(pw, 1), max(ph, 1)
	if pw > maxTilePixels || ph > maxTilePixels {
		f := float64(maxTilePixels) / float64(max(pw, ph))
		pw, ph = max(int(float64(pw)*f), 1), max(int(float64(ph)*f), 1)
	}
	key := tileKey{pattern: pt, w: pw, h: ph}
	if pt.Units == UnitsObjectBoundingBox || pt.ContentUnits == UnitsObjectBoundingBox {
		key.box = bbox
	}
	img, ok := p.tiles[key]
	if !ok {
		img = p.makeTile(pt, w, h, pw, ph, content)
		p.tiles[key] = img
	}
	if img == nil {
		return nil, false
	}
	m := tileToLocal.Mul(Scale(w/float64(pw), h/float64(ph)))
	return ui.NewImageShader(img, ui.TileModeRepeated, ui.TileModeRepeated, m.Matrix4(),
		ui.ImageShaderOpts{FilterQuality: ui.FilterQualityLow}), true
}

// makeTile draws the content of a pattern into an image of pw x ph pixels that stands for a tile of w x h user
// units, or returns nil if that failed.
func (p *painter) makeTile(pt *Pattern, w, h float64, pw, ph int, content Affine) *ui.Image {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	q := newPainter(c, 1)
	q.ras, q.depth, q.doc, q.text = p.ras, p.depth, p.doc, p.text
	q.vw, q.vh = p.vw, p.vh
	q.active = p.active
	if pt.ViewBox != nil {
		q.vw, q.vh = pt.ViewBox.Width, pt.ViewBox.Height
	}
	q.concat(Scale(float64(pw)/w, float64(ph)/h))
	c.ClipRect(ui.NewRectFromLTWH(0, 0, w, h))
	q.concat(content)
	for _, n := range pt.Children {
		q.drawNode(n, 1)
	}
	img, err := rec.EndRecording().ToImage(pw, ph, ui.ToImageOpts{Rasterizer: p.ras})
	if err != nil {
		return nil
	}
	return img
}
