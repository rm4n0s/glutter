package widgets

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// NavigatorDefaultRouteName is the name of the default route: "/".
//
// Flutter: widgets/navigator.dart Navigator.defaultRouteName
const NavigatorDefaultRouteName = "/"

// routeLifecycle is the state of a routeEntry. The state machine only goes down:
// staging, add/push/pushReplace/replace, adding/pushing, idle, pop/complete/
// remove, popping/removing, dispose, disposing, disposed.
//
// Flutter: widgets/navigator.dart _RouteLifecycle
type routeLifecycle int

const (
	routeStaging     routeLifecycle = iota // waits for the transition delegate to decide what to do with the route
	routeAdd                               // install, didAdd, ...: a route from onGenerateInitialRoutes or the initial pages
	routeAdding                            // waits for the future from didPush of the top-most route
	routePush                              // install, didPush, ...: a route added via push() and friends
	routePushReplace                       // install, didPush, ...: a route added via pushReplace() and friends
	routePushing                           // waits for the future from didPush
	routeReplace                           // install, didReplace, ...: a route added via replace() and friends
	routeIdle                              // the route is being harmless
	routePop                               // will call didPop
	routeComplete                          // will call didComplete
	routeRemove                            // will run didReplace/didRemove etc
	routePopping                           // waits for the route to call finalizeRoute to switch to dispose
	routeRemoving                          // waits for the subsequent routes to be done animating, then switches to dispose
	routeDispose                           // will dispose the route
	routeDisposing                         // waits for the widget subtree to be disposed first
	routeDisposed                          // the route has been disposed
)

// routeAnnouncement is the value of the last route that was announced to a
// route; before any announcement it is not announced.
type routeAnnouncement struct {
	announced bool
	route     Route
}

func announced(r Route) routeAnnouncement { return routeAnnouncement{true, r} }

// routeEntry is the history element of the navigator: a route with its state
// in the lifecycle.
//
// Flutter: widgets/navigator.dart _RouteEntry
type routeEntry struct {
	route        Route
	pageBased    bool
	currentState routeLifecycle

	lastAnnouncedPreviousRoute   routeAnnouncement // last argument to Route.didChangePrevious
	lastAnnouncedPoppedNextRoute routeAnnouncement // last argument to Route.didPopNext
	lastAnnouncedNextRoute       routeAnnouncement // last argument to Route.didChangeNext

	// imperativeRemoval is whether this route is removed without the Pages API:
	// Navigator.pop or Navigator.pushReplacement, for example.
	imperativeRemoval bool

	pendingResult           any
	reportRemovalToObserver bool
}

func newRouteEntry(route Route, initialState routeLifecycle, pageBased bool) *routeEntry {
	if foundation.DebugMode {
		if pageBased {
			if _, ok := route.Settings().(Page); !ok {
				panic(errors.New("RouteEntrySettingsNotAPage", "the settings of a page-based route must be a Page"))
			}
		}
		switch initialState {
		case routeStaging, routeAdd, routePush, routePushReplace, routeReplace:
		default:
			panic(errors.New("RouteEntryInitialStateInvalid", "a route entry starts in staging, add, push, pushReplace or replace",
				"state", int(initialState)))
		}
	}
	return &routeEntry{route: route, pageBased: pageBased, currentState: initialState, reportRemovalToObserver: true}
}

func (e *routeEntry) base() *RouteBase { return e.route.AsRoute() }

func (e *routeEntry) handleAdd(navigator *NavigatorState, previousPresent Route) {
	e.debugRequireState("handleAdd", routeAdd)
	navigator.debugRequireLocked()
	e.currentState = routeAdding
	navigator.observedRouteAdditions = append(navigator.observedRouteAdditions, &navigatorObservation{kind: observationPush, primary: e.route, secondary: previousPresent})
}

func (e *routeEntry) handlePush(navigator *NavigatorState, isNewFirst bool, previous, previousPresent Route) {
	if foundation.DebugMode {
		if e.currentState != routePush && e.currentState != routePushReplace && e.currentState != routeReplace {
			panic(errors.New("RouteEntryStateInvalid", "handlePush needs a route in the push, pushReplace or replace state",
				"state", int(e.currentState)))
		}
		if e.base().installed() {
			panic(errors.New("RouteAlreadyInstalled", "the pushed route has already been used; when pushing a route, a new Route object must be provided",
				"route", e.route.String()))
		}
	}
	navigator.debugRequireLocked()
	previousState := e.currentState
	e.base().navigator = navigator
	e.route.Install()
	if foundation.DebugMode && len(e.route.OverlayEntries()) == 0 {
		panic(errors.New("RouteOverlayEntriesEmpty", "a route must create at least one overlay entry in Install",
			"route", e.route.String()))
	}
	if e.currentState == routePush || e.currentState == routePushReplace {
		routeFuture := e.route.DidPush()
		e.currentState = routePushing
		binding := WidgetsBindingOf(navigator.Context())
		routeFuture.WhenCompleteOrCancel(func() {
			// Dart runs the callback of a future in a microtask, also when the
			// future is complete already, which happens with a zero transition
			// duration.
			binding.Post(func() {
				if e.currentState == routePushing {
					e.currentState = routeIdle
					navigator.debugRequireUnlocked()
					if foundation.DebugMode {
						navigator.debugLocked = true
					}
					navigator.flushHistoryUpdates(true)
					if foundation.DebugMode {
						navigator.debugLocked = false
					}
				}
			})
		})
	} else {
		e.route.DidReplace(previous)
		e.currentState = routeIdle
	}
	if isNewFirst {
		e.route.DidChangeNext(nil)
	}
	if previousState == routeReplace || previousState == routePushReplace {
		navigator.observedRouteAdditions = append(navigator.observedRouteAdditions, &navigatorObservation{kind: observationReplace, primary: e.route, secondary: previousPresent})
		// TODO(step 10.3): tell onDidRemovePage when previousPresent is page-based.
	} else {
		navigator.observedRouteAdditions = append(navigator.observedRouteAdditions, &navigatorObservation{kind: observationPush, primary: e.route, secondary: previousPresent})
	}
}

func (e *routeEntry) handleDidPopNext(poppedRoute Route) {
	e.route.DidPopNext(poppedRoute)
	e.lastAnnouncedPoppedNextRoute = announced(poppedRoute)
	// Deviation: Flutter also moves the accessibility focus back on Android and
	// iOS here (SPEC §12.10).
}

func (e *routeEntry) handlePop(navigator *NavigatorState, previousPresent Route) bool {
	navigator.debugRequireLocked()
	if foundation.DebugMode && !e.base().isInstalledIn(navigator) {
		panic(errors.New("RouteNotInstalledHere", "the route is not installed in this navigator", "route", e.route.String()))
	}
	e.currentState = routePopping
	if e.base().popCompleter.IsCompleted() {
		// This is a page-based route popped through Navigator.pop: didPop was
		// called already and nothing else needs to happen.
		return true
	}
	if !e.route.DidPop(e.pendingResult) {
		e.currentState = routeIdle
		return false
	}
	e.route.OnPopInvokedWithResult(true, e.pendingResult)
	// TODO(step 10.3): tell onDidRemovePage when the route is page-based and
	// removed imperatively.
	e.pendingResult = nil
	return true
}

func (e *routeEntry) handleComplete() {
	e.route.DidComplete(e.pendingResult)
	e.pendingResult = nil
	e.currentState = routeRemove
}

func (e *routeEntry) handleRemoval(navigator *NavigatorState, previousPresent Route) {
	navigator.debugRequireLocked()
	if e.base().isInstalledIn(navigator) {
		e.currentState = routeRemoving
	} else {
		// The route is still waiting to be added while a top-most push or pop
		// animation is going on, so nothing was initialized and it can be
		// disposed directly.
		e.currentState = routeDispose
	}
	if e.reportRemovalToObserver {
		navigator.observedRouteDeletions = append(navigator.observedRouteDeletions, &navigatorObservation{kind: observationRemove, primary: e.route, secondary: previousPresent})
	}
}

func (e *routeEntry) didAdd(navigator *NavigatorState, isNewFirst bool) {
	if foundation.DebugMode && e.base().installed() {
		panic(errors.New("RouteAlreadyInstalled", "the added route has already been used", "route", e.route.String()))
	}
	e.base().navigator = navigator
	e.route.Install()
	if foundation.DebugMode && len(e.route.OverlayEntries()) == 0 {
		panic(errors.New("RouteOverlayEntriesEmpty", "a route must create at least one overlay entry in Install",
			"route", e.route.String()))
	}
	e.route.DidAdd()
	e.currentState = routeIdle
	if isNewFirst {
		e.route.DidChangeNext(nil)
	}
}

func (e *routeEntry) pop(result any, imperativeRemoval bool) {
	e.pendingResult = result
	e.currentState = routePop
	e.imperativeRemoval = imperativeRemoval
}

// complete makes the route complete with the result and be removed.
func (e *routeEntry) complete(result any, isReplaced, imperativeRemoval bool) {
	if e.currentState >= routeRemove {
		return
	}
	e.reportRemovalToObserver = !isReplaced
	e.pendingResult = result
	e.currentState = routeComplete
	e.imperativeRemoval = imperativeRemoval
}

