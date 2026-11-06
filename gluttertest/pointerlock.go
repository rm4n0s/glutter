package gluttertest

import (
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
)

// PointerLocked says whether the app locked the pointer to the window of the view and the lock has not ended.
//
// Flutter: no counterpart
func (t *Tester) PointerLocked(view ui.ViewID) bool {
	t.host.Lock()
	defer t.host.Unlock()
	return t.host.LockedViews[view]
}

// FailPointerLock makes the fake platform refuse to lock the pointer with err (ui.TagPointerLockUnavailable or
// ui.TagPointerLockDenied errors, as a platform that cannot or will not); nil makes it lock again.
//
// Flutter: no counterpart
func (t *Tester) FailPointerLock(err error) {
	t.host.Lock()
	defer t.host.Unlock()
	t.host.LockPointerErr = err
}

// EndPointerLock ends the lock of the window of the view as the platform does when the compositor or the browser
// ends it, or the window loses the focus: the app's listeners are told. It does nothing when the view is not locked.
//
// Flutter: no counterpart
func (t *Tester) EndPointerLock(view ui.ViewID) {
	t.host.Lock()
	was := t.host.LockedViews[view]
	if was {
		t.host.LockedViews[view] = false
	}
	t.host.Unlock()
	if was {
		t.d.PointerLockEnded(view)
	}
}

// MoveLockedMouse sends the motion of a mouse whose pointer is locked: a hover (a move while a button is down) that
// keeps the position the mouse had and carries delta, in logical pixels, as the platforms report it. Hover first, to
// put the mouse where the lock began.
//
// Flutter: no counterpart
func (t *Tester) MoveLockedMouse(delta ui.Offset) {
	t.adopt()
	t.tb.Helper()
	g := t.Binding.Gestures()
	o := gestures.PointerEventOpts{TimeStamp: t.Elapsed(), Pointer: t.mousePointer(), Kind: ui.PointerDeviceKindMouse, Position: t.mousePosition, Delta: delta}
	if !t.mouseAdded {
		t.mouseAdded = true
		g.HandlePointerEvent(gestures.NewPointerAddedEvent(o))
	}
	g.HandlePointerEvent(gestures.NewPointerHoverEvent(o))
}
