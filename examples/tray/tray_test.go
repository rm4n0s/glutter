package main

import (
	"reflect"
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

// newApp starts the app on a fake desktop and waits until its tray icon is registered.
func newApp(t *testing.T) (*gluttertest.Tester, *gluttertest.FakeDesktop) {
	t.Helper()
	useLinux(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: m.Size{Width: 480, Height: 360}})
	fd := ts.FakeDesktop()
	ts.PumpWidget(NewTrayApp())
	fd.WaitFor("the tray icon", fd.TrayRegistered)
	return ts, fd
}

func actionKinds(ts *gluttertest.Tester) []ui.WindowActionKind {
	var kinds []ui.WindowActionKind
	for _, a := range ts.WindowActions() {
		kinds = append(kinds, a.Kind)
	}
	return kinds
}

func lastAction(ts *gluttertest.Tester) ui.WindowActionKind {
	kinds := actionKinds(ts)
	if len(kinds) == 0 {
		return -1
	}
	return kinds[len(kinds)-1]
}

func TestTrayMenuAndIcon(t *testing.T) {
	ts, fd := newApp(t)
	want := []string{"Show window", "New notification", "Open file…", "-", "Quit"}
	if got := fd.TrayMenu(); !reflect.DeepEqual(got, want) {
		t.Errorf("menu = %q, want %q", got, want)
	}
	if got := fd.TrayProperty("Title"); got != "Glutter tray" {
		t.Errorf("Title = %v", got)
	}
	if got := fd.TrayProperty("Status"); got != "Active" {
		t.Errorf("Status = %v", got)
	}
	if n := ts.Count(gluttertest.ByType[*m.TextField]()); n != 1 {
		t.Errorf("%d note fields, want 1", n)
	}
	if !drawsAllSizes(t) {
		t.Error("the icon does not have the five sizes")
	}
}

func drawsAllSizes(t *testing.T) bool {
	t.Helper()
	icon, err := drawIcon()
	if err != nil {
		t.Fatal(err)
	}
	var sizes []int
	for _, img := range icon.Images {
		sizes = append(sizes, img.Size)
	}
	return reflect.DeepEqual(sizes, []int{16, 22, 24, 32, 48})
}

func TestClosingTheWindowHidesItAndKeepsTheApp(t *testing.T) {
	ts, _ := newApp(t)
	if ts.Dispatcher().RequestClose(ts.View.ViewID()) {
		t.Error("the window was allowed to close")
	}
	if got := lastAction(ts); got != ui.WindowActionMinimize {
		t.Errorf("last window action = %v, want minimize", got)
	}
	if n := ts.ExitRequests(); n != 0 {
		t.Errorf("%d exit requests", n)
	}
}

func TestShowWindowFromTheMenuAndTheIcon(t *testing.T) {
	for _, tt := range []struct {
		name  string
		bring func(fd *gluttertest.FakeDesktop)
	}{
		{"the menu item", func(fd *gluttertest.FakeDesktop) { fd.ClickTrayMenuItem("Show window") }},
		{"a click on the icon", func(fd *gluttertest.FakeDesktop) { fd.ActivateTray() }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts, fd := newApp(t)
			ts.Dispatcher().RequestClose(ts.View.ViewID())
			tt.bring(fd)
			if got := lastAction(ts); got != ui.WindowActionActivate {
				t.Errorf("last window action = %v, want activate", got)
			}
		})
	}
}

func TestNewNotificationHasAnOpenActionThatBringsTheWindowBack(t *testing.T) {
	for _, action := range []string{"open", "default"} {
		t.Run(action, func(t *testing.T) {
			ts, fd := newApp(t)
			ts.EnterText(gluttertest.ByType[*m.TextField](), "buy milk")
			fd.ClickTrayMenuItem("New notification")
			fd.WaitFor("the notification", func() bool { return len(fd.Notifications()) == 1 })
			n := fd.Notifications()[0]
			if n.Summary != "Glutter tray" || n.Body != "buy milk" {
				t.Errorf("summary %q, body %q", n.Summary, n.Body)
			}
			var ids []string
			for _, a := range n.Actions {
				ids = append(ids, a.ID)
			}
			if !reflect.DeepEqual(ids, []string{"default", "open"}) {
				t.Errorf("actions = %v", ids)
			}
			ts.Pump()
			ts.Element(gluttertest.ByText("Notification shown."))

			ts.Dispatcher().RequestClose(ts.View.ViewID())
			fd.InvokeNotificationAction(n.ID, action)
			if got := lastAction(ts); got != ui.WindowActionActivate {
				t.Errorf("last window action = %v, want activate", got)
			}
			fd.InvokeNotificationAction(n.ID, "something else")
		})
	}
}

func TestNewNotificationWithAnEmptyNote(t *testing.T) {
	_, fd := newApp(t)
	fd.ClickTrayMenuItem("New notification")
	fd.WaitFor("the notification", func() bool { return len(fd.Notifications()) == 1 })
	if got := fd.Notifications()[0].Body; got != "Your note is empty." {
		t.Errorf("body = %q", got)
	}
}

func TestOpenFileShowsThePath(t *testing.T) {
	ts, fd := newApp(t)
	fd.ClickTrayMenuItem("Open file…")
	fd.WaitFor("the file dialog", func() bool { return fd.PendingFileChoosers() == 1 })
	fd.AnswerFileChooser("/home/me/my notes.txt")
	fd.WaitFor("the path", func() bool { ts.Pump(); return ts.Exists(gluttertest.ByText("Opened: /home/me/my notes.txt")) })
	if got := lastAction(ts); got != ui.WindowActionActivate {
		t.Errorf("last window action = %v, want activate", got)
	}
}

func TestOpenFileCancelled(t *testing.T) {
	ts, fd := newApp(t)
	fd.ClickTrayMenuItem("Open file…")
	fd.WaitFor("the file dialog", func() bool { return fd.PendingFileChoosers() == 1 })
	fd.CancelFileChooser()
	fd.WaitFor("the status", func() bool { ts.Pump(); return ts.Exists(gluttertest.ByText("Open cancelled.")) })
}

func TestQuitEndsTheApp(t *testing.T) {
	ts, fd := newApp(t)
	fd.ClickTrayMenuItem("Quit")
	if n := ts.ExitRequests(); n != 1 {
		t.Errorf("%d exit requests, want 1", n)
	}
	if !ts.Dispatcher().RequestClose(ts.View.ViewID()) {
		t.Error("the window can not close after Quit")
	}
}
