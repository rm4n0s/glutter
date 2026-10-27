//go:build linux && !android

package input_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/platform/linux/input"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// TestDoubleTapThroughTheLinuxInput sends three taps to a GestureDetector with a double-tap
// recognizer, as the Linux input makes them (the data goes to the dispatcher, the pointer ids come
// from the batchers), without running the posted functions between the first two taps, as the
// events of one batch do (the sequence that made the arena panic, step 29.11). The mouse used to
// give every press the id 0 and the touch the id of the compositor, which it reuses; with a new id
// at each down the arenas of the taps are never confused, and the third tap makes a double tap.
func TestDoubleTapThroughTheLinuxInput(t *testing.T) {
	tests := []struct {
		name string
		// device makes the function that sends one tap through the device's batcher and returns its data.
		device func(b *input.PointerIDs, now func() time.Duration, at ui.Offset) func(d *ui.PlatformDispatcher) []ui.PointerData
	}{
		{"the mouse", func(ids *input.PointerIDs, now func() time.Duration, at ui.Offset) func(d *ui.PlatformDispatcher) []ui.PointerData {
			b := input.NewPointerBatcher(now, ids)
			b.Enter(0, at.Dx, at.Dy)
			return func(d *ui.PlatformDispatcher) []ui.PointerData {
				b.Button(0x110, true)
				b.Button(0x110, false)
				data := b.Frame()
				d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: data})
				return data
			}
		}},
		{"a touch screen that reuses the id 0", func(ids *input.PointerIDs, now func() time.Duration, at ui.Offset) func(d *ui.PlatformDispatcher) []ui.PointerData {
			b := input.NewTouchBatcher(now, ids)
			return func(d *ui.PlatformDispatcher) []ui.PointerData {
				b.Down(0, 0, at.Dx, at.Dy)
				b.Up(0)
				data := b.Take()
				d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: data})
				return data
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var log []string
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewAlign(NewGestureDetector(
				NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
				GestureDetectorOpts{
					Behavior:        rendering.HitTestBehaviorOpaque,
					OnDoubleTap:     func() { log = append(log, "double") },
					OnDoubleTapDown: func(gestures.TapDownDetails) { log = append(log, "tap down") },
				}), AlignOpts{Alignment: painting.AlignmentTopLeft})))
			at := ts.Center(gluttertest.ByType[*GestureDetector]())
			// The time of the data is the test clock plus a skew: the second tap is 50 ms after the
			// first without the posted functions having run.
			var skew time.Duration
			tap := tc.device(input.NewPointerIDs(), func() time.Duration { return ts.Elapsed() + skew }, at)

			var sent []ui.PointerData
			sent = append(sent, tap(ts.Dispatcher())...)
			skew = 50 * time.Millisecond
			sent = append(sent, tap(ts.Dispatcher())...)
			ts.FlushPosted()
			if len(log) != 0 {
				// The second tap came too soon after the first (kDoubleTapMinTime), so it
				// starts over as a first tap.
				t.Errorf("log = %v after the quick second tap, want none", log)
			}
			ts.Pump(50 * time.Millisecond)
			sent = append(sent, tap(ts.Dispatcher())...)
			ts.PumpAndSettle()
			// The three downs have three different pointer ids.
			seen := map[int]bool{}
			for _, d := range sent {
				if d.Change == ui.PointerChangeDown {
					if d.PointerIdentifier == 0 || seen[d.PointerIdentifier] {
						t.Errorf("down with the identifier %d; the others had %v", d.PointerIdentifier, seen)
					}
					seen[d.PointerIdentifier] = true
				}
			}
			if len(seen) != 3 {
				t.Errorf("%d different identifiers in the downs, want 3", len(seen))
			}
			if want := []string{"tap down", "double"}; !reflect.DeepEqual(log, want) {
				t.Errorf("log = %v, want %v", log, want)
			}
		})
	}
}
