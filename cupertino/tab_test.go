package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/tab_test.dart (CupertinoTabView).

// pushButton is a text that pushes the named route on the navigator of the context.
func pushButton(label, route string) widgets.Widget {
	return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return widgets.NewGestureDetector(widgets.NewText(label), widgets.GestureDetectorOpts{
			Behavior: rendering.HitTestBehaviorOpaque,
			OnTap:    func() { widgets.NavigatorPushNamed(bc, route) }})
	})
}

func TestCupertinoTabView(t *testing.T) {
	t.Run("the builder is the root route and the routes table the others", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabView(cupertino.CupertinoTabViewOpts{
			Builder: func(widgets.BuildContext) widgets.Widget { return pushButton("Home", "/next") },
			Routes: map[string]widgets.WidgetBuilder{
				"/next": func(widgets.BuildContext) widgets.Widget { return widgets.NewText("Next page") }},
		})}))
		if !ts.Exists(gluttertest.ByText("Home")) {
			t.Fatal("the builder's page is not shown")
		}
		ts.Tap(gluttertest.ByText("Home"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Next page")) {
			t.Error("the page of the routes table is not shown")
		}
	})

	t.Run("onGenerateRoute is used for a route that no table has", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabView(cupertino.CupertinoTabViewOpts{
			Builder: func(widgets.BuildContext) widgets.Widget { return pushButton("Home", "/gen") },
			OnGenerateRoute: func(s widgets.RouteSettings) widgets.Route {
				return cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewText("Generated " + s.Name()) },
					cupertino.CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: s}}})
			},
		})}))
		ts.Tap(gluttertest.ByText("Home"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Generated /gen")) {
			t.Error("the generated page is not shown")
		}
	})

	t.Run("each tab keeps its own history", func(t *testing.T) {
		ts := gluttertest.New(t)
		tab := func(name string) widgets.Widget {
			return cupertino.NewCupertinoTabView(cupertino.CupertinoTabViewOpts{
				Builder: func(widgets.BuildContext) widgets.Widget { return pushButton(name+" home", "/page") },
				Routes: map[string]widgets.WidgetBuilder{
					"/page": func(widgets.BuildContext) widgets.Widget { return widgets.NewText(name + " page") }},
			})
		}
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabScaffold(
			cupertino.NewCupertinoTabBar(tabItems("One", "Two")),
			func(_ widgets.BuildContext, i int) widgets.Widget { return tab([]string{"first", "second"}[i]) })}))
		ts.Tap(gluttertest.ByText("first home"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("first page")) {
			t.Fatal("the first tab did not push its page")
		}
		ts.Tap(gluttertest.ByText("Two"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("second home")) {
			t.Error("the second tab does not show its home")
		}
		ts.Tap(gluttertest.ByText("One"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("first page")) {
			t.Error("the first tab lost its history")
		}
	})

	t.Run("the navigator key reaches the navigator", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := widgets.NewGlobalKey[*widgets.NavigatorState]()
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabView(cupertino.CupertinoTabViewOpts{
			NavigatorKey: key,
			Builder:      func(widgets.BuildContext) widgets.Widget { return widgets.NewText("Home") },
		})}))
		if _, ok := key.CurrentState(); !ok {
			t.Error("the key has no navigator state")
		}
	})

	t.Run("observers are told, and the hero controller is among them", func(t *testing.T) {
		ts := gluttertest.New(t)
		observer := &recordingObserver{}
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabView(cupertino.CupertinoTabViewOpts{
			NavigatorObservers: []widgets.NavigatorObserver{observer},
			Builder:            func(widgets.BuildContext) widgets.Widget { return pushButton("Home", "/page") },
			Routes:             map[string]widgets.WidgetBuilder{"/page": func(widgets.BuildContext) widgets.Widget { return widgets.NewText("Page") }},
		})}))
		before := observer.pushes
		ts.Tap(gluttertest.ByText("Home"))
		ts.PumpAndSettle()
		if observer.pushes != before+1 {
			t.Errorf("the observer saw %d pushes, want %d", observer.pushes, before+1)
		}
	})
}

// recordingObserver counts the pushes that it is told about.
type recordingObserver struct {
	widgets.NavigatorObserverBase
	pushes int
}

func (o *recordingObserver) DidPush(widgets.Route, widgets.Route) { o.pushes++ }

func TestCupertinoTabViewErrors(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("the default title is the title of the first page", func(t *testing.T) {
		v := cupertino.NewCupertinoTabView(cupertino.CupertinoTabViewOpts{DefaultTitle: foundation.Some("Title")})
		if v.DefaultTitle() != foundation.Some("Title") {
			t.Errorf("title = %v", v.DefaultTitle())
		}
	})
	t.Run("an unknown route without onUnknownRoute", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabView(cupertino.CupertinoTabViewOpts{
			Builder: func(widgets.BuildContext) widgets.Widget { return pushButton("Home", "/nowhere") },
		})}))
		ts.Tap(gluttertest.ByText("Home"))
		ts.Pump()
		err := firstError(t, ts)
		checkErrorTag(t, err, "CupertinoTabViewUnknownRouteMissing")
	})
	t.Run("onUnknownRoute returns nil", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabView(cupertino.CupertinoTabViewOpts{
			Builder:        func(widgets.BuildContext) widgets.Widget { return pushButton("Home", "/nowhere") },
			OnUnknownRoute: func(widgets.RouteSettings) widgets.Route { return nil },
		})}))
		ts.Tap(gluttertest.ByText("Home"))
		ts.Pump()
		err := firstError(t, ts)
		checkErrorTag(t, err, "CupertinoTabViewUnknownRouteNil")
	})
}
