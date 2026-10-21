package web

import "github.com/rm4n0s/glutter/ui"

// ElementConfig is how the hidden text field (SPEC §23.5) is made for a text field of the framework: the
// kind of element and the attributes that tell the browser which virtual keyboard to show and how to treat
// the text. It is a plain value, so that the choice is tested without a browser.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart InputConfiguration
type ElementConfig struct {
	// Tag is the element: "textarea" for a multiline field, "input" for the others.
	Tag string
	// Type is the type attribute of an input element: "password" for hidden text, else empty.
	Type string
	// InputMode is the inputmode attribute; empty for none.
	InputMode string
	// EnterKeyHint is the enterkeyhint attribute; empty for none.
	EnterKeyHint string
	// Autocapitalize is the autocapitalize attribute: "off", "words", "characters" or "sentences".
	Autocapitalize string
	// Autocorrect is the autocorrect attribute: "on" or "off".
	Autocorrect string
}

// ElementConfigOf chooses the element and its attributes for the state of the text field. mobile says
// the browser is on iOS or Android: inputmode and enterkeyhint are set only there, where they pick the
// virtual keyboard, as Flutter web does.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart DefaultTextEditingStrategy.applyConfiguration
func ElementConfigOf(state ui.TextInputState, mobile bool) ElementConfig {
	c := ElementConfig{Tag: "input", Autocapitalize: AutocapitalizeAttribute(state.Hints), Autocorrect: "off"}
	if state.Hints.Has(ui.TextInputHintMultiline) {
		c.Tag = "textarea"
	}
	if state.Purpose == ui.TextInputPurposePassword {
		c.Type = "password"
	}
	if state.Hints.Has(ui.TextInputHintSpellcheck) {
		c.Autocorrect = "on"
	}
	if mobile {
		c.InputMode = InputModeAttribute(state.Purpose)
		c.EnterKeyHint = EnterKeyHintAttribute(state.EnterKey)
	}
	return c
}

// InputModeAttribute is the inputmode attribute for what the text field is for: the numeric, decimal, tel,
// email and url keyboards; empty for the others.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/input_type.dart EngineInputType.inputmodeAttribute
func InputModeAttribute(p ui.TextInputPurpose) string {
	switch p {
	case ui.TextInputPurposeDigits, ui.TextInputPurposePIN:
		return "numeric"
	case ui.TextInputPurposeNumber:
		return "decimal"
	case ui.TextInputPurposePhone:
		return "tel"
	case ui.TextInputPurposeEmail:
		return "email"
	case ui.TextInputPurposeURL:
		return "url"
	}
	return ""
}

// EnterKeyHintAttribute is the enterkeyhint attribute for the action of the text field; empty for the
// platform's own label.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/input_action.dart EngineInputAction.enterkeyhintAttribute
func EnterKeyHintAttribute(k ui.TextInputEnterKey) string {
	switch k {
	case ui.TextInputEnterKeyEnter:
		return "enter"
	case ui.TextInputEnterKeyDone:
		return "done"
	case ui.TextInputEnterKeyGo:
		return "go"
	case ui.TextInputEnterKeyNext:
		return "next"
	case ui.TextInputEnterKeyPrevious:
		return "previous"
	case ui.TextInputEnterKeySearch:
		return "search"
	case ui.TextInputEnterKeySend:
		return "send"
	}
	return ""
}

// AutocapitalizeAttribute is the autocapitalize attribute for how the text field capitalizes: words,
// characters or sentences, else off.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_capitalization.dart TextCapitalizationConfig.setAutocapitalizeAttribute
func AutocapitalizeAttribute(h ui.TextInputHints) string {
	switch {
	case h.Has(ui.TextInputHintTitlecase):
		return "words"
	case h.Has(ui.TextInputHintUppercase):
		return "characters"
	case h.Has(ui.TextInputHintAutoCapitalization):
		return "sentences"
	}
	return "off"
}

// FieldBox is where the hidden text field is, in CSS pixels of the page, and the size of its text.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditableTextGeometry
type FieldBox struct {
	Left, Top, Width, Height float64
	// FontSize is the font size of the field, so that the input method's window (opened below the field)
	// is as high as the caret.
	FontSize float64
}

// FieldBoxOf places the hidden text field at the caret: the caret's rectangle, in logical pixels of the
// view, moved by the position of the view in the page. The field is at least one pixel wide and high.
//
// Flutter: lib/web_ui/lib/src/engine/text_editing/text_editing.dart EditableTextGeometry.applyToDomElement
func FieldBoxOf(caret ui.Rect, originX, originY float64) FieldBox {
	h := max(caret.Height(), 1)
	return FieldBox{
		Left: originX + caret.Left, Top: originY + caret.Top,
		Width: max(caret.Width(), 1), Height: h, FontSize: h,
	}
}
