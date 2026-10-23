package widgets

import (
	"sort"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// AnimatedItemBuilder builds an item of an animated list or grid, which it
// animates with the animation while the item is inserted.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedItemBuilder
type AnimatedItemBuilder func(bc BuildContext, index int, animation animation.Animation[float64]) Widget

// AnimatedRemovedItemBuilder builds an item that is being removed from an
// animated list or grid, which it animates with the animation, from 1 to 0.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedRemovedItemBuilder
type AnimatedRemovedItemBuilder func(bc BuildContext, animation animation.Animation[float64]) Widget

// animatedScrollDuration is the default duration of the insert and remove
// animations.
const animatedScrollDuration = 300 * time.Millisecond

// AnimatedItemOpts are the optional parameters of the methods that insert and
// remove items of an animated list or grid.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedListState.insertItem
type AnimatedItemOpts struct {
	// Duration is the duration of the animation; the default is 300ms.
	Duration foundation.Opt[time.Duration]
}

func (o AnimatedItemOpts) duration() time.Duration { return o.Duration.OrElse(animatedScrollDuration) }

// AnimatedScrollViewOpts are the optional parameters of NewAnimatedList,
// NewAnimatedListSeparated and NewAnimatedGrid.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedList.new
type AnimatedScrollViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// InitialItemCount is the number of items that the view starts with.
	InitialItemCount int
	// ScrollDirection is the axis along which the view scrolls; the default is
	// AxisVertical.
	ScrollDirection painting.Axis
	// Reverse is whether the view scrolls against the reading direction.
	Reverse bool
	// Controller reads and controls the scroll position.
	Controller ScrollController
	// Primary is whether this is the primary scroll view.
	Primary foundation.Opt[bool]
	// Physics is how the view responds to the user.
	Physics ScrollPhysics
	// ShrinkWrap is whether the extent of the view is the extent of its content.
	// It does not apply to AnimatedGrid.
	ShrinkWrap bool
	// Padding is the amount of space by which to inset the items; nil lets the
	// MediaQuery decide.
	Padding painting.EdgeInsetsGeometry
	// ClipBehavior is how the view clips its content; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// ScrollCacheExtent is how far beyond the viewport the view builds items.
	ScrollCacheExtent foundation.Opt[rendering.ScrollCacheExtent]
}

// animatedScrollView is what AnimatedList and AnimatedGrid share.
//
// Flutter: widgets/animated_scroll_view.dart _AnimatedScrollView
type animatedScrollView struct {
	itemBuilder             AnimatedItemBuilder
	removedSeparatorBuilder AnimatedItemBuilder
	opts                    AnimatedScrollViewOpts
}

func newAnimatedScrollView(itemBuilder AnimatedItemBuilder, o AnimatedScrollViewOpts) animatedScrollView {
	if foundation.DebugMode && o.InitialItemCount < 0 {
		panic(errors.New(tagAnimatedItemCountNegative, "the initialItemCount must not be negative", "initialItemCount", o.InitialItemCount))
	}
	if o.ScrollDirection == painting.AxisUnspecified {
		o.ScrollDirection = painting.AxisVertical
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return animatedScrollView{itemBuilder: itemBuilder, opts: o}
}

// AnimatedList is a scrolling container that animates items when they are
// inserted or removed.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedList
type AnimatedList struct {
	StatefulWidget
	animatedScrollView
}

// NewAnimatedList creates an animated list whose items the builder builds.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedList.new
func NewAnimatedList(itemBuilder AnimatedItemBuilder, opts ...AnimatedScrollViewOpts) *AnimatedList {
	o := firstOpts(opts)
	return &AnimatedList{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), newAnimatedScrollView(itemBuilder, o)}
}

