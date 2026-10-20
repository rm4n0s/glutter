// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"math"
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// CacheExtentStyle is how a ScrollCacheExtent measures the cache area.
//
// Flutter: rendering/viewport.dart CacheExtentStyle
type CacheExtentStyle int

// The values of CacheExtentStyle follow Flutter's declaration order.
const (
	// CacheExtentStyleUnspecified is the zero value; it is not a real style.
	//
	// Flutter: no counterpart
	CacheExtentStyleUnspecified CacheExtentStyle = iota
	// CacheExtentStylePixel measures the cache area in logical pixels.
	//
	// Flutter: rendering/viewport.dart CacheExtentStyle.pixel
	CacheExtentStylePixel
	// CacheExtentStyleViewport measures the cache area as a multiple of the main
	// axis extent of the viewport.
	//
	// Flutter: rendering/viewport.dart CacheExtentStyle.viewport
	CacheExtentStyleViewport
)

// ScrollCacheExtent is how far the viewport lays out and keeps children before
// and after the visible area, in pixels or as a multiple of the viewport.
//
// Flutter: rendering/viewport.dart ScrollCacheExtent
type ScrollCacheExtent struct {
	style CacheExtentStyle
	value float64
}

// NewScrollCacheExtentPixels is a cache area of a number of logical pixels.
//
// Flutter: rendering/viewport.dart ScrollCacheExtent.pixels
func NewScrollCacheExtentPixels(pixels float64) ScrollCacheExtent {
	return ScrollCacheExtent{CacheExtentStylePixel, pixels}
}

// NewScrollCacheExtentViewport is a cache area that is a multiple of the main
// axis extent of the viewport.
//
// Flutter: rendering/viewport.dart ScrollCacheExtent.viewport
func NewScrollCacheExtentViewport(value float64) ScrollCacheExtent {
	return ScrollCacheExtent{CacheExtentStyleViewport, value}
}

// Style is how the extent is measured.
//
// Flutter: rendering/viewport.dart ScrollCacheExtent.style
func (e ScrollCacheExtent) Style() CacheExtentStyle { return e.style }

// Value is the number of pixels, or the multiple of the viewport.
//
// Flutter: rendering/viewport.dart ScrollCacheExtent.value
func (e ScrollCacheExtent) Value() float64 { return e.value }

// CalculateCacheOffset is the cache extent in pixels for a viewport with the main
// axis extent.
//
// Flutter: rendering/viewport.dart ScrollCacheExtent._calculateCacheOffset
func (e ScrollCacheExtent) CalculateCacheOffset(mainAxisExtent float64) float64 {
	if e.style == CacheExtentStyleViewport {
		return e.value * mainAxisExtent
	}
	return e.value
}

// String is "ScrollCacheExtent.pixels(250.0)" or "ScrollCacheExtent.viewport(0.5)".
//
// Flutter: rendering/viewport.dart ScrollCacheExtent.toString
func (e ScrollCacheExtent) String() string {
	name := "pixels"
	if e.style == CacheExtentStyleViewport {
		name = "viewport"
	}
	return "ScrollCacheExtent." + name + "(" + dartDouble(e.value) + ")"
}

// SliverPaintOrder is the order in which a viewport paints its slivers.
//
// Flutter: rendering/viewport.dart SliverPaintOrder
type SliverPaintOrder int

// The values of SliverPaintOrder follow Flutter's declaration order.
const (
	// SliverPaintOrderUnspecified is the zero value: the default.
	//
	// Flutter: no counterpart
	SliverPaintOrderUnspecified SliverPaintOrder = iota
	// SliverPaintOrderFirstIsTop paints the first sliver on top of the others.
	//
	// Flutter: rendering/viewport.dart SliverPaintOrder.firstIsTop
	SliverPaintOrderFirstIsTop
	// SliverPaintOrderLastIsTop paints the last sliver on top of the others.
	//
	// Flutter: rendering/viewport.dart SliverPaintOrder.lastIsTop
	SliverPaintOrderLastIsTop
)

// RenderAbstractViewportDefaultCacheExtent is the default cache extent in pixels.
//
// Flutter: rendering/viewport.dart RenderAbstractViewport.defaultCacheExtent
const RenderAbstractViewportDefaultCacheExtent = 250.0

// RevealedOffset is a scroll offset that reveals a target, and where the target
// ends up in the viewport.
//
// Flutter: rendering/viewport.dart RevealedOffset
type RevealedOffset struct {
	// Offset is the scroll offset that puts the target at the alignment.
	//
	// Flutter: rendering/viewport.dart RevealedOffset.offset
	Offset float64
	// Rect is the rectangle of the target in the coordinates of the viewport once
	// it is scrolled to Offset.
	//
	// Flutter: rendering/viewport.dart RevealedOffset.rect
	Rect ui.Rect
}

// RevealedOffsetClampOffset is the offset, of the two that reveal the leading and
// the trailing edge of a target, that brings the current offset nearest to
// showing the target; none if the target is visible already.
//
// Flutter: rendering/viewport.dart RevealedOffset.clampOffset
func RevealedOffsetClampOffset(leadingEdgeOffset, trailingEdgeOffset RevealedOffset, currentOffset float64) foundation.Opt[RevealedOffset] {
	inverted := leadingEdgeOffset.Offset < trailingEdgeOffset.Offset
	smaller, larger := trailingEdgeOffset, leadingEdgeOffset
	if inverted {
		smaller, larger = leadingEdgeOffset, trailingEdgeOffset
	}
	if currentOffset > larger.Offset {
		return foundation.Some(larger)
	} else if currentOffset < smaller.Offset {
		return foundation.Some(smaller)
	}
	return foundation.None[RevealedOffset]()
}

// String describes the offset and the rectangle.
//
// Flutter: rendering/viewport.dart RevealedOffset.toString
func (r RevealedOffset) String() string {
	return "RevealedOffset(offset: " + dartDouble(r.Offset) + ", rect: " + r.Rect.String() + ")"
}

// GetOffsetToRevealOpts are the optional parameters of GetOffsetToReveal.
//
// Flutter: rendering/viewport.dart RenderAbstractViewport.getOffsetToReveal
type GetOffsetToRevealOpts struct {
	// Rect is the part of the target to reveal, in its coordinates; the default
	// is its paint bounds.
	Rect foundation.Opt[ui.Rect]
	// Axis is the axis to reveal along; the default is the main axis of the
	// viewport.
	Axis painting.Axis
}

// RenderAbstractViewport is the interface of a render object that shows part of
// its descendants, and knows the scroll offset that reveals one of them.
//
// Flutter: rendering/viewport.dart RenderAbstractViewport
type RenderAbstractViewport interface {
	RenderObject
	// GetOffsetToReveal returns the scroll offset that puts the target at the
	// alignment in the viewport (0 for the leading edge, 1 for the trailing edge),
	// and where the target is then.
	//
	// Flutter: rendering/viewport.dart RenderAbstractViewport.getOffsetToReveal
	GetOffsetToReveal(target RenderObject, alignment float64, opts ...GetOffsetToRevealOpts) RevealedOffset
}

// RenderAbstractViewportMaybeOf finds the closest viewport, starting at the
// object itself; nil if there is none.
//
// Flutter: rendering/viewport.dart RenderAbstractViewport.maybeOf
func RenderAbstractViewportMaybeOf(object RenderObject) RenderAbstractViewport {
	for !isNil(object) {
		if v, ok := object.(RenderAbstractViewport); ok {
			return v
		}
		object = object.Parent()
	}
	return nil
}

// RenderAbstractViewportOf finds the closest viewport, starting at the object
// itself. In debug builds it panics with the tag ViewportNotFound if there is
// none.
//
// Flutter: rendering/viewport.dart RenderAbstractViewport.of
func RenderAbstractViewportOf(object RenderObject) RenderAbstractViewport {
	v := RenderAbstractViewportMaybeOf(object)
	if v == nil && foundation.DebugMode {
		panic(errors.New("ViewportNotFound", "RenderAbstractViewportOf() was called with a render object that was not a descendant of a "+
			"RenderAbstractViewport. No RenderAbstractViewport render object ancestor could be found starting from the object that was "+
			"passed to RenderAbstractViewportOf(). The render object where the viewport search started was: "+foundation.DescribeIdentity(object)))
	}
	return v
}

// RenderViewportUseTwoPaneSemantics is the tag that the viewport gives its
// children, so that the semantics of the children that are out of view stay in
// a separate pane.
//
// Flutter: rendering/viewport.dart RenderViewport.useTwoPaneSemantics
var RenderViewportUseTwoPaneSemantics = semantics.NewSemanticsTag("RenderViewport.twoPane")

// RenderViewportExcludeFromScrolling is the tag of the children that the
// semantics of the viewport leave out of scrolling.
//
// Flutter: rendering/viewport.dart RenderViewport.excludeFromScrolling
var RenderViewportExcludeFromScrolling = semantics.NewSemanticsTag("RenderViewport.excludeFromScrolling")

