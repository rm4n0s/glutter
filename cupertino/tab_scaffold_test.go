package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/tab_scaffold_test.dart. Not ported: the
// restoration tests (SPEC §2), the semantics cases and the cases of the CupertinoTabView with the CupertinoApp
// route observers (those are in tab_test.go).

// tabContent is the content of a tab: a text with the label that fills its space.
func tabContent(label string) widgets.Widget {
	return widgets.NewSizedBoxExpand(widgets.NewText(label))
}

func TestCupertinoTabScaffold(t *testing.T) {
	t.Run("tabs are built lazily and kept", func(t *testing.T) {
		ts := gluttertest.New(t)
		built := map[int]int{}
		scaffold := func() widgets.Widget {
			return cupertino.NewCupertinoTabScaffold(cupertino.NewCupertinoTabBar(tabItems("Tab 1", "Tab 2", "Tab 3")),
				func(_ widgets.BuildContext, i int) widgets.Widget {
					built[i]++
					return tabContent("Content " + string(rune('1'+i)))
				})
		}
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: scaffold()}))
		if len(built) != 1 || built[0] == 0 {
			t.Fatalf("built tabs = %v, want only the first", built)
		}
		ts.Tap(gluttertest.ByText("Tab 2"))
		ts.Pump()
		if built[1] == 0 || built[2] != 0 {
			t.Errorf("built tabs = %v, want the first and the second", built)
		}
		// The first tab is still in the tree, offstage.
		if !ts.Exists(gluttertest.ByText("Content 1", gluttertest.FindOpts{IncludeOffstage: true})) {
			t.Error("the first tab was removed from the tree")
		}
		if ts.Exists(gluttertest.ByText("Content 1")) {
			t.Error("the first tab is visible")
		}
		if !ts.Exists(gluttertest.ByText("Content 2")) {
			t.Error("the second tab is not visible")
		}
	})

	t.Run("an opaque tab bar shortens the content and a translucent one does not", func(t *testing.T) {
		for _, tt := range []struct {
			background ui.Color
			height     float64
			padding    float64
		}{{ui.NewColor(0xFFFFFFFF), 550, 0}, {ui.NewColor(0x80FFFFFF), 600, 50}} {
			ts := gluttertest.New(t)
			var padding float64
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabScaffold(
				cupertino.NewCupertinoTabBar(tabItems("Tab 1", "Tab 2"), cupertino.CupertinoTabBarOpts{BackgroundColor: tt.background}),
				func(bc widgets.BuildContext, i int) widgets.Widget {
					padding = widgets.MediaQueryOf(bc).Padding.Bottom
					return widgets.NewSizedBoxExpand(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey("content")}))
				})}))
			if got := ts.Size(gluttertest.ByKey(foundation.NewValueKey("content"))).Height; got != tt.height {
				t.Errorf("background %v: content height = %v, want %v", tt.background, got, tt.height)
			}
			if padding != tt.padding {
				t.Errorf("background %v: bottom padding for the content = %v, want %v", tt.background, padding, tt.padding)
			}
		}
	})

	t.Run("the keyboard resizes the content only when asked", func(t *testing.T) {
		for _, tt := range []struct {
			notResize bool
			height    float64
		}{{false, 300}, {true, 550}} {
			ts := gluttertest.New(t)
			home := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				data := widgets.MediaQueryOf(bc)
				return widgets.NewMediaQuery(data.CopyWith(widgets.MediaQueryDataCopyOpts{ViewInsets: foundation.Some(
					painting.EdgeInsets{Bottom: 300})}),
					cupertino.NewCupertinoTabScaffold(cupertino.NewCupertinoTabBar(tabItems("Tab 1", "Tab 2"),
						cupertino.CupertinoTabBarOpts{BackgroundColor: ui.NewColor(0xFFFFFFFF)}),
						func(widgets.BuildContext, int) widgets.Widget {
							return widgets.NewSizedBoxExpand(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey("content")}))
						}, cupertino.CupertinoTabScaffoldOpts{NotResizeToAvoidBottomInset: tt.notResize}))
			})
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: home}))
			if got := ts.Size(gluttertest.ByKey(foundation.NewValueKey("content"))).Height; got != tt.height {
				t.Errorf("notResize %v: content height = %v, want %v", tt.notResize, got, tt.height)
			}
		}
	})

	t.Run("a controller switches the tab and tells the bar", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := cupertino.NewCupertinoTabController(cupertino.CupertinoTabControllerOpts{InitialIndex: 1})
		t.Cleanup(controller.Dispose)
		var taps []int
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabScaffold(
			cupertino.NewCupertinoTabBar(tabItems("Tab 1", "Tab 2", "Tab 3"), cupertino.CupertinoTabBarOpts{OnTap: func(i int) { taps = append(taps, i) }}),
			func(_ widgets.BuildContext, i int) widgets.Widget {
				return tabContent("Content " + string(rune('1'+i)))
			},
			cupertino.CupertinoTabScaffoldOpts{Controller: controller})}))
		if !ts.Exists(gluttertest.ByText("Content 2")) {
			t.Fatal("the controller's initial tab is not the one that shows")
		}
		controller.SetIndex(2)
		ts.Pump()
		if !ts.Exists(gluttertest.ByText("Content 3")) {
			t.Error("the tab did not change with the controller")
		}
		ts.Tap(gluttertest.ByText("Tab 1"))
		ts.Pump()
		if controller.Index() != 0 || len(taps) != 1 || taps[0] != 0 {
			t.Errorf("index %d, taps %v after a tap on the first item", controller.Index(), taps)
		}
		if !ts.Exists(gluttertest.ByText("Content 1")) {
			t.Error("the tab did not change with the tap")
		}
		if bar := ts.Widget(gluttertest.ByType[*cupertino.CupertinoTabBar]()).(*cupertino.CupertinoTabBar); bar.CurrentIndex() != 0 {
			t.Errorf("the bar shows the index %d", bar.CurrentIndex())
		}
	})

	t.Run("the controller can be replaced", func(t *testing.T) {
		ts := gluttertest.New(t)
		first := cupertino.NewCupertinoTabController()
		second := cupertino.NewCupertinoTabController(cupertino.CupertinoTabControllerOpts{InitialIndex: 1})
		t.Cleanup(first.Dispose)
		t.Cleanup(second.Dispose)
		pump := func(c *cupertino.CupertinoTabController) {
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabScaffold(cupertino.NewCupertinoTabBar(tabItems("Tab 1", "Tab 2", "Tab 3")),
				func(_ widgets.BuildContext, i int) widgets.Widget {
					return tabContent("Content " + string(rune('1'+i)))
				},
				cupertino.CupertinoTabScaffoldOpts{Controller: c})}))
		}
		pump(first)
		pump(second)
		if !ts.Exists(gluttertest.ByText("Content 2")) {
			t.Error("the new controller's tab does not show")
		}
		if first.HasListeners() || !second.HasListeners() {
			t.Errorf("listeners: first %v, second %v", first.HasListeners(), second.HasListeners())
		}
	})

	t.Run("a shorter bar clamps the index of the scaffold's own controller", func(t *testing.T) {
		ts := gluttertest.New(t)
		pump := func(labels ...string) {
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabScaffold(cupertino.NewCupertinoTabBar(tabItems(labels...)),
				func(_ widgets.BuildContext, i int) widgets.Widget {
					return tabContent("Content " + string(rune('1'+i)))
				})}))
		}
		pump("Tab 1", "Tab 2", "Tab 3")
		ts.Tap(gluttertest.ByText("Tab 3"))
		ts.Pump()
		pump("Tab 1", "Tab 2")
		if !ts.Exists(gluttertest.ByText("Content 2")) {
			t.Error("the index was not clamped to the last tab")
		}
	})

	t.Run("the scaffold's own controller starts at the index of the bar", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabScaffold(
			cupertino.NewCupertinoTabBar(tabItems("Tab 1", "Tab 2"), cupertino.CupertinoTabBarOpts{CurrentIndex: 1}),
			func(_ widgets.BuildContext, i int) widgets.Widget {
				return tabContent("Content " + string(rune('1'+i)))
			})}))
		if !ts.Exists(gluttertest.ByText("Content 2")) {
			t.Error("the second tab does not show")
		}
	})

	t.Run("the active tab has the focus scope that is first", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabScaffold(cupertino.NewCupertinoTabBar(tabItems("Tab 1", "Tab 2")),
			func(_ widgets.BuildContext, i int) widgets.Widget {
				return widgets.NewFocus(tabContent("Content "+string(rune('1'+i))), widgets.FocusOpts{Autofocus: true, DebugLabel: "tab " + string(rune('1'+i))})
			})}))
		ts.Pump()
		focus := ts.Binding.FocusManager().PrimaryFocus()
		if focus == nil || focus.DebugLabel() != "tab 1" {
			t.Fatalf("the primary focus is %v, want the focus of the first tab", focus)
		}
		ts.Tap(gluttertest.ByText("Tab 2"))
		ts.PumpAndSettle()
		focus = ts.Binding.FocusManager().PrimaryFocus()
		if focus == nil || focus.DebugLabel() != "tab 2" {
			t.Errorf("the primary focus is %v, want the focus of the second tab", focus)
		}
	})

	t.Run("the background color is the one given, or the theme's", func(t *testing.T) {
		for _, bg := range []ui.ColorValue{nil, colorOf(0x44)} {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabScaffold(cupertino.NewCupertinoTabBar(tabItems("Tab 1", "Tab 2")),
				func(_ widgets.BuildContext, i int) widgets.Widget { return tabContent("x") },
				cupertino.CupertinoTabScaffoldOpts{BackgroundColor: bg})}))
			box := ts.Widgets(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoTabScaffold](), gluttertest.ByType[*widgets.DecoratedBox]()))[0].(*widgets.DecoratedBox)
			got := box.Decoration().(*painting.BoxDecoration)
			want := cupertino.CupertinoThemeOf(ts.Element(gluttertest.ByType[*cupertino.CupertinoTabScaffold]())).ScaffoldBackgroundColor().EffectiveColor()
			if bg != nil {
				want = bg.EffectiveColor()
			}
			if got.Color != foundation.Some(want) {
				t.Errorf("the background is %v, want %v", got.Color, want)
			}
		}
	})
}

