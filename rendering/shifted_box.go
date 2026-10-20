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

// BoxConstraintsTransform turns the constraints of a box into those of its
// child.
//
// Flutter: rendering/shifted_box.dart BoxConstraintsTransform
type BoxConstraintsTransform func(constraints BoxConstraints) BoxConstraints

// RenderShiftedBox is a render box that contains a single child and positions
// it at an offset that its subclass sets in the child's BoxParentData. Embed it
// and call InitRenderShiftedBox in the constructor.
//
// Flutter: rendering/shifted_box.dart RenderShiftedBox
type RenderShiftedBox struct {
	RenderBox
	RenderObjectWithChild[Box]
}

// InitRenderShiftedBox sets the outer object of a shifted box and its child (nil
// for none).
//
// Flutter: rendering/shifted_box.dart RenderShiftedBox.new
func InitRenderShiftedBox(s *RenderShiftedBox, self Box, child Box) {
	InitRenderBox(&s.RenderBox, self)
	InitRenderObjectWithChild(&s.RenderObjectWithChild, self)
	if !isNil(child) {
		s.SetChild(child)
	}
}

func (s *RenderShiftedBox) hasChild() bool { return !isNil(s.Child()) }

// ComputeMinIntrinsicWidth is the child's, or 0.
//
// Flutter: rendering/shifted_box.dart RenderShiftedBox.computeMinIntrinsicWidth
func (s *RenderShiftedBox) ComputeMinIntrinsicWidth(height float64) float64 {
	if s.hasChild() {
		return s.Child().GetMinIntrinsicWidth(height)
	}
	return 0
}

// ComputeMaxIntrinsicWidth is the child's, or 0.
//
// Flutter: rendering/shifted_box.dart RenderShiftedBox.computeMaxIntrinsicWidth
func (s *RenderShiftedBox) ComputeMaxIntrinsicWidth(height float64) float64 {
	if s.hasChild() {
		return s.Child().GetMaxIntrinsicWidth(height)
	}
	return 0
}

// ComputeMinIntrinsicHeight is the child's, or 0.
//
// Flutter: rendering/shifted_box.dart RenderShiftedBox.computeMinIntrinsicHeight
func (s *RenderShiftedBox) ComputeMinIntrinsicHeight(width float64) float64 {
	if s.hasChild() {
		return s.Child().GetMinIntrinsicHeight(width)
	}
	return 0
}

// ComputeMaxIntrinsicHeight is the child's, or 0.
//
// Flutter: rendering/shifted_box.dart RenderShiftedBox.computeMaxIntrinsicHeight
func (s *RenderShiftedBox) ComputeMaxIntrinsicHeight(width float64) float64 {
	if s.hasChild() {
		return s.Child().GetMaxIntrinsicHeight(width)
	}
	return 0
}

// ComputeDistanceToActualBaseline is the child's baseline plus the child's
// offset.
//
// Flutter: rendering/shifted_box.dart RenderShiftedBox.computeDistanceToActualBaseline
func (s *RenderShiftedBox) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	if !s.hasChild() {
		return s.RenderBox.ComputeDistanceToActualBaseline(baseline)
	}
	child := s.Child()
	if result, ok := child.GetDistanceToActualBaseline(baseline).Get(); ok {
		return foundation.Some(result + BoxParentDataOf(child).Offset.Dy)
	}
	return foundation.None[float64]()
}

// ComputeDryBaseline is the child's baseline for the constraints.
//
// Flutter: rendering/shifted_box.dart RenderShiftedBox.computeDryBaseline
func (s *RenderShiftedBox) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !s.hasChild() {
		return foundation.None[float64]()
	}
	return s.Child().GetDryBaseline(constraints, baseline)
}

// Paint paints the child at its offset.
//
// Flutter: rendering/shifted_box.dart RenderShiftedBox.paint
func (s *RenderShiftedBox) Paint(context *PaintingContext, offset ui.Offset) {
	if s.hasChild() {
		child := s.Child()
		context.PaintChild(child, BoxParentDataOf(child).Offset.Add(offset))
	}
}

// HitTestChildren hit tests the child at its offset.
//
// Flutter: rendering/shifted_box.dart RenderShiftedBox.hitTestChildren
func (s *RenderShiftedBox) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	if !s.hasChild() {
		return false
	}
	child := s.Child()
	childOffset := BoxParentDataOf(child).Offset
	return result.AddWithPaintOffset(&childOffset, position, func(result *BoxHitTestResult, transformed ui.Offset) bool {
		return child.HitTest(result, transformed)
	})
}

// sizeDiff is the difference of two sizes as an offset, as Dart's Size - Size.
func sizeDiff(a, b ui.Size) ui.Offset {
	return ui.Offset{Dx: a.Width - b.Width, Dy: a.Height - b.Height}
}

// RenderPadding insets its child by the given padding.
//
// Flutter: rendering/shifted_box.dart RenderPadding
type RenderPadding struct {
	RenderShiftedBox
	padding         painting.EdgeInsetsGeometry
	textDirection   ui.TextDirection
	resolvedPadding foundation.Opt[painting.EdgeInsets]
}

// RenderPaddingOpts are the optional parameters of NewRenderPadding.
//
// Flutter: rendering/shifted_box.dart RenderPadding.new
type RenderPaddingOpts struct {
	// TextDirection resolves a directional padding.
	TextDirection ui.TextDirection
}

// NewRenderPadding creates the box. A negative padding panics with
// PaddingNegative in debug builds.
//
// Flutter: rendering/shifted_box.dart RenderPadding.new
func NewRenderPadding(padding painting.EdgeInsetsGeometry, child Box, opts ...RenderPaddingOpts) *RenderPadding {
	assertPaddingNonNegative(padding)
	r := &RenderPadding{padding: padding, textDirection: firstOpts(opts).TextDirection}
	InitRenderShiftedBox(&r.RenderShiftedBox, r, child)
	return r
}

func assertPaddingNonNegative(padding painting.EdgeInsetsGeometry) {
	if foundation.DebugMode && !padding.IsNonNegative() {
		panic(errors.New("PaddingNegative", "the padding must not be negative", "padding", padding))
	}
}

func (r *RenderPadding) resolved() painting.EdgeInsets {
	if !r.resolvedPadding.IsSome() {
		r.resolvedPadding = foundation.Some(r.padding.Resolve(r.textDirection))
	}
	v := r.resolvedPadding.Value()
	if foundation.DebugMode && !v.IsNonNegative() {
		panic(errors.New("PaddingNegative", "the resolved padding must not be negative", "padding", v))
	}
	return v
}

func (r *RenderPadding) markNeedResolution() {
	r.resolvedPadding = foundation.None[painting.EdgeInsets]()
	r.MarkNeedsLayout()
}

// Padding is the amount to pad the child in each dimension.
//
// Flutter: rendering/shifted_box.dart RenderPadding.padding
func (r *RenderPadding) Padding() painting.EdgeInsetsGeometry { return r.padding }

// SetPadding changes the padding and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderPadding.padding
func (r *RenderPadding) SetPadding(value painting.EdgeInsetsGeometry) {
	assertPaddingNonNegative(value)
	if r.padding.Equal(value) {
		return
	}
	r.padding = value
	r.markNeedResolution()
}

// TextDirection resolves a directional padding.
//
// Flutter: rendering/shifted_box.dart RenderPadding.textDirection
func (r *RenderPadding) TextDirection() ui.TextDirection { return r.textDirection }

// SetTextDirection changes the direction and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderPadding.textDirection
func (r *RenderPadding) SetTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.markNeedResolution()
}

// ComputeMinIntrinsicWidth is the child's plus the horizontal padding.
//
// Flutter: rendering/shifted_box.dart RenderPadding.computeMinIntrinsicWidth
func (r *RenderPadding) ComputeMinIntrinsicWidth(height float64) float64 {
	p := r.resolved()
	if r.hasChild() {
		return r.Child().GetMinIntrinsicWidth(math.Max(0, height-p.Vertical())) + p.Horizontal()
	}
	return p.Horizontal()
}

