package ui

// TextInputPurpose is what the focused text field is for, so an input method
// can show a fitting keyboard or enable the right dictionaries. The zero value,
// TextInputPurposeUnspecified, means normal text.
//
// Flutter: no counterpart (Flutter sends the TextInputType of the configuration
// over the flutter/textinput channel)
type TextInputPurpose int

// The values of TextInputPurpose follow the content purposes of the Wayland
// text-input protocol.
const (
	// TextInputPurposeUnspecified is the zero value: normal text.
	//
	// Flutter: no counterpart
	TextInputPurposeUnspecified TextInputPurpose = iota
	// TextInputPurposeAlpha allows letters only.
	//
	// Flutter: no counterpart
	TextInputPurposeAlpha
	// TextInputPurposeDigits allows digits only.
	//
	// Flutter: no counterpart
	TextInputPurposeDigits
	// TextInputPurposeNumber is a number, with a sign and a decimal separator.
	//
	// Flutter: no counterpart
	TextInputPurposeNumber
	// TextInputPurposePhone is a phone number.
	//
	// Flutter: no counterpart
	TextInputPurposePhone
	// TextInputPurposeURL is a URL.
	//
	// Flutter: no counterpart
	TextInputPurposeURL
	// TextInputPurposeEmail is an email address.
	//
	// Flutter: no counterpart
	TextInputPurposeEmail
	// TextInputPurposeName is a person's name.
	//
	// Flutter: no counterpart
	TextInputPurposeName
	// TextInputPurposePassword is a password; the text is hidden.
	//
	// Flutter: no counterpart
	TextInputPurposePassword
	// TextInputPurposePIN is a numeric PIN; the text is hidden.
	//
	// Flutter: no counterpart
	TextInputPurposePIN
	// TextInputPurposeDate is a date.
	//
	// Flutter: no counterpart
	TextInputPurposeDate
	// TextInputPurposeTime is a time.
	//
	// Flutter: no counterpart
	TextInputPurposeTime
	// TextInputPurposeDatetime is a date and a time.
	//
	// Flutter: no counterpart
	TextInputPurposeDatetime
)

var textInputPurposeNames = [...]string{
	"TextInputPurposeUnspecified", "TextInputPurposeAlpha", "TextInputPurposeDigits",
	"TextInputPurposeNumber", "TextInputPurposePhone", "TextInputPurposeURL", "TextInputPurposeEmail",
	"TextInputPurposeName", "TextInputPurposePassword", "TextInputPurposePIN", "TextInputPurposeDate",
	"TextInputPurposeTime", "TextInputPurposeDatetime",
}

// String returns the name of the value, such as "TextInputPurposeEmail".
func (p TextInputPurpose) String() string {
	return enumName(textInputPurposeNames[:], int(p), "TextInputPurpose")
}

// TextInputEnterKey is what the Enter key of a virtual keyboard is labeled and
// does, as the platform can show it: the action of the text field. The zero
// value, TextInputEnterKeyUnspecified, means the platform's own label.
//
// Flutter: no counterpart (Flutter sends the TextInputAction of the
// configuration over the flutter/textinput channel)
type TextInputEnterKey int

// The values of TextInputEnterKey follow the values of the HTML enterkeyhint
// attribute.
const (
	// TextInputEnterKeyUnspecified is the zero value: the platform's own label.
	//
	// Flutter: no counterpart
	TextInputEnterKeyUnspecified TextInputEnterKey = iota
	// TextInputEnterKeyEnter inserts a line break.
	//
	// Flutter: no counterpart
	TextInputEnterKeyEnter
	// TextInputEnterKeyDone means the user is done with the input.
	//
	// Flutter: no counterpart
	TextInputEnterKeyDone
	// TextInputEnterKeyGo takes the user to the target of the text.
	//
	// Flutter: no counterpart
	TextInputEnterKeyGo
	// TextInputEnterKeyNext moves the focus to the next field.
	//
	// Flutter: no counterpart
	TextInputEnterKeyNext
	// TextInputEnterKeyPrevious moves the focus to the previous field.
	//
	// Flutter: no counterpart
	TextInputEnterKeyPrevious
	// TextInputEnterKeySearch runs a search for the text.
	//
	// Flutter: no counterpart
	TextInputEnterKeySearch
	// TextInputEnterKeySend sends the text.
	//
	// Flutter: no counterpart
	TextInputEnterKeySend
)

var textInputEnterKeyNames = [...]string{
	"TextInputEnterKeyUnspecified", "TextInputEnterKeyEnter", "TextInputEnterKeyDone", "TextInputEnterKeyGo",
	"TextInputEnterKeyNext", "TextInputEnterKeyPrevious", "TextInputEnterKeySearch", "TextInputEnterKeySend",
}

