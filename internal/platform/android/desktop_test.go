package android

import (
	"context"
	"encoding/binary"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
)

// decodeFields reads the fields of encodeNotification.
func decodeFields(t *testing.T, s string) []string {
	t.Helper()
	b := []byte(s)
	var out []string
	for len(b) > 0 {
		if len(b) < 4 {
			t.Fatalf("a truncated length in %q", s)
		}
		n := int(binary.BigEndian.Uint32(b))
		b = b[4:]
		if n > len(b) {
			t.Fatalf("a field of %d bytes in %d", n, len(b))
		}
		out = append(out, string(b[:n]))
		b = b[n:]
	}
	return out
}

func TestEncodeNotification(t *testing.T) {
	tests := []struct {
		name string
		n    mobile.Notification
		want []string
	}{
		{"plain", mobile.Notification{Title: "T", Body: "B"}, []string{"normal", "T", "B"}},
		{"low", mobile.Notification{Urgency: mobile.UrgencyLow}, []string{"low", "", ""}},
		{"critical with buttons", mobile.Notification{Urgency: mobile.UrgencyCritical, Title: "T", Actions: []mobile.Action{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}},
			[]string{"critical", "T", "", "a", "A", "b", "B"}},
		{"text that needs the length", mobile.Notification{Title: "a\x00b\U0001F600", Body: "x\x1fy"}, []string{"normal", "a\x00b\U0001F600", "x\x1fy"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeFields(t, encodeNotification(tt.n)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fields %q, want %q", got, tt.want)
			}
		})
	}
}

// java plays the main thread: it answers the commands the link sends.
type java struct {
	host *fakeHost
	link *desktopLink
}

func newJava() *java {
	h := newFakeHost()
	return &java{host: h, link: newDesktopLink(h)}
}

// next waits for the n-th command (from 0) and returns it.
func (j *java) next(t *testing.T, n int) command {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if cmds := j.host.sentCommands(); len(cmds) > n {
			return cmds[n]
		}
		if time.Now().After(deadline) {
			t.Fatalf("command %d was never sent", n)
		}
		time.Sleep(time.Millisecond)
	}
}

// answer replies to the n-th command.
func (j *java) answer(t *testing.T, n, code int, text string) command {
	t.Helper()
	c := j.next(t, n)
	j.link.result(int(c.A), code, text)
	return c
}

func TestLinkOpen(t *testing.T) {
	tests := []struct {
		name       string
		code       int
		text       string
		wantReason string
	}{
		{"started", answerOK, "", ""},
		{"no app", answerNoApp, "no app can open glutter-nothing://x", "no app can open glutter-nothing://x"},
		{"failed", answerFailed, "no activity is shown", "no activity is shown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := newJava()
			got := make(chan error, 1)
			go func() { got <- j.link.Open(context.Background(), "glutter-nothing://x") }()
			c := j.answer(t, 0, tt.code, tt.text)
			if c.Op != opDesktopOpen || c.S != "glutter-nothing://x" || c.A == 0 {
				t.Errorf("command %+v", c)
			}
			err := <-got
			if tt.wantReason == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var r mobile.Refused
			if !errors.As(err, &r) || r.Detail != tt.wantReason {
				t.Errorf("got %v, want a Refused saying %q", err, tt.wantReason)
			}
		})
	}
}

func TestLinkWaitsEndWithoutAnAnswer(t *testing.T) {
	t.Run("the context ends", func(t *testing.T) {
		j := newJava()
		ctx, cancel := context.WithCancel(context.Background())
		got := make(chan error, 1)
		go func() { got <- j.link.Open(ctx, "https://example.org") }()
		c := j.next(t, 0)
		cancel()
		if err := <-got; err == nil || !strings.Contains(err.Error(), "the context ended") {
			t.Errorf("got %v", err)
		}
		// The late answer is ignored.
		j.link.result(int(c.A), answerOK, "")
		if n := len(j.link.pending); n != 0 {
			t.Errorf("%d requests are still pending", n)
		}
	})
	t.Run("the activity does not answer", func(t *testing.T) {
		j := newJava()
		j.link.wait = 10 * time.Millisecond
		_, err := j.link.Status(context.Background())
		if err == nil || !strings.Contains(err.Error(), "did not answer") {
			t.Errorf("got %v", err)
		}
		if n := len(j.link.pending); n != 0 {
			t.Errorf("%d requests are still pending", n)
		}
	})
}

