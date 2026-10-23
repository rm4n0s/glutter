//go:build linux && !android

package wayland

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

// These tests cover the EGL mode of a window (everything but the GPU) against
// the fake compositor: the native window, the order of the configure
// handshake, and frame pacing. The GPU itself is tested in internal/gpu/gles.

// frameHandler is a handler that stands in for the renderer: on every frame it
// "swaps", that is commits the surface, which is what eglSwapBuffers does.
type frameHandler struct {
	platform.NopHandler
	p *Platform

	mu      sync.Mutex
	metrics []platform.ViewMetrics
	frames  []time.Duration
	// configures is the number of acknowledged configures at each frame.
	configures []int
	// settings has the settings of each OnSettings.
	settings []platform.Settings
	// displays has what Displays returned at each OnDisplaysChanged.
	displays [][]platform.Display
	// onFrame, when set, runs inside OnFrame before the commit (on the UI thread).
	onFrame func()
}

func (h *frameHandler) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metrics = append(h.metrics, m)
}

func (h *frameHandler) OnSettings(s platform.Settings) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.settings = append(h.settings, s)
}

func (h *frameHandler) settingsSeen() []platform.Settings {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]platform.Settings(nil), h.settings...)
}

func (h *frameHandler) OnDisplaysChanged() {
	d := h.p.Displays()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.displays = append(h.displays, d)
}

// lastDisplays returns what Displays returned the last time the outputs changed.
func (h *frameHandler) lastDisplays() []platform.Display {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.displays) == 0 {
		return nil
	}
	return h.displays[len(h.displays)-1]
}

func (h *frameHandler) OnFrame(view ui.ViewID, t time.Duration) {
	if h.onFrame != nil {
		h.onFrame()
	}
	h.p.windows[view].surface.Commit() // the swap
	h.mu.Lock()
	h.frames = append(h.frames, t)
	h.configures = append(h.configures, h.p.windows[view].configures)
	h.mu.Unlock()
}

func (h *frameHandler) frameCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.frames)
}

func (h *frameHandler) metricsSeen() []platform.ViewMetrics {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]platform.ViewMetrics(nil), h.metrics...)
}

// resizeCall is one wl_egl_window_resize and how many configures had been
// acknowledged when it was made.
type resizeCall struct {
	width, height int32
	configures    int
}

type eglEnv struct {
	mu       sync.Mutex
	resizes  []resizeCall
	t        *testing.T
	p        *Platform
	srv      *fakecomp.Server
	h        *frameHandler
	win      platform.Window
	finished chan struct{} // closed when Run has returned
	runErr   error         // Run's result; valid once finished is closed
	cancel   context.CancelFunc
}

func startEGLWindow(t *testing.T, hold bool, toplevel fakecomp.ToplevelConfigure) *eglEnv {
	t.Helper()
	return startEGLWindowWith(t, fakecomp.Config{HoldFrames: hold, Toplevel: toplevel})
}

// startEGLWindowWith starts a window against a fake compositor with the config. The
// window's calls of wl_egl_window_resize are recorded.
func startEGLWindowWith(t *testing.T, cfg fakecomp.Config) *eglEnv {
	t.Helper()
	p, srv, err := connectWith(t, cfg, Options{NoLibdecor: true})
	if err != nil {
		t.Fatal(err)
	}
	win, err := p.NewWindow(config640())
	if err != nil {
		var appErr *errors.Error
		if ok := errors.As(err, &appErr); ok && appErr.Tag == tagLibraryLoadFailed {
			t.Skipf("libwayland-egl is not available: %v", err)
		}
		t.Fatal(err)
	}
	h := &frameHandler{p: p}
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	e := &eglEnv{t: t, p: p, srv: srv, h: h, win: win, finished: make(chan struct{}), cancel: cancel}
	w := win.(*window)
	real := w.egl
	w.egl = &waylandEGL{
		create: real.create,
		resize: func(native uintptr, width, height, dx, dy int32) {
			e.mu.Lock()
			e.resizes = append(e.resizes, resizeCall{width, height, w.configures})
			e.mu.Unlock()
			real.resize(native, width, height, dx, dy)
		},
		destroy: real.destroy,
	}
	go func() {
		e.runErr = p.Run(ctx, h)
		close(e.finished)
	}()
	t.Cleanup(func() {
		cancel()
		<-e.finished
	})
	return e
}

