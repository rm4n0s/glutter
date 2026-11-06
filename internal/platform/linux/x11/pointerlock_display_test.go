//go:build linux && !android && glutter_display

package x11

import (
	"github.com/rm4n0s/errors"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// xdotool drives the X server through the XTest extension, as a user's mouse would.
func xdotool(t *testing.T, args ...string) string {
	t.Helper()
	path, err := exec.LookPath("xdotool")
	if err != nil {
		t.Skip("xdotool is not installed (sudo dnf install xdotool)")
	}
	out, err := exec.Command(path, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("xdotool %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// mouseAt returns where the pointer is on the screen.
func mouseAt(t *testing.T) (x, y int) {
	t.Helper()
	out := xdotool(t, "getmouselocation")
	for _, f := range strings.Fields(out) {
		k, v, ok := strings.Cut(f, ":")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		switch k {
		case "x":
			x = n
		case "y":
			y = n
		}
	}
	return x, y
}

func TestDisplayPointerLockGivesRawMotionAtAFixedPosition(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	locker, ok := e.win.(platform.PointerLocker)
	if !ok {
		t.Fatal("the window cannot lock the pointer")
	}
	xdotool(t, "mousemove", "120", "90")
	waitFor(t, func() bool { _, _, _, b, _ := e.h.snapshot(); return len(b) > 0 }, "the pointer never entered the window")
	px, py := mouseAt(t)
	var err error
	e.onLoop(func() { err = locker.LockPointer() })
	if err != nil {
		t.Fatal(err)
	}
	// The motion of the mouse is reported as deltas, at the position where the lock began.
	_, _, _, before, _ := e.h.snapshot()
	n := len(before)
	xdotool(t, "mousemove_relative", "--", "7", "-3")
	var got ui.PointerData
	waitFor(t, func() bool {
		_, _, _, b, _ := e.h.snapshot()
		for _, batch := range b[n:] {
			for _, d := range batch {
				if d.PhysicalDeltaX != 0 || d.PhysicalDeltaY != 0 {
					got = d
					return true
				}
			}
		}
		return false
	}, "no relative motion arrived")
	if got.PhysicalDeltaX != 7 || got.PhysicalDeltaY != -3 {
		t.Errorf("delta = (%v, %v), want (7, -3)", got.PhysicalDeltaX, got.PhysicalDeltaY)
	}
	if got.PhysicalX != float64(px) || got.PhysicalY != float64(py) {
		t.Errorf("position = (%v, %v), want the lock point (%d, %d)", got.PhysicalX, got.PhysicalY, px, py)
	}
	// Pushing against the edge of the window goes on giving motion: the pointer cannot leave the window.
	xdotool(t, "mousemove_relative", "--", "-900", "0")
	waitFor(t, func() bool {
		_, _, _, b, _ := e.h.snapshot()
		for _, batch := range b[n:] {
			for _, d := range batch {
				if d.PhysicalDeltaX <= -900 {
					return true
				}
			}
		}
		return false
	}, "the motion past the edge was not reported")
	if x, _ := mouseAt(t); x < 0 || x >= 400 {
		t.Errorf("the pointer left the window: x = %d", x)
	}
	// Unlocking puts the pointer back and the handler is not told.
	e.onLoop(func() { locker.UnlockPointer() })
	waitFor(t, func() bool { x, y := mouseAt(t); return x == px && y == py }, "the pointer was not put back where the lock began")
	e.h.mu.Lock()
	ends := len(e.h.lockEnds)
	e.h.mu.Unlock()
	if ends != 0 {
		t.Errorf("the handler was told of a lock that the app ended")
	}
	// A second lock works: the first grab was released.
	e.onLoop(func() { err = locker.LockPointer() })
	if err != nil {
		t.Fatalf("the second lock: %v", err)
	}
	e.onLoop(func() { locker.UnlockPointer() })
}

func TestDisplayPointerLockDeniedWhenAnotherClientHasThePointer(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	locker := e.win.(platform.PointerLocker)
	other, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = other.Close() })
	xdotool(t, "mousemove", "50", "50")
	// A second connection grabs the pointer on the root window.
	status := other.x.grabPointer(other.display, other.root, 0, grabEvents, grabModeAsync, grabModeAsync, 0, 0, 0)
	other.x.flush(other.display)
	if status != grabSuccess {
		t.Fatalf("the test's own grab failed: %d", status)
	}
	var lockErr error
	e.onLoop(func() { lockErr = locker.LockPointer() })
	if appErr, ok := errors.FromError(lockErr); !ok || appErr.Tag != ui.TagPointerLockDenied || !appErr.HasRoute("window.LockPointer.PointerLockDenied") {
		t.Errorf("LockPointer = %v, want PointerLockDenied", lockErr)
	}
}
