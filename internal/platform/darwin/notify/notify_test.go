package notify

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
)

// fakeCenter is a Backend that records what it is asked.
type fakeCenter struct {
	mu        sync.Mutex
	notReady  string
	grant     bool
	authErr   error
	authCalls int
	cats      [][]Category
	added     []Request
	removed   []string
	addErr    error
}

func (f *fakeCenter) Ready() (bool, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.notReady == "", f.notReady
}

func (f *fakeCenter) Authorize(context.Context) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.authCalls++
	return f.grant, f.authErr
}

func (f *fakeCenter) SetCategories(c []Category) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cats = append(f.cats, c)
	return nil
}

func (f *fakeCenter) Add(_ context.Context, r Request) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.addErr != nil {
		return f.addErr
	}
	f.added = append(f.added, r)
	return nil
}

func (f *fakeCenter) Remove(id string) {
	f.mu.Lock()
	f.removed = append(f.removed, id)
	f.mu.Unlock()
}

// fakeClock is the timer of a notifier that the test fires.
type fakeClock struct {
	mu     sync.Mutex
	timers []*fakeTimer
}

type fakeTimer struct {
	d       time.Duration
	fn      func()
	stopped bool
}

func (c *fakeClock) after(d time.Duration, fn func()) func() bool {
	t := &fakeTimer{d: d, fn: fn}
	c.mu.Lock()
	c.timers = append(c.timers, t)
	c.mu.Unlock()
	return func() bool { c.mu.Lock(); defer c.mu.Unlock(); was := !t.stopped; t.stopped = true; return was }
}

func (c *fakeClock) fireAll() {
	c.mu.Lock()
	ts := append([]*fakeTimer(nil), c.timers...)
	c.mu.Unlock()
	for _, t := range ts {
		if !t.stopped {
			t.fn()
		}
	}
}

func newTestNotifier(b Backend) (*Notifier, *fakeClock) {
	n := New(b)
	c := &fakeClock{}
	n.after = c.after
	return n, c
}

func routeOf(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute(route) {
		t.Errorf("error %v, want the route %s", err, route)
	}
}

func TestNotificationNeedsABundleAndThePermission(t *testing.T) {
	ctx := context.Background()
	t.Run("no bundle", func(t *testing.T) {
		n, _ := newTestNotifier(&fakeCenter{notReady: "the app has no bundle identifier"})
		_, err := n.Show(ctx, mobile.NotificationSpec{Title: "x"})
		routeOf(t, err, "Notifier.Show."+tagNotificationsUnavailable)
	})
	t.Run("refused, asked once", func(t *testing.T) {
		f := &fakeCenter{grant: false}
		n, _ := newTestNotifier(f)
		_, err := n.Show(ctx, mobile.NotificationSpec{Title: "x"})
		routeOf(t, err, "Notifier.authorize."+tagNotificationsUnavailable)
		_, err = n.Show(ctx, mobile.NotificationSpec{Title: "y"})
		routeOf(t, err, "Notifier.authorize."+tagNotificationsUnavailable)
		if f.authCalls != 1 {
			t.Errorf("the permission was asked %d times, want once", f.authCalls)
		}
	})
	t.Run("the permission cannot be asked", func(t *testing.T) {
		n, _ := newTestNotifier(&fakeCenter{authErr: errors.New(tagNotificationsUnavailable, "boom")})
		_, err := n.Show(ctx, mobile.NotificationSpec{Title: "x"})
		routeOf(t, err, tagNotificationsUnavailable)
	})
	t.Run("the center refuses", func(t *testing.T) {
		n, _ := newTestNotifier(&fakeCenter{grant: true, addErr: errors.New(tagNotificationsUnavailable, "no")})
		_, err := n.Show(ctx, mobile.NotificationSpec{Title: "x"})
		routeOf(t, err, tagNotificationsUnavailable)
	})
}

