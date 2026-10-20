// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"strconv"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// ContainerLayer is a layer that has a list of children.
//
// Flutter: rendering/layer.dart ContainerLayer
type ContainerLayer struct {
	LayerBase
	firstChild Layer
	lastChild  Layer
}

// NewContainerLayer creates a layer without children.
//
// Flutter: rendering/layer.dart ContainerLayer.new
func NewContainerLayer() *ContainerLayer {
	l := &ContainerLayer{}
	InitContainerLayer(l, l)
	return l
}

// InitContainerLayer sets the outer object of a layer that embeds
// ContainerLayer.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitContainerLayer(c *ContainerLayer, self Layer) { InitLayer(&c.LayerBase, self) }

// AsContainerLayer returns the layer itself.
func (c *ContainerLayer) AsContainerLayer() *ContainerLayer { return c }

// ContainerLayerLike is a layer that has children: every type that embeds
// ContainerLayer.
type ContainerLayerLike interface {
	Layer
	AsContainerLayer() *ContainerLayer
	// ApplyTransform applies to the transform the transformation that this
	// layer applies to the child.
	//
	// Flutter: rendering/layer.dart ContainerLayer.applyTransform
	ApplyTransform(child Layer, transform *ui.Matrix4)
}

func (c *ContainerLayer) fireCompositionCallbacks(includeChildren bool) {
	c.LayerBase.fireCompositionCallbacks(includeChildren)
	if !includeChildren {
		return
	}
	for child := c.firstChild; child != nil; child = child.AsLayer().nextSibling {
		child.fireCompositionCallbacks(includeChildren)
	}
}

// FirstChild is the first child, or nil.
//
// Flutter: rendering/layer.dart ContainerLayer.firstChild
func (c *ContainerLayer) FirstChild() Layer { return c.firstChild }

// LastChild is the last child, or nil.
//
// Flutter: rendering/layer.dart ContainerLayer.lastChild
func (c *ContainerLayer) LastChild() Layer { return c.lastChild }

// HasChildren reports whether the layer has a child.
//
// Flutter: rendering/layer.dart ContainerLayer.hasChildren
func (c *ContainerLayer) HasChildren() bool { return c.firstChild != nil }

// SupportsRasterization is true if every child supports it.
//
// Flutter: rendering/layer.dart ContainerLayer.supportsRasterization
func (c *ContainerLayer) SupportsRasterization() bool {
	for child := c.lastChild; child != nil; child = child.AsLayer().previousSibling {
		if !child.SupportsRasterization() {
			return false
		}
	}
	return true
}

// BuildScene builds the scene of the layer tree with the builder, adding the
// layers that changed and retaining the others. It is called on the root.
//
// Flutter: rendering/layer.dart ContainerLayer.buildScene
func (c *ContainerLayer) BuildScene(builder *ui.SceneBuilder) *ui.Scene {
	c.self.UpdateSubtreeNeedsAddToScene()
	c.self.AddToScene(builder)
	if c.SubtreeHasCompositionCallbacks() {
		c.self.fireCompositionCallbacks(true)
	}
	c.needsAddToScene = false
	return builder.Build()
}

// Dispose removes the children and the callbacks.
//
// Flutter: rendering/layer.dart ContainerLayer.dispose
func (c *ContainerLayer) Dispose() {
	c.RemoveAllChildren()
	c.callbacks = nil
	c.LayerBase.Dispose()
}

// UpdateSubtreeNeedsAddToScene marks this layer if it or a descendant needs
// AddToScene.
//
// Flutter: rendering/layer.dart ContainerLayer.updateSubtreeNeedsAddToScene
func (c *ContainerLayer) UpdateSubtreeNeedsAddToScene() {
	c.LayerBase.UpdateSubtreeNeedsAddToScene()
	for child := c.firstChild; child != nil; child = child.AsLayer().nextSibling {
		child.UpdateSubtreeNeedsAddToScene()
		c.needsAddToScene = c.needsAddToScene || child.AsLayer().needsAddToScene
	}
}

// FindAnnotations searches the children from the last (front) to the first.
//
// Flutter: rendering/layer.dart ContainerLayer.findAnnotations
func (c *ContainerLayer) FindAnnotations(result *AnnotationResult, localPosition ui.Offset, onlyFirst bool) bool {
	for child := c.lastChild; child != nil; child = child.AsLayer().previousSibling {
		isAbsorbed := child.FindAnnotations(result, localPosition, onlyFirst)
		if isAbsorbed {
			return true
		}
		if onlyFirst && len(result.entries) > 0 {
			return isAbsorbed
		}
	}
	return false
}

