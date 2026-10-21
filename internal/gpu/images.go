package gpu

import (
	"image"
	"math"
	"unsafe"

	"github.com/rm4n0s/glutter/ui"
)

// Images (SPEC §9.1): each image is uploaded once as a texture with mipmaps and kept while it
// is drawn. An image rectangle is one instance whose fragment shader maps the pixel back to
// the image's texels and samples them with the paint's filter quality: nearest, bilinear,
// bilinear on one mip level for a downscale, or bicubic. The coverage of the rectangle's
// edge comes from the geometry. The ImageEncoder makes the instances, for both renderers.

// ImageKey identifies an image's pixels: clones share them.
//
// Flutter: no counterpart
type ImageKey struct {
	Pix  *byte
	W, H int
}

// KeyOfImage returns the key of an image's pixels.
//
// Flutter: no counterpart
func KeyOfImage(img *ui.Image) ImageKey {
	return ImageKey{Pix: unsafe.SliceData(img.Pixels()), W: img.Width(), H: img.Height()}
}

// MipLevels is the number of levels of an image of w x h pixels down to one pixel, the image
// itself included.
//
// Flutter: no counterpart
func MipLevels(w, h int) int {
	levels := 1
	for n := max(w, h); n > 1; n /= 2 {
		levels++
	}
	return levels
}

// The sizes of the caches of clip masks and images, and how long an entry may go unused.
const (
	// ClipMaskBudget is the most memory, in bytes, that clip masks that are not used in the
	// current frame may hold.
	//
	// Flutter: no counterpart
	ClipMaskBudget = 64 << 20
	// ImageBudget is the same for image textures (RGBA, with mipmaps).
	//
	// Flutter: no counterpart
	ImageBudget = 256 << 20
	// IdleFrames is how long a cached mask or image may go unused before it is dropped,
	// whatever the budget.
	//
	// Flutter: no counterpart
	IdleFrames = 240
)

// The image instance: eight vec4 attributes, see the image shaders.
const (
	// ImageFloats is the size of an image instance in floats.
	//
	// Flutter: no counterpart
	ImageFloats = 32
	// ImageFlagBox says that the instance's box is in pixels: exact coverage of an
	// axis-aligned rectangle.
	//
	// Flutter: no counterpart
	ImageFlagBox = 1
	// ImageFlagAntialias says that the edge is smoothed.
	//
	// Flutter: no counterpart
	ImageFlagAntialias = 2
)

// ImageInstance is the attributes of one image rectangle: the quad in pixels (0-3), the
// inverse transform to texels (4-9), the flags (10), the paint's alpha as a multiplier
// (12-15), the clip (16-19), the box (20-23), the level of detail, the quality and the blend
// index of the tint (24-26, -1 for none), and the tint (28-31).
//
// Flutter: no counterpart
type ImageInstance = [ImageFloats]float32

// Tint is a color that a sprite is blended with: the blend index and the premultiplied color,
// as the destination.
//
// Flutter: no counterpart
type Tint struct {
	Mode  int
	Color [4]float32
}

// ImageEncoder turns the image draws of one drawing command into instances. The renderer fills
// in the command's state and Emit, which receives each instance with the pixels it covers.
//
// Flutter: no counterpart
type ImageEncoder struct {
	// T is the transform of the command, and Outer the pixels its clip reaches.
	T     Affine2D
	Outer image.Rectangle
	// Clip is the clip rectangle with fractional edges.
	Clip [4]float32
	// Paint gives the alpha (as the color's four components) and the antialiasing.
	Paint ShapePaint
	// Quality is the paint's filter quality.
	Quality ui.FilterQuality
	// Levels is the number of mip levels of the image.
	Levels int
	// Emit is called with each finished instance and the pixels its quad covers.
	Emit func(in *ImageInstance, covers image.Rectangle)
}

// Rect draws the part src of an image of w x h pixels into dst, as Canvas.drawImageRect.
//
// Flutter: no counterpart
func (e *ImageEncoder) Rect(w, h int, src, dst ui.Rect) {
	bounds := ui.NewRectFromLTWH(0, 0, float64(w), float64(h))
	if src.IsEmpty() || dst.IsEmpty() {
		return
	}
	if clipped := src.Intersect(bounds); clipped != src {
		// Only the part of src inside the image is drawn, in the matching part of dst.
		if clipped.IsEmpty() {
			return
		}
		fx, fy := dst.Width()/src.Width(), dst.Height()/src.Height()
		dst = ui.NewRectFromLTRB(dst.Left+(clipped.Left-src.Left)*fx, dst.Top+(clipped.Top-src.Top)*fy,
			dst.Left+(clipped.Right-src.Left)*fx, dst.Top+(clipped.Bottom-src.Top)*fy)
		src = clipped
	}
	toLocal := Translation(-src.Left, -src.Top).
		Then(Scaling(dst.Width()/src.Width(), dst.Height()/src.Height())).
		Then(Translation(dst.Left, dst.Top))
	e.Blit(src, toLocal.Then(e.T), nil)
}

