package macos

import "sync/atomic"

// frameRequests merges the requests of one window for a frame (SPEC §29.3): any number made while a
// frame is wanted are one frame, a request made during a frame asks for the next, and the request of a
// window that cannot draw (minimized, hidden) is kept until it can.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterVSyncWaiter.mm (one request is answered
// by one tick of the display link)
type frameRequests struct {
	wanted bool
}

// request records a request for a frame.
func (f *frameRequests) request() { f.wanted = true }

// take says whether the window draws in this tick: a frame is wanted and the window can draw it. The
// request is consumed before the frame is drawn, so that a request made during the frame asks for the
// one after it.
func (f *frameRequests) take(canDraw bool) bool {
	if !f.wanted || !canDraw {
		return false
	}
	f.wanted = false
	return true
}

// clear drops the request: a frame was drawn for another reason (a live resize draws at once).
func (f *frameRequests) clear() { f.wanted = false }

// linkPacer decides when the display link runs (SPEC §29.3): it runs only while some window wants a
// frame and can draw it, so an idle app has no link ticking. It belongs to the UI thread.
//
// Flutter: no counterpart (FlutterDisplayLink is paused by the vsync waiter when nothing waits)
type linkPacer struct {
	running bool
}

// update is told after every change whether some window wants a frame it can draw, and says whether the
// link must be started (true, false) or stopped (false, true). With neither it is already as wanted.
func (p *linkPacer) update(anyWanted bool) (start, stop bool) {
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

// tickGate lets one tick of a display link thread (CVDisplayLink calls back on a thread of its own) be
// offered to the loop at a time: a tick that arrives while the previous one is still waiting is dropped,
// as Flutter's FlutterDisplayLink does, so a slow loop never collects a queue of ticks. It is safe to
// use from the link's thread and from the UI thread.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterDisplayLink.mm (the tick is dropped
// while the main thread has not handled the last)
type tickGate struct {
	pending atomic.Bool
}

// offer says whether the tick must be posted to the loop: false when one is already on its way.
func (g *tickGate) offer() bool { return g.pending.CompareAndSwap(false, true) }

// done records that the loop handled the tick, so that the next one is let through.
func (g *tickGate) done() { g.pending.Store(false) }
