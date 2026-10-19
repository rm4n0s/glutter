package widgets

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// ReorderCallback is called when an item is dropped at a new place: oldIndex is
// the index of the item, newIndex the index of the place.
//
// Flutter: widgets/reorderable_list.dart ReorderCallback
type ReorderCallback func(oldIndex, newIndex int)

// ReorderItemProxyDecorator decorates the item that is being dragged. The
// animation goes from 0 to 1 when the drag starts, and from 1 to 0 when it ends.
//
// Flutter: widgets/reorderable_list.dart ReorderItemProxyDecorator
type ReorderItemProxyDecorator func(child Widget, index int, animation animation.Animation[float64]) Widget

// ReorderDragBoundaryProvider gives the boundary that an item that is dragged
// stays within, or nil for none.
//
// Flutter: widgets/reorderable_list.dart ReorderDragBoundaryProvider
type ReorderDragBoundaryProvider func(bc BuildContext) DragBoundaryDelegate[ui.Rect]

// reorderableAutoScrollVelocityScalar is an eyeballed value for a smooth
// scrolling experience.
//
// Flutter: widgets/reorderable_list.dart SliverReorderableList._kDefaultAutoScrollVelocityScalar
const reorderableAutoScrollVelocityScalar = 50.0

// reorderableExtentChecks are the checks the constructors of ReorderableList and
// SliverReorderableList share.
func reorderableChecks(itemCount int, itemExtent foundation.Opt[float64], itemExtentBuilder rendering.ItemExtentBuilder, prototypeItem Widget, onReorder, onReorderItem ReorderCallback) {
	if !foundation.DebugMode {
		return
	}
	if itemCount < 0 {
		panic(errors.New("ReorderableItemCountNegative", "the itemCount must not be negative", "itemCount", itemCount))
	}
	given := 0
	if itemExtent.IsSome() {
		given++
	}
	if itemExtentBuilder != nil {
		given++
	}
	if prototypeItem != nil {
		given++
	}
	if given > 1 {
		panic(errors.New("ReorderableExtentConflict", "you can only pass one of itemExtent, prototypeItem and itemExtentBuilder"))
	}
	if (onReorderItem != nil) == (onReorder != nil) {
		panic(errors.New("ReorderableCallbackInvalid", "exactly one of onReorderItem and onReorder must be given; the onReorder callback is obsolete and is replaced by onReorderItem"))
	}
}

// ReorderableListOpts are the optional parameters of NewReorderableList.
//
// Flutter: widgets/reorderable_list.dart ReorderableList.new
type ReorderableListOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnReorder is called when an item is dropped. Deprecated in Flutter in favor
	// of OnReorderItem; exactly one of the two must be given.
	OnReorder ReorderCallback
	// OnReorderItem is called when an item is dropped; the new index is that of
	// the item after it is removed from the old one.
	OnReorderItem ReorderCallback
	// OnReorderStart is called when the user starts to drag an item.
	OnReorderStart func(index int)
	// OnReorderEnd is called when the user drops the item.
	OnReorderEnd func(index int)
	// ItemExtent is the extent of all the items.
	ItemExtent foundation.Opt[float64]
	// ItemExtentBuilder gives the extent of each item.
	ItemExtentBuilder rendering.ItemExtentBuilder
	// PrototypeItem is the widget whose extent all the items have.
	PrototypeItem Widget
	// ProxyDecorator decorates the item that is being dragged.
	ProxyDecorator ReorderItemProxyDecorator
	// Padding is the space around the items.
	Padding painting.EdgeInsetsGeometry
	// ScrollDirection is the axis along which the list scrolls; the default is
	// AxisVertical.
	ScrollDirection painting.Axis
	// Reverse is whether the list scrolls against the reading direction.
	Reverse bool
	// Controller reads and controls the scroll position.
	Controller ScrollController
	// Primary is whether this is the primary scroll view.
	Primary foundation.Opt[bool]
	// Physics is how the list responds to the user.
	Physics ScrollPhysics
	// ShrinkWrap is whether the extent of the list is the extent of its content.
	ShrinkWrap bool
	// Anchor is the relative position of the zero scroll offset, from 0 to 1.
	Anchor float64
	// ScrollCacheExtent is how far beyond the viewport the list builds items.
	ScrollCacheExtent foundation.Opt[rendering.ScrollCacheExtent]
	// DragStartBehavior is where a drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// KeyboardDismissBehavior is how the list dismisses the keyboard.
	KeyboardDismissBehavior ScrollViewKeyboardDismissBehavior
	// RestorationID is the identifier for restoring the scroll offset.
	RestorationID string
	// ClipBehavior is how the list clips; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// AutoScrollerVelocityScalar is the speed of the scrolling when an item is
	// dragged to the edge; the default is 50.
	AutoScrollerVelocityScalar foundation.Opt[float64]
	// DragBoundaryProvider gives the boundary the dragged item stays within.
	DragBoundaryProvider ReorderDragBoundaryProvider
}

// ReorderableList is a scrolling container that allows the user to interactively
// reorder the list items. The items need keys, and a drag start listener
// (ReorderableDragStartListener or ReorderableDelayedDragStartListener) in each
// item starts the drag.
//
// Flutter: widgets/reorderable_list.dart ReorderableList
type ReorderableList struct {
	StatefulWidget
	itemBuilder IndexedWidgetBuilder
	itemCount   int
	opts        ReorderableListOpts
}

// NewReorderableList creates a list of itemCount items that the builder builds.
// In debug builds it panics when itemCount is negative (ReorderableItemCountNegative),
// when more than one of the extents is given (ReorderableExtentConflict), or when
// not exactly one of OnReorder and OnReorderItem is given
// (ReorderableCallbackInvalid).
//
// Flutter: widgets/reorderable_list.dart ReorderableList.new
func NewReorderableList(itemBuilder IndexedWidgetBuilder, itemCount int, opts ...ReorderableListOpts) *ReorderableList {
	o := firstOpts(opts)
	reorderableChecks(itemCount, o.ItemExtent, o.ItemExtentBuilder, o.PrototypeItem, o.OnReorder, o.OnReorderItem)
	if o.ScrollDirection == painting.AxisUnspecified {
		o.ScrollDirection = painting.AxisVertical
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &ReorderableList{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), itemBuilder, itemCount, o}
}

// ItemCount is the number of items.
//
// Flutter: widgets/reorderable_list.dart ReorderableList.itemCount
func (w *ReorderableList) ItemCount() int { return w.itemCount }

