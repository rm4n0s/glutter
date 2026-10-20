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

// FlexOpts are the optional parameters of NewFlex.
//
// Flutter: widgets/basic.dart Flex.new
type FlexOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// MainAxisAlignment is how the children are placed along the main axis; the default is MainAxisAlignmentStart.
	MainAxisAlignment rendering.MainAxisAlignment
	// MainAxisSize is how much space is occupied along the main axis; the default is MainAxisSizeMax.
	MainAxisSize rendering.MainAxisSize
	// CrossAxisAlignment is how the children are placed along the cross axis; the default is CrossAxisAlignmentCenter.
	CrossAxisAlignment rendering.CrossAxisAlignment
	// TextDirection is the order in which a horizontal flex lays out its children; the default is the one of the closest Directionality.
	TextDirection ui.TextDirection
	// VerticalDirection is the order in which a vertical flex lays out its children; the default is VerticalDirectionDown.
	VerticalDirection painting.VerticalDirection
	// TextBaseline is the baseline to use when the cross axis alignment is baseline.
	TextBaseline ui.TextBaseline
	// ClipBehavior is whether the content is clipped when it overflows; the default is ClipNone.
	ClipBehavior ui.Clip
	// Spacing is the space between the children along the main axis.
	Spacing float64
}

// Flex displays its children in a one-dimensional array. Use Row or Column when the direction is known.
//
// Flutter: widgets/basic.dart Flex
type Flex struct {
	MultiChildRenderObjectWidget
	direction          painting.Axis
	mainAxisAlignment  rendering.MainAxisAlignment
	mainAxisSize       rendering.MainAxisSize
	crossAxisAlignment rendering.CrossAxisAlignment
	textDirection      ui.TextDirection
	verticalDirection  painting.VerticalDirection
	textBaseline       ui.TextBaseline
	clipBehavior       ui.Clip
	spacing            float64
}

// NewFlex creates the widget.
//
// Flutter: widgets/basic.dart Flex.new
func NewFlex(direction painting.Axis, children []Widget, opts ...FlexOpts) *Flex {
	o := firstOpts(opts)
	if o.MainAxisAlignment == rendering.MainAxisAlignmentUnspecified {
		o.MainAxisAlignment = rendering.MainAxisAlignmentStart
	}
	if o.MainAxisSize == rendering.MainAxisSizeUnspecified {
		o.MainAxisSize = rendering.MainAxisSizeMax
	}
	if o.CrossAxisAlignment == rendering.CrossAxisAlignmentUnspecified {
		o.CrossAxisAlignment = rendering.CrossAxisAlignmentCenter
	}
	if o.VerticalDirection == painting.VerticalDirectionUnspecified {
		o.VerticalDirection = painting.VerticalDirectionDown
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	w := &Flex{MultiChildRenderObjectWidget: NewMultiChildRenderObjectWidgetBase(children, WidgetOpts{Key: o.Key}), direction: direction, mainAxisAlignment: o.MainAxisAlignment, mainAxisSize: o.MainAxisSize, crossAxisAlignment: o.CrossAxisAlignment, textDirection: o.TextDirection, verticalDirection: o.VerticalDirection, textBaseline: o.TextBaseline, clipBehavior: o.ClipBehavior, spacing: o.Spacing}
	return w
}

// Direction is the direction to use as the main axis.
//
// Flutter: widgets/basic.dart Flex.direction
func (w *Flex) Direction() painting.Axis { return w.direction }

// MainAxisAlignment is how the children are placed along the main axis; the default is MainAxisAlignmentStart.
//
// Flutter: widgets/basic.dart Flex.mainAxisAlignment
func (w *Flex) MainAxisAlignment() rendering.MainAxisAlignment { return w.mainAxisAlignment }

// MainAxisSize is how much space is occupied along the main axis; the default is MainAxisSizeMax.
//
// Flutter: widgets/basic.dart Flex.mainAxisSize
func (w *Flex) MainAxisSize() rendering.MainAxisSize { return w.mainAxisSize }

// CrossAxisAlignment is how the children are placed along the cross axis; the default is CrossAxisAlignmentCenter.
//
// Flutter: widgets/basic.dart Flex.crossAxisAlignment
func (w *Flex) CrossAxisAlignment() rendering.CrossAxisAlignment { return w.crossAxisAlignment }

// TextDirection is the order in which a horizontal flex lays out its children; the default is the one of the closest Directionality.
//
// Flutter: widgets/basic.dart Flex.textDirection
func (w *Flex) TextDirection() ui.TextDirection { return w.textDirection }

// VerticalDirection is the order in which a vertical flex lays out its children; the default is VerticalDirectionDown.
//
// Flutter: widgets/basic.dart Flex.verticalDirection
func (w *Flex) VerticalDirection() painting.VerticalDirection { return w.verticalDirection }

// TextBaseline is the baseline to use when the cross axis alignment is baseline.
//
// Flutter: widgets/basic.dart Flex.textBaseline
func (w *Flex) TextBaseline() ui.TextBaseline { return w.textBaseline }

// ClipBehavior is whether the content is clipped when it overflows; the default is ClipNone.
//
// Flutter: widgets/basic.dart Flex.clipBehavior
func (w *Flex) ClipBehavior() ui.Clip { return w.clipBehavior }

// Spacing is the space between the children along the main axis.
//
// Flutter: widgets/basic.dart Flex.spacing
func (w *Flex) Spacing() float64 { return w.spacing }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Flex.createRenderObject
func (w *Flex) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	w.debugCheckDirections(bc)
	return rendering.NewRenderFlex(nil, rendering.RenderFlexOpts{Direction: w.direction, MainAxisAlignment: w.mainAxisAlignment,
		MainAxisSize: w.mainAxisSize, CrossAxisAlignment: w.crossAxisAlignment, TextDirection: w.effectiveTextDirection(bc),
		VerticalDirection: w.verticalDirection, TextBaseline: w.textBaseline, ClipBehavior: w.clipBehavior, Spacing: w.spacing})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart Flex.updateRenderObject
func (w *Flex) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	w.debugCheckDirections(bc)
	r := renderObject.(*rendering.RenderFlex)
	r.SetDirection(w.direction)
	r.SetMainAxisAlignment(w.mainAxisAlignment)
	r.SetMainAxisSize(w.mainAxisSize)
	r.SetCrossAxisAlignment(w.crossAxisAlignment)
	r.SetTextDirection(w.effectiveTextDirection(bc))
	r.SetVerticalDirection(w.verticalDirection)
	r.SetTextBaseline(w.textBaseline)
	r.SetClipBehavior(w.clipBehavior)
	r.SetSpacing(w.spacing)
}

