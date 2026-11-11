package cupertino_test

import (
	"runtime"
	"testing"
	"time"
	"weak"

	"golang.org/x/image/font/gofont/gomono"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/painting"
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
			if !foundation.DebugMode {
				tb.Skip("the platform override is a debug-only feature")
			}
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

// TestLeaksControls is the memory test of the controls (step 34.4): each scenario mounts the controls, uses them
// and takes them out of the tree, so a controller, a ticker, a focus node, a recognizer or a registration that a
// control does not release shows as growth.
func TestLeaksControls(t *testing.T) {
	type pick int
	tests := []struct {
		name string
		page func(value *int) widgets.Widget
		use  func(ts *gluttertest.Tester)
	}{
		{"checkbox, switch, radio and slider", func(value *int) widgets.Widget {
			return widgets.NewStatefulBuilder(func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
				return widgets.NewRadioGroup(func(v foundation.Opt[pick]) { setState(func() { *value = int(v.OrElse(0)) }) },
					widgets.NewColumn([]widgets.Widget{
						cupertino.NewCupertinoCheckbox(foundation.Some(*value%2 == 0), func(foundation.Opt[bool]) { setState(func() { *value++ }) }),
						cupertino.NewCupertinoSwitch(*value%2 == 0, func(bool) { setState(func() { *value++ }) }),
						cupertino.NewCupertinoRadio(pick(1)), cupertino.NewCupertinoRadio(pick(2)),
						cupertino.NewCupertinoSlider(0.5, func(float64) {}),
						cupertino.NewCupertinoActivityIndicator(),
					}, widgets.ColumnOpts{MainAxisSize: 1}), widgets.RadioGroupOpts[pick]{GroupValue: foundation.Some(pick(*value % 3))})
			})
		}, func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByType[*cupertino.CupertinoCheckbox]())
			ts.Tap(gluttertest.ByType[*cupertino.CupertinoSwitch]())
			ts.Pump(time.Second) // the activity indicator spins forever: nothing settles
			ts.Drag(gluttertest.ByType[*cupertino.CupertinoSlider](), ui.Offset{Dx: 30})
			ts.Pump(time.Second)
		}},
		{"segmented controls", func(value *int) widgets.Widget {
			return widgets.NewStatefulBuilder(func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
				labels := []string{"A", "B", "C"}
				return widgets.NewColumn([]widgets.Widget{
					cupertino.NewCupertinoSegmentedControl(func(string) { setState(func() { *value++ }) }, segments(labels...),
						cupertino.CupertinoSegmentedControlOpts[string]{GroupValue: foundation.Some(labels[*value%3])}),
					cupertino.NewCupertinoSlidingSegmentedControl(func(string) { setState(func() { *value++ }) }, segments(labels...),
						cupertino.CupertinoSlidingSegmentedControlOpts[string]{GroupValue: foundation.Some(labels[*value%3])}),
				}, widgets.ColumnOpts{MainAxisSize: 1})
			})
		}, func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByType[*cupertino.CupertinoSegmentedControl[string]]())
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByType[*cupertino.CupertinoSlidingSegmentedControl[string]]())
			ts.PumpAndSettle()
			ts.Drag(gluttertest.ByType[*cupertino.CupertinoSlidingSegmentedControl[string]](), ui.Offset{Dx: 60})
			ts.PumpAndSettle()
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "cupertino/controls/"+tt.name, func() func() {
				ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}, DiscardScenes: true})
				value := 0
				mount := func() {
					ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(tt.page(&value)))}))
					ts.Pump()
				}
				return func() {
					mount()
					tt.use(ts)
					ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewSizedBox(nil)}))
					ts.PumpAndSettle()
					ts.ClearLogs()
				}
			}, leaktest.Opts{N: 25, SlackObjects: 150})
		})
	}
}

