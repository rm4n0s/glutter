package widgets_test

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// TestGestureDetectorFirstArenaIsResolvedBeforeTheSecondPress is the report of
// Emetos (step 29.17): a GestureDetector gets a press, a move of 150 pixels, a
// release and a press of pointer 0 in one turn of the event loop, with nothing
// posted run in between. The default win of the first arena (a microtask in
// Dart) must come before the second press is handled, as the order of the
// callbacks shows, and nothing may panic.
func TestGestureDetectorFirstArenaIsResolvedBeforeTheSecondPress(t *testing.T) {
	tests := []struct {
		name         string
		first, next  int
		wantInTurn   []string
		wantNextTurn []string
	}{
		{"the same pointer id", 0, 0,
			[]string{"down", "cancel"}, []string{"down", "cancel", "down"}},
		{"new pointer ids", 1, 2,
			[]string{"down", "cancel"}, []string{"down", "cancel", "down"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var log []string
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, NewGestureDetector(target(), GestureDetectorOpts{
				OnTapDown:   func(gestures.TapDownDetails) { log = append(log, "down") },
				OnTapCancel: func() { log = append(log, "cancel") },
				OnTap:       func() { log = append(log, "tap") },
			}), ui.TextDirectionLTR)
			at := ts.Center(gluttertest.ByType[*GestureDetector]())
			view := MaybeViewOf(ts.Element(gluttertest.ByType[*GestureDetector]()))
			if view == nil {
				t.Fatal("the detector is not in a view")
			}
			g := ts.Binding.Gestures()
			opts := func(id int, dx float64) gestures.PointerEventOpts {
				return gestures.PointerEventOpts{TimeStamp: ts.Elapsed(), Pointer: id, Kind: ui.PointerDeviceKindTouch,
					Position: at.Add(ui.Offset{Dx: dx}), ViewID: view.ViewID()}
			}
			g.HandlePointerEvent(gestures.NewPointerDownEvent(opts(tc.first, 0)))
			g.HandlePointerEvent(gestures.NewPointerMoveEvent(opts(tc.first, 150)))
			g.HandlePointerEvent(gestures.NewPointerUpEvent(opts(tc.first, 150)))
			g.HandlePointerEvent(gestures.NewPointerDownEvent(opts(tc.next, 0)))
			if !reflect.DeepEqual(log, tc.wantInTurn) {
				t.Errorf("log = %v before the next turn, want %v", log, tc.wantInTurn)
			}
			ts.FlushPosted()
			if !reflect.DeepEqual(log, tc.wantNextTurn) {
				t.Errorf("log = %v after the next turn, want %v", log, tc.wantNextTurn)
			}
			g.HandlePointerEvent(gestures.NewPointerUpEvent(opts(tc.next, 0)))
			ts.PumpAndSettle()
		})
	}
}
