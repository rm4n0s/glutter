// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// CreateLocalImageConfiguration is the ImageConfiguration for what the context
// shows: the pixel ratio of its MediaQuery (when there is one), its text
// direction, the platform and the asset bundle and painting binding of the
// binding.
//
// Deviation: the bundle is the one of the painting binding, as there is no
// DefaultAssetBundle widget (SPEC §12.10).
//
// Flutter: widgets/image.dart createLocalImageConfiguration
func CreateLocalImageConfiguration(bc BuildContext, size foundation.Opt[ui.Size]) painting.ImageConfiguration {
	c := painting.ImageConfiguration{
		DevicePixelRatio: foundation.Some(1.0),
		Size:             size,
		Platform:         foundation.Some(foundation.DefaultTargetPlatform()),
		TextDirection:    maybeTextDirection(bc),
	}
	if b := MaybeWidgetsBindingOf(bc); b != nil {
		c.Binding = b.Painting()
		c.Bundle = b.Painting().AssetBundle()
	}
	if r, ok := MediaQueryMaybeDevicePixelRatioOf(bc); ok {
		c.DevicePixelRatio = foundation.Some(r)
	}
	if l, ok := LocalizationsMaybeLocaleOf(bc); ok {
		c.Locale = foundation.Some(l)
	}
	return c
}

// NewRepaintBoundaryWrap creates a RepaintBoundary with a key made from the
// child's key, or from the index of the child when it has none, so that the
// boundaries of a list of children keep their identity.
//
// Flutter: widgets/basic.dart RepaintBoundary.wrap
func NewRepaintBoundaryWrap(child Widget, childIndex int) *RepaintBoundary {
	var key foundation.Key
	if k := child.Key(); k != nil {
		key = foundation.NewValueKey[foundation.Key](k)
	} else {
		key = foundation.NewValueKey(childIndex)
	}
	return NewRepaintBoundary(child, RepaintBoundaryOpts{Key: key})
}

// RepaintBoundaryWrapAll wraps every widget in a RepaintBoundary with
// NewRepaintBoundaryWrap.
//
// Flutter: widgets/basic.dart RepaintBoundary.wrapAll
func RepaintBoundaryWrapAll(widgets []Widget) []Widget {
	out := make([]Widget, len(widgets))
	for i, w := range widgets {
		out[i] = NewRepaintBoundaryWrap(w, i)
	}
	return out
}

// NewTransformTranslate creates a widget that translates its child.
//
// Flutter: widgets/basic.dart Transform.translate
func NewTransformTranslate(offset ui.Offset, child Widget, opts ...TransformOpts) *Transform {
	o := firstOpts(opts)
	o.Origin, o.Alignment = foundation.None[ui.Offset](), nil
	return NewTransform(ui.NewMatrix4TranslationValues(float32(offset.Dx), float32(offset.Dy), 0), child, o)
}

// NewTransformRotate creates a widget that rotates its child by the angle, in
// radians, around the alignment (the center by default).
//
// Flutter: widgets/basic.dart Transform.rotate
func NewTransformRotate(angle float64, child Widget, opts ...TransformOpts) *Transform {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	return NewTransform(ui.NewMatrix4RotationZ(angle), child, o)
}

// TransformScaleOpts are the optional parameters of NewTransformScale.
//
// Flutter: widgets/basic.dart Transform.scale
type TransformScaleOpts struct {
	TransformOpts
	// Scale scales both axes by the same amount; it excludes ScaleX and ScaleY.
	Scale foundation.Opt[float64]
	// ScaleX scales the x axis.
	ScaleX foundation.Opt[float64]
	// ScaleY scales the y axis.
	ScaleY foundation.Opt[float64]
}

// NewTransformScale creates a widget that scales its child around the
// alignment (the center by default). In debug builds it panics with the tag
// TransformScaleInvalid when Scale is given together with ScaleX or ScaleY, or
// when none of them is.
//
// Flutter: widgets/basic.dart Transform.scale
func NewTransformScale(child Widget, opts ...TransformScaleOpts) *Transform {
	o := firstOpts(opts)
	if foundation.DebugMode && (o.Scale.IsSome() && (o.ScaleX.IsSome() || o.ScaleY.IsSome()) ||
		!o.Scale.IsSome() && !o.ScaleX.IsSome() && !o.ScaleY.IsSome()) {
		panic(errors.New("TransformScaleInvalid", "provide either a scale or a scaleX and a scaleY, not both"))
	}
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	sx := o.Scale.OrElse(o.ScaleX.OrElse(1))
	sy := o.Scale.OrElse(o.ScaleY.OrElse(1))
	return NewTransform(ui.NewMatrix4Diagonal3Values(float32(sx), float32(sy), 1), child, o.TransformOpts)
}

