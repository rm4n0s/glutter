//go:build linux

package wayland

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

// recHandler is a platform.Handler that records what it is told. Its fields are
// guarded by mu, because tests read them while Run is going.
type recHandler struct {
	platform.NopHandler
	mu         sync.Mutex
	metrics    []platform.ViewMetrics
	closeCalls []ui.ViewID
	allowClose bool
}

func (h *recHandler) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metrics = append(h.metrics, m)
}

func (h *recHandler) OnCloseRequest(v ui.ViewID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closeCalls = append(h.closeCalls, v)
	return h.allowClose
}

func (h *recHandler) metricsSeen() []platform.ViewMetrics {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]platform.ViewMetrics(nil), h.metrics...)
}

func (h *recHandler) closeRequests() []ui.ViewID {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]ui.ViewID(nil), h.closeCalls...)
}

// windowEnv is a platform on a fake compositor, running.
type windowEnv struct {
	t       *testing.T
	p       *Platform
	srv     *fakecomp.Server
	h       *recHandler
	done    <-chan error
	cancel  context.CancelFunc
	stopped bool
}

// startWindows connects to a fake compositor, creates the windows from cfgs on
// the UI thread (before Run starts, as an engine does), and starts Run.
func startWindows(t *testing.T, fake fakecomp.Config, cfgs ...platform.WindowConfig) *windowEnv {
	t.Helper()
	p, srv, err := connect(t, fake)
	if err != nil {
		t.Fatal(err)
	}
	for _, cfg := range cfgs {
		if _, err := p.NewWindow(cfg); err != nil {
			t.Fatalf("NewWindow: %v", err)
		}
	}
	h := &recHandler{allowClose: true}
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()
	e := &windowEnv{t: t, p: p, srv: srv, h: h, done: done, cancel: cancel}
	t.Cleanup(func() {
		cancel()
		if !e.stopped {
			<-done
		}
	})
	return e
}

// onLoop runs fn on the UI thread and waits for it.
func (e *windowEnv) onLoop(fn func()) {
	e.t.Helper()
	finished := make(chan struct{})
	e.p.Post(func() {
		fn()
		close(finished)
	})
	select {
	case <-finished:
	case <-time.After(testTimeout):
		e.t.Fatal("the loop did not run a posted function")
	}
}

// stop cancels Run and returns its result.
func (e *windowEnv) stop() error {
	e.t.Helper()
	e.cancel()
	e.stopped = true
	return wait(e.t, e.done)
}

// waitMapped waits until the compositor shows a buffer, and returns its size.
func (e *windowEnv) waitMapped() (w, h int32) {
	e.t.Helper()
	waitUntil(e.t, func() bool { _, _, ok := e.srv.Mapped(); return ok }, "the window was never mapped")
	w, h, _ = e.srv.Mapped()
	return w, h
}

func config640() platform.WindowConfig {
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter dev window"
	cfg.AppID = "dev.glutter.test"
	cfg.Size = ui.Size{Width: 640, Height: 400}
	return cfg
}

// index returns the position of the first line equal to or starting with want
// at or after from, or -1.
func index(lines []string, from int, want string) int {
	for i := from; i < len(lines); i++ {
		if lines[i] == want || strings.HasPrefix(lines[i], want) {
			return i
		}
	}
	return -1
}

