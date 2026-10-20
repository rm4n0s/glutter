// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// editableSemanticsNodes are the semantics nodes of the text spans of an
// editable, kept so that their ids are stable.
type editableSemanticsNodes []*semantics.SemanticsNode

func (r *RenderEditable) combinedSemanticsInfos() []painting.InlineSpanSemanticsInformation {
	if !r.haveCombinedInfos {
		r.cachedCombinedSemanticsInfos = painting.CombineSemanticsInfo(r.semanticsInfo)
		r.haveCombinedInfos = true
	}
	return r.cachedCombinedSemanticsInfos
}

// DescribeSemanticsConfiguration describes a text field: its value, its focus,
// the selection and the actions that move the cursor. Selectable rich text with
// recognizers makes a semantics node for each text fragment instead.
//
// Flutter: rendering/editable.dart RenderEditable.describeSemanticsConfiguration
func (r *RenderEditable) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderBox.DescribeSemanticsConfiguration(config)
	r.semanticsInfo = painting.InlineSpanGetSemanticsInformation(r.textPainter.Text())
	r.haveCombinedInfos = false
	for _, info := range r.semanticsInfo {
		if info.Recognizer != nil {
			if foundation.DebugMode && (!r.readOnly || r.obscureText) {
				panic(errors.New("EditableRecognizerNotReadOnly", "an editable with a span that has a recognizer must be read only and not obscured"))
			}
			// For selectable rich text with a recognizer, a semantics node is
			// needed for each text fragment.
			config.SetIsSemanticBoundary(true)
			config.ExplicitChildNodes = true
			return
		}
	}
	if r.cachedAttributedValue == nil {
		var value semantics.AttributedString
		if r.obscureText {
			value = semantics.NewAttributedString(r.obscured())
		} else {
			var buffer []byte
			offset := 0
			var attributes []ui.StringAttribute
			for _, info := range r.semanticsInfo {
				label := info.Text
				if info.SemanticsLabelSet {
					label = info.SemanticsLabel
				}
				for _, a := range info.StringAttributes {
					orig := a.Range()
					attributes = append(attributes, a.CopyWithRange(ui.NewTextRange(offset+orig.Start, offset+orig.End)))
				}
				buffer = append(buffer, label...)
				offset += len(label)
			}
			value = semantics.NewAttributedString(string(buffer), attributes...)
		}
		r.cachedAttributedValue = &value
	}
	config.SetAttributedValue(*r.cachedAttributedValue)
	config.SetIsObscured(r.obscureText)
	config.SetIsMultiline(r.isMultiline())
	config.SetTextDirection(r.TextDirection())
	config.SetIsFocused(foundation.Some(r.hasFocus))
	config.SetIsFocusable(true)
	config.SetIsTextField(true)
	config.SetIsReadOnly(r.readOnly)
	// This is the default for those who use RenderEditable directly; the real
	// value is usually set by EditableText.
	config.SetInputType(ui.SemanticsInputTypeText)

	if r.hasFocus && r.SelectionEnabled() {
		config.SetOnSetSelection(r.handleSetSelection)
	}
	if r.hasFocus && !r.readOnly {
		config.SetOnSetText(r.handleSetText)
	}
	if sel, ok := r.selection.Get(); ok && r.SelectionEnabled() && sel.IsValid() {
		config.SetTextSelection(sel)
		if _, ok := r.textPainter.GetOffsetBefore(sel.ExtentOffset); ok {
			config.SetOnMoveCursorBackwardByWord(r.handleMoveCursorBackwardByWord)
			config.SetOnMoveCursorBackwardByCharacter(r.handleMoveCursorBackwardByCharacter)
		}
		if _, ok := r.textPainter.GetOffsetAfter(sel.ExtentOffset); ok {
			config.SetOnMoveCursorForwardByWord(r.handleMoveCursorForwardByWord)
			config.SetOnMoveCursorForwardByCharacter(r.handleMoveCursorForwardByCharacter)
		}
	}
}

