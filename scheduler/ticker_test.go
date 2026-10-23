package scheduler

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/scheduler/ticker_test.dart and binding_test.dart.

const frame = 10 * time.Millisecond

func TestTickerMuteControl(t *testing.T) {
	skipUnlessDebug(t)
	tt := newTester()
	ticks := 0
	ticker := NewTicker(tt.SchedulerBinding, func(time.Duration) { ticks++ })
	defer ticker.Dispose()

	type state struct{ ticking, active bool }
	check := func(step string, wantTicks int, want state) {
		t.Helper()
		if ticks != wantTicks || ticker.IsTicking() != want.ticking || ticker.IsActive() != want.active {
			t.Errorf("%s: ticks %d (want %d), ticking %v (want %v), active %v (want %v)",
				step, ticks, wantTicks, ticker.IsTicking(), want.ticking, ticker.IsActive(), want.active)
		}
	}
	check("new", 0, state{false, false})
	ticker.Start()
	check("started", 0, state{true, true})

	assertRoute(t, recoverError(t, func() { ticker.Start() }), "Ticker.Start.TickerStartedTwice")
	err := recoverError(t, func() { ticker.Start() })
	appErr, _ := errors.FromError(err)
	if got := appErr.Metadata["ticker"]; got != "Ticker()" {
		t.Errorf("the error names the ticker %v, want Ticker()", got)
	}

	tt.pump(frame)
	check("first frame", 1, state{true, true})
	ticker.SetMuted(true)
	tt.pump(frame)
	check("muted", 1, state{false, true})
	ticker.SetMuted(false)
	tt.pump(frame)
	check("unmuted", 2, state{true, true})
	ticker.SetMuted(true)
	tt.pump(frame)
	check("muted again", 2, state{false, true})
	ticker.Stop()
	check("stopped", 2, state{false, false})
	ticker.SetMuted(false)
	check("unmuted after stop", 2, state{false, false})
	tt.pump(frame)
	check("a frame after stop", 2, state{false, false})
}

func TestTickerElapsedTimeStartsAtTheFirstFrame(t *testing.T) {
	tt := newTester()
	var elapsed []time.Duration
	ticker := NewTicker(tt.SchedulerBinding, func(e time.Duration) { elapsed = append(elapsed, e) })
	defer ticker.Dispose()
	ticker.Start()
	for i := 0; i < 3; i++ {
		tt.pump(frame)
	}
	want := []time.Duration{0, frame, 2 * frame}
	if len(elapsed) != 3 || elapsed[0] != want[0] || elapsed[1] != want[1] || elapsed[2] != want[2] {
		t.Errorf("elapsed = %v, want %v", elapsed, want)
	}
}

func TestTickerToString(t *testing.T) {
	skipUnlessDebug(t)
	tt := newTester()
	tests := []struct {
		label string
		want  string
	}{
		{"", "Ticker()"},
		{"my label", "Ticker(my label)"},
	}
	for _, tc := range tests {
		ticker := NewTicker(tt.SchedulerBinding, func(time.Duration) {}, TickerOpts{DebugLabel: tc.label})
		if got := ticker.String(); got != tc.want {
			t.Errorf("String() = %q, want %q", got, tc.want)
		}
	}
}

func TestTickerTimeDilation(t *testing.T) {
	tests := []struct {
		name     string
		dilation float64
		want     time.Duration
	}{
		{"sped up", 0.5, 20 * time.Millisecond},  // moves twice as fast
		{"slowed down", 2, 5 * time.Millisecond}, // moves half as fast
		{"no dilation", 1, 10 * time.Millisecond},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tt := newTester()
			tt.SetTimeDilation(tc.dilation)
			var last time.Duration
			ticker := NewTicker(tt.SchedulerBinding, func(e time.Duration) { last = e })
			defer ticker.Dispose()
			ticker.Start()
			tt.pump(frame)
			tt.pump(frame)
			if last != tc.want {
				t.Errorf("last elapsed = %v, want %v", last, tc.want)
			}
		})
	}
}

func TestTickerStopsTickingWhenTheAppIsPaused(t *testing.T) {
	tt := newTester()
	ticks := 0
	ticker := NewTicker(tt.SchedulerBinding, func(time.Duration) { ticks++ })
	defer ticker.Dispose()
	ticker.Start()
	if !ticker.IsTicking() || !ticker.IsActive() || ticks != 0 {
		t.Fatal("started")
	}
	tt.HandleAppLifecycleStateChanged(ui.AppLifecycleStatePaused)
	if ticker.IsTicking() || !ticker.IsActive() {
		t.Errorf("paused: ticking %v, active %v", ticker.IsTicking(), ticker.IsActive())
	}
	ticker.Stop()
	tt.HandleAppLifecycleStateChanged(ui.AppLifecycleStateResumed)
}