// ComputeMaxIntrinsicWidth is the child's plus the horizontal padding.
//
// Flutter: rendering/shifted_box.dart RenderPadding.computeMaxIntrinsicWidth
func (r *RenderPadding) ComputeMaxIntrinsicWidth(height float64) float64 {
	p := r.resolved()
	if r.hasChild() {
		return r.Child().GetMaxIntrinsicWidth(math.Max(0, height-p.Vertical())) + p.Horizontal()
	}
	return p.Horizontal()
}

// ComputeMinIntrinsicHeight is the child's plus the vertical padding.
//
// Flutter: rendering/shifted_box.dart RenderPadding.computeMinIntrinsicHeight
func (r *RenderPadding) ComputeMinIntrinsicHeight(width float64) float64 {
	p := r.resolved()
	if r.hasChild() {
		return r.Child().GetMinIntrinsicHeight(math.Max(0, width-p.Horizontal())) + p.Vertical()
	}
	return p.Vertical()
}

// ComputeMaxIntrinsicHeight is the child's plus the vertical padding.
//
// Flutter: rendering/shifted_box.dart RenderPadding.computeMaxIntrinsicHeight
func (r *RenderPadding) ComputeMaxIntrinsicHeight(width float64) float64 {
	p := r.resolved()
	if r.hasChild() {
		return r.Child().GetMaxIntrinsicHeight(math.Max(0, width-p.Horizontal())) + p.Vertical()
	}
	return p.Vertical()
}

// ComputeDryLayout is the size of the child plus the padding.
//
// Flutter: rendering/shifted_box.dart RenderPadding.computeDryLayout
func (r *RenderPadding) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	p := r.resolved()
	if !r.hasChild() {
		return constraints.Constrain(ui.Size{Width: p.Horizontal(), Height: p.Vertical()})
	}
	childSize := r.Child().GetDryLayout(constraints.Deflate(p))
	return constraints.Constrain(ui.Size{Width: p.Horizontal() + childSize.Width, Height: p.Vertical() + childSize.Height})
}

// ComputeDryBaseline is the child's baseline plus the top padding.
//
// Flutter: rendering/shifted_box.dart RenderPadding.computeDryBaseline
func (r *RenderPadding) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	p := r.resolved()
	if b, ok := r.Child().GetDryBaseline(constraints.Deflate(p), baseline).Get(); ok {
		return foundation.Some(b + p.Top)
	}
	return foundation.None[float64]()
}

// PerformLayout lays the child out in the deflated constraints.
//
// Flutter: rendering/shifted_box.dart RenderPadding.performLayout
func (r *RenderPadding) PerformLayout() {
	constraints := r.BoxConstraints()
	p := r.resolved()
	if !r.hasChild() {
		r.SetSize(constraints.Constrain(ui.Size{Width: p.Horizontal(), Height: p.Vertical()}))
		return
	}
	child := r.Child()
	child.Layout(constraints.Deflate(p), LayoutOpts{ParentUsesSize: true})
	BoxParentDataOf(child).Offset = ui.Offset{Dx: p.Left, Dy: p.Top}
	r.SetSize(constraints.Constrain(ui.Size{Width: p.Horizontal() + child.Size().Width, Height: p.Vertical() + child.Size().Height}))
}

// DebugPaintSize also paints the padding.
//
// Flutter: rendering/shifted_box.dart RenderPadding.debugPaintSize
func (r *RenderPadding) DebugPaintSize(context *PaintingContext, offset ui.Offset) {
	r.RenderShiftedBox.DebugPaintSize(context, offset)
	if foundation.DebugMode {
		outer := offset.And(r.Size())
		inner := foundation.None[ui.Rect]()
		if r.hasChild() {
			inner = foundation.Some(r.resolved().DeflateRect(outer))
		}
		DebugPaintPadding(context.Canvas(), outer, inner, 0)
	}
}

// DebugFillProperties adds the padding and the text direction.
//
// Flutter: rendering/shifted_box.dart RenderPadding.debugFillProperties
func (r *RenderPadding) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderShiftedBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("padding", r.padding))
	b.Add(foundation.NewEnumProperty("textDirection", foundation.Some(r.textDirection), foundation.EnumPropertyOpts{DefaultValue: ui.TextDirectionUnspecified}))
}

// RenderAligningShiftedBox is a shifted box that aligns its child within itself.
// Embed it and call InitRenderAligningShiftedBox in the constructor.
//
// Flutter: rendering/shifted_box.dart RenderAligningShiftedBox
type RenderAligningShiftedBox struct {
	RenderShiftedBox
	alignment         painting.AlignmentGeometry
	textDirection     ui.TextDirection
	resolvedAlignment foundation.Opt[painting.Alignment]
}

// InitRenderAligningShiftedBox sets the outer object, the alignment (nil is the
// center), the text direction and the child (nil for none).
//
// Flutter: rendering/shifted_box.dart RenderAligningShiftedBox.new
func InitRenderAligningShiftedBox(a *RenderAligningShiftedBox, self Box, alignment painting.AlignmentGeometry, textDirection ui.TextDirection, child Box) {
	if alignment == nil {
		alignment = painting.AlignmentCenter
	}
	a.alignment = alignment
	a.textDirection = textDirection
	InitRenderShiftedBox(&a.RenderShiftedBox, self, child)
}

// ResolvedAlignment is the alignment resolved with the text direction.
//
// Flutter: rendering/shifted_box.dart RenderAligningShiftedBox.resolvedAlignment
func (a *RenderAligningShiftedBox) ResolvedAlignment() painting.Alignment {
	if !a.resolvedAlignment.IsSome() {
		a.resolvedAlignment = foundation.Some(a.alignment.Resolve(a.textDirection))
	}
	return a.resolvedAlignment.Value()
}

func (a *RenderAligningShiftedBox) markNeedResolution() {
	a.resolvedAlignment = foundation.None[painting.Alignment]()
	a.MarkNeedsLayout()
}

// Alignment is how to align the child.
//
// Flutter: rendering/shifted_box.dart RenderAligningShiftedBox.alignment
func (a *RenderAligningShiftedBox) Alignment() painting.AlignmentGeometry { return a.alignment }

// SetAlignment changes the alignment and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderAligningShiftedBox.alignment
func (a *RenderAligningShiftedBox) SetAlignment(value painting.AlignmentGeometry) {
	if a.alignment.Equal(value) {
		return
	}
	a.alignment = value
	a.markNeedResolution()
}

// TextDirection resolves a directional alignment.
//
// Flutter: rendering/shifted_box.dart RenderAligningShiftedBox.textDirection
func (a *RenderAligningShiftedBox) TextDirection() ui.TextDirection { return a.textDirection }

// SetTextDirection changes the direction and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderAligningShiftedBox.textDirection
func (a *RenderAligningShiftedBox) SetTextDirection(value ui.TextDirection) {
	if a.textDirection == value {
		return
	}
	a.textDirection = value
	a.markNeedResolution()
}

// AlignChild sets the child's offset from its alignment, once the box and the
// child have a size.
//
// Flutter: rendering/shifted_box.dart RenderAligningShiftedBox.alignChild
func (a *RenderAligningShiftedBox) AlignChild() {
	child := a.Child()
	BoxParentDataOf(child).Offset = a.ResolvedAlignment().AlongOffset(sizeDiff(a.Size(), child.Size()))
}

// DebugFillProperties adds the alignment and the text direction.
//
// Flutter: rendering/shifted_box.dart RenderAligningShiftedBox.debugFillProperties
func (a *RenderAligningShiftedBox) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	a.RenderShiftedBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("alignment", a.alignment))
	b.Add(foundation.NewEnumProperty("textDirection", foundation.Some(a.textDirection), foundation.EnumPropertyOpts{DefaultValue: ui.TextDirectionUnspecified}))
}

// RenderPositionedBox positions its child using an alignment within itself.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox
type RenderPositionedBox struct {
	RenderAligningShiftedBox
	widthFactor, heightFactor foundation.Opt[float64]
}

// RenderPositionedBoxOpts are the optional parameters of NewRenderPositionedBox.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.new
type RenderPositionedBoxOpts struct {
	// WidthFactor sizes the box to the child's width times it, if set.
	WidthFactor foundation.Opt[float64]
	// HeightFactor sizes the box to the child's height times it, if set.
	HeightFactor foundation.Opt[float64]
	// Alignment defaults to painting.AlignmentCenter.
	Alignment painting.AlignmentGeometry
	// TextDirection resolves a directional alignment.
	TextDirection ui.TextDirection
}

