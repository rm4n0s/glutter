// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// SingleChildScrollViewOpts are the optional parameters of
// NewSingleChildScrollView.
//
// Flutter: widgets/single_child_scroll_view.dart SingleChildScrollView.new
type SingleChildScrollViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ScrollDirection is the axis along which the view scrolls; the default is
	// AxisVertical.
	ScrollDirection painting.Axis
	// Reverse is whether the view scrolls against the reading direction.
	Reverse bool
	// Padding is the amount of space by which to inset the child.
	Padding painting.EdgeInsetsGeometry
	// Primary is whether this is the primary scroll view that uses the
	// PrimaryScrollController above; the default is true for a vertical view
	// with no controller.
	Primary foundation.Opt[bool]
	// Physics is how the view responds to the user.
	Physics ScrollPhysics
	// Controller reads and controls the scroll position.
	Controller ScrollController
	// DragStartBehavior is where a drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// ClipBehavior is how the view clips its content; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// HitTestBehavior is how the view behaves during hit testing; the default is
	// HitTestBehaviorOpaque.
	HitTestBehavior rendering.HitTestBehavior
	// RestorationID is the identifier for restoring the scroll offset; it has no
	// effect until state restoration is ported.
	RestorationID string
	// KeyboardDismissBehavior is how the view dismisses the keyboard; the default
	// is the one of the ScrollBehavior.
	KeyboardDismissBehavior ScrollViewKeyboardDismissBehavior
}

// SingleChildScrollView is a box in which a single widget can be scrolled.
//
// Flutter: widgets/single_child_scroll_view.dart SingleChildScrollView
type SingleChildScrollView struct {
	StatelessWidget
	child                   Widget
	scrollDirection         painting.Axis
	reverse                 bool
	padding                 painting.EdgeInsetsGeometry
	primary                 foundation.Opt[bool]
	physics                 ScrollPhysics
	controller              ScrollController
	dragStartBehavior       gestures.DragStartBehavior
	clipBehavior            ui.Clip
	hitTestBehavior         rendering.HitTestBehavior
	restorationID           string
	keyboardDismissBehavior ScrollViewKeyboardDismissBehavior
}

