//go:build linux && !android && glutter_display

package x11

import (
	"context"
	"encoding/binary"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dbus"
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

	settings []platform.Settings
	// displays counts the OnDisplaysChanged calls.
	displays int
	// drops and dragEnds are the drag and drop events; answer is what OnDrop returns.
	drops    []ui.DropEvent
	dragEnds []ui.DragResult
	answer   func(ui.DropEvent) ui.DropActions

	// lost counts the OnContentLost calls, and focus is what OnViewFocus was told.
	lost  int
	focus []bool

	// draw, when set, is called for each frame on the UI thread.
	draw func()
}

func (h *displayHandler) OnViewFocus(_ ui.ViewID, focused bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.focus = append(h.focus, focused)
}

func (h *displayHandler) focusSeen() []bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]bool(nil), h.focus...)
}

func (h *displayHandler) OnContentLost(ui.ViewID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lost++
}

func (h *displayHandler) contentLost() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lost
}

func (h *displayHandler) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) {
	h.mu.Lock()
	h.metrics = append(h.metrics, m)
	h.mu.Unlock()
}

func (h *displayHandler) OnDisplaysChanged() {
	h.mu.Lock()
	h.displays++
	h.mu.Unlock()
}

func (h *displayHandler) displayChanges() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.displays
}

func (h *displayHandler) OnSettings(s platform.Settings) {
	h.mu.Lock()
	h.settings = append(h.settings, s)
	h.mu.Unlock()
}

func (h *displayHandler) settingsSeen() []platform.Settings {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]platform.Settings(nil), h.settings...)
}

func (h *displayHandler) OnDrop(e ui.DropEvent) ui.DropActions {
	h.mu.Lock()
	h.drops = append(h.drops, e)
	answer := h.answer
	h.mu.Unlock()
	if answer == nil {
		return ui.DropActionNone
	}
	return answer(e)
}

func (h *displayHandler) OnDragEnd(r ui.DragResult) {
	h.mu.Lock()
	h.dragEnds = append(h.dragEnds, r)
	h.mu.Unlock()
}

func (h *displayHandler) dropEvents() []ui.DropEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]ui.DropEvent(nil), h.drops...)
}

func (h *displayHandler) dragResults() []ui.DragResult {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]ui.DragResult(nil), h.dragEnds...)
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
	return startDisplayWith(t, cfg, nil)
}

