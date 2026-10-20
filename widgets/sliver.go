// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"math"
	"sort"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
)

// sliverMultiBoxWidget is what the element of a sliver with many boxes needs of
// its widget.
type sliverMultiBoxWidget interface {
	renderObjectWidget
	// Delegate supplies the children.
	Delegate() SliverChildDelegate
	// EstimateMaxScrollOffset estimates the scroll extent of the list.
	EstimateMaxScrollOffset(constraints rendering.SliverConstraints, firstIndex, lastIndex int, leadingScrollOffset, trailingScrollOffset float64) foundation.Opt[float64]
}

// SliverMultiBoxAdaptorWidget is the base of the slivers that lay out many boxes
// that a SliverChildDelegate supplies: SliverList, SliverFixedExtentList,
// SliverVariedExtentList and SliverGrid. Embed it and implement
// CreateRenderObject.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorWidget
type SliverMultiBoxAdaptorWidget struct {
	RenderObjectWidget
	delegate SliverChildDelegate
}

// NewSliverMultiBoxAdaptorWidgetBase creates the embedded base of a sliver with
// a delegate.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorWidget.new
func NewSliverMultiBoxAdaptorWidgetBase(delegate SliverChildDelegate, opts ...WidgetOpts) SliverMultiBoxAdaptorWidget {
	return SliverMultiBoxAdaptorWidget{RenderObjectWidget: NewRenderObjectWidgetBase(opts...), delegate: delegate}
}

// Delegate supplies the children.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorWidget.delegate
func (w *SliverMultiBoxAdaptorWidget) Delegate() SliverChildDelegate { return w.delegate }

// EstimateMaxScrollOffset asks the delegate; none lets the sliver extrapolate.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorWidget.estimateMaxScrollOffset
func (w *SliverMultiBoxAdaptorWidget) EstimateMaxScrollOffset(_ rendering.SliverConstraints, firstIndex, lastIndex int, leadingScrollOffset, trailingScrollOffset float64) foundation.Opt[float64] {
	if foundation.DebugMode && lastIndex < firstIndex {
		panic(errors.New("SliverLastIndexBeforeFirst", "the last index must not be less than the first", "first", firstIndex, "last", lastIndex))
	}
	return w.delegate.EstimateMaxScrollOffset(firstIndex, lastIndex, leadingScrollOffset, trailingScrollOffset)
}

// SliverBuilderOpts are the optional parameters of the builder constructors of
// the slivers with boxes (SliverList, SliverFixedExtentList,
// SliverVariedExtentList and SliverGrid).
//
// Flutter: widgets/sliver.dart SliverList.builder
type SliverBuilderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// FindChildIndexCallback finds the index of the child that has a key.
	FindChildIndexCallback ChildIndexGetter
	// ItemCount is the number of items; none means unbounded, until the builder
	// returns nil.
	ItemCount foundation.Opt[int]
	// AddAutomaticKeepAlives is whether to wrap each item in an
	// AutomaticKeepAlive; the default is true.
	AddAutomaticKeepAlives foundation.Opt[bool]
	// AddRepaintBoundaries is whether to wrap each item in a RepaintBoundary; the
	// default is true.
	AddRepaintBoundaries foundation.Opt[bool]
	// AddSemanticIndexes is whether to wrap each item in an IndexedSemantics; the
	// default is true.
	AddSemanticIndexes foundation.Opt[bool]
	// SemanticIndexOffset is added to the semantic index of every item.
	SemanticIndexOffset int
}

func (o SliverBuilderOpts) delegate(itemBuilder NullableIndexedWidgetBuilder) *SliverChildBuilderDelegate {
	return NewSliverChildBuilderDelegate(itemBuilder, SliverChildBuilderDelegateOpts{
		FindChildIndexCallback: o.FindChildIndexCallback, ChildCount: o.ItemCount, AddAutomaticKeepAlives: o.AddAutomaticKeepAlives,
		AddRepaintBoundaries: o.AddRepaintBoundaries, AddSemanticIndexes: o.AddSemanticIndexes, SemanticIndexOffset: o.SemanticIndexOffset})
}

// SliverListOpts are the optional parameters of the list constructors of the
// slivers with boxes.
//
// Flutter: widgets/sliver.dart SliverList.list
type SliverListOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AddAutomaticKeepAlives is whether to wrap each child in an
	// AutomaticKeepAlive; the default is true.
	AddAutomaticKeepAlives foundation.Opt[bool]
	// AddRepaintBoundaries is whether to wrap each child in a RepaintBoundary; the
	// default is true.
	AddRepaintBoundaries foundation.Opt[bool]
	// AddSemanticIndexes is whether to wrap each child in an IndexedSemantics; the
	// default is true.
	AddSemanticIndexes foundation.Opt[bool]
	// SemanticIndexOffset is added to the semantic index of every child.
	SemanticIndexOffset int
}

