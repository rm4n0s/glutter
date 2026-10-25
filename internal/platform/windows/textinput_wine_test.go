//go:build windows && glutter_wine

package windows

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests run the text input of a window under Wine (SPEC §25.7, §25.13): the input method is
// on while a text field has the focus, the windows of the input method follow the caret, and the
// composition messages, the scripted text and Wine's own input method reach the framework as
// ui.TextInputEvents. Real input methods (Microsoft IME, Pinyin, Korean) need a manual check.

// textField is the state of a focused text field with the caret rectangle given.
func textField(r ui.Rect) ui.TextInputState {
	return ui.TextInputState{Enabled: true, CaretRect: r}
}

// ev is a text event of the tests with the cursor of the preedit.
func commit(s string) ui.TextInputEvent {
	return ui.TextInputEvent{Commit: s, PreeditCursorBegin: -1, PreeditCursorEnd: -1}
}

func preedit(s string) ui.TextInputEvent {
	return ui.TextInputEvent{Preedit: s, PreeditCursorBegin: len(s), PreeditCursorEnd: len(s)}
}

// dev runs scripted steps (text:, compose:, composing:) on the first window.
func dev(scripts ...string) func(*window, *keyRec) {
	return func(w *window, _ *keyRec) {
		for _, script := range scripts {
			events, err := ParseDevEvents(script)
			if err != nil {
				panic(err)
			}
			for _, e := range events {
				switch e.Kind {
				case "text":
					w.p.runDevText(e)
				default:
					w.p.runDevCompose(e)
				}
			}
		}
	}
}

func setField(state ui.TextInputState) func(*window, *keyRec) {
	return func(w *window, _ *keyRec) { w.SetTextInput(state) }
}

func TestWineInputMethodFollowsTheTextField(t *testing.T) {
	var states []string
	record := func(label string) func(*window, *keyRec) {
		return func(w *window, _ *keyRec) {
			ctx := win32.ImmGetContext(w.hwnd)
			if ctx != 0 {
				win32.ImmReleaseContext(w.hwnd, ctx)
			}
			if (ctx != 0) != w.text.IMEOn() {
				t.Errorf("%s: ImmGetContext = %#x but the input says on = %v", label, ctx, w.text.IMEOn())
			}
			states = append(states, label+map[bool]string{true: " on", false: " off"}[ctx != 0])
		}
	}
	keyRun(t, false, platform.WindowConfig{},
		record("no field"),
		setField(textField(ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4})), record("field"),
		setField(ui.TextInputState{}), record("field gone"),
		setField(textField(ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4})), record("field again"),
	)
	want := []string{"no field off", "field on", "field gone off", "field again on"}
	if !reflect.DeepEqual(states, want) {
		t.Errorf("the input context of the window: %v, want %v", states, want)
	}
}

func TestWineCandidateWindowIsAtTheCaretRectangle(t *testing.T) {
	type placed struct {
		cand  win32.CandidateForm
		comp  win32.CompositionForm
		caret win32.Point
	}
	var got []placed
	read := func(w *window, _ *keyRec) {
		ctx := win32.ImmGetContext(w.hwnd)
		defer win32.ImmReleaseContext(w.hwnd, ctx)
		cand, ok1 := win32.ImmGetCandidateWindow(ctx, 0)
		comp, ok2 := win32.ImmGetCompositionWindow(ctx)
		caret, ok3 := win32.GetCaretPos()
		if !ok1 || !ok2 || !ok3 {
			t.Errorf("reading the places failed: candidate %v, composition %v, caret %v", ok1, ok2, ok3)
		}
		got = append(got, placed{cand, comp, caret})
	}
	// The window opens at scale 1 under Wine, so the rectangle is in pixels as it is.
	r1 := ui.Rect{Left: 10, Top: 20, Right: 12, Bottom: 40}
	r2 := ui.Rect{Left: 100, Top: 60, Right: 102, Bottom: 80}
	keyRun(t, false, platform.WindowConfig{},
		setField(textField(r1)), read,
		setField(textField(r2)), read,
	)
	want := func(r ui.Rect) placed {
		rect := win32.Rect{Left: int32(r.Left), Top: int32(r.Top), Right: int32(r.Right), Bottom: int32(r.Bottom)}
		return placed{
			cand:  win32.CandidateForm{Index: 0, Style: win32.CFSExclude, CurrentPos: win32.Point{X: rect.Left, Y: rect.Bottom}, Area: rect},
			comp:  win32.CompositionForm{Style: win32.CFSPoint, CurrentPos: win32.Point{X: rect.Left, Y: rect.Top}},
			caret: win32.Point{X: rect.Left, Y: rect.Bottom},
		}
	}
	if len(got) != 2 || got[0] != want(r1) || got[1] != want(r2) {
		t.Errorf("places = %+v\nwant %+v and %+v", got, want(r1), want(r2))
	}
}

