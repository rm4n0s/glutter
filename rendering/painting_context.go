// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// PaintingContextCallback paints into a context at an offset.
//
// Flutter: rendering/object.dart PaintingContextCallback
type PaintingContextCallback func(context *PaintingContext, offset ui.Offset)

// PaintingContext is a place to record painting commands: it hands out a
// canvas, and builds the layer tree around the pictures.
//
// Deviation: the constructor is not protected, and CreateChildContext cannot be
// overridden; the picture recorder and the canvas are made by the ui package
// directly, not by the renderer binding (SPEC §12.10).
//
// Flutter: rendering/object.dart PaintingContext
type PaintingContext struct {
	containerLayer  *ContainerLayer
	estimatedBounds ui.Rect

	currentLayer *PictureLayer
	recorder     *ui.PictureRecorder
	canvas       *ui.Canvas
}

// NewPaintingContext creates a context that adds its layers to the container
// layer; estimatedBounds is where it expects to paint.
//
// Flutter: rendering/object.dart PaintingContext.new
func NewPaintingContext(containerLayer *ContainerLayer, estimatedBounds ui.Rect) *PaintingContext {
	return &PaintingContext{containerLayer: containerLayer, estimatedBounds: estimatedBounds}
}

// EstimatedBounds is the area that the context expects to paint in.
//
// Flutter: rendering/object.dart PaintingContext.estimatedBounds
func (c *PaintingContext) EstimatedBounds() ui.Rect { return c.estimatedBounds }

// RepaintCompositedChild repaints a repaint boundary into its own layer. It
// is called by the pipeline owner, and by the context when a boundary has to be
// painted as part of its parent's painting (debugAlsoPaintedParent).
//
// Flutter: rendering/object.dart PaintingContext.repaintCompositedChild
func RepaintCompositedChild(child RenderObject, debugAlsoPaintedParent bool) {
	if foundation.DebugMode && !child.AsRenderObject().needsPaint {
		panic(errors.New("RepaintNotNeeded", "a render object that does not need paint was repainted",
			"object", foundation.DescribeIdentity(child)))
	}
	repaintCompositedChild(child, debugAlsoPaintedParent, nil)
}

func repaintCompositedChild(child RenderObject, debugAlsoPaintedParent bool, childContext *PaintingContext) {
	cb := child.AsRenderObject()
	if foundation.DebugMode && !child.IsRepaintBoundary() {
		panic(errors.New("RepaintNotBoundary", "a render object that is not a repaint boundary was repainted into its own layer",
			"object", foundation.DescribeIdentity(child)))
	}
	if foundation.DebugMode {
		child.DebugRegisterRepaintBoundaryPaint(debugAlsoPaintedParent, true)
	}
	childLayer, _ := cb.layerHandle.Layer().(OffsetLayerLike)
	if childLayer == nil {
		layer := child.UpdateCompositedLayer(nil)
		cb.layerHandle.SetLayer(layer)
		childLayer = layer
	} else {
		if foundation.DebugMode && !debugAlsoPaintedParent && !childLayer.AsLayer().Attached() {
			panic(errors.New("RepaintLayerDetached", "a repaint boundary with a detached layer was repainted",
				"object", foundation.DescribeIdentity(child)))
		}
		var oldOffset ui.Offset
		if foundation.DebugMode {
			oldOffset = childLayer.AsOffsetLayer().Offset()
		}
		childLayer.AsContainerLayer().RemoveAllChildren()
		updated := child.UpdateCompositedLayer(childLayer)
		debugCheckUpdatedLayer(child, childLayer, updated, oldOffset)
	}
	cb.needsCompositedLayerUpdate = false
	if foundation.DebugMode {
		childLayer.AsLayer().DebugCreator = creatorOr(cb.DebugCreator, child)
	}
	if childContext == nil {
		childContext = NewPaintingContext(childLayer.AsContainerLayer(), child.PaintBounds())
	}
	cb.paintWithContext(childContext, ui.OffsetZero)
	childContext.StopRecordingIfNeeded()
}

func creatorOr(creator any, fallback any) any {
	if creator != nil {
		return creator
	}
	return fallback
}

