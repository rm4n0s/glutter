// Package fakeclock is a clock that a test moves by hand, so that timers and
// frame times in tests need no sleeping.
package fakeclock

import (
	"sort"
	"sync"
	"time"
)

// Clock tells the time and runs functions later. The real clock and the fake
// one both implement it, and code that needs time takes a Clock.
//
// Flutter: no counterpart (flutter_test's FakeAsync)
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// AfterFunc runs f in its own goroutine (the real clock) or inside Advance
	// (the fake one) after d. The returned Timer cancels it.
	AfterFunc(d time.Duration, f func()) Timer
}

// Timer is a function waiting to run.
//
// Flutter: no counterpart
type Timer interface {
	// Stop prevents the function from running. It reports whether it did: false
	// when the function already ran or was already stopped.
	Stop() bool
}

// Real is the clock of the machine.
//
// Flutter: no counterpart
type Real struct{}

var _ Clock = Real{}

// Now returns time.Now().
func (Real) Now() time.Time { return time.Now() }

// AfterFunc is time.AfterFunc.
func (Real) AfterFunc(d time.Duration, f func()) Timer { return time.AfterFunc(d, f) }

// Fake is a Clock that only moves when Advance or Set is called. It is safe
// for concurrent use; the functions run on the goroutine that advances it.
//
// Flutter: no counterpart (flutter_test's FakeAsync)
type Fake struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
	seq    uint64
}

var _ Clock = (*Fake)(nil)

// New returns a fake clock that reads start. A zero start is the Unix epoch,
// so that times print the same on every run.
func New(start time.Time) *Fake {
	if start.IsZero() {
		start = time.Unix(0, 0).UTC()
	}
	return &Fake{now: start}
}

type fakeTimer struct {
	clock *Fake
	when  time.Time
	seq   uint64
	f     func()
	state timerState
}

type timerState int

const (
	timerPending timerState = iota
	timerFired
	timerStopped
)

// Stop cancels the timer.
func (t *fakeTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	if t.state != timerPending {
		return false
	}
	t.state = timerStopped
	t.clock.removeLocked(t)
	return true
}

// Now returns the time of the clock.
func (c *Fake) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Since returns the time elapsed from t, by this clock.
func (c *Fake) Since(t time.Time) time.Duration { return c.Now().Sub(t) }

// AfterFunc schedules f to run when the clock has advanced by d. A d of zero
// or less runs f at the next Advance (of any amount, including zero).
func (c *Fake) AfterFunc(d time.Duration, f func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	t := &fakeTimer{clock: c, when: c.now.Add(d), seq: c.seq, f: f}
	c.timers = append(c.timers, t)
	return t
}

// Pending returns how many timers are waiting.
func (c *Fake) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.timers)
}

func (c *Fake) removeLocked(t *fakeTimer) {
	for i, x := range c.timers {
		if x == t {
			c.timers = append(c.timers[:i], c.timers[i+1:]...)
			return
		}
	}
}

// Advance moves the clock forward by d and runs the timers that come due, in
// order of their time (and of their creation for equal times). While a
// function runs the clock reads the time of its timer, so a function that
// reads the clock or sets another timer sees a consistent time; a timer it
// sets that falls within d runs in the same call. A negative d panics.
func (c *Fake) Advance(d time.Duration) {
	if d < 0 {
		panic("fakeclock: Advance with a negative duration")
	}
	c.mu.Lock()
	target := c.now.Add(d)
	c.mu.Unlock()
	for {
		c.mu.Lock()
		sort.SliceStable(c.timers, func(i, j int) bool {
			a, b := c.timers[i], c.timers[j]
			if !a.when.Equal(b.when) {
				return a.when.Before(b.when)
			}
			return a.seq < b.seq
		})
		if len(c.timers) == 0 || c.timers[0].when.After(target) {
			c.now = target
			c.mu.Unlock()
			return
		}
		t := c.timers[0]
		c.timers = c.timers[1:]
		t.state = timerFired
		if t.when.After(c.now) {
			c.now = t.when
		}
		c.mu.Unlock()
		t.f()
	}
}

// Set moves the clock to t, running the timers that come due on the way. A t
// before the current time panics.
func (c *Fake) Set(t time.Time) { c.Advance(t.Sub(c.Now())) }
