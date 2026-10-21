// Package gputest has what the display tests of the GPU renderers share: the scenes the
// renderers are compared on, the comparison with the software renderer's image, and its
// tolerance. It is test support and has no Flutter counterpart.
package gputest

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// Diff compares two images inside a rectangle: the number of pixels whose largest channel
// difference is above each threshold, and the largest difference.
type Diff struct {
	Pixels, Over8, Over32, Over96, Max int
}

// DiffRect compares the images a and b inside r.
func DiffRect(a, b *image.RGBA, r image.Rectangle) Diff {
	var s Diff
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			ia, ib := a.PixOffset(x, y), b.PixOffset(x, y)
			m := 0
			for c := 0; c < 4; c++ {
				d := int(a.Pix[ia+c]) - int(b.Pix[ib+c])
				if d < 0 {
					d = -d
				}
				m = max(m, d)
			}
			s.Pixels++
			if m > 8 {
				s.Over8++
			}
			if m > 32 {
				s.Over32++
			}
			if m > 96 {
				s.Over96++
			}
			s.Max = max(s.Max, m)
		}
	}
	return s
}

// Tolerance is how far a GPU image may be from the software renderer's in a region: of its
// pixels, at most Over8Permille differ by more than 8 levels (the edges, where the distance
// field and the exact area differ a little), at most Over32Permille by more than 32, and at
// most Over96 pixels by more than 96.
type Tolerance struct{ Over8Permille, Over32Permille, Over96 int }

// Check logs the difference and fails the test when it is more than the tolerance.
func (tol Tolerance) Check(t testing.TB, label string, s Diff) {
	t.Helper()
	t.Logf("%-28s differs: >8 %5d  >32 %4d  >96 %3d  max %3d of %d pixels", label, s.Over8, s.Over32, s.Over96, s.Max, s.Pixels)
	if s.Over8*1000 > tol.Over8Permille*s.Pixels || s.Over32*1000 > tol.Over32Permille*s.Pixels || s.Over96 > tol.Over96 {
		t.Errorf("%s: the GPU image is too far from the software image", label)
	}
}

// FillPaint is an anti-aliased paint of the color.
func FillPaint(argb uint32) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(argb))
	return p
}

// StrokePaint is a stroking paint of the color with the width, cap and join.
func StrokePaint(argb uint32, width float64, cap ui.StrokeCap, join ui.StrokeJoin) *ui.Paint {
	p := FillPaint(argb)
	p.SetStyle(ui.PaintingStyleStroke)
	p.SetStrokeWidth(width)
	p.SetStrokeCap(cap)
	p.SetStrokeJoin(join)
	return p
}

// CanvasScene is a scene of one picture: a white background, then what fn draws.
func CanvasScene(fn func(c *ui.Canvas)) *ui.Scene {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver)
	fn(c)
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	return b.Build()
}

// ShapeScene is a drawing of analytic shapes on a surface of ShapeSize pixels, with the
// tolerance the GPU renderers are held to against the software renderer.
type ShapeScene struct {
	Name string
	// Over8Permille, Over32Permille and Over96 are the Tolerance.
	Over8Permille, Over32Permille, Over96 int
	Draw                                  func(c *ui.Canvas)
}

// Tolerance returns the scene's tolerance.
func (s ShapeScene) Tolerance() Tolerance {
	return Tolerance{s.Over8Permille, s.Over32Permille, s.Over96}
}

// ShapeSize is the width and height of the surface of the shape scenes.
const ShapeSize = 128

