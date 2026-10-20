package material

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// MaterialRoute is a page route that Material makes: it builds its content with
// BuildContent and its transitions with the page transitions theme.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin
type MaterialRoute interface {
	widgets.PageRoute
	// BuildContent builds the primary contents of the route.
	//
	// Flutter: material/page.dart MaterialRouteTransitionMixin.buildContent
	BuildContent(bc widgets.BuildContext) widgets.Widget
	materialRouteBase() *MaterialRouteBase
}

// MaterialRouteBase is the part of a Material page route that Flutter's
// MaterialRouteTransitionMixin gives to a PageRoute: the transitions of the
// theme of the app, which a platform adaptive route needs. Embed it in a page
// route, and call InitMaterialRouteBase with the outer route.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin
type MaterialRouteBase struct {
	widgets.PageRouteBase
	mself MaterialRoute
}

// InitMaterialRouteBase sets the outer value of the route.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin
func InitMaterialRouteBase(b *MaterialRouteBase, self MaterialRoute, opts widgets.PageRouteOpts) {
	b.mself = self
	widgets.InitPageRouteBase(&b.PageRouteBase, self, opts)
}

func (b *MaterialRouteBase) materialRouteBase() *MaterialRouteBase { return b }

// pageTransitionsThemeAndPlatform are the page transitions theme and the platform
// of the theme above the navigator.
func (b *MaterialRouteBase) pageTransitionsThemeAndPlatform() (*PageTransitionsTheme, foundation.TargetPlatform) {
	theme := ThemeOf(b.Navigator().Context())
	return theme.PageTransitionsTheme, theme.Platform
}

func (b *MaterialRouteBase) transitionBuilder() widgets.PageTransitionsBuilder {
	theme, platform := b.pageTransitionsThemeAndPlatform()
	return theme.builderFor(platform)
}

// TransitionDuration is the duration of the transition of the builder of the
// platform.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin.transitionDuration
func (b *MaterialRouteBase) TransitionDuration() time.Duration {
	return b.transitionBuilder().TransitionDuration()
}

// ReverseTransitionDuration is the duration of the transition out of the
// builder of the platform.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin.reverseTransitionDuration
func (b *MaterialRouteBase) ReverseTransitionDuration() time.Duration {
	return b.transitionBuilder().ReverseTransitionDuration()
}

// DidPush sets the duration of the controller, as the theme may have changed
// since the route was installed, and starts the transition in.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin.didPush
func (b *MaterialRouteBase) DidPush() *scheduler.TickerFuture {
	if c := b.Controller(); c != nil {
		c.Duration = foundation.Some(b.mself.TransitionDuration())
	}
	return b.PageRouteBase.DidPush()
}

// DidPop sets the reverse duration of the controller and starts the transition
// out.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin.didPop
func (b *MaterialRouteBase) DidPop(result any) bool {
	if c := b.Controller(); c != nil {
		c.ReverseDuration = foundation.Some(b.mself.ReverseTransitionDuration())
	}
	return b.PageRouteBase.DidPop(result)
}

// BarrierColor is none: a page has no barrier.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin.barrierColor
func (b *MaterialRouteBase) BarrierColor() foundation.Opt[ui.Color] {
	return foundation.None[ui.Color]()
}

// BarrierLabel is empty.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin.barrierLabel
func (b *MaterialRouteBase) BarrierLabel() string { return "" }

// materialDelegatedTransition is the transition that a Material route gives to
// the route below it: the one of the page transitions theme. It is a function
// of the package so that every Material route has the same one.
func materialDelegatedTransition(bc widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], allowSnapshotting bool, child widgets.Widget) widgets.Widget {
	theme := ThemeOf(bc)
	delegated := theme.PageTransitionsTheme.DelegatedTransition(theme.Platform)
	if delegated == nil {
		return nil
	}
	return delegated(bc, a, secondaryAnimation, allowSnapshotting, child)
}

