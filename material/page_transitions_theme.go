package material

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Deviation: Flutter draws the zoom transitions of a page into a snapshot of the
// page (SnapshotWidget) when the page allows it, as an optimization. Glutter has
// no SnapshotWidget, so every transition is painted with the animated widgets of
// the paths that do not use a snapshot (a FadeTransition, a ScaleTransition and
// the scrim), which look the same. ZoomPageTransitionsBuilder has no
// allowSnapshotting and allowEnterRouteSnapshotting.

var (
	zoomFastOutExtraSlowInSequence = animation.NewTweenSequence([]animation.TweenSequenceItem[float64]{
		{Tween: animation.NewTween(0.0, 0.4).Chain(animation.NewCurveTween(animation.NewCubic(0.05, 0.0, 0.133333, 0.06))), Weight: 0.166666},
		{Tween: animation.NewTween(0.4, 1.0).Chain(animation.NewCurveTween(animation.NewCubic(0.208333, 0.82, 0.25, 1.0))), Weight: 1.0 - 0.166666},
	})

	zoomEnterFadeInTransition = animation.NewTween(0.0, 1.0).Chain(animation.NewCurveTween(animation.NewInterval(0.125, 0.250)))
	zoomEnterScaleDown        = animation.NewTween(1.10, 1.00).Chain(zoomFastOutExtraSlowInSequence)
	zoomEnterScaleUp          = animation.NewTween(0.85, 1.00).Chain(zoomFastOutExtraSlowInSequence)
	zoomEnterScrimOpacity     = animation.NewTween(0.0, 0.60).Chain(animation.NewCurveTween(animation.NewInterval(0.2075, 0.4175)))

	zoomExitFadeOut   = animation.NewTween(1.0, 0.0).Chain(animation.NewCurveTween(animation.NewInterval(0.0825, 0.2075)))
	zoomExitScaleUp   = animation.NewTween(1.00, 1.05).Chain(zoomFastOutExtraSlowInSequence)
	zoomExitScaleDown = animation.NewTween(1.00, 0.90).Chain(zoomFastOutExtraSlowInSequence)
)

// zoomPageTransition is the zoom transition of a page: the page zooms in and
// fades in when it is pushed, and zooms out and fades out when it is popped; the
// page below it is the delegated transition.
type zoomPageTransition struct {
	widgets.StatelessWidget
	animation          animation.Animation[float64]
	secondaryAnimation animation.Animation[float64]
	backgroundColor    foundation.Opt[ui.Color]
	child              widgets.Widget
}

func (w *zoomPageTransition) Build(bc widgets.BuildContext) widgets.Widget {
	enterBackground := w.backgroundColor.OrElse(ColorSchemeOf(bc).Surface())
	return widgets.NewDualTransitionBuilder(w.animation,
		func(_ widgets.BuildContext, a animation.Animation[float64], child widgets.Widget) widgets.Widget {
			return newZoomEnterTransition(a, false, enterBackground, child)
		},
		func(_ widgets.BuildContext, a animation.Animation[float64], child widgets.Widget) widgets.Widget {
			return newZoomExitTransition(a, true, child)
		},
		zoomDelegatedTransition(w.animation, w.secondaryAnimation, w.child, enterBackground))
}

// zoomDelegatedTransition is the transition that the page below takes while
// another page is pushed on it.
func zoomDelegatedTransition(_, secondaryAnimation animation.Animation[float64], child widgets.Widget, backgroundColor ui.Color) widgets.Widget {
	return widgets.NewDualTransitionBuilder(animation.NewReverseAnimation(secondaryAnimation),
		func(_ widgets.BuildContext, a animation.Animation[float64], child widgets.Widget) widgets.Widget {
			return newZoomEnterTransition(a, true, backgroundColor, child)
		},
		func(_ widgets.BuildContext, a animation.Animation[float64], child widgets.Widget) widgets.Widget {
			return newZoomExitTransition(a, false, child)
		},
		child)
}

// zoomEnterTransition zooms in and fades in the child, with a scrim behind it
// that fades in, unless it is the reverse of the delegated transition.
type zoomEnterTransition struct {
	widgets.StatelessWidget
	animation       animation.Animation[float64]
	reverse         bool
	backgroundColor ui.Color
	child           widgets.Widget
}

func newZoomEnterTransition(a animation.Animation[float64], reverse bool, backgroundColor ui.Color, child widgets.Widget) *zoomEnterTransition {
	return &zoomEnterTransition{StatelessWidget: widgets.NewStatelessWidgetBase(), animation: a, reverse: reverse, backgroundColor: backgroundColor, child: child}
}

