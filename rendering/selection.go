// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// SelectionResult is the return value of Selectable.DispatchSelectionEvent: how
// the selection event was handled, and where the next one should go.
//
// Flutter: rendering/selection.dart SelectionResult
type SelectionResult int

// The values of SelectionResult follow Flutter's declaration order.
const (
	// SelectionResultUnspecified is the zero value: no result.
	//
	// Flutter: no counterpart
	SelectionResultUnspecified SelectionResult = iota
	// SelectionResultNext means the selection edge is after the selectable, so
	// the next one must handle the event.
	//
	// Flutter: rendering/selection.dart SelectionResult.next
	SelectionResultNext
	// SelectionResultPrevious means the selection edge is before the selectable.
	//
	// Flutter: rendering/selection.dart SelectionResult.previous
	SelectionResultPrevious
	// SelectionResultEnd means the selection edge is in the selectable.
	//
	// Flutter: rendering/selection.dart SelectionResult.end
	SelectionResultEnd
	// SelectionResultPending means the selectable did not handle the event yet.
	//
	// Flutter: rendering/selection.dart SelectionResult.pending
	SelectionResultPending
	// SelectionResultNone means the selectable does not take part in the
	// selection (for example, it has no content).
	//
	// Flutter: rendering/selection.dart SelectionResult.none
	SelectionResultNone
)

// Name returns the Dart enum name, such as "next".
//
// Flutter: rendering/selection.dart SelectionResult.name
func (r SelectionResult) Name() string {
	return enumName(int(r), "next", "previous", "end", "pending", "none")
}

// String returns the name Dart prints, such as "SelectionResult.next".
//
// Flutter: rendering/selection.dart SelectionResult.toString
func (r SelectionResult) String() string { return "SelectionResult." + r.Name() }

// SelectionHandler handles selection events and holds the geometry of the
// selection.
//
// Flutter: rendering/selection.dart SelectionHandler
type SelectionHandler interface {
	foundation.ValueListenable[SelectionGeometry]
	// PushHandleLayers gives the layer links that the selection handles follow.
	//
	// Flutter: rendering/selection.dart SelectionHandler.pushHandleLayers
	PushHandleLayers(startHandle, endHandle *LayerLink)
	// GetSelectedContent is the content that is selected, or nil.
	//
	// Flutter: rendering/selection.dart SelectionHandler.getSelectedContent
	GetSelectedContent() *SelectedContent
	// GetSelection is the range of the selection in the content, or nil.
	//
	// Flutter: rendering/selection.dart SelectionHandler.getSelection
	GetSelection() *SelectedContentRange
	// DispatchSelectionEvent handles the event.
	//
	// Flutter: rendering/selection.dart SelectionHandler.dispatchSelectionEvent
	DispatchSelectionEvent(event SelectionEvent) SelectionResult
	// ContentLength is the number of selectable positions in the content.
	//
	// Flutter: rendering/selection.dart SelectionHandler.contentLength
	ContentLength() int
}

// SelectedContentRange is the range of a selection in the content of a
// selectable.
//
// Flutter: rendering/selection.dart SelectedContentRange
type SelectedContentRange struct {
	// StartOffset is where the selection starts.
	//
	// Flutter: rendering/selection.dart SelectedContentRange.startOffset
	StartOffset int
	// EndOffset is where the selection ends.
	//
	// Flutter: rendering/selection.dart SelectedContentRange.endOffset
	EndOffset int
}

// DebugFillProperties adds the offsets.
//
// Flutter: rendering/selection.dart SelectedContentRange.debugFillProperties
func (r SelectedContentRange) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewIntProperty("startOffset", foundation.Some(r.StartOffset)))
	p.Add(foundation.NewIntProperty("endOffset", foundation.Some(r.EndOffset)))
}

// SelectedContent is the content that a selection holds.
//
// Flutter: rendering/selection.dart SelectedContent
type SelectedContent struct {
	// PlainText is the selected text.
	//
	// Flutter: rendering/selection.dart SelectedContent.plainText
	PlainText string
}

// DebugFillProperties adds the text.
//
// Flutter: rendering/selection.dart SelectedContent.debugFillProperties
func (c SelectedContent) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewStringProperty("plainText", foundation.Some(c.PlainText)))
}

