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
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// CustomPainter is the interface that custom painters implement: it paints on a
// canvas of a given size and tells when it needs to paint again.
//
// The semantics of a painter are optional interfaces: SemanticsPainter and
// SemanticsRebuildPainter.
//
// Flutter: rendering/custom_paint.dart CustomPainter
type CustomPainter interface {
	foundation.Listenable
	// Paint paints on the canvas, in the box of the given size at the origin.
	//
	// Flutter: rendering/custom_paint.dart CustomPainter.paint
	Paint(canvas *ui.Canvas, size ui.Size)
	// ShouldRepaint says whether a new painter paints differently from the old
	// one of the same type.
	//
	// Flutter: rendering/custom_paint.dart CustomPainter.shouldRepaint
	ShouldRepaint(oldDelegate CustomPainter) bool
	// HitTest says whether the position is on the painting: none defers to the
	// default (a background painter is hit, a foreground one is not).
	//
	// Flutter: rendering/custom_paint.dart CustomPainter.hitTest
	HitTest(position ui.Offset) foundation.Opt[bool]
}

// CustomPainterBase has the listenable and the defaults of a CustomPainter: embed
// it and implement Paint and ShouldRepaint.
//
// Flutter: rendering/custom_paint.dart CustomPainter
type CustomPainterBase struct {
	repaint foundation.Listenable
}

// NewCustomPainterBase creates the base; the box repaints when the repaint
// listenable (nil for none) notifies.
//
// Flutter: rendering/custom_paint.dart CustomPainter.new
func NewCustomPainterBase(repaint foundation.Listenable) CustomPainterBase {
	return CustomPainterBase{repaint: repaint}
}

// AddListener adds the listener to the repaint listenable.
//
// Flutter: rendering/custom_paint.dart CustomPainter.addListener
func (p CustomPainterBase) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	if p.repaint == nil {
		return foundation.ListenerHandle{}
	}
	return p.repaint.AddListener(listener)
}

// RemoveListener removes the listener from the repaint listenable.
//
// Flutter: rendering/custom_paint.dart CustomPainter.removeListener
func (p CustomPainterBase) RemoveListener(handle foundation.ListenerHandle) {
	if p.repaint != nil {
		p.repaint.RemoveListener(handle)
	}
}

// HitTest defers to the default.
//
// Flutter: rendering/custom_paint.dart CustomPainter.hitTest
func (CustomPainterBase) HitTest(ui.Offset) foundation.Opt[bool] { return foundation.None[bool]() }

// RenderCustomPaint provides a canvas on which to draw during the paint phase.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint
type RenderCustomPaint struct {
	RenderProxyBox
	painter           CustomPainter
	painterHandle     foundation.ListenerHandle
	foregroundPainter CustomPainter
	foregroundHandle  foundation.ListenerHandle
	preferredSize     ui.Size
	IsComplex         bool
	WillChange        bool

	backgroundSemanticsBuilder SemanticsBuilderCallback
	foregroundSemanticsBuilder SemanticsBuilderCallback
	backgroundSemanticsNodes   []*semantics.SemanticsNode
	foregroundSemanticsNodes   []*semantics.SemanticsNode
}

// RenderCustomPaintOpts are the optional parameters of NewRenderCustomPaint.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.new
type RenderCustomPaintOpts struct {
	// Painter paints before the child.
	Painter CustomPainter
	// ForegroundPainter paints after the child.
	ForegroundPainter CustomPainter
	// PreferredSize is the size of the box when it has no child.
	PreferredSize ui.Size
	// IsComplex hints that the painting is expensive to rasterize.
	IsComplex bool
	// WillChange hints that the painting will change in the next frame.
	WillChange bool
}

