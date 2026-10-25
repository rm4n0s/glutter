package mobile

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
)

// This file is the mobile side of the desktop services of package desktop (SPEC §24.11): opening a
// URI with an intent and showing notifications through the notification manager. The logic is plain
// Go over two small interfaces, IntentOpener and NotificationAPI, so that the normal test run tests
// it; the platform (internal/platform/android) is the Java side behind them. Package desktop reaches
// it through the Services interface: a notification is a NotificationSpec, a number and two callbacks.
// It follows internal/platform/web.Desktop (SPEC §23.7) closely.

// IntentOpener starts the activity that views a URI (ACTION_VIEW).
//
// Flutter: no counterpart
type IntentOpener interface {
	// Open starts an ACTION_VIEW intent for the URI. It fails with a Refused that says why when no app
	// can handle the URI, when the app has no activity to start it from, or when the system refuses.
	// It returns when the system has answered or ctx ends.
	Open(ctx context.Context, uri string) error
}

// Notification is what NotificationAPI.Show posts: one notification of the notification manager.
//
// Flutter: no counterpart
type Notification struct {
	// ID is the notification's ID in the notification manager; a notification with the ID of one that is
	// shown replaces it. Serial tells the notifications that had the same ID apart: the events of the
	// sink carry it.
	ID     uint32
	Serial int
	// Title and Body are plain text.
	Title, Body string
	// Urgency chooses the channel: one per urgency.
	Urgency Urgency
	// Actions are the buttons, in order; the action "default" is not among them (it is the click on the
	// notification itself).
	Actions []Action
	// Timeout, when positive, is how long the notification stays; the system removes it after that
	// (plus some grace), as a backstop for the Desktop's own timer.
	Timeout time.Duration
}

// Status is what the notification manager says about the app.
//
// Flutter: no counterpart
type Status struct {
	// PermissionNeeded says that the system asks the user for a permission before the app may post (the
	// runtime permission POST_NOTIFICATIONS of Android 13 and later), and PermissionGranted that the app
	// has it. Both are meaningful only together: PermissionGranted is true when none is needed.
	PermissionNeeded, PermissionGranted bool
	// Enabled is NotificationManager.areNotificationsEnabled: false when the user turned the app's
	// notifications off.
	Enabled bool
}

// Permission is the answer of the user to the system's permission prompt.
//
// Flutter: no counterpart
type Permission int

// The answers to a request for the permission to post notifications.
const (
	// PermissionUndecided is a request that ended without a decision of the user (no activity to show
	// the prompt over, or the prompt was interrupted): it is not remembered.
	//
	// Flutter: no counterpart
	PermissionUndecided Permission = iota
	// PermissionGranted is the user allowing notifications.
	//
	// Flutter: no counterpart
	PermissionGranted
	// PermissionDenied is the user refusing (or the system refusing for the user, after two refusals).
	//
	// Flutter: no counterpart
	PermissionDenied
)

// Sink is told about the events of the notifications that Show posted. Its methods are called on any
// goroutine, including a thread of the platform that must not wait.
//
// Flutter: no counterpart
type Sink interface {
	// Clicked is the click on the notification with the serial.
	Clicked(serial int)
	// Action is the click on the button with the ID, of the notification with the serial.
	Action(serial int, id string)
	// Closed is the end of the notification with the serial, which the system did (CloseDismissed when
	// the user swiped it away) and the Desktop did not.
	Closed(serial int, reason CloseReason)
}

// NotificationAPI is the notification manager as the Desktop uses it. internal/platform/android is the
// real one, and the tests use a fake.
//
// Flutter: no counterpart
type NotificationAPI interface {
	// SetSink says who to tell about the clicks and the closing of the notifications.
	SetSink(Sink)
	// Status reads whether the app may post now.
	Status(ctx context.Context) (Status, error)
	// RequestPermission starts the system's permission prompt and returns; done is called once, later,
	// on any goroutine, with the user's answer or the reason the request failed.
	RequestPermission(done func(Permission, error))
	// Show posts the notification. It fails with a Blocked when the notification manager takes it but
	// the user turned it off (the app's notifications, or the channel), and otherwise with a Refused.
	// It returns when the system has answered or ctx ends.
	Show(ctx context.Context, n Notification) error
	// Cancel removes the notification with the ID (NotificationManager.cancel).
	Cancel(id uint32)
}

