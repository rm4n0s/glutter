//go:build !glutter_release && !glutter_profile

package painting

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestTextPainterErrors(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		call func(t *testing.T)
	}{
		{"layout without text", "TextPainterTextNotSet", func(t *testing.T) {
			NewTextPainter(TextPainterOpts{TextEngine: bundledEngine(t), TextDirection: ui.TextDirectionLTR}).Layout(0, 100)
		}},
		{"layout without direction", "TextPainterDirectionNotSet", func(t *testing.T) {
			NewTextPainter(TextPainterOpts{TextEngine: bundledEngine(t), Text: ts("a")}).Layout(0, 100)
		}},
		{"paint before layout", "TextPainterPaintBeforeLayout", func(t *testing.T) {
			newPainter(t, "a").Paint(ui.NewCanvas(ui.NewPictureRecorder()), ui.Offset{})
		}},
		{"width before layout", "TextPainterLayoutNotAvailable", func(t *testing.T) { newPainter(t, "a").Width() }},
		{"width after the layout was invalidated", "TextPainterLayoutNotAvailable", func(t *testing.T) {
			p := newPainter(t, "a")
			p.Layout(0, 100)
			p.SetTextAlign(ui.TextAlignRight)
			p.Width()
		}},
		{"width after the basis changed", "TextPainterNeedsRelayout", func(t *testing.T) {
			p := newPainter(t, "a")
			p.Layout(0, 100)
			p.SetTextWidthBasis(TextWidthBasisLongestLine)
			p.Width()
		}},
		{"zero max lines", "TextPainterMaxLinesInvalid", func(t *testing.T) {
			NewTextPainter(TextPainterOpts{MaxLines: foundation.Some(0)})
		}},
		{"set zero max lines", "TextPainterMaxLinesInvalid", func(t *testing.T) { newPainter(t, "a").SetMaxLines(foundation.Some(0)) }},
		{"NaN width", "TextPainterConstraintsNaN", func(t *testing.T) { newPainter(t, "a").Layout(math.NaN(), 100) }},
		{"disposed twice", "TextPainterDisposed", func(t *testing.T) {
			p := newPainter(t, "a")
			p.Dispose()
			p.Dispose()
		}},
		{"used after dispose", "TextPainterDisposed", func(t *testing.T) {
			p := newPainter(t, "a")
			p.Layout(0, 100)
			p.Dispose()
			p.Width()
		}},
		{"placeholder count", "TextPainterPlaceholderCountMismatch", func(t *testing.T) {
			p := newPainter(t, "a")
			p.SetPlaceholderDimensions([]PlaceholderDimensions{PlaceholderDimensionsEmpty()})
		}},
		{"selection invalid", "TextPainterSelectionInvalid", func(t *testing.T) {
			p := newPainter(t, "a")
			p.Layout(0, 100)
			p.GetBoxesForSelection(ui.NewTextRange(-1, -1))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTaggedPanic(t, tt.tag, func() { tt.call(t) }) })
	}
}