// Selectable is a handler that is a part of a selection: it has a size and a
// transform in the render tree, and bounding boxes.
//
// Flutter: rendering/selection.dart Selectable
type Selectable interface {
	SelectionHandler
	// GetTransformTo is the transform from the selectable to the ancestor.
	//
	// Flutter: rendering/selection.dart Selectable.getTransformTo
	GetTransformTo(ancestor RenderObject) *ui.Matrix4
	// Size is the size of the selectable.
	//
	// Flutter: rendering/selection.dart Selectable.size
	Size() ui.Size
	// BoundingBoxes are the rectangles that enclose the selectable's content.
	//
	// Flutter: rendering/selection.dart Selectable.boundingBoxes
	BoundingBoxes() []ui.Rect
	// Dispose releases the resources of the selectable.
	//
	// Flutter: rendering/selection.dart Selectable.dispose
	Dispose()
}

// SelectionRegistrar is an object that selectables register with.
//
// Flutter: rendering/selection.dart SelectionRegistrar
type SelectionRegistrar interface {
	// Add adds the selectable.
	//
	// Flutter: rendering/selection.dart SelectionRegistrar.add
	Add(selectable Selectable)
	// Remove removes the selectable.
	//
	// Flutter: rendering/selection.dart SelectionRegistrar.remove
	Remove(selectable Selectable)
}

// SelectionRegistrant keeps a selectable registered with a registrar while it
// has content. Embed it in a selectable and call InitSelectionRegistrant.
//
// Flutter: rendering/selection.dart SelectionRegistrant
type SelectionRegistrant struct {
	self       Selectable
	registrar  SelectionRegistrar
	subscribed bool
	handle     foundation.ListenerHandle
}

// InitSelectionRegistrant sets the selectable that the registrant registers.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitSelectionRegistrant(r *SelectionRegistrant, self Selectable) { r.self = self }

// Registrar is the registrar that the selectable is registered with, or nil.
//
// Flutter: rendering/selection.dart SelectionRegistrant.registrar
func (r *SelectionRegistrant) Registrar() SelectionRegistrar { return r.registrar }

// SetRegistrar changes the registrar; the selectable is registered with it
// whenever it has content.
//
// Flutter: rendering/selection.dart SelectionRegistrant.registrar
func (r *SelectionRegistrant) SetRegistrar(value SelectionRegistrar) {
	if value == r.registrar {
		return
	}
	if value == nil {
		r.self.RemoveListener(r.handle)
		r.handle = foundation.ListenerHandle{}
	} else if r.registrar == nil {
		r.handle = r.self.AddListener(r.updateSubscription)
	}
	r.removeSubscription()
	r.registrar = value
	r.updateSubscription()
}

// Dispose removes the selectable from the registrar.
//
// Flutter: rendering/selection.dart SelectionRegistrant.dispose
func (r *SelectionRegistrant) Dispose() { r.removeSubscription() }

func (r *SelectionRegistrant) updateSubscription() {
	if r.registrar == nil {
		r.subscribed = false
		return
	}
	hasContent := r.self.Value().HasContent
	if r.subscribed && !hasContent {
		r.registrar.Remove(r.self)
		r.subscribed = false
	} else if !r.subscribed && hasContent {
		r.registrar.Add(r.self)
		r.subscribed = true
	}
}

func (r *SelectionRegistrant) removeSubscription() {
	if r.subscribed {
		r.registrar.Remove(r.self)
		r.subscribed = false
	}
}

// SelectionUtilsGetResultBasedOnRect is how an event at the point is handled by
// a selectable whose content fills the rectangle: the end if it is in the
// rectangle, else the next or the previous one.
//
// Flutter: rendering/selection.dart SelectionUtils.getResultBasedOnRect
func SelectionUtilsGetResultBasedOnRect(targetRect ui.Rect, point ui.Offset) SelectionResult {
	switch {
	case targetRect.Contains(point):
		return SelectionResultEnd
	case point.Dy < targetRect.Top:
		return SelectionResultPrevious
	case point.Dy > targetRect.Bottom:
		return SelectionResultNext
	case point.Dx >= targetRect.Right:
		return SelectionResultNext
	}
	return SelectionResultPrevious
}

