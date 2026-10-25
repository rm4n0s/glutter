package windows

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestLifecycleOf(t *testing.T) {
	shown := lifecycleWindow{Visible: true}
	active := lifecycleWindow{Visible: true, Active: true}
	tests := []struct {
		name    string
		windows []lifecycleWindow
		ended   bool
		want    ui.AppLifecycleState
	}{
		{"no window", nil, false, ui.AppLifecycleStateHidden},
		{"a visible active window", []lifecycleWindow{active}, false, ui.AppLifecycleStateResumed},
		{"a visible window that is not active", []lifecycleWindow{shown}, false, ui.AppLifecycleStateInactive},
		{"a minimized window", []lifecycleWindow{{Visible: true, Minimized: true}}, false, ui.AppLifecycleStateHidden},
		{"a minimized window that is still active", []lifecycleWindow{{Visible: true, Minimized: true, Active: true}}, false, ui.AppLifecycleStateHidden},
		{"a hidden window", []lifecycleWindow{{Active: true}}, false, ui.AppLifecycleStateHidden},
		{"two windows, one active", []lifecycleWindow{shown, active}, false, ui.AppLifecycleStateResumed},
		{"two windows, none active", []lifecycleWindow{shown, shown}, false, ui.AppLifecycleStateInactive},
		{"one minimized and one visible", []lifecycleWindow{{Visible: true, Minimized: true}, shown}, false, ui.AppLifecycleStateInactive},
		{"all minimized or hidden", []lifecycleWindow{{Visible: true, Minimized: true}, {}}, false, ui.AppLifecycleStateHidden},
		{"the active window is hidden, another is visible", []lifecycleWindow{{Active: true}, shown}, false, ui.AppLifecycleStateInactive},
		{"the session ends", []lifecycleWindow{active}, true, ui.AppLifecycleStateDetached},
		{"the app ends with no window", nil, true, ui.AppLifecycleStateDetached},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lifecycleOf(tt.windows, tt.ended); got != tt.want {
				t.Errorf("state = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLifecycleTrackerReportsChanges(t *testing.T) {
	active := []lifecycleWindow{{Visible: true, Active: true}}
	inactive := []lifecycleWindow{{Visible: true}}
	minimized := []lifecycleWindow{{Visible: true, Minimized: true}}
	steps := []struct {
		name    string
		windows []lifecycleWindow
		ended   bool
		want    ui.AppLifecycleState
		changed bool
	}{
		{"no window yet is not reported", nil, false, ui.AppLifecycleStateHidden, false},
		{"the first window is shown and active", active, false, ui.AppLifecycleStateResumed, true},
		{"nothing changed", active, false, ui.AppLifecycleStateResumed, false},
		{"another program is active", inactive, false, ui.AppLifecycleStateInactive, true},
		{"the window is active again", active, false, ui.AppLifecycleStateResumed, true},
		{"minimized", minimized, false, ui.AppLifecycleStateHidden, true},
		{"still minimized", minimized, false, ui.AppLifecycleStateHidden, false},
		{"restored", active, false, ui.AppLifecycleStateResumed, true},
		{"the session ends", active, true, ui.AppLifecycleStateDetached, true},
		{"the end is reported once", nil, true, ui.AppLifecycleStateDetached, false},
	}
	var tr lifecycleTracker
	for _, s := range steps {
		got, changed := tr.update(s.windows, s.ended)
		if got != s.want || changed != s.changed {
			t.Errorf("%s: state %v changed %v, want %v and %v", s.name, got, changed, s.want, s.changed)
		}
	}
	// An app that starts with its window minimized is reported Hidden once it has been something else.
	var tr2 lifecycleTracker
	if _, changed := tr2.update(minimized, false); changed {
		t.Error("an app that starts minimized reported Hidden before it was anything else")
	}
}
