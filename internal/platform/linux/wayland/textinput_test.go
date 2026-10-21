//go:build linux

package wayland

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// textEnv is a window with a text-input focus in a fake compositor that offers
// zwp_text_input_manager_v3.
type textEnv struct {
	*keyEnv
	seenTexts int
	seenReq   int
}

func startTextInput(t *testing.T, focus bool) *textEnv {
	t.Helper()
	e := &textEnv{keyEnv: startKeyboardNoFocus(t, fakecomp.Config{Globals: fakecomp.TextInputGlobals()})}
	if focus {
		e.enter()
	}
	return e
}

func (e *textEnv) enter() {
	e.t.Helper()
	e.srv.KeyboardEnter(e.surface)
	e.srv.TextInputEnter(e.surface)
	e.settle()
}

// set is Window.SetTextInput on the UI thread.
func (e *textEnv) set(s ui.TextInputState) {
	e.t.Helper()
	e.onLoop(func() bool { e.p.windows[ui.ImplicitViewID].SetTextInput(s); return true })
	e.settle()
}

// requests returns the text input requests since the last call.
func (e *textEnv) requests() []string {
	e.t.Helper()
	e.settle()
	var out []string
	for _, r := range e.srv.Requests()[e.seenReq:] {
		if strings.HasPrefix(r, "zwp_text_input_v3.") {
			out = append(out, strings.TrimPrefix(r, "zwp_text_input_v3."))
		}
	}
	e.seenReq = len(e.srv.Requests())
	return out
}

func (e *textEnv) texts() []ui.TextInputEvent {
	e.t.Helper()
	e.settle()
	all := e.h.textEvents()
	out := all[e.seenTexts:]
	e.seenTexts = len(all)
	return out
}

func field(text string, cursor int) ui.TextInputState {
	return ui.TextInputState{
		Enabled: true, Text: text, SelectionAnchor: cursor, SelectionCursor: cursor,
		CaretRect: ui.Rect{Left: 10.2, Top: 20, Right: 11.4, Bottom: 34.5}, CausedByInputMethod: true,
	}
}

func TestTextInputEnablesWithTheFocusAndSendsTheState(t *testing.T) {
	e := startTextInput(t, true)
	e.requests()
	state := field("héllo", 3)
	state.Purpose = ui.TextInputPurposeEmail
	state.Hints = ui.TextInputHintSpellcheck | ui.TextInputHintMultiline
	e.set(state)
	want := []string{
		"enable",
		`set_surrounding_text text="héllo" cursor=3 anchor=3`,
		"set_content_type hint=0x1202 purpose=6", // spellcheck 0x2, multiline 0x200, preedit shown 0x1000
		"set_cursor_rectangle x=10 y=20 width=2 height=15",
		"commit serial=1",
	}
	if got := e.requests(); !reflect.DeepEqual(got, want) {
		t.Errorf("requests = %q\nwant %q", got, want)
	}
}

func TestTextInputWaitsForTheFocus(t *testing.T) {
	e := startTextInput(t, false)
	e.set(field("abc", 3))
	if got := e.requests(); len(got) != 0 {
		t.Fatalf("requests before the enter = %q, want none", got)
	}
	e.enter()
	got := e.requests()
	if len(got) == 0 || got[0] != "enable" || got[len(got)-1] != "commit serial=1" {
		t.Errorf("requests after the enter = %q, want enable ... commit", got)
	}
}

