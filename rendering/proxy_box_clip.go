// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"reflect"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// CustomClipper creates the clip of a clip render object from the size of the
// box.
//
// Flutter: rendering/proxy_box.dart CustomClipper
type CustomClipper[T any] interface {
	foundation.Listenable
	// GetClip returns the clip for the size.
	//
	// Flutter: rendering/proxy_box.dart CustomClipper.getClip
	GetClip(size ui.Size) T
	// GetApproximateClipRect returns a rectangle that approximates the clip,
	// for semantics; the default is the whole box.
	//
	// Flutter: rendering/proxy_box.dart CustomClipper.getApproximateClipRect
	GetApproximateClipRect(size ui.Size) ui.Rect
	// ShouldReclip says whether the clip has to be computed again, given the
	// clipper of the same type that it replaces.
	//
	// Flutter: rendering/proxy_box.dart CustomClipper.shouldReclip
	ShouldReclip(oldClipper CustomClipper[T]) bool
}

// CustomClipperBase has the listenable and the default approximate rectangle of
// a CustomClipper: embed it in a clipper.
//
// Flutter: rendering/proxy_box.dart CustomClipper
type CustomClipperBase[T any] struct {
	reclip foundation.Listenable
}

// NewCustomClipperBase creates the base; the clip is computed again when the
// reclip listenable (nil for none) notifies.
//
// Flutter: rendering/proxy_box.dart CustomClipper.new
func NewCustomClipperBase[T any](reclip foundation.Listenable) CustomClipperBase[T] {
	return CustomClipperBase[T]{reclip: reclip}
}

// AddListener adds the listener to the reclip listenable.
//
// Flutter: rendering/proxy_box.dart CustomClipper.addListener
func (c CustomClipperBase[T]) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	if c.reclip == nil {
		return foundation.ListenerHandle{}
	}
	return c.reclip.AddListener(listener)
}

// RemoveListener removes the listener from the reclip listenable.
//
// Flutter: rendering/proxy_box.dart CustomClipper.removeListener
func (c CustomClipperBase[T]) RemoveListener(handle foundation.ListenerHandle) {
	if c.reclip != nil {
		c.reclip.RemoveListener(handle)
	}
}

// GetApproximateClipRect is the whole box.
//
// Flutter: rendering/proxy_box.dart CustomClipper.getApproximateClipRect
func (c CustomClipperBase[T]) GetApproximateClipRect(size ui.Size) ui.Rect {
	return ui.OffsetZero.And(size)
}

// ShapeBorderClipper clips to the outer path of a shape border.
//
// Flutter: rendering/proxy_box.dart ShapeBorderClipper
type ShapeBorderClipper struct {
	CustomClipperBase[*ui.Path]
	shape         painting.ShapeBorder
	textDirection ui.TextDirection
}

// ShapeBorderClipperOpts are the optional parameters of NewShapeBorderClipper.
//
// Flutter: rendering/proxy_box.dart ShapeBorderClipper.new
type ShapeBorderClipperOpts struct {
	// TextDirection resolves directional shapes.
	TextDirection ui.TextDirection
}

// NewShapeBorderClipper creates a clipper for the shape.
//
// Flutter: rendering/proxy_box.dart ShapeBorderClipper.new
func NewShapeBorderClipper(shape painting.ShapeBorder, opts ...ShapeBorderClipperOpts) *ShapeBorderClipper {
	return &ShapeBorderClipper{shape: shape, textDirection: firstOpts(opts).TextDirection}
}

// Shape is the shape that the clip follows.
//
// Flutter: rendering/proxy_box.dart ShapeBorderClipper.shape
func (c *ShapeBorderClipper) Shape() painting.ShapeBorder { return c.shape }

// GetClip is the outer path of the shape in the rectangle of the box.
//
// Flutter: rendering/proxy_box.dart ShapeBorderClipper.getClip
func (c *ShapeBorderClipper) GetClip(size ui.Size) *ui.Path {
	return c.shape.GetOuterPath(ui.OffsetZero.And(size), c.textDirection)
}

// ShouldReclip is true unless the old clipper has the same shape and direction.
//
// Flutter: rendering/proxy_box.dart ShapeBorderClipper.shouldReclip
func (c *ShapeBorderClipper) ShouldReclip(oldClipper CustomClipper[*ui.Path]) bool {
	old, ok := oldClipper.(*ShapeBorderClipper)
	if !ok {
		return true
	}
	return !reflect.DeepEqual(old.shape, c.shape) || old.textDirection != c.textDirection
}

// String describes the clipper.
//
// Flutter: rendering/proxy_box.dart CustomClipper.toString
func (c *ShapeBorderClipper) String() string { return "ShapeBorderClipper" }

