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

// The blurs, shadows, backdrop filters and shader masks of step 3.5, compared with
// the software renderer.

func TestDisplayEffectsMatchSoftware(t *testing.T) {
	gpuMatchesSoft(t, "effects", func(t *testing.T, got, want *image.RGBA, r *Renderer) {
		st := r.LastFrame()
		t.Logf("frame: %+v", st)
		if st.Skipped != 0 {
			t.Errorf("the GPU skipped %d commands: %v", st.Skipped, st.Unsupported)
		}
		for _, tt := range []struct {
			name string
			rect image.Rectangle
			tol  tolerance
		}{
			{"card shadows", image.Rect(0, 0, 640, 135), tolerance{}},
			{"star and circle shadows", image.Rect(0, 135, 300, 225), tolerance{}},
			{"mask filter styles", image.Rect(320, 135, 640, 225), tolerance{}},
			{"frosted glass", image.Rect(0, 225, 320, 480), tolerance{over8Permille: 1}},
			{"shader mask", image.Rect(320, 225, 640, 480), tolerance{over8Permille: 1}},
		} {
			tt.tol.check(t, tt.name, diffRect(got, want, tt.rect))
		}
	})
}

// compareScene draws a scene with the GPU and the software renderer and checks the
// two within the tolerance.
func compareScene(t *testing.T, name string, scene *ui.Scene, w, h int, tol tolerance) {
	t.Helper()
	want, err := soft.Render(scene, w, h)
	if err != nil {
		t.Fatal(err)
	}
	onDisplay(t, func(handle gpu.SurfaceHandle, _, _ int) error {
		r := NewRenderer()
		defer func() { _ = r.Close() }()
		if _, err := r.Attach(handle); err != nil {
			return err
		}
		got, err := renderGPU(r, scene, w, h)
		if err != nil {
			return err
		}
		slug := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(name, " ", "-"), ",", ""), ":", "")
		dumpImage(t, slug+"-gpu", got)
		dumpImage(t, slug+"-soft", want)
		if st := r.LastFrame(); st.Skipped != 0 {
			t.Errorf("the GPU skipped %d commands: %v", st.Skipped, st.Unsupported)
		}
		tol.check(t, name, diffRect(got, want, got.Bounds()))
		return nil
	})
}