// Refused is the reason that the Java side did not do what it was asked.
//
// Flutter: no counterpart
type Refused struct{ Detail string }

// Error implements error.
func (r Refused) Error() string { return r.Detail }

// Blocked is a notification that the user turned off for the app or for its channel.
//
// Flutter: no counterpart
type Blocked struct{ Reason string }

// Error implements error.
func (b Blocked) Error() string { return b.Reason }

// Urgency is how important a notification is. The zero value is Normal.
//
// Flutter: no counterpart
type Urgency int

// The urgencies, one channel each.
const (
	// UrgencyNormal is the usual one (IMPORTANCE_DEFAULT).
	//
	// Flutter: no counterpart
	UrgencyNormal Urgency = iota
	// UrgencyLow makes no sound (IMPORTANCE_LOW).
	//
	// Flutter: no counterpart
	UrgencyLow
	// UrgencyCritical shows over the screen (IMPORTANCE_HIGH).
	//
	// Flutter: no counterpart
	UrgencyCritical
)

// Action is a button of a notification, or, with the ID "default", the click on the notification.
//
// Flutter: no counterpart
type Action struct{ ID, Label string }

// CloseReason is why a notification went away. The values are those of the Desktop Notifications
// specification, as package desktop's NotificationCloseReason has them.
//
// Flutter: no counterpart
type CloseReason int

// The reasons a notification closes.
const (
	// CloseUnknown is a close that the app cannot explain.
	//
	// Flutter: no counterpart
	CloseUnknown CloseReason = iota
	// CloseExpired is a notification whose Timeout was up.
	//
	// Flutter: no counterpart
	CloseExpired
	// CloseDismissed is a notification that the user swiped away, or that closed after a click.
	//
	// Flutter: no counterpart
	CloseDismissed
	// CloseByApp is a notification that the app closed.
	//
	// Flutter: no counterpart
	CloseByApp
)

// NotificationSpec is a notification for Desktop.ShowNotification.
//
// Flutter: no counterpart
type NotificationSpec struct {
	// Title is the summary; Body the plain text under it.
	Title, Body string
	// Urgency chooses the channel.
	Urgency Urgency
	// Timeout, when positive, closes the notification after that long, with CloseExpired. A
	// notification that must be acted on (Persistent, Critical) has none.
	Timeout time.Duration
	// ReplacesID is the ID of a notification that is shown, which this one replaces; zero makes a new
	// one.
	ReplacesID uint32
	// Actions are the buttons; the one with the ID "default" is the click on the notification.
	Actions []Action
	// Icon is the path of an image file or the name of a theme icon, as the app gave it. Windows uses a
	// PNG or JPEG file as the large icon of the balloon; Android and the web have their own icons.
	Icon string
	// OnAction is called, on any goroutine, with the ID of the action the user chose; the click on the
	// notification is the action "default", only when Actions has it. nil listens to nothing.
	OnAction func(id string)
	// OnClose is called, on any goroutine, once, when the notification goes away. A notification that
	// another one replaced does not call it.
	OnClose func(CloseReason)
}

// ServiceDesktop is the name under which the engine hands Services to package desktop
// (ui.PlatformDispatcher.PlatformService).
//
// Flutter: no counterpart
const ServiceDesktop = "desktop"

// DefaultAction is the ID of the action that a click on the notification itself calls.
//
// Flutter: no counterpart
const DefaultAction = "default"

// Services is what package desktop calls on a mobile platform. *Desktop is the implementation.
//
// Flutter: no counterpart
type Services interface {
	// OpenURI opens the URI in the app that views it.
	OpenURI(ctx context.Context, uri string) error
	// ShowNotification shows the notification and returns its ID.
	ShowNotification(ctx context.Context, spec NotificationSpec) (uint32, error)
	// CloseNotification closes the notification with the ID.
	CloseNotification(ctx context.Context, id uint32) error
}

// liveNotification is a notification that is shown.
type liveNotification struct {
	serial int
	spec   NotificationSpec
	stop   func()
}