// NewAnimatedListSeparated creates an animated list that has a separator between
// its items. The initial item count is the number of items, without the
// separators, and the removedSeparatorBuilder builds a separator that is
// removed.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedList.separated
func NewAnimatedListSeparated(itemBuilder, separatorBuilder, removedSeparatorBuilder AnimatedItemBuilder, opts ...AnimatedScrollViewOpts) *AnimatedList {
	o := firstOpts(opts)
	if foundation.DebugMode && removedSeparatorBuilder == nil {
		panic(errors.New(tagAnimatedRemovedSeparatorBuilderNil, "the removedSeparatorBuilder is required"))
	}
	if o.InitialItemCount > 0 {
		o.InitialItemCount = o.InitialItemCount*2 - 1
	}
	v := newAnimatedScrollView(func(bc BuildContext, index int, a animation.Animation[float64]) Widget {
		itemIndex := index / 2
		if index%2 == 0 {
			return itemBuilder(bc, itemIndex, a)
		}
		return separatorBuilder(bc, itemIndex, a)
	}, o)
	v.removedSeparatorBuilder = removedSeparatorBuilder
	return &AnimatedList{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), v}
}

// CreateState creates the state.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedList.createState
func (w *AnimatedList) CreateState() State {
	return &AnimatedListState{animatedScrollViewState: animatedScrollViewState{key: NewGlobalKey[animatedMultiBoxHost]()}}
}

// AnimatedListMaybeOf is the state of the closest AnimatedList above the
// context, and false if there is none.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedList.maybeOf
func AnimatedListMaybeOf(bc BuildContext) (*AnimatedListState, bool) {
	return FindAncestorStateOfType[*AnimatedListState](bc)
}

// AnimatedListOf is the state of the closest AnimatedList above the context. In
// debug builds it panics with the tag AnimatedListMissing when there is none.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedList.of
func AnimatedListOf(bc BuildContext) *AnimatedListState {
	s, ok := AnimatedListMaybeOf(bc)
	if !ok && foundation.DebugMode {
		panic(errors.New(tagAnimatedListMissing, "AnimatedListOf() called with a context that does not contain an AnimatedList; this can happen when the context provided is from the same StatefulWidget that built the AnimatedList",
			"context", bc.(Element).String()))
	}
	return s
}

// AnimatedListState is the state of an AnimatedList: it inserts and removes
// items with animations.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedListState
type AnimatedListState struct {
	StateOf[*AnimatedList]
	animatedScrollViewState
}

// Build builds the scroll view that has the sliver.
func (s *AnimatedListState) Build(bc BuildContext) Widget {
	w := s.Widget()
	return s.wrap(bc, &w.animatedScrollView, NewSliverAnimatedList(w.itemBuilder, SliverAnimatedListOpts{Key: s.key, InitialItemCount: w.opts.InitialItemCount}))
}

// AnimatedGrid is a scrolling container that animates items when they are
// inserted or removed.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedGrid
type AnimatedGrid struct {
	StatefulWidget
	animatedScrollView
	gridDelegate rendering.SliverGridDelegate
}

// NewAnimatedGrid creates an animated grid whose items the builder builds and
// the delegate lays out.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedGrid.new
func NewAnimatedGrid(itemBuilder AnimatedItemBuilder, gridDelegate rendering.SliverGridDelegate, opts ...AnimatedScrollViewOpts) *AnimatedGrid {
	o := firstOpts(opts)
	return &AnimatedGrid{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), newAnimatedScrollView(itemBuilder, o), gridDelegate}
}

// GridDelegate is the delegate that lays the items out.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedGrid.gridDelegate
func (w *AnimatedGrid) GridDelegate() rendering.SliverGridDelegate { return w.gridDelegate }

// CreateState creates the state.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedGrid.createState
func (w *AnimatedGrid) CreateState() State {
	return &AnimatedGridState{animatedScrollViewState: animatedScrollViewState{key: NewGlobalKey[animatedMultiBoxHost]()}}
}

// AnimatedGridMaybeOf is the state of the closest AnimatedGrid above the
// context, and false if there is none.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedGrid.maybeOf
func AnimatedGridMaybeOf(bc BuildContext) (*AnimatedGridState, bool) {
	return FindAncestorStateOfType[*AnimatedGridState](bc)
}

