package soft

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/ui"
)

func px(img *image.RGBA, x, y int) [4]uint8 {
	i := img.PixOffset(x, y)
	return [4]uint8{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}
}

// coverageSum adds up the alpha of the image, as a number of pixels.
func coverageSum(img *image.RGBA) float64 {
	var s float64
	for i := 3; i < len(img.Pix); i += 4 {
		s += float64(img.Pix[i]) / 255
	}
	return s
}

func fillOf(argb uint32) *ui.Paint { return paint(argb) }

func TestGoldenShapes(t *testing.T) {
	rect := ui.NewRectFromLTWH(10.5, 12, 40, 30)
	rrect := ui.NewRRectFromRectXY(rect, 12, 8)
	tests := []struct {
		name string
		draw func(c *ui.Canvas)
	}{
		{"rect-fill", func(c *ui.Canvas) { c.DrawRect(rect, fillOf(0xFF3060C0)) }},
		{"rect-stroke", func(c *ui.Canvas) { c.DrawRect(rect, stroked(0xFF3060C0, 6, ui.StrokeCapButt, ui.StrokeJoinMiter)) }},
		{"rect-stroke-round", func(c *ui.Canvas) { c.DrawRect(rect, stroked(0xFF3060C0, 6, ui.StrokeCapButt, ui.StrokeJoinRound)) }},
		{"rrect-fill", func(c *ui.Canvas) { c.DrawRRect(rrect, fillOf(0xFFC06030)) }},
		{"rrect-stroke", func(c *ui.Canvas) { c.DrawRRect(rrect, stroked(0xFFC06030, 4, ui.StrokeCapButt, ui.StrokeJoinMiter)) }},
		{"rrect-corners", func(c *ui.Canvas) {
			c.DrawRRect(ui.NewRRectFromRectAndCorners(rect, ui.Corners{
				TopLeft: ui.NewRadiusCircular(14), BottomRight: ui.NewRadiusCircular(14),
			}), fillOf(0xFF30A060))
		}},
		{"rsuperellipse-fill", func(c *ui.Canvas) {
			c.DrawRSuperellipse(ui.NewRSuperellipseFromRectXY(rect, 14, 14), fillOf(0xFF8040A0))
		}},
		{"oval-fill", func(c *ui.Canvas) { c.DrawOval(rect, fillOf(0xFF40A0A0)) }},
		{"oval-stroke", func(c *ui.Canvas) { c.DrawOval(rect, stroked(0xFF40A0A0, 5, ui.StrokeCapButt, ui.StrokeJoinMiter)) }},
		{"circle-fill", func(c *ui.Canvas) { c.DrawCircle(ui.Offset{Dx: 32, Dy: 32}, 20, fillOf(0xFFA04060)) }},
		{"circle-stroke", func(c *ui.Canvas) {
			c.DrawCircle(ui.Offset{Dx: 32, Dy: 32}, 20, stroked(0xFFA04060, 6, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}},
		{"drrect", func(c *ui.Canvas) {
			c.DrawDRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(6, 8, 52, 48), 14, 14),
				ui.NewRRectFromRectXY(ui.NewRectFromLTWH(16, 18, 32, 28), 6, 6), fillOf(0xFF606090))
		}},
		{"arc-pie", func(c *ui.Canvas) {
			c.DrawArc(ui.NewRectFromLTWH(8, 8, 48, 48), -0.4*math.Pi, 1.3*math.Pi, true, fillOf(0xFFB08020))
		}},
		{"arc-chord", func(c *ui.Canvas) {
			c.DrawArc(ui.NewRectFromLTWH(8, 8, 48, 48), -0.4*math.Pi, 1.3*math.Pi, false, fillOf(0xFFB08020))
		}},
		{"arc-open-stroke", func(c *ui.Canvas) {
			c.DrawArc(ui.NewRectFromLTWH(8, 8, 48, 48), -0.4*math.Pi, 1.3*math.Pi, false, stroked(0xFFB08020, 6, ui.StrokeCapRound, ui.StrokeJoinMiter))
		}},
		{"arc-pie-stroke", func(c *ui.Canvas) {
			c.DrawArc(ui.NewRectFromLTWH(8, 8, 48, 48), -0.4*math.Pi, 1.3*math.Pi, true, stroked(0xFFB08020, 5, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}},
		{"line-round", func(c *ui.Canvas) {
			c.DrawLine(ui.Offset{Dx: 10, Dy: 12}, ui.Offset{Dx: 54, Dy: 50}, stroked(0xFF204080, 10, ui.StrokeCapRound, ui.StrokeJoinMiter))
		}},
		{"line-square", func(c *ui.Canvas) {
			c.DrawLine(ui.Offset{Dx: 10, Dy: 12}, ui.Offset{Dx: 54, Dy: 50}, stroked(0xFF204080, 10, ui.StrokeCapSquare, ui.StrokeJoinMiter))
		}},
		{"points", func(c *ui.Canvas) {
			c.DrawPoints(ui.PointModePoints, []ui.Offset{{Dx: 12, Dy: 12}, {Dx: 32, Dy: 40}, {Dx: 52, Dy: 14}}, stroked(0xFF802060, 9, ui.StrokeCapRound, ui.StrokeJoinMiter))
		}},
		{"points-square", func(c *ui.Canvas) {
			c.DrawPoints(ui.PointModePoints, []ui.Offset{{Dx: 12, Dy: 12}, {Dx: 32, Dy: 40}, {Dx: 52, Dy: 14}}, stroked(0xFF802060, 9, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}},
		{"points-lines", func(c *ui.Canvas) {
			c.DrawPoints(ui.PointModeLines, []ui.Offset{{Dx: 8, Dy: 10}, {Dx: 56, Dy: 14}, {Dx: 8, Dy: 30}, {Dx: 56, Dy: 34}, {Dx: 30, Dy: 50}}, stroked(0xFF802060, 4, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}},
		{"points-polygon", func(c *ui.Canvas) {
			c.DrawPoints(ui.PointModePolygon, []ui.Offset{{Dx: 8, Dy: 50}, {Dx: 24, Dy: 10}, {Dx: 40, Dy: 50}, {Dx: 56, Dy: 10}}, stroked(0xFF802060, 5, ui.StrokeCapButt, ui.StrokeJoinRound))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := renderOf(t, 64, 64, func(c *ui.Canvas) { white(c); tt.draw(c) })
			goldens.Check(t, "shape-"+tt.name, img)
		})
	}
}

func TestShapeAreas(t *testing.T) {
	tests := []struct {
		name string
		draw func(c *ui.Canvas)
		want float64
	}{
		{"rect", func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(2.5, 3, 10, 8), paint(0xFF000000)) }, 80},
		{"rrect", func(c *ui.Canvas) {
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(2, 2, 20, 20), 5, 5), paint(0xFF000000))
		}, 400 - (4-math.Pi)*25},
		{"oval", func(c *ui.Canvas) { c.DrawOval(ui.NewRectFromLTWH(2, 2, 20, 10), paint(0xFF000000)) }, math.Pi * 10 * 5},
		{"circle", func(c *ui.Canvas) { c.DrawCircle(ui.Offset{Dx: 16, Dy: 16}, 10, paint(0xFF000000)) }, math.Pi * 100},
		{"drrect", func(c *ui.Canvas) {
			c.DrawDRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(2, 2, 20, 20), 0, 0), ui.NewRRectFromRectXY(ui.NewRectFromLTWH(6, 6, 12, 12), 0, 0), paint(0xFF000000))
		}, 400 - 144},
		{"pie", func(c *ui.Canvas) {
			c.DrawArc(ui.NewRectFromLTWH(2, 2, 20, 20), 0, math.Pi/2, true, paint(0xFF000000))
		}, math.Pi * 100 / 4},
		{"full-turn arc is the oval", func(c *ui.Canvas) {
			c.DrawArc(ui.NewRectFromLTWH(2, 2, 20, 20), 0.3, 7, false, paint(0xFF000000))
		}, math.Pi * 100},
		{"line", func(c *ui.Canvas) {
			c.DrawLine(ui.Offset{Dx: 2, Dy: 5}, ui.Offset{Dx: 22, Dy: 5}, stroked(0xFF000000, 4, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}, 80},
		{"fill style is ignored by a line", func(c *ui.Canvas) {
			c.DrawLine(ui.Offset{Dx: 2, Dy: 5}, ui.Offset{Dx: 22, Dy: 5}, paint(0xFF000000))
		}, 1 * 20}, // a hairline, one pixel wide
		{"point", func(c *ui.Canvas) {
			c.DrawPoints(ui.PointModePoints, []ui.Offset{{Dx: 10, Dy: 10}}, stroked(0xFF000000, 8, ui.StrokeCapRound, ui.StrokeJoinMiter))
		}, math.Pi * 16},
		{"square point", func(c *ui.Canvas) {
			c.DrawPoints(ui.PointModePoints, []ui.Offset{{Dx: 10, Dy: 10}}, stroked(0xFF000000, 8, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}, 64},
		{"odd line point is dropped", func(c *ui.Canvas) {
			c.DrawPoints(ui.PointModeLines, []ui.Offset{{Dx: 2, Dy: 5}, {Dx: 22, Dy: 5}, {Dx: 9, Dy: 9}}, stroked(0xFF000000, 2, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}, 40},
		{"no points", func(c *ui.Canvas) {
			c.DrawPoints(ui.PointModePoints, nil, stroked(0xFF000000, 2, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coverageSum(renderOf(t, 32, 32, tt.draw))
			if math.Abs(got-tt.want) > tt.want*0.01+0.5 {
				t.Errorf("area = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShapeStrokeIgnoresTransformSkewSign(t *testing.T) {
	// A mirrored transform must not change the area of a stroke.
	draw := func(sx float64) float64 {
		return coverageSum(renderOf(t, 40, 40, func(c *ui.Canvas) {
			c.Translate(20, 20)
			c.Scale(sx, 1)
			c.DrawCircle(ui.Offset{}, 10, stroked(0xFF000000, 4, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}))
	}
	if a, b := draw(1), draw(-1); math.Abs(a-b) > 0.5 {
		t.Errorf("area %v vs mirrored %v", a, b)
	}
}

func TestDrawPaintAndPicture(t *testing.T) {
	inner := ui.NewPictureRecorder()
	ic := ui.NewCanvas(inner)
	ic.Translate(100, 100) // must not leak out of DrawPicture
	ic.DrawRect(ui.NewRectFromLTWH(0, 0, 1, 1), paint(0xFF0000FF))
	pic := inner.EndRecording()

	img := renderOf(t, 8, 8, func(c *ui.Canvas) {
		c.Save()
		c.ClipRect(ui.NewRectFromLTWH(1, 1, 3, 3))
		c.DrawPaint(paint(0xFF00FF00)) // fills the clip only
		c.Restore()
		c.Translate(5, 5)
		c.DrawPicture(pic) // drawn at 105, 105: outside
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 1, 1), paint(0xFFFF0000))
	})
	if got := px(img, 2, 2); got != [4]uint8{0, 255, 0, 255} {
		t.Errorf("inside the clip = %v", got)
	}
	if got := px(img, 0, 0); got != [4]uint8{} {
		t.Errorf("outside the clip = %v", got)
	}
	if got := px(img, 5, 5); got != [4]uint8{255, 0, 0, 255} {
		t.Errorf("after DrawPicture the transform = %v", got)
	}
	if got := px(img, 4, 4); got != [4]uint8{} {
		t.Errorf("clip edge = %v, want transparent", got)
	}
}