func (o SliverListOpts) delegate(children []Widget) *SliverChildListDelegate {
	return NewSliverChildListDelegate(children, SliverChildListDelegateOpts{
		AddAutomaticKeepAlives: o.AddAutomaticKeepAlives, AddRepaintBoundaries: o.AddRepaintBoundaries,
		AddSemanticIndexes: o.AddSemanticIndexes, SemanticIndexOffset: o.SemanticIndexOffset})
}

// SliverSeparatedOpts are the optional parameters of the separated constructors
// of the slivers with boxes.
//
// Flutter: widgets/sliver.dart SliverList.separated
type SliverSeparatedOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// FindItemIndexCallback finds the index of the item (not of the child, which
	// counts the separators) that has a key.
	FindItemIndexCallback ChildIndexGetter
	// ItemCount is the number of items; none means unbounded, until the builder
	// returns nil.
	ItemCount foundation.Opt[int]
	// AddAutomaticKeepAlives is whether to wrap each item in an
	// AutomaticKeepAlive; the default is true.
	AddAutomaticKeepAlives foundation.Opt[bool]
	// AddRepaintBoundaries is whether to wrap each item in a RepaintBoundary; the
	// default is true.
	AddRepaintBoundaries foundation.Opt[bool]
	// AddSemanticIndexes is whether to wrap each item in an IndexedSemantics; the
	// default is true.
	AddSemanticIndexes foundation.Opt[bool]
}

// separatedDelegate is the delegate of SliverList.separated and
// ListView.separated: items at the even indices, separators at the odd ones. A
// separator that is nil panics in debug builds with the tag SeparatorBuilderNil.
func separatedDelegate(itemBuilder, separatorBuilder NullableIndexedWidgetBuilder, o SliverSeparatedOpts) *SliverChildBuilderDelegate {
	var find ChildIndexGetter
	if o.FindItemIndexCallback != nil {
		find = func(key foundation.Key) foundation.Opt[int] {
			if itemIndex, ok := o.FindItemIndexCallback(key).Get(); ok {
				return foundation.Some(itemIndex * 2)
			}
			return foundation.None[int]()
		}
	}
	var count foundation.Opt[int]
	if n, ok := o.ItemCount.Get(); ok {
		count = foundation.Some(max(0, n*2-1))
	}
	return NewSliverChildBuilderDelegate(func(bc BuildContext, index int) Widget {
		itemIndex := index / 2
		if index%2 == 0 {
			return itemBuilder(bc, itemIndex)
		}
		widget := separatorBuilder(bc, itemIndex)
		if foundation.DebugMode && widget == nil {
			panic(errors.New("SeparatorBuilderNil", "separatorBuilder cannot return nil"))
		}
		return widget
	}, SliverChildBuilderDelegateOpts{
		FindChildIndexCallback: find, ChildCount: count, AddAutomaticKeepAlives: o.AddAutomaticKeepAlives,
		AddRepaintBoundaries: o.AddRepaintBoundaries, AddSemanticIndexes: o.AddSemanticIndexes,
		SemanticIndexCallback: func(_ Widget, index int) foundation.Opt[int] {
			if index%2 == 0 {
				return foundation.Some(index / 2)
			}
			return foundation.None[int]()
		}})
}

// SliverListWidgetOpts are the optional parameters of NewSliverList.
//
// Flutter: widgets/sliver.dart SliverList.new
type SliverListWidgetOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// SliverList is a sliver that places box children in a linear array along the
// main axis.
//
// Flutter: widgets/sliver.dart SliverList
type SliverList struct {
	SliverMultiBoxAdaptorWidget
}

// NewSliverList creates a sliver list whose children the delegate supplies.
//
// Flutter: widgets/sliver.dart SliverList.new
func NewSliverList(delegate SliverChildDelegate, opts ...SliverListWidgetOpts) *SliverList {
	return &SliverList{NewSliverMultiBoxAdaptorWidgetBase(delegate, WidgetOpts{Key: firstOpts(opts).Key})}
}

// NewSliverListBuilder creates a sliver list whose items are built lazily.
//
// Flutter: widgets/sliver.dart SliverList.builder
func NewSliverListBuilder(itemBuilder NullableIndexedWidgetBuilder, opts ...SliverBuilderOpts) *SliverList {
	o := firstOpts(opts)
	return NewSliverList(o.delegate(itemBuilder), SliverListWidgetOpts{Key: o.Key})
}

// NewSliverListSeparated creates a sliver list with a separator between the
// items.
//
// Flutter: widgets/sliver.dart SliverList.separated
func NewSliverListSeparated(itemBuilder, separatorBuilder NullableIndexedWidgetBuilder, opts ...SliverSeparatedOpts) *SliverList {
	o := firstOpts(opts)
	return NewSliverList(separatedDelegate(itemBuilder, separatorBuilder, o), SliverListWidgetOpts{Key: o.Key})
}

