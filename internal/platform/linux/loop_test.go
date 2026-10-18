//go:build linux

package linux

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

const testTimeout = 10 * time.Second

func newTestLoop(t *testing.T) *Loop {
	t.Helper()
	l, err := NewLoop()
	if err != nil {
		t.Fatalf("NewLoop: %v", err)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return l
}

// runInBackground starts l.Run and returns a channel with its result.
func runInBackground(ctx context.Context, l *Loop) <-chan error {
	done := make(chan error, 1)
	go func() { done <- l.Run(ctx) }()
	return done
}

func waitFor(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(testTimeout):
		t.Fatal("Run did not return in time")
		return nil
	}
}

func TestPostFromManyGoroutinesKeepsOrderPerGoroutine(t *testing.T) {
	const goroutines, perGoroutine = 100, 50
	l := newTestLoop(t)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	got := make([][]int, goroutines) // touched only by the loop's goroutine
	var remaining atomic.Int64
	remaining.Store(goroutines * perGoroutine)
	done := runInBackground(ctx, l)

	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range perGoroutine {
				l.Post(func() {
					got[g] = append(got[g], i)
					if remaining.Add(-1) == 0 {
						l.Stop()
					}
				})
			}
		}()
	}
	wg.Wait()
	if err := waitFor(t, done); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for g, seq := range got {
		if len(seq) != perGoroutine {
			t.Errorf("goroutine %d: %d functions ran, want %d", g, len(seq), perGoroutine)
			continue
		}
		for i, v := range seq {
			if v != i {
				t.Errorf("goroutine %d: position %d ran post %d", g, i, v)
				break
			}
		}
	}
}

