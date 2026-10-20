// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package animation

import (
	"fmt"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// AnimatableCallback computes a value from the animation's t.
//
// Flutter: animation/tween.dart AnimatableCallback
type AnimatableCallback[T any] func(value float64) T

// Animatable is an object that maps a double t, usually from 0.0 to 1.0, to a
// value of type T. Tween, CurveTween and TweenSequence are Animatables; embed
// AnimatableBase (see InitAnimatableBase) to get the other methods in a type of
// your own that implements Transform.
//
// Flutter: animation/tween.dart Animatable
type Animatable[T any] interface {
	// Transform returns the value of the animatable at t.
	//
	// Flutter: animation/tween.dart Animatable.transform
	Transform(t float64) T
	// Evaluate returns the value of the animatable at the value of the animation.
	//
	// Flutter: animation/tween.dart Animatable.evaluate
	Evaluate(animation Animation[float64]) T
	// Animate returns an animation of T that follows the parent.
	//
	// Flutter: animation/tween.dart Animatable.animate
	Animate(parent Animation[float64]) Animation[T]
	// Chain returns an animatable that passes t through the parent first.
	//
	// Flutter: animation/tween.dart Animatable.chain
	Chain(parent Animatable[float64]) Animatable[T]
}

// AnimatableBase has Evaluate, Animate and Chain for a type that implements
// Transform. Embed it and call InitAnimatableBase with the outer type.
//
// Flutter: animation/tween.dart Animatable
type AnimatableBase[T any] struct{ self Animatable[T] }

// InitAnimatableBase sets the outer type of an embedded AnimatableBase.
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitAnimatableBase[T any](b *AnimatableBase[T], self Animatable[T]) { b.self = self }

// Evaluate returns the value of the animatable at the value of the animation.
//
// Flutter: animation/tween.dart Animatable.evaluate
func (b *AnimatableBase[T]) Evaluate(animation Animation[float64]) T {
	return b.self.Transform(animation.Value())
}

// Animate returns an animation of T that follows the parent.
//
// Flutter: animation/tween.dart Animatable.animate
func (b *AnimatableBase[T]) Animate(parent Animation[float64]) Animation[T] {
	return &animatedEvaluation[T]{AnimationWithParent: AnimationWithParent[float64]{Parent: parent}, evaluatable: b.self}
}

// Chain returns an animatable that passes t through the parent first.
//
// Flutter: animation/tween.dart Animatable.chain
func (b *AnimatableBase[T]) Chain(parent Animatable[float64]) Animatable[T] {
	return newChainedEvaluation(parent, b.self)
}

type callbackAnimatable[T any] struct {
	AnimatableBase[T]
	callback AnimatableCallback[T]
}

// NewAnimatableFromCallback creates an animatable that calls the function.
//
// Flutter: animation/tween.dart Animatable.fromCallback
func NewAnimatableFromCallback[T any](callback AnimatableCallback[T]) Animatable[T] {
	a := &callbackAnimatable[T]{callback: callback}
	InitAnimatableBase(&a.AnimatableBase, Animatable[T](a))
	return a
}

func (a *callbackAnimatable[T]) Transform(t float64) T { return a.callback(t) }

// AnimationWithParent has the listener methods of an animation that follows a
// parent animation, and its status. Embed it and give the Parent.
//
// Flutter: animation/animations.dart AnimationWithParentMixin
type AnimationWithParent[P any] struct {
	// Parent is the animation that this one follows.
	Parent Animation[P]
}

// AddListener adds the listener to the parent.
//
// Flutter: animation/animations.dart AnimationWithParentMixin.addListener
func (a AnimationWithParent[P]) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	return a.Parent.AddListener(listener)
}

// RemoveListener removes the listener from the parent.
//
// Flutter: animation/animations.dart AnimationWithParentMixin.removeListener
func (a AnimationWithParent[P]) RemoveListener(handle foundation.ListenerHandle) {
	a.Parent.RemoveListener(handle)
}

// AddStatusListener adds the listener to the parent.
//
// Flutter: animation/animations.dart AnimationWithParentMixin.addStatusListener
func (a AnimationWithParent[P]) AddStatusListener(listener AnimationStatusListener) StatusListenerHandle {
	return a.Parent.AddStatusListener(listener)
}

// RemoveStatusListener removes the listener from the parent.
//
// Flutter: animation/animations.dart AnimationWithParentMixin.removeStatusListener
func (a AnimationWithParent[P]) RemoveStatusListener(handle StatusListenerHandle) {
	a.Parent.RemoveStatusListener(handle)
}

