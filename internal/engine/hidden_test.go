package engine

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/ui"
)

func TestHiddenWindowsGiveTheLifecycle(t *testing.T) {
	type step struct {
		do   func(e *windowsEngine, second ui.ViewID)
		want ui.AppLifecycleState
	}
	hide := func(view ui.ViewID) func(*windowsEngine, ui.ViewID) {
		return func(e *windowsEngine, _ ui.ViewID) { e.d.SetWindowHidden(view, true) }
	}
	show := func(view ui.ViewID) func(*windowsEngine, ui.ViewID) {
		return func(e *windowsEngine, _ ui.ViewID) { e.d.SetWindowHidden(view, false) }
	}
	focus := func(view ui.ViewID, focused bool) func(*windowsEngine, ui.ViewID) {
		return func(e *windowsEngine, _ ui.ViewID) {
			e.h.OnViewFocus(view, focused)
			e.h.input.flush() // as the next frame does
		}
	}
	hideSecond := func(e *windowsEngine, second ui.ViewID) { e.d.SetWindowHidden(second, true) }
	showSecond := func(e *windowsEngine, second ui.ViewID) { e.d.SetWindowHidden(second, false) }
	closeFirst := func(e *windowsEngine, _ ui.ViewID) { e.h.OnCloseRequest(ui.ImplicitViewID) }
	tests := []struct {
		name       string
		twoWindows bool
		steps      []step
	}{
		{"the only window is hidden, then shown and focused", false, []step{
			{focus(ui.ImplicitViewID, true), ui.AppLifecycleStateResumed},
			{hide(ui.ImplicitViewID), ui.AppLifecycleStateHidden},
			{focus(ui.ImplicitViewID, false), ui.AppLifecycleStateHidden},
			{show(ui.ImplicitViewID), ui.AppLifecycleStateInactive},
			{focus(ui.ImplicitViewID, true), ui.AppLifecycleStateResumed},
		}},
		{"hiding the same window twice", false, []step{
			{hide(ui.ImplicitViewID), ui.AppLifecycleStateHidden},
			{hide(ui.ImplicitViewID), ui.AppLifecycleStateHidden},
			{show(ui.ImplicitViewID), ui.AppLifecycleStateInactive},
		}},
		{"the app is hidden only when every window is", true, []step{
			{focus(ui.ImplicitViewID, true), ui.AppLifecycleStateResumed},
			{hideSecond, ui.AppLifecycleStateResumed},
			{hide(ui.ImplicitViewID), ui.AppLifecycleStateHidden},
			{showSecond, ui.AppLifecycleStateInactive},
		}},
		{"closing the last visible window leaves the hidden ones", true, []step{
			{hideSecond, ui.AppLifecycleStateInactive},
			{closeFirst, ui.AppLifecycleStateHidden},
			{showSecond, ui.AppLifecycleStateInactive},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newWindowsEngine(t)
			var second ui.ViewID
			if tt.twoWindows {
				v, err := e.d.CreateWindow(ui.WindowSpec{Title: "Second"})
				if err != nil {
					t.Fatal(err)
				}
				second = v.ViewID()
			}
			var got []ui.AppLifecycleState
			e.d.OnAppLifecycleStateChanged = func(s ui.AppLifecycleState) { got = append(got, s) }
			for i, s := range tt.steps {
				s.do(e, second)
				if state := e.d.AppLifecycleState(); state != s.want {
					t.Fatalf("step %d: lifecycle %v, want %v (changes %v)", i, state, s.want, got)
				}
			}
		})
	}
}

func TestAPlatformThatReportsTheLifecycleKeepsIt(t *testing.T) {
	e := newWindowsEngine(t)
	e.h.OnLifecycle(ui.AppLifecycleStateResumed)
	e.d.SetWindowHidden(ui.ImplicitViewID, true)
	e.h.OnViewFocus(ui.ImplicitViewID, false)
	e.h.input.flush()
	if state := e.d.AppLifecycleState(); state != ui.AppLifecycleStateResumed {
		t.Errorf("lifecycle %v, want the platform's Resumed", state)
	}
}

