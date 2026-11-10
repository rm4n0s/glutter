package ios

import "github.com/rm4n0s/glutter/ui"

// This file is the plain-Go part of the configuration of the text input (SPEC §30.5): what the framework says of a field (its purpose, its
// hints and its enter key) as the traits of a UITextInputTraits, with UIKit's numbers, so that a table of fields tests it on every host.

// The values of UIKeyboardType.
const (
	keyboardDefault               = 0
	keyboardASCIICapable          = 1
	keyboardNumbersAndPunctuation = 2
	keyboardURL                   = 3
	keyboardNumberPad             = 4
	keyboardPhonePad              = 5
	keyboardNamePhonePad          = 6
	keyboardEmailAddress          = 7
	keyboardDecimalPad            = 8
)

// The values of UIReturnKeyType.
const (
	returnDefault = 0
	returnGo      = 1
	returnNext    = 4
	returnSearch  = 6
	returnSend    = 7
	returnDone    = 9
)

// The values of UITextAutocapitalizationType.
const (
	capitalizeNone          = 0
	capitalizeWords         = 1
	capitalizeSentences     = 2
	capitalizeAllCharacters = 3
)

// The values of the UITextAutocorrectionType, UITextSpellCheckingType and UITextSmartQuotesType (and dashes, and insert delete) traits:
// 0 is the system's choice, 1 is no and 2 is yes.
const (
	traitDefault = 0
	traitNo      = 1
	traitYes     = 2
)

// textConfig are the traits of a field, with UIKit's numbers.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterTextInputPlugin.mm configureWithDictionary:
type textConfig struct {
	Keyboard       int
	ReturnKey      int
	Capitalization int
	Autocorrect    int
	SpellCheck     int
	SmartQuotes    int
	SmartDashes    int
	Secure         bool
	// ContentType is the UITextContentType string that AutoFill reads, or "" for none.
	ContentType string
	// Multiline says the enter key makes a new line, so that it is the return key of a text view.
	Multiline bool
}

// configOf is the traits of the field that the state describes. A field that hides its text (a password, a PIN, the hidden-text hint)
// is secure and has no suggestions, no corrections and no smart punctuation; a field has corrections only when the framework asks for
// completion and spell checking only when it asks for that, and the smart quotes and dashes follow the corrections, as in Flutter.
//
// Flutter: FlutterTextInputPlugin.mm ToUIKeyboardType, ToUIReturnKeyType, ToUITextAutoCapitalizationType, configureWithDictionary:
func configOf(st ui.TextInputState) textConfig {
	c := textConfig{
		Keyboard: keyboardOf(st.Purpose, st.Hints), ReturnKey: returnKeyOf(st.EnterKey),
		Capitalization: capitalizationOf(st.Hints), ContentType: contentTypeOf(st.Purpose),
		Multiline: st.Hints.Has(ui.TextInputHintMultiline),
	}
	c.Secure = st.Purpose == ui.TextInputPurposePassword || st.Purpose == ui.TextInputPurposePIN || st.Hints.Has(ui.TextInputHintHiddenText)
	switch {
	case c.Secure || st.Hints.Has(ui.TextInputHintSensitiveData):
		c.Autocorrect, c.SpellCheck, c.SmartQuotes, c.SmartDashes = traitNo, traitNo, traitNo, traitNo
	default:
		c.Autocorrect, c.SpellCheck = yesNo(st.Hints.Has(ui.TextInputHintCompletion)), yesNo(st.Hints.Has(ui.TextInputHintSpellcheck))
		c.SmartQuotes, c.SmartDashes = traitYes, traitYes
	}
	return c
}

func yesNo(b bool) int {
	if b {
		return traitYes
	}
	return traitNo
}

// keyboardOf is the keyboard of a purpose: a number pad for digits and a PIN, the decimal pad for numbers, the phone pad, the keyboards of
// a URL and an email address, numbers and punctuation for dates and times; a password or the Latin hint is an ASCII-capable keyboard.
func keyboardOf(p ui.TextInputPurpose, h ui.TextInputHints) int {
	switch p {
	case ui.TextInputPurposeDigits, ui.TextInputPurposePIN:
		return keyboardNumberPad
	case ui.TextInputPurposeNumber:
		return keyboardDecimalPad
	case ui.TextInputPurposePhone:
		return keyboardPhonePad
	case ui.TextInputPurposeURL:
		return keyboardURL
	case ui.TextInputPurposeEmail:
		return keyboardEmailAddress
	case ui.TextInputPurposeDate, ui.TextInputPurposeTime, ui.TextInputPurposeDatetime:
		return keyboardNumbersAndPunctuation
	case ui.TextInputPurposePassword:
		return keyboardASCIICapable
	}
	if h.Has(ui.TextInputHintLatin) {
		return keyboardASCIICapable
	}
	return keyboardDefault
}

// returnKeyOf is the return key of an enter key.
func returnKeyOf(k ui.TextInputEnterKey) int {
	switch k {
	case ui.TextInputEnterKeyDone:
		return returnDone
	case ui.TextInputEnterKeyGo:
		return returnGo
	case ui.TextInputEnterKeyNext:
		return returnNext
	case ui.TextInputEnterKeySearch:
		return returnSearch
	case ui.TextInputEnterKeySend:
		return returnSend
	}
	return returnDefault
}

// capitalizationOf is the capitalization of the hints: upper case for every character, title case for words, the sentences' first letter.
func capitalizationOf(h ui.TextInputHints) int {
	switch {
	case h.Has(ui.TextInputHintUppercase):
		return capitalizeAllCharacters
	case h.Has(ui.TextInputHintTitlecase):
		return capitalizeWords
	case h.Has(ui.TextInputHintAutoCapitalization):
		return capitalizeSentences
	}
	return capitalizeNone
}

// contentTypeOf is the UITextContentType of a purpose, which AutoFill and the password manager read.
func contentTypeOf(p ui.TextInputPurpose) string {
	switch p {
	case ui.TextInputPurposePassword:
		return "password"
	case ui.TextInputPurposeEmail:
		return "emailAddress"
	case ui.TextInputPurposeURL:
		return "URL"
	case ui.TextInputPurposeName:
		return "name"
	case ui.TextInputPurposePhone:
		return "telephoneNumber"
	}
	return ""
}
