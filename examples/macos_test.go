//go:build darwin && !ios && glutter_display

package examples

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/mactool"
	"github.com/rm4n0s/glutter/internal/perfstats"
)

// TestEveryExampleDrawsItsFirstFrameOnMacOS builds each example, `tray` and `multiwindow` included, as a signed bundle with
// glutter-macos's library (mactool) and runs the bundle's executable with Metal (GLUTTER_RENDERER=metal, so that nothing falls
// back to the software renderer) and waits for its first frame (step 33.15, SPEC §19, §29.13). The app is an accessory app, which
// never takes the focus, and its self-capture (proof method G) makes it end after the first idle frame: it must end with status 0,
// its log must have the engine's "first frame presented" line with the Metal renderer and no panic, the frame it saved must have
// more than one color, and `codesign --verify` must pass on the bundle. Run it with `go test -tags glutter_display -run
// TestEveryExampleDrawsItsFirstFrameOnMacOS ./examples`.
func TestEveryExampleDrawsItsFirstFrameOnMacOS(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and starts every example; skipped by -short")
	}
	names := allExamples(t)
	if len(names) < 11 {
		t.Fatalf("found %d examples, want at least 11: %v", len(names), names)
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			res, err := mactool.Build(ctx, mactool.BuildOptions{Debug: true, Package: "./examples/" + name, Dir: "..", Out: filepath.Join(dir, name+".app"), Name: name})
			if err != nil {
				t.Fatalf("glutter-macos build ./examples/%s: %v", name, err)
			}
			if err := mactool.Verify(ctx, res.Layout.App, mactool.SignOptions{}); err != nil {
				t.Errorf("codesign --verify: %v", err)
			}
			shot := filepath.Join(dir, name+".png")
			var out strings.Builder
			cmd := exec.CommandContext(ctx, res.Layout.Executable)
			cmd.Env = append(os.Environ(), "GLUTTER_RENDERER=metal", "GLUTTER_FRAME_STATS=1", "GLUTTER_MACOS_ACCESSORY=1", "GLUTTER_SCREENSHOT="+shot)
			cmd.Stdout, cmd.Stderr = &out, &out
			cmd.WaitDelay = 10 * time.Second
			if err := cmd.Run(); err != nil {
				t.Fatalf("the example: %v\n%s", err, out.String())
			}
			for _, bad := range []string{"panic:", "fatal error:", "goroutine "} {
				if strings.Contains(out.String(), bad) {
					t.Errorf("the log has %q:\n%s", bad, out.String())
				}
			}
			var first string
			for _, line := range strings.Split(out.String(), "\n") {
				if strings.Contains(line, "first frame presented") {
					first = line
					break
				}
			}
			if first == "" {
				t.Fatalf("the example logged no first frame:\n%s", out.String())
			}
			if r := perfstats.ParseLogLine(first)["renderer"]; !strings.HasPrefix(r, "metal") {
				t.Errorf("the first frame was drawn by %q, want metal: %s", r, first)
			}
			img, err := goldens.ReadPNG(shot)
			if err != nil {
				t.Fatalf("the example saved no frame: %v\n%s", err, out.String())
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
