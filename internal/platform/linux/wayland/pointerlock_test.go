//go:build linux && !android

package wayland

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

var (
	constraintsGlobal     = fakecomp.Global{Name: 41, Interface: "zwp_pointer_constraints_v1", Version: 1}
	relativePointerGlobal = fakecomp.Global{Name: 42, Interface: "zwp_relative_pointer_manager_v1", Version: 1}
)

// lockEnv is a platform on a compositor with the two protocols of the pointer lock and the cursor shapes, with the
// pointer inside the window.
func lockEnv(t *testing.T) *keyEnv {
	t.Helper()
	e := startInput(t, capPointer, 0, constraintsGlobal, relativePointerGlobal, cursorShapeGlobal)
	e.srv.PointerEnter(e.surface, 100, 50)
	e.srv.PointerFrame()
	e.batches()
	return e
}

// lockPointer asks for the lock and waits until the compositor has the request.
func (e *keyEnv) lockPointer() error {
	err := e.onLoopErr(func() error { return e.p.windows[0].LockPointer() })
	e.settle()
	return err
}

// grantLock sends "locked" and waits until the platform has taken it.
func (e *keyEnv) grantLock() {
	e.t.Helper()
	e.srv.PointerLocked()
	waitUntil(e.t, func() bool { return e.onLoop(func() bool { return e.p.windows[0].pointerLockActive() }) }, "the lock was not activated")
}

func TestPointerLockLocksAndHidesTheCursor(t *testing.T) {
	e := lockEnv(t)
	if err := e.lockPointer(); err != nil {
		t.Fatal(err)
	}
	if err := e.lockPointer(); err != nil { // locking a locked window
		t.Fatal(err)
	}
	e.settle()
	for _, want := range []string{
		"zwp_relative_pointer_manager_v1.get_relative_pointer pointer=true",
		"zwp_pointer_constraints_v1.lock_pointer surface=true pointer=true region=false lifetime=1", // one shot
	} {
		if got := e.srv.RequestsWith(want); len(got) != 1 {
			t.Errorf("%q: %d requests, want 1; log:\n%v", want, len(got), e.srv.Requests())
		}
	}
	// Not locked until the compositor says so: the cursor is still shown, motion is absolute.
	e.srv.PointerMotion(101, 50)
	e.srv.PointerFrame()
	got := e.batches()
	if len(got) != 1 || got[0][0].PhysicalX != 101 {
		t.Fatalf("motion before the lock was granted: %+v", got)
	}
	before := len(e.srv.RequestsWith("wp_cursor_shape_device_v1.set_shape"))
	e.grantLock()
	// The cursor is hidden: wl_pointer.set_cursor without a surface.
	waitUntil(e.t, func() bool { return len(e.srv.RequestsWith("wl_pointer.set_cursor serial=")) > 0 }, "the cursor was not hidden")
	if after := len(e.srv.RequestsWith("wp_cursor_shape_device_v1.set_shape")); after != before {
		t.Errorf("a cursor shape was set while the pointer was locked")
	}
	// The motion is relative: the unaccelerated one, at the position the pointer had.
	e.srv.RelativeMotion(2, 2, 3, -4)
	e.srv.PointerFrame()
	e.srv.RelativeMotion(0.5, 0, 1, 0)
	e.srv.PointerButton(0x110, true)
	e.srv.PointerFrame()
	got = e.batches()
	if len(got) != 2 || len(got[0]) != 1 || len(got[1]) != 2 {
		t.Fatalf("batches = %+v", got)
	}
	m := got[0][0]
	if m.Change != ui.PointerChangeHover || m.PhysicalX != 101 || m.PhysicalY != 50 || m.PhysicalDeltaX != 3 || m.PhysicalDeltaY != -4 {
		t.Errorf("relative motion = %+v, want a hover at 101,50 with delta 3,-4", m)
	}
	if got[1][0].PhysicalDeltaX != 1 || got[1][1].Change != ui.PointerChangeDown || got[1][1].PhysicalX != 101 {
		t.Errorf("motion and button = %+v", got[1])
	}
	// A pointer that the compositor warps is not followed.
	e.srv.PointerMotion(300, 300)
	e.srv.PointerFrame()
	if got := e.batches(); len(got) != 0 {
		t.Errorf("absolute motion was reported while locked: %+v", got)
	}
}

