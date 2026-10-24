// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// SpellCheckConfigurationOpts are the optional parameters of
// NewSpellCheckConfiguration.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.new
type SpellCheckConfigurationOpts struct {
	// SpellCheckService is the service used to fetch the spell check results for
	// the text of the field. If it is nil, the DefaultSpellCheckService is used when
	// the platform has a spell checker (Android), and spell check is disabled
	// otherwise.
	SpellCheckService services.SpellCheckService
	// MisspelledSelectionColor is the color the selection is painted in while the
	// spell check suggestions toolbar is shown; the default is the selection color
	// of the field.
	MisspelledSelectionColor foundation.Opt[ui.Color]
	// MisspelledTextStyle is the style that misspelled words are shown in. The field
	// needs it when spell check is enabled; a TextField gives the Material one
	// (TextFieldMaterialMisspelledTextStyle) when it is nil.
	MisspelledTextStyle *painting.TextStyle
	// SpellCheckSuggestionsToolbarBuilder builds the toolbar with the suggestions
	// for a misspelled word; a TextField gives the Material one
	// (TextFieldDefaultSpellCheckSuggestionsToolbarBuilder) when it is nil.
	SpellCheckSuggestionsToolbarBuilder EditableTextContextMenuBuilder
}

// SpellCheckConfiguration controls how spell check is performed for a text
// field: the service, the style of the misspelled words and the toolbar with the
// suggestions. It is immutable.
//
// Deviation: Dart's operator == compares the toolbar builder by identity; Go
// cannot compare functions, so Equal compares the code that the functions run
// (SPEC §12.10).
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration
type SpellCheckConfiguration struct {
	spellCheckService                   services.SpellCheckService
	misspelledSelectionColor            foundation.Opt[ui.Color]
	misspelledTextStyle                 *painting.TextStyle
	spellCheckSuggestionsToolbarBuilder EditableTextContextMenuBuilder
	spellCheckEnabled                   bool
}

// NewSpellCheckConfiguration creates a configuration that enables spell check.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.new
func NewSpellCheckConfiguration(opts ...SpellCheckConfigurationOpts) *SpellCheckConfiguration {
	o := firstOpts(opts)
	return &SpellCheckConfiguration{
		spellCheckService: o.SpellCheckService, misspelledSelectionColor: o.MisspelledSelectionColor,
		misspelledTextStyle: o.MisspelledTextStyle, spellCheckSuggestionsToolbarBuilder: o.SpellCheckSuggestionsToolbarBuilder,
		spellCheckEnabled: true,
	}
}

// NewSpellCheckConfigurationDisabled creates a configuration that disables spell
// check.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.disabled
func NewSpellCheckConfigurationDisabled() *SpellCheckConfiguration {
	return &SpellCheckConfiguration{}
}

// SpellCheckService is the service used to fetch the spell check results for the
// text of the field; nil when the field is to infer it.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.spellCheckService
func (c *SpellCheckConfiguration) SpellCheckService() services.SpellCheckService {
	return c.spellCheckService
}

// MisspelledSelectionColor is the color the selection is painted in while the
// spell check suggestions toolbar is shown.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.misspelledSelectionColor
func (c *SpellCheckConfiguration) MisspelledSelectionColor() foundation.Opt[ui.Color] {
	return c.misspelledSelectionColor
}

// MisspelledTextStyle is the style that misspelled words are shown in.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.misspelledTextStyle
func (c *SpellCheckConfiguration) MisspelledTextStyle() *painting.TextStyle {
	return c.misspelledTextStyle
}

// SpellCheckSuggestionsToolbarBuilder builds the toolbar with the suggestions for
// a misspelled word.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.spellCheckSuggestionsToolbarBuilder
func (c *SpellCheckConfiguration) SpellCheckSuggestionsToolbarBuilder() EditableTextContextMenuBuilder {
	return c.spellCheckSuggestionsToolbarBuilder
}

