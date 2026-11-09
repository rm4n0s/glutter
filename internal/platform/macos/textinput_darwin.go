//go:build darwin && !ios

package macos

import (
	"strings"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the AppKit side of the text input (SPEC §29.7): the view is the NSTextInputClient of its
// window, so that the input methods of macOS (the ones for Chinese, Japanese and Korean, the accent menu of
// a held key, dead keys, the emoji viewer, dictation) compose text in it; the model that they read and
// write is in textinput.go.

// The NSEventModifierFlags that tell an arrow key: both are set for them.
const (
	flagNumericPad uint64 = 1 << 21
	flagFunction   uint64 = 1 << 23
)

// nsRangeVal is an NSRange as a callback returns it.
type nsRangeVal struct{ Location, Length uint64 }

func rangeOf(loc, length uint64) nsRange {
	if loc == uint64(notFound) {
		return nsRange{notFound, 0}
	}
	return nsRange{int(loc), int(length)}
}

func rangeVal(r nsRange) nsRangeVal {
	if !r.found() {
		return nsRangeVal{uint64(notFound), 0}
	}
	return nsRangeVal{uint64(r.Location), uint64(r.Length)}
}

// textMethods are the NSTextInputClient methods of GlutterView, and the commands that the input context asks
// for by selector.
//
// Flutter: FlutterTextInputPlugin.mm (NSTextInputClient)
func textMethods() []objcx.Method {
	// onWindow runs fn for the window of the view under the guard, and returns false when there is none.
	onWindow := func(self objc.ID, fn func(w *window)) bool {
		w := windowOf(self)
		if w == nil || w.closed {
			return false
		}
		w.p.guard(func() { fn(w) })
		return true
	}
	return []objcx.Method{
		{Name: "hasMarkedText", Fn: func(self objc.ID, _ objc.SEL) (has bool) {
			onWindow(self, func(w *window) { has = w.text.hasMarked() })
			return has
		}},
		{Name: "markedRange", Fn: func(self objc.ID, _ objc.SEL) (r nsRangeVal) {
			r = rangeVal(nsRange{notFound, 0})
			onWindow(self, func(w *window) { r = rangeVal(w.text.markedRange()) })
			return r
		}},
		{Name: "selectedRange", Fn: func(self objc.ID, _ objc.SEL) (r nsRangeVal) {
			r = rangeVal(nsRange{notFound, 0})
			onWindow(self, func(w *window) { r = rangeVal(w.text.selectedRange()) })
			return r
		}},
		{Name: "setMarkedText:selectedRange:replacementRange:", Fn: func(self objc.ID, _ objc.SEL, str objc.ID, selLoc, selLen, replLoc, replLen uint64) {
			onWindow(self, func(w *window) {
				if ev, ok := w.text.setMarkedText(stringOf(str), rangeOf(selLoc, selLen), rangeOf(replLoc, replLen)); ok {
					w.sendTextInput(ev)
				}
			})
		}},
		{Name: "unmarkText", Fn: func(self objc.ID, _ objc.SEL) {
			onWindow(self, func(w *window) {
				if ev, ok := w.text.unmarkText(); ok {
					w.sendTextInput(ev)
				}
			})
		}},
		{Name: "validAttributesForMarkedText", Fn: func(objc.ID, objc.SEL) objc.ID {
			return class("NSArray").Send(sel("array"))
		}},
		{Name: "attributedSubstringForProposedRange:actualRange:", Fn: func(self objc.ID, _ objc.SEL, loc, length uint64, actual uintptr) (out objc.ID) {
			onWindow(self, func(w *window) {
				s, got, ok := w.text.substring(rangeOf(loc, length))
				if !ok {
					return
				}
				if actual != 0 {
					*(*nsRangeVal)(dl.Pointer(actual)) = rangeVal(got)
				}
				ns := objcx.NewString(s)
				out = objcx.Autorelease(class("NSAttributedString").Send(sel("alloc")).Send(sel("initWithString:"), ns))
				objcx.Release(ns)
			})
			return out
		}},
		{Name: "insertText:replacementRange:", Fn: func(self objc.ID, _ objc.SEL, str objc.ID, loc, length uint64) {
			onWindow(self, func(w *window) {
				w.p.kb.produced = true
				if ev, ok := w.text.insertText(stringOf(str), rangeOf(loc, length)); ok {
					w.sendTextInput(ev)
				}
			})
		}},
		{Name: "characterIndexForPoint:", Fn: func(objc.ID, objc.SEL, float64, float64) uint64 {
			// Flutter: not implemented either; the input method does not need it for the places it asks.
			return 0
		}},
		{Name: "firstRectForCharacterRange:actualRange:", Fn: func(self objc.ID, _ objc.SEL, _, _ uint64, _ uintptr) (r nsRect) {
			onWindow(self, func(w *window) { r = w.caretScreenRect() })
			return r
		}},
		{Name: "doCommandBySelector:", Fn: func(self objc.ID, _ objc.SEL, cmd objc.SEL) {
			onWindow(self, func(w *window) { w.doCommand(selectorName(cmd)) })
		}},
		// Implementing insertTab: makes AppKit send Tab as a command and not as the text "\t".
		{Name: "insertTab:", Fn: func(objc.ID, objc.SEL, objc.ID) {}},
	}
}

// stringOf is the string of what an input method gives for text: an NSString or an NSAttributedString.
func stringOf(id objc.ID) string {
	if id == 0 {
		return ""
	}
	if send[bool](id, "isKindOfClass:", class("NSAttributedString")) {
		id = send[objc.ID](id, "string")
	}
	return nsString(id)
}

// selectorName is the name of a selector.
func selectorName(s objc.SEL) string { return objcx.SelName(s) }

// sendTextInput gives the framework an event of the input method, at once when the handler can take it so, so
// that the next call of the input method finds the field as this one left it.
func (w *window) sendTextInput(ev ui.TextInputEvent) {
	h := w.p.handler
	if h == nil {
		return
	}
	if sync, ok := h.(platform.SyncInputHandler); ok {
		sync.OnTextInputSync(ev)
		return
	}
	h.OnTextInput(ev)
}

// doCommand is a command of the input context (doCommandBySelector:): the keys that it does not type. The
// framework does it as an intent (ui.TextInputEvent.Selector); the one that Enter makes ends a composition
// and nothing else, since the framework types the line break or does the action of the field itself.
//
// Flutter: FlutterTextInputPlugin.mm doCommandBySelector:, insertNewline:
func (w *window) doCommand(name string) {
	k := w.p.kb
	k.produced = k.produced || name != "noop:"
	if !w.text.enabled {
		return // macOS may still call it when the view is no longer the first responder
	}
	if name == "insertNewline:" {
		if ev, ok := w.text.unmarkText(); ok {
			w.sendTextInput(ev)
		}
		return
	}
	w.sendTextInput(ui.TextInputEvent{View: w.text.view, Selector: name, PreeditCursorBegin: -1, PreeditCursorEnd: -1})
}

// caretScreenRect is the rectangle of the caret in screen coordinates, where the candidate window and the
// accent menu are put; it is empty when the framework has not said where the caret is.
//
// Flutter: FlutterTextInputPlugin.mm firstRectForCharacterRange:actualRange:, screenRectFromFrameworkTransform:
func (w *window) caretScreenRect() nsRect {
	c, ok := w.text.caretInView()
	if !ok {
		return nsRect{}
	}
	h := send[nsRect](w.view, "bounds").Size.Height
	// The view's origin is its bottom left, the framework's its top left.
	inView := rectOf(c.Left, h-c.Bottom, c.Right-c.Left, c.Bottom-c.Top)
	inWindow := objc.Send[nsRect](w.view, sel("convertRect:toView:"), inView, objc.ID(0))
	return objc.Send[nsRect](w.ns, sel("convertRectToScreen:"), inWindow)
}

// SetTextInput tells the input method about the focused text field (platform.Window): its text, its
// selection and where its caret is. A field that is gone, or a text that the app changed, ends the
// composition of the input context; a caret that moved moves its windows.
//
// Flutter: FlutterTextInputPlugin.mm handleMethodCall (TextInput.setClient, TextInput.setEditingState, TextInput.setCaretRect, TextInput.clearClient)
func (w *window) SetTextInput(st ui.TextInputState) {
	if w.closed {
		return
	}
	before, _ := w.text.caretInView()
	c := w.text.setState(st)
	ctx := send[objc.ID](w.view, "inputContext")
	if ctx == 0 {
		return
	}
	if c.markedLost {
		ctx.Send(sel("discardMarkedText"))
	}
	if after, _ := w.text.caretInView(); c.enabled && after != before {
		ctx.Send(sel("invalidateCharacterCoordinates"))
	}
}

// textKey gives a key event to the input context: the input method composes with it, a dead key marks text
// with it, a held key shows the accent menu, and a key that makes text inserts it. It reports whether the
// event was taken. A key equivalent that made no output (Command+Q) was not taken: it goes on to the menus.
// A key up does not go, and a modifier that was only released does not either.
//
// Flutter: FlutterTextInputPlugin.mm handleKeyEvent:
func (k *keyboard) textKey(w *window, ev objc.ID, kind int) bool {
	flags := send[uint64](ev, "modifierFlags")
	if kind == keyKindUp || (kind == keyKindFlagsChanged && flags < k.prevFlags) {
		return false
	}
	k.prevFlags = flags
	if !w.text.enabled {
		return false
	}
	ctx := send[objc.ID](w.view, "inputContext")
	if ctx == 0 {
		return false
	}
	k.produced = false
	handled := send[bool](ctx, "handleEvent:", ev)
	// Both flags are set for the arrow keys.
	navigation := flags&flagFunction != 0 && flags&flagNumericPad != 0
	if ev == k.equivalent && !(navigation && w.text.hasMarked()) && !k.produced {
		return false
	}
	return handled
}

// runDevText runs a "text", "compose" or "composing" step; it reports whether the step was one of them. The
// input method is simulated by calling the methods of the view that the real one calls, in the order it
// calls them.
func (p *Platform) runDevText(ev DevEvent) bool {
	switch ev.Kind {
	case "text", "compose", "composing":
	default:
		return false
	}
	w := p.devWindow()
	if w == nil || w.closed {
		return true
	}
	p.guard(func() {
		insert := func(s string) {
			ns := objcx.NewString(s)
			w.view.Send(sel("insertText:replacementRange:"), ns, uint64(notFound), uint64(0))
			objcx.Release(ns)
		}
		switch ev.Kind {
		case "text":
			for _, r := range ev.Text {
				if r < 0x20 || (r >= 0x7f && r < 0xa0) {
					continue
				}
				insert(string(r))
			}
		default:
			for _, piece := range ev.Steps {
				ns := objcx.NewString(piece)
				w.view.Send(sel("setMarkedText:selectedRange:replacementRange:"), ns, uint64(u16Len(piece)), uint64(0), uint64(notFound), uint64(0))
				objcx.Release(ns)
			}
			if ev.Kind == "compose" {
				insert(ev.Steps[len(ev.Steps)-1])
			}
		}
	})
	return true
}

// hasTextSteps says whether a script has steps that need a window with the keyboard focus.
func hasTextSteps(events []DevEvent) bool {
	for _, ev := range events {
		if ev.Kind == "key" || strings.HasPrefix(ev.Kind, "text") || strings.HasPrefix(ev.Kind, "compos") {
			return true
		}
	}
	return false
}
