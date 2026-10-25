package windows

// frameRequests merges the requests of one window for a frame (SPEC §25.3): any number made while a
// frame is wanted are one frame, a request made during a frame asks for the next, and the request
// of a window that cannot draw (minimized, hidden) is kept until it can.
//
// Flutter: no counterpart (the embedder asks the engine for one frame at a time, shell/platform/windows/flutter_windows_engine.cc)
type frameRequests struct {
	wanted bool
}

// request records a request for a frame.
func (f *frameRequests) request() { f.wanted = true }

// take says whether the window draws in this vsync tick: a frame is wanted and the window can
// draw it. The request is consumed before the frame is drawn, so that a request made during the
// frame asks for the one after it.
func (f *frameRequests) take(canDraw bool) bool {
	if !f.wanted || !canDraw {
		return false
	}
	f.wanted = false
	return true
}

// clear drops the request: a frame was drawn for another reason (a resize draws at once).
func (f *frameRequests) clear() { f.wanted = false }

// vsyncGate allows one tick of the vsync thread to be asked for and not delivered at a time, so
// that the thread works only while a frame is wanted and an idle app arms nothing.
type vsyncGate struct {
	pending bool
}

// shouldRequest says whether the thread must be asked for a tick: some window wants a frame and
// can draw it, and no tick is on its way.
func (g *vsyncGate) shouldRequest(anyCanDraw bool) bool {
	if g.pending || !anyCanDraw {
		return false
	}
	g.pending = true
	return true
}

// delivered records that the tick came.
func (g *vsyncGate) delivered() { g.pending = false }