// SpellCheckEnabled is whether spell check is enabled: false for the configuration
// made by NewSpellCheckConfigurationDisabled.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.spellCheckEnabled
func (c *SpellCheckConfiguration) SpellCheckEnabled() bool { return c.spellCheckEnabled }

// SpellCheckConfigurationCopy are the values CopyWith replaces: a nil or unset
// value keeps the one of the configuration.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.copyWith
type SpellCheckConfigurationCopy struct {
	SpellCheckService                   services.SpellCheckService
	MisspelledSelectionColor            foundation.Opt[ui.Color]
	MisspelledTextStyle                 *painting.TextStyle
	SpellCheckSuggestionsToolbarBuilder EditableTextContextMenuBuilder
}

// CopyWith returns a copy of the configuration with the given values replaced. A
// disabled configuration stays disabled: a new configuration has to be made to
// enable spell check.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.copyWith
func (c *SpellCheckConfiguration) CopyWith(v SpellCheckConfigurationCopy) *SpellCheckConfiguration {
	if !c.spellCheckEnabled {
		// A new configuration should be constructed to enable spell check.
		return NewSpellCheckConfigurationDisabled()
	}
	n := NewSpellCheckConfiguration(SpellCheckConfigurationOpts{
		SpellCheckService: c.spellCheckService, MisspelledSelectionColor: c.misspelledSelectionColor,
		MisspelledTextStyle: c.misspelledTextStyle, SpellCheckSuggestionsToolbarBuilder: c.spellCheckSuggestionsToolbarBuilder,
	})
	if v.SpellCheckService != nil {
		n.spellCheckService = v.SpellCheckService
	}
	if v.MisspelledSelectionColor.IsSome() {
		n.misspelledSelectionColor = v.MisspelledSelectionColor
	}
	if v.MisspelledTextStyle != nil {
		n.misspelledTextStyle = v.MisspelledTextStyle
	}
	if v.SpellCheckSuggestionsToolbarBuilder != nil {
		n.spellCheckSuggestionsToolbarBuilder = v.SpellCheckSuggestionsToolbarBuilder
	}
	return n
}

// String describes the configuration.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.toString
func (c *SpellCheckConfiguration) String() string {
	state := "disabled"
	if c.spellCheckEnabled {
		state = "enabled"
	}
	style := "null"
	if c.misspelledTextStyle != nil {
		style = c.misspelledTextStyle.String()
	}
	service := "null"
	if c.spellCheckService != nil {
		service = fmt.Sprint(c.spellCheckService)
	}
	builder := "null"
	if c.spellCheckSuggestionsToolbarBuilder != nil {
		builder = "Closure"
	}
	return fmt.Sprintf("SpellCheckConfiguration(%s, service: %s, text style: %s, toolbar builder: %s)", state, service, style, builder)
}

// Equal reports whether the configurations are the same: both enabled or both
// disabled, with the same service, the same misspelled text style and the same
// toolbar builder. Two nil configurations are the same.
//
// Flutter: widgets/spell_check.dart SpellCheckConfiguration.==
func (c *SpellCheckConfiguration) Equal(other *SpellCheckConfiguration) bool {
	if c == nil || other == nil {
		return c == other
	}
	return sameSpellCheckService(c.spellCheckService, other.spellCheckService) &&
		c.misspelledTextStyle.Equal(other.misspelledTextStyle) &&
		sameContextMenuBuilder(c.spellCheckSuggestionsToolbarBuilder, other.spellCheckSuggestionsToolbarBuilder) &&
		c.spellCheckEnabled == other.spellCheckEnabled
}

// sameSpellCheckService is Dart's == on two services: the same object.
func sameSpellCheckService(a, b services.SpellCheckService) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ta := reflect.TypeOf(a)
	if ta != reflect.TypeOf(b) || !ta.Comparable() {
		return false
	}
	return a == b
}

