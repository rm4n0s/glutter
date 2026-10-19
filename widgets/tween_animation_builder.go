package widgets

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
)

// TweenAnimationBuilderOpts are the optional parameters of
// NewTweenAnimationBuilder.
//
// Flutter: widgets/tween_animation_builder.dart TweenAnimationBuilder.new
type TweenAnimationBuilderOpts[T any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// OnEnd is called every time an animation completes.
	OnEnd func()
	// Equal compares two values of the tween to see whether the target changed;
	// the default is foundation.ValuesEqual (Go's == for comparable values).
	// Give one for a type that has its own notion of equality, such as a
	// pointer to a struct.
	Equal func(a, b T) bool
}

// TweenAnimationBuilder is a widget builder that animates a property of its
// subtree to a target value whenever the target changes: the tween's end is the
// target, and the builder is called with the animated value on every frame of
// the animation.
//
// The tween is kept, and changed, by the widget: pass a new one only to change
// how the value is interpolated for the first time. A tween made with
// animation.NewTweenEnd (or NewTweenFuncEnd) has no beginning: the widget starts
// at its end, so nothing animates at first.
//
// Flutter: widgets/tween_animation_builder.dart TweenAnimationBuilder
type TweenAnimationBuilder[T any] struct {
	ImplicitlyAnimatedWidget
	tween   *animation.Tween[T]
	builder ValueWidgetBuilder[T]
	child   Widget
	equal   func(a, b T) bool
}

// NewTweenAnimationBuilder creates the widget. The child, which may be nil, is
// the part of the subtree that does not depend on the value: it is given to the
// builder.
//
// Flutter: widgets/tween_animation_builder.dart TweenAnimationBuilder.new
func NewTweenAnimationBuilder[T any](tween *animation.Tween[T], duration time.Duration, builder ValueWidgetBuilder[T], child Widget,
	opts ...TweenAnimationBuilderOpts[T]) *TweenAnimationBuilder[T] {
	o := firstOpts(opts)
	if o.Equal == nil {
		o.Equal = func(a, b T) bool { return foundation.ValuesEqual(a, b) }
	}
	return &TweenAnimationBuilder[T]{NewImplicitlyAnimatedWidgetBase(duration, ImplicitlyAnimatedWidgetOpts{Key: o.Key, Curve: o.Curve, OnEnd: o.OnEnd}),
		tween, builder, child, o.Equal}
}

// Tween is the tween that interpolates the value; its end is the target.
//
// Flutter: widgets/tween_animation_builder.dart TweenAnimationBuilder.tween
func (w *TweenAnimationBuilder[T]) Tween() *animation.Tween[T] { return w.tween }

// Builder is the function that builds the subtree with the animated value.
//
// Flutter: widgets/tween_animation_builder.dart TweenAnimationBuilder.builder
func (w *TweenAnimationBuilder[T]) Builder() ValueWidgetBuilder[T] { return w.builder }

// Child is the part of the subtree that does not depend on the value.
//
// Flutter: widgets/tween_animation_builder.dart TweenAnimationBuilder.child
func (w *TweenAnimationBuilder[T]) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/tween_animation_builder.dart TweenAnimationBuilder.createState
func (w *TweenAnimationBuilder[T]) CreateState() State { return &tweenAnimationBuilderState[T]{} }

type tweenAnimationBuilderState[T any] struct {
	AnimatedWidgetBaseState[*TweenAnimationBuilder[T]]
	current *animation.Tween[T]
}

func (s *tweenAnimationBuilderState[T]) InitState() {
	s.current = s.Widget().tween
	s.current.FillBegin()
	s.AnimatedWidgetBaseState.InitState()
	if !s.Widget().equal(s.current.Begin, s.current.End) {
		s.controller.Forward()
	}
}

func (s *tweenAnimationBuilderState[T]) ForEachTween(v *TweenVisitor) {
	VisitTween(v, &s.current, foundation.Some(s.Widget().tween.End), s.Widget().equal,
		// The constructor is never called: the current tween is never nil.
		func(end T) *animation.Tween[T] { return animation.NewTweenEnd(end) })
}

func (s *tweenAnimationBuilderState[T]) Build(bc BuildContext) Widget {
	return s.Widget().builder(bc, s.current.Evaluate(s.Animation()), s.Widget().child)
}