func debugCheckUpdatedLayer(child RenderObject, old, updated OffsetLayerLike, oldOffset ui.Offset) {
	if !foundation.DebugMode {
		return
	}
	if updated != old {
		panic(errors.New("UpdateCompositedLayerNewInstance",
			"UpdateCompositedLayer created a new layer instance instead of reusing the existing layer; "+
				"it must return the old layer when it is given one",
			"object", foundation.DescribeIdentity(child), "newLayer", foundation.DescribeIdentity(updated),
			"oldLayer", foundation.DescribeIdentity(old)))
	}
	if oldOffset != updated.AsOffsetLayer().Offset() {
		panic(errors.New("UpdateCompositedLayerOffsetChanged",
			"UpdateCompositedLayer must not change the offset of the layer",
			"object", foundation.DescribeIdentity(child)))
	}
}

// UpdateLayerProperties updates the properties of the layer of a repaint
// boundary that does not need to repaint (only the layer's properties changed).
//
// Flutter: rendering/object.dart PaintingContext.updateLayerProperties
func UpdateLayerProperties(child RenderObject) {
	cb := child.AsRenderObject()
	if foundation.DebugMode && (!child.IsRepaintBoundary() || !cb.wasRepaintBoundary || cb.needsPaint || isNil(cb.layerHandle.Layer())) {
		panic(errors.New("UpdateLayerPropertiesInvalid", "the layer properties of this render object can not be updated",
			"object", foundation.DescribeIdentity(child)))
	}
	childLayer := cb.layerHandle.Layer().(OffsetLayerLike)
	var oldOffset ui.Offset
	if foundation.DebugMode {
		oldOffset = childLayer.AsOffsetLayer().Offset()
	}
	updated := child.UpdateCompositedLayer(childLayer)
	debugCheckUpdatedLayer(child, childLayer, updated, oldOffset)
	cb.needsCompositedLayerUpdate = false
}

// DebugInstrumentRepaintCompositedChild repaints a boundary into a custom
// context (debug builds only).
//
// Flutter: rendering/object.dart PaintingContext.debugInstrumentRepaintCompositedChild
func DebugInstrumentRepaintCompositedChild(child RenderObject, debugAlsoPaintedParent bool, customContext *PaintingContext) {
	if foundation.DebugMode {
		repaintCompositedChild(child, debugAlsoPaintedParent, customContext)
	}
}

// PaintChild paints a child render object at an offset: into this context, or
// into the child's own layer when it is a repaint boundary.
//
// Flutter: rendering/object.dart PaintingContext.paintChild
func (c *PaintingContext) PaintChild(child RenderObject, offset ui.Offset) {
	cb := child.AsRenderObject()
	if foundation.DebugMode {
		if o := cb.DebugOptions(); o.OnProfilePaint != nil {
			o.OnProfilePaint(child)
		}
	}
	switch {
	case child.IsRepaintBoundary():
		c.StopRecordingIfNeeded()
		c.compositeChild(child, offset)
	case cb.wasRepaintBoundary:
		cb.layerHandle.SetLayer(nil)
		cb.paintWithContext(c, offset)
	default:
		cb.paintWithContext(c, offset)
	}
}

func (c *PaintingContext) compositeChild(child RenderObject, offset ui.Offset) {
	cb := child.AsRenderObject()
	if foundation.DebugMode && c.isRecording() {
		panic(errors.New("PaintingContextRecording", "a child was composited while the context was still recording"))
	}
	if cb.needsPaint || !cb.wasRepaintBoundary {
		RepaintCompositedChild(child, true)
	} else {
		if cb.needsCompositedLayerUpdate {
			UpdateLayerProperties(child)
		}
		if foundation.DebugMode {
			child.DebugRegisterRepaintBoundaryPaint(true, false)
			cb.layerHandle.Layer().AsLayer().DebugCreator = creatorOr(cb.DebugCreator, child)
		}
	}
	childOffsetLayer := cb.layerHandle.Layer().(OffsetLayerLike)
	childOffsetLayer.AsOffsetLayer().SetOffset(offset)
	c.AppendLayer(childOffsetLayer)
}

// AppendLayer adds the layer to the container layer, after removing it from
// where it was. The context must not be recording.
//
// Flutter: rendering/object.dart PaintingContext.appendLayer
func (c *PaintingContext) AppendLayer(layer Layer) {
	if foundation.DebugMode && c.isRecording() {
		panic(errors.New("PaintingContextRecording", "a layer was appended while the context was still recording"))
	}
	layer.AsLayer().Remove()
	c.containerLayer.Append(layer)
}

func (c *PaintingContext) isRecording() bool { return c.canvas != nil }

