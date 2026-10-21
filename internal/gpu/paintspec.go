package gpu

import (
	"math"
	"sort"

	"github.com/rm4n0s/glutter/ui"
)

// A paint's shader, color filter, inverted colors and blend mode are the uniforms of a draw
// call (gles/shaders/paint.glsl, explicit/shaders/paint.wgsl.inc). A PaintSpec describes them;
// it is comparable, so that draws with the same paint can share a draw call. The two GPU
// renderers build the same specs and send them to their shaders.

// The shader kinds of the paint shaders (PaintSpec.Shader).
const (
	// ShaderNone is no shader: the paint's color.
	//
	// Flutter: no counterpart
	ShaderNone = 0
	// ShaderLinear is a linear gradient.
	//
	// Flutter: no counterpart
	ShaderLinear = 1
	// ShaderRadial is a radial gradient.
	//
	// Flutter: no counterpart
	ShaderRadial = 2
	// ShaderConical is a two-point conical gradient.
	//
	// Flutter: no counterpart
	ShaderConical = 3
	// ShaderSweep is a sweep gradient.
	//
	// Flutter: no counterpart
	ShaderSweep = 4
	// ShaderImage is an image shader.
	//
	// Flutter: no counterpart
	ShaderImage = 5
	// ShaderConst is a degenerate gradient: one color.
	//
	// Flutter: no counterpart
	ShaderConst = 6
)

// The color filter kinds of the paint shaders (PaintSpec.Filter).
const (
	// FilterNone is no color filter.
	//
	// Flutter: no counterpart
	FilterNone = 0
	// FilterMode blends a color with the paint by a blend mode.
	//
	// Flutter: no counterpart
	FilterMode = 1
	// FilterMatrix is a 4x5 color matrix.
	//
	// Flutter: no counterpart
	FilterMatrix = 2
	// FilterToSrgb is the linear to sRGB gamma.
	//
	// Flutter: no counterpart
	FilterToSrgb = 3
	// FilterFromSrgb is the sRGB to linear gamma.
	//
	// Flutter: no counterpart
	FilterFromSrgb = 4
)

// MaxStops is the most color stops a gradient can have; it is the size of the arrays of the
// paint shaders.
//
// Flutter: no counterpart
const MaxStops = 32

// PaintSpec is the shader, color filter and blend of a paint, as uniforms.
//
// Flutter: no counterpart
type PaintSpec struct {
	// Shader is one of the Shader constants.
	Shader int32
	// XF maps a pixel of the surface to the shader's space: a, b, c, d, e, f.
	XF [6]float32
	// P0 and P1 are the geometry of the gradient: linear: from x, y, dx, dy and
	// 1 / (dx*dx + dy*dy); radial: center x, y, radius; conical: focal x, y, focal radius and
	// end center x, y, end radius; sweep: center x, y, start, span.
	P0, P1 [4]float32
	// Tile is the tile mode in x and y: 0 clamp, 1 repeated, 2 mirror, 3 decal.
	Tile [2]int32

	// RampN is the number of color stops, RampCol their colors (not premultiplied) and RampPos
	// their positions.
	RampN   int32
	RampCol [MaxStops][4]float32
	RampPos [MaxStops]float32

	// Image is the renderer's texture of an image shader, and ImgInfo its quality (0 none, 1
	// low, 2 medium, 3 high) and level.
	Image   any
	ImgInfo [4]float32

	// ConstCol is the color of a degenerate gradient, premultiplied.
	ConstCol [4]float32
	// Dither says that the ordered dither of gradients is added.
	Dither bool

	// Filter is one of the Filter constants, FilterCol the premultiplied color of a mode filter,
	// FilterBlend its blend index, and FilterMat the columns of a matrix filter.
	Filter      int32
	FilterCol   [4]float32
	FilterBlend int32
	FilterMat   [20]float32
	// Invert says that the colors are inverted after the filter.
	Invert bool

	// ReadBlend is 0 when the fixed-function blend does the paint's blend mode, else the blend
	// index plus one: the shader blends with a copy of the pixels under the draw.
	ReadBlend int32
}

// PaintResult says what to do with a paint after a spec was made from it.
//
// Flutter: no counterpart
type PaintResult int

