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
	"github.com/rm4n0s/glutter/painting"
)

// SliverGridGeometry describes the placement of a child in a SliverGridLayout.
//
// Flutter: rendering/sliver_grid.dart SliverGridGeometry
type SliverGridGeometry struct {
	// ScrollOffset is the scroll offset of the leading edge of the child relative
	// to the leading edge of the parent.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridGeometry.scrollOffset
	ScrollOffset float64
	// CrossAxisOffset is the offset of the child in the non-scrolling axis.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridGeometry.crossAxisOffset
	CrossAxisOffset float64
	// MainAxisExtent is the extent of the child in the scrolling axis.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridGeometry.mainAxisExtent
	MainAxisExtent float64
	// CrossAxisExtent is the extent of the child in the non-scrolling axis.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridGeometry.crossAxisExtent
	CrossAxisExtent float64
}

// TrailingScrollOffset is the scroll offset of the trailing edge of the child
// relative to the leading edge of the parent.
//
// Flutter: rendering/sliver_grid.dart SliverGridGeometry.trailingScrollOffset
func (g SliverGridGeometry) TrailingScrollOffset() float64 { return g.ScrollOffset + g.MainAxisExtent }

// GetBoxConstraints is the constraints that tightly fit the child in the
// geometry.
//
// Flutter: rendering/sliver_grid.dart SliverGridGeometry.getBoxConstraints
func (g SliverGridGeometry) GetBoxConstraints(constraints SliverConstraints) BoxConstraints {
	return constraints.AsBoxConstraints(AsBoxConstraintsOpts{
		MinExtent:       g.MainAxisExtent,
		MaxExtent:       foundation.Some(g.MainAxisExtent),
		CrossAxisExtent: foundation.Some(g.CrossAxisExtent),
	})
}

// String describes the geometry.
//
// Flutter: rendering/sliver_grid.dart SliverGridGeometry.toString
func (g SliverGridGeometry) String() string {
	return "SliverGridGeometry(scrollOffset: " + dartDouble(g.ScrollOffset) + ", crossAxisOffset: " + dartDouble(g.CrossAxisOffset) +
		", mainAxisExtent: " + dartDouble(g.MainAxisExtent) + ", crossAxisExtent: " + dartDouble(g.CrossAxisExtent) + ")"
}

// SliverGridLayout describes the placement of the children in a RenderSliverGrid.
//
// Flutter: rendering/sliver_grid.dart SliverGridLayout
type SliverGridLayout interface {
	// GetMinChildIndexForScrollOffset is the minimum child index that is visible
	// at the scroll offset.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridLayout.getMinChildIndexForScrollOffset
	GetMinChildIndexForScrollOffset(scrollOffset float64) int
	// GetMaxChildIndexForScrollOffset is the maximum child index that is visible
	// at the scroll offset.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridLayout.getMaxChildIndexForScrollOffset
	GetMaxChildIndexForScrollOffset(scrollOffset float64) int
	// GetGeometryForChildIndex is the geometry of the child at the index.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridLayout.getGeometryForChildIndex
	GetGeometryForChildIndex(index int) SliverGridGeometry
	// ComputeMaxScrollOffset is the scroll extent needed to fit the children.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridLayout.computeMaxScrollOffset
	ComputeMaxScrollOffset(childCount int) float64
}

// SliverGridRegularTileLayout is a layout of tiles of the same size, in a grid
// with a fixed number of tiles in the cross axis.
//
// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout
type SliverGridRegularTileLayout struct {
	// CrossAxisCount is the number of children in the cross axis.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout.crossAxisCount
	CrossAxisCount int
	// MainAxisStride is the number of pixels from the leading edge of one tile to
	// the leading edge of the next tile in the main axis.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout.mainAxisStride
	MainAxisStride float64
	// CrossAxisStride is the same, in the cross axis.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout.crossAxisStride
	CrossAxisStride float64
	// ChildMainAxisExtent is the number of pixels from the leading edge of one
	// tile to the trailing edge of the same tile in the main axis.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout.childMainAxisExtent
	ChildMainAxisExtent float64
	// ChildCrossAxisExtent is the same, in the cross axis.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout.childCrossAxisExtent
	ChildCrossAxisExtent float64
	// ReverseCrossAxis is whether the children are laid out in the opposite
	// direction of the cross axis.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout.reverseCrossAxis
	ReverseCrossAxis bool
}