// sameContextMenuBuilder says whether two builders run the same code.
func sameContextMenuBuilder(a, b EditableTextContextMenuBuilder) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}

// Methods for displaying spell check results:

// correctSpellCheckResults adjusts the spans of results that are for resultsText
// to the newText, which the field has now, dropping the spans whose word is not in
// the new text.
//
// Flutter: widgets/spell_check.dart _correctSpellCheckResults
func correctSpellCheckResults(newText, resultsText string, results []services.SuggestionSpan) []services.SuggestionSpan {
	correctedSpellCheckResults := []services.SuggestionSpan{}
	spanPointer := 0
	offset := 0

	// Assumes that the order of spans has not been jumbled for optimization
	// purposes, and will only search since the previously found span.
	searchStart := 0

	for spanPointer < len(results) {
		currentSpan := results[spanPointer]
		currentSpanText := resultsText[currentSpan.Range.Start:currentSpan.Range.End]
		spanLength := currentSpan.Range.End - currentSpan.Range.Start

		// Try finding SuggestionSpan from resultsText in new text.
		currentSpanTextRegexp := regexp.MustCompile(`\b` + regexp.QuoteMeta(currentSpanText) + `\b`)
		foundIndex := -1
		if loc := currentSpanTextRegexp.FindStringIndex(newText[searchStart:]); loc != nil {
			foundIndex = loc[0]
		}

		// Check whether word was found exactly where expected or elsewhere in the newText.
		currentSpanFoundExactly := currentSpan.Range.Start == foundIndex+searchStart
		currentSpanFoundExactlyWithOffset := currentSpan.Range.Start+offset == foundIndex+searchStart
		currentSpanFoundElsewhere := foundIndex >= 0

		if currentSpanFoundExactly || currentSpanFoundExactlyWithOffset {
			// currentSpan was found at the same index in newText and resultsText
			// or at the same index with the previously calculated adjustment by
			// the offset value, so apply it to new text by adding it to the list of
			// corrected results.
			adjustedSpan := services.SuggestionSpan{
				Range:       ui.TextRange{Start: currentSpan.Range.Start + offset, End: currentSpan.Range.End + offset},
				Suggestions: currentSpan.Suggestions,
			}

			// Start search for the next misspelled word at the end of currentSpan.
			searchStart = clampSearchStart(currentSpan.Range.End+1+offset, len(newText))
			correctedSpellCheckResults = append(correctedSpellCheckResults, adjustedSpan)
		} else if currentSpanFoundElsewhere {
			// Word was pushed forward but not modified.
			adjustedSpanStart := searchStart + foundIndex
			adjustedSpanEnd := adjustedSpanStart + spanLength
			adjustedSpan := services.SuggestionSpan{
				Range:       ui.TextRange{Start: adjustedSpanStart, End: adjustedSpanEnd},
				Suggestions: currentSpan.Suggestions,
			}

			// Start search for the next misspelled word at the end of the
			// adjusted currentSpan.
			searchStart = clampSearchStart(adjustedSpanEnd+1, len(newText))
			// Adjust offset to reflect the difference between where currentSpan
			// was positioned in resultsText versus in newText.
			offset = adjustedSpanStart - currentSpan.Range.Start
			correctedSpellCheckResults = append(correctedSpellCheckResults, adjustedSpan)
		}
		spanPointer++
	}
	return correctedSpellCheckResults
}

// clampSearchStart is Dart's math.min(start, length), kept inside the text.
func clampSearchStart(start, length int) int {
	return max(0, min(start, length))
}

