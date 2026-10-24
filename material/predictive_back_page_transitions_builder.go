package material

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Deviation: the corner radii of the display come from
// MediaQueryData.DisplayCornerRadii, which no view fills in Glutter (the engine
// does not report them), so the transitions always use the fallback radius of 32
// logical pixels unless a MediaQuery above sets the radii with
// MediaQueryData.ApplyDisplayCornerRadii.

// PredictiveBackPageTransitionsBuilderOpts are the optional parameters of
// NewPredictiveBackPageTransitionsBuilder.
//
// Flutter: material/predictive_back_page_transitions_builder.dart PredictiveBackPageTransitionsBuilder.new
type PredictiveBackPageTransitionsBuilderOpts struct {
	// FallbackColor is the color of the scrim (background) when the predictive
	// back transition is not supported; the default is the background color of
	// a default FadeForwardsPageTransitionsBuilder.
	FallbackColor foundation.Opt[ui.Color]
}

// PredictiveBackPageTransitionsBuilder is used by PageTransitionsTheme to define
// a MaterialPageRoute page transition animation that looks like the default page
// transition used on Android U and above when using predictive back.
//
// Predictive back is only supported on Android U and above, and if this builder
// is used by any other platform, it falls back to
// FadeForwardsPageTransitionsBuilder.
//
// When used on Android U and above, it animates along with the back gesture to
// reveal the destination route. The gesture can be canceled by dragging back
// towards the edge of the screen.
//
// Flutter: material/predictive_back_page_transitions_builder.dart PredictiveBackPageTransitionsBuilder
type PredictiveBackPageTransitionsBuilder struct {
	widgets.PageTransitionsBuilderBase
	fallbackColor foundation.Opt[ui.Color]
}

// NewPredictiveBackPageTransitionsBuilder creates an instance of a
// PageTransitionsBuilder that matches Android U's predictive back transition.
//
// Flutter: material/predictive_back_page_transitions_builder.dart PredictiveBackPageTransitionsBuilder.new
func NewPredictiveBackPageTransitionsBuilder(opts ...PredictiveBackPageTransitionsBuilderOpts) PredictiveBackPageTransitionsBuilder {
	return PredictiveBackPageTransitionsBuilder{fallbackColor: firstOpts(opts).FallbackColor}
}

// FallbackColor is the color of the scrim (background) when the predictive back
// transition is not supported.
//
// Flutter: material/predictive_back_page_transitions_builder.dart PredictiveBackPageTransitionsBuilder.fallbackColor
func (b PredictiveBackPageTransitionsBuilder) FallbackColor() foundation.Opt[ui.Color] {
	return b.fallbackColor
}

// TransitionDuration is the duration of FadeForwardsPageTransitionsBuilder, 450
// milliseconds.
//
// Flutter: material/predictive_back_page_transitions_builder.dart PredictiveBackPageTransitionsBuilder.transitionDuration
func (PredictiveBackPageTransitionsBuilder) TransitionDuration() time.Duration {
	return FadeForwardsTransitionMilliseconds * time.Millisecond
}

// ReverseTransitionDuration is the same as TransitionDuration.
//
// Flutter: widgets/page_transitions_builder.dart PageTransitionsBuilder.reverseTransitionDuration
func (PredictiveBackPageTransitionsBuilder) ReverseTransitionDuration() time.Duration {
	return FadeForwardsTransitionMilliseconds * time.Millisecond
}

// BuildTransitions builds the shared element transition while a back gesture
// pops the route, and the transition of FadeForwardsPageTransitionsBuilder
// otherwise.
//
// Flutter: material/predictive_back_page_transitions_builder.dart PredictiveBackPageTransitionsBuilder.buildTransitions
func (b PredictiveBackPageTransitionsBuilder) BuildTransitions(route widgets.PageRoute, _ widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return newPredictiveBackGestureDetector(route,
		func(bc widgets.BuildContext, phase predictiveBackPhase, startBackEvent, currentBackEvent foundation.Opt[services.PredictiveBackEvent]) widgets.Widget {
			// Only do a predictive back transition when the user is performing a
			// pop gesture. Otherwise, for things like button presses or other
			// programmatic navigation, fall back to
			// FadeForwardsPageTransitionsBuilder.
			if route.PopGestureInProgress() {
				return newPredictiveBackSharedElementPageTransition(true, a, secondaryAnimation, phase, startBackEvent, currentBackEvent, child)
			}
			return NewFadeForwardsPageTransitionsBuilder(FadeForwardsPageTransitionsBuilderOpts{BackgroundColor: b.fallbackColor}).
				BuildTransitions(route, bc, a, secondaryAnimation, child)
		})
}