// AnimatedGridOf is the state of the closest AnimatedGrid above the context. In
// debug builds it panics with the tag AnimatedGridMissing when there is none.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedGrid.of
func AnimatedGridOf(bc BuildContext) *AnimatedGridState {
	s, ok := AnimatedGridMaybeOf(bc)
	if !ok && foundation.DebugMode {
		panic(errors.New(tagAnimatedGridMissing, "AnimatedGridOf() called with a context that does not contain an AnimatedGrid; this can happen when the context provided is from the same StatefulWidget that built the AnimatedGrid",
			"context", bc.(Element).String()))
	}
	return s
}

// AnimatedGridState is the state of an AnimatedGrid: it inserts and removes
// items with animations.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedGridState
type AnimatedGridState struct {
	StateOf[*AnimatedGrid]
	animatedScrollViewState
}

// Build builds the scroll view that has the sliver.
func (s *AnimatedGridState) Build(bc BuildContext) Widget {
	w := s.Widget()
	return s.wrap(bc, &w.animatedScrollView, NewSliverAnimatedGrid(w.itemBuilder, w.gridDelegate, SliverAnimatedListOpts{Key: s.key, InitialItemCount: w.opts.InitialItemCount}))
}

// animatedScrollViewState is the part of AnimatedListState and AnimatedGridState
// that is the same: it forwards to the state of the sliver, and handles the
// separators.
//
// Flutter: widgets/animated_scroll_view.dart _AnimatedScrollViewState
type animatedScrollViewState struct {
	key *GlobalKey[animatedMultiBoxHost]
	// config is set on each build.
	config *animatedScrollView
}

func (s *animatedScrollViewState) sliver() *animatedMultiBox {
	st, ok := s.key.CurrentState()
	if !ok {
		panic(errors.New(tagAnimatedScrollViewNotBuilt, "the animated list or grid has not been built yet"))
	}
	return st.multiBox()
}

// InsertItem inserts an item at the index, and animates it in. The index is the
// one the item has after the insertion.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedListState.insertItem
func (s *animatedScrollViewState) InsertItem(index int, opts ...AnimatedItemOpts) {
	duration := firstOpts(opts).duration()
	box := s.sliver()
	if s.config.removedSeparatorBuilder == nil {
		box.InsertItem(index, AnimatedItemOpts{Duration: foundation.Some(duration)})
		return
	}
	itemIndex := s.computeItemIndex(index)
	box.InsertItem(itemIndex, AnimatedItemOpts{Duration: foundation.Some(duration)})
	if box.itemsCount > 1 {
		// Because InsertItem moves the items after the index, the separator is
		// inserted at the same index as the item. With one item there is no
		// separator.
		box.InsertItem(itemIndex, AnimatedItemOpts{Duration: foundation.Some(duration)})
	}
}

// InsertAllItems inserts length items from the index, and animates them in.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedListState.insertAllItems
func (s *animatedScrollViewState) InsertAllItems(index, length int, opts ...AnimatedItemOpts) {
	duration := firstOpts(opts).duration()
	box := s.sliver()
	if s.config.removedSeparatorBuilder == nil {
		box.InsertAllItems(index, length, AnimatedItemOpts{Duration: foundation.Some(duration)})
		return
	}
	itemIndex := s.computeItemIndex(index)
	lengthWithSeparators := length * 2
	if box.itemsCount == 0 {
		lengthWithSeparators = length*2 - 1
	}
	box.InsertAllItems(itemIndex, lengthWithSeparators, AnimatedItemOpts{Duration: foundation.Some(duration)})
}

