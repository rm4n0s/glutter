// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

import (
	"unicode/utf8"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// TextEditingDelta is a change to the text of an editing field, with the
// selection and composing range that follow it. The delta types are
// TextEditingDeltaInsertion, TextEditingDeltaDeletion,
// TextEditingDeltaReplacement and TextEditingDeltaNonTextUpdate. Offsets are
// UTF-8 byte offsets (SPEC §12.10).
//
// Flutter: services/text_editing_delta.dart TextEditingDelta
type TextEditingDelta interface {
	// OldText is the text before the change.
	//
	// Flutter: services/text_editing_delta.dart TextEditingDelta.oldText
	OldText() string
	// Selection is the selection after the change.
	//
	// Flutter: services/text_editing_delta.dart TextEditingDelta.selection
	Selection() TextSelection
	// Composing is the composing range after the change.
	//
	// Flutter: services/text_editing_delta.dart TextEditingDelta.composing
	Composing() ui.TextRange
	// Apply applies the change to the value. The delta applies to its own
	// OldText ("last write wins", as the platform and the framework talk
	// asynchronously), so the text of the value is only kept for its type.
	//
	// Flutter: services/text_editing_delta.dart TextEditingDelta.apply
	Apply(value TextEditingValue) TextEditingValue
	foundation.Diagnosticable
}

type deltaBase struct {
	oldText   string
	selection TextSelection
	composing ui.TextRange
}

func (d deltaBase) OldText() string          { return d.oldText }
func (d deltaBase) Selection() TextSelection { return d.selection }
func (d deltaBase) Composing() ui.TextRange  { return d.composing }

func (d deltaBase) applyTo(value TextEditingValue, newText, tag, kind string) TextEditingValue {
	debugCheckRange(tag, "Applying "+kind+" failed, the selection", d.selection.Range(), newText)
	debugCheckRange(tag, "Applying "+kind+" failed, the composing range", d.composing, newText)
	return value.CopyWith(TextEditingValueCopy{
		Text: foundation.Some(newText), Selection: foundation.Some(d.selection), Composing: foundation.Some(d.composing),
	})
}

func (d deltaBase) fill(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("oldText", d.oldText))
}

func (d deltaBase) fillEnd(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("selection", d.selection))
	p.Add(foundation.NewDiagnosticsProperty("composing", d.composing))
}

// TextEditingDeltaInsertion is a text inserted into the field.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaInsertion
type TextEditingDeltaInsertion struct {
	deltaBase
	textInserted    string
	insertionOffset int
}

// NewTextEditingDeltaInsertion creates the delta of an insertion of
// textInserted into oldText at insertionOffset.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaInsertion.new
func NewTextEditingDeltaInsertion(oldText, textInserted string, insertionOffset int, selection TextSelection, composing ui.TextRange) TextEditingDeltaInsertion {
	return TextEditingDeltaInsertion{deltaBase{oldText, selection, composing}, textInserted, insertionOffset}
}

// TextInserted is the text that was inserted.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaInsertion.textInserted
func (d TextEditingDeltaInsertion) TextInserted() string { return d.textInserted }

// InsertionOffset is where in OldText the text was inserted.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaInsertion.insertionOffset
func (d TextEditingDeltaInsertion) InsertionOffset() int { return d.insertionOffset }

// Apply applies the insertion to the old text. An offset, selection or
// composing range outside the text panics in a debug build with the tag
// TextEditingDeltaInsertionInvalid.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaInsertion.apply
func (d TextEditingDeltaInsertion) Apply(value TextEditingValue) TextEditingValue {
	const tag = "TextEditingDeltaInsertionInvalid"
	debugCheckRange(tag, "Applying TextEditingDeltaInsertion failed, the insertionOffset", ui.NewTextRangeCollapsed(d.insertionOffset), d.oldText)
	newText := replaceRange(d.oldText, ui.TextRange{Start: d.insertionOffset, End: d.insertionOffset}, d.textInserted)
	return d.applyTo(value, newText, tag, "TextEditingDeltaInsertion")
}