func (w *Flex) needTextDirection() bool {
	if w.direction == painting.AxisHorizontal {
		return true
	}
	return w.crossAxisAlignment == rendering.CrossAxisAlignmentStart || w.crossAxisAlignment == rendering.CrossAxisAlignmentEnd
}

func (w *Flex) effectiveTextDirection(bc BuildContext) ui.TextDirection {
	if w.textDirection != ui.TextDirectionUnspecified {
		return w.textDirection
	}
	if w.needTextDirection() {
		return maybeTextDirection(bc)
	}
	return ui.TextDirectionUnspecified
}

// debugCheckDirections panics in debug builds when the flex needs a text
// direction or a text baseline and has none (Flutter's _debugHasNecessaryDirections).
func (w *Flex) debugCheckDirections(bc BuildContext) {
	if !foundation.DebugMode {
		return
	}
	if w.needTextDirection() && w.effectiveTextDirection(bc) == ui.TextDirectionUnspecified {
		panic(errors.New("FlexNeedsTextDirection", foundation.RuntimeTypeName(w)+" needs either a Directionality ancestor or a specified TextDirection",
			"widget", foundation.RuntimeTypeName(w)))
	}
	if w.crossAxisAlignment == rendering.CrossAxisAlignmentBaseline && w.textBaseline == ui.TextBaselineUnspecified {
		panic(errors.New("FlexNeedsTextBaseline", "a Flex with CrossAxisAlignmentBaseline needs a TextBaseline",
			"widget", foundation.RuntimeTypeName(w)))
	}
}

// StackOpts are the optional parameters of NewStack.
//
// Flutter: widgets/basic.dart Stack.new
type StackOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Alignment is how to align the non-positioned children; the default is AlignmentDirectional.topStart.
	Alignment painting.AlignmentGeometry
	// TextDirection is the direction used to resolve a directional alignment; the default is the one of the closest Directionality.
	TextDirection ui.TextDirection
	// Fit is how to size the non-positioned children; the default is StackFitLoose.
	Fit rendering.StackFit
	// ClipBehavior is whether the content is clipped when it overflows; the default is ClipHardEdge.
	ClipBehavior ui.Clip
}

