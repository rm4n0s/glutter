package rendering

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// textBoundaryRecord is the start and the end of a text boundary.
//
// Flutter: rendering/paragraph.dart _TextBoundaryRecord
type textBoundaryRecord struct {
	boundaryStart ui.TextPosition
	boundaryEnd   ui.TextPosition
}

// textBoundaryAtPosition finds the text boundary at a position.
//
// Flutter: rendering/paragraph.dart _TextBoundaryAtPosition
type textBoundaryAtPosition func(position ui.TextPosition) textBoundaryRecord

// textBoundaryAtPositionInText finds the text boundary at a position in a text.
//
// Flutter: rendering/paragraph.dart _TextBoundaryAtPositionInText
type textBoundaryAtPositionInText func(position ui.TextPosition, text string) textBoundaryRecord

// placeholderCharacter is the character that RenderParagraph uses for a
// placeholder (the object replacement character).
//
// Flutter: rendering/paragraph.dart _SelectableFragment._placeholderCharacter
var placeholderCharacter = string(rune(painting.PlaceholderCodeUnit))

// placeholderLength is the length of the placeholder character, in bytes.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._placeholderLength
var placeholderLength = len(placeholderCharacter)

// eqPosition reports whether two optional positions are the same: both are
// missing, or both are present with the same offset and affinity.
func eqPosition(a, b foundation.Opt[ui.TextPosition]) bool {
	if a.IsSome() != b.IsSome() {
		return false
	}
	return a.IsNone() || a.Value().Equal(b.Value())
}

// selectableFragment is the Selectable of a run of text between two
// placeholders of a paragraph.
//
// Flutter: rendering/paragraph.dart _SelectableFragment
type selectableFragment struct {
	foundation.ChangeNotifier
	rng       ui.TextRange
	paragraph *RenderParagraph
	fullText  string

	textSelectionStart foundation.Opt[ui.TextPosition]
	textSelectionEnd   foundation.Opt[ui.TextPosition]

	selectableContainsOriginTextBoundary bool

	startHandleLayerLink *LayerLink
	endHandleLayerLink   *LayerLink

	selectionGeometry SelectionGeometry

	cachedBoundingBoxes []ui.Rect
	hasBoundingBoxes    bool
	cachedRect          ui.Rect
	hasRect             bool
}

var (
	_ Selectable                 = (*selectableFragment)(nil)
	_ services.TextLayoutMetrics = (*selectableFragment)(nil)
)

func newSelectableFragment(p *RenderParagraph, rng ui.TextRange, fullText string) *selectableFragment {
	f := &selectableFragment{paragraph: p, rng: rng, fullText: fullText}
	f.selectionGeometry = f.getSelectionGeometry()
	return f
}

// Value is the geometry of the selection.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.value
func (f *selectableFragment) Value() SelectionGeometry { return f.selectionGeometry }

func (f *selectableFragment) updateSelectionGeometry() {
	newValue := f.getSelectionGeometry()
	if f.selectionGeometry.Equal(newValue) {
		return
	}
	f.selectionGeometry = newValue
	f.NotifyListeners()
}

func (f *selectableFragment) getSelectionGeometry() SelectionGeometry {
	if f.textSelectionStart.IsNone() || f.textSelectionEnd.IsNone() {
		return SelectionGeometry{Status: SelectionStatusNone, HasContent: true}
	}
	start, end := f.textSelectionStart.Value(), f.textSelectionEnd.Value()
	selectionStart, selectionEnd := start.Offset, end.Offset
	isReversed := selectionStart > selectionEnd
	startOffsetInParagraphCoordinates := f.paragraph.getOffsetForPosition(start)
	endOffsetInParagraphCoordinates := startOffsetInParagraphCoordinates
	if selectionStart != selectionEnd {
		endOffsetInParagraphCoordinates = f.paragraph.getOffsetForPosition(end)
	}
	flipHandles := isReversed != (ui.TextDirectionRTL == f.paragraph.TextDirection())
	var selectionRects []ui.Rect
	for _, textBox := range f.paragraph.GetBoxesForSelection(ui.TextRange{Start: selectionStart, End: selectionEnd}) {
		selectionRects = append(selectionRects, textBox.ToRect())
	}
	selectionCollapsed := selectionStart == selectionEnd
	var startHandleType, endHandleType TextSelectionHandleType
	switch {
	case selectionCollapsed:
		// Always prefer collapsed handle when selection is collapsed.
		startHandleType, endHandleType = TextSelectionHandleTypeCollapsed, TextSelectionHandleTypeCollapsed
	case flipHandles:
		startHandleType, endHandleType = TextSelectionHandleTypeRight, TextSelectionHandleTypeLeft
	default:
		startHandleType, endHandleType = TextSelectionHandleTypeLeft, TextSelectionHandleTypeRight
	}
	status := SelectionStatusUncollapsed
	if selectionCollapsed {
		status = SelectionStatusCollapsed
	}
	lineHeight := f.paragraph.textPainter.PreferredLineHeight()
	return SelectionGeometry{
		StartSelectionPoint: &SelectionPoint{LocalPosition: startOffsetInParagraphCoordinates, LineHeight: lineHeight, HandleType: startHandleType},
		EndSelectionPoint:   &SelectionPoint{LocalPosition: endOffsetInParagraphCoordinates, LineHeight: lineHeight, HandleType: endHandleType},
		SelectionRects:      selectionRects,
		Status:              status,
		HasContent:          true,
	}
}

// DispatchSelectionEvent handles the selection event.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.dispatchSelectionEvent
func (f *selectableFragment) DispatchSelectionEvent(event SelectionEvent) SelectionResult {
	var result SelectionResult
	existingSelectionStart := f.textSelectionStart
	existingSelectionEnd := f.textSelectionEnd
	switch e := event.(type) {
	case SelectionEdgeUpdateEvent:
		isEnd := e.Type() == SelectionEventTypeEndEdgeUpdate
		switch e.Granularity {
		case TextGranularityUnspecified, TextGranularityCharacter:
			result = f.updateSelectionEdge(e.GlobalPosition, isEnd)
		case TextGranularityWord:
			result = f.updateSelectionEdgeByTextBoundary(e.GlobalPosition, isEnd, f.getWordBoundaryAtPosition)
		case TextGranularityParagraph:
			result = f.updateSelectionEdgeByMultiSelectableTextBoundary(e.GlobalPosition, isEnd, f.getParagraphBoundaryAtPosition, f.getClampedParagraphBoundaryAtPosition)
		default:
			if foundation.DebugMode {
				panic(errors.New(tagSelectionGranularityUnsupported, "moving the selection edge by line or document is not supported"))
			}
		}
	case ClearSelectionEvent:
		result = f.handleClearSelection()
	case SelectAllSelectionEvent:
		result = f.handleSelectAll()
	case SelectWordSelectionEvent:
		result = f.handleSelectWord(e.GlobalPosition)
	case SelectParagraphSelectionEvent:
		if e.Absorb {
			f.handleSelectAll()
			result = SelectionResultNext
			f.selectableContainsOriginTextBoundary = true
		} else {
			result = f.handleSelectParagraph(e.GlobalPosition)
		}
	case GranularlyExtendSelectionEvent:
		result = f.handleGranularlyExtendSelection(e.Forward, e.IsEnd, e.Granularity)
	case DirectionallyExtendSelectionEvent:
		result = f.handleDirectionallyExtendSelection(e.Dx, e.IsEnd, e.Direction)
	}
	if !eqPosition(existingSelectionStart, f.textSelectionStart) || !eqPosition(existingSelectionEnd, f.textSelectionEnd) {
		f.didChangeSelection()
	}
	return result
}

// GetSelectedContent is the text that is selected.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.getSelectedContent
func (f *selectableFragment) GetSelectedContent() *SelectedContent {
	if f.textSelectionStart.IsNone() || f.textSelectionEnd.IsNone() {
		return nil
	}
	start := min(f.textSelectionStart.Value().Offset, f.textSelectionEnd.Value().Offset)
	end := max(f.textSelectionStart.Value().Offset, f.textSelectionEnd.Value().Offset)
	return &SelectedContent{PlainText: f.fullText[start:end]}
}

// GetSelection is the range of the selection.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.getSelection
func (f *selectableFragment) GetSelection() *SelectedContentRange {
	if f.textSelectionStart.IsNone() || f.textSelectionEnd.IsNone() {
		return nil
	}
	return &SelectedContentRange{StartOffset: f.textSelectionStart.Value().Offset, EndOffset: f.textSelectionEnd.Value().Offset}
}

func (f *selectableFragment) didChangeSelection() {
	f.paragraph.MarkNeedsPaint()
	f.updateSelectionGeometry()
}

// updateSelectionStartEdgeByTextBoundary moves the start edge by a text
// boundary; it returns the position the edge goes to.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._updateSelectionStartEdgeByTextBoundary
func (f *selectableFragment) updateSelectionStartEdgeByTextBoundary(textBoundary *textBoundaryRecord, getTextBoundary textBoundaryAtPosition,
	position ui.TextPosition, existingSelectionStart, existingSelectionEnd foundation.Opt[ui.TextPosition]) ui.TextPosition {
	var targetPosition foundation.Opt[ui.TextPosition]
	if textBoundary != nil {
		if f.selectableContainsOriginTextBoundary && existingSelectionStart.IsSome() && existingSelectionEnd.IsSome() {
			start, end := existingSelectionStart.Value(), existingSelectionEnd.Value()
			isSamePosition := position.Offset == end.Offset
			isSelectionInverted := start.Offset > end.Offset
			shouldSwapEdges := !isSamePosition && (isSelectionInverted != (position.Offset > end.Offset))
			if shouldSwapEdges {
				if position.Offset < end.Offset {
					targetPosition = foundation.Some(textBoundary.boundaryStart)
				} else {
					targetPosition = foundation.Some(textBoundary.boundaryEnd)
				}
				// When the selection is inverted by the new position it is necessary to
				// swap the start edge (moving edge) with the end edge (static edge) to
				// maintain the origin text boundary within the selection.
				localTextBoundary := getTextBoundary(end)
				if end.Offset == localTextBoundary.boundaryStart.Offset {
					f.setSelectionPosition(foundation.Some(localTextBoundary.boundaryEnd), true)
				} else {
					f.setSelectionPosition(foundation.Some(localTextBoundary.boundaryStart), true)
				}
			} else {
				switch {
				case position.Offset < end.Offset:
					targetPosition = foundation.Some(textBoundary.boundaryStart)
				case position.Offset > end.Offset:
					targetPosition = foundation.Some(textBoundary.boundaryEnd)
				default:
					// Keep the origin text boundary in bounds when position is at the static edge.
					targetPosition = existingSelectionStart
				}
			}
		} else {
			if existingSelectionEnd.IsSome() {
				// If the end edge exists and the start edge is being moved, then the
				// start edge is moved to encompass the entire text boundary at the new position.
				if position.Offset < existingSelectionEnd.Value().Offset {
					targetPosition = foundation.Some(textBoundary.boundaryStart)
				} else {
					targetPosition = foundation.Some(textBoundary.boundaryEnd)
				}
			} else {
				// Move the start edge to the closest text boundary.
				targetPosition = foundation.Some(f.closestTextBoundary(*textBoundary, position))
			}
		}
	} else {
		// The position is not contained within the current rect. The targetPosition
		// will either be at the end or beginning of the current rect. See
		// SelectionUtilsAdjustDragOffset for a more in depth explanation on this adjustment.
		if f.selectableContainsOriginTextBoundary && existingSelectionStart.IsSome() && existingSelectionEnd.IsSome() {
			start, end := existingSelectionStart.Value(), existingSelectionEnd.Value()
			isSamePosition := position.Offset == end.Offset
			isSelectionInverted := start.Offset > end.Offset
			shouldSwapEdges := !isSamePosition && (isSelectionInverted != (position.Offset > end.Offset))
			if shouldSwapEdges {
				localTextBoundary := getTextBoundary(end)
				if isSelectionInverted {
					f.setSelectionPosition(foundation.Some(localTextBoundary.boundaryEnd), true)
				} else {
					f.setSelectionPosition(foundation.Some(localTextBoundary.boundaryStart), true)
				}
			}
		}
	}
	return targetPosition.OrElse(position)
}

