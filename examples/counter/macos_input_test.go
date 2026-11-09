//go:build darwin && !ios && glutter_display

package main

import (
	"bytes"
	"image"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
)

// TestMacOSThreeClicksCountThree runs the counter on macOS (an accessory app, through glutter-macos shot) and
// clicks its floating action button with the scripted events of GLUTTER_DEV_EVENTS, which make real NSEvents
// (SPEC §29.6, §29.13). The frame after three clicks must show what an app that starts at 3 shows
// (GLUTTER_DEV_TAPS=3) in the text of the counter, pixel for pixel; one and two clicks show other frames.
// Run it with `go test -tags glutter_display ./examples/counter`.
func TestMacOSThreeClicksCountThree(t *testing.T) {
	dir := t.TempDir()
	run := func(name string, env ...string) *image.RGBA { return macShot(t, dir, name, env...) }
	// The floating action button is at the bottom right of the 480 x 640 window.
	const fabX, fabY = 436, 596
	clicks := func(n int) string {
		steps := []string{"wait:300ms"}
		for range n {
			steps = append(steps, "tap:"+strconv.Itoa(fabX)+","+strconv.Itoa(fabY), "wait:150ms")
		}
		return "GLUTTER_DEV_EVENTS=" + strings.Join(steps, ";")
	}
	text := image.Rect(60, 300, 420, 390) // "You have pushed ..." and the count
	three := run("three-taps", "GLUTTER_DEV_TAPS=3")
	zero := run("zero")
	if macDiff(zero, three, text) == 0 {
		t.Fatal("the text of the counter is the same at 0 and at 3")
	}
	for n, wantSame := range map[int]bool{1: false, 2: false, 3: true} {
		got := run("clicks-"+strconv.Itoa(n), clicks(n))
		d := macDiff(three, got, text)
		if wantSame && d != 0 {
			t.Errorf("after 3 clicks the text differs from the counter at 3 in %d pixels", d)
		}
		if !wantSame && d == 0 {
			t.Errorf("after %d clicks the text is the one of 3", n)
		}
	}
}

// macShot runs the counter with `glutter-macos shot` (the environment is the extra settings of the run) and returns
// the frame it saved.
func macShot(t *testing.T, dir, name string, env ...string) *image.RGBA {
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

// TestMacOSCounterWithMetalEqualsSoft draws the counter with Metal and with the software renderer and compares the
// two frames within the tolerance of the GPU renderers (SPEC §9.6, §29.5): it is the first frame of an app on Metal,
// and the text of the counter is in both.
func TestMacOSCounterWithMetalEqualsSoft(t *testing.T) {
	dir := t.TempDir()
	soft := macShot(t, dir, "soft", "GLUTTER_RENDERER=soft")
	metal := macShot(t, dir, "metal", "GLUTTER_RENDERER=metal")
	if soft.Bounds() != metal.Bounds() {
		t.Fatalf("the frames are %v (soft) and %v (metal)", soft.Bounds(), metal.Bounds())
	}
	gputest.Tolerance{Over8Permille: 30, Over32Permille: 3}.Check(t, "counter, metal against soft", gputest.DiffRect(metal, soft, metal.Bounds()))
	scale := soft.Bounds().Dx() / 480
	text := image.Rect(60*scale, 300*scale, 420*scale, 390*scale)
	if n := macDiff(soft, metal, text); n == text.Dx()*text.Dy() {
		t.Error("the text of the counter differs in every pixel")
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
