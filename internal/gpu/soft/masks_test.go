package soft

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// maskMatchesRender checks that the mask of an operation is the alpha that Render
// draws for it with an opaque black paint.
func maskMatchesRender(t *testing.T, op ui.DisplayOp, draw func(c *ui.Canvas), m *ui.Matrix4) {
	t.Helper()
	const w, h = 64, 48
	img := renderOf(t, w, h, draw)
	mask, ok := OpMask(op, m, image.Rect(0, 0, w, h))
	if !ok {
		t.Fatalf("OpMask does not know %T", op)
	}
	var worst float64
	var total float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			got := float64(mask.At(x, y))
			want := float64(img.Pix[img.PixOffset(x, y)+3]) / 255
			worst = math.Max(worst, math.Abs(got-want))
			total += got
		}
	}
	if worst > 1.0/255+1e-6 {
		t.Errorf("%T: the mask differs from the rendered alpha by %v", op, worst)
	}
	if total == 0 {
		t.Errorf("%T: the mask covers nothing", op)
	}
}

func TestOpMaskMatchesRender(t *testing.T) {
	rect := ui.NewRectFromLTWH(8.3, 6.6, 40.4, 30.2)
	star := ui.NewPath()
	star.MoveTo(32, 4)
	star.LineTo(44, 40)
	star.LineTo(10, 18)
	star.LineTo(54, 18)
	star.LineTo(20, 40)
	star.Close()
	evenOdd := ui.NewPath()
	evenOdd.AddPath(star, ui.Offset{})
	evenOdd.SetFillType(ui.PathFillTypeEvenOdd)
	black := func() *ui.Paint { return paint(0xFF000000) }
	thick := stroked(0xFF000000, 5, ui.StrokeCapRound, ui.StrokeJoinRound)
	hair := stroked(0xFF000000, 0, ui.StrokeCapButt, ui.StrokeJoinMiter)
	noAA := black()
	noAA.SetIsAntiAlias(false)
	rot := ui.NewMatrix4Identity()
	rot.Translate(30, 20, 0)
	rot.RotateZ(0.4)
	id := ui.NewMatrix4Identity()
	tests := []struct {
		name string
		op   ui.DisplayOp
		draw func(c *ui.Canvas)
		m    *ui.Matrix4
	}{
		{"path nonzero", ui.OpDrawPath{Path: star, Paint: *black()}, func(c *ui.Canvas) { c.DrawPath(star, black()) }, id},
		{"path even-odd", ui.OpDrawPath{Path: evenOdd, Paint: *black()}, func(c *ui.Canvas) { c.DrawPath(evenOdd, black()) }, id},
		{"stroked path", ui.OpDrawPath{Path: star, Paint: *thick}, func(c *ui.Canvas) { c.DrawPath(star, thick) }, id},
		{"hairline", ui.OpDrawPath{Path: star, Paint: *hair}, func(c *ui.Canvas) { c.DrawPath(star, hair) }, id},
		{"no anti-aliasing", ui.OpDrawPath{Path: star, Paint: *noAA}, func(c *ui.Canvas) { c.DrawPath(star, noAA) }, id},
		{"rect", ui.OpDrawRect{Rect: rect, Paint: *black()}, func(c *ui.Canvas) { c.DrawRect(rect, black()) }, id},
		{"rrect", ui.OpDrawRRect{RRect: ui.NewRRectFromRectXY(rect, 7, 5), Paint: *black()}, func(c *ui.Canvas) { c.DrawRRect(ui.NewRRectFromRectXY(rect, 7, 5), black()) }, id},
		{"superellipse", ui.OpDrawRSuperellipse{RSuperellipse: ui.NewRSuperellipseFromRectXY(rect, 8, 8), Paint: *black()},
			func(c *ui.Canvas) { c.DrawRSuperellipse(ui.NewRSuperellipseFromRectXY(rect, 8, 8), black()) }, id},
		{"drrect", ui.OpDrawDRRect{Outer: ui.NewRRectFromRectXY(rect, 7, 7), Inner: ui.NewRRectFromRectXY(rect.Deflate(8), 3, 3), Paint: *black()},
			func(c *ui.Canvas) {
				c.DrawDRRect(ui.NewRRectFromRectXY(rect, 7, 7), ui.NewRRectFromRectXY(rect.Deflate(8), 3, 3), black())
			}, id},
		{"oval", ui.OpDrawOval{Rect: rect, Paint: *black()}, func(c *ui.Canvas) { c.DrawOval(rect, black()) }, id},
		{"circle", ui.OpDrawCircle{Center: ui.Offset{Dx: 30, Dy: 24}, Radius: 15.5, Paint: *thick}, func(c *ui.Canvas) { c.DrawCircle(ui.Offset{Dx: 30, Dy: 24}, 15.5, thick) }, id},
		{"arc", ui.OpDrawArc{Rect: rect, StartAngle: 0.3, SweepAngle: 4, UseCenter: true, Paint: *black()},
			func(c *ui.Canvas) { c.DrawArc(rect, 0.3, 4, true, black()) }, id},
		{"line", ui.OpDrawLine{P1: ui.Offset{Dx: 5, Dy: 5}, P2: ui.Offset{Dx: 50, Dy: 40}, Paint: *thick},
			func(c *ui.Canvas) { c.DrawLine(ui.Offset{Dx: 5, Dy: 5}, ui.Offset{Dx: 50, Dy: 40}, thick) }, id},
		{"points", ui.OpDrawPoints{Mode: ui.PointModePoints, Points: []ui.Offset{{Dx: 10, Dy: 10}, {Dx: 30, Dy: 30}}, Paint: *thick},
			func(c *ui.Canvas) {
				c.DrawPoints(ui.PointModePoints, []ui.Offset{{Dx: 10, Dy: 10}, {Dx: 30, Dy: 30}}, thick)
			}, id},
		{"polygon", ui.OpDrawPoints{Mode: ui.PointModePolygon, Points: []ui.Offset{{Dx: 10, Dy: 10}, {Dx: 30, Dy: 40}, {Dx: 50, Dy: 10}}, Paint: *thick},
			func(c *ui.Canvas) {
				c.DrawPoints(ui.PointModePolygon, []ui.Offset{{Dx: 10, Dy: 10}, {Dx: 30, Dy: 40}, {Dx: 50, Dy: 10}}, thick)
			}, id},
		{"transformed rrect", ui.OpDrawRRect{RRect: ui.NewRRectFromRectXY(ui.NewRectFromLTWH(-15, -10, 30, 20), 6, 6), Paint: *black()},
			func(c *ui.Canvas) {
				c.Translate(30, 20)
				c.Rotate(0.4)
				c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(-15, -10, 30, 20), 6, 6), black())
			}, rot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { maskMatchesRender(t, tt.op, tt.draw, tt.m) })
	}
}

