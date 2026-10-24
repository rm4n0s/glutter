package gluttertest

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// BackGestureOpts are the optional parameters of StartBackGesture and
// UpdateBackGestureProgress: where the finger is and which edge the gesture
// started from.
//
// Flutter: flutter_test/lib/src/binding.dart (the arguments of the flutter/backgesture channel's startBackGesture and updateBackGestureProgress)
type BackGestureOpts struct {
	// Touch is the position of the finger in logical pixels; None is a back that
	// a button or a key makes (the event has no touch offset).
	Touch foundation.Opt[ui.Offset]
	// SwipeEdge is the edge the gesture started from; the default is the left
	// edge.
	SwipeEdge ui.BackSwipeEdge
}

func backGestureEvent(progress float64, opts []BackGestureOpts) ui.BackGestureEvent {
	var o BackGestureOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.SwipeEdge == ui.BackSwipeEdgeUnspecified {
		o.SwipeEdge = ui.BackSwipeEdgeLeft
	}
	e := ui.BackGestureEvent{Progress: progress, SwipeEdge: o.SwipeEdge}
	if touch, ok := o.Touch.Get(); ok {
		e.HasTouchOffset, e.TouchX, e.TouchY = true, touch.Dx, touch.Dy
	}
	return e
}

// StartBackGesture plays the system starting a predictive back gesture at the
// progress, as the platform does: through PlatformDispatcher.StartBackGesture.
// It reports whether the framework took the gesture (an observer returned true
// from HandleStartBackGesture). It runs what the call posted but does not pump;
// call Pump after it, as flutter_test's tests do.
//
// Flutter: widgets/binding.dart WidgetsBinding._handleStartBackGesture (the startBackGesture call of the flutter/backgesture channel)
func (t *Tester) StartBackGesture(progress float64, opts ...BackGestureOpts) bool {
	t.tb.Helper()
	t.adopt()
	took := t.d.StartBackGesture(backGestureEvent(progress, opts))
	t.FlushPosted()
	return took
}

// UpdateBackGestureProgress plays the system moving the finger of a predictive
// back gesture, through PlatformDispatcher.UpdateBackGestureProgress. It does
// not pump.
//
// Flutter: widgets/binding.dart WidgetsBinding._handleUpdateBackGestureProgress (the updateBackGestureProgress call of the flutter/backgesture channel)
func (t *Tester) UpdateBackGestureProgress(progress float64, opts ...BackGestureOpts) {
	t.tb.Helper()
	t.adopt()
	t.d.UpdateBackGestureProgress(backGestureEvent(progress, opts))
	t.FlushPosted()
}

// CommitBackGesture plays the system ending a predictive back gesture with a
// pop, through PlatformDispatcher.CommitBackGesture. It does not pump.
//
// Flutter: widgets/binding.dart WidgetsBinding._handleCommitBackGesture (the commitBackGesture call of the flutter/backgesture channel)
func (t *Tester) CommitBackGesture() {
	t.tb.Helper()
	t.adopt()
	t.d.CommitBackGesture()
	t.FlushPosted()
}

// CancelBackGesture plays the system ending a predictive back gesture without a
// pop, through PlatformDispatcher.CancelBackGesture. It does not pump.
//
// Flutter: widgets/binding.dart WidgetsBinding._handleCancelBackGesture (the cancelBackGesture call of the flutter/backgesture channel)
func (t *Tester) CancelBackGesture() {
	t.tb.Helper()
	t.adopt()
	t.d.CancelBackGesture()
	t.FlushPosted()
}

// PopRoute plays the system asking the app to go back with no gesture (the back
// button of an old Android), through PlatformDispatcher.PopRoute. It does not
// pump.
//
// Flutter: widgets/binding.dart WidgetsBinding.handlePopRoute (the popRoute call of the flutter/navigation channel)
func (t *Tester) PopRoute() {
	t.tb.Helper()
	t.adopt()
	t.d.PopRoute()
	t.FlushPosted()
}
