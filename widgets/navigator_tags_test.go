package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Route tests of the tags of the failures of the navigator and the routes that
// the public API can reach. The tags of states that only the framework itself
// can reach are tested in navigator_tags_internal_test.go.

// overlayOnlyRoute is an OverlayRoute with one entry.
type overlayOnlyRoute struct{ OverlayRouteBase }

func newOverlayOnlyRoute() *overlayOnlyRoute {
	r := &overlayOnlyRoute{}
	InitOverlayRouteBase(&r.OverlayRouteBase, r, RouteOpts{})
	return r
}

func (r *overlayOnlyRoute) CreateOverlayEntries() []*OverlayEntry {
	return []*OverlayEntry{NewOverlayEntry(func(BuildContext) Widget { return NewContainer(nil) })}
}

func navTagsApp(t *testing.T, ts *gluttertest.Tester) *GlobalKey[*NavigatorState] {
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, home: navText("home")}))
	// The tests trip the checks in the middle of a change of the history, which
	// leaves the navigator locked: Flutter never unlocks it, so that the object
	// becomes unusable. The unmounting at the end of the test disposes it, which
	// needs it unlocked, so the cleanup (it runs before the unmounting) does.
	t.Cleanup(func() {
		if state, ok := key.CurrentState(); ok {
			state.ForceUnlockHistory()
		}
	})
	return key
}

func newPlainPage() *PageRouteBuilder {
	return NewPageRouteBuilder(routesPageBuilder(func(BuildContext) Widget { return navText("page") }))
}

func TestNavigatorTagsOfThePublicAPI(t *testing.T) {
	skipUnlessDebug(t)

	t.Run("HeroControllerScopeMissing", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, navText("x")))
		bc := ts.Element(gluttertest.ByText("x"))
		requireTag(t, "HeroControllerScopeMissing", func() { HeroControllerScopeOf(bc) })
	})
	t.Run("LocalHistoryEntryOwned and NotOwned", func(t *testing.T) {
		l := newRoutesLog()
		route := newLHTestRoute(l, "A")
		entry := NewLocalHistoryEntry()
		route.AddLocalHistoryEntry(entry)
		requireTag(t, "LocalHistoryEntryOwned", func() { route.AddLocalHistoryEntry(entry) })
		requireTag(t, "LocalHistoryEntryNotOwned", func() { route.RemoveLocalHistoryEntry(NewLocalHistoryEntry()) })
	})
	t.Run("NavigatorHistoryEmpty", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
			OnGenerateInitialRoutes: func(*NavigatorState, string) []Route { return nil },
		})))
		_ = requireErrorTag(t, ts.TakeError(), "NavigatorHistoryEmpty")
	})
	t.Run("NavigatorLocked", func(t *testing.T) {
		// A route that pushes another one while it is installed.
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		state := navState(t, key)
		l := newRoutesLog()
		route := newLHTestRoute(l, "pusher")
		requireTag(t, "NavigatorLocked", func() {
			route.installHook = func() { state.Push(newPlainPage()) }
			state.Push(route)
		})
		// The pusher is half installed, so the navigator cannot be disposed cleanly;
		// what the unmounting reports is not what the test is about.
		state.ForceUnlockHistory()
		ts.PumpWidget(NewSizedBox(nil))
		ts.TakeErrors()
	})
	t.Run("NavigatorNoActiveRoutes", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		state := navState(t, key)
		state.RemoveRoute(ModalRouteOf(ts.Element(gluttertest.ByText("home"))))
		requireTag(t, "NavigatorNoActiveRoutes", func() { state.PushReplacement(newPlainPage()) })
	})
	t.Run("NavigatorObserverShared", func(t *testing.T) {
		ts := gluttertest.New(t)
		observer := &navTestObserver{}
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewColumn([]Widget{
			NewExpanded(NewNavigator(NavigatorOpts{Observers: []NavigatorObserver{observer}, OnGenerateRoute: func(RouteSettings) Route { return newPlainPage() }})),
			NewExpanded(NewNavigator(NavigatorOpts{Observers: []NavigatorObserver{observer}, OnGenerateRoute: func(RouteSettings) Route { return newPlainPage() }})),
		})))
		_ = requireErrorTag(t, ts.TakeError(), "NavigatorObserverShared")
	})
	t.Run("NavigatorRouteBelowMissing", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		first := ModalRouteOf(ts.Element(gluttertest.ByText("home")))
		requireTag(t, "NavigatorRouteBelowMissing", func() { navState(t, key).ReplaceRouteBelow(first, newPlainPage()) })
	})
	t.Run("NavigatorRouteMissing", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		requireTag(t, "NavigatorRouteMissing", func() { navState(t, key).Replace(newPlainPage(), newPlainPage()) })
	})
	t.Run("NavigatorRouteRemoved", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		state := navState(t, key)
		route := newPlainPage()
		state.Push(route)
		ts.PumpAndSettle()
		state.Pop()
		requireTag(t, "NavigatorRouteRemoved", func() { state.Replace(route, newPlainPage()) })
	})
	t.Run("NavigatorUserGestureNotStarted", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		requireTag(t, "NavigatorUserGestureNotStarted", func() { navState(t, key).DidStopUserGesture() })
	})
	t.Run("OverlayRouteInstalledTwice", func(t *testing.T) {
		route := newOverlayOnlyRoute()
		route.Install()
		requireTag(t, "OverlayRouteInstalledTwice", func() { route.Install() })
	})
	t.Run("RouteAlreadyInstalled", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		route := newPlainPage()
		navPush(key, route)
		requireTag(t, "RouteAlreadyInstalled", func() { navPush(key, route) })
	})
	t.Run("RouteOverlayEntriesEmpty", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		requireTag(t, "RouteOverlayEntriesEmpty", func() { navPush(key, newNotAnnouncedRoute()) })
	})
	t.Run("RouteNotCurrent", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		home := ModalRouteOf(ts.Element(gluttertest.ByText("home"))).(TransitionRoute)
		navPush(key, newPlainPage())
		ts.PumpAndSettle()
		requireTag(t, "RouteNotCurrent", func() { home.HandleStartBackGesture(1) })
	})
	t.Run("HeroControllerRouteNotCurrent", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		home := ModalRouteOf(ts.Element(gluttertest.ByText("home")))
		navPush(key, newPlainPage())
		ts.PumpAndSettle()
		controller := NewHeroController()
		defer controller.Dispose()
		requireTag(t, "HeroControllerRouteNotCurrent", func() { controller.DidChangeTop(home, nil) })
	})
	t.Run("HeroControllerNavigatorMissing", func(t *testing.T) {
		ts := gluttertest.New(t)
		navTagsApp(t, ts)
		home := ModalRouteOf(ts.Element(gluttertest.ByText("home")))
		controller := NewHeroController()
		defer controller.Dispose()
		requireTag(t, "HeroControllerNavigatorMissing", func() { controller.DidChangeTop(home, nil) })
	})
}