// NewRenderPositionedBox creates the box. A negative factor panics with
// SizeFactorNegative in debug builds.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.new
func NewRenderPositionedBox(child Box, opts ...RenderPositionedBoxOpts) *RenderPositionedBox {
	o := firstOpts(opts)
	assertFactor(o.WidthFactor)
	assertFactor(o.HeightFactor)
	r := &RenderPositionedBox{widthFactor: o.WidthFactor, heightFactor: o.HeightFactor}
	InitRenderAligningShiftedBox(&r.RenderAligningShiftedBox, r, o.Alignment, o.TextDirection, child)
	return r
}

func assertFactor(f foundation.Opt[float64]) {
	if v, ok := f.Get(); ok && foundation.DebugMode && v < 0 {
		panic(errors.New("SizeFactorNegative", "a size factor must not be negative", "factor", v))
	}
}

// WidthFactor sizes the box to the child's width times it, if set.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.widthFactor
func (r *RenderPositionedBox) WidthFactor() foundation.Opt[float64] { return r.widthFactor }

// SetWidthFactor changes the factor and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.widthFactor
func (r *RenderPositionedBox) SetWidthFactor(value foundation.Opt[float64]) {
	assertFactor(value)
	if r.widthFactor == value {
		return
	}
	r.widthFactor = value
	r.MarkNeedsLayout()
}

// HeightFactor sizes the box to the child's height times it, if set.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.heightFactor
func (r *RenderPositionedBox) HeightFactor() foundation.Opt[float64] { return r.heightFactor }

// SetHeightFactor changes the factor and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.heightFactor
func (r *RenderPositionedBox) SetHeightFactor(value foundation.Opt[float64]) {
	assertFactor(value)
	if r.heightFactor == value {
		return
	}
	r.heightFactor = value
	r.MarkNeedsLayout()
}

// ComputeMinIntrinsicWidth is the child's times the width factor.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.computeMinIntrinsicWidth
func (r *RenderPositionedBox) ComputeMinIntrinsicWidth(height float64) float64 {
	return r.RenderAligningShiftedBox.ComputeMinIntrinsicWidth(height) * r.widthFactor.OrElse(1)
}

// ComputeMaxIntrinsicWidth is the child's times the width factor.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.computeMaxIntrinsicWidth
func (r *RenderPositionedBox) ComputeMaxIntrinsicWidth(height float64) float64 {
	return r.RenderAligningShiftedBox.ComputeMaxIntrinsicWidth(height) * r.widthFactor.OrElse(1)
}

// ComputeMinIntrinsicHeight is the child's times the height factor.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.computeMinIntrinsicHeight
func (r *RenderPositionedBox) ComputeMinIntrinsicHeight(width float64) float64 {
	return r.RenderAligningShiftedBox.ComputeMinIntrinsicHeight(width) * r.heightFactor.OrElse(1)
}

// ComputeMaxIntrinsicHeight is the child's times the height factor.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.computeMaxIntrinsicHeight
func (r *RenderPositionedBox) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.RenderAligningShiftedBox.ComputeMaxIntrinsicHeight(width) * r.heightFactor.OrElse(1)
}

func (r *RenderPositionedBox) sizeFor(constraints BoxConstraints, childSize foundation.Opt[ui.Size]) ui.Size {
	shrinkW := r.widthFactor.IsSome() || math.IsInf(constraints.MaxWidth, 1)
	shrinkH := r.heightFactor.IsSome() || math.IsInf(constraints.MaxHeight, 1)
	w, h := math.Inf(1), math.Inf(1)
	cs, hasChild := childSize.Get()
	if shrinkW {
		w = 0
		if hasChild {
			w = cs.Width * r.widthFactor.OrElse(1)
		}
	}
	if shrinkH {
		h = 0
		if hasChild {
			h = cs.Height * r.heightFactor.OrElse(1)
		}
	}
	return constraints.Constrain(ui.Size{Width: w, Height: h})
}

// ComputeDryLayout is the size for the child's loosened dry layout.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.computeDryLayout
func (r *RenderPositionedBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if r.hasChild() {
		return r.sizeFor(constraints, foundation.Some(r.Child().GetDryLayout(constraints.Loosen())))
	}
	return r.sizeFor(constraints, foundation.None[ui.Size]())
}

// PerformLayout lays the child out loosely and aligns it.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.performLayout
func (r *RenderPositionedBox) PerformLayout() {
	constraints := r.BoxConstraints()
	if !r.hasChild() {
		r.SetSize(r.sizeFor(constraints, foundation.None[ui.Size]()))
		return
	}
	child := r.Child()
	child.Layout(constraints.Loosen(), LayoutOpts{ParentUsesSize: true})
	r.SetSize(r.sizeFor(constraints, foundation.Some(child.Size())))
	r.AlignChild()
}

// ComputeDryBaseline is the child's baseline plus the aligned offset.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.computeDryBaseline
func (r *RenderPositionedBox) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	child := r.Child()
	childConstraints := constraints.Loosen()
	result, ok := child.GetDryBaseline(childConstraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	childSize := child.GetDryLayout(childConstraints)
	size := r.sizeFor(constraints, foundation.Some(childSize))
	return foundation.Some(result + r.ResolvedAlignment().AlongOffset(sizeDiff(size, childSize)).Dy)
}

// DebugPaintSize paints arrows that show the free space around the child.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.debugPaintSize
func (r *RenderPositionedBox) DebugPaintSize(context *PaintingContext, offset ui.Offset) {
	r.RenderAligningShiftedBox.DebugPaintSize(context, offset)
	if !foundation.DebugMode {
		return
	}
	if !r.hasChild() || r.Child().Size().IsEmpty() {
		paint := ui.NewPaint()
		paint.SetColor(ui.NewColor(0x90909090))
		context.Canvas().DrawRect(offset.And(r.Size()), paint)
		return
	}
	paint := ui.NewPaint()
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(1)
	paint.SetColor(ui.NewColor(0xFFFFFF00))
	path := ui.NewPath()
	size := r.Size()
	co := BoxParentDataOf(r.Child()).Offset
	if co.Dy > 0 {
		head := math.Min(co.Dy*0.2, 10)
		path.MoveTo(offset.Dx+size.Width/2, offset.Dy)
		path.RelativeLineTo(0, co.Dy-head)
		path.RelativeLineTo(head, 0)
		path.RelativeLineTo(-head, head)
		path.RelativeLineTo(-head, -head)
		path.RelativeLineTo(head, 0)
		path.MoveTo(offset.Dx+size.Width/2, offset.Dy+size.Height)
		path.RelativeLineTo(0, -co.Dy+head)
		path.RelativeLineTo(head, 0)
		path.RelativeLineTo(-head, -head)
		path.RelativeLineTo(-head, head)
		path.RelativeLineTo(head, 0)
		context.Canvas().DrawPath(path, paint)
	}
	if co.Dx > 0 {
		head := math.Min(co.Dx*0.2, 10)
		path.MoveTo(offset.Dx, offset.Dy+size.Height/2)
		path.RelativeLineTo(co.Dx-head, 0)
		path.RelativeLineTo(0, head)
		path.RelativeLineTo(head, -head)
		path.RelativeLineTo(-head, -head)
		path.RelativeLineTo(0, head)
		path.MoveTo(offset.Dx+size.Width, offset.Dy+size.Height/2)
		path.RelativeLineTo(-co.Dx+head, 0)
		path.RelativeLineTo(0, head)
		path.RelativeLineTo(-head, -head)
		path.RelativeLineTo(head, -head)
		path.RelativeLineTo(0, head)
		context.Canvas().DrawPath(path, paint)
	}
}

// DebugFillProperties adds the factors.
//
// Flutter: rendering/shifted_box.dart RenderPositionedBox.debugFillProperties
func (r *RenderPositionedBox) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderAligningShiftedBox.DebugFillProperties(b)
	b.Add(foundation.NewDoubleProperty("widthFactor", r.widthFactor, foundation.NumberPropertyOpts{IfNull: "expand"}))
	b.Add(foundation.NewDoubleProperty("heightFactor", r.heightFactor, foundation.NumberPropertyOpts{IfNull: "expand"}))
}