// PredictiveBackFullscreenPageTransitionsBuilderOpts are the optional parameters
// of NewPredictiveBackFullscreenPageTransitionsBuilder.
//
// Flutter: material/predictive_back_page_transitions_builder.dart PredictiveBackFullscreenPageTransitionsBuilder.new
type PredictiveBackFullscreenPageTransitionsBuilderOpts struct {
	// FallbackColor is the color of the scrim (background) when the predictive
	// back transition is not supported; the default is the background color of
	// a default ZoomPageTransitionsBuilder.
	FallbackColor foundation.Opt[ui.Color]
}

// PredictiveBackFullscreenPageTransitionsBuilder is used by PageTransitionsTheme
// to define a MaterialPageRoute page transition animation that looks like
// Android's Full Screen page transition.
//
// Predictive back is only supported on Android U and above, and if this builder
// is used by any other platform, it falls back to ZoomPageTransitionsBuilder.
//
// When used on Android U and above, it animates along with the back gesture to
// reveal the destination route. The gesture can be canceled by dragging back
// towards the edge of the screen.
//
// Flutter: material/predictive_back_page_transitions_builder.dart PredictiveBackFullscreenPageTransitionsBuilder
type PredictiveBackFullscreenPageTransitionsBuilder struct {
	widgets.PageTransitionsBuilderBase
	fallbackColor foundation.Opt[ui.Color]
}

// NewPredictiveBackFullscreenPageTransitionsBuilder creates an instance of a
// PageTransitionsBuilder that matches Android U's full screen predictive back
// transition.
//
// Flutter: material/predictive_back_page_transitions_builder.dart PredictiveBackFullscreenPageTransitionsBuilder.new
func NewPredictiveBackFullscreenPageTransitionsBuilder(opts ...PredictiveBackFullscreenPageTransitionsBuilderOpts) PredictiveBackFullscreenPageTransitionsBuilder {
	return PredictiveBackFullscreenPageTransitionsBuilder{fallbackColor: firstOpts(opts).FallbackColor}
}

// FallbackColor is the color of the scrim (background) when the predictive back
// transition is not supported.
//
// Flutter: material/predictive_back_page_transitions_builder.dart PredictiveBackFullscreenPageTransitionsBuilder.fallbackColor
func (b PredictiveBackFullscreenPageTransitionsBuilder) FallbackColor() foundation.Opt[ui.Color] {
	return b.fallbackColor
}

// BuildTransitions builds the full screen transition while a back gesture pops
// the route, and the transition of ZoomPageTransitionsBuilder otherwise.
//
// Flutter: material/predictive_back_page_transitions_builder.dart PredictiveBackFullscreenPageTransitionsBuilder.buildTransitions
func (b PredictiveBackFullscreenPageTransitionsBuilder) BuildTransitions(route widgets.PageRoute, _ widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return newPredictiveBackGestureDetector(route,
		func(bc widgets.BuildContext, phase predictiveBackPhase, _, _ foundation.Opt[services.PredictiveBackEvent]) widgets.Widget {
			// Only do a predictive back transition when the user is performing a
			// pop gesture. Otherwise, for things like button presses or other
			// programmatic navigation, fall back to ZoomPageTransitionsBuilder.
			if route.PopGestureInProgress() {
				return newPredictiveBackFullscreenPageTransition(a, secondaryAnimation, route.IsCurrent, phase, child)
			}
			return NewZoomPageTransitionsBuilder(ZoomPageTransitionsBuilderOpts{BackgroundColor: b.fallbackColor}).
				BuildTransitions(route, bc, a, secondaryAnimation, child)
		})
}

