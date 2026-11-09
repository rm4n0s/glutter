package macos

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
)

func TestCursorTableCoversEveryMouseCursor(t *testing.T) {
	arrow := map[platform.MouseCursor]bool{
		platform.MouseCursorUnspecified: true, platform.MouseCursorBasic: true,
		// The cursors that Flutter's embedder has no entry for.
		platform.MouseCursorWait: true, platform.MouseCursorProgress: true, platform.MouseCursorHelp: true,
		platform.MouseCursorCell: true, platform.MouseCursorMove: true, platform.MouseCursorAllScroll: true,
		platform.MouseCursorResizeUpLeftDownRight: true, platform.MouseCursorResizeUpRightDownLeft: true,
		platform.MouseCursorResizeUpLeft: true, platform.MouseCursorResizeUpRight: true,
		platform.MouseCursorResizeDownLeft: true, platform.MouseCursorResizeDownRight: true,
		platform.MouseCursorZoomIn: true, platform.MouseCursorZoomOut: true,
	}
	named := map[platform.MouseCursor]string{
		platform.MouseCursorAlias: "dragLinkCursor", platform.MouseCursorClick: "pointingHandCursor",
		platform.MouseCursorContextMenu: "contextualMenuCursor", platform.MouseCursorCopy: "dragCopyCursor",
		platform.MouseCursorDisappearing: "disappearingItemCursor", platform.MouseCursorForbidden: "operationNotAllowedCursor",
		platform.MouseCursorNoDrop: "operationNotAllowedCursor", platform.MouseCursorGrab: "openHandCursor",
		platform.MouseCursorGrabbing: "closedHandCursor", platform.MouseCursorPrecise: "crosshairCursor",
		platform.MouseCursorText: "IBeamCursor", platform.MouseCursorVerticalText: "IBeamCursorForVerticalLayout",
		platform.MouseCursorResizeColumn: "resizeLeftRightCursor", platform.MouseCursorResizeLeftRight: "resizeLeftRightCursor",
		platform.MouseCursorResizeRow: "resizeUpDownCursor", platform.MouseCursorResizeUpDown: "resizeUpDownCursor",
		platform.MouseCursorResizeUp: "resizeUpCursor", platform.MouseCursorResizeDown: "resizeDownCursor",
		platform.MouseCursorResizeLeft: "resizeLeftCursor", platform.MouseCursorResizeRight: "resizeRightCursor",
	}
	n := 0
	for c := platform.MouseCursorUnspecified; c.Valid(); c++ {
		n++
		sel, visible := cursorSelector(c)
		switch {
		case c == platform.MouseCursorNone:
			if visible || sel != "" {
				t.Errorf("%v: the cursor of none is %q, visible %v", c, sel, visible)
			}
		case arrow[c]:
			if !visible || sel != "arrowCursor" {
				t.Errorf("%v: %q, want the arrow", c, sel)
			}
		default:
			want, ok := named[c]
			if !ok {
				t.Errorf("%v is in no table of the test: a new cursor?", c)
				continue
			}
			if !visible || sel != want {
				t.Errorf("%v: %q, want %q", c, sel, want)
			}
		}
	}
	if n != 37 {
		t.Errorf("the platform has %d cursors, the test knows 37", n)
	}
}
