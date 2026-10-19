package rendering

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

func assertSliverOutOfExtent(extent float64) {
	if foundation.DebugMode && extent <= 0 {
		panic(errors.New("SliverCrossAxisGroupOutOfExtent", "SliverCrossAxisGroup ran out of extent before child could be laid out. "+
			"SliverCrossAxisGroup lays out any slivers with a constrained cross axis before laying out those which expand. In this case, cross axis "+
			"extent was used up before the next sliver could be laid out. Make sure that the total amount of extent allocated by constrained "+
			"child slivers does not exceed the cross axis extent that is available for the SliverCrossAxisGroup.", "extent", extent))
	}
}

// RenderSliverCrossAxisGroup places its sliver children next to each other in the
// cross axis.
//
// Flutter: rendering/sliver_group.dart RenderSliverCrossAxisGroup
type RenderSliverCrossAxisGroup struct {
	RenderSliver
	ContainerRenderObject[Sliver]
}

// NewRenderSliverCrossAxisGroup creates the sliver with the children.
//
// Flutter: rendering/sliver_group.dart RenderSliverCrossAxisGroup.new
func NewRenderSliverCrossAxisGroup(children ...Sliver) *RenderSliverCrossAxisGroup {
	r := &RenderSliverCrossAxisGroup{}
	InitRenderSliver(&r.RenderSliver, r)
	InitContainerRenderObject(&r.ContainerRenderObject, r)
	r.AddAll(children)
	return r
}

// SetupParentData gives the child a SliverPhysicalContainerParentData with a
// cross axis flex of 1 if it has none.
//
// Flutter: rendering/sliver_group.dart RenderSliverCrossAxisGroup.setupParentData
func (r *RenderSliverCrossAxisGroup) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*SliverPhysicalContainerParentData); !ok {
		data := &SliverPhysicalContainerParentData{}
		data.CrossAxisFlex = foundation.Some(1)
		child.AsRenderObject().SetParentData(data)
	}
}

// ChildMainAxisPosition is zero.
//
// Flutter: rendering/sliver_group.dart RenderSliverCrossAxisGroup.childMainAxisPosition
func (r *RenderSliverCrossAxisGroup) ChildMainAxisPosition(RenderObject) float64 { return 0 }

// ChildCrossAxisPosition is the offset of the child in the cross axis.
//
// Flutter: rendering/sliver_group.dart RenderSliverCrossAxisGroup.childCrossAxisPosition
func (r *RenderSliverCrossAxisGroup) ChildCrossAxisPosition(child RenderObject) float64 {
	paintOffset := SliverPhysicalParentDataOf(child).PaintOffset
	if r.SliverConstraints().Axis() == painting.AxisVertical {
		return paintOffset.Dx
	}
	return paintOffset.Dy
}