func (e *routeEntry) finalize() {
	e.currentState = routeDispose
}

func (e *routeEntry) forcedDispose() {
	e.currentState = routeDisposed
	e.route.Dispose()
}

func (e *routeEntry) dispose() {
	e.currentState = routeDisposing
	// If the overlay entries are still mounted, widgets in the route's subtree
	// may still reference resources from the route and the disposal of the route
	// is delayed until the overlay entries are no longer mounted. The overlay
	// entry is the root of the subtree of the route, so it is unmounted after
	// every other widget in the subtree.
	var mountedEntries []*OverlayEntry
	for _, oe := range e.route.OverlayEntries() {
		if oe.Mounted() {
			mountedEntries = append(mountedEntries, oe)
		}
	}
	if len(mountedEntries) == 0 {
		e.forcedDispose()
		return
	}
	mounted := len(mountedEntries)
	navigator := e.base().navigator
	navigator.entryWaitingForSubTreeDisposal = append(navigator.entryWaitingForSubTreeDisposal, e)
	binding := WidgetsBindingOf(navigator.Context())
	for _, entry := range mountedEntries {
		var handle foundation.ListenerHandle
		handle = entry.AddListener(func() {
			mounted--
			entry.RemoveListener(handle)
			if mounted == 0 {
				// This is a listener of one of the overlay entries of this route.
				// Disposing the route also disposes its overlay entries, and a
				// change notifier can't be disposed during its notification, so
				// the disposal waits a microtask.
				binding.Post(func() {
					if !navigator.removeEntryWaitingForSubTreeDisposal(e) {
						// The route was destroyed by the navigator's forced disposal.
						return
					}
					e.forcedDispose()
				})
			}
		})
	}
}

func (e *routeEntry) willBePresent() bool {
	return e.currentState <= routeIdle && e.currentState >= routeAdd
}

func (e *routeEntry) isPresent() bool {
	return e.currentState <= routeRemove && e.currentState >= routeAdd
}

func (e *routeEntry) suitableForAnnouncement() bool {
	return e.currentState <= routeRemoving && e.currentState >= routePush
}

func (e *routeEntry) suitableForTransitionAnimation() bool {
	return e.currentState <= routeRemove && e.currentState >= routePush
}

func (e *routeEntry) shouldAnnounceChangeToNext(nextRoute Route) bool {
	// Do not announce if next changes from a just popped route to nil: that
	// change was announced already by calling didPopNext.
	return nextRoute != nil || e.lastAnnouncedPoppedNextRoute != e.lastAnnouncedNextRoute
}

func routeEntryIsPresent(e *routeEntry) bool     { return e.isPresent() }
func routeEntryWillBePresent(e *routeEntry) bool { return e.willBePresent() }
func routeEntrySuitableForTransition(e *routeEntry) bool {
	return e.suitableForTransitionAnimation()
}

func routeEntryIsRoute(route Route) func(*routeEntry) bool {
	return func(e *routeEntry) bool { return e.route == route }
}

func (e *routeEntry) debugRequireState(method string, want routeLifecycle) {
	if foundation.DebugMode && e.currentState != want {
		panic(errors.New("RouteEntryStateInvalid", method+" was called on a route entry in the wrong state",
			"state", int(e.currentState), "want", int(want)))
	}
}

type observationKind int

const (
	observationPush observationKind = iota
	observationPop
	observationRemove
	observationReplace
)

// navigatorObservation is a change of the history that the observers are told
// about when the history was flushed.
//
// Flutter: widgets/navigator.dart _NavigatorObservation
type navigatorObservation struct {
	kind      observationKind
	primary   Route
	secondary Route
}

func (o *navigatorObservation) notify(observer NavigatorObserver) {
	switch o.kind {
	case observationPush:
		observer.DidPush(o.primary, o.secondary)
	case observationPop:
		observer.DidPop(o.primary, o.secondary)
	case observationRemove:
		observer.DidRemove(o.primary, o.secondary)
	case observationReplace:
		observer.DidReplace(o.primary, o.secondary)
	}
}

// navigatorHistory is the history of the navigator, a notifier that tells its
// listeners when entries were added or removed.
//
// Flutter: widgets/navigator.dart _History
type navigatorHistory struct {
	foundation.ChangeNotifier
	entries []*routeEntry
}

func (h *navigatorHistory) indexWhere(test func(*routeEntry) bool) int {
	return slices.IndexFunc(h.entries, test)
}

func (h *navigatorHistory) add(e *routeEntry) {
	h.entries = append(h.entries, e)
	h.NotifyListeners()
}

func (h *navigatorHistory) addAll(es []*routeEntry) {
	h.entries = append(h.entries, es...)
	if len(es) > 0 {
		h.NotifyListeners()
	}
}

func (h *navigatorHistory) insert(index int, e *routeEntry) {
	h.entries = slices.Insert(h.entries, index, e)
	h.NotifyListeners()
}

func (h *navigatorHistory) removeAt(index int) *routeEntry {
	e := h.entries[index]
	h.entries = slices.Delete(h.entries, index, index+1)
	h.NotifyListeners()
	return e
}

func (h *navigatorHistory) removeLast() *routeEntry { return h.removeAt(len(h.entries) - 1) }

