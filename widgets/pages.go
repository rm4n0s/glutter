package widgets

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// PageRoute is a modal route that replaces the entire screen.
//
// Flutter: widgets/pages.dart PageRoute
type PageRoute interface {
	ModalRoute
	pageRouteBase() *PageRouteBase
}

// PageRouteOpts are the optional parameters of the page route bases.
//
// Flutter: widgets/pages.dart PageRoute.new
type PageRouteOpts struct {
	RouteOpts
	// FullscreenDialog is whether the route is a fullscreen dialog.
	FullscreenDialog bool
	// DisallowSnapshotting makes AllowSnapshotting false; snapshotting is allowed
	// by default.
	DisallowSnapshotting bool
	// BarrierDismissible is whether tapping the barrier pops the route.
	BarrierDismissible bool
	// TraversalEdgeBehavior controls the transfer of focus beyond the first and the
	// last items of the focus scope of the route; Unspecified uses
	// Navigator.RouteTraversalEdgeBehavior.
	TraversalEdgeBehavior TraversalEdgeBehavior
	// DirectionalTraversalEdgeBehavior controls the directional transfer of focus
	// beyond the first and the last items of the focus scope of the route;
	// Unspecified uses Navigator.RouteDirectionalTraversalEdgeBehavior.
	DirectionalTraversalEdgeBehavior TraversalEdgeBehavior
}

// PageRouteBase implements PageRoute: it is opaque and only transitions with
// other page routes. Embed it, call InitPageRouteBase and implement BuildPage,
// MaintainState, TransitionDuration, BarrierColor and BarrierLabel.
//
// Flutter: widgets/pages.dart PageRoute
type PageRouteBase struct {
	ModalRouteBase
	pself                PageRoute
	fullscreenDialog     bool
	disallowSnapshotting bool
	barrierDismissible   bool
}

// InitPageRouteBase sets the outer value of the route.
//
// Flutter: widgets/pages.dart PageRoute.new
func InitPageRouteBase(b *PageRouteBase, self PageRoute, opts PageRouteOpts) {
	InitModalRouteBase(&b.ModalRouteBase, self, ModalRouteOpts{RouteOpts: opts.RouteOpts,
		TraversalEdgeBehavior: opts.TraversalEdgeBehavior, DirectionalTraversalEdgeBehavior: opts.DirectionalTraversalEdgeBehavior})
	b.pself = self
	b.fullscreenDialog = opts.FullscreenDialog
	b.disallowSnapshotting = opts.DisallowSnapshotting
	b.barrierDismissible = opts.BarrierDismissible
}

func (b *PageRouteBase) pageRouteBase() *PageRouteBase { return b }

// FullscreenDialog is whether the route is a fullscreen dialog.
//
// Flutter: widgets/pages.dart PageRoute.fullscreenDialog
func (b *PageRouteBase) FullscreenDialog() bool { return b.fullscreenDialog }

// AllowSnapshotting is whether the transitions of the route may be snapshotted.
//
// Flutter: widgets/pages.dart PageRoute.allowSnapshotting
func (b *PageRouteBase) AllowSnapshotting() bool { return !b.disallowSnapshotting }

// Opaque is true.
//
// Flutter: widgets/pages.dart PageRoute.opaque
func (b *PageRouteBase) Opaque() bool { return true }

// BarrierDismissible is whether tapping the barrier pops the route.
//
// Flutter: widgets/pages.dart PageRoute.barrierDismissible
func (b *PageRouteBase) BarrierDismissible() bool { return b.barrierDismissible }

// CanTransitionTo is whether the next route is a page route.
//
// Flutter: widgets/pages.dart PageRoute.canTransitionTo
func (b *PageRouteBase) CanTransitionTo(nextRoute TransitionRoute) bool {
	_, ok := nextRoute.(PageRoute)
	return ok
}

// CanTransitionFrom is whether the previous route is a page route.
//
// Flutter: widgets/pages.dart PageRoute.canTransitionFrom
func (b *PageRouteBase) CanTransitionFrom(previousRoute TransitionRoute) bool {
	_, ok := previousRoute.(PageRoute)
	return ok
}

// PopGestureEnabled is whether a back gesture may pop the route; fullscreen
// dialogs can't be dismissed by a back swipe.
//
// Flutter: widgets/pages.dart PageRoute.popGestureEnabled
func (b *PageRouteBase) PopGestureEnabled() bool {
	return !b.fullscreenDialog && b.ModalRouteBase.PopGestureEnabled()
}