// Attach attaches the layer and its children.
//
// Flutter: rendering/layer.dart ContainerLayer.attach
func (c *ContainerLayer) Attach(owner any) {
	c.debugCheckMutable("attach")
	c.LayerBase.Attach(owner)
	for child := c.firstChild; child != nil; child = child.AsLayer().nextSibling {
		child.Attach(owner)
	}
}

// Detach detaches the layer and its children, and calls its own composition
// callbacks.
//
// Flutter: rendering/layer.dart ContainerLayer.detach
func (c *ContainerLayer) Detach() {
	c.debugCheckMutable("detach")
	c.LayerBase.Detach()
	for child := c.firstChild; child != nil; child = child.AsLayer().nextSibling {
		child.Detach()
	}
	c.self.fireCompositionCallbacks(false)
}

// Append adds the layer to the end of the children. A debug build panics with
// LayerAppendInvalid if the child already has a parent or an owner, or would
// make a cycle.
//
// Flutter: rendering/layer.dart ContainerLayer.append
func (c *ContainerLayer) Append(child Layer) {
	c.debugCheckMutable("append")
	cb := child.AsLayer()
	if foundation.DebugMode {
		bad := ""
		switch {
		case child == c.self || child == c.firstChild || child == c.lastChild:
			bad = "the layer is a child of the container already, or the container itself"
		case cb.parent != nil:
			bad = "the layer has a parent"
		case cb.Attached():
			bad = "the layer is attached"
		case cb.nextSibling != nil || cb.previousSibling != nil:
			bad = "the layer has siblings"
		case cb.parentHandle.layer != nil:
			bad = "the layer is held as a child"
		default:
			node := c
			for node.parent != nil {
				node = node.parent
			}
			if Layer(node.self) == child {
				bad = "the layer is an ancestor: this would make a cycle"
			}
		}
		if bad != "" {
			panic(errors.New("LayerAppendInvalid", "a layer can not be appended: "+bad,
				"container", foundation.DescribeIdentity(c.self), "child", foundation.DescribeIdentity(child)))
		}
	}
	c.adoptChild(child)
	cb.previousSibling = c.lastChild
	if c.lastChild != nil {
		c.lastChild.AsLayer().nextSibling = child
	}
	c.lastChild = child
	if c.firstChild == nil {
		c.firstChild = child
	}
	cb.parentHandle.SetLayer(child)
}

func (c *ContainerLayer) adoptChild(child Layer) {
	c.debugCheckMutable("adoptChild")
	if !c.self.AlwaysNeedsAddToScene() {
		c.markNeedsAddToScene()
	}
	cb := child.AsLayer()
	if cb.compositionCount != 0 {
		c.updateSubtreeCompositionObserverCount(cb.compositionCount)
	}
	cb.parent = c
	if c.Attached() {
		child.Attach(c.owner)
	}
	c.RedepthChild(child)
}

// RedepthChildren updates the depth of every child.
//
// Flutter: rendering/layer.dart ContainerLayer.redepthChildren
func (c *ContainerLayer) RedepthChildren() {
	for child := c.firstChild; child != nil; child = child.AsLayer().nextSibling {
		c.RedepthChild(child)
	}
}

// RedepthChild makes the child deeper than this layer, if it is not.
//
// Flutter: rendering/layer.dart ContainerLayer.redepthChild
func (c *ContainerLayer) RedepthChild(child Layer) {
	cb := child.AsLayer()
	if cb.depth <= c.depth {
		cb.depth = c.depth + 1
		child.RedepthChildren()
	}
}

func (c *ContainerLayer) removeChild(child Layer) {
	cb := child.AsLayer()
	if cb.previousSibling == nil {
		c.firstChild = cb.nextSibling
	} else {
		cb.previousSibling.AsLayer().nextSibling = cb.nextSibling
	}
	if cb.nextSibling == nil {
		c.lastChild = cb.previousSibling
	} else {
		cb.nextSibling.AsLayer().previousSibling = cb.previousSibling
	}
	cb.previousSibling = nil
	cb.nextSibling = nil
	c.dropChild(child)
	cb.parentHandle.SetLayer(nil)
}

func (c *ContainerLayer) dropChild(child Layer) {
	c.debugCheckMutable("dropChild")
	if !c.self.AlwaysNeedsAddToScene() {
		c.markNeedsAddToScene()
	}
	cb := child.AsLayer()
	if cb.compositionCount != 0 {
		c.updateSubtreeCompositionObserverCount(-cb.compositionCount)
	}
	cb.parent = nil
	if c.Attached() {
		child.Detach()
	}
}

