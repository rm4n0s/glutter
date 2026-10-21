//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"syscall/js"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/web"
	"github.com/rm4n0s/glutter/internal/platform/web/synthetic"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

// field describes the hidden text field of the platform at a moment.
type field struct {
	Present bool `json:"present"`
	// Tag, Type, InputMode, EnterKeyHint and Autocapitalize are the element and its attributes.
	Tag            string `json:"tag"`
	Type           string `json:"type"`
	InputMode      string `json:"inputMode"`
	EnterKeyHint   string `json:"enterKeyHint"`
	Autocapitalize string `json:"autocapitalize"`
	// Focused says the element is the active element of the page.
	Focused bool `json:"focused"`
	// Value is the text of the element.
	Value string `json:"value"`
}

// state is what the framework holds in a text field.
type state struct {
	Text string `json:"text"`
	// ComposingStart and ComposingEnd are the composing range in bytes; ComposingValid says there is one.
	ComposingValid bool `json:"composingValid"`
	ComposingStart int  `json:"composingStart"`
	ComposingEnd   int  `json:"composingEnd"`
	Selection      int  `json:"selection"`
}

// observations is what the program posts as the result "text".
type observations struct {
	// AfterTap is the hidden field after a tap on the single-line text field.
	AfterTap field `json:"afterTap"`
	// Typed is the framework's text and the element's after synthetic beforeinput and input events typed
	// "hello", then after a deleteContentBackward.
	Typed        state `json:"typed"`
	TypedElement field `json:"typedElement"`
	Backspaced   state `json:"backspaced"`
	// MidComposition is the framework's state while an input method composes "ni", Composed after it
	// committed 你好.
	MidComposition        state `json:"midComposition"`
	MidCompositionElement field `json:"midCompositionElement"`
	Composed              state `json:"composed"`
	ComposedElement       field `json:"composedElement"`
	// PastePrevented says the paste event's default was prevented, and Pasted is the state after it.
	PastePrevented bool  `json:"pastePrevented"`
	Pasted         state `json:"pasted"`
	PastedElement  field `json:"pastedElement"`
	// KeyTypedPrevented says the keydown of the A key (a character that nothing handles) was prevented,
	// and KeyTyped is the state after it: the framework typed the character once.
	KeyTypedPrevented bool  `json:"keyTypedPrevented"`
	KeyTyped          state `json:"keyTyped"`
	// PasteKeyPrevented says the keydown of Control+V was prevented, which it must not be: the browser
	// pastes natively. PasteKeyState is the state after it, which the framework did not change.
	PasteKeyPrevented bool  `json:"pasteKeyPrevented"`
	PasteKeyState     state `json:"pasteKeyState"`
	// PendingKeysPrevented says, for each key sent while a clipboard read was pending, whether it was
	// handled (typed); PendingAnswered whether the read had been answered by then; PendingState the text
	// after the keys.
	PendingKeysPrevented []bool `json:"pendingKeysPrevented"`
	PendingAnswered      bool   `json:"pendingAnswered"`
	PendingText          string `json:"pendingText"`
	// ReadText and ReadError are the answer of the read once the stub's promise was resolved;
	// RejectedTag is the tag of the error when it was rejected instead.
	ReadText    string `json:"readText"`
	ReadError   string `json:"readError"`
	RejectedTag string `json:"rejectedTag"`
	// Written is the text the stub's writeText got from Clipboard.SetData, WriteError the answer, and
	// Copied the text it got from the Control+C shortcut.
	Written    string `json:"written"`
	WriteError string `json:"writeError"`
	Copied     string `json:"copied"`
	// Secret, SecretElement describe the password field after it got the focus and "s3cret" was typed.
	Secret        state `json:"secret"`
	SecretElement field `json:"secretElement"`
	// Multi is the multiline field after an Enter key at the hidden field; EnterPrevented says the
	// keydown was prevented.
	MultiElement   field `json:"multiElement"`
	Multi          state `json:"multi"`
	EnterPrevented bool  `json:"enterPrevented"`
	// Gone is the hidden field after the text field lost the focus, and HostFocused whether the focus
	// went back to the host.
	Gone        field `json:"gone"`
	HostFocused bool  `json:"hostFocused"`
}

