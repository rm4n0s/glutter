package web

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// field is a text editor whose framework state is the text with the selection [anchor, cursor] in bytes.
func field(text string, anchor, cursor int) *TextEditor {
	e := &TextEditor{}
	e.SetState(ui.TextInputState{Enabled: true, Text: text, SelectionAnchor: anchor, SelectionCursor: cursor})
	return e
}

// typed is the event of text that the user typed: nothing to delete, nothing composing.
func typed(commit string) ui.TextInputEvent {
	return ui.TextInputEvent{Commit: commit, PreeditCursorBegin: -1, PreeditCursorEnd: -1}
}

// del is the event of a deletion around the cursor.
func del(before, after int, commit string) ui.TextInputEvent {
	return ui.TextInputEvent{DeleteBefore: before, DeleteAfter: after, Commit: commit, PreeditCursorBegin: -1, PreeditCursorEnd: -1}
}

// TestHandlePortedFromFlutter has the cases of the TextEditingDeltaState group of text_editing_test.dart
// ('insertion', 'Backward deletion - Empty selection', 'Forward deletion - Empty selection', 'Deletion -
// Non-empty selection', 'composing region replacement', 'double space to insert a period', 'accent menu')
// and 'Supports deletion at inverted selection' and 'Supports new line at inverted selection': the same
// texts and selections, with the event of the input method in place of the delta.
func TestHandlePortedFromFlutter(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *TextEditor
		dom   EditingState
		want  ui.TextInputEvent
	}{
		{"insertion", func() *TextEditor { return field("worl", 4, 4) },
			NewEditingState("world", 5, 5), typed("d")},
		{"backward deletion, empty selection", func() *TextEditor { return field("world", 5, 5) },
			NewEditingState("worl", 4, 4), del(1, 0, "")},
		{"forward deletion, empty selection", func() *TextEditor { return field("world", 4, 4) },
			NewEditingState("worl", 4, 4), del(0, 1, "")},
		{"deletion, non-empty selection", func() *TextEditor { return field("world", 1, 5) },
			NewEditingState("w", 1, 1),
			ui.TextInputEvent{DeleteSelection: true, PreeditCursorBegin: -1, PreeditCursorEnd: -1}},
		{"double space inserts a period", func() *TextEditor { return field("hello ", 6, 6) },
			NewEditingState("hello. ", 7, 7), del(1, 0, ". ")},
		{"accent menu", func() *TextEditor { return field("a", 1, 1) },
			NewEditingState("à", 1, 1), del(1, 0, "à")},
		{"deletion at an inverted selection", func() *TextEditor { return field("Hello world", 9, 3) },
			NewEditingState("Helld", 3, 3),
			ui.TextInputEvent{DeleteSelection: true, PreeditCursorBegin: -1, PreeditCursorEnd: -1}},
		{"new line at an inverted selection", func() *TextEditor { return field("Hello world", 9, 3) },
			NewEditingState("Hel\nld", 3, 3), typed("\n")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.setup().Handle(tc.dom)
			if !ok || !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v (%v), want %+v", got, ok, tc.want)
			}
		})
	}

	t.Run("composing region replacement", func(t *testing.T) {
		// 'ni hao ma' is composed, and the input method replaces it with '你好吗' and commits.
		e := field("", 0, 0)
		e.CompositionStart()
		e.CompositionUpdate("ni hao ma")
		got, ok := e.Handle(NewEditingState("ni hao ma", 9, 9))
		want := ui.TextInputEvent{Preedit: "ni hao ma", PreeditCursorBegin: 9, PreeditCursorEnd: 9}
		if !ok || !reflect.DeepEqual(got, want) {
			t.Fatalf("composing: got %+v (%v), want %+v", got, ok, want)
		}
		if d := e.DOM(); d.ComposingBase != 0 || d.ComposingExtent != 9 {
			t.Errorf("composing range = %d..%d, want 0..9", d.ComposingBase, d.ComposingExtent)
		}
		e.CompositionUpdate("你好吗")
		got, ok = e.Handle(NewEditingState("你好吗", 3, 3))
		want = ui.TextInputEvent{Preedit: "你好吗", PreeditCursorBegin: 9, PreeditCursorEnd: 9}
		if !ok || !reflect.DeepEqual(got, want) {
			t.Fatalf("updated: got %+v (%v), want %+v", got, ok, want)
		}
		e.CompositionEnd()
		got, ok = e.Handle(NewEditingState("你好吗", 3, 3))
		if !ok || !reflect.DeepEqual(got, typed("你好吗")) {
			t.Errorf("committed: got %+v (%v), want %+v", got, ok, typed("你好吗"))
		}
	})
}

