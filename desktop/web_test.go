package desktop_test

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/platform/web"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// fakeWeb is the web's desktop: it records what package desktop asks of the browser and answers with the
// errors it is told to.
type fakeWeb struct {
	mu      sync.Mutex
	opened  []string
	shown   []web.NotificationSpec
	closed  []uint32
	openErr error
	showErr error
	nextID  uint32
}

func (f *fakeWeb) OpenURI(_ context.Context, uri string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opened = append(f.opened, uri)
	return f.openErr
}

func (f *fakeWeb) ShowNotification(_ context.Context, spec web.NotificationSpec) (uint32, error) {
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

func (f *fakeWeb) CloseNotification(_ context.Context, id uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = append(f.closed, id)
	return nil
}

func (f *fakeWeb) lastSpec(t *testing.T) web.NotificationSpec {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.shown) == 0 {
		t.Fatal("no notification was shown")
	}
	return f.shown[len(f.shown)-1]
}

// onTheWeb makes the app run as on the web, with a session bus that must never be asked for.
func onTheWeb(t *testing.T, bc widgets.BuildContext) *fakeWeb {
	t.Helper()
	desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) {
		t.Error("the web tried the session bus")
		return nil, errors.New(tagDBusConnectFailed, "must not be called")
	})
	f := &fakeWeb{}
	desktop.UseWeb(bc, f)
	return f
}

func TestOpenURIOnTheWeb(t *testing.T) {
	ts, bc := app(t)
	f := onTheWeb(t, bc)
	if err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "https://example.org/a?b=c") }); err != nil {
		t.Fatal(err)
	}
	if len(f.opened) != 1 || f.opened[0] != "https://example.org/a?b=c" {
		t.Errorf("the browser was asked to open %v", f.opened)
	}
	// The validation is the one of Linux, before the browser is asked.
	err := inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "example.org") })
	assertTag(t, err, "OpenURIInvalid")
	if len(f.opened) != 1 {
		t.Errorf("an invalid URI reached the browser: %v", f.opened)
	}
	// What the web says comes back untouched.
	f.openErr = errors.New(tagOpenURISchemeBlocked, "blocked")
	err = inGoroutine(t, ts, func() error { return desktop.OpenURI(context.Background(), bc, "javascript:alert(1)") })
	assertTag(t, err, "OpenURISchemeBlocked")
}

