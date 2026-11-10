package ios

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// fakeField is the text field of the framework, which applies the events of the input method in the order
// of services' platformTextInputControl (the text being composed goes, the surrounding text is deleted, the
// commit replaces the selection, the new text to compose is shown at the cursor) and sends the state back as
// the framework does: the text without the text being composed.
type fakeField struct {
	t                *testing.T
	in               *textInput
	text             string
	base, extent     int // bytes
	compBeg, compEnd int // bytes; equal when nothing is composing
	events           []ui.TextInputEvent
}

func newField(t *testing.T, text string, base, extent int) *fakeField {
	f := &fakeField{t: t, in: &textInput{}, text: text, base: base, extent: extent}
	f.sendState(false)
	return f
}

func (f *fakeField) state(fromIM bool) ui.TextInputState {
	text, a, c := f.text, f.base, f.extent
	if f.compEnd > f.compBeg {
		out := func(off int) int {
			switch {
			case off >= f.compEnd:
				return off - (f.compEnd - f.compBeg)
			case off > f.compBeg:
				return f.compBeg
			}
			return off
		}
		text = f.text[:f.compBeg] + f.text[f.compEnd:]
		a, c = out(a), out(c)
	}
	return ui.TextInputState{Enabled: true, Text: text, SelectionAnchor: a, SelectionCursor: c,
		CaretRect: ui.Rect{Left: 10, Top: 20, Right: 11, Bottom: 40}, CausedByInputMethod: fromIM}
}

func (f *fakeField) sendState(fromIM bool) stateChange { return f.in.setState(f.state(fromIM)) }

func (f *fakeField) apply(e ui.TextInputEvent) {
	f.events = append(f.events, e)
	if f.compEnd > f.compBeg {
		f.text = f.text[:f.compBeg] + f.text[f.compEnd:]
		adj := func(off int) int {
			switch {
			case off >= f.compEnd:
				return off - (f.compEnd - f.compBeg)
			case off > f.compBeg:
				return f.compBeg
			}
			return off
		}
		f.base, f.extent = adj(f.base), adj(f.extent)
	}
	f.compBeg, f.compEnd = 0, 0
	lo, hi := min(f.base, f.extent), max(f.base, f.extent)
	if e.DeleteBefore > 0 {
		f.text = f.text[:lo-e.DeleteBefore] + f.text[lo:]
		lo, hi = lo-e.DeleteBefore, hi-e.DeleteBefore
	}
	if e.DeleteAfter > 0 {
		f.text = f.text[:hi] + f.text[hi+e.DeleteAfter:]
	}
	if e.Commit != "" || (lo != hi && (e.Preedit != "" || e.DeleteSelection)) {
		f.text = f.text[:lo] + e.Commit + f.text[hi:]
		f.base, f.extent = lo+len(e.Commit), lo+len(e.Commit)
	} else {
		f.base, f.extent = lo, hi
	}
	if e.Preedit != "" {
		at := max(f.base, f.extent)
		f.text = f.text[:at] + e.Preedit + f.text[at:]
		f.compBeg, f.compEnd = at, at+len(e.Preedit)
		b, x := e.PreeditCursorBegin, e.PreeditCursorEnd
		if b < 0 || x < 0 {
			b, x = len(e.Preedit), len(e.Preedit)
		}
		f.base, f.extent = at+b, at+x
	}
	f.sendState(true)
}

// check compares what the input method sees with what the framework's field holds: the document is the
// field's text, with the text being composed in it, and the selection and the composing range are the same
// in UTF-16 units.
func (f *fakeField) check() {
	f.t.Helper()
	d := f.in.document()
	if d.Text != f.text {
		f.t.Errorf("the input method sees %q, the field holds %q", d.Text, f.text)
	}
	if d.MinOffset() != u16Len(f.text[:min(f.base, f.extent)]) || d.MaxOffset() != u16Len(f.text[:max(f.base, f.extent)]) {
		f.t.Errorf("the input method's selection is %d-%d, the field's is %d-%d", d.MinOffset(), d.MaxOffset(), f.base, f.extent)
	}
	b, e, ok := d.Composing()
	if ok != (f.compEnd > f.compBeg) || (ok && (b != u16Len(f.text[:f.compBeg]) || e != u16Len(f.text[:f.compEnd]))) {
		f.t.Errorf("the input method's composing range is %d-%d (%v), the field's is %d-%d", b, e, ok, f.compBeg, f.compEnd)
	}
}

func (f *fakeField) insert(s string) {
	f.t.Helper()
	if ev, ok := f.in.insertText(s); ok {
		f.apply(ev)
	}
	f.check()
}

func (f *fakeField) mark(s string, sel textRange) {
	f.t.Helper()
	if ev, ok := f.in.setMarkedText(s, sel); ok {
		f.apply(ev)
	}
	f.check()
}

