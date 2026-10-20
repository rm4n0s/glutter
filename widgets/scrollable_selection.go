package widgets

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// scrollableDefaultDragTargetSize: pointer drag is a single point, it should
// not have a size.
//
// Flutter: widgets/scrollable.dart _ScrollableSelectionContainerDelegate._kDefaultDragTargetSize
const scrollableDefaultDragTargetSize = 0.0

// scrollableDefaultSelectToScrollVelocityScalar is an eye-balled value for a
// smooth scrolling speed.
//
// Flutter: widgets/scrollable.dart _ScrollableSelectionContainerDelegate._kDefaultSelectToScrollVelocityScalar
const scrollableDefaultSelectToScrollVelocityScalar = 30.0

// scrollableSelectionHandler handles selection for a scrollable. It registers
// itself to the registrar and uses a SelectionContainer to collect selectables
// from its subtree.
//
// Flutter: widgets/scrollable.dart _ScrollableSelectionHandler
type scrollableSelectionHandler struct {
	StatefulWidget
	state     *ScrollableState
	position  ScrollPosition
	registrar rendering.SelectionRegistrar
	child     Widget
}

func (w *scrollableSelectionHandler) CreateState() State { return &scrollableSelectionHandlerState{} }

type scrollableSelectionHandlerState struct {
	StateOf[*scrollableSelectionHandler]
	selectionDelegate *scrollableSelectionContainerDelegate
}

func (s *scrollableSelectionHandlerState) InitState() {
	w := s.Widget()
	s.selectionDelegate = newScrollableSelectionContainerDelegate(w.state, w.position)
}

func (s *scrollableSelectionHandlerState) DidUpdateWidget(old *scrollableSelectionHandler) {
	if old.position != s.Widget().position {
		s.selectionDelegate.setPosition(s.Widget().position)
	}
}

func (s *scrollableSelectionHandlerState) Dispose() { s.selectionDelegate.Dispose() }

func (s *scrollableSelectionHandlerState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewSelectionContainer(s.selectionDelegate, w.child, SelectionContainerOpts{Registrar: w.registrar})
}

// scrollableSelectionContainerDelegate handles the case where the selectables
// change frequently, and it optimizes toward scrolling updates. It keeps track
// of the drag start offset relative to scroll origin for every selectable. The
// records are used to determine whether the selection is up to date with the
// scroll position when it sends the drag update event to a selectable.
//
// Flutter: widgets/scrollable.dart _ScrollableSelectionContainerDelegate
type scrollableSelectionContainerDelegate struct {
	MultiSelectableSelectionContainerDelegate
	state      *ScrollableState
	autoScroll *EdgeDraggingAutoScroller
	position   ScrollPosition
	posHandle  foundation.ListenerHandle

	scheduledLayoutChange           bool
	currentDragStartRelatedToOrigin foundation.Opt[ui.Offset]
	currentDragEndRelatedToOrigin   foundation.Opt[ui.Offset]
	// selectionStartsInScrollable: the scrollable only auto scrolls if the
	// selection starts in the scrollable.
	selectionStartsInScrollable bool

	// selectableStartEdgeUpdateRecords stores the scroll offset when a scrollable
	// receives the last start SelectionEdgeUpdateEvent. When a new event is
	// dispatched to a selectable, the delegate checks the current scroll offset
	// against the one stored in these records. If the scroll offset is different,
	// it synthesizes an opposite SelectionEdgeUpdateEvent and dispatches the
	// event before dispatching the new event.
	selectableStartEdgeUpdateRecords map[rendering.Selectable]float64
	selectableEndEdgeUpdateRecords   map[rendering.Selectable]float64
}

