package ime

import (
	"reflect"
	"slices"
	"strconv"
	"testing"
	"unicode/utf16"

	"github.com/rm4n0s/glutter/internal/platform/windows/ime/imetest"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

func u16(s string) []uint16 { return utf16.Encode([]rune(s)) }

const testHWND = 0x1005c

// framework is a text field as the framework holds it: it applies the events in the order of
// ui.TextInputEvent and tells the input what its state is, as the platform's text input control
// does after each change.
type framework struct {
	text         string
	anchor, ext  int
	preedit      string
	preeditStart int
	events       []ui.TextInputEvent
	in           *Input
}

// state is what the framework sends to the platform: the text without the preedit, with the
// selection outside it (at its start when it was inside).
func (f *framework) state() ui.TextInputState {
	text, a, e := f.text, f.anchor, f.ext
	if f.preedit != "" {
		s, n := f.preeditStart, len(f.preedit)
		text = f.text[:s] + f.text[s+n:]
		out := func(o int) int {
			switch {
			case o >= s+n:
				return o - n
			case o > s:
				return s
			}
			return o
		}
		a, e = out(a), out(e)
	}
	return ui.TextInputState{Enabled: true, Text: text, SelectionAnchor: a, SelectionCursor: e,
		CaretRect: ui.Rect{Left: 10, Top: 20, Right: 12, Bottom: 40}}
}

func (f *framework) send() { f.in.SetState(f.state(), 1) }

// handle applies an event the way platformTextInputControl.handleEvent does.
func (f *framework) handle(ev ui.TextInputEvent) {
	f.events = append(f.events, ev)
	text, sel0, sel1 := f.text, min(f.anchor, f.ext), max(f.anchor, f.ext)
	if f.preedit != "" {
		s, n := f.preeditStart, len(f.preedit)
		text = text[:s] + text[s+n:]
		fix := func(o int) int {
			if o >= s+n {
				return o - n
			}
			return min(o, s)
		}
		sel0, sel1 = fix(sel0), fix(sel1)
		f.preedit = ""
	}
	if ev.DeleteBefore > 0 {
		text = text[:sel0-ev.DeleteBefore] + text[sel0:]
		sel1 -= ev.DeleteBefore
		sel0 -= ev.DeleteBefore
	}
	if ev.DeleteAfter > 0 {
		text = text[:sel1] + text[sel1+ev.DeleteAfter:]
	}
	if ev.Commit != "" || (sel0 != sel1 && (ev.Preedit != "" || ev.DeleteSelection)) {
		text = text[:sel0] + ev.Commit + text[sel1:]
		sel0 += len(ev.Commit)
		sel1 = sel0
	}
	f.anchor, f.ext = sel0, sel1
	if ev.Preedit != "" {
		at := sel1
		text = text[:at] + ev.Preedit + text[at:]
		f.preedit, f.preeditStart = ev.Preedit, at
		b, e := ev.PreeditCursorBegin, ev.PreeditCursorEnd
		if b < 0 || e < 0 {
			b, e = len(ev.Preedit), len(ev.Preedit)
		}
		f.anchor, f.ext = at+b, at+e
	}
	f.text = text
	f.send()
}

// newField makes an input over a fake IMM with a text field that has the focus, text and a caret
// at the byte offset caret (selecting to the offset to when to is not caret).
func newField(t *testing.T, text string, caret, to int) (*framework, *imetest.Fake) {
	t.Helper()
	imm := imetest.New(testHWND)
	f := &framework{text: text, anchor: caret, ext: to}
	f.in = New(imm, testHWND, f.handle)
	f.send()
	return f, imm
}

// step is a message of the input method: the strings it holds in its input context and the message.
type step struct {
	name string
	// msg is "start", "comp", "end" or "char".
	msg    string
	comp   string
	result string
	// cursor is the position in the composition string, in UTF-16 units; -1 leaves GCS_CURSORPOS out.
	cursor int32
	flags  uintptr
	// unit is the character of "char".
	unit uint16
	// wantText and wantPreedit are the framework's text and its preedit after the step; wantSel is its
	// selection ("" when it is not checked).
	wantText, wantPreedit string
	wantSel               [2]int
	// wantEvents is the number of events the step makes.
	wantEvents int
}

func (s step) run(t *testing.T, f *framework, imm *imetest.Fake) {
	t.Helper()
	before := len(f.events)
	switch s.msg {
	case "start":
		f.in.HandleStartComposition()
	case "comp":
		imm.Comp, imm.Result = u16(s.comp), u16(s.result)
		imm.Cursor = max(s.cursor, 0)
		flags := s.flags
		if flags == 0 {
			flags = win32.GCSCompStr
			if s.cursor >= 0 {
				flags |= win32.GCSCursorPos
			}
			if s.result != "" {
				flags |= win32.GCSResultStr
			}
		}
		f.in.HandleComposition(flags)
	case "end":
		f.in.HandleEndComposition()
	case "char":
		f.in.HandleChar(s.unit)
	}
	if n := len(f.events) - before; n != s.wantEvents {
		t.Errorf("%s: %d events, want %d: %+v", s.name, n, s.wantEvents, f.events[before:])
	}
	if f.text != s.wantText || f.preedit != s.wantPreedit {
		t.Errorf("%s: text %q with preedit %q, want %q with %q", s.name, f.text, f.preedit, s.wantText, s.wantPreedit)
	}
	if s.wantSel != [2]int{} && (f.anchor != s.wantSel[0] || f.ext != s.wantSel[1]) {
		t.Errorf("%s: selection %d..%d, want %d..%d", s.name, f.anchor, f.ext, s.wantSel[0], s.wantSel[1])
	}
}

func TestCompositionSessions(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		caret, to    int
		steps        []step
		finalText    string
		finalCaret   int
		composingEnd bool
	}{
		{
			// Japanese: roman letters become kana, the caret moves inside the string, a conversion
			// replaces it with kanji, and Enter commits.
			name: "a Japanese session with the caret inside and a conversion", text: "abcd", caret: 2, to: 2,
			steps: []step{
				{name: "start", msg: "start", wantText: "abcd", wantEvents: 0},
				{name: "k", msg: "comp", comp: "k", cursor: 1, wantText: "abkcd", wantPreedit: "k", wantSel: [2]int{3, 3}, wantEvents: 1},
				{name: "か", msg: "comp", comp: "か", cursor: 1, wantText: "abかcd", wantPreedit: "か", wantSel: [2]int{5, 5}, wantEvents: 1},
				{name: "かん with the caret after か", msg: "comp", comp: "かん", cursor: 1, wantText: "abかんcd", wantPreedit: "かん", wantSel: [2]int{5, 5}, wantEvents: 1},
				{name: "the same string again", msg: "comp", comp: "かん", cursor: 1, wantText: "abかんcd", wantPreedit: "かん", wantEvents: 0},
				{name: "the conversion", msg: "comp", comp: "漢字", cursor: 2, wantText: "ab漢字cd", wantPreedit: "漢字", wantSel: [2]int{8, 8}, wantEvents: 1},
				{name: "the result", msg: "comp", result: "漢字", cursor: -1, flags: win32.GCSResultStr, wantText: "ab漢字cd", wantPreedit: "", wantSel: [2]int{8, 8}, wantEvents: 1},
				{name: "end", msg: "end", wantText: "ab漢字cd", wantEvents: 0},
			},
			finalText: "ab漢字cd", finalCaret: 8,
		},
		{
			// Chinese: the candidate is chosen, so the message has the result and an empty string.
			name: "a Chinese session committed by a candidate", text: "", caret: 0, to: 0,
			steps: []step{
				{name: "start", msg: "start", wantText: ""},
				{name: "n", msg: "comp", comp: "n", cursor: 1, wantText: "n", wantPreedit: "n", wantEvents: 1},
				{name: "ni", msg: "comp", comp: "ni", cursor: 2, wantText: "ni", wantPreedit: "ni", wantEvents: 1},
				{name: "nihao", msg: "comp", comp: "ni hao", cursor: 6, wantText: "ni hao", wantPreedit: "ni hao", wantEvents: 1},
				{name: "the candidate", msg: "comp", comp: "", result: "你好", cursor: 0, flags: win32.GCSResultStr | win32.GCSCompStr, wantText: "你好", wantPreedit: "", wantSel: [2]int{6, 6}, wantEvents: 1},
				{name: "end", msg: "end", wantText: "你好"},
			},
			finalText: "你好", finalCaret: 6,
		},
		{
			name: "a commit with no preedit before it", text: "ab", caret: 1, to: 1,
			steps: []step{
				{name: "the result alone", msg: "comp", result: "X", cursor: -1, flags: win32.GCSResultStr, wantText: "aXb", wantSel: [2]int{2, 2}, wantEvents: 1},
				{name: "end", msg: "end", wantText: "aXb"},
			},
			finalText: "aXb", finalCaret: 2,
		},
		{
			// Google Japanese Input and ATOK commit and start the next composition in one message.
			name: "a result and a new composition string in one message", text: "", caret: 0, to: 0,
			steps: []step{
				{name: "한", msg: "comp", comp: "한", cursor: 1, wantText: "한", wantPreedit: "한", wantEvents: 1},
				{name: "한 and 그", msg: "comp", comp: "그", result: "한", cursor: 1, wantText: "한그", wantPreedit: "그", wantSel: [2]int{6, 6}, wantEvents: 1},
				{name: "end commits what is left", msg: "end", wantText: "한그", wantPreedit: "", wantEvents: 1},
			},
			finalText: "한그", finalCaret: 6,
		},
		{
			name: "the first string replaces the selection", text: "abcde", caret: 1, to: 4,
			steps: []step{
				{name: "x", msg: "comp", comp: "x", cursor: 1, wantText: "axe", wantPreedit: "x", wantSel: [2]int{2, 2}, wantEvents: 1},
				{name: "xy", msg: "comp", comp: "xy", cursor: 2, wantText: "axye", wantPreedit: "xy", wantSel: [2]int{3, 3}, wantEvents: 1},
				{name: "the result", msg: "comp", result: "Z", cursor: -1, flags: win32.GCSResultStr, wantText: "aZe", wantSel: [2]int{2, 2}, wantEvents: 1},
			},
			finalText: "aZe", finalCaret: 2,
		},
		{
			// 𠮷 is two UTF-16 units and four bytes: a position of 3 units is the end of "a𠮷".
			name: "a surrogate pair counts two units and four bytes", text: "", caret: 0, to: 0,
			steps: []step{
				{name: "a𠮷b with the caret after 𠮷", msg: "comp", comp: "a𠮷b", cursor: 3, wantText: "a𠮷b", wantPreedit: "a𠮷b", wantSel: [2]int{5, 5}, wantEvents: 1},
				{name: "the caret in the middle of the pair moves back", msg: "comp", comp: "a𠮷c", cursor: 2, wantText: "a𠮷c", wantPreedit: "a𠮷c", wantSel: [2]int{1, 1}, wantEvents: 1},
				{name: "the result", msg: "comp", result: "a𠮷c", cursor: -1, flags: win32.GCSResultStr, wantText: "a𠮷c", wantSel: [2]int{6, 6}, wantEvents: 1},
			},
			finalText: "a𠮷c", finalCaret: 6,
		},
		{
			name: "text with combining marks", text: "", caret: 0, to: 0,
			steps: []step{
				// "e" and U+0301 are two units and three bytes; the caret between them is at byte 1.
				{name: "e then the accent, caret between", msg: "comp", comp: "éx", cursor: 1, wantText: "éx", wantPreedit: "éx", wantSel: [2]int{1, 1}, wantEvents: 1},
				{name: "the caret after the accent", msg: "comp", comp: "éx", cursor: 2, wantText: "éx", wantPreedit: "éx", wantSel: [2]int{3, 3}, wantEvents: 1},
			},
			finalText: "éx", finalCaret: 3,
		},
		{
			name: "no cursor position in the message puts the caret at the end", text: "", caret: 0, to: 0,
			steps: []step{
				{name: "ni", msg: "comp", comp: "ni", cursor: -1, wantText: "ni", wantPreedit: "ni", wantSel: [2]int{2, 2}, wantEvents: 1},
			},
			finalText: "ni", finalCaret: 2,
		},
		{
			name: "a cursor position outside the string puts the caret at the end", text: "", caret: 0, to: 0,
			steps: []step{
				{name: "ni, position 9", msg: "comp", comp: "ni", cursor: 9, wantText: "ni", wantPreedit: "ni", wantSel: [2]int{2, 2}, wantEvents: 1},
			},
			finalText: "ni", finalCaret: 2,
		},
		{
			name: "characters of a key with no input method", text: "ab", caret: 1, to: 1,
			steps: []step{
				{name: "x", msg: "char", unit: 'x', wantText: "axb", wantSel: [2]int{2, 2}, wantEvents: 1},
				{name: "a surrogate pair: the high half waits", msg: "char", unit: 0xD842, wantText: "axb", wantEvents: 0},
				{name: "the low half", msg: "char", unit: 0xDFB7, wantText: "ax𠮷b", wantSel: [2]int{6, 6}, wantEvents: 1},
				{name: "a lone low surrogate is dropped", msg: "char", unit: 0xDFB7, wantText: "ax𠮷b", wantEvents: 0},
				{name: "a control character is dropped", msg: "char", unit: 0x08, wantText: "ax𠮷b", wantEvents: 0},
				{name: "DEL is dropped", msg: "char", unit: 0x7F, wantText: "ax𠮷b", wantEvents: 0},
				{name: "a high half then a letter drops the half", msg: "char", unit: 0xD842, wantText: "ax𠮷b", wantEvents: 0},
				{name: "y", msg: "char", unit: 'y', wantText: "ax𠮷yb", wantEvents: 1},
			},
			finalText: "ax𠮷yb", finalCaret: 7,
		},
		{
			name: "a typed character replaces the preedit and the selection", text: "ab", caret: 2, to: 2,
			steps: []step{
				{name: "ni", msg: "comp", comp: "ni", cursor: 2, wantText: "abni", wantPreedit: "ni", wantEvents: 1},
				{name: "z", msg: "char", unit: 'z', wantText: "abz", wantPreedit: "", wantSel: [2]int{3, 3}, wantEvents: 1},
			},
			finalText: "abz", finalCaret: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, imm := newField(t, tt.text, tt.caret, tt.to)
			for _, s := range tt.steps {
				s.run(t, f, imm)
			}
			if f.text != tt.finalText || f.ext != tt.finalCaret {
				t.Errorf("final text %q with the caret at %d, want %q at %d", f.text, f.ext, tt.finalText, tt.finalCaret)
			}
		})
	}
}