// RemoveItem removes the item at the index, and animates it out with the item
// that the builder builds.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedListState.removeItem
func (s *animatedScrollViewState) RemoveItem(index int, builder AnimatedRemovedItemBuilder, opts ...AnimatedItemOpts) {
	duration := firstOpts(opts).duration()
	box := s.sliver()
	removedSeparatorBuilder := s.config.removedSeparatorBuilder
	if removedSeparatorBuilder == nil {
		box.RemoveItem(index, builder, AnimatedItemOpts{Duration: foundation.Some(duration)})
		return
	}
	itemIndex := s.computeItemIndex(index)
	// Children animating out from earlier removals still count towards the
	// count, so subtract them to recognize the new tail of the visible list.
	visibleItemsCount := box.itemsCount - len(box.outgoing)
	box.RemoveItem(itemIndex, builder, AnimatedItemOpts{Duration: foundation.Some(duration)})
	if visibleItemsCount > 1 {
		if itemIndex == visibleItemsCount-1 {
			// The item was at the end, so the separator to remove is the one before.
			box.RemoveItem(itemIndex-1, toRemovedItemBuilder(removedSeparatorBuilder, index-1), AnimatedItemOpts{Duration: foundation.Some(duration)})
		} else {
			// The separator took the place of the item.
			box.RemoveItem(itemIndex, toRemovedItemBuilder(removedSeparatorBuilder, index), AnimatedItemOpts{Duration: foundation.Some(duration)})
		}
	}
}

// RemoveAllItems removes all the items, and animates them out with the item
// that the builder builds.
//
// Flutter: widgets/animated_scroll_view.dart AnimatedListState.removeAllItems
func (s *animatedScrollViewState) RemoveAllItems(builder AnimatedRemovedItemBuilder, opts ...AnimatedItemOpts) {
	duration := firstOpts(opts).duration()
	box := s.sliver()
	removedSeparatorBuilder := s.config.removedSeparatorBuilder
	if removedSeparatorBuilder == nil {
		box.RemoveAllItems(builder, AnimatedItemOpts{Duration: foundation.Some(duration)})
		return
	}
	for index := box.itemsCount - 1; index >= 0; index-- {
		if index%2 == 0 {
			box.RemoveItem(index, builder, AnimatedItemOpts{Duration: foundation.Some(duration)})
		} else {
			box.RemoveItem(index, toRemovedItemBuilder(removedSeparatorBuilder, index/2), AnimatedItemOpts{Duration: foundation.Some(duration)})
		}
	}
}

// computeItemIndex is the index in the sliver for the item, which has the
// separators between the items.
func (s *animatedScrollViewState) computeItemIndex(index int) int {
	if index == 0 {
		return index
	}
	count := s.sliver().itemsCount
	separatorsCount := count / 2
	separatedItemsCount := count - separatorsCount
	indexAdjusted := index * 2
	if index == separatedItemsCount {
		return indexAdjusted - 1
	}
	return indexAdjusted
}

func toRemovedItemBuilder(builder AnimatedItemBuilder, index int) AnimatedRemovedItemBuilder {
	return func(bc BuildContext, a animation.Animation[float64]) Widget { return builder(bc, index, a) }
}

// wrap puts the sliver in a CustomScrollView, with the padding.
func (s *animatedScrollViewState) wrap(bc BuildContext, v *animatedScrollView, sliver Widget) Widget {
	s.config = v
	o := v.opts
	effectivePadding := o.Padding
	if o.Padding == nil {
		if mediaQuery, ok := MediaQueryMaybeOf(bc); ok {
			zero := foundation.Some(0.0)
			horizontal := mediaQuery.Padding.CopyWith(painting.EdgeInsetsCopy{Top: zero, Bottom: zero})
			vertical := mediaQuery.Padding.CopyWith(painting.EdgeInsetsCopy{Left: zero, Right: zero})
			if o.ScrollDirection == painting.AxisVertical {
				effectivePadding = vertical
				sliver = NewMediaQuery(mediaQuery.CopyWith(MediaQueryDataCopyOpts{Padding: foundation.Some(horizontal)}), sliver)
			} else {
				effectivePadding = horizontal
				sliver = NewMediaQuery(mediaQuery.CopyWith(MediaQueryDataCopyOpts{Padding: foundation.Some(vertical)}), sliver)
			}
		}
	}
	if effectivePadding != nil {
		sliver = NewSliverPadding(effectivePadding, sliver)
	}
	return NewCustomScrollView([]Widget{sliver}, CustomScrollViewOpts{
		ScrollDirection: o.ScrollDirection, Reverse: o.Reverse, Controller: o.Controller, Primary: o.Primary, Physics: o.Physics,
		ClipBehavior: o.ClipBehavior, ShrinkWrap: o.ShrinkWrap, ScrollCacheExtent: o.ScrollCacheExtent,
	})
}