func (f *fakeField) unmark() {
	f.t.Helper()
	if ev, ok := f.in.unmarkText(); ok {
		f.apply(ev)
	}
	f.check()
}

func (f *fakeField) deleteBackward() {
	f.t.Helper()
	if ev, ok := f.in.deleteBackward(); ok {
		f.apply(ev)
	}
	f.check()
}

func (f *fakeField) replace(r textRange, s string) {
	f.t.Helper()
	if ev, ok := f.in.replaceRange(r, s); ok {
		f.apply(ev)
	}
	f.check()
}

// moveCaret is the framework moving the selection of the field (a tap, an arrow key) and sending the state: the model learns it.
func (f *fakeField) moveCaret(r textRange) {
	f.t.Helper()
	f.base, f.extent = byteAtU16(f.text, r.Start), byteAtU16(f.text, r.End)
	f.sendState(false)
	f.check()
}

func (f *fakeField) want(text string, composing string) {
	f.t.Helper()
	if f.compEnd > f.compBeg && f.text[f.compBeg:f.compEnd] != composing || f.compEnd == f.compBeg && composing != "" {
		f.t.Errorf("composing %q, want %q", f.text[f.compBeg:f.compEnd], composing)
	}
	if f.text != text {
		f.t.Errorf("text = %q, want %q", f.text, text)
	}
}

func TestPinyinComposesAndCommits(t *testing.T) {
	f := newField(t, "", 0, 0)
	f.mark("n", textRange{1, 1})
	f.want("n", "n")
	f.mark("ni", textRange{2, 2})
	f.mark("ni h", textRange{4, 4})
	f.want("ni h", "ni h")
	f.insert("你好")
	f.want("你好", "")
	if f.in.hasMarked() {
		t.Error("text is still marked after the commit")
	}
}

func TestJapaneseComposesInTheMiddleOfText(t *testing.T) {
	f := newField(t, "ab", 1, 1)
	f.mark("に", textRange{1, 1})
	f.want("aにb", "に")
	f.mark("にほ", textRange{2, 2})
	f.mark("日本", textRange{2, 2})
	f.want("a日本b", "日本")
	f.insert("日本")
	f.want("a日本b", "")
	if r := f.in.selectedRange(); r != (textRange{3, 3}) {
		t.Errorf("selection = %v, want the caret after the commit at 3", r)
	}
}

func TestMarkedTextWithASelectionInsideIt(t *testing.T) {
	f := newField(t, "", 0, 0)
	f.mark("hello", textRange{1, 3})
	if r := f.in.selectedRange(); r != (textRange{1, 3}) {
		t.Errorf("selection = %v, want 1-3 inside the marked text", r)
	}
	if r, ok := f.in.markedRange(); !ok || r != (textRange{0, 5}) {
		t.Errorf("marked range = %v %v, want 0-5", r, ok)
	}
}

func TestEmptyMarkedTextEndsTheComposition(t *testing.T) {
	f := newField(t, "x", 1, 1)
	f.mark("ab", textRange{2, 2})
	f.mark("", textRange{0, 0})
	f.want("x", "")
	if f.in.hasMarked() {
		t.Error("an empty marked text left a composition")
	}
}

func TestUnmarkTextKeepsTheCompositionAsText(t *testing.T) {
	f := newField(t, "", 0, 0)
	f.mark("abc", textRange{3, 3})
	f.unmark()
	f.want("abc", "")
	if _, ok := f.in.unmarkText(); ok {
		t.Error("unmarkText with nothing marked made an event")
	}
}

func TestMarkedTextReplacesTheSelection(t *testing.T) {
	f := newField(t, "abcd", 1, 3)
	f.mark("X", textRange{1, 1})
	f.want("aXd", "X")
}

func TestDeleteBackward(t *testing.T) {
	f := newField(t, "ab😀c", 7, 7) // the emoji is two units and four bytes: the caret is after the c
	f.deleteBackward()             // c
	f.want("ab😀", "")
	f.deleteBackward() // the emoji, both units at once
	f.want("ab", "")
	f.moveCaret(textRange{0, 2})
	f.deleteBackward() // the selection
	f.want("", "")
	f.deleteBackward() // nothing to delete
	f.want("", "")
}

func TestDeleteBackwardInsideAComposition(t *testing.T) {
	f := newField(t, "", 0, 0)
	f.mark("abc", textRange{3, 3})
	f.deleteBackward()
	f.want("ab", "ab")
	f.deleteBackward()
	f.deleteBackward()
	f.want("", "")
	if f.in.hasMarked() {
		t.Error("the composition outlived its last character")
	}
}

