package widgets_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/heroes_test.dart.
// The Material widgets of the Dart file are replaced by plain ones: a Card is a
// heroTestCard (a padded box that the tests can look for), a TextButton is a
// tappable Text, a MaterialPageRoute is a page route of 300 ms, and a Scaffold
// or Material is left out. Not ported here:
//   - the cases that use the Pages API (step 10.3: 'Heroes should unhide if no
//     animation', 'Can add two page with heroes simultaneously using page API',
//     'Can still trigger hero even if page underneath changes') and the
//     CupertinoApp deep link case;
//   - the iOS back swipe cases (they need CupertinoPageRoute) and the ones that
//     need a ListView inside a PageView or TabBarView;
//   - 'Hero createRectTween' and the other cases that depend on
//     MaterialRectArcTween or MaterialRectCenterArcTween (the default tween is a
//     RectTween here, see SPEC 12.10) use the tweens of this file instead;
//   - the image cases (they need a decoded image provider).

var (
	heroFirstKey     = foundation.NewKey("first")
	heroSecondKey    = foundation.NewKey("second")
	heroThirdKey     = foundation.NewKey("third")
	heroHomeRouteKey = foundation.NewKey("homeRoute")
	heroRouteTwoKey  = foundation.NewKey("routeTwo")
)

// heroTestCard is Card: a box with a margin that the tests can find.
type heroTestCard struct {
	StatelessWidget
	child Widget
}

func newHeroTestCard(child Widget) *heroTestCard {
	return &heroTestCard{NewStatelessWidgetBase(), child}
}

func (w *heroTestCard) Build(BuildContext) Widget {
	return NewPadding(painting.NewEdgeInsetsAll(4), w.child)
}

// heroButton is TextButton.
func heroButton(label string, onTap func()) Widget {
	return NewGestureDetector(navText(label), GestureDetectorOpts{OnTap: onTap, Behavior: rendering.HitTestBehaviorOpaque})
}

func heroSized(key foundation.Key, w, h float64) Widget {
	return NewSizedBox(nil, SizedBoxOpts{Key: key, Width: foundation.Some(w), Height: foundation.Some(h)})
}

// heroPageRoute is a MaterialPageRoute: a page route of 300 ms without a
// transition.
func heroPageRoute(builder WidgetBuilder, settings ...RouteSettings) *navTestRoute {
	o := navTestRouteOpts{
		builder:            builder,
		maintainState:      true,
		transitionDuration: 300 * time.Millisecond, reverseTransitionDuration: 300 * time.Millisecond,
	}
	if len(settings) > 0 {
		o.settings = settings[0]
	}
	return newNavTestRoute(o)
}

func heroAppFor(builder RouteFactory, observers ...NavigatorObserver) Widget {
	return navTestApp(navTestAppOpts{onGenerateRoute: builder, observers: observers})
}

func heroApp(home Widget, observers ...NavigatorObserver) Widget {
	return navTestApp(navTestAppOpts{home: home, observers: observers})
}

// heroRoutes are the routes of the Dart file: '/', '/two', '/twoInset' and
// '/simple'.
func heroRoutes(fromUserGestures bool) map[string]WidgetBuilder {
	push := func(name string) func(BuildContext) func() {
		return func(bc BuildContext) func() { return func() { NavigatorPushNamed(bc, name) } }
	}
	return map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return NewListView([]Widget{
				heroSized(nil, 100, 100),
				newHeroTestCard(NewHero("a", heroSized(heroFirstKey, 100, 100), HeroOpts{TransitionOnUserGestures: fromUserGestures})),
				heroSized(nil, 100, 100),
				heroButton("two", push("/two")(bc)),
				heroButton("twoInset", push("/twoInset")(bc)),
				heroButton("simple", push("/simple")(bc)),
			}, ListViewOpts{Key: heroHomeRouteKey})
		},
		"/two": func(bc BuildContext) Widget {
			return NewListView([]Widget{
				heroButton("pop", func() { NavigatorPop(bc) }),
				heroSized(nil, 150, 150),
				newHeroTestCard(NewHero("a", heroSized(heroSecondKey, 150, 150), HeroOpts{TransitionOnUserGestures: fromUserGestures})),
				heroSized(nil, 150, 150),
				heroButton("three", func() { NavigatorPush(bc, heroThreeRoute()) }),
			}, ListViewOpts{Key: heroRouteTwoKey})
		},
		"/twoInset": func(bc BuildContext) Widget {
			return NewListView([]Widget{
				heroButton("pop", func() { NavigatorPop(bc) }),
				heroSized(nil, 150, 150),
				newHeroTestCard(NewPadding(painting.NewEdgeInsetsFromLTRB(50, 0, 0, 0),
					NewHero("a", heroSized(heroSecondKey, 150, 150), HeroOpts{TransitionOnUserGestures: fromUserGestures}))),
				heroSized(nil, 150, 150),
				heroButton("three", func() { NavigatorPush(bc, heroThreeRoute()) }),
			}, ListViewOpts{Key: heroRouteTwoKey})
		},
	}
}

func heroThreeRoute() Route {
	return heroPageRoute(func(BuildContext) Widget {
		return NewListView([]Widget{
			heroSized(nil, 200, 200),
			newHeroTestCard(NewHero("a", heroSized(heroThirdKey, 200, 200))),
			heroSized(nil, 200, 200),
		}, ListViewOpts{Key: foundation.NewKey("routeThree")})
	})
}

func heroByKey(ts *gluttertest.Tester, key foundation.Key, offstage ...bool) gluttertest.Finder {
	return gluttertest.ByKey(key, gluttertest.FindOpts{IncludeOffstage: len(offstage) > 0 && offstage[0]})
}

func heroExists(ts *gluttertest.Tester, key foundation.Key) bool {
	return ts.Count(heroByKey(ts, key)) > 0
}

func heroIsOnstage(ts *gluttertest.Tester, key foundation.Key) bool {
	return ts.Count(heroByKey(ts, key)) == 1
}

// heroIsOffstage is whether the widget is only found when offstage ones count.
func heroIsOffstage(ts *gluttertest.Tester, key foundation.Key) bool {
	return ts.Count(heroByKey(ts, key)) == 0 && ts.Count(heroByKey(ts, key, true)) == 1
}

func heroIsInCard(ts *gluttertest.Tester, key foundation.Key, offstage ...bool) bool {
	return ts.Count(gluttertest.Ancestor(heroByKey(ts, key, offstage...), gluttertest.ByType[*heroTestCard](gluttertest.FindOpts{IncludeOffstage: len(offstage) > 0 && offstage[0]}))) > 0
}

func heroHeight(ts *gluttertest.Tester, key foundation.Key, offstage ...bool) float64 {
	return ts.Size(heroByKey(ts, key, offstage...)).Height
}

func approx(a, b, epsilon float64) bool { return math.Abs(a-b) <= epsilon }

func TestHeroesAnimate(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navRoutesApp(heroRoutes(false)))

	check := func(what string, cond bool) {
		t.Helper()
		if !cond {
			t.Errorf("%s", what)
		}
	}
	check("first onstage", heroIsOnstage(ts, heroFirstKey))
	check("first in card", heroIsInCard(ts, heroFirstKey))
	check("no second", !heroExists(ts, heroSecondKey))

	ts.Tap(gluttertest.ByText("two"))
	ts.Pump() // begin navigation

	// The second route is offstage, so that the hero party can be formed.
	check("first onstage 2", heroIsOnstage(ts, heroFirstKey))
	check("first in card 2", heroIsInCard(ts, heroFirstKey))
	check("second offstage", heroIsOffstage(ts, heroSecondKey))
	check("second in card (offstage)", heroIsInCard(ts, heroSecondKey, true))

	ts.Pump()
	// The heroes have just gone on their journey, at t=16ms. The original page
	// no longer contains the hero.
	check("first gone", !heroExists(ts, heroFirstKey))
	check("second there", heroIsOnstage(ts, heroSecondKey))
	check("second flying", !heroIsInCard(ts, heroSecondKey))

	ts.Pump()
	// t=32ms for the journey.
	check("first gone at 32ms", !heroExists(ts, heroFirstKey))
	check("second there at 32ms", heroIsOnstage(ts, heroSecondKey))
	check("second flying at 32ms", !heroIsInCard(ts, heroSecondKey))

	ts.Pump(time.Second)
	// The journey has ended: the hero is in the new page, onstage.
	check("first gone at the end", !heroExists(ts, heroFirstKey))
	check("second there at the end", heroIsOnstage(ts, heroSecondKey))
	check("second in card at the end", heroIsInCard(ts, heroSecondKey))

	ts.Pump()
	check("second still there", heroIsOnstage(ts, heroSecondKey) && heroIsInCard(ts, heroSecondKey) && !heroExists(ts, heroFirstKey))

	// Now move on to view 3.
	ts.Tap(gluttertest.ByText("three"))
	ts.Pump()
	check("second onstage", heroIsOnstage(ts, heroSecondKey))
	check("second in card 3", heroIsInCard(ts, heroSecondKey))
	check("third offstage", heroIsOffstage(ts, heroThirdKey))
	check("third in card (offstage)", heroIsInCard(ts, heroThirdKey, true))

	ts.Pump()
	check("second gone", !heroExists(ts, heroSecondKey))
	check("third there", heroIsOnstage(ts, heroThirdKey))
	check("third flying", !heroIsInCard(ts, heroThirdKey))

	ts.Pump()
	check("second gone at 32ms", !heroExists(ts, heroSecondKey))
	check("third there at 32ms", heroIsOnstage(ts, heroThirdKey))
	check("third flying at 32ms", !heroIsInCard(ts, heroThirdKey))

	ts.Pump(time.Second)
	check("second gone at the end", !heroExists(ts, heroSecondKey))
	check("third there at the end", heroIsOnstage(ts, heroThirdKey))
	check("third in card at the end", heroIsInCard(ts, heroThirdKey))

	ts.Pump()
	check("third still there", heroIsOnstage(ts, heroThirdKey) && heroIsInCard(ts, heroThirdKey) && !heroExists(ts, heroSecondKey))
}