// NewSingleChildScrollView creates a scrollable box around the child, which may
// be nil. In debug builds it panics with the tag ScrollViewPrimaryWithController
// when the view is primary and has a controller.
//
// Flutter: widgets/single_child_scroll_view.dart SingleChildScrollView.new
func NewSingleChildScrollView(child Widget, opts ...SingleChildScrollViewOpts) *SingleChildScrollView {
	o := firstOpts(opts)
	if foundation.DebugMode && !isNil(o.Controller) && o.Primary.OrElse(false) {
		panic(errors.New("ScrollViewPrimaryWithController",
			"primary scroll views obtain their ScrollController via inheritance from a PrimaryScrollController widget; you cannot both set primary to true and pass an explicit controller"))
	}
	if o.ScrollDirection == painting.AxisUnspecified {
		o.ScrollDirection = painting.AxisVertical
	}
	if o.DragStartBehavior == gestures.DragStartBehaviorUnspecified {
		o.DragStartBehavior = gestures.DragStartBehaviorStart
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.HitTestBehavior == rendering.HitTestBehaviorUnspecified {
		o.HitTestBehavior = rendering.HitTestBehaviorOpaque
	}
	return &SingleChildScrollView{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child, o.ScrollDirection, o.Reverse, o.Padding, o.Primary,
		o.Physics, o.Controller, o.DragStartBehavior, o.ClipBehavior, o.HitTestBehavior, o.RestorationID, o.KeyboardDismissBehavior}
}

// Child is the widget that scrolls.
//
// Flutter: widgets/single_child_scroll_view.dart SingleChildScrollView.child
func (w *SingleChildScrollView) Child() Widget { return w.child }

// ScrollDirection is the axis along which the view scrolls.
//
// Flutter: widgets/single_child_scroll_view.dart SingleChildScrollView.scrollDirection
func (w *SingleChildScrollView) ScrollDirection() painting.Axis { return w.scrollDirection }

// Reverse is whether the view scrolls against the reading direction.
//
// Flutter: widgets/single_child_scroll_view.dart SingleChildScrollView.reverse
func (w *SingleChildScrollView) Reverse() bool { return w.reverse }

// Padding is the amount of space by which to inset the child, or nil.
//
// Flutter: widgets/single_child_scroll_view.dart SingleChildScrollView.padding
func (w *SingleChildScrollView) Padding() painting.EdgeInsetsGeometry { return w.padding }

// Controller reads and controls the scroll position.
//
// Flutter: widgets/single_child_scroll_view.dart SingleChildScrollView.controller
func (w *SingleChildScrollView) Controller() ScrollController { return w.controller }

// Primary is whether this is the primary scroll view, or none for the default.
//
// Flutter: widgets/single_child_scroll_view.dart SingleChildScrollView.primary
func (w *SingleChildScrollView) Primary() foundation.Opt[bool] { return w.primary }

// Physics is how the view responds to the user.
//
// Flutter: widgets/single_child_scroll_view.dart SingleChildScrollView.physics
func (w *SingleChildScrollView) Physics() ScrollPhysics { return w.physics }

// Build builds the Scrollable with the viewport of the child.
//
// Flutter: widgets/single_child_scroll_view.dart SingleChildScrollView.build
func (w *SingleChildScrollView) Build(bc BuildContext) Widget {
	axisDirection := AxisDirectionFromAxisReverseAndDirectionality(bc, w.scrollDirection, w.reverse)
	contents := w.child
	if w.padding != nil {
		contents = NewPadding(w.padding, contents)
	}
	effectivePrimary := w.primary.OrElse(isNil(w.controller) && PrimaryScrollControllerShouldInherit(bc, w.scrollDirection))
	scrollController := w.controller
	if effectivePrimary {
		scrollController = PrimaryScrollControllerMaybeOf(bc)
	}
	var scrollable Widget = NewScrollable(func(_ BuildContext, offset rendering.ViewportOffset) Widget {
		return &singleChildViewport{
			SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(contents),
			axisDirection:                 axisDirection, offset: offset, clipBehavior: w.clipBehavior,
		}
	}, ScrollableOpts{
		DragStartBehavior: w.dragStartBehavior, AxisDirection: axisDirection, Controller: scrollController, Physics: w.physics,
		ClipBehavior: w.clipBehavior, HitTestBehavior: w.hitTestBehavior,
	})
	if effectivePrimary && !isNil(scrollController) {
		// Further descendant ScrollViews will not inherit the same PrimaryScrollController.
		scrollable = NewPrimaryScrollControllerNone(scrollable)
	}
	return wrapKeyboardDismiss(bc, w.keyboardDismissBehavior, nil, scrollable)
}

// singleChildViewport is the viewport of a SingleChildScrollView.
//
// Flutter: widgets/single_child_scroll_view.dart _SingleChildViewport
type singleChildViewport struct {
	SingleChildRenderObjectWidget
	axisDirection painting.AxisDirection
	offset        rendering.ViewportOffset
	clipBehavior  ui.Clip
}

func (w *singleChildViewport) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := &renderSingleChildViewport{axisDirection: w.axisDirection, offset: w.offset, clipBehavior: w.clipBehavior,
		clipRectLayer: rendering.NewLayerHandle[*rendering.ClipRectLayer](nil)}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitRenderObjectWithChild(&r.RenderObjectWithChild, r)
	return r
}

func (w *singleChildViewport) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderSingleChildViewport)
	r.setAxisDirection(w.axisDirection)
	r.setOffset(w.offset)
	r.setClipBehavior(w.clipBehavior)
}