func newScrollableSelectionContainerDelegate(state *ScrollableState, position ScrollPosition) *scrollableSelectionContainerDelegate {
	d := &scrollableSelectionContainerDelegate{
		state: state, position: position,
		selectableStartEdgeUpdateRecords: map[rendering.Selectable]float64{},
		selectableEndEdgeUpdateRecords:   map[rendering.Selectable]float64{},
	}
	InitMultiSelectableSelectionContainerDelegate(&d.MultiSelectableSelectionContainerDelegate, d)
	d.autoScroll = NewEdgeDraggingAutoScroller(state, scrollableDefaultSelectToScrollVelocityScalar, nil)
	d.posHandle = position.AddListener(d.scheduleLayoutChange)
	return d
}

func (d *scrollableSelectionContainerDelegate) setPosition(other ScrollPosition) {
	if other == d.position {
		return
	}
	d.position.RemoveListener(d.posHandle)
	d.position = other
	d.posHandle = other.AddListener(d.scheduleLayoutChange)
}

// scheduleLayoutChange: the layout will only be updated a frame later than
// position changes, so it schedules a post frame callback to capture the
// accurate layout.
func (d *scrollableSelectionContainerDelegate) scheduleLayoutChange() {
	if d.scheduledLayoutChange {
		return
	}
	d.scheduledLayoutChange = true
	d.binding().RunPostFrame(func(time.Duration) {
		if !d.scheduledLayoutChange {
			return
		}
		d.scheduledLayoutChange = false
		d.LayoutDidChange()
	})
}

func (d *scrollableSelectionContainerDelegate) DidChangeSelectables() {
	set := map[rendering.Selectable]struct{}{}
	for _, s := range d.Selectables {
		set[s] = struct{}{}
	}
	for k := range d.selectableStartEdgeUpdateRecords {
		if _, ok := set[k]; !ok {
			delete(d.selectableStartEdgeUpdateRecords, k)
		}
	}
	for k := range d.selectableEndEdgeUpdateRecords {
		if _, ok := set[k]; !ok {
			delete(d.selectableEndEdgeUpdateRecords, k)
		}
	}
	d.MultiSelectableSelectionContainerDelegate.DidChangeSelectables()
}

func (d *scrollableSelectionContainerDelegate) HandleClearSelection(event rendering.ClearSelectionEvent) rendering.SelectionResult {
	clear(d.selectableStartEdgeUpdateRecords)
	clear(d.selectableEndEdgeUpdateRecords)
	d.currentDragStartRelatedToOrigin = foundation.None[ui.Offset]()
	d.currentDragEndRelatedToOrigin = foundation.None[ui.Offset]()
	d.selectionStartsInScrollable = false
	return d.MultiSelectableSelectionContainerDelegate.HandleClearSelection(event)
}

func (d *scrollableSelectionContainerDelegate) HandleSelectionEdgeUpdate(event rendering.SelectionEdgeUpdateEvent) rendering.SelectionResult {
	if d.currentDragEndRelatedToOrigin.IsNone() && d.currentDragStartRelatedToOrigin.IsNone() {
		d.selectionStartsInScrollable = d.globalPositionInScrollable(event.GlobalPosition)
	}
	deltaToOrigin := d.state.DeltaToScrollOrigin()
	if event.Type() == rendering.SelectionEventTypeEndEdgeUpdate {
		origin := d.inferPositionRelatedToOrigin(event.GlobalPosition)
		d.currentDragEndRelatedToOrigin = foundation.Some(origin)
		event = rendering.NewSelectionEdgeUpdateEventForEnd(origin.Translate(-deltaToOrigin.Dx, -deltaToOrigin.Dy), event.Granularity)
	} else {
		origin := d.inferPositionRelatedToOrigin(event.GlobalPosition)
		d.currentDragStartRelatedToOrigin = foundation.Some(origin)
		event = rendering.NewSelectionEdgeUpdateEventForStart(origin.Translate(-deltaToOrigin.Dx, -deltaToOrigin.Dy), event.Granularity)
	}
	result := d.MultiSelectableSelectionContainerDelegate.HandleSelectionEdgeUpdate(event)
	// Result may be pending if one of the selectable child is also a
	// scrollable. In that case, the parent scrollable needs to wait for the child
	// to finish scrolling.
	if result == rendering.SelectionResultPending {
		d.autoScroll.StopAutoScroll()
		return result
	}
	if d.selectionStartsInScrollable {
		d.autoScroll.StartAutoScrollIfNecessary(ui.NewRectFromCenter(event.GlobalPosition, scrollableDefaultDragTargetSize, scrollableDefaultDragTargetSize))
		if d.autoScroll.Scrolling() {
			return rendering.SelectionResultPending
		}
	}
	return result
}