// viewportHooks are the methods that the concrete viewports supply to
// RenderViewportBase.
type viewportHooks interface {
	HasVisualOverflow() bool
	UpdateOutOfBandData(growthDirection GrowthDirection, childLayoutGeometry SliverGeometry)
	UpdateChildLayoutOffset(child Sliver, layoutOffset float64, growthDirection GrowthDirection)
	PaintOffsetOf(child Sliver) ui.Offset
	ScrollOffsetOf(child Sliver, scrollOffsetWithinChild float64) float64
	MaxScrollObstructionExtentBefore(child Sliver) float64
	ComputeChildMainAxisPosition(child Sliver, parentMainAxisPosition float64) float64
	IndexOfFirstChild() int
	LabelForChild(index int) string
}

// RenderViewportBase is the base of the viewports: boxes that show part of their
// slivers, which scroll along the main axis by the offset. The concrete type
// embeds it and implements HasVisualOverflow, UpdateOutOfBandData,
// UpdateChildLayoutOffset, PaintOffsetOf, ScrollOffsetOf,
// MaxScrollObstructionExtentBefore, ComputeChildMainAxisPosition,
// IndexOfFirstChild and LabelForChild.
//
// Flutter: rendering/viewport.dart RenderViewportBase
type RenderViewportBase struct {
	RenderBox
	ContainerRenderObject[Sliver]

	axisDirection         painting.AxisDirection
	crossAxisDirection    painting.AxisDirection
	offset                ViewportOffset
	offsetHandle          foundation.ListenerHandle
	scrollCacheExtent     ScrollCacheExtent
	calculatedCacheExtent foundation.Opt[float64]
	paintOrder            SliverPaintOrder
	clipBehavior          ui.Clip
	clipRectLayer         *LayerHandle[*ClipRectLayer]
}

// RenderViewportBaseOpts are the optional parameters of InitRenderViewportBase.
//
// Flutter: rendering/viewport.dart RenderViewportBase.new
type RenderViewportBaseOpts struct {
	// AxisDirection defaults to painting.AxisDirectionDown.
	AxisDirection painting.AxisDirection
	// ScrollCacheExtent defaults to ScrollCacheExtent.pixels(250).
	ScrollCacheExtent foundation.Opt[ScrollCacheExtent]
	// PaintOrder defaults to SliverPaintOrderFirstIsTop.
	PaintOrder SliverPaintOrder
	// ClipBehavior defaults to ui.ClipHardEdge.
	ClipBehavior ui.Clip
}

// InitRenderViewportBase sets the outer object and the properties. In debug
// builds the two directions along the same axis panic with the tag
// ViewportAxesInvalid.
//
// Deviation: the deprecated cacheExtent and cacheExtentStyle parameters are not
// ported; use ScrollCacheExtent (SPEC §12.10).
//
// Flutter: rendering/viewport.dart RenderViewportBase.new
func InitRenderViewportBase(b *RenderViewportBase, self Box, crossAxisDirection painting.AxisDirection, offset ViewportOffset, opts RenderViewportBaseOpts) {
	if opts.AxisDirection == painting.AxisDirectionUnspecified {
		opts.AxisDirection = painting.AxisDirectionDown
	}
	if foundation.DebugMode && painting.AxisDirectionToAxis(opts.AxisDirection) == painting.AxisDirectionToAxis(crossAxisDirection) {
		panic(errors.New("ViewportAxesInvalid", "the axisDirection and the crossAxisDirection of a viewport must be along different axes",
			"axisDirection", opts.AxisDirection.String(), "crossAxisDirection", crossAxisDirection.String()))
	}
	b.axisDirection = opts.AxisDirection
	b.crossAxisDirection = crossAxisDirection
	b.offset = offset
	b.scrollCacheExtent = opts.ScrollCacheExtent.OrElse(NewScrollCacheExtentPixels(RenderAbstractViewportDefaultCacheExtent))
	b.paintOrder = opts.PaintOrder
	if b.paintOrder == SliverPaintOrderUnspecified {
		b.paintOrder = SliverPaintOrderFirstIsTop
	}
	b.clipBehavior = opts.ClipBehavior
	if b.clipBehavior == ui.ClipUnspecified {
		b.clipBehavior = ui.ClipHardEdge
	}
	b.clipRectLayer = NewLayerHandle[*ClipRectLayer](nil)
	InitRenderBox(&b.RenderBox, self)
	InitContainerRenderObject(&b.ContainerRenderObject, self)
}

func (b *RenderViewportBase) hooks() viewportHooks { return b.self.(viewportHooks) }

// DescribeSemanticsConfiguration tags the children with the two pane semantics.
//
// Flutter: rendering/viewport.dart RenderViewportBase.describeSemanticsConfiguration
func (b *RenderViewportBase) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	b.RenderBox.DescribeSemanticsConfiguration(config)
	config.AddTagForChildren(RenderViewportUseTwoPaneSemantics)
}

// VisitChildrenForSemantics visits the slivers that are visible, are in the
// cache area or keep their semantics, in paint order.
//
// Flutter: rendering/viewport.dart RenderViewportBase.visitChildrenForSemantics
func (b *RenderViewportBase) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	for _, sliver := range b.ChildrenInPaintOrder() {
		g := sliver.Geometry()
		if g.Visible || g.CacheExtent > 0 || sliver.EnsureSemantics() {
			visitor(sliver)
		}
	}
}

// AxisDirection is the direction in which the scroll offset increases.
//
// Flutter: rendering/viewport.dart RenderViewportBase.axisDirection
func (b *RenderViewportBase) AxisDirection() painting.AxisDirection { return b.axisDirection }

// SetAxisDirection changes the axis direction and marks the viewport as needing
// layout.
//
// Flutter: rendering/viewport.dart RenderViewportBase.axisDirection
func (b *RenderViewportBase) SetAxisDirection(value painting.AxisDirection) {
	if value == b.axisDirection {
		return
	}
	b.axisDirection = value
	b.MarkNeedsLayout()
}

// CrossAxisDirection is the direction in which the children are laid out in the
// cross axis.
//
// Flutter: rendering/viewport.dart RenderViewportBase.crossAxisDirection
func (b *RenderViewportBase) CrossAxisDirection() painting.AxisDirection { return b.crossAxisDirection }

// SetCrossAxisDirection changes the cross axis direction and marks the viewport as
// needing layout.
//
// Flutter: rendering/viewport.dart RenderViewportBase.crossAxisDirection
func (b *RenderViewportBase) SetCrossAxisDirection(value painting.AxisDirection) {
	if value == b.crossAxisDirection {
		return
	}
	b.crossAxisDirection = value
	b.MarkNeedsLayout()
}

// Axis is the axis of the main direction.
//
// Flutter: rendering/viewport.dart RenderViewportBase.axis
func (b *RenderViewportBase) Axis() painting.Axis {
	return painting.AxisDirectionToAxis(b.axisDirection)
}

// Offset is the offset of the content.
//
// Flutter: rendering/viewport.dart RenderViewportBase.offset
func (b *RenderViewportBase) Offset() ViewportOffset { return b.offset }

// SetOffset changes the offset: the viewport listens to the new one while it is
// attached.
//
// Flutter: rendering/viewport.dart RenderViewportBase.offset
func (b *RenderViewportBase) SetOffset(value ViewportOffset) {
	if value == b.offset {
		return
	}
	if b.Attached() {
		b.offset.RemoveListener(b.offsetHandle)
	}
	b.offset = value
	if b.Attached() {
		b.offsetHandle = b.offset.AddListener(b.MarkNeedsLayout)
	}
	b.MarkNeedsLayout()
}

// ScrollCacheExtent is how far the viewport lays out children beyond the visible
// area.
//
// Flutter: rendering/viewport.dart RenderViewportBase.scrollCacheExtent
func (b *RenderViewportBase) ScrollCacheExtent() ScrollCacheExtent { return b.scrollCacheExtent }

// SetScrollCacheExtent changes the cache extent (none means the default of 250
// pixels) and marks the viewport as needing layout.
//
// Flutter: rendering/viewport.dart RenderViewportBase.scrollCacheExtent
func (b *RenderViewportBase) SetScrollCacheExtent(value foundation.Opt[ScrollCacheExtent]) {
	effective := value.OrElse(NewScrollCacheExtentPixels(RenderAbstractViewportDefaultCacheExtent))
	if effective == b.scrollCacheExtent {
		return
	}
	b.scrollCacheExtent = effective
	b.MarkNeedsLayout()
}

// PaintOrder is the order in which the slivers are painted.
//
// Flutter: rendering/viewport.dart RenderViewportBase.paintOrder
func (b *RenderViewportBase) PaintOrder() SliverPaintOrder { return b.paintOrder }

// SetPaintOrder changes the paint order and marks the viewport as needing paint
// and a semantics update.
//
// Flutter: rendering/viewport.dart RenderViewportBase.paintOrder
func (b *RenderViewportBase) SetPaintOrder(value SliverPaintOrder) {
	if value != b.paintOrder {
		b.paintOrder = value
		b.MarkNeedsPaint()
		b.MarkNeedsSemanticsUpdate()
	}
}

// ClipBehavior is how the content is clipped when it overflows.
//
// Flutter: rendering/viewport.dart RenderViewportBase.clipBehavior
func (b *RenderViewportBase) ClipBehavior() ui.Clip { return b.clipBehavior }

