//go:build darwin && !ios && glutter_display

package macos

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/mainthread"
	"github.com/rm4n0s/glutter/ui"
)

// The display tests open real windows on the logged-in desktop (SPEC §29.13): the app is an accessory app,
// so it never takes the focus from the user's apps, and every window is closed before a test ends.

func TestMain(m *testing.M) { mainthread.Main(m) }

// recorder is a platform.Handler that records what the platform reports.
type recorder struct {
	platform.NopHandler
	mu       sync.Mutex
	metrics  []platform.ViewMetrics
	frames   []ui.ViewID
	frameAt  []time.Duration
	focus    []bool
	closeReq int
	onFrame  func(view ui.ViewID)
	pointers []ui.PointerData
	// keys are the key events that OnKeySync heard, which it answers with keyHandled (the recorder is a
	// platform.SyncInputHandler, as the engine's handler is).
	keys       []ui.KeyData
	keyHandled bool
	// texts are the events of the input method that OnTextInputSync heard.
	texts []ui.TextInputEvent
}

func (r *recorder) OnPointer(data []ui.PointerData) {
	r.mu.Lock()
	r.pointers = append(r.pointers, data...)
	r.mu.Unlock()
}

func (r *recorder) OnKeySync(d ui.KeyData) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys = append(r.keys, d)
	return r.keyHandled
}
func (r *recorder) OnPointerSync(data []ui.PointerData) { r.OnPointer(data) }
func (r *recorder) OnTextInputSync(e ui.TextInputEvent) {
	r.mu.Lock()
	r.texts = append(r.texts, e)
	r.mu.Unlock()
}

// takeTexts returns the text input events heard since the last call.
func (r *recorder) takeTexts() []ui.TextInputEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.texts
	r.texts = nil
	return out
}

// takeKeys returns the key events heard since the last call.
func (r *recorder) takeKeys() []ui.KeyData {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.keys
	r.keys = nil
	return out
}

// takePointers returns the pointer data heard since the last call.
func (r *recorder) takePointers() []ui.PointerData {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.pointers
	r.pointers = nil
	return out
}

func (r *recorder) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) {
	r.mu.Lock()
	r.metrics = append(r.metrics, m)
	r.mu.Unlock()
}

func (r *recorder) OnFrame(view ui.ViewID, at time.Duration) {
	r.mu.Lock()
	r.frames = append(r.frames, view)
	r.frameAt = append(r.frameAt, at)
	f := r.onFrame
	r.mu.Unlock()
	if f != nil {
		f(view)
	}
}

func (r *recorder) OnViewFocus(_ ui.ViewID, focused bool) {
	r.mu.Lock()
	r.focus = append(r.focus, focused)
	r.mu.Unlock()
}

func (r *recorder) OnCloseRequest(ui.ViewID) bool {
	r.mu.Lock()
	r.closeReq++
	r.mu.Unlock()
	return true
}

func (r *recorder) metricsSnapshot() []platform.ViewMetrics {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]platform.ViewMetrics(nil), r.metrics...)
}

func (r *recorder) frameCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.frames)
}

// session is a platform running on the main thread for the length of a test: the test's own goroutine
// does the checks and talks to the UI thread with Post.
type session struct {
	t    *testing.T
	p    *Platform
	h    *recorder
	done chan struct{}
}

// runApp makes the platform, calls body on a goroutine of its own and runs the loop until body returns.
// body uses s.onUI to run what must be on the main thread.
func runApp(t *testing.T, opts Options, h *recorder, body func(s *session)) {
	t.Helper()
	opts.Accessory = true
	if opts.Getenv == nil {
		opts.Getenv = linkEnvOf("cv")
	}
	var runErr error
	mainthread.Do(func() {
		p, err := New(opts)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() {
			if err := p.Close(); err != nil {
				t.Error(err)
			}
		}()
		s := &session{t: t, p: p, h: h, done: make(chan struct{})}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			defer cancel()
			body(s)
		}()
		runErr = p.Run(ctx, h)
	})
	if runErr != nil {
		t.Error(runErr)
	}
}

