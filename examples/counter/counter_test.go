package main

import (
	"strconv"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

// useLinux makes the framework behave as on the Linux desktop, which the example
// is for: the tests default to Android.
func useLinux(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("the platform override is a debug-only feature")
	}
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformLinux))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}

func newTester(t *testing.T) *gluttertest.Tester {
	t.Helper()
	useLinux(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: m.Size{Width: 480, Height: 640}})
	ts.PumpWidget(NewMyApp())
	return ts
}

func TestTitleAndLabels(t *testing.T) {
	ts := newTester(t)
	for _, text := range []string{"Counter", "You have pushed the button this many times:", "0"} {
		if n := ts.Count(gluttertest.ByText(text)); n != 1 {
			t.Errorf("the text %q is shown %d times, want 1", text, n)
		}
	}
	if n := ts.Count(gluttertest.ByType[*m.FloatingActionButton]()); n != 1 {
		t.Errorf("%d floating action buttons, want 1", n)
	}
}

func TestTapIncrements(t *testing.T) {
	ts := newTester(t)
	for i := 1; i <= 3; i++ {
		ts.Tap(gluttertest.ByType[*m.FloatingActionButton]())
		ts.Pump(50 * time.Millisecond)
		if n := ts.Count(gluttertest.ByText(strconv.Itoa(i))); n != 1 {
			t.Fatalf("after %d taps the text %q is shown %d times, want 1", i, strconv.Itoa(i), n)
		}
	}
	if n := ts.Count(gluttertest.ByText("0")); n != 0 {
		t.Errorf("the old count is still shown")
	}
}

// TestLayout checks the Material 3 geometry: a 56 px app bar with the title
// 16 px from the start, and the 56 px button 16 px from the end and the bottom.
func TestLayout(t *testing.T) {
	ts := newTester(t)
	bar := ts.Size(gluttertest.ByType[*m.AppBar]())
	if bar.Width != 480 || bar.Height != 56 {
		t.Errorf("app bar size = %v, want 480 x 56", bar)
	}
	if tl := ts.TopLeft(gluttertest.ByText("Counter")); tl.Dx != 16 {
		t.Errorf("title left = %v, want 16", tl.Dx)
	}
	fab := gluttertest.ByType[*m.FloatingActionButton]()
	size, tl := ts.Size(fab), ts.TopLeft(fab)
	if size.Width != 56 || size.Height != 56 || tl.Dx != 480-16-56 || tl.Dy != 640-16-56 {
		t.Errorf("button at %v with size %v", tl, size)
	}
	body := ts.Size(gluttertest.ByType[*m.Column]())
	if body.Height < 40 {
		t.Errorf("column size = %v", body)
	}
}

func TestDevTaps(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("GLUTTER_DEV_TAPS is read in debug builds only")
	}
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{"unset", "", 0},
		{"three", "3", 3},
		{"negative", "-1", 0},
		{"not a number", "x", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := devTaps(func(k string) string {
				if k != "GLUTTER_DEV_TAPS" {
					t.Errorf("read %q", k)
				}
				return tc.value
			})
			if got != tc.want {
				t.Errorf("devTaps(%q) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}

// TestCounterGolden draws the app after three taps.
func TestCounterGolden(t *testing.T) {
	ts := newTester(t)
	for range 3 {
		ts.Tap(gluttertest.ByType[*m.FloatingActionButton]())
		ts.Pump(50 * time.Millisecond)
	}
	ts.Pump(time.Second)
	ts.MatchesGolden("counter")
}
