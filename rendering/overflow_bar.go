// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// OverflowBarAlignment is how a RenderOverflowBar aligns its children when they
// overflow horizontally and are stacked vertically.
//
// Flutter: widgets/overflow_bar.dart OverflowBarAlignment
type OverflowBarAlignment int

const (
	// OverflowBarAlignmentUnspecified is the zero value: the default (start).
	//
	// Flutter: no counterpart
	OverflowBarAlignmentUnspecified OverflowBarAlignment = iota
	// OverflowBarAlignmentStart aligns each child to the start of the bar.
	//
	// Flutter: widgets/overflow_bar.dart OverflowBarAlignment.start
	OverflowBarAlignmentStart
	// OverflowBarAlignmentEnd aligns each child to the end of the bar.
	//
	// Flutter: widgets/overflow_bar.dart OverflowBarAlignment.end
	OverflowBarAlignmentEnd
	// OverflowBarAlignmentCenter centers each child.
	//
	// Flutter: widgets/overflow_bar.dart OverflowBarAlignment.center
	OverflowBarAlignmentCenter
)

// Name returns the Dart enum name.
//
// Flutter: widgets/overflow_bar.dart OverflowBarAlignment.name
func (a OverflowBarAlignment) Name() string {
	return enumName(int(a), "start", "end", "center")
}

// String returns the name Dart prints, such as "OverflowBarAlignment.end".
//
// Flutter: widgets/overflow_bar.dart OverflowBarAlignment.toString
func (a OverflowBarAlignment) String() string { return "OverflowBarAlignment." + a.Name() }

func (a OverflowBarAlignment) orStart() OverflowBarAlignment {
	if a == OverflowBarAlignmentUnspecified {
		return OverflowBarAlignmentStart
	}
	return a
}

// OverflowBarParentData is the parent data of the children of a
// RenderOverflowBar.
//
// Flutter: widgets/overflow_bar.dart _OverflowBarParentData
type OverflowBarParentData struct {
	ContainerBoxParentData[Box]
}

// RenderOverflowBar lays out its children in a row with spacing, or, when they
// do not fit, in a column.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar
type RenderOverflowBar struct {
	RenderBox
	ContainerRenderObject[Box]
	spacing           float64
	alignment         foundation.Opt[MainAxisAlignment]
	overflowSpacing   float64
	overflowAlignment OverflowBarAlignment
	overflowDirection painting.VerticalDirection
	textDirection     ui.TextDirection
}

// RenderOverflowBarOpts are the optional parameters of NewRenderOverflowBar.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.new
type RenderOverflowBarOpts struct {
	// Spacing is the space between the children in the row; the default is 0.
	Spacing float64
	// Alignment is how the children are placed in the row; None sizes the row to
	// its children and starts them at the start.
	Alignment foundation.Opt[MainAxisAlignment]
	// OverflowSpacing is the space between the children in the column; the
	// default is 0.
	OverflowSpacing float64
	// OverflowAlignment is how the children align in the column; the default is
	// start.
	OverflowAlignment OverflowBarAlignment
	// OverflowDirection is the direction of the column; the default is down.
	OverflowDirection painting.VerticalDirection
	// TextDirection resolves start and end; the default is left to right.
	TextDirection ui.TextDirection
}

// NewRenderOverflowBar creates the box with the children (nil for none).
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.new
func NewRenderOverflowBar(children []Box, opts ...RenderOverflowBarOpts) *RenderOverflowBar {
	o := firstOpts(opts)
	if o.OverflowDirection == painting.VerticalDirectionUnspecified {
		o.OverflowDirection = painting.VerticalDirectionDown
	}
	if o.TextDirection == ui.TextDirectionUnspecified {
		o.TextDirection = ui.TextDirectionLTR
	}
	r := &RenderOverflowBar{spacing: o.Spacing, alignment: o.Alignment, overflowSpacing: o.OverflowSpacing,
		overflowAlignment: o.OverflowAlignment.orStart(), overflowDirection: o.OverflowDirection, textDirection: o.TextDirection}
	InitRenderBox(&r.RenderBox, r)
	InitContainerRenderObject(&r.ContainerRenderObject, r)
	r.AddAll(children)
	return r
}

// Spacing is the space between the children in the row.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.spacing
func (r *RenderOverflowBar) Spacing() float64 { return r.spacing }

// SetSpacing changes the spacing and marks the box as needing layout.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.spacing
func (r *RenderOverflowBar) SetSpacing(v float64) {
	if r.spacing != v {
		r.spacing = v
		r.MarkNeedsLayout()
	}
}