func TestTextInputSendsOnlyWhatChanged(t *testing.T) {
	e := startTextInput(t, true)
	e.set(field("abc", 3))
	e.requests()

	t.Run("caret moves", func(t *testing.T) {
		s := field("abc", 3)
		s.CaretRect = ui.Rect{Left: 30, Top: 20, Right: 31, Bottom: 35}
		e.set(s)
		want := []string{"set_cursor_rectangle x=30 y=20 width=1 height=15", "commit serial=2"}
		if got := e.requests(); !reflect.DeepEqual(got, want) {
			t.Errorf("requests = %q, want %q", got, want)
		}
	})
	t.Run("nothing changes", func(t *testing.T) {
		s := field("abc", 3)
		s.CaretRect = ui.Rect{Left: 30, Top: 20, Right: 31, Bottom: 35}
		e.set(s)
		if got := e.requests(); len(got) != 0 {
			t.Errorf("requests = %q, want none", got)
		}
	})
	t.Run("text changes by the input method", func(t *testing.T) {
		s := field("abcd", 4)
		s.CaretRect = ui.Rect{Left: 30, Top: 20, Right: 31, Bottom: 35}
		e.set(s)
		want := []string{`set_surrounding_text text="abcd" cursor=4 anchor=4`, "commit serial=3"}
		if got := e.requests(); !reflect.DeepEqual(got, want) {
			t.Errorf("requests = %q, want %q", got, want)
		}
	})
	t.Run("text changes by the user", func(t *testing.T) {
		s := field("abcde", 5)
		s.CaretRect = ui.Rect{Left: 30, Top: 20, Right: 31, Bottom: 35}
		s.CausedByInputMethod = false
		e.set(s)
		want := []string{
			`set_surrounding_text text="abcde" cursor=5 anchor=5`,
			"set_text_change_cause cause=1",
			"commit serial=4",
		}
		if got := e.requests(); !reflect.DeepEqual(got, want) {
			t.Errorf("requests = %q, want %q", got, want)
		}
	})
	t.Run("selection", func(t *testing.T) {
		s := field("abcde", 5)
		s.SelectionAnchor = 1
		s.CaretRect = ui.Rect{Left: 30, Top: 20, Right: 31, Bottom: 35}
		e.set(s)
		want := []string{`set_surrounding_text text="abcde" cursor=5 anchor=1`, "commit serial=5"}
		if got := e.requests(); !reflect.DeepEqual(got, want) {
			t.Errorf("requests = %q, want %q", got, want)
		}
	})
}

func TestTextInputDisables(t *testing.T) {
	e := startTextInput(t, true)
	e.set(field("abc", 3))
	e.requests()
	e.set(ui.TextInputState{})
	if got, want := e.requests(), []string{"disable", "commit serial=2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("requests = %q, want %q", got, want)
	}
	e.set(ui.TextInputState{})
	if got := e.requests(); len(got) != 0 {
		t.Errorf("a second disable sent %q", got)
	}
	// And enables again, from scratch.
	e.set(field("abc", 3))
	got := e.requests()
	if len(got) == 0 || got[0] != "enable" {
		t.Errorf("requests = %q, want enable first", got)
	}
}

func TestTextInputLeaveAndEnterAgain(t *testing.T) {
	e := startTextInput(t, true)
	e.set(field("abc", 3))
	e.requests()
	e.srv.TextInputLeave(e.surface)
	e.settle()
	// While the window has no focus, changes are kept and not sent.
	e.set(field("abcd", 4))
	if got := e.requests(); len(got) != 0 {
		t.Fatalf("requests while unfocused = %q", got)
	}
	e.srv.TextInputEnter(e.surface)
	got := e.requests()
	if len(got) == 0 || got[0] != "enable" || !strings.Contains(strings.Join(got, "\n"), `text="abcd" cursor=4`) {
		t.Errorf("requests after the enter = %q, want enable with the latest text", got)
	}
}

