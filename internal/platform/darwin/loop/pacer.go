package loop

import "sync/atomic"

// FrameRequests merges the requests of one window for a frame (SPEC §29.3): any number made while a
// frame is wanted are one frame, a request made during a frame asks for the next, and the request of a
// window that cannot draw (minimized, hidden) is kept until it can.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterVSyncWaiter.mm (one request is answered
// by one tick of the display link)
type FrameRequests struct {
	wanted bool
}

// Request records a request for a frame.
//
// Flutter: no counterpart
func (f *FrameRequests) Request() { f.wanted = true }

// Wanted says whether a frame is wanted now.
//
// Flutter: no counterpart
func (f *FrameRequests) Wanted() bool { return f.wanted }

// Take says whether the window draws in this tick: a frame is wanted and the window can draw it. The
// Request is consumed before the frame is drawn, so that a request made during the frame asks for the
// one after it.
//
// Flutter: no counterpart
func (f *FrameRequests) Take(canDraw bool) bool {
	if !f.wanted || !canDraw {
		return false
	}
	f.wanted = false
	return true
}

// Clear drops the request: a frame was drawn for another reason (a live resize draws at once).
//
// Flutter: no counterpart
func (f *FrameRequests) Clear() { f.wanted = false }

// LinkPacer decides when the display link runs (SPEC §29.3): it runs only while some window wants a
// frame and can draw it, so an idle app has no link ticking. It belongs to the UI thread.
//
// Flutter: no counterpart (FlutterDisplayLink is paused by the vsync waiter when nothing waits)
type LinkPacer struct {
	running bool
}

// Running says whether the link is running now: Update last said to start it and has not said to stop it.
//
// Flutter: no counterpart
func (p *LinkPacer) Running() bool { return p.running }

// Update is told after every change whether some window wants a frame it can draw, and says whether the
// link must be started (true, false) or stopped (false, true). With neither it is already as wanted.
//
// Flutter: no counterpart
func (p *LinkPacer) Update(anyWanted bool) (start, stop bool) {
	switch {
	case anyWanted && !p.running:
		p.running = true
		return true, false
	case !anyWanted && p.running:
		p.running = false
		return false, true
	}
	return false, false
}

// TickGate lets one tick of a display link thread (CVDisplayLink calls back on a thread of its own) be
// offered to the loop at a time: a tick that arrives while the previous one is still waiting is dropped,
// as Flutter's FlutterDisplayLink does, so a slow loop never collects a queue of ticks. It is safe to
// use from the link's thread and from the UI thread.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterDisplayLink.mm (the tick is dropped
// while the main thread has not handled the last)
type TickGate struct {
	pending atomic.Bool
}

// Offer says whether the tick must be posted to the loop: false when one is already on its way.
//
// Flutter: no counterpart
func (g *TickGate) Offer() bool { return g.pending.CompareAndSwap(false, true) }

// Done records that the loop handled the tick, so that the next one is let through.
//
// Flutter: no counterpart
func (g *TickGate) Done() { g.pending.Store(false) }

// LinkWatch decides which of two display links paces the frames (SPEC §29.3): the preferred one (the
// view's CADisplayLink, which follows the display and its refresh rate) and a fallback that always ticks
// (a CVDisplayLink). The preferred link stays silent for a window on a display that sleeps, which is also
// what a Mac shows when it is used through Screen Sharing with its own screen asleep: the window is
// looked at and must be drawn. So the preferred link is watched while it runs: when a watch interval
// passes with no tick of it the fallback takes over, a tick of the preferred link takes the pacing back,
// and while the preferred link is silent every start goes to the fallback at once. It belongs to the UI
// thread.
//
// Flutter: no counterpart (FlutterDisplayLink is a CVDisplayLink on every version of macOS)
type LinkWatch struct {
	running, fallback, ticked bool
}

// Start is told that the links are wanted. It says whether the fallback must be started as well (the
// preferred link was silent the last time it was watched); the preferred link is always started, so that
// its first tick is seen.
//
// Flutter: no counterpart
func (w *LinkWatch) Start() (startFallback bool) {
	w.running, w.ticked = true, false
	return w.fallback
}

// Stop is told that no link is wanted any more. The fallback, when it ran, is stopped by the caller;
// that the preferred link was silent is remembered for the next Start.
//
// Flutter: no counterpart
func (w *LinkWatch) Stop() { w.running = false }

// Tick is told of a tick of the preferred link. It says whether the fallback must be stopped: the
// preferred link paces the frames again.
//
// Flutter: no counterpart
func (w *LinkWatch) Tick() (stopFallback bool) {
	w.ticked = true
	if w.fallback {
		w.fallback = false
		return w.running
	}
	return false
}

// Check is called when a watch interval has passed. It says whether the fallback must be started: the
// links run, the preferred one paces and it did not tick in the interval. The next interval starts.
//
// Flutter: no counterpart
func (w *LinkWatch) Check() (startFallback bool) {
	if !w.running || w.fallback {
		return false
	}
	silent := !w.ticked
	w.ticked = false
	if silent {
		w.fallback = true
	}
	return silent
}

// Fallback says whether the fallback paces the frames: a tick of it is a frame only then.
//
// Flutter: no counterpart
func (w *LinkWatch) Fallback() bool { return w.fallback }

// Watching says whether a watch interval must be timed: the links run and the preferred one paces.
//
// Flutter: no counterpart
func (w *LinkWatch) Watching() bool { return w.running && !w.fallback }