// SetAlignment changes the alignment and marks the box as needing layout.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.alignment
func (r *RenderOverflowBar) SetAlignment(v foundation.Opt[MainAxisAlignment]) {
	if r.alignment != v {
		r.alignment = v
		r.MarkNeedsLayout()
	}
}

// SetOverflowSpacing changes the spacing of the column and marks the box as
// needing layout.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.overflowSpacing
func (r *RenderOverflowBar) SetOverflowSpacing(v float64) {
	if r.overflowSpacing != v {
		r.overflowSpacing = v
		r.MarkNeedsLayout()
	}
}

// SetOverflowAlignment changes the alignment of the column and marks the box as
// needing layout.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.overflowAlignment
func (r *RenderOverflowBar) SetOverflowAlignment(v OverflowBarAlignment) {
	v = v.orStart()
	if r.overflowAlignment != v {
		r.overflowAlignment = v
		r.MarkNeedsLayout()
	}
}

// SetOverflowDirection changes the direction of the column and marks the box as
// needing layout.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.overflowDirection
func (r *RenderOverflowBar) SetOverflowDirection(v painting.VerticalDirection) {
	if v == painting.VerticalDirectionUnspecified {
		v = painting.VerticalDirectionDown
	}
	if r.overflowDirection != v {
		r.overflowDirection = v
		r.MarkNeedsLayout()
	}
}

// SetTextDirection changes the text direction and marks the box as needing
// layout.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.textDirection
func (r *RenderOverflowBar) SetTextDirection(v ui.TextDirection) {
	if v == ui.TextDirectionUnspecified {
		v = ui.TextDirectionLTR
	}
	if r.textDirection != v {
		r.textDirection = v
		r.MarkNeedsLayout()
	}
}

// SetupParentData gives the child an OverflowBarParentData if it has none.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.setupParentData
func (r *RenderOverflowBar) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*OverflowBarParentData); !ok {
		child.AsRenderObject().SetParentData(&OverflowBarParentData{})
	}
}

func overflowBarParentData(child Box) *OverflowBarParentData {
	return child.ParentData().(*OverflowBarParentData)
}

// minChildrenWidth is the width of the row of the children at their minimum
// intrinsic widths, with the spacing.
func (r *RenderOverflowBar) minChildrenWidth() float64 {
	w := 0.0
	for c := r.FirstChild(); !isNil(c); c = r.ChildAfter(c) {
		w += c.GetMinIntrinsicWidth(math.Inf(1))
	}
	return w + r.spacing*float64(r.ChildCount()-1)
}

// ComputeMinIntrinsicHeight is the height of the column if the children would
// overflow, else the tallest child's.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.computeMinIntrinsicHeight
func (r *RenderOverflowBar) ComputeMinIntrinsicHeight(width float64) float64 {
	if isNil(r.FirstChild()) {
		return 0
	}
	height := 0.0
	if r.minChildrenWidth() > width {
		for c := r.FirstChild(); !isNil(c); c = r.ChildAfter(c) {
			height += c.GetMinIntrinsicHeight(width)
		}
		return height + r.overflowSpacing*float64(r.ChildCount()-1)
	}
	for c := r.FirstChild(); !isNil(c); c = r.ChildAfter(c) {
		height = math.Max(height, c.GetMinIntrinsicHeight(width))
	}
	return height
}

// ComputeMaxIntrinsicHeight is the same with the maximum intrinsic heights.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.computeMaxIntrinsicHeight
func (r *RenderOverflowBar) ComputeMaxIntrinsicHeight(width float64) float64 {
	if isNil(r.FirstChild()) {
		return 0
	}
	height := 0.0
	if r.minChildrenWidth() > width {
		for c := r.FirstChild(); !isNil(c); c = r.ChildAfter(c) {
			height += c.GetMaxIntrinsicHeight(width)
		}
		return height + r.overflowSpacing*float64(r.ChildCount()-1)
	}
	for c := r.FirstChild(); !isNil(c); c = r.ChildAfter(c) {
		height = math.Max(height, c.GetMaxIntrinsicHeight(width))
	}
	return height
}

// ComputeMinIntrinsicWidth is the width of the row of the minimum widths.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.computeMinIntrinsicWidth
func (r *RenderOverflowBar) ComputeMinIntrinsicWidth(float64) float64 {
	if isNil(r.FirstChild()) {
		return 0
	}
	width := 0.0
	for c := r.FirstChild(); !isNil(c); c = r.ChildAfter(c) {
		width += c.GetMinIntrinsicWidth(math.Inf(1))
	}
	return width + r.spacing*float64(r.ChildCount()-1)
}