// Status is the status of the parent.
//
// Flutter: animation/animations.dart AnimationWithParentMixin.status
func (a AnimationWithParent[P]) Status() AnimationStatus { return a.Parent.Status() }

type animatedEvaluation[T any] struct {
	AnimationWithParent[float64]
	evaluatable Animatable[T]
}

func (a *animatedEvaluation[T]) Value() T { return a.evaluatable.Evaluate(a.Parent) }

func (a *animatedEvaluation[T]) String() string {
	return fmt.Sprintf("%v➩%v➩%v", a.Parent, a.evaluatable, a.Value())
}

type chainedEvaluation[T any] struct {
	AnimatableBase[T]
	parent      Animatable[float64]
	evaluatable Animatable[T]
}

func newChainedEvaluation[T any](parent Animatable[float64], evaluatable Animatable[T]) *chainedEvaluation[T] {
	c := &chainedEvaluation[T]{parent: parent, evaluatable: evaluatable}
	InitAnimatableBase(&c.AnimatableBase, Animatable[T](c))
	return c
}

func (c *chainedEvaluation[T]) Transform(t float64) T {
	return c.evaluatable.Transform(c.parent.Transform(t))
}

func (c *chainedEvaluation[T]) String() string {
	return fmt.Sprintf("%v➩%v", c.parent, c.evaluatable)
}

// Lerper is a type that can interpolate between two of its values: Tween uses
// it when the type of the tween has it (ui.Color, ui.Offset, ui.Size and
// ui.Rect do).
//
// Flutter: animation/tween.dart Tween (the +, - and * operators of T)
type Lerper[T any] interface {
	Lerp(other T, t float64) T
}

// TweenLerp interpolates between two values of the type.
//
// Flutter: animation/tween.dart Tween.lerp
type TweenLerp[T any] func(begin, end T, t float64) T

// Tween is a linear interpolation between a beginning and ending value. Its
// type must be float64 or have a Lerp method (see Lerper), or give a function
// to NewTweenFunc. For int values use NewIntTween or NewStepTween.
//
// Flutter: animation/tween.dart Tween
type Tween[T any] struct {
	AnimatableBase[T]
	// Begin is the value at t = 0.
	//
	// Flutter: animation/tween.dart Tween.begin
	Begin T
	// End is the value at t = 1.
	//
	// Flutter: animation/tween.dart Tween.end
	End  T
	lerp TweenLerp[T]
	name string
	// noBegin is whether the tween was made with only an end (Flutter's null
	// begin): TweenAnimationBuilder starts such a tween at its end.
	noBegin bool
	// isNone says which values are null (Flutter's nullable tweens, such as
	// ColorTween), so that a null begin counts as no begin.
	isNone func(T) bool
}

// NewTween creates a tween from begin to end with the built-in interpolation of
// the type T. It panics with the tag TweenLerpMissing when T is neither float64
// nor a Lerper.
//
// Flutter: animation/tween.dart Tween.new
func NewTween[T any](begin, end T) *Tween[T] {
	lerp, ok := defaultLerp[T]()
	if !ok {
		panic(errors.New("TweenLerpMissing", fmt.Sprintf("cannot lerp between %v and %v; the type %T has no built-in interpolation; give the function to NewTweenFunc or add a Lerp(other, t) method to the type", begin, end, begin),
			"type", fmt.Sprintf("%T", begin)))
	}
	return NewTweenFunc(begin, end, lerp)
}

// NewTweenFunc creates a tween from begin to end that interpolates with the
// function.
//
// Flutter: animation/tween.dart Tween.lerp
func NewTweenFunc[T any](begin, end T, lerp TweenLerp[T]) *Tween[T] {
	t := &Tween[T]{Begin: begin, End: end, lerp: lerp}
	InitAnimatableBase(&t.AnimatableBase, Animatable[T](t))
	return t
}

// NewTweenEnd creates a tween that has only an end, like Flutter's
// Tween(end: x) with a null begin. See NewTween for the interpolation.
//
// Flutter: animation/tween.dart Tween.new
func NewTweenEnd[T any](end T) *Tween[T] { return NewTween(end, end).WithoutBegin() }

