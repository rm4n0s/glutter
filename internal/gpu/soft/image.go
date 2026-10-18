package soft

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// mipLevel is one size of an image's mipmap chain, in premultiplied RGBA.
type mipLevel struct {
	w, h int
	pix  []byte
}

// levels returns the mipmap chain of the image, the image itself first. Each
// level halves the one before, rounding up, and averages 2x2 pixels.
func (r *renderer) levels(img *ui.Image) []*mipLevel {
	if l, ok := r.mips[img]; ok {
		return l
	}
	chain := []*mipLevel{{w: img.Width(), h: img.Height(), pix: img.Pixels()}}
	for {
		prev := chain[len(chain)-1]
		if prev.w == 1 && prev.h == 1 {
			break
		}
		next := &mipLevel{w: max(prev.w/2, 1), h: max(prev.h/2, 1)}
		next.pix = make([]byte, 4*next.w*next.h)
		for y := 0; y < next.h; y++ {
			for x := 0; x < next.w; x++ {
				x0, y0 := min(2*x, prev.w-1), min(2*y, prev.h-1)
				x1, y1 := min(2*x+1, prev.w-1), min(2*y+1, prev.h-1)
				for k := 0; k < 4; k++ {
					sum := int(prev.pix[4*(y0*prev.w+x0)+k]) + int(prev.pix[4*(y0*prev.w+x1)+k]) +
						int(prev.pix[4*(y1*prev.w+x0)+k]) + int(prev.pix[4*(y1*prev.w+x1)+k])
					next.pix[4*(y*next.w+x)+k] = byte((sum + 2) / 4)
				}
			}
		}
		chain = append(chain, next)
	}
	r.mips[img] = chain
	return chain
}

// texel returns the premultiplied color of a pixel of the level, clamping the
// coordinates to the edge.
func (l *mipLevel) texel(x, y int) [4]float32 {
	x, y = min(max(x, 0), l.w-1), min(max(y, 0), l.h-1)
	p := l.pix[4*(y*l.w+x):][:4]
	return [4]float32{float32(p[0]) / 255, float32(p[1]) / 255, float32(p[2]) / 255, float32(p[3]) / 255}
}

// sample reads the level at the point u, v of the image, whose size is w0 x h0.
func (l *mipLevel) sample(u, v float64, w0, h0 int, q ui.FilterQuality) [4]float32 {
	u *= float64(l.w) / float64(w0)
	v *= float64(l.h) / float64(h0)
	switch q {
	case ui.FilterQualityNone:
		return l.texel(int(math.Floor(u)), int(math.Floor(v)))
	case ui.FilterQualityHigh:
		return l.bicubic(u-0.5, v-0.5)
	}
	return l.bilinear(u-0.5, v-0.5)
}

func (l *mipLevel) bilinear(fu, fv float64) [4]float32 {
	x0, y0 := math.Floor(fu), math.Floor(fv)
	tx, ty := float32(fu-x0), float32(fv-y0)
	ix, iy := int(x0), int(y0)
	c00, c10 := l.texel(ix, iy), l.texel(ix+1, iy)
	c01, c11 := l.texel(ix, iy+1), l.texel(ix+1, iy+1)
	var out [4]float32
	for k := range out {
		top := c00[k] + (c10[k]-c00[k])*tx
		bot := c01[k] + (c11[k]-c01[k])*tx
		out[k] = top + (bot-top)*ty
	}
	return out
}

// mitchell is the Mitchell-Netravali cubic with B = C = 1/3, which Skia uses for
// the high filter quality.
func mitchell(x float64) float64 {
	const b, c = 1.0 / 3, 1.0 / 3
	x = math.Abs(x)
	switch {
	case x < 1:
		return ((12-9*b-6*c)*x*x*x + (-18+12*b+6*c)*x*x + (6 - 2*b)) / 6
	case x < 2:
		return ((-b-6*c)*x*x*x + (6*b+30*c)*x*x + (-12*b-48*c)*x + (8*b + 24*c)) / 6
	}
	return 0
}

func (l *mipLevel) bicubic(fu, fv float64) [4]float32 {
	x0, y0 := math.Floor(fu), math.Floor(fv)
	tx, ty := fu-x0, fv-y0
	var out [4]float64
	for j := -1; j <= 2; j++ {
		wy := mitchell(float64(j) - ty)
		for i := -1; i <= 2; i++ {
			w := wy * mitchell(float64(i)-tx)
			c := l.texel(int(x0)+i, int(y0)+j)
			for k := range out {
				out[k] += w * float64(c[k])
			}
		}
	}
	// The cubic can overshoot; keep a valid premultiplied color.
	a := math.Min(math.Max(out[3], 0), 1)
	var res [4]float32
	res[3] = float32(a)
	for k := 0; k < 3; k++ {
		res[k] = float32(math.Min(math.Max(out[k], 0), a))
	}
	return res
}

// checkImage returns an error for an image that cannot be drawn.
func checkImage(img *ui.Image) error {
	if img == nil || img.IsDisposed() {
		return errors.New("SoftImageDisposed", "The image was disposed, or is nil, and cannot be drawn.")
	}
	return nil
}