func TestTickerCreatedBeforeTheAppUnpauses(t *testing.T) {
	tt := newTester()
	tt.HandleAppLifecycleStateChanged(ui.AppLifecycleStatePaused)
	ticks := 0
	ticker := NewTicker(tt.SchedulerBinding, func(time.Duration) { ticks++ })
	defer ticker.Dispose()
	ticker.Start()
	if ticks != 0 || ticker.IsTicking() {
		t.Fatalf("ticks %d, ticking %v", ticks, ticker.IsTicking())
	}
	tt.pump(frame)
	if ticks != 0 || ticker.IsTicking() {
		t.Fatalf("a paused app has no frames: ticks %d, ticking %v", ticks, ticker.IsTicking())
	}
	tt.HandleAppLifecycleStateChanged(ui.AppLifecycleStateResumed)
	tt.pump(frame)
	if ticks != 1 || !ticker.IsTicking() {
		t.Errorf("ticks %d, ticking %v", ticks, ticker.IsTicking())
	}
	ticker.Stop()
}

func TestTickerForceFramesRequestsForcedFrames(t *testing.T) {
	tt := newTester()
	tt.HandleAppLifecycleStateChanged(ui.AppLifecycleStatePaused)
	ticker := NewTicker(tt.SchedulerBinding, func(time.Duration) {})
	ticker.ForceFrames = true
	f := ticker.Start()
	// A forced frame is scheduled even though frames are disabled.
	if !tt.HasScheduledFrame() {
		t.Error("a forced frame should be scheduled")
	}
	ticker.Stop()
	select {
	case <-f.Done():
	default:
		t.Error("the future completes on Stop")
	}

	normal := newTester()
	normal.HandleAppLifecycleStateChanged(ui.AppLifecycleStatePaused)
	other := NewTicker(normal.SchedulerBinding, func(time.Duration) {})
	other.Start()
	if normal.HasScheduledFrame() {
		t.Error("a ticker without ForceFrames schedules no frame while frames are disabled")
	}
}

func TestTickerStartedInsideAFrameTakesTheFrameTime(t *testing.T) {
	tt := newTester()
	var elapsed []time.Duration
	var ticker *Ticker
	tt.ScheduleFrameCallback(func(time.Duration) {
		ticker = NewTicker(tt.SchedulerBinding, func(e time.Duration) { elapsed = append(elapsed, e) })
		ticker.Start()
	})
	tt.tick(100 * time.Millisecond) // the ticker starts in this frame, at its time (0 since the epoch)
	tt.tick(116 * time.Millisecond)
	if len(elapsed) != 1 || elapsed[0] != 16*time.Millisecond {
		t.Errorf("elapsed = %v, want [16ms]", elapsed)
	}
	ticker.Dispose()
}

func TestTickerStopAndStartInsideTheCallback(t *testing.T) {
	tt := newTester()
	ticks := 0
	var ticker *Ticker
	ticker = NewTicker(tt.SchedulerBinding, func(time.Duration) {
		ticks++
		if ticks == 1 {
			ticker.Stop()
			ticker.Start()
		}
	})
	defer ticker.Dispose()
	ticker.Start()
	tt.pump(frame)
	tt.pump(frame)
	if ticks != 2 || !ticker.IsActive() {
		t.Errorf("ticks %d, active %v: a ticker restarted in its callback goes on", ticks, ticker.IsActive())
	}
}

func TestTickerDisposeCancelsTheFuture(t *testing.T) {
	skipUnlessDebug(t)
	tt := newTester()
	ticker := NewTicker(tt.SchedulerBinding, func(time.Duration) {})
	f := ticker.Start()
	ticker.Dispose()
	if ticker.IsActive() || tt.TransientCallbackCount() != 0 {
		t.Errorf("active %v, transient callbacks %d", ticker.IsActive(), tt.TransientCallbackCount())
	}
	select {
	case <-f.Done():
		t.Fatal("a canceled ticker never completes its future")
	default:
	}
	err := <-f.OrCancel()
	assertRoute(t, err, "TickerFuture.OrCancel->TickerFuture.whenEnded.TickerCanceled")
	appErr, _ := errors.FromError(err)
	if !strings.Contains(appErr.Message, "was canceled before") {
		t.Errorf("message = %q: canceled before OrCancel was used", appErr.Message)
	}
	assertRoute(t, recoverError(t, func() { ticker.Start() }), "Ticker.Start.TickerUsedAfterDispose")
	ticker.Dispose() // twice is fine
}