func TestTextInputEventsAreAppliedAtDone(t *testing.T) {
	e := startTextInput(t, true)
	e.set(field("abc", 3))
	e.texts()

	e.srv.TextInputPreedit("ni", 1, 2)
	e.settle()
	if got := e.texts(); len(got) != 0 {
		t.Fatalf("events before done = %+v", got)
	}
	e.srv.TextInputDone()
	got := e.texts()
	want := []ui.TextInputEvent{{View: ui.ImplicitViewID, Preedit: "ni", PreeditCursorBegin: 1, PreeditCursorEnd: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %+v, want %+v", got, want)
	}

	e.srv.TextInputPreedit("", 0, 0)
	e.srv.TextInputCommit("你")
	e.srv.TextInputDelete(1, 2)
	e.srv.TextInputDone()
	got = e.texts()
	want = []ui.TextInputEvent{{View: ui.ImplicitViewID, Commit: "你", DeleteBefore: 1, DeleteAfter: 2, PreeditCursorBegin: -1, PreeditCursorEnd: -1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v, want %+v", got, want)
	}

	// The changes are reset by done.
	e.srv.TextInputDone()
	if got := e.texts(); len(got) != 0 {
		t.Errorf("an empty done sent %+v", got)
	}
}

func TestTextInputDoneThatRemovesAPreeditIsDelivered(t *testing.T) {
	e := startTextInput(t, true)
	e.set(field("abc", 3))
	e.srv.TextInputPreedit("ni", 0, 2)
	e.srv.TextInputDone()
	e.texts()
	e.srv.TextInputDone() // no preedit_string: the preedit goes
	got := e.texts()
	want := []ui.TextInputEvent{{View: ui.ImplicitViewID, PreeditCursorBegin: -1, PreeditCursorEnd: -1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v, want %+v", got, want)
	}
}

func TestTextInputLeaveRemovesThePreedit(t *testing.T) {
	e := startTextInput(t, true)
	e.set(field("abc", 3))
	e.srv.TextInputPreedit("ni", 0, 2)
	e.srv.TextInputDone()
	e.texts()
	e.srv.TextInputLeave(e.surface)
	got := e.texts()
	want := []ui.TextInputEvent{{PreeditCursorBegin: -1, PreeditCursorEnd: -1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v, want %+v", got, want)
	}
}

func TestTextInputStaleDoneIsStillApplied(t *testing.T) {
	e := startTextInput(t, true)
	e.set(field("abc", 3))
	e.texts()
	e.srv.TextInputCommit("x")
	e.srv.TextInputDoneSerial(0) // older than the commit the app made
	got := e.texts()
	if len(got) != 1 || got[0].Commit != "x" {
		t.Errorf("events = %+v, want the commit applied", got)
	}
}

func TestTextInputWithoutTheManager(t *testing.T) {
	e := startKeyboardNoFocus(t, fakecomp.Config{})
	if e.p.text != nil {
		t.Fatal("a text input exists without the manager")
	}
	// The input method over D-Bus takes over; keep this test off the real bus.
	e.onLoop(func() bool { e.p.dbusIME.close(); e.p.dbusIME = nil; return true })
	e.srv.KeyboardEnter(e.surface)
	te := &textEnv{keyEnv: e}
	te.set(field("abc", 3)) // must do nothing
	if got := te.requests(); len(got) != 0 {
		t.Errorf("requests = %q", got)
	}
}

func TestTextInputOfAnotherWindowIsNotEnabled(t *testing.T) {
	e := startTextInput(t, true)
	e.set(field("abc", 3))
	e.requests()
	e.onLoop(func() bool {
		w, err := e.p.NewWindow(config640())
		if err != nil {
			t.Error(err)
			return false
		}
		w.SetTextInput(field("other", 5))
		return true
	})
	// The second window has no text-input focus, so nothing is sent for it and
	// the first window's text field is turned off.
	got := e.requests()
	if len(got) == 0 || got[0] != "disable" {
		t.Errorf("requests = %q, want the first field disabled", got)
	}
}

func TestClosingTheWindowForgetsTheTextInput(t *testing.T) {
	e := startTextInput(t, true)
	e.set(field("abc", 3))
	e.onLoop(func() bool { e.p.windows[ui.ImplicitViewID].Close(); return true })
	if e.p.text.focus != nil || e.p.text.wantWindow != nil {
		t.Errorf("the text input still holds the window: focus=%v want=%v", e.p.text.focus, e.p.text.wantWindow)
	}
}

func TestClipSurrounding(t *testing.T) {
	long := strings.Repeat("a", 5000)
	tests := []struct {
		name           string
		text           string
		cursor, anchor int
		wantLen        int
		check          func(t *testing.T, text string, cursor, anchor int)
	}{
		{"short", "héllo", 3, 3, len("héllo"), nil},
		{"cursor out of range", "abc", 99, -4, 3, func(t *testing.T, _ string, c, a int) {
			if c != 3 || a != 0 {
				t.Errorf("cursor=%d anchor=%d", c, a)
			}
		}},
		{"long, cursor in the middle", long, 2500, 2500, 4000, func(t *testing.T, _ string, c, a int) {
			if c != 2000 || a != 2000 {
				t.Errorf("cursor=%d anchor=%d, want 2000", c, a)
			}
		}},
		{"long, cursor at the start", long, 0, 0, 4000, func(t *testing.T, _ string, c, _ int) {
			if c != 0 {
				t.Errorf("cursor=%d", c)
			}
		}},
		{"long, cursor at the end", long, 5000, 5000, 4000, func(t *testing.T, _ string, c, _ int) {
			if c != 4000 {
				t.Errorf("cursor=%d", c)
			}
		}},
		{"long, selection kept", long, 3000, 2000, 4000, func(t *testing.T, _ string, c, a int) {
			if c-a != 1000 {
				t.Errorf("the selection is %d..%d, want a length of 1000", a, c)
			}
		}},
		{"multibyte characters are not split", strings.Repeat("é", 3000), 3000, 3000, -1, func(t *testing.T, text string, _, _ int) {
			if !isValidUTF8(text) || len(text) > 4000 {
				t.Errorf("clipped to %d bytes, valid=%v", len(text), isValidUTF8(text))
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text, c, a := clipSurrounding(tc.text, tc.cursor, tc.anchor)
			if tc.wantLen >= 0 && len(text) != tc.wantLen {
				t.Errorf("len = %d, want %d", len(text), tc.wantLen)
			}
			if tc.check != nil {
				tc.check(t, text, c, a)
			}
		})
	}
}

func isValidUTF8(s string) bool { return strings.ToValidUTF8(s, "\x00") == s }

func TestCaretRectangle(t *testing.T) {
	tests := []struct {
		name       string
		rect       ui.Rect
		x, y, w, h int32
	}{
		{"whole", ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 6}, 1, 2, 2, 4},
		{"fractions grow to cover the caret", ui.Rect{Left: 1.5, Top: 2.5, Right: 2.5, Bottom: 6.1}, 1, 2, 2, 5},
		{"negative", ui.Rect{Left: -3.5, Top: -1, Right: -2, Bottom: 0}, -4, -1, 2, 1},
		{"not finite", ui.Rect{Left: 1, Top: 2, Right: math.Inf(1), Bottom: 6}, 0, 0, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			x, y, w, h := caretRectangle(tc.rect)
			if x != tc.x || y != tc.y || w != tc.w || h != tc.h {
				t.Errorf("caretRectangle = %d,%d,%d,%d want %d,%d,%d,%d", x, y, w, h, tc.x, tc.y, tc.w, tc.h)
			}
		})
	}
}

func TestContentTypeMapping(t *testing.T) {
	purposes := map[ui.TextInputPurpose]wl.ZwpTextInputV3ContentPurpose{
		ui.TextInputPurposeUnspecified: wl.ZwpTextInputV3ContentPurposeNormal,
		ui.TextInputPurposeAlpha:       wl.ZwpTextInputV3ContentPurposeAlpha,
		ui.TextInputPurposeDigits:      wl.ZwpTextInputV3ContentPurposeDigits,
		ui.TextInputPurposeNumber:      wl.ZwpTextInputV3ContentPurposeNumber,
		ui.TextInputPurposePhone:       wl.ZwpTextInputV3ContentPurposePhone,
		ui.TextInputPurposeURL:         wl.ZwpTextInputV3ContentPurposeUrl,
		ui.TextInputPurposeEmail:       wl.ZwpTextInputV3ContentPurposeEmail,
		ui.TextInputPurposeName:        wl.ZwpTextInputV3ContentPurposeName,
		ui.TextInputPurposePassword:    wl.ZwpTextInputV3ContentPurposePassword,
		ui.TextInputPurposePIN:         wl.ZwpTextInputV3ContentPurposePin,
		ui.TextInputPurposeDate:        wl.ZwpTextInputV3ContentPurposeDate,
		ui.TextInputPurposeTime:        wl.ZwpTextInputV3ContentPurposeTime,
		ui.TextInputPurposeDatetime:    wl.ZwpTextInputV3ContentPurposeDatetime,
	}
	for in, want := range purposes {
		if got := contentPurpose(in); got != want {
			t.Errorf("contentPurpose(%v) = %v, want %v", in, got, want)
		}
	}
	hints := map[ui.TextInputHints]wl.ZwpTextInputV3ContentHint{
		0:                                  wl.ZwpTextInputV3ContentHintPreeditShown,
		ui.TextInputHintCompletion:         wl.ZwpTextInputV3ContentHintCompletion,
		ui.TextInputHintSpellcheck:         wl.ZwpTextInputV3ContentHintSpellcheck,
		ui.TextInputHintAutoCapitalization: wl.ZwpTextInputV3ContentHintAutoCapitalization,
		ui.TextInputHintLowercase:          wl.ZwpTextInputV3ContentHintLowercase,
		ui.TextInputHintUppercase:          wl.ZwpTextInputV3ContentHintUppercase,
		ui.TextInputHintTitlecase:          wl.ZwpTextInputV3ContentHintTitlecase,
		ui.TextInputHintHiddenText:         wl.ZwpTextInputV3ContentHintHiddenText,
		ui.TextInputHintSensitiveData:      wl.ZwpTextInputV3ContentHintSensitiveData,
		ui.TextInputHintLatin:              wl.ZwpTextInputV3ContentHintLatin,
		ui.TextInputHintMultiline:          wl.ZwpTextInputV3ContentHintMultiline,
		ui.TextInputHintCompletion | ui.TextInputHintLatin: wl.ZwpTextInputV3ContentHintCompletion | wl.ZwpTextInputV3ContentHintLatin,
	}
	for in, flag := range hints {
		want := flag | wl.ZwpTextInputV3ContentHintPreeditShown
		if got := contentHint(in); got != want {
			t.Errorf("contentHint(%#x) = %#x, want %#x", uint32(in), uint32(got), uint32(want))
		}
	}
}