// renderCustomClip is the base of the clip render objects: it keeps the clipper,
// the clip behavior and the cached clip of type T.
//
// Flutter: rendering/proxy_box.dart _RenderCustomClip
type renderCustomClip[T any] struct {
	RenderProxyBox
	clipper         CustomClipper[T]
	clipperHandle   foundation.ListenerHandle
	clip            foundation.Opt[T]
	clipBehavior    ui.Clip
	defaultClipFunc func() T
	debugPaint      *ui.Paint
}

func (c *renderCustomClip[T]) initCustomClip(self Box, child Box, clipper CustomClipper[T], clipBehavior ui.Clip, defaultClip func() T) {
	if clipBehavior == ui.ClipUnspecified {
		clipBehavior = ui.ClipAntiAlias
	}
	c.clipper = clipper
	c.clipBehavior = clipBehavior
	c.defaultClipFunc = defaultClip
	InitRenderProxyBox(&c.RenderProxyBox, self, child)
}

// Clipper creates the clip from the size of the box; nil means the default clip
// (the whole box, or the rounded rectangle of the object).
//
// Flutter: rendering/proxy_box.dart _RenderCustomClip.clipper
func (c *renderCustomClip[T]) Clipper() CustomClipper[T] { return c.clipper }

// SetClipper changes the clipper. The clip is computed again when the clipper
// differs and says it should reclip.
//
// Flutter: rendering/proxy_box.dart _RenderCustomClip.clipper
func (c *renderCustomClip[T]) SetClipper(newClipper CustomClipper[T]) {
	if any(c.clipper) == any(newClipper) {
		return
	}
	oldClipper := c.clipper
	c.clipper = newClipper
	if newClipper == nil || oldClipper == nil ||
		foundation.RuntimeTypeName(newClipper) != foundation.RuntimeTypeName(oldClipper) ||
		newClipper.ShouldReclip(oldClipper) {
		c.markNeedsClip()
	}
	if c.Attached() {
		if oldClipper != nil {
			oldClipper.RemoveListener(c.clipperHandle)
		}
		c.clipperHandle = foundation.ListenerHandle{}
		if newClipper != nil {
			c.clipperHandle = newClipper.AddListener(c.markNeedsClip)
		}
	}
}

// Attach starts listening to the clipper.
//
// Flutter: rendering/proxy_box.dart _RenderCustomClip.attach
func (c *renderCustomClip[T]) Attach(owner *PipelineOwner) {
	c.RenderProxyBox.Attach(owner)
	if c.clipper != nil {
		c.clipperHandle = c.clipper.AddListener(c.markNeedsClip)
	}
}

// Detach stops listening to the clipper.
//
// Flutter: rendering/proxy_box.dart _RenderCustomClip.detach
func (c *renderCustomClip[T]) Detach() {
	if c.clipper != nil {
		c.clipper.RemoveListener(c.clipperHandle)
		c.clipperHandle = foundation.ListenerHandle{}
	}
	c.RenderProxyBox.Detach()
}

func (c *renderCustomClip[T]) markNeedsClip() {
	c.clip = foundation.None[T]()
	c.MarkNeedsPaint()
	c.MarkNeedsSemanticsUpdate()
}

// ClipBehavior is how the clip is applied.
//
// Flutter: rendering/proxy_box.dart _RenderCustomClip.clipBehavior
func (c *renderCustomClip[T]) ClipBehavior() ui.Clip { return c.clipBehavior }

// SetClipBehavior changes the clip behavior and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart _RenderCustomClip.clipBehavior
func (c *renderCustomClip[T]) SetClipBehavior(value ui.Clip) {
	if value != c.clipBehavior {
		c.clipBehavior = value
		c.MarkNeedsPaint()
	}
}

// PerformLayout lays out as a proxy box and drops the clip if the size changed.
//
// Flutter: rendering/proxy_box.dart _RenderCustomClip.performLayout
func (c *renderCustomClip[T]) PerformLayout() {
	oldSize := foundation.None[ui.Size]()
	if c.HasSize() {
		oldSize = foundation.Some(c.Size())
	}
	c.RenderProxyBox.PerformLayout()
	if oldSize != foundation.Some(c.Size()) {
		c.clip = foundation.None[T]()
	}
}

func (c *renderCustomClip[T]) updateClip() T {
	if !c.clip.IsSome() {
		if c.clipper != nil {
			c.clip = foundation.Some(c.clipper.GetClip(c.Size()))
		} else {
			c.clip = foundation.Some(c.defaultClipFunc())
		}
	}
	return c.clip.Value()
}

// DescribeApproximatePaintClip is the approximate rectangle of the clip, unless
// the clip behavior is none.
//
// Flutter: rendering/proxy_box.dart _RenderCustomClip.describeApproximatePaintClip
func (c *renderCustomClip[T]) DescribeApproximatePaintClip(RenderObject) foundation.Opt[ui.Rect] {
	if c.clipBehavior == ui.ClipNone {
		return foundation.None[ui.Rect]()
	}
	if c.clipper != nil {
		return foundation.Some(c.clipper.GetApproximateClipRect(c.Size()))
	}
	return foundation.Some(ui.OffsetZero.And(c.Size()))
}

