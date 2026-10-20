package widgets

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// PageTransitionsBuilder builds the transitions of a page route: the way a page
// enters and leaves, and the way the page below it moves out of the way.
// Implementations are comparable values, so that two themes that hold the same
// builders are equal.
//
// Flutter: widgets/page_transitions_builder.dart PageTransitionsBuilder
type PageTransitionsBuilder interface {
	// DelegatedTransition is the transition that a route with this builder
	// gives to the route below it, or nil.
	//
	// Flutter: widgets/page_transitions_builder.dart PageTransitionsBuilder.delegatedTransition
	DelegatedTransition() DelegatedTransitionBuilder
	// TransitionDuration is the duration of the transition in.
	//
	// Flutter: widgets/page_transitions_builder.dart PageTransitionsBuilder.transitionDuration
	TransitionDuration() time.Duration
	// ReverseTransitionDuration is the duration of the transition out.
	//
	// Flutter: widgets/page_transitions_builder.dart PageTransitionsBuilder.reverseTransitionDuration
	ReverseTransitionDuration() time.Duration
	// BuildTransitions builds the transition of the child for the route.
	//
	// Flutter: widgets/page_transitions_builder.dart PageTransitionsBuilder.buildTransitions
	BuildTransitions(route PageRoute, bc BuildContext, animation, secondaryAnimation animation.Animation[float64], child Widget) Widget
}

// PageTransitionsBuilderBase has the defaults of a PageTransitionsBuilder: no
// delegated transition and 300 milliseconds in both directions. A builder that
// has another duration overrides both TransitionDuration and
// ReverseTransitionDuration (Dart's reverse duration follows the duration; the
// base of Go has no way to see the type that embeds it).
//
// Flutter: widgets/page_transitions_builder.dart PageTransitionsBuilder
type PageTransitionsBuilderBase struct{}

// DelegatedTransition is nil.
//
// Flutter: widgets/page_transitions_builder.dart PageTransitionsBuilder.delegatedTransition
func (PageTransitionsBuilderBase) DelegatedTransition() DelegatedTransitionBuilder { return nil }

// TransitionDuration is 300 milliseconds.
//
// Flutter: widgets/page_transitions_builder.dart PageTransitionsBuilder.transitionDuration
func (PageTransitionsBuilderBase) TransitionDuration() time.Duration {
	return 300 * time.Millisecond
}

// ReverseTransitionDuration is 300 milliseconds.
//
// Flutter: widgets/page_transitions_builder.dart PageTransitionsBuilder.reverseTransitionDuration
func (PageTransitionsBuilderBase) ReverseTransitionDuration() time.Duration {
	return 300 * time.Millisecond
}

var (
	fadeUpwardsBottomUpTween    = animation.NewTween(ui.Offset{Dx: 0, Dy: 0.25}, ui.Offset{})
	fadeUpwardsFastOutSlowInTwn = animation.NewCurveTween(animation.Curves.FastOutSlowIn)
	fadeUpwardsEaseInTween      = animation.NewCurveTween(animation.Curves.EaseIn)
)

// fadeUpwardsPageTransition slides the page up by a quarter of its height while
// it fades in.
type fadeUpwardsPageTransition struct {
	StatelessWidget
	positionAnimation animation.Animation[ui.Offset]
	opacityAnimation  animation.Animation[float64]
	child             Widget
}

func newFadeUpwardsPageTransition(routeAnimation animation.Animation[float64], child Widget) *fadeUpwardsPageTransition {
	return &fadeUpwardsPageTransition{
		StatelessWidget:   NewStatelessWidgetBase(),
		positionAnimation: animation.Drive[ui.Offset](routeAnimation, fadeUpwardsBottomUpTween.Chain(fadeUpwardsFastOutSlowInTwn)),
		opacityAnimation:  animation.Drive[float64](routeAnimation, fadeUpwardsEaseInTween),
		child:             child,
	}
}

func (w *fadeUpwardsPageTransition) Build(BuildContext) Widget {
	return NewSlideTransition(w.positionAnimation, NewFadeTransition(w.opacityAnimation, w.child))
}

// FadeUpwardsPageTransitionsBuilder is the transition of a page that slides up
// while it fades in, like the Android Oreo one.
//
// Flutter: widgets/page_transitions_builder.dart FadeUpwardsPageTransitionsBuilder
type FadeUpwardsPageTransitionsBuilder struct{ PageTransitionsBuilderBase }

// NewFadeUpwardsPageTransitionsBuilder creates the builder.
//
// Flutter: widgets/page_transitions_builder.dart FadeUpwardsPageTransitionsBuilder.new
func NewFadeUpwardsPageTransitionsBuilder() FadeUpwardsPageTransitionsBuilder {
	return FadeUpwardsPageTransitionsBuilder{}
}

// BuildTransitions builds the slide up and fade in of the child.
//
// Flutter: widgets/page_transitions_builder.dart FadeUpwardsPageTransitionsBuilder.buildTransitions
func (FadeUpwardsPageTransitionsBuilder) BuildTransitions(_ PageRoute, _ BuildContext, a, _ animation.Animation[float64], child Widget) Widget {
	return newFadeUpwardsPageTransition(a, child)
}