// SetClipBehavior changes the clip behavior and marks the viewport as needing paint
// and a semantics update.
//
// Flutter: rendering/viewport.dart RenderViewportBase.clipBehavior
func (b *RenderViewportBase) SetClipBehavior(value ui.Clip) {
	if value != b.clipBehavior {
		b.clipBehavior = value
		b.MarkNeedsPaint()
		b.MarkNeedsSemanticsUpdate()
	}
}

// Attach starts listening to the offset.
//
// Flutter: rendering/viewport.dart RenderViewportBase.attach
func (b *RenderViewportBase) Attach(owner *PipelineOwner) {
	b.RenderBox.Attach(owner)
	b.offsetHandle = b.offset.AddListener(b.MarkNeedsLayout)
}

// Detach stops listening to the offset.
//
// Flutter: rendering/viewport.dart RenderViewportBase.detach
func (b *RenderViewportBase) Detach() {
	b.offset.RemoveListener(b.offsetHandle)
	b.RenderBox.Detach()
}

// debugThrowIfNotCheckingIntrinsics panics, in debug builds, with
// ViewportIntrinsicsUnsupported unless intrinsic dimensions are being checked.
func (b *RenderViewportBase) debugThrowIfNotCheckingIntrinsics() {
	if foundation.DebugMode && !b.dbg().checkingIntrinsics {
		panic(errors.New("ViewportIntrinsicsUnsupported", foundation.RuntimeTypeName(b.self)+" does not support returning intrinsic dimensions. "+
			"Calculating the intrinsic dimensions would require instantiating every child of the viewport, which defeats the point of "+
			"viewports being lazy. If you are merely trying to shrink-wrap the viewport in the main axis direction, consider a "+
			"RenderShrinkWrappingViewport render object (ShrinkWrappingViewport widget), which achieves that effect without "+
			"implementing the intrinsic dimension API.", "object", foundation.DescribeIdentity(b.self)))
	}
}

// ComputeMinIntrinsicWidth is zero; it panics in debug builds.
//
// Flutter: rendering/viewport.dart RenderViewportBase.computeMinIntrinsicWidth
func (b *RenderViewportBase) ComputeMinIntrinsicWidth(float64) float64 {
	b.hooks().(interface{ debugThrowIfNotCheckingIntrinsics() }).debugThrowIfNotCheckingIntrinsics()
	return 0
}

// ComputeMaxIntrinsicWidth is zero; it panics in debug builds.
//
// Flutter: rendering/viewport.dart RenderViewportBase.computeMaxIntrinsicWidth
func (b *RenderViewportBase) ComputeMaxIntrinsicWidth(float64) float64 {
	b.hooks().(interface{ debugThrowIfNotCheckingIntrinsics() }).debugThrowIfNotCheckingIntrinsics()
	return 0
}

// ComputeMinIntrinsicHeight is zero; it panics in debug builds.
//
// Flutter: rendering/viewport.dart RenderViewportBase.computeMinIntrinsicHeight
func (b *RenderViewportBase) ComputeMinIntrinsicHeight(float64) float64 {
	b.hooks().(interface{ debugThrowIfNotCheckingIntrinsics() }).debugThrowIfNotCheckingIntrinsics()
	return 0
}

// ComputeMaxIntrinsicHeight is zero; it panics in debug builds.
//
// Flutter: rendering/viewport.dart RenderViewportBase.computeMaxIntrinsicHeight
func (b *RenderViewportBase) ComputeMaxIntrinsicHeight(float64) float64 {
	b.hooks().(interface{ debugThrowIfNotCheckingIntrinsics() }).debugThrowIfNotCheckingIntrinsics()
	return 0
}

// IsRepaintBoundary is true.
//
// Flutter: rendering/viewport.dart RenderViewportBase.isRepaintBoundary
func (b *RenderViewportBase) IsRepaintBoundary() bool { return true }

// LayoutChildSequenceOpts are the parameters of LayoutChildSequence.
//
// Flutter: rendering/viewport.dart RenderViewportBase.layoutChildSequence
type LayoutChildSequenceOpts struct {
	// Child is the first sliver of the sequence.
	Child Sliver
	// ScrollOffset is the scroll offset of the first sliver.
	ScrollOffset float64
	// Overlap is how much the first sliver overlaps the leading edge.
	Overlap float64
	// LayoutOffset is where the first sliver is laid out, from the leading edge.
	LayoutOffset float64
	// RemainingPaintExtent is how much of the viewport remains to be painted.
	RemainingPaintExtent float64
	// MainAxisExtent is the extent of the viewport in the main axis.
	MainAxisExtent float64
	// CrossAxisExtent is the extent of the viewport in the cross axis.
	CrossAxisExtent float64
	// GrowthDirection is the growth direction of the sequence.
	GrowthDirection GrowthDirection
	// Advance gives the next sliver of the sequence.
	Advance func(child Sliver) Sliver
	// RemainingCacheExtent is how much of the cache area remains.
	RemainingCacheExtent float64
	// CacheOrigin is where the cache area starts.
	CacheOrigin float64
}

// LayoutChildSequence lays out a sequence of slivers, each with the constraints
// that follow from the geometry of the one before. It returns zero, or the
// scroll offset correction that the first sliver that asked for one wants.
//
// Flutter: rendering/viewport.dart RenderViewportBase.layoutChildSequence
func (b *RenderViewportBase) LayoutChildSequence(o LayoutChildSequenceOpts) float64 {
	if foundation.DebugMode && (math.IsInf(o.ScrollOffset, 0) || math.IsNaN(o.ScrollOffset) || o.ScrollOffset < 0) {
		panic(errors.New("ViewportScrollOffsetInvalid", "the scroll offset of a viewport must be finite and not negative",
			"scrollOffset", o.ScrollOffset, "object", foundation.DescribeIdentity(b.self)))
	}
	child, scrollOffset, layoutOffset := o.Child, o.ScrollOffset, o.LayoutOffset
	remainingCacheExtent, cacheOrigin := o.RemainingCacheExtent, o.CacheOrigin
	hooks := b.hooks()
	initialLayoutOffset := layoutOffset
	adjustedUserScrollDirection := ApplyGrowthDirectionToScrollDirection(b.offset.UserScrollDirection(), o.GrowthDirection)
	maxPaintOffset := layoutOffset + o.Overlap
	precedingScrollExtent := 0.0

	for !isNil(child) {
		sliverScrollOffset := 0.0
		if scrollOffset > 0 {
			sliverScrollOffset = scrollOffset
		}
		// If the scrollOffset is too small we adjust the paddedOrigin because it
		// doesn't make sense to ask a sliver for content before its scroll offset.
		correctedCacheOrigin := math.Max(cacheOrigin, -sliverScrollOffset)
		cacheExtentCorrection := cacheOrigin - correctedCacheOrigin

		child.Layout(SliverConstraints{
			AxisDirection:          b.axisDirection,
			GrowthDirection:        o.GrowthDirection,
			UserScrollDirection:    adjustedUserScrollDirection,
			ScrollOffset:           sliverScrollOffset,
			PrecedingScrollExtent:  precedingScrollExtent,
			Overlap:                maxPaintOffset - layoutOffset,
			RemainingPaintExtent:   math.Max(0, o.RemainingPaintExtent-layoutOffset+initialLayoutOffset),
			CrossAxisExtent:        o.CrossAxisExtent,
			CrossAxisDirection:     b.crossAxisDirection,
			ViewportMainAxisExtent: o.MainAxisExtent,
			RemainingCacheExtent:   math.Max(0, remainingCacheExtent+cacheExtentCorrection),
			CacheOrigin:            correctedCacheOrigin,
		}, LayoutOpts{ParentUsesSize: true})

		childLayoutGeometry := child.Geometry()
		if foundation.DebugMode {
			childLayoutGeometry.DebugAssertIsValid()
		}

		// If there is a correction to apply, we'll have to start over.
		if c, ok := childLayoutGeometry.ScrollOffsetCorrection.Get(); ok {
			return c
		}

		// We use the child's paint origin in our coordinate system as the layout
		// offset we store in the child's parent data.
		effectiveLayoutOffset := layoutOffset + childLayoutGeometry.PaintOrigin

		// If the child's visible interval (from the paint origin to the paint
		// extent) is not visible, the child is laid out at the scroll position.
		if childLayoutGeometry.Visible || scrollOffset > 0 {
			hooks.UpdateChildLayoutOffset(child, effectiveLayoutOffset, o.GrowthDirection)
		} else {
			hooks.UpdateChildLayoutOffset(child, -scrollOffset+initialLayoutOffset, o.GrowthDirection)
		}

		maxPaintOffset = math.Max(effectiveLayoutOffset+childLayoutGeometry.PaintExtent, maxPaintOffset)
		scrollOffset -= childLayoutGeometry.ScrollExtent
		precedingScrollExtent += childLayoutGeometry.ScrollExtent
		layoutOffset += childLayoutGeometry.LayoutExtent
		if childLayoutGeometry.CacheExtent != 0 {
			remainingCacheExtent -= childLayoutGeometry.CacheExtent - cacheExtentCorrection
			cacheOrigin = math.Min(correctedCacheOrigin+childLayoutGeometry.CacheExtent, 0)
		}

		hooks.UpdateOutOfBandData(o.GrowthDirection, childLayoutGeometry)

		// Move on to the next child.
		child = o.Advance(child)
	}

	// We made it without a correction, whee!
	return 0
}

