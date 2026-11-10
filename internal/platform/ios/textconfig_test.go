package ios

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestConfigOf(t *testing.T) {
	for _, tc := range []struct {
		name string
		st   ui.TextInputState
		want textConfig
	}{
		{"plain text", ui.TextInputState{}, textConfig{SmartQuotes: traitYes, SmartDashes: traitYes, Autocorrect: traitNo, SpellCheck: traitNo}},
		{"corrections and spelling", ui.TextInputState{Hints: ui.TextInputHintCompletion | ui.TextInputHintSpellcheck | ui.TextInputHintAutoCapitalization},
			textConfig{Capitalization: capitalizeSentences, Autocorrect: traitYes, SpellCheck: traitYes, SmartQuotes: traitYes, SmartDashes: traitYes}},
		{"email", ui.TextInputState{Purpose: ui.TextInputPurposeEmail, EnterKey: ui.TextInputEnterKeyNext},
			textConfig{Keyboard: keyboardEmailAddress, ReturnKey: returnNext, ContentType: "emailAddress", Autocorrect: traitNo, SpellCheck: traitNo, SmartQuotes: traitYes, SmartDashes: traitYes}},
		{"password is secure with nothing suggested", ui.TextInputState{Purpose: ui.TextInputPurposePassword, Hints: ui.TextInputHintCompletion, EnterKey: ui.TextInputEnterKeyDone},
			textConfig{Keyboard: keyboardASCIICapable, ReturnKey: returnDone, Secure: true, ContentType: "password", Autocorrect: traitNo, SpellCheck: traitNo, SmartQuotes: traitNo, SmartDashes: traitNo}},
		{"a PIN is a secure number pad", ui.TextInputState{Purpose: ui.TextInputPurposePIN},
			textConfig{Keyboard: keyboardNumberPad, Secure: true, Autocorrect: traitNo, SpellCheck: traitNo, SmartQuotes: traitNo, SmartDashes: traitNo}},
		{"hidden text is secure", ui.TextInputState{Hints: ui.TextInputHintHiddenText},
			textConfig{Secure: true, Autocorrect: traitNo, SpellCheck: traitNo, SmartQuotes: traitNo, SmartDashes: traitNo}},
		{"sensitive data is not corrected", ui.TextInputState{Hints: ui.TextInputHintSensitiveData | ui.TextInputHintCompletion},
			textConfig{Autocorrect: traitNo, SpellCheck: traitNo, SmartQuotes: traitNo, SmartDashes: traitNo}},
		{"a search field with the go key and the title case", ui.TextInputState{EnterKey: ui.TextInputEnterKeySearch, Hints: ui.TextInputHintTitlecase | ui.TextInputHintMultiline},
			textConfig{ReturnKey: returnSearch, Capitalization: capitalizeWords, Multiline: true, Autocorrect: traitNo, SpellCheck: traitNo, SmartQuotes: traitYes, SmartDashes: traitYes}},
		{"every character in upper case, Latin", ui.TextInputState{Hints: ui.TextInputHintUppercase | ui.TextInputHintLatin},
			textConfig{Keyboard: keyboardASCIICapable, Capitalization: capitalizeAllCharacters, Autocorrect: traitNo, SpellCheck: traitNo, SmartQuotes: traitYes, SmartDashes: traitYes}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := configOf(tc.st); got != tc.want {
				t.Errorf("configOf = %+v\nwant         %+v", got, tc.want)
			}
		})
	}
}

func TestKeyboardAndReturnKeys(t *testing.T) {
	for p, want := range map[ui.TextInputPurpose]int{
		ui.TextInputPurposeDigits: keyboardNumberPad, ui.TextInputPurposeNumber: keyboardDecimalPad, ui.TextInputPurposePhone: keyboardPhonePad,
		ui.TextInputPurposeURL: keyboardURL, ui.TextInputPurposeDate: keyboardNumbersAndPunctuation, ui.TextInputPurposeTime: keyboardNumbersAndPunctuation,
		ui.TextInputPurposeDatetime: keyboardNumbersAndPunctuation, ui.TextInputPurposeAlpha: keyboardDefault, ui.TextInputPurposeName: keyboardDefault,
	} {
		if got := keyboardOf(p, 0); got != want {
			t.Errorf("keyboard of purpose %d = %d, want %d", p, got, want)
		}
	}
	for k, want := range map[ui.TextInputEnterKey]int{
		ui.TextInputEnterKeyEnter: returnDefault, ui.TextInputEnterKeyDone: returnDone, ui.TextInputEnterKeyGo: returnGo,
		ui.TextInputEnterKeyNext: returnNext, ui.TextInputEnterKeyPrevious: returnDefault, ui.TextInputEnterKeySearch: returnSearch, ui.TextInputEnterKeySend: returnSend,
	} {
		if got := returnKeyOf(k); got != want {
			t.Errorf("return key of enter key %d = %d, want %d", k, got, want)
		}
	}
}
