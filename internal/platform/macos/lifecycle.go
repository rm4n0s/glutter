package macos

import "github.com/rm4n0s/glutter/ui"

// This file is the plain-Go part of the lifecycle of the app (SPEC §29.8): the platform reports the
// state itself, from what its windows and the app are, as Flutter's macOS embedder reports it from the
// notifications of NSApplication (FlutterAppLifecycleDelegate).

// lifecycleWindow is what the lifecycle looks at in a window.
type lifecycleWindow struct {
	// Visible is whether the window was shown after its first frame and the app did not hide it
	// (SetHidden).
	Visible bool
	// Minimized is whether the window is minimized to the Dock.
	Minimized bool
	// Key is whether the window is the key window of the app, with the keyboard focus.
	Key bool
}

// lifecycleOf is the lifecycle state:
//   - Detached when the app is ending (the loop is stopping);
//   - Hidden when the app is hidden (Command+H) or every window is minimized or hidden (and so with no
//     window yet);
//   - Resumed when a window is visible, not minimized and the key window;
//   - Inactive when a window is visible and none is the key window (another app is in front).
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterAppDelegate.mm FlutterAppDelegate applicationDidHide
func lifecycleOf(windows []lifecycleWindow, appHidden, ended bool) ui.AppLifecycleState {
	if ended {
		return ui.AppLifecycleStateDetached
	}
	if appHidden {
		return ui.AppLifecycleStateHidden
	}
	state := ui.AppLifecycleStateHidden
	for _, w := range windows {
		if !w.Visible || w.Minimized {
			continue
		}
		if w.Key {
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

// update gives the state to report: changed is false when it is the one reported last. A first state
// that is Hidden is not reported: an app with no window shown yet has not left its start state, and the
// first window brings Resumed or Inactive.
func (t *lifecycleTracker) update(windows []lifecycleWindow, appHidden, ended bool) (state ui.AppLifecycleState, changed bool) {
	state = lifecycleOf(windows, appHidden, ended)
	if t.reported && state == t.last {
		return state, false
	}
	if !t.reported && state == ui.AppLifecycleStateHidden {
		return state, false
	}
	t.last, t.reported = state, true
	return state, true
}
