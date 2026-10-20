// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
)

// SliverPrototypeExtentList is a sliver that places its box children in a linear
// array and constrains them to have the same extent as a prototype item along
// the main axis.
//
// Flutter: widgets/sliver_prototype_extent_list.dart SliverPrototypeExtentList
type SliverPrototypeExtentList struct {
	SliverMultiBoxAdaptorWidget
	prototypeItem Widget
}

// NewSliverPrototypeExtentList creates a sliver list whose children the delegate
// supplies, with the main axis extent of the prototype item.
//
// Flutter: widgets/sliver_prototype_extent_list.dart SliverPrototypeExtentList.new
func NewSliverPrototypeExtentList(delegate SliverChildDelegate, prototypeItem Widget, opts ...SliverListWidgetOpts) *SliverPrototypeExtentList {
	return &SliverPrototypeExtentList{NewSliverMultiBoxAdaptorWidgetBase(delegate, WidgetOpts{Key: firstOpts(opts).Key}), prototypeItem}
}

// NewSliverPrototypeExtentListBuilder creates a prototype extent sliver list
// whose items are built lazily.
//
// Flutter: widgets/sliver_prototype_extent_list.dart SliverPrototypeExtentList.builder
func NewSliverPrototypeExtentListBuilder(itemBuilder NullableIndexedWidgetBuilder, prototypeItem Widget, opts ...SliverBuilderOpts) *SliverPrototypeExtentList {
	o := firstOpts(opts)
	return NewSliverPrototypeExtentList(o.delegate(itemBuilder), prototypeItem, SliverListWidgetOpts{Key: o.Key})
}

// NewSliverPrototypeExtentListList creates a prototype extent sliver list of the
// children.
//
// Flutter: widgets/sliver_prototype_extent_list.dart SliverPrototypeExtentList.list
func NewSliverPrototypeExtentListList(prototypeItem Widget, children []Widget, opts ...SliverListOpts) *SliverPrototypeExtentList {
	o := firstOpts(opts)
	return NewSliverPrototypeExtentList(o.delegate(children), prototypeItem, SliverListWidgetOpts{Key: o.Key})
}

// PrototypeItem is the widget that gives the main axis extent of all the items.
//
// Flutter: widgets/sliver_prototype_extent_list.dart SliverPrototypeExtentList.prototypeItem
func (w *SliverPrototypeExtentList) PrototypeItem() Widget { return w.prototypeItem }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/sliver_prototype_extent_list.dart SliverPrototypeExtentList.createRenderObject
func (w *SliverPrototypeExtentList) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	r := &renderSliverPrototypeExtentList{}
	rendering.InitRenderSliverFixedExtentBoxAdaptor(&r.RenderSliverFixedExtentBoxAdaptor, r, bc.(rendering.RenderSliverBoxChildManager))
	return r
}

// CreateElement creates the element, which also builds the prototype.
//
// Flutter: widgets/sliver_prototype_extent_list.dart SliverPrototypeExtentList.createElement
func (w *SliverPrototypeExtentList) CreateElement() Element {
	e := &sliverPrototypeExtentListElement{}
	InitSliverMultiBoxAdaptorElement(&e.SliverMultiBoxAdaptorElement, e, w, false)
	return e
}

// prototypeSlot is the slot of the prototype in its element.
//
// Flutter: widgets/sliver_prototype_extent_list.dart _SliverPrototypeExtentListElement._prototypeSlot
type prototypeSlotType struct{}

var prototypeSlot = &prototypeSlotType{}

// sliverPrototypeExtentListElement is the element of a SliverPrototypeExtentList.
//
// Flutter: widgets/sliver_prototype_extent_list.dart _SliverPrototypeExtentListElement
type sliverPrototypeExtentListElement struct {
	SliverMultiBoxAdaptorElement
	prototype Element
}

func (e *sliverPrototypeExtentListElement) render() *renderSliverPrototypeExtentList {
	return e.renderObject.(*renderSliverPrototypeExtentList)
}

// InsertRenderObjectChild makes the prototype the child of the render object, and
// any other child one of its list.
func (e *sliverPrototypeExtentListElement) InsertRenderObjectChild(child rendering.RenderObject, slot any) {
	if slot == any(prototypeSlot) {
		e.render().setChild(child.(rendering.Box))
		return
	}
	e.SliverMultiBoxAdaptorElement.InsertRenderObjectChild(child, slot)
}

// DidAdoptChild ignores the prototype.
func (e *sliverPrototypeExtentListElement) DidAdoptChild(child rendering.Box) {
	if child != e.render().child {
		e.SliverMultiBoxAdaptorElement.DidAdoptChild(child)
	}
}

