package ios

import "github.com/rm4n0s/glutter/ui"

// This file is the plain-Go part of the lifecycle of the app (SPEC §30.7): the platform reports the state itself, from the
// activation state of the scenes of the app, as Flutter's iOS embedder reports it from the notifications of the scene and the app.

// activation is the activation state of a window scene (UISceneActivationState); the numbers are UIKit's.
type activation int

const (
	// activationUnattached is a scene that is not connected to the app (UISceneActivationStateUnattached).
	activationUnattached activation = -1
	// activationForegroundActive is a scene on the screen that receives events (UISceneActivationStateForegroundActive).
	activationForegroundActive activation = 0
	// activationForegroundInactive is a scene on the screen that receives no events: an alert, the app switcher, a call
	// (UISceneActivationStateForegroundInactive).
	activationForegroundInactive activation = 1
	// activationBackground is a scene that is not on the screen (UISceneActivationStateBackground).
	activationBackground activation = 2
)

// lifecycleOf is the lifecycle state:
//   - Detached when the app is ending;
//   - Resumed when a scene is in the foreground and active;
//   - Inactive when a scene is in the foreground and not active;
//   - Paused when every scene is in the background (the app is not visible to the user and runs in the background);
//   - Detached when no scene is connected.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterAppDelegate.mm applicationDidEnterBackground
func lifecycleOf(scenes []activation, ended bool) ui.AppLifecycleState {
	if ended {
		return ui.AppLifecycleStateDetached
	}
	state := ui.AppLifecycleStateDetached
	for _, s := range scenes {
		switch s {
		case activationForegroundActive:
			return ui.AppLifecycleStateResumed
		case activationForegroundInactive:
			state = ui.AppLifecycleStateInactive
		case activationBackground:
			if state == ui.AppLifecycleStateDetached {
				state = ui.AppLifecycleStatePaused
			}
		}
	}
	return state
}

// lifecycleTracker remembers the state that was reported, so that only a change is reported again.
type lifecycleTracker struct {
	last     ui.AppLifecycleState
	reported bool
}

// update gives the state to report: changed is false when it is the one reported last. A first state that is Detached is not
// reported: an app whose scene has not connected has not left its start state.
func (t *lifecycleTracker) update(scenes []activation, ended bool) (state ui.AppLifecycleState, changed bool) {
	state = lifecycleOf(scenes, ended)
	if t.reported && state == t.last {
		return state, false
	}
	if !t.reported && state == ui.AppLifecycleStateDetached && !ended {
		return state, false
	}
	t.last, t.reported = state, true
	return state, true
}
