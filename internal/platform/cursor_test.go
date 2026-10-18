package platform

import "testing"

func TestMouseCursorNamesCoverEveryCursor(t *testing.T) {
	if len(mouseCursorNames) != int(mouseCursorEnd) {
		t.Fatalf("%d names for %d cursors", len(mouseCursorNames), mouseCursorEnd)
	}
	seen := map[string]bool{}
	for c := MouseCursorUnspecified; c < mouseCursorEnd; c++ {
		name := c.String()
		if name == "" || seen[name] {
			t.Errorf("cursor %d has the name %q", int(c), name)
		}
		seen[name] = true
		if !c.Valid() {
			t.Errorf("%s is not valid", name)
		}
	}
	for _, c := range []MouseCursor{-1, mouseCursorEnd} {
		if c.Valid() || c.String() != "MouseCursor(invalid)" {
			t.Errorf("MouseCursor(%d): Valid %v, String %q", int(c), c.Valid(), c.String())
		}
	}
	if MouseCursorUnspecified != 0 {
		t.Error("the zero value is not MouseCursorUnspecified")
	}
}