func (h *navigatorHistory) String() string {
	parts := make([]string, len(h.entries))
	for i, e := range h.entries {
		parts[i] = e.route.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// HeroControllerScope provides the HeroController that the navigators below it
// use, unless a nearer scope hides it.
//
// Flutter: widgets/navigator.dart HeroControllerScope
type HeroControllerScope struct {
	InheritedWidget
	controller *HeroController
}

// NewHeroControllerScope creates a scope with the controller.
//
// Flutter: widgets/navigator.dart HeroControllerScope.new
func NewHeroControllerScope(controller *HeroController, child Widget, opts ...WidgetOpts) *HeroControllerScope {
	return &HeroControllerScope{NewInheritedWidgetBase(child, opts...), controller}
}

// NewHeroControllerScopeNone creates a scope that tells the navigators below it
// not to use a HeroController.
//
// Flutter: widgets/navigator.dart HeroControllerScope.none
func NewHeroControllerScopeNone(child Widget, opts ...WidgetOpts) *HeroControllerScope {
	return &HeroControllerScope{NewInheritedWidgetBase(child, opts...), nil}
}

// Controller is the controller of the scope, or nil.
//
// Flutter: widgets/navigator.dart HeroControllerScope.controller
func (w *HeroControllerScope) Controller() *HeroController { return w.controller }

// UpdateShouldNotify is whether the controller changed.
//
// Flutter: widgets/navigator.dart HeroControllerScope.updateShouldNotify
func (w *HeroControllerScope) UpdateShouldNotify(old Widget) bool {
	return old.(*HeroControllerScope).controller != w.controller
}

// HeroControllerScopeMaybeOf returns the controller of the closest scope above
// the context, or nil.
//
// Flutter: widgets/navigator.dart HeroControllerScope.maybeOf
func HeroControllerScopeMaybeOf(bc BuildContext) *HeroController {
	host, ok := DependOn[*HeroControllerScope](bc)
	if !ok {
		return nil
	}
	return host.controller
}

// HeroControllerScopeOf returns the controller of the closest scope above the
// context. In debug builds it panics with the tag HeroControllerScopeMissing
// when there is none.
//
// Flutter: widgets/navigator.dart HeroControllerScope.of
func HeroControllerScopeOf(bc BuildContext) *HeroController {
	controller := HeroControllerScopeMaybeOf(bc)
	if foundation.DebugMode && controller == nil {
		panic(errors.New("HeroControllerScopeMissing",
			"HeroControllerScope.of() was called with a context that does not contain a HeroControllerScope widget",
			"context", describeContext(bc)))
	}
	return controller
}

// NavigatorOpts are the optional parameters of NewNavigator.
//
// Flutter: widgets/navigator.dart Navigator.new
type NavigatorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// InitialRoute is the name of the first route to show; empty means "/".
	InitialRoute string
	// OnGenerateInitialRoutes creates the routes to start with; the default is
	// NavigatorDefaultGenerateInitialRoutes.
	OnGenerateInitialRoutes RouteListFactory
	// OnGenerateRoute creates a route for the settings (a name and arguments)
	// that PushNamed and friends were called with.
	OnGenerateRoute RouteFactory
	// OnUnknownRoute is used when OnGenerateRoute returns nil.
	OnUnknownRoute RouteFactory
	// ClipBehavior is how the overlay of the navigator clips; the default is
	// ClipHardEdge.
	ClipBehavior ui.Clip
	// Observers are told about the changes of the history.
	Observers []NavigatorObserver
	// RequestFocus is whether the routes request the focus when they become the
	// top one, unless the route says otherwise; the default is true.
	RequestFocus foundation.Opt[bool]
}

// Navigator manages a stack of Route objects and offers methods for managing
// the stack. The Navigator keeps the routes in an Overlay.
//
// Flutter: widgets/navigator.dart Navigator
type Navigator struct {
	StatefulWidget
	o NavigatorOpts
}

// NewNavigator creates a navigator.
//
// Flutter: widgets/navigator.dart Navigator.new
func NewNavigator(opts ...NavigatorOpts) *Navigator {
	o := firstOpts(opts)
	if o.OnGenerateInitialRoutes == nil {
		o.OnGenerateInitialRoutes = NavigatorDefaultGenerateInitialRoutes
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &Navigator{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), o}
}

// InitialRoute is the name of the first route, or empty.
//
// Flutter: widgets/navigator.dart Navigator.initialRoute
func (w *Navigator) InitialRoute() string { return w.o.InitialRoute }

// OnGenerateRoute creates the routes of the named pushes, or nil.
//
// Flutter: widgets/navigator.dart Navigator.onGenerateRoute
func (w *Navigator) OnGenerateRoute() RouteFactory { return w.o.OnGenerateRoute }

// OnUnknownRoute creates the route when OnGenerateRoute returns nil, or is nil.
//
// Flutter: widgets/navigator.dart Navigator.onUnknownRoute
func (w *Navigator) OnUnknownRoute() RouteFactory { return w.o.OnUnknownRoute }

// Observers are the observers of the navigator.
//
// Flutter: widgets/navigator.dart Navigator.observers
func (w *Navigator) Observers() []NavigatorObserver { return w.o.Observers }

// ClipBehavior is how the overlay of the navigator clips.
//
// Flutter: widgets/navigator.dart Navigator.clipBehavior
func (w *Navigator) ClipBehavior() ui.Clip { return w.o.ClipBehavior }

// RequestFocus is whether the routes request the focus by default.
//
// Flutter: widgets/navigator.dart Navigator.requestFocus
func (w *Navigator) RequestFocus() bool { return w.o.RequestFocus.OrElse(true) }

// OnGenerateInitialRoutes creates the routes to start with.
//
// Flutter: widgets/navigator.dart Navigator.onGenerateInitialRoutes
func (w *Navigator) OnGenerateInitialRoutes() RouteListFactory { return w.o.OnGenerateInitialRoutes }

// CreateState creates the state.
//
// Flutter: widgets/navigator.dart Navigator.createState
func (w *Navigator) CreateState() State { return &NavigatorState{} }

func optionalResult(result []any) any {
	if len(result) == 0 {
		return nil
	}
	return result[0]
}

// NavigatorOfOpts are the optional parameters of NavigatorOf and NavigatorMaybeOf.
//
// Flutter: widgets/navigator.dart Navigator.of
type NavigatorOfOpts struct {
	// RootNavigator makes the lookup return the root navigator: the farthest
	// one above the context.
	RootNavigator bool
}

func findNavigator(bc BuildContext, opts []NavigatorOfOpts) *NavigatorState {
	o := firstOpts(opts)
	var navigator *NavigatorState
	if se, ok := bc.(*StatefulElement); ok {
		if ns, isNav := se.state.(*NavigatorState); isNav {
			navigator = ns
		}
	}
	if o.RootNavigator {
		if root, ok := FindRootAncestorStateOfType[*NavigatorState](bc); ok {
			return root
		}
		return navigator
	}
	if navigator != nil {
		return navigator
	}
	if found, ok := FindAncestorStateOfType[*NavigatorState](bc); ok {
		return found
	}
	return nil
}

// NavigatorOf returns the state of the closest navigator that contains the
// context, or the root one with RootNavigator. In debug builds it panics with
// the tag NavigatorMissing when there is none.
//
// Flutter: widgets/navigator.dart Navigator.of
func NavigatorOf(bc BuildContext, opts ...NavigatorOfOpts) *NavigatorState {
	navigator := findNavigator(bc, opts)
	if foundation.DebugMode && navigator == nil {
		panic(errors.New("NavigatorMissing",
			"Navigator operation requested with a context that does not include a Navigator; the context used to push or pop routes from the Navigator must be that of a widget that is a descendant of a Navigator widget",
			"context", describeContext(bc)))
	}
	return navigator
}

// NavigatorMaybeOf returns the state of the closest navigator that contains the
// context, or nil.
//
// Flutter: widgets/navigator.dart Navigator.maybeOf
func NavigatorMaybeOf(bc BuildContext, opts ...NavigatorOfOpts) *NavigatorState {
	return findNavigator(bc, opts)
}

// NavigatorCanPop is whether the navigator of the context can pop.
//
// Flutter: widgets/navigator.dart Navigator.canPop
func NavigatorCanPop(bc BuildContext) bool {
	navigator := NavigatorMaybeOf(bc)
	return navigator != nil && navigator.CanPop()
}

// NavigatorPush pushes the route onto the navigator of the context; the channel
// gets the result of the route when it is popped.
//
// Flutter: widgets/navigator.dart Navigator.push
func NavigatorPush(bc BuildContext, route Route) <-chan any {
	return NavigatorOf(bc).Push(route)
}

// NavigatorPushNamed pushes the route that has the name onto the navigator of
// the context.
//
// Flutter: widgets/navigator.dart Navigator.pushNamed
func NavigatorPushNamed(bc BuildContext, routeName string, opts ...NavigatorPushNamedOpts) <-chan any {
	return NavigatorOf(bc).PushNamed(routeName, opts...)
}

// NavigatorPushReplacement replaces the current route of the navigator of the
// context by pushing the new one.
//
// Flutter: widgets/navigator.dart Navigator.pushReplacement
func NavigatorPushReplacement(bc BuildContext, newRoute Route, opts ...NavigatorPushReplacementOpts) <-chan any {
	return NavigatorOf(bc).PushReplacement(newRoute, opts...)
}

// NavigatorPushReplacementNamed replaces the current route by the named one.
//
// Flutter: widgets/navigator.dart Navigator.pushReplacementNamed
func NavigatorPushReplacementNamed(bc BuildContext, routeName string, opts ...NavigatorPushNamedOpts) <-chan any {
	return NavigatorOf(bc).PushReplacementNamed(routeName, opts...)
}

// NavigatorPopAndPushNamed pops the current route and pushes the named one.
//
// Flutter: widgets/navigator.dart Navigator.popAndPushNamed
func NavigatorPopAndPushNamed(bc BuildContext, routeName string, opts ...NavigatorPushNamedOpts) <-chan any {
	return NavigatorOf(bc).PopAndPushNamed(routeName, opts...)
}

// NavigatorPushNamedAndRemoveUntil pushes the named route and removes the
// routes below it until the predicate is true.
//
// Flutter: widgets/navigator.dart Navigator.pushNamedAndRemoveUntil
func NavigatorPushNamedAndRemoveUntil(bc BuildContext, newRouteName string, predicate RoutePredicate, opts ...NavigatorPushNamedOpts) <-chan any {
	return NavigatorOf(bc).PushNamedAndRemoveUntil(newRouteName, predicate, opts...)
}

// NavigatorPushAndRemoveUntil pushes the route and removes the routes below it
// until the predicate is true.
//
// Flutter: widgets/navigator.dart Navigator.pushAndRemoveUntil
func NavigatorPushAndRemoveUntil(bc BuildContext, newRoute Route, predicate RoutePredicate) <-chan any {
	return NavigatorOf(bc).PushAndRemoveUntil(newRoute, predicate)
}

// NavigatorReplace replaces the old route by the new one.
//
// Flutter: widgets/navigator.dart Navigator.replace
func NavigatorReplace(bc BuildContext, oldRoute, newRoute Route) {
	NavigatorOf(bc).Replace(oldRoute, newRoute)
}

// NavigatorReplaceRouteBelow replaces the route below the anchor by the new one.
//
// Flutter: widgets/navigator.dart Navigator.replaceRouteBelow
func NavigatorReplaceRouteBelow(bc BuildContext, anchorRoute, newRoute Route) {
	NavigatorOf(bc).ReplaceRouteBelow(anchorRoute, newRoute)
}

// NavigatorMaybePop pops the top route of the navigator of the context, unless
// the route does not want to be popped; it reports whether the pop was handled.
//
// Flutter: widgets/navigator.dart Navigator.maybePop
func NavigatorMaybePop(bc BuildContext, result ...any) bool {
	return NavigatorOf(bc).MaybePop(result...)
}

// NavigatorPop pops the top route of the navigator of the context; the optional
// result is passed to the channel that Push returned.
//
// Flutter: widgets/navigator.dart Navigator.pop
func NavigatorPop(bc BuildContext, result ...any) {
	NavigatorOf(bc).Pop(result...)
}

// NavigatorPopUntil pops routes until the predicate is true.
//
// Flutter: widgets/navigator.dart Navigator.popUntil
func NavigatorPopUntil(bc BuildContext, predicate RoutePredicate) {
	NavigatorOf(bc).PopUntil(predicate)
}

// NavigatorPopUntilWithResult pops routes until the predicate is true, and the
// last one popped gets the result.
//
// Flutter: widgets/navigator.dart Navigator.popUntilWithResult
func NavigatorPopUntilWithResult(bc BuildContext, predicate RoutePredicate, result any) {
	NavigatorOf(bc).PopUntilWithResult(predicate, result)
}

// NavigatorRemoveRoute removes the route without animating.
//
// Flutter: widgets/navigator.dart Navigator.removeRoute
func NavigatorRemoveRoute(bc BuildContext, route Route, result ...any) {
	NavigatorOf(bc).RemoveRoute(route, result...)
}

// NavigatorRemoveRouteBelow removes the route below the anchor without
// animating.
//
// Flutter: widgets/navigator.dart Navigator.removeRouteBelow
func NavigatorRemoveRouteBelow(bc BuildContext, anchorRoute Route, result ...any) {
	NavigatorOf(bc).RemoveRouteBelow(anchorRoute, result...)
}

// NavigatorDefaultGenerateInitialRoutes is the default OnGenerateInitialRoutes:
// for the initial route "/a/b" it creates the routes "/", "/a" and "/a/b" (the
// ones that exist), and for the default route just that one.
//
// Flutter: widgets/navigator.dart Navigator.defaultGenerateInitialRoutes
func NavigatorDefaultGenerateInitialRoutes(navigator *NavigatorState, initialRouteName string) []Route {
	var result []Route
	if strings.HasPrefix(initialRouteName, "/") && len(initialRouteName) > 1 {
		initialRouteName = initialRouteName[1:] // strip the leading '/'
		result = append(result, navigator.routeNamed(NavigatorDefaultRouteName, nil, true))
		routeParts := strings.Split(initialRouteName, "/")
		if initialRouteName != "" {
			routeName := ""
			for _, part := range routeParts {
				routeName += "/" + part
				result = append(result, navigator.routeNamed(routeName, nil, true))
			}
		}
		if result[len(result)-1] == nil {
			navigator.reportError(errors.New("NavigatorInitialRouteUnknown",
				"could not navigate to initial route; there was no corresponding route in the app, and therefore the initial route specified will be ignored and \"/\" will be used instead",
				"route", "/"+initialRouteName))
			for _, route := range result {
				if route != nil {
					route.Dispose()
				}
			}
			result = nil
		}
	} else if initialRouteName != NavigatorDefaultRouteName {
		// If the initial route name isn't "/", it is fetched with allowNull so
		// that "/" is the fallback when that fails.
		result = append(result, navigator.routeNamed(initialRouteName, nil, true))
	}
	// A nil route may be the result of a gap in the initial route name: with the
	// routes "A" and "A/B/C" and the initial route "A/B/C", the result is
	// ["A", nil, "A/B/C"], and the nil is filtered out.
	result = slices.DeleteFunc(result, func(r Route) bool { return r == nil })
	if len(result) == 0 {
		result = append(result, navigator.routeNamed(NavigatorDefaultRouteName, nil, false))
	}
	return result
}

// NavigatorPushNamedOpts are the optional parameters of the named pushes.
//
// Flutter: widgets/navigator.dart NavigatorState.pushNamed
type NavigatorPushNamedOpts struct {
	// Arguments are passed to OnGenerateRoute in the settings of the route.
	Arguments any
	// Result is the result of the route that is replaced or popped by the call.
	Result any
}

// NavigatorPushReplacementOpts are the optional parameters of PushReplacement.
//
// Flutter: widgets/navigator.dart NavigatorState.pushReplacement
type NavigatorPushReplacementOpts struct {
	// Result is the result of the route that is replaced.
	Result any
}

// NavigatorState is the state of a Navigator: it holds the history of routes,
// the overlay they live in, and the methods that change the history.
//
// Flutter: widgets/navigator.dart NavigatorState
type NavigatorState struct {
	StateOf[*Navigator]
	TickerProvider

	overlayKey                     *GlobalKey[*OverlayState]
	history                        navigatorHistory
	entryWaitingForSubTreeDisposal []*routeEntry
	observedRouteAdditions         []*navigatorObservation
	observedRouteDeletions         []*navigatorObservation
	debugLocked                    bool // prevents re-entrant calls to push, pop and friends
	heroControllerFromScope        *HeroController
	effectiveObservers             []NavigatorObserver
	historyHandle                  foundation.ListenerHandle
	historyInitialized             bool

	lastTopmostRoute    *routeEntry
	flushingHistory     bool
	userGesturesCount   int
	userGestureNotifier *foundation.ValueNotifier[bool]
	activePointers      map[int]bool
}

func (s *NavigatorState) reportError(err error) {
	if binding := MaybeWidgetsBindingOf(s.Context()); binding != nil {
		if report := binding.ErrorReporter(); report != nil {
			report(err)
		}
	}
}

func (s *NavigatorState) debugRequireLocked() {
	if foundation.DebugMode && !s.debugLocked {
		panic(errors.New("NavigatorNotLocked", "the history of the navigator was changed without locking it"))
	}
}

func (s *NavigatorState) debugRequireUnlocked() {
	if foundation.DebugMode && s.debugLocked {
		panic(errors.New("NavigatorLocked",
			"the navigator was asked to change its history while it was changing it; do not push or pop from inside the methods of a route that run while the history changes",
			"navigator", s.String()))
	}
}

// UserGestureInProgressNotifier tells its listeners when a user gesture that may
// pop the route starts or stops.
//
// Flutter: widgets/navigator.dart NavigatorState.userGestureInProgressNotifier
func (s *NavigatorState) UserGestureInProgressNotifier() *foundation.ValueNotifier[bool] {
	if s.userGestureNotifier == nil {
		s.userGestureNotifier = foundation.NewValueNotifier(false)
	}
	return s.userGestureNotifier
}

// UserGestureInProgress is whether a user gesture that may pop the route is
// going on.
//
// Flutter: widgets/navigator.dart NavigatorState.userGestureInProgress
func (s *NavigatorState) UserGestureInProgress() bool {
	return s.UserGestureInProgressNotifier().Value()
}

func (s *NavigatorState) handleHistoryChanged() {
	// Avoid dispatching a notification in the middle of a build.
	sched := WidgetsBindingOf(s.Context()).Scheduler()
	switch sched.SchedulerPhase() {
	case scheduler.SchedulerPhasePostFrameCallbacks:
		DispatchNotification(s.Context(), NavigationNotification{CanHandlePop: s.getNavigatorCanHandlePop()})
	default:
		sched.AddPostFrameCallback(func(_ time.Duration) {
			if !s.Mounted() {
				return
			}
			DispatchNotification(s.Context(), NavigationNotification{CanHandlePop: s.getNavigatorCanHandlePop()})
		})
	}
}

func (s *NavigatorState) getNavigatorCanHandlePop() bool {
	if s.CanPop() {
		return true
	}
	last := s.lastRouteEntryWhere(routeEntryIsPresent)
	return last != nil && last.route.PopDisposition() == RoutePopDispositionDoNotPop
}

// InitState subscribes to the observers and the hero controller.
//
// Flutter: widgets/navigator.dart NavigatorState.initState
func (s *NavigatorState) InitState() {
	s.activePointers = map[int]bool{}
	for _, observer := range s.Widget().o.Observers {
		if foundation.DebugMode && observer.Navigator() != nil {
			panic(errors.New("NavigatorObserverShared", "a NavigatorObserver can not be shared by two navigators",
				"observer", foundation.DescribeIdentity(observer)))
		}
		observer.setNavigator(s)
	}
	s.effectiveObservers = s.Widget().o.Observers
	// The inherited widget is extracted by hand because the context is not
	// completely initialized in InitState.
	if el := GetElementForInheritedWidget[*HeroControllerScope](s.Context()); el != nil {
		s.updateHeroController(el.AsElement().widget.(*HeroControllerScope).controller)
	} else {
		s.updateHeroController(nil)
	}
	s.historyHandle = s.history.AddListener(s.handleHistoryChanged)
}

// restoreState fills the history with the initial routes. Flutter does it in
// RestorationMixin.didChangeDependencies; this port has no state restoration, so
// it runs the first time DidChangeDependencies is called.
func (s *NavigatorState) restoreState() {
	s.forcedDisposeAllRouteEntries()
	s.overlayKey = NewGlobalKey[*OverlayState]()
	initialRoute := s.Widget().o.InitialRoute
	if initialRoute == "" {
		initialRoute = NavigatorDefaultRouteName
	}
	routes := s.Widget().o.OnGenerateInitialRoutes(s, initialRoute)
	entries := make([]*routeEntry, len(routes))
	for i, route := range routes {
		entries[i] = newRouteEntry(route, routeAdd, false)
	}
	s.history.addAll(entries)
	if foundation.DebugMode && len(s.history.entries) == 0 {
		panic(errors.New("NavigatorHistoryEmpty", "the Navigator needs at least one initial route; OnGenerateInitialRoutes returned none"))
	}
	s.debugRequireUnlocked()
	if foundation.DebugMode {
		s.debugLocked = true
	}
	s.flushHistoryUpdates(true)
	if foundation.DebugMode {
		s.debugLocked = false
	}
}

// DidChangeDependencies creates the initial history the first time, then
// updates the hero controller and tells the routes.
//
// Flutter: widgets/navigator.dart NavigatorState.didChangeDependencies
func (s *NavigatorState) DidChangeDependencies() {
	if !s.historyInitialized {
		s.historyInitialized = true
		s.restoreState()
	}
	s.updateHeroController(HeroControllerScopeMaybeOf(s.Context()))
	for _, entry := range slices.Clone(s.history.entries) {
		if entry.route.Navigator() == s {
			entry.route.ChangedExternalState()
		}
	}
}

func (s *NavigatorState) forcedDisposeAllRouteEntries() {
	waiting := s.entryWaitingForSubTreeDisposal
	s.entryWaitingForSubTreeDisposal = nil
	for _, entry := range waiting {
		entry.forcedDispose()
	}
	for len(s.history.entries) > 0 {
		disposeRouteEntry(s.history.removeLast(), false)
	}
}

func (s *NavigatorState) removeEntryWaitingForSubTreeDisposal(entry *routeEntry) bool {
	i := slices.Index(s.entryWaitingForSubTreeDisposal, entry)
	if i < 0 {
		return false
	}
	s.entryWaitingForSubTreeDisposal = slices.Delete(s.entryWaitingForSubTreeDisposal, i, i+1)
	return true
}

func disposeRouteEntry(entry *routeEntry, graceful bool) {
	for _, overlayEntry := range entry.route.OverlayEntries() {
		overlayEntry.Remove()
	}
	if graceful {
		entry.dispose()
	} else {
		entry.forcedDispose()
	}
}

func (s *NavigatorState) updateHeroController(newHeroController *HeroController) {
	if s.heroControllerFromScope == newHeroController {
		return
	}
	if newHeroController != nil {
		// Makes sure the same hero controller is not shared between two navigators.
		if foundation.DebugMode && newHeroController.Navigator() != nil {
			// The hero controller may subscribe to an existing navigator; that is
			// fine as long as that navigator gives it up before the end of the
			// frame.
			previousOwner := newHeroController.Navigator()
			WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
				// Only checks if this navigator still owns the hero controller.
				if s.heroControllerFromScope == newHeroController {
					hasOwnership := s.heroControllerFromScope.Navigator() == s
					if !hasOwnership || previousOwner.heroControllerFromScope == newHeroController {
						otherOwner := previousOwner
						if !hasOwnership {
							otherOwner = s.heroControllerFromScope.Navigator()
						}
						s.reportError(errors.New("HeroControllerShared",
							"a HeroController can not be shared by multiple Navigators; create a HeroControllerScope for each Navigator or use a HeroControllerScope.none to prevent the subtree from receiving a HeroController",
							"navigators", s.String()+", "+otherOwner.String()))
					}
				}
			})
		}
		newHeroController.setNavigator(s)
	}
	// Only unsubscribes the hero controller when it is currently subscribed to
	// this navigator.
	if s.heroControllerFromScope != nil && s.heroControllerFromScope.Navigator() == s {
		s.heroControllerFromScope.setNavigator(nil)
	}
	s.heroControllerFromScope = newHeroController
	s.updateEffectiveObservers()
}

