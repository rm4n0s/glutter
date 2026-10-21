//go:build js && wasm

package synthetic

import (
	"syscall/js"
	"unicode/utf16"

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

// TextInputAttribute is the attribute of the hidden text field of a view (web.TextInputAttribute).
//
// Flutter: no counterpart
const TextInputAttribute = "data-glutter-text-input"

// TextField finds the hidden text field of the first view of the page. It fails with
// SyntheticTextFieldNotFound when there is no page or no text field of the framework has the focus.
//
// Flutter: no counterpart
func TextField() (js.Value, error) {
	f := js.Undefined()
	if doc := js.Global().Get("document"); doc.Type() == js.TypeObject {
		f = doc.Call("querySelector", "["+TextInputAttribute+"]")
	}
	if f.Type() != js.TypeObject {
		return js.Undefined(), errors.New("SyntheticTextFieldNotFound", "there is no page, or no text field of the framework has the focus")
	}
	return f, nil
}

// inputEvent dispatches an InputEvent at the text field and reports whether its default action was
// prevented. A synthetic event has no default action of its own: the callers change the field's text
// themselves, between the beforeinput and the input event, as the browser does.
func inputEvent(field js.Value, typ, inputType string, data any, composing bool) bool {
	ev := js.Global().Get("InputEvent").New(typ, map[string]any{
		"bubbles": true, "cancelable": typ == "beforeinput" && !composing, "composed": true,
		"inputType": inputType, "data": data, "isComposing": composing,
	})
	field.Call("dispatchEvent", ev)
	return ev.Get("defaultPrevented").Bool()
}

// replace puts text in place of the characters [from, to) of the field and puts the caret after it.
func replace(field js.Value, text string, from, to int) {
	field.Call("setRangeText", text, from, to, "end")
}

// TypeText types text into the text field as a browser does for a key that makes a character: for each
// character a beforeinput event (insertText), the change of the field's text at its selection, and an
// input event; a line break (for a textarea) is an insertLineBreak. A beforeinput event whose default
// action is prevented changes nothing, as in a browser.
//
// Flutter: no counterpart
func TypeText(field js.Value, text string) {
	for _, r := range text {
		s := string(r)
		kind, data := "insertText", any(s)
		if r == '\n' {
			kind, data = "insertLineBreak", nil
		}
		if inputEvent(field, "beforeinput", kind, data, false) {
			continue
		}
		replace(field, s, field.Get("selectionStart").Int(), field.Get("selectionEnd").Int())
		inputEvent(field, "input", kind, data, false)
	}
}

// DeleteBackward deletes the character before the caret, or the selection, as a soft keyboard's
// backspace does: a beforeinput, the change and an input event of the type deleteContentBackward.
//
// Flutter: no counterpart
func DeleteBackward(field js.Value) {
	if inputEvent(field, "beforeinput", "deleteContentBackward", nil, false) {
		return
	}
	from, to := field.Get("selectionStart").Int(), field.Get("selectionEnd").Int()
	if from == to && from > 0 {
		from--
	}
	replace(field, "", from, to)
	inputEvent(field, "input", "deleteContentBackward", nil, false)
}

// Composer makes the events of an input method for one composition: Start, any number of Update, and End.
// The text of each update replaces the previous one in the field.
//
// Flutter: no counterpart
type Composer struct {
	field    js.Value
	from, to int
}

// StartComposition dispatches compositionstart at the text field; the composition begins at its
// selection, which the composed text replaces.
//
// Flutter: no counterpart
func StartComposition(field js.Value) *Composer {
	c := &Composer{field: field}
	c.event("compositionstart", "")
	c.from, c.to = field.Get("selectionStart").Int(), field.Get("selectionEnd").Int()
	return c
}

func (c *Composer) event(typ, data string) {
	c.field.Call("dispatchEvent", js.Global().Get("CompositionEvent").New(typ, map[string]any{
		"bubbles": true, "composed": true, "data": data,
	}))
}

// Update dispatches a compositionupdate with the text so far, then the beforeinput and the input event
// (insertCompositionText) that follow it, with the field's text changed in between.
//
// Flutter: no counterpart
func (c *Composer) Update(text string) {
	c.event("compositionupdate", text)
	inputEvent(c.field, "beforeinput", "insertCompositionText", text, true)
	replace(c.field, text, c.from, c.to)
	c.to = c.from + len(utf16.Encode([]rune(text)))
	inputEvent(c.field, "input", "insertCompositionText", text, true)
}

// End dispatches compositionend with the committed text.
//
// Flutter: no counterpart
func (c *Composer) End(text string) { c.event("compositionend", text) }

// Compose makes the events of an input method that composes each of the steps in turn and commits the
// last one: compositionstart, an update for each step, and compositionend.
//
// Flutter: no counterpart
func Compose(field js.Value, steps ...string) {
	if len(steps) == 0 {
		return
	}
	c := StartComposition(field)
	for _, s := range steps {
		c.Update(s)
	}
	c.End(steps[len(steps)-1])
}

// Paste dispatches a paste event with text as the plain text of the clipboard at the text field, and
// reports whether its default action was prevented (the web platform takes the text itself).
//
// Flutter: no counterpart
func Paste(field js.Value, text string) (defaultPrevented bool) {
	data := js.Global().Get("DataTransfer").New()
	data.Call("setData", "text/plain", text)
	ev := js.Global().Get("ClipboardEvent").New("paste", map[string]any{
		"bubbles": true, "cancelable": true, "composed": true, "clipboardData": data,
	})
	// Firefox gives a constructed event a clipboardData of its own that holds nothing, whatever the
	// init says: the data becomes an own property of the event, which wins over the one of the prototype.
	js.Global().Get("Object").Call("defineProperty", ev, "clipboardData", map[string]any{"value": data})
	field.Call("dispatchEvent", ev)
	return ev.Get("defaultPrevented").Bool()
}
