//go:build darwin && !ios && glutter_display

package main

import (
	"bytes"
	"image"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
)

// TestMacOSScriptedKeysMoveTheSelection runs the editor on macOS (an accessory app, through glutter-macos
// shot) and presses the keys of the macOS text bindings with the scripted keys of GLUTTER_DEV_EVENTS, which
// make real NSEvents of keys (SPEC §29.6, §29.13). The editor starts with "Hello world" and the caret at the
// end; each frame after the keys must show in the text area what the frame of an editor that starts with
// that text and the selection the keys end in shows (GLUTTER_DEV_TEXT and GLUTTER_DEV_SELECT), pixel for
// pixel. Every script ends with a selection: a caret blinks, and two runs would show it in different phases.
// Typing comes with the text input (step 33.5). Run it with `go test -tags glutter_display ./examples/editor`.
// macEditor returns a function that runs the editor on macOS (an accessory app, through glutter-macos shot)
// with the given environment and returns the frame it saved, 2 seconds after the start. Each run has a data
// directory and a pasteboard of its own (GLUTTER_DEV_PASTEBOARD), so that a test never reads or changes the
// clipboard of the user.
func macEditor(t *testing.T) func(name string, env ...string) *image.RGBA {
	t.Helper()
	dir := t.TempDir()
	return func(name string, env ...string) *image.RGBA {
		t.Helper()
		out := filepath.Join(dir, strings.ReplaceAll(name, " ", "-")+".png")
		unique := filepath.Base(out)
		args := []string{"run", "../../cmd/glutter-macos", "shot", "-timeout", "2m",
			"-env", "GLUTTER_SCREENSHOT_AFTER=2s", "-env", "GLUTTER_RASTER_CACHE=0",
			"-env", "GLUTTER_DEV_DATA_DIR=" + filepath.Join(dir, "data-"+unique),
			"-env", "GLUTTER_DEV_PASTEBOARD=glutter-test-" + unique + "-" + filepath.Base(dir)}
		for _, e := range env {
			args = append(args, "-env", e)
		}
		args = append(args, "-o", out, ".")
		var log bytes.Buffer
		cmd := exec.Command("go", args...)
		cmd.Stdout, cmd.Stderr = &log, &log
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s: %v\n%s", name, err, log.String())
		}
		img, err := goldens.ReadPNG(out)
		if err != nil {
			t.Fatalf("%s saved no frame: %v\n%s", name, err, log.String())
		}
		return img
	}
}

func TestMacOSScriptedKeysMoveTheSelection(t *testing.T) {
	run := macEditor(t)
	const hello = "Hello world"
	area := image.Rect(16, 36, 480, 412) // the text area of the 820 x 460 window, without the status bar
	tests := []struct {
		name string
		keys string
		text string
		sel  string
	}{
		{"Command+A selects everything", "key:KeyA,meta", hello, "0,11"},
		{"Shift+Left three times", "key:ArrowLeft,shift;key:ArrowLeft,shift;key:ArrowLeft,shift", hello, "8,11"},
		{"Option+Shift+Left selects the word", "key:ArrowLeft,alt+shift", hello, "6,11"},
		{"Command+Shift+Left selects to the start of the line", "key:ArrowLeft,meta+shift", hello, "0,11"},
		{"Control+A goes to the start of the line, Shift+Right selects a character", "key:KeyA,control;key:ArrowRight,shift", hello, "0,1"},
		{"Command+Left goes to the start and Option+Shift+Right selects the word", "key:ArrowLeft,meta;key:ArrowRight,alt+shift", hello, "0,5"},
		{"Option+Backspace deletes the word", "key:Backspace,alt;wait:200ms;key:KeyA,meta", "Hello ", "0,6"},
		{"Command+Backspace deletes to the start of the line", "key:Backspace,meta;wait:200ms;key:KeyA,meta", "", "0,0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := run("keys", "GLUTTER_DEV_TEXT="+hello, "GLUTTER_DEV_SELECT=11,11", "GLUTTER_DEV_EVENTS=wait:300ms;"+tt.keys+";wait:300ms")
			want := run("start", "GLUTTER_DEV_TEXT="+tt.text, "GLUTTER_DEV_SELECT="+tt.sel)
			if n := macDiff(want, got, area); n != 0 {
				t.Errorf("the text area differs from the editor that starts with %q selected at %s in %d pixels", tt.text, tt.sel, n)
			}
		})
	}
}

// macDiff counts the pixels of the rectangle that differ.
func macDiff(want, got *image.RGBA, r image.Rectangle) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if want.RGBAAt(x, y) != got.RGBAAt(x, y) {
				n++
			}
		}
	}
	return n
}
