package soft

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/ui"
)

func colorsOf(argb ...uint32) []ui.Color {
	out := make([]ui.Color, len(argb))
	for i, c := range argb {
		out[i] = ui.NewColor(c)
	}
	return out
}

// evenStops spaces n stops evenly from 0 to 1, which Flutter wants for more than
// two colors.
func evenStops(n int) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = float64(i) / float64(n-1)
	}
	return s
}

func rgbaNear(a, b rgba, tol float64) bool {
	for k := range a {
		if math.Abs(float64(a[k]-b[k])) > tol {
			return false
		}
	}
	return true
}

func TestTileT(t *testing.T) {
	tests := []struct {
		t    float64
		mode ui.TileMode
		want float64
		ok   bool
	}{
		{-0.5, ui.TileModeClamp, 0, true},
		{1.5, ui.TileModeClamp, 1, true},
		{0.25, ui.TileModeUnspecified, 0.25, true},
		{1.25, ui.TileModeRepeated, 0.25, true},
		{-0.25, ui.TileModeRepeated, 0.75, true},
		{1.25, ui.TileModeMirror, 0.75, true},
		{2.25, ui.TileModeMirror, 0.25, true},
		{-0.25, ui.TileModeMirror, 0.25, true},
		{0.5, ui.TileModeDecal, 0.5, true},
		{1.5, ui.TileModeDecal, 1.5, false},
		{-0.1, ui.TileModeDecal, -0.1, false},
	}
	for _, tt := range tests {
		got, ok := tileT(tt.t, tt.mode)
		if ok != tt.ok || (ok && math.Abs(got-tt.want) > 1e-12) {
			t.Errorf("tileT(%v, %v) = %v, %v, want %v, %v", tt.t, tt.mode, got, ok, tt.want, tt.ok)
		}
	}
}

func TestWrapIndex(t *testing.T) {
	tests := []struct {
		i, n int
		mode ui.TileMode
		want int
		ok   bool
	}{
		{-1, 4, ui.TileModeClamp, 0, true},
		{5, 4, ui.TileModeClamp, 3, true},
		{-1, 4, ui.TileModeRepeated, 3, true},
		{5, 4, ui.TileModeRepeated, 1, true},
		{4, 4, ui.TileModeMirror, 3, true},
		{7, 4, ui.TileModeMirror, 0, true},
		{-1, 4, ui.TileModeMirror, 0, true},
		{8, 4, ui.TileModeMirror, 0, true},
		{2, 4, ui.TileModeDecal, 2, true},
		{4, 4, ui.TileModeDecal, 4, false},
		{-1, 4, ui.TileModeDecal, -1, false},
	}
	for _, tt := range tests {
		got, ok := wrapIndex(tt.i, tt.n, tt.mode)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("wrapIndex(%d, %d, %v) = %v, %v, want %v, %v", tt.i, tt.n, tt.mode, got, ok, tt.want, tt.ok)
		}
	}
}

func TestColorRamp(t *testing.T) {
	red, green, blue := ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF)
	tests := []struct {
		name   string
		colors []ui.Color
		stops  []float64
		at     float64
		want   rgba
	}{
		{"two colors, start", []ui.Color{red, blue}, nil, 0, rgba{1, 0, 0, 1}},
		{"two colors, middle", []ui.Color{red, blue}, nil, 0.5, rgba{0.5, 0, 0.5, 1}},
		{"two colors, end", []ui.Color{red, blue}, nil, 1, rgba{0, 0, 1, 1}},
		{"three colors even", []ui.Color{red, green, blue}, nil, 0.5, rgba{0, 1, 0, 1}},
		{"three colors, a quarter", []ui.Color{red, green, blue}, nil, 0.25, rgba{0.5, 0.5, 0, 1}},
		{"stops move the middle color", []ui.Color{red, green, blue}, []float64{0, 0.25, 1}, 0.25, rgba{0, 1, 0, 1}},
		{"before the first stop", []ui.Color{red, blue}, []float64{0.5, 1}, 0.2, rgba{1, 0, 0, 1}},
		{"after the last stop", []ui.Color{red, blue}, []float64{0, 0.5}, 0.9, rgba{0, 0, 1, 1}},
		{"stops out of order are held", []ui.Color{red, green, blue}, []float64{0, 0.8, 0.4}, 0.9, rgba{0, 0, 1, 1}},
		{"equal stops are a hard edge", []ui.Color{red, blue}, []float64{0.5, 0.5}, 0.6, rgba{0, 0, 1, 1}},
		{"alpha is interpolated unpremultiplied", []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0x000000FF)}, nil, 0.5, rgba{0.5 * 0.5, 0, 0.5 * 0.5, 0.5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newColorRamp(tt.colors, tt.stops).at(tt.at); !rgbaNear(got, tt.want, 1e-5) {
				t.Errorf("at(%v) = %v, want %v", tt.at, got, tt.want)
			}
		})
	}
}

