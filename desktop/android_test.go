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
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// fakeAndroid is the desktop of Android: it records what package desktop asks of the platform and
// answers with the errors it is told to.
type fakeAndroid struct {
	mu      sync.Mutex
	opened  []string
	shown   []mobile.NotificationSpec
	closed  []uint32
	openErr error
	showErr error
	nextID  uint32
}

func (f *fakeAndroid) OpenURI(_ context.Context, uri string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opened = append(f.opened, uri)
	return f.openErr
}

func (f *fakeAndroid) ShowNotification(_ context.Context, spec mobile.NotificationSpec) (uint32, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shown = append(f.shown, spec)
	if f.showErr != nil {
		return 0, f.showErr
	}
	if spec.ReplacesID != 0 {
		return spec.ReplacesID, nil
	}
	f.nextID++
	return f.nextID, nil
}

func (f *fakeAndroid) CloseNotification(_ context.Context, id uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = append(f.closed, id)
	return nil
}

func (f *fakeAndroid) lastSpec(t *testing.T) mobile.NotificationSpec {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.shown) == 0 {
		t.Fatal("no notification was shown")
	}
	return f.shown[len(f.shown)-1]
}

// onAndroid makes the app run as on Android, with a session bus that must never be asked for.
func onAndroid(t *testing.T, bc widgets.BuildContext) *fakeAndroid {
	t.Helper()
	desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) {
		t.Error("Android tried the session bus")
		return nil, errors.New(tagDBusConnectFailed, "must not be called")
	})
	f := &fakeAndroid{}
	desktop.UseAndroid(bc, f)
	return f
}

func TestOpenURIOnAndroid(t *testing.T) {
	ts, bc := app(t)
	f := onAndroid(t, bc)
	if err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "https://example.org/a?b=c") }); err != nil {
		t.Fatal(err)
	}
	if len(f.opened) != 1 || f.opened[0] != "https://example.org/a?b=c" {
		t.Errorf("the platform was asked to open %v", f.opened)
	}
	// The validation is the one of Linux, before the platform is asked.
	err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "example.org") })
	assertTag(t, err, "OpenURIInvalid")
	err = inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "a\nb:c") })
	assertTag(t, err, "OpenURIInvalid")
	if len(f.opened) != 1 {
		t.Errorf("an invalid URI reached the platform: %v", f.opened)
	}
	// What the platform says comes back untouched.
	f.openErr = errors.New(tagOpenURIUnavailable, "no app can open glutter-nothing://x")
	err = inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "glutter-nothing://x") })
	assertTag(t, err, "OpenURIUnavailable")
	f.openErr = errors.New(tagOpenURISchemeBlocked, "blocked")
	err = inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "javascript:alert(1)") })
	assertTag(t, err, "OpenURISchemeBlocked")
}

