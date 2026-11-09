package cupertino_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/route_test.dart: the modal popup, the dialog,
// the page, the transitions builder and the cases of a fullscreen route and of a pop during a back swipe.

// routeObserver counts the routes of a type that are pushed, and keeps the settings of the last one.
type routeObserver struct {
	widgets.NavigatorObserverBase
	popups, dialogs int
	lastName        string
}

func (o *routeObserver) DidPush(route, _ widgets.Route) {
	switch route.(type) {
	case *cupertino.CupertinoModalPopupRoute:
		o.popups++
	}
	if _, ok := route.(*widgets.RawDialogRoute); ok {
		o.dialogs++
	}
	o.lastName = route.Settings().Name()
}

func tapToShow(show func(bc widgets.BuildContext)) widgets.Widget {
	return cupertino.NewCupertinoPageScaffold(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return widgets.NewGestureDetector(widgets.NewText("tap"), widgets.GestureDetectorOpts{OnTap: func() { show(bc) }})
	}))
}

func lastBarrier(ts *gluttertest.Tester) *widgets.ModalBarrier {
	barriers := ts.Widgets(gluttertest.ByType[*widgets.ModalBarrier]())
	return barriers[len(barriers)-1].(*widgets.ModalBarrier)
}

func TestShowCupertinoModalPopup(t *testing.T) {
	nested := func(root, inner *routeObserver, show func(bc widgets.BuildContext)) widgets.Widget {
		return widgets.NewNavigator(widgets.NavigatorOpts{Observers: []widgets.NavigatorObserver{inner},
			OnGenerateRoute: func(widgets.RouteSettings) widgets.Route {
				return widgets.NewPageRouteBuilder(func(bc widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
					return widgets.NewGestureDetector(widgets.NewText("tap"), widgets.GestureDetectorOpts{OnTap: func() { show(bc) }})
				})
			}})
	}
	empty := func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) }

	for _, tt := range []struct {
		name                string
		notUseRoot          bool
		wantRoot, wantInner int
	}{{"it uses the root navigator by default", false, 1, 0}, {"it uses the nested navigator when it is asked to", true, 0, 1}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			root, inner := &routeObserver{}, &routeObserver{}
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{NavigatorObservers: []widgets.NavigatorObserver{root},
				Home: nested(root, inner, func(bc widgets.BuildContext) {
					cupertino.ShowCupertinoModalPopup(bc, empty, cupertino.ShowCupertinoModalPopupOpts{NotUseRootNavigator: tt.notUseRoot})
				})}))
			ts.Tap(gluttertest.ByText("tap"))
			if root.popups != tt.wantRoot || inner.popups != tt.wantInner {
				t.Errorf("root navigator %d popups (want %d), nested navigator %d (want %d)", root.popups, tt.wantRoot, inner.popups, tt.wantInner)
			}
		})
	}

	t.Run("it passes the route settings to the route", func(t *testing.T) {
		ts := gluttertest.New(t)
		observer := &routeObserver{}
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{NavigatorObservers: []widgetsObserver{observer}, Home: tapToShow(func(bc widgets.BuildContext) {
			cupertino.ShowCupertinoModalPopup(bc, empty, cupertino.ShowCupertinoModalPopupOpts{
				RouteSettings: widgets.NewRouteSettings(widgets.RouteSettingsOpts{Name: "/modal"})})
		})}))
		ts.Tap(gluttertest.ByText("tap"))
		if observer.lastName != "/modal" {
			t.Errorf("route name = %q, want /modal", observer.lastName)
		}
	})

	t.Run("the barrier color", func(t *testing.T) {
		for _, tt := range []struct {
			name  string
			color ui.ColorValue
			want  foundation.Opt[ui.Color]
		}{
			{"transparent is no color", ui.NewColor(0x00000000), foundation.None[ui.Color]()},
			{"the default is the gray of Cupertino", nil, foundation.Some(ui.NewColor(0x33000000))},
			{"a custom color", ui.NewColor(0x11223344), foundation.Some(ui.NewColor(0x11223344))},
		} {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: tapToShow(func(bc widgets.BuildContext) {
				cupertino.ShowCupertinoModalPopup(bc, empty, cupertino.ShowCupertinoModalPopupOpts{BarrierColor: tt.color})
			})}))
			ts.Tap(gluttertest.ByText("tap"))
			ts.PumpAndSettle()
			if got := lastBarrier(ts).Color(); got != tt.want {
				t.Errorf("%s: the barrier color is %v, want %v", tt.name, got, tt.want)
			}
		}
	})

	t.Run("the barrier is dark in dark mode", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{
			Theme: testTheme(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessDark}),
			Home:  tapToShow(func(bc widgets.BuildContext) { cupertino.ShowCupertinoModalPopup(bc, empty) })}))
		ts.Tap(gluttertest.ByText("tap"))
		ts.PumpAndSettle()
		if got, want := lastBarrier(ts).Color(), foundation.Some(ui.NewColor(0x7A000000)); got != want {
			t.Errorf("the barrier color is %v, want %v", got, want)
		}
	})

	for _, tt := range []struct {
		name         string
		notDismiss   bool
		wantVisible  int
		tapOnTheEdge bool
	}{{"a tap on the barrier dismisses the popup", false, 0, true}, {"a tap on the barrier does not dismiss it when it is not dismissible", true, 1, true}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: tapToShow(func(bc widgets.BuildContext) {
				cupertino.ShowCupertinoModalPopup(bc, func(widgets.BuildContext) widgets.Widget { return widgets.NewText("Visible") },
					cupertino.ShowCupertinoModalPopupOpts{NotBarrierDismissible: tt.notDismiss})
			})}))
			ts.Tap(gluttertest.ByText("tap"))
			ts.PumpAndSettle()
			ts.TapAt(ts.TopLeft(gluttertest.Ancestor(gluttertest.ByText("tap"), gluttertest.ByType[*cupertino.CupertinoPageScaffold]())))
			ts.PumpAndSettle()
			if got := ts.Count(gluttertest.ByText("Visible")); got != tt.wantVisible {
				t.Errorf("%d popups are shown, want %d", got, tt.wantVisible)
			}
		})
	}

	t.Run("the popup slides up and goes away with the value it is popped with", func(t *testing.T) {
		ts := gluttertest.New(t)
		var result <-chan any
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: tapToShow(func(bc widgets.BuildContext) {
			result = cupertino.ShowCupertinoModalPopup(bc, func(widgets.BuildContext) widgets.Widget { return widgets.NewText("Visible") })
		})}))
		ts.Tap(gluttertest.ByText("tap"))
		ts.Pump()
		ts.Pump(100 * time.Millisecond)
		if dy := ts.TopLeft(gluttertest.ByText("Visible")).Dy; dy <= 0 || dy >= 600 {
			t.Errorf("the popup is at dy %v: it should be on its way up", dy)
		}
		ts.PumpAndSettle()
		widgets.NavigatorOf(navigatorContext(ts), widgets.NavigatorOfOpts{RootNavigator: true}).Pop("done")
		ts.PumpAndSettle()
		if got := <-result; got != "done" {
			t.Errorf("the popup was popped with %v, want done", got)
		}
	})
}

