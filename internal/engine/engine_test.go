package engine

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/internal/platform/linux"
	"github.com/rm4n0s/glutter/ui"
)

// conn is the fake platform with a Close, so that it is a linux.Connection.
type conn struct {
	*fake.Platform
	closeErr error
	closed   bool
}

func (c *conn) Close() error { c.closed = true; return c.closeErr }

// failingWindow is a connection whose NewWindow fails.
type failingWindow struct {
	*conn
	err error
}

func (c *failingWindow) NewWindow(platform.WindowConfig) (platform.Window, error) { return nil, c.err }

// drawCall is one call of the fake renderer.
type drawCall struct {
	handle        gpu.SurfaceHandle
	width, height int
	scene         *ui.Scene
	capture       bool
}

// fakeRenderer records its calls and returns the image it is told to.
type fakeRenderer struct {
	mu     sync.Mutex
	calls  []drawCall
	closed int
	err    error
	img    *image.RGBA
}

func (r *fakeRenderer) Draw(h gpu.SurfaceHandle, w, ht int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, drawCall{h, w, ht, scene, capture})
	if r.err != nil {
		return nil, r.err
	}
	if capture {
		return r.img, nil
	}
	return nil, nil
}

func (r *fakeRenderer) Close() {
	r.mu.Lock()
	r.closed++
	r.mu.Unlock()
}

func (r *fakeRenderer) snapshot() ([]drawCall, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]drawCall(nil), r.calls...), r.closed
}

// quiet is a logger that says nothing.
func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)) }

// harness is an engine on a fake platform.
type harness struct {
	c   *conn
	r   *fakeRenderer
	cfg Config

	mu    sync.Mutex // guards the fields below, set by the Window hook
	stop  func()
	win   platform.Window
	count int // how many times the Window hook ran
}

// hooked reports what the Window hook saw.
func (h *harness) hooked() (stop func(), win platform.Window, count int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stop, h.win, h.count
}

// stopEngine ends the run, once the Window hook gave the stop function.
func (h *harness) stopEngine(t *testing.T) {
	t.Helper()
	waitFor(t, func() bool { stop, _, _ := h.hooked(); return stop != nil })
	stop, _, _ := h.hooked()
	stop()
}

func newHarness(t *testing.T, env map[string]string) *harness {
	t.Helper()
	h := &harness{c: &conn{Platform: fake.New()}, r: &fakeRenderer{}}
	h.cfg = Config{
		Getenv: func(k string) string { return env[k] },
		Logger: quiet(),
		x11:    func() (linux.Connection, error) { return h.c, nil },
		wayland: func() (linux.Connection, error) {
			return nil, errors.New("WaylandConnectFailed", "no compositor")
		},
		newRenderer: func(*Config, *slog.Logger) renderer { return h.r },
	}
	h.cfg.Hooks.Window = func(win platform.Window, stop func()) {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.win, h.stop, h.count = win, stop, h.count+1
	}
	return h
}

// run runs the engine in a goroutine; the result is on the channel.
func (h *harness) run(t *testing.T) <-chan error {
	t.Helper()
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	go func() { done <- Run(ctx, h.cfg) }()
	return done
}

func result(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return")
		return nil
	}
}

func metrics(w, h float64) fake.MetricsEvent {
	return fake.MetricsEvent{View: 0, Metrics: platform.ViewMetrics{PhysicalSize: ui.Size{Width: w, Height: h}, DevicePixelRatio: 1}}
}

