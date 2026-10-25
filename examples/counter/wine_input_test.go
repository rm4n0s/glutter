//go:build linux && glutter_wine

package main

import (
	"context"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/winetest"
)

// TestWineThreeClicksCountThree builds the counter for Windows and clicks its floating action button
// with the scripted events of GLUTTER_DEV_EVENTS, which send the window messages of a real mouse
// (SPEC §25.6, §25.13). The frame after three clicks must show what the frame of an app that starts
// at 3 shows (GLUTTER_DEV_TAPS=3), in the text of the counter, pixel for pixel; one and two clicks
// show other frames. Run it with `go test -tags glutter_wine -p 1 ./examples/counter`.
func TestWineThreeClicksCountThree(t *testing.T) {
	h, err := winetest.New("")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "counter.exe")
	build := exec.Command("go", "build", "-o", exe, "./examples/counter")
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
		res, err := h.Run(ctx, exe, winetest.RunOptions{Env: append([]string{"GLUTTER_SCREENSHOT=" + shot, "GLUTTER_SCREENSHOT_AFTER=1500ms"}, env...), Timeout: time.Minute})
		if err != nil || res.ExitCode != 0 {
			t.Fatalf("%s: %v, status %d\n%s", name, err, res.ExitCode, res.Output)
		}
		img, err := goldens.ReadPNG(shot)
		if err != nil {
			t.Fatalf("%s saved no frame: %v\n%s", name, err, res.Output)
		}
		return img
	}
	// The floating action button is at the bottom right of the 480 x 640 window.
	const fabX, fabY = 436, 596
	clicks := func(n int) string {
		var steps []string
		for range n {
			steps = append(steps, "tap:"+strconv.Itoa(fabX)+","+strconv.Itoa(fabY), "wait:150ms")
		}
		return "GLUTTER_DEV_EVENTS=" + strings.Join(steps, ";")
	}
	text := image.Rect(60, 300, 420, 390) // "You have pushed ..." and the count
	three := run("three-taps", "GLUTTER_DEV_TAPS=3")
	zero := run("zero")
	if d := diff(zero, three, text); d == 0 {
		t.Fatal("the text of the counter is the same at 0 and at 3")
	}
	for n, wantSame := range map[int]bool{1: false, 2: false, 3: true} {
		got := run("clicks-"+strconv.Itoa(n), clicks(n))
		d := diff(three, got, text)
		if wantSame && d != 0 {
			t.Errorf("after 3 clicks the text differs from the counter at 3 in %d pixels", d)
		}
		if !wantSame && d == 0 {
			t.Errorf("after %d clicks the text is the one of 3", n)
		}
	}
}
