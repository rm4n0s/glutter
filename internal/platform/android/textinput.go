package android

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/textedit"
	"github.com/rm4n0s/glutter/ui"
)

// The operations of the activity's command call for text input and the clipboard (systemui.go has
// the first seven).
const (
	// opTextStart: a text field has the focus. A is the Android inputType, B the imeOptions (without
	// the flags that the Java side adds), C is 1 for a multiline field.
	opTextStart = 8
	// opTextState: the framework's editing state. A is the number of the last edit of the input
	// method that this state follows, B and C the selection in UTF-16 units, S the text.
	opTextState = 9
	// opTextStop: no text field has the focus.
	opTextStop = 10
	// opClipboardSet: S is the text to put on the clipboard.
	opClipboardSet = 12
	// opClipboardGet: A is the number of the read; the answer is Native.clipboardText.
	opClipboardGet = 13
)

// The values of android.text.InputType that the input type is made of.
const (
	typeClassText     = 0x1
	typeClassNumber   = 0x2
	typeClassPhone    = 0x3
	typeClassDatetime = 0x4

	typeTextFlagCapCharacters = 0x1000
	typeTextFlagCapWords      = 0x2000
	typeTextFlagCapSentences  = 0x4000
	typeTextFlagAutoCorrect   = 0x8000
	typeTextFlagMultiLine     = 0x20000
	typeTextFlagNoSuggestions = 0x80000

	typeTextVariationURI             = 0x10
	typeTextVariationEmailAddress    = 0x20
	typeTextVariationPersonName      = 0x60
	typeTextVariationPassword        = 0x80
	typeTextVariationVisiblePassword = 0x90

	typeNumberFlagSigned      = 0x1000
	typeNumberFlagDecimal     = 0x2000
	typeNumberVariationPasswd = 0x10

	typeDatetimeVariationDate = 0x10
	typeDatetimeVariationTime = 0x20
)

// The values of EditorInfo.IME_ACTION_* and the flag that keeps the keyboard from learning.
const (
	imeActionNone     = 0x1
	imeActionGo       = 0x2
	imeActionSearch   = 0x3
	imeActionSend     = 0x4
	imeActionNext     = 0x5
	imeActionDone     = 0x6
	imeActionPrevious = 0x7

	imeFlagNoPersonalizedLearning = 0x1000000
)

// editorConfig is what the input method is told about the text field: the Android input type and
// the IME options of the EditorInfo.
//
// Flutter: shell/platform/android/io/flutter/plugin/editing/TextInputPlugin.java TextInputPlugin.createInputConnection
type editorConfig struct {
	inputType, imeOptions int
	multiline             bool
}

// editorConfigOf makes the EditorInfo of a text field from what the framework said about it
// (ui.TextInputState): the purpose is the input type, the hints are what Flutter's configuration
// says about obscured text, autocorrect, suggestions and capitalization, and the enter key is the
// IME action; a field with no action asks for Done, a multiline one for none.
//
// Flutter: shell/platform/android/io/flutter/plugin/editing/TextInputPlugin.java TextInputPlugin.inputTypeFromTextInputType
func editorConfigOf(s ui.TextInputState) editorConfig {
	multiline := s.Hints.Has(ui.TextInputHintMultiline)
	c := editorConfig{inputType: inputTypeOf(s), multiline: multiline}
	if s.Hints.Has(ui.TextInputHintSensitiveData) {
		c.imeOptions |= imeFlagNoPersonalizedLearning
	}
	switch s.EnterKey {
	case ui.TextInputEnterKeyEnter:
		c.imeOptions |= imeActionNone
	case ui.TextInputEnterKeyDone:
		c.imeOptions |= imeActionDone
	case ui.TextInputEnterKeyGo:
		c.imeOptions |= imeActionGo
	case ui.TextInputEnterKeyNext:
		c.imeOptions |= imeActionNext
	case ui.TextInputEnterKeyPrevious:
		c.imeOptions |= imeActionPrevious
	case ui.TextInputEnterKeySearch:
		c.imeOptions |= imeActionSearch
	case ui.TextInputEnterKeySend:
		c.imeOptions |= imeActionSend
	default:
		if multiline {
			c.imeOptions |= imeActionNone
		} else {
			c.imeOptions |= imeActionDone
		}
	}
	return c
}

