package gles

import (
	"image"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// Images (SPEC §9.1): each image is uploaded once as a texture with mipmaps and
// kept while it is drawn. An image rectangle is one instance whose fragment shader
// maps the pixel back to the image's texels and samples them with the paint's
// filter quality: nearest, bilinear, bilinear on one mip level for a downscale, or
// bicubic. The coverage of the rectangle's edge comes from the geometry.

// imageEntry is an image's texture.
type imageEntry struct {
	tex      *Texture
	w, h     int
	levels   int    // the number of mip levels, the image itself included
	pix      []byte // kept so that the address in the key is not reused
	bytes    int
	lastUsed uint64
}

// imageTexture returns the texture of the image, uploading it the first time.
func (r *resources) imageTexture(img *ui.Image) (*imageEntry, error) {
	pix := img.Pixels()
	w, h := img.Width(), img.Height()
	key := gpu.KeyOfImage(img)
	if e, ok := r.images[key]; ok {
		e.lastUsed = r.frame
		return e, nil
	}
	r.imagesUploaded++
	tex, err := r.dev.AcquireTexture(w, h, TextureFormatRGBA8)
	if err != nil {
		return nil, err
	}
	r.dev.SetTextureParams(tex, glLinearMipmapNearest, glLinear, glClampToEdge, glClampToEdge)
	if err := r.dev.UploadTexture(tex, 0, 0, w, h, pix); err != nil {
		r.dev.ReleaseTexture(tex)
		return nil, err
	}
	levels := gpu.MipLevels(w, h)
	if levels > 1 {
		r.dev.GenerateMipmaps(tex)
	}
	e := &imageEntry{tex: tex, w: w, h: h, levels: levels, pix: pix, bytes: len(pix) * 4 / 3, lastUsed: r.frame}
	r.images[key] = e
	r.imageBytes += e.bytes
	return e, nil
}

func (r *resources) dropImage(key gpu.ImageKey, e *imageEntry) {
	r.dev.ReleaseTexture(e.tex)
	r.imageBytes -= e.bytes
	delete(r.images, key)
}

// dropOldestImage drops the least recently used image that this frame does not
// use; false when there is none.
func (r *resources) dropOldestImage() bool {
	var oldestKey gpu.ImageKey
	var oldest *imageEntry
	for k, e := range r.images {
		if e.lastUsed != r.frame && (oldest == nil || e.lastUsed < oldest.lastUsed) {
			oldestKey, oldest = k, e
		}
	}
	if oldest == nil {
		return false
	}
	r.dropImage(oldestKey, oldest)
	return true
}

// imagePaint reads what the image pipeline takes from a paint: its alpha, color
// filters and blend mode. The paint's shader and mask filter do not apply to
// images, as in Skia.
func (s *shapeContext) imagePaint(p *ui.Paint) (info paintInfo, status paintStatus, err error) {
	return s.paintWith(p, false, s.t)
}

// checkImage fails with GLImageDisposed for an image that cannot be drawn.
func checkImage(img *ui.Image) error {
	if img == nil || img.IsDisposed() {
		return errors.New("GLImageDisposed", "the image was disposed, or is nil, and cannot be drawn")
	}
	return nil
}

// encoder returns the image encoder of the command for an image whose texture has the
// entry, which adds each instance it makes to the frame under the key of info.
func (s *shapeContext) imageEncoder(info paintInfo, entry *imageEntry, p *ui.Paint) *gpu.ImageEncoder {
	return &gpu.ImageEncoder{
		T: s.t, Outer: s.outer, Clip: s.clip,
		Paint:   gpu.ShapePaint{Color: info.color, Antialias: info.aa},
		Quality: p.FilterQuality(), Levels: entry.levels,
		Emit: func(in *gpu.ImageInstance, covers image.Rectangle) {
			if key, ok := s.key(pipeImage, info, entry.tex, covers); ok {
				s.b.frame.add(key, in[:], 1)
			}
		},
	}
}

// imageRect draws the part src of the image into dst, as Canvas.drawImageRect.
func (s *shapeContext) imageRect(img *ui.Image, src, dst ui.Rect, p *ui.Paint) (bool, error) {
	if err := checkImage(img); err != nil {
		return false, err
	}
	info, status, err := s.imagePaint(p)
	if status != paintDraw {
		return status == paintNothing, err
	}
	entry, err := s.b.res.imageTexture(img)
	if err != nil {
		return false, err
	}
	s.imageEncoder(info, entry, p).Rect(img.Width(), img.Height(), src, dst)
	return true, nil
}

// imageNine draws an image as a nine-patch: the corners keep their size, the edges
// stretch along one axis and the center along both.
func (s *shapeContext) imageNine(op ui.OpDrawImageNine) (bool, error) {
	if err := checkImage(op.Image); err != nil {
		return false, err
	}
	info, status, err := s.imagePaint(&op.Paint)
	if status != paintDraw {
		return status == paintNothing, err
	}
	entry, err := s.b.res.imageTexture(op.Image)
	if err != nil {
		return false, err
	}
	s.imageEncoder(info, entry, &op.Paint).Nine(op.Image.Width(), op.Image.Height(), op.Center, op.Dst)
	return true, nil
}

// imageAtlas draws the parts of an atlas image, each moved by its transform. With
// colors, each sprite is the source and its color the destination of the blend
// mode, so that srcIn with a transparent white scales the sprite's alpha.
func (s *shapeContext) imageAtlas(op ui.OpDrawAtlas) (bool, error) {
	if err := checkImage(op.Atlas); err != nil {
		return false, err
	}
	mode := 0
	if len(op.Colors) > 0 {
		var ok bool
		if mode, ok = blendIndex(op.BlendMode); !ok {
			return false, s.b.unsupported("atlas blend mode " + op.BlendMode.String())
		}
	}
	info, status, err := s.imagePaint(&op.Paint)
	if status != paintDraw {
		return status == paintNothing, err
	}
	if len(op.Rects) == 0 {
		return true, nil
	}
	entry, err := s.b.res.imageTexture(op.Atlas)
	if err != nil {
		return false, err
	}
	s.imageEncoder(info, entry, &op.Paint).Atlas(op.Atlas.Width(), op.Atlas.Height(), op, mode)
	return true, nil
}