// updateSelectionEndEdgeByTextBoundary moves the end edge by a text boundary.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._updateSelectionEndEdgeByTextBoundary
func (f *selectableFragment) updateSelectionEndEdgeByTextBoundary(textBoundary *textBoundaryRecord, getTextBoundary textBoundaryAtPosition,
	position ui.TextPosition, existingSelectionStart, existingSelectionEnd foundation.Opt[ui.TextPosition]) ui.TextPosition {
	var targetPosition foundation.Opt[ui.TextPosition]
	if textBoundary != nil {
		if f.selectableContainsOriginTextBoundary && existingSelectionStart.IsSome() && existingSelectionEnd.IsSome() {
			start, end := existingSelectionStart.Value(), existingSelectionEnd.Value()
			isSamePosition := position.Offset == start.Offset
			isSelectionInverted := start.Offset > end.Offset
			shouldSwapEdges := !isSamePosition && (isSelectionInverted != (position.Offset < start.Offset))
			if shouldSwapEdges {
				if position.Offset < start.Offset {
					targetPosition = foundation.Some(textBoundary.boundaryStart)
				} else {
					targetPosition = foundation.Some(textBoundary.boundaryEnd)
				}
				// When the selection is inverted by the new position it is necessary to
				// swap the end edge (moving edge) with the start edge (static edge) to
				// maintain the origin text boundary within the selection.
				localTextBoundary := getTextBoundary(start)
				if start.Offset == localTextBoundary.boundaryStart.Offset {
					f.setSelectionPosition(foundation.Some(localTextBoundary.boundaryEnd), false)
				} else {
					f.setSelectionPosition(foundation.Some(localTextBoundary.boundaryStart), false)
				}
			} else {
				switch {
				case position.Offset < start.Offset:
					targetPosition = foundation.Some(textBoundary.boundaryStart)
				case position.Offset > start.Offset:
					targetPosition = foundation.Some(textBoundary.boundaryEnd)
				default:
					// Keep the origin text boundary in bounds when position is at the static edge.
					targetPosition = existingSelectionEnd
				}
			}
		} else {
			if existingSelectionStart.IsSome() {
				// If the start edge exists and the end edge is being moved, then the
				// end edge is moved to encompass the entire text boundary at the new position.
				if position.Offset < existingSelectionStart.Value().Offset {
					targetPosition = foundation.Some(textBoundary.boundaryStart)
				} else {
					targetPosition = foundation.Some(textBoundary.boundaryEnd)
				}
			} else {
				// Move the end edge to the closest text boundary.
				targetPosition = foundation.Some(f.closestTextBoundary(*textBoundary, position))
			}
		}
	} else {
		// The position is not contained within the current rect. The targetPosition
		// will either be at the end or beginning of the current rect.
		if f.selectableContainsOriginTextBoundary && existingSelectionStart.IsSome() && existingSelectionEnd.IsSome() {
			start, end := existingSelectionStart.Value(), existingSelectionEnd.Value()
			isSamePosition := position.Offset == start.Offset
			isSelectionInverted := start.Offset > end.Offset
			shouldSwapEdges := isSelectionInverted != (position.Offset < start.Offset) || isSamePosition
			if shouldSwapEdges {
				localTextBoundary := getTextBoundary(start)
				if isSelectionInverted {
					f.setSelectionPosition(foundation.Some(localTextBoundary.boundaryStart), false)
				} else {
					f.setSelectionPosition(foundation.Some(localTextBoundary.boundaryEnd), false)
				}
			}
		}
	}
	return targetPosition.OrElse(position)
}

// globalToParagraph is the point in the coordinates of the paragraph.
func (f *selectableFragment) globalToParagraph(globalPosition ui.Offset) ui.Offset {
	transform := f.paragraph.GetTransformTo(nil)
	transform.Invert()
	return transform.TransformPoint(globalPosition)
}

// updateSelectionEdgeByTextBoundary updates an edge of the selection by a text
// boundary of the text of this fragment.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._updateSelectionEdgeByTextBoundary
func (f *selectableFragment) updateSelectionEdgeByTextBoundary(globalPosition ui.Offset, isEnd bool, getTextBoundary textBoundaryAtPosition) SelectionResult {
	// When the start/end edges are swapped, i.e. the start is after the end, and
	// the scrollable synthesizes an event for the opposite edge, this will potentially
	// move the opposite edge outside of the origin text boundary and we are unable to recover.
	existingSelectionStart := f.textSelectionStart
	existingSelectionEnd := f.textSelectionEnd

	f.setSelectionPosition(foundation.None[ui.TextPosition](), isEnd)
	localPosition := f.globalToParagraph(globalPosition)
	rect := f.rect()
	if rect.IsEmpty() {
		result := SelectionUtilsGetResultBasedOnRect(rect, localPosition)
		f.setEmptyRectPosition(result, isEnd)
		return result
	}
	adjustedOffset := SelectionUtilsAdjustDragOffset(rect, localPosition, f.paragraph.TextDirection())

	position := f.paragraph.GetPositionForOffset(adjustedOffset)
	// Check if the original local position is within the rect, if it is not then
	// we do not need to look up the text boundary for that position. This is to
	// maintain a selectables selection collapsed at 0 when the local position is
	// not located inside its rect.
	var textBoundary *textBoundaryRecord
	if rect.Contains(localPosition) {
		b := getTextBoundary(position)
		textBoundary = &b
	}
	textBoundary = f.invalidateForeignBoundary(textBoundary)
	var target ui.TextPosition
	if isEnd {
		target = f.updateSelectionEndEdgeByTextBoundary(textBoundary, getTextBoundary, position, existingSelectionStart, existingSelectionEnd)
	} else {
		target = f.updateSelectionStartEdgeByTextBoundary(textBoundary, getTextBoundary, position, existingSelectionStart, existingSelectionEnd)
	}
	targetPosition := f.clampTextPosition(target)

	f.setSelectionPosition(foundation.Some(targetPosition), isEnd)
	return f.resultForTarget(targetPosition, rect, localPosition)
}

// invalidateForeignBoundary drops a text boundary that does not belong to this
// fragment: when the position is located at a placeholder inside of the text,
// the text boundary may be one of another fragment.
func (f *selectableFragment) invalidateForeignBoundary(b *textBoundaryRecord) *textBoundaryRecord {
	if b != nil && ((b.boundaryStart.Offset < f.rng.Start && b.boundaryEnd.Offset <= f.rng.Start) ||
		(b.boundaryStart.Offset >= f.rng.End && b.boundaryEnd.Offset > f.rng.End)) {
		return nil
	}
	return b
}

func (f *selectableFragment) setEmptyRectPosition(result SelectionResult, isEnd bool) {
	if result == SelectionResultNext {
		f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
	} else {
		f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start, Affinity: ui.TextAffinityUpstream}), isEnd)
	}
}

// resultForTarget is the selection result after an edge was moved to the
// position.
func (f *selectableFragment) resultForTarget(target ui.TextPosition, rect ui.Rect, localPosition ui.Offset) SelectionResult {
	if target.Offset == f.rng.End {
		return SelectionResultNext
	}
	if target.Offset == f.rng.Start {
		return SelectionResultPrevious
	}
	// TODO(chunhtai): The geometry information should not be used to determine
	// selection result. This is a workaround to RenderParagraph, where it does
	// not have a way to get accurate text length if its text is truncated due to
	// layout constraint.
	return SelectionUtilsGetResultBasedOnRect(rect, localPosition)
}

// updateSelectionEdge updates an edge of the selection by a character.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._updateSelectionEdge
func (f *selectableFragment) updateSelectionEdge(globalPosition ui.Offset, isEnd bool) SelectionResult {
	f.setSelectionPosition(foundation.None[ui.TextPosition](), isEnd)
	localPosition := f.globalToParagraph(globalPosition)
	rect := f.rect()
	if rect.IsEmpty() {
		result := SelectionUtilsGetResultBasedOnRect(rect, localPosition)
		f.setEmptyRectPosition(result, isEnd)
		return result
	}
	adjustedOffset := SelectionUtilsAdjustDragOffset(rect, localPosition, f.paragraph.TextDirection())
	position := f.clampTextPosition(f.paragraph.GetPositionForOffset(adjustedOffset))
	f.setSelectionPosition(foundation.Some(position), isEnd)
	return f.resultForTarget(position, rect, localPosition)
}

