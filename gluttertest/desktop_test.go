package gluttertest

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/testhost"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// everyTargetPlatform are the target platforms that the tests of the fake desktop run as.
var everyTargetPlatform = []foundation.TargetPlatform{foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS,
	foundation.TargetPlatformWindows, foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS}

// runAs makes the test run as the target platform p until it ends. Call it after New, which
// pins Linux on an Apple host when the test chose nothing.
func runAs(t *testing.T, p foundation.TargetPlatform) {
	t.Helper()
	if foundation.DebugMode {
		foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(p))
		t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
	} else {
		testhost.SetPlatform(int32(p))
		t.Cleanup(func() { testhost.SetPlatform(0) })
	}
	if got := foundation.DefaultTargetPlatform(); got != p {
		t.Fatalf("the test runs as %v, want %v", got, p)
	}
}

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

// TestFakeDesktopTrayOnEveryTargetPlatform shows that the fake desktop replaces the desktop of every
// platform, not only Linux's D-Bus: a test may run as macOS or Windows and still see the tray (mardp's
// issue of 1.17.7 took the fake for the Linux tray and pinned its tests to Linux).
func TestFakeDesktopTrayOnEveryTargetPlatform(t *testing.T) {
	for _, p := range []foundation.TargetPlatform{foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows} {
		t.Run(p.String(), func(t *testing.T) {
			ts := New(t)
			runAs(t, p)
			fd := ts.FakeDesktop()
			var calls []string
			ts.PumpWidget(trayApp(desktop.SystemTrayOpts{
				Title:      "Tray on " + p.String(),
				OnActivate: func() { calls = append(calls, "activate") },
				Menu:       []desktop.TrayMenuEntry{desktop.NewTrayMenuItem("Show", func() { calls = append(calls, "show") })},
			}))
			fd.WaitFor("the tray", fd.TrayRegistered)
			if got := fd.TrayMenu(); !reflect.DeepEqual(got, []string{"Show"}) {
				t.Errorf("menu = %q, want [Show]", got)
			}
			if got := fd.TrayProperty("Title"); got != "Tray on "+p.String() {
				t.Errorf("Title = %v", got)
			}
			fd.ClickTrayMenuItem("Show")
			fd.ActivateTray()
			if want := []string{"show", "activate"}; !reflect.DeepEqual(calls, want) {
				t.Errorf("calls = %v, want %v", calls, want)
			}
		})
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

func TestFakeDesktopWithoutAFileChooser(t *testing.T) {
	tests := []struct {
		name string
		call func(ctx context.Context, bc widgets.BuildContext) error
		// route is the path from the function the app calls down to the tag.
		route string
	}{
		{"open files", func(ctx context.Context, bc widgets.BuildContext) error {
			_, err := desktop.OpenFiles(ctx, bc)
			return err
		}, "OpenFiles->call.chooseFiles.FileChooserUnavailable"},
		{"open a folder", func(ctx context.Context, bc widgets.BuildContext) error {
			_, err := desktop.OpenFiles(ctx, bc, desktop.OpenFilesOpts{Directory: true})
			return err
		}, "OpenFiles->call.chooseFiles.FileChooserUnavailable"},
		{"save a file", func(ctx context.Context, bc widgets.BuildContext) error {
			_, err := desktop.SaveFile(ctx, bc)
			return err
		}, "SaveFile->call.chooseFiles.FileChooserUnavailable"},
		{"save files", func(ctx context.Context, bc widgets.BuildContext) error {
			_, err := desktop.SaveFiles(ctx, bc, []string{"a.txt"})
			return err
		}, "SaveFiles->call.chooseFiles.FileChooserUnavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := New(t)
			fd := ts.FakeDesktop()
			fd.DisableFileChooser()
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("main")))
			bc := ts.Element(ByText("main"))
			errs := make(chan error, 1)
			go func() { errs <- tt.call(context.Background(), bc) }()
			var err error
			fd.WaitFor("the failure", func() bool {
				select {
				case err = <-errs:
					return true
				default:
					return false
				}
			})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["method"] == nil {
				t.Errorf("the error has no method in its metadata: %v", appErr.Metadata)
			}
			if fd.PendingFileChoosers() != 0 {
				t.Error("a dialog is open")
			}
		})
	}
}

