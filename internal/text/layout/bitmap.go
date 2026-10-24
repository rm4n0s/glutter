package layout

import (
	"bytes"
	"context"
	"image"
	"image/draw"
	_ "image/jpeg" // registers the JPEG decoder (sbix strikes may be JPEG)
	"image/png"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype/tables"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// maxBitmapGlyphSize is the largest width or height, in pixels, of a bitmap glyph that is
// decoded: a bound on the memory that a broken font can make the decoder allocate. Emoji
// strikes are about 136 pixels.
const maxBitmapGlyphSize = 2048

// chooseBitmapStrike returns the index of the strike that a size draws with, or -1 when there is
// none. It is the choice of Skia: the strike with exactly that many pixels per em, else the
// smallest strike that is bigger than the size, else (when all are smaller) the biggest.
//
// Flutter: engine/src/flutter/third_party/skia/src/ports/SkFontHost_FreeType.cpp chooseBitmapStrike
func chooseBitmapStrike(strikes []font.BitmapSize, size float64) int {
	chosen, chosenPpem := -1, 0.0
	for i, s := range strikes {
		ppem := float64(s.YPpem)
		switch {
		case ppem == size:
			return i
		case chosenPpem < size:
			// Attempt to increase the pixels per em.
			if chosenPpem < ppem {
				chosen, chosenPpem = i, ppem
			}
		default:
			// Attempt to decrease them, but not below the size.
			if size < ppem && ppem < chosenPpem {
				chosen, chosenPpem = i, ppem
			}
		}
	}
	return chosen
}

// bitmapGlyph returns the picture of a glyph that the face has as a bitmap (CBDT, the color
// emoji of Android 8 to 12, or sbix), in font units with y growing up like colrGlyph's, or nil
// if the glyph is not a color bitmap or cannot be drawn. The picture is the same for the sizes
// that pick the same strike. A bitmap that cannot be decoded is reported through OnFontError,
// once, and draws nothing.
//
// Flutter: no counterpart (Skia draws bitmap glyphs through FreeType)
func (e *Engine) bitmapGlyph(face *font.Face, gid font.GID, size float64) *ui.Picture {
	strikes := e.shapes.info(face).strikes
	i := chooseBitmapStrike(strikes, size)
	if i < 0 {
		return nil
	}
	key := colorKey{face: face, gid: gid, ppem: strikes[i].YPpem}
	if pic, ok := e.pictures.get(key); ok {
		return pic
	}
	pic, err := e.recordBitmap(face, gid, strikes[i])
	if err != nil {
		e.fontError(err)
	}
	e.pictures.put(key, pic)
	return pic
}

// recordBitmap records the drawing of the bitmap of a glyph in a strike: the
// bitmap goes where the bearings of the strike put it, scaled from the pixels of the strike to
// font units (upem/ppem of the strike per pixel), and is drawn with a filter. The picture is
// nil, without an error, when the glyph has no bitmap in a format that is drawn (PNG and JPEG).
func (e *Engine) recordBitmap(face *font.Face, gid font.GID, strike font.BitmapSize) (*ui.Picture, error) {
	bm, ext, ok := bitmapOf(face, gid, strike)
	if !ok || (bm.Format != font.PNG && bm.Format != font.JPG) || bm.Width <= 0 || bm.Height <= 0 || ext.Width <= 0 || ext.Height >= 0 {
		return nil, nil
	}
	img, err := decodeGlyphImage(e.ctx, bm)
	if err != nil {
		return nil, err
	}
	// A pixel of the strike is upem/ppem font units wide (the extents are in font units, and
	// the metrics of the strike are in its pixels).
	sx, sy := float64(ext.Width)/float64(bm.Width), -float64(ext.Height)/float64(bm.Height)
	left, top := float64(ext.XBearing), -float64(ext.YBearing) // y grows down from the baseline
	dst := ui.Rect{Left: left, Top: top, Right: left + sx*float64(img.Width()), Bottom: top + sy*float64(img.Height())}

	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.Rect{Left: dst.Left, Top: -dst.Bottom, Right: dst.Right, Bottom: -dst.Top})})
	c.Scale(1, -1) // the picture has y up; the bitmap is drawn with y down
	paint := ui.NewPaint()
	paint.SetFilterQuality(ui.FilterQualityMedium)
	c.DrawImageRect(img, ui.Rect{Right: float64(img.Width()), Bottom: float64(img.Height())}, dst, paint)
	return rec.EndRecording(), nil
}

// bitmapOf finds the bitmap of a glyph in a strike, and its extents in font units (the bearings
// and size of its metrics). The face keeps its own pixels per em.
func bitmapOf(face *font.Face, gid font.GID, strike font.BitmapSize) (font.GlyphBitmap, font.GlyphExtents, bool) {
	px, py := face.Ppem()
	face.SetPpem(strike.XPpem, strike.YPpem) // the exact pixels per em of the strike pick it
	defer face.SetPpem(px, py)
	bm, ok := face.GlyphDataBitmap(tables.GlyphID(gid))
	if !ok {
		return font.GlyphBitmap{}, font.GlyphExtents{}, false
	}
	ext, ok := face.GlyphExtents(gid)
	return bm, ext, ok
}

// decodeGlyphImage decodes the PNG or JPEG data of a bitmap glyph to an image with premultiplied
// pixels. It fails with BitmapGlyphInvalid when the data is not an image or is too big.
func decodeGlyphImage(ctx context.Context, bm font.GlyphBitmap) (*ui.Image, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(bm.Data))
	if err != nil {
		return nil, errors.NewErr(tagBitmapGlyphInvalid, err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxBitmapGlyphSize || cfg.Height > maxBitmapGlyphSize {
		return nil, errors.New(tagBitmapGlyphInvalid, "the size of a bitmap glyph is not valid", "width", cfg.Width, "height", cfg.Height)
	}
	var src image.Image
	if bm.Format == font.PNG {
		src, err = png.Decode(bytes.NewReader(bm.Data))
	} else {
		src, _, err = image.Decode(bytes.NewReader(bm.Data))
	}
	if err != nil {
		return nil, errors.NewErr(tagBitmapGlyphInvalid, err)
	}
	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src) // an RGBA image is premultiplied
	return ui.DecodeImageFromPixels(ctx, rgba.Pix, rgba.Rect.Dx(), rgba.Rect.Dy(), ui.PixelFormatRGBA8888)
}

// tagBitmapGlyphInvalid is the tag of the error of a bitmap glyph whose image cannot be decoded
// or is too big. It is reported through EngineOpts.OnFontError.
const tagBitmapGlyphInvalid = "BitmapGlyphInvalid"