func (d *scrollableSelectionContainerDelegate) scrollableBox() rendering.Box {
	return d.state.Context().FindRenderObject().(rendering.Box)
}

func (d *scrollableSelectionContainerDelegate) inferPositionRelatedToOrigin(globalPosition ui.Offset) ui.Offset {
	box := d.scrollableBox()
	localPosition := box.GlobalToLocal(globalPosition, nil)
	if !d.selectionStartsInScrollable {
		// If the selection starts outside of the scrollable, selecting across the
		// scrollable boundary will act as selecting the entire content in the
		// scrollable. This logic move the offset to the 0.0 or infinity to cover
		// the entire content if the input position is outside of the scrollable.
		if localPosition.Dy < 0 || localPosition.Dx < 0 {
			return box.LocalToGlobal(ui.Offset{}, nil)
		}
		if localPosition.Dy > box.Size().Height || localPosition.Dx > box.Size().Width {
			return ui.OffsetInfinite
		}
	}
	deltaToOrigin := d.state.DeltaToScrollOrigin()
	return box.LocalToGlobal(localPosition.Translate(deltaToOrigin.Dx, deltaToOrigin.Dy), nil)
}

// updateDragLocationsFromGeometries infers the current drag start and end
// related to origin from the geometry. It is called after a select word and
// select all event where the selection is triggered by none drag events.
func (d *scrollableSelectionContainerDelegate) updateDragLocationsFromGeometries(forceUpdateStart, forceUpdateEnd bool) {
	deltaToOrigin := d.state.DeltaToScrollOrigin()
	box := d.scrollableBox()
	transform := box.GetTransformTo(nil)
	if d.CurrentSelectionStartIndex != -1 && (d.currentDragStartRelatedToOrigin.IsNone() || forceUpdateStart) {
		s := d.Selectables[d.CurrentSelectionStartIndex]
		start := s.Value().StartSelectionPoint
		if start != nil {
			localDragStart := s.GetTransformTo(box).TransformPoint(start.LocalPosition.Add(ui.Offset{Dy: -start.LineHeight / 2}))
			d.currentDragStartRelatedToOrigin = foundation.Some(transform.TransformPoint(localDragStart.Add(deltaToOrigin)))
		}
	}
	if d.CurrentSelectionEndIndex != -1 && (d.currentDragEndRelatedToOrigin.IsNone() || forceUpdateEnd) {
		s := d.Selectables[d.CurrentSelectionEndIndex]
		end := s.Value().EndSelectionPoint
		if end != nil {
			localDragEnd := s.GetTransformTo(box).TransformPoint(end.LocalPosition.Add(ui.Offset{Dy: -end.LineHeight / 2}))
			d.currentDragEndRelatedToOrigin = foundation.Some(transform.TransformPoint(localDragEnd.Add(deltaToOrigin)))
		}
	}
}

func (d *scrollableSelectionContainerDelegate) HandleSelectAll(event rendering.SelectAllSelectionEvent) rendering.SelectionResult {
	result := d.MultiSelectableSelectionContainerDelegate.HandleSelectAll(event)
	if d.CurrentSelectionStartIndex != -1 {
		d.updateDragLocationsFromGeometries(true, true)
	}
	return result
}

func (d *scrollableSelectionContainerDelegate) HandleSelectWord(event rendering.SelectWordSelectionEvent) rendering.SelectionResult {
	d.selectionStartsInScrollable = d.globalPositionInScrollable(event.GlobalPosition)
	result := d.MultiSelectableSelectionContainerDelegate.HandleSelectWord(event)
	d.updateDragLocationsFromGeometries(true, true)
	return result
}

