// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ReorderableListViewOpts are the optional parameters of NewReorderableListView
// and NewReorderableListViewBuilder.
//
// Flutter: material/reorderable_list.dart ReorderableListView.new
type ReorderableListViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnReorder is called when an item is dropped at a new place. Deprecated in
	// Flutter in favor of OnReorderItem; exactly one of the two must be given.
	OnReorder widgets.ReorderCallback
	// OnReorderItem is called when an item is dropped at a new place; the new index
	// is that of the item after it is removed from the old one.
	OnReorderItem widgets.ReorderCallback
	// OnReorderStart is called when the user starts to drag an item.
	OnReorderStart func(index int)
	// OnReorderEnd is called when the user drops the item.
	OnReorderEnd func(index int)
	// ItemExtent is the extent of all the items; at most one of ItemExtent,
	// ItemExtentBuilder and PrototypeItem may be given.
	ItemExtent foundation.Opt[float64]
	// ItemExtentBuilder gives the extent of each item.
	ItemExtentBuilder rendering.ItemExtentBuilder
	// PrototypeItem is the widget whose extent all the items have.
	PrototypeItem widgets.Widget
	// ProxyDecorator decorates the item that is being dragged; the default
	// raises it on a Material with an elevation of up to 6.
	ProxyDecorator widgets.ReorderItemProxyDecorator
	// BuildDefaultDragHandles is whether the items get a drag handle on the
	// desktop and a long-press drag on the other platforms; the default is true.
	BuildDefaultDragHandles foundation.Opt[bool]
	// Padding is the space around the items, the header and the footer.
	Padding painting.EdgeInsets
	// Header is a widget to show before the items; it does not reorder.
	Header widgets.Widget
	// Footer is a widget to show after the items; it does not reorder.
	Footer widgets.Widget
	// ScrollDirection is the axis along which the list scrolls; the default is
	// AxisVertical.
	ScrollDirection painting.Axis
	// Reverse is whether the list scrolls against the reading direction.
	Reverse bool
	// ScrollController reads and controls the scroll position.
	ScrollController widgets.ScrollController
	// Primary is whether this is the primary scroll view; the default is true
	// when the list scrolls vertically and has no controller.
	Primary foundation.Opt[bool]
	// Physics is how the list responds to the user.
	Physics widgets.ScrollPhysics
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
	KeyboardDismissBehavior widgets.ScrollViewKeyboardDismissBehavior
	// RestorationID is the identifier for restoring the scroll offset.
	RestorationID string
	// ClipBehavior is how the list clips; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// AutoScrollerVelocityScalar is the speed of the scrolling when an item is
	// dragged to the edge; the default is 50.
	AutoScrollerVelocityScalar foundation.Opt[float64]
	// DragBoundaryProvider gives the boundary the dragged item stays within.
	DragBoundaryProvider widgets.ReorderDragBoundaryProvider
	// MouseCursor is the cursor over the drag handle; the default is the grab
	// cursor, and the grabbing cursor while an item is dragged. A
	// widgets.WidgetStateMouseCursor is resolved with widgets.WidgetStateDragged.
	MouseCursor services.MouseCursor
}

// ReorderableListView is a list of items that the user can reorder by dragging
// them. Each item needs a key. On the desktop every item has a drag handle at its
// end; on the other platforms a long press starts the drag.
//
// Flutter: material/reorderable_list.dart ReorderableListView
type ReorderableListView struct {
	widgets.StatefulWidget
	itemBuilder widgets.IndexedWidgetBuilder
	itemCount   int
	opts        ReorderableListViewOpts
}

// NewReorderableListView creates a list of the children, which must have keys. In
// debug builds it panics when a child has no key (ReorderableListViewChildKeyMissing),
// when more than one of the extents is given (ReorderableExtentConflict), or when
// not exactly one of OnReorder and OnReorderItem is given
// (ReorderableCallbackInvalid).
//
// Flutter: material/reorderable_list.dart ReorderableListView.new
func NewReorderableListView(children []widgets.Widget, opts ...ReorderableListViewOpts) *ReorderableListView {
	if foundation.DebugMode {
		for i, c := range children {
			if c.Key() == nil {
				panic(errors.New("ReorderableListViewChildKeyMissing", "all children of this widget must have a key", "index", i))
			}
		}
	}
	return NewReorderableListViewBuilder(func(_ widgets.BuildContext, index int) widgets.Widget { return children[index] },
		len(children), opts...)
}

