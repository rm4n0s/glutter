//go:build linux && !android

package wayland

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// pointerLock is the pointer lock of a window (SPEC §27.6): a zwp_locked_pointer_v1 of the surface and the
// zwp_relative_pointer_v1 that gives the motion while the pointer cannot move.
type pointerLock struct {
	locked   *wl.ZwpLockedPointerV1
	relative *wl.ZwpRelativePointerV1
	// active is whether the compositor said "locked": from then until the lock ends the cursor is hidden and the
	// batcher reports relative motion. Before it, the lock is only asked for; the compositor activates it when the
	// pointer is in the surface.
	active bool
}

var _ platform.PointerLocker = (*window)(nil)

// LockPointer asks the compositor to lock the pointer to the window (zwp_pointer_constraints_v1.lock_pointer, one
// shot: the lock does not come back after it ended) and to send the unaccelerated motion
// (zwp_relative_pointer_v1). It fails with PointerLockUnavailable when the compositor lacks either protocol or the
// seat has no pointer.
//
// Flutter: no counterpart
func (w *window) LockPointer() error {
	if w.lock != nil {
		return nil
	}
	sess := w.p.sess
	pt := w.p.pointer
	switch {
	case w.closed:
		return errors.New(ui.TagPointerLockUnavailable, "the window is closed")
	case sess.pointerConstraints == nil:
		return errors.New(ui.TagPointerLockUnavailable, "the compositor has no zwp_pointer_constraints_v1",
			"interface", wl.ZwpPointerConstraintsV1Interface.Name)
	case sess.relativePointer == nil:
		return errors.New(ui.TagPointerLockUnavailable, "the compositor has no zwp_relative_pointer_manager_v1",
			"interface", wl.ZwpRelativePointerManagerV1Interface.Name)
	case pt == nil:
		return errors.New(ui.TagPointerLockUnavailable, "the seat has no pointer")
	}
	l := &pointerLock{}
	l.relative = sess.relativePointer.GetRelativePointer(pt.obj)
	l.relative.SetEvents(&wl.ZwpRelativePointerV1Events{
		OnRelativeMotion: func(_, _ uint32, _, _ float64, dxUnaccel, dyUnaccel float64) {
			if l.active && w.lock == l {
				pt.b.Relative(dxUnaccel, dyUnaccel)
				pt.eventDone()
			}
		},
	})
	l.locked = sess.pointerConstraints.LockPointer(w.surface, pt.obj, nil, wl.ZwpPointerConstraintsV1LifetimeOneshot)
	l.locked.SetEvents(&wl.ZwpLockedPointerV1Events{
		OnLocked: func() {
			if w.lock != l {
				return
			}
			l.active = true
			pt.b.SetLocked(true)
			if pt.focus == w {
				w.p.cursors.apply(pt, w.cursor) // hides the cursor while the lock is active
			}
		},
		OnUnlocked: func() {
			if w.lock == l {
				w.dropPointerLock(true) // the compositor ended it (or never granted it)
			}
		},
	})
	w.lock = l
	return nil
}

// UnlockPointer ends the lock and shows the cursor again. The handler is not told.
//
// Flutter: no counterpart
func (w *window) UnlockPointer() { w.dropPointerLock(false) }

// pointerLockActive says whether the compositor locked the pointer to the window: the cursor is hidden then.
func (w *window) pointerLockActive() bool { return w.lock != nil && w.lock.active }

// dropPointerLock destroys the lock objects and puts the pointer back to normal. notify tells the handler that the
// lock ended by itself (the compositor ended it, the pointer went away, the window was hidden).
func (w *window) dropPointerLock(notify bool) {
	l := w.lock
	if l == nil {
		return
	}
	w.lock = nil
	wasActive := l.active
	l.active = false
	if l.relative != nil {
		l.relative.Destroy()
	}
	if l.locked != nil {
		l.locked.Destroy()
	}
	if pt := w.p.pointer; pt != nil {
		if wasActive {
			pt.b.SetLocked(false)
		}
		if wasActive && pt.focus == w {
			w.p.cursors.apply(pt, w.cursor)
		}
	}
	if notify && !w.closed {
		if h, ok := w.p.handler.(platform.PointerLockHandler); ok {
			h.OnPointerLockEnded(w.id)
		}
	}
}