// NewSliverListList creates a sliver list of the children.
//
// Flutter: widgets/sliver.dart SliverList.list
func NewSliverListList(children []Widget, opts ...SliverListOpts) *SliverList {
	o := firstOpts(opts)
	return NewSliverList(o.delegate(children), SliverListWidgetOpts{Key: o.Key})
}

// CreateElement creates the element; its moved children leave their index empty.
//
// Flutter: widgets/sliver.dart SliverList.createElement
func (w *SliverList) CreateElement() Element { return NewSliverMultiBoxAdaptorElement(w, true) }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/sliver.dart SliverList.createRenderObject
func (w *SliverList) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverList(bc.(rendering.RenderSliverBoxChildManager))
}

// SliverFixedExtentList is a sliver that places box children, all with the same
// main axis extent, in a linear array.
//
// Flutter: widgets/sliver.dart SliverFixedExtentList
type SliverFixedExtentList struct {
	SliverMultiBoxAdaptorWidget
	itemExtent float64
}

// NewSliverFixedExtentList creates a fixed extent sliver list whose children the
// delegate supplies.
//
// Flutter: widgets/sliver.dart SliverFixedExtentList.new
func NewSliverFixedExtentList(delegate SliverChildDelegate, itemExtent float64, opts ...SliverListWidgetOpts) *SliverFixedExtentList {
	return &SliverFixedExtentList{NewSliverMultiBoxAdaptorWidgetBase(delegate, WidgetOpts{Key: firstOpts(opts).Key}), itemExtent}
}

// NewSliverFixedExtentListBuilder creates a fixed extent sliver list whose items
// are built lazily.
//
// Flutter: widgets/sliver.dart SliverFixedExtentList.builder
func NewSliverFixedExtentListBuilder(itemBuilder NullableIndexedWidgetBuilder, itemExtent float64, opts ...SliverBuilderOpts) *SliverFixedExtentList {
	o := firstOpts(opts)
	return NewSliverFixedExtentList(o.delegate(itemBuilder), itemExtent, SliverListWidgetOpts{Key: o.Key})
}

// NewSliverFixedExtentListList creates a fixed extent sliver list of the
// children.
//
// Flutter: widgets/sliver.dart SliverFixedExtentList.list
func NewSliverFixedExtentListList(itemExtent float64, children []Widget, opts ...SliverListOpts) *SliverFixedExtentList {
	o := firstOpts(opts)
	return NewSliverFixedExtentList(o.delegate(children), itemExtent, SliverListWidgetOpts{Key: o.Key})
}

// ItemExtent is the extent of every child along the main axis.
//
// Flutter: widgets/sliver.dart SliverFixedExtentList.itemExtent
func (w *SliverFixedExtentList) ItemExtent() float64 { return w.itemExtent }

// CreateElement creates the element.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorWidget.createElement
func (w *SliverFixedExtentList) CreateElement() Element {
	return NewSliverMultiBoxAdaptorElement(w, false)
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/sliver.dart SliverFixedExtentList.createRenderObject
func (w *SliverFixedExtentList) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverFixedExtentList(bc.(rendering.RenderSliverBoxChildManager), w.itemExtent)
}

// UpdateRenderObject sets the item extent.
//
// Flutter: widgets/sliver.dart SliverFixedExtentList.updateRenderObject
func (w *SliverFixedExtentList) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderSliverFixedExtentList).SetItemExtent(w.itemExtent)
}

// SliverVariedExtentList is a sliver that places box children with the extents
// that a function gives, in a linear array.
//
// Flutter: widgets/sliver.dart SliverVariedExtentList
type SliverVariedExtentList struct {
	SliverMultiBoxAdaptorWidget
	itemExtentBuilder rendering.ItemExtentBuilder
}

// NewSliverVariedExtentList creates a sliver list whose children the delegate
// supplies, and whose children's extents the function gives.
//
// Flutter: widgets/sliver.dart SliverVariedExtentList.new
func NewSliverVariedExtentList(delegate SliverChildDelegate, itemExtentBuilder rendering.ItemExtentBuilder, opts ...SliverListWidgetOpts) *SliverVariedExtentList {
	return &SliverVariedExtentList{NewSliverMultiBoxAdaptorWidgetBase(delegate, WidgetOpts{Key: firstOpts(opts).Key}), itemExtentBuilder}
}

// NewSliverVariedExtentListBuilder creates a varied extent sliver list whose
// items are built lazily.
//
// Flutter: widgets/sliver.dart SliverVariedExtentList.builder
func NewSliverVariedExtentListBuilder(itemBuilder NullableIndexedWidgetBuilder, itemExtentBuilder rendering.ItemExtentBuilder, opts ...SliverBuilderOpts) *SliverVariedExtentList {
	o := firstOpts(opts)
	return NewSliverVariedExtentList(o.delegate(itemBuilder), itemExtentBuilder, SliverListWidgetOpts{Key: o.Key})
}

