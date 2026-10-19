package gluttertest

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

var (
	keyA = foundation.NewKey("a")
	grey = ui.NewColor(0xFF808080)
)

// finderTree is the tree of the finder tests: two swatches that stand for Text,
// Icon and Tooltip widgets, an offstage subtree, and a counter.
func finderTree(t *testing.T) (*Tester, *swatch) {
	t.Helper()
	ts := New(t)
	a := newSwatch(grey, 20, 20).keyed(keyA)
	a.text = "Hello"
	b := newSwatch(grey, 30, 20)
	b.text, b.icon, b.tooltip = "World", "star", "Tip"
	secret := newSwatch(grey, 10, 10)
	secret.text = "Secret"
	ts.PumpWidget(newRow(a, b, newHidden(secret), newCounter()))
	return ts, a
}

func TestFinders(t *testing.T) {
	ts, a := finderTree(t)
	tests := []struct {
		name string
		f    Finder
		want int
	}{
		{"text", ByText("Hello"), 1},
		{"text, no match", ByText("Hell"), 0},
		{"text containing", ByTextContaining("o"), 2},
		{"text of the offstage subtree is skipped", ByText("Secret"), 0},
		{"offstage included", ByText("Secret", FindOpts{IncludeOffstage: true}), 1},
		{"key", ByKey(keyA), 1},
		{"key, no match", ByKey(foundation.NewKey("zzz")), 0},
		{"type", ByType[*swatch](), 3},
		{"type with offstage", ByType[*swatch](FindOpts{IncludeOffstage: true}), 4},
		{"widget", ByWidget(a), 1},
		{"predicate", ByPredicate("wide", func(w widgets.Widget) bool { s, ok := w.(*swatch); return ok && s.width == 30 }), 1},
		{"element predicate", ByElementPredicate("a row", func(e widgets.Element) bool { _, ok := e.Widget().(*row); return ok }), 2},
		{"icon", ByIcon("star"), 1},
		{"icon, no match", ByIcon("moon"), 0},
		{"tooltip", ByTooltip("Tip"), 1},
		{"first", First(ByType[*swatch]()), 1},
		{"last", Last(ByType[*swatch]()), 1},
		{"at", At(ByType[*swatch](), 1), 1},
		{"at, out of range", At(ByType[*swatch](), 9), 0},
		{"descendant", Descendant(ByType[*row](), ByType[*swatch]()), 3},
		{"descendant, the root is not counted", Descendant(ByType[*row](), ByType[*row]()), 1},
		{"descendant, the root counted", Descendant(ByType[*row](), ByType[*row](), DescendantOpts{MatchRoot: true}), 2},
		{"ancestor", Ancestor(ByText("Hello"), ByType[*row]()), 1},
		{"ancestor, none", Ancestor(ByText("Hello"), ByType[*counter]()), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ts.Count(tt.f); got != tt.want {
				t.Errorf("Count(%s) = %d, want %d", tt.f.Describe(), got, tt.want)
			}
			if got := ts.Exists(tt.f); got != (tt.want > 0) {
				t.Errorf("Exists(%s) = %v", tt.f.Describe(), got)
			}
		})
	}
}

func TestEverythingFindsTheOffstageOnlyOnRequest(t *testing.T) {
	ts, _ := finderTree(t)
	if ts.Count(Everything(FindOpts{IncludeOffstage: true})) <= ts.Count(Everything()) {
		t.Errorf("Everything with offstage did not find more elements")
	}
}

func TestFinderOrderIsDepthFirst(t *testing.T) {
	ts, a := finderTree(t)
	ws := ts.Widgets(ByType[*swatch]())
	if len(ws) != 3 || ws[0] != widgets.Widget(a) {
		t.Errorf("widgets = %v", ws)
	}
}