// NewSliverGridRegularTileLayout creates the layout. In debug builds a negative
// value, or a count that is not positive, panics with the tag
// SliverGridLayoutInvalid.
//
// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout.new
func NewSliverGridRegularTileLayout(crossAxisCount int, mainAxisStride, crossAxisStride, childMainAxisExtent, childCrossAxisExtent float64, reverseCrossAxis bool) *SliverGridRegularTileLayout {
	if foundation.DebugMode && (crossAxisCount <= 0 || !(mainAxisStride >= 0) || !(crossAxisStride >= 0) ||
		!(childMainAxisExtent >= 0) || !(childCrossAxisExtent >= 0)) {
		panic(errors.New("SliverGridLayoutInvalid", "a regular tile layout needs a positive cross axis count and strides and extents that are not negative",
			"crossAxisCount", crossAxisCount, "mainAxisStride", mainAxisStride, "crossAxisStride", crossAxisStride,
			"childMainAxisExtent", childMainAxisExtent, "childCrossAxisExtent", childCrossAxisExtent))
	}
	return &SliverGridRegularTileLayout{crossAxisCount, mainAxisStride, crossAxisStride, childMainAxisExtent, childCrossAxisExtent, reverseCrossAxis}
}

// GetMinChildIndexForScrollOffset is the index of the first child of the row at
// the scroll offset.
//
// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout.getMinChildIndexForScrollOffset
func (l *SliverGridRegularTileLayout) GetMinChildIndexForScrollOffset(scrollOffset float64) int {
	if l.MainAxisStride > foundation.PrecisionErrorTolerance {
		return l.CrossAxisCount * int(math.Trunc(scrollOffset/l.MainAxisStride))
	}
	return 0
}

// GetMaxChildIndexForScrollOffset is the index of the last child of the last row
// at the scroll offset.
//
// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout.getMaxChildIndexForScrollOffset
func (l *SliverGridRegularTileLayout) GetMaxChildIndexForScrollOffset(scrollOffset float64) int {
	if l.MainAxisStride > 0 {
		mainAxisCount := int(math.Ceil(scrollOffset / l.MainAxisStride))
		if v := l.CrossAxisCount*mainAxisCount - 1; v > 0 {
			return v
		}
	}
	return 0
}

func (l *SliverGridRegularTileLayout) getOffsetFromStartInCrossAxis(crossAxisStart float64) float64 {
	if l.ReverseCrossAxis {
		return float64(l.CrossAxisCount)*l.CrossAxisStride - crossAxisStart - l.ChildCrossAxisExtent - (l.CrossAxisStride - l.ChildCrossAxisExtent)
	}
	return crossAxisStart
}

// GetGeometryForChildIndex is the geometry of the child at the index.
//
// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout.getGeometryForChildIndex
func (l *SliverGridRegularTileLayout) GetGeometryForChildIndex(index int) SliverGridGeometry {
	crossAxisStart := float64(index%l.CrossAxisCount) * l.CrossAxisStride
	return SliverGridGeometry{
		ScrollOffset:    float64(index/l.CrossAxisCount) * l.MainAxisStride,
		CrossAxisOffset: l.getOffsetFromStartInCrossAxis(crossAxisStart),
		MainAxisExtent:  l.ChildMainAxisExtent,
		CrossAxisExtent: l.ChildCrossAxisExtent,
	}
}