// NewRenderCustomPaint creates the box.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.new
func NewRenderCustomPaint(child Box, opts ...RenderCustomPaintOpts) *RenderCustomPaint {
	o := firstOpts(opts)
	r := &RenderCustomPaint{painter: o.Painter, foregroundPainter: o.ForegroundPainter, preferredSize: o.PreferredSize,
		IsComplex: o.IsComplex, WillChange: o.WillChange}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Painter paints before the child.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.painter
func (r *RenderCustomPaint) Painter() CustomPainter { return r.painter }

// SetPainter changes the painter; the box repaints if the painter should.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.painter
func (r *RenderCustomPaint) SetPainter(value CustomPainter) {
	if any(r.painter) == any(value) {
		return
	}
	old := r.painter
	r.painter = value
	r.didUpdatePainter(value, old, &r.painterHandle)
}

// ForegroundPainter paints after the child.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.foregroundPainter
func (r *RenderCustomPaint) ForegroundPainter() CustomPainter { return r.foregroundPainter }

// SetForegroundPainter changes the painter; the box repaints if the painter
// should.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.foregroundPainter
func (r *RenderCustomPaint) SetForegroundPainter(value CustomPainter) {
	if any(r.foregroundPainter) == any(value) {
		return
	}
	old := r.foregroundPainter
	r.foregroundPainter = value
	r.didUpdatePainter(value, old, &r.foregroundHandle)
}

func (r *RenderCustomPaint) didUpdatePainter(newPainter, oldPainter CustomPainter, handle *foundation.ListenerHandle) {
	if newPainter == nil || oldPainter == nil ||
		foundation.RuntimeTypeName(newPainter) != foundation.RuntimeTypeName(oldPainter) ||
		newPainter.ShouldRepaint(oldPainter) {
		r.MarkNeedsPaint()
	}
	if r.Attached() {
		if oldPainter != nil {
			oldPainter.RemoveListener(*handle)
		}
		*handle = foundation.ListenerHandle{}
		if newPainter != nil {
			*handle = newPainter.AddListener(r.MarkNeedsPaint)
		}
	}
	if newPainter == nil {
		if r.Attached() {
			r.MarkNeedsSemanticsUpdate()
		}
	} else if oldPainter == nil || foundation.RuntimeTypeName(newPainter) != foundation.RuntimeTypeName(oldPainter) ||
		shouldRebuildPainterSemantics(newPainter, oldPainter) {
		r.MarkNeedsSemanticsUpdate()
	}
}

// PreferredSize is the size that this render object would like to be when it has
// no child.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.preferredSize
func (r *RenderCustomPaint) PreferredSize() ui.Size { return r.preferredSize }

// SetPreferredSize changes the size and marks the box as needing layout.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.preferredSize
func (r *RenderCustomPaint) SetPreferredSize(value ui.Size) {
	if r.preferredSize == value {
		return
	}
	r.preferredSize = value
	r.MarkNeedsLayout()
}

func finiteOrZero(v float64) float64 {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0
	}
	return v
}

// ComputeMinIntrinsicWidth is the child's, or the preferred width if finite.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.computeMinIntrinsicWidth
func (r *RenderCustomPaint) ComputeMinIntrinsicWidth(height float64) float64 {
	if !r.hasChild() {
		return finiteOrZero(r.preferredSize.Width)
	}
	return r.RenderProxyBox.ComputeMinIntrinsicWidth(height)
}

// ComputeMaxIntrinsicWidth is the child's, or the preferred width if finite.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.computeMaxIntrinsicWidth
func (r *RenderCustomPaint) ComputeMaxIntrinsicWidth(height float64) float64 {
	if !r.hasChild() {
		return finiteOrZero(r.preferredSize.Width)
	}
	return r.RenderProxyBox.ComputeMaxIntrinsicWidth(height)
}

// ComputeMinIntrinsicHeight is the child's, or the preferred height if finite.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.computeMinIntrinsicHeight
func (r *RenderCustomPaint) ComputeMinIntrinsicHeight(width float64) float64 {
	if !r.hasChild() {
		return finiteOrZero(r.preferredSize.Height)
	}
	return r.RenderProxyBox.ComputeMinIntrinsicHeight(width)
}

// ComputeMaxIntrinsicHeight is the child's, or the preferred height if finite.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.computeMaxIntrinsicHeight
func (r *RenderCustomPaint) ComputeMaxIntrinsicHeight(width float64) float64 {
	if !r.hasChild() {
		return finiteOrZero(r.preferredSize.Height)
	}
	return r.RenderProxyBox.ComputeMaxIntrinsicHeight(width)
}

