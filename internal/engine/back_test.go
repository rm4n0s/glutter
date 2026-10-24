package engine

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// backWindow is a window that wants to know whether the framework handles back.
type backWindow struct {
	platform.Window
	calls []bool
}

func (w *backWindow) SetFrameworkHandlesBack(h bool) { w.calls = append(w.calls, h) }

// TestBackEventsReachTheDispatcherAndTheWindow checks the engine both ways: what the platform reports
// about back and memory reaches the dispatcher's callbacks, and what the framework asks goes to a
// window that can do it, and is ignored by one that cannot.
func TestBackEventsReachTheDispatcherAndTheWindow(t *testing.T) {
	e := newWindowsEngine(t)
	var got []string
	e.d.OnPopRoute = func() { got = append(got, "pop") }
	e.d.OnStartBackGesture = func(ev ui.BackGestureEvent) bool { got = append(got, "start"); return ev.Progress == 0.5 }
	e.d.OnUpdateBackGestureProgress = func(ev ui.BackGestureEvent) { got = append(got, "update") }
	e.d.OnCommitBackGesture = func() { got = append(got, "commit") }
	e.d.OnCancelBackGesture = func() { got = append(got, "cancel") }
	e.d.OnMemoryPressure = func() { got = append(got, "memory") }
	var h platform.BackHandler = e.h
	h.OnPopRoute()
	if !h.OnStartBackGesture(ui.BackGestureEvent{Progress: 0.5}) || h.OnStartBackGesture(ui.BackGestureEvent{}) {
		t.Error("the answer of the framework to the start of the gesture was not given back")
	}
	h.OnUpdateBackGestureProgress(ui.BackGestureEvent{})
	h.OnCommitBackGesture()
	h.OnCancelBackGesture()
	h.OnMemoryPressure()
	want := []string{"pop", "start", "start", "update", "commit", "cancel", "memory"}
	if len(got) != len(want) {
		t.Fatalf("the callbacks were %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the callbacks were %v, want %v", got, want)
		}
	}

	e.d.SetFrameworkHandlesBack(true) // a window that cannot: nothing happens
	w := &backWindow{Window: e.h.windowOf(ui.ImplicitViewID).Window}
	e.h.windowOf(ui.ImplicitViewID).Window = w
	e.d.SetFrameworkHandlesBack(true)
	e.d.SetFrameworkHandlesBack(false)
	if len(w.calls) != 2 || !w.calls[0] || w.calls[1] {
		t.Errorf("the window was told %v", w.calls)
	}
}