// Stack positions its children relative to the edges of its box.
//
// Flutter: widgets/basic.dart Stack
type Stack struct {
	MultiChildRenderObjectWidget
	alignment     painting.AlignmentGeometry
	textDirection ui.TextDirection
	fit           rendering.StackFit
	clipBehavior  ui.Clip
}

// NewStack creates the widget.
//
// Flutter: widgets/basic.dart Stack.new
func NewStack(children []Widget, opts ...StackOpts) *Stack {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentDirectionalTopStart
	}
	if o.Fit == rendering.StackFitUnspecified {
		o.Fit = rendering.StackFitLoose
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	w := &Stack{MultiChildRenderObjectWidget: NewMultiChildRenderObjectWidgetBase(children, WidgetOpts{Key: o.Key}), alignment: o.Alignment, textDirection: o.TextDirection, fit: o.Fit, clipBehavior: o.ClipBehavior}
	return w
}

// Alignment is how to align the non-positioned children; the default is AlignmentDirectional.topStart.
//
// Flutter: widgets/basic.dart Stack.alignment
func (w *Stack) Alignment() painting.AlignmentGeometry { return w.alignment }

// TextDirection is the direction used to resolve a directional alignment; the default is the one of the closest Directionality.
//
// Flutter: widgets/basic.dart Stack.textDirection
func (w *Stack) TextDirection() ui.TextDirection { return w.textDirection }

// Fit is how to size the non-positioned children; the default is StackFitLoose.
//
// Flutter: widgets/basic.dart Stack.fit
func (w *Stack) Fit() rendering.StackFit { return w.fit }

// ClipBehavior is whether the content is clipped when it overflows; the default is ClipHardEdge.
//
// Flutter: widgets/basic.dart Stack.clipBehavior
func (w *Stack) ClipBehavior() ui.Clip { return w.clipBehavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Stack.createRenderObject
func (w *Stack) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	w.debugCheckDirectionality(bc)
	return rendering.NewRenderStack(nil, rendering.RenderStackOpts{Alignment: w.alignment, TextDirection: firstDirection(w.textDirection, bc),
		Fit: w.fit, ClipBehavior: w.clipBehavior})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart Stack.updateRenderObject
func (w *Stack) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	w.debugCheckDirectionality(bc)
	r := renderObject.(*rendering.RenderStack)
	r.SetAlignment(w.alignment)
	r.SetTextDirection(firstDirection(w.textDirection, bc))
	r.SetFit(w.fit)
	r.SetClipBehavior(w.clipBehavior)
}

// debugCheckDirectionality panics in debug builds when the alignment is
// directional and there is no text direction to resolve it.
func (w *Stack) debugCheckDirectionality(bc BuildContext) {
	if !foundation.DebugMode || w.textDirection != ui.TextDirectionUnspecified {
		return
	}
	if _, ok := w.alignment.(painting.AlignmentDirectional); ok {
		DebugCheckHasDirectionality(bc)
	}
}

// WrapOpts are the optional parameters of NewWrap.
//
// Flutter: widgets/basic.dart Wrap.new
type WrapOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Direction is the direction to use as the main axis; the default is AxisHorizontal.
	Direction painting.Axis
	// Alignment is how the children within a run are placed along the main axis; the default is WrapAlignmentStart.
	Alignment rendering.WrapAlignment
	// Spacing is how much space to place between children in a run in the main axis.
	Spacing float64
	// RunAlignment is how the runs themselves are placed along the cross axis; the default is WrapAlignmentStart.
	RunAlignment rendering.WrapAlignment
	// RunSpacing is how much space to place between the runs themselves in the cross axis.
	RunSpacing float64
	// CrossAxisAlignment is how the children within a run are aligned relative to each other in the cross axis; the default is WrapCrossAlignmentStart.
	CrossAxisAlignment rendering.WrapCrossAlignment
	// TextDirection is the order in which children and runs are laid out horizontally; the default is the one of the closest Directionality.
	TextDirection ui.TextDirection
	// VerticalDirection is the order in which children and runs are laid out vertically; the default is VerticalDirectionDown.
	VerticalDirection painting.VerticalDirection
	// ClipBehavior is whether the content is clipped when it overflows; the default is ClipNone.
	ClipBehavior ui.Clip
}