// DelegatedTransition is the transition that the route below takes.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin.delegatedTransition
func (b *MaterialRouteBase) DelegatedTransition() widgets.DelegatedTransitionBuilder {
	return materialDelegatedTransition
}

// CanTransitionTo is whether the next route is a Material route, or has a
// delegated transition, and is not a fullscreen dialog.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin.canTransitionTo
func (b *MaterialRouteBase) CanTransitionTo(nextRoute widgets.TransitionRoute) bool {
	next, isPage := nextRoute.(widgets.PageRoute)
	nextIsNotFullscreen := !isPage || !next.FullscreenDialog()
	_, nextIsMaterial := nextRoute.(MaterialRoute)
	nextHasDelegated := false
	if modal, ok := nextRoute.(widgets.ModalRoute); ok {
		nextHasDelegated = modal.DelegatedTransition() != nil
	}
	return nextIsNotFullscreen && (nextIsMaterial || nextHasDelegated)
}

// CanTransitionFrom is whether the previous route is a page route and this one
// is not a fullscreen dialog.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin.canTransitionFrom
func (b *MaterialRouteBase) CanTransitionFrom(previousRoute widgets.TransitionRoute) bool {
	_, ok := previousRoute.(widgets.PageRoute)
	return ok && !b.mself.FullscreenDialog()
}

// BuildPage builds the content in a semantics scope of the route.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin.buildPage
func (b *MaterialRouteBase) BuildPage(bc widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
	return widgets.NewSemantics(b.mself.BuildContent(bc), widgets.SemanticsOpts{
		ScopesRoute: foundation.Some(true), ExplicitChildNodes: true,
	})
}

// BuildTransitions builds the transitions of the page transitions theme.
//
// Flutter: material/page.dart MaterialRouteTransitionMixin.buildTransitions
func (b *MaterialRouteBase) BuildTransitions(bc widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], child widgets.Widget) widgets.Widget {
	return ThemeOf(bc).PageTransitionsTheme.BuildTransitions(b.mself, bc, a, secondaryAnimation, child)
}

// MaterialPageRouteOpts are the optional parameters of NewMaterialPageRoute.
//
// Flutter: material/page.dart MaterialPageRoute.new
type MaterialPageRouteOpts struct {
	widgets.PageRouteOpts
	// MaintainState is whether the route keeps its state in memory when it is not
	// the current one; the default is true.
	MaintainState foundation.Opt[bool]
}

// MaterialPageRoute is a modal route that replaces the entire screen with a
// platform-adaptive transition.
//
// Flutter: material/page.dart MaterialPageRoute
type MaterialPageRoute struct {
	MaterialRouteBase
	builder       widgets.WidgetBuilder
	maintainState bool
}

// NewMaterialPageRoute creates a route whose content is built by builder.
//
// Flutter: material/page.dart MaterialPageRoute.new
func NewMaterialPageRoute(builder widgets.WidgetBuilder, opts ...MaterialPageRouteOpts) *MaterialPageRoute {
	o := firstOpts(opts)
	r := &MaterialPageRoute{builder: builder, maintainState: o.MaintainState.OrElse(true)}
	InitMaterialRouteBase(&r.MaterialRouteBase, r, o.PageRouteOpts)
	return r
}

// Builder builds the primary contents of the route.
//
// Flutter: material/page.dart MaterialPageRoute.builder
func (r *MaterialPageRoute) Builder() widgets.WidgetBuilder { return r.builder }

// BuildContent builds the primary contents of the route.
//
// Flutter: material/page.dart MaterialPageRoute.buildContent
func (r *MaterialPageRoute) BuildContent(bc widgets.BuildContext) widgets.Widget {
	return r.builder(bc)
}

// MaintainState is whether the route keeps its state when it is not the current
// one.
//
// Flutter: material/page.dart MaterialPageRoute.maintainState
func (r *MaterialPageRoute) MaintainState() bool { return r.maintainState }

