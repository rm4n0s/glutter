// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// The intents of the text editing shortcuts. EditableText binds actions to them,
// and DefaultTextEditingShortcuts maps the keys to them.

// DoNothingAndStopPropagationTextIntent is an intent that does nothing and stops
// a key from reaching the handlers above, in a text field: a shortcut that would
// otherwise take a key a text field needs, such as Space, is mapped to it.
//
// Flutter: widgets/text_editing_intents.dart DoNothingAndStopPropagationTextIntent
type DoNothingAndStopPropagationTextIntent struct{ IntentBase }

// DirectionalTextEditingIntent is the interface of the intents that act in a
// direction in the text: forward is toward the end of the text.
//
// Flutter: widgets/text_editing_intents.dart DirectionalTextEditingIntent
type DirectionalTextEditingIntent interface {
	Intent
	// IsForward is whether the intent acts forward, toward the end of the text.
	//
	// Flutter: widgets/text_editing_intents.dart DirectionalTextEditingIntent.forward
	IsForward() bool
}

// DirectionalTextEditingIntentBase implements DirectionalTextEditingIntent;
// embed it.
//
// Flutter: widgets/text_editing_intents.dart DirectionalTextEditingIntent
type DirectionalTextEditingIntentBase struct {
	IntentBase
	forward bool
}

// IsForward is whether the intent acts toward the end of the text.
//
// Flutter: widgets/text_editing_intents.dart DirectionalTextEditingIntent.forward
func (i DirectionalTextEditingIntentBase) IsForward() bool { return i.forward }

// DeleteCharacterIntent is an intent that deletes a character in a direction.
//
// Flutter: widgets/text_editing_intents.dart DeleteCharacterIntent
type DeleteCharacterIntent struct {
	DirectionalTextEditingIntentBase
}

// NewDeleteCharacterIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart DeleteCharacterIntent.new
func NewDeleteCharacterIntent(forward bool) DeleteCharacterIntent {
	return DeleteCharacterIntent{DirectionalTextEditingIntentBase{forward: forward}}
}

// DeleteToNextWordBoundaryIntent is an intent that deletes from the caret to the
// next word boundary in a direction.
//
// Flutter: widgets/text_editing_intents.dart DeleteToNextWordBoundaryIntent
type DeleteToNextWordBoundaryIntent struct {
	DirectionalTextEditingIntentBase
}

// NewDeleteToNextWordBoundaryIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart DeleteToNextWordBoundaryIntent.new
func NewDeleteToNextWordBoundaryIntent(forward bool) DeleteToNextWordBoundaryIntent {
	return DeleteToNextWordBoundaryIntent{DirectionalTextEditingIntentBase{forward: forward}}
}

// DeleteToLineBreakIntent is an intent that deletes from the caret to the next
// line break in a direction.
//
// Flutter: widgets/text_editing_intents.dart DeleteToLineBreakIntent
type DeleteToLineBreakIntent struct {
	DirectionalTextEditingIntentBase
}

// NewDeleteToLineBreakIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart DeleteToLineBreakIntent.new
func NewDeleteToLineBreakIntent(forward bool) DeleteToLineBreakIntent {
	return DeleteToLineBreakIntent{DirectionalTextEditingIntentBase{forward: forward}}
}

// DirectionalCaretMovementIntent is the interface of the intents that move or
// extend the caret in a direction.
//
// Flutter: widgets/text_editing_intents.dart DirectionalCaretMovementIntent
type DirectionalCaretMovementIntent interface {
	DirectionalTextEditingIntent
	// CollapseSelection is whether the selection collapses to the new caret.
	//
	// Flutter: widgets/text_editing_intents.dart DirectionalCaretMovementIntent.collapseSelection
	CollapseSelection() bool
	// CollapseAtReversal is whether the selection collapses when the direction
	// reverses.
	//
	// Flutter: widgets/text_editing_intents.dart DirectionalCaretMovementIntent.collapseAtReversal
	CollapseAtReversal() bool
	// ContinuesAtWrap is whether the movement continues at a line wrap.
	//
	// Flutter: widgets/text_editing_intents.dart DirectionalCaretMovementIntent.continuesAtWrap
	ContinuesAtWrap() bool
}