func TestHeroesStillAnimateAfterHeroControllerIsSwapped(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	heroKey := foundation.NewUniqueKey()
	shuttle := func(BuildContext, animation.Animation[float64], HeroFlightDirection, BuildContext, BuildContext) Widget {
		return NewContainer(nil, ContainerOpts{Key: heroKey})
	}
	heroPage := func(BuildContext) Widget {
		return NewHero("hero", NewContainer(nil), HeroOpts{FlightShuttleBuilder: shuttle})
	}
	app := func(controller *HeroController) Widget {
		return NewDirectionality(ui.TextDirectionLTR, NewHeroControllerScope(controller, NewNavigator(NavigatorOpts{
			Key:          key,
			InitialRoute: "navigator1",
			OnGenerateRoute: func(s RouteSettings) Route {
				return heroPageRoute(heroPage, s)
			},
		})))
	}
	controller1 := NewHeroController()
	defer controller1.Dispose()
	ts.PumpWidget(app(controller1))
	navPush(key, heroPageRoute(heroPage))
	if heroExists(ts, heroKey) {
		t.Fatal("the shuttle should not show yet")
	}
	// Begins the navigation.
	ts.Pump()
	ts.Pump(30 * time.Millisecond)
	if !heroIsOnstage(ts, heroKey) {
		t.Fatal("the hero should be flying")
	}
	controller2 := NewHeroController()
	defer controller2.Dispose()
	ts.PumpWidget(app(controller2))
	// The original animation still flies.
	if !heroIsOnstage(ts, heroKey) {
		t.Fatal("the hero should still be flying")
	}
	ts.PumpAndSettle()
	if heroExists(ts, heroKey) {
		t.Error("the flight should be over")
	}
}

func TestHeroesAnimateShouldHideOriginalHero(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navRoutesApp(heroRoutes(false)))
	if !heroIsOnstage(ts, heroFirstKey) || !heroIsInCard(ts, heroFirstKey) || heroExists(ts, heroSecondKey) {
		t.Fatal("unexpected initial state")
	}
	ts.Tap(gluttertest.ByText("two"))
	ts.PumpAndSettle() // waits for the transition to finish
	if heroExists(ts, heroFirstKey) {
		t.Fatal("the first hero should be offstage")
	}
	ancestors := ts.Elements(gluttertest.Ancestor(heroByKey(ts, heroFirstKey, true), gluttertest.ByType[*Offstage](gluttertest.FindOpts{IncludeOffstage: true})))
	if len(ancestors) == 0 {
		t.Fatal("no Offstage above the first hero")
	}
	// The original hero should stay hidden.
	if !ancestors[0].Widget().(*Offstage).IsOffstage() {
		t.Error("the original hero should be offstage")
	}
	if !heroIsOnstage(ts, heroSecondKey) || !heroIsInCard(ts, heroSecondKey) {
		t.Error("the second hero should show in its card")
	}
}

func TestDestinationHeroIsRebuiltMidflight(t *testing.T) {
	ts := gluttertest.New(t)
	route := heroPageRoute(func(BuildContext) Widget {
		return NewHero("a", NewText("MutatingRoute"), HeroOpts{Key: foundation.NewUniqueKey()})
	})
	ts.PumpWidget(heroApp(NewListView([]Widget{
		NewHero("a", NewText("foo")),
		NewBuilder(func(bc BuildContext) Widget {
			return heroButton("two", func() { NavigatorPush(bc, route) })
		}),
	})))
	ts.Tap(gluttertest.ByText("two"))
	ts.Pump(10 * time.Millisecond)
	route.SetState(func() {})
	ts.Pump(10 * time.Millisecond)
	ts.Pump(time.Second)
	if err := ts.TakeError(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func heroCurveFlight(t *testing.T, curve animation.Curve, hero func(curve animation.Curve) map[string]WidgetBuilder) {
	t.Helper()
	ts := gluttertest.New(t)
	observer := &transitionDurationObserver{}
	ts.PumpWidget(navRoutesApp(hero(curve), navTestAppOpts{observers: []NavigatorObserver{observer}}))
	ts.Tap(gluttertest.ByText("two"))
	ts.Pump() // begin navigation

	// The height of the second hero varies from 100 to 150 over the duration,
	// according to the curve.
	duration := observer.duration.OrElse(0)
	if duration == 0 {
		t.Fatal("no transition duration observed")
	}
	initial := heroHeight(ts, heroFirstKey, true)
	final := heroHeight(ts, heroSecondKey, true)
	delta := final - initial
	for _, fraction := range []float64{0.25, 0.5, 0.75, 1} {
		ts.Pump(time.Duration(float64(duration) * 0.25))
		want := curve.Transform(fraction)*delta + initial
		if got := heroHeight(ts, heroSecondKey); !approx(got, want, 0.001) {
			t.Errorf("curve %v: height at %v = %v, want %v", curve, fraction, got, want)
		}
	}
}

func TestDefaultHeroAnimationIsFastOutSlowIn(t *testing.T) {
	heroCurveFlight(t, animation.Curves.FastOutSlowIn, func(animation.Curve) map[string]WidgetBuilder { return heroRoutes(false) })
}

func TestHeroesAnimationCurveIsCustomizable(t *testing.T) {
	for _, curve := range []animation.Curve{
		animation.Curves.Linear, animation.Curves.Ease, animation.Curves.LinearToEaseOut,
		animation.Curves.EaseInOutCubicEmphasized, animation.Curves.FastOutSlowIn,
	} {
		t.Run(fmt.Sprint(curve), func(t *testing.T) {
			heroCurveFlight(t, curve, func(curve animation.Curve) map[string]WidgetBuilder {
				return map[string]WidgetBuilder{
					"/": func(bc BuildContext) Widget {
						return NewListView([]Widget{
							heroSized(nil, 100, 100),
							newHeroTestCard(NewHero("a", heroSized(heroFirstKey, 100, 100), HeroOpts{Curve: curve})),
							heroSized(nil, 100, 100),
							heroButton("two", func() { NavigatorPushNamed(bc, "/two") }),
						})
					},
					"/two": func(bc BuildContext) Widget {
						return NewListView([]Widget{
							heroButton("pop", func() { NavigatorPop(bc) }),
							heroSized(nil, 150, 150),
							newHeroTestCard(NewHero("a", heroSized(heroSecondKey, 150, 150), HeroOpts{Curve: curve})),
						}, ListViewOpts{Key: heroRouteTwoKey})
					},
				}
			})
		})
	}
}

func TestHeroesAreNotInteractive(t *testing.T) {
	ts := gluttertest.New(t)
	var log []string
	ts.PumpWidget(navTestApp(navTestAppOpts{
		home: NewCenter(NewHero("foo", NewGestureDetector(
			NewSizedBox(NewText("foo"), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
			GestureDetectorOpts{OnTap: func() { log = append(log, "foo") }}))),
		routes: map[string]WidgetBuilder{
			"/next": func(BuildContext) Widget {
				return NewAlign(NewHero("foo", NewGestureDetector(
					NewSizedBox(NewText("bar"), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(150.0)}),
					GestureDetectorOpts{OnTap: func() { log = append(log, "bar") }})),
					AlignOpts{Alignment: painting.AlignmentTopLeft})
			},
		},
	}))
	if len(log) != 0 {
		t.Fatalf("log = %v", log)
	}
	ts.Tap(gluttertest.ByText("foo"))
	if len(log) != 1 || log[0] != "foo" {
		t.Fatalf("log = %v, want [foo]", log)
	}
	log = nil

	gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]()).PushNamed("/next")
	if len(log) != 0 {
		t.Fatalf("log = %v", log)
	}
	off := func(text string) gluttertest.Finder {
		return gluttertest.ByText(text, gluttertest.FindOpts{IncludeOffstage: true})
	}
	ts.Tap(off("foo"))
	if len(log) != 0 {
		t.Fatalf("log = %v after tapping foo offstage", log)
	}
	ts.Pump(10 * time.Millisecond)
	ts.Tap(off("foo"))
	ts.Tap(off("bar"))
	if len(log) != 0 {
		t.Fatalf("log = %v during the flight", log)
	}
	ts.Pump(10 * time.Millisecond)
	if ts.Count(gluttertest.ByText("foo")) != 0 {
		t.Fatal("foo should be gone")
	}
	ts.Tap(off("bar"))
	if len(log) != 0 {
		t.Fatalf("log = %v during the flight", log)
	}
	ts.Pump(time.Second)
	ts.Tap(gluttertest.ByText("bar"))
	if len(log) != 1 || log[0] != "bar" {
		t.Errorf("log = %v, want [bar]", log)
	}
}