// RemoveAllChildren removes every child.
//
// Flutter: rendering/layer.dart ContainerLayer.removeAllChildren
func (c *ContainerLayer) RemoveAllChildren() {
	c.debugCheckMutable("removeAllChildren")
	child := c.firstChild
	for child != nil {
		cb := child.AsLayer()
		next := cb.nextSibling
		cb.previousSibling = nil
		cb.nextSibling = nil
		c.dropChild(child)
		cb.parentHandle.SetLayer(nil)
		child = next
	}
	c.firstChild = nil
	c.lastChild = nil
}

// AddToScene adds the children to the scene.
//
// Flutter: rendering/layer.dart ContainerLayer.addToScene
func (c *ContainerLayer) AddToScene(builder *ui.SceneBuilder) { c.AddChildrenToScene(builder) }

// AddChildrenToScene adds every child to the scene, retained when it did not
// change.
//
// Flutter: rendering/layer.dart ContainerLayer.addChildrenToScene
func (c *ContainerLayer) AddChildrenToScene(builder *ui.SceneBuilder) {
	for child := c.firstChild; child != nil; child = child.AsLayer().nextSibling {
		child.AsLayer().addToSceneWithRetainedRendering(builder)
	}
}

// ApplyTransform applies the transform that this layer gives its child; none
// for a plain container.
//
// Flutter: rendering/layer.dart ContainerLayer.applyTransform
func (c *ContainerLayer) ApplyTransform(child Layer, transform *ui.Matrix4) {}

// DepthFirstIterateChildren returns every descendant, depth first.
//
// Flutter: rendering/layer.dart ContainerLayer.depthFirstIterateChildren
func (c *ContainerLayer) DepthFirstIterateChildren() []Layer {
	var out []Layer
	for child := c.firstChild; child != nil; child = child.AsLayer().nextSibling {
		out = append(out, child)
		if cc, ok := child.(interface{ AsContainerLayer() *ContainerLayer }); ok {
			out = append(out, cc.AsContainerLayer().DepthFirstIterateChildren()...)
		}
	}
	return out
}

// DebugDescribeChildren names the children "child 1", "child 2" and so on.
//
// Flutter: rendering/layer.dart ContainerLayer.debugDescribeChildren
func (c *ContainerLayer) DebugDescribeChildren() []foundation.DiagnosticsNode {
	var out []foundation.DiagnosticsNode
	count := 1
	for child := c.firstChild; child != nil; child = child.AsLayer().nextSibling {
		out = append(out, foundation.ToDiagnosticsNode(child, foundation.DiagnosticsNodeOpts{Name: foundation.Some("child " + strconv.Itoa(count))}))
		count++
	}
	return out
}

// OffsetLayer is a layer that offsets its children. It is the layer of a
// repaint boundary.
//
// Flutter: rendering/layer.dart OffsetLayer
type OffsetLayer struct {
	ContainerLayer
	offset ui.Offset
}

// OffsetLayerLike is a layer that is an OffsetLayer: the layer of a repaint
// boundary can be any of them.
type OffsetLayerLike interface {
	ContainerLayerLike
	AsOffsetLayer() *OffsetLayer
}

// OffsetLayerOpts are the optional parameters of NewOffsetLayer.
//
// Flutter: rendering/layer.dart OffsetLayer.new
type OffsetLayerOpts struct {
	// Offset is how far the children are moved.
	Offset ui.Offset
}

// NewOffsetLayer creates an offset layer.
//
// Flutter: rendering/layer.dart OffsetLayer.new
func NewOffsetLayer(opts ...OffsetLayerOpts) *OffsetLayer {
	l := &OffsetLayer{offset: firstOpts(opts).Offset}
	InitContainerLayer(&l.ContainerLayer, l)
	return l
}

// InitOffsetLayer sets the outer object of a layer that embeds OffsetLayer.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitOffsetLayer(l *OffsetLayer, self Layer, opts ...OffsetLayerOpts) {
	l.offset = firstOpts(opts).Offset
	InitContainerLayer(&l.ContainerLayer, self)
}

// AsOffsetLayer returns the layer itself.
func (l *OffsetLayer) AsOffsetLayer() *OffsetLayer { return l }

// Offset is how far the children are moved.
//
// Flutter: rendering/layer.dart OffsetLayer.offset
func (l *OffsetLayer) Offset() ui.Offset { return l.offset }

