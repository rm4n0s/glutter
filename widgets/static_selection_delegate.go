// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"slices"

	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// StaticSelectionContainerDelegate is a delegate that manages updating multiple
// Selectable children where the Selectables do not change or move around
// frequently.
//
// This delegate keeps track of the Selectables that received start or end
// SelectionEvents and the global locations of those events to accurately
// synthesize SelectionEvents for children Selectables when needed.
//
// When a new SelectionEdgeUpdateEvent is dispatched to a Selectable, this
// delegate checks whether the Selectable has already received a selection
// update for each edge that currently exists, and synthesizes an event for the
// edges that have not yet received an update. This synthesized event is
// dispatched before dispatching the new event.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate
type StaticSelectionContainerDelegate struct {
	MultiSelectableSelectionContainerDelegate
	self StaticDelegate

	// hasReceivedStartEvent is the set of selectables that have received start events.
	hasReceivedStartEvent map[rendering.Selectable]struct{}
	// hasReceivedEndEvent is the set of selectables that have received end events.
	hasReceivedEndEvent map[rendering.Selectable]struct{}
	// lastStartEdgeUpdateGlobalPosition is the global position of the last
	// selection start edge update.
	lastStartEdgeUpdateGlobalPosition *ui.Offset
	// lastEndEdgeUpdateGlobalPosition is the global position of the last
	// selection end edge update.
	lastEndEdgeUpdateGlobalPosition *ui.Offset
}

// StaticDelegate is what a delegate that embeds StaticSelectionContainerDelegate
// implements: the MultiSelectableDelegate.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate
type StaticDelegate interface {
	MultiSelectableDelegate
}

// NewStaticSelectionContainerDelegate creates the delegate.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.new
func NewStaticSelectionContainerDelegate() *StaticSelectionContainerDelegate {
	d := &StaticSelectionContainerDelegate{}
	InitStaticSelectionContainerDelegate(d, d)
	return d
}

// InitStaticSelectionContainerDelegate sets the delegate that embeds the
// static delegate, so that it calls its overrides.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitStaticSelectionContainerDelegate(d *StaticSelectionContainerDelegate, self StaticDelegate) {
	d.self = self
	d.hasReceivedStartEvent = map[rendering.Selectable]struct{}{}
	d.hasReceivedEndEvent = map[rendering.Selectable]struct{}{}
	InitMultiSelectableSelectionContainerDelegate(&d.MultiSelectableSelectionContainerDelegate, self)
}

// DidReceiveSelectionEventFor tracks whether a selection edge update event for
// a given selectable was received. When forEnd is true, the selectable is
// registered as having received an end event. When false, the selectable is
// registered as having received a start event. When forEnd is None, the
// selectable is registered as having received both. Call it when a selection
// event is dispatched to a child selectable managed by this delegate.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.didReceiveSelectionEventFor
func (d *StaticSelectionContainerDelegate) DidReceiveSelectionEventFor(selectable rendering.Selectable, forEnd *bool) {
	switch {
	case forEnd == nil:
		d.hasReceivedStartEvent[selectable] = struct{}{}
		d.hasReceivedEndEvent[selectable] = struct{}{}
	case *forEnd:
		d.hasReceivedEndEvent[selectable] = struct{}{}
	default:
		d.hasReceivedStartEvent[selectable] = struct{}{}
	}
}

// DidReceiveSelectionBoundaryEvents updates the internal selection state after
// a selection event that selects a boundary such as a word, a paragraph or all.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.didReceiveSelectionBoundaryEvents
func (d *StaticSelectionContainerDelegate) DidReceiveSelectionBoundaryEvents() {
	if d.CurrentSelectionStartIndex == -1 || d.CurrentSelectionEndIndex == -1 {
		return
	}
	start := min(d.CurrentSelectionStartIndex, d.CurrentSelectionEndIndex)
	end := max(d.CurrentSelectionStartIndex, d.CurrentSelectionEndIndex)
	for index := start; index <= end; index++ {
		d.DidReceiveSelectionEventFor(d.Selectables[index], nil)
	}
	d.updateLastSelectionEdgeLocationsFromGeometries()
}

// UpdateLastSelectionEdgeLocation updates the last selection edge location of
// the edge specified by forEnd to the provided global location.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.updateLastSelectionEdgeLocation
func (d *StaticSelectionContainerDelegate) UpdateLastSelectionEdgeLocation(globalSelectionEdgeLocation ui.Offset, forEnd bool) {
	if forEnd {
		d.lastEndEdgeUpdateGlobalPosition = &globalSelectionEdgeLocation
	} else {
		d.lastStartEdgeUpdateGlobalPosition = &globalSelectionEdgeLocation
	}
}