func TestTickerFutureCompletion(t *testing.T) {
	tt := newTester()
	ticker := NewTicker(tt.SchedulerBinding, func(time.Duration) {}, TickerOpts{DebugLabel: "x"})
	f := ticker.Start()

	var log []string
	f.Then(func() { log = append(log, "then") })
	f.WhenCompleteOrCancel(func() { log = append(log, "end") })
	orCancel := f.OrCancel()
	if v, ended := f.Completed(); ended || v {
		t.Error("a running ticker's future has not ended")
	}
	if !strings.HasSuffix(ignoreHash(f.String()), "#00000(active)") {
		t.Errorf("String() = %q", f.String())
	}
	ticker.Stop()
	if got := strings.Join(log, ","); got != "then,end" {
		t.Errorf("callbacks = %q, want then,end", got)
	}
	select {
	case <-f.Done():
	default:
		t.Error("Done is closed after Stop")
	}
	if err := <-orCancel; err != nil {
		t.Errorf("OrCancel got %v, want nil", err)
	}
	// Asking after the end still works.
	f.Then(func() { log = append(log, "late then") })
	f.WhenCompleteOrCancel(func() { log = append(log, "late end") })
	if err := <-f.OrCancel(); err != nil {
		t.Errorf("late OrCancel got %v", err)
	}
	if got := strings.Join(log, ","); got != "then,end,late then,late end" {
		t.Errorf("callbacks = %q", got)
	}
	if v, ended := f.Completed(); !ended || !v || !strings.HasSuffix(ignoreHash(f.String()), "(complete)") {
		t.Errorf("Completed %v %v, String %q", v, ended, f.String())
	}
}

func TestTickerFutureCancel(t *testing.T) {
	skipUnlessDebug(t)
	tt := newTester()
	ticker := NewTicker(tt.SchedulerBinding, func(time.Duration) {}, TickerOpts{DebugLabel: "spinner"})
	f := ticker.Start()
	thenCalled := false
	endCalled := false
	f.Then(func() { thenCalled = true })
	f.WhenCompleteOrCancel(func() { endCalled = true })
	orCancel := f.OrCancel() // asked before the cancel: the error names the ticker
	ticker.Stop(TickerStopOpts{Canceled: true})
	if thenCalled || !endCalled {
		t.Errorf("then %v, whenCompleteOrCancel %v", thenCalled, endCalled)
	}
	select {
	case <-f.Done():
		t.Error("a canceled future is not done")
	default:
	}
	err := <-orCancel
	assertRoute(t, err, "Ticker.Stop->TickerFuture.cancel.TickerCanceled")
	appErr, _ := errors.FromError(err)
	if appErr.Message != "This ticker was canceled: Ticker(spinner)" {
		t.Errorf("message = %q", appErr.Message)
	}
	f.Then(func() { thenCalled = true })
	if thenCalled {
		t.Error("Then after a cancel is never called")
	}
	if v, ended := f.Completed(); !ended || v || !strings.HasSuffix(ignoreHash(f.String()), "(canceled)") {
		t.Errorf("Completed %v %v, String %q", v, ended, f.String())
	}
}

func TestTickerFutureCompleteConstructor(t *testing.T) {
	f := NewTickerFutureComplete()
	select {
	case <-f.Done():
	default:
		t.Error("TickerFuture.complete is done")
	}
	if err := <-f.OrCancel(); err != nil {
		t.Error(err)
	}
}

func TestTickerStopOfAnInactiveTickerDoesNothing(t *testing.T) {
	tt := newTester()
	ticker := NewTicker(tt.SchedulerBinding, func(time.Duration) {})
	ticker.Stop()
	ticker.Stop(TickerStopOpts{Canceled: true})
	if ticker.IsActive() {
		t.Error("active")
	}
}