func (d *scrollableSelectionContainerDelegate) HandleGranularlyExtendSelection(event rendering.GranularlyExtendSelectionEvent) rendering.SelectionResult {
	result := d.MultiSelectableSelectionContainerDelegate.HandleGranularlyExtendSelection(event)
	// The selection geometry may not have the accurate offset for the edges that
	// are outside of the viewport whose transform may not be valid. Only the edge
	// this event is updating is sure to be accurate.
	d.updateDragLocationsFromGeometries(!event.IsEnd, event.IsEnd)
	if d.selectionStartsInScrollable {
		d.jumpToEdge(event.IsEnd)
	}
	return result
}

func (d *scrollableSelectionContainerDelegate) HandleDirectionallyExtendSelection(event rendering.DirectionallyExtendSelectionEvent) rendering.SelectionResult {
	result := d.MultiSelectableSelectionContainerDelegate.HandleDirectionallyExtendSelection(event)
	d.updateDragLocationsFromGeometries(!event.IsEnd, event.IsEnd)
	if d.selectionStartsInScrollable {
		d.jumpToEdge(event.IsEnd)
	}
	return result
}

func (d *scrollableSelectionContainerDelegate) jumpToEdge(isExtent bool) {
	var selectable rendering.Selectable
	var edge *rendering.SelectionPoint
	if isExtent {
		selectable = d.Selectables[d.CurrentSelectionEndIndex]
		edge = selectable.Value().EndSelectionPoint
	} else {
		selectable = d.Selectables[d.CurrentSelectionStartIndex]
		edge = selectable.Value().StartSelectionPoint
	}
	if edge == nil {
		return
	}
	lineHeight := edge.LineHeight
	scrollableBox := d.scrollableBox()
	edgeOffset := selectable.GetTransformTo(scrollableBox).TransformPoint(edge.LocalPosition)
	rect := ui.NewRectFromLTRB(0, 0, scrollableBox.Size().Width, scrollableBox.Size().Height)
	pos := d.state.Position()
	switch d.state.AxisDirection() {
	case painting.AxisDirectionUp:
		edgeBottom, edgeTop := edgeOffset.Dy, edgeOffset.Dy-lineHeight
		if edgeBottom >= rect.Bottom && edgeTop <= rect.Top {
			return
		}
		if edgeBottom > rect.Bottom {
			pos.JumpTo(pos.Pixels() + rect.Bottom - edgeBottom)
			return
		}
		if edgeTop < rect.Top {
			pos.JumpTo(pos.Pixels() + rect.Top - edgeTop)
		}
	case painting.AxisDirectionRight:
		e := edgeOffset.Dx
		if e >= rect.Right && e <= rect.Left {
			return
		}
		if e > rect.Right {
			pos.JumpTo(pos.Pixels() + e - rect.Right)
			return
		}
		if e < rect.Left {
			pos.JumpTo(pos.Pixels() + e - rect.Left)
		}
	case painting.AxisDirectionDown:
		edgeBottom, edgeTop := edgeOffset.Dy, edgeOffset.Dy-lineHeight
		if edgeBottom >= rect.Bottom && edgeTop <= rect.Top {
			return
		}
		if edgeBottom > rect.Bottom {
			pos.JumpTo(pos.Pixels() + edgeBottom - rect.Bottom)
			return
		}
		if edgeTop < rect.Top {
			pos.JumpTo(pos.Pixels() + edgeTop - rect.Top)
		}
	case painting.AxisDirectionLeft:
		e := edgeOffset.Dx
		if e >= rect.Right && e <= rect.Left {
			return
		}
		if e > rect.Right {
			pos.JumpTo(pos.Pixels() + rect.Right - e)
			return
		}
		if e < rect.Left {
			pos.JumpTo(pos.Pixels() + rect.Left - e)
		}
	}
}