// updateLastSelectionEdgeLocationsFromGeometries updates the last selection edge
// locations of both start and end selection edges based on their geometry.
func (d *StaticSelectionContainerDelegate) updateLastSelectionEdgeLocationsFromGeometries() {
	if d.CurrentSelectionStartIndex != -1 && d.Selectables[d.CurrentSelectionStartIndex].Value().HasSelection() {
		start := d.Selectables[d.CurrentSelectionStartIndex]
		p := start.Value().StartSelectionPoint
		if p != nil {
			local := p.LocalPosition.Add(ui.Offset{Dy: -p.LineHeight / 2})
			d.UpdateLastSelectionEdgeLocation(start.GetTransformTo(nil).TransformPoint(local), false)
		}
	}
	if d.CurrentSelectionEndIndex != -1 && d.Selectables[d.CurrentSelectionEndIndex].Value().HasSelection() {
		end := d.Selectables[d.CurrentSelectionEndIndex]
		p := end.Value().EndSelectionPoint
		if p != nil {
			local := p.LocalPosition.Add(ui.Offset{Dy: -p.LineHeight / 2})
			d.UpdateLastSelectionEdgeLocation(end.GetTransformTo(nil).TransformPoint(local), true)
		}
	}
}

// ClearInternalSelectionState clears the internal selection state: no
// selectable child under this delegate has received start or end events, and
// the tracked global locations are reset.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.clearInternalSelectionState
func (d *StaticSelectionContainerDelegate) ClearInternalSelectionState() {
	for _, s := range d.Selectables {
		d.ClearInternalSelectionStateForSelectable(s)
	}
	d.lastStartEdgeUpdateGlobalPosition = nil
	d.lastEndEdgeUpdateGlobalPosition = nil
}

// ClearInternalSelectionStateForSelectable clears the internal selection state
// for a given selectable.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.clearInternalSelectionStateForSelectable
func (d *StaticSelectionContainerDelegate) ClearInternalSelectionStateForSelectable(selectable rendering.Selectable) {
	delete(d.hasReceivedStartEvent, selectable)
	delete(d.hasReceivedEndEvent, selectable)
}

// Remove removes the selectable and its state.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.remove
func (d *StaticSelectionContainerDelegate) Remove(selectable rendering.Selectable) {
	d.ClearInternalSelectionStateForSelectable(selectable)
	d.MultiSelectableSelectionContainerDelegate.Remove(selectable)
}

// HandleSelectAll selects all and tracks the boundary events.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.handleSelectAll
func (d *StaticSelectionContainerDelegate) HandleSelectAll(event rendering.SelectAllSelectionEvent) rendering.SelectionResult {
	result := d.MultiSelectableSelectionContainerDelegate.HandleSelectAll(event)
	d.DidReceiveSelectionBoundaryEvents()
	return result
}

// HandleSelectWord selects a word and tracks the boundary events.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.handleSelectWord
func (d *StaticSelectionContainerDelegate) HandleSelectWord(event rendering.SelectWordSelectionEvent) rendering.SelectionResult {
	result := d.MultiSelectableSelectionContainerDelegate.HandleSelectWord(event)
	d.DidReceiveSelectionBoundaryEvents()
	return result
}

// HandleSelectParagraph selects a paragraph and tracks the boundary events.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.handleSelectParagraph
func (d *StaticSelectionContainerDelegate) HandleSelectParagraph(event rendering.SelectParagraphSelectionEvent) rendering.SelectionResult {
	result := d.MultiSelectableSelectionContainerDelegate.HandleSelectParagraph(event)
	d.DidReceiveSelectionBoundaryEvents()
	return result
}

// HandleClearSelection clears the selection and the internal state.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.handleClearSelection
func (d *StaticSelectionContainerDelegate) HandleClearSelection(event rendering.ClearSelectionEvent) rendering.SelectionResult {
	result := d.MultiSelectableSelectionContainerDelegate.HandleClearSelection(event)
	d.ClearInternalSelectionState()
	return result
}

// HandleSelectionEdgeUpdate remembers the location of the edge and updates it.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.handleSelectionEdgeUpdate
func (d *StaticSelectionContainerDelegate) HandleSelectionEdgeUpdate(event rendering.SelectionEdgeUpdateEvent) rendering.SelectionResult {
	d.UpdateLastSelectionEdgeLocation(event.GlobalPosition, event.Type() == rendering.SelectionEventTypeEndEdgeUpdate)
	return d.MultiSelectableSelectionContainerDelegate.HandleSelectionEdgeUpdate(event)
}

