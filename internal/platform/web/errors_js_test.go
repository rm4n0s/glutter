//go:build js && wasm

package web

import (
	"context"
	"syscall/js"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// assertWebRoute checks that err is an *errors.Error with the tag and a route that contains route.
func assertWebRoute(t *testing.T, err error, tag, route string) *errors.Error {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s (tag %s), want %s", appErr.Route(), appErr.Tag, route)
	}
	return appErr
}

// setGlobal sets a property of the JavaScript global object for the test and puts it back after. It
// defines the property, because some globals of Node (navigator) are getters that a plain set ignores.
func setGlobal(t *testing.T, name string, v any) {
	t.Helper()
	g := js.Global()
	object := g.Get("Object")
	prev := object.Call("getOwnPropertyDescriptor", g, name)
	desc := object.New()
	desc.Set("value", v)
	desc.Set("configurable", true)
	desc.Set("writable", true)
	object.Call("defineProperty", g, name, desc)
	t.Cleanup(func() {
		if prev.Type() == js.TypeUndefined {
			g.Delete(name)
			return
		}
		object.Call("defineProperty", g, name, prev)
	})
}

// TestNewWithoutAPage: under Node there is no page to draw in.
func TestNewWithoutAPage(t *testing.T) {
	_, err := New()
	_ = assertWebRoute(t, err, "BackendUnavailable", "New.BackendUnavailable")
}

// TestNewWithAHostElementThatMatchesNothing: a page whose querySelector finds nothing.
func TestNewWithAHostElementThatMatchesNothing(t *testing.T) {
	doc := js.Global().Get("Object").New()
	doc.Set("querySelector", js.FuncOf(func(js.Value, []js.Value) any { return nil }))
	setGlobal(t, "document", doc)
	cfg := js.Global().Get("Object").New()
	cfg.Set("hostElement", "#missing")
	setGlobal(t, "glutterConfig", cfg)

	_, err := New()
	appErr := assertWebRoute(t, err, "HostElementNotFound", "New.HostElementNotFound")
	if appErr.Metadata["selector"] != "#missing" {
		t.Errorf("metadata = %v, want the selector", appErr.Metadata)
	}
}

func TestRunTwiceFails(t *testing.T) {
	p := &Platform{running: true}
	err := p.Run(context.Background(), nil)
	_ = assertWebRoute(t, err, "PlatformAlreadyRunning", "Platform.Run.PlatformAlreadyRunning")
}

func TestOnlyOneRegularWindow(t *testing.T) {
	tests := []struct {
		name string
		p    *Platform
		cfg  platform.WindowConfig
	}{
		{"a second window", &Platform{win: &window{}}, platform.WindowConfig{}},
		{"a dialog", &Platform{}, platform.WindowConfig{Kind: platform.WindowKindDialog}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.p.NewWindow(tc.cfg)
			_ = assertWebRoute(t, err, "WindowUnsupported", "Platform.NewWindow.WindowUnsupported")
		})
	}
}

func TestWindowStubsFail(t *testing.T) {
	w := &window{}
	_ = assertWebRoute(t, w.StartDrag(ui.DragRequest{}), "DragUnsupported", "window.StartDrag.DragUnsupported")
	_ = assertWebRoute(t, w.KeepScreenOn(true), "IdleInhibitUnavailable", "window.KeepScreenOn.IdleInhibitUnavailable")
	if err := w.KeepScreenOn(false); err != nil {
		t.Errorf("KeepScreenOn(false) = %v, want nil", err)
	}
}

func TestScreenshotFailuresAreReported(t *testing.T) {
	tests := []struct {
		name  string
		call  func(p *Platform, resp js.Value)
		resp  map[string]any
		route string
	}{
		{"status", func(p *Platform, resp js.Value) { p.onScreenshotResponse(js.Undefined(), []js.Value{resp}) },
			map[string]any{"ok": false, "status": 500}, "Platform.onScreenshotResponse.ScreenshotWriteFailed"},
		{"network", func(p *Platform, resp js.Value) { p.onScreenshotFailure(js.Undefined(), []js.Value{resp}) },
			map[string]any{"message": "Failed to fetch"}, "Platform.onScreenshotFailure.ScreenshotWriteFailed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p := &Platform{q: newQueue()}
			got := make(chan error, 1)
			p.shotDone = func(err error) { got <- err }
			go p.q.run(ctx)
			tc.call(p, js.ValueOf(tc.resp))
			select {
			case err := <-got:
				appErr := assertWebRoute(t, err, "ScreenshotWriteFailed", tc.route)
				if detail, _ := appErr.Metadata["detail"].(string); detail == "" {
					t.Errorf("metadata = %v, want the detail", appErr.Metadata)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("the screenshot was never reported")
			}
		})
	}
}

