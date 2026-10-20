package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

// openDialogsPage opens the dialogs page through the navigation rail.
func openDialogsPage(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	ts.Tap(gluttertest.First(gluttertest.ByText("Dialogs & pickers")))
	ts.PumpAndSettle()
	return ts
}

// TestDialogsGolden draws the dialogs page in the light theme.
func TestDialogsGolden(t *testing.T) {
	ts := openDialogsPage(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-dialogs")
}

// TestDialogsDevOpen checks that GLUTTER_DEV_OPEN opens the dialogs when the page
// is shown, in a debug build, and draws the date picker and the time picker.
func TestDialogsDevOpen(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		want   string
		golden string
	}{
		{"none", "", "", ""},
		{"alert", "alert", "Discard draft?", ""},
		{"simple", "simple", "Choose an account", ""},
		{"fullscreen", "fullscreen", "New event", ""},
		{"date picker", "datepicker", "Select a date", "gallery-date-picker"},
		{"date picker input", "datepickerinput", "Select a date", ""},
		{"date range picker", "daterange", "Select range", "gallery-date-range-picker"},
		{"time picker", "timepicker", "Select time", "gallery-time-picker"},
		{"time picker input", "timepickerinput", "Enter time", "gallery-time-picker-input"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !foundation.DebugMode {
				t.Skip("the dev variables work only in debug builds")
			}
			t.Setenv("GLUTTER_DEV_OPEN", tc.value)
			ts := openDialogsPage(t)
			ts.Pump(time.Second)
			ts.PumpAndSettle()
			if tc.want != "" && !ts.Exists(gluttertest.ByText(tc.want)) {
				t.Fatalf("the dialog with %q is not open", tc.want)
			}
			if tc.want == "" && ts.Exists(gluttertest.ByText("Cancel")) {
				t.Fatal("a dialog is open")
			}
			if tc.golden != "" {
				ts.MatchesGolden(tc.golden)
			}
		})
	}
}

// TestDialogsResults taps through each dialog and checks the result the page shows.
func TestDialogsResults(t *testing.T) {
	tests := []struct {
		name string
		act  func(ts *gluttertest.Tester)
		want string
	}{
		{"alert discard", func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("AlertDialog"))
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByText("Discard"))
		}, "Result: AlertDialog: Discard"},
		{"alert dismissed by the barrier", func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("AlertDialog"))
			ts.PumpAndSettle()
			ts.TapAt(m.Offset{Dx: 5, Dy: 700})
		}, "Result: AlertDialog: dismissed"},
		{"simple", func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("SimpleDialog"))
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByText("Grace"))
		}, "Result: SimpleDialog: Grace"},
		{"fullscreen", func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("Fullscreen"))
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByText("Save"))
		}, "Result: Fullscreen: Saved"},
		{"date", func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("Date"))
			ts.PumpAndSettle()
			ts.Tap(gluttertest.Last(gluttertest.ByText("21")))
			ts.Tap(gluttertest.ByText("OK"))
		}, "Date: Jun 21, 2025"},
		{"time", func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("Time"))
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByText("OK"))
		}, "Time: 10:30 AM"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := openDialogsPage(t)
			tc.act(ts)
			// The result comes back through a goroutine and the binding's Post.
			deadline := time.Now().Add(2 * time.Second)
			for !ts.Exists(gluttertest.ByText(tc.want)) && time.Now().Before(deadline) {
				time.Sleep(10 * time.Millisecond)
				ts.PumpAndSettle()
			}
			if !ts.Exists(gluttertest.ByText(tc.want)) {
				t.Errorf("the page does not show %q", tc.want)
			}
		})
	}
}
