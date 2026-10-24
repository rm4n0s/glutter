package android

import (
	"context"
	"encoding/binary"
	"sync"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
)

// The operations of the activity's command call for the desktop services (SPEC §24.11); systemui.go,
// textinput.go, back.go and accessibility.go have 1 to 18. They are done on the main thread by
// DesktopServices.java, and may be called from any goroutine: the answers come back in
// Native.desktopResult and the events of the notifications in Native.notificationEvent.
const (
	// opDesktopOpen: A is the number of the request, S the URI. The answer's code is 0 when the
	// activity started, 1 when it failed (the text is the reason), 2 when no app can handle the URI.
	opDesktopOpen = 19
	// opDesktopStatus: A is the number of the request. The answer's code is a set of bits:
	// statusPermissionNeeded, statusPermissionGranted and statusEnabled.
	opDesktopStatus = 20
	// opDesktopPermission: A is the number of the request. The answer's code is 1 when the user
	// granted the permission, 0 when the user refused, and 2 when it ended without a decision (the text
	// is the reason).
	opDesktopPermission = 21
	// opDesktopNotify: A is the number of the request, B the notification's ID, C its serial and D its
	// timeout in milliseconds (0 for none); S is the notification, see encodeNotification. The answer's
	// code is 0 when it was posted, 1 when the system refused (the text is the reason), 2 when the user
	// turned off the app's notifications or the channel (the text is the reason).
	opDesktopNotify = 22
	// opDesktopCancel: B is the notification's ID. Nothing is answered.
	opDesktopCancel = 23
)

// The bits of the answer to opDesktopStatus.
const (
	statusPermissionNeeded  = 1
	statusPermissionGranted = 2
	statusEnabled           = 4
)

// The codes of the answers to opDesktopOpen, opDesktopPermission and opDesktopNotify.
const (
	answerOK     = 0
	answerFailed = 1
	// answerNoApp is no app for the URI, or (for a notification) a channel or an app that is turned off.
	answerNoApp = 2
	// For opDesktopPermission: the user said no (0), said yes (1), or did not say (2).
	answerDenied    = 0
	answerGranted   = 1
	answerUndecided = 2
)

// The kinds of Native.notificationEvent.
const (
	eventClick  = 1
	eventAction = 2
	eventClosed = 3
)

// The reason that Native.notificationEvent gives for a notification the user swiped away.
const eventReasonDismissed = "dismissed"

// answerWait is how long the link waits for the main thread's answer before it gives up.
const answerWait = 10 * time.Second

// encodeNotification is the text of opDesktopNotify: fields, each a four-byte big-endian length and
// that many bytes of UTF-8 (any text, with NUL and four-byte characters): the urgency ("low", "normal"
// or "critical"), the title, the body, then the ID and the label of each button.
func encodeNotification(n mobile.Notification) string {
	urgency := "normal"
	switch n.Urgency {
	case mobile.UrgencyLow:
		urgency = "low"
	case mobile.UrgencyCritical:
		urgency = "critical"
	}
	fields := []string{urgency, n.Title, n.Body}
	for _, a := range n.Actions {
		fields = append(fields, a.ID, a.Label)
	}
	var out []byte
	for _, f := range fields {
		out = binary.BigEndian.AppendUint32(out, uint32(len(f)))
		out = append(out, f...)
	}
	return string(out)
}

// desktopAnswer is the main thread's answer to a request.
type desktopAnswer struct {
	code int
	text string
}

// desktopLink is the Java side of the desktop services: the intent opener and the notification
// manager of the mobile.Desktop. It sends the activity commands and gets the answers and the events
// back from Native.desktopResult and Native.notificationEvent, which the main thread calls; it never
// waits on the UI thread, and its callers are not the UI thread either (SPEC §24.3).
//
// Flutter: no counterpart (the url_launcher and flutter_local_notifications plugins' Java side)
type desktopLink struct {
	host host
	// wait is how long a request waits for the answer; answerWait, or less in a test.
	wait time.Duration

	mu      sync.Mutex
	next    int
	pending map[int]func(desktopAnswer)
	sink    mobile.Sink
}

var (
	_ mobile.IntentOpener      = (*desktopLink)(nil)
	_ mobile.NotificationAPI   = (*desktopLink)(nil)
	_ platform.DesktopProvider = (*Platform)(nil)
)

func newDesktopLink(h host) *desktopLink {
	return &desktopLink{host: h, wait: answerWait, pending: map[int]func(desktopAnswer){}}
}

// request sends the command with a new request number and calls done, once, with the answer.
func (l *desktopLink) request(c command, done func(desktopAnswer)) int {
	l.mu.Lock()
	l.next++
	id := l.next
	l.pending[id] = done
	l.mu.Unlock()
	c.A = int64(id)
	l.host.Command(c)
	return id
}

// forget drops the wait for the answer to the request; an answer that comes later is ignored.
func (l *desktopLink) forget(id int) {
	l.mu.Lock()
	delete(l.pending, id)
	l.mu.Unlock()
}

