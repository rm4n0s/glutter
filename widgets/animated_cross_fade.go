package widgets

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// CrossFadeState is the child that an AnimatedCrossFade shows.
//
// Flutter: widgets/animated_cross_fade.dart CrossFadeState
type CrossFadeState int

// The values of CrossFadeState.
const (
	// CrossFadeStateUnspecified is the zero value; it is not a real state.
	//
	// Flutter: no counterpart
	CrossFadeStateUnspecified CrossFadeState = iota
	// CrossFadeStateShowFirst shows the first child and hides the second.
	//
	// Flutter: widgets/animated_cross_fade.dart CrossFadeState.showFirst
	CrossFadeStateShowFirst
	// CrossFadeStateShowSecond shows the second child and hides the first.
	//
	// Flutter: widgets/animated_cross_fade.dart CrossFadeState.showSecond
	CrossFadeStateShowSecond
)

// AnimatedCrossFadeBuilder lays out the two children of an AnimatedCrossFade:
// the one on top is fading in, and the one at the bottom is fading out. The
// keys must be given to the children.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFadeBuilder
type AnimatedCrossFadeBuilder func(topChild Widget, topChildKey foundation.Key, bottomChild Widget, bottomChildKey foundation.Key) Widget

// AnimatedCrossFadeDefaultLayoutBuilder is the layout builder of an
// AnimatedCrossFade that has none: the bottom child is positioned at the top
// and stretched to the width, and the top child decides the size.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.defaultLayoutBuilder
func AnimatedCrossFadeDefaultLayoutBuilder(topChild Widget, topChildKey foundation.Key, bottomChild Widget, bottomChildKey foundation.Key) Widget {
	return NewStack([]Widget{
		NewPositioned(bottomChild, PositionedOpts{Key: bottomChildKey, Left: foundation.Some(0.0), Top: foundation.Some(0.0), Right: foundation.Some(0.0)}),
		NewPositioned(topChild, PositionedOpts{Key: topChildKey}),
	}, StackOpts{ClipBehavior: ui.ClipNone})
}

// AnimatedCrossFadeOpts are the optional parameters of NewAnimatedCrossFade.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.new
type AnimatedCrossFadeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// FirstCurve is the fade-in curve of the first child; the default is
	// Curves.Linear.
	FirstCurve animation.Curve
	// SecondCurve is the fade-in curve of the second child; the default is
	// Curves.Linear.
	SecondCurve animation.Curve
	// SizeCurve is the curve of the size animation; the default is Curves.Linear.
	SizeCurve animation.Curve
	// Alignment is how the children are aligned while the size animates; the
	// default is the top center.
	Alignment painting.AlignmentGeometry
	// ReverseDuration is the duration of the transition from the second child to
	// the first; the default is the duration.
	ReverseDuration foundation.Opt[time.Duration]
	// LayoutBuilder lays out the children; the default is
	// AnimatedCrossFadeDefaultLayoutBuilder.
	LayoutBuilder AnimatedCrossFadeBuilder
	// ExcludeBottomFocus is whether the child that is fading out is excluded
	// from focus; the default is true.
	//
	// Deviation: it has no effect until ExcludeFocus exists (step 10).
	ExcludeBottomFocus foundation.Opt[bool]
	// ClipBehavior is how the children are clipped; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// OnEnd is called every time the animation completes.
	OnEnd func()
}

// AnimatedCrossFade is a widget that cross-fades between two given children and
// animates itself between their sizes.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade
type AnimatedCrossFade struct {
	StatefulWidget
	firstChild         Widget
	secondChild        Widget
	crossFadeState     CrossFadeState
	duration           time.Duration
	reverseDuration    foundation.Opt[time.Duration]
	firstCurve         animation.Curve
	secondCurve        animation.Curve
	sizeCurve          animation.Curve
	alignment          painting.AlignmentGeometry
	layoutBuilder      AnimatedCrossFadeBuilder
	excludeBottomFocus bool
	clipBehavior       ui.Clip
	onEnd              func()
}