// DebugFillProperties adds the properties of the delta.
func (d TextEditingDeltaInsertion) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	d.fill(p)
	p.Add(foundation.NewDiagnosticsProperty("textInserted", d.textInserted))
	p.Add(foundation.NewDiagnosticsProperty("insertionOffset", d.insertionOffset))
	d.fillEnd(p)
}

// TextEditingDeltaDeletion is a text deleted from the field.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaDeletion
type TextEditingDeltaDeletion struct {
	deltaBase
	deletedRange ui.TextRange
}

// NewTextEditingDeltaDeletion creates the delta of a deletion of deletedRange
// from oldText.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaDeletion.new
func NewTextEditingDeltaDeletion(oldText string, deletedRange ui.TextRange, selection TextSelection, composing ui.TextRange) TextEditingDeltaDeletion {
	return TextEditingDeltaDeletion{deltaBase{oldText, selection, composing}, deletedRange}
}

// DeletedRange is the range of OldText that was deleted.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaDeletion.deletedRange
func (d TextEditingDeltaDeletion) DeletedRange() ui.TextRange { return d.deletedRange }

// TextDeleted is the text that was deleted.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaDeletion.textDeleted
func (d TextEditingDeltaDeletion) TextDeleted() string {
	return d.oldText[d.deletedRange.Start:d.deletedRange.End]
}

// Apply applies the deletion to the old text. A range, selection or composing
// range outside the text panics in a debug build with the tag
// TextEditingDeltaDeletionInvalid.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaDeletion.apply
func (d TextEditingDeltaDeletion) Apply(value TextEditingValue) TextEditingValue {
	const tag = "TextEditingDeltaDeletionInvalid"
	debugCheckRange(tag, "Applying TextEditingDeltaDeletion failed, the deletedRange", d.deletedRange, d.oldText)
	newText := replaceRange(d.oldText, d.deletedRange, "")
	return d.applyTo(value, newText, tag, "TextEditingDeltaDeletion")
}

// DebugFillProperties adds the properties of the delta.
func (d TextEditingDeltaDeletion) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	d.fill(p)
	p.Add(foundation.NewDiagnosticsProperty("textDeleted", d.TextDeleted()))
	p.Add(foundation.NewDiagnosticsProperty("deletedRange", d.deletedRange))
	d.fillEnd(p)
}

// TextEditingDeltaReplacement is a text replaced by another in the field.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaReplacement
type TextEditingDeltaReplacement struct {
	deltaBase
	replacementText string
	replacedRange   ui.TextRange
}

// NewTextEditingDeltaReplacement creates the delta of a replacement of
// replacedRange of oldText with replacementText.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaReplacement.new
func NewTextEditingDeltaReplacement(oldText, replacementText string, replacedRange ui.TextRange, selection TextSelection, composing ui.TextRange) TextEditingDeltaReplacement {
	return TextEditingDeltaReplacement{deltaBase{oldText, selection, composing}, replacementText, replacedRange}
}

// ReplacementText is the text that replaced the old one.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaReplacement.replacementText
func (d TextEditingDeltaReplacement) ReplacementText() string { return d.replacementText }

// ReplacedRange is the range of OldText that was replaced.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaReplacement.replacedRange
func (d TextEditingDeltaReplacement) ReplacedRange() ui.TextRange { return d.replacedRange }

// TextReplaced is the text that was replaced.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaReplacement.textReplaced
func (d TextEditingDeltaReplacement) TextReplaced() string {
	return d.oldText[d.replacedRange.Start:d.replacedRange.End]
}