func TestNotificationLife(t *testing.T) {
	ctx := context.Background()
	f := &fakeCenter{grant: true}
	n, clock := newTestNotifier(f)
	var mu sync.Mutex
	var log []string
	rec := func(s string) { mu.Lock(); log = append(log, s); mu.Unlock() }
	spec := mobile.NotificationSpec{
		Title: "Hello", Body: "World", Urgency: mobile.UrgencyLow, Timeout: 5 * time.Second,
		Actions:  []mobile.Action{{ID: mobile.DefaultAction, Label: ""}, {ID: "reply", Label: "Reply"}},
		OnAction: func(id string) { rec("action:" + id) },
		OnClose:  func(r mobile.CloseReason) { rec("close:" + itoa(int(r))) },
	}
	id, err := n.Show(ctx, spec)
	if err != nil || id != 1 {
		t.Fatalf("show = %d, %v", id, err)
	}
	if len(f.added) != 1 || f.added[0].Identifier != "glutter.1" || !f.added[0].Quiet || f.added[0].Title != "Hello" || f.added[0].Body != "World" {
		t.Errorf("request = %+v", f.added)
	}
	if len(f.cats) != 1 || len(f.cats[0]) != 1 || len(f.cats[0][0].Actions) != 1 || f.cats[0][0].Actions[0].ID != "reply" {
		t.Errorf("categories = %+v (the default action is no button)", f.cats)
	}
	// A second notification with the same buttons shares the category: the set is not set again.
	id2, _ := n.Show(ctx, mobile.NotificationSpec{Title: "Two", Actions: spec.Actions})
	if id2 != 2 || len(f.cats) != 1 {
		t.Errorf("second notification: id %d, %d category sets", id2, len(f.cats))
	}
	// The click on the notification and on a button.
	n.Response("glutter.2", mobile.DefaultAction) // not listed as an action of this notification? it is: "default"
	n.Response("glutter.1", "reply")
	mu.Lock()
	got := append([]string(nil), log...)
	mu.Unlock()
	if !reflect.DeepEqual(got, []string{"action:reply", "close:2"}) {
		t.Errorf("log = %v (the second notification had no callbacks; the first: its action, then dismissed)", got)
	}
	if !contains(f.removed, "glutter.1") {
		t.Errorf("a button left the notification in the center: %v", f.removed)
	}
	// A response to a notification that is gone does nothing.
	n.Response("glutter.1", "reply")
	n.Response("other.identifier", "reply")
	mu.Lock()
	n2 := len(log)
	mu.Unlock()
	if n2 != 2 {
		t.Errorf("a response to a gone notification ran callbacks: %v", log)
	}
	// The timer of a notification that was answered does nothing.
	clock.fireAll()
	mu.Lock()
	n3 := len(log)
	mu.Unlock()
	if n3 != 2 {
		t.Errorf("the timer of an answered notification ran: %v", log)
	}
}

func TestNotificationTimeoutReplaceAndClose(t *testing.T) {
	ctx := context.Background()
	f := &fakeCenter{grant: true}
	n, clock := newTestNotifier(f)
	var mu sync.Mutex
	var log []string
	rec := func(s string) { mu.Lock(); log = append(log, s); mu.Unlock() }
	mk := func(name string, timeout time.Duration, replaces uint32) mobile.NotificationSpec {
		return mobile.NotificationSpec{Title: name, Timeout: timeout, ReplacesID: replaces,
			OnClose: func(r mobile.CloseReason) { rec(name + ":" + itoa(int(r))) }}
	}
	id, _ := n.Show(ctx, mk("a", time.Second, 0))
	// Replacing keeps the number, does not close the old one, and the old timer does nothing.
	id2, _ := n.Show(ctx, mk("b", 0, id))
	if id2 != id {
		t.Errorf("a replacement has the number %d, want %d", id2, id)
	}
	if f.added[1].Identifier != f.added[0].Identifier {
		t.Errorf("a replacement has another identifier: %v", f.added)
	}
	clock.fireAll()
	mu.Lock()
	if len(log) != 0 {
		t.Errorf("the replaced notification's timer closed it: %v", log)
	}
	mu.Unlock()
	// The timeout of a notification that stays.
	id3, _ := n.Show(ctx, mk("c", time.Second, 0))
	clock.fireAll()
	mu.Lock()
	if !reflect.DeepEqual(log, []string{"c:1"}) {
		t.Errorf("log = %v, want c expired", log)
	}
	mu.Unlock()
	if !contains(f.removed, identifierOf(id3)) {
		t.Errorf("an expired notification stays in the center: %v", f.removed)
	}
	// The app closes b, once.
	n.Close(id)
	n.Close(id)
	mu.Lock()
	if !reflect.DeepEqual(log, []string{"c:1", "b:3"}) {
		t.Errorf("log = %v, want b closed by the app once", log)
	}
	mu.Unlock()
	n.Shutdown()
}

func TestIdentifiers(t *testing.T) {
	if idOfIdentifier(identifierOf(42)) != 42 || idOfIdentifier("glutter.x") != 0 || idOfIdentifier("x.5") != 0 || idOfIdentifier("glutter.99999999999") != 0 {
		t.Error("identifiers do not round trip")
	}
	a := categoryOf([]mobile.Action{{ID: "a", Label: "A"}})
	b := categoryOf([]mobile.Action{{ID: "a", Label: "A"}})
	c := categoryOf([]mobile.Action{{ID: "a", Label: "Other"}})
	d := categoryOf(nil)
	if a.ID != b.ID || a.ID == c.ID || a.ID == d.ID {
		t.Errorf("category ids: %s %s %s %s", a.ID, b.ID, c.ID, d.ID)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func itoa(i int) string {
	return string(rune('0' + i))
}
