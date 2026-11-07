//go:build linux && !android && glutter_display

package engine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// These tests run the software renderer of the Linux desktop on a real Wayland compositor
// (scripts/headless-wayland.sh, step 32.1): it draws into wl_shm buffers, and a blinking caret
// rasterizes and reports only its own tiles. The app is a process of its own, as in the other
// display tests. The checked renderer (engine.CheckSoftRenderer) compares the memory of the
// buffer on display with a whole redraw of each scene.

const softChildEnv = "GLUTTER_SOFT_CHILD"

const (
	softW, softH       = 480, 320
	softBigW, softBigH = 640, 400
	softFrames         = 16
	// softResizeAt is the frame after which the child asks for the larger window.
	softResizeAt = 9
)

// TestDisplaySoftChild is the app, started by TestDisplaySoftOnWayland; it prints what the
// renderer asked of the window as one JSON line. It does nothing in a normal run.
func TestDisplaySoftChild(t *testing.T) {
	if os.Getenv(softChildEnv) == "" {
		t.Skip("run by TestDisplaySoftOnWayland")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// A background that fills the window, bars like lines of text, and a caret that blinks: it is
	// in the scene every other frame, and the pictures of the other layers are the same in each.
	back := gputest.PictureOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 4000, 4000), gputest.FillPaint(0xFFFAFAFA))
	})
	bars := gputest.PictureOf(func(c *ui.Canvas) {
		for i := 0; i < 8; i++ {
			c.DrawRect(ui.NewRectFromLTWH(20, 20+float64(i)*30, 300-float64(i)*20, 12), gputest.FillPaint(0xFF3366CC))
		}
	})
	caret := gputest.PictureOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(400, 130, 2, 18), gputest.FillPaint(0xFF000000))
	})
	scene := func(frame int) *ui.Scene {
		b := ui.NewSceneBuilder()
		b.AddPicture(ui.Offset{}, back)
		b.AddPicture(ui.Offset{}, bars)
		if frame%2 == 0 {
			b.AddPicture(ui.Offset{}, caret)
		}
		return b.Build()
	}

	var stats engine.SoftStats
	var win platform.Window
	var end func()
	frame := 0
	cfg := engine.Config{
		Backend:  "wayland",
		Renderer: "soft",
		Window:   platform.WindowConfig{Title: "soft", AppID: "dev.glutter.soft", Size: ui.Size{Width: softW, Height: softH}},
		Hooks:    engine.Hooks{Window: func(w platform.Window, stop func()) { win, end = w, stop }},
		Start: func(d *ui.PlatformDispatcher) error {
			d.OnDrawFrame = func() {
				if err := d.View(win.ID()).Render(scene(frame)); err != nil {
					t.Errorf("Render: %v", err)
				}
				frame++
				switch {
				case frame == softResizeAt:
					d.Post(func() { win.SetSize(ui.Size{Width: softBigW, Height: softBigH}) })
				case frame >= softFrames:
					d.Post(end)
					return
				}
				d.ScheduleFrame()
			}
			return nil
		},
	}
	engine.CheckSoftRenderer(&cfg, &stats)
	if err := engine.Run(ctx, cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}
	line, err := json.Marshal(stats)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("SOFT-STATS: " + string(line))
}

func TestDisplaySoftOnWayland(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("there is no Wayland compositor (run under scripts/headless-wayland.sh)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run", "^TestDisplaySoftChild$", "-test.count=1", "-test.v")
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = append(os.Environ(), softChildEnv+"=1", "GLUTTER_BACKEND=wayland")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the app failed: %v\n%s", err, out)
	}
	var stats engine.SoftStats
	found := false
	for _, line := range strings.Split(string(out), "\n") {
		if rest, ok := strings.CutPrefix(line, "SOFT-STATS: "); ok {
			if err := json.Unmarshal([]byte(rest), &stats); err != nil {
				t.Fatalf("cannot read the stats %q: %v", rest, err)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("the app did not print its stats:\n%s", out)
	}
	t.Logf("%d frames, %d buffers acquired, presentations %v, sizes %v", stats.Frames, stats.Acquires, stats.Presents, stats.Sizes)

	if len(stats.Mismatches) != 0 {
		t.Errorf("the buffer on display was not a whole redraw of the scene:\n%s", strings.Join(stats.Mismatches, "\n"))
	}
	if stats.Frames < softFrames {
		t.Fatalf("%d frames were drawn, want %d", stats.Frames, softFrames)
	}
	whole := func(d []image.Rectangle, w, h int) bool { return len(d) == 1 && d[0] == image.Rect(0, 0, w, h) }
	// The first frame draws and reports the whole window.
	if !whole(stats.Presents[0], softW, softH) {
		t.Errorf("the first frame reported %v, want the whole window", stats.Presents[0])
	}
	// A blinking caret reports its own tiles and nothing else (two tiles of 64x64 at most, which a
	// frame may report twice over when the caret leaves one and enters the other); a frame that
	// changed nothing takes no buffer.
	small, grew := 0, false
	for i, d := range stats.Presents {
		if whole(d, softBigW, softBigH) {
			grew = true
			continue
		}
		if i == 0 || (stats.Frames > softResizeAt && i >= softResizeAt-1 && i <= softResizeAt+1) {
			continue // the first frame, and the frames around the resize
		}
		area := 0
		for _, r := range d {
			area += r.Dx() * r.Dy()
		}
		if len(d) > 0 && area <= 2*64*64 {
			small++
		} else if len(d) > 0 {
			t.Errorf("frame %d reported %v (%d pixels): more than the caret's tiles", i, d, area)
		}
	}
	if small < 4 {
		t.Errorf("only %d frames reported just the caret's tiles", small)
	}
	empty := 0
	for _, d := range stats.Presents {
		if len(d) == 0 {
			empty++
		}
	}
	if stats.Acquires != len(stats.Presents)-empty {
		t.Errorf("%d buffers acquired for %d presentations, %d of them without damage", stats.Acquires, len(stats.Presents), empty)
	}
	if !grew {
		t.Errorf("no whole frame at %dx%d (the window was resized) in %v", softBigW, softBigH, stats.Presents)
	}
}
