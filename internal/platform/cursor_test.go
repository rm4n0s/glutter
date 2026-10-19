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

func TestMouseCursorFromKind(t *testing.T) {
	kinds := []string{
		"none", "basic", "click", "forbidden", "wait", "progress", "contextMenu", "help", "text", "verticalText",
		"cell", "precise", "move", "grab", "grabbing", "noDrop", "alias", "copy", "disappearing", "allScroll",
		"resizeLeftRight", "resizeUpDown", "resizeUpLeftDownRight", "resizeUpRightDownLeft", "resizeUp", "resizeDown",
		"resizeLeft", "resizeRight", "resizeUpLeft", "resizeUpRight", "resizeDownLeft", "resizeDownRight",
		"resizeColumn", "resizeRow", "zoomIn", "zoomOut",
	}
	seen := map[MouseCursor]string{}
	for _, kind := range kinds {
		c, ok := MouseCursorFromKind(kind)
		if !ok || !c.Valid() || c == MouseCursorUnspecified {
			t.Errorf("MouseCursorFromKind(%q) = %v, %v", kind, c, ok)
			continue
		}
		if prev, dup := seen[c]; dup {
			t.Errorf("%q and %q give the same cursor %v", prev, kind, c)
		}
		seen[c] = kind
	}
	// Every cursor but the unspecified one has a kind.
	if len(seen) != int(mouseCursorEnd)-1 {
		t.Errorf("%d kinds map to %d cursors, want %d", len(kinds), len(seen), int(mouseCursorEnd)-1)
	}
	if _, ok := MouseCursorFromKind("nope"); ok {
		t.Error("an unknown kind was found")
	}
}
