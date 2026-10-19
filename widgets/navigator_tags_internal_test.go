package widgets

import (
	"testing"

	"github.com/rm4n0s/glutter/animation"
)

// Route tests of the failure tags of step 10.2 that guard the invariants of the
// private route entries and hero machinery (Flutter's asserts). The public ones
// are tested in navigator_tags_test.go.

func tagsTestRoute() Route {
	return NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
		return NewContainer(nil)
	})
}

func TestRouteEntryFailures(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("RouteEntryInitialStateInvalid", func(t *testing.T) {
		requireTag(t, "RouteEntryInitialStateInvalid", func() { newRouteEntry(tagsTestRoute(), routePop, false) })
	})
	t.Run("RouteEntrySettingsNotAPage", func(t *testing.T) {
		requireTag(t, "RouteEntrySettingsNotAPage", func() { newRouteEntry(tagsTestRoute(), routeAdd, true) })
	})
	t.Run("RouteEntryStateInvalid", func(t *testing.T) {
		entry := newRouteEntry(tagsTestRoute(), routeAdd, false)
		requireTag(t, "RouteEntryStateInvalid", func() { entry.handlePush(&NavigatorState{}, true, nil, nil) })
	})
	t.Run("NavigatorNotLocked", func(t *testing.T) {
		entry := newRouteEntry(tagsTestRoute(), routePush, false)
		requireTag(t, "NavigatorNotLocked", func() { entry.handlePop(&NavigatorState{}, nil) })
	})
	t.Run("RouteNotInstalledHere", func(t *testing.T) {
		entry := newRouteEntry(tagsTestRoute(), routePush, false)
		navigator := &NavigatorState{debugLocked: true}
		requireTag(t, "RouteNotInstalledHere", func() { entry.handlePop(navigator, nil) })
	})
}

func TestHeroFailures(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("HeroNotMounted", func(t *testing.T) {
		requireTag(t, "HeroNotMounted", func() { (&heroState{}).startFlight(false) })
	})
	t.Run("HeroSubtreeMissing", func(t *testing.T) {
		requireTag(t, "HeroSubtreeMissing", func() { heroBoundingBoxFor(nil, nil) })
	})
}

func tagsTestHeroState(tag string) *heroState {
	s := &heroState{}
	s.widget, s.hasWidget = NewHero(tag, nil), true
	return s
}

func TestHeroFlightFailures(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("HeroFlightAborted", func(t *testing.T) {
		requireTag(t, "HeroFlightAborted", func() { (&heroFlight{aborted: true}).start(nil) })
	})
	t.Run("HeroFlightStatusInvalid", func(t *testing.T) {
		dismissed := animation.NewCurvedAnimation(animation.AlwaysDismissed, animation.Curves.Linear)
		for _, flightType := range []HeroFlightDirection{HeroFlightDirectionPop, HeroFlightDirectionPush} {
			manifest := &heroFlightManifest{flightType: flightType, curved: dismissed}
			requireTag(t, "HeroFlightStatusInvalid", func() { (&heroFlight{}).start(manifest) })
		}
	})
	t.Run("HeroFlightTagMismatch", func(t *testing.T) {
		flight := &heroFlight{manifest: &heroFlightManifest{fromHero: tagsTestHeroState("a")}}
		requireTag(t, "HeroFlightTagMismatch", func() { flight.divert(&heroFlightManifest{fromHero: tagsTestHeroState("b")}) })
	})
}

func TestModalScopeRouteChanged(t *testing.T) {
	skipUnlessDebug(t)
	one, two := tagsTestRoute().(ModalRoute), tagsTestRoute().(ModalRoute)
	s := &modalScopeState{}
	s.widget, s.hasWidget = newModalScope(nil, one), true
	requireTag(t, "ModalScopeRouteChanged", func() { s.DidUpdateWidget(newModalScope(nil, two)) })
}

func TestNavigatorPagesUpdateReentrant(t *testing.T) {
	skipUnlessDebug(t)
	s := &NavigatorState{debugUpdatingPage: true}
	requireTag(t, "NavigatorPagesUpdateReentrant", func() { s.updatePages() })
}
