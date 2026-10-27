package svg

import (
	"math"
)

// pixbuf is an image that a filter works on: w x h pixels of premultiplied RGBA, 8 bits to a channel, like the images
// of ui.
type pixbuf struct {
	w, h int
	pix  []byte
	// linear says that the values are in linearRGB, not sRGB.
	linear bool
	// sub is the subregion that the primitive that made the image drew in, in pixels: the image is transparent outside
	// it.
	x0, y0, x1, y1 int
}

func newPixbuf(w, h int) *pixbuf {
	return &pixbuf{w: w, h: h, pix: make([]byte, 4*w*h), x1: w, y1: h}
}

func (b *pixbuf) clone() *pixbuf {
	c := *b
	c.pix = append([]byte(nil), b.pix...)
	return &c
}

// clipTo makes the image transparent outside the rectangle, and records it as the subregion.
func (b *pixbuf) clipTo(x0, y0, x1, y1 int) {
	x0, y0 = min(max(x0, 0), b.w), min(max(y0, 0), b.h)
	x1, y1 = min(max(x1, x0), b.w), min(max(y1, y0), b.h)
	for y := 0; y < b.h; y++ {
		row := b.pix[4*y*b.w : 4*(y+1)*b.w]
		if y < y0 || y >= y1 {
			clear(row)
			continue
		}
		clear(row[:4*x0])
		clear(row[4*x1:])
	}
	b.x0, b.y0, b.x1, b.y1 = x0, y0, x1, y1
}

