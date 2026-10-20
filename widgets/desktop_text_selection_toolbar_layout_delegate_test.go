package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests cover DesktopTextSelectionToolbarLayoutDelegate
// (widgets/desktop_text_selection_toolbar_layout_delegate.dart).

func TestDesktopTextSelectionToolbarLayoutDelegatePosition(t *testing.T) {
	box, child := ui.Size{Width: 400, Height: 300}, ui.Size{Width: 100, Height: 50}
	tests := []struct {
		name   string
		anchor ui.Offset
		want   ui.Offset
	}{
		{"fits", ui.Offset{Dx: 10, Dy: 20}, ui.Offset{Dx: 10, Dy: 20}},
		{"fits exactly", ui.Offset{Dx: 300, Dy: 250}, ui.Offset{Dx: 300, Dy: 250}},
		{"too far right", ui.Offset{Dx: 380, Dy: 20}, ui.Offset{Dx: 300, Dy: 20}},
		{"too far down", ui.Offset{Dx: 10, Dy: 290}, ui.Offset{Dx: 10, Dy: 250}},
		{"too far in both", ui.Offset{Dx: 399, Dy: 299}, ui.Offset{Dx: 300, Dy: 250}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDesktopTextSelectionToolbarLayoutDelegate(tc.anchor)
			if got := d.GetPositionForChild(box, child); got != tc.want {
				t.Errorf("position = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDesktopTextSelectionToolbarLayoutDelegateConstraintsAndRelayout(t *testing.T) {
	d := NewDesktopTextSelectionToolbarLayoutDelegate(ui.Offset{Dx: 1, Dy: 2})
	tight := rendering.NewBoxConstraintsTight(ui.Size{Width: 50, Height: 60})
	if got := d.GetConstraintsForChild(tight); got.MinWidth != 0 || got.MinHeight != 0 || got.MaxWidth != 50 || got.MaxHeight != 60 {
		t.Errorf("constraints = %v, want them loosened", got)
	}
	if d.Anchor() != (ui.Offset{Dx: 1, Dy: 2}) {
		t.Errorf("anchor = %v", d.Anchor())
	}
	if d.ShouldRelayout(NewDesktopTextSelectionToolbarLayoutDelegate(ui.Offset{Dx: 1, Dy: 2})) {
		t.Error("the same anchor needs no relayout")
	}
	if !d.ShouldRelayout(NewDesktopTextSelectionToolbarLayoutDelegate(ui.Offset{Dx: 3, Dy: 2})) {
		t.Error("another anchor needs a relayout")
	}
}
