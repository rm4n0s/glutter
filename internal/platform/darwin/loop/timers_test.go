package loop

import (
	"slices"
	"testing"
	"time"
)

func TestTimerHeapRunsDueTimersInOrder(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ms := func(n int) time.Time { return base.Add(time.Duration(n) * time.Millisecond) }
	tests := []struct {
		name string
		arm  []int // when each timer is due, in ms; the timer is named by its index
		now  int
		want []int
	}{
		{"none armed", nil, 100, nil},
		{"none due", []int{50, 60}, 10, nil},
		{"the earliest first", []int{30, 10, 20}, 100, []int{1, 2, 0}},
		{"one instant runs in the order of arming", []int{10, 10, 10}, 10, []int{0, 1, 2}},
		{"only the due ones", []int{10, 90, 20}, 25, []int{0, 2}},
		{"exactly at the deadline", []int{40}, 40, []int{0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h Timers
			var got []int
			for i, at := range tt.arm {
				h.Add(ms(at), func() { got = append(got, i) })
			}
			n := h.RunDue(ms(tt.now))
			if !slices.Equal(got, tt.want) || n != len(tt.want) {
				t.Errorf("ran %v (%d), want %v", got, n, tt.want)
			}
		})
	}
}

func TestTimerHeapCancelAndDeadline(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var h Timers
	if _, ok := h.Deadline(); ok {
		t.Fatal("an empty heap has a deadline")
	}
	if _, ok := h.WaitFor(base); ok {
		t.Fatal("an empty heap says how long to wait: the loop must wait for ever")
	}
	ran := map[string]bool{}
	a := h.Add(base.Add(10*time.Millisecond), func() { ran["a"] = true })
	b := h.Add(base.Add(30*time.Millisecond), func() { ran["b"] = true })
	if at, ok := h.Deadline(); !ok || !at.Equal(base.Add(10*time.Millisecond)) {
		t.Errorf("deadline = %v, %v", at, ok)
	}
	if d, ok := h.WaitFor(base); !ok || d != 10*time.Millisecond {
		t.Errorf("waitFor = %v, %v", d, ok)
	}
	if d, _ := h.WaitFor(base.Add(time.Second)); d != 0 {
		t.Errorf("waitFor of a due timer = %v, want 0", d)
	}
	if !h.Cancel(a) {
		t.Error("cancel of an armed timer is false")
	}
	if h.Cancel(a) {
		t.Error("cancel of a cancelled timer is true")
	}
	if at, _ := h.Deadline(); !at.Equal(base.Add(30 * time.Millisecond)) {
		t.Errorf("deadline after the cancel = %v", at)
	}
	h.RunDue(base.Add(time.Second))
	if ran["a"] || !ran["b"] {
		t.Errorf("ran = %v, want only b", ran)
	}
	if h.Cancel(b) {
		t.Error("cancel of a timer that ran is true")
	}
}

func TestTimerHeapFunctionsMayArmTimers(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var h Timers
	var got []string
	h.Add(base, func() {
		got = append(got, "first")
		h.Add(base, func() { got = append(got, "second, due already") })
		h.Add(base.Add(time.Hour), func() { got = append(got, "later") })
	})
	h.RunDue(base)
	if !slices.Equal(got, []string{"first", "second, due already"}) {
		t.Errorf("ran %v", got)
	}
	if _, ok := h.Deadline(); !ok {
		t.Error("the timer for later is gone")
	}
}
