//go:build darwin && glutter_ios

package examples

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/iostest"
	"github.com/rm4n0s/glutter/internal/iostool"
)

// TestEveryExampleDrawsItsFirstFrameOnIOS builds each example for the Simulator as a signed bundle with glutter-ios's library
// (iostool), installs it on an iPhone and runs it with Metal (GLUTTER_RENDERER=metal, so that nothing falls back to the software
// renderer); the engine's self-capture saves the first frame in the app's container and the app ends (step 35.9, SPEC §19, §30.11).
// `multiwindow` and `tray` are left out: an iPhone has one window and no tray (SPEC §30.1). The app's log must say that Metal drew,
// have no panic, and the frame must be as large as a phone's portrait screen and have more than one color. Run it with
// `go test -p 1 -tags glutter_ios -run TestEveryExampleDrawsItsFirstFrameOnIOS ./examples`.
func TestEveryExampleDrawsItsFirstFrameOnIOS(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and starts every example on the Simulator; skipped by -short")
	}
	names := allExamples(t)
	if len(names) < 11 {
		t.Fatalf("found %d examples, want at least 11: %v", len(names), names)
	}
	h := iostest.New(t)
	for _, name := range names {
		if name == "multiwindow" || name == "tray" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
			defer cancel()
			res, err := iostool.Build(ctx, iostool.BuildOptions{
				Debug:   true,
				Package: "./examples/" + name, Dir: "..", Out: filepath.Join(t.TempDir(), name+".app"), Simulator: true, ID: "dev.glutter.example." + name,
			})
			if err != nil {
				t.Fatalf("glutter-ios build ./examples/%s: %v", name, err)
			}
			dev, err := h.Device(ctx, "iphone")
			if err != nil {
				t.Fatalf("device: %v", err)
			}
			if err := h.Simctl.Install(ctx, dev.UDID, res.Layout.App); err != nil {
				t.Fatalf("install: %v", err)
			}
			container, err := h.Simctl.Container(ctx, dev.UDID, res.Info.ID)
			if err != nil {
				t.Fatalf("container: %v", err)
			}
			shot := filepath.Join(container, "Documents", "first-frame.png")
			_ = os.Remove(shot)
			app, err := h.Launch(ctx, dev.UDID, res.Info.ID, "GLUTTER_RENDERER=metal", "GLUTTER_SCREENSHOT="+shot)
			if err != nil {
				t.Fatalf("launch: %v", err)
			}
			defer app.Stop()
			var colors map[[3]uint32]bool
			var w, hgt int
			deadline := time.Now().Add(3 * time.Minute)
			for colors == nil {
				if f, err := os.Open(shot); err == nil {
					img, err := png.Decode(f)
					_ = f.Close()
					if err == nil {
						b := img.Bounds()
						w, hgt = b.Dx(), b.Dy()
						colors = map[[3]uint32]bool{}
						for y := b.Min.Y; y < b.Max.Y && len(colors) < 8; y += 5 {
							for x := b.Min.X; x < b.Max.X && len(colors) < 8; x += 5 {
								r, g, bl, _ := img.At(x, y).RGBA()
								colors[[3]uint32{r, g, bl}] = true
							}
						}
						break
					}
				}
				if time.Now().After(deadline) {
					t.Fatalf("the example saved no frame; output:\n%s", strings.Join(app.Output(), "\n"))
				}
				time.Sleep(200 * time.Millisecond)
			}
			out := strings.Join(app.Output(), "\n")
			for _, bad := range []string{"panic:", "fatal error:", "goroutine "} {
				if strings.Contains(out, bad) {
					t.Errorf("the log has %q:\n%s", bad, out)
				}
			}
			if !strings.Contains(out, `renderer="metal`) {
				t.Errorf("metal was not the renderer; output:\n%s", out)
			}
			if w >= hgt || w < 600 {
				t.Errorf("the frame is %dx%d: not a phone's portrait screen", w, hgt)
			}
			if len(colors) < 8 {
				t.Errorf("the frame has %d colors in a sample, want a drawn window", len(colors))
			}
		})
	}
}
