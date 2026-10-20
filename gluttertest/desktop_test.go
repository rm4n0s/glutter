package gluttertest

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func trayApp(opts desktop.SystemTrayOpts) widgets.Widget {
	return widgets.NewDirectionality(ui.TextDirectionLTR,
		desktop.NewSystemTray(ui.WindowIcon{Name: "x"}, widgets.NewText("child"), opts))
}

func TestFakeDesktopTray(t *testing.T) {
	ts := New(t)
	fd := ts.FakeDesktop()
	var calls []string
	ts.PumpWidget(trayApp(desktop.SystemTrayOpts{
		Title:      "Tray title",
		OnActivate: func() { calls = append(calls, "activate") },
		Menu: []desktop.TrayMenuEntry{
			desktop.NewTrayMenuItem("Do_it", func() { calls = append(calls, "doit") }),
			desktop.NewTrayMenuSeparator(),
			desktop.NewTrayMenuItem("Quit", func() { calls = append(calls, "quit") }),
		},
	}))
	fd.WaitFor("the tray", fd.TrayRegistered)
	if got, want := fd.TrayMenu(), []string{"Do_it", "-", "Quit"}; !reflect.DeepEqual(got, want) {
		t.Errorf("menu = %q, want %q", got, want)
	}
	if got := fd.TrayProperty("Title"); got != "Tray title" {
		t.Errorf("Title = %v", got)
	}
	if got := fd.TrayProperty("Nope"); got != nil {
		t.Errorf("a property that is not there = %v", got)
	}
	fd.ClickTrayMenuItem("Do_it")
	fd.ActivateTray()
	fd.ClickTrayMenuItem("Quit")
	if want := []string{"doit", "activate", "quit"}; !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestFakeDesktopClickOfAMissingItemFails(t *testing.T) {
	tb := &fakeTB{}
	ts := New(tb)
	fd := ts.FakeDesktop()
	ts.PumpWidget(trayApp(desktop.SystemTrayOpts{Menu: []desktop.TrayMenuEntry{desktop.NewTrayMenuItem("A", nil)}}))
	fd.WaitFor("the tray", fd.TrayRegistered)
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(fatalStop); !ok {
					panic(r)
				}
			}
		}()
		fd.ClickTrayMenuItem("B")
	}()
	if len(tb.errorfs) != 1 || !strings.Contains(tb.errorfs[0], `no item "B"`) {
		t.Errorf("failures = %q", tb.errorfs)
	}
}

func TestFakeDesktopNotifications(t *testing.T) {
	ts := New(t)
	fd := ts.FakeDesktop()
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("main")))
	bc := ts.Element(ByText("main"))

	var actions []string
	var closed []desktop.NotificationCloseReason
	show := func(summary string, replaces desktop.NotificationID) desktop.NotificationID {
		var id desktop.NotificationID
		done := make(chan error, 1)
		go func() {
			var err error
			id, err = desktop.ShowNotification(context.Background(), bc, summary, desktop.NotificationOpts{
				Body: "b", Icon: "i", Urgency: desktop.NotificationUrgencyCritical, ReplacesID: replaces,
				Actions:  []desktop.NotificationAction{{ID: "a", Label: "A"}},
				OnAction: func(a string) { actions = append(actions, a) },
				OnClosed: func(r desktop.NotificationCloseReason) { closed = append(closed, r) },
			})
			done <- err
		}()
		fd.WaitFor("the notification call", func() bool {
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
				return true
			default:
				return false
			}
		})
		return id
	}
	first := show("one", 0)
	second := show("two", 0)
	replaced := show("three", first)
	if replaced != first || second == first {
		t.Fatalf("ids %d %d %d", first, second, replaced)
	}
	notes := fd.Notifications()
	if len(notes) != 2 || notes[0].Summary != "three" || notes[1].Summary != "two" {
		t.Fatalf("notifications = %+v", notes)
	}
	n := notes[0]
	if n.Body != "b" || n.Icon != "i" || n.AppName != "Glutter" || n.Urgency != 2 || n.Closed ||
		!reflect.DeepEqual(n.Actions, []FakeNotificationAction{{ID: "a", Label: "A"}}) {
		t.Errorf("notification = %+v", n)
	}
	fd.InvokeNotificationAction(uint32(first), "a")
	fd.CloseNotification(uint32(second), 2)
	if !reflect.DeepEqual(actions, []string{"a"}) || !reflect.DeepEqual(closed, []desktop.NotificationCloseReason{desktop.NotificationClosedDismissed}) {
		t.Errorf("actions %v, closed %v", actions, closed)
	}
	if !fd.Notifications()[1].Closed {
		t.Error("the closed notification is not marked")
	}
}

func TestFakeDesktopFileChooser(t *testing.T) {
	ts := New(t)
	fd := ts.FakeDesktop()
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("main")))
	bc := ts.Element(ByText("main"))

	results := make(chan desktop.FilesResult, 2)
	open := func() {
		go func() {
			res, err := desktop.OpenFiles(context.Background(), bc)
			if err != nil {
				t.Error(err)
			}
			results <- res
		}()
	}
	open()
	fd.WaitFor("a dialog", func() bool { return fd.PendingFileChoosers() == 1 })
	fd.AnswerFileChooser("/a b/c.txt", "/d.txt")
	res := <-waitResult(fd, results)
	if res.Cancelled || !reflect.DeepEqual(res.Paths, []string{"/a b/c.txt", "/d.txt"}) {
		t.Errorf("result = %+v", res)
	}
	open()
	fd.WaitFor("a dialog", func() bool { return fd.PendingFileChoosers() == 1 })
	fd.CancelFileChooser()
	if res := <-waitResult(fd, results); !res.Cancelled {
		t.Errorf("result = %+v, want cancelled", res)
	}
	if fd.PendingFileChoosers() != 0 {
		t.Error("a dialog is still open")
	}
}

// waitResult pumps the UI thread until a result arrives, and hands it over.
func waitResult(fd *FakeDesktop, in <-chan desktop.FilesResult) <-chan desktop.FilesResult {
	out := make(chan desktop.FilesResult, 1)
	var got desktop.FilesResult
	have := false
	fd.WaitFor("the dialog's result", func() bool {
		select {
		case got = <-in:
			have = true
		default:
		}
		return have
	})
	out <- got
	return out
}

func TestFakeDesktopAnswerWithoutADialogFails(t *testing.T) {
	tb := &fakeTB{}
	ts := New(tb)
	fd := ts.FakeDesktop()
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(fatalStop); !ok {
					panic(r)
				}
			}
		}()
		fd.CancelFileChooser()
	}()
	if len(tb.errorfs) != 1 || !strings.Contains(tb.errorfs[0], "no file dialog is open") {
		t.Errorf("failures = %q", tb.errorfs)
	}
}