// TransformFlipOpts are the optional parameters of NewTransformFlip.
//
// Flutter: widgets/basic.dart Transform.flip
type TransformFlipOpts struct {
	TransformOpts
	// FlipX mirrors the child around the vertical axis.
	FlipX bool
	// FlipY mirrors the child around the horizontal axis.
	FlipY bool
}

// NewTransformFlip creates a widget that mirrors its child.
//
// Flutter: widgets/basic.dart Transform.flip
func NewTransformFlip(child Widget, opts ...TransformFlipOpts) *Transform {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	x, y := float32(1), float32(1)
	if o.FlipX {
		x = -1
	}
	if o.FlipY {
		y = -1
	}
	return NewTransform(ui.NewMatrix4Diagonal3Values(x, y, 1), child, o.TransformOpts)
}

// NewClipPathShape creates a widget that clips its child to the outer path of a
// shape border.
//
// Flutter: widgets/basic.dart ClipPath.shape
func NewClipPathShape(shape painting.ShapeBorder, child Widget, opts ...ClipPathOpts) *ClipPath {
	o := firstOpts(opts)
	o.Clipper = rendering.NewShapeBorderClipper(shape)
	return NewClipPath(child, o)
}

// LayoutId is a widget that provides the id of a child of a
// CustomMultiChildLayout to its delegate.
//
// Flutter: widgets/basic.dart LayoutId
type LayoutId struct {
	ParentDataWidgetOf[*rendering.MultiChildLayoutParentData]
	id any
}

// NewLayoutId creates a widget that identifies the child to the layout
// delegate; the key of the widget is a ValueKey of the id.
//
// Flutter: widgets/basic.dart LayoutId.new
func NewLayoutId(id any, child Widget) *LayoutId {
	return &LayoutId{NewParentDataWidgetBase[*rendering.MultiChildLayoutParentData](child, WidgetOpts{Key: foundation.NewValueKey(id)}), id}
}

// ID is the identity of the child.
//
// Flutter: widgets/basic.dart LayoutId.id
func (w *LayoutId) ID() any { return w.id }

// ApplyParentData writes the id to the parent data of the render object.
//
// Flutter: widgets/basic.dart LayoutId.applyParentData
func (w *LayoutId) ApplyParentData(renderObject rendering.RenderObject) {
	pd := renderObject.ParentData().(*rendering.MultiChildLayoutParentData)
	if pd.ID != w.id {
		pd.ID = w.id
		if p := renderObject.AsRenderObject().Parent(); p != nil {
			p.MarkNeedsLayout()
		}
	}
}

// DebugTypicalAncestorWidgetClass is the name of the widget this one belongs in.
//
// Flutter: widgets/basic.dart LayoutId.debugTypicalAncestorWidgetClass
func (*LayoutId) DebugTypicalAncestorWidgetClass() string { return "CustomMultiChildLayout" }

// ColoredBoxOpts are the optional parameters of NewColoredBox.
//
// Flutter: widgets/basic.dart ColoredBox.new
type ColoredBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// IsAntiAlias is whether to paint the box with anti-aliasing; the default is
	// true.
	IsAntiAlias foundation.Opt[bool]
}

// ColoredBox is a widget that paints its area with a specified color and then
// draws its child on top of that color.
//
// Flutter: widgets/basic.dart ColoredBox
type ColoredBox struct {
	SingleChildRenderObjectWidget
	color       ui.Color
	isAntiAlias bool
}

// NewColoredBox creates a widget that paints its area with the color.
//
// Flutter: widgets/basic.dart ColoredBox.new
func NewColoredBox(color ui.Color, child Widget, opts ...ColoredBoxOpts) *ColoredBox {
	o := firstOpts(opts)
	return &ColoredBox{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), color, o.IsAntiAlias.OrElse(true)}
}

