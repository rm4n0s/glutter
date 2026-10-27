package gputest

import (
	"github.com/rm4n0s/glutter/ui"
)

// The size of the viewport of ScrolledList, and the height of one of its rows.
const (
	ScrolledWidth     = 800
	ScrolledHeight    = 600
	ScrolledRowHeight = 48
)

// ScrolledList is a tall list drawn through a viewport: rows rows of ScrolledRowHeight pixels, each
// with an icon made of curves (filled), a stroked outline with round joins, an arc and a dotted
// line, the way a page of icons and SVG pictures draws, clipped to the viewport of
// ScrolledWidth x ScrolledHeight pixels and moved up by scroll pixels. Only the rows in the viewport
// are visible; the rest are what a renderer must skip cheaply. It is for the benchmarks and the
// culling tests of the GPU renderers.
//
// Flutter: no counterpart
func ScrolledList(rows int, scroll float64) *ui.Scene {
	icon := ui.NewPath()
	icon.MoveTo(2, 12)
	icon.CubicTo(2, 4, 10, 0, 16, 6)
	icon.CubicTo(22, 0, 30, 4, 30, 12)
	icon.CubicTo(30, 20, 22, 28, 16, 32)
	icon.CubicTo(10, 28, 2, 20, 2, 12)
	icon.Close()
	icon.AddOval(ui.NewRectFromLTWH(10, 8, 12, 12))
	icon.QuadraticBezierTo(14, 40, 18, 24)
	for i := 0; i < 20; i++ {
		icon.RelativeLineTo(3, float64(i%5)-2)
	}
	icon.Close()

	fill := ui.NewPaint()
	fill.SetColor(ui.NewColor(0xFF6750A4))
	stroke := ui.NewPaint()
	stroke.SetColor(ui.NewColor(0xFF1D1B20))
	stroke.SetStyle(ui.PaintingStyleStroke)
	stroke.SetStrokeWidth(2)
	stroke.SetStrokeJoin(ui.StrokeJoinRound)
	stroke.SetStrokeCap(ui.StrokeCapRound)
	dots := ui.NewPaint()
	dots.SetColor(ui.NewColor(0xFF625B71))
	dots.SetStrokeWidth(3)

	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.ClipRect(ui.NewRectFromLTWH(0, 0, ScrolledWidth, ScrolledHeight))
	c.Translate(0, -scroll)
	for i := 0; i < rows; i++ {
		c.Save()
		c.Translate(0, float64(i)*ScrolledRowHeight+8)
		c.DrawPath(icon, fill)
		c.Translate(40, 0)
		c.DrawPath(icon, stroke)
		c.Translate(40, 0)
		c.DrawArc(ui.NewRectFromLTWH(0, 0, 32, 32), 0.3, 4.5, false, stroke)
		c.DrawPoints(ui.PointModePoints, []ui.Offset{{Dx: 0, Dy: 4}, {Dx: 8, Dy: 4}, {Dx: 16, Dy: 4}}, dots)
		c.DrawCircle(ui.Offset{Dx: 100, Dy: 16}, 12, stroke)
		c.Restore()
	}
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	return sb.Build()
}
