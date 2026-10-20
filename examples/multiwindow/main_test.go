package main

import (
	"testing"

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

// newTester pumps the app as RunApp shows it: in the first window, which has a
// controller of its own.
func newTester(t *testing.T) *gluttertest.Tester {
	t.Helper()
	useLinux(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: m.Size{Width: 420, Height: 360}})
	ts.PumpWidget(ts.Binding.WrapWithWindowScope(m.NewWindowManager(NewMultiWindowApp())))
	return ts
}

func TestFirstWindowShowsTheButtons(t *testing.T) {
	ts := newTester(t)
	for _, text := range []string{"This window's counter:", "0", "Add one", "New window", "Open dialog"} {
		if n := ts.Count(gluttertest.ByText(text)); n != 1 {
			t.Errorf("the text %q is shown %d times, want 1", text, n)
		}
	}
	if n := len(ts.Windows()); n != 0 {
		t.Errorf("%d windows exist before any button is tapped", n)
	}
}

func TestNewWindowsHaveTheirOwnCounters(t *testing.T) {
	ts := newTester(t)
	ts.Tap(gluttertest.ByText("New window"))
	ts.Pump()
	// The new window has the button too: tap the first window's.
	ts.Tap(gluttertest.First(gluttertest.ByText("New window")))
	ts.Pump()
	ws := ts.Windows()
	if len(ws) != 2 || ws[0].Title != "Window 1" || ws[1].Title != "Window 2" || ws[0].Spec.HasParent {
		t.Fatalf("windows = %+v", ws)
	}
	// Three pages: the first window and the two new ones.
	if n := ts.Count(gluttertest.ByText("Add one")); n != 3 {
		t.Fatalf("%d pages, want 3", n)
	}
	// Tap the button of the second window: only its counter moves.
	second := gluttertest.ByElementPredicate("in the second window", func(e m.Element) bool {
		return m.MaybeViewOf(e) != nil && m.MaybeViewOf(e).ViewID() == ws[1].View
	})
	ts.Tap(gluttertest.Descendant(second, gluttertest.ByText("Add one")))
	ts.Pump()
	if n := ts.Count(gluttertest.ByText("1")); n != 1 {
		t.Errorf("%d counters show 1, want only the second window's", n)
	}
	if n := ts.Count(gluttertest.ByText("0")); n != 2 {
		t.Errorf("%d counters show 0, want 2", n)
	}
}

func TestADialogIsModalToTheWindowThatOpenedIt(t *testing.T) {
	ts := newTester(t)
	ts.Tap(gluttertest.ByText("Open dialog"))
	ts.Pump()
	ws := ts.Windows()
	if len(ws) != 1 || !ws[0].Spec.HasParent || ws[0].Spec.Parent != 0 || ws[0].Title != "Dialog" {
		t.Fatalf("windows = %+v", ws)
	}
	if n := ts.Count(gluttertest.ByText("Close")); n != 1 {
		t.Fatalf("%d close buttons, want 1", n)
	}
	// The dialog closes itself; its window leaves the registry.
	ts.Tap(gluttertest.ByText("Close"))
	ts.Pump()
	if !ts.Windows()[0].Destroyed {
		t.Error("the dialog is still open")
	}
	if n := ts.Count(gluttertest.ByText("Close")); n != 0 {
		t.Errorf("the dialog's content is still shown")
	}
}

func TestClosingAWindowRemovesItsPage(t *testing.T) {
	ts := newTester(t)
	ts.Tap(gluttertest.ByText("New window"))
	ts.Pump()
	if ts.RequestWindowClose(ts.Windows()[0].View) {
		t.Fatal("the framework should take the request to close")
	}
	ts.Pump()
	if n := ts.Count(gluttertest.ByText("Add one")); n != 1 {
		t.Errorf("%d pages after closing the second window, want 1", n)
	}
}

func TestDevOpen(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("GLUTTER_DEV_OPEN is a debug-only feature")
	}
	tests := []struct {
		value string
		want  []string
	}{
		{"", nil},
		{"window", []string{"window"}},
		{"window,dialog", []string{"window", "dialog"}},
		{" dialog , bogus,window", []string{"dialog", "window"}},
	}
	for _, tt := range tests {
		got := devOpen(func(string) string { return tt.value })
		if len(got) != len(tt.want) {
			t.Errorf("devOpen(%q) = %v, want %v", tt.value, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("devOpen(%q) = %v, want %v", tt.value, got, tt.want)
			}
		}
	}
}
