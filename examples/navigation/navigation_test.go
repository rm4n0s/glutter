package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func newTester(t *testing.T, o options) *gluttertest.Tester {
	t.Helper()
	if o.route == "" {
		o.route = routeHome
	}
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 640, Height: 480}})
	ts.PumpWidget(newApp(o))
	ts.PumpAndSettle()
	return ts
}

func noErrors(t *testing.T, ts *gluttertest.Tester) {
	t.Helper()
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("the example reported errors: %v", errs)
	}
}

// focusedLink is the text of the link that shows the focus highlight, or "".
func focusedLink(ts *gluttertest.Tester) string {
	n := ts.Count(gluttertest.ByType[*link]())
	for i := range n {
		f := gluttertest.At(gluttertest.ByType[*link](), i)
		if gluttertest.StateOf[*linkState](ts, f).focused {
			return ts.Widget(f).(*link).text
		}
	}
	return ""
}

// homeState is the state of the home page.
func homeStateOf(ts *gluttertest.Tester) *homeState {
	return gluttertest.StateOf[*homeState](ts, gluttertest.ByType[*homePage]())
}

// settleResult pumps frames until the goroutine that waits for the result of a
// route has handed it to the UI thread.
func settleResult(t *testing.T, ts *gluttertest.Tester, done func() bool) {
	t.Helper()
	for range 200 {
		ts.Pump(10 * time.Millisecond)
		if done() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("the result of the route did not arrive")
}

func pressKey(ts *gluttertest.Tester, key services.LogicalKeyboardKey) {
	ts.SendKeyEvent(key)
	ts.Pump()
}

func pressShiftTab(ts *gluttertest.Tester) {
	ts.SendKeyDownEvent(services.LogicalKeyboardKeyShiftLeft)
	ts.SendKeyEvent(services.LogicalKeyboardKeyTab)
	ts.SendKeyUpEvent(services.LogicalKeyboardKeyShiftLeft)
	ts.Pump()
}

// TestNavigationGoldens draws each page after its transition and compares it with
// its golden image.
func TestNavigationGoldens(t *testing.T) {
	tests := []struct {
		name string
		opts options
	}{
		{"home", options{}},
		{"home-pages", options{tab: tabPages}},
		{"detail", options{route: routeDetail}},
		{"pick", options{route: routePick}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t, tt.opts)
			ts.MatchesGolden("navigation-" + tt.name)
			noErrors(t, ts)
		})
	}
}

// TestPushSlidesTheDetailPageAndPopRemovesIt taps the link of the detail page,
// checks that the page slides in, and pops it with the Back link.
func TestPushSlidesTheDetailPageAndPopRemovesIt(t *testing.T) {
	ts := newTester(t, options{})
	detail := gluttertest.ByType[*detailPage]()
	if ts.Exists(detail) {
		t.Fatal("the detail page exists before the push")
	}
	ts.Tap(gluttertest.ByText("Open detail page"))
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	if x := ts.TopLeft(detail).Dx; x <= 0 || x >= 640 {
		t.Errorf("the detail page is at x = %v after 100 ms, want it between the edge (640) and 0", x)
	}
	ts.PumpAndSettle()
	if x := ts.TopLeft(detail).Dx; x != 0 {
		t.Errorf("the detail page ends at x = %v, want 0", x)
	}
	if !ts.Exists(gluttertest.ByText("Sunrise over the lake")) {
		t.Error("the detail page does not show its title")
	}
	ts.Tap(gluttertest.ByText("Back"))
	ts.PumpAndSettle()
	if ts.Exists(detail) {
		t.Error("the detail page is still there after Back")
	}
	if !ts.Exists(gluttertest.ByText("Open detail page")) {
		t.Error("the home page is not back")
	}
	noErrors(t, ts)
}

// TestHeroFliesBetweenThePages checks that the thumbnail and the large picture
// are heroes with one tag, and that the flight puts one hero in the overlay while
// the transition runs.
func TestHeroFliesBetweenThePages(t *testing.T) {
	ts := newTester(t, options{})
	if got := ts.Count(gluttertest.ByType[*widgets.Hero]()); got != 1 {
		t.Fatalf("the home page has %d heroes, want 1", got)
	}
	thumb := ts.Size(gluttertest.ByType[*widgets.Hero]())
	ts.Tap(gluttertest.ByText("Open detail page"))
	ts.Pump()
	ts.Pump(150 * time.Millisecond)
	flying := ts.Count(gluttertest.ByType[*widgets.Hero]())
	if flying == 0 {
		t.Error("no hero while the route transition runs")
	}
	ts.PumpAndSettle()
	if got := ts.Count(gluttertest.ByType[*widgets.Hero]()); got != 1 {
		t.Errorf("%d heroes after the transition, want only the one of the detail page", got)
	}
	if big := ts.Size(gluttertest.ByType[*widgets.Hero]()); big.Width <= thumb.Width || big.Height <= thumb.Height {
		t.Errorf("the hero of the detail page is %v, want it larger than the thumbnail %v", big, thumb)
	}
	noErrors(t, ts)
}