// inputTypeOf is the android.text.InputType of the text field.
func inputTypeOf(s ui.TextInputState) int {
	switch s.Purpose {
	case ui.TextInputPurposeDatetime:
		return typeClassDatetime
	case ui.TextInputPurposeDate:
		return typeClassDatetime | typeDatetimeVariationDate
	case ui.TextInputPurposeTime:
		return typeClassDatetime | typeDatetimeVariationTime
	case ui.TextInputPurposeDigits:
		return typeClassNumber
	case ui.TextInputPurposeNumber:
		return typeClassNumber | typeNumberFlagSigned | typeNumberFlagDecimal
	case ui.TextInputPurposePIN:
		return typeClassNumber | typeNumberVariationPasswd
	case ui.TextInputPurposePhone:
		return typeClassPhone
	}
	t := typeClassText
	if s.Hints.Has(ui.TextInputHintMultiline) {
		t |= typeTextFlagMultiLine
	}
	switch s.Purpose {
	case ui.TextInputPurposeEmail:
		t |= typeTextVariationEmailAddress
	case ui.TextInputPurposeURL:
		t |= typeTextVariationURI
	case ui.TextInputPurposeName:
		t |= typeTextVariationPersonName
	}
	if s.Purpose == ui.TextInputPurposePassword || s.Hints.Has(ui.TextInputHintHiddenText) {
		// Both are needed: some devices ignore the flag.
		t |= typeTextFlagNoSuggestions | typeTextVariationPassword
	} else {
		if s.Hints.Has(ui.TextInputHintSpellcheck) {
			t |= typeTextFlagAutoCorrect
		}
		if !s.Hints.Has(ui.TextInputHintCompletion) {
			t |= typeTextFlagNoSuggestions | typeTextVariationVisiblePassword
		}
	}
	switch {
	case s.Hints.Has(ui.TextInputHintUppercase):
		t |= typeTextFlagCapCharacters
	case s.Hints.Has(ui.TextInputHintTitlecase):
		t |= typeTextFlagCapWords
	case s.Hints.Has(ui.TextInputHintAutoCapitalization):
		t |= typeTextFlagCapSentences
	}
	return t
}

// textInput is the platform's side of the text field that has the focus: the model that finds
// what the input method changed (the web's, over UTF-16 text and a composing range as the
// Editable of the Java side holds them) and what the activity was last told (SPEC §24.6). It
// belongs to the UI thread.
//
// Flutter: shell/platform/android/io/flutter/plugin/editing/TextInputPlugin.java TextInputPlugin
type textInput struct {
	editor textedit.TextEditor
	// active is whether a text field has the focus; config is what the input method was told.
	active bool
	config editorConfig
	// seq is the number of the last edit of the input method that was handled; the editing state that
	// goes to the Java side carries it, and the Java side ignores a state that follows an older
	// edit than the one it has sent since.
	seq int
}

// setState takes what the framework says about the text field. It starts text input when the field
// got the focus or its configuration changed, writes the state into the Java side's editing state
// when that differs from it, and stops text input for a state that is not enabled.
//
// Flutter: shell/platform/android/io/flutter/plugin/editing/TextInputPlugin.java TextInputPlugin.setTextInputEditingState
func (t *textInput) setState(h host, s ui.TextInputState) {
	if !s.Enabled {
		if t.active {
			t.active = false
			t.editor.Reset()
			h.Command(command{Op: opTextStop})
		}
		return
	}
	cfg := editorConfigOf(s)
	if !t.active || cfg != t.config {
		t.active, t.config = true, cfg
		t.editor.Reset()
		multiline := int64(0)
		if cfg.multiline {
			multiline = 1
		}
		h.Command(command{Op: opTextStart, A: int64(cfg.inputType), B: int64(cfg.imeOptions), C: multiline})
	}
	if w := t.editor.SetState(s); w.Write {
		h.Command(command{Op: opTextState, A: int64(t.seq), B: int64(w.Base), C: int64(w.Extent), S: w.Text})
	}
}

// edit takes the editing state of the Java side after the input method changed it (text, selection
// and composing range in UTF-16 units; the composing range is -1, -1 when there is none) and returns
// the event that makes the framework's field the same, and whether there is one. seq is the number
// of the edit.
//
// Flutter: shell/platform/android/io/flutter/plugin/editing/InputConnectionAdaptor.java InputConnectionAdaptor (updateEditingState)
func (t *textInput) edit(seq int, text string, base, extent, composeBase, composeExtent int) (ui.TextInputEvent, bool) {
	t.seq = seq
	if !t.active {
		return ui.TextInputEvent{}, false
	}
	return t.editor.Handle(textedit.EditingState{
		Text: text, Base: max(0, base), Extent: max(0, extent), ComposingBase: composeBase, ComposingExtent: composeExtent,
	})
}

// SetTextInput tells the input method about the focused text field (SPEC §24.6).
//
// Flutter: shell/platform/android/io/flutter/plugin/editing/TextInputPlugin.java TextInputPlugin
func (w *window) SetTextInput(state ui.TextInputState) { w.p.text.setState(w.p.host, state) }

// textEdit is an edit of the input method (Native.textEdit): the event for the framework goes to
// the handler at once, after the input that waits for its frame, as the web's does.
func (p *Platform) textEdit(seq int, text string, base, extent, composeBase, composeExtent int) {
	ev, ok := p.text.edit(seq, text, base, extent, composeBase, composeExtent)
	if !ok || p.handler == nil || p.win == nil {
		return
	}
	ev.View = p.win.id
	if sh, ok := p.handler.(platform.SyncInputHandler); ok {
		sh.OnTextInputSync(ev)
		return
	}
	p.handler.OnTextInput(ev)
}

// resend asks the activity again for the text input that is on, after a new activity came into the
// process: it starts it and writes the framework's state.
func (t *textInput) resend(h host) {
	if !t.active {
		return
	}
	state := t.editor.State()
	t.active = false
	t.setState(h, state)
}
