package android

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// fakeHost is a host that a test drives: Poll returns when the test fires a vsync or the loop is
// woken.
type fakeHost struct {
	wake chan struct{}
	fire chan time.Duration

	mu       sync.Mutex
	vsyncs   int
	released []uintptr
	// keyAnswers are the answers to key events, "id:handled"; cursors the cursors set.
	keyAnswers []string
	cursors    []int
	injected   []DevEvent
	// commands are the calls to the activity; fontScale is how ScaledFontSize scales (1 when 0).
	commands  []command
	fontScale float64
}

func newFakeHost() *fakeHost {
	return &fakeHost{wake: make(chan struct{}, 1), fire: make(chan time.Duration, 8)}
}

func (f *fakeHost) Poll() (time.Duration, bool) {
	select {
	case <-f.wake:
		return 0, false
	case t := <-f.fire:
		return t, true
	}
}

func (f *fakeHost) Wake() {
	select {
	case f.wake <- struct{}{}:
	default:
	}
}

func (f *fakeHost) RequestVsync() {
	f.mu.Lock()
	f.vsyncs++
	f.mu.Unlock()
}

func (f *fakeHost) KeyHandled(id int, handled bool) {
	f.mu.Lock()
	f.keyAnswers = append(f.keyAnswers, fmt.Sprintf("%d:%v", id, handled))
	f.mu.Unlock()
}

func (f *fakeHost) Inject(e DevEvent) {
	f.mu.Lock()
	f.injected = append(f.injected, e)
	f.mu.Unlock()
}

func (f *fakeHost) Command(c command) {
	f.mu.Lock()
	f.commands = append(f.commands, c)
	f.mu.Unlock()
}

// ScaledFontSize scales by fontScale.
func (f *fakeHost) ScaledFontSize(size float64) float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	scale := f.fontScale
	if scale == 0 {
		scale = 1
	}
	return size * scale
}

func (f *fakeHost) sentCommands() []command {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]command(nil), f.commands...)
}

func (f *fakeHost) SetCursor(t int) {
	f.mu.Lock()
	f.cursors = append(f.cursors, t)
	f.mu.Unlock()
}

func (f *fakeHost) ReleaseWindow(w uintptr) {
	f.mu.Lock()
	f.released = append(f.released, w)
	f.mu.Unlock()
}

func (f *fakeHost) vsyncRequests() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.vsyncs
}

func (f *fakeHost) releasedWindows() []uintptr {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]uintptr(nil), f.released...)
}

// recorder is a platform.Handler that records what it is told, and can hold a frame.
type recorder struct {
	platform.NopHandler
	mu    sync.Mutex
	calls []string
	// batches are the changes of the pointer data, one list for each batch.
	batches  [][]string
	settings []platform.Settings
	metrics  platform.ViewMetrics
	// onFrame, when set, runs inside OnFrame.
	onFrame func(time.Duration)
}

func (r *recorder) add(format string, a ...any) {
	r.mu.Lock()
	r.calls = append(r.calls, fmt.Sprintf(format, a...))
	r.mu.Unlock()
}

func (r *recorder) log() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

func (r *recorder) count(prefix string) int {
	n := 0
	for _, c := range r.log() {
		if len(c) >= len(prefix) && c[:len(prefix)] == prefix {
			n++
		}
	}
	return n
}

func (r *recorder) OnFrame(_ ui.ViewID, t time.Duration) {
	r.add("frame %d", t)
	if r.onFrame != nil {
		r.onFrame(t)
	}
}
func (r *recorder) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) {
	r.mu.Lock()
	r.metrics = m
	r.mu.Unlock()
	r.add("metrics %dx%d@%.3f", int(m.PhysicalSize.Width), int(m.PhysicalSize.Height), m.DevicePixelRatio)
}
func (r *recorder) OnSettings(s platform.Settings) {
	r.mu.Lock()
	r.settings = append(r.settings, s)
	r.mu.Unlock()
	r.add("settings")
}

// lastSettings is what the last OnSettings reported.
func (r *recorder) lastSettings() platform.Settings {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.settings[len(r.settings)-1]
}

// lastMetrics is what the last OnMetrics reported.
func (r *recorder) lastMetrics() platform.ViewMetrics {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.metrics
}

func (r *recorder) OnLifecycle(s ui.AppLifecycleState) { r.add("lifecycle %v", s) }
func (r *recorder) OnViewFocus(_ ui.ViewID, f bool)    { r.add("focus %v", f) }
func (r *recorder) OnDisplaysChanged()                 { r.add("displays") }
func (r *recorder) OnPointer(data []ui.PointerData) {
	var changes []string
	for _, d := range data {
		changes = append(changes, d.Change.String())
	}
	r.mu.Lock()
	r.batches = append(r.batches, changes)
	r.mu.Unlock()
}

// pointerChanges are the changes of each batch of pointer data, by their names.
func (r *recorder) pointerChanges() [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([][]string(nil), r.batches...)
}

func (r *recorder) OnSurfaceLost(_ ui.ViewID, h gpu.SurfaceHandle) {
	r.add("lost %s:%d", h.Kind, h.Window)
}

var _ platform.SurfaceHandler = (*recorder)(nil)

// harness is a platform over fakes, running.
type harness struct {
	t    *testing.T
	b    *bridge
	host *fakeHost
	p    *Platform
	h    *recorder
	win  platform.Window
	done chan error
	stop func()
}

// newHarness makes a platform with its window, over fakes, that is not running yet.
func newHarness(t *testing.T) *harness {
	t.Helper()
	b := newBridge()
	b.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	host := newFakeHost()
	p := newPlatform(b, host, b.log)
	win, err := p.NewWindow(platform.WindowConfig{})
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, b: b, host: host, p: p, h: &recorder{}, win: win}
}

// run starts the loop on a goroutine and waits until it drains the queue.
func (h *harness) run() {
	h.t.Helper()
	ctx, cancel := contextWithCancel()
	h.done = make(chan error, 1)
	h.stop = func() {
		cancel()
		select {
		case err := <-h.done:
			if err != nil {
				h.t.Errorf("Run returned %v", err)
			}
		case <-time.After(5 * time.Second):
			h.t.Error("Run did not return after the context ended")
		}
	}
	go func() { h.done <- h.p.Run(ctx, h.h) }()
	h.t.Cleanup(func() {
		if h.stop != nil {
			h.stop()
			h.stop = nil
		}
	})
	waitFor(h.t, "Run to start", func() bool { return h.h.count("lifecycle") > 0 })
}

// sync runs fn on the UI thread and waits for it.
func (h *harness) sync(fn func()) {
	h.t.Helper()
	done := make(chan struct{})
	h.p.Post(func() { fn(); close(done) })
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		h.t.Fatal("the UI thread did not run the function")
	}
}

// waitFor polls cond for up to 5 seconds.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func contextWithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// request asks for a frame of the window on the UI thread and waits until it has been recorded.
func (h *harness) request() {
	h.t.Helper()
	h.sync(func() { h.win.RequestFrame() })
}
