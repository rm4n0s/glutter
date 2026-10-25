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

// TestWineWheelScrollsTheView builds the scrolling example for Windows and turns the wheel over its
// CustomScrollView tab with the scripted events of GLUTTER_DEV_EVENTS, which send WM_MOUSEWHEEL as
// a real wheel does (SPEC §25.6, §25.13). A wheel of 400 logical pixels (four notches of the 100
// pixels that three lines make) must leave the view where GLUTTER_DEV_SCROLL=400 puts it, pixel for
// pixel in the content; a wheel of 800 leaves it elsewhere. Run it with
// `go test -tags glutter_wine -p 1 ./examples/scrolling`.
func TestWineWheelScrollsTheView(t *testing.T) {
	h, err := winetest.New("")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "scrolling.exe")
	build := exec.Command("go", "build", "-o", exe, "./examples/scrolling")
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
		res, err := h.Run(ctx, exe, winetest.RunOptions{Env: append([]string{"GLUTTER_SCREENSHOT=" + shot, "GLUTTER_SCREENSHOT_AFTER=2s", "GLUTTER_DEV_TAB=2"}, env...), Timeout: time.Minute})
		if err != nil || res.ExitCode != 0 {
			t.Fatalf("%s: %v, status %d\n%s", name, err, res.ExitCode, res.Output)
		}
		img, err := goldens.ReadPNG(shot)
		if err != nil {
			t.Fatalf("%s saved no frame: %v\n%s", name, err, res.Output)
		}
		return img
	}
	top := run("top")
	b := top.Bounds()
	// The content under the tabs, without the column of the scrollbar.
	content := image.Rect(0, 120, b.Dx()-24, b.Dy())
	jumped := run("jumped", "GLUTTER_DEV_SCROLL=400")
	wheeled := run("wheeled", "GLUTTER_DEV_EVENTS=move:300,300;wheel:300,300,0,400;wait:200ms")
	farther := run("farther", "GLUTTER_DEV_EVENTS=move:300,300;wheel:300,300,0,800;wait:200ms")
	if diff(top, jumped, content) == 0 {
		t.Fatal("GLUTTER_DEV_SCROLL=400 left the view where it was")
	}
	if d := diff(jumped, wheeled, content); d != 0 {
		t.Errorf("after a wheel of 400 pixels the content differs from the view at offset 400 in %d pixels", d)
	}
	if d := diff(jumped, farther, content); d == 0 {
		t.Error("a wheel of 800 pixels left the view where 400 does")
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
