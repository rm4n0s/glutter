//go:build linux && !android

package x11

import (
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
)

// xiRawMotion is XI_RawMotion: the motion of a device as the device reports it, which XInput2 delivers to the root
// window of the client that asked for it, even while the pointer is grabbed or clamped to the window.
const xiRawMotion = 17

// The values of XGrabPointer that the lock uses.
const (
	grabModeAsync = 1
	grabSuccess   = 0
	// grabEvents are the core events of the grab (ButtonPress, ButtonRelease, PointerMotion); the XInput2 events of
	// the window come through the selection of the window, as owner_events says.
	grabEvents = 1<<2 | 1<<3 | 1<<6
)

// pointerLock is the pointer lock of a window (SPEC §27.6): the pointer is grabbed and confined to the window with
// no cursor, and the raw motion of the devices gives the relative motion. The pointer moves inside the window, but
// the app sees it where it was when the lock began; the pointer is warped back there at the end.
type pointerLock struct {
	w *window
	// x and y are where the pointer was, in window coordinates.
	x, y float64
}

var _ platform.PointerLocker = (*window)(nil)

// LockPointer grabs the pointer with XGrabPointer, confined to the window, with the empty cursor, and asks for the
// raw motion of the devices (XI_RawMotion) on the root window. It fails with PointerLockUnavailable without
// XInput2, and with PointerLockDenied when the grab does not succeed (another client holds the pointer, the window
// is not on the screen).
//
// Flutter: no counterpart
func (w *window) LockPointer() error {
	in := w.p.input
	if in == nil {
		return errors.New(ui.TagPointerLockUnavailable, "the X server has no XInput2, which gives the raw motion that a lock needs")
	}
	if in.lock != nil {
		if in.lock.w == w {
			return nil
		}
		return errors.New(ui.TagPointerLockDenied, "the pointer is locked to another window of the app")
	}
	cursor := w.p.cursors.hidden()
	status := w.p.x.grabPointer(w.p.display, w.xid, 1, grabEvents, grabModeAsync, grabModeAsync, w.xid, cursor, 0)
	w.p.x.flush(w.p.display)
	if status != grabSuccess {
		return errors.New(ui.TagPointerLockDenied, "the X server did not give the pointer grab", "status", int(status))
	}
	in.lock = &pointerLock{w: w, x: in.lastX, y: in.lastY}
	in.selectEventsFrom(xiAllMasterDevices, w.p.root, xiRawMotion)
	in.pointer.SetLocked(true)
	w.p.x.flush(w.p.display)
	return nil
}

// UnlockPointer ends the grab, stops the raw motion, puts the pointer back where the lock began and shows the cursor
// again. The handler is not told.
//
// Flutter: no counterpart
func (w *window) UnlockPointer() { w.p.input.endLock(w, false) }

// endLock ends the lock of the window w, if it has one. notify tells the handler that it ended by itself.
func (in *inputState) endLock(w *window, notify bool) {
	if in == nil || in.lock == nil || in.lock.w != w {
		return
	}
	l := in.lock
	in.lock = nil
	in.pointer.SetLocked(false)
	p := in.p
	in.selectEventsFrom(xiAllMasterDevices, p.root) // no events: the mask is empty
	if !w.closed {
		p.x.warpPointer(p.display, 0, w.xid, 0, 0, 0, 0, int32(l.x), int32(l.y))
	}
	p.x.ungrabPointer(p.display, 0)
	p.x.flush(p.display)
	if notify && !w.closed {
		if h, ok := p.handler.(platform.PointerLockHandler); ok {
			h.OnPointerLockEnded(w.id)
		}
	}
}

// onRawMotion reports the raw motion of a device while the pointer is locked: the first two valuators are x and y.
func (in *inputState) onRawMotion(data uintptr) {
	if in.lock == nil {
		return
	}
	ev := view[xiRawEvent](data)
	v := &ev.valuators
	if v.maskLen == 0 || v.mask == 0 || ev.rawValues == 0 {
		return
	}
	mask := dl.Bytes(v.mask, int(v.maskLen))
	values := unsafe.Slice(view[float64](ev.rawValues), int(v.maskLen)*8)
	var dx, dy float64
	index := 0
	for number := range 2 {
		if mask[number>>3]&(1<<(number&7)) == 0 {
			continue
		}
		if number == 0 {
			dx = values[index]
		} else {
			dy = values[index]
		}
		index++
	}
	if dx == 0 && dy == 0 {
		return
	}
	scale := in.pointer.Scale
	if scale <= 0 {
		scale = 1
	}
	in.pointer.Relative(dx/scale, dy/scale)
	in.flushPointer()
}
