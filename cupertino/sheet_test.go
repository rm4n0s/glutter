package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/sheet_test.dart. A GestureDetector over a Text
// stands in for the CupertinoButton (step 34.4). Not ported: the cases that need the navigation bar
// (step 34.3), and the restoration cases.

const sheetTopGapRatio = 0.08

// sheetApp is an app whose page pushes the sheet that the opts make.
type sheetApp struct {
	home, sheet *widgets.GlobalKey[widgets.State]
}

func newSheetApp(t *testing.T, ts *gluttertest.Tester, sheetContent func(widgets.BuildContext, widgets.ScrollController) widgets.Widget, open func(bc widgets.BuildContext)) *sheetApp {
	t.Helper()
	a := &sheetApp{home: widgets.NewGlobalKey[widgets.State](), sheet: widgets.NewGlobalKey[widgets.State]()}
	if open == nil {
		open = func(bc widgets.BuildContext) {
			cupertino.ShowCupertinoSheet(bc, cupertino.ShowCupertinoSheetOpts{ScrollableBuilder: sheetContent})
		}
	}
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(
		widgets.NewCenter(widgets.NewColumn([]widgets.Widget{widgets.NewText("Page 1"),
			button("Push Page 2", func() { open(a.home.CurrentContext()) })})),
		cupertino.CupertinoPageScaffoldOpts{Key: a.home})}))
	return a
}

// pageScaffold is a page scaffold with the sheet's key.
func (a *sheetApp) pageScaffold(child widgets.Widget) widgets.Widget {
	return cupertino.NewCupertinoPageScaffold(child, cupertino.CupertinoPageScaffoldOpts{Key: a.sheet})
}

func (a *sheetApp) sheetY(ts *gluttertest.Tester) float64 {
	return ts.TopLeft(gluttertest.ByKey(a.sheet)).Dy
}

