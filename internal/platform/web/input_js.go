//go:build js && wasm

package web

import (
	"sync"
	"syscall/js"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The input of the page (SPEC §23.4). Every browser event has one js.Func, made in New, and the
// callback only copies the fields of the event into the plain Go values of pointer.go and
// keyboard.go and queues the conversion for the UI goroutine. The two kinds of event that the
// browser wants an answer to, keydown/keyup (preventDefault when the framework handled the key) and
// wheel (preventDefault unless a scrollable at its boundary lets the page scroll), wait for the UI
// goroutine's answer (SPEC §23.3): that is safe because once Run has installed the listeners the UI
// goroutine never waits for a promise.
//
// A callback that runs from a function on the UI goroutine (the function dispatched an event, and the
// browser calls the listener on the goroutine that dispatched it) cannot wait for the queue, which
// only that function can serve: it does the work at once. A program that dispatches synthetic events
// from another goroutine, as the browser tests do, takes the first path.

// inputState is what the input of the platform holds.
type inputState struct {
	// mu guards stop.
	mu sync.Mutex
	// stop is closed when Run ends; nil while it does not run.
	stop chan struct{}

	keys     *KeyConverter
	pointers *PointerInput
	// lineHeight is the height of a line of scroll in physical pixels; zero until a wheel event
	// reports lines.
	lineHeight float64

	// fns are all the listeners the input can have, and active the ones of this run.
	fns, active []listener
	// preventFn prevents the default action of the contextmenu event while the browser's context
	// menu is disabled.
	preventFn js.Func
	menuOff   bool
}

// listener is an event listener on a target.
type listener struct {
	target  js.Value
	event   string
	fn      js.Func
	options any
}

// makeInputFuncs makes the callbacks of the input once, at startup (SPEC §23.3).
func (p *Platform) makeInputFuncs() {
	down := p.keep(js.FuncOf(p.onPointerDown))
	move := p.keep(js.FuncOf(p.onPointerMove))
	up := p.keep(js.FuncOf(p.onPointerUp))
	cancel := p.keep(js.FuncOf(p.onPointerCancel))
	leave := p.keep(js.FuncOf(p.onPointerLeave))
	wheel := p.keep(js.FuncOf(p.onWheel))
	key := p.keep(js.FuncOf(p.onKey))
	touch := p.keep(js.FuncOf(p.onTouchEnd))
	p.in.preventFn = p.keep(js.FuncOf(func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		return nil
	}))
	// The wheel listener is not passive, so that preventDefault can keep the page from scrolling.
	nonPassive := map[string]any{"passive": false}
	p.in.fns = []listener{
		{event: "pointerdown", fn: down},
		{event: "pointermove", fn: move},
		{event: "pointerup", fn: up},
		{event: "pointercancel", fn: cancel},
		{event: "pointerleave", fn: leave},
		{event: "wheel", fn: wheel, options: nonPassive},
		{event: "keydown", fn: key},
		{event: "keyup", fn: key},
		{event: "touchend", fn: touch},
		{event: "touchcancel", fn: touch},
	}
}

// targetOf is where a listener goes: the pointer and wheel events to the canvas (the pointer is
// captured by it on down, so the moves and the up arrive even outside it), the key events to the
// host element (the canvas and the text field of step 21.3 are in it), the touch events to the
// host.
func (p *Platform) targetOf(event string) js.Value {
	switch event {
	case "keydown", "keyup", "touchend", "touchcancel":
		return p.host
	}
	return p.win.canvas
}

// installInput installs the listeners of the input and makes the state of the conversions.
func (p *Platform) installInput() {
	p.in.mu.Lock()
	p.in.stop = make(chan struct{})
	p.in.mu.Unlock()
	p.in.keys = NewKeyConverter(p.dispatchKey, p.os, p.scheduleKey)
	p.in.pointers = NewPointerInput(p.win.id, p.os, PointerInputOptions{Keys: p.in.keys, Firefox: p.firefox})
	p.in.active = p.in.fns
	if p.os != OSIOS {
		// Only iOS WebKit abandons touches (PointerInput.CancelAbandonedTouches).
		p.in.active = p.in.fns[:len(p.in.fns)-2]
	}
	for i := range p.in.active {
		l := &p.in.active[i]
		l.target = p.targetOf(l.event)
		if l.options != nil {
			l.target.Call("addEventListener", l.event, l.fn, l.options)
		} else {
			l.target.Call("addEventListener", l.event, l.fn)
		}
	}
	// The host is focusable, and a click puts the focus on it.
	if !p.host.Call("hasAttribute", "tabindex").Bool() {
		p.host.Call("setAttribute", "tabindex", "0")
	}
	setStyle(p.host, "outline", "none")
	if p.fullPage {
		try(func() { p.host.Call("focus", map[string]any{"preventScroll": true}) })
	}
	if p.in.menuOff {
		p.win.canvas.Call("addEventListener", "contextmenu", p.in.preventFn)
	}
}