func (s *NavigatorState) updateEffectiveObservers() {
	if s.heroControllerFromScope != nil {
		s.effectiveObservers = append(slices.Clone(s.Widget().o.Observers), s.heroControllerFromScope.AsObserver())
	} else {
		s.effectiveObservers = s.Widget().o.Observers
	}
}

// DidUpdateWidget moves the observers when they changed and tells the routes.
//
// Flutter: widgets/navigator.dart NavigatorState.didUpdateWidget
func (s *NavigatorState) DidUpdateWidget(old *Navigator) {
	if !slices.Equal(old.o.Observers, s.Widget().o.Observers) {
		for _, observer := range old.o.Observers {
			observer.setNavigator(nil)
		}
		for _, observer := range s.Widget().o.Observers {
			if foundation.DebugMode && observer.Navigator() != nil {
				panic(errors.New("NavigatorObserverShared", "a NavigatorObserver can not be shared by two navigators",
					"observer", foundation.DescribeIdentity(observer)))
			}
			observer.setNavigator(s)
		}
		s.updateEffectiveObservers()
	}
	for _, entry := range slices.Clone(s.history.entries) {
		if entry.route.Navigator() == s {
			entry.route.ChangedExternalState()
		}
	}
}

// Deactivate lets the observers go.
//
// Flutter: widgets/navigator.dart NavigatorState.deactivate
func (s *NavigatorState) Deactivate() {
	for _, observer := range s.effectiveObservers {
		observer.setNavigator(nil)
	}
	s.effectiveObservers = nil
}