func (f *selectableFragment) closestTextBoundary(b textBoundaryRecord, position ui.TextPosition) ui.TextPosition {
	differenceA := abs(position.Offset - b.boundaryStart.Offset)
	differenceB := abs(position.Offset - b.boundaryEnd.Offset)
	if differenceA < differenceB {
		return b.boundaryStart
	}
	return b.boundaryEnd
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// textOf is the plain text of a paragraph, without the semantics labels.
func textOf(p *RenderParagraph) string {
	return painting.InlineSpanToPlainText(p.Text(), painting.ToPlainTextOpts{ExcludeSemanticsLabels: true})
}

// isPlaceholder reports whether this selectable fragment is a placeholder: its
// paragraph is inside of another paragraph.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._isPlaceholder
func (f *selectableFragment) isPlaceholder() bool {
	for current := f.paragraph.Parent(); !isNil(current); current = current.Parent() {
		if _, ok := current.(*RenderParagraph); ok {
			return true
		}
	}
	return false
}

// getOriginParagraph is the highest paragraph that contains the origin text
// boundary. It should only be called from a fragment that contains the origin
// boundary.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._getOriginParagraph
func (f *selectableFragment) getOriginParagraph() *RenderParagraph {
	// Begin at the parent because it is guaranteed the paragraph containing
	// this selectable fragment contains the origin boundary.
	var originParagraph *RenderParagraph
	for current := f.paragraph.Parent(); !isNil(current); current = current.Parent() {
		p, ok := current.(*RenderParagraph)
		if !ok || p.lastSelectableFragments == nil {
			continue
		}
		contains := false
		for _, fragment := range p.lastSelectableFragments {
			if fragment.selectableContainsOriginTextBoundary {
				contains = true
				originParagraph = p
				break
			}
		}
		if !contains {
			if originParagraph != nil {
				return originParagraph
			}
			return f.paragraph
		}
	}
	if originParagraph != nil {
		return originParagraph
	}
	return f.paragraph
}

// getParagraphContainingPosition is the closest paragraph, from this one up,
// whose rect contains the global position, and the position relative to it.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._getParagraphContainingPosition
func (f *selectableFragment) getParagraphContainingPosition(globalPosition ui.Offset) (*RenderParagraph, ui.Offset, bool) {
	var current RenderObject = f.paragraph
	for !isNil(current) {
		if p, ok := current.(*RenderParagraph); ok {
			transform := p.GetTransformTo(nil)
			transform.Invert()
			local := transform.TransformPoint(globalPosition)
			if p.PaintBounds().Contains(local) {
				return p, local, true
			}
		}
		current = current.Parent()
	}
	return nil, ui.Offset{}, false
}

func (f *selectableFragment) boundingBoxesContains(position ui.Offset) bool {
	for _, rect := range f.BoundingBoxes() {
		if rect.Contains(position) {
			return true
		}
	}
	return false
}

// clampTextPosition keeps the position in the range of the fragment. The
// affinity of range.end is upstream.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._clampTextPosition
func (f *selectableFragment) clampTextPosition(position ui.TextPosition) ui.TextPosition {
	if position.Offset > f.rng.End || (position.Offset == f.rng.End && !position.Affinity.IsUpstream()) {
		return ui.TextPosition{Offset: f.rng.End, Affinity: ui.TextAffinityUpstream}
	}
	if position.Offset < f.rng.Start {
		return ui.TextPosition{Offset: f.rng.Start}
	}
	return position
}

func (f *selectableFragment) setSelectionPosition(position foundation.Opt[ui.TextPosition], isEnd bool) {
	if isEnd {
		f.textSelectionEnd = position
	} else {
		f.textSelectionStart = position
	}
}

func (f *selectableFragment) handleClearSelection() SelectionResult {
	f.textSelectionStart = foundation.None[ui.TextPosition]()
	f.textSelectionEnd = foundation.None[ui.TextPosition]()
	f.selectableContainsOriginTextBoundary = false
	return SelectionResultNone
}

func (f *selectableFragment) handleSelectAll() SelectionResult {
	f.textSelectionStart = foundation.Some(ui.TextPosition{Offset: f.rng.Start})
	f.textSelectionEnd = foundation.Some(ui.TextPosition{Offset: f.rng.End, Affinity: ui.TextAffinityUpstream})
	return SelectionResultNone
}

func (f *selectableFragment) handleSelectTextBoundary(b textBoundaryRecord) SelectionResult {
	// This fragment may not contain the boundary, decide what direction the target
	// fragment is located in. Because fragments are separated by placeholder
	// spans, we also check if the beginning or end of the boundary is touching
	// either edge of this fragment.
	if b.boundaryStart.Offset < f.rng.Start && b.boundaryEnd.Offset <= f.rng.Start {
		return SelectionResultPrevious
	} else if b.boundaryStart.Offset >= f.rng.End && b.boundaryEnd.Offset > f.rng.End {
		return SelectionResultNext
	}
	// Fragments are separated by placeholder span, the text boundary shouldn't
	// expand across fragments.
	f.textSelectionStart = foundation.Some(b.boundaryStart)
	f.textSelectionEnd = foundation.Some(b.boundaryEnd)
	f.selectableContainsOriginTextBoundary = true
	return SelectionResultEnd
}

func intersectRanges(a, b ui.TextRange) (ui.TextRange, bool) {
	startMax := max(a.Start, b.Start)
	endMin := min(a.End, b.End)
	if startMax <= endMin {
		return ui.TextRange{Start: startMax, End: endMin}, true
	}
	return ui.TextRange{}, false
}

func (f *selectableFragment) handleSelectMultiFragmentTextBoundary(b textBoundaryRecord) SelectionResult {
	if b.boundaryStart.Offset < f.rng.Start && b.boundaryEnd.Offset <= f.rng.Start {
		return SelectionResultPrevious
	} else if b.boundaryStart.Offset >= f.rng.End && b.boundaryEnd.Offset > f.rng.End {
		return SelectionResultNext
	}
	boundaryAsRange := ui.TextRange{Start: b.boundaryStart.Offset, End: b.boundaryEnd.Offset}
	if intersect, ok := intersectRanges(f.rng, boundaryAsRange); ok {
		f.textSelectionStart = foundation.Some(ui.TextPosition{Offset: intersect.Start})
		f.textSelectionEnd = foundation.Some(ui.TextPosition{Offset: intersect.End})
		f.selectableContainsOriginTextBoundary = true
		if f.rng.End < b.boundaryEnd.Offset {
			return SelectionResultNext
		}
		return SelectionResultEnd
	}
	return SelectionResultNone
}

func (f *selectableFragment) adjustTextBoundaryAtPosition(textBoundary ui.TextRange, position ui.TextPosition) textBoundaryRecord {
	var start, end ui.TextPosition
	if position.Offset > textBoundary.End {
		start = ui.TextPosition{Offset: position.Offset}
		end = start
	} else {
		start = ui.TextPosition{Offset: textBoundary.Start}
		end = ui.TextPosition{Offset: textBoundary.End, Affinity: ui.TextAffinityUpstream}
	}
	return textBoundaryRecord{boundaryStart: start, boundaryEnd: end}
}

func (f *selectableFragment) handleSelectWord(globalPosition ui.Offset) SelectionResult {
	position := f.paragraph.GetPositionForOffset(f.paragraph.GlobalToLocal(globalPosition, nil))
	if f.positionIsWithinCurrentSelection(position) && !eqPosition(f.textSelectionStart, f.textSelectionEnd) {
		return SelectionResultEnd
	}
	return f.handleSelectTextBoundary(f.getWordBoundaryAtPosition(position))
}

func (f *selectableFragment) getWordBoundaryAtPosition(position ui.TextPosition) textBoundaryRecord {
	word := f.paragraph.GetWordBoundary(position)
	return f.adjustTextBoundaryAtPosition(word, position)
}

func (f *selectableFragment) handleSelectParagraph(globalPosition ui.Offset) SelectionResult {
	localPosition := f.paragraph.GlobalToLocal(globalPosition, nil)
	position := f.paragraph.GetPositionForOffset(localPosition)
	return f.handleSelectMultiFragmentTextBoundary(f.getParagraphBoundaryAtPosition(position, f.fullText))
}

// getPositionInParagraph is the position, in the target paragraph, of the
// middle of the left side of this fragment's paragraph.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._getPositionInParagraph
func (f *selectableFragment) getPositionInParagraph(target *RenderParagraph) ui.TextPosition {
	transform := f.paragraph.GetTransformTo(target)
	localCenter := f.paragraph.PaintBounds().CenterLeft()
	return target.GetPositionForOffset(transform.TransformPoint(localCenter))
}

func (f *selectableFragment) getParagraphBoundaryAtPosition(position ui.TextPosition, text string) textBoundaryRecord {
	paragraphBoundary := services.NewParagraphBoundary(text)
	// Use position.offset - 1 when `position` is at the end of the selectable to
	// retrieve the previous text boundary's location.
	at := position.Offset
	if position.Offset == len(text) || position.Affinity.IsUpstream() {
		at = position.Offset - 1
	}
	paragraphStart, ok := paragraphBoundary.GetLeadingTextBoundaryAt(at)
	if !ok {
		paragraphStart = 0
	}
	paragraphEnd, ok := paragraphBoundary.GetTrailingTextBoundaryAt(position.Offset)
	if !ok {
		paragraphEnd = len(text)
	}
	return f.adjustTextBoundaryAtPosition(ui.TextRange{Start: paragraphStart, End: paragraphEnd}, position)
}

func (f *selectableFragment) getClampedParagraphBoundaryAtPosition(position ui.TextPosition) textBoundaryRecord {
	paragraphBoundary := services.NewParagraphBoundary(f.fullText)
	at := position.Offset
	if position.Offset == len(f.fullText) || position.Affinity.IsUpstream() {
		at = position.Offset - 1
	}
	paragraphStart, ok := paragraphBoundary.GetLeadingTextBoundaryAt(at)
	if !ok {
		paragraphStart = 0
	}
	paragraphEnd, ok := paragraphBoundary.GetTrailingTextBoundaryAt(position.Offset)
	if !ok {
		paragraphEnd = len(f.fullText)
	}
	paragraphStart = min(max(paragraphStart, f.rng.Start), f.rng.End)
	paragraphEnd = min(max(paragraphEnd, f.rng.Start), f.rng.End)
	return f.adjustTextBoundaryAtPosition(ui.TextRange{Start: paragraphStart, End: paragraphEnd}, position)
}

// resultPtr is an optional SelectionResult: the multi-selectable methods answer
// nothing when the caller has to go on with the general handling.
type resultPtr = *SelectionResult

func someResult(r SelectionResult) resultPtr { return &r }

// updateSelectionStartEdgeByMultiSelectableTextBoundary updates the start edge
// by a text boundary that may not be contained within this selectable fragment:
// a boundary can span multiple fragments when the text contains placeholders.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._updateSelectionStartEdgeByMultiSelectableTextBoundary
func (f *selectableFragment) updateSelectionStartEdgeByMultiSelectableTextBoundary(getTextBoundary textBoundaryAtPositionInText, paragraphContainsPosition bool,
	position ui.TextPosition, existingSelectionStart, existingSelectionEnd foundation.Opt[ui.TextPosition]) resultPtr {
	const isEnd = false
	if f.selectableContainsOriginTextBoundary && existingSelectionStart.IsSome() && existingSelectionEnd.IsSome() {
		// If this selectable contains the origin boundary, maintain the existing selection.
		start, end := existingSelectionStart.Value(), existingSelectionEnd.Value()
		forwardSelection := end.Offset >= start.Offset
		if paragraphContainsPosition {
			// When the position is within the root paragraph, swap the start and end
			// edges when the selection is inverted.
			boundaryAtPosition := getTextBoundary(position, f.fullText)
			// To accurately retrieve the origin text boundary when the selection is
			// forward, use existingSelectionEnd.offset - 1: in a forwards selection,
			// existingSelectionEnd marks the end of the origin text boundary.
			originAt := end
			if forwardSelection {
				originAt = ui.TextPosition{Offset: end.Offset - 1, Affinity: end.Affinity}
			}
			originTextBoundary := getTextBoundary(originAt, f.fullText)
			var targetPosition ui.TextPosition
			pivotOffset := originTextBoundary.boundaryStart.Offset
			if forwardSelection {
				pivotOffset = originTextBoundary.boundaryEnd.Offset
			}
			shouldSwapEdges := !forwardSelection != (position.Offset > pivotOffset)
			switch {
			case position.Offset < pivotOffset:
				targetPosition = boundaryAtPosition.boundaryStart
			case position.Offset > pivotOffset:
				targetPosition = boundaryAtPosition.boundaryEnd
			default:
				// Keep the origin text boundary in bounds when position is at the static edge.
				if forwardSelection {
					targetPosition = start
				} else {
					targetPosition = end
				}
			}
			if shouldSwapEdges {
				swap := originTextBoundary.boundaryEnd
				if forwardSelection {
					swap = originTextBoundary.boundaryStart
				}
				f.setSelectionPosition(foundation.Some(f.clampTextPosition(swap)), true)
			}
			f.setSelectionPosition(foundation.Some(f.clampTextPosition(targetPosition)), isEnd)
			finalSelectionIsForward := f.textSelectionEnd.Value().Offset >= f.textSelectionStart.Value().Offset
			if boundaryAtPosition.boundaryStart.Offset > f.rng.End && boundaryAtPosition.boundaryEnd.Offset > f.rng.End {
				return someResult(SelectionResultNext)
			}
			if boundaryAtPosition.boundaryStart.Offset < f.rng.Start && boundaryAtPosition.boundaryEnd.Offset < f.rng.Start {
				return someResult(SelectionResultPrevious)
			}
			if finalSelectionIsForward {
				if boundaryAtPosition.boundaryStart.Offset >= originTextBoundary.boundaryStart.Offset {
					return someResult(SelectionResultEnd)
				}
				if boundaryAtPosition.boundaryStart.Offset < originTextBoundary.boundaryStart.Offset {
					return someResult(SelectionResultPrevious)
				}
			} else {
				if boundaryAtPosition.boundaryEnd.Offset <= originTextBoundary.boundaryEnd.Offset {
					return someResult(SelectionResultEnd)
				}
				if boundaryAtPosition.boundaryEnd.Offset > originTextBoundary.boundaryEnd.Offset {
					return someResult(SelectionResultNext)
				}
			}
		} else {
			// When the drag position is not contained within the root paragraph,
			// swap the edges when the selection changes direction.
			clampedPosition := f.clampTextPosition(position)
			originAt := end
			if forwardSelection {
				originAt = ui.TextPosition{Offset: end.Offset - 1, Affinity: end.Affinity}
			}
			originTextBoundary := getTextBoundary(originAt, f.fullText)
			if forwardSelection && clampedPosition.Offset == f.rng.Start {
				f.setSelectionPosition(foundation.Some(clampedPosition), isEnd)
				return someResult(SelectionResultPrevious)
			}
			if !forwardSelection && clampedPosition.Offset == f.rng.End {
				f.setSelectionPosition(foundation.Some(clampedPosition), isEnd)
				return someResult(SelectionResultNext)
			}
			if forwardSelection && clampedPosition.Offset == f.rng.End {
				f.setSelectionPosition(foundation.Some(f.clampTextPosition(originTextBoundary.boundaryStart)), true)
				f.setSelectionPosition(foundation.Some(clampedPosition), isEnd)
				return someResult(SelectionResultNext)
			}
			if !forwardSelection && clampedPosition.Offset == f.rng.Start {
				f.setSelectionPosition(foundation.Some(f.clampTextPosition(originTextBoundary.boundaryEnd)), true)
				f.setSelectionPosition(foundation.Some(clampedPosition), isEnd)
				return someResult(SelectionResultPrevious)
			}
		}
	} else {
		// A paragraph boundary may not be completely contained within this root
		// selectable fragment. Keep searching until we find the end of the
		// boundary. Do not search when the current drag position is on a
		// placeholder to allow traversal to reach that placeholder.
		positionOnPlaceholder := f.paragraph.GetWordBoundary(position).TextInside(f.fullText) == placeholderCharacter
		if !paragraphContainsPosition || positionOnPlaceholder {
			return nil
		}
		if existingSelectionEnd.IsSome() {
			end := existingSelectionEnd.Value()
			boundaryAtPosition := getTextBoundary(position, f.fullText)
			backwardSelection := (existingSelectionStart.IsNone() && end.Offset == f.rng.Start) ||
				(eqPosition(existingSelectionStart, existingSelectionEnd) && end.Offset == f.rng.Start) ||
				(existingSelectionStart.IsSome() && existingSelectionStart.Value().Offset > end.Offset)
			if boundaryAtPosition.boundaryStart.Offset < f.rng.Start && boundaryAtPosition.boundaryEnd.Offset < f.rng.Start {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
				return someResult(SelectionResultPrevious)
			}
			if boundaryAtPosition.boundaryStart.Offset > f.rng.End && boundaryAtPosition.boundaryEnd.Offset > f.rng.End {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
				return someResult(SelectionResultNext)
			}
			if backwardSelection {
				if boundaryAtPosition.boundaryEnd.Offset <= f.rng.End {
					f.setSelectionPosition(foundation.Some(f.clampTextPosition(boundaryAtPosition.boundaryEnd)), isEnd)
					return someResult(SelectionResultEnd)
				}
				if boundaryAtPosition.boundaryEnd.Offset > f.rng.End {
					f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
					return someResult(SelectionResultNext)
				}
			} else {
				f.setSelectionPosition(foundation.Some(f.clampTextPosition(boundaryAtPosition.boundaryStart)), isEnd)
				if boundaryAtPosition.boundaryStart.Offset < f.rng.Start {
					return someResult(SelectionResultPrevious)
				}
				if boundaryAtPosition.boundaryStart.Offset >= f.rng.Start {
					return someResult(SelectionResultEnd)
				}
			}
		}
	}
	return nil
}

// updateSelectionEndEdgeByMultiSelectableTextBoundary is the same for the end
// edge.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._updateSelectionEndEdgeByMultiSelectableTextBoundary
func (f *selectableFragment) updateSelectionEndEdgeByMultiSelectableTextBoundary(getTextBoundary textBoundaryAtPositionInText, paragraphContainsPosition bool,
	position ui.TextPosition, existingSelectionStart, existingSelectionEnd foundation.Opt[ui.TextPosition]) resultPtr {
	const isEnd = true
	if f.selectableContainsOriginTextBoundary && existingSelectionStart.IsSome() && existingSelectionEnd.IsSome() {
		start, end := existingSelectionStart.Value(), existingSelectionEnd.Value()
		forwardSelection := end.Offset >= start.Offset
		if paragraphContainsPosition {
			boundaryAtPosition := getTextBoundary(position, f.fullText)
			// To accurately retrieve the origin text boundary when the selection is
			// backwards, use existingSelectionStart.offset - 1: in a backwards
			// selection, existingSelectionStart marks the end of the origin text boundary.
			originAt := start
			if !forwardSelection {
				originAt = ui.TextPosition{Offset: start.Offset - 1, Affinity: start.Affinity}
			}
			originTextBoundary := getTextBoundary(originAt, f.fullText)
			var targetPosition ui.TextPosition
			pivotOffset := originTextBoundary.boundaryEnd.Offset
			if forwardSelection {
				pivotOffset = originTextBoundary.boundaryStart.Offset
			}
			shouldSwapEdges := !forwardSelection != (position.Offset < pivotOffset)
			switch {
			case position.Offset < pivotOffset:
				targetPosition = boundaryAtPosition.boundaryStart
			case position.Offset > pivotOffset:
				targetPosition = boundaryAtPosition.boundaryEnd
			default:
				// Keep the origin text boundary in bounds when position is at the static edge.
				if forwardSelection {
					targetPosition = end
				} else {
					targetPosition = start
				}
			}
			if shouldSwapEdges {
				swap := originTextBoundary.boundaryStart
				if forwardSelection {
					swap = originTextBoundary.boundaryEnd
				}
				f.setSelectionPosition(foundation.Some(f.clampTextPosition(swap)), false)
			}
			f.setSelectionPosition(foundation.Some(f.clampTextPosition(targetPosition)), isEnd)
			finalSelectionIsForward := f.textSelectionEnd.Value().Offset >= f.textSelectionStart.Value().Offset
			if boundaryAtPosition.boundaryStart.Offset > f.rng.End && boundaryAtPosition.boundaryEnd.Offset > f.rng.End {
				return someResult(SelectionResultNext)
			}
			if boundaryAtPosition.boundaryStart.Offset < f.rng.Start && boundaryAtPosition.boundaryEnd.Offset < f.rng.Start {
				return someResult(SelectionResultPrevious)
			}
			if finalSelectionIsForward {
				if boundaryAtPosition.boundaryEnd.Offset <= originTextBoundary.boundaryEnd.Offset {
					return someResult(SelectionResultEnd)
				}
				if boundaryAtPosition.boundaryEnd.Offset > originTextBoundary.boundaryEnd.Offset {
					return someResult(SelectionResultNext)
				}
			} else {
				if boundaryAtPosition.boundaryStart.Offset >= originTextBoundary.boundaryStart.Offset {
					return someResult(SelectionResultEnd)
				}
				if boundaryAtPosition.boundaryStart.Offset < originTextBoundary.boundaryStart.Offset {
					return someResult(SelectionResultPrevious)
				}
			}
		} else {
			clampedPosition := f.clampTextPosition(position)
			originAt := start
			if !forwardSelection {
				originAt = ui.TextPosition{Offset: start.Offset - 1, Affinity: start.Affinity}
			}
			originTextBoundary := getTextBoundary(originAt, f.fullText)
			if forwardSelection && clampedPosition.Offset == f.rng.Start {
				f.setSelectionPosition(foundation.Some(f.clampTextPosition(originTextBoundary.boundaryEnd)), false)
				f.setSelectionPosition(foundation.Some(clampedPosition), isEnd)
				return someResult(SelectionResultPrevious)
			}
			if !forwardSelection && clampedPosition.Offset == f.rng.End {
				f.setSelectionPosition(foundation.Some(f.clampTextPosition(originTextBoundary.boundaryStart)), false)
				f.setSelectionPosition(foundation.Some(clampedPosition), isEnd)
				return someResult(SelectionResultNext)
			}
			if forwardSelection && clampedPosition.Offset == f.rng.End {
				f.setSelectionPosition(foundation.Some(clampedPosition), isEnd)
				return someResult(SelectionResultNext)
			}
			if !forwardSelection && clampedPosition.Offset == f.rng.Start {
				f.setSelectionPosition(foundation.Some(clampedPosition), isEnd)
				return someResult(SelectionResultPrevious)
			}
		}
	} else {
		positionOnPlaceholder := f.paragraph.GetWordBoundary(position).TextInside(f.fullText) == placeholderCharacter
		if !paragraphContainsPosition || positionOnPlaceholder {
			return nil
		}
		if existingSelectionStart.IsSome() {
			start := existingSelectionStart.Value()
			boundaryAtPosition := getTextBoundary(position, f.fullText)
			backwardSelection := (existingSelectionEnd.IsNone() && start.Offset == f.rng.End) ||
				(eqPosition(existingSelectionStart, existingSelectionEnd) && start.Offset == f.rng.End) ||
				(existingSelectionEnd.IsSome() && start.Offset > existingSelectionEnd.Value().Offset)
			if boundaryAtPosition.boundaryStart.Offset < f.rng.Start && boundaryAtPosition.boundaryEnd.Offset < f.rng.Start {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
				return someResult(SelectionResultPrevious)
			}
			if boundaryAtPosition.boundaryStart.Offset > f.rng.End && boundaryAtPosition.boundaryEnd.Offset > f.rng.End {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
				return someResult(SelectionResultNext)
			}
			if backwardSelection {
				f.setSelectionPosition(foundation.Some(f.clampTextPosition(boundaryAtPosition.boundaryStart)), isEnd)
				if boundaryAtPosition.boundaryStart.Offset < f.rng.Start {
					return someResult(SelectionResultPrevious)
				}
				if boundaryAtPosition.boundaryStart.Offset >= f.rng.Start {
					return someResult(SelectionResultEnd)
				}
			} else {
				if boundaryAtPosition.boundaryEnd.Offset <= f.rng.End {
					f.setSelectionPosition(foundation.Some(f.clampTextPosition(boundaryAtPosition.boundaryEnd)), isEnd)
					return someResult(SelectionResultEnd)
				}
				if boundaryAtPosition.boundaryEnd.Offset > f.rng.End {
					f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
					return someResult(SelectionResultNext)
				}
			}
		}
	}
	return nil
}

func (f *selectableFragment) placeholderRange(p *RenderParagraph) ui.TextRange {
	pos := f.getPositionInParagraph(p)
	return ui.TextRange{Start: pos.Offset, End: pos.Offset + placeholderLength}
}

// updateSelectionStartEdgeAtPlaceholderByMultiSelectableTextBoundary updates the
// start edge by a text boundary that may not be contained within this
// selectable fragment. Differently from
// updateSelectionStartEdgeByMultiSelectableTextBoundary, to maintain the origin
// text boundary selected at a placeholder, this fragment must be aware of the
// paragraph that closely encompasses the complete origin text boundary.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._updateSelectionStartEdgeAtPlaceholderByMultiSelectableTextBoundary
func (f *selectableFragment) updateSelectionStartEdgeAtPlaceholderByMultiSelectableTextBoundary(getTextBoundary textBoundaryAtPositionInText, globalPosition ui.Offset,
	paragraphContainsPosition bool, position ui.TextPosition, existingSelectionStart, existingSelectionEnd foundation.Opt[ui.TextPosition]) resultPtr {
	const isEnd = false
	if f.selectableContainsOriginTextBoundary && existingSelectionStart.IsSome() && existingSelectionEnd.IsSome() {
		start, end := existingSelectionStart.Value(), existingSelectionEnd.Value()
		forwardSelection := end.Offset >= start.Offset
		originParagraph := f.getOriginParagraph()
		if originParagraph == f.paragraph {
			return f.updateSelectionStartEdgeByMultiSelectableTextBoundary(getTextBoundary, paragraphContainsPosition, position, existingSelectionStart, existingSelectionEnd)
		}
		originTransform := originParagraph.GetTransformTo(nil)
		originTransform.Invert()
		originParagraphLocalPosition := originTransform.TransformPoint(globalPosition)
		positionWithinOriginParagraph := originParagraph.PaintBounds().Contains(originParagraphLocalPosition)
		positionRelativeToOriginParagraph := originParagraph.GetPositionForOffset(originParagraphLocalPosition)
		if positionWithinOriginParagraph {
			// When the selection is inverted by the new position it is necessary to
			// swap the start edge (moving edge) with the end edge (static edge) to
			// maintain the origin text boundary within the selection.
			originText := textOf(originParagraph)
			boundaryAtPosition := getTextBoundary(positionRelativeToOriginParagraph, originText)
			originTextBoundary := getTextBoundary(f.getPositionInParagraph(originParagraph), originText)
			var targetPosition ui.TextPosition
			pivotOffset := originTextBoundary.boundaryStart.Offset
			if forwardSelection {
				pivotOffset = originTextBoundary.boundaryEnd.Offset
			}
			shouldSwapEdges := !forwardSelection != (positionRelativeToOriginParagraph.Offset > pivotOffset)
			switch {
			case positionRelativeToOriginParagraph.Offset < pivotOffset:
				targetPosition = boundaryAtPosition.boundaryStart
			case positionRelativeToOriginParagraph.Offset > pivotOffset:
				targetPosition = boundaryAtPosition.boundaryEnd
			default:
				// Keep the origin text boundary in bounds when position is at the static edge.
				targetPosition = start
			}
			if shouldSwapEdges {
				f.setSelectionPosition(existingSelectionStart, true)
			}
			f.setSelectionPosition(foundation.Some(f.clampTextPosition(targetPosition)), isEnd)
			finalSelectionIsForward := f.textSelectionEnd.Value().Offset >= f.textSelectionStart.Value().Offset
			placeholderRange := f.placeholderRange(originParagraph)
			if boundaryAtPosition.boundaryStart.Offset > placeholderRange.End && boundaryAtPosition.boundaryEnd.Offset > placeholderRange.End {
				return someResult(SelectionResultNext)
			}
			if boundaryAtPosition.boundaryStart.Offset < placeholderRange.Start && boundaryAtPosition.boundaryEnd.Offset < placeholderRange.Start {
				return someResult(SelectionResultPrevious)
			}
			if finalSelectionIsForward {
				if boundaryAtPosition.boundaryEnd.Offset <= originTextBoundary.boundaryEnd.Offset {
					return someResult(SelectionResultEnd)
				}
				if boundaryAtPosition.boundaryEnd.Offset > originTextBoundary.boundaryEnd.Offset {
					return someResult(SelectionResultNext)
				}
			} else {
				if boundaryAtPosition.boundaryStart.Offset >= originTextBoundary.boundaryStart.Offset {
					return someResult(SelectionResultEnd)
				}
				if boundaryAtPosition.boundaryStart.Offset < originTextBoundary.boundaryStart.Offset {
					return someResult(SelectionResultPrevious)
				}
			}
		} else {
			// When the drag position is not contained within the origin paragraph,
			// swap the edges when the selection changes direction.
			adjustedOffset := SelectionUtilsAdjustDragOffset(originParagraph.PaintBounds(), originParagraphLocalPosition, f.paragraph.TextDirection())
			adjustedPositionRelativeToOriginParagraph := originParagraph.GetPositionForOffset(adjustedOffset)
			placeholderRange := f.placeholderRange(originParagraph)
			if forwardSelection && adjustedPositionRelativeToOriginParagraph.Offset <= placeholderRange.Start {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
				return someResult(SelectionResultPrevious)
			}
			if !forwardSelection && adjustedPositionRelativeToOriginParagraph.Offset >= placeholderRange.End {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
				return someResult(SelectionResultNext)
			}
			if forwardSelection && adjustedPositionRelativeToOriginParagraph.Offset >= placeholderRange.End {
				f.setSelectionPosition(existingSelectionStart, true)
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
				return someResult(SelectionResultNext)
			}
			if !forwardSelection && adjustedPositionRelativeToOriginParagraph.Offset <= placeholderRange.Start {
				f.setSelectionPosition(existingSelectionStart, true)
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
				return someResult(SelectionResultPrevious)
			}
		}
	} else {
		// When the drag position is somewhere on the root text and not a
		// placeholder, traverse the selectable fragments relative to the paragraph
		// that contains the drag position.
		if paragraphContainsPosition {
			return f.updateSelectionStartEdgeByMultiSelectableTextBoundary(getTextBoundary, paragraphContainsPosition, position, existingSelectionStart, existingSelectionEnd)
		}
		if existingSelectionEnd.IsSome() {
			end := existingSelectionEnd.Value()
			targetParagraph, targetLocal, ok := f.getParagraphContainingPosition(globalPosition)
			if !ok {
				return nil
			}
			positionRelativeToTargetParagraph := targetParagraph.GetPositionForOffset(targetLocal)
			targetText := textOf(targetParagraph)
			if targetParagraph.GetWordBoundary(positionRelativeToTargetParagraph).TextInside(targetText) == placeholderCharacter {
				return nil
			}
			backwardSelection := (existingSelectionStart.IsNone() && end.Offset == f.rng.Start) ||
				(eqPosition(existingSelectionStart, existingSelectionEnd) && end.Offset == f.rng.Start) ||
				(existingSelectionStart.IsSome() && existingSelectionStart.Value().Offset > end.Offset)
			boundaryAtPositionRelativeToTargetParagraph := getTextBoundary(positionRelativeToTargetParagraph, targetText)
			placeholderRange := f.placeholderRange(targetParagraph)
			b := boundaryAtPositionRelativeToTargetParagraph
			if b.boundaryStart.Offset < placeholderRange.Start && b.boundaryEnd.Offset < placeholderRange.Start {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
				return someResult(SelectionResultPrevious)
			}
			if b.boundaryStart.Offset > placeholderRange.End && b.boundaryEnd.Offset > placeholderRange.End {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
				return someResult(SelectionResultNext)
			}
			if backwardSelection {
				if b.boundaryEnd.Offset <= placeholderRange.End {
					f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
					return someResult(SelectionResultEnd)
				}
				if b.boundaryEnd.Offset > placeholderRange.End {
					f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
					return someResult(SelectionResultNext)
				}
			} else {
				if b.boundaryStart.Offset >= placeholderRange.Start {
					f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
					return someResult(SelectionResultEnd)
				}
				if b.boundaryStart.Offset < placeholderRange.Start {
					f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
					return someResult(SelectionResultPrevious)
				}
			}
		}
	}
	return nil
}

// updateSelectionEndEdgeAtPlaceholderByMultiSelectableTextBoundary is the same
// for the end edge.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._updateSelectionEndEdgeAtPlaceholderByMultiSelectableTextBoundary
func (f *selectableFragment) updateSelectionEndEdgeAtPlaceholderByMultiSelectableTextBoundary(getTextBoundary textBoundaryAtPositionInText, globalPosition ui.Offset,
	paragraphContainsPosition bool, position ui.TextPosition, existingSelectionStart, existingSelectionEnd foundation.Opt[ui.TextPosition]) resultPtr {
	const isEnd = true
	if f.selectableContainsOriginTextBoundary && existingSelectionStart.IsSome() && existingSelectionEnd.IsSome() {
		start, end := existingSelectionStart.Value(), existingSelectionEnd.Value()
		forwardSelection := end.Offset >= start.Offset
		originParagraph := f.getOriginParagraph()
		if originParagraph == f.paragraph {
			return f.updateSelectionEndEdgeByMultiSelectableTextBoundary(getTextBoundary, paragraphContainsPosition, position, existingSelectionStart, existingSelectionEnd)
		}
		originTransform := originParagraph.GetTransformTo(nil)
		originTransform.Invert()
		originParagraphLocalPosition := originTransform.TransformPoint(globalPosition)
		positionWithinOriginParagraph := originParagraph.PaintBounds().Contains(originParagraphLocalPosition)
		positionRelativeToOriginParagraph := originParagraph.GetPositionForOffset(originParagraphLocalPosition)
		if positionWithinOriginParagraph {
			originText := textOf(originParagraph)
			boundaryAtPosition := getTextBoundary(positionRelativeToOriginParagraph, originText)
			originTextBoundary := getTextBoundary(f.getPositionInParagraph(originParagraph), originText)
			var targetPosition ui.TextPosition
			pivotOffset := originTextBoundary.boundaryEnd.Offset
			if forwardSelection {
				pivotOffset = originTextBoundary.boundaryStart.Offset
			}
			shouldSwapEdges := !forwardSelection != (positionRelativeToOriginParagraph.Offset < pivotOffset)
			switch {
			case positionRelativeToOriginParagraph.Offset < pivotOffset:
				targetPosition = boundaryAtPosition.boundaryStart
			case positionRelativeToOriginParagraph.Offset > pivotOffset:
				targetPosition = boundaryAtPosition.boundaryEnd
			default:
				// Keep the origin text boundary in bounds when position is at the static edge.
				targetPosition = end
			}
			if shouldSwapEdges {
				f.setSelectionPosition(existingSelectionEnd, false)
			}
			f.setSelectionPosition(foundation.Some(f.clampTextPosition(targetPosition)), isEnd)
			finalSelectionIsForward := f.textSelectionEnd.Value().Offset >= f.textSelectionStart.Value().Offset
			placeholderRange := f.placeholderRange(originParagraph)
			if boundaryAtPosition.boundaryStart.Offset > placeholderRange.End && boundaryAtPosition.boundaryEnd.Offset > placeholderRange.End {
				return someResult(SelectionResultNext)
			}
			if boundaryAtPosition.boundaryStart.Offset < placeholderRange.Start && boundaryAtPosition.boundaryEnd.Offset < placeholderRange.Start {
				return someResult(SelectionResultPrevious)
			}
			if finalSelectionIsForward {
				if boundaryAtPosition.boundaryEnd.Offset <= originTextBoundary.boundaryEnd.Offset {
					return someResult(SelectionResultEnd)
				}
				if boundaryAtPosition.boundaryEnd.Offset > originTextBoundary.boundaryEnd.Offset {
					return someResult(SelectionResultNext)
				}
			} else {
				if boundaryAtPosition.boundaryStart.Offset >= originTextBoundary.boundaryStart.Offset {
					return someResult(SelectionResultEnd)
				}
				if boundaryAtPosition.boundaryStart.Offset < originTextBoundary.boundaryStart.Offset {
					return someResult(SelectionResultPrevious)
				}
			}
		} else {
			adjustedOffset := SelectionUtilsAdjustDragOffset(originParagraph.PaintBounds(), originParagraphLocalPosition, f.paragraph.TextDirection())
			adjustedPositionRelativeToOriginParagraph := originParagraph.GetPositionForOffset(adjustedOffset)
			placeholderRange := f.placeholderRange(originParagraph)
			if forwardSelection && adjustedPositionRelativeToOriginParagraph.Offset <= placeholderRange.Start {
				f.setSelectionPosition(existingSelectionEnd, false)
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
				return someResult(SelectionResultPrevious)
			}
			if !forwardSelection && adjustedPositionRelativeToOriginParagraph.Offset >= placeholderRange.End {
				f.setSelectionPosition(existingSelectionEnd, false)
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
				return someResult(SelectionResultNext)
			}
			if forwardSelection && adjustedPositionRelativeToOriginParagraph.Offset >= placeholderRange.End {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
				return someResult(SelectionResultNext)
			}
			if !forwardSelection && adjustedPositionRelativeToOriginParagraph.Offset <= placeholderRange.Start {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
				return someResult(SelectionResultPrevious)
			}
		}
	} else {
		if paragraphContainsPosition {
			return f.updateSelectionEndEdgeByMultiSelectableTextBoundary(getTextBoundary, paragraphContainsPosition, position, existingSelectionStart, existingSelectionEnd)
		}
		if existingSelectionStart.IsSome() {
			start := existingSelectionStart.Value()
			targetParagraph, targetLocal, ok := f.getParagraphContainingPosition(globalPosition)
			if !ok {
				return nil
			}
			positionRelativeToTargetParagraph := targetParagraph.GetPositionForOffset(targetLocal)
			targetText := textOf(targetParagraph)
			if targetParagraph.GetWordBoundary(positionRelativeToTargetParagraph).TextInside(targetText) == placeholderCharacter {
				return nil
			}
			backwardSelection := (existingSelectionEnd.IsNone() && start.Offset == f.rng.End) ||
				(eqPosition(existingSelectionStart, existingSelectionEnd) && start.Offset == f.rng.End) ||
				(existingSelectionEnd.IsSome() && start.Offset > existingSelectionEnd.Value().Offset)
			b := getTextBoundary(positionRelativeToTargetParagraph, targetText)
			placeholderRange := f.placeholderRange(targetParagraph)
			if b.boundaryStart.Offset < placeholderRange.Start && b.boundaryEnd.Offset < placeholderRange.Start {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
				return someResult(SelectionResultPrevious)
			}
			if b.boundaryStart.Offset > placeholderRange.End && b.boundaryEnd.Offset > placeholderRange.End {
				f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
				return someResult(SelectionResultNext)
			}
			if backwardSelection {
				if b.boundaryStart.Offset >= placeholderRange.Start {
					f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
					return someResult(SelectionResultEnd)
				}
				if b.boundaryStart.Offset < placeholderRange.Start {
					f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.Start}), isEnd)
					return someResult(SelectionResultPrevious)
				}
			} else {
				if b.boundaryEnd.Offset <= placeholderRange.End {
					f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
					return someResult(SelectionResultEnd)
				}
				if b.boundaryEnd.Offset > placeholderRange.End {
					f.setSelectionPosition(foundation.Some(ui.TextPosition{Offset: f.rng.End}), isEnd)
					return someResult(SelectionResultNext)
				}
			}
		}
	}
	return nil
}