func (r *RenderEditable) handleSetText(text string) {
	r.textSelectionDelegate.UserUpdateTextEditingValue(services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: text, Selection: foundation.Some(services.NewTextSelectionCollapsed(len(text))),
	}), services.SelectionChangedCauseKeyboard)
}

// AssembleSemanticsNode makes a node for each text span (the case of a read only
// editable with recognizers), and puts the nodes of the placeholders in order.
//
// Flutter: rendering/editable.dart RenderEditable.assembleSemanticsNode
func (r *RenderEditable) AssembleSemanticsNode(node *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, children []*semantics.SemanticsNode) {
	if foundation.DebugMode && len(r.semanticsInfo) == 0 {
		panic(errors.New("EditableSemanticsInfoMissing", "AssembleSemanticsNode was called without the semantics information of the text"))
	}
	var newChildren []*semantics.SemanticsNode
	currentDirection := r.TextDirection()
	ordinal := 0.0
	start := 0
	placeholderIndex, childIndex := 0, 0
	child := r.FirstChild()
	var newChildCache editableSemanticsNodes
	oldCache := r.cachedChildNodes
	for _, info := range r.combinedSemanticsInfos() {
		selection := ui.NewTextRange(start, start+len(info.Text))
		start += len(info.Text)
		if info.IsPlaceholder {
			// A placeholder may have 0 or more semantics nodes: all are
			// annotated with the index of the placeholder.
			for len(children) > childIndex && children[childIndex].IsTagged(NewPlaceholderSpanIndexSemanticsTag(placeholderIndex)) {
				newChildren = append(newChildren, children[childIndex])
				childIndex++
			}
			child = r.ChildAfter(child)
			placeholderIndex++
			continue
		}
		initialDirection := currentDirection
		rects := r.textPainter.GetBoxesForSelection(selection)
		if len(rects) == 0 {
			continue
		}
		rect := rects[0].ToRect()
		currentDirection = rects[0].Direction
		for _, textBox := range rects[1:] {
			rect = rect.ExpandToInclude(textBox.ToRect())
			currentDirection = textBox.Direction
		}
		// A text box may have infinite dimensions, which must not go up to the
		// bridges.
		c := r.BoxConstraints()
		rect = ui.NewRectFromLTWH(math.Max(0, rect.Left), math.Max(0, rect.Top),
			math.Min(rect.Width(), c.MaxWidth), math.Min(rect.Height(), c.MaxHeight))
		// Round the rectangle to make the API testable, and pad it so that the
		// accessibility rectangles do not overlap with the text.
		currentRect := ui.NewRectFromLTRB(math.Floor(rect.Left)-4, math.Floor(rect.Top)-4, math.Ceil(rect.Right)+4, math.Ceil(rect.Bottom)+4)
		configuration := semantics.NewSemanticsConfiguration()
		configuration.SetSortKey(semantics.NewOrdinalSortKey(ordinal))
		ordinal++
		configuration.SetTextDirection(initialDirection)
		label := info.Text
		if info.SemanticsLabelSet {
			label = info.SemanticsLabel
		}
		configuration.SetAttributedLabel(semantics.NewAttributedString(label, info.StringAttributes...))
		switch rec := info.Recognizer.(type) {
		case nil:
		case *gestures.TapGestureRecognizer:
			if rec.OnTap != nil {
				configuration.SetOnTap(semantics.VoidCallback(rec.OnTap))
				configuration.SetIsLink(true)
			}
		case *gestures.DoubleTapGestureRecognizer:
			if rec.OnDoubleTap != nil {
				configuration.SetOnTap(semantics.VoidCallback(rec.OnDoubleTap))
				configuration.SetIsLink(true)
			}
		case *gestures.LongPressGestureRecognizer:
			if rec.OnLongPress != nil {
				configuration.SetOnLongPress(semantics.VoidCallback(rec.OnLongPress))
			}
		default:
			if foundation.DebugMode {
				panic(errors.New("EditableRecognizerUnsupported", foundation.RuntimeTypeName(info.Recognizer)+" is not supported",
					"recognizer", foundation.RuntimeTypeName(info.Recognizer)))
			}
		}
		if clip, ok := node.ParentPaintClipRect.Get(); ok {
			paintRect := clip.Intersect(currentRect)
			configuration.SetIsHidden(paintRect.IsEmpty() && !currentRect.IsEmpty())
		}
		var newChild *semantics.SemanticsNode
		if len(oldCache) > 0 {
			newChild, oldCache = oldCache[0], oldCache[1:]
		} else {
			key := foundation.NewUniqueKey()
			newChild = semantics.NewSemanticsNode(semantics.SemanticsNodeOpts{Key: key, ShowOnScreen: r.createShowOnScreenFor(key)})
		}
		newChild.UpdateWith(configuration, nil)
		newChild.SetRect(currentRect)
		newChildCache = append(newChildCache, newChild)
		newChildren = append(newChildren, newChild)
	}
	r.cachedChildNodes = newChildCache
	node.UpdateWith(config, newChildren)
}

