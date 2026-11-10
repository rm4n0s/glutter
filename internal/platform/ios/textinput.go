package ios

import (
	"github.com/rm4n0s/glutter/internal/platform/textedit"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the text input of the iOS backend that has no UIKit call in it (SPEC §30.5): the document that UIKit reads and writes
// through UITextInput, kept as the model of internal/platform/textedit keeps the text field of the web, of Android and of macOS, and the
// events that the framework's field gets from it. The UIKit side (the classes of the text input view, its positions and ranges) is in
// textinput_ios.go.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterTextInputPlugin.mm FlutterTextInputView (its text model is
// shell/platform/common/text_input_model.cc)

// textRange is a range of the document in UTF-16 units, as UITextRange is (a position is an offset); end is never before start.
type textRange struct{ Start, End int }

// length is the number of UTF-16 units of the range.
func (r textRange) length() int { return r.End - r.Start }

// empty says the range has no length: a caret.
func (r textRange) empty() bool { return r.Start == r.End }

// textInput is the text input of the platform: the document of the focused text field as UIKit sees it (the text with the text being
// composed in it), the place of the caret, the traits and the view that the field is in. It belongs to the UI thread.
type textInput struct {
	ed      textedit.TextEditor
	enabled bool
	view    ui.ViewID
	config  textConfig
	// caret is the rectangle of the caret in points of the view, from the top left.
	caret ui.Rect
}

// stateChange tells what a new state of the field did, for the UIKit side.
type stateChange struct {
	// enabled and wasEnabled are whether a field has the focus after and before.
	enabled, wasEnabled bool
	// configChanged says the traits of the field are not those of the one before (the keyboard must be reloaded).
	configChanged bool
	// markedLost says that the text that UIKit was composing is not in the document any more (the field lost the focus or its text was
	// set by the app): the input method must forget it.
	markedLost bool
	// textChanged says the text or the selection that the framework sent is not what the document held: UIKit is told with
	// textWillChange: and textDidChange: around it.
	textChanged bool
}

// setState takes the state of the focused text field (platform.Window.SetTextInput). The state is the text without the text being
// composed; UIKit's own change of the text, which the framework sent back (CausedByInputMethod), is what the model already holds, and the
// composition stays. A change that the app made ends the composition.
//
// Flutter: FlutterTextInputPlugin.mm setTextInputEditingState:, setTextInputClient:withConfiguration:
func (t *textInput) setState(st ui.TextInputState) stateChange {
	c := stateChange{wasEnabled: t.enabled}
	wasMarked := t.enabled && t.hasMarked()
	if !st.Enabled {
		t.ed.Reset()
		t.enabled, t.caret, t.config = false, ui.Rect{}, textConfig{}
		c.markedLost = wasMarked
		return c
	}
	before := t.document()
	cfg := configOf(st)
	c.configChanged = !t.enabled || cfg != t.config
	t.enabled, t.view, t.caret, t.config = true, st.View, st.CaretRect, cfg
	w := t.ed.SetState(st)
	c.enabled = true
	c.markedLost = wasMarked && w.Write
	c.textChanged = !before.Equal(t.document())
	return c
}

// document is the text that UIKit reads: the field's text with the text being composed in it.
func (t *textInput) document() textedit.EditingState { return t.ed.DOM() }

// length is the length of the document in UTF-16 units.
func (t *textInput) length() int { return u16Len(t.document().Text) }

// hasText says the document has text (UIKeyInput hasText).
//
// Flutter: FlutterTextInputPlugin.mm hasText
func (t *textInput) hasText() bool { return t.enabled && t.document().Text != "" }

// hasMarked says that text is being composed.
//
// Flutter: FlutterTextInputPlugin.mm markedTextRange
func (t *textInput) hasMarked() bool {
	_, _, ok := t.document().Composing()
	return t.enabled && ok
}

// markedRange is the range of the text being composed in the document.
func (t *textInput) markedRange() (textRange, bool) {
	if !t.enabled {
		return textRange{}, false
	}
	if b, e, ok := t.document().Composing(); ok {
		return textRange{b, e}, true
	}
	return textRange{}, false
}

// selectedRange is the selection in the document (a caret has no length). While text is being composed it is inside the composing text,
// where the input method put it.
//
// Flutter: FlutterTextInputPlugin.mm selectedTextRange
func (t *textInput) selectedRange() textRange {
	d := t.document()
	return textRange{d.MinOffset(), d.MaxOffset()}
}

// substring is the text of the document in the range, cut to the document.
//
// Flutter: FlutterTextInputPlugin.mm textInRange:
func (t *textInput) substring(r textRange) string {
	return sliceU16(t.document().Text, r.Start, r.End)
}

// insertText types text in place of the text being composed if there is some (the input method commits its candidate), and in place of
// the selection otherwise. The composition ends.
//
// Flutter: FlutterTextInputPlugin.mm insertText:
func (t *textInput) insertText(s string) (ui.TextInputEvent, bool) {
	if !t.enabled {
		return ui.TextInputEvent{}, false
	}
	d := t.document()
	from, to := t.target(d)
	text := replaceU16(d.Text, from, to, s)
	caret := from + u16Len(s)
	return t.apply(textedit.NewEditingState(text, caret, caret))
}

// deleteBackward deletes the selection, or the character before the caret: a character outside the Basic Multilingual Plane is two units.
// The text being composed stays composed as far as it is left.
//
// Flutter: FlutterTextInputPlugin.mm deleteBackward
func (t *textInput) deleteBackward() (ui.TextInputEvent, bool) {
	if !t.enabled {
		return ui.TextInputEvent{}, false
	}
	d := t.document()
	from, to := d.MinOffset(), d.MaxOffset()
	if from == to {
		if from == 0 {
			return ui.TextInputEvent{}, false
		}
		from -= u16Len(lastRune(sliceU16(d.Text, 0, from)))
	}
	text := replaceU16(d.Text, from, to, "")
	next := textedit.NewEditingState(text, from, from)
	if b, e, ok := d.Composing(); ok {
		// The composing range loses what was deleted of it.
		nb, ne := shrink(b, from, to), shrink(e, from, to)
		if ne > nb {
			next = next.WithComposing(nb, ne)
		}
	}
	return t.apply(next)
}

// shrink is the position p of a document after the range from to to was deleted from it.
func shrink(p, from, to int) int {
	switch {
	case p <= from:
		return p
	case p >= to:
		return p - (to - from)
	}
	return from
}

// lastRune is the last character of s, or "".
func lastRune(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	return string(r[len(r)-1])
}

// setMarkedText shows the text that the input method is composing in place of what was composing, or of the selection; selected is the
// selection inside the new text. Empty text ends the composition.
//
// Flutter: FlutterTextInputPlugin.mm setMarkedText:selectedRange:
func (t *textInput) setMarkedText(s string, selected textRange) (ui.TextInputEvent, bool) {
	if !t.enabled {
		return ui.TextInputEvent{}, false
	}
	d := t.document()
	from, to := t.target(d)
	text := replaceU16(d.Text, from, to, s)
	n := u16Len(s)
	if n == 0 {
		return t.apply(textedit.NewEditingState(text, from, from))
	}
	start := min(max(selected.Start, 0), n)
	end := min(max(selected.End, start), n)
	return t.apply(textedit.NewEditingState(text, from+start, from+end).WithComposing(from, from+n))
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

// replaceRange puts text in place of a range of the document, as the autocorrection and a suggestion do. The selection stays where it was
// in the text: before the range it does not move, after it it moves by the change in length, and inside it it goes to the end of the new text.
//
// Flutter: FlutterTextInputPlugin.mm replaceRange:withText:
func (t *textInput) replaceRange(r textRange, s string) (ui.TextInputEvent, bool) {
	if !t.enabled {
		return ui.TextInputEvent{}, false
	}
	d := t.document()
	n := u16Len(d.Text)
	from, to := min(max(r.Start, 0), n), min(max(r.End, 0), n)
	if to < from {
		from, to = to, from
	}
	text := replaceU16(d.Text, from, to, s)
	delta := u16Len(s) - (to - from)
	move := func(p int) int {
		switch {
		case p <= from:
			return p
		case p >= to:
			return p + delta
		}
		return from + u16Len(s)
	}
	base, extent := move(d.Base), move(d.Extent)
	next := textedit.NewEditingState(text, base, extent)
	// The text being composed stays composed when the change was not in it.
	if b, e, ok := d.Composing(); ok && (to <= b || from >= e) {
		next = next.WithComposing(move(b), move(e))
	}
	return t.apply(next)
}

// setSelectedRange moves the selection (UITextInput's setSelectedTextRange: from a tap, the space bar's cursor or a selection handle).
//
// Flutter: FlutterTextInputPlugin.mm setSelectedTextRange:
func (t *textInput) setSelectedRange(r textRange) (ui.TextInputEvent, bool) {
	if !t.enabled {
		return ui.TextInputEvent{}, false
	}
	d := t.document()
	n := u16Len(d.Text)
	next := textedit.NewEditingState(d.Text, min(max(r.Start, 0), n), min(max(r.End, 0), n))
	if b, e, ok := d.Composing(); ok {
		next = next.WithComposing(b, e)
	}
	return t.apply(next)
}

// target is the range of the document that a change replaces: the text being composed, else the selection.
func (t *textInput) target(d textedit.EditingState) (from, to int) {
	if b, e, ok := d.Composing(); ok {
		return b, e
	}
	n := u16Len(d.Text)
	return min(d.MinOffset(), n), min(d.MaxOffset(), n)
}

// apply makes the document the one given, and returns the event that makes the framework's field the same.
func (t *textInput) apply(d textedit.EditingState) (ui.TextInputEvent, bool) {
	ev, ok := t.ed.Handle(d)
	ev.View = t.view
	return ev, ok
}

// caretRect is the rectangle that UIKit puts its windows at (the caret and the selection rectangles), in points of the view, with a
// width of one point when the framework gave none. It is empty when the framework did not say.
//
// Flutter: FlutterTextInputPlugin.mm caretRectForPosition:
func (t *textInput) caretRect() (r ui.Rect, ok bool) {
	r = t.caret
	if !t.enabled || r == (ui.Rect{}) {
		return ui.Rect{}, false
	}
	if r.Right <= r.Left {
		r.Right = r.Left + 1
	}
	return r, true
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

// sliceU16 is the part of s from the UTF-16 offset from to the offset to. An offset in the middle of a surrogate pair is moved to the end
// of the character, and offsets are cut to s.
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

// byteAtU16 converts a UTF-16 offset of s to a byte offset; one in the middle of a character is the end of that character, and one out of
// s is cut to it.
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
