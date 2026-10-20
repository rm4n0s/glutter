//go:build !glutter_release && !glutter_profile

package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func TestParagraphSelectionFailures(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		do   func(t *testing.T)
	}{
		{"edge update by a line", "SelectionGranularityUnsupported", func(t *testing.T) {
			p, registrar := newSelectableParagraph("how are you")
			layout(t, newHarness(t), p)
			registrar.selectables[0].DispatchSelectionEvent(rendering.NewSelectionEdgeUpdateEventForEnd(ui.Offset{Dx: 5, Dy: 5}, rendering.TextGranularityLine))
		}},
		{"extend by a line without a selection", "SelectionNotStarted", func(t *testing.T) {
			p, registrar := newSelectableParagraph("how are you\nI am fine")
			layout(t, newHarness(t), p)
			registrar.selectables[0].DispatchSelectionEvent(rendering.DirectionallyExtendSelectionEvent{
				IsEnd: true, Dx: 5, Direction: rendering.SelectionExtendDirectionNextLine})
		}},
		{"a baseline that is not a number", "SelectionBaselineNaN", func(t *testing.T) {
			p, registrar := newSelectableParagraph("how are you")
			layout(t, newHarness(t), p)
			registrar.selectables[0].DispatchSelectionEvent(rendering.DirectionallyExtendSelectionEvent{
				IsEnd: true, Dx: math.NaN(), Direction: rendering.SelectionExtendDirectionForward})
		}},
		{"handles on a detached paragraph", "SelectionHandlesOnDetached", func(t *testing.T) {
			_, registrar := newSelectableParagraph("how are you")
			if len(registrar.selectables) == 0 {
				t.Fatal("the paragraph did not register a selectable before it was attached")
			}
			registrar.selectables[0].PushHandleLayers(rendering.NewLayerLink(), nil)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requireTag(t, tc.tag, func() { tc.do(t) })
		})
	}
}