// Activate takes the observers again.
//
// Flutter: widgets/navigator.dart NavigatorState.activate
func (s *NavigatorState) Activate() {
	s.updateEffectiveObservers()
	for _, observer := range s.effectiveObservers {
		if foundation.DebugMode && observer.Navigator() != nil {
			panic(errors.New("NavigatorObserverShared", "a NavigatorObserver can not be shared by two navigators",
				"observer", foundation.DescribeIdentity(observer)))
		}
		observer.setNavigator(s)
	}
}

// Dispose disposes every route.
//
// Flutter: widgets/navigator.dart NavigatorState.dispose
func (s *NavigatorState) Dispose() {
	s.debugRequireUnlocked()
	if foundation.DebugMode {
		s.debugLocked = true
	}
	s.updateHeroController(nil)
	s.forcedDisposeAllRouteEntries()
	if s.userGestureNotifier != nil {
		s.userGestureNotifier.Dispose()
	}
	s.history.RemoveListener(s.historyHandle)
	s.history.Dispose()
	// The navigator stays locked, so that the object becomes unusable.
}

// Overlay is the overlay of the navigator, or nil before it is built.
//
// Flutter: widgets/navigator.dart NavigatorState.overlay
func (s *NavigatorState) Overlay() *OverlayState {
	if s.overlayKey == nil {
		return nil
	}
	state, _ := s.overlayKey.CurrentState()
	return state
}

func (s *NavigatorState) allRouteOverlayEntries() []*OverlayEntry {
	var entries []*OverlayEntry
	for _, entry := range s.history.entries {
		entries = append(entries, entry.route.OverlayEntries()...)
	}
	return entries
}

func (s *NavigatorState) flushHistoryUpdates(rearrangeOverlay bool) {
	s.debugRequireLocked()
	s.flushingHistory = true
	// Cleans up the list, sending updates to the routes that changed. The
	// didChangePrevious and didChangeNext updates are not sent to the routes that
	// did not change at this point, because it is not yet certain what the routes
	// will be at the end (some might get disposed).
	index := len(s.history.entries) - 1
	var next *routeEntry
	var entry, previous *routeEntry
	if index >= 0 {
		entry = s.history.entries[index]
	}
	if index > 0 {
		previous = s.history.entries[index-1]
	}
	canRemoveOrAdd := false // whether there is a fully opaque route on top to silently remove or add routes underneath
	var poppedRoute Route   // the route that should trigger didPopNext on the top active route
	seenTopActiveRoute := false
	var toBeDisposed []*routeEntry
	for index >= 0 {
		switch entry.currentState {
		case routeAdd:
			entry.handleAdd(s, s.getRouteBefore(index-1, routeEntryIsPresent).routeOrNil())
			continue
		case routeAdding:
			if canRemoveOrAdd || next == nil {
				entry.didAdd(s, next == nil)
				continue
			}
		case routePush, routePushReplace, routeReplace:
			entry.handlePush(s, next == nil, previous.routeOrNil(), s.getRouteBefore(index-1, routeEntryIsPresent).routeOrNil())
			if entry.currentState == routeIdle {
				continue
			}
		case routePushing: // exits this state when the animation completes
			if !seenTopActiveRoute && poppedRoute != nil {
				entry.handleDidPopNext(poppedRoute)
			}
			seenTopActiveRoute = true
		case routeIdle:
			if !seenTopActiveRoute && poppedRoute != nil {
				entry.handleDidPopNext(poppedRoute)
			}
			seenTopActiveRoute = true
			// This route is idle, so the subsequent (earlier) routes that wait to
			// be removed can be removed silently.
			canRemoveOrAdd = true
		case routePop:
			if !entry.handlePop(s, s.getRouteBefore(index, routeEntryWillBePresent).routeOrNil()) {
				continue
			}
			if !seenTopActiveRoute {
				if poppedRoute != nil {
					entry.handleDidPopNext(poppedRoute)
				}
				poppedRoute = entry.route
			}
			s.observedRouteDeletions = append(s.observedRouteDeletions, &navigatorObservation{
				kind: observationPop, primary: entry.route, secondary: s.getRouteBefore(index, routeEntryWillBePresent).routeOrNil(),
			})
			if entry.currentState == routeDispose {
				// The pop finished synchronously; this happens if the transition
				// duration is zero.
				continue
			}
			canRemoveOrAdd = true
		case routePopping:
			// exits this state when the animation completes
		case routeComplete:
			entry.handleComplete()
			continue
		case routeRemove:
			// Routes that are not installed yet exit as if they had never been on
			// the navigator: they do not announce their presence.
			if !seenTopActiveRoute && entry.base().installed() {
				if poppedRoute != nil {
					entry.handleDidPopNext(poppedRoute)
				}
				poppedRoute = nil
			}
			entry.handleRemoval(s, s.getRouteBefore(index, routeEntryWillBePresent).routeOrNil())
			continue
		case routeRemoving:
			if !canRemoveOrAdd && next != nil {
				// This route is not allowed to be removed yet.
				break
			}
			entry.currentState = routeDispose
			continue
		case routeDispose:
			// Delays the disposal until didChangeNext and didChangePrevious were sent.
			toBeDisposed = append(toBeDisposed, s.history.removeAt(index))
			entry = next
		case routeDisposing, routeDisposed, routeStaging:
			if foundation.DebugMode {
				panic(errors.New("RouteEntryStateInvalid", "a route entry in the history is in a state that cannot be flushed",
					"state", int(entry.currentState)))
			}
		}
		index--
		next = entry
		entry = previous
		if index > 0 {
			previous = s.history.entries[index-1]
		} else {
			previous = nil
		}
	}
	// Informs the navigator observers about the route changes.
	s.flushObserverNotifications()
	// Now that the list is clean, sends the didChangeNext and didChangePrevious
	// notifications.
	s.flushRouteAnnouncement()
	lastEntry := s.lastRouteEntryWhere(routeEntryIsPresent)
	if lastEntry != nil && s.lastTopmostRoute != lastEntry {
		var lastTop Route
		if s.lastTopmostRoute != nil {
			lastTop = s.lastTopmostRoute.route
		}
		for _, observer := range slices.Clone(s.effectiveObservers) {
			observer.DidChangeTop(lastEntry.route, lastTop)
		}
	}
	s.lastTopmostRoute = lastEntry
	// Lastly, removes the overlay entries of all the marked entries and disposes
	// them.
	for _, e := range toBeDisposed {
		disposeRouteEntry(e, true)
	}
	if rearrangeOverlay {
		if overlay := s.Overlay(); overlay != nil {
			overlay.Rearrange(s.allRouteOverlayEntries())
		}
	}
	s.flushingHistory = false
}