// NewAnimatedCrossFade creates the widget.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.new
func NewAnimatedCrossFade(firstChild, secondChild Widget, crossFadeState CrossFadeState, duration time.Duration, opts ...AnimatedCrossFadeOpts) *AnimatedCrossFade {
	o := firstOpts(opts)
	if o.FirstCurve == nil {
		o.FirstCurve = animation.Curves.Linear
	}
	if o.SecondCurve == nil {
		o.SecondCurve = animation.Curves.Linear
	}
	if o.SizeCurve == nil {
		o.SizeCurve = animation.Curves.Linear
	}
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentTopCenter
	}
	if o.LayoutBuilder == nil {
		o.LayoutBuilder = AnimatedCrossFadeDefaultLayoutBuilder
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &AnimatedCrossFade{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), firstChild, secondChild, crossFadeState, duration,
		o.ReverseDuration, o.FirstCurve, o.SecondCurve, o.SizeCurve, o.Alignment, o.LayoutBuilder, o.ExcludeBottomFocus.OrElse(true), o.ClipBehavior, o.OnEnd}
}

// FirstChild is the child that is visible when CrossFadeState is show first.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.firstChild
func (w *AnimatedCrossFade) FirstChild() Widget { return w.firstChild }

// SecondChild is the child that is visible when CrossFadeState is show second.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.secondChild
func (w *AnimatedCrossFade) SecondChild() Widget { return w.secondChild }

// CrossFadeState is the child that is visible.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.crossFadeState
func (w *AnimatedCrossFade) CrossFadeState() CrossFadeState { return w.crossFadeState }

// Duration is the duration of the transition.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.duration
func (w *AnimatedCrossFade) Duration() time.Duration { return w.duration }

// ReverseDuration is the duration of the transition to the first child.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.reverseDuration
func (w *AnimatedCrossFade) ReverseDuration() foundation.Opt[time.Duration] { return w.reverseDuration }

// FirstCurve is the fade-in curve of the first child.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.firstCurve
func (w *AnimatedCrossFade) FirstCurve() animation.Curve { return w.firstCurve }

// SecondCurve is the fade-in curve of the second child.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.secondCurve
func (w *AnimatedCrossFade) SecondCurve() animation.Curve { return w.secondCurve }

// SizeCurve is the curve of the size animation.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.sizeCurve
func (w *AnimatedCrossFade) SizeCurve() animation.Curve { return w.sizeCurve }

// Alignment is how the children are aligned while the size animates.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.alignment
func (w *AnimatedCrossFade) Alignment() painting.AlignmentGeometry { return w.alignment }

// ExcludeBottomFocus is whether the child that is fading out is excluded from
// focus.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.excludeBottomFocus
func (w *AnimatedCrossFade) ExcludeBottomFocus() bool { return w.excludeBottomFocus }

// ClipBehavior is how the children are clipped.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.clipBehavior
func (w *AnimatedCrossFade) ClipBehavior() ui.Clip { return w.clipBehavior }

// OnEnd is called every time the animation completes, or nil.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.onEnd
func (w *AnimatedCrossFade) OnEnd() func() { return w.onEnd }

// CreateState creates the state.
//
// Flutter: widgets/animated_cross_fade.dart AnimatedCrossFade.createState
func (w *AnimatedCrossFade) CreateState() State { return &animatedCrossFadeState{} }

type animatedCrossFadeState struct {
	StateOf[*AnimatedCrossFade]
	TickerProvider
	controller      *animation.AnimationController
	firstAnimation  animation.Animation[float64]
	secondAnimation animation.Animation[float64]
}

