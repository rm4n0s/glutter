package android

// frameRequests merges the requests for a frame (SPEC §24.3): the platform asks the choreographer
// for one callback at a time, and only while a frame can be drawn. Requests made while a frame is
// in flight, or while there is no surface, are kept as one more frame.
//
// Flutter: shell/platform/android/vsync_waiter_android.cc VsyncWaiterAndroid
type frameRequests struct {
	// wanted is whether a frame was asked for and has not been drawn.
	wanted bool
	// scheduled is whether a frame callback of the choreographer is waiting.
	scheduled bool
}

// request records a request for a frame.
func (f *frameRequests) request() { f.wanted = true }

// shouldSchedule says whether a callback must be asked for now: a frame is wanted, none is
// waiting, and a frame can be drawn (a surface, and the app is not hidden).
func (f *frameRequests) shouldSchedule(canDraw bool) bool {
	return f.wanted && !f.scheduled && canDraw
}

// scheduledNow records that the callback was asked for.
func (f *frameRequests) scheduledNow() { f.scheduled = true }

// fired records that the callback came and returns whether to draw: only when a frame is still
// wanted and can be drawn. The request is consumed before the frame is drawn, so that a request
// made during the frame asks for the next one.
func (f *frameRequests) fired(canDraw bool) (draw bool) {
	f.scheduled = false
	if !f.wanted || !canDraw {
		return false
	}
	f.wanted = false
	return true
}
