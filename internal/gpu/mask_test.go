package gpu

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func solidPaint(argb uint32) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(argb))
	return p
}

func linePaint(argb uint32, width float64, cap ui.StrokeCap, join ui.StrokeJoin) *ui.Paint {
	p := solidPaint(argb)
	p.SetStyle(ui.PaintingStyleStroke)
	p.SetStrokeWidth(width)
	p.SetStrokeCap(cap)
	p.SetStrokeJoin(join)
	return p
}

func wave() *ui.Path {
	p := ui.NewPath()
	p.MoveTo(0, 0)
	p.CubicTo(10, 20, 20, -20, 30, 0)
	p.LineTo(30, 30)
	p.Close()
	return p
}

func TestMaskKeysAreStableAndDistinct(t *testing.T) {
	a, ok := ShapeOf(ui.OpDrawPath{Path: wave(), Paint: *solidPaint(0xFF000000)})
	b, _ := ShapeOf(ui.OpDrawPath{Path: wave(), Paint: *solidPaint(0xFFFF0000)})
	if !ok || a.Geom != b.Geom {
		t.Error("the same path should have the same geometry hash")
	}
	rect := ui.NewRectFromLTWH(0, 0, 10, 10)
	seen := map[uint64]string{}
	for name, op := range map[string]ui.DisplayOp{
		"path":   ui.OpDrawPath{Path: wave()},
		"rect":   ui.OpDrawRect{Rect: rect},
		"rrect":  ui.OpDrawRRect{RRect: ui.NewRRectFromRectXY(rect, 2, 2)},
		"super":  ui.OpDrawRSuperellipse{RSuperellipse: ui.NewRSuperellipseFromRectXY(rect, 2, 2)},
		"drrect": ui.OpDrawDRRect{Outer: ui.NewRRectFromRectXY(rect, 2, 2), Inner: ui.NewRRectFromRectXY(rect.Deflate(2), 1, 1)},
		"oval":   ui.OpDrawOval{Rect: rect},
		"circle": ui.OpDrawCircle{Radius: 5},
		"arc":    ui.OpDrawArc{Rect: rect, SweepAngle: 1},
		"arc2":   ui.OpDrawArc{Rect: rect, SweepAngle: 2},
		"arc3":   ui.OpDrawArc{Rect: rect, SweepAngle: 2, UseCenter: true},
		"line":   ui.OpDrawLine{P2: ui.Offset{Dx: 5, Dy: 5}},
		"points": ui.OpDrawPoints{Points: []ui.Offset{{Dx: 1, Dy: 1}}},
		"lines":  ui.OpDrawPoints{Mode: ui.PointModeLines, Points: []ui.Offset{{Dx: 1, Dy: 1}}},
	} {
		sh, ok := ShapeOf(op)
		if !ok {
			t.Fatalf("%s: unknown", name)
		}
		if other, dup := seen[sh.Geom]; dup {
			t.Errorf("%s and %s have the same geometry hash", name, other)
		}
		seen[sh.Geom] = name
	}
	if _, ok := ShapeOf(ui.OpSave{}); ok {
		t.Error("Save is not a shape")
	}
	// Strokes and anti-aliasing change the style, a color does not.
	fill, stroke, aliased := solidPaint(0xFF000000), linePaint(0xFF000000, 3, ui.StrokeCapButt, ui.StrokeJoinMiter), solidPaint(0xFF000000)
	aliased.SetIsAntiAlias(false)
	if StyleKey(fill, false) == StyleKey(stroke, true) || StyleKey(fill, false) == StyleKey(aliased, false) {
		t.Error("style keys of different coverage are equal")
	}
	if StyleKey(fill, false) != StyleKey(solidPaint(0xFF123456), false) {
		t.Error("a color changed the style key")
	}
	wide := linePaint(0xFF000000, 5, ui.StrokeCapButt, ui.StrokeJoinMiter)
	if StyleKey(stroke, true) == StyleKey(wide, true) {
		t.Error("stroke widths share a style key")
	}
}

func TestFloorDiv(t *testing.T) {
	for _, tt := range []struct{ a, b, want int }{{7, 4, 1}, {8, 4, 2}, {0, 4, 0}, {-1, 4, -1}, {-4, 4, -1}, {-5, 4, -2}, {255, 256, 0}, {256, 256, 1}} {
		if got := FloorDiv(tt.a, tt.b); got != tt.want {
			t.Errorf("FloorDiv(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
