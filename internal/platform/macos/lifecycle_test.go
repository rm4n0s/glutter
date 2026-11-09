package macos

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestLifecycleOf(t *testing.T) {
	vis := lifecycleWindow{Visible: true}
	key := lifecycleWindow{Visible: true, Key: true}
	min := lifecycleWindow{Visible: true, Minimized: true, Key: true}
	hid := lifecycleWindow{Key: true}
	tests := []struct {
		name    string
		windows []lifecycleWindow
		hidden  bool
		ended   bool
		want    ui.AppLifecycleState
	}{
		{"no window", nil, false, false, ui.AppLifecycleStateHidden},
		{"key window", []lifecycleWindow{key}, false, false, ui.AppLifecycleStateResumed},
		{"no key window", []lifecycleWindow{vis, vis}, false, false, ui.AppLifecycleStateInactive},
		{"one key of two", []lifecycleWindow{vis, key}, false, false, ui.AppLifecycleStateResumed},
		{"minimized", []lifecycleWindow{min}, false, false, ui.AppLifecycleStateHidden},
		{"hidden window", []lifecycleWindow{hid}, false, false, ui.AppLifecycleStateHidden},
		{"minimized and visible", []lifecycleWindow{min, vis}, false, false, ui.AppLifecycleStateInactive},
		{"app hidden", []lifecycleWindow{key}, true, false, ui.AppLifecycleStateHidden},
		{"ended", []lifecycleWindow{key}, false, true, ui.AppLifecycleStateDetached},
		{"ended and hidden", nil, true, true, ui.AppLifecycleStateDetached},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lifecycleOf(tt.windows, tt.hidden, tt.ended); got != tt.want {
				t.Errorf("lifecycleOf = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLifecycleTrackerReportsChangesOnly(t *testing.T) {
	var tr lifecycleTracker
	step := func(windows []lifecycleWindow, hidden, ended bool) (ui.AppLifecycleState, bool) {
		return tr.update(windows, hidden, ended)
	}
	key := []lifecycleWindow{{Visible: true, Key: true}}
	inactive := []lifecycleWindow{{Visible: true}}
	if _, changed := step(nil, false, false); changed {
		t.Error("the Hidden of an app with no window is reported")
	}
	if s, changed := step(key, false, false); !changed || s != ui.AppLifecycleStateResumed {
		t.Errorf("first window: %v %v", s, changed)
	}
	if _, changed := step(key, false, false); changed {
		t.Error("an unchanged state is reported again")
	}
	if s, changed := step(inactive, false, false); !changed || s != ui.AppLifecycleStateInactive {
		t.Errorf("focus lost: %v %v", s, changed)
	}
	if s, changed := step(inactive, true, false); !changed || s != ui.AppLifecycleStateHidden {
		t.Errorf("app hidden after the first report is reported: %v %v", s, changed)
	}
	if s, changed := step(inactive, false, true); !changed || s != ui.AppLifecycleStateDetached {
		t.Errorf("ended: %v %v", s, changed)
	}
}