// updateSelectionEdgeByMultiSelectableTextBoundary updates an edge by a text
// boundary that may span several fragments (the paragraph granularity).
//
// Flutter: rendering/paragraph.dart _SelectableFragment._updateSelectionEdgeByMultiSelectableTextBoundary
func (f *selectableFragment) updateSelectionEdgeByMultiSelectableTextBoundary(globalPosition ui.Offset, isEnd bool,
	getTextBoundary textBoundaryAtPositionInText, getClampedTextBoundary textBoundaryAtPosition) SelectionResult {
	// When the start/end edges are swapped, i.e. the start is after the end, and
	// the scrollable synthesizes an event for the opposite edge, this will
	// potentially move the opposite edge outside of the origin text boundary and
	// we are unable to recover.
	existingSelectionStart := f.textSelectionStart
	existingSelectionEnd := f.textSelectionEnd

	f.setSelectionPosition(foundation.None[ui.TextPosition](), isEnd)
	localPosition := f.globalToParagraph(globalPosition)
	rect := f.rect()
	if rect.IsEmpty() {
		result := SelectionUtilsGetResultBasedOnRect(rect, localPosition)
		f.setEmptyRectPosition(result, isEnd)
		return result
	}
	adjustedOffset := SelectionUtilsAdjustDragOffset(rect, localPosition, f.paragraph.TextDirection())
	adjustedOffsetRelativeToParagraph := SelectionUtilsAdjustDragOffset(f.paragraph.PaintBounds(), localPosition, f.paragraph.TextDirection())

	position := f.paragraph.GetPositionForOffset(adjustedOffset)
	positionInFullText := f.paragraph.GetPositionForOffset(adjustedOffsetRelativeToParagraph)
	paragraphContainsPosition := f.paragraph.PaintBounds().Contains(localPosition)

	var result resultPtr
	switch {
	case f.isPlaceholder() && isEnd:
		result = f.updateSelectionEndEdgeAtPlaceholderByMultiSelectableTextBoundary(getTextBoundary, globalPosition, paragraphContainsPosition, positionInFullText, existingSelectionStart, existingSelectionEnd)
	case f.isPlaceholder():
		result = f.updateSelectionStartEdgeAtPlaceholderByMultiSelectableTextBoundary(getTextBoundary, globalPosition, paragraphContainsPosition, positionInFullText, existingSelectionStart, existingSelectionEnd)
	case isEnd:
		result = f.updateSelectionEndEdgeByMultiSelectableTextBoundary(getTextBoundary, paragraphContainsPosition, positionInFullText, existingSelectionStart, existingSelectionEnd)
	default:
		result = f.updateSelectionStartEdgeByMultiSelectableTextBoundary(getTextBoundary, paragraphContainsPosition, positionInFullText, existingSelectionStart, existingSelectionEnd)
	}
	if result != nil {
		return *result
	}
	// Check if the original local position is within the rect, if it is not then
	// we do not need to look up the text boundary for that position. This is to
	// maintain a selectables selection collapsed at 0 when the local position is
	// not located inside its rect.
	var textBoundary *textBoundaryRecord
	if f.boundingBoxesContains(localPosition) {
		b := getClampedTextBoundary(position)
		textBoundary = &b
	}
	textBoundary = f.invalidateForeignBoundary(textBoundary)
	var target ui.TextPosition
	if isEnd {
		target = f.updateSelectionEndEdgeByTextBoundary(textBoundary, getClampedTextBoundary, position, existingSelectionStart, existingSelectionEnd)
	} else {
		target = f.updateSelectionStartEdgeByTextBoundary(textBoundary, getClampedTextBoundary, position, existingSelectionStart, existingSelectionEnd)
	}
	targetPosition := f.clampTextPosition(target)

	f.setSelectionPosition(foundation.Some(targetPosition), isEnd)
	return f.resultForTarget(targetPosition, rect, localPosition)
}

