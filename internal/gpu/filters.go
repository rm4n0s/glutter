package gpu

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// Image filters (SPEC §9.1) run as passes over a layer's pixels: each pass draws a rectangle
// over a pooled render target and reads the target or texture before it. The planning of the
// passes is shared by the GPU renderers; each runs them with its own shaders (gles/shaders/
// filter.*, explicit/shaders/filter.wgsl). A blur is two passes, one along each axis; when its
// deviation is large the picture is first averaged down to a smaller one and the blur is made
// there, then the result is scaled up again, so that the kernel stays short.

// MinSigma is the deviation below which a blur does nothing, as in the software
// renderer.
//
// Flutter: no counterpart
const MinSigma = 0.01

// MaxLowSigma is the largest deviation a blur pass works with: a blur with more is
// done on a picture scaled down by a power of two until it is not.
//
// Flutter: no counterpart
const MaxLowSigma = 8.0

// MaxKernelRadius bounds the taps of a blur pass on each side of the center.
//
// Flutter: no counterpart
const MaxKernelRadius = 64

// PassKind is what a filter pass does.
//
// Flutter: no counterpart
type PassKind int

const (
	// PassCopy is the pixels as they are.
	//
	// Flutter: no counterpart
	PassCopy PassKind = iota
	// PassBlur is one axis of a Gaussian blur.
	//
	// Flutter: no counterpart
	PassBlur
	// PassMorph is one axis of a dilation or erosion.
	//
	// Flutter: no counterpart
	PassMorph
	// PassMatrix is the pixels moved by a transform.
	//
	// Flutter: no counterpart
	PassMatrix
	// PassColor is a color filter.
	//
	// Flutter: no counterpart
	PassColor
	// PassDown is the average of blocks of pixels.
	//
	// Flutter: no counterpart
	PassDown
	// PassUp is bilinear scaling up.
	//
	// Flutter: no counterpart
	PassUp
)

// FilterPass is one pass over the pixels.
//
// Flutter: no counterpart
type FilterPass struct {
	Kind PassKind
	// OutW and OutH are the size of the pass's output.
	OutW, OutH int
	// Axis is 0 for x and 1 for y, for blur and morphology.
	Axis int32
	// Sigma, Radius and Norm describe the Gaussian kernel: its deviation, the taps
	// on each side of the center, and 1 over the sum of the weights.
	Sigma, Norm float32
	Radius      int32
	// Tile is what lies beyond the edges of the source for blur and resampling.
	Tile int32
	// Dilate says that morphology takes the largest value, else the smallest.
	Dilate bool
	// Factor is how many source pixels one output pixel spans, for PassDown and PassUp.
	Factor [2]int32
	// Inv maps a pixel of the surface to where the output pixel is read from, for
	// PassMatrix, and Quality is 0 for bilinear, 1 for nearest, 2 for nothing.
	Inv     [6]float32
	Quality int32
	// Spec holds the color filter of PassColor.
	Spec *PaintSpec
}

// kernel fills the Gaussian kernel of a deviation.
func (p *FilterPass) kernel(sigma float64) {
	r := min(int(math.Ceil(3*sigma)), MaxKernelRadius)
	var sum float64
	for i := -r; i <= r; i++ {
		sum += math.Exp(-float64(i*i) / (2 * sigma * sigma))
	}
	p.Sigma, p.Radius, p.Norm = float32(sigma), int32(r), float32(1/sum)
}