// debugPaintSize paints the outline of the box only: Flutter also draws a
// scissors glyph, which needs a text engine.
//
// Deviation: the scissors glyph is not drawn (SPEC §12.10).
func (c *renderCustomClip[T]) debugClipPaint() *ui.Paint {
	if c.debugPaint == nil {
		p := ui.NewPaint()
		p.SetShader(ui.NewGradientLinear(ui.OffsetZero, ui.Offset{Dx: 10, Dy: 10}, []ui.Color{
			ui.NewColor(0x00000000), ui.NewColor(0xFFFF00FF), ui.NewColor(0xFFFF00FF), ui.NewColor(0x00000000),
		}, ui.GradientLinearOpts{ColorStops: []float64{0.25, 0.25, 0.75, 0.75}, TileMode: ui.TileModeRepeated}))
		p.SetStrokeWidth(2)
		p.SetStyle(ui.PaintingStyleStroke)
		c.debugPaint = p
	}
	return c.debugPaint
}

// RenderClipRect clips its child using a rectangle.
//
// Flutter: rendering/proxy_box.dart RenderClipRect
type RenderClipRect struct {
	renderCustomClip[ui.Rect]
}

// RenderClipRectOpts are the optional parameters of NewRenderClipRect.
//
// Flutter: rendering/proxy_box.dart _RenderCustomClip.new
type RenderClipRectOpts struct {
	// Clipper creates the clip; nil clips to the box.
	Clipper CustomClipper[ui.Rect]
	// ClipBehavior defaults to ClipAntiAlias.
	ClipBehavior ui.Clip
}

// NewRenderClipRect creates the box.
//
// Flutter: rendering/proxy_box.dart RenderClipRect.new
func NewRenderClipRect(child Box, opts ...RenderClipRectOpts) *RenderClipRect {
	o := firstOpts(opts)
	r := &RenderClipRect{}
	r.initCustomClip(r, child, o.Clipper, o.ClipBehavior, func() ui.Rect { return ui.OffsetZero.And(r.Size()) })
	return r
}

// HitTest only hits inside the clip if there is a clipper.
//
// Flutter: rendering/proxy_box.dart RenderClipRect.hitTest
func (r *RenderClipRect) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	if r.clipper != nil && !r.updateClip().Contains(position) {
		return false
	}
	return r.RenderProxyBox.HitTest(result, position)
}

// Paint pushes a clip rectangle layer.
//
// Flutter: rendering/proxy_box.dart RenderClipRect.paint
func (r *RenderClipRect) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() {
		r.SetLayer(nil)
		return
	}
	if r.clipBehavior == ui.ClipNone {
		context.PaintChild(r.Child(), offset)
		r.SetLayer(nil)
		return
	}
	clip := r.updateClip()
	setObjectLayer(&r.RenderObjectBase, context.PushClipRect(r.NeedsCompositing(), offset, clip, r.RenderProxyBox.Paint,
		PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: layerAs[*ClipRectLayer](r.Layer())}))
}

// DebugPaintSize paints the clip rectangle.
//
// Flutter: rendering/proxy_box.dart RenderClipRect.debugPaintSize
func (r *RenderClipRect) DebugPaintSize(context *PaintingContext, offset ui.Offset) {
	if foundation.DebugMode && r.hasChild() {
		r.RenderProxyBox.DebugPaintSize(context, offset)
		if r.clipBehavior != ui.ClipNone {
			context.Canvas().DrawRect(r.updateClip().Shift(offset), r.debugClipPaint())
		}
	}
}

// RenderClipRRect clips its child using a rounded rectangle.
//
// Flutter: rendering/proxy_box.dart RenderClipRRect
type RenderClipRRect struct {
	renderCustomClip[ui.RRect]
	borderRadius  painting.BorderRadiusGeometry
	textDirection ui.TextDirection
}

// RenderClipRRectOpts are the optional parameters of NewRenderClipRRect.
//
// Flutter: rendering/proxy_box.dart RenderClipRRect.new
type RenderClipRRectOpts struct {
	// BorderRadius defaults to painting.BorderRadiusZero.
	BorderRadius painting.BorderRadiusGeometry
	// Clipper creates the clip; nil uses the border radius.
	Clipper CustomClipper[ui.RRect]
	// ClipBehavior defaults to ClipAntiAlias.
	ClipBehavior ui.Clip
	// TextDirection resolves a directional border radius.
	TextDirection ui.TextDirection
}

