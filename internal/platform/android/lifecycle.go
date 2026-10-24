package android

import "github.com/rm4n0s/glutter/ui"

// lifecycleState is what the activity reports (SPEC §24.8): resumed, paused, stopped and destroyed.
// The framework's state also depends on whether the window has the focus.
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
	// lifecycleDestroyed is onDestroy of an activity that finishes: the app is detached.
	lifecycleDestroyed
)

// appState maps the activity's state and the focus of its window to the framework's: Resumed while
// the activity is resumed and has the focus, Inactive while it is resumed without it or paused,
// Paused once it is stopped (the framework goes through Hidden itself), Detached when it is
// destroyed. Before the activity said anything the app is Resumed.
//
// Flutter: shell/platform/android/io/flutter/embedding/engine/systemchannels/LifecycleChannel.java LifecycleChannel
func (l lifecycleState) appState(focused bool) ui.AppLifecycleState {
	switch l {
	case lifecycleResumed:
		if focused {
			return ui.AppLifecycleStateResumed
		}
		return ui.AppLifecycleStateInactive
	case lifecyclePaused:
		return ui.AppLifecycleStateInactive
	case lifecycleStopped:
		return ui.AppLifecycleStatePaused
	case lifecycleDestroyed:
		return ui.AppLifecycleStateDetached
	}
	return ui.AppLifecycleStateResumed
}

// visible says whether frames can be drawn: the activity is neither stopped nor destroyed.
func (l lifecycleState) visible() bool { return l != lifecycleStopped && l != lifecycleDestroyed }
