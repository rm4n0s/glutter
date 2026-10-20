// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// fixedExtentHooks are the methods that the concrete slivers supply to
// RenderSliverFixedExtentBoxAdaptor.
type fixedExtentHooks interface {
	ItemExtent() foundation.Opt[float64]
	ItemExtentBuilder() ItemExtentBuilder
}

// RenderSliverFixedExtentBoxAdaptor is the base of the slivers that place box
// children of a known extent along the main axis: the extent is the same for all
// (ItemExtent) or comes from a builder (ItemExtentBuilder). The concrete type
// embeds it, calls InitRenderSliverFixedExtentBoxAdaptor, and implements
// ItemExtent and ItemExtentBuilder (exactly one of the two gives a value).
//
// Deviation: the deprecated itemExtent parameters of IndexToLayoutOffset,
// GetMinChildIndexForScrollOffset, GetMaxChildIndexForScrollOffset and
// ComputeMaxScrollOffset are not ported (SPEC §12.10).
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor
type RenderSliverFixedExtentBoxAdaptor struct {
	RenderSliverMultiBoxAdaptor

	currentLayoutDimensions foundation.Opt[SliverLayoutDimensions]
}

// InitRenderSliverFixedExtentBoxAdaptor sets the outer object, which must have
// ItemExtent and ItemExtentBuilder methods, and the child manager.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.new
func InitRenderSliverFixedExtentBoxAdaptor(a *RenderSliverFixedExtentBoxAdaptor, self Sliver, childManager RenderSliverBoxChildManager) {
	InitRenderSliverMultiBoxAdaptor(&a.RenderSliverMultiBoxAdaptor, self, childManager)
}

func (a *RenderSliverFixedExtentBoxAdaptor) hooks() fixedExtentHooks {
	return a.self.(fixedExtentHooks)
}

func (a *RenderSliverFixedExtentBoxAdaptor) fixedItemExtent() float64 {
	return a.hooks().ItemExtent().Value()
}

// IndexToLayoutOffset is the layout offset of the child at the index.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.indexToLayoutOffset
func (a *RenderSliverFixedExtentBoxAdaptor) IndexToLayoutOffset(index int) float64 {
	builder := a.hooks().ItemExtentBuilder()
	if builder == nil {
		return a.fixedItemExtent() * float64(index)
	}
	offset := 0.0
	for i := 0; i < index; i++ {
		if childCount, ok := a.childManager.EstimatedChildCount().Get(); ok && i > childCount-1 {
			break
		}
		extent, ok := builder(i, a.LayoutDimensions()).Get()
		if !ok {
			break
		}
		offset += extent
	}
	return offset
}

// GetMinChildIndexForScrollOffset is the index of the first child that is
// visible at the scroll offset.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.getMinChildIndexForScrollOffset
func (a *RenderSliverFixedExtentBoxAdaptor) GetMinChildIndexForScrollOffset(scrollOffset float64) int {
	builder := a.hooks().ItemExtentBuilder()
	if builder != nil {
		return a.getChildIndexForScrollOffset(scrollOffset, builder)
	}
	itemExtent := a.fixedItemExtent()
	if itemExtent > 0 {
		actual := scrollOffset / itemExtent
		round := math.Round(actual)
		if math.Abs(actual*itemExtent-round*itemExtent) < foundation.PrecisionErrorTolerance {
			return int(round)
		}
		return int(math.Floor(actual))
	}
	return 0
}

// GetMaxChildIndexForScrollOffset is the index of the last child that is visible
// at the scroll offset.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.getMaxChildIndexForScrollOffset
func (a *RenderSliverFixedExtentBoxAdaptor) GetMaxChildIndexForScrollOffset(scrollOffset float64) int {
	builder := a.hooks().ItemExtentBuilder()
	if builder != nil {
		return a.getChildIndexForScrollOffset(scrollOffset, builder)
	}
	itemExtent := a.fixedItemExtent()
	if itemExtent > 0 {
		actual := scrollOffset/itemExtent - 1
		round := math.Round(actual)
		if math.Abs(actual*itemExtent-round*itemExtent) < foundation.PrecisionErrorTolerance {
			return int(math.Max(0, round))
		}
		return int(math.Max(0, math.Ceil(actual)))
	}
	return 0
}

// EstimateMaxScrollOffset asks the child manager for the estimate.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.estimateMaxScrollOffset
func (a *RenderSliverFixedExtentBoxAdaptor) EstimateMaxScrollOffset(constraints SliverConstraints, opts EstimateMaxScrollOffsetOpts) float64 {
	return a.childManager.EstimateMaxScrollOffset(constraints, opts)
}