func TestPoppingOnFirstFrameDoesNotCauseHeroObserverToCrash(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(heroAppFor(func(settings RouteSettings) Route {
		return heroPageRoute(func(BuildContext) Widget { return NewHero("test", NewContainer(nil)) }, settings)
	}))
	ts.Pump()
	heroes := ts.Elements(gluttertest.ByType[*Hero]())
	if len(heroes) != 1 {
		t.Fatalf("%d heroes, want 1", len(heroes))
	}
	NavigatorPushNamed(heroes[0], "test")
	ts.Pump() // adds the new page to the tree...
	NavigatorPop(heroes[0])
	ts.Pump() // ...and removes it straight away
	if err := ts.TakeError(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOverlappingStartingAndEndingAHeroTransitionWorksOk(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(heroAppFor(func(settings RouteSettings) Route {
		return heroPageRoute(func(BuildContext) Widget { return NewHero("test", NewContainer(nil)) }, settings)
	}))
	ts.Pump()
	hero := ts.Elements(gluttertest.ByType[*Hero]())[0]
	NavigatorPushNamed(hero, "test")
	ts.Pump()
	ts.Pump(time.Hour)
	NavigatorPushNamed(hero, "test")
	ts.Pump()
	ts.Pump(time.Hour)
	NavigatorPop(hero)
	ts.Pump()
	NavigatorPop(hero)
	ts.Pump(time.Hour) // the first transition is finished, but the second has not started
	ts.Pump()
	if err := ts.TakeError(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOneRouteTwoHeroesSameTagThrows(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(heroApp(NewListView([]Widget{
		NewHero("a", NewText("a")),
		NewHero("a", NewText("a too")),
		NewBuilder(func(bc BuildContext) Widget {
			return heroButton("push", func() {
				NavigatorPush(bc, NewPageRouteBuilder(routesPageBuilder(func(BuildContext) Widget { return NewText("fail") })))
			})
		}),
	})))
	ts.Tap(gluttertest.ByText("push"))
	ts.Pump()
	_ = requireErrorTag(t, ts.TakeError(), "HeroTagDuplicated")
}

func heroRectTweenOf(begin, end foundation.Opt[ui.Rect]) *animation.Tween[foundation.Opt[ui.Rect]] {
	return animation.NewRectTween(begin, end)
}

func heroVerticalRoutes(tween CreateRectTween) map[string]WidgetBuilder {
	return map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return NewColumn([]Widget{
				NewHero("a", heroSized(heroFirstKey, 100, 100), HeroOpts{CreateRectTween: tween}),
				heroButton("two", func() { NavigatorPushNamed(bc, "/two") }),
			}, ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})
		},
		"/two": func(bc BuildContext) Widget {
			return NewColumn([]Widget{
				NewSizedBox(heroButton("pop", func() { NavigatorPop(bc) }), SizedBoxOpts{Height: foundation.Some(200.0)}),
				NewHero("a", heroSized(heroSecondKey, 100, 200), HeroOpts{CreateRectTween: tween}),
			})
		},
	}
}

func heroRect(ts *gluttertest.Tester, key foundation.Key) ui.Rect {
	f := heroByKey(ts, key)
	o, size := ts.TopLeft(f), ts.Size(f)
	return ui.NewRectFromLTWH(o.Dx, o.Dy, size.Width, size.Height)
}

func rectNear(a, b ui.Rect, epsilon float64) bool {
	return approx(a.Left, b.Left, epsilon) && approx(a.Top, b.Top, epsilon) &&
		approx(a.Right, b.Right, epsilon) && approx(a.Bottom, b.Bottom, epsilon)
}

func TestHeroCreateRectTweenForNavigatorThatIsNotFullScreen(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/25272.
	ts := gluttertest.New(t)
	observer := &transitionDurationObserver{}
	const leftPadding = 10.0
	ts.PumpWidget(NewPadding(painting.NewEdgeInsetsFromLTRB(leftPadding, 0, 0, 0),
		navTestApp(navTestAppOpts{routes: heroVerticalRoutes(heroRectTweenOf), observers: []NavigatorObserver{observer}})))
	if got := ts.Center(heroByKey(ts, heroFirstKey)); got != (ui.Offset{Dx: leftPadding + 50, Dy: 50}) {
		t.Fatalf("center = %v", got)
	}
	duration := observer.duration.OrElse(0)
	curve := animation.Curves.FastOutSlowIn
	rect := func(l, tp, w, h float64) foundation.Opt[ui.Rect] {
		return foundation.Some(ui.NewRectFromLTWH(l, tp, w, h))
	}
	tween := func(begin, end foundation.Opt[ui.Rect]) func(at float64) ui.Rect {
		rt := animation.NewRectTween(begin, end)
		return func(at float64) ui.Rect { return rt.Lerp(curve.Transform(at)).OrElse(ui.Rect{}) }
	}
	push := tween(rect(leftPadding, 0, 100, 100), rect(350+leftPadding/2, 200, 100, 200))
	check := func(key foundation.Key, want func(float64) ui.Rect, steps ...float64) {
		t.Helper()
		for _, at := range steps {
			ts.Pump(time.Duration(float64(duration) * 0.25))
			got := heroRect(ts, key)
			if !rectNear(got, want(at), 0.001) {
				t.Errorf("rect at %v = %v, want %v", at, got, want(at))
			}
		}
	}
	ts.Tap(gluttertest.ByText("two"))
	ts.Pump() // begin navigation
	ts.Pump()
	if got := ts.Center(heroByKey(ts, heroSecondKey)); got != (ui.Offset{Dx: 50 + leftPadding, Dy: 50}) {
		t.Fatalf("center = %v", got)
	}
	check(heroSecondKey, push, 0.25, 0.5, 0.75)
	ts.PumpAndSettle()
	if got := ts.Center(heroByKey(ts, heroSecondKey)); got != (ui.Offset{Dx: 400 + leftPadding/2, Dy: 300}) {
		t.Fatalf("center = %v", got)
	}

	ts.Tap(gluttertest.ByText("pop"))
	ts.Pump() // begin navigation
	pop := tween(rect(350+leftPadding/2, 200, 100, 200), rect(leftPadding, 0, 100, 100))
	ts.Pump()
	if got := ts.Center(heroByKey(ts, heroFirstKey)); got != (ui.Offset{Dx: 400 + leftPadding/2, Dy: 300}) {
		t.Fatalf("center = %v", got)
	}
	check(heroFirstKey, pop, 0.25, 0.5, 0.75)
	ts.PumpAndSettle()
	if got := ts.Center(heroByKey(ts, heroFirstKey)); got != (ui.Offset{Dx: 50 + leftPadding, Dy: 50}) {
		t.Errorf("center = %v", got)
	}
}

func TestHeroPushTransitionInterruptedByAPop(t *testing.T) {
	ts := gluttertest.New(t)
	observer := &transitionDurationObserver{}
	ts.PumpWidget(navTestApp(navTestAppOpts{routes: heroRoutes(false), observers: []NavigatorObserver{observer}}))
	if !heroIsOnstage(ts, heroFirstKey) || !heroIsInCard(ts, heroFirstKey) || heroExists(ts, heroSecondKey) {
		t.Fatal("unexpected initial state")
	}
	ts.Tap(gluttertest.ByText("two"))
	// Starts the flight of the hero from '/' to '/two'; '/two' is offstage.
	ts.Pump()
	initialHeight := heroHeight(ts, heroFirstKey)
	finalHeight := heroHeight(ts, heroSecondKey, true)
	if finalHeight <= initialHeight {
		t.Fatalf("heights %v, %v", initialHeight, finalHeight)
	}
	// Builds the first hero animation frame in the navigator's overlay.
	ts.Pump()
	// The hero widgets have been replaced by placeholders and the destination
	// hero has been moved to the overlay.
	if ts.Count(gluttertest.Descendant(gluttertest.ByKey(heroHomeRouteKey), heroByKey(ts, heroFirstKey))) != 0 ||
		ts.Count(gluttertest.Descendant(gluttertest.ByKey(heroRouteTwoKey), heroByKey(ts, heroSecondKey))) != 0 {
		t.Fatal("the heroes should not be in their routes")
	}
	if heroExists(ts, heroFirstKey) || !heroIsOnstage(ts, heroSecondKey) {
		t.Fatal("the second hero should be flying alone")
	}
	duration := observer.duration.OrElse(0)
	ts.Pump(duration / 2)
	heightMidFlight := heroHeight(ts, heroSecondKey)
	if heightMidFlight <= initialHeight || heightMidFlight >= finalHeight {
		t.Fatalf("mid flight height %v not between %v and %v", heightMidFlight, initialHeight, finalHeight)
	}
	// Pops '/two' before the push transition has finished.
	ts.Tap(gluttertest.ByText("pop"))
	// Restarts the flight; now it flies from '/two' to '/'.
	ts.Pump()
	ts.Pump(50 * time.Millisecond)
	heightBeforeMid := heroHeight(ts, heroSecondKey)
	if heightBeforeMid >= heightMidFlight || heightBeforeMid >= finalHeight {
		t.Errorf("the hero should be shrinking: %v after %v", heightBeforeMid, heightMidFlight)
	}
	ts.Pump(99 * time.Millisecond)
	// The flight is finished: back to where it started.
	ts.Pump(duration)
	if !heroIsOnstage(ts, heroFirstKey) || !heroIsInCard(ts, heroFirstKey) || heroExists(ts, heroSecondKey) {
		t.Error("the first hero should be back in its card")
	}
}

func TestHeroPopTransitionInterruptedByAPush(t *testing.T) {
	ts := gluttertest.New(t)
	observer := &transitionDurationObserver{}
	ts.PumpWidget(navTestApp(navTestAppOpts{routes: heroRoutes(false), observers: []NavigatorObserver{observer}}))
	ts.Tap(gluttertest.ByText("two"))
	ts.Pump()
	ts.Pump(time.Second)
	if !heroIsOnstage(ts, heroSecondKey) || !heroIsInCard(ts, heroSecondKey) || heroExists(ts, heroFirstKey) {
		t.Fatal("the second hero should show in its card")
	}
	ts.Tap(gluttertest.ByText("pop"))
	// Starts the flight from '/two' to '/'; '/two' is now offstage.
	ts.Pump()
	initialHeight := heroHeight(ts, heroSecondKey)
	finalHeight := heroHeight(ts, heroFirstKey, true)
	if finalHeight >= initialHeight {
		t.Fatalf("heights %v, %v", initialHeight, finalHeight)
	}
	ts.Pump()
	if ts.Count(gluttertest.Descendant(gluttertest.ByKey(heroHomeRouteKey), heroByKey(ts, heroFirstKey))) != 0 ||
		ts.Count(gluttertest.Descendant(gluttertest.ByKey(heroRouteTwoKey), heroByKey(ts, heroSecondKey))) != 0 {
		t.Fatal("the heroes should not be in their routes")
	}
	if !heroIsOnstage(ts, heroFirstKey) || heroExists(ts, heroSecondKey) {
		t.Fatal("the first hero should be flying alone")
	}
	// At 150ms the hero is mid-flight.
	ts.Pump(150 * time.Millisecond)
	height150 := heroHeight(ts, heroFirstKey)
	if height150 >= initialHeight || height150 <= finalHeight {
		t.Fatalf("height at 150ms = %v, not between %v and %v", height150, finalHeight, initialHeight)
	}
	// Pushes '/two' before the pop transition has finished.
	ts.Tap(gluttertest.ByText("two"))
	ts.Pump()
	ts.Pump(50 * time.Millisecond)
	if height200 := heroHeight(ts, heroFirstKey); height200 <= height150 || height200 <= finalHeight {
		t.Errorf("the hero should be growing: %v after %v", height200, height150)
	}
	ts.Pump(99 * time.Millisecond)
	ts.Pump(observer.duration.OrElse(0))
	if !heroIsOnstage(ts, heroSecondKey) || !heroIsInCard(ts, heroSecondKey) || heroExists(ts, heroFirstKey) {
		t.Error("the second hero should be back in its card")
	}
}

func TestPopInterruptsPushReversesFlight(t *testing.T) {
	// The path of the flight is the same in both directions here, because the
	// default tween is not an arc (SPEC 12.10); the second half of the Dart case,
	// which shows that the arcs differ after a completed push, is not ported.
	ts := gluttertest.New(t)
	observer := &transitionDurationObserver{}
	ts.PumpWidget(navTestApp(navTestAppOpts{routes: heroRoutes(false), observers: []NavigatorObserver{observer}}))
	ts.Tap(gluttertest.ByText("twoInset"))
	ts.Pump() // begin navigation from / to /twoInset
	duration := observer.duration.OrElse(0)
	step := time.Duration(float64(duration) * 0.1)
	dx := func() float64 { return ts.TopLeft(heroByKey(ts, heroSecondKey)).Dx }
	ts.Pump()
	x0 := dx()
	// The flight begins with the second hero lined up with the first.
	if x0 != 4 {
		t.Fatalf("x0 = %v, want 4", x0)
	}
	var xs []float64
	for i := 0; i < 4; i++ {
		ts.Pump(step)
		xs = append(xs, dx())
	}
	// Pops /twoInset before the push transition has finished.
	ts.Tap(gluttertest.ByText("pop"))
	// The hero takes the same path flying back; it must not jump where the push
	// was interrupted, and ends up where the push started.
	ts.Pump()
	if got := dx(); !approx(got, xs[3], 0.001) {
		t.Errorf("x after the pop = %v, want %v", got, xs[3])
	}
	for i := 2; i >= 0; i-- {
		ts.Pump(step)
		if got := dx(); !approx(got, xs[i], 0.001) {
			t.Errorf("x = %v, want %v", got, xs[i])
		}
	}
	ts.Pump(step)
	if got := dx(); !approx(got, x0, 0.001) {
		t.Errorf("x at the end = %v, want %v", got, x0)
	}
	ts.PumpAndSettle()
	if got := ts.TopLeft(heroByKey(ts, heroFirstKey)).Dx; got != 4 { // card contents are inset by 4
		t.Errorf("first hero x = %v, want 4", got)
	}
}

func TestShuttleAndPlaceholderBuilders(t *testing.T) {
	shuttle := func(text string) HeroFlightShuttleBuilder {
		return func(BuildContext, animation.Animation[float64], HeroFlightDirection, BuildContext, BuildContext) Widget {
			return NewText(text)
		}
	}
	placeholder := func(text string) HeroPlaceholderBuilder {
		return func(BuildContext, ui.Size, Widget) Widget { return NewText(text) }
	}
	run := func(from, to *Hero) *gluttertest.Tester {
		ts := gluttertest.New(t)
		ts.PumpWidget(heroApp(NewListView([]Widget{
			from,
			NewBuilder(func(bc BuildContext) Widget {
				return heroButton("two", func() {
					NavigatorPush(bc, heroPageRoute(func(BuildContext) Widget { return to }))
				})
			}),
		})))
		ts.Tap(gluttertest.ByText("two"))
		ts.Pump()
		ts.Pump(10 * time.Millisecond)
		return ts
	}
	count := func(ts *gluttertest.Tester, text string) int { return ts.Count(gluttertest.ByText(text)) }

	t.Run("can override flight shuttle in to hero", func(t *testing.T) {
		ts := run(NewHero("a", NewText("foo")), NewHero("a", NewText("bar"), HeroOpts{FlightShuttleBuilder: shuttle("baz")}))
		if count(ts, "foo") != 0 || count(ts, "bar") != 0 || count(ts, "baz") != 1 {
			t.Error("only the shuttle of the destination should show")
		}
	})
	t.Run("can override flight shuttle in from hero", func(t *testing.T) {
		ts := run(NewHero("a", NewText("foo"), HeroOpts{FlightShuttleBuilder: shuttle("baz")}), NewHero("a", NewText("bar")))
		if count(ts, "foo") != 0 || count(ts, "bar") != 0 || count(ts, "baz") != 1 {
			t.Error("only the shuttle of the source should show")
		}
	})
	t.Run("toHero's shuttle builder over fromHero's shuttle builder", func(t *testing.T) {
		ts := run(NewHero("a", NewText("foo"), HeroOpts{FlightShuttleBuilder: shuttle("fromHero text")}),
			NewHero("a", NewText("bar"), HeroOpts{FlightShuttleBuilder: shuttle("toHero text")}))
		if count(ts, "foo") != 0 || count(ts, "bar") != 0 || count(ts, "fromHero text") != 0 || count(ts, "toHero text") != 1 {
			t.Error("only the shuttle of the destination should show")
		}
	})
	t.Run("can override flight launch pads", func(t *testing.T) {
		ts := run(NewHero("a", NewText("Batman"), HeroOpts{PlaceholderBuilder: placeholder("Venom")}),
			NewHero("a", NewText("Wolverine"), HeroOpts{PlaceholderBuilder: placeholder("Joker")}))
		// Wolverine shows once, in the hero, because by default the child of the
		// destination hero is the widget in flight.
		if count(ts, "Batman") != 0 || count(ts, "Wolverine") != 1 || count(ts, "Venom") != 1 || count(ts, "Joker") != 1 {
			t.Error("unexpected texts during the flight")
		}
	})
}

func TestHandlesTransitionsWhenANonDefaultInitialRouteIsSet(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navTestApp(navTestAppOpts{routes: heroRoutes(false), initialRoute: "/two"}))
	if err := ts.TakeError(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Count(gluttertest.ByText("two")) != 0 || ts.Count(gluttertest.ByText("three")) != 1 {
		t.Error("the second route should show")
	}
}

func TestHeroNestedNavigators(t *testing.T) {
	t.Run("can push and pop on the outer navigator if the nested one contains heroes", func(t *testing.T) {
		// Regression test for https://github.com/flutter/flutter/issues/28042.
		ts := gluttertest.New(t)
		root := NewGlobalKey[*NavigatorState]()
		nested := NewGlobalKey[*NavigatorState]()
		bottom, top := foundation.NewUniqueKey(), foundation.NewUniqueKey()
		const tag = "You are my hero!"
		ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: root, home: NewNavigator(NavigatorOpts{
			Key: nested,
			OnGenerateRoute: func(RouteSettings) Route {
				return heroPageRoute(func(BuildContext) Widget { return NewHero(tag, NewPlaceholder(nil, PlaceholderOpts{Key: bottom})) })
			},
		})}))
		navPush(nested, heroPageRoute(func(BuildContext) Widget { return NewHero(tag, NewPlaceholder(nil, PlaceholderOpts{Key: top})) }))
		ts.PumpAndSettle()
		// Both heroes are in the tree, one is offstage.
		if !heroIsOnstage(ts, top) || heroExists(ts, bottom) || ts.Count(heroByKey(ts, bottom, true)) != 1 {
			t.Fatal("the top hero should show over the bottom one")
		}
		navPush(root, heroPageRoute(func(BuildContext) Widget { return NewText("Foo") }))
		ts.PumpAndSettle()
		if textsFound(ts, "Foo", false) != 1 {
			t.Fatal("Foo should show")
		}
		// Both heroes are still in the tree, both offstage.
		if heroExists(ts, bottom) || heroExists(ts, top) || ts.Count(heroByKey(ts, bottom, true)) != 1 || ts.Count(heroByKey(ts, top, true)) != 1 {
			t.Fatal("both heroes should be offstage")
		}
		if err := ts.TakeError(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		navState(t, root).Pop()
		ts.PumpAndSettle()
		if textsFound(ts, "Foo", false) != 0 || !heroIsOnstage(ts, top) || heroExists(ts, bottom) || ts.Count(heroByKey(ts, bottom, true)) != 1 {
			t.Error("the nested navigator should show again")
		}
	})

	t.Run("can hero from a route in the root navigator to a route in a nested one", func(t *testing.T) {
		// The Dart case pushes to the root navigator twice; this is the flight
		// between the two routes of the root navigator.
		ts := gluttertest.New(t)
		root := NewGlobalKey[*NavigatorState]()
		small, large := foundation.NewUniqueKey(), foundation.NewUniqueKey()
		card := func(key foundation.Key, size float64) Widget {
			return NewCenter(newHeroTestCard(NewHero("foo", NewContainer(nil, ContainerOpts{Key: key, Color: foundation.Some(routesTestGreen),
				Width: foundation.Some(size), Height: foundation.Some(size)}))))
		}
		ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: root, home: card(large, 200)}))
		if !heroIsOnstage(ts, large) || !heroIsInCard(ts, large) || ts.Count(heroByKey(ts, small, true)) != 0 {
			t.Fatal("unexpected initial state")
		}
		navPush(root, heroPageRoute(func(BuildContext) Widget { return card(small, 100) }))
		ts.Pump()
		if !heroIsOnstage(ts, large) || !heroIsInCard(ts, large) || !heroIsOffstage(ts, small) || !heroIsInCard(ts, small, true) {
			t.Fatal("the second route should exist offstage")
		}
		ts.Pump()
		if heroExists(ts, large) || !heroIsOnstage(ts, small) || heroIsInCard(ts, small) {
			t.Fatal("the hero should have started flying")
		}
		ts.Pump(100 * time.Millisecond)
		if heroExists(ts, large) || !heroIsOnstage(ts, small) || heroIsInCard(ts, small) {
			t.Fatal("the hero should be in flight")
		}
		size := ts.Size(heroByKey(ts, small))
		if size.Height <= 100 || size.Width <= 100 || size.Height >= 200 || size.Width >= 200 {
			t.Errorf("size in flight = %v", size)
		}
		ts.PumpAndSettle()
		if heroExists(ts, large) || !heroIsOnstage(ts, small) || !heroIsInCard(ts, small) || ts.Size(heroByKey(ts, small)) != (ui.Size{Width: 100, Height: 100}) {
			t.Error("the transition should have ended")
		}
	})
}

