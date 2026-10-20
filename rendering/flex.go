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
	"github.com/rm4n0s/glutter/ui"
)

// enumName is the Dart name of an enum value whose zero value is "unspecified"
// and whose other values, in Flutter's declaration order, are 1..len(names).
func enumName(v int, names ...string) string {
	if v >= 1 && v <= len(names) {
		return names[v-1]
	}
	return "unspecified"
}

// axisSize is the size of a box along (main) and across (cross) the main axis
// of a flex layout.
//
// Flutter: rendering/flex.dart _AxisSize
type axisSize struct{ main, cross float64 }

func axisSizeFromSize(size ui.Size, direction painting.Axis) axisSize {
	if direction == painting.AxisVertical {
		size = size.Flipped()
	}
	return axisSize{size.Width, size.Height}
}

func (a axisSize) toSize(direction painting.Axis) ui.Size {
	s := ui.Size{Width: a.main, Height: a.cross}
	if direction == painting.AxisVertical {
		s = s.Flipped()
	}
	return s
}

func (a axisSize) applyConstraints(constraints BoxConstraints, direction painting.Axis) axisSize {
	if direction == painting.AxisVertical {
		constraints = constraints.Flipped()
	}
	s := constraints.Constrain(ui.Size{Width: a.main, Height: a.cross})
	return axisSize{s.Width, s.Height}
}

func (a axisSize) add(o axisSize) axisSize {
	return axisSize{a.main + o.main, math.Max(a.cross, o.cross)}
}

func (a axisSize) flipped() axisSize { return axisSize{a.cross, a.main} }

func (a axisSize) sub(o axisSize) axisSize { return axisSize{a.main - o.main, a.cross - o.cross} }

// ascentDescent is the largest ascent and descent among the children that have
// a baseline, or none.
//
// Flutter: rendering/flex.dart _AscentDescent
type ascentDescent struct {
	valid           bool
	ascent, descent float64
}

func newAscentDescent(baselineOffset foundation.Opt[float64], crossSize float64) ascentDescent {
	b, ok := baselineOffset.Get()
	if !ok {
		return ascentDescent{}
	}
	return ascentDescent{true, b, crossSize - b}
}

func (a ascentDescent) add(o ascentDescent) ascentDescent {
	switch {
	case !a.valid:
		return o
	case !o.valid:
		return a
	}
	return ascentDescent{true, math.Max(a.ascent, o.ascent), math.Max(a.descent, o.descent)}
}

func (a ascentDescent) baselineOffset() foundation.Opt[float64] {
	if !a.valid {
		return foundation.None[float64]()
	}
	return foundation.Some(a.ascent)
}

// flexLayoutSizes is the result of the sizing pass of a flex layout.
//
// Flutter: rendering/flex.dart _LayoutSizes
type flexLayoutSizes struct {
	axisSize          axisSize
	mainAxisFreeSpace float64
	baselineOffset    foundation.Opt[float64]
	spacePerFlex      foundation.Opt[float64]
}

// FlexFit is how a flexible child is inscribed into the available space. The
// zero value is FlexFitUnspecified: tight.
//
// Flutter: rendering/flex.dart FlexFit
type FlexFit int

// The values of FlexFit follow Flutter's declaration order.
const (
	// FlexFitUnspecified is the zero value: the default (tight).
	//
	// Flutter: no counterpart
	FlexFitUnspecified FlexFit = iota
	// FlexFitTight forces the child to take all the available space.
	//
	// Flutter: rendering/flex.dart FlexFit.tight
	FlexFitTight
	// FlexFitLoose lets the child be at most as large as the available space.
	//
	// Flutter: rendering/flex.dart FlexFit.loose
	FlexFitLoose
)

// Name returns the Dart enum name, such as "tight".
//
// Flutter: rendering/flex.dart FlexFit.name
func (f FlexFit) Name() string { return enumName(int(f), "tight", "loose") }

// String returns the name Dart prints, such as "FlexFit.tight".
//
// Flutter: rendering/flex.dart FlexFit.toString
func (f FlexFit) String() string { return "FlexFit." + f.Name() }

func (f FlexFit) orTight() FlexFit {
	if f == FlexFitUnspecified {
		return FlexFitTight
	}
	return f
}

// FlexParentData is the parent data of the children of a RenderFlex: the flex
// factor and the fit, besides the offset.
//
// Flutter: rendering/flex.dart FlexParentData
type FlexParentData struct {
	ContainerBoxParentData[Box]
	// Flex is the flex factor to use for this child. If None or zero, the
	// child is inflexible and determines its own size in the main axis.
	//
	// Flutter: rendering/flex.dart FlexParentData.flex
	Flex foundation.Opt[int]
	// Fit is how the child is inscribed into the available space. The
	// zero value (FlexFitUnspecified) means tight.
	//
	// Flutter: rendering/flex.dart FlexParentData.fit
	Fit FlexFit
}

// String describes the offset, the flex factor and the fit.
//
// Flutter: rendering/flex.dart FlexParentData.toString
func (d *FlexParentData) String() string {
	flex, fit := "null", "null"
	if f, ok := d.Flex.Get(); ok {
		flex = itoa(f)
	}
	if d.Fit != FlexFitUnspecified {
		fit = d.Fit.String()
	}
	return d.BoxParentData.String() + "; flex=" + flex + "; fit=" + fit
}

// MainAxisSize is how much space is occupied along the main axis. The zero
// value is MainAxisSizeUnspecified: max.
//
// Flutter: rendering/flex.dart MainAxisSize
type MainAxisSize int

// The values of MainAxisSize follow Flutter's declaration order.
const (
	// MainAxisSizeUnspecified is the zero value: the default (max).
	//
	// Flutter: no counterpart
	MainAxisSizeUnspecified MainAxisSize = iota
	// MainAxisSizeMin minimizes the amount of free space along the main axis.
	//
	// Flutter: rendering/flex.dart MainAxisSize.min
	MainAxisSizeMin
	// MainAxisSizeMax maximizes the amount of free space along the main axis.
	//
	// Flutter: rendering/flex.dart MainAxisSize.max
	MainAxisSizeMax
)

// Name returns the Dart enum name, such as "min".
//
// Flutter: rendering/flex.dart MainAxisSize.name
func (m MainAxisSize) Name() string { return enumName(int(m), "min", "max") }

