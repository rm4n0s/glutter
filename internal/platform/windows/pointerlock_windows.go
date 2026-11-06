//go:build windows

package windows

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// The pointer lock of a window (SPEC §27.6): the mouse cursor is hidden and clipped to one pixel of
// the screen (ClipCursor), and the raw input of the mouse (WM_INPUT) gives the motion, which the
// app sees as the delta of a pointer that stays where it was. The raw input is not accelerated.
// Windows lets a process clip the cursor and register the raw input for one window at a time, so
// only one window of the app is locked at once.

// pointerLock is the state of the lock of one window.
type pointerLock struct {
	w     *window
	trans *lockTranslator
}

var _ platform.PointerLocker = (*window)(nil)

// LockPointer hides the cursor, clips it to the place it is at (the middle of the window when it is
// elsewhere) and asks for the raw input of the mouse. It fails with PointerLockDenied when the
// window is not the active one or another window of the app holds the lock, and with
// PointerLockUnavailable when Windows does not accept the raw input.
//
// Flutter: no counterpart
func (w *window) LockPointer() error {
	p := w.p
	if p.lock != nil {
		if p.lock.w == w {
			return nil
		}
		return errors.New(ui.TagPointerLockDenied, "the pointer is locked to another window of the app")
	}
	if w.closed || !w.shown || w.hidden || !w.activated {
		return errors.New(ui.TagPointerLockDenied, "the window is not the active window", "active", w.activated)
	}
	if !win32.RegisterRawInputDevices([]win32.RawInputDevice{{UsagePage: win32.HIDUsagePageGeneric, Usage: win32.HIDUsageGenericMouse, Target: w.hwnd}}) {
		return errors.New(ui.TagPointerLockUnavailable, "Windows did not accept the request for the raw input of the mouse")
	}
	var cx, cy int32
	over := false
	if pt, ok := win32.GetCursorPos(); ok {
		if c, ok := win32.ScreenToClient(w.hwnd, pt); ok {
			cx, cy, over = c.X, c.Y, win32.WindowFromPoint(pt) == w.hwnd
		}
	}
	x, y := lockPoint(cx, cy, w.clientW, w.clientH, over)
	p.lock = &pointerLock{w: w}
	w.holdCursor(x, y)
	p.lock.trans = newLockTranslator(p.pointers, w.id, float64(x), float64(y))
	win32.SetCursor(0)
	w.flushPointers()
	return nil
}

// UnlockPointer gives the cursor back: it is shown, free to move, and the raw input stops. The
// handler is not told.
//
// Flutter: no counterpart
func (w *window) UnlockPointer() { w.endPointerLock(false) }

// holdCursor puts the cursor on a place of the client area and clips it to that pixel.
func (w *window) holdCursor(x, y int32) {
	pt, ok := win32.ClientToScreen(w.hwnd, win32.Point{X: x, Y: y})
	if !ok {
		return
	}
	win32.SetCursorPos(pt.X, pt.Y)
	win32.ClipCursor(&win32.Rect{Left: pt.X, Top: pt.Y, Right: pt.X + 1, Bottom: pt.Y + 1})
}

// reclip clips the cursor to the held place again, after the window moved or changed its size.
func (w *window) reclip() {
	if l := w.p.lock; l != nil && l.w == w {
		w.holdCursor(int32(l.trans.x), int32(l.trans.y))
	}
}

// endPointerLock ends the lock of the window, if it has one. notify tells the handler that it ended
// by itself (the window lost the focus).
func (w *window) endPointerLock(notify bool) {
	p := w.p
	if p.lock == nil || p.lock.w != w {
		return
	}
	p.lock = nil
	win32.ClipCursor(nil)
	win32.RegisterRawInputDevices([]win32.RawInputDevice{{UsagePage: win32.HIDUsagePageGeneric, Usage: win32.HIDUsageGenericMouse, Flags: win32.RIDEVRemove}})
	if !w.closed && w.pointerOverClient() {
		w.showCursor()
	}
	if h, ok := p.handler.(platform.PointerLockHandler); ok && notify && !w.closed {
		h.OnPointerLockEnded(w.id)
	}
}

// onRawInput handles WM_INPUT: the report of the mouse is the motion of the locked pointer. Input
// that arrives with no lock (a report that was on its way when the lock ended) does nothing.
func (w *window) onRawInput(lparam uintptr) {
	l := w.p.lock
	if l == nil || l.w != w {
		return
	}
	in, ok := win32.GetRawInputMouse(lparam)
	if !ok {
		return
	}
	l.trans.motion(in.Mouse.Flags, in.Mouse.LastX, in.Mouse.LastY)
	w.flushPointers()
}