// predictiveBackGestureDetectorBuilder builds the transition of a phase of the
// gesture.
type predictiveBackGestureDetectorBuilder func(bc widgets.BuildContext, phase predictiveBackPhase, startBackEvent, currentBackEvent foundation.Opt[services.PredictiveBackEvent]) widgets.Widget

// predictiveBackPhase is the phases of a predictive back gesture.
type predictiveBackPhase int

const (
	// predictiveBackPhaseIdle: there is no active predictive back gesture in
	// progress.
	predictiveBackPhaseIdle predictiveBackPhase = iota
	// predictiveBackPhaseStart: the user pointer has contacted the screen.
	predictiveBackPhaseStart
	// predictiveBackPhaseUpdate: the user pointer has moved.
	predictiveBackPhaseUpdate
	// predictiveBackPhaseCommit: the user pointer has released in a position in
	// which Android has determined that the back gesture is successful and the
	// current route should be popped.
	predictiveBackPhaseCommit
	// predictiveBackPhaseCancel: the user pointer has released in a position in
	// which Android has determined that the back gesture should be canceled and
	// the original route should be shown.
	predictiveBackPhaseCancel
)

// predictiveBackGestureDetector listens to the back gesture of the binding and
// gives it to the route.
type predictiveBackGestureDetector struct {
	widgets.StatefulWidget
	route   widgets.PageRoute
	builder predictiveBackGestureDetectorBuilder
}

func newPredictiveBackGestureDetector(route widgets.PageRoute, builder predictiveBackGestureDetectorBuilder) *predictiveBackGestureDetector {
	return &predictiveBackGestureDetector{StatefulWidget: widgets.NewStatefulWidgetBase(), route: route, builder: builder}
}

func (w *predictiveBackGestureDetector) CreateState() widgets.State {
	return &predictiveBackGestureDetectorState{}
}

type predictiveBackGestureDetectorState struct {
	widgets.StateOf[*predictiveBackGestureDetector]
	widgets.WidgetsBindingObserverBase
	phase predictiveBackPhase
	// startBackEvent is the back event when the gesture first started.
	startBackEvent foundation.Opt[services.PredictiveBackEvent]
	// currentBackEvent is the most recent back event during the gesture.
	currentBackEvent foundation.Opt[services.PredictiveBackEvent]
}

// isEnabled is true when the predictive back gesture is enabled.
func (s *predictiveBackGestureDetectorState) isEnabled() bool {
	return s.Widget().route.IsCurrent() && s.Widget().route.PopGestureEnabled()
}

func (s *predictiveBackGestureDetectorState) setPhase(phase predictiveBackPhase) {
	if s.phase != phase && s.Mounted() {
		s.SetState(func() { s.phase = phase })
	}
}

func (s *predictiveBackGestureDetectorState) setStartBackEvent(e foundation.Opt[services.PredictiveBackEvent]) {
	if s.startBackEvent != e && s.Mounted() {
		s.SetState(func() { s.startBackEvent = e })
	}
}

func (s *predictiveBackGestureDetectorState) setCurrentBackEvent(e foundation.Opt[services.PredictiveBackEvent]) {
	if s.currentBackEvent != e && s.Mounted() {
		s.SetState(func() { s.currentBackEvent = e })
	}
}

// HandleStartBackGesture takes the gesture when the route is the current one
// and can be popped, and the event is not a button press.
//
// Flutter: material/predictive_back_page_transitions_builder.dart _PredictiveBackGestureDetectorState.handleStartBackGesture
func (s *predictiveBackGestureDetectorState) HandleStartBackGesture(backEvent services.PredictiveBackEvent) bool {
	s.setPhase(predictiveBackPhaseStart)
	gestureInProgress := !backEvent.IsButtonEvent() && s.isEnabled()
	if !gestureInProgress {
		return false
	}

	s.Widget().route.HandleStartBackGesture(1 - backEvent.Progress())
	s.setStartBackEvent(foundation.Some(backEvent))
	s.setCurrentBackEvent(foundation.Some(backEvent))
	return true
}

