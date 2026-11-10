package cupertino

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are of the errors of the functions that no exported function reaches by itself.

func requireRoute(t *testing.T, f func(), route string) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug-only checks of the framework")
	}
	defer func() {
		r := recover()
		err, ok := r.(error)
		if !ok {
			t.Fatalf("panic value is %T (%v), want an error", r, r)
		}
		appErr, ok := errors.FromError(err)
		if !ok || !appErr.HasRoute(route) {
			t.Errorf("unexpected failure path: %v (want %s)", err, route)
		}
	}()
	f()
}

func TestStartPopGestureOfARouteThatDoesNotAllowIt(t *testing.T) {
	route := NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) },
		CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{FullscreenDialog: true}})
	defer route.Dispose()
	requireRoute(t, func() { startPopGesture(route) }, "startPopGesture.CupertinoPopGestureDisabled")
}

func TestHitTestableAtOriginOfAnUnmountedStatusBar(t *testing.T) {
	requireRoute(t, func() { hitTestableAtOrigin(widgets.NewGlobalKey[widgets.State]()) }, "hitTestableAtOrigin.CupertinoStatusBarNotMounted")
}

func TestNavBarLargeTitleMissing(t *testing.T) {
	// A large title that no one gives and the route does not have: the components cannot be made.
	requireRoute(t, func() {
		createNavBarLargeTitle(nil, navBarComponentsOpts{keys: newNavBarComponentsKeys(), large: true})
	}, "createNavBarLargeTitle.NavigationBarLargeTitleMissing")
}

func TestNavBarLargeTitleStyleMissing(t *testing.T) {
	requireRoute(t, func() {
		newTransitionableNavigationBar(transitionableNavigationBarOpts{componentsKeys: newNavBarComponentsKeys(), largeExpanded: true})
	}, "newTransitionableNavigationBar.NavigationBarLargeTitleStyleMissing")
}

func TestSearchFieldScrollOpacity(t *testing.T) {
	// The fade is 1 under the threshold of four fifths of the height, 0 from the full height, and linear between.
	for _, tc := range []struct{ current, max, want float64 }{
		{50, 50, 0}, {60, 50, 0}, {40, 50, 1}, {20, 50, 1}, {45, 50, 0.5}, {42, 50, 0.8},
	} {
		if got := searchFieldScrollOpacity(tc.current, tc.max); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("opacity(%v, %v) = %v, want %v", tc.current, tc.max, got, tc.want)
		}
	}
}
