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

// CompositedTransformTarget is a widget that a CompositedTransformFollower can
// follow: it provides the anchor that is linked by the LayerLink.
//
// Flutter: widgets/basic.dart CompositedTransformTarget
type CompositedTransformTarget struct {
	SingleChildRenderObjectWidget
	link *rendering.LayerLink
}

// NewCompositedTransformTarget creates the widget; the link is what followers
// use to find the target.
//
// Flutter: widgets/basic.dart CompositedTransformTarget.new
func NewCompositedTransformTarget(link *rendering.LayerLink, child Widget, opts ...WidgetOpts) *CompositedTransformTarget {
	return &CompositedTransformTarget{NewSingleChildRenderObjectWidgetBase(child, firstOpts(opts)), link}
}

// Link is the link that connects this target to its followers.
//
// Flutter: widgets/basic.dart CompositedTransformTarget.link
func (w *CompositedTransformTarget) Link() *rendering.LayerLink { return w.link }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart CompositedTransformTarget.createRenderObject
func (w *CompositedTransformTarget) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderLeaderLayer(w.link, nil)
}

// UpdateRenderObject sets the link.
//
// Flutter: widgets/basic.dart CompositedTransformTarget.updateRenderObject
func (w *CompositedTransformTarget) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderLeaderLayer).SetLink(w.link)
}

// CompositedTransformFollower is a widget that follows a CompositedTransformTarget.
//
// When this widget is composited during the compositing phase (which comes after
// the paint phase, as described in WidgetsBinding.drawFrame), it applies a
// transformation that brings TargetAnchor of the linked
// CompositedTransformTarget and FollowerAnchor of this widget together.
//
// Flutter: widgets/basic.dart CompositedTransformFollower
type CompositedTransformFollower struct {
	SingleChildRenderObjectWidget
	link *rendering.LayerLink
	o    CompositedTransformFollowerOpts
}

// CompositedTransformFollowerOpts are the optional parameters of
// NewCompositedTransformFollower.
//
// Flutter: widgets/basic.dart CompositedTransformFollower.new
type CompositedTransformFollowerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// HideWhenUnlinked hides the follower when the link has no leader (Flutter's
	// showWhenUnlinked is true by default).
	HideWhenUnlinked bool
	// Offset is the additional offset to apply to the follower.
	Offset ui.Offset
	// TargetAnchor is the anchor point on the linked CompositedTransformTarget
	// that FollowerAnchor will line up with; the default is the top left.
	TargetAnchor foundation.Opt[painting.Alignment]
	// FollowerAnchor is the anchor point on this widget that TargetAnchor will
	// line up with; the default is the top left.
	FollowerAnchor foundation.Opt[painting.Alignment]
}

// NewCompositedTransformFollower creates a follower of the link.
//
// Flutter: widgets/basic.dart CompositedTransformFollower.new
func NewCompositedTransformFollower(link *rendering.LayerLink, child Widget, opts ...CompositedTransformFollowerOpts) *CompositedTransformFollower {
	o := firstOpts(opts)
	return &CompositedTransformFollower{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), link: link, o: o}
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart CompositedTransformFollower.createRenderObject
func (w *CompositedTransformFollower) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderFollowerLayer(w.link, nil, rendering.RenderFollowerLayerOpts{
		ShowWhenUnlinked: foundation.Some(!w.o.HideWhenUnlinked), Offset: w.o.Offset, LeaderAnchor: w.o.TargetAnchor, FollowerAnchor: w.o.FollowerAnchor})
}

// UpdateRenderObject sets the link and the anchors.
//
// Flutter: widgets/basic.dart CompositedTransformFollower.updateRenderObject
func (w *CompositedTransformFollower) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderFollowerLayer)
	r.SetLink(w.link)
	r.SetShowWhenUnlinked(!w.o.HideWhenUnlinked)
	r.SetOffset(w.o.Offset)
	r.SetLeaderAnchor(w.o.TargetAnchor.OrElse(painting.AlignmentTopLeft))
	r.SetFollowerAnchor(w.o.FollowerAnchor.OrElse(painting.AlignmentTopLeft))
}
