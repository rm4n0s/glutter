package macos

import "github.com/rm4n0s/glutter/internal/platform"

// cursorSelector is the class method of NSCursor that gives the cursor for a mouse cursor, and false for
// MouseCursorNone, which is a blank cursor that the glue makes. It is the table of Flutter's embedder, in
// which a cursor that macOS has no public counterpart of is the arrow (SPEC §29.6):
//
//	wait, progress, help, cell, move, allScroll, the four diagonal resizes, zoom in and zoom out
//
// all show the arrow, as in Flutter; there is no deviation.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterMouseCursorPlugin.mm GetCursorForKind
func cursorSelector(c platform.MouseCursor) (selector string, visible bool) {
	switch c {
	case platform.MouseCursorNone:
		return "", false
	case platform.MouseCursorAlias:
		return "dragLinkCursor", true
	case platform.MouseCursorClick:
		return "pointingHandCursor", true
	case platform.MouseCursorContextMenu:
		return "contextualMenuCursor", true
	case platform.MouseCursorCopy:
		return "dragCopyCursor", true
	case platform.MouseCursorDisappearing:
		return "disappearingItemCursor", true
	case platform.MouseCursorForbidden, platform.MouseCursorNoDrop:
		return "operationNotAllowedCursor", true
	case platform.MouseCursorGrab:
		return "openHandCursor", true
	case platform.MouseCursorGrabbing:
		return "closedHandCursor", true
	case platform.MouseCursorPrecise:
		return "crosshairCursor", true
	case platform.MouseCursorText:
		return "IBeamCursor", true
	case platform.MouseCursorVerticalText:
		return "IBeamCursorForVerticalLayout", true
	case platform.MouseCursorResizeLeftRight, platform.MouseCursorResizeColumn:
		return "resizeLeftRightCursor", true
	case platform.MouseCursorResizeUpDown, platform.MouseCursorResizeRow:
		return "resizeUpDownCursor", true
	case platform.MouseCursorResizeUp:
		return "resizeUpCursor", true
	case platform.MouseCursorResizeDown:
		return "resizeDownCursor", true
	case platform.MouseCursorResizeLeft:
		return "resizeLeftCursor", true
	case platform.MouseCursorResizeRight:
		return "resizeRightCursor", true
	}
	// MouseCursorUnspecified, MouseCursorBasic and the cursors that Flutter shows as the arrow.
	return "arrowCursor", true
}
