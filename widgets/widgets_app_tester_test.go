package widgets_test

import (
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// testWidgetsAppOpts are the parameters of testWidgetsApp (TestWidgetsApp).
type testWidgetsAppOpts struct {
	navigatorKey       AnyGlobalKey
	home               Widget
	initialRoute       string
	onGenerateRoute    RouteFactory
	navigatorObservers []NavigatorObserver
	routes             map[string]WidgetBuilder
	color              ui.Color
	textStyle          *painting.TextStyle
	pageRouteBuilder   PageRouteFactory
	builder            TransitionBuilder
	shortcuts          ShortcutMap
	actions            ActionMap
}

// defaultTestPageRoute is the page route builder of TestWidgetsApp: no
// transition.
func defaultTestPageRoute(settings RouteSettings, builder WidgetBuilder) PageRoute {
	return NewPageRouteBuilder(
		func(bc BuildContext, _, _ animation.Animation[float64]) Widget { return builder(bc) },
		PageRouteBuilderOpts{
			PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}},
			TransitionsBuilder: func(_ BuildContext, _, _ animation.Animation[float64], child Widget) Widget {
				return child
			},
		})
}

// testWidgetsApp is TestWidgetsApp: a minimal WidgetsApp for widget tests, whose
// page routes have no transition.
func testWidgetsApp(opts ...testWidgetsAppOpts) Widget {
	var o testWidgetsAppOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.color == (ui.Color{}) {
		o.color = ui.NewColor(0xFFFFFFFF)
	}
	if o.pageRouteBuilder == nil {
		o.pageRouteBuilder = defaultTestPageRoute
	}
	return NewWidgetsApp(o.color, WidgetsAppOpts{
		TextStyle:          o.textStyle,
		NavigatorKey:       o.navigatorKey,
		NavigatorObservers: o.navigatorObservers,
		Home:               o.home,
		InitialRoute:       o.initialRoute,
		OnGenerateRoute:    o.onGenerateRoute,
		Routes:             o.routes,
		PageRouteBuilder:   o.pageRouteBuilder,
		Builder:            o.builder,
		Shortcuts:          o.shortcuts,
		Actions:            o.actions,
	})
}