// DescribeApproximatePaintClip is the viewport, minus the part that a previous
// pinned or floating sliver covers.
//
// Flutter: rendering/viewport.dart RenderViewportBase.describeApproximatePaintClip
func (b *RenderViewportBase) DescribeApproximatePaintClip(child RenderObject) foundation.Opt[ui.Rect] {
	sliver := child.(Sliver)
	g := sliver.Geometry()
	if sliver.EnsureSemantics() && !(g.Visible || g.CacheExtent > 0) {
		// Not in the cache extent: the clip would be wrong for the semantics.
		return foundation.None[ui.Rect]()
	}
	if b.clipBehavior == ui.ClipNone {
		return foundation.None[ui.Rect]()
	}
	viewportClip := ui.OffsetZero.And(b.Size())
	c := sliver.SliverConstraints()
	if c.Overlap == 0 || math.IsInf(c.ViewportMainAxisExtent, 0) {
		return foundation.Some(viewportClip)
	}
	left, right, top, bottom := viewportClip.Left, viewportClip.Right, viewportClip.Top, viewportClip.Bottom
	startOfOverlap := c.ViewportMainAxisExtent - c.RemainingPaintExtent
	overlapCorrection := startOfOverlap + c.Overlap
	switch ApplyGrowthDirectionToAxisDirection(b.axisDirection, c.GrowthDirection) {
	case painting.AxisDirectionDown:
		top += overlapCorrection
	case painting.AxisDirectionUp:
		bottom -= overlapCorrection
	case painting.AxisDirectionRight:
		left += overlapCorrection
	case painting.AxisDirectionLeft:
		right -= overlapCorrection
	}
	return foundation.Some(ui.NewRectFromLTRB(left, top, right, bottom))
}

// DescribeSemanticsClip is the viewport enlarged by the cache extent.
//
// Flutter: rendering/viewport.dart RenderViewportBase.describeSemanticsClip
func (b *RenderViewportBase) DescribeSemanticsClip(child RenderObject) foundation.Opt[ui.Rect] {
	if sliver, ok := child.(Sliver); ok && !isNil(sliver) {
		g := sliver.Geometry()
		if sliver.EnsureSemantics() && !(g.Visible || g.CacheExtent > 0) {
			return foundation.None[ui.Rect]()
		}
	}
	bounds := b.self.SemanticBounds()
	cache, ok := b.calculatedCacheExtent.Get()
	if !ok {
		return foundation.Some(bounds)
	}
	if b.Axis() == painting.AxisVertical {
		return foundation.Some(ui.NewRectFromLTRB(bounds.Left, bounds.Top-cache, bounds.Right, bounds.Bottom+cache))
	}
	return foundation.Some(ui.NewRectFromLTRB(bounds.Left-cache, bounds.Top, bounds.Right+cache, bounds.Bottom))
}

// Paint paints the slivers, clipped to the viewport if they overflow.
//
// Flutter: rendering/viewport.dart RenderViewportBase.paint
func (b *RenderViewportBase) Paint(context *PaintingContext, offset ui.Offset) {
	if isNil(b.FirstChild()) {
		return
	}
	if b.hooks().HasVisualOverflow() && b.clipBehavior != ui.ClipNone {
		b.clipRectLayer.SetLayer(context.PushClipRect(b.NeedsCompositing(), offset, ui.OffsetZero.And(b.Size()), b.paintContents,
			PushClipRectOpts{ClipBehavior: b.clipBehavior, OldLayer: b.clipRectLayer.Layer()}))
	} else {
		b.clipRectLayer.SetLayer(nil)
		b.paintContents(context, offset)
	}
}

// Dispose lets the clip layer go.
//
// Flutter: rendering/viewport.dart RenderViewportBase.dispose
func (b *RenderViewportBase) Dispose() {
	b.clipRectLayer.SetLayer(nil)
	b.RenderBox.Dispose()
}

func (b *RenderViewportBase) paintContents(context *PaintingContext, offset ui.Offset) {
	hooks := b.hooks()
	for _, child := range b.ChildrenInPaintOrder() {
		if child.Geometry().Visible {
			context.PaintChild(child, offset.Add(hooks.PaintOffsetOf(child)))
		}
	}
}

// DebugPaintSize paints the layout extent of every sliver.
//
// Flutter: rendering/viewport.dart RenderViewportBase.debugPaintSize
func (b *RenderViewportBase) DebugPaintSize(context *PaintingContext, offset ui.Offset) {
	if !foundation.DebugMode {
		return
	}
	b.RenderBox.DebugPaintSize(context, offset)
	paint := ui.NewPaint()
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(1)
	paint.SetColor(ui.NewColor(0xFF00FF00))
	canvas := context.Canvas()
	hooks := b.hooks()
	for child := b.FirstChild(); !isNil(child); child = b.ChildAfter(child) {
		var size ui.Size
		if b.Axis() == painting.AxisVertical {
			size = ui.Size{Width: child.SliverConstraints().CrossAxisExtent, Height: child.Geometry().LayoutExtent}
		} else {
			size = ui.Size{Width: child.Geometry().LayoutExtent, Height: child.SliverConstraints().CrossAxisExtent}
		}
		canvas.DrawRect(offset.Add(hooks.PaintOffsetOf(child)).And(size).Deflate(0.5), paint)
	}
}

// HitTestChildren hit tests the slivers in hit test order.
//
// Flutter: rendering/viewport.dart RenderViewportBase.hitTestChildren
func (b *RenderViewportBase) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	var mainAxisPosition, crossAxisPosition float64
	if b.Axis() == painting.AxisVertical {
		mainAxisPosition, crossAxisPosition = position.Dy, position.Dx
	} else {
		mainAxisPosition, crossAxisPosition = position.Dx, position.Dy
	}
	sliverResult := NewSliverHitTestResultWrap(result.HitTestResult)
	hooks := b.hooks()
	for _, child := range b.ChildrenInHitTestOrder() {
		if !child.Geometry().Visible {
			continue
		}
		transform := ui.NewMatrix4Identity()
		b.self.ApplyPaintTransform(child, transform) // must be invertible
		isHit := result.AddWithOutOfBandPosition(OutOfBandOpts{PaintTransform: transform}, func(*BoxHitTestResult) bool {
			return child.HitTest(sliverResult, hooks.ComputeChildMainAxisPosition(child, mainAxisPosition), crossAxisPosition)
		})
		if isHit {
			return true
		}
	}
	return false
}