// DirectionalCaretMovementIntentBase implements DirectionalCaretMovementIntent;
// embed it.
//
// Flutter: widgets/text_editing_intents.dart DirectionalCaretMovementIntent
type DirectionalCaretMovementIntentBase struct {
	DirectionalTextEditingIntentBase
	collapseSelection, collapseAtReversal, continuesAtWrap bool
}

func newCaretMovement(forward, collapseSelection, collapseAtReversal, continuesAtWrap bool) DirectionalCaretMovementIntentBase {
	if foundation.DebugMode && collapseSelection && collapseAtReversal {
		panic(errors.New("CaretMovementIntentInvalid", "an intent can not both collapse the selection and collapse it at a reversal"))
	}
	return DirectionalCaretMovementIntentBase{DirectionalTextEditingIntentBase{forward: forward}, collapseSelection, collapseAtReversal, continuesAtWrap}
}

// CollapseSelection is whether the selection collapses to the new caret.
//
// Flutter: widgets/text_editing_intents.dart DirectionalCaretMovementIntent.collapseSelection
func (i DirectionalCaretMovementIntentBase) CollapseSelection() bool { return i.collapseSelection }

// CollapseAtReversal is whether the selection collapses when the direction
// reverses.
//
// Flutter: widgets/text_editing_intents.dart DirectionalCaretMovementIntent.collapseAtReversal
func (i DirectionalCaretMovementIntentBase) CollapseAtReversal() bool { return i.collapseAtReversal }

// ContinuesAtWrap is whether the movement continues at a line wrap.
//
// Flutter: widgets/text_editing_intents.dart DirectionalCaretMovementIntent.continuesAtWrap
func (i DirectionalCaretMovementIntentBase) ContinuesAtWrap() bool { return i.continuesAtWrap }

// ExtendSelectionByCharacterIntent is an intent that extends or moves the
// selection by a character.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionByCharacterIntent
type ExtendSelectionByCharacterIntent struct {
	DirectionalCaretMovementIntentBase
}

// NewExtendSelectionByCharacterIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionByCharacterIntent.new
func NewExtendSelectionByCharacterIntent(forward, collapseSelection bool) ExtendSelectionByCharacterIntent {
	return ExtendSelectionByCharacterIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ExtendSelectionToNextWordBoundaryIntent is an intent that extends or moves the
// selection to the next word boundary.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToNextWordBoundaryIntent
type ExtendSelectionToNextWordBoundaryIntent struct {
	DirectionalCaretMovementIntentBase
}

// NewExtendSelectionToNextWordBoundaryIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToNextWordBoundaryIntent.new
func NewExtendSelectionToNextWordBoundaryIntent(forward, collapseSelection bool) ExtendSelectionToNextWordBoundaryIntent {
	return ExtendSelectionToNextWordBoundaryIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ExtendSelectionToNextWordBoundaryOrCaretLocationIntent is an intent that
// extends the selection to the next word boundary, or to the caret location when
// the direction reverses.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToNextWordBoundaryOrCaretLocationIntent
type ExtendSelectionToNextWordBoundaryOrCaretLocationIntent struct {
	DirectionalCaretMovementIntentBase
}

// NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToNextWordBoundaryOrCaretLocationIntent.new
func NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(forward bool) ExtendSelectionToNextWordBoundaryOrCaretLocationIntent {
	return ExtendSelectionToNextWordBoundaryOrCaretLocationIntent{newCaretMovement(forward, false, true, false)}
}

// ExpandSelectionToDocumentBoundaryIntent is an intent that expands the selection
// to the start or end of the document.
//
// Flutter: widgets/text_editing_intents.dart ExpandSelectionToDocumentBoundaryIntent
type ExpandSelectionToDocumentBoundaryIntent struct {
	DirectionalCaretMovementIntentBase
}

// NewExpandSelectionToDocumentBoundaryIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ExpandSelectionToDocumentBoundaryIntent.new
func NewExpandSelectionToDocumentBoundaryIntent(forward bool) ExpandSelectionToDocumentBoundaryIntent {
	return ExpandSelectionToDocumentBoundaryIntent{newCaretMovement(forward, false, false, false)}
}

// ExpandSelectionToLineBreakIntent is an intent that expands the selection to the
// next line break.
//
// Flutter: widgets/text_editing_intents.dart ExpandSelectionToLineBreakIntent
type ExpandSelectionToLineBreakIntent struct {
	DirectionalCaretMovementIntentBase
}

// NewExpandSelectionToLineBreakIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ExpandSelectionToLineBreakIntent.new
func NewExpandSelectionToLineBreakIntent(forward bool) ExpandSelectionToLineBreakIntent {
	return ExpandSelectionToLineBreakIntent{newCaretMovement(forward, false, false, false)}
}

// ExtendSelectionToLineBreakIntent is an intent that extends or moves the
// selection to the next line break.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToLineBreakIntent
type ExtendSelectionToLineBreakIntent struct {
	DirectionalCaretMovementIntentBase
}

// ExtendSelectionToLineBreakIntentOpts are the optional parameters of
// NewExtendSelectionToLineBreakIntent.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToLineBreakIntent.new
type ExtendSelectionToLineBreakIntentOpts struct {
	// CollapseAtReversal collapses the selection when the direction reverses.
	CollapseAtReversal bool
	// ContinuesAtWrap continues the movement at a line wrap.
	ContinuesAtWrap bool
}

// NewExtendSelectionToLineBreakIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToLineBreakIntent.new
func NewExtendSelectionToLineBreakIntent(forward, collapseSelection bool, opts ...ExtendSelectionToLineBreakIntentOpts) ExtendSelectionToLineBreakIntent {
	o := firstOpts(opts)
	return ExtendSelectionToLineBreakIntent{newCaretMovement(forward, collapseSelection, o.CollapseAtReversal, o.ContinuesAtWrap)}
}

// ExtendSelectionVerticallyToAdjacentLineIntent is an intent that extends or
// moves the selection to the adjacent line.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionVerticallyToAdjacentLineIntent
type ExtendSelectionVerticallyToAdjacentLineIntent struct {
	DirectionalCaretMovementIntentBase
}

// NewExtendSelectionVerticallyToAdjacentLineIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionVerticallyToAdjacentLineIntent.new
func NewExtendSelectionVerticallyToAdjacentLineIntent(forward, collapseSelection bool) ExtendSelectionVerticallyToAdjacentLineIntent {
	return ExtendSelectionVerticallyToAdjacentLineIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ExtendSelectionVerticallyToAdjacentPageIntent is an intent that extends or
// moves the selection to the adjacent page.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionVerticallyToAdjacentPageIntent
type ExtendSelectionVerticallyToAdjacentPageIntent struct {
	DirectionalCaretMovementIntentBase
}

// NewExtendSelectionVerticallyToAdjacentPageIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionVerticallyToAdjacentPageIntent.new
func NewExtendSelectionVerticallyToAdjacentPageIntent(forward, collapseSelection bool) ExtendSelectionVerticallyToAdjacentPageIntent {
	return ExtendSelectionVerticallyToAdjacentPageIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ExtendSelectionToNextParagraphBoundaryIntent is an intent that extends or moves
// the selection to the next paragraph boundary.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToNextParagraphBoundaryIntent
type ExtendSelectionToNextParagraphBoundaryIntent struct {
	DirectionalCaretMovementIntentBase
}

// NewExtendSelectionToNextParagraphBoundaryIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToNextParagraphBoundaryIntent.new
func NewExtendSelectionToNextParagraphBoundaryIntent(forward, collapseSelection bool) ExtendSelectionToNextParagraphBoundaryIntent {
	return ExtendSelectionToNextParagraphBoundaryIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent is an intent that
// extends the selection to the next paragraph boundary, or to the caret location
// when the direction reverses.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent
type ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent struct {
	DirectionalCaretMovementIntentBase
}

// NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent creates the
// intent.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent.new
func NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(forward bool) ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent {
	return ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent{newCaretMovement(forward, false, true, false)}
}

// ExtendSelectionToDocumentBoundaryIntent is an intent that extends or moves the
// selection to the start or end of the document.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToDocumentBoundaryIntent
type ExtendSelectionToDocumentBoundaryIntent struct {
	DirectionalCaretMovementIntentBase
}

// NewExtendSelectionToDocumentBoundaryIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ExtendSelectionToDocumentBoundaryIntent.new
func NewExtendSelectionToDocumentBoundaryIntent(forward, collapseSelection bool) ExtendSelectionToDocumentBoundaryIntent {
	return ExtendSelectionToDocumentBoundaryIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ScrollToDocumentBoundaryIntent is an intent that scrolls to the start or end of
// the document.
//
// Flutter: widgets/text_editing_intents.dart ScrollToDocumentBoundaryIntent
type ScrollToDocumentBoundaryIntent struct {
	DirectionalTextEditingIntentBase
}

// NewScrollToDocumentBoundaryIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ScrollToDocumentBoundaryIntent.new
func NewScrollToDocumentBoundaryIntent(forward bool) ScrollToDocumentBoundaryIntent {
	return ScrollToDocumentBoundaryIntent{DirectionalTextEditingIntentBase{forward: forward}}
}

// SelectAllTextIntent is an intent that selects all the text.
//
// Flutter: widgets/text_editing_intents.dart SelectAllTextIntent
type SelectAllTextIntent struct {
	IntentBase
	// Cause is what caused the selection to change.
	//
	// Flutter: widgets/text_editing_intents.dart SelectAllTextIntent.cause
	Cause services.SelectionChangedCause
}

// CopySelectionTextIntent is an intent that copies (or cuts) the selected text.
//
// Flutter: widgets/text_editing_intents.dart CopySelectionTextIntent
type CopySelectionTextIntent struct {
	IntentBase
	// Cause is what caused the selection to change.
	//
	// Flutter: widgets/text_editing_intents.dart CopySelectionTextIntent.cause
	Cause services.SelectionChangedCause
	// CollapseSelection is whether the selection collapses after the copy: a cut.
	//
	// Flutter: widgets/text_editing_intents.dart CopySelectionTextIntent.collapseSelection
	CollapseSelection bool
}

// NewCopySelectionTextIntentCut creates the intent that cuts the selection.
//
// Flutter: widgets/text_editing_intents.dart CopySelectionTextIntent.cut
func NewCopySelectionTextIntentCut(cause services.SelectionChangedCause) CopySelectionTextIntent {
	return CopySelectionTextIntent{Cause: cause, CollapseSelection: true}
}

// NewCopySelectionTextIntentCopy creates the intent that copies the selection,
// by the keyboard.
//
// Flutter: widgets/text_editing_intents.dart CopySelectionTextIntent.copy
func NewCopySelectionTextIntentCopy() CopySelectionTextIntent {
	return CopySelectionTextIntent{Cause: services.SelectionChangedCauseKeyboard}
}

// PasteTextIntent is an intent that pastes the clipboard text.
//
// Flutter: widgets/text_editing_intents.dart PasteTextIntent
type PasteTextIntent struct {
	IntentBase
	// Cause is what caused the selection to change.
	//
	// Flutter: widgets/text_editing_intents.dart PasteTextIntent.cause
	Cause services.SelectionChangedCause
}

// RedoTextIntent is an intent that redoes the last undone change.
//
// Flutter: widgets/text_editing_intents.dart RedoTextIntent
type RedoTextIntent struct {
	IntentBase
	// Cause is what caused the selection to change.
	//
	// Flutter: widgets/text_editing_intents.dart RedoTextIntent.cause
	Cause services.SelectionChangedCause
}

// UndoTextIntent is an intent that undoes the last change.
//
// Flutter: widgets/text_editing_intents.dart UndoTextIntent
type UndoTextIntent struct {
	IntentBase
	// Cause is what caused the selection to change.
	//
	// Flutter: widgets/text_editing_intents.dart UndoTextIntent.cause
	Cause services.SelectionChangedCause
}

// TransposeCharactersIntent is an intent that swaps the characters around the
// caret.
//
// Flutter: widgets/text_editing_intents.dart TransposeCharactersIntent
type TransposeCharactersIntent struct{ IntentBase }

// EditableTextTapOutsideIntent is an intent that runs when a pointer goes down
// outside a text field.
//
// Flutter: widgets/text_editing_intents.dart EditableTextTapOutsideIntent
type EditableTextTapOutsideIntent struct {
	IntentBase
	// FocusNode is the node of the text field.
	//
	// Flutter: widgets/text_editing_intents.dart EditableTextTapOutsideIntent.focusNode
	FocusNode *FocusNode
	// PointerDownEvent is the event of the pointer going down.
	//
	// Flutter: widgets/text_editing_intents.dart EditableTextTapOutsideIntent.pointerDownEvent
	PointerDownEvent *gestures.PointerDownEvent
}

// EditableTextTapUpOutsideIntent is an intent that runs when a pointer goes up
// outside a text field.
//
// Flutter: widgets/text_editing_intents.dart EditableTextTapUpOutsideIntent
type EditableTextTapUpOutsideIntent struct {
	IntentBase
	// FocusNode is the node of the text field.
	//
	// Flutter: widgets/text_editing_intents.dart EditableTextTapUpOutsideIntent.focusNode
	FocusNode *FocusNode
	// PointerUpEvent is the event of the pointer going up.
	//
	// Flutter: widgets/text_editing_intents.dart EditableTextTapUpOutsideIntent.pointerUpEvent
	PointerUpEvent *gestures.PointerUpEvent
}

// ReplaceTextIntent is an intent that replaces the text in a range with another
// text, in a value.
//
// Flutter: widgets/text_editing_intents.dart ReplaceTextIntent
type ReplaceTextIntent struct {
	IntentBase
	// CurrentTextEditingValue is the value the replacement is made in.
	//
	// Flutter: widgets/text_editing_intents.dart ReplaceTextIntent.currentTextEditingValue
	CurrentTextEditingValue services.TextEditingValue
	// ReplacementText is the text that replaces the range.
	//
	// Flutter: widgets/text_editing_intents.dart ReplaceTextIntent.replacementText
	ReplacementText string
	// ReplacementRange is the range that is replaced.
	//
	// Flutter: widgets/text_editing_intents.dart ReplaceTextIntent.replacementRange
	ReplacementRange ui.TextRange
	// Cause is what caused the change.
	//
	// Flutter: widgets/text_editing_intents.dart ReplaceTextIntent.cause
	Cause services.SelectionChangedCause
}

// NewReplaceTextIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart ReplaceTextIntent.new
func NewReplaceTextIntent(currentTextEditingValue services.TextEditingValue, replacementText string, replacementRange ui.TextRange, cause services.SelectionChangedCause) ReplaceTextIntent {
	return ReplaceTextIntent{CurrentTextEditingValue: currentTextEditingValue, ReplacementText: replacementText, ReplacementRange: replacementRange, Cause: cause}
}

// UpdateSelectionIntent is an intent that sets the selection of a value.
//
// Flutter: widgets/text_editing_intents.dart UpdateSelectionIntent
type UpdateSelectionIntent struct {
	IntentBase
	// CurrentTextEditingValue is the value whose selection is set.
	//
	// Flutter: widgets/text_editing_intents.dart UpdateSelectionIntent.currentTextEditingValue
	CurrentTextEditingValue services.TextEditingValue
	// NewSelection is the selection to set.
	//
	// Flutter: widgets/text_editing_intents.dart UpdateSelectionIntent.newSelection
	NewSelection services.TextSelection
	// Cause is what caused the change.
	//
	// Flutter: widgets/text_editing_intents.dart UpdateSelectionIntent.cause
	Cause services.SelectionChangedCause
}

// NewUpdateSelectionIntent creates the intent.
//
// Flutter: widgets/text_editing_intents.dart UpdateSelectionIntent.new
func NewUpdateSelectionIntent(currentTextEditingValue services.TextEditingValue, newSelection services.TextSelection, cause services.SelectionChangedCause) UpdateSelectionIntent {
	return UpdateSelectionIntent{CurrentTextEditingValue: currentTextEditingValue, NewSelection: newSelection, Cause: cause}
}
