//go:build js && wasm

package web

import (
	"fmt"
	"sync"
	"syscall/js"

	"github.com/rm4n0s/errors"
)

// NewDesktop makes the web's desktop services over the browser's window.open and Notifications API. The
// API objects are looked up at each call, so a page that has none (an insecure context) or a test page
// that replaces them is served. The callbacks of the notifications are made here, once, and kept for the
// life of the program (SPEC §23.3).
//
// Flutter: no counterpart
func NewDesktop() *Desktop {
	return newDesktop(browserOpener{}, newBrowserNotifications(), nil)
}

// browserOpener is window.open as the uriOpener.
type browserOpener struct{}

// Open calls window.open. noopener makes it answer null, so the answer says nothing; an exception
// (a sandboxed frame without allow-popups) is the error.
func (browserOpener) Open(uri, target, features string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = refused{"window.open threw: " + fmt.Sprint(r)}
		}
	}()
	win := global()
	if !isObject(win.Get("open")) {
		return refused{"the page has no window.open"}
	}
	win.Call("open", uri, target, features)
	return nil
}

// browserNotifications is the Notifications API as the notificationAPI.
type browserNotifications struct {
	sink notificationSink

	// click and closeFn are the event handlers of every notification: they find the notification by the
	// serial that was put on it.
	click, closeFn js.Func
	// chain(promise, id, ok, fail) attaches the two callbacks of a request for the permission; a
	// js.Func cannot carry the call it is for.
	chain        js.Value
	okFn, failFn js.Func

	mu      sync.Mutex
	next    int
	pending map[int]func(string, error)
	open    map[int]js.Value
}

var _ notificationAPI = (*browserNotifications)(nil)

func newBrowserNotifications() *browserNotifications {
	n := &browserNotifications{pending: map[int]func(string, error){}, open: map[int]js.Value{}}
	n.chain = global().Get("Function").New("promise", "id", "ok", "fail", `
		promise.then(function (value) { ok(id, String(value)); },
			function (e) { fail(id, e && e.name ? e.name + ": " + (e.message || "") : String(e)); });`)
	n.okFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
		n.finish(int(args[0].Float()), args[1].String(), nil)
		return nil
	})
	n.failFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
		n.finish(int(args[0].Float()), "", refused{args[1].String()})
		return nil
	})
	n.click = js.FuncOf(func(this js.Value, _ []js.Value) any {
		if serial, ok := serialOf(this); ok && n.sink != nil {
			n.sink.Clicked(serial)
		}
		return nil
	})
	n.closeFn = js.FuncOf(func(this js.Value, _ []js.Value) any {
		serial, ok := serialOf(this)
		if !ok {
			return nil
		}
		n.mu.Lock()
		delete(n.open, serial)
		n.mu.Unlock()
		if n.sink != nil {
			n.sink.Closed(serial)
		}
		return nil
	})
	return n
}

// serialOf reads the serial that Show put on a notification.
func serialOf(notification js.Value) (int, bool) {
	if !isObject(notification) {
		return 0, false
	}
	v := notification.Get("glutterSerial")
	if v.Type() != js.TypeNumber {
		return 0, false
	}
	return v.Int(), true
}

// SetSink says who to tell about the clicks and the closing of the notifications.
func (n *browserNotifications) SetSink(s notificationSink) { n.sink = s }

// api is the Notification constructor of the page, or an undefined value.
func (n *browserNotifications) api() js.Value { return global().Get("Notification") }

// Permission is Notification.permission.
func (n *browserNotifications) Permission() string {
	api := n.api()
	if !isObject(api) {
		return permissionUnsupported
	}
	if p := api.Get("permission"); p.Type() == js.TypeString {
		return p.String()
	}
	return "default"
}

func (n *browserNotifications) finish(id int, permission string, err error) {
	n.mu.Lock()
	fn := n.pending[id]
	delete(n.pending, id)
	n.mu.Unlock()
	if fn != nil {
		fn(permission, err)
	}
}

// RequestPermission calls Notification.requestPermission; its promise continues in a callback made at
// startup, which runs done.
func (n *browserNotifications) RequestPermission(done func(permission string, err error)) {
	api := n.api()
	if !isObject(api) {
		done("", refused{noNotificationsReason})
		return
	}
	n.mu.Lock()
	n.next++
	id := n.next
	n.pending[id] = done
	n.mu.Unlock()
	var promise js.Value
	ok := try(func() { promise = api.Call("requestPermission") })
	if !ok || !isObject(promise) {
		n.finish(id, "", refused{"Notification.requestPermission did not give a promise"})
		return
	}
	n.chain.Invoke(promise, id, n.okFn, n.failFn)
}

// Show makes the Notification.
func (n *browserNotifications) Show(b browserNotification, serial int) (err error) {
	api := n.api()
	if !isObject(api) {
		return errors.New(tagNotificationsUnavailable, "the page cannot show notifications", "reason", noNotificationsReason)
	}
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(tagWebNotificationFailed, "the browser could not make the notification", "detail", fmt.Sprint(r))
		}
	}()
	opts := global().Get("Object").New()
	if b.Body != "" {
		opts.Set("body", b.Body)
	}
	if b.Icon != "" {
		opts.Set("icon", b.Icon)
	}
	opts.Set("tag", b.Tag)
	if b.RequireInteraction {
		opts.Set("requireInteraction", true)
	}
	if b.Silent {
		opts.Set("silent", true)
	}
	notification := api.New(b.Title, opts)
	notification.Set("glutterSerial", serial)
	notification.Set("onclick", n.click)
	notification.Set("onclose", n.closeFn)
	n.mu.Lock()
	n.open[serial] = notification
	n.mu.Unlock()
	return nil
}

// Close calls Notification.close on the notification with the serial.
func (n *browserNotifications) Close(serial int) {
	n.mu.Lock()
	notification, ok := n.open[serial]
	delete(n.open, serial)
	n.mu.Unlock()
	if ok {
		try(func() { notification.Call("close") })
	}
}