func TestCompositionCancelledByAnEmptyMessage(t *testing.T) {
	f, imm := newField(t, "ab", 2, 2)
	f.in.HandleStartComposition()
	imm.Comp, imm.Cursor = u16("ni"), 2
	f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
	if f.preedit != "ni" {
		t.Fatalf("preedit = %q", f.preedit)
	}
	// Escape: Windows sends WM_IME_COMPOSITION with lParam 0, which is not answered (DefWindowProc
	// has it), and the preedit goes.
	if handled := f.in.HandleComposition(0); handled {
		t.Error("a message with no flag was handled; DefWindowProc must have it")
	}
	if f.preedit != "" || f.text != "ab" {
		t.Errorf("text %q with preedit %q, want %q with none", f.text, f.preedit, "ab")
	}
	f.in.HandleEndComposition()
	if len(f.events) != 2 {
		t.Errorf("%d events, want 2 (the preedit and its removal): %+v", len(f.events), f.events)
	}
	if f.text != "ab" || f.in.Composing() {
		t.Errorf("after the end: text %q, composing %v", f.text, f.in.Composing())
	}
}

func TestCompositionEmptyStringRemovesPreedit(t *testing.T) {
	// Backspace on the last letter of a composition: the string is empty and the flags say COMPSTR.
	f, imm := newField(t, "", 0, 0)
	f.in.HandleStartComposition()
	imm.Comp, imm.Cursor = u16("n"), 1
	f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
	imm.Comp, imm.Cursor = nil, 0
	f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
	if f.preedit != "" || f.text != "" {
		t.Errorf("text %q with preedit %q, want both empty", f.text, f.preedit)
	}
}

