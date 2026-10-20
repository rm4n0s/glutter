// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// selectableTextContainer is Flutter's _SelectableTextContainer: it creates a
// SelectionContainer around a RichText so the selectables of its paragraph
// register with a delegate that knows the paragraph.
//
// Flutter: widgets/text.dart _SelectableTextContainer
type selectableTextContainer struct {
	StatefulWidget
	text *RichText
}

func (w *selectableTextContainer) CreateState() State {
	return &selectableTextContainerState{textKey: NewGlobalKey[State]()}
}

type selectableTextContainerState struct {
	StateOf[*selectableTextContainer]
	textKey           *GlobalKey[State]
	selectionDelegate *selectableTextContainerDelegate
}

func (s *selectableTextContainerState) InitState() {
	s.selectionDelegate = newSelectableTextContainerDelegate(s.textKey)
}

func (s *selectableTextContainerState) Dispose() { s.selectionDelegate.Dispose() }

func (s *selectableTextContainerState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewSelectionContainer(s.selectionDelegate, &selectableRichText{
		StatelessWidget: NewStatelessWidgetBase(), textKey: s.textKey, text: w.text,
	})
}

// selectableRichText is Flutter's _RichText: a wrapper so the underlying
// RenderParagraph can register its Selectables to the SelectionContainer above.
//
// Flutter: widgets/text.dart _RichText
type selectableRichText struct {
	StatelessWidget
	textKey *GlobalKey[State]
	text    *RichText
}

func (w *selectableRichText) Build(bc BuildContext) Widget {
	o := w.text.o
	o.Key = w.textKey
	o.SelectionRegistrar = SelectionContainerMaybeOf(bc)
	return NewRichText(w.text.text, o)
}

// selectableTextContainerDelegate is Flutter's _SelectableTextContainerDelegate.
//
// Flutter: widgets/text.dart _SelectableTextContainerDelegate
type selectableTextContainerDelegate struct {
	StaticSelectionContainerDelegate
	textKey *GlobalKey[State]
}

func newSelectableTextContainerDelegate(textKey *GlobalKey[State]) *selectableTextContainerDelegate {
	d := &selectableTextContainerDelegate{textKey: textKey}
	InitStaticSelectionContainerDelegate(&d.StaticSelectionContainerDelegate, d)
	return d
}

func (d *selectableTextContainerDelegate) paragraph() *rendering.RenderParagraph {
	return d.textKey.CurrentContext().FindRenderObject().(*rendering.RenderParagraph)
}

func (d *selectableTextContainerDelegate) HandleSelectParagraph(event rendering.SelectParagraphSelectionEvent) rendering.SelectionResult {
	result := d.handleSelectParagraph(event)
	d.DidReceiveSelectionBoundaryEvents()
	return result
}

func (d *selectableTextContainerDelegate) handleSelectParagraph(event rendering.SelectParagraphSelectionEvent) rendering.SelectionResult {
	if event.Absorb {
		for _, s := range d.Selectables {
			d.DispatchSelectionEventToChild(s, event)
		}
		d.CurrentSelectionStartIndex = 0
		d.CurrentSelectionEndIndex = len(d.Selectables) - 1
		return rendering.SelectionResultNext
	}
	paragraph := d.paragraph()
	// First pass, if the position is on a placeholder then dispatch the
	// selection event to the Selectable at the location and terminate.
	for index, s := range d.Selectables {
		selectableIsPlaceholder := !paragraph.SelectableBelongsToParagraph(s)
		if selectableIsPlaceholder && len(s.BoundingBoxes()) > 0 {
			for _, rect := range s.BoundingBoxes() {
				if s.GetTransformTo(nil).TransformRect(rect).Contains(event.GlobalPosition) {
					d.CurrentSelectionStartIndex, d.CurrentSelectionEndIndex = index, index
					return d.DispatchSelectionEventToChild(s, event)
				}
			}
		}
	}
	var lastResult foundation.Opt[rendering.SelectionResult]
	foundStart := false
	lastNextIndex := foundation.None[int]()
	absorbEvent := rendering.SelectParagraphSelectionEvent{GlobalPosition: event.GlobalPosition, Absorb: true}
	for index := 0; index < len(d.Selectables); index++ {
		s := d.Selectables[index]
		if !paragraph.SelectableBelongsToParagraph(s) {
			if foundStart {
				result := d.DispatchSelectionEventToChild(s, absorbEvent)
				if len(d.Selectables)-1 == index {
					d.CurrentSelectionEndIndex = index
					d.flushInactiveSelections()
					return result
				}
			}
			continue
		}
		existing := s.Value()
		result := d.DispatchSelectionEventToChild(s, event)
		lastResult = foundation.Some(result)
		if index == len(d.Selectables)-1 && result == rendering.SelectionResultNext {
			if foundStart {
				d.CurrentSelectionEndIndex = index
			} else {
				d.CurrentSelectionStartIndex, d.CurrentSelectionEndIndex = index, index
			}
			return rendering.SelectionResultNext
		}
		if result == rendering.SelectionResultNext {
			if s.Value().Equal(existing) && !foundStart {
				lastNextIndex = foundation.Some(index)
			}
			if !s.Value().Equal(existing) && !foundStart {
				selectionAtStartOfSelectable := s.BoundingBoxes()[0].Overlaps(s.Value().SelectionRects[0])
				var startIndex int
				switch {
				case lastNextIndex.IsSome() && selectionAtStartOfSelectable:
					startIndex = lastNextIndex.Value() + 1
				case lastNextIndex.IsNone() && selectionAtStartOfSelectable:
					startIndex = 0
				default:
					startIndex = index
				}
				for i := startIndex; i < index; i++ {
					d.DispatchSelectionEventToChild(d.Selectables[i], absorbEvent)
				}
				d.CurrentSelectionStartIndex = startIndex
				foundStart = true
			}
			continue
		}
		if index == 0 && result == rendering.SelectionResultPrevious {
			return rendering.SelectionResultPrevious
		}
		if !s.Value().Equal(existing) {
			if !foundStart && lastNextIndex.IsNone() {
				d.CurrentSelectionStartIndex = 0
				for i := 0; i < index; i++ {
					d.DispatchSelectionEventToChild(d.Selectables[i], absorbEvent)
				}
			}
			d.CurrentSelectionEndIndex = index
			// Geometry has changed as a result of select paragraph, need to clear
			// the selection of other selectables to keep selection in sync.
			d.flushInactiveSelections()
		}
		return rendering.SelectionResultEnd
	}
	_ = lastResult
	return rendering.SelectionResultEnd
}