// Dispose clears the internal state and releases the resources.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.dispose
func (d *StaticSelectionContainerDelegate) Dispose() {
	d.ClearInternalSelectionState()
	d.MultiSelectableSelectionContainerDelegate.Dispose()
}

// DispatchSelectionEventToChild tracks the events that the child received.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.dispatchSelectionEventToChild
func (d *StaticSelectionContainerDelegate) DispatchSelectionEventToChild(selectable rendering.Selectable, event rendering.SelectionEvent) rendering.SelectionResult {
	yes, no := true, false
	switch event.Type() {
	case rendering.SelectionEventTypeStartEdgeUpdate:
		d.DidReceiveSelectionEventFor(selectable, &no)
		d.self.EnsureChildUpdated(selectable)
	case rendering.SelectionEventTypeEndEdgeUpdate:
		d.DidReceiveSelectionEventFor(selectable, &yes)
		d.self.EnsureChildUpdated(selectable)
	case rendering.SelectionEventTypeClear:
		d.ClearInternalSelectionStateForSelectable(selectable)
	case rendering.SelectionEventTypeGranularlyExtendSelection, rendering.SelectionEventTypeDirectionallyExtendSelection:
		d.DidReceiveSelectionEventFor(selectable, nil)
		d.self.EnsureChildUpdated(selectable)
	}
	return d.MultiSelectableSelectionContainerDelegate.DispatchSelectionEventToChild(selectable, event)
}

// EnsureChildUpdated ensures the selectable child has received the most up to
// date selection events. It is called when a new selectable is added to the
// delegate, and its screen location falls into the previous selection, and
// before a start or end edge update, or an extend selection event is dispatched
// to a child.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.ensureChildUpdated
func (d *StaticSelectionContainerDelegate) EnsureChildUpdated(selectable rendering.Selectable) {
	if d.lastEndEdgeUpdateGlobalPosition != nil {
		if _, had := d.hasReceivedEndEvent[selectable]; !had {
			d.hasReceivedEndEvent[selectable] = struct{}{}
			synthesized := rendering.NewSelectionEdgeUpdateEventForEnd(*d.lastEndEdgeUpdateGlobalPosition, rendering.TextGranularityUnspecified)
			if d.CurrentSelectionEndIndex == -1 {
				d.self.HandleSelectionEdgeUpdate(synthesized)
			}
			selectable.DispatchSelectionEvent(synthesized)
		}
	}
	if d.lastStartEdgeUpdateGlobalPosition != nil {
		if _, had := d.hasReceivedStartEvent[selectable]; !had {
			d.hasReceivedStartEvent[selectable] = struct{}{}
			synthesized := rendering.NewSelectionEdgeUpdateEventForStart(*d.lastStartEdgeUpdateGlobalPosition, rendering.TextGranularityUnspecified)
			if d.CurrentSelectionStartIndex == -1 {
				d.self.HandleSelectionEdgeUpdate(synthesized)
			}
			selectable.DispatchSelectionEvent(synthesized)
		}
	}
}

// DidChangeSelectables updates the edges for the new set of selectables.
//
// Flutter: widgets/selectable_region.dart StaticSelectionContainerDelegate.didChangeSelectables
func (d *StaticSelectionContainerDelegate) DidChangeSelectables() {
	if d.lastEndEdgeUpdateGlobalPosition != nil {
		d.self.HandleSelectionEdgeUpdate(rendering.NewSelectionEdgeUpdateEventForEnd(*d.lastEndEdgeUpdateGlobalPosition, rendering.TextGranularityUnspecified))
	}
	if d.lastStartEdgeUpdateGlobalPosition != nil {
		d.self.HandleSelectionEdgeUpdate(rendering.NewSelectionEdgeUpdateEventForStart(*d.lastStartEdgeUpdateGlobalPosition, rendering.TextGranularityUnspecified))
	}
	for s := range d.hasReceivedEndEvent {
		if !slices.Contains(d.Selectables, s) {
			delete(d.hasReceivedEndEvent, s)
		}
	}
	for s := range d.hasReceivedStartEvent {
		if !slices.Contains(d.Selectables, s) {
			delete(d.hasReceivedStartEvent, s)
		}
	}
	d.MultiSelectableSelectionContainerDelegate.DidChangeSelectables()
}
