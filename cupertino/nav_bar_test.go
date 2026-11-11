package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/nav_bar_test.dart. Not ported: the semantics
// cases, the golden image tests of Flutter, and the cases that set the system UI style by the platform channel
// (those are checked with the binding's SystemChrome).

// pageWithBar is a page scaffold with the navigation bar and a text as the content.
func pageWithBar(bar cupertino.ObstructingPreferredSizeWidget, content string) widgets.Widget {
	return cupertino.NewCupertinoPageScaffold(widgets.NewText(content), cupertino.CupertinoPageScaffoldOpts{NavigationBar: bar})
}

func TestCupertinoNavigationBar(t *testing.T) {
	t.Run("the middle shows and the bar has the height of its content", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: pageWithBar(cupertino.NewCupertinoNavigationBar(
			cupertino.CupertinoNavigationBarOpts{Middle: widgets.NewText("Title")}), "Content")}))
		if !ts.Exists(gluttertest.ByText("Title")) {
			t.Fatal("the middle is not shown")
		}
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoNavigationBar]()).Height; got != 44 {
			t.Errorf("the height is %v, want 44", got)
		}
		if got := ts.Center(gluttertest.ByText("Title")).Dx; got != 400 {
			t.Errorf("the title is centered at %v, want 400", got)
		}
	})

	t.Run("a large bar is as high as the title adds", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: pageWithBar(cupertino.NewCupertinoNavigationBarLarge(
			cupertino.CupertinoNavigationBarOpts{LargeTitle: widgets.NewText("Large")}), "Content")}))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoNavigationBar]()).Height; got != 96 {
			t.Errorf("the height is %v, want 96", got)
		}
		if !ts.Exists(gluttertest.ByText("Large")) {
			t.Error("the large title is not shown")
		}
		if h := cupertino.NewCupertinoNavigationBarLarge(cupertino.CupertinoNavigationBarOpts{LargeTitle: widgets.NewText("x")}).PreferredSize().Height; h != 96 {
			t.Errorf("preferred height = %v, want 96", h)
		}
	})

	t.Run("a pushed page has a back button with the title of the previous page", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: pageWithBar(cupertino.NewCupertinoNavigationBar(
			cupertino.CupertinoNavigationBarOpts{Middle: widgets.NewText("Home")}), "Home content")}))
		ts.Element(gluttertest.ByType[*cupertino.CupertinoNavigationBar]())
		nav := widgets.NavigatorOf(ts.Element(gluttertest.ByType[*cupertino.CupertinoNavigationBar]()))
		nav.Push(cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget {
			return pageWithBar(cupertino.NewCupertinoNavigationBar(cupertino.CupertinoNavigationBarOpts{Middle: widgets.NewText("Second")}), "Second content")
		}, cupertino.CupertinoPageRouteOpts{Title: foundation.Some("First")}))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByType[*cupertino.CupertinoNavigationBarBackButton]()) {
			t.Error("the second page has no back button")
		}
	})

	t.Run("a leading widget replaces the back button", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: pageWithBar(cupertino.NewCupertinoNavigationBar(
			cupertino.CupertinoNavigationBarOpts{Leading: widgets.NewText("Lead"), Middle: widgets.NewText("T"), Trailing: widgets.NewText("Trail")}), "c")}))
		if !ts.Exists(gluttertest.ByText("Lead")) || !ts.Exists(gluttertest.ByText("Trail")) {
			t.Error("the leading or the trailing widget is not shown")
		}
		if got, want := ts.TopLeft(gluttertest.ByText("Lead")).Dx, 16.0; got != want {
			t.Errorf("the leading widget starts at %v, want %v", got, want)
		}
		if got, want := ts.BottomRight(gluttertest.ByText("Trail")).Dx, 800.0-16; got != want {
			t.Errorf("the trailing widget ends at %v, want %v", got, want)
		}
	})

	t.Run("hero tag with no transition is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		err := recoverError(func() {
			cupertino.NewCupertinoNavigationBar(cupertino.CupertinoNavigationBarOpts{HeroTag: "tag"})
		})
		checkErrorTag(t, err, "NewCupertinoNavigationBar.NavigationBarHeroTagWithoutTransition")
		err = recoverError(func() {
			cupertino.NewCupertinoNavigationBarLarge(cupertino.CupertinoNavigationBarOpts{HeroTag: "tag"})
		})
		checkErrorTag(t, err, "NewCupertinoNavigationBarLarge.NavigationBarHeroTagWithoutTransition")
	})

	t.Run("a bar that is opaque obstructs, and a translucent one does not", func(t *testing.T) {
		ts := gluttertest.New(t)
		var opaque, translucent bool
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			opaque = cupertino.NewCupertinoNavigationBar(cupertino.CupertinoNavigationBarOpts{BackgroundColor: ui.NewColor(0xFF112233)}).ShouldFullyObstruct(bc)
			translucent = cupertino.NewCupertinoNavigationBar().ShouldFullyObstruct(bc)
			return widgets.NewText("x")
		})}))
		if !opaque || translucent {
			t.Errorf("opaque %v, translucent %v, want true and false", opaque, translucent)
		}
	})

	t.Run("the status bar style follows the background", func(t *testing.T) {
		for _, tt := range []struct {
			background ui.Color
			want       ui.Brightness
		}{{ui.NewColor(0xFF000000), ui.BrightnessLight}, {ui.NewColor(0xFFFFFFFF), ui.BrightnessDark}} {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: pageWithBar(cupertino.NewCupertinoNavigationBar(
				cupertino.CupertinoNavigationBarOpts{BackgroundColor: tt.background, Middle: widgets.NewText("T"), NotAutomaticBackgroundVisibility: true}), "c")}))
			ts.Pump()
			style, ok := ts.Binding.SystemChrome().LatestStyle()
			if !ok {
				t.Fatalf("background %v: no system UI style was set", tt.background)
			}
			if style.StatusBarIconBrightness != tt.want {
				t.Errorf("background %v: status bar icons are %v, want %v", tt.background, style.StatusBarIconBrightness, tt.want)
			}
		}
	})

	t.Run("a bar that is not opaque blurs and one that is opaque does not", func(t *testing.T) {
		for _, tt := range []struct {
			background ui.Color
			blur       bool
		}{{ui.NewColor(0xFF112233), false}, {ui.NewColor(0x80112233), true}} {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: pageWithBar(cupertino.NewCupertinoNavigationBar(
				cupertino.CupertinoNavigationBarOpts{BackgroundColor: tt.background, NotAutomaticBackgroundVisibility: true}), "c")}))
			filter := ts.Widget(gluttertest.ByType[*widgets.BackdropFilter]()).(*widgets.BackdropFilter)
			if filter.Enabled().OrElse(true) != tt.blur {
				t.Errorf("background %v: blur = %v, want %v", tt.background, filter.Enabled(), tt.blur)
			}
		}
	})

	t.Run("the bar has no border when asked and a hairline by default", func(t *testing.T) {
		borderOf := func(opts cupertino.CupertinoNavigationBarOpts) painting.BoxBorder {
			ts := gluttertest.New(t)
			opts.NotAutomaticBackgroundVisibility = true
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: pageWithBar(cupertino.NewCupertinoNavigationBar(opts), "c")}))
			box := ts.Widgets(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoNavigationBar](), gluttertest.ByType[*widgets.DecoratedBox]()))[0].(*widgets.DecoratedBox)
			return box.Decoration().(*painting.BoxDecoration).Border
		}
		def, ok := borderOf(cupertino.CupertinoNavigationBarOpts{}).(painting.Border)
		if !ok || def.Bottom.Width != 0 || def.Bottom.Style != painting.BorderStyleSolid || def.Bottom.Color != ui.NewColor(0x4D000000) {
			t.Errorf("default border = %#v", def)
		}
		if got := borderOf(cupertino.CupertinoNavigationBarOpts{NoBorder: true}); got != nil {
			t.Errorf("border = %v, want none", got)
		}
	})

	t.Run("the bottom is under the persistent bar", func(t *testing.T) {
		ts := gluttertest.New(t)
		bottom := widgets.NewPreferredSize(ui.Size{Height: 30}, widgets.NewText("Bottom"))
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: pageWithBar(cupertino.NewCupertinoNavigationBar(
			cupertino.CupertinoNavigationBarOpts{Middle: widgets.NewText("T"), Bottom: bottom}), "c")}))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoNavigationBar]()).Height; got != 74 {
			t.Errorf("the height is %v, want 74", got)
		}
		if got := ts.TopLeft(gluttertest.ByText("Bottom")).Dy; got != 44 {
			t.Errorf("the bottom starts at %v, want 44", got)
		}
	})
}