func (e *routeEntry) routeOrNil() Route {
	if e == nil {
		return nil
	}
	return e.route
}

func (s *NavigatorState) flushObserverNotifications() {
	if len(s.effectiveObservers) == 0 {
		s.observedRouteDeletions = nil
		s.observedRouteAdditions = nil
		return
	}
	for len(s.observedRouteAdditions) > 0 {
		last := len(s.observedRouteAdditions) - 1
		observation := s.observedRouteAdditions[last]
		s.observedRouteAdditions = s.observedRouteAdditions[:last]
		for _, observer := range slices.Clone(s.effectiveObservers) {
			observation.notify(observer)
		}
	}
	for len(s.observedRouteDeletions) > 0 {
		observation := s.observedRouteDeletions[0]
		s.observedRouteDeletions = s.observedRouteDeletions[1:]
		for _, observer := range slices.Clone(s.effectiveObservers) {
			observation.notify(observer)
		}
	}
}

func (s *NavigatorState) flushRouteAnnouncement() {
	index := len(s.history.entries) - 1
	for index >= 0 {
		entry := s.history.entries[index]
		if !entry.suitableForAnnouncement() {
			index--
			continue
		}
		next := s.getRouteAfter(index+1, routeEntrySuitableForTransition)
		nextRoute := next.routeOrNil()
		if (announced(nextRoute)) != entry.lastAnnouncedNextRoute {
			if entry.shouldAnnounceChangeToNext(nextRoute) {
				entry.route.DidChangeNext(nextRoute)
			}
			entry.lastAnnouncedNextRoute = announced(nextRoute)
		}
		previous := s.getRouteBefore(index-1, routeEntrySuitableForTransition)
		previousRoute := previous.routeOrNil()
		if announced(previousRoute) != entry.lastAnnouncedPreviousRoute {
			entry.route.DidChangePrevious(previousRoute)
			entry.lastAnnouncedPreviousRoute = announced(previousRoute)
		}
		index--
	}
}

func (s *NavigatorState) getRouteBefore(index int, predicate func(*routeEntry) bool) *routeEntry {
	index = s.getIndexBefore(index, predicate)
	if index >= 0 {
		return s.history.entries[index]
	}
	return nil
}

func (s *NavigatorState) getIndexBefore(index int, predicate func(*routeEntry) bool) int {
	for index >= 0 && !predicate(s.history.entries[index]) {
		index--
	}
	return index
}

func (s *NavigatorState) getRouteAfter(index int, predicate func(*routeEntry) bool) *routeEntry {
	for index < len(s.history.entries) && !predicate(s.history.entries[index]) {
		index++
	}
	if index < len(s.history.entries) {
		return s.history.entries[index]
	}
	return nil
}

func (s *NavigatorState) firstRouteEntryWhere(test func(*routeEntry) bool) *routeEntry {
	for _, e := range s.history.entries {
		if test(e) {
			return e
		}
	}
	return nil
}

func (s *NavigatorState) lastRouteEntryWhere(test func(*routeEntry) bool) *routeEntry {
	var result *routeEntry
	for _, e := range s.history.entries {
		if test(e) {
			result = e
		}
	}
	return result
}

// routeNamed creates the route for the name with OnGenerateRoute, or
// OnUnknownRoute. It returns nil only when allowNull is set.
//
// Flutter: widgets/navigator.dart NavigatorState._routeNamed
func (s *NavigatorState) routeNamed(name string, arguments any, allowNull bool) Route {
	s.debugRequireUnlocked()
	w := s.Widget()
	if allowNull && w.o.OnGenerateRoute == nil {
		return nil
	}
	if foundation.DebugMode && w.o.OnGenerateRoute == nil {
		panic(errors.New("NavigatorOnGenerateRouteMissing",
			fmt.Sprintf("Navigator.onGenerateRoute was null, but the route named %q was referenced; to use the Navigator API with named routes (pushNamed, pushReplacementNamed, or pushNamedAndRemoveUntil), the Navigator must be provided with an onGenerateRoute handler", name),
			"navigator", s.String()))
	}
	settings := &RouteSettingsBase{name, arguments}
	route := w.o.OnGenerateRoute(settings)
	if route == nil && !allowNull {
		if foundation.DebugMode && w.o.OnUnknownRoute == nil {
			panic(errors.New("NavigatorOnGenerateRouteReturnedNull",
				fmt.Sprintf("Navigator.onGenerateRoute returned null when requested to build route %q; the onGenerateRoute callback must never return null, unless an onUnknownRoute callback is provided as well", name),
				"navigator", s.String()))
		}
		route = w.o.OnUnknownRoute(settings)
		if foundation.DebugMode && route == nil {
			panic(errors.New("NavigatorOnUnknownRouteReturnedNull",
				fmt.Sprintf("Navigator.onUnknownRoute returned null when requested to build route %q; the onUnknownRoute callback must never return null", name),
				"navigator", s.String()))
		}
	}
	return route
}

// PushNamed pushes the route that OnGenerateRoute creates for the name; the
// channel gets the result of the route. In debug builds it panics with the tags
// NavigatorOnGenerateRouteMissing, NavigatorOnGenerateRouteReturnedNull and
// NavigatorOnUnknownRouteReturnedNull when the route can't be created.
//
// Flutter: widgets/navigator.dart NavigatorState.pushNamed
func (s *NavigatorState) PushNamed(routeName string, opts ...NavigatorPushNamedOpts) <-chan any {
	o := firstOpts(opts)
	return s.Push(s.routeNamed(routeName, o.Arguments, false))
}

// PushReplacementNamed replaces the current route by the named one: the new
// route is pushed, and the old one completes with the Result of the options once
// the transition is done.
//
// Flutter: widgets/navigator.dart NavigatorState.pushReplacementNamed
func (s *NavigatorState) PushReplacementNamed(routeName string, opts ...NavigatorPushNamedOpts) <-chan any {
	o := firstOpts(opts)
	return s.PushReplacement(s.routeNamed(routeName, o.Arguments, false), NavigatorPushReplacementOpts{Result: o.Result})
}

// PopAndPushNamed pops the current route (with the Result of the options) and
// pushes the named one in its place.
//
// Flutter: widgets/navigator.dart NavigatorState.popAndPushNamed
func (s *NavigatorState) PopAndPushNamed(routeName string, opts ...NavigatorPushNamedOpts) <-chan any {
	o := firstOpts(opts)
	s.Pop(o.Result)
	return s.PushNamed(routeName, NavigatorPushNamedOpts{Arguments: o.Arguments})
}

// PushNamedAndRemoveUntil pushes the named route and removes the routes below
// it until the predicate is true.
//
// Flutter: widgets/navigator.dart NavigatorState.pushNamedAndRemoveUntil
func (s *NavigatorState) PushNamedAndRemoveUntil(newRouteName string, predicate RoutePredicate, opts ...NavigatorPushNamedOpts) <-chan any {
	o := firstOpts(opts)
	return s.PushAndRemoveUntil(s.routeNamed(newRouteName, o.Arguments, false), predicate)
}

// Push adds the route to the top of the history; the channel gets the result of
// the route when it is popped.
//
// Flutter: widgets/navigator.dart NavigatorState.push
func (s *NavigatorState) Push(route Route) <-chan any {
	s.pushEntry(newRouteEntry(route, routePush, false))
	return route.Popped()
}

func (s *NavigatorState) lockHistory() {
	s.debugRequireUnlocked()
	if foundation.DebugMode {
		s.debugLocked = true
	}
}

func (s *NavigatorState) unlockHistory() {
	if foundation.DebugMode {
		s.debugLocked = false
	}
}

