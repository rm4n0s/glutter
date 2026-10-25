//go:build !glutter_release && !glutter_profile

package rendering

import (
	"testing"

	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// The cursor of a pressed pointer (step 27.7): a region that holds the cursor keeps it for
// a mouse that went down inside it until the button is released, whatever the mouse moves
// over.

var (
	sashCursor  services.MouseCursor = services.SystemMouseCursors.ResizeLeftRight
	otherCursor services.MouseCursor = services.SystemMouseCursors.Click
)

// holdLayout has a sash at x 0 to 10 and another region at 10 to 20; beyond that there is
// nothing under the pointer.
func holdLayout(sash, other *testAnnotationTarget) *trackerHarness {
	h := newTrackerHarness()
	h.find = func(p ui.Offset) []testEntry {
		switch {
		case p.Dx < 10:
			return []testEntry{{target: sash}}
		case p.Dx < 20:
			return []testEntry{{target: other}}
		}
		return nil
	}
	return h
}

func at(x float64) ui.Offset { return ui.Offset{Dx: x, Dy: 5} }

func activeCursor(t *testing.T, h *trackerHarness, device int) services.MouseCursor {
	t.Helper()
	c, ok := h.tracker.DebugDeviceActiveCursor(device)
	if !ok {
		return nil
	}
	return c
}

func TestMouseTrackerHoldsTheCursorOfAPressedPointer(t *testing.T) {
	type step struct {
		event string
		x     float64
		want  services.MouseCursor // the cursor after the step
	}
	tests := []struct {
		name  string
		hold  bool
		steps []step
	}{
		{"press in the sash, move over another region and over nothing, release", true, []step{
			{"add", 5, sashCursor},
			{"down", 5, sashCursor},
			{"move", 15, sashCursor},
			{"move", 30, sashCursor},
			{"up", 30, services.MouseCursorUncontrolled},
			{"hover", 15, otherCursor},
		}},
		{"release over another region", true, []step{
			{"add", 5, sashCursor},
			{"down", 5, sashCursor},
			{"move", 15, sashCursor},
			{"up", 15, otherCursor},
		}},
		{"without the option the cursor follows the regions", false, []step{
			{"add", 5, sashCursor},
			{"down", 5, sashCursor},
			{"move", 15, otherCursor},
			{"move", 30, services.MouseCursorUncontrolled},
			{"up", 30, services.MouseCursorUncontrolled},
		}},
		{"a press elsewhere holds nothing, also when the pointer goes over the sash", true, []step{
			{"add", 15, otherCursor},
			{"down", 15, otherCursor},
			{"move", 5, sashCursor},
			{"move", 30, services.MouseCursorUncontrolled},
			{"up", 30, services.MouseCursorUncontrolled},
		}},
		{"a cancel ends the hold", true, []step{
			{"add", 5, sashCursor},
			{"down", 5, sashCursor},
			{"move", 30, sashCursor},
			{"cancel", 30, services.MouseCursorUncontrolled},
		}},
		{"the next press holds again", true, []step{
			{"add", 5, sashCursor},
			{"down", 5, sashCursor},
			{"move", 30, sashCursor},
			{"up", 30, services.MouseCursorUncontrolled},
			{"hover", 5, sashCursor},
			{"down", 5, sashCursor},
			{"move", 15, sashCursor},
			{"up", 15, otherCursor},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sash := &testAnnotationTarget{cursor: sashCursor, hold: tt.hold}
			other := &testAnnotationTarget{cursor: otherCursor}
			h := holdLayout(sash, other)
			for i, s := range tt.steps {
				var e gestures.PointerEvent
				if s.event == "cancel" {
					e = gestures.NewPointerCancelEvent(gestures.PointerEventOpts{Position: at(s.x), Kind: ui.PointerDeviceKindMouse})
				} else {
					e = evAt(s.event, at(s.x), 0, ui.PointerDeviceKindMouse)
				}
				h.send(e)
				if got := activeCursor(t, h, 0); got != s.want {
					t.Fatalf("step %d (%s at %v): cursor = %v, want %v", i, s.event, s.x, got, s.want)
				}
			}
		})
	}
}

func TestMouseTrackerHoldFollowsTheRegion(t *testing.T) {
	sash := &testAnnotationTarget{cursor: sashCursor, hold: true}
	other := &testAnnotationTarget{cursor: otherCursor}
	h := holdLayout(sash, other)
	h.send(mouseEv("add", at(5)))
	h.send(mouseEv("down", at(5)))
	h.send(mouseEv("move", at(30)))

	// The cursor of the region changes during the hold.
	sash.cursor = services.SystemMouseCursors.ResizeUpDown
	h.tracker.UpdateAllDevices()
	if got := activeCursor(t, h, 0); got != services.SystemMouseCursors.ResizeUpDown {
		t.Errorf("after the region changed its cursor: %v", got)
	}

	// The region leaves the tree during the hold: the hold ends, also when the pointer
	// is over something else.
	sash.invalid = true
	h.send(mouseEv("move", at(15)))
	if got := activeCursor(t, h, 0); got != otherCursor {
		t.Errorf("after the region was removed: %v, want the cursor of the region under the pointer", got)
	}
	h.send(mouseEv("up", at(15)))
	h.send(mouseEv("move", at(30)))
	if got := activeCursor(t, h, 0); got != services.MouseCursorUncontrolled {
		t.Errorf("after the release: %v", got)
	}
}

func TestMouseTrackerHoldKeepsTheHoverEvents(t *testing.T) {
	var sashEvents, otherEvents []gestures.PointerEvent
	sash := recordingTarget(&sashEvents)
	sash.cursor, sash.hold = sashCursor, true
	other := recordingTarget(&otherEvents)
	other.cursor = otherCursor
	h := holdLayout(sash, other)
	h.send(mouseEv("add", at(5)))
	h.send(mouseEv("down", at(5)))
	h.send(mouseEv("move", at(15)))
	h.send(mouseEv("move", at(30)))
	// Entering and leaving go on as always while the cursor is held.
	wantEvents(t, sashEvents, gestures.NewPointerEnterEvent(gestures.PointerEventOpts{Position: at(5)}), gestures.NewPointerExitEvent(gestures.PointerEventOpts{Position: at(15)}))
	wantEvents(t, otherEvents, gestures.NewPointerEnterEvent(gestures.PointerEventOpts{Position: at(15)}), gestures.NewPointerExitEvent(gestures.PointerEventOpts{Position: at(30)}))
}

func TestMouseTrackerHoldIsPerDeviceAndForMiceOnly(t *testing.T) {
	sash := &testAnnotationTarget{cursor: sashCursor, hold: true}
	other := &testAnnotationTarget{cursor: otherCursor}
	h := holdLayout(sash, other)
	h.send(evAt("add", at(5), 0, ui.PointerDeviceKindMouse))
	h.send(evAt("add", at(15), 1, ui.PointerDeviceKindMouse))
	h.send(evAt("down", at(5), 0, ui.PointerDeviceKindMouse))
	h.send(evAt("move", at(30), 0, ui.PointerDeviceKindMouse))
	h.send(evAt("hover", at(5), 1, ui.PointerDeviceKindMouse))
	// The first mouse holds the sash's cursor over nothing; the second follows what is under it.
	if a, b := activeCursor(t, h, 0), activeCursor(t, h, 1); a != sashCursor || b != sashCursor {
		t.Fatalf("cursors = %v, %v; want the sash for both (the second is over it)", a, b)
	}
	h.send(evAt("hover", at(15), 1, ui.PointerDeviceKindMouse))
	if a, b := activeCursor(t, h, 0), activeCursor(t, h, 1); a != sashCursor || b != otherCursor {
		t.Errorf("cursors = %v, %v; want the first to hold the sash and the second to follow", a, b)
	}
	// A touch pointer has no cursor and no hold.
	h.send(evAt("add", at(5), 2, ui.PointerDeviceKindTouch))
	h.send(evAt("down", at(5), 2, ui.PointerDeviceKindTouch))
	if c := activeCursor(t, h, 2); c != nil {
		t.Errorf("a touch pointer has the cursor %v", c)
	}
}