// activeItem is an incoming or outgoing item.
//
// Flutter: widgets/animated_scroll_view.dart _ActiveItem
type activeItem struct {
	controller         *animation.AnimationController
	removedItemBuilder AnimatedRemovedItemBuilder
	itemIndex          int
}

// SliverAnimatedListOpts are the optional parameters of NewSliverAnimatedList and
// NewSliverAnimatedGrid.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedList.new
type SliverAnimatedListOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// FindChildIndexCallback finds the index of the item that has a key, so that
	// items that move keep their state.
	FindChildIndexCallback ChildIndexGetter
	// InitialItemCount is the number of items that the sliver starts with.
	InitialItemCount int
}

// animatedMultiBoxHost is the state of a SliverAnimatedList or SliverAnimatedGrid.
type animatedMultiBoxHost interface {
	State
	multiBox() *animatedMultiBox
}

// sliverAnimatedMultiBox is the widget part that SliverAnimatedList and
// SliverAnimatedGrid share.
//
// Flutter: widgets/animated_scroll_view.dart _SliverAnimatedMultiBoxAdaptor
type sliverAnimatedMultiBox struct {
	itemBuilder            AnimatedItemBuilder
	findChildIndexCallback ChildIndexGetter
	initialItemCount       int
}

func newSliverAnimatedMultiBox(itemBuilder AnimatedItemBuilder, o SliverAnimatedListOpts) sliverAnimatedMultiBox {
	if foundation.DebugMode && o.InitialItemCount < 0 {
		panic(errors.New(tagAnimatedItemCountNegative, "the initialItemCount must not be negative", "initialItemCount", o.InitialItemCount))
	}
	return sliverAnimatedMultiBox{itemBuilder, o.FindChildIndexCallback, o.InitialItemCount}
}

// SliverAnimatedList is a sliver that animates items when they are inserted or
// removed.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedList
type SliverAnimatedList struct {
	StatefulWidget
	sliverAnimatedMultiBox
}

// NewSliverAnimatedList creates a sliver list whose items the builder builds.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedList.new
func NewSliverAnimatedList(itemBuilder AnimatedItemBuilder, opts ...SliverAnimatedListOpts) *SliverAnimatedList {
	o := firstOpts(opts)
	return &SliverAnimatedList{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), newSliverAnimatedMultiBox(itemBuilder, o)}
}

// CreateState creates the state.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedList.createState
func (w *SliverAnimatedList) CreateState() State { return &SliverAnimatedListState{} }

// SliverAnimatedListMaybeOf is the state of the closest SliverAnimatedList above
// the context, and false if there is none.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedList.maybeOf
func SliverAnimatedListMaybeOf(bc BuildContext) (*SliverAnimatedListState, bool) {
	return FindAncestorStateOfType[*SliverAnimatedListState](bc)
}

// SliverAnimatedListOf is the state of the closest SliverAnimatedList above the
// context. In debug builds it panics with the tag SliverAnimatedListMissing when
// there is none.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedList.of
func SliverAnimatedListOf(bc BuildContext) *SliverAnimatedListState {
	s, ok := SliverAnimatedListMaybeOf(bc)
	if !ok && foundation.DebugMode {
		panic(errors.New(tagSliverAnimatedListMissing, "SliverAnimatedListOf() called with a context that does not contain a SliverAnimatedList; this can happen when the context provided is from the same StatefulWidget that built the AnimatedList",
			"context", bc.(Element).String()))
	}
	return s
}

// SliverAnimatedListState is the state of a SliverAnimatedList.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedListState
type SliverAnimatedListState struct {
	StateOf[*SliverAnimatedList]
	TickerProvider
	animatedMultiBox
}

func (s *SliverAnimatedListState) multiBox() *animatedMultiBox { return &s.animatedMultiBox }

// InitState sets up the items.
func (s *SliverAnimatedListState) InitState() {
	w := s.Widget()
	s.initBox(s, &s.TickerProvider, w.itemBuilder, w.initialItemCount)
}