// ComputeMaxIntrinsicWidth is the width of the row of the maximum widths.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.computeMaxIntrinsicWidth
func (r *RenderOverflowBar) ComputeMaxIntrinsicWidth(float64) float64 {
	if isNil(r.FirstChild()) {
		return 0
	}
	width := 0.0
	for c := r.FirstChild(); !isNil(c); c = r.ChildAfter(c) {
		width += c.GetMaxIntrinsicWidth(math.Inf(1))
	}
	return width + r.spacing*float64(r.ChildCount()-1)
}

// ComputeDistanceToActualBaseline is the highest baseline of the children.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.computeDistanceToActualBaseline
func (r *RenderOverflowBar) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	return DefaultComputeDistanceToHighestActualBaseline(&r.ContainerRenderObject, baseline)
}

// ComputeDryBaseline is the baseline for the constraints.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.computeDryBaseline
func (r *RenderOverflowBar) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	childConstraints := constraints.Loosen()
	next, start := r.ChildAfter, r.FirstChild()
	if r.overflowDirection == painting.VerticalDirectionUp {
		next, start = r.ChildBefore, r.LastChild()
	}
	maxChildHeight, y, childrenWidth := 0.0, 0.0, 0.0
	minHorizontalBaseline, verticalBaseline := NoBaseline, NoBaseline
	for child := start; !isNil(child); child = next(child) {
		childSize := child.GetDryLayout(childConstraints)
		if heightDiff := childSize.Height - maxChildHeight; heightDiff > 0 {
			minHorizontalBaseline = minHorizontalBaseline.Add(heightDiff / 2)
			maxChildHeight = childSize.Height
		}
		offset := NewBaselineOffset(child.GetDryBaseline(childConstraints, baseline))
		if offset.Offset.IsSome() {
			if verticalBaseline.Offset.IsNone() {
				verticalBaseline = offset.Add(y)
			}
			minHorizontalBaseline = minHorizontalBaseline.MinOf(offset.Add(maxChildHeight - childSize.Height))
		}
		y += childSize.Height + r.overflowSpacing
		childrenWidth += childSize.Width
	}
	if childrenWidth+r.spacing*float64(r.ChildCount()-1) > constraints.MaxWidth {
		return verticalBaseline.Offset
	}
	return minHorizontalBaseline.Offset
}

// ComputeDryLayout is the size for the constraints.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.computeDryLayout
func (r *RenderOverflowBar) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if isNil(r.FirstChild()) {
		return constraints.Smallest()
	}
	childConstraints := constraints.Loosen()
	childrenWidth, maxChildHeight, y := 0.0, 0.0, 0.0
	for c := r.FirstChild(); !isNil(c); c = r.ChildAfter(c) {
		s := c.GetDryLayout(childConstraints)
		childrenWidth += s.Width
		maxChildHeight = math.Max(maxChildHeight, s.Height)
		y += s.Height + r.overflowSpacing
	}
	actualWidth := childrenWidth + r.spacing*float64(r.ChildCount()-1)
	if actualWidth > constraints.MaxWidth {
		return constraints.Constrain(ui.Size{Width: constraints.MaxWidth, Height: y - r.overflowSpacing})
	}
	overallWidth := actualWidth
	if r.alignment.IsSome() {
		overallWidth = constraints.MaxWidth
	}
	return constraints.Constrain(ui.Size{Width: overallWidth, Height: maxChildHeight})
}

