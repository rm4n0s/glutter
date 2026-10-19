package services

import (
	"fmt"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/opts"
	"github.com/rm4n0s/glutter/ui"
)

// TextEditingValue is the current text, selection and composing state of an
// editing field. Offsets are UTF-8 byte offsets (SPEC §12.10).
//
// Flutter's default value has no selection (a collapsed one at -1) and no
// composing range (TextRange.empty), which are not Go's zero values: make a
// value with NewTextEditingValue, or start from TextEditingValueEmpty. The zero
// TextEditingValue{} has a caret at 0.
//
// Deviation: Flutter's toJSON and fromJSON, which serialize the value for the
// flutter/textinput channel, don't exist (SPEC §12.10).
//
// Flutter: services/text_input.dart TextEditingValue
type TextEditingValue struct {
	// Text is the text.
	Text string
	// Selection is the range of text that is selected, or where the caret is when
	// it is collapsed. It is collapsed at -1 when there is none.
	Selection TextSelection
	// Composing is the range of text that is still being composed by the input
	// method. It is empty (-1 to -1) when there is none.
	Composing ui.TextRange
}

// TextEditingValueOpts are the optional parameters of NewTextEditingValue.
//
// Flutter: services/text_input.dart TextEditingValue.new
type TextEditingValueOpts struct {
	// Text defaults to the empty string.
	Text string
	// Selection defaults to a collapsed selection at -1.
	Selection foundation.Opt[TextSelection]
	// Composing defaults to ui.TextRangeEmpty.
	Composing foundation.Opt[ui.TextRange]
}

// NewTextEditingValue creates a value; it has no selection and no composing
// range unless the options give them.
//
// Flutter: services/text_input.dart TextEditingValue.new
func NewTextEditingValue(o ...TextEditingValueOpts) TextEditingValue {
	op := first(o)
	return TextEditingValue{
		Text:      op.Text,
		Selection: op.Selection.OrElse(NewTextSelectionCollapsed(-1)),
		Composing: op.Composing.OrElse(ui.TextRangeEmpty),
	}
}

// TextEditingValueEmpty is a value with no text, no selection and no composing
// range.
//
// Flutter: services/text_input.dart TextEditingValue.empty
var TextEditingValueEmpty = NewTextEditingValue()

// TextEditingValueCopy holds the values TextEditingValue.CopyWith replaces.
//
// Flutter: services/text_input.dart TextEditingValue.copyWith
type TextEditingValueCopy struct {
	Text      foundation.Opt[string]
	Selection foundation.Opt[TextSelection]
	Composing foundation.Opt[ui.TextRange]
}

// CopyWith returns a copy with the given values replaced.
//
// Flutter: services/text_input.dart TextEditingValue.copyWith
func (v TextEditingValue) CopyWith(c TextEditingValueCopy) TextEditingValue {
	return TextEditingValue{
		Text:      c.Text.OrElse(v.Text),
		Selection: c.Selection.OrElse(v.Selection),
		Composing: c.Composing.OrElse(v.Composing),
	}
}

// IsComposingRangeValid is whether the composing range is a valid range of the
// text: not empty-invalid, normalized and inside the text.
//
// Flutter: services/text_input.dart TextEditingValue.isComposingRangeValid
func (v TextEditingValue) IsComposingRangeValid() bool {
	return v.Composing.IsValid() && v.Composing.IsNormalized() && v.Composing.End <= len(v.Text)
}

// Replaced returns the value with the text in the range replaced by the string,
// and the selection and composing range moved to follow the text they were on.
// An invalid range changes nothing. A range outside the text panics with the
// tag TextEditingValueRangeOutOfBounds.
//
// Flutter: services/text_input.dart TextEditingValue.replaced
func (v TextEditingValue) Replaced(replacementRange ui.TextRange, replacementString string) TextEditingValue {
	if !replacementRange.IsValid() {
		return v
	}
	newText := replaceRange(v.Text, replacementRange, replacementString)
	if replacementRange.End-replacementRange.Start == len(replacementString) {
		return v.CopyWith(TextEditingValueCopy{Text: foundation.Some(newText)})
	}
	adjustIndex := func(originalIndex int) int {
		// The length added by adding the replacementString.
		replacedLength := len(replacementString)
		if originalIndex <= replacementRange.Start && originalIndex < replacementRange.End {
			replacedLength = 0
		}
		// The length removed by removing the replacementRange.
		removedLength := min(max(originalIndex, replacementRange.Start), replacementRange.End) - replacementRange.Start
		return originalIndex + replacedLength - removedLength
	}
	selection := NewTextSelection(adjustIndex(v.Selection.BaseOffset), adjustIndex(v.Selection.ExtentOffset))
	composing := ui.TextRange{Start: adjustIndex(v.Composing.Start), End: adjustIndex(v.Composing.End)}
	debugCheckRange("TextEditingValueRangeOutOfBounds", "the adjusted selection", selection.Range(), newText)
	debugCheckRange("TextEditingValueRangeOutOfBounds", "the adjusted composing range", composing, newText)
	return TextEditingValue{Text: newText, Selection: selection, Composing: composing}
}

// Equal reports whether two values have the same text, selection and composing
// range.
//
// Flutter: services/text_input.dart TextEditingValue.==
func (v TextEditingValue) Equal(other TextEditingValue) bool {
	return v.Text == other.Text && v.Selection.Equal(other.Selection) && v.Composing == other.Composing
}

// String returns what Dart prints for the value.
func (v TextEditingValue) String() string {
	return fmt.Sprintf("TextEditingValue(text: ┤%s├, selection: %v, composing: %v)", v.Text, v.Selection, v.Composing)
}

// replaceRange is Dart's String.replaceRange. A range outside the text panics
// with the tag TextEditingValueRangeOutOfBounds.
func replaceRange(text string, r ui.TextRange, replacement string) string {
	if r.Start < 0 || r.End < r.Start || r.End > len(text) {
		panic(errors.New("TextEditingValueRangeOutOfBounds", "the range is not inside the text",
			"start", r.Start, "end", r.End, "length", len(text)))
	}
	return text[:r.Start] + replacement + text[r.End:]
}

// debugCheckRange is Dart's _textRangeIsValid assertion: in a debug build, a
// valid range that is not inside the text panics with the tag.
func debugCheckRange(tag, what string, r ui.TextRange, text string) {
	if !foundation.DebugMode {
		return
	}
	if r.Start == -1 && r.End == -1 {
		return
	}
	if r.Start < 0 || r.Start > len(text) || r.End < 0 || r.End > len(text) {
		panic(errors.New(tag, what+" is out of the text", "start", r.Start, "end", r.End, "length", len(text)))
	}
}

// first returns the one options value of a trailing opts parameter.
func first[T any](o []T) T { return opts.First(o) }
