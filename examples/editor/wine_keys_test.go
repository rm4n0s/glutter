//go:build linux && glutter_wine

package main

import (
	"context"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/winetest"
)

// TestWineScriptedKeysEditTheText builds the editor for Windows and types into it with the scripted
// keys of GLUTTER_DEV_EVENTS, which put real key input into the system's queue (SPEC §25.6, §25.13).
// Each frame, after the keys, must show in the text area what the frame of an editor that starts
// with that text and selection shows (GLUTTER_DEV_TEXT and GLUTTER_DEV_SELECT), pixel for pixel:
// letters, Shift, Backspace, Delete, the arrows, Home and End, Page Down, Control+A and the word
// movement of Control+Shift+arrows. The clipboard keys (Control+C, X and V) need the clipboard of
// Windows, which comes with step 28.6. A key that moves the caret by the layout (a page, a line
// break, a word) waits for a frame after the edit it follows: those actions work on the value that
// the last frame laid out, as Flutter's do, and keys of one step can all arrive before the next frame. Run it with `go test -tags glutter_wine -p 1 ./examples/editor`.
func TestWineScriptedKeysEditTheText(t *testing.T) {
	h, err := winetest.New("")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "editor.exe")
	build := exec.Command("go", "build", "-o", exe, "./examples/editor")
	build.Dir = h.Root
	build.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		stop, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := h.Stop(stop); err != nil {
			t.Errorf("Stop: %v", err)
		}
		if left := winetest.Leftovers(h.Root); len(left) > 0 {
			t.Errorf("processes left running:\n%s", strings.Join(left, "\n"))
		}
	})
	run := func(name string, env ...string) *image.RGBA {
		t.Helper()
		shot := filepath.Join(dir, name+".png")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		data := filepath.Join(dir, name+"-data")
		res, err := h.Run(ctx, exe, winetest.RunOptions{
			Env:     append([]string{"GLUTTER_DEV_DATA_DIR=" + data, "GLUTTER_SCREENSHOT=" + shot, "GLUTTER_SCREENSHOT_AFTER=5s"}, env...),
			Timeout: time.Minute,
		})
		if err != nil || res.ExitCode != 0 {
			t.Fatalf("%s: %v, status %d\n%s", name, err, res.ExitCode, res.Output)
		}
		img, err := goldens.ReadPNG(shot)
		if err != nil {
			t.Fatalf("%s saved no frame: %v\n%s", name, err, res.Output)
		}
		return img
	}
	typeHello := "key:SHIFT+H;key:E;key:L;key:L;key:O;key:SPACE;key:W;key:O;key:R;key:L;key:D;wait:300ms"
	// The text area of the 820 x 460 window, without the status bar (it changes with the caret).
	area := image.Rect(16, 36, 480, 412)
	tests := []struct {
		name string
		// keys is the script after the first two seconds, text and sel the editor that must look the same.
		keys string
		text string
		sel  string
	}{
		{"selected with Shift and the arrows", typeHello + ";keydown:SHIFT;key:LEFT;key:LEFT;key:LEFT;keyup:SHIFT", "Hello world", "8,11"},
		{"Backspace and select to the start with Shift+Home", typeHello + ";key:BACK;wait:300ms;key:SHIFT+HOME", "Hello worl", "0,10"},
		{"Home, Delete and select to the end with Shift+End", typeHello + ";key:HOME;key:DELETE;wait:300ms;key:SHIFT+END", "ello world", "0,10"},
		{"Control+A", typeHello + ";key:CONTROL+A", "Hello world", "0,11"},
		{"Control+Shift+Left selects words", typeHello + ";key:CONTROL+SHIFT+LEFT", "Hello world", "6,11"},
		{"Control+Left and Shift+Right", typeHello + ";key:CONTROL+LEFT;key:LEFT;key:HOME;keydown:SHIFT;key:RIGHT;key:RIGHT;keyup:SHIFT", "Hello world", "0,2"},
		{"Enter makes a line, and Page Down goes to the last line", typeHello + ";key:RETURN;key:A;wait:300ms;key:CONTROL+HOME;wait:300ms;key:PAGEDOWN;wait:300ms;key:SHIFT+END", "Hello world\\na", "12,13"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := run("keys", "GLUTTER_DEV_EVENTS=wait:2s;"+tt.keys)
			want := run("start", "GLUTTER_DEV_TEXT="+tt.text, "GLUTTER_DEV_SELECT="+tt.sel)
			if n := diff(want, got, area); n != 0 {
				t.Errorf("the text area differs from the editor that starts with %q selected at %s in %d pixels", tt.text, tt.sel, n)
			}
		})
	}
}

// diff counts the pixels of the rectangle that differ.
func diff(want, got *image.RGBA, r image.Rectangle) int {
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
