package widgets_test

import (
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from the 'Page api' group of
// packages/flutter/test/widgets/navigator_test.dart. MaterialPage is a page
// whose route is a page route of 300 ms (a MaterialPageRoute without the
// transition); the cases of the group 'Navigator page API' that belong to the
// Android predictive back gesture are not ported.

// testPage is TestPage, BuilderPage, ZeroTransitionPage and MaterialPage: a page
// that creates its route with a function.
type testPage struct {
	PageBase
	create func(p *testPage) Route
}

func newPage(key, name string, create func(p *testPage) Route) *testPage {
	p := &testPage{create: create}
	var k foundation.Key
	if key != "" {
		k = foundation.NewKey(key)
	}
	InitPageBase(&p.PageBase, p, PageOpts{Key: k, Name: name})
	return p
}

func (p *testPage) CreateRoute(BuildContext) Route { return p.create(p) }

// newTestPage is TestPage: a page that shows its name.
func newTestPage(key, name string) *testPage {
	return newPage(key, name, func(p *testPage) Route {
		return heroPageRoute(func(BuildContext) Widget { return NewText(name) }, p)
	})
}

// newBuilderPage is BuilderPage.
func newBuilderPage(key, name string, pageBuilder RoutePageBuilder) *testPage {
	return newPage(key, name, func(p *testPage) Route {
		return NewPageRouteBuilder(pageBuilder, PageRouteBuilderOpts{PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: p}}})
	})
}

// newMaterialPage is MaterialPage.
func newMaterialPage(key string, child Widget) *testPage {
	return newPage(key, "", func(p *testPage) Route {
		return heroPageRoute(func(BuildContext) Widget { return child }, p)
	})
}

func pagesOf(pages []*testPage) []Page {
	out := make([]Page, len(pages))
	for i, p := range pages {
		out[i] = p
	}
	return out
}

type pagesAppOpts struct {
	key       *GlobalKey[*NavigatorState]
	pages     []Page
	onPopPage PopPageCallback
	onRemoved DidRemovePageCallback
	delegate  TransitionDelegate
	observers []NavigatorObserver
}

func pagesApp(o pagesAppOpts) Widget {
	return NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
		Key:                navigatorKeyOf(o.key),
		Pages:              o.pages,
		OnPopPage:          o.onPopPage,
		OnDidRemovePage:    o.onRemoved,
		TransitionDelegate: o.delegate,
		Observers:          o.observers,
	}))
}

// pageList is the myPages of the tests: pages that onPopPage removes.
type pageList struct{ pages []*testPage }

func (l *pageList) removeRoute(route Route) {
	l.pages = slices.DeleteFunc(l.pages, func(p *testPage) bool { return RouteSettings(p) == route.Settings() })
}

func (l *pageList) onPopPage(route Route, result any) bool {
	l.removeRoute(route)
	return route.DidPop(result)
}

func TestPageAPICanInitializeWithPagesList(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	myPages := &pageList{[]*testPage{newTestPage("1", "initial"), newTestPage("2", "second"), newTestPage("3", "third")}}
	ts.PumpWidget(pagesApp(pagesAppOpts{key: key, pages: pagesOf(myPages.pages), onPopPage: myPages.onPopPage}))
	shows := func(name string) {
		t.Helper()
		for _, n := range []string{"third", "second", "initial"} {
			if got, want := textsFound(ts, n, false), map[bool]int{true: 1, false: 0}[n == name]; got != want {
				t.Fatalf("%q: %d found, want %d", n, got, want)
			}
		}
	}
	shows("third")
	navState(t, key).Pop()
	ts.PumpAndSettle()
	shows("second")
	navState(t, key).Pop()
	ts.PumpAndSettle()
	shows("initial")
}