func run() error {
	p := newProbe()
	go func() {
		report := observe(p)
		postResult("text", report)
	}()
	return m.RunApp(newTextApp(p), m.RunAppOpts{Window: m.WindowConfig{Title: "Text test"}})
}

// clipboardStub replaces navigator.clipboard by an object whose readText returns a promise that stays
// pending until the test settles it, and whose writeText records the text.
const clipboardStub = `
	var stub = {reads: [], writes: []};
	Object.defineProperty(navigator, "clipboard", {configurable: true, value: {
		readText: function () { return new Promise(function (resolve, reject) { stub.reads.push({resolve: resolve, reject: reject}); }); },
		writeText: function (text) { stub.writes.push(text); return Promise.resolve(); }
	}});
	globalThis.clipboardStub = stub;`

// observe waits for the app and makes the events. It runs on a goroutine of its own: the listeners of the
// platform that wait for the UI goroutine would wait for themselves otherwise.
func observe(p *probe) observations {
	var o observations
	<-p.ready
	time.Sleep(time.Second) // the first frames
	canvas, err := synthetic.Canvas()
	if err != nil {
		panic(err)
	}
	settle := func() { time.Sleep(200 * time.Millisecond) }
	doc := js.Global().Get("document")

	describe := func() field {
		el := doc.Call("querySelector", "["+synthetic.TextInputAttribute+"]")
		if el.Type() != js.TypeObject {
			return field{}
		}
		attr := func(name string) string {
			if v := el.Call("getAttribute", name); v.Type() == js.TypeString {
				return v.String()
			}
			return ""
		}
		return field{
			Present: true, Tag: el.Get("tagName").String(), Type: attr("type"), InputMode: attr("inputmode"),
			EnterKeyHint: attr("enterkeyhint"), Autocapitalize: attr("autocapitalize"),
			Focused: el.Equal(doc.Get("activeElement")), Value: el.Get("value").String(),
		}
	}
	read := func(c *m.TextEditingController) state {
		var s state
		p.onUI(func() {
			v := c.Value()
			s.Text, s.Selection = v.Text, v.Selection.ExtentOffset
			if v.IsComposingRangeValid() && !v.Composing.IsCollapsed() {
				s.ComposingValid, s.ComposingStart, s.ComposingEnd = true, v.Composing.Start, v.Composing.End
			}
		})
		return s
	}
	key := func(el js.Value, typ, code, k string, ctrl bool) bool {
		return synthetic.DispatchKey(el, synthetic.Key{Type: typ, Code: code, Key: k, Ctrl: ctrl})
	}

	// A tap on the single-line text field gives it the focus, and the platform puts its hidden field at the caret.
	synthetic.DispatchPointer(canvas, synthetic.Pointer{Type: "pointerdown", X: singleX, Y: singleY, Buttons: 1, Pressure: 0.5})
	time.Sleep(50 * time.Millisecond)
	synthetic.DispatchPointer(canvas, synthetic.Pointer{Type: "pointerup", X: singleX, Y: singleY})
	settle()
	o.AfterTap = describe()
	input, err := synthetic.TextField()
	if err != nil {
		return o
	}

	// beforeinput and input events type into the field; the framework answers with the new state.
	synthetic.TypeText(input, "hello")
	o.Typed, o.TypedElement = read(p.single), describe()
	synthetic.DeleteBackward(input)
	o.Backspaced = read(p.single)

	// An input method composes "ni" (shown, not committed) and then commits 你好.
	c := synthetic.StartComposition(input)
	c.Update("n")
	c.Update("ni")
	o.MidComposition, o.MidCompositionElement = read(p.single), describe()
	c.Update("你")
	c.Update("你好")
	c.End("你好")
	o.Composed, o.ComposedElement = read(p.single), describe()
	// The paste event brings the text without a permission, and the browser does not insert it itself.
	o.PastePrevented = synthetic.Paste(input, " world")
	o.Pasted, o.PastedElement = read(p.single), describe()

	// A character that nothing handles is typed by the framework, and the browser must not type it again.
	o.KeyTypedPrevented = key(input, "keydown", "KeyA", "a", false)
	key(input, "keyup", "KeyA", "a", false)
	o.KeyTyped = read(p.single)
	// Control+V is the browser's paste shortcut: the key is not given to the framework and not prevented.
	o.PasteKeyPrevented = key(input, "keydown", "KeyV", "v", true)
	key(input, "keyup", "KeyV", "v", true)
	o.PasteKeyState = read(p.single)

	// The clipboard: a read that stays pending while keys go to the framework.
	js.Global().Get("Function").New(clipboardStub).Invoke()
	stub := js.Global().Get("clipboardStub")
	var readText, readErr string
	answered := false
	getData := func(done func(text string, err error)) {
		p.onUI(func() {
			p.binding.Clipboard().GetData(context.Background(), services.ClipboardKTextPlain, func(d *services.ClipboardData, err error) {
				text := ""
				if d != nil {
					text = d.Text
				}
				done(text, err)
			})
		})
	}
	getData(func(text string, err error) {
		answered, readText = true, text
		if err != nil {
			readErr = tagOf(err)
		}
	})
	settle()
	for _, k := range []struct{ code, key string }{{"KeyX", "x"}, {"KeyY", "y"}, {"KeyZ", "z"}} {
		o.PendingKeysPrevented = append(o.PendingKeysPrevented, key(input, "keydown", k.code, k.key, false))
		key(input, "keyup", k.code, k.key, false)
	}
	var pendingAnswered bool
	p.onUI(func() { pendingAnswered = answered })
	o.PendingAnswered, o.PendingText = pendingAnswered, read(p.single).Text
	stub.Get("reads").Index(0).Call("resolve", "from clipboard")
	time.Sleep(300 * time.Millisecond)
	p.onUI(func() { o.ReadText, o.ReadError = readText, readErr })

	// A refused read is a WebClipboardReadFailed.
	var rejected string
	getData(func(_ string, err error) { rejected = tagOf(err) })
	stub.Get("reads").Index(1).Call("reject", js.Global().Get("Error").New("denied"))
	time.Sleep(300 * time.Millisecond)
	p.onUI(func() { o.RejectedTag = rejected })

	// SetData goes to writeText.
	var writeErr string
	p.onUI(func() {
		p.binding.Clipboard().SetData(context.Background(), services.ClipboardData{Text: "to clipboard"}, func(err error) {
			if err != nil {
				writeErr = tagOf(err)
			}
		})
	})
	time.Sleep(300 * time.Millisecond)
	if w := stub.Get("writes"); w.Length() > 0 {
		o.Written = w.Index(0).String()
	}
	p.onUI(func() { o.WriteError = writeErr })

	// Control+A and Control+C: the framework copies the selection with writeText, inside the key event.
	key(input, "keydown", "ControlLeft", "Control", true)
	key(input, "keydown", "KeyA", "a", true)
	key(input, "keyup", "KeyA", "a", true)
	key(input, "keydown", "KeyC", "c", true)
	key(input, "keyup", "KeyC", "c", true)
	key(input, "keyup", "ControlLeft", "Control", false)
	time.Sleep(300 * time.Millisecond)
	if w := stub.Get("writes"); w.Length() > 1 {
		o.Copied = w.Index(w.Length() - 1).String()
	}

	// The password field gets a hidden field of type password.
	p.onUI(func() { p.secretFocus.RequestFocus() })
	settle()
	if input, err = synthetic.TextField(); err != nil {
		return o
	}
	synthetic.TypeText(input, "s3cret")
	o.Secret, o.SecretElement = read(p.secret), describe()

	// The multiline field gets a textarea, and Enter in it is a new line, which the browser must not insert too.
	p.onUI(func() { p.multiFocus.RequestFocus() })
	settle()
	if input, err = synthetic.TextField(); err != nil {
		return o
	}
	synthetic.TypeText(input, "one")
	o.EnterPrevented = key(input, "keydown", "Enter", "Enter", false)
	key(input, "keyup", "Enter", "Enter", false)
	synthetic.TypeText(input, "two")
	o.Multi, o.MultiElement = read(p.multi), describe()

	// Without a text field with the focus, the hidden field is gone and the host has the focus again.
	p.onUI(func() { p.multiFocus.Unfocus() })
	settle()
	o.Gone = describe()
	o.HostFocused = doc.Get("activeElement").Equal(doc.Get("body"))
	return o
}

// tagOf is the tag of an error of the framework.
func tagOf(err error) string {
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
