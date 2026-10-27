package material_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/leaktest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/svg"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The memory tests of the app level (step 29.21, SPEC §15 "Memory over time"): each scenario
// runs a step N times and 4N times and compares the live heap and the goroutines
// (internal/leaktest). The step leaves the app as it found it.

// leakApp pumps a Material app whose body is the widget, in a view of 800 x 600.
func leakApp(tb testing.TB, body widgets.Widget) *gluttertest.Tester {
	tb.Helper()
	ts := gluttertest.New(tb, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}, DiscardScenes: true})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: body})}))
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
	tests := []struct {
		name string
		step func(tb testing.TB) (*gluttertest.Tester, func())
	}{
		{"push and pop a page", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, func(widgets.BuildContext) widgets.Widget { return widgets.NewText("HOME") })
			return ts, func() {
				NavigatorPush(*bc, NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget {
					return NewScaffold(ScaffoldOpts{Body: widgets.NewText("PAGE")})
				}))
				ts.PumpAndSettle()
				NavigatorPop(*bc)
				ts.PumpAndSettle()
			}
		}},
		{"show and close a dialog", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, func(widgets.BuildContext) widgets.Widget { return widgets.NewText("HOME") })
			return ts, func() {
				ShowDialog[any](*bc, func(widgets.BuildContext) widgets.Widget {
					return NewAlertDialog(AlertDialogOpts{Title: widgets.NewText("T"), Content: widgets.NewText("C"),
						Actions: []widgets.Widget{NewTextButton(func() {}, widgets.NewText("OK"))}})
				})
				ts.PumpAndSettle()
				NavigatorPop(*bc)
				ts.PumpAndSettle()
			}
		}},
		{"show and hide a tooltip", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts := leakApp(tb, widgets.NewCenter(NewTooltip(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(40.0), Height: foundation.Some(40.0)}),
				TooltipOpts{Message: "tip"})))
			center := ts.Center(gluttertest.ByType[*Tooltip]())
			return ts, func() {
				ts.Hover(center)
				ts.Pump(50 * time.Millisecond)
				ts.PumpAndSettle()
				ts.Hover(ui.Offset{Dx: 5, Dy: 5})
				ts.Pump(time.Second)
				ts.PumpAndSettle()
			}
		}},
		{"show and dismiss a snack bar", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, func(widgets.BuildContext) widgets.Widget { return widgets.NewText("HOME") })
			return ts, func() {
				c := ScaffoldMessengerOf(*bc).ShowSnackBar(NewSnackBar(widgets.NewText("snack"), SnackBarOpts{Action: NewSnackBarAction("UNDO", func() {})}))
				ts.Pump(100 * time.Millisecond)
				ts.Pump(500 * time.Millisecond)
				c.Close()
				ts.Pump(100 * time.Millisecond)
				ts.Pump(500 * time.Millisecond)
			}
		}},
		{"open and close a bottom sheet", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, func(widgets.BuildContext) widgets.Widget { return widgets.NewText("HOME") })
			return ts, func() {
				ShowModalBottomSheet(*bc, func(widgets.BuildContext) widgets.Widget {
					return widgets.NewSizedBox(widgets.NewText("sheet"), widgets.SizedBoxOpts{Height: foundation.Some(200.0)})
				})
				ts.Pump(100 * time.Millisecond)
				ts.Pump(500 * time.Millisecond)
				NavigatorPop(*bc)
				ts.Pump(100 * time.Millisecond)
				ts.Pump(500 * time.Millisecond)
			}
		}},
		{"show and close a popup menu", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, func(widgets.BuildContext) widgets.Widget { return widgets.NewText("HOME") })
			return ts, func() {
				ShowMenu(*bc, []PopupMenuEntry{NewPopupMenuItem[int](widgets.NewText("one"), PopupMenuItemOpts[int]{Value: foundation.Some(1)})},
					ShowMenuOpts{Position: foundation.Some(rendering.NewRelativeRectFromLTRB(100, 100, 100, 100))})
				ts.Pump(100 * time.Millisecond)
				ts.Pump(500 * time.Millisecond)
				NavigatorPop(*bc)
				ts.Pump(100 * time.Millisecond)
				ts.Pump(500 * time.Millisecond)
			}
		}},
		{"push and pop a page with a hero", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts, bc := leakAppBC(tb, func(widgets.BuildContext) widgets.Widget {
				return widgets.NewHero("h", widgets.NewSizedBox(widgets.NewText("home hero"), widgets.SizedBoxOpts{Width: foundation.Some(50.0), Height: foundation.Some(50.0)}))
			})
			return ts, func() {
				NavigatorPush(*bc, NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget {
					return NewScaffold(ScaffoldOpts{Body: widgets.NewHero("h", widgets.NewSizedBox(widgets.NewText("page hero"), widgets.SizedBoxOpts{Width: foundation.Some(150.0), Height: foundation.Some(150.0)}))})
				}))
				ts.Pump(100 * time.Millisecond)
				ts.Pump(600 * time.Millisecond)
				NavigatorPop(*bc)
				ts.Pump(100 * time.Millisecond)
				ts.Pump(600 * time.Millisecond)
			}
		}},
		{"open and close a menu", func(tb testing.TB) (*gluttertest.Tester, func()) {
			ctl := widgets.NewMenuController()
			ts := leakApp(tb, widgets.NewCenter(NewMenuAnchor([]widgets.Widget{
				NewMenuItemButton(MenuItemButtonOpts{Child: widgets.NewText("one"), OnPressed: func() {}}),
				NewMenuItemButton(MenuItemButtonOpts{Child: widgets.NewText("two"), OnPressed: func() {}}),
			}, MenuAnchorOpts{Controller: ctl, Child: widgets.NewText("anchor")})))
			return ts, func() {
				ctl.Open()
				ts.PumpAndSettle()
				ctl.Close()
				ts.PumpAndSettle()
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "material/routes/"+tt.name, func() func() {
				ts, step := tt.step(t)
				return func() { step(); ts.ClearLogs() }
			}, leaktest.Opts{N: 25, SlackObjects: 150})
		})
	}
}