func TestHeroWithinAHeroThrows(t *testing.T) {
	skipUnlessDebug(t)
	for _, tc := range []struct {
		name  string
		child Widget
	}{
		{"direct", NewHero("a", NewHero("b", NewText("Child of a Hero")))},
		{"with Builder", NewHero("a", NewBuilder(func(BuildContext) Widget { return NewHero("b", NewText("Child of a Hero")) }))},
		{"with LayoutBuilder", NewHero("a", NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
			return NewHero("b", NewText("Child of a Hero"))
		}))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(heroApp(tc.child))
			_ = requireErrorTag(t, ts.TakeError(), "HeroDescendantOfHero")
		})
	}
}

func TestHeroesFlyOnPushReplacement(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/28041.
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	small, large := foundation.NewUniqueKey(), foundation.NewUniqueKey()
	card := func(key foundation.Key, size float64) Widget {
		return NewCenter(newHeroTestCard(NewHero("foo", NewContainer(nil, ContainerOpts{Key: key, Color: foundation.Some(routesTestGreen),
			Width: foundation.Some(size), Height: foundation.Some(size)}))))
	}
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: key, home: card(large, 200)}))
	if !heroIsOnstage(ts, large) || !heroIsInCard(ts, large) || ts.Count(heroByKey(ts, small, true)) != 0 {
		t.Fatal("unexpected initial state")
	}
	navState(t, key).PushReplacement(heroPageRoute(func(BuildContext) Widget { return card(small, 100) }))
	ts.Pump()
	if !heroIsOnstage(ts, large) || !heroIsInCard(ts, large) || !heroIsOffstage(ts, small) || !heroIsInCard(ts, small, true) {
		t.Fatal("the second route should exist offstage")
	}
	ts.Pump()
	if heroExists(ts, large) || !heroIsOnstage(ts, small) || heroIsInCard(ts, small) {
		t.Fatal("the hero should have started flying")
	}
	ts.Pump(100 * time.Millisecond)
	if heroExists(ts, large) || !heroIsOnstage(ts, small) || heroIsInCard(ts, small) {
		t.Fatal("the hero should be in flight")
	}
	size := ts.Size(heroByKey(ts, small))
	if size.Height <= 100 || size.Width <= 100 || size.Height >= 200 || size.Width >= 200 {
		t.Errorf("size in flight = %v", size)
	}
	ts.PumpAndSettle()
	if heroExists(ts, large) || !heroIsOnstage(ts, small) || !heroIsInCard(ts, small) || ts.Size(heroByKey(ts, small)) != (ui.Size{Width: 100, Height: 100}) {
		t.Error("the transition should have ended")
	}
}

