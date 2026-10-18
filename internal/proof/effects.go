package proof

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

func init() {
	register(Scene{
		Name:        "effects",
		Description: "soft shadows under cards at six elevations, a frosted-glass rectangle over stripes, mask-filter blur styles, and shapes faded by a gradient mask",
		Width:       640,
		Height:      480,
		Build:       buildEffects,
	})
}

func picture(width, height float64, fn func(c *ui.Canvas)) *ui.Picture {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.NewRectFromLTWH(0, 0, width, height))})
	fn(c)
	return rec.EndRecording()
}

func buildEffects() *ui.Scene {
	b := ui.NewSceneBuilder()

	// The background and the cards with their shadows, in one picture.
	b.AddPicture(ui.Offset{}, picture(640, 480, func(c *ui.Canvas) {
		c.DrawColor(ui.NewColor(0xFFF1EEF6), ui.BlendModeSrcOver)
		for i, e := range []float64{1, 2, 4, 8, 12, 24} {
			card := ui.NewRRectFromRectXY(ui.NewRectFromLTWH(float64(i)*104+22, 28, 72, 72), 10, 10)
			p := ui.NewPath()
			p.AddRRect(card)
			c.DrawShadow(p, ui.NewColor(0xFF000000), e, false)
			c.DrawRRect(card, fillPaint(0xFFFFFFFF))
		}
		// Mask filter blur styles on rounded squares.
		for i, style := range []ui.BlurStyle{ui.BlurStyleNormal, ui.BlurStyleSolid, ui.BlurStyleOuter, ui.BlurStyleInner} {
			p := fillPaint(0xFF6750A4)
			p.SetMaskFilter(ui.NewMaskFilterBlur(style, 6))
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(float64(i)*76+336, 142, 56, 56), 14, 14), p)
		}
		// A shadow that follows a star and a circle.
		star := ui.NewPath()
		for i := 0; i < 5; i++ {
			a := -math.Pi/2 + float64(i)*4*math.Pi/5
			x, y := 70+34*math.Cos(a), 172+34*math.Sin(a)
			if i == 0 {
				star.MoveTo(x, y)
			} else {
				star.LineTo(x, y)
			}
		}
		star.Close()
		c.DrawShadow(star, ui.NewColor(0xFF000000), 8, false)
		c.DrawPath(star, fillPaint(0xFFF2B600))
		circle := ui.NewPath()
		circle.AddOval(ui.NewRectFromCircle(ui.Offset{Dx: 190, Dy: 172}, 34))
		c.DrawShadow(circle, ui.NewColor(0xFF1E88E5), 10, false)
		c.DrawPath(circle, fillPaint(0xFF1E88E5))
	}))

	// Stripes, with a frosted-glass rectangle over the right of them: the
	// backdrop is blurred inside a rounded clip, and a translucent white panel
	// is drawn on top.
	b.AddPicture(ui.Offset{}, picture(640, 480, func(c *ui.Canvas) {
		c.Save()
		c.ClipRect(ui.NewRectFromLTWH(20, 230, 280, 230))
		for i := 0; i < 20; i++ {
			col := uint32(0xFF3060C0)
			if i%2 == 1 {
				col = 0xFFF0A020
			}
			c.DrawRect(ui.NewRectFromLTWH(20+float64(i)*14, 230, 14, 230), fillPaint(col))
		}
		c.DrawCircle(ui.Offset{Dx: 90, Dy: 300}, 36, fillPaint(0xFFE53935))
		c.Restore()
	}))
	b.PushClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(110, 270, 160, 150), 20, 20))
	b.PushBackdropFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 8, SigmaY: 8}))
	b.AddPicture(ui.Offset{}, picture(640, 480, func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(110, 270, 160, 150), fillPaint(0x66FFFFFF))
	}))
	b.Pop()
	b.Pop()

	// Shapes faded out to the right by a gradient mask: the layer is drawn,
	// then the gradient's alpha is multiplied in (dstIn).
	mask := ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 280},
		[]ui.Color{ui.NewColor(0xFFFFFFFF), ui.NewColor(0x00FFFFFF)})
	b.PushShaderMask(mask, ui.NewRectFromLTWH(340, 250, 280, 190), ui.BlendModeDstIn)
	b.AddPicture(ui.Offset{}, picture(640, 480, func(c *ui.Canvas) {
		colors := []uint32{0xFFE53935, 0xFFFDD835, 0xFF43A047, 0xFF1E88E5}
		for i, col := range colors {
			c.DrawCircle(ui.Offset{Dx: 372 + float64(i)*70, Dy: 300}, 30, fillPaint(col))
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(342+float64(i)*70, 350, 60, 70), 12, 12), fillPaint(col))
		}
	}))
	b.Pop()
	return b.Build()
}
