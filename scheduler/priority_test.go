package scheduler

import "testing"

// Ported from packages/flutter/test/scheduler/priority_test.dart.

func TestPriorityOperators(t *testing.T) {
	tests := []struct {
		name string
		got  Priority
		want int
	}{
		{"idle plus more than the max offset", PriorityIdle.Add(PriorityMaxOffset + 100), PriorityIdle.Value() + PriorityMaxOffset},
		{"animation minus more than the max offset", PriorityAnimation.Sub(PriorityMaxOffset + 100), PriorityAnimation.Value() - PriorityMaxOffset},
		{"touch plus a small offset", PriorityTouch.Add(5), 200005},
		{"touch minus a small offset", PriorityTouch.Sub(5), 199995},
		{"animation plus a negative offset", PriorityAnimation.Add(-20000), PriorityAnimation.Value() - PriorityMaxOffset},
		{"the zero value is idle", Priority(0), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got.Value() != tt.want {
				t.Errorf("value = %d, want %d", tt.got.Value(), tt.want)
			}
		})
	}
	if PriorityIdle >= PriorityAnimation || PriorityAnimation >= PriorityTouch {
		t.Error("idle < animation < touch")
	}
}