// String returns the name Dart prints, such as "MainAxisSize.min".
//
// Flutter: rendering/flex.dart MainAxisSize.toString
func (m MainAxisSize) String() string { return "MainAxisSize." + m.Name() }

func (m MainAxisSize) orMax() MainAxisSize {
	if m == MainAxisSizeUnspecified {
		return MainAxisSizeMax
	}
	return m
}

// MainAxisAlignment is how the children are placed along the main axis. The
// zero value is MainAxisAlignmentUnspecified: start.
//
// Flutter: rendering/flex.dart MainAxisAlignment
type MainAxisAlignment int

// The values of MainAxisAlignment follow Flutter's declaration order.
const (
	// MainAxisAlignmentUnspecified is the zero value: the default (start).
	//
	// Flutter: no counterpart
	MainAxisAlignmentUnspecified MainAxisAlignment = iota
	// MainAxisAlignmentStart places the children as close to the start of the
	// main axis as possible.
	//
	// Flutter: rendering/flex.dart MainAxisAlignment.start
	MainAxisAlignmentStart
	// MainAxisAlignmentEnd places the children as close to the end of the main
	// axis as possible.
	//
	// Flutter: rendering/flex.dart MainAxisAlignment.end
	MainAxisAlignmentEnd
	// MainAxisAlignmentCenter places the children as close to the middle of the
	// main axis as possible.
	//
	// Flutter: rendering/flex.dart MainAxisAlignment.center
	MainAxisAlignmentCenter
	// MainAxisAlignmentSpaceBetween places the free space evenly between the
	// children.
	//
	// Flutter: rendering/flex.dart MainAxisAlignment.spaceBetween
	MainAxisAlignmentSpaceBetween
	// MainAxisAlignmentSpaceAround places the free space evenly between the
	// children as well as half of that space before and after the first and
	// last child.
	//
	// Flutter: rendering/flex.dart MainAxisAlignment.spaceAround
	MainAxisAlignmentSpaceAround
	// MainAxisAlignmentSpaceEvenly places the free space evenly between the
	// children as well as before and after the first and last child.
	//
	// Flutter: rendering/flex.dart MainAxisAlignment.spaceEvenly
	MainAxisAlignmentSpaceEvenly
)

// Name returns the Dart enum name, such as "spaceBetween".
//
// Flutter: rendering/flex.dart MainAxisAlignment.name
func (m MainAxisAlignment) Name() string {
	return enumName(int(m), "start", "end", "center", "spaceBetween", "spaceAround", "spaceEvenly")
}

// String returns the name Dart prints, such as "MainAxisAlignment.start".
//
// Flutter: rendering/flex.dart MainAxisAlignment.toString
func (m MainAxisAlignment) String() string { return "MainAxisAlignment." + m.Name() }

func (m MainAxisAlignment) orStart() MainAxisAlignment {
	if m == MainAxisAlignmentUnspecified {
		return MainAxisAlignmentStart
	}
	return m
}

// distributeSpace returns the space before the first child and between the
// children.
//
// Flutter: rendering/flex.dart MainAxisAlignment._distributeSpace
func (m MainAxisAlignment) distributeSpace(freeSpace float64, itemCount int, flipped bool, spacing float64) (leading, between float64) {
	switch m {
	case MainAxisAlignmentEnd:
		return MainAxisAlignmentStart.distributeSpace(freeSpace, itemCount, !flipped, spacing)
	case MainAxisAlignmentSpaceBetween:
		if itemCount < 2 {
			return MainAxisAlignmentStart.distributeSpace(freeSpace, itemCount, flipped, spacing)
		}
		return 0, freeSpace/float64(itemCount-1) + spacing
	case MainAxisAlignmentSpaceAround:
		if itemCount == 0 {
			return MainAxisAlignmentStart.distributeSpace(freeSpace, itemCount, flipped, spacing)
		}
		return freeSpace / float64(itemCount) / 2, freeSpace/float64(itemCount) + spacing
	case MainAxisAlignmentCenter:
		return freeSpace / 2, spacing
	case MainAxisAlignmentSpaceEvenly:
		return freeSpace / float64(itemCount+1), freeSpace/float64(itemCount+1) + spacing
	}
	if flipped {
		return freeSpace, spacing
	}
	return 0, spacing
}

// CrossAxisAlignment is how the children are placed along the cross axis. The
// zero value is CrossAxisAlignmentUnspecified: center.
//
// Flutter: rendering/flex.dart CrossAxisAlignment
type CrossAxisAlignment int

// The values of CrossAxisAlignment follow Flutter's declaration order.
const (
	// CrossAxisAlignmentUnspecified is the zero value: the default (center).
	//
	// Flutter: no counterpart
	CrossAxisAlignmentUnspecified CrossAxisAlignment = iota
	// CrossAxisAlignmentStart places the children with their start edge aligned
	// with the start side of the cross axis.
	//
	// Flutter: rendering/flex.dart CrossAxisAlignment.start
	CrossAxisAlignmentStart
	// CrossAxisAlignmentEnd places the children as close to the end of the cross
	// axis as possible.
	//
	// Flutter: rendering/flex.dart CrossAxisAlignment.end
	CrossAxisAlignmentEnd
	// CrossAxisAlignmentCenter places the children so that their centers align
	// with the middle of the cross axis.
	//
	// Flutter: rendering/flex.dart CrossAxisAlignment.center
	CrossAxisAlignmentCenter
	// CrossAxisAlignmentStretch requires the children to fill the cross axis.
	//
	// Flutter: rendering/flex.dart CrossAxisAlignment.stretch
	CrossAxisAlignmentStretch
	// CrossAxisAlignmentBaseline places the children along the cross axis such
	// that their baselines match.
	//
	// Flutter: rendering/flex.dart CrossAxisAlignment.baseline
	CrossAxisAlignmentBaseline
)

// Name returns the Dart enum name, such as "stretch".
//
// Flutter: rendering/flex.dart CrossAxisAlignment.name
func (c CrossAxisAlignment) Name() string {
	return enumName(int(c), "start", "end", "center", "stretch", "baseline")
}

// String returns the name Dart prints, such as "CrossAxisAlignment.start".
//
// Flutter: rendering/flex.dart CrossAxisAlignment.toString
func (c CrossAxisAlignment) String() string { return "CrossAxisAlignment." + c.Name() }