// Recorder is the picture recorder, which starts recording if it was not.
//
// Flutter: rendering/object.dart PaintingContext.recorder
func (c *PaintingContext) Recorder() *ui.PictureRecorder {
	if c.recorder == nil {
		c.startRecording()
	}
	return c.recorder
}

// Canvas is the canvas to paint on, which starts a new picture layer if the
// context was not recording.
//
// Flutter: rendering/object.dart PaintingContext.canvas
func (c *PaintingContext) Canvas() *ui.Canvas {
	if c.canvas == nil {
		c.startRecording()
	}
	return c.canvas
}

func (c *PaintingContext) startRecording() {
	c.currentLayer = NewPictureLayer(c.estimatedBounds)
	c.recorder = ui.NewPictureRecorder()
	c.canvas = ui.NewCanvas(c.recorder)
	c.containerLayer.Append(c.currentLayer)
}

// AddCompositionCallback registers a callback on the container layer that runs
// when it is composited into a scene. It returns the function that removes it.
//
// Flutter: rendering/object.dart PaintingContext.addCompositionCallback
func (c *PaintingContext) AddCompositionCallback(callback CompositionCallback) func() {
	return c.containerLayer.AddCompositionCallback(callback)
}

// StopRecordingIfNeeded ends the picture that is being recorded, if any, and
// stores it in its layer. Paint code does not call it; the context does.
//
// Flutter: rendering/object.dart PaintingContext.stopRecordingIfNeeded
func (c *PaintingContext) StopRecordingIfNeeded() {
	if !c.isRecording() {
		return
	}
	if foundation.DebugMode {
		var o *DebugOptions
		if owner := c.containerLayer.owner; owner != nil {
			if d, ok := owner.(interface{ DebugOptions() *DebugOptions }); ok {
				o = d.DebugOptions()
			}
		}
		if o != nil {
			if o.RepaintRainbowEnabled {
				paint := ui.NewPaint()
				paint.SetStyle(ui.PaintingStyleStroke)
				paint.SetStrokeWidth(6)
				paint.SetColor(o.repaintColor().ToColor())
				c.canvas.DrawRect(c.estimatedBounds.Deflate(3), paint)
			}
			if o.PaintLayerBordersEnabled {
				paint := ui.NewPaint()
				paint.SetStyle(ui.PaintingStyleStroke)
				paint.SetStrokeWidth(1)
				paint.SetColor(ui.NewColorFromARGB(0xFF, 0xFF, 0x98, 0x00))
				c.canvas.DrawRect(c.estimatedBounds, paint)
			}
		}
	}
	c.currentLayer.SetPicture(c.recorder.EndRecording())
	c.currentLayer = nil
	c.recorder = nil
	c.canvas = nil
}

// SetIsComplexHint hints that the current picture is expensive to draw.
//
// Flutter: rendering/object.dart PaintingContext.setIsComplexHint
func (c *PaintingContext) SetIsComplexHint() {
	if c.currentLayer == nil {
		c.startRecording()
	}
	c.currentLayer.SetIsComplexHint(true)
}

// SetWillChangeHint hints that the current picture is likely to change.
//
// Flutter: rendering/object.dart PaintingContext.setWillChangeHint
func (c *PaintingContext) SetWillChangeHint() {
	if c.currentLayer == nil {
		c.startRecording()
	}
	c.currentLayer.SetWillChangeHint(true)
}

// AddLayer stops recording and adds the layer.
//
// Flutter: rendering/object.dart PaintingContext.addLayer
func (c *PaintingContext) AddLayer(layer Layer) {
	c.StopRecordingIfNeeded()
	c.AppendLayer(layer)
}

// PushLayer paints into a child layer: it appends the layer, creates a child
// context for it and calls the painter. childPaintBounds are the estimated
// bounds of the child (the default is those of this context).
//
// Flutter: rendering/object.dart PaintingContext.pushLayer
func (c *PaintingContext) PushLayer(childLayer ContainerLayerLike, painter PaintingContextCallback, offset ui.Offset, childPaintBounds foundation.Opt[ui.Rect]) {
	cl := childLayer.AsContainerLayer()
	if cl.HasChildren() {
		cl.RemoveAllChildren()
	}
	c.StopRecordingIfNeeded()
	c.AppendLayer(childLayer)
	childContext := NewPaintingContext(cl, childPaintBounds.OrElse(c.estimatedBounds))
	painter(childContext, offset)
	childContext.StopRecordingIfNeeded()
}

