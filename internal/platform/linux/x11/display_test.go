//go:build glutter_display

package x11

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
)

// These tests need an X server. Run them under the headless one:
//
//	scripts/headless-x11.sh -- go test -race -tags glutter_display ./internal/platform/linux/x11/

const displayTimeout = 15 * time.Second

// x11Env is a platform on the X server, running, with a window.
type x11Env struct {
	t      *testing.T
	p      *Platform
	win    platform.Window
	h      *displayHandler
	cancel context.CancelFunc
	done   chan error
}

// displayHandler records what the platform reports and draws the window with
// OpenGL ES when asked for a frame.
type displayHandler struct {
	platform.NopHandler
	mu         sync.Mutex
	metrics    []platform.ViewMetrics
	frames     int
	keys       []ui.KeyData
	batches    [][]ui.PointerData
	closeCalls int
	allowClose bool

	// draw, when set, is called for each frame on the UI thread.
	draw func()
}

func (h *displayHandler) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) {
	h.mu.Lock()
	h.metrics = append(h.metrics, m)
	h.mu.Unlock()
}

func (h *displayHandler) OnFrame(ui.ViewID, time.Duration) {
	h.mu.Lock()
	h.frames++
	draw := h.draw
	h.mu.Unlock()
	if draw != nil {
		draw()
	}
}

func (h *displayHandler) OnKey(d ui.KeyData) {
	h.mu.Lock()
	h.keys = append(h.keys, d)
	h.mu.Unlock()
}

func (h *displayHandler) OnPointer(b []ui.PointerData) {
	h.mu.Lock()
	h.batches = append(h.batches, append([]ui.PointerData(nil), b...))
	h.mu.Unlock()
}

func (h *displayHandler) OnCloseRequest(ui.ViewID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closeCalls++
	return h.allowClose
}

func (h *displayHandler) snapshot() (metrics []platform.ViewMetrics, frames int, keys []ui.KeyData, batches [][]ui.PointerData, closes int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]platform.ViewMetrics(nil), h.metrics...), h.frames, append([]ui.KeyData(nil), h.keys...),
		append([][]ui.PointerData(nil), h.batches...), h.closeCalls
}

func startDisplay(t *testing.T, cfg platform.WindowConfig) *x11Env {
	t.Helper()
	if os.Getenv("DISPLAY") == "" {
		t.Skip("DISPLAY is not set: run the X11 display tests under scripts/headless-x11.sh")
	}
	t.Setenv("LC_ALL", "en_US.UTF-8")
	p, err := New("")
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			t.Fatalf("could not connect to the X server: %s %v", appErr.Tag, appErr.Metadata)
		}
		t.Fatalf("could not connect to the X server: %v", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	e := &x11Env{t: t, p: p, win: win, h: &displayHandler{allowClose: true}, done: make(chan error, 1)}
	ctx, cancel := context.WithTimeout(context.Background(), displayTimeout)
	e.cancel = cancel
	go func() { e.done <- p.Run(ctx, e.h) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-e.done:
		case <-time.After(displayTimeout):
			t.Error("Run did not return")
		}
	})
	return e
}

