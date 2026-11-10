package cupertino_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The tests of CupertinoExpansionTile, ported from cupertino/expansion_tile_test.dart (step 34.8).

const (
	expansionDuration    = 250 * time.Millisecond
	infinitesimalElapsed = time.Microsecond
)

func expansionApp(child widgets.Widget) widgets.Widget {
	return testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(child)})
}

func expansionTile(title, content string, opts ...cupertino.CupertinoExpansionTileOpts) *cupertino.CupertinoExpansionTile {
	return cupertino.NewCupertinoExpansionTile(widgets.NewText(title), widgets.NewText(content), opts...)
}

// pumpExpansion pumps the widget and one frame more: the font of the chevron arrives after the first frame and invalidates the
// layout of every text, which a tap must not hit before it is laid out again.
func pumpExpansion(ts *gluttertest.Tester, w widgets.Widget) {
	ts.PumpWidget(w)
	ts.Pump()
}

func count(ts *gluttertest.Tester, text string) int { return ts.Count(gluttertest.ByText(text)) }

func TestCupertinoExpansionTile(t *testing.T) {
	t.Run("toggles the expansion on a tap", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		pumpExpansion(ts, expansionApp(expansionTile("Title", "Content")))
		if n := count(ts, "Content"); n != 0 {
			t.Fatalf("%d contents while collapsed", n)
		}
		ts.Tap(gluttertest.ByText("Title"))
		ts.Pump()
		// One is in the tree, hidden while the other fades in the overlay.
		if n := count(ts, "Content"); n != 2 {
			t.Errorf("%d contents while expanding, want 2", n)
		}
		ts.Tap(gluttertest.ByText("Title"))
		ts.Pump()
		if n := count(ts, "Content"); n != 0 {
			t.Errorf("%d contents after the collapse", n)
		}
	})
	t.Run("can be controlled by an ExpansibleController", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		controller := widgets.NewExpansibleController()
		defer controller.Dispose()
		pumpExpansion(ts, expansionApp(expansionTile("Title", "Content", cupertino.CupertinoExpansionTileOpts{Controller: controller})))
		if controller.IsExpanded() || count(ts, "Content") != 0 {
			t.Fatal("the tile is not collapsed at the start")
		}
		controller.Expand()
		ts.Pump()
		if !controller.IsExpanded() || count(ts, "Content") != 1 {
			t.Errorf("expanded %v, %d contents", controller.IsExpanded(), count(ts, "Content"))
		}
		controller.Collapse()
		ts.Pump()
		if controller.IsExpanded() || count(ts, "Content") != 0 {
			t.Errorf("expanded %v, %d contents", controller.IsExpanded(), count(ts, "Content"))
		}
	})
	t.Run("a tap collapses a tile that the controller expanded", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		controller := widgets.NewExpansibleController()
		defer controller.Dispose()
		pumpExpansion(ts, expansionApp(expansionTile("Title", "Content", cupertino.CupertinoExpansionTileOpts{Controller: controller})))
		controller.Expand()
		ts.Pump()
		ts.Tap(gluttertest.ByText("Title"))
		ts.Pump()
		if controller.IsExpanded() || count(ts, "Content") != 0 {
			t.Errorf("expanded %v, %d contents", controller.IsExpanded(), count(ts, "Content"))
		}
	})
	t.Run("nested tiles", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		pumpExpansion(ts, expansionApp(cupertino.NewCupertinoExpansionTile(widgets.NewText("Outer"), expansionTile("Inner", "Content"))))
		step := func(title string, wantContent int) {
			t.Helper()
			ts.Tap(gluttertest.ByText(title))
			ts.Pump()
			ts.Pump(expansionDuration + infinitesimalElapsed)
			if n := count(ts, "Content"); n != wantContent {
				t.Fatalf("after a tap on %s: %d contents, want %d", title, n, wantContent)
			}
		}
		step("Outer", 0)
		step("Inner", 1)
		step("Inner", 0)
		step("Outer", 0)
	})
	t.Run("the transition mode is fade by default", func(t *testing.T) {
		if got := expansionTile("T", "C").TransitionMode(); got != cupertino.ExpansionTileTransitionModeFade {
			t.Errorf("the mode is %v", got)
		}
	})
	t.Run("the icon turns a quarter as the tile expands", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		pumpExpansion(ts, expansionApp(expansionTile("Title", "Content")))
		turns := func() float64 {
			return ts.Widget(gluttertest.ByType[*widgets.RotationTransition]()).(*widgets.RotationTransition).Turns().Value()
		}
		if got := turns(); got != 0 {
			t.Fatalf("turns at the start are %v", got)
		}
		ts.Tap(gluttertest.ByText("Title"))
		ts.PumpAndSettle()
		if got := turns(); got != 0.25 {
			t.Errorf("turns when expanded are %v, want 0.25", got)
		}
	})
	t.Run("semantics of the header", func(t *testing.T) {
		skipUnlessDebug(t)
		l := cupertino.DefaultCupertinoLocalizations{}
		for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformIOS, foundation.TargetPlatformAndroid} {
			t.Run(platform.String(), func(t *testing.T) {
				foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(platform))
				t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
				ts := dialogTester(t, ui.Size{})
				controller := widgets.NewExpansibleController()
				defer controller.Dispose()
				pumpExpansion(ts, testApp(cupertino.CupertinoAppOpts{Home: widgets.NewColumn([]widgets.Widget{
					expansionTile("First", "Content"),
					expansionTile("Second", "Content", cupertino.CupertinoExpansionTileOpts{Controller: controller}),
				})}))
				controller.Expand()
				ts.PumpAndSettle()
				// headerSemantics is the Semantics above the tile that has the hints of the tile.
				headerSemantics := func(i int) *widgets.Semantics {
					tile := gluttertest.At(gluttertest.ByType[*cupertino.CupertinoListTile](), i)
					for _, w := range ts.Widgets(gluttertest.Ancestor(tile, gluttertest.ByType[*widgets.Semantics]())) {
						if s := w.(*widgets.Semantics); s.Properties().HintOverrides != nil {
							return s
						}
					}
					t.Fatalf("no Semantics with hints above tile %d", i)
					return nil
				}
				first, second := headerSemantics(0), headerSemantics(1)
				if got := first.Properties().HintOverrides.OnTapHint; got != l.ExpansionTileCollapsedTapHint() {
					t.Errorf("the tap hint of the collapsed tile is %q", got)
				}
				if got := second.Properties().HintOverrides.OnTapHint; got != l.ExpansionTileExpandedTapHint() {
					t.Errorf("the tap hint of the expanded tile is %q", got)
				}
				wantFirst := foundation.None[string]()
				wantSecond := foundation.None[string]()
				if platform == foundation.TargetPlatformIOS {
					wantFirst = foundation.Some(l.ExpandedHint() + "\n " + l.ExpansionTileCollapsedHint())
					wantSecond = foundation.Some(l.CollapsedHint() + "\n " + l.ExpansionTileExpandedHint())
				}
				if first.Properties().Hint != wantFirst || second.Properties().Hint != wantSecond {
					t.Errorf("the hints are %v and %v, want %v and %v", first.Properties().Hint, second.Properties().Hint, wantFirst, wantSecond)
				}
			})
		}
	})
	t.Run("the tiles of a column have a tap action and their title as the label", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.EnsureSemantics()
		pumpExpansion(ts, testApp(cupertino.CupertinoAppOpts{Home: widgets.NewColumn([]widgets.Widget{
			cupertino.NewCupertinoExpansionTile(widgets.NewText("First Expansion Tile"), widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(50.0)})),
			cupertino.NewCupertinoExpansionTile(widgets.NewText("Second Expansion Tile"), widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(50.0)})),
		}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})}))
		ts.PumpAndSettle()
		for _, label := range []string{"First Expansion Tile", "Second Expansion Tile"} {
			n := pickerSemanticsNodeWithLabel(ts, label)
			if n == nil {
				t.Errorf("no node is labeled %q", label)
				continue
			}
			if !n.GetSemanticsData().HasAction(ui.SemanticsActionTap) {
				t.Errorf("%q has no tap action", label)
			}
		}
	})
}