// Dispose disposes the controllers of the animations.
func (s *SliverAnimatedListState) Dispose() { s.disposeBox() }

// Build builds the sliver list.
func (s *SliverAnimatedListState) Build(BuildContext) Widget {
	return NewSliverList(s.createDelegate(s.Widget().findChildIndexCallback))
}

// SliverAnimatedGrid is a sliver that animates items when they are inserted or
// removed.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedGrid
type SliverAnimatedGrid struct {
	StatefulWidget
	sliverAnimatedMultiBox
	gridDelegate rendering.SliverGridDelegate
}

// NewSliverAnimatedGrid creates a sliver grid whose items the builder builds and
// the delegate lays out.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedGrid.new
func NewSliverAnimatedGrid(itemBuilder AnimatedItemBuilder, gridDelegate rendering.SliverGridDelegate, opts ...SliverAnimatedListOpts) *SliverAnimatedGrid {
	o := firstOpts(opts)
	return &SliverAnimatedGrid{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), newSliverAnimatedMultiBox(itemBuilder, o), gridDelegate}
}

// GridDelegate is the delegate that lays the items out.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedGrid.gridDelegate
func (w *SliverAnimatedGrid) GridDelegate() rendering.SliverGridDelegate { return w.gridDelegate }

// CreateState creates the state.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedGrid.createState
func (w *SliverAnimatedGrid) CreateState() State { return &SliverAnimatedGridState{} }

// SliverAnimatedGridMaybeOf is the state of the closest SliverAnimatedGrid above
// the context, and false if there is none.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedGrid.maybeOf
func SliverAnimatedGridMaybeOf(bc BuildContext) (*SliverAnimatedGridState, bool) {
	return FindAncestorStateOfType[*SliverAnimatedGridState](bc)
}

// SliverAnimatedGridOf is the state of the closest SliverAnimatedGrid above the
// context. In debug builds it panics with the tag SliverAnimatedGridMissing when
// there is none.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedGrid.of
func SliverAnimatedGridOf(bc BuildContext) *SliverAnimatedGridState {
	s, ok := SliverAnimatedGridMaybeOf(bc)
	if !ok && foundation.DebugMode {
		panic(errors.New(tagSliverAnimatedGridMissing, "SliverAnimatedGridOf() called with a context that does not contain a SliverAnimatedGrid; this can happen when the context provided is from the same StatefulWidget that built the AnimatedGrid",
			"context", bc.(Element).String()))
	}
	return s
}

// SliverAnimatedGridState is the state of a SliverAnimatedGrid.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedGridState
type SliverAnimatedGridState struct {
	StateOf[*SliverAnimatedGrid]
	TickerProvider
	animatedMultiBox
}

func (s *SliverAnimatedGridState) multiBox() *animatedMultiBox { return &s.animatedMultiBox }

// InitState sets up the items.
func (s *SliverAnimatedGridState) InitState() {
	w := s.Widget()
	s.initBox(s, &s.TickerProvider, w.itemBuilder, w.initialItemCount)
}

// Dispose disposes the controllers of the animations.
func (s *SliverAnimatedGridState) Dispose() { s.disposeBox() }

// Build builds the sliver grid.
func (s *SliverAnimatedGridState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewSliverGrid(s.createDelegate(w.findChildIndexCallback), w.gridDelegate)
}

// animatedMultiBox is what the states of SliverAnimatedList and
// SliverAnimatedGrid share: the incoming and outgoing items.
//
// Flutter: widgets/animated_scroll_view.dart _SliverAnimatedMultiBoxAdaptorState
type animatedMultiBox struct {
	host       interface{ SetState(func()) }
	vsync      *TickerProvider
	builder    AnimatedItemBuilder
	incoming   []*activeItem
	outgoing   []*activeItem
	itemsCount int
}

func (b *animatedMultiBox) initBox(host interface{ SetState(func()) }, vsync *TickerProvider, builder AnimatedItemBuilder, count int) {
	b.builder = builder
	b.host = host
	b.vsync = vsync
	b.itemsCount = count
}

