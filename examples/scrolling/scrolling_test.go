package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func newTester(t *testing.T, o options) (*gluttertest.Tester, *scrollingState) {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 640, Height: 480}})
	ts.PumpWidget(newScrollingApp(o))
	ts.Pump()
	ts.Pump(time.Second) // the thumbs fade in
	return ts, gluttertest.StateOf[*scrollingState](ts, gluttertest.ByType[*scrollingApp]())
}

func noErrors(t *testing.T, ts *gluttertest.Tester) {
	t.Helper()
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("the example reported errors: %v", errs)
	}
}

// TestScrollingGoldens draws each tab and compares it with its golden image. The
// CustomScrollView tab is drawn scrolled, so that the pinned header overlaps the
// list.
func TestScrollingGoldens(t *testing.T) {
	tests := []struct {
		name string
		opts options
	}{
		{"list", options{tab: tabList}},
		{"grid", options{tab: tabGrid}},
		{"custom", options{tab: tabCustom}},
		{"custom-scrolled", options{tab: tabCustom, scroll: 600}},
		{"pages", options{tab: tabPages}},
		{"reorder", options{tab: tabReorder}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, _ := newTester(t, tt.opts)
			ts.MatchesGolden("scrolling-" + tt.name)
			noErrors(t, ts)
		})
	}
}

// TestTabsSwitchTheView taps each tab button and checks which view is built.
func TestTabsSwitchTheView(t *testing.T) {
	tests := []struct {
		tab  int
		view func(...gluttertest.FindOpts) gluttertest.Finder
	}{
		{tabList, gluttertest.ByType[*widgets.ListView]},
		{tabGrid, gluttertest.ByType[*widgets.GridView]},
		{tabCustom, gluttertest.ByType[*widgets.CustomScrollView]},
		{tabPages, gluttertest.ByType[*widgets.PageView]},
		{tabReorder, gluttertest.ByType[*widgets.ReorderableList]},
	}
	ts, state := newTester(t, options{tab: tabReorder})
	for _, tt := range tests {
		ts.Tap(gluttertest.ByText(tabNames[tt.tab]))
		ts.Pump()
		if state.tab != tt.tab {
			t.Fatalf("tab = %d after tapping %q, want %d", state.tab, tabNames[tt.tab], tt.tab)
		}
		if got := ts.Count(tt.view()); got != 1 {
			t.Errorf("tab %q builds %d views, want 1", tabNames[tt.tab], got)
		}
	}
	noErrors(t, ts)
}

// TestEveryViewHasAScrollbar checks that each tab with a scroll view builds a
// RawScrollbar whose thumb is visible.
func TestEveryViewHasAScrollbar(t *testing.T) {
	for _, tab := range []int{tabList, tabGrid, tabCustom, tabReorder} {
		ts, _ := newTester(t, options{tab: tab})
		if got := ts.Count(gluttertest.ByType[*widgets.RawScrollbar]()); got != 1 {
			t.Errorf("tab %q: %d scrollbars, want 1", tabNames[tab], got)
		}
		noErrors(t, ts)
	}
}

// TestFlingTheList flings the list and checks that it keeps scrolling after the
// finger leaves and that only the visible rows are built.
func TestFlingTheList(t *testing.T) {
	ts, state := newTester(t, options{tab: tabList})
	c := state.controllers[tabList]
	if c.Offset() != 0 {
		t.Fatalf("initial offset = %v, want 0", c.Offset())
	}
	ts.Fling(gluttertest.ByType[*widgets.ListView](), ui.Offset{Dy: -200}, 1500)
	atRelease := c.Offset()
	if atRelease < 200 {
		t.Fatalf("offset after the drag = %v, want 200 or more", atRelease)
	}
	ts.PumpAndSettle()
	if got := c.Offset(); got <= atRelease {
		t.Errorf("offset after the fling settles = %v, want more than %v", got, atRelease)
	}
	if n := ts.Count(gluttertest.ByType[*widgets.Text]()); n > 40 {
		t.Errorf("%d texts are built, want only the rows near the viewport", n)
	}
	noErrors(t, ts)
}

