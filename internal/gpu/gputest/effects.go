package gputest

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// EffectSize is the width and height of the surface of the effect scenes.
const EffectSize = 128

// EffectScene is a drawing of the paint shaders, color filters, blend modes and layers, with
// the tolerance the GPU renderers are held to against the software renderer.
type EffectScene struct {
	Name      string
	Tolerance Tolerance
	// Build makes the scene, EffectSize pixels square.
	Build func() *ui.Scene
}

// PictureOf records a picture of drawing.
func PictureOf(fn func(c *ui.Canvas)) *ui.Picture {
	rec := ui.NewPictureRecorder()
	fn(ui.NewCanvas(rec))
	return rec.EndRecording()
}

// StripesAndDisc is something with an edge and a gradient to put layers over.
func StripesAndDisc(c *ui.Canvas) {
	for i := 0; i < 8; i++ {
		c.DrawRect(ui.NewRectFromLTWH(0, float64(i)*16, 128, 8), FillPaint(0xFF3050C0))
		c.DrawRect(ui.NewRectFromLTWH(0, float64(i)*16+8, 128, 8), FillPaint(0xFFF0A020))
	}
	g := ui.NewPaint()
	g.SetShader(ui.NewGradientRadial(ui.Offset{Dx: 64, Dy: 64}, 40, []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0x4000FF00)}))
	c.DrawCircle(ui.Offset{Dx: 64, Dy: 64}, 40, g)
}

// ShaderPaint is a paint with a shader.
func ShaderPaint(sh ui.Shader) *ui.Paint {
	p := ui.NewPaint()
	p.SetShader(sh)
	return p
}

// Pattern is an n x n image of a color gradient with a grid and a transparent corner, to
// show orientation, filtering and alpha.
func Pattern(n int) *ui.Image {
	pix := make([]byte, 4*n*n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			r, g, b, a := byte(x*255/(n-1)), byte(y*255/(n-1)), byte(255-x*255/(n-1)), byte(255)
			if x%8 == 0 || y%8 == 0 {
				r, g, b = 255, 255, 255
			}
			if x < n/4 && y < n/4 {
				a = 128
			}
			// Premultiplied.
			f := float64(a) / 255
			copy(pix[4*(y*n+x):], []byte{byte(float64(r) * f), byte(float64(g) * f), byte(float64(b) * f), a})
		}
	}
	return ui.NewImageFromRGBA(pix, n, n)
}

