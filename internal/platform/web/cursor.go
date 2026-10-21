package web

import "github.com/rm4n0s/glutter/internal/platform"

// defaultCursor is the CSS cursor of the basic arrow: the cursor property is removed instead of
// set to it, so that the page's own style decides.
const defaultCursor = "default"

// cursorKindCSS maps the kind of a Flutter system mouse cursor to the CSS value of the cursor
// property. It must be kept in sync with the kinds of SystemMouseCursors.
//
// Flutter: lib/web_ui/lib/src/engine/mouse/cursor.dart MouseCursor._kindToCssValueMap
var cursorKindCSS = map[string]string{
	"alias":                 "alias",
	"allScroll":             "all-scroll",
	"basic":                 defaultCursor,
	"cell":                  "cell",
	"click":                 "pointer",
	"contextMenu":           "context-menu",
	"copy":                  "copy",
	"forbidden":             "not-allowed",
	"grab":                  "grab",
	"grabbing":              "grabbing",
	"help":                  "help",
	"move":                  "move",
	"none":                  "none",
	"noDrop":                "no-drop",
	"precise":               "crosshair",
	"progress":              "progress",
	"text":                  "text",
	"resizeColumn":          "col-resize",
	"resizeDown":            "s-resize",
	"resizeDownLeft":        "sw-resize",
	"resizeDownRight":       "se-resize",
	"resizeLeft":            "w-resize",
	"resizeLeftRight":       "ew-resize",
	"resizeRight":           "e-resize",
	"resizeRow":             "row-resize",
	"resizeUp":              "n-resize",
	"resizeUpDown":          "ns-resize",
	"resizeUpLeft":          "nw-resize",
	"resizeUpRight":         "ne-resize",
	"resizeUpLeftDownRight": "nwse-resize",
	"resizeUpRightDownLeft": "nesw-resize",
	"verticalText":          "vertical-text",
	"wait":                  "wait",
	"zoomIn":                "zoom-in",
	"zoomOut":               "zoom-out",
}

// cursorCSS is cursorKindCSS by cursor, made once from it.
var cursorCSS = func() map[platform.MouseCursor]string {
	m := make(map[platform.MouseCursor]string, len(cursorKindCSS))
	for kind, css := range cursorKindCSS {
		if c, ok := platform.MouseCursorFromKind(kind); ok {
			m[c] = css
		}
	}
	return m
}()

// CursorCSS returns the CSS value of the cursor property for a cursor. The basic arrow, the
// unspecified cursor and any cursor the table does not know are "default", for which the
// platform removes the property.
//
// Flutter: lib/web_ui/lib/src/engine/mouse/cursor.dart MouseCursor.activateSystemCursor
func CursorCSS(c platform.MouseCursor) string {
	if css, ok := cursorCSS[c]; ok {
		return css
	}
	return defaultCursor
}
