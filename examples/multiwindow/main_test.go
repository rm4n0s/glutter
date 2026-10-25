package main

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
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

// newTester pumps the app as main shows it: the first window, which has a controller
// of its own, below the WindowManager.
func newTester(t *testing.T) *gluttertest.Tester {
	t.Helper()
	useLinux(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: m.Size{Width: 420, Height: 520}})
	ts.PumpWidget(NewMultiWindowRoot(), gluttertest.PumpWidgetOpts{NoView: true})
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

func TestADropOnTheZoneShowsThePaths(t *testing.T) {
	ts := newTester(t)
	zone := gluttertest.ByText("Drop here: a file or some text")
	if n := ts.Count(zone); n != 1 {
		t.Fatalf("the drop zone is shown %d times", n)
	}
	at := ts.Center(zone)
	types := []string{"text/uri-list", "text/plain"}
	// A file manager drags a file over the zone: it would be accepted.
	if got := ts.Drop(0, ui.DropPhaseEnter, at, ui.DropEvent{Types: types, Actions: ui.DropActionCopy}); got != ui.DropActionCopy {
		t.Errorf("the zone does not accept a file: %v", got)
	}
	ts.Pump()
	// Dropped: the page shows the path.
	ts.Drop(0, ui.DropPhaseDrop, at, ui.DropEvent{Types: types, Actions: ui.DropActionCopy, URIs: []string{"file:///home/me/My%20File.txt"}})
	ts.Pump()
	if n := ts.Count(gluttertest.ByText("/home/me/My File.txt")); n != 1 {
		t.Errorf("the path is shown %d times, want 1", n)
	}
	// Text instead of a file.
	at = ts.Center(gluttertest.ByText("/home/me/My File.txt"))
	ts.Drop(0, ui.DropPhaseDrop, at, ui.DropEvent{Types: []string{"text/plain"}, Actions: ui.DropActionCopy, Text: "some words"})
	ts.Pump()
	if n := ts.Count(gluttertest.ByText("some words")); n != 1 {
		t.Errorf("the text is shown %d times, want 1", n)
	}
}

func TestTheChipIsDraggedOutOfTheWindowAsText(t *testing.T) {
	ts := newTester(t)
	g := ts.StartGesture(ts.Center(gluttertest.ByText("Drag me")))
	ts.Pump()
	drags := ts.Drags()
	if len(drags) != 1 || drags[0].Text != "Dragged from this window" {
		t.Fatalf("drags = %+v", drags)
	}
	g.Up()
	ts.Pump()
	// The platform drops it on the zone of the same window: the zone shows the chip's data.
	at := ts.Center(gluttertest.ByText("Drop here: a file or some text"))
	ts.Drop(0, ui.DropPhaseDrop, at, ui.DropEvent{Types: []string{"text/plain"}, Actions: ui.DropActionCopy, LocalID: drags[0].ID})
	ts.Pump()
	if n := ts.Count(gluttertest.ByText("From this app: this window")); n != 1 {
		t.Errorf("the data of the chip is shown %d times, want 1", n)
	}
}

func TestClosingTheFirstWindowKeepsTheOtherOpen(t *testing.T) {
	ts := newTester(t)
	ts.Tap(gluttertest.ByText("New window"))
	ts.Pump()
	second := ts.Windows()[0]
	// The user closes the first window: its delegate destroys it, and nothing asks the app to exit.
	if ts.RequestWindowClose(ui.ImplicitViewID) {
		t.Error("the engine would close the first window itself, with the request to exit")
	}
	ts.Pump()
	if ts.Dispatcher().View(ui.ImplicitViewID) != nil {
		t.Error("the first window is still there")
	}
	if n := ts.Count(gluttertest.ByText("Add one")); n != 1 {
		t.Fatalf("%d pages are shown, want only the second window's", n)
	}
	if second.Destroyed || ts.ExitRequests() != 0 {
		t.Errorf("the second window is destroyed = %v, exit requests = %d", second.Destroyed, ts.ExitRequests())
	}
	// The second window still works.
	ts.Tap(gluttertest.ByText("Add one"))
	ts.Pump()
	if n := ts.Count(gluttertest.ByText("1")); n != 1 {
		t.Errorf("the counter of the second window shows 1 in %d places, want 1", n)
	}
}
