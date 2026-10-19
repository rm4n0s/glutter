package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from the TextSelectionGestureDetector tests of
// packages/flutter/test/widgets/text_selection_test.dart: the taps, the long
// presses and the drags. Not ported: the tests of TextSelectionGestureDetectorBuilder
// with a fake editable (step 11.5 ports them with the selection overlay), and the
// force press ones (there are no force presses on Linux, SPEC §12.10).

type gestureCounts struct {
	tap, singleTapUp, singleTapCancel, singleLongTapStart, doubleTapDown, tripleTapDown int
	dragStart, dragUpdate, dragEnd                                                      int
}

func pumpGestureDetector(ts *gluttertest.Tester) *gestureCounts {
	c := &gestureCounts{}
	ts.PumpWidget(NewTextSelectionGestureDetector(NewContainer(nil), TextSelectionGestureDetectorOpts{
		Behavior:             rendering.HitTestBehaviorOpaque,
		OnTapDown:            func(gestures.TapDragDownDetails) { c.tap++ },
		OnSingleTapUp:        func(gestures.TapDragUpDetails) { c.singleTapUp++ },
		OnSingleTapCancel:    func() { c.singleTapCancel++ },
		OnSingleLongTapStart: func(gestures.LongPressStartDetails) { c.singleLongTapStart++ },
		OnDoubleTapDown:      func(gestures.TapDragDownDetails) { c.doubleTapDown++ },
		OnTripleTapDown:      func(gestures.TapDragDownDetails) { c.tripleTapDown++ },
		OnDragSelectionStart: func(gestures.TapDragStartDetails) { c.dragStart++ },
		OnDragSelectionUpdate: func(gestures.TapDragUpdateDetails) {
			c.dragUpdate++
		},
		OnDragSelectionEnd: func(gestures.TapDragEndDetails) { c.dragEnd++ },
	}))
	return c
}

var tapPoint = ui.Offset{Dx: 200, Dy: 200}

func TestTextSelectionGestureDetectorSeriesOfTaps(t *testing.T) {
	ts := gluttertest.New(t)
	c := pumpGestureDetector(ts)
	for i := 0; i < 5; i++ {
		ts.TapAt(tapPoint)
		ts.Pump(150 * time.Millisecond)
	}
	ts.TapAt(tapPoint)
	if c.tap != 6 {
		t.Errorf("tapCount = %d, want 6", c.tap)
	}
}

func TestTextSelectionGestureDetectorRapidTapsAlternate(t *testing.T) {
	ts := gluttertest.New(t)
	c := pumpGestureDetector(ts)
	steps := []struct{ singleUp, double, triple int }{
		{1, 0, 0}, {1, 1, 0}, {1, 1, 1}, {2, 1, 1}, {2, 2, 1}, {2, 2, 2},
	}
	for i, want := range steps {
		ts.TapAt(tapPoint)
		ts.Pump(50 * time.Millisecond)
		if c.singleTapUp != want.singleUp || c.doubleTapDown != want.double || c.tripleTapDown != want.triple {
			t.Errorf("after tap %d: single %d, double %d, triple %d; want %+v", i+1, c.singleTapUp, c.doubleTapDown, c.tripleTapDown, want)
		}
	}
	ts.TapAt(tapPoint)
	if c.singleTapUp != 3 || c.doubleTapDown != 2 || c.tripleTapDown != 2 || c.tap != 7 {
		t.Errorf("after the last tap: single %d, double %d, triple %d, taps %d; want 3, 2, 2, 7", c.singleTapUp, c.doubleTapDown, c.tripleTapDown, c.tap)
	}
}

func TestTextSelectionGestureDetectorQuickTapTapHoldIsADoubleTapDown(t *testing.T) {
	ts := gluttertest.New(t)
	c := pumpGestureDetector(ts)
	ts.TapAt(tapPoint)
	ts.Pump(50 * time.Millisecond)
	if c.singleTapUp != 1 {
		t.Fatalf("singleTapUp = %d", c.singleTapUp)
	}
	g := ts.StartGesture(tapPoint)
	ts.Pump(200 * time.Millisecond)
	check := func(when string) {
		t.Helper()
		// Every down is counted. There are no cancels because the second tap of the
		// double tap is a second successful single tap behind the scene. The double
		// tap down hold supersedes the single tap down.
		if c.singleTapUp != 1 || c.tap != 2 || c.singleTapCancel != 0 || c.doubleTapDown != 1 || c.singleLongTapStart != 0 {
			t.Errorf("%s: %+v", when, *c)
		}
	}
	check("while holding")
	g.Up()
	check("after the release")
}

