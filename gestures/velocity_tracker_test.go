package gestures

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/fakeclock"
	"github.com/rm4n0s/glutter/ui"
)

func withinTolerance(actual, expected float64) bool {
	return math.Abs((actual-expected)/expected) < 0.001 // within .1% of the expected value
}

func newFakeClockTracker(kind func(SamplingClock, ui.PointerDeviceKind) *VelocityTracker) (*VelocityTracker, *fakeclock.Fake) {
	clock := fakeclock.New(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))
	return kind(sampleClock{clock}, ui.PointerDeviceKindTouch), clock
}

func TestVelocityTrackerGivesExpectedResults(t *testing.T) {
	expected := []ui.Offset{
		{Dx: 219.59280094228163, Dy: 1304.701682306001},
		{Dx: 355.71046950050845, Dy: 967.2112857054104},
		{Dx: 12.657970884022308, Dy: -36.90447839251946},
		{Dx: 714.1399654786744, Dy: -2561.534447931869},
		{Dx: -19.668121066218564, Dy: -2910.105747052462},
		{Dx: 646.8690114934209, Dy: 2976.977762577527},
		{Dx: 396.6988447819592, Dy: 2106.225572911095},
		{Dx: 298.31594440044495, Dy: -3660.8315955215294},
		{Dx: -1.7334232785165882, Dy: -3288.13174127454},
		{Dx: 384.6361280392334, Dy: -2645.6612524779835},
		{Dx: 176.37900397918557, Dy: 2711.2542876273264},
		{Dx: 396.9328560260098, Dy: 4280.651578291764},
		{Dx: -71.51939428321249, Dy: 3716.7385187526947},
	}
	tracker, _ := newFakeClockTracker(NewVelocityTrackerWithKind)
	i := 0
	for _, e := range velocityEventData {
		switch e.kind {
		case 'd', 'm':
			tracker.AddPosition(time.Duration(e.ms)*time.Millisecond, ui.Offset{Dx: e.x, Dy: e.y})
		case 'u':
			v := tracker.GetVelocity().PixelsPerSecond
			if !withinTolerance(v.Dx, expected[i].Dx) || !withinTolerance(v.Dy, expected[i].Dy) {
				t.Errorf("velocity %d = %v, want %v", i, v, expected[i])
			}
			i++
		}
	}
	if i != len(expected) {
		t.Errorf("checked %d velocities, want %d", i, len(expected))
	}
}

func TestVelocityControl(t *testing.T) {
	v1 := NewVelocity(ui.Offset{Dx: 7})
	v2 := NewVelocity(ui.Offset{Dx: 12})
	if v1 != NewVelocity(ui.Offset{Dx: 7}) || v1 == v2 {
		t.Error("velocity equality is wrong")
	}
	if got := v2.Sub(v1); got != NewVelocity(ui.Offset{Dx: 5}) {
		t.Errorf("v2 - v1 = %v", got)
	}
	if got := v1.Neg().PixelsPerSecond; got != (ui.Offset{Dx: -7}) {
		t.Errorf("-v1 = %v", got)
	}
	if got := v1.Add(v2); got != NewVelocity(ui.Offset{Dx: 19}) {
		t.Errorf("v1 + v2 = %v", got)
	}
	if s := v1.String(); strings.Contains(s, "\n") || s != "Velocity(7.0, 0.0)" {
		t.Errorf("String() = %q", s)
	}
}

func TestInterruptedVelocityEstimation(t *testing.T) {
	tracker, _ := newFakeClockTracker(NewVelocityTrackerWithKind)
	for _, e := range interruptedVelocityEventData {
		switch e.kind {
		case 'd', 'm':
			tracker.AddPosition(time.Duration(e.ms)*time.Millisecond, ui.Offset{Dx: e.x, Dy: e.y})
		case 'u':
			v := tracker.GetVelocity().PixelsPerSecond
			if !withinTolerance(v.Dx, 649.5) || !withinTolerance(v.Dy, 3890.3) {
				t.Errorf("velocity = %v, want (649.5, 3890.3)", v)
			}
		}
	}
}

func TestNoDataVelocityEstimation(t *testing.T) {
	tracker, _ := newFakeClockTracker(NewVelocityTrackerWithKind)
	if got := tracker.GetVelocity(); got != VelocityZero {
		t.Errorf("GetVelocity() = %v, want zero", got)
	}
}

func TestIOSVelocityEstimateWithManyPoints(t *testing.T) {
	tracker, _ := newFakeClockTracker(NewIOSScrollViewFlingVelocityTracker)
	position, tm := ui.Offset{}, time.Duration(0)
	for i := 0; i < 5; i++ {
		position = position.Add(ui.Offset{Dy: -1})
		tm += time.Second
		tracker.AddPosition(tm, position)
		tracker.GetVelocity() // must not panic
	}
}

func TestConsistentVelocityEstimates(t *testing.T) {
	tracker, _ := newFakeClockTracker(NewIOSScrollViewFlingVelocityTracker)
	position, tm := ui.Offset{}, time.Duration(0)
	delta := ui.Offset{Dy: -1}
	for i := 0; i < 10; i++ {
		position = position.Add(delta)
		tm += time.Second
		tracker.AddPosition(tm, position)
		if i >= 3 {
			if got := tracker.GetVelocity().PixelsPerSecond; got != delta {
				t.Errorf("velocity %d = %v, want %v", i, got, delta)
			}
		}
	}
}

func TestZeroVelocityWithoutRecentSamples(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind func(SamplingClock, ui.PointerDeviceKind) *VelocityTracker
	}{
		{"base", NewVelocityTrackerWithKind},
		{"iOS", NewIOSScrollViewFlingVelocityTracker},
		{"macOS", NewMacOSScrollViewFlingVelocityTracker},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tracker, clock := newFakeClockTracker(tc.kind)
			position, tm := ui.Offset{}, time.Duration(0)
			for i := 0; i < 10; i++ {
				position = position.Add(ui.Offset{Dy: -1})
				tm += time.Second
				tracker.AddPosition(tm, position)
			}
			clock.Advance(time.Second) // pumpAndSettle
			if got := tracker.GetVelocity().PixelsPerSecond; got != (ui.Offset{}) {
				t.Errorf("velocity = %v, want zero", got)
			}
		})
	}
}
