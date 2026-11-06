package android

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// lockHandler records the pointer data and the locks that ended.
type lockHandler struct {
	platform.NopHandler
	data  []ui.PointerData
	ended []ui.ViewID
}

func (h *lockHandler) OnPointer(d []ui.PointerData)      { h.data = append(h.data, d...) }
func (h *lockHandler) OnPointerLockEnded(view ui.ViewID) { h.ended = append(h.ended, view) }
func (h *lockHandler) take() []ui.PointerData            { d := h.data; h.data = nil; return d }
func (h *lockHandler) deltas() (out [][2]float64) {
	for _, d := range h.data {
		if !d.Synthesized && (d.Change == ui.PointerChangeHover || d.Change == ui.PointerChangeMove) {
			out = append(out, [2]float64{d.PhysicalDeltaX, d.PhysicalDeltaY})
		}
	}
	return out
}

// lockHarness is a platform over fakes with the focus, whose handler is called directly.
func lockHarness(t *testing.T, focused bool) (*harness, *lockHandler, *window) {
	t.Helper()
	h := newHarness(t)
	h.b.mu.Lock()
	h.b.focused = focused
	h.b.mu.Unlock()
	lh := &lockHandler{}
	h.p.handler = lh
	return h, lh, h.win.(*window)
}

func TestPointerLockOnAndroid(t *testing.T) {
	mouseAt := func(p *Platform, x, y float64) {
		p.onMotion(motionEvent{ActionMasked: actionHoverMove, Source: sourceMouse, Pointers: []motionPointer{{ToolType: toolTypeMouse, X: x, Y: y}}}, true, 0, 0)
	}
	t.Run("the motion is the delta of a pointer held where the mouse was", func(t *testing.T) {
		h, lh, w := lockHarness(t, true)
		mouseAt(h.p, 120, 80)
		lh.take()
		if err := w.LockPointer(); err != nil {
			t.Fatal(err)
		}
		if got, want := h.host.sentCommands(), []command{{Op: opPointerCapture, A: 1}}; !reflect.DeepEqual(got, want) {
			t.Errorf("commands = %+v, want %+v", got, want)
		}
		lh.take()
		h.p.lockedMotion(actionMove, 4, -2, 0, 10)
		h.p.lockedMotion(actionMove, 1, 1, 0, 11)
		if got, want := lh.deltas(), [][2]float64{{4, -2}, {1, 1}}; !reflect.DeepEqual(got, want) {
			t.Errorf("deltas = %v, want %v", got, want)
		}
		for _, d := range lh.data {
			if d.PhysicalX != 120 || d.PhysicalY != 80 || d.Kind != ui.PointerDeviceKindMouse {
				t.Errorf("%+v is not the mouse at the held place 120,80", d)
			}
		}
	})
	t.Run("a button makes a down and an up at the held place, and the motion between is a move", func(t *testing.T) {
		h, lh, w := lockHarness(t, true)
		mouseAt(h.p, 50, 60)
		if err := w.LockPointer(); err != nil {
			t.Fatal(err)
		}
		lh.take()
		const primary = 1
		h.p.lockedMotion(actionDown, 0, 0, primary, 10)
		h.p.lockedMotion(actionMove, 3, 3, primary, 11)
		h.p.lockedMotion(actionUp, 0, 0, 0, 12)
		var changes []ui.PointerChange
		for _, d := range lh.data {
			changes = append(changes, d.Change)
			if d.PhysicalX != 50 || d.PhysicalY != 60 {
				t.Errorf("%+v is not at the held place 50,60", d)
			}
		}
		want := []ui.PointerChange{ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeUp}
		if !reflect.DeepEqual(changes, want) {
			t.Errorf("changes = %v, want %v", changes, want)
		}
		if got := lh.deltas(); !reflect.DeepEqual(got, [][2]float64{{3, 3}}) {
			t.Errorf("deltas = %v, want the move's 3,3 only", got)
		}
	})
	t.Run("a pointer not seen is held in the middle of the surface", func(t *testing.T) {
		h, lh, w := lockHarness(t, true)
		h.b.mu.Lock()
		h.b.surf.width, h.b.surf.height = 400, 200
		h.b.mu.Unlock()
		if err := w.LockPointer(); err != nil {
			t.Fatal(err)
		}
		if h.p.lock.x != 200 || h.p.lock.y != 100 {
			t.Errorf("held at %v,%v, want 200,100", h.p.lock.x, h.p.lock.y)
		}
		_ = lh
	})
	t.Run("a window with no focus is denied", func(t *testing.T) {
		_, _, w := lockHarness(t, false)
		err := w.LockPointer()
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != ui.TagPointerLockDenied || !appErr.HasRoute("window.LockPointer.PointerLockDenied") {
			t.Errorf("LockPointer = %v, want PointerLockDenied from window.LockPointer", err)
		}
	})
	t.Run("the unlock gives the capture back and does not tell the handler", func(t *testing.T) {
		h, lh, w := lockHarness(t, true)
		_ = w.LockPointer()
		w.UnlockPointer()
		h.p.pointerCaptureChanged(false) // the system's answer to the release
		if got := h.host.sentCommands(); !reflect.DeepEqual(got, []command{{Op: opPointerCapture, A: 1}, {Op: opPointerCapture, A: 0}}) {
			t.Errorf("commands = %+v", got)
		}
		if len(lh.ended) != 0 || h.p.lock != nil {
			t.Errorf("lock = %v, ended = %v: want none", h.p.lock, lh.ended)
		}
	})
	t.Run("the system taking the capture back ends the lock", func(t *testing.T) {
		h, lh, w := lockHarness(t, true)
		_ = w.LockPointer()
		h.p.pointerCaptureChanged(true)
		h.p.pointerCaptureChanged(false)
		if len(lh.ended) != 1 || lh.ended[0] != w.id || h.p.lock != nil {
			t.Errorf("lock = %v, ended = %v: want the window once", h.p.lock, lh.ended)
		}
		h.p.lockedMotion(actionMove, 5, 5, 0, 20)
		if len(lh.deltas()) != 0 {
			t.Errorf("motion after the end made %v", lh.deltas())
		}
	})
	t.Run("a capture that comes after the unlock is given back", func(t *testing.T) {
		h, _, w := lockHarness(t, true)
		_ = w.LockPointer()
		w.UnlockPointer()
		h.p.pointerCaptureChanged(true)
		got := h.host.sentCommands()
		if last := got[len(got)-1]; last != (command{Op: opPointerCapture, A: 0}) {
			t.Errorf("last command = %+v, want the release", last)
		}
	})
}

// TestPointerLockThroughTheBridge sends the captured motion and the system's answer the way the Java side does
// (through the bridge, from another thread), to a running platform.
func TestPointerLockThroughTheBridge(t *testing.T) {
	h := newHarness(t)
	h.b.mu.Lock()
	h.b.focused = true
	h.b.mu.Unlock()
	h.run()
	var err error
	h.sync(func() { err = h.win.(*window).LockPointer() })
	if err != nil {
		t.Fatal(err)
	}
	h.b.pointerCapture(true)
	h.b.capturedMotion(actionMove, 3, 4, 0, 10)
	waitFor(t, "the motion", func() bool { return len(h.h.pointerChanges()) >= 2 })
	var locked bool
	h.b.pointerCapture(false)
	waitFor(t, "the end of the lock", func() bool {
		h.sync(func() { locked = h.p.lock != nil })
		return !locked
	})
}