func (w *singleChildViewport) CreateElement() Element {
	e := &singleChildViewportElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// singleChildViewportElement counts the notifications that bubble through the
// viewport.
//
// Flutter: widgets/single_child_scroll_view.dart _SingleChildViewportElement
type singleChildViewportElement struct {
	SingleChildRenderObjectElement
}

// OnNotification counts the notification as one that went through a viewport.
func (e *singleChildViewportElement) OnNotification(notification Notification) bool {
	return ViewportElementOnNotification(notification)
}

// renderSingleChildViewport is the render object of a SingleChildScrollView.
//
// Flutter: widgets/single_child_scroll_view.dart _RenderSingleChildViewport
type renderSingleChildViewport struct {
	rendering.RenderBox
	rendering.RenderObjectWithChild[rendering.Box]
	axisDirection painting.AxisDirection
	offset        rendering.ViewportOffset
	offsetHandle  foundation.ListenerHandle
	clipBehavior  ui.Clip
	clipRectLayer *rendering.LayerHandle[*rendering.ClipRectLayer]
}

func (r *renderSingleChildViewport) setAxisDirection(value painting.AxisDirection) {
	if value == r.axisDirection {
		return
	}
	r.axisDirection = value
	r.MarkNeedsLayout()
}

func (r *renderSingleChildViewport) axis() painting.Axis {
	return painting.AxisDirectionToAxis(r.axisDirection)
}

func (r *renderSingleChildViewport) setOffset(value rendering.ViewportOffset) {
	if value == r.offset {
		return
	}
	if r.Attached() {
		r.offset.RemoveListener(r.offsetHandle)
	}
	r.offset = value
	if r.Attached() {
		r.offsetHandle = r.offset.AddListener(r.hasScrolled)
	}
	r.MarkNeedsLayout()
}

// ClipBehavior is how the viewport clips its child.
//
// Flutter: widgets/single_child_scroll_view.dart _RenderSingleChildViewport.clipBehavior
func (r *renderSingleChildViewport) ClipBehavior() ui.Clip { return r.clipBehavior }

func (r *renderSingleChildViewport) setClipBehavior(value ui.Clip) {
	if value != r.clipBehavior {
		r.clipBehavior = value
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

func (r *renderSingleChildViewport) hasScrolled() {
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderSingleChildViewport) Attach(owner *rendering.PipelineOwner) {
	r.RenderBox.Attach(owner)
	r.offsetHandle = r.offset.AddListener(r.hasScrolled)
}

func (r *renderSingleChildViewport) Detach() {
	r.offset.RemoveListener(r.offsetHandle)
	r.RenderBox.Detach()
}

func (r *renderSingleChildViewport) IsRepaintBoundary() bool { return true }

func (r *renderSingleChildViewport) viewportExtent() float64 {
	if r.axis() == painting.AxisHorizontal {
		return r.Size().Width
	}
	return r.Size().Height
}

func (r *renderSingleChildViewport) minScrollExtent() float64 { return 0 }

func (r *renderSingleChildViewport) maxScrollExtent() float64 {
	child := r.Child()
	if isNil(child) {
		return 0
	}
	if r.axis() == painting.AxisHorizontal {
		return math.Max(0, child.Size().Width-r.Size().Width)
	}
	return math.Max(0, child.Size().Height-r.Size().Height)
}

func (r *renderSingleChildViewport) innerConstraints(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	if r.axis() == painting.AxisHorizontal {
		return constraints.HeightConstraints()
	}
	return constraints.WidthConstraints()
}

func (r *renderSingleChildViewport) ComputeMinIntrinsicWidth(height float64) float64 {
	if child := r.Child(); !isNil(child) {
		return child.GetMinIntrinsicWidth(height)
	}
	return 0
}

func (r *renderSingleChildViewport) ComputeMaxIntrinsicWidth(height float64) float64 {
	if child := r.Child(); !isNil(child) {
		return child.GetMaxIntrinsicWidth(height)
	}
	return 0
}

func (r *renderSingleChildViewport) ComputeMinIntrinsicHeight(width float64) float64 {
	if child := r.Child(); !isNil(child) {
		return child.GetMinIntrinsicHeight(width)
	}
	return 0
}

func (r *renderSingleChildViewport) ComputeMaxIntrinsicHeight(width float64) float64 {
	if child := r.Child(); !isNil(child) {
		return child.GetMaxIntrinsicHeight(width)
	}
	return 0
}

func (r *renderSingleChildViewport) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	child := r.Child()
	if isNil(child) {
		return constraints.Smallest()
	}
	return constraints.Constrain(child.GetDryLayout(r.innerConstraints(constraints)))
}

func (r *renderSingleChildViewport) PerformLayout() {
	constraints := r.BoxConstraints()
	if child := r.Child(); isNil(child) {
		r.SetSize(constraints.Smallest())
	} else {
		child.Layout(r.innerConstraints(constraints), rendering.LayoutOpts{ParentUsesSize: true})
		r.SetSize(constraints.Constrain(child.Size()))
	}
	if r.offset.HasPixels() {
		if r.offset.Pixels() > r.maxScrollExtent() {
			r.offset.CorrectBy(r.maxScrollExtent() - r.offset.Pixels())
		} else if r.offset.Pixels() < r.minScrollExtent() {
			r.offset.CorrectBy(r.minScrollExtent() - r.offset.Pixels())
		}
	}
	r.offset.ApplyViewportDimension(r.viewportExtent())
	r.offset.ApplyContentDimensions(r.minScrollExtent(), r.maxScrollExtent())
}

func (r *renderSingleChildViewport) paintOffset() ui.Offset {
	return r.paintOffsetForPosition(r.offset.Pixels())
}

func (r *renderSingleChildViewport) paintOffsetForPosition(position float64) ui.Offset {
	child := r.Child()
	switch r.axisDirection {
	case painting.AxisDirectionUp:
		return ui.Offset{Dx: 0, Dy: position - child.Size().Height + r.Size().Height}
	case painting.AxisDirectionLeft:
		return ui.Offset{Dx: position - child.Size().Width + r.Size().Width, Dy: 0}
	case painting.AxisDirectionRight:
		return ui.Offset{Dx: -position, Dy: 0}
	}
	return ui.Offset{Dx: 0, Dy: -position}
}

func (r *renderSingleChildViewport) shouldClipAtPaintOffset(paintOffset ui.Offset) bool {
	if r.clipBehavior == ui.ClipNone {
		return false
	}
	child := r.Child()
	return paintOffset.Dx < 0 || paintOffset.Dy < 0 || paintOffset.Dx+child.Size().Width > r.Size().Width ||
		paintOffset.Dy+child.Size().Height > r.Size().Height
}

func (r *renderSingleChildViewport) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	child := r.Child()
	if isNil(child) {
		return
	}
	paintOffset := r.paintOffset()
	paintContents := func(context *rendering.PaintingContext, offset ui.Offset) {
		context.PaintChild(child, offset.Add(paintOffset))
	}
	if r.shouldClipAtPaintOffset(paintOffset) {
		r.clipRectLayer.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()), paintContents,
			rendering.PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: r.clipRectLayer.Layer()}))
	} else {
		r.clipRectLayer.SetLayer(nil)
		paintContents(context, offset)
	}
}