func TestDestinationHeroDisappearsMidFlight(t *testing.T) {
	ts := gluttertest.New(t)
	homeHeroKey, routeHeroKey := foundation.NewKey("home hero"), foundation.NewKey("route hero")
	observer := &transitionDurationObserver{}
	routeIncludesHero := true
	var heroCardSetState StateSetter
	route := heroPageRoute(func(BuildContext) Widget {
		return NewListView([]Widget{
			NewStatefulBuilder(func(bc BuildContext, setState StateSetter) Widget {
				heroCardSetState = setState
				if routeIncludesHero {
					return newHeroTestCard(NewHero("H", heroSized(routeHeroKey, 200, 200)))
				}
				return newHeroTestCard(heroSized(nil, 200, 200))
			}),
			NewBuilder(func(bc BuildContext) Widget {
				return heroButton("POP", func() { NavigatorPop(bc) })
			}),
		})
	})
	ts.PumpWidget(navTestApp(navTestAppOpts{observers: []NavigatorObserver{observer}, home: NewBuilder(func(bc BuildContext) Widget {
		return NewListView([]Widget{
			newHeroTestCard(NewHero("H", heroSized(homeHeroKey, 100, 100))),
			heroButton("PUSH", func() { NavigatorPush(bc, route) }),
		})
	})}))
	ts.Tap(gluttertest.ByText("PUSH"))
	ts.Pump()
	ts.Pump()
	initialHeight := heroHeight(ts, routeHeroKey)
	ts.Pump(10 * time.Millisecond)
	if mid := heroHeight(ts, routeHeroKey); mid <= initialHeight || mid >= 200 {
		t.Fatalf("mid flight height = %v", mid)
	}
	duration := observer.duration.OrElse(0)
	ts.Pump(duration)
	ts.Pump()
	finalHeight := heroHeight(ts, routeHeroKey)
	if finalHeight != 200 {
		t.Fatalf("final height = %v, want 200", finalHeight)
	}
	// Completes the flight.
	ts.Pump(100 * time.Millisecond)
	// Rebuilds the route with its hero.
	heroCardSetState(func() { routeIncludesHero = true })
	ts.Pump()
	// Pops the route.
	ts.Tap(gluttertest.ByText("POP"))
	ts.Pump()
	ts.Pump()
	ts.Pump(10 * time.Millisecond)
	if mid := heroHeight(ts, homeHeroKey); mid >= finalHeight || mid <= 100 {
		t.Fatalf("mid flight height = %v", mid)
	}
	// Removes the destination hero midflight.
	heroCardSetState(func() { routeIncludesHero = false })
	ts.Pump()
	ts.Pump(duration)
	if got := heroHeight(ts, homeHeroKey); got != 100 {
		t.Errorf("final height = %v, want 100", got)
	}
}

func heroScrollApp(ts *gluttertest.Tester, observer NavigatorObserver, routeChildren []Widget, routeOpts ListViewOpts, homeChildren func(push func()) []Widget) {
	route := heroPageRoute(func(BuildContext) Widget { return NewListView(routeChildren, routeOpts) })
	ts.PumpWidget(navTestApp(navTestAppOpts{observers: []NavigatorObserver{observer}, home: NewBuilder(func(bc BuildContext) Widget {
		return NewListView(homeChildren(func() { NavigatorPush(bc, route) }))
	})}))
}

func TestDestinationHeroScrollsMidFlight(t *testing.T) {
	ts := gluttertest.New(t)
	homeHeroKey, routeHeroKey, routeContainerKey := foundation.NewKey("home hero"), foundation.NewKey("route hero"), foundation.NewKey("route hero container")
	observer := &transitionDurationObserver{}
	heroScrollApp(ts, observer, []Widget{
		heroSized(nil, 0, 100),
		// This container appears at Y=100.
		NewContainer(NewHero("H", heroSized(routeHeroKey, 200, 200)), ContainerOpts{Key: routeContainerKey}),
		heroButton("POP", func() {}),
		heroSized(nil, 0, 600),
	}, ListViewOpts{}, func(push func()) []Widget {
		return []Widget{
			heroSized(nil, 0, 200),
			// This container appears at Y=200.
			NewHero("H", heroSized(homeHeroKey, 100, 100)),
			heroButton("PUSH", push),
			heroSized(nil, 0, 600),
		}
	})
	ts.Tap(gluttertest.ByText("PUSH"))
	ts.Pump()
	ts.Pump()
	if got := ts.TopLeft(heroByKey(ts, routeHeroKey)).Dy; got != 200 {
		t.Fatalf("initial Y = %v, want 200", got)
	}
	ts.Pump(100 * time.Millisecond)
	yAt100 := ts.TopLeft(heroByKey(ts, routeHeroKey)).Dy
	if yAt100 >= 200 || yAt100 <= 100 {
		t.Fatalf("Y at 100ms = %v", yAt100)
	}
	// Scrolls the target upwards by 25 pixels: the Y coordinate of the flight is
	// redirected from 100 to 75.
	ts.Drag(gluttertest.ByKey(routeContainerKey), ui.Offset{Dy: -25})
	ts.Pump()
	ts.Pump(10 * time.Millisecond)
	yAt110 := ts.TopLeft(heroByKey(ts, routeHeroKey)).Dy
	if yAt110 >= yAt100 || yAt110 <= 75 {
		t.Fatalf("Y at 110ms = %v", yAt110)
	}
	ts.Pump(observer.duration.OrElse(0))
	ts.Pump()
	if got := ts.TopLeft(heroByKey(ts, routeHeroKey)).Dy; got != 75 {
		t.Errorf("final Y = %v, want 75 (100 less 25 for the scroll)", got)
	}
}