// DebugLabel names the route and its settings.
//
// Flutter: material/page.dart MaterialPageRoute.debugLabel
func (r *MaterialPageRoute) DebugLabel() string {
	return r.MaterialRouteBase.DebugLabel() + "(" + r.Settings().Name() + ")"
}

// MaterialPageOpts are the optional parameters of NewMaterialPage.
//
// Flutter: material/page.dart MaterialPage.new
type MaterialPageOpts struct {
	widgets.PageOpts
	// MaintainState is whether the route keeps its state in memory when it is not
	// the current one; the default is true.
	MaintainState foundation.Opt[bool]
	// FullscreenDialog is whether the route is a full-screen dialog.
	FullscreenDialog bool
	// DisallowSnapshotting is true to keep the transition of the route from using
	// a snapshot of it.
	DisallowSnapshotting bool
}

// MaterialPage is a page that creates a Material style PageRoute.
//
// Flutter: material/page.dart MaterialPage
type MaterialPage struct {
	widgets.PageBase
	child                widgets.Widget
	maintainState        bool
	fullscreenDialog     bool
	disallowSnapshotting bool
}

// NewMaterialPage creates a page of the child.
//
// Flutter: material/page.dart MaterialPage.new
func NewMaterialPage(child widgets.Widget, opts ...MaterialPageOpts) *MaterialPage {
	o := firstOpts(opts)
	p := &MaterialPage{child: child, maintainState: o.MaintainState.OrElse(true), fullscreenDialog: o.FullscreenDialog,
		disallowSnapshotting: o.DisallowSnapshotting}
	widgets.InitPageBase(&p.PageBase, p, o.PageOpts)
	return p
}

// Child is the content of the route.
//
// Flutter: material/page.dart MaterialPage.child
func (p *MaterialPage) Child() widgets.Widget { return p.child }

// MaintainState is whether the route keeps its state when it is not the current
// one.
//
// Flutter: material/page.dart MaterialPage.maintainState
func (p *MaterialPage) MaintainState() bool { return p.maintainState }

// FullscreenDialog is whether the route is a full-screen dialog.
//
// Flutter: material/page.dart MaterialPage.fullscreenDialog
func (p *MaterialPage) FullscreenDialog() bool { return p.fullscreenDialog }

// AllowSnapshotting is whether the transition of the route may use a snapshot.
//
// Flutter: material/page.dart MaterialPage.allowSnapshotting
func (p *MaterialPage) AllowSnapshotting() bool { return !p.disallowSnapshotting }

// CreateRoute creates the page based route.
//
// Flutter: material/page.dart MaterialPage.createRoute
func (p *MaterialPage) CreateRoute(widgets.BuildContext) widgets.Route {
	r := &pageBasedMaterialPageRoute{}
	InitMaterialRouteBase(&r.MaterialRouteBase, r, widgets.PageRouteOpts{
		RouteOpts:            widgets.RouteOpts{Settings: p},
		FullscreenDialog:     p.fullscreenDialog,
		DisallowSnapshotting: p.disallowSnapshotting,
	})
	return r
}

// pageBasedMaterialPageRoute is the route of a MaterialPage: the page, which is
// the settings of the route, gives it its content; the navigator replaces the
// settings when the page changes.
type pageBasedMaterialPageRoute struct {
	MaterialRouteBase
}

func (r *pageBasedMaterialPageRoute) page() *MaterialPage { return r.Settings().(*MaterialPage) }

func (r *pageBasedMaterialPageRoute) BuildContent(widgets.BuildContext) widgets.Widget {
	return r.page().child
}

func (r *pageBasedMaterialPageRoute) MaintainState() bool { return r.page().maintainState }

func (r *pageBasedMaterialPageRoute) FullscreenDialog() bool { return r.page().fullscreenDialog }

func (r *pageBasedMaterialPageRoute) DebugLabel() string {
	return r.MaterialRouteBase.DebugLabel() + "(" + r.page().Name() + ")"
}
