//go:build darwin && !ios && glutter_display

package main

import (
	"bytes"
	"image"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
)

// TestMacOSTrackpadScrollMovesTheViewAndTheFlingGoesOn runs the scrolling example on macOS and scripts a
// two-finger scroll of the trackpad (dev events `pan`: real NSEvents with their phases, SPEC §29.6). The list
// moves; a frame taken a second later than the first shows it further down, because the framework's own
// ballistic scroll takes over where the gesture ended and the momentum events of the system are dropped.
// Run it with `go test -tags glutter_display ./examples/scrolling`.
func TestMacOSTrackpadScrollMovesTheViewAndTheFlingGoesOn(t *testing.T) {
	dir := t.TempDir()
	shot := func(name string, env ...string) *image.RGBA {
		t.Helper()
		out := filepath.Join(dir, name+".png")
		args := []string{"run", "../../cmd/glutter-macos", "shot", "-timeout", "2m"}
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
	pan := "GLUTTER_DEV_EVENTS=wait:300ms;pan:200,300,0,-400,200ms"
	list := image.Rect(0, 60, 600, 480) // the rows, under the tabs
	still := shot("still")
	justAfter := shot("just-after", pan, "GLUTTER_SCREENSHOT_AFTER=800ms")
	later := shot("later", pan, "GLUTTER_SCREENSHOT_AFTER=3s")
	if macDiff(still, justAfter, list) == 0 {
		t.Error("the scripted scroll did not move the list")
	}
	if macDiff(justAfter, later, list) == 0 {
		t.Error("the list stopped where the gesture ended: the fling did not go on")
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