// NewRenderClipRRect creates the box.
//
// Flutter: rendering/proxy_box.dart RenderClipRRect.new
func NewRenderClipRRect(child Box, opts ...RenderClipRRectOpts) *RenderClipRRect {
	o := firstOpts(opts)
	if o.BorderRadius == nil {
		o.BorderRadius = painting.BorderRadiusZero
	}
	r := &RenderClipRRect{borderRadius: o.BorderRadius, textDirection: o.TextDirection}
	r.initCustomClip(r, child, o.Clipper, o.ClipBehavior, func() ui.RRect {
		return r.borderRadius.Resolve(r.textDirection).ToRRect(ui.OffsetZero.And(r.Size()))
	})
	return r
}

// BorderRadius is the border radius of the rounded corners.
//
// Flutter: rendering/proxy_box.dart RenderClipRRect.borderRadius
func (r *RenderClipRRect) BorderRadius() painting.BorderRadiusGeometry { return r.borderRadius }

// SetBorderRadius changes the radius and drops the clip.
//
// Flutter: rendering/proxy_box.dart RenderClipRRect.borderRadius
func (r *RenderClipRRect) SetBorderRadius(value painting.BorderRadiusGeometry) {
	if r.borderRadius.Equal(value) {
		return
	}
	r.borderRadius = value
	r.markNeedsClip()
}

// TextDirection resolves a directional border radius.
//
// Flutter: rendering/proxy_box.dart RenderClipRRect.textDirection
func (r *RenderClipRRect) TextDirection() ui.TextDirection { return r.textDirection }

// SetTextDirection changes the direction and drops the clip.
//
// Flutter: rendering/proxy_box.dart RenderClipRRect.textDirection
func (r *RenderClipRRect) SetTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.markNeedsClip()
}

// HitTest only hits inside the clip if there is a clipper.
//
// Flutter: rendering/proxy_box.dart RenderClipRRect.hitTest
func (r *RenderClipRRect) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	if r.clipper != nil && !r.updateClip().Contains(position) {
		return false
	}
	return r.RenderProxyBox.HitTest(result, position)
}

// Paint pushes a clip rounded rectangle layer.
//
// Flutter: rendering/proxy_box.dart RenderClipRRect.paint
func (r *RenderClipRRect) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() {
		r.SetLayer(nil)
		return
	}
	if r.clipBehavior == ui.ClipNone {
		context.PaintChild(r.Child(), offset)
		r.SetLayer(nil)
		return
	}
	clip := r.updateClip()
	setObjectLayer(&r.RenderObjectBase, context.PushClipRRect(r.NeedsCompositing(), offset, clip.OuterRect(), clip, r.RenderProxyBox.Paint,
		PushClipRRectOpts{ClipBehavior: r.clipBehavior, OldLayer: layerAs[*ClipRRectLayer](r.Layer())}))
}

// DebugPaintSize paints the clip rounded rectangle.
//
// Flutter: rendering/proxy_box.dart RenderClipRRect.debugPaintSize
func (r *RenderClipRRect) DebugPaintSize(context *PaintingContext, offset ui.Offset) {
	if foundation.DebugMode && r.hasChild() {
		r.RenderProxyBox.DebugPaintSize(context, offset)
		if r.clipBehavior != ui.ClipNone {
			context.Canvas().DrawRRect(r.updateClip().Shift(offset), r.debugClipPaint())
		}
	}
}

// RenderClipRSuperellipse clips its child using a rounded superellipse.
//
// Flutter: rendering/proxy_box.dart RenderClipRSuperellipse
type RenderClipRSuperellipse struct {
	renderCustomClip[ui.RSuperellipse]
	borderRadius  painting.BorderRadiusGeometry
	textDirection ui.TextDirection
}

// RenderClipRSuperellipseOpts are the optional parameters of
// NewRenderClipRSuperellipse.
//
// Flutter: rendering/proxy_box.dart RenderClipRSuperellipse.new
type RenderClipRSuperellipseOpts struct {
	// BorderRadius defaults to painting.BorderRadiusZero.
	BorderRadius painting.BorderRadiusGeometry
	// Clipper creates the clip; nil uses the border radius.
	Clipper CustomClipper[ui.RSuperellipse]
	// ClipBehavior defaults to ClipAntiAlias.
	ClipBehavior ui.Clip
	// TextDirection resolves a directional border radius.
	TextDirection ui.TextDirection
}

// NewRenderClipRSuperellipse creates the box.
//
// Flutter: rendering/proxy_box.dart RenderClipRSuperellipse.new
func NewRenderClipRSuperellipse(child Box, opts ...RenderClipRSuperellipseOpts) *RenderClipRSuperellipse {
	o := firstOpts(opts)
	if o.BorderRadius == nil {
		o.BorderRadius = painting.BorderRadiusZero
	}
	r := &RenderClipRSuperellipse{borderRadius: o.BorderRadius, textDirection: o.TextDirection}
	r.initCustomClip(r, child, o.Clipper, o.ClipBehavior, func() ui.RSuperellipse {
		return r.borderRadius.Resolve(r.textDirection).ToRSuperellipse(ui.OffsetZero.And(r.Size()))
	})
	return r
}