func (s *animatedCrossFadeState) InitState() {
	w := s.Widget()
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(w.duration), ReverseDuration: w.reverseDuration})
	if w.crossFadeState == CrossFadeStateShowSecond {
		s.controller.SetValue(1)
	}
	s.firstAnimation = s.initAnimation(w.firstCurve, true)
	s.secondAnimation = s.initAnimation(w.secondCurve, false)
	s.controller.AddStatusListener(func(status animation.AnimationStatus) {
		// Trigger a rebuild, because the build depends on whether the controller
		// is animating, which changes together with the status.
		s.SetState(func() {})
		if status == animation.AnimationStatusCompleted || status == animation.AnimationStatusDismissed {
			if onEnd := s.Widget().onEnd; onEnd != nil {
				onEnd()
			}
		}
	})
}

func (s *animatedCrossFadeState) initAnimation(curve animation.Curve, inverted bool) animation.Animation[float64] {
	result := animation.Drive[float64](s.controller, animation.NewCurveTween(curve))
	if inverted {
		result = animation.Drive[float64](result, animation.NewTween(1.0, 0.0))
	}
	return result
}

func (s *animatedCrossFadeState) Dispose() { s.controller.Dispose() }

func (s *animatedCrossFadeState) DidUpdateWidget(old *AnimatedCrossFade) {
	w := s.Widget()
	if w.duration != old.duration {
		s.controller.Duration = foundation.Some(w.duration)
	}
	if w.reverseDuration != old.reverseDuration {
		s.controller.ReverseDuration = w.reverseDuration
	}
	if !animation.CurveEqual(w.firstCurve, old.firstCurve) {
		s.firstAnimation = s.initAnimation(w.firstCurve, true)
	}
	if !animation.CurveEqual(w.secondCurve, old.secondCurve) {
		s.secondAnimation = s.initAnimation(w.secondCurve, false)
	}
	if w.crossFadeState != old.crossFadeState {
		switch w.crossFadeState {
		case CrossFadeStateShowFirst:
			s.controller.Reverse()
		case CrossFadeStateShowSecond:
			s.controller.Forward()
		}
	}
}

func (s *animatedCrossFadeState) Build(BuildContext) Widget {
	w := s.Widget()
	firstKey := foundation.NewValueKey(CrossFadeStateShowFirst)
	secondKey := foundation.NewValueKey(CrossFadeStateShowSecond)
	var topKey, bottomKey foundation.Key
	var topChild, bottomChild Widget
	var topAnimation, bottomAnimation animation.Animation[float64]
	if animation.IsForwardOrCompleted[float64](s.controller) {
		topKey, topChild, topAnimation = secondKey, w.secondChild, s.secondAnimation
		bottomKey, bottomChild, bottomAnimation = firstKey, w.firstChild, s.firstAnimation
	} else {
		topKey, topChild, topAnimation = firstKey, w.firstChild, s.firstAnimation
		bottomKey, bottomChild, bottomAnimation = secondKey, w.secondChild, s.secondAnimation
	}

	// Deviation: Flutter puts an ExcludeFocus (excluding when ExcludeBottomFocus)
	// around each FadeTransition; focus arrives in step 10 (SPEC §12.10).
	bottomChild = NewTickerMode(s.controller.IsAnimating(),
		NewIgnorePointer(
			// Always exclude the semantics of the widget that is fading out.
			NewExcludeSemantics(NewFadeTransition(bottomAnimation, bottomChild))),
		TickerModeOpts{Key: bottomKey})
	topChild = NewTickerMode(true, // the top widget always has its animations enabled
		NewIgnorePointer(
			// Always publish the semantics of the widget that is fading in.
			NewExcludeSemantics(NewFadeTransition(topAnimation, topChild), ExcludeSemanticsOpts{Excluding: foundation.Some(false)}),
			IgnorePointerOpts{Ignoring: foundation.Some(false)}),
		TickerModeOpts{Key: topKey})

	return NewClipRect(NewAnimatedSize(w.duration,
		w.layoutBuilder(topChild, topKey, bottomChild, bottomKey),
		AnimatedSizeOpts{Alignment: w.alignment, ReverseDuration: w.reverseDuration, Curve: w.sizeCurve, ClipBehavior: w.clipBehavior}),
		ClipRectOpts{ClipBehavior: w.clipBehavior})
}