// BlurPasses makes the passes of a blur of a w x h picture with the deviations in
// pixels along x and y.
//
// Flutter: no counterpart
func BlurPasses(sx, sy float64, tile ui.TileMode, w, h int) []FilterPass {
	blurs := func(s float64) bool { return s > MinSigma && !math.IsInf(s, 0) }
	bx, by := blurs(sx), blurs(sy)
	fx, fy := 1, 1
	for bx && sx/float64(fx) > MaxLowSigma && fx < 1<<12 {
		fx *= 2
	}
	for by && sy/float64(fy) > MaxLowSigma && fy < 1<<12 {
		fy *= 2
	}
	lw, lh := (w+fx-1)/fx, (h+fy-1)/fy
	t := TileIndex(tile)
	var out []FilterPass
	if fx > 1 || fy > 1 {
		out = append(out, FilterPass{Kind: PassDown, OutW: lw, OutH: lh, Tile: t, Factor: [2]int32{int32(fx), int32(fy)}})
	}
	if bx {
		p := FilterPass{Kind: PassBlur, OutW: lw, OutH: lh, Tile: t}
		p.kernel(sx / float64(fx))
		out = append(out, p)
	}
	if by {
		p := FilterPass{Kind: PassBlur, OutW: lw, OutH: lh, Tile: t, Axis: 1}
		p.kernel(sy / float64(fy))
		out = append(out, p)
	}
	if fx > 1 || fy > 1 {
		out = append(out, FilterPass{Kind: PassUp, OutW: w, OutH: h, Factor: [2]int32{int32(fx), int32(fy)}})
	}
	return out
}

// FilterPasses turns an image filter into the passes over a w x h picture, under the
// transform that the filter's radii and matrix are in; tile is the mode a blur gets
// when it has none. false says that the filter is not one the renderer knows.
//
// Flutter: no counterpart
func FilterPasses(f ui.ImageFilter, ctm Affine2D, tile ui.TileMode, w, h int) ([]FilterPass, bool) {
	sx, sy := math.Hypot(ctm.A, ctm.B), math.Hypot(ctm.C, ctm.D)
	switch v := f.(type) {
	case *ui.BlurImageFilter:
		t := v.TileMode()
		if t == ui.TileModeUnspecified {
			t = tile
		}
		return BlurPasses(v.SigmaX()*sx, v.SigmaY()*sy, t, w, h), true
	case *ui.DilateImageFilter:
		return MorphPasses(v.RadiusX()*sx, v.RadiusY()*sy, true, w, h), true
	case *ui.ErodeImageFilter:
		return MorphPasses(v.RadiusX()*sx, v.RadiusY()*sy, false, w, h), true
	case *ui.MatrixImageFilter:
		return []FilterPass{MatrixPass(v, ctm, w, h)}, true
	case *ui.ComposeImageFilter:
		inner, ok := FilterPasses(v.Inner(), ctm, ui.TileModeClamp, w, h)
		if !ok {
			return nil, false
		}
		outer, ok := FilterPasses(v.Outer(), ctm, ui.TileModeClamp, w, h)
		return append(inner, outer...), ok
	case *ui.ColorFilter:
		var sp PaintSpec
		sp.SetColorFilter(v, false)
		return []FilterPass{{Kind: PassColor, OutW: w, OutH: h, Spec: &sp}}, true
	}
	return nil, false
}

// MorphPasses makes the passes of a dilation or erosion by radii in pixels, rounded
// to whole pixels.
//
// Flutter: no counterpart
func MorphPasses(rx, ry float64, dilate bool, w, h int) []FilterPass {
	irx, iry := int(math.Round(rx)), int(math.Round(ry))
	var out []FilterPass
	if irx > 0 {
		out = append(out, FilterPass{Kind: PassMorph, OutW: w, OutH: h, Radius: int32(min(irx, MaxKernelRadius)), Dilate: dilate})
	}
	if iry > 0 {
		out = append(out, FilterPass{Kind: PassMorph, OutW: w, OutH: h, Radius: int32(min(iry, MaxKernelRadius)), Dilate: dilate, Axis: 1})
	}
	return out
}

// MatrixPass moves the pixels by the filter's matrix, which acts in the local space
// of the canvas: on the surface the transform is ctm * M * ctm^-1.
//
// Flutter: no counterpart
func MatrixPass(f *ui.MatrixImageFilter, ctm Affine2D, w, h int) FilterPass {
	p := FilterPass{Kind: PassMatrix, OutW: w, OutH: h, Quality: 2}
	inv, ok := ctm.Invert()
	if !ok {
		return p
	}
	back, ok := inv.Then(AffineOf(f.Matrix())).Then(ctm).Invert()
	if !ok {
		return p
	}
	p.Inv = [6]float32{float32(back.A), float32(back.B), float32(back.C), float32(back.D), float32(back.E), float32(back.F)}
	p.Quality = 0
	if f.FilterQuality() == ui.FilterQualityNone {
		p.Quality = 1
	}
	return p
}