var (
	openUpwardsPrimaryTranslationTween   = animation.NewTween(ui.Offset{Dx: 0, Dy: 0.05}, ui.Offset{})
	openUpwardsSecondaryTranslationTween = animation.NewTween(ui.Offset{}, ui.Offset{Dx: 0, Dy: -0.025})
	openUpwardsScrimOpacityTween         = animation.NewTween(0.0, 0.25)
	openUpwardsTransitionCurve           = animation.NewCubic(0.20, 0.00, 0.00, 1.00)
	openUpwardsScrimColor                = ui.NewColor(0xFF000000)
)

// openUpwardsPageTransition opens the page from the bottom edge of the screen,
// like the Android Pie one.
type openUpwardsPageTransition struct {
	StatefulWidget
	animation          animation.Animation[float64]
	secondaryAnimation animation.Animation[float64]
	child              Widget
}

func (w *openUpwardsPageTransition) CreateState() State { return &openUpwardsPageTransitionState{} }

type openUpwardsPageTransitionState struct {
	StateOf[*openUpwardsPageTransition]
	primaryAnimation                   *animation.CurvedAnimation
	secondaryTranslationCurvedAnimator *animation.CurvedAnimation
}

func (s *openUpwardsPageTransitionState) InitState() { s.setAnimations() }

func (s *openUpwardsPageTransitionState) DidUpdateWidget(old *openUpwardsPageTransition) {
	w := s.Widget()
	if old.animation != w.animation || old.secondaryAnimation != w.secondaryAnimation {
		s.disposeAnimations()
		s.setAnimations()
	}
}

func (s *openUpwardsPageTransitionState) setAnimations() {
	w := s.Widget()
	s.primaryAnimation = animation.NewCurvedAnimation(w.animation, openUpwardsTransitionCurve, openUpwardsTransitionCurve.Flipped())
	s.secondaryTranslationCurvedAnimator = animation.NewCurvedAnimation(w.secondaryAnimation, openUpwardsTransitionCurve, openUpwardsTransitionCurve.Flipped())
}

func (s *openUpwardsPageTransitionState) disposeAnimations() {
	s.primaryAnimation.Dispose()
	s.secondaryTranslationCurvedAnimator.Dispose()
}

func (s *openUpwardsPageTransitionState) Dispose() { s.disposeAnimations() }

func (s *openUpwardsPageTransitionState) Build(BuildContext) Widget {
	w := s.Widget()
	return NewLayoutBuilder(func(_ BuildContext, constraints rendering.BoxConstraints) Widget {
		size := constraints.Biggest()
		clipAnimation := animation.NewTween(0.0, size.Height).Animate(s.primaryAnimation)
		opacityAnimation := openUpwardsScrimOpacityTween.Animate(s.primaryAnimation)
		primaryTranslationAnimation := openUpwardsPrimaryTranslationTween.Animate(s.primaryAnimation)
		secondaryTranslationAnimation := openUpwardsSecondaryTranslationTween.Animate(s.secondaryTranslationCurvedAnimator)
		return NewAnimatedBuilder(foundation.MergeListenables(w.animation, w.secondaryAnimation), func(BuildContext, Widget) Widget {
			return NewColoredBox(openUpwardsScrimColor.WithOpacity(opacityAnimation.Value()),
				NewAlign(
					NewClipRect(
						NewSizedBox(
							NewOverflowBox(
								NewFractionalTranslation(secondaryTranslationAnimation.Value(),
									NewFractionalTranslation(primaryTranslationAnimation.Value(), w.child)),
								OverflowBoxOpts{Alignment: painting.AlignmentBottomLeft, MaxHeight: foundation.Some(size.Height)}),
							SizedBoxOpts{Height: foundation.Some(clipAnimation.Value())})),
					AlignOpts{Alignment: painting.AlignmentBottomLeft}))
		}, nil)
	})
}

// OpenUpwardsPageTransitionsBuilder is the transition of a page that opens from
// the bottom edge of the screen, like the Android Pie one.
//
// Flutter: widgets/page_transitions_builder.dart OpenUpwardsPageTransitionsBuilder
type OpenUpwardsPageTransitionsBuilder struct{ PageTransitionsBuilderBase }

// NewOpenUpwardsPageTransitionsBuilder creates the builder.
//
// Flutter: widgets/page_transitions_builder.dart OpenUpwardsPageTransitionsBuilder.new
func NewOpenUpwardsPageTransitionsBuilder() OpenUpwardsPageTransitionsBuilder {
	return OpenUpwardsPageTransitionsBuilder{}
}

// BuildTransitions builds the opening of the child from the bottom edge.
//
// Flutter: widgets/page_transitions_builder.dart OpenUpwardsPageTransitionsBuilder.buildTransitions
func (OpenUpwardsPageTransitionsBuilder) BuildTransitions(_ PageRoute, _ BuildContext, a, secondary animation.Animation[float64], child Widget) Widget {
	return &openUpwardsPageTransition{StatefulWidget: NewStatefulWidgetBase(), animation: a, secondaryAnimation: secondary, child: child}
}
