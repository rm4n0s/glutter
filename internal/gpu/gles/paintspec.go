package gles

import (
	"math"
	"sort"

	"github.com/rm4n0s/glutter/ui"
)

// A paint's shader, color filter, inverted colors and blend mode are the uniforms
// of a draw call (shaders/paint.glsl). The builder describes them in a paintSpec,
// which is comparable and interned, so that draws with the same paint share a
// draw call.

// The shader kinds of paint.glsl.
const (
	shaderNone    = 0
	shaderLinear  = 1
	shaderRadial  = 2
	shaderConical = 3
	shaderSweep   = 4
	shaderImage   = 5
	shaderConst   = 6
)

// The color filter kinds of paint.glsl.
const (
	filterNone     = 0
	filterMode     = 1
	filterMatrix   = 2
	filterToSrgb   = 3
	filterFromSrgb = 4
)

// maxStops is the most color stops a gradient can have; it is the size of the
// uniform array of paint.glsl.
const maxStops = 32

// paintSpec is the shader, color filter and blend of a paint, as uniforms.
type paintSpec struct {
	shader int32
	// xf maps a pixel of the surface to the shader's space: a, b, c, d, e, f.
	xf [6]float32
	// p0 and p1 are the geometry of the gradient, see paint.glsl.
	p0, p1 [4]float32
	// tile is the tile mode in x and y: 0 clamp, 1 repeated, 2 mirror, 3 decal.
	tile [2]int32

	rampN   int32
	rampCol [maxStops][4]float32 // not premultiplied
	rampPos [maxStops]float32

	// image is the texture of an image shader, and imgInfo its quality and level.
	image   *Texture
	imgInfo [4]float32

	constCol [4]float32 // premultiplied
	dither   bool

	filter     int32
	filterCol  [4]float32 // premultiplied
	filterMode int32
	filterMat  [20]float32
	invert     bool

	// readBlend is 0 when the fixed-function blend does the paint's blend mode,
	// else the blend index plus one: the shader blends with a copy of the pixels
	// under the draw.
	readBlend int32
}

// intern returns the one paintSpec that equals sp.
func (b *frameBuilder) intern(sp paintSpec) *paintSpec {
	if p, ok := b.specs[sp]; ok {
		return p
	}
	if b.specs == nil {
		b.specs = map[paintSpec]*paintSpec{}
	}
	p := &sp
	b.specs[sp] = p
	return p
}