func (w *zoomEnterTransition) Build(widgets.BuildContext) widgets.Widget {
	fade := animation.AlwaysComplete
	if !w.reverse {
		fade = zoomEnterFadeInTransition.Animate(w.animation)
	}
	scaleTween := zoomEnterScaleUp
	if w.reverse {
		scaleTween = zoomEnterScaleDown
	}
	scale := scaleTween.Animate(w.animation)
	content := widgets.NewFadeTransition(fade, widgets.NewScaleTransition(scale, w.child,
		widgets.ScaleTransitionOpts{FilterQuality: foundation.Some(ui.FilterQualityMedium)}))
	// While the animation does not run the child is painted as it is.
	return widgets.NewAnimatedBuilder(w.animation, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
		if !animation.IsAnimating(w.animation) {
			return w.child
		}
		scrim := 0.0
		if !w.reverse && !animation.IsCompleted(w.animation) {
			scrim = zoomEnterScrimOpacity.Transform(w.animation.Value())
		}
		return widgets.NewColoredBox(w.backgroundColor.WithOpacity(scrim), child)
	}, content)
}

// zoomExitTransition zooms out and fades out the child when it is the reverse
// of the pop, and zooms it up when another page is pushed over it.
type zoomExitTransition struct {
	widgets.StatelessWidget
	animation animation.Animation[float64]
	reverse   bool
	child     widgets.Widget
}

func newZoomExitTransition(a animation.Animation[float64], reverse bool, child widgets.Widget) *zoomExitTransition {
	return &zoomExitTransition{StatelessWidget: widgets.NewStatelessWidgetBase(), animation: a, reverse: reverse, child: child}
}

func (w *zoomExitTransition) Build(widgets.BuildContext) widgets.Widget {
	fade := animation.AlwaysComplete
	scaleTween := zoomExitScaleUp
	if w.reverse {
		fade = zoomExitFadeOut.Animate(w.animation)
		scaleTween = zoomExitScaleDown
	}
	scale := scaleTween.Animate(w.animation)
	content := widgets.NewFadeTransition(fade, widgets.NewScaleTransition(scale, w.child,
		widgets.ScaleTransitionOpts{FilterQuality: foundation.Some(ui.FilterQualityMedium)}))
	return widgets.NewAnimatedBuilder(w.animation, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
		if !animation.IsAnimating(w.animation) {
			return w.child
		}
		return child
	}, content)
}

// ZoomPageTransitionsBuilderOpts are the optional parameters of
// NewZoomPageTransitionsBuilder.
//
// Flutter: material/page_transitions_theme.dart ZoomPageTransitionsBuilder.new
type ZoomPageTransitionsBuilderOpts struct {
	// BackgroundColor is the color of the scrim that fades in behind the entering
	// page; the default is the surface color of the color scheme.
	BackgroundColor foundation.Opt[ui.Color]
}

// ZoomPageTransitionsBuilder is the transition of the Android 10 and later and of
// the desktop: the new page zooms in and fades in over the old page, which
// zooms out a little. It is the default on Linux and Windows.
//
// Flutter: material/page_transitions_theme.dart ZoomPageTransitionsBuilder
type ZoomPageTransitionsBuilder struct {
	widgets.PageTransitionsBuilderBase
	backgroundColor foundation.Opt[ui.Color]
}

// NewZoomPageTransitionsBuilder creates the builder.
//
// Flutter: material/page_transitions_theme.dart ZoomPageTransitionsBuilder.new
func NewZoomPageTransitionsBuilder(opts ...ZoomPageTransitionsBuilderOpts) ZoomPageTransitionsBuilder {
	return ZoomPageTransitionsBuilder{backgroundColor: firstOpts(opts).BackgroundColor}
}

// BackgroundColor is the color of the scrim behind the entering page.
//
// Flutter: material/page_transitions_theme.dart ZoomPageTransitionsBuilder.backgroundColor
func (b ZoomPageTransitionsBuilder) BackgroundColor() foundation.Opt[ui.Color] {
	return b.backgroundColor
}

// DelegatedTransition is the transition that the page below takes when a page
// with this builder is pushed on it.
//
// Flutter: material/page_transitions_theme.dart ZoomPageTransitionsBuilder.delegatedTransition
func (b ZoomPageTransitionsBuilder) DelegatedTransition() widgets.DelegatedTransitionBuilder {
	background := b.backgroundColor
	return func(bc widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], _ bool, child widgets.Widget) widgets.Widget {
		return zoomDelegatedTransition(a, secondaryAnimation, child, background.OrElse(ColorSchemeOf(bc).Surface()))
	}
}

