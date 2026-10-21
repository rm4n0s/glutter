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

// call starts the method and attaches the callback. A method that throws at once (a permissions policy
// forbids the clipboard) is a failure as well.
func (n *navigatorClipboard) call(method string, arg any, done func(value js.Value, err error)) {
	api := global().Get("navigator").Get("clipboard")
	if !isObject(api) {
		done(js.Undefined(), errClipboardUnavailable())
		return
	}
	n.mu.Lock()
	n.next++
	id := n.next
	n.pending[id] = done
	n.mu.Unlock()
	var promise js.Value
	ok := try(func() {
		if arg == nil {
			promise = api.Call(method)
		} else {
			promise = api.Call(method, arg)
		}
	})
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
	n.call("readText", nil, func(v js.Value, err error) {
		var r refused
		if errors.As(err, &r) {
			done("", errClipboardReadFailed(r.detail))
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
	n.call("writeText", text, func(_ js.Value, err error) {
		var r refused
		if errors.As(err, &r) {
			done(errClipboardWriteFailed(r.detail))
			return
		}
		done(err)
	})
}