// OverflowBoxFit is how a RenderConstrainedOverflowBox sizes itself.
//
// Flutter: rendering/shifted_box.dart OverflowBoxFit
type OverflowBoxFit int

// The values of OverflowBoxFit follow Flutter's declaration order.
const (
	// OverflowBoxFitUnspecified is the zero value: the widget's default (max).
	//
	// Flutter: no counterpart
	OverflowBoxFitUnspecified OverflowBoxFit = iota
	// OverflowBoxFitMax passes the constraints of the parent: the box is as big
	// as possible.
	//
	// Flutter: rendering/shifted_box.dart OverflowBoxFit.max
	OverflowBoxFitMax
	// OverflowBoxFitDeferToChild sizes the box to the child.
	//
	// Flutter: rendering/shifted_box.dart OverflowBoxFit.deferToChild
	OverflowBoxFitDeferToChild
)

// Name returns the Dart enum name, such as "max".
//
// Flutter: rendering/shifted_box.dart OverflowBoxFit.name
func (f OverflowBoxFit) Name() string {
	switch f {
	case OverflowBoxFitMax:
		return "max"
	case OverflowBoxFitDeferToChild:
		return "deferToChild"
	}
	return "unspecified"
}

// String returns the name Dart prints, such as "OverflowBoxFit.max".
//
// Flutter: rendering/shifted_box.dart OverflowBoxFit.toString
func (f OverflowBoxFit) String() string { return "OverflowBoxFit." + f.Name() }

// RenderConstrainedOverflowBox imposes different constraints on its child than
// it gets from its parent, possibly allowing the child to overflow.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox
type RenderConstrainedOverflowBox struct {
	RenderAligningShiftedBox
	minWidth, maxWidth, minHeight, maxHeight foundation.Opt[float64]
	fit                                      OverflowBoxFit
}

// RenderConstrainedOverflowBoxOpts are the optional parameters of
// NewRenderConstrainedOverflowBox.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.new
type RenderConstrainedOverflowBoxOpts struct {
	// MinWidth replaces the minimum width constraint, if set.
	MinWidth foundation.Opt[float64]
	// MaxWidth replaces the maximum width constraint, if set.
	MaxWidth foundation.Opt[float64]
	// MinHeight replaces the minimum height constraint, if set.
	MinHeight foundation.Opt[float64]
	// MaxHeight replaces the maximum height constraint, if set.
	MaxHeight foundation.Opt[float64]
	// Fit defaults to OverflowBoxFitMax.
	Fit OverflowBoxFit
	// Alignment defaults to painting.AlignmentCenter.
	Alignment painting.AlignmentGeometry
	// TextDirection resolves a directional alignment.
	TextDirection ui.TextDirection
}

// NewRenderConstrainedOverflowBox creates the box.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.new
func NewRenderConstrainedOverflowBox(child Box, opts ...RenderConstrainedOverflowBoxOpts) *RenderConstrainedOverflowBox {
	o := firstOpts(opts)
	if o.Fit == OverflowBoxFitUnspecified {
		o.Fit = OverflowBoxFitMax
	}
	r := &RenderConstrainedOverflowBox{minWidth: o.MinWidth, maxWidth: o.MaxWidth, minHeight: o.MinHeight, maxHeight: o.MaxHeight, fit: o.Fit}
	InitRenderAligningShiftedBox(&r.RenderAligningShiftedBox, r, o.Alignment, o.TextDirection, child)
	return r
}

// MinWidth replaces the minimum width constraint of the child, if set.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.minWidth
func (r *RenderConstrainedOverflowBox) MinWidth() foundation.Opt[float64] { return r.minWidth }

// SetMinWidth changes the constraint and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.minWidth
func (r *RenderConstrainedOverflowBox) SetMinWidth(value foundation.Opt[float64]) {
	if r.minWidth == value {
		return
	}
	r.minWidth = value
	r.MarkNeedsLayout()
}

// MaxWidth replaces the maximum width constraint of the child, if set.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.maxWidth
func (r *RenderConstrainedOverflowBox) MaxWidth() foundation.Opt[float64] { return r.maxWidth }

// SetMaxWidth changes the constraint and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.maxWidth
func (r *RenderConstrainedOverflowBox) SetMaxWidth(value foundation.Opt[float64]) {
	if r.maxWidth == value {
		return
	}
	r.maxWidth = value
	r.MarkNeedsLayout()
}

// MinHeight replaces the minimum height constraint of the child, if set.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.minHeight
func (r *RenderConstrainedOverflowBox) MinHeight() foundation.Opt[float64] { return r.minHeight }

// SetMinHeight changes the constraint and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.minHeight
func (r *RenderConstrainedOverflowBox) SetMinHeight(value foundation.Opt[float64]) {
	if r.minHeight == value {
		return
	}
	r.minHeight = value
	r.MarkNeedsLayout()
}

// MaxHeight replaces the maximum height constraint of the child, if set.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.maxHeight
func (r *RenderConstrainedOverflowBox) MaxHeight() foundation.Opt[float64] { return r.maxHeight }

// SetMaxHeight changes the constraint and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.maxHeight
func (r *RenderConstrainedOverflowBox) SetMaxHeight(value foundation.Opt[float64]) {
	if r.maxHeight == value {
		return
	}
	r.maxHeight = value
	r.MarkNeedsLayout()
}

// Fit is how the box sizes itself.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.fit
func (r *RenderConstrainedOverflowBox) Fit() OverflowBoxFit { return r.fit }

// SetFit changes the fit; the box is sized by its parent when it is max.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.fit
func (r *RenderConstrainedOverflowBox) SetFit(value OverflowBoxFit) {
	if r.fit == value {
		return
	}
	r.fit = value
	r.MarkNeedsLayoutForSizedByParentChange()
}

func (r *RenderConstrainedOverflowBox) innerConstraints(c BoxConstraints) BoxConstraints {
	return BoxConstraints{
		MinWidth: r.minWidth.OrElse(c.MinWidth), MaxWidth: r.maxWidth.OrElse(c.MaxWidth),
		MinHeight: r.minHeight.OrElse(c.MinHeight), MaxHeight: r.maxHeight.OrElse(c.MaxHeight)}
}

// SizedByParent is true for the max fit.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.sizedByParent
func (r *RenderConstrainedOverflowBox) SizedByParent() bool { return r.fit == OverflowBoxFitMax }

// ComputeDryLayout is the biggest size for the max fit, the child's otherwise.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.computeDryLayout
func (r *RenderConstrainedOverflowBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if r.fit == OverflowBoxFitMax {
		return constraints.Biggest()
	}
	if r.hasChild() {
		return r.Child().GetDryLayout(constraints)
	}
	return constraints.Smallest()
}

// ComputeDryBaseline is the child's baseline plus the aligned offset.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.computeDryBaseline
func (r *RenderConstrainedOverflowBox) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	child := r.Child()
	childConstraints := r.innerConstraints(constraints)
	result, ok := child.GetDryBaseline(childConstraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	childSize := child.GetDryLayout(childConstraints)
	size := r.GetDryLayout(constraints)
	return foundation.Some(result + r.ResolvedAlignment().AlongOffset(sizeDiff(size, childSize)).Dy)
}

// PerformLayout lays the child out with the replaced constraints.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.performLayout
func (r *RenderConstrainedOverflowBox) PerformLayout() {
	constraints := r.BoxConstraints()
	if r.hasChild() {
		child := r.Child()
		child.Layout(r.innerConstraints(constraints), LayoutOpts{ParentUsesSize: true})
		if r.fit == OverflowBoxFitDeferToChild {
			r.SetSize(constraints.Constrain(child.Size()))
		}
		r.AlignChild()
		return
	}
	if r.fit == OverflowBoxFitDeferToChild {
		r.SetSize(constraints.Smallest())
	}
}

