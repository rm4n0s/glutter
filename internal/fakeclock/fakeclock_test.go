package fakeclock

import (
	"reflect"
	"testing"
	"time"
)

func TestFakeNowAndAdvance(t *testing.T) {
	c := New(time.Time{})
	if !c.Now().Equal(time.Unix(0, 0)) {
		t.Errorf("start = %v", c.Now())
	}
	start := c.Now()
	tests := []struct {
		advance time.Duration
		since   time.Duration
	}{
		{0, 0},
		{16 * time.Millisecond, 16 * time.Millisecond},
		{time.Second, time.Second + 16*time.Millisecond},
	}
	for _, tt := range tests {
		c.Advance(tt.advance)
		if got := c.Since(start); got != tt.since {
			t.Errorf("after Advance(%v): Since = %v, want %v", tt.advance, got, tt.since)
		}
	}
	custom := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	if got := New(custom).Now(); !got.Equal(custom) {
		t.Errorf("custom start = %v", got)
	}
}

func TestFakeTimersRunInOrder(t *testing.T) {
	c := New(time.Time{})
	var log []string
	c.AfterFunc(30*time.Millisecond, func() { log = append(log, "c") })
	c.AfterFunc(10*time.Millisecond, func() { log = append(log, "a") })
	c.AfterFunc(10*time.Millisecond, func() { log = append(log, "a2") })
	c.AfterFunc(20*time.Millisecond, func() { log = append(log, "b") })
	c.AfterFunc(-time.Second, func() { log = append(log, "now") })

	c.Advance(0)
	if want := []string{"now"}; !reflect.DeepEqual(log, want) {
		t.Errorf("after Advance(0): %v, want %v", log, want)
	}
	c.Advance(25 * time.Millisecond)
	if want := []string{"now", "a", "a2", "b"}; !reflect.DeepEqual(log, want) {
		t.Errorf("after 25ms: %v, want %v", log, want)
	}
	if c.Pending() != 1 {
		t.Errorf("pending = %d, want 1", c.Pending())
	}
	c.Advance(5 * time.Millisecond)
	if want := []string{"now", "a", "a2", "b", "c"}; !reflect.DeepEqual(log, want) {
		t.Errorf("after 30ms: %v, want %v", log, want)
	}
}

func TestFakeTimerSeesItsOwnTime(t *testing.T) {
	c := New(time.Time{})
	start := c.Now()
	var seen []time.Duration
	c.AfterFunc(10*time.Millisecond, func() {
		seen = append(seen, c.Since(start))
		// A timer set by a timer runs in the same Advance when it is due.
		c.AfterFunc(5*time.Millisecond, func() { seen = append(seen, c.Since(start)) })
	})
	c.Advance(100 * time.Millisecond)
	want := []time.Duration{10 * time.Millisecond, 15 * time.Millisecond}
	if !reflect.DeepEqual(seen, want) {
		t.Errorf("seen = %v, want %v", seen, want)
	}
	if got := c.Since(start); got != 100*time.Millisecond {
		t.Errorf("clock = %v after the Advance, want 100ms", got)
	}
}

func TestFakeTimerStop(t *testing.T) {
	c := New(time.Time{})
	ran := 0
	tm := c.AfterFunc(time.Second, func() { ran++ })
	if !tm.Stop() {
		t.Error("Stop of a pending timer must report true")
	}
	if tm.Stop() {
		t.Error("Stop twice must report false")
	}
	fired := c.AfterFunc(time.Millisecond, func() { ran++ })
	c.Advance(time.Minute)
	if ran != 1 || fired.Stop() {
		t.Errorf("ran %d: the stopped timer must not run, and a fired one cannot be stopped", ran)
	}
}

func TestFakeSetAndNegativeAdvance(t *testing.T) {
	c := New(time.Time{})
	target := time.Unix(100, 0)
	ran := false
	c.AfterFunc(50*time.Second, func() { ran = true })
	c.Set(target)
	if !c.Now().Equal(target) || !ran {
		t.Errorf("now %v, ran %v", c.Now(), ran)
	}
	defer func() {
		if recover() == nil {
			t.Error("a negative Advance must panic")
		}
	}()
	c.Advance(-time.Nanosecond)
}

func TestRealClock(t *testing.T) {
	var c Clock = Real{}
	before := time.Now()
	if c.Now().Before(before) {
		t.Error("Real.Now went back")
	}
	done := make(chan struct{})
	tm := c.AfterFunc(time.Millisecond, func() { close(done) })
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the timer did not run")
	}
	if tm.Stop() {
		t.Error("a timer that ran cannot be stopped")
	}
}