// tileIndex is the number of a tile mode in the shader.
func tileIndex(m ui.TileMode) int32 {
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

// blendIndex is the index of a blend mode in shaders/blend.glsl: its place in
// the order of Flutter's BlendMode.
func blendIndex(m ui.BlendMode) (int, bool) {
	if m == ui.BlendModeUnspecified {
		return int(ui.BlendModeSrcOver - ui.BlendModeClear), true
	}
	if m < ui.BlendModeClear || m > ui.BlendModeLuminosity {
		return 0, false
	}
	return int(m - ui.BlendModeClear), true
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

// shaderSpec fills the shader part of a spec for the shader sh under the
// transform that takes the shader's local space to the surface. status says what
// to do with the paint: paintNothing for a shader that draws nothing, and
// paintUnsupported with the error of unsupported for one that cannot be drawn.
func (s *shapeContext) shaderSpec(sp *paintSpec, sh ui.Shader, toDevice affine2D) (paintStatus, error) {
	switch v := sh.(type) {
	case *ui.Gradient:
		return s.gradientSpec(sp, v, toDevice)
	case *ui.ImageShader:
		return s.imageShaderSpec(sp, v, toDevice)
	}
	return paintUnsupported, s.b.unsupported("paint shader")
}

func setXf(sp *paintSpec, inv affine2D) {
	sp.xf = [6]float32{float32(inv.A), float32(inv.B), float32(inv.C), float32(inv.D), float32(inv.E), float32(inv.F)}
}

func (s *shapeContext) gradientSpec(sp *paintSpec, g *ui.Gradient, toDevice affine2D) (paintStatus, error) {
	if len(g.Colors()) == 0 {
		return paintNothing, nil
	}
	if len(g.Colors()) > maxStops {
		return paintUnsupported, s.b.unsupported("gradient with more than 32 colors")
	}
	r := newRamp(g.Colors(), g.ColorStops())
	if len(r.pos) > maxStops {
		return paintUnsupported, s.b.unsupported("gradient with more than 32 colors")
	}
	sp.dither = true
	if m, ok := g.Matrix4(); ok {
		toDevice = affineOf(m).Then(toDevice)
	}
	inv, ok := toDevice.Invert()
	if !ok {
		sp.shader = shaderConst
		return paintDraw, nil
	}
	setXf(sp, inv)
	tile := g.TileMode()
	sp.tile[0] = tileIndex(tile)
	sp.rampN = int32(len(r.pos))
	for i := range r.pos {
		sp.rampCol[i] = r.colors[i]
		sp.rampPos[i] = float32(r.pos[i])
	}
	switch g.Kind() {
	case ui.GradientKindLinear:
		from, to := g.From(), g.To()
		dx, dy := to.Dx-from.Dx, to.Dy-from.Dy
		den := dx*dx + dy*dy
		if den == 0 {
			// A degenerate gradient is the last color, or nothing for decal.
			sp.rampN, sp.rampCol, sp.rampPos = 0, [maxStops][4]float32{}, [maxStops]float32{}
			sp.xf = [6]float32{}
			if tile == ui.TileModeDecal {
				sp.shader = shaderConst
				return paintDraw, nil
			}
			sp.shader = shaderConst
			sp.constCol = r.premultipliedAt(1)
			return paintDraw, nil
		}
		sp.shader = shaderLinear
		sp.p0 = [4]float32{float32(from.Dx), float32(from.Dy), float32(dx), float32(dy)}
		sp.p1[0] = float32(1 / den)
	case ui.GradientKindRadial:
		c := g.Center()
		sp.shader = shaderRadial
		sp.p0 = [4]float32{float32(c.Dx), float32(c.Dy), float32(g.Radius()), 0}
	case ui.GradientKindConical:
		f, c := g.Focal(), g.Center()
		sp.shader = shaderConical
		sp.p0 = [4]float32{float32(f.Dx), float32(f.Dy), float32(g.FocalRadius()), 0}
		sp.p1 = [4]float32{float32(c.Dx), float32(c.Dy), float32(g.Radius()), 0}
	default:
		c := g.Center()
		sp.shader = shaderSweep
		sp.p0 = [4]float32{float32(c.Dx), float32(c.Dy), float32(g.StartAngle()), float32(g.EndAngle() - g.StartAngle())}
	}
	return paintDraw, nil
}

func (s *shapeContext) imageShaderSpec(sp *paintSpec, is *ui.ImageShader, toDevice affine2D) (paintStatus, error) {
	img := is.Image()
	if err := checkImage(img); err != nil {
		return paintUnsupported, err
	}
	if s.b.res == nil {
		return paintUnsupported, s.b.unsupported("image shader")
	}
	inv, ok := affineOf(is.Matrix4()).Then(toDevice).Invert()
	if !ok {
		sp.shader = shaderConst
		return paintDraw, nil
	}
	entry, err := s.b.res.imageTexture(img)
	if err != nil {
		return paintUnsupported, err
	}
	quality := is.FilterQuality()
	if quality == ui.FilterQualityUnspecified {
		quality = ui.FilterQualityLow
	}
	level := 0
	if quality == ui.FilterQualityMedium || quality == ui.FilterQualityHigh {
		// How many image pixels one surface pixel spans: the inverse transform's scale.
		if scale := math.Sqrt(math.Abs(inv.Det())); scale > 1 {
			level = min(int(math.Log2(scale)+0.5), entry.levels-1)
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
	setXf(sp, inv)
	sp.shader = shaderImage
	sp.tile = [2]int32{tileIndex(is.TileModeX()), tileIndex(is.TileModeY())}
	sp.image = entry.tex
	sp.imgInfo = [4]float32{q, float32(level), 0, 0}
	return paintDraw, nil
}

// filterSpec fills the color filter and inverted colors of a spec.
func filterSpec(sp *paintSpec, f *ui.ColorFilter, invert bool) {
	sp.invert = invert
	if f == nil {
		return
	}
	switch f.Kind() {
	case ui.ColorFilterKindMode:
		c := f.Color()
		a := float32(c.Alpha()) / 255
		sp.filter = filterMode
		sp.filterCol = [4]float32{float32(c.Red()) / 255 * a, float32(c.Green()) / 255 * a, float32(c.Blue()) / 255 * a, a}
		idx, _ := blendIndex(f.BlendMode())
		sp.filterMode = int32(idx)
	case ui.ColorFilterKindMatrix:
		m := f.Matrix()
		sp.filter = filterMatrix
		// The uniform holds the matrix by columns: column c is the weights of input
		// component c in each of the four outputs.
		for col := 0; col < 5; col++ {
			for row := 0; row < 4; row++ {
				sp.filterMat[col*4+row] = float32(m[row*5+col])
			}
		}
	case ui.ColorFilterKindLinearToSrgbGamma:
		sp.filter = filterToSrgb
	case ui.ColorFilterKindSrgbToLinearGamma:
		sp.filter = filterFromSrgb
	}
}
