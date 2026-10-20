// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
)

// ImplicitlyAnimated is what an implicitly animated widget implements: the
// embedded ImplicitlyAnimatedWidget gives Curve, Duration and OnEnd.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidget
type ImplicitlyAnimated interface {
	Widget
	// Curve is the curve of the animation.
	Curve() animation.Curve
	// Duration is the length of the animation.
	Duration() time.Duration
	// OnEnd is called every time an animation completes, or nil.
	OnEnd() func()
}

// ImplicitlyAnimatedWidgetOpts are the optional parameters of
// NewImplicitlyAnimatedWidgetBase.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidget.new
type ImplicitlyAnimatedWidgetOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve to apply when animating the parameters of this widget;
	// the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes. It can be useful to
	// trigger another animation after the previous one ends.
	OnEnd func()
}

// ImplicitlyAnimatedWidget is the embedded base of a widget that animates its
// properties implicitly: whenever the widget is rebuilt with new values, its
// state animates from the old value to the new one. The widget type embeds it;
// its state type embeds ImplicitlyAnimatedWidgetState or
// AnimatedWidgetBaseState.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidget
type ImplicitlyAnimatedWidget struct {
	StatefulWidget
	curve    animation.Curve
	duration time.Duration
	onEnd    func()
}

// NewImplicitlyAnimatedWidgetBase creates the embedded base.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidget.new
func NewImplicitlyAnimatedWidgetBase(duration time.Duration, opts ...ImplicitlyAnimatedWidgetOpts) ImplicitlyAnimatedWidget {
	o := firstOpts(opts)
	if o.Curve == nil {
		o.Curve = animation.Curves.Linear
	}
	return ImplicitlyAnimatedWidget{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), o.Curve, duration, o.OnEnd}
}

// Curve is the curve to apply when animating the parameters of this widget.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidget.curve
func (w *ImplicitlyAnimatedWidget) Curve() animation.Curve { return w.curve }

// Duration is the duration over which to animate the parameters of this widget.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidget.duration
func (w *ImplicitlyAnimatedWidget) Duration() time.Duration { return w.duration }

// OnEnd is called every time an animation completes.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidget.onEnd
func (w *ImplicitlyAnimatedWidget) OnEnd() func() { return w.onEnd }

// TweenVisitor is given to ForEachTween of an implicitly animated state. The
// state calls VisitTween for each of its tweens. The framework runs the visit
// twice: first to create the tweens and see whether any target changed, then,
// if one did, to start every tween from the value it has now.
//
// Flutter: widgets/implicit_animations.dart TweenVisitor
type TweenVisitor struct {
	anim        animation.Animation[float64]
	update      bool
	shouldStart bool
}

// VisitTween visits one tween of an implicitly animated state. The slot is the
// state's field for the tween; the target is the value the widget has now (none
// when the widget doesn't set it, which makes the slot nil); construct makes a
// tween that starts and ends at a value; equal compares two values.
//
// Flutter: widgets/implicit_animations.dart TweenVisitor
func VisitTween[T any](v *TweenVisitor, slot **animation.Tween[T], target foundation.Opt[T], equal func(a, b T) bool,
	construct func(value T) *animation.Tween[T]) {
	tv, ok := target.Get()
	if v.update {
		if t := *slot; t != nil {
			t.Begin = t.Evaluate(v.anim)
			if ok {
				t.End = tv
			}
		}
		return
	}
	if !ok {
		*slot = nil
		return
	}
	if *slot == nil {
		*slot = construct(tv)
	}
	if !equal((*slot).End, tv) {
		v.shouldStart = true
	}
}

// TweenOf evaluates the tween (if there is one) at the animation's value.
//
// Flutter: widgets/implicit_animations.dart (tween?.evaluate(animation))
func TweenOf[T any](t *animation.Tween[T], anim animation.Animation[float64]) foundation.Opt[T] {
	if t == nil {
		return foundation.None[T]()
	}
	return foundation.Some(t.Evaluate(anim))
}

type implicitTweener interface{ ForEachTween(v *TweenVisitor) }

type tweensUpdater interface{ DidUpdateTweens() }

