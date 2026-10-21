package web

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
)

// TestCursorCSS: every system mouse cursor of Flutter maps to the CSS cursor Flutter web uses (the
// table of cursor.dart), and the arrow and the unspecified cursor are the default, for which the
// property is removed.
func TestCursorCSS(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{"alias", "alias"}, {"allScroll", "all-scroll"}, {"basic", "default"}, {"cell", "cell"},
		{"click", "pointer"}, {"contextMenu", "context-menu"}, {"copy", "copy"}, {"forbidden", "not-allowed"},
		{"grab", "grab"}, {"grabbing", "grabbing"}, {"help", "help"}, {"move", "move"}, {"none", "none"},
		{"noDrop", "no-drop"}, {"precise", "crosshair"}, {"progress", "progress"}, {"text", "text"},
		{"resizeColumn", "col-resize"}, {"resizeDown", "s-resize"}, {"resizeDownLeft", "sw-resize"},
		{"resizeDownRight", "se-resize"}, {"resizeLeft", "w-resize"}, {"resizeLeftRight", "ew-resize"},
		{"resizeRight", "e-resize"}, {"resizeRow", "row-resize"}, {"resizeUp", "n-resize"},
		{"resizeUpDown", "ns-resize"}, {"resizeUpLeft", "nw-resize"}, {"resizeUpRight", "ne-resize"},
		{"resizeUpLeftDownRight", "nwse-resize"}, {"resizeUpRightDownLeft", "nesw-resize"},
		{"verticalText", "vertical-text"}, {"wait", "wait"}, {"zoomIn", "zoom-in"}, {"zoomOut", "zoom-out"},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			c, ok := platform.MouseCursorFromKind(tt.kind)
			if !ok {
				t.Fatalf("%q is not a system mouse cursor", tt.kind)
			}
			if got := CursorCSS(c); got != tt.want {
				t.Errorf("CursorCSS(%v) = %q, want %q", c, got, tt.want)
			}
		})
	}
	if got := CursorCSS(platform.MouseCursorUnspecified); got != "default" {
		t.Errorf("the unspecified cursor = %q, want default", got)
	}
	if got := CursorCSS(platform.MouseCursor(999)); got != "default" {
		t.Errorf("an invalid cursor = %q, want default", got)
	}
}

// TestEveryCursorIsMapped: no system cursor of the framework is left out of the table, except the
// one Flutter web has no CSS cursor for (disappearing), which stays the default as in Flutter.
func TestEveryCursorIsMapped(t *testing.T) {
	for c := platform.MouseCursorNone; c.Valid(); c++ {
		if _, ok := cursorCSS[c]; !ok && c != platform.MouseCursorDisappearing {
			t.Errorf("%v has no CSS cursor", c)
		}
	}
	if got := CursorCSS(platform.MouseCursorDisappearing); got != "default" {
		t.Errorf("disappearing = %q, want default", got)
	}
}