// SelectionUtilsAdjustDragOffset moves a point that is outside the rectangle to
// the corner of the rectangle that is closest in reading order, in the text
// direction.
//
// Flutter: rendering/selection.dart SelectionUtils.adjustDragOffset
func SelectionUtilsAdjustDragOffset(targetRect ui.Rect, point ui.Offset, direction ui.TextDirection) ui.Offset {
	if targetRect.Contains(point) {
		return point
	}
	rtl := direction == ui.TextDirectionRTL
	if point.Dy <= targetRect.Top || point.Dy <= targetRect.Bottom && point.Dx <= targetRect.Left {
		if rtl {
			return targetRect.TopRight()
		}
		return targetRect.TopLeft()
	}
	if rtl {
		return targetRect.BottomLeft()
	}
	return targetRect.BottomRight()
}

// SelectionEventType is the type of a selection event.
//
// Flutter: rendering/selection.dart SelectionEventType
type SelectionEventType int

// The values of SelectionEventType follow Flutter's declaration order.
const (
	// SelectionEventTypeUnspecified is the zero value: no type.
	//
	// Flutter: no counterpart
	SelectionEventTypeUnspecified SelectionEventType = iota
	// SelectionEventTypeStartEdgeUpdate moves the start edge.
	//
	// Flutter: rendering/selection.dart SelectionEventType.startEdgeUpdate
	SelectionEventTypeStartEdgeUpdate
	// SelectionEventTypeEndEdgeUpdate moves the end edge.
	//
	// Flutter: rendering/selection.dart SelectionEventType.endEdgeUpdate
	SelectionEventTypeEndEdgeUpdate
	// SelectionEventTypeClear clears the selection.
	//
	// Flutter: rendering/selection.dart SelectionEventType.clear
	SelectionEventTypeClear
	// SelectionEventTypeSelectAll selects all the content.
	//
	// Flutter: rendering/selection.dart SelectionEventType.selectAll
	SelectionEventTypeSelectAll
	// SelectionEventTypeSelectWord selects a word.
	//
	// Flutter: rendering/selection.dart SelectionEventType.selectWord
	SelectionEventTypeSelectWord
	// SelectionEventTypeSelectParagraph selects a paragraph.
	//
	// Flutter: rendering/selection.dart SelectionEventType.selectParagraph
	SelectionEventTypeSelectParagraph
	// SelectionEventTypeGranularlyExtendSelection extends the selection by a
	// granularity.
	//
	// Flutter: rendering/selection.dart SelectionEventType.granularlyExtendSelection
	SelectionEventTypeGranularlyExtendSelection
	// SelectionEventTypeDirectionallyExtendSelection extends the selection in a
	// direction.
	//
	// Flutter: rendering/selection.dart SelectionEventType.directionallyExtendSelection
	SelectionEventTypeDirectionallyExtendSelection
)

// Name returns the Dart enum name, such as "clear".
//
// Flutter: rendering/selection.dart SelectionEventType.name
func (t SelectionEventType) Name() string {
	return enumName(int(t), "startEdgeUpdate", "endEdgeUpdate", "clear", "selectAll", "selectWord", "selectParagraph",
		"granularlyExtendSelection", "directionallyExtendSelection")
}

// String returns the name Dart prints, such as "SelectionEventType.clear".
//
// Flutter: rendering/selection.dart SelectionEventType.toString
func (t SelectionEventType) String() string { return "SelectionEventType." + t.Name() }

// TextGranularity is the granularity by which a selection is extended.
//
// Flutter: rendering/selection.dart TextGranularity
type TextGranularity int

// The values of TextGranularity follow Flutter's declaration order.
const (
	// TextGranularityUnspecified is the zero value: the default (character).
	//
	// Flutter: no counterpart
	TextGranularityUnspecified TextGranularity = iota
	// TextGranularityCharacter extends by a character.
	//
	// Flutter: rendering/selection.dart TextGranularity.character
	TextGranularityCharacter
	// TextGranularityWord extends by a word.
	//
	// Flutter: rendering/selection.dart TextGranularity.word
	TextGranularityWord
	// TextGranularityParagraph extends by a paragraph.
	//
	// Flutter: rendering/selection.dart TextGranularity.paragraph
	TextGranularityParagraph
	// TextGranularityLine extends by a line.
	//
	// Flutter: rendering/selection.dart TextGranularity.line
	TextGranularityLine
	// TextGranularityDocument extends by the whole document.
	//
	// Flutter: rendering/selection.dart TextGranularity.document
	TextGranularityDocument
)

