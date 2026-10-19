package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func newTester(t *testing.T) (*gluttertest.Tester, *animationsState) {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 640, Height: 480}})
	ts.PumpWidget(newAnimationsApp(false))
	return ts, gluttertest.StateOf[*animationsState](ts, gluttertest.ByType[*animationsApp]())
}

// TestAnimationsGoldens moves every animation at once and compares the frames
// at 0, 150 and 300 ms with the golden images.
func TestAnimationsGoldens(t *testing.T) {
	ts, state := newTester(t)
	state.advance()
	ts.Pump()
	ts.MatchesGolden("animations-0ms")
	ts.Pump(150 * time.Millisecond)
	ts.MatchesGolden("animations-150ms")
	ts.Pump(150 * time.Millisecond)
	ts.MatchesGolden("animations-300ms")
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("the example reported errors: %v", errs)
	}
}

// TestTapToggleTheContainer taps the first tile and checks the size the container
// animates to.
func TestTapToggleTheContainer(t *testing.T) {
	ts, state := newTester(t)
	container := gluttertest.ByType[*widgets.AnimatedContainer]()
	if got := ts.Size(container); got != (ui.Size{Width: 70, Height: 70}) {
		t.Fatalf("size = %v, want 70x70", got)
	}
	ts.Tap(container)
	if !state.expanded {
		t.Fatal("the tap did not expand the container")
	}
	ts.Pump()
	ts.Pump(300 * time.Millisecond)
	mid := ts.Size(container)
	if mid.Width <= 70 || mid.Width >= 150 {
		t.Errorf("width halfway = %v, want between 70 and 150", mid.Width)
	}
	ts.Pump(time.Second) // the pulse never settles, so no PumpAndSettle
	if got := ts.Size(container); got != (ui.Size{Width: 150, Height: 90}) {
		t.Errorf("size = %v, want 150x90", got)
	}
}

// TestAdvanceMovesTheCounter changes the counter and checks that the old number
// fades out while the new one fades in.
func TestAdvanceMovesTheCounter(t *testing.T) {
	ts, state := newTester(t)
	if ts.Count(gluttertest.ByText("0")) != 1 || ts.Count(gluttertest.ByText("1")) != 0 {
		t.Fatal("the counter does not start at 0")
	}
	state.advance()
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	if ts.Count(gluttertest.ByText("0")) != 1 || ts.Count(gluttertest.ByText("1")) != 1 {
		t.Error("the old number is not fading out next to the new one")
	}
	ts.Pump(time.Second)
	if ts.Count(gluttertest.ByText("0")) != 0 || ts.Count(gluttertest.ByText("1")) != 1 {
		t.Error("the old number did not leave")
	}
}

// TestPulseNeverSettles checks that the repeating controller keeps asking for
// frames.
func TestPulseNeverSettles(t *testing.T) {
	ts, _ := newTester(t)
	scale := func() float64 {
		return ts.Widget(gluttertest.First(gluttertest.ByType[*widgets.ScaleTransition]())).(*widgets.ScaleTransition).Scale().Value()
	}
	ts.Pump(500 * time.Millisecond)
	half := scale()
	ts.Pump(500 * time.Millisecond)
	end := scale()
	ts.Pump(500 * time.Millisecond)
	back := scale()
	if !(half > 0.7 && half < 1.2) || end < half || back > end {
		t.Errorf("scales %v, %v, %v do not grow and then shrink", half, end, back)
	}
}