func TestFakeDesktopFileChooserWorksUntilItIsDisabled(t *testing.T) {
	ts := New(t)
	fd := ts.FakeDesktop()
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("main")))
	bc := ts.Element(ByText("main"))
	// A dialog that is open stays open when the portal goes away, and can be answered.
	results := make(chan desktop.FilesResult, 1)
	go func() {
		res, err := desktop.OpenFiles(context.Background(), bc)
		if err != nil {
			t.Error(err)
		}
		results <- res
	}()
	fd.WaitFor("a dialog", func() bool { return fd.PendingFileChoosers() == 1 })
	fd.DisableFileChooser()
	fd.AnswerFileChooser("/d.txt")
	if res := <-waitResult(fd, results); !reflect.DeepEqual(res.Paths, []string{"/d.txt"}) {
		t.Errorf("result = %+v", res)
	}
}

// TestFakeDesktopFileChooserOptions reads back what the app asked of each file dialog (Theatro's
// issue of 1.17.9: its add dialog must open in the folder chosen last, which a test could not see).
func TestFakeDesktopFileChooserOptions(t *testing.T) {
	images := desktop.FileFilter{Name: "Images", Patterns: []string{"*.png", "*.PNG"}, MIMETypes: []string{"image/png"}}
	text := desktop.FileFilter{Name: "Text", Patterns: []string{"*.txt"}}
	tests := []struct {
		name string
		call func(ctx context.Context, bc widgets.BuildContext) (desktop.FilesResult, error)
		want FakeFileChooser
	}{
		{"OpenFiles without options", func(ctx context.Context, bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.OpenFiles(ctx, bc)
		}, FakeFileChooser{Method: "OpenFile", Modal: true}},
		{"OpenFiles with every option", func(ctx context.Context, bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.OpenFiles(ctx, bc, desktop.OpenFilesOpts{Title: "Add pictures", AcceptLabel: "Add",
				Filters: []desktop.FileFilter{images, text}, Multiple: true, CurrentFolder: "/home/u/Pictures 2", NotModal: true})
		}, FakeFileChooser{Method: "OpenFile", Title: "Add pictures", AcceptLabel: "Add", Multiple: true,
			Filters: []desktop.FileFilter{images, text}, CurrentFilter: "Images", CurrentFolder: "/home/u/Pictures 2"}},
		{"OpenFiles of folders", func(ctx context.Context, bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.OpenFiles(ctx, bc, desktop.OpenFilesOpts{Directory: true, CurrentFolder: "/srv/films",
				Filters: []desktop.FileFilter{text}})
		}, FakeFileChooser{Method: "OpenFile", Modal: true, Directory: true, CurrentFolder: "/srv/films"}},
		{"SaveFile", func(ctx context.Context, bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.SaveFile(ctx, bc, desktop.SaveFileOpts{Title: "Export", Filters: []desktop.FileFilter{text},
				CurrentName: "list.txt", CurrentFolder: "/tmp/out"})
		}, FakeFileChooser{Method: "SaveFile", Title: "Export", Modal: true, Filters: []desktop.FileFilter{text},
			CurrentFilter: "Text", CurrentName: "list.txt", CurrentFolder: "/tmp/out"}},
		{"SaveFile over a file", func(ctx context.Context, bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.SaveFile(ctx, bc, desktop.SaveFileOpts{CurrentFile: "/tmp/out/old.txt"})
		}, FakeFileChooser{Method: "SaveFile", Modal: true, CurrentFile: "/tmp/out/old.txt"}},
		{"SaveFiles", func(ctx context.Context, bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.SaveFiles(ctx, bc, []string{"a.txt", "b c.txt"}, desktop.SaveFilesOpts{AcceptLabel: "Save all",
				CurrentFolder: "/tmp/out", NotModal: true})
		}, FakeFileChooser{Method: "SaveFiles", AcceptLabel: "Save all", CurrentFolder: "/tmp/out", Names: []string{"a.txt", "b c.txt"}}},
	}
	for _, p := range everyTargetPlatform {
		t.Run(p.String(), func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					ts := New(t)
					runAs(t, p)
					fd := ts.FakeDesktop()
					ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("main")))
					bc := ts.Element(ByText("main"))
					results := make(chan desktop.FilesResult, 2)
					open := func() {
						go func() {
							res, err := tt.call(context.Background(), bc)
							if err != nil {
								t.Error(err)
							}
							results <- res
						}()
					}
					open()
					fd.WaitFor("a dialog", func() bool { return fd.PendingFileChoosers() == 1 })
					got := fd.FileChoosers()
					if len(got) != 1 || !reflect.DeepEqual(got[0], tt.want) {
						t.Fatalf("FileChoosers() = %+v, want [%+v]", got, tt.want)
					}
					// The list is a copy: a test that changes it changes nothing in the fake.
					got[0].Filters = append(got[0].Filters, desktop.FileFilter{Name: "x"})
					got[0].Names = append(got[0].Names, "x")
					got[0].Title = "changed"

					fd.AnswerFileChooser("/tmp/out/a.txt")
					<-waitResult(fd, results)
					open()
					fd.WaitFor("a second dialog", func() bool { return fd.PendingFileChoosers() == 1 })
					fd.CancelFileChooser()
					if res := <-waitResult(fd, results); !res.Cancelled {
						t.Errorf("result = %+v, want cancelled", res)
					}
					answered := tt.want
					answered.Answered = true
					got = fd.FileChoosers()
					if len(got) != 2 || !reflect.DeepEqual(got[0], answered) || !reflect.DeepEqual(got[1], answered) {
						t.Errorf("FileChoosers() after the answers = %+v, want two of %+v", got, answered)
					}
				})
			}
		})
	}
}