func (d *selectableTextContainerDelegate) adjustSelection(event rendering.SelectionEdgeUpdateEvent, isEnd bool) rendering.SelectionResult {
	var finalResult foundation.Opt[rendering.SelectionResult]
	v := d.Value()
	currentEdgeWithin, oppositeEdgeWithin := v.StartSelectionPoint != nil, v.EndSelectionPoint != nil
	if isEnd {
		currentEdgeWithin, oppositeEdgeWithin = oppositeEdgeWithin, currentEdgeWithin
	}
	var newIndex int
	switch {
	case isEnd && currentEdgeWithin:
		newIndex = d.CurrentSelectionEndIndex
	case isEnd && oppositeEdgeWithin:
		newIndex = d.CurrentSelectionStartIndex
	case isEnd:
		newIndex = 0
	case currentEdgeWithin:
		newIndex = d.CurrentSelectionStartIndex
	case oppositeEdgeWithin:
		newIndex = d.CurrentSelectionEndIndex
	}
	forward := foundation.None[bool]()
	for newIndex < len(d.Selectables) && newIndex >= 0 && finalResult.IsNone() {
		current := d.DispatchSelectionEventToChild(d.Selectables[newIndex], event)
		switch current {
		case rendering.SelectionResultEnd, rendering.SelectionResultPending, rendering.SelectionResultNone:
			finalResult = foundation.Some(current)
		case rendering.SelectionResultNext:
			if f, ok := forward.Get(); ok && !f {
				newIndex++
				finalResult = foundation.Some(rendering.SelectionResultEnd)
			} else if newIndex == len(d.Selectables)-1 {
				finalResult = foundation.Some(current)
			} else {
				forward = foundation.Some(true)
				newIndex++
			}
		case rendering.SelectionResultPrevious:
			if forward.OrElse(false) {
				newIndex--
				finalResult = foundation.Some(rendering.SelectionResultEnd)
			} else if newIndex == 0 {
				finalResult = foundation.Some(current)
			} else {
				forward = foundation.Some(false)
				newIndex--
			}
		}
	}
	f, hasForward := forward.Get()
	forwardSelection := d.CurrentSelectionEndIndex >= d.CurrentSelectionStartIndex
	if isEnd {
		if hasForward && ((!forwardSelection && f && newIndex >= d.CurrentSelectionStartIndex) ||
			(forwardSelection && !f && newIndex <= d.CurrentSelectionStartIndex)) {
			d.CurrentSelectionStartIndex = d.CurrentSelectionEndIndex
		}
		d.CurrentSelectionEndIndex = newIndex
	} else {
		if hasForward && ((!forwardSelection && !f && newIndex <= d.CurrentSelectionEndIndex) ||
			(forwardSelection && f && newIndex >= d.CurrentSelectionEndIndex)) {
			d.CurrentSelectionEndIndex = d.CurrentSelectionStartIndex
		}
		d.CurrentSelectionStartIndex = newIndex
	}
	d.flushInactiveSelections()
	return finalResult.OrElse(rendering.SelectionResultNone)
}