// BuildTextSpanWithSpellCheckSuggestions builds the TextSpan tree for the text of
// the value, with the misspelled words of the spell check results in the
// misspelled text style. When the text changed since the results were made, the
// spans are corrected first, or dropped for a word that changed. On Android the
// tree also underlines the composing region, unless composingWithinCurrentTextRange.
//
// Flutter: widgets/spell_check.dart buildTextSpanWithSpellCheckSuggestions
func BuildTextSpanWithSpellCheckSuggestions(value services.TextEditingValue, composingWithinCurrentTextRange bool,
	style, misspelledTextStyle *painting.TextStyle, spellCheckResults services.SpellCheckResults) *painting.TextSpan {
	spellCheckResultsSpans := spellCheckResults.SuggestionSpans
	spellCheckResultsText := spellCheckResults.SpellCheckedText

	if spellCheckResultsText != value.Text {
		spellCheckResultsSpans = correctSpellCheckResults(value.Text, spellCheckResultsText, spellCheckResultsSpans)
	}

	// We will draw the TextSpan tree based on the composing region, if it is
	// available.
	shouldConsiderComposingRegion := foundation.DefaultTargetPlatform() == foundation.TargetPlatformAndroid
	var children []*painting.TextSpan
	if shouldConsiderComposingRegion {
		children = buildSubtreesWithComposingRegion(spellCheckResultsSpans, value, style, misspelledTextStyle, composingWithinCurrentTextRange)
	} else {
		children = buildSubtreesWithoutComposingRegion(spellCheckResultsSpans, value, style, misspelledTextStyle, value.Selection.BaseOffset)
	}
	return painting.NewTextSpan(painting.TextSpanOpts{Style: style, Children: textSpanChildren(children)})
}

func textSpanChildren(spans []*painting.TextSpan) []painting.InlineSpan {
	out := make([]painting.InlineSpan, len(spans))
	for i, s := range spans {
		out[i] = s
	}
	return out
}

// mergeStyle is Dart's style?.merge(other) ?? other.
func mergeStyle(style, other *painting.TextStyle) *painting.TextStyle {
	if style != nil {
		return style.Merge(other)
	}
	return other
}

func spellCheckTextSpan(style *painting.TextStyle, text string) *painting.TextSpan {
	return painting.NewTextSpan(painting.TextSpanOpts{Style: style, Text: foundation.Some(text)})
}

// buildSubtreesWithoutComposingRegion builds the children of the tree for a
// platform that does not mark the composing region: the word that has the
// cursor is not marked.
//
// Flutter: widgets/spell_check.dart _buildSubtreesWithoutComposingRegion
func buildSubtreesWithoutComposingRegion(spellCheckSuggestions []services.SuggestionSpan, value services.TextEditingValue,
	style, misspelledStyle *painting.TextStyle, cursorIndex int) []*painting.TextSpan {
	textSpanTreeChildren := []*painting.TextSpan{}

	textPointer := 0
	currentSpanPointer := 0
	text := value.Text
	misspelledJointStyle := mergeStyle(style, misspelledStyle)

	// Add text interwoven with any misspelled words to the tree.
	if spellCheckSuggestions != nil {
		for textPointer < len(text) && currentSpanPointer < len(spellCheckSuggestions) {
			currentSpan := spellCheckSuggestions[currentSpanPointer]

			if currentSpan.Range.Start > textPointer {
				endIndex := min(currentSpan.Range.Start, len(text))
				textSpanTreeChildren = append(textSpanTreeChildren, spellCheckTextSpan(style, text[textPointer:endIndex]))
				textPointer = endIndex
			} else {
				endIndex := min(currentSpan.Range.End, len(text))
				cursorInCurrentSpan := currentSpan.Range.Start <= cursorIndex && currentSpan.Range.End >= cursorIndex
				spanStyle := misspelledJointStyle
				if cursorInCurrentSpan {
					spanStyle = style
				}
				textSpanTreeChildren = append(textSpanTreeChildren, spellCheckTextSpan(spanStyle, text[currentSpan.Range.Start:endIndex]))

				textPointer = endIndex
				currentSpanPointer++
			}
		}
	}

	// Add any remaining text to the tree if applicable.
	if textPointer < len(text) {
		textSpanTreeChildren = append(textSpanTreeChildren, spellCheckTextSpan(style, text[textPointer:]))
	}

	return textSpanTreeChildren
}