// TestFakeDesktopOpenURI opens URIs through the fake desktop's OpenURI portal with each of its answers
// (Theatro's issue of 1.17.9: the fake had no OpenURI portal, so desktop.OpenURI reached xdg-open).
func TestFakeDesktopOpenURI(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(fd *FakeDesktop)
		route   string // empty: OpenURI returns nil
		portal  string // the metadata "portalTag" of the error
		opened  bool   // the URI is among OpenedURIs
		program bool   // the cause of the error is ProgramNotStarted
	}{
		{"opens by default", func(*FakeDesktop) {}, "", "", true, false},
		{"opens", func(fd *FakeDesktop) { fd.SetOpenURIAnswer(FakeOpenURIAnswerOpens) }, "", "", true, false},
		{"the user cancels", func(fd *FakeDesktop) { fd.SetOpenURIAnswer(FakeOpenURIAnswerCancelled) }, "", "", true, false},
		{"the portal fails", func(fd *FakeDesktop) { fd.SetOpenURIAnswer(FakeOpenURIAnswerFails) },
			"OpenURI->call.openURI.OpenURIUnavailable", "OpenURIFailed", true, true},
		{"no portal", func(fd *FakeDesktop) { fd.DisableOpenURI() },
			"OpenURI->call.openURI.OpenURIUnavailable", "DBusCallFailed", false, true},
	}
	for _, p := range everyTargetPlatform {
		t.Run(p.String(), func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					ts := New(t)
					runAs(t, p)
					fd := ts.FakeDesktop()
					ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("main")))
					bc := ts.Element(ByText("main"))
					tt.setup(fd)
					const uri = "file:///srv/films/A%20Film/"
					res := make(chan error, 1)
					go func() { res <- desktop.OpenURI(context.Background(), bc, uri) }()
					var err error
					done := false
					fd.WaitFor("OpenURI", func() bool {
						select {
						case err = <-res:
							done = true
						default:
						}
						return done
					})
					if tt.route == "" {
						if err != nil {
							t.Fatalf("OpenURI = %v, want nil", err)
						}
					} else {
						appErr, ok := errors.FromError(err)
						if !ok {
							t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
						}
						if !appErr.HasRoute(tt.route) {
							t.Errorf("route = %s, want %s", appErr.Route(), tt.route)
						}
						if appErr.Metadata["portalTag"] != tt.portal || appErr.Metadata["uri"] != uri {
							t.Errorf("metadata = %v, want the portal's %s", appErr.Metadata, tt.portal)
						}
						if got := errors.Is(err, &errors.Error{Tag: "ProgramNotStarted"}); got != tt.program {
							t.Errorf("the cause is ProgramNotStarted: %v, want %v (%v)", got, tt.program, appErr.OriginalErr)
						}
					}
					var want []string
					if tt.opened {
						want = []string{uri}
					}
					if got := fd.OpenedURIs(); !reflect.DeepEqual(got, want) {
						t.Errorf("OpenedURIs() = %q, want %q", got, want)
					}
				})
			}
		})
	}
}
