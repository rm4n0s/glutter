//go:build !glutter_release && !glutter_profile

package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestImageAndSpanDebugPanics(t *testing.T) {
	pt := newPaintingTester(t)
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"resize without a size", "ResizeImageNoSize", func() { NewResizeImage(NewMemoryImage(nil)) }},
		{"accumulator", "AccumulatorNegativeIncrement", func() { NewAccumulator(0).Increment(-1) }},
		{"placeholder semantics", "InlineSpanSemanticsPlaceholderInvalid", func() {
			NewInlineSpanSemanticsInformation("x", InlineSpanSemanticsInformation{IsPlaceholder: true})
		}},
		{"label without text", "TextSpanLabelWithoutText", func() { NewTextSpan(TextSpanOpts{SemanticsLabel: foundation.Some("l")}) }},
		{"placeholder span validity", "PlaceholderSpanNotImplemented", func() { NewPlaceholderSpanBase().DebugAssertIsValid() }},
		{"zig zag", "PaintZigZagInvalid", func() {
			PaintZigZag(ui.NewCanvas(ui.NewPictureRecorder()), ui.NewPaint(), ui.Offset{}, ui.Offset{Dx: 10}, 0, 2)
		}},
		{"decoration image without a text direction", "DecorationImageNoTextDirection", func() {
			p := NewDecorationImage(NewMemoryImage(nil), DecorationImageOpts{MatchTextDirection: true}).CreatePainter(func() {})
			p.Paint(ui.NewCanvas(ui.NewPictureRecorder()), lrtb(0, 0, 4, 4), nil, ImageConfiguration{Binding: pt.PaintingBinding})
		}},
		{"decoration image without a binding", "ImageConfigurationNoBinding", func() {
			p := NewDecorationImage(NewMemoryImage(nil)).CreatePainter(func() {})
			p.Paint(ui.NewCanvas(ui.NewPictureRecorder()), lrtb(0, 0, 4, 4), nil, ImageConfigurationEmpty)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTaggedPanic(t, tt.tag, tt.call) })
	}
}