func TestShowNotificationOnTheWebMapsTheOptions(t *testing.T) {
	tests := []struct {
		name string
		opts desktop.NotificationOpts
		want web.NotificationSpec
	}{
		{"plain", desktop.NotificationOpts{Body: "a < b"}, web.NotificationSpec{Title: "Title", Body: "a < b"}},
		{"markup is stripped", desktop.NotificationOpts{Body: "<b>Hi</b> &amp; <a href=\"x\">bye</a>", BodyMarkup: true},
			web.NotificationSpec{Title: "Title", Body: "Hi & bye"}},
		{"a themed icon has no meaning", desktop.NotificationOpts{Icon: "dialog-information"}, web.NotificationSpec{Title: "Title"}},
		{"an icon URL stays", desktop.NotificationOpts{Icon: "/assets/icon.png"}, web.NotificationSpec{Title: "Title", Icon: "/assets/icon.png"}},
		{"critical stays until the user acts", desktop.NotificationOpts{Urgency: desktop.NotificationUrgencyCritical},
			web.NotificationSpec{Title: "Title", RequireInteraction: true}},
		{"low is silent", desktop.NotificationOpts{Urgency: desktop.NotificationUrgencyLow}, web.NotificationSpec{Title: "Title", Silent: true}},
		{"a timeout", desktop.NotificationOpts{Timeout: 3 * time.Second}, web.NotificationSpec{Title: "Title", Timeout: 3 * time.Second}},
		{"persistent has no timeout", desktop.NotificationOpts{Timeout: 3 * time.Second, Persistent: true},
			web.NotificationSpec{Title: "Title", RequireInteraction: true}},
		{"replaces", desktop.NotificationOpts{ReplacesID: 7}, web.NotificationSpec{Title: "Title", ReplacesID: 7}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, bc := app(t)
			f := onTheWeb(t, bc)
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
			if got.OnClick != nil || got.OnClose != nil {
				t.Error("callbacks were set for options that have none")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("the browser was asked for %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestShowNotificationOnTheWebCallsBack: the click is the action with the ID "default" and nothing else
// (the web cannot show buttons), and both callbacks arrive on the UI thread.
func TestShowNotificationOnTheWebCallsBack(t *testing.T) {
	ts, bc := app(t)
	f := onTheWeb(t, bc)
	var actions []string
	var reasons []desktop.NotificationCloseReason
	show := func(o desktop.NotificationOpts) web.NotificationSpec {
		o.OnAction = func(id string) { actions = append(actions, id) }
		o.OnClosed = func(r desktop.NotificationCloseReason) { reasons = append(reasons, r) }
		err := inGoroutine(t, ts, func() error {
			_, err := desktop.ShowNotification(context.Background(), bc, "Title", o)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return f.lastSpec(t)
	}

	// Only buttons: there is nothing to click.
	spec := show(desktop.NotificationOpts{Actions: []desktop.NotificationAction{{ID: "open", Label: "Open"}}})
	if spec.OnClick != nil {
		t.Error("a notification with only buttons listens to clicks")
	}

	spec = show(desktop.NotificationOpts{Actions: []desktop.NotificationAction{{ID: "open", Label: "Open"}, {ID: "default", Label: "Open the app"}}})
	if spec.OnClick == nil || spec.OnClose == nil {
		t.Fatal("the click and the close are not listened to")
	}
	// The browser calls from its own goroutine; the handlers run when the UI thread gets to them.
	go spec.OnClick()
	waitFor(t, ts, "the action", func() bool { return len(actions) == 1 })
	if actions[0] != "default" {
		t.Errorf("the action is %q, want default", actions[0])
	}
	for _, tc := range []struct {
		from web.CloseReason
		want desktop.NotificationCloseReason
	}{
		{web.CloseUnknown, desktop.NotificationClosedUnknown},
		{web.CloseExpired, desktop.NotificationClosedExpired},
		{web.CloseDismissed, desktop.NotificationClosedDismissed},
		{web.CloseByApp, desktop.NotificationClosedByApp},
	} {
		n := len(reasons)
		go spec.OnClose(tc.from)
		waitFor(t, ts, "the close", func() bool { return len(reasons) == n+1 })
		if reasons[n] != tc.want {
			t.Errorf("web reason %d became %v, want %v", tc.from, reasons[n], tc.want)
		}
	}
}

func TestShowNotificationOnTheWebErrorsComeBackUntouched(t *testing.T) {
	ts, bc := app(t)
	f := onTheWeb(t, bc)
	f.showErr = errors.New(tagNotificationsUnavailable, "the user did not allow the page to show notifications")
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
}

func TestCloseNotificationOnTheWeb(t *testing.T) {
	ts, bc := app(t)
	f := onTheWeb(t, bc)
	if err := inGoroutine(t, ts, func() error { return desktop.CloseNotification(context.Background(), bc, 5) }); err != nil {
		t.Fatal(err)
	}
	if len(f.closed) != 1 || f.closed[0] != 5 {
		t.Errorf("closed %v, want [5]", f.closed)
	}
}

// TestTheOtherServicesAreUnavailableOnTheWeb: the file dialogs and the tray return their Unavailable tags
// without trying D-Bus (SPEC §23.7).
func TestTheOtherServicesAreUnavailableOnTheWeb(t *testing.T) {
	ts, bc := app(t)
	onTheWeb(t, bc)
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
			if inner, ok := errors.FromError(appErr.OriginalErr); !ok || inner.Tag != "DBusUnavailableOnWeb" {
				t.Errorf("the cause is %v, want DBusUnavailableOnWeb", appErr.OriginalErr)
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
		if inner, ok := errors.FromError(appErr.OriginalErr); !ok || inner.Tag != "DBusUnavailableOnWeb" {
			t.Errorf("the cause is %v, want DBusUnavailableOnWeb", appErr.OriginalErr)
		}
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagNotificationsUnavailable = "NotificationsUnavailable"
	tagOpenURISchemeBlocked     = "OpenURISchemeBlocked"
)