// ComputeMaxScrollOffset is the total extent of all the children, if they all
// were laid out.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.computeMaxScrollOffset
func (a *RenderSliverFixedExtentBoxAdaptor) ComputeMaxScrollOffset(SliverConstraints) float64 {
	builder := a.hooks().ItemExtentBuilder()
	if builder == nil {
		return float64(a.childManager.ChildCount()) * a.fixedItemExtent()
	}
	offset := 0.0
	for i := 0; i < a.childManager.ChildCount(); i++ {
		extent, ok := builder(i, a.LayoutDimensions()).Get()
		if !ok {
			break
		}
		offset += extent
	}
	return offset
}

func (a *RenderSliverFixedExtentBoxAdaptor) getChildIndexForScrollOffset(scrollOffset float64, callback ItemExtentBuilder) int {
	if scrollOffset == 0 {
		return 0
	}
	position := 0.0
	index := 0
	for position < scrollOffset {
		if childCount, ok := a.childManager.EstimatedChildCount().Get(); ok && index > childCount-1 {
			break
		}
		extent, ok := callback(index, a.LayoutDimensions()).Get()
		if !ok {
			break
		}
		position += extent
		index++
	}
	return index - 1
}

func (a *RenderSliverFixedExtentBoxAdaptor) getChildConstraints(index int) BoxConstraints {
	var extent float64
	if builder := a.hooks().ItemExtentBuilder(); builder == nil {
		extent = a.fixedItemExtent()
	} else {
		extent = builder(index, a.LayoutDimensions()).Value()
	}
	return a.SliverConstraints().AsBoxConstraints(AsBoxConstraintsOpts{MinExtent: extent, MaxExtent: foundation.Some(extent)})
}

// LayoutDimensions are the dimensions of the layout in progress, or of the
// constraints if there is none.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.layoutDimensions
func (a *RenderSliverFixedExtentBoxAdaptor) LayoutDimensions() SliverLayoutDimensions {
	if d, ok := a.currentLayoutDimensions.Get(); ok {
		return d
	}
	return a.dimensionsOfConstraints()
}

func (a *RenderSliverFixedExtentBoxAdaptor) dimensionsOfConstraints() SliverLayoutDimensions {
	c := a.SliverConstraints()
	return SliverLayoutDimensions{
		ScrollOffset:           c.ScrollOffset,
		PrecedingScrollExtent:  c.PrecedingScrollExtent,
		ViewportMainAxisExtent: c.ViewportMainAxisExtent,
		CrossAxisExtent:        c.CrossAxisExtent,
	}
}

// PaintExtentOf is the extent of the child along the main axis.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.paintExtentOf
func (a *RenderSliverFixedExtentBoxAdaptor) PaintExtentOf(child Box) float64 {
	builder := a.hooks().ItemExtentBuilder()
	if builder == nil {
		return a.fixedItemExtent()
	}
	return builder(a.IndexOf(child), a.LayoutDimensions()).Value()
}

// DebugAssertDoesMeetConstraints also checks that the scroll extent is a multiple
// of the item extent; it panics with the tag SliverFixedExtentNotMultiple
// otherwise.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.debugAssertDoesMeetConstraints
func (a *RenderSliverFixedExtentBoxAdaptor) DebugAssertDoesMeetConstraints() {
	a.RenderSliver.DebugAssertDoesMeetConstraints()
	if !foundation.DebugMode {
		return
	}
	g := a.Geometry()
	if a.hooks().ItemExtentBuilder() == nil && !math.IsInf(g.ScrollExtent, 0) && !math.IsNaN(g.ScrollExtent) {
		itemExtent := a.fixedItemExtent()
		count := g.ScrollExtent / itemExtent
		diff := math.Abs(math.Round(count) - count)
		if diff*itemExtent > foundation.PrecisionErrorTolerance && diff > foundation.PrecisionErrorTolerance {
			panic(errors.New("SliverFixedExtentNotMultiple",
				"RenderSliverFixedExtentBoxAdaptor.computeMaxScrollOffset() returned a value that is not an even multiple of its itemExtent. "+
					"The itemExtent was "+dartDouble(itemExtent)+", but the scrollExtent was "+dartDouble(g.ScrollExtent)+". "+
					"The difference was "+dartDouble(diff)+", which is greater than precisionErrorTolerance ("+dartDouble(foundation.PrecisionErrorTolerance)+").",
				"object", foundation.DescribeIdentity(a.self)))
		}
	}
}