func TestDestinationHeroScrollsOutOfViewMidFlight(t *testing.T) {
	ts := gluttertest.New(t)
	homeHeroKey, routeHeroKey, routeContainerKey := foundation.NewKey("home hero"), foundation.NewKey("route hero"), foundation.NewKey("route hero container")
	observer := &transitionDurationObserver{}
	heroScrollApp(ts, observer, []Widget{
		heroSized(nil, 0, 100),
		// This container appears at Y=100.
		NewContainer(NewHero("H", heroSized(routeHeroKey, 200, 200)), ContainerOpts{Key: routeContainerKey}),
		heroSized(nil, 0, 800),
	}, ListViewOpts{ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(0))}, func(push func()) []Widget {
		return []Widget{
			heroSized(nil, 0, 200),
			NewHero("H", heroSized(homeHeroKey, 100, 100)),
			heroButton("PUSH", push),
		}
	})
	ts.Tap(gluttertest.ByText("PUSH"))
	ts.Pump()
	ts.Pump()
	if got := ts.TopLeft(heroByKey(ts, routeHeroKey)).Dy; got != 200 {
		t.Fatalf("initial Y = %v, want 200", got)
	}
	ts.Pump(100 * time.Millisecond)
	yAt100 := ts.TopLeft(heroByKey(ts, routeHeroKey)).Dy
	if yAt100 >= 200 || yAt100 <= 100 {
		t.Fatalf("Y at 100ms = %v", yAt100)
	}
	ts.Drag(gluttertest.ByKey(routeContainerKey), ui.Offset{Dy: -400})
	ts.Pump()
	ts.Pump(10 * time.Millisecond)
	if heroExists(ts, routeContainerKey) {
		t.Fatal("the container should have scrolled off the top")
	}
	// The flight continues (the hero fades out) though the destination no longer
	// exists.
	yAt110 := ts.TopLeft(heroByKey(ts, routeHeroKey)).Dy
	if yAt110 >= yAt100 || yAt110 <= 100 {
		t.Fatalf("Y at 110ms = %v", yAt110)
	}
	ts.Pump(observer.duration.OrElse(0))
	if heroExists(ts, routeHeroKey) {
		t.Error("the flight should be over")
	}
}

func TestHeroAbortedFlight(t *testing.T) {
	// See https://github.com/flutter/flutter/issues/5798.
	ts := gluttertest.New(t)
	heroABKey, heroBCKey := foundation.NewKey("AB hero"), foundation.NewKey("BC hero")
	observer := &transitionDurationObserver{}
	hero := func(tag string, key foundation.Key, height float64) Widget {
		return NewHero(tag, NewSizedBox(NewText("Hero"), SizedBoxOpts{Key: key, Height: foundation.Some(height)}))
	}
	routeC := heroPageRoute(func(BuildContext) Widget {
		return NewListView([]Widget{hero("BC", heroBCKey, 150), heroSized(nil, 0, 800)})
	})
	routeB := heroPageRoute(func(BuildContext) Widget {
		return NewListView([]Widget{
			heroSized(nil, 0, 100),
			// This container appears at Y=100.
			hero("AB", heroABKey, 200),
			NewBuilder(func(bc BuildContext) Widget {
				return heroButton("PUSH C", func() { NavigatorPush(bc, routeC) })
			}),
			hero("BC", nil, 150),
			heroSized(nil, 0, 800),
		})
	})
	ts.PumpWidget(navTestApp(navTestAppOpts{observers: []NavigatorObserver{observer}, home: NewBuilder(func(bc BuildContext) Widget {
		return NewListView([]Widget{
			heroSized(nil, 0, 200),
			// This container appears at Y=200.
			NewHero("AB", NewSizedBox(NewText("Hero"), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)})),
			heroButton("PUSH B", func() { NavigatorPush(bc, routeB) }),
		})
	})}))
	ts.Tap(gluttertest.ByText("PUSH B"))
	ts.Pump()
	ts.Pump()
	if got := ts.TopLeft(heroByKey(ts, heroABKey)).Dy; got != 200 {
		t.Fatalf("initial Y = %v, want 200", got)
	}
	duration := observer.duration.OrElse(0)
	ts.Pump(duration * 2 / 3)
	// Hero AB is mid flight.
	if y := ts.TopLeft(heroByKey(ts, heroABKey)).Dy; y >= 200 || y <= 100 {
		t.Fatalf("Y at 200ms = %v", y)
	}
	// Pushes route C: the flight of hero AB aborts and the one of hero BC starts.
	ts.Tap(gluttertest.ByText("PUSH C"))
	ts.Pump()
	ts.Pump()
	// The aborted flight of hero AB finishes where expected, though faded out.
	ts.Pump(duration / 3)
	if y := ts.TopLeft(heroByKey(ts, heroABKey)).Dy; !approx(y, 100, 0.1) {
		t.Errorf("Y of hero AB = %v, want 100", y)
	}
	isVisible := func(node rendering.RenderObject) bool {
		for current := node; current != nil; current = current.Parent() {
			if o, ok := current.(*rendering.RenderAnimatedOpacity); ok && o.Opacity().Value() == 0 {
				return false
			}
		}
		return true
	}
	// Of all heroes, only one should be visible now.
	elements := ts.Elements(gluttertest.ByText("Hero"))
	ts.Pump(time.Millisecond)
	visible := 0
	for _, e := range elements {
		// The element of the aborted flight is gone by now.
		if ro := e.RenderObject(); ro != nil && isVisible(ro) {
			visible++
		}
	}
	if visible != 1 {
		t.Errorf("%d heroes are visible, want 1", visible)
	}
	// The flight of hero BC finishes normally.
	ts.Pump(duration)
	if y := ts.TopLeft(heroByKey(ts, heroBCKey)).Dy; y != 0 {
		t.Errorf("Y of hero BC = %v, want 0", y)
	}
}

// heroSimpleState is _SimpleState.
type heroSimpleWidget struct{ StatefulWidget }

func (w *heroSimpleWidget) CreateState() State { return &heroSimpleState{} }

type heroSimpleState struct {
	StateOf[*heroSimpleWidget]
	state int
}

func (s *heroSimpleState) Build(BuildContext) Widget { return NewText(fmt.Sprint(s.state)) }

type heroValueWidget struct {
	StatefulWidget
	value string
}

func (w *heroValueWidget) CreateState() State { return &heroValueState{} }

type heroValueState struct{ StateOf[*heroValueWidget] }

func (s *heroValueState) Build(BuildContext) Widget { return NewText(s.Widget().value) }

func TestStatefulHeroChildStateSurvivesFlight(t *testing.T) {
	ts := gluttertest.New(t)
	observer := &transitionDurationObserver{}
	stateful := func() Widget { return &heroValueWidget{NewStatefulWidgetBase(WidgetOpts{}), "456"} }
	route := heroPageRoute(func(BuildContext) Widget {
		return NewListView([]Widget{
			newHeroTestCard(NewHero("H", NewSizedBox(stateful(), SizedBoxOpts{Height: foundation.Some(200.0)}))),
			NewBuilder(func(bc BuildContext) Widget { return heroButton("POP", func() { NavigatorPop(bc) }) }),
		})
	})
	ts.PumpWidget(navTestApp(navTestAppOpts{observers: []NavigatorObserver{observer}, home: NewBuilder(func(bc BuildContext) Widget {
		return NewListView([]Widget{
			newHeroTestCard(NewHero("H", NewSizedBox(stateful(), SizedBoxOpts{Height: foundation.Some(100.0)}))),
			heroButton("PUSH", func() { NavigatorPush(bc, route) }),
		})
	})}))
	one := func(what string) {
		t.Helper()
		if got := ts.Count(gluttertest.ByText("456")); got != 1 {
			t.Fatalf("%s: %d texts of 456, want 1", what, got)
		}
	}
	one("start")
	ts.Tap(gluttertest.ByText("PUSH"))
	ts.Pump()
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	one("push flight underway")
	ts.Pump(observer.duration.OrElse(0))
	one("push flight finished")
	ts.Tap(gluttertest.ByText("POP"))
	ts.Pump()
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	one("pop flight underway")
	ts.Pump(observer.duration.OrElse(0))
	one("pop flight finished")
}

func TestHeroFromHeroStateIsPreservedWithGlobalKeys(t *testing.T) {
	ts := gluttertest.New(t)
	navigatorKey := NewGlobalKey[*NavigatorState]()
	key1 := NewGlobalKey[*heroSimpleState]()
	key2 := NewGlobalKey[*heroSimpleState]()
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: navigatorKey, home: NewRow([]Widget{
		NewHero("hero", &heroSimpleWidget{NewStatefulWidgetBase(WidgetOpts{Key: key1})}, HeroOpts{TransitionOnUserGestures: true}),
		NewSizedBox(NewText("1"), SizedBoxOpts{Width: foundation.Some(10.0), Height: foundation.Some(10.0)}),
	}, RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})}))
	state1, _ := key1.CurrentState()
	state1.state = 1
	// key2 is a GlobalKey: the flight must not put the same keyed widget in more
	// than one place in the tree.
	navPush(navigatorKey, heroPageRoute(func(BuildContext) Widget {
		return NewHero("hero", &heroSimpleWidget{NewStatefulWidgetBase(WidgetOpts{Key: key2})}, HeroOpts{TransitionOnUserGestures: true})
	}))
	ts.Pump()
	if !state1.Mounted() {
		t.Fatal("state1 should be mounted")
	}
	ts.PumpAndSettle()
	if state1.state != 1 || !state1.Mounted() {
		t.Fatalf("state1 = %d, mounted %v", state1.state, state1.Mounted())
	}
	navState(t, navigatorKey).Pop()
	ts.PumpAndSettle()
	if state1.state != 1 || !state1.Mounted() {
		t.Errorf("state1 = %d, mounted %v after the pop", state1.state, state1.Mounted())
	}
	if err := ts.TakeError(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHeroInAPopTransitionWhenFromHeroIsNull(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/40239.
	ts := gluttertest.New(t)
	navigatorKey := NewGlobalKey[*NavigatorState]()
	var setState StateSetter
	shouldDisplayHero := true
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: navigatorKey, home: NewHero("tag", NewPlaceholder(nil))}))
	navPush(navigatorKey, heroPageRoute(func(BuildContext) Widget {
		return NewStatefulBuilder(func(bc BuildContext, setter StateSetter) Widget {
			setState = setter
			if shouldDisplayHero {
				return NewHero("tag", NewText("text"))
			}
			return NewSizedBoxShrink(nil)
		})
	}))
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByText("text")) != 1 || ts.Count(gluttertest.ByType[*Placeholder]()) != 0 {
		t.Fatal("the hero of the second route should show")
	}
	setState(func() { shouldDisplayHero = false })
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByText("text")) != 0 {
		t.Fatal("the hero should be gone")
	}
	navState(t, navigatorKey).Pop()
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByType[*Placeholder]()) != 1 {
		t.Error("the hero of the first route should be visible again")
	}
}