// HandleUpdateBackGestureProgress moves the route with the gesture.
//
// Flutter: material/predictive_back_page_transitions_builder.dart _PredictiveBackGestureDetectorState.handleUpdateBackGestureProgress
func (s *predictiveBackGestureDetectorState) HandleUpdateBackGestureProgress(backEvent services.PredictiveBackEvent) {
	s.setPhase(predictiveBackPhaseUpdate)

	s.Widget().route.HandleUpdateBackGestureProgress(1 - backEvent.Progress())
	s.setCurrentBackEvent(foundation.Some(backEvent))
}

// HandleCancelBackGesture brings the route back.
//
// Flutter: material/predictive_back_page_transitions_builder.dart _PredictiveBackGestureDetectorState.handleCancelBackGesture
func (s *predictiveBackGestureDetectorState) HandleCancelBackGesture() {
	s.setPhase(predictiveBackPhaseCancel)

	s.Widget().route.HandleCancelBackGesture()
	s.setStartBackEvent(foundation.None[services.PredictiveBackEvent]())
	s.setCurrentBackEvent(foundation.None[services.PredictiveBackEvent]())
}

// HandleCommitBackGesture pops the route.
//
// Flutter: material/predictive_back_page_transitions_builder.dart _PredictiveBackGestureDetectorState.handleCommitBackGesture
func (s *predictiveBackGestureDetectorState) HandleCommitBackGesture() {
	s.setPhase(predictiveBackPhaseCommit)

	s.Widget().route.HandleCommitBackGesture()
	s.setStartBackEvent(foundation.None[services.PredictiveBackEvent]())
	s.setCurrentBackEvent(foundation.None[services.PredictiveBackEvent]())
}

func (s *predictiveBackGestureDetectorState) InitState() {
	widgets.WidgetsBindingOf(s.Context()).AddObserver(s)
}

func (s *predictiveBackGestureDetectorState) Dispose() {
	if b := widgets.MaybeWidgetsBindingOf(s.Context()); b != nil {
		b.RemoveObserver(s)
	}
}

func (s *predictiveBackGestureDetectorState) Build(bc widgets.BuildContext) widgets.Widget {
	effectivePhase := predictiveBackPhaseIdle
	if s.Widget().route.PopGestureInProgress() {
		effectivePhase = s.phase
	}
	return s.Widget().builder(bc, effectivePhase, s.startBackEvent, s.currentBackEvent)
}

// Constants of the shared element transition as per the motion specs
// https://developer.android.com/design/ui/mobile/guides/patterns/predictive-back#motion-specs
const (
	predictiveBackMinScale        = 0.90
	predictiveBackDivisionFactor  = 20.0
	predictiveBackMargin          = 8.0
	predictiveBackYPositionFactor = 0.1

	// predictiveBackCommitMilliseconds is the duration of the commit transition.
	//
	// This is not the same as PredictiveBackPageTransitionsBuilder's duration,
	// which is the duration of the animation of the route, so an Interval is
	// used.
	//
	// Eyeballed on a Pixel 9 running Android 16.
	predictiveBackCommitMilliseconds = 400

	// predictiveBackDeviceBorderRadius is a fallback corner radius used when the
	// display corner radii are unavailable (e.g., on Android API levels below 31,
	// iOS, and other platforms). This is a best-guess value that looks
	// reasonable on most devices.
	// See https://github.com/flutter/flutter/issues/97349.
	predictiveBackDeviceBorderRadius = 32.0
)

var predictiveBackCurve animation.Curve = animation.Curves.EaseInOutCubicEmphasized

var predictiveBackCommitInterval = animation.NewInterval(0.0,
	float64(predictiveBackCommitMilliseconds)/float64(FadeForwardsTransitionMilliseconds), predictiveBackCurve)

// predictiveBackSharedElementPageTransition is Android's predictive back page
// shared element transition.
//
// See https://developer.android.com/design/ui/mobile/guides/patterns/predictive-back#shared-element-transition,
// which is the Android spec for this transition.
type predictiveBackSharedElementPageTransition struct {
	widgets.StatefulWidget
	isDelegatedTransition bool
	animation             animation.Animation[float64]
	secondaryAnimation    animation.Animation[float64]
	phase                 predictiveBackPhase
	startBackEvent        foundation.Opt[services.PredictiveBackEvent]
	currentBackEvent      foundation.Opt[services.PredictiveBackEvent]
	child                 widgets.Widget
}

