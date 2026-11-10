package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// reduceTransparency turns the system's Reduce transparency on or off for the tester (SPEC §12.10: Flutter does not read it).
func reduceTransparency(ts *gluttertest.Tester, on bool) {
	ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.AccessibilityFeatures.ReduceTransparency = on })
}

// With Reduce transparency the bars and the popup surfaces are opaque and do not blur what is behind them.
func TestReduceTransparencyMakesTheSurfacesOpaque(t *testing.T) {
	translucent := ui.NewColor(0x80112233)
	t.Run("a navigation bar", func(t *testing.T) {
		for _, tt := range []struct {
			reduce, blur bool
		}{{false, true}, {true, false}} {
			ts := gluttertest.New(t)
			reduceTransparency(ts, tt.reduce)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: pageWithBar(cupertino.NewCupertinoNavigationBar(
				cupertino.CupertinoNavigationBarOpts{BackgroundColor: translucent, NotAutomaticBackgroundVisibility: true}), "c")}))
			filter := ts.Widget(gluttertest.ByType[*widgets.BackdropFilter]()).(*widgets.BackdropFilter)
			if got := filter.Enabled().OrElse(true); got != tt.blur {
				t.Errorf("reduce transparency %v: blur = %v, want %v", tt.reduce, got, tt.blur)
			}
		}
	})
	t.Run("a tab bar", func(t *testing.T) {
		for _, tt := range []struct {
			reduce, blur bool
		}{{false, true}, {true, false}} {
			ts := gluttertest.New(t)
			reduceTransparency(ts, tt.reduce)
			bar := cupertino.NewCupertinoTabBar(tabItems("One", "Two"), cupertino.CupertinoTabBarOpts{BackgroundColor: translucent})
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(bar, widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})}))
			if got := ts.Exists(gluttertest.ByType[*widgets.BackdropFilter]()); got != tt.blur {
				t.Errorf("reduce transparency %v: blur = %v, want %v", tt.reduce, got, tt.blur)
			}
		}
	})
	t.Run("a popup surface", func(t *testing.T) {
		for _, tt := range []struct {
			reduce, blur bool
		}{{false, true}, {true, false}} {
			ts := dialogTester(t, ui.Size{})
			reduceTransparency(ts, tt.reduce)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoPopupSurface(widgets.NewText("X")))}))
			surface := gluttertest.ByType[*cupertino.CupertinoPopupSurface]()
			if got := ts.Exists(gluttertest.Descendant(surface, gluttertest.ByType[*widgets.BackdropFilter]())); got != tt.blur {
				t.Errorf("reduce transparency %v: blur = %v, want %v", tt.reduce, got, tt.blur)
			}
			colored := ts.Widget(gluttertest.Descendant(surface, gluttertest.ByType[*widgets.ColoredBox]())).(*widgets.ColoredBox)
			if opaque := colored.Color().Alpha() == 0xFF; opaque != tt.reduce {
				t.Errorf("reduce transparency %v: the surface color %v is opaque = %v", tt.reduce, colored.Color(), opaque)
			}
		}
	})
	t.Run("the setting turned on while the bar is shown", func(t *testing.T) {
		ts := gluttertest.New(t)
		bar := cupertino.NewCupertinoTabBar(tabItems("One", "Two"), cupertino.CupertinoTabBarOpts{BackgroundColor: translucent})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewAlign(bar, widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})}))
		if !ts.Exists(gluttertest.ByType[*widgets.BackdropFilter]()) {
			t.Fatal("the translucent bar does not blur")
		}
		reduceTransparency(ts, true)
		ts.Pump()
		if ts.Exists(gluttertest.ByType[*widgets.BackdropFilter]()) {
			t.Error("the bar still blurs after Reduce transparency was turned on")
		}
	})
}