// TestLeaksText mounts, uses and unmounts the text fields of step 34.5: a field with the clear button, a search
// field, and a field whose selection shows the handles and the toolbar, on iOS (the pager of the toolbar, the
// magnifier and the controllers of the animations are what could stay). The warm-up is long on purpose: the live
// objects of these scenarios dip by about 300 at the first measure and come back, as the longer runs show
// (LEAKTEST_SCALE=20 grows by 30 objects from 4000 to 16000 steps), so a short warm-up compares a dip with its
// recovery.
func TestLeaksText(t *testing.T) {
	skipUnlessDebug(t)
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformIOS))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
	tests := []struct {
		name string
		page func() widgets.Widget
		use  func(ts *gluttertest.Tester)
	}{
		{"text field and clear button", func() widgets.Widget {
			return cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Placeholder: foundation.Some("name"),
				ClearButtonMode: cupertino.OverlayVisibilityModeEditing, Prefix: widgets.NewText("P")})
		}, func(ts *gluttertest.Tester) {
			ts.EnterText(gluttertest.ByType[*cupertino.CupertinoTextField](), "text")
			ts.Pump()
			ts.Tap(gluttertest.ByIcon(cupertino.CupertinoIcons.ClearThickCircled()))
			ts.Pump(time.Second)
		}},
		{"search field", func() widgets.Widget {
			return cupertino.NewCupertinoSearchTextField()
		}, func(ts *gluttertest.Tester) {
			ts.EnterText(gluttertest.ByType[*cupertino.CupertinoSearchTextField](), "query")
			ts.Pump()
			ts.Tap(gluttertest.ByIcon(cupertino.CupertinoIcons.XmarkCircleFill()))
			ts.Pump(time.Second)
		}},
		{"selection with the handles and the toolbar", func() widgets.Widget {
			return widgets.NewPadding(painting.NewEdgeInsetsAll(20), cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{
				Controller: widgets.NewTextEditingController("one two three four five six seven eight nine ten")}))
		}, func(ts *gluttertest.Tester) {
			ts.LongPress(gluttertest.ByType[*cupertino.CupertinoTextField]())
			ts.PumpAndSettle()
			ts.TapAt(ui.Offset{Dx: 700, Dy: 500})
			ts.PumpAndSettle()
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "cupertino/text/"+tt.name, func() func() {
				ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}, DiscardScenes: true})
				return func() {
					ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(tt.page()))}))
					ts.Pump()
					tt.use(ts)
					ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewSizedBox(nil)}))
					ts.PumpAndSettle()
					ts.ClearLogs()
				}
			}, leaktest.Opts{N: 200, Warmup: 700})
		})
	}
}