// ComputeMaxScrollOffset is the extent of the rows, without the spacing after
// the last.
//
// Flutter: rendering/sliver_grid.dart SliverGridRegularTileLayout.computeMaxScrollOffset
func (l *SliverGridRegularTileLayout) ComputeMaxScrollOffset(childCount int) float64 {
	if childCount == 0 {
		// There are no children in the grid. The max scroll offset should be
		// zero.
		return 0
	}
	mainAxisCount := (childCount-1)/l.CrossAxisCount + 1
	mainAxisSpacing := l.MainAxisStride - l.ChildMainAxisExtent
	return l.MainAxisStride*float64(mainAxisCount) - mainAxisSpacing
}

// SliverGridDelegate controls the layout of the children in a RenderSliverGrid.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegate
type SliverGridDelegate interface {
	// GetLayout is the layout for the constraints.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridDelegate.getLayout
	GetLayout(constraints SliverConstraints) SliverGridLayout
	// ShouldRelayout is whether the grid has to be laid out again if the delegate
	// replaces the old one (of the same type).
	//
	// Flutter: rendering/sliver_grid.dart SliverGridDelegate.shouldRelayout
	ShouldRelayout(oldDelegate SliverGridDelegate) bool
}

func assertGridSpacing(mainAxisSpacing, crossAxisSpacing, childAspectRatio float64, mainAxisExtent foundation.Opt[float64]) {
	if foundation.DebugMode && (!(mainAxisSpacing >= 0) || !(crossAxisSpacing >= 0) || !(childAspectRatio > 0) ||
		(mainAxisExtent.IsSome() && !(mainAxisExtent.Value() >= 0))) {
		panic(errors.New("SliverGridDelegateInvalid", "a grid delegate needs spacings that are not negative, a positive aspect ratio and a main axis extent that is not negative",
			"mainAxisSpacing", mainAxisSpacing, "crossAxisSpacing", crossAxisSpacing, "childAspectRatio", childAspectRatio))
	}
}

// SliverGridDelegateWithFixedCrossAxisCountOpts are the optional parameters of
// NewSliverGridDelegateWithFixedCrossAxisCount.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithFixedCrossAxisCount.new
type SliverGridDelegateWithFixedCrossAxisCountOpts struct {
	// MainAxisSpacing is the number of logical pixels between each child along
	// the main axis.
	MainAxisSpacing float64
	// CrossAxisSpacing is the number of logical pixels between each child along
	// the cross axis.
	CrossAxisSpacing float64
	// ChildAspectRatio is the ratio of the cross-axis to the main-axis extent of
	// each child; the default is 1.
	ChildAspectRatio foundation.Opt[float64]
	// MainAxisExtent is the extent of each tile in the main axis; if it is none,
	// the extent follows from the aspect ratio.
	MainAxisExtent foundation.Opt[float64]
}

// SliverGridDelegateWithFixedCrossAxisCount creates grids that have a fixed
// number of tiles in the cross axis.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithFixedCrossAxisCount
type SliverGridDelegateWithFixedCrossAxisCount struct {
	crossAxisCount   int
	mainAxisSpacing  float64
	crossAxisSpacing float64
	childAspectRatio float64
	mainAxisExtent   foundation.Opt[float64]
}

// NewSliverGridDelegateWithFixedCrossAxisCount creates the delegate. In debug
// builds a count that is not positive, or an invalid option, panics with the tag
// SliverGridDelegateInvalid.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithFixedCrossAxisCount.new
func NewSliverGridDelegateWithFixedCrossAxisCount(crossAxisCount int, opts ...SliverGridDelegateWithFixedCrossAxisCountOpts) *SliverGridDelegateWithFixedCrossAxisCount {
	o := firstOpts(opts)
	d := &SliverGridDelegateWithFixedCrossAxisCount{
		crossAxisCount: crossAxisCount, mainAxisSpacing: o.MainAxisSpacing, crossAxisSpacing: o.CrossAxisSpacing,
		childAspectRatio: o.ChildAspectRatio.OrElse(1), mainAxisExtent: o.MainAxisExtent,
	}
	if foundation.DebugMode && crossAxisCount <= 0 {
		panic(errors.New("SliverGridDelegateInvalid", "the cross axis count of a grid delegate must be positive", "crossAxisCount", crossAxisCount))
	}
	assertGridSpacing(d.mainAxisSpacing, d.crossAxisSpacing, d.childAspectRatio, d.mainAxisExtent)
	return d
}

