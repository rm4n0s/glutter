package proof

import (
	"github.com/rm4n0s/glutter/ui"
)

func init() {
	register(Scene{
		Name:        "gradients",
		Description: "linear, radial, conical, sweep and image shaders with every tile mode, and long ramps that show no banding",
		Width:       640,
		Height:      480,
		Build:       func() *ui.Scene { return singlePicture(640, 480, drawGradients) },
	})
}

func colorList(argb ...uint32) []ui.Color {
	out := make([]ui.Color, len(argb))
	for i, c := range argb {
		out[i] = ui.NewColor(c)
	}
	return out
}

// evenStops returns n color stops spaced evenly from 0 to 1, which Flutter asks
// for whenever a gradient has more than two colors.
func evenStops(n int) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = float64(i) / float64(n-1)
	}
	return s
}

func drawGradients(c *ui.Canvas) {
	const cw, ch = 160.0, 160.0
	rainbow := colorList(0xFFE53935, 0xFFFDD835, 0xFF43A047, 0xFF1E88E5)
	cell := func(col, row int, fn func(r, inner ui.Rect) ui.Shader) {
		r := ui.NewRectFromLTWH(float64(col)*cw, float64(row)*ch, cw, ch)
		c.Save()
		c.ClipRect(r.Deflate(4))
		p := ui.NewPaint()
		p.SetShader(fn(r, r.Deflate(44)))
		c.DrawRect(r, p)
		c.Restore()
	}
	tiles := []ui.TileMode{ui.TileModeClamp, ui.TileModeRepeated, ui.TileModeMirror, ui.TileModeDecal}
	for i, tm := range tiles {
		cell(i, 0, func(r, in ui.Rect) ui.Shader {
			return ui.NewGradientLinear(in.TopLeft(), in.BottomRight(), rainbow, ui.GradientLinearOpts{TileMode: tm, ColorStops: evenStops(len(rainbow))})
		})
	}
	cell(0, 1, func(r, in ui.Rect) ui.Shader {
		return ui.NewGradientRadial(r.Center(), 40, rainbow, ui.GradientRadialOpts{TileMode: ui.TileModeClamp, ColorStops: evenStops(len(rainbow))})
	})
	cell(1, 1, func(r, in ui.Rect) ui.Shader {
		return ui.NewGradientRadial(r.Center(), 30, rainbow, ui.GradientRadialOpts{TileMode: ui.TileModeMirror, ColorStops: evenStops(len(rainbow))})
	})
	cell(2, 1, func(r, in ui.Rect) ui.Shader {
		focal := ui.Offset{Dx: r.Center().Dx - 24, Dy: r.Center().Dy - 20}
		return ui.NewGradientRadial(r.Center(), 60, rainbow, ui.GradientRadialOpts{Focal: &focal, FocalRadius: 6, ColorStops: evenStops(len(rainbow))})
	})
	cell(3, 1, func(r, in ui.Rect) ui.Shader {
		return ui.NewGradientSweep(r.Center(), append(rainbow, rainbow[0]), ui.GradientSweepOpts{ColorStops: []float64{0, 0.25, 0.5, 0.75, 1}})
	})

	// Row 3: an image shader in two tile modes, and ramps without banding.
	tile := testImage()
	for i, tm := range []ui.TileMode{ui.TileModeRepeated, ui.TileModeMirror} {
		cell(i, 2, func(r, in ui.Rect) ui.Shader {
			m := ui.NewMatrix4Identity()
			m.Translate(float32(r.Left)+30, float32(r.Top)+30, 0)
			m.Scale(0.6, 0.6, 1)
			return ui.NewImageShader(tile, tm, tm, m, ui.ImageShaderOpts{FilterQuality: ui.FilterQualityLow})
		})
	}
	strip := func(y float64, colors []ui.Color) {
		r := ui.NewRectFromLTWH(2*cw+10, y, 2*cw-20, 36)
		p := ui.NewPaint()
		p.SetShader(ui.NewGradientLinear(r.TopLeft(), r.TopRight(), colors))
		c.DrawRect(r, p)
	}
	strip(2*ch+12, colorList(0xFF000000, 0xFFFFFFFF))
	strip(2*ch+56, colorList(0xFF1E3A8A, 0xFF1E88E5))
	strip(2*ch+100, colorList(0xFF202020, 0xFF2C2C2C))
}