// SetOffset changes the offset and marks the layer as needing AddToScene.
//
// Flutter: rendering/layer.dart OffsetLayer.offset
func (l *OffsetLayer) SetOffset(v ui.Offset) {
	if v != l.offset {
		l.markNeedsAddToScene()
	}
	l.offset = v
}

// FindAnnotations searches the children at the position minus the offset.
//
// Flutter: rendering/layer.dart OffsetLayer.findAnnotations
func (l *OffsetLayer) FindAnnotations(result *AnnotationResult, localPosition ui.Offset, onlyFirst bool) bool {
	return l.ContainerLayer.FindAnnotations(result, localPosition.Sub(l.offset), onlyFirst)
}

// ApplyTransform translates the transform by the offset.
//
// Flutter: rendering/layer.dart OffsetLayer.applyTransform
func (l *OffsetLayer) ApplyTransform(child Layer, transform *ui.Matrix4) {
	transform.Translate(float32(l.offset.Dx), float32(l.offset.Dy), 0)
}

// AddToScene pushes the offset, adds the children and pops.
//
// Flutter: rendering/layer.dart OffsetLayer.addToScene
func (l *OffsetLayer) AddToScene(builder *ui.SceneBuilder) {
	old, _ := l.engineLayer.(*ui.OffsetEngineLayer)
	l.SetEngineLayer(builder.PushOffset(l.offset.Dx, l.offset.Dy, ui.PushOffsetOpts{OldLayer: old}))
	l.AddChildrenToScene(builder)
	builder.Pop()
}

// DebugFillProperties adds the offset.
//
// Flutter: rendering/layer.dart OffsetLayer.debugFillProperties
func (l *OffsetLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.ContainerLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("offset", l.offset))
}

func (l *OffsetLayer) createSceneForImage(bounds ui.Rect, pixelRatio float64) *ui.Scene {
	builder := ui.NewSceneBuilder()
	transform := ui.NewMatrix4Diagonal3Values(float32(pixelRatio), float32(pixelRatio), 1)
	transform.Translate(float32(-(bounds.Left + l.offset.Dx)), float32(-(bounds.Top + l.offset.Dy)), 0)
	builder.PushTransform(transform)
	return l.BuildScene(builder)
}

// ToImage renders the layer tree (as of the last compositing) to an image of
// the bounds, at the pixel ratio (1 if zero).
//
// Deviation: Flutter's returns a Future and draws with the engine; this blocks,
// returns the error, and is given the rasterizer in the options (SPEC §12.10).
//
// Flutter: rendering/layer.dart OffsetLayer.toImage
func (l *OffsetLayer) ToImage(bounds ui.Rect, pixelRatio float64, opts ...ui.ToImageOpts) (*ui.Image, error) {
	if pixelRatio == 0 {
		pixelRatio = 1
	}
	scene := l.createSceneForImage(bounds, pixelRatio)
	defer scene.Dispose()
	return scene.ToImage(ceilInt(pixelRatio*bounds.Width()), ceilInt(pixelRatio*bounds.Height()), opts...)
}

// ToImageSync is ToImage on the calling goroutine.
//
// Flutter: rendering/layer.dart OffsetLayer.toImageSync
func (l *OffsetLayer) ToImageSync(bounds ui.Rect, pixelRatio float64, opts ...ui.ToImageOpts) (*ui.Image, error) {
	if pixelRatio == 0 {
		pixelRatio = 1
	}
	scene := l.createSceneForImage(bounds, pixelRatio)
	defer scene.Dispose()
	return scene.ToImageSync(ceilInt(pixelRatio*bounds.Width()), ceilInt(pixelRatio*bounds.Height()), opts...)
}

func ceilInt(v float64) int {
	i := int(v)
	if float64(i) < v {
		i++
	}
	return i
}

func debugCheckClipBehavior(b ui.Clip) {
	if foundation.DebugMode && b == ui.ClipNone {
		panic(errors.New("ClipLayerBehaviorNone", "a clip layer needs a clip behavior other than none"))
	}
}

// ClipRectLayer is a layer that clips its children to a rectangle.
//
// Flutter: rendering/layer.dart ClipRectLayer
type ClipRectLayer struct {
	ContainerLayer
	clipRect     foundation.Opt[ui.Rect]
	clipBehavior ui.Clip
}

// ClipRectLayerOpts are the optional parameters of NewClipRectLayer.
//
// Flutter: rendering/layer.dart ClipRectLayer.new
type ClipRectLayerOpts struct {
	// ClipRect is the rectangle; it must be set before the layer is added to a
	// scene.
	ClipRect foundation.Opt[ui.Rect]
	// ClipBehavior defaults to ClipHardEdge; ClipNone panics in debug builds.
	ClipBehavior ui.Clip
}