// ShapeScenes returns the drawings of shapes, clips and the blend modes of the blend unit
// that both GPU renderers are compared with the software renderer on.
func ShapeScenes() []ShapeScene {
	r10 := ui.NewRectFromLTWH(10, 10, 60, 40)
	return []ShapeScene{
		{"rotated rect and rrect", 25, 3, 0, func(c *ui.Canvas) {
			c.Save()
			c.Translate(60, 60)
			c.Rotate(0.5)
			c.DrawRect(ui.NewRectFromLTWH(-40, -25, 80, 50), FillPaint(0xFF2060C0))
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(-30, -15, 60, 30), 10, 6), FillPaint(0xCCF0A000))
			c.Restore()
		}},
		{"non-uniform scale and skew", 30, 4, 0, func(c *ui.Canvas) {
			c.Save()
			c.Translate(20, 20)
			c.Scale(1.6, 0.7)
			c.DrawCircle(ui.Offset{Dx: 30, Dy: 30}, 25, StrokePaint(0xFF802080, 6, ui.StrokeCapButt, ui.StrokeJoinMiter))
			c.DrawOval(ui.NewRectFromLTWH(10, 70, 60, 30), FillPaint(0xFF20A060))
			c.Restore()
			c.Save()
			c.Translate(20, 90)
			c.Skew(0.4, 0)
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(0, 0, 70, 30), 8, 8), FillPaint(0xFFC04020))
			c.Restore()
		}},
		{"mirrored", 20, 2, 0, func(c *ui.Canvas) {
			c.Translate(120, 0)
			c.Scale(-1, 1)
			c.DrawRRect(ui.NewRRectFromRectAndCorners(ui.NewRectFromLTWH(10, 10, 70, 50), ui.Corners{TopLeft: ui.Radius{X: 20, Y: 20}, BottomRight: ui.Radius{X: 12, Y: 6}}), FillPaint(0xFF3050A0))
		}},
		{"fractional anti-aliased clip", 20, 2, 0, func(c *ui.Canvas) {
			c.ClipRect(ui.NewRectFromLTWH(10.3, 10.6, 60.4, 50.5))
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 128, 128), FillPaint(0xFF204080))
			c.DrawCircle(ui.Offset{Dx: 70, Dy: 60}, 30, FillPaint(0xCCFF8000))
		}},
		{"hard clip and nested clips", 20, 2, 0, func(c *ui.Canvas) {
			c.Save()
			c.ClipRect(ui.NewRectFromLTWH(10.3, 10.6, 60.4, 50.5), ui.ClipRectOpts{DoAntiAlias: ui.Ptr(false)})
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 128, 128), FillPaint(0xFF204080))
			c.Restore()
			c.Translate(5, 70)
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 80, 40))
			c.ClipRect(ui.NewRectFromLTWH(20.5, 10.5, 80, 40))
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(0, 0, 120, 60), 20, 20), FillPaint(0xFFA02060))
		}},
		{"translucent overlaps and blend modes", 10, 1, 0, func(c *ui.Canvas) {
			c.DrawRect(ui.NewRectFromLTWH(10, 10, 50, 50), FillPaint(0x802060C0))
			c.DrawRect(ui.NewRectFromLTWH(35, 35, 50, 50), FillPaint(0x80C06020))
			dst := FillPaint(0xFF30A030)
			dst.SetBlendMode(ui.BlendModeDstOver)
			c.DrawRect(ui.NewRectFromLTWH(0, 80, 128, 20), dst)
			plus := FillPaint(0x80402010)
			plus.SetBlendMode(ui.BlendModePlus)
			c.DrawCircle(ui.Offset{Dx: 95, Dy: 35}, 25, plus)
		}},
		// At the corners of strokes of a pixel or less the distance field misses the
		// union of two strips, and the software renderer counts both: a few pixels are
		// far from each other, neither being exact.
		{"strokes of every width", 25, 4, 20, func(c *ui.Canvas) {
			for i, w := range []float64{0, 0.4, 1, 2.5, 6} {
				x := 8 + float64(i)*24
				c.DrawRect(ui.NewRectFromLTWH(x, 8, 18, 30), StrokePaint(0xFF000000, w, ui.StrokeCapButt, ui.StrokeJoinMiter))
				c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(x, 46, 18, 30), 6, 6), StrokePaint(0xFF1030A0, w, ui.StrokeCapButt, ui.StrokeJoinMiter))
				c.DrawRect(ui.NewRectFromLTWH(x, 84, 18, 30), StrokePaint(0xFFA03010, w, ui.StrokeCapButt, ui.StrokeJoinRound))
			}
		}},
		{"stroke with a radius smaller than the width", 25, 3, 0, func(c *ui.Canvas) {
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(15, 15, 90, 40), 3, 3), StrokePaint(0xFF108040, 12, ui.StrokeCapButt, ui.StrokeJoinMiter))
			c.DrawOval(ui.NewRectFromLTWH(15, 70, 90, 40), StrokePaint(0xFF8010A0, 5, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}},
		{"radii that do not fit", 20, 2, 0, func(c *ui.Canvas) {
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10, 10, 100, 30), 60, 40), FillPaint(0xFF206080))
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10, 50, 30, 70), 40, 40), FillPaint(0xFF806020))
		}},
		{"lines, caps and points", 30, 3, 0, func(c *ui.Canvas) {
			caps := []ui.StrokeCap{ui.StrokeCapButt, ui.StrokeCapRound, ui.StrokeCapSquare}
			for i, cp := range caps {
				y := 20 + float64(i)*30
				c.DrawLine(ui.Offset{Dx: 15, Dy: y}, ui.Offset{Dx: 70, Dy: y + 12}, StrokePaint(0xFF2040A0, 12, cp, ui.StrokeJoinMiter))
				c.DrawLine(ui.Offset{Dx: 90, Dy: y}, ui.Offset{Dx: 90, Dy: y}, StrokePaint(0xFFA04020, 14, cp, ui.StrokeJoinMiter))
			}
			c.DrawPoints(ui.PointModePoints, []ui.Offset{{Dx: 20, Dy: 110}, {Dx: 40, Dy: 112}}, StrokePaint(0xFF008060, 9, ui.StrokeCapRound, ui.StrokeJoinMiter))
			c.DrawPoints(ui.PointModePoints, []ui.Offset{{Dx: 60, Dy: 110}}, StrokePaint(0xFF008060, 9, ui.StrokeCapButt, ui.StrokeJoinMiter))
			c.DrawPoints(ui.PointModeLines, []ui.Offset{{Dx: 80, Dy: 105}, {Dx: 110, Dy: 118}, {Dx: 80, Dy: 118}, {Dx: 110, Dy: 105}}, StrokePaint(0xFF606060, 2, ui.StrokeCapRound, ui.StrokeJoinMiter))
		}},
		{"colors, paint and tiny shapes", 20, 2, 0, func(c *ui.Canvas) {
			c.Save()
			c.ClipRect(r10)
			c.DrawColor(ui.NewColor(0x80FF0000), ui.BlendModeSrcOver)
			c.Restore()
			c.Save()
			c.ClipRect(ui.NewRectFromLTWH(80, 10, 30.5, 40.5))
			c.DrawPaint(FillPaint(0x9900A000))
			c.Restore()
			c.DrawCircle(ui.Offset{Dx: 30, Dy: 90}, 0.5, FillPaint(0xFF000000))
			c.DrawCircle(ui.Offset{Dx: 50, Dy: 90}, 1.5, FillPaint(0xFF000000))
			c.DrawRect(ui.NewRectFromLTWH(70.25, 80.25, 0.5, 0.5), FillPaint(0xFF000000))
		}},
		// Without anti-aliasing a pixel is all or nothing; where the exact coverage is
		// close to half the two renderers may choose differently.
		{"no anti-aliasing", 20, 20, 8, func(c *ui.Canvas) {
			p := FillPaint(0xFF2040C0)
			p.SetIsAntiAlias(false)
			c.DrawCircle(ui.Offset{Dx: 40, Dy: 40}, 30, p)
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(60.4, 60.4, 50, 50), 12, 12), p)
			q := StrokePaint(0xFFC04020, 5, ui.StrokeCapRound, ui.StrokeJoinMiter)
			q.SetIsAntiAlias(false)
			c.DrawLine(ui.Offset{Dx: 10, Dy: 110}, ui.Offset{Dx: 50, Dy: 80}, q)
		}},
		{"superellipse corners", 30, 3, 0, func(c *ui.Canvas) {
			c.DrawRSuperellipse(ui.NewRSuperellipseFromRectXY(ui.NewRectFromLTWH(10, 10, 100, 100), 30, 30), FillPaint(0xFF006A6A))
			c.Translate(64, 64)
			c.Rotate(0.3)
			c.DrawRSuperellipse(ui.NewRSuperellipseFromRectAndCorners(ui.NewRectFromLTWH(-30, -20, 60, 40), ui.Corners{TopLeft: ui.Radius{X: 16, Y: 16}, BottomRight: ui.Radius{X: 8, Y: 12}}), FillPaint(0xAAFFC000))
		}},
	}
}
