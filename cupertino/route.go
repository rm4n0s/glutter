package cupertino

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// backGestureWidth is the width of the area at the start edge in which a drag is a back swipe.
	backGestureWidth = 20.0
	// minFlingVelocity is the velocity of a swipe, in screen widths per second, that pops or keeps the route
	// whatever the distance.
	minFlingVelocity = 1.0
	// droppedSwipePageAnimationDuration is how long the route animates after the finger is lifted.
	droppedSwipePageAnimationDuration = 350 * time.Millisecond
	// modalPopupTransitionDuration is the duration of the transition of a modal popup.
	modalPopupTransitionDuration = 335 * time.Millisecond
)

// pageTransitionBarrierColor is the color of the barrier that a page route has under it.
var pageTransitionBarrierColor = ui.NewColor(0x18000000)

// CupertinoModalBarrierColor is the color of the barrier of a modal popup and of a dialog: a
// translucent black, darker in dark mode.
//
// Flutter: cupertino/route.dart kCupertinoModalBarrierColor
var CupertinoModalBarrierColor = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x33000000), ui.NewColor(0x7A000000))

// The tweens of the transitions, as functions so that the package keeps no state.
func rightMiddleTween() *animation.Tween[ui.Offset] {
	return animation.NewTween(ui.Offset{Dx: 1, Dy: 0}, ui.Offset{})
}

func middleLeftTween() *animation.Tween[ui.Offset] {
	return animation.NewTween(ui.Offset{}, ui.Offset{Dx: -1.0 / 3.0, Dy: 0})
}

func bottomUpTween() *animation.Tween[ui.Offset] {
	return animation.NewTween(ui.Offset{Dx: 0, Dy: 1}, ui.Offset{})
}

// CupertinoRoute is a page route that Cupertino makes: it builds its content with BuildContent and
// transitions with the parallax slide of iOS. It gives the navigation bar the title of the route and
// the title of the one before it.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin
type CupertinoRoute interface {
	widgets.PageRoute
	// BuildContent builds the primary contents of the route.
	//
	// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.buildContent
	BuildContent(bc widgets.BuildContext) widgets.Widget
	// Title is the title of the route, used by a navigation bar as the label of the back button of the route
	// above this one; none for no title.
	//
	// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.title
	Title() foundation.Opt[string]
	// PreviousTitle is the title of the route below this one, when it is a CupertinoRoute with a title.
	// It is the label of the back button of this route's navigation bar.
	//
	// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.previousTitle
	PreviousTitle() foundation.ValueListenable[foundation.Opt[string]]
	cupertinoRouteBase() *CupertinoRouteBase
}

// CupertinoRouteBase is the part of a Cupertino page route that Flutter's CupertinoRouteTransitionMixin
// gives to a PageRoute. Embed it in a page route and call InitCupertinoRouteBase with the outer route.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin
type CupertinoRouteBase struct {
	widgets.PageRouteBase
	cself         CupertinoRoute
	previousTitle *foundation.ValueNotifier[foundation.Opt[string]]
}

// InitCupertinoRouteBase sets the outer value of the route.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin
func InitCupertinoRouteBase(b *CupertinoRouteBase, self CupertinoRoute, opts widgets.PageRouteOpts) {
	b.cself = self
	widgets.InitPageRouteBase(&b.PageRouteBase, self, opts)
}

func (b *CupertinoRouteBase) cupertinoRouteBase() *CupertinoRouteBase { return b }

// PreviousTitle is the title of the route below this one. It panics with the tag
// CupertinoRoutePreviousTitleBeforeInstall when the route has not been installed yet.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.previousTitle
func (b *CupertinoRouteBase) PreviousTitle() foundation.ValueListenable[foundation.Opt[string]] {
	if b.previousTitle == nil {
		panic(errors.New(tagCupertinoRoutePreviousTitleBeforeInstall, "cannot read the previousTitle for a route that has not yet been installed"))
	}
	return b.previousTitle
}

// Dispose discards the notifier of the previous title.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.dispose
func (b *CupertinoRouteBase) Dispose() {
	if b.previousTitle != nil {
		b.previousTitle.Dispose()
	}
	b.PageRouteBase.Dispose()
}

// DidChangePrevious gives the title of the route below to PreviousTitle.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.didChangePrevious
func (b *CupertinoRouteBase) DidChangePrevious(previousRoute widgets.Route) {
	title := foundation.None[string]()
	if prev, ok := previousRoute.(CupertinoRoute); ok {
		title = prev.Title()
	}
	if b.previousTitle == nil {
		b.previousTitle = foundation.NewValueNotifier(title)
	} else {
		b.previousTitle.SetValue(title)
	}
	b.PageRouteBase.DidChangePrevious(previousRoute)
}

// CupertinoRouteTransitionDuration is the duration of the transition of a Cupertino page: 500 ms.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.kTransitionDuration
const CupertinoRouteTransitionDuration = 500 * time.Millisecond

// TransitionDuration is 500 ms.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.transitionDuration
func (b *CupertinoRouteBase) TransitionDuration() time.Duration {
	return CupertinoRouteTransitionDuration
}

// ReverseTransitionDuration is 500 ms.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.reverseTransitionDuration
func (b *CupertinoRouteBase) ReverseTransitionDuration() time.Duration {
	return CupertinoRouteTransitionDuration
}

// BarrierColor is a faint shadow under a page, and none under a fullscreen dialog.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.barrierColor
func (b *CupertinoRouteBase) BarrierColor() foundation.Opt[ui.Color] {
	if b.cself.FullscreenDialog() {
		return foundation.None[ui.Color]()
	}
	return foundation.Some(pageTransitionBarrierColor)
}

// BarrierLabel is empty.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.barrierLabel
func (b *CupertinoRouteBase) BarrierLabel() string { return "" }

// DelegatedTransition is the transition that the route below takes: it slides a third of the screen to the
// start.
//
// Flutter: cupertino/route.dart CupertinoPageRoute.delegatedTransition
func (b *CupertinoRouteBase) DelegatedTransition() widgets.DelegatedTransitionBuilder {
	return CupertinoPageTransitionDelegatedTransition
}

// CanTransitionTo is whether the next route is a Cupertino route, or has a delegated transition, and is not a
// fullscreen dialog.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.canTransitionTo
func (b *CupertinoRouteBase) CanTransitionTo(nextRoute widgets.TransitionRoute) bool {
	next, isPage := nextRoute.(widgets.PageRoute)
	nextIsNotFullscreen := !isPage || !next.FullscreenDialog()
	_, nextIsCupertino := nextRoute.(CupertinoRoute)
	nextHasDelegated := false
	if modal, ok := nextRoute.(widgets.ModalRoute); ok {
		nextHasDelegated = modal.DelegatedTransition() != nil
	}
	return nextIsNotFullscreen && (nextIsCupertino || nextHasDelegated)
}