// orCenter is the alignment, or center for the zero value.
func (c CrossAxisAlignment) orCenter() CrossAxisAlignment {
	if c == CrossAxisAlignmentUnspecified {
		return CrossAxisAlignmentCenter
	}
	return c
}

// getChildCrossAxisOffset is the offset of a child along the cross axis.
//
// Flutter: rendering/flex.dart CrossAxisAlignment._getChildCrossAxisOffset
func (c CrossAxisAlignment) getChildCrossAxisOffset(freeSpace float64, flipped bool) float64 {
	switch c.orCenter() {
	case CrossAxisAlignmentStretch, CrossAxisAlignmentBaseline:
		return 0
	case CrossAxisAlignmentCenter:
		return freeSpace / 2
	case CrossAxisAlignmentEnd:
		return CrossAxisAlignmentStart.getChildCrossAxisOffset(freeSpace, !flipped)
	}
	if flipped {
		return freeSpace
	}
	return 0
}

// RenderFlex displays its children in a one-dimensional array.
//
// Flutter: rendering/flex.dart RenderFlex
type RenderFlex struct {
	RenderBox
	ContainerRenderObject[Box]
	DebugOverflowIndicator
	direction          painting.Axis
	mainAxisSize       MainAxisSize
	mainAxisAlignment  MainAxisAlignment
	crossAxisAlignment CrossAxisAlignment
	textDirection      ui.TextDirection
	verticalDirection  painting.VerticalDirection
	textBaseline       ui.TextBaseline
	clipBehavior       ui.Clip
	spacing            float64
	overflow           float64
	clipRectLayer      *LayerHandle[*ClipRectLayer]
}

// RenderFlexOpts are the optional parameters of NewRenderFlex.
//
// Flutter: rendering/flex.dart RenderFlex.new
type RenderFlexOpts struct {
	// Direction defaults to painting.AxisHorizontal.
	Direction painting.Axis
	// MainAxisSize defaults to MainAxisSizeMax.
	MainAxisSize MainAxisSize
	// MainAxisAlignment defaults to MainAxisAlignmentStart.
	MainAxisAlignment MainAxisAlignment
	// CrossAxisAlignment defaults to CrossAxisAlignmentCenter.
	CrossAxisAlignment CrossAxisAlignment
	// TextDirection is the order of a horizontal flex; unspecified is null.
	TextDirection ui.TextDirection
	// VerticalDirection defaults to painting.VerticalDirectionDown.
	VerticalDirection painting.VerticalDirection
	// TextBaseline is the baseline to align with; unspecified is null.
	TextBaseline ui.TextBaseline
	// ClipBehavior defaults to ui.ClipNone.
	ClipBehavior ui.Clip
	// Spacing is the space between the children; it must not be negative.
	Spacing float64
}