func TestHandleCompositionAnswers(t *testing.T) {
	tests := []struct {
		name   string
		lparam uintptr
		want   bool
	}{
		{"the result string", win32.GCSResultStr, true},
		{"the composition string", win32.GCSCompStr, true},
		{"both", win32.GCSResultStr | win32.GCSCompStr | win32.GCSCursorPos, true},
		{"only the attributes", win32.GCSCompAttr, false},
		{"only the cursor", win32.GCSCursorPos, false},
		{"no flag", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, imm := newField(t, "", 0, 0)
			imm.Comp, imm.Result = u16("a"), u16("b")
			if got := f.in.HandleComposition(tt.lparam); got != tt.want {
				t.Errorf("handled = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessagesWithoutATextField(t *testing.T) {
	imm := imetest.New(testHWND)
	var events []ui.TextInputEvent
	in := New(imm, testHWND, func(e ui.TextInputEvent) { events = append(events, e) })
	imm.Comp, imm.Result = u16("ni"), u16("x")
	in.HandleStartComposition()
	in.HandleComposition(win32.GCSCompStr | win32.GCSResultStr)
	in.HandleChar('a')
	in.Commit("b")
	in.HandleEndComposition()
	if len(events) != 0 {
		t.Errorf("events with no text field: %+v", events)
	}
}

func TestInputMethodOnAndOffWithTheField(t *testing.T) {
	imm := imetest.New(testHWND)
	in := New(imm, testHWND, func(ui.TextInputEvent) {})
	if !reflect.DeepEqual(imm.Calls, []string{"associate 0"}) || in.IMEOn() {
		t.Fatalf("a new input: calls %v, on %v; want the input method off", imm.Calls, in.IMEOn())
	}
	imm.Calls = nil
	st := ui.TextInputState{Enabled: true, CaretRect: ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4}}
	in.SetState(st, 1)
	want := []string{"associate-ex 0 16", "create-caret 1x1"}
	if !reflect.DeepEqual(imm.Calls, want) || !in.IMEOn() || !in.Enabled() {
		t.Fatalf("field focused: calls %v, on %v; want %v", imm.Calls, in.IMEOn(), want)
	}
	imm.Calls = nil
	in.SetState(st, 1) // a state again changes nothing
	if len(imm.Calls) != 0 {
		t.Errorf("the same state made calls: %v", imm.Calls)
	}
	in.SetState(ui.TextInputState{}, 1)
	want = []string{"destroy-caret", "associate 0"}
	if !reflect.DeepEqual(imm.Calls, want) || in.IMEOn() || in.Enabled() {
		t.Errorf("field gone: calls %v, on %v; want %v", imm.Calls, in.IMEOn(), want)
	}
	imm.Calls = nil
	in.SetState(ui.TextInputState{}, 1)
	if len(imm.Calls) != 0 {
		t.Errorf("a second disabled state made calls: %v", imm.Calls)
	}
}

func TestLosingTheFieldAbortsTheComposition(t *testing.T) {
	f, imm := newField(t, "ab", 2, 2)
	f.in.HandleStartComposition()
	imm.Comp, imm.Cursor = u16("ni"), 2
	f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
	imm.Calls = nil
	f.in.SetState(ui.TextInputState{}, 1)
	want := []string{"notify 21 4 0", "notify 17 0 0", "clear", "destroy-caret", "associate 0"}
	if !reflect.DeepEqual(imm.Calls, want) {
		t.Errorf("calls = %v, want %v", imm.Calls, want)
	}
	if f.in.Composing() || f.in.Preedit() != "" {
		t.Errorf("composing %v, preedit %q after the field went", f.in.Composing(), f.in.Preedit())
	}
	// What the input method says after that is not typed anywhere.
	n := len(f.events)
	f.in.HandleEndComposition()
	if len(f.events) != n {
		t.Errorf("an event came after the field was gone: %+v", f.events[n:])
	}
}

func TestSetContextClearsTheCompositionWindowFlag(t *testing.T) {
	tests := []struct {
		name           string
		wparam, lparam uintptr
		want           uintptr
	}{
		{"activated, all windows", 1, win32.ISCShowUICompositionWindow | win32.ISCShowUIAllCandidateWindow, win32.ISCShowUIAllCandidateWindow},
		{"deactivated", 0, win32.ISCShowUICompositionWindow | 3, 3},
		{"the flag was not set", 1, 0x0F, 0x0F},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := newField(t, "", 0, 0)
			if got := f.in.HandleSetContext(tt.wparam, tt.lparam); got != tt.want {
				t.Errorf("lparam = %#x, want %#x", got, tt.want)
			}
		})
	}
}

// TestIMEWindowsFollowTheCaret checks the candidate window math (SPEC §25.7): the rectangle of the
// framework is in logical pixels, each edge times the scale and cut to a whole number, the
// composition window is at its top left, the candidate window excludes the rectangle and starts at
// its bottom left, and the system caret is at the bottom left.
func TestIMEWindowsFollowTheCaret(t *testing.T) {
	r := ui.Rect{Left: 10.5, Top: 20.25, Right: 12.5, Bottom: 40.75}
	tests := []struct {
		scale float64
		want  win32.Rect
	}{
		{1, win32.Rect{Left: 10, Top: 20, Right: 12, Bottom: 40}},
		{1.25, win32.Rect{Left: 13, Top: 25, Right: 15, Bottom: 50}},
		{1.5, win32.Rect{Left: 15, Top: 30, Right: 18, Bottom: 61}},
		{2, win32.Rect{Left: 21, Top: 40, Right: 25, Bottom: 81}},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(int(tt.scale*100)), func(t *testing.T) {
			imm := imetest.New(testHWND)
			in := New(imm, testHWND, func(ui.TextInputEvent) {})
			in.SetState(ui.TextInputState{Enabled: true, CaretRect: r}, tt.scale)
			if got := CaretRect(r, tt.scale); got != tt.want || in.CaretRect() != tt.want {
				t.Errorf("caret rectangle = %+v (%+v), want %+v", got, in.CaretRect(), tt.want)
			}
			wantComp := win32.CompositionForm{Style: win32.CFSPoint, CurrentPos: win32.Point{X: tt.want.Left, Y: tt.want.Top}}
			wantCand := win32.CandidateForm{Style: win32.CFSExclude, CurrentPos: win32.Point{X: tt.want.Left, Y: tt.want.Bottom}, Area: tt.want}
			if imm.CompForm == nil || *imm.CompForm != wantComp {
				t.Errorf("composition window = %+v, want %+v", imm.CompForm, wantComp)
			}
			if imm.CandForm == nil || *imm.CandForm != wantCand {
				t.Errorf("candidate window = %+v, want %+v", imm.CandForm, wantCand)
			}
			if imm.CaretPos != (win32.Point{X: tt.want.Left, Y: tt.want.Bottom}) {
				t.Errorf("system caret at %+v, want the bottom left %d,%d", imm.CaretPos, tt.want.Left, tt.want.Bottom)
			}
			if imm.Released != imm.Contexts-1 && imm.Released != imm.Contexts {
				t.Errorf("%d contexts got, %d released", imm.Contexts, imm.Released)
			}
		})
	}
}