// DebugFillProperties adds the constraints and the fit.
//
// Flutter: rendering/shifted_box.dart RenderConstrainedOverflowBox.debugFillProperties
func (r *RenderConstrainedOverflowBox) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderAligningShiftedBox.DebugFillProperties(b)
	b.Add(foundation.NewDoubleProperty("minWidth", r.minWidth, foundation.NumberPropertyOpts{IfNull: "use parent minWidth constraint"}))
	b.Add(foundation.NewDoubleProperty("maxWidth", r.maxWidth, foundation.NumberPropertyOpts{IfNull: "use parent maxWidth constraint"}))
	b.Add(foundation.NewDoubleProperty("minHeight", r.minHeight, foundation.NumberPropertyOpts{IfNull: "use parent minHeight constraint"}))
	b.Add(foundation.NewDoubleProperty("maxHeight", r.maxHeight, foundation.NumberPropertyOpts{IfNull: "use parent maxHeight constraint"}))
	b.Add(foundation.NewEnumProperty("fit", foundation.Some(r.fit)))
}

// RenderConstraintsTransformBox transforms the constraints that its parent
// gives, with a function, before they reach its child, and clips (or marks) the
// overflow.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox
type RenderConstraintsTransformBox struct {
	RenderAligningShiftedBox
	DebugOverflowIndicator
	constraintsTransform BoxConstraintsTransform
	clipBehavior         ui.Clip
	clipRectLayer        *LayerHandle[*ClipRectLayer]
	overflowContainer    ui.Rect
	overflowChild        ui.Rect
	isOverflowing        bool
	childConstraints     foundation.Opt[BoxConstraints]
}

// RenderConstraintsTransformBoxOpts are the optional parameters of
// NewRenderConstraintsTransformBox.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.new
type RenderConstraintsTransformBoxOpts struct {
	// ClipBehavior defaults to ClipNone.
	ClipBehavior ui.Clip
}

// NewRenderConstraintsTransformBox creates the box.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.new
func NewRenderConstraintsTransformBox(alignment painting.AlignmentGeometry, textDirection ui.TextDirection,
	constraintsTransform BoxConstraintsTransform, child Box, opts ...RenderConstraintsTransformBoxOpts) *RenderConstraintsTransformBox {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	r := &RenderConstraintsTransformBox{constraintsTransform: constraintsTransform, clipBehavior: o.ClipBehavior,
		clipRectLayer: NewLayerHandle[*ClipRectLayer](nil)}
	InitRenderAligningShiftedBox(&r.RenderAligningShiftedBox, r, alignment, textDirection, child)
	return r
}

// ConstraintsTransform turns the parent's constraints into the child's.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.constraintsTransform
func (r *RenderConstraintsTransformBox) ConstraintsTransform() BoxConstraintsTransform {
	return r.constraintsTransform
}

// SetConstraintsTransform changes the function; layout is needed if the child's
// constraints change.
//
// Deviation: Dart skips an equal function; Go functions can not be compared, so
// the new constraints are compared instead (SPEC §12.10).
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.constraintsTransform
func (r *RenderConstraintsTransformBox) SetConstraintsTransform(value BoxConstraintsTransform) {
	r.constraintsTransform = value
	if c, ok := r.childConstraints.Get(); !ok || c != value(r.BoxConstraints()) {
		r.MarkNeedsLayout()
	}
}

// ClipBehavior is how the overflow is clipped.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.clipBehavior
func (r *RenderConstraintsTransformBox) ClipBehavior() ui.Clip { return r.clipBehavior }

// SetClipBehavior changes the clip behavior and marks the box as needing paint.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.clipBehavior
func (r *RenderConstraintsTransformBox) SetClipBehavior(value ui.Clip) {
	if value != r.clipBehavior {
		r.clipBehavior = value
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

// ComputeMinIntrinsicHeight transforms the width.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.computeMinIntrinsicHeight
func (r *RenderConstraintsTransformBox) ComputeMinIntrinsicHeight(width float64) float64 {
	return r.RenderAligningShiftedBox.ComputeMinIntrinsicHeight(
		r.constraintsTransform(BoxConstraints{MaxWidth: width, MaxHeight: math.Inf(1)}).MaxWidth)
}

// ComputeMaxIntrinsicHeight transforms the width.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.computeMaxIntrinsicHeight
func (r *RenderConstraintsTransformBox) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.RenderAligningShiftedBox.ComputeMaxIntrinsicHeight(
		r.constraintsTransform(BoxConstraints{MaxWidth: width, MaxHeight: math.Inf(1)}).MaxWidth)
}

// ComputeMinIntrinsicWidth transforms the height.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.computeMinIntrinsicWidth
func (r *RenderConstraintsTransformBox) ComputeMinIntrinsicWidth(height float64) float64 {
	return r.RenderAligningShiftedBox.ComputeMinIntrinsicWidth(
		r.constraintsTransform(BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: height}).MaxHeight)
}

// ComputeMaxIntrinsicWidth transforms the height.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.computeMaxIntrinsicWidth
func (r *RenderConstraintsTransformBox) ComputeMaxIntrinsicWidth(height float64) float64 {
	return r.RenderAligningShiftedBox.ComputeMaxIntrinsicWidth(
		r.constraintsTransform(BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: height}).MaxHeight)
}

// ComputeDryLayout is the child's size for the transformed constraints.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.computeDryLayout
func (r *RenderConstraintsTransformBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if !r.hasChild() {
		return constraints.Smallest()
	}
	return constraints.Constrain(r.Child().GetDryLayout(r.constraintsTransform(constraints)))
}

// ComputeDryBaseline is the child's baseline plus the aligned offset.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.computeDryBaseline
func (r *RenderConstraintsTransformBox) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	child := r.Child()
	childConstraints := r.constraintsTransform(constraints)
	result, ok := child.GetDryBaseline(childConstraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	childSize := child.GetDryLayout(childConstraints)
	size := constraints.Constrain(childSize)
	return foundation.Some(result + r.ResolvedAlignment().AlongOffset(sizeDiff(size, childSize)).Dy)
}

// PerformLayout lays the child out with the transformed constraints and works
// out whether it overflows.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.performLayout
func (r *RenderConstraintsTransformBox) PerformLayout() {
	constraints := r.BoxConstraints()
	if r.hasChild() {
		child := r.Child()
		childConstraints := r.constraintsTransform(constraints)
		if foundation.DebugMode && !childConstraints.IsNormalized() {
			panic(errors.New("ConstraintsTransformNotNormalized", "the transformed constraints are not normalized",
				"constraints", childConstraints, "object", foundation.DescribeIdentity(r.self)))
		}
		r.childConstraints = foundation.Some(childConstraints)
		child.Layout(childConstraints, LayoutOpts{ParentUsesSize: true})
		r.SetSize(constraints.Constrain(child.Size()))
		r.AlignChild()
		r.overflowContainer = ui.OffsetZero.And(r.Size())
		r.overflowChild = BoxParentDataOf(child).Offset.And(child.Size())
	} else {
		r.SetSize(constraints.Smallest())
		r.overflowContainer, r.overflowChild = ui.Rect{}, ui.Rect{}
	}
	r.isOverflowing = NewRelativeRectFromRect(r.overflowContainer, r.overflowChild).HasInsets()
}

// Paint clips the child if it overflows, and marks the overflow in debug builds.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.paint
func (r *RenderConstraintsTransformBox) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() {
		return
	}
	if !r.isOverflowing {
		r.RenderAligningShiftedBox.Paint(context, offset)
		return
	}
	r.clipRectLayer.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()),
		r.RenderAligningShiftedBox.Paint, PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: r.clipRectLayer.Layer()}))
	if foundation.DebugMode && !r.Size().IsEmpty() && r.clipBehavior == ui.ClipNone {
		r.PaintOverflowIndicator(r.self, context, offset, r.overflowContainer, r.overflowChild)
	}
}

// Dispose lets the clip layer go.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.dispose
func (r *RenderConstraintsTransformBox) Dispose() {
	r.clipRectLayer.SetLayer(nil)
	r.RenderAligningShiftedBox.Dispose()
}

// Reassemble makes the overflow be reported again.
//
// Flutter: rendering/debug_overflow_indicator.dart DebugOverflowIndicatorMixin.reassemble
func (r *RenderConstraintsTransformBox) Reassemble() {
	r.RenderAligningShiftedBox.Reassemble()
	r.ResetOverflowReport()
}