// GetOffsetToReveal returns the scroll offset that puts the target at the
// alignment in the viewport, and where the target is then.
//
// Flutter: rendering/viewport.dart RenderViewportBase.getOffsetToReveal
func (b *RenderViewportBase) GetOffsetToReveal(target RenderObject, alignment float64, opts ...GetOffsetToRevealOpts) RevealedOffset {
	o := firstOpts(opts)
	axis := b.Axis() // the axis of the options is ignored, as in Flutter
	leadingScrollOffset := 0.0
	// Starting at `target` and walking towards the root, `child` is the last
	// object before the viewport.
	child := target
	var pivot Box
	// `onlySlivers` is whether all the objects between the viewport and the target
	// (the target included) are slivers.
	_, onlySlivers := target.(Sliver)
	for child.Parent() != RenderObject(b.self) {
		parent := child.Parent()
		if box, ok := child.(Box); ok {
			pivot = box
		}
		if sliver, ok := parent.(Sliver); ok {
			leadingScrollOffset += sliver.ChildScrollOffset(child).Value()
		} else {
			onlySlivers = false
			leadingScrollOffset = 0
		}
		child = parent
	}

	rect := o.Rect
	var rectLocal ui.Rect
	var pivotExtent float64
	var growthDirection GrowthDirection
	switch {
	case !isNil(pivot):
		pivotParent := pivot.Parent().(Sliver)
		growthDirection = pivotParent.SliverConstraints().GrowthDirection
		if axis == painting.AxisHorizontal {
			pivotExtent = pivot.Size().Width
		} else {
			pivotExtent = pivot.Size().Height
		}
		if !rect.IsSome() {
			rect = foundation.Some(target.PaintBounds())
		}
		rectLocal = target.GetTransformTo(pivot).TransformRect(rect.Value())
	case onlySlivers:
		// The target is a sliver, or only slivers are between the target and the
		// viewport.
		targetSliver := target.(Sliver)
		growthDirection = targetSliver.SliverConstraints().GrowthDirection
		pivotExtent = targetSliver.Geometry().ScrollExtent
		if !rect.IsSome() {
			if axis == painting.AxisHorizontal {
				rect = foundation.Some(ui.NewRectFromLTWH(0, 0, targetSliver.Geometry().ScrollExtent, targetSliver.SliverConstraints().CrossAxisExtent))
			} else {
				rect = foundation.Some(ui.NewRectFromLTWH(0, 0, targetSliver.SliverConstraints().CrossAxisExtent, targetSliver.Geometry().ScrollExtent))
			}
		}
		rectLocal = rect.Value()
	default:
		return RevealedOffset{Offset: b.offset.Pixels(), Rect: rect.Value()}
	}

	sliver := child.(Sliver)
	switch ApplyGrowthDirectionToAxisDirection(b.axisDirection, growthDirection) {
	case painting.AxisDirectionUp:
		leadingScrollOffset += pivotExtent - rectLocal.Bottom
	case painting.AxisDirectionLeft:
		leadingScrollOffset += pivotExtent - rectLocal.Right
	case painting.AxisDirectionRight:
		leadingScrollOffset += rectLocal.Left
	default:
		leadingScrollOffset += rectLocal.Top
	}

	// So far leadingScrollOffset is the scroll offset of the leading edge of the
	// target inside the sliver; make it relative to the zero scroll offset of the
	// viewport.
	isPinned := sliver.Geometry().MaxScrollObstructionExtent > 0 && leadingScrollOffset >= 0
	hooks := b.hooks()
	leadingScrollOffset = hooks.ScrollOffsetOf(sliver, leadingScrollOffset)

	transform := target.GetTransformTo(b.self)
	targetRect := transform.TransformRect(rect.Value())
	extentOfPinnedSlivers := hooks.MaxScrollObstructionExtentBefore(sliver)

	switch sliver.SliverConstraints().GrowthDirection {
	case GrowthDirectionForward:
		if isPinned && alignment <= 0 {
			return RevealedOffset{Offset: math.Inf(1), Rect: targetRect}
		}
		leadingScrollOffset -= extentOfPinnedSlivers
	case GrowthDirectionReverse:
		if isPinned && alignment >= 1 {
			return RevealedOffset{Offset: math.Inf(-1), Rect: targetRect}
		}
		// If the sliver is reversed, the leading edge of the target is its trailing
		// edge, in the direction of the scroll offset.
		if axis == painting.AxisVertical {
			leadingScrollOffset -= targetRect.Height()
		} else {
			leadingScrollOffset -= targetRect.Width()
		}
	}

	var mainAxisExtentDifference float64
	if axis == painting.AxisHorizontal {
		mainAxisExtentDifference = b.Size().Width - extentOfPinnedSlivers - rectLocal.Width()
	} else {
		mainAxisExtentDifference = b.Size().Height - extentOfPinnedSlivers - rectLocal.Height()
	}
	targetOffset := leadingScrollOffset - mainAxisExtentDifference*alignment
	offsetDifference := b.offset.Pixels() - targetOffset
	switch b.axisDirection {
	case painting.AxisDirectionUp:
		targetRect = targetRect.Translate(0, -offsetDifference)
	case painting.AxisDirectionDown:
		targetRect = targetRect.Translate(0, offsetDifference)
	case painting.AxisDirectionLeft:
		targetRect = targetRect.Translate(-offsetDifference, 0)
	case painting.AxisDirectionRight:
		targetRect = targetRect.Translate(offsetDifference, 0)
	}
	return RevealedOffset{Offset: targetOffset, Rect: targetRect}
}

// ComputeAbsolutePaintOffset is the paint offset of a sliver in the coordinates of
// the viewport, from its layout offset.
//
// Flutter: rendering/viewport.dart RenderViewportBase.computeAbsolutePaintOffset
func (b *RenderViewportBase) ComputeAbsolutePaintOffset(child Sliver, layoutOffset float64, growthDirection GrowthDirection) ui.Offset {
	size := b.Size()
	switch ApplyGrowthDirectionToAxisDirection(b.axisDirection, growthDirection) {
	case painting.AxisDirectionUp:
		return ui.Offset{Dy: size.Height - layoutOffset - child.Geometry().PaintExtent}
	case painting.AxisDirectionLeft:
		return ui.Offset{Dx: size.Width - layoutOffset - child.Geometry().PaintExtent}
	case painting.AxisDirectionRight:
		return ui.Offset{Dx: layoutOffset}
	}
	return ui.Offset{Dy: layoutOffset}
}

// DebugFillProperties adds the directions and the offset.
//
// Flutter: rendering/viewport.dart RenderViewportBase.debugFillProperties
func (b *RenderViewportBase) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	b.RenderBox.DebugFillProperties(p)
	p.Add(foundation.NewEnumProperty("axisDirection", foundation.Some(b.axisDirection)))
	p.Add(foundation.NewEnumProperty("crossAxisDirection", foundation.Some(b.crossAxisDirection)))
	p.Add(foundation.NewDiagnosticsProperty("offset", b.offset))
}

// DebugDescribeChildren describes the slivers by their label.
//
// Flutter: rendering/viewport.dart RenderViewportBase.debugDescribeChildren
func (b *RenderViewportBase) DebugDescribeChildren() []foundation.DiagnosticsNode {
	var children []foundation.DiagnosticsNode
	if isNil(b.FirstChild()) {
		return children
	}
	hooks := b.hooks()
	count := hooks.IndexOfFirstChild()
	for child := b.FirstChild(); !isNil(child); child = b.ChildAfter(child) {
		children = append(children, foundation.ToDiagnosticsNode(child, foundation.DiagnosticsNodeOpts{Name: foundation.Some(hooks.LabelForChild(count))}))
		count++
	}
	return children
}

// ChildrenInPaintOrder are the slivers in the order they are painted.
//
// Flutter: rendering/viewport.dart RenderViewportBase.childrenInPaintOrder
func (b *RenderViewportBase) ChildrenInPaintOrder() []Sliver {
	if b.paintOrder == SliverPaintOrderLastIsTop {
		return b.childrenFirstToLast()
	}
	return b.childrenLastToFirst()
}

// ChildrenInHitTestOrder are the slivers in the order they are hit tested.
//
// Flutter: rendering/viewport.dart RenderViewportBase.childrenInHitTestOrder
func (b *RenderViewportBase) ChildrenInHitTestOrder() []Sliver {
	if b.paintOrder == SliverPaintOrderLastIsTop {
		return b.childrenLastToFirst()
	}
	return b.childrenFirstToLast()
}

func (b *RenderViewportBase) childrenLastToFirst() []Sliver {
	var children []Sliver
	for child := b.LastChild(); !isNil(child); child = b.ChildBefore(child) {
		children = append(children, child)
	}
	return children
}

func (b *RenderViewportBase) childrenFirstToLast() []Sliver {
	var children []Sliver
	for child := b.FirstChild(); !isNil(child); child = b.ChildAfter(child) {
		children = append(children, child)
	}
	return children
}

// ShowOnScreen scrolls the offset to reveal the descendant, when the offset allows
// implicit scrolling, then asks the parent to show the rectangle.
//
// Flutter: rendering/viewport.dart RenderViewportBase.showOnScreen
func (b *RenderViewportBase) ShowOnScreen(opts ShowOnScreenOpts) {
	if !b.offset.AllowImplicitScrolling() {
		b.RenderBox.ShowOnScreen(opts)
		return
	}
	newRect := RenderViewportShowInViewport(opts.Descendant, opts.Rect, b.self.(RenderAbstractViewport), b.offset, opts)
	b.RenderObjectBase.ShowOnScreen(ShowOnScreenOpts{Rect: newRect, Duration: opts.Duration, Curve: opts.Curve, Descendant: b.self})
}

// RenderViewportShowInViewport scrolls the offset so that the descendant is
// visible in the viewport. It returns the rectangle of the descendant (or of the
// rect of it) in the coordinates of the parent of the viewport, once it is
// scrolled. A nil descendant does nothing and returns the rect.
//
// Flutter: rendering/viewport.dart RenderViewportBase.showInViewport
func RenderViewportShowInViewport(descendant RenderObject, rect foundation.Opt[ui.Rect], viewport RenderAbstractViewport, offset ViewportOffset, opts ShowOnScreenOpts) foundation.Opt[ui.Rect] {
	if isNil(descendant) {
		return rect
	}
	leadingEdgeOffset := viewport.GetOffsetToReveal(descendant, 0, GetOffsetToRevealOpts{Rect: rect})
	trailingEdgeOffset := viewport.GetOffsetToReveal(descendant, 1, GetOffsetToRevealOpts{Rect: rect})
	currentOffset := offset.Pixels()
	targetOffset, ok := RevealedOffsetClampOffset(leadingEdgeOffset, trailingEdgeOffset, currentOffset).Get()
	if !ok {
		// The descendant is fully visible: nothing to scroll.
		transform := descendant.GetTransformTo(viewport.Parent())
		r := rect.OrElse(descendant.PaintBounds())
		return foundation.Some(transform.TransformRect(r))
	}
	ViewportOffsetMoveTo(offset, targetOffset.Offset, ViewportOffsetMoveToOpts{Duration: foundation.Some(opts.Duration), Curve: opts.Curve})
	return foundation.Some(targetOffset.Rect)
}

// RenderViewport is a viewport with a center sliver: the slivers before it grow in
// the reverse direction, and the ones from it on in the forward direction, so
// the content can extend in both directions from the anchor.
//
// Flutter: rendering/viewport.dart RenderViewport
type RenderViewport struct {
	RenderViewportBase
	anchor           float64
	center           Sliver
	minScrollExtent  float64
	maxScrollExtent  float64
	hasVisualOverflo bool
}

