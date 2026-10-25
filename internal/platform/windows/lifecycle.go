package windows

import "github.com/rm4n0s/glutter/ui"

// This file is the plain-Go part of the lifecycle of the app (SPEC §25.8), a port of Flutter's
// WindowsLifecycleManager: the platform reports the state itself, from what its windows are.

// lifecycleWindow is what the lifecycle looks at in a window.
type lifecycleWindow struct {
	// Visible is whether the window is on the screen: it was shown after its first frame and the
	// app did not hide it (SetHidden).
	Visible bool
	// Minimized is whether the window is minimized to the task bar.
	Minimized bool
	// Active is whether the window is the active window of the system.
	Active bool
}

// lifecycleOf is the lifecycle state of the windows:
//   - Detached when the app is ending (the last window was destroyed, or the session ends);
//   - Hidden when every window is minimized or hidden (and so with no window yet);
//   - Resumed when a window is visible, not minimized and active;
//   - Inactive when a window is visible and none is active.
//
// Flutter: shell/platform/windows/windows_lifecycle_manager.cc WindowsLifecycleManager::UpdateState
func lifecycleOf(windows []lifecycleWindow, ended bool) ui.AppLifecycleState {
	if ended {
		return ui.AppLifecycleStateDetached
	}
	state := ui.AppLifecycleStateHidden
	for _, w := range windows {
		if !w.Visible || w.Minimized {
			continue
		}
		if w.Active {
			return ui.AppLifecycleStateResumed
		}
		state = ui.AppLifecycleStateInactive
	}
	return state
}

// lifecycleTracker remembers the state that was reported, so that only a change is reported again.
type lifecycleTracker struct {
	last     ui.AppLifecycleState
	reported bool
}

// update gives the state to report: changed is false when it is the one reported last. The first
// state that is Hidden is not reported: an app with no window shown yet has not left its start
// state, and the first window brings Resumed or Inactive.
func (t *lifecycleTracker) update(windows []lifecycleWindow, ended bool) (state ui.AppLifecycleState, changed bool) {
	state = lifecycleOf(windows, ended)
	if t.reported && state == t.last {
		return state, false
	}
	if !t.reported && state == ui.AppLifecycleStateHidden {
		return state, false
	}
	t.last, t.reported = state, true
	return state, true
}
