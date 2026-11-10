package ios

import (
	"testing"
	"time"
)

func TestKeyboardOverlap(t *testing.T) {
	for _, tc := range []struct {
		name                                       string
		kbTop, kbBottom, viewTop, viewBottom, want float64
	}{
		{"a keyboard at the bottom", 574, 874, 0, 874, 300},
		{"hidden below the view", 874, 1174, 0, 874, 0},
		{"no height", 600, 600, 0, 874, 0},
		{"taller than the view", -100, 874, 0, 874, 874},
		{"a floating keyboard does not reach the bottom", 400, 700, 0, 874, 0},
		{"a view that starts lower", 574, 874, 100, 874, 300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := keyboardOverlap(tc.kbTop, tc.kbBottom, tc.viewTop, tc.viewBottom); got != tc.want {
				t.Errorf("overlap = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInsetAnimation(t *testing.T) {
	var a insetAnimation
	if v, running := a.value(0); v != 0 || running {
		t.Errorf("an animation that never began = %v %v", v, running)
	}
	a.begin(0, 300, time.Second, 250*time.Millisecond)
	if v, running := a.value(time.Second); v != 0 || !running {
		t.Errorf("at the start = %v %v", v, running)
	}
	mid, running := a.value(time.Second + 125*time.Millisecond)
	if !running || mid <= 150 || mid >= 300 {
		t.Errorf("halfway = %v %v, want more than half of it (ease out)", mid, running)
	}
	if v, running := a.value(time.Second + 250*time.Millisecond); v != 300 || running {
		t.Errorf("at the end = %v %v", v, running)
	}
	if v, running := a.value(time.Hour); v != 300 || running {
		t.Errorf("after the end = %v %v", v, running)
	}
	// Going back from where it is: a keyboard that is hidden while it comes.
	a.begin(120, 0, time.Minute, 250*time.Millisecond)
	if v, _ := a.value(time.Minute); v != 120 {
		t.Errorf("a new animation starts from %v, want 120", v)
	}
	// No duration, or no change: the target at once.
	a.begin(10, 50, 0, 0)
	if v, running := a.value(0); v != 50 || running {
		t.Errorf("no duration = %v %v", v, running)
	}
	a.begin(50, 50, 0, time.Second)
	if _, running := a.value(0); running {
		t.Error("an animation with no change runs")
	}
}
