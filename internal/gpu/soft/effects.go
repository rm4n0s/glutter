package soft

import (
	"image"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// applyMaskFilter returns the coverage of a shape after the paint's mask filter:
// a blur of the given style. The shape is src, in device pixels; the result
// covers the blurred shape within the pixels visible through the clip. The
// sigma is scaled by the transform, as Flutter's MaskFilter.blur respects the
// transform.
func applyMaskFilter(src *Mask, mf *ui.MaskFilter, m *ui.Matrix4, visible image.Rectangle) *Mask {
	if src == nil {
		return nil
	}
	sigma := mf.Sigma() * affineScale(m)
	if !(sigma > minSigma) {
		return src
	}
	out := padRect(src.Rect, sigma, sigma).Intersect(visible)
	blurred := blurredMask(src, sigma, sigma, out)
	if blurred == nil && mf.Style() != ui.BlurStyleSolid {
		return nil
	}
	style := mf.Style()
	if style == ui.BlurStyleUnspecified || style == ui.BlurStyleNormal {
		return blurred
	}
	// The other styles combine the blur with the shape itself: outer keeps what
	// is outside, inner what is inside, solid is the shape plus the blur outside.
	res := &Mask{Rect: out, Cov: make([]float32, out.Dx()*out.Dy())}
	any := false
	for y := out.Min.Y; y < out.Max.Y; y++ {
		for x := out.Min.X; x < out.Max.X; x++ {
			orig, b := src.At(x, y), blurred.At(x, y)
			var v float32
			switch style {
			case ui.BlurStyleOuter:
				v = b * (1 - orig)
			case ui.BlurStyleInner:
				v = b * orig
			default: // solid
				v = orig + b*(1-orig)
			}
			if v >= 1.0/1024 {
				res.Cov[(y-out.Min.Y)*out.Dx()+(x-out.Min.X)] = v
				any = true
			}
		}
	}
	if !any {
		return nil
	}
	return res
}

// Skia's constants for the shadows of Flutter's physical shapes
// (SkShadowUtils, flutter/display_list's DrawShadow).
const (
	shadowAmbientAlpha  = 0.039
	shadowSpotAlpha     = 0.25
	shadowLightHeight   = 600.0
	shadowLightRadius   = 800.0
	shadowAmbientFactor = 0.5 // blur radius per unit of height: 64 / 128
	shadowMaxAmbient    = 300.0
)

// blurRadiusToSigma is Skia's conversion of a blur radius to a deviation.
func blurRadiusToSigma(radius float64) float64 {
	if radius > 0 {
		return 0.57735*radius + 0.5
	}
	return 0
}

// drawShadow draws the shadow of a path the way Flutter's engine does for
// Canvas.drawShadow: an ambient shadow, a blur around the shape that grows with
// its elevation, and a spot shadow, the shape moved away from a light above it
// and blurred. Both are in the shadow color with alpha scaled by 0.039 and 0.25.
// The elevation is in logical pixels and is scaled by the transform's scale, which
// stands for the device pixel ratio that Flutter's engine multiplies it by. Unless
// the occluder is transparent the shadow is not drawn inside the shape.
//
// Deviation: Skia tessellates the shadow geometry; this blurs the shape's
// coverage, which has the same extent and falloff but not the same pixels.
func (r *renderer) drawShadow(op ui.OpDrawShadow, st *state) error {
	scale := affineScale(st.m)
	z := op.Elevation * scale
	if st.clip.empty() || !(z > 0) {
		return nil
	}
	visible := st.clip.limit(r.img.Rect)
	if visible.Empty() {
		return nil
	}
	// The shape's coverage in an area wide enough for the widest blur.
	ambientSigma := blurRadiusToSigma(math.Min(z*shadowAmbientFactor, shadowMaxAmbient))
	lightZ, lightR := shadowLightHeight*scale, shadowLightRadius*scale
	if z >= lightZ {
		return nil
	}
	spotSigma := blurRadiusToSigma(lightR * z / (lightZ - z))
	reach := math.Max(ambientSigma, spotSigma)
	bounds := op.Path.GetBounds()
	devBounds := deviceBounds(st.m, bounds)
	if devBounds.Empty() {
		return nil
	}
	// The spot shadow moves down and grows, away from a light that is above the
	// shape and 600 pixels above its top.
	spotScale := lightZ / (lightZ - z)
	height := float64(devBounds.Dy())
	dy := z * (shadowLightHeight*scale + height/2) / (lightZ - z)
	center := ui.Offset{Dx: float64(devBounds.Min.X+devBounds.Max.X) / 2, Dy: float64(devBounds.Min.Y+devBounds.Max.Y) / 2}

	// The shape's coverage is needed wherever a blur of it can reach the clip.
	expand := int(math.Ceil(3*reach + dy))
	area := devBounds.Intersect(st.clip.bounds.Inset(-expand))
	shape := Rasterize(flatPolys(op.Path, st.m), fillRuleOf(op.Path), area)
	if shape == nil {
		return nil
	}
	color := op.Color
	paintShadow := func(mask *Mask, alpha float64) {
		if mask == nil {
			return
		}
		c := color
		c.A = float32(float64(c.A) * alpha)
		r.composite(mask, st.clip, paintSetup{src: solid(c), mode: ui.BlendModeSrcOver})
	}
	cut := func(mask *Mask) *Mask {
		if mask == nil || op.TransparentOccluder {
			return mask
		}
		out := &Mask{Rect: mask.Rect, Cov: append([]float32(nil), mask.Cov...)}
		for y := out.Rect.Min.Y; y < out.Rect.Max.Y; y++ {
			for x := out.Rect.Min.X; x < out.Rect.Max.X; x++ {
				out.Cov[(y-out.Rect.Min.Y)*out.Rect.Dx()+(x-out.Rect.Min.X)] *= 1 - shape.At(x, y)
			}
		}
		return out
	}

	// The ambient shadow: the shape blurred in place.
	paintShadow(cut(blurredMask(shape, ambientSigma, ambientSigma, padRect(shape.Rect, ambientSigma, ambientSigma).Intersect(visible))), shadowAmbientAlpha)

	// The spot shadow: the shape grown about its center and moved down, blurred.
	moved := transformMask(shape, affine{a: spotScale, d: spotScale, e: center.Dx * (1 - spotScale), f: center.Dy*(1-spotScale) + dy})
	paintShadow(cut(blurredMask(moved, spotSigma, spotSigma, padRect(moved.Rect, spotSigma, spotSigma).Intersect(visible))), shadowSpotAlpha)
	return nil
}

// transformMask returns the coverage of a mask resampled under a scale and
// translation, with bilinear filtering.
func transformMask(m *Mask, t affine) *Mask {
	corners := []ui.Offset{t.apply(float64(m.Rect.Min.X), float64(m.Rect.Min.Y)), t.apply(float64(m.Rect.Max.X), float64(m.Rect.Max.Y))}
	out := image.Rect(int(math.Floor(math.Min(corners[0].Dx, corners[1].Dx))), int(math.Floor(math.Min(corners[0].Dy, corners[1].Dy))),
		int(math.Ceil(math.Max(corners[0].Dx, corners[1].Dx))), int(math.Ceil(math.Max(corners[0].Dy, corners[1].Dy))))
	inv, ok := t.invert()
	if !ok || out.Empty() {
		return nil
	}
	res := &Mask{Rect: out, Cov: make([]float32, out.Dx()*out.Dy())}
	for y := out.Min.Y; y < out.Max.Y; y++ {
		for x := out.Min.X; x < out.Max.X; x++ {
			p := inv.apply(float64(x)+0.5, float64(y)+0.5)
			fx, fy := p.Dx-0.5, p.Dy-0.5
			x0, y0 := math.Floor(fx), math.Floor(fy)
			tx, ty := float32(fx-x0), float32(fy-y0)
			ix, iy := int(x0), int(y0)
			top := m.At(ix, iy)*(1-tx) + m.At(ix+1, iy)*tx
			bot := m.At(ix, iy+1)*(1-tx) + m.At(ix+1, iy+1)*tx
			res.Cov[(y-out.Min.Y)*out.Dx()+(x-out.Min.X)] = top*(1-ty) + bot*ty
		}
	}
	return res
}

// applyImageFilter filters a layer's pixels and returns a new image of the same
// rectangle. ctm is the transform in force where the filter was set, which the
// radii and the matrix of the filter are in; unspecifiedTile is the tile mode a
// blur gets when it has none.
func applyImageFilter(layer *image.RGBA, f ui.ImageFilter, ctm affine, unspecifiedTile ui.TileMode) (*image.RGBA, error) {
	switch v := f.(type) {
	case *ui.BlurImageFilter:
		tile := v.TileMode()
		if tile == ui.TileModeUnspecified {
			tile = unspecifiedTile
		}
		sx := v.SigmaX() * math.Hypot(ctm.a, ctm.b)
		sy := v.SigmaY() * math.Hypot(ctm.c, ctm.d)
		return blurImage(layer, sx, sy, tile), nil
	case *ui.DilateImageFilter:
		return morphology(layer, v.RadiusX()*math.Hypot(ctm.a, ctm.b), v.RadiusY()*math.Hypot(ctm.c, ctm.d), true), nil
	case *ui.ErodeImageFilter:
		return morphology(layer, v.RadiusX()*math.Hypot(ctm.a, ctm.b), v.RadiusY()*math.Hypot(ctm.c, ctm.d), false), nil
	case *ui.MatrixImageFilter:
		return matrixFilter(layer, v, ctm), nil
	case *ui.ComposeImageFilter:
		inner, err := applyImageFilter(layer, v.Inner(), ctm, ui.TileModeClamp)
		if err != nil {
			return nil, err
		}
		return applyImageFilter(inner, v.Outer(), ctm, ui.TileModeClamp)
	case *ui.ColorFilter:
		fn := colorFilterFunc(v)
		out := image.NewRGBA(layer.Rect)
		for y := layer.Rect.Min.Y; y < layer.Rect.Max.Y; y++ {
			for x := layer.Rect.Min.X; x < layer.Rect.Max.X; x++ {
				c := load(layer, x, y)
				if fn != nil {
					c = fn(c)
				}
				store(out, x, y, [4]float32{c[0] * 255, c[1] * 255, c[2] * 255, c[3] * 255}, 0)
			}
		}
		return out, nil
	}
	return nil, errors.NewErrf("SoftNotImplemented", "The software renderer does not support the image filter %T.", f)
}

// morphology grows (dilate) or shrinks (erode) the shapes of the image: each
// channel becomes the largest or the smallest within the radii, which are
// rounded to whole pixels. Beyond the edges the image is transparent.
func morphology(img *image.RGBA, rx, ry float64, dilate bool) *image.RGBA {
	irx, iry := int(math.Round(rx)), int(math.Round(ry))
	if irx <= 0 && iry <= 0 {
		return img
	}
	irx, iry = max(irx, 0), max(iry, 0)
	pass := func(src *image.RGBA, dx, dy int) *image.RGBA {
		out := image.NewRGBA(src.Rect)
		for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
			for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
				var acc rgba
				if !dilate {
					acc = rgba{1, 1, 1, 1}
				}
				for k := -max(dx, dy); k <= max(dx, dy); k++ {
					px, py := x+k*sign(dx), y+k*sign(dy)
					var c rgba
					if (image.Point{X: px, Y: py}).In(src.Rect) {
						c = load(src, px, py)
					}
					for ch := range acc {
						if dilate {
							acc[ch] = max(acc[ch], c[ch])
						} else {
							acc[ch] = min(acc[ch], c[ch])
						}
					}
				}
				store(out, x, y, [4]float32{acc[0] * 255, acc[1] * 255, acc[2] * 255, acc[3] * 255}, 0)
			}
		}
		return out
	}
	out := img
	if irx > 0 {
		out = pass(out, irx, 0)
	}
	if iry > 0 {
		out = pass(out, 0, iry)
	}
	return out
}

