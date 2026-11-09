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

// These tests are ported from packages/flutter/test/cupertino/nav_bar_transition_test.dart (the flights of the
// bars between routes), with the checks of what Flutter measures on the elements of the flight made on the
// widgets that the test can reach (the flight itself is internal to the package).

// barPage is a page with a bar of the given title, and a button that pushes the next page.
func barPage(title string, next func(bc widgets.BuildContext), opts cupertino.CupertinoNavigationBarOpts) widgets.Widget {
	opts.Middle = widgets.NewText(title)
	return cupertino.NewCupertinoPageScaffold(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return widgets.NewCenter(widgets.NewGestureDetector(widgets.NewText("push "+title), widgets.GestureDetectorOpts{
			OnTap: func() { next(bc) }}))
	}), cupertino.CupertinoPageScaffoldOpts{NavigationBar: cupertino.NewCupertinoNavigationBar(opts)})
}

func pushPage(title string, opts cupertino.CupertinoNavigationBarOpts, routeTitle foundation.Opt[string]) func(bc widgets.BuildContext) {
	return func(bc widgets.BuildContext) {
		widgets.NavigatorPush(bc, cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget {
			return barPage(title, func(widgets.BuildContext) {}, opts)
		}, cupertino.CupertinoPageRouteOpts{Title: routeTitle}))
	}
}