// CanTransitionFrom is whether the previous route is a page route and this one is not a fullscreen dialog.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.canTransitionFrom
func (b *CupertinoRouteBase) CanTransitionFrom(previousRoute widgets.TransitionRoute) bool {
	_, ok := previousRoute.(widgets.PageRoute)
	return ok && !b.cself.FullscreenDialog()
}

// BuildPage builds the content in a semantics scope of the route.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.buildPage
func (b *CupertinoRouteBase) BuildPage(bc widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
	return widgets.NewSemantics(b.cself.BuildContent(bc), widgets.SemanticsOpts{
		ScopesRoute: foundation.Some(true), ExplicitChildNodes: true,
	})
}

// BuildTransitions builds the page transitions of iOS.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.buildTransitions
func (b *CupertinoRouteBase) BuildTransitions(bc widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return CupertinoRouteBuildPageTransitions(b.cself, bc, a, secondaryAnimation, child)
}

// CupertinoRouteBuildPageTransitions builds the transitions of a page route in the style of iOS: the
// parallax slide with the back swipe for a page, and the slide up for a fullscreen dialog.
//
// Flutter: cupertino/route.dart CupertinoRouteTransitionMixin.buildPageTransitions
func CupertinoRouteBuildPageTransitions(route widgets.PageRoute, bc widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], child widgets.Widget) widgets.Widget {
	linearTransition := route.PopGestureInProgress()
	if route.FullscreenDialog() {
		return NewCupertinoFullscreenDialogTransition(a, secondaryAnimation, linearTransition, child)
	}
	return NewCupertinoPageTransition(a, secondaryAnimation, linearTransition, newBackGestureDetector(
		route.PopGestureEnabled, func() *backGestureController { return startPopGesture(route) }, child))
}

// startPopGesture starts the back gesture of the route.
func startPopGesture(route widgets.PageRoute) *backGestureController {
	if foundation.DebugMode && !route.PopGestureEnabled() {
		panic(errors.New(tagCupertinoPopGestureDisabled, "the back gesture was started on a route that does not allow it", "route", route.DebugLabel()))
	}
	return newBackGestureController(route.Navigator(), route.Controller(), route.IsCurrent, route.IsActive)
}

// CupertinoPageRouteOpts are the optional parameters of NewCupertinoPageRoute.
//
// Flutter: cupertino/route.dart CupertinoPageRoute.new
type CupertinoPageRouteOpts struct {
	widgets.PageRouteOpts
	// Title is the title of the route: the label of the back button of the navigation bar of the route
	// above it. None for no title.
	Title foundation.Opt[string]
	// MaintainState is whether the route keeps its state in memory when it is not the current one; the default
	// is true.
	MaintainState foundation.Opt[bool]
}

// CupertinoPageRoute is a modal route that replaces the entire screen with an iOS transition: the new page
// slides in from the right and the one below it slides a third of the way to the left, and a swipe from the
// edge of the screen pops it.
//
// Flutter: cupertino/route.dart CupertinoPageRoute
type CupertinoPageRoute struct {
	CupertinoRouteBase
	builder       widgets.WidgetBuilder
	title         foundation.Opt[string]
	maintainState bool
}

// NewCupertinoPageRoute creates a route whose content is built by builder.
//
// Flutter: cupertino/route.dart CupertinoPageRoute.new
func NewCupertinoPageRoute(builder widgets.WidgetBuilder, opts ...CupertinoPageRouteOpts) *CupertinoPageRoute {
	o := firstOf(opts)
	r := &CupertinoPageRoute{builder: builder, title: o.Title, maintainState: o.MaintainState.OrElse(true)}
	InitCupertinoRouteBase(&r.CupertinoRouteBase, r, o.PageRouteOpts)
	return r
}

// Builder builds the primary contents of the route.
//
// Flutter: cupertino/route.dart CupertinoPageRoute.builder
func (r *CupertinoPageRoute) Builder() widgets.WidgetBuilder { return r.builder }

// BuildContent builds the primary contents of the route.
//
// Flutter: cupertino/route.dart CupertinoPageRoute.buildContent
func (r *CupertinoPageRoute) BuildContent(bc widgets.BuildContext) widgets.Widget {
	return r.builder(bc)
}

// Title is the title of the route.
//
// Flutter: cupertino/route.dart CupertinoPageRoute.title
func (r *CupertinoPageRoute) Title() foundation.Opt[string] { return r.title }

// MaintainState is whether the route keeps its state when it is not the current one.
//
// Flutter: cupertino/route.dart CupertinoPageRoute.maintainState
func (r *CupertinoPageRoute) MaintainState() bool { return r.maintainState }

// DebugLabel names the route and its settings.
//
// Flutter: cupertino/route.dart CupertinoPageRoute.debugLabel
func (r *CupertinoPageRoute) DebugLabel() string {
	return r.CupertinoRouteBase.DebugLabel() + "(" + r.Settings().Name() + ")"
}

// CupertinoPageOpts are the optional parameters of NewCupertinoPage.
//
// Flutter: cupertino/route.dart CupertinoPage.new
type CupertinoPageOpts struct {
	widgets.PageOpts
	// MaintainState is whether the route keeps its state in memory when it is not the current one; the default
	// is true.
	MaintainState foundation.Opt[bool]
	// Title is the title of the route.
	Title foundation.Opt[string]
	// FullscreenDialog is whether the route is a full-screen dialog.
	FullscreenDialog bool
	// DisallowSnapshotting is true to keep the transition of the route from using a snapshot of it.
	DisallowSnapshotting bool
}

// CupertinoPage is a page that creates a Cupertino style PageRoute.
//
// Flutter: cupertino/route.dart CupertinoPage
type CupertinoPage struct {
	widgets.PageBase
	child                widgets.Widget
	title                foundation.Opt[string]
	maintainState        bool
	fullscreenDialog     bool
	disallowSnapshotting bool
}

// NewCupertinoPage creates a page of the child.
//
// Flutter: cupertino/route.dart CupertinoPage.new
func NewCupertinoPage(child widgets.Widget, opts ...CupertinoPageOpts) *CupertinoPage {
	o := firstOf(opts)
	p := &CupertinoPage{child: child, title: o.Title, maintainState: o.MaintainState.OrElse(true),
		fullscreenDialog: o.FullscreenDialog, disallowSnapshotting: o.DisallowSnapshotting}
	widgets.InitPageBase(&p.PageBase, p, o.PageOpts)
	return p
}