func (s *NavigatorState) pushEntry(entry *routeEntry) {
	s.lockHistory()
	if foundation.DebugMode && entry.base().installed() {
		panic(errors.New("RouteAlreadyInstalled", "the pushed route has already been used; when pushing a route, a new Route object must be provided",
			"route", entry.route.String()))
	}
	s.history.add(entry)
	s.flushHistoryUpdates(true)
	s.unlockHistory()
	s.afterNavigation()
}

func (s *NavigatorState) afterNavigation() {
	s.cancelActivePointers()
}

// PushReplacement replaces the current route by pushing the new one: the old
// route completes with the Result of the options once the transition is done.
//
// Flutter: widgets/navigator.dart NavigatorState.pushReplacement
func (s *NavigatorState) PushReplacement(newRoute Route, opts ...NavigatorPushReplacementOpts) <-chan any {
	o := firstOpts(opts)
	if foundation.DebugMode && newRoute.AsRoute().installed() {
		panic(errors.New("RouteAlreadyInstalled", "the pushed route has already been used; when pushing a route, a new Route object must be provided",
			"route", newRoute.String()))
	}
	s.pushReplacementEntry(newRouteEntry(newRoute, routePushReplace, false), o.Result)
	return newRoute.Popped()
}

func (s *NavigatorState) pushReplacementEntry(entry *routeEntry, result any) {
	s.lockHistory()
	if foundation.DebugMode {
		if len(s.history.entries) == 0 || s.history.indexWhere(routeEntryIsPresent) < 0 {
			panic(errors.New("NavigatorNoActiveRoutes", "Navigator has no active routes to replace"))
		}
	}
	s.lastRouteEntryWhere(routeEntryIsPresent).complete(result, true, true)
	s.history.add(entry)
	s.flushHistoryUpdates(true)
	s.unlockHistory()
	s.afterNavigation()
}

// PushAndRemoveUntil pushes the route and removes the routes below it until the
// predicate returns true for a route, which stays.
//
// Flutter: widgets/navigator.dart NavigatorState.pushAndRemoveUntil
func (s *NavigatorState) PushAndRemoveUntil(newRoute Route, predicate RoutePredicate) <-chan any {
	if foundation.DebugMode && newRoute.AsRoute().installed() {
		panic(errors.New("RouteAlreadyInstalled", "the pushed route has already been used; when pushing a route, a new Route object must be provided",
			"route", newRoute.String()))
	}
	s.pushEntryAndRemoveUntil(newRouteEntry(newRoute, routePush, false), predicate)
	return newRoute.Popped()
}

func (s *NavigatorState) pushEntryAndRemoveUntil(entry *routeEntry, predicate RoutePredicate) {
	s.lockHistory()
	index := len(s.history.entries) - 1
	s.history.add(entry)
	for index >= 0 && !predicate(s.history.entries[index].route) {
		if s.history.entries[index].isPresent() {
			s.history.entries[index].complete(nil, false, true)
		}
		index--
	}
	s.flushHistoryUpdates(true)
	s.unlockHistory()
	s.afterNavigation()
}

// Replace replaces a route of the history by a new one without animating: the
// new one goes above the old one, which completes and is removed. In debug
// builds it panics with the tags NavigatorRouteMissing and
// NavigatorRouteRemoved when the old route is not in the history or was removed.
//
// Flutter: widgets/navigator.dart NavigatorState.replace
func (s *NavigatorState) Replace(oldRoute, newRoute Route) {
	s.debugRequireUnlocked()
	if foundation.DebugMode && !oldRoute.AsRoute().isInstalledIn(s) {
		panic(errors.New("NavigatorRouteMissing", "this Navigator does not contain the specified oldRoute", "route", oldRoute.String()))
	}
	s.replaceEntry(newRouteEntry(newRoute, routeReplace, false), oldRoute)
}

func (s *NavigatorState) replaceEntry(entry *routeEntry, oldRoute Route) {
	s.debugRequireUnlocked()
	if oldRoute == entry.route {
		return
	}
	s.lockHistory()
	if foundation.DebugMode && entry.base().installed() {
		panic(errors.New("RouteAlreadyInstalled", "the new route has already been used", "route", entry.route.String()))
	}
	index := s.history.indexWhere(routeEntryIsRoute(oldRoute))
	if foundation.DebugMode {
		if index < 0 {
			panic(errors.New("NavigatorRouteMissing", "this Navigator does not contain the specified oldRoute", "route", oldRoute.String()))
		}
		if !s.history.entries[index].isPresent() {
			panic(errors.New("NavigatorRouteRemoved", "the specified oldRoute has already been removed from the Navigator", "route", oldRoute.String()))
		}
	}
	wasCurrent := oldRoute.IsCurrent()
	s.history.insert(index+1, entry)
	s.history.entries[index].complete(nil, true, true)
	s.flushHistoryUpdates(true)
	s.unlockHistory()
	if wasCurrent {
		s.afterNavigation()
	}
}

// ReplaceRouteBelow replaces the route below the anchor by a new one without
// animating.
//
// Flutter: widgets/navigator.dart NavigatorState.replaceRouteBelow
func (s *NavigatorState) ReplaceRouteBelow(anchorRoute, newRoute Route) {
	if foundation.DebugMode {
		if newRoute.AsRoute().installed() {
			panic(errors.New("RouteAlreadyInstalled", "the new route has already been used", "route", newRoute.String()))
		}
		if !anchorRoute.AsRoute().isInstalledIn(s) {
			panic(errors.New("NavigatorRouteMissing", "this Navigator does not contain the specified anchorRoute", "route", anchorRoute.String()))
		}
	}
	s.replaceEntryBelow(newRouteEntry(newRoute, routeReplace, false), anchorRoute)
}

func (s *NavigatorState) indexBelowAnchor(anchorRoute Route) int {
	anchorIndex := s.history.indexWhere(routeEntryIsRoute(anchorRoute))
	if foundation.DebugMode {
		if anchorIndex < 0 {
			panic(errors.New("NavigatorRouteMissing", "this Navigator does not contain the specified anchorRoute", "route", anchorRoute.String()))
		}
		if !s.history.entries[anchorIndex].isPresent() {
			panic(errors.New("NavigatorRouteRemoved", "the specified anchorRoute has already been removed from the Navigator", "route", anchorRoute.String()))
		}
	}
	index := anchorIndex - 1
	for index >= 0 {
		if s.history.entries[index].isPresent() {
			break
		}
		index--
	}
	if foundation.DebugMode && index < 0 {
		panic(errors.New("NavigatorRouteBelowMissing", "there are no routes below the specified anchorRoute", "route", anchorRoute.String()))
	}
	return index
}

func (s *NavigatorState) replaceEntryBelow(entry *routeEntry, anchorRoute Route) {
	s.lockHistory()
	index := s.indexBelowAnchor(anchorRoute)
	s.history.insert(index+1, entry)
	s.history.entries[index].complete(nil, true, true)
	s.flushHistoryUpdates(true)
	s.unlockHistory()
}

// CanPop is whether the navigator has more than one active route, or the first
// one handles pops itself.
//
// Flutter: widgets/navigator.dart NavigatorState.canPop
func (s *NavigatorState) CanPop() bool {
	var present []*routeEntry
	for _, e := range s.history.entries {
		if e.isPresent() {
			present = append(present, e)
			if len(present) == 2 {
				break
			}
		}
	}
	if len(present) == 0 {
		// There are no active routes, so there is nothing to pop.
		return false
	}
	if present[0].route.WillHandlePopInternally() {
		// The first route can handle pops itself, so it can pop.
		return true
	}
	// With only one route there is nothing to pop; with two there is.
	return len(present) > 1
}

// MaybePop pops the top route unless it wants to stay (a PopScope that blocks
// pops, for example). It returns false when the route bubbles the pop up to the
// parent, and true when the pop was handled, even if the route stayed.
//
// Deviation: Flutter's maybePop is asynchronous because of the deprecated
// Route.willPop, which Glutter does not port (SPEC §12.10).
//
// Flutter: widgets/navigator.dart NavigatorState.maybePop
func (s *NavigatorState) MaybePop(result ...any) bool {
	lastEntry := s.lastRouteEntryWhere(routeEntryIsPresent)
	if lastEntry == nil {
		return false
	}
	switch lastEntry.route.PopDisposition() {
	case RoutePopDispositionBubble:
		return false
	case RoutePopDispositionPop:
		s.Pop(result...)
		return true
	case RoutePopDispositionDoNotPop:
		lastEntry.route.OnPopInvokedWithResult(false, optionalResult(result))
		return true
	}
	return true
}

// Pop removes the top route of the history; the result is passed to the channel
// that Push returned. The route may keep animating out.
//
// Flutter: widgets/navigator.dart NavigatorState.pop
func (s *NavigatorState) Pop(result ...any) {
	s.lockHistory()
	entry := s.lastRouteEntryWhere(routeEntryIsPresent)
	if entry == nil {
		s.unlockHistory()
		if foundation.DebugMode {
			panic(errors.New("NavigatorNoActiveRoutes", "Navigator has no active routes to pop"))
		}
		return
	}
	// TODO(step 10.3): the Pages API asks onPopPage here for page-based routes.
	entry.pop(optionalResult(result), true)
	if entry.currentState == routePop {
		s.flushHistoryUpdates(false)
	}
	s.unlockHistory()
	s.afterNavigation()
}