// CrossAxisCount is the number of children in the cross axis.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithFixedCrossAxisCount.crossAxisCount
func (d *SliverGridDelegateWithFixedCrossAxisCount) CrossAxisCount() int { return d.crossAxisCount }

// MainAxisSpacing is the number of logical pixels between each child along the
// main axis.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithFixedCrossAxisCount.mainAxisSpacing
func (d *SliverGridDelegateWithFixedCrossAxisCount) MainAxisSpacing() float64 {
	return d.mainAxisSpacing
}

// CrossAxisSpacing is the number of logical pixels between each child along the
// cross axis.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithFixedCrossAxisCount.crossAxisSpacing
func (d *SliverGridDelegateWithFixedCrossAxisCount) CrossAxisSpacing() float64 {
	return d.crossAxisSpacing
}

// ChildAspectRatio is the ratio of the cross-axis to the main-axis extent of each
// child.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithFixedCrossAxisCount.childAspectRatio
func (d *SliverGridDelegateWithFixedCrossAxisCount) ChildAspectRatio() float64 {
	return d.childAspectRatio
}

// MainAxisExtent is the extent of each tile in the main axis, or none.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithFixedCrossAxisCount.mainAxisExtent
func (d *SliverGridDelegateWithFixedCrossAxisCount) MainAxisExtent() foundation.Opt[float64] {
	return d.mainAxisExtent
}

// GetLayout is a regular tile layout with the cross axis count.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithFixedCrossAxisCount.getLayout
func (d *SliverGridDelegateWithFixedCrossAxisCount) GetLayout(constraints SliverConstraints) SliverGridLayout {
	assertGridSpacing(d.mainAxisSpacing, d.crossAxisSpacing, d.childAspectRatio, d.mainAxisExtent)
	usableCrossAxisExtent := math.Max(0, constraints.CrossAxisExtent-d.crossAxisSpacing*float64(d.crossAxisCount-1))
	childCrossAxisExtent := usableCrossAxisExtent / float64(d.crossAxisCount)
	childMainAxisExtent := d.mainAxisExtent.OrElse(childCrossAxisExtent / d.childAspectRatio)
	return NewSliverGridRegularTileLayout(d.crossAxisCount, childMainAxisExtent+d.mainAxisSpacing, childCrossAxisExtent+d.crossAxisSpacing,
		childMainAxisExtent, childCrossAxisExtent, painting.AxisDirectionIsReversed(constraints.CrossAxisDirection))
}

// ShouldRelayout is whether a value differs.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithFixedCrossAxisCount.shouldRelayout
func (d *SliverGridDelegateWithFixedCrossAxisCount) ShouldRelayout(oldDelegate SliverGridDelegate) bool {
	old := oldDelegate.(*SliverGridDelegateWithFixedCrossAxisCount)
	return old.crossAxisCount != d.crossAxisCount || old.mainAxisSpacing != d.mainAxisSpacing ||
		old.crossAxisSpacing != d.crossAxisSpacing || old.childAspectRatio != d.childAspectRatio || old.mainAxisExtent != d.mainAxisExtent
}