// String returns the name of the value, such as "TextInputEnterKeyDone".
func (k TextInputEnterKey) String() string {
	return enumName(textInputEnterKeyNames[:], int(k), "TextInputEnterKey")
}

// TextInputHints are the flags that tell an input method how to treat the text
// of the focused text field. The zero value has no hint.
//
// Flutter: no counterpart
type TextInputHints uint32

// The flags of TextInputHints; their values are the content hints of the
// Wayland text-input protocol.
const (
	// TextInputHintCompletion suggests completions of the word.
	//
	// Flutter: no counterpart
	TextInputHintCompletion TextInputHints = 1 << iota
	// TextInputHintSpellcheck checks the spelling.
	//
	// Flutter: no counterpart
	TextInputHintSpellcheck
	// TextInputHintAutoCapitalization capitalizes the first letter of a
	// sentence.
	//
	// Flutter: no counterpart
	TextInputHintAutoCapitalization
	// TextInputHintLowercase prefers lowercase letters.
	//
	// Flutter: no counterpart
	TextInputHintLowercase
	// TextInputHintUppercase prefers uppercase letters.
	//
	// Flutter: no counterpart
	TextInputHintUppercase
	// TextInputHintTitlecase capitalizes the first letter of every word.
	//
	// Flutter: no counterpart
	TextInputHintTitlecase
	// TextInputHintHiddenText hides the text: no suggestions are shown.
	//
	// Flutter: no counterpart
	TextInputHintHiddenText
	// TextInputHintSensitiveData keeps the text out of any learning or history.
	//
	// Flutter: no counterpart
	TextInputHintSensitiveData
	// TextInputHintLatin expects Latin letters.
	//
	// Flutter: no counterpart
	TextInputHintLatin
	// TextInputHintMultiline expects several lines of text.
	//
	// Flutter: no counterpart
	TextInputHintMultiline
)

// Has reports whether all the flags of f are set.
//
// Flutter: no counterpart
func (h TextInputHints) Has(f TextInputHints) bool { return h&f == f }

// TextInputState is what the input method (IME) of the platform needs to know
// about the focused text field. The framework sends a new state to the
// platform each time one of its parts changes.
//
// Flutter: no counterpart (Flutter sends the pieces over the flutter/textinput
// channel: TextInput.setClient, setEditingState, setCaretRect and the like)
type TextInputState struct {
	// Enabled is whether a text field has focus. When false, the other fields
	// mean nothing and the input method is turned off.
	Enabled bool
	// View is the view that the text field is in.
	View ViewID
	// Purpose is what the text field is for.
	Purpose TextInputPurpose
	// Hints tell the input method how to treat the text.
	Hints TextInputHints
	// Text is the text of the field around the caret, without the text that the
	// input method is composing (the preedit). It is whole unless the field is
	// too long for the platform; then it is cut around the caret and
	// the selection is relative to the cut text.
	Text string
	// SelectionAnchor and SelectionCursor are the selection in Text, in UTF-8
	// byte offsets: the anchor is where the selection began and the cursor is
	// the end that moves. They are equal for a caret.
	SelectionAnchor, SelectionCursor int
	// CaretRect is the rectangle of the caret, in logical pixels of the view, so
	// the input method can place its candidate window next to it.
	CaretRect Rect
	// EnterKey is the action of the field, for a platform whose keyboard labels
	// its Enter key (the web's enterkeyhint).
	EnterKey TextInputEnterKey
	// CausedByInputMethod is whether the last change of Text or Selection comes
	// from the input method, not from the user or the app.
	CausedByInputMethod bool
}

// TextInputEvent is a change that the input method asks of the text field. The
// platform sends one event for each batch of changes the input method makes at
// once; apply them in this order: remove the text being composed, delete the
// surrounding text, insert Commit (or remove the selection, when DeleteSelection
// asks for it), then show Preedit.
//
// Flutter: no counterpart (Flutter receives TextInputClient.updateEditingState
// and the like over the flutter/textinput channel)
type TextInputEvent struct {
	// View is the view whose text field the event is for.
	View ViewID
	// DeleteBefore and DeleteAfter are the bytes to delete before and after the
	// cursor, once the text being composed is removed.
	DeleteBefore, DeleteAfter int
	// DeleteSelection removes the selected text even when Commit and Preedit are
	// empty, as a Backspace does. Without it an event that commits nothing leaves
	// the selection as it is.
	DeleteSelection bool
	// Commit is the text to insert at the cursor, replacing the selection.
	Commit string
	// Preedit is the text the input method is composing, shown at the cursor
	// until it is committed. Empty means there is none.
	Preedit string
	// PreeditCursorBegin and PreeditCursorEnd are the selection inside Preedit,
	// in UTF-8 byte offsets. Both -1 means the input method hides the cursor.
	PreeditCursorBegin, PreeditCursorEnd int
}
