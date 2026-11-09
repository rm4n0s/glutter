//go:build darwin && !ios && glutter_display

package macos

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// click sends a press and a release of the left button to the window, as the scripts of the dev events do.
func (s *session) click(w *window, x, y float64) {
	s.onUI(func() {
		w.p.guard(func() {
			w.devMove(x, y)
			w.devButton(x, y, eventTypeLeftMouseDown)
			w.devButton(x, y, eventTypeLeftMouseUp)
			w.p.flushPointers()
		})
	})
}

// viewsOf are the views that the pointer data are for.
func viewsOf(ds []ui.PointerData) map[ui.ViewID]int {
	out := map[ui.ViewID]int{}
	for _, d := range ds {
		out[d.ViewID]++
	}
	return out
}

func (s *session) dialog(parent *window, modal bool) *window {
	return s.newWindow(platform.WindowConfig{
		Kind: platform.WindowKindDialog, Size: ui.Size{Width: 200, Height: 120},
		Parent: parent.id, HasParent: true, Modal: modal,
	})
}

func TestModalWindowBlocksItsParent(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		parent := inputWindow(s, h)
		modal := s.dialog(parent, true)
		s.waitFor("the modal window's first frame", 5*time.Second, func() bool { return h.frameCount() > 1 })
		h.takePointers()

		// A click in the parent is dropped.
		s.click(parent, 50, 40)
		if got := viewsOf(h.takePointers()); got[parent.id] != 0 {
			t.Errorf("the blocked parent got %d pointer events", got[parent.id])
		}
		// The parent cannot be closed by the user: the handler is not even asked.
		var asked int
		s.onUI(func() {
			h.mu.Lock()
			before := h.closeReq
			h.mu.Unlock()
			if parent.shouldClose() {
				t.Error("a window with a modal window over it agreed to close")
			}
			h.mu.Lock()
			asked = h.closeReq - before
			h.mu.Unlock()
		})
		if asked != 0 {
			t.Errorf("the handler was asked %d times to close a blocked window", asked)
		}
		// The modal window takes input.
		s.click(modal, 50, 40)
		if got := viewsOf(h.takePointers()); got[modal.id] == 0 {
			t.Errorf("the modal window got no pointer events: %v", got)
		}
		// Moves, scrolls and keys are dropped too.
		s.onUI(func() {
			parent.p.guard(func() {
				parent.devMove(10, 10)
				if ev, err := scrollEvent(cgScrollEventUnitLine, -1, 0, false, 0); err == nil && ev != 0 {
					parent.view.Send(sel("scrollWheel:"), ev) // the way AppKit delivers it (the dev wheel skips the view)
				}
				parent.p.flushPointers()
			})
		})
		if got := viewsOf(h.takePointers()); got[parent.id] != 0 {
			t.Errorf("the blocked parent got %d events from moves and a wheel", got[parent.id])
		}
		// Closing the modal window frees the parent.
		s.onUI(modal.Close)
		h.takePointers()
		s.click(parent, 50, 40)
		if got := viewsOf(h.takePointers()); got[parent.id] == 0 {
			t.Errorf("the parent got no pointer events after the modal window closed: %v", got)
		}
	})
}

func TestNonModalDialogDoesNotBlock(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		parent := inputWindow(s, h)
		s.dialog(parent, false)
		s.waitFor("the dialog's first frame", 5*time.Second, func() bool { return h.frameCount() > 1 })
		h.takePointers()
		s.click(parent, 50, 40)
		if got := viewsOf(h.takePointers()); got[parent.id] == 0 {
			t.Errorf("a parent of a dialog that is not modal got no pointer events: %v", got)
		}
	})
}

func TestNestedModalWindowsBlockEachOther(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		parent := inputWindow(s, h)
		first := s.dialog(parent, true)
		second := s.dialog(first, true)
		s.waitFor("the windows' first frames", 5*time.Second, func() bool { return h.frameCount() > 2 })
		h.takePointers()
		s.click(parent, 50, 40)
		s.click(first, 50, 40)
		if got := viewsOf(h.takePointers()); got[parent.id]+got[first.id] != 0 {
			t.Errorf("windows under a modal window got events: %v", got)
		}
		s.click(second, 50, 40)
		if got := viewsOf(h.takePointers()); got[second.id] == 0 {
			t.Errorf("the innermost modal window got no events: %v", got)
		}
		// Closing the outer one first: the parent is free, the inner one stays modal over a window that is gone.
		s.onUI(first.Close)
		h.takePointers()
		s.click(parent, 50, 40)
		if got := viewsOf(h.takePointers()); got[parent.id] == 0 {
			t.Errorf("the parent got no events after the first modal window closed: %v", got)
		}
		s.onUI(second.Close)
	})
}