// PopUntil calls Pop repeatedly until the predicate returns true.
//
// Flutter: widgets/navigator.dart NavigatorState.popUntil
func (s *NavigatorState) PopUntil(predicate RoutePredicate) {
	candidate := s.lastRouteEntryWhere(routeEntryIsPresent)
	for candidate != nil {
		if predicate(candidate.route) {
			return
		}
		s.Pop()
		candidate = s.lastRouteEntryWhere(routeEntryIsPresent)
	}
}

// PopUntilWithResult calls Pop repeatedly until the predicate returns true; the
// last route popped completes with the result.
//
// Flutter: widgets/navigator.dart NavigatorState.popUntilWithResult
func (s *NavigatorState) PopUntilWithResult(predicate RoutePredicate, result any) {
	candidate := s.lastRouteEntryWhere(routeEntryIsPresent)
	for candidate != nil {
		if predicate(candidate.route) {
			return
		}
		// Checks what would be next if this route is popped.
		current := candidate
		next := s.lastRouteEntryWhere(func(e *routeEntry) bool { return e.isPresent() && e != current })
		if next != nil && !next.route.WillHandlePopInternally() && predicate(next.route) {
			s.Pop(result)
		} else {
			s.Pop()
		}
		candidate = s.lastRouteEntryWhere(routeEntryIsPresent)
	}
}

// RemoveRoute removes the route from the history without animating it; the
// route completes with the result.
//
// Flutter: widgets/navigator.dart NavigatorState.removeRoute
func (s *NavigatorState) RemoveRoute(route Route, result ...any) {
	s.lockHistory()
	if foundation.DebugMode && !route.AsRoute().isInstalledIn(s) {
		panic(errors.New("NavigatorRouteMissing", "this Navigator does not contain the specified route", "route", route.String()))
	}
	wasCurrent := route.IsCurrent()
	entry := s.firstRouteEntryWhere(routeEntryIsRoute(route))
	entry.complete(optionalResult(result), false, true)
	s.flushHistoryUpdates(false)
	s.unlockHistory()
	if wasCurrent {
		s.afterNavigation()
	}
}

// RemoveRouteBelow removes the route below the anchor without animating it.
//
// Flutter: widgets/navigator.dart NavigatorState.removeRouteBelow
func (s *NavigatorState) RemoveRouteBelow(anchorRoute Route, result ...any) {
	s.lockHistory()
	if foundation.DebugMode && !anchorRoute.AsRoute().isInstalledIn(s) {
		panic(errors.New("NavigatorRouteMissing", "this Navigator does not contain the specified anchorRoute", "route", anchorRoute.String()))
	}
	index := s.indexBelowAnchor(anchorRoute)
	s.history.entries[index].complete(optionalResult(result), false, true)
	s.flushHistoryUpdates(false)
	s.unlockHistory()
}

// FinalizeRoute is called by a route that was popped when it is done and can be
// removed from the history and disposed.
//
// Flutter: widgets/navigator.dart NavigatorState.finalizeRoute
func (s *NavigatorState) FinalizeRoute(route Route) {
	// FinalizeRoute may be called while the navigator is locked as a response to
	// route.didPop; it leaves the lock as it found it.
	wasDebugLocked := s.debugLocked
	if foundation.DebugMode {
		s.debugLocked = true
	}
	index := s.history.indexWhere(routeEntryIsRoute(route))
	entry := s.history.entries[index]
	// For a page-based route with zero transition, finalizeRoute can be called
	// in any lifecycle above pop.
	if entry.pageBased && entry.currentState < routePop {
		s.observedRouteDeletions = append(s.observedRouteDeletions, &navigatorObservation{
			kind: observationPop, primary: route, secondary: s.getRouteBefore(index-1, routeEntryWillBePresent).routeOrNil(),
		})
	} else if foundation.DebugMode && entry.currentState != routePopping {
		panic(errors.New("RouteEntryStateInvalid", "finalizeRoute was called for a route that is not popping",
			"state", int(entry.currentState)))
	}
	entry.finalize()
	// finalizeRoute can be called during flushHistoryUpdates if a pop finishes
	// synchronously.
	if !s.flushingHistory {
		s.flushHistoryUpdates(false)
	}
	if foundation.DebugMode {
		s.debugLocked = wasDebugLocked
	}
}

// DidStartUserGesture is called when a gesture that may pop the route starts,
// for instance a swipe to go back.
//
// Flutter: widgets/navigator.dart NavigatorState.didStartUserGesture
func (s *NavigatorState) DidStartUserGesture() {
	s.setUserGesturesInProgress(s.userGesturesCount + 1)
	if s.userGesturesCount == 1 {
		routeIndex := s.getIndexBefore(len(s.history.entries)-1, routeEntryWillBePresent)
		route := s.history.entries[routeIndex].route
		var previousRoute Route
		if !route.WillHandlePopInternally() && routeIndex > 0 {
			previousRoute = s.getRouteBefore(routeIndex-1, routeEntryWillBePresent).route
		}
		for _, observer := range slices.Clone(s.effectiveObservers) {
			observer.DidStartUserGesture(route, previousRoute)
		}
	}
}

// DidStopUserGesture is called when the gesture stopped.
//
// Flutter: widgets/navigator.dart NavigatorState.didStopUserGesture
func (s *NavigatorState) DidStopUserGesture() {
	if foundation.DebugMode && s.userGesturesCount <= 0 {
		panic(errors.New("NavigatorUserGestureNotStarted", "didStopUserGesture was called without a user gesture in progress"))
	}
	s.setUserGesturesInProgress(s.userGesturesCount - 1)
	if s.userGesturesCount == 0 {
		for _, observer := range slices.Clone(s.effectiveObservers) {
			observer.DidStopUserGesture()
		}
	}
}

func (s *NavigatorState) setUserGesturesInProgress(value int) {
	s.userGesturesCount = value
	s.UserGestureInProgressNotifier().SetValue(value > 0)
}

func (s *NavigatorState) handlePointerDown(e *gestures.PointerDownEvent) {
	s.activePointers[e.Pointer()] = true
}

func (s *NavigatorState) handlePointerUp(e *gestures.PointerUpEvent) {
	delete(s.activePointers, e.Pointer())
}

func (s *NavigatorState) handlePointerCancel(e *gestures.PointerCancelEvent) {
	delete(s.activePointers, e.Pointer())
}

func (s *NavigatorState) cancelActivePointers() {
	binding := WidgetsBindingOf(s.Context())
	if binding.Scheduler().SchedulerPhase() == scheduler.SchedulerPhaseIdle {
		// Between frames, any subsequent pointer of this frame is absorbed. The
		// flag is reset in the next frame, see Build.
		if overlay := s.Overlay(); overlay != nil {
			if absorber, ok := FindAncestorRenderObjectOfType[*rendering.RenderAbsorbPointer](overlay.Context()); ok {
				s.SetState(func() { absorber.SetAbsorbing(true) })
			}
		}
	}
	pointers := make([]int, 0, len(s.activePointers))
	for p := range s.activePointers {
		pointers = append(pointers, p)
	}
	slices.Sort(pointers)
	for _, p := range pointers {
		binding.Gestures().CancelPointer(p)
	}
}

// Build builds the overlay of the routes.
//
// Flutter: widgets/navigator.dart NavigatorState.build
func (s *NavigatorState) Build(bc BuildContext) Widget {
	s.debugRequireUnlocked()
	if foundation.DebugMode && len(s.history.entries) == 0 {
		panic(errors.New("NavigatorHistoryEmpty", "the Navigator has no routes to build"))
	}
	var initialEntries []*OverlayEntry
	if s.Overlay() == nil {
		initialEntries = s.allRouteOverlayEntries()
	}
	// Hides the HeroControllerScope from the widget subtree, so that the nested
	// navigators underneath do not pick up the hero controller above this level.
	return NewHeroControllerScopeNone(
		NewNotificationListener(
			NewListener(
				NewAbsorbPointer(
					// TODO(step 10.5): FocusTraversalGroup and Focus (the navigator's
					// focus node) wrap the overlay.
					NewOverlay(OverlayOpts{Key: s.overlayKey, ClipBehavior: s.Widget().o.ClipBehavior, InitialEntries: initialEntries}),
					AbsorbPointerOpts{Absorbing: foundation.Some(false)},
				),
				ListenerOpts{
					OnPointerDown:   s.handlePointerDown,
					OnPointerUp:     s.handlePointerUp,
					OnPointerCancel: s.handlePointerCancel,
				},
			),
			func(notification NavigationNotification) bool {
				// If the state of this Navigator does not change whether or not the
				// whole framework can pop, the notification is propagated as it is.
				if notification.CanHandlePop || !s.getNavigatorCanHandlePop() {
					return false
				}
				// Otherwise, a new notification with the correct canPop is
				// dispatched, and the propagation of the old one stops.
				DispatchNotification(bc, NavigationNotification{CanHandlePop: true})
				return true
			},
		),
	)
}

// String describes the navigator.
func (s *NavigatorState) String() string {
	return foundation.DescribeIdentity(s)
}