func TestTickerAbsorbTicker(t *testing.T) {
	skipUnlessDebug(t)
	tt := newTester()
	var origTicks, newTicks []time.Duration
	orig := NewTicker(tt.SchedulerBinding, func(e time.Duration) { origTicks = append(origTicks, e) })
	f := orig.Start()
	tt.pump(frame) // the original ticks at 0
	next := NewTicker(tt.SchedulerBinding, func(e time.Duration) { newTicks = append(newTicks, e) })
	next.AbsorbTicker(orig)
	if orig.IsActive() || !next.IsActive() {
		t.Fatalf("orig active %v, new active %v", orig.IsActive(), next.IsActive())
	}
	tt.pump(frame)
	if len(origTicks) != 1 || len(newTicks) != 1 || newTicks[0] != frame {
		t.Errorf("orig ticks %v, new ticks %v: the new one continues from the start of the old", origTicks, newTicks)
	}
	select {
	case <-f.Done():
		t.Fatal("the future must not complete by absorbing")
	default:
	}
	next.Stop()
	select {
	case <-f.Done():
	default:
		t.Error("the new ticker completes the old future")
	}
	// Absorbing an unstarted ticker only disposes it.
	idle := NewTicker(tt.SchedulerBinding, func(time.Duration) {})
	NewTicker(tt.SchedulerBinding, func(time.Duration) {}).AbsorbTicker(idle)
	assertRoute(t, recoverError(t, func() { idle.Start() }), "TickerUsedAfterDispose")
}

func TestTickerProviderInterface(t *testing.T) {
	tt := newTester()
	var p TickerProvider = providerFunc(func(on TickerCallback) *Ticker { return NewTicker(tt.SchedulerBinding, on) })
	ticks := 0
	ticker := p.CreateTicker(func(time.Duration) { ticks++ })
	ticker.Start()
	tt.pump(frame)
	ticker.Dispose()
	if ticks != 1 {
		t.Errorf("ticks = %d", ticks)
	}
}

type providerFunc func(TickerCallback) *Ticker

func (f providerFunc) CreateTicker(on TickerCallback) *Ticker { return f(on) }

// ignoreHash replaces the short hash of a description.
func ignoreHash(s string) string {
	i := strings.Index(s, "#")
	if i < 0 || len(s) < i+6 {
		return s
	}
	return s[:i] + "#00000" + s[i+6:]
}

func TestTickerOnDisposeIsCalledOnce(t *testing.T) {
	tt := newTester()
	calls := 0
	ticker := NewTicker(tt.SchedulerBinding, func(time.Duration) {})
	ticker.OnDispose = func() { calls++ }
	ticker.Start()
	ticker.Dispose()
	if calls != 1 {
		t.Errorf("OnDispose was called %d times, want 1", calls)
	}
}

func TestTickerFuturePending(t *testing.T) {
	f, complete := NewTickerFuturePending()
	var log []string
	f.Then(func() { log = append(log, "then") })
	if v, ended := f.Completed(); ended || v {
		t.Fatalf("a pending future has ended: %v %v", v, ended)
	}
	complete()
	complete() // calling it again does nothing
	if got := strings.Join(log, ","); got != "then" {
		t.Errorf("callbacks = %q, want then once", got)
	}
	select {
	case <-f.Done():
	default:
		t.Error("Done is not closed after completing")
	}
	if v, ended := f.Completed(); !ended || !v {
		t.Errorf("Completed = %v %v", v, ended)
	}
}

func TestWaitTickerFutures(t *testing.T) {
	tests := []struct {
		name   string
		count  int
		finish []int // the futures to complete, in order
		want   []bool
	}{
		{"none", 0, nil, nil},
		{"one", 1, []int{0}, []bool{true}},
		{"two in order", 2, []int{0, 1}, []bool{false, true}},
		{"two in reverse", 2, []int{1, 0}, []bool{false, true}},
		{"three", 3, []int{2, 0, 1}, []bool{false, false, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			futures := make([]*TickerFuture, tt.count)
			completes := make([]func(), tt.count)
			for i := range futures {
				futures[i], completes[i] = NewTickerFuturePending()
			}
			all := WaitTickerFutures(futures...)
			if tt.count == 0 {
				if v, ended := all.Completed(); !ended || !v {
					t.Error("waiting for no futures should be complete at once")
				}
				return
			}
			for i, idx := range tt.finish {
				completes[idx]()
				if _, ended := all.Completed(); ended != tt.want[i] {
					t.Errorf("after completing %v, ended = %v, want %v", tt.finish[:i+1], ended, tt.want[i])
				}
			}
		})
	}
}

func TestWaitTickerFuturesNeverCompletesWithACanceledFuture(t *testing.T) {
	tt := newTester()
	ticker := NewTicker(tt.SchedulerBinding, func(time.Duration) {})
	canceled := ticker.Start()
	other, complete := NewTickerFuturePending()
	all := WaitTickerFutures(canceled, other)
	ticker.Stop(TickerStopOpts{Canceled: true})
	complete()
	if _, ended := all.Completed(); ended {
		t.Error("the wait completed although one of the futures was canceled")
	}
}