// startDisplayWith is startDisplay that calls before(p) when the platform is made
// and before its window is: the place to start a window manager, which must be there
// for the window's map request.
func startDisplayWith(t *testing.T, cfg platform.WindowConfig, before func(*Platform)) *x11Env {
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
	if before != nil {
		before(p)
	}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	e := &x11Env{t: t, p: p, win: win, h: &displayHandler{allowClose: true}, done: make(chan error, 1)}
	ctx, cancel := context.WithTimeout(context.Background(), displayTimeout)
	e.cancel = cancel
	go func() {
		// An OpenGL context belongs to the OS thread that made it current, so the
		// goroutine that runs the loop stays on one thread, as the main goroutine does
		// in an app (SPEC §8.4).
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		e.done <- p.Run(ctx, e.h)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-e.done:
			if err != nil {
				t.Errorf("Run: %v", err)
			}
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

	if h := e.win.NativeSurface(); h.Surface == 0 || h.Surface != h.Window {
		t.Errorf("NativeSurface() = %+v, want the window ID as the surface too", h)
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

// TestDisplayExposeReportsLostContentAndAsksForAFrame checks that an exposed window
// has its next frame presented whole: the handler is told, then a frame is asked for.
func TestDisplayExposeReportsLostContentAndAsksForAFrame(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	// The server's own expose after the map may or may not have come yet.
	time.Sleep(3 * e.p.interval)
	_, frames, _, _, _ := e.h.snapshot()
	lost := e.h.contentLost()
	e.onLoop(func() { e.win.(*window).onExpose() })
	if got := e.h.contentLost(); got != lost+1 {
		t.Errorf("OnContentLost was called %d times by the expose, want 1", got-lost)
	}
	waitFor(t, func() bool { _, f, _, _, _ := e.h.snapshot(); return f > frames }, "no frame followed the expose")
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
		basic = loadedCursor(e.p.cursors, platform.MouseCursorBasic)
		hidden = e.p.cursors.invisible
		for c := platform.MouseCursorBasic; c.Valid(); c++ {
			if loadedCursor(e.p.cursors, c) == 0 {
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
	dialog.Decorated = foundation.Some(false)
	dialog.Resizable = foundation.Some(false)
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

// setXftDPI writes Xft.dpi to RESOURCE_MANAGER of the root window over a connection
// of its own, as a settings daemon does.
func setXftDPI(t *testing.T, dpi int) {
	t.Helper()
	if os.Getenv("DISPLAY") == "" {
		t.Skip("DISPLAY is not set: run the X11 display tests under scripts/headless-x11.sh")
	}
	other, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := other.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()
	text := "Xft.dpi:\t" + strconv.Itoa(dpi) + "\n"
	c := dl.CString(text)
	defer dl.Free(c)
	other.x.changeProperty(other.display, other.root, other.atoms.resourceManager, xaString, 8, propModeReplace, c, int32(len(text)))
	other.x.sync(other.display, 0)
}

func TestDisplayXftDPIScalesTheWindowAndFollowsChanges(t *testing.T) {
	setXftDPI(t, 96)
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	last := func() platform.ViewMetrics {
		m, _, _, _, _ := e.h.snapshot()
		return m[len(m)-1]
	}
	if m := last(); m.DevicePixelRatio != 1 || m.PhysicalSize != (ui.Size{Width: 400, Height: 300}) {
		t.Fatalf("metrics at 96 dpi = %+v", m)
	}
	var displays []platform.Display
	e.onLoop(func() { displays = e.p.Displays() })
	if len(displays) == 0 {
		t.Fatal("the X server has no monitors")
	}
	if displays[0].DevicePixelRatio != 1 || !displays[0].Primary || displays[0].RefreshRate <= 0 {
		t.Errorf("displays = %+v", displays)
	}
	if m := last(); m.Display.IsNone() {
		t.Errorf("the metrics do not name the monitor of the window: %+v", m)
	}

	// 192 dpi: the pixel size stays, the ratio is 2, and a frame is drawn at it.
	_, frames, _, _, _ := e.h.snapshot()
	setXftDPI(t, 192)
	waitFor(t, func() bool { return last().DevicePixelRatio == 2 }, "Xft.dpi 192 never reached the metrics")
	waitFor(t, func() bool { _, f, _, _, _ := e.h.snapshot(); return f > frames }, "no frame at the new scale")
	e.onLoop(func() { displays = e.p.Displays() })
	if displays[0].DevicePixelRatio != 2 {
		t.Errorf("the display scale after the change = %v", displays[0].DevicePixelRatio)
	}

	setXftDPI(t, 96)
	waitFor(t, func() bool { return last().DevicePixelRatio == 1 }, "Xft.dpi 96 never reached the metrics")
}

func TestDisplayWindowIsMadeInLogicalPixels(t *testing.T) {
	setXftDPI(t, 192)
	t.Cleanup(func() { setXftDPI(t, 96) })
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	m, _, _, _, _ := e.h.snapshot()
	if got := m[len(m)-1]; got.DevicePixelRatio != 2 || got.PhysicalSize != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("a 400x300 window at 192 dpi has metrics %+v, want 800x600 pixels at 2", got)
	}
}

// loadedCursor is the X cursor ID of a cursor that was loaded, at any size, or 0.
func loadedCursor(c *cursors, cursor platform.MouseCursor) uint64 {
	for key, id := range c.loaded {
		if key.cursor == cursor && id != 0 {
			return id
		}
	}
	return 0
}

func TestDisplayCursorsHaveTheSizeOfTheScale(t *testing.T) {
	setXftDPI(t, 192)
	t.Cleanup(func() { setXftDPI(t, 96) })
	t.Setenv("XCURSOR_SIZE", "")
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	e.onLoop(func() { e.win.SetCursor(platform.MouseCursorText); e.p.x.sync(e.p.display, 0) })
	var sizes []int32
	e.onLoop(func() {
		for key := range e.p.cursors.loaded {
			sizes = append(sizes, key.size)
		}
	})
	if len(sizes) != 1 || sizes[0] != 48 {
		t.Errorf("cursors loaded at sizes %v, want one at 48 (24 at a scale of 2)", sizes)
	}
	if e.p.xErrors != 0 {
		t.Errorf("the X server reported %d errors", e.p.xErrors)
	}
}

// xsettingsDaemon is a settings daemon of the test: a window of its own connection
// that owns _XSETTINGS_S0 and keeps the settings in its property.
type xsettingsDaemon struct {
	t      *testing.T
	p      *Platform
	window uint64
}

func startXSettingsDaemon(t *testing.T, data []byte) *xsettingsDaemon {
	t.Helper()
	if os.Getenv("DISPLAY") == "" {
		t.Skip("DISPLAY is not set: run the X11 display tests under scripts/headless-x11.sh")
	}
	p, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	d := &xsettingsDaemon{t: t, p: p}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	d.window = p.x.createWindow(p.display, p.root, 0, 0, 1, 1, 0, 0, inputOnly, 0, 0, 0)
	if d.window == 0 {
		t.Fatal("could not make the window of the daemon")
	}
	d.set(data)
	p.x.setSelectionOwner(p.display, p.atoms.xsettingsSelection, d.window, currentTime)
	p.x.sync(p.display, 0)
	if p.x.getSelectionOwner(p.display, p.atoms.xsettingsSelection) != d.window {
		t.Fatal("the daemon does not own the selection")
	}
	return d
}

// set writes the settings into the property.
func (d *xsettingsDaemon) set(data []byte) {
	mem := dl.Calloc(1, uintptr(len(data)+1))
	defer dl.Free(mem)
	copy(dl.Bytes(mem, len(data)), data)
	d.p.x.changeProperty(d.p.display, d.window, d.p.atoms.xsettingsProperty, d.p.atoms.xsettingsProperty, 8, propModeReplace, mem, int32(len(data)))
	d.p.x.sync(d.p.display, 0)
}

func TestDisplaySettingsFromXSettingsFollowTheDaemon(t *testing.T) {
	dark := newXSettingsData(binary.LittleEndian).String("Net/ThemeName", "Adwaita-dark").
		String("Gtk/CursorThemeName", "Bibata").Int("Gtk/CursorThemeSize", 36).Bytes()
	daemon := startXSettingsDaemon(t, dark)
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	t.Setenv("LC_ALL", "el_GR.UTF-8") // startDisplay sets another

	var s platform.Settings
	e.onLoop(func() {
		// No portal: the settings come from XSETTINGS and the environment.
		e.p.settings.Open = func(context.Context) (dbus.Bus, error) { return nil, errors.New(tagNoBus, "the test has no bus") }
		s = e.p.Settings()
	})
	if s.PlatformBrightness != ui.BrightnessDark || s.CursorTheme != "Bibata" || s.CursorSize != 36 {
		t.Errorf("Settings() = %+v, want the dark theme and the cursor of the daemon", s)
	}
	if len(s.Locales) != 1 || s.Locales[0] != ui.NewLocale("el", "GR") {
		t.Errorf("locales = %v", s.Locales)
	}

	// The user picks a light theme and animations off: the app hears of it.
	daemon.set(newXSettingsData(binary.LittleEndian).String("Net/ThemeName", "Adwaita").
		String("Gtk/CursorThemeName", "Bibata").Int("Gtk/CursorThemeSize", 36).Int("Gtk/EnableAnimations", 0).Bytes())
	waitFor(t, func() bool { return len(e.h.settingsSeen()) == 1 }, "OnSettings was not called for the new XSETTINGS")
	if got := e.h.settingsSeen()[0]; got.PlatformBrightness != ui.BrightnessLight || !got.DisableAnimations || got.CursorSize != 36 {
		t.Errorf("after the change: %+v", got)
	}
	var theme string
	e.onLoop(func() { theme = e.p.cursors.themeName })
	if theme != "Bibata" {
		t.Errorf("the cursors use the theme %q", theme)
	}
}

// TestDisplayDragBetweenTwoWindows drags from one window to another of the same app
// over the real XDND protocol: the messages, the selection and the reply of the
// target all go through the X server. The pointer is played by calling what the
// XInput2 events of the pointer call.
func TestDisplayDragBetweenTwoWindows(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	var second platform.Window
	e.onLoop(func() {
		w, err := e.p.NewWindow(windowConfig())
		if err != nil {
			t.Error(err)
			return
		}
		second = w
	})
	if second == nil {
		return
	}
	waitFor(t, func() bool {
		m, _, _, _, _ := e.h.snapshot()
		for _, x := range m {
			if x.PhysicalSize.Width > 0 && len(m) >= 2 {
				return true
			}
		}
		return false
	}, "the second window was never mapped")
	e.h.mu.Lock()
	e.h.answer = func(ev ui.DropEvent) ui.DropActions {
		if ev.ViewID == second.ID() {
			return ui.DropActionCopy
		}
		return ui.DropActionNone
	}
	e.h.mu.Unlock()

	req := ui.DragRequest{ID: 21, Text: "héllo", URIs: []string{"file:///tmp/a%20b", "file:///tmp/c"}, Actions: ui.DropActionCopy | ui.DropActionMove}
	var source *xdndSource
	e.onLoop(func() {
		e.p.input.buttonsDown = 1 << 1
		if err := e.win.StartDrag(req); err != nil {
			t.Error(err)
			return
		}
		source = e.p.xdnd.source
		source.onMotion(100, 100) // over the second window, which is above the first
	})
	if source == nil {
		return
	}
	waitFor(t, func() bool {
		var accepted bool
		e.onLoop(func() { accepted = source.accepted })
		return accepted
	}, "the target never accepted the drag")
	events := e.h.dropEvents()
	if len(events) == 0 || events[0].Phase != ui.DropPhaseEnter || events[0].ViewID != second.ID() ||
		events[0].Position != (ui.Offset{Dx: 100, Dy: 100}) || events[0].LocalID != 21 || events[0].Actions == ui.DropActionNone {
		t.Fatalf("enter = %+v", events)
	}
	for _, typ := range events[0].Types {
		if strings.HasPrefix(typ, localMarkerPrefix) {
			t.Errorf("the marker is among the types: %v", events[0].Types)
		}
	}
	if !slices.Contains(events[0].Types, "text/uri-list") || !slices.Contains(events[0].Types, "text/plain;charset=utf-8") {
		t.Errorf("types = %v", events[0].Types)
	}

	e.onLoop(func() { source.onMotion(120, 130) })
	e.onLoop(func() { source.onRelease() })
	waitFor(t, func() bool { return len(e.h.dragResults()) == 1 }, "the drag never ended")
	if got := e.h.dragResults()[0]; got.ID != 21 || got.Action != ui.DropActionCopy {
		t.Errorf("result = %+v", got)
	}
	events = e.h.dropEvents()
	drop := events[len(events)-1]
	if drop.Phase != ui.DropPhaseDrop || drop.Text != "héllo" || !slices.Equal(drop.URIs, req.URIs) {
		t.Errorf("drop = %+v", drop)
	}
	if e.p.xErrors != 0 {
		t.Errorf("the X server reported %d errors", e.p.xErrors)
	}
}

// TestDisplayDragNobodyTakes drops where no window takes it, and leaves one the app
// does not accept.
func TestDisplayDragNobodyTakes(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	var source *xdndSource
	e.onLoop(func() {
		e.p.input.buttonsDown = 1 << 1
		if err := e.win.StartDrag(ui.DragRequest{ID: 3, Text: "x"}); err != nil {
			t.Error(err)
			return
		}
		source = e.p.xdnd.source
		// Over the window, which refuses (the handler has no answer), then outside it.
		source.onMotion(50, 50)
	})
	if source == nil {
		return
	}
	waitFor(t, func() bool { return len(e.h.dropEvents()) >= 1 }, "the window never heard of the drag")
	var err error
	e.onLoop(func() { err = e.win.StartDrag(ui.DragRequest{ID: 4, Text: "y"}) })
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagDragInProgress {
		t.Errorf("a second drag: %v", err)
	}
	e.onLoop(func() { source.onMotion(1270, 790); source.onRelease() })
	waitFor(t, func() bool { return len(e.h.dragResults()) == 1 }, "the drag never ended")
	if got := e.h.dragResults()[0]; got.ID != 3 || got.Action != ui.DropActionNone {
		t.Errorf("result = %+v", got)
	}
	waitFor(t, func() bool {
		ev := e.h.dropEvents()
		return ev[len(ev)-1].Phase == ui.DropPhaseLeave
	}, "the window never heard the drag leave")
	e.onLoop(func() { e.p.input.buttonsDown = 0 })
	e.onLoop(func() { err = e.win.StartDrag(ui.DragRequest{ID: 5, Text: "z"}) })
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagDragNoButton {
		t.Errorf("a drag without a button: %v", err)
	}
}

func TestDisplayDragFromAClosedWindowFails(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	var err error
	e.onLoop(func() {
		w := e.win.(*window)
		e.p.input.buttonsDown = 1 << 1
		w.Close()
		err = e.p.xdnd.startDrag(w, ui.DragRequest{ID: 1, Text: "x"})
	})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagDragSelectionFailed || !appErr.HasRoute("xdnd.startDrag.DragSelectionFailed") {
		t.Errorf("err = %v", err)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagNoBus = "NoBus"
)
