//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"syscall/js"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/internal/platform/web"
	m "github.com/rm4n0s/glutter/material"
)

// stubs replaces window.open and the Notification constructor. window.open records its arguments, and
// throws for the host "throws.example" as a sandboxed frame without allow-popups does. A Notification
// records what it was made with, and its permission prompt is a promise that the test settles by
// calling answer.
const stubs = `
	globalThis.__opened = [];
	window.open = function (uri, target, features) {
		globalThis.__opened.push([uri, target, features]);
		if (uri.indexOf("throws.example") >= 0) { throw new DOMException("popups are not allowed", "SecurityError"); }
		return null;
	};
	globalThis.__prompts = [];
	globalThis.__notes = [];
	class StubNotification {
		constructor(title, options) {
			this.title = title;
			this.options = options;
			this.closed = false;
			globalThis.__notes.push(this);
		}
		close() {
			this.closed = true;
			if (this.onclose) { this.onclose(new Event("close")); }
		}
		static requestPermission() {
			return new Promise(function (resolve) { globalThis.__prompts.push(resolve); });
		}
	}
	StubNotification.permission = "default";
	window.Notification = StubNotification;
	globalThis.__answer = function (n, permission) {
		StubNotification.permission = permission;
		globalThis.__prompts[n](permission);
	};`

// noteOptions are the options a Notification was made with.
type noteOptions struct {
	Title              string `json:"title"`
	Body               string `json:"body"`
	Icon               string `json:"icon"`
	Tag                string `json:"tag"`
	RequireInteraction bool   `json:"requireInteraction"`
	Silent             bool   `json:"silent"`
}