// handleDirectionallyExtendSelection moves an edge of the selection to the
// previous or next line, or by a character, to the horizontal baseline.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._handleDirectionallyExtendSelection
func (f *selectableFragment) handleDirectionallyExtendSelection(horizontalBaseline float64, isExtent bool, movement SelectionExtendDirection) SelectionResult {
	transform := f.paragraph.GetTransformTo(nil)
	if transform.Invert() == 0.0 {
		switch movement {
		case SelectionExtendDirectionPreviousLine, SelectionExtendDirectionBackward:
			return SelectionResultPrevious
		default:
			return SelectionResultNext
		}
	}
	baselineInParagraphCoordinates := transform.TransformPoint(ui.Offset{Dx: horizontalBaseline}).Dx
	if foundation.DebugMode && math.IsNaN(baselineInParagraphCoordinates) {
		panic(errors.New(tagSelectionBaselineNaN, "the horizontal baseline in the paragraph coordinates is NaN"))
	}
	var newPosition ui.TextPosition
	var result SelectionResult
	switch movement {
	case SelectionExtendDirectionPreviousLine, SelectionExtendDirectionNextLine:
		if foundation.DebugMode && (f.textSelectionEnd.IsNone() || f.textSelectionStart.IsNone()) {
			panic(errors.New(tagSelectionNotStarted, "extending the selection by a line needs a selection"))
		}
		targetedEdge := f.textSelectionStart.Value()
		if isExtent {
			targetedEdge = f.textSelectionEnd.Value()
		}
		newPosition, result = f.handleVerticalMovement(targetedEdge, baselineInParagraphCoordinates, movement == SelectionExtendDirectionNextLine)
	default:
		if f.textSelectionEnd.IsNone() {
			if movement == SelectionExtendDirectionForward {
				f.textSelectionEnd = foundation.Some(ui.TextPosition{Offset: f.rng.Start})
			} else {
				f.textSelectionEnd = foundation.Some(ui.TextPosition{Offset: f.rng.End, Affinity: ui.TextAffinityUpstream})
			}
		}
		if f.textSelectionStart.IsNone() {
			f.textSelectionStart = f.textSelectionEnd
		}
		targetedEdge := f.textSelectionStart.Value()
		if isExtent {
			targetedEdge = f.textSelectionEnd.Value()
		}
		edgeOffsetInParagraphCoordinates := f.paragraph.getOffsetForPosition(targetedEdge)
		baselineOffsetInParagraphCoordinates := ui.Offset{
			Dx: baselineInParagraphCoordinates,
			// Use half of line height to point to the middle of the line.
			Dy: edgeOffsetInParagraphCoordinates.Dy - f.paragraph.textPainter.PreferredLineHeight()/2,
		}
		newPosition = f.paragraph.GetPositionForOffset(baselineOffsetInParagraphCoordinates)
		result = SelectionResultEnd
	}
	if isExtent {
		f.textSelectionEnd = foundation.Some(newPosition)
	} else {
		f.textSelectionStart = foundation.Some(newPosition)
	}
	return result
}