func TestGradientFunctions(t *testing.T) {
	bw := colorsOf(0xFF000000, 0xFFFFFFFF)
	gray := func(v float32) rgba { return rgba{v, v, v, 1} }
	at := func(g *ui.Gradient, x, y float64) rgba { return gradientFunc(g)(ui.Offset{Dx: x, Dy: y}) }
	tests := []struct {
		name string
		g    *ui.Gradient
		x, y float64
		want rgba
	}{
		{"linear start", ui.NewGradientLinear(ui.Offset{Dx: 10}, ui.Offset{Dx: 30}, bw), 10, 5, gray(0)},
		{"linear middle ignores the cross axis", ui.NewGradientLinear(ui.Offset{Dx: 10}, ui.Offset{Dx: 30}, bw), 20, 99, gray(0.5)},
		{"linear diagonal", ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 10, Dy: 10}, bw), 5, 5, gray(0.5)},
		{"linear clamps", ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 10}, bw), -5, 0, gray(0)},
		{"linear repeats", ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 10}, bw, ui.GradientLinearOpts{TileMode: ui.TileModeRepeated}), 12.5, 0, gray(0.25)},
		{"linear mirrors", ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 10}, bw, ui.GradientLinearOpts{TileMode: ui.TileModeMirror}), 12.5, 0, gray(0.75)},
		{"linear decal outside", ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 10}, bw, ui.GradientLinearOpts{TileMode: ui.TileModeDecal}), 12.5, 0, rgba{}},
		{"degenerate linear is the last color", ui.NewGradientLinear(ui.Offset{Dx: 3}, ui.Offset{Dx: 3}, bw), 0, 0, gray(1)},
		{"degenerate linear decal is nothing", ui.NewGradientLinear(ui.Offset{Dx: 3}, ui.Offset{Dx: 3}, bw, ui.GradientLinearOpts{TileMode: ui.TileModeDecal}), 0, 0, rgba{}},
		{"radial center", ui.NewGradientRadial(ui.Offset{Dx: 10, Dy: 10}, 10, bw), 10, 10, gray(0)},
		{"radial halfway", ui.NewGradientRadial(ui.Offset{Dx: 10, Dy: 10}, 10, bw), 10, 15, gray(0.5)},
		{"radial beyond clamps", ui.NewGradientRadial(ui.Offset{Dx: 10, Dy: 10}, 10, bw), 40, 10, gray(1)},
		{"sweep starts to the right", ui.NewGradientSweep(ui.Offset{}, bw), 5, 0.0001, gray(0)},
		{"sweep halfway is to the left", ui.NewGradientSweep(ui.Offset{}, bw), -5, 0, gray(0.5)},
		{"sweep goes clockwise: down is a quarter", ui.NewGradientSweep(ui.Offset{}, bw), 0, 5, gray(0.25)},
		{"sweep with angles", ui.NewGradientSweep(ui.Offset{}, bw, ui.GradientSweepOpts{StartAngle: math.Pi / 2, EndAngle: math.Pi}), 0, 5, gray(0)},
		{"sweep with angles, middle", ui.NewGradientSweep(ui.Offset{}, bw, ui.GradientSweepOpts{StartAngle: math.Pi / 2, EndAngle: math.Pi}), -5, 5, gray(0.5)},
		{"sweep clamps after the end", ui.NewGradientSweep(ui.Offset{}, bw, ui.GradientSweepOpts{StartAngle: 0, EndAngle: math.Pi}), 0, -5, gray(1)},
		{"sweep decal after the end", ui.NewGradientSweep(ui.Offset{}, bw, ui.GradientSweepOpts{StartAngle: 0, EndAngle: math.Pi, TileMode: ui.TileModeDecal}), 0, -5, rgba{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := at(tt.g, tt.x, tt.y); !rgbaNear(got, tt.want, 2e-3) {
				t.Errorf("at(%v, %v) = %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestConicalGradient(t *testing.T) {
	bw := colorsOf(0xFF000000, 0xFFFFFFFF)
	focal := ui.Offset{Dx: 4}
	// A focal point inside the end circle: every point is colored, and the color
	// grows with the distance from the focal point along each ray.
	g := ui.NewGradientRadial(ui.Offset{}, 10, bw, ui.GradientRadialOpts{Focal: &focal})
	f := gradientFunc(g)
	if got := f(ui.Offset{Dx: 4}); !rgbaNear(got, rgba{0, 0, 0, 1}, 1e-4) {
		t.Errorf("at the focal point = %v, want black", got)
	}
	if got := f(ui.Offset{Dx: 10}); !rgbaNear(got, rgba{1, 1, 1, 1}, 1e-4) {
		t.Errorf("on the end circle = %v, want white", got)
	}
	if got := f(ui.Offset{Dx: -10}); !rgbaNear(got, rgba{1, 1, 1, 1}, 1e-4) {
		t.Errorf("on the far side of the end circle = %v, want white", got)
	}
	if got := f(ui.Offset{Dx: 7}); got[0] < 0.4 || got[0] > 0.6 {
		t.Errorf("halfway along the ray = %v", got)
	}
	// A focal circle with a radius: the start circle is not a point.
	g2 := ui.NewGradientRadial(ui.Offset{}, 10, bw, ui.GradientRadialOpts{Focal: &focal, FocalRadius: 2})
	if got := gradientFunc(g2)(ui.Offset{Dx: 5}); !rgbaNear(got, rgba{0, 0, 0, 1}, 1e-4) {
		t.Errorf("inside the focal circle = %v, want black", got)
	}
	// Two circles that do not contain one another leave a gap, which decal does not paint.
	out := ui.Offset{Dx: 30}
	g3 := ui.NewGradientRadial(ui.Offset{}, 5, bw, ui.GradientRadialOpts{Focal: &out, FocalRadius: 5, TileMode: ui.TileModeDecal})
	f3 := gradientFunc(g3)
	if got := f3(ui.Offset{Dx: 0, Dy: 40}); got != (rgba{}) {
		t.Errorf("outside the cone = %v, want nothing", got)
	}
	// Equal-sized circles: a cylinder.
	right := ui.Offset{Dx: 20}
	g4 := ui.NewGradientRadial(right, 5, bw, ui.GradientRadialOpts{Focal: &ui.Offset{}, FocalRadius: 5, TileMode: ui.TileModeDecal})
	f4 := gradientFunc(g4)
	// The circle at t covers (10, 3) for t = 0.3 and t = 0.7; the larger one wins.
	if got := f4(ui.Offset{Dx: 10, Dy: 3}); !rgbaNear(got, rgba{0.7, 0.7, 0.7, 1}, 1e-3) {
		t.Errorf("inside the cylinder = %v", got)
	}
	if got := f4(ui.Offset{Dx: 10, Dy: 8}); got != (rgba{}) {
		t.Errorf("outside the cylinder = %v", got)
	}
}

func TestGradientPixelsAndDither(t *testing.T) {
	// A black to white ramp over 256 pixels: with dithering each pixel is within
	// one level of its exact value, and the ramp has no bands wider than a pixel or two.
	img := renderOf(t, 256, 4, func(c *ui.Canvas) {
		p := ui.NewPaint()
		p.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 256}, colorsOf(0xFF000000, 0xFFFFFFFF)))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 256, 4), p)
	})
	for x := 0; x < 256; x++ {
		want := (float64(x) + 0.5) / 256 * 255
		if got := float64(px(img, x, 1)[0]); math.Abs(got-want) > 1 {
			t.Fatalf("x=%d: %v, exact %v", x, got, want)
		}
	}
	// A shader scaled by the paint's alpha.
	img = renderOf(t, 4, 4, func(c *ui.Canvas) {
		p := ui.NewPaint()
		p.SetColor(ui.NewColor(0x80FFFFFF))
		p.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 4}, colorsOf(0xFFFF0000, 0xFFFF0000)))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 4, 4), p)
	})
	if got := px(img, 1, 1); got[3] < 127 || got[3] > 129 || got[0] < 127 || got[0] > 129 {
		t.Errorf("half-alpha red = %v", got)
	}
	// A solid color is not dithered: the same pixel everywhere.
	img = renderOf(t, 8, 8, func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 8, 8), paint(0xFF336699)) })
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if px(img, x, y) != [4]uint8{0x33, 0x66, 0x99, 255} {
				t.Fatalf("solid color at %d,%d = %v", x, y, px(img, x, y))
			}
		}
	}
}