// Child is the content of the route.
//
// Flutter: cupertino/route.dart CupertinoPage.child
func (p *CupertinoPage) Child() widgets.Widget { return p.child }

// Title is the title of the route.
//
// Flutter: cupertino/route.dart CupertinoPage.title
func (p *CupertinoPage) Title() foundation.Opt[string] { return p.title }

// MaintainState is whether the route keeps its state when it is not the current one.
//
// Flutter: cupertino/route.dart CupertinoPage.maintainState
func (p *CupertinoPage) MaintainState() bool { return p.maintainState }

// FullscreenDialog is whether the route is a full-screen dialog.
//
// Flutter: cupertino/route.dart CupertinoPage.fullscreenDialog
func (p *CupertinoPage) FullscreenDialog() bool { return p.fullscreenDialog }

// AllowSnapshotting is whether the transition of the route may use a snapshot.
//
// Flutter: cupertino/route.dart CupertinoPage.allowSnapshotting
func (p *CupertinoPage) AllowSnapshotting() bool { return !p.disallowSnapshotting }

// CreateRoute creates the page based route.
//
// Flutter: cupertino/route.dart CupertinoPage.createRoute
func (p *CupertinoPage) CreateRoute(widgets.BuildContext) widgets.Route {
	r := &pageBasedCupertinoPageRoute{}
	InitCupertinoRouteBase(&r.CupertinoRouteBase, r, widgets.PageRouteOpts{
		RouteOpts:            widgets.RouteOpts{Settings: p},
		FullscreenDialog:     p.fullscreenDialog,
		DisallowSnapshotting: p.disallowSnapshotting,
	})
	return r
}

// pageBasedCupertinoPageRoute is the route of a CupertinoPage: the page, which is the settings of the route,
// gives it its content; the navigator replaces the settings when the page changes.
type pageBasedCupertinoPageRoute struct {
	CupertinoRouteBase
}

func (r *pageBasedCupertinoPageRoute) page() *CupertinoPage { return r.Settings().(*CupertinoPage) }

func (r *pageBasedCupertinoPageRoute) BuildContent(widgets.BuildContext) widgets.Widget {
	return r.page().child
}

func (r *pageBasedCupertinoPageRoute) Title() foundation.Opt[string] { return r.page().title }

func (r *pageBasedCupertinoPageRoute) MaintainState() bool { return r.page().maintainState }

func (r *pageBasedCupertinoPageRoute) FullscreenDialog() bool { return r.page().fullscreenDialog }

func (r *pageBasedCupertinoPageRoute) DelegatedTransition() widgets.DelegatedTransitionBuilder {
	if r.FullscreenDialog() {
		return nil
	}
	return CupertinoPageTransitionDelegatedTransition
}

func (r *pageBasedCupertinoPageRoute) DebugLabel() string {
	return r.CupertinoRouteBase.DebugLabel() + "(" + r.page().Name() + ")"
}

// CupertinoPageTransition provides an iOS style page transition animation: the page slides in from the
// right edge with a shadow along its start edge, and the page below slides a third of the way to the start.
//
// Flutter: cupertino/route.dart CupertinoPageTransition
type CupertinoPageTransition struct {
	widgets.StatefulWidget
	primaryRouteAnimation   animation.Animation[float64]
	secondaryRouteAnimation animation.Animation[float64]
	linearTransition        bool
	child                   widgets.Widget
}

// NewCupertinoPageTransition creates the transition of the child. linearTransition is true while the user
// drags the route with the back swipe: the animation then follows the finger, with no easing.
//
// Flutter: cupertino/route.dart CupertinoPageTransition.new
func NewCupertinoPageTransition(primaryRouteAnimation, secondaryRouteAnimation animation.Animation[float64], linearTransition bool, child widgets.Widget, opts ...widgets.WidgetOpts) *CupertinoPageTransition {
	return &CupertinoPageTransition{widgets.NewStatefulWidgetBase(firstOf(opts)), primaryRouteAnimation, secondaryRouteAnimation, linearTransition, child}
}

// Child is the widget that slides.
//
// Flutter: cupertino/route.dart CupertinoPageTransition.child
func (w *CupertinoPageTransition) Child() widgets.Widget { return w.child }

// PrimaryRouteAnimation is the animation of the route that is coming in or going out.
//
// Flutter: cupertino/route.dart CupertinoPageTransition.primaryRouteAnimation
func (w *CupertinoPageTransition) PrimaryRouteAnimation() animation.Animation[float64] {
	return w.primaryRouteAnimation
}

// SecondaryRouteAnimation is the animation of the route that the page below is giving way to.
//
// Flutter: cupertino/route.dart CupertinoPageTransition.secondaryRouteAnimation
func (w *CupertinoPageTransition) SecondaryRouteAnimation() animation.Animation[float64] {
	return w.secondaryRouteAnimation
}

// LinearTransition is whether the animation is linear: true while the user drags the route.
//
// Flutter: cupertino/route.dart CupertinoPageTransition.linearTransition
func (w *CupertinoPageTransition) LinearTransition() bool { return w.linearTransition }

// CupertinoPageTransitionDelegatedTransition is the transition that a route below a Cupertino page takes: the
// slide a third of the way to the start. A route with a delegated transition lets the one below move with it.
//
// Flutter: cupertino/route.dart CupertinoPageTransition.delegatedTransition
func CupertinoPageTransitionDelegatedTransition(bc widgets.BuildContext, _, secondaryAnimation animation.Animation[float64], _ bool, child widgets.Widget) widgets.Widget {
	curved := animation.NewCurvedAnimation(secondaryAnimation, animation.Curves.LinearToEaseOut, animation.Curves.EaseInToLinear)
	delegatedPositionAnimation := animation.Drive(curved, middleLeftTween())
	curved.Dispose()
	textDirection := widgets.DirectionalityOf(bc)
	return widgets.NewSlideTransition(delegatedPositionAnimation, child, widgets.SlideTransitionOpts{
		TextDirection: textDirection, TransformHitTests: foundation.Some(false)})
}

// CreateState creates the state.
//
// Flutter: cupertino/route.dart CupertinoPageTransition.createState
func (w *CupertinoPageTransition) CreateState() widgets.State { return &cupertinoPageTransitionState{} }

type cupertinoPageTransitionState struct {
	widgets.StateOf[*CupertinoPageTransition]
	primaryPosition                                                  animation.Animation[ui.Offset]
	secondaryPosition                                                animation.Animation[ui.Offset]
	primaryShadow                                                    animation.Animation[painting.Decoration]
	primaryPositionCurve, secondaryPositionCurve, primaryShadowCurve *animation.CurvedAnimation
}