// RenderViewportOpts are the optional parameters of NewRenderViewport.
//
// Flutter: rendering/viewport.dart RenderViewport.new
type RenderViewportOpts struct {
	RenderViewportBaseOpts
	// Anchor is the relative position of the zero scroll offset, from 0.0 (the
	// leading edge) to 1.0 (the trailing edge).
	Anchor float64
	// Children are the slivers.
	Children []Sliver
	// Center is the sliver at the zero scroll offset; the default is the first
	// child.
	Center Sliver
}

// NewRenderViewport creates the viewport. In debug builds an anchor outside 0..1
// panics with the tag ViewportAnchorInvalid.
//
// Flutter: rendering/viewport.dart RenderViewport.new
func NewRenderViewport(crossAxisDirection painting.AxisDirection, offset ViewportOffset, opts ...RenderViewportOpts) *RenderViewport {
	r := &RenderViewport{}
	InitRenderViewport(r, r, crossAxisDirection, offset, opts...)
	return r
}

// InitRenderViewport sets up the viewport; self is the outer render object, for a
// type that embeds RenderViewport. The panics of NewRenderViewport apply.
//
// Flutter: rendering/viewport.dart RenderViewport.new
func InitRenderViewport(r *RenderViewport, self Box, crossAxisDirection painting.AxisDirection, offset ViewportOffset, opts ...RenderViewportOpts) {
	o := firstOpts(opts)
	if foundation.DebugMode && (o.Anchor < 0 || o.Anchor > 1) {
		panic(errors.New("ViewportAnchorInvalid", "the anchor of a viewport must be between 0.0 and 1.0", "anchor", o.Anchor))
	}
	r.anchor, r.center = o.Anchor, o.Center
	InitRenderViewportBase(&r.RenderViewportBase, self, crossAxisDirection, offset, o.RenderViewportBaseOpts)
	r.AddAll(o.Children)
	if isNil(o.Center) && !isNil(r.FirstChild()) {
		r.center = r.FirstChild()
	}
}

// AsRenderViewport is the viewport that the render object embeds.
//
// Flutter: no counterpart (Dart subclasses are the viewport)
func (r *RenderViewport) AsRenderViewport() *RenderViewport { return r }

// SetupParentData gives the child a SliverPhysicalContainerParentData if it has
// none.
//
// Flutter: rendering/viewport.dart RenderViewport.setupParentData
func (r *RenderViewport) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*SliverPhysicalContainerParentData); !ok {
		child.AsRenderObject().SetParentData(&SliverPhysicalContainerParentData{})
	}
}

// Anchor is the relative position of the zero scroll offset.
//
// Flutter: rendering/viewport.dart RenderViewport.anchor
func (r *RenderViewport) Anchor() float64 { return r.anchor }

// SetAnchor changes the anchor and marks the viewport as needing layout.
//
// Flutter: rendering/viewport.dart RenderViewport.anchor
func (r *RenderViewport) SetAnchor(value float64) {
	if foundation.DebugMode && (value < 0 || value > 1) {
		panic(errors.New("ViewportAnchorInvalid", "the anchor of a viewport must be between 0.0 and 1.0", "anchor", value))
	}
	if value == r.anchor {
		return
	}
	r.anchor = value
	r.MarkNeedsLayout()
}

// Center is the sliver at the zero scroll offset, or nil.
//
// Flutter: rendering/viewport.dart RenderViewport.center
func (r *RenderViewport) Center() Sliver { return r.center }

// SetCenter changes the center sliver and marks the viewport as needing layout.
//
// Flutter: rendering/viewport.dart RenderViewport.center
func (r *RenderViewport) SetCenter(value Sliver) {
	if value == r.center {
		return
	}
	r.center = value
	r.MarkNeedsLayout()
}

// SizedByParent is true: the viewport takes as much room as it is given.
//
// Flutter: rendering/viewport.dart RenderViewport.sizedByParent
func (r *RenderViewport) SizedByParent() bool { return true }

// ComputeDryLayout is the biggest size the constraints allow.
//
// Flutter: rendering/viewport.dart RenderViewport.computeDryLayout
func (r *RenderViewport) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	DebugCheckHasBoundedAxis(r.Axis(), constraints)
	return constraints.Biggest()
}

const maxLayoutCyclesPerChild = 10

// PerformLayout lays the slivers out, starting from the center, until the offset
// accepts the content dimensions.
//
// Flutter: rendering/viewport.dart RenderViewport.performLayout
func (r *RenderViewport) PerformLayout() {
	size := r.Size()
	if r.Axis() == painting.AxisVertical {
		r.offset.ApplyViewportDimension(size.Height)
	} else {
		r.offset.ApplyViewportDimension(size.Width)
	}

	if isNil(r.center) {
		r.minScrollExtent = 0
		r.maxScrollExtent = 0
		r.hasVisualOverflo = false
		r.offset.ApplyContentDimensions(0, 0)
		return
	}

	mainAxisExtent, crossAxisExtent := size.Height, size.Width
	if r.Axis() == painting.AxisHorizontal {
		mainAxisExtent, crossAxisExtent = size.Width, size.Height
	}

	centerOffsetAdjustment := r.center.CenterOffsetAdjustment()
	maxLayoutCycles := maxLayoutCyclesPerChild * r.ChildCount()
	count := 0
	for {
		correction := r.attemptLayout(mainAxisExtent, crossAxisExtent, r.offset.Pixels()+centerOffsetAdjustment)
		if correction != 0 {
			r.offset.CorrectBy(correction)
		} else if r.offset.ApplyContentDimensions(
			math.Min(0, r.minScrollExtent+mainAxisExtent*r.anchor),
			math.Max(0, r.maxScrollExtent-mainAxisExtent*(1-r.anchor))) {
			break
		}
		count++
		if count >= maxLayoutCycles {
			break
		}
	}
	if foundation.DebugMode && count >= maxLayoutCycles {
		panic(errors.New("ViewportLayoutCyclesExceeded", "A RenderViewport exceeded its maximum number of layout cycles. "+
			"RenderViewport render objects, during layout, can retry if either their slivers or their ViewportOffset decide that the offset "+
			"should be corrected to take into account information collected during that layout. In the case of this RenderViewport object, "+
			"however, this happened "+strconv.Itoa(count)+" times and still there was no consensus on the scroll offset. This usually indicates a bug. "+
			"Specifically, it means that one of the following three problems is being experienced by the RenderViewport object: "+
			"(1) One of the RenderSliver children or the ViewportOffset have a bug such that they always think that they need to correct the "+
			"offset regardless. (2) Some combination of the RenderSliver children and the ViewportOffset have a bad interaction such that one "+
			"applies a correction then another applies a reverse correction, leading to an infinite loop of corrections. (3) There is a "+
			"pathological case that would eventually resolve, but it is so complicated that it cannot be resolved in any reasonable number of "+
			"layout passes.", "object", foundation.DescribeIdentity(r.self), "cycles", count))
	}
}

func (r *RenderViewport) attemptLayout(mainAxisExtent, crossAxisExtent, correctedOffset float64) float64 {
	if foundation.DebugMode && (math.IsNaN(mainAxisExtent) || mainAxisExtent < 0 || math.IsInf(mainAxisExtent, 0) ||
		math.IsInf(crossAxisExtent, 0) || crossAxisExtent < 0 || math.IsInf(correctedOffset, 0) || math.IsNaN(correctedOffset)) {
		panic(errors.New("ViewportExtentInvalid", "the extents and the offset of a viewport must be finite and not negative",
			"mainAxisExtent", mainAxisExtent, "crossAxisExtent", crossAxisExtent, "correctedOffset", correctedOffset))
	}
	r.minScrollExtent = 0
	r.maxScrollExtent = 0
	r.hasVisualOverflo = false

	// centerOffset is the offset from the leading edge of the RenderViewport to
	// the zero scroll offset (the leading edge of the center sliver).
	centerOffset := mainAxisExtent*r.anchor - correctedOffset
	reverseDirectionRemainingPaintExtent := ui.ClampDouble(centerOffset, 0, mainAxisExtent)
	forwardDirectionRemainingPaintExtent := ui.ClampDouble(mainAxisExtent-centerOffset, 0, mainAxisExtent)

	cache := r.scrollCacheExtent.CalculateCacheOffset(mainAxisExtent)
	r.calculatedCacheExtent = foundation.Some(cache)
	fullCacheExtent := mainAxisExtent + 2*cache
	centerCacheOffset := centerOffset + cache
	reverseDirectionRemainingCacheExtent := ui.ClampDouble(centerCacheOffset, 0, fullCacheExtent)
	forwardDirectionRemainingCacheExtent := ui.ClampDouble(fullCacheExtent-centerCacheOffset, 0, fullCacheExtent)

	leadingNegativeChild := r.ChildBefore(r.center)
	if !isNil(leadingNegativeChild) {
		// negative scroll offsets
		result := r.LayoutChildSequence(LayoutChildSequenceOpts{
			Child:                leadingNegativeChild,
			ScrollOffset:         math.Max(mainAxisExtent, centerOffset) - mainAxisExtent,
			Overlap:              0,
			LayoutOffset:         forwardDirectionRemainingPaintExtent,
			RemainingPaintExtent: reverseDirectionRemainingPaintExtent,
			MainAxisExtent:       mainAxisExtent,
			CrossAxisExtent:      crossAxisExtent,
			GrowthDirection:      GrowthDirectionReverse,
			Advance:              r.ChildBefore,
			RemainingCacheExtent: reverseDirectionRemainingCacheExtent,
			CacheOrigin:          ui.ClampDouble(mainAxisExtent-centerOffset, -cache, 0),
		})
		if result != 0 {
			return -result
		}
	}

	// positive scroll offsets
	overlap := 0.0
	if isNil(leadingNegativeChild) {
		overlap = math.Min(0, -centerOffset)
	}
	layoutOffset := reverseDirectionRemainingPaintExtent
	if centerOffset >= mainAxisExtent {
		layoutOffset = centerOffset
	}
	return r.LayoutChildSequence(LayoutChildSequenceOpts{
		Child:                r.center,
		ScrollOffset:         math.Max(0, -centerOffset),
		Overlap:              overlap,
		LayoutOffset:         layoutOffset,
		RemainingPaintExtent: forwardDirectionRemainingPaintExtent,
		MainAxisExtent:       mainAxisExtent,
		CrossAxisExtent:      crossAxisExtent,
		GrowthDirection:      GrowthDirectionForward,
		Advance:              r.ChildAfter,
		RemainingCacheExtent: forwardDirectionRemainingCacheExtent,
		CacheOrigin:          ui.ClampDouble(centerOffset, -cache, 0),
	})
}

