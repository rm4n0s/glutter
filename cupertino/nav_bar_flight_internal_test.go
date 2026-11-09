package cupertino

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TestNavBarFlightIsBuilt checks that a push builds the transition widget of the flight, and that it is gone
// when the flight ends.
func TestNavBarFlightIsBuilt(t *testing.T) {
	ts := gluttertest.New(t)
	theme := NewCupertinoThemeData(CupertinoThemeDataOpts{TextTheme: NewCupertinoTextThemeData(CupertinoTextThemeDataOpts{
		TextStyle: &painting.TextStyle{FontFamily: "Roboto", FontSize: foundation.Some(17.0), Color: foundation.Some(ui.NewColor(0xFF000000))}})})
	page := func(title string) widgets.Widget {
		return NewCupertinoPageScaffold(widgets.NewText("content"), CupertinoPageScaffoldOpts{
			NavigationBar: NewCupertinoNavigationBar(CupertinoNavigationBarOpts{Middle: widgets.NewText(title)})})
	}
	ts.PumpWidget(NewCupertinoApp(CupertinoAppOpts{Theme: theme, DebugShowCheckedModeBanner: foundation.Some(false), Home: page("First")}))
	nav := widgets.NavigatorOf(ts.Element(gluttertest.ByType[*CupertinoPageScaffold]()))
	nav.Push(NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget { return page("Second") }))
	ts.Pump()
	ts.Pump(150 * time.Millisecond)
	if !ts.Exists(gluttertest.ByType[*navBarTransition]()) {
		t.Fatal("no bar transition in the tree during the push")
	}
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByType[*navBarTransition]()) {
		t.Error("the bar transition is still in the tree after the push")
	}
	nav.Pop()
	ts.Pump()
	ts.Pump(150 * time.Millisecond)
	if !ts.Exists(gluttertest.ByType[*navBarTransition]()) {
		t.Fatal("no bar transition in the tree during the pop")
	}
	ts.PumpAndSettle()
}

// fakeFlight is a widget that is marked as a flight between bars, to put a transitionable bar inside.
type fakeFlight struct {
	widgets.StatelessWidget
	child widgets.Widget
}

func (*fakeFlight) isNavBarFlight() {}

func (w *fakeFlight) Build(widgets.BuildContext) widgets.Widget { return w.child }

func transitionableForTest() *transitionableNavigationBar {
	return newTransitionableNavigationBar(transitionableNavigationBarOpts{componentsKeys: newNavBarComponentsKeys(), child: widgets.NewText("bar")})
}

// TestNavBarTransitionableErrors checks the two ways that a transitionable bar is misplaced: outside a hero,
// and inside a flight.
func TestNavBarTransitionableErrors(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("needs the debug-only checks of the framework")
	}
	for _, tt := range []struct {
		name  string
		child func() widgets.Widget
		route string
	}{
		{"outside a hero", func() widgets.Widget { return transitionableForTest() },
			"transitionableNavigationBar.Build.NavigationBarTransitionableOutsideHero"},
		{"inside a flight", func() widgets.Widget {
			return &fakeFlight{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), widgets.NewHero("tag", transitionableForTest())}
		}, "transitionableNavigationBar.Build->ElementBase.VisitAncestorElements->transitionableNavigationBar.Build.func1.NavigationBarTransitionableInFlight"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, tt.child()))
			errs := ts.TakeErrors()
			if len(errs) == 0 {
				t.Fatal("no error was reported")
			}
			inner := errors.Unwrap(errs[0])
			if inner == nil {
				inner = errs[0]
			}
			appErr, ok := errors.FromError(inner)
			if !ok || !appErr.HasRoute(tt.route) {
				t.Errorf("route %v (ok %v), want %s", func() string {
					if ok {
						return appErr.Route()
					}
					return ""
				}(), ok, tt.route)
			}
		})
	}
}
