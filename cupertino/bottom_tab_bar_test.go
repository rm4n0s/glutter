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

// These tests are ported from packages/flutter/test/cupertino/bottom_tab_bar_test.dart. Not ported: the
// semantics tester cases and the test of the web cursor.

// tabItems is the items of the tests: icons that are boxes of 24 by 24 pixels, with a label each.
func tabItems(labels ...string) []widgets.BottomNavigationBarItem {
	var items []widgets.BottomNavigationBarItem
	for _, l := range labels {
		items = append(items, widgets.NewBottomNavigationBarItem(
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)}),
			widgets.BottomNavigationBarItemOpts{Label: l}))
	}
	return items
}

// withPadding runs the child under a MediaQuery whose view padding at the bottom is bottom.
func withPadding(bottom float64, child widgets.Widget) widgets.Widget {
	return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		data := widgets.MediaQueryOf(bc)
		pad := painting.EdgeInsets{Bottom: bottom}
		return widgets.NewMediaQuery(data.CopyWith(widgets.MediaQueryDataCopyOpts{Padding: foundation.Some(pad), ViewPadding: foundation.Some(pad)}), child)
	})
}

func TestCupertinoTabBar(t *testing.T) {
	t.Run("the bar is as high as it says, plus the bottom padding", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: withPadding(40, widgets.NewAlign(cupertino.NewCupertinoTabBar(tabItems("One", "Two")),
			widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter}))}))
		if got, want := ts.Size(gluttertest.ByType[*cupertino.CupertinoTabBar]()), (ui.Size{Width: 800, Height: 90}); got != want {
			t.Errorf("size = %v, want %v", got, want)
		}
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(cupertino.NewCupertinoTabBar(tabItems("One", "Two"),
			cupertino.CupertinoTabBarOpts{Height: foundation.Some(70.0)}), widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})}))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoTabBar]()).Height; got != 70 {
			t.Errorf("height = %v, want 70", got)
		}
	})

	t.Run("the preferred size has the height", func(t *testing.T) {
		bar := cupertino.NewCupertinoTabBar(tabItems("One", "Two"), cupertino.CupertinoTabBarOpts{Height: foundation.Some(61.0)})
		if bar.PreferredSize().Height != 61 || cupertino.NewCupertinoTabBar(tabItems("One", "Two")).PreferredSize().Height != 50 {
			t.Errorf("preferred size = %v", bar.PreferredSize())
		}
	})

	t.Run("a tap calls onTap with the index", func(t *testing.T) {
		ts := gluttertest.New(t)
		var taps []int
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(cupertino.NewCupertinoTabBar(tabItems("One", "Two", "Three"),
			cupertino.CupertinoTabBarOpts{OnTap: func(i int) { taps = append(taps, i) }}), widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})}))
		ts.Tap(gluttertest.ByText("Two"))
		ts.Tap(gluttertest.ByText("Three"))
		ts.Tap(gluttertest.ByText("One"))
		if len(taps) != 3 || taps[0] != 1 || taps[1] != 2 || taps[2] != 0 {
			t.Errorf("taps = %v, want [1 2 0]", taps)
		}
	})

	t.Run("a bar without onTap takes taps", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(cupertino.NewCupertinoTabBar(tabItems("One", "Two")),
			widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})}))
		ts.Tap(gluttertest.ByText("Two"))
	})

	t.Run("the active item has the active color and the others the inactive one", func(t *testing.T) {
		ts := gluttertest.New(t)
		active, inactive := colorOf(0x11), colorOf(0x22)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(cupertino.NewCupertinoTabBar(tabItems("One", "Two"),
			cupertino.CupertinoTabBarOpts{CurrentIndex: 1, ActiveColor: active, InactiveColor: inactive}), widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})}))
		colorOfLabel := func(label string) ui.Color {
			var color ui.Color
			el := ts.Element(gluttertest.ByText(label))
			color = widgets.DefaultTextStyleOf(el).Style().Color.Value()
			return color
		}
		if got := colorOfLabel("One"); got != inactive {
			t.Errorf("the inactive label has the color %v, want %v", got, inactive)
		}
		if got := colorOfLabel("Two"); got != active {
			t.Errorf("the active label has the color %v, want %v", got, active)
		}
	})

	t.Run("the active icon shows for the active item", func(t *testing.T) {
		ts := gluttertest.New(t)
		icon := widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey("icon"), Width: foundation.Some(10.0)})
		active := widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey("active"), Width: foundation.Some(10.0)})
		items := append(tabItems("One"), widgets.NewBottomNavigationBarItem(icon, widgets.BottomNavigationBarItemOpts{ActiveIcon: active, Label: "Two"}))
		for index, want := range []string{"icon", "active"} {
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(cupertino.NewCupertinoTabBar(items,
				cupertino.CupertinoTabBarOpts{CurrentIndex: index}), widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})}))
			if !ts.Exists(gluttertest.ByKey(foundation.NewValueKey(want))) {
				t.Errorf("index %d: the %s icon is missing", index, want)
			}
		}
	})

	t.Run("a translucent bar blurs, an opaque one does not", func(t *testing.T) {
		for _, tt := range []struct {
			background ui.Color
			blur       bool
		}{{ui.NewColor(0xFF112233), false}, {ui.NewColor(0x80112233), true}} {
			ts := gluttertest.New(t)
			bar := cupertino.NewCupertinoTabBar(tabItems("One", "Two"), cupertino.CupertinoTabBarOpts{BackgroundColor: tt.background})
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(bar, widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})}))
			if got := ts.Exists(gluttertest.ByType[*widgets.BackdropFilter]()); got != tt.blur {
				t.Errorf("background %v: blur = %v, want %v", tt.background, got, tt.blur)
			}
		}
	})

	t.Run("the default background is the one of the theme", func(t *testing.T) {
		ts := gluttertest.New(t)
		bar := cupertino.NewCupertinoTabBar(tabItems("One", "Two"))
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(bar, widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})}))
		// The bar background of the light theme is translucent.
		if ts.Element(gluttertest.ByType[*cupertino.CupertinoTabBar]()) == nil || bar.Opaque(ts.Element(gluttertest.ByType[*cupertino.CupertinoTabBar]())) {
			t.Error("the default bar is opaque")
		}
	})

	t.Run("the default border is a hairline on top, and it can be changed or removed", func(t *testing.T) {
		borderOf := func(opts cupertino.CupertinoTabBarOpts) painting.BoxBorder {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(cupertino.NewCupertinoTabBar(tabItems("One", "Two"), opts),
				widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})}))
			box := ts.Widget(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoTabBar](), gluttertest.ByType[*widgets.DecoratedBox]())).(*widgets.DecoratedBox)
			return box.Decoration().(*painting.BoxDecoration).Border
		}
		def, ok := borderOf(cupertino.CupertinoTabBarOpts{}).(painting.Border)
		if !ok || def.Top.Width != 0 || def.Top.Style != painting.BorderStyleSolid || def.Bottom != painting.BorderSideNone {
			t.Errorf("default border = %#v", def)
		}
		given := painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(colorOf(0x33)), Width: foundation.Some(2.0)})
		if got := borderOf(cupertino.CupertinoTabBarOpts{Border: foundation.Some(given)}); got != painting.BoxBorder(given) {
			t.Errorf("given border = %v", got)
		}
		if got := borderOf(cupertino.CupertinoTabBarOpts{NoBorder: true}); got != nil {
			t.Errorf("border = %v, want none", got)
		}
	})

	t.Run("the tab bar is not scaled by the text scaler of the app", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(cupertino.NewCupertinoTabBar(tabItems("One", "Two")),
			widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})}))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoTabBar]()).Height; got != 50 {
			t.Errorf("height = %v, want 50", got)
		}
	})

	t.Run("CopyWith replaces what is given", func(t *testing.T) {
		bar := cupertino.NewCupertinoTabBar(tabItems("One", "Two"), cupertino.CupertinoTabBarOpts{CurrentIndex: 1, IconSize: foundation.Some(20.0), ActiveColor: colorOf(1)})
		c := bar.CopyWith(cupertino.CupertinoTabBarCopyOpts{CurrentIndex: foundation.Some(0), Height: foundation.Some(44.0), InactiveColor: colorOf(2)})
		if c.CurrentIndex() != 0 || c.Height() != 44 || c.IconSize() != 20 || c.ActiveColor() != ui.ColorValue(colorOf(1)) || c.InactiveColor() != ui.ColorValue(colorOf(2)) || len(c.Items()) != 2 {
			t.Errorf("copy = %+v", c)
		}
	})
}

