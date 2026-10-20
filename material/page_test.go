package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/material/page_test.dart. The
// cases of the Cupertino transition and of the back gestures (iOS and Android),
// of the rasterized transitions (there is no SnapshotWidget), and the one of the
// restoration are not ported; the cases of the escape key and of the first page
// use the pages of Material and a Focus instead of the Scaffold.

// transitionDetector is TransitionDetector: it records that the delegate was
// asked to decide.
type transitionDetector struct {
	widgets.DefaultTransitionDelegate
	hasTransition bool
}

func (d *transitionDetector) Resolve(history []widgets.RouteTransitionRecord, exiting map[widgets.RouteTransitionRecord]widgets.RouteTransitionRecord,
	pageless map[widgets.RouteTransitionRecord][]widgets.RouteTransitionRecord) []widgets.RouteTransitionRecord {
	d.hasTransition = true
	return d.DefaultTransitionDelegate.Resolve(history, exiting, pageless)
}

// pagesApp is buildNavigator: a navigator of the pages inside the localizations.
func pagesApp(pages []widgets.Page, detector widgets.TransitionDelegate) widgets.Widget {
	return widgets.NewLocalizations(ui.NewLocale("en", "US"),
		[]widgets.LocalizationsDelegate{DefaultMaterialLocalizationsDelegate, widgets.DefaultWidgetsLocalizationsDelegate},
		widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewNavigator(widgets.NavigatorOpts{
			Pages:              pages,
			OnPopPage:          func(widgets.Route, any) bool { panic("the test should never pop a page") },
			TransitionDelegate: detector,
		})))
}

func TestMaterialPage(t *testing.T) {
	t.Run("works", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := foundation.NewValueKey("page")
		detector := &transitionDetector{}
		ts.PumpWidget(pagesApp([]widgets.Page{NewMaterialPage(widgets.NewText("first"), MaterialPageOpts{PageOpts: widgets.PageOpts{Key: key}})}, detector))
		if detector.hasTransition || ts.Count(gluttertest.ByText("first")) != 1 {
			t.Fatalf("transition %v, texts %d", detector.hasTransition, ts.Count(gluttertest.ByText("first")))
		}
		ts.PumpWidget(pagesApp([]widgets.Page{NewMaterialPage(widgets.NewText("second"), MaterialPageOpts{PageOpts: widgets.PageOpts{Key: key}})}, detector))
		// There is no transition, because the page has the same key, and the content updates.
		if detector.hasTransition {
			t.Error("a page with the same key should not transition")
		}
		if ts.Count(gluttertest.ByText("first")) != 0 || ts.Count(gluttertest.ByText("second")) != 1 {
			t.Error("the content should have been updated")
		}
	})

	t.Run("can toggle MaintainState", func(t *testing.T) {
		ts := gluttertest.New(t)
		keyOne, keyTwo := foundation.NewValueKey("one"), foundation.NewValueKey("two")
		detector := &transitionDetector{}
		pages := func(maintain bool) []widgets.Page {
			return []widgets.Page{
				NewMaterialPage(widgets.NewText("first"), MaterialPageOpts{PageOpts: widgets.PageOpts{Key: keyOne}, MaintainState: foundation.Some(maintain)}),
				NewMaterialPage(widgets.NewText("second"), MaterialPageOpts{PageOpts: widgets.PageOpts{Key: keyTwo}}),
			}
		}
		ts.PumpWidget(pagesApp(pages(false), detector))
		if detector.hasTransition || ts.Count(gluttertest.ByText("first", includeOffstage)) != 0 || ts.Count(gluttertest.ByText("second")) != 1 {
			t.Fatal("page one should not maintain its state")
		}
		ts.PumpWidget(pagesApp(pages(true), detector))
		if detector.hasTransition {
			t.Error("a page with the same key should not transition")
		}
		if ts.Count(gluttertest.ByText("first", includeOffstage)) != 1 || ts.Count(gluttertest.ByText("second")) != 1 {
			t.Error("page one should have been built")
		}
	})

	t.Run("does not lose its state when transitioning out", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := widgets.NewGlobalKey[*widgets.NavigatorState]()
		var host *keepsStateState
		ts.PumpWidget(&keepsState{StatefulWidget: widgets.NewStatefulWidgetBase(), key: key, capture: func(s *keepsStateState) { host = s }})
		if ts.Count(gluttertest.ByText("subpage")) != 1 || ts.Count(gluttertest.ByText("home")) != 0 {
			t.Fatal("the subpage should be shown")
		}
		navigator, _ := key.CurrentState()
		navigator.Pop()
		ts.Pump()
		if ts.Count(gluttertest.ByText("subpage")) != 1 || ts.Count(gluttertest.ByText("home")) != 1 {
			t.Error("the subpage should still be there, with the home page below it")
		}
		if host.subpage != "" {
			t.Error("the page was not removed from the pages")
		}
	})

	t.Run("creates a route with its settings", func(t *testing.T) {
		ts := gluttertest.New(t)
		page := NewMaterialPage(widgets.NewText("x"), MaterialPageOpts{PageOpts: widgets.PageOpts{Name: "named"}, FullscreenDialog: true})
		ts.PumpWidget(pagesApp([]widgets.Page{page}, nil))
		route, ok := page.CreateRoute(nil).(MaterialRoute)
		if !ok {
			t.Fatal("the route is not a MaterialRoute")
		}
		if !route.FullscreenDialog() || route.Settings().Name() != "named" || !route.MaintainState() {
			t.Errorf("fullscreen %v, name %q, maintainState %v", route.FullscreenDialog(), route.Settings().Name(), route.MaintainState())
		}
		if route.DebugLabel() == "" {
			t.Error("the route has no debug label")
		}
	})
}