// buildSubtreesWithComposingRegion builds the children of the tree for a
// platform that marks the composing region (Android): the composing region is
// underlined, and it is not marked as misspelled.
//
// Flutter: widgets/spell_check.dart _buildSubtreesWithComposingRegion
func buildSubtreesWithComposingRegion(spellCheckSuggestions []services.SuggestionSpan, value services.TextEditingValue,
	style, misspelledStyle *painting.TextStyle, composingWithinCurrentTextRange bool) []*painting.TextSpan {
	textSpanTreeChildren := []*painting.TextSpan{}

	textPointer := 0
	currentSpanPointer := 0
	text := value.Text
	composingRegion := value.Composing
	composingTextStyle := mergeStyle(style, painting.NewTextStyle(painting.TextStyleOpts{Decoration: foundation.Some(ui.TextDecorationUnderline)}))
	misspelledJointStyle := mergeStyle(style, misspelledStyle)

	// Add text interwoven with any misspelled words to the tree.
	if spellCheckSuggestions != nil {
		for textPointer < len(text) && currentSpanPointer < len(spellCheckSuggestions) {
			currentSpan := spellCheckSuggestions[currentSpanPointer]

			if currentSpan.Range.Start > textPointer {
				endIndex := min(currentSpan.Range.Start, len(text))
				textPointerWithinComposingRegion := composingRegion.Start >= textPointer &&
					composingRegion.End <= endIndex && !composingWithinCurrentTextRange

				if textPointerWithinComposingRegion {
					textSpanTreeChildren = addComposingRegionTextSpans(textSpanTreeChildren, text, textPointer, composingRegion, style, composingTextStyle)
					textSpanTreeChildren = append(textSpanTreeChildren, spellCheckTextSpan(style, text[composingRegion.End:endIndex]))
				} else {
					textSpanTreeChildren = append(textSpanTreeChildren, spellCheckTextSpan(style, text[textPointer:endIndex]))
				}

				textPointer = endIndex
			} else {
				endIndex := min(currentSpan.Range.End, len(text))
				currentSpanIsComposingRegion := textPointer >= composingRegion.Start &&
					endIndex <= composingRegion.End && !composingWithinCurrentTextRange
				spanStyle := misspelledJointStyle
				if currentSpanIsComposingRegion {
					spanStyle = composingTextStyle
				}
				textSpanTreeChildren = append(textSpanTreeChildren, spellCheckTextSpan(spanStyle, text[currentSpan.Range.Start:endIndex]))

				textPointer = endIndex
				currentSpanPointer++
			}
		}
	}

	// Add any remaining text to the tree if applicable.
	if textPointer < len(text) {
		if textPointer < composingRegion.Start && !composingWithinCurrentTextRange {
			textSpanTreeChildren = addComposingRegionTextSpans(textSpanTreeChildren, text, textPointer, composingRegion, style, composingTextStyle)

			if composingRegion.End != len(text) {
				textSpanTreeChildren = append(textSpanTreeChildren, spellCheckTextSpan(style, text[composingRegion.End:]))
			}
		} else {
			textSpanTreeChildren = append(textSpanTreeChildren, spellCheckTextSpan(style, text[textPointer:]))
		}
	}

	return textSpanTreeChildren
}

// addComposingRegionTextSpans adds the text before the composing region and the
// composing region to the children.
//
// Flutter: widgets/spell_check.dart _addComposingRegionTextSpans
func addComposingRegionTextSpans(treeChildren []*painting.TextSpan, text string, start int, composingRegion ui.TextRange,
	style, composingTextStyle *painting.TextStyle) []*painting.TextSpan {
	treeChildren = append(treeChildren, spellCheckTextSpan(style, text[start:composingRegion.Start]))
	return append(treeChildren, spellCheckTextSpan(composingTextStyle, text[composingRegion.Start:composingRegion.End]))
}
