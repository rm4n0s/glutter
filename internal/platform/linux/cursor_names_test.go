//go:build linux

package linux

import (
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
)

func TestCursorNames(t *testing.T) {
	// Every cursor but none has names, and the first one is the CSS name.
	for c := platform.MouseCursorUnspecified; c.Valid(); c++ {
		names := CursorNames(c)
		if c == platform.MouseCursorNone {
			if names != nil {
				t.Errorf("%s has names %v, want none", c, names)
			}
			continue
		}
		if len(names) == 0 {
			t.Errorf("%s has no names", c)
		}
	}
	tests := []struct {
		cursor platform.MouseCursor
		first  string
	}{
		{platform.MouseCursorUnspecified, "default"},
		{platform.MouseCursorBasic, "default"},
		{platform.MouseCursorClick, "pointer"},
		{platform.MouseCursorText, "text"},
		{platform.MouseCursorGrab, "grab"},
		{platform.MouseCursorGrabbing, "grabbing"},
		{platform.MouseCursorResizeLeftRight, "ew-resize"},
		{platform.MouseCursorResizeDownRight, "se-resize"},
		{platform.MouseCursorWait, "wait"},
		{platform.MouseCursorForbidden, "not-allowed"},
	}
	for _, tt := range tests {
		t.Run(tt.cursor.String(), func(t *testing.T) {
			if names := CursorNames(tt.cursor); len(names) == 0 || names[0] != tt.first {
				t.Errorf("CursorNames = %v, want %s first", names, tt.first)
			}
		})
	}
	if CursorNames(platform.MouseCursor(-5)) != nil {
		t.Error("an invalid cursor has names")
	}
	// The result is a copy.
	names := CursorNames(platform.MouseCursorBasic)
	names[0] = "changed"
	if CursorNames(platform.MouseCursorBasic)[0] != "default" {
		t.Error("CursorNames returned the table itself")
	}
}

// TestEveryCursorHasItsNames lists, for every cursor, the CSS name that comes first
// (the one Flutter's GTK embedder asks for) and the old X name that a theme without
// CSS names has.
func TestEveryCursorHasItsNames(t *testing.T) {
	tests := []struct {
		cursor      platform.MouseCursor
		css, legacy string
	}{
		{platform.MouseCursorUnspecified, "default", "left_ptr"},
		{platform.MouseCursorBasic, "default", "left_ptr"},
		{platform.MouseCursorClick, "pointer", "hand2"},
		{platform.MouseCursorForbidden, "not-allowed", "crossed_circle"},
		{platform.MouseCursorWait, "wait", "watch"},
		{platform.MouseCursorProgress, "progress", "left_ptr_watch"},
		{platform.MouseCursorContextMenu, "context-menu", "context-menu"},
		{platform.MouseCursorHelp, "help", "question_arrow"},
		{platform.MouseCursorText, "text", "xterm"},
		{platform.MouseCursorVerticalText, "vertical-text", "vertical-text"},
		{platform.MouseCursorCell, "cell", "plus"},
		{platform.MouseCursorPrecise, "crosshair", "cross"},
		{platform.MouseCursorMove, "move", "fleur"},
		{platform.MouseCursorGrab, "grab", "openhand"},
		{platform.MouseCursorGrabbing, "grabbing", "closedhand"},
		{platform.MouseCursorNoDrop, "no-drop", "dnd-no-drop"},
		{platform.MouseCursorAlias, "alias", "dnd-link"},
		{platform.MouseCursorCopy, "copy", "dnd-copy"},
		{platform.MouseCursorDisappearing, "default", "left_ptr"},
		{platform.MouseCursorAllScroll, "all-scroll", "fleur"},
		{platform.MouseCursorResizeLeftRight, "ew-resize", "sb_h_double_arrow"},
		{platform.MouseCursorResizeUpDown, "ns-resize", "sb_v_double_arrow"},
		{platform.MouseCursorResizeUpLeftDownRight, "nwse-resize", "bd_double_arrow"},
		{platform.MouseCursorResizeUpRightDownLeft, "nesw-resize", "fd_double_arrow"},
		{platform.MouseCursorResizeUp, "n-resize", "top_side"},
		{platform.MouseCursorResizeDown, "s-resize", "bottom_side"},
		{platform.MouseCursorResizeLeft, "w-resize", "left_side"},
		{platform.MouseCursorResizeRight, "e-resize", "right_side"},
		{platform.MouseCursorResizeUpLeft, "nw-resize", "top_left_corner"},
		{platform.MouseCursorResizeUpRight, "ne-resize", "top_right_corner"},
		{platform.MouseCursorResizeDownLeft, "sw-resize", "bottom_left_corner"},
		{platform.MouseCursorResizeDownRight, "se-resize", "bottom_right_corner"},
		{platform.MouseCursorResizeColumn, "col-resize", "sb_h_double_arrow"},
		{platform.MouseCursorResizeRow, "row-resize", "sb_v_double_arrow"},
		{platform.MouseCursorZoomIn, "zoom-in", "zoom-in"},
		{platform.MouseCursorZoomOut, "zoom-out", "zoom-out"},
	}
	covered := map[platform.MouseCursor]bool{platform.MouseCursorNone: true} // none hides the cursor
	for _, tt := range tests {
		t.Run(tt.cursor.String(), func(t *testing.T) {
			names := CursorNames(tt.cursor)
			if len(names) == 0 || names[0] != tt.css {
				t.Fatalf("names = %v, want %q first", names, tt.css)
			}
			if !slices.Contains(names, tt.legacy) {
				t.Errorf("names = %v, want the old name %q among them", names, tt.legacy)
			}
			seen := map[string]bool{}
			for _, n := range names {
				if seen[n] {
					t.Errorf("names = %v: %q twice", names, n)
				}
				seen[n] = true
			}
		})
		covered[tt.cursor] = true
	}
	for c := platform.MouseCursorUnspecified; c.Valid(); c++ {
		if !covered[c] {
			t.Errorf("%s is not in the table", c)
		}
	}
}