// keepsState is KeepsStateTestWidget: a MaterialApp whose navigator has a home
// page and a subpage, and removes the subpage when it is popped.
type keepsState struct {
	widgets.StatefulWidget
	key     *widgets.GlobalKey[*widgets.NavigatorState]
	capture func(*keepsStateState)
}

func (w *keepsState) CreateState() widgets.State { return &keepsStateState{subpage: "subpage"} }

type keepsStateState struct {
	widgets.StateOf[*keepsState]
	subpage string
}

func (s *keepsStateState) Build(widgets.BuildContext) widgets.Widget {
	s.Widget().capture(s)
	pages := []widgets.Page{NewMaterialPage(widgets.NewText("home"), MaterialPageOpts{PageOpts: widgets.PageOpts{Key: foundation.NewValueKey("home")}})}
	if s.subpage != "" {
		pages = append(pages, NewMaterialPage(widgets.NewText(s.subpage), MaterialPageOpts{PageOpts: widgets.PageOpts{Key: foundation.NewValueKey("sub")}}))
	}
	return NewMaterialApp(MaterialAppOpts{Home: widgets.NewNavigator(widgets.NavigatorOpts{
		Key:   s.Widget().key,
		Pages: pages,
		OnPopPage: func(route widgets.Route, result any) bool {
			if !route.DidPop(result) {
				return false
			}
			s.SetState(func() { s.subpage = "" })
			return true
		},
	})})
}