// NewReorderableListViewBuilder creates a list of itemCount items that the builder
// builds on demand. The items must have keys; see NewReorderableListView for the
// errors, and ReorderableItemCountNegative when itemCount is negative.
//
// Flutter: material/reorderable_list.dart ReorderableListView.builder
func NewReorderableListViewBuilder(itemBuilder widgets.IndexedWidgetBuilder, itemCount int, opts ...ReorderableListViewOpts) *ReorderableListView {
	var o ReorderableListViewOpts
	if len(opts) > 1 {
		panic(errors.New("OptsCount", "pass at most one options value", "got", len(opts)))
	}
	if len(opts) == 1 {
		o = opts[0]
	}
	if foundation.DebugMode {
		if itemCount < 0 {
			panic(errors.New("ReorderableItemCountNegative", "the itemCount must not be negative", "itemCount", itemCount))
		}
		given := 0
		if o.ItemExtent.IsSome() {
			given++
		}
		if o.ItemExtentBuilder != nil {
			given++
		}
		if o.PrototypeItem != nil {
			given++
		}
		if given > 1 {
			panic(errors.New("ReorderableExtentConflict", "you can only pass one of itemExtent, prototypeItem and itemExtentBuilder"))
		}
		if (o.OnReorderItem != nil) == (o.OnReorder != nil) {
			panic(errors.New("ReorderableCallbackInvalid", "exactly one of onReorderItem and onReorder must be given; the onReorder callback is obsolete and is replaced by onReorderItem"))
		}
	}
	if o.ScrollDirection == painting.AxisUnspecified {
		o.ScrollDirection = painting.AxisVertical
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &ReorderableListView{
		StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		itemBuilder:    itemBuilder, itemCount: itemCount, opts: o,
	}
}

// ItemCount is the number of items.
//
// Flutter: material/reorderable_list.dart ReorderableListView.itemCount
func (w *ReorderableListView) ItemCount() int { return w.itemCount }

// CreateState creates the state.
//
// Flutter: material/reorderable_list.dart ReorderableListView.createState
func (w *ReorderableListView) CreateState() widgets.State {
	return &reorderableListViewState{dragging: foundation.NewValueNotifier(false)}
}

// defaultReorderMouseCursor is the cursor over the drag handle: grab, and
// grabbing while an item is dragged. It is read-only.
var defaultReorderMouseCursor = widgets.WidgetStateMouseCursorFromMap(widgets.WidgetStateMap[services.MouseCursor]{
	{Key: widgets.WidgetStateDragged, Value: services.SystemMouseCursors.Grabbing},
	{Key: widgets.WidgetStateAny, Value: services.SystemMouseCursors.Grab},
})

// reorderableListViewState is the state of ReorderableListView.
//
// Flutter: material/reorderable_list.dart _ReorderableListViewState
type reorderableListViewState struct {
	widgets.StateOf[*ReorderableListView]
	dragging *foundation.ValueNotifier[bool]
}

// reorderableListViewChildKey is what the global key of an item is made of: the key
// of the child and the state. Unlike a GlobalObjectKey of a widget key, two keys
// are equal when the keys of the children are equal.
//
// Flutter: material/reorderable_list.dart _ReorderableListViewChildGlobalKey
type reorderableListViewChildKey struct {
	subKey foundation.Key
	state  *reorderableListViewState
}

func (s *reorderableListViewState) itemBuilder(bc widgets.BuildContext, index int) widgets.Widget {
	w := s.Widget()
	item := w.itemBuilder(bc, index)
	if foundation.DebugMode && item.Key() == nil {
		panic(errors.New("ReorderableListViewItemKeyMissing", "every item of ReorderableListView must have a key", "index", index))
	}
	itemKey := widgets.NewGlobalObjectKey[widgets.State](reorderableListViewChildKey{item.Key(), s})
	if !w.opts.BuildDefaultDragHandles.OrElse(true) {
		return widgets.NewKeyedSubtree(item, widgets.WidgetOpts{Key: itemKey})
	}
	switch ThemeOf(bc).Platform {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
		return widgets.NewReorderableDelayedDragStartListener(index, item, widgets.ReorderableDragStartListenerOpts{Key: itemKey})
	}
	cursor := w.opts.MouseCursor
	if cursor == nil {
		cursor = defaultReorderMouseCursor
	}
	dragHandle := widgets.NewListenableBuilder(s.dragging, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
		var states widgets.WidgetStates
		if s.dragging.Value() {
			states = widgets.NewWidgetStates(widgets.WidgetStateDragged)
		}
		return widgets.NewMouseRegion(child, widgets.MouseRegionOpts{Cursor: widgets.WidgetStateResolveAs[services.MouseCursor](cursor, states)})
	}, widgets.NewIcon(Icons.DragHandle()))
	var positioned widgets.Widget
	if w.opts.ScrollDirection == painting.AxisHorizontal {
		positioned = widgets.NewPositionedDirectional(
			widgets.NewAlign(widgets.NewReorderableDragStartListener(index, dragHandle), widgets.AlignOpts{Alignment: painting.AlignmentDirectionalBottomCenter}),
			widgets.PositionedDirectionalOpts{Start: foundation.Some(0.0), End: foundation.Some(0.0), Bottom: foundation.Some(8.0)})
	} else {
		positioned = widgets.NewPositionedDirectional(
			widgets.NewAlign(widgets.NewReorderableDragStartListener(index, dragHandle), widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterEnd}),
			widgets.PositionedDirectionalOpts{Top: foundation.Some(0.0), Bottom: foundation.Some(0.0), End: foundation.Some(8.0)})
	}
	return widgets.NewStack([]widgets.Widget{item, positioned}, widgets.StackOpts{Key: itemKey})
}