// onUI runs fn on the UI thread and waits for it.
func (s *session) onUI(fn func()) {
	s.t.Helper()
	done := make(chan struct{})
	s.p.Post(func() {
		defer close(done)
		fn()
	})
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		s.t.Error("the UI thread did not run a posted function in 10 s")
	}
}

// waitFor polls until cond is true or the time is out.
func (s *session) waitFor(what string, d time.Duration, cond func() bool) bool {
	s.t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.t.Errorf("timed out waiting for %s", what)
	return false
}

func TestWindowOpensAndReportsMetrics(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		var w platform.Window
		var err error
		s.onUI(func() {
			w, err = s.p.NewWindow(platform.WindowConfig{Title: "Glutter test", Size: ui.Size{Width: 320, Height: 240}})
		})
		if err != nil {
			s.t.Error(err)
			return
		}
		if !s.waitFor("the first metrics", 5*time.Second, func() bool { return len(h.metricsSnapshot()) > 0 }) {
			return
		}
		m := h.metricsSnapshot()[0]
		if m.DevicePixelRatio < 1 {
			t.Errorf("scale = %v", m.DevicePixelRatio)
		}
		if got, want := m.PhysicalSize, (ui.Size{Width: 320 * m.DevicePixelRatio, Height: 240 * m.DevicePixelRatio}); got != want {
			t.Errorf("physical size = %v, want %v", got, want)
		}
		if h.frameCount() == 0 {
			t.Error("the first frame was not asked for before the window was shown")
		}
		s.onUI(w.Close)
	})
}

// newWindow makes a window on the UI thread.
func (s *session) newWindow(cfg platform.WindowConfig) *window {
	s.t.Helper()
	var w platform.Window
	var err error
	s.onUI(func() { w, err = s.p.NewWindow(cfg) })
	if err != nil {
		s.t.Fatal(err)
	}
	return w.(*window)
}

// lastMetrics is the last metrics the handler heard.
func (h *recorder) lastMetrics() (platform.ViewMetrics, bool) {
	m := h.metricsSnapshot()
	if len(m) == 0 {
		return platform.ViewMetrics{}, false
	}
	return m[len(m)-1], true
}

func TestSetSizeReportsMetricsAndDrawsBeforeItReturns(t *testing.T) {
	h := &recorder{}
	// What the handler knew when each frame was asked for: the size of the last metrics.
	var frameSizes []ui.Size
	h.onFrame = func(ui.ViewID) {
		if m, ok := h.lastMetrics(); ok {
			frameSizes = append(frameSizes, m.PhysicalSize)
		}
	}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		var framesBefore, framesAfter, metricsBefore int
		var last platform.ViewMetrics
		s.onUI(func() {
			framesBefore, metricsBefore = h.frameCount(), len(h.metricsSnapshot())
			w.SetSize(ui.Size{Width: 420, Height: 310})
			framesAfter = h.frameCount()
			last, _ = h.lastMetrics()
		})
		scale := last.DevicePixelRatio
		if want := (ui.Size{Width: 420 * scale, Height: 310 * scale}); last.PhysicalSize != want {
			t.Errorf("metrics after SetSize = %v, want %v", last.PhysicalSize, want)
		}
		if len(h.metricsSnapshot()) <= metricsBefore {
			t.Error("SetSize did not report new metrics before it returned")
		}
		if framesAfter <= framesBefore {
			t.Error("SetSize did not draw a frame before it returned")
		}
		s.onUI(func() {
			if got := frameSizes[len(frameSizes)-1]; got != last.PhysicalSize {
				t.Errorf("the frame was asked for with the old size %v, want %v", got, last.PhysicalSize)
			}
		})
		s.onUI(w.Close)
	})
}