// DescribeApproximatePaintClip is the box if the child overflows and is clipped.
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.describeApproximatePaintClip
func (r *RenderConstraintsTransformBox) DescribeApproximatePaintClip(RenderObject) foundation.Opt[ui.Rect] {
	if r.clipBehavior != ui.ClipNone && r.isOverflowing {
		return foundation.Some(ui.OffsetZero.And(r.Size()))
	}
	return foundation.None[ui.Rect]()
}

// ToStringShort adds OVERFLOWING when the child overflows (debug builds).
//
// Flutter: rendering/shifted_box.dart RenderConstraintsTransformBox.toStringShort
func (r *RenderConstraintsTransformBox) ToStringShort() string {
	header := r.RenderAligningShiftedBox.ToStringShort()
	if foundation.DebugMode && r.isOverflowing {
		header += " OVERFLOWING"
	}
	return header
}

// RenderSizedOverflowBox is a fixed-size box that lets its child have a
// different size, possibly overflowing.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox
type RenderSizedOverflowBox struct {
	RenderAligningShiftedBox
	requestedSize ui.Size
}

// RenderSizedOverflowBoxOpts are the optional parameters of
// NewRenderSizedOverflowBox.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.new
type RenderSizedOverflowBoxOpts struct {
	// Alignment defaults to painting.AlignmentCenter.
	Alignment painting.AlignmentGeometry
	// TextDirection resolves a directional alignment.
	TextDirection ui.TextDirection
}

// NewRenderSizedOverflowBox creates the box.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.new
func NewRenderSizedOverflowBox(requestedSize ui.Size, child Box, opts ...RenderSizedOverflowBoxOpts) *RenderSizedOverflowBox {
	o := firstOpts(opts)
	r := &RenderSizedOverflowBox{requestedSize: requestedSize}
	InitRenderAligningShiftedBox(&r.RenderAligningShiftedBox, r, o.Alignment, o.TextDirection, child)
	return r
}

// RequestedSize is the size this render box should attempt to be.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.requestedSize
func (r *RenderSizedOverflowBox) RequestedSize() ui.Size { return r.requestedSize }

// SetRequestedSize changes the size and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.requestedSize
func (r *RenderSizedOverflowBox) SetRequestedSize(value ui.Size) {
	if r.requestedSize == value {
		return
	}
	r.requestedSize = value
	r.MarkNeedsLayout()
}

// ComputeMinIntrinsicWidth is the requested width.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.computeMinIntrinsicWidth
func (r *RenderSizedOverflowBox) ComputeMinIntrinsicWidth(float64) float64 {
	return r.requestedSize.Width
}

// ComputeMaxIntrinsicWidth is the requested width.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.computeMaxIntrinsicWidth
func (r *RenderSizedOverflowBox) ComputeMaxIntrinsicWidth(float64) float64 {
	return r.requestedSize.Width
}

// ComputeMinIntrinsicHeight is the requested height.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.computeMinIntrinsicHeight
func (r *RenderSizedOverflowBox) ComputeMinIntrinsicHeight(float64) float64 {
	return r.requestedSize.Height
}

// ComputeMaxIntrinsicHeight is the requested height.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.computeMaxIntrinsicHeight
func (r *RenderSizedOverflowBox) ComputeMaxIntrinsicHeight(float64) float64 {
	return r.requestedSize.Height
}

// ComputeDistanceToActualBaseline is the child's baseline plus its offset.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.computeDistanceToActualBaseline
func (r *RenderSizedOverflowBox) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	if r.hasChild() {
		if v, ok := r.Child().GetDistanceToActualBaseline(baseline).Get(); ok {
			return foundation.Some(v + BoxParentDataOf(r.Child()).Offset.Dy)
		}
	}
	return r.RenderBox.ComputeDistanceToActualBaseline(baseline)
}

// ComputeDryBaseline is the child's baseline plus the aligned offset.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.computeDryBaseline
func (r *RenderSizedOverflowBox) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	child := r.Child()
	result, ok := child.GetDryBaseline(constraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	childSize := child.GetDryLayout(constraints)
	size := r.GetDryLayout(constraints)
	return foundation.Some(result + r.ResolvedAlignment().AlongOffset(sizeDiff(size, childSize)).Dy)
}

// ComputeDryLayout is the requested size, constrained.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.computeDryLayout
func (r *RenderSizedOverflowBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return constraints.Constrain(r.requestedSize)
}

// PerformLayout takes the requested size and lays the child out with the
// parent's constraints.
//
// Flutter: rendering/shifted_box.dart RenderSizedOverflowBox.performLayout
func (r *RenderSizedOverflowBox) PerformLayout() {
	constraints := r.BoxConstraints()
	r.SetSize(constraints.Constrain(r.requestedSize))
	if r.hasChild() {
		r.Child().Layout(constraints, LayoutOpts{ParentUsesSize: true})
		r.AlignChild()
	}
}

// RenderFractionallySizedOverflowBox sizes its child to a fraction of the
// parent's size and lets it overflow.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox
type RenderFractionallySizedOverflowBox struct {
	RenderAligningShiftedBox
	widthFactor, heightFactor foundation.Opt[float64]
}

// RenderFractionallySizedOverflowBoxOpts are the optional parameters of
// NewRenderFractionallySizedOverflowBox.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.new
type RenderFractionallySizedOverflowBoxOpts struct {
	// WidthFactor is the fraction of the parent's width; unset passes it through.
	WidthFactor foundation.Opt[float64]
	// HeightFactor is the fraction of the parent's height; unset passes it
	// through.
	HeightFactor foundation.Opt[float64]
	// Alignment defaults to painting.AlignmentCenter.
	Alignment painting.AlignmentGeometry
	// TextDirection resolves a directional alignment.
	TextDirection ui.TextDirection
}

// NewRenderFractionallySizedOverflowBox creates the box. A negative factor
// panics with SizeFactorNegative in debug builds.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.new
func NewRenderFractionallySizedOverflowBox(child Box, opts ...RenderFractionallySizedOverflowBoxOpts) *RenderFractionallySizedOverflowBox {
	o := firstOpts(opts)
	assertFactor(o.WidthFactor)
	assertFactor(o.HeightFactor)
	r := &RenderFractionallySizedOverflowBox{widthFactor: o.WidthFactor, heightFactor: o.HeightFactor}
	InitRenderAligningShiftedBox(&r.RenderAligningShiftedBox, r, o.Alignment, o.TextDirection, child)
	return r
}

// WidthFactor is the fraction of the parent's width, if set.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.widthFactor
func (r *RenderFractionallySizedOverflowBox) WidthFactor() foundation.Opt[float64] {
	return r.widthFactor
}

// SetWidthFactor changes the factor and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.widthFactor
func (r *RenderFractionallySizedOverflowBox) SetWidthFactor(value foundation.Opt[float64]) {
	assertFactor(value)
	if r.widthFactor == value {
		return
	}
	r.widthFactor = value
	r.MarkNeedsLayout()
}

// HeightFactor is the fraction of the parent's height, if set.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.heightFactor
func (r *RenderFractionallySizedOverflowBox) HeightFactor() foundation.Opt[float64] {
	return r.heightFactor
}

// SetHeightFactor changes the factor and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.heightFactor
func (r *RenderFractionallySizedOverflowBox) SetHeightFactor(value foundation.Opt[float64]) {
	assertFactor(value)
	if r.heightFactor == value {
		return
	}
	r.heightFactor = value
	r.MarkNeedsLayout()
}

func (r *RenderFractionallySizedOverflowBox) innerConstraints(c BoxConstraints) BoxConstraints {
	minW, maxW, minH, maxH := c.MinWidth, c.MaxWidth, c.MinHeight, c.MaxHeight
	if f, ok := r.widthFactor.Get(); ok {
		minW, maxW = maxW*f, maxW*f
	}
	if f, ok := r.heightFactor.Get(); ok {
		minH, maxH = maxH*f, maxH*f
	}
	return BoxConstraints{MinWidth: minW, MaxWidth: maxW, MinHeight: minH, MaxHeight: maxH}
}