// PushClipRectOpts are the optional parameters of PushClipRect.
//
// Flutter: rendering/object.dart PaintingContext.pushClipRect
type PushClipRectOpts struct {
	// ClipBehavior defaults to ClipHardEdge.
	ClipBehavior ui.Clip
	// OldLayer is the layer of the previous paint, which is reused.
	OldLayer *ClipRectLayer
}

// PushClipRect clips the painter to the rectangle. If needsCompositing it uses
// a ClipRectLayer (the one returned, to keep as the next OldLayer); otherwise it
// clips the canvas and returns nil.
//
// Flutter: rendering/object.dart PaintingContext.pushClipRect
func (c *PaintingContext) PushClipRect(needsCompositing bool, offset ui.Offset, clipRect ui.Rect, painter PaintingContextCallback, opts ...PushClipRectOpts) *ClipRectLayer {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.ClipBehavior == ui.ClipNone {
		painter(c, offset)
		return nil
	}
	offsetClipRect := clipRect.Shift(offset)
	if needsCompositing {
		layer := o.OldLayer
		if layer == nil {
			layer = NewClipRectLayer()
		}
		layer.SetClipRect(offsetClipRect)
		layer.SetClipBehavior(o.ClipBehavior)
		c.PushLayer(layer, painter, offset, foundation.Some(offsetClipRect))
		return layer
	}
	painting.ClipRectAndPaint(c, offsetClipRect, o.ClipBehavior, offsetClipRect, func() { painter(c, offset) })
	return nil
}

// PushClipRRectOpts are the optional parameters of PushClipRRect.
//
// Flutter: rendering/object.dart PaintingContext.pushClipRRect
type PushClipRRectOpts struct {
	// ClipBehavior defaults to ClipAntiAlias.
	ClipBehavior ui.Clip
	// OldLayer is the layer of the previous paint, which is reused.
	OldLayer *ClipRRectLayer
}

// PushClipRRect clips the painter to the rounded rectangle; bounds are the
// bounds of what is painted. See PushClipRect.
//
// Flutter: rendering/object.dart PaintingContext.pushClipRRect
func (c *PaintingContext) PushClipRRect(needsCompositing bool, offset ui.Offset, bounds ui.Rect, clipRRect ui.RRect, painter PaintingContextCallback, opts ...PushClipRRectOpts) *ClipRRectLayer {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipAntiAlias
	}
	if o.ClipBehavior == ui.ClipNone {
		painter(c, offset)
		return nil
	}
	offsetBounds := bounds.Shift(offset)
	offsetClip := clipRRect.Shift(offset)
	if needsCompositing {
		layer := o.OldLayer
		if layer == nil {
			layer = NewClipRRectLayer()
		}
		layer.SetClipRRect(offsetClip)
		layer.SetClipBehavior(o.ClipBehavior)
		c.PushLayer(layer, painter, offset, foundation.Some(offsetBounds))
		return layer
	}
	painting.ClipRRectAndPaint(c, offsetClip, o.ClipBehavior, offsetBounds, func() { painter(c, offset) })
	return nil
}

// PushClipRSuperellipseOpts are the optional parameters of
// PushClipRSuperellipse.
//
// Flutter: rendering/object.dart PaintingContext.pushClipRSuperellipse
type PushClipRSuperellipseOpts struct {
	// ClipBehavior defaults to ClipAntiAlias.
	ClipBehavior ui.Clip
	// OldLayer is the layer of the previous paint, which is reused.
	OldLayer *ClipRSuperellipseLayer
}

// PushClipRSuperellipse clips the painter to the rounded superellipse. See
// PushClipRect.
//
// Flutter: rendering/object.dart PaintingContext.pushClipRSuperellipse
func (c *PaintingContext) PushClipRSuperellipse(needsCompositing bool, offset ui.Offset, bounds ui.Rect, clip ui.RSuperellipse, painter PaintingContextCallback, opts ...PushClipRSuperellipseOpts) *ClipRSuperellipseLayer {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipAntiAlias
	}
	if o.ClipBehavior == ui.ClipNone {
		painter(c, offset)
		return nil
	}
	offsetBounds := bounds.Shift(offset)
	offsetShape := clip.Shift(offset)
	if needsCompositing {
		layer := o.OldLayer
		if layer == nil {
			layer = NewClipRSuperellipseLayer()
		}
		layer.SetClipRSuperellipse(offsetShape)
		layer.SetClipBehavior(o.ClipBehavior)
		c.PushLayer(layer, painter, offset, foundation.Some(offsetBounds))
		return layer
	}
	painting.ClipRSuperellipseAndPaint(c, offsetShape, o.ClipBehavior, offsetBounds, func() { painter(c, offset) })
	return nil
}