// IsAntiAlias is whether the box is painted with anti-aliasing.
//
// Flutter: widgets/basic.dart ColoredBox.isAntiAlias
func (w *ColoredBox) IsAntiAlias() bool { return w.isAntiAlias }

// Color is the color to paint the background area with.
//
// Flutter: widgets/basic.dart ColoredBox.color
func (w *ColoredBox) Color() ui.Color { return w.color }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart ColoredBox.createRenderObject
func (w *ColoredBox) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := &renderColoredBox{color: w.color, isAntiAlias: w.isAntiAlias}
	rendering.InitRenderProxyBoxWithHitTestBehavior(&r.RenderProxyBoxWithHitTestBehavior, r, rendering.HitTestBehaviorOpaque, nil)
	return r
}

// UpdateRenderObject copies the color to the render object.
//
// Flutter: widgets/basic.dart ColoredBox.updateRenderObject
func (w *ColoredBox) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderColoredBox)
	r.setColor(w.color)
	r.setIsAntiAlias(w.isAntiAlias)
}

// renderColoredBox paints a color over its area, then its child.
//
// Flutter: widgets/basic.dart _RenderColoredBox
type renderColoredBox struct {
	rendering.RenderProxyBoxWithHitTestBehavior
	color       ui.Color
	isAntiAlias bool
}

func (r *renderColoredBox) setIsAntiAlias(v bool) {
	if r.isAntiAlias != v {
		r.isAntiAlias = v
		r.MarkNeedsPaint()
	}
}

func (r *renderColoredBox) setColor(c ui.Color) {
	if r.color == c {
		return
	}
	r.color = c
	r.MarkNeedsPaint()
}

// Paint paints the color, even when it is transparent, then the child.
func (r *renderColoredBox) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	size := r.Size()
	if size.Width > 0 || size.Height > 0 {
		paint := ui.NewPaint()
		paint.SetColor(r.color)
		paint.SetIsAntiAlias(r.isAntiAlias)
		context.Canvas().DrawRect(ui.NewRectFromLTWH(offset.Dx, offset.Dy, size.Width, size.Height), paint)
	}
	if child := r.Child(); child != nil {
		context.PaintChild(child, offset)
	}
}

// KeyedSubtree is a widget that builds its child; it exists to give the child a
// key, so that a list of children can be told apart.
//
// Flutter: widgets/framework.dart KeyedSubtree
type KeyedSubtree struct {
	StatelessWidget
	child Widget
}

// NewKeyedSubtree creates a widget that builds the child.
//
// Flutter: widgets/framework.dart KeyedSubtree.new
func NewKeyedSubtree(child Widget, opts ...WidgetOpts) *KeyedSubtree {
	return &KeyedSubtree{NewStatelessWidgetBase(firstOpts(opts)), child}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/framework.dart KeyedSubtree.child
func (w *KeyedSubtree) Child() Widget { return w.child }

// Build returns the child.
//
// Flutter: widgets/framework.dart KeyedSubtree.build
func (w *KeyedSubtree) Build(BuildContext) Widget { return w.child }

// NewKeyedSubtreeWrap wraps the child in a KeyedSubtree with a ValueKey of the
// child's key, or of the index when it has none.
//
// Flutter: widgets/framework.dart KeyedSubtree.wrap
func NewKeyedSubtreeWrap(child Widget, childIndex int) *KeyedSubtree {
	var key foundation.Key
	if k := child.Key(); k != nil {
		key = foundation.NewValueKey[foundation.Key](k)
	} else {
		key = foundation.NewValueKey(childIndex)
	}
	return NewKeyedSubtree(child, WidgetOpts{Key: key})
}

// KeyedSubtreeWrapList wraps each item of the list in a KeyedSubtree, with
// indices that start at baseIndex.
//
// Flutter: widgets/framework.dart KeyedSubtree.ensureUniqueKeysForList
func KeyedSubtreeWrapList(items []Widget, baseIndex int) []Widget {
	out := make([]Widget, len(items))
	for i, w := range items {
		out[i] = NewKeyedSubtreeWrap(w, baseIndex+i)
	}
	return out
}