func (b *animatedMultiBox) disposeBox() {
	for _, item := range append(append([]*activeItem{}, b.incoming...), b.outgoing...) {
		item.controller.Dispose()
	}
}

// findActive returns the position of the item with the index in the sorted
// items, or -1.
func findActive(items []*activeItem, itemIndex int) int {
	i := sort.Search(len(items), func(i int) bool { return items[i].itemIndex >= itemIndex })
	if i < len(items) && items[i].itemIndex == itemIndex {
		return i
	}
	return -1
}

func removeActiveItemAt(items *[]*activeItem, itemIndex int) *activeItem {
	i := findActive(*items, itemIndex)
	if i == -1 {
		return nil
	}
	item := (*items)[i]
	*items = append((*items)[:i], (*items)[i+1:]...)
	return item
}

func activeItemAt(items []*activeItem, itemIndex int) *activeItem {
	if i := findActive(items, itemIndex); i != -1 {
		return items[i]
	}
	return nil
}

func sortActive(items []*activeItem) {
	sort.SliceStable(items, func(a, b int) bool { return items[a].itemIndex < items[b].itemIndex })
}

// indexToItemIndex converts an index, as the caller sees the list when the
// removed items are gone, to the index of the item in the sliver.
func (b *animatedMultiBox) indexToItemIndex(index int) int {
	itemIndex := index
	for _, item := range b.outgoing {
		if item.itemIndex <= itemIndex {
			itemIndex++
		} else {
			break
		}
	}
	return itemIndex
}

func (b *animatedMultiBox) itemIndexToIndex(itemIndex int) int {
	index := itemIndex
	for _, item := range b.outgoing {
		if foundation.DebugMode && item.itemIndex == itemIndex {
			panic(errors.New(tagAnimatedItemOutgoing, "the item is being removed", "itemIndex", itemIndex))
		}
		if item.itemIndex < itemIndex {
			index--
		} else {
			break
		}
	}
	return index
}

func (b *animatedMultiBox) createDelegate(findChildIndex ChildIndexGetter) SliverChildDelegate {
	var find ChildIndexGetter
	if findChildIndex != nil {
		find = func(key foundation.Key) foundation.Opt[int] {
			if index := findChildIndex(key); index.IsSome() {
				return foundation.Some(b.indexToItemIndex(index.Value()))
			}
			return foundation.None[int]()
		}
	}
	return NewSliverChildBuilderDelegate(b.itemBuilderAt, SliverChildBuilderDelegateOpts{
		ChildCount: foundation.Some(b.itemsCount), FindChildIndexCallback: find,
	})
}

// itemBuilderAt builds the item at the index in the sliver.
func (b *animatedMultiBox) itemBuilderAt(bc BuildContext, itemIndex int) Widget {
	if outgoing := activeItemAt(b.outgoing, itemIndex); outgoing != nil {
		return outgoing.removedItemBuilder(bc, outgoing.controller.View())
	}
	anim := animation.AlwaysComplete
	if incoming := activeItemAt(b.incoming, itemIndex); incoming != nil {
		anim = incoming.controller.View()
	}
	return b.builder(bc, b.itemIndexToIndex(itemIndex), anim)
}

// InsertItem inserts an item at the index, and animates it in. The index is the
// one the item has after the insertion.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedListState.insertItem
func (b *animatedMultiBox) InsertItem(index int, opts ...AnimatedItemOpts) {
	duration := firstOpts(opts).duration()
	itemIndex := b.indexToItemIndex(index)
	if foundation.DebugMode && (index < 0 || itemIndex < 0 || itemIndex > b.itemsCount) {
		panic(errors.New(tagAnimatedItemIndexInvalid, "the index to insert is out of range", "index", index, "itemsCount", b.itemsCount))
	}
	// Increment the incoming and outgoing item indices to account for the
	// insertion.
	for _, item := range b.incoming {
		if item.itemIndex >= itemIndex {
			item.itemIndex++
		}
	}
	for _, item := range b.outgoing {
		if item.itemIndex >= itemIndex {
			item.itemIndex++
		}
	}
	controller := animation.NewAnimationController(b.vsync, animation.AnimationControllerOpts{Duration: foundation.Some(duration)})
	incomingItem := &activeItem{controller: controller, itemIndex: itemIndex}
	b.host.SetState(func() {
		b.incoming = append(b.incoming, incomingItem)
		sortActive(b.incoming)
		b.itemsCount++
	})
	controller.Forward().Then(func() {
		removeActiveItemAt(&b.incoming, incomingItem.itemIndex).controller.Dispose()
	})
}

