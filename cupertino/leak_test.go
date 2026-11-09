package cupertino_test

import (
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The memory tests of the routes of Cupertino (step 34.2, SPEC §15 "Memory over time"): each scenario runs
// a step N times and 4N times and compares the live heap and the goroutines (internal/leaktest). The step
// leaves the app as it found it.

// leakApp pumps a Cupertino app whose home is a scaffold with the body, in a view of 800 x 600.
func leakApp(tb testing.TB, body widgets.Widget) *gluttertest.Tester {
	tb.Helper()
	ts := gluttertest.New(tb, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}, DiscardScenes: true})
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(body)}))
	ts.PumpAndSettle()
	return ts
}

// leakAppBC is leakApp with the build context of a widget in the body.
func leakAppBC(tb testing.TB, body func(bc widgets.BuildContext) widgets.Widget) (*gluttertest.Tester, *widgets.BuildContext) {
	tb.Helper()
	var bc widgets.BuildContext
	ts := leakApp(tb, widgets.NewBuilder(func(c widgets.BuildContext) widgets.Widget {
		bc = c
		return body(c)
	}))
	return ts, &bc
}

func TestLeaksRoutes(t *testing.T) {
	home := func(widgets.BuildContext) widgets.Widget { return widgets.NewText("HOME") }
	page := func(widgets.BuildContext) widgets.Widget {
		return cupertino.NewCupertinoPageScaffold(widgets.NewText("PAGE"))
	}
	tests := []struct {
		name string
		step func(tb testing.TB) (*gluttertest.Tester, func())
	}{
		{"push and pop a page", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, home)
			return ts, func() {
				widgets.NavigatorPush(*bc, cupertino.NewCupertinoPageRoute(page))
				ts.PumpAndSettle()
				widgets.NavigatorPop(*bc)
				ts.PumpAndSettle()
			}
		}},
		{"push a page and swipe it back", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, home)
			return ts, func() {
				widgets.NavigatorPush(*bc, cupertino.NewCupertinoPageRoute(page))
				ts.PumpAndSettle()
				ts.DragFrom(ui.Offset{Dx: 5, Dy: 300}, ui.Offset{Dx: 500})
				ts.PumpAndSettle()
			}
		}},
		{"push and pop a fullscreen dialog", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, home)
			return ts, func() {
				widgets.NavigatorPush(*bc, cupertino.NewCupertinoPageRoute(page, cupertino.CupertinoPageRouteOpts{PageRouteOpts: widgets.PageRouteOpts{FullscreenDialog: true}}))
				ts.PumpAndSettle()
				widgets.NavigatorPop(*bc)
				ts.PumpAndSettle()
			}
		}},
		{"show and close a sheet", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, home)
			return ts, func() {
				cupertino.ShowCupertinoSheet(*bc, cupertino.ShowCupertinoSheetOpts{ShowDragHandle: true, Builder: page})
				ts.PumpAndSettle()
				widgets.NavigatorPop(*bc)
				ts.PumpAndSettle()
			}
		}},
		{"show a sheet and drag it down", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, home)
			return ts, func() {
				cupertino.ShowCupertinoSheet(*bc, cupertino.ShowCupertinoSheetOpts{Builder: page})
				ts.PumpAndSettle()
				ts.DragFrom(ui.Offset{Dx: 100, Dy: 200}, ui.Offset{Dy: 350})
				ts.PumpAndSettle()
			}
		}},
		{"show a sheet over a sheet", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, home)
			return ts, func() {
				cupertino.ShowCupertinoSheet(*bc, cupertino.ShowCupertinoSheetOpts{Builder: page})
				ts.PumpAndSettle()
				cupertino.ShowCupertinoSheet(*bc, cupertino.ShowCupertinoSheetOpts{Builder: page})
				ts.PumpAndSettle()
				widgets.NavigatorPop(*bc)
				ts.PumpAndSettle()
				widgets.NavigatorPop(*bc)
				ts.PumpAndSettle()
			}
		}},
		{"show and close a modal popup", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, home)
			return ts, func() {
				cupertino.ShowCupertinoModalPopup(*bc, func(widgets.BuildContext) widgets.Widget {
					return widgets.NewSizedBox(widgets.NewText("popup"), widgets.SizedBoxOpts{Height: foundation.Some(200.0)})
				})
				ts.PumpAndSettle()
				widgets.NavigatorPop(*bc)
				ts.PumpAndSettle()
			}
		}},
		{"show and close a dialog", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, home)
			return ts, func() {
				cupertino.ShowCupertinoDialog(*bc, func(widgets.BuildContext) widgets.Widget { return widgets.NewCenter(widgets.NewText("dialog")) })
				ts.PumpAndSettle()
				widgets.NavigatorPop(*bc)
				ts.PumpAndSettle()
			}
		}},
		{"scroll a list with a scrollbar", func(tb testing.TB) (*gluttertest.Tester, func()) {
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformAndroid))
			if t, ok := tb.(*testing.T); ok {
				t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
			}
			ts := leakApp(tb, cupertino.NewCupertinoScrollbar(widgets.NewSingleChildScrollView(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(4000.0)}))))
			return ts, func() {
				ts.Drag(gluttertest.ByType[*widgets.SingleChildScrollView](), ui.Offset{Dy: -200})
				ts.PumpAndSettle()
				ts.Pump(2 * time.Second)
				ts.PumpAndSettle()
				ts.Drag(gluttertest.ByType[*widgets.SingleChildScrollView](), ui.Offset{Dy: 200})
				ts.PumpAndSettle()
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "cupertino/routes/"+tt.name, func() func() {
				ts, step := tt.step(t)
				return func() { step(); ts.ClearLogs() }
			}, leaktest.Opts{N: 25, SlackObjects: 150})
		})
	}
}

