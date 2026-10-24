package mobile

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

// fakeOpener records the intents it was asked to start.
type fakeOpener struct {
	calls []string
	err   error
}

func (f *fakeOpener) Open(_ context.Context, uri string) error {
	f.calls = append(f.calls, uri)
	return f.err
}

// fakeNotes is the notification manager whose permission prompt the test answers by hand.
type fakeNotes struct {
	mu        sync.Mutex
	sink      Sink
	status    Status
	statusErr error
	requests  []func(Permission, error)
	shown     []Notification
	showErr   error
	canceled  []uint32
}

func (f *fakeNotes) SetSink(s Sink) { f.sink = s }

func (f *fakeNotes) Status(context.Context) (Status, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status, f.statusErr
}

func (f *fakeNotes) RequestPermission(done func(Permission, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, done)
}

func (f *fakeNotes) Show(_ context.Context, n Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.showErr != nil {
		return f.showErr
	}
	f.shown = append(f.shown, n)
	return nil
}

func (f *fakeNotes) Cancel(id uint32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.canceled = append(f.canceled, id)
}

// answer settles the n-th request for the permission, once the call has made it; like the system, a
// granted answer changes the status.
func (f *fakeNotes) answer(t *testing.T, n int, p Permission, err error) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		f.mu.Lock()
		if len(f.requests) > n {
			done := f.requests[n]
			if p == PermissionGranted {
				f.status.PermissionGranted, f.status.Enabled = true, true
			}
			f.mu.Unlock()
			done(p, err)
			return
		}
		f.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("the permission was never asked for")
		}
		time.Sleep(time.Millisecond)
	}
}

func (f *fakeNotes) requested() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fakeNotes) shownCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.shown)
}

func (f *fakeNotes) serial(i int) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.shown[i].Serial
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

func (c *timers) fire(i int) {
	c.mu.Lock()
	fn, stopped := c.fns[i], c.stop[i]
	c.mu.Unlock()
	if !stopped {
		fn()
	}
}

// api33 is a status of Android 13 or later: the permission is needed.
func api33(granted bool) Status {
	return Status{PermissionNeeded: true, PermissionGranted: granted, Enabled: granted}
}

// api26 is a status below Android 13: no permission is needed.
var api26 = Status{PermissionGranted: true, Enabled: true}

func newTestDesktop(st Status) (*Desktop, *fakeNotes, *fakeOpener, *timers) {
	n, o, c := &fakeNotes{status: st}, &fakeOpener{}, &timers{}
	return NewDesktop(o, n, c.after), n, o, c
}

func tagOf(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		return ""
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	return appErr.Tag
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
		{"a link no app handles is the opener's to refuse", "glutter-nothing://x", "", true},
		{"script", "javascript:alert(1)", "OpenURISchemeBlocked", false},
		{"script in capitals", "JavaScript:alert(1)", "OpenURISchemeBlocked", false},
		{"script with a tab inside", "java\tscript:alert(1)", "OpenURISchemeBlocked", false},
		{"script after control characters", "\x01 \x1fjavascript:alert(1)", "OpenURISchemeBlocked", false},
		{"vbscript", "vbscript:msgbox", "OpenURISchemeBlocked", false},
		{"javascript only in the path", "https://example.org/javascript:x", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, _, o, _ := newTestDesktop(api26)
			err := d.OpenURI(context.Background(), tc.uri)
			if got := tagOf(t, err); got != tc.wantTag {
				t.Fatalf("got %q, want %q (%v)", got, tc.wantTag, err)
			}
			if tc.opened != (len(o.calls) == 1) {
				t.Fatalf("the opener was called %d times", len(o.calls))
			}
			if tc.opened && o.calls[0] != tc.uri {
				t.Errorf("the intent was for %q, want %q", o.calls[0], tc.uri)
			}
			if tc.wantTag != "" {
				appErr, _ := errors.FromError(err)
				if !appErr.HasRoute("Desktop.OpenURI." + tc.wantTag) {
					t.Errorf("unexpected failure path: %s", appErr.Route())
				}
			}
		})
	}
}