// Apply applies the replacement to the old text. A range, selection or
// composing range outside the text panics in a debug build with the tag
// TextEditingDeltaReplacementInvalid.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaReplacement.apply
func (d TextEditingDeltaReplacement) Apply(value TextEditingValue) TextEditingValue {
	const tag = "TextEditingDeltaReplacementInvalid"
	debugCheckRange(tag, "Applying TextEditingDeltaReplacement failed, the replacedRange", d.replacedRange, d.oldText)
	newText := replaceRange(d.oldText, d.replacedRange, d.replacementText)
	return d.applyTo(value, newText, tag, "TextEditingDeltaReplacement")
}

// DebugFillProperties adds the properties of the delta.
func (d TextEditingDeltaReplacement) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	d.fill(p)
	p.Add(foundation.NewDiagnosticsProperty("textReplaced", d.TextReplaced()))
	p.Add(foundation.NewDiagnosticsProperty("replacementText", d.replacementText))
	p.Add(foundation.NewDiagnosticsProperty("replacedRange", d.replacedRange))
	d.fillEnd(p)
}

// TextEditingDeltaNonTextUpdate is a change of the selection or the composing
// range that does not change the text.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaNonTextUpdate
type TextEditingDeltaNonTextUpdate struct {
	deltaBase
}

// NewTextEditingDeltaNonTextUpdate creates the delta of a change of the
// selection and the composing range of oldText.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaNonTextUpdate.new
func NewTextEditingDeltaNonTextUpdate(oldText string, selection TextSelection, composing ui.TextRange) TextEditingDeltaNonTextUpdate {
	return TextEditingDeltaNonTextUpdate{deltaBase{oldText, selection, composing}}
}

// Apply returns the old text with the new selection and composing range. A
// selection or composing range outside the text panics in a debug build with
// the tag TextEditingDeltaNonTextUpdateInvalid.
//
// Flutter: services/text_editing_delta.dart TextEditingDeltaNonTextUpdate.apply
func (d TextEditingDeltaNonTextUpdate) Apply(TextEditingValue) TextEditingValue {
	const tag = "TextEditingDeltaNonTextUpdateInvalid"
	debugCheckRange(tag, "Applying TextEditingDeltaNonTextUpdate failed, the selection", d.selection.Range(), d.oldText)
	debugCheckRange(tag, "Applying TextEditingDeltaNonTextUpdate failed, the composing region", d.composing, d.oldText)
	return TextEditingValue{Text: d.oldText, Selection: d.selection, Composing: d.composing}
}

// DebugFillProperties adds the properties of the delta.
func (d TextEditingDeltaNonTextUpdate) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	d.fill(p)
	d.fillEnd(p)
}