// Desktop is what package desktop calls on Android (SPEC §24.11): OpenURI, ShowNotification and
// CloseNotification. Its methods may be called from any goroutine but the UI's: ShowNotification
// waits for the permission prompt, which the UI goroutine must never do (SPEC §24.3).
//
// Flutter: no counterpart (Flutter apps use the url_launcher and flutter_local_notifications plugins)
type Desktop struct {
	opener IntentOpener
	notes  NotificationAPI
	after  func(d time.Duration, fn func()) (stop func())

	// ask serializes the prompts: one is on screen at a time.
	ask sync.Mutex

	mu       sync.Mutex
	next     uint32
	serial   int
	live     map[uint32]*liveNotification
	bySerial map[int]uint32
	// denied is whether the user refused the permission during this run; the prompt is not shown again
	// after that (the user decides in the system's settings).
	denied bool
}

var _ Services = (*Desktop)(nil)
var _ Sink = (*Desktop)(nil)

// NewDesktop makes a Desktop over the Java side; either part may be nil, for a host that does not
// have it. after starts a timer (time.AfterFunc when nil); tests give their own clock.
//
// Flutter: no counterpart
func NewDesktop(opener IntentOpener, notes NotificationAPI, after func(time.Duration, func()) func()) *Desktop {
	d := &Desktop{opener: opener, notes: notes, after: after, live: map[uint32]*liveNotification{}, bySerial: map[int]uint32{}}
	if after == nil {
		d.after = func(dur time.Duration, fn func()) func() {
			t := time.AfterFunc(dur, fn)
			return func() { t.Stop() }
		}
	}
	if notes != nil {
		notes.SetSink(d)
	}
	return d
}

// blockedSchemes are the schemes that OpenURI does not open: they run script in the page that opens
// them, as on the web (SPEC §23.7).
var blockedSchemes = []string{"javascript", "vbscript"}