func TestCupertinoSheetRoute(t *testing.T) {
	pushRoute := func(ts *gluttertest.Tester, route func() widgets.Route) *sheetApp {
		return newSheetApp(t, ts, nil, func(bc widgets.BuildContext) { widgets.NavigatorPush(bc, route()) })
	}
	page := func(label string) widgets.WidgetBuilder {
		return func(widgets.BuildContext) widgets.Widget {
			return cupertino.NewCupertinoPageScaffold(widgets.NewText(label))
		}
	}

	t.Run("the sheet route does not cover the whole screen", func(t *testing.T) {
		ts := gluttertest.New(t)
		pushRoute(ts, func() widgets.Route {
			return cupertino.NewCupertinoSheetRoute(cupertino.CupertinoSheetRouteOpts{Builder: page("Page 2")})
		})
		if !ts.Exists(gluttertest.ByText("Page 1")) || ts.Exists(gluttertest.ByText("Page 2")) {
			t.Fatal("page 1 should be shown alone")
		}
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Page 2")) {
			t.Fatal("page 2 should be shown")
		}
		if dy := ts.TopLeft(gluttertest.Ancestor(gluttertest.ByText("Page 2"), gluttertest.ByType[*cupertino.CupertinoPageScaffold]())).Dy; dy <= 0 {
			t.Errorf("the sheet is at dy %v, want more than 0", dy)
		}
	})

	t.Run("the content does not go below the bottom of the screen", func(t *testing.T) {
		ts := gluttertest.New(t)
		pushRoute(ts, func() widgets.Route {
			return cupertino.NewCupertinoSheetRoute(cupertino.CupertinoSheetRouteOpts{Builder: func(widgets.BuildContext) widgets.Widget {
				return cupertino.NewCupertinoPageScaffold(widgets.NewContainer(nil))
			}})
		})
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		if got, want := ts.Size(gluttertest.ByType[*widgets.Container]()).Height, 600.0-600.0*sheetTopGapRatio; !near(got, want, 0.001) {
			t.Errorf("height = %v, want %v", got, want)
		}
	})

	t.Run("showDragHandle adds a drag handle to the top of the sheet", func(t *testing.T) {
		ts := gluttertest.New(t)
		pushRoute(ts, func() widgets.Route {
			return cupertino.NewCupertinoSheetRoute(cupertino.CupertinoSheetRouteOpts{Builder: page("Page 2"), ShowDragHandle: true})
		})
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		handles := ts.Count(dragHandle())
		if handles != 1 {
			t.Errorf("%d drag handles, want 1", handles)
		}
	})

	t.Run("showDragHandle adds a MediaQuery padding so content can render below the handle", func(t *testing.T) {
		ts := gluttertest.New(t)
		pushRoute(ts, func() widgets.Route {
			return cupertino.NewCupertinoSheetRoute(cupertino.CupertinoSheetRouteOpts{ShowDragHandle: true, Builder: func(widgets.BuildContext) widgets.Widget {
				return cupertino.NewCupertinoPageScaffold(widgets.NewSafeArea(widgets.NewText("Page 2")))
			}})
		})
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		handle := ts.TopLeft(dragHandle())
		content := ts.TopLeft(gluttertest.ByText("Page 2"))
		if content.Dy <= handle.Dy {
			t.Errorf("the content (%v) is not below the handle (%v)", content.Dy, handle.Dy)
		}
	})

	t.Run("the previous route moves slightly downward when a sheet is pushed", func(t *testing.T) {
		ts := gluttertest.New(t)
		pushRoute(ts, func() widgets.Route {
			return cupertino.NewCupertinoSheetRoute(cupertino.CupertinoSheetRouteOpts{Builder: page("Page 2")})
		})
		scaffold1 := gluttertest.Ancestor(gluttertest.ByText("Page 1"), gluttertest.ByType[*cupertino.CupertinoPageScaffold]())
		if dy := ts.TopLeft(scaffold1).Dy; dy != 0 {
			t.Fatalf("page 1 is at dy %v, want 0", dy)
		}
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		// The previous page is still visible behind the new sheet.
		if !ts.Exists(gluttertest.ByText("Page 1")) {
			t.Fatal("page 1 should still be shown")
		}
		one := ts.TopLeft(scaffold1)
		if one.Dy <= 0 || one.Dx <= 0 {
			t.Errorf("page 1 is at %v, want it to be down and to the right", one)
		}
		two := ts.TopLeft(gluttertest.Ancestor(gluttertest.ByText("Page 2"), gluttertest.ByType[*cupertino.CupertinoPageScaffold]())).Dy
		if two <= one.Dy {
			t.Errorf("page 2 (%v) is not below page 1 (%v)", two, one.Dy)
		}
	})

	t.Run("a sheet that covers another sheet moves it slightly upwards", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := widgets.NewGlobalKey[widgets.State]()
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(
			widgets.NewColumn([]widgets.Widget{widgets.NewText("Page 1"), button("Push Page 2", func() {
				widgets.NavigatorPush(key.CurrentContext(), cupertino.NewCupertinoSheetRoute(cupertino.CupertinoSheetRouteOpts{Builder: func(widgets.BuildContext) widgets.Widget {
					return cupertino.NewCupertinoPageScaffold(widgets.NewColumn([]widgets.Widget{widgets.NewText("Page 2"), button("Push Page 3", func() {
						widgets.NavigatorPush(key.CurrentContext(), cupertino.NewCupertinoSheetRoute(cupertino.CupertinoSheetRouteOpts{Builder: page("Page 3")}))
					})}))
				}}))
			})}), cupertino.CupertinoPageScaffoldOpts{Key: key})}))
		scaffold2 := gluttertest.Ancestor(gluttertest.ByText("Page 2"), gluttertest.ByType[*cupertino.CupertinoPageScaffold]())
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		before := ts.TopLeft(scaffold2).Dy
		ts.Tap(gluttertest.ByText("Push Page 3"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Page 3")) {
			t.Fatal("page 3 should be shown")
		}
		if after := ts.TopLeft(scaffold2).Dy; before <= 0 || before <= after {
			t.Errorf("page 2 was at %v and is at %v: it should have moved up", before, after)
		}
	})

	t.Run("the route has the settings it is given", func(t *testing.T) {
		ts := gluttertest.New(t)
		var route *cupertino.CupertinoSheetRoute
		pushRoute(ts, func() widgets.Route {
			route = cupertino.NewCupertinoSheetRoute(cupertino.CupertinoSheetRouteOpts{Builder: page("Page 2"),
				RouteSettings: widgets.NewRouteSettings(widgets.RouteSettingsOpts{Name: "sheet"})})
			return route
		})
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		if got := route.Settings().Name(); got != "sheet" {
			t.Errorf("name = %q, want sheet", got)
		}
	})

	t.Run("the sheet sets the style of the system bars", func(t *testing.T) {
		ts := gluttertest.New(t)
		var chrome *services.SystemChrome
		a := pushRoute(ts, func() widgets.Route {
			return cupertino.NewCupertinoSheetRoute(cupertino.CupertinoSheetRouteOpts{Builder: page("Page 2")})
		})
		chrome = widgets.WidgetsBindingOf(a.home.CurrentContext()).SystemChrome()
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		ts.FlushPosted()
		got, _ := chrome.LatestStyle()
		if got.StatusBarBrightness != ui.BrightnessDark || got.StatusBarIconBrightness != ui.BrightnessLight {
			t.Errorf("latest style = %+v, want the one of a dark status bar with light icons", got)
		}
	})
}

