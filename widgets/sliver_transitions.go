// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// SliverFadeTransitionOpts are the optional parameters of NewSliverFadeTransition.
//
// Flutter: widgets/transitions.dart SliverFadeTransition.new
type SliverFadeTransitionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AlwaysIncludeSemantics is whether the semantics of the sliver are included
	// even when it is fully transparent.
	AlwaysIncludeSemantics bool
}

// SliverFadeTransition animates the opacity of a sliver; the sliver is not
// rebuilt or repainted while the opacity changes.
//
// Flutter: widgets/transitions.dart SliverFadeTransition
type SliverFadeTransition struct {
	SingleChildRenderObjectWidget
	opacity                animation.Animation[float64]
	alwaysIncludeSemantics bool
}

// NewSliverFadeTransition creates the widget; the sliver may be nil.
//
// Flutter: widgets/transitions.dart SliverFadeTransition.new
func NewSliverFadeTransition(opacity animation.Animation[float64], sliver Widget, opts ...SliverFadeTransitionOpts) *SliverFadeTransition {
	o := firstOpts(opts)
	return &SliverFadeTransition{NewSingleChildRenderObjectWidgetBase(sliver, WidgetOpts{Key: o.Key}), opacity, o.AlwaysIncludeSemantics}
}

// Opacity is the animation of the opacity of the sliver.
//
// Flutter: widgets/transitions.dart SliverFadeTransition.opacity
func (w *SliverFadeTransition) Opacity() animation.Animation[float64] { return w.opacity }

// AlwaysIncludeSemantics is whether the semantics of the sliver are included
// even when it is fully transparent.
//
// Flutter: widgets/transitions.dart SliverFadeTransition.alwaysIncludeSemantics
func (w *SliverFadeTransition) AlwaysIncludeSemantics() bool { return w.alwaysIncludeSemantics }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/transitions.dart SliverFadeTransition.createRenderObject
func (w *SliverFadeTransition) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverAnimatedOpacity(w.opacity, nil, rendering.RenderSliverAnimatedOpacityOpts{AlwaysIncludeSemantics: w.alwaysIncludeSemantics})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/transitions.dart SliverFadeTransition.updateRenderObject
func (w *SliverFadeTransition) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderSliverAnimatedOpacity)
	r.SetOpacity(w.opacity)
	r.SetAlwaysIncludeSemantics(w.alwaysIncludeSemantics)
}

// SliverAnimatedOpacityOpts are the optional parameters of NewSliverAnimatedOpacity.
//
// Flutter: widgets/implicit_animations.dart SliverAnimatedOpacity.new
type SliverAnimatedOpacityOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// AlwaysIncludeSemantics is whether the semantics of the sliver are included
	// even when it is fully transparent.
	AlwaysIncludeSemantics bool
}

// SliverAnimatedOpacity is the animated version of SliverOpacity: it animates the
// opacity of its sliver when it changes.
//
// Flutter: widgets/implicit_animations.dart SliverAnimatedOpacity
type SliverAnimatedOpacity struct {
	ImplicitlyAnimatedWidget
	sliver                 Widget
	opacity                float64
	alwaysIncludeSemantics bool
}

// NewSliverAnimatedOpacity creates the widget; the sliver may be nil. In debug
// builds it panics with the tag OpacityOutOfRange when the opacity is not
// between 0 and 1.
//
// Flutter: widgets/implicit_animations.dart SliverAnimatedOpacity.new
func NewSliverAnimatedOpacity(opacity float64, duration time.Duration, sliver Widget, opts ...SliverAnimatedOpacityOpts) *SliverAnimatedOpacity {
	o := firstOpts(opts)
	if foundation.DebugMode && !(opacity >= 0 && opacity <= 1) {
		panic(errors.New("OpacityOutOfRange", "the opacity must be between 0.0 and 1.0", "opacity", opacity))
	}
	return &SliverAnimatedOpacity{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		sliver, opacity, o.AlwaysIncludeSemantics}
}

// Sliver is the sliver below this widget in the tree.
//
// Flutter: widgets/implicit_animations.dart SliverAnimatedOpacity.sliver
func (w *SliverAnimatedOpacity) Sliver() Widget { return w.sliver }

// Opacity is the target opacity, from 0 (transparent) to 1 (opaque).
//
// Flutter: widgets/implicit_animations.dart SliverAnimatedOpacity.opacity
func (w *SliverAnimatedOpacity) Opacity() float64 { return w.opacity }

// AlwaysIncludeSemantics is whether the semantics of the sliver are included
// even when it is fully transparent.
//
// Flutter: widgets/implicit_animations.dart SliverAnimatedOpacity.alwaysIncludeSemantics
func (w *SliverAnimatedOpacity) AlwaysIncludeSemantics() bool { return w.alwaysIncludeSemantics }

// CreateState creates the state.
//
// Flutter: widgets/implicit_animations.dart SliverAnimatedOpacity.createState
func (w *SliverAnimatedOpacity) CreateState() State { return &sliverAnimatedOpacityState{} }

type sliverAnimatedOpacityState struct {
	ImplicitlyAnimatedWidgetState[*SliverAnimatedOpacity]
	opacity          *animation.Tween[float64]
	opacityAnimation animation.Animation[float64]
}

func (s *sliverAnimatedOpacityState) ForEachTween(v *TweenVisitor) {
	VisitTween(v, &s.opacity, foundation.Some(s.Widget().opacity), floatEqual, floatTween)
}

func (s *sliverAnimatedOpacityState) DidUpdateTweens() {
	s.opacityAnimation = animation.Drive(s.Animation(), animation.Animatable[float64](s.opacity))
}

func (s *sliverAnimatedOpacityState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewSliverFadeTransition(s.opacityAnimation, w.sliver, SliverFadeTransitionOpts{AlwaysIncludeSemantics: w.alwaysIncludeSemantics})
}
