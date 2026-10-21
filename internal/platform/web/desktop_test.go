package web

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

// fakeOpener records the calls of window.open.
type fakeOpener struct {
	calls [][3]string
	err   error
}

func (f *fakeOpener) Open(uri, target, features string) error {
	f.calls = append(f.calls, [3]string{uri, target, features})
	return f.err
}

// fakeNotes is a Notifications API whose permission prompt the test answers by hand.
type fakeNotes struct {
	mu         sync.Mutex
	sink       notificationSink
	permission string
	requests   []func(string, error)
	shown      []browserNotification
	serials    []int
	showErr    error
	closed     []int
}

func (f *fakeNotes) SetSink(s notificationSink) { f.sink = s }

func (f *fakeNotes) Permission() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.permission
}

func (f *fakeNotes) RequestPermission(done func(string, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, done)
}

func (f *fakeNotes) Show(n browserNotification, serial int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.showErr != nil {
		return f.showErr
	}
	f.shown, f.serials = append(f.shown, n), append(f.serials, serial)
	return nil
}

func (f *fakeNotes) Close(serial int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = append(f.closed, serial)
}

// answer settles the n-th request for the permission, once the call has made it.
func (f *fakeNotes) answer(t *testing.T, n int, permission string, err error) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		f.mu.Lock()
		if len(f.requests) > n {
			done := f.requests[n]
			if permission != "" {
				f.permission = permission
			}
			f.mu.Unlock()
			done(permission, err)
			return
		}
		f.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("the permission was never asked for")
		}
		time.Sleep(time.Millisecond)
	}
}

func (f *fakeNotes) shownCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.shown)
}

func (f *fakeNotes) serial(i int) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.serials[i]
}

// timers is a clock whose timers the test fires by hand.
type timers struct {
	mu   sync.Mutex
	fns  []func()
	durs []time.Duration
	stop []bool
}

func (c *timers) after(d time.Duration, fn func()) func() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fns, c.durs, c.stop = append(c.fns, fn), append(c.durs, d), append(c.stop, false)
	i := len(c.fns) - 1
	return func() {
		c.mu.Lock()
		c.stop[i] = true
		c.mu.Unlock()
	}
}

// fire runs the timer unless it was stopped.
func (c *timers) fire(i int) {
	c.mu.Lock()
	fn, stopped := c.fns[i], c.stop[i]
	c.mu.Unlock()
	if !stopped {
		fn()
	}
}

func newTestDesktop(permission string) (*Desktop, *fakeNotes, *fakeOpener, *timers) {
	n, o, c := &fakeNotes{permission: permission}, &fakeOpener{}, &timers{}
	return newDesktop(o, n, c.after), n, o, c
}

func TestOpenURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantTag string
		opened  bool
	}{
		{"a web page", "https://example.org/a?b=c#d", "", true},
		{"a mail link", "mailto:me@example.org", "", true},
		{"script", "javascript:alert(1)", "OpenURISchemeBlocked", false},
		{"script in capitals", "JavaScript:alert(1)", "OpenURISchemeBlocked", false},
		{"script with a tab inside, as a browser reads it", "java\tscript:alert(1)", "OpenURISchemeBlocked", false},
		{"script after control characters", "\x01 \x1fjavascript:alert(1)", "OpenURISchemeBlocked", false},
		{"vbscript", "vbscript:msgbox", "OpenURISchemeBlocked", false},
		{"javascript only in the path", "https://example.org/javascript:x", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, _, o, _ := newTestDesktop("granted")
			err := d.OpenURI(context.Background(), tc.uri)
			if tc.wantTag == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if got := tagOf(t, err); got != tc.wantTag {
				t.Fatalf("got %s, want %s", got, tc.wantTag)
			}
			if tc.opened != (len(o.calls) == 1) {
				t.Fatalf("window.open was called %d times", len(o.calls))
			}
			if tc.opened && o.calls[0] != [3]string{tc.uri, "_blank", "noopener"} {
				t.Errorf("window.open%q, want the URI, _blank and noopener", o.calls[0])
			}
		})
	}
}