// Attach starts listening to the painters.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.attach
func (r *RenderCustomPaint) Attach(owner *PipelineOwner) {
	r.RenderProxyBox.Attach(owner)
	if r.painter != nil {
		r.painterHandle = r.painter.AddListener(r.MarkNeedsPaint)
	}
	if r.foregroundPainter != nil {
		r.foregroundHandle = r.foregroundPainter.AddListener(r.MarkNeedsPaint)
	}
}

// Detach stops listening to the painters.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.detach
func (r *RenderCustomPaint) Detach() {
	if r.painter != nil {
		r.painter.RemoveListener(r.painterHandle)
		r.painterHandle = foundation.ListenerHandle{}
	}
	if r.foregroundPainter != nil {
		r.foregroundPainter.RemoveListener(r.foregroundHandle)
		r.foregroundHandle = foundation.ListenerHandle{}
	}
	r.RenderProxyBox.Detach()
}

// HitTestChildren is true if the foreground painter says the position is hit,
// otherwise the child's.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.hitTestChildren
func (r *RenderCustomPaint) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	if r.foregroundPainter != nil && r.foregroundPainter.HitTest(position).OrElse(false) {
		return true
	}
	return r.RenderProxyBox.HitTestChildren(result, position)
}

// HitTestSelf is whether the background painter says the position is hit.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.hitTestSelf
func (r *RenderCustomPaint) HitTestSelf(position ui.Offset) bool {
	return r.painter != nil && r.painter.HitTest(position).OrElse(true)
}

// PerformLayout lays out as a proxy box.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.performLayout
func (r *RenderCustomPaint) PerformLayout() {
	r.RenderProxyBox.PerformLayout()
	r.MarkNeedsSemanticsUpdate()
}

// ComputeSizeForNoChild is the preferred size, constrained.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.computeSizeForNoChild
func (r *RenderCustomPaint) ComputeSizeForNoChild(constraints BoxConstraints) ui.Size {
	return constraints.Constrain(r.preferredSize)
}

func (r *RenderCustomPaint) paintWithPainter(canvas *ui.Canvas, offset ui.Offset, painter CustomPainter) {
	canvas.Save()
	previous := canvas.GetSaveCount()
	if offset != ui.OffsetZero {
		canvas.Translate(offset.Dx, offset.Dy)
	}
	painter.Paint(canvas, r.Size())
	if foundation.DebugMode {
		now := canvas.GetSaveCount()
		name := foundation.RuntimeTypeName(painter)
		if now > previous {
			panic(errors.New("CustomPainterSaveUnbalanced", "The "+name+" custom painter called canvas.save() or canvas.saveLayer() at least "+
				strconv.Itoa(now-previous)+" more time(s) than it called canvas.restore(). This leaves the canvas in an inconsistent "+
				"state and will probably result in a broken display. You must pair each call to save()/saveLayer() with a later matching call to restore().",
				"painter", name))
		}
		if now < previous {
			panic(errors.New("CustomPainterRestoreUnbalanced", "The "+name+" custom painter called canvas.restore() "+
				strconv.Itoa(previous-now)+" more time(s) than it called canvas.save() or canvas.saveLayer(). This leaves the canvas in "+
				"an inconsistent state and will result in a broken display. You should only call restore() if you first called save() or saveLayer().",
				"painter", name))
		}
	}
	canvas.Restore()
}

func (r *RenderCustomPaint) setRasterCacheHints(context *PaintingContext) {
	if r.IsComplex {
		context.SetIsComplexHint()
	}
	if r.WillChange {
		context.SetWillChangeHint()
	}
}

// Paint paints the background painter, the child and the foreground painter.
//
// Flutter: rendering/custom_paint.dart RenderCustomPaint.paint
func (r *RenderCustomPaint) Paint(context *PaintingContext, offset ui.Offset) {
	if r.painter != nil {
		r.paintWithPainter(context.Canvas(), offset, r.painter)
		r.setRasterCacheHints(context)
	}
	r.RenderProxyBox.Paint(context, offset)
	if r.foregroundPainter != nil {
		r.paintWithPainter(context.Canvas(), offset, r.foregroundPainter)
		r.setRasterCacheHints(context)
	}
}