const (
	// PaintDraw says to draw with the spec.
	//
	// Flutter: no counterpart
	PaintDraw PaintResult = iota
	// PaintNothing says that the paint draws nothing: the command is done.
	//
	// Flutter: no counterpart
	PaintNothing
	// PaintUnsupported says that the renderer cannot draw the paint; the name of the feature
	// comes with it.
	//
	// Flutter: no counterpart
	PaintUnsupported
)

// TileIndex is the number of a tile mode in the shaders.
//
// Flutter: no counterpart
func TileIndex(m ui.TileMode) int32 {
	switch m {
	case ui.TileModeRepeated:
		return 1
	case ui.TileModeMirror:
		return 2
	case ui.TileModeDecal:
		return 3
	}
	return 0
}

// ramp is the colors of a gradient with their positions, built as Skia does:
// evenly spaced when there are no stops, a color at 0 and at 1 added when the
// stops do not reach them, and the stops kept in order.
type ramp struct {
	pos    []float64
	colors [][4]float32 // not premultiplied
}

func newRamp(colors []ui.Color, stops []float64) ramp {
	var r ramp
	n := len(colors)
	for i, c := range colors {
		p := float64(i) / math.Max(float64(n-1), 1)
		if len(stops) == n {
			p = stops[i]
		}
		col := [4]float32{c.R, c.G, c.B, c.A}
		if len(r.pos) > 0 {
			p = math.Max(p, r.pos[len(r.pos)-1])
		} else if p > 0 {
			r.pos = append(r.pos, 0)
			r.colors = append(r.colors, col)
		}
		p = math.Min(math.Max(p, 0), 1)
		r.pos = append(r.pos, p)
		r.colors = append(r.colors, col)
	}
	if last := r.pos[len(r.pos)-1]; last < 1 {
		r.pos = append(r.pos, 1)
		r.colors = append(r.colors, r.colors[len(r.colors)-1])
	}
	return r
}

// premultipliedAt returns the color at t, as the shader does.
func (r ramp) premultipliedAt(t float64) [4]float32 {
	i := sort.SearchFloat64s(r.pos, t)
	var c [4]float32
	switch {
	case i == 0:
		c = r.colors[0]
	case i >= len(r.pos):
		c = r.colors[len(r.colors)-1]
	default:
		p0, p1 := r.pos[i-1], r.pos[i]
		if p1 <= p0 {
			c = r.colors[i]
			break
		}
		f := float32((t - p0) / (p1 - p0))
		for k := range c {
			c[k] = r.colors[i-1][k] + (r.colors[i][k]-r.colors[i-1][k])*f
		}
	}
	for k := range c {
		c[k] = min(max(c[k], 0), 1)
	}
	return [4]float32{c[0] * c[3], c[1] * c[3], c[2] * c[3], c[3]}
}

// SetGradient fills the shader part of a spec for a gradient under the transform that takes the
// shader's local space to the surface. The result says what to do with the paint; for
// PaintUnsupported the string is the feature.
//
// Flutter: no counterpart
func (sp *PaintSpec) SetGradient(g *ui.Gradient, toDevice Affine2D) (PaintResult, string) {
	if len(g.Colors()) == 0 {
		return PaintNothing, ""
	}
	if len(g.Colors()) > MaxStops {
		return PaintUnsupported, "gradient with more than 32 colors"
	}
	r := newRamp(g.Colors(), g.ColorStops())
	if len(r.pos) > MaxStops {
		return PaintUnsupported, "gradient with more than 32 colors"
	}
	sp.Dither = true
	if m, ok := g.Matrix4(); ok {
		toDevice = AffineOf(m).Then(toDevice)
	}
	inv, ok := toDevice.Invert()
	if !ok {
		sp.Shader = ShaderConst
		return PaintDraw, ""
	}
	sp.setXF(inv)
	tile := g.TileMode()
	sp.Tile[0] = TileIndex(tile)
	sp.RampN = int32(len(r.pos))
	for i := range r.pos {
		sp.RampCol[i] = r.colors[i]
		sp.RampPos[i] = float32(r.pos[i])
	}
	switch g.Kind() {
	case ui.GradientKindLinear:
		from, to := g.From(), g.To()
		dx, dy := to.Dx-from.Dx, to.Dy-from.Dy
		den := dx*dx + dy*dy
		if den == 0 {
			// A degenerate gradient is the last color, or nothing for decal.
			sp.RampN, sp.RampCol, sp.RampPos = 0, [MaxStops][4]float32{}, [MaxStops]float32{}
			sp.XF = [6]float32{}
			if tile == ui.TileModeDecal {
				sp.Shader = ShaderConst
				return PaintDraw, ""
			}
			sp.Shader = ShaderConst
			sp.ConstCol = r.premultipliedAt(1)
			return PaintDraw, ""
		}
		sp.Shader = ShaderLinear
		sp.P0 = [4]float32{float32(from.Dx), float32(from.Dy), float32(dx), float32(dy)}
		sp.P1[0] = float32(1 / den)
	case ui.GradientKindRadial:
		c := g.Center()
		sp.Shader = ShaderRadial
		sp.P0 = [4]float32{float32(c.Dx), float32(c.Dy), float32(g.Radius()), 0}
	case ui.GradientKindConical:
		f, c := g.Focal(), g.Center()
		sp.Shader = ShaderConical
		sp.P0 = [4]float32{float32(f.Dx), float32(f.Dy), float32(g.FocalRadius()), 0}
		sp.P1 = [4]float32{float32(c.Dx), float32(c.Dy), float32(g.Radius()), 0}
	default:
		c := g.Center()
		sp.Shader = ShaderSweep
		sp.P0 = [4]float32{float32(c.Dx), float32(c.Dy), float32(g.StartAngle()), float32(g.EndAngle() - g.StartAngle())}
	}
	return PaintDraw, ""
}