func TestOpenURIUnavailable(t *testing.T) {
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name       string
		desktop    func() (*Desktop, *fakeOpener)
		ctx        context.Context
		wantReason string
		wantCalls  int
	}{
		{"no app handles it", func() (*Desktop, *fakeOpener) {
			d, _, o, _ := newTestDesktop(api26)
			o.err = Refused{"no app can open glutter-nothing://x"}
			return d, o
		}, context.Background(), "no app can open", 1},
		{"no activity", func() (*Desktop, *fakeOpener) {
			d, _, o, _ := newTestDesktop(api26)
			o.err = Refused{"no activity is shown to start the intent from"}
			return d, o
		}, context.Background(), "no activity", 1},
		{"a nil Desktop", func() (*Desktop, *fakeOpener) { return nil, nil }, context.Background(), "not running in an Android activity", 0},
		{"no opener", func() (*Desktop, *fakeOpener) { return NewDesktop(nil, nil, nil), nil }, context.Background(), "not running in an Android activity", 0},
		{"an ended context", func() (*Desktop, *fakeOpener) {
			d, _, o, _ := newTestDesktop(api26)
			return d, o
		}, ended, "the context ended", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, o := tc.desktop()
			err := d.OpenURI(tc.ctx, "glutter-nothing://x")
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tagOpenURIUnavailable {
				t.Fatalf("got %v, want OpenURIUnavailable", err)
			}
			if !appErr.HasRoute("Desktop.OpenURI.OpenURIUnavailable") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if reason, _ := appErr.Metadata["reason"].(string); !strings.Contains(reason, tc.wantReason) {
				t.Errorf("reason %q, want it to say %q", reason, tc.wantReason)
			}
			if appErr.Metadata["uri"] != "glutter-nothing://x" {
				t.Errorf("metadata %v has no uri", appErr.Metadata)
			}
			if o != nil && len(o.calls) != tc.wantCalls {
				t.Errorf("%d intents, want %d", len(o.calls), tc.wantCalls)
			}
		})
	}
}

func showAsync(d *Desktop, ctx context.Context, spec NotificationSpec) <-chan showResult {
	got := make(chan showResult, 1)
	go func() {
		id, err := d.ShowNotification(ctx, spec)
		got <- showResult{id, err}
	}()
	return got
}

type showResult struct {
	id  uint32
	err error
}

func TestShowNotificationNeedsNoPermissionBelowAndroid13(t *testing.T) {
	d, n, _, _ := newTestDesktop(api26)
	id, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi", Body: "there"})
	if err != nil || id != 1 {
		t.Fatalf("got %d, %v; want the ID 1", id, err)
	}
	if n.requested() != 0 {
		t.Error("the permission was asked for below Android 13")
	}
	want := Notification{ID: 1, Serial: 1, Title: "Hi", Body: "there"}
	if got := n.shown[0]; got.ID != want.ID || got.Serial != want.Serial || got.Title != want.Title || got.Body != want.Body || got.Urgency != UrgencyNormal {
		t.Errorf("the notification manager got %+v, want %+v", got, want)
	}
}

func TestShowNotificationAsksForThePermissionOncePerDecision(t *testing.T) {
	d, n, _, _ := newTestDesktop(api33(false))
	got := showAsync(d, context.Background(), NotificationSpec{Title: "Hi"})
	n.answer(t, 0, PermissionGranted, nil)
	r := <-got
	if r.err != nil || r.id != 1 {
		t.Fatalf("got %d, %v; want the ID 1", r.id, r.err)
	}
	// The user has decided: no second prompt.
	if _, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Again"}); err != nil {
		t.Fatal(err)
	}
	if n.requested() != 1 || n.shownCount() != 2 {
		t.Errorf("%d requests and %d notifications, want 1 and 2", n.requested(), n.shownCount())
	}
}

