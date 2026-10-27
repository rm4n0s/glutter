package widgets_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// TestGestureDetectorDoubleTapBeforeThePostOfTheFirstArena sends two taps to a
// GestureDetector with a double-tap recognizer without running the posted
// functions between them, as pointer events that arrive in one batch do: the
// default resolution of the first arena (a post to the UI thread) comes after
// the second down (the second tap is too quick to be a double tap, so it
// restarts the recognizer, which gives up the first arena). Some platforms give a new pointer the id of one that is
// finished, so the second tap can have the id of the first.
func TestGestureDetectorDoubleTapBeforeThePostOfTheFirstArena(t *testing.T) {
	tests := []struct {
		name               string
		first, next, third int // the pointer ids of the three taps
		want               []string
	}{
		{"the taps have new pointer ids", 1, 2, 3, []string{"tap down", "double"}},
		{"the taps have the same pointer id", 1, 1, 1, []string{"tap down", "double"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var log []string
			ts := gluttertest.New(t, gluttertest.Opts{})
			pumpLoose(ts, NewGestureDetector(target(), GestureDetectorOpts{
				OnDoubleTap:     func() { log = append(log, "double") },
				OnDoubleTapDown: func(gestures.TapDownDetails) { log = append(log, "tap down") },
			}), ui.TextDirectionLTR)
			at := ts.Center(gluttertest.ByType[*GestureDetector]())
			view := MaybeViewOf(ts.Element(gluttertest.ByType[*GestureDetector]()))
			if view == nil {
				t.Fatal("the detector is not in a view")
			}
			g := ts.Binding.Gestures()
			send := func(e gestures.PointerEvent) { g.HandlePointerEvent(e) }
			opts := func(id int, after time.Duration) gestures.PointerEventOpts {
				return gestures.PointerEventOpts{TimeStamp: ts.Elapsed() + after, Pointer: id, Kind: ui.PointerDeviceKindTouch, Position: at, ViewID: view.ViewID()}
			}
			send(gestures.NewPointerDownEvent(opts(tc.first, 0)))
			send(gestures.NewPointerUpEvent(opts(tc.first, 0)))
			send(gestures.NewPointerDownEvent(opts(tc.next, 50*time.Millisecond)))
			send(gestures.NewPointerUpEvent(opts(tc.next, 50*time.Millisecond)))
			ts.FlushPosted()
			if len(log) != 0 {
				// The second tap came too soon after the first (kDoubleTapMinTime), so it
				// starts over as a first tap.
				t.Errorf("log = %v after the quick second tap, want none", log)
			}
			// A third tap after the minimum time is the second tap of a double tap
			// with the second one, whatever the pointer ids are.
			ts.Pump(50 * time.Millisecond)
			ts.TapAt(at, gluttertest.GestureOpts{Pointer: tc.third})
			ts.PumpAndSettle()
			if !reflect.DeepEqual(log, tc.want) {
				t.Errorf("log = %v, want %v", log, tc.want)
			}
		})
	}
}