func (s *cupertinoPageTransitionState) InitState() { s.setupAnimation() }

func (s *cupertinoPageTransitionState) DidUpdateWidget(old *CupertinoPageTransition) {
	w := s.Widget()
	if old.primaryRouteAnimation != w.primaryRouteAnimation || old.secondaryRouteAnimation != w.secondaryRouteAnimation ||
		old.linearTransition != w.linearTransition {
		s.disposeCurve()
		s.setupAnimation()
	}
}

func (s *cupertinoPageTransitionState) Dispose() { s.disposeCurve() }

func (s *cupertinoPageTransitionState) disposeCurve() {
	for _, c := range []**animation.CurvedAnimation{&s.primaryPositionCurve, &s.secondaryPositionCurve, &s.primaryShadowCurve} {
		if *c != nil {
			(*c).Dispose()
			*c = nil
		}
	}
}

func (s *cupertinoPageTransitionState) setupAnimation() {
	w := s.Widget()
	primary, secondary, shadow := w.primaryRouteAnimation, w.secondaryRouteAnimation, w.primaryRouteAnimation
	if !w.linearTransition {
		s.primaryPositionCurve = animation.NewCurvedAnimation(w.primaryRouteAnimation, animation.Curves.FastEaseInToSlowEaseOut, animation.Curves.FastEaseInToSlowEaseOut.Flipped())
		s.secondaryPositionCurve = animation.NewCurvedAnimation(w.secondaryRouteAnimation, animation.Curves.LinearToEaseOut, animation.Curves.EaseInToLinear)
		s.primaryShadowCurve = animation.NewCurvedAnimation(w.primaryRouteAnimation, animation.Curves.LinearToEaseOut)
		primary, secondary, shadow = s.primaryPositionCurve, s.secondaryPositionCurve, s.primaryShadowCurve
	}
	s.primaryPosition = animation.Drive(primary, rightMiddleTween())
	s.secondaryPosition = animation.Drive(secondary, middleLeftTween())
	s.primaryShadow = animation.Drive(shadow, edgeShadowDecorationTween())
}

func (s *cupertinoPageTransitionState) Build(bc widgets.BuildContext) widgets.Widget {
	textDirection := widgets.DirectionalityOf(bc)
	return widgets.NewSlideTransition(s.secondaryPosition,
		widgets.NewSlideTransition(s.primaryPosition,
			widgets.NewDecoratedBoxTransition(s.primaryShadow, s.Widget().child),
			widgets.SlideTransitionOpts{TextDirection: textDirection}),
		widgets.SlideTransitionOpts{TextDirection: textDirection, TransformHitTests: foundation.Some(false)})
}

// CupertinoFullscreenDialogTransition provides an iOS style transition for a fullscreen dialog: the page
// slides up from the bottom, and the page below moves a third of the way to the start.
//
// Flutter: cupertino/route.dart CupertinoFullscreenDialogTransition
type CupertinoFullscreenDialogTransition struct {
	widgets.StatefulWidget
	primaryRouteAnimation   animation.Animation[float64]
	secondaryRouteAnimation animation.Animation[float64]
	linearTransition        bool
	child                   widgets.Widget
}

// NewCupertinoFullscreenDialogTransition creates the transition of the child.
//
// Flutter: cupertino/route.dart CupertinoFullscreenDialogTransition.new
func NewCupertinoFullscreenDialogTransition(primaryRouteAnimation, secondaryRouteAnimation animation.Animation[float64], linearTransition bool, child widgets.Widget, opts ...widgets.WidgetOpts) *CupertinoFullscreenDialogTransition {
	return &CupertinoFullscreenDialogTransition{widgets.NewStatefulWidgetBase(firstOf(opts)), primaryRouteAnimation, secondaryRouteAnimation, linearTransition, child}
}

// Child is the widget that slides.
//
// Flutter: cupertino/route.dart CupertinoFullscreenDialogTransition.child
func (w *CupertinoFullscreenDialogTransition) Child() widgets.Widget { return w.child }

// CreateState creates the state.
//
// Flutter: cupertino/route.dart CupertinoFullscreenDialogTransition.createState
func (w *CupertinoFullscreenDialogTransition) CreateState() widgets.State {
	return &cupertinoFullscreenDialogTransitionState{}
}

type cupertinoFullscreenDialogTransitionState struct {
	widgets.StateOf[*CupertinoFullscreenDialogTransition]
	primaryPosition, secondaryPosition           animation.Animation[ui.Offset]
	primaryPositionCurve, secondaryPositionCurve *animation.CurvedAnimation
}

func (s *cupertinoFullscreenDialogTransitionState) InitState() { s.setupAnimation() }

func (s *cupertinoFullscreenDialogTransitionState) DidUpdateWidget(old *CupertinoFullscreenDialogTransition) {
	w := s.Widget()
	if old.primaryRouteAnimation != w.primaryRouteAnimation || old.secondaryRouteAnimation != w.secondaryRouteAnimation ||
		old.linearTransition != w.linearTransition {
		s.disposeCurve()
		s.setupAnimation()
	}
}

func (s *cupertinoFullscreenDialogTransitionState) Dispose() { s.disposeCurve() }

func (s *cupertinoFullscreenDialogTransitionState) disposeCurve() {
	for _, c := range []**animation.CurvedAnimation{&s.primaryPositionCurve, &s.secondaryPositionCurve} {
		if *c != nil {
			(*c).Dispose()
			*c = nil
		}
	}
}

func (s *cupertinoFullscreenDialogTransitionState) setupAnimation() {
	w := s.Widget()
	s.primaryPositionCurve = animation.NewCurvedAnimation(w.primaryRouteAnimation, animation.Curves.LinearToEaseOut, animation.Curves.LinearToEaseOut.Flipped())
	s.primaryPosition = animation.Drive(s.primaryPositionCurve, bottomUpTween())
	secondary := w.secondaryRouteAnimation
	if !w.linearTransition {
		s.secondaryPositionCurve = animation.NewCurvedAnimation(w.secondaryRouteAnimation, animation.Curves.LinearToEaseOut, animation.Curves.EaseInToLinear)
		secondary = s.secondaryPositionCurve
	}
	s.secondaryPosition = animation.Drive(secondary, middleLeftTween())
}

func (s *cupertinoFullscreenDialogTransitionState) Build(bc widgets.BuildContext) widgets.Widget {
	textDirection := widgets.DirectionalityOf(bc)
	return widgets.NewSlideTransition(s.secondaryPosition,
		widgets.NewSlideTransition(s.primaryPosition, s.Widget().child),
		widgets.SlideTransitionOpts{TextDirection: textDirection, TransformHitTests: foundation.Some(false)})
}