func (r *renderSingleChildViewport) Dispose() {
	r.clipRectLayer.SetLayer(nil)
	r.RenderBox.Dispose()
}

func (r *renderSingleChildViewport) ApplyPaintTransform(_ rendering.RenderObject, transform *ui.Matrix4) {
	paintOffset := r.paintOffset()
	transform.Translate(float32(paintOffset.Dx), float32(paintOffset.Dy), 0)
}

func (r *renderSingleChildViewport) DescribeApproximatePaintClip(child rendering.RenderObject) foundation.Opt[ui.Rect] {
	if !isNil(child) && r.shouldClipAtPaintOffset(r.paintOffset()) {
		return foundation.Some(ui.OffsetZero.And(r.Size()))
	}
	return foundation.None[ui.Rect]()
}

func (r *renderSingleChildViewport) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	child := r.Child()
	if isNil(child) {
		return false
	}
	paintOffset := r.paintOffset()
	return result.AddWithPaintOffset(&paintOffset, position, func(result *rendering.BoxHitTestResult, transformed ui.Offset) bool {
		return child.HitTest(result, transformed)
	})
}

// GetOffsetToReveal is the scroll offset that puts the target at the alignment.
func (r *renderSingleChildViewport) GetOffsetToReveal(target rendering.RenderObject, alignment float64, opts ...rendering.GetOffsetToRevealOpts) rendering.RevealedOffset {
	var o rendering.GetOffsetToRevealOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	rect := o.Rect.OrElse(target.PaintBounds())
	targetBox, ok := target.(rendering.Box)
	if !ok {
		return rendering.RevealedOffset{Offset: r.offset.Pixels(), Rect: rect}
	}
	child := r.Child()
	transform := targetBox.GetTransformTo(child)
	bounds := transform.TransformRect(rect)
	contentSize := child.Size()
	var mainAxisExtent, leadingScrollOffset, targetMainAxisExtent float64
	switch r.axisDirection {
	case painting.AxisDirectionUp:
		mainAxisExtent, leadingScrollOffset, targetMainAxisExtent = r.Size().Height, contentSize.Height-bounds.Bottom, bounds.Height()
	case painting.AxisDirectionLeft:
		mainAxisExtent, leadingScrollOffset, targetMainAxisExtent = r.Size().Width, contentSize.Width-bounds.Right, bounds.Width()
	case painting.AxisDirectionRight:
		mainAxisExtent, leadingScrollOffset, targetMainAxisExtent = r.Size().Width, bounds.Left, bounds.Width()
	default:
		mainAxisExtent, leadingScrollOffset, targetMainAxisExtent = r.Size().Height, bounds.Top, bounds.Height()
	}
	targetOffset := leadingScrollOffset - (mainAxisExtent-targetMainAxisExtent)*alignment
	targetRect := bounds.Shift(r.paintOffsetForPosition(targetOffset))
	return rendering.RevealedOffset{Offset: targetOffset, Rect: targetRect}
}