// NewSliverVariedExtentListList creates a varied extent sliver list of the
// children.
//
// Flutter: widgets/sliver.dart SliverVariedExtentList.list
func NewSliverVariedExtentListList(itemExtentBuilder rendering.ItemExtentBuilder, children []Widget, opts ...SliverListOpts) *SliverVariedExtentList {
	o := firstOpts(opts)
	return NewSliverVariedExtentList(o.delegate(children), itemExtentBuilder, SliverListWidgetOpts{Key: o.Key})
}

// ItemExtentBuilder gives the extent of the child at an index.
//
// Flutter: widgets/sliver.dart SliverVariedExtentList.itemExtentBuilder
func (w *SliverVariedExtentList) ItemExtentBuilder() rendering.ItemExtentBuilder {
	return w.itemExtentBuilder
}

// CreateElement creates the element.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorWidget.createElement
func (w *SliverVariedExtentList) CreateElement() Element {
	return NewSliverMultiBoxAdaptorElement(w, false)
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/sliver.dart SliverVariedExtentList.createRenderObject
func (w *SliverVariedExtentList) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverVariedExtentList(bc.(rendering.RenderSliverBoxChildManager), w.itemExtentBuilder)
}

// UpdateRenderObject sets the extent builder.
//
// Flutter: widgets/sliver.dart SliverVariedExtentList.updateRenderObject
func (w *SliverVariedExtentList) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderSliverVariedExtentList).SetItemExtentBuilder(w.itemExtentBuilder)
}

// SliverGrid is a sliver that places multiple box children in a two dimensional
// arrangement.
//
// Flutter: widgets/sliver.dart SliverGrid
type SliverGrid struct {
	SliverMultiBoxAdaptorWidget
	gridDelegate rendering.SliverGridDelegate
}

// NewSliverGrid creates a sliver grid whose children the delegate supplies,
// laid out by the grid delegate.
//
// Flutter: widgets/sliver.dart SliverGrid.new
func NewSliverGrid(delegate SliverChildDelegate, gridDelegate rendering.SliverGridDelegate, opts ...SliverListWidgetOpts) *SliverGrid {
	return &SliverGrid{NewSliverMultiBoxAdaptorWidgetBase(delegate, WidgetOpts{Key: firstOpts(opts).Key}), gridDelegate}
}

// NewSliverGridBuilder creates a sliver grid whose items are built lazily.
//
// Flutter: widgets/sliver.dart SliverGrid.builder
func NewSliverGridBuilder(gridDelegate rendering.SliverGridDelegate, itemBuilder NullableIndexedWidgetBuilder, opts ...SliverBuilderOpts) *SliverGrid {
	o := firstOpts(opts)
	return NewSliverGrid(o.delegate(itemBuilder), gridDelegate, SliverListWidgetOpts{Key: o.Key})
}

// NewSliverGridList creates a sliver grid of the children.
//
// Flutter: widgets/sliver.dart SliverGrid.list
func NewSliverGridList(gridDelegate rendering.SliverGridDelegate, children []Widget, opts ...SliverListOpts) *SliverGrid {
	o := firstOpts(opts)
	return NewSliverGrid(o.delegate(children), gridDelegate, SliverListWidgetOpts{Key: o.Key})
}

// SliverGridCountOpts are the optional parameters of NewSliverGridCount and
// NewSliverGridExtent.
//
// Flutter: widgets/sliver.dart SliverGrid.count
type SliverGridCountOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// MainAxisSpacing is the number of logical pixels between each child along
	// the main axis.
	MainAxisSpacing float64
	// CrossAxisSpacing is the number of logical pixels between each child along
	// the cross axis.
	CrossAxisSpacing float64
	// ChildAspectRatio is the ratio of the cross axis to the main axis extent of
	// each child; the default is 1.
	ChildAspectRatio foundation.Opt[float64]
}

// NewSliverGridCount creates a sliver grid with a fixed number of tiles in the
// cross axis.
//
// Flutter: widgets/sliver.dart SliverGrid.count
func NewSliverGridCount(crossAxisCount int, children []Widget, opts ...SliverGridCountOpts) *SliverGrid {
	o := firstOpts(opts)
	return NewSliverGrid(NewSliverChildListDelegate(children), rendering.NewSliverGridDelegateWithFixedCrossAxisCount(crossAxisCount,
		rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{MainAxisSpacing: o.MainAxisSpacing, CrossAxisSpacing: o.CrossAxisSpacing,
			ChildAspectRatio: o.ChildAspectRatio}), SliverListWidgetOpts{Key: o.Key})
}

// NewSliverGridExtent creates a sliver grid with tiles that have a maximum
// cross axis extent.
//
// Flutter: widgets/sliver.dart SliverGrid.extent
func NewSliverGridExtent(maxCrossAxisExtent float64, children []Widget, opts ...SliverGridCountOpts) *SliverGrid {
	o := firstOpts(opts)
	return NewSliverGrid(NewSliverChildListDelegate(children), rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(maxCrossAxisExtent,
		rendering.SliverGridDelegateWithMaxCrossAxisExtentOpts{MainAxisSpacing: o.MainAxisSpacing, CrossAxisSpacing: o.CrossAxisSpacing,
			ChildAspectRatio: o.ChildAspectRatio}), SliverListWidgetOpts{Key: o.Key})
}