func TestCupertinoExpansionTileGoldens(t *testing.T) {
	red := widgets.NewSizedBox(widgets.NewColoredBox(ui.NewColor(0xFFFF0000), nil), widgets.SizedBoxOpts{Width: foundation.Some(120.0), Height: foundation.Some(50.0)})
	for _, c := range []struct {
		name string
		mode cupertino.ExpansionTileTransitionMode
	}{{"default", cupertino.ExpansionTileTransitionModeFade}, {"scroll_mode", cupertino.ExpansionTileTransitionModeScroll}} {
		t.Run(c.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 400, Height: 200})
			pumpExpansion(ts, testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewRepaintBoundary(
				cupertino.NewCupertinoExpansionTile(widgets.NewText("Title"), red, cupertino.CupertinoExpansionTileOpts{TransitionMode: c.mode})))}))
			if c.mode == cupertino.ExpansionTileTransitionModeFade {
				ts.MatchesGolden("expansion_tile.default.collapsed")
			}
			ts.Tap(gluttertest.ByText("Title"))
			ts.Pump()
			ts.Pump(expansionDuration / 2)
			ts.MatchesGolden("expansion_tile." + c.name + ".forward")
			ts.PumpAndSettle()
			if c.mode == cupertino.ExpansionTileTransitionModeFade {
				ts.MatchesGolden("expansion_tile.default.expanded")
			}
			ts.Tap(gluttertest.ByText("Title"))
			ts.Pump()
			ts.Pump(expansionDuration / 2)
			ts.MatchesGolden("expansion_tile." + c.name + ".reverse")
			ts.PumpAndSettle()
		})
	}
}

// pickerSemanticsNodeWithLabel is the node of the semantics tree with the label, or nil.
func pickerSemanticsNodeWithLabel(ts *gluttertest.Tester, label string) *semantics.SemanticsNode {
	owner := ts.RenderObject(gluttertest.First(gluttertest.ByType[*cupertino.CupertinoListTile]())).AsRenderObject().Owner().SemanticsOwner()
	var found *semantics.SemanticsNode
	var walk func(n *semantics.SemanticsNode)
	walk = func(n *semantics.SemanticsNode) {
		if n.Label() == label {
			found = n
		}
		n.VisitChildren(func(c *semantics.SemanticsNode) bool { walk(c); return true })
	}
	walk(owner.RootSemanticsNode())
	return found
}
