package android

import "github.com/rm4n0s/glutter/internal/platform"

// The android.view.PointerIcon types that Flutter's MouseCursorPlugin uses.
const (
	pointerIconNull                      = 0
	pointerIconArrow                     = 1000
	pointerIconContextMenu               = 1001
	pointerIconHand                      = 1002
	pointerIconHelp                      = 1003
	pointerIconWait                      = 1004
	pointerIconCell                      = 1006
	pointerIconCrosshair                 = 1007
	pointerIconText                      = 1008
	pointerIconVerticalText              = 1009
	pointerIconAlias                     = 1010
	pointerIconCopy                      = 1011
	pointerIconNoDrop                    = 1012
	pointerIconAllScroll                 = 1013
	pointerIconHorizontalDoubleArrow     = 1014
	pointerIconVerticalDoubleArrow       = 1015
	pointerIconTopRightDiagonalDoubleArr = 1016
	pointerIconTopLeftDiagonalDoubleArr  = 1017
	pointerIconZoomIn                    = 1018
	pointerIconZoomOut                   = 1019
	pointerIconGrab                      = 1020
	pointerIconGrabbing                  = 1021
)

// cursorKindPointerIcon maps the kind of a Flutter system mouse cursor to the PointerIcon type.
// A kind that is not in it is the arrow, as in Flutter.
//
// Flutter: shell/platform/android/io/flutter/plugin/mouse/MouseCursorPlugin.java MouseCursorPlugin.resolveSystemCursor
var cursorKindPointerIcon = map[string]int{
	"alias":                 pointerIconAlias,
	"allScroll":             pointerIconAllScroll,
	"basic":                 pointerIconArrow,
	"cell":                  pointerIconCell,
	"click":                 pointerIconHand,
	"contextMenu":           pointerIconContextMenu,
	"copy":                  pointerIconCopy,
	"forbidden":             pointerIconNoDrop,
	"grab":                  pointerIconGrab,
	"grabbing":              pointerIconGrabbing,
	"help":                  pointerIconHelp,
	"move":                  pointerIconAllScroll,
	"none":                  pointerIconNull,
	"noDrop":                pointerIconNoDrop,
	"precise":               pointerIconCrosshair,
	"text":                  pointerIconText,
	"resizeColumn":          pointerIconHorizontalDoubleArrow,
	"resizeDown":            pointerIconVerticalDoubleArrow,
	"resizeUpLeft":          pointerIconTopLeftDiagonalDoubleArr,
	"resizeDownRight":       pointerIconTopLeftDiagonalDoubleArr,
	"resizeLeft":            pointerIconHorizontalDoubleArrow,
	"resizeLeftRight":       pointerIconHorizontalDoubleArrow,
	"resizeRight":           pointerIconHorizontalDoubleArrow,
	"resizeRow":             pointerIconVerticalDoubleArrow,
	"resizeUp":              pointerIconVerticalDoubleArrow,
	"resizeUpDown":          pointerIconVerticalDoubleArrow,
	"resizeUpRight":         pointerIconTopRightDiagonalDoubleArr,
	"resizeUpLeftDownRight": pointerIconTopLeftDiagonalDoubleArr,
	"resizeUpRightDownLeft": pointerIconTopRightDiagonalDoubleArr,
	"verticalText":          pointerIconVerticalText,
	"wait":                  pointerIconWait,
	"zoomIn":                pointerIconZoomIn,
	"zoomOut":               pointerIconZoomOut,
}

// cursorPointerIcon is cursorKindPointerIcon by cursor, made once from it.
var cursorPointerIcon = func() map[platform.MouseCursor]int {
	m := make(map[platform.MouseCursor]int, len(cursorKindPointerIcon))
	for kind, icon := range cursorKindPointerIcon {
		if c, ok := platform.MouseCursorFromKind(kind); ok {
			m[c] = icon
		}
	}
	return m
}()

// PointerIconType returns the android.view.PointerIcon type of a cursor: the arrow for the
// unspecified cursor and for any cursor that Flutter's table does not know.
//
// Flutter: shell/platform/android/io/flutter/plugin/mouse/MouseCursorPlugin.java MouseCursorPlugin.resolveSystemCursor
func PointerIconType(c platform.MouseCursor) int {
	if t, ok := cursorPointerIcon[c]; ok {
		return t
	}
	return pointerIconArrow
}