func TestPointerLockUnlockShowsTheCursor(t *testing.T) {
	e := lockEnv(t)
	if err := e.lockPointer(); err != nil {
		t.Fatal(err)
	}
	e.grantLock()
	hidden := len(e.srv.RequestsWith("wl_pointer.set_cursor serial="))
	shapes := len(e.srv.RequestsWith("wp_cursor_shape_device_v1.set_shape"))
	e.onLoop(func() bool { e.p.windows[0].UnlockPointer(); return true })
	e.settle()
	for _, want := range []string{"zwp_locked_pointer_v1.destroy", "zwp_relative_pointer_v1.destroy"} {
		if n := len(e.srv.RequestsWith(want)); n != 1 {
			t.Errorf("%q: %d requests, want 1", want, n)
		}
	}
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wp_cursor_shape_device_v1.set_shape")) == shapes+1 },
		"the cursor was not shown again")
	_ = hidden
	if ends := e.h.pointerLockEnds(); len(ends) != 0 {
		t.Errorf("the handler was told of a lock that the app ended: %v", ends)
	}
	// Absolute motion is back.
	e.srv.PointerMotion(120, 60)
	e.srv.PointerFrame()
	got := e.batches()
	if len(got) != 1 || got[0][0].PhysicalX != 120 || got[0][0].PhysicalDeltaX != 20 {
		t.Errorf("after the unlock: %+v", got)
	}
	// Relative motion that arrives late is ignored.
	e.srv.RelativeMotion(1, 1, 1, 1)
	e.srv.PointerFrame()
	if got := e.batches(); len(got) != 0 {
		t.Errorf("relative motion after the unlock: %+v", got)
	}
	e.onLoop(func() bool { e.p.windows[0].UnlockPointer(); return true }) // nothing to do
}

func TestPointerLockEndedByTheCompositor(t *testing.T) {
	e := lockEnv(t)
	if err := e.lockPointer(); err != nil {
		t.Fatal(err)
	}
	e.grantLock()
	e.srv.PointerUnlocked() // the focus moved away, or the compositor refused
	waitUntil(t, func() bool { return len(e.h.pointerLockEnds()) == 1 }, "the handler was not told that the lock ended")
	if ends := e.h.pointerLockEnds(); ends[0] != 0 {
		t.Fatalf("the handler was told %v, want the view 0 once", ends)
	}
	if n := len(e.srv.RequestsWith("zwp_locked_pointer_v1.destroy")); n != 1 {
		t.Errorf("%d destroy requests", n)
	}
	// It does not come back by itself: a second lock is a new request.
	if err := e.lockPointer(); err != nil {
		t.Fatal(err)
	}
	e.settle()
	if n := len(e.srv.RequestsWith("zwp_pointer_constraints_v1.lock_pointer")); n != 2 {
		t.Errorf("%d lock requests, want 2", n)
	}
}

func TestPointerLockRefusedAtOnceIsReportedAsEnded(t *testing.T) {
	e := lockEnv(t)
	if err := e.lockPointer(); err != nil {
		t.Fatal(err)
	}
	e.srv.PointerUnlocked() // the lock is never granted
	e.settle()
	waitUntil(t, func() bool { return len(e.h.pointerLockEnds()) == 1 }, "the handler was not told that the lock was never granted")
}

func TestPointerLockGoesWithItsWindow(t *testing.T) {
	e := lockEnv(t)
	if err := e.lockPointer(); err != nil {
		t.Fatal(err)
	}
	e.grantLock()
	e.p.Post(func() { e.p.windows[0].Close() }) // the last window ends the loop: nothing to wait for
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("zwp_locked_pointer_v1.destroy")) == 1 }, "the lock outlived its window")
	if ends := e.h.pointerLockEnds(); len(ends) != 0 {
		t.Errorf("a closed window reported its lock: %v", ends)
	}
}

func TestPointerLockWithoutWhatItNeeds(t *testing.T) {
	tests := []struct {
		name    string
		globals []fakecomp.Global
		caps    uint32
		missing string
	}{
		{"no constraints", []fakecomp.Global{relativePointerGlobal}, capPointer, "zwp_pointer_constraints_v1"},
		{"no relative pointer", []fakecomp.Global{constraintsGlobal}, capPointer, "zwp_relative_pointer_manager_v1"},
		{"no pointer", []fakecomp.Global{constraintsGlobal, relativePointerGlobal}, capTouch, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := startInput(t, tt.caps, 0, tt.globals...)
			err := e.lockPointer()
			assertRoute(t, err, "window.LockPointer.PointerLockUnavailable")
			if tt.missing != "" {
				appErr, _ := errors.FromError(err)
				if got := appErr.Metadata["interface"]; got != tt.missing {
					t.Errorf("interface = %v, want %s", got, tt.missing)
				}
			}
			if n := len(e.srv.RequestsWith("zwp_pointer_constraints_v1.lock_pointer")); n != 0 {
				t.Errorf("a lock was requested")
			}
		})
	}
}