// WithoutBegin makes the tween one that has only an end (Flutter's null begin):
// HasBegin is false and FillBegin gives it a beginning. It returns the tween.
//
// Flutter: animation/tween.dart Tween.begin (null)
func (t *Tween[T]) WithoutBegin() *Tween[T] {
	t.noBegin = true
	return t
}

// SetNullable says which values of the tween are null, for the tweens of
// nullable types (ColorTween's none): a null begin counts as no begin, like
// Flutter's `begin ??= end`. It returns the tween.
//
// Flutter: no counterpart (Dart's null)
func (t *Tween[T]) SetNullable(isNone func(T) bool) *Tween[T] {
	t.isNone = isNone
	return t
}

// NewTweenFuncEnd creates a tween that has only an end and interpolates with
// the function.
//
// Flutter: animation/tween.dart Tween.new
func NewTweenFuncEnd[T any](end T, lerp TweenLerp[T]) *Tween[T] {
	return NewTweenFunc(end, end, lerp).WithoutBegin()
}

// SetName sets the name that String shows, such as "AlignmentTween"; a tween of
// your own type uses it as the class name. It returns the tween.
//
// Flutter: no counterpart (the name of the Dart class)
func (t *Tween[T]) SetName(name string) *Tween[T] {
	t.name = name
	return t
}

// HasBegin is false for a tween made with NewTweenEnd or NewTweenFuncEnd until
// FillBegin is called.
//
// Flutter: animation/tween.dart Tween.begin (null)
func (t *Tween[T]) HasBegin() bool {
	return !t.noBegin && (t.isNone == nil || !t.isNone(t.Begin))
}

// FillBegin sets the beginning to the end if the tween has none, which is
// Flutter's `tween.begin ??= tween.end`.
//
// Flutter: animation/tween.dart Tween.begin
func (t *Tween[T]) FillBegin() {
	if !t.HasBegin() {
		t.Begin = t.End
		t.noBegin = false
	}
}

func defaultLerp[T any]() (TweenLerp[T], bool) {
	var zero T
	switch any(zero).(type) {
	case float64:
		f := TweenLerp[float64](func(a, b float64, t float64) float64 { return a + (b-a)*t })
		return any(f).(TweenLerp[T]), true
	}
	if _, ok := any(zero).(Lerper[T]); ok {
		return func(begin, end T, t float64) T { return any(begin).(Lerper[T]).Lerp(end, t) }, true
	}
	return nil, false
}

// Lerp returns the value of the tween at t, without the special cases of
// Transform for 0 and 1.
//
// Flutter: animation/tween.dart Tween.lerp
func (t *Tween[T]) Lerp(at float64) T { return t.lerp(t.Begin, t.End, at) }

// Transform returns Begin at 0, End at 1 and the interpolated value between.
//
// Flutter: animation/tween.dart Tween.transform
func (t *Tween[T]) Transform(at float64) T {
	if at == 0 {
		return t.Begin
	}
	if at == 1 {
		return t.End
	}
	return t.Lerp(at)
}

// String describes the tween.
//
// Flutter: animation/tween.dart Tween.toString
func (t *Tween[T]) String() string {
	name := t.name
	if name == "" {
		name = "Animatable"
	}
	return fmt.Sprintf("%s(%v → %v)", name, t.Begin, t.End)
}

// NewReverseTween creates a tween that runs the parent backwards: it begins at
// the parent's end.
//
// Flutter: animation/tween.dart ReverseTween.new
func NewReverseTween[T any](parent *Tween[T]) *Tween[T] {
	t := NewTweenFunc(parent.End, parent.Begin, func(_, _ T, at float64) T { return parent.Lerp(1 - at) })
	t.name = "ReverseTween"
	return t
}

// NewColorTween creates a tween between two colors. A color that is none counts
// as transparent when the other one is some, as Color.lerp does with null.
//
// Flutter: animation/tween.dart ColorTween
func NewColorTween(begin, end foundation.Opt[ui.Color]) *Tween[foundation.Opt[ui.Color]] {
	t := NewTweenFunc(begin, end, func(b, e foundation.Opt[ui.Color], at float64) foundation.Opt[ui.Color] {
		var pb, pe *ui.Color
		if v, ok := b.Get(); ok {
			pb = &v
		}
		if v, ok := e.Get(); ok {
			pe = &v
		}
		if r := ui.ColorLerp(pb, pe, at); r != nil {
			return foundation.Some(*r)
		}
		return foundation.None[ui.Color]()
	})
	t.name = "ColorTween"
	t.isNone = func(v foundation.Opt[ui.Color]) bool { return v.IsNone() }
	return t
}