func TestPageAPICanHandleDuplicatePageKeyIfUpdateBeforeTransitionFinishes(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/97363.
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	onPopPage := func(Route, any) bool { return false }
	build := func(pages ...*testPage) Widget {
		return pagesApp(pagesAppOpts{key: key, pages: pagesOf(pages), onPopPage: onPopPage})
	}
	ts.PumpWidget(build(newTestPage("1", "initial")))
	ts.Pump()
	if textsFound(ts, "initial", false) != 1 {
		t.Fatal("initial should show")
	}
	// Updates several times without waiting for the pop to finish, to leave
	// several popping route entries in the stack with the same page key.
	ts.PumpWidget(build(newTestPage("2", "second")))
	ts.Pump()
	ts.PumpWidget(build(newTestPage("1", "initial")))
	ts.Pump()
	ts.PumpWidget(build(newTestPage("2", "second")))
	ts.PumpAndSettle()
	if textsFound(ts, "second", false) != 1 {
		t.Error("second should show")
	}
	if err := ts.TakeError(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPageAPIThrowsIfNoPopCallbackIsProvided(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(pagesApp(pagesAppOpts{pages: pagesOf([]*testPage{newTestPage("1", "initial")})}))
	_ = requireErrorTag(t, takeFirst(t, ts), "NavigatorPageCallbacksInvalid")
	ts.TakeErrors()
}

func TestPageAPIThrowsIfPageListIsEmpty(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	// Only the first error matters: an empty list also leaves the navigator
	// without routes.
	ts.PumpWidget(pagesApp(pagesAppOpts{pages: []Page{}, onPopPage: func(Route, any) bool { return false }}))
	_ = requireErrorTag(t, takeFirst(t, ts), "NavigatorPagesEmpty")
	ts.TakeErrors()
}

func TestPageAPICanPopRouteWithLocalHistoryEntries(t *testing.T) {
	ts := gluttertest.New(t)
	pages := func(children ...string) []Page {
		var out []Page
		for _, c := range children {
			out = append(out, newMaterialPage(c, NewText(c)))
		}
		return out
	}
	app := func(p []Page) Widget {
		return pagesApp(pagesAppOpts{pages: p, onPopPage: func(Route, any) bool { return false }})
	}
	ts.PumpWidget(app(pages("page1", "page2")))
	if textsFound(ts, "page2", false) != 1 {
		t.Fatal("page2 should show")
	}
	route := ModalRouteOf(ts.Element(gluttertest.ByText("page2")))
	entryRemoved := false
	route.AddLocalHistoryEntry(NewLocalHistoryEntry(LocalHistoryEntryOpts{OnRemove: func() { entryRemoved = true }}))
	if !route.WillHandlePopInternally() {
		t.Fatal("the route should handle pops internally")
	}
	ts.PumpWidget(app(pages("page1")))
	if textsFound(ts, "page1", false) != 1 || !entryRemoved {
		t.Errorf("page1 shows: %v, entry removed: %v", textsFound(ts, "page1", false) == 1, entryRemoved)
	}
}

func TestPageAPIModalRouteComplyWithWillHandlePopInternallyWithAPopScope(t *testing.T) {
	ts := gluttertest.New(t)
	pages := []Page{
		newMaterialPage("1", NewText("page1")),
		newMaterialPage("2", NewPopScope(NewText("page2"), PopScopeOpts{CanPop: foundation.Some(false)})),
	}
	ts.PumpWidget(pagesApp(pagesAppOpts{pages: pages, onPopPage: func(Route, any) bool { return false }}))
	route := ModalRouteOf(ts.Element(gluttertest.ByText("page2")))
	// PopScope only prevents what the user triggers, such as Navigator.MaybePop;
	// the page can still be popped by the system.
	if route.WillHandlePopInternally() {
		t.Error("the route should not handle pops internally")
	}
	if !route.DidPop(nil) {
		t.Error("the route should be popped")
	}
}

// pageAnimations records the animations that a BuilderPage got.
type pageAnimations struct {
	secondary, primary animation.Animation[float64]
}

func builderPageRecording(key, name string, rec *pageAnimations) *testPage {
	return newBuilderPage(key, name, func(_ BuildContext, a, s animation.Animation[float64]) Widget {
		rec.secondary, rec.primary = s, a
		return NewText(name)
	})
}

func TestPageAPICanPushAndPopPages(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	var one, two, three pageAnimations
	myPages := &pageList{[]*testPage{builderPageRecording("1", "initial", &one)}}
	build := func() Widget {
		return pagesApp(pagesAppOpts{key: key, pages: pagesOf(myPages.pages), onPopPage: myPages.onPopPage})
	}
	ts.PumpWidget(build())
	if textsFound(ts, "initial", false) != 1 {
		t.Fatal("initial should show")
	}
	myPages.pages = []*testPage{
		builderPageRecording("1", "initial", &one),
		builderPageRecording("2", "second", &two),
		builderPageRecording("3", "third", &three),
	}
	ts.PumpWidget(build())
	// The third page is transitioning, and the secondary animation of the first
	// page chains with it. The animation of the second page does not start until
	// the third page finishes its transition.
	if one.secondary.Value() != three.primary.Value() || one.primary.Status() != animation.AnimationStatusCompleted ||
		two.primary != nil || three.secondary.Status() != animation.AnimationStatusDismissed ||
		three.primary.Status() != animation.AnimationStatusForward {
		t.Fatal("unexpected animations at the start")
	}
	ts.Pump(30 * time.Millisecond)
	if one.secondary.Value() != three.primary.Value() || one.primary.Status() != animation.AnimationStatusCompleted ||
		two.primary != nil || three.primary.Value() != 0.1 {
		t.Fatalf("unexpected animations at 30ms: %v", three.primary.Value())
	}
	ts.PumpAndSettle()
	// After the transition finishes, the animations of the routes are chained.
	if one.secondary.Value() != two.primary.Value() || one.primary.Status() != animation.AnimationStatusCompleted ||
		two.secondary.Value() != three.primary.Value() || two.primary.Status() != animation.AnimationStatusCompleted ||
		three.secondary.Status() != animation.AnimationStatusDismissed || three.primary.Status() != animation.AnimationStatusCompleted {
		t.Fatal("unexpected animations after the push")
	}
	if textsFound(ts, "third", false) != 1 || textsFound(ts, "second", false) != 0 || textsFound(ts, "initial", false) != 0 {
		t.Fatal("only the third page should show")
	}
	// Pops the pages using the Pages API and verifies that the animations chain.
	myPages.pages = []*testPage{builderPageRecording("1", "initial", &one), builderPageRecording("2", "second", &two)}
	ts.PumpWidget(build())
	ts.Pump(30 * time.Millisecond)
	if one.secondary.Value() != two.primary.Value() || one.primary.Status() != animation.AnimationStatusCompleted ||
		two.secondary.Value() != three.primary.Value() || two.primary.Status() != animation.AnimationStatusCompleted ||
		three.secondary.Status() != animation.AnimationStatusDismissed || three.primary.Value() != 0.9 {
		t.Fatalf("unexpected animations while popping: %v", three.primary.Value())
	}
	ts.PumpAndSettle()
	if one.secondary.Value() != two.primary.Value() || two.secondary.Value() != three.primary.Value() ||
		three.primary.Status() != animation.AnimationStatusDismissed {
		t.Error("unexpected animations after the pop")
	}
}

func TestPageAPICanModifyRoutesHistoryAndSecondaryAnimationStillWorks(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	var one, two, three pageAnimations
	pagesInOrder := func() []*testPage {
		return []*testPage{builderPageRecording("1", "initial", &one), builderPageRecording("2", "second", &two), builderPageRecording("3", "third", &three)}
	}
	myPages := &pageList{pagesInOrder()}
	build := func() Widget {
		return pagesApp(pagesAppOpts{key: key, pages: pagesOf(myPages.pages), onPopPage: myPages.onPopPage})
	}
	ts.PumpWidget(build())
	if textsFound(ts, "third", false) != 1 || textsFound(ts, "second", false) != 0 || textsFound(ts, "initial", false) != 0 {
		t.Fatal("only the third page should show")
	}
	if one.secondary.Value() != two.primary.Value() || one.primary.Status() != animation.AnimationStatusCompleted ||
		two.secondary.Value() != three.primary.Value() || two.primary.Status() != animation.AnimationStatusCompleted ||
		three.secondary.Status() != animation.AnimationStatusDismissed || three.primary.Status() != animation.AnimationStatusCompleted {
		t.Fatal("unexpected animations at the start")
	}
	slices.Reverse(myPages.pages)
	ts.PumpWidget(build())
	// The reversed routes are still chained correctly.
	if three.secondary.Value() != two.primary.Value() || three.primary.Status() != animation.AnimationStatusCompleted ||
		two.secondary.Value() != one.primary.Value() || two.primary.Status() != animation.AnimationStatusCompleted ||
		one.secondary.Status() != animation.AnimationStatusDismissed || one.primary.Status() != animation.AnimationStatusCompleted {
		t.Fatal("unexpected animations after reversing")
	}
	navState(t, key).Pop()
	ts.Pump()
	ts.Pump(30 * time.Millisecond)
	if three.secondary.Value() != two.primary.Value() || three.primary.Status() != animation.AnimationStatusCompleted ||
		two.secondary.Value() != one.primary.Value() || two.primary.Status() != animation.AnimationStatusCompleted ||
		one.secondary.Status() != animation.AnimationStatusDismissed || one.primary.Value() != 0.9 {
		t.Fatalf("unexpected animations during the first pop: %v", one.primary.Value())
	}
	ts.PumpAndSettle()
	if three.secondary.Value() != two.primary.Value() || two.secondary.Value() != one.primary.Value() ||
		one.secondary.Status() != animation.AnimationStatusDismissed || one.primary.Status() != animation.AnimationStatusDismissed {
		t.Fatal("unexpected animations after the first pop")
	}
	navState(t, key).Pop()
	ts.Pump()
	ts.Pump(30 * time.Millisecond)
	if three.secondary.Value() != two.primary.Value() || three.primary.Status() != animation.AnimationStatusCompleted ||
		two.secondary.Value() != one.primary.Value() || two.primary.Value() != 0.9 ||
		one.secondary.Status() != animation.AnimationStatusDismissed || one.primary.Status() != animation.AnimationStatusDismissed {
		t.Fatalf("unexpected animations during the second pop: %v", two.primary.Value())
	}
	ts.PumpAndSettle()
	if three.secondary.Value() != two.primary.Value() || three.primary.Status() != animation.AnimationStatusCompleted ||
		two.secondary.Value() != one.primary.Value() || two.primary.Status() != animation.AnimationStatusDismissed ||
		one.secondary.Status() != animation.AnimationStatusDismissed || one.primary.Status() != animation.AnimationStatusDismissed {
		t.Error("unexpected animations after the second pop")
	}
}

func TestPageAPIPopNoAnimationPageDoesNotCrash(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/86604.
	ts := gluttertest.New(t)
	zero := func(text string) *testPage {
		return newPage("", "", func(p *testPage) Route {
			return navNoAnimationRoute(p, func(BuildContext) Widget { return NewText(text) })
		})
	}
	build := func(secondPage bool) Widget {
		pages := []Page{zero("page1")}
		if secondPage {
			pages = append(pages, zero("page2"))
		}
		return pagesApp(pagesAppOpts{pages: pages, onPopPage: func(Route, any) bool { return false }})
	}
	ts.PumpWidget(build(true))
	if textsFound(ts, "page2", false) != 1 {
		t.Fatal("page2 should show")
	}
	ts.PumpWidget(build(false))
	if textsFound(ts, "page1", false) != 1 {
		t.Error("page1 should show")
	}
}

func TestPageAPIAddingMultiplePagesDoesNotJumpBeforeTheTopMostFinishesPushing(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/156033.
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	unique := func(child string) *testPage {
		return newMaterialPage(string(rune('a'+len(child)))+child, NewText(child))
	}
	pages := []Page{unique("home")}
	build := func() Widget {
		return pagesApp(pagesAppOpts{key: key, pages: pages, onRemoved: func(Page) {}})
	}
	ts.PumpWidget(build())
	if textsFound(ts, "home", false) != 1 {
		t.Fatal("home should show")
	}
	pages = append(slices.Clone(pages), unique("child1"), unique("child2"))
	ts.PumpWidget(build())
	// child1 is not created yet because the top-most route is still in transition.
	check := func(home, child1, child2 int) {
		t.Helper()
		if textsFound(ts, "home", false) != home || textsFound(ts, "child1", false) != child1 || textsFound(ts, "child2", false) != child2 {
			t.Fatalf("texts: home %d, child1 %d, child2 %d, want %d, %d, %d",
				textsFound(ts, "home", false), textsFound(ts, "child1", false), textsFound(ts, "child2", false), home, child1, child2)
		}
	}
	check(1, 0, 1)
	ts.Pump(150 * time.Millisecond)
	check(1, 0, 1)
	ts.PumpAndSettle()
	// Makes sure that child1 is in the navigator.
	navState(t, key).Pop()
	ts.PumpAndSettle()
	check(0, 1, 0)
	navState(t, key).Pop()
	ts.PumpAndSettle()
	check(1, 0, 0)
}

func TestPageAPICanWorkWithPagelessRoute(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	myPages := &pageList{[]*testPage{newTestPage("1", "initial"), newTestPage("2", "second")}}
	build := func() Widget {
		return pagesApp(pagesAppOpts{key: key, pages: pagesOf(myPages.pages), onPopPage: myPages.onPopPage})
	}
	pushText := func(text string) {
		navPush(key, heroPageRoute(func(BuildContext) Widget { return NewText(text) }))
	}
	only := func(text string, all ...string) {
		t.Helper()
		for _, n := range all {
			want := 0
			if n == text {
				want = 1
			}
			if got := textsFound(ts, n, false); got != want {
				t.Fatalf("only %q should show: %q found %d times", text, n, got)
			}
		}
	}
	names := []string{"initial", "second", "second-pageless1", "second-pageless2", "third", "third-pageless1"}
	ts.PumpWidget(build())
	only("second", names...)
	// Pushes two pageless routes above the second page route.
	pushText("second-pageless1")
	pushText("second-pageless2")
	ts.PumpAndSettle()
	// The history is [initial, second, second-pageless1, second-pageless2].
	only("second-pageless2", names...)

	myPages.pages = []*testPage{newTestPage("1", "initial"), newTestPage("2", "second"), newTestPage("3", "third")}
	ts.PumpWidget(build())
	ts.PumpAndSettle()
	only("third", names...)

	// Pushes one pageless route above the third page route.
	pushText("third-pageless1")
	ts.PumpAndSettle()
	// The history is [initial, second, second-pageless1, second-pageless2, third, third-pageless1].
	only("third-pageless1", names...)

	myPages.pages = []*testPage{newTestPage("1", "initial"), newTestPage("3", "third"), newTestPage("2", "second")}
	ts.PumpWidget(build())
	// Swapping the order without adding or removing does not trigger a transition:
	// the history is [initial, third, third-pageless1, second, second-pageless1,
	// second-pageless2], without a pumpAndSettle.
	only("second-pageless2", names...)
	// Pops the routes one by one to make sure the order is correct.
	for _, step := range []struct {
		shows string
		pages int
	}{
		{"second-pageless1", 3}, {"second", 3}, {"third-pageless1", 2}, {"third", 2}, {"initial", 1},
	} {
		navState(t, key).Pop()
		ts.PumpAndSettle()
		only(step.shows, names...)
		if len(myPages.pages) != step.pages {
			t.Fatalf("%d pages after showing %q, want %d", len(myPages.pages), step.shows, step.pages)
		}
	}
}

// resultTracker records which of the pushed routes completed.
type resultTracker struct {
	chans map[string]<-chan any
	done  map[string]bool
}

func (r *resultTracker) completed(name string) bool {
	if !r.done[name] {
		if _, ok := nowOrNothing(r.chans[name]); ok {
			r.done[name] = true
		}
	}
	return r.done[name]
}

func TestPageAPIComplexCases(t *testing.T) {
	for _, tc := range []struct {
		name     string
		delegate TransitionDelegate
	}{
		{"complex case 1", nil},
		{"complex case 1 - with always remove transition delegate", alwaysRemoveDelegate{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			key := NewGlobalKey[*NavigatorState]()
			myPages := &pageList{[]*testPage{newTestPage("1", "initial")}}
			build := func() Widget {
				return pagesApp(pagesAppOpts{key: key, pages: pagesOf(myPages.pages), onPopPage: myPages.onPopPage, delegate: tc.delegate})
			}
			tracker := &resultTracker{chans: map[string]<-chan any{}, done: map[string]bool{}}
			push := func(name string) {
				tracker.chans[name] = navPush(key, heroPageRoute(func(BuildContext) Widget { return NewText(name) }))
			}
			check := func(want map[string]bool) {
				t.Helper()
				for name, w := range want {
					if got := tracker.completed(name); got != w {
						t.Errorf("%s completed = %v, want %v", name, got, w)
					}
				}
			}
			// Adds the initial page route with one pageless route.
			ts.PumpWidget(build())
			push("initial-pageless1")
			ts.PumpAndSettle()
			// Pushes the second page route with two pageless routes.
			myPages.pages = []*testPage{newTestPage("1", "initial"), newTestPage("2", "second")}
			ts.PumpWidget(build())
			ts.PumpAndSettle()
			push("second-pageless1")
			ts.PumpAndSettle()
			push("second-pageless2")
			ts.PumpAndSettle()
			// Pushes the third page route with one pageless route.
			myPages.pages = []*testPage{newTestPage("1", "initial"), newTestPage("2", "second"), newTestPage("3", "third")}
			ts.PumpWidget(build())
			ts.PumpAndSettle()
			push("third-pageless1")
			ts.PumpAndSettle()
			// Nothing has been popped.
			check(map[string]bool{"initial-pageless1": false, "second-pageless1": false, "second-pageless2": false, "third-pageless1": false})
			// Switches the order and removes the initial page route.
			myPages.pages = []*testPage{newTestPage("3", "third"), newTestPage("2", "second")}
			ts.PumpWidget(build())
			// The pageless route of the initial page route is completed.
			check(map[string]bool{"initial-pageless1": true, "second-pageless1": false, "second-pageless2": false, "third-pageless1": false})
			myPages.pages = []*testPage{newTestPage("3", "third")}
			ts.PumpWidget(build())
			ts.PumpAndSettle()
			check(map[string]bool{"second-pageless1": true, "second-pageless2": true, "third-pageless1": false})
			myPages.pages = []*testPage{newTestPage("4", "forth")}
			ts.PumpWidget(build())
			if tc.delegate != nil {
				ts.Pump()
			}
			check(map[string]bool{"third-pageless1": true})
			ts.PumpAndSettle()
			if textsFound(ts, "forth", false) != 1 {
				t.Error("forth should show")
			}
		})
	}
}

// alwaysRemoveDelegate is AlwaysRemoveTransitionDelegate: it completes every
// exiting route without a transition and adds every new one.
type alwaysRemoveDelegate struct{}

func (alwaysRemoveDelegate) Resolve(newPageRouteHistory []RouteTransitionRecord, locationToExitingPageRoute map[RouteTransitionRecord]RouteTransitionRecord, pageRouteToPagelessRoutes map[RouteTransitionRecord][]RouteTransitionRecord) []RouteTransitionRecord {
	var results []RouteTransitionRecord
	var handleExitingRoute func(location RouteTransitionRecord)
	handleExitingRoute = func(location RouteTransitionRecord) {
		exiting, ok := locationToExitingPageRoute[location]
		if !ok {
			return
		}
		if exiting.IsWaitingForExitingDecision() {
			exiting.MarkForComplete()
			for _, pageless := range pageRouteToPagelessRoutes[exiting] {
				pageless.MarkForComplete()
			}
		}
		results = append(results, exiting)
		handleExitingRoute(exiting)
	}
	handleExitingRoute(nil)
	for _, pageRoute := range newPageRouteHistory {
		if pageRoute.IsWaitingForEnteringDecision() {
			pageRoute.MarkForAdd()
		}
		results = append(results, pageRoute)
		handleExitingRoute(pageRoute)
	}
	return results
}

func TestPageAPIComplexCase2(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/115887.
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	myPages := &pageList{[]*testPage{newTestPage("1", "initial"), newTestPage("2", "second")}}
	build := func() Widget {
		return pagesApp(pagesAppOpts{key: key, pages: pagesOf(myPages.pages), onPopPage: myPages.onPopPage})
	}
	ts.PumpWidget(build())
	if textsFound(ts, "second", false) != 1 || textsFound(ts, "initial", false) != 0 {
		t.Fatal("second should show")
	}
	// Pushes a pageless route above the second page route.
	navPush(key, heroPageRoute(func(BuildContext) Widget { return NewText("second-pageless1") }))
	ts.PumpAndSettle()
	// The history is [initial, second, second-pageless1].
	if textsFound(ts, "second-pageless1", false) != 1 || textsFound(ts, "second", false) != 0 || len(myPages.pages) != 2 {
		t.Fatal("the pageless route should show")
	}
	myPages.pages = []*testPage{newTestPage("2", "second")}
	ts.PumpWidget(build())
	ts.PumpAndSettle()
	// The history is [second, second-pageless1].
	if textsFound(ts, "second-pageless1", false) != 1 || textsFound(ts, "initial", false) != 0 || len(myPages.pages) != 1 {
		t.Fatal("the pageless route should still show")
	}
	// Pops the pageless route.
	navState(t, key).Pop()
	ts.PumpAndSettle()
	if len(myPages.pages) != 1 || textsFound(ts, "second", false) != 1 || textsFound(ts, "second-pageless1", false) != 0 {
		t.Error("the second page should show")
	}
}

func TestPageAPIUpdatingPagesDuringAPop(t *testing.T) {
	t.Run("can repush a page that was popped before it has finished popping", func(t *testing.T) {
		ts := gluttertest.New(t)
		myPages := &pageList{[]*testPage{newTestPage("1", "initial"), newTestPage("2", "second")}}
		build := func() Widget {
			return pagesApp(pagesAppOpts{pages: pagesOf(myPages.pages), onPopPage: myPages.onPopPage})
		}
		ts.PumpWidget(build())
		// Pops the second page route.
		myPages.pages = []*testPage{newTestPage("1", "initial")}
		ts.PumpWidget(build())
		// Re-pushes the second page before it finishes popping.
		myPages.pages = []*testPage{newTestPage("1", "initial"), newTestPage("2", "second")}
		ts.PumpWidget(build())
		if err := ts.TakeError(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ts.PumpAndSettle()
		if textsFound(ts, "second", false) != 1 {
			t.Error("second should show")
		}
	})
	t.Run("can update pages before a route has finished popping", func(t *testing.T) {
		ts := gluttertest.New(t)
		myPages := &pageList{[]*testPage{newTestPage("1", "initial"), newTestPage("2", "second")}}
		build := func() Widget {
			return pagesApp(pagesAppOpts{pages: pagesOf(myPages.pages), onPopPage: myPages.onPopPage})
		}
		ts.PumpWidget(build())
		myPages.pages = []*testPage{newTestPage("1", "initial")}
		ts.PumpWidget(build())
		// Updates the pages again before the second page finishes popping.
		myPages.pages = []*testPage{newTestPage("1", "initial")}
		ts.PumpWidget(build())
		if err := ts.TakeError(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ts.PumpAndSettle()
		if textsFound(ts, "initial", false) != 1 {
			t.Error("initial should show")
		}
	})
	t.Run("can update pages before a pageless route has finished popping", func(t *testing.T) {
		// Regression test for https://github.com/flutter/flutter/issues/68162.
		ts := gluttertest.New(t)
		key := NewGlobalKey[*NavigatorState]()
		myPages := &pageList{[]*testPage{newTestPage("1", "initial"), newTestPage("2", "second")}}
		build := func() Widget {
			return pagesApp(pagesAppOpts{key: key, pages: pagesOf(myPages.pages), onPopPage: myPages.onPopPage})
		}
		ts.PumpWidget(build())
		// Pushes a pageless route.
		current := key.CurrentContext()
		ShowGeneralDialog(current, func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
			return NewText("dialog")
		}, ShowGeneralDialogOpts{UseRootNavigator: foundation.Some(false)})
		ts.PumpAndSettle()
		if textsFound(ts, "dialog", false) != 1 {
			t.Fatal("the dialog should show")
		}
		// Pops the pageless route; before the pop finishes, updates the page list.
		navState(t, key).Pop()
		myPages.pages = []*testPage{newTestPage("1", "initial")}
		ts.PumpWidget(build())
		if err := ts.TakeError(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ts.PumpAndSettle()
		if textsFound(ts, "initial", false) != 1 {
			t.Error("initial should show")
		}
	})
}

func TestPageAPIPagesRemoveAndAddTriggerObserversInTheRightOrder(t *testing.T) {
	ts := gluttertest.New(t)
	var observations []navObservation
	name := func(r Route) string {
		if r == nil {
			return ""
		}
		return r.Settings().Name()
	}
	observer := &navTestObserver{}
	observer.onPushed = func(route, previous Route) {
		observations = append(observations, navObservation{name(route), name(previous), "push"})
	}
	observer.onRemoved = func(route, previous Route) {
		observations = append(observations, navObservation{name(route), name(previous), "remove"})
	}
	build := func(pages ...*testPage) Widget {
		return pagesApp(pagesAppOpts{pages: pagesOf(pages), onPopPage: func(Route, any) bool { return false }, observers: []NavigatorObserver{observer}})
	}
	ts.PumpWidget(build(newTestPage("1", "first"), newTestPage("2", "second"), newTestPage("3", "third")))
	ts.PumpWidget(build(newTestPage("4", "forth"), newTestPage("5", "fifth")))
	ts.PumpAndSettle()
	want := []navObservation{
		// The initial routes are pushed.
		{"first", "", "push"}, {"second", "first", "push"}, {"third", "second", "push"},
		// The new routes are pushed before the initial routes are removed.
		{"forth", "third", "push"}, {"fifth", "forth", "push"},
		// The initial routes are removed.
		{"third", "", "remove"}, {"second", "", "remove"}, {"first", "", "remove"},
	}
	if !slices.Equal(observations, want) {
		t.Errorf("observations = %v, want %v", observations, want)
	}
}

func TestPageAPIOnDidRemovePageIsCalledWhenAPageRouteIsPopped(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[*NavigatorState]()
	pages := []*testPage{newTestPage("1", "first"), newTestPage("2", "second")}
	var removed []Page
	ts.PumpWidget(pagesApp(pagesAppOpts{key: key, pages: pagesOf(pages), onRemoved: func(p Page) { removed = append(removed, p) }}))
	navState(t, key).Pop()
	ts.PumpAndSettle()
	if len(removed) != 1 || removed[0] != Page(pages[1]) {
		t.Errorf("removed = %v, want the second page", removed)
	}
}

// canPopPage is CanPopPage: a page whose canPop is a function, and that records
// the calls of onPopInvoked.
type canPopPage struct {
	testPage
	pageCanPop func() bool
	popInvoked [][2]any
}

func (p *canPopPage) CanPop() bool { return p.pageCanPop() }

func TestPageAPICanPopAndOnPopInvoked(t *testing.T) {
	ts := gluttertest.New(t)
	page2CanPop, page3CanPop := false, false
	newCanPopPage := func(name string, canPop func() bool) *canPopPage {
		p := &canPopPage{pageCanPop: canPop}
		p.create = func(*testPage) Route {
			return heroPageRoute(func(BuildContext) Widget { return NewText(name) }, p)
		}
		InitPageBase(&p.PageBase, p, PageOpts{Name: name, OnPopInvoked: func(didPop bool, result any) {
			p.popInvoked = append(p.popInvoked, [2]any{didPop, result})
		}})
		return p
	}
	page1 := newCanPopPage("page1", func() bool { return false })
	page2 := newCanPopPage("page2", func() bool { return page2CanPop })
	page3 := newCanPopPage("page3", func() bool { return page3CanPop })
	pages := []Page{page1, page2, page3}
	key := NewGlobalKey[*NavigatorState]()
	ts.PumpWidget(pagesApp(pagesAppOpts{key: key, pages: pages, onRemoved: func(p Page) {
		pages = slices.DeleteFunc(pages, func(q Page) bool { return q == p })
	}}))
	if textsFound(ts, "page3", false) != 1 {
		t.Fatal("page3 should show")
	}
	navState(t, key).MaybePop(true)
	ts.PumpAndSettle()
	if textsFound(ts, "page3", false) != 1 || !slices.Equal(page3.popInvoked, [][2]any{{false, true}}) {
		t.Fatalf("page3 should stay: %v", page3.popInvoked)
	}
	page3CanPop = true
	navState(t, key).MaybePop(false)
	ts.PumpAndSettle()
	if textsFound(ts, "page3", false) != 0 || textsFound(ts, "page2", false) != 1 ||
		!slices.Equal(page3.popInvoked, [][2]any{{false, true}, {true, false}}) {
		t.Fatalf("page3 should pop: %v", page3.popInvoked)
	}
	navState(t, key).MaybePop("some string")
	ts.PumpAndSettle()
	if textsFound(ts, "page2", false) != 1 || !slices.Equal(page2.popInvoked, [][2]any{{false, "some string"}}) {
		t.Fatalf("page2 should stay: %v", page2.popInvoked)
	}
	page2CanPop = true
	navState(t, key).MaybePop("another string")
	ts.PumpAndSettle()
	if textsFound(ts, "page2", false) != 0 || textsFound(ts, "page1", false) != 1 ||
		!slices.Equal(page2.popInvoked, [][2]any{{false, "some string"}, {true, "another string"}}) {
		t.Fatalf("page2 should pop: %v", page2.popInvoked)
	}
	navState(t, key).MaybePop(1)
	ts.PumpAndSettle()
	if textsFound(ts, "page1", false) != 1 || !slices.Equal(page1.popInvoked, [][2]any{{false, 1}}) {
		t.Errorf("page1 should stay: %v", page1.popInvoked)
	}
}
