package windows

import "github.com/rm4n0s/glutter/ui"

// This file is the plain-Go part of the pointer lock (SPEC §27.6): the raw reports of the mouse
// turned into the pointer data of a locked pointer. It takes the numbers of the reports and calls
// nothing of Windows, so a test runs it on any host. The Windows side (RegisterRawInputDevices,
// ClipCursor, the hidden cursor) is pointerlock_windows.go.

// relative reports motion of a locked mouse: a hover (or a move, while a button is down) that stays
// at the place the pointer had when the lock began and carries the motion as its delta.
func (t *pointerTracker) relative(view ui.ViewID, x, y, dx, dy float64) {
	st := t.stateFor(view, ui.PointerDeviceKindMouse, mouseDevice)
	change := ui.PointerChangeHover
	if st.down {
		change = ui.PointerChangeMove
	}
	t.send(st, change, x, y, func(d *ui.PointerData) {
		d.PhysicalDeltaX, d.PhysicalDeltaY = dx, dy
	})
}

// lockTranslator is the state of the pointer lock of one window: where the pointer is held, and
// what the raw reports of the mouse make of it.
type lockTranslator struct {
	tracker *pointerTracker
	view    ui.ViewID
	// x and y are the place of the pointer, in physical pixels of the client area.
	x, y float64
}

// newLockTranslator makes the translator of a lock that holds the pointer of the view at a place.
// The pointer is moved there first, with no motion, so that the app sees it where the cursor is held
// and a jump to the held place is not mistaken for motion of the mouse.
func newLockTranslator(tracker *pointerTracker, view ui.ViewID, x, y float64) *lockTranslator {
	tracker.relative(view, x, y, 0, 0)
	return &lockTranslator{tracker: tracker, view: view, x: x, y: y}
}

// motion handles a raw report of the mouse: relative motion is the delta of a hover at the held
// place. A report with no motion, and an absolute one (a remote desktop, a pen that acts as a
// mouse: its X and Y are a place, not motion), make nothing.
func (l *lockTranslator) motion(flags uint16, dx, dy int32) {
	if flags&mouseMoveAbsolute != 0 || (dx == 0 && dy == 0) {
		return
	}
	l.tracker.relative(l.view, l.x, l.y, float64(dx), float64(dy))
}

// mouseMoveAbsolute is MOUSE_MOVE_ABSOLUTE, the flag of a raw report whose X and Y are a place.
const mouseMoveAbsolute = 0x01

// lockPoint chooses the place a lock holds the pointer at: where the cursor is when it is over the
// client area (of w by h pixels), else the middle of the client area.
func lockPoint(cx, cy, w, h int32, over bool) (x, y int32) {
	if over && cx >= 0 && cy >= 0 && cx < w && cy < h {
		return cx, cy
	}
	return w / 2, h / 2
}