// NewSizeTween creates a tween between two sizes. A size that is none counts as
// zero when the other one is some, as Size.lerp does with null.
//
// Flutter: animation/tween.dart SizeTween
func NewSizeTween(begin, end foundation.Opt[ui.Size]) *Tween[foundation.Opt[ui.Size]] {
	t := NewTweenFunc(begin, end, func(b, e foundation.Opt[ui.Size], at float64) foundation.Opt[ui.Size] {
		var pb, pe *ui.Size
		if v, ok := b.Get(); ok {
			pb = &v
		}
		if v, ok := e.Get(); ok {
			pe = &v
		}
		if r := ui.SizeLerp(pb, pe, at); r != nil {
			return foundation.Some(*r)
		}
		return foundation.None[ui.Size]()
	})
	t.name = "SizeTween"
	t.isNone = func(v foundation.Opt[ui.Size]) bool { return v.IsNone() }
	return t
}

// NewRectTween creates a tween between two rectangles. A rectangle that is none
// counts as zero when the other one is some, as Rect.lerp does with null.
//
// Flutter: animation/tween.dart RectTween
func NewRectTween(begin, end foundation.Opt[ui.Rect]) *Tween[foundation.Opt[ui.Rect]] {
	t := NewTweenFunc(begin, end, func(b, e foundation.Opt[ui.Rect], at float64) foundation.Opt[ui.Rect] {
		var pb, pe *ui.Rect
		if v, ok := b.Get(); ok {
			pb = &v
		}
		if v, ok := e.Get(); ok {
			pe = &v
		}
		if r := ui.RectLerp(pb, pe, at); r != nil {
			return foundation.Some(*r)
		}
		return foundation.None[ui.Rect]()
	})
	t.name = "RectTween"
	t.isNone = func(v foundation.Opt[ui.Rect]) bool { return v.IsNone() }
	return t
}

// NewIntTween creates a tween between two ints that rounds each value.
//
// Flutter: animation/tween.dart IntTween
func NewIntTween(begin, end int) *Tween[int] {
	t := NewTweenFunc(begin, end, func(b, e int, at float64) int { return int(math.Round(float64(b) + float64(e-b)*at)) })
	t.name = "IntTween"
	return t
}

// NewStepTween creates a tween between two ints that rounds each value down,
// which makes steps.
//
// Flutter: animation/tween.dart StepTween
func NewStepTween(begin, end int) *Tween[int] {
	t := NewTweenFunc(begin, end, func(b, e int, at float64) int { return int(math.Floor(float64(b) + float64(e-b)*at)) })
	t.name = "StepTween"
	return t
}

// NewConstantTween creates a tween that has the same value at every t.
//
// Flutter: animation/tween.dart ConstantTween
func NewConstantTween[T any](value T) *Tween[T] {
	t := NewTweenFunc(value, value, func(b, _ T, _ float64) T { return b })
	t.name = "ConstantTween"
	return t
}

// CurveTween is an animatable that applies a curve to t.
//
// Flutter: animation/tween.dart CurveTween
type CurveTween struct {
	AnimatableBase[float64]
	// Curve is the curve to apply.
	//
	// Flutter: animation/tween.dart CurveTween.curve
	Curve Curve
}

// NewCurveTween creates a tween that applies the curve.
//
// Flutter: animation/tween.dart CurveTween.new
func NewCurveTween(curve Curve) *CurveTween {
	c := &CurveTween{Curve: curve}
	InitAnimatableBase(&c.AnimatableBase, Animatable[float64](c))
	return c
}

// Transform returns t for 0 and 1 and the curve's value between.
//
// Flutter: animation/tween.dart CurveTween.transform
func (c *CurveTween) Transform(t float64) float64 {
	if t == 0 || t == 1 {
		if foundation.DebugMode && math.Round(c.Curve.Transform(t)) != t {
			panic(errors.New("CurveEndpointInvalid", fmt.Sprintf("the curve %v does not map %v to near %v", c.Curve, t, t), "curve", fmt.Sprint(c.Curve), "t", t))
		}
		return t
	}
	return c.Curve.Transform(t)
}

// String describes the tween.
//
// Flutter: animation/tween.dart CurveTween.toString
func (c *CurveTween) String() string { return fmt.Sprintf("CurveTween(curve: %v)", c.Curve) }