// PerformLayout gives the children without a flex their cross axis extent first,
// and shares what remains between the others by their flex factors. In debug
// builds an infinite cross axis extent panics with the tag
// SliverCrossAxisGroupExtentInfinite, and a child that has no extent left panics
// with SliverCrossAxisGroupOutOfExtent.
//
// Flutter: rendering/sliver_group.dart RenderSliverCrossAxisGroup.performLayout
func (r *RenderSliverCrossAxisGroup) PerformLayout() {
	c := r.SliverConstraints()
	crossAxisExtent := c.CrossAxisExtent
	if foundation.DebugMode && (math.IsInf(crossAxisExtent, 0) || math.IsNaN(crossAxisExtent)) {
		panic(errors.New("SliverCrossAxisGroupExtentInfinite", "a SliverCrossAxisGroup needs a cross axis extent that is finite",
			"object", foundation.DescribeIdentity(r.self), "crossAxisExtent", crossAxisExtent))
	}

	totalFlex := 0
	remainingExtent := crossAxisExtent
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		flex := SliverPhysicalParentDataOf(child).CrossAxisFlex.OrElse(0)
		if flex == 0 {
			assertSliverOutOfExtent(remainingExtent)
			child.Layout(c.CopyWith(SliverConstraintsCopy{CrossAxisExtent: foundation.Some(remainingExtent)}), LayoutOpts{ParentUsesSize: true})
			childCrossAxisExtent := child.Geometry().CrossAxisExtent
			if foundation.DebugMode && !childCrossAxisExtent.IsSome() {
				panic(errors.New("SliverCrossAxisGroupChildExtentMissing", "a child of a SliverCrossAxisGroup without a flex must set the cross axis extent of its geometry",
					"object", foundation.DescribeIdentity(r.self), "child", foundation.DescribeIdentity(child)))
			}
			remainingExtent = math.Max(0, remainingExtent-childCrossAxisExtent.Value())
		} else {
			totalFlex += flex
		}
	}
	extentPerFlexValue := remainingExtent / float64(totalFlex)

	r.SetGeometry(SliverGeometryZero)
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		flex := SliverPhysicalParentDataOf(child).CrossAxisFlex.OrElse(0)
		if flex != 0 {
			assertSliverOutOfExtent(extentPerFlexValue * float64(flex))
			child.Layout(c.CopyWith(SliverConstraintsCopy{CrossAxisExtent: foundation.Some(extentPerFlexValue * float64(flex))}), LayoutOpts{ParentUsesSize: true})
		}
		childGeometry := child.Geometry()
		if r.Geometry().ScrollExtent < childGeometry.ScrollExtent {
			r.SetGeometry(childGeometry)
		}
	}

	offset := 0.0
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		data := SliverPhysicalParentDataOf(child)
		childGeometry := child.Geometry()
		remaining := r.Geometry().ScrollExtent - c.ScrollOffset
		paintCorrection := 0.0
		if childGeometry.PaintExtent > remaining {
			paintCorrection = childGeometry.PaintExtent - remaining
		}
		childExtent := childGeometry.CrossAxisExtent.OrElse(extentPerFlexValue * float64(data.CrossAxisFlex.OrElse(0)))
		if c.Axis() == painting.AxisVertical {
			data.PaintOffset = ui.Offset{Dx: offset, Dy: -paintCorrection}
		} else {
			data.PaintOffset = ui.Offset{Dx: -paintCorrection, Dy: offset}
		}
		offset += childExtent
	}
}

// Paint paints the children that are visible.
//
// Flutter: rendering/sliver_group.dart RenderSliverCrossAxisGroup.paint
func (r *RenderSliverCrossAxisGroup) Paint(context *PaintingContext, offset ui.Offset) {
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		if child.Geometry().Visible {
			context.PaintChild(child, offset.Add(SliverPhysicalParentDataOf(child).PaintOffset))
		}
	}
}

// ApplyPaintTransform translates by the paint offset of the child.
//
// Flutter: rendering/sliver_group.dart RenderSliverCrossAxisGroup.applyPaintTransform
func (r *RenderSliverCrossAxisGroup) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	SliverPhysicalParentDataOf(child).ApplyPaintTransform(transform)
}

// HitTestChildren hit tests the children, from the last.
//
// Flutter: rendering/sliver_group.dart RenderSliverCrossAxisGroup.hitTestChildren
func (r *RenderSliverCrossAxisGroup) HitTestChildren(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	self := r.debugSelf()
	for child := r.LastChild(); !isNil(child); child = r.ChildBefore(child) {
		paintOffset := SliverPhysicalParentDataOf(child).PaintOffset
		if result.AddWithAxisOffset(&paintOffset, self.ChildMainAxisPosition(child), self.ChildCrossAxisPosition(child),
			mainAxisPosition, crossAxisPosition, child.HitTest) {
			return true
		}
	}
	return false
}

// RenderSliverMainAxisGroup places its sliver children one after the other in the
// main axis, like slivers in a viewport.
//
// Flutter: rendering/sliver_group.dart RenderSliverMainAxisGroup
type RenderSliverMainAxisGroup struct {
	RenderSliver
	ContainerRenderObject[Sliver]
}

