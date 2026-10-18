package soft

import (
	"image"
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// minSigma is the standard deviation below which a blur does nothing.
const minSigma = 0.01

// gaussianKernel returns the weights of a Gaussian of standard deviation sigma,
// from -r to r where r is 3 sigma rounded up, summing to 1; nil when sigma is too
// small to blur.
func gaussianKernel(sigma float64) []float32 {
	if !(sigma > minSigma) || math.IsInf(sigma, 0) {
		return nil
	}
	r := int(math.Ceil(3 * sigma))
	k := make([]float32, 2*r+1)
	var sum float64
	w := make([]float64, len(k))
	for i := range k {
		d := float64(i - r)
		w[i] = math.Exp(-d * d / (2 * sigma * sigma))
		sum += w[i]
	}
	for i := range k {
		k[i] = float32(w[i] / sum)
	}
	return k
}

// blurData blurs a w x h plane of ch interleaved channels, first along x with
// sigma sx and then along y with sigma sy. What lies outside the plane is given
// by the tile mode: decal is transparent, the others repeat the plane as they do
// for shaders.
func blurData(data []float32, w, h, ch int, sx, sy float64, tile ui.TileMode) []float32 {
	out := data
	if k := gaussianKernel(sx); k != nil {
		out = blurPass(out, w, h, ch, k, tile, true)
	}
	if k := gaussianKernel(sy); k != nil {
		out = blurPass(out, w, h, ch, k, tile, false)
	}
	return out
}

// blurPass convolves the plane with the kernel along one axis.
func blurPass(src []float32, w, h, ch int, k []float32, tile ui.TileMode, horizontal bool) []float32 {
	dst := make([]float32, len(src))
	r := len(k) / 2
	n, lines := w, h
	if !horizontal {
		n, lines = h, w
	}
	// The source index of each tap for each position along the axis, or -1.
	taps := make([]int, n*len(k))
	for i := 0; i < n; i++ {
		for j := range k {
			if idx, ok := wrapIndex(i+j-r, n, tile); ok {
				taps[i*len(k)+j] = idx
			} else {
				taps[i*len(k)+j] = -1
			}
		}
	}
	for line := 0; line < lines; line++ {
		for i := 0; i < n; i++ {
			var acc [4]float32
			for j, wt := range k {
				idx := taps[i*len(k)+j]
				if idx < 0 {
					continue
				}
				var o int
				if horizontal {
					o = (line*w + idx) * ch
				} else {
					o = (idx*w + line) * ch
				}
				for c := 0; c < ch; c++ {
					acc[c] += wt * src[o+c]
				}
			}
			var o int
			if horizontal {
				o = (line*w + i) * ch
			} else {
				o = (i*w + line) * ch
			}
			for c := 0; c < ch; c++ {
				dst[o+c] = acc[c]
			}
		}
	}
	return dst
}

// blurredMask blurs the coverage of a shape and returns it over the rectangle
// out. The shape is what src covers, nothing outside src's rectangle; the blur
// reaches 3 sigma, so everything that can touch out is read.
func blurredMask(src *Mask, sx, sy float64, out image.Rectangle) *Mask {
	if src == nil || out.Empty() {
		return nil
	}
	mx, my := int(math.Ceil(3*sx)), int(math.Ceil(3*sy))
	region := image.Rect(out.Min.X-mx, out.Min.Y-my, out.Max.X+mx, out.Max.Y+my)
	w, h := region.Dx(), region.Dy()
	data := make([]float32, w*h)
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			data[(y-region.Min.Y)*w+(x-region.Min.X)] = src.At(x, y)
		}
	}
	data = blurData(data, w, h, 1, sx, sy, ui.TileModeDecal)
	m := &Mask{Rect: out, Cov: make([]float32, out.Dx()*out.Dy())}
	any := false
	for y := out.Min.Y; y < out.Max.Y; y++ {
		for x := out.Min.X; x < out.Max.X; x++ {
			v := data[(y-region.Min.Y)*w+(x-region.Min.X)]
			if v < 1.0/1024 {
				continue
			}
			m.Cov[(y-out.Min.Y)*out.Dx()+(x-out.Min.X)] = min(v, 1)
			any = true
		}
	}
	if !any {
		return nil
	}
	return m
}

// padRect grows the rectangle by 3 sigma on each side.
func padRect(r image.Rectangle, sx, sy float64) image.Rectangle {
	mx, my := int(math.Ceil(3*sx)), int(math.Ceil(3*sy))
	return image.Rect(r.Min.X-mx, r.Min.Y-my, r.Max.X+mx, r.Max.Y+my)
}

// blurImage blurs a premultiplied image with the sigmas, and returns a new image
// of the same rectangle. The tile mode says what is beyond the edges.
func blurImage(img *image.RGBA, sx, sy float64, tile ui.TileMode) *image.RGBA {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	out := image.NewRGBA(img.Rect)
	if w == 0 || h == 0 {
		return out
	}
	data := make([]float32, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := load(img, img.Rect.Min.X+x, img.Rect.Min.Y+y)
			copy(data[(y*w+x)*4:], c[:])
		}
	}
	data = blurData(data, w, h, 4, sx, sy, tile)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			o := (y*w + x) * 4
			store(out, img.Rect.Min.X+x, img.Rect.Min.Y+y,
				[4]float32{data[o] * 255, data[o+1] * 255, data[o+2] * 255, data[o+3] * 255}, 0)
		}
	}
	return out
}