// backGestureDetector is the strip at the start edge of a page that starts the back swipe.
//
// Flutter: cupertino/route.dart _CupertinoBackGestureDetector
type backGestureDetector struct {
	widgets.StatefulWidget
	enabledCallback   func() bool
	onStartPopGesture func() *backGestureController
	child             widgets.Widget
}

func newBackGestureDetector(enabled func() bool, onStart func() *backGestureController, child widgets.Widget) *backGestureDetector {
	return &backGestureDetector{widgets.NewStatefulWidgetBase(), enabled, onStart, child}
}

func (w *backGestureDetector) CreateState() widgets.State { return &backGestureDetectorState{} }

type backGestureDetectorState struct {
	widgets.StateOf[*backGestureDetector]
	controller *backGestureController
	recognizer *gestures.HorizontalDragGestureRecognizer
}

func (s *backGestureDetectorState) InitState() {
	s.recognizer = gestures.NewHorizontalDragGestureRecognizer(widgets.WidgetsBindingOf(s.Context()).Gestures(), gestures.DragGestureRecognizerOpts{DebugOwner: s})
	s.recognizer.OnStart = s.handleDragStart
	s.recognizer.OnUpdate = s.handleDragUpdate
	s.recognizer.OnEnd = s.handleDragEnd
	s.recognizer.OnCancel = s.handleDragCancel
}

func (s *backGestureDetectorState) Dispose() {
	s.recognizer.Dispose()
	if c := s.controller; c != nil {
		// The gesture is over when the detector goes away: tell the navigator after the frame, as it may be
		// the one that is being rebuilt now.
		binding := widgets.WidgetsBindingOf(s.Context())
		binding.RunPostFrame(func(time.Duration) {
			if c.navigator.Mounted() {
				c.navigator.DidStopUserGesture()
			}
		})
		s.controller = nil
	}
}

func (s *backGestureDetectorState) handleDragStart(gestures.DragStartDetails) {
	s.controller = s.Widget().onStartPopGesture()
}

func (s *backGestureDetectorState) handleDragUpdate(details gestures.DragUpdateDetails) {
	size, _ := s.Context().Size()
	s.controller.dragUpdate(s.convertToLogical(details.PrimaryDelta.Value() / size.Width))
}

func (s *backGestureDetectorState) handleDragEnd(details gestures.DragEndDetails) {
	size, _ := s.Context().Size()
	s.controller.dragEnd(s.convertToLogical(details.Velocity.PixelsPerSecond.Dx / size.Width))
	s.controller = nil
}

func (s *backGestureDetectorState) handleDragCancel() {
	if s.controller != nil {
		s.controller.dragEnd(0)
	}
	s.controller = nil
}

func (s *backGestureDetectorState) handlePointerDown(event *gestures.PointerDownEvent) {
	if s.Widget().enabledCallback() {
		s.recognizer.AddPointer(event)
	}
}

func (s *backGestureDetectorState) convertToLogical(value float64) float64 {
	if widgets.DirectionalityOf(s.Context()) == ui.TextDirectionRTL {
		return -value
	}
	return value
}

func (s *backGestureDetectorState) Build(bc widgets.BuildContext) widgets.Widget {
	padding := widgets.MediaQueryPaddingOf(bc)
	dragAreaWidth := padding.Left
	if widgets.DirectionalityOf(bc) == ui.TextDirectionRTL {
		dragAreaWidth = padding.Right
	}
	return widgets.NewStack([]widgets.Widget{
		s.Widget().child,
		widgets.NewPositionedDirectional(
			widgets.NewListener(nil, widgets.ListenerOpts{OnPointerDown: s.handlePointerDown, Behavior: rendering.HitTestBehaviorTranslucent}),
			widgets.PositionedDirectionalOpts{Start: foundation.Some(0.0), Width: foundation.Some(max(dragAreaWidth, backGestureWidth)),
				Top: foundation.Some(0.0), Bottom: foundation.Some(0.0)}),
	}, widgets.StackOpts{Fit: rendering.StackFitPassthrough})
}

// backGestureController drives the route with the finger of a back swipe, and lets it go on or come back
// when the finger is lifted.
//
// Flutter: cupertino/route.dart _CupertinoBackGestureController
type backGestureController struct {
	controller *animation.AnimationController
	navigator  *widgets.NavigatorState
	isActive   func() bool
	isCurrent  func() bool
}

func newBackGestureController(navigator *widgets.NavigatorState, controller *animation.AnimationController, isCurrent, isActive func() bool) *backGestureController {
	c := &backGestureController{controller: controller, navigator: navigator, isActive: isActive, isCurrent: isCurrent}
	navigator.DidStartUserGesture()
	return c
}

// dragUpdate moves the route by the fraction of the screen width that the finger moved.
func (c *backGestureController) dragUpdate(delta float64) {
	c.controller.SetValue(c.controller.Value() - delta)
}

// dragEnd lets go of the route: it pops when the finger was lifted past the middle or flinging, and comes
// back otherwise.
func (c *backGestureController) dragEnd(velocity float64) {
	var animationCurve animation.Curve = animation.Curves.FastEaseInToSlowEaseOut
	isCurrent := c.isCurrent()
	var animateForward bool
	switch {
	case !isCurrent:
		// If the route is no longer current, it is going away or has been popped: finish whichever it was.
		animateForward = c.isActive()
	case velocity >= minFlingVelocity || velocity <= -minFlingVelocity:
		// The closer the fling is to 1, the more it matters over the distance.
		animateForward = velocity <= 0
	default:
		animateForward = c.controller.Value() > 0.5
	}
	opts := animation.AnimateOpts{Duration: foundation.Some(droppedSwipePageAnimationDuration), Curve: animationCurve}
	if animateForward {
		// The closer the panel is to dismissing, the shorter the animation is.
		c.controller.AnimateTo(1.0, opts)
	} else {
		if isCurrent {
			// This route is destined to pop at this point: pop it now so that the hero flight is not delayed.
			c.navigator.Pop()
		}
		// The popping may have finished the animation already, so there is nothing more to animate.
		if c.controller.IsAnimating() {
			c.controller.AnimateBack(0.0, opts)
		}
	}
	if c.controller.IsAnimating() {
		// Keep the userGestureInProgress in true state so that the hero animation does not start until the
		// animation is over.
		var handle animation.StatusListenerHandle
		handle = c.controller.AddStatusListener(func(animation.AnimationStatus) {
			c.navigator.DidStopUserGesture()
			c.controller.RemoveStatusListener(handle)
		})
	} else {
		c.navigator.DidStopUserGesture()
	}
}

