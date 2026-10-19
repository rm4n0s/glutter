package animation

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/fakeclock"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// fakeHost is the engine of a test: it counts the frames asked for.
type fakeHost struct{ scheduled int }

func (h *fakeHost) ScheduleFrame()                        { h.scheduled++ }
func (h *fakeHost) RenderView(ui.ViewID, *ui.Scene) error { return nil }
func (h *fakeHost) RequestAppExit()                       {}
func (h *fakeHost) Post(fn func())                        { fn() }

// tester is a SchedulerBinding on a fake engine and a fake clock, and a
// TickerProvider: the TestVSync and the scheduler_tester.dart of flutter's
// animation tests.
type tester struct {
	*scheduler.SchedulerBinding
	d     *ui.PlatformDispatcher
	clock *fakeclock.Fake
	start time.Time

	disableAnimations bool
	reporter          foundation.ErrorReporter
	tickers           []*scheduler.Ticker
}

func newTester() *tester {
	d := ui.NewPlatformDispatcher(&fakeHost{})
	clock := fakeclock.New(time.Time{})
	return &tester{SchedulerBinding: scheduler.NewSchedulerBinding(d), d: d, clock: clock, start: clock.Now()}
}

// CreateTicker makes a ticker on the test binding.
func (tt *tester) CreateTicker(onTick scheduler.TickerCallback) *scheduler.Ticker {
	ticker := scheduler.NewTicker(tt.SchedulerBinding, onTick)
	tt.tickers = append(tt.tickers, ticker)
	return ticker
}

// vsync makes the tester an Environment, with the accessibility setting.
type envTester struct{ *tester }

func (e envTester) DisableAnimations() bool                 { return e.disableAnimations }
func (e envTester) ErrorReporter() foundation.ErrorReporter { return e.reporter }

// tick runs a frame at the raw time, like scheduler_tester.dart's tick.
func (tt *tester) tick(raw time.Duration) {
	tt.HandleBeginFrame(foundation.Some(raw))
	tt.HandleDrawFrame()
}

// pump advances the clock by d and runs a frame if one was scheduled.
func (tt *tester) pump(d time.Duration) bool {
	tt.clock.Advance(d)
	if !tt.HasScheduledFrame() {
		return false
	}
	tt.d.BeginFrame(tt.clock.Since(tt.start))
	tt.d.DrawFrame()
	return true
}

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func near(t *testing.T, name string, got, want, eps float64) {
	t.Helper()
	if math.Abs(got-want) > eps || math.IsNaN(got) {
		t.Errorf("%s = %v, want %v (±%v)", name, got, want, eps)
	}
}

// moreOrLess is flutter_test's moreOrLessEquals with its default epsilon.
func moreOrLess(t *testing.T, name string, got, want float64) {
	t.Helper()
	near(t, name, got, want, 1e-10)
}

func oneLine(t *testing.T, name string, v any) {
	t.Helper()
	s := ""
	if st, ok := v.(interface{ String() string }); ok {
		s = st.String()
	}
	if s == "" || strings.Contains(s, "\n") {
		t.Errorf("%s: description %q is not one line", name, s)
	}
}

// panicTag runs fn and returns the tag of the *errors.Error it panicked with,
// or "" when it did not panic.
func panicTag(t *testing.T, fn func()) (tag string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err, ok := r.(error)
		if !ok {
			t.Fatalf("panicked with %T (%v), want an error", r, r)
		}
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("panicked with %T, want *errors.Error", err)
		}
		tag = appErr.Tag
	}()
	fn()
	return ""
}

// wantPanic checks that fn panics with the tag (in debug builds only).
func wantPanic(t *testing.T, tag string, fn func()) {
	t.Helper()
	if !foundation.DebugMode {
		return
	}
	if got := panicTag(t, fn); got != tag {
		t.Errorf("panic tag = %q, want %q", got, tag)
	}
}

func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug-only checks of the framework")
	}
}

func approxSlice(t *testing.T, name string, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s = %v, want %v", name, got, want)
		return
	}
	for i := range got {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Errorf("%s = %v, want %v", name, got, want)
			return
		}
	}
}