// GridDelegate lays out the children.
//
// Flutter: widgets/sliver.dart SliverGrid.gridDelegate
func (w *SliverGrid) GridDelegate() rendering.SliverGridDelegate { return w.gridDelegate }

// CreateElement creates the element.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorWidget.createElement
func (w *SliverGrid) CreateElement() Element { return NewSliverMultiBoxAdaptorElement(w, false) }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/sliver.dart SliverGrid.createRenderObject
func (w *SliverGrid) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverGrid(bc.(rendering.RenderSliverBoxChildManager), w.gridDelegate)
}

// UpdateRenderObject sets the grid delegate.
//
// Flutter: widgets/sliver.dart SliverGrid.updateRenderObject
func (w *SliverGrid) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderSliverGrid).SetGridDelegate(w.gridDelegate)
}

// EstimateMaxScrollOffset asks the delegate, then the grid layout.
//
// Flutter: widgets/sliver.dart SliverGrid.estimateMaxScrollOffset
func (w *SliverGrid) EstimateMaxScrollOffset(constraints rendering.SliverConstraints, firstIndex, lastIndex int, leadingScrollOffset, trailingScrollOffset float64) foundation.Opt[float64] {
	if v := w.SliverMultiBoxAdaptorWidget.EstimateMaxScrollOffset(constraints, firstIndex, lastIndex, leadingScrollOffset, trailingScrollOffset); v.IsSome() {
		return v
	}
	count, _ := w.delegate.EstimatedChildCount().Get()
	return foundation.Some(w.gridDelegate.GetLayout(constraints).ComputeMaxScrollOffset(count))
}

// intKeyedElements is the map from the index of a child to its element, with the
// keys in order, as Dart's SplayTreeMap. A child that is nil stays in the map.
type intKeyedElements struct {
	m map[int]Element
}

func newIntKeyedElements() *intKeyedElements { return &intKeyedElements{m: map[int]Element{}} }

func (e *intKeyedElements) get(i int) Element { return e.m[i] }

func (e *intKeyedElements) has(i int) bool { _, ok := e.m[i]; return ok }

func (e *intKeyedElements) set(i int, el Element) { e.m[i] = el }

func (e *intKeyedElements) remove(i int) { delete(e.m, i) }