// ComputeMinIntrinsicWidth is the child's, scaled by the factors.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.computeMinIntrinsicWidth
func (r *RenderFractionallySizedOverflowBox) ComputeMinIntrinsicWidth(height float64) float64 {
	var result float64
	if r.hasChild() {
		result = r.Child().GetMinIntrinsicWidth(height * r.heightFactor.OrElse(1))
	} else {
		result = r.RenderAligningShiftedBox.ComputeMinIntrinsicWidth(height)
	}
	return result / r.widthFactor.OrElse(1)
}

// ComputeMaxIntrinsicWidth is the child's, scaled by the factors.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.computeMaxIntrinsicWidth
func (r *RenderFractionallySizedOverflowBox) ComputeMaxIntrinsicWidth(height float64) float64 {
	var result float64
	if r.hasChild() {
		result = r.Child().GetMaxIntrinsicWidth(height * r.heightFactor.OrElse(1))
	} else {
		result = r.RenderAligningShiftedBox.ComputeMaxIntrinsicWidth(height)
	}
	return result / r.widthFactor.OrElse(1)
}

// ComputeMinIntrinsicHeight is the child's, scaled by the factors.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.computeMinIntrinsicHeight
func (r *RenderFractionallySizedOverflowBox) ComputeMinIntrinsicHeight(width float64) float64 {
	var result float64
	if r.hasChild() {
		result = r.Child().GetMinIntrinsicHeight(width * r.widthFactor.OrElse(1))
	} else {
		result = r.RenderAligningShiftedBox.ComputeMinIntrinsicHeight(width)
	}
	return result / r.heightFactor.OrElse(1)
}

// ComputeMaxIntrinsicHeight is the child's, scaled by the factors.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.computeMaxIntrinsicHeight
func (r *RenderFractionallySizedOverflowBox) ComputeMaxIntrinsicHeight(width float64) float64 {
	var result float64
	if r.hasChild() {
		result = r.Child().GetMaxIntrinsicHeight(width * r.widthFactor.OrElse(1))
	} else {
		result = r.RenderAligningShiftedBox.ComputeMaxIntrinsicHeight(width)
	}
	return result / r.heightFactor.OrElse(1)
}

// ComputeDryLayout is the child's dry size for the fractional constraints.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.computeDryLayout
func (r *RenderFractionallySizedOverflowBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if r.hasChild() {
		return constraints.Constrain(r.Child().GetDryLayout(r.innerConstraints(constraints)))
	}
	return constraints.Constrain(r.innerConstraints(constraints).Constrain(ui.Size{}))
}

// ComputeDryBaseline is the child's baseline plus the aligned offset.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.computeDryBaseline
func (r *RenderFractionallySizedOverflowBox) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	child := r.Child()
	childConstraints := r.innerConstraints(constraints)
	result, ok := child.GetDryBaseline(childConstraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	childSize := child.GetDryLayout(childConstraints)
	size := r.GetDryLayout(constraints)
	return foundation.Some(result + r.ResolvedAlignment().AlongOffset(sizeDiff(size, childSize)).Dy)
}

// PerformLayout lays the child out with the fractional constraints.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.performLayout
func (r *RenderFractionallySizedOverflowBox) PerformLayout() {
	constraints := r.BoxConstraints()
	if r.hasChild() {
		child := r.Child()
		child.Layout(r.innerConstraints(constraints), LayoutOpts{ParentUsesSize: true})
		r.SetSize(constraints.Constrain(child.Size()))
		r.AlignChild()
		return
	}
	r.SetSize(constraints.Constrain(r.innerConstraints(constraints).Constrain(ui.Size{})))
}

// DebugFillProperties adds the factors.
//
// Flutter: rendering/shifted_box.dart RenderFractionallySizedOverflowBox.debugFillProperties
func (r *RenderFractionallySizedOverflowBox) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderAligningShiftedBox.DebugFillProperties(b)
	b.Add(foundation.NewDoubleProperty("widthFactor", r.widthFactor, foundation.NumberPropertyOpts{IfNull: "pass-through"}))
	b.Add(foundation.NewDoubleProperty("heightFactor", r.heightFactor, foundation.NumberPropertyOpts{IfNull: "pass-through"}))
}

// SingleChildLayoutDelegate customizes the layout of a
// RenderCustomSingleChildLayoutBox.
//
// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate
type SingleChildLayoutDelegate interface {
	foundation.Listenable
	// GetSize is the size of the box for the constraints.
	//
	// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate.getSize
	GetSize(constraints BoxConstraints) ui.Size
	// GetConstraintsForChild is the constraints for the child.
	//
	// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate.getConstraintsForChild
	GetConstraintsForChild(constraints BoxConstraints) BoxConstraints
	// GetPositionForChild is the position of the child in the box.
	//
	// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate.getPositionForChild
	GetPositionForChild(size, childSize ui.Size) ui.Offset
	// ShouldRelayout says whether the box should be laid out again, given the
	// delegate of the same type that it replaces.
	//
	// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate.shouldRelayout
	ShouldRelayout(oldDelegate SingleChildLayoutDelegate) bool
}

// SingleChildLayoutDelegateBase has the listenable and the defaults of a
// SingleChildLayoutDelegate: embed it and implement ShouldRelayout.
//
// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate
type SingleChildLayoutDelegateBase struct {
	relayout foundation.Listenable
}

// NewSingleChildLayoutDelegateBase creates the base; the box is laid out again
// when the relayout listenable (nil for none) notifies.
//
// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate.new
func NewSingleChildLayoutDelegateBase(relayout foundation.Listenable) SingleChildLayoutDelegateBase {
	return SingleChildLayoutDelegateBase{relayout: relayout}
}

// AddListener adds the listener to the relayout listenable.
//
// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate._relayout
func (d SingleChildLayoutDelegateBase) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	if d.relayout == nil {
		return foundation.ListenerHandle{}
	}
	return d.relayout.AddListener(listener)
}

// RemoveListener removes the listener from the relayout listenable.
//
// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate._relayout
func (d SingleChildLayoutDelegateBase) RemoveListener(handle foundation.ListenerHandle) {
	if d.relayout != nil {
		d.relayout.RemoveListener(handle)
	}
}

// GetSize is the biggest size.
//
// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate.getSize
func (SingleChildLayoutDelegateBase) GetSize(constraints BoxConstraints) ui.Size {
	return constraints.Biggest()
}

// GetConstraintsForChild is the constraints of the box.
//
// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate.getConstraintsForChild
func (SingleChildLayoutDelegateBase) GetConstraintsForChild(constraints BoxConstraints) BoxConstraints {
	return constraints
}

// GetPositionForChild is the top left corner.
//
// Flutter: rendering/shifted_box.dart SingleChildLayoutDelegate.getPositionForChild
func (SingleChildLayoutDelegateBase) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	return ui.OffsetZero
}

// RenderCustomSingleChildLayoutBox defers the layout of its single child to a
// delegate.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox
type RenderCustomSingleChildLayoutBox struct {
	RenderShiftedBox
	delegate       SingleChildLayoutDelegate
	delegateHandle foundation.ListenerHandle
}

// NewRenderCustomSingleChildLayoutBox creates the box.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.new
func NewRenderCustomSingleChildLayoutBox(delegate SingleChildLayoutDelegate, child Box) *RenderCustomSingleChildLayoutBox {
	r := &RenderCustomSingleChildLayoutBox{delegate: delegate}
	InitRenderShiftedBox(&r.RenderShiftedBox, r, child)
	return r
}

// Delegate controls the layout of the child.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.delegate
func (r *RenderCustomSingleChildLayoutBox) Delegate() SingleChildLayoutDelegate { return r.delegate }

// SetDelegate changes the delegate; layout is needed if the delegate is of
// another type or says so.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.delegate
func (r *RenderCustomSingleChildLayoutBox) SetDelegate(newDelegate SingleChildLayoutDelegate) {
	if any(r.delegate) == any(newDelegate) {
		return
	}
	old := r.delegate
	if foundation.RuntimeTypeName(newDelegate) != foundation.RuntimeTypeName(old) || newDelegate.ShouldRelayout(old) {
		r.MarkNeedsLayout()
	}
	r.delegate = newDelegate
	if r.Attached() {
		old.RemoveListener(r.delegateHandle)
		r.delegateHandle = newDelegate.AddListener(r.MarkNeedsLayout)
	}
}

