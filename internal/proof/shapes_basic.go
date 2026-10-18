package proof

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

func init() {
	register(Scene{
		Name:        "shapes-basic",
		Description: "a filled star, stroked curves with every cap and join, an even-odd ring and a rotated rectangle",
		Width:       640,
		Height:      480,
		Build:       func() *ui.Scene { return singlePicture(640, 480, drawShapesBasic) },
	})
}

func fillPaint(c uint32) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(c))
	return p
}

func strokePaint(c uint32, width float64, cap ui.StrokeCap, join ui.StrokeJoin) *ui.Paint {
	p := fillPaint(c)
	p.SetStyle(ui.PaintingStyleStroke)
	p.SetStrokeWidth(width)
	p.SetStrokeCap(cap)
	p.SetStrokeJoin(join)
	return p
}

// star returns a pentagram through five points on a circle.
func star(cx, cy, r float64) *ui.Path {
	p := ui.NewPath()
	for i := 0; i < 5; i++ {
		a := -math.Pi/2 + float64(i)*4*math.Pi/5
		x, y := cx+r*math.Cos(a), cy+r*math.Sin(a)
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return p
}

// zigzag returns an open path with a sharp corner and a curve.
func zigzag(x, y float64) *ui.Path {
	p := ui.NewPath()
	p.MoveTo(x, y+60)
	p.LineTo(x+50, y)
	p.LineTo(x+100, y+60)
	p.QuadraticBezierTo(x+130, y+100, x+160, y+40)
	return p
}

func drawShapesBasic(c *ui.Canvas) {
	// A filled star with the non-zero rule: the middle is filled.
	c.DrawPath(star(110, 110, 90), fillPaint(0xFF6750A4))
	// The same star with the even-odd rule: the middle is a hole.
	eo := star(320, 110, 90)
	eo.SetFillType(ui.PathFillTypeEvenOdd)
	c.DrawPath(eo, fillPaint(0xFF006A6A))

	// A ring: two circles in one path, filled even-odd.
	ring := ui.NewPath()
	ring.AddOval(ui.NewRectFromCircle(ui.Offset{Dx: 530, Dy: 110}, 80))
	ring.AddOval(ui.NewRectFromCircle(ui.Offset{Dx: 530, Dy: 110}, 45))
	ring.SetFillType(ui.PathFillTypeEvenOdd)
	c.DrawPath(ring, fillPaint(0xFFB3261E))

	// Stroked curves: the caps in a row (butt, round, square), then the joins
	// (miter, round, bevel).
	caps := []ui.StrokeCap{ui.StrokeCapButt, ui.StrokeCapRound, ui.StrokeCapSquare}
	joins := []ui.StrokeJoin{ui.StrokeJoinMiter, ui.StrokeJoinRound, ui.StrokeJoinBevel}
	for i := range caps {
		x := 20 + float64(i)*210
		c.DrawPath(zigzag(x, 240), strokePaint(0xFF1D1B20, 18, caps[i], joins[i]))
		// The same curve as a thin line shows where the path itself runs.
		c.DrawPath(zigzag(x, 240), strokePaint(0xFFFFFFFF, 1, ui.StrokeCapButt, ui.StrokeJoinMiter))
	}

	// A rotated rectangle, filled with an outline.
	c.Save()
	c.Translate(120, 400)
	c.Rotate(math.Pi / 9)
	rect := ui.NewPath()
	rect.AddRect(ui.NewRectFromLTWH(-80, -35, 160, 70))
	c.DrawPath(rect, fillPaint(0xFFF2B600))
	c.DrawPath(rect, strokePaint(0xFF7D5700, 4, ui.StrokeCapButt, ui.StrokeJoinMiter))
	c.Restore()

	// A hairline and a stroked rounded shape under a scale.
	c.DrawPath(star(320, 400, 60), strokePaint(0xFF1D1B20, 0, ui.StrokeCapButt, ui.StrokeJoinMiter))
	c.Save()
	c.Translate(520, 400)
	c.Scale(1.6, 0.8)
	round := ui.NewPath()
	round.AddOval(ui.NewRectFromCircle(ui.Offset{}, 45))
	c.DrawPath(round, strokePaint(0xFF006A6A, 10, ui.StrokeCapRound, ui.StrokeJoinRound))
	c.Restore()
}