// StarPath is a five-pointed star in a 128 x 128 box.
func StarPath() *ui.Path {
	p := ui.NewPath()
	for i := 0; i < 5; i++ {
		a := -math.Pi/2 + float64(i)*4*math.Pi/5
		x, y := 64+50*math.Cos(a), 64+50*math.Sin(a)
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return p
}

// EvenStops are n color stops evenly spaced from 0 to 1.
func EvenStops(n int) []float64 {
	stops := make([]float64, n)
	for i := range stops {
		stops[i] = float64(i) / float64(n-1)
	}
	return stops
}

// Scale4 is a uniform scale of the x and y axes.
func Scale4(s float32) *ui.Matrix4 { return ui.NewMatrix4Diagonal3Values(s, s, 1) }

// EffectScenes returns the drawings of gradients, image shaders, color filters, blend modes,
// vertices and layers that both GPU renderers are compared with the software renderer on.
func EffectScenes() []EffectScene {
	img := Pattern(32)
	cols := func(argbs ...uint32) []ui.Color {
		out := make([]ui.Color, len(argbs))
		for i, a := range argbs {
			out[i] = ui.NewColor(a)
		}
		return out
	}
	scale := func(sx, sy float64) *ui.Matrix4 { return ui.NewMatrix4Diagonal3Values(float32(sx), float32(sy), 1) }
	canvas := func(fn func(c *ui.Canvas)) func() *ui.Scene {
		return func() *ui.Scene { return CanvasScene(fn) }
	}
	return []EffectScene{
		{"gradient tile modes on every kind", Tolerance{4, 1, 0}, canvas(func(c *ui.Canvas) {
			for i, tm := range []ui.TileMode{ui.TileModeClamp, ui.TileModeRepeated, ui.TileModeMirror, ui.TileModeDecal} {
				x, y := float64(i%2)*64, float64(i/2)*64
				c.Save()
				c.Translate(x, y)
				stops := []float64{0.2, 0.5, 0.8}
				cs := cols(0xFFFF0000, 0x8000FF00, 0xFF0000FF)
				c.DrawRect(ui.NewRectFromLTWH(2, 2, 28, 28), ShaderPaint(ui.NewGradientLinear(ui.Offset{Dx: 8, Dy: 0}, ui.Offset{Dx: 24, Dy: 10}, cs, ui.GradientLinearOpts{ColorStops: stops, TileMode: tm})))
				c.DrawRect(ui.NewRectFromLTWH(32, 2, 28, 28), ShaderPaint(ui.NewGradientRadial(ui.Offset{Dx: 46, Dy: 16}, 9, cs, ui.GradientRadialOpts{ColorStops: stops, TileMode: tm})))
				focal := ui.Offset{Dx: 40, Dy: 46}
				c.DrawRect(ui.NewRectFromLTWH(2, 32, 28, 28), ShaderPaint(ui.NewGradientRadial(ui.Offset{Dx: 16, Dy: 46}, 12, cs, ui.GradientRadialOpts{TileMode: tm, Focal: &focal, FocalRadius: 2, ColorStops: stops})))
				c.DrawRect(ui.NewRectFromLTWH(32, 32, 28, 28), ShaderPaint(ui.NewGradientSweep(ui.Offset{Dx: 46, Dy: 46}, cs, ui.GradientSweepOpts{ColorStops: stops, TileMode: tm, StartAngle: 0.5, EndAngle: 4})))
				c.Restore()
			}
		})},
		{"gradient with a matrix, alpha, a degenerate one and many stops", Tolerance{4, 1, 0}, canvas(func(c *ui.Canvas) {
			rot := ui.NewMatrix4Identity()
			rot.RotateZ(0.6)
			c.DrawRect(ui.NewRectFromLTWH(4, 4, 56, 56), ShaderPaint(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 30, Dy: 0}, cols(0xFF202020, 0xFFF0F000), ui.GradientLinearOpts{TileMode: ui.TileModeMirror, Matrix4: rot})))
			faded := ShaderPaint(ui.NewGradientLinear(ui.Offset{Dx: 70, Dy: 0}, ui.Offset{Dx: 120, Dy: 0}, cols(0xFFFF00FF, 0xFF00FFFF)))
			faded.SetColor(ui.NewColor(0x80000000))
			c.DrawRect(ui.NewRectFromLTWH(68, 4, 56, 56), faded)
			c.DrawRect(ui.NewRectFromLTWH(4, 68, 28, 28), ShaderPaint(ui.NewGradientLinear(ui.Offset{Dx: 5, Dy: 5}, ui.Offset{Dx: 5, Dy: 5}, cols(0xFFFF0000, 0xFF0000FF))))
			dec := ui.NewGradientLinear(ui.Offset{Dx: 5, Dy: 5}, ui.Offset{Dx: 5, Dy: 5}, cols(0xFFFF0000, 0xFF0000FF), ui.GradientLinearOpts{TileMode: ui.TileModeDecal})
			c.DrawRect(ui.NewRectFromLTWH(36, 68, 28, 28), ShaderPaint(dec))
			many := make([]ui.Color, 30)
			for i := range many {
				many[i] = ui.NewColor(0xFF000000 | uint32(i*8)<<16 | uint32(255-i*8)<<8 | uint32(i%2)*255)
			}
			c.DrawRect(ui.NewRectFromLTWH(68, 68, 56, 56), ShaderPaint(ui.NewGradientLinear(ui.Offset{Dx: 68, Dy: 0}, ui.Offset{Dx: 124, Dy: 0}, many, ui.GradientLinearOpts{ColorStops: EvenStops(len(many))})))
			c.DrawPath(StarPath().Shift(ui.Offset{Dx: -20, Dy: 20}), ShaderPaint(ui.NewGradientRadial(ui.Offset{Dx: 44, Dy: 84}, 30, cols(0xFFFFFFFF, 0xFF2040A0))))
		})},
		{"image shaders: tile modes, qualities and a matrix", Tolerance{6, 2, 0}, canvas(func(c *ui.Canvas) {
			for i, tm := range []ui.TileMode{ui.TileModeClamp, ui.TileModeRepeated, ui.TileModeMirror, ui.TileModeDecal} {
				m := ui.NewMatrix4TranslationValues(float32(float64(i%2)*64+10), float32(float64(i/2)*64+10), 0)
				sh := ui.NewImageShader(img, tm, tm, m, ui.ImageShaderOpts{FilterQuality: ui.FilterQualityNone})
				c.DrawRect(ui.NewRectFromLTWH(float64(i%2)*64+2, float64(i/2)*64+2, 60, 60), ShaderPaint(sh))
			}
			c.Save()
			c.Translate(0, 0)
			big := ui.NewMatrix4Diagonal3Values(float32(0.3), float32(0.3), 1)
			c.DrawRect(ui.NewRectFromLTWH(8, 8, 40, 40), ShaderPaint(ui.NewImageShader(img, ui.TileModeRepeated, ui.TileModeMirror, big, ui.ImageShaderOpts{FilterQuality: ui.FilterQualityHigh})))
			c.DrawRect(ui.NewRectFromLTWH(72, 72, 40, 40), ShaderPaint(ui.NewImageShader(img, ui.TileModeRepeated, ui.TileModeRepeated, scale(1.7, 1.7), ui.ImageShaderOpts{FilterQuality: ui.FilterQualityMedium})))
			c.Restore()
		})},
		{"color filters and inverted colors", Tolerance{4, 1, 0}, canvas(func(c *ui.Canvas) {
			g := ui.NewGradientLinear(ui.Offset{Dx: 0, Dy: 0}, ui.Offset{Dx: 120, Dy: 0}, cols(0xFFFF4020, 0xCC2080FF, 0x8040E060), ui.GradientLinearOpts{ColorStops: EvenStops(3)})
			filters := []*ui.ColorFilter{
				ui.NewColorFilterMode(ui.NewColor(0x8000A000), ui.BlendModeSrcOver),
				ui.NewColorFilterMode(ui.NewColor(0xFFFFFF00), ui.BlendModeMultiply),
				ui.NewColorFilterMatrix([20]float64{0.3, 0.6, 0.1, 0, 0, 0.3, 0.6, 0.1, 0, 0, 0.3, 0.6, 0.1, 0, 0, 0, 0, 0, 1, 0}),
				ui.NewColorFilterLinearToSrgbGamma(),
				ui.NewColorFilterSrgbToLinearGamma(),
				ui.NewColorFilterSaturation(0.2),
			}
			for i, f := range filters {
				p := ShaderPaint(g)
				p.SetColorFilter(f)
				c.DrawRect(ui.NewRectFromLTWH(4, float64(i)*20+4, 60, 16), p)
				q := FillPaint(0xFFD08030)
				q.SetColorFilter(f)
				c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(68, float64(i)*20+4, 56, 16), 6, 6), q)
			}
			inv := FillPaint(0xFF3080C0)
			inv.SetInvertColors(true)
			c.DrawCircle(ui.Offset{Dx: 100, Dy: 120}, 6, inv)
			// Nearest-neighbor sampling at a pixel center that is exactly on the edge of
			// a texel picks either by rounding, and GPUs round differently, so the
			// source is an odd number of texels wide: with an even destination
			// size no pixel center can fall on an edge, and with an odd one
			// (13 here) the center of the middle pixel is the center of a texel.
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 31, 31), ui.NewRectFromLTWH(4, 112, 13, 13), func() *ui.Paint {
				p := ui.NewPaint()
				p.SetFilterQuality(ui.FilterQualityNone)
				p.SetInvertColors(true)
				p.SetColorFilter(filters[2])
				return p
			}())
		})},
		{"blend modes on shapes, paths, images and vertices", Tolerance{6, 2, 0}, canvas(func(c *ui.Canvas) {
			StripesAndDisc(c)
			modes := []ui.BlendMode{ui.BlendModeMultiply, ui.BlendModeScreen, ui.BlendModeDifference, ui.BlendModeSrc, ui.BlendModeClear, ui.BlendModeHue, ui.BlendModeSrcATop, ui.BlendModeXor}
			for i, m := range modes {
				x, y := float64(i%4)*32+2, float64(i/4)*64+4
				p := FillPaint(0xB0E04080)
				p.SetBlendMode(m)
				switch i % 4 {
				case 0:
					c.DrawRect(ui.NewRectFromLTWH(x, y, 28, 28), p)
				case 1:
					c.DrawPath(StarPath().Shift(ui.Offset{Dx: x - 20, Dy: y - 10}).Transform(Scale4(0.5)), p)
				case 2:
					ip := ui.NewPaint()
					ip.SetBlendMode(m)
					c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 32, 32), ui.NewRectFromLTWH(x, y, 28, 28), ip)
				default:
					pos := []ui.Offset{{Dx: x, Dy: y}, {Dx: x + 28, Dy: y + 4}, {Dx: x + 10, Dy: y + 28}}
					c.DrawVertices(ui.NewVertices(ui.VertexModeTriangles, pos, ui.VerticesOpts{Colors: cols(0xFFFF0000, 0xFF00FF00, 0xFF0000FF)}), ui.BlendModeSrc, p)
				}
			}
			c.DrawColor(ui.NewColor(0x6080FF80), ui.BlendModeColorBurn)
		})},
		{"vertices with a shader, texture coordinates and colors", Tolerance{4, 1, 0}, canvas(func(c *ui.Canvas) {
			pos := []ui.Offset{{Dx: 5, Dy: 5}, {Dx: 60, Dy: 10}, {Dx: 10, Dy: 60}, {Dx: 65, Dy: 62}}
			tex := []ui.Offset{{Dx: 0, Dy: 0}, {Dx: 32, Dy: 0}, {Dx: 0, Dy: 32}, {Dx: 32, Dy: 32}}
			sh := ShaderPaint(ui.NewImageShader(img, ui.TileModeClamp, ui.TileModeClamp, nil, ui.ImageShaderOpts{FilterQuality: ui.FilterQualityLow}))
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangleStrip, pos, ui.VerticesOpts{TextureCoordinates: tex}), ui.BlendModeSrc, sh)
			c.Save()
			c.Translate(64, 0)
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangleStrip, pos, ui.VerticesOpts{TextureCoordinates: tex, Colors: cols(0xFFFFFFFF, 0xFFFF8080, 0xFF80FF80, 0x80FFFFFF)}), ui.BlendModeModulate, sh)
			c.Restore()
			c.Save()
			c.Translate(0, 64)
			g := ShaderPaint(ui.NewGradientLinear(ui.Offset{Dx: 0, Dy: 0}, ui.Offset{Dx: 64, Dy: 0}, cols(0xFFFF0000, 0xFF0000FF)))
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangleStrip, pos), ui.BlendModeSrc, g)
			c.Restore()
		})},
		{"save layers: alpha, bounds, filters, blend modes, nesting", Tolerance{6, 2, 0}, canvas(func(c *ui.Canvas) {
			StripesAndDisc(c)
			alpha := FillPaint(0x80FFFFFF)
			c.SaveLayer(&ui.Rect{Left: 4, Top: 4, Right: 60, Bottom: 60}, alpha)
			c.DrawCircle(ui.Offset{Dx: 30, Dy: 30}, 24, FillPaint(0xFFFF0000))
			c.DrawCircle(ui.Offset{Dx: 44, Dy: 30}, 24, FillPaint(0xFF0000FF))
			c.Restore()
			f := ui.NewPaint()
			f.SetColorFilter(ui.NewColorFilterMatrix([20]float64{0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0}))
			c.SaveLayer(&ui.Rect{Left: 66, Top: 4, Right: 124, Bottom: 60}, f)
			c.DrawRect(ui.NewRectFromLTWH(70, 10, 40, 40), FillPaint(0xFFFF8000))
			c.Restore()
			b := ui.NewPaint()
			b.SetBlendMode(ui.BlendModeDifference)
			c.SaveLayer(nil, b)
			c.DrawRect(ui.NewRectFromLTWH(4, 70, 56, 50), FillPaint(0xFFFFFFFF))
			c.Save()
			c.Translate(40, 80)
			c.Rotate(0.5)
			c.SaveLayer(nil, FillPaint(0xA0FFFFFF))
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 40, 30), FillPaint(0xFF00A0FF))
			c.DrawCircle(ui.Offset{Dx: 20, Dy: 15}, 10, FillPaint(0xFFFF2020))
			c.Restore()
			c.Restore()
			c.Restore()
			inv := ui.NewPaint()
			inv.SetInvertColors(true)
			c.SaveLayer(&ui.Rect{Left: 70, Top: 70, Right: 120, Bottom: 120}, inv)
			c.DrawCircle(ui.Offset{Dx: 95, Dy: 95}, 20, FillPaint(0xFF20C0A0))
			c.Restore()
			// A layer under a clip with edges that are not pixels, and a shaped one.
			c.Save()
			c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(2.5, 2.5, 30, 30), 8, 8))
			c.SaveLayer(nil, FillPaint(0xC0FFFFFF))
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 128, 128), FillPaint(0xFF000000))
			c.Restore()
			c.Restore()
		})},
		{"opacity, color filter and clip layers of a scene", Tolerance{6, 2, 0}, func() *ui.Scene {
			b := ui.NewSceneBuilder()
			b.PushOpacity(150, ui.PushOpacityOpts{Offset: ui.Offset{Dx: 4, Dy: 3}})
			b.AddPicture(ui.Offset{}, PictureOf(func(c *ui.Canvas) {
				c.DrawRect(ui.NewRectFromLTWH(0, 0, 60, 60), FillPaint(0xFFFF0000))
				c.DrawRect(ui.NewRectFromLTWH(30, 30, 60, 60), FillPaint(0xFF0000FF))
			}))
			b.Pop()
			b.PushColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFF00FF00), ui.BlendModeScreen))
			b.PushOffset(64, 0)
			b.AddPicture(ui.Offset{}, PictureOf(StripesAndDisc))
			b.Pop()
			b.Pop()
			b.PushOffset(0, 64)
			b.PushClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(4.25, 4.25, 56, 56), 20, 20), ui.PushClipRRectOpts{ClipBehavior: ui.ClipAntiAliasWithSaveLayer})
			b.AddPicture(ui.Offset{}, PictureOf(StripesAndDisc))
			b.Pop()
			b.Pop()
			b.PushOffset(64, 64)
			b.PushClipPath(StarPath().Transform(Scale4(0.5)), ui.PushClipPathOpts{ClipBehavior: ui.ClipAntiAliasWithSaveLayer})
			b.PushOpacity(200)
			b.AddPicture(ui.Offset{}, PictureOf(StripesAndDisc))
			b.Pop()
			b.Pop()
			b.Pop()
			return b.Build()
		}},
		{"a backdrop matrix filter reads outside its clip (a magnifier)", Tolerance{6, 2, 0}, func() *ui.Scene {
			b := ui.NewSceneBuilder()
			b.AddPicture(ui.Offset{}, PictureOf(StripesAndDisc))
			// A lens at (32, 20) to (96, 50) that magnifies by 2 what is 45 pixels below its center.
			lens := ui.NewRectFromLTWH(32, 20, 64, 30)
			const scale = 2.0
			c, f := lens.Center(), ui.Offset{Dx: lens.Center().Dx, Dy: lens.Center().Dy + 45}
			m := ui.NewMatrix4Identity()
			m.Translate(float32(scale*(-f.Dx-c.Dx)+c.Dx), float32(scale*(-f.Dy-c.Dy)+c.Dy), 0)
			m.Scale(scale, scale, scale)
			b.PushClipRRect(ui.NewRRectFromRectXY(lens, 12, 12))
			b.PushBackdropFilter(ui.NewImageFilterMatrix(m, ui.ImageFilterMatrixOpts{FilterQuality: ui.FilterQualityHigh}))
			b.Pop()
			b.Pop()
			// A second lens at the bottom edge, which reads from below the surface: nothing.
			bottom := ui.NewRectFromLTWH(20, 100, 40, 24)
			m2 := ui.NewMatrix4Identity()
			m2.Translate(0, -60, 0)
			b.PushClipRect(bottom)
			b.PushBackdropFilter(ui.NewImageFilterMatrix(m2))
			b.Pop()
			b.Pop()
			return b.Build()
		}},
	}
}
