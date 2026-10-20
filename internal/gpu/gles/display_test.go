//go:build glutter_display

package gles_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland"
	"github.com/rm4n0s/glutter/ui"
)

// This test needs a Wayland compositor and a working EGL driver. Run it under the
// headless compositor:
//
//	scripts/headless-wayland.sh -- go test -race -tags glutter_display ./internal/gpu/gles/

// drawHandler draws one frame with OpenGL ES when the compositor asks for it.
type drawHandler struct {
	platform.NopHandler
	win  platform.Window
	done chan<- result
	w, h int
	ran  bool
}

type result struct {
	version string
	w, h    int
	pixels  []uint8 // RGBA, top row first
	err     error
}

func (d *drawHandler) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) {
	d.w, d.h = int(m.PhysicalSize.Width), int(m.PhysicalSize.Height)
}

func (d *drawHandler) OnFrame(ui.ViewID, time.Duration) {
	// The window draws again when its scale becomes known; one frame is enough here.
	if d.ran {
		return
	}
	d.ran = true
	h := d.win.NativeSurface()
	if h.Kind != gpu.SurfaceKindWayland {
		d.done <- result{err: errors.NewErrf("TestSurfaceKind", "surface kind = %v", h.Kind)}
		return
	}
	c, s, err := gles.Open(h)
	if err != nil {
		d.done <- result{err: err}
		return
	}
	res := result{version: c.Version(), w: d.w, h: d.h}
	c.Viewport(d.w, d.h)
	c.Clear(0x67/255.0, 0x50/255.0, 0xA4/255.0, 1)
	img, err := c.ReadPixels(d.w, d.h)
	if err == nil {
		res.pixels = img.Pix
		err = c.SwapBuffers(s)
	}
	res.err = err
	s.Destroy()
	c.Close()
	d.done <- res
}

func TestDisplayClearsTheWindowToPurple(t *testing.T) {
	p, err := wayland.New("")
	if err != nil {
		t.Fatalf("could not connect to the compositor: %v (run under scripts/headless-wayland.sh)", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	cfg := platform.DefaultWindowConfig()
	cfg.Title = "gles display test"
	cfg.Size = ui.Size{Width: 200, Height: 100}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan result, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runErr := make(chan error, 1)
	go func() {
		runtime.LockOSThread() // an OpenGL context belongs to the thread that made it current
		defer runtime.UnlockOSThread()
		runErr <- p.Run(ctx, &drawHandler{win: win, done: done})
	}()

	var res result
	select {
	case res = <-done:
	case <-ctx.Done():
		t.Fatal("no frame was drawn within 20 seconds")
	}
	cancel()
	if err := <-runErr; err != nil {
		t.Errorf("Run: %v", err)
	}
	if res.err != nil {
		if appErr, ok := errors.FromError(res.err); ok {
			t.Fatalf("%s: %s %v", appErr.Tag, appErr.Message, appErr.Metadata)
		}
		t.Fatal(res.err)
	}
	t.Logf("OpenGL ES version: %s", res.version)
	if res.w <= 0 || res.h <= 0 || len(res.pixels) != res.w*res.h*4 {
		t.Fatalf("read %d bytes for a %dx%d window", len(res.pixels), res.w, res.h)
	}
	for i := 0; i < len(res.pixels); i += 4 {
		got := [4]uint8{res.pixels[i], res.pixels[i+1], res.pixels[i+2], res.pixels[i+3]}
		if want := [4]uint8{0x67, 0x50, 0xA4, 0xFF}; got != want {
			t.Fatalf("pixel %d (x=%d y=%d) = %v, want %v", i/4, i/4%res.w, i/4/res.w, got, want)
		}
	}
}