// drawImageRect draws the part src of the image into dst, as Canvas.DrawImageRect.
func (r *renderer) drawImageRect(img *ui.Image, src, dst ui.Rect, p *ui.Paint, st *state) error {
	if err := checkImage(img); err != nil {
		return err
	}
	if err := checkPaint(p); err != nil {
		return err
	}
	bounds := ui.NewRectFromLTWH(0, 0, float64(img.Width()), float64(img.Height()))
	if src.IsEmpty() || dst.IsEmpty() {
		return nil
	}
	if clipped := src.Intersect(bounds); clipped != src {
		// Only the part of src inside the image is drawn, in the matching part of dst.
		if clipped.IsEmpty() {
			return nil
		}
		fx, fy := dst.Width()/src.Width(), dst.Height()/src.Height()
		dst = ui.NewRectFromLTRB(dst.Left+(clipped.Left-src.Left)*fx, dst.Top+(clipped.Top-src.Top)*fy,
			dst.Left+(clipped.Right-src.Left)*fx, dst.Top+(clipped.Bottom-src.Top)*fy)
		src = clipped
	}
	toLocal := translation(-src.Left, -src.Top).
		then(scaling(dst.Width()/src.Width(), dst.Height()/src.Height())).
		then(translation(dst.Left, dst.Top))
	return r.blit(img, src, toLocal.then(affineOf(st.m)), p, st)
}

// blit draws the part src of the image, whose pixels the transform maps to the
// surface.
func (r *renderer) blit(img *ui.Image, src ui.Rect, toDevice affine, p *ui.Paint, st *state) error {
	if st.clip.empty() {
		return nil
	}
	inv, ok := toDevice.invert()
	if !ok {
		return nil
	}
	quad := []ui.Offset{
		toDevice.apply(src.Left, src.Top), toDevice.apply(src.Right, src.Top),
		toDevice.apply(src.Right, src.Bottom), toDevice.apply(src.Left, src.Bottom),
	}
	mask := Rasterize([][]ui.Offset{quad}, FillNonZero, st.clip.limit(r.img.Rect))
	if mask == nil {
		return nil
	}
	if !p.IsAntiAlias() {
		mask = threshold(mask)
	}
	quality := p.FilterQuality()
	chain := r.levels(img)
	level := 0
	if quality == ui.FilterQualityMedium || quality == ui.FilterQualityHigh {
		if scale := math.Sqrt(math.Abs(toDevice.det())); scale > 0 && scale < 1 {
			level = min(int(-math.Log2(scale)+0.5), len(chain)-1)
		}
	}
	alpha := float32(p.Color().Alpha()) / 255
	w0, h0 := img.Width(), img.Height()
	l := chain[level]
	r.composite(mask, st.clip, func(x, y int) [4]float32 {
		pt := inv.apply(float64(x)+0.5, float64(y)+0.5)
		c := l.sample(pt.Dx, pt.Dy, w0, h0, quality)
		return [4]float32{c[0] * alpha, c[1] * alpha, c[2] * alpha, c[3] * alpha}
	})
	return nil
}

// drawImageNine draws an image as a nine-patch: the corners keep their size, the
// edges stretch along one axis and the center along both.
func (r *renderer) drawImageNine(op ui.OpDrawImageNine, st *state) error {
	if err := checkImage(op.Image); err != nil {
		return err
	}
	w, h := float64(op.Image.Width()), float64(op.Image.Height())
	c := op.Center.Intersect(ui.NewRectFromLTWH(0, 0, w, h))
	xs, ys := [4]float64{0, c.Left, c.Right, w}, [4]float64{0, c.Top, c.Bottom, h}
	dxs, dys := ninePieces(op.Dst.Left, op.Dst.Width(), xs), ninePieces(op.Dst.Top, op.Dst.Height(), ys)
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			src := ui.NewRectFromLTRB(xs[i], ys[j], xs[i+1], ys[j+1])
			dst := ui.NewRectFromLTRB(dxs[i], dys[j], dxs[i+1], dys[j+1])
			if src.IsEmpty() || dst.IsEmpty() {
				continue
			}
			if err := r.drawImageRect(op.Image, src, dst, &op.Paint, st); err != nil {
				return err
			}
		}
	}
	return nil
}

// ninePieces returns the four edges of the three pieces along one axis of the
// destination, starting at origin and size long. The outer pieces keep their
// size in the image, and the middle one takes the rest; when they do not fit the
// outer pieces shrink in proportion and the middle one vanishes.
func ninePieces(origin, size float64, src [4]float64) [4]float64 {
	first, last := src[1]-src[0], src[3]-src[2]
	if fixed := first + last; fixed > size && fixed > 0 {
		k := size / fixed
		first, last = first*k, last*k
	}
	return [4]float64{origin, origin + first, origin + size - last, origin + size}
}

// drawAtlas draws the parts of the atlas, each moved by its transform.
func (r *renderer) drawAtlas(op ui.OpDrawAtlas, st *state) error {
	if err := checkImage(op.Atlas); err != nil {
		return err
	}
	if err := checkPaint(&op.Paint); err != nil {
		return err
	}
	if len(op.Colors) > 0 {
		return notImplemented("atlas colors", op.BlendMode)
	}
	bounds := ui.NewRectFromLTWH(0, 0, float64(op.Atlas.Width()), float64(op.Atlas.Height()))
	for i, src := range op.Rects {
		src = src.Intersect(bounds)
		if src.IsEmpty() {
			continue
		}
		t := op.Transforms[i]
		rs := affine{a: float64(t.Scos), b: float64(t.Ssin), c: -float64(t.Ssin), d: float64(t.Scos), e: float64(t.Tx), f: float64(t.Ty)}
		toDevice := translation(-op.Rects[i].Left, -op.Rects[i].Top).then(rs).then(affineOf(st.m))
		if err := r.blit(op.Atlas, src, toDevice, &op.Paint, st); err != nil {
			return err
		}
	}
	return nil
}