// Blit draws the part src of the image, whose texels the transform maps to the surface.
//
// Flutter: no counterpart
func (e *ImageEncoder) Blit(src ui.Rect, toDevice Affine2D, t *Tint) {
	inv, ok := toDevice.Invert()
	if !ok {
		return
	}
	// The pixels the rectangle can touch.
	corners := [4]ui.Offset{
		toDevice.Apply(src.Left, src.Top), toDevice.Apply(src.Right, src.Top),
		toDevice.Apply(src.Right, src.Bottom), toDevice.Apply(src.Left, src.Bottom),
	}
	dev := ui.Rect{Left: math.Inf(1), Top: math.Inf(1), Right: math.Inf(-1), Bottom: math.Inf(-1)}
	for _, c := range corners {
		dev.Left, dev.Top = math.Min(dev.Left, c.Dx), math.Min(dev.Top, c.Dy)
		dev.Right, dev.Bottom = math.Max(dev.Right, c.Dx), math.Max(dev.Bottom, c.Dy)
	}
	if !dev.IsFinite() {
		return
	}
	o := e.Outer
	x0, y0 := math.Max(math.Floor(dev.Left)-1, float64(o.Min.X)), math.Max(math.Floor(dev.Top)-1, float64(o.Min.Y))
	x1, y1 := math.Min(math.Ceil(dev.Right)+1, float64(o.Max.X)), math.Min(math.Ceil(dev.Bottom)+1, float64(o.Max.Y))
	if x1 <= x0 || y1 <= y0 {
		return
	}
	level := 0
	if e.Quality == ui.FilterQualityMedium || e.Quality == ui.FilterQualityHigh {
		if scale := math.Sqrt(math.Abs(toDevice.Det())); scale > 0 && scale < 1 {
			level = min(int(-math.Log2(scale)+0.5), e.Levels-1)
		}
	}
	q := float32(1)
	switch e.Quality {
	case ui.FilterQualityNone:
		q = 0
	case ui.FilterQualityMedium:
		q = 2
	case ui.FilterQualityHigh:
		q = 3
	}
	flags := 0
	if e.Paint.Antialias {
		flags |= ImageFlagAntialias
	}
	box := [4]float32{float32(src.Left), float32(src.Top), float32(src.Right), float32(src.Bottom)}
	if toDevice.AxisAligned() {
		flags |= ImageFlagBox
		box = [4]float32{float32(dev.Left), float32(dev.Top), float32(dev.Right), float32(dev.Bottom)}
	}
	var in ImageInstance
	in[0], in[1], in[2], in[3] = float32(x0), float32(y0), float32(x1), float32(y1)
	in[4], in[5], in[6], in[7] = float32(inv.A), float32(inv.B), float32(inv.C), float32(inv.D)
	in[8], in[9], in[10] = float32(inv.E), float32(inv.F), float32(flags)
	copy(in[12:16], e.Paint.Color[:])
	copy(in[16:20], e.Clip[:])
	copy(in[20:24], box[:])
	in[24], in[25], in[26] = float32(level), q, -1
	if t != nil {
		in[26] = float32(t.Mode)
		copy(in[28:32], t.Color[:])
	}
	e.Emit(&in, image.Rect(int(x0), int(y0), int(x1), int(y1)))
}

// Nine draws an image as a nine-patch: the corners keep their size, the edges stretch along
// one axis and the center along both.
//
// Flutter: no counterpart
func (e *ImageEncoder) Nine(w, h int, center, dst ui.Rect) {
	c := center.Intersect(ui.NewRectFromLTWH(0, 0, float64(w), float64(h)))
	fw, fh := float64(w), float64(h)
	xs, ys := [4]float64{0, c.Left, c.Right, fw}, [4]float64{0, c.Top, c.Bottom, fh}
	dxs, dys := NinePieces(dst.Left, dst.Width(), xs), NinePieces(dst.Top, dst.Height(), ys)
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			src := ui.NewRectFromLTRB(xs[i], ys[j], xs[i+1], ys[j+1])
			d := ui.NewRectFromLTRB(dxs[i], dys[j], dxs[i+1], dys[j+1])
			if src.IsEmpty() || d.IsEmpty() {
				continue
			}
			e.Rect(w, h, src, d)
		}
	}
}

// NinePieces returns the four edges of the three pieces along one axis of the destination,
// starting at origin and size long. The outer pieces keep their size in the image, and the
// middle one takes the rest; when they do not fit the outer pieces shrink in proportion and
// the middle one vanishes.
//
// Flutter: no counterpart
func NinePieces(origin, size float64, src [4]float64) [4]float64 {
	first, last := src[1]-src[0], src[3]-src[2]
	if fixed := first + last; fixed > size && fixed > 0 {
		k := size / fixed
		first, last = first*k, last*k
	}
	return [4]float64{origin, origin + first, origin + size - last, origin + size}
}

// Atlas draws the parts of an atlas image of w x h pixels, each moved by its transform. With
// colors, each sprite is the source and its color the destination of the blend mode with the
// index mode, so that srcIn with a transparent white scales the sprite's alpha.
//
// Flutter: no counterpart
func (e *ImageEncoder) Atlas(w, h int, op ui.OpDrawAtlas, mode int) {
	bounds := ui.NewRectFromLTWH(0, 0, float64(w), float64(h))
	for i, src := range op.Rects {
		src = src.Intersect(bounds)
		if src.IsEmpty() {
			continue
		}
		t := op.Transforms[i]
		rs := Affine2D{A: float64(t.Scos), B: float64(t.Ssin), C: -float64(t.Ssin), D: float64(t.Scos), E: float64(t.Tx), F: float64(t.Ty)}
		toDevice := Translation(-op.Rects[i].Left, -op.Rects[i].Top).Then(rs).Then(e.T)
		var tn *Tint
		if len(op.Colors) > 0 {
			c := op.Colors[i]
			tn = &Tint{Mode: mode, Color: [4]float32{c.R * c.A, c.G * c.A, c.B * c.A, c.A}}
		}
		e.Blit(src, toDevice, tn)
	}
}
