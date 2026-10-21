//go:build js && wasm

package synthetic

import (
	"syscall/js"

	"github.com/rm4n0s/errors"
)

// ViewAttribute is the attribute of the canvas of a view (web.ViewAttribute).
const ViewAttribute = "data-glutter-view"

// Canvas finds the canvas of the first view of the page. It fails with SyntheticCanvasNotFound
// when there is no page (the program runs under Node) or the app has not put one there yet.
//
// Flutter: no counterpart
func Canvas() (js.Value, error) {
	c := js.Undefined()
	if doc := js.Global().Get("document"); doc.Type() == js.TypeObject {
		c = doc.Call("querySelector", "canvas["+ViewAttribute+"]")
	}
	if c.Type() != js.TypeObject {
		return js.Undefined(), errors.New("SyntheticCanvasNotFound", "there is no page, or it has no canvas of a Glutter view yet")
	}
	return c, nil
}

// at turns a position in CSS pixels from the top left of the canvas into the clientX and clientY of
// an event.
func at(canvas js.Value, x, y float64) (float64, float64) {
	r := canvas.Call("getBoundingClientRect")
	return r.Get("left").Float() + x, r.Get("top").Float() + y
}

// Pointer is a pointer event to dispatch.
//
// Flutter: no counterpart
type Pointer struct {
	// Type is the event type: pointerdown, pointermove, pointerup, pointercancel or pointerleave.
	Type string
	// X and Y are the position in CSS pixels from the top left of the canvas.
	X, Y float64
	// PointerType is "mouse" (the default), "touch" or "pen".
	PointerType string
	// ID is the pointerId; 1 by default.
	ID int
	// Button is the button that changed and Buttons the buttons that are down after the event.
	Button, Buttons int
	// Pressure is the pressure; 0.5 for a down or a move with buttons, else 0 when zero.
	Pressure float64
	// Alt, Ctrl, Meta and Shift are the modifier keys that are down.
	Alt, Ctrl, Meta, Shift bool
}

// DispatchPointer dispatches a PointerEvent at the canvas and reports whether its default
// action was prevented.
//
// Flutter: no counterpart
func DispatchPointer(canvas js.Value, p Pointer) (defaultPrevented bool) {
	x, y := at(canvas, p.X, p.Y)
	kind := p.PointerType
	if kind == "" {
		kind = "mouse"
	}
	id := p.ID
	if id == 0 {
		id = 1
	}
	init := map[string]any{
		"bubbles": true, "cancelable": true, "composed": true,
		"clientX": x, "clientY": y, "pointerType": kind, "pointerId": id, "isPrimary": true,
		"button": p.Button, "buttons": p.Buttons, "pressure": p.Pressure,
		"altKey": p.Alt, "ctrlKey": p.Ctrl, "metaKey": p.Meta, "shiftKey": p.Shift,
	}
	ev := js.Global().Get("PointerEvent").New(p.Type, init)
	canvas.Call("dispatchEvent", ev)
	return ev.Get("defaultPrevented").Bool()
}

// DispatchWheel dispatches a WheelEvent in pixels at the canvas and reports whether its default
// action was prevented (the page would not have scrolled).
//
// Flutter: no counterpart
func DispatchWheel(canvas js.Value, x, y, deltaX, deltaY float64) (defaultPrevented bool) {
	cx, cy := at(canvas, x, y)
	ev := js.Global().Get("WheelEvent").New("wheel", map[string]any{
		"bubbles": true, "cancelable": true, "composed": true,
		"clientX": cx, "clientY": cy, "deltaX": deltaX, "deltaY": deltaY, "deltaMode": 0,
	})
	canvas.Call("dispatchEvent", ev)
	return ev.Get("defaultPrevented").Bool()
}

// Key is a keyboard event to dispatch.
//
// Flutter: no counterpart
type Key struct {
	// Type is keydown or keyup.
	Type string
	// Code is KeyboardEvent.code and Key is KeyboardEvent.key.
	Code, Key string
	// Location is KeyboardEvent.location.
	Location int
	// Repeat is KeyboardEvent.repeat.
	Repeat bool
	// Alt, Ctrl, Meta and Shift are the modifier keys that are down.
	Alt, Ctrl, Meta, Shift bool
}

// DispatchKey dispatches a KeyboardEvent at the canvas, from where it bubbles to the host
// element, and reports whether its default action was prevented.
//
// Flutter: no counterpart
func DispatchKey(canvas js.Value, k Key) (defaultPrevented bool) {
	ev := js.Global().Get("KeyboardEvent").New(k.Type, map[string]any{
		"bubbles": true, "cancelable": true, "composed": true,
		"code": k.Code, "key": k.Key, "location": k.Location, "repeat": k.Repeat,
		"altKey": k.Alt, "ctrlKey": k.Ctrl, "metaKey": k.Meta, "shiftKey": k.Shift,
	})
	canvas.Call("dispatchEvent", ev)
	return ev.Get("defaultPrevented").Bool()
}

// DispatchContextMenu dispatches a contextmenu event (a right click) at the canvas and reports
// whether its default action, showing the browser's context menu, was prevented.
//
// Flutter: no counterpart
func DispatchContextMenu(canvas js.Value, x, y float64) (defaultPrevented bool) {
	cx, cy := at(canvas, x, y)
	ev := js.Global().Get("MouseEvent").New("contextmenu", map[string]any{
		"bubbles": true, "cancelable": true, "composed": true, "clientX": cx, "clientY": cy, "button": 2, "buttons": 2,
	})
	canvas.Call("dispatchEvent", ev)
	return ev.Get("defaultPrevented").Bool()
}
