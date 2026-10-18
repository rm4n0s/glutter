package linux

import (
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
