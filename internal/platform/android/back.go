package android

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The operations of the activity's command call for back navigation (textinput.go has 8 to 13).
const (
	// opFrameworkHandlesBack: A is 1 when the framework handles the back gesture, 0 when it does not.
	opFrameworkHandlesBack = 14
	// 15 finishes the activity (BackNavigation.OP_FINISH); Go does not send it, the end of Run does.
)

// The kinds of the back events of the Java side (Native.back).
const (
	// backPop is a plain back: the framework pops a route, or ends the app.
	backPop = 1
	// backStart, backProgress, backCommit and backCancel are the steps of a predictive back gesture.
	backStart    = 2
	backProgress = 3
	backCommit   = 4
	backCancel   = 5
)

// trimMemoryRunningLow is ComponentCallbacks2.TRIM_MEMORY_RUNNING_LOW: from this level the app is
// told that memory is low, once its first frame is drawn.
const trimMemoryRunningLow = 10

var _ ui.BackHost = (*window)(nil)

// SetFrameworkHandlesBack registers the activity's back callback while the framework wants the
// back gesture (a route that can pop, or a PopScope), so that the system gives the gesture to the
// app and shows its predictive animation, and unregisters it otherwise: the system then does its
// own back (SPEC §24.8).
//
// Flutter: shell/platform/android/io/flutter/embedding/android/FlutterActivityAndFragmentDelegate.java FlutterActivityAndFragmentDelegate.setFrameworkHandlesBack
func (w *window) SetFrameworkHandlesBack(handles bool) {
	if w.p.handlesBack == handles {
		return
	}
	w.p.handlesBack = handles
	w.p.sendHandlesBack()
}

// sendHandlesBack tells the activity whether the framework handles back.
func (p *Platform) sendHandlesBack() {
	a := int64(0)
	if p.handlesBack {
		a = 1
	}
	p.host.Command(command{Op: opFrameworkHandlesBack, A: a})
}

// onBack is a back event of the Java side (Native.back): a plain back, or a step of a predictive
// gesture, with the finger in pixels of the view and the progress from 0 to 1. The framework gets
// it in logical pixels.
//
// Flutter: shell/platform/android/io/flutter/embedding/engine/systemchannels/BackGestureChannel.java BackGestureChannel
func (p *Platform) onBack(kind int, x, y, progress float64, edge int) {
	h, ok := p.handler.(platform.BackHandler)
	if !ok {
		return
	}
	state := p.b.snapshot()
	dpr := state.pixelRatio()
	e := ui.BackGestureEvent{HasTouchOffset: kind == backStart || kind == backProgress, TouchX: x / dpr, TouchY: y / dpr, Progress: progress}
	switch edge {
	case 1:
		e.SwipeEdge = ui.BackSwipeEdgeLeft
	case 2:
		e.SwipeEdge = ui.BackSwipeEdgeRight
	}
	if (kind == backStart || kind == backProgress) && e.SwipeEdge == ui.BackSwipeEdgeUnspecified {
		// A back from a button or a key (BackEvent.EDGE_NONE) is not a gesture: Android starts it as
		// one, but there is nothing to animate, and the commit that follows is a plain pop
		// (HandleCommitBackGesture with no observer that took the start).
		return
	}
	switch kind {
	case backPop:
		h.OnPopRoute()
	case backStart:
		h.OnStartBackGesture(e)
	case backProgress:
		h.OnUpdateBackGestureProgress(e)
	case backCommit:
		h.OnCommitBackGesture()
	case backCancel:
		h.OnCancelBackGesture()
	}
}

// onTrimMemory is the system's call for memory (Native.trimMemory): from the level of
// TRIM_MEMORY_RUNNING_LOW, and once the first frame is drawn, the framework is told that memory is
// low (SPEC §24.8).
//
// Flutter: shell/platform/android/io/flutter/embedding/android/FlutterActivityAndFragmentDelegate.java FlutterActivityAndFragmentDelegate.onTrimMemory
func (p *Platform) onTrimMemory(level int) {
	if level < trimMemoryRunningLow || !p.framed {
		return
	}
	if h, ok := p.handler.(platform.BackHandler); ok {
		h.OnMemoryPressure()
	}
}

// activityCreated is a new activity in the process (Native.activityCreated), which knows nothing of
// what the framework asked of the one before: the callback for the back gesture and the text input
// are asked for again (SPEC §24.2).
func (p *Platform) activityCreated() {
	if p.handlesBack {
		p.sendHandlesBack()
	}
	p.failSpellChecks()
	p.text.resend(p.host)
	p.a11y.resend()
}

// bridge methods

// back is a back event of the Java side.
func (b *bridge) back(kind int, x, y, progress float64, edge int) {
	b.post(func(p *Platform) { p.onBack(kind, x, y, progress, edge) })
}

// trimMemory is the system's call for memory.
func (b *bridge) trimMemory(level int) {
	b.post(func(p *Platform) { p.onTrimMemory(level) })
}

// activityCreated is a new activity.
func (b *bridge) activityCreated() {
	b.post(func(p *Platform) { p.activityCreated() })
}