func TestIMEWindowsMoveOnlyWhenTheRectangleChanges(t *testing.T) {
	imm := imetest.New(testHWND)
	in := New(imm, testHWND, func(ui.TextInputEvent) {})
	st := ui.TextInputState{Enabled: true, CaretRect: ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4}}
	in.SetState(st, 1)
	n := imm.Placements
	if n == 0 {
		t.Fatal("the windows were not placed when the field got the focus")
	}
	in.SetState(st, 1)
	if imm.Placements != n {
		t.Error("the same rectangle placed the windows again")
	}
	st.CaretRect = ui.Rect{Left: 5, Top: 6, Right: 7, Bottom: 8}
	in.SetState(st, 1)
	if imm.Placements != n+1 || imm.CandForm.CurrentPos != (win32.Point{X: 5, Y: 8}) {
		t.Errorf("a new rectangle: %d placements (want %d), candidate at %+v", imm.Placements, n+1, imm.CandForm)
	}
	in.SetState(st, 2) // the scale changed: the same rectangle is other pixels
	if imm.Placements != n+2 || imm.CandForm.CurrentPos != (win32.Point{X: 10, Y: 16}) {
		t.Errorf("a new scale: %d placements (want %d), candidate at %+v", imm.Placements, n+2, imm.CandForm)
	}
}

