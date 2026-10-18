package soft

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/ui"
)

// sceneOf records the drawing of fn into a scene of one picture, shifted by offset.
func sceneOf(offset ui.Offset, fn func(c *ui.Canvas)) *ui.Scene {
	rec := ui.NewPictureRecorder()
	fn(ui.NewCanvas(rec))
	b := ui.NewSceneBuilder()
	b.AddPicture(offset, rec.EndRecording())
	return b.Build()
}

func renderOf(t *testing.T, w, h int, fn func(c *ui.Canvas)) *image.RGBA {
	t.Helper()
	img, err := Render(sceneOf(ui.Offset{}, fn), w, h)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return img
}

func paint(argb uint32) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(argb))
	return p
}

func stroked(argb uint32, width float64, cap ui.StrokeCap, join ui.StrokeJoin) *ui.Paint {
	p := paint(argb)
	p.SetStyle(ui.PaintingStyleStroke)
	p.SetStrokeWidth(width)
	p.SetStrokeCap(cap)
	p.SetStrokeJoin(join)
	return p
}

func white(c *ui.Canvas) { c.DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver) }

func vee() *ui.Path {
	p := ui.NewPath()
	p.MoveTo(15, 12)
	p.LineTo(50, 50)
	p.LineTo(85, 12)
	return p
}

func TestGoldenCaps(t *testing.T) {
	for _, tt := range []struct {
		name string
		cap  ui.StrokeCap
	}{{"butt", ui.StrokeCapButt}, {"round", ui.StrokeCapRound}, {"square", ui.StrokeCapSquare}} {
		t.Run(tt.name, func(t *testing.T) {
			img := renderOf(t, 100, 40, func(c *ui.Canvas) {
				white(c)
				p := ui.NewPath()
				p.MoveTo(20, 20)
				p.LineTo(80, 20)
				c.DrawPath(p, stroked(0xFF3050A0, 14, tt.cap, ui.StrokeJoinMiter))
				// The dot of a zero-length contour.
				d := ui.NewPath()
				d.MoveTo(10, 6)
				d.LineTo(10, 6)
				c.DrawPath(d, stroked(0xFFA03050, 8, tt.cap, ui.StrokeJoinMiter))
			})
			goldens.Check(t, "stroke-cap-"+tt.name, img)
		})
	}
}

func TestGoldenJoins(t *testing.T) {
	for _, tt := range []struct {
		name  string
		join  ui.StrokeJoin
		miter float64
	}{
		{"miter", ui.StrokeJoinMiter, 0},
		{"miter-limited", ui.StrokeJoinMiter, 1.2},
		{"round", ui.StrokeJoinRound, 0},
		{"bevel", ui.StrokeJoinBevel, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			img := renderOf(t, 100, 70, func(c *ui.Canvas) {
				white(c)
				p := stroked(0xFF206040, 16, ui.StrokeCapButt, tt.join)
				if tt.miter != 0 {
					p.SetStrokeMiterLimit(tt.miter)
				}
				c.DrawPath(vee(), p)
				// A closed triangle has a join at every corner, including the start.
				tri := ui.NewPath()
				tri.MoveTo(70, 56)
				tri.LineTo(90, 56)
				tri.LineTo(80, 40)
				tri.Close()
				c.DrawPath(tri, stroked(0xFF602040, 4, ui.StrokeCapButt, tt.join))
			})
			goldens.Check(t, "stroke-join-"+tt.name, img)
		})
	}
}

func TestGoldenFillRules(t *testing.T) {
	for _, tt := range []struct {
		name string
		typ  ui.PathFillType
	}{{"nonzero", ui.PathFillTypeNonZero}, {"evenodd", ui.PathFillTypeEvenOdd}} {
		t.Run(tt.name, func(t *testing.T) {
			img := renderOf(t, 100, 100, func(c *ui.Canvas) {
				white(c)
				p := ui.NewPath()
				for i := 0; i < 5; i++ {
					a := -math.Pi/2 + float64(i)*4*math.Pi/5
					x, y := 50+40*math.Cos(a), 50+40*math.Sin(a)
					if i == 0 {
						p.MoveTo(x, y)
					} else {
						p.LineTo(x, y)
					}
				}
				p.Close()
				p.SetFillType(tt.typ)
				c.DrawPath(p, paint(0xFF804080))
			})
			goldens.Check(t, "fill-star-"+tt.name, img)
		})
	}
	t.Run("ring", func(t *testing.T) {
		img := renderOf(t, 100, 100, func(c *ui.Canvas) {
			white(c)
			p := ui.NewPath()
			p.AddOval(ui.NewRectFromCircle(ui.Offset{Dx: 50, Dy: 50}, 40))
			p.AddOval(ui.NewRectFromCircle(ui.Offset{Dx: 50, Dy: 50}, 20))
			p.SetFillType(ui.PathFillTypeEvenOdd)
			c.DrawPath(p, paint(0xFFC04020))
		})
		goldens.Check(t, "fill-ring-evenodd", img)
	})
}