// Name returns the Dart enum name, such as "word".
//
// Flutter: rendering/selection.dart TextGranularity.name
func (g TextGranularity) Name() string {
	return enumName(int(g), "character", "word", "paragraph", "line", "document")
}

// String returns the name Dart prints, such as "TextGranularity.word".
//
// Flutter: rendering/selection.dart TextGranularity.toString
func (g TextGranularity) String() string { return "TextGranularity." + g.Name() }

func (g TextGranularity) orCharacter() TextGranularity {
	if g == TextGranularityUnspecified {
		return TextGranularityCharacter
	}
	return g
}

// SelectionEvent is an event that the selection system sends to selectables.
//
// Flutter: rendering/selection.dart SelectionEvent
type SelectionEvent interface {
	// Type is the type of the event.
	//
	// Flutter: rendering/selection.dart SelectionEvent.type
	Type() SelectionEventType
}

// SelectAllSelectionEvent selects all the content.
//
// Flutter: rendering/selection.dart SelectAllSelectionEvent
type SelectAllSelectionEvent struct{}

// Type is SelectionEventTypeSelectAll.
//
// Flutter: rendering/selection.dart SelectAllSelectionEvent
func (SelectAllSelectionEvent) Type() SelectionEventType { return SelectionEventTypeSelectAll }

// ClearSelectionEvent clears the selection.
//
// Flutter: rendering/selection.dart ClearSelectionEvent
type ClearSelectionEvent struct{}

// Type is SelectionEventTypeClear.
//
// Flutter: rendering/selection.dart ClearSelectionEvent
func (ClearSelectionEvent) Type() SelectionEventType { return SelectionEventTypeClear }

// SelectWordSelectionEvent selects the word at a global position.
//
// Flutter: rendering/selection.dart SelectWordSelectionEvent
type SelectWordSelectionEvent struct {
	// GlobalPosition is where the word is, in global coordinates.
	//
	// Flutter: rendering/selection.dart SelectWordSelectionEvent.globalPosition
	GlobalPosition ui.Offset
}

// Type is SelectionEventTypeSelectWord.
//
// Flutter: rendering/selection.dart SelectWordSelectionEvent
func (SelectWordSelectionEvent) Type() SelectionEventType { return SelectionEventTypeSelectWord }

// SelectParagraphSelectionEvent selects the paragraph at a global position.
//
// Flutter: rendering/selection.dart SelectParagraphSelectionEvent
type SelectParagraphSelectionEvent struct {
	// GlobalPosition is where the paragraph is, in global coordinates.
	//
	// Flutter: rendering/selection.dart SelectParagraphSelectionEvent.globalPosition
	GlobalPosition ui.Offset
	// Absorb tells that the handler that is hit absorbs the selection.
	//
	// Flutter: rendering/selection.dart SelectParagraphSelectionEvent.absorb
	Absorb bool
}

// Type is SelectionEventTypeSelectParagraph.
//
// Flutter: rendering/selection.dart SelectParagraphSelectionEvent
func (SelectParagraphSelectionEvent) Type() SelectionEventType {
	return SelectionEventTypeSelectParagraph
}

// SelectionEdgeUpdateEvent moves the start or the end edge of the selection to
// a global position.
//
// Flutter: rendering/selection.dart SelectionEdgeUpdateEvent
type SelectionEdgeUpdateEvent struct {
	// GlobalPosition is where the edge moves to, in global coordinates.
	//
	// Flutter: rendering/selection.dart SelectionEdgeUpdateEvent.globalPosition
	GlobalPosition ui.Offset
	// Granularity is how much the edge snaps to the text.
	//
	// Flutter: rendering/selection.dart SelectionEdgeUpdateEvent.granularity
	Granularity TextGranularity
	isEnd       bool
}

