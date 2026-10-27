package rendering

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
)

// RenderSliverList is a sliver that places multiple box children in a linear
// array along the main axis, lazily: it asks the child manager for the children
// that are in the viewport and the cache area.
//
// Flutter: rendering/sliver_list.dart RenderSliverList
type RenderSliverList struct {
	RenderSliverMultiBoxAdaptor

	// What the sliver measured, for the EstimateItemOffsets option: the sum and
	// the number of the extents of the children that came into the layout, and
	// the indices of the first and the last child of the last layout (both -1
	// before the first).
	measuredSum                 float64
	measuredCount               int
	measuredFirst, measuredLast int
}

// NewRenderSliverList creates the sliver.
//
// Flutter: rendering/sliver_list.dart RenderSliverList.new
func NewRenderSliverList(childManager RenderSliverBoxChildManager) *RenderSliverList {
	r := &RenderSliverList{measuredFirst: -1, measuredLast: -1}
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

	estimator, estimates := r.childManager.(SliverItemOffsetEstimator)
	estimates = estimates && estimator.EstimatesItemOffsets()
	if estimates && !r.jumpToEstimatedChild(constraints, scrollOffset, targetEndScrollOffset) {
		// The manager has no child at the estimated index: no children are left.
		r.SetGeometry(SliverGeometryZero)
		r.childManager.DidFinishLayout()
		return
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
			if estimates {
				// Deviation: the offsets are estimates (the EstimateItemOffsets
				// option), so the children before this one are not known to be
				// few: all of them are inserted in this pass, and the scroll
				// offset is corrected once, by the offset of the first item, not
				// by the extent of one child for each pass.
				for r.IndexOf(r.FirstChild()) > 0 {
					leading := r.InsertAndLayoutLeadingChild(childConstraints, LayoutOpts{ParentUsesSize: true})
					if isNil(leading) {
						break
					}
					firstChildScrollOffset -= r.PaintExtentOf(r.FirstChild())
				}
			}
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
	keeper, isKeeper := r.childManager.(SliverVisibleContentKeeper)
	if (isKeeper && keeper.KeepsVisibleContentPosition() || estimates) &&
		scrollOffset < foundation.PrecisionErrorTolerance {
		// Deviation: when items were inserted before the children that were kept in
		// place (the MaintainVisibleContentPosition option), or the offsets are
		// estimates (the EstimateItemOffsets option), all of the children
		// before the first one are inserted in this pass and the scroll offset is
		// corrected once, by their extent less the offset of the first child. One
		// child for each pass, as below, would take more passes than a viewport
		// allows when many items were inserted.
		firstChildScrollOffset := r.ChildScrollOffset(r.FirstChild()).Value()
		inserted := false
		for r.IndexOf(r.FirstChild()) > 0 {
			leading := r.InsertAndLayoutLeadingChild(childConstraints, LayoutOpts{ParentUsesSize: true})
			if isNil(leading) {
				break
			}
			earliestUsefulChild, inserted = leading, true
			firstChildScrollOffset -= r.PaintExtentOf(r.FirstChild())
		}
		if inserted {
			SliverMultiBoxAdaptorParentDataOf(r.FirstChild()).LayoutOffset = foundation.Some(0.0)
			// At the very start of the sliver (a jump to offset 0) the user asks for
			// its first child: the children are laid out from it, with no correction.
			if firstChildScrollOffset < -foundation.PrecisionErrorTolerance && constraints.ScrollOffset > foundation.PrecisionErrorTolerance {
				r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{ScrollOffsetCorrection: foundation.Some(-firstChildScrollOffset)}))
				return
			}
		}
	} else if scrollOffset < foundation.PrecisionErrorTolerance {
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
	if estimates {
		r.recordMeasuredExtents()
	}
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

// recordMeasuredExtents adds the extents of the children that were not in the
// last layout to the average that the EstimateItemOffsets option estimates with.
// All of the children are laid out when it is called.
func (r *RenderSliverList) recordMeasuredExtents() {
	first, last := r.FirstChild(), r.LastChild()
	if isNil(first) || isNil(last) {
		return
	}
	for child := first; !isNil(child); child = r.ChildAfter(child) {
		index := r.IndexOf(child)
		if index >= r.measuredFirst && index <= r.measuredLast || !child.HasSize() {
			continue
		}
		r.measuredSum += r.PaintExtentOf(child)
		r.measuredCount++
	}
	r.measuredFirst, r.measuredLast = r.IndexOf(first), r.IndexOf(last)
}

// jumpToEstimatedChild is the EstimateItemOffsets option. When the window that
// the layout has to fill (the viewport and the cache area) starts or ends more
// than a window away from the children that the sliver has, it drops them and
// creates the child at the index that the average extent gives for the scroll
// offset, with the offset that the average gives, so the layout goes on from
// there and does not walk through the children in between. It returns false
// only if the manager has no child at all.
func (r *RenderSliverList) jumpToEstimatedChild(constraints SliverConstraints, scrollOffset, targetEndScrollOffset float64) bool {
	if r.measuredCount == 0 {
		return true
	}
	first, last := r.FirstChild(), r.LastChild()
	if isNil(first) || isNil(last) || !first.HasSize() || !last.HasSize() ||
		!r.ChildScrollOffset(first).IsSome() || !r.ChildScrollOffset(last).IsSome() {
		return true
	}
	window := max(constraints.ViewportMainAxisExtent, constraints.RemainingCacheExtent)
	firstOffset := r.ChildScrollOffset(first).Value()
	lastEnd := r.ChildScrollOffset(last).Value() + r.PaintExtentOf(last)
	if scrollOffset <= lastEnd+window && targetEndScrollOffset >= firstOffset-window {
		return true
	}
	average := r.measuredSum / float64(r.measuredCount)
	if average <= 0 {
		return true
	}
	index := int(math.Floor(scrollOffset / average))
	if count, ok := r.childManager.EstimatedChildCount().Get(); ok {
		index = min(index, count-1)
	}
	index = max(index, 0)
	if index == r.IndexOf(first) {
		return true
	}
	r.CollectGarbage(r.ChildCount(), 0)
	r.measuredFirst, r.measuredLast = -1, -1
	if r.AddInitialChild(AddInitialChildOpts{Index: index, LayoutOffset: float64(index) * average}) {
		return true
	}
	// An unbounded list that ends before the estimate: start again from its first
	// child, as the list does without the option.
	r.childManager.SetDidUnderflow(false)
	return r.AddInitialChild()
}