func newPredictiveBackSharedElementPageTransition(isDelegatedTransition bool, a, secondaryAnimation animation.Animation[float64], phase predictiveBackPhase,
	startBackEvent, currentBackEvent foundation.Opt[services.PredictiveBackEvent], child widgets.Widget) *predictiveBackSharedElementPageTransition {
	return &predictiveBackSharedElementPageTransition{StatefulWidget: widgets.NewStatefulWidgetBase(), isDelegatedTransition: isDelegatedTransition,
		animation: a, secondaryAnimation: secondaryAnimation, phase: phase, startBackEvent: startBackEvent, currentBackEvent: currentBackEvent, child: child}
}

func (w *predictiveBackSharedElementPageTransition) CreateState() widgets.State {
	return newPredictiveBackSharedElementPageTransitionState()
}

type predictiveBackSharedElementPageTransitionState struct {
	widgets.StateOf[*predictiveBackSharedElementPageTransition]

	// borderRadiusTween provides a smooth transition between the default radius
	// and the predictiveBackDeviceBorderRadius, when the display corner radii are
	// unavailable.
	borderRadiusTween *animation.Tween[float64]

	// opacityTween: the route fades out after commit.
	opacityTween *animation.Tween[float64]

	// scaleTween: the route shrinks during the gesture and animates back to
	// normal after commit.
	scaleTween *animation.Tween[float64]

	// commitAnimation stays constant at zero before the commit, and after the
	// commit goes from zero to one.
	commitAnimation *animation.ProxyAnimation

	// bounceAnimation goes from zero to a maximum of one during a predictive back
	// gesture, and then at commit, it goes from its current value to zero. Used
	// for animations that follow the gesture and then animate back to their
	// original value after commit.
	bounceAnimation          *animation.ProxyAnimation
	lastBounceAnimationValue float64

	// animation proxies to the animation of the route during the gesture and
	// then to commitAnimation after the commit. So, it goes from zero to a
	// maximum of one before commit, and then after commit goes from zero to one
	// again.
	animation *animation.ProxyAnimation

	// curvedAnimation is the animation of the route with a curve applied.
	curvedAnimation *animation.CurvedAnimation

	// curvedAnimationReversed is the reverse of curvedAnimation.
	curvedAnimationReversed *animation.CurvedAnimation

	positionAnimation animation.Animation[ui.Offset]

	lastDrag ui.Offset
}

func newPredictiveBackSharedElementPageTransitionState() *predictiveBackSharedElementPageTransitionState {
	return &predictiveBackSharedElementPageTransitionState{
		borderRadiusTween: animation.NewTween(0.0, predictiveBackDeviceBorderRadius),
		opacityTween:      animation.NewTween(1.0, 0.0),
		scaleTween:        animation.NewTween(1.0, predictiveBackMinScale),
		commitAnimation:   animation.NewProxyAnimation(nil),
		bounceAnimation:   animation.NewProxyAnimation(nil),
		animation:         animation.NewProxyAnimation(nil),
	}
}

// yShiftPosition isn't done as an animation because it's based on the vertical
// drag amount, not the progression of the back gesture like the animation of
// the route is.
func (s *predictiveBackSharedElementPageTransitionState) yShiftPosition(screenHeight float64) float64 {
	w := s.Widget()
	startTouchY, currentTouchY := 0.0, 0.0
	if e, ok := w.startBackEvent.Get(); ok {
		if o, ok := e.TouchOffset().Get(); ok {
			startTouchY = o.Dy
		}
	}
	if e, ok := w.currentBackEvent.Get(); ok {
		if o, ok := e.TouchOffset().Get(); ok {
			currentTouchY = o.Dy
		}
	}

	yShiftMax := (screenHeight / predictiveBackDivisionFactor) - predictiveBackMargin

	rawYShift := currentTouchY - startTouchY
	// This curve was eyeballed on a Pixel 9 running Android 16.
	easedYShift := animation.Curves.EaseOut.Transform(ui.ClampDouble(math.Abs(rawYShift)/screenHeight, 0.0, 1.0)) *
		sign(rawYShift) * yShiftMax

	return ui.ClampDouble(easedYShift, -yShiftMax, yShiftMax)
}

