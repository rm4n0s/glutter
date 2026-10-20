//go:build glutter_display

package gles

import (
	"image"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// The paint shaders, color filters, blend modes and layers of step 3.4, compared
// with the software renderer.

func TestDisplayGradientsMatchSoftware(t *testing.T) {
	gpuMatchesSoft(t, "gradients", func(t *testing.T, got, want *image.RGBA, r *Renderer) {
		st := r.LastFrame()
		t.Logf("frame: %+v", st)
		if st.Skipped != 0 {
			t.Errorf("the GPU skipped %d commands: %v", st.Skipped, st.Unsupported)
		}
		// Gradients are dithered the same way on both, so only the rounding of the
		// interpolation and of the edges differs.
		tolerance{over8Permille: 10, over32Permille: 3}.check(t, "gradients", diffRect(got, want, got.Bounds()))
	})
}

func TestDisplayBlendModesMatchSoftware(t *testing.T) {
	gpuMatchesSoft(t, "blend-modes", func(t *testing.T, got, want *image.RGBA, r *Renderer) {
		st := r.LastFrame()
		t.Logf("frame: %+v", st)
		if st.Skipped != 0 {
			t.Errorf("the GPU skipped %d commands: %v", st.Skipped, st.Unsupported)
		}
		for i, mode := range blendModeNames {
			cell := image.Rect(i%6*106, i/6*96, i%6*106+106, i/6*96+96)
			tolerance{over8Permille: 15, over32Permille: 5}.check(t, mode, diffRect(got, want, cell))
		}
	})
}

// blendModeNames are the 29 modes in the order of BlendMode, as the cells of the
// blend-modes scene are.
var blendModeNames = []string{
	"clear", "src", "dst", "srcOver", "dstOver", "srcIn", "dstIn", "srcOut", "dstOut", "srcATop",
	"dstATop", "xor", "plus", "modulate", "screen", "overlay", "darken", "lighten", "colorDodge", "colorBurn",
	"hardLight", "softLight", "difference", "exclusion", "multiply", "hue", "saturation", "color", "luminosity",
}

// pictureOf records a picture of drawing.
func pictureOf(fn func(c *ui.Canvas)) *ui.Picture {
	rec := ui.NewPictureRecorder()
	fn(ui.NewCanvas(rec))
	return rec.EndRecording()
}

// stripesAndDisc is something with an edge and a gradient to put layers over.
func stripesAndDisc(c *ui.Canvas) {
	for i := 0; i < 8; i++ {
		c.DrawRect(ui.NewRectFromLTWH(0, float64(i)*16, 128, 8), fillP(0xFF3050C0))
		c.DrawRect(ui.NewRectFromLTWH(0, float64(i)*16+8, 128, 8), fillP(0xFFF0A020))
	}
	g := ui.NewPaint()
	g.SetShader(ui.NewGradientRadial(ui.Offset{Dx: 64, Dy: 64}, 40, []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0x4000FF00)}))
	c.DrawCircle(ui.Offset{Dx: 64, Dy: 64}, 40, g)
}

func shaderP(sh ui.Shader) *ui.Paint {
	p := ui.NewPaint()
	p.SetShader(sh)
	return p
}