func TestSizeConstraintsHold(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		s.onUI(func() {
			w.SetSizeConstraints(ui.Size{Width: 250, Height: 180}, ui.Size{Width: 500, Height: 400})
			w.SetSize(ui.Size{Width: 100, Height: 100})
			m, _ := h.lastMetrics()
			if want := (ui.Size{Width: 250 * m.DevicePixelRatio, Height: 180 * m.DevicePixelRatio}); m.PhysicalSize != want {
				t.Errorf("smaller than the minimum: %v, want %v", m.PhysicalSize, want)
			}
			w.SetSize(ui.Size{Width: 900, Height: 900})
			m, _ = h.lastMetrics()
			if want := (ui.Size{Width: 500 * m.DevicePixelRatio, Height: 400 * m.DevicePixelRatio}); m.PhysicalSize != want {
				t.Errorf("larger than the maximum: %v, want %v", m.PhysicalSize, want)
			}
			w.SetSizeConstraints(ui.Size{}, ui.Size{})
			w.SetSize(ui.Size{Width: 900, Height: 700})
			m, _ = h.lastMetrics()
			if want := (ui.Size{Width: 900 * m.DevicePixelRatio, Height: 700 * m.DevicePixelRatio}); m.PhysicalSize != want {
				t.Errorf("no constraints: %v, want %v", m.PhysicalSize, want)
			}
		})
		s.onUI(w.Close)
	})
}

func TestWindowStatesAreReported(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 400, Height: 300}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		info := func() ui.WindowInfo {
			m, _ := h.lastMetrics()
			return m.Window
		}
		set := func(st platform.WindowState) { s.onUI(func() { w.SetState(st) }) }

		set(platform.WindowStateMinimized)
		s.waitFor("minimized", 8*time.Second, func() bool { return info().Minimized })
		set(platform.WindowStateNormal)
		s.waitFor("restored from minimized", 8*time.Second, func() bool { return !info().Minimized })

		set(platform.WindowStateMaximized)
		s.waitFor("maximized (zoomed)", 8*time.Second, func() bool { return info().Maximized })
		set(platform.WindowStateNormal)
		s.waitFor("restored from zoom", 8*time.Second, func() bool { return !info().Maximized })

		// Full screen is a space of its own; an accessory app is allowed it on current systems. When the
		// system refuses, the test says so and goes on.
		set(platform.WindowStateFullscreen)
		if fs := waitQuiet(3*time.Second, func() bool { return info().Fullscreen }); fs {
			set(platform.WindowStateNormal)
			s.waitFor("restored from full screen", 8*time.Second, func() bool { return !info().Fullscreen })
		} else {
			t.Log("the system did not make the window of an accessory app full screen")
		}
		s.onUI(w.Close)
	})
}

// waitQuiet polls like waitFor but only says whether the condition came true.
func waitQuiet(d time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestHiddenWindowAsksForNoFrame(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		visible := func() bool {
			var v bool
			s.onUI(func() { v = send[bool](w.ns, "isVisible") })
			return v
		}
		if !visible() {
			t.Error("the window is not visible after the first frame")
		}
		s.onUI(func() { w.SetHidden(true) })
		if visible() {
			t.Error("the window is visible after SetHidden(true)")
		}
		before := h.frameCount()
		s.onUI(w.RequestFrame)
		time.Sleep(300 * time.Millisecond)
		if got := h.frameCount(); got != before {
			t.Errorf("%d frames were drawn while hidden", got-before)
		}
		s.onUI(func() { w.SetHidden(false) })
		if !visible() {
			t.Error("the window is not visible after SetHidden(false)")
		}
		s.onUI(func() { w.RequestFrame() })
		s.waitFor("the frame that was asked for while hidden", 5*time.Second, func() bool { return h.frameCount() > before })
		s.onUI(w.Close)
	})
}

func TestPostRunsInTheTrackingModes(t *testing.T) {
	// A menu being tracked and a window being resized run the loop in NSEventTrackingRunLoopMode: a posted
	// function must run there too (SPEC §29.3).
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		_ = w
		ran := make(chan struct{})
		var inTracking bool
		// The UI thread runs the loop in the tracking mode for a while, as a menu does.
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.p.Post(func() { close(ran) })
		}()
		s.onUI(func() {
			mode := cfModeNamed("NSEventTrackingRunLoopMode")
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				runLoopOnce(mode, 50*time.Millisecond)
				select {
				case <-ran:
					inTracking = true
					return
				default:
				}
			}
		})
		if !inTracking {
			t.Error("a posted function did not run while the loop ran in NSEventTrackingRunLoopMode")
		}
		s.onUI(w.Close)
	})
}