// edgeShadowDecoration is the shadow along the start edge of a page that slides over another: bands of
// colors from the first to the last, over 5% of the width of the page.
//
// Flutter: cupertino/route.dart _CupertinoEdgeShadowDecoration
type edgeShadowDecoration struct {
	painting.DecorationDefaults
	colors []ui.Color
}

// edgeShadowDecorationTween goes from no shadow to the shadow: black at 1.5% opacity to transparent.
func edgeShadowDecorationTween() *animation.Tween[painting.Decoration] {
	return widgets.NewDecorationTween(&edgeShadowDecoration{},
		&edgeShadowDecoration{colors: []ui.Color{ui.NewColor(0x04000000), CupertinoColors.Transparent}})
}

func lerpNullableColor(a, b *ui.Color, t float64) ui.Color {
	if c := ui.ColorLerp(a, b, t); c != nil {
		return *c
	}
	return ui.Color{}
}

// lerpEdgeShadow interpolates between two shadows; either may be nil.
func lerpEdgeShadow(a, b *edgeShadowDecoration, t float64) *edgeShadowDecoration {
	if a == b {
		return a
	}
	if a == nil {
		if b.colors == nil {
			return b
		}
		out := make([]ui.Color, len(b.colors))
		for i, c := range b.colors {
			c := c
			out[i] = lerpNullableColor(nil, &c, t)
		}
		return &edgeShadowDecoration{colors: out}
	}
	if b == nil {
		if a.colors == nil {
			return a
		}
		out := make([]ui.Color, len(a.colors))
		for i, c := range a.colors {
			c := c
			out[i] = lerpNullableColor(nil, &c, 1-t)
		}
		return &edgeShadowDecoration{colors: out}
	}
	n := max(len(a.colors), len(b.colors))
	out := make([]ui.Color, n)
	for i := range out {
		var ca, cb *ui.Color
		if i < len(a.colors) {
			ca = &a.colors[i]
		}
		if i < len(b.colors) {
			cb = &b.colors[i]
		}
		out[i] = lerpNullableColor(ca, cb, t)
	}
	return &edgeShadowDecoration{colors: out}
}

// LerpFrom interpolates from another shadow, or from none.
func (d *edgeShadowDecoration) LerpFrom(a painting.Decoration, t float64) painting.Decoration {
	if prev, ok := a.(*edgeShadowDecoration); ok {
		return lerpEdgeShadow(prev, d, t)
	}
	return lerpEdgeShadow(nil, d, t)
}

// LerpTo interpolates to another shadow, or to none.
func (d *edgeShadowDecoration) LerpTo(b painting.Decoration, t float64) painting.Decoration {
	if next, ok := b.(*edgeShadowDecoration); ok {
		return lerpEdgeShadow(d, next, t)
	}
	return lerpEdgeShadow(d, nil, t)
}

// CreateBoxPainter makes the painter of the shadow.
func (d *edgeShadowDecoration) CreateBoxPainter(func()) painting.BoxPainter {
	return &edgeShadowPainter{d}
}

// Equal is whether the other decoration has the same colors.
func (d *edgeShadowDecoration) Equal(other painting.Decoration) bool {
	o, ok := other.(*edgeShadowDecoration)
	if !ok || o == nil {
		return false
	}
	if len(d.colors) != len(o.colors) || (d.colors == nil) != (o.colors == nil) {
		return false
	}
	for i := range d.colors {
		if d.colors[i] != o.colors[i] {
			return false
		}
	}
	return true
}

// DebugFillProperties adds the colors.
func (d *edgeShadowDecoration) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewIterableProperty("colors", d.colors))
}

// ToStringShort names the type.
func (d *edgeShadowDecoration) ToStringShort() string { return "_CupertinoEdgeShadowDecoration" }

// edgeShadowPainter paints the bands of the shadow, one pixel column at a time.
//
// Flutter: cupertino/route.dart _CupertinoEdgeShadowPainter
type edgeShadowPainter struct{ decoration *edgeShadowDecoration }

// Paint paints the shadow to the start side of the page, outside it.
func (p *edgeShadowPainter) Paint(canvas *ui.Canvas, offset ui.Offset, configuration painting.ImageConfiguration) {
	colors := p.decoration.colors
	if colors == nil {
		return
	}
	size := configuration.Size.Value()
	// The width of the shadow is 5% of the width of the page.
	shadowWidth := 0.05 * size.Width
	shadowHeight := size.Height
	bandWidth := shadowWidth / float64(len(colors)-1)
	shadowDirection, start := -1.0, offset.Dx
	if configuration.TextDirection == ui.TextDirectionRTL {
		shadowDirection, start = 1, offset.Dx+size.Width
	}
	bandColorIndex := 0
	for dx := 0; float64(dx) < shadowWidth; dx++ {
		if int(float64(dx)/bandWidth) != bandColorIndex {
			bandColorIndex++
		}
		paint := ui.NewPaint()
		a, b := colors[bandColorIndex], colors[bandColorIndex+1]
		frac := (float64(dx) - bandWidth*float64(int(float64(dx)/bandWidth))) / bandWidth
		paint.SetColor(a.Lerp(b, frac))
		x := start + shadowDirection*float64(dx)
		canvas.DrawRect(ui.NewRectFromLTWH(x-1, offset.Dy, 1, shadowHeight), paint)
	}
}

// Dispose has nothing to discard.
func (p *edgeShadowPainter) Dispose() {}

// The spring of the modal popup and the dialog.
var (
	standardSpring    = physics.NewSpringDescription(1, 522.35, 45.7099552)
	standardTolerance = physics.Tolerance{Velocity: 0.03}
)

// standardSimulation is the spring that moves a popup or a dialog from the value of the controller to the end.
func standardSimulation(controller *animation.AnimationController, forward bool) physics.Simulation {
	end := 0.0
	if forward {
		end = 1.0
	}
	return physics.NewSpringSimulation(standardSpring, controller.Value(), end, 0,
		physics.SpringSimulationOpts{Tolerance: foundation.Some(standardTolerance), SnapToEnd: true})
}

// CupertinoModalPopupRoute is a route that shows a modal iOS style popup surface: it slides up from the
// bottom on a spring, over a barrier.
//
// Flutter: cupertino/route.dart CupertinoModalPopupRoute
type CupertinoModalPopupRoute struct {
	widgets.PopupRouteBase
	builder              widgets.WidgetBuilder
	barrierLabel         string
	barrierColor         foundation.Opt[ui.Color]
	barrierDismissible   bool
	semanticsDismissible bool
}