type widgetsObserver = widgets.NavigatorObserver

// navigatorContext is the context of the navigator of the app.
func navigatorContext(ts *gluttertest.Tester) widgets.BuildContext {
	return ts.Element(gluttertest.ByType[*widgets.Navigator]())
}

func TestShowCupertinoDialog(t *testing.T) {
	for _, tt := range []struct {
		name       string
		notUseRoot bool
		wantRoot   int
		wantInner  int
	}{{"it uses the root navigator by default", false, 1, 0}, {"it uses the nested navigator when it is asked to", true, 0, 1}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			root, inner := &routeObserver{}, &routeObserver{}
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{NavigatorObservers: []widgetsObserver{root},
				Home: widgets.NewNavigator(widgets.NavigatorOpts{Observers: []widgetsObserver{inner},
					OnGenerateRoute: func(widgets.RouteSettings) widgets.Route {
						return widgets.NewPageRouteBuilder(func(bc widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
							return widgets.NewGestureDetector(widgets.NewText("tap"), widgets.GestureDetectorOpts{OnTap: func() {
								cupertino.ShowCupertinoDialog(bc, func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) },
									cupertino.CupertinoDialogRouteOpts{NotUseRootNavigator: tt.notUseRoot})
							}})
						})
					}})}))
			ts.Tap(gluttertest.ByText("tap"))
			if root.dialogs != tt.wantRoot || inner.dialogs != tt.wantInner {
				t.Errorf("root navigator %d dialogs (want %d), nested navigator %d (want %d)", root.dialogs, tt.wantRoot, inner.dialogs, tt.wantInner)
			}
		})
	}

	t.Run("the dialog has the barrier of Cupertino, and a tap on it dismisses the dialog only when it can", func(t *testing.T) {
		for _, dismissible := range []bool{false, true} {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: tapToShow(func(bc widgets.BuildContext) {
				cupertino.ShowCupertinoDialog(bc, func(widgets.BuildContext) widgets.Widget { return widgets.NewCenter(widgets.NewText("Visible")) },
					cupertino.CupertinoDialogRouteOpts{BarrierDismissible: dismissible})
			})}))
			ts.Tap(gluttertest.ByText("tap"))
			ts.PumpAndSettle()
			if got := lastBarrier(ts).Color(); got != foundation.Some(ui.NewColor(0x33000000)) {
				t.Errorf("the barrier color is %v", got)
			}
			ts.TapAt(ui.Offset{Dx: 780, Dy: 580})
			ts.PumpAndSettle()
			if shown := ts.Exists(gluttertest.ByText("Visible")); shown == dismissible {
				t.Errorf("dismissible %v: the dialog shown is %v", dismissible, shown)
			}
		}
	})
}