func TestQueries(t *testing.T) {
	ts, _ := finderTree(t)
	// The row lays the swatches out from the left edge: a, then b.
	tests := []struct {
		name string
		got  ui.Offset
		want ui.Offset
	}{
		{"top left of a", ts.TopLeft(ByText("Hello")), ui.Offset{}},
		{"top left of b", ts.TopLeft(ByText("World")), ui.Offset{Dx: 20}},
		{"center of a", ts.Center(ByText("Hello")), ui.Offset{Dx: 10, Dy: 10}},
		{"bottom right of b", ts.BottomRight(ByText("World")), ui.Offset{Dx: 50, Dy: 20}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
	if got := ts.Size(ByText("World")); got != (ui.Size{Width: 30, Height: 20}) {
		t.Errorf("Size = %v", got)
	}
	if _, ok := ts.RenderObject(ByText("World")).(rendering.Box); !ok {
		t.Errorf("RenderObject is not a box")
	}
	if got := WidgetAs[*swatch](ts, ByText("Hello")); got.text != "Hello" {
		t.Errorf("WidgetAs = %v", got)
	}
	if s := StateOf[*counterState](ts, ByType[*counter]()); s.taps != 0 {
		t.Errorf("taps = %d", s.taps)
	}
}

func TestFinderFailures(t *testing.T) {
	tests := []struct {
		name  string
		do    func(ts *Tester)
		route string
	}{
		{"no match", func(ts *Tester) { ts.Element(ByText("Nope")) }, "FinderNoMatch"},
		{"widget, no match", func(ts *Tester) { ts.Widget(ByText("Nope")) }, "FinderNoMatch"},
		{"many matches", func(ts *Tester) { ts.Element(ByType[*swatch]()) }, "FinderManyMatches"},
		{"wrong widget type", func(ts *Tester) { WidgetAs[*row](ts, ByText("Hello")) }, "FinderWrongType"},
		{"not stateful", func(ts *Tester) { StateOf[*counterState](ts, ByText("Hello")) }, "FinderNotStateful"},
		{"wrong state type", func(ts *Tester) { StateOf[*fakeState](ts, ByType[*counter]()) }, "FinderWrongType"},
		{"not a box", func(ts *Tester) {
			ts.Center(First(ByElementPredicate("the render view", func(e widgets.Element) bool {
				_, ok := e.RenderObject().(*rendering.RenderView)
				return ok
			})))
		}, "FinderNotBox"},
		{"top left, no match", func(ts *Tester) { ts.TopLeft(ByText("Nope")) }, "FinderNoMatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := failure(t, func(ts *Tester) {
				a := newSwatch(grey, 20, 20)
				a.text = "Hello"
				ts.PumpWidget(newRow(a, newSwatch(grey, 5, 5), newCounter()))
				tt.do(ts)
			})
			requireRoute(t, appErr, tt.route)
		})
	}
}

// fakeState is a state type that no widget of the tests has.
type fakeState struct{ widgets.StateOf[*counter] }

func (*fakeState) Build(widgets.BuildContext) widgets.Widget { return nil }

func TestMatchers(t *testing.T) {
	ts, _ := finderTree(t)
	tests := []struct {
		name      string
		err       error
		wantRoute string
	}{
		{"nothing, ok", FindsNothing(ts, ByText("Nope")), ""},
		{"nothing, fails", FindsNothing(ts, ByText("Hello")), "FindsNothingFailed"},
		{"one widget, ok", FindsOneWidget(ts, ByText("Hello")), ""},
		{"one widget, none", FindsOneWidget(ts, ByText("Nope")), "FindsNWidgetsFailed"},
		{"one widget, two", FindsOneWidget(ts, ByType[*swatch]()), "FindsNWidgetsFailed"},
		{"widgets, ok", FindsWidgets(ts, ByType[*swatch]()), ""},
		{"widgets, none", FindsWidgets(ts, ByText("Nope")), "FindsAtLeastNWidgetsFailed"},
		{"n widgets, ok", FindsNWidgets(ts, ByType[*swatch](), 3), ""},
		{"n widgets, fails", FindsNWidgets(ts, ByType[*swatch](), 2), "FindsNWidgetsFailed"},
		{"at least n, ok", FindsAtLeastNWidgets(ts, ByType[*swatch](), 1), ""},
		{"at least n, fails", FindsAtLeastNWidgets(ts, ByType[*swatch](), 4), "FindsAtLeastNWidgetsFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantRoute == "" {
				if tt.err != nil {
					t.Fatalf("unexpected failure: %v", tt.err)
				}
				return
			}
			appErr := requireTagged(t, tt.err)
			requireRoute(t, appErr, tt.wantRoute)
		})
	}
}