func TestMaterialPageRoute(t *testing.T) {
	pushNext := func(ts *gluttertest.Tester, route func() widgets.Route) *widgets.NavigatorState {
		var navigator *widgets.NavigatorState
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			navigator = widgets.NavigatorOf(bc)
			return NewMaterial(widgets.NewAlign(widgets.NewText("Page 1")))
		})}))
		_ = route
		return navigator
	}

	t.Run("the zoom transition on the desktop", func(t *testing.T) {
		ts := gluttertest.New(t)
		route := func() widgets.Route {
			return NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget {
				return NewMaterial(widgets.NewAlign(widgets.NewText("Page 2")))
			})
		}
		navigator := pushNext(ts, route)
		initial := ts.TopLeft(gluttertest.ByText("Page 1"))
		navigator.Push(route())
		ts.Pump()
		ts.Pump(50 * time.Millisecond)

		// Page 1 is enlarging, from 1.0: its top left corner moves out of the view.
		page1 := ts.TopLeft(gluttertest.ByText("Page 1"))
		if page1.Dx >= initial.Dx || page1.Dy >= initial.Dy {
			t.Errorf("page 1 at %v, it should be enlarging from %v", page1, initial)
		}
		// Page 2 is enlarging from less than 1.0: its top left corner is inside the view.
		page2 := ts.TopLeft(gluttertest.ByText("Page 2"))
		if page2.Dx <= initial.Dx || page2.Dy <= initial.Dy {
			t.Errorf("page 2 at %v, it should be enlarging from less than 1.0", page2)
		}
		// Page 2 is fading in.
		opacity := func(text string) float64 {
			fade := gluttertest.Ancestor(gluttertest.ByText(text), gluttertest.ByType[*widgets.FadeTransition]())
			var least = 1.0
			for _, w := range ts.Widgets(fade) {
				if v := w.(*widgets.FadeTransition).Opacity().Value(); v < least {
					least = v
				}
			}
			return least
		}
		if got := opacity("Page 2"); got >= 1.0 {
			t.Errorf("page 2 has the opacity %v, it should be fading in", got)
		}

		ts.Pump(250 * time.Millisecond)
		ts.Pump(time.Millisecond)
		// Page 2 covers page 1.
		if ts.Count(gluttertest.ByText("Page 1")) != 0 || ts.Count(gluttertest.ByText("Page 2")) != 1 {
			t.Fatal("page 2 should cover page 1")
		}

		navigator.Pop()
		ts.Pump()
		ts.Pump(100 * time.Millisecond)
		// Page 1 is narrowing down but still larger than 1.0, page 2 is smaller than 1.0 and fading out.
		if page1 := ts.TopLeft(gluttertest.ByText("Page 1")); page1.Dx >= initial.Dx {
			t.Errorf("page 1 at %v, it should still be larger than at rest", page1)
		}
		if page2 := ts.TopLeft(gluttertest.ByText("Page 2")); page2.Dx <= initial.Dx {
			t.Errorf("page 2 at %v, it should be smaller than at rest", page2)
		}
		if got := opacity("Page 2"); got >= 1.0 {
			t.Errorf("page 2 has the opacity %v, it should be fading out", got)
		}
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("Page 1")) != 1 || ts.Count(gluttertest.ByText("Page 2")) != 0 {
			t.Error("page 1 should be back")
		}
		if got := ts.TopLeft(gluttertest.ByText("Page 1")); got != initial {
			t.Errorf("page 1 at %v, it should be back at %v", got, initial)
		}
	})

	t.Run("a fullscreen dialog does not transition the previous route", func(t *testing.T) {
		ts := gluttertest.New(t)
		var navigator *widgets.NavigatorState
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
			Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				navigator = widgets.NavigatorOf(bc)
				return NewMaterial(widgets.NewAlign(widgets.NewText("Page 1")))
			}),
		}))
		initial := ts.TopLeft(gluttertest.ByText("Page 1"))
		navigator.Push(NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewText("Page 2") },
			MaterialPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{FullscreenDialog: true}}))
		ts.Pump()
		ts.Pump(100 * time.Millisecond)
		if ts.Count(gluttertest.ByText("Page 2")) != 1 {
			t.Fatal("the second page transition should have started")
		}
		if got := ts.TopLeft(gluttertest.ByText("Page 1")); got != initial {
			t.Errorf("page 1 moved to %v from %v", got, initial)
		}
	})

	t.Run("a page route that is not Material does not transition the previous route", func(t *testing.T) {
		ts := gluttertest.New(t)
		prev := NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewText("x") })
		fullscreen := NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewText("y") },
			MaterialPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{FullscreenDialog: true}})
		plain := widgets.NewPageRouteBuilder(func(widgets.BuildContext, animation.Animation[float64], animation.Animation[float64]) widgets.Widget {
			return widgets.NewText("z")
		})
		_ = ts
		tests := []struct {
			name           string
			next           widgets.TransitionRoute
			transitionTo   bool
			transitionFrom bool
		}{
			{"a Material route", NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewText("a") }), true, true},
			{"a fullscreen dialog", fullscreen, false, false},
			{"a PageRouteBuilder, which has no delegated transition", plain, false, true},
		}
		for _, tt := range tests {
			if got := prev.CanTransitionTo(tt.next); got != tt.transitionTo {
				t.Errorf("%s: CanTransitionTo = %v, want %v", tt.name, got, tt.transitionTo)
			}
			if got := tt.next.CanTransitionFrom(prev); got != tt.transitionFrom {
				t.Errorf("%s: CanTransitionFrom = %v, want %v", tt.name, got, tt.transitionFrom)
			}
		}
	})

	t.Run("can be dismissed with the escape key", func(t *testing.T) {
		ts := gluttertest.New(t)
		var navigator *widgets.NavigatorState
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewMaterial(widgets.NewCenter(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			navigator = widgets.NavigatorOf(bc)
			return tapText("push", func() {
				navigator.Push(NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewCenter(widgets.NewText("route")) },
					MaterialPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{BarrierDismissible: true}}))
			})
		})))}))
		ts.Tap(gluttertest.ByText("push"))
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("route")) != 1 || ts.Count(gluttertest.ByText("push")) != 0 {
			t.Fatal("the route should cover the page")
		}
		ts.SendKeyEvent(services.LogicalKeyboardKeyEscape)
		ts.PumpAndSettle()
		if ts.Count(gluttertest.ByText("route")) != 0 {
			t.Error("the escape key should have dismissed the route")
		}
	})

	t.Run("requestFocus false does not request focus on the page", func(t *testing.T) {
		ts := gluttertest.New(t)
		var saved widgets.BuildContext
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			saved = bc
			return widgets.NewContainer(nil)
		})}))
		ts.Pump()
		navigator := widgets.NavigatorOf(saved)
		focused := func(request foundation.Opt[bool]) bool {
			navigator.Push(NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewText("Page Two") },
				MaterialPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{RequestFocus: request}}}))
			ts.Pump()
			ts.Pump(100 * time.Millisecond)
			has := widgets.FocusScopeOf(ts.Element(gluttertest.ByText("Page Two"))).HasFocus()
			navigator.Pop()
			ts.Pump()
			ts.Pump(100 * time.Millisecond)
			return has
		}
		if !focused(foundation.None[bool]()) {
			t.Error("the page should have the focus")
		}
		if focused(foundation.Some(false)) {
			t.Error("the page should not have the focus")
		}
	})
}

