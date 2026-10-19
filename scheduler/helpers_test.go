package scheduler

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/fakeclock"
	"github.com/rm4n0s/glutter/ui"
)

// fakeHost is the engine of a test: it counts the frames asked for and keeps
// the functions posted to the "UI thread" until the test runs them.
type fakeHost struct {
	scheduled int
	posted    []func()
	exits     int
}

func (h *fakeHost) ScheduleFrame()                        { h.scheduled++ }
func (h *fakeHost) RenderView(ui.ViewID, *ui.Scene) error { return nil }
func (h *fakeHost) RequestAppExit()                       { h.exits++ }
func (h *fakeHost) Post(fn func())                        { h.posted = append(h.posted, fn) }

// runPosted runs what was posted, including what that posts, like an event
// loop that is not busy. It reports how many functions ran.
func (h *fakeHost) runPosted() int {
	n := 0
	for len(h.posted) > 0 {
		fn := h.posted[0]
		h.posted = h.posted[1:]
		fn()
		n++
	}
	return n
}

// tester is a SchedulerBinding on a fake engine and a fake clock, with the
// pump of flutter_test: time passes, and if a frame is scheduled it runs.
type tester struct {
	*SchedulerBinding
	host  *fakeHost
	d     *ui.PlatformDispatcher
	clock *fakeclock.Fake
	start time.Time
}

func newTester() *tester {
	host := &fakeHost{}
	d := ui.NewPlatformDispatcher(host)
	clock := fakeclock.New(time.Time{})
	return &tester{SchedulerBinding: NewSchedulerBinding(d), host: host, d: d, clock: clock, start: clock.Now()}
}

// tick runs a frame at the raw time, like scheduler_tester.dart's tick.
func (tt *tester) tick(raw time.Duration) {
	tt.HandleBeginFrame(foundation.Some(raw))
	tt.HandleDrawFrame()
}

// pump advances the clock by d and, if a frame was scheduled, runs it at the
// clock's time, as WidgetTester.pump does. It reports whether a frame ran.
func (tt *tester) pump(d time.Duration) bool {
	tt.clock.Advance(d)
	if !tt.HasScheduledFrame() {
		return false
	}
	tt.d.BeginFrame(tt.clock.Since(tt.start))
	tt.d.DrawFrame()
	return true
}

// recoverError runs fn and returns the error it panicked with.
func recoverError(t *testing.T, fn func()) (err error) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic")
		}
		e, ok := r.(error)
		if !ok {
			t.Fatalf("expected an error panic, got %T (%v)", r, r)
		}
		err = e
	}()
	fn()
	return nil
}

// assertRoute checks that err is a tagged error whose route contains route.
func assertRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}
