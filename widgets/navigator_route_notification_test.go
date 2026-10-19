package widgets_test

import (
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from
// packages/flutter/test/widgets/route_notification_messages_test.dart. The mock
// handler of SystemChannels.navigation is the hooks of the binding's
// SystemNavigator. The provider's reports are also tested in router_test.go.

// navLog records the calls of the SystemNavigator.
type navLog struct{ calls []routeUpdate }

func hookNavigator(ts *gluttertest.Tester) *navLog {
	l := &navLog{}
	n := ts.Binding.SystemNavigator()
	n.OnSelectSingleEntryHistory = func() { l.calls = append(l.calls, routeUpdate{call: "selectSingleEntryHistory"}) }
	n.OnRouteInformationUpdated = func(u *url.URL, state any, replace bool) {
		l.calls = append(l.calls, routeUpdate{"routeInformationUpdated", u.String(), state, replace})
	}
	return l
}

func (l *navLog) take() []routeUpdate {
	out := l.calls
	l.calls = nil
	return out
}

func routeUpdated(uri string) routeUpdate {
	return routeUpdate{"routeInformationUpdated", uri, nil, false}
}

// onTapPage is OnTapPage.
func onTapPage(id string, onTap func()) Widget {
	return NewGestureDetector(NewCenter(NewText(id)), GestureDetectorOpts{OnTap: onTap, Behavior: rendering.HitTestBehaviorOpaque})
}

func TestNavigatorPushAndPopSendPlatformMessages(t *testing.T) {
	ts := gluttertest.New(t)
	log := hookNavigator(ts)
	ts.PumpWidget(navTestApp(navTestAppOpts{routes: map[string]WidgetBuilder{
		"/":  func(bc BuildContext) Widget { return onTapPage("/", func() { NavigatorPushNamed(bc, "/A") }) },
		"/A": func(bc BuildContext) Widget { return onTapPage("A", func() { NavigatorPop(bc) }) },
	}}))
	want := []routeUpdate{{call: "selectSingleEntryHistory"}, routeUpdated("/")}
	if got := log.take(); !slices.Equal(got, want) {
		t.Fatalf("log = %v, want %v", got, want)
	}
	ts.Tap(gluttertest.ByText("/"))
	ts.Pump()
	ts.Pump(time.Second)
	if got := log.take(); !slices.Equal(got, []routeUpdate{routeUpdated("/A")}) {
		t.Fatalf("log after the push = %v", got)
	}
	ts.Tap(gluttertest.ByText("A"))
	ts.Pump()
	ts.Pump(time.Second)
	if got := log.take(); !slices.Equal(got, []routeUpdate{routeUpdated("/")}) {
		t.Errorf("log after the pop = %v", got)
	}
}

func TestNavigatorDoesNotReportRouteNameByDefault(t *testing.T) {
	ts := gluttertest.New(t)
	log := hookNavigator(ts)
	pages := func(names ...string) Widget {
		list := make([]Page, len(names))
		for i, n := range names {
			list[i] = newTestPage(n, n)
		}
		return NewDirectionality(ui.TextDirectionLTR, NewNavigator(NavigatorOpts{
			Pages:     list,
			OnPopPage: func(Route, any) bool { return false },
		}))
	}
	ts.PumpWidget(pages("/"))
	if got := log.take(); len(got) != 0 {
		t.Fatalf("log = %v, want none", got)
	}
	ts.PumpWidget(pages("/", "/abc"))
	ts.PumpAndSettle()
	if got := log.take(); len(got) != 0 {
		t.Errorf("log = %v, want none", got)
	}
}

func TestNavigatorReplaceSendsPlatformMessages(t *testing.T) {
	ts := gluttertest.New(t)
	log := hookNavigator(ts)
	ts.PumpWidget(navTestApp(navTestAppOpts{routes: map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget { return onTapPage("/", func() { NavigatorPushNamed(bc, "/A") }) },
		"/A": func(bc BuildContext) Widget {
			return onTapPage("A", func() { NavigatorPushReplacementNamed(bc, "/B") })
		},
		"/B": func(BuildContext) Widget { return onTapPage("B", func() {}) },
	}}))
	log.take()
	ts.Tap(gluttertest.ByText("/"))
	ts.Pump()
	ts.Pump(time.Second)
	if got := log.take(); !slices.Equal(got, []routeUpdate{routeUpdated("/A")}) {
		t.Fatalf("log after the push = %v", got)
	}
	ts.Tap(gluttertest.ByText("A"))
	ts.Pump()
	ts.Pump(time.Second)
	if got := log.take(); !slices.Equal(got, []routeUpdate{routeUpdated("/B")}) {
		t.Errorf("log after the replacement = %v", got)
	}
}

func TestNavigatorNamelessRoutesSendNoPlatformMessages(t *testing.T) {
	ts := gluttertest.New(t)
	log := hookNavigator(ts)
	ts.PumpWidget(navTestApp(navTestAppOpts{
		initialRoute: "/home",
		routes: map[string]WidgetBuilder{
			"/": func(BuildContext) Widget { return NewSizedBoxShrink(nil) },
			"/home": func(bc BuildContext) Widget {
				return onTapPage("Home", func() {
					// A route with no name.
					NavigatorPush(bc, NewPageRouteBuilder(func(BuildContext, animation.Animation[float64], animation.Animation[float64]) Widget {
						return NewText("Nameless Route")
					}))
				})
			},
		},
	}))
	want := []routeUpdate{{call: "selectSingleEntryHistory"}, routeUpdated("/home")}
	if got := log.take(); !slices.Equal(got, want) {
		t.Fatalf("log = %v, want %v", got, want)
	}
	ts.Tap(gluttertest.ByText("Home"))
	ts.Pump()
	ts.Pump(time.Second)
	if got := log.take(); len(got) != 0 {
		t.Errorf("log = %v, want none", got)
	}
}

func TestPlatformRouteInformationProviderReportsURLInAWidgetsAppRouter(t *testing.T) {
	ts := gluttertest.New(t)
	var log []routeUpdate
	logSystemNavigator(ts, &log)
	provider := NewPlatformRouteInformationProvider(ts.Binding, uriInfo("initial"))
	defer provider.Dispose()
	delegate := newSimpleDelegate(func(_ BuildContext, i *routeInfo) Widget { return infoText(i) }, nil)
	defer delegate.Dispose()
	delegate.reportConfiguration = true
	delegate.routeInformation = uriInfo("initial")
	ts.PumpWidget(NewWidgetsAppRouter(ui.NewColor(0xFFFFFFFF), WidgetsAppRouterOpts[*routeInfo]{
		RouteInformationProvider: provider, RouteInformationParser: newSimpleParser(), RouterDelegate: delegate,
	}))
	if !shows(ts, "initial") {
		t.Fatal("the initial route information should be shown")
	}
	if want := updated("initial", nil, false); !slices.Equal(log, want) {
		t.Fatalf("log = %v, want %v", log, want)
	}
	log = nil
	// Triggers a router rebuild and verifies that the route information is reported.
	delegate.setRouteInformation(stateInfo("update", "state"))
	ts.Pump()
	if !shows(ts, "update") {
		t.Fatal("the new route information should be shown")
	}
	if want := updated("update", "state", false); !slices.Equal(log, want) {
		t.Errorf("log = %v, want %v", log, want)
	}
}
