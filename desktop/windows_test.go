package desktop_test

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// fakeWindows is the desktop of Windows: it records what package desktop asks of the platform and answers
// with what it is told to. It has the notifications and URIs of fakeAndroid, the tray and the dialogs.
type fakeWindows struct {
	fakeAndroid

	mu       sync.Mutex
	trays    []*fakeTray
	trayErr  error
	dialogs  []platform.FileDialogSpec
	dialog   platform.FileDialogResult
	dialogEr error
}

func (f *fakeWindows) NewTray(spec platform.TraySpec) (platform.Tray, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.trayErr != nil {
		return nil, f.trayErr
	}
	t := &fakeTray{specs: []platform.TraySpec{spec}}
	f.trays = append(f.trays, t)
	return t, nil
}

func (f *fakeWindows) ShowFileDialog(_ context.Context, spec platform.FileDialogSpec) (platform.FileDialogResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dialogs = append(f.dialogs, spec)
	return f.dialog, f.dialogEr
}

// fakeTray is the tray icon of the platform.
type fakeTray struct {
	specs  []platform.TraySpec
	closed int
}

func (t *fakeTray) Update(spec platform.TraySpec) { t.specs = append(t.specs, spec) }
func (t *fakeTray) Close()                        { t.closed++ }
func (t *fakeTray) last() platform.TraySpec       { return t.specs[len(t.specs)-1] }

var _ desktop.WindowsDesktop = (*fakeWindows)(nil)

// onWindows makes the app run as on Windows, with a session bus that must never be asked for.
func onWindows(t *testing.T, bc widgets.BuildContext) *fakeWindows {
	t.Helper()
	desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) {
		t.Error("Windows tried the session bus")
		return nil, errors.New(tagDBusConnectFailed, "must not be called")
	})
	f := &fakeWindows{}
	desktop.UseWindows(bc, f)
	return f
}

func TestOpenURIOnWindows(t *testing.T) {
	ts, bc := app(t)
	f := onWindows(t, bc)
	if err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "https://example.org/a?b=c") }); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.opened, []string{"https://example.org/a?b=c"}) {
		t.Errorf("the platform was asked to open %v", f.opened)
	}
	// The validation of Linux comes first.
	err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "example.org") })
	assertTag(t, err, "OpenURIInvalid")
	// What the platform says comes back untouched.
	f.openErr = errors.New(tagOpenURISchemeBlocked, "blocked")
	err = inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, `C:\Windows\notepad.exe`) })
	assertTag(t, err, "OpenURISchemeBlocked")
	if len(f.opened) != 2 {
		t.Errorf("the platform got %v", f.opened)
	}
}

