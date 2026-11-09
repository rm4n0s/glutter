package macos

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

func (f *fakeField) insert(s string, r ...nsRange) {
	f.t.Helper()
	rr := nsRange{notFound, 0}
	if len(r) > 0 {
		rr = r[0]
	}
	if ev, ok := f.in.insertText(s, rr); ok {
		f.apply(ev)
	}
	f.check()
}

func (f *fakeField) mark(s string, sel nsRange, r ...nsRange) {
	f.t.Helper()
	rr := nsRange{notFound, 0}
	if len(r) > 0 {
		rr = r[0]
	}
	if ev, ok := f.in.setMarkedText(s, sel, rr); ok {
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
	f.mark("n", nsRange{1, 0})
	f.want("n", "n")
	f.mark("ni", nsRange{2, 0})
	f.want("ni", "ni")
	if !f.in.hasMarked() || f.in.markedRange() != (nsRange{0, 2}) || f.in.selectedRange() != (nsRange{2, 0}) {
		t.Errorf("marked %v at %v, selected %v", f.in.hasMarked(), f.in.markedRange(), f.in.selectedRange())
	}
	f.insert("你") // the candidate is chosen: it replaces the text being composed
	f.want("你", "")
	if f.in.hasMarked() || f.in.markedRange().found() {
		t.Error("the composition did not end")
	}
	if got := f.events[len(f.events)-1]; got.Commit != "你" || got.Preedit != "" {
		t.Errorf("last event = %+v, want the commit of 你", got)
	}
}

func TestJapaneseComposesInTheMiddleOfText(t *testing.T) {
	f := newField(t, "abc def", 3, 3)
	f.mark("に", nsRange{1, 0})
	f.mark("にほん", nsRange{3, 0})
	f.want("abcにほん def", "にほん")
	if f.in.markedRange() != (nsRange{3, 3}) {
		t.Errorf("marked range = %v, want {3 3}", f.in.markedRange())
	}
	// The user moves inside the composition with the arrow keys: the selection of the marked text is a part.
	f.mark("にほん", nsRange{1, 1})
	if f.in.selectedRange() != (nsRange{4, 1}) {
		t.Errorf("selected range = %v, want {4 1}", f.in.selectedRange())
	}
	f.insert("日本")
	f.want("abc日本 def", "")
	if f.in.selectedRange() != (nsRange{5, 0}) {
		t.Errorf("after the commit the caret is at %v, want {5 0}", f.in.selectedRange())
	}
}

func TestKoreanSyllablesCommitOneByOne(t *testing.T) {
	f := newField(t, "", 0, 0)
	f.mark("ㅎ", nsRange{1, 0})
	f.mark("하", nsRange{1, 0})
	f.mark("한", nsRange{1, 0})
	// The next consonant ends the syllable and starts another: the input method commits and marks.
	f.insert("한")
	f.mark("ㄱ", nsRange{1, 0})
	f.want("한ㄱ", "ㄱ")
	f.mark("그", nsRange{1, 0})
	f.insert("그")
	f.want("한그", "")
}

func TestAccentMenuReplacesTheCharacterBeforeTheCaret(t *testing.T) {
	f := newField(t, "caf", 3, 3)
	// Press and hold e: the key types e, and the menu's choice replaces it.
	f.insert("e")
	f.want("cafe", "")
	// The input method marks the replacement over the e, in the document's coordinates, with no
	// composition going on.
	f.mark("é", nsRange{1, 0}, nsRange{3, 1})
	f.want("café", "é")
	if f.in.markedRange() != (nsRange{3, 1}) {
		t.Errorf("marked range = %v, want {3 1}", f.in.markedRange())
	}
	f.insert("é")
	f.want("café", "")
	if f.in.selectedRange() != (nsRange{4, 0}) {
		t.Errorf("caret = %v, want {4 0}", f.in.selectedRange())
	}
}

func TestDeadKeyMakesMarkedTextThatTheNextLetterReplaces(t *testing.T) {
	f := newField(t, "x", 1, 1)
	f.mark("´", nsRange{1, 0}) // Option+E
	f.want("x´", "´")
	f.insert("é") // e
	f.want("xé", "")
}

func TestEmptyMarkedTextEndsTheComposition(t *testing.T) {
	f := newField(t, "ab", 2, 2)
	f.mark("へ", nsRange{1, 0})
	f.want("abへ", "へ")
	f.mark("", nsRange{0, 0}) // the user deleted the last character of the composition
	f.want("ab", "")
	if f.in.hasMarked() {
		t.Error("the composition is still going on")
	}
	// Nothing is composing: unmarkText and an empty insert change nothing.
	n := len(f.events)
	f.unmark()
	f.insert("")
	if len(f.events) != n {
		t.Errorf("%d events more, want none", len(f.events)-n)
	}
}

func TestUnmarkTextKeepsTheCompositionAsText(t *testing.T) {
	f := newField(t, "", 0, 0)
	f.mark("かな", nsRange{2, 0})
	f.unmark()
	f.want("かな", "")
	if f.in.selectedRange() != (nsRange{2, 0}) {
		t.Errorf("caret = %v, want {2 0}", f.in.selectedRange())
	}
}

func TestMarkedTextReplacesTheSelection(t *testing.T) {
	f := newField(t, "hello", 0, 5)
	f.mark("你", nsRange{1, 0})
	f.want("你", "你")
	f = newField(t, "hello", 1, 4)
	f.insert("J")
	f.want("hJo", "")
}

func TestInsertTextWithAReplacementRange(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		caret int
		s     string
		r     nsRange
		want  string
	}{
		{"at the end", "abc", 3, "d", nsRange{3, 0}, "abcd"},
		{"beyond the end is cut to it", "abc", 3, "d", nsRange{50, 3}, "abcd"},
		{"over a character", "abc", 3, "X", nsRange{1, 1}, "aXc"},
		{"over a pair of UTF-16 units", "a😀b", 6, "!", nsRange{1, 2}, "a!b"},
		{"in the middle of a pair is moved to its end", "a😀b", 6, "!", nsRange{2, 0}, "a😀!b"},
		{"not found is the selection", "abc", 1, "X", nsRange{notFound, 0}, "aXbc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newField(t, tt.text, tt.caret, tt.caret)
			f.insert(tt.s, tt.r)
			f.want(tt.want, "")
		})
	}
}