// observations is what the program posts as the result "desktop".
type observations struct {
	// Opened is what window.open got, in order: an https URI, and a URI for which the stub throws (a
	// javascript: URI and a URI without a scheme never reach it). OpenTags are the tags of the answers
	// of OpenURI for the https, javascript: and throwing URIs ("" for success), and InvalidTag that of
	// the URI without a scheme.
	Opened     [][]string `json:"opened"`
	OpenTags   []string   `json:"openTags"`
	InvalidTag string     `json:"invalidTag"`
	// Refused is the tag ShowNotification returned when the user denied the permission, RefusedAgain the
	// tag of the next call (which must not ask again), PromptsAfter how many times the browser was asked
	// for the permission by then, and NotesWhileRefused how many notifications were made.
	Refused           string `json:"refused"`
	RefusedAgain      string `json:"refusedAgain"`
	PromptsAfter      int    `json:"promptsAfter"`
	NotesWhileRefused int    `json:"notesWhileRefused"`
	// Granted is the tag of the call that the user granted ("" for success), and First the options of the
	// notification that it made.
	Granted string      `json:"granted"`
	First   noteOptions `json:"first"`
	// Actions are the IDs the OnAction callback got after a click, Reasons the close reasons that the
	// OnClosed callbacks got in order (the browser closing the clicked notification, then the app closing
	// another), and BrowserClosed whether the app's CloseNotification closed that one in the browser.
	Actions       []string `json:"actions"`
	Reasons       []int    `json:"reasons"`
	BrowserClosed bool     `json:"browserClosed"`
	// CloseTag is the tag of CloseNotification ("" for success).
	CloseTag string `json:"closeTag"`
	// Replaced is the options of the notification made with ReplacesID (same tag), SameID whether the ID
	// was kept.
	Replaced noteOptions `json:"replaced"`
	SameID   bool        `json:"sameID"`
	// NoAPI is the tag of ShowNotification on a page that has no Notification.
	NoAPI string `json:"noAPI"`
	// CallbacksOnUIThread is whether every callback ran on the UI goroutine.
	CallbacksOnUIThread bool `json:"callbacksOnUIThread"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	p := newProbe()
	go func() { postResult("desktop", observe(p)) }()
	return m.RunApp(&page{p: p}, m.RunAppOpts{Window: m.WindowConfig{Title: "Desktop test"}})
}

func eval(code string) js.Value { return js.Global().Get("Function").New(code).Invoke() }

func noteOf(i int) noteOptions {
	n := js.Global().Get("__notes").Index(i)
	o := n.Get("options")
	get := func(k string) string {
		if v := o.Get(k); v.Type() == js.TypeString {
			return v.String()
		}
		return "" // an option that was not given
	}
	return noteOptions{
		Title: n.Get("title").String(), Body: get("body"), Icon: get("icon"), Tag: get("tag"),
		RequireInteraction: o.Get("requireInteraction").Truthy(), Silent: o.Get("silent").Truthy(),
	}
}

func count(name string) int { return js.Global().Get(name).Length() }

// waitFor polls until cond holds.
func waitFor(cond func() bool) bool {
	for i := 0; i < 200; i++ {
		if cond() {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

// observe waits for the app and calls package desktop as an app does, from a goroutine that is not the
// UI's.
func observe(p *probe) observations {
	var o observations
	<-p.ready
	time.Sleep(500 * time.Millisecond) // the first frames
	eval(stubs)
	ctx := context.Background()

	// OpenURI: window.open(uri, "_blank", "noopener"); a script URI is refused before it; a throwing
	// window.open is OpenURIUnavailable; a URI without a scheme is invalid.
	o.OpenTags = append(o.OpenTags,
		tagOf(desktop.OpenURI(ctx, p.bc, "https://example.org/a?b=c")),
		tagOf(desktop.OpenURI(ctx, p.bc, "javascript:alert(1)")),
		tagOf(desktop.OpenURI(ctx, p.bc, "https://throws.example/")))
	o.InvalidTag = tagOf(desktop.OpenURI(ctx, p.bc, "example.org"))
	opened := js.Global().Get("__opened")
	for i := 0; i < opened.Length(); i++ {
		call := opened.Index(i)
		o.Opened = append(o.Opened, []string{call.Index(0).String(), call.Index(1).String(), call.Index(2).String()})
	}

	type result struct {
		id  desktop.NotificationID
		err error
	}
	show := func(summary string, opts desktop.NotificationOpts) chan result {
		ch := make(chan result, 1)
		go func() {
			id, err := desktop.ShowNotification(ctx, p.bc, summary, opts)
			ch <- result{id, err}
		}()
		return ch
	}

	// A permission that the user refuses: NotificationsUnavailable, no notification, and the browser is not
	// asked again.
	refused := show("Hello", desktop.NotificationOpts{})
	waitFor(func() bool { return count("__prompts") == 1 })
	js.Global().Call("__answer", 0, "denied")
	o.Refused = tagOf((<-refused).err)
	o.RefusedAgain = tagOf((<-show("Hello again", desktop.NotificationOpts{})).err)
	o.PromptsAfter, o.NotesWhileRefused = count("__prompts"), count("__notes")

	// The user changes their mind in the browser's settings (the stub goes back to "default"), and the
	// prompt comes back and is granted.
	js.Global().Get("Notification").Set("permission", "default")
	var mu sync.Mutex
	onUI := true
	seen := func() {
		mu.Lock()
		defer mu.Unlock()
		onUI = onUI && p.isUIThread()
	}
	granted := show("Build done", desktop.NotificationOpts{
		Body: "<b>2</b> files", BodyMarkup: true, Icon: "/icons/done.png", Urgency: desktop.NotificationUrgencyCritical,
		Actions: []desktop.NotificationAction{{ID: "default", Label: "Open"}, {ID: "undo", Label: "Undo"}},
		OnAction: func(id string) {
			seen()
			mu.Lock()
			o.Actions = append(o.Actions, id)
			mu.Unlock()
		},
		OnClosed: func(r desktop.NotificationCloseReason) {
			seen()
			mu.Lock()
			o.Reasons = append(o.Reasons, int(r))
			mu.Unlock()
		},
	})
	waitFor(func() bool { return count("__prompts") == 2 })
	js.Global().Call("__answer", 1, "granted")
	g := <-granted
	o.Granted = tagOf(g.err)
	if g.err != nil {
		return o
	}
	o.First = noteOf(0)

	// The user clicks the notification (the default action) and the browser closes it.
	notes := js.Global().Get("__notes")
	notes.Index(0).Call("onclick", js.Global().Get("Event").New("click"))
	waitFor(func() bool { mu.Lock(); defer mu.Unlock(); return len(o.Actions) == 1 })
	notes.Index(0).Call("onclose", js.Global().Get("Event").New("close"))
	waitFor(func() bool { mu.Lock(); defer mu.Unlock(); return len(o.Reasons) == 1 })

	// The app closes a notification: the browser is told, and OnClosed gets ByApp.
	second := <-show("Second", desktop.NotificationOpts{OnClosed: func(r desktop.NotificationCloseReason) {
		seen()
		mu.Lock()
		o.Reasons = append(o.Reasons, int(r))
		mu.Unlock()
	}})
	if second.err == nil {
		o.CloseTag = tagOf(desktop.CloseNotification(ctx, p.bc, second.id))
		waitFor(func() bool { mu.Lock(); defer mu.Unlock(); return len(o.Reasons) == 2 })
		o.BrowserClosed = notes.Index(1).Get("closed").Truthy()
	}

	// Replacing keeps the tag, so that the browser replaces, and the ID.
	third := <-show("Download", desktop.NotificationOpts{})
	repl := <-show("Download 50%", desktop.NotificationOpts{ReplacesID: third.id})
	o.Replaced = noteOf(3)
	o.SameID = third.err == nil && repl.err == nil && repl.id == third.id && noteOf(2).Tag == o.Replaced.Tag

	// A page without the Notifications API.
	eval("delete window.Notification;")
	o.NoAPI = tagOf((<-show("Nobody sees this", desktop.NotificationOpts{})).err)
	mu.Lock()
	o.CallbacksOnUIThread = onUI
	mu.Unlock()
	return o
}

// tagOf is the tag of an error, empty for none.
func tagOf(err error) string {
	if err == nil {
		return ""
	}
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Tag
	}
	return err.Error()
}

// postResult posts a result to the test server. The answer is not waited for.
func postResult(kind string, data any) {
	body, err := json.Marshal(map[string]any{"kind": kind, "data": data})
	if err != nil {
		panic(err)
	}
	init := map[string]any{"method": "POST", "headers": map[string]any{"Content-Type": "application/json"}, "body": string(body)}
	js.Global().Call("fetch", web.ResultsPath, init)
}