// SliverGridDelegateWithMaxCrossAxisExtentOpts are the optional parameters of
// NewSliverGridDelegateWithMaxCrossAxisExtent.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithMaxCrossAxisExtent.new
type SliverGridDelegateWithMaxCrossAxisExtentOpts struct {
	// MainAxisSpacing is the number of logical pixels between each child along
	// the main axis.
	MainAxisSpacing float64
	// CrossAxisSpacing is the number of logical pixels between each child along
	// the cross axis.
	CrossAxisSpacing float64
	// ChildAspectRatio is the ratio of the cross-axis to the main-axis extent of
	// each child; the default is 1.
	ChildAspectRatio foundation.Opt[float64]
	// MainAxisExtent is the extent of each tile in the main axis; if it is none,
	// the extent follows from the aspect ratio.
	MainAxisExtent foundation.Opt[float64]
}

// SliverGridDelegateWithMaxCrossAxisExtent creates grids whose tiles have a
// maximum cross-axis extent.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithMaxCrossAxisExtent
type SliverGridDelegateWithMaxCrossAxisExtent struct {
	maxCrossAxisExtent float64
	mainAxisSpacing    float64
	crossAxisSpacing   float64
	childAspectRatio   float64
	mainAxisExtent     foundation.Opt[float64]
}

// NewSliverGridDelegateWithMaxCrossAxisExtent creates the delegate. In debug
// builds a maximum that is not positive, or an invalid option, panics with the
// tag SliverGridDelegateInvalid.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithMaxCrossAxisExtent.new
func NewSliverGridDelegateWithMaxCrossAxisExtent(maxCrossAxisExtent float64, opts ...SliverGridDelegateWithMaxCrossAxisExtentOpts) *SliverGridDelegateWithMaxCrossAxisExtent {
	o := firstOpts(opts)
	d := &SliverGridDelegateWithMaxCrossAxisExtent{
		maxCrossAxisExtent: maxCrossAxisExtent, mainAxisSpacing: o.MainAxisSpacing, crossAxisSpacing: o.CrossAxisSpacing,
		childAspectRatio: o.ChildAspectRatio.OrElse(1), mainAxisExtent: o.MainAxisExtent,
	}
	if foundation.DebugMode && !(maxCrossAxisExtent > 0) {
		panic(errors.New("SliverGridDelegateInvalid", "the maximum cross axis extent of a grid delegate must be positive", "maxCrossAxisExtent", maxCrossAxisExtent))
	}
	assertGridSpacing(d.mainAxisSpacing, d.crossAxisSpacing, d.childAspectRatio, d.mainAxisExtent)
	return d
}

// MaxCrossAxisExtent is the maximum extent of a tile in the cross axis.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithMaxCrossAxisExtent.maxCrossAxisExtent
func (d *SliverGridDelegateWithMaxCrossAxisExtent) MaxCrossAxisExtent() float64 {
	return d.maxCrossAxisExtent
}

// MainAxisSpacing is the number of logical pixels between each child along the
// main axis.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithMaxCrossAxisExtent.mainAxisSpacing
func (d *SliverGridDelegateWithMaxCrossAxisExtent) MainAxisSpacing() float64 {
	return d.mainAxisSpacing
}

// CrossAxisSpacing is the number of logical pixels between each child along the
// cross axis.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithMaxCrossAxisExtent.crossAxisSpacing
func (d *SliverGridDelegateWithMaxCrossAxisExtent) CrossAxisSpacing() float64 {
	return d.crossAxisSpacing
}

// ChildAspectRatio is the ratio of the cross-axis to the main-axis extent of each
// child.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithMaxCrossAxisExtent.childAspectRatio
func (d *SliverGridDelegateWithMaxCrossAxisExtent) ChildAspectRatio() float64 {
	return d.childAspectRatio
}

// MainAxisExtent is the extent of each tile in the main axis, or none.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithMaxCrossAxisExtent.mainAxisExtent
func (d *SliverGridDelegateWithMaxCrossAxisExtent) MainAxisExtent() foundation.Opt[float64] {
	return d.mainAxisExtent
}