// handleGranularlyExtendSelection moves an edge of the selection by a
// granularity, forward or backward.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._handleGranularlyExtendSelection
func (f *selectableFragment) handleGranularlyExtendSelection(forward, isExtent bool, granularity TextGranularity) SelectionResult {
	if f.textSelectionEnd.IsNone() {
		if forward {
			f.textSelectionEnd = foundation.Some(ui.TextPosition{Offset: f.rng.Start})
		} else {
			f.textSelectionEnd = foundation.Some(ui.TextPosition{Offset: f.rng.End, Affinity: ui.TextAffinityUpstream})
		}
	}
	if f.textSelectionStart.IsNone() {
		f.textSelectionStart = f.textSelectionEnd
	}
	targetedEdge := f.textSelectionStart.Value()
	if isExtent {
		targetedEdge = f.textSelectionEnd.Value()
	}
	if forward && targetedEdge.Offset == f.rng.End {
		return SelectionResultNext
	}
	if !forward && targetedEdge.Offset == f.rng.Start {
		return SelectionResultPrevious
	}
	var result SelectionResult
	var newPosition ui.TextPosition
	switch granularity {
	case TextGranularityWord:
		textBoundary := f.paragraph.textPainter.WordBoundaries().MoveByWordBoundary
		newPosition = f.moveBeyondTextBoundaryAtDirection(targetedEdge, forward, textBoundary)
		result = SelectionResultEnd
	case TextGranularityParagraph:
		text := f.rng.TextInside(f.fullText)
		newPosition = f.moveBeyondTextBoundaryAtDirection(targetedEdge, forward, services.NewParagraphBoundary(text))
		result = SelectionResultEnd
	case TextGranularityLine:
		newPosition = f.moveToTextBoundaryAtDirection(targetedEdge, forward, services.NewLineBoundary(f))
		result = SelectionResultEnd
	case TextGranularityDocument:
		text := f.rng.TextInside(f.fullText)
		newPosition = f.moveBeyondTextBoundaryAtDirection(targetedEdge, forward, services.NewDocumentBoundary(text))
		switch {
		case forward && newPosition.Offset == f.rng.End:
			result = SelectionResultNext
		case !forward && newPosition.Offset == f.rng.Start:
			result = SelectionResultPrevious
		default:
			result = SelectionResultEnd
		}
	default: // character
		text := f.rng.TextInside(f.fullText)
		newPosition = f.moveBeyondTextBoundaryAtDirection(targetedEdge, forward, services.NewCharacterBoundary(text))
		result = SelectionResultEnd
	}
	if isExtent {
		f.textSelectionEnd = foundation.Some(newPosition)
	} else {
		f.textSelectionStart = foundation.Some(newPosition)
	}
	return result
}