func TestDisplayEffectScenesMatchSoftware(t *testing.T) {
	rect := func(l, t, w, h float64) ui.Rect { return ui.NewRectFromLTWH(l, t, w, h) }
	// layered puts the drawing of fn inside a layer that the build function pushes.
	layered := func(w, h int, push func(b *ui.SceneBuilder), fn func(c *ui.Canvas)) func() *ui.Scene {
		return func() *ui.Scene {
			b := ui.NewSceneBuilder()
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) { c.DrawColor(ui.NewColor(0xFFEEEEEE), ui.BlendModeSrcOver) }))
			push(b)
			b.AddPicture(ui.Offset{}, pictureOf(fn))
			b.Pop()
			return b.Build()
		}
	}
	canvas := func(w, h int, fn func(c *ui.Canvas)) func() *ui.Scene {
		return func() *ui.Scene { return canvasScene(w, h, fn) }
	}
	blur := func(sx, sy float64, tm ui.TileMode) ui.ImageFilter {
		return ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: sx, SigmaY: sy, TileMode: tm})
	}
	shapes := func(c *ui.Canvas) {
		c.DrawRRect(ui.NewRRectFromRectXY(rect(10, 10, 50, 40), 10, 10), fillP(0xFFD03030))
		c.DrawCircle(ui.Offset{Dx: 90, Dy: 40}, 22, fillP(0xFF2060D0))
		c.DrawPath(starPath().Shift(ui.Offset{Dx: 0, Dy: 40}), fillP(0xFF30A040))
	}
	tests := []struct {
		name  string
		w, h  int
		tol   tolerance
		scene func() *ui.Scene
	}{
		{"image filter layer blurs with each tile mode", 192, 128, tolerance{}, func() *ui.Scene {
			b := ui.NewSceneBuilder()
			for i, tm := range []ui.TileMode{ui.TileModeUnspecified, ui.TileModeClamp, ui.TileModeRepeated, ui.TileModeMirror} {
				b.PushOffset(float64(i%2)*96, float64(i/2)*64)
				b.PushClipRect(rect(0, 0, 96, 64))
				b.PushImageFilter(blur(5, 3, tm))
				b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) {
					c.DrawRect(rect(10, 10, 30, 30), fillP(0xFFE04020))
					c.DrawCircle(ui.Offset{Dx: 70, Dy: 40}, 18, fillP(0xFF2040E0))
				}))
				b.Pop()
				b.Pop()
				b.Pop()
			}
			return b.Build()
		}},
		{"large blurs are made on a smaller picture", 160, 128, tolerance{}, layered(160, 128, func(b *ui.SceneBuilder) {
			b.PushImageFilter(blur(24, 6, ui.TileModeDecal))
		}, func(c *ui.Canvas) {
			c.DrawRRect(ui.NewRRectFromRectXY(rect(30, 30, 100, 60), 14, 14), fillP(0xFF8030C0))
			c.DrawRect(rect(0, 100, 160, 8), fillP(0xFF20A0A0))
		})},
		{"dilate, erode, matrix, compose and color image filters", 192, 128, tolerance{over8Permille: 5}, func() *ui.Scene {
			rot := ui.NewMatrix4Identity()
			rot.RotateZ(0.4)
			rot.Scale(1.2, 0.9, 1)
			filters := []ui.ImageFilter{
				ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 3, RadiusY: 1}),
				ui.NewImageFilterErode(ui.ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 2}),
				ui.NewImageFilterMatrix(rot, ui.ImageFilterMatrixOpts{FilterQuality: ui.FilterQualityLow}),
				ui.NewImageFilterCompose(blur(2, 2, ui.TileModeClamp), ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 2})),
				ui.NewColorFilterMatrix([20]float64{0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0}),
				ui.NewImageFilterMatrix(rot, ui.ImageFilterMatrixOpts{FilterQuality: ui.FilterQualityNone}),
			}
			b := ui.NewSceneBuilder()
			for i, f := range filters {
				b.PushOffset(float64(i%3)*64, float64(i/3)*64)
				b.PushClipRect(rect(0, 0, 64, 64))
				b.PushImageFilter(f)
				b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) {
					c.DrawRRect(ui.NewRRectFromRectXY(rect(12, 14, 36, 30), 8, 8), fillP(0xFFE08020))
					c.DrawRect(rect(20, 22, 6, 6), fillP(0xFF204080))
				}))
				b.Pop()
				b.Pop()
				b.Pop()
			}
			return b.Build()
		}},
		{"save layer and paint image filters", 160, 128, tolerance{}, canvas(160, 128, func(c *ui.Canvas) {
			c.DrawColor(ui.NewColor(0xFFF4F0E8), ui.BlendModeSrcOver)
			p := ui.NewPaint()
			p.SetImageFilter(blur(4, 4, ui.TileModeUnspecified))
			c.SaveLayer(&ui.Rect{Left: 4, Top: 4, Right: 76, Bottom: 60}, p)
			shapes(c)
			c.Restore()
			// A paint image filter: a blurred shape; its blend mode is applied to the result.
			q := fillP(0xFF3050D0)
			q.SetImageFilter(blur(3, 3, ui.TileModeUnspecified))
			c.DrawRRect(ui.NewRRectFromRectXY(rect(90, 10, 50, 40), 8, 8), q)
			r := fillP(0xC0D03020)
			r.SetImageFilter(ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 2}))
			r.SetBlendMode(ui.BlendModeMultiply)
			c.DrawCircle(ui.Offset{Dx: 40, Dy: 90}, 20, r)
			s := fillP(0xFF30A050)
			s.SetImageFilter(blur(2, 8, ui.TileModeUnspecified))
			c.Save()
			c.Rotate(0.3)
			c.DrawRect(rect(100, 40, 40, 20), s)
			c.Restore()
		})},
		// One pixel is 9 levels off on the GPU that kwin's virtual output uses.
		{"mask filters: styles, strokes, paths and transforms", 192, 128, tolerance{over8Permille: 1}, canvas(192, 128, func(c *ui.Canvas) {
			c.DrawColor(ui.NewColor(0xFFF4F0E8), ui.BlendModeSrcOver)
			styles := []ui.BlurStyle{ui.BlurStyleNormal, ui.BlurStyleSolid, ui.BlurStyleOuter, ui.BlurStyleInner}
			for i, st := range styles {
				// A star goes through the layer path, with a gradient for a paint.
				p := ui.NewPaint()
				p.SetShader(ui.NewGradientLinear(ui.Offset{Dx: float64(i) * 48, Dy: 0}, ui.Offset{Dx: float64(i)*48 + 40, Dy: 40}, []ui.Color{ui.NewColor(0xFFE03020), ui.NewColor(0xFF2040E0)}))
				p.SetMaskFilter(ui.NewMaskFilterBlur(st, 4))
				c.DrawPath(starPath().Shift(ui.Offset{Dx: float64(i)*48 - 30, Dy: -22}).Transform(scale4(0.8)), p)
			}
			s := strokeP(0xFF206040, 6, ui.StrokeCapButt, ui.StrokeJoinMiter)
			s.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 3))
			c.DrawRRect(ui.NewRRectFromRectXY(rect(10, 50, 50, 30), 8, 8), s)
			c.DrawLine(ui.Offset{Dx: 70, Dy: 52}, ui.Offset{Dx: 110, Dy: 78}, s)
			// A rotated rounded rectangle, a flipped and scaled one, a circle and a rect.
			c.Save()
			c.Translate(150, 65)
			c.Rotate(0.5)
			rr := fillP(0xFF8040C0)
			rr.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 5))
			c.DrawRRect(ui.NewRRectFromRectXY(rect(-20, -14, 40, 28), 6, 6), rr)
			c.Restore()
			c.Save()
			c.Translate(40, 120)
			c.Scale(-1.5, -0.8)
			fl := fillP(0xFFC08020)
			fl.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleSolid, 3))
			c.DrawRRect(ui.NewRRectFromRectAndCorners(rect(2, 10, 30, 14), ui.Corners{TopLeft: ui.Radius{X: 12, Y: 8}, BottomRight: ui.Radius{X: 3, Y: 6}}), fl)
			c.Restore()
			ci := fillP(0xFF2080A0)
			ci.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleInner, 6))
			c.DrawCircle(ui.Offset{Dx: 90, Dy: 105}, 16, ci)
			re := fillP(0xFF30A060)
			re.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 8))
			c.DrawRect(rect(120, 90, 40, 20), re)
			// A shape that is off the clip still blurs onto it.
			c.Save()
			c.ClipRect(rect(160, 90, 30, 30))
			of := fillP(0xFFD02080)
			of.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 6))
			c.DrawRect(rect(130, 100, 28, 28), of)
			c.Restore()
		})},
		{"shadows: elevations, shapes, a transparent occluder and a scale", 192, 128, tolerance{}, canvas(192, 128, func(c *ui.Canvas) {
			c.DrawColor(ui.NewColor(0xFFF1EEF6), ui.BlendModeSrcOver)
			for i, e := range []float64{2, 6, 16} {
				card := ui.NewPath()
				card.AddRRect(ui.NewRRectFromRectXY(rect(float64(i)*60+10, 12, 40, 36), 8, 8))
				c.DrawShadow(card, ui.NewColor(0xFF000000), e, false)
				c.DrawPath(card, fillP(0xFFFFFFFF))
			}
			star := starPath().Shift(ui.Offset{Dx: -10, Dy: 55})
			c.DrawShadow(star, ui.NewColor(0xFF3030A0), 8, true)
			c.DrawPath(star, fillP(0x40FFFFFF))
			c.Save()
			c.Scale(1.5, 1.5)
			oval := ui.NewPath()
			oval.AddOval(rect(70, 50, 40, 24))
			c.DrawShadow(oval, ui.NewColor(0xFF000000), 4, false)
			c.DrawPath(oval, fillP(0xFFE0E040))
			c.Restore()
		})},
		{"backdrop filters: blur with a blend mode, a matrix, nested in a clip", 160, 128, tolerance{}, func() *ui.Scene {
			b := ui.NewSceneBuilder()
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) {
				for i := 0; i < 16; i++ {
					c.DrawRect(rect(float64(i)*10, 0, 10, 128), fillP([]uint32{0xFF3060C0, 0xFFF0A020}[i%2]))
				}
				c.DrawCircle(ui.Offset{Dx: 40, Dy: 40}, 26, fillP(0xFFE53935))
			}))
			b.PushClipRect(rect(10, 10, 60, 60))
			b.PushBackdropFilter(blur(6, 2, ui.TileModeUnspecified))
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) { c.DrawRect(rect(0, 0, 160, 128), fillP(0x40FFFFFF)) }))
			b.Pop()
			b.Pop()
			b.PushClipRRect(ui.NewRRectFromRectXY(rect(80, 10, 60, 50), 14, 14))
			b.PushBackdropFilter(ui.NewColorFilterMatrix([20]float64{0.3, 0.6, 0.1, 0, 0, 0.3, 0.6, 0.1, 0, 0, 0.3, 0.6, 0.1, 0, 0, 0, 0, 0, 1, 0}), ui.PushBackdropFilterOpts{BlendMode: ui.BlendModeSrcOver})
			b.Pop()
			b.Pop()
			b.PushClipRect(rect(20, 80, 100, 40))
			b.PushBackdropFilter(blur(10, 10, ui.TileModeClamp), ui.PushBackdropFilterOpts{BlendMode: ui.BlendModeMultiply})
			b.Pop()
			b.Pop()
			return b.Build()
		}},
		{"shader masks: blend modes and a transformed layer", 160, 128, tolerance{over8Permille: 4}, func() *ui.Scene {
			b := ui.NewSceneBuilder()
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) { c.DrawColor(ui.NewColor(0xFFEEEEEE), ui.BlendModeSrcOver) }))
			g := ui.NewGradientRadial(ui.Offset{Dx: 30, Dy: 30}, 30, []ui.Color{ui.NewColor(0xFFFFFFFF), ui.NewColor(0x00FFFFFF)})
			b.PushShaderMask(g, rect(10, 10, 60, 60), ui.BlendModeDstIn)
			b.AddPicture(ui.Offset{}, pictureOf(shapes))
			b.Pop()
			lin := ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 60, Dy: 0}, []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF0000FF)})
			b.PushTransform(ui.NewMatrix4TranslationValues(80, 60, 0))
			b.PushShaderMask(lin, rect(0, 0, 60, 60), ui.BlendModeSrcATop)
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) {
				c.DrawCircle(ui.Offset{Dx: 30, Dy: 30}, 24, fillP(0xFFFFFFFF))
			}))
			b.Pop()
			b.Pop()
			return b.Build()
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compareScene(t, tt.name, tt.scene(), tt.w, tt.h, tt.tol)
		})
	}
}