func (d *scrollableSelectionContainerDelegate) globalPositionInScrollable(globalPosition ui.Offset) bool {
	box := d.scrollableBox()
	local := box.GlobalToLocal(globalPosition, nil)
	return ui.NewRectFromLTWH(0, 0, box.Size().Width, box.Size().Height).Contains(local)
}

func (d *scrollableSelectionContainerDelegate) DispatchSelectionEventToChild(selectable rendering.Selectable, event rendering.SelectionEvent) rendering.SelectionResult {
	pixels := d.state.Position().Pixels()
	switch event.Type() {
	case rendering.SelectionEventTypeStartEdgeUpdate:
		d.selectableStartEdgeUpdateRecords[selectable] = pixels
		d.EnsureChildUpdated(selectable)
	case rendering.SelectionEventTypeEndEdgeUpdate:
		d.selectableEndEdgeUpdateRecords[selectable] = pixels
		d.EnsureChildUpdated(selectable)
	case rendering.SelectionEventTypeGranularlyExtendSelection, rendering.SelectionEventTypeDirectionallyExtendSelection:
		d.EnsureChildUpdated(selectable)
		d.selectableStartEdgeUpdateRecords[selectable] = pixels
		d.selectableEndEdgeUpdateRecords[selectable] = pixels
	case rendering.SelectionEventTypeClear:
		delete(d.selectableEndEdgeUpdateRecords, selectable)
		delete(d.selectableStartEdgeUpdateRecords, selectable)
	case rendering.SelectionEventTypeSelectAll, rendering.SelectionEventTypeSelectWord, rendering.SelectionEventTypeSelectParagraph:
		d.selectableEndEdgeUpdateRecords[selectable] = pixels
		d.selectableStartEdgeUpdateRecords[selectable] = pixels
	}
	return d.MultiSelectableSelectionContainerDelegate.DispatchSelectionEventToChild(selectable, event)
}

func (d *scrollableSelectionContainerDelegate) EnsureChildUpdated(selectable rendering.Selectable) {
	newRecord := d.state.Position().Pixels()
	previousStart, hasStart := d.selectableStartEdgeUpdateRecords[selectable]
	if origin, ok := d.currentDragStartRelatedToOrigin.Get(); ok && (!hasStart || math.Abs(newRecord-previousStart) > foundation.PrecisionErrorTolerance) {
		// Make sure the selectable has up to date events.
		delta := d.state.DeltaToScrollOrigin()
		start := origin.Translate(-delta.Dx, -delta.Dy)
		selectable.DispatchSelectionEvent(rendering.NewSelectionEdgeUpdateEventForStart(start, rendering.TextGranularityUnspecified))
		// Make sure we track that we have synthesized a start event for this
		// selectable, so we don't synthesize events unnecessarily.
		d.selectableStartEdgeUpdateRecords[selectable] = newRecord
	}
	previousEnd, hasEnd := d.selectableEndEdgeUpdateRecords[selectable]
	if origin, ok := d.currentDragEndRelatedToOrigin.Get(); ok && (!hasEnd || math.Abs(newRecord-previousEnd) > foundation.PrecisionErrorTolerance) {
		delta := d.state.DeltaToScrollOrigin()
		end := origin.Translate(-delta.Dx, -delta.Dy)
		selectable.DispatchSelectionEvent(rendering.NewSelectionEdgeUpdateEventForEnd(end, rendering.TextGranularityUnspecified))
		d.selectableEndEdgeUpdateRecords[selectable] = newRecord
	}
}

func (d *scrollableSelectionContainerDelegate) Dispose() {
	clear(d.selectableStartEdgeUpdateRecords)
	clear(d.selectableEndEdgeUpdateRecords)
	d.scheduledLayoutChange = false
	d.autoScroll.StopAutoScroll()
	d.position.RemoveListener(d.posHandle)
	d.MultiSelectableSelectionContainerDelegate.Dispose()
}