// removeInput takes the listeners away and wakes the callbacks that wait for the UI goroutine.
func (p *Platform) removeInput() {
	for _, l := range p.in.active {
		if isObject(l.target) {
			l.target.Call("removeEventListener", l.event, l.fn)
		}
	}
	if p.win != nil && p.in.menuOff {
		p.win.canvas.Call("removeEventListener", "contextmenu", p.in.preventFn)
	}
	p.in.mu.Lock()
	if p.in.stop != nil {
		close(p.in.stop)
		p.in.stop = nil
	}
	p.in.mu.Unlock()
}

// callSync runs fn on the UI goroutine and waits for it to finish, which is how a callback that
// needs an answer gets one. It reports false, without having run fn, when Run is not running or
// ended before fn ran.
func (p *Platform) callSync(fn func()) bool {
	p.in.mu.Lock()
	stop := p.in.stop
	p.in.mu.Unlock()
	if stop == nil {
		return false
	}
	if p.q.busy() {
		fn() // called from the UI goroutine's own function: see the comment at the top
		return true
	}
	done := make(chan struct{})
	p.q.post(func() {
		fn()
		close(done)
	})
	select {
	case <-done:
		return true
	case <-stop:
		return false
	}
}

// viewDPR is the device pixel ratio the view's pixels are measured with.
func (p *Platform) viewDPR() float64 {
	if w := p.win; w != nil && w.geo.DPR > 0 {
		return w.geo.DPR
	}
	return p.dpr()
}

// readPointerEvent copies the fields of a PointerEvent. The position is relative to the canvas.
func readPointerEvent(ev js.Value, rect js.Value) PointerEvent {
	left, top := number(rect.Get("left"), 0), number(rect.Get("top"), 0)
	e := PointerEvent{
		PointerType: ev.Get("pointerType").String(),
		PointerID:   int(number(ev.Get("pointerId"), 0)),
		Button:      int(number(ev.Get("button"), 0)),
		Buttons:     int(number(ev.Get("buttons"), 0)),
		X:           number(ev.Get("clientX"), 0) - left,
		Y:           number(ev.Get("clientY"), 0) - top,
		Pressure:    number(ev.Get("pressure"), 0),
		TiltX:       number(ev.Get("tiltX"), 0),
		TiltY:       number(ev.Get("tiltY"), 0),
		TimeStamp:   number(ev.Get("timeStamp"), 0),
		Alt:         ev.Get("altKey").Truthy(),
		Ctrl:        ev.Get("ctrlKey").Truthy(),
		Meta:        ev.Get("metaKey").Truthy(),
		Shift:       ev.Get("shiftKey").Truthy(),
	}
	return e
}

// readCoalesced adds the coalesced events of a pointermove to e, when the browser has them.
func readCoalesced(ev, rect js.Value, e *PointerEvent) {
	if ev.Get("getCoalescedEvents").Type() != js.TypeFunction {
		return
	}
	list := ev.Call("getCoalescedEvents")
	for i := 0; i < list.Length(); i++ {
		c := readPointerEvent(list.Index(i), rect)
		if c.PointerType == "" {
			c.PointerType = e.PointerType
		}
		e.Coalesced = append(e.Coalesced, c)
	}
}

// canvasRect is the position of the canvas in the page now.
func (p *Platform) canvasRect() js.Value { return p.win.canvas.Call("getBoundingClientRect") }

// postPointer queues the conversion of a pointer event for the UI goroutine.
func (p *Platform) postPointer(convert func(in *PointerInput, dpr float64) []ui.PointerData) {
	p.q.post(func() {
		if p.handler == nil || p.in.pointers == nil || p.win == nil {
			return
		}
		if data := convert(p.in.pointers, p.viewDPR()); len(data) > 0 {
			p.handler.OnPointer(data)
		}
	})
}