func (r *renderSingleChildViewport) ShowOnScreen(opts rendering.ShowOnScreenOpts) {
	if !r.offset.AllowImplicitScrolling() {
		r.RenderBox.ShowOnScreen(opts)
		return
	}
	newRect := rendering.RenderViewportShowInViewport(opts.Descendant, opts.Rect, r, r.offset, opts)
	r.RenderBox.ShowOnScreen(rendering.ShowOnScreenOpts{Rect: newRect, Duration: opts.Duration, Curve: opts.Curve})
}

func (r *renderSingleChildViewport) DescribeSemanticsClip(rendering.RenderObject) foundation.Opt[ui.Rect] {
	remainingOffset := r.maxScrollExtent() - r.offset.Pixels()
	b := r.SemanticBounds()
	pixels := r.offset.Pixels()
	switch r.axisDirection {
	case painting.AxisDirectionUp:
		return foundation.Some(ui.NewRectFromLTRB(b.Left, b.Top-remainingOffset, b.Right, b.Bottom+pixels))
	case painting.AxisDirectionRight:
		return foundation.Some(ui.NewRectFromLTRB(b.Left-pixels, b.Top, b.Right+remainingOffset, b.Bottom))
	case painting.AxisDirectionLeft:
		return foundation.Some(ui.NewRectFromLTRB(b.Left-remainingOffset, b.Top, b.Right+pixels, b.Bottom))
	}
	return foundation.Some(ui.NewRectFromLTRB(b.Left, b.Top-pixels, b.Right, b.Bottom+remainingOffset))
}