// Wrap displays its children in multiple horizontal or vertical runs.
//
// Flutter: widgets/basic.dart Wrap
type Wrap struct {
	MultiChildRenderObjectWidget
	direction          painting.Axis
	alignment          rendering.WrapAlignment
	spacing            float64
	runAlignment       rendering.WrapAlignment
	runSpacing         float64
	crossAxisAlignment rendering.WrapCrossAlignment
	textDirection      ui.TextDirection
	verticalDirection  painting.VerticalDirection
	clipBehavior       ui.Clip
}

// NewWrap creates the widget.
//
// Flutter: widgets/basic.dart Wrap.new
func NewWrap(children []Widget, opts ...WrapOpts) *Wrap {
	o := firstOpts(opts)
	if o.Direction == painting.AxisUnspecified {
		o.Direction = painting.AxisHorizontal
	}
	if o.Alignment == rendering.WrapAlignmentUnspecified {
		o.Alignment = rendering.WrapAlignmentStart
	}
	if o.RunAlignment == rendering.WrapAlignmentUnspecified {
		o.RunAlignment = rendering.WrapAlignmentStart
	}
	if o.CrossAxisAlignment == rendering.WrapCrossAlignmentUnspecified {
		o.CrossAxisAlignment = rendering.WrapCrossAlignmentStart
	}
	if o.VerticalDirection == painting.VerticalDirectionUnspecified {
		o.VerticalDirection = painting.VerticalDirectionDown
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	w := &Wrap{MultiChildRenderObjectWidget: NewMultiChildRenderObjectWidgetBase(children, WidgetOpts{Key: o.Key}), direction: o.Direction, alignment: o.Alignment, spacing: o.Spacing, runAlignment: o.RunAlignment, runSpacing: o.RunSpacing, crossAxisAlignment: o.CrossAxisAlignment, textDirection: o.TextDirection, verticalDirection: o.VerticalDirection, clipBehavior: o.ClipBehavior}
	return w
}

// Direction is the direction to use as the main axis; the default is AxisHorizontal.
//
// Flutter: widgets/basic.dart Wrap.direction
func (w *Wrap) Direction() painting.Axis { return w.direction }

// Alignment is how the children within a run are placed along the main axis; the default is WrapAlignmentStart.
//
// Flutter: widgets/basic.dart Wrap.alignment
func (w *Wrap) Alignment() rendering.WrapAlignment { return w.alignment }

// Spacing is how much space to place between children in a run in the main axis.
//
// Flutter: widgets/basic.dart Wrap.spacing
func (w *Wrap) Spacing() float64 { return w.spacing }

// RunAlignment is how the runs themselves are placed along the cross axis; the default is WrapAlignmentStart.
//
// Flutter: widgets/basic.dart Wrap.runAlignment
func (w *Wrap) RunAlignment() rendering.WrapAlignment { return w.runAlignment }

// RunSpacing is how much space to place between the runs themselves in the cross axis.
//
// Flutter: widgets/basic.dart Wrap.runSpacing
func (w *Wrap) RunSpacing() float64 { return w.runSpacing }

// CrossAxisAlignment is how the children within a run are aligned relative to each other in the cross axis; the default is WrapCrossAlignmentStart.
//
// Flutter: widgets/basic.dart Wrap.crossAxisAlignment
func (w *Wrap) CrossAxisAlignment() rendering.WrapCrossAlignment { return w.crossAxisAlignment }

// TextDirection is the order in which children and runs are laid out horizontally; the default is the one of the closest Directionality.
//
// Flutter: widgets/basic.dart Wrap.textDirection
func (w *Wrap) TextDirection() ui.TextDirection { return w.textDirection }

// VerticalDirection is the order in which children and runs are laid out vertically; the default is VerticalDirectionDown.
//
// Flutter: widgets/basic.dart Wrap.verticalDirection
func (w *Wrap) VerticalDirection() painting.VerticalDirection { return w.verticalDirection }

// ClipBehavior is whether the content is clipped when it overflows; the default is ClipNone.
//
// Flutter: widgets/basic.dart Wrap.clipBehavior
func (w *Wrap) ClipBehavior() ui.Clip { return w.clipBehavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Wrap.createRenderObject
func (w *Wrap) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderWrap(nil, rendering.RenderWrapOpts{Direction: w.direction, Alignment: w.alignment, Spacing: w.spacing,
		RunAlignment: w.runAlignment, RunSpacing: w.runSpacing, CrossAxisAlignment: w.crossAxisAlignment,
		TextDirection: firstDirection(w.textDirection, bc), VerticalDirection: w.verticalDirection, ClipBehavior: w.clipBehavior})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart Wrap.updateRenderObject
func (w *Wrap) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderWrap)
	r.SetDirection(w.direction)
	r.SetAlignment(w.alignment)
	r.SetSpacing(w.spacing)
	r.SetRunAlignment(w.runAlignment)
	r.SetRunSpacing(w.runSpacing)
	r.SetCrossAxisAlignment(w.crossAxisAlignment)
	r.SetTextDirection(firstDirection(w.textDirection, bc))
	r.SetVerticalDirection(w.verticalDirection)
	r.SetClipBehavior(w.clipBehavior)
}

