//go:build js && wasm

package web

import (
	"strconv"
	"sync/atomic"
	"syscall/js"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// TextInputAttribute is the attribute that marks the hidden text field of a view with the view's ID.
// The browser tests and the scripted events find the field with it.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart HybridTextEditing.textEditingClass
const TextInputAttribute = "data-glutter-text-input"

// textInput is the hidden text field of the page (SPEC §23.5): an <input> or <textarea> in the host
// element, transparent, put at the caret, with the focus while a text field of the framework has it.
// The browser types, composes and pastes into it, and the plain Go TextEditor (texteditor.go) turns what
// changed into the events of the input method. This type only makes the element and copies the fields of
// its events into plain values inside callbacks that were made once, at startup (SPEC §23.3).
//
// The callbacks wait for the UI goroutine (callSync) so that an event is handled, and the framework's
// answer written back to the element, before the next one is read: that is safe because the UI goroutine
// never waits for a promise.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy
type textInput struct {
	p  *Platform
	ed TextEditor
	// el is the element while on is true.
	el  js.Value
	on  bool
	cfg ElementConfig
	// gen counts the elements that were made: an event of an element that is gone is ignored.
	gen atomic.Int64
	// box is where the element is.
	box    FieldBox
	hasBox bool
	// active are the listeners of the element and the document that are installed now.
	active []listener

	fnInput, fnBefore, fnStart, fnUpdate, fnEnd, fnSelection, fnBlur, fnPaste js.Func
}

// makeTextInputFuncs makes the callbacks of the text field once, at startup (SPEC §23.3).
func (p *Platform) makeTextInputFuncs() {
	t := &textInput{p: p}
	p.text = t
	t.fnInput = p.keep(js.FuncOf(t.onInput))
	t.fnBefore = p.keep(js.FuncOf(t.onBeforeInput))
	t.fnStart = p.keep(js.FuncOf(t.onCompositionStart))
	t.fnUpdate = p.keep(js.FuncOf(t.onCompositionUpdate))
	t.fnEnd = p.keep(js.FuncOf(t.onCompositionEnd))
	t.fnSelection = p.keep(js.FuncOf(t.onSelectionChange))
	t.fnBlur = p.keep(js.FuncOf(t.onBlur))
	t.fnPaste = p.keep(js.FuncOf(t.onPaste))
}

// readEditingState copies the value and the selection of the element.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.fromDomElement
func readEditingState(el js.Value) EditingState {
	return EditingStateFromDOM(el.Get("value").String(), int(number(el.Get("selectionStart"), 0)),
		int(number(el.Get("selectionEnd"), 0)), stringOf(el.Get("selectionDirection")))
}

// writeEditingState puts the value and the selection into the element.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.applyToDomElement
func writeEditingState(el js.Value, w DOMWrite) {
	el.Set("value", w.Text)
	dir := "forward"
	if w.Extent < w.Base {
		dir = "backward"
	}
	try(func() { el.Call("setSelectionRange", min(w.Base, w.Extent), max(w.Base, w.Extent), dir) })
}

// set takes the state of the focused text field of the framework: it makes, changes, places and fills the
// element, or takes it away when no text field has the focus.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart HybridTextEditing.acceptCommand
func (t *textInput) set(state ui.TextInputState) {
	if !state.Enabled {
		t.disable()
		return
	}
	cfg := ElementConfigOf(state, t.p.os == OSAndroid || t.p.os == OSIOS)
	if t.on && (cfg.Tag != t.cfg.Tag || cfg.Type != t.cfg.Type) {
		t.disable() // another kind of element
	}
	if !t.on {
		t.enable(cfg)
	} else if cfg != t.cfg {
		t.cfg = cfg
		applyAttributes(t.el, cfg)
	}
	t.place(state.CaretRect)
	if w := t.ed.SetState(state); w.Write {
		writeEditingState(t.el, w)
	}
	t.focus()
}

// applyAttributes sets the attributes that depend on the configuration.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.applyConfiguration
func applyAttributes(el js.Value, cfg ElementConfig) {
	el.Call("setAttribute", "autocomplete", "off")
	el.Call("setAttribute", "autocorrect", cfg.Autocorrect)
	el.Call("setAttribute", "autocapitalize", cfg.Autocapitalize)
	for name, value := range map[string]string{"inputmode": cfg.InputMode, "enterkeyhint": cfg.EnterKeyHint} {
		if value == "" {
			el.Call("removeAttribute", name)
		} else {
			el.Call("setAttribute", name, value)
		}
	}
}

// enable makes the element, puts it into the host and listens to it.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.enable
func (t *textInput) enable(cfg ElementConfig) {
	doc := global().Get("document")
	el := doc.Call("createElement", cfg.Tag)
	el.Set("tabIndex", -1) // not reachable with the Tab key
	el.Call("setAttribute", TextInputAttribute, strconv.Itoa(int(t.p.win.id)))
	if cfg.Type != "" {
		el.Set("type", cfg.Type)
	}
	// The static style of Flutter's hidden field: invisible, but a real text field to the browser and the
	// input method (the text is transparent, not hidden, which would keep the input method away).
	for _, s := range [][2]string{
		{"forced-color-adjust", "none"}, {"white-space", "pre-wrap"}, {"position", "fixed"},
		{"margin", "0"}, {"padding", "0"}, {"box-sizing", "border-box"}, {"opacity", "1"},
		{"color", "transparent"}, {"background", "transparent"}, {"caret-color", "transparent"},
		{"outline", "none"}, {"border", "none"}, {"resize", "none"}, {"text-shadow", "none"},
		{"overflow", "hidden"}, {"transform-origin", "0 0 0"}, {"line-height", "1"},
		// The text field is only for the keyboard and the input method: the pointer goes to the canvas.
		{"pointer-events", "none"}, {"z-index", "1"},
	} {
		setStyle(el, s[0], s[1])
	}
	if cfg.Tag == "textarea" {
		setStyle(el, "scrollbar-width", "none") // the width of a scroll bar changes the layout
	}
	applyAttributes(el, cfg)
	t.p.host.Call("appendChild", el)
	t.el, t.on, t.cfg, t.hasBox = el, true, cfg, false
	t.gen.Add(1)
	for _, l := range []listener{
		{target: el, event: "input", fn: t.fnInput},
		{target: el, event: "beforeinput", fn: t.fnBefore},
		{target: el, event: "compositionstart", fn: t.fnStart},
		{target: el, event: "compositionupdate", fn: t.fnUpdate},
		{target: el, event: "compositionend", fn: t.fnEnd},
		{target: el, event: "blur", fn: t.fnBlur},
		{target: el, event: "paste", fn: t.fnPaste, options: map[string]any{"passive": false}},
		{target: doc, event: "selectionchange", fn: t.fnSelection},
	} {
		l.target.Call("addEventListener", l.event, l.fn)
		t.active = append(t.active, l)
	}
}

// disable takes the element away and gives the focus back to the host.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.disable
func (t *textInput) disable() {
	if !t.on {
		return
	}
	hadFocus := t.el.Equal(global().Get("document").Get("activeElement"))
	for _, l := range t.active {
		l.target.Call("removeEventListener", l.event, l.fn)
	}
	t.active = nil
	t.el.Call("remove")
	t.on, t.el = false, js.Undefined()
	t.gen.Add(1)
	t.ed.Reset()
	if hadFocus {
		try(func() { t.p.host.Call("focus", map[string]any{"preventScroll": true}) })
	}
}

// place puts the element at the caret, in CSS pixels of the page.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditableTextGeometry.applyToDomElement
func (t *textInput) place(caret ui.Rect) {
	rect := t.p.win.canvas.Call("getBoundingClientRect")
	box := FieldBoxOf(caret, number(rect.Get("left"), 0), number(rect.Get("top"), 0))
	if t.hasBox && box == t.box {
		return
	}
	t.box, t.hasBox = box, true
	px := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) + "px" }
	setStyle(t.el, "left", px(box.Left))
	setStyle(t.el, "top", px(box.Top))
	setStyle(t.el, "width", px(box.Width))
	setStyle(t.el, "height", px(box.Height))
	setStyle(t.el, "font-size", px(box.FontSize))
}