func TestOpenURIFailures(t *testing.T) {
	d, _, o, _ := newTestDesktop("granted")
	o.err = errors.New("Thrown", "window.open threw")
	appErr, ok := errors.FromError(d.OpenURI(context.Background(), "https://example.org"))
	if !ok || appErr.Tag != "OpenURIUnavailable" || appErr.Metadata["reason"] == "" {
		t.Errorf("got %+v, want OpenURIUnavailable with a reason", appErr)
	}
	var none *Desktop
	if got := tagOf(t, none.OpenURI(context.Background(), "https://example.org")); got != "OpenURIUnavailable" {
		t.Errorf("a nil Desktop gave %s", got)
	}
	if got := tagOf(t, NewDesktop().OpenURI(context.Background(), "https://example.org")); got != "OpenURIUnavailable" {
		t.Errorf("outside a page (or with no opener) gave %s", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d, _, o, _ = newTestDesktop("granted")
	if got := tagOf(t, d.OpenURI(ctx, "https://example.org")); got != "OpenURIUnavailable" || len(o.calls) != 0 {
		t.Errorf("an ended context gave %s and %d calls", got, len(o.calls))
	}
}

func TestShowNotificationAsksForThePermissionOnce(t *testing.T) {
	d, n, _, _ := newTestDesktop("default")
	type result struct {
		id  uint32
		err error
	}
	got := make(chan result, 1)
	go func() {
		id, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi", Body: "there"})
		got <- result{id, err}
	}()
	n.answer(t, 0, "granted", nil)
	r := <-got
	if r.err != nil || r.id != 1 {
		t.Fatalf("got %d, %v; want the ID 1", r.id, r.err)
	}
	// The user has decided: no second prompt.
	if _, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Again"}); err != nil {
		t.Fatal(err)
	}
	if len(n.requests) != 1 || n.shownCount() != 2 {
		t.Errorf("%d requests and %d notifications, want 1 and 2", len(n.requests), n.shownCount())
	}
	if n.shown[0] != (browserNotification{Title: "Hi", Body: "there", Tag: "glutter-1"}) {
		t.Errorf("the browser got %+v", n.shown[0])
	}
}

func TestShowNotificationWithoutPermission(t *testing.T) {
	tests := []struct {
		name       string
		permission string
		answer     string // the user's answer to the prompt, when there is one
		answerErr  error
		wantReason string
	}{
		{"denied before", "denied", "", nil, ""},
		{"refused at the prompt", "default", "denied", nil, ""},
		{"the prompt dismissed", "default", "default", nil, ""},
		{"the request failed", "default", "", errors.New("Thrown", "requestPermission threw"), "requesting the permission failed"},
		{"no Notification in the page", permissionUnsupported, "", nil, noNotificationsReason},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, n, _, _ := newTestDesktop(tc.permission)
			got := make(chan error, 1)
			go func() {
				_, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi"})
				got <- err
			}()
			if tc.permission == "default" {
				n.answer(t, 0, tc.answer, tc.answerErr)
			}
			err := <-got
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != "NotificationsUnavailable" {
				t.Fatalf("got %v, want NotificationsUnavailable", err)
			}
			if !appErr.HasRoute("NotificationsUnavailable") {
				t.Errorf("route %s", appErr.Route())
			}
			if n.shownCount() != 0 {
				t.Error("a notification was shown without the permission")
			}
			if reason, _ := appErr.Metadata["reason"].(string); tc.wantReason != "" && !strings.Contains(reason, tc.wantReason) {
				t.Errorf("reason %q, want it to say %q", reason, tc.wantReason)
			}
		})
	}
}

