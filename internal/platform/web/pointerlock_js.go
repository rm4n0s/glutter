//go:build js && wasm

package web

import (
	"syscall/js"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The pointer lock of the page (SPEC §27.6) over the browser's Pointer Lock API:
// canvas.requestPointerLock({unadjustedMovement: true}), the pointerlockchange and pointerlockerror
// events of the document and document.exitPointerLock(). The browser asks the user nothing for the
// canvas of a page that the user is working with, but it wants a user activation (a click or a key)
// and ends the lock when the user presses Escape or the page loses the focus. The UI goroutine never
// waits for the promise that Chromium returns: its two callbacks queue their work.

// lockState is what the platform keeps about the lock.
type lockState struct {
	state pointerLockState
	// changeFn and errorFn are the listeners of pointerlockchange and pointerlockerror.
	changeFn, errorFn js.Func
	// rejectFn is the callback of the rejection of the promise of requestPointerLock.
	rejectFn js.Func
	// promised says the browser answers requests with a promise (Chromium): its rejection, not
	// pointerlockerror, is then the signal of a refusal, because it says why. plain says the
	// request was repeated without unadjustedMovement.
	promised, plain bool
}

var _ platform.PointerLocker = (*window)(nil)

// makeLockFuncs makes the callbacks of the lock once, at startup (SPEC §23.3).
func (p *Platform) makeLockFuncs() {
	p.lock.changeFn = p.keep(js.FuncOf(p.onLockChange))
	p.lock.errorFn = p.keep(js.FuncOf(p.onLockError))
	p.lock.rejectFn = p.keep(js.FuncOf(p.onLockRejected))
}

// lockListeners are the listeners of the document that the lock needs.
func (p *Platform) lockListeners() []listener {
	return []listener{
		{event: "pointerlockchange", fn: p.lock.changeFn},
		{event: "pointerlockerror", fn: p.lock.errorFn},
	}
}

// LockPointer asks the browser to lock the pointer to the canvas. It fails with
// PointerLockUnavailable when the browser has no Pointer Lock API, and with PointerLockDenied when
// the page has no user activation (the browser would refuse): call it from the handler of a click
// or a key. The browser's answer comes later; a refusal, and the user's Escape, reach the handler as
// OnPointerLockEnded. The mouse is reported unaccelerated where the browser can (unadjustedMovement),
// else as the system moves it.
//
// Flutter: no counterpart
func (w *window) LockPointer() error {
	p := w.p
	supported := w.canvas.Get("requestPointerLock").Type() == js.TypeFunction
	switch p.lock.state.request(supported, p.userActive()) {
	case lockUnsupported:
		return errors.New(ui.TagPointerLockUnavailable, "the browser has no Pointer Lock API")
	case lockNoGesture:
		return errors.New(ui.TagPointerLockDenied, "the browser locks the pointer only after a click or a key of the user")
	}
	if p.in.pointers == nil {
		p.lock.state.unlock()
		return errors.New(ui.TagPointerLockUnavailable, "the page does not take input yet")
	}
	p.lock.plain = false
	p.requestLock(w.canvas, map[string]any{"unadjustedMovement": true})
	if data := p.in.pointers.Lock(w.geo.CSSWidth/2, w.geo.CSSHeight/2, p.viewDPR()); len(data) > 0 && p.handler != nil {
		p.handler.OnPointer(data)
	}
	return nil
}

// userActive says whether the page has a user activation now. A browser without navigator.userActivation
// decides for itself, so the request is made.
func (p *Platform) userActive() bool {
	ua := global().Get("navigator").Get("userActivation")
	if !isObject(ua) {
		return true
	}
	return ua.Get("isActive").Truthy()
}

// requestLock calls requestPointerLock and hooks the rejection of the promise it returns.
func (p *Platform) requestLock(canvas js.Value, opts any) {
	var ret js.Value
	ok := try(func() {
		if opts == nil {
			ret = canvas.Call("requestPointerLock")
		} else {
			ret = canvas.Call("requestPointerLock", opts)
		}
	})
	if !ok {
		p.q.post(func() { p.lockRefused() })
		return
	}
	p.lock.promised = isObject(ret) && ret.Get("catch").Type() == js.TypeFunction
	if p.lock.promised {
		ret.Call("catch", p.lock.rejectFn)
	}
}

// UnlockPointer lets the pointer go. The handler is not told.
//
// Flutter: no counterpart
func (w *window) UnlockPointer() {
	p := w.p
	if p.lock.state.unlock() {
		try(func() { global().Get("document").Call("exitPointerLock") })
	}
	if p.in.pointers != nil {
		p.in.pointers.Unlock()
	}
}

// onLockChange handles pointerlockchange: the lock was granted, or it ended.
func (p *Platform) onLockChange(js.Value, []js.Value) any {
	doc := global().Get("document")
	var ours bool
	if p.win != nil {
		ours = doc.Get("pointerLockElement").Equal(p.win.canvas)
	}
	p.q.post(func() {
		ended, leave := p.lock.state.changed(ours)
		if leave {
			try(func() { doc.Call("exitPointerLock") })
		}
		if ended {
			p.lockEnded()
		}
	})
	return nil
}

// onLockError handles pointerlockerror: with a promise the rejection says it.
func (p *Platform) onLockError(js.Value, []js.Value) any {
	p.q.post(func() {
		if !p.lock.promised {
			p.lockRefused()
		}
	})
	return nil
}

// onLockRejected handles the rejection of the request. A browser that cannot give the unaccelerated
// motion (NotSupportedError) is asked again for the plain lock.
func (p *Platform) onLockRejected(_ js.Value, args []js.Value) any {
	name := ""
	if len(args) > 0 && isObject(args[0]) {
		name = args[0].Get("name").String()
	}
	p.q.post(func() {
		if name == "NotSupportedError" && !p.lock.plain && p.win != nil && p.lock.state.pending() {
			p.lock.plain = true
			p.requestLock(p.win.canvas, nil)
			return
		}
		p.lockRefused()
	})
	return nil
}

// lockRefused ends a lock that the browser refused.
func (p *Platform) lockRefused() {
	if p.lock.state.failed() {
		p.lockEnded()
	}
}

// lockEnded tells the handler that the lock ended by itself.
func (p *Platform) lockEnded() {
	if p.in.pointers != nil {
		p.in.pointers.Unlock()
	}
	if h, ok := p.handler.(platform.PointerLockHandler); ok && p.win != nil {
		h.OnPointerLockEnded(p.win.id)
	}
}