func TestGoldenTransformedPath(t *testing.T) {
	img := renderOf(t, 120, 100, func(c *ui.Canvas) {
		white(c)
		c.Translate(60, 50)
		c.Rotate(math.Pi / 6)
		r := ui.NewPath()
		r.AddRect(ui.NewRectFromLTWH(-40, -20, 80, 40))
		c.DrawPath(r, paint(0xFFF0B000))
		// A stroke under a non-uniform scale is thicker on one axis.
		c.Scale(1.4, 0.6)
		o := ui.NewPath()
		o.AddOval(ui.NewRectFromCircle(ui.Offset{}, 30))
		c.DrawPath(o, stroked(0x99206090, 10, ui.StrokeCapButt, ui.StrokeJoinMiter))
	})
	goldens.Check(t, "path-transformed", img)
}

func TestGoldenHairlineAndAliasing(t *testing.T) {
	img := renderOf(t, 100, 60, func(c *ui.Canvas) {
		white(c)
		hairline := stroked(0xFF000000, 0, ui.StrokeCapButt, ui.StrokeJoinMiter)
		p := ui.NewPath()
		p.MoveTo(5, 5)
		p.LineTo(95, 25)
		c.DrawPath(p, hairline)
		// Hairlines stay one pixel wide under a scale.
		c.Save()
		c.Scale(3, 3)
		q := ui.NewPath()
		q.MoveTo(2, 10)
		q.LineTo(30, 14)
		c.DrawPath(q, hairline)
		c.Restore()
		// Without anti-aliasing every pixel is all or nothing.
		a := paint(0xFF2040C0)
		a.SetIsAntiAlias(false)
		tri := ui.NewPath()
		tri.MoveTo(60, 55)
		tri.LineTo(95, 55)
		tri.LineTo(95, 30)
		tri.Close()
		c.DrawPath(tri, a)
	})
	goldens.Check(t, "path-hairline-noaa", img)
}