// TestLeaksPopups opens and closes the alert dialog, the action sheet, the context menu and the menu of an anchor,
// and a swipe over the menu, and leaves the app: the objects must be as many at the end as at the start (step 34.6).
func TestLeaksPopups(t *testing.T) {
	skipUnlessDebug(t)
	if runtime.GOOS == "js" {
		t.Skip("500 runs of four popups take longer than the test timeout under Node; the host runs them")
	}
	tests := []struct {
		name string
		page func(controller *widgets.MenuController) widgets.Widget
		use  func(ts *gluttertest.Tester, controller *widgets.MenuController)
	}{
		{"alert dialog", func(*widgets.MenuController) widgets.Widget {
			return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				return cupertino.NewCupertinoButton(func() {
					cupertino.ShowCupertinoDialog(bc, func(bc widgets.BuildContext) widgets.Widget {
						return cupertino.NewCupertinoAlertDialog(cupertino.CupertinoAlertDialogOpts{Title: widgets.NewText("Title"), Content: widgets.NewText("Content"),
							Actions: []widgets.Widget{cupertino.NewCupertinoDialogAction(widgets.NewText("OK"), cupertino.CupertinoDialogActionOpts{OnPressed: func() { widgets.NavigatorOf(bc).Pop() }})}})
					})
				}, widgets.NewText("Go"))
			})
		}, func(ts *gluttertest.Tester, _ *widgets.MenuController) {
			ts.Tap(gluttertest.ByText("Go"))
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByText("OK"))
			ts.PumpAndSettle()
		}},
		{"action sheet", func(*widgets.MenuController) widgets.Widget {
			return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				return cupertino.NewCupertinoButton(func() {
					cupertino.ShowCupertinoModalPopup(bc, func(bc widgets.BuildContext) widgets.Widget {
						return cupertino.NewCupertinoActionSheet(cupertino.CupertinoActionSheetOpts{Title: widgets.NewText("Title"),
							Actions:      []widgets.Widget{cupertino.NewCupertinoActionSheetAction(func() { widgets.NavigatorOf(bc).Pop() }, widgets.NewText("One"))},
							CancelButton: cupertino.NewCupertinoActionSheetAction(func() { widgets.NavigatorOf(bc).Pop() }, widgets.NewText("Cancel"))})
					})
				}, widgets.NewText("Go"))
			})
		}, func(ts *gluttertest.Tester, _ *widgets.MenuController) {
			ts.Tap(gluttertest.ByText("Go"))
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByText("One"))
			ts.PumpAndSettle()
		}},
		{"context menu", func(*widgets.MenuController) widgets.Widget {
			return cupertino.NewCupertinoContextMenu([]widgets.Widget{cupertino.NewCupertinoContextMenuAction(widgets.NewText("Copy"))},
				widgets.NewContainer(widgets.NewText("Child"), widgets.ContainerOpts{Width: foundation.Some(200.0), Height: foundation.Some(80.0)}))
		}, func(ts *gluttertest.Tester, _ *widgets.MenuController) {
			g := ts.StartGesture(ts.Center(gluttertest.ByText("Child")))
			ts.PumpAndSettle()
			g.Up()
			ts.PumpAndSettle()
			ts.TapAt(ui.Offset{Dx: 1, Dy: 1})
			ts.PumpAndSettle()
		}},
		{"menu anchor with a swipe", func(controller *widgets.MenuController) widgets.Widget {
			return cupertino.NewCupertinoMenuAnchor([]widgets.Widget{
				cupertino.NewCupertinoMenuItem(widgets.NewText("One"), cupertino.CupertinoMenuItemOpts{OnPressed: func() {}}),
				cupertino.NewCupertinoMenuItem(widgets.NewText("Two"), cupertino.CupertinoMenuItemOpts{OnPressed: func() {}})},
				cupertino.CupertinoMenuAnchorOpts{Controller: controller, Child: widgets.NewText("Anchor")})
		}, func(ts *gluttertest.Tester, controller *widgets.MenuController) {
			controller.Open()
			ts.PumpAndSettle()
			g := ts.StartGesture(ts.Center(gluttertest.ByText("One")))
			ts.Pump()
			g.MoveTo(ts.Center(gluttertest.ByText("Two")))
			ts.Pump(100 * time.Millisecond)
			g.MoveTo(ui.Offset{Dx: 5, Dy: 500})
			ts.Pump(300 * time.Millisecond)
			g.Cancel()
			controller.Close()
			ts.PumpAndSettle()
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "cupertino/popups/"+tt.name, func() func() {
				ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}, DiscardScenes: true, Fonts: []gluttertest.Font{
					{Family: cupertino.CupertinoSystemTextFamily, Data: gomono.TTF}, {Family: cupertino.CupertinoSystemDisplayFamily, Data: gomono.TTF}}})
				return func() {
					controller := widgets.NewMenuController()
					ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(tt.page(controller)))}))
					ts.Pump()
					tt.use(ts, controller)
					ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewSizedBox(nil)}))
					ts.PumpAndSettle()
					ts.ClearLogs()
				}
			}, leaktest.Opts{N: 100, Warmup: 400})
		})
	}
}

