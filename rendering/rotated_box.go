// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

const quarterTurnsInRadians = math.Pi / 2

// RenderRotatedBox rotates its child by a whole number of quarter turns.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox
type RenderRotatedBox struct {
	RenderBox
	RenderObjectWithChild[Box]
	quarterTurns   int
	paintTransform *ui.Matrix4
	transformLayer *LayerHandle[*TransformLayer]
}

// NewRenderRotatedBox creates the box.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.new
func NewRenderRotatedBox(quarterTurns int, child Box) *RenderRotatedBox {
	r := &RenderRotatedBox{quarterTurns: quarterTurns, transformLayer: NewLayerHandle[*TransformLayer](nil)}
	InitRenderBox(&r.RenderBox, r)
	InitRenderObjectWithChild(&r.RenderObjectWithChild, r)
	if !isNil(child) {
		r.SetChild(child)
	}
	return r
}

func (r *RenderRotatedBox) hasChild() bool { return !isNil(r.Child()) }

// QuarterTurns is the number of clockwise quarter turns the child is rotated.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.quarterTurns
func (r *RenderRotatedBox) QuarterTurns() int { return r.quarterTurns }

// SetQuarterTurns changes the turns and marks the box as needing layout.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.quarterTurns
func (r *RenderRotatedBox) SetQuarterTurns(value int) {
	if r.quarterTurns == value {
		return
	}
	r.quarterTurns = value
	r.MarkNeedsLayout()
}

func (r *RenderRotatedBox) isVertical() bool { return r.quarterTurns%2 != 0 }

// ComputeMinIntrinsicWidth is the child's height when rotated by an odd number
// of turns.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.computeMinIntrinsicWidth
func (r *RenderRotatedBox) ComputeMinIntrinsicWidth(height float64) float64 {
	if !r.hasChild() {
		return 0
	}
	if r.isVertical() {
		return r.Child().GetMinIntrinsicHeight(height)
	}
	return r.Child().GetMinIntrinsicWidth(height)
}

// ComputeMaxIntrinsicWidth is the child's height when rotated by an odd number
// of turns.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.computeMaxIntrinsicWidth
func (r *RenderRotatedBox) ComputeMaxIntrinsicWidth(height float64) float64 {
	if !r.hasChild() {
		return 0
	}
	if r.isVertical() {
		return r.Child().GetMaxIntrinsicHeight(height)
	}
	return r.Child().GetMaxIntrinsicWidth(height)
}

// ComputeMinIntrinsicHeight is the child's width when rotated by an odd number
// of turns.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.computeMinIntrinsicHeight
func (r *RenderRotatedBox) ComputeMinIntrinsicHeight(width float64) float64 {
	if !r.hasChild() {
		return 0
	}
	if r.isVertical() {
		return r.Child().GetMinIntrinsicWidth(width)
	}
	return r.Child().GetMinIntrinsicHeight(width)
}

// ComputeMaxIntrinsicHeight is the child's width when rotated by an odd number
// of turns.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.computeMaxIntrinsicHeight
func (r *RenderRotatedBox) ComputeMaxIntrinsicHeight(width float64) float64 {
	if !r.hasChild() {
		return 0
	}
	if r.isVertical() {
		return r.Child().GetMaxIntrinsicWidth(width)
	}
	return r.Child().GetMaxIntrinsicHeight(width)
}

// ComputeDryLayout is the child's dry size, swapped when rotated by an odd
// number of turns.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.computeDryLayout
func (r *RenderRotatedBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if !r.hasChild() {
		return constraints.Smallest()
	}
	if r.isVertical() {
		s := r.Child().GetDryLayout(constraints.Flipped())
		return ui.Size{Width: s.Height, Height: s.Width}
	}
	return r.Child().GetDryLayout(constraints)
}

// PerformLayout lays the child out with the flipped constraints when rotated by
// an odd number of turns, and works out the paint transform.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.performLayout
func (r *RenderRotatedBox) PerformLayout() {
	r.paintTransform = nil
	constraints := r.BoxConstraints()
	if !r.hasChild() {
		r.SetSize(constraints.Smallest())
		return
	}
	child := r.Child()
	if r.isVertical() {
		child.Layout(constraints.Flipped(), LayoutOpts{ParentUsesSize: true})
		r.SetSize(ui.Size{Width: child.Size().Height, Height: child.Size().Width})
	} else {
		child.Layout(constraints, LayoutOpts{ParentUsesSize: true})
		r.SetSize(child.Size())
	}
	t := ui.NewMatrix4Identity()
	t.Translate(float32(r.Size().Width/2), float32(r.Size().Height/2), 0)
	turns := r.quarterTurns % 4
	if turns < 0 {
		turns += 4
	}
	t.RotateZ(quarterTurnsInRadians * float64(turns))
	t.Translate(float32(-child.Size().Width/2), float32(-child.Size().Height/2), 0)
	r.paintTransform = t
}

// HitTestChildren hit tests the child in the rotated coordinates.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.hitTestChildren
func (r *RenderRotatedBox) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	if !r.hasChild() || r.paintTransform == nil {
		return false
	}
	child := r.Child()
	return result.AddWithPaintTransform(r.paintTransform, position, func(result *BoxHitTestResult, position ui.Offset) bool {
		return child.HitTest(result, position)
	})
}

func (r *RenderRotatedBox) paintChild(context *PaintingContext, offset ui.Offset) {
	context.PaintChild(r.Child(), offset)
}

// Paint pushes a transform layer around the child.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.paint
func (r *RenderRotatedBox) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() {
		r.transformLayer.SetLayer(nil)
		return
	}
	r.transformLayer.SetLayer(context.PushTransform(r.NeedsCompositing(), offset, r.paintTransform, r.paintChild, r.transformLayer.Layer()))
}

// Dispose lets the transform layer go.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.dispose
func (r *RenderRotatedBox) Dispose() {
	r.transformLayer.SetLayer(nil)
	r.RenderBox.Dispose()
}

// ApplyPaintTransform applies the rotation.
//
// Flutter: rendering/rotated_box.dart RenderRotatedBox.applyPaintTransform
func (r *RenderRotatedBox) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	if r.paintTransform != nil {
		transform.MultiplyBy(r.paintTransform)
	}
	r.RenderBox.ApplyPaintTransform(child, transform)
}