func TestReplaceRangeKeepsTheCaretWhereItWas(t *testing.T) {
	f := newField(t, "teh cat", 7, 7)
	f.replace(textRange{0, 3}, "the") // the autocorrection: same length
	f.want("the cat", "")
	if r := f.in.selectedRange(); r != (textRange{7, 7}) {
		t.Errorf("selection = %v after a same-length replacement, want 7", r)
	}
	f.replace(textRange{4, 7}, "kitten") // after the range: nothing moves; inside it: the caret goes to the end
	f.want("the kitten", "")
	if r := f.in.selectedRange(); r != (textRange{10, 10}) {
		t.Errorf("selection = %v, want 10 at the end of the new text", r)
	}
	f.moveCaret(textRange{10, 10})
	f.replace(textRange{0, 3}, "a") // before the caret: it moves by the change in length
	f.want("a kitten", "")
	if r := f.in.selectedRange(); r != (textRange{8, 8}) {
		t.Errorf("selection = %v, want 8", r)
	}
}

func TestSetSelectedRangeKeepsTheComposition(t *testing.T) {
	f := newField(t, "", 0, 0)
	f.mark("abc", textRange{3, 3})
	// UIKit moves the selection inside the composition: the model has it, the composition stays, and the framework is told through the
	// cursor of the preedit.
	ev, ok := f.in.setSelectedRange(textRange{1, 1})
	if !ok || ev.Preedit != "abc" || ev.PreeditCursorBegin != 1 || ev.PreeditCursorEnd != 1 {
		t.Errorf("event = %+v %v, want the preedit abc with its cursor at 1", ev, ok)
	}
	// Outside a composition the change is not sent, as in the other platforms.
	g := newField(t, "xyz", 3, 3)
	if _, ok := g.in.setSelectedRange(textRange{1, 1}); ok {
		t.Error("a change of the selection alone made an event")
	}
	f.want("abc", "abc")
	if r := f.in.selectedRange(); r != (textRange{1, 1}) {
		t.Errorf("selection = %v, want 1", r)
	}
}

func TestDocumentRangesAreInUTF16Units(t *testing.T) {
	f := newField(t, "a😀b", 0, 0)
	if f.in.length() != 4 {
		t.Errorf("length = %d, want 4", f.in.length())
	}
	if s := f.in.substring(textRange{1, 3}); s != "😀" {
		t.Errorf("substring = %q, want the emoji", s)
	}
	if s := f.in.substring(textRange{0, 100}); s != "a😀b" {
		t.Errorf("a range past the end gives %q", s)
	}
	if !f.in.hasText() {
		t.Error("hasText is false for a field with text")
	}
	f.moveCaret(textRange{4, 4})
	f.insert("é")
	f.want("a😀bé", "")
}

func TestStateChangeTellsWhatHappened(t *testing.T) {
	in := &textInput{}
	st := ui.TextInputState{Enabled: true, Text: "hi", SelectionAnchor: 2, SelectionCursor: 2, Purpose: ui.TextInputPurposeEmail}
	c := in.setState(st)
	if !c.enabled || c.wasEnabled || !c.configChanged || !c.textChanged {
		t.Errorf("a field gets the focus: %+v", c)
	}
	c = in.setState(st)
	if c.configChanged || c.textChanged {
		t.Errorf("the same state again changed something: %+v", c)
	}
	st.Text = "hi!"
	st.SelectionAnchor, st.SelectionCursor = 3, 3
	if c = in.setState(st); !c.textChanged || c.configChanged {
		t.Errorf("a new text: %+v", c)
	}
	st.Purpose = ui.TextInputPurposeURL
	if c = in.setState(st); !c.configChanged {
		t.Errorf("a new purpose: %+v", c)
	}
	in.setState(st)
	in.ed.CompositionStart()
	if ev, ok := in.setMarkedText("é", textRange{1, 1}); !ok || ev.View != 0 {
		t.Fatalf("setMarkedText: %+v %v", ev, ok)
	}
	if c = in.setState(ui.TextInputState{}); !c.markedLost || c.enabled || !c.wasEnabled {
		t.Errorf("the field loses the focus while composing: %+v", c)
	}
	if in.hasText() || in.hasMarked() {
		t.Error("a field that lost the focus still has text")
	}
}

func TestCaretRect(t *testing.T) {
	in := &textInput{}
	if _, ok := in.caretRect(); ok {
		t.Error("a disabled field has a caret rectangle")
	}
	in.setState(ui.TextInputState{Enabled: true, CaretRect: ui.Rect{Left: 5, Top: 6, Right: 5, Bottom: 26}})
	r, ok := in.caretRect()
	if !ok || r.Right != 6 || r.Left != 5 {
		t.Errorf("caret rect = %v %v, want a width of one point", r, ok)
	}
	in.setState(ui.TextInputState{Enabled: true})
	if _, ok := in.caretRect(); ok {
		t.Error("a field without a caret rectangle has one")
	}
}

func TestEventsCarryTheViewOfTheField(t *testing.T) {
	in := &textInput{}
	in.setState(ui.TextInputState{Enabled: true, View: 7})
	ev, ok := in.insertText("x")
	if !ok || ev.View != 7 {
		t.Errorf("event = %+v %v, want the view 7", ev, ok)
	}
}
