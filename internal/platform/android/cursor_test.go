package android

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
)

func TestPointerIconType(t *testing.T) {
	tests := []struct {
		kind string
		want int
	}{
		{"basic", 1000}, {"click", 1002}, {"text", 1008}, {"none", 0}, {"forbidden", 1012}, {"noDrop", 1012},
		{"move", 1013}, {"allScroll", 1013}, {"grab", 1020}, {"grabbing", 1021}, {"wait", 1004},
		{"resizeLeftRight", 1014}, {"resizeUpDown", 1015}, {"resizeUpLeft", 1017}, {"resizeUpRight", 1016},
		{"resizeUpLeftDownRight", 1017}, {"resizeUpRightDownLeft", 1016}, {"zoomIn", 1018}, {"zoomOut", 1019},
		{"precise", 1007}, {"verticalText", 1009}, {"help", 1003}, {"contextMenu", 1001}, {"cell", 1006},
		{"alias", 1010}, {"copy", 1011},
		{"progress", 1000}, // not in Flutter's table: the arrow
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			c, ok := platform.MouseCursorFromKind(tt.kind)
			if !ok {
				t.Fatalf("%q is not a system mouse cursor", tt.kind)
			}
			if got := PointerIconType(c); got != tt.want {
				t.Errorf("PointerIconType(%s) = %d, want %d", tt.kind, got, tt.want)
			}
		})
	}
	if got := PointerIconType(platform.MouseCursorUnspecified); got != pointerIconArrow {
		t.Errorf("the unspecified cursor is %d, want the arrow", got)
	}
}

// Every kind of Flutter's table is a cursor of the platform package.
func TestEveryCursorKindIsKnown(t *testing.T) {
	for kind := range cursorKindPointerIcon {
		if _, ok := platform.MouseCursorFromKind(kind); !ok {
			t.Errorf("%q is in the table but is not a system mouse cursor", kind)
		}
	}
}