// call sends the command and waits for the answer, for l.wait or until ctx ends.
func (l *desktopLink) call(ctx context.Context, c command) (desktopAnswer, error) {
	got := make(chan desktopAnswer, 1)
	id := l.request(c, func(a desktopAnswer) { got <- a })
	t := time.NewTimer(l.wait)
	defer t.Stop()
	select {
	case a := <-got:
		return a, nil
	case <-ctx.Done():
		l.forget(id)
		return desktopAnswer{}, mobile.Refused{Detail: "the context ended: " + ctx.Err().Error()}
	case <-t.C:
		l.forget(id)
		return desktopAnswer{}, mobile.Refused{Detail: "the activity did not answer in " + l.wait.String()}
	}
}

// result is Native.desktopResult: the answer to the request. It is called on the main thread of
// Android, and runs the waiting call's function, which only hands the answer over.
func (l *desktopLink) result(id, code int, text string) {
	l.mu.Lock()
	done := l.pending[id]
	delete(l.pending, id)
	l.mu.Unlock()
	if done != nil {
		done(desktopAnswer{code, text})
	}
}

// event is Native.notificationEvent: a click, a button or the close of the notification with the
// serial. It is called on the main thread of Android.
func (l *desktopLink) event(serial, kind int, text string) {
	l.mu.Lock()
	sink := l.sink
	l.mu.Unlock()
	if sink == nil {
		return
	}
	switch kind {
	case eventClick:
		sink.Clicked(serial)
	case eventAction:
		sink.Action(serial, text)
	case eventClosed:
		reason := mobile.CloseUnknown
		if text == eventReasonDismissed {
			reason = mobile.CloseDismissed
		}
		sink.Closed(serial, reason)
	}
}

// Open implements mobile.IntentOpener: ACTION_VIEW on the activity.
func (l *desktopLink) Open(ctx context.Context, uri string) error {
	a, err := l.call(ctx, command{Op: opDesktopOpen, S: uri})
	if err != nil {
		return err
	}
	if a.code != answerOK {
		return mobile.Refused{Detail: a.text}
	}
	return nil
}

// SetSink implements mobile.NotificationAPI.
func (l *desktopLink) SetSink(s mobile.Sink) {
	l.mu.Lock()
	l.sink = s
	l.mu.Unlock()
}

// Status implements mobile.NotificationAPI.
func (l *desktopLink) Status(ctx context.Context) (mobile.Status, error) {
	a, err := l.call(ctx, command{Op: opDesktopStatus})
	if err != nil {
		return mobile.Status{}, err
	}
	return mobile.Status{
		PermissionNeeded:  a.code&statusPermissionNeeded != 0,
		PermissionGranted: a.code&statusPermissionGranted != 0,
		Enabled:           a.code&statusEnabled != 0,
	}, nil
}

// RequestPermission implements mobile.NotificationAPI: the prompt belongs to the activity, and the
// request has no time limit (the user answers when they answer); done is called when the answer
// comes.
func (l *desktopLink) RequestPermission(done func(mobile.Permission, error)) {
	l.request(command{Op: opDesktopPermission}, func(a desktopAnswer) {
		switch a.code {
		case answerGranted:
			done(mobile.PermissionGranted, nil)
		case answerDenied:
			done(mobile.PermissionDenied, nil)
		default:
			done(mobile.PermissionUndecided, mobile.Refused{Detail: a.text})
		}
	})
}

// Show implements mobile.NotificationAPI.
func (l *desktopLink) Show(ctx context.Context, n mobile.Notification) error {
	a, err := l.call(ctx, command{Op: opDesktopNotify, B: int64(n.ID), C: int64(n.Serial), D: n.Timeout.Milliseconds(), S: encodeNotification(n)})
	if err != nil {
		return err
	}
	switch a.code {
	case answerOK:
		return nil
	case answerNoApp:
		return mobile.Blocked{Reason: a.text}
	}
	return mobile.Refused{Detail: a.text}
}

// Cancel implements mobile.NotificationAPI.
func (l *desktopLink) Cancel(id uint32) {
	l.host.Command(command{Op: opDesktopCancel, B: int64(id)})
}

// desktopLinkFor returns the link of the process, made by the first Desktop call, or nil before.
func (b *bridge) desktopLinkFor() *desktopLink {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.link
}

// desktopServices returns the desktop services of the process, made over the host the first time.
func (b *bridge) desktopServices(h host) mobile.Services {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.desktop == nil {
		b.link = newDesktopLink(h)
		b.desktop = mobile.NewDesktop(b.link, b.link, nil)
	}
	return b.desktop
}

// desktopResult is the answer of the main thread to a request of the desktop services
// (Native.desktopResult). It needs no UI thread: it only hands the answer to the call that waits.
func (b *bridge) desktopResult(id, code int, text string) {
	if l := b.desktopLinkFor(); l != nil {
		l.result(id, code, text)
	}
}

// notificationEvent is a click, a button or the close of a notification (Native.notificationEvent).
func (b *bridge) notificationEvent(serial, kind int, text string) {
	if l := b.desktopLinkFor(); l != nil {
		l.event(serial, kind, text)
	}
}

// Desktop returns the desktop services of Android: opening a URI and notifications (SPEC §24.11). It
// may be called from any goroutine.
//
// Flutter: no counterpart (the url_launcher and flutter_local_notifications plugins)
func (p *Platform) Desktop() mobile.Services { return p.b.desktopServices(p.host) }
