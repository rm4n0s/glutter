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

// These tests are ported from packages/flutter/test/cupertino/scaffold_test.dart. A bar of 44 pixels that
// obstructs or not stands in for the CupertinoNavigationBar (step 34.3); the cases of the tab scaffold, of the
// paint patterns and of the scaling of the text of the bar are not ported.

// testBar is a navigation bar for the tests: 44 pixels high, opaque or translucent.
type testBar struct {
	widgets.StatelessWidget
	opaque bool
}

func newTestBar(opaque bool) *testBar { return &testBar{widgets.NewStatelessWidgetBase(), opaque} }

func (b *testBar) PreferredSize() ui.Size { return ui.Size{Width: 0, Height: 44} }

func (b *testBar) ShouldFullyObstruct(widgets.BuildContext) bool { return b.opaque }

func (b *testBar) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(44.0)})
}

// plain is a widget tree with a MediaQuery of the data and no app.
func plain(data widgets.MediaQueryData, child widgets.Widget) widgets.Widget {
	return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewMediaQuery(data, child))
}

func TestCupertinoPageScaffold(t *testing.T) {
	t.Run("the contents are behind a translucent bar", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(nil),
			cupertino.CupertinoPageScaffoldOpts{NavigationBar: newTestBar(false)})}))
		if got := ts.TopLeft(gluttertest.ByType[*widgets.Center]()); got != (ui.Offset{}) {
			t.Errorf("the contents are at %v, want the origin", got)
		}
	})

	t.Run("an opaque bar pushes the contents down", func(t *testing.T) {
		ts := gluttertest.New(t)
		var child widgets.BuildContext
		ts.PumpWidget(plain(widgets.MediaQueryData{Size: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 1, ViewInsets: painting.EdgeInsets{Top: 20}},
			cupertino.NewCupertinoPageScaffold(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				child = bc
				return widgets.NewContainer(nil)
			}), cupertino.CupertinoPageScaffoldOpts{NavigationBar: newTestBar(true)})))
		if top := widgets.MediaQueryOf(child).Padding.Top; top != 0 {
			t.Errorf("the padding at the top is %v, want 0", top)
		}
		// The contents start 44 pixels from the top: they are pushed down by the opaque bar of 44 pixels.
		got := ts.TopLeft(gluttertest.ByType[*widgets.Container]())
		size := ts.Size(gluttertest.ByType[*widgets.Container]())
		if got != (ui.Offset{Dy: 44}) || size != (ui.Size{Width: 800, Height: 556}) {
			t.Errorf("the contents are at %v with size %v, want (0, 44) and 800 x 556", got, size)
		}
	})

	t.Run("the padding of the contents comes from the view insets", func(t *testing.T) {
		data := widgets.MediaQueryData{Size: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 1, ViewInsets: painting.EdgeInsets{Bottom: 100}}
		container := gluttertest.ByType[*widgets.Container]()
		ts := gluttertest.New(t)
		ts.PumpWidget(plain(data, cupertino.NewCupertinoPageScaffold(widgets.NewContainer(nil),
			cupertino.CupertinoPageScaffoldOpts{NavigationBar: newTestBar(true)})))
		if got := ts.Size(container).Height; got != 600-44-100 {
			t.Errorf("height with an opaque bar = %v, want %v", got, 600-44-100)
		}
		var child widgets.BuildContext
		ts.PumpWidget(plain(data, cupertino.NewCupertinoPageScaffold(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			child = bc
			return widgets.NewContainer(nil)
		}), cupertino.CupertinoPageScaffoldOpts{NavigationBar: newTestBar(false)})))
		if got := ts.Size(container).Height; got != 600-100 {
			t.Errorf("height with a translucent bar = %v, want %v", got, 600-100)
		}
		// The contents do not see the view inset: the scaffold consumed it.
		if got := widgets.MediaQueryOf(child).ViewInsets.Bottom; got != 0 {
			t.Errorf("the contents see a bottom view inset of %v, want 0", got)
		}
		ts.PumpWidget(plain(data, cupertino.NewCupertinoPageScaffold(widgets.NewContainer(nil),
			cupertino.CupertinoPageScaffoldOpts{NavigationBar: newTestBar(false), NotResizeToAvoidBottomInset: true})))
		if got := ts.Size(container).Height; got != 600 {
			t.Errorf("height without resizing = %v, want 600", got)
		}
	})

	t.Run("it resizes to avoid the bottom inset when there is no bar", func(t *testing.T) {
		data := widgets.MediaQueryData{Size: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 1, ViewInsets: painting.EdgeInsets{Bottom: 100}}
		ts := gluttertest.New(t)
		ts.PumpWidget(plain(data, cupertino.NewCupertinoPageScaffold(widgets.NewContainer(nil))))
		if got := ts.Size(gluttertest.ByType[*widgets.Container]()).Height; got != 500 {
			t.Errorf("height = %v, want 500", got)
		}
		ts.PumpWidget(plain(data, cupertino.NewCupertinoPageScaffold(widgets.NewContainer(nil), cupertino.CupertinoPageScaffoldOpts{NotResizeToAvoidBottomInset: true})))
		if got := ts.Size(gluttertest.ByType[*widgets.Container]()).Height; got != 600 {
			t.Errorf("height without resizing = %v, want 600", got)
		}
	})

	t.Run("the background is white by default, and the given color overrides it", func(t *testing.T) {
		for _, tt := range []struct {
			name  string
			color ui.ColorValue
			want  ui.Color
		}{{"default", nil, ui.NewColor(0xFFFFFFFF)}, {"overridden", ui.NewColor(0xFF010203), ui.NewColor(0xFF010203)}} {
			ts := gluttertest.New(t)
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(nil),
				cupertino.CupertinoPageScaffoldOpts{BackgroundColor: tt.color})}))
			box := ts.Widget(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoPageScaffold](), gluttertest.ByType[*widgets.DecoratedBox]())).(*widgets.DecoratedBox)
			decoration, ok := box.Decoration().(*painting.BoxDecoration)
			if !ok {
				t.Fatalf("%s: the decoration is %T", tt.name, box.Decoration())
			}
			if got := decoration.Color; got != foundation.Some(tt.want) {
				t.Errorf("%s: the background is %v, want %v", tt.name, got, tt.want)
			}
		}
	})

	t.Run("a dynamic background follows the brightness", func(t *testing.T) {
		dynamic := cupertino.NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF000000))
		for _, tt := range []struct {
			brightness ui.Brightness
			want       ui.Color
		}{{ui.BrightnessLight, ui.NewColor(0xFFFFFFFF)}, {ui.BrightnessDark, ui.NewColor(0xFF000000)}} {
			ts := gluttertest.New(t)
			var got foundation.Opt[ui.Color]
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{
				Theme: cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: tt.brightness}),
				Home: cupertino.NewCupertinoPageScaffold(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
					got = cupertino.CupertinoPageScaffoldBackgroundColorMaybeOf(bc)
					return widgets.NewCenter(nil)
				}), cupertino.CupertinoPageScaffoldOpts{BackgroundColor: dynamic})}))
			if got != foundation.Some(tt.want) {
				t.Errorf("brightness %v: the background is %v, want %v", tt.brightness, got, tt.want)
			}
		}
	})

	t.Run("it does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewSizedBoxShrink(
			cupertino.NewCupertinoPageScaffold(widgets.NewText("X"))))}))
		if size := ts.Size(gluttertest.ByType[*cupertino.CupertinoPageScaffold]()); size != (ui.Size{}) {
			t.Errorf("size = %v, want zero", size)
		}
	})
}