// NewRenderSliverMainAxisGroup creates the sliver with the children.
//
// Flutter: rendering/sliver_group.dart RenderSliverMainAxisGroup.new
func NewRenderSliverMainAxisGroup(children ...Sliver) *RenderSliverMainAxisGroup {
	r := &RenderSliverMainAxisGroup{}
	InitRenderSliver(&r.RenderSliver, r)
	InitContainerRenderObject(&r.ContainerRenderObject, r)
	r.AddAll(children)
	return r
}

// SetupParentData gives the child a SliverPhysicalContainerParentData if it has
// none.
//
// Flutter: rendering/sliver_group.dart RenderSliverMainAxisGroup.setupParentData
func (r *RenderSliverMainAxisGroup) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*SliverPhysicalContainerParentData); !ok {
		child.AsRenderObject().SetParentData(&SliverPhysicalContainerParentData{})
	}
}

// ChildScrollOffset is the scroll extent of the children before the child, minus
// what the pinned ones obstruct.
//
// Flutter: rendering/sliver_group.dart RenderSliverMainAxisGroup.childScrollOffset
func (r *RenderSliverMainAxisGroup) ChildScrollOffset(child RenderObject) foundation.Opt[float64] {
	sliver := child.(Sliver)
	extentOfPinnedSlivers := r.maxScrollObstructionExtentBefore(sliver)
	childScrollOffset := 0.0
	if r.SliverConstraints().GrowthDirection == GrowthDirectionForward {
		for current := r.ChildBefore(sliver); !isNil(current); current = r.ChildBefore(current) {
			childScrollOffset += current.Geometry().ScrollExtent
		}
	} else {
		for current := r.ChildAfter(sliver); !isNil(current); current = r.ChildAfter(current) {
			childScrollOffset -= current.Geometry().ScrollExtent
		}
	}
	return foundation.Some(childScrollOffset - extentOfPinnedSlivers)
}

func (r *RenderSliverMainAxisGroup) maxScrollObstructionExtentBefore(child Sliver) float64 {
	pinnedExtent := 0.0
	if child.SliverConstraints().GrowthDirection == GrowthDirectionForward {
		for current := r.FirstChild(); current != child; current = r.ChildAfter(current) {
			pinnedExtent += current.Geometry().MaxScrollObstructionExtent
		}
		return pinnedExtent
	}
	for current := r.LastChild(); current != child; current = r.ChildBefore(current) {
		pinnedExtent += current.Geometry().MaxScrollObstructionExtent
	}
	return pinnedExtent
}

// ChildMainAxisPosition is the position of the child in the main axis.
//
// Flutter: rendering/sliver_group.dart RenderSliverMainAxisGroup.childMainAxisPosition
func (r *RenderSliverMainAxisGroup) ChildMainAxisPosition(child RenderObject) float64 {
	sliver := child.(Sliver)
	data := SliverPhysicalParentDataOf(child)
	c := sliver.SliverConstraints()
	switch ApplyGrowthDirectionToAxisDirection(c.AxisDirection, c.GrowthDirection) {
	case painting.AxisDirectionRight:
		return data.PaintOffset.Dx
	case painting.AxisDirectionUp:
		return r.Geometry().PaintExtent - sliver.Geometry().PaintExtent - data.PaintOffset.Dy
	case painting.AxisDirectionLeft:
		return r.Geometry().PaintExtent - sliver.Geometry().PaintExtent - data.PaintOffset.Dx
	}
	return data.PaintOffset.Dy
}

// ChildCrossAxisPosition is zero.
//
// Flutter: rendering/sliver_group.dart RenderSliverMainAxisGroup.childCrossAxisPosition
func (r *RenderSliverMainAxisGroup) ChildCrossAxisPosition(RenderObject) float64 { return 0 }

func fixPrecisionError(number float64) float64 {
	if math.Abs(number) < foundation.PrecisionErrorTolerance {
		return 0
	}
	return number
}