func TestRunDrawsTheWindowAndPassesInputToTheHooks(t *testing.T) {
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	var mu sync.Mutex
	var keys []ui.KeyData
	var batches [][]ui.PointerData
	var gotMetrics []platform.ViewMetrics
	h.cfg.Hooks.Key = func(d ui.KeyData) { mu.Lock(); keys = append(keys, d); mu.Unlock() }
	h.cfg.Hooks.Pointer = func(b []ui.PointerData) { mu.Lock(); batches = append(batches, b); mu.Unlock() }
	h.cfg.Hooks.Metrics = func(_ ui.ViewID, m platform.ViewMetrics) { mu.Lock(); gotMetrics = append(gotMetrics, m); mu.Unlock() }
	h.cfg.Window = platform.WindowConfig{Title: "mine", AppID: "dev.glutter.x", Resizable: true, Decorated: true}

	h.c.Inject(metrics(640, 400))
	h.c.Inject(fake.FrameEvent{View: 0, FrameTime: time.Millisecond})
	h.c.Inject(fake.KeyEvent{Data: ui.KeyData{Type: ui.KeyEventTypeDown, Logical: 'a'}})
	h.c.Inject(fake.PointerEvent{Data: []ui.PointerData{{Change: ui.PointerChangeAdd}, {Change: ui.PointerChangeHover}}})
	h.c.Inject(fake.FrameEvent{View: 0, FrameTime: 2 * time.Millisecond})
	h.c.Inject(fake.CloseRequestEvent{View: 0})
	done := h.run(t)
	// The fake platform runs until it is stopped; the close request destroyed the
	// renderer, so the test stops once that happened.
	for i := 0; i < 400; i++ {
		if _, closed := h.r.snapshot(); closed > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	h.stopEngine(t)
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}

	calls, closed := h.r.snapshot()
	if len(calls) != 2 || calls[0].width != 640 || calls[0].height != 400 || calls[0].capture {
		t.Errorf("draw calls %+v, want two at 640x400", calls)
	}
	if calls[0].handle.Kind != gpu.SurfaceKindFake {
		t.Errorf("the renderer got the handle %+v, want the window's", calls[0].handle)
	}
	if closed < 1 {
		t.Error("the renderer was not closed")
	}
	if !h.c.closed {
		t.Error("the connection was not closed")
	}
	if _, _, count := h.hooked(); count != 1 {
		t.Errorf("the Window hook ran %d times, want once", count)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(keys) != 1 || keys[0].Logical != 'a' || len(batches) != 1 || len(batches[0]) != 2 || len(gotMetrics) != 1 || gotMetrics[0].PhysicalSize.Width != 640 {
		t.Errorf("hooks saw keys %v, batches %v, metrics %v", keys, batches, gotMetrics)
	}
	// The window was made from the config.
	w := h.c.Windows()[0]
	if cfg := w.Config(); cfg.Title != "mine" || cfg.AppID != "dev.glutter.x" {
		t.Errorf("window config %+v", cfg)
	}
}

func TestRunGivesTheDefaultWindowWhenTheConfigHasNone(t *testing.T) {
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	done := h.run(t)
	h.stopEngine(t)
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
	cfg := h.c.Windows()[0].Config()
	want := platform.DefaultWindowConfig()
	want.Title = "Glutter"
	if cfg != want {
		t.Errorf("window config %+v, want %+v", cfg, want)
	}
}

func TestRunWithoutWindowHook(t *testing.T) {
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	h.cfg.Hooks = Hooks{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // returns at once
	if err := Run(ctx, h.cfg); err != nil {
		t.Fatal(err)
	}
	if !h.c.closed {
		t.Error("the connection was not closed")
	}
}

func TestRunSavesTheScreenshotAndReturns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shots", "frame.png")
	h := newHarness(t, map[string]string{"DISPLAY": ":0", ScreenshotEnv: path})
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for i := range 4 {
		img.SetRGBA(i%2, i/2, color.RGBA{0x67, 0x50, 0xA4, 255})
	}
	h.r.img = img

	h.c.Inject(metrics(2, 2))
	h.c.Inject(fake.FrameEvent{View: 0}) // the first frame arms the capture
	done := h.run(t)
	// With no delay the engine asks for the frame to capture.
	waitFor(t, func() bool { return hasEntry(h.c, fake.EntryRequestFrame) })
	h.c.Inject(fake.FrameEvent{View: 0})
	if err := result(t, done); err != nil {
		t.Fatalf("Run: %v", err)
	}
	calls, _ := h.r.snapshot()
	if len(calls) != 2 || calls[0].capture || !calls[1].capture {
		t.Errorf("draw calls %+v, want a plain frame and then a capturing one", calls)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	}()
	got, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if r, g, b, _ := got.At(1, 1).RGBA(); r>>8 != 0x67 || g>>8 != 0x50 || b>>8 != 0xA4 {
		t.Errorf("pixel = %x %x %x", r>>8, g>>8, b>>8)
	}
}

func TestRunScreenshotAfterADelay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "late.png")
	h := newHarness(t, map[string]string{"DISPLAY": ":0", ScreenshotEnv: path, ScreenshotAfterEnv: "50ms"})
	h.r.img = image.NewRGBA(image.Rect(0, 0, 1, 1))
	h.c.Inject(metrics(1, 1))
	h.c.Inject(fake.FrameEvent{View: 0})
	start := time.Now()
	done := h.run(t)
	// The request for the frame to capture comes only after the delay.
	waitFor(t, func() bool { return hasEntry(h.c, fake.EntryRequestFrame) })
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Errorf("the capture was asked for after %v, before the 50ms delay", elapsed)
	}
	h.c.Inject(fake.FrameEvent{View: 0})
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("no screenshot: %v", err)
	}
}

