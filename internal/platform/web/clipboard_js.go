//go:build js && wasm

package web

import (
	"sync"
	"syscall/js"

	"github.com/rm4n0s/errors"
)

// navigatorClipboard is navigator.clipboard as the clipboardAPI. A promise continues in a then callback
// made once at startup (SPEC §23.3): the callbacks find the call by the number it was given and run its
// done function, which posts the answer to the UI goroutine. The UI goroutine never waits for the
// promise.
//
// Flutter: lib/web_ui/lib/src/engine/clipboard.dart ClipboardStrategy
type navigatorClipboard struct {
	// chain is a function made once from a script: chain(promise, id, ok, fail) attaches the two
	// callbacks to the promise and tells them the id, because a js.Func cannot carry the call it is for.
	chain      js.Value
	okFn, fail js.Func
	// readHTML is a function made once from a script: readHTML(api) is a promise of the text/html
	// of the first item of navigator.clipboard.read() that has one, "" when none has or the browser
	// cannot read items (navigator.clipboard.read is missing).
	readHTML js.Value

	mu      sync.Mutex
	next    int
	pending map[int]func(value js.Value, err error)
}

var _ clipboardAPI = (*navigatorClipboard)(nil)

// newNavigatorClipboard makes the callbacks. navigator.clipboard is looked up at each call: a page in an
// insecure context has none, and the browser tests replace it.
func (p *Platform) newNavigatorClipboard() *navigatorClipboard {
	n := &navigatorClipboard{pending: map[int]func(js.Value, error){}}
	n.chain = global().Get("Function").New("promise", "id", "ok", "fail", `
		promise.then(function (value) { ok(id, value); },
			function (e) { fail(id, e && e.name ? e.name + ": " + (e.message || "") : String(e)); });`)
	n.readHTML = global().Get("Function").New("api", `
		if (typeof api.read !== "function") { return Promise.resolve(""); }
		return api.read().then(function (items) {
			for (var i = 0; i < items.length; i++) {
				if (items[i].types.indexOf("text/html") >= 0) {
					return items[i].getType("text/html").then(function (blob) { return blob.text(); });
				}
			}
			return "";
		});`)
	n.okFn = p.keep(js.FuncOf(func(_ js.Value, args []js.Value) any {
		n.finish(int(args[0].Float()), args[1], nil)
		return nil
	}))
	n.fail = p.keep(js.FuncOf(func(_ js.Value, args []js.Value) any {
		n.finish(int(args[0].Float()), js.Undefined(), refused{args[1].String()})
		return nil
	}))
	return n
}

// refused is the reason a promise was rejected; the method that made the call wraps it in its own error.
type refused struct{ detail string }

func (r refused) Error() string { return r.detail }

func (n *navigatorClipboard) finish(id int, value js.Value, err error) {
	n.mu.Lock()
	fn := n.pending[id]
	delete(n.pending, id)
	n.mu.Unlock()
	if fn != nil {
		fn(value, err)
	}
}

// call starts a call that start makes on navigator.clipboard (it returns the promise; method names the
// call in the failure) and attaches the callback. A call that throws at once (a permissions policy
// forbids the clipboard) is a failure as well.
func (n *navigatorClipboard) call(method string, start func(api js.Value) js.Value, done func(value js.Value, err error)) {
	api := global().Get("navigator").Get("clipboard")
	if !isObject(api) {
		done(js.Undefined(), errors.New(tagWebClipboardUnavailable,
			"the page has no navigator.clipboard: it needs a secure context (https or localhost)"))
		return
	}
	n.mu.Lock()
	n.next++
	id := n.next
	n.pending[id] = done
	n.mu.Unlock()
	var promise js.Value
	ok := try(func() { promise = start(api) })
	if !ok || !isObject(promise) {
		n.finish(id, js.Undefined(), refused{"navigator.clipboard." + method + " threw"})
		return
	}
	n.chain.Invoke(promise, id, n.okFn, n.fail)
}

// ReadText calls navigator.clipboard.readText.
//
// Flutter: lib/web_ui/lib/src/engine/clipboard.dart ClipboardStrategy.getData
func (n *navigatorClipboard) ReadText(done func(text string, err error)) {
	n.call("readText", func(api js.Value) js.Value { return api.Call("readText") }, func(v js.Value, err error) {
		var r refused
		if errors.As(err, &r) {
			done("", errors.New(tagWebClipboardReadFailed, "navigator.clipboard.readText failed", "detail", r.detail))
			return
		}
		if err != nil {
			done("", err)
			return
		}
		text := ""
		if v.Type() == js.TypeString {
			text = v.String()
		}
		done(text, nil)
	})
}

// WriteText calls navigator.clipboard.writeText.
//
// Flutter: lib/web_ui/lib/src/engine/clipboard.dart ClipboardStrategy.setData
func (n *navigatorClipboard) WriteText(text string, done func(err error)) {
	n.call("writeText", func(api js.Value) js.Value { return api.Call("writeText", text) }, func(_ js.Value, err error) {
		var r refused
		if errors.As(err, &r) {
			done(errors.New(tagWebClipboardWriteFailed, "navigator.clipboard.writeText failed", "detail", r.detail))
			return
		}
		done(err)
	})
}

// ReadHTML reads the text/html of the clipboard with navigator.clipboard.read (the browser may ask the
// user, or refuse a page without a user action: WebClipboardReadFailed). A browser that cannot read
// items, or a clipboard with no HTML, gives "".
//
// Flutter: no counterpart
func (n *navigatorClipboard) ReadHTML(done func(html string, err error)) {
	n.call("read", func(api js.Value) js.Value { return n.readHTML.Invoke(api) }, func(v js.Value, err error) {
		var r refused
		if errors.As(err, &r) {
			done("", errors.New(tagWebClipboardReadFailed, "navigator.clipboard.read failed", "detail", r.detail))
			return
		}
		if err != nil {
			done("", err)
			return
		}
		html := ""
		if v.Type() == js.TypeString {
			html = v.String()
		}
		done(html, nil)
	})
}

// WriteRich writes a ClipboardItem with the text/html and, when hasText, the text/plain of one copy
// with navigator.clipboard.write. A browser without ClipboardItem writes the text only; with no text
// to fall back on that is a WebClipboardWriteFailed.
//
// Flutter: no counterpart
func (n *navigatorClipboard) WriteRich(text string, hasText bool, html string, done func(err error)) {
	if !isObject(global().Get("ClipboardItem")) {
		if hasText {
			n.WriteText(text, done)
			return
		}
		done(errors.New(tagWebClipboardWriteFailed, "navigator.clipboard.write failed", "detail", "the browser has no ClipboardItem"))
		return
	}
	n.call("write", func(api js.Value) js.Value {
		blob := func(mime, data string) js.Value {
			return global().Get("Blob").New([]any{data}, map[string]any{"type": mime})
		}
		parts := global().Get("Object").New()
		if hasText {
			parts.Set("text/plain", blob("text/plain", text))
		}
		parts.Set("text/html", blob("text/html", html))
		return api.Call("write", []any{global().Get("ClipboardItem").New(parts)})
	}, func(_ js.Value, err error) {
		var r refused
		if errors.As(err, &r) {
			done(errors.New(tagWebClipboardWriteFailed, "navigator.clipboard.write failed", "detail", r.detail))
			return
		}
		done(err)
	})
}
