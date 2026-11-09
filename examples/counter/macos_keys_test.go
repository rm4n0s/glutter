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

// TestMacOSKeysPressTheFocusedButton runs the counter on macOS and presses its button with the keys of
// GLUTTER_DEV_EVENTS, which make real NSEvents of keys (SPEC §29.6, §29.13): Tab moves the focus to the
// floating action button, and each Enter or Space presses it. The frame after Tab and three presses must
// show what an app that starts at 3 shows (GLUTTER_DEV_TAPS=3) in the text of the counter, pixel for pixel,
// and a different number of presses another frame. Run it with `go test -tags glutter_display ./examples/counter`.
func TestMacOSKeysPressTheFocusedButton(t *testing.T) {
	dir := t.TempDir()
	run := func(name string, env ...string) *image.RGBA {
		t.Helper()
		out := filepath.Join(dir, name+".png")
		args := []string{"run", "../../cmd/glutter-macos", "shot", "-timeout", "2m", "-env", "GLUTTER_SCREENSHOT_AFTER=1500ms"}
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
	presses := func(n int, keys ...string) string {
		steps := []string{"wait:300ms", "key:Tab", "wait:50ms"}
		for i := range n {
			steps = append(steps, "key:"+keys[i%len(keys)], "wait:100ms")
		}
		return "GLUTTER_DEV_EVENTS=" + strings.Join(steps, ";")
	}
	text := image.Rect(60, 300, 420, 390)
	three := run("three-taps", "GLUTTER_DEV_TAPS=3")
	for _, tc := range []struct {
		name     string
		n        int
		keys     []string
		wantSame bool
	}{
		{"one Enter", 1, []string{"Enter"}, false},
		{"two Enters", 2, []string{"Enter"}, false},
		{"three Enters", 3, []string{"Enter"}, true},
		{"Space and Enter", 3, []string{"Space", "Enter"}, true},
	} {
		got := run(strings.ReplaceAll(tc.name, " ", "-"), presses(tc.n, tc.keys...))
		d := macDiff(three, got, text)
		if tc.wantSame && d != 0 {
			t.Errorf("%s: the text differs from the counter at 3 in %d pixels", tc.name, d)
		}
		if !tc.wantSame && d == 0 {
			t.Errorf("%s: the text is the one of 3", tc.name)
		}
	}
}
