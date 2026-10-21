package services

import (
	"unicode/utf8"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// platformTextInputControl is the control of the input method of the platform
// (Flutter's _PlatformTextInputControl). Flutter's engine keeps a model of the
// edited text and answers the platform's events with it; here the control keeps
// that model: it holds the value of the client, gives the platform the
// ui.TextInputState of it, and turns each ui.TextInputEvent into a new value
// for the client.
//
// Flutter: services/text_input.dart _PlatformTextInputControl
type platformTextInputControl struct {
	input   *TextInput
	backend TextInputBackend

	attached bool
	shown    bool
	config   TextInputConfiguration
	value    TextEditingValue
	// size and transform place the editable in the view; the caret rectangle is
	// in the coordinates of the editable.
	transform *ui.Matrix4
	caretRect ui.Rect
	// fromInputMethod is whether the last change of the value came from the
	// input method.
	fromInputMethod bool
}

// Attach remembers the client's configuration. The input method is turned on by
// Show.
func (p *platformTextInputControl) Attach(_ TextInputClient, configuration TextInputConfiguration) {
	p.attached = true
	p.config = configuration
	p.value = TextEditingValueEmpty
	p.fromInputMethod = false
	p.send()
}

// Detach turns the input method off.
func (p *platformTextInputControl) Detach(TextInputClient) {
	p.attached = false
	p.shown = false
	p.send()
}

// Show turns the input method on for the client.
func (p *platformTextInputControl) Show() {
	p.shown = true
	p.send()
}

// Hide turns the input method off.
func (p *platformTextInputControl) Hide() {
	p.shown = false
	p.send()
}

// UpdateConfig remembers the new configuration.
func (p *platformTextInputControl) UpdateConfig(configuration TextInputConfiguration) {
	p.config = configuration
	p.send()
}

// SetEditingState remembers the value of the client.
func (p *platformTextInputControl) SetEditingState(value TextEditingValue) {
	p.value = value
	p.fromInputMethod = false
	p.send()
}

// SetEditableSizeAndTransform remembers where the editable is in the view.
func (p *platformTextInputControl) SetEditableSizeAndTransform(_ ui.Size, transform *ui.Matrix4) {
	p.transform = transform.Clone()
	p.send()
}

// SetComposingRect does nothing: the input method gets the caret rectangle.
func (p *platformTextInputControl) SetComposingRect(ui.Rect) {}

// SetCaretRect remembers the rectangle of the caret.
func (p *platformTextInputControl) SetCaretRect(rect ui.Rect) {
	p.caretRect = rect
	p.send()
}

// SetSelectionRects does nothing: only iOS uses them.
func (p *platformTextInputControl) SetSelectionRects([]SelectionRect) {}

// UpdateStyle does nothing: no input method of Linux draws text.
func (p *platformTextInputControl) UpdateStyle(TextInputStyle) {}

// RequestAutofill does nothing: no input method of Linux fills fields.
func (p *platformTextInputControl) RequestAutofill() {}

// FinishAutofillContext does nothing.
func (p *platformTextInputControl) FinishAutofillContext(bool) {}

// enabled is whether the input method is on: a client is attached and has asked
// for it, and the field takes text. A field of the type none, or any field while
// the app has a control of its own, does without the input method of the
// platform, as Flutter makes the platform think the type is none then.
func (p *platformTextInputControl) enabled() bool {
	if !p.attached || !p.shown {
		return false
	}
	if p.config.InputType == TextInputTypeNone || p.input.currentControl != TextInputControl(p) {
		return false
	}
	return true
}

func (p *platformTextInputControl) send() {
	if p.backend == nil {
		return
	}
	p.backend.SetTextInput(p.state())
}

// state builds the state the input method needs.
func (p *platformTextInputControl) state() ui.TextInputState {
	if !p.enabled() {
		return ui.TextInputState{}
	}
	purpose, hints := p.contentType()
	text, anchor, cursor := surroundingText(p.value)
	caret := p.caretRect
	if p.transform != nil && caret.IsFinite() {
		caret = p.transform.TransformRect(caret)
	}
	if !caret.IsFinite() {
		caret = ui.Rect{}
	}
	return ui.TextInputState{
		Enabled:             true,
		View:                ui.ViewID(p.config.ViewID.OrElse(int(ui.ImplicitViewID))),
		Purpose:             purpose,
		Hints:               hints,
		Text:                text,
		SelectionAnchor:     anchor,
		SelectionCursor:     cursor,
		CaretRect:           caret,
		EnterKey:            enterKeyOf(p.config.InputAction),
		CausedByInputMethod: p.fromInputMethod,
	}
}

// enterKeyOf maps the action of the field to the label of the Enter key of a
// virtual keyboard, as Flutter web maps TextInputAction to enterkeyhint.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/input_action.dart EngineInputAction.fromName
func enterKeyOf(a TextInputAction) ui.TextInputEnterKey {
	switch a {
	case TextInputActionUnspecified, TextInputActionDone:
		return ui.TextInputEnterKeyDone
	case TextInputActionNext, TextInputActionContinueAction:
		return ui.TextInputEnterKeyNext
	case TextInputActionPrevious:
		return ui.TextInputEnterKeyPrevious
	case TextInputActionGo:
		return ui.TextInputEnterKeyGo
	case TextInputActionNewline:
		return ui.TextInputEnterKeyEnter
	case TextInputActionSearch:
		return ui.TextInputEnterKeySearch
	case TextInputActionSend:
		return ui.TextInputEnterKeySend
	}
	return ui.TextInputEnterKeyUnspecified
}

// contentType maps the configuration to what the input method understands.
func (p *platformTextInputControl) contentType() (ui.TextInputPurpose, ui.TextInputHints) {
	c := p.config
	var purpose ui.TextInputPurpose
	var hints ui.TextInputHints
	switch c.InputType {
	case TextInputTypeMultiline:
		hints |= ui.TextInputHintMultiline
	case TextInputTypePhone:
		purpose = ui.TextInputPurposePhone
	case TextInputTypeDatetime:
		purpose = ui.TextInputPurposeDatetime
	case TextInputTypeEmailAddress:
		purpose = ui.TextInputPurposeEmail
	case TextInputTypeURL:
		purpose = ui.TextInputPurposeURL
	case TextInputTypeVisiblePassword:
		hints |= ui.TextInputHintSensitiveData
	case TextInputTypeName:
		purpose = ui.TextInputPurposeName
	default:
		if c.InputType.index == TextInputTypeNumber.index {
			purpose = ui.TextInputPurposeDigits
			if c.InputType.signed == foundation.Some(true) || c.InputType.decimal == foundation.Some(true) {
				purpose = ui.TextInputPurposeNumber
			}
		}
	}
	if c.ObscureText {
		purpose = ui.TextInputPurposePassword
		hints |= ui.TextInputHintHiddenText | ui.TextInputHintSensitiveData
	}
	if c.Autocorrect {
		hints |= ui.TextInputHintSpellcheck
	}
	if c.EnableSuggestions {
		hints |= ui.TextInputHintCompletion
	}
	if !c.EnableIMEPersonalizedLearning {
		hints |= ui.TextInputHintSensitiveData
	}
	switch c.TextCapitalization {
	case TextCapitalizationSentences:
		hints |= ui.TextInputHintAutoCapitalization
	case TextCapitalizationWords:
		hints |= ui.TextInputHintTitlecase
	case TextCapitalizationCharacters:
		hints |= ui.TextInputHintUppercase
	}
	return purpose, hints
}

// surroundingText returns the text of the value without the text being composed,
// which the input method keeps to itself, and the anchor and the cursor in it.
func surroundingText(v TextEditingValue) (text string, anchor, cursor int) {
	text = v.Text
	anchor, cursor = v.Selection.BaseOffset, v.Selection.ExtentOffset
	if anchor < 0 || cursor < 0 {
		anchor, cursor = 0, 0
	}
	if v.IsComposingRangeValid() && !v.Composing.IsCollapsed() {
		c := v.Composing
		text = v.Text[:c.Start] + v.Text[c.End:]
		anchor, cursor = outsideComposing(anchor, c), outsideComposing(cursor, c)
	}
	return text, min(anchor, len(text)), min(cursor, len(text))
}

// outsideComposing is the offset in the text without the composing range.
func outsideComposing(offset int, c ui.TextRange) int {
	switch {
	case offset >= c.End:
		return offset - (c.End - c.Start)
	case offset > c.Start:
		return c.Start
	}
	return offset
}

// handleEvent applies a change that the input method asks for, in the order of
// the Wayland text-input protocol (which the other input methods follow): the
// text being composed goes, the surrounding text is deleted, the commit string
// replaces the selection, and the new text to compose is inserted at the cursor.
// The result goes to the client, as a value or as a delta.
func (p *platformTextInputControl) handleEvent(e ui.TextInputEvent) {
	oldValue := p.value
	v := oldValue

	// 1. Remove the preedit that was shown.
	if v.IsComposingRangeValid() && !v.Composing.IsCollapsed() {
		v = v.Replaced(v.Composing, "")
		v.Composing = ui.TextRangeEmpty
	}
	if !v.Selection.IsValid() {
		v.Selection = NewTextSelectionCollapsed(len(v.Text))
	}

	// 2. Delete the text around the cursor: the lengths don't count the
	// selection, which stays.
	if e.DeleteBefore > 0 {
		end := v.Selection.Start()
		start := alignRuneStart(v.Text, end-e.DeleteBefore)
		v = v.Replaced(ui.TextRange{Start: start, End: end}, "")
	}
	if e.DeleteAfter > 0 {
		start := v.Selection.End()
		end := alignRuneEnd(v.Text, start+e.DeleteAfter)
		v = v.Replaced(ui.TextRange{Start: start, End: end}, "")
	}

	// 3. Insert the commit string in place of the selection (or remove the selection,
	// when the event asks for it).
	if e.Commit != "" || (!v.Selection.IsCollapsed() && (e.Preedit != "" || e.DeleteSelection)) {
		start := v.Selection.Start()
		v = v.Replaced(ui.TextRange{Start: start, End: v.Selection.End()}, e.Commit)
		v.Selection = NewTextSelectionCollapsed(start + len(e.Commit))
	}

	// 4. Show the new preedit at the cursor.
	if e.Preedit != "" {
		at := v.Selection.End()
		v = v.Replaced(ui.TextRange{Start: at, End: at}, e.Preedit)
		v.Composing = ui.TextRange{Start: at, End: at + len(e.Preedit)}
		begin, end := e.PreeditCursorBegin, e.PreeditCursorEnd
		if begin < 0 || end < 0 {
			begin, end = len(e.Preedit), len(e.Preedit)
		}
		begin, end = min(begin, len(e.Preedit)), min(end, len(e.Preedit))
		v.Selection = NewTextSelection(at+begin, at+end)
	} else {
		v.Composing = ui.TextRangeEmpty
	}

	p.value = v
	p.fromInputMethod = true
	p.send()
	p.deliver(oldValue, v)
}

// deliver gives the client the new value, as Flutter's engines do: as deltas to a
// client that asked for them, and as a value to the other controls and clients.
func (p *platformTextInputControl) deliver(oldValue, newValue TextEditingValue) {
	conn := p.input.currentConnection
	if conn == nil {
		return
	}
	if d, ok := conn.client.(DeltaTextInputClient); ok && p.config.EnableDeltaModel {
		d.UpdateEditingValueWithDeltas([]TextEditingDelta{deltaBetween(oldValue, newValue)})
		return
	}
	p.input.updateEditingValue(newValue, p)
}

// deltaBetween describes the change from the old value to the new one as the
// platforms report it: the range of the old text that is replaced, and what
// replaces it.
func deltaBetween(oldValue, newValue TextEditingValue) TextEditingDelta {
	oldText, newText := oldValue.Text, newValue.Text
	if oldText == newText {
		return NewTextEditingDeltaNonTextUpdate(oldText, newValue.Selection, newValue.Composing)
	}
	prefix := 0
	for prefix < len(oldText) && prefix < len(newText) && oldText[prefix] == newText[prefix] {
		prefix++
	}
	prefix = alignRuneStart(oldText, prefix)
	suffix := 0
	for suffix < len(oldText)-prefix && suffix < len(newText)-prefix &&
		oldText[len(oldText)-1-suffix] == newText[len(newText)-1-suffix] {
		suffix++
	}
	// The common end must start at a character: a shorter end does.
	suffix = len(oldText) - alignRuneEnd(oldText, len(oldText)-suffix)
	return NewTextEditingDeltaFromReplacement(oldText, prefix, len(oldText)-suffix,
		newText[prefix:len(newText)-suffix], newValue.Selection, newValue.Composing)
}

// alignRuneStart moves the offset back to the start of the character it is in.
func alignRuneStart(text string, offset int) int {
	offset = min(max(offset, 0), len(text))
	for offset > 0 && offset < len(text) && !utf8.RuneStart(text[offset]) {
		offset--
	}
	return offset
}

// alignRuneEnd moves the offset forward to the end of the character it is in.
func alignRuneEnd(text string, offset int) int {
	offset = min(max(offset, 0), len(text))
	for offset < len(text) && !utf8.RuneStart(text[offset]) {
		offset++
	}
	return offset
}