// MoveRenderObjectChild moves a child of the list.
func (e *sliverPrototypeExtentListElement) MoveRenderObjectChild(child rendering.RenderObject, oldSlot, newSlot any) {
	if newSlot == any(prototypeSlot) {
		panic(foundation.RuntimeTypeName(e) + ": the prototype can not move")
	}
	e.SliverMultiBoxAdaptorElement.MoveRenderObjectChild(child, oldSlot, newSlot)
}

// RemoveRenderObjectChild removes the prototype or a child of the list.
func (e *sliverPrototypeExtentListElement) RemoveRenderObjectChild(child rendering.RenderObject, slot any) {
	if e.render().child == child {
		e.render().setChild(nil)
		return
	}
	e.SliverMultiBoxAdaptorElement.RemoveRenderObjectChild(child, slot)
}

// VisitChildren visits the prototype, then the children of the list.
func (e *sliverPrototypeExtentListElement) VisitChildren(visitor ElementVisitor) {
	if e.prototype != nil {
		visitor(e.prototype)
	}
	e.SliverMultiBoxAdaptorElement.VisitChildren(visitor)
}

// Mount builds the prototype.
func (e *sliverPrototypeExtentListElement) Mount(parent Element, newSlot any) {
	e.SliverMultiBoxAdaptorElement.Mount(parent, newSlot)
	e.prototype = e.UpdateChild(e.prototype, e.widget.(*SliverPrototypeExtentList).prototypeItem, prototypeSlot)
}

// Update builds the prototype again.
func (e *sliverPrototypeExtentListElement) Update(newWidget Widget) {
	e.SliverMultiBoxAdaptorElement.Update(newWidget)
	e.prototype = e.UpdateChild(e.prototype, e.widget.(*SliverPrototypeExtentList).prototypeItem, prototypeSlot)
}

// ForgetChild drops the prototype or a child of the list.
func (e *sliverPrototypeExtentListElement) ForgetChild(child Element) {
	if child == e.prototype {
		e.prototype = nil
		e.RenderObjectElement.ForgetChild(child)
		return
	}
	e.SliverMultiBoxAdaptorElement.ForgetChild(child)
}

// renderSliverPrototypeExtentList is the render object of a
// SliverPrototypeExtentList: the extent of its items is the one of its prototype.
//
// Flutter: widgets/sliver_prototype_extent_list.dart _RenderSliverPrototypeExtentList
type renderSliverPrototypeExtentList struct {
	rendering.RenderSliverFixedExtentBoxAdaptor
	child rendering.Box
}

func (r *renderSliverPrototypeExtentList) setChild(value rendering.Box) {
	if r.child != nil {
		r.DropChild(r.child)
	}
	r.child = value
	if r.child != nil {
		r.AdoptChild(r.child)
	}
	r.MarkNeedsLayout()
}

func (r *renderSliverPrototypeExtentList) PerformLayout() {
	r.child.Layout(r.SliverConstraints().AsBoxConstraints(), rendering.LayoutOpts{ParentUsesSize: true})
	r.RenderSliverFixedExtentBoxAdaptor.PerformLayout()
}

func (r *renderSliverPrototypeExtentList) Attach(owner *rendering.PipelineOwner) {
	r.RenderSliverFixedExtentBoxAdaptor.Attach(owner)
	if r.child != nil {
		r.child.Attach(owner)
	}
}

func (r *renderSliverPrototypeExtentList) Detach() {
	r.RenderSliverFixedExtentBoxAdaptor.Detach()
	if r.child != nil {
		r.child.Detach()
	}
}

func (r *renderSliverPrototypeExtentList) RedepthChildren() {
	if r.child != nil {
		r.RedepthChild(r.child)
	}
	r.RenderSliverFixedExtentBoxAdaptor.RedepthChildren()
}

func (r *renderSliverPrototypeExtentList) VisitChildren(visitor rendering.RenderObjectVisitor) {
	if r.child != nil {
		visitor(r.child)
	}
	r.RenderSliverFixedExtentBoxAdaptor.VisitChildren(visitor)
}

// ItemExtent is the extent of the prototype along the main axis.
func (r *renderSliverPrototypeExtentList) ItemExtent() foundation.Opt[float64] {
	size := r.child.Size()
	if r.SliverConstraints().Axis() == painting.AxisVertical {
		return foundation.Some(size.Height)
	}
	return foundation.Some(size.Width)
}

// ItemExtentBuilder is nil.
func (r *renderSliverPrototypeExtentList) ItemExtentBuilder() rendering.ItemExtentBuilder { return nil }