// CreateState creates the state.
//
// Flutter: widgets/reorderable_list.dart ReorderableList.createState
func (w *ReorderableList) CreateState() State {
	return &ReorderableListState{key: NewGlobalKey[*SliverReorderableListState]()}
}

// ReorderableListMaybeOf is the state of the closest ReorderableList above the
// context, and false if there is none.
//
// Flutter: widgets/reorderable_list.dart ReorderableList.maybeOf
func ReorderableListMaybeOf(bc BuildContext) (*ReorderableListState, bool) {
	return FindAncestorStateOfType[*ReorderableListState](bc)
}

// ReorderableListOf is the state of the closest ReorderableList above the
// context. In debug builds it panics with the tag ReorderableListMissing when
// there is none.
//
// Flutter: widgets/reorderable_list.dart ReorderableList.of
func ReorderableListOf(bc BuildContext) *ReorderableListState {
	s, ok := ReorderableListMaybeOf(bc)
	if !ok && foundation.DebugMode {
		panic(errors.New("ReorderableListMissing", "ReorderableListOf() called with a context that does not contain a ReorderableList; this can happen when the context provided is from the same StatefulWidget that built the ReorderableList",
			"context", bc.(Element).String()))
	}
	return s
}

// ReorderableListState is the state of a ReorderableList: it can start and
// cancel the reordering of an item.
//
// Flutter: widgets/reorderable_list.dart ReorderableListState
type ReorderableListState struct {
	StateOf[*ReorderableList]
	key *GlobalKey[*SliverReorderableListState]
}

// StartItemDragReorder starts dragging the item at the index with the pointer
// of the event, which the recognizer tracks.
//
// Flutter: widgets/reorderable_list.dart ReorderableListState.startItemDragReorder
func (s *ReorderableListState) StartItemDragReorder(index int, event *gestures.PointerDownEvent, recognizer *gestures.MultiDragGestureRecognizer) {
	list, _ := s.key.CurrentState()
	list.StartItemDragReorder(index, event, recognizer)
}

// CancelReorder cancels the reordering that is in progress.
//
// Flutter: widgets/reorderable_list.dart ReorderableListState.cancelReorder
func (s *ReorderableListState) CancelReorder() {
	list, _ := s.key.CurrentState()
	list.CancelReorder()
}

// Build builds the scroll view with the reorderable sliver.
func (s *ReorderableListState) Build(BuildContext) Widget {
	w := s.Widget()
	o := w.opts
	var padding painting.EdgeInsetsGeometry = painting.EdgeInsetsZero
	if o.Padding != nil {
		padding = o.Padding
	}
	return NewCustomScrollView([]Widget{
		NewSliverPadding(padding, NewSliverReorderableList(w.itemBuilder, w.itemCount, SliverReorderableListOpts{
			Key: s.key, ItemExtent: o.ItemExtent, PrototypeItem: o.PrototypeItem, ItemExtentBuilder: o.ItemExtentBuilder,
			OnReorder: o.OnReorder, OnReorderItem: o.OnReorderItem, OnReorderStart: o.OnReorderStart, OnReorderEnd: o.OnReorderEnd,
			ProxyDecorator: o.ProxyDecorator, AutoScrollerVelocityScalar: o.AutoScrollerVelocityScalar, DragBoundaryProvider: o.DragBoundaryProvider,
		})),
	}, CustomScrollViewOpts{
		ScrollDirection: o.ScrollDirection, Reverse: o.Reverse, Controller: o.Controller, Primary: o.Primary, Physics: o.Physics,
		ShrinkWrap: o.ShrinkWrap, Anchor: o.Anchor, ScrollCacheExtent: o.ScrollCacheExtent, DragStartBehavior: o.DragStartBehavior,
		KeyboardDismissBehavior: o.KeyboardDismissBehavior, RestorationID: o.RestorationID, ClipBehavior: o.ClipBehavior,
	})
}

// SliverReorderableListOpts are the optional parameters of
// NewSliverReorderableList.
//
// Flutter: widgets/reorderable_list.dart SliverReorderableList.new
type SliverReorderableListOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// FindChildIndexCallback finds the index of the item that has a key.
	FindChildIndexCallback ChildIndexGetter
	// OnReorder is called when an item is dropped; see ReorderableListOpts.
	OnReorder ReorderCallback
	// OnReorderItem is called when an item is dropped; see ReorderableListOpts.
	OnReorderItem ReorderCallback
	// OnReorderStart is called when the user starts to drag an item.
	OnReorderStart func(index int)
	// OnReorderEnd is called when the user drops the item.
	OnReorderEnd func(index int)
	// ItemExtent is the extent of all the items.
	ItemExtent foundation.Opt[float64]
	// ItemExtentBuilder gives the extent of each item.
	ItemExtentBuilder rendering.ItemExtentBuilder
	// PrototypeItem is the widget whose extent all the items have.
	PrototypeItem Widget
	// ProxyDecorator decorates the item that is being dragged.
	ProxyDecorator ReorderItemProxyDecorator
	// DragBoundaryProvider gives the boundary the dragged item stays within.
	DragBoundaryProvider ReorderDragBoundaryProvider
	// AutoScrollerVelocityScalar is the speed of the scrolling when an item is
	// dragged to the edge; the default is 50.
	AutoScrollerVelocityScalar foundation.Opt[float64]
}

// SliverReorderableList is a sliver list that allows the user to interactively
// reorder the list items.
//
// Flutter: widgets/reorderable_list.dart SliverReorderableList
type SliverReorderableList struct {
	StatefulWidget
	itemBuilder                IndexedWidgetBuilder
	itemCount                  int
	autoScrollerVelocityScalar float64
	opts                       SliverReorderableListOpts
}

// NewSliverReorderableList creates the sliver; see NewReorderableList for the
// errors.
//
// Flutter: widgets/reorderable_list.dart SliverReorderableList.new
func NewSliverReorderableList(itemBuilder IndexedWidgetBuilder, itemCount int, opts ...SliverReorderableListOpts) *SliverReorderableList {
	o := firstOpts(opts)
	reorderableChecks(itemCount, o.ItemExtent, o.ItemExtentBuilder, o.PrototypeItem, o.OnReorder, o.OnReorderItem)
	return &SliverReorderableList{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), itemBuilder, itemCount,
		o.AutoScrollerVelocityScalar.OrElse(reorderableAutoScrollVelocityScalar), o}
}

