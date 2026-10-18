package gles

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// Image filters (SPEC §9.1) run as passes over a layer's pixels: each pass draws a
// rectangle over a pooled framebuffer and reads the framebuffer or texture before it
// (shaders/filter.*). A blur is two passes, one along each axis; when its deviation
// is large the picture is first averaged down to a smaller one and the blur is made
// there, then the result is scaled up again, so that the kernel stays short.

// minSigma is the deviation below which a blur does nothing, as in the software
// renderer.
const minSigma = 0.01

// maxLowSigma is the largest deviation a blur pass works with: a blur with more is
// done on a picture scaled down by a power of two until it is not.
const maxLowSigma = 8.0

// maxKernelRadius bounds the taps of a blur pass on each side of the center.
const maxKernelRadius = 64

// passKind is what a filter pass does.
type passKind int

const (
	passCopy   passKind = iota // the pixels as they are
	passBlur                   // one axis of a Gaussian blur
	passMorph                  // one axis of a dilation or erosion
	passMatrix                 // the pixels moved by a transform
	passColor                  // a color filter
	passDown                   // the average of blocks of pixels
	passUp                     // bilinear scaling up
)

// filterPass is one pass over the pixels.
type filterPass struct {
	kind passKind
	// outW and outH are the size of the pass's output.
	outW, outH int
	// axis is 0 for x and 1 for y, for blur and morphology.
	axis int32
	// sigma, radius and norm describe the Gaussian kernel: its deviation, the taps
	// on each side of the center, and 1 over the sum of the weights.
	sigma, norm float32
	radius      int32
	// tile is what lies beyond the edges of the source for blur and resampling.
	tile int32
	// dilate says that morphology takes the largest value, else the smallest.
	dilate bool
	// factor is how many source pixels one output pixel spans, for passDown and passUp.
	factor [2]int32
	// inv maps a pixel of the surface to where the output pixel is read from, for
	// passMatrix, and quality is 0 for bilinear, 1 for nearest, 2 for nothing.
	inv     [6]float32
	quality int32
	// spec holds the color filter of passColor.
	spec *paintSpec
}

// kernel fills the Gaussian kernel of a deviation.
func (p *filterPass) kernel(sigma float64) {
	r := min(int(math.Ceil(3*sigma)), maxKernelRadius)
	var sum float64
	for i := -r; i <= r; i++ {
		sum += math.Exp(-float64(i*i) / (2 * sigma * sigma))
	}
	p.sigma, p.radius, p.norm = float32(sigma), int32(r), float32(1/sum)
}

// blurPasses makes the passes of a blur of a w x h picture with the deviations in
// pixels along x and y.
func blurPasses(sx, sy float64, tile ui.TileMode, w, h int) []filterPass {
	blurs := func(s float64) bool { return s > minSigma && !math.IsInf(s, 0) }
	bx, by := blurs(sx), blurs(sy)
	fx, fy := 1, 1
	for bx && sx/float64(fx) > maxLowSigma && fx < 1<<12 {
		fx *= 2
	}
	for by && sy/float64(fy) > maxLowSigma && fy < 1<<12 {
		fy *= 2
	}
	lw, lh := (w+fx-1)/fx, (h+fy-1)/fy
	t := tileIndex(tile)
	var out []filterPass
	if fx > 1 || fy > 1 {
		out = append(out, filterPass{kind: passDown, outW: lw, outH: lh, tile: t, factor: [2]int32{int32(fx), int32(fy)}})
	}
	if bx {
		p := filterPass{kind: passBlur, outW: lw, outH: lh, tile: t}
		p.kernel(sx / float64(fx))
		out = append(out, p)
	}
	if by {
		p := filterPass{kind: passBlur, outW: lw, outH: lh, tile: t, axis: 1}
		p.kernel(sy / float64(fy))
		out = append(out, p)
	}
	if fx > 1 || fy > 1 {
		out = append(out, filterPass{kind: passUp, outW: w, outH: h, factor: [2]int32{int32(fx), int32(fy)}})
	}
	return out
}

// filterPasses turns an image filter into the passes over a w x h picture, under the
// transform that the filter's radii and matrix are in; tile is the mode a blur gets
// when it has none. false says that the filter is not one the renderer knows.
func filterPasses(f ui.ImageFilter, ctm affine2D, tile ui.TileMode, w, h int) ([]filterPass, bool) {
	sx, sy := math.Hypot(ctm.a, ctm.b), math.Hypot(ctm.c, ctm.d)
	switch v := f.(type) {
	case *ui.BlurImageFilter:
		t := v.TileMode()
		if t == ui.TileModeUnspecified {
			t = tile
		}
		return blurPasses(v.SigmaX()*sx, v.SigmaY()*sy, t, w, h), true
	case *ui.DilateImageFilter:
		return morphPasses(v.RadiusX()*sx, v.RadiusY()*sy, true, w, h), true
	case *ui.ErodeImageFilter:
		return morphPasses(v.RadiusX()*sx, v.RadiusY()*sy, false, w, h), true
	case *ui.MatrixImageFilter:
		return []filterPass{matrixPass(v, ctm, w, h)}, true
	case *ui.ComposeImageFilter:
		inner, ok := filterPasses(v.Inner(), ctm, ui.TileModeClamp, w, h)
		if !ok {
			return nil, false
		}
		outer, ok := filterPasses(v.Outer(), ctm, ui.TileModeClamp, w, h)
		return append(inner, outer...), ok
	case *ui.ColorFilter:
		var sp paintSpec
		filterSpec(&sp, v, false)
		return []filterPass{{kind: passColor, outW: w, outH: h, spec: &sp}}, true
	}
	return nil, false
}