// FlowOpts are the optional parameters of NewFlow.
//
// Flutter: widgets/basic.dart Flow.new
type FlowOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ClipBehavior is whether the content is clipped when it overflows; the default is ClipHardEdge.
	ClipBehavior ui.Clip
}

// Flow implements the flow layout algorithm: a delegate positions and paints the children.
//
// Flutter: widgets/basic.dart Flow
type Flow struct {
	MultiChildRenderObjectWidget
	delegate     rendering.FlowDelegate
	clipBehavior ui.Clip
}

// NewFlow creates a flow whose children are wrapped in repaint boundaries, as Flow.new does.
//
// Flutter: widgets/basic.dart Flow.new
func NewFlow(delegate rendering.FlowDelegate, children []Widget, opts ...FlowOpts) *Flow {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	children = RepaintBoundaryWrapAll(children)
	w := &Flow{MultiChildRenderObjectWidget: NewMultiChildRenderObjectWidgetBase(children, WidgetOpts{Key: o.Key}), delegate: delegate, clipBehavior: o.ClipBehavior}
	return w
}

// Delegate controls the size of the flow and the layout of the children.
//
// Flutter: widgets/basic.dart Flow.delegate
func (w *Flow) Delegate() rendering.FlowDelegate { return w.delegate }

// ClipBehavior is whether the content is clipped when it overflows; the default is ClipHardEdge.
//
// Flutter: widgets/basic.dart Flow.clipBehavior
func (w *Flow) ClipBehavior() ui.Clip { return w.clipBehavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Flow.createRenderObject
func (w *Flow) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderFlow(nil, w.delegate, rendering.RenderFlowOpts{ClipBehavior: w.clipBehavior})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart Flow.updateRenderObject
func (w *Flow) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderFlow)
	r.SetDelegate(w.delegate)
	r.SetClipBehavior(w.clipBehavior)
}

// ListBodyOpts are the optional parameters of NewListBody.
//
// Flutter: widgets/basic.dart ListBody.new
type ListBodyOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// MainAxis is the direction to use as the main axis; the default is AxisVertical.
	MainAxis painting.Axis
	// Reverse is whether the children are laid out from the end of the axis.
	Reverse bool
}

// ListBody displays its children sequentially along a given axis, forcing them to the dimensions of the parent in the other axis.
//
// Flutter: widgets/basic.dart ListBody
type ListBody struct {
	MultiChildRenderObjectWidget
	mainAxis painting.Axis
	reverse  bool
}

// NewListBody creates the widget.
//
// Flutter: widgets/basic.dart ListBody.new
func NewListBody(children []Widget, opts ...ListBodyOpts) *ListBody {
	o := firstOpts(opts)
	if o.MainAxis == painting.AxisUnspecified {
		o.MainAxis = painting.AxisVertical
	}
	w := &ListBody{MultiChildRenderObjectWidget: NewMultiChildRenderObjectWidgetBase(children, WidgetOpts{Key: o.Key}), mainAxis: o.MainAxis, reverse: o.Reverse}
	return w
}

// MainAxis is the direction to use as the main axis; the default is AxisVertical.
//
// Flutter: widgets/basic.dart ListBody.mainAxis
func (w *ListBody) MainAxis() painting.Axis { return w.mainAxis }

// Reverse is whether the children are laid out from the end of the axis.
//
// Flutter: widgets/basic.dart ListBody.reverse
func (w *ListBody) Reverse() bool { return w.reverse }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart ListBody.createRenderObject
func (w *ListBody) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderListBody(nil, rendering.RenderListBodyOpts{AxisDirection: w.axisDirection(bc)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart ListBody.updateRenderObject
func (w *ListBody) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderListBody).SetAxisDirection(w.axisDirection(bc))
}

func (w *ListBody) axisDirection(bc BuildContext) painting.AxisDirection {
	return AxisDirectionFromAxisReverseAndDirectionality(bc, w.mainAxis, w.reverse)
}