// proxyDecorator raises the item that is dragged on a Material.
//
// Flutter: material/reorderable_list.dart _ReorderableListViewState._proxyDecorator
func (s *reorderableListViewState) proxyDecorator(child widgets.Widget, _ int, anim animation.Animation[float64]) widgets.Widget {
	return widgets.NewAnimatedBuilder(anim, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
		animValue := animation.Curves.EaseInOut.Transform(anim.Value())
		return NewMaterial(child, MaterialOpts{Elevation: ui.LerpDouble(0, 6, animValue)})
	}, child)
}

// Dispose disposes the notifier of the drag.
func (s *reorderableListViewState) Dispose() { s.dragging.Dispose() }

// Build builds the scroll view with the reorderable sliver, splitting the padding
// between the header, the footer and the list.
func (s *reorderableListViewState) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	widgets.DebugCheckHasOverlay(bc)
	w := s.Widget()
	o := w.opts

	// If there is a header or footer we can't just apply the padding to the list,
	// so we break it up into padding for the header, footer and padding for the list.
	var start, end foundation.Opt[float64]
	if o.Header != nil {
		start = foundation.Some(0.0)
	}
	if o.Footer != nil {
		end = foundation.Some(0.0)
	}
	if o.Reverse {
		start, end = end, start
	}
	zero := foundation.Some(0.0)
	var startPadding, endPadding, listPadding painting.EdgeInsets
	switch {
	case !start.IsSome() && !end.IsSome():
		listPadding = o.Padding
	case o.ScrollDirection == painting.AxisHorizontal:
		startPadding = o.Padding.CopyWith(painting.EdgeInsetsCopy{Left: zero})
		endPadding = o.Padding.CopyWith(painting.EdgeInsetsCopy{Right: zero})
		listPadding = o.Padding.CopyWith(painting.EdgeInsetsCopy{Left: start, Right: end})
	default:
		startPadding = o.Padding.CopyWith(painting.EdgeInsetsCopy{Top: zero})
		endPadding = o.Padding.CopyWith(painting.EdgeInsetsCopy{Bottom: zero})
		listPadding = o.Padding.CopyWith(painting.EdgeInsetsCopy{Top: start, Bottom: end})
	}
	headerPadding, footerPadding := endPadding, startPadding
	if o.Reverse {
		headerPadding, footerPadding = startPadding, endPadding
	}

	proxy := o.ProxyDecorator
	if proxy == nil {
		proxy = s.proxyDecorator
	}
	var slivers []widgets.Widget
	if o.Header != nil {
		slivers = append(slivers, widgets.NewSliverPadding(headerPadding, widgets.NewSliverToBoxAdapter(o.Header)))
	}
	slivers = append(slivers, widgets.NewSliverPadding(listPadding, widgets.NewSliverReorderableList(s.itemBuilder, w.itemCount, widgets.SliverReorderableListOpts{
		ItemExtent: o.ItemExtent, ItemExtentBuilder: o.ItemExtentBuilder, PrototypeItem: o.PrototypeItem,
		OnReorder: o.OnReorder, OnReorderItem: o.OnReorderItem,
		OnReorderStart: func(index int) {
			s.dragging.SetValue(true)
			if o.OnReorderStart != nil {
				o.OnReorderStart(index)
			}
		},
		OnReorderEnd: func(index int) {
			s.dragging.SetValue(false)
			if o.OnReorderEnd != nil {
				o.OnReorderEnd(index)
			}
		},
		ProxyDecorator: proxy, AutoScrollerVelocityScalar: o.AutoScrollerVelocityScalar, DragBoundaryProvider: o.DragBoundaryProvider,
	})))
	if o.Footer != nil {
		slivers = append(slivers, widgets.NewSliverPadding(footerPadding, widgets.NewSliverToBoxAdapter(o.Footer)))
	}
	return widgets.NewCustomScrollView(slivers, widgets.CustomScrollViewOpts{
		ScrollDirection: o.ScrollDirection, Reverse: o.Reverse, Controller: o.ScrollController, Primary: o.Primary,
		Physics: o.Physics, ShrinkWrap: o.ShrinkWrap, Anchor: o.Anchor, ScrollCacheExtent: o.ScrollCacheExtent,
		DragStartBehavior: o.DragStartBehavior, KeyboardDismissBehavior: o.KeyboardDismissBehavior,
		RestorationID: o.RestorationID, ClipBehavior: o.ClipBehavior,
	})
}
