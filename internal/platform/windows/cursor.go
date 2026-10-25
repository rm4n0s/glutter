package windows

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// cursorResource is the stock cursor (IDC_*) that stands for a mouse cursor, and false for
// MouseCursorNone, which hides the cursor (SetCursor(NULL)). It is the table of Flutter's embedder
// (GetCursorByName) with the cursors that Windows has no stock cursor for given the nearest one
// (SPEC §25.6, §12.10):
//
// Deviation: Flutter's embedder has no entry for grab, grabbing, zoom in, zoom out, copy, alias,
// cell, context menu, vertical text and disappearing, and shows the arrow for all of them. Here
// grab and grabbing (the closed hand) take IDC_HAND; zoom in, zoom out and cell take IDC_CROSS;
// vertical text takes IDC_IBEAM; copy, alias, context menu and disappearing keep IDC_ARROW.
//
// Flutter: shell/platform/windows/flutter_windows_engine.cc FlutterWindowsEngine::GetCursorByName
func cursorResource(c platform.MouseCursor) (idc uintptr, visible bool) {
	switch c {
	case platform.MouseCursorNone:
		return 0, false
	case platform.MouseCursorClick, platform.MouseCursorGrab, platform.MouseCursorGrabbing:
		return win32.IDCHand, true
	case platform.MouseCursorText, platform.MouseCursorVerticalText:
		return win32.IDCIBeam, true
	case platform.MouseCursorWait:
		return win32.IDCWait, true
	case platform.MouseCursorProgress:
		return win32.IDCAppStarting, true
	case platform.MouseCursorForbidden, platform.MouseCursorNoDrop:
		return win32.IDCNo, true
	case platform.MouseCursorHelp:
		return win32.IDCHelp, true
	case platform.MouseCursorPrecise, platform.MouseCursorCell, platform.MouseCursorZoomIn, platform.MouseCursorZoomOut:
		return win32.IDCCross, true
	case platform.MouseCursorMove, platform.MouseCursorAllScroll:
		return win32.IDCSizeAll, true
	case platform.MouseCursorResizeLeftRight, platform.MouseCursorResizeLeft, platform.MouseCursorResizeRight, platform.MouseCursorResizeColumn:
		return win32.IDCSizeWE, true
	case platform.MouseCursorResizeUpDown, platform.MouseCursorResizeUp, platform.MouseCursorResizeDown, platform.MouseCursorResizeRow:
		return win32.IDCSizeNS, true
	case platform.MouseCursorResizeUpLeftDownRight, platform.MouseCursorResizeUpLeft, platform.MouseCursorResizeDownRight:
		return win32.IDCSizeNWSE, true
	case platform.MouseCursorResizeUpRightDownLeft, platform.MouseCursorResizeUpRight, platform.MouseCursorResizeDownLeft:
		return win32.IDCSizeNESW, true
	}
	// MouseCursorUnspecified, MouseCursorBasic, MouseCursorCopy, MouseCursorAlias,
	// MouseCursorContextMenu and MouseCursorDisappearing.
	return win32.IDCArrow, true
}