// ItemCount is the number of items.
//
// Flutter: widgets/reorderable_list.dart SliverReorderableList.itemCount
func (w *SliverReorderableList) ItemCount() int { return w.itemCount }

// CreateState creates the state.
//
// Flutter: widgets/reorderable_list.dart SliverReorderableList.createState
func (w *SliverReorderableList) CreateState() State { return &SliverReorderableListState{} }

// SliverReorderableListMaybeOf is the state of the closest SliverReorderableList
// above the context, and false if there is none.
//
// Flutter: widgets/reorderable_list.dart SliverReorderableList.maybeOf
func SliverReorderableListMaybeOf(bc BuildContext) (*SliverReorderableListState, bool) {
	return FindAncestorStateOfType[*SliverReorderableListState](bc)
}

// SliverReorderableListOf is the state of the closest SliverReorderableList above
// the context. In debug builds it panics with the tag
// SliverReorderableListMissing when there is none.
//
// Flutter: widgets/reorderable_list.dart SliverReorderableList.of
func SliverReorderableListOf(bc BuildContext) *SliverReorderableListState {
	s, ok := SliverReorderableListMaybeOf(bc)
	if !ok && foundation.DebugMode {
		panic(errors.New("SliverReorderableListMissing", "SliverReorderableListOf() called with a context that does not contain a SliverReorderableList; this can happen when the context provided is from the same StatefulWidget that built the SliverReorderableList",
			"context", bc.(Element).String()))
	}
	return s
}

// reorderableItems maps indices to item states and iterates in insertion order,
// as Dart's map does.
type reorderableItems struct {
	items map[int]*reorderableItemState
	order []int
}

func (m *reorderableItems) get(i int) *reorderableItemState { return m.items[i] }

func (m *reorderableItems) set(i int, s *reorderableItemState) {
	if m.items == nil {
		m.items = map[int]*reorderableItemState{}
	}
	if _, ok := m.items[i]; !ok {
		m.order = append(m.order, i)
	}
	m.items[i] = s
}

func (m *reorderableItems) remove(i int) {
	if _, ok := m.items[i]; !ok {
		return
	}
	delete(m.items, i)
	for k, v := range m.order {
		if v == i {
			m.order = append(m.order[:k], m.order[k+1:]...)
			break
		}
	}
}

func (m *reorderableItems) values() []*reorderableItemState {
	out := make([]*reorderableItemState, 0, len(m.order))
	for _, i := range m.order {
		out = append(out, m.items[i])
	}
	return out
}

// SliverReorderableListState is the state of a SliverReorderableList: it tracks
// the item that is dragged and moves the others to make room for it.
//
// Flutter: widgets/reorderable_list.dart SliverReorderableListState
type SliverReorderableListState struct {
	StateOf[*SliverReorderableList]
	TickerProvider
	// items maps the index to the state of the item, to manage where the dragging
	// item will be inserted.
	items             reorderableItems
	overlayEntry      *OverlayEntry
	dragIndex         foundation.Opt[int]
	dragInfo          *reorderDragInfo
	insertIndex       foundation.Opt[int]
	finalDropPosition foundation.Opt[ui.Offset]
	recognizer        *gestures.MultiDragGestureRecognizer
	recognizerPointer foundation.Opt[int]
	autoScroller      *EdgeDraggingAutoScroller
	scrollable        *ScrollableState
}

func (s *SliverReorderableListState) scrollDirection() painting.Axis {
	return painting.AxisDirectionToAxis(s.scrollable.AxisDirection())
}

func (s *SliverReorderableListState) reverse() bool {
	return painting.AxisDirectionIsReversed(s.scrollable.AxisDirection())
}

// DidChangeDependencies finds the scrollable.
func (s *SliverReorderableListState) DidChangeDependencies() {
	s.scrollable = ScrollableOf(s.Context())
	if s.autoScroller == nil || s.autoScroller.Scrollable() != s.scrollable {
		if s.autoScroller != nil {
			s.autoScroller.StopAutoScroll()
		}
		s.autoScroller = NewEdgeDraggingAutoScroller(s.scrollable, s.Widget().autoScrollerVelocityScalar, s.handleScrollableAutoScrolled)
	}
}

// DidUpdateWidget cancels the reordering when the item count changed.
func (s *SliverReorderableListState) DidUpdateWidget(oldWidget *SliverReorderableList) {
	w := s.Widget()
	if w.itemCount != oldWidget.itemCount {
		s.CancelReorder()
	}
	if w.autoScrollerVelocityScalar != oldWidget.autoScrollerVelocityScalar {
		if s.autoScroller != nil {
			s.autoScroller.StopAutoScroll()
		}
		s.autoScroller = NewEdgeDraggingAutoScroller(s.scrollable, w.autoScrollerVelocityScalar, s.handleScrollableAutoScrolled)
	}
}

// Dispose resets the drag.
func (s *SliverReorderableListState) Dispose() {
	s.dragReset()
	if s.recognizer != nil {
		s.recognizer.Dispose()
	}
}

// StartItemDragReorder starts dragging the item at the index with the pointer of
// the event, which the recognizer tracks. In debug builds it panics with the tag
// ReorderableItemNotVisible when the item is not built.
//
// Flutter: widgets/reorderable_list.dart SliverReorderableListState.startItemDragReorder
func (s *SliverReorderableListState) StartItemDragReorder(index int, event *gestures.PointerDownEvent, recognizer *gestures.MultiDragGestureRecognizer) {
	if foundation.DebugMode && (index < 0 || index >= s.Widget().itemCount) {
		panic(errors.New("ReorderableIndexInvalid", "the index to drag is out of range", "index", index))
	}
	s.SetState(func() {
		if s.dragInfo != nil {
			s.CancelReorder()
		} else if s.recognizer != nil && s.recognizerPointer != foundation.Some(event.Pointer()) {
			s.recognizer.Dispose()
			s.recognizer = nil
			s.recognizerPointer = foundation.None[int]()
		}
		if s.items.get(index) != nil {
			s.dragIndex = foundation.Some(index)
			s.recognizer = recognizer
			s.recognizer.OnStart = s.dragStart
			s.recognizer.AddPointer(event)
			s.recognizerPointer = foundation.Some(event.Pointer())
		} else {
			panic(errors.New("ReorderableItemNotVisible", "attempting to start a drag on a non-visible item", "index", index))
		}
	})
}