func TestCupertinoPageRoutePages(t *testing.T) {
	detector := func() *transitionDetector { return &transitionDetector{} }
	app := func(pages []widgets.Page, d widgets.TransitionDelegate) widgets.Widget {
		return widgets.NewLocalizations(ui.NewLocale("en", "US"),
			[]widgets.LocalizationsDelegate{cupertino.DefaultCupertinoLocalizationsDelegate, widgets.DefaultWidgetsLocalizationsDelegate},
			widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewNavigator(widgets.NavigatorOpts{
				Pages: pages, OnPopPage: func(widgets.Route, any) bool { panic("the test should never pop a page") }, TransitionDelegate: d})))
	}
	t.Run("a page works", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := foundation.NewValueKey("page")
		d := detector()
		ts.PumpWidget(app([]widgets.Page{cupertino.NewCupertinoPage(widgets.NewText("first"), cupertino.CupertinoPageOpts{PageOpts: widgets.PageOpts{Key: key}, Title: foundation.Some("title one")})}, d))
		if d.hasTransition || ts.Count(gluttertest.ByText("first")) != 1 {
			t.Fatalf("transition %v, texts %d", d.hasTransition, ts.Count(gluttertest.ByText("first")))
		}
		ts.PumpWidget(app([]widgets.Page{cupertino.NewCupertinoPage(widgets.NewText("second"), cupertino.CupertinoPageOpts{PageOpts: widgets.PageOpts{Key: key}, Title: foundation.Some("title two")})}, d))
		// There is no transition, because the page has the same key, and the content updates.
		if d.hasTransition {
			t.Error("a page with the same key should not transition")
		}
		if ts.Count(gluttertest.ByText("first")) != 0 || ts.Count(gluttertest.ByText("second")) != 1 {
			t.Error("the content should have been updated")
		}
	})

	t.Run("a page can toggle MaintainState", func(t *testing.T) {
		ts := gluttertest.New(t)
		keyOne, keyTwo := foundation.NewValueKey("one"), foundation.NewValueKey("two")
		d := detector()
		pages := func(maintain bool) []widgets.Page {
			return []widgets.Page{
				cupertino.NewCupertinoPage(widgets.NewText("first"), cupertino.CupertinoPageOpts{PageOpts: widgets.PageOpts{Key: keyOne}, MaintainState: foundation.Some(maintain)}),
				cupertino.NewCupertinoPage(widgets.NewText("second"), cupertino.CupertinoPageOpts{PageOpts: widgets.PageOpts{Key: keyTwo}}),
			}
		}
		ts.PumpWidget(app(pages(false), d))
		if d.hasTransition || ts.Count(gluttertest.ByText("first", includeOffstage)) != 0 || ts.Count(gluttertest.ByText("second")) != 1 {
			t.Fatal("page one should not maintain its state")
		}
		ts.PumpWidget(app(pages(true), d))
		if d.hasTransition {
			t.Error("a page with the same key should not transition")
		}
		if ts.Count(gluttertest.ByText("first", includeOffstage)) != 1 || ts.Count(gluttertest.ByText("second")) != 1 {
			t.Error("page one should have been built")
		}
	})

	t.Run("a page makes a route with its settings", func(t *testing.T) {
		page := cupertino.NewCupertinoPage(widgets.NewText("x"), cupertino.CupertinoPageOpts{PageOpts: widgets.PageOpts{Name: "named"}, FullscreenDialog: true, Title: foundation.Some("t")})
		route, ok := page.CreateRoute(nil).(cupertino.CupertinoRoute)
		if !ok {
			t.Fatal("the route is not a CupertinoRoute")
		}
		defer route.Dispose()
		if !route.FullscreenDialog() || route.Settings().Name() != "named" || !route.MaintainState() || route.Title() != foundation.Some("t") {
			t.Errorf("fullscreen %v, name %q, maintainState %v, title %v", route.FullscreenDialog(), route.Settings().Name(), route.MaintainState(), route.Title())
		}
		if route.DelegatedTransition() != nil {
			t.Error("a fullscreen dialog has no delegated transition")
		}
	})
}

// transitionDetector records that the delegate was asked to decide.
type transitionDetector struct {
	widgets.DefaultTransitionDelegate
	hasTransition bool
}