// NewClipRectLayer creates a layer that clips to a rectangle.
//
// Flutter: rendering/layer.dart ClipRectLayer.new
func NewClipRectLayer(opts ...ClipRectLayerOpts) *ClipRectLayer {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	debugCheckClipBehavior(o.ClipBehavior)
	l := &ClipRectLayer{clipRect: o.ClipRect, clipBehavior: o.ClipBehavior}
	InitContainerLayer(&l.ContainerLayer, l)
	return l
}

// ClipRect is the rectangle that is the clip.
//
// Flutter: rendering/layer.dart ClipRectLayer.clipRect
func (l *ClipRectLayer) ClipRect() foundation.Opt[ui.Rect] { return l.clipRect }

// SetClipRect changes the clip and marks the layer as needing AddToScene.
//
// Flutter: rendering/layer.dart ClipRectLayer.clipRect
func (l *ClipRectLayer) SetClipRect(v ui.Rect) {
	if cur, ok := l.clipRect.Get(); !ok || cur != v {
		l.clipRect = foundation.Some(v)
		l.markNeedsAddToScene()
	}
}

// DescribeClipBounds is the clip rectangle.
//
// Flutter: rendering/layer.dart ClipRectLayer.describeClipBounds
func (l *ClipRectLayer) DescribeClipBounds() foundation.Opt[ui.Rect] { return l.clipRect }

// ClipBehavior is how the edge of the clip is drawn.
//
// Flutter: rendering/layer.dart ClipRectLayer.clipBehavior
func (l *ClipRectLayer) ClipBehavior() ui.Clip { return l.clipBehavior }

// SetClipBehavior changes the behavior; ClipNone panics in debug builds.
//
// Flutter: rendering/layer.dart ClipRectLayer.clipBehavior
func (l *ClipRectLayer) SetClipBehavior(v ui.Clip) {
	debugCheckClipBehavior(v)
	if v != l.clipBehavior {
		l.clipBehavior = v
		l.markNeedsAddToScene()
	}
}

// FindAnnotations finds nothing outside the clip rectangle.
//
// Flutter: rendering/layer.dart ClipRectLayer.findAnnotations
func (l *ClipRectLayer) FindAnnotations(result *AnnotationResult, localPosition ui.Offset, onlyFirst bool) bool {
	if r, _ := l.clipRect.Get(); !r.Contains(localPosition) {
		return false
	}
	return l.ContainerLayer.FindAnnotations(result, localPosition, onlyFirst)
}

// AddToScene pushes the clip, adds the children and pops.
//
// Flutter: rendering/layer.dart ClipRectLayer.addToScene
func (l *ClipRectLayer) AddToScene(builder *ui.SceneBuilder) {
	r, ok := l.clipRect.Get()
	if !ok {
		panic(errors.New("LayerPropertyMissing", "a ClipRectLayer without a clip rectangle was added to a scene"))
	}
	enabled := !foundation.DebugMode || !l.debugOptions().DisableClipLayers
	if enabled {
		old, _ := l.engineLayer.(*ui.ClipRectEngineLayer)
		l.SetEngineLayer(builder.PushClipRect(r, ui.PushClipRectOpts{ClipBehavior: l.clipBehavior, OldLayer: old}))
	} else {
		l.SetEngineLayer(nil)
	}
	l.AddChildrenToScene(builder)
	if enabled {
		builder.Pop()
	}
}

// DebugFillProperties adds the clip.
//
// Flutter: rendering/layer.dart ClipRectLayer.debugFillProperties
func (l *ClipRectLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.ContainerLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsPropertyOpt("clipRect", l.clipRect))
	p.Add(foundation.NewDiagnosticsProperty("clipBehavior", l.clipBehavior))
}

// ClipRRectLayer is a layer that clips its children to a rounded rectangle.
//
// Flutter: rendering/layer.dart ClipRRectLayer
type ClipRRectLayer struct {
	ContainerLayer
	clipRRect    foundation.Opt[ui.RRect]
	clipBehavior ui.Clip
}

// ClipRRectLayerOpts are the optional parameters of NewClipRRectLayer.
//
// Flutter: rendering/layer.dart ClipRRectLayer.new
type ClipRRectLayerOpts struct {
	// ClipRRect is the rounded rectangle; it must be set before the layer is
	// added to a scene.
	ClipRRect foundation.Opt[ui.RRect]
	// ClipBehavior defaults to ClipAntiAlias; ClipNone panics in debug builds.
	ClipBehavior ui.Clip
}