// focus gives the element the focus, when it does not have it.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.moveFocusToActiveDomElement
func (t *textInput) focus() {
	if !t.on || t.el.Equal(global().Get("document").Get("activeElement")) {
		return
	}
	try(func() { t.el.Call("focus", map[string]any{"preventScroll": true}) })
}

// send gives the framework an event of the input method, at once when the engine can.
func (t *textInput) send(ev ui.TextInputEvent) {
	h := t.p.handler
	if h == nil || t.p.win == nil {
		return
	}
	ev.View = t.p.win.id
	if s, ok := h.(platform.SyncInputHandler); ok {
		s.OnTextInputSync(ev)
		return
	}
	h.OnTextInput(ev)
}

// handle turns the state of the element after an event into an event of the input method, if it changed
// the text. It runs on the UI goroutine.
func (t *textInput) handle(gen int64, st EditingState) {
	if !t.on || gen != t.gen.Load() {
		return
	}
	if ev, ok := t.ed.Handle(st); ok {
		t.send(ev)
	}
}

// changed reads the element now and queues what it holds.
func (t *textInput) changed() {
	if !t.on {
		return
	}
	st, gen := readEditingState(t.el), t.gen.Load()
	t.p.callSync(func() { t.handle(gen, st) })
}

// onInput handles input: the text of the element changed.
func (t *textInput) onInput(js.Value, []js.Value) any {
	t.changed()
	return nil
}