// HasVisualOverflow is whether a sliver overflowed.
//
// Flutter: rendering/viewport.dart RenderViewport.hasVisualOverflow
func (r *RenderViewport) HasVisualOverflow() bool { return r.hasVisualOverflo }

// UpdateOutOfBandData accumulates the scroll extents.
//
// Flutter: rendering/viewport.dart RenderViewport.updateOutOfBandData
func (r *RenderViewport) UpdateOutOfBandData(growthDirection GrowthDirection, childLayoutGeometry SliverGeometry) {
	if growthDirection == GrowthDirectionForward {
		r.maxScrollExtent += childLayoutGeometry.ScrollExtent
	} else {
		r.minScrollExtent -= childLayoutGeometry.ScrollExtent
	}
	if childLayoutGeometry.HasVisualOverflow {
		r.hasVisualOverflo = true
	}
}

// UpdateChildLayoutOffset sets the paint offset of the sliver.
//
// Flutter: rendering/viewport.dart RenderViewport.updateChildLayoutOffset
func (r *RenderViewport) UpdateChildLayoutOffset(child Sliver, layoutOffset float64, growthDirection GrowthDirection) {
	SliverPhysicalParentDataOf(child).PaintOffset = r.ComputeAbsolutePaintOffset(child, layoutOffset, growthDirection)
}

// PaintOffsetOf is the paint offset of the sliver.
//
// Flutter: rendering/viewport.dart RenderViewport.paintOffsetOf
func (r *RenderViewport) PaintOffsetOf(child Sliver) ui.Offset {
	return SliverPhysicalParentDataOf(child).PaintOffset
}

// ScrollOffsetOf converts an offset within the sliver to a scroll offset of the
// viewport.
//
// Flutter: rendering/viewport.dart RenderViewport.scrollOffsetOf
func (r *RenderViewport) ScrollOffsetOf(child Sliver, scrollOffsetWithinChild float64) float64 {
	if child.SliverConstraints().GrowthDirection == GrowthDirectionForward {
		scrollOffsetToChild := 0.0
		for current := r.center; current != child; current = r.ChildAfter(current) {
			scrollOffsetToChild += current.Geometry().ScrollExtent
		}
		return scrollOffsetToChild + scrollOffsetWithinChild
	}
	scrollOffsetToChild := 0.0
	for current := r.ChildBefore(r.center); current != child; current = r.ChildBefore(current) {
		scrollOffsetToChild -= current.Geometry().ScrollExtent
	}
	return scrollOffsetToChild - scrollOffsetWithinChild
}

// MaxScrollObstructionExtentBefore is how much the slivers before the child
// obstruct the viewport.
//
// Flutter: rendering/viewport.dart RenderViewport.maxScrollObstructionExtentBefore
func (r *RenderViewport) MaxScrollObstructionExtentBefore(child Sliver) float64 {
	pinnedExtent := 0.0
	if child.SliverConstraints().GrowthDirection == GrowthDirectionForward {
		for current := r.center; current != child; current = r.ChildAfter(current) {
			pinnedExtent += current.Geometry().MaxScrollObstructionExtent
		}
		return pinnedExtent
	}
	for current := r.ChildBefore(r.center); current != child; current = r.ChildBefore(current) {
		pinnedExtent += current.Geometry().MaxScrollObstructionExtent
	}
	return pinnedExtent
}

// ApplyPaintTransform translates by the paint offset of the child.
//
// Flutter: rendering/viewport.dart RenderViewport.applyPaintTransform
func (r *RenderViewport) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	SliverPhysicalParentDataOf(child).ApplyPaintTransform(transform)
}

// ComputeChildMainAxisPosition converts a main axis position of the viewport to
// that of the child.
//
// Flutter: rendering/viewport.dart RenderViewport.computeChildMainAxisPosition
func (r *RenderViewport) ComputeChildMainAxisPosition(child Sliver, parentMainAxisPosition float64) float64 {
	paintOffset := SliverPhysicalParentDataOf(child).PaintOffset
	c := child.SliverConstraints()
	switch ApplyGrowthDirectionToAxisDirection(c.AxisDirection, c.GrowthDirection) {
	case painting.AxisDirectionRight:
		return parentMainAxisPosition - paintOffset.Dx
	case painting.AxisDirectionUp:
		return child.Geometry().PaintExtent - (parentMainAxisPosition - paintOffset.Dy)
	case painting.AxisDirectionLeft:
		return child.Geometry().PaintExtent - (parentMainAxisPosition - paintOffset.Dx)
	}
	return parentMainAxisPosition - paintOffset.Dy
}

// IndexOfFirstChild is minus the number of children before the center.
//
// Flutter: rendering/viewport.dart RenderViewport.indexOfFirstChild
func (r *RenderViewport) IndexOfFirstChild() int {
	count := 0
	for child := r.center; child != r.FirstChild(); child = r.ChildBefore(child) {
		count--
	}
	return count
}

// LabelForChild is "center child" for the center and "child N" for the others.
//
// Flutter: rendering/viewport.dart RenderViewport.labelForChild
func (r *RenderViewport) LabelForChild(index int) string {
	if index == 0 {
		return "center child"
	}
	return "child " + strconv.Itoa(index)
}

// DebugFillProperties adds the anchor.
//
// Flutter: rendering/viewport.dart RenderViewport.debugFillProperties
func (r *RenderViewport) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	r.RenderViewportBase.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("anchor", r.anchor))
}

// RenderShrinkWrappingViewport is a viewport that is as big as its content in the
// main axis (up to the constraints). Its slivers all grow forward from the first.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport
type RenderShrinkWrappingViewport struct {
	RenderViewportBase
	maxScrollExtent  float64
	shrinkWrapExtent float64
	hasVisualOverflo bool
}

// RenderShrinkWrappingViewportOpts are the optional parameters of
// NewRenderShrinkWrappingViewport.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.new
type RenderShrinkWrappingViewportOpts struct {
	RenderViewportBaseOpts
	// Children are the slivers.
	Children []Sliver
}

// NewRenderShrinkWrappingViewport creates the viewport.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.new
func NewRenderShrinkWrappingViewport(crossAxisDirection painting.AxisDirection, offset ViewportOffset, opts ...RenderShrinkWrappingViewportOpts) *RenderShrinkWrappingViewport {
	o := firstOpts(opts)
	r := &RenderShrinkWrappingViewport{}
	InitRenderViewportBase(&r.RenderViewportBase, r, crossAxisDirection, offset, o.RenderViewportBaseOpts)
	r.AddAll(o.Children)
	return r
}

// SetupParentData gives the child a SliverLogicalContainerParentData if it has none.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.setupParentData
func (r *RenderShrinkWrappingViewport) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*SliverLogicalContainerParentData); !ok {
		child.AsRenderObject().SetParentData(&SliverLogicalContainerParentData{})
	}
}

// debugThrowIfNotCheckingIntrinsics has its own message.
func (r *RenderShrinkWrappingViewport) debugThrowIfNotCheckingIntrinsics() {
	if foundation.DebugMode && !r.dbg().checkingIntrinsics {
		panic(errors.New("ViewportIntrinsicsUnsupported", foundation.RuntimeTypeName(r.self)+" does not support returning intrinsic dimensions. "+
			"Calculating the intrinsic dimensions would require instantiating every child of the viewport, which defeats the point of "+
			"viewports being lazy. If you are merely trying to shrink-wrap the viewport in the main axis direction, you should be able to "+
			"achieve that effect by just giving the viewport loose constraints, without needing to measure its intrinsic dimensions.",
			"object", foundation.DescribeIdentity(r.self)))
	}
}

