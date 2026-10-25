//go:build linux && glutter_wine

package main

import (
	"context"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/winetest"
)

// TestWineTwoWindowsAndADialogShareOneDirect3D12Device runs the example under Wine on a headless X server
// with the Direct3D 12 renderer, with a second window and a dialog that it opens by itself
// (GLUTTER_DEV_OPEN), and saves the root window of the server (proof method F, SPEC §25.13). The three
// windows are drawn, each in its own place (a window opens a step below the one before it, and the
// dialog is centered over the first window), and the log has one device and one renderer for all of them.
// Run it with `go test -tags glutter_wine -p 1 ./examples/multiwindow`.
func TestWineTwoWindowsAndADialogShareOneDirect3D12Device(t *testing.T) {
	h, err := winetest.New("")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "multiwindow.exe")
	build := exec.Command("go", "build", "-o", exe, ".")
	build.Dir = filepath.Join(h.Root, "examples", "multiwindow")
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
	shot := filepath.Join(dir, "root.png")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	res, err := h.Exec(ctx, winetest.Command{
		Argv: []string{filepath.Join(h.Root, "scripts", "wine-screenshot.sh"), shot, "--", filepath.Join(h.Root, "scripts", "wine-exec.sh"), exe},
		Env:  h.Environment("GLUTTER_DEV_OPEN=window,dialog", "GLUTTER_RENDERER=d3d12", "GLUTTER_SHOT_DELAY=8"),
		Dir:  h.Root,
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("the screenshot: %v, status %d\n%s", err, res.ExitCode, res.Output)
	}
	if n := strings.Count(res.Output, "Direct3D 12 device"); n != 1 {
		t.Errorf("%d Direct3D 12 devices were opened for three windows, want 1\n%s", n, res.Output)
	}
	if n := strings.Count(res.Output, "renderer chosen"); n != 1 {
		t.Errorf("%d renderers were chosen, want 1\n%s", n, res.Output)
	}
	for _, bad := range []string{"ERROR", "panic", "WARN"} {
		if strings.Contains(res.Output, bad) {
			t.Errorf("the log has %q:\n%s", bad, res.Output)
		}
	}
	img, err := goldens.ReadPNG(shot)
	if err != nil {
		t.Fatal(err)
	}
	// The server's root is black where no window is, and with no window manager the windows have no frame:
	// the first window is 420 by 520 at the corner, the second one a step (32 pixels) below and to its
	// right, and the dialog (320 by 180, with its message at its middle) is centered over the first window.
	// Sample a place only the first window covers, one only the second covers, and the dialog's message.
	bounds := img.Bounds()
	if bounds.Dx() < 600 || bounds.Dy() < 600 {
		t.Fatalf("the root window is %v", bounds)
	}
	lit := func(name string, x, y int) {
		t.Helper()
		if c := img.RGBAAt(x, y); c == (color.RGBA{A: 255}) || c == (color.RGBA{}) {
			t.Errorf("%s: the pixel %d,%d is the empty root %v", name, x, y, c)
		}
	}
	lit("the first window, left of the second", 10, 400)
	lit("the second window, right of the first", 440, 400)
	if n := darkPixels(img, image.Rect(60, 250, 370, 270)); n < 50 {
		t.Errorf("only %d dark pixels where the dialog's message is: the dialog is not centered over the first window", n)
	}
}

// darkPixels counts the dark pixels of the rectangle.
func darkPixels(img *image.RGBA, r image.Rectangle) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if c := img.RGBAAt(x, y); int(c.R)+int(c.G)+int(c.B) < 300 && c.A != 0 {
				n++
			}
		}
	}
	return n
}
