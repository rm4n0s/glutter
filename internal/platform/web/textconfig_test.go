package web

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// TestInputModePortedFromFlutter has the cases of 'sets correct input type in Android', 'for Firefox on
// Android' and 'in iOS' of text_editing_test.dart: inputmode is numeric for a number, decimal for a decimal
// number, tel, email and url, and nothing for text; it is set on those systems and not on a desktop one.
func TestInputModePortedFromFlutter(t *testing.T) {
	rows := []struct {
		name    string
		purpose ui.TextInputPurpose
		want    string
	}{
		{"text", ui.TextInputPurposeUnspecified, ""},
		{"number", ui.TextInputPurposeDigits, "numeric"},
		{"decimal number", ui.TextInputPurposeNumber, "decimal"},
		{"phone", ui.TextInputPurposePhone, "tel"},
		{"email address", ui.TextInputPurposeEmail, "email"},
		{"url", ui.TextInputPurposeURL, "url"},
		{"pin", ui.TextInputPurposePIN, "numeric"},
		{"name", ui.TextInputPurposeName, ""},
		{"password", ui.TextInputPurposePassword, ""},
	}
	for _, system := range []string{"Android with Blink", "Android with Firefox", "iOS"} {
		for _, r := range rows {
			t.Run(system+"/"+r.name, func(t *testing.T) {
				got := ElementConfigOf(ui.TextInputState{Enabled: true, Purpose: r.purpose}, true)
				if got.InputMode != r.want {
					t.Errorf("inputmode = %q, want %q", got.InputMode, r.want)
				}
				if desktop := ElementConfigOf(ui.TextInputState{Enabled: true, Purpose: r.purpose}, false); desktop.InputMode != "" {
					t.Errorf("a desktop browser got the inputmode %q", desktop.InputMode)
				}
			})
		}
	}
}