// GetLayout is a regular tile layout with as few tiles in the cross axis as keep
// them under the maximum extent.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithMaxCrossAxisExtent.getLayout
func (d *SliverGridDelegateWithMaxCrossAxisExtent) GetLayout(constraints SliverConstraints) SliverGridLayout {
	if foundation.DebugMode && !(constraints.CrossAxisExtent > 0) {
		panic(errors.New("SliverGridDelegateInvalid", "a grid with a maximum cross axis extent needs a cross axis extent that is positive",
			"crossAxisExtent", constraints.CrossAxisExtent))
	}
	assertGridSpacing(d.mainAxisSpacing, d.crossAxisSpacing, d.childAspectRatio, d.mainAxisExtent)
	crossAxisCount := int(math.Ceil(constraints.CrossAxisExtent / (d.maxCrossAxisExtent + d.crossAxisSpacing)))
	if crossAxisCount < 1 {
		crossAxisCount = 1
	}
	usableCrossAxisExtent := math.Max(0, constraints.CrossAxisExtent-d.crossAxisSpacing*float64(crossAxisCount-1))
	childCrossAxisExtent := usableCrossAxisExtent / float64(crossAxisCount)
	childMainAxisExtent := d.mainAxisExtent.OrElse(childCrossAxisExtent / d.childAspectRatio)
	return NewSliverGridRegularTileLayout(crossAxisCount, childMainAxisExtent+d.mainAxisSpacing, childCrossAxisExtent+d.crossAxisSpacing,
		childMainAxisExtent, childCrossAxisExtent, painting.AxisDirectionIsReversed(constraints.CrossAxisDirection))
}

// ShouldRelayout is whether a value differs.
//
// Flutter: rendering/sliver_grid.dart SliverGridDelegateWithMaxCrossAxisExtent.shouldRelayout
func (d *SliverGridDelegateWithMaxCrossAxisExtent) ShouldRelayout(oldDelegate SliverGridDelegate) bool {
	old := oldDelegate.(*SliverGridDelegateWithMaxCrossAxisExtent)
	return old.maxCrossAxisExtent != d.maxCrossAxisExtent || old.mainAxisSpacing != d.mainAxisSpacing ||
		old.crossAxisSpacing != d.crossAxisSpacing || old.childAspectRatio != d.childAspectRatio || old.mainAxisExtent != d.mainAxisExtent
}

// SliverGridParentData is the parent data of the children of a RenderSliverGrid:
// it adds the offset in the cross axis.
//
// Flutter: rendering/sliver_grid.dart SliverGridParentData
type SliverGridParentData struct {
	SliverMultiBoxAdaptorParentData
	// CrossAxisOffset is the offset of the child in the non-scrolling axis; none
	// before layout.
	//
	// Flutter: rendering/sliver_grid.dart SliverGridParentData.crossAxisOffset
	CrossAxisOffset foundation.Opt[float64]
}

// String is the cross axis offset and what the parent data says.
//
// Flutter: rendering/sliver_grid.dart SliverGridParentData.toString
func (d *SliverGridParentData) String() string {
	offset := "null"
	if o, ok := d.CrossAxisOffset.Get(); ok {
		offset = dartDouble(o)
	}
	return "crossAxisOffset=" + offset + "; " + d.SliverMultiBoxAdaptorParentData.String()
}

// RenderSliverGrid is a sliver that places multiple box children in a
// two dimensional arrangement, lazily.
//
// Flutter: rendering/sliver_grid.dart RenderSliverGrid
type RenderSliverGrid struct {
	RenderSliverMultiBoxAdaptor
	gridDelegate SliverGridDelegate
}

// NewRenderSliverGrid creates the sliver.
//
// Flutter: rendering/sliver_grid.dart RenderSliverGrid.new
func NewRenderSliverGrid(childManager RenderSliverBoxChildManager, gridDelegate SliverGridDelegate) *RenderSliverGrid {
	r := &RenderSliverGrid{gridDelegate: gridDelegate}
	InitRenderSliverMultiBoxAdaptor(&r.RenderSliverMultiBoxAdaptor, r, childManager)
	return r
}