// NewClipRRectLayer creates a layer that clips to a rounded rectangle.
//
// Flutter: rendering/layer.dart ClipRRectLayer.new
func NewClipRRectLayer(opts ...ClipRRectLayerOpts) *ClipRRectLayer {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipAntiAlias
	}
	debugCheckClipBehavior(o.ClipBehavior)
	l := &ClipRRectLayer{clipRRect: o.ClipRRect, clipBehavior: o.ClipBehavior}
	InitContainerLayer(&l.ContainerLayer, l)
	return l
}

// ClipRRect is the rounded rectangle that is the clip.
//
// Flutter: rendering/layer.dart ClipRRectLayer.clipRRect
func (l *ClipRRectLayer) ClipRRect() foundation.Opt[ui.RRect] { return l.clipRRect }

// SetClipRRect changes the clip and marks the layer as needing AddToScene.
//
// Flutter: rendering/layer.dart ClipRRectLayer.clipRRect
func (l *ClipRRectLayer) SetClipRRect(v ui.RRect) {
	if cur, ok := l.clipRRect.Get(); !ok || cur != v {
		l.clipRRect = foundation.Some(v)
		l.markNeedsAddToScene()
	}
}

// DescribeClipBounds is the outer rectangle of the clip.
//
// Flutter: rendering/layer.dart ClipRRectLayer.describeClipBounds
func (l *ClipRRectLayer) DescribeClipBounds() foundation.Opt[ui.Rect] {
	if r, ok := l.clipRRect.Get(); ok {
		return foundation.Some(r.OuterRect())
	}
	return foundation.None[ui.Rect]()
}

// ClipBehavior is how the edge of the clip is drawn.
//
// Flutter: rendering/layer.dart ClipRRectLayer.clipBehavior
func (l *ClipRRectLayer) ClipBehavior() ui.Clip { return l.clipBehavior }

// SetClipBehavior changes the behavior; ClipNone panics in debug builds.
//
// Flutter: rendering/layer.dart ClipRRectLayer.clipBehavior
func (l *ClipRRectLayer) SetClipBehavior(v ui.Clip) {
	debugCheckClipBehavior(v)
	if v != l.clipBehavior {
		l.clipBehavior = v
		l.markNeedsAddToScene()
	}
}

// FindAnnotations finds nothing outside the clip.
//
// Flutter: rendering/layer.dart ClipRRectLayer.findAnnotations
func (l *ClipRRectLayer) FindAnnotations(result *AnnotationResult, localPosition ui.Offset, onlyFirst bool) bool {
	if r, _ := l.clipRRect.Get(); !r.Contains(localPosition) {
		return false
	}
	return l.ContainerLayer.FindAnnotations(result, localPosition, onlyFirst)
}

// AddToScene pushes the clip, adds the children and pops.
//
// Flutter: rendering/layer.dart ClipRRectLayer.addToScene
func (l *ClipRRectLayer) AddToScene(builder *ui.SceneBuilder) {
	r, ok := l.clipRRect.Get()
	if !ok {
		panic(errors.New("LayerPropertyMissing", "a ClipRRectLayer without a clip was added to a scene"))
	}
	enabled := !foundation.DebugMode || !l.debugOptions().DisableClipLayers
	if enabled {
		old, _ := l.engineLayer.(*ui.ClipRRectEngineLayer)
		l.SetEngineLayer(builder.PushClipRRect(r, ui.PushClipRRectOpts{ClipBehavior: l.clipBehavior, OldLayer: old}))
	} else {
		l.SetEngineLayer(nil)
	}
	l.AddChildrenToScene(builder)
	if enabled {
		builder.Pop()
	}
}

// DebugFillProperties adds the clip.
//
// Flutter: rendering/layer.dart ClipRRectLayer.debugFillProperties
func (l *ClipRRectLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.ContainerLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsPropertyOpt("clipRRect", l.clipRRect))
	p.Add(foundation.NewDiagnosticsProperty("clipBehavior", l.clipBehavior))
}

// ClipRSuperellipseLayer is a layer that clips its children to a rounded
// superellipse.
//
// Flutter: rendering/layer.dart ClipRSuperellipseLayer
type ClipRSuperellipseLayer struct {
	ContainerLayer
	clip         foundation.Opt[ui.RSuperellipse]
	clipBehavior ui.Clip
}

