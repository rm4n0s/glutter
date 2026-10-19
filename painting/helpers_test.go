package painting

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/fakeclock"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/layout"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// TestMain points the cache of the font scan to a directory of its own.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "glutter-painting-test")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_CACHE_HOME", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		panic(err)
	}
	os.Exit(code)
}

type noSystemFonts struct{}

func (noSystemFonts) Match(context.Context, string, int, bool, string) ([]text.FontFile, error) {
	return nil, nil
}

// bundledEngine is a text engine with Roboto and Material Icons only, so the
// tests don't depend on the machine.
func bundledEngine(t *testing.T) *layout.Engine {
	t.Helper()
	c := text.NewCollection(text.CollectionOpts{Matcher: noSystemFonts{}})
	if err := fonts.Register(c); err != nil {
		t.Fatal(err)
	}
	return layout.NewEngine(context.Background(), c, layout.EngineOpts{OnFontError: func(err error) { t.Errorf("font error: %v", err) }})
}

// robotoStyle is a text style in Roboto at a size.
func robotoStyle(size float64) *TextStyle {
	return NewTextStyle(TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(size)})
}

// fakeHost is the engine of a test: it keeps the functions posted to the "UI
// thread" until the test runs them.
type fakeHost struct {
	mu        sync.Mutex
	posted    []func()
	scheduled int
}

func (h *fakeHost) ScheduleFrame()                        { h.mu.Lock(); h.scheduled++; h.mu.Unlock() }
func (h *fakeHost) RenderView(ui.ViewID, *ui.Scene) error { return nil }
func (h *fakeHost) RequestAppExit()                       {}
func (h *fakeHost) Post(fn func())                        { h.mu.Lock(); h.posted = append(h.posted, fn); h.mu.Unlock() }

// runPosted runs what was posted, including what that posts.
func (h *fakeHost) runPosted() int {
	n := 0
	for {
		h.mu.Lock()
		if len(h.posted) == 0 {
			h.mu.Unlock()
			return n
		}
		fn := h.posted[0]
		h.posted = h.posted[1:]
		h.mu.Unlock()
		fn()
		n++
	}
}

// paintingTester is a PaintingBinding on a fake engine and a fake clock.
type paintingTester struct {
	*PaintingBinding
	host  *fakeHost
	sched *scheduler.SchedulerBinding
	clock *fakeclock.Fake
	d     *ui.PlatformDispatcher
	// errors collects the errors nobody handled.
	errs []ImageErrorDetails
}

func newPaintingTester(t *testing.T, opts ...PaintingBindingOpts) *paintingTester {
	t.Helper()
	host := &fakeHost{}
	d := ui.NewPlatformDispatcher(host)
	clock := fakeclock.New(time.Time{})
	sched := scheduler.NewSchedulerBinding(d)
	pt := &paintingTester{host: host, sched: sched, clock: clock, d: d}
	var o PaintingBindingOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	o.ErrorReporter = func(e ImageErrorDetails) { pt.errs = append(pt.errs, e) }
	o.AfterFunc = func(dur time.Duration, fn func()) func() {
		tm := clock.AfterFunc(dur, fn)
		return func() { tm.Stop() }
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	pt.PaintingBinding = NewPaintingBinding(ctx, sched, o)
	return pt
}

// frame runs a frame at the raw time, like flutter_test's pump.
func (pt *paintingTester) frame(raw time.Duration) {
	pt.sched.HandleBeginFrame(foundation.Some(raw))
	pt.sched.HandleDrawFrame()
}

// waitFor runs what is posted until the condition holds, or fails the test.
func (pt *paintingTester) waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		pt.host.runPosted()
		if cond() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}