func TestDocumentRangesAreInUTF16Units(t *testing.T) {
	// a, an emoji (two units), b, e and a combining acute (two units).
	f := newField(t, "a😀bé", len("a😀"), len("a😀"))
	if got := f.in.selectedRange(); got != (nsRange{3, 0}) {
		t.Errorf("selectedRange = %v, want {3 0}", got)
	}
	tests := []struct {
		name       string
		r          nsRange
		want       string
		wantActual nsRange
		ok         bool
	}{
		{"the emoji", nsRange{1, 2}, "😀", nsRange{1, 2}, true},
		{"the combining pair", nsRange{4, 2}, "é", nsRange{4, 2}, true},
		{"cut at the end", nsRange{4, 10}, "é", nsRange{4, 2}, true},
		{"the first character", nsRange{0, 1}, "a", nsRange{0, 1}, true},
		{"empty", nsRange{3, 0}, "", nsRange{3, 0}, true},
		{"after the end", nsRange{6, 1}, "", nsRange{}, false},
		{"not found", nsRange{notFound, 0}, "", nsRange{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, actual, ok := f.in.substring(tt.r)
			if got != tt.want || actual != tt.wantActual || ok != tt.ok {
				t.Errorf("substring(%v) = %q, %v, %v; want %q, %v, %v", tt.r, got, actual, ok, tt.want, tt.wantActual, tt.ok)
			}
		})
	}
}

func TestStateOfTheFieldEndsTheCompositionWhenTheAppChangesTheText(t *testing.T) {
	f := newField(t, "ab", 2, 2)
	f.mark("に", nsRange{1, 0})
	// The framework sends back the state that follows the input method's own change: the composition stays.
	if c := f.sendState(true); c.markedLost || !f.in.hasMarked() {
		t.Errorf("the echo of the input method's change lost the composition: %+v", c)
	}
	// The app sets another text: the composition is not in the document any more.
	c := f.in.setState(ui.TextInputState{Enabled: true, Text: "other", SelectionAnchor: 5, SelectionCursor: 5})
	if !c.markedLost || f.in.hasMarked() {
		t.Errorf("an app's change kept the composition: %+v", c)
	}
	if d := f.in.document(); d.Text != "other" {
		t.Errorf("document = %q, want other", d.Text)
	}
}

func TestFieldThatLosesTheFocusEndsTheCompositionAndTheModel(t *testing.T) {
	f := newField(t, "", 0, 0)
	f.mark("に", nsRange{1, 0})
	c := f.in.setState(ui.TextInputState{})
	if !c.markedLost || !c.wasEnabled || c.enabled {
		t.Errorf("change = %+v, want the composition lost by a field that was enabled and is not", c)
	}
	if f.in.hasMarked() || f.in.markedRange().found() || f.in.selectedRange().found() {
		t.Error("the model of a field that is gone still has a document")
	}
	if _, ok := f.in.insertText("x", nsRange{notFound, 0}); ok {
		t.Error("text was typed with no field")
	}
	if _, ok := f.in.setMarkedText("x", nsRange{1, 0}, nsRange{notFound, 0}); ok {
		t.Error("text was marked with no field")
	}
	if _, ok := f.in.unmarkText(); ok {
		t.Error("text was unmarked with no field")
	}
	if _, _, ok := f.in.substring(nsRange{0, 1}); ok {
		t.Error("a substring came from no field")
	}
}

func TestEventsCarryTheViewOfTheField(t *testing.T) {
	in := &textInput{}
	in.setState(ui.TextInputState{Enabled: true, View: 3, Text: "", CaretRect: ui.Rect{Right: 1, Bottom: 1}})
	ev, ok := in.insertText("a", nsRange{notFound, 0})
	if !ok || ev.View != 3 || ev.Commit != "a" {
		t.Errorf("event = %+v, %v; want the commit of a for view 3", ev, ok)
	}
	if r, ok := in.caretInView(); !ok || r.Right != 1 {
		t.Errorf("caretInView = %v, %v", r, ok)
	}
	in.setState(ui.TextInputState{Enabled: true, View: 3})
	if _, ok := in.caretInView(); ok {
		t.Error("an empty caret rectangle counts as one")
	}
}