func TestCupertinoSliverNavigationBar(t *testing.T) {
	list := func(bar widgets.Widget) widgets.Widget {
		var items []widgets.Widget
		for i := 0; i < 30; i++ {
			items = append(items, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(50.0)}))
		}
		return cupertino.NewCupertinoPageScaffold(widgets.NewCustomScrollView([]widgets.Widget{bar, widgets.NewSliverList(widgets.NewSliverChildListDelegate(items))}))
	}
	t.Run("the large title shows and collapses into the middle", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t)
		ts.SetViewSize(ui.Size{Width: 400, Height: 800})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: list(cupertino.NewCupertinoSliverNavigationBar(
			cupertino.CupertinoSliverNavigationBarOpts{LargeTitle: widgets.NewText("Large"), Middle: widgets.NewText("Mid")}))}))
		if h := ts.Size(gluttertest.ByType[*widgets.BackdropFilter]()).Height; h != 96 {
			t.Errorf("the header is %v high, want 96", h)
		}
		ts.DragFrom(ui.Offset{Dx: 200, Dy: 400}, ui.Offset{Dy: -200})
		ts.PumpAndSettle()
		if h := ts.Size(gluttertest.ByType[*widgets.BackdropFilter]()).Height; h != 44 {
			t.Errorf("the collapsed header is %v high, want 44", h)
		}
		if !ts.Exists(gluttertest.ByText("Mid")) {
			t.Error("the middle is not shown")
		}
	})

	t.Run("a bar with neither a title nor the route's is an error", func(t *testing.T) {
		skipUnlessDebug(t)
		err := recoverError(func() {
			cupertino.NewCupertinoSliverNavigationBar(cupertino.CupertinoSliverNavigationBarOpts{NotAutomaticallyImplyTitle: true})
		})
		checkErrorTag(t, err, "NewCupertinoSliverNavigationBar.NavigationBarLargeTitleRequired")
		err = recoverError(func() {
			cupertino.NewCupertinoSliverNavigationBar(cupertino.CupertinoSliverNavigationBarOpts{BottomMode: cupertino.NavigationBarBottomModeAlways})
		})
		checkErrorTag(t, err, "NewCupertinoSliverNavigationBar.NavigationBarBottomModeWithoutBottom")
	})

	t.Run("the search bar shows the field", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t)
		ts.SetViewSize(ui.Size{Width: 400, Height: 800})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: list(cupertino.NewCupertinoSliverNavigationBarSearch(
			widgets.NewText("Search"), cupertino.CupertinoSliverNavigationBarOpts{LargeTitle: widgets.NewText("Large")}))}))
		if !ts.Exists(gluttertest.ByText("Search")) {
			t.Fatal("the search field is not shown")
		}
		_ = rendering.HitTestBehaviorOpaque
	})
}