// TestElementConfigPortedFromFlutter has the cases 'Knows how to create password fields', 'Knows how to
// create non-default text actions', 'Knows to turn autocorrect off' and 'on', 'Multi-line mode also
// works' and the four capitalization cases ('No capitalization', 'All characters', 'Sentences', 'Words')
// of text_editing_test.dart.
func TestElementConfigPortedFromFlutter(t *testing.T) {
	tests := []struct {
		name   string
		state  ui.TextInputState
		mobile bool
		want   ElementConfig
	}{
		{"defaults", ui.TextInputState{},
			false, ElementConfig{Tag: "input", Autocapitalize: "off", Autocorrect: "off"}},
		{"password fields", ui.TextInputState{Purpose: ui.TextInputPurposePassword, Hints: ui.TextInputHintHiddenText},
			false, ElementConfig{Tag: "input", Type: "password", Autocapitalize: "off", Autocorrect: "off"}},
		{"send action on a phone", ui.TextInputState{EnterKey: ui.TextInputEnterKeySend},
			true, ElementConfig{Tag: "input", EnterKeyHint: "send", Autocapitalize: "off", Autocorrect: "off"}},
		{"send action on a desktop has no enterkeyhint", ui.TextInputState{EnterKey: ui.TextInputEnterKeySend},
			false, ElementConfig{Tag: "input", Autocapitalize: "off", Autocorrect: "off"}},
		{"autocorrect on", ui.TextInputState{Hints: ui.TextInputHintSpellcheck},
			false, ElementConfig{Tag: "input", Autocapitalize: "off", Autocorrect: "on"}},
		{"multi-line mode", ui.TextInputState{Hints: ui.TextInputHintMultiline},
			false, ElementConfig{Tag: "textarea", Autocapitalize: "off", Autocorrect: "off"}},
		{"all characters capitalization", ui.TextInputState{Hints: ui.TextInputHintUppercase},
			false, ElementConfig{Tag: "input", Autocapitalize: "characters", Autocorrect: "off"}},
		{"sentences capitalization", ui.TextInputState{Hints: ui.TextInputHintAutoCapitalization},
			false, ElementConfig{Tag: "input", Autocapitalize: "sentences", Autocorrect: "off"}},
		{"words capitalization", ui.TextInputState{Hints: ui.TextInputHintTitlecase},
			false, ElementConfig{Tag: "input", Autocapitalize: "words", Autocorrect: "off"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ElementConfigOf(tc.state, tc.mobile); got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestEnterKeyHintAttribute(t *testing.T) {
	tests := []struct {
		key  ui.TextInputEnterKey
		want string
	}{
		{ui.TextInputEnterKeyUnspecified, ""}, {ui.TextInputEnterKeyEnter, "enter"}, {ui.TextInputEnterKeyDone, "done"},
		{ui.TextInputEnterKeyGo, "go"}, {ui.TextInputEnterKeyNext, "next"}, {ui.TextInputEnterKeyPrevious, "previous"},
		{ui.TextInputEnterKeySearch, "search"}, {ui.TextInputEnterKeySend, "send"},
	}
	for _, tc := range tests {
		if got := EnterKeyHintAttribute(tc.key); got != tc.want {
			t.Errorf("EnterKeyHintAttribute(%v) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestFieldBoxOf(t *testing.T) {
	tests := []struct {
		name           string
		caret          ui.Rect
		originX, orgnY float64
		want           FieldBox
	}{
		{"at the origin", ui.Rect{Left: 10, Top: 20, Right: 12, Bottom: 38}, 0, 0, FieldBox{10, 20, 2, 18, 18}},
		{"in a host that is moved", ui.Rect{Left: 10, Top: 20, Right: 12, Bottom: 38}, 100, 50, FieldBox{110, 70, 2, 18, 18}},
		{"an empty rectangle is one pixel", ui.Rect{Left: 5, Top: 5, Right: 5, Bottom: 5}, 0, 0, FieldBox{5, 5, 1, 1, 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := FieldBoxOf(tc.caret, tc.originX, tc.orgnY); got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestIsNativePasteKey(t *testing.T) {
	tests := []struct {
		name string
		e    KeyEvent
		os   OperatingSystem
		want bool
	}{
		{"Control+V", KeyEvent{Down: true, Code: "KeyV", Key: "v", Ctrl: true}, OSLinux, true},
		{"Control+Shift+V pastes plain text", KeyEvent{Down: true, Code: "KeyV", Key: "V", Ctrl: true, Shift: true}, OSLinux, true},
		{"Control+V on Windows", KeyEvent{Down: true, Code: "KeyV", Key: "v", Ctrl: true}, OSWindows, true},
		{"Control+V on a layout that is not Latin", KeyEvent{Down: true, Code: "KeyV", Key: "м", Ctrl: true}, OSLinux, true},
		{"Shift+Insert", KeyEvent{Down: true, Code: "Insert", Key: "Insert", Shift: true}, OSLinux, true},
		{"Control+V on macOS pastes nothing", KeyEvent{Down: true, Code: "KeyV", Key: "v", Ctrl: true}, OSMacOS, false},
		{"Command+V is the browser's, and nothing in the framework", KeyEvent{Down: true, Code: "KeyV", Key: "v", Meta: true}, OSMacOS, false},
		{"Control+Alt+V", KeyEvent{Down: true, Code: "KeyV", Key: "v", Ctrl: true, Alt: true}, OSLinux, false},
		{"V alone", KeyEvent{Down: true, Code: "KeyV", Key: "v"}, OSLinux, false},
		{"Control+C", KeyEvent{Down: true, Code: "KeyC", Key: "c", Ctrl: true}, OSLinux, false},
		{"Control+Insert is copy", KeyEvent{Down: true, Code: "Insert", Key: "Insert", Ctrl: true}, OSLinux, false},
		{"the key up", KeyEvent{Down: false, Code: "KeyV", Key: "v", Ctrl: true}, OSLinux, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNativePasteKey(tc.e, tc.os); got != tc.want {
				t.Errorf("IsNativePasteKey = %v, want %v", got, tc.want)
			}
		})
	}
}
