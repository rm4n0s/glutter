package gles

import (
	"image"
	"math"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// Images (SPEC §9.1): each image is uploaded once as a texture with mipmaps and
// kept while it is drawn. An image rectangle is one instance whose fragment shader
// maps the pixel back to the image's texels and samples them with the paint's
// filter quality: nearest, bilinear, bilinear on one mip level for a downscale, or
// bicubic. The coverage of the rectangle's edge comes from the geometry.

// imageKey identifies an image's pixels: clones share them.
type imageKey struct {
	pix  *byte
	w, h int
}

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
	key := imageKey{pix: unsafe.SliceData(pix), w: w, h: h}
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
	levels := 1
	for n := max(w, h); n > 1; n /= 2 {
		levels++
	}
	if levels > 1 {
		r.dev.GenerateMipmaps(tex)
	}
	e := &imageEntry{tex: tex, w: w, h: h, levels: levels, pix: pix, bytes: len(pix) * 4 / 3, lastUsed: r.frame}
	r.images[key] = e
	r.imageBytes += e.bytes
	return e, nil
}

func (r *resources) dropImage(key imageKey, e *imageEntry) {
	r.dev.ReleaseTexture(e.tex)
	r.imageBytes -= e.bytes
	delete(r.images, key)
}

// dropOldestImage drops the least recently used image that this frame does not
// use; false when there is none.
func (r *resources) dropOldestImage() bool {
	var oldestKey imageKey
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

// The image instance: eight vec4 attributes, see shaders/image.vert.glsl.
const (
	imageFlagBox       = 1
	imageFlagAntialias = 2
)

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

// imageRect draws the part src of the image into dst, as Canvas.drawImageRect.
func (s *shapeContext) imageRect(img *ui.Image, src, dst ui.Rect, p *ui.Paint) (bool, error) {
	if err := checkImage(img); err != nil {
		return false, err
	}
	bounds := ui.NewRectFromLTWH(0, 0, float64(img.Width()), float64(img.Height()))
	if src.IsEmpty() || dst.IsEmpty() {
		return true, nil
	}
	if clipped := src.Intersect(bounds); clipped != src {
		// Only the part of src inside the image is drawn, in the matching part of dst.
		if clipped.IsEmpty() {
			return true, nil
		}
		fx, fy := dst.Width()/src.Width(), dst.Height()/src.Height()
		dst = ui.NewRectFromLTRB(dst.Left+(clipped.Left-src.Left)*fx, dst.Top+(clipped.Top-src.Top)*fy,
			dst.Left+(clipped.Right-src.Left)*fx, dst.Top+(clipped.Bottom-src.Top)*fy)
		src = clipped
	}
	toLocal := translation(-src.Left, -src.Top).
		then(scaling(dst.Width()/src.Width(), dst.Height()/src.Height())).
		then(translation(dst.Left, dst.Top))
	return s.blit(img, src, toLocal.then(s.t), p, nil)
}

// tint is a color that a sprite is blended with: the blend index and the
// premultiplied color, as the destination.
type tint struct {
	mode  int
	color [4]float32
}

// blit draws the part src of the image, whose texels the transform maps to the
// surface.
func (s *shapeContext) blit(img *ui.Image, src ui.Rect, toDevice affine2D, p *ui.Paint, t *tint) (bool, error) {
	info, status, err := s.imagePaint(p)
	if status != paintDraw {
		return status == paintNothing, err
	}
	inv, ok := toDevice.invert()
	if !ok {
		return true, nil
	}
	entry, err := s.b.res.imageTexture(img)
	if err != nil {
		return false, err
	}
	// The pixels the rectangle can touch.
	corners := [4]ui.Offset{
		toDevice.apply(src.Left, src.Top), toDevice.apply(src.Right, src.Top),
		toDevice.apply(src.Right, src.Bottom), toDevice.apply(src.Left, src.Bottom),
	}
	dev := ui.Rect{Left: math.Inf(1), Top: math.Inf(1), Right: math.Inf(-1), Bottom: math.Inf(-1)}
	for _, c := range corners {
		dev.Left, dev.Top = math.Min(dev.Left, c.Dx), math.Min(dev.Top, c.Dy)
		dev.Right, dev.Bottom = math.Max(dev.Right, c.Dx), math.Max(dev.Bottom, c.Dy)
	}
	if !dev.IsFinite() {
		return true, nil
	}
	o := s.outer
	x0, y0 := math.Max(math.Floor(dev.Left)-1, float64(o.Min.X)), math.Max(math.Floor(dev.Top)-1, float64(o.Min.Y))
	x1, y1 := math.Min(math.Ceil(dev.Right)+1, float64(o.Max.X)), math.Min(math.Ceil(dev.Bottom)+1, float64(o.Max.Y))
	if x1 <= x0 || y1 <= y0 {
		return true, nil
	}
	quality := p.FilterQuality()
	level := 0
	if quality == ui.FilterQualityMedium || quality == ui.FilterQualityHigh {
		if scale := math.Sqrt(math.Abs(toDevice.det())); scale > 0 && scale < 1 {
			level = min(int(-math.Log2(scale)+0.5), entry.levels-1)
		}
	}
	q := float32(1)
	switch quality {
	case ui.FilterQualityNone:
		q = 0
	case ui.FilterQualityMedium:
		q = 2
	case ui.FilterQualityHigh:
		q = 3
	}
	flags := 0
	if info.aa {
		flags |= imageFlagAntialias
	}
	box := [4]float32{float32(src.Left), float32(src.Top), float32(src.Right), float32(src.Bottom)}
	if toDevice.axisAligned() {
		flags |= imageFlagBox
		box = [4]float32{float32(dev.Left), float32(dev.Top), float32(dev.Right), float32(dev.Bottom)}
	}
	var in [32]float32
	in[0], in[1], in[2], in[3] = float32(x0), float32(y0), float32(x1), float32(y1)
	in[4], in[5], in[6], in[7] = float32(inv.a), float32(inv.b), float32(inv.c), float32(inv.d)
	in[8], in[9], in[10] = float32(inv.e), float32(inv.f), float32(flags)
	copy(in[12:16], info.color[:])
	copy(in[16:20], s.clip[:])
	copy(in[20:24], box[:])
	in[24], in[25], in[26] = float32(level), q, -1
	if t != nil {
		in[26] = float32(t.mode)
		copy(in[28:32], t.color[:])
	}
	key, ok := s.key(pipeImage, info, entry.tex, image.Rect(int(x0), int(y0), int(x1), int(y1)))
	if !ok {
		return true, nil
	}
	s.b.frame.add(key, in[:], 1)
	return true, nil
}

// imageNine draws an image as a nine-patch: the corners keep their size, the edges
// stretch along one axis and the center along both.
func (s *shapeContext) imageNine(op ui.OpDrawImageNine) (bool, error) {
	if err := checkImage(op.Image); err != nil {
		return false, err
	}
	w, h := float64(op.Image.Width()), float64(op.Image.Height())
	c := op.Center.Intersect(ui.NewRectFromLTWH(0, 0, w, h))
	xs, ys := [4]float64{0, c.Left, c.Right, w}, [4]float64{0, c.Top, c.Bottom, h}
	dxs, dys := ninePieces(op.Dst.Left, op.Dst.Width(), xs), ninePieces(op.Dst.Top, op.Dst.Height(), ys)
	drawn := true
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			src := ui.NewRectFromLTRB(xs[i], ys[j], xs[i+1], ys[j+1])
			dst := ui.NewRectFromLTRB(dxs[i], dys[j], dxs[i+1], dys[j+1])
			if src.IsEmpty() || dst.IsEmpty() {
				continue
			}
			ok, err := s.imageRect(op.Image, src, dst, &op.Paint)
			if err != nil {
				return false, err
			}
			drawn = drawn && ok
		}
	}
	return drawn, nil
}