func TestOpMaskOtherOperations(t *testing.T) {
	for _, op := range []ui.DisplayOp{ui.OpSave{}, ui.OpDrawColor{}, ui.OpDrawImage{}, ui.OpDrawShadow{}} {
		if m, ok := OpMask(op, ui.NewMatrix4Identity(), image.Rect(0, 0, 10, 10)); ok || m != nil {
			t.Errorf("%T: OpMask = %v, %v, want false", op, m, ok)
		}
	}
	// Nothing to cover: no mask, but the operation is known.
	m, ok := OpMask(ui.OpDrawPoints{Mode: ui.PointModePoints, Paint: *paint(0xFF000000)}, ui.NewMatrix4Identity(), image.Rect(0, 0, 10, 10))
	if !ok || m != nil {
		t.Errorf("empty points: %v, %v", m, ok)
	}
	off := ui.OpDrawRect{Rect: ui.NewRectFromLTWH(100, 100, 5, 5), Paint: *paint(0xFF000000)}
	if m, ok := OpMask(off, ui.NewMatrix4Identity(), image.Rect(0, 0, 10, 10)); !ok || m != nil {
		t.Errorf("a rect outside the area: %v, %v", m, ok)
	}
}

func TestClipMask(t *testing.T) {
	circle := ui.NewPath()
	circle.AddOval(ui.NewRectFromLTWH(10.5, 10.5, 30, 30))
	area := image.Rect(0, 0, 64, 64)
	id := ui.NewMatrix4Identity()
	soft := ClipMask(circle, id, true, area)
	hard := ClipMask(circle, id, false, area)
	if soft == nil || hard == nil {
		t.Fatal("no mask")
	}
	if got := soft.At(25, 25); got != 1 {
		t.Errorf("the center is covered %v", got)
	}
	partial := false
	for _, c := range soft.Cov {
		if c > 0 && c < 1 {
			partial = true
		}
	}
	if !partial {
		t.Error("an anti-aliased clip has partial coverage on its edge")
	}
	for _, c := range hard.Cov {
		if c != 0 && c != 1 {
			t.Fatalf("a hard clip has partial coverage %v", c)
		}
	}
	// The hard clip keeps the pixels whose centers are inside.
	if hard.At(10, 25) != 0 || hard.At(11, 25) != 1 {
		t.Errorf("hard edge: %v %v", hard.At(10, 25), hard.At(11, 25))
	}
	// The transform applies.
	moved := ui.NewMatrix4TranslationValues(20, 0, 0)
	if m := ClipMask(circle, moved, true, area); m == nil || m.At(45, 25) != 1 || m.At(25, 25) == 1 {
		t.Error("the clip did not move with the transform")
	}
	if ClipMask(circle, id, true, image.Rect(50, 50, 60, 60)) != nil {
		t.Error("a clip outside the area covers nothing")
	}
}