// BuildTransitions builds the zoom of the page.
//
// Flutter: material/page_transitions_theme.dart ZoomPageTransitionsBuilder.buildTransitions
func (b ZoomPageTransitionsBuilder) BuildTransitions(_ widgets.PageRoute, _ widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return &zoomPageTransition{StatelessWidget: widgets.NewStatelessWidgetBase(), animation: a, secondaryAnimation: secondaryAnimation,
		backgroundColor: b.backgroundColor, child: child}
}

var (
	fadeForwardsTransitionCurve     animation.Curve = animation.Curves.EaseInOutCubicEmphasized
	fadeForwardsFadeIn                              = animation.NewTween(0.0, 1.0).Chain(animation.NewCurveTween(animation.NewInterval(0.0, 0.75)))
	fadeForwardsFadeOut                             = animation.NewTween(1.0, 0.0).Chain(animation.NewCurveTween(animation.NewInterval(0.0, 0.25)))
	fadeForwardsForwardTranslation                  = animation.NewTween(ui.Offset{Dx: 0.25}, ui.Offset{}).Chain(animation.NewCurveTween(fadeForwardsTransitionCurve))
	fadeForwardsBackwardTranslation                 = animation.NewTween(ui.Offset{}, ui.Offset{Dx: 0.25}).Chain(animation.NewCurveTween(fadeForwardsTransitionCurve))
	fadeForwardsSecondaryBackward                   = animation.NewTween(ui.Offset{}, ui.Offset{Dx: -0.25}).Chain(animation.NewCurveTween(fadeForwardsTransitionCurve))
	fadeForwardsSecondaryForward                    = animation.NewTween(ui.Offset{Dx: -0.25}, ui.Offset{}).Chain(animation.NewCurveTween(fadeForwardsTransitionCurve))
)

// FadeForwardsTransitionMilliseconds is the duration of the transition of
// FadeForwardsPageTransitionsBuilder, in milliseconds.
//
// Flutter: material/page_transitions_theme.dart FadeForwardsPageTransitionsBuilder.kTransitionMilliseconds
const FadeForwardsTransitionMilliseconds = 450

// fadeForwardsPageTransition fades and slides the page in from the right.
type fadeForwardsPageTransition struct {
	widgets.StatelessWidget
	animation          animation.Animation[float64]
	secondaryAnimation animation.Animation[float64]
	backgroundColor    foundation.Opt[ui.Color]
	child              widgets.Widget
}

func (w *fadeForwardsPageTransition) Build(bc widgets.BuildContext) widgets.Widget {
	return widgets.NewDualTransitionBuilder(w.animation,
		func(_ widgets.BuildContext, a animation.Animation[float64], child widgets.Widget) widgets.Widget {
			return widgets.NewFadeTransition(fadeForwardsFadeIn.Animate(a),
				widgets.NewSlideTransition(fadeForwardsForwardTranslation.Animate(a), child))
		},
		func(_ widgets.BuildContext, a animation.Animation[float64], child widgets.Widget) widgets.Widget {
			return widgets.NewIgnorePointer(widgets.NewFadeTransition(fadeForwardsFadeOut.Animate(a),
				widgets.NewSlideTransition(fadeForwardsBackwardTranslation.Animate(a), child)),
				widgets.IgnorePointerOpts{Ignoring: foundation.Some(a.Status() == animation.AnimationStatusForward)})
		},
		fadeForwardsDelegatedTransition(bc, w.secondaryAnimation, w.backgroundColor, w.child))
}

func fadeForwardsDelegatedTransition(bc widgets.BuildContext, secondaryAnimation animation.Animation[float64], backgroundColor foundation.Opt[ui.Color], child widgets.Widget) widgets.Widget {
	builder := widgets.NewDualTransitionBuilder(animation.NewReverseAnimation(secondaryAnimation),
		func(_ widgets.BuildContext, a animation.Animation[float64], child widgets.Widget) widgets.Widget {
			return widgets.NewFadeTransition(fadeForwardsFadeIn.Animate(a),
				widgets.NewSlideTransition(fadeForwardsSecondaryForward.Animate(a), child))
		},
		func(_ widgets.BuildContext, a animation.Animation[float64], child widgets.Widget) widgets.Widget {
			return widgets.NewFadeTransition(fadeForwardsFadeOut.Animate(a),
				widgets.NewSlideTransition(fadeForwardsSecondaryBackward.Animate(a), child))
		},
		child)
	if !widgets.ModalRouteOpaqueOf(bc).OrElse(true) {
		return builder
	}
	color := Colors.Transparent
	if animation.IsAnimating(secondaryAnimation) {
		color = backgroundColor.OrElse(ColorSchemeOf(bc).Surface())
	}
	return widgets.NewColoredBox(color, builder)
}