func TestLastWindowClosedKeepsTheAppWhenAsked(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{QuitOnLastWindowClosed: foundation.Some(false)}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		s.onUI(w.Close)
		time.Sleep(1500 * time.Millisecond)
		// The loop is still going: a function that is posted runs, and a new window opens.
		var n int
		s.onUI(func() { n = s.p.windowCount() })
		if n != 0 {
			t.Errorf("%d windows are left", n)
		}
		before := h.frameCount()
		s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}})
		s.waitFor("the frame of a window made after the last one closed", 5*time.Second, func() bool { return h.frameCount() > before })
	})
}

func TestDockReopenBringsBackAMinimizedWindow(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		s.onUI(func() { w.SetState(platform.WindowStateMinimized) })
		s.waitFor("the window to be minimized", 5*time.Second, func() bool {
			var m bool
			s.onUI(func() { m = send[bool](w.ns, "isMiniaturized") })
			return m
		})
		var handled bool
		s.onUI(func() {
			handled = objc.Send[bool](s.p.delegate, sel("applicationShouldHandleReopen:hasVisibleWindows:"), s.p.app, false)
		})
		if handled {
			t.Error("the reopen handler said that AppKit has nothing to do, which makes it open a new document")
		}
		s.waitFor("the window to come back", 5*time.Second, func() bool {
			var m bool
			s.onUI(func() { m = send[bool](w.ns, "isMiniaturized") })
			return !m
		})
	})
}

// mouseEvent makes an event of the left button with the click count, for the tests of the actions of a title bar.
func mouseEvent(w *window, typ uint64, clicks int64) objc.ID {
	return class("NSEvent").Send(sel("mouseEventWithType:location:modifierFlags:timestamp:windowNumber:context:eventNumber:clickCount:pressure:"),
		typ, w.windowPoint(20, 10), uint64(0), w.p.s.mediaTime(), send[int64](w.ns, "windowNumber"), objc.ID(0), int64(0), clicks, float32(1))
}

func TestMoveActionStartsADragFromThePress(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		var dragged atomic.Pointer[window]
		var kind atomic.Uint64
		s.onUI(func() {
			ev := mouseEvent(w, eventTypeLeftMouseDown, 1)
			s.p.testEvent = func() objc.ID { return ev }
			s.p.performDrag = func(dw *window, e objc.ID) {
				kind.Store(send[uint64](e, "type"))
				dragged.Store(dw)
			}
			w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionMove})
		})
		s.waitFor("the move to start", 3*time.Second, func() bool { return dragged.Load() == w })
		if kind.Load() != eventTypeLeftMouseDown {
			t.Errorf("the move got an event of type %d", kind.Load())
		}
		// With no press that is down, there is nothing to move by.
		dragged.Store(nil)
		s.onUI(func() {
			s.p.testEvent = func() objc.ID { return 0 }
			w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionMove})
			s.p.testEvent = func() objc.ID { return mouseEvent(w, eventTypeLeftMouseUp, 1) }
			w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionMove})
		})
		time.Sleep(300 * time.Millisecond)
		if dragged.Load() != nil {
			t.Error("a move started with no press")
		}
	})
}

func TestMoveWhoseButtonIsUpDoesNothing(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		done := make(chan struct{})
		s.p.Post(func() {
			defer close(done)
			// No button is down on this machine: AppKit's move would wait for a release that never comes.
			dragWindow(w, mouseEvent(w, eventTypeLeftMouseDown, 1))
		})
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("dragWindow waited for a button that is not down")
		}
	})
}

func TestDoubleClickOnTheBarFollowsTheSystemSetting(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		state := func() (zoomed, mini bool) {
			s.onUI(func() { zoomed, mini = send[bool](w.ns, "isZoomed"), send[bool](w.ns, "isMiniaturized") })
			return
		}
		act := func(setting string, clicks int64, kind ui.WindowActionKind) {
			s.onUI(func() {
				ev := mouseEvent(w, eventTypeLeftMouseDown, clicks)
				s.p.testEvent = func() objc.ID { return ev }
				s.p.testDoubleClick = func() string { return setting }
				w.PerformWindowAction(ui.WindowAction{Kind: kind})
			})
		}
		// "None": the second click of a double click does nothing.
		act(doubleClickNothing, 2, ui.WindowActionMaximize)
		time.Sleep(500 * time.Millisecond)
		if z, m := state(); z || m {
			t.Errorf("setting None: zoomed %v, minimized %v", z, m)
		}
		// The button is not the double click: a single click zooms whatever the setting says.
		act(doubleClickNothing, 1, ui.WindowActionMaximize)
		s.waitFor("the zoom", 5*time.Second, func() bool { z, _ := state(); return z })
		act(doubleClickNothing, 1, ui.WindowActionRestore)
		s.waitFor("the restore", 5*time.Second, func() bool { z, _ := state(); return !z })
		// "Minimize": the second click minimizes.
		act(doubleClickMinimize, 2, ui.WindowActionMaximize)
		s.waitFor("the minimize", 5*time.Second, func() bool { _, m := state(); return m })
	})
}