// SetupParentData gives the child a SliverGridParentData if it has none.
//
// Flutter: rendering/sliver_grid.dart RenderSliverGrid.setupParentData
func (r *RenderSliverGrid) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*SliverGridParentData); !ok {
		child.AsRenderObject().SetParentData(&SliverGridParentData{})
	}
}

// GridDelegate is the delegate that controls the layout of the children.
//
// Flutter: rendering/sliver_grid.dart RenderSliverGrid.gridDelegate
func (r *RenderSliverGrid) GridDelegate() SliverGridDelegate { return r.gridDelegate }

// SetGridDelegate changes the delegate; the sliver is laid out again if the
// delegate is of another type, or says it should be.
//
// Flutter: rendering/sliver_grid.dart RenderSliverGrid.gridDelegate
func (r *RenderSliverGrid) SetGridDelegate(value SliverGridDelegate) {
	if r.gridDelegate == value {
		return
	}
	if foundation.RuntimeTypeName(value) != foundation.RuntimeTypeName(r.gridDelegate) || value.ShouldRelayout(r.gridDelegate) {
		r.MarkNeedsLayout()
	}
	r.gridDelegate = value
}

func sliverGridParentDataOf(child RenderObject) *SliverGridParentData {
	return child.ParentData().(*SliverGridParentData)
}

// ChildCrossAxisPosition is the cross axis offset of the child.
//
// Flutter: rendering/sliver_grid.dart RenderSliverGrid.childCrossAxisPosition
func (r *RenderSliverGrid) ChildCrossAxisPosition(child RenderObject) float64 {
	return sliverGridParentDataOf(child).CrossAxisOffset.Value()
}