// NewRenderFlex creates a flex with the children (nil for none).
//
// Flutter: rendering/flex.dart RenderFlex.new
func NewRenderFlex(children []Box, opts ...RenderFlexOpts) *RenderFlex {
	o := firstOpts(opts)
	if o.Direction == painting.AxisUnspecified {
		o.Direction = painting.AxisHorizontal
	}
	if o.VerticalDirection == painting.VerticalDirectionUnspecified {
		o.VerticalDirection = painting.VerticalDirectionDown
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	if o.Spacing < 0 {
		panic(errors.New("FlexSpacingNegative", "the spacing of a RenderFlex must not be negative", "spacing", o.Spacing))
	}
	r := &RenderFlex{
		direction: o.Direction, mainAxisSize: o.MainAxisSize.orMax(), mainAxisAlignment: o.MainAxisAlignment.orStart(),
		crossAxisAlignment: o.CrossAxisAlignment.orCenter(), textDirection: o.TextDirection,
		verticalDirection: o.VerticalDirection, textBaseline: o.TextBaseline, clipBehavior: o.ClipBehavior,
		spacing: o.Spacing, clipRectLayer: NewLayerHandle[*ClipRectLayer](nil),
	}
	InitRenderBox(&r.RenderBox, r)
	InitContainerRenderObject(&r.ContainerRenderObject, r)
	r.AddAll(children)
	return r
}

// Direction is the direction to use as the main axis.
//
// Flutter: rendering/flex.dart RenderFlex.direction
func (r *RenderFlex) Direction() painting.Axis { return r.direction }

// SetDirection changes the main axis and marks the flex as needing layout.
//
// Flutter: rendering/flex.dart RenderFlex.direction
func (r *RenderFlex) SetDirection(value painting.Axis) {
	if r.direction != value {
		r.direction = value
		r.MarkNeedsLayout()
	}
}

// MainAxisAlignment is how the children are placed along the main axis.
//
// Flutter: rendering/flex.dart RenderFlex.mainAxisAlignment
func (r *RenderFlex) MainAxisAlignment() MainAxisAlignment { return r.mainAxisAlignment }

// SetMainAxisAlignment marks the flex as needing layout when the value changes.
//
// Flutter: rendering/flex.dart RenderFlex.mainAxisAlignment
func (r *RenderFlex) SetMainAxisAlignment(value MainAxisAlignment) {
	value = value.orStart()
	if r.mainAxisAlignment != value {
		r.mainAxisAlignment = value
		r.MarkNeedsLayout()
	}
}

// MainAxisSize is how much space the flex takes along the main axis.
//
// Flutter: rendering/flex.dart RenderFlex.mainAxisSize
func (r *RenderFlex) MainAxisSize() MainAxisSize { return r.mainAxisSize }

// SetMainAxisSize marks the flex as needing layout when the value changes.
//
// Flutter: rendering/flex.dart RenderFlex.mainAxisSize
func (r *RenderFlex) SetMainAxisSize(value MainAxisSize) {
	value = value.orMax()
	if r.mainAxisSize != value {
		r.mainAxisSize = value
		r.MarkNeedsLayout()
	}
}

// CrossAxisAlignment is how the children are placed along the cross axis.
//
// Flutter: rendering/flex.dart RenderFlex.crossAxisAlignment
func (r *RenderFlex) CrossAxisAlignment() CrossAxisAlignment { return r.crossAxisAlignment }

// SetCrossAxisAlignment marks the flex as needing layout when the value changes.
//
// Flutter: rendering/flex.dart RenderFlex.crossAxisAlignment
func (r *RenderFlex) SetCrossAxisAlignment(value CrossAxisAlignment) {
	value = value.orCenter()
	if r.crossAxisAlignment != value {
		r.crossAxisAlignment = value
		r.MarkNeedsLayout()
	}
}

// TextDirection determines the order to lay children out horizontally and how
// to interpret start and end in the horizontal direction. Unspecified is null.
//
// Flutter: rendering/flex.dart RenderFlex.textDirection
func (r *RenderFlex) TextDirection() ui.TextDirection { return r.textDirection }

// SetTextDirection marks the flex as needing layout when the value changes.
//
// Flutter: rendering/flex.dart RenderFlex.textDirection
func (r *RenderFlex) SetTextDirection(value ui.TextDirection) {
	if r.textDirection != value {
		r.textDirection = value
		r.MarkNeedsLayout()
	}
}

// VerticalDirection determines the order to lay children out vertically and how
// to interpret start and end in the vertical direction.
//
// Flutter: rendering/flex.dart RenderFlex.verticalDirection
func (r *RenderFlex) VerticalDirection() painting.VerticalDirection { return r.verticalDirection }

// SetVerticalDirection marks the flex as needing layout when the value changes.
//
// Flutter: rendering/flex.dart RenderFlex.verticalDirection
func (r *RenderFlex) SetVerticalDirection(value painting.VerticalDirection) {
	if value == painting.VerticalDirectionUnspecified {
		value = painting.VerticalDirectionDown
	}
	if r.verticalDirection != value {
		r.verticalDirection = value
		r.MarkNeedsLayout()
	}
}

// TextBaseline is the baseline to use when aligning children with
// CrossAxisAlignmentBaseline. Unspecified is null.
//
// Flutter: rendering/flex.dart RenderFlex.textBaseline
func (r *RenderFlex) TextBaseline() ui.TextBaseline { return r.textBaseline }

// SetTextBaseline marks the flex as needing layout when the value changes. In
// debug builds it panics with FlexBaselineMissing if the cross axis alignment
// is baseline and the value is unspecified.
//
// Flutter: rendering/flex.dart RenderFlex.textBaseline
func (r *RenderFlex) SetTextBaseline(value ui.TextBaseline) {
	if foundation.DebugMode && r.crossAxisAlignment == CrossAxisAlignmentBaseline && value == ui.TextBaselineUnspecified {
		panic(errors.New("FlexBaselineMissing", "the text baseline of a RenderFlex aligned to the baseline must be set",
			"object", foundation.DescribeIdentity(r)))
	}
	if r.textBaseline != value {
		r.textBaseline = value
		r.MarkNeedsLayout()
	}
}

// ClipBehavior is how the content is clipped when it overflows.
//
// Flutter: rendering/flex.dart RenderFlex.clipBehavior
func (r *RenderFlex) ClipBehavior() ui.Clip { return r.clipBehavior }

// SetClipBehavior marks the flex as needing paint when the value changes.
//
// Flutter: rendering/flex.dart RenderFlex.clipBehavior
func (r *RenderFlex) SetClipBehavior(value ui.Clip) {
	if value == ui.ClipUnspecified {
		value = ui.ClipNone
	}
	if value != r.clipBehavior {
		r.clipBehavior = value
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

// Spacing is how much space to place between children in the main axis.
//
// Flutter: rendering/flex.dart RenderFlex.spacing
func (r *RenderFlex) Spacing() float64 { return r.spacing }

// SetSpacing marks the flex as needing layout when the value changes.
//
// Flutter: rendering/flex.dart RenderFlex.spacing
func (r *RenderFlex) SetSpacing(value float64) {
	if r.spacing == value {
		return
	}
	r.spacing = value
	r.MarkNeedsLayout()
}

func (r *RenderFlex) hasOverflow() bool { return r.overflow > foundation.PrecisionErrorTolerance }

// SetupParentData gives the child a FlexParentData if it has none.
//
// Flutter: rendering/flex.dart RenderFlex.setupParentData
func (r *RenderFlex) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*FlexParentData); !ok {
		child.AsRenderObject().SetParentData(&FlexParentData{})
	}
}

func flexParentData(child Box) *FlexParentData { return child.ParentData().(*FlexParentData) }

func flexOf(child Box) int { return flexParentData(child).Flex.OrElse(0) }

func fitOf(child Box) FlexFit { return flexParentData(child).Fit.orTight() }

func (r *RenderFlex) isBaselineAligned() bool {
	return r.crossAxisAlignment == CrossAxisAlignmentBaseline && r.direction == painting.AxisHorizontal
}

func (r *RenderFlex) crossSize(size ui.Size) float64 {
	if r.direction == painting.AxisHorizontal {
		return size.Height
	}
	return size.Width
}

func (r *RenderFlex) mainSize(size ui.Size) float64 {
	if r.direction == painting.AxisHorizontal {
		return size.Width
	}
	return size.Height
}

func (r *RenderFlex) getIntrinsicSize(sizingDirection painting.Axis, extent float64, childSize func(child Box, extent float64) float64) float64 {
	if r.direction == sizingDirection {
		totalFlex := 0.0
		inflexibleSpace := r.spacing * float64(r.ChildCount()-1)
		maxFlexFractionSoFar := 0.0
		for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
			flex := flexOf(child)
			totalFlex += float64(flex)
			if flex > 0 {
				maxFlexFractionSoFar = math.Max(maxFlexFractionSoFar, childSize(child, extent)/float64(flex))
			} else {
				inflexibleSpace += childSize(child, extent)
			}
		}
		return maxFlexFractionSoFar*totalFlex + inflexibleSpace
	}
	isHorizontal := r.direction == painting.AxisHorizontal
	layoutChild := func(child Box, constraints BoxConstraints) ui.Size {
		mainFromConstraints := constraints.MaxHeight
		if isHorizontal {
			mainFromConstraints = constraints.MaxWidth
		}
		maxMain := mainFromConstraints
		if math.IsInf(mainFromConstraints, 0) {
			if isHorizontal {
				maxMain = child.GetMaxIntrinsicWidth(math.Inf(1))
			} else {
				maxMain = child.GetMaxIntrinsicHeight(math.Inf(1))
			}
		}
		if isHorizontal {
			return ui.Size{Width: maxMain, Height: childSize(child, maxMain)}
		}
		return ui.Size{Width: childSize(child, maxMain), Height: maxMain}
	}
	inf := math.Inf(1)
	constraints := BoxConstraints{MaxWidth: inf, MaxHeight: extent}
	if isHorizontal {
		constraints = BoxConstraints{MaxWidth: extent, MaxHeight: inf}
	}
	return r.computeSizes(constraints, layoutChild, ChildLayoutHelper.GetDryBaseline).axisSize.cross
}