// CompareOrder sorts the selectables by their top left Rect.
//
// Flutter: widgets/text.dart _SelectableTextContainerDelegate.compareOrder
func (d *selectableTextContainerDelegate) CompareOrder(a, b rendering.Selectable) int {
	rectA := a.GetTransformTo(nil).TransformRect(a.BoundingBoxes()[0])
	rectB := b.GetTransformTo(nil).TransformRect(b.BoundingBoxes()[0])
	if result := compareRectsVertically(rectA, rectB); result != 0 {
		return result
	}
	return compareRectsHorizontally(rectA, rectB)
}

// GetSelection returns the range considering the range from each selectable
// child, and where each starts in the paragraph.
//
// Flutter: widgets/text.dart _SelectableTextContainerDelegate.getSelection
func (d *selectableTextContainerDelegate) GetSelection() *rendering.SelectedContentRange {
	selections := make([]selectionInfo, 0, len(d.Selectables))
	for _, s := range d.Selectables {
		selections = append(selections, selectionInfo{contentLength: s.ContentLength(), rng: s.GetSelection()})
	}
	return d.calculateTextLocalRange(selections)
}

func (d *selectableTextContainerDelegate) calculateTextLocalRange(selections []selectionInfo) *rendering.SelectedContentRange {
	if d.CurrentSelectionStartIndex == -1 || d.CurrentSelectionEndIndex == -1 {
		return nil
	}
	startOffset, endOffset := 0, 0
	foundStart := false
	forwardSelection := d.CurrentSelectionEndIndex >= d.CurrentSelectionStartIndex
	if d.CurrentSelectionEndIndex == d.CurrentSelectionStartIndex {
		if r := d.Selectables[d.CurrentSelectionStartIndex].GetSelection(); r != nil {
			forwardSelection = r.EndOffset >= r.StartOffset
		}
	}
	result := func() *rendering.SelectedContentRange {
		if forwardSelection {
			return &rendering.SelectedContentRange{StartOffset: startOffset, EndOffset: endOffset}
		}
		return &rendering.SelectedContentRange{StartOffset: endOffset, EndOffset: startOffset}
	}
	for index, selection := range selections {
		if selection.rng == nil {
			if foundStart {
				return result()
			}
			startOffset += selection.contentLength
			endOffset = startOffset
			continue
		}
		selectionStartNormalized := min(selection.rng.StartOffset, selection.rng.EndOffset)
		selectionEndNormalized := max(selection.rng.StartOffset, selection.rng.EndOffset)
		if !foundStart {
			// Because a RenderParagraph may split its content into multiple
			// selectables we have to consider at what offset a selectable starts at
			// relative to the RenderParagraph, when the selectable is not the start
			// of the content.
			paragraph := d.paragraph()
			shouldConsiderContentStart := index > 0 && paragraph.SelectableBelongsToParagraph(d.Selectables[index])
			contentStart := 0
			if shouldConsiderContentStart {
				contentStart = paragraph.GetPositionForOffset(leftCenter(d.Selectables[index].BoundingBoxes()[0])).Offset
			}
			startOffset += absInt(selectionStartNormalized - contentStart)
			endOffset = startOffset + absInt(selectionEndNormalized-selectionStartNormalized)
			foundStart = true
		} else {
			endOffset += absInt(selectionEndNormalized - selectionStartNormalized)
		}
	}
	return result()
}

func leftCenter(r ui.Rect) ui.Offset { return ui.Offset{Dx: r.Left, Dy: r.Top + r.Height()/2} }

// HandleSelectionEdgeUpdate handles the paragraph granularity itself and leaves
// the other granularities to the static delegate.
//
// Flutter: widgets/text.dart _SelectableTextContainerDelegate.handleSelectionEdgeUpdate
func (d *selectableTextContainerDelegate) HandleSelectionEdgeUpdate(event rendering.SelectionEdgeUpdateEvent) rendering.SelectionResult {
	if event.Granularity != rendering.TextGranularityParagraph {
		return d.StaticSelectionContainerDelegate.HandleSelectionEdgeUpdate(event)
	}
	d.UpdateLastSelectionEdgeLocation(event.GlobalPosition, event.Type() == rendering.SelectionEventTypeEndEdgeUpdate)
	if event.Type() == rendering.SelectionEventTypeEndEdgeUpdate {
		if d.CurrentSelectionEndIndex == -1 {
			return d.StaticSelectionContainerDelegate.HandleSelectionEdgeUpdate(event)
		}
		return d.adjustSelection(event, true)
	}
	if d.CurrentSelectionStartIndex == -1 {
		return d.StaticSelectionContainerDelegate.HandleSelectionEdgeUpdate(event)
	}
	return d.adjustSelection(event, false)
}
