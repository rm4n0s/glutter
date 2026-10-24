package textedit

import (
	"strings"
	"unicode/utf8"

	"github.com/rm4n0s/glutter/ui"
)

// TextEditor is the model behind the hidden text field (SPEC §23.5), as plain Go: it knows what the
// framework last said about the field (a ui.TextInputState, UTF-8) and what the DOM element holds
// (an EditingState, UTF-16), turns the state of the framework into a write to the element, and turns
// each change in the element into the ui.TextInputEvent that makes the framework's field the same.
// The code that talks to the browser only copies the fields of the events into these values.
//
// The framework owns the caret and the selection (it handles the arrow keys and the mouse), so only the
// text, and the text being composed, are sent from the element; a selection that the browser alone moved
// is not.
//
// Deviation: Flutter's engine sends the framework the whole editing state, or the deltas, and the
// framework's text field takes the platform's word for the selection. Here the contract is the one the
// Linux input methods use, ui.TextInputEvent, so the change is found by comparing the element's text with
// the framework's (SPEC §12.10).
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy
type TextEditor struct {
	// state is what the framework said last, or what the editor made of the events it sent since.
	state    ui.TextInputState
	hasState bool
	// dom is the element's state as last read or written, with the composing range.
	dom  EditingState
	comp Composition
	// preedit, preeditBegin and preeditEnd are the composing text the framework shows now, so that
	// an event that changes nothing is not sent again.
	preedit                  string
	preeditBegin, preeditEnd int
}

// DOMWrite is what to write into the element: its value and its selection in UTF-16 units. Write is false
// when the element already holds what the framework says.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditingState.applyToDomElement
type DOMWrite struct {
	Write        bool
	Text         string
	Base, Extent int
}

// Reset forgets the field: the next state is written to a new element.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.disable
func (e *TextEditor) Reset() { *e = TextEditor{} }

// State is what the editor believes the framework's field holds.
//
// Flutter: no counterpart
func (e *TextEditor) State() ui.TextInputState { return e.state }

// DOM is the state of the element as the editor knows it.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.lastEditingState
func (e *TextEditor) DOM() EditingState { return e.dom }

// Composing says whether the input method is composing text.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/composition_aware_mixin.dart CompositionAwareMixin.composingText
func (e *TextEditor) Composing() bool { return e.comp.Active() }

// CompositionStart handles compositionstart.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/composition_aware_mixin.dart CompositionAwareMixin._handleCompositionStart
func (e *TextEditor) CompositionStart() { e.comp.Start() }

// CompositionUpdate handles compositionupdate; data is the composing text.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/composition_aware_mixin.dart CompositionAwareMixin._handleCompositionUpdate
func (e *TextEditor) CompositionUpdate(data string) { e.comp.Update(data) }

// CompositionEnd handles compositionend. The caller reads the element and calls Handle afterwards, so
// that the composed text is committed (Flutter leaves that to the next input or selectionchange event).
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/composition_aware_mixin.dart CompositionAwareMixin._handleCompositionEnd
func (e *TextEditor) CompositionEnd() { e.comp.End() }

// BeforeInput handles beforeinput. An insertCompositionText that no compositionupdate announced
// (browsers differ in the order of the two) starts the composition with its data.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.handleBeforeInput
func (e *TextEditor) BeforeInput(inputType, data string) {
	if inputType == "insertCompositionText" && !e.comp.Active() && data != "" {
		e.comp.Update(data)
	}
}

// SetState takes the state the framework gave the field and says what to write into the element. The
// element is written only when it differs from the state, leaving out the text being composed: writing
// to it would end the composition, and the framework's state of a composing field is the text without it.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.setEditingState
func (e *TextEditor) SetState(state ui.TextInputState) DOMWrite {
	first := !e.hasState
	e.state, e.hasState = state, true
	text, start, end := e.surrounding()
	a, b := min(state.SelectionAnchor, state.SelectionCursor), max(state.SelectionAnchor, state.SelectionCursor)
	if !first && text == state.Text && start == a && end == b {
		return DOMWrite{}
	}
	base := utf16Offset(state.Text, state.SelectionAnchor)
	extent := utf16Offset(state.Text, state.SelectionCursor)
	e.dom = NewEditingState(state.Text, base, extent)
	e.comp.End()
	e.preedit, e.preeditBegin, e.preeditEnd = "", 0, 0
	return DOMWrite{Write: true, Text: state.Text, Base: e.dom.Base, Extent: e.dom.Extent}
}

// surrounding is the element's text without the text being composed, and the selection in it as byte
// offsets (start before end), as the framework's state holds them. A selection inside the composing text
// is at its start.
func (e *TextEditor) surrounding() (text string, start, end int) {
	d := e.dom
	cb, ce, composing := d.Composing()
	if !composing {
		return d.Text, byteOffset(d.Text, d.MinOffset()), byteOffset(d.Text, d.MaxOffset())
	}
	text = sliceUTF16(d.Text, 0, cb) + sliceUTF16(d.Text, ce, utf16Len(d.Text))
	out := func(off int) int {
		switch {
		case off >= ce:
			return off - (ce - cb)
		case off > cb:
			return cb
		}
		return off
	}
	return text, byteOffset(text, out(d.MinOffset())), byteOffset(text, out(d.MaxOffset()))
}