// ComputeMinIntrinsicWidth is the minimum width the children need.
//
// Flutter: rendering/flex.dart RenderFlex.computeMinIntrinsicWidth
func (r *RenderFlex) ComputeMinIntrinsicWidth(height float64) float64 {
	return r.getIntrinsicSize(painting.AxisHorizontal, height, func(c Box, e float64) float64 { return c.GetMinIntrinsicWidth(e) })
}

// ComputeMaxIntrinsicWidth is the width the children want.
//
// Flutter: rendering/flex.dart RenderFlex.computeMaxIntrinsicWidth
func (r *RenderFlex) ComputeMaxIntrinsicWidth(height float64) float64 {
	return r.getIntrinsicSize(painting.AxisHorizontal, height, func(c Box, e float64) float64 { return c.GetMaxIntrinsicWidth(e) })
}

// ComputeMinIntrinsicHeight is the minimum height the children need.
//
// Flutter: rendering/flex.dart RenderFlex.computeMinIntrinsicHeight
func (r *RenderFlex) ComputeMinIntrinsicHeight(width float64) float64 {
	return r.getIntrinsicSize(painting.AxisVertical, width, func(c Box, e float64) float64 { return c.GetMinIntrinsicHeight(e) })
}

// ComputeMaxIntrinsicHeight is the height the children want.
//
// Flutter: rendering/flex.dart RenderFlex.computeMaxIntrinsicHeight
func (r *RenderFlex) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.getIntrinsicSize(painting.AxisVertical, width, func(c Box, e float64) float64 { return c.GetMaxIntrinsicHeight(e) })
}

// ComputeDistanceToActualBaseline is the highest baseline of a row and the first
// baseline of a column.
//
// Flutter: rendering/flex.dart RenderFlex.computeDistanceToActualBaseline
func (r *RenderFlex) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	if r.direction == painting.AxisHorizontal {
		return DefaultComputeDistanceToHighestActualBaseline(&r.ContainerRenderObject, baseline)
	}
	return DefaultComputeDistanceToFirstActualBaseline(&r.ContainerRenderObject, baseline)
}

func (r *RenderFlex) flipMainAxis() bool {
	if isNil(r.FirstChild()) {
		return false
	}
	if r.direction == painting.AxisHorizontal {
		return r.textDirection == ui.TextDirectionRTL
	}
	return r.verticalDirection == painting.VerticalDirectionUp
}

func (r *RenderFlex) flipCrossAxis() bool {
	if isNil(r.FirstChild()) {
		return false
	}
	if r.direction == painting.AxisVertical {
		return r.textDirection == ui.TextDirectionRTL
	}
	return r.verticalDirection == painting.VerticalDirectionUp
}

func (r *RenderFlex) constraintsForNonFlexChild(constraints BoxConstraints) BoxConstraints {
	fillCrossAxis := r.crossAxisAlignment == CrossAxisAlignmentStretch
	inf := math.Inf(1)
	if r.direction == painting.AxisHorizontal {
		if fillCrossAxis {
			return BoxConstraints{MaxWidth: inf, MinHeight: constraints.MaxHeight, MaxHeight: constraints.MaxHeight}
		}
		return BoxConstraints{MaxWidth: inf, MaxHeight: constraints.MaxHeight}
	}
	if fillCrossAxis {
		return BoxConstraints{MinWidth: constraints.MaxWidth, MaxWidth: constraints.MaxWidth, MaxHeight: inf}
	}
	return BoxConstraints{MaxWidth: constraints.MaxWidth, MaxHeight: inf}
}

func (r *RenderFlex) constraintsForFlexChild(child Box, constraints BoxConstraints, maxChildExtent float64) BoxConstraints {
	minChildExtent := 0.0
	if fitOf(child) == FlexFitTight {
		minChildExtent = maxChildExtent
	}
	fillCrossAxis := r.crossAxisAlignment == CrossAxisAlignmentStretch
	if r.direction == painting.AxisHorizontal {
		minHeight := 0.0
		if fillCrossAxis {
			minHeight = constraints.MaxHeight
		}
		return BoxConstraints{MinWidth: minChildExtent, MaxWidth: maxChildExtent, MinHeight: minHeight, MaxHeight: constraints.MaxHeight}
	}
	minWidth := 0.0
	if fillCrossAxis {
		minWidth = constraints.MaxWidth
	}
	return BoxConstraints{MinWidth: minWidth, MaxWidth: constraints.MaxWidth, MinHeight: minChildExtent, MaxHeight: maxChildExtent}
}

// ComputeDryBaseline is the baseline of the layout the constraints would give.
//
// Flutter: rendering/flex.dart RenderFlex.computeDryBaseline
func (r *RenderFlex) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	sizes := r.computeSizes(constraints, ChildLayoutHelper.DryLayoutChild, ChildLayoutHelper.GetDryBaseline)
	if r.isBaselineAligned() {
		return sizes.baselineOffset
	}
	if r.direction == painting.AxisHorizontal {
		return r.computeDryDistanceToHighestBaseline(constraints, baseline, sizes)
	}
	return r.computeDryDistanceToFirstBaseline(constraints, baseline, sizes)
}

func (r *RenderFlex) dryConstraintsForChild(constraints BoxConstraints, sizes flexLayoutSizes) func(child Box) BoxConstraints {
	nonFlex := r.constraintsForNonFlexChild(constraints)
	return func(child Box) BoxConstraints {
		if spacePerFlex, ok := sizes.spacePerFlex.Get(); ok {
			if flex := flexOf(child); flex > 0 {
				return r.constraintsForFlexChild(child, constraints, float64(flex)*spacePerFlex)
			}
		}
		return nonFlex
	}
}