func TestWindowIsMappedAfterTheFirstConfigure(t *testing.T) {
	cfg := config640()
	cfg.MinSize = ui.Size{Width: 200, Height: 100}
	e := startWindows(t, fakecomp.Config{}, cfg)

	w, h := e.waitMapped()
	if w != 640 || h != 400 {
		t.Errorf("mapped buffer is %dx%d, want 640x400 (the config's size, the compositor suggested none)", w, h)
	}

	// The order that makes the compositor match the window to its .desktop file:
	// title and app id first, then the buffer-less commit, and only then, after
	// the configure, the buffer.
	lines := e.srv.Requests()
	order := []string{
		"wl_compositor.create_surface",
		"xdg_wm_base.get_xdg_surface",
		"xdg_surface.get_toplevel",
		`xdg_toplevel.set_title "Glutter dev window"`,
		`xdg_toplevel.set_app_id "dev.glutter.test"`,
		"xdg_toplevel.set_min_size 200x100",
		"xdg_toplevel.set_max_size 0x0",
		"wl_surface.commit", // the initial commit has no buffer
		"wl_shm.create_pool",
		"wl_shm_pool.create_buffer 640x400 offset=0 stride=2560 format=1",
		"wl_shm_pool.destroy",
		"xdg_surface.ack_configure",
		"wl_surface.attach",
		"wl_surface.damage_buffer 0,0 640x400",
		"wl_surface.commit buffer=",
	}
	at := 0
	for _, want := range order {
		i := index(lines, at, want)
		if i < 0 {
			t.Fatalf("%q is missing, or comes too early, in:\n%s", want, strings.Join(lines, "\n"))
		}
		at = i + 1
	}

	metrics := e.h.metricsSeen()
	if len(metrics) != 1 || metrics[0].PhysicalSize != (ui.Size{Width: 640, Height: 400}) || metrics[0].DevicePixelRatio != 1 {
		t.Errorf("metrics = %+v, want one report of 640x400 at ratio 1", metrics)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultSizeWhenNeitherConfigNorCompositorSaysOne(t *testing.T) {
	cfg := platform.DefaultWindowConfig()
	e := startWindows(t, fakecomp.Config{}, cfg)
	if w, h := e.waitMapped(); w != 800 || h != 600 {
		t.Errorf("mapped buffer is %dx%d, want the default 800x600", w, h)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestConfigureSizeWinsAndStatesAreStored(t *testing.T) {
	e := startWindows(t, fakecomp.Config{Toplevel: fakecomp.ToplevelConfigure{
		Width: 1000, Height: 700, States: []uint32{1, 4}, // maximized, activated
	}}, config640())

	if w, h := e.waitMapped(); w != 1000 || h != 700 {
		t.Errorf("mapped buffer is %dx%d, want the compositor's 1000x700", w, h)
	}
	var states WindowStates
	e.onLoop(func() { states = e.p.windows[0].States() })
	if want := (WindowStates{Maximized: true, Activated: true}); states != want {
		t.Errorf("states = %+v, want %+v", states, want)
	}

	// Another configure: all states change.
	e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{Width: 0, Height: 0, States: []uint32{2, 3, 5, 6, 7, 8, 9}})
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("xdg_surface.ack_configure")) == 2 }, "the second configure was not acknowledged")
	e.onLoop(func() { states = e.p.windows[0].States() })
	want := WindowStates{Fullscreen: true, Resizing: true, TiledLeft: true, TiledRight: true, TiledTop: true, TiledBottom: true, Suspended: true}
	if states != want {
		t.Errorf("states = %+v, want %+v", states, want)
	}
	// A configure with size 0x0 keeps the current size.
	if w, h, _ := e.srv.Mapped(); w != 1000 || h != 700 {
		t.Errorf("a 0x0 configure changed the size to %dx%d", w, h)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestResizeDrawsBeforeTheAckAndFreesTheOldBuffer(t *testing.T) {
	e := startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()

	e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{Width: 300, Height: 200, States: []uint32{3}})
	waitUntil(t, func() bool { w, h, ok := e.srv.Mapped(); return ok && w == 300 && h == 200 }, "the new size was never shown")

	lines := e.srv.Requests()
	create := index(lines, 0, "wl_shm_pool.create_buffer 300x200")
	ack := index(lines, create, "xdg_surface.ack_configure")
	commit := index(lines, ack, "wl_surface.commit buffer=")
	if create < 0 || ack < 0 || commit < 0 || create >= ack || ack >= commit {
		t.Errorf("want create_buffer 300x200, then ack_configure, then the commit; got positions %d, %d, %d in:\n%s",
			create, ack, commit, strings.Join(lines, "\n"))
	}

	// The compositor released the first buffer when the second replaced it; the
	// window frees it then.
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_buffer.destroy")) == 1 }, "the replaced buffer was never freed")

	metrics := e.h.metricsSeen()
	if len(metrics) != 2 || metrics[1].PhysicalSize != (ui.Size{Width: 300, Height: 200}) {
		t.Errorf("metrics = %+v, want 640x400 and then 300x200", metrics)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestDecorations(t *testing.T) {
	globals := append(fakecomp.DefaultGlobals(), fakecomp.Global{Name: 50, Interface: "zxdg_decoration_manager_v1", Version: 1})
	tests := []struct {
		name      string
		decorated bool
		globals   []fakecomp.Global
		wantMode  string // the set_mode request, or "" for none
	}{
		{"server-side by default", true, globals, "zxdg_toplevel_decoration_v1.set_mode 2"},
		{"client-side when the app draws its own", false, globals, "zxdg_toplevel_decoration_v1.set_mode 1"},
		{"no protocol, no request", true, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config640()
			cfg.Decorated = foundation.Some(tt.decorated)
			e := startWindows(t, fakecomp.Config{Globals: tt.globals}, cfg)
			e.waitMapped()
			got := e.srv.RequestsWith("zxdg_toplevel_decoration_v1.set_mode")
			if tt.wantMode == "" {
				if len(got) != 0 || len(e.srv.RequestsWith("get_toplevel_decoration")) != 0 {
					t.Errorf("decoration requests were sent without the protocol: %v", got)
				}
			} else {
				if !reflect.DeepEqual(got, []string{tt.wantMode}) {
					t.Errorf("set_mode requests = %q, want %q", got, tt.wantMode)
				}
				// get_toplevel_decoration comes before the first commit.
				lines := e.srv.Requests()
				if d, c := index(lines, 0, "zxdg_decoration_manager_v1.get_toplevel_decoration"), index(lines, 0, "wl_surface.commit"); d < 0 || d > c {
					t.Errorf("the decoration was requested at %d, after the first commit at %d", d, c)
				}
			}
			if err := e.stop(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCloseRequest(t *testing.T) {
	t.Run("the handler agrees: the window is destroyed and Run returns", func(t *testing.T) {
		e := startWindows(t, fakecomp.Config{}, config640())
		e.waitMapped()
		e.srv.CloseToplevel()
		if err := wait(t, e.done); err != nil {
			t.Fatalf("Run returned %v, want nil after the last window closed", err)
		}
		e.stopped = true
		if got := e.h.closeRequests(); !reflect.DeepEqual(got, []ui.ViewID{0}) {
			t.Errorf("close requests = %v, want [0]", got)
		}
		// Run flushed the destroy requests before it returned; the compositor
		// handles them a moment later.
		for _, want := range []string{"xdg_toplevel.destroy", "xdg_surface.destroy", "wl_surface.destroy", "wl_buffer.destroy"} {
			waitUntil(t, func() bool { return len(e.srv.RequestsWith(want)) > 0 }, want+" was never sent when the window closed")
		}
	})
	t.Run("the handler refuses: the window stays", func(t *testing.T) {
		e := startWindows(t, fakecomp.Config{}, config640())
		e.h.allowClose = false
		e.waitMapped()
		e.srv.CloseToplevel()
		waitUntil(t, func() bool { return len(e.h.closeRequests()) == 1 }, "the handler was never asked")
		select {
		case err := <-e.done:
			t.Fatalf("Run returned %v although the window was kept", err)
		case <-time.After(100 * time.Millisecond):
		}
		if len(e.srv.RequestsWith("xdg_toplevel.destroy")) != 0 {
			t.Error("the window was destroyed although the handler refused")
		}
		if err := e.stop(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestWindowMethods(t *testing.T) {
	e := startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()

	e.onLoop(func() {
		w := e.p.windows[0]
		w.SetTitle("renamed")
		w.SetState(platform.WindowStateMaximized)
		w.SetState(platform.WindowStateFullscreen)
		w.SetState(platform.WindowStateMinimized)
		w.SetState(platform.WindowStateNormal)
		w.SetSize(ui.Size{Width: 500, Height: 300})
		w.SetSize(ui.Size{Width: 500, Height: 300}) // same size: nothing to do
		w.SetSize(ui.Size{Width: 0, Height: 10})    // invalid: ignored
		w.SetCursor(0)
		w.SetTextInput(ui.TextInputState{})
	})
	waitUntil(t, func() bool { w, h, ok := e.srv.Mapped(); return ok && w == 500 && h == 300 }, "SetSize did not show a 500x300 buffer")

	lines := e.srv.Requests()
	tail := lines[index(lines, 0, `xdg_toplevel.set_title "renamed"`):]
	var got []string
	for _, l := range tail {
		if strings.HasPrefix(l, "xdg_toplevel.") {
			got = append(got, l)
		}
	}
	want := []string{
		`xdg_toplevel.set_title "renamed"`,
		"xdg_toplevel.set_maximized",
		"xdg_toplevel.set_fullscreen",
		"xdg_toplevel.set_minimized",
		"xdg_toplevel.unset_maximized",
		"xdg_toplevel.unset_fullscreen",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("toplevel requests = %q, want %q", got, want)
	}
	if n := len(e.srv.RequestsWith("create_buffer 500x300")); n != 1 {
		t.Errorf("created %d buffers of 500x300, want 1 (the repeated SetSize must do nothing)", n)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestSeveralWindowsAndParents(t *testing.T) {
	main := config640()
	dialog := config640()
	dialog.Kind = platform.WindowKindDialog
	dialog.HasParent = true
	dialog.Parent = 0
	dialog.Title = "dialog"
	e := startWindows(t, fakecomp.Config{}, main, dialog)

	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_surface.commit buffer=")) >= 2 }, "both windows are not mapped")
	var ids []ui.ViewID
	e.onLoop(func() {
		for id := range e.p.windows {
			ids = append(ids, id)
		}
	})
	if len(ids) != 2 {
		t.Fatalf("the platform has %d windows, want 2", len(ids))
	}
	if got := e.srv.RequestsWith("xdg_toplevel.set_parent"); !reflect.DeepEqual(got, []string{"xdg_toplevel.set_parent true"}) {
		t.Errorf("set_parent requests = %q, want one with a parent", got)
	}

	// Closing one window keeps Run going; closing the other ends it.
	e.onLoop(func() { e.p.windows[1].Close() })
	select {
	case err := <-e.done:
		t.Fatalf("Run returned %v while a window is open", err)
	case <-time.After(100 * time.Millisecond):
	}
	e.onLoop(func() {
		w := e.p.windows[0]
		w.Close()
		w.Close() // closing twice does nothing
	})
	if err := wait(t, e.done); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
	e.stopped = true
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("xdg_toplevel.destroy")) == 2 }, "both toplevels should have been destroyed exactly once")
}

func TestBufferFailureEndsRun(t *testing.T) {
	e := startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()
	// 40000 x 40000 pixels is 6.4 GB, over the limit of one buffer.
	e.onLoop(func() { e.p.windows[0].SetSize(ui.Size{Width: 40000, Height: 40000}) })
	err := wait(t, e.done)
	e.stopped = true
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "ShmBufferFailed" || appErr.Metadata["op"] != "size" {
		t.Fatalf("Run returned %v, want ShmBufferFailed (op size)", err)
	}
}

func TestNewShmBufferRejectsBadSizes(t *testing.T) {
	for _, size := range [][2]int32{{0, 10}, {10, 0}, {-1, 5}, {30000, 30000}} {
		_, err := newShmBuffer(nil, size[0], size[1], 0, nil)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "ShmBufferFailed" || !appErr.HasRoute("newShmBufferFormat.ShmBufferFailed") {
			t.Errorf("newShmBuffer(%v) returned %v, want ShmBufferFailed via newShmBufferFormat", size, err)
		}
	}
}

func TestFill(t *testing.T) {
	data := make([]byte, 16)
	fill(data, 0xFF6750A4)
	want := []byte{0xA4, 0x50, 0x67, 0xFF, 0xA4, 0x50, 0x67, 0xFF, 0xA4, 0x50, 0x67, 0xFF, 0xA4, 0x50, 0x67, 0xFF}
	if !reflect.DeepEqual(data, want) {
		t.Errorf("fill = %x, want %x (little-endian XRGB, blue first)", data, want)
	}
	fill(nil, 1) // no panic on an empty buffer
}

func TestDecodeStates(t *testing.T) {
	tests := []struct {
		in   []uint32
		want WindowStates
	}{
		{nil, WindowStates{}},
		{[]uint32{1}, WindowStates{Maximized: true}},
		{[]uint32{4, 1}, WindowStates{Maximized: true, Activated: true}},
		{[]uint32{9}, WindowStates{Suspended: true}},
		{[]uint32{999}, WindowStates{}}, // an unknown state is ignored
	}
	for _, tt := range tests {
		if got := decodeStates(tt.in); got != tt.want {
			t.Errorf("decodeStates(%v) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}