// Handle takes the state of the element after a DOM event (input, selectionchange, compositionend, or a
// compositionupdate that changes the element) and returns the event that makes the framework's field the
// same, and whether there is one. The composing range comes from the composition events. A change of the
// selection alone is not sent.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.handleChange
func (e *TextEditor) Handle(dom EditingState) (ui.TextInputEvent, bool) {
	dom = e.comp.Determine(dom)
	e.dom = dom
	text, _, selEnd := e.surrounding()
	var preedit string
	caret := selEnd
	begin, end := -1, -1
	if cb, ce, ok := dom.Composing(); ok {
		preedit = sliceUTF16(dom.Text, cb, ce)
		caret = byteOffset(text, cb)
		if dom.MinOffset() >= cb && dom.MaxOffset() <= ce {
			begin = byteOffset(preedit, dom.MinOffset()-cb)
			end = byteOffset(preedit, dom.MaxOffset()-cb)
		}
	}
	a, b := min(e.state.SelectionAnchor, e.state.SelectionCursor), max(e.state.SelectionAnchor, e.state.SelectionCursor)
	d := diffEdit(e.state.Text, a, b, text, caret)
	if !d.changed && preedit == e.preedit && (preedit == "" || (begin == e.preeditBegin && end == e.preeditEnd)) {
		return ui.TextInputEvent{}, false
	}
	if preedit == "" {
		begin, end = -1, -1
	}
	ev := ui.TextInputEvent{
		DeleteBefore: d.deleteBefore, DeleteAfter: d.deleteAfter, Commit: d.commit, DeleteSelection: d.deleteSelection,
		Preedit: preedit, PreeditCursorBegin: begin, PreeditCursorEnd: end,
	}
	e.preedit, e.preeditBegin, e.preeditEnd = preedit, begin, end
	// What the framework's field holds once it has applied the event: the text, and a caret after what was
	// committed, unless a selection survives.
	e.state.Text = text
	if a == b || ev.Commit != "" || ev.DeleteSelection || preedit != "" {
		pos := a - d.deleteBefore + len(d.commit)
		e.state.SelectionAnchor, e.state.SelectionCursor = pos, pos
	}
	return ev, true
}

// Paste takes the text of a paste event and returns the event that inserts it in place of the selection.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.handleClipboardEvent
func (e *TextEditor) Paste(text string) (ui.TextInputEvent, bool) {
	if text == "" {
		return ui.TextInputEvent{}, false
	}
	a, b := min(e.state.SelectionAnchor, e.state.SelectionCursor), max(e.state.SelectionAnchor, e.state.SelectionCursor)
	a, b = min(a, len(e.state.Text)), min(b, len(e.state.Text))
	e.state.Text = e.state.Text[:a] + text + e.state.Text[b:]
	e.state.SelectionAnchor, e.state.SelectionCursor = a+len(text), a+len(text)
	return ui.TextInputEvent{Commit: text, PreeditCursorBegin: -1, PreeditCursorEnd: -1}, true
}

// edit is the change from the framework's text to the element's, in the terms of a ui.TextInputEvent.
type edit struct {
	// changed says the text is not the same.
	changed                   bool
	deleteBefore, deleteAfter int
	commit                    string
	deleteSelection           bool
}

// diffEdit finds the change that turns old, with the selection [selStart, selEnd), into nw, whose caret
// is at caret after the change. The changed range ends at the caret when the text after the caret is the
// end of the old text (typing and deleting at the caret), and is found from the common beginning and end
// otherwise. A range that does not reach the selection is widened to it, because the event deletes around
// the selection and replaces it.
func diffEdit(old string, selStart, selEnd int, nw string, caret int) edit {
	selStart, selEnd = min(max(selStart, 0), len(old)), min(max(selEnd, 0), len(old))
	if selStart > selEnd {
		selStart, selEnd = selEnd, selStart
	}
	caret = alignRuneStart(nw, caret)
	var r0, r1 int
	var x string
	if tail := nw[caret:]; strings.HasSuffix(old, tail) {
		r1 = len(old) - len(tail)
		head := nw[:caret]
		r0 = commonPrefix(old[:r1], head)
		x = head[r0:]
	} else {
		r0 = commonPrefix(old, nw)
		q := commonSuffix(old[r0:], nw[r0:])
		r1 = len(old) - q
		x = nw[r0 : len(nw)-q]
	}
	if r0 == r1 && x == "" {
		return edit{}
	}
	lo, hi := min(r0, selStart), max(r1, selEnd)
	commit := old[lo:r0] + x + old[r1:hi]
	return edit{
		changed: true, deleteBefore: selStart - lo, deleteAfter: hi - selEnd, commit: commit,
		deleteSelection: selStart < selEnd && commit == "",
	}
}

// commonPrefix is the length in bytes of the longest common beginning of a and b that ends at a character.
func commonPrefix(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	for n > 0 && ((n < len(a) && !utf8.RuneStart(a[n])) || (n < len(b) && !utf8.RuneStart(b[n]))) {
		n--
	}
	return n
}

// commonSuffix is the length in bytes of the longest common end of a and b that begins at a character.
func commonSuffix(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[len(a)-1-n] == b[len(b)-1-n] {
		n++
	}
	for n > 0 && (!utf8.RuneStart(a[len(a)-n]) || !utf8.RuneStart(b[len(b)-n])) {
		n--
	}
	return n
}