// TestPickAColorReturnsTheResult follows the choices of the picker page and reads
// the value that arrives through the channel of NavigatorPush.
func TestPickAColorReturnsTheResult(t *testing.T) {
	tests := []struct {
		name string
		do   func(ts *gluttertest.Tester)
		want string
	}{
		{"choice", func(ts *gluttertest.Tester) { ts.Tap(gluttertest.ByText("Amber")) }, "Amber"},
		{"cancel", func(ts *gluttertest.Tester) { ts.Tap(gluttertest.ByText("Cancel")) }, ""},
		{"escape", func(ts *gluttertest.Tester) { pressKey(ts, services.LogicalKeyboardKeyEscape) }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t, options{})
			ts.Tap(gluttertest.ByText("Pick a color"))
			ts.PumpAndSettle()
			if !ts.Exists(gluttertest.ByType[*pickPage]()) {
				t.Fatal("the picker page was not pushed")
			}
			tt.do(ts)
			ts.PumpAndSettle()
			if ts.Exists(gluttertest.ByType[*pickPage]()) {
				t.Fatal("the picker page is still there")
			}
			if tt.want != "" {
				settleResult(t, ts, func() bool { return homeStateOf(ts).picked == tt.want })
				if !ts.Exists(gluttertest.ByText("Picked color: " + tt.want)) {
					t.Errorf("the home page does not show %q", tt.want)
				}
			} else {
				ts.Pump(50 * time.Millisecond)
				if got := homeStateOf(ts).picked; got != "" {
					t.Errorf("picked = %q after a dismissal, want none", got)
				}
			}
			noErrors(t, ts)
		})
	}
}

// TestTabMovesTheFocusAndShiftTabMovesItBack follows the focus through the links
// of the home page with Tab, and back with Shift+Tab.
func TestTabMovesTheFocusAndShiftTabMovesItBack(t *testing.T) {
	ts := newTester(t, options{})
	if got := focusedLink(ts); got != "" {
		t.Fatalf("link %q has the focus highlight before any key", got)
	}
	want := []string{"Imperative", "Pages API", "Open detail page", "Pick a color"}
	for _, text := range want {
		pressKey(ts, services.LogicalKeyboardKeyTab)
		if got := focusedLink(ts); got != text {
			t.Fatalf("after Tab the focus is on %q, want %q", got, text)
		}
	}
	for i := len(want) - 2; i >= 0; i-- {
		pressShiftTab(ts)
		if got := focusedLink(ts); got != want[i] {
			t.Fatalf("after Shift+Tab the focus is on %q, want %q", got, want[i])
		}
	}
	noErrors(t, ts)
}

// TestEnterFollowsTheFocusedLink opens the Pages API tab with the keyboard.
func TestEnterFollowsTheFocusedLink(t *testing.T) {
	ts := newTester(t, options{})
	pressKey(ts, services.LogicalKeyboardKeyTab)
	pressKey(ts, services.LogicalKeyboardKeyTab)
	if got := focusedLink(ts); got != "Pages API" {
		t.Fatalf("the focus is on %q, want %q", got, "Pages API")
	}
	pressKey(ts, services.LogicalKeyboardKeyEnter)
	ts.PumpAndSettle()
	if got := homeStateOf(ts).tab; got != tabPages {
		t.Errorf("tab = %d after Enter, want %d", got, tabPages)
	}
	noErrors(t, ts)
}

// TestDetailPageFocusesBackAndEscapePops checks the focus on the pushed page and
// that Escape pops it, and does nothing on the home page.
func TestDetailPageFocusesBackAndEscapePops(t *testing.T) {
	ts := newTester(t, options{})
	pressKey(ts, services.LogicalKeyboardKeyEscape) // nothing to go back to
	if !ts.Exists(gluttertest.ByType[*homePage]()) {
		t.Fatal("Escape on the home page removed it")
	}
	ts.Tap(gluttertest.ByText("Open detail page"))
	ts.PumpAndSettle()
	pressKey(ts, services.LogicalKeyboardKeyTab) // switches the highlight to the keyboard
	if got := focusedLink(ts); got == "" {
		t.Error("no link shows the focus highlight on the detail page")
	}
	pressKey(ts, services.LogicalKeyboardKeyEscape)
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByType[*detailPage]()) {
		t.Error("Escape did not pop the detail page")
	}
	noErrors(t, ts)
}