// TestLeaksPickers drags a wheel, the date picker in two modes and the timer picker, and builds the tree again (N and 4N).
func TestLeaksPickers(t *testing.T) {
	skipUnlessDebug(t)
	if runtime.GOOS == "js" {
		t.Skip("the runs of the pickers take longer than the test timeout under Node; the host runs them")
	}
	today := time.Date(2019, 1, 1, 4, 12, 0, 0, time.Local)
	tests := []struct {
		name   string
		picker func() widgets.Widget
		drag   string
	}{
		{"wheel", func() widgets.Widget {
			return widgets.NewSizedBox(cupertino.NewCupertinoPicker(32, func(int) {}, []widgets.Widget{
				widgets.NewText("A"), widgets.NewText("B"), widgets.NewText("C"), widgets.NewText("D")}), widgets.SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(216.0)})
		}, "B"},
		{"date and time", func() widgets.Widget {
			return widgets.NewSizedBox(cupertino.NewCupertinoDatePicker(func(time.Time) {}, cupertino.CupertinoDatePickerOpts{InitialDateTime: foundation.Some(today)}),
				widgets.SizedBoxOpts{Width: foundation.Some(500.0), Height: foundation.Some(216.0)})
		}, "12"},
		{"date", func() widgets.Widget {
			return widgets.NewSizedBox(cupertino.NewCupertinoDatePicker(func(time.Time) {}, cupertino.CupertinoDatePickerOpts{
				Mode: cupertino.CupertinoDatePickerModeDate, InitialDateTime: foundation.Some(today)}), widgets.SizedBoxOpts{Width: foundation.Some(500.0), Height: foundation.Some(216.0)})
		}, "2019"},
		{"timer", func() widgets.Widget {
			return widgets.NewSizedBox(cupertino.NewCupertinoTimerPicker(func(time.Duration) {}, cupertino.CupertinoTimerPickerOpts{
				InitialTimerDuration: 5*time.Hour + 17*time.Minute + 19*time.Second}), widgets.SizedBoxOpts{Width: foundation.Some(400.0), Height: foundation.Some(216.0)})
		}, "17"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "cupertino/pickers/"+tt.name, func() func() {
				ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}, DiscardScenes: true, Fonts: []gluttertest.Font{
					{Family: cupertino.CupertinoSystemTextFamily, Data: gomono.TTF}, {Family: cupertino.CupertinoSystemDisplayFamily, Data: gomono.TTF}}})
				return func() {
					ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(tt.picker()))}))
					ts.Pump()
					ts.Drag(gluttertest.First(gluttertest.ByText(tt.drag)), ui.Offset{Dy: -40})
					ts.PumpAndSettle()
					ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewSizedBox(nil)}))
					ts.PumpAndSettle()
					ts.ClearLogs()
				}
			}, leaktest.Opts{N: 50, Warmup: 100})
		})
	}
}

// TestLeaksLists mounts and unmounts the lists of step 34.8: the page of sections, the expansion tile (expanded and
// collapsed again, with the overlay of the fade), the form rows with a text field, and the refresh control (pulled, refreshing and
// done).
func TestLeaksLists(t *testing.T) {
	skipUnlessDebug(t)
	if runtime.GOOS == "js" {
		t.Skip("the runs take longer than the test timeout under Node; the host runs them")
	}
	tests := []struct {
		name string
		page func() widgets.Widget
		use  func(ts *gluttertest.Tester)
	}{
		{"sections", func() widgets.Widget { return widgets.NewSingleChildScrollView(listsPage()) }, func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("Wi-Fi"))
			ts.Pump()
		}},
		{"expansion tile", func() widgets.Widget {
			return expansionTile("Title", "Content")
		}, func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("Title"))
			ts.Pump(300 * time.Millisecond)
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByText("Title"))
			ts.PumpAndSettle()
		}},
		{"text form row", func() widgets.Widget {
			return widgets.NewForm(cupertino.NewCupertinoFormSection([]widgets.Widget{
				cupertino.NewCupertinoTextFormFieldRow(cupertino.CupertinoTextFormFieldRowOpts{Prefix: widgets.NewText("Name"),
					AutovalidateMode: widgets.AutovalidateModeAlways,
					Validator:        func(v foundation.Opt[string]) foundation.Opt[string] { return str("Error") }})}))
		}, func(ts *gluttertest.Tester) {
			ts.EnterText(gluttertest.ByType[*cupertino.CupertinoTextField](), "Soup")
			ts.Pump()
		}},
		{"refresh control", func() widgets.Widget {
			return widgets.NewCustomScrollView([]widgets.Widget{
				cupertino.NewCupertinoSliverRefreshControl(cupertino.CupertinoSliverRefreshControlOpts{OnRefresh: func() <-chan struct{} {
					done := make(chan struct{})
					close(done)
					return done
				}}), refreshList(10)})
		}, func(ts *gluttertest.Tester) {
			ts.Drag(gluttertest.ByText("0"), ui.Offset{Dy: 150})
			ts.PumpAndSettle()
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "cupertino/lists/"+tt.name, func() func() {
				ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 700}, DiscardScenes: true, Fonts: []gluttertest.Font{
					{Family: cupertino.CupertinoSystemTextFamily, Data: gomono.TTF}, {Family: cupertino.CupertinoSystemDisplayFamily, Data: gomono.TTF}}})
				return func() {
					ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(tt.page())}))
					ts.Pump()
					tt.use(ts)
					ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewSizedBox(nil)}))
					ts.PumpAndSettle()
					ts.ClearLogs()
				}
			}, leaktest.Opts{N: 50, Warmup: 100})
		})
	}
}