// InsertAllItems inserts length items from the index, and animates them in.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedListState.insertAllItems
func (b *animatedMultiBox) InsertAllItems(index, length int, opts ...AnimatedItemOpts) {
	for i := 0; i < length; i++ {
		b.InsertItem(index+i, opts...)
	}
}

// RemoveItem removes the item at the index, and animates it out with the item
// that the builder builds.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedListState.removeItem
func (b *animatedMultiBox) RemoveItem(index int, builder AnimatedRemovedItemBuilder, opts ...AnimatedItemOpts) {
	duration := firstOpts(opts).duration()
	itemIndex := b.indexToItemIndex(index)
	if foundation.DebugMode {
		if index < 0 || itemIndex < 0 || itemIndex >= b.itemsCount {
			panic(errors.New(tagAnimatedItemIndexInvalid, "the index to remove is out of range", "index", index, "itemsCount", b.itemsCount))
		}
		if activeItemAt(b.outgoing, itemIndex) != nil {
			panic(errors.New(tagAnimatedItemRemovedTwice, "the item is already being removed", "index", index))
		}
	}
	var controller *animation.AnimationController
	if incomingItem := removeActiveItemAt(&b.incoming, itemIndex); incomingItem != nil {
		controller = incomingItem.controller
	} else {
		controller = animation.NewAnimationController(b.vsync, animation.AnimationControllerOpts{Duration: foundation.Some(duration), Value: foundation.Some(1.0)})
	}
	outgoingItem := &activeItem{controller: controller, itemIndex: itemIndex, removedItemBuilder: builder}
	b.host.SetState(func() {
		b.outgoing = append(b.outgoing, outgoingItem)
		sortActive(b.outgoing)
	})
	controller.Reverse().Then(func() {
		removeActiveItemAt(&b.outgoing, outgoingItem.itemIndex).controller.Dispose()
		// Decrement the incoming and outgoing item indices to account for the
		// removal.
		for _, item := range b.incoming {
			if item.itemIndex > outgoingItem.itemIndex {
				item.itemIndex--
			}
		}
		for _, item := range b.outgoing {
			if item.itemIndex > outgoingItem.itemIndex {
				item.itemIndex--
			}
		}
		b.host.SetState(func() { b.itemsCount-- })
	})
}

// RemoveAllItems removes all the items, and animates them out with the item
// that the builder builds.
//
// Flutter: widgets/animated_scroll_view.dart SliverAnimatedListState.removeAllItems
func (b *animatedMultiBox) RemoveAllItems(builder AnimatedRemovedItemBuilder, opts ...AnimatedItemOpts) {
	visibleItemCount := b.itemsCount - len(b.outgoing)
	for i := visibleItemCount - 1; i >= 0; i-- {
		b.RemoveItem(i, builder, opts...)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAnimatedGridMissing                = "AnimatedGridMissing"
	tagAnimatedItemCountNegative          = "AnimatedItemCountNegative"
	tagAnimatedItemIndexInvalid           = "AnimatedItemIndexInvalid"
	tagAnimatedItemOutgoing               = "AnimatedItemOutgoing"
	tagAnimatedItemRemovedTwice           = "AnimatedItemRemovedTwice"
	tagAnimatedListMissing                = "AnimatedListMissing"
	tagAnimatedRemovedSeparatorBuilderNil = "AnimatedRemovedSeparatorBuilderNil"
	tagAnimatedScrollViewNotBuilt         = "AnimatedScrollViewNotBuilt"
	tagSliverAnimatedGridMissing          = "SliverAnimatedGridMissing"
	tagSliverAnimatedListMissing          = "SliverAnimatedListMissing"
)