func TestIMEWindowsWaitForTheKeyboardFocus(t *testing.T) {
	imm := imetest.New(testHWND)
	imm.FocusWindow = 0x999 // another window has the focus
	in := New(imm, testHWND, func(ui.TextInputEvent) {})
	in.SetState(ui.TextInputState{Enabled: true, CaretRect: ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4}}, 1)
	if imm.Placements != 0 {
		t.Fatal("the windows were placed while another window has the focus")
	}
	imm.FocusWindow = testHWND
	in.FocusChanged()
	if imm.Placements != 1 || imm.CandForm == nil {
		t.Errorf("the focus came: %d placements", imm.Placements)
	}
	// No input context (the input method is off): nothing to place.
	imm.Off = true
	imm.Placements = 0
	in.SetState(ui.TextInputState{Enabled: true, CaretRect: ui.Rect{Left: 9, Top: 9, Right: 9, Bottom: 9}}, 1)
	if imm.Placements != 0 {
		t.Error("the windows were placed without an input context")
	}
}

// TestCompositionCrossesAStateChange checks what the framework's changes do to a composition that
// goes on: its echo changes nothing, and a change of the text or the caret by the app or the user
// moves where the next message of the input method lands, with the input method left alone.
func TestCompositionCrossesAStateChange(t *testing.T) {
	t.Run("the echo of the framework", func(t *testing.T) {
		f, imm := newField(t, "ab", 2, 2)
		f.in.HandleStartComposition()
		imm.Comp, imm.Cursor = u16("ni"), 2
		f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
		// The framework sent its state after the event (f.send in handle) and again now.
		f.send()
		f.send()
		imm.Comp, imm.Cursor = u16("nih"), 3
		f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
		if f.text != "abnih" || f.preedit != "nih" || len(f.events) != 2 {
			t.Errorf("text %q, preedit %q, %d events", f.text, f.preedit, len(f.events))
		}
	})
	t.Run("the app changes the text", func(t *testing.T) {
		f, imm := newField(t, "ab", 2, 2)
		f.in.HandleStartComposition()
		imm.Comp, imm.Cursor = u16("ni"), 2
		f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
		// The app sets another text (a formatter, a controller) and the preedit is gone with it.
		f.text, f.anchor, f.ext, f.preedit = "xyz", 3, 3, ""
		f.send()
		if !f.in.Composing() {
			t.Fatal("the state change ended the composition of the input method")
		}
		imm.Comp, imm.Cursor = u16("nih"), 3
		f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
		if f.text != "xyznih" || f.preedit != "nih" || f.ext != 6 {
			t.Errorf("text %q, preedit %q, caret %d; want xyznih with nih composing, caret 6", f.text, f.preedit, f.ext)
		}
		imm.Comp, imm.Result, imm.Cursor = nil, u16("你好"), 0
		f.in.HandleComposition(win32.GCSResultStr)
		f.in.HandleEndComposition()
		if f.text != "xyz你好" || f.preedit != "" {
			t.Errorf("after the result: text %q, preedit %q", f.text, f.preedit)
		}
	})
	t.Run("the user moves the caret", func(t *testing.T) {
		f, imm := newField(t, "abcd", 4, 4)
		f.in.HandleStartComposition()
		imm.Comp, imm.Cursor = u16("ni"), 2
		f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
		// A click puts the caret at 1: the framework's state has the text without the preedit.
		f.text, f.anchor, f.ext, f.preedit = "abcd", 1, 1, ""
		f.send()
		imm.Comp, imm.Cursor = u16("nih"), 3
		f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
		if f.text != "anihbcd" || f.preedit != "nih" {
			t.Errorf("text %q, preedit %q; want anihbcd with nih composing", f.text, f.preedit)
		}
	})
}