// PerformLayout lays out the children that are in the viewport and the cache
// area, whose positions follow from their index.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.performLayout
func (a *RenderSliverFixedExtentBoxAdaptor) PerformLayout() {
	hooks := a.hooks()
	if foundation.DebugMode {
		hasExtent, hasBuilder := hooks.ItemExtent().IsSome(), hooks.ItemExtentBuilder() != nil
		if hasExtent == hasBuilder {
			panic(errors.New("SliverItemExtentInvalid", "exactly one of the itemExtent and the itemExtentBuilder of a fixed extent sliver must be given",
				"object", foundation.DescribeIdentity(a.self)))
		}
		if !hasBuilder {
			if e := hooks.ItemExtent().Value(); math.IsInf(e, 0) || math.IsNaN(e) || e < 0 {
				panic(errors.New("SliverItemExtentInvalid", "the itemExtent of a fixed extent sliver must be finite and not negative",
					"object", foundation.DescribeIdentity(a.self), "itemExtent", e))
			}
		}
	}

	constraints := a.SliverConstraints()
	a.childManager.DidStartLayout()
	a.childManager.SetDidUnderflow(false)

	scrollOffset := constraints.ScrollOffset + constraints.CacheOrigin
	remainingExtent := constraints.RemainingCacheExtent
	targetEndScrollOffset := scrollOffset + remainingExtent

	a.currentLayoutDimensions = foundation.Some(a.dimensionsOfConstraints())

	firstIndex := a.GetMinChildIndexForScrollOffset(scrollOffset)
	targetLastIndex := foundation.None[int]()
	if !math.IsInf(targetEndScrollOffset, 0) {
		targetLastIndex = foundation.Some(a.GetMaxChildIndexForScrollOffset(targetEndScrollOffset))
	}

	if !isNil(a.FirstChild()) {
		leadingGarbage := a.CalculateLeadingGarbage(firstIndex)
		trailingGarbage := 0
		if last, ok := targetLastIndex.Get(); ok {
			trailingGarbage = a.CalculateTrailingGarbage(last)
		}
		a.CollectGarbage(leadingGarbage, trailingGarbage)
	} else {
		a.CollectGarbage(0, 0)
	}

	if isNil(a.FirstChild()) {
		layoutOffset := a.IndexToLayoutOffset(firstIndex)
		if !a.AddInitialChild(AddInitialChildOpts{Index: firstIndex, LayoutOffset: layoutOffset}) {
			// There are either no children, or we are past the end of all our children.
			max := 0.0
			if firstIndex > 0 {
				max = a.ComputeMaxScrollOffset(constraints)
			}
			a.SetGeometry(NewSliverGeometry(SliverGeometryOpts{ScrollExtent: max, MaxPaintExtent: max}))
			a.childManager.DidFinishLayout()
			return
		}
	}

	var trailingChildWithLayout Box

	for index := a.IndexOf(a.FirstChild()) - 1; index >= firstIndex; index-- {
		child := a.InsertAndLayoutLeadingChild(a.getChildConstraints(index))
		if isNil(child) {
			// Items before the previously first child are no longer present.
			// Reset the scroll offset to offset all items prior and up to the
			// missing item. Let parent re-layout everything.
			a.SetGeometry(NewSliverGeometry(SliverGeometryOpts{ScrollOffsetCorrection: foundation.Some(a.IndexToLayoutOffset(index))}))
			return
		}
		SliverMultiBoxAdaptorParentDataOf(child).LayoutOffset = foundation.Some(a.IndexToLayoutOffset(index))
		if isNil(trailingChildWithLayout) {
			trailingChildWithLayout = child
		}
	}

	if isNil(trailingChildWithLayout) {
		a.FirstChild().Layout(a.getChildConstraints(a.IndexOf(a.FirstChild())))
		SliverMultiBoxAdaptorParentDataOf(a.FirstChild()).LayoutOffset = foundation.Some(a.IndexToLayoutOffset(firstIndex))
		trailingChildWithLayout = a.FirstChild()
	}

	estimatedMaxScrollOffset := math.Inf(1)
	for index := a.IndexOf(trailingChildWithLayout) + 1; !targetLastIndex.IsSome() || index <= targetLastIndex.Value(); index++ {
		child := a.ChildAfter(trailingChildWithLayout)
		if isNil(child) || a.IndexOf(child) != index {
			child = a.InsertAndLayoutChild(a.getChildConstraints(index), trailingChildWithLayout)
			if isNil(child) {
				// We have run out of children.
				estimatedMaxScrollOffset = a.IndexToLayoutOffset(index)
				break
			}
		} else {
			child.Layout(a.getChildConstraints(index))
		}
		trailingChildWithLayout = child
		data := SliverMultiBoxAdaptorParentDataOf(child)
		data.LayoutOffset = foundation.Some(a.IndexToLayoutOffset(data.Index.Value()))
	}

	lastIndex := a.IndexOf(a.LastChild())
	leadingScrollOffset := a.IndexToLayoutOffset(firstIndex)
	trailingScrollOffset := a.IndexToLayoutOffset(lastIndex + 1)

	a.DebugAssertChildListIsNonEmptyAndContiguous()

	estimatedMaxScrollOffset = math.Min(estimatedMaxScrollOffset, a.EstimateMaxScrollOffset(constraints, EstimateMaxScrollOffsetOpts{
		FirstIndex:           foundation.Some(firstIndex),
		LastIndex:            foundation.Some(lastIndex),
		LeadingScrollOffset:  foundation.Some(leadingScrollOffset),
		TrailingScrollOffset: foundation.Some(trailingScrollOffset),
	}))

	paintExtent := a.CalculatePaintOffset(constraints, leadingScrollOffset, trailingScrollOffset)
	cacheExtent := a.CalculateCacheOffset(constraints, leadingScrollOffset, trailingScrollOffset)

	targetEndScrollOffsetForPaint := constraints.ScrollOffset + constraints.RemainingPaintExtent
	hasVisualOverflow := constraints.ScrollOffset > 0
	if !math.IsInf(targetEndScrollOffsetForPaint, 0) {
		if lastIndex >= a.GetMaxChildIndexForScrollOffset(targetEndScrollOffsetForPaint) {
			hasVisualOverflow = true
		}
	}

	a.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent:   estimatedMaxScrollOffset,
		PaintExtent:    paintExtent,
		CacheExtent:    foundation.Some(cacheExtent),
		MaxPaintExtent: estimatedMaxScrollOffset,
		// Conservative to avoid flickering away the clip during scroll.
		HasVisualOverflow: hasVisualOverflow,
	}))

	// We may have started the layout while scrolled to the end, which would not
	// expose a new child.
	if estimatedMaxScrollOffset == trailingScrollOffset {
		a.childManager.SetDidUnderflow(true)
	}
	a.childManager.DidFinishLayout()
}