func TestPostBeforeRunAndFromInsideAPostedFunction(t *testing.T) {
	l := newTestLoop(t)
	var order []string
	l.Post(func() {
		order = append(order, "first")
		l.Post(func() { // runs on a later pass, after the functions already queued
			order = append(order, "nested")
			l.Stop()
		})
	})
	l.Post(func() { order = append(order, "second") })

	if err := waitFor(t, runInBackground(context.Background(), l)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if want := []string{"first", "second", "nested"}; !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestRunStops(t *testing.T) {
	tests := []struct {
		name string
		stop func(cancel context.CancelFunc, l *Loop)
	}{
		{"context cancelled before Run", func(cancel context.CancelFunc, _ *Loop) { cancel() }},
		{"context cancelled while idle", func(cancel context.CancelFunc, _ *Loop) { time.AfterFunc(20*time.Millisecond, cancel) }},
		{"Stop while idle", func(_ context.CancelFunc, l *Loop) { time.AfterFunc(20*time.Millisecond, l.Stop) }},
		{"Stop before Run", func(_ context.CancelFunc, l *Loop) { l.Stop() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newTestLoop(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tt.stop(cancel, l)
			if err := waitFor(t, runInBackground(ctx, l)); err != nil {
				t.Errorf("Run returned %v, want nil", err)
			}
		})
	}
}

func TestTimers(t *testing.T) {
	l := newTestLoop(t)
	start := time.Now()
	type firing struct {
		name string
		at   time.Duration
	}
	var fired []firing
	record := func(name string) func() {
		return func() { fired = append(fired, firing{name, time.Since(start)}) }
	}

	if _, err := l.AfterFunc(60*time.Millisecond, record("c")); err != nil {
		t.Fatal(err)
	}
	if _, err := l.AfterFunc(20*time.Millisecond, record("a")); err != nil {
		t.Fatal(err)
	}
	cancelled, err := l.AfterFunc(30*time.Millisecond, record("cancelled"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.AfterFunc(40*time.Millisecond, func() {
		record("b")()
		// A timer set from a callback.
		if _, err := l.AfterFunc(10*time.Millisecond, func() { record("nested")() }); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := l.AfterFunc(-time.Second, record("overdue")); err != nil {
		t.Fatal(err)
	}
	if !cancelled.Stop() {
		t.Error("Stop on a waiting timer reported false")
	}
	if cancelled.Stop() {
		t.Error("a second Stop reported true")
	}

	// "c" is due last at 60 ms; stop a little after.
	if _, err := l.AfterFunc(120*time.Millisecond, l.Stop); err != nil {
		t.Fatal(err)
	}
	if err := waitFor(t, runInBackground(context.Background(), l)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var names []string
	for _, f := range fired {
		names = append(names, f.name)
	}
	if want := []string{"overdue", "a", "b", "nested", "c"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("timers fired %v, want %v", names, want)
	}
	deadlines := map[string]time.Duration{"overdue": 0, "a": 20 * time.Millisecond, "b": 40 * time.Millisecond, "c": 60 * time.Millisecond}
	for _, f := range fired {
		if d, ok := deadlines[f.name]; ok && f.at < d {
			t.Errorf("timer %s fired at %v, before its deadline %v", f.name, f.at, d)
		}
	}
	if fired[3].at < 50*time.Millisecond {
		t.Errorf("the nested timer fired at %v, want at least 10 ms after the timer that set it (40 ms)", fired[3].at)
	}
}

// pipeSource is a Source on the read end of a pipe.
type pipeSource struct {
	fd         int
	before     atomic.Int64
	after      []Events
	wantsWrite bool
	failAfter  error
	onAfter    func(Events)
}

func (s *pipeSource) FD() int          { return s.fd }
func (s *pipeSource) WantsWrite() bool { return s.wantsWrite }
func (s *pipeSource) BeforeWait() error {
	s.before.Add(1)
	return nil
}
func (s *pipeSource) AfterWait(ready Events) error {
	s.after = append(s.after, ready)
	if s.onAfter != nil {
		s.onAfter(ready)
	}
	return s.failAfter
}

func newPipe(t *testing.T) (r, w int) {
	t.Helper()
	var fds [2]int
	if err := syscall.Pipe2(fds[:], syscall.O_CLOEXEC|syscall.O_NONBLOCK); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, fd := range fds {
			if err := syscall.Close(fd); err != nil && err != syscall.EBADF {
				t.Errorf("close: %v", err)
			}
		}
	})
	return fds[0], fds[1]
}

func TestSourceReadable(t *testing.T) {
	l := newTestLoop(t)
	r, w := newPipe(t)
	src := &pipeSource{fd: r}
	src.onAfter = func(ev Events) {
		if ev&Readable != 0 {
			var buf [8]byte
			_, _ = syscall.Read(r, buf[:])
			l.Stop()
		}
	}
	if err := l.AddSource(src); err != nil {
		t.Fatal(err)
	}
	done := runInBackground(context.Background(), l)
	time.Sleep(30 * time.Millisecond)
	if _, err := syscall.Write(w, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := waitFor(t, done); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := src.after[len(src.after)-1]; got&Readable == 0 {
		t.Errorf("last AfterWait events = %b, want Readable", got)
	}
}

func TestIdleLoopSleeps(t *testing.T) {
	l := newTestLoop(t)
	r, _ := newPipe(t)
	src := &pipeSource{fd: r}
	if err := l.AddSource(src); err != nil {
		t.Fatal(err)
	}
	done := runInBackground(context.Background(), l)
	time.Sleep(150 * time.Millisecond)
	l.Stop()
	if err := waitFor(t, done); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// One pass to start waiting, one when Stop woke the loop. A spinning loop
	// would have made thousands.
	if got := src.before.Load(); got > 3 {
		t.Errorf("BeforeWait ran %d times while idle, want at most 3", got)
	}
}

func TestSourceWriteInterest(t *testing.T) {
	l := newTestLoop(t)
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC|syscall.SOCK_NONBLOCK, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Close(fds[0]); _ = syscall.Close(fds[1]) })

	src := &pipeSource{fd: fds[0]}
	var sawWritable bool
	src.onAfter = func(ev Events) {
		if ev&Writable != 0 {
			sawWritable = true
			src.wantsWrite = false // output is flushed
			l.Stop()
		}
	}
	if err := l.AddSource(src); err != nil {
		t.Fatal(err)
	}
	// Without write interest, an empty socket buffer reports nothing.
	src.wantsWrite = false
	time.AfterFunc(50*time.Millisecond, func() {
		l.Post(func() { src.wantsWrite = true })
		l.Post(func() {}) // one more pass so BeforeWait sees the new interest
	})
	if err := waitFor(t, runInBackground(context.Background(), l)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !sawWritable {
		t.Error("the source never got Writable after it asked for write interest")
	}
}

func TestSourceHangupIsReadable(t *testing.T) {
	l := newTestLoop(t)
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC|syscall.SOCK_NONBLOCK, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Close(fds[0]) })
	src := &pipeSource{fd: fds[0]}
	src.onAfter = func(ev Events) {
		if ev&Hangup != 0 {
			if ev&Readable == 0 {
				t.Errorf("a hangup must also be Readable so that reading shows the error: %b", ev)
			}
			l.Stop()
		}
	}
	if err := l.AddSource(src); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Close(fds[1]); err != nil {
		t.Fatal(err)
	}
	if err := waitFor(t, runInBackground(context.Background(), l)); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

type failingSource struct {
	pipeSource
	err error
}

func (s *failingSource) BeforeWait() error { return s.err }

func TestSourceErrorEndsRunUnchanged(t *testing.T) {
	l := newTestLoop(t)
	r, _ := newPipe(t)
	sentinel := errors.New("SourceFailed", "the source failed")
	if err := l.AddSource(&failingSource{pipeSource: pipeSource{fd: r}, err: sentinel}); err != nil {
		t.Fatal(err)
	}
	err := waitFor(t, runInBackground(context.Background(), l))
	if err != error(sentinel) {
		t.Errorf("Run returned %v, want the source's own error", err)
	}

	l2 := newTestLoop(t)
	r2, w2 := newPipe(t)
	src := &pipeSource{fd: r2, failAfter: sentinel}
	if err := l2.AddSource(src); err != nil {
		t.Fatal(err)
	}
	if _, err := syscall.Write(w2, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := waitFor(t, runInBackground(context.Background(), l2)); err != error(sentinel) {
		t.Errorf("Run returned %v after AfterWait failed, want the source's own error", err)
	}
}

func TestRemoveSource(t *testing.T) {
	l := newTestLoop(t)
	r, w := newPipe(t)
	src := &pipeSource{fd: r}
	if err := l.AddSource(src); err != nil {
		t.Fatal(err)
	}
	if err := l.RemoveSource(src); err != nil {
		t.Fatal(err)
	}
	if _, err := syscall.Write(w, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if _, err := l.AfterFunc(60*time.Millisecond, l.Stop); err != nil {
		t.Fatal(err)
	}
	if err := waitFor(t, runInBackground(context.Background(), l)); err != nil {
		t.Fatal(err)
	}
	if len(src.after) != 0 || src.before.Load() != 0 {
		t.Errorf("a removed source was still called: %d before, %d after", src.before.Load(), len(src.after))
	}
}

func TestSyscallFailures(t *testing.T) {
	l := newTestLoop(t)
	tests := []struct {
		name  string
		do    func() error
		route string
	}{
		{"AddSource with a bad descriptor", func() error { return l.AddSource(&pipeSource{fd: -1}) }, "Loop.AddSource.LoopSyscallFailed"},
		{"RemoveSource of an unknown descriptor", func() error { return l.RemoveSource(&pipeSource{fd: 987654}) }, "Loop.RemoveSource.LoopSyscallFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.do()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != "LoopSyscallFailed" || !appErr.HasRoute(tt.route) {
				t.Errorf("Tag = %q, route = %s, want %s", appErr.Tag, appErr.Route(), tt.route)
			}
		})
	}
}
