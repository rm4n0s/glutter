package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// The benchmarks of the render tree alone: layout, paint and compositing of about 500
// render objects, the part of SPEC §15's frame that the widgets do not touch. Run them
// with
//
//	go test ./rendering -run xxx -bench . -benchmem

// benchTree is a vertical flex of 40 rows, each a padded decorated box with a flex of
// six colored boxes: 40 x (1 + 1 + 1 + 6 x 1) = 360 render objects and more below.
func benchTree() (*renderhost.Stack, []*rendering.RenderDecoratedBox) {
	s := renderhost.New(renderhost.Opts{Size: ui.Size{Width: 1536, Height: 864}, DevicePixelRatio: 1.25})
	s.Host.DiscardScenes = true
	var boxes []*rendering.RenderDecoratedBox
	deco := func(argb uint32) *rendering.RenderDecoratedBox {
		b := rendering.NewRenderDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color: foundation.Some(ui.NewColor(argb)), BorderRadius: painting.NewBorderRadiusCircular(6)}), nil)
		boxes = append(boxes, b)
		return b
	}
	var rows []rendering.Box
	for i := 0; i < 40; i++ {
		var cells []rendering.Box
		for j := 0; j < 6; j++ {
			cells = append(cells, rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(40.0), Height: foundation.Some(10.0)}), deco(0xFF6750A4+uint32(i*6+j))))
		}
		row := rendering.NewRenderFlex(cells, rendering.RenderFlexOpts{Direction: painting.AxisHorizontal, TextDirection: ui.TextDirectionLTR})
		rows = append(rows, rendering.NewRenderPadding(painting.NewEdgeInsetsAll(2), rendering.NewRenderDecoratedBox(
			painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFFF3EDF7))}), row)))
	}
	root := rendering.NewRenderFlex(rows, rendering.RenderFlexOpts{Direction: painting.AxisVertical, CrossAxisAlignment: rendering.CrossAxisAlignmentStart, TextDirection: ui.TextDirectionLTR})
	s.SetRoot(root)
	s.Pump()
	return s, boxes
}

func BenchmarkLayoutFrame(b *testing.B) {
	s, _ := benchTree()
	root := s.RenderView.Child()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.MarkNeedsLayout()
		s.Pump()
	}
}

func BenchmarkPaintFrame(b *testing.B) {
	s, boxes := benchTree()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		boxes[0].MarkNeedsPaint()
		s.Pump()
	}
}

// BenchmarkIdleFrame is a Pump with nothing marked: no frame runs.
func BenchmarkIdleFrame(b *testing.B) {
	s, _ := benchTree()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Pump()
	}
}
