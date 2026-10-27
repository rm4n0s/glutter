package gputest

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// CullShape is one kind of shape command for the culling tests of the GPU renderers: Draw draws a
// shape that fits the rectangle with the paint.
type CullShape struct {
	Name string
	// Atlas says that the renderers draw the shape from the coverage atlas, so that drawing it in
	// view asks the atlas for tiles.
	Atlas bool
	Draw  func(c *ui.Canvas, r ui.Rect, p *ui.Paint)
}

// CullShapes are the kinds of shape commands that culling decides about.
func CullShapes() []CullShape {
	center := func(r ui.Rect) ui.Offset { return ui.Offset{Dx: (r.Left + r.Right) / 2, Dy: (r.Top + r.Bottom) / 2} }
	return []CullShape{
		{"rect", false, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) { c.DrawRect(r, p) }},
		{"rrect", false, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) { c.DrawRRect(ui.NewRRectFromRectXY(r, 4, 4), p) }},
		{"drrect", false, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) {
			c.DrawDRRect(ui.NewRRectFromRectXY(r, 4, 4), ui.NewRRectFromRectXY(r.Deflate(5), 2, 2), p)
		}},
		{"rsuperellipse", false, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) {
			c.DrawRSuperellipse(ui.NewRSuperellipseFromRectXY(r, 6, 6), p)
		}},
		{"oval", false, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) { c.DrawOval(r, p) }},
		{"circle", false, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) { c.DrawCircle(center(r), (r.Right-r.Left)/2, p) }},
		{"arc", true, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) { c.DrawArc(r, 0.3, 3, false, p) }},
		{"path", true, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) {
			path := ui.NewPath()
			path.MoveTo(r.Left, r.Top)
			path.CubicTo(r.Right, r.Top, r.Right, r.Bottom, r.Left, r.Bottom)
			path.Close()
			c.DrawPath(path, p)
		}},
		{"line", false, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) {
			c.DrawLine(ui.Offset{Dx: r.Left, Dy: r.Top}, ui.Offset{Dx: r.Right, Dy: r.Bottom}, p)
		}},
		{"points", false, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) {
			c.DrawPoints(ui.PointModePolygon, []ui.Offset{{Dx: r.Left, Dy: r.Top}, {Dx: r.Right, Dy: r.Bottom}, {Dx: r.Left, Dy: r.Bottom}}, p)
		}},
		{"image", false, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) {
			c.DrawImageRect(ui.NewImageFromRGBA(make([]byte, 16*8*4), 16, 8), ui.NewRectFromLTWH(0, 0, 16, 8), r, p)
		}},
		{"shadow", false, func(c *ui.Canvas, r ui.Rect, p *ui.Paint) {
			path := ui.NewPath()
			path.AddRRect(ui.NewRRectFromRectXY(r, 4, 4))
			c.DrawShadow(path, ui.NewColor(0xFF000000), 4, false)
		}},
	}
}

// CullTransform is a transform the culling tests draw under.
type CullTransform struct {
	Name  string
	Apply func(c *ui.Canvas)
}

// CullTransforms are a translation, a scale and a rotation (and no transform).
func CullTransforms() []CullTransform {
	return []CullTransform{
		{"identity", func(c *ui.Canvas) {}},
		{"translation", func(c *ui.Canvas) { c.Translate(7.25, -3.5) }},
		{"scale", func(c *ui.Canvas) { c.Scale(2, 3) }},
		{"rotation", func(c *ui.Canvas) { c.Rotate(math.Pi / 6) }},
	}
}
