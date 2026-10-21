package web

import (
	"context"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// This file is the web's side of the desktop services of package desktop (SPEC §23.7): opening a URI
// with window.open and showing notifications with the Notifications API. The logic is plain Go over two
// small interfaces, uriOpener and notificationAPI, so that the normal test run and scripts/test-wasm.sh
// test it; desktop_js.go is the browser behind them. Package desktop reaches it through NewDesktop and
// keeps no JavaScript type: a notification is a NotificationSpec, a number and two callbacks.

// uriOpener opens a URI in the browser (window.open).
type uriOpener interface {
	// Open calls window.open(uri, target, features). It fails when the call throws.
	Open(uri, target, features string) error
}

// browserNotification is what Show makes a Notification of.
type browserNotification struct {
	// Title and Body are the text; Icon is a URL, or empty.
	Title, Body, Icon string
	// Tag makes a new notification with the tag of a shown one replace it.
	Tag string
	// RequireInteraction keeps it on screen until the user acts; Silent makes it without a sound.
	RequireInteraction, Silent bool
}

// notificationSink is told about the events of the notifications that Show made.
type notificationSink interface {
	// Clicked is the click on the notification with the serial.
	Clicked(serial int)
	// Closed is the end of the notification with the serial, whatever the cause.
	Closed(serial int)
}

// permissionUnsupported is what notificationAPI.Permission says when the page has no Notification.
const permissionUnsupported = "unsupported"

// noNotificationsReason is why a page cannot show notifications when it has no Notifications API.
const noNotificationsReason = "the page has no Notifications API: it needs a secure context (https or localhost)"

// notificationAPI is the part of the browser's Notifications API that the desktop uses. The code in
// desktop_js.go is the real one, and the tests use a fake.
type notificationAPI interface {
	// SetSink says who to tell about the clicks and the closing of the notifications.
	SetSink(notificationSink)
	// Permission is Notification.permission: "granted", "denied" or "default", or "unsupported" when
	// the page has no Notification.
	Permission() string
	// RequestPermission is Notification.requestPermission. It starts the request and returns; done is
	// called once, later, from the promise's callback on any goroutine, with the permission or the
	// reason the request failed.
	RequestPermission(done func(permission string, err error))
	// Show makes the notification, which is told apart from others by the serial. It fails when the
	// browser refuses to construct it.
	Show(n browserNotification, serial int) error
	// Close closes the notification with the serial (Notification.close).
	Close(serial int)
}

// CloseReason is why a notification went away. The values are those of the Desktop Notifications
// specification, as package desktop's NotificationCloseReason has them.
//
// Flutter: no counterpart
type CloseReason int

// The reasons a notification closes.
const (
	// CloseUnknown is a close that the page cannot explain: the browser's own timeout, or the user
	// clearing the notification center.
	//
	// Flutter: no counterpart
	CloseUnknown CloseReason = iota
	// CloseExpired is a notification whose Timeout was up.
	//
	// Flutter: no counterpart
	CloseExpired
	// CloseDismissed is a notification that closed after the user clicked it.
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
	// Title is the summary; Body the plain text under it; Icon the URL of an image, or empty (the web
	// has no icon themes: IconURL tells a URL from a name).
	Title, Body, Icon string
	// RequireInteraction keeps the notification on screen until the user acts (Persistent, Critical).
	RequireInteraction bool
	// Silent asks for no sound (Low urgency).
	Silent bool
	// Timeout, when positive, closes the notification after that long, with CloseExpired. The browser
	// has no timeout of its own to set.
	Timeout time.Duration
	// ReplacesID is the ID of a notification that is shown, which this one replaces; zero makes a new
	// one. The browser does the replacing through the tag.
	ReplacesID uint32
	// OnClick is called, on any goroutine, when the user clicks the notification; nil listens to
	// nothing.
	OnClick func()
	// OnClose is called, on any goroutine, once, when the notification goes away. A notification that
	// another one replaced does not call it.
	OnClose func(CloseReason)
}

// liveNotification is a notification that is shown.
type liveNotification struct {
	serial  int
	spec    NotificationSpec
	clicked bool
	stop    func()
}

// Desktop is what package desktop calls on the web (SPEC §23.7): OpenURI, ShowNotification and
// CloseNotification. Its methods may be called from any goroutine but the UI's: ShowNotification
// waits for the permission prompt, which the UI goroutine must never do (SPEC §23.3).
//
// Flutter: no counterpart (Flutter web apps use the url_launcher_web and flutter_local_notifications plugins)
type Desktop struct {
	opener uriOpener
	notes  notificationAPI
	after  func(d time.Duration, fn func()) (stop func())

	mu       sync.Mutex
	next     uint32
	serial   int
	live     map[uint32]*liveNotification
	bySerial map[int]uint32
}

// newDesktop makes a Desktop over the two parts of the browser; either may be nil, for a page that
// does not have it.
func newDesktop(opener uriOpener, notes notificationAPI, after func(time.Duration, func()) func()) *Desktop {
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

// blockedSchemes are the schemes that OpenURI does not open on the web: they run script in the page that
// opens them.
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

// OpenURI opens the URI in a new browsing context, with window.open(uri, "_blank", "noopener") (SPEC §23.7).
// A scheme that runs script (javascript:, vbscript:) is OpenURISchemeBlocked, and a window.open that
// throws, or a page without a window, OpenURIUnavailable. The call has no way to know that the browser's
// pop-up blocker refused (noopener makes window.open answer null whatever happened), and the browser lets
// a page open a window only soon after a click or a key press: an app calls OpenURI from the handler of one.
//
// Flutter: no counterpart (url_launcher_web: launchUrl)
func (d *Desktop) OpenURI(ctx context.Context, uri string) error {
	if err := ctx.Err(); err != nil {
		return errOpenURIUnavailable(uri, "the context ended: "+err.Error())
	}
	if scheme := uriScheme(uri); scheme != "" {
		for _, b := range blockedSchemes {
			if scheme == b {
				return errOpenURISchemeBlocked(uri, scheme)
			}
		}
	}
	if d == nil || d.opener == nil {
		return errOpenURIUnavailable(uri, "the program is not running in a web page")
	}
	if err := d.opener.Open(uri, "_blank", "noopener"); err != nil {
		return errOpenURIUnavailable(uri, err.Error())
	}
	return nil
}

// ensurePermission asks for the permission to show notifications when the user has not decided yet
// ("default"), and says why when it is not granted. The prompt belongs to the browser; the wait ends when
// the user answers or ctx ends.
func (d *Desktop) ensurePermission(ctx context.Context) error {
	switch p := d.notes.Permission(); p {
	case "granted":
		return nil
	case "denied":
		return errNotificationsRefused(p)
	case permissionUnsupported:
		return errNotificationsUnavailable(noNotificationsReason)
	}
	type answer struct {
		permission string
		err        error
	}
	got := make(chan answer, 1)
	d.notes.RequestPermission(func(permission string, err error) {
		select {
		case got <- answer{permission, err}:
		default:
		}
	})
	select {
	case a := <-got:
		if a.err != nil {
			return errNotificationsUnavailable("requesting the permission failed: " + a.err.Error())
		}
		if a.permission != "granted" {
			return errNotificationsRefused(a.permission)
		}
		return nil
	case <-ctx.Done():
		return errPermissionAborted(ctx.Err())
	}
}

// notificationTag is the tag of the notification with the ID: a new notification with the tag of a shown one
// replaces it.
func notificationTag(id uint32) string { return "glutter-" + strconv.FormatUint(uint64(id), 10) }

// ShowNotification shows the notification and returns its ID. The first call asks the browser for the
// permission when the user has not decided; a permission that is not granted, and a page without the
// Notifications API, are NotificationsUnavailable (SPEC §23.7). A spec with ReplacesID of a notification
// that is shown replaces it and keeps its ID. A browser that refuses to construct the notification is
// WebNotificationFailed.
//
// Flutter: no counterpart (flutter_local_notifications)
func (d *Desktop) ShowNotification(ctx context.Context, spec NotificationSpec) (uint32, error) {
	if d == nil || d.notes == nil {
		return 0, errNotificationsUnavailable(noNotificationsReason)
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

	err := d.notes.Show(browserNotification{
		Title: spec.Title, Body: spec.Body, Icon: spec.Icon, Tag: notificationTag(id),
		RequireInteraction: spec.RequireInteraction, Silent: spec.Silent,
	}, ln.serial)
	if err != nil {
		d.mu.Lock()
		if d.live[id] == ln {
			delete(d.live, id)
		}
		delete(d.bySerial, ln.serial)
		d.mu.Unlock()
		return 0, err
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
// Closing one that is gone is not an error; a page without the Notifications API is
// NotificationsUnavailable.
//
// Flutter: no counterpart
func (d *Desktop) CloseNotification(_ context.Context, id uint32) error {
	if d == nil || d.notes == nil {
		return errNotificationsUnavailable(noNotificationsReason)
	}
	d.mu.Lock()
	ln := d.live[id]
	d.mu.Unlock()
	if ln != nil {
		d.end(id, ln.serial, CloseByApp, true)
	}
	return nil
}

// end takes the notification out of the shown ones, closes it in the browser when the page ended it
// (closeInBrowser), and tells its OnClose. A notification that is not the one with the serial any more
// (it was replaced or already closed) is left alone.
func (d *Desktop) end(id uint32, serial int, reason CloseReason, closeInBrowser bool) {
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
	if closeInBrowser {
		d.notes.Close(serial)
	}
	if ln.spec.OnClose != nil {
		ln.spec.OnClose(reason)
	}
}

// Clicked is the browser telling that the user clicked the notification with the serial.
//
// Flutter: no counterpart
func (d *Desktop) Clicked(serial int) {
	d.mu.Lock()
	ln := d.live[d.bySerial[serial]]
	if ln != nil && ln.serial == serial {
		ln.clicked = true
	} else {
		ln = nil
	}
	d.mu.Unlock()
	if ln != nil && ln.spec.OnClick != nil {
		ln.spec.OnClick()
	}
}

// Closed is the browser telling that the notification with the serial is gone: after a click the user
// dismissed it, and otherwise the page cannot tell why (the browser's timeout, the notification center).
//
// Flutter: no counterpart
func (d *Desktop) Closed(serial int) {
	d.mu.Lock()
	id, ok := d.bySerial[serial]
	var clicked bool
	if ok {
		if ln := d.live[id]; ln != nil {
			clicked = ln.clicked
		}
	}
	d.mu.Unlock()
	if !ok {
		return
	}
	reason := CloseUnknown
	if clicked {
		reason = CloseDismissed
	}
	d.end(id, serial, reason, false)
}

var (
	markupTag    = regexp.MustCompile(`<[^>]*>`)
	markupEscape = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&apos;", "'", "&#39;", "'", "&amp;", "&")
)

// StripMarkup makes the plain text of a notification body that has the markup of the Desktop
// Notifications specification (<b>, <i>, <u>, <a href>, <img>): the tags are dropped and the entities
// read. A Notification's body is plain text.
//
// Flutter: no counterpart
func StripMarkup(body string) string {
	return markupEscape.Replace(markupTag.ReplaceAllString(body, ""))
}

// IconURL returns the icon when it can be an image URL of the web: it has a slash (a path or a URL) or is
// a data: URI. A name from an icon theme ("dialog-information") is not, and gives an empty string, since
// the browser has no icon themes.
//
// Flutter: no counterpart
func IconURL(icon string) string {
	if strings.Contains(icon, "/") || strings.HasPrefix(icon, "data:") {
		if _, err := url.Parse(icon); err == nil {
			return icon
		}
	}
	return ""
}
