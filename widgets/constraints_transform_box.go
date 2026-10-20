// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// ConstraintsTransformBoxOpts are the optional parameters of NewConstraintsTransformBox.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.new
type ConstraintsTransformBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// TextDirection is the direction used to resolve a directional alignment; the default is the one of the closest Directionality.
	TextDirection ui.TextDirection
	// Alignment is how to align the child; the default is Alignment.center.
	Alignment painting.AlignmentGeometry
	// ClipBehavior is whether the child is clipped when it overflows; the default is ClipNone.
	ClipBehavior ui.Clip
}

// ConstraintsTransformBox applies a BoxConstraintsTransform to the constraints its parent gives, before laying out its child.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox
type ConstraintsTransformBox struct {
	SingleChildRenderObjectWidget
	constraintsTransform rendering.BoxConstraintsTransform
	textDirection        ui.TextDirection
	alignment            painting.AlignmentGeometry
	clipBehavior         ui.Clip
}

// NewConstraintsTransformBox creates the widget.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.new
func NewConstraintsTransformBox(constraintsTransform rendering.BoxConstraintsTransform, child Widget, opts ...ConstraintsTransformBoxOpts) *ConstraintsTransformBox {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	w := &ConstraintsTransformBox{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), constraintsTransform: constraintsTransform, textDirection: o.TextDirection, alignment: o.Alignment, clipBehavior: o.ClipBehavior}
	return w
}

// ConstraintsTransform turns the constraints of the parent into those of the child.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.constraintsTransform
func (w *ConstraintsTransformBox) ConstraintsTransform() rendering.BoxConstraintsTransform {
	return w.constraintsTransform
}

// TextDirection is the direction used to resolve a directional alignment; the default is the one of the closest Directionality.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.textDirection
func (w *ConstraintsTransformBox) TextDirection() ui.TextDirection { return w.textDirection }

// Alignment is how to align the child; the default is Alignment.center.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.alignment
func (w *ConstraintsTransformBox) Alignment() painting.AlignmentGeometry { return w.alignment }

// ClipBehavior is whether the child is clipped when it overflows; the default is ClipNone.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.clipBehavior
func (w *ConstraintsTransformBox) ClipBehavior() ui.Clip { return w.clipBehavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.createRenderObject
func (w *ConstraintsTransformBox) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderConstraintsTransformBox(w.alignment, firstDirection(w.textDirection, bc), w.constraintsTransform, nil,
		rendering.RenderConstraintsTransformBoxOpts{ClipBehavior: w.clipBehavior})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart ConstraintsTransformBox.updateRenderObject
func (w *ConstraintsTransformBox) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderConstraintsTransformBox)
	r.SetAlignment(w.alignment)
	r.SetTextDirection(firstDirection(w.textDirection, bc))
	r.SetConstraintsTransform(w.constraintsTransform)
	r.SetClipBehavior(w.clipBehavior)
}

// UnconstrainedBoxOpts are the optional parameters of NewUnconstrainedBox.
//
// Flutter: widgets/basic.dart UnconstrainedBox.new
type UnconstrainedBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// TextDirection is the direction used to resolve a directional alignment.
	TextDirection ui.TextDirection
	// Alignment is how to align the child; the default is Alignment.center.
	Alignment painting.AlignmentGeometry
	// ConstrainedAxis is the axis on which to keep the constraints, if any; the zero value keeps neither.
	ConstrainedAxis painting.Axis
	// ClipBehavior is whether the child is clipped when it overflows; the default is ClipNone.
	ClipBehavior ui.Clip
}

// UnconstrainedBox imposes no constraints on its child, allowing it to render at its natural size.
//
// Flutter: widgets/basic.dart UnconstrainedBox
type UnconstrainedBox struct {
	StatelessWidget
	textDirection   ui.TextDirection
	alignment       painting.AlignmentGeometry
	constrainedAxis painting.Axis
	clipBehavior    ui.Clip
	child           Widget
}

// NewUnconstrainedBox creates the widget.
//
// Flutter: widgets/basic.dart UnconstrainedBox.new
func NewUnconstrainedBox(child Widget, opts ...UnconstrainedBoxOpts) *UnconstrainedBox {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	w := &UnconstrainedBox{StatelessWidget: NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child: child, textDirection: o.TextDirection, alignment: o.Alignment, constrainedAxis: o.ConstrainedAxis, clipBehavior: o.ClipBehavior}
	return w
}

// TextDirection is the direction used to resolve a directional alignment.
//
// Flutter: widgets/basic.dart UnconstrainedBox.textDirection
func (w *UnconstrainedBox) TextDirection() ui.TextDirection { return w.textDirection }

// Alignment is how to align the child; the default is Alignment.center.
//
// Flutter: widgets/basic.dart UnconstrainedBox.alignment
func (w *UnconstrainedBox) Alignment() painting.AlignmentGeometry { return w.alignment }

// ConstrainedAxis is the axis on which to keep the constraints, if any; the zero value keeps neither.
//
// Flutter: widgets/basic.dart UnconstrainedBox.constrainedAxis
func (w *UnconstrainedBox) ConstrainedAxis() painting.Axis { return w.constrainedAxis }

// ClipBehavior is whether the child is clipped when it overflows; the default is ClipNone.
//
// Flutter: widgets/basic.dart UnconstrainedBox.clipBehavior
func (w *UnconstrainedBox) ClipBehavior() ui.Clip { return w.clipBehavior }

// Child is the widget below this one in the tree.
//
// Flutter: widgets/basic.dart UnconstrainedBox.child
func (w *UnconstrainedBox) Child() Widget { return w.child }

// Build builds a ConstraintsTransformBox with the transform of the axis.
//
// Flutter: widgets/basic.dart UnconstrainedBox.build
func (w *UnconstrainedBox) Build(BuildContext) Widget {
	transform := ConstraintsTransformUnconstrained
	switch w.constrainedAxis {
	case painting.AxisHorizontal:
		transform = ConstraintsTransformHeightUnconstrained
	case painting.AxisVertical:
		transform = ConstraintsTransformWidthUnconstrained
	}
	return NewConstraintsTransformBox(transform, w.child, ConstraintsTransformBoxOpts{
		TextDirection: w.textDirection, Alignment: w.alignment, ClipBehavior: w.clipBehavior})
}