// dragHandle finds the drag handle of a sheet: a DecoratedBox that has a ShapeDecoration of the tertiary label.
func dragHandle() gluttertest.Finder {
	return gluttertest.ByPredicate("the drag handle of a sheet", func(w widgets.Widget) bool {
		box, ok := w.(*widgets.DecoratedBox)
		if !ok {
			return false
		}
		d, ok := box.Decoration().(*painting.ShapeDecoration)
		return ok && d.Color == foundation.Some(cupertino.CupertinoColors.TertiaryLabel.EffectiveColor())
	})
}

func TestCupertinoSheetDragDismiss(t *testing.T) {
	setup := func(t *testing.T, opts cupertino.ShowCupertinoSheetOpts) (*gluttertest.Tester, *sheetApp) {
		ts := gluttertest.New(t)
		var a *sheetApp
		a = newSheetApp(t, ts, nil, func(bc widgets.BuildContext) {
			opts.Builder = func(widgets.BuildContext) widgets.Widget {
				return a.pageScaffold(widgets.NewCenter(widgets.NewText("Page 2")))
			}
			cupertino.ShowCupertinoSheet(bc, opts)
		})
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Page 2")) {
			t.Fatal("page 2 should be shown")
		}
		return ts, a
	}

	t.Run("a partial drag and drop does not pop the sheet", func(t *testing.T) {
		ts, a := setup(t, cupertino.ShowCupertinoSheetOpts{})
		initial := a.sheetY(ts)
		gesture := ts.StartGesture(ui.Offset{Dx: 100, Dy: 200})
		gesture.MoveBy(ui.Offset{Dy: 200})
		ts.Pump()
		middle := a.sheetY(ts)
		if middle <= initial {
			t.Fatalf("the sheet did not move down: %v then %v", initial, middle)
		}
		gesture.Up()
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Page 2")) {
			t.Fatal("the sheet was popped")
		}
		if final := a.sheetY(ts); final >= middle || final != initial {
			t.Errorf("final position %v, want %v (it was %v mid-drag)", final, initial, middle)
		}
	})

	t.Run("dropping the drag further down the page pops the sheet", func(t *testing.T) {
		ts, _ := setup(t, cupertino.ShowCupertinoSheetOpts{})
		gesture := ts.StartGesture(ui.Offset{Dx: 100, Dy: 200})
		gesture.MoveBy(ui.Offset{Dy: 350})
		ts.Pump()
		gesture.Up()
		ts.PumpAndSettle()
		if ts.Exists(gluttertest.ByText("Page 2")) {
			t.Error("the sheet should be popped")
		}
	})

	t.Run("dragging does not move the sheet when drag is not enabled", func(t *testing.T) {
		ts, a := setup(t, cupertino.ShowCupertinoSheetOpts{NotEnableDrag: true})
		initial := a.sheetY(ts)
		gesture := ts.StartGesture(ui.Offset{Dx: 100, Dy: 200})
		gesture.MoveBy(ui.Offset{Dy: 200})
		ts.Pump()
		if middle := a.sheetY(ts); middle != initial {
			t.Errorf("the sheet moved from %v to %v", initial, middle)
		}
		gesture.Up()
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Page 2")) || a.sheetY(ts) != initial {
			t.Error("the sheet should not have moved")
		}
	})

	t.Run("a partial upward drag stretches and returns without popping", func(t *testing.T) {
		ts, a := setup(t, cupertino.ShowCupertinoSheetOpts{})
		initial := a.sheetY(ts)
		gesture := ts.StartGesture(ui.Offset{Dx: 100, Dy: 400})
		gesture.MoveBy(ui.Offset{Dy: -100})
		ts.Pump()
		if stretched := a.sheetY(ts); stretched >= initial {
			t.Errorf("the sheet did not stretch: %v then %v", initial, stretched)
		}
		gesture.Up()
		ts.PumpAndSettle()
		if final := a.sheetY(ts); final != initial {
			t.Errorf("final position %v, want %v", final, initial)
		}
	})

	t.Run("a sheet does not block a scrollable inside", func(t *testing.T) {
		ts := gluttertest.New(t)
		newSheetApp(t, ts, nil, func(bc widgets.BuildContext) {
			cupertino.ShowCupertinoSheet(bc, cupertino.ShowCupertinoSheetOpts{Builder: func(widgets.BuildContext) widgets.Widget {
				return cupertino.NewCupertinoPageScaffold(widgets.NewListView([]widgets.Widget{
					widgets.NewText("Top of Scroll"), widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(100.0)}),
					// The test platform is not a mobile one, so the list is not primary and is not always scrollable:
					// the content is taller than the sheet.
					widgets.NewText("Middle of Scroll"), widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(900.0)}),
				}))
			}})
		})
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		start := ts.TopLeft(gluttertest.ByText("Middle of Scroll")).Dy
		ts.Drag(gluttertest.ByText("Top of Scroll"), ui.Offset{Dy: -50})
		ts.PumpAndSettle()
		if end := ts.TopLeft(gluttertest.ByText("Middle of Scroll")).Dy; end >= start {
			t.Errorf("the list did not scroll: %v then %v", start, end)
		}
	})
}

