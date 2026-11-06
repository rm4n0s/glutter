package widgets_test

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// lockRig is a tester with a Listener over the whole view that records the hover events.
type lockRig struct {
	ts     *gluttertest.Tester
	view   *ui.View
	lock   *PointerLock
	hovers []*gestures.PointerHoverEvent
	ended  int
}

func newLockRig(t *testing.T) *lockRig {
	t.Helper()
	r := &lockRig{ts: gluttertest.New(t)}
	r.ts.PumpWidget(NewListener(NewSizedBoxExpand(nil), ListenerOpts{
		Behavior:       rendering.HitTestBehaviorOpaque,
		OnPointerHover: func(e *gestures.PointerHoverEvent) { r.hovers = append(r.hovers, e) },
	}))
	r.view = r.ts.View
	r.lock = r.ts.Binding.PointerLock()
	r.lock.AddListener(func() { r.ended++ })
	return r
}

func TestPointerLock(t *testing.T) {
	t.Run("lock and unlock", func(t *testing.T) {
		r := newLockRig(t)
		if r.lock.Locked() {
			t.Fatal("locked before Lock")
		}
		if err := r.lock.Lock(r.view); err != nil {
			t.Fatal(err)
		}
		if !r.lock.Locked() || !r.ts.PointerLocked(r.view.ViewID()) {
			t.Errorf("Locked = %v, the platform's %v", r.lock.Locked(), r.ts.PointerLocked(r.view.ViewID()))
		}
		if err := r.lock.Lock(r.view); err != nil {
			t.Errorf("locking a locked view: %v", err)
		}
		r.lock.Unlock()
		if r.lock.Locked() || r.ts.PointerLocked(r.view.ViewID()) {
			t.Errorf("still locked after Unlock")
		}
		if r.ended != 0 {
			t.Errorf("the listener was told %d times of a lock that the app ended", r.ended)
		}
		r.lock.Unlock() // nothing to do
	})
	t.Run("PointerLockOf is the binding's", func(t *testing.T) {
		r := newLockRig(t)
		var got *PointerLock
		r.ts.PumpWidget(NewBuilder(func(bc BuildContext) Widget { got = PointerLockOf(bc); return NewSizedBoxShrink(nil) }))
		if got != r.lock {
			t.Errorf("PointerLockOf gave another lock than the binding's")
		}
	})
	t.Run("the platform fails", func(t *testing.T) {
		tests := []struct {
			name string
			tag  string
		}{
			{"unavailable", ui.TagPointerLockUnavailable},
			{"denied", ui.TagPointerLockDenied},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := newLockRig(t)
				r.ts.FailPointerLock(errors.New(tt.tag, "the platform says no"))
				err := r.lock.Lock(r.view)
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tt.tag {
					t.Fatalf("Lock = %v, want the tag %s", err, tt.tag)
				}
				if r.lock.Locked() {
					t.Errorf("Locked after a failed Lock")
				}
				r.ts.FailPointerLock(nil)
				if err := r.lock.Lock(r.view); err != nil || !r.lock.Locked() {
					t.Errorf("Lock again = %v, locked %v", err, r.lock.Locked())
				}
			})
		}
	})
	t.Run("a lock that the platform ends", func(t *testing.T) {
		r := newLockRig(t)
		if err := r.lock.Lock(r.view); err != nil {
			t.Fatal(err)
		}
		r.ts.EndPointerLock(r.view.ViewID())
		if r.lock.Locked() || r.ended != 1 {
			t.Errorf("Locked = %v, listener told %d times; want unlocked and once", r.lock.Locked(), r.ended)
		}
		r.ts.EndPointerLock(r.view.ViewID()) // already over
		if r.ended != 1 {
			t.Errorf("the listener was told again: %d", r.ended)
		}
		// It does not come back: the app locks again when it wants to.
		if err := r.lock.Lock(r.view); err != nil || !r.ts.PointerLocked(r.view.ViewID()) {
			t.Errorf("Lock again = %v", err)
		}
	})
	t.Run("a removed listener is not told", func(t *testing.T) {
		r := newLockRig(t)
		n := 0
		h := r.lock.AddListener(func() { n++ })
		r.lock.RemoveListener(h)
		if err := r.lock.Lock(r.view); err != nil {
			t.Fatal(err)
		}
		r.ts.EndPointerLock(r.view.ViewID())
		if n != 0 || r.ended != 1 {
			t.Errorf("removed listener told %d times, the other %d", n, r.ended)
		}
	})
	t.Run("a locked mouse keeps its position and carries the motion", func(t *testing.T) {
		r := newLockRig(t)
		at := ui.Offset{Dx: 300, Dy: 200}
		r.ts.Hover(at)
		if err := r.lock.Lock(r.view); err != nil {
			t.Fatal(err)
		}
		r.hovers = nil
		moves := []ui.Offset{{Dx: 5, Dy: -3}, {Dx: -40, Dy: 12}, {Dx: 0.5, Dy: 0}}
		for _, d := range moves {
			r.ts.MoveLockedMouse(d)
		}
		if len(r.hovers) != len(moves) {
			t.Fatalf("%d hover events, want %d", len(r.hovers), len(moves))
		}
		for i, e := range r.hovers {
			if e.Position() != at || e.Delta() != moves[i] {
				t.Errorf("event %d: position %v delta %v, want %v and %v", i, e.Position(), e.Delta(), at, moves[i])
			}
		}
	})
}

func TestPointerLockWithoutAHost(t *testing.T) {
	d := ui.NewPlatformDispatcher(nil)
	err := d.LockPointer(ui.ImplicitViewID)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != ui.TagPointerLockUnavailable || !appErr.HasRoute("PlatformDispatcher.LockPointer.PointerLockUnavailable") {
		t.Fatalf("LockPointer = %v, want PointerLockUnavailable from LockPointer", err)
	}
	d.UnlockPointer(ui.ImplicitViewID) // nothing happens
	called := false
	d.OnPointerLockEnded = func(ui.ViewID) { called = true }
	d.PointerLockEnded(ui.ImplicitViewID)
	if !called {
		t.Errorf("OnPointerLockEnded was not called")
	}
	_ = foundation.DebugMode
}