func (d *transitionDetector) Resolve(history []widgets.RouteTransitionRecord, exiting map[widgets.RouteTransitionRecord]widgets.RouteTransitionRecord,
	pageless map[widgets.RouteTransitionRecord][]widgets.RouteTransitionRecord) []widgets.RouteTransitionRecord {
	d.hasTransition = true
	return d.DefaultTransitionDelegate.Resolve(history, exiting, pageless)
}

var includeOffstage = gluttertest.FindOpts{IncludeOffstage: true}

func TestCupertinoFullscreenRouteDoesNotTransitionThePreviousRoute(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{
		InitialRoute: "/",
		OnGenerateRoute: func(settings widgets.RouteSettings) widgets.Route {
			if settings.Name() == "/" {
				return widgets.NewPageRouteBuilder(func(widgets.BuildContext, animation.Animation[float64], animation.Animation[float64]) widgets.Widget {
					return cupertino.NewCupertinoPageScaffold(widgets.NewText("Page 1"))
				})
			}
			return cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget {
				return cupertino.NewCupertinoPageScaffold(widgets.NewText("Page 2"))
			}, cupertino.CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{FullscreenDialog: true}})
		},
	}))
	if ts.Exists(gluttertest.ByText("Page 2")) {
		t.Fatal("page 2 should not be shown yet")
	}
	dx := ts.TopLeft(gluttertest.ByText("Page 1")).Dx
	navigatorState(ts).PushNamed("/next")
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	if !ts.Exists(gluttertest.ByText("Page 2")) {
		t.Fatal("the second page transition should have started")
	}
	if got := ts.TopLeft(gluttertest.ByText("Page 1")).Dx; got != dx {
		t.Errorf("page 1 moved from %v to %v", dx, got)
	}
}

func TestCupertinoPoppingARouteDuringABackSwipeDoesNotCrash(t *testing.T) {
	ts := gluttertest.New(t)
	route := cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget {
		return cupertino.NewCupertinoPageScaffold(widgets.NewCenter(widgets.NewText("child")))
	})
	var navigator *widgets.NavigatorState
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return button("Home", func() { navigator = widgets.NavigatorOf(bc); navigator.Push(route) })
	}))}))
	ts.Tap(gluttertest.ByText("Home"))
	ts.PumpAndSettle()
	gesture := ts.StartGesture(ui.Offset{Dx: 3, Dy: 300})
	// Two events form a valid drag.
	ts.Pump(100 * time.Millisecond)
	gesture.MoveByAt(ui.Offset{Dx: 27}, 100*time.Millisecond)
	ts.Pump(200 * time.Millisecond)
	gesture.MoveByAt(ui.Offset{Dx: 20}, 200*time.Millisecond)
	// Pause a while so that the route is popped when the drag is canceled.
	ts.Pump(1000 * time.Millisecond)
	gesture.MoveByAt(ui.Offset{Dx: 1}, 1200*time.Millisecond)
	// Remove the drag.
	navigator.RemoveRoute(route)
	ts.Pump()
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("the framework reported %v", errs)
	}
}

func TestCupertinoPageTransitionsBuilder(t *testing.T) {
	t.Run("it builds a CupertinoPageTransition", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{
			Theme: testTheme(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessLight}),
			Routes: map[string]widgets.WidgetBuilder{
				"/": func(bc widgets.BuildContext) widgets.Widget {
					return cupertino.NewCupertinoPageScaffold(button("push", func() { widgets.NavigatorOf(bc).PushNamed("/b") }))
				},
				"/b": func(widgets.BuildContext) widgets.Widget {
					return cupertino.NewCupertinoPageScaffold(widgets.NewText("page b"))
				},
			}}))
		if n := ts.Count(gluttertest.ByType[*cupertino.CupertinoPageTransition]()); n != 1 {
			t.Fatalf("%d transitions, want 1", n)
		}
		ts.Tap(gluttertest.ByText("push"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("page b")) || ts.Count(gluttertest.ByType[*cupertino.CupertinoPageTransition]()) != 1 {
			t.Error("page b should be shown with one transition")
		}
	})
	t.Run("it has the duration and the delegated transition of Cupertino", func(t *testing.T) {
		var b cupertino.CupertinoPageTransitionsBuilder
		if b.TransitionDuration() != 500*time.Millisecond {
			t.Errorf("duration = %v, want 500ms", b.TransitionDuration())
		}
		if b.DelegatedTransition() == nil {
			t.Error("the builder has no delegated transition")
		}
	})
}

func TestCupertinoRoutePreviousTitleBeforeInstall(t *testing.T) {
	skipUnlessDebug(t)
	route := cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) })
	defer route.Dispose()
	err := recoverError(func() { route.PreviousTitle() })
	requireErrorTag(t, err, "CupertinoRouteBase.PreviousTitle.CupertinoRoutePreviousTitleBeforeInstall")
}