// NewSelectionEdgeUpdateEventForStart moves the start edge. An unspecified
// granularity is character.
//
// Flutter: rendering/selection.dart SelectionEdgeUpdateEvent.forStart
func NewSelectionEdgeUpdateEventForStart(globalPosition ui.Offset, granularity TextGranularity) SelectionEdgeUpdateEvent {
	return SelectionEdgeUpdateEvent{GlobalPosition: globalPosition, Granularity: granularity.orCharacter()}
}

// NewSelectionEdgeUpdateEventForEnd moves the end edge. An unspecified
// granularity is character.
//
// Flutter: rendering/selection.dart SelectionEdgeUpdateEvent.forEnd
func NewSelectionEdgeUpdateEventForEnd(globalPosition ui.Offset, granularity TextGranularity) SelectionEdgeUpdateEvent {
	return SelectionEdgeUpdateEvent{GlobalPosition: globalPosition, Granularity: granularity.orCharacter(), isEnd: true}
}

// Type is the start or the end edge update.
//
// Flutter: rendering/selection.dart SelectionEdgeUpdateEvent
func (e SelectionEdgeUpdateEvent) Type() SelectionEventType {
	if e.isEnd {
		return SelectionEventTypeEndEdgeUpdate
	}
	return SelectionEventTypeStartEdgeUpdate
}

// GranularlyExtendSelectionEvent extends the selection by a granularity.
//
// Flutter: rendering/selection.dart GranularlyExtendSelectionEvent
type GranularlyExtendSelectionEvent struct {
	// Forward is whether the edge moves forward.
	//
	// Flutter: rendering/selection.dart GranularlyExtendSelectionEvent.forward
	Forward bool
	// IsEnd is whether the end edge moves, instead of the start one.
	//
	// Flutter: rendering/selection.dart GranularlyExtendSelectionEvent.isEnd
	IsEnd bool
	// Granularity is how far the edge moves.
	//
	// Flutter: rendering/selection.dart GranularlyExtendSelectionEvent.granularity
	Granularity TextGranularity
}

// Type is SelectionEventTypeGranularlyExtendSelection.
//
// Flutter: rendering/selection.dart GranularlyExtendSelectionEvent
func (GranularlyExtendSelectionEvent) Type() SelectionEventType {
	return SelectionEventTypeGranularlyExtendSelection
}

// SelectionExtendDirection is the direction in which a selection is extended.
//
// Flutter: rendering/selection.dart SelectionExtendDirection
type SelectionExtendDirection int

// The values of SelectionExtendDirection follow Flutter's declaration order.
const (
	// SelectionExtendDirectionUnspecified is the zero value: no direction.
	//
	// Flutter: no counterpart
	SelectionExtendDirectionUnspecified SelectionExtendDirection = iota
	// SelectionExtendDirectionPreviousLine extends to the previous line.
	//
	// Flutter: rendering/selection.dart SelectionExtendDirection.previousLine
	SelectionExtendDirectionPreviousLine
	// SelectionExtendDirectionNextLine extends to the next line.
	//
	// Flutter: rendering/selection.dart SelectionExtendDirection.nextLine
	SelectionExtendDirectionNextLine
	// SelectionExtendDirectionForward extends forward.
	//
	// Flutter: rendering/selection.dart SelectionExtendDirection.forward
	SelectionExtendDirectionForward
	// SelectionExtendDirectionBackward extends backward.
	//
	// Flutter: rendering/selection.dart SelectionExtendDirection.backward
	SelectionExtendDirectionBackward
)

// Name returns the Dart enum name, such as "nextLine".
//
// Flutter: rendering/selection.dart SelectionExtendDirection.name
func (d SelectionExtendDirection) Name() string {
	return enumName(int(d), "previousLine", "nextLine", "forward", "backward")
}

// String returns the name Dart prints, such as "SelectionExtendDirection.forward".
//
// Flutter: rendering/selection.dart SelectionExtendDirection.toString
func (d SelectionExtendDirection) String() string { return "SelectionExtendDirection." + d.Name() }