func (r *RenderEditable) createShowOnScreenFor(key foundation.Key) func() {
	return func() {
		for _, n := range r.cachedChildNodes {
			if n.Key == key {
				r.ShowOnScreen(ShowOnScreenOpts{Descendant: r, Rect: foundation.Some(n.Rect())})
				return
			}
		}
	}
}

// ClearSemantics forgets the nodes of the spans too.
//
// Flutter: rendering/editable.dart RenderEditable.clearSemantics
func (r *RenderEditable) ClearSemantics() {
	r.RenderBox.ClearSemantics()
	r.cachedChildNodes = nil
}

func (r *RenderEditable) handleSetSelection(selection services.TextSelection) {
	r.setSelection(selection, services.SelectionChangedCauseKeyboard)
}

func (r *RenderEditable) handleMoveCursorForwardByCharacter(extendSelection bool) {
	sel := r.selection.Value()
	extentOffset, ok := r.textPainter.GetOffsetAfter(sel.ExtentOffset)
	if !ok {
		return
	}
	baseOffset := sel.BaseOffset
	if !extendSelection {
		baseOffset = extentOffset
	}
	r.setSelection(services.NewTextSelection(baseOffset, extentOffset), services.SelectionChangedCauseKeyboard)
}

func (r *RenderEditable) handleMoveCursorBackwardByCharacter(extendSelection bool) {
	sel := r.selection.Value()
	extentOffset, ok := r.textPainter.GetOffsetBefore(sel.ExtentOffset)
	if !ok {
		return
	}
	baseOffset := sel.BaseOffset
	if !extendSelection {
		baseOffset = extentOffset
	}
	r.setSelection(services.NewTextSelection(baseOffset, extentOffset), services.SelectionChangedCauseKeyboard)
}

func (r *RenderEditable) handleMoveCursorForwardByWord(extendSelection bool) {
	sel := r.selection.Value()
	currentWord := r.textPainter.GetWordBoundary(sel.Extent())
	nextWord, ok := r.getNextWord(currentWord.End)
	if !ok {
		return
	}
	baseOffset := nextWord.Start
	if extendSelection {
		baseOffset = sel.BaseOffset
	}
	r.setSelection(services.NewTextSelection(baseOffset, nextWord.Start), services.SelectionChangedCauseKeyboard)
}

func (r *RenderEditable) handleMoveCursorBackwardByWord(extendSelection bool) {
	sel := r.selection.Value()
	currentWord := r.textPainter.GetWordBoundary(sel.Extent())
	previousWord, ok := r.getPreviousWord(currentWord.Start - 1)
	if !ok {
		return
	}
	baseOffset := previousWord.Start
	if extendSelection {
		baseOffset = sel.BaseOffset
	}
	r.setSelection(services.NewTextSelection(baseOffset, previousWord.Start), services.SelectionChangedCauseKeyboard)
}