func TestShowNotificationOnWindowsMapsTheOptions(t *testing.T) {
	tests := []struct {
		name string
		opts desktop.NotificationOpts
		want mobile.NotificationSpec
	}{
		{"plain", desktop.NotificationOpts{Body: "a < b"}, mobile.NotificationSpec{Title: "Title", Body: "a < b"}},
		{"markup is stripped", desktop.NotificationOpts{Body: "<b>Hi</b> &amp; bye", BodyMarkup: true}, mobile.NotificationSpec{Title: "Title", Body: "Hi & bye"}},
		{"the icon goes to the balloon", desktop.NotificationOpts{Icon: `C:\icons\a.png`}, mobile.NotificationSpec{Title: "Title", Icon: `C:\icons\a.png`}},
		{"critical", desktop.NotificationOpts{Urgency: desktop.NotificationUrgencyCritical, Timeout: time.Second}, mobile.NotificationSpec{Title: "Title", Urgency: mobile.UrgencyCritical}},
		{"low", desktop.NotificationOpts{Urgency: desktop.NotificationUrgencyLow}, mobile.NotificationSpec{Title: "Title", Urgency: mobile.UrgencyLow}},
		{"a timeout", desktop.NotificationOpts{Timeout: 3 * time.Second}, mobile.NotificationSpec{Title: "Title", Timeout: 3 * time.Second}},
		{"persistent has no timeout", desktop.NotificationOpts{Timeout: 3 * time.Second, Persistent: true}, mobile.NotificationSpec{Title: "Title"}},
		{"replaces", desktop.NotificationOpts{ReplacesID: 7}, mobile.NotificationSpec{Title: "Title", ReplacesID: 7}},
		{"actions", desktop.NotificationOpts{Actions: []desktop.NotificationAction{{ID: "default", Label: "Open"}}},
			mobile.NotificationSpec{Title: "Title", Actions: []mobile.Action{{ID: "default", Label: "Open"}}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, bc := app(t)
			f := onWindows(t, bc)
			err := inGoroutine(t, ts, func() error {
				_, err := desktop.ShowNotification(context.Background(), bc, "Title", tc.opts)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			got := f.lastSpec(t)
			if got.OnAction != nil || got.OnClose != nil {
				t.Error("callbacks were set for options that have none")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("the platform was asked for %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestShowNotificationOnWindowsCallsBackOnTheUIThread(t *testing.T) {
	ts, bc := app(t)
	f := onWindows(t, bc)
	var actions []string
	var reasons []desktop.NotificationCloseReason
	err := inGoroutine(t, ts, func() error {
		_, err := desktop.ShowNotification(context.Background(), bc, "Title", desktop.NotificationOpts{
			Actions:  []desktop.NotificationAction{{ID: "default", Label: "Open"}},
			OnAction: func(id string) { actions = append(actions, id) },
			OnClosed: func(r desktop.NotificationCloseReason) { reasons = append(reasons, r) },
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	spec := f.lastSpec(t)
	go spec.OnAction("default")
	waitFor(t, ts, "the action", func() bool { return len(actions) == 1 })
	go spec.OnClose(mobile.CloseDismissed)
	waitFor(t, ts, "the close", func() bool { return len(reasons) == 1 })
	if actions[0] != "default" || reasons[0] != desktop.NotificationClosedDismissed {
		t.Errorf("action %v, reasons %v", actions, reasons)
	}
	// Errors come back untouched, and the checks of Linux come first.
	f.showErr = errors.New(tagWindowsBalloonFailed, "the shell did not take the balloon")
	err = inGoroutine(t, ts, func() error { _, err := desktop.ShowNotification(context.Background(), bc, "Title"); return err })
	assertTag(t, err, "WindowsBalloonFailed")
	err = inGoroutine(t, ts, func() error { _, err := desktop.ShowNotification(context.Background(), bc, ""); return err })
	assertTag(t, err, "NotificationSummaryEmpty")
	if err := inGoroutine(t, ts, func() error { return desktop.CloseNotification(context.Background(), bc, 5) }); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.closed, []uint32{5}) {
		t.Errorf("closed %v, want [5]", f.closed)
	}
}

func TestFileDialogsOnWindowsGetTheirSpecs(t *testing.T) {
	filters := []desktop.FileFilter{{Name: "Images", Patterns: []string{"*.png", "*.jpg"}, MIMETypes: []string{"image/png"}}, {Name: "All"}}
	wantFilters := []platform.FileFilter{{Name: "Images", Patterns: []string{"*.png", "*.jpg"}}, {Name: "All"}}
	tests := []struct {
		name string
		call func(bc widgets.BuildContext) (desktop.FilesResult, error)
		want platform.FileDialogSpec
	}{
		{"open", func(bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.OpenFiles(context.Background(), bc, desktop.OpenFilesOpts{Title: "Open", AcceptLabel: "Go", Filters: filters, Multiple: true, CurrentFolder: `C:\pics`})
		}, platform.FileDialogSpec{Kind: platform.FileDialogOpen, Title: "Open", AcceptLabel: "Go", Filters: wantFilters, Multiple: true, Folder: `C:\pics`}},
		{"open a folder takes no filters", func(bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.OpenFiles(context.Background(), bc, desktop.OpenFilesOpts{Directory: true, Filters: filters})
		}, platform.FileDialogSpec{Kind: platform.FileDialogOpen, Directory: true}},
		{"open without an owner", func(bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.OpenFiles(context.Background(), bc, desktop.OpenFilesOpts{NotModal: true})
		}, platform.FileDialogSpec{Kind: platform.FileDialogOpen, Owner: "-"}},
		{"save", func(bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.SaveFile(context.Background(), bc, desktop.SaveFileOpts{Title: "Save", Filters: filters, CurrentName: "a.png", CurrentFolder: `C:\out`})
		}, platform.FileDialogSpec{Kind: platform.FileDialogSave, Title: "Save", Filters: wantFilters, FileName: "a.png", Folder: `C:\out`}},
		{"save over a file that exists", func(bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.SaveFile(context.Background(), bc, desktop.SaveFileOpts{CurrentFile: "/home/me/report.pdf", CurrentName: "ignored"})
		}, platform.FileDialogSpec{Kind: platform.FileDialogSave, FileName: "report.pdf", Folder: "/home/me"}},
		{"save several", func(bc widgets.BuildContext) (desktop.FilesResult, error) {
			return desktop.SaveFiles(context.Background(), bc, []string{"a.txt", "b.txt"}, desktop.SaveFilesOpts{Title: "Into", CurrentFolder: `C:\out`})
		}, platform.FileDialogSpec{Kind: platform.FileDialogSaveMany, Title: "Into", Folder: `C:\out`, Names: []string{"a.txt", "b.txt"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := app(t)
			f := onWindows(t, bc)
			f.dialog = platform.FileDialogResult{Paths: []string{`C:\a.png`}, Filter: "Images"}
			var res desktop.FilesResult
			err := inGoroutine(t, ts, func() (err error) { res, err = tt.call(bc); return err })
			if err != nil {
				t.Fatal(err)
			}
			if len(f.dialogs) != 1 {
				t.Fatalf("%d dialogs were shown", len(f.dialogs))
			}
			got := f.dialogs[0]
			if tt.want.Owner == "-" {
				tt.want.Owner = ""
			} else {
				tt.want.Owner = got.Owner
				if !strings.HasPrefix(got.Owner, "test:") {
					t.Errorf("owner = %q: the window's handle from the platform", got.Owner)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("the dialog spec = %+v, want %+v", got, tt.want)
			}
			if !reflect.DeepEqual(res, desktop.FilesResult{Paths: []string{`C:\a.png`}, Filter: "Images"}) {
				t.Errorf("result = %+v", res)
			}
		})
	}
}

func TestFileDialogOnWindowsCancelAndErrors(t *testing.T) {
	ts, bc := app(t)
	f := onWindows(t, bc)
	f.dialog = platform.FileDialogResult{Cancelled: true}
	var res desktop.FilesResult
	err := inGoroutine(t, ts, func() (err error) { res, err = desktop.OpenFiles(context.Background(), bc); return err })
	if err != nil || !res.Cancelled || len(res.Paths) != 0 {
		t.Errorf("result %+v, err %v", res, err)
	}
	f.dialog = platform.FileDialogResult{}
	f.dialogEr = errors.New(tagFileChooserUnavailable, "the item dialog could not be made")
	err = inGoroutine(t, ts, func() error { _, err := desktop.SaveFile(context.Background(), bc); return err })
	assertTag(t, err, "FileChooserUnavailable")
	// The checks of SaveFiles come first.
	err = inGoroutine(t, ts, func() error { _, err := desktop.SaveFiles(context.Background(), bc, nil); return err })
	assertTag(t, err, "SaveFilesWithoutNames")
	err = inGoroutine(t, ts, func() error { _, err := desktop.SaveFiles(context.Background(), bc, []string{"a/b"}); return err })
	assertTag(t, err, "SaveFilesNameInvalid")
	if len(f.dialogs) != 2 {
		t.Errorf("%d dialogs reached the platform, want 2", len(f.dialogs))
	}
}

func menuLabels(items []platform.TrayMenuItem) []string {
	var out []string
	for _, it := range items {
		switch {
		case it.Separator:
			out = append(out, "-")
		case len(it.Children) > 0:
			out = append(out, it.Label+"["+strings.Join(menuLabels(it.Children), ",")+"]")
		default:
			out = append(out, it.Label)
		}
	}
	return out
}

func TestSystemTrayOnWindows(t *testing.T) {
	ts, bc := app(t)
	f := onWindows(t, bc)
	var pressed []string
	var activated, secondary int
	icon := ui.WindowIcon{Images: []ui.WindowIconImage{pixels(16, 255, 0, 0, 255)}}
	menu := []desktop.TrayMenuEntry{
		desktop.NewTrayMenuItem("Open", func() { pressed = append(pressed, "open") }),
		desktop.NewTrayMenuSeparator(),
		desktop.NewTrayMenuItem("Dark", nil, desktop.TrayMenuItemOpts{Checkable: true, Checked: true}),
		desktop.NewTrayMenuItem("Hidden", nil, desktop.TrayMenuItemOpts{Hidden: true, Disabled: true}),
		desktop.NewTrayMenuSubmenu("More", []desktop.TrayMenuEntry{
			desktop.NewTrayMenuItem("Sub", func() { pressed = append(pressed, "sub") }, desktop.TrayMenuItemOpts{Radio: true}),
		}),
	}
	opts := desktop.SystemTrayOpts{
		ID: "my-id", Title: "My App", Tooltip: "Tip", TooltipBody: "Body", Menu: menu,
		OnActivate: func() { activated++ }, OnSecondaryActivate: func() { secondary++ },
		Status: desktop.TrayStatusNeedsAttention, OnScroll: func(int, bool) {},
	}
	ts.PumpWidget(trayTree(icon, opts))
	if len(f.trays) != 1 {
		t.Fatalf("%d trays were made", len(f.trays))
	}
	tr := f.trays[0]
	spec := tr.last()
	if spec.ID != "my-id" || spec.Title != "My App" || spec.Tooltip != "Tip" || spec.TooltipBody != "Body" || !reflect.DeepEqual(spec.Icon, icon) {
		t.Errorf("spec = %+v", spec)
	}
	if got, want := menuLabels(spec.Menu), []string{"Open", "-", "Dark", "Hidden", "More[Sub]"}; !reflect.DeepEqual(got, want) {
		t.Errorf("menu = %v, want %v", got, want)
	}
	if m := spec.Menu; !m[2].Checkable || !m[2].Checked || !m[3].Hidden || !m[3].Disabled || !m[4].Children[0].Radio {
		t.Errorf("menu flags = %+v", m)
	}
	// The callbacks are the app's, called directly (the platform is on the UI thread).
	spec.OnActivate()
	spec.OnSecondaryActivate()
	spec.Menu[0].OnPressed()
	spec.Menu[4].Children[0].OnPressed()
	if activated != 1 || secondary != 1 || !reflect.DeepEqual(pressed, []string{"open", "sub"}) {
		t.Errorf("activate %d secondary %d pressed %v", activated, secondary, pressed)
	}
	if spec.Menu[2].OnPressed != nil || spec.Menu[4].OnPressed != nil {
		t.Error("an item with no function has one")
	}

	// A rebuild gives the platform the new model.
	opts.Tooltip = "New tip"
	opts.Menu = menu[:2]
	ts.PumpWidget(trayTree(icon, opts))
	if len(tr.specs) < 2 || tr.last().Tooltip != "New tip" || len(tr.last().Menu) != 2 {
		t.Fatalf("specs after the rebuild: %d, last %+v", len(tr.specs), tr.last())
	}
	if len(f.trays) != 1 || tr.closed != 0 {
		t.Errorf("trays %d, closed %d: the rebuild made a new tray", len(f.trays), tr.closed)
	}

	// The tray goes with the widget; then another one can be made.
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("no tray")))
	if tr.closed != 1 {
		t.Errorf("the icon was closed %d times", tr.closed)
	}
	ts.PumpWidget(trayTree(icon, desktop.SystemTrayOpts{}))
	if len(f.trays) != 2 {
		t.Errorf("%d trays after the second widget", len(f.trays))
	}
	if err := ts.TakeError(); err != nil {
		t.Errorf("an error was reported: %v", err)
	}
}

func TestSystemTrayOnWindowsHasOneTrayAtATime(t *testing.T) {
	ts, bc := app(t)
	f := onWindows(t, bc)
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColumn([]widgets.Widget{
		desktop.NewSystemTray(ui.WindowIcon{Name: "a"}, nil),
		desktop.NewSystemTray(ui.WindowIcon{Name: "b"}, nil),
	})))
	if len(f.trays) != 1 {
		t.Errorf("%d trays were made", len(f.trays))
	}
	err := ts.TakeError()
	appErr := tagged(t, err, "TrayDuplicate")
	if !appErr.HasRoute("newWinTray.TrayDuplicate") {
		t.Errorf("route %s", appErr.Route())
	}
}

func TestSystemTrayOnWindowsReportsWhatThePlatformSays(t *testing.T) {
	ts, bc := app(t)
	f := onWindows(t, bc)
	f.trayErr = errors.New(tagTrayRegisterFailed, "the shell did not take the icon of the tray")
	ts.PumpWidget(trayTree(ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{}))
	assertTag(t, ts.TakeError(), "TrayRegisterFailed")
	// A tray that failed is not asked for again, and the child is still built.
	ts.PumpWidget(trayTree(ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{Tooltip: "again"}))
	if ts.TakeError() != nil || len(f.trays) != 0 {
		t.Errorf("a failed tray was asked for again (%d trays)", len(f.trays))
	}
	// And the next widget can make one: the failure released the reservation.
	f.trayErr = nil
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("none")))
	ts.PumpWidget(trayTree(ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{}))
	if len(f.trays) != 1 {
		t.Errorf("%d trays after the release", len(f.trays))
	}
}

// TestTheBusIsNeverTriedOnWindows: a service that has no counterpart over D-Bus on Windows ends with its own
// Unavailable tag, whose cause is DBusUnavailableOnWindows (SPEC §14.1).
func TestTheBusIsNeverTriedOnWindows(t *testing.T) {
	ts, bc := app(t)
	desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) {
		t.Error("Windows tried the session bus")
		return nil, errors.New(tagDBusConnectFailed, "must not be called")
	})
	desktop.UseWindows(bc, &fakeWindows{})
	_ = ts
	for _, tag := range []string{"TrayUnavailable", "FileChooserUnavailable", "NotificationsUnavailable", "DBusConnectFailed"} {
		t.Run(tag, func(t *testing.T) {
			err := desktop.ConnectForTest(context.Background(), bc, tag)
			appErr := tagged(t, err, tag)
			inner, ok := errors.FromError(appErr.OriginalErr)
			if !ok || inner.Tag != "DBusUnavailableOnWindows" {
				t.Errorf("the cause is %v, want DBusUnavailableOnWindows", appErr.OriginalErr)
			}
			if !appErr.HasRoute("services.connect." + tag) {
				t.Errorf("route %s", appErr.Route())
			}
		})
	}
}

// TestTheWindowsServicesComeFromThePlatform: the binding gets the services of Windows from the engine, by the
// name of the service, only when what the engine hands out has a tray and file dialogs; Android's services
// are not taken for them.
func TestTheWindowsServicesComeFromThePlatform(t *testing.T) {
	tests := []struct {
		name        string
		service     any
		wantWindows bool
		wantAndroid bool
	}{
		{"the platform of Windows", &fakeWindows{}, true, false},
		{"the platform of Android", &fakeAndroid{}, false, true},
		{"the real mobile desktop", mobile.NewDesktop(nil, nil, nil), false, true},
		{"the platform has none", nil, false, false},
		{"something else under the name", "not a desktop", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := ui.NewPlatformDispatcher(&serviceHost{service: tt.service})
			b := widgets.NewWidgetsBinding(context.Background(), d)
			if got := desktop.HasWindows(b); got != tt.wantWindows {
				t.Errorf("services of Windows = %v, want %v", got, tt.wantWindows)
			}
			if got := desktop.HasAndroid(b); got != tt.wantAndroid {
				t.Errorf("services of Android = %v, want %v", got, tt.wantAndroid)
			}
		})
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTrayRegisterFailed     = "TrayRegisterFailed"
	tagFileChooserUnavailable = "FileChooserUnavailable"
	tagWindowsBalloonFailed   = "WindowsBalloonFailed"
)