// DirectionallyExtendSelectionEvent extends the selection in a direction.
//
// Flutter: rendering/selection.dart DirectionallyExtendSelectionEvent
type DirectionallyExtendSelectionEvent struct {
	// Dx is the horizontal position that the edge keeps when it moves lines.
	//
	// Flutter: rendering/selection.dart DirectionallyExtendSelectionEvent.dx
	Dx float64
	// IsEnd is whether the end edge moves, instead of the start one.
	//
	// Flutter: rendering/selection.dart DirectionallyExtendSelectionEvent.isEnd
	IsEnd bool
	// Direction is where the edge moves.
	//
	// Flutter: rendering/selection.dart DirectionallyExtendSelectionEvent.direction
	Direction SelectionExtendDirection
}

// Type is SelectionEventTypeDirectionallyExtendSelection.
//
// Flutter: rendering/selection.dart DirectionallyExtendSelectionEvent
func (DirectionallyExtendSelectionEvent) Type() SelectionEventType {
	return SelectionEventTypeDirectionallyExtendSelection
}

// DirectionallyExtendSelectionEventCopy holds the replacements of CopyWith.
//
// Flutter: rendering/selection.dart DirectionallyExtendSelectionEvent.copyWith
type DirectionallyExtendSelectionEventCopy struct {
	Dx        foundation.Opt[float64]
	IsEnd     foundation.Opt[bool]
	Direction SelectionExtendDirection
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: rendering/selection.dart DirectionallyExtendSelectionEvent.copyWith
func (e DirectionallyExtendSelectionEvent) CopyWith(c DirectionallyExtendSelectionEventCopy) DirectionallyExtendSelectionEvent {
	out := e
	if v, ok := c.Dx.Get(); ok {
		out.Dx = v
	}
	if v, ok := c.IsEnd.Get(); ok {
		out.IsEnd = v
	}
	if c.Direction != SelectionExtendDirectionUnspecified {
		out.Direction = c.Direction
	}
	return out
}

// SelectionStatus is how much is selected.
//
// Flutter: rendering/selection.dart SelectionStatus
type SelectionStatus int

// The values of SelectionStatus follow Flutter's declaration order.
const (
	// SelectionStatusUnspecified is the zero value: no status.
	//
	// Flutter: no counterpart
	SelectionStatusUnspecified SelectionStatus = iota
	// SelectionStatusUncollapsed is a selection with a range.
	//
	// Flutter: rendering/selection.dart SelectionStatus.uncollapsed
	SelectionStatusUncollapsed
	// SelectionStatusCollapsed is a selection with a single position.
	//
	// Flutter: rendering/selection.dart SelectionStatus.collapsed
	SelectionStatusCollapsed
	// SelectionStatusNone is no selection.
	//
	// Flutter: rendering/selection.dart SelectionStatus.none
	SelectionStatusNone
)

// Name returns the Dart enum name, such as "collapsed".
//
// Flutter: rendering/selection.dart SelectionStatus.name
func (s SelectionStatus) Name() string { return enumName(int(s), "uncollapsed", "collapsed", "none") }

// String returns the name Dart prints, such as "SelectionStatus.none".
//
// Flutter: rendering/selection.dart SelectionStatus.toString
func (s SelectionStatus) String() string { return "SelectionStatus." + s.Name() }

// TextSelectionHandleType is the type of a selection handle.
//
// Flutter: rendering/selection.dart TextSelectionHandleType
type TextSelectionHandleType int

// The values of TextSelectionHandleType follow Flutter's declaration order.
const (
	// TextSelectionHandleTypeUnspecified is the zero value: no type.
	//
	// Flutter: no counterpart
	TextSelectionHandleTypeUnspecified TextSelectionHandleType = iota
	// TextSelectionHandleTypeLeft is a handle of the left edge of a selection.
	//
	// Flutter: rendering/selection.dart TextSelectionHandleType.left
	TextSelectionHandleTypeLeft
	// TextSelectionHandleTypeRight is a handle of the right edge of a selection.
	//
	// Flutter: rendering/selection.dart TextSelectionHandleType.right
	TextSelectionHandleTypeRight
	// TextSelectionHandleTypeCollapsed is a handle of a collapsed selection.
	//
	// Flutter: rendering/selection.dart TextSelectionHandleType.collapsed
	TextSelectionHandleTypeCollapsed
)

// Name returns the Dart enum name, such as "left".
//
// Flutter: rendering/selection.dart TextSelectionHandleType.name
func (t TextSelectionHandleType) Name() string { return enumName(int(t), "left", "right", "collapsed") }

// String returns the name Dart prints, such as "TextSelectionHandleType.left".
//
// Flutter: rendering/selection.dart TextSelectionHandleType.toString
func (t TextSelectionHandleType) String() string { return "TextSelectionHandleType." + t.Name() }

// SelectionPoint is a point of a selection, where a handle is drawn.
//
// Flutter: rendering/selection.dart SelectionPoint
type SelectionPoint struct {
	// LocalPosition is the position, in the selectable's coordinates.
	//
	// Flutter: rendering/selection.dart SelectionPoint.localPosition
	LocalPosition ui.Offset
	// LineHeight is the height of the line at the point.
	//
	// Flutter: rendering/selection.dart SelectionPoint.lineHeight
	LineHeight float64
	// HandleType is the type of the handle at the point.
	//
	// Flutter: rendering/selection.dart SelectionPoint.handleType
	HandleType TextSelectionHandleType
}

// DebugFillProperties adds the position, the line height and the handle type.
//
// Flutter: rendering/selection.dart SelectionPoint.debugFillProperties
func (p SelectionPoint) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	b.Add(foundation.NewDiagnosticsProperty("localPosition", p.LocalPosition))
	b.Add(foundation.NewDoubleProperty("lineHeight", foundation.Some(p.LineHeight)))
	b.Add(foundation.NewEnumProperty("handleType", foundation.Some(p.HandleType)))
}