func sign(v int) int {
	if v > 0 {
		return 1
	}
	return 0
}

// matrixFilter draws the image transformed by the filter's matrix, which acts in
// the local space of the canvas: the device transform is ctm * M * ctm^-1.
func matrixFilter(img *image.RGBA, f *ui.MatrixImageFilter, ctm affine) *image.RGBA {
	inv, ok := ctm.invert()
	out := image.NewRGBA(img.Rect)
	if !ok {
		return out
	}
	dev := inv.then(affineOf(f.Matrix())).then(ctm)
	back, ok := dev.invert()
	if !ok {
		return out
	}
	quality := f.FilterQuality()
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			p := back.apply(float64(x)+0.5, float64(y)+0.5)
			c := sampleImage(img, p.Dx, p.Dy, quality)
			store(out, x, y, [4]float32{c[0] * 255, c[1] * 255, c[2] * 255, c[3] * 255}, 0)
		}
	}
	return out
}

// sampleImage reads a surface at a point of device space, nearest or bilinear,
// transparent outside it.
func sampleImage(img *image.RGBA, x, y float64, q ui.FilterQuality) rgba {
	texel := func(ix, iy int) rgba {
		if !(image.Point{X: ix, Y: iy}).In(img.Rect) {
			return rgba{}
		}
		return load(img, ix, iy)
	}
	if q == ui.FilterQualityNone {
		return texel(int(math.Floor(x)), int(math.Floor(y)))
	}
	fx, fy := x-0.5, y-0.5
	x0, y0 := math.Floor(fx), math.Floor(fy)
	tx, ty := float32(fx-x0), float32(fy-y0)
	ix, iy := int(x0), int(y0)
	c00, c10, c01, c11 := texel(ix, iy), texel(ix+1, iy), texel(ix, iy+1), texel(ix+1, iy+1)
	var out rgba
	for k := range out {
		top := c00[k] + (c10[k]-c00[k])*tx
		bot := c01[k] + (c11[k]-c01[k])*tx
		out[k] = top + (bot-top)*ty
	}
	return out
}
