package ui

// BackSwipeEdge is the edge of the screen that a back gesture starts from.
//
// Flutter: services/predictive_back_event.dart PredictiveBackEvent.swipeEdge (SwipeEdge)
type BackSwipeEdge int

// The values of BackSwipeEdge follow Flutter's SwipeEdge (left, right).
const (
	// BackSwipeEdgeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	BackSwipeEdgeUnspecified BackSwipeEdge = iota
	// BackSwipeEdgeLeft is the left edge.
	//
	// Flutter: services/predictive_back_event.dart SwipeEdge.left
	BackSwipeEdgeLeft
	// BackSwipeEdgeRight is the right edge.
	//
	// Flutter: services/predictive_back_event.dart SwipeEdge.right
	BackSwipeEdgeRight
)

// BackGestureEvent is a step of the back gesture of the system: where the finger is, how far the
// gesture has gone and from which edge it started.
//
// Flutter: services/predictive_back_event.dart PredictiveBackEvent
type BackGestureEvent struct {
	// HasTouchOffset is whether TouchX and TouchY are known: a back that a button or a key makes
	// has no finger.
	HasTouchOffset bool
	// TouchX and TouchY are the position of the finger in the view, in logical pixels.
	TouchX, TouchY float64
	// Progress is how far the gesture has gone, from 0 to 1.
	Progress float64
	// SwipeEdge is the edge that the gesture started from.
	SwipeEdge BackSwipeEdge
}

// BackHost is what the engine implements to tell the system whether the framework wants the back
// gesture. The dispatcher uses it when the host has it; a host without it does nothing.
//
// Flutter: no counterpart (the engine's side of the flutter/platform channel)
type BackHost interface {
	// SetFrameworkHandlesBack says whether the framework handles the back gesture of the system
	// (something that can pop, or a PopScope that cannot): the system then gives the gesture to
	// the app and shows its predictive animation, else it does its own (SPEC §24.8).
	SetFrameworkHandlesBack(handles bool)
}

// SetFrameworkHandlesBack tells the engine whether the framework handles the back gesture.
//
// Deviation: Flutter sends it as the SystemNavigator.setFrameworkHandlesBack call of the
// flutter/platform channel (SPEC §12.10).
//
// Flutter: services/system_navigator.dart SystemNavigator.setFrameworkHandlesBack
func (d *PlatformDispatcher) SetFrameworkHandlesBack(handles bool) {
	if h, ok := d.host.(BackHost); ok {
		h.SetFrameworkHandlesBack(handles)
	}
}

// PopRoute is called by the engine when the system asks the app to go back (the back button of an
// old Android, or the back gesture when it is not predictive): OnPopRoute decides what that is.
//
// Flutter: services/system_channels.dart SystemChannels.navigation (popRoute)
func (d *PlatformDispatcher) PopRoute() {
	if d.OnPopRoute != nil {
		d.OnPopRoute()
	}
}

// StartBackGesture is called by the engine when a predictive back gesture starts. It reports
// whether the framework took it.
//
// Flutter: widgets/binding.dart WidgetsBinding._handleStartBackGesture
func (d *PlatformDispatcher) StartBackGesture(e BackGestureEvent) bool {
	if d.OnStartBackGesture != nil {
		return d.OnStartBackGesture(e)
	}
	return false
}

// UpdateBackGestureProgress is called by the engine as a predictive back gesture goes on.
//
// Flutter: widgets/binding.dart WidgetsBinding._handleUpdateBackGestureProgress
func (d *PlatformDispatcher) UpdateBackGestureProgress(e BackGestureEvent) {
	if d.OnUpdateBackGestureProgress != nil {
		d.OnUpdateBackGestureProgress(e)
	}
}

// CommitBackGesture is called by the engine when the user finishes a predictive back gesture.
//
// Flutter: widgets/binding.dart WidgetsBinding._handleCommitBackGesture
func (d *PlatformDispatcher) CommitBackGesture() {
	if d.OnCommitBackGesture != nil {
		d.OnCommitBackGesture()
	}
}

// CancelBackGesture is called by the engine when the user cancels a predictive back gesture.
//
// Flutter: widgets/binding.dart WidgetsBinding._handleCancelBackGesture
func (d *PlatformDispatcher) CancelBackGesture() {
	if d.OnCancelBackGesture != nil {
		d.OnCancelBackGesture()
	}
}

// NotifyMemoryPressure is called by the engine when the system is low on memory.
//
// Flutter: services/system_channels.dart SystemChannels.system (memoryPressure)
func (d *PlatformDispatcher) NotifyMemoryPressure() {
	if d.OnMemoryPressure != nil {
		d.OnMemoryPressure()
	}
}