// PerformLayout lays the children out one after the other. In debug builds a
// sliver after one of infinite extent panics with the tag
// SliverUnreachableSliver.
//
// Flutter: rendering/sliver_group.dart RenderSliverMainAxisGroup.performLayout
func (r *RenderSliverMainAxisGroup) PerformLayout() {
	c := r.SliverConstraints()
	scrollOffset, layoutOffset, maxPaintExtent := 0.0, 0.0, 0.0
	paintOffset := c.Overlap
	maxScrollObstructionExtent := 0.0

	cacheOrigin := c.CacheOrigin
	remainingCacheExtent := c.RemainingCacheExtent

	var leadingChild Sliver
	var advance func(Sliver) Sliver
	if c.GrowthDirection == GrowthDirectionForward {
		leadingChild, advance = r.FirstChild(), r.ChildAfter
	} else {
		leadingChild, advance = r.LastChild(), r.ChildBefore
	}
	child := leadingChild
	for !isNil(child) {
		beforeOffsetPaintExtent := r.CalculatePaintOffset(c, 0, scrollOffset)

		childScrollOffset := math.Max(0, c.ScrollOffset-scrollOffset)
		correctedCacheOrigin := math.Max(cacheOrigin, -childScrollOffset)
		cacheExtentCorrection := cacheOrigin - correctedCacheOrigin

		child.Layout(c.CopyWith(SliverConstraintsCopy{
			ScrollOffset:          foundation.Some(childScrollOffset),
			CacheOrigin:           foundation.Some(correctedCacheOrigin),
			Overlap:               foundation.Some(math.Max(0, fixPrecisionError(paintOffset-beforeOffsetPaintExtent))),
			RemainingPaintExtent:  foundation.Some(fixPrecisionError(c.RemainingPaintExtent - beforeOffsetPaintExtent)),
			RemainingCacheExtent:  foundation.Some(math.Max(0, fixPrecisionError(remainingCacheExtent+cacheExtentCorrection))),
			PrecedingScrollExtent: foundation.Some(scrollOffset + c.PrecedingScrollExtent),
		}), LayoutOpts{ParentUsesSize: true})

		childGeometry := child.Geometry()

		if correction, ok := childGeometry.ScrollOffsetCorrection.Get(); ok {
			r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{ScrollOffsetCorrection: foundation.Some(correction)}))
			return
		}

		if foundation.DebugMode {
			childGeometry.DebugAssertIsValid()
		}

		childPaintOffset := layoutOffset + childGeometry.PaintOrigin
		data := SliverPhysicalParentDataOf(child)
		if c.Axis() == painting.AxisVertical {
			data.PaintOffset = ui.Offset{Dy: childPaintOffset}
		} else {
			data.PaintOffset = ui.Offset{Dx: childPaintOffset}
		}
		scrollOffset += childGeometry.ScrollExtent
		layoutOffset += childGeometry.LayoutExtent
		maxPaintExtent += childGeometry.MaxPaintExtent
		maxScrollObstructionExtent += childGeometry.MaxScrollObstructionExtent
		paintOffset = math.Max(childPaintOffset+childGeometry.PaintExtent, paintOffset)
		if childGeometry.CacheExtent != 0 {
			remainingCacheExtent = fixPrecisionError(remainingCacheExtent - childGeometry.CacheExtent - cacheExtentCorrection)
			cacheOrigin = math.Min(correctedCacheOrigin+childGeometry.CacheExtent, 0)
		}
		child = advance(child)
		if foundation.DebugMode && !isNil(child) && math.IsInf(maxPaintExtent, 0) {
			panic(errors.New("SliverUnreachableSliver", "Unreachable sliver found, you may have a sliver following a sliver with an infinite extent.",
				"object", foundation.DescribeIdentity(r.self)))
		}
	}

	remainingExtent := math.Max(0, scrollOffset-c.ScrollOffset)
	if paintOffset > remainingExtent {
		pinnedChildrenOverflow := maxScrollObstructionExtent > remainingExtent-c.Overlap
		paintCorrection := paintOffset - remainingExtent
		paintOffset = remainingExtent
		for child = r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
			childGeometry := child.Geometry()
			data := SliverPhysicalParentDataOf(child)
			childMainAxisPaintOffset := data.PaintOffset.Dx
			if c.Axis() == painting.AxisVertical {
				childMainAxisPaintOffset = data.PaintOffset.Dy
			}
			childPaintEnd := childMainAxisPaintOffset + childGeometry.PaintExtent
			childIsPinned := childGeometry.MaxScrollObstructionExtent > 0
			if childPaintEnd > remainingExtent || (pinnedChildrenOverflow && childIsPinned) {
				if c.Axis() == painting.AxisVertical {
					data.PaintOffset = ui.Offset{Dy: data.PaintOffset.Dy - paintCorrection}
				} else {
					data.PaintOffset = ui.Offset{Dx: data.PaintOffset.Dx - paintCorrection}
				}
			}
		}
	}

	cacheExtent := r.CalculateCacheOffset(c, math.Min(c.ScrollOffset, 0), scrollOffset)
	paintExtent := ui.ClampDouble(paintOffset, 0, c.RemainingPaintExtent)

	r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent:      scrollOffset,
		PaintExtent:       paintExtent,
		CacheExtent:       foundation.Some(cacheExtent),
		MaxPaintExtent:    maxPaintExtent,
		HasVisualOverflow: scrollOffset > c.RemainingPaintExtent || c.ScrollOffset > 0,
	}))

	for child = leadingChild; !isNil(child); child = advance(child) {
		data := SliverPhysicalParentDataOf(child)
		childPaintExtent := child.Geometry().PaintExtent
		switch ApplyGrowthDirectionToAxisDirection(c.AxisDirection, c.GrowthDirection) {
		case painting.AxisDirectionUp:
			data.PaintOffset = ui.Offset{Dy: paintExtent - data.PaintOffset.Dy - childPaintExtent}
		case painting.AxisDirectionLeft:
			data.PaintOffset = ui.Offset{Dx: paintExtent - data.PaintOffset.Dx - childPaintExtent}
		}
	}
}