func TestTextSelectionGestureDetectorSwipes(t *testing.T) {
	tests := []struct {
		name  string
		delay time.Duration
	}{
		{"a very quick swipe is ignored", 20 * time.Millisecond},
		{"a slower swipe has a tap down and a canceled tap", 120 * time.Millisecond},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			c := pumpGestureDetector(ts)
			g := ts.StartGesture(tapPoint)
			ts.Pump(tc.delay)
			g.MoveBy(ui.Offset{Dx: 100, Dy: 100})
			ts.Pump()
			// The tap and drag recognizer has two thresholds, so the tap down is
			// counted even though the pointer moved past the tap threshold.
			check := func(when string) {
				t.Helper()
				if c.singleTapUp != 0 || c.tap != 1 || c.singleTapCancel != 0 || c.doubleTapDown != 0 || c.singleLongTapStart != 0 {
					t.Errorf("%s: %+v", when, *c)
				}
			}
			check("after the move")
			g.Up()
			check("after the release")
		})
	}
}

func TestTextSelectionGestureDetectorLongPressAndDrags(t *testing.T) {
	mouse := ui.PointerDeviceKindMouse
	tests := []struct {
		name  string
		kind  ui.PointerDeviceKind
		hold  time.Duration
		drag  bool
		check func(t *testing.T, c *gestureCounts)
	}{
		{"a long press from a touch device is recognized as a long single tap", ui.PointerDeviceKindTouch, 2 * time.Second, false,
			func(t *testing.T, c *gestureCounts) {
				if c.tap != 1 || c.singleTapUp != 0 || c.singleLongTapStart != 1 {
					t.Errorf("%+v", *c)
				}
			}},
		{"a long press from a mouse is just a tap", mouse, 2 * time.Second, false,
			func(t *testing.T, c *gestureCounts) {
				if c.tap != 1 || c.singleTapUp != 1 || c.singleLongTapStart != 0 {
					t.Errorf("%+v", *c)
				}
			}},
		{"a touch drag is recognized for text selection", ui.PointerDeviceKindTouch, 0, true,
			func(t *testing.T, c *gestureCounts) {
				if c.tap != 1 || c.singleTapUp != 0 || c.singleTapCancel != 0 || c.dragStart != 1 || c.dragUpdate != 1 || c.dragEnd != 1 {
					t.Errorf("%+v", *c)
				}
			}},
		{"a mouse drag is recognized for text selection", mouse, 0, true,
			func(t *testing.T, c *gestureCounts) {
				// The recognizer detects the tap down, but not the tap up.
				if c.tap != 1 || c.singleTapCancel != 0 || c.singleTapUp != 0 || c.dragStart != 1 || c.dragUpdate != 1 || c.dragEnd != 1 {
					t.Errorf("%+v", *c)
				}
			}},
		{"a slow mouse drag is still recognized for text selection", mouse, 2 * time.Second, true,
			func(t *testing.T, c *gestureCounts) {
				if c.tap != 1 || c.singleTapCancel != 0 || c.singleTapUp != 0 || c.dragStart != 1 || c.dragUpdate != 1 || c.dragEnd != 1 {
					t.Errorf("%+v", *c)
				}
			}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			c := pumpGestureDetector(ts)
			g := ts.StartGesture(tapPoint, gluttertest.GestureOpts{Kind: tc.kind})
			if tc.hold > 0 {
				ts.Pump(tc.hold)
			} else {
				ts.Pump()
			}
			if tc.drag {
				g.MoveBy(ui.Offset{Dx: 210, Dy: 200})
				ts.Pump()
			}
			g.Up()
			ts.PumpAndSettle()
			tc.check(t, c)
		})
	}
}