func (s *predictiveBackSharedElementPageTransitionState) updateAnimations(screenSize ui.Size) {
	w := s.Widget()
	commit := w.phase == predictiveBackPhaseCommit

	if commit {
		s.animation.SetParent(s.curvedAnimationReversed)
	} else {
		s.animation.SetParent(w.animation)
	}

	if commit {
		s.bounceAnimation.SetParent(animation.NewTween(0.0, s.lastBounceAnimationValue).Animate(s.curvedAnimation))
	} else {
		s.bounceAnimation.SetParent(animation.NewReverseAnimation(w.animation))
	}

	if commit {
		s.commitAnimation.SetParent(s.animation)
	} else {
		s.commitAnimation.SetParent(animation.AlwaysDismissed)
	}

	xShift := (screenSize.Width / predictiveBackDivisionFactor) - predictiveBackMargin
	if commit {
		s.positionAnimation = animation.NewTween(s.lastDrag, ui.Offset{Dx: screenSize.Height * predictiveBackYPositionFactor}).Animate(s.animation)
		return
	}
	// The y position before commit is given by the vertical drag, not by an
	// animation.
	begin := ui.Offset{Dx: xShift, Dy: s.yShiftPosition(screenSize.Height)}
	if e, ok := w.currentBackEvent.Get(); ok && e.SwipeEdge() == services.SwipeEdgeRight {
		begin = ui.Offset{Dx: -xShift, Dy: s.yShiftPosition(screenSize.Height)}
	}
	s.positionAnimation = animation.NewTween(begin, ui.Offset{}).Animate(s.animation)
}

func (s *predictiveBackSharedElementPageTransitionState) updateCurvedAnimations() {
	if s.curvedAnimation != nil {
		s.curvedAnimation.Dispose()
	}
	if s.curvedAnimationReversed != nil {
		s.curvedAnimationReversed.Dispose()
	}
	a := s.Widget().animation
	s.curvedAnimation = animation.NewCurvedAnimation(a, predictiveBackCommitInterval)
	s.curvedAnimationReversed = animation.NewCurvedAnimation(animation.NewReverseAnimation(a), predictiveBackCommitInterval)
}

// TODO(justinmc): Should have a delegatedTransition to animate the incoming
// route regardless of its page transition.
// https://github.com/flutter/flutter/issues/153577

func (s *predictiveBackSharedElementPageTransitionState) DidUpdateWidget(old *predictiveBackSharedElementPageTransition) {
	w := s.Widget()
	if w.animation != old.animation {
		s.updateCurvedAnimations()
	}
	if w.phase != old.phase && w.phase == predictiveBackPhaseCommit {
		s.updateAnimations(widgets.MediaQuerySizeOf(s.Context()))
	}
}

func (s *predictiveBackSharedElementPageTransitionState) DidChangeDependencies() {
	s.updateCurvedAnimations()
	s.updateAnimations(widgets.MediaQuerySizeOf(s.Context()))
}

func (s *predictiveBackSharedElementPageTransitionState) Dispose() {
	s.curvedAnimation.Dispose()
	s.curvedAnimationReversed.Dispose()
}

func (s *predictiveBackSharedElementPageTransitionState) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewAnimatedBuilder(s.Widget().animation, func(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
		s.lastBounceAnimationValue = s.bounceAnimation.Value()
		var offset ui.Offset
		if s.Widget().phase == predictiveBackPhaseCommit {
			offset = s.positionAnimation.Value()
		} else {
			s.lastDrag = ui.Offset{Dx: s.positionAnimation.Value().Dx, Dy: s.yShiftPosition(widgets.MediaQueryHeightOf(bc))}
			offset = s.lastDrag
		}
		borderRadius := painting.NewBorderRadiusCircular(s.borderRadiusTween.Evaluate(s.bounceAnimation))
		if r, ok := widgets.MediaQueryDisplayCornerRadiiOf(bc).Get(); ok {
			borderRadius = r
		}
		return widgets.NewTransformScale(
			widgets.NewTransformTranslate(offset,
				widgets.NewOpacity(s.opacityTween.Evaluate(s.commitAnimation),
					widgets.NewClipRRect(child, widgets.ClipRRectOpts{BorderRadius: borderRadius}))),
			widgets.TransformScaleOpts{Scale: foundation.Some(s.scaleTween.Evaluate(s.bounceAnimation))})
	}, s.Widget().child)
}

