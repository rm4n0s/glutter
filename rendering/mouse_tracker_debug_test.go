//go:build !glutter_release && !glutter_profile

package rendering

import (
	"testing"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// The debug-only API of the tracker.

func TestMouseTrackerCursor(t *testing.T) {
	h := newTrackerHarness()
	h.find = func(ui.Offset) []testEntry { return nil }
	h.send(mouseEv("add", ui.Offset{}))
	if c, ok := h.tracker.DebugDeviceActiveCursor(0); !ok || c != services.MouseCursorUncontrolled {
		t.Errorf("active cursor = %v, %v; want the fallback", c, ok)
	}
	h.send(mouseEv("remove", ui.Offset{}))
	if _, ok := h.tracker.DebugDeviceActiveCursor(0); ok {
		t.Error("the removed device has a cursor")
	}
}

func TestMouseTrackerErrors(t *testing.T) {
	for _, tc := range []struct {
		name  string
		route string
		fn    func(h *trackerHarness)
	}{
		{"updated while updating", "deviceUpdatePhase.MouseTrackerReentered", func(h *trackerHarness) {
			h.tracker.deviceUpdatePhase(func() { h.tracker.UpdateAllDevices() })
		}},
		{"an added event for a tracked device", "shouldMarkStateDirty.MouseTrackerEventOrder", func(h *trackerHarness) {
			h.send(mouseEv("add", ui.Offset{}))
			h.send(mouseEv("add", off(1, 1)))
		}},
		{"a device that changes", "replaceLatestEvent.MouseTrackerDeviceChanged", func(h *trackerHarness) {
			s := &mouseState{annotations: newAnnotationMap(), latestEvent: mouseEv("add", ui.Offset{})}
			s.replaceLatestEvent(evAt("hover", ui.Offset{}, 1, ui.PointerDeviceKindMouse))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newTrackerHarness()
			err := recoverError(t, func() { tc.fn(h) })
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute(tc.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

func recoverError(t *testing.T, fn func()) (err error) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic")
		}
		e, ok := r.(error)
		if !ok {
			t.Fatalf("expected an error panic, got %T (%v)", r, r)
		}
		err = e
	}()
	fn()
	return nil
}