func TestSheetIsFreedAfterItsRouteIsPopped(t *testing.T) {
	ts, bc := leakAppBC(t, func(widgets.BuildContext) widgets.Widget { return widgets.NewText("HOME") })
	var pages []weak.Pointer[widgets.Text]
	for i := 0; i < 5; i++ {
		cupertino.ShowCupertinoSheet(*bc, cupertino.ShowCupertinoSheetOpts{Builder: func(widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoPageScaffold(widgets.NewText("PAGE"))
		}})
		ts.PumpAndSettle()
		pages = append(pages, weak.Make(ts.Widget(gluttertest.ByText("PAGE")).(*widgets.Text)))
		widgets.NavigatorPop(*bc)
		ts.PumpAndSettle()
	}
	runtime.GC()
	runtime.GC()
	for i, p := range pages {
		if p.Value() != nil {
			t.Errorf("the page of the sheet %d is still alive after its route was popped", i)
		}
	}
}

// TestLeaksNavigation is the memory test of the navigation bars and the tabs (step 34.3): the flights of the
// bars on a push, a pop and a swipe back, the sliver bar that scrolls, and the tab scaffold that switches tabs
// and pushes pages in a tab.
func TestLeaksNavigation(t *testing.T) {
	barPage := func(title string) func(widgets.BuildContext) widgets.Widget {
		return func(widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoPageScaffold(widgets.NewText("PAGE"), cupertino.CupertinoPageScaffoldOpts{
				NavigationBar: cupertino.NewCupertinoNavigationBar(cupertino.CupertinoNavigationBarOpts{Middle: widgets.NewText(title)})})
		}
	}
	tests := []struct {
		name string
		step func(tb testing.TB) (*gluttertest.Tester, func())
	}{
		{"push and pop a page with a bar", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts := gluttertest.New(tb, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}, DiscardScenes: true})
			var bc widgets.BuildContext
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(c widgets.BuildContext) widgets.Widget {
				bc = c
				return barPage("Home")(c)
			})}))
			ts.PumpAndSettle()
			return ts, func() {
				widgets.NavigatorPush(bc, cupertino.NewCupertinoPageRoute(barPage("Next")))
				ts.PumpAndSettle()
				widgets.NavigatorPop(bc)
				ts.PumpAndSettle()
			}
		}},
		{"push a page with a bar and swipe it back", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts := gluttertest.New(tb, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}, DiscardScenes: true})
			var bc widgets.BuildContext
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(c widgets.BuildContext) widgets.Widget {
				bc = c
				return barPage("Home")(c)
			})}))
			ts.PumpAndSettle()
			return ts, func() {
				widgets.NavigatorPush(bc, cupertino.NewCupertinoPageRoute(barPage("Next")))
				ts.PumpAndSettle()
				ts.DragFrom(ui.Offset{Dx: 5, Dy: 300}, ui.Offset{Dx: 500})
				ts.PumpAndSettle()
			}
		}},
		{"switch the tabs and push a page in a tab", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts := gluttertest.New(tb, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}, DiscardScenes: true})
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabScaffold(cupertino.NewCupertinoTabBar(tabItems("One", "Two")),
				func(_ widgets.BuildContext, i int) widgets.Widget {
					return cupertino.NewCupertinoTabView(cupertino.CupertinoTabViewOpts{Builder: func(widgets.BuildContext) widgets.Widget {
						return barPage("Tab")(nil)
					}})
				})}))
			ts.PumpAndSettle()
			return ts, func() {
				ts.Tap(gluttertest.ByText("Two"))
				ts.PumpAndSettle()
				nav := widgets.NavigatorOf(ts.Element(gluttertest.ByType[*cupertino.CupertinoNavigationBar]()))
				nav.Push(cupertino.NewCupertinoPageRoute(barPage("Pushed")))
				ts.PumpAndSettle()
				nav.Pop()
				ts.PumpAndSettle()
				ts.Tap(gluttertest.ByText("One"))
				ts.PumpAndSettle()
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "cupertino/navigation/"+tt.name, func() func() {
				ts, step := tt.step(t)
				return func() { step(); ts.ClearLogs() }
			}, leaktest.Opts{N: 25, SlackObjects: 150})
		})
	}
}