// PageRouteBuilderOpts are the optional parameters of NewPageRouteBuilder.
//
// Flutter: widgets/pages.dart PageRouteBuilder.new
type PageRouteBuilderOpts struct {
	PageRouteOpts
	// TransitionsBuilder builds the transition of the page; the default returns
	// the child.
	TransitionsBuilder RouteTransitionsBuilder
	// TransitionDuration is the duration of the transition in; the default is
	// 300 ms.
	TransitionDuration foundation.Opt[time.Duration]
	// ReverseTransitionDuration is the duration of the transition out; the
	// default is 300 ms.
	ReverseTransitionDuration foundation.Opt[time.Duration]
	// NotOpaque makes the route not obscure the previous one; routes are opaque
	// by default.
	NotOpaque bool
	// BarrierColor is the color of the barrier, if any.
	BarrierColor foundation.Opt[ui.Color]
	// BarrierLabel is the semantic label of the barrier.
	BarrierLabel string
	// DiscardState makes the route lose its state when it is not on top; the
	// state is kept by default.
	DiscardState bool
}

// PageRouteBuilder is a utility class for creating page routes with callbacks
// for the page and the transitions.
//
// Flutter: widgets/pages.dart PageRouteBuilder
type PageRouteBuilder struct {
	PageRouteBase
	pageBuilder        RoutePageBuilder
	transitionsBuilder RouteTransitionsBuilder
	transitionDuration time.Duration
	reverseDuration    time.Duration
	opaque             bool
	barrierColor       foundation.Opt[ui.Color]
	barrierLabel       string
	maintainState      bool
}

// NewPageRouteBuilder creates a route from the page builder.
//
// Flutter: widgets/pages.dart PageRouteBuilder.new
func NewPageRouteBuilder(pageBuilder RoutePageBuilder, opts ...PageRouteBuilderOpts) *PageRouteBuilder {
	o := firstOpts(opts)
	r := &PageRouteBuilder{
		pageBuilder:        pageBuilder,
		transitionsBuilder: o.TransitionsBuilder,
		transitionDuration: o.TransitionDuration.OrElse(300 * time.Millisecond),
		reverseDuration:    o.ReverseTransitionDuration.OrElse(300 * time.Millisecond),
		opaque:             !o.NotOpaque,
		barrierColor:       o.BarrierColor,
		barrierLabel:       o.BarrierLabel,
		maintainState:      !o.DiscardState,
	}
	if r.transitionsBuilder == nil {
		r.transitionsBuilder = func(_ BuildContext, _, _ animation.Animation[float64], child Widget) Widget { return child }
	}
	InitPageRouteBase(&r.PageRouteBase, r, o.PageRouteOpts)
	return r
}

// TransitionDuration is the duration of the transition in.
//
// Flutter: widgets/pages.dart PageRouteBuilder.transitionDuration
func (r *PageRouteBuilder) TransitionDuration() time.Duration { return r.transitionDuration }

// ReverseTransitionDuration is the duration of the transition out.
//
// Flutter: widgets/pages.dart PageRouteBuilder.reverseTransitionDuration
func (r *PageRouteBuilder) ReverseTransitionDuration() time.Duration { return r.reverseDuration }

// Opaque is whether the route obscures the previous one.
//
// Flutter: widgets/pages.dart PageRouteBuilder.opaque
func (r *PageRouteBuilder) Opaque() bool { return r.opaque }

// BarrierColor is the color of the barrier, if any.
//
// Flutter: widgets/pages.dart PageRouteBuilder.barrierColor
func (r *PageRouteBuilder) BarrierColor() foundation.Opt[ui.Color] { return r.barrierColor }

// BarrierLabel is the semantic label of the barrier.
//
// Flutter: widgets/pages.dart PageRouteBuilder.barrierLabel
func (r *PageRouteBuilder) BarrierLabel() string { return r.barrierLabel }

// MaintainState is whether the route keeps its state while it is not on top.
//
// Flutter: widgets/pages.dart PageRouteBuilder.maintainState
func (r *PageRouteBuilder) MaintainState() bool { return r.maintainState }

// BuildPage calls the page builder.
//
// Flutter: widgets/pages.dart PageRouteBuilder.buildPage
func (r *PageRouteBuilder) BuildPage(bc BuildContext, a, secondary animation.Animation[float64]) Widget {
	return r.pageBuilder(bc, a, secondary)
}

// BuildTransitions calls the transitions builder.
//
// Flutter: widgets/pages.dart PageRouteBuilder.buildTransitions
func (r *PageRouteBuilder) BuildTransitions(bc BuildContext, a, secondary animation.Animation[float64], child Widget) Widget {
	return r.transitionsBuilder(bc, a, secondary, child)
}