func heroPoppedCurve(t *testing.T, curve, reverseCurve animation.Curve) {
	t.Helper()
	ts := gluttertest.New(t)
	container1, container2 := foundation.NewUniqueKey(), foundation.NewUniqueKey()
	navigator := NewGlobalKey[*NavigatorState]()
	observer := &transitionDurationObserver{}
	opts := func() HeroOpts {
		o := HeroOpts{CreateRectTween: heroRectTweenOf, Curve: curve}
		if reverseCurve != nil {
			o.ReverseCurve = reverseCurve.Flipped()
		}
		return o
	}
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: navigator, observers: []NavigatorObserver{observer},
		home: NewCenter(NewHero("test", heroSized(container1, 100, 100), opts()))}))
	originalSize := ts.Size(heroByKey(ts, container1))
	if originalSize != (ui.Size{Width: 100, Height: 100}) {
		t.Fatalf("original size = %v", originalSize)
	}
	navPush(navigator, heroPageRoute(func(BuildContext) Widget {
		return NewCenter(NewHero("test", heroSized(container2, 200, 200), opts()))
	}))
	ts.PumpAndSettle()
	newSize := ts.Size(heroByKey(ts, container2))
	if newSize != (ui.Size{Width: 200, Height: 200}) {
		t.Fatalf("new size = %v", newSize)
	}
	navState(t, navigator).Pop()
	ts.Pump()

	duration := observer.duration.OrElse(0)
	expectedCurve := curve
	if reverseCurve != nil {
		expectedCurve = reverseCurve
	}
	for _, fraction := range []float64{0.25, 0.5, 0.75, 1} {
		ts.Pump(duration / 4)
		want := expectedCurve.Transform(fraction)
		wantSize := ui.Size{
			Width:  newSize.Width + (originalSize.Width-newSize.Width)*want,
			Height: newSize.Height + (originalSize.Height-newSize.Height)*want,
		}
		got := ts.Size(heroByKey(ts, container1))
		if !approx(got.Width, wantSize.Width, 0.001) || !approx(got.Height, wantSize.Height, 0.001) {
			t.Errorf("curve=%v, t=%v: size = %v, want %v", expectedCurve, fraction, got, wantSize)
		}
	}
}

func TestDefaultPoppedHeroUsesFastOutSlowIn(t *testing.T) {
	ts := gluttertest.New(t)
	container1, container2 := foundation.NewUniqueKey(), foundation.NewUniqueKey()
	navigator := NewGlobalKey[*NavigatorState]()
	observer := &transitionDurationObserver{}
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: navigator, observers: []NavigatorObserver{observer},
		home: NewCenter(NewHero("test", heroSized(container1, 100, 100), HeroOpts{CreateRectTween: heroRectTweenOf}))}))
	navPush(navigator, heroPageRoute(func(BuildContext) Widget {
		return NewCenter(NewHero("test", heroSized(container2, 200, 200), HeroOpts{CreateRectTween: heroRectTweenOf}))
	}))
	ts.PumpAndSettle()
	navState(t, navigator).Pop()
	ts.Pump()
	duration := observer.duration.OrElse(0)
	for _, fraction := range []float64{0.25, 0.5, 0.75, 1} {
		ts.Pump(duration / 4)
		want := 200 - 100*animation.Curves.FastOutSlowIn.Transform(fraction)
		if got := ts.Size(heroByKey(ts, container1)); !approx(got.Width, want, 1e-9) || !approx(got.Height, want, 1e-9) {
			t.Errorf("size at %v = %v, want %v", fraction, got, want)
		}
	}
}

func TestPoppedHeroCurve(t *testing.T) {
	curves := []animation.Curve{
		animation.Curves.Linear, animation.Curves.Ease, animation.Curves.LinearToEaseOut,
		animation.Curves.EaseInOutCubicEmphasized, animation.Curves.FastOutSlowIn,
	}
	for _, reverseCurve := range curves {
		t.Run(fmt.Sprintf("customizable with reverseCurve=%v", reverseCurve), func(t *testing.T) {
			heroPoppedCurve(t, animation.Curves.Linear, reverseCurve)
		})
	}
	for _, curve := range curves {
		t.Run(fmt.Sprintf("uses the flipped curve=%v when reverseCurve is not specified", curve), func(t *testing.T) {
			heroPoppedCurve(t, curve, nil)
		})
	}
}

func TestHeroMode(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("enabled=%v", enabled), func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(heroApp(NewColumn([]Widget{
				NewHeroMode(newHeroTestCard(NewHero("a", heroSized(heroFirstKey, 100, 100))), HeroModeOpts{Enabled: foundation.Some(enabled)}),
				NewBuilder(func(bc BuildContext) Widget {
					return heroButton("push", func() {
						NavigatorPush(bc, NewPageRouteBuilder(routesPageBuilder(func(BuildContext) Widget {
							return newHeroTestCard(NewHero("a", heroSized(heroSecondKey, 150, 150)))
						})))
					})
				}),
			})))
			if !heroIsOnstage(ts, heroFirstKey) || !heroIsInCard(ts, heroFirstKey) || heroExists(ts, heroSecondKey) {
				t.Fatal("unexpected initial state")
			}
			ts.Tap(gluttertest.ByText("push"))
			ts.Pump()
			if !heroIsOnstage(ts, heroFirstKey) || !heroIsInCard(ts, heroFirstKey) || !heroIsOffstage(ts, heroSecondKey) || !heroIsInCard(ts, heroSecondKey, true) {
				t.Fatal("the second route should exist offstage")
			}
			ts.Pump()
			if enabled {
				if heroExists(ts, heroFirstKey) || !heroIsOnstage(ts, heroSecondKey) || heroIsInCard(ts, heroSecondKey) {
					t.Error("the hero should be flying")
				}
			} else {
				// When HeroMode is disabled, heroes do not move: the original page
				// keeps its hero.
				if !heroIsOnstage(ts, heroFirstKey) || !heroIsOnstage(ts, heroSecondKey) || !heroIsInCard(ts, heroSecondKey) {
					t.Error("the heroes should stay in their cards")
				}
			}
			ts.Pump(time.Second)
			if heroExists(ts, heroFirstKey) || !heroIsOnstage(ts, heroSecondKey) || !heroIsInCard(ts, heroSecondKey) {
				t.Error("the second hero should be in its card at the end")
			}
		})
	}
}

func TestKeptAliveHeroDoesNotThrowWhenTheTransitionBegins(t *testing.T) {
	ts := gluttertest.New(t)
	navigatorKey := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: navigatorKey, home: NewListView([]Widget{
		NewKeepAlive(true, NewHero("a", NewPlaceholder(nil))),
		NewContainer(nil, ContainerOpts{Height: foundation.Some(1000.0)}),
	}, ListViewOpts{
		AddAutomaticKeepAlives: foundation.Some(false), AddRepaintBoundaries: foundation.Some(false), AddSemanticIndexes: foundation.Some(false),
	})}))
	// Scrolls to make the hero invisible.
	ts.Drag(gluttertest.ByType[*ListView](), ui.Offset{Dy: -1000})
	ts.Pump()
	navPush(navigatorKey, heroPageRoute(func(BuildContext) Widget { return NewCenter(NewHero("a", NewPlaceholder(nil))) }))
	ts.PumpAndSettle()
	if err := ts.TakeError(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The hero on the new route should be visible.
	if got := ts.Count(gluttertest.ByType[*Placeholder]()); got != 1 {
		t.Errorf("%d placeholders, want 1", got)
	}
}

func TestToHeroBecomesUnpaintableAfterTheTransitionBegins(t *testing.T) {
	ts := gluttertest.New(t)
	navigatorKey := NewGlobalKey[*NavigatorState]()
	controller := NewScrollController()
	defer controller.Dispose()
	findOpacity := func() *rendering.RenderAnimatedOpacity {
		for p := ts.RenderObject(gluttertest.ByType[*Placeholder]()); p != nil; p = p.Parent() {
			if o, ok := p.(*rendering.RenderAnimatedOpacity); ok {
				return o
			}
		}
		return nil
	}
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: navigatorKey, home: NewListView([]Widget{
		NewKeepAlive(true, NewHero("a", NewPlaceholder(nil))),
		NewContainer(nil, ContainerOpts{Height: foundation.Some(1000.0)}),
	}, ListViewOpts{
		Controller:             controller,
		AddAutomaticKeepAlives: foundation.Some(false), AddRepaintBoundaries: foundation.Some(false), AddSemanticIndexes: foundation.Some(false),
	})}))
	navPush(navigatorKey, heroPageRoute(func(BuildContext) Widget { return NewCenter(NewHero("a", NewPlaceholder(nil))) }))
	ts.Pump()
	ts.PumpAndSettle()
	// Pops the new route and, before the animation finishes, scrolls the toHero
	// to make it unpaintable.
	navState(t, navigatorKey).Pop()
	ts.Pump()
	controller.JumpTo(1000)
	// Starts the hero animation and the scroll animation almost simultaneously.
	ts.Pump()
	if o := findOpacity(); o != nil && o.Opacity().Value() != 1 {
		t.Fatalf("opacity = %v, want none or 1", o.Opacity().Value())
	}
	// In this frame the hero animation finds out the toHero is not paintable and
	// starts fading.
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	if o := findOpacity(); o == nil || o.Opacity().Value() >= 1 {
		t.Fatal("the hero should be fading")
	}
	ts.PumpAndSettle()
	// The hero on the new route should be invisible.
	if got := ts.Count(gluttertest.ByType[*Placeholder]()); got != 0 {
		t.Errorf("%d placeholders, want 0", got)
	}
}

