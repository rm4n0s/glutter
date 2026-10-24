package android

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// backRecorder is a handler that takes the back events and says what it got.
type backRecorder struct {
	recorder
	got []string
	// took is what OnStartBackGesture answers.
	took bool
}

func (b *backRecorder) note(s string) {
	b.mu.Lock()
	b.got = append(b.got, s)
	b.mu.Unlock()
}

// told are the names of what the handler has been told.
func (b *backRecorder) told() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.got...)
}

func (b *backRecorder) OnPopRoute() { b.note("pop") }
func (b *backRecorder) OnStartBackGesture(e ui.BackGestureEvent) bool {
	b.note("start")
	b.backs = append(b.backs, e)
	return b.took
}
func (b *backRecorder) OnUpdateBackGestureProgress(e ui.BackGestureEvent) {
	b.note("update")
	b.backs = append(b.backs, e)
}
func (b *backRecorder) OnCommitBackGesture() { b.note("commit") }
func (b *backRecorder) OnCancelBackGesture() { b.note("cancel") }
func (b *backRecorder) OnMemoryPressure()    { b.note("memory") }

func (b *backRecorder) backEvents() []ui.BackGestureEvent { return b.backs }

func backPlatform(t *testing.T) (*Platform, *fakeHost, *backRecorder) {
	t.Helper()
	p, host, _ := textPlatform(t)
	h := &backRecorder{}
	p.handler = h
	p.b.surfaceChanged(1080, 2400, 2.625, 60, 1080, 2400)
	p.b.surf.density = 2.625
	return p, host, h
}

// TestBackEventsReachTheHandlerInLogicalPixels checks the steps of a back gesture and a plain back: the
// finger is given in logical pixels, the edge as the framework's, and the plain back and the end of a
// gesture have no finger.
func TestBackEventsReachTheHandlerInLogicalPixels(t *testing.T) {
	p, _, h := backPlatform(t)
	p.onBack(backPop, 0, 0, 0, 0)
	p.onBack(backStart, 262.5, 525, 0, 1)
	p.onBack(backProgress, 525, 525, 0.5, 2)
	p.onBack(backCommit, 0, 0, 0, 0)
	p.onBack(backCancel, 0, 0, 0, 0)
	if want := []string{"pop", "start", "update", "commit", "cancel"}; !reflect.DeepEqual(h.got, want) {
		t.Fatalf("the handler got %v, want %v", h.got, want)
	}
	want := []ui.BackGestureEvent{
		{HasTouchOffset: true, TouchX: 100, TouchY: 200, Progress: 0, SwipeEdge: ui.BackSwipeEdgeLeft},
		{HasTouchOffset: true, TouchX: 200, TouchY: 200, Progress: 0.5, SwipeEdge: ui.BackSwipeEdgeRight},
	}
	if got := h.backEvents(); !reflect.DeepEqual(got, want) {
		t.Errorf("the events were %+v, want %+v", got, want)
	}
}

// TestABackFromAButtonIsNotAGesture has a start and progress with no edge (Android's EDGE_NONE, a back
// key) dropped, so that the commit that follows is a plain pop.
func TestABackFromAButtonIsNotAGesture(t *testing.T) {
	p, _, h := backPlatform(t)
	p.onBack(backStart, 0, 0, 0, 3)
	p.onBack(backProgress, 0, 0, 0.2, 3)
	p.onBack(backCommit, 0, 0, 0, 0)
	if want := []string{"commit"}; !reflect.DeepEqual(h.got, want) {
		t.Fatalf("the handler got %v, want %v", h.got, want)
	}
}

// TestMemoryIsLowOnlyFromTheRunningLowLevelAndAfterTheFirstFrame has the trim levels of Android:
// the framework hears of memory pressure from TRIM_MEMORY_RUNNING_LOW, once a frame was drawn.
func TestMemoryIsLowOnlyFromTheRunningLowLevelAndAfterTheFirstFrame(t *testing.T) {
	p, _, h := backPlatform(t)
	p.onTrimMemory(15) // no frame yet
	p.framed = true
	for _, level := range []int{5, 9} {
		p.onTrimMemory(level)
	}
	if len(h.got) != 0 {
		t.Fatalf("memory pressure at a low level: %v", h.got)
	}
	for _, level := range []int{10, 15, 80} {
		p.onTrimMemory(level)
	}
	if len(h.got) != 3 || h.got[0] != "memory" {
		t.Errorf("the handler got %v", h.got)
	}
}