func TestGradientMatrixAndTransform(t *testing.T) {
	// The gradient follows the canvas transform, and its own matrix applies first.
	grad := func(m *ui.Matrix4) *ui.Gradient {
		return ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 10}, colorsOf(0xFF000000, 0xFFFFFFFF), ui.GradientLinearOpts{Matrix4: m})
	}
	draw := func(translate float64, m *ui.Matrix4) *image.RGBA {
		return renderOf(t, 40, 4, func(c *ui.Canvas) {
			c.Translate(translate, 0)
			p := ui.NewPaint()
			p.SetShader(grad(m))
			c.DrawRect(ui.NewRectFromLTWH(-30, 0, 70, 4), p)
		})
	}
	plain := draw(10, nil)
	// Pixel 15's center is at 5.5 in the gradient's space, 55% of the way.
	if got := px(plain, 15, 1)[0]; got < 136 || got > 144 {
		t.Errorf("under a translation: %d, want about 140", got)
	}
	scale := ui.NewMatrix4Identity()
	scale.Scale(2, 1, 1)
	scaled := draw(0, scale)
	// The matrix doubles the gradient's length: pixel 10's center is 52.5% along.
	if got := px(scaled, 10, 1)[0]; got < 130 || got > 138 {
		t.Errorf("with a scaling matrix: %d, want about 134", got)
	}
}

