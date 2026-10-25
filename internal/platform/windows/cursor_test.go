package windows

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// flutterCursors is the table of FlutterWindowsEngine::GetCursorByName, by the names of
// SystemMouseCursors (0 is the cursor of "none": SetCursor(NULL)).
var flutterCursors = map[string]uintptr{
	"allScroll": win32.IDCSizeAll, "basic": win32.IDCArrow, "click": win32.IDCHand, "forbidden": win32.IDCNo,
	"help": win32.IDCHelp, "move": win32.IDCSizeAll, "none": 0, "noDrop": win32.IDCNo, "precise": win32.IDCCross,
	"progress": win32.IDCAppStarting, "text": win32.IDCIBeam, "resizeColumn": win32.IDCSizeWE,
	"resizeDown": win32.IDCSizeNS, "resizeDownLeft": win32.IDCSizeNESW, "resizeDownRight": win32.IDCSizeNWSE,
	"resizeLeft": win32.IDCSizeWE, "resizeLeftRight": win32.IDCSizeWE, "resizeRight": win32.IDCSizeWE,
	"resizeRow": win32.IDCSizeNS, "resizeUp": win32.IDCSizeNS, "resizeUpDown": win32.IDCSizeNS,
	"resizeUpLeft": win32.IDCSizeNWSE, "resizeUpRight": win32.IDCSizeNESW,
	"resizeUpLeftDownRight": win32.IDCSizeNWSE, "resizeUpRightDownLeft": win32.IDCSizeNESW, "wait": win32.IDCWait,
}

// TestCursorTableIsFlutters checks every entry of Flutter's embedder against ours (cursor_handler_unittests.cc
// ActivateSystemCursor checks that "click" is IDC_HAND).
func TestCursorTableIsFlutters(t *testing.T) {
	for name, want := range flutterCursors {
		t.Run(name, func(t *testing.T) {
			c, ok := platform.MouseCursorFromKind(name)
			if !ok {
				t.Fatalf("%q is not a system mouse cursor", name)
			}
			idc, visible := cursorResource(c)
			if want == 0 {
				if visible {
					t.Errorf("%s is visible, want it hidden", name)
				}
				return
			}
			if !visible || idc != want {
				t.Errorf("%s = %d (visible %v), want %d", name, idc, visible, want)
			}
		})
	}
}

// TestCursorTableCoversEverySystemMouseCursor lists the cursor that every name of SystemMouseCursors
// takes; the cursors Windows has no stock cursor for are the ones of SPEC §12.10.
func TestCursorTableCoversEverySystemMouseCursor(t *testing.T) {
	want := map[platform.MouseCursor]uintptr{
		platform.MouseCursorUnspecified:           win32.IDCArrow,
		platform.MouseCursorNone:                  0,
		platform.MouseCursorBasic:                 win32.IDCArrow,
		platform.MouseCursorClick:                 win32.IDCHand,
		platform.MouseCursorForbidden:             win32.IDCNo,
		platform.MouseCursorWait:                  win32.IDCWait,
		platform.MouseCursorProgress:              win32.IDCAppStarting,
		platform.MouseCursorContextMenu:           win32.IDCArrow, // no stock cursor
		platform.MouseCursorHelp:                  win32.IDCHelp,
		platform.MouseCursorText:                  win32.IDCIBeam,
		platform.MouseCursorVerticalText:          win32.IDCIBeam, // no stock cursor
		platform.MouseCursorCell:                  win32.IDCCross, // no stock cursor
		platform.MouseCursorPrecise:               win32.IDCCross,
		platform.MouseCursorMove:                  win32.IDCSizeAll,
		platform.MouseCursorGrab:                  win32.IDCHand,
		platform.MouseCursorGrabbing:              win32.IDCHand, // no stock cursor
		platform.MouseCursorNoDrop:                win32.IDCNo,
		platform.MouseCursorAlias:                 win32.IDCArrow, // no stock cursor
		platform.MouseCursorCopy:                  win32.IDCArrow, // no stock cursor
		platform.MouseCursorDisappearing:          win32.IDCArrow, // no stock cursor
		platform.MouseCursorAllScroll:             win32.IDCSizeAll,
		platform.MouseCursorResizeLeftRight:       win32.IDCSizeWE,
		platform.MouseCursorResizeUpDown:          win32.IDCSizeNS,
		platform.MouseCursorResizeUpLeftDownRight: win32.IDCSizeNWSE,
		platform.MouseCursorResizeUpRightDownLeft: win32.IDCSizeNESW,
		platform.MouseCursorResizeUp:              win32.IDCSizeNS,
		platform.MouseCursorResizeDown:            win32.IDCSizeNS,
		platform.MouseCursorResizeLeft:            win32.IDCSizeWE,
		platform.MouseCursorResizeRight:           win32.IDCSizeWE,
		platform.MouseCursorResizeUpLeft:          win32.IDCSizeNWSE,
		platform.MouseCursorResizeUpRight:         win32.IDCSizeNESW,
		platform.MouseCursorResizeDownLeft:        win32.IDCSizeNESW,
		platform.MouseCursorResizeDownRight:       win32.IDCSizeNWSE,
		platform.MouseCursorResizeColumn:          win32.IDCSizeWE,
		platform.MouseCursorResizeRow:             win32.IDCSizeNS,
		platform.MouseCursorZoomIn:                win32.IDCCross, // no stock cursor
		platform.MouseCursorZoomOut:               win32.IDCCross, // no stock cursor
	}
	n := 0
	for c := platform.MouseCursorUnspecified; c.Valid(); c++ {
		n++
		w, ok := want[c]
		if !ok {
			t.Errorf("%v has no entry in this test: a new cursor needs one in cursorResource", c)
			continue
		}
		idc, visible := cursorResource(c)
		if w == 0 {
			if visible {
				t.Errorf("%v is visible, want it hidden", c)
			}
			continue
		}
		if !visible || idc != w {
			t.Errorf("%v = %d (visible %v), want %d", c, idc, visible, w)
		}
	}
	if n != len(want) {
		t.Errorf("%d cursors, %d entries", n, len(want))
	}
}