// PerformLayout lays out the children that are in the viewport and the cache
// area, whose positions the layout of the delegate gives.
//
// Flutter: rendering/sliver_grid.dart RenderSliverGrid.performLayout
func (r *RenderSliverGrid) PerformLayout() {
	constraints := r.SliverConstraints()
	r.childManager.DidStartLayout()
	r.childManager.SetDidUnderflow(false)

	scrollOffset := constraints.ScrollOffset + constraints.CacheOrigin
	remainingExtent := constraints.RemainingCacheExtent
	targetEndScrollOffset := scrollOffset + remainingExtent

	layout := r.gridDelegate.GetLayout(constraints)

	firstIndex := layout.GetMinChildIndexForScrollOffset(scrollOffset)
	targetLastIndex := foundation.None[int]()
	if !math.IsInf(targetEndScrollOffset, 0) {
		targetLastIndex = foundation.Some(layout.GetMaxChildIndexForScrollOffset(targetEndScrollOffset))
	}
	if !isNil(r.FirstChild()) {
		leadingGarbage := r.CalculateLeadingGarbage(firstIndex)
		trailingGarbage := 0
		if last, ok := targetLastIndex.Get(); ok {
			trailingGarbage = r.CalculateTrailingGarbage(last)
		}
		r.CollectGarbage(leadingGarbage, trailingGarbage)
	} else {
		r.CollectGarbage(0, 0)
	}

	firstChildGridGeometry := layout.GetGeometryForChildIndex(firstIndex)

	if isNil(r.FirstChild()) {
		if !r.AddInitialChild(AddInitialChildOpts{Index: firstIndex, LayoutOffset: firstChildGridGeometry.ScrollOffset}) {
			// There are either no children, or we are past the end of all our children.
			max := layout.ComputeMaxScrollOffset(r.childManager.ChildCount())
			r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{ScrollExtent: max, MaxPaintExtent: max}))
			r.childManager.DidFinishLayout()
			return
		}
	}

	leadingScrollOffset := firstChildGridGeometry.ScrollOffset
	trailingScrollOffset := firstChildGridGeometry.TrailingScrollOffset()
	var trailingChildWithLayout Box
	reachedEnd := false

	for index := r.IndexOf(r.FirstChild()) - 1; index >= firstIndex; index-- {
		gridGeometry := layout.GetGeometryForChildIndex(index)
		child := r.InsertAndLayoutLeadingChild(gridGeometry.GetBoxConstraints(constraints))
		data := sliverGridParentDataOf(child)
		data.LayoutOffset = foundation.Some(gridGeometry.ScrollOffset)
		data.CrossAxisOffset = foundation.Some(gridGeometry.CrossAxisOffset)
		if isNil(trailingChildWithLayout) {
			trailingChildWithLayout = child
		}
		trailingScrollOffset = math.Max(trailingScrollOffset, gridGeometry.TrailingScrollOffset())
	}

	if isNil(trailingChildWithLayout) {
		r.FirstChild().Layout(firstChildGridGeometry.GetBoxConstraints(constraints))
		data := sliverGridParentDataOf(r.FirstChild())
		data.LayoutOffset = foundation.Some(firstChildGridGeometry.ScrollOffset)
		data.CrossAxisOffset = foundation.Some(firstChildGridGeometry.CrossAxisOffset)
		trailingChildWithLayout = r.FirstChild()
	}

	for index := r.IndexOf(trailingChildWithLayout) + 1; !targetLastIndex.IsSome() || index <= targetLastIndex.Value(); index++ {
		gridGeometry := layout.GetGeometryForChildIndex(index)
		childConstraints := gridGeometry.GetBoxConstraints(constraints)
		child := r.ChildAfter(trailingChildWithLayout)
		if isNil(child) || r.IndexOf(child) != index {
			child = r.InsertAndLayoutChild(childConstraints, trailingChildWithLayout)
			if isNil(child) {
				reachedEnd = true
				break
			}
		} else {
			child.Layout(childConstraints)
		}
		trailingChildWithLayout = child
		data := sliverGridParentDataOf(child)
		data.LayoutOffset = foundation.Some(gridGeometry.ScrollOffset)
		data.CrossAxisOffset = foundation.Some(gridGeometry.CrossAxisOffset)
		trailingScrollOffset = math.Max(trailingScrollOffset, gridGeometry.TrailingScrollOffset())
	}

	lastIndex := r.IndexOf(r.LastChild())

	r.DebugAssertChildListIsNonEmptyAndContiguous()

	var estimatedTotalExtent float64
	if reachedEnd {
		estimatedTotalExtent = trailingScrollOffset
	} else {
		estimatedTotalExtent = r.childManager.EstimateMaxScrollOffset(constraints, EstimateMaxScrollOffsetOpts{
			FirstIndex:           foundation.Some(firstIndex),
			LastIndex:            foundation.Some(lastIndex),
			LeadingScrollOffset:  foundation.Some(leadingScrollOffset),
			TrailingScrollOffset: foundation.Some(trailingScrollOffset),
		})
	}
	paintExtent := r.CalculatePaintOffset(constraints, math.Min(constraints.ScrollOffset, leadingScrollOffset), trailingScrollOffset)
	cacheExtent := r.CalculateCacheOffset(constraints, leadingScrollOffset, trailingScrollOffset)

	r.SetGeometry(NewSliverGeometry(SliverGeometryOpts{
		ScrollExtent:      estimatedTotalExtent,
		PaintExtent:       paintExtent,
		MaxPaintExtent:    estimatedTotalExtent,
		CacheExtent:       foundation.Some(cacheExtent),
		HasVisualOverflow: estimatedTotalExtent > paintExtent || constraints.ScrollOffset > 0 || constraints.Overlap != 0,
	}))

	// We may have started the layout while scrolled to the end, which would not
	// expose a new child.
	if estimatedTotalExtent == trailingScrollOffset {
		r.childManager.SetDidUnderflow(true)
	}
	r.childManager.DidFinishLayout()
}