// BorderRadius is the border radius of the rounded corners.
//
// Flutter: rendering/proxy_box.dart RenderClipRSuperellipse.borderRadius
func (r *RenderClipRSuperellipse) BorderRadius() painting.BorderRadiusGeometry {
	return r.borderRadius
}

// SetBorderRadius changes the radius and drops the clip.
//
// Flutter: rendering/proxy_box.dart RenderClipRSuperellipse.borderRadius
func (r *RenderClipRSuperellipse) SetBorderRadius(value painting.BorderRadiusGeometry) {
	if r.borderRadius.Equal(value) {
		return
	}
	r.borderRadius = value
	r.markNeedsClip()
}

// TextDirection resolves a directional border radius.
//
// Flutter: rendering/proxy_box.dart RenderClipRSuperellipse.textDirection
func (r *RenderClipRSuperellipse) TextDirection() ui.TextDirection { return r.textDirection }

// SetTextDirection changes the direction and drops the clip.
//
// Flutter: rendering/proxy_box.dart RenderClipRSuperellipse.textDirection
func (r *RenderClipRSuperellipse) SetTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.markNeedsClip()
}

// HitTest only hits inside the outer rectangle of the clip if there is a
// clipper.
//
// Flutter: rendering/proxy_box.dart RenderClipRSuperellipse.hitTest
func (r *RenderClipRSuperellipse) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	if r.clipper != nil && !r.updateClip().OuterRect().Contains(position) {
		return false
	}
	return r.RenderProxyBox.HitTest(result, position)
}

// Paint pushes a clip rounded superellipse layer.
//
// Flutter: rendering/proxy_box.dart RenderClipRSuperellipse.paint
func (r *RenderClipRSuperellipse) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() {
		r.SetLayer(nil)
		return
	}
	if r.clipBehavior == ui.ClipNone {
		context.PaintChild(r.Child(), offset)
		r.SetLayer(nil)
		return
	}
	clip := r.updateClip()
	setObjectLayer(&r.RenderObjectBase, context.PushClipRSuperellipse(r.NeedsCompositing(), offset, clip.OuterRect(), clip, r.RenderProxyBox.Paint,
		PushClipRSuperellipseOpts{ClipBehavior: r.clipBehavior, OldLayer: layerAs[*ClipRSuperellipseLayer](r.Layer())}))
}

// DebugPaintSize paints the clip shape.
//
// Flutter: rendering/proxy_box.dart RenderClipRSuperellipse.debugPaintSize
func (r *RenderClipRSuperellipse) DebugPaintSize(context *PaintingContext, offset ui.Offset) {
	if foundation.DebugMode && r.hasChild() {
		r.RenderProxyBox.DebugPaintSize(context, offset)
		if r.clipBehavior != ui.ClipNone {
			context.Canvas().DrawRSuperellipse(r.updateClip().Shift(offset), r.debugClipPaint())
		}
	}
}

// RenderClipOval clips its child using an oval.
//
// Flutter: rendering/proxy_box.dart RenderClipOval
type RenderClipOval struct {
	renderCustomClip[ui.Rect]
	cachedRect foundation.Opt[ui.Rect]
	cachedPath *ui.Path
}

// NewRenderClipOval creates the box.
//
// Flutter: rendering/proxy_box.dart RenderClipOval.new
func NewRenderClipOval(child Box, opts ...RenderClipRectOpts) *RenderClipOval {
	o := firstOpts(opts)
	r := &RenderClipOval{}
	r.initCustomClip(r, child, o.Clipper, o.ClipBehavior, func() ui.Rect { return ui.OffsetZero.And(r.Size()) })
	return r
}

func (r *RenderClipOval) clipPath(rect ui.Rect) *ui.Path {
	if r.cachedRect != foundation.Some(rect) {
		r.cachedRect = foundation.Some(rect)
		r.cachedPath = ui.NewPath()
		r.cachedPath.AddOval(rect)
	}
	return r.cachedPath
}

// HitTest hits inside the oval of the clip.
//
// Flutter: rendering/proxy_box.dart RenderClipOval.hitTest
func (r *RenderClipOval) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	clip := r.updateClip()
	center := clip.Center()
	offset := ui.Offset{Dx: (position.Dx - center.Dx) / clip.Width(), Dy: (position.Dy - center.Dy) / clip.Height()}
	if offset.DistanceSquared() > 0.25 {
		return false
	}
	return r.RenderProxyBox.HitTest(result, position)
}

