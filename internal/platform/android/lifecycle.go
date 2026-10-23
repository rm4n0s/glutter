package android

import "github.com/rm4n0s/glutter/ui"

// lifecycleState is what the activity reports (SPEC §24.8): the first version of step 25.3 knows
// resumed, paused and stopped; step 25.9 completes it.
//
// Flutter: shell/platform/android/io/flutter/embedding/engine/systemchannels/LifecycleChannel.java
type lifecycleState int

// The states the Java side reports. The values are the ones GlutterActivity passes to Native.
const (
	// lifecycleUnknown is the state before the activity said anything.
	lifecycleUnknown lifecycleState = iota
	// lifecycleResumed is onResume.
	lifecycleResumed
	// lifecyclePaused is onPause: the app is visible and not in the foreground.
	lifecyclePaused
	// lifecycleStopped is onStop: the app is not visible.
	lifecycleStopped
)

// appState maps the activity's state to the framework's.
func (l lifecycleState) appState() ui.AppLifecycleState {
	switch l {
	case lifecycleResumed:
		return ui.AppLifecycleStateResumed
	case lifecyclePaused:
		return ui.AppLifecycleStateInactive
	case lifecycleStopped:
		return ui.AppLifecycleStateHidden
	}
	return ui.AppLifecycleStateResumed
}

// visible says whether frames can be drawn: the activity is not stopped.
func (l lifecycleState) visible() bool { return l != lifecycleStopped }