func TestLinkStatus(t *testing.T) {
	tests := []struct {
		code int
		want mobile.Status
	}{
		{0, mobile.Status{}},
		{statusPermissionNeeded, mobile.Status{PermissionNeeded: true}},
		{statusPermissionNeeded | statusPermissionGranted | statusEnabled, mobile.Status{PermissionNeeded: true, PermissionGranted: true, Enabled: true}},
		{statusPermissionGranted | statusEnabled, mobile.Status{PermissionGranted: true, Enabled: true}},
	}
	for _, tt := range tests {
		j := newJava()
		got := make(chan mobile.Status, 1)
		go func() {
			st, err := j.link.Status(context.Background())
			if err != nil {
				t.Error(err)
			}
			got <- st
		}()
		if c := j.answer(t, 0, tt.code, ""); c.Op != opDesktopStatus {
			t.Errorf("command %+v", c)
		}
		if st := <-got; st != tt.want {
			t.Errorf("code %d gave %+v, want %+v", tt.code, st, tt.want)
		}
	}
}

func TestLinkRequestPermission(t *testing.T) {
	tests := []struct {
		name       string
		code       int
		text       string
		want       mobile.Permission
		wantReason string
	}{
		{"granted", answerGranted, "", mobile.PermissionGranted, ""},
		{"denied", answerDenied, "", mobile.PermissionDenied, ""},
		{"undecided", answerUndecided, "no activity", mobile.PermissionUndecided, "no activity"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := newJava()
			type result struct {
				p   mobile.Permission
				err error
			}
			got := make(chan result, 1)
			j.link.RequestPermission(func(p mobile.Permission, err error) { got <- result{p, err} })
			if c := j.answer(t, 0, tt.code, tt.text); c.Op != opDesktopPermission {
				t.Errorf("command %+v", c)
			}
			r := <-got
			if r.p != tt.want || (r.err == nil) != (tt.wantReason == "") || (r.err != nil && r.err.Error() != tt.wantReason) {
				t.Errorf("got %v, %v; want %v, %q", r.p, r.err, tt.want, tt.wantReason)
			}
		})
	}
}

func TestLinkShow(t *testing.T) {
	n := mobile.Notification{ID: 7, Serial: 9, Title: "T", Body: "B", Timeout: 1500 * time.Millisecond, Actions: []mobile.Action{{ID: "reply", Label: "Reply"}}}
	tests := []struct {
		name        string
		code        int
		text        string
		wantBlocked bool
		wantRefused bool
	}{
		{"posted", answerOK, "", false, false},
		{"turned off", answerNoApp, "the channel is off", true, false},
		{"refused", answerFailed, "SecurityException", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := newJava()
			got := make(chan error, 1)
			go func() { got <- j.link.Show(context.Background(), n) }()
			c := j.answer(t, 0, tt.code, tt.text)
			if c.Op != opDesktopNotify || c.B != 7 || c.C != 9 || c.D != 1500 {
				t.Errorf("command %+v", c)
			}
			if f := decodeFields(t, c.S); !reflect.DeepEqual(f, []string{"normal", "T", "B", "reply", "Reply"}) {
				t.Errorf("fields %q", f)
			}
			err := <-got
			var blocked mobile.Blocked
			var refused mobile.Refused
			switch {
			case tt.wantBlocked:
				if !errors.As(err, &blocked) || blocked.Reason != tt.text {
					t.Errorf("got %v, want Blocked", err)
				}
			case tt.wantRefused:
				if !errors.As(err, &refused) || refused.Detail != tt.text {
					t.Errorf("got %v, want Refused", err)
				}
			case err != nil:
				t.Error(err)
			}
		})
	}
}

