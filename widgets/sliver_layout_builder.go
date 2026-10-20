// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// SliverLayoutWidgetBuilder builds a widget for the constraints that a viewport
// gives to a sliver.
//
// Flutter: widgets/sliver_layout_builder.dart SliverLayoutWidgetBuilder
type SliverLayoutWidgetBuilder func(bc BuildContext, constraints rendering.SliverConstraints) Widget

// SliverLayoutBuilder provides the SliverConstraints of its parent to a builder,
// which builds a sliver. The builder runs at layout time, whenever the
// constraints change or the widget is updated.
//
// Flutter: widgets/sliver_layout_builder.dart SliverLayoutBuilder
type SliverLayoutBuilder struct {
	RenderObjectWidget
	builder SliverLayoutWidgetBuilder
}

// NewSliverLayoutBuilder creates a sliver that builds its child with the
// constraints of its parent.
//
// Flutter: widgets/sliver_layout_builder.dart SliverLayoutBuilder.new
func NewSliverLayoutBuilder(builder SliverLayoutWidgetBuilder, opts ...LayoutBuilderOpts) *SliverLayoutBuilder {
	return &SliverLayoutBuilder{NewRenderObjectWidgetBase(WidgetOpts(firstOpts(opts))), builder}
}

// Builder is the function that builds the child.
//
// Flutter: widgets/layout_builder.dart ConstrainedLayoutBuilder.builder
func (w *SliverLayoutBuilder) Builder() SliverLayoutWidgetBuilder { return w.builder }

func (w *SliverLayoutBuilder) buildLayout(bc BuildContext, info any) Widget {
	return w.builder(bc, info.(rendering.SliverConstraints))
}

func (w *SliverLayoutBuilder) updateShouldRebuild(Widget) bool { return true }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/sliver_layout_builder.dart SliverLayoutBuilder.createRenderObject
func (w *SliverLayoutBuilder) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := &renderSliverLayoutBuilder{}
	rendering.InitRenderProxySliver(&r.RenderProxySliver, r, nil)
	rendering.InitLayoutCallbackMixin(&r.LayoutCallbackMixin, r, r.layoutCallback)
	return r
}

// CreateElement creates the element, which builds the child during layout.
//
// Flutter: widgets/layout_builder.dart AbstractLayoutBuilder.createElement
func (w *SliverLayoutBuilder) CreateElement() Element {
	e := &layoutBuilderElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// renderSliverLayoutBuilder is the render object of a SliverLayoutBuilder.
//
// Flutter: widgets/sliver_layout_builder.dart _RenderSliverLayoutBuilder
type renderSliverLayoutBuilder struct {
	rendering.RenderProxySliver
	rendering.LayoutCallbackMixin
	callback func(rendering.Constraints)
}

func (r *renderSliverLayoutBuilder) updateCallback(f func(rendering.Constraints)) {
	hadCallback := r.callback != nil
	r.callback = f
	if !hadCallback {
		r.ScheduleLayoutCallback()
	}
}

func (r *renderSliverLayoutBuilder) clearCallback()          { r.callback = nil }
func (r *renderSliverLayoutBuilder) scheduleLayoutCallback() { r.ScheduleLayoutCallback() }
func (r *renderSliverLayoutBuilder) layoutInfo() any         { return r.SliverConstraints() }

func (r *renderSliverLayoutBuilder) layoutCallback() {
	if r.callback != nil {
		r.callback(r.Constraints())
	}
}

// PerformLayout runs the layout callback, then lays the child out with the same
// constraints and takes its geometry.
func (r *renderSliverLayoutBuilder) PerformLayout() {
	r.RunLayoutCallback()
	if child := r.Child(); child != nil {
		child.Layout(r.SliverConstraints(), rendering.LayoutOpts{ParentUsesSize: true})
		r.SetGeometry(child.Geometry())
		return
	}
	r.SetGeometry(rendering.SliverGeometryZero)
}

// Paint paints the child if it is visible: this object adds no offset.
func (r *renderSliverLayoutBuilder) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if child := r.Child(); child != nil && child.Geometry().Visible {
		context.PaintChild(child, offset)
	}
}