func TestWineScriptedTextAndComposition(t *testing.T) {
	field := setField(textField(ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4}))
	tests := []struct {
		name  string
		steps []func(*window, *keyRec)
		want  []ui.TextInputEvent
	}{
		{"text arrives as one commit for each character", []func(*window, *keyRec){field, dev("text:Hi")},
			[]ui.TextInputEvent{commit("H"), commit("i")}},
		{"a surrogate pair is one commit", []func(*window, *keyRec){field, dev("text:𠮷")},
			[]ui.TextInputEvent{commit("𠮷")}},
		{"text types nothing while no field has the focus", []func(*window, *keyRec){dev("text:abc")}, nil},
		{"a composition shows each string and commits the last", []func(*window, *keyRec){field, dev("compose:k,か,漢字")},
			[]ui.TextInputEvent{preedit("k"), preedit("か"), preedit("漢字"), commit("漢字")}},
		{"a composition in progress shows the preedit and stays", []func(*window, *keyRec){field, dev("composing:n,ni")},
			[]ui.TextInputEvent{preedit("n"), preedit("ni")}},
		{"a composition in progress goes on and is committed later", []func(*window, *keyRec){field, dev("composing:n,ni"), dev("compose:ni,你")},
			[]ui.TextInputEvent{preedit("n"), preedit("ni"), preedit("你"), commit("你")}},
		{"a composition with no field makes nothing", []func(*window, *keyRec){dev("compose:ni,你")}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, k, _ := keyRun(t, false, platform.WindowConfig{}, tt.steps...)
			if !reflect.DeepEqual(k.texts, tt.want) {
				t.Errorf("events:\n got  %+v\n want %+v", k.texts, tt.want)
			}
		})
	}
}