// CancelReorder cancels the reordering that is in progress.
//
// Flutter: widgets/reorderable_list.dart SliverReorderableListState.cancelReorder
func (s *SliverReorderableListState) CancelReorder() {
	s.SetState(func() { s.dragReset() })
}

func (s *SliverReorderableListState) registerItem(item *reorderableItemState) {
	if s.dragInfo != nil && s.items.get(item.index()) != item {
		item.updateForGap(s.dragInfo.index, s.dragInfo.index, s.dragInfo.itemExtent, false, s.reverse())
	}
	s.items.set(item.index(), item)
	if s.dragInfo != nil && item.index() == s.dragInfo.index {
		item.dragging = true
		item.rebuild()
	}
}

func (s *SliverReorderableListState) unregisterItem(index int, item *reorderableItemState) {
	if s.items.get(index) == item {
		s.items.remove(index)
	}
}

func (s *SliverReorderableListState) dragStart(position ui.Offset) gestures.Drag {
	if foundation.DebugMode && s.dragInfo != nil {
		panic(errors.New("ReorderableDragActive", "a drag started while another one is active"))
	}
	item := s.items.get(s.dragIndex.Value())
	item.SetDragging(true)
	if cb := s.Widget().opts.OnReorderStart; cb != nil {
		cb(s.dragIndex.Value())
	}
	item.rebuild()

	s.insertIndex = foundation.Some(item.index())
	s.dragInfo = newReorderDragInfo(item, position, s.scrollDirection(), s.dragUpdate, s.dragCancel, s.dragEnd, s.dropCompleted,
		s.Widget().opts.ProxyDecorator, s)
	s.dragInfo.startDrag()

	overlay := OverlayOf(s.Context(), OverlayOfOpts{DebugRequiredFor: s.Widget()})
	s.overlayEntry = NewOverlayEntry(s.dragInfo.createProxy)
	overlay.Insert(s.overlayEntry)

	for _, childItem := range s.items.values() {
		if childItem == item || !childItem.Mounted() {
			continue
		}
		childItem.updateForGap(s.insertIndex.Value(), s.insertIndex.Value(), s.dragInfo.itemExtent, false, s.reverse())
	}
	return s.dragInfo
}

func (s *SliverReorderableListState) dragUpdate(_ *reorderDragInfo, _, _ ui.Offset) {
	s.SetState(func() {
		if s.overlayEntry != nil {
			s.overlayEntry.MarkNeedsBuild()
		}
		s.dragUpdateItems()
		if s.autoScroller != nil {
			s.autoScroller.StartAutoScrollIfNecessary(s.dragTargetRect())
		}
	})
}

func (s *SliverReorderableListState) dragCancel(*reorderDragInfo) {
	s.SetState(func() { s.dragReset() })
}

func (s *SliverReorderableListState) dragEnd(item *reorderDragInfo) {
	s.SetState(func() {
		insertIndex := s.insertIndex.Value()
		switch {
		case insertIndex-item.index == 1:
			// When returning to the original position from below, the insert index
			// equals item.index + 1 because it is calculated with the dragged item
			// still present. The actual target position is used for the animation.
			s.finalDropPosition = foundation.Some(s.itemOffsetAt(insertIndex - 1))
		case insertIndex == item.index:
			// No movement: animate to the current position.
			s.finalDropPosition = foundation.Some(s.itemOffsetAt(insertIndex))
		case s.reverse():
			if insertIndex >= len(s.items.items) {
				// Drop at the starting position of the last element and offset its own
				// extent.
				s.finalDropPosition = foundation.Some(s.itemOffsetAt(len(s.items.items) - 1).Sub(extentOffset(item.itemExtent, s.scrollDirection())))
			} else {
				// Drop at the end of the current element occupying the insert position.
				s.finalDropPosition = foundation.Some(s.itemOffsetAt(insertIndex).Add(extentOffset(s.itemExtentAt(insertIndex), s.scrollDirection())))
			}
		default:
			if insertIndex == 0 {
				// Drop at the starting position of the first element and offset its own
				// extent.
				s.finalDropPosition = foundation.Some(s.itemOffsetAt(0).Sub(extentOffset(item.itemExtent, s.scrollDirection())))
			} else {
				// Drop at the end of the previous element occupying the insert position.
				atIndex := insertIndex - 1
				s.finalDropPosition = foundation.Some(s.itemOffsetAt(atIndex).Add(extentOffset(s.itemExtentAt(atIndex), s.scrollDirection())))
			}
		}
	})
	if cb := s.Widget().opts.OnReorderEnd; cb != nil {
		cb(s.insertIndex.Value())
	}
}

func (s *SliverReorderableListState) dropCompleted() {
	oldIndex, newIndex := s.dragIndex.Value(), s.insertIndex.Value()
	s.handleReorderItem(oldIndex, newIndex)
	s.SetState(func() { s.dragReset() })
}

func (s *SliverReorderableListState) dragReset() {
	if s.dragInfo == nil {
		return
	}
	if idx, ok := s.dragIndex.Get(); ok && s.items.get(idx) != nil {
		dragItem := s.items.get(idx)
		dragItem.dragging = false
		dragItem.rebuild()
		s.dragIndex = foundation.None[int]()
	}
	s.dragInfo.dispose()
	s.dragInfo = nil
	if s.autoScroller != nil {
		s.autoScroller.StopAutoScroll()
	}
	s.resetItemGap()
	if s.recognizer != nil {
		s.recognizer.Dispose()
		s.recognizer = nil
	}
	if s.overlayEntry != nil {
		s.overlayEntry.Remove()
		s.overlayEntry.Dispose()
		s.overlayEntry = nil
	}
	s.finalDropPosition = foundation.None[ui.Offset]()
}

func (s *SliverReorderableListState) resetItemGap() {
	for _, item := range s.items.values() {
		item.resetGap()
	}
}

func (s *SliverReorderableListState) handleReorderItem(oldIndex, newIndex int) {
	w := s.Widget().opts
	if w.OnReorder != nil && oldIndex != newIndex {
		w.OnReorder(oldIndex, newIndex)
		return
	}
	// Removing an item at the old index shortens the list by one.
	if newIndex > oldIndex {
		newIndex--
	}
	if oldIndex != newIndex && w.OnReorderItem != nil {
		w.OnReorderItem(oldIndex, newIndex)
	}
}