func (r *RenderFlex) computeDryDistanceToHighestBaseline(constraints BoxConstraints, baseline ui.TextBaseline, sizes flexLayoutSizes) foundation.Opt[float64] {
	constraintsForChild := r.dryConstraintsForChild(constraints, sizes)
	flipCrossAxis := r.flipCrossAxis()
	next, start := r.ChildAfter, r.FirstChild()
	if r.flipMainAxis() {
		next, start = r.ChildBefore, r.LastChild()
	}
	alignedBaseline := foundation.None[float64]()
	if r.isBaselineAligned() && r.textBaseline != ui.TextBaselineUnspecified {
		alignedBaseline = sizes.baselineOffset
	}
	minBaseline := NoBaseline
	for child := start; !isNil(child); child = next(child) {
		childConstraints := constraintsForChild(child)
		childBaseline, ok := child.GetDryBaseline(childConstraints, baseline).Get()
		if !ok {
			continue
		}
		childBaselineOffset := foundation.None[float64]()
		if r.isBaselineAligned() && r.textBaseline != ui.TextBaselineUnspecified {
			childBaselineOffset = child.GetDryBaseline(childConstraints, r.textBaseline)
		}
		var crossPosition float64
		ab, aok := alignedBaseline.Get()
		cb, cok := childBaselineOffset.Get()
		switch {
		case aok && cok:
			crossPosition = ab - cb
		case r.crossAxisAlignment == CrossAxisAlignmentBaseline && r.direction == painting.AxisHorizontal:
			childSize := child.GetDryLayout(childConstraints)
			crossPosition = CrossAxisAlignmentStart.getChildCrossAxisOffset(sizes.axisSize.cross-r.crossSize(childSize), false)
		default:
			childSize := child.GetDryLayout(childConstraints)
			crossPosition = r.crossAxisAlignment.getChildCrossAxisOffset(sizes.axisSize.cross-r.crossSize(childSize), flipCrossAxis)
		}
		minBaseline = minBaseline.MinOf(NewBaselineOffset(foundation.Some(childBaseline)).Add(crossPosition))
	}
	return minBaseline.Offset
}

func (r *RenderFlex) computeDryDistanceToFirstBaseline(constraints BoxConstraints, baseline ui.TextBaseline, sizes flexLayoutSizes) foundation.Opt[float64] {
	constraintsForChild := r.dryConstraintsForChild(constraints, sizes)
	remainingSpace := math.Max(0, sizes.mainAxisFreeSpace)
	flipMainAxis := r.flipMainAxis()
	leading, between := r.mainAxisAlignment.distributeSpace(remainingSpace, r.ChildCount(), flipMainAxis, r.spacing)
	mainPositions := map[Box]float64{}
	next, start := r.ChildAfter, r.FirstChild()
	if flipMainAxis {
		next, start = r.ChildBefore, r.LastChild()
	}
	pos := leading
	for child := start; !isNil(child); child = next(child) {
		mainPositions[child] = pos
		cs := child.GetDryLayout(constraintsForChild(child))
		pos += r.mainSize(cs) + between
	}
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		if childBaseline, ok := child.GetDryBaseline(constraintsForChild(child), baseline).Get(); ok {
			position, found := mainPositions[child]
			if !found {
				position = leading
			}
			return foundation.Some(childBaseline + position)
		}
	}
	return foundation.None[float64]()
}

// ComputeDryLayout is the size the constraints would give. A flex whose
// constraints are invalid (flexible children in an unbounded direction) can not
// compute its dry layout.
//
// Flutter: rendering/flex.dart RenderFlex.computeDryLayout
func (r *RenderFlex) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if foundation.DebugMode {
		if err := r.debugCheckConstraints(constraints, false); err != nil {
			r.DebugCannotComputeDryLayout(err.Error())
			return ui.Size{}
		}
	}
	return r.computeSizes(constraints, ChildLayoutHelper.DryLayoutChild, ChildLayoutHelper.GetDryBaseline).axisSize.toSize(r.direction)
}

// debugCheckConstraints returns the error that a layout with flexible children
// and unbounded main-axis constraints is, or nil.
//
// Flutter: rendering/flex.dart RenderFlex._debugCheckConstraints
func (r *RenderFlex) debugCheckConstraints(constraints BoxConstraints, reportParentConstraints bool) error {
	maxMainSize := constraints.MaxHeight
	identity, axis, dimension := "column", "vertical", "height"
	if r.direction == painting.AxisHorizontal {
		maxMainSize = constraints.MaxWidth
		identity, axis, dimension = "row", "horizontal", "width"
	}
	canFlex := maxMainSize < math.Inf(1)
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		if flexOf(child) <= 0 {
			continue
		}
		if canFlex || (r.mainAxisSize != MainAxisSizeMax && fitOf(child) != FlexFitTight) {
			return nil
		}
		ancestor := ""
		if reportParentConstraints {
			var node RenderObject = r
			for {
				box, ok := node.(Box)
				if !ok {
					node = nil
					break
				}
				bounded := box.BoxConstraints().HasBoundedHeight()
				if r.direction == painting.AxisHorizontal {
					bounded = box.BoxConstraints().HasBoundedWidth()
				}
				if bounded {
					break
				}
				if _, parentIsBox := box.Parent().(Box); !parentIsBox {
					node = nil
					break
				}
				node = box.Parent()
			}
			if node != nil {
				ancestor = foundation.DescribeIdentity(node)
			}
		}
		return errors.New("RenderFlexUnboundedFlex",
			"RenderFlex children have non-zero flex but incoming "+dimension+" constraints are unbounded. "+
				"When a "+identity+" is in a parent that does not provide a finite "+dimension+" constraint, for example "+
				"if it is in a "+axis+" scrollable, it will try to shrink-wrap its children along the "+axis+
				" axis. Setting a flex on a child (e.g. using Expanded) indicates that the child is to "+
				"expand to fill the remaining space in the "+axis+" direction. "+
				"These two directives are mutually exclusive. If a parent is to shrink-wrap its child, the child "+
				"cannot simultaneously expand to fit its parent. "+
				"Consider setting mainAxisSize to MainAxisSize.min and using FlexFit.loose fits for the flexible "+
				"children (using Flexible rather than Expanded). This will allow the flexible children "+
				"to size themselves to less than the infinite remaining space they would otherwise be "+
				"forced to take, and then will cause the RenderFlex to shrink-wrap the children "+
				"rather than expanding to fit the maximum constraints provided by the parent.",
			"object", foundation.DescribeIdentity(r), "constraints", constraints, "ancestor", ancestor)
	}
	return nil
}