// Paint pushes a clip path layer for the oval.
//
// Flutter: rendering/proxy_box.dart RenderClipOval.paint
func (r *RenderClipOval) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() {
		r.SetLayer(nil)
		return
	}
	if r.clipBehavior == ui.ClipNone {
		context.PaintChild(r.Child(), offset)
		r.SetLayer(nil)
		return
	}
	clip := r.updateClip()
	setObjectLayer(&r.RenderObjectBase, context.PushClipPath(r.NeedsCompositing(), offset, clip, r.clipPath(clip), r.RenderProxyBox.Paint,
		PushClipPathOpts{ClipBehavior: r.clipBehavior, OldLayer: layerAs[*ClipPathLayer](r.Layer())}))
}

// DebugPaintSize paints the clip oval.
//
// Flutter: rendering/proxy_box.dart RenderClipOval.debugPaintSize
func (r *RenderClipOval) DebugPaintSize(context *PaintingContext, offset ui.Offset) {
	if foundation.DebugMode && r.hasChild() {
		r.RenderProxyBox.DebugPaintSize(context, offset)
		if r.clipBehavior != ui.ClipNone {
			context.Canvas().DrawPath(r.clipPath(r.updateClip()).Shift(offset), r.debugClipPaint())
		}
	}
}

// RenderClipPath clips its child using a path.
//
// Flutter: rendering/proxy_box.dart RenderClipPath
type RenderClipPath struct {
	renderCustomClip[*ui.Path]
}

// RenderClipPathOpts are the optional parameters of NewRenderClipPath.
//
// Flutter: rendering/proxy_box.dart RenderClipPath.new
type RenderClipPathOpts struct {
	// Clipper creates the clip; nil clips to the box.
	Clipper CustomClipper[*ui.Path]
	// ClipBehavior defaults to ClipAntiAlias.
	ClipBehavior ui.Clip
}

// NewRenderClipPath creates the box.
//
// Flutter: rendering/proxy_box.dart RenderClipPath.new
func NewRenderClipPath(child Box, opts ...RenderClipPathOpts) *RenderClipPath {
	o := firstOpts(opts)
	r := &RenderClipPath{}
	r.initCustomClip(r, child, o.Clipper, o.ClipBehavior, func() *ui.Path {
		p := ui.NewPath()
		p.AddRect(ui.OffsetZero.And(r.Size()))
		return p
	})
	return r
}

// HitTest only hits inside the clip path if there is a clipper.
//
// Flutter: rendering/proxy_box.dart RenderClipPath.hitTest
func (r *RenderClipPath) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	if r.clipper != nil && !r.updateClip().Contains(position) {
		return false
	}
	return r.RenderProxyBox.HitTest(result, position)
}

// Paint pushes a clip path layer.
//
// Flutter: rendering/proxy_box.dart RenderClipPath.paint
func (r *RenderClipPath) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() {
		r.SetLayer(nil)
		return
	}
	if r.clipBehavior == ui.ClipNone {
		context.PaintChild(r.Child(), offset)
		r.SetLayer(nil)
		return
	}
	clip := r.updateClip()
	setObjectLayer(&r.RenderObjectBase, context.PushClipPath(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()), clip, r.RenderProxyBox.Paint,
		PushClipPathOpts{ClipBehavior: r.clipBehavior, OldLayer: layerAs[*ClipPathLayer](r.Layer())}))
}

// DebugPaintSize paints the clip path.
//
// Flutter: rendering/proxy_box.dart RenderClipPath.debugPaintSize
func (r *RenderClipPath) DebugPaintSize(context *PaintingContext, offset ui.Offset) {
	if foundation.DebugMode && r.hasChild() {
		r.RenderProxyBox.DebugPaintSize(context, offset)
		if r.clipBehavior != ui.ClipNone {
			context.Canvas().DrawPath(r.updateClip().Shift(offset), r.debugClipPaint())
		}
	}
}

// renderPhysicalModelBase is the base of the physical model render objects: it
// adds the elevation, the color and the shadow color to a clip.
//
// Flutter: rendering/proxy_box.dart _RenderPhysicalModelBase
type renderPhysicalModelBase[T any] struct {
	renderCustomClip[T]
	elevation   float64
	color       ui.Color
	shadowColor ui.Color
}

func (p *renderPhysicalModelBase[T]) initPhysicalModel(self Box, child Box, elevation float64, color, shadowColor ui.Color, clipBehavior ui.Clip, clipper CustomClipper[T], defaultClip func() T) {
	if clipBehavior == ui.ClipUnspecified {
		clipBehavior = ui.ClipNone
	}
	if foundation.DebugMode && elevation < 0 {
		panic(errors.New("ElevationNegative", "the elevation must not be negative", "elevation", elevation))
	}
	p.elevation, p.color, p.shadowColor = elevation, color, shadowColor
	p.initCustomClip(self, child, clipper, clipBehavior, defaultClip)
}

// Elevation is the z-coordinate of the box relative to the parent.
//
// Flutter: rendering/proxy_box.dart _RenderPhysicalModelBase.elevation
func (p *renderPhysicalModelBase[T]) Elevation() float64 { return p.elevation }