// The conversions between sRGB and linearRGB on 8 bits, as lookup tables.
var srgbToLinearLUT, linearToSRGBLUT = func() (a, b [256]uint8) {
	for i := range a {
		v := float64(i) / 255
		var l float64
		if v <= 0.04045 {
			l = v / 12.92
		} else {
			l = math.Pow((v+0.055)/1.055, 2.4)
		}
		a[i] = uint8(math.Round(l * 255))
		var s float64
		if v <= 0.0031308 {
			s = 12.92 * v
		} else {
			s = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		b[i] = uint8(math.Round(math.Min(1, math.Max(0, s)) * 255))
	}
	return
}()

// unpremul returns a premultiplied channel c of a pixel with alpha a as a straight one.
func unpremul(c, a uint8) uint8 {
	if a == 255 || c == 0 {
		return c
	}
	if a == 0 {
		return 0
	}
	return uint8(min(255, (int(c)*255+int(a)/2)/int(a)))
}

// premul multiplies a straight channel by an alpha.
func premul(c, a uint8) uint8 {
	if a == 255 {
		return c
	}
	return uint8((int(c)*int(a) + 127) / 255)
}

// inSpace returns the image in the color space (a copy if it has to be converted, the image itself otherwise).
func (b *pixbuf) inSpace(linear bool) *pixbuf {
	if b.linear == linear {
		return b
	}
	lut := &linearToSRGBLUT
	if linear {
		lut = &srgbToLinearLUT
	}
	out := b.clone()
	out.linear = linear
	for i := 0; i < len(out.pix); i += 4 {
		a := out.pix[i+3]
		if a == 0 {
			continue
		}
		for k := 0; k < 3; k++ {
			out.pix[i+k] = premul(lut[unpremul(out.pix[i+k], a)], a)
		}
	}
	return out
}

// blurred is the Gaussian blur of the image with the standard deviations in pixels: three box blurs for a deviation of
// 2 or more, as the specification says, and the Gaussian itself for a smaller one. A deviation of 0 leaves that
// direction as it is. Beyond the edges the image is transparent.
//
// SVG 1.1: filters.html#feGaussianBlurElement
func (b *pixbuf) blurred(sx, sy float64) *pixbuf {
	out := b.clone()
	out.x0, out.y0, out.x1, out.y1 = 0, 0, b.w, b.h
	if sx > 0 {
		blurAxis(out, sx, true)
	}
	if sy > 0 {
		blurAxis(out, sy, false)
	}
	return out
}

// maxBlurSigma bounds the deviation in pixels: beyond a few times the size of the image the result is the same.
const maxBlurSigma = 4096

// blurAxis blurs the image along one axis, in place.
func blurAxis(b *pixbuf, sigma float64, horizontal bool) {
	if math.IsNaN(sigma) || sigma <= 0 {
		return
	}
	sigma = math.Min(sigma, maxBlurSigma)
	n, lines, stride, lineStride := b.w, b.h, 4, 4*b.w
	if !horizontal {
		n, lines, stride, lineStride = b.h, b.w, 4*b.w, 4
	}
	src := make([]float32, 4*n)
	dst := make([]float32, 4*n)
	var kernel []float32
	var boxes []box
	if sigma >= 2 {
		d := int(math.Floor(sigma*3*math.Sqrt(2*math.Pi)/4 + 0.5))
		boxes = boxesFor(d)
	} else {
		kernel = gaussKernel(sigma)
	}
	for l := 0; l < lines; l++ {
		base := l * lineStride
		empty := true
		for i := 0; i < n; i++ {
			for k := 0; k < 4; k++ {
				v := b.pix[base+i*stride+k]
				src[4*i+k] = float32(v)
				if v != 0 {
					empty = false
				}
			}
		}
		if empty {
			continue
		}
		if kernel != nil {
			convolveLine(src, dst, n, kernel)
			src, dst = dst, src
		} else {
			for _, bx := range boxes {
				boxLine(src, dst, n, bx)
				src, dst = dst, src
			}
		}
		for i := 0; i < n; i++ {
			for k := 0; k < 4; k++ {
				v := src[4*i+k]
				b.pix[base+i*stride+k] = uint8(min(255, max(0, int(v+0.5))))
			}
		}
	}
	// Premultiplied colors never exceed the alpha; rounding can break that by one.
	for i := 0; i < len(b.pix); i += 4 {
		a := b.pix[i+3]
		for k := 0; k < 3; k++ {
			if b.pix[i+k] > a {
				b.pix[i+k] = a
			}
		}
	}
}

// box is a box blur of a width, whose window reaches left pixels before the output pixel and right after it.
type box struct{ size, left, right int }

// boxesFor returns the three box blurs that approximate a Gaussian when the box size is d: for an odd d three boxes of
// d centered on the pixel; for an even d two boxes of d, one shifted to the left and one to the right, and a box of d+1
// centered.
//
// SVG 1.1: filters.html#feGaussianBlurElement
func boxesFor(d int) []box {
	if d < 1 {
		d = 1
	}
	if d%2 == 1 {
		r := (d - 1) / 2
		return []box{{d, r, r}, {d, r, r}, {d, r, r}}
	}
	r := d / 2
	return []box{{d, r, r - 1}, {d, r - 1, r}, {d + 1, r, r}}
}

// boxLine averages each pixel of a line over its window (the pixels outside the line count as transparent).
func boxLine(src, dst []float32, n int, bx box) {
	inv := 1 / float32(bx.size)
	var sum [4]float32
	// The window of pixel i is [i-left, i+right].
	for k := range sum {
		sum[k] = 0
	}
	for j := 0; j <= min(bx.right, n-1); j++ {
		for k := 0; k < 4; k++ {
			sum[k] += src[4*j+k]
		}
	}
	for i := 0; i < n; i++ {
		for k := 0; k < 4; k++ {
			dst[4*i+k] = sum[k] * inv
		}
		// Slide: add i+right+1, remove i-left.
		if add := i + bx.right + 1; add < n {
			for k := 0; k < 4; k++ {
				sum[k] += src[4*add+k]
			}
		}
		if rem := i - bx.left; rem >= 0 {
			for k := 0; k < 4; k++ {
				sum[k] -= src[4*rem+k]
			}
		}
	}
}

// gaussKernel is the Gaussian of the deviation, sampled at the pixels and normalized.
func gaussKernel(sigma float64) []float32 {
	r := int(math.Ceil(sigma * 3))
	k := make([]float32, 2*r+1)
	sum := 0.0
	for i := -r; i <= r; i++ {
		v := math.Exp(-float64(i*i) / (2 * sigma * sigma))
		k[i+r] = float32(v)
		sum += v
	}
	for i := range k {
		k[i] /= float32(sum)
	}
	return k
}

func convolveLine(src, dst []float32, n int, kernel []float32) {
	r := len(kernel) / 2
	for i := 0; i < n; i++ {
		var acc [4]float32
		for j := max(0, i-r); j <= min(n-1, i+r); j++ {
			w := kernel[j-i+r]
			for k := 0; k < 4; k++ {
				acc[k] += src[4*j+k] * w
			}
		}
		copy(dst[4*i:4*i+4], acc[:])
	}
}

// offset moves the image by whole pixels; what moves in from outside is transparent.
func (b *pixbuf) offset(dx, dy int) *pixbuf {
	out := newPixbuf(b.w, b.h)
	out.linear = b.linear
	if dx <= -b.w || dx >= b.w || dy <= -b.h || dy >= b.h {
		return out
	}
	for y := 0; y < b.h; y++ {
		sy := y - dy
		if sy < 0 || sy >= b.h {
			continue
		}
		x0, x1 := max(0, dx), min(b.w, b.w+dx)
		copy(out.pix[4*(y*b.w+x0):4*(y*b.w+x1)], b.pix[4*(sy*b.w+x0-dx):4*(sy*b.w+x1-dx)])
	}
	return out
}

// morphology erodes or dilates the image: each channel of a pixel becomes the least or the greatest of the channel over
// the window of the radii around it. The pixels outside the image are transparent.
//
// SVG 1.1: filters.html#feMorphologyElement
func (b *pixbuf) morphology(rx, ry int, dilate bool) *pixbuf {
	out := b.clone()
	out.x0, out.y0, out.x1, out.y1 = 0, 0, b.w, b.h
	if rx > 0 {
		morphAxis(out, rx, dilate, true)
	}
	if ry > 0 {
		morphAxis(out, ry, dilate, false)
	}
	return out
}

func morphAxis(b *pixbuf, r int, dilate, horizontal bool) {
	n, lines, stride, lineStride := b.w, b.h, 4, 4*b.w
	if !horizontal {
		n, lines, stride, lineStride = b.h, b.w, 4*b.w, 4
	}
	line := make([]uint8, n)
	res := make([]uint8, n)
	dq := make([]int32, n)
	for l := 0; l < lines; l++ {
		base := l * lineStride
		for k := 0; k < 4; k++ {
			for i := 0; i < n; i++ {
				line[i] = b.pix[base+i*stride+k]
			}
			slideExtreme(line, res, dq, r, dilate)
			for i := 0; i < n; i++ {
				b.pix[base+i*stride+k] = res[i]
			}
		}
	}
}

// slideExtreme sets res[i] to the greatest (dilate) or the least of line over [i-r, i+r], with 0 for what is outside
// the line, in time linear in the length of the line (a monotonic queue of indices).
func slideExtreme(line, res []uint8, dq []int32, r int, dilate bool) {
	n := len(line)
	better := func(a, b uint8) bool {
		if dilate {
			return a >= b
		}
		return a <= b
	}
	head, tail := 0, 0
	next := 0
	for i := 0; i < n; i++ {
		for next <= min(n-1, i+r) {
			for tail > head && better(line[next], line[dq[tail-1]]) {
				tail--
			}
			dq[tail] = int32(next)
			tail++
			next++
		}
		for head < tail && int(dq[head]) < i-r {
			head++
		}
		v := line[dq[head]]
		if !dilate && (i-r < 0 || i+r >= n) {
			v = 0
		}
		res[i] = v
	}
}

// colorMatrix applies a 4x5 color matrix (in the order of feColorMatrix's values, offsets last) to the straight colors.
//
// SVG 1.1: filters.html#feColorMatrixElement
func (b *pixbuf) colorMatrix(m [20]float64) *pixbuf {
	out := b.clone()
	out.x0, out.y0, out.x1, out.y1 = 0, 0, b.w, b.h
	for i := 0; i < len(out.pix); i += 4 {
		a := out.pix[i+3]
		var c [4]float64
		if a != 0 {
			c = [4]float64{float64(unpremul(out.pix[i], a)) / 255, float64(unpremul(out.pix[i+1], a)) / 255,
				float64(unpremul(out.pix[i+2], a)) / 255, float64(a) / 255}
		}
		var o [4]uint8
		for r := 0; r < 4; r++ {
			v := m[r*5]*c[0] + m[r*5+1]*c[1] + m[r*5+2]*c[2] + m[r*5+3]*c[3] + m[r*5+4]
			o[r] = uint8(math.Round(math.Min(1, math.Max(0, v)) * 255))
		}
		for k := 0; k < 3; k++ {
			out.pix[i+k] = premul(o[k], o[3])
		}
		out.pix[i+3] = o[3]
	}
	return out
}

// channelFuncs are the four transfer functions of feComponentTransfer, as lookup tables of the straight channel; nil
// is the identity.
type channelFuncs [4]*[256]uint8

// transfer applies the transfer functions to the straight colors.
//
// SVG 1.1: filters.html#feComponentTransferElement
func (b *pixbuf) transfer(f channelFuncs) *pixbuf {
	out := b.clone()
	out.x0, out.y0, out.x1, out.y1 = 0, 0, b.w, b.h
	for i := 0; i < len(out.pix); i += 4 {
		a := out.pix[i+3]
		var c [4]uint8
		if a != 0 {
			c = [4]uint8{unpremul(out.pix[i], a), unpremul(out.pix[i+1], a), unpremul(out.pix[i+2], a), a}
		}
		for k := 0; k < 4; k++ {
			if f[k] != nil {
				c[k] = f[k][c[k]]
			}
		}
		for k := 0; k < 3; k++ {
			out.pix[i+k] = premul(c[k], c[3])
		}
		out.pix[i+3] = c[3]
	}
	return out
}

// composite combines two images with a Porter-Duff operator or arithmetically; a is the first input (in) and b the
// second (in2).
//
// SVG 1.1: filters.html#feCompositeElement
func compositeImages(a, b *pixbuf, op string, k [4]float64) *pixbuf {
	out := newPixbuf(a.w, a.h)
	out.linear = a.linear
	mul := func(x, y uint8) int { return (int(x)*int(y) + 127) / 255 }
	for i := 0; i < len(out.pix); i += 4 {
		aa, ab := a.pix[i+3], b.pix[i+3]
		if op == "arithmetic" {
			var o [4]float64
			for c := 0; c < 4; c++ {
				x, y := float64(a.pix[i+c])/255, float64(b.pix[i+c])/255
				o[c] = math.Min(1, math.Max(0, k[0]*x*y+k[1]*x+k[2]*y+k[3]))
			}
			for c := 0; c < 3; c++ {
				o[c] = math.Min(o[c], o[3])
			}
			for c := 0; c < 4; c++ {
				out.pix[i+c] = uint8(math.Round(o[c] * 255))
			}
			continue
		}
		for c := 0; c < 4; c++ {
			x, y := a.pix[i+c], b.pix[i+c]
			var v int
			switch op {
			case "in":
				v = mul(x, ab)
			case "out":
				v = mul(x, 255-ab)
			case "atop":
				v = mul(x, ab) + mul(y, 255-aa)
			case "xor":
				v = mul(x, 255-ab) + mul(y, 255-aa)
			default: // over
				v = int(x) + mul(y, 255-aa)
			}
			out.pix[i+c] = uint8(min(255, v))
		}
	}
	return out
}

// mergeImages draws the images one over the other, the first at the bottom.
//
// SVG 1.1: filters.html#feMergeElement
func mergeImages(w, h int, layers []*pixbuf) *pixbuf {
	out := newPixbuf(w, h)
	if len(layers) > 0 {
		out.linear = layers[0].linear
	}
	for _, l := range layers {
		for i := 0; i < len(out.pix); i += 4 {
			a := l.pix[i+3]
			if a == 0 {
				continue
			}
			inv := 255 - int(a)
			for c := 0; c < 4; c++ {
				out.pix[i+c] = uint8(min(255, int(l.pix[i+c])+(int(out.pix[i+c])*inv+127)/255))
			}
		}
	}
	return out
}

// blendImages blends the top image (in) over the backdrop (in2) with a blend mode of the specification.
//
// SVG 1.1: filters.html#feBlendElement; Compositing and Blending Level 1: #blending
func blendImages(top, back *pixbuf, mode string) *pixbuf {
	out := newPixbuf(top.w, top.h)
	out.linear = top.linear
	sep := separableBlend(mode)
	for i := 0; i < len(out.pix); i += 4 {
		as, ab := float64(top.pix[i+3])/255, float64(back.pix[i+3])/255
		ar := as + ab - as*ab
		var cs, cb [3]float64
		if as > 0 {
			for c := 0; c < 3; c++ {
				cs[c] = math.Min(1, float64(top.pix[i+c])/255/as)
			}
		}
		if ab > 0 {
			for c := 0; c < 3; c++ {
				cb[c] = math.Min(1, float64(back.pix[i+c])/255/ab)
			}
		}
		var mix [3]float64
		if sep != nil {
			for c := 0; c < 3; c++ {
				mix[c] = sep(cb[c], cs[c])
			}
		} else {
			mix = nonSeparableBlend(mode, cb, cs)
		}
		for c := 0; c < 3; c++ {
			cr := (1-ab)*as*cs[c] + (1-as)*ab*cb[c] + as*ab*mix[c]
			out.pix[i+c] = uint8(math.Round(math.Min(ar, math.Max(0, cr)) * 255))
		}
		out.pix[i+3] = uint8(math.Round(ar * 255))
	}
	return out
}

// separableBlend returns the function B(cb, cs) of a blend mode that works on each channel alone, or nil.
func separableBlend(mode string) func(cb, cs float64) float64 {
	switch mode {
	case "multiply":
		return func(cb, cs float64) float64 { return cb * cs }
	case "screen":
		return func(cb, cs float64) float64 { return cb + cs - cb*cs }
	case "overlay":
		return func(cb, cs float64) float64 { return hardLight(cs, cb) }
	case "darken":
		return math.Min
	case "lighten":
		return math.Max
	case "color-dodge":
		return func(cb, cs float64) float64 {
			switch {
			case cb == 0:
				return 0
			case cs >= 1:
				return 1
			}
			return math.Min(1, cb/(1-cs))
		}
	case "color-burn":
		return func(cb, cs float64) float64 {
			switch {
			case cb >= 1:
				return 1
			case cs <= 0:
				return 0
			}
			return 1 - math.Min(1, (1-cb)/cs)
		}
	case "hard-light":
		return hardLight
	case "soft-light":
		return func(cb, cs float64) float64 {
			if cs <= 0.5 {
				return cb - (1-2*cs)*cb*(1-cb)
			}
			var d float64
			if cb <= 0.25 {
				d = ((16*cb-12)*cb + 4) * cb
			} else {
				d = math.Sqrt(cb)
			}
			return cb + (2*cs-1)*(d-cb)
		}
	case "difference":
		return func(cb, cs float64) float64 { return math.Abs(cb - cs) }
	case "exclusion":
		return func(cb, cs float64) float64 { return cb + cs - 2*cb*cs }
	case "hue", "saturation", "color", "luminosity":
		return nil
	}
	// normal, and any name that is not known.
	return func(_, cs float64) float64 { return cs }
}

func hardLight(cb, cs float64) float64 {
	if cs <= 0.5 {
		return cb * 2 * cs
	}
	s := 2*cs - 1
	return cb + s - cb*s
}

func lum(c [3]float64) float64 { return 0.3*c[0] + 0.59*c[1] + 0.11*c[2] }

func clipColor(c [3]float64) [3]float64 {
	l := lum(c)
	n, x := math.Min(c[0], math.Min(c[1], c[2])), math.Max(c[0], math.Max(c[1], c[2]))
	if n < 0 {
		for i := range c {
			c[i] = l + (c[i]-l)*l/(l-n)
		}
	}
	if x > 1 {
		for i := range c {
			c[i] = l + (c[i]-l)*(1-l)/(x-l)
		}
	}
	return c
}

func setLum(c [3]float64, l float64) [3]float64 {
	d := l - lum(c)
	return clipColor([3]float64{c[0] + d, c[1] + d, c[2] + d})
}

func sat(c [3]float64) float64 {
	return math.Max(c[0], math.Max(c[1], c[2])) - math.Min(c[0], math.Min(c[1], c[2]))
}

func setSat(c [3]float64, s float64) [3]float64 {
	idx := [3]int{0, 1, 2}
	// Sort the indices by value: min, mid, max.
	if c[idx[0]] > c[idx[1]] {
		idx[0], idx[1] = idx[1], idx[0]
	}
	if c[idx[1]] > c[idx[2]] {
		idx[1], idx[2] = idx[2], idx[1]
	}
	if c[idx[0]] > c[idx[1]] {
		idx[0], idx[1] = idx[1], idx[0]
	}
	var out [3]float64
	mn, md, mx := c[idx[0]], c[idx[1]], c[idx[2]]
	if mx > mn {
		out[idx[1]] = (md - mn) * s / (mx - mn)
		out[idx[2]] = s
	}
	return out
}

func nonSeparableBlend(mode string, cb, cs [3]float64) [3]float64 {
	switch mode {
	case "hue":
		return setLum(setSat(cs, sat(cb)), lum(cb))
	case "saturation":
		return setLum(setSat(cb, sat(cs)), lum(cb))
	case "color":
		return setLum(cs, lum(cb))
	}
	// luminosity
	return setLum(cb, lum(cs))
}

// flooded returns an image of the size with the color (straight, 0 to 255) at the opacity inside the rectangle.
func flooded(w, h int, r, g, b uint8, opacity float64, x0, y0, x1, y1 int) *pixbuf {
	out := newPixbuf(w, h)
	a := uint8(math.Round(math.Min(1, math.Max(0, opacity)) * 255))
	px := [4]uint8{premul(r, a), premul(g, a), premul(b, a), a}
	x0, y0, x1, y1 = max(x0, 0), max(y0, 0), min(x1, w), min(y1, h)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			copy(out.pix[4*(y*w+x):], px[:])
		}
	}
	return out
}

// alphaOnly returns a black image with the alpha of the image.
func (b *pixbuf) alphaOnly() *pixbuf {
	out := newPixbuf(b.w, b.h)
	out.linear = b.linear
	for i := 3; i < len(out.pix); i += 4 {
		out.pix[i] = b.pix[i]
	}
	return out
}

// colored returns an image that has the color (straight) where the image has alpha: the shape of the image filled.
func (b *pixbuf) colored(r, g, bl uint8) *pixbuf {
	out := newPixbuf(b.w, b.h)
	out.linear = b.linear
	for i := 0; i < len(out.pix); i += 4 {
		a := b.pix[i+3]
		out.pix[i], out.pix[i+1], out.pix[i+2], out.pix[i+3] = premul(r, a), premul(g, a), premul(bl, a), a
	}
	return out
}
