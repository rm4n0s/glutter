package desktop_test

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	notifyMethod   = "org.freedesktop.Notifications.Notify"
	closeMethod    = "org.freedesktop.Notifications.CloseNotification"
	capsMethod     = "org.freedesktop.Notifications.GetCapabilities"
	portalAdd      = "org.freedesktop.portal.Notification.AddNotification"
	portalRemove   = "org.freedesktop.portal.Notification.RemoveNotification"
	notifyPath     = godbus.ObjectPath("/org/freedesktop/Notifications")
	notifyDest     = "org.freedesktop.Notifications"
	actionInvoked  = "org.freedesktop.Notifications.ActionInvoked"
	notifyClosed   = "org.freedesktop.Notifications.NotificationClosed"
	portalInvoked  = "org.freedesktop.portal.Notification.ActionInvoked"
	notifyServerID = 41
)

// notifyServer makes the fake bus act as a notification server with the capabilities.
func notifyServer(b *fakebus.Bus, caps ...string) {
	next := uint32(notifyServerID)
	var mu sync.Mutex
	b.Handle(notifyDest, notifyPath, notifyMethod, func(c fakebus.Call) ([]any, error) {
		mu.Lock()
		defer mu.Unlock()
		if replaces, _ := c.Args[1].(uint32); replaces != 0 {
			return []any{replaces}, nil
		}
		next++
		return []any{next}, nil
	})
	b.Handle(notifyDest, notifyPath, capsMethod, func(fakebus.Call) ([]any, error) { return []any{caps}, nil })
	b.Handle(notifyDest, notifyPath, closeMethod, func(fakebus.Call) ([]any, error) { return nil, nil })
}

