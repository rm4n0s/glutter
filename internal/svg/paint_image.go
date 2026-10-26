package svg

import (
	"context"

	"github.com/rm4n0s/glutter/ui"
)

// drawImage draws an image element: a raster image through ui's decoder, or an SVG document through this painter. The
// image is fitted in its rectangle with preserveAspectRatio and clipped to it. An image that cannot be decoded is
// not drawn, as in browsers.
//
// SVG 1.1: struct.html#ImageElement
func (p *painter) drawImage(n *Node, alpha float64) {
	im := n.Image
	if im == nil || !n.Style.Visible {
		return
	}
	var (
		raster *ui.Image
		inner  *Document
		iw, ih float64
	)
	if im.MIME != "image/svg+xml" {
		raster = decodeRaster(im.Data)
	}
	if raster != nil {
		iw, ih = float64(raster.Width()), float64(raster.Height())
	} else if p.depth < maxImageDepth {
		d, err := Parse(im.Data)
		if err != nil {
			return
		}
		inner = d
		size := d.Size()
		iw, ih = size.Width, size.Height
	}
	if !(iw > 0 && ih > 0) {
		return
	}
	w, h := im.Width, im.Height
	switch {
	case w == 0 && h == 0:
		w, h = iw, ih
	case w == 0:
		w = h * iw / ih
	case h == 0:
		h = w * ih / iw
	}
	p.save()
	defer p.restore()
	p.c.ClipRect(ui.NewRectFromLTWH(im.X, im.Y, w, h))
	p.concat(Translate(im.X, im.Y).Mul(ViewBoxTransform(ViewBox{Width: iw, Height: ih}, im.PreserveAspectRatio, w, h)))
	if raster != nil {
		paint := ui.NewPaint()
		paint.SetColor(ui.NewColorFromARGB(alpha8(alpha), 0, 0, 0))
		paint.SetFilterQuality(ui.FilterQualityMedium)
		p.c.DrawImage(raster, ui.Offset{}, paint)
		return
	}
	if alpha < 1 {
		p.c.SaveLayer(nil, alphaPaint(alpha))
	}
	p.depth++
	p.drawDocument(inner)
	p.depth--
	if alpha < 1 {
		p.c.Restore()
	}
}

// decodeRaster decodes PNG, JPEG, GIF (the first frame), WebP or BMP data, or returns nil.
func decodeRaster(data []byte) *ui.Image {
	// Decoding is CPU work on bytes that are already in memory: nothing to cancel.
	ctx := context.Background()
	codec, err := ui.InstantiateImageCodec(ctx, data)
	if err != nil {
		return nil
	}
	defer codec.Dispose()
	frame, err := codec.GetNextFrame(ctx)
	if err != nil {
		return nil
	}
	return frame.Image
}
