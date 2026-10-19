//go:build !glutter_release && !glutter_profile

package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// notABorder is a BoxBorder that BoxBorderLerp doesn't know.
type notABorder struct{ Border }

func TestBoxBorderDebugPanics(t *testing.T) {
	rect := ui.NewRectFromLTWH(10, 20, 30, 40)
	canvas := func() *ui.Canvas { return ui.NewCanvas(ui.NewPictureRecorder()) }
	col := func(c uint32, w float64) BorderSide {
		return side(BorderSideOpts{Color: foundation.Some(ui.NewColor(c)), Width: foundation.Some(w)})
	}
	radius := foundation.Some(NewBorderRadiusCircular(5))
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"uniform circle with radius", "BoxBorderCircleRadius", func() {
			NewBorderAll().PaintBox(canvas(), rect, BoxPaintOpts{Shape: BoxShapeCircle, BorderRadius: radius})
		}},
		{"non-uniform circle with radius", "BoxBorderCircleRadius", func() {
			PaintNonUniformBorder(canvas(), rect, NonUniformBorderOpts{
				Shape: BoxShapeCircle, BorderRadius: foundation.Some[BorderRadiusGeometry](NewBorderRadiusCircular(5)),
				Top: col(0xFF000000, 1), Right: col(0xFF000000, 2), Bottom: col(0xFF000000, 1), Left: col(0xFF000000, 2), Color: ui.NewColor(0xFF000000),
			})
		}},
		{"hairline with radius", "BoxBorderHairlineRadius", func() {
			Border{Top: col(0xFF000000, 1), Left: side(BorderSideOpts{Width: foundation.Some(0.0)})}.PaintBox(canvas(), rect, BoxPaintOpts{BorderRadius: radius})
		}},
		{"different colors with radius", "BoxBorderNonUniformRadius", func() {
			Border{Top: col(0xFFFF0000, 1), Bottom: col(0xFF0000FF, 1)}.PaintBox(canvas(), rect, BoxPaintOpts{BorderRadius: radius})
		}},
		{"directional with radius", "BoxBorderNonUniformRadius", func() {
			BorderDirectional{Top: col(0xFFFF0000, 1), Bottom: col(0xFF0000FF, 1)}.PaintBox(canvas(), rect,
				BoxPaintOpts{BorderRadius: radius, TextDirection: ui.TextDirectionLTR})
		}},
		{"different colors as circle", "BoxBorderNonUniformShape", func() {
			Border{Top: col(0xFFFF0000, 1), Bottom: col(0xFF0000FF, 1)}.PaintBox(canvas(), rect, BoxPaintOpts{Shape: BoxShapeCircle})
		}},
		{"stroke align", "BoxBorderStrokeAlignNonUniform", func() {
			Border{
				Left:  side(BorderSideOpts{Color: foundation.Some(ui.NewColor(0xff000001)), StrokeAlign: foundation.Some(StrokeAlignCenter)}),
				Right: side(BorderSideOpts{Color: foundation.Some(ui.NewColor(0xff000002)), StrokeAlign: foundation.Some(StrokeAlignOutside)}),
			}.Paint(canvas(), rect, ui.TextDirectionLTR)
		}},
		{"lerp invalid", "BoxBorderLerpInvalid", func() { BoxBorderLerp(notABorder{}, Border{}, 0.5) }},
		{"inner path", "TextDirectionRequired", func() { Border{}.GetInnerPath(rect, ui.TextDirectionUnspecified) }},
		{"outer path", "TextDirectionRequired", func() { Border{}.GetOuterPath(rect, ui.TextDirectionUnspecified) }},
		{"directional paint", "TextDirectionRequired", func() {
			BorderDirectional{Start: col(0xFFFF0000, 1), End: col(0xFF0000FF, 2)}.Paint(canvas(), rect, ui.TextDirectionUnspecified)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireTaggedPanic(t, tt.tag, tt.call)
		})
	}
}

func TestBorderMergeCannotMerge(t *testing.T) {
	requireTaggedPanic(t, "BorderSideCannotMerge", func() { BorderMerge(Border{Left: magenta3}, Border{Left: yellow2}) })
	requireTaggedPanic(t, "BorderSideCannotMerge", func() {
		BorderDirectionalMerge(BorderDirectional{Start: magenta3}, BorderDirectional{Start: yellow2})
	})
}

func TestShapeBorderDefaultsPaintInterior(t *testing.T) {
	requireTaggedPanic(t, "ShapeBorderNoPaintInterior", func() {
		ShapeBorderDefaults{}.PaintInterior(nil, ui.Rect{}, nil, ui.TextDirectionLTR)
	})
}