// PerformLayout lays the children out in a row, or in a column when they do not
// fit.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.performLayout
func (r *RenderOverflowBar) PerformLayout() {
	constraints := r.BoxConstraints()
	if isNil(r.FirstChild()) {
		r.SetSize(constraints.Smallest())
		return
	}
	childConstraints := constraints.Loosen()
	childrenWidth, maxChildHeight := 0.0, 0.0
	for c := r.FirstChild(); !isNil(c); c = r.ChildAfter(c) {
		c.Layout(childConstraints, LayoutOpts{ParentUsesSize: true})
		childrenWidth += c.Size().Width
		maxChildHeight = math.Max(maxChildHeight, c.Size().Height)
	}
	rtl := r.textDirection == ui.TextDirectionRTL
	count := float64(r.ChildCount())
	actualWidth := childrenWidth + r.spacing*(count-1)

	if actualWidth > constraints.MaxWidth {
		// Overflow vertical layout.
		child, nextChild := r.FirstChild(), r.ChildAfter
		if r.overflowDirection != painting.VerticalDirectionDown {
			child, nextChild = r.LastChild(), r.ChildBefore
		}
		y := 0.0
		for !isNil(child) {
			var x float64
			switch r.overflowAlignment {
			case OverflowBarAlignmentCenter:
				x = (constraints.MaxWidth - child.Size().Width) / 2
			case OverflowBarAlignmentStart:
				if rtl {
					x = constraints.MaxWidth - child.Size().Width
				}
			case OverflowBarAlignmentEnd:
				if !rtl {
					x = constraints.MaxWidth - child.Size().Width
				}
			}
			overflowBarParentData(child).Offset = ui.Offset{Dx: x, Dy: y}
			y += child.Size().Height + r.overflowSpacing
			child = nextChild(child)
		}
		r.SetSize(constraints.Constrain(ui.Size{Width: constraints.MaxWidth, Height: y - r.overflowSpacing}))
		return
	}

	// Default horizontal layout.
	child := r.FirstChild()
	firstChildWidth := child.Size().Width
	overallWidth := actualWidth
	if r.alignment.IsSome() {
		overallWidth = constraints.MaxWidth
	}
	size := constraints.Constrain(ui.Size{Width: overallWidth, Height: maxChildHeight})
	r.SetSize(size)
	var x float64 // initial value: origin of the first child
	layoutSpacing := r.spacing
	startX := func() float64 {
		if rtl {
			return size.Width - firstChildWidth
		}
		return 0
	}
	if a, ok := r.alignment.Get(); !ok {
		x = startX()
	} else {
		switch a {
		case MainAxisAlignmentCenter:
			half := (size.Width - actualWidth) / 2
			if rtl {
				x = size.Width - half - firstChildWidth
			} else {
				x = half
			}
		case MainAxisAlignmentEnd:
			if rtl {
				x = actualWidth - firstChildWidth
			} else {
				x = size.Width - actualWidth
			}
		case MainAxisAlignmentSpaceBetween:
			layoutSpacing = (size.Width - childrenWidth) / (count - 1)
			x = startX()
		case MainAxisAlignmentSpaceAround:
			if count > 0 {
				layoutSpacing = (size.Width - childrenWidth) / count
			} else {
				layoutSpacing = 0
			}
			if rtl {
				x = size.Width - layoutSpacing/2 - firstChildWidth
			} else {
				x = layoutSpacing / 2
			}
		case MainAxisAlignmentSpaceEvenly:
			layoutSpacing = (size.Width - childrenWidth) / (count + 1)
			if rtl {
				x = size.Width - layoutSpacing - firstChildWidth
			} else {
				x = layoutSpacing
			}
		default:
			x = startX()
		}
	}
	for !isNil(child) {
		overflowBarParentData(child).Offset = ui.Offset{Dx: x, Dy: (maxChildHeight - child.Size().Height) / 2}
		// x is the horizontal origin of child. To advance x to the next child's
		// origin for LTR: add the width of the current child. To advance x to the
		// origin of the next child for RTL: subtract the width of the next child (if
		// there is one).
		if !rtl {
			x += child.Size().Width + layoutSpacing
		}
		child = r.ChildAfter(child)
		if rtl && !isNil(child) {
			x -= child.Size().Width + layoutSpacing
		}
	}
}

// HitTestChildren hit tests the children.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.hitTestChildren
func (r *RenderOverflowBar) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	return DefaultHitTestChildren(&r.ContainerRenderObject, result, position)
}

// Paint paints the children.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.paint
func (r *RenderOverflowBar) Paint(context *PaintingContext, offset ui.Offset) {
	DefaultPaint(&r.ContainerRenderObject, context, offset)
}

// DebugFillProperties adds the configuration.
//
// Flutter: widgets/overflow_bar.dart _RenderOverflowBar.debugFillProperties
func (r *RenderOverflowBar) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	r.RenderBox.DebugFillProperties(p)
	p.Add(foundation.NewDoubleProperty("spacing", foundation.Some(r.spacing), foundation.NumberPropertyOpts{DefaultValue: 0.0}))
	p.Add(foundation.NewDoubleProperty("overflowSpacing", foundation.Some(r.overflowSpacing), foundation.NumberPropertyOpts{DefaultValue: 0.0}))
	p.Add(foundation.NewEnumProperty("overflowAlignment", foundation.Some(r.overflowAlignment), foundation.EnumPropertyOpts{DefaultValue: OverflowBarAlignmentStart}))
	p.Add(foundation.NewEnumProperty("overflowDirection", foundation.Some(r.overflowDirection), foundation.EnumPropertyOpts{DefaultValue: painting.VerticalDirectionDown}))
	p.Add(foundation.NewEnumProperty("textDirection", foundation.Some(r.textDirection), foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultNull}))
}
