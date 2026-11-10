// Package scenelines holds the lines that the app of the scene tests (internal/iostest/sceneapp) prints and the test reads, so that
// the two cannot drift apart. It imports only package ui, so both the app (built for iOS) and the test (built for the host) use it.
//
// Flutter: no counterpart
package scenelines

import "github.com/rm4n0s/glutter/ui"

// The lines of the app; each is followed by its values.
const (
	// Metrics is the line of a change of the metrics: "<WxH in pixels> scale=<s> padding=<l>,<t>,<r>,<b>".
	Metrics = "GLUTTER_IOS_SCENE metrics"
	// Frame is the line of a frame: "<n> <WxH of the buffer>".
	Frame = "GLUTTER_IOS_SCENE frame"
	// Lifecycle is the line of a change of the lifecycle: the name of the state.
	Lifecycle = "GLUTTER_IOS_SCENE lifecycle"
	// Failed is the line of a call that failed.
	Failed = "GLUTTER_IOS_SCENE failed"
	// RotateEnv is the environment variable that makes the app ask for the landscape orientations after its first frame.
	RotateEnv = "GLUTTER_IOS_SCENE_ROTATE"
)

// StateName is the name that the app prints for a lifecycle state.
func StateName(s ui.AppLifecycleState) string {
	switch s {
	case ui.AppLifecycleStateResumed:
		return "resumed"
	case ui.AppLifecycleStateInactive:
		return "inactive"
	case ui.AppLifecycleStateHidden:
		return "hidden"
	case ui.AppLifecycleStatePaused:
		return "paused"
	case ui.AppLifecycleStateDetached:
		return "detached"
	}
	return "unspecified"
}
