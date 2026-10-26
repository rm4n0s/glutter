package svg

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestParseColor(t *testing.T) {
	argb := ui.NewColorFromARGB
	tests := []struct {
		in      string
		want    ui.Color
		current bool
		ok      bool
	}{
		{"red", argb(255, 255, 0, 0), false, true},
		{"RED", argb(255, 255, 0, 0), false, true},
		{"CornflowerBlue", argb(255, 100, 149, 237), false, true},
		{"rebeccapurple", argb(255, 102, 51, 153), false, true},
		{"transparent", argb(0, 0, 0, 0), false, true},
		{"currentColor", ui.Color{}, true, true},
		{"currentcolor", ui.Color{}, true, true},
		{"#f00", argb(255, 255, 0, 0), false, true},
		{"#f008", argb(0x88, 255, 0, 0), false, true},
		{"#ff8000", argb(255, 255, 128, 0), false, true},
		{"#FF800080", argb(128, 255, 128, 0), false, true},
		{"#ff80", argb(0x00, 0xff, 0xff, 0x88), false, true},
		{"rgb(255, 128, 0)", argb(255, 255, 128, 0), false, true},
		{"rgb(255 128 0)", argb(255, 255, 128, 0), false, true},
		{"rgb(100%, 50%, 0%)", argb(255, 255, 128, 0), false, true},
		{"rgb(300, -5, 0)", argb(255, 255, 0, 0), false, true},
		{"rgba(255, 0, 0, 0.5)", argb(128, 255, 0, 0), false, true},
		{"rgb(255 0 0 / 50%)", argb(128, 255, 0, 0), false, true},
		{"hsl(0, 100%, 50%)", argb(255, 255, 0, 0), false, true},
		{"hsl(120deg 100% 25%)", argb(255, 0, 128, 0), false, true},
		{"hsla(240, 100%, 50%, 0.25)", argb(64, 0, 0, 255), false, true},
		{"hsl(0.5turn, 100%, 50%)", argb(255, 0, 255, 255), false, true},
		{"hsl(-120, 100%, 50%)", argb(255, 0, 0, 255), false, true},
		{"#ggg", ui.Color{}, false, false},
		{"#12345", ui.Color{}, false, false},
		{"notacolor", ui.Color{}, false, false},
		{"rgb(1, 2)", ui.Color{}, false, false},
		{"rgb(1, 2, x)", ui.Color{}, false, false},
		{"", ui.Color{}, false, false},
	}
	for _, tt := range tests {
		got, cur, ok := ParseColor(tt.in)
		if ok != tt.ok || cur != tt.current || (ok && !cur && got != tt.want) {
			t.Errorf("ParseColor(%q) = %v, %v, %v; want %v, %v, %v", tt.in, got, cur, ok, tt.want, tt.current, tt.ok)
		}
	}
}

func TestNamedColorsAreComplete(t *testing.T) {
	if len(namedColors) != 148 {
		t.Errorf("%d named colors, want 148 (147 of SVG 1.1 and rebeccapurple)", len(namedColors))
	}
}
