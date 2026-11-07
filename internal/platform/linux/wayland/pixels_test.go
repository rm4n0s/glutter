//go:build linux && !android

package wayland

import (
	"context"
	"image"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

// pixelHandler is a handler whose frames run a script on the window, as the software renderer
// does: each OnFrame calls the next function with the window as a gpu.PixelTarget.
type pixelHandler struct {
	recHandler
	p *Platform

	fmu    sync.Mutex
	script []func(w *window) // one for each frame
	frames int
	got    []gpu.PixelBuffer
	errs   []error
}

func (h *pixelHandler) OnFrame(view ui.ViewID, _ time.Duration) {
	h.fmu.Lock()
	defer h.fmu.Unlock()
	h.frames++
	if len(h.script) == 0 {
		// A frame the script has no step for presents what it has, so that the frame callbacks go on.
		_ = h.p.windows[view].PresentPixels(nil)
		return
	}
	step := h.script[0]
	h.script = h.script[1:]
	step(h.p.windows[view])
}

func (h *pixelHandler) frameCount() int {
	h.fmu.Lock()
	defer h.fmu.Unlock()
	return h.frames
}

// frame is a script step: it acquires a buffer of the window's size, writes a byte into its first
// pixel, and presents it with the damage.
func (h *pixelHandler) frame(width, height int, mark byte, damage ...image.Rectangle) func(*window) {
	return func(w *window) {
		b, err := w.AcquirePixels(width, height)
		h.got = append(h.got, b)
		if err != nil {
			h.errs = append(h.errs, err)
			return
		}
		b.Pix[0] = mark
		h.errs = append(h.errs, w.PresentPixels(damage))
	}
}

// startPixelWindow connects a platform that does not fill its windows itself, as the engine does for
// a renderer, creates a window and starts Run with the handler.
func startPixelWindow(t *testing.T, fake fakecomp.Config, h *pixelHandler) (*windowEnv, *fakecomp.Server) {
	t.Helper()
	p, srv, err := connectWith(t, fake, Options{NoLibdecor: true})
	if err != nil {
		t.Fatal(err)
	}
	h.p = p
	h.allowClose = true
	if _, err := p.NewWindow(config640()); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()
	e := &windowEnv{t: t, p: p, srv: srv, h: &h.recHandler, done: done, cancel: cancel}
	t.Cleanup(func() {
		cancel()
		if !e.stopped {
			if err := <-done; err != nil {
				t.Errorf("Run returned %v", err)
			}
		}
	})
	return e, srv
}

func TestAWindowIsAPixelTarget(t *testing.T) {
	h := &pixelHandler{}
	full := image.Rect(0, 0, 640, 400)
	h.script = []func(*window){
		h.frame(640, 400, 1, full),                       // the first frame, whole
		h.frame(640, 400, 2, image.Rect(10, 20, 30, 40)), // the compositor still holds the first buffer
		h.frame(640, 400, 3, image.Rect(0, 0, 64, 64), image.Rect(128, 0, 192, 64)),
	}
	e, srv := startPixelWindow(t, fakecomp.Config{}, h)

	waitUntil(t, func() bool { return len(srv.RequestsWith("wl_surface.commit buffer=")) >= 1 }, "no frame was committed")
	// The first configure asks for the first frame; the next frames are asked for with RequestFrame.
	for i := 0; i < 2; i++ {
		want := i + 2
		e.onLoop(func() { e.p.windows[0].RequestFrame() })
		waitUntil(t, func() bool { return len(srv.RequestsWith("wl_surface.commit buffer=")) >= want }, "the frame was not committed")
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}

	lines := srv.Requests()
	for _, e := range h.errs {
		if e != nil {
			t.Errorf("a step failed: %v", e)
		}
	}
	// The first frame: a buffer was made at the size of the window (XRGB8888), attached, damaged
	// as the renderer said (not the whole buffer: the rectangle it gave), and committed.
	for _, want := range []string{
		"wl_shm_pool.create_buffer 640x400 offset=0 stride=2560 format=1",
		"wl_surface.attach",
		"wl_surface.damage_buffer 0,0 640x400",
		"wl_surface.damage_buffer 10,20 20x20",
		"wl_surface.damage_buffer 0,0 64x64",
		"wl_surface.damage_buffer 128,0 64x64",
	} {
		if index(lines, 0, want) < 0 {
			t.Errorf("%q is missing in:\n%s", want, strings.Join(lines, "\n"))
		}
	}
	// The frame callback is requested before the commit that it goes with.
	if i := index(lines, 0, "wl_surface.frame"); i < 0 || index(lines, i, "wl_surface.commit buffer=") < 0 {
		t.Errorf("no frame callback before the first commit in:\n%s", strings.Join(lines, "\n"))
	}
	if len(h.got) != 3 {
		t.Fatalf("%d frames were drawn, want 3", len(h.got))
	}
	// Ages: the first buffer is new; the second frame finds the first on display, so its buffer is new
	// too; the third reuses the first, which is two presentations old.
	for i, want := range []int{0, 0, 2} {
		if h.got[i].Age != want {
			t.Errorf("frame %d: age = %d, want %d", i, h.got[i].Age, want)
		}
		if h.got[i].Stride != 2560 || h.got[i].Format != gpu.PixelFormatBGRX8888 || len(h.got[i].Pix) != 2560*400 {
			t.Errorf("frame %d: stride %d, format %v, %d bytes", i, h.got[i].Stride, h.got[i].Format, len(h.got[i].Pix))
		}
	}
	if n := len(srv.RequestsWith("wl_shm_pool.create_buffer")); n != 2 {
		t.Errorf("%d buffers were made, want 2 (the pool reuses a buffer once it is released)", n)
	}
}

func TestAPixelWindowWithNothingToShowStillCommits(t *testing.T) {
	h := &pixelHandler{}
	h.script = []func(*window){h.frame(640, 400, 1, image.Rect(0, 0, 640, 400))}
	e, srv := startPixelWindow(t, fakecomp.Config{}, h)
	waitUntil(t, func() bool { return len(srv.RequestsWith("wl_surface.commit buffer=")) >= 1 }, "no frame was committed")
	srv.ClearRequests()

	// A frame that changed nothing presents with no buffer: a commit with no attach, so the
	// compositor answers with the frame callback and the next frame comes.
	e.onLoop(func() { e.p.windows[0].RequestFrame() })
	waitUntil(t, func() bool { return h.frameCount() >= 2 }, "the second frame did not come")
	waitUntil(t, func() bool { return len(srv.Requests()) >= 2 }, "nothing was sent for the frame")
	lines := srv.Requests()
	if index(lines, 0, "wl_surface.attach") >= 0 || index(lines, 0, "wl_surface.damage_buffer") >= 0 {
		t.Errorf("a frame with nothing to show attached or damaged:\n%s", strings.Join(lines, "\n"))
	}
	if index(lines, 0, "wl_surface.frame") < 0 || index(lines, 0, "wl_surface.commit") < 0 {
		t.Errorf("a frame with nothing to show sent no frame callback and commit:\n%s", strings.Join(lines, "\n"))
	}
	e.onLoop(func() { e.p.windows[0].RequestFrame() })
	waitUntil(t, func() bool { return h.frameCount() >= 3 }, "the third frame did not come: the frame callback was lost")
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestAPixelWindowResizesAndFreesTheBuffersOfTheOldSize(t *testing.T) {
	h := &pixelHandler{}
	h.script = []func(*window){
		h.frame(640, 400, 1, image.Rect(0, 0, 640, 400)),
		h.frame(800, 500, 2, image.Rect(0, 0, 800, 500)),
		h.frame(800, 500, 3, image.Rect(0, 0, 10, 10)),
	}
	e, srv := startPixelWindow(t, fakecomp.Config{}, h)
	waitUntil(t, func() bool { return len(srv.RequestsWith("wl_surface.commit buffer=")) >= 1 }, "no frame was committed")

	// The compositor resizes the window: the configure is acknowledged and a frame at the new size
	// is asked for at once.
	srv.ConfigureToplevel(fakecomp.ToplevelConfigure{Width: 800, Height: 500})
	waitUntil(t, func() bool { return len(srv.RequestsWith("wl_surface.commit buffer=")) >= 2 }, "the resized frame was not committed")
	e.onLoop(func() { e.p.windows[0].RequestFrame() })
	waitUntil(t, func() bool { return len(srv.RequestsWith("wl_surface.commit buffer=")) >= 3 }, "the third frame was not committed")
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
	for _, e := range h.errs {
		if e != nil {
			t.Errorf("a step failed: %v", e)
		}
	}
	lines := srv.Requests()
	if index(lines, 0, "wl_shm_pool.create_buffer 800x500 offset=0 stride=3200 format=1") < 0 {
		t.Errorf("no buffer of the new size in:\n%s", strings.Join(lines, "\n"))
	}
	if h.got[1].Age != 0 {
		t.Errorf("the first buffer of the new size has age %d, want 0", h.got[1].Age)
	}
	// The buffer of the old size is destroyed once the compositor released it (the commit of the
	// new size), and the window's own closing destroys the rest.
	if n := len(srv.RequestsWith("wl_buffer.destroy")); n < 1 {
		t.Errorf("no buffer was destroyed in:\n%s", strings.Join(lines, "\n"))
	}
}

func TestAPixelWindowShownAgainGetsABufferAfterTheConfigure(t *testing.T) {
	h := &pixelHandler{}
	h.script = []func(*window){
		h.frame(640, 400, 1, image.Rect(0, 0, 640, 400)),
		h.frame(640, 400, 2, image.Rect(0, 0, 640, 400)),
	}
	e, srv := startPixelWindow(t, fakecomp.Config{}, h)
	waitUntil(t, func() bool { return len(srv.RequestsWith("wl_surface.commit buffer=")) >= 1 }, "no frame was committed")
	e.onLoop(func() { e.p.windows[0].SetHidden(true) })
	waitUntil(t, func() bool { return len(srv.RequestsWith("wl_surface.commit buffer=null")) == 1 }, "the window was not unmapped")
	e.onLoop(func() { e.p.windows[0].SetHidden(false) })
	waitUntil(t, func() bool { return len(srv.RequestsWith("wl_surface.commit buffer=")) >= 3 }, "the window was not drawn after it was shown")
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
	for _, err := range h.errs {
		if err != nil {
			t.Errorf("a step failed: %v", err)
		}
	}
	lines := srv.Requests()
	// A buffer is attached only after the configure that answers the new role.
	hide := index(lines, 0, "wl_surface.commit buffer=null")
	ack := index(lines, hide, "xdg_surface.ack_configure")
	attach := index(lines, hide, "wl_surface.attach buffer=")
	if hide < 0 || ack < 0 || attach < 0 || attach < ack {
		t.Errorf("hide at %d, ack at %d, attach at %d: a buffer was attached before the configure in:\n%s", hide, ack, attach, strings.Join(lines, "\n"))
	}
}

func TestSeveralPixelWindowsEachHaveAPool(t *testing.T) {
	h := &pixelHandler{}
	p, srv, err := connectWith(t, fakecomp.Config{}, Options{NoLibdecor: true})
	if err != nil {
		t.Fatal(err)
	}
	h.p, h.allowClose = p, true
	for i := 0; i < 2; i++ {
		if _, err := p.NewWindow(config640()); err != nil {
			t.Fatal(err)
		}
	}
	a, b := p.windows[0], p.windows[1]
	ba, err := a.AcquirePixels(64, 64)
	if err != nil {
		t.Fatal(err)
	}
	bb, err := b.AcquirePixels(64, 64)
	if err != nil {
		t.Fatal(err)
	}
	if &ba.Pix[0] == &bb.Pix[0] {
		t.Error("two windows share a buffer")
	}
	if a.pixelPool == b.pixelPool || a.pixelPool == nil {
		t.Error("two windows share a pool, or the pool was not made")
	}
	_ = srv
}

func TestPixelBufferFailureRoute(t *testing.T) {
	h := &pixelHandler{}
	p, _, err := connectWith(t, fakecomp.Config{}, Options{NoLibdecor: true})
	if err != nil {
		t.Fatal(err)
	}
	h.p = p
	if _, err := p.NewWindow(config640()); err != nil {
		t.Fatal(err)
	}
	_, err = p.windows[0].AcquirePixels(0, 10)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("window.AcquirePixels->pixelPool.acquire->window.pixelPoolOf.func1->newShmBufferFormat.ShmBufferFailed") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}
