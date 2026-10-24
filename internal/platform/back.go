package platform

import "github.com/rm4n0s/glutter/ui"

// BackHandler is implemented by a Handler that takes the system's back navigation and its memory
// warnings: Android's back button and gesture (predictive or not), and onTrimMemory (SPEC §24.8).
// The platform asks for it with a type assertion; a handler without it ignores them.
//
// Flutter: no counterpart (the navigation channel, the back gesture channel and the system
// channel of the embedder)
type BackHandler interface {
	// OnPopRoute reports that the system asks the app to go back: the framework pops a route or
	// ends the app.
	OnPopRoute()
	// OnStartBackGesture reports that a predictive back gesture started and returns whether the
	// framework took it.
	OnStartBackGesture(e ui.BackGestureEvent) bool
	// OnUpdateBackGestureProgress reports the progress of a predictive back gesture.
	OnUpdateBackGestureProgress(e ui.BackGestureEvent)
	// OnCommitBackGesture reports that the gesture was completed: the route pops.
	OnCommitBackGesture()
	// OnCancelBackGesture reports that the gesture was cancelled.
	OnCancelBackGesture()
	// OnMemoryPressure reports that the system is low on memory.
	OnMemoryPressure()
}