func TestTransitionRouteTags(t *testing.T) {
	skipUnlessDebug(t)
	page := routesPageBuilder(func(BuildContext) Widget { return navText("page") })

	t.Run("TransitionRouteControllerMissing", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		route := newCurvedPageRoute(page, func(r *curvedPageRoute) { r.nilController = true })
		requireTag(t, "TransitionRouteControllerMissing", func() { navPush(key, route) })
	})
	t.Run("TransitionRouteAnimationMissing", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := navTagsApp(t, ts)
		route := newCurvedPageRoute(page, func(r *curvedPageRoute) { r.nilAnimation = true })
		requireTag(t, "TransitionRouteAnimationMissing", func() { navPush(key, route) })
	})
	t.Run("TransitionRouteDurationNegative", func(t *testing.T) {
		route := NewPageRouteBuilder(page, PageRouteBuilderOpts{TransitionDuration: foundation.Some(-time.Millisecond)})
		requireTag(t, "TransitionRouteDurationNegative", func() { route.CreateSimulation(true) })
	})
	t.Run("TransitionRouteNotInstalled", func(t *testing.T) {
		route := NewPageRouteBuilder(page)
		requireTag(t, "TransitionRouteNotInstalled", func() { route.DidPush() })
	})
	t.Run("TransitionRouteDisposed", func(t *testing.T) {
		route := NewPageRouteBuilder(page)
		route.Dispose()
		requireTag(t, "TransitionRouteDisposed", func() { route.CreateAnimation() })
		requireTag(t, "TransitionRouteDisposed", func() { route.Dispose() })
	})
}

var _ = animation.AlwaysDismissed