// moveBeyondTextBoundaryAtDirection moves beyond the local boundary of the given
// type (unless the start or the end of the range is reached). It is used for
// most granularities except for the line, to ensure the selection movement
// doesn't get stuck at a local fixed point.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._moveBeyondTextBoundaryAtDirection
func (f *selectableFragment) moveBeyondTextBoundaryAtDirection(end ui.TextPosition, forward bool, textBoundary services.TextBoundary) ui.TextPosition {
	var newOffset int
	if forward {
		if v, ok := textBoundary.GetTrailingTextBoundaryAt(end.Offset); ok {
			newOffset = v
		} else {
			newOffset = f.rng.End
		}
	} else {
		if v, ok := textBoundary.GetLeadingTextBoundaryAt(end.Offset - 1); ok {
			newOffset = v
		} else {
			newOffset = f.rng.Start
		}
	}
	return ui.TextPosition{Offset: newOffset}
}

// moveToTextBoundaryAtDirection moves to the local boundary of the given type.
// It is typically used for line boundaries, such that performing "move to line
// start" more than once never moves the selection to the previous line.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._moveToTextBoundaryAtDirection
func (f *selectableFragment) moveToTextBoundaryAtDirection(end ui.TextPosition, forward bool, textBoundary services.TextBoundary) ui.TextPosition {
	var caretOffset int
	if end.Affinity.IsUpstream() {
		if end.Offset < 1 && !forward {
			return ui.TextPosition{Offset: 0}
		}
		characterBoundary := services.NewCharacterBoundary(f.fullText)
		leading, ok := characterBoundary.GetLeadingTextBoundaryAt(f.rng.Start + end.Offset)
		if !ok {
			leading = f.rng.Start
		}
		caretOffset = max(0, leading) - 1
	} else {
		caretOffset = end.Offset
	}
	var offset int
	if forward {
		v, ok := textBoundary.GetTrailingTextBoundaryAt(caretOffset)
		if !ok {
			v = f.rng.End
		}
		offset = v
	} else {
		v, ok := textBoundary.GetLeadingTextBoundaryAt(caretOffset)
		if !ok {
			v = f.rng.Start
		}
		offset = v
	}
	return ui.TextPosition{Offset: offset}
}

// handleVerticalMovement moves a position to the line above or below at the
// horizontal baseline.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._handleVerticalMovement
func (f *selectableFragment) handleVerticalMovement(position ui.TextPosition, horizontalBaselineInParagraphCoordinates float64, below bool) (ui.TextPosition, SelectionResult) {
	lines := f.paragraph.textPainter.ComputeLineMetrics()
	offset := f.paragraph.GetOffsetForCaret(position, ui.Rect{})
	currentLine := len(lines) - 1
	for _, lineMetrics := range lines {
		if lineMetrics.Baseline > offset.Dy {
			currentLine = lineMetrics.LineNumber
			break
		}
	}
	var newPosition ui.TextPosition
	switch {
	case below && currentLine == len(lines)-1:
		newPosition = ui.TextPosition{Offset: f.rng.End, Affinity: ui.TextAffinityUpstream}
	case !below && currentLine == 0:
		newPosition = ui.TextPosition{Offset: f.rng.Start}
	default:
		newLine := currentLine - 1
		if below {
			newLine = currentLine + 1
		}
		newPosition = f.clampTextPosition(f.paragraph.GetPositionForOffset(ui.Offset{Dx: horizontalBaselineInParagraphCoordinates, Dy: lines[newLine].Baseline}))
	}
	var result SelectionResult
	switch newPosition.Offset {
	case f.rng.Start:
		result = SelectionResultPrevious
	case f.rng.End:
		result = SelectionResultNext
	default:
		result = SelectionResultEnd
	}
	return newPosition, result
}