// predictiveBackFullscreenPageTransition is Android's predictive back page
// transition for full screen surfaces.
//
// See https://developer.android.com/design/ui/mobile/guides/patterns/predictive-back#full-screen-surfaces,
// which is the Android spec for this transition.
type predictiveBackFullscreenPageTransition struct {
	widgets.StatefulWidget
	animation          animation.Animation[float64]
	secondaryAnimation animation.Animation[float64]
	phase              predictiveBackPhase
	getIsCurrent       func() bool
	child              widgets.Widget
}

func newPredictiveBackFullscreenPageTransition(a, secondaryAnimation animation.Animation[float64], getIsCurrent func() bool, phase predictiveBackPhase, child widgets.Widget) *predictiveBackFullscreenPageTransition {
	return &predictiveBackFullscreenPageTransition{StatefulWidget: widgets.NewStatefulWidgetBase(), animation: a,
		secondaryAnimation: secondaryAnimation, phase: phase, getIsCurrent: getIsCurrent, child: child}
}

func (w *predictiveBackFullscreenPageTransition) CreateState() widgets.State {
	return &predictiveBackFullscreenPageTransitionState{}
}

// These values were eyeballed to match the Android spec for the Full Screen page
// transition:
// https://developer.android.com/design/ui/mobile/guides/patterns/predictive-back#full-screen-surfaces
const (
	predictiveBackScaleStart             = 1.0
	predictiveBackScaleCommit            = 0.95
	predictiveBackOpacityFullyOpened     = 1.0
	predictiveBackOpacityStartTransition = 0.95
	// predictiveBackCommitAt is the point at which the drag would cause a commit
	// instead of a cancel if it were released.
	predictiveBackCommitAt            = 0.65
	predictiveBackWeightPreCommit     = predictiveBackCommitAt
	predictiveBackWeightPostCommit    = 1 - predictiveBackWeightPreCommit
	predictiveBackScreenWidthDivision = 20.0
	predictiveBackXShiftAdjustment    = 8.0
	predictiveBackCommitDuration      = 100 * time.Millisecond
)

type predictiveBackFullscreenPageTransitionState struct {
	widgets.StateOf[*predictiveBackFullscreenPageTransition]
	primaryPositionTween          animation.Animatable[ui.Offset]
	secondaryPositionTween        animation.Animatable[ui.Offset]
	secondaryCurrentPositionTween animation.Animatable[ui.Offset]
}

var (
	predictiveBackPrimaryOpacityTween = animation.NewTween(predictiveBackOpacityStartTransition, predictiveBackOpacityFullyOpened)

	predictiveBackPrimaryScaleTween = animation.NewTweenSequence([]animation.TweenSequenceItem[float64]{
		{Tween: animation.NewTween(predictiveBackScaleStart, predictiveBackScaleStart), Weight: predictiveBackWeightPreCommit},
		{Tween: animation.NewTween(predictiveBackScaleCommit, predictiveBackScaleStart), Weight: predictiveBackWeightPostCommit},
	})

	predictiveBackSecondaryScaleTweenCurrent = animation.NewConstantTween(predictiveBackScaleStart)
	predictiveBackSecondaryTweenScale        = animation.NewTweenSequence([]animation.TweenSequenceItem[float64]{
		{Tween: animation.NewTween(predictiveBackScaleCommit, predictiveBackScaleStart), Weight: predictiveBackWeightPreCommit},
		{Tween: animation.NewTween(predictiveBackScaleStart, predictiveBackScaleStart), Weight: predictiveBackWeightPostCommit},
	})

	predictiveBackSecondaryOpacityTweenCurrent = animation.NewConstantTween(predictiveBackOpacityFullyOpened)
	predictiveBackSecondaryOpacityTween        = animation.NewTweenSequence([]animation.TweenSequenceItem[float64]{
		{Tween: animation.NewTween(predictiveBackOpacityFullyOpened, predictiveBackOpacityStartTransition), Weight: predictiveBackWeightPreCommit},
		{Tween: animation.NewTween(predictiveBackOpacityFullyOpened, predictiveBackOpacityFullyOpened), Weight: predictiveBackWeightPostCommit},
	})
)

