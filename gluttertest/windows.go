package gluttertest

import (
	"github.com/rm4n0s/glutter/ui"
)

// TestWindow is a window that the app asked the fake platform for.
//
// Flutter: no counterpart
type TestWindow struct {
	// View is the view of the window.
	View ui.ViewID
	// Spec is what the app asked for.
	Spec ui.WindowSpec
	// Title is the last title the app set; the one it asked for until then.
	Title string
	// Size is the last content size the app asked for, in logical pixels.
	Size ui.Size
	// Min and Max are the last limits the app set on the content size; zero is no
	// limit.
	Min, Max ui.Size
	// Destroyed is whether the app destroyed the window.
	Destroyed bool
}

// Windows are the windows that the app asked the platform for with the window
// controllers, in order, destroyed ones included. The implicit view is not one of
// them.
//
// Flutter: no counterpart
func (t *Tester) Windows() []TestWindow {
	t.host.Lock()
	defer t.host.Unlock()
	out := make([]TestWindow, len(t.host.Windows))
	for i, w := range t.host.Windows {
		out[i] = TestWindow{View: w.View, Spec: w.Spec, Title: w.Title, Size: w.Size, Min: w.Min, Max: w.Max, Destroyed: w.Destroyed}
	}
	return out
}

// ViewActions are the actions that were asked of the window of a view (minimize,
// maximize, ...), in order.
//
// Flutter: no counterpart
func (t *Tester) ViewActions(view ui.ViewID) []ui.WindowAction {
	t.host.Lock()
	defer t.host.Unlock()
	var out []ui.WindowAction
	for _, a := range t.host.ViewActions {
		if a.View == view {
			out = append(out, a.Action)
		}
	}
	return out
}

// ViewScenes are the scenes that the view rendered so far.
//
// Flutter: no counterpart
func (t *Tester) ViewScenes(view ui.ViewID) []*ui.Scene { return t.host.ScenesOf(view) }

// FocusRequests are the views that the app asked the platform to focus, in order.
//
// Flutter: no counterpart
func (t *Tester) FocusRequests() []ui.ViewID {
	t.host.Lock()
	defer t.host.Unlock()
	var out []ui.ViewID
	for _, e := range t.host.FocusEvents {
		if e.State == ui.ViewFocusStateFocused {
			out = append(out, e.ViewID)
		}
	}
	return out
}

// FocusEvents are the focus changes that the app asked the platform for, in order,
// with the direction: what Flutter's TestPlatformDispatcher.testFocusEvents has.
//
// Flutter: flutter_test/lib/src/platform.dart TestPlatformDispatcher.testFocusEvents
func (t *Tester) FocusEvents() []ui.ViewFocusEvent {
	t.host.Lock()
	defer t.host.Unlock()
	return append([]ui.ViewFocusEvent(nil), t.host.FocusEvents...)
}

// ResetFocusEvents forgets the focus changes that FocusEvents has.
//
// Flutter: flutter_test/lib/src/platform.dart TestPlatformDispatcher.resetFocusedViewTestValues
func (t *Tester) ResetFocusEvents() {
	t.host.Lock()
	defer t.host.Unlock()
	t.host.FocusEvents = nil
}

// SetWindowCreateError makes the platform fail to create windows with the error,
// or, with nil, create them again.
//
// Flutter: no counterpart
func (t *Tester) SetWindowCreateError(err error) {
	t.host.Lock()
	defer t.host.Unlock()
	t.host.CreateErr = err
}

// SetWindowSize tells the framework, as the platform would, that the content of
// the window of the view now has the size, in logical pixels.
//
// Flutter: no counterpart
func (t *Tester) SetWindowSize(view ui.ViewID, size ui.Size) {
	t.adopt()
	v := t.d.View(view)
	if v == nil {
		t.tb.Fatalf("there is no view %d", view)
	}
	dpr := v.DevicePixelRatio()
	t.d.UpdateViewConfiguration(view, ui.ViewConfiguration{
		PhysicalSize: ui.Size{Width: size.Width * dpr, Height: size.Height * dpr}, DevicePixelRatio: dpr, Window: v.Window(),
	})
}

// SetWindowInfo tells the framework, as the platform would, what it now knows of
// the window of the view: who draws the title bar, and whether the window is
// maximized, fullscreen or activated.
//
// Flutter: no counterpart
func (t *Tester) SetWindowInfo(view ui.ViewID, info ui.WindowInfo) {
	t.adopt()
	v := t.d.View(view)
	if v == nil {
		t.tb.Fatalf("there is no view %d", view)
	}
	t.d.UpdateViewConfiguration(view, ui.ViewConfiguration{
		PhysicalSize: v.PhysicalSize(), DevicePixelRatio: v.DevicePixelRatio(), Window: info,
	})
}

// RequestWindowClose tells the framework, as the platform would, that the user
// asked to close the window of the view. It reports whether the platform would
// close the window: false when the delegate of the controller took the request,
// or the app answered the request to exit with a cancel.
//
// Flutter: no counterpart
func (t *Tester) RequestWindowClose(view ui.ViewID) bool {
	t.adopt()
	return t.d.RequestClose(view)
}

// SetViewFocus tells the framework, as the platform would, that the keyboard focus
// moved into the view (true), or out of it, from the direction (undefined by
// default).
//
// Flutter: no counterpart
func (t *Tester) SetViewFocus(view ui.ViewID, focused bool, direction ...ui.ViewFocusDirection) {
	t.adopt()
	state := ui.ViewFocusStateUnfocused
	if focused {
		state = ui.ViewFocusStateFocused
	}
	event := ui.ViewFocusEvent{ViewID: view, State: state}
	if len(direction) > 0 {
		event.Direction = direction[0]
	}
	t.d.DispatchViewFocus(event)
}

// Drop plays a drag and drop operation of another application (or of the
// platform) over a view, and returns the action the framework accepts: the way the
// platform delivers the enter, move, leave and drop events. The position is in
// logical pixels of the view; the event has it in physical pixels.
//
// Flutter: no counterpart
func (t *Tester) Drop(view ui.ViewID, phase ui.DropPhase, at ui.Offset, event ui.DropEvent) ui.DropActions {
	t.tb.Helper()
	event.ViewID, event.Phase = view, phase
	dpr := 1.0
	if v := t.d.View(view); v != nil {
		dpr = v.DevicePixelRatio()
	}
	event.Position = at.Mul(dpr)
	return t.d.DispatchDrop(event)
}

// Drags are the drag and drop operations the app started, in order.
//
// Flutter: no counterpart
func (t *Tester) Drags() []ui.DragRequest {
	t.host.Lock()
	defer t.host.Unlock()
	return append([]ui.DragRequest(nil), t.host.Drags...)
}

// FailDrags makes the fake platform refuse to start drags with err, as a platform
// that cannot.
//
// Flutter: no counterpart
func (t *Tester) FailDrags(err error) {
	t.host.Lock()
	defer t.host.Unlock()
	t.host.DragErr = err
}