// frameDone waits until the compositor holds a frame callback, which means it
// has received the commit, and then answers it.
func (e *eglEnv) frameDone() {
	e.t.Helper()
	waitUntil(e.t, func() bool { return e.srv.HeldFrames() > 0 }, "the compositor never received a frame callback")
	e.srv.FrameDone()
}

func (e *eglEnv) onLoop(fn func()) {
	e.t.Helper()
	finished := make(chan struct{})
	e.p.Post(func() { fn(); close(finished) })
	select {
	case <-finished:
	case <-time.After(testTimeout):
		e.t.Fatal("the loop did not run a posted function")
	}
}

func TestEGLWindowHandshakeOrder(t *testing.T) {
	e := startEGLWindow(t, true, fakecomp.ToplevelConfigure{})
	waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "the first frame was never requested")

	// Metrics first, then the frame; the frame callback is requested before the
	// commit that the swap makes, and the configure is acknowledged before that
	// commit as well.
	lines := e.srv.Requests()
	order := []string{
		`xdg_toplevel.set_title "Glutter dev window"`,
		"wl_surface.commit",
		"xdg_surface.ack_configure",
		"wl_surface.frame",
		"wl_surface.commit",
	}
	at := 0
	for _, want := range order {
		i := index(lines, at, want)
		if i < 0 {
			t.Fatalf("%q is missing, or comes too early, in:\n%s", want, strings.Join(lines, "\n"))
		}
		at = i + 1
	}
	if strings.Contains(strings.Join(lines, "\n"), "wl_shm.create_pool") {
		t.Error("an EGL window created a shared-memory pool")
	}
	if m := e.h.metricsSeen(); len(m) != 1 || m[0].PhysicalSize != (ui.Size{Width: 640, Height: 400}) {
		t.Errorf("metrics = %+v, want one report of 640x400", m)
	}

	var handle gpu.SurfaceHandle
	e.onLoop(func() { handle = e.win.NativeSurface() })
	if handle.Kind != gpu.SurfaceKindWayland || handle.Window == 0 || handle.Surface == 0 || handle.Display != e.p.sess.conn.DisplayPtr() {
		t.Errorf("NativeSurface() = %+v, want a Wayland handle with the connection's display, a window and a surface", handle)
	}
}

func TestEGLConfigureRendersWithoutWaitingForTheFrameCallback(t *testing.T) {
	e := startEGLWindow(t, true, fakecomp.ToplevelConfigure{})
	waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "the first frame was never requested")
	if e.srv.HeldFrames() != 1 {
		t.Fatalf("held frame callbacks = %d, want the one of the first frame", e.srv.HeldFrames())
	}

	// The first frame is still in flight (the compositor holds its callback),
	// and a configure arrives: the new size is rendered at once.
	e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{Width: 300, Height: 200})
	waitUntil(t, func() bool { return e.h.frameCount() == 2 }, "a configure did not render a frame while one was in flight")

	if m := e.h.metricsSeen(); len(m) != 2 || m[1].PhysicalSize != (ui.Size{Width: 300, Height: 200}) {
		t.Errorf("metrics = %+v, want 640x400 and then 300x200", m)
	}
	lines := e.srv.Requests()
	// The second configure's ack comes before its commit.
	ack := index(lines, index(lines, 0, "xdg_surface.ack_configure")+1, "xdg_surface.ack_configure")
	commit := index(lines, ack, "wl_surface.commit")
	if ack < 0 || commit < 0 {
		t.Fatalf("the second ack and its commit are missing in:\n%s", strings.Join(lines, "\n"))
	}
	// Only one callback was ever requested: one in flight at a time.
	if n := len(e.srv.RequestsWith("wl_surface.frame")); n != 1 {
		t.Errorf("%d frame callbacks requested, want 1 (at most one in flight)", n)
	}
}