func TestLinkCancelAndEvents(t *testing.T) {
	j := newJava()
	j.link.Cancel(12)
	if cmds := j.host.sentCommands(); len(cmds) != 1 || cmds[0].Op != opDesktopCancel || cmds[0].B != 12 {
		t.Errorf("commands %+v", cmds)
	}

	var mu sync.Mutex
	var log []string
	// The sink of a platform is the Desktop; a recording sink shows the mapping.
	j.link.SetSink(sinkFunc(func(s string) { mu.Lock(); log = append(log, s); mu.Unlock() }))
	for _, e := range []struct {
		kind int
		text string
	}{{eventClick, ""}, {eventAction, "reply"}, {eventClosed, eventReasonDismissed}, {eventClosed, "whatever"}, {99, ""}} {
		j.link.event(5, e.kind, e.text)
	}
	want := []string{"clicked 5", "action 5 reply", "closed 5 2", "closed 5 0"}
	if !reflect.DeepEqual(log, want) {
		t.Errorf("events %q, want %q", log, want)
	}
	// Without a sink an event is dropped.
	j.link.SetSink(nil)
	j.link.event(5, eventClick, "")
}

// sinkFunc records the events of the link as text.
type sinkFunc func(string)

func (f sinkFunc) Clicked(serial int) { f("clicked " + strconv.Itoa(serial)) }
func (f sinkFunc) Action(serial int, id string) {
	f("action " + strconv.Itoa(serial) + " " + id)
}
func (f sinkFunc) Closed(serial int, r mobile.CloseReason) {
	f("closed " + strconv.Itoa(serial) + " " + strconv.Itoa(int(r)))
}

// TestPlatformDesktopEndToEnd drives the services of the platform with the natives of the Java side:
// a permission prompt that is answered, a notification that is clicked and then closes, the same
// service on the second call, and natives that arrive before there is a service.
func TestPlatformDesktopEndToEnd(t *testing.T) {
	b := newBridge()
	// Natives before the first Desktop call are dropped.
	b.desktopResult(1, 0, "")
	b.notificationEvent(1, eventClick, "")

	host := newFakeHost()
	p := newPlatform(b, host, nil)
	svc := p.Desktop()
	if svc == nil || p.Desktop() != svc {
		t.Fatalf("Desktop() = %v, then %v; want one service", svc, p.Desktop())
	}

	var mu sync.Mutex
	var actions []string
	var reasons []mobile.CloseReason
	type result struct {
		id  uint32
		err error
	}
	got := make(chan result, 1)
	go func() {
		id, err := svc.ShowNotification(context.Background(), mobile.NotificationSpec{
			Title: "Hi", Actions: []mobile.Action{{ID: "default", Label: "Open"}, {ID: "reply", Label: "Reply"}},
			OnAction: func(id string) { mu.Lock(); actions = append(actions, id); mu.Unlock() },
			OnClose:  func(r mobile.CloseReason) { mu.Lock(); reasons = append(reasons, r); mu.Unlock() },
		})
		got <- result{id, err}
	}()
	reply := func(n, code int, text string) command {
		deadline := time.Now().Add(5 * time.Second)
		for len(host.sentCommands()) <= n {
			if time.Now().After(deadline) {
				t.Fatalf("command %d never sent", n)
			}
			time.Sleep(time.Millisecond)
		}
		c := host.sentCommands()[n]
		b.desktopResult(int(c.A), code, text)
		return c
	}
	// Android 13 and later, nothing granted: the status, the prompt, the status again, the notification.
	if c := reply(0, statusPermissionNeeded, ""); c.Op != opDesktopStatus {
		t.Fatalf("command 0 is %+v", c)
	}
	if c := reply(1, answerGranted, ""); c.Op != opDesktopPermission {
		t.Fatalf("command 1 is %+v", c)
	}
	if c := reply(2, statusPermissionNeeded|statusPermissionGranted|statusEnabled, ""); c.Op != opDesktopStatus {
		t.Fatalf("command 2 is %+v", c)
	}
	show := reply(3, answerOK, "")
	if show.Op != opDesktopNotify {
		t.Fatalf("command 3 is %+v", show)
	}
	r := <-got
	if r.err != nil || r.id != uint32(show.B) {
		t.Fatalf("got %d, %v; the command had the ID %d", r.id, r.err, show.B)
	}
	// The user clicks the notification: the default action, and the close.
	b.notificationEvent(int(show.C), eventClick, "")
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(actions, []string{"default"}) || !reflect.DeepEqual(reasons, []mobile.CloseReason{mobile.CloseDismissed}) {
		t.Errorf("actions %v, reasons %v", actions, reasons)
	}
	if cmds := host.sentCommands(); len(cmds) != 5 || cmds[4].Op != opDesktopCancel || cmds[4].B != show.B {
		t.Errorf("commands after the click: %+v", cmds[4:])
	}
}