func TestRenderPixels(t *testing.T) {
	// An opaque rect on a transparent surface: exact pixels, premultiplied.
	img := renderOf(t, 8, 8, func(c *ui.Canvas) {
		p := ui.NewPath()
		p.AddRect(ui.NewRectFromLTWH(2, 2, 4, 4))
		c.DrawPath(p, paint(0x80FF0000))
	})
	tests := []struct {
		x, y int
		want [4]uint8
	}{
		{0, 0, [4]uint8{0, 0, 0, 0}},
		{2, 2, [4]uint8{128, 0, 0, 128}},
		{5, 5, [4]uint8{128, 0, 0, 128}},
		{6, 6, [4]uint8{0, 0, 0, 0}},
	}
	for _, tt := range tests {
		i := img.PixOffset(tt.x, tt.y)
		got := [4]uint8{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}
		if got != tt.want {
			t.Errorf("pixel %d,%d = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
	// A second layer of the same color composites with source-over.
	img = renderOf(t, 4, 4, func(c *ui.Canvas) {
		white(c)
		p := ui.NewPath()
		p.AddRect(ui.NewRectFromLTWH(0, 0, 4, 4))
		c.DrawPath(p, paint(0x80000000))
	})
	if got := img.Pix[0]; got != 127 && got != 128 {
		t.Errorf("black at 50%% over white = %d, want about 127", got)
	}
	if got := img.Pix[3]; got != 255 {
		t.Errorf("alpha = %d, want 255", got)
	}
}

func TestRenderLayersAndState(t *testing.T) {
	rect := func() *ui.Path {
		p := ui.NewPath()
		p.AddRect(ui.NewRectFromLTWH(0, 0, 2, 2))
		return p
	}
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.Save()
	c.Translate(1, 0)
	c.DrawPath(rect(), paint(0xFF0000FF))
	c.Restore()
	c.Restore() // nothing to restore; ignored
	c.DrawPath(rect(), paint(0xFF00FF00))
	pic := rec.EndRecording()

	b := ui.NewSceneBuilder()
	b.PushOffset(0, 4)
	b.PushTransform(ui.NewMatrix4TranslationValues(1, 0, 0))
	b.AddPicture(ui.Offset{Dx: 4}, pic)
	scene := b.Build()
	img, err := Render(scene, 10, 10)
	if err != nil {
		t.Fatal(err)
	}
	// The layers and the picture's offset move everything by 5, 4.
	for _, tt := range []struct {
		x, y int
		a    uint8
		g, b uint8
	}{
		{5, 4, 255, 255, 0}, // green
		{6, 5, 255, 255, 0}, // green, drawn over the blue
		{7, 4, 255, 0, 255}, // blue, the part green does not reach
		{4, 4, 0, 0, 0},
		{8, 4, 0, 0, 0},
	} {
		i := img.PixOffset(tt.x, tt.y)
		if img.Pix[i+3] != tt.a || img.Pix[i+1] != tt.g || img.Pix[i+2] != tt.b {
			t.Errorf("pixel %d,%d = %v, want alpha %d g %d b %d", tt.x, tt.y, img.Pix[i:i+4], tt.a, tt.g, tt.b)
		}
	}
}

func TestRenderErrors(t *testing.T) {
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(ui.NewRectFromLTWH(0, 0, 1, 1), paint(0xFF000000))
	rectPic := rec.EndRecording()

	pathWith := func(p *ui.Paint) *ui.Scene {
		return sceneOf(ui.Offset{}, func(c *ui.Canvas) {
			path := ui.NewPath()
			path.AddRect(ui.NewRectFromLTWH(0, 0, 2, 2))
			c.DrawPath(path, p)
		})
	}
	blend := paint(0xFF000000)
	blend.SetBlendMode(ui.BlendModeMultiply)
	filter := paint(0xFF000000)
	filter.SetColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFF000000), ui.BlendModeSrcOver))
	inverted := paint(0xFF000000)
	inverted.SetInvertColors(true)
	blurred := paint(0xFF000000)
	blurred.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 2))

	opacityScene := func() *ui.Scene {
		b := ui.NewSceneBuilder()
		b.PushOpacity(100)
		b.AddPicture(ui.Offset{}, rectPic)
		return b.Build()
	}()

	tests := []struct {
		name  string
		scene *ui.Scene
		w, h  int
		tag   string
	}{
		{"nil scene", nil, 10, 10, "SoftSceneNil"},
		{"zero width", pathWith(paint(0xFF000000)), 0, 10, "SoftSizeInvalid"},
		{"negative height", pathWith(paint(0xFF000000)), 10, -1, "SoftSizeInvalid"},
		{"too large", pathWith(paint(0xFF000000)), 1 << 20, 10, "SoftSizeInvalid"},
		{"opacity layer", opacityScene, 10, 10, "SoftNotImplemented"},
		{"save layer", sceneOf(ui.Offset{}, func(c *ui.Canvas) { c.SaveLayer(nil, paint(0x80000000)) }), 10, 10, "SoftNotImplemented"},
		{"shadow", sceneOf(ui.Offset{}, func(c *ui.Canvas) { c.DrawShadow(ui.NewPath(), ui.NewColor(0xFF000000), 2, false) }), 10, 10, "SoftNotImplemented"},
		{"blend mode", pathWith(blend), 10, 10, "SoftNotImplemented"},
		{"color filter", pathWith(filter), 10, 10, "SoftNotImplemented"},
		{"invert colors", pathWith(inverted), 10, 10, "SoftNotImplemented"},
		{"mask filter", pathWith(blurred), 10, 10, "SoftNotImplemented"},
		{"draw color blend mode", sceneOf(ui.Offset{}, func(c *ui.Canvas) { c.DrawColor(ui.NewColor(0xFF000000), ui.BlendModeClear) }), 10, 10, "SoftNotImplemented"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := Render(tt.scene, tt.w, tt.h)
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.tag) {
				t.Fatalf("expected %s, got %v", tt.tag, err)
			}
			if img != nil {
				t.Error("an image came back with the error")
			}
		})
	}
}