// CupertinoModalPopupRouteOpts are the optional parameters of NewCupertinoModalPopupRoute.
//
// Flutter: cupertino/route.dart CupertinoModalPopupRoute.new
type CupertinoModalPopupRouteOpts struct {
	widgets.ModalRouteOpts
	// BarrierLabel is the semantic label of the barrier; the default is "Dismiss".
	BarrierLabel foundation.Opt[string]
	// BarrierColor is the color of the barrier; the default is CupertinoModalBarrierColor (not resolved: the
	// function ShowCupertinoModalPopup resolves it).
	BarrierColor foundation.Opt[ui.Color]
	// NoBarrierColor makes the barrier colorless.
	NoBarrierColor bool
	// NotBarrierDismissible makes a tap on the barrier not pop the route (the barrier is dismissible by default).
	NotBarrierDismissible bool
	// SemanticsDismissible is whether the barrier can be dismissed by assistive technology; the default is false.
	SemanticsDismissible bool
}

// NewCupertinoModalPopupRoute creates a route whose content is built by builder.
//
// Deviation: Flutter's anchorPoint (the screen of a display feature to show it on) is not ported, as Glutter
// has no display features (SPEC §12.10).
//
// Flutter: cupertino/route.dart CupertinoModalPopupRoute.new
func NewCupertinoModalPopupRoute(builder widgets.WidgetBuilder, opts ...CupertinoModalPopupRouteOpts) *CupertinoModalPopupRoute {
	o := firstOf(opts)
	r := &CupertinoModalPopupRoute{
		builder: builder, barrierLabel: o.BarrierLabel.OrElse("Dismiss"),
		barrierDismissible: !o.NotBarrierDismissible, semanticsDismissible: o.SemanticsDismissible,
	}
	if !o.NoBarrierColor {
		r.barrierColor = foundation.Some(o.BarrierColor.OrElse(CupertinoModalBarrierColor.EffectiveColor()))
	}
	widgets.InitPopupRouteBase(&r.PopupRouteBase, r, o.ModalRouteOpts)
	return r
}

// BarrierLabel is the semantic label of the barrier.
//
// Flutter: cupertino/route.dart CupertinoModalPopupRoute.barrierLabel
func (r *CupertinoModalPopupRoute) BarrierLabel() string { return r.barrierLabel }

// BarrierColor is the color of the barrier.
//
// Flutter: cupertino/route.dart CupertinoModalPopupRoute.barrierColor
func (r *CupertinoModalPopupRoute) BarrierColor() foundation.Opt[ui.Color] { return r.barrierColor }

// BarrierDismissible is whether tapping the barrier pops the route.
//
// Flutter: cupertino/route.dart CupertinoModalPopupRoute.barrierDismissible
func (r *CupertinoModalPopupRoute) BarrierDismissible() bool { return r.barrierDismissible }

// SemanticsDismissible is whether assistive technology can dismiss the barrier.
//
// Flutter: cupertino/route.dart CupertinoModalPopupRoute.semanticsDismissible
func (r *CupertinoModalPopupRoute) SemanticsDismissible() bool { return r.semanticsDismissible }

// TransitionDuration is 335 ms.
//
// Flutter: cupertino/route.dart CupertinoModalPopupRoute.transitionDuration
func (r *CupertinoModalPopupRoute) TransitionDuration() time.Duration {
	return modalPopupTransitionDuration
}

// CreateSimulation is the spring that moves the popup.
//
// Flutter: cupertino/route.dart CupertinoModalPopupRoute.createSimulation
func (r *CupertinoModalPopupRoute) CreateSimulation(forward bool) physics.Simulation {
	return standardSimulation(r.Controller(), forward)
}

// BuildPage builds the popup at the elevated interface level.
//
// Flutter: cupertino/route.dart CupertinoModalPopupRoute.buildPage
func (r *CupertinoModalPopupRoute) BuildPage(bc widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
	return NewCupertinoUserInterfaceLevel(CupertinoUserInterfaceLevelElevated, widgets.NewBuilder(r.builder))
}

// BuildTransitions slides the popup up from the bottom.
//
// Flutter: cupertino/route.dart CupertinoModalPopupRoute.buildTransitions
func (r *CupertinoModalPopupRoute) BuildTransitions(_ widgets.BuildContext, a, _ animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return widgets.NewAlign(widgets.NewFractionalTranslation(bottomUpTween().Evaluate(a), child),
		widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})
}

// ShowCupertinoModalPopupOpts are the optional parameters of ShowCupertinoModalPopup.
//
// Flutter: cupertino/route.dart showCupertinoModalPopup
type ShowCupertinoModalPopupOpts struct {
	// Filter is the filter applied to what is behind the barrier, such as a blur.
	Filter ui.ImageFilter
	// BarrierColor is the color of the barrier; the default is CupertinoModalBarrierColor, resolved in the context.
	BarrierColor ui.ColorValue
	// NotBarrierDismissible makes a tap on the barrier not pop the popup.
	NotBarrierDismissible bool
	// NotUseRootNavigator pushes the popup onto the closest navigator instead of the root one.
	NotUseRootNavigator bool
	// SemanticsDismissible is whether the barrier can be dismissed by assistive technology.
	SemanticsDismissible bool
	// RouteSettings are the settings of the route.
	RouteSettings widgets.RouteSettings
	// RequestFocus is whether the popup requests the focus.
	RequestFocus foundation.Opt[bool]
}

// ShowCupertinoModalPopup shows a modal iOS style popup that slides up from the bottom of the screen; such a
// popup is an alternative to a menu or a dialog and prevents the user from interacting with the rest of the
// app. It returns a channel that gets the value the popup is popped with, then is closed.
//
// Flutter: cupertino/route.dart showCupertinoModalPopup
func ShowCupertinoModalPopup(bc widgets.BuildContext, builder widgets.WidgetBuilder, opts ...ShowCupertinoModalPopupOpts) <-chan any {
	o := firstOf(opts)
	var barrier ui.ColorValue = CupertinoModalBarrierColor
	if o.BarrierColor != nil {
		barrier = o.BarrierColor
	}
	return widgets.NavigatorOf(bc, widgets.NavigatorOfOpts{RootNavigator: !o.NotUseRootNavigator}).Push(NewCupertinoModalPopupRoute(builder, CupertinoModalPopupRouteOpts{
		ModalRouteOpts: widgets.ModalRouteOpts{RouteOpts: widgets.RouteOpts{Settings: o.RouteSettings, RequestFocus: o.RequestFocus}, Filter: o.Filter},
		BarrierColor:   foundation.Some(CupertinoDynamicColorResolve(barrier, bc)), NotBarrierDismissible: o.NotBarrierDismissible,
		SemanticsDismissible: o.SemanticsDismissible,
	}))
}