// TestPinnedHeaderStaysAtTheTop scrolls the CustomScrollView tab and checks the
// header's size: 140 at rest, 56 once it is pinned.
func TestPinnedHeaderStaysAtTheTop(t *testing.T) {
	ts, _ := newTester(t, options{tab: tabCustom})
	header := gluttertest.Descendant(gluttertest.ByType[*widgets.SliverPersistentHeader](), gluttertest.ByType[*widgets.DecoratedBox]())
	if got := ts.Size(header).Height; got != headerMax {
		t.Fatalf("header height at rest = %v, want %v", got, headerMax)
	}
	top := ts.TopLeft(header).Dy
	ts.Drag(gluttertest.ByType[*widgets.CustomScrollView](), ui.Offset{Dy: -400})
	ts.PumpAndSettle()
	if got := ts.Size(header).Height; got != headerMin {
		t.Errorf("header height when pinned = %v, want %v", got, headerMin)
	}
	if got := ts.TopLeft(header).Dy; got != top {
		t.Errorf("header top = %v, want %v (pinned)", got, top)
	}
	noErrors(t, ts)
}

// TestPageViewSwipes swipes the PageView from the first page to the last.
func TestPageViewSwipes(t *testing.T) {
	ts, state := newTester(t, options{tab: tabPages})
	for want := 1; want <= 2; want++ {
		ts.Fling(gluttertest.ByType[*widgets.PageView](), ui.Offset{Dx: -300}, 1000)
		ts.PumpAndSettle()
		if got := state.pages.Page().Value(); got != float64(want) {
			t.Fatalf("page = %v, want %d", got, want)
		}
	}
	noErrors(t, ts)
}

// TestReorderMovesAnItem drags the handle of the first item two rows down and
// checks the order the list reports.
func TestReorderMovesAnItem(t *testing.T) {
	ts, state := newTester(t, options{tab: tabReorder})
	drag := ts.StartGesture(ts.Center(gluttertest.First(gluttertest.ByType[*widgets.ReorderableDragStartListener]())))
	ts.Pump(gestures.PressTimeout)
	drag.MoveBy(ui.Offset{Dy: 20})
	for range 8 { // a hand moves a little at a time; one big jump skips the gap logic
		drag.MoveBy(ui.Offset{Dy: 16})
		ts.Pump(50 * time.Millisecond)
	}
	ts.PumpAndSettle()
	drag.Up()
	ts.PumpAndSettle()
	want := []int{1, 2, 0, 3}
	for i, w := range want {
		if state.order[i] != w {
			t.Fatalf("order = %v, want it to start with %v", state.order, want)
		}
	}
	noErrors(t, ts)
}

func TestOptionsFromEnv(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	tests := []struct {
		name  string
		env   map[string]string
		want  options
		route string
	}{
		{"empty", nil, options{}, ""},
		{"tab and scroll", map[string]string{"GLUTTER_DEV_TAB": "2", "GLUTTER_DEV_SCROLL": "600"}, options{tab: 2, scroll: 600}, ""},
		{"tab not a number", map[string]string{"GLUTTER_DEV_TAB": "x"}, options{}, "ScrollingTabInvalid"},
		{"tab out of range", map[string]string{"GLUTTER_DEV_TAB": "5"}, options{}, "ScrollingTabInvalid"},
		{"scroll negative", map[string]string{"GLUTTER_DEV_SCROLL": "-1"}, options{}, "ScrollingOffsetInvalid"},
		{"scroll not a number", map[string]string{"GLUTTER_DEV_SCROLL": "far"}, options{}, "ScrollingOffsetInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !foundation.DebugMode {
				t.Skip("the environment is read in debug builds only")
			}
			got, err := optionsFromEnv(env(tt.env))
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
	if err != nil || got != (options{}) {
		t.Errorf("options = %+v, %v; want the zero options", got, err)
	}
}