func TestImageShader(t *testing.T) {
	tile := imageFunc(2, 2, func(x, y int) [4]uint8 {
		if (x+y)%2 == 0 {
			return [4]uint8{255, 0, 0, 255}
		}
		return [4]uint8{0, 0, 255, 255}
	})
	draw := func(tx, ty ui.TileMode, q ui.FilterQuality, m *ui.Matrix4) *image.RGBA {
		return renderOf(t, 8, 8, func(c *ui.Canvas) {
			p := ui.NewPaint()
			p.SetShader(ui.NewImageShader(tile, tx, ty, m, ui.ImageShaderOpts{FilterQuality: q}))
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 8, 8), p)
		})
	}
	red, blue, none := [4]uint8{255, 0, 0, 255}, [4]uint8{0, 0, 255, 255}, [4]uint8{}
	rep := draw(ui.TileModeRepeated, ui.TileModeRepeated, ui.FilterQualityNone, nil)
	for _, tt := range []struct {
		x, y int
		want [4]uint8
	}{{0, 0, red}, {1, 0, blue}, {2, 0, red}, {3, 1, red}, {7, 7, red}, {6, 7, blue}} {
		if got := px(rep, tt.x, tt.y); got != tt.want {
			t.Errorf("repeated: %d,%d = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
	mir := draw(ui.TileModeMirror, ui.TileModeMirror, ui.FilterQualityNone, nil)
	// Mirrored: pixels 2 and 3 are the image reversed, so pixel 2 repeats pixel 1.
	if px(mir, 2, 0) != px(mir, 1, 0) || px(mir, 3, 0) != px(mir, 0, 0) {
		t.Errorf("mirrored row = %v %v %v %v", px(mir, 0, 0), px(mir, 1, 0), px(mir, 2, 0), px(mir, 3, 0))
	}
	clamp := draw(ui.TileModeClamp, ui.TileModeClamp, ui.FilterQualityNone, nil)
	if px(clamp, 7, 0) != px(clamp, 1, 0) || px(clamp, 0, 7) != px(clamp, 0, 1) {
		t.Errorf("clamped edge pixels %v, %v", px(clamp, 7, 0), px(clamp, 0, 7))
	}
	decal := draw(ui.TileModeDecal, ui.TileModeDecal, ui.FilterQualityNone, nil)
	if px(decal, 1, 1) != red || px(decal, 2, 0) != none || px(decal, 0, 2) != none {
		t.Errorf("decal: %v %v %v", px(decal, 1, 1), px(decal, 2, 0), px(decal, 0, 2))
	}
	// The shader's matrix scales the tile: 4x makes each pixel a 4x4 block.
	scale := ui.NewMatrix4Identity()
	scale.Scale(4, 4, 1)
	big := draw(ui.TileModeRepeated, ui.TileModeRepeated, ui.FilterQualityNone, scale)
	if px(big, 3, 3) != red || px(big, 4, 3) != blue || px(big, 0, 4) != blue || px(big, 7, 7) != red {
		t.Errorf("scaled tile: %v %v %v %v", px(big, 3, 3), px(big, 4, 3), px(big, 0, 4), px(big, 7, 7))
	}
	// Bilinear smoothing mixes the two colors between texel centers.
	lin := draw(ui.TileModeRepeated, ui.TileModeRepeated, ui.FilterQualityLow, scale)
	if got := px(lin, 4, 0); got[0] == 0 || got[2] == 0 {
		t.Errorf("bilinear at a texel boundary = %v, want a mix", got)
	}
}

func TestGoldenGradients(t *testing.T) {
	tiles := []ui.TileMode{ui.TileModeClamp, ui.TileModeRepeated, ui.TileModeMirror, ui.TileModeDecal}
	rainbow := colorsOf(0xFFE53935, 0xFFFDD835, 0xFF43A047, 0xFF1E88E5)
	row := func(c *ui.Canvas, mk func(r ui.Rect, tm ui.TileMode) *ui.Gradient) {
		white(c)
		for i, tm := range tiles {
			r := ui.NewRectFromLTWH(float64(i%2)*64, float64(i/2)*64, 64, 64)
			c.Save()
			c.ClipRect(r.Deflate(2))
			p := ui.NewPaint()
			p.SetShader(mk(r.Deflate(18), tm))
			c.DrawRect(r, p)
			c.Restore()
		}
	}
	tests := map[string]func(r ui.Rect, tm ui.TileMode) *ui.Gradient{
		"linear": func(r ui.Rect, tm ui.TileMode) *ui.Gradient {
			return ui.NewGradientLinear(r.TopLeft(), r.BottomRight(), rainbow, ui.GradientLinearOpts{TileMode: tm, ColorStops: evenStops(4)})
		},
		"linear-stops": func(r ui.Rect, tm ui.TileMode) *ui.Gradient {
			return ui.NewGradientLinear(r.TopLeft(), r.TopRight(), rainbow, ui.GradientLinearOpts{TileMode: tm, ColorStops: []float64{0, 0.1, 0.9, 1}})
		},
		"radial": func(r ui.Rect, tm ui.TileMode) *ui.Gradient {
			return ui.NewGradientRadial(r.Center(), r.Width()/2, rainbow, ui.GradientRadialOpts{TileMode: tm, ColorStops: evenStops(4)})
		},
		"conical": func(r ui.Rect, tm ui.TileMode) *ui.Gradient {
			focal := ui.Offset{Dx: r.Left + 6, Dy: r.Top + 8}
			return ui.NewGradientRadial(r.Center(), r.Width()/2, rainbow, ui.GradientRadialOpts{TileMode: tm, Focal: &focal, FocalRadius: 3, ColorStops: evenStops(4)})
		},
		"sweep": func(r ui.Rect, tm ui.TileMode) *ui.Gradient {
			return ui.NewGradientSweep(r.Center(), rainbow, ui.GradientSweepOpts{TileMode: tm, StartAngle: 0.5, EndAngle: 4, ColorStops: evenStops(4)})
		},
	}
	for name, mk := range tests {
		t.Run(name, func(t *testing.T) {
			goldens.Check(t, "gradient-"+name, renderOf(t, 128, 128, func(c *ui.Canvas) { row(c, mk) }))
		})
	}
	t.Run("image-shader", func(t *testing.T) {
		tile := quads()
		goldens.Check(t, "shader-image-tiles", renderOf(t, 128, 128, func(c *ui.Canvas) {
			white(c)
			for i, tm := range tiles {
				r := ui.NewRectFromLTWH(float64(i%2)*64, float64(i/2)*64, 64, 64)
				c.Save()
				c.ClipRect(r.Deflate(2))
				m := ui.NewMatrix4Identity()
				m.Translate(float32(r.Left)+14, float32(r.Top)+14, 0)
				m.Scale(4, 4, 1)
				p := ui.NewPaint()
				p.SetShader(ui.NewImageShader(tile, tm, tm, m, ui.ImageShaderOpts{FilterQuality: ui.FilterQualityLow}))
				c.DrawRect(r, p)
				c.Restore()
			}
		}))
	})
}