// CupertinoDialogRouteOpts are the optional parameters of NewCupertinoDialogRoute and ShowCupertinoDialog.
//
// Flutter: cupertino/route.dart CupertinoDialogRoute.new
type CupertinoDialogRouteOpts struct {
	// BarrierDismissible is whether tapping the barrier pops the dialog; the default is false.
	BarrierDismissible bool
	// BarrierColor is the color of the barrier; the default is CupertinoModalBarrierColor, resolved in the context.
	BarrierColor foundation.Opt[ui.Color]
	// BarrierLabel is the semantic label of the barrier; the default is the modal barrier dismiss label of the
	// localizations.
	BarrierLabel foundation.Opt[string]
	// TransitionDuration is the duration of the transition; the default is 250 ms.
	TransitionDuration foundation.Opt[time.Duration]
	// TransitionBuilder builds the transition of the dialog; the default fades and scales it in on a spring.
	TransitionBuilder widgets.RouteTransitionsBuilder
	// RouteSettings are the settings of the route.
	RouteSettings widgets.RouteSettings
	// RequestFocus is whether the dialog requests the focus.
	RequestFocus foundation.Opt[bool]
	// NotUseRootNavigator (ShowCupertinoDialog) pushes the dialog onto the closest navigator.
	NotUseRootNavigator bool
}

// NewCupertinoDialogRoute creates a route for an iOS style dialog: it fades in and scales from 1.3 to 1 on a
// spring, and fades out. The barrier label comes from CupertinoLocalizations.
//
// Deviation: Flutter's CupertinoDialogRoute is a subclass of RawDialogRoute; the route here is a RawDialogRoute
// with the spring and the transition (SPEC §12.10).
//
// Flutter: cupertino/route.dart CupertinoDialogRoute.new
func NewCupertinoDialogRoute(bc widgets.BuildContext, builder widgets.WidgetBuilder, opts ...CupertinoDialogRouteOpts) *widgets.RawDialogRoute {
	o := firstOf(opts)
	label := o.BarrierLabel.OrElse("")
	if !o.BarrierLabel.IsSome() {
		label = CupertinoLocalizationsOf(bc).ModalBarrierDismissLabel()
	}
	color := o.BarrierColor.OrElse(CupertinoDynamicColorResolve(CupertinoModalBarrierColor, bc))
	var route *widgets.RawDialogRoute
	transition := o.TransitionBuilder
	custom := transition != nil
	if !custom {
		transition = func(_ widgets.BuildContext, a, _ animation.Animation[float64], child widgets.Widget) widgets.Widget {
			if a.Status() == animation.AnimationStatusReverse {
				return widgets.NewFadeTransition(a, child)
			}
			return widgets.NewFadeTransition(a, widgets.NewScaleTransition(
				animation.Drive(a, animation.NewTween(1.3, 1.0)), child))
		}
	}
	route = widgets.NewRawDialogRoute(func(bc widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
		return builder(bc)
	}, widgets.RawDialogRouteOpts{
		ModalRouteOpts:     widgets.ModalRouteOpts{RouteOpts: widgets.RouteOpts{Settings: o.RouteSettings, RequestFocus: o.RequestFocus}},
		BarrierDismissible: foundation.Some(o.BarrierDismissible),
		BarrierColor:       foundation.Some(color),
		BarrierLabel:       label,
		TransitionDuration: foundation.Some(o.TransitionDuration.OrElse(250 * time.Millisecond)),
		TransitionBuilder:  transition,
		Simulation:         func(forward bool) physics.Simulation { return standardSimulation(route.Controller(), forward) },
	})
	return route
}

// ShowCupertinoDialog displays an iOS style dialog above the current contents of the app, with iOS style
// entrance and exit animations, modal barrier color, and modal barrier behavior (by default the dialog is
// not dismissible with a tap on the barrier). It returns a channel that gets the value the dialog is popped
// with, then is closed.
//
// Flutter: cupertino/route.dart showCupertinoDialog
func ShowCupertinoDialog(bc widgets.BuildContext, builder widgets.WidgetBuilder, opts ...CupertinoDialogRouteOpts) <-chan any {
	o := firstOf(opts)
	return widgets.NavigatorOf(bc, widgets.NavigatorOfOpts{RootNavigator: !o.NotUseRootNavigator}).Push(NewCupertinoDialogRoute(bc, builder, o))
}

// CupertinoPageTransitionsBuilder builds the iOS page transition for a page route: the parallax slide with
// the back swipe. It is what a PageTransitionsTheme uses for iOS and macOS.
//
// Flutter: cupertino/route.dart CupertinoPageTransitionsBuilder
type CupertinoPageTransitionsBuilder struct{}

// DelegatedTransition is the slide of the route below.
//
// Flutter: cupertino/route.dart CupertinoPageTransitionsBuilder.delegatedTransition
func (CupertinoPageTransitionsBuilder) DelegatedTransition() widgets.DelegatedTransitionBuilder {
	return CupertinoPageTransitionDelegatedTransition
}

// TransitionDuration is 500 ms.
//
// Flutter: cupertino/route.dart CupertinoPageTransitionsBuilder.transitionDuration
func (CupertinoPageTransitionsBuilder) TransitionDuration() time.Duration {
	return CupertinoRouteTransitionDuration
}

// ReverseTransitionDuration is 500 ms.
//
// Flutter: cupertino/route.dart CupertinoPageTransitionsBuilder.reverseTransitionDuration
func (CupertinoPageTransitionsBuilder) ReverseTransitionDuration() time.Duration {
	return CupertinoRouteTransitionDuration
}

// BuildTransitions builds the transitions of the route.
//
// Flutter: cupertino/route.dart CupertinoPageTransitionsBuilder.buildTransitions
func (CupertinoPageTransitionsBuilder) BuildTransitions(route widgets.PageRoute, bc widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return CupertinoRouteBuildPageTransitions(route, bc, a, secondaryAnimation, child)
}

// NewCupertinoPageTransitionsBuilder creates the builder.
//
// Flutter: cupertino/route.dart CupertinoPageTransitionsBuilder.new
func NewCupertinoPageTransitionsBuilder() CupertinoPageTransitionsBuilder {
	return CupertinoPageTransitionsBuilder{}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoRoutePreviousTitleBeforeInstall = "CupertinoRoutePreviousTitleBeforeInstall"
	tagCupertinoPopGestureDisabled              = "CupertinoPopGestureDisabled"
)