// TestNavigatorClipboardWithoutTheAPI: under Node, navigator has no clipboard.
func TestNavigatorClipboardWithoutTheAPI(t *testing.T) {
	setGlobal(t, "navigator", js.Global().Get("Object").New())
	setGlobal(t, "ClipboardItem", clipboardItemStub())
	p := &Platform{}
	n := p.newNavigatorClipboard()
	var readErr, writeErr error
	n.ReadText(func(_ string, err error) { readErr = err })
	n.WriteText("x", func(err error) { writeErr = err })
	var readHTMLErr, writeRichErr error
	n.ReadHTML(func(_ string, err error) { readHTMLErr = err })
	n.WriteRich("x", true, "<b>x</b>", func(err error) { writeRichErr = err })
	_ = assertWebRoute(t, readHTMLErr, "WebClipboardUnavailable", "navigatorClipboard.ReadHTML->navigatorClipboard.call.WebClipboardUnavailable")
	_ = assertWebRoute(t, writeRichErr, "WebClipboardUnavailable", "navigatorClipboard.WriteRich->navigatorClipboard.call.WebClipboardUnavailable")
	_ = assertWebRoute(t, readErr, "WebClipboardUnavailable", "navigatorClipboard.ReadText->navigatorClipboard.call.WebClipboardUnavailable")
	_ = assertWebRoute(t, writeErr, "WebClipboardUnavailable", "navigatorClipboard.WriteText->navigatorClipboard.call.WebClipboardUnavailable")
}

// TestNavigatorClipboardRejections: a promise that the browser rejects (a denied permission, a page
// without focus) is a read or write failure that tells why.
func TestNavigatorClipboardRejections(t *testing.T) {
	api := js.Global().Get("Object").New()
	reject := js.Global().Get("Function").New("return Promise.reject(new Error('denied'));")
	api.Set("readText", reject)
	api.Set("writeText", reject)
	nav := js.Global().Get("Object").New()
	nav.Set("clipboard", api)
	setGlobal(t, "navigator", nav)

	p := &Platform{}
	n := p.newNavigatorClipboard()
	got := make(chan error, 2)
	n.ReadText(func(_ string, err error) { got <- err })
	n.WriteText("x", func(err error) { got <- err })
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case err := <-got:
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			seen[appErr.Tag] = true
			if detail, _ := appErr.Metadata["detail"].(string); detail != "Error: denied" {
				t.Errorf("%s: detail = %q", appErr.Tag, detail)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("a rejected promise was never reported")
		}
	}
	if !seen["WebClipboardReadFailed"] || !seen["WebClipboardWriteFailed"] {
		t.Errorf("tags seen = %v, want WebClipboardReadFailed and WebClipboardWriteFailed", seen)
	}
}

// TestShowWithoutTheNotificationAPI: under Node there is no Notification.
func TestShowWithoutTheNotificationAPI(t *testing.T) {
	n := &browserNotifications{open: map[int]js.Value{}}
	err := n.Show(browserNotification{Title: "Hi"}, 1)
	_ = assertWebRoute(t, err, "NotificationsUnavailable", "browserNotifications.Show.NotificationsUnavailable")
}

// TestShowWhenTheConstructorThrows: a browser that refuses to construct the Notification.
func TestShowWhenTheConstructorThrows(t *testing.T) {
	setGlobal(t, "Notification", js.Global().Get("Function").New("throw new TypeError('Illegal constructor');"))
	n := &browserNotifications{open: map[int]js.Value{}}
	err := n.Show(browserNotification{Title: "Hi"}, 1)
	appErr := assertWebRoute(t, err, "WebNotificationFailed", "browserNotifications.Show")
	if appErr.Metadata["detail"] == "" {
		t.Errorf("metadata = %v, want the detail", appErr.Metadata)
	}
}

// clipboardItemStub is a ClipboardItem of the shape the browsers have: the types it was made with and
// getType, which gives the blob of one.
func clipboardItemStub() js.Value {
	return js.Global().Get("Function").New("parts", `
		this.parts = parts;
		this.types = Object.keys(parts);
		this.getType = function (type) { return Promise.resolve(parts[type]); };`)
}

// settle waits for a callback that the promise chain of the clipboard calls.
func settle[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("a promise of the clipboard was never settled")
		var zero T
		return zero
	}
}

