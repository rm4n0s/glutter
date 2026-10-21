package web

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestLifecycleOf(t *testing.T) {
	tests := []struct {
		name             string
		visible, focused bool
		want             ui.AppLifecycleState
	}{
		{"visible and focused", true, true, ui.AppLifecycleStateResumed},
		{"visible without focus", true, false, ui.AppLifecycleStateInactive},
		{"hidden", false, false, ui.AppLifecycleStateHidden},
		{"hidden keeps the focus", false, true, ui.AppLifecycleStateHidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := LifecycleOf(tc.visible, tc.focused); got != tc.want {
				t.Errorf("LifecycleOf(%v, %v) = %v, want %v", tc.visible, tc.focused, got, tc.want)
			}
		})
	}
}