func (r *RenderShrinkWrappingViewport) debugCheckHasBoundedCrossAxis(constraints BoxConstraints) {
	if !foundation.DebugMode {
		return
	}
	if r.Axis() == painting.AxisVertical {
		if !constraints.HasBoundedWidth() {
			panic(errors.New("ViewportUnboundedCrossAxis", "Vertical viewport was given unbounded width. Viewports expand in the cross axis "+
				"to fill their container and constrain their children to match their extent in the cross axis. In this case, a vertical "+
				"shrinkwrapping viewport was given an unlimited amount of horizontal space in which to expand."))
		}
	} else if !constraints.HasBoundedHeight() {
		panic(errors.New("ViewportUnboundedCrossAxis", "Horizontal viewport was given unbounded height. Viewports expand in the cross axis "+
			"to fill their container and constrain their children to match their extent in the cross axis. In this case, a horizontal "+
			"shrinkwrapping viewport was given an unlimited amount of vertical space in which to expand."))
	}
}

// PerformLayout lays the slivers out until the offset accepts the viewport and
// content dimensions, then sizes the viewport to the content.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.performLayout
func (r *RenderShrinkWrappingViewport) PerformLayout() {
	constraints := r.BoxConstraints()
	if isNil(r.FirstChild()) {
		r.debugCheckHasBoundedCrossAxis(constraints)
		if r.Axis() == painting.AxisVertical {
			r.SetSize(ui.Size{Width: constraints.MaxWidth, Height: constraints.MinHeight})
		} else {
			r.SetSize(ui.Size{Width: constraints.MinWidth, Height: constraints.MaxHeight})
		}
		r.offset.ApplyViewportDimension(0)
		r.maxScrollExtent = 0
		r.shrinkWrapExtent = 0
		r.hasVisualOverflo = false
		r.offset.ApplyContentDimensions(0, 0)
		return
	}
	r.debugCheckHasBoundedCrossAxis(constraints)

	mainAxisExtent, crossAxisExtent := constraints.MaxHeight, constraints.MaxWidth
	if r.Axis() == painting.AxisHorizontal {
		mainAxisExtent, crossAxisExtent = constraints.MaxWidth, constraints.MaxHeight
	}

	var effectiveExtent float64
	for {
		correction := r.attemptLayout(mainAxisExtent, crossAxisExtent, r.offset.Pixels())
		if correction != 0 {
			r.offset.CorrectBy(correction)
			continue
		}
		if r.Axis() == painting.AxisVertical {
			effectiveExtent = constraints.ConstrainHeight(r.shrinkWrapExtent)
		} else {
			effectiveExtent = constraints.ConstrainWidth(r.shrinkWrapExtent)
		}
		// Make both calls, so the offset sees both dimensions.
		didAcceptViewportDimension := r.offset.ApplyViewportDimension(effectiveExtent)
		didAcceptContentDimension := r.offset.ApplyContentDimensions(0, math.Max(0, r.maxScrollExtent-effectiveExtent))
		if didAcceptViewportDimension && didAcceptContentDimension {
			break
		}
	}
	if r.Axis() == painting.AxisVertical {
		r.SetSize(constraints.ConstrainDimensions(crossAxisExtent, effectiveExtent))
	} else {
		r.SetSize(constraints.ConstrainDimensions(effectiveExtent, crossAxisExtent))
	}
}

func (r *RenderShrinkWrappingViewport) attemptLayout(mainAxisExtent, crossAxisExtent, correctedOffset float64) float64 {
	if foundation.DebugMode && (math.IsNaN(mainAxisExtent) || mainAxisExtent < 0 || math.IsInf(crossAxisExtent, 0) ||
		crossAxisExtent < 0 || math.IsInf(correctedOffset, 0) || math.IsNaN(correctedOffset)) {
		panic(errors.New("ViewportExtentInvalid", "the extents and the offset of a viewport must be finite and not negative",
			"mainAxisExtent", mainAxisExtent, "crossAxisExtent", crossAxisExtent, "correctedOffset", correctedOffset))
	}
	r.maxScrollExtent = 0
	r.shrinkWrapExtent = 0
	// The viewport has overflow if the offset is negative: the content can not
	// be scrolled but the top of it is hidden.
	r.hasVisualOverflo = correctedOffset < 0
	cache := 0.0
	if !math.IsInf(mainAxisExtent, 0) {
		cache = r.scrollCacheExtent.CalculateCacheOffset(mainAxisExtent)
	}
	r.calculatedCacheExtent = foundation.Some(cache)
	return r.LayoutChildSequence(LayoutChildSequenceOpts{
		Child:                r.FirstChild(),
		ScrollOffset:         math.Max(0, correctedOffset),
		Overlap:              math.Min(0, correctedOffset),
		LayoutOffset:         math.Max(0, -correctedOffset),
		RemainingPaintExtent: mainAxisExtent + math.Min(0, correctedOffset),
		MainAxisExtent:       mainAxisExtent,
		CrossAxisExtent:      crossAxisExtent,
		GrowthDirection:      GrowthDirectionForward,
		Advance:              r.ChildAfter,
		RemainingCacheExtent: mainAxisExtent + 2*cache,
		CacheOrigin:          -cache,
	})
}

// HasVisualOverflow is whether a sliver overflowed.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.hasVisualOverflow
func (r *RenderShrinkWrappingViewport) HasVisualOverflow() bool { return r.hasVisualOverflo }

// UpdateOutOfBandData accumulates the scroll extent and the extent to wrap.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.updateOutOfBandData
func (r *RenderShrinkWrappingViewport) UpdateOutOfBandData(_ GrowthDirection, childLayoutGeometry SliverGeometry) {
	r.maxScrollExtent += childLayoutGeometry.ScrollExtent
	if childLayoutGeometry.HasVisualOverflow {
		r.hasVisualOverflo = true
	}
	r.shrinkWrapExtent += childLayoutGeometry.MaxPaintExtent
}

// UpdateChildLayoutOffset stores the layout offset of the sliver.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.updateChildLayoutOffset
func (r *RenderShrinkWrappingViewport) UpdateChildLayoutOffset(child Sliver, layoutOffset float64, _ GrowthDirection) {
	SliverLogicalParentDataOf(child).LayoutOffset = foundation.Some(layoutOffset)
}

// PaintOffsetOf is the paint offset of the sliver, from its layout offset.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.paintOffsetOf
func (r *RenderShrinkWrappingViewport) PaintOffsetOf(child Sliver) ui.Offset {
	return r.ComputeAbsolutePaintOffset(child, SliverLogicalParentDataOf(child).LayoutOffset.Value(), GrowthDirectionForward)
}

// ScrollOffsetOf converts an offset within the sliver to a scroll offset.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.scrollOffsetOf
func (r *RenderShrinkWrappingViewport) ScrollOffsetOf(child Sliver, scrollOffsetWithinChild float64) float64 {
	scrollOffsetToChild := 0.0
	for current := r.FirstChild(); current != child; current = r.ChildAfter(current) {
		scrollOffsetToChild += current.Geometry().ScrollExtent
	}
	return scrollOffsetToChild + scrollOffsetWithinChild
}

// MaxScrollObstructionExtentBefore is how much the slivers before the child
// obstruct the viewport.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.maxScrollObstructionExtentBefore
func (r *RenderShrinkWrappingViewport) MaxScrollObstructionExtentBefore(child Sliver) float64 {
	pinnedExtent := 0.0
	for current := r.FirstChild(); current != child; current = r.ChildAfter(current) {
		pinnedExtent += current.Geometry().MaxScrollObstructionExtent
	}
	return pinnedExtent
}

// ApplyPaintTransform translates by the paint offset of the child.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.applyPaintTransform
func (r *RenderShrinkWrappingViewport) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	offset := r.PaintOffsetOf(child.(Sliver))
	transform.Translate(float32(offset.Dx), float32(offset.Dy), 0)
}

// ComputeChildMainAxisPosition converts a main axis position of the viewport to
// that of the child.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.computeChildMainAxisPosition
func (r *RenderShrinkWrappingViewport) ComputeChildMainAxisPosition(child Sliver, parentMainAxisPosition float64) float64 {
	layoutOffset := SliverLogicalParentDataOf(child).LayoutOffset.Value()
	c := child.SliverConstraints()
	switch ApplyGrowthDirectionToAxisDirection(c.AxisDirection, c.GrowthDirection) {
	case painting.AxisDirectionUp:
		return r.Size().Height - parentMainAxisPosition - layoutOffset
	case painting.AxisDirectionLeft:
		return r.Size().Width - parentMainAxisPosition - layoutOffset
	}
	return parentMainAxisPosition - layoutOffset
}

// IndexOfFirstChild is zero.
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.indexOfFirstChild
func (r *RenderShrinkWrappingViewport) IndexOfFirstChild() int { return 0 }

// LabelForChild is "child N".
//
// Flutter: rendering/viewport.dart RenderShrinkWrappingViewport.labelForChild
func (r *RenderShrinkWrappingViewport) LabelForChild(index int) string {
	return "child " + strconv.Itoa(index)
}
