package macos

import (
	"github.com/rm4n0s/glutter/internal/platform/textedit"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the text input of the macOS backend that has no AppKit call in it (SPEC §29.7): the document
// that the input method of macOS reads and writes through NSTextInputClient, kept as the model of
// internal/platform/textedit keeps the text field of the web and of Android, and the events that the
// framework's field gets from it. The AppKit side (the methods of the view, the input context) is in
// textinput_darwin.go.
//
// Flutter: engine/src/flutter/shell/platform/darwin/macos/framework/Source/FlutterTextInputPlugin.mm
// FlutterTextInputPlugin (its text model is shell/platform/common/text_input_model.cc)

// notFound is NSNotFound, the location of a range that is not there.
const notFound = int(^uint(0) >> 1)

// nsRange is an NSRange: a location and a length in UTF-16 units of the document.
type nsRange struct{ Location, Length int }

// found says that the range is one, not NSNotFound.
func (r nsRange) found() bool { return r.Location != notFound }

// textInput is the text input of the platform: the document of the focused text field as the input method
// sees it (the text with the text being composed in it), the place of the caret, and the view that the
// field is in. It belongs to the UI thread.
type textInput struct {
	ed      textedit.TextEditor
	enabled bool
	view    ui.ViewID
	// caret is the rectangle of the caret in logical pixels of the view, from the top left.
	caret ui.Rect
}

// stateChange tells what a new state of the field did, for the AppKit side.
type stateChange struct {
	// enabled and wasEnabled are whether a field has the focus after and before.
	enabled, wasEnabled bool
	// markedLost says that the text that the input method was composing is not in the document any more
	// (the field lost the focus or its text was set by the app): the input context must forget it.
	markedLost bool
}

// setState takes the state of the focused text field (platform.Window.SetTextInput). The state is the text
// without the text being composed; the input method's own change of the text, which the framework sent
// back (CausedByInputMethod), is what the model already holds, and the composition stays. A change that
// the app made ends the composition.
//
// Flutter: FlutterTextInputPlugin.mm handleMethodCall (TextInput.setClient, TextInput.setEditingState, TextInput.clearClient)
func (t *textInput) setState(st ui.TextInputState) stateChange {
	c := stateChange{wasEnabled: t.enabled}
	wasMarked := t.enabled && t.hasMarked()
	if !st.Enabled {
		t.ed.Reset()
		t.enabled, t.caret = false, ui.Rect{}
		c.markedLost = wasMarked
		return c
	}
	t.enabled, t.view, t.caret = true, st.View, st.CaretRect
	w := t.ed.SetState(st)
	c.enabled = true
	c.markedLost = wasMarked && w.Write
	return c
}

// document is the text that the input method reads: the field's text with the text being composed in it.
func (t *textInput) document() textedit.EditingState { return t.ed.DOM() }

// hasMarked says that text is being composed.
//
// Flutter: FlutterTextInputPlugin.mm hasMarkedText
func (t *textInput) hasMarked() bool {
	_, _, ok := t.document().Composing()
	return t.enabled && ok
}

// markedRange is the range of the text being composed in the document, or NSNotFound.
//
// Flutter: FlutterTextInputPlugin.mm markedRange
func (t *textInput) markedRange() nsRange {
	if !t.enabled {
		return nsRange{notFound, 0}
	}
	if b, e, ok := t.document().Composing(); ok {
		return nsRange{b, e - b}
	}
	return nsRange{notFound, 0}
}

// selectedRange is the selection in the document (the caret has a length of 0). While text is being
// composed it is inside the composing text, where the input method put it.
func (t *textInput) selectedRange() nsRange {
	if !t.enabled {
		return nsRange{notFound, 0}
	}
	d := t.document()
	return nsRange{d.MinOffset(), d.MaxOffset() - d.MinOffset()}
}

// substring is the text of the document in the range, which is cut to the document; ok is false when the
// range begins after the end. actual is the range that was cut.
//
// Flutter: FlutterTextInputPlugin.mm attributedSubstringForProposedRange:actualRange:
func (t *textInput) substring(r nsRange) (s string, actual nsRange, ok bool) {
	if !t.enabled || !r.found() {
		return "", nsRange{}, false
	}
	text := t.document().Text
	n := u16Len(text)
	if r.Location < 0 || r.Location >= n {
		return "", nsRange{}, false
	}
	length := min(r.Length, n-r.Location)
	return sliceU16(text, r.Location, r.Location+length), nsRange{r.Location, length}, true
}

// insertText types text, in the range of the document if there is one (a replacement range), in place of
// the text being composed if there is some (the input method commits its candidate), and in place of the
// selection otherwise. The composition ends.
//
// Flutter: FlutterTextInputPlugin.mm insertText:replacementRange:
func (t *textInput) insertText(s string, replace nsRange) (ui.TextInputEvent, bool) {
	if !t.enabled {
		return ui.TextInputEvent{}, false
	}
	d := t.document()
	from, to := t.target(d, replace)
	text := replaceU16(d.Text, from, to, s)
	caret := from + u16Len(s)
	return t.apply(textedit.NewEditingState(text, caret, caret))
}

// setMarkedText shows the text that the input method is composing in place of what was composing, or of
// the range, or of the selection; selected is the selection inside the new text. Empty text ends the
// composition.
//
// Flutter: FlutterTextInputPlugin.mm setMarkedText:selectedRange:replacementRange:
func (t *textInput) setMarkedText(s string, selected, replace nsRange) (ui.TextInputEvent, bool) {
	if !t.enabled {
		return ui.TextInputEvent{}, false
	}
	d := t.document()
	from, to := t.target(d, replace)
	text := replaceU16(d.Text, from, to, s)
	n := u16Len(s)
	if n == 0 {
		return t.apply(textedit.NewEditingState(text, from, from))
	}
	loc := min(max(selected.Location, 0), n)
	if !selected.found() {
		loc = n
	}
	length := min(max(selected.Length, 0), n-loc)
	return t.apply(textedit.NewEditingState(text, from+loc, from+loc+length).WithComposing(from, from+n))
}

// unmarkText makes the text being composed part of the text, as it is.
//
// Flutter: FlutterTextInputPlugin.mm unmarkText
func (t *textInput) unmarkText() (ui.TextInputEvent, bool) {
	if !t.hasMarked() {
		return ui.TextInputEvent{}, false
	}
	d := t.document()
	_, end, _ := d.Composing()
	return t.apply(textedit.NewEditingState(d.Text, end, end))
}

// target is the range of the document that a change replaces: the replacement range if the input method gave
// one, else the text being composed, else the selection. A range is cut to the document.
func (t *textInput) target(d textedit.EditingState, replace nsRange) (from, to int) {
	n := u16Len(d.Text)
	switch {
	case replace.found():
		return min(max(replace.Location, 0), n), min(max(replace.Location+replace.Length, 0), n)
	default:
		if b, e, ok := d.Composing(); ok {
			return b, e
		}
	}
	return min(d.MinOffset(), n), min(d.MaxOffset(), n)
}

// apply makes the document the one given, and returns the event that makes the framework's field the same.
func (t *textInput) apply(d textedit.EditingState) (ui.TextInputEvent, bool) {
	ev, ok := t.ed.Handle(d)
	ev.View = t.view
	return ev, ok
}

// caretInView is the caret rectangle that the input method puts its windows at, in logical pixels of the
// view. It is empty when the framework did not say.
func (t *textInput) caretInView() (r ui.Rect, ok bool) {
	return t.caret, t.enabled && t.caret != (ui.Rect{})
}

// u16Len is the length of s in UTF-16 units.
func u16Len(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// sliceU16 is the part of s from the UTF-16 offset from to the offset to. An offset in the middle of a
// surrogate pair is moved to the end of the character, and offsets are cut to s.
func sliceU16(s string, from, to int) string {
	a, b := byteAtU16(s, from), byteAtU16(s, to)
	if b < a {
		b = a
	}
	return s[a:b]
}

// replaceU16 puts repl in place of the part of s between the UTF-16 offsets from and to.
func replaceU16(s string, from, to int, repl string) string {
	a, b := byteAtU16(s, from), byteAtU16(s, to)
	if b < a {
		b = a
	}
	return s[:a] + repl + s[b:]
}

// byteAtU16 converts a UTF-16 offset of s to a byte offset; one in the middle of a character is the end of
// that character, and one out of s is cut to it.
func byteAtU16(s string, off int) int {
	if off <= 0 {
		return 0
	}
	n := 0
	for i, r := range s {
		if n >= off {
			return i
		}
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return len(s)
}