// RenderSliverFixedExtentList is a sliver that places multiple box children with
// the same extent in the main axis in a linear array.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentList
type RenderSliverFixedExtentList struct {
	RenderSliverFixedExtentBoxAdaptor
	itemExtent float64
}

// NewRenderSliverFixedExtentList creates the sliver.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentList.new
func NewRenderSliverFixedExtentList(childManager RenderSliverBoxChildManager, itemExtent float64) *RenderSliverFixedExtentList {
	r := &RenderSliverFixedExtentList{itemExtent: itemExtent}
	InitRenderSliverFixedExtentBoxAdaptor(&r.RenderSliverFixedExtentBoxAdaptor, r, childManager)
	return r
}

// ItemExtent is the main axis extent of every child.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentList.itemExtent
func (r *RenderSliverFixedExtentList) ItemExtent() foundation.Opt[float64] {
	return foundation.Some(r.itemExtent)
}

// ItemExtentBuilder is nil.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.itemExtentBuilder
func (r *RenderSliverFixedExtentList) ItemExtentBuilder() ItemExtentBuilder { return nil }

// SetItemExtent changes the extent and marks the sliver as needing layout.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentList.itemExtent
func (r *RenderSliverFixedExtentList) SetItemExtent(value float64) {
	if r.itemExtent == value {
		return
	}
	r.itemExtent = value
	r.MarkNeedsLayout()
}

// RenderSliverVariedExtentList is a sliver that places multiple box children,
// whose main axis extents the builder gives, in a linear array.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverVariedExtentList
type RenderSliverVariedExtentList struct {
	RenderSliverFixedExtentBoxAdaptor
	itemExtentBuilder ItemExtentBuilder
}

// NewRenderSliverVariedExtentList creates the sliver.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverVariedExtentList.new
func NewRenderSliverVariedExtentList(childManager RenderSliverBoxChildManager, itemExtentBuilder ItemExtentBuilder) *RenderSliverVariedExtentList {
	r := &RenderSliverVariedExtentList{itemExtentBuilder: itemExtentBuilder}
	InitRenderSliverFixedExtentBoxAdaptor(&r.RenderSliverFixedExtentBoxAdaptor, r, childManager)
	return r
}

// ItemExtentBuilder is the function that gives the extent of a child.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverVariedExtentList.itemExtentBuilder
func (r *RenderSliverVariedExtentList) ItemExtentBuilder() ItemExtentBuilder {
	return r.itemExtentBuilder
}

// SetItemExtentBuilder changes the builder and marks the sliver as needing
// layout. Go can not compare functions, so it always does.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverVariedExtentList.itemExtentBuilder
func (r *RenderSliverVariedExtentList) SetItemExtentBuilder(value ItemExtentBuilder) {
	r.itemExtentBuilder = value
	r.MarkNeedsLayout()
}

// ItemExtent is none.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverVariedExtentList.itemExtent
func (r *RenderSliverVariedExtentList) ItemExtent() foundation.Opt[float64] {
	return foundation.None[float64]()
}