// show calls ShowNotification as an app does.
func show(t *testing.T, ts *gluttertest.Tester, bc widgets.BuildContext, summary string, o desktop.NotificationOpts) desktop.NotificationID {
	t.Helper()
	var id desktop.NotificationID
	err := inGoroutine(t, ts, func() (err error) {
		id, err = desktop.ShowNotification(context.Background(), bc, summary, o)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// waitFor flushes the UI thread until cond holds.
func waitFor(t *testing.T, ts *gluttertest.Tester, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		ts.FlushPosted()
		time.Sleep(time.Millisecond)
	}
}

func lastCall(t *testing.T, b *fakebus.Bus, method string) fakebus.Call {
	t.Helper()
	calls := b.Calls()
	for i := len(calls) - 1; i >= 0; i-- {
		if calls[i].Method == method {
			return calls[i]
		}
	}
	t.Fatalf("no call of %s in %+v", method, calls)
	return fakebus.Call{}
}

func TestShowNotificationSendsNotify(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	notifyServer(b)
	desktop.UseBus(bc, b)

	id := show(t, ts, bc, "Build done", desktop.NotificationOpts{
		Body: "2 < 3 & 4 > 1", Icon: "dialog-information", Urgency: desktop.NotificationUrgencyCritical,
		Category: "transfer.complete", Timeout: 3 * time.Second,
		Actions: []desktop.NotificationAction{{ID: "open", Label: "Open"}, {ID: "undo", Label: "Undo"}},
	})
	if id != notifyServerID+1 {
		t.Errorf("id = %d, want the one of the server", id)
	}
	c := lastCall(t, b, notifyMethod)
	if c.Dest != notifyDest || c.Path != notifyPath {
		t.Errorf("went to %s %s", c.Dest, c.Path)
	}
	hints := c.Args[6].(map[string]godbus.Variant)
	want := []any{"Glutter", uint32(0), "dialog-information", "Build done", "2 < 3 & 4 > 1",
		[]string{"open", "Open", "undo", "Undo"}, hints, int32(3000)}
	if !reflect.DeepEqual(c.Args, want) {
		t.Errorf("args = %#v\nwant   %#v", c.Args, want)
	}
	if got := hints["urgency"].Value(); got != byte(2) {
		t.Errorf("urgency = %v, want 2", got)
	}
	if got := hints["category"].Value(); got != "transfer.complete" {
		t.Errorf("category = %v", got)
	}
}

func TestShowNotificationDefaults(t *testing.T) {
	tests := []struct {
		name    string
		opts    desktop.NotificationOpts
		urgency byte
		timeout int32
		appName string
	}{
		{"zero options", desktop.NotificationOpts{}, 1, -1, "Glutter"},
		{"low", desktop.NotificationOpts{Urgency: desktop.NotificationUrgencyLow}, 0, -1, "Glutter"},
		{"persistent beats the timeout", desktop.NotificationOpts{Persistent: true, Timeout: time.Second}, 1, 0, "Glutter"},
		{"app name", desktop.NotificationOpts{AppName: "Mail"}, 1, -1, "Mail"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := app(t)
			b := fakebus.New(unique)
			notifyServer(b)
			desktop.UseBus(bc, b)
			show(t, ts, bc, "s", tt.opts)
			c := lastCall(t, b, notifyMethod)
			if got := c.Args[0]; got != tt.appName {
				t.Errorf("app name = %v", got)
			}
			if got := c.Args[6].(map[string]godbus.Variant)["urgency"].Value(); got != tt.urgency {
				t.Errorf("urgency = %v, want %d", got, tt.urgency)
			}
			if got := c.Args[7]; got != tt.timeout {
				t.Errorf("timeout = %v, want %d", got, tt.timeout)
			}
		})
	}
}

func TestShowNotificationEscapesBodyMarkup(t *testing.T) {
	tests := []struct {
		name string
		caps []string
		opts desktop.NotificationOpts
		want string
	}{
		{"the server reads markup", []string{"body", "body-markup"}, desktop.NotificationOpts{Body: "a <b> & c"}, "a &lt;b&gt; &amp; c"},
		{"markup is asked for", []string{"body-markup"}, desktop.NotificationOpts{Body: "a <b>b</b>", BodyMarkup: true}, "a <b>b</b>"},
		{"the server reads plain text", []string{"body"}, desktop.NotificationOpts{Body: "a <b> & c"}, "a <b> & c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := app(t)
			b := fakebus.New(unique)
			notifyServer(b, tt.caps...)
			desktop.UseBus(bc, b)
			show(t, ts, bc, "s", tt.opts)
			if got := lastCall(t, b, notifyMethod).Args[4]; got != tt.want {
				t.Errorf("body = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShowNotificationReplaces(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	notifyServer(b)
	desktop.UseBus(bc, b)
	first := show(t, ts, bc, "one", desktop.NotificationOpts{})
	second := show(t, ts, bc, "two", desktop.NotificationOpts{ReplacesID: first})
	if second != first {
		t.Errorf("replacing gave %d, want %d", second, first)
	}
	if got := lastCall(t, b, notifyMethod).Args[1]; got != uint32(first) {
		t.Errorf("replaces_id = %v, want %d", got, first)
	}
}

func TestNotificationCallbacksRunOnTheUIThread(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	notifyServer(b)
	desktop.UseBus(bc, b)

	var actions []string
	var reasons []desktop.NotificationCloseReason
	id := show(t, ts, bc, "s", desktop.NotificationOpts{
		Actions:  []desktop.NotificationAction{{ID: "open", Label: "Open"}},
		OnAction: func(a string) { actions = append(actions, a) },
		OnClosed: func(r desktop.NotificationCloseReason) { reasons = append(reasons, r) },
	})
	other := show(t, ts, bc, "other", desktop.NotificationOpts{OnAction: func(string) { t.Error("the action of another notification") }})

	b.Emit(&godbus.Signal{Path: notifyPath, Name: actionInvoked, Body: []any{uint32(id), "open"}})
	waitFor(t, ts, "the action", func() bool { return len(actions) == 1 })
	if actions[0] != "open" {
		t.Errorf("action = %q", actions[0])
	}
	if len(reasons) != 0 {
		t.Errorf("an action closed the notification: %v", reasons)
	}

	b.Emit(&godbus.Signal{Path: notifyPath, Name: notifyClosed, Body: []any{uint32(id), uint32(2)}})
	waitFor(t, ts, "the close", func() bool { return len(reasons) == 1 })
	if reasons[0] != desktop.NotificationClosedDismissed {
		t.Errorf("reason = %v, want dismissed", reasons[0])
	}
	// The callbacks are dropped with the notification.
	b.Emit(&godbus.Signal{Path: notifyPath, Name: actionInvoked, Body: []any{uint32(id), "open"}})
	b.Emit(&godbus.Signal{Path: notifyPath, Name: notifyClosed, Body: []any{uint32(other), uint32(1)}})
	time.Sleep(20 * time.Millisecond)
	ts.FlushPosted()
	if len(actions) != 1 {
		t.Errorf("a closed notification got another action: %v", actions)
	}
}

func TestNotificationCloseReasons(t *testing.T) {
	tests := []struct {
		code uint32
		want desktop.NotificationCloseReason
	}{
		{1, desktop.NotificationClosedExpired},
		{2, desktop.NotificationClosedDismissed},
		{3, desktop.NotificationClosedByApp},
		{4, desktop.NotificationClosedUnknown},
		{99, desktop.NotificationClosedUnknown},
	}
	for _, tt := range tests {
		ts, bc := app(t)
		b := fakebus.New(unique)
		notifyServer(b)
		desktop.UseBus(bc, b)
		got, done := desktop.NotificationCloseReason(-1), false
		id := show(t, ts, bc, "s", desktop.NotificationOpts{OnClosed: func(r desktop.NotificationCloseReason) { got, done = r, true }})
		b.Emit(&godbus.Signal{Path: notifyPath, Name: notifyClosed, Body: []any{uint32(id), tt.code}})
		waitFor(t, ts, "the close", func() bool { return done })
		if got != tt.want {
			t.Errorf("code %d: reason = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestNotificationCallbackPanicIsReported(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	notifyServer(b)
	desktop.UseBus(bc, b)
	called := false
	id := show(t, ts, bc, "s", desktop.NotificationOpts{OnAction: func(string) { called = true; panic("boom") }})
	b.Emit(&godbus.Signal{Path: notifyPath, Name: actionInvoked, Body: []any{uint32(id), "x"}})
	waitFor(t, ts, "the action", func() bool { return called })
	assertTag(t, ts.TakeError(), "CallbackPanicked")
	// The next callback still arrives: the panic did not end the subscription.
	again := false
	id2 := show(t, ts, bc, "s", desktop.NotificationOpts{OnAction: func(string) { again = true }})
	b.Emit(&godbus.Signal{Path: notifyPath, Name: actionInvoked, Body: []any{uint32(id2), "x"}})
	waitFor(t, ts, "the second action", func() bool { return again })
}

func TestCloseNotification(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	notifyServer(b)
	desktop.UseBus(bc, b)
	id := show(t, ts, bc, "s", desktop.NotificationOpts{})
	if err := inGoroutine(t, ts, func() error { return desktop.CloseNotification(context.Background(), bc, id) }); err != nil {
		t.Fatal(err)
	}
	if got := lastCall(t, b, closeMethod).Args; !reflect.DeepEqual(got, []any{uint32(id)}) {
		t.Errorf("args = %#v", got)
	}
}

func TestNotificationsThroughThePortalInFlatpak(t *testing.T) {
	ts, bc := app(t)
	b := fakebus.New(unique)
	b.Handle(dbus.PortalDest, dbus.PortalPath, portalAdd, func(fakebus.Call) ([]any, error) { return nil, nil })
	b.Handle(dbus.PortalDest, dbus.PortalPath, portalRemove, func(fakebus.Call) ([]any, error) { return nil, nil })
	desktop.UseBus(bc, b)
	desktop.UseFlatpak(bc, true)

	var actions []string
	id := show(t, ts, bc, "Hello", desktop.NotificationOpts{
		Body: "<b>plain</b>", Icon: "mail", Urgency: desktop.NotificationUrgencyCritical,
		Actions:  []desktop.NotificationAction{{ID: "default", Label: "Open"}, {ID: "reply", Label: "Reply"}},
		OnAction: func(a string) { actions = append(actions, a) },
	})
	if id == 0 {
		t.Fatal("no id")
	}
	if got := lastCallCount(b, notifyMethod); got != 0 {
		t.Errorf("%d calls to the notification service inside the sandbox", got)
	}
	c := lastCall(t, b, portalAdd)
	name, _ := c.Args[0].(string)
	n := c.Args[1].(map[string]godbus.Variant)
	if name == "" || n["title"].Value() != "Hello" || n["body"].Value() != "<b>plain</b>" || n["priority"].Value() != "urgent" {
		t.Errorf("name %q, notification %#v", name, n)
	}
	if n["default-action"].Value() != "default" {
		t.Errorf("default-action = %v", n["default-action"])
	}
	buttons := n["buttons"].Value().([]map[string]godbus.Variant)
	if len(buttons) != 1 || buttons[0]["label"].Value() != "Reply" || buttons[0]["action"].Value() != "reply" {
		t.Errorf("buttons = %#v", buttons)
	}

	b.Emit(&godbus.Signal{Path: dbus.PortalPath, Name: portalInvoked, Body: []any{name, "reply", []godbus.Variant{}}})
	waitFor(t, ts, "the action", func() bool { return len(actions) == 1 })
	if actions[0] != "reply" {
		t.Errorf("action = %q", actions[0])
	}

	// Replacing keeps the name; closing removes it.
	show(t, ts, bc, "Hello again", desktop.NotificationOpts{ReplacesID: id})
	if again, _ := lastCall(t, b, portalAdd).Args[0].(string); again != name {
		t.Errorf("a replacement has the name %q, want %q", again, name)
	}
	if err := inGoroutine(t, ts, func() error { return desktop.CloseNotification(context.Background(), bc, id) }); err != nil {
		t.Fatal(err)
	}
	if got := lastCall(t, b, portalRemove).Args; !reflect.DeepEqual(got, []any{name}) {
		t.Errorf("RemoveNotification args = %#v", got)
	}
}

func lastCallCount(b *fakebus.Bus, method string) int {
	n := 0
	for _, c := range b.Calls() {
		if c.Method == method {
			n++
		}
	}
	return n
}

func TestShowNotificationErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(bc widgets.BuildContext)
		sum   string
		opts  desktop.NotificationOpts
		tag   string
	}{
		{"no summary", func(bc widgets.BuildContext) { desktop.UseBus(bc, fakebus.New(unique)) }, "", desktop.NotificationOpts{}, "NotificationSummaryEmpty"},
		{"an action without a label", func(bc widgets.BuildContext) { desktop.UseBus(bc, fakebus.New(unique)) }, "s",
			desktop.NotificationOpts{Actions: []desktop.NotificationAction{{ID: "a"}}}, "NotificationActionInvalid"},
		{"no service on the bus", func(bc widgets.BuildContext) { desktop.UseBus(bc, fakebus.New(unique)) }, "s", desktop.NotificationOpts{}, "NotificationsUnavailable"},
		{"no session bus", func(bc widgets.BuildContext) {
			desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) { return nil, errors.New("DBusConnectFailed", "no bus") })
		}, "s", desktop.NotificationOpts{}, "NotificationsUnavailable"},
		{"no portal in the sandbox", func(bc widgets.BuildContext) {
			desktop.UseBus(bc, fakebus.New(unique))
			desktop.UseFlatpak(bc, true)
		}, "s", desktop.NotificationOpts{}, "NotificationsUnavailable"},
		{"a reply that is not an id", func(bc widgets.BuildContext) {
			b := fakebus.New(unique)
			b.Handle(notifyDest, notifyPath, notifyMethod, func(fakebus.Call) ([]any, error) { return []any{"x"}, nil })
			desktop.UseBus(bc, b)
		}, "s", desktop.NotificationOpts{}, "NotificationReplyInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := app(t)
			tt.setup(bc)
			err := inGoroutine(t, ts, func() error {
				_, err := desktop.ShowNotification(context.Background(), bc, tt.sum, tt.opts)
				return err
			})
			assertTag(t, err, tt.tag)
		})
	}
}

func TestCloseNotificationUnavailable(t *testing.T) {
	ts, bc := app(t)
	desktop.UseBus(bc, fakebus.New(unique))
	err := inGoroutine(t, ts, func() error { return desktop.CloseNotification(context.Background(), bc, 3) })
	assertTag(t, err, "NotificationsUnavailable")
}

func TestShowNotificationOnTheUIThreadFails(t *testing.T) {
	_, bc := app(t)
	desktop.UseBus(bc, fakebus.New(unique))
	_, err := desktop.ShowNotification(context.Background(), bc, "s")
	assertTag(t, err, "DesktopCallOnUIThread")
}
