package platform

// MouseCursor is a cursor shown over a window: one of Flutter's system mouse
// cursors. The zero value, MouseCursorUnspecified, means the platform's default
// arrow, like MouseCursorBasic.
//
// Flutter: services/mouse_cursor.dart SystemMouseCursors
type MouseCursor int

// The values of MouseCursor follow the declaration order of Flutter's
// SystemMouseCursors.
const (
	// MouseCursorUnspecified is the zero value: the default arrow.
	//
	// Flutter: no counterpart
	MouseCursorUnspecified MouseCursor = iota
	// MouseCursorNone hides the cursor.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.none
	MouseCursorNone
	// MouseCursorBasic is the arrow.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.basic
	MouseCursorBasic
	// MouseCursorClick is the pointing hand, for something that can be clicked.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.click
	MouseCursorClick
	// MouseCursorForbidden says the action is not allowed.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.forbidden
	MouseCursorForbidden
	// MouseCursorWait says the app is busy and cannot be used.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.wait
	MouseCursorWait
	// MouseCursorProgress says the app is busy but can still be used.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.progress
	MouseCursorProgress
	// MouseCursorContextMenu says a context menu is available.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.contextMenu
	MouseCursorContextMenu
	// MouseCursorHelp says help is available.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.help
	MouseCursorHelp
	// MouseCursorText is the I-beam, for text that can be selected.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.text
	MouseCursorText
	// MouseCursorVerticalText is the I-beam for vertical text.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.verticalText
	MouseCursorVerticalText
	// MouseCursorCell is the cross of a spreadsheet cell.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.cell
	MouseCursorCell
	// MouseCursorPrecise is the crosshair.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.precise
	MouseCursorPrecise
	// MouseCursorMove says an item can be moved.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.move
	MouseCursorMove
	// MouseCursorGrab is the open hand, for something that can be grabbed.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.grab
	MouseCursorGrab
	// MouseCursorGrabbing is the closed hand, for something being dragged.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.grabbing
	MouseCursorGrabbing
	// MouseCursorNoDrop says the item cannot be dropped here.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.noDrop
	MouseCursorNoDrop
	// MouseCursorAlias says a shortcut will be made.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.alias
	MouseCursorAlias
	// MouseCursorCopy says the item will be copied.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.copy
	MouseCursorCopy
	// MouseCursorDisappearing says the item will disappear.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.disappearing
	MouseCursorDisappearing
	// MouseCursorAllScroll says the content scrolls in every direction.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.allScroll
	MouseCursorAllScroll
	// MouseCursorResizeLeftRight resizes horizontally.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeLeftRight
	MouseCursorResizeLeftRight
	// MouseCursorResizeUpDown resizes vertically.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUpDown
	MouseCursorResizeUpDown
	// MouseCursorResizeUpLeftDownRight resizes along the diagonal from the top
	// left to the bottom right.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUpLeftDownRight
	MouseCursorResizeUpLeftDownRight
	// MouseCursorResizeUpRightDownLeft resizes along the diagonal from the top
	// right to the bottom left.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUpRightDownLeft
	MouseCursorResizeUpRightDownLeft
	// MouseCursorResizeUp moves the top edge.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUp
	MouseCursorResizeUp
	// MouseCursorResizeDown moves the bottom edge.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeDown
	MouseCursorResizeDown
	// MouseCursorResizeLeft moves the left edge.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeLeft
	MouseCursorResizeLeft
	// MouseCursorResizeRight moves the right edge.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeRight
	MouseCursorResizeRight
	// MouseCursorResizeUpLeft moves the top left corner.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUpLeft
	MouseCursorResizeUpLeft
	// MouseCursorResizeUpRight moves the top right corner.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeUpRight
	MouseCursorResizeUpRight
	// MouseCursorResizeDownLeft moves the bottom left corner.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeDownLeft
	MouseCursorResizeDownLeft
	// MouseCursorResizeDownRight moves the bottom right corner.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeDownRight
	MouseCursorResizeDownRight
	// MouseCursorResizeColumn resizes a column.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeColumn
	MouseCursorResizeColumn
	// MouseCursorResizeRow resizes a row.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.resizeRow
	MouseCursorResizeRow
	// MouseCursorZoomIn says the content will zoom in.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.zoomIn
	MouseCursorZoomIn
	// MouseCursorZoomOut says the content will zoom out.
	//
	// Flutter: services/mouse_cursor.dart SystemMouseCursors.zoomOut
	MouseCursorZoomOut

	// mouseCursorEnd is one past the last cursor.
	mouseCursorEnd
)

// Valid reports whether the cursor is one of the values above.
//
// Flutter: no counterpart
func (c MouseCursor) Valid() bool { return c >= MouseCursorUnspecified && c < mouseCursorEnd }

var mouseCursorNames = [...]string{
	"MouseCursorUnspecified", "MouseCursorNone", "MouseCursorBasic", "MouseCursorClick",
	"MouseCursorForbidden", "MouseCursorWait", "MouseCursorProgress", "MouseCursorContextMenu",
	"MouseCursorHelp", "MouseCursorText", "MouseCursorVerticalText", "MouseCursorCell",
	"MouseCursorPrecise", "MouseCursorMove", "MouseCursorGrab", "MouseCursorGrabbing",
	"MouseCursorNoDrop", "MouseCursorAlias", "MouseCursorCopy", "MouseCursorDisappearing",
	"MouseCursorAllScroll", "MouseCursorResizeLeftRight", "MouseCursorResizeUpDown",
	"MouseCursorResizeUpLeftDownRight", "MouseCursorResizeUpRightDownLeft", "MouseCursorResizeUp",
	"MouseCursorResizeDown", "MouseCursorResizeLeft", "MouseCursorResizeRight",
	"MouseCursorResizeUpLeft", "MouseCursorResizeUpRight", "MouseCursorResizeDownLeft",
	"MouseCursorResizeDownRight", "MouseCursorResizeColumn", "MouseCursorResizeRow",
	"MouseCursorZoomIn", "MouseCursorZoomOut",
}

// String returns the name of the cursor, such as "MouseCursorText".
//
// Flutter: services/mouse_cursor.dart SystemMouseCursors
func (c MouseCursor) String() string {
	if !c.Valid() {
		return "MouseCursor(invalid)"
	}
	return mouseCursorNames[c]
}