// FadeForwardsPageTransitionsBuilderOpts are the optional parameters of
// NewFadeForwardsPageTransitionsBuilder.
//
// Flutter: material/page_transitions_theme.dart FadeForwardsPageTransitionsBuilder.new
type FadeForwardsPageTransitionsBuilderOpts struct {
	// BackgroundColor is the color behind the page below while it moves; the
	// default is the surface color of the color scheme.
	BackgroundColor foundation.Opt[ui.Color]
}

// FadeForwardsPageTransitionsBuilder is a transition where the new page fades
// in while it slides in from the right, and the old page fades out while it
// slides to the left.
//
// Flutter: material/page_transitions_theme.dart FadeForwardsPageTransitionsBuilder
type FadeForwardsPageTransitionsBuilder struct {
	widgets.PageTransitionsBuilderBase
	backgroundColor foundation.Opt[ui.Color]
}

// NewFadeForwardsPageTransitionsBuilder creates the builder.
//
// Flutter: material/page_transitions_theme.dart FadeForwardsPageTransitionsBuilder.new
func NewFadeForwardsPageTransitionsBuilder(opts ...FadeForwardsPageTransitionsBuilderOpts) FadeForwardsPageTransitionsBuilder {
	return FadeForwardsPageTransitionsBuilder{backgroundColor: firstOpts(opts).BackgroundColor}
}

// TransitionDuration is 450 milliseconds.
//
// Flutter: material/page_transitions_theme.dart FadeForwardsPageTransitionsBuilder.transitionDuration
func (FadeForwardsPageTransitionsBuilder) TransitionDuration() time.Duration {
	return FadeForwardsTransitionMilliseconds * time.Millisecond
}

// ReverseTransitionDuration is 450 milliseconds.
//
// Flutter: widgets/page_transitions_builder.dart PageTransitionsBuilder.reverseTransitionDuration
func (FadeForwardsPageTransitionsBuilder) ReverseTransitionDuration() time.Duration {
	return FadeForwardsTransitionMilliseconds * time.Millisecond
}

// DelegatedTransition is the transition that the page below takes.
//
// Flutter: material/page_transitions_theme.dart FadeForwardsPageTransitionsBuilder.delegatedTransition
func (b FadeForwardsPageTransitionsBuilder) DelegatedTransition() widgets.DelegatedTransitionBuilder {
	background := b.backgroundColor
	return func(bc widgets.BuildContext, _, secondaryAnimation animation.Animation[float64], _ bool, child widgets.Widget) widgets.Widget {
		return fadeForwardsDelegatedTransition(bc, secondaryAnimation, background, child)
	}
}

// BuildTransitions builds the fade and slide of the page.
//
// Flutter: material/page_transitions_theme.dart FadeForwardsPageTransitionsBuilder.buildTransitions
func (b FadeForwardsPageTransitionsBuilder) BuildTransitions(_ widgets.PageRoute, _ widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return &fadeForwardsPageTransition{StatelessWidget: widgets.NewStatelessWidgetBase(), animation: a, secondaryAnimation: secondaryAnimation,
		backgroundColor: b.backgroundColor, child: child}
}

// PageTransitionsThemeOpts are the optional parameters of NewPageTransitionsTheme.
//
// Flutter: material/page_transitions_theme.dart PageTransitionsTheme.new
type PageTransitionsThemeOpts struct {
	// Builders are the builders of the transitions of each platform; the default
	// is a PredictiveBackPageTransitionsBuilder for Android and a
	// ZoomPageTransitionsBuilder for Linux and Windows. A platform without a
	// builder uses ZoomPageTransitionsBuilder.
	Builders map[foundation.TargetPlatform]widgets.PageTransitionsBuilder
}

// PageTransitionsTheme defines the page transitions of each platform used by a
// PageRoute that Material makes.
//
// Deviation: Flutter's defaults include CupertinoPageTransitionsBuilder for iOS
// and macOS; Glutter has no Cupertino library, so those two platforms have no
// default builder and use the zoom transition.
//
// Flutter: material/page_transitions_theme.dart PageTransitionsTheme
type PageTransitionsTheme struct {
	builders map[foundation.TargetPlatform]widgets.PageTransitionsBuilder
}