// TestFavoriteLinkTogglesItsLabel follows the second link of the detail page with
// the keyboard.
func TestFavoriteLinkTogglesItsLabel(t *testing.T) {
	ts := newTester(t, options{route: routeDetail})
	if got := focusedLink(ts); got != "Back" {
		t.Fatalf("the focus is on %q, want the autofocused Back", got)
	}
	pressKey(ts, services.LogicalKeyboardKeyTab)
	if got := focusedLink(ts); got != "Favorite" {
		t.Fatalf("the focus is on %q, want Favorite", got)
	}
	pressKey(ts, services.LogicalKeyboardKeySpace)
	if !ts.Exists(gluttertest.ByText("Favorited")) {
		t.Error("the link does not say Favorited after Space")
	}
	noErrors(t, ts)
}

// TestPagesAPITabKeepsAStackOfPages pushes and pops pages of the nested navigator
// by changing the list, with the link, the Back link and Escape.
func TestPagesAPITabKeepsAStackOfPages(t *testing.T) {
	ts := newTester(t, options{tab: tabPages})
	stack := func() []int { return homeStateOf(ts).stack }
	equal := func(got []int, want ...int) bool {
		if len(got) != len(want) {
			return false
		}
		for i := range got {
			if got[i] != want[i] {
				return false
			}
		}
		return true
	}
	if !ts.Exists(gluttertest.ByText("Stack: Step 1")) {
		t.Fatal("the tab does not show the first page")
	}
	ts.Tap(gluttertest.ByText("Push page"))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("Push page"))
	ts.PumpAndSettle()
	if !equal(stack(), 1, 2, 3) {
		t.Fatalf("stack = %v after two pushes, want [1 2 3]", stack())
	}
	if !ts.Exists(gluttertest.ByText("Stack: Step 1 > Step 2 > Step 3")) {
		t.Error("the tab does not list the three pages")
	}
	ts.Tap(gluttertest.ByText("Back")) // the Back link of the top page pops the nested navigator
	ts.PumpAndSettle()
	if !equal(stack(), 1, 2) {
		t.Fatalf("stack = %v after Back, want [1 2]", stack())
	}
	if !ts.Exists(gluttertest.ByText("Step 2")) {
		t.Error("page 2 is not on top")
	}
	pressKey(ts, services.LogicalKeyboardKeyEscape) // the focus is on page 2, in the nested navigator
	ts.PumpAndSettle()
	if !equal(stack(), 1) {
		t.Errorf("stack = %v after Escape, want [1]", stack())
	}
	if !ts.Exists(gluttertest.ByType[*homePage]()) {
		t.Error("Escape in the nested navigator popped the home page")
	}
	noErrors(t, ts)
}

// TestOptionsFromEnv reads the variables of a debug build.
func TestOptionsFromEnv(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		want  options
		route string
	}{
		{"none", nil, options{route: routeHome}, ""},
		{"route and tab", map[string]string{"GLUTTER_DEV_ROUTE": "/detail", "GLUTTER_DEV_TAB": "1"}, options{route: routeDetail, tab: 1}, ""},
		{"bad route", map[string]string{"GLUTTER_DEV_ROUTE": "/nope"}, options{}, "NavigationRouteInvalid"},
		{"tab out of range", map[string]string{"GLUTTER_DEV_TAB": "7"}, options{}, "NavigationTabInvalid"},
		{"tab not a number", map[string]string{"GLUTTER_DEV_TAB": "x"}, options{}, "NavigationTabInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !foundation.DebugMode {
				t.Skip("the environment is read in debug builds only")
			}
			got, err := optionsFromEnv(func(k string) string { return tt.env[k] })
			if tt.route == "" {
				if err != nil {
					t.Fatalf("error: %v", err)
				}
				if got != tt.want {
					t.Errorf("options = %+v, want %+v", got, tt.want)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

// TestOptionsIgnoredInRelease checks that a release build ignores the variables.
func TestOptionsIgnoredInRelease(t *testing.T) {
	if foundation.DebugMode {
		t.Skip("debug builds read the environment")
	}
	got, err := optionsFromEnv(func(string) string { return "99" })
	if err != nil || got != (options{route: routeHome}) {
		t.Errorf("options = %+v, %v; want the home route", got, err)
	}
}
