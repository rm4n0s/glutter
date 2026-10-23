//go:build js && wasm

package main

import (
	"encoding/json"
	"log"
	"syscall/js"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/web"
	"github.com/rm4n0s/glutter/internal/platform/web/synthetic"
	m "github.com/rm4n0s/glutter/material"
)

// Observations is what the program posts as the result "inputs".
type observations struct {
	// TabIndex is the tabindex attribute of the host element.
	TabIndex string `json:"tabIndex"`
	// Downs are the pointer-down events the framework saw, as "x,y".
	Downs []string `json:"downs"`
	// Count is the number the button counted after three taps.
	Count int `json:"count"`
	// Cursor is the CSS cursor of the canvas with the mouse over the button, and CursorAway with it
	// over the list.
	Cursor     string `json:"cursor"`
	CursorAway string `json:"cursorAway"`
	// KeyHandledPrevented says whether the default action of the K key, which the Shortcuts widget
	// handles, was prevented; KeyUnhandledPrevented the same for the Q key, which nothing handles.
	KeyHandledPrevented   bool `json:"keyHandledPrevented"`
	KeyUnhandledPrevented bool `json:"keyUnhandledPrevented"`
	// KeyCount is how many times the Shortcuts widget invoked its action.
	KeyCount int `json:"keyCount"`
	// WheelList is the offset of the list after a wheel event of 300 pixels over it, and
	// WheelListPrevented whether the default action of that event was prevented.
	WheelList          float64 `json:"wheelList"`
	WheelListPrevented bool    `json:"wheelListPrevented"`
	// WheelBand says whether the default action of a wheel event over the band above the list,
	// which does not scroll, was prevented.
	WheelBandPrevented bool `json:"wheelBandPrevented"`
	// WheelBack is the offset after the list was scrolled back to the top, and WheelBoundary whether the default
	// action of a wheel event upward at the top of the list was prevented.
	WheelBack              float64 `json:"wheelBack"`
	WheelBoundaryPrevented bool    `json:"wheelBoundaryPrevented"`
	// The same K key and a wheel event of 100 pixels over the list, dispatched by the browser's timer
	// instead of from a goroutine of the program: the listener then runs on a goroutine of its own
	// as it does for the user's events, and waits for the UI goroutine. They tell whether the default
	// of each was prevented, how many times the action has been invoked and the offset of the list.
	TimerKeyPrevented   bool    `json:"timerKeyPrevented"`
	TimerWheelPrevented bool    `json:"timerWheelPrevented"`
	TimerKeyCount       int     `json:"timerKeyCount"`
	TimerOffset         float64 `json:"timerOffset"`
	// Menu is whether the contextmenu event was prevented while the browser's context menu was
	// disabled by the app, and MenuEnabled when it was enabled again.
	MenuDisabledPrevented bool `json:"menuDisabledPrevented"`
	MenuEnabledPrevented  bool `json:"menuEnabledPrevented"`
	MenuEnabledFlag       bool `json:"menuEnabledFlag"`
	MenuDisabledFlag      bool `json:"menuDisabledFlag"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	p := newProbe()
	go func() {
		report := observe(p)
		postResult("inputs", report)
	}()
	return m.RunApp(newInputApp(p), m.RunAppOpts{Window: m.WindowConfig{Title: "Input test"}})
}

// observe waits for the app and makes the events. It runs on a goroutine of its own: the
// listeners of the platform that wait for the UI goroutine would wait for themselves otherwise.
func observe(p *probe) observations {
	var o observations
	<-p.ready
	time.Sleep(time.Second) // the first frames
	canvas, err := synthetic.Canvas()
	if err != nil {
		panic(err)
	}
	settle := func() { time.Sleep(200 * time.Millisecond) }
	host := js.Global().Get("document").Get("body")
	o.TabIndex = host.Call("getAttribute", "tabindex").String()

	// Three taps on the button.
	for range 3 {
		synthetic.DispatchPointer(canvas, synthetic.Pointer{Type: "pointerdown", X: buttonX, Y: buttonY, Buttons: 1, Pressure: 0.5})
		time.Sleep(50 * time.Millisecond)
		synthetic.DispatchPointer(canvas, synthetic.Pointer{Type: "pointerup", X: buttonX, Y: buttonY})
		settle()
	}
	p.onUI(func() { o.Count, o.Downs = p.count, p.downs })

	// The cursor follows the mouse: the button asks for the pointing hand.
	cursor := func() string { return js.Global().Call("getComputedStyle", canvas).Get("cursor").String() }
	synthetic.DispatchPointer(canvas, synthetic.Pointer{Type: "pointermove", X: buttonX, Y: buttonY})
	settle()
	o.Cursor = cursor()
	synthetic.DispatchPointer(canvas, synthetic.Pointer{Type: "pointermove", X: listX, Y: listY})
	settle()
	o.CursorAway = cursor()

	// A key that the Shortcuts widget handles, and one that nothing handles.
	o.KeyHandledPrevented = synthetic.DispatchKey(canvas, synthetic.Key{Type: "keydown", Code: "KeyK", Key: "k"})
	synthetic.DispatchKey(canvas, synthetic.Key{Type: "keyup", Code: "KeyK", Key: "k"})
	o.KeyUnhandledPrevented = synthetic.DispatchKey(canvas, synthetic.Key{Type: "keydown", Code: "KeyQ", Key: "q"})
	synthetic.DispatchKey(canvas, synthetic.Key{Type: "keyup", Code: "KeyQ", Key: "q"})
	settle()
	p.onUI(func() { o.KeyCount = p.keyCount })

	// The wheel scrolls the list, and the page does not scroll.
	o.WheelListPrevented = synthetic.DispatchWheel(canvas, listX, listY, 0, 300)
	settle()
	p.onUI(func() { o.WheelList = p.offset })
	o.WheelBandPrevented = synthetic.DispatchWheel(canvas, 240, 20, 0, 300)
	settle()
	synthetic.DispatchWheel(canvas, listX, listY, 0, -1000)
	settle()
	p.onUI(func() { o.WheelBack = p.offset })
	o.WheelBoundaryPrevented = synthetic.DispatchWheel(canvas, listX, listY, 0, -50)
	settle()

	// The same, from the browser's timer.
	fromTimer := js.Global().Get("Function").New("canvas", "x", "y", `
		setTimeout(function () {
			var init = {bubbles: true, cancelable: true, composed: true, code: "KeyK", key: "k"};
			var down = new KeyboardEvent("keydown", init);
			canvas.dispatchEvent(down);
			canvas.dispatchEvent(new KeyboardEvent("keyup", init));
			var r = canvas.getBoundingClientRect();
			var wheel = new WheelEvent("wheel", {bubbles: true, cancelable: true, composed: true,
				clientX: r.left + x, clientY: r.top + y, deltaY: 100, deltaMode: 0});
			canvas.dispatchEvent(wheel);
			globalThis.timerResult = {key: down.defaultPrevented, wheel: wheel.defaultPrevented};
		}, 0);`)
	fromTimer.Invoke(canvas, listX, listY)
	time.Sleep(time.Second)
	if r := js.Global().Get("timerResult"); r.Type() == js.TypeObject {
		o.TimerKeyPrevented, o.TimerWheelPrevented = r.Get("key").Bool(), r.Get("wheel").Bool()
	}
	p.onUI(func() { o.TimerKeyCount, o.TimerOffset = p.keyCount, p.offset })

	// The browser's context menu.
	p.onUI(func() {
		o.MenuEnabledFlag = p.binding.BrowserContextMenu().Enabled()
		p.binding.BrowserContextMenu().DisableContextMenu()
		o.MenuDisabledFlag = !p.binding.BrowserContextMenu().Enabled()
	})
	o.MenuDisabledPrevented = synthetic.DispatchContextMenu(canvas, listX, listY)
	p.onUI(func() { p.binding.BrowserContextMenu().EnableContextMenu() })
	o.MenuEnabledPrevented = synthetic.DispatchContextMenu(canvas, listX, listY)
	return o
}

// postResult posts a result to the test server. The answer is not waited for: the callback of the
// promise does not matter, and the program goes on.
func postResult(kind string, data any) {
	body, err := json.Marshal(map[string]any{"kind": kind, "data": data})
	if err != nil {
		panic(err)
	}
	init := map[string]any{"method": "POST", "headers": map[string]any{"Content-Type": "application/json"}, "body": string(body)}
	js.Global().Call("fetch", web.ResultsPath, init)
}