func windowConfig() platform.WindowConfig {
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter X11 test"
	cfg.AppID = "dev.glutter.x11test"
	cfg.Size = ui.Size{Width: 400, Height: 300}
	return cfg
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(displayTimeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal(msg)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// onLoop runs fn on the UI thread and waits for it.
func (e *x11Env) onLoop(fn func()) {
	e.t.Helper()
	finished := make(chan struct{})
	e.p.Post(func() { fn(); close(finished) })
	select {
	case <-finished:
	case <-time.After(displayTimeout):
		e.t.Fatal("the loop did not run a posted function")
	}
}

func (e *x11Env) waitMapped() {
	e.t.Helper()
	waitFor(e.t, func() bool { m, f, _, _, _ := e.h.snapshot(); return len(m) > 0 && f > 0 }, "the window was never mapped and drawn")
}

func TestDisplayWindowMapsWithTheRequestedSizeAndProperties(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	metrics, frames, _, _, _ := e.h.snapshot()
	if m := metrics[0]; m.PhysicalSize.Width != 400 || m.PhysicalSize.Height != 300 || m.DevicePixelRatio != 1 {
		t.Errorf("metrics = %+v, want 400x300 at ratio 1", m)
	}
	if frames < 1 {
		t.Errorf("%d frames, want at least 1", frames)
	}
	if e.p.xErrors != 0 {
		// Reading a field of the UI thread from the test thread is fine here: the
		// loop only writes it on an X error.
		t.Errorf("the X server reported %d errors", e.p.xErrors)
	}

	xprop, err := exec.LookPath("xprop")
	if err != nil {
		t.Skip("xprop is not installed (sudo dnf install xorg-x11-utils): the window properties are not checked")
	}
	id := "0x" + strings.ToLower(hex(e.win.NativeSurface().Window))
	out := func(args ...string) string {
		b, err := exec.Command(xprop, append([]string{"-id", id}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("xprop %v: %v\n%s", args, err, b)
		}
		return string(b)
	}
	if got := out("_NET_WM_NAME"); !strings.Contains(got, `"Glutter X11 test"`) {
		t.Errorf("_NET_WM_NAME: %s", got)
	}
	if got := out("WM_NAME"); !strings.Contains(got, `"Glutter X11 test"`) {
		t.Errorf("WM_NAME: %s", got)
	}
	if got := out("WM_CLASS"); !strings.Contains(got, `"dev.glutter.x11test", "dev.glutter.x11test"`) {
		t.Errorf("WM_CLASS: %s", got)
	}
	if got := out("WM_PROTOCOLS"); !strings.Contains(got, "WM_DELETE_WINDOW") || !strings.Contains(got, "_NET_WM_PING") {
		t.Errorf("WM_PROTOCOLS: %s", got)
	}
	if got := out("_NET_WM_PID"); !strings.Contains(got, "= "+itoa(os.Getpid())) {
		t.Errorf("_NET_WM_PID: %s", got)
	}
	if got := out("_NET_WM_WINDOW_TYPE"); !strings.Contains(got, "_NET_WM_WINDOW_TYPE_NORMAL") {
		t.Errorf("_NET_WM_WINDOW_TYPE: %s", got)
	}

	// SetTitle changes both names.
	e.onLoop(func() { e.win.SetTitle("Другой title") })
	if got := out("_NET_WM_NAME"); !strings.Contains(got, `"Другой title"`) {
		t.Errorf("_NET_WM_NAME after SetTitle: %s", got)
	}
}

func hex(v uintptr) string {
	const digits = "0123456789ABCDEF"
	if v == 0 {
		return "0"
	}
	var b []byte
	for ; v > 0; v >>= 4 {
		b = append([]byte{digits[v&15]}, b...)
	}
	return string(b)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for ; v > 0; v /= 10 {
		b = append([]byte{byte('0' + v%10)}, b...)
	}
	return string(b)
}

func TestDisplayEGLContextDrawsPurple(t *testing.T) {
	e := startDisplay(t, windowConfig())
	type result struct {
		version string
		pix     []uint8
		w, h    int
		err     error
	}
	done := make(chan result, 1)
	var once sync.Once
	e.h.mu.Lock()
	e.h.draw = func() {
		once.Do(func() {
			c, s, err := gles.Open(e.win.NativeSurface())
			if err != nil {
				done <- result{err: err}
				return
			}
			w, h := 400, 300
			c.Viewport(w, h)
			c.Clear(0x67/255.0, 0x50/255.0, 0xA4/255.0, 1)
			img, err := c.ReadPixels(w, h)
			if err == nil {
				err = c.SwapBuffers(s)
			}
			r := result{version: c.Version(), w: w, h: h, err: err}
			if img != nil {
				r.pix = img.Pix
			}
			s.Destroy()
			c.Close()
			done <- r
		})
	}
	e.h.mu.Unlock()
	e.p.Post(func() { e.win.RequestFrame() })

	var r result
	select {
	case r = <-done:
	case <-time.After(displayTimeout):
		t.Fatal("no frame was drawn")
	}
	if r.err != nil {
		if appErr, ok := errors.FromError(r.err); ok {
			t.Fatalf("%s: %s %v", appErr.Tag, appErr.Message, appErr.Metadata)
		}
		t.Fatal(r.err)
	}
	t.Logf("OpenGL ES version: %s", r.version)
	if len(r.pix) != r.w*r.h*4 {
		t.Fatalf("%d bytes for %dx%d", len(r.pix), r.w, r.h)
	}
	for i := 0; i < len(r.pix); i += 4 {
		got := [3]uint8{r.pix[i], r.pix[i+1], r.pix[i+2]}
		if want := [3]uint8{0x67, 0x50, 0xA4}; got != want {
			t.Fatalf("pixel %d = %v, want %v", i/4, got, want)
		}
	}
	if e.p.xErrors != 0 {
		t.Errorf("the X server reported %d errors", e.p.xErrors)
	}
	if e.p.input == nil || e.p.input.layout == nil {
		t.Error("the keyboard was not set up through XKB")
	}
	t.Logf("frame interval: %v", e.p.interval)
}

func TestDisplayFramesArePacedAndMerged(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	_, before, _, _, _ := e.h.snapshot()
	// A burst of requests makes one frame, one interval after the last.
	start := time.Now()
	e.onLoop(func() {
		for range 20 {
			e.win.RequestFrame()
		}
	})
	waitFor(t, func() bool { _, f, _, _, _ := e.h.snapshot(); return f > before }, "the requested frame never came")
	time.Sleep(3 * e.p.interval)
	_, after, _, _, _ := e.h.snapshot()
	if after-before != 1 {
		t.Errorf("%d frames for 20 merged requests, want 1 (took %v)", after-before, time.Since(start))
	}
	// Requests keep coming at no more than one frame per interval.
	n0 := after
	t0 := time.Now()
	for time.Since(t0) < 20*e.p.interval {
		e.onLoop(func() { e.win.RequestFrame() })
		time.Sleep(e.p.interval / 4)
	}
	_, n1, _, _, _ := e.h.snapshot()
	if frames, limit := n1-n0, 22; frames > limit || frames < 5 {
		t.Errorf("%d frames in 20 intervals, want between 5 and %d", frames, limit)
	}
}

// second is a second connection to the server, which plays the window manager and
// other clients.
type second struct {
	t       *testing.T
	x       *xlib
	display uintptr
}

func openSecond(t *testing.T, p *Platform) *second {
	t.Helper()
	d := p.x.openDisplay(0)
	if d == 0 {
		t.Fatal("could not open a second connection")
	}
	s := &second{t: t, x: p.x, display: d}
	t.Cleanup(func() { s.x.closeDisplay(d) })
	return s
}

// message sends a client message of format 32 to the window.
func (s *second) message(window, typ uint64, mask int64, data ...int64) {
	mem := dl.Calloc(1, xEventSize)
	defer dl.Free(mem)
	ev := view[xClientMessageEvent](mem)
	ev.typ, ev.window, ev.messageType, ev.format = clientMessage, window, typ, 32
	copy(ev.data[:], data)
	s.x.sendEvent(s.display, window, 0, mask, mem)
	s.x.flush(s.display)
}

func (s *second) atom(name string) uint64 {
	c := dl.CString(name)
	defer dl.Free(c)
	return s.x.internAtom(s.display, c, 0)
}

func TestDisplayCloseRequestAndPing(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	s := openSecond(t, e.p)
	win := uint64(e.win.NativeSurface().Window)

	// The ping is answered with the same message, sent to the root window, where a
	// window manager listens.
	root := e.p.root
	s.x.selectInput(s.display, root, substructureNotifyMask)
	s.x.flush(s.display)
	ping, protocols := s.atom("_NET_WM_PING"), s.atom("WM_PROTOCOLS")
	s.message(win, protocols, 0, int64(ping), 1234567, int64(win))
	reply := dl.Calloc(1, xEventSize)
	defer dl.Free(reply)
	got := make(chan bool, 1)
	go func() {
		for range 100 {
			if s.x.pending(s.display) > 0 {
				s.x.nextEvent(s.display, reply)
				if view[xAnyEvent](reply).typ == clientMessage {
					m := view[xClientMessageEvent](reply)
					got <- m.messageType == protocols && uint64(m.data[0]) == ping && m.data[1] == 1234567 && m.window == root
					return
				}
				continue
			}
			time.Sleep(20 * time.Millisecond)
		}
		got <- false
	}()
	if !<-got {
		t.Error("the ping was not answered with the same message sent to the root window")
	}

	// A close request is passed to the handler; refusing keeps the window.
	e.h.mu.Lock()
	e.h.allowClose = false
	e.h.mu.Unlock()
	deleteWindow := s.atom("WM_DELETE_WINDOW")
	s.message(win, protocols, 0, int64(deleteWindow), 0)
	waitFor(t, func() bool { _, _, _, _, c := e.h.snapshot(); return c == 1 }, "the close request did not reach the handler")
	select {
	case err := <-e.done:
		t.Fatalf("Run returned (%v) although the close request was refused", err)
	case <-time.After(200 * time.Millisecond):
	}

	// Agreeing closes the window, and the last window ending ends Run.
	e.h.mu.Lock()
	e.h.allowClose = true
	e.h.mu.Unlock()
	s.message(win, protocols, 0, int64(deleteWindow), 0)
	select {
	case err := <-e.done:
		if err != nil {
			t.Errorf("Run returned %v after the last window closed", err)
		}
	case <-time.After(displayTimeout):
		t.Fatal("Run did not return after the window closed")
	}
	e.done <- nil // for the cleanup, which waits for Run
}

func TestDisplayResizeAndStateRequests(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	e.onLoop(func() { e.win.SetSize(ui.Size{Width: 520, Height: 340}) })
	waitFor(t, func() bool {
		m, _, _, _, _ := e.h.snapshot()
		return len(m) > 1 && m[len(m)-1].PhysicalSize.Width == 520 && m[len(m)-1].PhysicalSize.Height == 340
	}, "the new size never reached the handler")
	// The size change draws a frame at once.
	_, f0, _, _, _ := e.h.snapshot()
	e.onLoop(func() { e.win.SetSize(ui.Size{Width: 530, Height: 340}) })
	waitFor(t, func() bool { _, f, _, _, _ := e.h.snapshot(); return f > f0 }, "a resize did not draw a frame")

	// Without a window manager nothing handles the state requests, but they are
	// well-formed: the server reports no error for them.
	e.onLoop(func() {
		for _, s := range []platform.WindowState{
			platform.WindowStateMaximized, platform.WindowStateFullscreen, platform.WindowStateNormal,
			platform.WindowStateMinimized, platform.WindowStateNormal,
		} {
			e.win.SetState(s)
		}
		e.p.x.sync(e.p.display, 0)
	})
	if e.p.xErrors != 0 {
		t.Errorf("the X server reported %d errors", e.p.xErrors)
	}
}

func TestDisplayCursors(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	e.onLoop(func() {
		for c := platform.MouseCursorUnspecified; c.Valid(); c++ {
			e.win.SetCursor(c)
		}
		e.p.x.sync(e.p.display, 0)
	})
	if e.p.xErrors != 0 {
		t.Errorf("the X server reported %d errors", e.p.xErrors)
	}
	var basic, hidden uint64
	var missing []string
	e.onLoop(func() {
		basic = e.p.cursors.loaded[platform.MouseCursorBasic]
		hidden = e.p.cursors.invisible
		for c := platform.MouseCursorBasic; c.Valid(); c++ {
			if e.p.cursors.loaded[c] == 0 {
				missing = append(missing, c.String())
			}
		}
	})
	if basic == 0 {
		t.Error("the default cursor was not loaded from the theme")
	}
	if hidden == 0 {
		t.Error("the hidden cursor was not made")
	}
	t.Logf("cursors that the theme lacks: %v", missing)
}

func TestDisplayWindowKinds(t *testing.T) {
	dialog := windowConfig()
	dialog.Kind = platform.WindowKindDialog
	dialog.Modal = true
	dialog.Decorated = false
	dialog.Resizable = false
	dialog.Title = "dialog"
	e := startDisplay(t, dialog)
	e.waitMapped()
	xprop, err := exec.LookPath("xprop")
	if err != nil {
		t.Skip("xprop is not installed")
	}
	id := "0x" + strings.ToLower(hex(e.win.NativeSurface().Window))
	out := func(args ...string) string {
		b, err := exec.Command(xprop, append([]string{"-id", id}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("xprop %v: %v\n%s", args, err, b)
		}
		return string(b)
	}
	if got := out("_NET_WM_WINDOW_TYPE"); !strings.Contains(got, "_NET_WM_WINDOW_TYPE_DIALOG") {
		t.Errorf("_NET_WM_WINDOW_TYPE: %s", got)
	}
	if got := out("_NET_WM_STATE"); !strings.Contains(got, "_NET_WM_STATE_MODAL") {
		t.Errorf("_NET_WM_STATE: %s", got)
	}
	if got := out("_MOTIF_WM_HINTS"); !strings.Contains(got, "0x2, 0x0, 0x0, 0x0, 0x0") {
		t.Errorf("_MOTIF_WM_HINTS: %s", got)
	}
	if got := out("WM_NORMAL_HINTS"); !strings.Contains(got, "program specified minimum size: 400 by 300") ||
		!strings.Contains(got, "program specified maximum size: 400 by 300") {
		t.Errorf("WM_NORMAL_HINTS: %s", got)
	}
}