func TestShowNotificationWaitsForThePromptUntilTheContextEnds(t *testing.T) {
	d, n, _, _ := newTestDesktop("default")
	ctx, cancel := context.WithCancel(context.Background())
	got := make(chan error, 1)
	go func() {
		_, err := d.ShowNotification(ctx, NotificationSpec{Title: "Hi"})
		got <- err
	}()
	for {
		n.mu.Lock()
		asked := len(n.requests)
		n.mu.Unlock()
		if asked > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	if tag := tagOf(t, <-got); tag != "NotificationPermissionAborted" {
		t.Errorf("got %s, want NotificationPermissionAborted", tag)
	}
	// An answer that comes after the wait ended does not block the browser's callback.
	n.answer(t, 0, "granted", nil)
}

func TestShowNotificationFailsWhenTheBrowserRefusesToConstructIt(t *testing.T) {
	d, n, _, _ := newTestDesktop("granted")
	n.showErr = errNotificationFailed("Illegal constructor")
	_, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi", Timeout: time.Second})
	if tag := tagOf(t, err); tag != "WebNotificationFailed" {
		t.Fatalf("got %s", tag)
	}
	if len(d.live) != 0 || len(d.bySerial) != 0 {
		t.Errorf("a notification that was never shown is kept: %v %v", d.live, d.bySerial)
	}
}

func TestShowNotificationReplacesByID(t *testing.T) {
	d, n, _, _ := newTestDesktop("granted")
	var closed []CloseReason
	spec := NotificationSpec{Title: "Download", OnClose: func(r CloseReason) { closed = append(closed, r) }}
	id1, _ := d.ShowNotification(context.Background(), spec)
	spec.Title, spec.ReplacesID = "Download 50%", id1
	id2, err := d.ShowNotification(context.Background(), spec)
	if err != nil || id2 != id1 {
		t.Fatalf("got %d, %v; want the ID %d again", id2, err, id1)
	}
	if n.shown[0].Tag != n.shown[1].Tag || n.shown[1].Title != "Download 50%" {
		t.Errorf("the tags %q and %q should be the same, so that the browser replaces", n.shown[0].Tag, n.shown[1].Tag)
	}
	// The browser closes the replaced one; the app is not told, and the new one is still there.
	n.sink.Closed(n.serial(0))
	if len(closed) != 0 || len(d.live) != 1 {
		t.Errorf("the replaced notification's close was reported (%v) or dropped the new one (%d live)", closed, len(d.live))
	}
	// An ID that is not shown makes a new notification.
	spec.ReplacesID = 99
	id3, _ := d.ShowNotification(context.Background(), spec)
	if id3 == 99 || id3 == id1 {
		t.Errorf("replacing an unknown ID gave %d", id3)
	}
}

func TestNotificationClickAndClose(t *testing.T) {
	tests := []struct {
		name  string
		do    func(d *Desktop, n *fakeNotes, id uint32, serial int)
		click bool
		want  []CloseReason
	}{
		{"closed by the browser", func(d *Desktop, n *fakeNotes, _ uint32, s int) { n.sink.Closed(s) }, false, []CloseReason{CloseUnknown}},
		{"a click, then the close of the browser", func(d *Desktop, n *fakeNotes, _ uint32, s int) {
			n.sink.Clicked(s)
			n.sink.Closed(s)
		}, true, []CloseReason{CloseDismissed}},
		{"closed by the app", func(d *Desktop, n *fakeNotes, id uint32, _ int) {
			if err := d.CloseNotification(context.Background(), id); err != nil {
				t.Error(err)
			}
		}, false, []CloseReason{CloseByApp}},
		{"closed by the app and then by the browser's event", func(d *Desktop, n *fakeNotes, id uint32, s int) {
			_ = d.CloseNotification(context.Background(), id)
			n.sink.Closed(s)
			_ = d.CloseNotification(context.Background(), id)
		}, false, []CloseReason{CloseByApp}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, n, _, _ := newTestDesktop("granted")
			var clicks int
			var reasons []CloseReason
			id, err := d.ShowNotification(context.Background(), NotificationSpec{
				Title: "Hi", OnClick: func() { clicks++ }, OnClose: func(r CloseReason) { reasons = append(reasons, r) },
			})
			if err != nil {
				t.Fatal(err)
			}
			tc.do(d, n, id, n.serial(0))
			if (clicks == 1) != tc.click {
				t.Errorf("%d clicks", clicks)
			}
			if len(reasons) != len(tc.want) || (len(reasons) == 1 && reasons[0] != tc.want[0]) {
				t.Errorf("close reasons %v, want %v", reasons, tc.want)
			}
			if len(d.live) != 0 || len(d.bySerial) != 0 {
				t.Errorf("the notification is still kept: %v %v", d.live, d.bySerial)
			}
		})
	}
	// An event for a notification that is not known does nothing.
	_, n, _, _ := newTestDesktop("granted")
	n.sink.Clicked(42)
	n.sink.Closed(42)
}