func countEntries(c *conn, kind fake.EntryKind) int {
	n := 0
	for _, e := range c.Log() {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

func hasEntry(c *conn, kind fake.EntryKind) bool {
	for _, e := range c.Log() {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func assertRoute(t *testing.T, err error, tag, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tag, route)
	}
}

func TestRunErrors(t *testing.T) {
	t.Run("no display", func(t *testing.T) {
		h := newHarness(t, nil)
		assertRoute(t, Run(context.Background(), h.cfg), "NoDisplay", "Run->Select.NoDisplay")
	})
	t.Run("an unknown backend", func(t *testing.T) {
		h := newHarness(t, map[string]string{"DISPLAY": ":0"})
		h.cfg.Backend = "quartz"
		assertRoute(t, Run(context.Background(), h.cfg), "BackendUnknown", "Run->Select.BackendUnknown")
	})
	t.Run("the backend's own error comes back unchanged", func(t *testing.T) {
		h := newHarness(t, map[string]string{"WAYLAND_DISPLAY": "wayland-0"})
		h.cfg.Backend = "wayland"
		assertRoute(t, Run(context.Background(), h.cfg), "WaylandConnectFailed", "Run->Select->connect->")
	})
	t.Run("a bad screenshot delay", func(t *testing.T) {
		h := newHarness(t, map[string]string{"DISPLAY": ":0", ScreenshotEnv: "x.png", ScreenshotAfterEnv: "soon"})
		assertRoute(t, Run(context.Background(), h.cfg), "ScreenshotAfterInvalid", "Run->ScreenshotFromEnv.ScreenshotAfterInvalid")
	})
	t.Run("a screenshot of the shared-memory mode", func(t *testing.T) {
		h := newHarness(t, map[string]string{"DISPLAY": ":0", ScreenshotEnv: "x.png"})
		h.cfg.ShmFill = true
		assertRoute(t, Run(context.Background(), h.cfg), "ScreenshotUnsupported", "Run.ScreenshotUnsupported")
	})
	t.Run("the window cannot be made", func(t *testing.T) {
		h := newHarness(t, map[string]string{"DISPLAY": ":0"})
		want := errors.New("X11WindowFailed", "XCreateWindow failed")
		fw := &failingWindow{conn: h.c, err: want}
		h.cfg.x11 = func() (linux.Connection, error) { return fw, nil }
		err := Run(context.Background(), h.cfg)
		assertRoute(t, err, "X11WindowFailed", "X11WindowFailed")
		if !h.c.closed {
			t.Error("the connection was not closed after the failure")
		}
	})
	t.Run("the renderer fails on the first frame", func(t *testing.T) {
		h := newHarness(t, map[string]string{"DISPLAY": ":0"})
		h.r.err = errors.New("EGLNoConfig", "no config")
		h.c.Inject(metrics(10, 10))
		h.c.Inject(fake.FrameEvent{View: 0})
		done := h.run(t)
		assertRoute(t, result(t, done), "EGLNoConfig", "EGLNoConfig")
		if _, closed := h.r.snapshot(); closed != 1 {
			t.Errorf("the renderer was closed %d times, want once", closed)
		}
	})
	t.Run("a failure to close is reported when nothing else failed", func(t *testing.T) {
		h := newHarness(t, map[string]string{"DISPLAY": ":0"})
		h.c.closeErr = errors.New("LoopSyscallFailed", "close")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assertRoute(t, Run(ctx, h.cfg), "LoopSyscallFailed", "LoopSyscallFailed")
	})
	t.Run("a failure to close does not hide the earlier failure", func(t *testing.T) {
		h := newHarness(t, nil)
		h.cfg.x11 = func() (linux.Connection, error) { return h.c, nil }
		h.c.closeErr = errors.New("LoopSyscallFailed", "close")
		h.cfg.Getenv = func(k string) string {
			if k == "DISPLAY" {
				return ":0"
			}
			return ""
		}
		h.r.err = errors.New("EGLInitFailed", "no display")
		h.c.Inject(metrics(10, 10))
		h.c.Inject(fake.FrameEvent{View: 0})
		assertRoute(t, result(t, h.run(t)), "EGLInitFailed", "EGLInitFailed")
	})
}

func TestShmFillRefusedByTheX11Connector(t *testing.T) {
	cfg := Config{ShmFill: true}
	_, err := cfg.x11Connector()()
	assertRoute(t, err, "ShmFillUnsupported", "ShmFillUnsupported")
}

func TestRealConnectorsReportTheirFailures(t *testing.T) {
	// No server is listening: the real connectors fail with their own tags, as a
	// session with a broken DISPLAY or WAYLAND_DISPLAY would.
	t.Setenv("DISPLAY", ":9999")
	t.Setenv("WAYLAND_DISPLAY", "glutter-no-such-socket")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	cfg := Config{Logger: quiet()}
	_, err := cfg.x11Connector()()
	assertRoute(t, err, "X11OpenDisplayFailed", "New.X11OpenDisplayFailed")
	_, err = cfg.waylandConnector()()
	if appErr, ok := errors.FromError(err); !ok || (appErr.Tag != "WaylandConnectFailed" && appErr.Tag != "LibraryLoadFailed") {
		t.Errorf("Wayland error = %v, want WaylandConnectFailed (or LibraryLoadFailed without libwayland)", err)
	}
	// With both broken, Run says there is no display.
	err = Run(context.Background(), Config{Logger: quiet()})
	assertRoute(t, err, "NoDisplay", "Run->Select.NoDisplay")
	appErr, _ := errors.FromError(err)
	if tried, _ := appErr.Metadata["tried"].([]string); len(tried) != 2 {
		t.Errorf("tried %v, want Wayland and X11", appErr.Metadata["tried"])
	}
}

func TestRunDrawsTheSceneOfTheSceneHook(t *testing.T) {
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	var mu sync.Mutex
	type ask struct {
		size ui.Size
		dpr  float64
	}
	var asks []ask
	want := ui.NewSceneBuilder().Build()
	h.cfg.Scene = func(size ui.Size, dpr float64) *ui.Scene {
		mu.Lock()
		defer mu.Unlock()
		asks = append(asks, ask{size, dpr})
		return want
	}
	h.c.Inject(fake.MetricsEvent{View: 0, Metrics: platform.ViewMetrics{PhysicalSize: ui.Size{Width: 300, Height: 200}, DevicePixelRatio: 1.5}})
	h.c.Inject(fake.FrameEvent{View: 0, FrameTime: time.Millisecond})
	done := h.run(t)
	waitFor(t, func() bool { calls, _ := h.r.snapshot(); return len(calls) == 1 })
	h.stopEngine(t)
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
	calls, _ := h.r.snapshot()
	if calls[0].scene != want {
		t.Error("the renderer did not get the scene of the hook")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(asks) != 1 || asks[0].size != (ui.Size{Width: 300, Height: 200}) || asks[0].dpr != 1.5 {
		t.Errorf("the hook was asked %+v, want one call with the window's size and scale", asks)
	}
}

func TestRunWithoutASceneHookDrawsNoScene(t *testing.T) {
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	h.c.Inject(metrics(100, 100))
	h.c.Inject(fake.FrameEvent{View: 0, FrameTime: time.Millisecond})
	done := h.run(t)
	waitFor(t, func() bool { calls, _ := h.r.snapshot(); return len(calls) == 1 })
	h.stopEngine(t)
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
	if calls, _ := h.r.snapshot(); calls[0].scene != nil {
		t.Error("a scene came from nowhere")
	}
}

func TestInputWaitsForTheNextFrameAndIsHandledBeforeTheScene(t *testing.T) {
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	var mu sync.Mutex
	var order []string
	note := func(s string) { mu.Lock(); order = append(order, s); mu.Unlock() }
	h.cfg.Hooks.Key = func(d ui.KeyData) { note(string(rune(d.Logical))) }
	h.cfg.Hooks.Pointer = func(b []ui.PointerData) { note("pointer" + string(rune('0'+len(b)))) }
	h.cfg.Scene = func(ui.Size, float64) *ui.Scene { note("scene"); return nil }

	h.c.Inject(metrics(10, 10))
	h.c.Inject(fake.FrameEvent{View: 0})
	done := h.run(t)
	waitFor(t, func() bool { mu.Lock(); defer mu.Unlock(); return len(order) == 1 })

	// Input between frames asks for a frame, and is not handled until it comes.
	h.c.Inject(fake.KeyEvent{Data: ui.KeyData{Type: ui.KeyEventTypeDown, Logical: 'a'}})
	h.c.Inject(fake.PointerEvent{Data: []ui.PointerData{{Change: ui.PointerChangeHover}, {Change: ui.PointerChangeHover}}})
	h.c.Inject(fake.KeyEvent{Data: ui.KeyData{Type: ui.KeyEventTypeDown, Logical: 'b'}})
	waitFor(t, func() bool { return countEntries(h.c, fake.EntryRequestFrame) == 3 })
	mu.Lock()
	if len(order) != 1 {
		t.Errorf("input was handled before a frame: %v", order)
	}
	mu.Unlock()

	// One frame handles all of it, in order, before its scene is built.
	h.c.Inject(fake.FrameEvent{View: 0})
	waitFor(t, func() bool { mu.Lock(); defer mu.Unlock(); return len(order) == 5 })
	mu.Lock()
	want := []string{"scene", "a", "pointer2", "b", "scene"}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("order = %v, want %v", order, want)
			break
		}
	}
	mu.Unlock()
	h.stopEngine(t)
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
}

func TestInputIsHandledAtOnceInTheSharedMemoryMode(t *testing.T) {
	var keys []rune
	h := &handler{cfg: &Config{ShmFill: true, Hooks: Hooks{Key: func(d ui.KeyData) { keys = append(keys, rune(d.Logical)) }}}}
	h.OnKey(ui.KeyData{Logical: 'x'})
	if len(keys) != 1 || keys[0] != 'x' || len(h.input.pending) != 0 {
		t.Errorf("keys %v, queued %d: with no frames there is nothing to wait for", keys, len(h.input.pending))
	}
}

func TestInputQueueFlushesInOrder(t *testing.T) {
	var queue inputQueue
	var got []int
	queue.add(func() { got = append(got, 1); queue.add(func() { got = append(got, 3) }) })
	queue.add(func() { got = append(got, 2) })
	queue.flush()
	// What the handling queues waits for the next flush.
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("handled %v", got)
	}
	queue.flush()
	queue.flush()
	if len(got) != 3 || got[2] != 3 {
		t.Errorf("handled %v", got)
	}
}