// TestNavigatorClipboardHTML: WriteRich makes one ClipboardItem with the text/html and the text/plain
// blobs, ReadHTML finds the text/html among the items that read() gives, and a browser without read() or
// without ClipboardItem is handled.
func TestNavigatorClipboardHTML(t *testing.T) {
	setGlobal(t, "ClipboardItem", clipboardItemStub())
	api := js.Global().Get("Object").New()
	written := js.Global().Get("Array").New()
	api.Set("write", js.Global().Get("Function").New("items", "this.written.push.apply(this.written, items); return Promise.resolve();"))
	api.Set("written", written)
	setNav := func(api js.Value) {
		nav := js.Global().Get("Object").New()
		nav.Set("clipboard", api)
		setGlobal(t, "navigator", nav)
	}
	setNav(api)
	p := &Platform{}
	n := p.newNavigatorClipboard()

	tests := []struct {
		name     string
		text     string
		hasText  bool
		html     string
		wantType []string
	}{
		{"text and HTML", "caf\u00e9", true, "<b>caf\u00e9 \u4e16</b>", []string{"text/plain", "text/html"}},
		{"HTML only", "", false, "<i>x</i>", []string{"text/html"}},
	}
	for i, tt := range tests {
		t.Run("write "+tt.name, func(t *testing.T) {
			done := make(chan error, 1)
			n.WriteRich(tt.text, tt.hasText, tt.html, func(err error) { done <- err })
			if err := settle(t, done); err != nil {
				t.Fatal(err)
			}
			if written.Length() != i+1 {
				t.Fatalf("%d items written, want %d", written.Length(), i+1)
			}
			item := written.Index(i)
			types := item.Get("types")
			if types.Length() != len(tt.wantType) {
				t.Fatalf("types = %v, want %v", types, tt.wantType)
			}
			texts := make(chan string, 2)
			check := func(typ, want string) {
				blob := item.Get("parts").Get(typ)
				if blob.Get("type").String() != typ {
					t.Errorf("the blob of %s has the type %s", typ, blob.Get("type").String())
				}
				then := js.FuncOf(func(_ js.Value, args []js.Value) any { texts <- args[0].String(); return nil })
				defer then.Release()
				blob.Call("text").Call("then", then)
				if got := settle(t, texts); got != want {
					t.Errorf("%s = %q, want %q", typ, got, want)
				}
			}
			check("text/html", tt.html)
			if tt.hasText {
				check("text/plain", tt.text)
			}
		})
	}

	readers := []struct {
		name  string
		read  string // the body of read()
		want  string
		fails bool
	}{
		{"an item with HTML", `return Promise.resolve([new ClipboardItem({"text/plain": new Blob(["t"], {type: "text/plain"})}), new ClipboardItem({"text/html": new Blob(["<p>caf\u00e9</p>"], {type: "text/html"})})]);`, "<p>caf\u00e9</p>", false},
		{"items without HTML", `return Promise.resolve([new ClipboardItem({"text/plain": new Blob(["t"], {type: "text/plain"})})]);`, "", false},
		{"no items", `return Promise.resolve([]);`, "", false},
		{"a refused read", `return Promise.reject(new Error("denied"));`, "", true},
		{"read() throws", `throw new Error("policy");`, "", true},
	}
	for _, tt := range readers {
		t.Run("read "+tt.name, func(t *testing.T) {
			api.Set("read", js.Global().Get("Function").New(tt.read))
			type result struct {
				html string
				err  error
			}
			done := make(chan result, 1)
			n.ReadHTML(func(html string, err error) { done <- result{html, err} })
			r := settle(t, done)
			if tt.fails {
				_ = assertWebRoute(t, r.err, "WebClipboardReadFailed", "navigatorClipboard.ReadHTML.func2.WebClipboardReadFailed")
				return
			}
			if r.err != nil || r.html != tt.want {
				t.Errorf("ReadHTML = %q, %v, want %q", r.html, r.err, tt.want)
			}
		})
	}

	t.Run("read without read()", func(t *testing.T) {
		api.Delete("read")
		done := make(chan string, 1)
		n.ReadHTML(func(html string, err error) {
			if err != nil {
				t.Error(err)
			}
			done <- html
		})
		if got := settle(t, done); got != "" {
			t.Errorf("ReadHTML = %q, want an empty one", got)
		}
	})

	t.Run("write refused", func(t *testing.T) {
		api.Set("write", js.Global().Get("Function").New("return Promise.reject(new Error('denied'));"))
		done := make(chan error, 1)
		n.WriteRich("t", true, "<b>t</b>", func(err error) { done <- err })
		appErr := assertWebRoute(t, settle(t, done), "WebClipboardWriteFailed", "navigatorClipboard.WriteRich.func2.WebClipboardWriteFailed")
		if detail, _ := appErr.Metadata["detail"].(string); detail != "Error: denied" {
			t.Errorf("detail = %q", detail)
		}
	})

	t.Run("a browser without ClipboardItem writes the text", func(t *testing.T) {
		setGlobal(t, "ClipboardItem", js.Undefined())
		plain := js.Global().Get("Object").New()
		plain.Set("writes", js.Global().Get("Array").New())
		plain.Set("writeText", js.Global().Get("Function").New("text", "this.writes.push(text); return Promise.resolve();"))
		setNav(plain)
		done := make(chan error, 1)
		n.WriteRich("only text", true, "<b>x</b>", func(err error) { done <- err })
		if err := settle(t, done); err != nil {
			t.Fatal(err)
		}
		if w := plain.Get("writes"); w.Length() != 1 || w.Index(0).String() != "only text" {
			t.Errorf("writes = %v", w)
		}
		// With no text to fall back on, it is a failure.
		var err error
		n.WriteRich("", false, "<b>x</b>", func(e error) { err = e })
		_ = assertWebRoute(t, err, "WebClipboardWriteFailed", "navigatorClipboard.WriteRich.WebClipboardWriteFailed")
	})
}
