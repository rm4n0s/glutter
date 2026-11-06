package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// The tags of the errors of PointerLock.Lock (SPEC §27.6, §27.11).
const (
	// TagPointerLockUnavailable is the tag of the error of a lock that the platform cannot make: Wayland without
	// zwp_pointer_constraints_v1 and zwp_relative_pointer_manager_v1, a platform without windows.
	//
	// Flutter: no counterpart
	TagPointerLockUnavailable = ui.TagPointerLockUnavailable
	// TagPointerLockDenied is the tag of the error of a lock that the platform refuses: the web, outside a user
	// gesture.
	//
	// Flutter: no counterpart
	TagPointerLockDenied = ui.TagPointerLockDenied
)

// pointerLockKey is the key under which a binding keeps its PointerLock.
type pointerLockKey struct{}

// PointerLock locks the pointer to a window, for the mouse look of a game and the orbit of a CAD that never stops at
// the edge of the screen (SPEC §27.6). While the pointer is locked the cursor is hidden, and the pointer events of the
// mouse keep the Position that the pointer had when the lock began and carry the motion in Delta, unaccelerated where
// the platform gives it. The lock ends when the app calls Unlock, when the window loses the focus, when the
// compositor ends it, when the window is closed, and on the web when the user presses Escape; it does not come back
// by itself, and the app locks again when it wants to. Flutter has no such API (SPEC §12.10).
//
// Flutter: no counterpart
type PointerLock struct {
	notifier foundation.ChangeNotifier
	d        *ui.PlatformDispatcher
	locked   map[ui.ViewID]bool
}

// PointerLockOf returns the pointer lock of the app (the one of the binding).
//
// Flutter: no counterpart
func PointerLockOf(bc BuildContext) *PointerLock { return WidgetsBindingOf(bc).PointerLock() }

// PointerLock returns the pointer lock of the binding.
//
// Flutter: no counterpart
func (b *WidgetsBinding) PointerLock() *PointerLock {
	return b.Extension(pointerLockKey{}, func() any {
		p := &PointerLock{d: b.d, locked: map[ui.ViewID]bool{}}
		p.notifier.SetErrorReporter(b.report)
		p.notifier.SetUIThread(b.thread)
		b.d.OnPointerLockEnded = p.ended
		return p
	}).(*PointerLock)
}

// Lock locks the pointer to the window of the view. It fails with PointerLockUnavailable (TagPointerLockUnavailable)
// when the platform cannot, and with PointerLockDenied (TagPointerLockDenied) when it refuses; the web refuses outside
// a user gesture. Locking a locked view is not an error.
//
// Flutter: no counterpart
func (p *PointerLock) Lock(view *ui.View) error {
	if p.locked[view.ViewID()] {
		return nil
	}
	if err := p.d.LockPointer(view.ViewID()); err != nil {
		return err
	}
	p.locked[view.ViewID()] = true
	return nil
}

// Unlock ends the lock of every view. The listeners are not told: the app asked for it.
//
// Flutter: no counterpart
func (p *PointerLock) Unlock() {
	for view := range p.locked {
		delete(p.locked, view)
		p.d.UnlockPointer(view)
	}
}

// Locked says whether the pointer is locked to a window of the app.
//
// Flutter: no counterpart
func (p *PointerLock) Locked() bool { return len(p.locked) > 0 }

// AddListener adds fn, which is called when a lock ends by itself: the window lost the focus, the platform ended it,
// the window was closed. It is not called for Unlock. Remove it with RemoveListener.
//
// Flutter: no counterpart
func (p *PointerLock) AddListener(fn foundation.VoidCallback) foundation.ListenerHandle {
	return p.notifier.AddListener(fn)
}

// RemoveListener removes a listener that AddListener added.
//
// Flutter: no counterpart
func (p *PointerLock) RemoveListener(handle foundation.ListenerHandle) {
	p.notifier.RemoveListener(handle)
}

// ended is what the dispatcher calls when the lock of a view ended by itself.
func (p *PointerLock) ended(view ui.ViewID) {
	if !p.locked[view] {
		return
	}
	delete(p.locked, view)
	p.notifier.NotifyListeners()
}