func TestShowNotificationOnAndroidMapsTheOptions(t *testing.T) {
	tests := []struct {
		name string
		opts desktop.NotificationOpts
		want mobile.NotificationSpec
	}{
		{"plain", desktop.NotificationOpts{Body: "a < b"}, mobile.NotificationSpec{Title: "Title", Body: "a < b"}},
		{"markup is stripped", desktop.NotificationOpts{Body: "<b>Hi</b> &amp; <a href=\"x\">bye</a>", BodyMarkup: true},
			mobile.NotificationSpec{Title: "Title", Body: "Hi & bye"}},
		{"an icon has nowhere to go", desktop.NotificationOpts{Icon: "dialog-information", Category: "email.arrived", AppName: "Mail"},
			mobile.NotificationSpec{Title: "Title"}},
		{"critical", desktop.NotificationOpts{Urgency: desktop.NotificationUrgencyCritical, Timeout: time.Second},
			mobile.NotificationSpec{Title: "Title", Urgency: mobile.UrgencyCritical}},
		{"low", desktop.NotificationOpts{Urgency: desktop.NotificationUrgencyLow}, mobile.NotificationSpec{Title: "Title", Urgency: mobile.UrgencyLow}},
		{"normal", desktop.NotificationOpts{Urgency: desktop.NotificationUrgencyNormal}, mobile.NotificationSpec{Title: "Title"}},
		{"a timeout", desktop.NotificationOpts{Timeout: 3 * time.Second}, mobile.NotificationSpec{Title: "Title", Timeout: 3 * time.Second}},
		{"persistent has no timeout", desktop.NotificationOpts{Timeout: 3 * time.Second, Persistent: true}, mobile.NotificationSpec{Title: "Title"}},
		{"replaces", desktop.NotificationOpts{ReplacesID: 7}, mobile.NotificationSpec{Title: "Title", ReplacesID: 7}},
		{"actions", desktop.NotificationOpts{Actions: []desktop.NotificationAction{{ID: "reply", Label: "Reply"}, {ID: "default", Label: "Open"}}},
			mobile.NotificationSpec{Title: "Title", Actions: []mobile.Action{{ID: "reply", Label: "Reply"}, {ID: "default", Label: "Open"}}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, bc := app(t)
			f := onAndroid(t, bc)
			var id desktop.NotificationID
			err := inGoroutine(t, ts, func() (err error) {
				id, err = desktop.ShowNotification(context.Background(), bc, "Title", tc.opts)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			if id == 0 {
				t.Error("the ID is zero")
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

// TestShowNotificationOnAndroidCallsBack: both callbacks arrive on the UI thread, with the action's ID
// and the close reason as Android reports them.
func TestShowNotificationOnAndroidCallsBack(t *testing.T) {
	ts, bc := app(t)
	f := onAndroid(t, bc)
	var actions []string
	var reasons []desktop.NotificationCloseReason
	err := inGoroutine(t, ts, func() error {
		_, err := desktop.ShowNotification(context.Background(), bc, "Title", desktop.NotificationOpts{
			Actions:  []desktop.NotificationAction{{ID: "reply", Label: "Reply"}},
			OnAction: func(id string) { actions = append(actions, id) },
			OnClosed: func(r desktop.NotificationCloseReason) { reasons = append(reasons, r) },
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	spec := f.lastSpec(t)
	if spec.OnAction == nil || spec.OnClose == nil {
		t.Fatal("the action and the close are not listened to")
	}
	// The platform calls from its own thread; the handlers run when the UI thread gets to them.
	go spec.OnAction("reply")
	waitFor(t, ts, "the action", func() bool { return len(actions) == 1 })
	if actions[0] != "reply" {
		t.Errorf("the action is %q, want reply", actions[0])
	}
	for _, tc := range []struct {
		from mobile.CloseReason
		want desktop.NotificationCloseReason
	}{
		{mobile.CloseUnknown, desktop.NotificationClosedUnknown},
		{mobile.CloseExpired, desktop.NotificationClosedExpired},
		{mobile.CloseDismissed, desktop.NotificationClosedDismissed},
		{mobile.CloseByApp, desktop.NotificationClosedByApp},
	} {
		n := len(reasons)
		go spec.OnClose(tc.from)
		waitFor(t, ts, "the close", func() bool { return len(reasons) == n+1 })
		if reasons[n] != tc.want {
			t.Errorf("platform reason %d became %v, want %v", tc.from, reasons[n], tc.want)
		}
	}
}

func TestShowNotificationOnAndroidErrorsComeBackUntouched(t *testing.T) {
	ts, bc := app(t)
	f := onAndroid(t, bc)
	f.showErr = errors.New(tagNotificationsUnavailable, "the user did not allow the app to show notifications")
	err := inGoroutine(t, ts, func() error {
		_, err := desktop.ShowNotification(context.Background(), bc, "Title")
		return err
	})
	assertTag(t, err, "NotificationsUnavailable")
	// The checks that Linux makes come first.
	err = inGoroutine(t, ts, func() error {
		_, err := desktop.ShowNotification(context.Background(), bc, "")
		return err
	})
	assertTag(t, err, "NotificationSummaryEmpty")
	err = inGoroutine(t, ts, func() error {
		_, err := desktop.ShowNotification(context.Background(), bc, "x", desktop.NotificationOpts{Actions: []desktop.NotificationAction{{ID: "a"}}})
		return err
	})
	assertTag(t, err, "NotificationActionInvalid")
	if len(f.shown) != 1 {
		t.Errorf("%d notifications reached the platform, want only the first", len(f.shown))
	}
}

func TestCloseNotificationOnAndroid(t *testing.T) {
	ts, bc := app(t)
	f := onAndroid(t, bc)
	if err := inGoroutine(t, ts, func() error { return desktop.CloseNotification(context.Background(), bc, 5) }); err != nil {
		t.Fatal(err)
	}
	if len(f.closed) != 1 || f.closed[0] != 5 {
		t.Errorf("closed %v, want [5]", f.closed)
	}
}

// TestTheOtherServicesAreUnavailableOnAndroid: the file dialogs and the tray return their Unavailable
// tags without trying D-Bus (SPEC §24.11).
func TestTheOtherServicesAreUnavailableOnAndroid(t *testing.T) {
	ts, bc := app(t)
	onAndroid(t, bc)
	calls := map[string]func() error{
		"OpenFiles": func() error { _, err := desktop.OpenFiles(context.Background(), bc); return err },
		"SaveFile":  func() error { _, err := desktop.SaveFile(context.Background(), bc); return err },
		"SaveFiles": func() error { _, err := desktop.SaveFiles(context.Background(), bc, []string{"a.txt"}); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			err := inGoroutine(t, ts, call)
			appErr := tagged(t, err, "FileChooserUnavailable")
			if !appErr.HasRoute("chooseFiles->services.connect.FileChooserUnavailable") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			inner, ok := errors.FromError(appErr.OriginalErr)
			if !ok || inner.Tag != "DBusUnavailableOnAndroid" || !strings.Contains(inner.Message, "file chooser is not available on Android") {
				t.Errorf("the cause is %v, want DBusUnavailableOnAndroid saying the file chooser is not available", appErr.OriginalErr)
			}
		})
	}
	t.Run("tray", func(t *testing.T) {
		ts.PumpWidget(trayTree(ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{}))
		var err error
		waitFor(t, ts, "the error", func() bool { err = ts.TakeError(); return err != nil })
		appErr := tagged(t, err, "TrayUnavailable")
		if !appErr.HasRoute("tray.run->services.connect.TrayUnavailable") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
		inner, ok := errors.FromError(appErr.OriginalErr)
		if !ok || inner.Tag != "DBusUnavailableOnAndroid" || !strings.Contains(inner.Message, "no system tray") {
			t.Errorf("the cause is %v, want DBusUnavailableOnAndroid saying there is no system tray", appErr.OriginalErr)
		}
	})
}

// TestTheServicesComeFromThePlatform: the binding gets the services of Android from the engine, by the
// name of the service, and only when what the engine hands out is a desktop of Android.
func TestTheServicesComeFromThePlatform(t *testing.T) {
	svc := &fakeAndroid{}
	tests := []struct {
		name    string
		service any
		want    bool
	}{
		{"the platform has them", svc, true},
		{"the real implementation", mobile.NewDesktop(nil, nil, nil), true},
		{"the platform has none", nil, false},
		{"something else under the name", "not a desktop", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := ui.NewPlatformDispatcher(&serviceHost{service: tt.service})
			b := widgets.NewWidgetsBinding(context.Background(), d)
			if got := desktop.HasAndroid(b); got != tt.want {
				t.Errorf("services of Android = %v, want %v", got, tt.want)
			}
		})
	}
}

// serviceHost is an engine whose platform hands out one service under the name package desktop uses.
type serviceHost struct{ service any }

func (*serviceHost) ScheduleFrame()                        {}
func (*serviceHost) RenderView(ui.ViewID, *ui.Scene) error { return nil }
func (*serviceHost) RequestAppExit()                       {}
func (*serviceHost) Post(fn func())                        { fn() }
func (h *serviceHost) PlatformService(name string) any {
	if name == mobile.ServiceDesktop {
		return h.service
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagOpenURIUnavailable = "OpenURIUnavailable"
)