func (e *intKeyedElements) keys() []int {
	keys := make([]int, 0, len(e.m))
	for k := range e.m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func (e *intKeyedElements) firstKey() (int, bool) {
	keys := e.keys()
	if len(keys) == 0 {
		return 0, false
	}
	return keys[0], true
}

func (e *intKeyedElements) lastKey() (int, bool) {
	keys := e.keys()
	if len(keys) == 0 {
		return 0, false
	}
	return keys[len(keys)-1], true
}

// SliverMultiBoxAdaptorElement is the element of a SliverMultiBoxAdaptorWidget.
// It is the child manager of its render object: it builds the children, with the
// delegate, as the render object asks for them while it lays itself out.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement
type SliverMultiBoxAdaptorElement struct {
	RenderObjectElement
	replaceMovedChildren        bool
	childElements               *intKeyedElements
	currentBeforeChild          rendering.Box
	currentlyUpdatingChildIndex foundation.Opt[int]
	didUnderflow                bool
}

// NewSliverMultiBoxAdaptorElement creates the element of a sliver with boxes.
// A widget that embeds SliverMultiBoxAdaptorWidget returns it from its
// CreateElement method; replaceMovedChildren is whether a child that moved
// leaves its index empty for the delegate to fill (true for SliverList).
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.new
func NewSliverMultiBoxAdaptorElement(w sliverMultiBoxWidget, replaceMovedChildren bool) *SliverMultiBoxAdaptorElement {
	e := &SliverMultiBoxAdaptorElement{}
	InitSliverMultiBoxAdaptorElement(e, e, w, replaceMovedChildren)
	return e
}

// InitSliverMultiBoxAdaptorElement sets up an element of a sliver with boxes; self
// is the outer element, for an element type that embeds this one.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.new
func InitSliverMultiBoxAdaptorElement(e *SliverMultiBoxAdaptorElement, self RenderObjectElementSelf, w sliverMultiBoxWidget, replaceMovedChildren bool) {
	e.replaceMovedChildren, e.childElements = replaceMovedChildren, newIntKeyedElements()
	InitRenderObjectElement(&e.RenderObjectElement, self, w)
}

func (e *SliverMultiBoxAdaptorElement) adaptor() sliverMultiBoxWidget {
	return e.widget.(sliverMultiBoxWidget)
}

func (e *SliverMultiBoxAdaptorElement) sliver() *rendering.RenderSliverMultiBoxAdaptor {
	return e.renderObject.(interface {
		AsRenderSliverMultiBoxAdaptor() *rendering.RenderSliverMultiBoxAdaptor
	}).AsRenderSliverMultiBoxAdaptor()
}

// Update rebuilds the children when the delegate says it must.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.update
func (e *SliverMultiBoxAdaptorElement) Update(newWidget Widget) {
	oldDelegate := e.adaptor().Delegate()
	e.RenderObjectElement.Update(newWidget)
	newDelegate := newWidget.(sliverMultiBoxWidget).Delegate()
	if newDelegate != oldDelegate && (foundation.RuntimeTypeName(newDelegate) != foundation.RuntimeTypeName(oldDelegate) || newDelegate.ShouldRebuild(oldDelegate)) {
		e.PerformRebuild()
	}
}

// PerformRebuild builds all the children that are live again.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.performRebuild
func (e *SliverMultiBoxAdaptorElement) PerformRebuild() {
	e.RenderObjectElement.PerformRebuild()
	e.currentBeforeChild = nil
	childrenUpdated := false
	if foundation.DebugMode && e.currentlyUpdatingChildIndex.IsSome() {
		panic(errors.New("SliverChildListLocked", "the children of a sliver were rebuilt while one of them was being updated"))
	}
	defer func() {
		e.currentlyUpdatingChildIndex = foundation.None[int]()
		e.sliver().SetDebugChildIntegrityEnabled(true)
	}()
	newChildren := newIntKeyedElements()
	indexToLayoutOffset := map[int]float64{}
	adaptor := e.adaptor()
	processElement := func(index int) {
		e.currentlyUpdatingChildIndex = foundation.Some(index)
		if cur := e.childElements.get(index); cur != nil && cur != newChildren.get(index) {
			e.childElements.set(index, e.UpdateChild(cur, nil, index))
			childrenUpdated = true
		}
		newChild := e.UpdateChild(newChildren.get(index), e.build(index, adaptor), index)
		if newChild != nil {
			childrenUpdated = childrenUpdated || e.childElements.get(index) != newChild
			e.childElements.set(index, newChild)
			parentData := rendering.SliverMultiBoxAdaptorParentDataOf(newChild.RenderObject())
			if index == 0 {
				parentData.LayoutOffset = foundation.Some(0.0)
			} else if offset, ok := indexToLayoutOffset[index]; ok {
				parentData.LayoutOffset = foundation.Some(offset)
			}
			if !parentData.KeptAlive() {
				e.currentBeforeChild, _ = newChild.RenderObject().(rendering.Box)
			}
		} else {
			childrenUpdated = true
			e.childElements.remove(index)
		}
	}
	for _, index := range e.childElements.keys() {
		child := e.childElements.get(index)
		var newIndex foundation.Opt[int]
		if key := child.AsElement().widget.Key(); key != nil {
			newIndex = adaptor.Delegate().FindIndexByKey(key)
		}
		var childParentData *rendering.SliverMultiBoxAdaptorParentData
		if ro := child.RenderObject(); ro != nil {
			childParentData = rendering.SliverMultiBoxAdaptorParentDataOf(ro)
		}
		if childParentData != nil && childParentData.LayoutOffset.IsSome() {
			indexToLayoutOffset[index] = childParentData.LayoutOffset.Value()
		}
		if ni, ok := newIndex.Get(); ok && ni != index {
			if childParentData != nil {
				childParentData.LayoutOffset = foundation.None[float64]()
			}
			newChildren.set(ni, child)
			if e.replaceMovedChildren && !newChildren.has(index) {
				newChildren.set(index, nil)
			}
			e.childElements.remove(index)
		} else if !newChildren.has(index) {
			newChildren.set(index, child)
		}
	}
	e.sliver().SetDebugChildIntegrityEnabled(false) // Moving children will temporary violate the integrity.
	for _, index := range newChildren.keys() {
		processElement(index)
	}
	if !childrenUpdated && e.didUnderflow {
		lastKey, ok := e.childElements.lastKey()
		if !ok {
			lastKey = -1
		}
		rightBoundary := lastKey + 1
		newChildren.set(rightBoundary, e.childElements.get(rightBoundary))
		processElement(rightBoundary)
	}
}

func (e *SliverMultiBoxAdaptorElement) build(index int, w sliverMultiBoxWidget) Widget {
	return w.Delegate().Build(e, index)
}

// CreateChild builds the child at the index and puts it after the given one.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.createChild
func (e *SliverMultiBoxAdaptorElement) CreateChild(index int, after rendering.Box) {
	if foundation.DebugMode && e.currentlyUpdatingChildIndex.IsSome() {
		panic(errors.New("SliverChildListLocked", "a child was created while another one was being updated"))
	}
	e.owner.BuildScope(e, func() {
		insertFirst := after == nil
		if foundation.DebugMode && !insertFirst && e.childElements.get(index-1) == nil {
			panic(errors.New("SliverChildBeforeMissing", "the child before the one to create is not live", "index", index))
		}
		if insertFirst {
			e.currentBeforeChild = nil
		} else {
			e.currentBeforeChild, _ = e.childElements.get(index - 1).RenderObject().(rendering.Box)
		}
		var newChild Element
		func() {
			defer func() { e.currentlyUpdatingChildIndex = foundation.None[int]() }()
			e.currentlyUpdatingChildIndex = foundation.Some(index)
			newChild = e.UpdateChild(e.childElements.get(index), e.build(index, e.adaptor()), index)
		}()
		if newChild != nil {
			e.childElements.set(index, newChild)
		} else {
			e.childElements.remove(index)
		}
	})
}

// UpdateChild updates the child, and keeps the layout offset of the parent data
// when the child gets a new one.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.updateChild
func (e *SliverMultiBoxAdaptorElement) UpdateChild(child Element, newWidget Widget, newSlot any) Element {
	var oldParentData *rendering.SliverMultiBoxAdaptorParentData
	if child != nil && child.RenderObject() != nil {
		oldParentData = rendering.SliverMultiBoxAdaptorParentDataOf(child.RenderObject())
	}
	newChild := e.RenderObjectElement.UpdateChild(child, newWidget, newSlot)
	var newParentData *rendering.SliverMultiBoxAdaptorParentData
	if newChild != nil && newChild.RenderObject() != nil {
		newParentData = rendering.SliverMultiBoxAdaptorParentDataOf(newChild.RenderObject())
	}
	if oldParentData != newParentData && oldParentData != nil && newParentData != nil {
		newParentData.LayoutOffset = oldParentData.LayoutOffset
	}
	return newChild
}

// ForgetChild removes the child from the live ones.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.forgetChild
func (e *SliverMultiBoxAdaptorElement) ForgetChild(child Element) {
	slot, _ := child.AsElement().slot.(int)
	e.childElements.remove(slot)
	e.RenderObjectElement.ForgetChild(child)
}

// RemoveChild removes the child that the render object gives up.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.removeChild
func (e *SliverMultiBoxAdaptorElement) RemoveChild(child rendering.Box) {
	index := e.sliver().IndexOf(child)
	if foundation.DebugMode && e.currentlyUpdatingChildIndex.IsSome() {
		panic(errors.New("SliverChildListLocked", "a child was removed while another one was being updated"))
	}
	e.owner.BuildScope(e, func() {
		func() {
			defer func() { e.currentlyUpdatingChildIndex = foundation.None[int]() }()
			e.currentlyUpdatingChildIndex = foundation.Some(index)
			e.UpdateChild(e.childElements.get(index), nil, index)
		}()
		e.childElements.remove(index)
	})
}

func extrapolateMaxScrollOffset(firstIndex, lastIndex int, leadingScrollOffset, trailingScrollOffset float64, childCount int) float64 {
	if lastIndex == childCount-1 {
		return trailingScrollOffset
	}
	reifiedCount := lastIndex - firstIndex + 1
	averageExtent := (trailingScrollOffset - leadingScrollOffset) / float64(reifiedCount)
	remainingCount := childCount - lastIndex - 1
	return trailingScrollOffset + averageExtent*float64(remainingCount)
}

// EstimateMaxScrollOffset estimates the scroll extent of the list.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.estimateMaxScrollOffset
func (e *SliverMultiBoxAdaptorElement) EstimateMaxScrollOffset(constraints rendering.SliverConstraints, opts rendering.EstimateMaxScrollOffsetOpts) float64 {
	childCount, ok := e.EstimatedChildCount().Get()
	if !ok {
		return math.Inf(1)
	}
	first, last := opts.FirstIndex.Value(), opts.LastIndex.Value()
	leading, trailing := opts.LeadingScrollOffset.Value(), opts.TrailingScrollOffset.Value()
	if v, ok := e.adaptor().EstimateMaxScrollOffset(constraints, first, last, leading, trailing).Get(); ok {
		return v
	}
	return extrapolateMaxScrollOffset(first, last, leading, trailing, childCount)
}

// EstimatedChildCount is the number of children that the delegate says.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.estimatedChildCount
func (e *SliverMultiBoxAdaptorElement) EstimatedChildCount() foundation.Opt[int] {
	return e.adaptor().Delegate().EstimatedChildCount()
}

// ChildCount is the number of children, found by a search if the delegate does
// not know it.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.childCount
func (e *SliverMultiBoxAdaptorElement) ChildCount() int {
	if result, ok := e.EstimatedChildCount().Get(); ok {
		return result
	}
	lo, hi := 0, 1
	const maxInt = int(^uint(0) >> 1)
	adaptor := e.adaptor()
	for e.build(hi-1, adaptor) != nil {
		lo = hi - 1
		switch {
		case hi < maxInt/2:
			hi *= 2
		case hi < maxInt:
			hi = maxInt
		default:
			panic(errors.New("SliverChildCountUnknown",
				"could not find the number of children in the delegate; the childCount getter was called (implying that the delegate's builder returned nil for a positive index), but even building the child with the maximum possible index did not return nil; consider implementing the child count to avoid the cost of searching for the final child"))
		}
	}
	for hi-lo > 1 {
		mid := (hi-lo)/2 + lo
		if e.build(mid-1, adaptor) == nil {
			hi = mid
		} else {
			lo = mid
		}
	}
	return lo
}

// DidStartLayout checks that the child list is not being changed.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.didStartLayout
func (e *SliverMultiBoxAdaptorElement) DidStartLayout() { e.DebugAssertChildListLocked() }

// DidFinishLayout tells the delegate the first and last live indices.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.didFinishLayout
func (e *SliverMultiBoxAdaptorElement) DidFinishLayout() {
	e.DebugAssertChildListLocked()
	first, _ := e.childElements.firstKey()
	last, _ := e.childElements.lastKey()
	e.adaptor().Delegate().DidFinishLayout(first, last)
}

// DebugAssertChildListLocked is true, and panics in debug builds if a child is
// being updated.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.debugAssertChildListLocked
func (e *SliverMultiBoxAdaptorElement) DebugAssertChildListLocked() bool {
	if foundation.DebugMode && e.currentlyUpdatingChildIndex.IsSome() {
		panic(errors.New("SliverChildListLocked", "the child list of a sliver was changed while the sliver lays out"))
	}
	return true
}

// DidAdoptChild records the index of the child in its parent data.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.didAdoptChild
func (e *SliverMultiBoxAdaptorElement) DidAdoptChild(child rendering.Box) {
	if foundation.DebugMode && e.currentlyUpdatingChildIndex.IsNone() {
		panic(errors.New("SliverChildAdoptedOutsideUpdate", "a child was adopted by a sliver while no child was being updated"))
	}
	rendering.SliverMultiBoxAdaptorParentDataOf(child).Index = e.currentlyUpdatingChildIndex
}

// SetDidUnderflow remembers that the list ran out of children.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.setDidUnderflow
func (e *SliverMultiBoxAdaptorElement) SetDidUnderflow(value bool) { e.didUnderflow = value }

// InsertRenderObjectChild inserts the child after the one before it.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.insertRenderObjectChild
func (e *SliverMultiBoxAdaptorElement) InsertRenderObjectChild(child rendering.RenderObject, slot any) {
	if foundation.DebugMode && e.currentlyUpdatingChildIndex != foundation.Some(slot.(int)) {
		panic(errors.New("SliverChildSlotMismatch", "a child was inserted into a slot other than the one being updated"))
	}
	e.sliver().Insert(child.(rendering.Box), e.currentBeforeChild)
}

// MoveRenderObjectChild moves the child after the one before it.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.moveRenderObjectChild
func (e *SliverMultiBoxAdaptorElement) MoveRenderObjectChild(child rendering.RenderObject, _, newSlot any) {
	if foundation.DebugMode && e.currentlyUpdatingChildIndex != foundation.Some(newSlot.(int)) {
		panic(errors.New("SliverChildSlotMismatch", "a child was moved into a slot other than the one being updated"))
	}
	e.sliver().Move(child.(rendering.Box), e.currentBeforeChild)
}

// RemoveRenderObjectChild removes the child.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.removeRenderObjectChild
func (e *SliverMultiBoxAdaptorElement) RemoveRenderObjectChild(child rendering.RenderObject, _ any) {
	if foundation.DebugMode && e.currentlyUpdatingChildIndex.IsNone() {
		panic(errors.New("SliverChildRemovedOutsideUpdate", "a child was removed from a sliver while no child was being updated"))
	}
	e.sliver().Remove(child.(rendering.Box))
}

// VisitChildren visits the live children in order of index.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.visitChildren
func (e *SliverMultiBoxAdaptorElement) VisitChildren(visitor ElementVisitor) {
	for _, index := range e.childElements.keys() {
		if child := e.childElements.get(index); child != nil {
			visitor(child)
		}
	}
}

// DebugVisitOnstageChildren visits the children that overlap the part of the
// sliver that is on the screen.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.debugVisitOnstageChildren
func (e *SliverMultiBoxAdaptorElement) DebugVisitOnstageChildren(visitor ElementVisitor) {
	constraints := e.sliver().SliverConstraints()
	for _, index := range e.childElements.keys() {
		child := e.childElements.get(index)
		if child == nil {
			continue
		}
		parentData := rendering.SliverMultiBoxAdaptorParentDataOf(child.RenderObject())
		bounds := child.RenderObject().PaintBounds()
		itemExtent := bounds.Height()
		if constraints.Axis() == painting.AxisHorizontal {
			itemExtent = bounds.Width()
		}
		if offset, ok := parentData.LayoutOffset.Get(); ok && offset < constraints.ScrollOffset+constraints.RemainingPaintExtent &&
			offset+itemExtent > constraints.ScrollOffset {
			visitor(child)
		}
	}
}