func TestIdleAppWakesRarely(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		time.Sleep(500 * time.Millisecond) // the last frame and what follows it
		before := s.p.Wakes()
		time.Sleep(3 * time.Second)
		if n := s.p.Wakes() - before; n > 5 { // fewer than 2 a second
			t.Errorf("an idle app woke %d times in 3 s", n)
		}
		s.onUI(w.Close)
	})
}

func TestRequestFrameIsAnsweredByTheDisplayLink(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		s.onUI(func() {
		})
		for i := 0; i < 3; i++ {
			before := h.frameCount()
			s.onUI(w.RequestFrame)
			s.waitFor("a frame for the request", 3*time.Second, func() bool { return h.frameCount() > before })
		}
		var running bool
		time.Sleep(200 * time.Millisecond)
		s.onUI(func() { running = w.pacer.running })
		if running {
			t.Error("the display link still runs when no frame is wanted")
		}
		s.onUI(w.Close)
	})
}

// linkEnvOf is an environment that picks the display link ("cv" is CVDisplayLink, "ca" the view's
// CADisplayLink): the tests use CVDisplayLink, which ticks whether or not the display is awake and shows
// the window, so that they give the same result in a session that nobody looks at.
func linkEnvOf(kind string) func(string) string {
	return func(k string) string {
		if k == linkEnv {
			return kind
		}
		return os.Getenv(k)
	}
}

func TestViewDisplayLinkTicksForAVisibleWindow(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{Getenv: linkEnvOf("ca")}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		var visible bool
		s.onUI(func() { visible = send[uint64](w.ns, "occlusionState")&2 != 0 })
		if !visible {
			t.Skip("the window is not visible to the window server (the screen is asleep or locked): CADisplayLink does not tick")
		}
		for i := 0; i < 3; i++ {
			before := h.frameCount()
			s.onUI(w.RequestFrame)
			s.waitFor("a frame for the request", 3*time.Second, func() bool { return h.frameCount() > before })
		}
		s.onUI(w.Close)
	})
}

// layerPixels renders the layer of the window into a bitmap, as Core Animation would show it, and returns the
// pixels (BGRA premultiplied, top row first) with the size; it needs no screen.
func layerPixels(t *testing.T, w *window, width, height int) []byte {
	t.Helper()
	cg, err := dl.OpenFramework("CoreGraphics")
	if err != nil {
		t.Fatal(err)
	}
	var space func() uintptr
	var create func(data uintptr, w, h, bpc, bpr uint64, space uintptr, info uint32) uintptr
	var data func(ctx uintptr) uintptr
	var release func(uintptr)
	for _, b := range []struct {
		f any
		n string
	}{{&space, "CGColorSpaceCreateDeviceRGB"}, {&create, "CGBitmapContextCreate"}, {&data, "CGBitmapContextGetData"},
		{&release, "CGContextRelease"}} {
		if err := cg.Bind(b.f, b.n); err != nil {
			t.Fatal(err)
		}
	}
	sp := space()
	// kCGImageAlphaPremultipliedFirst | kCGBitmapByteOrder32Little: BGRA in memory.
	ctx := create(0, uint64(width), uint64(height), 8, uint64(width*4), sp, 2|(2<<12))
	if ctx == 0 {
		t.Fatal("no bitmap context")
	}
	defer release(ctx)
	w.layer.Send(sel("renderInContext:"), ctx)
	return append([]byte(nil), unsafe.Slice((*byte)(dl.Pointer(data(ctx))), width*height*4)...)
}