func (s *SliverReorderableListState) handleScrollableAutoScrolled() {
	if s.dragInfo == nil {
		return
	}
	s.dragUpdateItems()
	// Continue scrolling if the drag is still in progress.
	if s.autoScroller != nil {
		s.autoScroller.StartAutoScrollIfNecessary(s.dragTargetRect())
	}
}

func (s *SliverReorderableListState) dragUpdateItems() {
	gapExtent := s.dragInfo.itemExtent
	proxyItemStart := offsetExtent(s.dragInfo.dragPosition.Sub(s.dragInfo.dragOffset), s.scrollDirection())
	proxyItemEnd := proxyItemStart + gapExtent
	reverse := s.reverse()
	dragIndex := s.dragIndex.Value()

	// Find the new index for inserting the item being dragged.
	newIndex := s.insertIndex.Value()
	for _, item := range s.items.values() {
		if (reverse && item.index() == dragIndex) || !item.Mounted() {
			continue
		}
		geometry := item.targetGeometry()
		itemStart, itemExtent := geometry.Left, geometry.Width()
		if s.scrollDirection() == painting.AxisVertical {
			itemStart, itemExtent = geometry.Top, geometry.Height()
		}
		itemEnd := itemStart + itemExtent
		itemMiddle := itemStart + itemExtent/2

		if reverse {
			if itemEnd >= proxyItemEnd && proxyItemEnd >= itemMiddle {
				// The start of the proxy is in the beginning half of the item, so the
				// item is swapped with the gap and the new index is found.
				newIndex = item.index()
				break
			} else if itemMiddle >= proxyItemStart && proxyItemStart >= itemStart {
				// The end of the proxy is in the ending half of the item.
				newIndex = item.index() + 1
				break
			} else if itemStart > proxyItemEnd && newIndex < item.index()+1 {
				newIndex = item.index() + 1
			} else if proxyItemStart > itemEnd && newIndex > item.index() {
				newIndex = item.index()
			}
		} else {
			if item.index() == dragIndex {
				// If the end of the proxy is not in the ending half of the item, it
				// is not processed, because it is the original dragged item.
				if itemMiddle <= proxyItemEnd && proxyItemEnd <= itemEnd {
					newIndex = dragIndex
				}
			} else if itemStart <= proxyItemStart && proxyItemStart <= itemMiddle {
				newIndex = item.index()
				break
			} else if itemMiddle <= proxyItemEnd && proxyItemEnd <= itemEnd {
				newIndex = item.index() + 1
				break
			} else if itemEnd < proxyItemStart && newIndex < item.index()+1 {
				newIndex = item.index() + 1
			} else if proxyItemEnd < itemStart && newIndex > item.index() {
				newIndex = item.index()
			}
		}
	}

	if newIndex != s.insertIndex.Value() {
		s.insertIndex = foundation.Some(newIndex)
		for _, item := range s.items.values() {
			if item.index() == dragIndex || !item.Mounted() {
				continue
			}
			item.updateForGap(dragIndex, newIndex, gapExtent, true, reverse)
		}
	}
}

func (s *SliverReorderableListState) dragTargetRect() ui.Rect {
	origin := s.dragInfo.dragPosition.Sub(s.dragInfo.dragOffset)
	return ui.NewRectFromLTWH(origin.Dx, origin.Dy, s.dragInfo.itemSize.Width, s.dragInfo.itemSize.Height)
}

func (s *SliverReorderableListState) itemOffsetAt(index int) ui.Offset {
	return s.items.get(index).targetGeometry().TopLeft()
}

func (s *SliverReorderableListState) itemExtentAt(index int) float64 {
	return sizeExtent(s.items.get(index).targetGeometry().Size(), s.scrollDirection())
}

func (s *SliverReorderableListState) itemBuilder(bc BuildContext, index int) Widget {
	w := s.Widget()
	if s.dragInfo != nil && index >= w.itemCount {
		if s.scrollDirection() == painting.AxisHorizontal {
			return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(s.dragInfo.itemExtent)})
		}
		return NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(s.dragInfo.itemExtent)})
	}
	child := w.itemBuilder(bc, index)
	if foundation.DebugMode && child.Key() == nil {
		panic(errors.New("ReorderableItemKeyMissing", "all list items must have a key", "index", index))
	}
	overlay := OverlayOf(bc, OverlayOfOpts{DebugRequiredFor: w})
	key := NewGlobalObjectKey[State](reorderableItemKeyValue{child.Key(), index, s})
	return &reorderableItem{
		StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: key}), index: index,
		capturedThemes: CaptureThemes(bc, overlay.Context()), child: s.wrapWithSemantics(bc, child, index),
	}
}

// reorderableItemKeyValue is what the global key of an item is made of: the key of
// the child, its index and the list. Unlike a GlobalObjectKey of a widget key, two
// keys are equal when these are equal.
//
// Flutter: widgets/reorderable_list.dart _ReorderableItemGlobalKey
type reorderableItemKeyValue struct {
	subKey foundation.Key
	index  int
	state  *SliverReorderableListState
}

func (s *SliverReorderableListState) wrapWithSemantics(bc BuildContext, child Widget, index int) Widget {
	// First, determine which semantics actions apply.
	actions := map[semantics.CustomSemanticsAction]func(){}
	itemCount := s.Widget().itemCount
	moveToStart := func() { s.handleReorderItem(index, 0) }
	moveToEnd := func() { s.handleReorderItem(index, itemCount) }
	moveBefore := func() { s.handleReorderItem(index, index-1) }
	// To move after, go to index+2 because it is moved to the space before
	// index+2, which is after the space at index+1.
	moveAfter := func() { s.handleReorderItem(index, index+2) }

	localizations := WidgetsLocalizationsOf(bc)
	isHorizontal := s.scrollDirection() == painting.AxisHorizontal
	ltr := DirectionalityOf(bc) == ui.TextDirectionLTR
	// If the item can move to before its current position in the list.
	if index > 0 {
		actions[semantics.NewCustomSemanticsAction(localizations.ReorderItemToStart())] = moveToStart
		reorderItemBefore := localizations.ReorderItemUp()
		if isHorizontal {
			if ltr {
				reorderItemBefore = localizations.ReorderItemLeft()
			} else {
				reorderItemBefore = localizations.ReorderItemRight()
			}
		}
		actions[semantics.NewCustomSemanticsAction(reorderItemBefore)] = moveBefore
	}
	// If the item can move to after its current position in the list.
	if index < itemCount-1 {
		reorderItemAfter := localizations.ReorderItemDown()
		if isHorizontal {
			if ltr {
				reorderItemAfter = localizations.ReorderItemRight()
			} else {
				reorderItemAfter = localizations.ReorderItemLeft()
			}
		}
		actions[semantics.NewCustomSemanticsAction(reorderItemAfter)] = moveAfter
		actions[semantics.NewCustomSemanticsAction(localizations.ReorderItemToEnd())] = moveToEnd
	}
	// Apply the relevant custom accessibility actions for moving the item up,
	// down, to the start, and to the end of the list.
	return NewSemantics(child, SemanticsOpts{Container: true, CustomSemanticsActions: actions})
}