func TestMaterialRoutePlatformAdaptation(t *testing.T) {
	t.Run("the transition durations are the ones of the builder", func(t *testing.T) {
		tests := []struct {
			name    string
			theme   *PageTransitionsTheme
			want    time.Duration
			wantRev time.Duration
		}{
			{"zoom", nil, 300 * time.Millisecond, 300 * time.Millisecond},
			{"fade forwards", NewPageTransitionsTheme(PageTransitionsThemeOpts{Builders: map[foundation.TargetPlatform]widgets.PageTransitionsBuilder{
				foundation.TargetPlatformLinux: NewFadeForwardsPageTransitionsBuilder()}}), 450 * time.Millisecond, 450 * time.Millisecond},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ts := gluttertest.New(t)
				var route *MaterialPageRoute
				var navigator *widgets.NavigatorState
				ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
					Theme: NewThemeData(ThemeDataOpts{PageTransitionsTheme: tt.theme, Platform: foundation.Some(foundation.TargetPlatformLinux)}),
					Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
						navigator = widgets.NavigatorOf(bc)
						return widgets.NewSizedBox(nil)
					})}))
				route = NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) })
				navigator.Push(route)
				ts.Pump()
				if got := route.Controller().Duration.Value(); got != tt.want {
					t.Errorf("controller duration = %v, want %v", got, tt.want)
				}
				if got := route.TransitionDuration(); got != tt.want {
					t.Errorf("TransitionDuration = %v, want %v", got, tt.want)
				}
				if got := route.ReverseTransitionDuration(); got != tt.wantRev {
					t.Errorf("ReverseTransitionDuration = %v, want %v", got, tt.wantRev)
				}
				navigator.Pop()
				ts.Pump()
				if got := route.Controller().ReverseDuration.Value(); got != tt.wantRev {
					t.Errorf("controller reverse duration = %v, want %v", got, tt.wantRev)
				}
				ts.PumpAndSettle()
			})
		}
	})
}
