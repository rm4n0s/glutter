// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"github.com/rm4n0s/glutter/foundation"
)

// RenderSliverList is a sliver that places multiple box children in a linear
// array along the main axis, lazily: it asks the child manager for the children
// that are in the viewport and the cache area.
//
// Flutter: rendering/sliver_list.dart RenderSliverList
type RenderSliverList struct {
	RenderSliverMultiBoxAdaptor
}

// NewRenderSliverList creates the sliver.
//
// Flutter: rendering/sliver_list.dart RenderSliverList.new
func NewRenderSliverList(childManager RenderSliverBoxChildManager) *RenderSliverList {
	r := &RenderSliverList{}
	InitRenderSliverMultiBoxAdaptor(&r.RenderSliverMultiBoxAdaptor, r, childManager)
	return r
}

// PerformLayout lays out the children from the first one that is needed to the
// last one that fits in the cache area, creating them as needed and collecting
// the ones that are not.
//
// Flutter: rendering/sliver_list.dart RenderSliverList.performLayout
func (r *RenderSliverList) PerformLayout() {
	constraints := r.SliverConstraints()
	r.childManager.DidStartLayout()
	r.childManager.SetDidUnderflow(false)

	scrollOffset := constraints.ScrollOffset + constraints.CacheOrigin
	remainingExtent := constraints.RemainingCacheExtent
	targetEndScrollOffset := scrollOffset + remainingExtent
	childConstraints := constraints.AsBoxConstraints()
	leadingGarbage, trailingGarbage := 0, 0
	reachedEnd := false

	// This algorithm in principle is straight-forward: find the first child that
	// overlaps the given scrollOffset, creating more children at the beginning of
	// the list if necessary.
	if isNil(r.FirstChild()) {
		if !r.AddInitialChild() {
			// There are no children.
			r.SetGeometry(SliverGeometryZero)
			r.childManager.DidFinishLayout()
			return
		}
	}

	// We have at least one child.

	// These variables track the range of children that we have laid out. Within
	// this range, the children have consecutive indices. Outside this range,
	// it's possible for a child to get removed without notice.
	var leadingChildWithLayout, trailingChildWithLayout Box

	earliestUsefulChild := r.FirstChild()

	// A firstChild with null layout offset is likely a result of children
	// reordering.
	//
	// We rely on firstChild to have accurate layout offset. In the case of null
	// layout offset, we have to find the first child that has valid layout offset.
	if !r.ChildScrollOffset(r.FirstChild()).IsSome() {
		leadingChildrenWithoutLayoutOffset := 0
		for !isNil(earliestUsefulChild) && !r.ChildScrollOffset(earliestUsefulChild).IsSome() {
			earliestUsefulChild = r.ChildAfter(earliestUsefulChild)
			leadingChildrenWithoutLayoutOffset++
		}
		// We should be able to destroy children with null layout offset safely,
		// because they are likely outside of viewport.
		r.CollectGarbage(leadingChildrenWithoutLayoutOffset, 0)
		// If can not find a valid layout offset, start from the initial child.
		if isNil(r.FirstChild()) {
			if !r.AddInitialChild() {
				// There are no children.
				r.SetGeometry(SliverGeometryZero)
				r.childManager.DidFinishLayout()
				return
			}
		}
	}

	// Find the last child that is at or before the scrollOffset.
	earliestUsefulChild = r.FirstChild()
	for earliestScrollOffset := r.ChildScrollOffset(earliestUsefulChild).Value(); earliestScrollOffset > scrollOffset; earliestScrollOffset = r.ChildScrollOffset(earliestUsefulChild).Value() {
		// We have to add children before the earliestUsefulChild.
		earliestUsefulChild = r.InsertAndLayoutLeadingChild(childConstraints, LayoutOpts{ParentUsesSize: true})
		if isNil(earliestUsefulChild) {
			// We ran out of children before reaching the scroll offset. We must
			// inform our parent that this sliver cannot fulfill its contract and
			// that we need a scroll offset correction.
			SliverMultiBoxAdaptorParentDataOf(r.FirstChild()).LayoutOffset = foundation.Some(0.0)

			if scrollOffset == 0 {
				// insertAndLayoutLeadingChild only lays out the children before
				// firstChild. In this case, nothing has been laid out. We have to
				// lay out firstChild manually.
				r.FirstChild().Layout(childConstraints, LayoutOpts{ParentUsesSize: true})
				earliestUsefulChild = r.FirstChild()
				leadingChildWithLayout = earliestUsefulChild
				if isNil(trailingChildWithLayout) {
					trailingChildWithLayout = earliestUsefulChild
				}
				break
			}
			// We ran out of children before reaching the scroll offset. We must
			// inform our parent that this sliver cannot fulfill its contract and
			// that we need a scroll offset correction.
			r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{ScrollOffsetCorrection: foundation.Some(-scrollOffset)}))
			return
		}

		firstChildScrollOffset := earliestScrollOffset - r.PaintExtentOf(r.FirstChild())
		// firstChildScrollOffset may contain double precision error
		if firstChildScrollOffset < -foundation.PrecisionErrorTolerance {
			// Let's assume there is no child before the first child. We will
			// correct it on the next layout if it is not.
			r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{ScrollOffsetCorrection: foundation.Some(-firstChildScrollOffset)}))
			// We want to make sure we keep the reference to the child that we
			// just inserted: it will be needed the next time.
			SliverMultiBoxAdaptorParentDataOf(r.FirstChild()).LayoutOffset = foundation.Some(0.0)
			return
		}

		SliverMultiBoxAdaptorParentDataOf(earliestUsefulChild).LayoutOffset = foundation.Some(firstChildScrollOffset)
		leadingChildWithLayout = earliestUsefulChild
		if isNil(trailingChildWithLayout) {
			trailingChildWithLayout = earliestUsefulChild
		}
	}

	// If the scroll offset is at zero, we should make sure we are actually at the
	// beginning of the list.
	if scrollOffset < foundation.PrecisionErrorTolerance {
		// We iterate from the firstChild in case the leading child has a 0 paint
		// extent.
		for r.IndexOf(r.FirstChild()) > 0 {
			earliestScrollOffset := r.ChildScrollOffset(r.FirstChild()).Value()
			// We correct one child at a time. If there are more children before
			// the earliestUsefulChild, we will correct it once the scroll offset
			// reaches zero again.
			earliestUsefulChild = r.InsertAndLayoutLeadingChild(childConstraints, LayoutOpts{ParentUsesSize: true})
			firstChildScrollOffset := earliestScrollOffset - r.PaintExtentOf(r.FirstChild())
			SliverMultiBoxAdaptorParentDataOf(r.FirstChild()).LayoutOffset = foundation.Some(0.0)
			if firstChildScrollOffset < -foundation.PrecisionErrorTolerance {
				// The first child is not at the zero offset: correct it.
				r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{ScrollOffsetCorrection: foundation.Some(-firstChildScrollOffset)}))
				return
			}
		}
	}

	// At this point, earliestUsefulChild is the first child, and is a child whose
	// scroll offset is at or before the scrollOffset, and
	// leadingChildWithLayout and trailingChildWithLayout are either null or
	// cover a range of render boxes that we have laid out with the first being
	// the same as earliestUsefulChild and the last being either at or after the
	// scroll offset.

	// Make sure we've laid out at least one child.
	if isNil(leadingChildWithLayout) {
		earliestUsefulChild.Layout(childConstraints, LayoutOpts{ParentUsesSize: true})
		trailingChildWithLayout = earliestUsefulChild
	}

	// Here, earliestUsefulChild is still the first child, it's got a scroll
	// offset that is at or before our actual scroll offset, and it has been laid
	// out, and is in fact our leadingChildWithLayout. It's possible that some
	// children beyond that one have also been laid out.

	inLayoutRange := true
	child := earliestUsefulChild
	index := r.IndexOf(child)
	endScrollOffset := r.ChildScrollOffset(child).Value() + r.PaintExtentOf(child)
	advance := func() bool {
		// returns true if we advanced, false if we have no more children
		// This function is used in two different places below, to avoid code
		// duplication.
		if child == trailingChildWithLayout {
			inLayoutRange = false
		}
		child = r.ChildAfter(child)
		if isNil(child) {
			inLayoutRange = false
		}
		index++
		if !inLayoutRange {
			if isNil(child) || r.IndexOf(child) != index {
				// We are missing a child. Insert it (and lay it out) if possible.
				child = r.InsertAndLayoutChild(childConstraints, trailingChildWithLayout, LayoutOpts{ParentUsesSize: true})
				if isNil(child) {
					// We have run out of children.
					return false
				}
			} else {
				// Lay out the child.
				child.Layout(childConstraints, LayoutOpts{ParentUsesSize: true})
			}
			trailingChildWithLayout = child
		}
		SliverMultiBoxAdaptorParentDataOf(child).LayoutOffset = foundation.Some(endScrollOffset)
		endScrollOffset = r.ChildScrollOffset(child).Value() + r.PaintExtentOf(child)
		return true
	}

	// Find the first child that ends after the scroll offset.
	for endScrollOffset < scrollOffset {
		leadingGarbage++
		if !advance() {
			// we want to make sure we keep the last child around so we know the
			// end scroll offset
			r.CollectGarbage(leadingGarbage-1, 0)
			extent := r.ChildScrollOffset(r.LastChild()).Value() + r.PaintExtentOf(r.LastChild())
			r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{ScrollExtent: extent, MaxPaintExtent: extent}))
			return
		}
	}

	// Now find the first child that ends after our end.
	for endScrollOffset < targetEndScrollOffset {
		if !advance() {
			reachedEnd = true
			break
		}
	}

	// Finally count up all the remaining children and label them as garbage.
	if !isNil(child) {
		child = r.ChildAfter(child)
		for !isNil(child) {
			trailingGarbage++
			child = r.ChildAfter(child)
		}
	}

	// At this point everything should be good to go, we just have to clean up the
	// garbage and report the geometry.
	r.CollectGarbage(leadingGarbage, trailingGarbage)

	r.DebugAssertChildListIsNonEmptyAndContiguous()
	var estimatedMaxScrollOffset float64
	if reachedEnd {
		estimatedMaxScrollOffset = endScrollOffset
	} else {
		estimatedMaxScrollOffset = r.childManager.EstimateMaxScrollOffset(constraints, EstimateMaxScrollOffsetOpts{
			FirstIndex:           foundation.Some(r.IndexOf(r.FirstChild())),
			LastIndex:            foundation.Some(r.IndexOf(r.LastChild())),
			LeadingScrollOffset:  r.ChildScrollOffset(r.FirstChild()),
			TrailingScrollOffset: foundation.Some(endScrollOffset),
		})
	}

	firstOffset := r.ChildScrollOffset(r.FirstChild()).Value()
	paintExtent := r.CalculatePaintOffset(constraints, firstOffset, endScrollOffset)
	cacheExtent := r.CalculateCacheOffset(constraints, firstOffset, endScrollOffset)
	targetEndScrollOffsetForPaint := constraints.ScrollOffset + constraints.RemainingPaintExtent
	r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent:   estimatedMaxScrollOffset,
		PaintExtent:    paintExtent,
		CacheExtent:    foundation.Some(cacheExtent),
		MaxPaintExtent: estimatedMaxScrollOffset,
		// Conservative to avoid flickering away the clip during scroll.
		HasVisualOverflow: endScrollOffset > targetEndScrollOffsetForPaint || constraints.ScrollOffset > 0,
	}))

	// We may have started the layout while scrolled to the end, which would not
	// expose a new child.
	if estimatedMaxScrollOffset == endScrollOffset {
		r.childManager.SetDidUnderflow(true)
	}
	r.childManager.DidFinishLayout()
}