// morphPasses makes the passes of a dilation or erosion by radii in pixels, rounded
// to whole pixels.
func morphPasses(rx, ry float64, dilate bool, w, h int) []filterPass {
	irx, iry := int(math.Round(rx)), int(math.Round(ry))
	var out []filterPass
	if irx > 0 {
		out = append(out, filterPass{kind: passMorph, outW: w, outH: h, radius: int32(min(irx, maxKernelRadius)), dilate: dilate})
	}
	if iry > 0 {
		out = append(out, filterPass{kind: passMorph, outW: w, outH: h, radius: int32(min(iry, maxKernelRadius)), dilate: dilate, axis: 1})
	}
	return out
}

// matrixPass moves the pixels by the filter's matrix, which acts in the local space
// of the canvas: on the surface the transform is ctm * M * ctm^-1.
func matrixPass(f *ui.MatrixImageFilter, ctm affine2D, w, h int) filterPass {
	p := filterPass{kind: passMatrix, outW: w, outH: h, quality: 2}
	inv, ok := ctm.invert()
	if !ok {
		return p
	}
	back, ok := inv.then(affineOf(f.Matrix())).then(ctm).invert()
	if !ok {
		return p
	}
	p.inv = [6]float32{float32(back.a), float32(back.b), float32(back.c), float32(back.d), float32(back.e), float32(back.f)}
	p.quality = 0
	if f.FilterQuality() == ui.FilterQualityNone {
		p.quality = 1
	}
	return p
}

// runPasses runs the passes over the w x h pixels of src, whose top left corner is
// the pixel origin of the surface, and returns the framebuffer that holds the result.
// With dst not nil the last pass draws into it; else the result is a pooled
// framebuffer that the caller releases. Nothing but the passes is drawn.
func (dr *drawer) runPasses(src *Texture, w, h int, origin [2]int, passes []filterPass, dst *Framebuffer) (*Framebuffer, error) {
	d := dr.r.ctx.dev
	gl := dr.r.ctx.gl
	prog, err := d.Program("filter")
	if err != nil {
		return nil, err
	}
	d.BindVertexArray(d.VertexArray("filter", func(uint32) {}))
	d.UseProgram(prog)
	dr.current = -1
	d.SetScissor(false, 0, 0, 0, 0)
	d.BlendFunc(glOne, glZero, glOne, glZero)
	gl.uniform1i(prog.Uniform("uSrc"), 0)
	if len(passes) == 0 {
		passes = []filterPass{{kind: passCopy, outW: w, outH: h}}
	}
	cur, curW, curH := src, w, h
	var owned, result *Framebuffer
	for i := range passes {
		ps := &passes[i]
		var out *Framebuffer
		if i == len(passes)-1 && dst != nil {
			out = dst
		} else if out, err = d.AcquireFramebuffer(ps.outW, ps.outH, false); err != nil {
			if owned != nil {
				d.ReleaseFramebuffer(owned)
			}
			return nil, err
		}
		d.BindFramebuffer(out)
		d.Viewport(0, 0, ps.outW, ps.outH)
		d.BindTexture(0, cur)
		gl.uniform1i(prog.Uniform("uPass"), int32(ps.kind))
		gl.uniform2f(prog.Uniform("uSrcSize"), float32(curW), float32(curH))
		gl.uniform2f(prog.Uniform("uOutSize"), float32(ps.outW), float32(ps.outH))
		gl.uniform1i(prog.Uniform("uAxis"), ps.axis)
		gl.uniform1i(prog.Uniform("uTile"), ps.tile)
		switch ps.kind {
		case passBlur:
			gl.uniform1f(prog.Uniform("uSigma"), ps.sigma)
			gl.uniform1i(prog.Uniform("uRadius"), ps.radius)
			gl.uniform1f(prog.Uniform("uNorm"), ps.norm)
		case passMorph:
			gl.uniform1i(prog.Uniform("uRadius"), ps.radius)
			gl.uniform1i(prog.Uniform("uDilate"), b2i(ps.dilate))
		case passDown, passUp:
			gl.uniform2f(prog.Uniform("uFactor"), float32(ps.factor[0]), float32(ps.factor[1]))
		case passMatrix:
			gl.uniform4f(prog.Uniform("uMat0"), ps.inv[0], ps.inv[1], ps.inv[2], ps.inv[3])
			gl.uniform2f(prog.Uniform("uMat1"), ps.inv[4], ps.inv[5])
			gl.uniform2f(prog.Uniform("uRectOrigin"), float32(origin[0]), float32(origin[1]))
			gl.uniform1i(prog.Uniform("uQuality"), ps.quality)
		case passColor:
			applySpec(gl, prog, ps.spec)
		}
		gl.drawArrays(glTriangleStrip, 0, 4)
		if owned != nil {
			d.ReleaseFramebuffer(owned)
			owned = nil
		}
		if out != dst {
			owned = out
		}
		cur, curW, curH = out.Texture(), ps.outW, ps.outH
		result = out
	}
	return result, nil
}