// ClipRSuperellipseLayerOpts are the optional parameters of
// NewClipRSuperellipseLayer.
//
// Flutter: rendering/layer.dart ClipRSuperellipseLayer.new
type ClipRSuperellipseLayerOpts struct {
	// ClipRSuperellipse is the shape; it must be set before the layer is added
	// to a scene.
	ClipRSuperellipse foundation.Opt[ui.RSuperellipse]
	// ClipBehavior defaults to ClipAntiAlias; ClipNone panics in debug builds.
	ClipBehavior ui.Clip
}

// NewClipRSuperellipseLayer creates a layer that clips to a rounded
// superellipse.
//
// Flutter: rendering/layer.dart ClipRSuperellipseLayer.new
func NewClipRSuperellipseLayer(opts ...ClipRSuperellipseLayerOpts) *ClipRSuperellipseLayer {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipAntiAlias
	}
	debugCheckClipBehavior(o.ClipBehavior)
	l := &ClipRSuperellipseLayer{clip: o.ClipRSuperellipse, clipBehavior: o.ClipBehavior}
	InitContainerLayer(&l.ContainerLayer, l)
	return l
}

// ClipRSuperellipse is the shape that is the clip.
//
// Flutter: rendering/layer.dart ClipRSuperellipseLayer.clipRSuperellipse
func (l *ClipRSuperellipseLayer) ClipRSuperellipse() foundation.Opt[ui.RSuperellipse] { return l.clip }

// SetClipRSuperellipse changes the clip and marks the layer as needing
// AddToScene.
//
// Flutter: rendering/layer.dart ClipRSuperellipseLayer.clipRSuperellipse
func (l *ClipRSuperellipseLayer) SetClipRSuperellipse(v ui.RSuperellipse) {
	if cur, ok := l.clip.Get(); !ok || cur != v {
		l.clip = foundation.Some(v)
		l.markNeedsAddToScene()
	}
}

// DescribeClipBounds is the outer rectangle of the clip.
//
// Flutter: rendering/layer.dart ClipRSuperellipseLayer.describeClipBounds
func (l *ClipRSuperellipseLayer) DescribeClipBounds() foundation.Opt[ui.Rect] {
	if r, ok := l.clip.Get(); ok {
		return foundation.Some(r.OuterRect())
	}
	return foundation.None[ui.Rect]()
}

// ClipBehavior is how the edge of the clip is drawn.
//
// Flutter: rendering/layer.dart ClipRSuperellipseLayer.clipBehavior
func (l *ClipRSuperellipseLayer) ClipBehavior() ui.Clip { return l.clipBehavior }

// SetClipBehavior changes the behavior; ClipNone panics in debug builds.
//
// Flutter: rendering/layer.dart ClipRSuperellipseLayer.clipBehavior
func (l *ClipRSuperellipseLayer) SetClipBehavior(v ui.Clip) {
	debugCheckClipBehavior(v)
	if v != l.clipBehavior {
		l.clipBehavior = v
		l.markNeedsAddToScene()
	}
}

// FindAnnotations finds nothing outside the outer rectangle of the clip.
//
// Flutter: rendering/layer.dart ClipRSuperellipseLayer.findAnnotations
func (l *ClipRSuperellipseLayer) FindAnnotations(result *AnnotationResult, localPosition ui.Offset, onlyFirst bool) bool {
	if r, _ := l.clip.Get(); !r.OuterRect().Contains(localPosition) {
		return false
	}
	return l.ContainerLayer.FindAnnotations(result, localPosition, onlyFirst)
}

// AddToScene pushes the clip, adds the children and pops.
//
// Flutter: rendering/layer.dart ClipRSuperellipseLayer.addToScene
func (l *ClipRSuperellipseLayer) AddToScene(builder *ui.SceneBuilder) {
	r, ok := l.clip.Get()
	if !ok {
		panic(errors.New("LayerPropertyMissing", "a ClipRSuperellipseLayer without a clip was added to a scene"))
	}
	enabled := !foundation.DebugMode || !l.debugOptions().DisableClipLayers
	if enabled {
		old, _ := l.engineLayer.(*ui.ClipRSuperellipseEngineLayer)
		l.SetEngineLayer(builder.PushClipRSuperellipse(r, ui.PushClipRSuperellipseOpts{ClipBehavior: l.clipBehavior, OldLayer: old}))
	} else {
		l.SetEngineLayer(nil)
	}
	l.AddChildrenToScene(builder)
	if enabled {
		builder.Pop()
	}
}