// SelectionGeometry is the geometry of a selection: its edges, its rectangles,
// and whether the selectable has content to select.
//
// Flutter: rendering/selection.dart SelectionGeometry
type SelectionGeometry struct {
	// StartSelectionPoint is the start edge of the selection, or nil.
	//
	// Flutter: rendering/selection.dart SelectionGeometry.startSelectionPoint
	StartSelectionPoint *SelectionPoint
	// EndSelectionPoint is the end edge of the selection, or nil.
	//
	// Flutter: rendering/selection.dart SelectionGeometry.endSelectionPoint
	EndSelectionPoint *SelectionPoint
	// Status is how much is selected.
	//
	// Flutter: rendering/selection.dart SelectionGeometry.status
	Status SelectionStatus
	// SelectionRects are the rectangles of the selection.
	//
	// Flutter: rendering/selection.dart SelectionGeometry.selectionRects
	SelectionRects []ui.Rect
	// HasContent is whether there is content that can be selected.
	//
	// Flutter: rendering/selection.dart SelectionGeometry.hasContent
	HasContent bool
}

// HasSelection is whether anything is selected.
//
// Flutter: rendering/selection.dart SelectionGeometry.hasSelection
func (g SelectionGeometry) HasSelection() bool { return g.Status != SelectionStatusNone }

// Equal reports whether the geometries are equal.
//
// Flutter: rendering/selection.dart SelectionGeometry.==
func (g SelectionGeometry) Equal(o SelectionGeometry) bool {
	eqPoint := func(a, b *SelectionPoint) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
	if !eqPoint(g.StartSelectionPoint, o.StartSelectionPoint) || !eqPoint(g.EndSelectionPoint, o.EndSelectionPoint) ||
		g.Status != o.Status || g.HasContent != o.HasContent || len(g.SelectionRects) != len(o.SelectionRects) {
		return false
	}
	for i := range g.SelectionRects {
		if g.SelectionRects[i] != o.SelectionRects[i] {
			return false
		}
	}
	return true
}

// DebugFillProperties adds the points, the rectangles, the status and the
// content flag.
//
// Flutter: rendering/selection.dart SelectionGeometry.debugFillProperties
func (g SelectionGeometry) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	b.Add(foundation.NewDiagnosticsPropertyOpt("startSelectionPoint", optPtr(g.StartSelectionPoint)))
	b.Add(foundation.NewDiagnosticsPropertyOpt("endSelectionPoint", optPtr(g.EndSelectionPoint)))
	b.Add(foundation.NewIterableProperty("selectionRects", g.SelectionRects))
	b.Add(foundation.NewEnumProperty("status", foundation.Some(g.Status)))
	b.Add(foundation.NewDiagnosticsProperty("hasContent", g.HasContent))
}
