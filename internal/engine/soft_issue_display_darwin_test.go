//go:build darwin && !ios && glutter_display

package engine_test

import (
	"context"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/mainthread"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// TestDisplayKeepsTheMipmapsOfAPicture runs the reproduction of Fishball's issue of step 49.1 (internal/issues, soft-mipmaps) with
// the software renderer of macOS (shmwindow into the IOSurfaces of the window), one frame of the reproduction each, as the Linux
// test does on Wayland and X11 (step 50.1). A spinner turns over a picture that is drawn at half its size: the picture's mipmap
// chain is made once and kept across the frames, and the buffer on display shows the picture and the spinner.
func TestDisplayKeepsTheMipmapsOfAPicture(t *testing.T) {
	scenes := issues.SoftMipmapsScenes()
	stats := engine.SoftStats{KeepFrames: true}
	var (
		win   platform.Window
		end   func()
		frame int
		scale float64
	)
	// The platform loads AppKit when the engine starts; the screen's scale is needed before, for the window's size.
	if _, err := dl.OpenFramework("AppKit"); err != nil {
		t.Fatalf("AppKit: %v", err)
	}
	mainthread.Do(func() {
		scale = objc.Send[float64](objc.ID(objcx.MustClass("NSScreen")).Send(objcx.Sel("mainScreen")), objcx.Sel("backingScaleFactor"))
	})
	if scale <= 0 {
		scale = 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cfg := engine.Config{
		Getenv:   macEnv,
		Renderer: "soft",
		// The window's size is in points: a frame of the reproduction is 640x480 pixels on a screen of any scale.
		Window: platform.WindowConfig{Title: "soft-mipmaps", Size: ui.Size{Width: issues.SoftMipmapsWidth / scale, Height: issues.SoftMipmapsHeight / scale}},
		Hooks:  engine.Hooks{Window: func(w platform.Window, stop func()) { win, end = w, stop }},
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
	var err error
	mainthread.Do(func() { err = engine.Run(ctx, cfg) })
	if err != nil {
		t.Fatalf("Run: %+v", err)
	}
	if len(stats.Mismatches) != 0 {
		t.Errorf("the buffer on display was not a whole redraw of the scene:\n%s", strings.Join(stats.Mismatches, "\n"))
	}
	// A frame of the reproduction is the last drawing of its scene: the window may be drawn again with the same scene (a change of
	// its metrics).
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
		t.Fatalf("%v (drawings of the sizes %v, screen scale %g)", err, sizes, scale)
	}
	t.Logf("renderer: %s", stats.Describe)
	for _, line := range rep.Lines() {
		t.Log(line)
	}
	for _, p := range rep.Problems() {
		t.Error(p)
	}
}