// debugCheckDirections panics if a direction that the layout needs is missing.
//
// Flutter: rendering/flex.dart RenderFlex._debugHasNecessaryDirections
func (r *RenderFlex) debugCheckDirections() {
	if r.dbg().checkingIntrinsics {
		return
	}
	name := foundation.RuntimeTypeName(r)
	missing := func(what string) {
		panic(errors.New("FlexDirectionMissing", what, "object", foundation.DescribeIdentity(r)))
	}
	if r.textDirection == ui.TextDirectionUnspecified {
		first, last := r.FirstChild(), r.LastChild()
		if !isNil(first) && first != last && r.direction == painting.AxisHorizontal {
			missing("Horizontal " + name + " with multiple children has a null textDirection, so the layout order is undefined.")
		}
		if (r.mainAxisAlignment == MainAxisAlignmentStart || r.mainAxisAlignment == MainAxisAlignmentEnd) &&
			r.direction == painting.AxisHorizontal {
			missing("Horizontal " + name + " with " + r.mainAxisAlignment.String() + " has a null textDirection, so the alignment cannot be resolved.")
		}
		if (r.crossAxisAlignment == CrossAxisAlignmentStart || r.crossAxisAlignment == CrossAxisAlignmentEnd) &&
			r.direction == painting.AxisVertical {
			missing("Vertical " + name + " with " + r.crossAxisAlignment.String() + " has a null textDirection, so the alignment cannot be resolved.")
		}
	}
}

// computeSizes is the sizing pass of the layout, real or dry.
//
// Flutter: rendering/flex.dart RenderFlex._computeSizes
func (r *RenderFlex) computeSizes(constraints BoxConstraints, layoutChild ChildLayouter, getBaseline ChildBaselineGetter) flexLayoutSizes {
	if foundation.DebugMode {
		r.debugCheckDirections()
	}
	maxMainSize := r.mainSize(constraints.Biggest())
	canFlex := !math.IsInf(maxMainSize, 0)
	nonFlexChildConstraints := r.constraintsForNonFlexChild(constraints)
	textBaseline := ui.TextBaselineUnspecified
	if r.isBaselineAligned() {
		if r.textBaseline == ui.TextBaselineUnspecified {
			panic(errors.New("FlexBaselineMissing",
				`To use CrossAxisAlignment.baseline, you must also specify which baseline to use using the "textBaseline" argument.`,
				"object", foundation.DescribeIdentity(r)))
		}
		textBaseline = r.textBaseline
	}
	totalFlex := 0
	var firstFlexChild Box
	var accumulatedAscentDescent ascentDescent
	accumulatedSize := axisSize{r.spacing * float64(r.ChildCount()-1), 0}
	for child := r.FirstChild(); !isNil(child); child = r.ChildAfter(child) {
		if flex := flexOf(child); canFlex && flex > 0 {
			totalFlex += flex
			if isNil(firstFlexChild) {
				firstFlexChild = child
			}
			continue
		}
		childSize := axisSizeFromSize(layoutChild(child, nonFlexChildConstraints), r.direction)
		accumulatedSize = accumulatedSize.add(childSize)
		baselineOffset := foundation.None[float64]()
		if textBaseline != ui.TextBaselineUnspecified {
			baselineOffset = getBaseline(child, nonFlexChildConstraints, textBaseline)
		}
		accumulatedAscentDescent = accumulatedAscentDescent.add(newAscentDescent(baselineOffset, childSize.cross))
	}
	flexSpace := math.Max(0, maxMainSize-accumulatedSize.main)
	spacePerFlex := flexSpace / float64(totalFlex)
	for child := firstFlexChild; !isNil(child) && totalFlex > 0; child = r.ChildAfter(child) {
		flex := flexOf(child)
		if flex == 0 {
			continue
		}
		totalFlex -= flex
		childConstraints := r.constraintsForFlexChild(child, constraints, spacePerFlex*float64(flex))
		childSize := axisSizeFromSize(layoutChild(child, childConstraints), r.direction)
		accumulatedSize = accumulatedSize.add(childSize)
		baselineOffset := foundation.None[float64]()
		if textBaseline != ui.TextBaselineUnspecified {
			baselineOffset = getBaseline(child, childConstraints, textBaseline)
		}
		accumulatedAscentDescent = accumulatedAscentDescent.add(newAscentDescent(baselineOffset, childSize.cross))
	}
	if accumulatedAscentDescent.valid {
		accumulatedSize = accumulatedSize.add(axisSize{0, accumulatedAscentDescent.ascent + accumulatedAscentDescent.descent})
	}
	idealMainSize := accumulatedSize.main
	if r.mainAxisSize == MainAxisSizeMax && !math.IsInf(maxMainSize, 0) {
		idealMainSize = maxMainSize
	}
	constrained := axisSize{idealMainSize, accumulatedSize.cross}.applyConstraints(constraints, r.direction)
	sizes := flexLayoutSizes{
		axisSize:          constrained,
		mainAxisFreeSpace: constrained.main - accumulatedSize.main,
		baselineOffset:    accumulatedAscentDescent.baselineOffset(),
	}
	if !isNil(firstFlexChild) {
		sizes.spacePerFlex = foundation.Some(spacePerFlex)
	}
	return sizes
}

