//go:build darwin && !ios

package macos

import (
	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// The glue of the modal windows (modal.go), the move of a window by its app-drawn title bar, the double click
// of that bar and the reopening of the app from the Dock (SPEC §29.4).

// blocked says that a modal window is open over this window: it takes no input.
func (w *window) blocked() bool { return w.p.modals.blocked(w.id) }

// openModal makes the new modal window block its parent: the pointer that was down in the parent is
// cancelled, so that nothing it was given is left half done.
func (p *Platform) openModal(w, parent *window) {
	p.modals.open(w.id, parent.id)
	p.ptr.cancelView(parent.id, p.frameTime(p.s.mediaTime()))
	p.flushPointers()
}

// closedModal gives the keys back to the window that the closed modal window blocked, or to the modal
// window that still covers it.
func (p *Platform) closedModal(w *window) {
	parent, ok := p.modals.close(w.id)
	if !ok || p.closed {
		return
	}
	if _, ok := p.windows[parent]; !ok {
		return
	}
	if t, ok := p.windows[p.modals.target(parent)]; ok && t.shown && !t.hidden && !t.miniaturized {
		t.orderFront()
	}
}

// redirectToModal brings the modal window that blocks this window to the front, as a click on a disabled
// window does.
func (w *window) redirectToModal() {
	if t, ok := w.p.windows[w.p.modals.target(w.id)]; ok && t != w && t.shown {
		objcx.WithPool(t.orderFront)
	}
}

// currentEvent is the event that AppKit is handling ([NSApp currentEvent]), or 0.
func (p *Platform) currentEvent() objc.ID {
	if p.testEvent != nil {
		return p.testEvent()
	}
	return send[objc.ID](p.app, "currentEvent")
}

// startMove starts moving the window with the mouse button that is down: AppKit runs the move itself
// (performWindowDragWithEvent:) until the button is up. The framework asks from inside the handler of the
// press, where the call would nest in the handling of the pointer, so the move starts after it, with the event
// kept.
func (w *window) startMove() {
	ev := w.p.currentEvent()
	if ev == 0 {
		return
	}
	if t := send[uint64](ev, "type"); t != eventTypeLeftMouseDown && t != eventTypeLeftMouseDragged {
		return
	}
	objcx.Retain(ev)
	w.p.Post(func() {
		defer objcx.Release(ev)
		if w.closed {
			return
		}
		w.p.performDrag(w, ev)
	})
}

// dragWindow is the move of the window: it returns when the button is up, which the move swallows, so the pointer
// that the framework knows as down is cancelled.
func dragWindow(w *window, ev objc.ID) {
	if send[uint64](class("NSEvent"), "pressedMouseButtons")&1 == 0 {
		return // the button went up before the move could start
	}
	objcx.WithPool(func() { w.ns.Send(sel("performWindowDragWithEvent:"), ev) })
	if w.closed {
		return
	}
	w.p.guard(func() {
		w.p.ptr.cancelView(w.id, w.p.frameTime(w.p.s.mediaTime()))
		w.p.flushPointers()
	})
}

// activate shows the window at its size and position, out of the minimized state, and asks for the focus.
func (w *window) activate() {
	objcx.WithPool(func() {
		if send[bool](w.ns, "isMiniaturized") {
			w.ns.Send(sel("deminiaturize:"), objc.ID(0))
		}
		if send[int64](w.p.app, "activationPolicy") == activationPolicyRegular {
			w.p.app.Send(sel("activateIgnoringOtherApps:"), true)
		}
		if w.shown && !w.hidden {
			w.orderFront()
		}
	})
}

// currentClickCount is the click count of the event that is being handled: 2 for the second click of a
// double click, 0 when the event has no clicks.
func (w *window) currentClickCount() int {
	ev := w.p.currentEvent()
	if ev == 0 {
		return 0
	}
	switch send[uint64](ev, "type") {
	case eventTypeLeftMouseDown, eventTypeLeftMouseUp:
		return int(send[int64](ev, "clickCount"))
	}
	return 0
}

// titleBarAction is the zoom or the minimize that the second click of a double click on the title bar that the
// app draws does, by the system setting "Double-click a window's title bar to" (titleDoubleClick). A maximize
// or restore that is not the second click of a double click is the zoom button's and is not changed.
func (w *window) titleBarAction(asked ui.WindowActionKind) (ui.WindowActionKind, bool) {
	if w.currentClickCount() < 2 {
		return asked, true
	}
	return titleDoubleClick(w.p.doubleClick(), asked)
}

// doubleClick is the system setting of the double click on a title bar (a test replaces it).
func (p *Platform) doubleClick() string {
	if p.testDoubleClick != nil {
		return p.testDoubleClick()
	}
	return p.doubleClickSetting()
}

// doubleClickSetting reads AppleActionOnDoubleClick of the user's defaults, and the older boolean
// AppleMiniaturizeOnDoubleClick; "" when neither is set (the default is the zoom).
func (p *Platform) doubleClickSetting() string {
	defaults := class("NSUserDefaults").Send(sel("standardUserDefaults"))
	key := objcx.NewString("AppleActionOnDoubleClick")
	defer objcx.Release(key)
	if v := objcx.String(defaults.Send(sel("stringForKey:"), key)); v != "" {
		return v
	}
	old := objcx.NewString("AppleMiniaturizeOnDoubleClick")
	defer objcx.Release(old)
	if send[bool](defaults, "boolForKey:", old) {
		return doubleClickMinimize
	}
	return ""
}

// reopen is the click on the Dock icon of the running app: the windows that were minimized or hidden come back
// (macOS's convention); it does nothing for a window that is on the screen.
func (p *Platform) reopen() {
	for _, w := range p.sortedWindows() {
		if w.closed || !w.shown {
			continue
		}
		if w.hidden {
			w.SetHidden(false)
		}
		if send[bool](w.ns, "isMiniaturized") && !w.blocked() {
			objcx.WithPool(func() { w.ns.Send(sel("deminiaturize:"), objc.ID(0)) })
		}
	}
}