// Paint paints the children that are visible, from the last.
//
// Flutter: rendering/sliver_group.dart RenderSliverMainAxisGroup.paint
func (r *RenderSliverMainAxisGroup) Paint(context *PaintingContext, offset ui.Offset) {
	for child := r.LastChild(); !isNil(child); child = r.ChildBefore(child) {
		if child.Geometry().Visible {
			context.PaintChild(child, offset.Add(SliverPhysicalParentDataOf(child).PaintOffset))
		}
	}
}

// ApplyPaintTransform translates by the paint offset of the child.
//
// Flutter: rendering/sliver_group.dart RenderSliverMainAxisGroup.applyPaintTransform
func (r *RenderSliverMainAxisGroup) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	SliverPhysicalParentDataOf(child).ApplyPaintTransform(transform)
}

// HitTestChildren hit tests the children, from the first.
//
// Flutter: rendering/sliver_group.dart RenderSliverMainAxisGroup.hitTestChildren
func (r *RenderSliverMainAxisGroup) HitTestChildren(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	self := r.debugSelf()
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		paintOffset := SliverPhysicalParentDataOf(child).PaintOffset
		if result.AddWithAxisOffset(&paintOffset, self.ChildMainAxisPosition(child), self.ChildCrossAxisPosition(child),
			mainAxisPosition, crossAxisPosition, child.HitTest) {
			return true
		}
	}
	return false
}

// VisitChildrenForSemantics visits the children that are visible, in the cache
// area or keep their semantics.
//
// Flutter: rendering/sliver_group.dart RenderSliverMainAxisGroup.visitChildrenForSemantics
func (r *RenderSliverMainAxisGroup) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		g := child.Geometry()
		if g.Visible || g.CacheExtent > 0 || child.EnsureSemantics() {
			visitor(child)
		}
	}
}
