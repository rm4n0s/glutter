package cupertino_test

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The tests of CupertinoListTile, ported from cupertino/list_tile_test.dart (step 34.8). The test of the long press that
// animates the two switches reads the private position of the switch state in Flutter: here the switches are checked
// by the drawing of the goldens instead.

func rawTileApp(child widgets.Widget) widgets.Widget {
	return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewMediaQuery(widgets.MediaQueryData{DevicePixelRatio: 1},
		cupertino.NewCupertinoListSection(cupertino.CupertinoListSectionOpts{Children: []widgets.Widget{child}})))
}

func tileColoredBox(t *testing.T, ts *gluttertest.Tester) ui.Color {
	t.Helper()
	f := gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoListTile](), gluttertest.ByType[*widgets.ColoredBox]())
	return ts.Widget(f).(*widgets.ColoredBox).Color()
}

func TestCupertinoListTile(t *testing.T) {
	t.Run("shows the parts", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(sectionApp(cupertino.NewCupertinoListTile(widgets.NewText("Title"), cupertino.CupertinoListTileOpts{
			Subtitle: widgets.NewText("Subtitle"), AdditionalInfo: widgets.NewText("Not Connected"),
			Leading: widgets.NewIcon(cupertino.CupertinoIcons.Add()), Trailing: widgets.NewIcon(cupertino.CupertinoIcons.RightChevron())})))
		for _, text := range []string{"Title", "Subtitle", "Not Connected"} {
			if ts.Count(gluttertest.ByText(text)) != 1 {
				t.Errorf("%q is not shown once", text)
			}
		}
		if n := ts.Count(gluttertest.ByType[*widgets.Icon]()); n != 2 {
			t.Errorf("%d icons, want 2", n)
		}
	})
	t.Run("backgroundColor", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		red := cupertino.CupertinoColors.SystemRed
		ts.PumpWidget(rawTileApp(cupertino.NewCupertinoListTile(widgets.NewText("T"), cupertino.CupertinoListTileOpts{BackgroundColor: red})))
		if got, want := tileColoredBox(t, ts), cupertino.CupertinoDynamicColorResolve(red, ts.Element(gluttertest.ByType[*cupertino.CupertinoListTile]())); got != want {
			t.Errorf("the background is %v, want %v", got, want)
		}
	})
	t.Run("a tile with no onTap does not change its color when tapped, and has no GestureDetector", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		blue, red := cupertino.CupertinoColors.SystemBlue, cupertino.CupertinoColors.SystemRed
		ts.PumpWidget(rawTileApp(cupertino.NewCupertinoListTile(widgets.NewText("T"), cupertino.CupertinoListTileOpts{BackgroundColor: blue, BackgroundColorActivated: red})))
		if ts.Exists(gluttertest.ByType[*widgets.GestureDetector]()) {
			t.Error("a tile with no onTap has a GestureDetector")
		}
		ts.Tap(gluttertest.ByType[*cupertino.CupertinoListTile]())
		ts.Pump()
		if got, want := tileColoredBox(t, ts), cupertino.CupertinoDynamicColorResolve(blue, ts.Element(gluttertest.ByType[*cupertino.CupertinoListTile]())); got != want {
			t.Errorf("the background is %v, want %v", got, want)
		}
	})
	t.Run("a tile with onTap shows its activated color while it is pressed", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		blue, red := cupertino.CupertinoColors.SystemBlue, cupertino.CupertinoColors.SystemRed
		ts.PumpWidget(rawTileApp(cupertino.NewCupertinoListTile(widgets.NewText("T"), cupertino.CupertinoListTileOpts{
			BackgroundColor: blue, BackgroundColorActivated: red, OnTap: func() {}})))
		if n := ts.Count(gluttertest.ByType[*widgets.GestureDetector]()); n != 1 {
			t.Errorf("%d GestureDetectors, want 1", n)
		}
		bc := ts.Element(gluttertest.ByType[*cupertino.CupertinoListTile]())
		if got, want := tileColoredBox(t, ts), cupertino.CupertinoDynamicColorResolve(blue, bc); got != want {
			t.Errorf("the background is %v, want %v", got, want)
		}
		g := ts.StartGesture(ts.Center(gluttertest.ByType[*cupertino.CupertinoListTile]()))
		ts.Pump()
		if got, want := tileColoredBox(t, ts), cupertino.CupertinoDynamicColorResolve(red, bc); got != want {
			t.Errorf("while pressed the background is %v, want %v", got, want)
		}
		g.Up()
		ts.PumpAndSettle()
	})
	t.Run("resets the background color when navigated back", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		blue, red := cupertino.CupertinoColors.SystemBlue, cupertino.CupertinoColors.SystemRed
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			second := widgets.NewCenter(cupertino.NewCupertinoButton(func() { widgets.NavigatorOf(bc).Pop() }, widgets.NewText("Go back")))
			return widgets.NewCenter(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewMediaQuery(widgets.MediaQueryData{DevicePixelRatio: 1},
				cupertino.NewCupertinoListTile(widgets.NewText("T"), cupertino.CupertinoListTileOpts{BackgroundColor: blue, BackgroundColorActivated: red,
					OnTap: func() {
						cupertino.NavigatorPush(bc, cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget { return second }))
					}}))))
		})}))
		ts.Tap(gluttertest.ByType[*cupertino.CupertinoListTile]())
		ts.PumpAndSettle()
		ts.Tap(gluttertest.ByType[*cupertino.CupertinoButton]())
		ts.PumpAndSettle()
		if got, want := tileColoredBox(t, ts), cupertino.CupertinoDynamicColorResolve(blue, ts.Element(gluttertest.ByType[*cupertino.CupertinoListTile]())); got != want {
			t.Errorf("the background is %v, want %v", got, want)
		}
	})
	for _, dir := range []struct {
		name string
		d    ui.TextDirection
	}{{"left to right", ui.TextDirectionLTR}, {"right to left", ui.TextDirectionRTL}} {
		t.Run("alignment "+dir.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{})
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewDirectionality(dir.d,
				cupertino.NewCupertinoListTile(widgets.NewText("Title"), cupertino.CupertinoListTileOpts{
					Subtitle: widgets.NewText("Subtitle"), AdditionalInfo: widgets.NewText("Info"),
					Leading: widgets.NewIcon(cupertino.CupertinoIcons.Add()), Trailing: widgets.NewIcon(cupertino.CupertinoIcons.RightChevron())})))}))
			leading := ts.TopLeft(gluttertest.First(gluttertest.ByType[*widgets.Icon]()))
			trailing := ts.TopLeft(gluttertest.At(gluttertest.ByType[*widgets.Icon](), 1))
			title, subtitle, info := ts.TopLeft(gluttertest.ByText("Title")), ts.TopLeft(gluttertest.ByText("Subtitle")), ts.TopLeft(gluttertest.ByText("Info"))
			// startOf is the edge of a text where it starts to flow.
			startOf := func(text string) float64 {
				f := gluttertest.ByText(text)
				if dir.d == ui.TextDirectionRTL {
					return ts.TopLeft(f).Dx + ts.Size(f).Width
				}
				return ts.TopLeft(f).Dx
			}
			// before is whether a is before b along the direction of the text.
			before := func(a, b ui.Offset) bool {
				if dir.d == ui.TextDirectionRTL {
					return a.Dx > b.Dx
				}
				return a.Dx < b.Dx
			}
			switch {
			case !before(leading, title):
				t.Errorf("the leading widget %v is not before the title %v", leading, title)
			case startOf("Title") != startOf("Subtitle") || title.Dy >= subtitle.Dy:
				t.Errorf("the subtitle %v is not under the title %v", subtitle, title)
			case !before(title, info):
				t.Errorf("the additional info %v is not after the title %v", info, title)
			case !before(info, trailing):
				t.Errorf("the trailing widget %v is not after the additional info %v", trailing, info)
			}
		})
	}
	t.Run("a tile that disappears in its onTap does not throw", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		show := true
		build := func() widgets.Widget {
			var children []widgets.Widget
			if show {
				children = append(children, cupertino.NewCupertinoListTile(widgets.NewText("T"), cupertino.CupertinoListTileOpts{OnTap: func() {
					show = false
				}}))
			}
			return testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(
				widgets.NewColumn(children, widgets.ColumnOpts{MainAxisAlignment: rendering.MainAxisAlignmentCenter})))})
		}
		ts.PumpWidget(build())
		ts.Tap(gluttertest.ByType[*cupertino.CupertinoListTile]())
		if show {
			t.Fatal("the tap did not run")
		}
		ts.PumpWidget(build())
		ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 5 * time.Second})
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
	for _, c := range []struct {
		name     string
		title    string
		subtitle widgets.Widget
	}{{"the title", strings.Repeat("CupertinoListTile", 10), nil}, {"the subtitle", "", widgets.NewText(strings.Repeat("CupertinoListTile", 10))}} {
		t.Run(c.name+" does not overflow", func(t *testing.T) {
			ts := dialogTester(t, ui.Size{})
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(
				cupertino.NewCupertinoListTile(widgets.NewText(c.title), cupertino.CupertinoListTileOpts{Subtitle: c.subtitle}))}))
			if errs := ts.TakeErrors(); len(errs) != 0 {
				t.Errorf("errors: %v", errs)
			}
		})
	}
	t.Run("a long press on a tile with a switch tells it to its widgets and the tile reacts to a tap", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		value := false
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewStatefulBuilder(
			func(bc widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
				return cupertino.NewCupertinoListTile(widgets.NewText(""), cupertino.CupertinoListTileOpts{
					OnTap:   func() { setState(func() { value = !value }) },
					Leading: cupertino.NewCupertinoSwitch(value, func(bool) {}), Trailing: cupertino.NewCupertinoSwitch(value, func(bool) {})})
			}))}))
		ts.LongPress(gluttertest.ByType[*cupertino.CupertinoListTile]())
		ts.PumpAndSettle()
		if !value {
			t.Error("the tile ignored the tap of the long press")
		}
	})
	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewSizedBoxShrink(
			cupertino.NewCupertinoListTile(widgets.NewText("X"), cupertino.CupertinoListTileOpts{Trailing: cupertino.NewCupertinoListTileChevron()})))}))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoListTile]()); got != (ui.Size{}) {
			t.Errorf("the size of the tile is %v", got)
		}
	})
	t.Run("the chevron does not crash at zero area", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewSizedBoxShrink(cupertino.NewCupertinoListTileChevron()))}))
		if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoListTileChevron]()); got != (ui.Size{}) {
			t.Errorf("the size of the chevron is %v", got)
		}
	})
	_ = foundation.DebugMode
}
