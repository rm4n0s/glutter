package gpu

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// opsWithPaint is one operation of every kind that has a paint, built with the paint.
func opsWithPaint(p ui.Paint) map[string]ui.DisplayOp {
	path := ui.NewPath()
	path.AddRect(ui.NewRectFromLTWH(0, 0, 4, 4))
	r := ui.NewRectFromLTWH(0, 0, 4, 4)
	return map[string]ui.DisplayOp{
		"path":         ui.OpDrawPath{Path: path, Paint: p},
		"rect":         ui.OpDrawRect{Rect: r, Paint: p},
		"rrect":        ui.OpDrawRRect{RRect: ui.NewRRectFromRectXY(r, 1, 1), Paint: p},
		"superellipse": ui.OpDrawRSuperellipse{Paint: p},
		"drrect":       ui.OpDrawDRRect{Paint: p},
		"oval":         ui.OpDrawOval{Rect: r, Paint: p},
		"circle":       ui.OpDrawCircle{Radius: 2, Paint: p},
		"arc":          ui.OpDrawArc{Rect: r, Paint: p},
		"line":         ui.OpDrawLine{Paint: p},
		"points":       ui.OpDrawPoints{Points: []ui.Offset{{}}, Paint: p},
		"image":        ui.OpDrawImage{Paint: p},
		"image rect":   ui.OpDrawImageRect{Paint: p},
		"image nine":   ui.OpDrawImageNine{Paint: p},
		"atlas":        ui.OpDrawAtlas{Paint: p},
		"vertices":     ui.OpDrawVertices{Paint: p},
		"paint":        ui.OpDrawPaint{Paint: p},
	}
}

func TestCopyPaintOfAgreesWithPaintOf(t *testing.T) {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(0xFF336699))
	p.SetStrokeWidth(3)
	for name, op := range opsWithPaint(*p) {
		t.Run(name, func(t *testing.T) {
			var got ui.Paint
			if !CopyPaintOf(op, &got) {
				t.Fatal("no paint")
			}
			if want := PaintOf(op); got != *want {
				t.Errorf("paint = %+v, want %+v", got, *want)
			}
		})
	}
	for name, op := range map[string]ui.DisplayOp{"save": ui.OpSave{}, "color": ui.OpDrawColor{}, "shadow": ui.OpDrawShadow{}, "nil": nil} {
		t.Run("without a paint/"+name, func(t *testing.T) {
			var got ui.Paint
			if CopyPaintOf(op, &got) || anyFilter(op) {
				t.Error("an operation without a paint has one")
			}
		})
	}
}

// anyFilter says whether PaintFilters finds any filter.
func anyFilter(op ui.DisplayOp) bool { i, m := PaintFilters(op); return i || m }

func TestPaintFilters(t *testing.T) {
	plain := ui.NewPaint()
	blurred := ui.NewPaint()
	blurred.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 3))
	filtered := ui.NewPaint()
	filtered.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 2}))
	tests := []struct {
		name        string
		paint       *ui.Paint
		image, mask bool
	}{{"a plain paint", plain, false, false}, {"a mask filter", blurred, false, true}, {"an image filter", filtered, true, false}}
	for _, tt := range tests {
		for kind, op := range opsWithPaint(*tt.paint) {
			if i, m := PaintFilters(op); i != tt.image || m != tt.mask {
				t.Errorf("%s, %s: image %v mask %v, want %v %v", tt.name, kind, i, m, tt.image, tt.mask)
			}
		}
	}
}

func TestShapeOfIntoKeepsThePaintInItsDestination(t *testing.T) {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(0xFFAA5500))
	p.SetStyle(ui.PaintingStyleStroke)
	p.SetStrokeWidth(2)
	for kind, op := range opsWithPaint(*p) {
		t.Run(kind, func(t *testing.T) {
			var dst ui.Paint
			got, ok := ShapeOfInto(op, &dst)
			want, wantOK := ShapeOf(op)
			if ok != wantOK {
				t.Fatalf("ok = %v, want %v", ok, wantOK)
			}
			if !ok {
				return
			}
			if got.Geom != want.Geom || got.Bounds != want.Bounds || got.Stroke != want.Stroke || got.Paint != &dst || dst != *want.Paint {
				t.Errorf("shape = %+v with the paint %p, want %+v with the paint in dst %p", got, got.Paint, want, &dst)
			}
		})
	}
}

func TestShapeOfIntoMakesNoAllocationForAGlyph(t *testing.T) {
	path := ui.NewPath()
	path.AddRect(ui.NewRectFromLTWH(0, 0, 4, 6))
	var op ui.DisplayOp = ui.OpDrawPath{Path: path, Paint: *ui.NewPaint()}
	var dst ui.Paint
	if n := testing.AllocsPerRun(50, func() {
		if _, ok := ShapeOfInto(op, &dst); !ok {
			t.Fatal("not a shape")
		}
		CopyPaintOf(op, &dst)
		PaintFilters(op)
	}); n != 0 {
		t.Errorf("%v allocations for the shape, the paint and the filters of a glyph, want none", n)
	}
}
