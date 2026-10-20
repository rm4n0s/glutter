// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
)

// AnimatedTransitionBuilder builds a transition of a child from an animation.
//
// Flutter: widgets/dual_transition_builder.dart AnimatedTransitionBuilder
type AnimatedTransitionBuilder func(bc BuildContext, animation animation.Animation[float64], child Widget) Widget

// DualTransitionBuilderOpts are the optional parameters of
// NewDualTransitionBuilder.
//
// Flutter: widgets/dual_transition_builder.dart DualTransitionBuilder.new
type DualTransitionBuilderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// DualTransitionBuilder builds two transitions, one for the animation going
// forward and one for it going backward, so that a widget can have a different
// transition in each direction. The reverse builder is applied to the child and
// the forward builder to the result.
//
// Flutter: widgets/dual_transition_builder.dart DualTransitionBuilder
type DualTransitionBuilder struct {
	StatefulWidget
	animation      animation.Animation[float64]
	forwardBuilder AnimatedTransitionBuilder
	reverseBuilder AnimatedTransitionBuilder
	child          Widget
}

// NewDualTransitionBuilder creates the widget. The child, which may be nil, is
// given to the reverse builder.
//
// Flutter: widgets/dual_transition_builder.dart DualTransitionBuilder.new
func NewDualTransitionBuilder(animation animation.Animation[float64], forwardBuilder, reverseBuilder AnimatedTransitionBuilder, child Widget, opts ...DualTransitionBuilderOpts) *DualTransitionBuilder {
	o := firstOpts(opts)
	return &DualTransitionBuilder{StatefulWidget: NewStatefulWidgetBase(WidgetOpts(o)), animation: animation,
		forwardBuilder: forwardBuilder, reverseBuilder: reverseBuilder, child: child}
}

// Animation is the animation that drives the transitions.
//
// Flutter: widgets/dual_transition_builder.dart DualTransitionBuilder.animation
func (w *DualTransitionBuilder) Animation() animation.Animation[float64] { return w.animation }

// ForwardBuilder builds the transition that is shown while the animation goes
// forward or is complete.
//
// Flutter: widgets/dual_transition_builder.dart DualTransitionBuilder.forwardBuilder
func (w *DualTransitionBuilder) ForwardBuilder() AnimatedTransitionBuilder { return w.forwardBuilder }

// ReverseBuilder builds the transition that is shown while the animation goes
// in reverse.
//
// Flutter: widgets/dual_transition_builder.dart DualTransitionBuilder.reverseBuilder
func (w *DualTransitionBuilder) ReverseBuilder() AnimatedTransitionBuilder { return w.reverseBuilder }

// Child is the widget that the transitions are applied to.
//
// Flutter: widgets/dual_transition_builder.dart DualTransitionBuilder.child
func (w *DualTransitionBuilder) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/dual_transition_builder.dart DualTransitionBuilder.createState
func (w *DualTransitionBuilder) CreateState() State { return &dualTransitionBuilderState{} }

type dualTransitionBuilderState struct {
	StateOf[*DualTransitionBuilder]
	effectiveStatus animation.AnimationStatus
	forward         *animation.ProxyAnimation
	reverse         *animation.ProxyAnimation
	handle          animation.StatusListenerHandle
}

func (s *dualTransitionBuilderState) InitState() {
	s.forward = animation.NewProxyAnimation(nil)
	s.reverse = animation.NewProxyAnimation(nil)
	s.effectiveStatus = s.Widget().animation.Status()
	s.handle = s.Widget().animation.AddStatusListener(s.animationListener)
	s.updateAnimations()
}

func (s *dualTransitionBuilderState) animationListener(status animation.AnimationStatus) {
	old := s.effectiveStatus
	s.effectiveStatus = calculateEffectiveAnimationStatus(s.effectiveStatus, status)
	if old != s.effectiveStatus {
		s.updateAnimations()
	}
}

func (s *dualTransitionBuilderState) DidUpdateWidget(old *DualTransitionBuilder) {
	if old.animation != s.Widget().animation {
		old.animation.RemoveStatusListener(s.handle)
		s.handle = s.Widget().animation.AddStatusListener(s.animationListener)
		s.animationListener(s.Widget().animation.Status())
	}
}

// calculateEffectiveAnimationStatus keeps the status of the direction that the
// animation started in until it is complete or dismissed.
func calculateEffectiveAnimationStatus(lastEffective, current animation.AnimationStatus) animation.AnimationStatus {
	switch current {
	case animation.AnimationStatusForward:
		if lastEffective == animation.AnimationStatusReverse {
			return lastEffective
		}
	case animation.AnimationStatusReverse:
		if lastEffective == animation.AnimationStatusForward {
			return lastEffective
		}
	}
	return current
}

func (s *dualTransitionBuilderState) updateAnimations() {
	switch s.effectiveStatus {
	case animation.AnimationStatusDismissed, animation.AnimationStatusForward:
		s.forward.SetParent(s.Widget().animation)
		s.reverse.SetParent(animation.AlwaysDismissed)
	case animation.AnimationStatusReverse, animation.AnimationStatusCompleted:
		s.forward.SetParent(animation.AlwaysComplete)
		s.reverse.SetParent(animation.NewReverseAnimation(s.Widget().animation))
	}
}

func (s *dualTransitionBuilderState) Dispose() {
	s.Widget().animation.RemoveStatusListener(s.handle)
}

func (s *dualTransitionBuilderState) Build(bc BuildContext) Widget {
	w := s.Widget()
	return w.forwardBuilder(bc, s.forward, w.reverseBuilder(bc, s.reverse, w.child))
}