// mounted is a Material app with a body that the step shows and hides: every widget of the tree
// is created and disposed in each step.
func mountedApp(tb testing.TB, build func(bc widgets.BuildContext) widgets.Widget) func() {
	tb.Helper()
	show := false
	var set widgets.StateSetter
	ts := leakApp(tb, widgets.NewStatefulBuilder(func(bc widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
		set = setState
		if !show {
			return widgets.NewText("none")
		}
		return build(bc)
	}))
	return func() {
		set(func() { show = true })
		ts.Pump(100 * time.Millisecond)
		ts.Pump(400 * time.Millisecond)
		set(func() { show = false })
		ts.Pump(100 * time.Millisecond)
		ts.Pump(400 * time.Millisecond)
	}
}

func leakPNG(tb testing.TB) []byte {
	tb.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		tb.Fatal(err)
	}
	return b.Bytes()
}

func TestLeaksMountAndUnmount(t *testing.T) {
	const svgSource = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M12 2l3 7h7l-5.5 4.5L18 21l-6-4-6 4 1.5-7.5L2 9h7z" fill="#6750a4"/></svg>`
	key := widgets.NewGlobalKey[widgets.State]()
	tests := []struct {
		name  string
		build func(tb testing.TB) func(bc widgets.BuildContext) widgets.Widget
	}{
		{"progress indicator", func(testing.TB) func(widgets.BuildContext) widgets.Widget {
			return func(widgets.BuildContext) widgets.Widget { return NewCircularProgressIndicator() }
		}},
		{"switch", func(testing.TB) func(widgets.BuildContext) widgets.Widget {
			return func(widgets.BuildContext) widgets.Widget { return NewSwitch(true, func(bool) {}) }
		}},
		{"checkbox", func(testing.TB) func(widgets.BuildContext) widgets.Widget {
			return func(widgets.BuildContext) widgets.Widget {
				return NewCheckbox(foundation.Some(true), func(foundation.Opt[bool]) {})
			}
		}},
		{"slider", func(testing.TB) func(widgets.BuildContext) widgets.Widget {
			return func(widgets.BuildContext) widgets.Widget { return NewSlider(0.5, func(float64) {}) }
		}},
		{"text field", func(testing.TB) func(widgets.BuildContext) widgets.Widget {
			return func(widgets.BuildContext) widgets.Widget {
				return NewTextField(TextFieldOpts{Controller: widgets.NewTextEditingController("hello")})
			}
		}},
		{"list with scroll controller", func(testing.TB) func(widgets.BuildContext) widgets.Widget {
			return func(widgets.BuildContext) widgets.Widget {
				return widgets.NewListViewBuilder(func(_ widgets.BuildContext, i int) widgets.Widget {
					if i >= 100 {
						return nil
					}
					return widgets.NewSizedBox(widgets.NewText("row"), widgets.SizedBoxOpts{Height: foundation.Some(30.0)})
				}, widgets.ListViewOpts{Controller: widgets.NewScrollController()})
			}
		}},
		{"focus nodes", func(testing.TB) func(widgets.BuildContext) widgets.Widget {
			return func(widgets.BuildContext) widgets.Widget {
				return widgets.NewFocus(widgets.NewFocusScope(widgets.NewFocus(widgets.NewText("f"), widgets.FocusOpts{Autofocus: true})))
			}
		}},
		{"global key", func(testing.TB) func(widgets.BuildContext) widgets.Widget {
			return func(widgets.BuildContext) widgets.Widget {
				return widgets.NewKeyedSubtree(widgets.NewText("keyed"), widgets.WidgetOpts{Key: key})
			}
		}},
		{"memory image", func(tb testing.TB) func(widgets.BuildContext) widgets.Widget {
			data := leakPNG(tb)
			return func(widgets.BuildContext) widgets.Widget { return widgets.NewImageMemory(data) }
		}},
		{"svg picture", func(testing.TB) func(widgets.BuildContext) widgets.Widget {
			return func(widgets.BuildContext) widgets.Widget {
				return widgets.NewSizedBox(svg.NewSvgPictureString(svgSource), widgets.SizedBoxOpts{Width: foundation.Some(48.0), Height: foundation.Some(48.0)})
			}
		}},
		{"tabs", func(testing.TB) func(widgets.BuildContext) widgets.Widget {
			return func(widgets.BuildContext) widgets.Widget {
				return NewDefaultTabController(2, widgets.NewColumn([]widgets.Widget{
					NewTabBar([]widgets.Widget{NewTab(TabOpts{Text: "a"}), NewTab(TabOpts{Text: "b"})}),
					widgets.NewExpanded(NewTabBarView([]widgets.Widget{widgets.NewText("A"), widgets.NewText("B")})),
				}))
			}
		}},
		{"ink well and buttons", func(testing.TB) func(widgets.BuildContext) widgets.Widget {
			return func(widgets.BuildContext) widgets.Widget {
				return widgets.NewColumn([]widgets.Widget{NewElevatedButton(func() {}, widgets.NewText("e")),
					NewTextButton(func() {}, widgets.NewText("t")), NewInkWell(widgets.NewText("i"), InkWellOpts{OnTap: func() {}})})
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "material/mount/"+tt.name, func() func() { return mountedApp(t, tt.build(t)) }, leaktest.Opts{N: 25, SlackObjects: 150})
		})
	}
}