func TestShowNotificationWithoutPermission(t *testing.T) {
	tests := []struct {
		name       string
		status     Status
		answer     Permission
		answerErr  error
		prompt     bool
		wantReason string
	}{
		{"refused at the prompt", api33(false), PermissionDenied, nil, true, ""},
		{"the prompt interrupted", api33(false), PermissionUndecided, nil, true, "without an answer"},
		{"the request failed", api33(false), PermissionUndecided, Refused{"no activity is shown"}, true, "requesting the permission failed"},
		{"granted, but the notifications are off", Status{PermissionNeeded: true, PermissionGranted: true}, 0, nil, false, "areNotificationsEnabled"},
		{"below Android 13 the user turned them off", Status{PermissionGranted: true}, 0, nil, false, "areNotificationsEnabled"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, n, _, _ := newTestDesktop(tc.status)
			got := showAsync(d, context.Background(), NotificationSpec{Title: "Hi"})
			if tc.prompt {
				n.answer(t, 0, tc.answer, tc.answerErr)
			}
			err := (<-got).err
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tagNotificationsUnavailable {
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

func TestADenialIsNotAskedAgainInTheSameRun(t *testing.T) {
	d, n, _, _ := newTestDesktop(api33(false))
	got := showAsync(d, context.Background(), NotificationSpec{Title: "Hi"})
	n.answer(t, 0, PermissionDenied, nil)
	if tag := tagOf(t, (<-got).err); tag != "NotificationsUnavailable" {
		t.Fatalf("got %s", tag)
	}
	for i := 0; i < 2; i++ {
		_, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Again"})
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tagNotificationsUnavailable || !appErr.HasRoute("Desktop.ensurePermission->Desktop.askPermission.NotificationsUnavailable") {
			t.Fatalf("call %d: got %v", i, err)
		}
	}
	if n.requested() != 1 {
		t.Errorf("%d prompts, want 1", n.requested())
	}
	// The user grants it in the settings later: the system's word wins over the remembered refusal.
	n.mu.Lock()
	n.status = api33(true)
	n.mu.Unlock()
	if _, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Now"}); err != nil {
		t.Fatal(err)
	}
}

func TestAnUndecidedPromptIsAskedAgain(t *testing.T) {
	d, n, _, _ := newTestDesktop(api33(false))
	got := showAsync(d, context.Background(), NotificationSpec{Title: "Hi"})
	n.answer(t, 0, PermissionUndecided, nil)
	if tag := tagOf(t, (<-got).err); tag != "NotificationsUnavailable" {
		t.Fatalf("got %s", tag)
	}
	got = showAsync(d, context.Background(), NotificationSpec{Title: "Hi"})
	n.answer(t, 1, PermissionGranted, nil)
	if r := <-got; r.err != nil {
		t.Fatal(r.err)
	}
}

func TestShowNotificationWaitsForThePromptUntilTheContextEnds(t *testing.T) {
	d, n, _, _ := newTestDesktop(api33(false))
	ctx, cancel := context.WithCancel(context.Background())
	got := showAsync(d, ctx, NotificationSpec{Title: "Hi"})
	for n.requested() == 0 {
		time.Sleep(time.Millisecond)
	}
	cancel()
	err := (<-got).err
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagNotificationPermissionAborted || !appErr.HasRoute("Desktop.ensurePermission->Desktop.askPermission.NotificationPermissionAborted") {
		t.Fatalf("got %v, want NotificationPermissionAborted", err)
	}
	// An answer that comes after the wait ended is not lost: a refusal is remembered.
	n.answer(t, 0, PermissionDenied, nil)
	_, err = d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi"})
	if tag := tagOf(t, err); tag != "NotificationsUnavailable" || n.requested() != 1 {
		t.Errorf("got %s after %d prompts, want NotificationsUnavailable after 1", tag, n.requested())
	}
}

func TestShowNotificationStatusFails(t *testing.T) {
	d, n, _, _ := newTestDesktop(api26)
	n.statusErr = Refused{"the activity is gone"}
	_, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi"})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagNotificationsUnavailable || !appErr.HasRoute("Desktop.ShowNotification->Desktop.ensurePermission.NotificationsUnavailable") {
		t.Fatalf("got %v", err)
	}
	if reason, _ := appErr.Metadata["reason"].(string); !strings.Contains(reason, "the activity is gone") {
		t.Errorf("reason %q", reason)
	}
}

func TestShowNotificationMapsTheSpec(t *testing.T) {
	tests := []struct {
		name string
		spec NotificationSpec
		want Notification
	}{
		{"plain", NotificationSpec{Title: "T", Body: "B"}, Notification{ID: 1, Serial: 1, Title: "T", Body: "B"}},
		{"low", NotificationSpec{Title: "T", Urgency: UrgencyLow}, Notification{ID: 1, Serial: 1, Title: "T", Urgency: UrgencyLow}},
		{"critical", NotificationSpec{Title: "T", Urgency: UrgencyCritical}, Notification{ID: 1, Serial: 1, Title: "T", Urgency: UrgencyCritical}},
		{"a timeout", NotificationSpec{Title: "T", Timeout: 3 * time.Second}, Notification{ID: 1, Serial: 1, Title: "T", Timeout: 3 * time.Second}},
		{"buttons without the default action", NotificationSpec{Title: "T", Actions: []Action{{"reply", "Reply"}, {"default", "Open"}, {"mute", "Mute"}}},
			Notification{ID: 1, Serial: 1, Title: "T", Actions: []Action{{"reply", "Reply"}, {"mute", "Mute"}}}},
		{"only the default action", NotificationSpec{Title: "T", Actions: []Action{{"default", "Open"}}}, Notification{ID: 1, Serial: 1, Title: "T"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, n, _, _ := newTestDesktop(api26)
			if _, err := d.ShowNotification(context.Background(), tc.spec); err != nil {
				t.Fatal(err)
			}
			got := n.shown[0]
			if got.ID != tc.want.ID || got.Serial != tc.want.Serial || got.Title != tc.want.Title || got.Body != tc.want.Body ||
				got.Urgency != tc.want.Urgency || got.Timeout != tc.want.Timeout || len(got.Actions) != len(tc.want.Actions) {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
			for i, a := range got.Actions {
				if a != tc.want.Actions[i] {
					t.Errorf("action %d is %+v, want %+v", i, a, tc.want.Actions[i])
				}
			}
		})
	}
}

func TestShowNotificationFailures(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantTag string
		route   string
	}{
		{"the system refuses", Refused{"SecurityException: no permission"}, "AndroidNotificationFailed", "Desktop.ShowNotification.AndroidNotificationFailed"},
		{"the channel is off", Blocked{"the channel glutter_low is turned off"}, "NotificationsUnavailable", "Desktop.ShowNotification.NotificationsUnavailable"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, n, _, _ := newTestDesktop(api26)
			n.showErr = tc.err
			_, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi", Timeout: time.Second})
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tc.wantTag || !appErr.HasRoute(tc.route) {
				t.Fatalf("got %v, want %s via %s", err, tc.wantTag, tc.route)
			}
			if reason, _ := appErr.Metadata["reason"].(string); reason != tc.err.Error() {
				t.Errorf("reason %q", reason)
			}
			if len(d.live) != 0 || len(d.bySerial) != 0 {
				t.Errorf("a notification that was never shown is kept: %v %v", d.live, d.bySerial)
			}
		})
	}
	var none *Desktop
	for name, call := range map[string]func() error{
		"show": func() error {
			_, err := none.ShowNotification(context.Background(), NotificationSpec{Title: "x"})
			return err
		},
		"close": func() error { return none.CloseNotification(context.Background(), 1) },
	} {
		if tag := tagOf(t, call()); tag != "NotificationsUnavailable" {
			t.Errorf("%s on a nil Desktop gave %q", name, tag)
		}
	}
}

func TestShowNotificationReplacesByID(t *testing.T) {
	d, n, _, _ := newTestDesktop(api26)
	var closed []CloseReason
	spec := NotificationSpec{Title: "Download", OnClose: func(r CloseReason) { closed = append(closed, r) }}
	id1, _ := d.ShowNotification(context.Background(), spec)
	spec.Title, spec.ReplacesID = "Download 50%", id1
	id2, err := d.ShowNotification(context.Background(), spec)
	if err != nil || id2 != id1 {
		t.Fatalf("got %d, %v; want the ID %d again", id2, err, id1)
	}
	if n.shown[0].ID != n.shown[1].ID || n.shown[1].Title != "Download 50%" || n.shown[0].Serial == n.shown[1].Serial {
		t.Errorf("the notifications %+v and %+v should have the same ID and different serials", n.shown[0], n.shown[1])
	}
	// The replaced notification's events are not the new one's: the app is not told, and the new one
	// is still there.
	n.sink.Closed(n.serial(0), CloseDismissed)
	n.sink.Clicked(n.serial(0))
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

func TestNotificationEvents(t *testing.T) {
	actions := []Action{{"reply", "Reply"}, {"default", "Open"}}
	tests := []struct {
		name        string
		actions     []Action
		do          func(d *Desktop, n *fakeNotes, id uint32, serial int)
		wantActions []string
		wantReasons []CloseReason
		wantCancel  bool
	}{
		{"a click calls the default action and closes as dismissed", actions,
			func(_ *Desktop, n *fakeNotes, _ uint32, s int) { n.sink.Clicked(s) },
			[]string{"default"}, []CloseReason{CloseDismissed}, true},
		{"a click on a notification without the default action only closes it", []Action{{"reply", "Reply"}},
			func(_ *Desktop, n *fakeNotes, _ uint32, s int) { n.sink.Clicked(s) },
			nil, []CloseReason{CloseDismissed}, true},
		{"a button calls its action and closes as dismissed", actions,
			func(_ *Desktop, n *fakeNotes, _ uint32, s int) { n.sink.Action(s, "reply") },
			[]string{"reply"}, []CloseReason{CloseDismissed}, true},
		{"a button the notification does not have is ignored", actions,
			func(_ *Desktop, n *fakeNotes, _ uint32, s int) {
				n.sink.Action(s, "other")
				n.sink.Action(s, "default")
			},
			nil, nil, false},
		{"a swipe", actions,
			func(_ *Desktop, n *fakeNotes, _ uint32, s int) { n.sink.Closed(s, CloseDismissed) },
			nil, []CloseReason{CloseDismissed}, false},
		{"a close for no known reason", actions,
			func(_ *Desktop, n *fakeNotes, _ uint32, s int) { n.sink.Closed(s, CloseUnknown) },
			nil, []CloseReason{CloseUnknown}, false},
		{"closed by the app", actions,
			func(d *Desktop, _ *fakeNotes, id uint32, _ int) {
				if err := d.CloseNotification(context.Background(), id); err != nil {
					t.Error(err)
				}
			},
			nil, []CloseReason{CloseByApp}, true},
		{"closed by the app and then by the system's event", actions,
			func(d *Desktop, n *fakeNotes, id uint32, s int) {
				_ = d.CloseNotification(context.Background(), id)
				n.sink.Closed(s, CloseDismissed)
				n.sink.Clicked(s)
				_ = d.CloseNotification(context.Background(), id)
			},
			nil, []CloseReason{CloseByApp}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, n, _, _ := newTestDesktop(api26)
			var got []string
			var reasons []CloseReason
			id, err := d.ShowNotification(context.Background(), NotificationSpec{
				Title: "Hi", Actions: tc.actions,
				OnAction: func(id string) { got = append(got, id) }, OnClose: func(r CloseReason) { reasons = append(reasons, r) },
			})
			if err != nil {
				t.Fatal(err)
			}
			tc.do(d, n, id, n.serial(0))
			if strings.Join(got, ",") != strings.Join(tc.wantActions, ",") {
				t.Errorf("actions %v, want %v", got, tc.wantActions)
			}
			if len(reasons) != len(tc.wantReasons) || (len(reasons) == 1 && reasons[0] != tc.wantReasons[0]) {
				t.Errorf("close reasons %v, want %v", reasons, tc.wantReasons)
			}
			if tc.wantCancel != (len(n.canceled) == 1 && n.canceled[0] == id) {
				t.Errorf("canceled %v, want it %v", n.canceled, tc.wantCancel)
			}
			wantLive := 0
			if len(tc.wantReasons) == 0 {
				wantLive = 1
			}
			if len(d.live) != wantLive || len(d.bySerial) != wantLive {
				t.Errorf("%d notifications are kept, want %d", len(d.live), wantLive)
			}
		})
	}
	// An event for a notification that is not known does nothing, and a notification with no callbacks
	// still closes.
	d, n, _, _ := newTestDesktop(api26)
	n.sink.Clicked(42)
	n.sink.Action(42, "x")
	n.sink.Closed(42, CloseDismissed)
	if _, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi"}); err != nil {
		t.Fatal(err)
	}
	n.sink.Clicked(1)
	if len(d.live) != 0 {
		t.Error("a click did not close a notification without callbacks")
	}
}