func TestCupertinoTabBarErrors(t *testing.T) {
	skipUnlessDebug(t)
	for _, tt := range []struct {
		name string
		make func()
		tag  string
	}{
		{"one item", func() { cupertino.NewCupertinoTabBar(tabItems("One")) }, "NewCupertinoTabBar.CupertinoTabBarItemsTooFew"},
		{"index too high", func() {
			cupertino.NewCupertinoTabBar(tabItems("One", "Two"), cupertino.CupertinoTabBarOpts{CurrentIndex: 2})
		}, "NewCupertinoTabBar.CupertinoTabBarCurrentIndexInvalid"},
		{"negative index", func() {
			cupertino.NewCupertinoTabBar(tabItems("One", "Two"), cupertino.CupertinoTabBarOpts{CurrentIndex: -1})
		}, "NewCupertinoTabBar.CupertinoTabBarCurrentIndexInvalid"},
		{"negative height", func() {
			cupertino.NewCupertinoTabBar(tabItems("One", "Two"), cupertino.CupertinoTabBarOpts{Height: foundation.Some(-1.0)})
		}, "NewCupertinoTabBar.CupertinoTabBarHeightInvalid"},
	} {
		t.Run(tt.name, func(t *testing.T) { checkErrorTag(t, recoverError(tt.make), tt.tag) })
	}
}
