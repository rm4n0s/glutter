package proof

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

func init() {
	register(Scene{
		Name:        "shapes-clips",
		Description: "a grid with one cell for each shape type, a nine-patch, and an image clipped by a rotated rounded rectangle",
		Width:       640,
		Height:      480,
		Build:       func() *ui.Scene { return singlePicture(640, 480, drawShapesClips) },
	})
}

// testImage is a 64x64 image: a color gradient with a white grid over it.
func testImage() *ui.Image {
	const n = 64
	pix := make([]byte, 4*n*n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			r, g, b := byte(x*255/(n-1)), byte(y*255/(n-1)), byte(255-x*255/(n-1))
			if x%16 == 0 || y%16 == 0 {
				r, g, b = 255, 255, 255
			}
			copy(pix[4*(y*n+x):], []byte{r, g, b, 255})
		}
	}
	return ui.NewImageFromRGBA(pix, n, n)
}

// frameImage is a 24x24 image for the nine-patch: a dark border with rounded
// look-alike corners and a light center.
func frameImage() *ui.Image {
	const n = 24
	pix := make([]byte, 4*n*n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			c := []byte{0x30, 0x60, 0xC0, 255}
			switch {
			case x >= 8 && x < 16 && y >= 8 && y < 16:
				c = []byte{0xF6, 0xF0, 0xDC, 255}
			case x < 8 && y < 8 && (x+y)%8 == 0, x >= 16 && y >= 16 && (x+y)%8 == 0:
				c = []byte{0xF0, 0xA0, 0x20, 255}
			}
			copy(pix[4*(y*n+x):], c)
		}
	}
	return ui.NewImageFromRGBA(pix, n, n)
}

func drawShapesClips(c *ui.Canvas) {
	const cw, ch = 160.0, 160.0
	cell := func(i int, fn func(r ui.Rect)) {
		r := ui.NewRectFromLTWH(float64(i%4)*cw, float64(i/4)*ch, cw, ch)
		c.Save()
		c.ClipRect(r)
		// A light cell with a thin frame.
		bg := fillPaint(0xFFF4F1F8)
		c.DrawRect(r.Deflate(4), bg)
		fn(r.Deflate(18))
		c.Restore()
	}
	blue, plum, teal, red := uint32(0xFF3060C0), uint32(0xFF6750A4), uint32(0xFF006A6A), uint32(0xFFB3261E)

	cell(0, func(r ui.Rect) { c.DrawRect(r, fillPaint(blue)) })
	cell(1, func(r ui.Rect) { c.DrawRRect(ui.NewRRectFromRectXY(r, 28, 20), fillPaint(plum)) })
	cell(2, func(r ui.Rect) { c.DrawRSuperellipse(ui.NewRSuperellipseFromRectXY(r, 34, 34), fillPaint(teal)) })
	cell(3, func(r ui.Rect) {
		c.DrawOval(ui.NewRectFromLTWH(r.Left, r.Top+22, r.Width(), r.Height()-44), fillPaint(red))
	})
	cell(4, func(r ui.Rect) {
		c.DrawCircle(r.Center(), r.Width()/2, strokePaint(blue, 14, ui.StrokeCapButt, ui.StrokeJoinMiter))
	})
	cell(5, func(r ui.Rect) {
		c.DrawDRRect(ui.NewRRectFromRectXY(r, 24, 24), ui.NewRRectFromRectXY(r.Deflate(22), 8, 8), fillPaint(plum))
	})
	cell(6, func(r ui.Rect) {
		c.DrawArc(r, -0.35*math.Pi, 1.3*math.Pi, true, fillPaint(teal))
		c.DrawArc(r.Deflate(-6), -0.35*math.Pi, 1.3*math.Pi, false, strokePaint(red, 5, ui.StrokeCapRound, ui.StrokeJoinMiter))
	})
	cell(7, func(r ui.Rect) {
		c.DrawLine(r.TopLeft(), r.BottomRight(), strokePaint(blue, 16, ui.StrokeCapRound, ui.StrokeJoinMiter))
		c.DrawLine(r.BottomLeft(), r.TopRight(), strokePaint(red, 16, ui.StrokeCapSquare, ui.StrokeJoinMiter))
		c.DrawPoints(ui.PointModePoints, []ui.Offset{r.Center()}, strokePaint(0xFFFFFFFF, 14, ui.StrokeCapRound, ui.StrokeJoinMiter))
	})
	cell(8, func(r ui.Rect) {
		colors := []ui.Color{ui.NewColor(0xFFE53935), ui.NewColor(0xFFFDD835), ui.NewColor(0xFF43A047), ui.NewColor(0xFF1E88E5), ui.NewColor(0xFF8E24AA), ui.NewColor(0xFFFFFFFF)}
		pts := []ui.Offset{r.TopLeft(), r.TopRight(), r.Center(), r.BottomRight(), r.BottomLeft(), {Dx: r.Left, Dy: r.Center().Dy}}
		v := ui.NewVertices(ui.VertexModeTriangleFan, append([]ui.Offset{r.Center()}, pts[0], pts[1], pts[3], pts[4], pts[0]),
			ui.VerticesOpts{Colors: []ui.Color{colors[5], colors[0], colors[1], colors[3], colors[2], colors[0]}})
		c.DrawVertices(v, ui.BlendModeDst, fillPaint(0xFF000000))
	})
	cell(9, func(r ui.Rect) {
		// The same nine-patch at two sizes: the corners keep their size.
		p := ui.NewPaint()
		p.SetFilterQuality(ui.FilterQualityNone)
		c.DrawImageNine(frameImage(), ui.NewRectFromLTWH(8, 8, 8, 8), ui.NewRectFromLTWH(r.Left, r.Top, r.Width(), 44), p)
		c.DrawImageNine(frameImage(), ui.NewRectFromLTWH(8, 8, 8, 8), ui.NewRectFromLTWH(r.Left, r.Top+56, 56, r.Height()-56), p)
	})
	cell(10, func(r ui.Rect) {
		// Image quality: nearest, bilinear, and a mipmapped downscale.
		img := testImage()
		half := (r.Width() - 8) / 2
		for i, q := range []ui.FilterQuality{ui.FilterQualityNone, ui.FilterQualityLow} {
			p := ui.NewPaint()
			p.SetFilterQuality(q)
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 8, 8), ui.NewRectFromLTWH(r.Left+float64(i)*(half+8), r.Top, half, half), p)
		}
		p := ui.NewPaint()
		p.SetFilterQuality(ui.FilterQualityMedium)
		c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 64, 64), ui.NewRectFromLTWH(r.Left, r.Top+half+8, half, half), p)
	})
	cell(11, func(r ui.Rect) {
		// An image clipped by a rounded rectangle that is rotated.
		c.Translate(r.Center().Dx, r.Center().Dy)
		c.Rotate(math.Pi / 9)
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromCenter(ui.Offset{}, r.Width(), r.Height()*0.7), 26, 26))
		c.Rotate(-math.Pi / 9)
		p := ui.NewPaint()
		p.SetFilterQuality(ui.FilterQualityLow)
		c.DrawImageRect(testImage(), ui.NewRectFromLTWH(0, 0, 64, 64), ui.NewRectFromCenter(ui.Offset{}, r.Width()*1.2, r.Width()*1.2), p)
	})
}
