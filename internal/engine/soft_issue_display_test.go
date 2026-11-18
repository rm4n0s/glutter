//go:build linux && !android && glutter_display

package engine_test

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// softIssueChildEnv names the backend ("wayland" or "x11") that the child app of
// TestDisplayKeepsTheMipmapsOfAPicture uses; it is empty in a normal run.
const softIssueChildEnv = "GLUTTER_SOFT_ISSUE_CHILD"

// TestDisplaySoftIssueChild is the app of TestDisplayKeepsTheMipmapsOfAPicture: it draws the frames
// of the reproduction of step 49.1 (internal/issues, soft-mipmaps) with the software renderer of the
// Linux desktop, one frame each, and prints the lines of the report. It does nothing in a normal run.
func TestDisplaySoftIssueChild(t *testing.T) {
	if os.Getenv(softIssueChildEnv) == "" {
		t.Skip("run by TestDisplayKeepsTheMipmapsOfAPicture")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	scenes := issues.SoftMipmapsScenes()
	stats := engine.SoftStats{KeepFrames: true}
	var win platform.Window
	var end func()
	frame := 0
	cfg := engine.Config{
		Backend:  os.Getenv(softIssueChildEnv),
		Renderer: "soft",
		Window:   platform.WindowConfig{Title: "soft-mipmaps", AppID: "dev.glutter.softmipmaps", Size: ui.Size{Width: issues.SoftMipmapsWidth, Height: issues.SoftMipmapsHeight}},
		Hooks:    engine.Hooks{Window: func(w platform.Window, stop func()) { win, end = w, stop }},
		Start: func(d *ui.PlatformDispatcher) error {
			d.OnDrawFrame = func() {
				stats.Current = min(frame, len(scenes)-1)
				if err := d.View(win.ID()).Render(scenes[min(frame, len(scenes)-1)].Scene); err != nil {
					t.Errorf("Render: %v", err)
				}
				if frame++; frame >= len(scenes) {
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
	if len(stats.Mismatches) != 0 {
		t.Errorf("the buffer on display was not a whole redraw of the scene:\n%s", strings.Join(stats.Mismatches, "\n"))
	}
	// A frame of the reproduction is the last drawing of its scene: an X server has the window drawn
	// again after it is mapped, with the same scene.
	last := map[int]engine.SoftFrame{}
	for _, f := range stats.Kept {
		last[f.Index] = f
	}
	i := 0
	rep, err := issues.RunSoftMipmaps(func(*ui.Scene, int, int) (*image.RGBA, issues.SoftMipmapsStats, error) {
		f, ok := last[i]
		i++
		if !ok {
			return nil, issues.SoftMipmapsStats{}, nil
		}
		return f.Shown, issues.SoftMipmapsStats{Chains: f.Chains, Built: f.Built}, nil
	})
	if err != nil {
		sizes := []image.Rectangle{}
		for _, f := range stats.Kept {
			sizes = append(sizes, f.Shown.Rect)
		}
		t.Fatalf("%v (drawings of the sizes %v)", err, sizes)
	}
	fmt.Println("RENDERER: " + stats.Describe)
	for _, line := range rep.Lines() {
		fmt.Println(line)
	}
}

// TestDisplayKeepsTheMipmapsOfAPicture runs the reproduction of Fishball's issue of step 49.1 with
// the software renderer of the Linux desktop on a real display server, in an app of its own: on a
// Wayland compositor (scripts/headless-wayland.sh), and on an X server (scripts/headless-x11.sh) with
// MIT-SHM and with XPutImage. A spinner turns over a picture that is drawn at half its size: the
// picture's mipmap chain is made once and kept across the frames, and the buffer on display shows the
// picture and the spinner. Before the step every frame made the chain again.
func TestDisplayKeepsTheMipmapsOfAPicture(t *testing.T) {
	for _, tt := range []struct {
		name, backend, display string
		env                    []string
		path                   string
	}{
		{"wayland", "wayland", "WAYLAND_DISPLAY", nil, "wl_shm"},
		{"x11 with MIT-SHM", "x11", "DISPLAY", nil, "MIT-SHM"},
		{"x11 with XPutImage", "x11", "DISPLAY", []string{"GLUTTER_X11_NOSHM=1"}, "XPutImage"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if os.Getenv(tt.display) == "" {
				t.Skipf("%s is not set (run under scripts/headless-%s.sh)", tt.display, tt.backend)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run", "^TestDisplaySoftIssueChild$", "-test.count=1", "-test.v")
			cmd.WaitDelay = 5 * time.Second
			cmd.Env = append(os.Environ(), softIssueChildEnv+"="+tt.backend, "GLUTTER_BACKEND="+tt.backend)
			cmd.Env = append(cmd.Env, tt.env...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("the app failed: %v\n%s", err, out)
			}
			lines := strings.Split(string(out), "\n")
			rep, err := issues.ParseSoftMipmaps(lines)
			if err != nil {
				t.Fatal(err)
			}
			renderer := ""
			for _, l := range lines {
				if r, ok := strings.CutPrefix(l, "RENDERER: "); ok {
					renderer = r
				}
			}
			if !strings.Contains(renderer, tt.path) {
				t.Errorf("the renderer is %q, want a name with %s", renderer, tt.path)
			}
			for _, l := range rep.Lines() {
				t.Log(l)
			}
			for _, p := range rep.Problems() {
				t.Error(p)
			}
		})
	}
}