// Build builds the list of the items.
func (s *SliverReorderableListState) Build(bc BuildContext) Widget {
	DebugCheckHasOverlay(bc)
	w := s.Widget()
	o := w.opts
	childrenDelegate := NewSliverChildBuilderDelegate(s.itemBuilder, SliverChildBuilderDelegateOpts{
		ChildCount: foundation.Some(w.itemCount), FindChildIndexCallback: o.FindChildIndexCallback})
	switch {
	case o.ItemExtent.IsSome():
		return NewSliverFixedExtentList(childrenDelegate, o.ItemExtent.Value())
	case o.ItemExtentBuilder != nil:
		return NewSliverVariedExtentList(childrenDelegate, o.ItemExtentBuilder)
	case o.PrototypeItem != nil:
		return NewSliverPrototypeExtentList(childrenDelegate, o.PrototypeItem)
	}
	return NewSliverList(childrenDelegate)
}

// reorderableItem is an item of a SliverReorderableList.
//
// Flutter: widgets/reorderable_list.dart _ReorderableItem
type reorderableItem struct {
	StatefulWidget
	index          int
	child          Widget
	capturedThemes *CapturedThemes
}

func (w *reorderableItem) CreateState() State { return &reorderableItemState{} }

// reorderableItemState is the state of a reorderableItem: it moves the item out
// of the way of the item that is dragged.
//
// Flutter: widgets/reorderable_list.dart _ReorderableItemState
type reorderableItemState struct {
	StateOf[*reorderableItem]
	listState       *SliverReorderableListState
	startOffset     ui.Offset
	targetOffset    ui.Offset
	offsetAnimation *animation.AnimationController
	dragging        bool
}

func (s *reorderableItemState) index() int { return s.Widget().index }

// SetDragging changes whether the item is the one that is dragged.
func (s *reorderableItemState) SetDragging(dragging bool) {
	if s.Mounted() {
		s.SetState(func() { s.dragging = dragging })
	}
}

func (s *reorderableItemState) InitState() {
	s.listState = SliverReorderableListOf(s.Context())
	s.listState.registerItem(s)
}

func (s *reorderableItemState) Dispose() {
	if s.offsetAnimation != nil {
		s.offsetAnimation.Dispose()
	}
	s.listState.unregisterItem(s.index(), s)
}

func (s *reorderableItemState) DidUpdateWidget(oldWidget *reorderableItem) {
	if oldWidget.index != s.index() {
		s.listState.unregisterItem(oldWidget.index, s)
		s.listState.registerItem(s)
	}
}

func (s *reorderableItemState) Build(BuildContext) Widget {
	if s.dragging {
		size := extentSize(s.listState.dragInfo.itemExtent, s.listState.scrollDirection())
		return NewSizedBoxFromSize(size, nil)
	}
	s.listState.registerItem(s)
	return NewTransformTranslate(s.offset(), s.Widget().child)
}

func (s *reorderableItemState) Deactivate() { s.listState.unregisterItem(s.index(), s) }

func (s *reorderableItemState) offset() ui.Offset {
	if s.offsetAnimation != nil {
		animValue := animation.Curves.EaseInOut.Transform(s.offsetAnimation.Value())
		return s.startOffset.Lerp(s.targetOffset, animValue)
	}
	return s.targetOffset
}

func (s *reorderableItemState) updateForGap(dragIndex, gapIndex int, gapExtent float64, animate, reverse bool) {
	// An offset needs to be added to create a gap when this item is between the
	// moving element (dragIndex) and the current gap position (gapIndex).
	index := s.index()
	var newTargetOffset ui.Offset
	switch {
	case gapIndex < dragIndex && index < dragIndex && index >= gapIndex:
		if reverse {
			newTargetOffset = extentOffset(-gapExtent, s.listState.scrollDirection())
		} else {
			newTargetOffset = extentOffset(gapExtent, s.listState.scrollDirection())
		}
	case gapIndex > dragIndex && index > dragIndex && index < gapIndex:
		if reverse {
			newTargetOffset = extentOffset(gapExtent, s.listState.scrollDirection())
		} else {
			newTargetOffset = extentOffset(-gapExtent, s.listState.scrollDirection())
		}
	}
	if newTargetOffset == s.targetOffset {
		return
	}
	previousTarget := s.targetOffset
	s.targetOffset = newTargetOffset
	if animate {
		if s.offsetAnimation == nil {
			c := animation.NewAnimationController(&s.listState.TickerProvider, animation.AnimationControllerOpts{Duration: foundation.Some(250 * time.Millisecond)})
			c.AddListener(s.rebuild)
			c.AddStatusListener(func(status animation.AnimationStatus) {
				if status.IsCompleted() {
					s.startOffset = s.targetOffset
					s.offsetAnimation.Dispose()
					s.offsetAnimation = nil
				}
			})
			s.offsetAnimation = c
			c.Forward()
		} else {
			// The animation is interrupted: the current position is calculated from
			// the previous animation.
			currentAnimValue := animation.Curves.EaseInOut.Transform(s.offsetAnimation.Value())
			s.startOffset = s.startOffset.Lerp(previousTarget, currentAnimValue)
			s.offsetAnimation.Forward(0)
		}
	} else {
		if s.offsetAnimation != nil {
			s.offsetAnimation.Dispose()
			s.offsetAnimation = nil
		}
		s.startOffset = s.targetOffset
	}
	s.rebuild()
}

func (s *reorderableItemState) resetGap() {
	if s.offsetAnimation != nil {
		s.offsetAnimation.Dispose()
		s.offsetAnimation = nil
	}
	s.startOffset = ui.Offset{}
	s.targetOffset = ui.Offset{}
	s.rebuild()
}