// onPointerDown handles pointerdown: the canvas captures the pointer, so that the moves and the
// up come to it even outside, and gets the focus; the default action (selecting, dragging) is
// prevented.
func (p *Platform) onPointerDown(_ js.Value, args []js.Value) any {
	ev := args[0]
	e := readPointerEvent(ev, p.canvasRect())
	// A pointer that is not active (a synthetic event) cannot be captured; the events then
	// still come to the canvas while the pointer is over it.
	try(func() { p.win.canvas.Call("setPointerCapture", ev.Get("pointerId")) })
	ev.Call("preventDefault")
	if p.text.on {
		p.text.focus() // the keyboard stays with the text field
	} else {
		try(func() { p.host.Call("focus", map[string]any{"preventScroll": true}) })
	}
	p.postPointer(func(in *PointerInput, dpr float64) []ui.PointerData { return in.Down(e, dpr) })
	return nil
}

func (p *Platform) onPointerMove(_ js.Value, args []js.Value) any {
	ev := args[0]
	rect := p.canvasRect()
	e := readPointerEvent(ev, rect)
	readCoalesced(ev, rect, &e)
	p.postPointer(func(in *PointerInput, dpr float64) []ui.PointerData { return in.Move(e, dpr) })
	return nil
}

func (p *Platform) onPointerUp(_ js.Value, args []js.Value) any {
	e := readPointerEvent(args[0], p.canvasRect())
	p.postPointer(func(in *PointerInput, dpr float64) []ui.PointerData { return in.Up(e, dpr) })
	return nil
}

func (p *Platform) onPointerCancel(_ js.Value, args []js.Value) any {
	e := readPointerEvent(args[0], p.canvasRect())
	p.postPointer(func(in *PointerInput, dpr float64) []ui.PointerData { return in.Cancel(e, dpr) })
	return nil
}

func (p *Platform) onPointerLeave(_ js.Value, args []js.Value) any {
	e := readPointerEvent(args[0], p.canvasRect())
	p.postPointer(func(in *PointerInput, dpr float64) []ui.PointerData { return in.Leave(e, dpr) })
	return nil
}

// onTouchEnd handles touchend and touchcancel: the safety net for the touches that iOS abandons.
func (p *Platform) onTouchEnd(_ js.Value, args []js.Value) any {
	ev := args[0]
	var ids []int
	if touches := ev.Get("touches"); isObject(touches) {
		for i := 0; i < touches.Length(); i++ {
			ids = append(ids, int(number(touches.Index(i).Get("identifier"), 0)))
		}
	}
	ts := number(ev.Get("timeStamp"), 0)
	p.postPointer(func(in *PointerInput, _ float64) []ui.PointerData { return in.CancelAbandonedTouches(ids, ts) })
	return nil
}

// onWheel handles wheel: the conversion waits for the UI goroutine, because the answer decides
// whether the page scrolls.
func (p *Platform) onWheel(_ js.Value, args []js.Value) any {
	ev := args[0]
	rect := p.canvasRect()
	e := WheelEvent{
		X:         number(ev.Get("clientX"), 0) - number(rect.Get("left"), 0),
		Y:         number(ev.Get("clientY"), 0) - number(rect.Get("top"), 0),
		DeltaX:    number(ev.Get("deltaX"), 0),
		DeltaY:    number(ev.Get("deltaY"), 0),
		DeltaMode: int(number(ev.Get("deltaMode"), 0)),
		Buttons:   int(number(ev.Get("buttons"), 0)),
		Ctrl:      ev.Get("ctrlKey").Truthy(),
		TimeStamp: number(ev.Get("timeStamp"), 0),
	}
	if wx, wy := ev.Get("wheelDeltaX"), ev.Get("wheelDeltaY"); wx.Type() == js.TypeNumber && wy.Type() == js.TypeNumber {
		e.WheelDeltaX, e.WheelDeltaY, e.HasWheelDelta = wx.Float(), wy.Float(), true
	}
	if e.DeltaMode == 1 && p.in.lineHeight == 0 {
		p.in.lineHeight = p.scrollLineHeight()
	}
	var res *WheelResult
	if !p.callSync(func() { res = p.handleWheel(e) }) {
		return nil
	}
	if res.PreventDefault(p.fullPage && p.inFrame()) {
		ev.Call("preventDefault")
	}
	return nil
}

