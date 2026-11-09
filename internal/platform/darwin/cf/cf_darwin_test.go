//go:build darwin

package cf

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

func TestStrings(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"ascii", "hello"},
		{"empty", ""},
		{"the basic plane", "héllo ☃ 日本語"},
		{"a surrogate pair in UTF-16", "a😀b 𝄞"},
		{"an embedded NUL", "ab\x00cd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewString(tt.in)
			if s == 0 {
				t.Fatal("NewString gave 0")
			}
			defer Release(s)
			if got := String(s); got != tt.in {
				t.Errorf("String(NewString(%q)) = %q", tt.in, got)
			}
		})
	}
	if s := NewString("a\xffb"); s != 0 {
		Release(s)
		t.Error("NewString of invalid UTF-8 did not give 0")
	}
	if got := String(0); got != "" {
		t.Errorf("String(0) = %q", got)
	}
	Release(0) // does nothing
	if Retain(0) != 0 {
		t.Error("Retain(0) != 0")
	}
}

func TestDictionary(t *testing.T) {
	k1, k2, missing := NewString("one"), NewString("two"), NewString("three")
	v1, v2 := NewString("1"), NewString("2")
	defer func() {
		for _, r := range []Ref{k1, k2, missing, v1, v2} {
			Release(r)
		}
	}()
	d := NewDictionary([]Ref{k1, k2}, []Ref{v1, v2})
	if d == 0 {
		t.Fatal("NewDictionary gave 0")
	}
	defer Release(d)
	if got := DictionaryCount(d); got != 2 {
		t.Errorf("DictionaryCount = %d, want 2", got)
	}
	if got := String(DictionaryGet(d, k2)); got != "2" {
		t.Errorf("value of two = %q", got)
	}
	// A different CFString with the same characters finds the entry: keys are compared by value.
	same := NewString("one")
	defer Release(same)
	if got := String(DictionaryGet(d, same)); got != "1" {
		t.Errorf("value of an equal key = %q", got)
	}
	if got := DictionaryGet(d, missing); got != 0 {
		t.Errorf("value of a missing key = %d, want 0", got)
	}
	empty := NewDictionary(nil, nil)
	defer Release(empty)
	if DictionaryCount(empty) != 0 {
		t.Error("an empty dictionary has entries")
	}
	defer func() {
		if recover() == nil {
			t.Error("NewDictionary with unequal slices did not panic")
		}
	}()
	NewDictionary([]Ref{k1}, nil)
}

func TestAbsoluteTime(t *testing.T) {
	ref := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		t    time.Time
		want float64
	}{
		{"the reference date", ref, 0},
		{"one second after", ref.Add(time.Second), 1},
		{"half a second before", ref.Add(-500 * time.Millisecond), -0.5},
		{"the Unix epoch", time.Unix(0, 0), -978307200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AbsoluteTime(tt.t); got != tt.want {
				t.Errorf("AbsoluteTime = %v, want %v", got, tt.want)
			}
		})
	}
	if d := Now() - AbsoluteTime(time.Now()); d > 1 || d < -1 {
		t.Errorf("Now() and AbsoluteTime(time.Now()) differ by %v seconds", d)
	}
}

// onThread runs fn on a goroutine locked to its OS thread, which is what a run loop needs.
func onThread(t *testing.T, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		fn()
	}()
	<-done
}

func TestSourceRunsOnTheThreadOfTheRunLoop(t *testing.T) {
	onThread(t, func() {
		var fired atomic.Int32
		src := NewSource(func() { fired.Add(1) })
		defer src.Close()
		rl := CurrentRunLoop()
		src.AddTo(rl, DefaultMode())
		src.Signal()
		if got := RunInMode(DefaultMode(), time.Second, true); got != RunHandledSource {
			t.Errorf("RunInMode = %d, want RunHandledSource (%d)", got, RunHandledSource)
		}
		if got := fired.Load(); got != 1 {
			t.Errorf("the source ran %d times, want 1", got)
		}
		// A source that is not signaled does not run.
		if got := RunInMode(DefaultMode(), 20*time.Millisecond, true); got != RunTimedOut {
			t.Errorf("RunInMode with nothing signaled = %d, want RunTimedOut (%d)", got, RunTimedOut)
		}
		if got := fired.Load(); got != 1 {
			t.Errorf("the source ran %d times without a signal, want 1", got)
		}
	})
}

func TestSourceSignaledFromAnotherGoroutineWakesTheLoop(t *testing.T) {
	onThread(t, func() {
		var fired atomic.Int32
		src := NewSource(func() { fired.Add(1) })
		defer src.Close()
		rl := CurrentRunLoop()
		src.AddTo(rl, CommonModes())
		go func() {
			time.Sleep(30 * time.Millisecond)
			src.Signal()
			WakeUp(rl)
		}()
		start := time.Now()
		got := RunInMode(DefaultMode(), 5*time.Second, true)
		if got != RunHandledSource {
			t.Errorf("RunInMode = %d, want RunHandledSource", got)
		}
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("the loop slept %v after the signal, want it woken at once", d)
		}
		if fired.Load() != 1 {
			t.Errorf("the source ran %d times, want 1", fired.Load())
		}
	})
}

