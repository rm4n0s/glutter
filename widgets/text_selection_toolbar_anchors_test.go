package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests cover TextSelectionToolbarAnchors
// (widgets/text_selection_toolbar_anchors.dart).

func TestTextSelectionToolbarAnchorsFromSelection(t *testing.T) {
	ts := gluttertest.New(t)
	pumpContainer(ts, NewAlign(NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey("box"), Width: foundation.Some(200.0), Height: foundation.Some(100.0)}),
		AlignOpts{Alignment: painting.AlignmentTopLeft}))
	box := ts.RenderObject(gluttertest.ByKey(foundation.NewValueKey("box"))).(rendering.Box)
	point := func(x, y float64) rendering.TextSelectionPoint {
		return rendering.TextSelectionPoint{Point: ui.Offset{Dx: x, Dy: y}}
	}
	tests := []struct {
		name      string
		endpoints []rendering.TextSelectionPoint
		wantRect  ui.Rect
		primary   ui.Offset
		secondary ui.Offset
	}{
		{"one line", []rendering.TextSelectionPoint{point(20, 30), point(120, 30)},
			ui.NewRectFromLTRB(20, 14, 120, 30), ui.Offset{Dx: 70, Dy: 14}, ui.Offset{Dx: 70, Dy: 30}},
		{"several lines", []rendering.TextSelectionPoint{point(20, 30), point(50, 70)},
			ui.NewRectFromLTRB(0, 14, 200, 70), ui.Offset{Dx: 100, Dy: 14}, ui.Offset{Dx: 100, Dy: 70}},
		{"clamped to the editing region", []rendering.TextSelectionPoint{point(20, 4), point(120, 4)},
			ui.NewRectFromLTRB(20, -12, 120, 4), ui.Offset{Dx: 70, Dy: 0}, ui.Offset{Dx: 70, Dy: 4}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := TextSelectionToolbarAnchorsSelectionRect(box, 16, 16, tc.endpoints); got != tc.wantRect {
				t.Errorf("selection rect = %v, want %v", got, tc.wantRect)
			}
			anchors := NewTextSelectionToolbarAnchorsFromSelection(box, 16, 16, tc.endpoints)
			if anchors.PrimaryAnchor != tc.primary {
				t.Errorf("primary = %v, want %v", anchors.PrimaryAnchor, tc.primary)
			}
			if s, ok := anchors.SecondaryAnchor.Get(); !ok || s != tc.secondary {
				t.Errorf("secondary = %v (%v), want %v", s, ok, tc.secondary)
			}
		})
	}
}
