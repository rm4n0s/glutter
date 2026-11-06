//go:build windows && glutter_wine

package windows

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests lock the pointer of a window under Wine (SPEC §27.6): the motion that SendInput makes
// comes as the delta of a pointer that stays where it was, the cursor cannot leave its place, and
// the lock ends with the activation of the window.

// relativeMove makes one relative motion of the mouse for SendInput.
func relativeMove(dx, dy int32) win32.Input {
	return win32.Input{Type: win32.InputMouse, Mouse: win32.MouseInput{DX: dx, DY: dy, Flags: win32.MouseEventFMove}}
}

func TestPointerLockGivesRelativeMotionAndHoldsTheCursor(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	var held, after win32.Point
	pointerSteps(t, p, r, 120*time.Millisecond,
		func() { moveCursorTo(t, w, 100, 90) },
		func() {
			if err := w.LockPointer(); err != nil {
				t.Fatalf("LockPointer: %v", err)
			}
			held, _ = win32.GetCursorPos()
			r.pointers = nil
		},
		func() { win32.SendInput([]win32.Input{relativeMove(7, -3)}) },
		func() { win32.SendInput([]win32.Input{relativeMove(5, 5)}) },
		func() { after, _ = win32.GetCursorPos() },
		func() {
			w.UnlockPointer()
			pt, _ := win32.ClientToScreen(w.hwnd, win32.Point{X: 200, Y: 150})
			win32.SetCursorPos(pt.X, pt.Y)
			if got, _ := win32.GetCursorPos(); got != pt {
				t.Errorf("after the unlock the cursor is at %+v, want it free to go to %+v", got, pt)
			}
		},
		func() {},
	)
	if after != held {
		t.Errorf("the cursor moved from %+v to %+v while the pointer was locked", held, after)
	}
	// Wine's X11 driver reports the first motion after a clip with more on X than SendInput gave
	// (7 comes as 14); the second one is exact. The sign and the place are what the lock promises.
	var motion []ui.PointerData
	for _, d := range r.pointers {
		if d.PhysicalX == 100 && d.PhysicalY == 90 && (d.PhysicalDeltaX != 0 || d.PhysicalDeltaY != 0) {
			motion = append(motion, d)
		}
	}
	if len(motion) != 2 {
		t.Fatalf("%d reports of motion at the held place, want 2: %v", len(motion), r.pointers)
	}
	if first := motion[0]; first.PhysicalDeltaX < 7 || first.PhysicalDeltaY != -3 {
		t.Errorf("first motion = %v,%v, want x of at least 7 and y of -3", first.PhysicalDeltaX, first.PhysicalDeltaY)
	}
	if second := motion[1]; second.PhysicalDeltaX != 5 || second.PhysicalDeltaY != 5 {
		t.Errorf("second motion = %v,%v, want 5,5", second.PhysicalDeltaX, second.PhysicalDeltaY)
	}
	if len(r.lockEnds) != 0 {
		t.Errorf("an unlock of the app told the handler: %v", r.lockEnds)
	}
}

func TestPointerLockEndsWhenTheWindowIsDeactivated(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	pointerSteps(t, p, r, 120*time.Millisecond,
		func() { moveCursorTo(t, w, 100, 90) },
		func() {
			if err := w.LockPointer(); err != nil {
				t.Fatalf("LockPointer: %v", err)
			}
		},
		func() { win32.SendMessage(w.hwnd, win32.WMActivate, win32.WAInactive, 0) },
		func() {
			if p.lock != nil {
				t.Error("the lock is still held after the window was deactivated")
			}
		},
		func() {},
	)
	if len(r.lockEnds) != 1 || r.lockEnds[0] != w.id {
		t.Errorf("the handler was told %v, want the window %v once", r.lockEnds, w.id)
	}
}

func TestPointerLockIsDeniedToAWindowThatIsNotActive(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	var lockErr error
	pointerSteps(t, p, r, 100*time.Millisecond,
		func() {
			w.activated = false
			lockErr = w.LockPointer()
		},
		func() {},
	)
	appErr, ok := errors.FromError(lockErr)
	if !ok || appErr.Tag != ui.TagPointerLockDenied || !appErr.HasRoute("window.LockPointer.PointerLockDenied") {
		t.Errorf("LockPointer = %v, want PointerLockDenied from window.LockPointer", lockErr)
	}
}