func TestAHiddenWindowAsksForNoFrameUntilItIsShown(t *testing.T) {
	e := newWindowsEngine(t)
	e.size(ui.ImplicitViewID, 400, 300)
	e.renderAll(nil)
	e.h.OnFrame(ui.ImplicitViewID, time.Millisecond) // makes the renderer
	win := e.p.Windows()[0]
	frames := func() int {
		n := 0
		for _, entry := range e.p.Log() {
			if entry.Kind == fake.EntryRequestFrame {
				n++
			}
		}
		return n
	}

	e.d.SetWindowHidden(ui.ImplicitViewID, true)
	if !win.Hidden() {
		t.Fatal("the platform window is not hidden")
	}
	before := frames()
	e.h.ScheduleFrame()
	if frames() != before {
		t.Error("a hidden window asked the platform for a frame")
	}

	e.d.SetWindowHidden(ui.ImplicitViewID, false)
	if win.Hidden() {
		t.Fatal("the platform window is still hidden")
	}
	if frames() != before+1 {
		t.Errorf("%d frames asked for when the window was shown, want the one that waited", frames()-before)
	}
	e.r.mu.Lock()
	invalidated := len(e.r.invalidated)
	e.r.mu.Unlock()
	if invalidated != 1 {
		t.Errorf("the window was invalidated %d times, want once: its next frame is presented whole", invalidated)
	}
	if e.d.View(ui.ImplicitViewID) == nil {
		t.Error("hiding the window removed its view")
	}
}

func TestRunGoesOnWithAHiddenWindowAndEndsOnExit(t *testing.T) {
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	var d *ui.PlatformDispatcher
	started := make(chan struct{})
	h.cfg.Start = func(disp *ui.PlatformDispatcher) error {
		d = disp
		close(started)
		return nil
	}
	h.c.Inject(metrics(1, 1))
	h.c.Inject(fake.FrameEvent{View: 0})
	done := h.run(t)
	<-started
	hidden := make(chan struct{})
	h.c.Post(func() {
		d.SetWindowHidden(ui.ImplicitViewID, true)
		close(hidden)
	})
	<-hidden
	select {
	case err := <-done:
		t.Fatalf("Run returned (%v) while a hidden window exists", err)
	case <-time.After(50 * time.Millisecond):
	}
	if w := h.c.Windows()[0]; !w.Hidden() || w.Closed() {
		t.Errorf("hidden %v, closed %v; want a hidden window that is still open", w.Hidden(), w.Closed())
	}
	h.c.Post(d.RequestAppExit)
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
}

func TestFocusMovingBetweenWindowsKeepsTheAppResumed(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{Title: "Second"})
	if err != nil {
		t.Fatal(err)
	}
	var changes []ui.AppLifecycleState
	var focusEvents int
	e.d.OnAppLifecycleStateChanged = func(s ui.AppLifecycleState) { changes = append(changes, s) }
	e.d.OnViewFocusChange = func(ui.ViewFocusEvent) {
		focusEvents++
		if len(changes) > 0 && focusEvents == 1 {
			t.Error("the lifecycle changed before the framework heard of the focus")
		}
	}
	e.h.OnViewFocus(ui.ImplicitViewID, true)
	e.h.input.flush()
	// The platform moves the focus to the other window: the old one is told first.
	e.h.OnViewFocus(ui.ImplicitViewID, false)
	e.h.OnViewFocus(v.ViewID(), true)
	e.h.input.flush()
	if want := []ui.AppLifecycleState{ui.AppLifecycleStateResumed}; len(changes) != 1 || changes[0] != want[0] {
		t.Errorf("lifecycle changes %v, want %v and no Inactive in between", changes, want)
	}
	// Another app takes the focus.
	e.h.OnViewFocus(v.ViewID(), false)
	e.h.input.flush()
	if got := e.d.AppLifecycleState(); got != ui.AppLifecycleStateInactive {
		t.Errorf("lifecycle %v with no window focused, want Inactive", got)
	}
}