// PushClipPathOpts are the optional parameters of PushClipPath.
//
// Flutter: rendering/object.dart PaintingContext.pushClipPath
type PushClipPathOpts struct {
	// ClipBehavior defaults to ClipAntiAlias.
	ClipBehavior ui.Clip
	// OldLayer is the layer of the previous paint, which is reused.
	OldLayer *ClipPathLayer
}

// PushClipPath clips the painter to the path. See PushClipRect.
//
// Flutter: rendering/object.dart PaintingContext.pushClipPath
func (c *PaintingContext) PushClipPath(needsCompositing bool, offset ui.Offset, bounds ui.Rect, clipPath *ui.Path, painter PaintingContextCallback, opts ...PushClipPathOpts) *ClipPathLayer {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipAntiAlias
	}
	if o.ClipBehavior == ui.ClipNone {
		painter(c, offset)
		return nil
	}
	offsetBounds := bounds.Shift(offset)
	offsetPath := clipPath.Shift(offset)
	if needsCompositing {
		layer := o.OldLayer
		if layer == nil {
			layer = NewClipPathLayer()
		}
		layer.SetClipPath(offsetPath)
		layer.SetClipBehavior(o.ClipBehavior)
		c.PushLayer(layer, painter, offset, foundation.Some(offsetBounds))
		return layer
	}
	painting.ClipPathAndPaint(c, offsetPath, o.ClipBehavior, offsetBounds, func() { painter(c, offset) })
	return nil
}

// PushColorFilter applies the color filter to what the painter paints, using a
// ColorFilterLayer (oldLayer is reused if not nil).
//
// Flutter: rendering/object.dart PaintingContext.pushColorFilter
func (c *PaintingContext) PushColorFilter(offset ui.Offset, colorFilter *ui.ColorFilter, painter PaintingContextCallback, oldLayer *ColorFilterLayer) *ColorFilterLayer {
	layer := oldLayer
	if layer == nil {
		layer = NewColorFilterLayer(colorFilter)
	} else {
		layer.SetColorFilter(colorFilter)
	}
	c.PushLayer(layer, painter, offset, foundation.None[ui.Rect]())
	return layer
}

// PushTransform transforms what the painter paints. If needsCompositing it uses
// a TransformLayer (oldLayer is reused if not nil) and returns it; otherwise
// it transforms the canvas and returns nil.
//
// Flutter: rendering/object.dart PaintingContext.pushTransform
func (c *PaintingContext) PushTransform(needsCompositing bool, offset ui.Offset, transform *ui.Matrix4, painter PaintingContextCallback, oldLayer *TransformLayer) *TransformLayer {
	effective := ui.NewMatrix4TranslationValues(float32(offset.Dx), float32(offset.Dy), 0)
	effective.MultiplyBy(transform)
	effective.Translate(float32(-offset.Dx), float32(-offset.Dy), 0)
	if needsCompositing {
		layer := oldLayer
		if layer == nil {
			layer = NewTransformLayer()
		}
		layer.SetTransform(effective)
		c.PushLayer(layer, painter, offset, foundation.Some(effective.InverseTransformRect(c.estimatedBounds)))
		return layer
	}
	c.Canvas().Save()
	c.Canvas().Transform(effective)
	painter(c, offset)
	c.Canvas().Restore()
	return nil
}

// PushOpacity makes what the painter paints partly transparent, using an
// OpacityLayer (oldLayer is reused if not nil).
//
// Flutter: rendering/object.dart PaintingContext.pushOpacity
func (c *PaintingContext) PushOpacity(offset ui.Offset, alpha int, painter PaintingContextCallback, oldLayer *OpacityLayer) *OpacityLayer {
	layer := oldLayer
	if layer == nil {
		layer = NewOpacityLayer()
	}
	layer.SetAlpha(alpha)
	layer.SetOffset(offset)
	c.PushLayer(layer, painter, ui.OffsetZero, foundation.None[ui.Rect]())
	return layer
}

// String describes the context.
//
// Flutter: rendering/object.dart PaintingContext.toString
func (c *PaintingContext) String() string {
	return foundation.DescribeIdentity(c) + "(layer: " + foundation.DescribeIdentity(c.containerLayer) +
		", canvas bounds: " + c.estimatedBounds.String() + ")"
}