func defaultPageTransitionBuilders() map[foundation.TargetPlatform]widgets.PageTransitionsBuilder {
	return map[foundation.TargetPlatform]widgets.PageTransitionsBuilder{
		foundation.TargetPlatformAndroid: NewPredictiveBackPageTransitionsBuilder(),
		foundation.TargetPlatformWindows: NewZoomPageTransitionsBuilder(),
		foundation.TargetPlatformLinux:   NewZoomPageTransitionsBuilder(),
	}
}

// NewPageTransitionsTheme creates the theme.
//
// Flutter: material/page_transitions_theme.dart PageTransitionsTheme.new
func NewPageTransitionsTheme(opts ...PageTransitionsThemeOpts) *PageTransitionsTheme {
	o := firstOpts(opts)
	if o.Builders == nil {
		o.Builders = defaultPageTransitionBuilders()
	}
	return &PageTransitionsTheme{builders: o.Builders}
}

// Builders are the builders of the transitions, by platform.
//
// Flutter: material/page_transitions_theme.dart PageTransitionsTheme.builders
func (t *PageTransitionsTheme) Builders() map[foundation.TargetPlatform]widgets.PageTransitionsBuilder {
	return t.builders
}

// builderFor is the builder of the platform: the one of the theme, or the zoom
// transition.
func (t *PageTransitionsTheme) builderFor(platform foundation.TargetPlatform) widgets.PageTransitionsBuilder {
	if b, ok := t.builders[platform]; ok && b != nil {
		return b
	}
	return NewZoomPageTransitionsBuilder()
}

// BuildTransitions delegates to the builder of the platform of the theme.
//
// Flutter: material/page_transitions_theme.dart PageTransitionsTheme.buildTransitions
func (t *PageTransitionsTheme) BuildTransitions(route widgets.PageRoute, bc widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return &pageTransitionsThemeTransitions{StatefulWidget: widgets.NewStatefulWidgetBase(), builders: t, route: route,
		animation: a, secondaryAnimation: secondaryAnimation, child: child}
}

// DelegatedTransition is the delegated transition of the builder of the platform.
//
// Flutter: material/page_transitions_theme.dart PageTransitionsTheme.delegatedTransition
func (t *PageTransitionsTheme) DelegatedTransition(platform foundation.TargetPlatform) widgets.DelegatedTransitionBuilder {
	return t.builderFor(platform).DelegatedTransition()
}

// Equal reports whether the two themes have equal builders for every platform.
//
// Flutter: material/page_transitions_theme.dart PageTransitionsTheme.==
func (t *PageTransitionsTheme) Equal(other *PageTransitionsTheme) bool {
	if t == other {
		return true
	}
	if t == nil || other == nil {
		return false
	}
	for p := foundation.TargetPlatformAndroid; p <= foundation.TargetPlatformWindows; p++ {
		if !foundation.ValuesEqual(t.builders[p], other.builders[p]) {
			return false
		}
	}
	return true
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (t *PageTransitionsTheme) ToStringShort() string { return "PageTransitionsTheme" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (t *PageTransitionsTheme) String() string { return foundation.DiagnosticableString(t) }

// DebugFillProperties adds the builders.
//
// Flutter: material/page_transitions_theme.dart PageTransitionsTheme.debugFillProperties
func (t *PageTransitionsTheme) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	properties.Add(foundation.NewDiagnosticsProperty("builders", t.builders, foundation.DiagnosticsPropertyOpts{DefaultValue: defaultPageTransitionBuilders()}))
}

// pageTransitionsThemeTransitions builds the transitions with the builder of the
// platform; while a pop gesture runs it keeps the platform that it started with.
type pageTransitionsThemeTransitions struct {
	widgets.StatefulWidget
	builders           *PageTransitionsTheme
	route              widgets.PageRoute
	animation          animation.Animation[float64]
	secondaryAnimation animation.Animation[float64]
	child              widgets.Widget
}

func (w *pageTransitionsThemeTransitions) CreateState() widgets.State {
	return &pageTransitionsThemeTransitionsState{}
}

type pageTransitionsThemeTransitionsState struct {
	widgets.StateOf[*pageTransitionsThemeTransitions]
	transitionPlatform foundation.Opt[foundation.TargetPlatform]
}

func (s *pageTransitionsThemeTransitionsState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	platform := ThemeOf(bc).Platform
	if w.route.PopGestureInProgress() {
		if s.transitionPlatform.IsNone() {
			s.transitionPlatform = foundation.Some(platform)
		}
		platform = s.transitionPlatform.Value()
	} else {
		s.transitionPlatform = foundation.None[foundation.TargetPlatform]()
	}
	return w.builders.builderFor(platform).BuildTransitions(w.route, bc, w.animation, w.secondaryAnimation, w.child)
}