// NewTextEditingDeltaFromReplacement works out the kind of change that replaced
// the range deltaStart to deltaEnd of oldText with deltaText, and returns its
// delta: this is what Flutter's TextEditingDelta.fromJSON does with a change
// that a platform reports.
//
// Platforms report the whole composing region as replaced, not the characters
// that changed in it. The change is an insertion or a deletion when the text of
// the original composing region did not change; a replacement when it did; and
// a non-text update when the text is the same (or deltaStart and deltaEnd are
// both -1). A range outside oldText, or a selection or composing range outside
// the new text, panics in a debug build with the tag
// TextEditingDeltaRangeInvalid.
//
// Deviation: a single deleted character is one UTF-8 character, not one byte
// (Flutter counts UTF-16 code units), so deleting the last character of the
// composing region deletes a whole character (SPEC §12.10).
//
// Flutter: services/text_editing_delta.dart TextEditingDelta.fromJSON
func NewTextEditingDeltaFromReplacement(oldText string, deltaStart, deltaEnd int, deltaText string, selection TextSelection, composing ui.TextRange) TextEditingDelta {
	const tag = "TextEditingDeltaRangeInvalid"
	replacementDestinationStart, replacementDestinationEnd := deltaStart, deltaEnd
	replacementSource := deltaText
	replacementSourceEnd := len(replacementSource)

	// This delta is explicitly a non text update.
	if replacementDestinationStart == -1 && replacementDestinationStart == replacementDestinationEnd {
		debugCheckRange(tag, "The selection range "+selection.String()+" is not within the bounds of the text", selection.Range(), oldText)
		debugCheckRange(tag, "The composing range "+composing.String()+" is not within the bounds of the text", composing, oldText)
		return NewTextEditingDeltaNonTextUpdate(oldText, selection, composing)
	}

	debugCheckRange(tag, "The delta range is not within the bounds of the text",
		ui.TextRange{Start: replacementDestinationStart, End: replacementDestinationEnd}, oldText)
	newText := replaceRange(oldText, ui.TextRange{Start: replacementDestinationStart, End: replacementDestinationEnd}, replacementSource)
	debugCheckRange(tag, "The selection range "+selection.String()+" is not within the bounds of the new text", selection.Range(), newText)
	debugCheckRange(tag, "The composing range "+composing.String()+" is not within the bounds of the new text", composing, newText)

	isEqual := oldText == newText

	// The size of the deleted text counts in characters, not bytes, for the one
	// character case: the length of the last character of the replaced text.
	destinationLength := replacementDestinationEnd - replacementDestinationStart
	lastCharLength := 1
	if destinationLength > 0 {
		_, size := utf8.DecodeLastRuneInString(oldText[replacementDestinationStart:replacementDestinationEnd])
		lastCharLength = size
	}
	isDeletionGreaterThanOne := destinationLength-replacementSourceEnd > lastCharLength
	isDeletingByReplacingWithEmpty := replacementSource == ""
	isReplacedByShorter := isDeletionGreaterThanOne && replacementSourceEnd < destinationLength
	isReplacedByLonger := replacementSourceEnd > destinationLength
	isReplacedBySame := replacementSourceEnd == destinationLength
	isInsertingInsideComposingRegion := replacementDestinationStart+replacementSourceEnd > replacementDestinationEnd
	isDeletingInsideComposingRegion := !isReplacedByShorter && !isDeletingByReplacingWithEmpty &&
		replacementDestinationStart+replacementSourceEnd < replacementDestinationEnd

	var newComposingText, originalComposingText string
	if isDeletingByReplacingWithEmpty || isDeletingInsideComposingRegion || isReplacedByShorter {
		newComposingText = replacementSource[:replacementSourceEnd]
		originalComposingText = oldText[replacementDestinationStart : replacementDestinationStart+replacementSourceEnd]
	} else {
		newComposingText = replacementSource[:destinationLength]
		originalComposingText = oldText[replacementDestinationStart:replacementDestinationEnd]
	}

	isOriginalComposingRegionTextChanged := originalComposingText != newComposingText
	isReplaced := isOriginalComposingRegionTextChanged || isReplacedByLonger || isReplacedByShorter || isReplacedBySame

	switch {
	case isEqual:
		return NewTextEditingDeltaNonTextUpdate(oldText, selection, composing)
	case (isDeletingByReplacingWithEmpty || isDeletingInsideComposingRegion) && !isOriginalComposingRegionTextChanged:
		// Deletion.
		actualStart := replacementDestinationStart
		if !isDeletionGreaterThanOne {
			actualStart = replacementDestinationEnd - lastCharLength
		}
		return NewTextEditingDeltaDeletion(oldText, ui.TextRange{Start: actualStart, End: replacementDestinationEnd}, selection, composing)
	case (replacementDestinationStart == replacementDestinationEnd || isInsertingInsideComposingRegion) && !isOriginalComposingRegionTextChanged:
		// Insertion.
		return NewTextEditingDeltaInsertion(oldText, replacementSource[destinationLength:], replacementDestinationEnd, selection, composing)
	case isReplaced:
		// Replacement.
		return NewTextEditingDeltaReplacement(oldText, replacementSource,
			ui.TextRange{Start: replacementDestinationStart, End: replacementDestinationEnd}, selection, composing)
	}
	// Not reachable: every other change is an insertion, a deletion or a
	// replacement (Dart asserts false here and returns this).
	return NewTextEditingDeltaNonTextUpdate(oldText, selection, composing)
}