func TestCupertinoSheetDraggableScrollable(t *testing.T) {
	setup := func(t *testing.T) (*gluttertest.Tester, *sheetApp) {
		ts := gluttertest.New(t)
		var a *sheetApp
		a = newSheetApp(t, ts, func(bc widgets.BuildContext, controller widgets.ScrollController) widgets.Widget {
			return a.pageScaffold(widgets.NewCustomScrollView([]widgets.Widget{
				widgets.NewSliverListBuilder(func(bc widgets.BuildContext, index int) widgets.Widget {
					return widgets.NewContainer(widgets.NewText("Scroll Item "+itoa(index)), widgets.ContainerOpts{Alignment: painting.AlignmentCenter, Height: foundation.Some(100.0)})
				}, widgets.SliverBuilderOpts{ItemCount: foundation.Some(20)}),
			}, widgets.CustomScrollViewOpts{Controller: controller, Primary: foundation.Some(false)}))
		}, nil)
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		return ts, a
	}
	itemY := func(ts *gluttertest.Tester) float64 { return ts.TopLeft(gluttertest.ByText("Scroll Item 3")).Dy }

	t.Run("it can be scrolled when at full height", func(t *testing.T) {
		ts, a := setup(t)
		sheet, item := a.sheetY(ts), itemY(ts)
		gesture := ts.StartGesture(ui.Offset{Dx: 100, Dy: 300})
		gesture.MoveBy(ui.Offset{Dy: -30}) // A small drag first to win the gesture arena.
		gesture.MoveBy(ui.Offset{Dy: -100})
		ts.Pump()
		if got := a.sheetY(ts); got != sheet {
			t.Errorf("the sheet moved from %v to %v", sheet, got)
		}
		if got := itemY(ts); got >= item {
			t.Errorf("the list did not scroll: %v then %v", item, got)
		}
		gesture.Up()
		ts.PumpAndSettle()
	})

	t.Run("the sheet slides down on a downwards drag when the content is at the top", func(t *testing.T) {
		ts, a := setup(t)
		sheet, item := a.sheetY(ts), itemY(ts)
		gesture := ts.StartGesture(ui.Offset{Dx: 100, Dy: 300})
		gesture.MoveBy(ui.Offset{Dy: 30})
		gesture.MoveBy(ui.Offset{Dy: 100})
		ts.Pump()
		finalSheet, finalItem := a.sheetY(ts), itemY(ts)
		if finalSheet <= sheet || finalItem <= item {
			t.Fatalf("the sheet went from %v to %v and the item from %v to %v", sheet, finalSheet, item, finalItem)
		}
		// The scrollable content moves down with the sheet.
		requireNear(t, "the item moved with the sheet", finalItem-item, finalSheet-sheet, 0.0005)
		gesture.Up()
		ts.PumpAndSettle()
	})

	t.Run("while the drag continues, the sheet switches between scrolling and the dismiss animation", func(t *testing.T) {
		ts, a := setup(t)
		sheet, item := a.sheetY(ts), itemY(ts)
		gesture := ts.StartGesture(ui.Offset{Dx: 100, Dy: 300})
		gesture.MoveBy(ui.Offset{Dy: -30})
		gesture.MoveBy(ui.Offset{Dy: -100})
		ts.Pump()
		if a.sheetY(ts) != sheet || itemY(ts) >= item {
			t.Fatalf("the sheet is at %v (want %v) and the item at %v (want less than %v)", a.sheetY(ts), sheet, itemY(ts), item)
		}
		// Drag back down the same amount: the sheet has not moved, and the scroll is back.
		gesture.MoveBy(ui.Offset{Dy: 100})
		ts.Pump()
		if a.sheetY(ts) != sheet || itemY(ts) != item {
			t.Fatalf("the sheet is at %v (want %v) and the item at %v (want %v)", a.sheetY(ts), sheet, itemY(ts), item)
		}
		// Drag down further: the whole sheet moves, and the content with it.
		gesture.MoveBy(ui.Offset{Dy: 100})
		ts.Pump()
		finalSheet, finalItem := a.sheetY(ts), itemY(ts)
		if finalSheet <= sheet || finalItem <= item {
			t.Fatalf("the sheet went from %v to %v and the item from %v to %v", sheet, finalSheet, item, finalItem)
		}
		if finalItem-item != finalSheet-sheet {
			t.Errorf("the item moved %v and the sheet %v", finalItem-item, finalSheet-sheet)
		}
		gesture.Up()
		ts.PumpAndSettle()
	})

	t.Run("a fling while scrolled down does not pop the sheet", func(t *testing.T) {
		ts, _ := setup(t)
		gesture := ts.StartGesture(ui.Offset{Dx: 100, Dy: 300})
		gesture.MoveBy(ui.Offset{Dy: -30})
		gesture.MoveBy(ui.Offset{Dy: -400})
		ts.Pump()
		gesture.Up()
		ts.PumpAndSettle()
		ts.FlingFrom(ui.Offset{Dx: 100, Dy: 400}, ui.Offset{Dy: 300}, 500)
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Scroll Item 3")) {
			t.Error("the sheet should still be open")
		}
	})

	t.Run("a fling while scrolled to the top pops the sheet", func(t *testing.T) {
		ts, _ := setup(t)
		ts.FlingFrom(ui.Offset{Dx: 100, Dy: 400}, ui.Offset{Dy: 300}, 500)
		ts.PumpAndSettle()
		if ts.Exists(gluttertest.ByText("Scroll Item 3")) {
			t.Error("the sheet should be popped")
		}
	})
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := ""
	for ; i > 0; i /= 10 {
		digits = string(rune('0'+i%10)) + digits
	}
	return digits
}