func TestDisplayPaintsAndLayersMatchSoftware(t *testing.T) {
	img := pattern(32)
	cols := func(argbs ...uint32) []ui.Color {
		out := make([]ui.Color, len(argbs))
		for i, a := range argbs {
			out[i] = ui.NewColor(a)
		}
		return out
	}
	rect := ui.NewRectFromLTWH(8, 8, 112, 112)
	scale := func(sx, sy float64) *ui.Matrix4 { return ui.NewMatrix4Diagonal3Values(float32(sx), float32(sy), 1) }
	canvas := func(fn func(c *ui.Canvas)) func() *ui.Scene {
		return func() *ui.Scene { return canvasScene(128, 128, fn) }
	}
	tests := []struct {
		name  string
		tol   tolerance
		scene func() *ui.Scene
	}{
		{"gradient tile modes on every kind", tolerance{4, 1, 0}, canvas(func(c *ui.Canvas) {
			for i, tm := range []ui.TileMode{ui.TileModeClamp, ui.TileModeRepeated, ui.TileModeMirror, ui.TileModeDecal} {
				x, y := float64(i%2)*64, float64(i/2)*64
				c.Save()
				c.Translate(x, y)
				stops := []float64{0.2, 0.5, 0.8}
				cs := cols(0xFFFF0000, 0x8000FF00, 0xFF0000FF)
				c.DrawRect(ui.NewRectFromLTWH(2, 2, 28, 28), shaderP(ui.NewGradientLinear(ui.Offset{Dx: 8, Dy: 0}, ui.Offset{Dx: 24, Dy: 10}, cs, ui.GradientLinearOpts{ColorStops: stops, TileMode: tm})))
				c.DrawRect(ui.NewRectFromLTWH(32, 2, 28, 28), shaderP(ui.NewGradientRadial(ui.Offset{Dx: 46, Dy: 16}, 9, cs, ui.GradientRadialOpts{ColorStops: stops, TileMode: tm})))
				focal := ui.Offset{Dx: 40, Dy: 46}
				c.DrawRect(ui.NewRectFromLTWH(2, 32, 28, 28), shaderP(ui.NewGradientRadial(ui.Offset{Dx: 16, Dy: 46}, 12, cs, ui.GradientRadialOpts{TileMode: tm, Focal: &focal, FocalRadius: 2, ColorStops: stops})))
				c.DrawRect(ui.NewRectFromLTWH(32, 32, 28, 28), shaderP(ui.NewGradientSweep(ui.Offset{Dx: 46, Dy: 46}, cs, ui.GradientSweepOpts{ColorStops: stops, TileMode: tm, StartAngle: 0.5, EndAngle: 4})))
				c.Restore()
			}
		})},
		{"gradient with a matrix, alpha, a degenerate one and many stops", tolerance{4, 1, 0}, canvas(func(c *ui.Canvas) {
			rot := ui.NewMatrix4Identity()
			rot.RotateZ(0.6)
			c.DrawRect(ui.NewRectFromLTWH(4, 4, 56, 56), shaderP(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 30, Dy: 0}, cols(0xFF202020, 0xFFF0F000), ui.GradientLinearOpts{TileMode: ui.TileModeMirror, Matrix4: rot})))
			faded := shaderP(ui.NewGradientLinear(ui.Offset{Dx: 70, Dy: 0}, ui.Offset{Dx: 120, Dy: 0}, cols(0xFFFF00FF, 0xFF00FFFF)))
			faded.SetColor(ui.NewColor(0x80000000))
			c.DrawRect(ui.NewRectFromLTWH(68, 4, 56, 56), faded)
			c.DrawRect(ui.NewRectFromLTWH(4, 68, 28, 28), shaderP(ui.NewGradientLinear(ui.Offset{Dx: 5, Dy: 5}, ui.Offset{Dx: 5, Dy: 5}, cols(0xFFFF0000, 0xFF0000FF))))
			dec := ui.NewGradientLinear(ui.Offset{Dx: 5, Dy: 5}, ui.Offset{Dx: 5, Dy: 5}, cols(0xFFFF0000, 0xFF0000FF), ui.GradientLinearOpts{TileMode: ui.TileModeDecal})
			c.DrawRect(ui.NewRectFromLTWH(36, 68, 28, 28), shaderP(dec))
			many := make([]ui.Color, 30)
			for i := range many {
				many[i] = ui.NewColor(0xFF000000 | uint32(i*8)<<16 | uint32(255-i*8)<<8 | uint32(i%2)*255)
			}
			c.DrawRect(ui.NewRectFromLTWH(68, 68, 56, 56), shaderP(ui.NewGradientLinear(ui.Offset{Dx: 68, Dy: 0}, ui.Offset{Dx: 124, Dy: 0}, many, ui.GradientLinearOpts{ColorStops: evenStops(len(many))})))
			c.DrawPath(starPath().Shift(ui.Offset{Dx: -20, Dy: 20}), shaderP(ui.NewGradientRadial(ui.Offset{Dx: 44, Dy: 84}, 30, cols(0xFFFFFFFF, 0xFF2040A0))))
		})},
		{"image shaders: tile modes, qualities and a matrix", tolerance{6, 2, 0}, canvas(func(c *ui.Canvas) {
			for i, tm := range []ui.TileMode{ui.TileModeClamp, ui.TileModeRepeated, ui.TileModeMirror, ui.TileModeDecal} {
				m := ui.NewMatrix4TranslationValues(float64To32(float64(i%2)*64+10), float64To32(float64(i/2)*64+10), 0)
				sh := ui.NewImageShader(img, tm, tm, m, ui.ImageShaderOpts{FilterQuality: ui.FilterQualityNone})
				c.DrawRect(ui.NewRectFromLTWH(float64(i%2)*64+2, float64(i/2)*64+2, 60, 60), shaderP(sh))
			}
			c.Save()
			c.Translate(0, 0)
			big := ui.NewMatrix4Diagonal3Values(float32(0.3), float32(0.3), 1)
			c.DrawRect(ui.NewRectFromLTWH(8, 8, 40, 40), shaderP(ui.NewImageShader(img, ui.TileModeRepeated, ui.TileModeMirror, big, ui.ImageShaderOpts{FilterQuality: ui.FilterQualityHigh})))
			c.DrawRect(ui.NewRectFromLTWH(72, 72, 40, 40), shaderP(ui.NewImageShader(img, ui.TileModeRepeated, ui.TileModeRepeated, scale(1.7, 1.7), ui.ImageShaderOpts{FilterQuality: ui.FilterQualityMedium})))
			c.Restore()
		})},
		{"color filters and inverted colors", tolerance{4, 1, 0}, canvas(func(c *ui.Canvas) {
			g := ui.NewGradientLinear(ui.Offset{Dx: 0, Dy: 0}, ui.Offset{Dx: 120, Dy: 0}, cols(0xFFFF4020, 0xCC2080FF, 0x8040E060), ui.GradientLinearOpts{ColorStops: evenStops(3)})
			filters := []*ui.ColorFilter{
				ui.NewColorFilterMode(ui.NewColor(0x8000A000), ui.BlendModeSrcOver),
				ui.NewColorFilterMode(ui.NewColor(0xFFFFFF00), ui.BlendModeMultiply),
				ui.NewColorFilterMatrix([20]float64{0.3, 0.6, 0.1, 0, 0, 0.3, 0.6, 0.1, 0, 0, 0.3, 0.6, 0.1, 0, 0, 0, 0, 0, 1, 0}),
				ui.NewColorFilterLinearToSrgbGamma(),
				ui.NewColorFilterSrgbToLinearGamma(),
				ui.NewColorFilterSaturation(0.2),
			}
			for i, f := range filters {
				p := shaderP(g)
				p.SetColorFilter(f)
				c.DrawRect(ui.NewRectFromLTWH(4, float64(i)*20+4, 60, 16), p)
				q := fillP(0xFFD08030)
				q.SetColorFilter(f)
				c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(68, float64(i)*20+4, 56, 16), 6, 6), q)
			}
			inv := fillP(0xFF3080C0)
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
		{"blend modes on shapes, paths, images and vertices", tolerance{6, 2, 0}, canvas(func(c *ui.Canvas) {
			stripesAndDisc(c)
			modes := []ui.BlendMode{ui.BlendModeMultiply, ui.BlendModeScreen, ui.BlendModeDifference, ui.BlendModeSrc, ui.BlendModeClear, ui.BlendModeHue, ui.BlendModeSrcATop, ui.BlendModeXor}
			for i, m := range modes {
				x, y := float64(i%4)*32+2, float64(i/4)*64+4
				p := fillP(0xB0E04080)
				p.SetBlendMode(m)
				switch i % 4 {
				case 0:
					c.DrawRect(ui.NewRectFromLTWH(x, y, 28, 28), p)
				case 1:
					c.DrawPath(starPath().Shift(ui.Offset{Dx: x - 20, Dy: y - 10}).Transform(scale4(0.5)), p)
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
		{"vertices with a shader, texture coordinates and colors", tolerance{4, 1, 0}, canvas(func(c *ui.Canvas) {
			pos := []ui.Offset{{Dx: 5, Dy: 5}, {Dx: 60, Dy: 10}, {Dx: 10, Dy: 60}, {Dx: 65, Dy: 62}}
			tex := []ui.Offset{{Dx: 0, Dy: 0}, {Dx: 32, Dy: 0}, {Dx: 0, Dy: 32}, {Dx: 32, Dy: 32}}
			sh := shaderP(ui.NewImageShader(img, ui.TileModeClamp, ui.TileModeClamp, nil, ui.ImageShaderOpts{FilterQuality: ui.FilterQualityLow}))
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangleStrip, pos, ui.VerticesOpts{TextureCoordinates: tex}), ui.BlendModeSrc, sh)
			c.Save()
			c.Translate(64, 0)
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangleStrip, pos, ui.VerticesOpts{TextureCoordinates: tex, Colors: cols(0xFFFFFFFF, 0xFFFF8080, 0xFF80FF80, 0x80FFFFFF)}), ui.BlendModeModulate, sh)
			c.Restore()
			c.Save()
			c.Translate(0, 64)
			g := shaderP(ui.NewGradientLinear(ui.Offset{Dx: 0, Dy: 0}, ui.Offset{Dx: 64, Dy: 0}, cols(0xFFFF0000, 0xFF0000FF)))
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangleStrip, pos), ui.BlendModeSrc, g)
			c.Restore()
		})},
		{"save layers: alpha, bounds, filters, blend modes, nesting", tolerance{6, 2, 0}, canvas(func(c *ui.Canvas) {
			stripesAndDisc(c)
			alpha := fillP(0x80FFFFFF)
			c.SaveLayer(&ui.Rect{Left: 4, Top: 4, Right: 60, Bottom: 60}, alpha)
			c.DrawCircle(ui.Offset{Dx: 30, Dy: 30}, 24, fillP(0xFFFF0000))
			c.DrawCircle(ui.Offset{Dx: 44, Dy: 30}, 24, fillP(0xFF0000FF))
			c.Restore()
			f := ui.NewPaint()
			f.SetColorFilter(ui.NewColorFilterMatrix([20]float64{0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0}))
			c.SaveLayer(&ui.Rect{Left: 66, Top: 4, Right: 124, Bottom: 60}, f)
			c.DrawRect(ui.NewRectFromLTWH(70, 10, 40, 40), fillP(0xFFFF8000))
			c.Restore()
			b := ui.NewPaint()
			b.SetBlendMode(ui.BlendModeDifference)
			c.SaveLayer(nil, b)
			c.DrawRect(ui.NewRectFromLTWH(4, 70, 56, 50), fillP(0xFFFFFFFF))
			c.Save()
			c.Translate(40, 80)
			c.Rotate(0.5)
			c.SaveLayer(nil, fillP(0xA0FFFFFF))
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 40, 30), fillP(0xFF00A0FF))
			c.DrawCircle(ui.Offset{Dx: 20, Dy: 15}, 10, fillP(0xFFFF2020))
			c.Restore()
			c.Restore()
			c.Restore()
			inv := ui.NewPaint()
			inv.SetInvertColors(true)
			c.SaveLayer(&ui.Rect{Left: 70, Top: 70, Right: 120, Bottom: 120}, inv)
			c.DrawCircle(ui.Offset{Dx: 95, Dy: 95}, 20, fillP(0xFF20C0A0))
			c.Restore()
			// A layer under a clip with edges that are not pixels, and a shaped one.
			c.Save()
			c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(2.5, 2.5, 30, 30), 8, 8))
			c.SaveLayer(nil, fillP(0xC0FFFFFF))
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 128, 128), fillP(0xFF000000))
			c.Restore()
			c.Restore()
		})},
		{"opacity, color filter and clip layers of a scene", tolerance{6, 2, 0}, func() *ui.Scene {
			b := ui.NewSceneBuilder()
			b.PushOpacity(150, ui.PushOpacityOpts{Offset: ui.Offset{Dx: 4, Dy: 3}})
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) {
				c.DrawRect(ui.NewRectFromLTWH(0, 0, 60, 60), fillP(0xFFFF0000))
				c.DrawRect(ui.NewRectFromLTWH(30, 30, 60, 60), fillP(0xFF0000FF))
			}))
			b.Pop()
			b.PushColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFF00FF00), ui.BlendModeScreen))
			b.PushOffset(64, 0)
			b.AddPicture(ui.Offset{}, pictureOf(stripesAndDisc))
			b.Pop()
			b.Pop()
			b.PushOffset(0, 64)
			b.PushClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(4.25, 4.25, 56, 56), 20, 20), ui.PushClipRRectOpts{ClipBehavior: ui.ClipAntiAliasWithSaveLayer})
			b.AddPicture(ui.Offset{}, pictureOf(stripesAndDisc))
			b.Pop()
			b.Pop()
			b.PushOffset(64, 64)
			b.PushClipPath(starPath().Transform(scale4(0.5)), ui.PushClipPathOpts{ClipBehavior: ui.ClipAntiAliasWithSaveLayer})
			b.PushOpacity(200)
			b.AddPicture(ui.Offset{}, pictureOf(stripesAndDisc))
			b.Pop()
			b.Pop()
			b.Pop()
			return b.Build()
		}},
	}
	_ = rect
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := tt.scene()
			want, err := soft.Render(scene, 128, 128)
			if err != nil {
				t.Fatal(err)
			}
			onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
				r := NewRenderer()
				defer func() { _ = r.Close() }()
				if _, err := r.Attach(h); err != nil {
					return err
				}
				got, err := renderGPU(r, scene, 128, 128)
				if err != nil {
					return err
				}
				slug := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(tt.name, " ", "-"), ",", ""), ":", "")
				dumpImage(t, slug+"-gpu", got)
				dumpImage(t, slug+"-soft", want)
				tt.tol.check(t, tt.name, diffRect(got, want, got.Bounds()))
				return nil
			})
		})
	}
}

func evenStops(n int) []float64 {
	stops := make([]float64, n)
	for i := range stops {
		stops[i] = float64(i) / float64(n-1)
	}
	return stops
}

func float64To32(v float64) float32 { return float32(v) }

func scale4(s float32) *ui.Matrix4 { return ui.NewMatrix4Diagonal3Values(float32(s), float32(s), 1) }