// PerformLayout sizes the inflexible children, divides the free space among the
// flexible ones, and positions them along the axes.
//
// Flutter: rendering/flex.dart RenderFlex.performLayout
func (r *RenderFlex) PerformLayout() {
	constraints := r.BoxConstraints()
	if foundation.DebugMode {
		if err := r.debugCheckConstraints(constraints, true); err != nil {
			panic(err)
		}
	}
	sizes := r.computeSizes(constraints, ChildLayoutHelper.LayoutChild, ChildLayoutHelper.GetBaseline)
	crossAxisExtent := sizes.axisSize.cross
	r.SetSize(sizes.axisSize.toSize(r.direction))
	r.overflow = math.Max(0, -sizes.mainAxisFreeSpace)
	remainingSpace := math.Max(0, sizes.mainAxisFreeSpace)
	flipMainAxis, flipCrossAxis := r.flipMainAxis(), r.flipCrossAxis()
	leading, between := r.mainAxisAlignment.distributeSpace(remainingSpace, r.ChildCount(), flipMainAxis, r.spacing)
	next, topLeft := r.ChildAfter, r.FirstChild()
	if flipMainAxis {
		next, topLeft = r.ChildBefore, r.LastChild()
	}
	baselineOffset, hasBaseline := sizes.baselineOffset.Get()
	childMainPosition := leading
	for child := topLeft; !isNil(child); child = next(child) {
		var childCrossPosition float64
		childBaseline, alignByBaseline := foundation.None[float64](), false
		if hasBaseline {
			childBaseline = child.GetDistanceToBaseline(r.textBaseline, BaselineOpts{OnlyReal: true})
			_, alignByBaseline = childBaseline.Get()
		}
		switch {
		case alignByBaseline:
			childCrossPosition = baselineOffset - childBaseline.OrElse(0)
		case r.crossAxisAlignment == CrossAxisAlignmentBaseline && r.direction == painting.AxisHorizontal:
			childCrossPosition = CrossAxisAlignmentStart.getChildCrossAxisOffset(crossAxisExtent-r.crossSize(child.Size()), false)
		default:
			childCrossPosition = r.crossAxisAlignment.getChildCrossAxisOffset(crossAxisExtent-r.crossSize(child.Size()), flipCrossAxis)
		}
		data := flexParentData(child)
		if r.direction == painting.AxisHorizontal {
			data.Offset = ui.Offset{Dx: childMainPosition, Dy: childCrossPosition}
		} else {
			data.Offset = ui.Offset{Dx: childCrossPosition, Dy: childMainPosition}
		}
		childMainPosition += r.mainSize(child.Size()) + between
	}
}

// HitTestChildren hit tests the children from the front.
//
// Flutter: rendering/flex.dart RenderFlex.hitTestChildren
func (r *RenderFlex) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	return DefaultHitTestChildren(&r.ContainerRenderObject, result, position)
}

func (r *RenderFlex) defaultPaint(context *PaintingContext, offset ui.Offset) {
	DefaultPaint(&r.ContainerRenderObject, context, offset)
}

// Paint paints the children, clipped to the flex if they overflow it, and marks
// the overflow in debug builds.
//
// Flutter: rendering/flex.dart RenderFlex.paint
func (r *RenderFlex) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasOverflow() {
		r.defaultPaint(context, offset)
		return
	}
	if r.Size().IsEmpty() {
		return
	}
	r.clipRectLayer.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()),
		r.defaultPaint, PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: r.clipRectLayer.Layer()}))
	if foundation.DebugMode {
		name := foundation.RuntimeTypeName(r)
		hints := []string{
			"The overflowing " + name + " has an orientation of " + r.direction.String() + ".",
			"The edge of the " + name + " that is overflowing has been marked in the rendering with a yellow and black " +
				"striped pattern. This is usually caused by the contents being too big for the " + name + ".",
			"Consider applying a flex factor (e.g. using an Expanded widget) to force the children of the " + name +
				" to fit within the available space instead of being sized to their natural size.",
			"This is considered an error condition because it indicates that there is content that cannot be seen. " +
				"If the content is legitimately bigger than the available space, consider clipping it with a ClipRect " +
				"widget before putting it in the flex, or using a scrollable container rather than a Flex, like a ListView.",
		}
		size := r.Size()
		childRect := ui.NewRectFromLTWH(0, 0, size.Width+r.overflow, 0)
		if r.direction == painting.AxisVertical {
			childRect = ui.NewRectFromLTWH(0, 0, 0, size.Height+r.overflow)
		}
		r.PaintOverflowIndicator(r, context, offset, ui.OffsetZero.And(size), childRect, hints...)
	}
}

// Dispose lets the clip layer go.
//
// Flutter: rendering/flex.dart RenderFlex.dispose
func (r *RenderFlex) Dispose() {
	r.clipRectLayer.SetLayer(nil)
	r.RenderBox.Dispose()
}

// Reassemble makes the overflow be reported again.
//
// Flutter: rendering/debug_overflow_indicator.dart DebugOverflowIndicatorMixin.reassemble
func (r *RenderFlex) Reassemble() {
	r.RenderBox.Reassemble()
	r.ResetOverflowReport()
}

// DescribeApproximatePaintClip is the flex's box when it overflows and clips.
//
// Flutter: rendering/flex.dart RenderFlex.describeApproximatePaintClip
func (r *RenderFlex) DescribeApproximatePaintClip(RenderObject) foundation.Opt[ui.Rect] {
	if r.clipBehavior != ui.ClipNone && r.hasOverflow() {
		return foundation.Some(ui.OffsetZero.And(r.Size()))
	}
	return foundation.None[ui.Rect]()
}

// ToStringShort adds OVERFLOWING when the children overflow (debug builds).
//
// Flutter: rendering/flex.dart RenderFlex.toStringShort
func (r *RenderFlex) ToStringShort() string {
	header := r.RenderBox.ToStringShort()
	if foundation.DebugMode && r.hasOverflow() {
		header += " OVERFLOWING"
	}
	return header
}

// DebugFillProperties adds the direction, the alignments and the spacing.
//
// Flutter: rendering/flex.dart RenderFlex.debugFillProperties
func (r *RenderFlex) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderBox.DebugFillProperties(b)
	b.Add(foundation.NewEnumProperty("direction", foundation.Some(r.direction)))
	b.Add(foundation.NewEnumProperty("mainAxisAlignment", foundation.Some(r.mainAxisAlignment)))
	b.Add(foundation.NewEnumProperty("mainAxisSize", foundation.Some(r.mainAxisSize)))
	b.Add(foundation.NewEnumProperty("crossAxisAlignment", foundation.Some(r.crossAxisAlignment)))
	b.Add(foundation.NewEnumProperty("textDirection", optEnum(r.textDirection, ui.TextDirectionUnspecified), foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultNull}))
	b.Add(foundation.NewEnumProperty("verticalDirection", foundation.Some(r.verticalDirection), foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultNull}))
	b.Add(foundation.NewEnumProperty("textBaseline", optEnum(r.textBaseline, ui.TextBaselineUnspecified), foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultNull}))
	b.Add(foundation.NewDoubleProperty("spacing", foundation.Some(r.spacing), foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}))
}

// optEnum is None for the unspecified value, else the value.
func optEnum[T comparable](v, unspecified T) foundation.Opt[T] {
	if v == unspecified {
		return foundation.None[T]()
	}
	return foundation.Some(v)
}