// TestScaffoldIsFreedAfterItsRouteIsPopped checks that nothing keeps a scaffold of a popped route
// alive. A scaffold registers with its ScaffoldMessenger each time its dependencies change, and
// is unregistered once, when it is disposed (step 29.21).
func TestScaffoldIsFreedAfterItsRouteIsPopped(t *testing.T) {
	ts, bc := leakAppBC(t, func(widgets.BuildContext) widgets.Widget { return widgets.NewText("HOME") })
	var pages []weak.Pointer[widgets.Text]
	for i := 0; i < 5; i++ {
		NavigatorPush(*bc, NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget {
			return NewScaffold(ScaffoldOpts{Body: widgets.NewText("PAGE")})
		}))
		ts.PumpAndSettle()
		pages = append(pages, weak.Make(ts.Widget(gluttertest.ByText("PAGE")).(*widgets.Text)))
		NavigatorPop(*bc)
		ts.PumpAndSettle()
	}
	runtime.GC()
	runtime.GC()
	for i, p := range pages {
		if p.Value() != nil {
			t.Errorf("the page of push %d is still alive after its route was popped", i)
		}
	}
}

func TestLeaksFocusAndGestures(t *testing.T) {
	tests := []struct {
		name string
		// n is N of the scenario: more for the cheap steps.
		n    int
		step func(tb testing.TB) (*gluttertest.Tester, func())
	}{
		{"focus and unfocus a text field", 150, func(tb testing.TB) (*gluttertest.Tester, func()) {
			fn := widgets.NewFocusNode()
			ts := leakApp(tb, widgets.NewColumn([]widgets.Widget{
				NewTextField(TextFieldOpts{FocusNode: fn}), widgets.NewText("outside"),
			}))
			field := ts.Center(gluttertest.ByType[*TextField]())
			outside := ts.Center(gluttertest.ByText("outside"))
			return ts, func() {
				ts.TapAt(field)
				ts.Pump(100 * time.Millisecond)
				ts.EnterText(gluttertest.ByType[*TextField](), "abc")
				ts.Pump(600 * time.Millisecond) // a caret blink
				fn.Unfocus()
				ts.Pump(100 * time.Millisecond)
				_ = outside
			}
		}},
		{"type in a field with the selection handles", 50, func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts := leakApp(tb, widgets.NewColumn([]widgets.Widget{NewTextField(TextFieldOpts{})}))
			field := ts.Center(gluttertest.ByType[*TextField]())
			return ts, func() {
				ts.TapAt(field)
				ts.Pump(100 * time.Millisecond)
				ts.EnterText(gluttertest.ByType[*TextField](), "hello world")
				ts.Pump(50 * time.Millisecond)
				ts.LongPressAt(field)
				ts.PumpAndSettle()
				ts.TapAt(ui.Offset{Dx: 700, Dy: 500})
				ts.PumpAndSettle()
			}
		}},
		{"taps with a new pointer each time", 300, func(tb testing.TB) (*gluttertest.Tester, func()) {
			taps := 0
			ts := leakApp(tb, widgets.NewCenter(widgets.NewGestureDetector(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
				widgets.GestureDetectorOpts{Behavior: rendering.HitTestBehaviorOpaque, OnTap: func() { taps++ }})))
			at := ts.Center(gluttertest.ByType[*widgets.GestureDetector]())
			return ts, func() { ts.TapAt(at); ts.Pump(16 * time.Millisecond) }
		}},
		{"double taps and long presses", 200, func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts := leakApp(tb, widgets.NewCenter(widgets.NewGestureDetector(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
				widgets.GestureDetectorOpts{Behavior: rendering.HitTestBehaviorOpaque, OnDoubleTap: func() {}, OnLongPress: func() {}})))
			at := ts.Center(gluttertest.ByType[*widgets.GestureDetector]())
			return ts, func() {
				ts.TapAt(at)
				ts.Pump(50 * time.Millisecond)
				ts.TapAt(at)
				ts.Pump(500 * time.Millisecond)
				ts.LongPressAt(at)
				ts.Pump(16 * time.Millisecond)
			}
		}},
		{"drags and flings", 40, func(tb testing.TB) (*gluttertest.Tester, func()) {
			ctl := widgets.NewScrollController()
			ts := leakApp(tb, widgets.NewListViewBuilder(func(_ widgets.BuildContext, i int) widgets.Widget {
				return widgets.NewSizedBox(widgets.NewText("row"), widgets.SizedBoxOpts{Height: foundation.Some(40.0)})
			}, widgets.ListViewOpts{Controller: ctl, ItemCount: foundation.Some(2000)}))
			at := ui.Offset{Dx: 400, Dy: 300}
			return ts, func() {
				ts.DragFrom(at, ui.Offset{Dx: 0, Dy: -200})
				ts.Pump(16 * time.Millisecond)
				ts.FlingFrom(at, ui.Offset{Dx: 0, Dy: 200}, 1000)
				ts.PumpAndSettle()
			}
		}},
		{"hover over many regions", 300, func(tb testing.TB) (*gluttertest.Tester, func()) {
			ts := leakApp(tb, widgets.NewListViewBuilder(func(_ widgets.BuildContext, i int) widgets.Widget {
				return NewListTile(ListTileOpts{Title: widgets.NewText("tile"), OnTap: func() {}})
			}, widgets.ListViewOpts{ItemCount: foundation.Some(30)}))
			i := 0
			return ts, func() {
				i++
				ts.Hover(ui.Offset{Dx: 100, Dy: float64(20 + (i%10)*50)})
				ts.Pump(16 * time.Millisecond)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "material/input/"+tt.name, func() func() {
				ts, step := tt.step(t)
				return func() { step(); ts.ClearLogs() }
			}, leaktest.Opts{N: tt.n, SlackObjects: 150})
		})
	}
}
