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

// TestWineFirstFrameIsTheCounter builds the counter for Windows, runs it under Wine on a headless X
// server with the self-capture (proof method F, SPEC §25.13), and compares the frame with the soft
// golden of the counter. Run it with `go test -tags glutter_wine -p 1 ./examples/counter`.
//
// The golden has the text in Roboto; on Windows the Material text is Segoe UI, which the prefix of
// Wine does not have (it has a copy of Liberation Sans), so the comparison leaves out the text: the
// rest of the frame, the background and the floating action button, must equal the golden, and the
// text must be there.
func TestWineFirstFrameIsTheCounter(t *testing.T) {
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
	shot := filepath.Join(dir, "counter.png")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	res, err := h.Run(ctx, exe, winetest.RunOptions{Env: []string{"GLUTTER_SCREENSHOT=" + shot}, Timeout: time.Minute})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("the counter: %v, status %d\n%s", err, res.ExitCode, res.Output)
	}
	got, err := goldens.ReadPNG(shot)
	if err != nil {
		t.Fatalf("the counter saved no frame: %v\n%s", err, res.Output)
	}
	want, err := goldens.ReadPNG(filepath.Join("testdata", "goldens", "counter.png"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds() != want.Bounds() {
		t.Fatalf("the frame is %v, the golden %v", got.Bounds(), want.Bounds())
	}
	// The floating action button (and its shadow) and the corners are what the golden has.
	for _, r := range []image.Rectangle{image.Rect(390, 550, 480, 640), image.Rect(0, 0, 480, 10), image.Rect(0, 400, 380, 640)} {
		if d := diff(want, got, r); d != 0 {
			t.Errorf("%v differs from the golden in %d pixels", r, d)
		}
	}
	// The text, in the app bar and in the middle, is dark on the light background.
	for _, r := range []image.Rectangle{image.Rect(0, 0, 480, 56), image.Rect(60, 310, 420, 380)} {
		if dark := darkPixels(got, r); dark < 50 {
			t.Errorf("only %d dark pixels in %v: the text is missing", dark, r)
		}
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

// darkPixels counts the pixels of the rectangle that are dark.
func darkPixels(img *image.RGBA, r image.Rectangle) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if c := img.RGBAAt(x, y); int(c.R)+int(c.G)+int(c.B) < 300 {
				n++
			}
		}
	}
	return n
}

// TestWineSelfCaptureAcceptsEveryFormOfPath runs the counter under Wine with GLUTTER_SCREENSHOT as a
// path with a drive and backslashes, with a drive and forward slashes, a relative path (the working
// directory is the program's), a relative path in directories that are not there yet, and with a
// delay (GLUTTER_SCREENSHOT_AFTER); each run must save the frame (SPEC §25.5, §25.13). Run it with
// `go test -tags glutter_wine -p 1 ./examples/counter`.
func TestWineSelfCaptureAcceptsEveryFormOfPath(t *testing.T) {
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
	// The host's /tmp/x is Z:\tmp\x to a Windows program.
	windowsPath := func(hostPath string) string { return `Z:` + strings.ReplaceAll(hostPath, "/", `\`) }
	work := filepath.Join(dir, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		path  string
		after string
		file  string // where the frame must be, on the host
	}{
		{"a drive and backslashes", windowsPath(filepath.Join(dir, "drive.png")), "", filepath.Join(dir, "drive.png")},
		{"a drive and forward slashes", "Z:" + filepath.Join(dir, "slashes.png"), "", filepath.Join(dir, "slashes.png")},
		{"a relative path", "relative.png", "", filepath.Join(work, "relative.png")},
		{"a relative path in new directories", `sub\dir\nested.png`, "", filepath.Join(work, "sub", "dir", "nested.png")},
		{"a delay", windowsPath(filepath.Join(dir, "after.png")), "700ms", filepath.Join(dir, "after.png")},
	}
	for _, tt := range tests {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		env := []string{"GLUTTER_SCREENSHOT=" + tt.path}
		if tt.after != "" {
			env = append(env, "GLUTTER_SCREENSHOT_AFTER="+tt.after)
		}
		res, err := h.Run(ctx, exe, winetest.RunOptions{Env: env, Dir: work, Timeout: time.Minute})
		cancel()
		if err != nil || res.ExitCode != 0 {
			t.Errorf("%s: %v, status %d\n%s", tt.name, err, res.ExitCode, res.Output)
			continue
		}
		img, err := goldens.ReadPNG(tt.file)
		if err != nil {
			t.Errorf("%s: no frame at %s: %v\n%s", tt.name, tt.file, err, res.Output)
			continue
		}
		if b := img.Bounds(); b.Dx() < 480 || b.Dy() < 640 {
			t.Errorf("%s: the frame is %v", tt.name, b)
		}
		if darkPixels(img, image.Rect(60, 310, 420, 380)) < 50 {
			t.Errorf("%s: the text of the counter is missing", tt.name)
		}
	}
}