func (s *predictiveBackFullscreenPageTransitionState) DidChangeDependencies() {
	screenWidth := widgets.MediaQueryWidthOf(s.Context())
	xShift := (screenWidth / predictiveBackScreenWidthDivision) - predictiveBackXShiftAdjustment
	s.primaryPositionTween = animation.NewTweenSequence([]animation.TweenSequenceItem[ui.Offset]{
		{Tween: animation.NewTween(ui.Offset{}, ui.Offset{}), Weight: predictiveBackWeightPreCommit},
		{Tween: animation.NewTween(ui.Offset{Dx: xShift}, ui.Offset{}), Weight: predictiveBackWeightPostCommit},
	})

	s.secondaryCurrentPositionTween = animation.NewConstantTween(ui.Offset{})
	s.secondaryPositionTween = animation.NewTween(ui.Offset{Dx: xShift}, ui.Offset{})
}

func (s *predictiveBackFullscreenPageTransitionState) secondaryAnimatedBuilder(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	w := s.Widget()
	isCurrent := w.getIsCurrent()

	offset := s.secondaryPositionTween.Evaluate(w.secondaryAnimation)
	scale := predictiveBackSecondaryTweenScale.Evaluate(w.secondaryAnimation)
	opacity := predictiveBackSecondaryOpacityTween.Evaluate(w.secondaryAnimation)
	if isCurrent {
		offset = s.secondaryCurrentPositionTween.Evaluate(w.secondaryAnimation)
		scale = predictiveBackSecondaryScaleTweenCurrent.Evaluate(w.secondaryAnimation)
		opacity = predictiveBackSecondaryOpacityTweenCurrent.Evaluate(w.secondaryAnimation)
	}
	return widgets.NewTransformTranslate(offset,
		widgets.NewTransformScale(widgets.NewOpacity(opacity, child), widgets.TransformScaleOpts{Scale: foundation.Some(scale)}))
}

func (s *predictiveBackFullscreenPageTransitionState) primaryAnimatedBuilder(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	w := s.Widget()
	fadeOpacity := 1.0
	switch {
	case w.phase == predictiveBackPhaseCommit:
		fadeOpacity = 0.0
	case w.animation.Value() < predictiveBackCommitAt:
		fadeOpacity = 0.0
	}
	return widgets.NewTransformTranslate(s.primaryPositionTween.Evaluate(w.animation),
		widgets.NewTransformScale(
			// A slight change in opacity before reaching the commit point.
			widgets.NewOpacity(predictiveBackPrimaryOpacityTween.Evaluate(w.animation),
				// A sudden fadeout at the commit point, driven by time and not
				// the gesture.
				widgets.NewAnimatedOpacity(fadeOpacity, predictiveBackCommitDuration, child)),
			widgets.TransformScaleOpts{Scale: foundation.Some(predictiveBackPrimaryScaleTween.Evaluate(w.animation))}))
}

func (s *predictiveBackFullscreenPageTransitionState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var borderRadius painting.BorderRadiusGeometry = painting.NewBorderRadiusAll(ui.NewRadiusCircular(predictiveBackDeviceBorderRadius))
	if r, ok := widgets.MediaQueryDisplayCornerRadiiOf(bc).Get(); ok {
		borderRadius = r
	}
	return widgets.NewAnimatedBuilder(w.secondaryAnimation, s.secondaryAnimatedBuilder,
		widgets.NewAnimatedBuilder(w.animation, s.primaryAnimatedBuilder,
			widgets.NewClipRRect(w.child, widgets.ClipRRectOpts{BorderRadius: borderRadius})))
}