func TestCupertinoNavigationBarTransition(t *testing.T) {
	t.Run("a push flies the bars and ends with the back button and its label", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: barPage("First",
			pushPage("Second", cupertino.CupertinoNavigationBarOpts{}, foundation.None[string]()), cupertino.CupertinoNavigationBarOpts{})}))
		nav := widgets.NavigatorOf(ts.Element(gluttertest.ByType[*cupertino.CupertinoPageScaffold]()))
		nav.Push(cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget {
			return barPage("Second", func(widgets.BuildContext) {}, cupertino.CupertinoNavigationBarOpts{})
		}, cupertino.CupertinoPageRouteOpts{}))
		ts.Pump()
		ts.Pump(150 * time.Millisecond) // In the middle of the flight.
		if err := ts.TakeErrors(); len(err) != 0 {
			t.Fatalf("errors in the flight: %v", err)
		}
		if got := len(ts.Widgets(gluttertest.ByType[*widgets.Hero]())); got < 2 {
			t.Errorf("%d heroes in the tree during the flight, want the two bars", got)
		}
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByType[*cupertino.CupertinoNavigationBarBackButton]()) {
			t.Error("the second page has no back button")
		}
		nav.Pop()
		ts.Pump()
		ts.Pump(150 * time.Millisecond)
		if err := ts.TakeErrors(); len(err) != 0 {
			t.Fatalf("errors in the flight back: %v", err)
		}
		ts.PumpAndSettle()
		if ts.Exists(gluttertest.ByType[*cupertino.CupertinoNavigationBarBackButton]()) {
			t.Error("the first page has a back button")
		}
	})

	t.Run("the back label is the title of the previous route, or Back when it is long", func(t *testing.T) {
		for _, tt := range []struct {
			title string
			want  string
		}{{"Short", "Short"}, {"A very long title", "Back"}} {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabView(cupertino.CupertinoTabViewOpts{
				Builder: func(widgets.BuildContext) widgets.Widget {
					return barPage("x", pushPage("Second", cupertino.CupertinoNavigationBarOpts{NotTransitionBetweenRoutes: true}, foundation.None[string]()),
						cupertino.CupertinoNavigationBarOpts{NotTransitionBetweenRoutes: true})
				},
				DefaultTitle: foundation.Some(tt.title),
			})}))
			ts.Tap(gluttertest.ByText("push x"))
			ts.PumpAndSettle()
			if !ts.Exists(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoNavigationBarBackButton](), gluttertest.ByText(tt.want))) {
				t.Errorf("title %q: the back label is not %q", tt.title, tt.want)
			}
		}
	})

	t.Run("the previous page title can be given to the bar", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: barPage("x",
			pushPage("Second", cupertino.CupertinoNavigationBarOpts{PreviousPageTitle: foundation.Some("Mine"), NotTransitionBetweenRoutes: true}, foundation.None[string]()),
			cupertino.CupertinoNavigationBarOpts{NotTransitionBetweenRoutes: true})}))
		ts.Tap(gluttertest.ByText("push x"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Mine")) {
			t.Error("the back label is not the title that was given")
		}
	})

	t.Run("a tap on the back button pops", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: barPage("First",
			pushPage("Second", cupertino.CupertinoNavigationBarOpts{NotTransitionBetweenRoutes: true}, foundation.Some("First")),
			cupertino.CupertinoNavigationBarOpts{NotTransitionBetweenRoutes: true})}))
		ts.Tap(gluttertest.ByText("push First"))
		ts.PumpAndSettle()
		ts.Tap(gluttertest.ByType[*cupertino.CupertinoNavigationBarBackButton]())
		ts.PumpAndSettle()
		if ts.Exists(gluttertest.ByText("Second")) || !ts.Exists(gluttertest.ByText("First")) {
			t.Error("the page was not popped")
		}
	})

	t.Run("the swipe back flies the bars along with the finger", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: barPage("First",
			pushPage("Second", cupertino.CupertinoNavigationBarOpts{}, foundation.None[string]()), cupertino.CupertinoNavigationBarOpts{})}))
		ts.Tap(gluttertest.ByText("push First"))
		ts.PumpAndSettle()
		gesture := ts.StartGesture(ui.Offset{Dx: 5, Dy: 300})
		gesture.MoveBy(ui.Offset{Dx: 200})
		ts.Pump()
		ts.Pump(50 * time.Millisecond)
		if err := ts.TakeErrors(); len(err) != 0 {
			t.Fatalf("errors in the flight of the swipe: %v", err)
		}
		gesture.MoveBy(ui.Offset{Dx: 400})
		gesture.Up()
		ts.PumpAndSettle()
		if err := ts.TakeErrors(); len(err) != 0 {
			t.Fatalf("errors after the swipe: %v", err)
		}
		if ts.Exists(gluttertest.ByText("Second")) {
			t.Error("the page was not popped by the swipe")
		}
	})

	t.Run("a full screen dialog has a Cancel button and no flight", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: barPage("First", func(bc widgets.BuildContext) {
			widgets.NavigatorPush(bc, cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget {
				return barPage("Dialog", func(widgets.BuildContext) {}, cupertino.CupertinoNavigationBarOpts{})
			}, cupertino.CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{FullscreenDialog: true}}))
		}, cupertino.CupertinoNavigationBarOpts{})}))
		ts.Tap(gluttertest.ByText("push First"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Cancel")) {
			t.Fatal("the dialog has no Cancel button")
		}
		if ts.Exists(gluttertest.ByType[*cupertino.CupertinoNavigationBarBackButton]()) {
			t.Error("the dialog has a back button")
		}
		ts.Tap(gluttertest.ByText("Cancel"))
		ts.PumpAndSettle()
		if ts.Exists(gluttertest.ByText("Dialog")) {
			t.Error("the dialog was not popped by Cancel")
		}
	})
}

func TestCupertinoNavigationBarBackButton(t *testing.T) {
	t.Run("the button calls onPressed in place of popping", func(t *testing.T) {
		ts := gluttertest.New(t)
		pressed := false
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoNavigationBarBackButton(
			cupertino.CupertinoNavigationBarBackButtonOpts{PreviousPageTitle: foundation.Some("Prev"), OnPressed: func() { pressed = true }}))}))
		if !ts.Exists(gluttertest.ByText("Prev")) {
			t.Fatal("the label is not shown")
		}
		ts.Tap(gluttertest.ByType[*cupertino.CupertinoNavigationBarBackButton]())
		if !pressed {
			t.Error("onPressed was not called")
		}
	})

	t.Run("a button that cannot pop and has no onPressed is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoNavigationBarBackButton())}))
		requireErrorTag(t, firstError(t, ts), "CupertinoNavigationBarBackButton.Build.NavigationBarBackButtonNotPoppable")
	})
}