func TestDivertingToAKeepAliveButUnpaintableHero(t *testing.T) {
	ts := gluttertest.New(t)
	navigatorKey := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: navigatorKey, home: NewListView([]Widget{
		NewKeepAlive(true, NewHero("a", NewPlaceholder(nil))),
		NewContainer(nil, ContainerOpts{Height: foundation.Some(1000.0)}),
	}, ListViewOpts{
		AddAutomaticKeepAlives: foundation.Some(false), AddRepaintBoundaries: foundation.Some(false), AddSemanticIndexes: foundation.Some(false),
	})}))
	// Scrolls to make the hero invisible.
	ts.Drag(gluttertest.ByType[*ListView](), ui.Offset{Dy: -1000})
	ts.Pump()
	if ts.Count(gluttertest.ByType[*Placeholder]()) != 0 || ts.Count(gluttertest.ByType[*Placeholder](gluttertest.FindOpts{IncludeOffstage: true})) != 1 {
		t.Fatal("the hero should be kept alive but not shown")
	}
	page := func(BuildContext) Widget { return NewCenter(NewHero("a", NewPlaceholder(nil))) }
	navPush(navigatorKey, heroPageRoute(page))
	ts.PumpAndSettle()
	// Yet another route that contains hero 'a'.
	navPush(navigatorKey, heroPageRoute(page))
	ts.PumpAndSettle()
	// Pops both routes.
	navState(t, navigatorKey).Pop()
	ts.Pump()
	ts.Pump(10 * time.Millisecond)
	navState(t, navigatorKey).Pop()
	ts.Pump()
	ts.Pump(10 * time.Millisecond)
	if got := ts.Count(gluttertest.ByType[*Placeholder]()); got != 1 {
		t.Errorf("%d placeholders, want 1", got)
	}
	ts.PumpAndSettle()
	if err := ts.TakeError(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHeroDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewSizedBoxShrink(NewHero("imageHero", NewText("X"))))))
	if got := ts.Size(gluttertest.ByType[*Hero]()); got != (ui.Size{}) {
		t.Errorf("size = %v, want zero", got)
	}
}

// heroGestureStart emulates the start of a back swipe on the current route:
// what the iOS back gesture does to the route and the navigator.
func heroGestureStart(t *testing.T, ts *gluttertest.Tester, finder any, progress float64) TransitionRoute {
	t.Helper()
	var f gluttertest.Finder
	switch v := finder.(type) {
	case foundation.Key:
		f = heroByKey(ts, v)
	case gluttertest.Finder:
		f = v
	}
	route, ok := ModalRouteOf(ts.Element(f)).(TransitionRoute)
	if !ok {
		t.Fatal("the route is not a TransitionRoute")
	}
	route.HandleStartBackGesture(progress)
	return route
}

func TestHeroesDoNotTransitionOnBackGesturesByDefault(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navRoutesApp(heroRoutes(false)))
	ts.Tap(gluttertest.ByText("two"))
	ts.Pump()
	ts.Pump(501 * time.Millisecond)
	if heroExists(ts, heroFirstKey) || !heroIsOnstage(ts, heroSecondKey) || !heroIsInCard(ts, heroSecondKey) {
		t.Fatal("the second route should show")
	}
	route := heroGestureStart(t, ts, heroSecondKey, 1)
	route.HandleUpdateBackGestureProgress(0.9)
	route.HandleUpdateBackGestureProgress(0.75)
	route.HandleCommitBackGesture()
	ts.Pump()
	ts.Pump()
	// Both heroes exist and are seated in their normal parents.
	for i := 0; i < 2; i++ {
		if !heroIsOnstage(ts, heroFirstKey) || !heroIsInCard(ts, heroFirstKey) || !heroIsOnstage(ts, heroSecondKey) || !heroIsInCard(ts, heroSecondKey) {
			t.Fatalf("the heroes should be in their cards (%d)", i)
		}
		// To make sure the hero had all chances of starting.
		ts.Pump(100 * time.Millisecond)
	}
}

func TestHeroesCanTransitionOnGestureInOneFrame(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navRoutesApp(heroRoutes(true)))
	ts.Tap(gluttertest.ByText("two"))
	ts.Pump()
	ts.Pump(501 * time.Millisecond)
	if heroExists(ts, heroFirstKey) || !heroIsOnstage(ts, heroSecondKey) || !heroIsInCard(ts, heroSecondKey) {
		t.Fatal("the second route should show")
	}
	route := heroGestureStart(t, ts, heroSecondKey, 1)
	route.HandleUpdateBackGestureProgress(0.75)
	ts.Pump()
	// Going to page 1, so the hero of page 1 is lifted into flight.
	if !heroIsOnstage(ts, heroFirstKey) || heroIsInCard(ts, heroFirstKey) || heroExists(ts, heroSecondKey) {
		t.Fatal("the first hero should be in flight")
	}
	// Moves further along.
	route.HandleUpdateBackGestureProgress(0)
	ts.Pump()
	if !heroIsOnstage(ts, heroFirstKey) || heroIsInCard(ts, heroFirstKey) || heroExists(ts, heroSecondKey) {
		t.Fatal("the first hero should still be in flight")
	}
	route.HandleCommitBackGesture()
	// Finishes the transition.
	ts.Pump()
	ts.Pump(500 * time.Millisecond)
	if !heroIsOnstage(ts, heroFirstKey) || !heroIsInCard(ts, heroFirstKey) || heroExists(ts, heroSecondKey) {
		t.Error("the first hero should be back in the card")
	}
}

func TestHeroesAnimateHideDestinationAndShowOriginalWhenDismissed(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navRoutesApp(heroRoutes(true)))
	ts.Tap(gluttertest.ByText("two"))
	ts.PumpAndSettle()
	if heroExists(ts, heroFirstKey) || !heroIsOnstage(ts, heroSecondKey) || !heroIsInCard(ts, heroSecondKey) {
		t.Fatal("the second route should show")
	}
	route := heroGestureStart(t, ts, heroSecondKey, 1)
	route.HandleUpdateBackGestureProgress(0.95)
	ts.Pump()
	route.HandleUpdateBackGestureProgress(0.9)
	ts.Pump()
	if !heroIsOnstage(ts, heroFirstKey) || heroIsInCard(ts, heroFirstKey) || heroExists(ts, heroSecondKey) {
		t.Fatal("the first hero should be in flight")
	}
	// Dismisses the hero transition.
	route.HandleCancelBackGesture()
	ts.Pump()
	ts.PumpAndSettle()
	if heroExists(ts, heroFirstKey) || !heroIsOnstage(ts, heroSecondKey) || !heroIsInCard(ts, heroSecondKey) {
		t.Error("we should be back at the second page")
	}
}

func TestOnABackSwipeAndSnapOnlyASingleFlightTakesPlace(t *testing.T) {
	ts := gluttertest.New(t)
	shuttlesBuilt := 0
	navigatorKey := NewGlobalKey[*NavigatorState]()
	tag := "swipe"
	ts.PumpWidget(navTestApp(navTestAppOpts{navigatorKey: navigatorKey, home: NewHero(tag, NewText("1"), HeroOpts{
		// Since the route pops, only the builder of the destination route is used.
		FlightShuttleBuilder: func(BuildContext, animation.Animation[float64], HeroFlightDirection, BuildContext, BuildContext) Widget {
			shuttlesBuilt++
			return NewText("I'm flying in a jetplane")
		},
		TransitionOnUserGestures: true,
	})}))
	navPush(navigatorKey, heroPageRoute(func(BuildContext) Widget {
		return NewHero(tag, NewText("2"), HeroOpts{TransitionOnUserGestures: true})
	}))
	ts.PumpAndSettle()
	if shuttlesBuilt != 1 {
		t.Fatalf("shuttles built = %d, want 1", shuttlesBuilt)
	}
	route := heroGestureStart(t, ts, gluttertest.ByText("2"), 1)
	route.HandleUpdateBackGestureProgress(0.3)
	ts.Pump()
	// Starting the back swipe creates a new hero shuttle.
	if shuttlesBuilt != 2 {
		t.Fatalf("shuttles built = %d, want 2", shuttlesBuilt)
	}
	route.HandleCommitBackGesture()
	ts.Pump()
	// After the lift, no additional shuttles are created: it is the same flight.
	if shuttlesBuilt != 2 {
		t.Fatalf("shuttles built = %d, want 2", shuttlesBuilt)
	}
	// Went far enough to snap out of this route.
	ts.Pump(301 * time.Millisecond)
	if ts.Count(gluttertest.ByText("2")) != 0 {
		t.Error("the second route should be gone")
	}
	if shuttlesBuilt != 2 {
		t.Errorf("shuttles built = %d, want 2", shuttlesBuilt)
	}
}
