package ios

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestLifecycleOf(t *testing.T) {
	for _, tc := range []struct {
		name   string
		scenes []activation
		ended  bool
		want   ui.AppLifecycleState
	}{
		{"no scene", nil, false, ui.AppLifecycleStateDetached},
		{"unattached", []activation{activationUnattached}, false, ui.AppLifecycleStateDetached},
		{"active", []activation{activationForegroundActive}, false, ui.AppLifecycleStateResumed},
		{"inactive", []activation{activationForegroundInactive}, false, ui.AppLifecycleStateInactive},
		{"background", []activation{activationBackground}, false, ui.AppLifecycleStatePaused},
		{"one active among others", []activation{activationBackground, activationForegroundActive}, false, ui.AppLifecycleStateResumed},
		{"inactive beats background", []activation{activationBackground, activationForegroundInactive}, false, ui.AppLifecycleStateInactive},
		{"ended wins", []activation{activationForegroundActive}, true, ui.AppLifecycleStateDetached},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := lifecycleOf(tc.scenes, tc.ended); got != tc.want {
				t.Errorf("lifecycleOf(%v, %v) = %v, want %v", tc.scenes, tc.ended, got, tc.want)
			}
		})
	}
}

func TestLifecycleTracker(t *testing.T) {
	var tr lifecycleTracker
	steps := []struct {
		scenes      []activation
		ended       bool
		want        ui.AppLifecycleState
		wantChanged bool
	}{
		{nil, false, ui.AppLifecycleStateDetached, false}, // no scene yet: nothing to report
		{[]activation{activationForegroundInactive}, false, ui.AppLifecycleStateInactive, true},
		{[]activation{activationForegroundActive}, false, ui.AppLifecycleStateResumed, true},
		{[]activation{activationForegroundActive}, false, ui.AppLifecycleStateResumed, false}, // the same: not again
		{[]activation{activationBackground}, false, ui.AppLifecycleStatePaused, true},
		{[]activation{activationBackground}, true, ui.AppLifecycleStateDetached, true},
	}
	for i, s := range steps {
		got, changed := tr.update(s.scenes, s.ended)
		if got != s.want || changed != s.wantChanged {
			t.Errorf("step %d: update(%v, %v) = %v, %v; want %v, %v", i, s.scenes, s.ended, got, changed, s.want, s.wantChanged)
		}
	}
}