// onSelectionChange handles selectionchange of the document, for the element that has the focus. The
// selection alone is not sent, but the cursor of a composition is.
func (t *textInput) onSelectionChange(js.Value, []js.Value) any {
	if t.on && t.el.Equal(global().Get("document").Get("activeElement")) {
		t.changed()
	}
	return nil
}

// onBeforeInput handles beforeinput.
func (t *textInput) onBeforeInput(_ js.Value, args []js.Value) any {
	if !t.on {
		return nil
	}
	inputType, data := stringOf(args[0].Get("inputType")), stringOf(args[0].Get("data"))
	gen := t.gen.Load()
	t.p.callSync(func() {
		if t.on && gen == t.gen.Load() {
			t.ed.BeforeInput(inputType, data)
		}
	})
	return nil
}

func (t *textInput) onCompositionStart(js.Value, []js.Value) any {
	if !t.on {
		return nil
	}
	gen := t.gen.Load()
	t.p.callSync(func() {
		if t.on && gen == t.gen.Load() {
			t.ed.CompositionStart()
		}
	})
	return nil
}

func (t *textInput) onCompositionUpdate(_ js.Value, args []js.Value) any {
	if !t.on {
		return nil
	}
	data, gen := stringOf(args[0].Get("data")), t.gen.Load()
	t.p.callSync(func() {
		if t.on && gen == t.gen.Load() {
			t.ed.CompositionUpdate(data)
		}
	})
	return nil
}

// onCompositionEnd handles compositionend: the composition is over, and the text it made is committed.
func (t *textInput) onCompositionEnd(js.Value, []js.Value) any {
	if !t.on {
		return nil
	}
	st, gen := readEditingState(t.el), t.gen.Load()
	t.p.callSync(func() {
		if t.on && gen == t.gen.Load() {
			t.ed.CompositionEnd()
			t.handle(gen, st)
		}
	})
	return nil
}

// onPaste handles paste, which the browser sends when the user pastes with a shortcut or a menu: the
// text is taken from the event, without a permission, and the browser does not insert it itself.
func (t *textInput) onPaste(_ js.Value, args []js.Value) any {
	if !t.on {
		return nil
	}
	ev := args[0]
	text := ""
	if data := ev.Get("clipboardData"); isObject(data) {
		text = stringOf(data.Call("getData", "text/plain"))
	}
	if text == "" {
		return nil
	}
	ev.Call("preventDefault")
	gen := t.gen.Load()
	t.p.callSync(func() {
		if !t.on || gen != t.gen.Load() {
			return
		}
		if e, ok := t.ed.Paste(text); ok {
			t.send(e)
		}
	})
	return nil
}

// onBlur handles blur: when the focus goes to another element of the same host (the canvas or the host
// itself, after a click), the field takes it back, so that the keyboard stays with the text field. When it
// leaves the page, or the window loses the focus, it stays away, and the page's focus event brings it back.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.handleBlur
func (t *textInput) onBlur(_ js.Value, args []js.Value) any {
	if !t.on {
		return nil
	}
	to := args[0].Get("relatedTarget")
	if isObject(to) && !to.Equal(t.el) && t.p.host.Call("contains", to).Bool() {
		gen := t.gen.Load()
		t.p.q.post(func() {
			if t.on && gen == t.gen.Load() {
				t.focus()
			}
		})
	}
	return nil
}