func TestCloseNotificationCancelsOnce(t *testing.T) {
	d, n, _, _ := newTestDesktop(api26)
	id, _ := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi"})
	if err := d.CloseNotification(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if len(n.canceled) != 1 || n.canceled[0] != id {
		t.Errorf("cancel was called for %v", n.canceled)
	}
	// One that is gone is not an error, and the system is not asked again.
	if err := d.CloseNotification(context.Background(), id); err != nil || len(n.canceled) != 1 {
		t.Errorf("closing a gone notification: %v, %d cancels", err, len(n.canceled))
	}
}

func TestNotificationTimeout(t *testing.T) {
	t.Run("expires", func(t *testing.T) {
		d, n, _, c := newTestDesktop(api26)
		var reasons []CloseReason
		id, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi", Timeout: 3 * time.Second, OnClose: func(r CloseReason) { reasons = append(reasons, r) }})
		if err != nil {
			t.Fatal(err)
		}
		if len(c.durs) != 1 || c.durs[0] != 3*time.Second {
			t.Fatalf("timers %v, want one of 3s", c.durs)
		}
		c.fire(0)
		if len(reasons) != 1 || reasons[0] != CloseExpired || len(n.canceled) != 1 || n.canceled[0] != id {
			t.Errorf("reasons %v, canceled %v; want CloseExpired and the notification removed", reasons, n.canceled)
		}
	})
	t.Run("a close stops the timer", func(t *testing.T) {
		d, _, _, c := newTestDesktop(api26)
		var reasons []CloseReason
		id, _ := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi", Timeout: time.Second, OnClose: func(r CloseReason) { reasons = append(reasons, r) }})
		_ = d.CloseNotification(context.Background(), id)
		c.fire(0)
		if len(reasons) != 1 || reasons[0] != CloseByApp {
			t.Errorf("reasons %v, want only CloseByApp", reasons)
		}
	})
	t.Run("a replacement stops the old timer and gets its own", func(t *testing.T) {
		d, n, _, c := newTestDesktop(api26)
		id, _ := d.ShowNotification(context.Background(), NotificationSpec{Title: "a", Timeout: time.Second})
		_, _ = d.ShowNotification(context.Background(), NotificationSpec{Title: "b", ReplacesID: id, Timeout: 2 * time.Second})
		c.fire(0) // the old one: stopped
		if len(n.canceled) != 0 {
			t.Error("the stopped timer of the replaced notification removed the new one")
		}
		c.fire(1)
		if len(n.canceled) != 1 {
			t.Error("the new timer did not remove its notification")
		}
	})
	t.Run("none without a timeout", func(t *testing.T) {
		d, _, _, c := newTestDesktop(api26)
		_, _ = d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi"})
		if len(c.durs) != 0 {
			t.Errorf("timers %v", c.durs)
		}
	})
	t.Run("the real clock is used by default", func(t *testing.T) {
		n := &fakeNotes{status: api26}
		d := NewDesktop(nil, n, nil)
		done := make(chan CloseReason, 1)
		if _, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi", Timeout: time.Millisecond, OnClose: func(r CloseReason) { done <- r }}); err != nil {
			t.Fatal(err)
		}
		select {
		case r := <-done:
			if r != CloseExpired {
				t.Errorf("got %v", r)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the timeout never closed the notification")
		}
	})
}
