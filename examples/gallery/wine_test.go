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
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/winetest"
)

// TestWineGalleryWithD3D12EqualsSoft builds the gallery for Windows and runs it under Wine on a headless X
// server with the self-capture (proof method F, SPEC §25.13): with nothing asked for it is drawn by d3d12 (the
// log says so), and its first page equals the one that GLUTTER_RENDERER=soft draws within the tolerance of the
// GPU renderers. The first frame under vkd3d compiles every pipeline, so the capture waits for it. Run it with
// `go test -tags glutter_wine -p 1 ./examples/gallery`.
func TestWineGalleryWithD3D12EqualsSoft(t *testing.T) {
	h, err := winetest.New("")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "gallery.exe")
	build := exec.Command("go", "build", "-o", exe, "./examples/gallery")
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
	capture := func(name string, env ...string) (*image.RGBA, string) {
		t.Helper()
		shot := filepath.Join(dir, name+".png")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		res, err := h.Run(ctx, exe, winetest.RunOptions{
			Env:     append([]string{"GLUTTER_SCREENSHOT=" + shot, "GLUTTER_SCREENSHOT_AFTER=2s"}, env...),
			Timeout: 4 * time.Minute,
		})
		if err != nil || res.ExitCode != 0 {
			t.Fatalf("%s: %v, status %d\n%s", name, err, res.ExitCode, res.Output)
		}
		img, err := goldens.ReadPNG(shot)
		if err != nil {
			t.Fatalf("%s saved no frame: %v\n%s", name, err, res.Output)
		}
		return img, res.Output
	}
	auto, autoLog := capture("auto")
	if !strings.Contains(autoLog, `renderer chosen renderer="d3d12:`) {
		t.Errorf("the log does not say that d3d12 was chosen:\n%s", autoLog)
	}
	soft, softLog := capture("soft", "GLUTTER_RENDERER=soft")
	if !strings.Contains(softLog, `renderer chosen renderer="soft: GDI`) {
		t.Errorf("GLUTTER_RENDERER=soft did not choose soft:\n%s", softLog)
	}
	if auto.Bounds() != soft.Bounds() || auto.Bounds().Dx() < 1100 || auto.Bounds().Dy() < 760 {
		t.Fatalf("the frames are %v (d3d12) and %v (soft), want 1100x760 or more", auto.Bounds(), soft.Bounds())
	}
	if dir := os.Getenv("GLUTTER_TEST_DUMP"); dir != "" {
		for name, img := range map[string]*image.RGBA{"gallery-d3d12": auto, "gallery-soft": soft} {
			if err := goldens.WritePNG(filepath.Join(dir, name+".png"), img); err != nil {
				t.Errorf("dump: %v", err)
			}
		}
	}
	gputest.Tolerance{Over8Permille: 30, Over32Permille: 3}.Check(t, "gallery, d3d12 against soft", gputest.DiffRect(auto, soft, auto.Bounds()))
}