func (s *reorderableItemState) targetGeometry() ui.Rect {
	box := s.Context().FindRenderObject().(rendering.Box)
	itemPosition := box.LocalToGlobal(ui.OffsetZero, nil).Add(s.targetOffset)
	return itemPosition.And(box.Size())
}

func (s *reorderableItemState) rebuild() {
	if s.Mounted() {
		s.SetState(func() {})
	}
}

// ReorderableDragStartListener starts the drag of an item of a
// ReorderableList or SliverReorderableList as soon as the pointer is down.
//
// Flutter: widgets/reorderable_list.dart ReorderableDragStartListener
type ReorderableDragStartListener struct {
	StatelessWidget
	child   Widget
	index   int
	enabled bool
}

// ReorderableDragStartListenerOpts are the optional parameters of
// NewReorderableDragStartListener and NewReorderableDelayedDragStartListener.
//
// Flutter: widgets/reorderable_list.dart ReorderableDragStartListener.new
type ReorderableDragStartListenerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Enabled is whether the drag can start; the default is true.
	Enabled foundation.Opt[bool]
}

// NewReorderableDragStartListener creates a listener for the item at the index.
//
// Flutter: widgets/reorderable_list.dart ReorderableDragStartListener.new
func NewReorderableDragStartListener(index int, child Widget, opts ...ReorderableDragStartListenerOpts) *ReorderableDragStartListener {
	o := firstOpts(opts)
	return &ReorderableDragStartListener{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child, index, o.Enabled.OrElse(true)}
}

func startReorderDragging(bc BuildContext, index int, event *gestures.PointerDownEvent, create func(binding *gestures.GestureBinding) *gestures.MultiDragGestureRecognizer) {
	settings, _ := MediaQueryMaybeGestureSettingsOf(bc)
	list, ok := SliverReorderableListMaybeOf(bc)
	if !ok {
		return
	}
	recognizer := create(WidgetsBindingOf(bc).Gestures())
	if _, has := MediaQueryMaybeGestureSettingsOf(bc); has {
		recognizer.GestureSettings = foundation.Some(settings)
	}
	list.StartItemDragReorder(index, event, recognizer)
}

func reorderDragStartListenerBuild(bc BuildContext, child Widget, index int, enabled bool, create func(binding *gestures.GestureBinding) *gestures.MultiDragGestureRecognizer) Widget {
	var onDown rendering.PointerDownEventListener
	if enabled {
		onDown = func(event *gestures.PointerDownEvent) { startReorderDragging(bc, index, event, create) }
	}
	return NewListener(child, ListenerOpts{OnPointerDown: onDown})
}

// Build builds a listener of the pointer down.
func (w *ReorderableDragStartListener) Build(bc BuildContext) Widget {
	return reorderDragStartListenerBuild(bc, w.child, w.index, w.enabled, func(b *gestures.GestureBinding) *gestures.MultiDragGestureRecognizer {
		r := gestures.NewImmediateMultiDragGestureRecognizer(b, gestures.MultiDragGestureRecognizerOpts{DebugOwner: w})
		return &r.MultiDragGestureRecognizer
	})
}

// ReorderableDelayedDragStartListener starts the drag of an item of a
// ReorderableList or SliverReorderableList after a long press.
//
// Flutter: widgets/reorderable_list.dart ReorderableDelayedDragStartListener
type ReorderableDelayedDragStartListener struct {
	StatelessWidget
	child   Widget
	index   int
	enabled bool
}

// NewReorderableDelayedDragStartListener creates a listener for the item at the
// index.
//
// Flutter: widgets/reorderable_list.dart ReorderableDelayedDragStartListener.new
func NewReorderableDelayedDragStartListener(index int, child Widget, opts ...ReorderableDragStartListenerOpts) *ReorderableDelayedDragStartListener {
	o := firstOpts(opts)
	return &ReorderableDelayedDragStartListener{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child, index, o.Enabled.OrElse(true)}
}

// Build builds a listener of the pointer down.
func (w *ReorderableDelayedDragStartListener) Build(bc BuildContext) Widget {
	return reorderDragStartListenerBuild(bc, w.child, w.index, w.enabled, func(b *gestures.GestureBinding) *gestures.MultiDragGestureRecognizer {
		r := gestures.NewDelayedMultiDragGestureRecognizer(b, gestures.DelayedMultiDragGestureRecognizerOpts{
			MultiDragGestureRecognizerOpts: gestures.MultiDragGestureRecognizerOpts{DebugOwner: w}})
		return &r.MultiDragGestureRecognizer
	})
}

// reorderDragInfo is the drag of an item: the Drag that the recognizer reports
// to, and what the proxy of the item needs.
//
// Flutter: widgets/reorderable_list.dart _DragInfo
type reorderDragInfo struct {
	scrollDirection painting.Axis
	onUpdate        func(item *reorderDragInfo, position, delta ui.Offset)
	onEnd           func(item *reorderDragInfo)
	onCancel        func(item *reorderDragInfo)
	onDropCompleted func()
	proxyDecorator  ReorderItemProxyDecorator
	tickerProvider  *SliverReorderableListState

	boundary              DragBoundaryDelegate[ui.Rect]
	listState             *SliverReorderableListState
	index                 int
	child                 Widget
	dragPosition          ui.Offset
	dragOffset            ui.Offset
	itemSize              ui.Size
	itemLayoutConstraints rendering.BoxConstraints
	itemExtent            float64
	capturedThemes        *CapturedThemes
	scrollable            *ScrollableState
	proxyAnimation        *animation.AnimationController
	rawDragPosition       ui.Offset
}

func newReorderDragInfo(item *reorderableItemState, initialPosition ui.Offset, scrollDirection painting.Axis,
	onUpdate func(*reorderDragInfo, ui.Offset, ui.Offset), onCancel, onEnd func(*reorderDragInfo), onDropCompleted func(),
	proxyDecorator ReorderItemProxyDecorator, tickerProvider *SliverReorderableListState) *reorderDragInfo {
	d := &reorderDragInfo{scrollDirection: scrollDirection, onUpdate: onUpdate, onEnd: onEnd, onCancel: onCancel,
		onDropCompleted: onDropCompleted, proxyDecorator: proxyDecorator, tickerProvider: tickerProvider}
	itemRenderBox := item.Context().FindRenderObject().(rendering.Box)
	d.listState = item.listState
	d.index = item.index()
	d.child = item.Widget().child
	d.capturedThemes = item.Widget().capturedThemes
	d.dragOffset = itemRenderBox.GlobalToLocal(initialPosition, nil)
	d.itemSize, _ = item.Context().Size()
	d.rawDragPosition = initialPosition
	if provider := d.listState.Widget().opts.DragBoundaryProvider; provider != nil {
		d.boundary = provider(d.listState.Context())
	} else {
		d.boundary = DragBoundaryForRectMaybeOf(d.listState.Context())
	}
	d.dragPosition = d.adjustedDragOffset(initialPosition)
	d.itemExtent = sizeExtent(d.itemSize, scrollDirection)
	d.itemLayoutConstraints = itemRenderBox.BoxConstraints()
	d.scrollable = ScrollableMaybeOf(item.Context())
	return d
}

