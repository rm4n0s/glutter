//go:build linux && !android

package linux

import "github.com/rm4n0s/glutter/internal/platform"

// cursorNames lists, for each cursor, the names to look up in an XCursor theme,
// best first: the CSS names that current themes use, then the older X11 names.
// The cursor MouseCursorNone has none, as it hides the cursor.
var cursorNames = map[platform.MouseCursor][]string{
	platform.MouseCursorUnspecified:           {"default", "left_ptr", "arrow"},
	platform.MouseCursorBasic:                 {"default", "left_ptr", "arrow"},
	platform.MouseCursorClick:                 {"pointer", "hand2", "hand1", "hand"},
	platform.MouseCursorForbidden:             {"not-allowed", "crossed_circle", "forbidden"},
	platform.MouseCursorWait:                  {"wait", "watch"},
	platform.MouseCursorProgress:              {"progress", "left_ptr_watch", "half-busy"},
	platform.MouseCursorContextMenu:           {"context-menu"},
	platform.MouseCursorHelp:                  {"help", "question_arrow", "left_ptr_help"},
	platform.MouseCursorText:                  {"text", "xterm", "ibeam"},
	platform.MouseCursorVerticalText:          {"vertical-text"},
	platform.MouseCursorCell:                  {"cell", "plus"},
	platform.MouseCursorPrecise:               {"crosshair", "cross", "tcross"},
	platform.MouseCursorMove:                  {"move", "fleur", "size_all"},
	platform.MouseCursorGrab:                  {"grab", "openhand", "hand1"},
	platform.MouseCursorGrabbing:              {"grabbing", "closedhand", "dnd-none"},
	platform.MouseCursorNoDrop:                {"no-drop", "dnd-no-drop", "forbidden"},
	platform.MouseCursorAlias:                 {"alias", "dnd-link"},
	platform.MouseCursorCopy:                  {"copy", "dnd-copy"},
	platform.MouseCursorDisappearing:          {"default", "left_ptr"},
	platform.MouseCursorAllScroll:             {"all-scroll", "fleur", "size_all"},
	platform.MouseCursorResizeLeftRight:       {"ew-resize", "sb_h_double_arrow", "h_double_arrow", "size_hor"},
	platform.MouseCursorResizeUpDown:          {"ns-resize", "sb_v_double_arrow", "v_double_arrow", "size_ver"},
	platform.MouseCursorResizeUpLeftDownRight: {"nwse-resize", "bd_double_arrow", "size_fdiag"},
	platform.MouseCursorResizeUpRightDownLeft: {"nesw-resize", "fd_double_arrow", "size_bdiag"},
	platform.MouseCursorResizeUp:              {"n-resize", "top_side"},
	platform.MouseCursorResizeDown:            {"s-resize", "bottom_side"},
	platform.MouseCursorResizeLeft:            {"w-resize", "left_side"},
	platform.MouseCursorResizeRight:           {"e-resize", "right_side"},
	platform.MouseCursorResizeUpLeft:          {"nw-resize", "top_left_corner"},
	platform.MouseCursorResizeUpRight:         {"ne-resize", "top_right_corner"},
	platform.MouseCursorResizeDownLeft:        {"sw-resize", "bottom_left_corner"},
	platform.MouseCursorResizeDownRight:       {"se-resize", "bottom_right_corner"},
	platform.MouseCursorResizeColumn:          {"col-resize", "sb_h_double_arrow", "split_h"},
	platform.MouseCursorResizeRow:             {"row-resize", "sb_v_double_arrow", "split_v"},
	platform.MouseCursorZoomIn:                {"zoom-in"},
	platform.MouseCursorZoomOut:               {"zoom-out"},
}

// CursorNames returns the names to look up in an XCursor theme for the cursor,
// best first, and nil for MouseCursorNone and for an invalid cursor. Wayland and
// X11 both use it.
//
// Flutter: no counterpart (the Linux embedder passes the names to GTK)
func CursorNames(c platform.MouseCursor) []string {
	return append([]string(nil), cursorNames[c]...)
}
