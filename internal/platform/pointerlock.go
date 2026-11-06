package platform

import "github.com/rm4n0s/glutter/ui"

// PointerLocker is implemented by a Window that can lock the pointer (SPEC §27.6): the cursor is hidden and the
// mouse reports motion without a position. A window that does not implement it, or a platform that cannot, makes the
// engine fail the lock with ui.TagPointerLockUnavailable.
//
// While the lock lasts, the pointer events of the mouse keep the position the pointer had when the lock began
// (PhysicalX and PhysicalY), carry the motion in PhysicalDeltaX and PhysicalDeltaY (unaccelerated where the
// platform has it), and the buttons still make down and up events there.
//
// Flutter: no counterpart
type PointerLocker interface {
	// LockPointer asks for the lock. It fails with ui.TagPointerLockUnavailable (the platform lacks what it needs) or
	// ui.TagPointerLockDenied (it refuses). Platforms that need the compositor's or the browser's answer lock
	// at once from the app's point of view: a refusal comes later as PointerLockHandler.OnPointerLockEnded. Locking
	// a window that is locked is not an error.
	LockPointer() error
	// UnlockPointer ends the lock and shows the cursor again. Nothing happens without a lock. It does not call
	// PointerLockHandler.OnPointerLockEnded.
	UnlockPointer()
}

// PointerLockHandler is implemented by a handler that wants to know when a lock ended by itself: the compositor or
// the browser ended it, or the window lost the focus (SPEC §27.6). A platform calls it, on the UI thread, once for
// each lock that ends this way, and never for an UnlockPointer of the app.
//
// Flutter: no counterpart
type PointerLockHandler interface {
	// OnPointerLockEnded reports that the lock of the view's window ended.
	OnPointerLockEnded(view ui.ViewID)
}