func TestCupertinoSheetTopGap(t *testing.T) {
	t.Run("a custom top gap is where the sheet is", func(t *testing.T) {
		ts := gluttertest.New(t)
		var a *sheetApp
		a = newSheetApp(t, ts, nil, func(bc widgets.BuildContext) {
			cupertino.ShowCupertinoSheet(bc, cupertino.ShowCupertinoSheetOpts{TopGap: foundation.Some(0.2), Builder: func(widgets.BuildContext) widgets.Widget {
				return a.pageScaffold(widgets.NewText("Page 2"))
			}})
		})
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		requireNear(t, "the sheet", a.sheetY(ts), 600*0.2, 0.01)
		// A sheet with a gap of its own does not move the route below it.
		if dy := ts.TopLeft(gluttertest.Ancestor(gluttertest.ByText("Page 1"), gluttertest.ByType[*cupertino.CupertinoPageScaffold]())).Dy; dy != 0 {
			t.Errorf("page 1 is at dy %v, want 0", dy)
		}
	})

	t.Run("the default top gap moves the route below it", func(t *testing.T) {
		ts := gluttertest.New(t)
		var a *sheetApp
		a = newSheetApp(t, ts, nil, func(bc widgets.BuildContext) {
			cupertino.ShowCupertinoSheet(bc, cupertino.ShowCupertinoSheetOpts{Builder: func(widgets.BuildContext) widgets.Widget {
				return a.pageScaffold(widgets.NewText("Page 2"))
			}})
		})
		ts.Tap(gluttertest.ByText("Push Page 2"))
		ts.PumpAndSettle()
		requireNear(t, "the sheet", a.sheetY(ts), 600*sheetTopGapRatio, 0.01)
		if dy := ts.TopLeft(gluttertest.Ancestor(gluttertest.ByText("Page 1"), gluttertest.ByType[*cupertino.CupertinoPageScaffold]())).Dy; dy <= 0 {
			t.Errorf("page 1 is at dy %v, want more than 0", dy)
		}
	})

	t.Run("a top gap out of range panics", func(t *testing.T) {
		skipUnlessDebug(t)
		for _, gap := range []float64{-0.1, 0.95} {
			err := recoverError(func() {
				cupertino.NewCupertinoSheetRoute(cupertino.CupertinoSheetRouteOpts{Builder: func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) }, TopGap: foundation.Some(gap)})
			})
			checkErrorTag(t, err, "NewCupertinoSheetRoute.CupertinoSheetTopGapOutOfRange")
		}
	})

	t.Run("a sheet with no builder panics", func(t *testing.T) {
		skipUnlessDebug(t)
		err := recoverError(func() { cupertino.NewCupertinoSheetRoute() })
		checkErrorTag(t, err, "NewCupertinoSheetRoute.CupertinoSheetBuilderMissing")
	})
}