func (sp *PaintSpec) setXF(inv Affine2D) {
	sp.XF = [6]float32{float32(inv.A), float32(inv.B), float32(inv.C), float32(inv.D), float32(inv.E), float32(inv.F)}
}

// SetImageShader fills the shader part of a spec for an image shader under the transform that
// takes the shader's local space to the surface. texture finds the renderer's texture of the
// image and its number of mip levels, making it if it must. The result says what to do with the
// paint; an error from texture is returned as it is.
//
// Flutter: no counterpart
func (sp *PaintSpec) SetImageShader(is *ui.ImageShader, toDevice Affine2D, texture func(img *ui.Image) (tex any, levels int, err error)) (PaintResult, error) {
	img := is.Image()
	inv, ok := AffineOf(is.Matrix4()).Then(toDevice).Invert()
	if !ok {
		sp.Shader = ShaderConst
		return PaintDraw, nil
	}
	tex, levels, err := texture(img)
	if err != nil {
		return PaintUnsupported, err
	}
	quality := is.FilterQuality()
	if quality == ui.FilterQualityUnspecified {
		quality = ui.FilterQualityLow
	}
	level := 0
	if quality == ui.FilterQualityMedium || quality == ui.FilterQualityHigh {
		// How many image pixels one surface pixel spans: the inverse transform's scale.
		if scale := math.Sqrt(math.Abs(inv.Det())); scale > 1 {
			level = min(int(math.Log2(scale)+0.5), levels-1)
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
	sp.setXF(inv)
	sp.Shader = ShaderImage
	sp.Tile = [2]int32{TileIndex(is.TileModeX()), TileIndex(is.TileModeY())}
	sp.Image = tex
	sp.ImgInfo = [4]float32{q, float32(level), 0, 0}
	return PaintDraw, nil
}

// SetColorFilter fills the color filter and inverted colors of a spec.
//
// Flutter: no counterpart
func (sp *PaintSpec) SetColorFilter(f *ui.ColorFilter, invert bool) {
	sp.Invert = invert
	if f == nil {
		return
	}
	switch f.Kind() {
	case ui.ColorFilterKindMode:
		c := f.Color()
		a := float32(c.Alpha()) / 255
		sp.Filter = FilterMode
		sp.FilterCol = [4]float32{float32(c.Red()) / 255 * a, float32(c.Green()) / 255 * a, float32(c.Blue()) / 255 * a, a}
		idx, _ := BlendIndex(f.BlendMode())
		sp.FilterBlend = int32(idx)
	case ui.ColorFilterKindMatrix:
		m := f.Matrix()
		sp.Filter = FilterMatrix
		// The uniform holds the matrix by columns: column c is the weights of input
		// component c in each of the four outputs.
		for col := 0; col < 5; col++ {
			for row := 0; row < 4; row++ {
				sp.FilterMat[col*4+row] = float32(m[row*5+col])
			}
		}
	case ui.ColorFilterKindLinearToSrgbGamma:
		sp.Filter = FilterToSrgb
	case ui.ColorFilterKindSrgbToLinearGamma:
		sp.Filter = FilterFromSrgb
	}
}
