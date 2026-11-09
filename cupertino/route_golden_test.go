package cupertino_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// goldenPage is a page with a colored background and a big title, so that the transition shows what moves.
func goldenPage(title string, background ui.Color) widgets.Widget {
	return cupertino.NewCupertinoPageScaffold(widgets.NewCenter(widgets.NewColumn([]widgets.Widget{
		widgets.NewText(title, widgets.TextOpts{Style: &painting.TextStyle{FontFamily: "Roboto", FontSize: foundation.Some(40.0), Color: foundation.Some(ui.NewColor(0xFF000000))}}),
		widgets.NewText("a Cupertino page", widgets.TextOpts{Style: &painting.TextStyle{FontFamily: "Roboto", FontSize: foundation.Some(20.0), Color: foundation.Some(ui.NewColor(0xFF3C3C43))}}),
	}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})), cupertino.CupertinoPageScaffoldOpts{BackgroundColor: background})
}

// TestCupertinoRouteTransitionGoldens draws the push of a page at 30% and at 70% of its transition: the new page
// slides in with its shadow, and the old one moves a third of the way to the start.
func TestCupertinoRouteTransitionGoldens(t *testing.T) {
	for _, tt := range []struct {
		name string
		at   time.Duration
	}{{"cupertino.route.push.30", 150 * time.Millisecond}, {"cupertino.route.push.70", 350 * time.Millisecond}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: goldenPage("Page 1", ui.NewColor(0xFFF2F2F7))}))
			navigatorState(ts).Push(cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget {
				return goldenPage("Page 2", ui.NewColor(0xFFE5F1FF))
			}))
			ts.Pump()
			ts.Pump(tt.at)
			ts.MatchesGolden(tt.name)
		})
	}
}

// TestCupertinoSheetGoldens draws the sheet at rest over the page it covers (which shrinks and gets rounded
// corners behind it), and with a drag handle.
func TestCupertinoSheetGoldens(t *testing.T) {
	for _, tt := range []struct {
		name   string
		handle bool
	}{{"cupertino.sheet", false}, {"cupertino.sheet.handle", true}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: goldenPage("Page 1", ui.NewColor(0xFFF2F2F7))}))
			navigatorState(ts).Push(cupertino.NewCupertinoSheetRoute(cupertino.CupertinoSheetRouteOpts{ShowDragHandle: tt.handle,
				Builder: func(widgets.BuildContext) widgets.Widget { return goldenPage("Sheet", ui.NewColor(0xFFFFFFFF)) }}))
			ts.PumpAndSettle()
			ts.MatchesGolden(tt.name)
		})
	}
}