// DebugFillProperties adds the clip.
//
// Flutter: rendering/layer.dart ClipRSuperellipseLayer.debugFillProperties
func (l *ClipRSuperellipseLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.ContainerLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsPropertyOpt("clipRSuperellipse", l.clip))
	p.Add(foundation.NewDiagnosticsProperty("clipBehavior", l.clipBehavior))
}

// ClipPathLayer is a layer that clips its children to a path.
//
// Flutter: rendering/layer.dart ClipPathLayer
type ClipPathLayer struct {
	ContainerLayer
	clipPath     *ui.Path
	clipBehavior ui.Clip
}

// ClipPathLayerOpts are the optional parameters of NewClipPathLayer.
//
// Flutter: rendering/layer.dart ClipPathLayer.new
type ClipPathLayerOpts struct {
	// ClipPath is the path; it must be set before the layer is added to a
	// scene.
	ClipPath *ui.Path
	// ClipBehavior defaults to ClipAntiAlias; ClipNone panics in debug builds.
	ClipBehavior ui.Clip
}

// NewClipPathLayer creates a layer that clips to a path.
//
// Flutter: rendering/layer.dart ClipPathLayer.new
func NewClipPathLayer(opts ...ClipPathLayerOpts) *ClipPathLayer {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipAntiAlias
	}
	debugCheckClipBehavior(o.ClipBehavior)
	l := &ClipPathLayer{clipPath: o.ClipPath, clipBehavior: o.ClipBehavior}
	InitContainerLayer(&l.ContainerLayer, l)
	return l
}

// ClipPath is the path that is the clip.
//
// Flutter: rendering/layer.dart ClipPathLayer.clipPath
func (l *ClipPathLayer) ClipPath() *ui.Path { return l.clipPath }

// SetClipPath changes the clip and marks the layer as needing AddToScene.
//
// Flutter: rendering/layer.dart ClipPathLayer.clipPath
func (l *ClipPathLayer) SetClipPath(v *ui.Path) {
	if v != l.clipPath {
		l.clipPath = v
		l.markNeedsAddToScene()
	}
}

// DescribeClipBounds is the bounds of the path.
//
// Flutter: rendering/layer.dart ClipPathLayer.describeClipBounds
func (l *ClipPathLayer) DescribeClipBounds() foundation.Opt[ui.Rect] {
	if l.clipPath == nil {
		return foundation.None[ui.Rect]()
	}
	return foundation.Some(l.clipPath.GetBounds())
}

// ClipBehavior is how the edge of the clip is drawn.
//
// Flutter: rendering/layer.dart ClipPathLayer.clipBehavior
func (l *ClipPathLayer) ClipBehavior() ui.Clip { return l.clipBehavior }

// SetClipBehavior changes the behavior; ClipNone panics in debug builds.
//
// Flutter: rendering/layer.dart ClipPathLayer.clipBehavior
func (l *ClipPathLayer) SetClipBehavior(v ui.Clip) {
	debugCheckClipBehavior(v)
	if v != l.clipBehavior {
		l.clipBehavior = v
		l.markNeedsAddToScene()
	}
}

// FindAnnotations finds nothing outside the path.
//
// Flutter: rendering/layer.dart ClipPathLayer.findAnnotations
func (l *ClipPathLayer) FindAnnotations(result *AnnotationResult, localPosition ui.Offset, onlyFirst bool) bool {
	if !l.clipPath.Contains(localPosition) {
		return false
	}
	return l.ContainerLayer.FindAnnotations(result, localPosition, onlyFirst)
}

// AddToScene pushes the clip, adds the children and pops.
//
// Flutter: rendering/layer.dart ClipPathLayer.addToScene
func (l *ClipPathLayer) AddToScene(builder *ui.SceneBuilder) {
	if l.clipPath == nil {
		panic(errors.New("LayerPropertyMissing", "a ClipPathLayer without a path was added to a scene"))
	}
	enabled := !foundation.DebugMode || !l.debugOptions().DisableClipLayers
	if enabled {
		old, _ := l.engineLayer.(*ui.ClipPathEngineLayer)
		l.SetEngineLayer(builder.PushClipPath(l.clipPath, ui.PushClipPathOpts{ClipBehavior: l.clipBehavior, OldLayer: old}))
	} else {
		l.SetEngineLayer(nil)
	}
	l.AddChildrenToScene(builder)
	if enabled {
		builder.Pop()
	}
}

// DebugFillProperties adds the clip behavior.
//
// Flutter: rendering/layer.dart ClipPathLayer.debugFillProperties
func (l *ClipPathLayer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	l.ContainerLayer.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("clipBehavior", l.clipBehavior))
}