func TestShowCupertinoSheetErrors(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	var bc widgets.BuildContext
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(c widgets.BuildContext) widgets.Widget { bc = c; return widgets.NewSizedBox(nil) })}))
	err := recoverError(func() {
		cupertino.ShowCupertinoSheet(bc, cupertino.ShowCupertinoSheetOpts{TopGap: foundation.Some(1.0), Builder: func(widgets.BuildContext) widgets.Widget { return nil }})
	})
	checkErrorTag(t, err, "ShowCupertinoSheet.CupertinoSheetTopGapOutOfRange")
	err = recoverError(func() { cupertino.ShowCupertinoSheet(bc, cupertino.ShowCupertinoSheetOpts{}) })
	checkErrorTag(t, err, "ShowCupertinoSheet.CupertinoSheetBuilderMissing")
}

func TestCupertinoSheetNestedNavigation(t *testing.T) {
	ts := gluttertest.New(t)
	var sheetContext widgets.BuildContext
	newSheetApp(t, ts, nil, func(bc widgets.BuildContext) {
		cupertino.ShowCupertinoSheet(bc, cupertino.ShowCupertinoSheetOpts{UseNestedNavigation: true, Builder: func(c widgets.BuildContext) widgets.Widget {
			sheetContext = c
			return cupertino.NewCupertinoPageScaffold(widgets.NewColumn([]widgets.Widget{widgets.NewText("Page 2"), button("Push Page 3", func() {
				widgets.NavigatorPush(sheetContext, cupertino.NewCupertinoPageRoute(page3))
			})}))
		}})
	})
	ts.Tap(gluttertest.ByText("Push Page 2"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Page 2")) || ts.Count(gluttertest.ByType[*widgets.Navigator]()) != 2 {
		t.Fatalf("the sheet should have a navigator of its own: %d navigators", ts.Count(gluttertest.ByType[*widgets.Navigator]()))
	}
	ts.Tap(gluttertest.ByText("Push Page 3"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Page 3")) {
		t.Fatal("page 3 should be pushed on the navigator of the sheet")
	}
	// A pop pops the nested navigator first, then the sheet.
	widgets.NavigatorOf(sheetContext).Pop()
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Page 2")) || ts.Exists(gluttertest.ByText("Page 3")) {
		t.Fatal("page 3 should be popped alone")
	}
	widgets.NavigatorOf(sheetContext).MaybePop()
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("Page 2")) {
		t.Error("the sheet should be popped")
	}
}

func page3(widgets.BuildContext) widgets.Widget {
	return cupertino.NewCupertinoPageScaffold(widgets.NewText("Page 3"))
}

func TestCupertinoSheetRoutePopSheet(t *testing.T) {
	ts := gluttertest.New(t)
	var sheetContext widgets.BuildContext
	newSheetApp(t, ts, nil, func(bc widgets.BuildContext) {
		cupertino.ShowCupertinoSheet(bc, cupertino.ShowCupertinoSheetOpts{Builder: func(c widgets.BuildContext) widgets.Widget {
			sheetContext = c
			return cupertino.NewCupertinoPageScaffold(widgets.NewText("Page 2"))
		}})
	})
	var homeContext widgets.BuildContext
	ts.Tap(gluttertest.ByText("Push Page 2"))
	ts.PumpAndSettle()
	if !cupertino.CupertinoSheetRouteHasParentSheet(sheetContext) {
		t.Error("the content of the sheet should know that it is in a sheet")
	}
	_ = homeContext
	cupertino.CupertinoSheetRoutePopSheet(sheetContext)
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("Page 2")) {
		t.Error("the sheet should be popped")
	}
}