// uriScheme is the scheme of the URI as a browser reads it: leading control characters and spaces are
// dropped, and tabs and newlines inside it are ignored (the URL standard), so that "java\tscript:" does
// not slip by.
func uriScheme(raw string) string {
	raw = strings.TrimLeft(raw, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f ")
	raw = strings.NewReplacer("\t", "", "\n", "", "\r", "").Replace(raw)
	i := strings.IndexByte(raw, ':')
	if i < 0 {
		return ""
	}
	return strings.ToLower(raw[:i])
}

// URIScheme is the scheme of the URI as a browser reads it (leading control characters and spaces are dropped,
// tabs and newlines inside it are ignored), in lower case; it is empty when the URI has none.
//
// Flutter: no counterpart
func URIScheme(raw string) string { return uriScheme(raw) }

// OpenURI starts an ACTION_VIEW intent for the URI (SPEC §24.11). A scheme that runs script
// (javascript:, vbscript:) is OpenURISchemeBlocked, as on the web. A URI that no app can handle, a
// program with no activity to start it from, and a refusal of the system are OpenURIUnavailable, with
// the reason in the metadata.
//
// Flutter: no counterpart (url_launcher_android: launchUrl)
func (d *Desktop) OpenURI(ctx context.Context, uri string) error {
	if err := ctx.Err(); err != nil {
		return errors.New(tagOpenURIUnavailable, "Android cannot open the URI", "uri", uri, "reason", "the context ended: "+err.Error())
	}
	if scheme := uriScheme(uri); scheme != "" {
		for _, b := range blockedSchemes {
			if scheme == b {
				return errors.New(tagOpenURISchemeBlocked, "Android does not open a URI that runs script", "uri", uri, "scheme", scheme)
			}
		}
	}
	if d == nil || d.opener == nil {
		return errors.New(tagOpenURIUnavailable, "Android cannot open the URI", "uri", uri, "reason", "the program is not running in an Android activity")
	}
	if err := d.opener.Open(ctx, uri); err != nil {
		return errors.New(tagOpenURIUnavailable, "Android cannot open the URI", "uri", uri, "reason", err.Error())
	}
	return nil
}

// ensurePermission checks that the app may post and asks the user when the system wants it asked:
// a permission that is granted goes on; one that is needed and not granted is asked for, once per
// decision (a refusal is remembered for the run, and not asked again); a refusal, and a user who
// turned the app's notifications off, are NotificationsUnavailable. The prompt belongs to the system;
// the wait ends when the user answers or ctx ends.
func (d *Desktop) ensurePermission(ctx context.Context) error {
	st, err := d.notes.Status(ctx)
	if err != nil {
		return errors.New(tagNotificationsUnavailable, "Android cannot show notifications", "reason", "the notification manager did not answer: "+err.Error())
	}
	if st.PermissionNeeded && !st.PermissionGranted {
		if err := d.askPermission(ctx); err != nil {
			return err
		}
		st, err = d.notes.Status(ctx)
		if err != nil {
			return errors.New(tagNotificationsUnavailable, "Android cannot show notifications", "reason", "the notification manager did not answer: "+err.Error())
		}
		if st.PermissionNeeded && !st.PermissionGranted {
			return errors.New(tagNotificationsUnavailable, "the user did not allow the app to show notifications", "permission", "denied")
		}
	}
	if !st.Enabled {
		return errors.New(tagNotificationsUnavailable, "the user turned off the notifications of the app", "reason", "NotificationManager.areNotificationsEnabled is false")
	}
	return nil
}

// askPermission shows the system's prompt unless the user refused it in this run, and waits for the
// answer.
func (d *Desktop) askPermission(ctx context.Context) error {
	d.ask.Lock()
	defer d.ask.Unlock()
	d.mu.Lock()
	denied := d.denied
	d.mu.Unlock()
	if denied {
		return errors.New(tagNotificationsUnavailable, "the user did not allow the app to show notifications", "permission", "denied")
	}
	type answer struct {
		permission Permission
		err        error
	}
	got := make(chan answer, 1)
	d.notes.RequestPermission(func(p Permission, err error) {
		if err == nil && p == PermissionDenied {
			// Remembered even when nobody waits any more: the user did decide.
			d.mu.Lock()
			d.denied = true
			d.mu.Unlock()
		}
		select {
		case got <- answer{p, err}:
		default:
		}
	})
	select {
	case a := <-got:
		switch {
		case a.err != nil:
			return errors.New(tagNotificationsUnavailable, "Android cannot show notifications", "reason", "requesting the permission failed: "+a.err.Error())
		case a.permission == PermissionDenied:
			return errors.New(tagNotificationsUnavailable, "the user did not allow the app to show notifications", "permission", "denied")
		case a.permission != PermissionGranted:
			return errors.New(tagNotificationsUnavailable, "Android cannot show notifications", "reason", "the permission prompt ended without an answer")
		}
		return nil
	case <-ctx.Done():
		return errors.NewErr(tagNotificationPermissionAborted, ctx.Err())
	}
}

// ShowNotification posts the notification and returns its ID. The first call asks the user for the
// permission when Android 13 or later wants it; a permission that is not granted, and a user who
// turned the notifications off, are NotificationsUnavailable (SPEC §24.11). A spec with ReplacesID of
// a notification that is shown replaces it and keeps its ID. A notification manager that refuses the
// notification is AndroidNotificationFailed.
//
// Flutter: no counterpart (flutter_local_notifications)
func (d *Desktop) ShowNotification(ctx context.Context, spec NotificationSpec) (uint32, error) {
	if d == nil || d.notes == nil {
		return 0, errors.New(tagNotificationsUnavailable, "Android cannot show notifications", "reason", "the program is not running in an Android activity")
	}
	if err := d.ensurePermission(ctx); err != nil {
		return 0, err
	}
	d.mu.Lock()
	id := spec.ReplacesID
	if old := d.live[id]; id != 0 && old != nil {
		if old.stop != nil {
			old.stop()
		}
		delete(d.bySerial, old.serial)
	} else {
		d.next++
		id = d.next
	}
	d.serial++
	ln := &liveNotification{serial: d.serial, spec: spec}
	d.live[id] = ln
	d.bySerial[ln.serial] = id
	d.mu.Unlock()

	n := Notification{ID: id, Serial: ln.serial, Title: spec.Title, Body: spec.Body, Urgency: spec.Urgency, Timeout: spec.Timeout}
	for _, a := range spec.Actions {
		if a.ID != DefaultAction {
			n.Actions = append(n.Actions, a)
		}
	}
	if err := d.notes.Show(ctx, n); err != nil {
		d.mu.Lock()
		if d.live[id] == ln {
			delete(d.live, id)
		}
		delete(d.bySerial, ln.serial)
		d.mu.Unlock()
		var blocked Blocked
		if errors.As(err, &blocked) {
			return 0, errors.New(tagNotificationsUnavailable, "the user turned off this kind of notification", "reason", blocked.Reason)
		}
		return 0, errors.New(tagAndroidNotificationFailed, "Android could not post the notification", "reason", err.Error())
	}
	if spec.Timeout > 0 {
		serial := ln.serial
		stop := d.after(spec.Timeout, func() { d.end(id, serial, CloseExpired, true) })
		d.mu.Lock()
		if d.live[id] == ln {
			ln.stop = stop
			stop = nil
		}
		d.mu.Unlock()
		if stop != nil {
			stop() // closed or replaced while the timer was being made
		}
	}
	return id, nil
}

// CloseNotification closes the notification with the ID, which calls its OnClose with CloseByApp.
// Closing one that is gone is not an error; a program without an Android activity is
// NotificationsUnavailable.
//
// Flutter: no counterpart
func (d *Desktop) CloseNotification(_ context.Context, id uint32) error {
	if d == nil || d.notes == nil {
		return errors.New(tagNotificationsUnavailable, "Android cannot show notifications", "reason", "the program is not running in an Android activity")
	}
	d.mu.Lock()
	ln := d.live[id]
	d.mu.Unlock()
	if ln != nil {
		d.end(id, ln.serial, CloseByApp, true)
	}
	return nil
}

// end takes the notification out of the shown ones, removes it from the notification manager when the
// Desktop ended it (cancel), and tells its OnClose. A notification that is not the one with the serial
// any more (it was replaced or already closed) is left alone.
func (d *Desktop) end(id uint32, serial int, reason CloseReason, cancel bool) {
	d.mu.Lock()
	ln := d.live[id]
	if ln == nil || ln.serial != serial {
		d.mu.Unlock()
		return
	}
	delete(d.live, id)
	delete(d.bySerial, serial)
	stop := ln.stop
	d.mu.Unlock()
	if stop != nil {
		stop()
	}
	if cancel {
		d.notes.Cancel(id)
	}
	if ln.spec.OnClose != nil {
		ln.spec.OnClose(reason)
	}
}

// find returns the shown notification with the serial, and its ID.
func (d *Desktop) find(serial int) (*liveNotification, uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id, ok := d.bySerial[serial]
	if !ok {
		return nil, 0
	}
	ln := d.live[id]
	if ln == nil || ln.serial != serial {
		return nil, 0
	}
	return ln, id
}

// Clicked is the system telling that the user clicked the notification with the serial: the action
// "default" is called when the notification has it, and the notification is closed (Android removes a
// notification that was clicked), as dismissed.
//
// Flutter: no counterpart
func (d *Desktop) Clicked(serial int) {
	ln, id := d.find(serial)
	if ln == nil {
		return
	}
	if ln.spec.OnAction != nil && hasAction(ln.spec.Actions, DefaultAction) {
		ln.spec.OnAction(DefaultAction)
	}
	d.end(id, serial, CloseDismissed, true)
}

// Action is the system telling that the user chose the button with the ID of the notification with
// the serial: the action is called and the notification closed as dismissed. A button that the
// notification does not have is ignored.
//
// Flutter: no counterpart
func (d *Desktop) Action(serial int, actionID string) {
	ln, id := d.find(serial)
	if ln == nil || actionID == DefaultAction || !hasAction(ln.spec.Actions, actionID) {
		return
	}
	if ln.spec.OnAction != nil {
		ln.spec.OnAction(actionID)
	}
	d.end(id, serial, CloseDismissed, true)
}

// Closed is the system telling that the notification with the serial is gone and the Desktop did not
// remove it: the user swiped it away.
//
// Flutter: no counterpart
func (d *Desktop) Closed(serial int, reason CloseReason) {
	if ln, id := d.find(serial); ln != nil {
		d.end(id, serial, reason, false)
	}
}

func hasAction(actions []Action, id string) bool {
	for _, a := range actions {
		if a.ID == id {
			return true
		}
	}
	return false
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAndroidNotificationFailed     = "AndroidNotificationFailed"
	tagNotificationPermissionAborted = "NotificationPermissionAborted"
	tagNotificationsUnavailable      = "NotificationsUnavailable"
	tagOpenURISchemeBlocked          = "OpenURISchemeBlocked"
	tagOpenURIUnavailable            = "OpenURIUnavailable"
)