// Attach starts listening to the delegate.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.attach
func (r *RenderCustomSingleChildLayoutBox) Attach(owner *PipelineOwner) {
	r.RenderShiftedBox.Attach(owner)
	r.delegateHandle = r.delegate.AddListener(r.MarkNeedsLayout)
}

// Detach stops listening to the delegate.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.detach
func (r *RenderCustomSingleChildLayoutBox) Detach() {
	r.delegate.RemoveListener(r.delegateHandle)
	r.delegateHandle = foundation.ListenerHandle{}
	r.RenderShiftedBox.Detach()
}

func (r *RenderCustomSingleChildLayoutBox) getSize(constraints BoxConstraints) ui.Size {
	return constraints.Constrain(r.delegate.GetSize(constraints))
}

// ComputeMinIntrinsicWidth is the delegate's width for the height, or 0.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.computeMinIntrinsicWidth
func (r *RenderCustomSingleChildLayoutBox) ComputeMinIntrinsicWidth(height float64) float64 {
	w := r.getSize(NewBoxConstraintsTightForFinite(TightForFiniteOpts{Height: foundation.Some(height)})).Width
	if math.IsInf(w, 0) {
		return 0
	}
	return w
}

// ComputeMaxIntrinsicWidth is the delegate's width for the height, or 0.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.computeMaxIntrinsicWidth
func (r *RenderCustomSingleChildLayoutBox) ComputeMaxIntrinsicWidth(height float64) float64 {
	return r.ComputeMinIntrinsicWidth(height)
}

// ComputeMinIntrinsicHeight is the delegate's height for the width, or 0.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.computeMinIntrinsicHeight
func (r *RenderCustomSingleChildLayoutBox) ComputeMinIntrinsicHeight(width float64) float64 {
	h := r.getSize(NewBoxConstraintsTightForFinite(TightForFiniteOpts{Width: foundation.Some(width)})).Height
	if math.IsInf(h, 0) {
		return 0
	}
	return h
}

// ComputeMaxIntrinsicHeight is the delegate's height for the width, or 0.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.computeMaxIntrinsicHeight
func (r *RenderCustomSingleChildLayoutBox) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.ComputeMinIntrinsicHeight(width)
}

// ComputeDryLayout is the delegate's size.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.computeDryLayout
func (r *RenderCustomSingleChildLayoutBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return r.getSize(constraints)
}

// ComputeDryBaseline is the child's baseline plus the delegate's position.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.computeDryBaseline
func (r *RenderCustomSingleChildLayoutBox) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	child := r.Child()
	childConstraints := r.delegate.GetConstraintsForChild(constraints)
	result, ok := child.GetDryBaseline(childConstraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	var childSize ui.Size
	if childConstraints.IsTight() {
		childSize = childConstraints.Smallest()
	} else {
		childSize = child.GetDryLayout(childConstraints)
	}
	return foundation.Some(result + r.delegate.GetPositionForChild(r.getSize(constraints), childSize).Dy)
}

// PerformLayout takes the delegate's size and lays the child out as it says.
//
// Flutter: rendering/shifted_box.dart RenderCustomSingleChildLayoutBox.performLayout
func (r *RenderCustomSingleChildLayoutBox) PerformLayout() {
	constraints := r.BoxConstraints()
	r.SetSize(r.getSize(constraints))
	if !r.hasChild() {
		return
	}
	child := r.Child()
	childConstraints := r.delegate.GetConstraintsForChild(constraints)
	if foundation.DebugMode {
		childConstraints.DebugAssertIsValid(DebugAssertIsValidOpts{IsAppliedConstraint: true})
	}
	child.Layout(childConstraints, LayoutOpts{ParentUsesSize: !childConstraints.IsTight()})
	var childSize ui.Size
	if childConstraints.IsTight() {
		childSize = childConstraints.Smallest()
	} else {
		childSize = child.Size()
	}
	BoxParentDataOf(child).Offset = r.delegate.GetPositionForChild(r.Size(), childSize)
}

// RenderBaseline shifts its child vertically based on the child's baseline.
//
// Flutter: rendering/shifted_box.dart RenderBaseline
type RenderBaseline struct {
	RenderShiftedBox
	baseline     float64
	baselineType ui.TextBaseline
}

// NewRenderBaseline creates the box: the child's baseline of the given type is
// put at the distance from the top.
//
// Flutter: rendering/shifted_box.dart RenderBaseline.new
func NewRenderBaseline(baseline float64, baselineType ui.TextBaseline, child Box) *RenderBaseline {
	r := &RenderBaseline{baseline: baseline, baselineType: baselineType}
	InitRenderShiftedBox(&r.RenderShiftedBox, r, child)
	return r
}

// Baseline is the number of logical pixels from the top of this box at which to
// position the child's baseline.
//
// Flutter: rendering/shifted_box.dart RenderBaseline.baseline
func (r *RenderBaseline) Baseline() float64 { return r.baseline }

// SetBaseline changes the baseline and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderBaseline.baseline
func (r *RenderBaseline) SetBaseline(value float64) {
	if r.baseline == value {
		return
	}
	r.baseline = value
	r.MarkNeedsLayout()
}

// BaselineType is the type of baseline to use.
//
// Flutter: rendering/shifted_box.dart RenderBaseline.baselineType
func (r *RenderBaseline) BaselineType() ui.TextBaseline { return r.baselineType }

// SetBaselineType changes the type and marks the box as needing layout.
//
// Flutter: rendering/shifted_box.dart RenderBaseline.baselineType
func (r *RenderBaseline) SetBaselineType(value ui.TextBaseline) {
	if r.baselineType == value {
		return
	}
	r.baselineType = value
	r.MarkNeedsLayout()
}

func (r *RenderBaseline) computeSizes(constraints BoxConstraints, layoutChild ChildLayouter, getBaseline ChildBaselineGetter) (ui.Size, float64) {
	if !r.hasChild() {
		return constraints.Smallest(), 0
	}
	child := r.Child()
	childConstraints := constraints.Loosen()
	childSize := layoutChild(child, childConstraints)
	childBaseline := getBaseline(child, childConstraints, r.baselineType).OrElse(childSize.Height)
	top := r.baseline - childBaseline
	return constraints.Constrain(ui.Size{Width: childSize.Width, Height: top + childSize.Height}), top
}

// ComputeDryLayout is the size for the child's dry layout.
//
// Flutter: rendering/shifted_box.dart RenderBaseline.computeDryLayout
func (r *RenderBaseline) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	size, _ := r.computeSizes(constraints, ChildLayoutHelper.DryLayoutChild, ChildLayoutHelper.GetDryBaseline)
	return size
}

// ComputeDryBaseline is the requested baseline adjusted by the child's baselines.
//
// Flutter: rendering/shifted_box.dart RenderBaseline.computeDryBaseline
func (r *RenderBaseline) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	loose := constraints.Loosen()
	r1, ok1 := r.Child().GetDryBaseline(loose, baseline).Get()
	r2, ok2 := r.Child().GetDryBaseline(loose, r.baselineType).Get()
	if !ok1 || !ok2 {
		return foundation.None[float64]()
	}
	return foundation.Some(r.baseline + r1 - r2)
}

// PerformLayout lays the child out and shifts it so that its baseline is at the
// requested distance.
//
// Flutter: rendering/shifted_box.dart RenderBaseline.performLayout
func (r *RenderBaseline) PerformLayout() {
	size, top := r.computeSizes(r.BoxConstraints(), ChildLayoutHelper.LayoutChild, ChildLayoutHelper.GetBaseline)
	r.SetSize(size)
	if r.hasChild() {
		BoxParentDataOf(r.Child()).Offset = ui.Offset{Dy: top}
	}
}

// DebugFillProperties adds the baseline and its type.
//
// Flutter: rendering/shifted_box.dart RenderBaseline.debugFillProperties
func (r *RenderBaseline) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderShiftedBox.DebugFillProperties(b)
	b.Add(foundation.NewDoubleProperty("baseline", foundation.Some(r.baseline)))
	b.Add(foundation.NewEnumProperty("baselineType", foundation.Some(r.baselineType)))
}