// TestFrameworkHandlesBackIsSentOnChangeAndAgainForANewActivity checks the commands for the back callback
// of the activity, and that a new activity in the process is told again.
func TestFrameworkHandlesBackIsSentOnChangeAndAgainForANewActivity(t *testing.T) {
	p, host, _ := backPlatform(t)
	w := p.win
	w.SetFrameworkHandlesBack(false) // the state it has already: nothing
	w.SetFrameworkHandlesBack(true)
	w.SetFrameworkHandlesBack(true)
	w.SetFrameworkHandlesBack(false)
	w.SetFrameworkHandlesBack(true)
	p.activityCreated()
	want := []command{
		{Op: opFrameworkHandlesBack, A: 1}, {Op: opFrameworkHandlesBack, A: 0}, {Op: opFrameworkHandlesBack, A: 1},
		{Op: opFrameworkHandlesBack, A: 1},
	}
	if got := host.sentCommands(); !reflect.DeepEqual(got, want) {
		t.Errorf("the commands were %+v, want %+v", got, want)
	}
	// A text field that is focused is started again, with the framework's state.
	w.SetTextInput(plainField("ab", 2, 2))
	n := len(host.sentCommands())
	p.activityCreated()
	got := host.sentCommands()[n:]
	if len(got) != 3 || got[1].Op != opTextStart || got[2] != (command{Op: opTextState, A: 0, B: 2, C: 2, S: "ab"}) {
		t.Errorf("a new activity was sent %+v", got)
	}
}

// TestTheFocusIsPartOfTheLifecycle runs the platform and has the activity go through its life: resumed
// with the focus, without it, stopped, resumed again and destroyed.
func TestTheFocusIsPartOfTheLifecycle(t *testing.T) {
	h := newHarness(t)
	h.b.lifecycleChanged(lifecycleResumed)
	h.b.focusChanged(true)
	h.run()
	steps := []struct {
		do   func()
		want string
	}{
		{func() { h.b.focusChanged(false) }, "lifecycle " + ui.AppLifecycleStateInactive.String()},
		{func() { h.b.lifecycleChanged(lifecyclePaused) }, "lifecycle " + ui.AppLifecycleStateInactive.String()},
		{func() { h.b.lifecycleChanged(lifecycleStopped) }, "lifecycle " + ui.AppLifecycleStatePaused.String()},
		{func() { h.b.lifecycleChanged(lifecycleResumed) }, "lifecycle " + ui.AppLifecycleStateInactive.String()},
		{func() { h.b.focusChanged(true) }, "lifecycle " + ui.AppLifecycleStateResumed.String()},
		{func() { h.b.lifecycleChanged(lifecycleDestroyed) }, "lifecycle " + ui.AppLifecycleStateDetached.String()},
	}
	for i, s := range steps {
		before := len(h.h.log())
		s.do()
		waitFor(t, "the step", func() bool { return len(h.h.log()) > before })
		var last string
		for _, l := range h.h.log()[before:] {
			if len(l) >= 9 && l[:9] == "lifecycle" {
				last = l
			}
		}
		if last != s.want {
			t.Errorf("step %d: the app was told %q (calls %v), want %q", i, last, h.h.log()[before:], s.want)
		}
	}
}

// TestTheBridgeQueuesTheBackEventsOfTheJavaSide runs the platform and gives the bridge what the native
// methods give it: a back, the trim memory of the system and a new activity.
func TestTheBridgeQueuesTheBackEventsOfTheJavaSide(t *testing.T) {
	b := newBridge()
	host := newFakeHost()
	p := newPlatform(b, host, nil)
	rec := &backRecorder{}
	ctx, cancel := contextWithCancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, rec) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	waitFor(t, "Run to start", func() bool { return rec.count("lifecycle") > 0 })
	p.Post(func() { p.framed = true })
	b.back(backPop, 0, 0, 0, 0)
	b.trimMemory(80)
	b.activityCreated()
	waitFor(t, "the events", func() bool { return len(rec.told()) == 2 })
	if want := []string{"pop", "memory"}; !reflect.DeepEqual(rec.told(), want) {
		t.Errorf("the handler got %v, want %v", rec.told(), want)
	}
}