// ninePieces returns the four edges of the three pieces along one axis of the
// destination, starting at origin and size long. The outer pieces keep their size
// in the image, and the middle one takes the rest; when they do not fit the outer
// pieces shrink in proportion and the middle one vanishes.
func ninePieces(origin, size float64, src [4]float64) [4]float64 {
	first, last := src[1]-src[0], src[3]-src[2]
	if fixed := first + last; fixed > size && fixed > 0 {
		k := size / fixed
		first, last = first*k, last*k
	}
	return [4]float64{origin, origin + first, origin + size - last, origin + size}
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
	bounds := ui.NewRectFromLTWH(0, 0, float64(op.Atlas.Width()), float64(op.Atlas.Height()))
	drawn := true
	for i, src := range op.Rects {
		src = src.Intersect(bounds)
		if src.IsEmpty() {
			continue
		}
		t := op.Transforms[i]
		rs := affine2D{a: float64(t.Scos), b: float64(t.Ssin), c: -float64(t.Ssin), d: float64(t.Scos), e: float64(t.Tx), f: float64(t.Ty)}
		toDevice := translation(-op.Rects[i].Left, -op.Rects[i].Top).then(rs).then(s.t)
		var tn *tint
		if len(op.Colors) > 0 {
			c := op.Colors[i]
			tn = &tint{mode: mode, color: [4]float32{c.R * c.A, c.G * c.A, c.B * c.A, c.A}}
		}
		ok, err := s.blit(op.Atlas, src, toDevice, &op.Paint, tn)
		if err != nil {
			return false, err
		}
		drawn = drawn && ok
	}
	return drawn, nil
}
