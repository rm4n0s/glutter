package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// BuildTextSpanFunc builds the span that a TextEditingController shows.
//
// Flutter: no counterpart
type BuildTextSpanFunc func(bc BuildContext, style *painting.TextStyle, withComposing bool) *painting.TextSpan

// TextEditingController is a controller for an editable text field: it holds the
// text, the selection and the composing range, and notifies its listeners when
// they change.
//
// Deviation: Dart subclasses override buildTextSpan; here SetBuildTextSpan
// installs the function that does (SPEC §12.10).
//
// Flutter: widgets/editable_text.dart TextEditingController
type TextEditingController struct {
	foundation.ValueNotifier[services.TextEditingValue]
	buildTextSpan BuildTextSpanFunc
}

// NewTextEditingController creates a controller with no selection that holds the
// text, which may be empty.
//
// Flutter: widgets/editable_text.dart TextEditingController.new
func NewTextEditingController(text ...string) *TextEditingController {
	c := &TextEditingController{}
	if len(text) > 0 {
		c.ValueNotifier = *foundation.NewValueNotifier(services.NewTextEditingValue(services.TextEditingValueOpts{Text: text[0]}))
	} else {
		c.ValueNotifier = *foundation.NewValueNotifier(services.TextEditingValueEmpty)
	}
	return c
}

// NewTextEditingControllerFromValue creates a controller with the value. In
// debug builds it panics with the tag TextEditingControllerComposingInvalid
// when the value has a composing range that is not valid for its text.
//
// Flutter: widgets/editable_text.dart TextEditingController.fromValue
func NewTextEditingControllerFromValue(value services.TextEditingValue) *TextEditingController {
	debugCheckComposing(value)
	return &TextEditingController{ValueNotifier: *foundation.NewValueNotifier(value)}
}

func debugCheckComposing(value services.TextEditingValue) {
	if foundation.DebugMode && value.Composing.IsValid() && !value.IsComposingRangeValid() {
		panic(errors.New(tagTextEditingControllerComposingInvalid,
			"the value has an invalid non-empty composing range; it is recommended to use a valid composing range, even for read-only text fields",
			"value", value.String(), "composing", value.Composing))
	}
}

// SetBuildTextSpan installs the function that builds the text span, in place of
// the default (which underlines the composing range); nil restores the default.
func (c *TextEditingController) SetBuildTextSpan(f BuildTextSpanFunc) { c.buildTextSpan = f }

// SetValue replaces the value; the composing range of it must be valid for its
// text (the tag TextEditingControllerComposingInvalid in debug builds).
//
// Flutter: widgets/editable_text.dart TextEditingController.value=
func (c *TextEditingController) SetValue(newValue services.TextEditingValue) {
	debugCheckComposing(newValue)
	c.ValueNotifier.SetValue(newValue)
}

// Text is the current text.
//
// Flutter: widgets/editable_text.dart TextEditingController.text
func (c *TextEditingController) Text() string { return c.Value().Text }

// SetText replaces the text and clears the selection and the composing range.
//
// Flutter: widgets/editable_text.dart TextEditingController.text=
func (c *TextEditingController) SetText(newText string) {
	c.SetValue(c.Value().CopyWith(services.TextEditingValueCopy{
		Text:      foundation.Some(newText),
		Selection: foundation.Some(services.NewTextSelectionCollapsed(-1)),
		Composing: foundation.Some(ui.TextRangeEmpty),
	}))
}

// Selection is the currently selected range of Text.
//
// Flutter: widgets/editable_text.dart TextEditingController.selection
func (c *TextEditingController) Selection() services.TextSelection { return c.Value().Selection }

// SetSelection sets the selection. It panics with the tag
// TextEditingControllerSelectionInvalid when the selection is outside the text.
// The composing range is kept only if the selection is inside it.
//
// Flutter: widgets/editable_text.dart TextEditingController.selection=
func (c *TextEditingController) SetSelection(newSelection services.TextSelection) {
	text := c.Text()
	if len(text) < newSelection.End() || len(text) < newSelection.Start() {
		panic(errors.New(tagTextEditingControllerSelectionInvalid, "invalid text selection",
			"selection", newSelection.String(), "textLength", len(text)))
	}
	composing := ui.TextRangeEmpty
	if c.isSelectionWithinComposingRange(newSelection) {
		composing = c.Value().Composing
	}
	c.SetValue(c.Value().CopyWith(services.TextEditingValueCopy{
		Selection: foundation.Some(newSelection), Composing: foundation.Some(composing),
	}))
}

// Clear sets the value to an empty text with the caret at the start.
//
// Flutter: widgets/editable_text.dart TextEditingController.clear
func (c *TextEditingController) Clear() {
	c.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{
		Selection: foundation.Some(services.NewTextSelectionCollapsed(0)),
	}))
}

// ClearComposing removes the composing range.
//
// Flutter: widgets/editable_text.dart TextEditingController.clearComposing
func (c *TextEditingController) ClearComposing() {
	c.SetValue(c.Value().CopyWith(services.TextEditingValueCopy{Composing: foundation.Some(ui.TextRangeEmpty)}))
}

func (c *TextEditingController) isSelectionWithinComposingRange(selection services.TextSelection) bool {
	v := c.Value()
	return selection.Start() >= v.Composing.Start && selection.End() <= v.Composing.End
}

// BuildTextSpan builds the span that the field shows for the text. The default
// underlines the composing range when withComposing is set.
//
// Flutter: widgets/editable_text.dart TextEditingController.buildTextSpan
func (c *TextEditingController) BuildTextSpan(bc BuildContext, style *painting.TextStyle, withComposing bool) *painting.TextSpan {
	if c.buildTextSpan != nil {
		return c.buildTextSpan(bc, style, withComposing)
	}
	value := c.Value()
	if foundation.DebugMode && value.Composing.IsValid() && withComposing && !value.IsComposingRangeValid() {
		panic(errors.New(tagTextEditingControllerComposingInvalid, "the composing range is not valid for the text",
			"composing", value.Composing, "textLength", len(value.Text)))
	}
	if !value.IsComposingRangeValid() || !withComposing {
		return painting.NewTextSpan(painting.TextSpanOpts{Style: style, Text: foundation.Some(value.Text)})
	}
	under := painting.NewTextStyle(painting.TextStyleOpts{Decoration: foundation.Some(ui.TextDecorationUnderline)})
	composingStyle := under
	if style != nil {
		composingStyle = style.Merge(under)
	}
	return painting.NewTextSpan(painting.TextSpanOpts{Style: style, Children: []painting.InlineSpan{
		painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(value.Composing.TextBefore(value.Text))}),
		painting.NewTextSpan(painting.TextSpanOpts{Style: composingStyle, Text: foundation.Some(value.Composing.TextInside(value.Text))}),
		painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(value.Composing.TextAfter(value.Text))}),
	}})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTextEditingControllerComposingInvalid = "TextEditingControllerComposingInvalid"
	tagTextEditingControllerSelectionInvalid = "TextEditingControllerSelectionInvalid"
)