// TestComposingRangePortedFromFlutter has the two cases of the 'composing range' group of
// composition_test.dart: [0, length] on a new composition and [before - length, before] in the middle.
func TestComposingRangePortedFromFlutter(t *testing.T) {
	t.Run("on a new composition", func(t *testing.T) {
		e := field("", 0, 0)
		e.CompositionUpdate("hi")
		e.Handle(NewEditingState("hi", 2, 2))
		if d := e.DOM(); d.ComposingBase != 0 || d.ComposingExtent != 2 {
			t.Errorf("composing = %d..%d, want 0..2", d.ComposingBase, d.ComposingExtent)
		}
	})
	t.Run("in the middle of the text", func(t *testing.T) {
		const before, after = "beforeComposingText", "afterComposingText"
		e := field(before+after, len(before), len(before))
		e.CompositionUpdate("hi")
		e.Handle(NewEditingState(before+after, len(before), len(before)))
		if d := e.DOM(); d.ComposingBase != len(before)-2 || d.ComposingExtent != len(before) {
			t.Errorf("composing = %d..%d, want %d..%d", d.ComposingBase, d.ComposingExtent, len(before)-2, len(before))
		}
	})
}

func TestHandleTypingAndEditing(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *TextEditor
		dom   EditingState
		want  ui.TextInputEvent
		ok    bool
	}{
		{"typing at the end", func() *TextEditor { return field("ab", 2, 2) }, NewEditingState("abc", 3, 3), typed("c"), true},
		{"typing in the middle", func() *TextEditor { return field("ac", 1, 1) }, NewEditingState("abc", 2, 2), typed("b"), true},
		{"typing a repeated character", func() *TextEditor { return field("aa", 2, 2) }, NewEditingState("aaa", 3, 3), typed("a"), true},
		{"typing over a selection", func() *TextEditor { return field("abcd", 1, 3) }, NewEditingState("aXd", 2, 2), typed("X"), true},
		{"typing a multibyte character", func() *TextEditor { return field("añb", 3, 3) }, NewEditingState("añéb", 3, 3), typed("é"), true},
		{"typing an emoji (two UTF-16 units)", func() *TextEditor { return field("a", 1, 1) }, NewEditingState("a😀", 3, 3), typed("😀"), true},
		{"backspace over a multibyte character", func() *TextEditor { return field("añ", 3, 3) }, NewEditingState("a", 1, 1), del(2, 0, ""), true},
		{"a soft keyboard's backspace in the middle", func() *TextEditor { return field("abc", 2, 2) }, NewEditingState("ac", 1, 1), del(1, 0, ""), true},
		{"a replaced word", func() *TextEditor { return field("teh ", 4, 4) }, NewEditingState("the ", 4, 4), del(3, 0, "he "), true},
		{"a change far from the caret reaches it", func() *TextEditor { return field("abcdef", 6, 6) }, NewEditingState("Xbcdef", 6, 6), del(6, 0, "Xbcdef"), true},
		{"no change", func() *TextEditor { return field("abc", 3, 3) }, NewEditingState("abc", 3, 3), ui.TextInputEvent{}, false},
		{"a selection that the browser moved alone", func() *TextEditor { return field("abc", 3, 3) }, NewEditingState("abc", 0, 1), ui.TextInputEvent{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.setup().Handle(tc.dom)
			if ok != tc.ok || (ok && !reflect.DeepEqual(got, tc.want)) {
				t.Errorf("got %+v (%v), want %+v (%v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

// TestHandleKeepsTheFrameworksViewOfTheText: the events that follow each other, before the framework has
// answered, are each found against what the framework holds after the one before.
func TestHandleKeepsTheFrameworksViewOfTheText(t *testing.T) {
	e := field("", 0, 0)
	var commits []string
	text := ""
	for _, r := range "héllo" {
		text += string(r)
		ev, ok := e.Handle(NewEditingState(text, utf16Len(text), utf16Len(text)))
		if !ok {
			t.Fatalf("no event for %q", text)
		}
		commits = append(commits, ev.Commit)
	}
	if !reflect.DeepEqual(commits, []string{"h", "é", "l", "l", "o"}) {
		t.Errorf("commits = %q", commits)
	}
	if got := e.State(); got.Text != "héllo" || got.SelectionCursor != len("héllo") {
		t.Errorf("state = %+v, want the typed text with the caret at its end", got)
	}
}

func TestHandleComposition(t *testing.T) {
	e := field("ab", 2, 2)
	steps := []struct {
		name string
		do   func()
		dom  EditingState
		want ui.TextInputEvent
		ok   bool
	}{
		{"the first character is shown, composing", func() { e.CompositionStart(); e.CompositionUpdate("n") },
			NewEditingState("abn", 3, 3), ui.TextInputEvent{Preedit: "n", PreeditCursorBegin: 1, PreeditCursorEnd: 1}, true},
		{"the same composition again sends nothing", func() {},
			NewEditingState("abn", 3, 3), ui.TextInputEvent{}, false},
		{"it grows", func() { e.CompositionUpdate("ni") },
			NewEditingState("abni", 4, 4), ui.TextInputEvent{Preedit: "ni", PreeditCursorBegin: 2, PreeditCursorEnd: 2}, true},
		{"the input method's own cursor moves in it", func() {},
			NewEditingState("abni", 3, 3), ui.TextInputEvent{Preedit: "ni", PreeditCursorBegin: 1, PreeditCursorEnd: 1}, true},
		{"it becomes the candidate", func() { e.CompositionUpdate("你") },
			NewEditingState("ab你", 3, 3), ui.TextInputEvent{Preedit: "你", PreeditCursorBegin: 3, PreeditCursorEnd: 3}, true},
		{"it is committed", func() { e.CompositionEnd() },
			NewEditingState("ab你", 3, 3), typed("你"), true},
	}
	for _, s := range steps {
		s.do()
		got, ok := e.Handle(s.dom)
		if ok != s.ok || (ok && !reflect.DeepEqual(got, s.want)) {
			t.Fatalf("%s: got %+v (%v), want %+v (%v)", s.name, got, ok, s.want, s.ok)
		}
	}
	if e.Composing() {
		t.Error("still composing after compositionend")
	}
	if got := e.State(); got.Text != "ab你" || got.SelectionCursor != len("ab你") {
		t.Errorf("state = %+v", got)
	}

	t.Run("cancelled without a commit", func(t *testing.T) {
		e := field("ab", 2, 2)
		e.CompositionUpdate("x")
		e.Handle(NewEditingState("abx", 3, 3))
		e.CompositionEnd()
		got, ok := e.Handle(NewEditingState("ab", 2, 2))
		want := ui.TextInputEvent{PreeditCursorBegin: -1, PreeditCursorEnd: -1}
		if !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v (%v), want %+v: the framework must drop its preedit", got, ok, want)
		}
	})
	t.Run("composing over a selection replaces it", func(t *testing.T) {
		e := field("abcd", 1, 3)
		e.CompositionUpdate("X")
		got, ok := e.Handle(NewEditingState("aXd", 2, 2))
		want := ui.TextInputEvent{DeleteSelection: true, Preedit: "X", PreeditCursorBegin: 1, PreeditCursorEnd: 1}
		if !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v (%v), want %+v", got, ok, want)
		}
	})
	t.Run("an insertCompositionText that no compositionupdate announced starts the composition", func(t *testing.T) {
		e := field("", 0, 0)
		e.BeforeInput("insertCompositionText", "x")
		if !e.Composing() {
			t.Error("not composing")
		}
		got, ok := e.Handle(NewEditingState("x", 1, 1))
		if !ok || got.Preedit != "x" {
			t.Errorf("got %+v (%v), want the preedit x", got, ok)
		}
		e = field("", 0, 0)
		e.BeforeInput("insertText", "x")
		e.BeforeInput("insertCompositionText", "")
		if e.Composing() {
			t.Error("composing without composing text")
		}
	})
}

func TestSetState(t *testing.T) {
	st := func(text string, anchor, cursor int) ui.TextInputState {
		return ui.TextInputState{Enabled: true, Text: text, SelectionAnchor: anchor, SelectionCursor: cursor}
	}
	t.Run("the first state is written", func(t *testing.T) {
		e := &TextEditor{}
		got := e.SetState(st("abc", 3, 3))
		if want := (DOMWrite{Write: true, Text: "abc", Base: 3, Extent: 3}); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
	t.Run("the same state is not written again", func(t *testing.T) {
		e := field("abc", 1, 2)
		if got := e.SetState(st("abc", 1, 2)); got.Write {
			t.Errorf("wrote %+v", got)
		}
	})
	t.Run("a new selection is written, the flipped one flipped (an inverted selection)", func(t *testing.T) {
		e := field("Hello world", 3, 3)
		got := e.SetState(st("Hello world", 9, 3))
		if want := (DOMWrite{Write: true, Text: "Hello world", Base: 9, Extent: 3}); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
	t.Run("offsets are UTF-16 units in the element", func(t *testing.T) {
		got := (&TextEditor{}).SetState(st("é😀x", len("é😀"), len("é😀x")))
		if want := (DOMWrite{Write: true, Text: "é😀x", Base: 3, Extent: 4}); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
	t.Run("the echo of an event is not written", func(t *testing.T) {
		e := field("ab", 2, 2)
		e.Handle(NewEditingState("abc", 3, 3))
		if got := e.SetState(st("abc", 3, 3)); got.Write {
			t.Errorf("wrote %+v: the element already holds it", got)
		}
	})
	t.Run("text the framework changed is written, and the composition ends", func(t *testing.T) {
		e := field("ab", 2, 2)
		e.CompositionUpdate("x")
		e.Handle(NewEditingState("abx", 3, 3))
		got := e.SetState(st("ab!", 3, 3))
		if want := (DOMWrite{Write: true, Text: "ab!", Base: 3, Extent: 3}); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
		if e.Composing() {
			t.Error("still composing after the element was rewritten")
		}
	})
	t.Run("a paste is written, which the browser did not insert", func(t *testing.T) {
		e := field("ab", 1, 1)
		if ev, ok := e.Paste("XY"); !ok || !reflect.DeepEqual(ev, typed("XY")) {
			t.Fatalf("paste = %+v (%v)", ev, ok)
		}
		got := e.SetState(st("aXYb", 3, 3))
		if want := (DOMWrite{Write: true, Text: "aXYb", Base: 3, Extent: 3}); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
	t.Run("the state of a composing field leaves out the composing text", func(t *testing.T) {
		e := field("ab", 2, 2)
		e.CompositionUpdate("你好")
		e.Handle(NewEditingState("ab你好", 4, 4))
		// The framework shows the preedit and says the text around it, with the caret before it.
		if got := e.SetState(st("ab", 2, 2)); got.Write {
			t.Errorf("wrote %+v: that would end the composition", got)
		}
		// Typing after the caret in the framework's field is a change the element does not have.
		if got := e.SetState(st("ab", 1, 1)); !got.Write {
			t.Error("a new caret was not written")
		}
	})
}

func TestPaste(t *testing.T) {
	tests := []struct {
		name       string
		e          *TextEditor
		text       string
		ok         bool
		after      string
		afterCaret int
	}{
		{"at the caret", field("ab", 1, 1), "XY", true, "aXYb", 3},
		{"over a selection", field("abcd", 3, 1), "-", true, "a-d", 2},
		{"multibyte", field("é", 2, 2), "你", true, "é你", 5},
		{"nothing to paste", field("ab", 1, 1), "", false, "ab", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ev, ok := tc.e.Paste(tc.text)
			if ok != tc.ok || (ok && !reflect.DeepEqual(ev, typed(tc.text))) {
				t.Errorf("event = %+v (%v)", ev, ok)
			}
			if s := tc.e.State(); s.Text != tc.after || s.SelectionCursor != tc.afterCaret {
				t.Errorf("state = %q caret %d, want %q caret %d", s.Text, s.SelectionCursor, tc.after, tc.afterCaret)
			}
		})
	}
}

func TestDiffEdit(t *testing.T) {
	tests := []struct {
		name       string
		old        string
		selA, selB int
		nw         string
		caret      int
		want       edit
	}{
		{"same text", "abc", 3, 3, "abc", 3, edit{}},
		{"insertion at the caret", "ac", 1, 1, "abc", 2, edit{changed: true, commit: "b"}},
		{"an ambiguous insertion is anchored at the caret", "aab", 1, 1, "aaab", 2, edit{changed: true, commit: "a"}},
		{"a multibyte deletion is aligned to the character", "é", 2, 2, "", 0, edit{changed: true, deleteBefore: 2}},
		{"a text with the same bytes before and after", "日本語", 6, 6, "日語", 3, edit{changed: true, deleteBefore: 3}},
		{"selection deleted", "abcd", 1, 3, "ad", 1, edit{changed: true, deleteSelection: true}},
		{"selection replaced by a longer text", "abcd", 1, 3, "aXYZd", 4, edit{changed: true, commit: "XYZ"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := diffEdit(tc.old, tc.selA, tc.selB, tc.nw, tc.caret); got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}