func (d *reorderDragInfo) dispose() {
	if d.proxyAnimation != nil {
		d.proxyAnimation.Dispose()
	}
}

func (d *reorderDragInfo) startDrag() {
	c := animation.NewAnimationController(&d.tickerProvider.TickerProvider, animation.AnimationControllerOpts{Duration: foundation.Some(250 * time.Millisecond)})
	c.AddStatusListener(func(status animation.AnimationStatus) {
		if status.IsDismissed() {
			d.dropCompleted()
		}
	})
	d.proxyAnimation = c
	c.Forward()
}

// Update moves the proxy.
func (d *reorderDragInfo) Update(details gestures.DragUpdateDetails) {
	delta := restrictAxis(details.Delta, d.scrollDirection)
	d.rawDragPosition = d.rawDragPosition.Add(delta)
	d.dragPosition = d.adjustedDragOffset(d.rawDragPosition)
	if d.onUpdate != nil {
		d.onUpdate(d, d.dragPosition, details.Delta)
	}
}

// End animates the proxy to the place where it is dropped.
func (d *reorderDragInfo) End(gestures.DragEndDetails) {
	d.proxyAnimation.Reverse()
	if d.onEnd != nil {
		d.onEnd(d)
	}
}

// Cancel cancels the drag.
func (d *reorderDragInfo) Cancel() {
	if d.proxyAnimation != nil {
		d.proxyAnimation.Dispose()
		d.proxyAnimation = nil
	}
	if d.onCancel != nil {
		d.onCancel(d)
	}
}

func (d *reorderDragInfo) adjustedDragOffset(offset ui.Offset) ui.Offset {
	if d.boundary == nil {
		return offset
	}
	return d.boundary.NearestPositionWithinBoundary(offset.Sub(d.dragOffset).And(d.itemSize)).Shift(d.dragOffset).TopLeft()
}

func (d *reorderDragInfo) dropCompleted() {
	if d.proxyAnimation != nil {
		d.proxyAnimation.Dispose()
		d.proxyAnimation = nil
	}
	if d.onDropCompleted != nil {
		d.onDropCompleted()
	}
}

func (d *reorderDragInfo) createProxy(bc BuildContext) Widget {
	return d.capturedThemes.Wrap(&dragItemProxy{
		StatelessWidget: NewStatelessWidgetBase(), listState: d.listState, index: d.index, size: d.itemSize, constraints: d.itemLayoutConstraints,
		animation: d.proxyAnimation, position: d.dragPosition.Sub(d.dragOffset).Sub(overlayOrigin(bc)), proxyDecorator: d.proxyDecorator, child: d.child,
	})
}

func overlayOrigin(bc BuildContext) ui.Offset {
	overlay := OverlayOf(bc, OverlayOfOpts{DebugRequiredFor: bc.Widget()})
	overlayBox := overlay.Context().FindRenderObject().(rendering.Box)
	return overlayBox.LocalToGlobal(ui.OffsetZero, nil)
}

// dragItemProxy is the item that follows the pointer, in the overlay.
//
// Flutter: widgets/reorderable_list.dart _DragItemProxy
type dragItemProxy struct {
	StatelessWidget
	listState      *SliverReorderableListState
	index          int
	child          Widget
	position       ui.Offset
	size           ui.Size
	constraints    rendering.BoxConstraints
	animation      *animation.AnimationController
	proxyDecorator ReorderItemProxyDecorator
}

func (w *dragItemProxy) Build(bc BuildContext) Widget {
	proxyChild := w.child
	if w.proxyDecorator != nil {
		proxyChild = w.proxyDecorator(w.child, w.index, w.animation.View())
	}
	origin := overlayOrigin(bc)
	return NewMediaQuery(MediaQueryOf(bc).RemovePadding(RemoveSides{Top: true}),
		NewAnimatedBuilder(w.animation, func(bc BuildContext, child Widget) Widget {
			effectivePosition := w.position
			if dropPosition, ok := w.listState.finalDropPosition.Get(); ok {
				effectivePosition = dropPosition.Sub(origin).Lerp(effectivePosition, animation.Curves.EaseOut.Transform(w.animation.Value()))
			}
			alignment := painting.AlignmentTopCenter
			if w.listState.scrollDirection() == painting.AxisHorizontal {
				alignment = painting.AlignmentCenterLeft
			}
			return NewPositioned(
				NewSizedBox(NewOverflowBox(child, OverflowBoxOpts{
					MinWidth: foundation.Some(w.constraints.MinWidth), MinHeight: foundation.Some(w.constraints.MinHeight),
					MaxWidth: foundation.Some(w.constraints.MaxWidth), MaxHeight: foundation.Some(w.constraints.MaxHeight), Alignment: alignment,
				}), SizedBoxOpts{Width: foundation.Some(w.size.Width), Height: foundation.Some(w.size.Height)}),
				PositionedOpts{Left: foundation.Some(effectivePosition.Dx), Top: foundation.Some(effectivePosition.Dy)})
		}, proxyChild))
}

func extentSize(extent float64, scrollDirection painting.Axis) ui.Size {
	if scrollDirection == painting.AxisHorizontal {
		return ui.Size{Width: extent}
	}
	return ui.Size{Height: extent}
}

func extentOffset(extent float64, scrollDirection painting.Axis) ui.Offset {
	if scrollDirection == painting.AxisHorizontal {
		return ui.Offset{Dx: extent}
	}
	return ui.Offset{Dy: extent}
}

func restrictAxis(offset ui.Offset, scrollDirection painting.Axis) ui.Offset {
	if scrollDirection == painting.AxisHorizontal {
		return ui.Offset{Dx: offset.Dx}
	}
	return ui.Offset{Dy: offset.Dy}
}