func (f *selectableFragment) positionIsWithinCurrentSelection(position ui.TextPosition) bool {
	if f.textSelectionStart.IsNone() || f.textSelectionEnd.IsNone() {
		return false
	}
	// Normalize current selection.
	currentStart, currentEnd := f.textSelectionEnd.Value(), f.textSelectionStart.Value()
	if compareTextPositions(f.textSelectionStart.Value(), f.textSelectionEnd.Value()) > 0 {
		currentStart, currentEnd = f.textSelectionStart.Value(), f.textSelectionEnd.Value()
	}
	return compareTextPositions(currentStart, position) >= 0 && compareTextPositions(currentEnd, position) <= 0
}

// compareTextPositions is 1 when the first position is before the other, -1
// when it is after, and 0 when they are the same.
//
// Flutter: rendering/paragraph.dart _SelectableFragment._compareTextPositions
func compareTextPositions(position, otherPosition ui.TextPosition) int {
	switch {
	case position.Offset < otherPosition.Offset:
		return 1
	case position.Offset > otherPosition.Offset:
		return -1
	case position.Affinity.IsUpstream() == otherPosition.Affinity.IsUpstream():
		return 0
	case position.Affinity.IsUpstream():
		return 1
	default:
		return -1
	}
}

// GetTransformTo is the transform of the paragraph to the ancestor.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.getTransformTo
func (f *selectableFragment) GetTransformTo(ancestor RenderObject) *ui.Matrix4 {
	return f.paragraph.GetTransformTo(ancestor)
}

// PushHandleLayers sets the layer links that the handles of the selection
// follow.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.pushHandleLayers
func (f *selectableFragment) PushHandleLayers(startHandle, endHandle *LayerLink) {
	if !f.paragraph.Attached() {
		if foundation.DebugMode && (startHandle != nil || endHandle != nil) {
			panic(errors.New(tagSelectionHandlesOnDetached, "only clean up can be called on a paragraph that is not attached"))
		}
		return
	}
	if f.startHandleLayerLink != startHandle {
		f.startHandleLayerLink = startHandle
		f.paragraph.MarkNeedsPaint()
	}
	if f.endHandleLayerLink != endHandle {
		f.endHandleLayerLink = endHandle
		f.paragraph.MarkNeedsPaint()
	}
}

// BoundingBoxes are the boxes of the text of the fragment.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.boundingBoxes
func (f *selectableFragment) BoundingBoxes() []ui.Rect {
	if !f.hasBoundingBoxes {
		boxes := f.paragraph.GetBoxesForSelection(f.rng, ui.BoxesForRangeOpts{HeightStyle: ui.BoxHeightStyleMax})
		if len(boxes) > 0 {
			f.cachedBoundingBoxes = nil
			for _, b := range boxes {
				f.cachedBoundingBoxes = append(f.cachedBoundingBoxes, b.ToRect())
			}
		} else {
			offset := f.paragraph.getOffsetForPosition(ui.TextPosition{Offset: f.rng.Start})
			f.cachedBoundingBoxes = []ui.Rect{ui.NewRectFromPoints(offset, offset.Translate(0, -f.paragraph.textPainter.PreferredLineHeight()))}
		}
		f.hasBoundingBoxes = true
	}
	return f.cachedBoundingBoxes
}

func (f *selectableFragment) rect() ui.Rect {
	if !f.hasRect {
		boxes := f.paragraph.GetBoxesForSelection(f.rng, ui.BoxesForRangeOpts{HeightStyle: ui.BoxHeightStyleMax})
		if len(boxes) > 0 {
			result := boxes[0].ToRect()
			for _, b := range boxes[1:] {
				result = result.ExpandToInclude(b.ToRect())
			}
			f.cachedRect = result
		} else {
			offset := f.paragraph.getOffsetForPosition(ui.TextPosition{Offset: f.rng.Start})
			f.cachedRect = ui.NewRectFromPoints(offset, offset.Translate(0, -f.paragraph.textPainter.PreferredLineHeight()))
		}
		f.hasRect = true
	}
	return f.cachedRect
}

// didChangeParagraphLayout forgets the cached geometry.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.didChangeParagraphLayout
func (f *selectableFragment) didChangeParagraphLayout() {
	f.hasRect = false
	f.hasBoundingBoxes = false
}

// ContentLength is the length of the text of the fragment.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.contentLength
func (f *selectableFragment) ContentLength() int { return f.rng.End - f.rng.Start }

// Size is the size of the text of the fragment.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.size
func (f *selectableFragment) Size() ui.Size { return f.rect().Size() }

// Dispose releases the fragment.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.dispose
func (f *selectableFragment) Dispose() { f.ChangeNotifier.Dispose() }

// paintSelection paints the selected part of the text with the selection color.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.paintSelection
func (f *selectableFragment) paintSelection(context *PaintingContext, offset ui.Offset) {
	if f.textSelectionStart.IsNone() || f.textSelectionEnd.IsNone() {
		return
	}
	if color, ok := f.paragraph.SelectionColor().Get(); ok {
		selection := ui.TextRange{Start: f.textSelectionStart.Value().Offset, End: f.textSelectionEnd.Value().Offset}
		selectionPaint := ui.NewPaint()
		selectionPaint.SetStyle(ui.PaintingStyleFill)
		selectionPaint.SetColor(color)
		for _, textBox := range f.paragraph.GetBoxesForSelection(selection) {
			context.Canvas().DrawRect(textBox.ToRect().Shift(offset), selectionPaint)
		}
	}
}

// paintHandles puts the leader layers of the handles at the ends of the
// selection.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.paintHandles
func (f *selectableFragment) paintHandles(context *PaintingContext, offset ui.Offset) {
	if f.textSelectionStart.IsNone() || f.textSelectionEnd.IsNone() {
		return
	}
	value := f.Value()
	if f.startHandleLayerLink != nil && value.StartSelectionPoint != nil {
		context.PushLayer(NewLeaderLayer(f.startHandleLayerLink, LeaderLayerOpts{Offset: offset.Add(value.StartSelectionPoint.LocalPosition)}),
			func(*PaintingContext, ui.Offset) {}, ui.OffsetZero, foundation.None[ui.Rect]())
	}
	if f.endHandleLayerLink != nil && value.EndSelectionPoint != nil {
		context.PushLayer(NewLeaderLayer(f.endHandleLayerLink, LeaderLayerOpts{Offset: offset.Add(value.EndSelectionPoint.LocalPosition)}),
			func(*PaintingContext, ui.Offset) {}, ui.OffsetZero, foundation.None[ui.Rect]())
	}
}

// GetLineAtOffset is the line of the position, clamped to the range of the
// fragment.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.getLineAtOffset
func (f *selectableFragment) GetLineAtOffset(position ui.TextPosition) services.TextSelection {
	line := f.paragraph.getLineAtOffset(position)
	start := min(max(line.Start, f.rng.Start), f.rng.End)
	end := min(max(line.End, f.rng.Start), f.rng.End)
	return services.NewTextSelection(start, end)
}

// GetTextPositionAbove is the position on the line above.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.getTextPositionAbove
func (f *selectableFragment) GetTextPositionAbove(position ui.TextPosition) ui.TextPosition {
	return f.clampTextPosition(f.paragraph.getTextPositionAbove(position))
}

// GetTextPositionBelow is the position on the line below.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.getTextPositionBelow
func (f *selectableFragment) GetTextPositionBelow(position ui.TextPosition) ui.TextPosition {
	return f.clampTextPosition(f.paragraph.getTextPositionBelow(position))
}

// GetWordBoundary is the word of the paragraph at the position.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.getWordBoundary
func (f *selectableFragment) GetWordBoundary(position ui.TextPosition) ui.TextRange {
	return f.paragraph.GetWordBoundary(position)
}

// DebugFillProperties adds the text and the range.
//
// Flutter: rendering/paragraph.dart _SelectableFragment.debugFillProperties
func (f *selectableFragment) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewStringProperty("textInsideRange", foundation.Some(f.rng.TextInside(f.fullText))))
	p.Add(foundation.NewDiagnosticsProperty("range", f.rng))
	p.Add(foundation.NewStringProperty("fullText", foundation.Some(f.fullText)))
}

// getOffsetForPosition is the offset of the bottom of the caret at the position,
// in the coordinates of the paragraph.
//
// Flutter: rendering/paragraph.dart RenderParagraph._getOffsetForPosition
func (r *RenderParagraph) getOffsetForPosition(position ui.TextPosition) ui.Offset {
	return r.GetOffsetForCaret(position, ui.Rect{}).Add(ui.Offset{Dy: r.GetFullHeightForCaret(position)})
}

func (r *RenderParagraph) getLineAtOffset(position ui.TextPosition) ui.TextRange {
	return r.textPainter.GetLineBoundary(position)
}

func (r *RenderParagraph) getTextPositionAbove(position ui.TextPosition) ui.TextPosition {
	// -0.5 of preferredLineHeight points to the middle of the line above.
	return r.getTextPositionVertical(position, -0.5*r.textPainter.PreferredLineHeight())
}

func (r *RenderParagraph) getTextPositionBelow(position ui.TextPosition) ui.TextPosition {
	// 1.5 of preferredLineHeight points to the middle of the line below.
	return r.getTextPositionVertical(position, 1.5*r.textPainter.PreferredLineHeight())
}

func (r *RenderParagraph) getTextPositionVertical(position ui.TextPosition, verticalOffset float64) ui.TextPosition {
	caretOffset := r.textPainter.GetOffsetForCaret(position, ui.Rect{})
	return r.textPainter.GetPositionForOffset(caretOffset.Translate(0, verticalOffset))
}

// Selections are the ongoing selections in this paragraph, one for each
// fragment that has one. They do not include selections in placeholders.
//
// Flutter: rendering/paragraph.dart RenderParagraph.selections
func (r *RenderParagraph) Selections() []services.TextSelection {
	var results []services.TextSelection
	for _, f := range r.lastSelectableFragments {
		if f.textSelectionStart.IsSome() && f.textSelectionEnd.IsSome() {
			results = append(results, services.NewTextSelection(f.textSelectionStart.Value().Offset, f.textSelectionEnd.Value().Offset))
		}
	}
	return results
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSelectionBaselineNaN            = "SelectionBaselineNaN"
	tagSelectionGranularityUnsupported = "SelectionGranularityUnsupported"
	tagSelectionHandlesOnDetached      = "SelectionHandlesOnDetached"
	tagSelectionNotStarted             = "SelectionNotStarted"
)