func TestUTF16Helpers(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{{"", 0}, {"abc", 3}, {"é", 1}, {"é", 2}, {"漢字", 2}, {"𠮷", 2}, {"a𠮷b", 4}, {"😀", 2}}
	for _, tt := range tests {
		if got := utf16Len(tt.s); got != tt.want {
			t.Errorf("utf16Len(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
	if got := utf16ToString(u16("a𠮷b")); got != "a𠮷b" {
		t.Errorf("utf16ToString = %q", got)
	}
	if got := utf16ToString([]uint16{'a', 0xD842}); got != "a�" {
		t.Errorf("a lone surrogate = %q, want U+FFFD", got)
	}
	if got := clampByte("a𠮷", 3); got != 1 {
		t.Errorf("clampByte in the middle of a character = %d, want 1", got)
	}
	if !slices.Equal(u16("𠮷"), []uint16{0xD842, 0xDFB7}) {
		t.Error("u16 of 𠮷 is not the pair D842 DFB7")
	}
}

// TestWindowUnittestsOnImeComposition has the cases of window_unittests.cc that concern WM_IME_COMPOSITION
// (OnImeCompositionCompose, OnImeCompositionDefaultsKoreanCursorToCompositionEnd, OnImeCompositionResult,
// OnImeCompositionResultAndCompose, OnImeCompositionUsesCursorPositionWhenProvided,
// OnImeCompositionFallsBackToTextEndForInvalidCursorPosition and OnImeCompositionClearChange): the same
// strings and flags, with the event of the input in place of the calls of OnComposeChange and
// OnComposeCommit (a cursor of Flutter is a number of UTF-16 units, here bytes of the preedit).
func TestWindowUnittestsOnImeComposition(t *testing.T) {
	tests := []struct {
		name           string
		comp, result   string
		cursor         int32
		lparam         uintptr
		want           []ui.TextInputEvent
		wantValueCalls int
	}{
		{"compose", "nihao", "`}", 0, win32.GCSCompStr,
			[]ui.TextInputEvent{{Preedit: "nihao", PreeditCursorBegin: 5, PreeditCursorEnd: 5}}, 0},
		{"Korean: the cursor defaults to the end of the string", "다", "", 0, win32.GCSCompStr,
			[]ui.TextInputEvent{{Preedit: "다", PreeditCursorBegin: 3, PreeditCursorEnd: 3}}, 0},
		{"result", "nihao", "`}", 0, win32.GCSResultStr,
			[]ui.TextInputEvent{{Commit: "`}", PreeditCursorBegin: -1, PreeditCursorEnd: -1}}, 0},
		{"result and compose in one message (Google Japanese Input)", "は", "今日", 0, win32.GCSCompStr | win32.GCSResultStr,
			[]ui.TextInputEvent{{Commit: "今日", Preedit: "は", PreeditCursorBegin: 3, PreeditCursorEnd: 3}}, 0},
		{"the cursor position is used when provided", "nihao", "", 2, win32.GCSCompStr | win32.GCSCursorPos,
			[]ui.TextInputEvent{{Preedit: "nihao", PreeditCursorBegin: 2, PreeditCursorEnd: 2}}, 1},
		{"an invalid cursor position falls back to the end of the text", "nihao", "", -1, win32.GCSCompStr | win32.GCSCursorPos,
			[]ui.TextInputEvent{{Preedit: "nihao", PreeditCursorBegin: 5, PreeditCursorEnd: 5}}, 1},
		{"a message with no flag changes nothing when nothing is composing", "", "", 0, 0, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imm := imetest.New(testHWND)
			var got []ui.TextInputEvent
			in := New(imm, testHWND, func(e ui.TextInputEvent) { got = append(got, e) })
			in.SetState(ui.TextInputState{Enabled: true}, 1)
			imm.Comp, imm.Result, imm.Cursor = u16(tt.comp), u16(tt.result), tt.cursor
			in.HandleComposition(tt.lparam)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("events = %+v, want %+v", got, tt.want)
			}
			if imm.ValueCalls != tt.wantValueCalls {
				t.Errorf("the caret was read %d times, want %d", imm.ValueCalls, tt.wantValueCalls)
			}
		})
	}
}

// TestCompositionCursorPosPortedFromFlutter is CompositionCursorPos of text_input_plugin_unittest.cc: the
// selection of the field follows the caret of the composition string, in a field that already has text.
func TestCompositionCursorPosPortedFromFlutter(t *testing.T) {
	f, imm := newField(t, "", 0, 0)
	comp := func(s string, cursor int32) {
		imm.Comp, imm.Cursor = u16(s), cursor
		f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
	}
	f.in.HandleStartComposition()
	if f.ext != 0 {
		t.Errorf("after the start the caret is at %d, want 0", f.ext)
	}
	comp("abc", 3)
	if f.ext != 3 {
		t.Errorf("abc: the caret is at %d, want 3", f.ext)
	}
	imm.Comp, imm.Result = nil, u16("abc")
	f.in.HandleComposition(win32.GCSResultStr)
	f.in.HandleEndComposition()
	if f.ext != 3 || f.text != "abc" || f.preedit != "" {
		t.Errorf("after the end: text %q, caret %d, preedit %q; want abc, 3, none", f.text, f.ext, f.preedit)
	}
	f.in.HandleStartComposition()
	for _, s := range []struct {
		text   string
		cursor int32
		want   int
	}{{"1", 1, 4}, {"12", 2, 5}, {"12", 1, 4}, {"12", 2, 5}} {
		comp(s.text, s.cursor)
		if f.ext != s.want {
			t.Errorf("%q with the caret at %d: the field's caret is at %d, want %d", s.text, s.cursor, f.ext, s.want)
		}
	}
}

// TestEventsOfAComposition is VerifyComposingSendStateUpdate of text_input_plugin_unittest.cc: the start sends
// nothing here (the framework needs no state to begin), a change sends an event, the commit and the end
// together send one, and a commit that is followed by a change sends it with the change.
func TestEventsOfAComposition(t *testing.T) {
	f, imm := newField(t, "", 0, 0)
	count := func() int { return len(f.events) }
	f.in.HandleStartComposition()
	if count() != 0 {
		t.Errorf("the start sent %d events", count())
	}
	imm.Comp, imm.Cursor = u16("4"), 1
	f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
	if count() != 1 {
		t.Errorf("the change sent %d events, want 1", count())
	}
	imm.Comp, imm.Result = nil, u16("4")
	f.in.HandleComposition(win32.GCSResultStr)
	if count() != 2 || f.events[1].Commit != "4" || f.events[1].Preedit != "" {
		t.Errorf("the commit: events %+v", f.events)
	}
	f.in.HandleEndComposition()
	if count() != 2 {
		t.Errorf("the end after the commit sent an event: %+v", f.events[2:])
	}
	// The same on another field: the state starts again.
	f.text, f.anchor, f.ext = "other", 5, 5
	f.in.SetState(ui.TextInputState{}, 1)
	f.send()
	f.in.HandleStartComposition()
	imm.Comp, imm.Cursor = u16("4"), 1
	f.in.HandleComposition(win32.GCSCompStr | win32.GCSCursorPos)
	if f.text != "other4" || f.preedit != "4" {
		t.Errorf("on another field: text %q, preedit %q", f.text, f.preedit)
	}
}

var _ IMM = (*imetest.Fake)(nil)