// scaffoldWithPrimaryScrollView is a page scaffold with a scroll view that uses the primary controller, which
// starts 1000 pixels down.
type scaffoldWithPrimaryScrollView struct {
	widgets.StatefulWidget
	height float64
	got    **widgets.ScrollControllerBase
}

func (w *scaffoldWithPrimaryScrollView) CreateState() widgets.State {
	return &scaffoldWithPrimaryScrollViewState{}
}

type scaffoldWithPrimaryScrollViewState struct {
	widgets.StateOf[*scaffoldWithPrimaryScrollView]
	controller *widgets.ScrollControllerBase
}

func (s *scaffoldWithPrimaryScrollViewState) InitState() {
	s.controller = widgets.NewScrollController(widgets.ScrollControllerOpts{InitialScrollOffset: 1000})
	if s.Widget().got != nil {
		*s.Widget().got = s.controller
	}
}

func (s *scaffoldWithPrimaryScrollViewState) Dispose() { s.controller.Dispose() }

func (s *scaffoldWithPrimaryScrollViewState) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewMediaQuery(widgets.MediaQueryData{Size: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 1, Padding: painting.EdgeInsets{Top: 25}},
		widgets.NewPrimaryScrollController(s.controller, cupertino.NewCupertinoPageScaffold(widgets.NewSingleChildScrollView(
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(s.Widget().height)}),
			widgets.SingleChildScrollViewOpts{Primary: foundation.Some(true)}))))
}

func TestCupertinoPageScaffoldStatusBarTap(t *testing.T) {
	t.Run("a tap on the status bar scrolls to the top", func(t *testing.T) {
		ts := gluttertest.New(t)
		var controller *widgets.ScrollControllerBase
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: &scaffoldWithPrimaryScrollView{widgets.NewStatefulWidgetBase(), 12345, &controller}}))
		if controller.Offset() != 1000 {
			t.Fatalf("the controller starts at %v, want 1000", controller.Offset())
		}
		ts.SimulateStatusBarTap()
		ts.PumpAndSettle()
		if controller.Offset() != 0 {
			t.Errorf("the controller is at %v, want 0", controller.Offset())
		}
	})

	t.Run("a tap on the status bar scrolls only the primary controller of the page in the foreground", func(t *testing.T) {
		ts := gluttertest.New(t)
		var first, second *widgets.ScrollControllerBase
		page := func(got **widgets.ScrollControllerBase) widgets.WidgetBuilder {
			return func(widgets.BuildContext) widgets.Widget {
				return &scaffoldWithPrimaryScrollView{widgets.NewStatefulWidgetBase(), 2000, got}
			}
		}
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{
			InitialRoute: "a",
			OnGenerateInitialRoutes: func(string) []widgets.Route {
				return []widgets.Route{cupertino.NewCupertinoPageRoute(page(&first)), cupertino.NewCupertinoPageRoute(page(&second))}
			},
			OnGenerateRoute: func(widgets.RouteSettings) widgets.Route { panic("no route") },
		}))
		if first.Offset() != 1000 || second.Offset() != 1000 {
			t.Fatalf("the controllers start at %v and %v, want 1000", first.Offset(), second.Offset())
		}
		ts.SimulateStatusBarTap()
		ts.PumpAndSettle()
		if first.Offset() != 1000 || second.Offset() != 0 {
			t.Errorf("the controllers are at %v and %v, want 1000 and 0", first.Offset(), second.Offset())
		}
	})
}