func TestEGLFramePacing(t *testing.T) {
	e := startEGLWindow(t, true, fakecomp.ToplevelConfigure{})
	waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "the first frame was never requested")

	// Three requests while the first frame is in flight become one frame, which
	// waits for the compositor.
	e.onLoop(func() {
		e.win.RequestFrame()
		e.win.RequestFrame()
		e.win.RequestFrame()
	})
	time.Sleep(50 * time.Millisecond)
	if n := e.h.frameCount(); n != 1 {
		t.Fatalf("%d frames before the compositor was ready, want 1", n)
	}

	e.frameDone()
	waitUntil(t, func() bool { return e.h.frameCount() == 2 }, "the merged request did not render after the frame callback")
	time.Sleep(50 * time.Millisecond)
	if n := e.h.frameCount(); n != 2 {
		t.Errorf("%d frames, want exactly 2: the three requests merge into one", n)
	}

	// With nothing requested, the next callback renders nothing.
	e.frameDone()
	time.Sleep(50 * time.Millisecond)
	if n := e.h.frameCount(); n != 2 {
		t.Errorf("a frame callback without a request rendered a frame (%d frames)", n)
	}

	// Nothing is in flight now, so a request renders on the next pass of the
	// loop, and not inside RequestFrame.
	var insideCall int
	e.onLoop(func() {
		e.win.RequestFrame()
		insideCall = e.h.frameCount()
	})
	if insideCall != 2 {
		t.Errorf("RequestFrame rendered inside the call (%d frames); it must wait for the loop", insideCall)
	}
	waitUntil(t, func() bool { return e.h.frameCount() == 3 }, "a request with no frame in flight was not rendered")

	// An animation: the handler asks for the next frame in every frame.
	e.h.onFrame = func() { e.win.RequestFrame() }
	e.frameDone() // frame 3 is done; nothing was requested, so nothing happens
	e.onLoop(func() { e.win.RequestFrame() })
	waitUntil(t, func() bool { return e.h.frameCount() == 4 }, "the animation did not start")
	for want := 5; want <= 8; want++ {
		e.frameDone()
		waitUntil(t, func() bool { return e.h.frameCount() == want }, "the animation did not continue after a frame callback")
	}
	if n := e.srv.HeldFrames(); n != 1 {
		t.Errorf("%d callbacks held, want 1: one frame in flight at a time", n)
	}
}

func TestEGLWindowCloseDestroysTheNativeWindow(t *testing.T) {
	e := startEGLWindow(t, false, fakecomp.ToplevelConfigure{})
	waitUntil(t, func() bool { return e.h.frameCount() >= 1 }, "no frame")
	e.onLoop(func() { e.win.Close() })
	select {
	case <-e.finished:
		if e.runErr != nil {
			t.Fatalf("Run returned %v", e.runErr)
		}
	case <-time.After(testTimeout):
		t.Fatal("Run did not return after the last window closed")
	}
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_surface.destroy")) == 1 }, "the surface was not destroyed")
	// RequestFrame on a closed window does nothing.
	e.win.RequestFrame()
}

func TestShmWindowIgnoresFrameRequestsAndHasNoNativeSurface(t *testing.T) {
	e := startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()
	e.onLoop(func() {
		w := e.p.windows[0]
		w.RequestFrame()
		if h := w.NativeSurface(); h != (gpu.SurfaceHandle{}) {
			t.Errorf("NativeSurface() = %+v in shared-memory mode, want empty", h)
		}
	})
	if n := len(e.srv.RequestsWith("wl_surface.frame")); n != 0 {
		t.Errorf("a shared-memory window requested %d frame callbacks", n)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestNewTakesOneOptionsValue(t *testing.T) {
	defer func() {
		appErr, ok := errors.FromError(asErr(recover()))
		if !ok || appErr.Tag != tagMultipleOptions {
			t.Errorf("recovered %v, want a MultipleOptions error", appErr)
		}
	}()
	_, _ = New("glutter-unused", Options{}, Options{})
	t.Error("New accepted two Options")
}

func asErr(v any) error {
	err, _ := v.(error)
	return err
}

var _ = reflect.DeepEqual