func TestCupertinoTabController(t *testing.T) {
	c := cupertino.NewCupertinoTabController()
	calls := 0
	c.AddListener(func() { calls++ })
	c.SetIndex(0)
	c.SetIndex(2)
	c.SetIndex(2)
	if c.Index() != 2 || calls != 1 {
		t.Errorf("index %d with %d notifications, want 2 with 1", c.Index(), calls)
	}
	c.Dispose()
}

func TestCupertinoTabScaffoldErrors(t *testing.T) {
	skipUnlessDebug(t)
	bar := cupertino.NewCupertinoTabBar(tabItems("Tab 1", "Tab 2"))
	builder := func(widgets.BuildContext, int) widgets.Widget { return tabContent("x") }
	for _, tt := range []struct {
		name string
		make func()
		tag  string
	}{
		{"negative initial index", func() {
			cupertino.NewCupertinoTabController(cupertino.CupertinoTabControllerOpts{InitialIndex: -1})
		}, "NewCupertinoTabController.CupertinoTabControllerIndexNegative"},
		{"negative index", func() { cupertino.NewCupertinoTabController().SetIndex(-1) }, "CupertinoTabController.SetIndex.CupertinoTabControllerIndexNegative"},
		{"a controller past the tabs", func() {
			cupertino.NewCupertinoTabScaffold(bar, builder, cupertino.CupertinoTabScaffoldOpts{
				Controller: cupertino.NewCupertinoTabController(cupertino.CupertinoTabControllerOpts{InitialIndex: 2})})
		}, "NewCupertinoTabScaffold.CupertinoTabScaffoldControllerIndexOutOfRange"},
	} {
		t.Run(tt.name, func(t *testing.T) { requireErrorTag(t, recoverError(tt.make), tt.tag) })
	}

	t.Run("a controller set past the tabs while the scaffold shows", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := cupertino.NewCupertinoTabController()
		t.Cleanup(controller.Dispose)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoTabScaffold(bar, builder, cupertino.CupertinoTabScaffoldOpts{Controller: controller})}))
		err := recoverError(func() { controller.SetIndex(5) })
		requireErrorTag(t, err, "cupertinoTabScaffoldState.onCurrentIndexChange.CupertinoTabScaffoldControllerIndexOutOfRange")
	})
}
