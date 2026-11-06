package ui

import "github.com/rm4n0s/errors"

// The tags of the errors of the pointer lock (SPEC §27.6, §27.11). The platforms make them; widgets.PointerLock
// returns them.
const (
	// TagPointerLockUnavailable is the tag of the error of a lock that the platform cannot make: Wayland without
	// zwp_pointer_constraints_v1 and zwp_relative_pointer_manager_v1, a host without windows, a window that does not
	// exist.
	//
	// Flutter: no counterpart
	TagPointerLockUnavailable = "PointerLockUnavailable"
	// TagPointerLockDenied is the tag of the error of a lock that the platform refuses: the web, outside a user
	// gesture.
	//
	// Flutter: no counterpart
	TagPointerLockDenied = "PointerLockDenied"
)

// PointerLockHost is what the engine implements to lock the pointer to a window (SPEC §27.6). A host without it
// cannot.
//
// Flutter: no counterpart
type PointerLockHost interface {
	// LockPointer locks the pointer to the window of the view: the cursor is hidden, and the mouse's pointer events
	// keep their position and carry the motion in PhysicalDeltaX and PhysicalDeltaY. It fails with
	// PointerLockUnavailable and PointerLockDenied. The lock ends through PlatformDispatcher.PointerLockEnded.
	LockPointer(view ViewID) error
	// UnlockPointer ends the lock of the view's window. Nothing happens when it is not locked.
	UnlockPointer(view ViewID)
}

// LockPointer locks the pointer to the window of the view (SPEC §27.6). It fails with PointerLockUnavailable when
// the host or the platform cannot, and with PointerLockDenied when the platform refuses.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) LockPointer(view ViewID) error {
	if h, ok := d.host.(PointerLockHost); ok {
		return h.LockPointer(view)
	}
	return errors.New(TagPointerLockUnavailable, "the platform cannot lock the pointer", "view", view)
}

// UnlockPointer ends the pointer lock of the window of the view.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) UnlockPointer(view ViewID) {
	if h, ok := d.host.(PointerLockHost); ok {
		h.UnlockPointer(view)
	}
}

// PointerLockEnded is called by the engine when the lock of a view ended without the app asking: the window lost
// the focus, the compositor or the browser ended it, or the window was closed. It calls OnPointerLockEnded.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) PointerLockEnded(view ViewID) {
	if d.OnPointerLockEnded != nil {
		d.OnPointerLockEnded(view)
	}
}