// ImplicitlyAnimatedWidgetState is the embedded base of the state of an
// implicitly animated widget. The state type embeds it, implements
// ForEachTween (calling VisitTween for each tween), Build, and optionally
// DidUpdateTweens.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidgetState
type ImplicitlyAnimatedWidgetState[W ImplicitlyAnimated] struct {
	StateOf[W]
	SingleTickerProvider
	controller *animation.AnimationController
	curved     *animation.CurvedAnimation
}

// Controller is the animation controller that runs the animations of the
// state.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidgetState.controller
func (s *ImplicitlyAnimatedWidgetState[W]) Controller() *animation.AnimationController {
	return s.controller
}

// Animation is the animation of the state, with the curve of the widget.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidgetState.animation
func (s *ImplicitlyAnimatedWidgetState[W]) Animation() animation.Animation[float64] { return s.curved }

// InitState creates the controller and the tweens.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidgetState.initState
func (s *ImplicitlyAnimatedWidgetState[W]) InitState() {
	opts := animation.AnimationControllerOpts{Duration: foundation.Some(s.Widget().Duration())}
	if foundation.DebugMode {
		opts.DebugLabel = foundation.DescribeIdentity(s.Widget())
	}
	s.controller = animation.NewAnimationController(&s.SingleTickerProvider, opts)
	s.curved = s.createCurve()
	s.controller.AddStatusListener(func(status animation.AnimationStatus) {
		if status.IsCompleted() {
			if f := s.Widget().OnEnd(); f != nil {
				f()
			}
		}
	})
	s.constructTweens()
	s.didUpdateTweens()
}

// DidUpdateWidget starts an animation when a target changed.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidgetState.didUpdateWidget
func (s *ImplicitlyAnimatedWidgetState[W]) DidUpdateWidget(old W) {
	if !animation.CurveEqual(s.Widget().Curve(), old.Curve()) {
		s.curved.Dispose()
		s.curved = s.createCurve()
	}
	s.controller.Duration = foundation.Some(s.Widget().Duration())
	if s.constructTweens() {
		s.forEachTween(&TweenVisitor{anim: s.curved, update: true})
		s.controller.Forward(0)
		s.didUpdateTweens()
	}
}

// Dispose releases the animation and the controller.
//
// Flutter: widgets/implicit_animations.dart ImplicitlyAnimatedWidgetState.dispose
func (s *ImplicitlyAnimatedWidgetState[W]) Dispose() {
	s.curved.Dispose()
	s.controller.Dispose()
}

func (s *ImplicitlyAnimatedWidgetState[W]) createCurve() *animation.CurvedAnimation {
	return animation.NewCurvedAnimation(s.controller, s.Widget().Curve())
}

func (s *ImplicitlyAnimatedWidgetState[W]) forEachTween(v *TweenVisitor) {
	s.self.(implicitTweener).ForEachTween(v)
}

func (s *ImplicitlyAnimatedWidgetState[W]) didUpdateTweens() {
	if u, ok := s.self.(tweensUpdater); ok {
		u.DidUpdateTweens()
	}
}

// constructTweens creates the tweens and reports whether an animation is
// needed.
func (s *ImplicitlyAnimatedWidgetState[W]) constructTweens() bool {
	v := &TweenVisitor{anim: s.curved}
	s.forEachTween(v)
	return v.shouldStart
}

// AnimatedWidgetBaseState is the embedded base of the state of an implicitly
// animated widget that rebuilds on every tick of the animation.
//
// Flutter: widgets/implicit_animations.dart AnimatedWidgetBaseState
type AnimatedWidgetBaseState[W ImplicitlyAnimated] struct {
	ImplicitlyAnimatedWidgetState[W]
}

// InitState starts listening to the controller.
//
// Flutter: widgets/implicit_animations.dart AnimatedWidgetBaseState.initState
func (s *AnimatedWidgetBaseState[W]) InitState() {
	s.ImplicitlyAnimatedWidgetState.InitState()
	s.controller.AddListener(s.handleAnimationChanged)
}

func (s *AnimatedWidgetBaseState[W]) handleAnimationChanged() {
	// The animation ticked. Rebuild with the new animation value.
	s.SetState(func() {})
}
