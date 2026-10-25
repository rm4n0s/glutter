//go:build linux && glutter_wine

package main

import (
	"image"
	"testing"
)

// TestWineScriptedTextAndCompositionEditTheText builds the editor for Windows and types into it with the
// scripted text and compositions of GLUTTER_DEV_EVENTS (text:, compose:, composing:), which reach the window
// as WM_CHAR and as the messages of an input method (SPEC §25.7, §25.13). The frame after a commit must be
// pixel for pixel the frame of an editor that starts with that text and a caret after it (GLUTTER_DEV_TEXT
// and GLUTTER_DEV_SELECT); the frame of a composition in progress must show the text of the preedit and
// differ from that only by the underline of the composing range. Run it with
// `go test -tags glutter_wine -p 1 ./examples/editor`.
func TestWineScriptedTextAndCompositionEditTheText(t *testing.T) {
	run := wineEditor(t)
	// The text area of the 820 x 460 window, without the status bar (it changes with the caret).
	area := image.Rect(16, 36, 480, 412)
	tests := []struct {
		name string
		// events is the script after the first two seconds; text and sel the editor that must look the same.
		events string
		text   string
		sel    string
	}{
		{"typed text", "text:Hello world;wait:300ms", "Hello world", "0,11"},
		{"typed text with a surrogate pair", "text:a😀b;wait:300ms", "a😀b", "0,6"},
		{"a composition that is committed", "text:Say ;wait:300ms;compose:n,ni,niño;wait:300ms", "Say niño", "0,9"},
		{"a composition between two pieces of text", "text:ab;wait:300ms;key:LEFT;wait:300ms;compose:x,xy;wait:300ms", "axyb", "0,4"},
		{"a Japanese session", "text:Go ;wait:300ms;composing:k,か,かん,漢;wait:300ms;compose:漢字;wait:300ms", "Go 漢字", "0,9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The caret blinks, so the frames are compared with everything selected, which shows the text and
			// where it ends.
			got := run("typed", "GLUTTER_DEV_EVENTS=wait:2s;"+tt.events+";key:CONTROL+A")
			want := run("start", "GLUTTER_DEV_TEXT="+tt.text, "GLUTTER_DEV_SELECT="+tt.sel)
			if n := diff(want, got, area); n != 0 {
				t.Errorf("the text area differs from the editor that starts with %q selected at %s in %d pixels", tt.text, tt.sel, n)
			}
		})
	}

	t.Run("a composition in progress is underlined", func(t *testing.T) {
		got := run("composing", "GLUTTER_DEV_EVENTS=wait:2s;text:Hello ;wait:300ms;composing:w,wo,wor;wait:300ms")
		plain := run("plain", "GLUTTER_DEV_TEXT=Hello wor", "GLUTTER_DEV_SELECT=9,9")
		// The frames differ by the line under the preedit, and by the caret when its blink is on in one
		// of them and off in the other: the line is a row of many pixels, the caret a column of few
		// pixels in each row.
		n := diff(plain, got, area)
		if n == 0 {
			t.Fatal("the composing text is not marked: the frame equals the plain text")
		}
		if n > 150 {
			t.Errorf("the frame with a composition differs from the plain text in %d pixels, want a line and perhaps the caret", n)
		}
		if widest := widestRow(plain, got, area); widest < 20 {
			t.Errorf("the widest row of changed pixels is %d pixels, want a line under the three letters of the preedit", widest)
		}
	})
}

// widestRow is the most pixels of one row of the rectangle that differ.
func widestRow(want, got *image.RGBA, r image.Rectangle) int {
	widest := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		n := 0
		for x := r.Min.X; x < r.Max.X; x++ {
			if want.RGBAAt(x, y) != got.RGBAAt(x, y) {
				n++
			}
		}
		widest = max(widest, n)
	}
	return widest
}