// handleWheel converts the wheel event and gives it to the framework at once.
func (p *Platform) handleWheel(e WheelEvent) *WheelResult {
	res := &WheelResult{}
	if p.handler == nil || p.in.pointers == nil || p.win == nil {
		res.AllowedDefault = true
		return res
	}
	pw, ph := p.win.geo.PhysicalSize()
	data, res := p.in.pointers.Wheel(e, WheelContext{
		DevicePixelRatio: p.viewDPR(), PhysicalWidth: float64(pw), PhysicalHeight: float64(ph), LineHeight: p.in.lineHeight,
	})
	if s, ok := p.handler.(platform.SyncInputHandler); ok {
		s.OnPointerSync(data)
	} else {
		p.handler.OnPointer(data)
	}
	return res
}

// inFrame says whether the page is in a frame (an iframe, as the browser tests and many sites
// embed an app).
//
// Flutter: lib/web_ui/lib/src/engine/dom.dart isEmbeddedInIframe
func (p *Platform) inFrame() bool {
	inFrame := false
	try(func() { inFrame = !global().Get("self").Equal(global().Get("top")) })
	return inFrame
}

// scrollLineHeight is the height of a line of scroll for the browsers that report lines instead
// of pixels (Firefox): a quarter of the default font size, found with a hidden probe element.
//
// Flutter: lib/web_ui/lib/src/engine/pointer_binding.dart _WheelEventListenerMixin._computeDefaultScrollLineHeight
func (p *Platform) scrollLineHeight() float64 {
	doc := global().Get("document")
	height := DefaultScrollLineHeight
	probe := doc.Call("createElement", "div")
	setStyle(probe, "font-size", "initial")
	setStyle(probe, "display", "none")
	doc.Get("body").Call("appendChild", probe)
	size := global().Call("getComputedStyle", probe).Get("fontSize").String()
	probe.Call("remove")
	if px, ok := ParsePixels(size); ok {
		height = px / 4.0
	}
	return height
}

// readKeyEvent copies the fields of a KeyboardEvent.
func readKeyEvent(ev js.Value) KeyEvent {
	key, code := ev.Get("key"), ev.Get("code")
	return KeyEvent{
		Down:       ev.Get("type").String() == "keydown",
		Key:        stringOf(key),
		Code:       stringOf(code),
		Incomplete: key.Type() != js.TypeString || code.Type() != js.TypeString,
		Repeat:     ev.Get("repeat").Truthy(),
		Location:   int(number(ev.Get("location"), 0)),
		TimeStamp:  number(ev.Get("timeStamp"), 0),
		Alt:        ev.Get("altKey").Truthy(),
		Ctrl:       ev.Get("ctrlKey").Truthy(),
		Shift:      ev.Get("shiftKey").Truthy(),
		Meta:       ev.Get("metaKey").Truthy(),
	}
}

func stringOf(v js.Value) string {
	if v.Type() == js.TypeString {
		return v.String()
	}
	return ""
}

// onKey handles keydown and keyup: the framework's answer decides whether the browser keeps
// its own action for the key (a shortcut it handled does not scroll the page or reload it).
func (p *Platform) onKey(_ js.Value, args []js.Value) any {
	ev := args[0]
	e := readKeyEvent(ev)
	if p.text.on && IsNativePasteKey(e, p.os) {
		return nil // the browser pastes into the text field, and its paste event brings the text (SPEC §23.5)
	}
	prevent := false
	if !p.callSync(func() {
		if p.in.keys != nil {
			prevent = p.in.keys.Handle(e)
		}
	}) {
		return nil
	}
	if prevent {
		ev.Call("preventDefault")
	}
	return nil
}

// dispatchKey gives key data to the framework and reports whether it handled it.
func (p *Platform) dispatchKey(d ui.KeyData) bool {
	if p.handler == nil {
		return false
	}
	if s, ok := p.handler.(platform.SyncInputHandler); ok {
		return s.OnKeySync(d)
	}
	p.handler.OnKey(d)
	return false
}

// scheduleKey runs a delayed event of the key converter on the UI goroutine.
func (p *Platform) scheduleKey(d time.Duration, fn func()) func() {
	t := time.AfterFunc(d, func() { p.q.post(fn) })
	return func() { t.Stop() }
}