func TestCloseNotificationClosesInTheBrowser(t *testing.T) {
	d, n, _, _ := newTestDesktop("granted")
	id, _ := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi"})
	if err := d.CloseNotification(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if len(n.closed) != 1 || n.closed[0] != n.serial(0) {
		t.Errorf("Notification.close was called for %v", n.closed)
	}
	// One that is gone is not an error, and the browser is not asked again.
	if err := d.CloseNotification(context.Background(), id); err != nil || len(n.closed) != 1 {
		t.Errorf("closing a gone notification: %v, %d closes", err, len(n.closed))
	}
	// A page with no Notifications API.
	var none *Desktop
	if tag := tagOf(t, none.CloseNotification(context.Background(), 1)); tag != "NotificationsUnavailable" {
		t.Errorf("got %s", tag)
	}
	if _, err := none.ShowNotification(context.Background(), NotificationSpec{Title: "x"}); tagOf(t, err) != "NotificationsUnavailable" {
		t.Errorf("a nil Desktop showed a notification: %v", err)
	}
}

func TestNotificationTimeout(t *testing.T) {
	t.Run("expires", func(t *testing.T) {
		d, n, _, c := newTestDesktop("granted")
		var reasons []CloseReason
		_, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi", Timeout: 3 * time.Second, OnClose: func(r CloseReason) { reasons = append(reasons, r) }})
		if err != nil {
			t.Fatal(err)
		}
		if len(c.durs) != 1 || c.durs[0] != 3*time.Second {
			t.Fatalf("timers %v, want one of 3s", c.durs)
		}
		c.fire(0)
		if len(reasons) != 1 || reasons[0] != CloseExpired || len(n.closed) != 1 {
			t.Errorf("reasons %v, %d closes; want CloseExpired and the browser told to close", reasons, len(n.closed))
		}
	})
	t.Run("a close stops the timer", func(t *testing.T) {
		d, _, _, c := newTestDesktop("granted")
		var reasons []CloseReason
		id, _ := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi", Timeout: time.Second, OnClose: func(r CloseReason) { reasons = append(reasons, r) }})
		_ = d.CloseNotification(context.Background(), id)
		c.fire(0)
		if len(reasons) != 1 || reasons[0] != CloseByApp {
			t.Errorf("reasons %v, want only CloseByApp", reasons)
		}
	})
	t.Run("a replacement stops the old timer and gets its own", func(t *testing.T) {
		d, n, _, c := newTestDesktop("granted")
		id, _ := d.ShowNotification(context.Background(), NotificationSpec{Title: "a", Timeout: time.Second})
		_, _ = d.ShowNotification(context.Background(), NotificationSpec{Title: "b", ReplacesID: id, Timeout: 2 * time.Second})
		c.fire(0) // the old one: stopped
		if len(n.closed) != 0 {
			t.Error("the stopped timer of the replaced notification closed the new one")
		}
		c.fire(1)
		if len(n.closed) != 1 {
			t.Error("the new timer did not close its notification")
		}
	})
	t.Run("none without a timeout", func(t *testing.T) {
		d, _, _, c := newTestDesktop("granted")
		_, _ = d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi"})
		if len(c.durs) != 0 {
			t.Errorf("timers %v", c.durs)
		}
	})
}

func TestStripMarkup(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain", "plain"},
		{"<b>bold</b> and <i>italic</i>", "bold and italic"},
		{`see <a href="https://example.org">this</a>`, "see this"},
		{"a &lt; b &amp;&amp; c &gt; d", "a < b && c > d"},
		{`say &quot;hi&quot; &apos;now&apos;`, `say "hi" 'now'`},
		{"<img src=\"x.png\" alt=\"x\"/>after", "after"},
		{"&amp;lt;", "&lt;"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := StripMarkup(tc.in); got != tc.want {
			t.Errorf("StripMarkup(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIconURL(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"dialog-information", ""},
		{"icon.png", ""},
		{"/icons/a.png", "/icons/a.png"},
		{"https://example.org/a.png", "https://example.org/a.png"},
		{"data:image/png;base64,AAAA", "data:image/png;base64,AAAA"},
		{"https://example.org/%zz", ""},
	}
	for _, tc := range tests {
		if got := IconURL(tc.in); got != tc.want {
			t.Errorf("IconURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