func TestSoftFrameShowsUpRightWayUp(t *testing.T) {
	h := &recorder{}
	var w *window
	h.onFrame = func(ui.ViewID) {
		if w == nil {
			return
		}
		sz := send[nsRect](w.view, "bounds").Size
		px, py := int(sz.Width*w.scale), int(sz.Height*w.scale)
		buf, err := w.AcquirePixels(px, py)
		if err != nil {
			t.Error(err)
			return
		}
		for y := 0; y < py; y++ {
			row := buf.Pix[y*buf.Stride : y*buf.Stride+px*4]
			for x := 0; x < px; x++ {
				// The top half is red, the bottom half blue, and the first column is green: BGRA.
				c := [4]byte{0, 0, 255, 255}
				if y >= py/2 {
					c = [4]byte{255, 0, 0, 255}
				}
				if x == 0 {
					c = [4]byte{0, 255, 0, 255}
				}
				copy(row[x*4:], c[:])
			}
		}
		if err := w.PresentPixels(nil); err != nil {
			t.Error(err)
		}
	}
	runApp(t, Options{}, h, func(s *session) {
		s.onUI(func() {
			ww, err := s.p.NewWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}})
			if err != nil {
				t.Error(err)
				return
			}
			w = ww.(*window)
		})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		var px []byte
		var pw, ph int
		s.onUI(func() {
			pw, ph = int(200*w.scale), int(100*w.scale)
			px = layerPixels(t, w, pw, ph)
		})
		at := func(x, y int) [3]byte { i := (y*pw + x) * 4; return [3]byte{px[i+2], px[i+1], px[i]} } // R, G, B
		if got := at(pw/2, ph/4); got != [3]byte{255, 0, 0} {
			t.Errorf("the top of the window is %v, want red", got)
		}
		if got := at(pw/2, 3*ph/4); got != [3]byte{0, 0, 255} {
			t.Errorf("the bottom of the window is %v, want blue", got)
		}
		if got := at(0, ph/4); got != [3]byte{0, 255, 0} {
			t.Errorf("the left column is %v, want green", got)
		}
		s.onUI(w.Close)
	})
}

func TestRunTwiceFailsAndTheDragComesLater(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		s.onUI(func() {
			err := s.p.Run(context.Background(), h)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != TagPlatformAlreadyRunning || !appErr.HasRoute("Platform.Run."+TagPlatformAlreadyRunning) {
				t.Errorf("a second Run: %v", err)
			}
			err = w.StartDrag(ui.DragRequest{})
			appErr, ok = errors.FromError(err)
			if !ok || appErr.Tag != TagMacOSDragUnavailable || !appErr.HasRoute("window.StartDrag."+TagMacOSDragUnavailable) {
				t.Errorf("StartDrag: %v", err)
			}
		})
		s.onUI(w.Close)
	})
}

func TestKeepScreenOnHoldsAndReleasesAnAssertion(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		s.onUI(func() {
			if err := w.KeepScreenOn(true); err != nil {
				t.Error(err)
			}
			if w.sleepAssertion == 0 {
				t.Error("no assertion is held")
			}
			if err := w.KeepScreenOn(true); err != nil { // once more: still one assertion
				t.Error(err)
			}
			if err := w.KeepScreenOn(false); err != nil {
				t.Error(err)
			}
			if w.sleepAssertion != 0 {
				t.Error("the assertion was not released")
			}
			// An assertion left at Close is released with the window.
			if err := w.KeepScreenOn(true); err != nil {
				t.Error(err)
			}
		})
		s.onUI(func() {
			w.Close()
			if w.sleepAssertion != 0 {
				t.Error("Close kept the assertion")
			}
		})
	})
}

func TestClosingTheLastWindowEndsRun(t *testing.T) {
	h := &recorder{}
	var closedAt atomic.Int64 // the body runs on another goroutine than the test
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		s.onUI(w.Close)
		closedAt.Store(time.Now().UnixNano())
		// The body does not end the loop: Run has to end by itself, before this wait is over.
		time.Sleep(5 * time.Second)
	})
	if d := time.Since(time.Unix(0, closedAt.Load())); d > 3*time.Second {
		t.Errorf("Run ended %v after the last window closed, which is the body's own timeout and not the platform's end", d)
	}
}

func TestCloseButtonAsksTheHandler(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		s.onUI(func() { w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionClose}) })
		s.waitFor("the close request", 3*time.Second, func() bool {
			h.mu.Lock()
			defer h.mu.Unlock()
			return h.closeReq > 0
		})
		s.waitFor("the window to go", 3*time.Second, func() bool {
			var n int
			s.onUI(func() { n = s.p.windowCount() })
			return n == 0
		})
	})
}