func TestTimerFiresOnceAtItsTime(t *testing.T) {
	onThread(t, func() {
		var fired atomic.Int32
		var firedAt atomic.Int64 // UnixNano of the last fire
		timer := NewTimer(func() {
			fired.Add(1)
			firedAt.Store(time.Now().UnixNano())
		})
		defer timer.Close()
		timer.AddTo(CurrentRunLoop(), DefaultMode())

		// Not armed: nothing fires.
		if got := RunInMode(DefaultMode(), 50*time.Millisecond, false); got != RunTimedOut {
			t.Errorf("RunInMode with an unarmed timer = %d, want RunTimedOut", got)
		}
		if fired.Load() != 0 {
			t.Fatal("an unarmed timer fired")
		}

		// RunInMode does not return when a timer fires (only a source ends it early), so it runs for a
		// fixed time and the time of the fire is recorded by the callback.
		start := time.Now()
		timer.SetNextFire(start.Add(40 * time.Millisecond))
		RunInMode(DefaultMode(), 200*time.Millisecond, false)
		if fired.Load() != 1 {
			t.Fatalf("the timer fired %d times, want 1", fired.Load())
		}
		if d := time.Duration(firedAt.Load() - start.UnixNano()); d < 30*time.Millisecond || d > 150*time.Millisecond {
			t.Errorf("the timer fired after %v, want about 40ms", d)
		}
		// It fires once: running longer does not fire it again.
		RunInMode(DefaultMode(), 80*time.Millisecond, false)
		if fired.Load() != 1 {
			t.Errorf("the timer fired %d times, want it once", fired.Load())
		}

		// It can be armed again after it fired. A time in the past fires at once.
		timer.SetNextFire(time.Now().Add(-time.Second))
		RunInMode(DefaultMode(), 100*time.Millisecond, false)
		if fired.Load() != 2 {
			t.Errorf("a timer set in the past fired %d times in all, want 2", fired.Load())
		}
		// Disarm stops a pending fire.
		timer.SetNextFire(time.Now().Add(30 * time.Millisecond))
		timer.Disarm()
		RunInMode(DefaultMode(), 100*time.Millisecond, false)
		if fired.Load() != 2 {
			t.Errorf("a disarmed timer fired: %d in all, want 2", fired.Load())
		}
		// And it can be armed once more after Disarm.
		timer.SetNextFire(time.Now().Add(20 * time.Millisecond))
		RunInMode(DefaultMode(), 150*time.Millisecond, false)
		if fired.Load() != 3 {
			t.Errorf("a timer armed after Disarm fired %d times in all, want 3", fired.Load())
		}
	})
}

func TestCloseIsSafeTwiceAndStopsTheCallbacks(t *testing.T) {
	onThread(t, func() {
		var fired atomic.Int32
		src := NewSource(func() { fired.Add(1) })
		timer := NewTimer(func() { fired.Add(1) })
		rl := CurrentRunLoop()
		src.AddTo(rl, DefaultMode())
		timer.AddTo(rl, DefaultMode())
		src.Close()
		timer.Close()
		src.Close()
		timer.Close()
		// Calls on a closed source or timer do nothing, and do not crash.
		src.Signal()
		src.AddTo(rl, DefaultMode())
		timer.SetNextFire(time.Now())
		timer.Disarm()
		timer.AddTo(rl, DefaultMode())
		RunInMode(DefaultMode(), 30*time.Millisecond, false)
		if fired.Load() != 0 {
			t.Errorf("a closed source or timer ran %d times", fired.Load())
		}
	})
}

func TestManySourcesCostTwoCallbacksInAll(t *testing.T) {
	// The perform and timer callbacks are made once and shared by every source and timer (SPEC §8.1 rule 2).
	NewSource(func() {}).Close() // makes the perform callback if no test did
	NewTimer(func() {}).Close()
	before := dl.CallbackCount()
	for range 100 {
		NewSource(func() {}).Close()
		NewTimer(func() {}).Close()
	}
	if got := dl.CallbackCount() - before; got != 0 {
		t.Errorf("200 sources and timers made %d callbacks, want 0", got)
	}
}

func TestCallbackTableHoldsOnlyWhatIsAlive(t *testing.T) {
	count := func() int {
		callbacks.mu.RLock()
		defer callbacks.mu.RUnlock()
		return len(callbacks.fns)
	}
	before := count()
	src, timer := NewSource(func() {}), NewTimer(func() {})
	if got := count() - before; got != 2 {
		t.Errorf("the table grew by %d, want 2", got)
	}
	src.Close()
	timer.Close()
	if got := count() - before; got != 0 {
		t.Errorf("the table is %d above where it began after Close, want 0", got)
	}
}