// SetElevation changes the elevation.
//
// Flutter: rendering/proxy_box.dart _RenderPhysicalModelBase.elevation
func (p *renderPhysicalModelBase[T]) SetElevation(value float64) {
	if foundation.DebugMode && value < 0 {
		panic(errors.New("ElevationNegative", "the elevation must not be negative", "elevation", value))
	}
	if p.elevation == value {
		return
	}
	didNeedCompositing := p.AlwaysNeedsCompositing()
	p.elevation = value
	if didNeedCompositing != p.AlwaysNeedsCompositing() {
		p.MarkNeedsCompositingBitsUpdate()
	}
	p.MarkNeedsPaint()
}

// ShadowColor is the color of the shadow.
//
// Flutter: rendering/proxy_box.dart _RenderPhysicalModelBase.shadowColor
func (p *renderPhysicalModelBase[T]) ShadowColor() ui.Color { return p.shadowColor }

// SetShadowColor changes the shadow color.
//
// Flutter: rendering/proxy_box.dart _RenderPhysicalModelBase.shadowColor
func (p *renderPhysicalModelBase[T]) SetShadowColor(value ui.Color) {
	if p.shadowColor == value {
		return
	}
	p.shadowColor = value
	p.MarkNeedsPaint()
}

// Color is the background color.
//
// Flutter: rendering/proxy_box.dart _RenderPhysicalModelBase.color
func (p *renderPhysicalModelBase[T]) Color() ui.Color { return p.color }

// SetColor changes the background color.
//
// Flutter: rendering/proxy_box.dart _RenderPhysicalModelBase.color
func (p *renderPhysicalModelBase[T]) SetColor(value ui.Color) {
	if p.color == value {
		return
	}
	p.color = value
	p.MarkNeedsPaint()
}

// DebugFillProperties adds the elevation and the colors.
//
// Flutter: rendering/proxy_box.dart _RenderPhysicalModelBase.debugFillProperties
func (p *renderPhysicalModelBase[T]) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	p.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDoubleProperty("elevation", foundation.Some(p.elevation)))
	b.Add(foundation.NewDiagnosticsProperty("color", p.color))
	// Deviation: Flutter reports the background color under shadowColor too.
	b.Add(foundation.NewDiagnosticsProperty("shadowColor", p.shadowColor))
}

// paintModel draws the shadow and the background, then the child through the
// clip layer. It is the common part of the two physical model paint methods.
func (p *renderPhysicalModelBase[T]) paintBackground(canvas *ui.Canvas, path *ui.Path, drawFill func(paint *ui.Paint)) (usesSaveLayer bool) {
	if p.elevation != 0 {
		canvas.DrawShadow(path, p.shadowColor, p.elevation, p.color.Alpha() != 0xFF)
	}
	usesSaveLayer = p.clipBehavior == ui.ClipAntiAliasWithSaveLayer
	if !usesSaveLayer {
		paint := ui.NewPaint()
		paint.SetColor(p.color)
		drawFill(paint)
	}
	return usesSaveLayer
}

// RenderPhysicalModel clips its child to a rectangle or a circle and draws a
// shadow below it.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalModel
type RenderPhysicalModel struct {
	renderPhysicalModelBase[ui.RRect]
	shape        painting.BoxShape
	borderRadius foundation.Opt[painting.BorderRadius]
}

// RenderPhysicalModelOpts are the optional parameters of NewRenderPhysicalModel.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalModel.new
type RenderPhysicalModelOpts struct {
	// Shape defaults to BoxShapeRectangle.
	Shape painting.BoxShape
	// ClipBehavior defaults to ClipNone.
	ClipBehavior ui.Clip
	// BorderRadius is the radius of a rectangle; none by default.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// Elevation defaults to 0.
	Elevation float64
	// ShadowColor defaults to opaque black.
	ShadowColor foundation.Opt[ui.Color]
}

// NewRenderPhysicalModel creates the box.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalModel.new
func NewRenderPhysicalModel(color ui.Color, child Box, opts ...RenderPhysicalModelOpts) *RenderPhysicalModel {
	o := firstOpts(opts)
	if o.Shape == painting.BoxShapeUnspecified {
		o.Shape = painting.BoxShapeRectangle
	}
	r := &RenderPhysicalModel{shape: o.Shape, borderRadius: o.BorderRadius}
	r.initPhysicalModel(r, child, o.Elevation, color, o.ShadowColor.OrElse(ui.NewColor(0xFF000000)), o.ClipBehavior, nil, func() ui.RRect {
		rect := ui.OffsetZero.And(r.Size())
		if r.shape == painting.BoxShapeCircle {
			return ui.NewRRectFromRectXY(rect, rect.Width()/2, rect.Height()/2)
		}
		return r.borderRadius.OrElse(painting.BorderRadiusZero).ToRRect(rect)
	})
	return r
}

