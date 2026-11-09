package cupertino_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// slimPage is a page of a list under a sliver bar with a large title: the bar is the object of the goldens.
func slimPage(title string, middle bool) widgets.Widget {
	var items []widgets.Widget
	for i := 0; i < 20; i++ {
		items = append(items, widgets.NewSizedBox(widgets.NewColoredBox(ui.NewColor(0xFFFFFFFF-uint32(i)*0x050505), widgets.NewText("Row")),
			widgets.SizedBoxOpts{Height: foundation.Some(50.0)}))
	}
	opts := cupertino.CupertinoSliverNavigationBarOpts{LargeTitle: widgets.NewText(title)}
	if middle {
		opts.Middle = widgets.NewText(title)
	}
	return cupertino.NewCupertinoPageScaffold(widgets.NewCustomScrollView([]widgets.Widget{
		cupertino.NewCupertinoSliverNavigationBar(opts), widgets.NewSliverList(widgets.NewSliverChildListDelegate(items)),
	}))
}

// TestCupertinoNavigationBarGoldens draws the sliver bar with its large title expanded, collapsed, and a bar
// to bar flight in the middle of a push.
func TestCupertinoNavigationBarGoldens(t *testing.T) {
	size := ui.Size{Width: 390, Height: 600}
	t.Run("cupertino.navbar.large.expanded", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t)
		ts.SetViewSize(size)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: slimPage("Title", true)}))
		ts.MatchesGolden("cupertino.navbar.large.expanded")
	})
	t.Run("cupertino.navbar.large.collapsed", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t)
		ts.SetViewSize(size)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: slimPage("Title", true)}))
		ts.DragFrom(ui.Offset{Dx: 200, Dy: 400}, ui.Offset{Dy: -200})
		ts.PumpAndSettle()
		ts.MatchesGolden("cupertino.navbar.large.collapsed")
	})
	t.Run("cupertino.navbar.flight", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t)
		ts.SetViewSize(size)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: slimPage("Inbox", true)}))
		navigatorState(ts).Push(cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget { return slimPage("Message", true) },
			cupertino.CupertinoPageRouteOpts{Title: foundation.Some("Inbox")}))
		ts.Pump()
		ts.Pump(120 * time.Millisecond)
		ts.MatchesGolden("cupertino.navbar.flight")
	})
}

func TestCupertinoSliverNavigationBarSearch(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.SetViewSize(ui.Size{Width: 390, Height: 600})
	var taps []bool
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCustomScrollView([]widgets.Widget{
		cupertino.NewCupertinoSliverNavigationBarSearch(widgets.NewColoredBox(ui.NewColor(0xFFDDDDDD), widgets.NewText("Search field")),
			cupertino.CupertinoSliverNavigationBarOpts{LargeTitle: widgets.NewText("Title"), OnSearchableBottomTap: func(active bool) { taps = append(taps, active) }}),
		widgets.NewSliverToBoxAdapter(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(1000.0)})),
	}))}))
	// The field is under the large title; a tap on it starts the search.
	ts.Tap(gluttertest.ByText("Search field"))
	ts.PumpAndSettle()
	if len(taps) != 1 || !taps[0] {
		t.Fatalf("taps = %v, want [true]", taps)
	}
	if !ts.Exists(gluttertest.ByText("Cancel")) {
		t.Fatal("the Cancel button is not shown while the search is active")
	}
	ts.Tap(gluttertest.ByText("Cancel"))
	ts.PumpAndSettle()
	if len(taps) != 2 || taps[1] {
		t.Errorf("taps = %v, want [true false]", taps)
	}
}
