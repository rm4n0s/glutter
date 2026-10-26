//go:build linux && glutter_wine

package examples

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/perfstats"
	"github.com/rm4n0s/glutter/internal/winetest"
	"github.com/rm4n0s/glutter/internal/wintool"
)

// TestEveryExampleDrawsItsFirstFrameUnderWine starts each example, `tray` and `multiwindow` included, under Wine on a
// headless X server with Direct3D 12 (GLUTTER_RENDERER=d3d12, so that nothing falls back to the software renderer)
// and waits for its first frame (step 28.15, SPEC §19, §25.13). The app's self-capture (proof method F) makes it end
// after the first idle frame, so the run is over when the frame is drawn: it must end with status 0, its log must
// have the engine's "first frame presented" line with the d3d12 renderer and no panic, and the frame it saved must
// have more than one color. Run it with `go test -tags glutter_wine -p 1 -run TestEveryExampleDrawsItsFirstFrameUnderWine ./examples`,
// one package at a time: Wine's Direct3D 12 here is vkd3d on lavapipe, which takes about 700 MB for an app.
func TestEveryExampleDrawsItsFirstFrameUnderWine(t *testing.T) {
	if testing.Short() {
		t.Skip("starts every example under Wine; skipped by -short")
	}
	h, err := winetest.New("")
	if err != nil {
		t.Fatal(err)
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
	names := allExamples(t)
	if len(names) < 11 {
		t.Fatalf("found %d examples, want at least 11: %v", len(names), names)
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
			defer cancel()
			res, err := wintool.Build(ctx, wintool.ExecRunner{}, wintool.BuildOptions{
				Package: "./examples/" + name, Dir: h.Root, Out: filepath.Join(dir, name+".exe"),
			})
			if err != nil {
				t.Fatalf("glutter-windows build ./examples/%s: %v", name, err)
			}
			shot := filepath.Join(dir, name+".png")
			run, err := h.Run(ctx, res.Exes[0].Path, winetest.RunOptions{
				Env:     []string{"GLUTTER_RENDERER=d3d12", "GLUTTER_FRAME_STATS=1", "GLUTTER_SCREENSHOT=" + wintool.WineLauncher{}.ProgramPath(shot)},
				Timeout: 5 * time.Minute,
			})
			if err != nil || run.ExitCode != 0 {
				t.Fatalf("the example: %v, status %d\n%s", err, run.ExitCode, run.Output)
			}
			for _, bad := range []string{"panic:", "fatal error:", "goroutine "} {
				if strings.Contains(run.Output, bad) {
					t.Errorf("the log has %q:\n%s", bad, run.Output)
				}
			}
			var first string
			for _, line := range strings.Split(run.Output, "\n") {
				if strings.Contains(line, "first frame presented") {
					first = line
					break
				}
			}
			if first == "" {
				t.Fatalf("the example logged no first frame:\n%s", run.Output)
			}
			if r := perfstats.ParseLogLine(first)["renderer"]; !strings.HasPrefix(r, "d3d12") {
				t.Errorf("the first frame was drawn by %q, want d3d12: %s", r, first)
			}
			img, err := goldens.ReadPNG(shot)
			if err != nil {
				t.Fatalf("the example saved no frame: %v\n%s", err, run.Output)
			}
			colors := map[[4]uint8]bool{}
			b := img.Bounds()
			for y := b.Min.Y; y < b.Max.Y && len(colors) < 8; y += 3 {
				for x := b.Min.X; x < b.Max.X && len(colors) < 8; x += 3 {
					c := img.RGBAAt(x, y)
					colors[[4]uint8{c.R, c.G, c.B, c.A}] = true
				}
			}
			if len(colors) < 8 {
				t.Errorf("the frame (%v) has %d colors in a sample, want a drawn window", b, len(colors))
			}
			t.Logf("first frame: %s", first)
		})
	}
}