// TestWineOwnInputMethodComposes uses Wine's own input method (the default IME of imm32): once the
// IME is open, ImmSetCompositionStringW makes it send the composition messages, and ImmNotifyIME
// completes or cancels the composition. Nothing is scripted: the strings come from the input
// context, as a real input method's do.
func TestWineOwnInputMethodComposes(t *testing.T) {
	compose := func(s string) func(*window, *keyRec) {
		return func(w *window, _ *keyRec) {
			ctx := win32.ImmGetContext(w.hwnd)
			if ctx == 0 {
				return // the input method is off for the window: there is nothing to compose in
			}
			defer win32.ImmReleaseContext(w.hwnd, ctx)
			win32.ImmSetOpenStatus(ctx, true)
			u := make([]uint16, 0, len(s))
			for _, r := range s {
				u = append(u, uint16(r))
			}
			if !win32.ImmSetCompositionString(ctx, win32.SCSSetStr, u, u) {
				t.Errorf("ImmSetCompositionStringW(%q) failed", s)
			}
		}
	}
	complete := func(w *window, _ *keyRec) {
		ctx := win32.ImmGetContext(w.hwnd)
		defer win32.ImmReleaseContext(w.hwnd, ctx)
		win32.ImmNotifyIME(ctx, win32.NICompositionStr, win32.CPSComplete, 0)
	}
	field := setField(textField(ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4}))
	t.Run("complete", func(t *testing.T) {
		_, k, w := keyRun(t, false, platform.WindowConfig{}, field, compose("ni"), compose("nihao"), complete)
		want := []ui.TextInputEvent{preedit("ni"), preedit("nihao"), commit("nihao")}
		if !reflect.DeepEqual(k.texts, want) {
			t.Errorf("events:\n got  %+v\n want %+v", k.texts, want)
		}
		if w.text.Composing() || w.text.Preedit() != "" {
			t.Errorf("composing %v, preedit %q after the composition was completed", w.text.Composing(), w.text.Preedit())
		}
	})
	// Wine's default IME commits the string on ImmNotifyIME(CPS_CANCEL), as it does on CPS_COMPLETE,
	// where Windows' input methods cancel: the cancelled composition is tested over the fake
	// (TestCompositionCancelledByAnEmptyMessage); an emptied string is the same message.
	t.Run("an emptied string removes the preedit", func(t *testing.T) {
		_, k, w := keyRun(t, false, platform.WindowConfig{}, field, compose("ni"), compose(""))
		if len(k.texts) < 2 || !reflect.DeepEqual(k.texts[0], preedit("ni")) {
			t.Fatalf("events: %+v, want the preedit first and its removal", k.texts)
		}
		for _, e := range k.texts {
			if e.Commit != "" {
				t.Errorf("emptying the string committed %q: %+v", e.Commit, k.texts)
			}
		}
		if w.text.Preedit() != "" {
			t.Errorf("preedit %q after the string was emptied", w.text.Preedit())
		}
	})
	t.Run("the input method off for the window cannot compose", func(t *testing.T) {
		_, k, _ := keyRun(t, false, platform.WindowConfig{}, compose("ni"))
		if len(k.texts) != 0 {
			t.Errorf("events with no text field: %+v", k.texts)
		}
	})
}

// TestWineSendInputUnicodeTypesText: SendInput with KEYEVENTF_UNICODE (what a touch keyboard, a
// remote desktop or an input method of another kind sends) makes VK_PACKET key messages and
// WM_CHAR; the character is typed once by the text input and the key makes no key data.
func TestWineSendInputUnicodeTypesText(t *testing.T) {
	releaseKeys(t)
	packet := func(r rune) func(*window, *keyRec) {
		return func(w *window, _ *keyRec) {
			var in []win32.KeyInput
			for _, u := range []uint16{uint16(r)} {
				in = append(in,
					win32.KeyInput{Type: win32.InputKeyboard, Key: win32.KeybdInput{Scan: u, Flags: win32.KeyEventFUnicode}},
					win32.KeyInput{Type: win32.InputKeyboard, Key: win32.KeybdInput{Scan: u, Flags: win32.KeyEventFUnicode | win32.KeyEventFKeyUp}})
			}
			win32.SendKeyInput(in)
		}
	}
	field := setField(textField(ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4}))
	_, k, _ := keyRun(t, false, platform.WindowConfig{}, field, packet('à'), packet('漢'))
	want := []ui.TextInputEvent{commit("à"), commit("漢")}
	if !reflect.DeepEqual(k.texts, want) {
		t.Errorf("events:\n got  %+v\n want %+v", k.texts, want)
	}
	if got := k.mainOnly(); len(got) != 0 {
		t.Errorf("VK_PACKET made key data: %v", got)
	}
}

// TestWineTypedLettersAreNotTypedTwice: the characters of a key are the key's (the framework types
// the character of a key that it does not handle), so the text input gets nothing for them.
func TestWineTypedLettersAreNotTypedTwice(t *testing.T) {
	releaseKeys(t)
	field := setField(textField(ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4}))
	_, k, _ := keyRun(t, false, platform.WindowConfig{}, field, keys("key:H"), keys("key:SHIFT+I"), keys("key:SPACE"))
	if len(k.texts) != 0 {
		t.Errorf("the text input typed %+v for keys with characters", k.texts)
	}
	var text string
	for _, d := range k.keys {
		if d.Type == down {
			text += d.Character
		}
	}
	if text != "hI " {
		t.Errorf("the characters of the keys = %q, want %q", text, "hI ")
	}
}