// Shape is the shape of the clip.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalModel.shape
func (r *RenderPhysicalModel) Shape() painting.BoxShape { return r.shape }

// SetShape changes the shape and drops the clip.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalModel.shape
func (r *RenderPhysicalModel) SetShape(value painting.BoxShape) {
	if r.shape == value {
		return
	}
	r.shape = value
	r.markNeedsClip()
}

// BorderRadius is the radius of the corners of a rectangle.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalModel.borderRadius
func (r *RenderPhysicalModel) BorderRadius() foundation.Opt[painting.BorderRadius] {
	return r.borderRadius
}

// SetBorderRadius changes the radius and drops the clip.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalModel.borderRadius
func (r *RenderPhysicalModel) SetBorderRadius(value foundation.Opt[painting.BorderRadius]) {
	if r.borderRadius == value {
		return
	}
	r.borderRadius = value
	r.markNeedsClip()
}

// HitTest only hits inside the clip if there is a clipper.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalModel.hitTest
func (r *RenderPhysicalModel) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	if r.clipper != nil && !r.updateClip().Contains(position) {
		return false
	}
	return r.RenderProxyBox.HitTest(result, position)
}

// Paint draws the shadow and the background and pushes a clip layer.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalModel.paint
func (r *RenderPhysicalModel) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() {
		r.SetLayer(nil)
		return
	}
	clip := r.updateClip()
	offsetRRect := clip.Shift(offset)
	path := ui.NewPath()
	path.AddRRect(offsetRRect)
	canvas := context.Canvas()
	usesSaveLayer := r.paintBackground(canvas, path, func(paint *ui.Paint) { canvas.DrawRRect(offsetRRect, paint) })
	color := r.color
	setObjectLayer(&r.RenderObjectBase, context.PushClipRRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()), clip,
		func(context *PaintingContext, offset ui.Offset) {
			if usesSaveLayer {
				paint := ui.NewPaint()
				paint.SetColor(color)
				context.Canvas().DrawPaint(paint)
			}
			r.RenderProxyBox.Paint(context, offset)
		}, PushClipRRectOpts{ClipBehavior: r.clipBehavior, OldLayer: layerAs[*ClipRRectLayer](r.Layer())}))
}

// DebugFillProperties adds the shape and the border radius.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalModel.debugFillProperties
func (r *RenderPhysicalModel) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.renderPhysicalModelBase.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("shape", r.shape))
	b.Add(foundation.NewDiagnosticsPropertyOpt("borderRadius", r.borderRadius))
}

// RenderPhysicalShape clips its child to a custom shape and draws a shadow below
// it.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalShape
type RenderPhysicalShape struct {
	renderPhysicalModelBase[*ui.Path]
}

// NewRenderPhysicalShape creates the box with the clipper that makes its shape.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalShape.new
func NewRenderPhysicalShape(clipper CustomClipper[*ui.Path], color ui.Color, child Box, opts ...RenderPhysicalModelOpts) *RenderPhysicalShape {
	o := firstOpts(opts)
	r := &RenderPhysicalShape{}
	r.initPhysicalModel(r, child, o.Elevation, color, o.ShadowColor.OrElse(ui.NewColor(0xFF000000)), o.ClipBehavior, clipper, func() *ui.Path {
		p := ui.NewPath()
		p.AddRect(ui.OffsetZero.And(r.Size()))
		return p
	})
	return r
}

// HitTest only hits inside the shape if there is a clipper.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalShape.hitTest
func (r *RenderPhysicalShape) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	if r.clipper != nil && !r.updateClip().Contains(position) {
		return false
	}
	return r.RenderProxyBox.HitTest(result, position)
}

// Paint draws the shadow and the background and pushes a clip path layer.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalShape.paint
func (r *RenderPhysicalShape) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() {
		r.SetLayer(nil)
		return
	}
	clip := r.updateClip()
	offsetPath := clip.Shift(offset)
	canvas := context.Canvas()
	usesSaveLayer := r.paintBackground(canvas, offsetPath, func(paint *ui.Paint) { canvas.DrawPath(offsetPath, paint) })
	color := r.color
	setObjectLayer(&r.RenderObjectBase, context.PushClipPath(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()), clip,
		func(context *PaintingContext, offset ui.Offset) {
			if usesSaveLayer {
				paint := ui.NewPaint()
				paint.SetColor(color)
				context.Canvas().DrawPaint(paint)
			}
			r.RenderProxyBox.Paint(context, offset)
		}, PushClipPathOpts{ClipBehavior: r.clipBehavior, OldLayer: layerAs[*ClipPathLayer](r.Layer())}))
}

// DebugFillProperties adds the clipper.
//
// Flutter: rendering/proxy_box.dart RenderPhysicalShape.debugFillProperties
func (r *RenderPhysicalShape) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.renderPhysicalModelBase.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("clipper", r.clipper))
}
