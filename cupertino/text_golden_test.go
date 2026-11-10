package cupertino_test

import (
	"testing"

	"golang.org/x/image/font/gofont/gomono"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// systemFonts gives the family "sans-serif" to the text of the toolbars, whose style has no family: the system font
// of the tests is not there, so they would draw no glyph (the Material tests do the same).
func systemFonts(size ui.Size) gluttertest.Opts {
	return gluttertest.Opts{Size: size, Fonts: []gluttertest.Font{{Family: "sans-serif", Data: gomono.TTF}}}
}

// darkTheme is a dark theme whose text is white (the text of testTheme is black).
func darkTheme() *cupertino.CupertinoThemeData {
	white := ui.NewColor(0xFFFFFFFF)
	return testTheme(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessDark, TextTheme: cupertino.NewCupertinoTextThemeData(cupertino.CupertinoTextThemeDataOpts{
		TextStyle: &painting.TextStyle{FontFamily: "Roboto", FontSize: foundation.Some(17.0), Color: foundation.Some(white)}})})
}

// TestCupertinoTextFieldGoldens draws the text fields of step 34.5 in light and dark: the rounded and the borderless
// one, with a placeholder, a prefix, the clear button, a suffix, a disabled one, and the search field, empty and with
// text.
func TestCupertinoTextFieldGoldens(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformIOS)
	page := func() widgets.Widget {
		pad := func(w widgets.Widget) widgets.Widget { return widgets.NewPadding(painting.NewEdgeInsetsAll(6), w) }
		text := func(s string) *widgets.TextEditingController { return widgets.NewTextEditingController(s) }
		return widgets.NewColumn([]widgets.Widget{
			pad(cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Placeholder: foundation.Some("Placeholder")})),
			pad(cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: text("Some text"), ClearButtonMode: cupertino.OverlayVisibilityModeEditing,
				Prefix: widgets.NewPadding(painting.NewEdgeInsetsAll(6), widgets.NewIcon(cupertino.CupertinoIcons.PersonCropCircleFill()))})),
			pad(cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: text("secret"), ObscureText: true,
				Suffix: widgets.NewPadding(painting.NewEdgeInsetsAll(6), widgets.NewIcon(cupertino.CupertinoIcons.LockFill()))})),
			pad(cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Placeholder: foundation.Some("Disabled"), Enabled: foundation.Some(false)})),
			pad(cupertino.NewCupertinoTextFieldBorderless(cupertino.CupertinoTextFieldOpts{Placeholder: foundation.Some("Borderless")})),
			pad(cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: text("one\ntwo\nthree"), MaxLines: foundation.Some(3), MinLines: foundation.Some(3)})),
			pad(cupertino.NewCupertinoSearchTextField()),
			pad(cupertino.NewCupertinoSearchTextField(cupertino.CupertinoSearchTextFieldOpts{Controller: text("query")})),
		}, widgets.ColumnOpts{MainAxisSize: 1})
	}
	for _, tt := range []struct {
		name       string
		brightness ui.Brightness
	}{{"cupertino.textfields.light", ui.BrightnessLight}, {"cupertino.textfields.dark", ui.BrightnessDark}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 360, Height: 460}})
			theme := testTheme(cupertino.CupertinoThemeDataOpts{Brightness: tt.brightness})
			if tt.brightness == ui.BrightnessDark {
				theme = darkTheme()
			}
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Theme: theme, Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(page()))}))
			ts.Pump()
			ts.MatchesGolden(tt.name)
		})
	}
}

// TestCupertinoTextSelectionGoldens draws a field with a selection, its handles and the toolbar on iOS, the desktop
// toolbar of macOS, and the magnifier of iOS.
func TestCupertinoTextSelectionGoldens(t *testing.T) {
	t.Run("the selection with the handles and the toolbar on iOS", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformIOS)
		ts := gluttertest.New(t, systemFonts(ui.Size{Width: 360, Height: 200}))
		controller := widgets.NewTextEditingController("Select this word now")
		defer controller.Dispose()
		ts.SetClipboardText("pasted")
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewAlign(
			widgets.NewPadding(painting.NewEdgeInsetsAll(20), cupertino.NewCupertinoTextField(cupertino.CupertinoTextFieldOpts{Controller: controller})),
			widgets.AlignOpts{Alignment: painting.AlignmentCenter}))}))
		ts.LongPressAt(caretPosition(ts, 8))
		ts.PumpAndSettle()
		ts.MatchesGolden("cupertino.textselection.ios")
	})
	t.Run("the desktop toolbar on macOS", func(t *testing.T) {
		usePlatform(t, foundation.TargetPlatformMacOS)
		ts := gluttertest.New(t, systemFonts(ui.Size{Width: 360, Height: 240}))
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewStack([]widgets.Widget{
			widgets.NewColoredBox(ui.NewColor(0xFFE5E5EA), widgets.NewSizedBoxExpand(nil)),
			cupertino.NewCupertinoDesktopTextSelectionToolbar(ui.Offset{Dx: 40, Dy: 30}, []widgets.Widget{
				cupertino.NewCupertinoDesktopTextSelectionToolbarButtonText(func() {}, "Cut"),
				cupertino.NewCupertinoDesktopTextSelectionToolbarButtonText(func() {}, "Copy"),
				cupertino.NewCupertinoDesktopTextSelectionToolbarButtonText(func() {}, "Paste"),
				cupertino.NewCupertinoDesktopTextSelectionToolbarButtonText(nil, "Select All"),
			}),
		}))}))
		ts.Pump()
		ts.MatchesGolden("cupertino.textselection.desktop")
	})
	t.Run("the toolbar of iOS in light and dark", func(t *testing.T) {
		for _, b := range []ui.Brightness{ui.BrightnessLight, ui.BrightnessDark} {
			name := "cupertino.textselection.toolbar.light"
			if b == ui.BrightnessDark {
				name = "cupertino.textselection.toolbar.dark"
			}
			ts := gluttertest.New(t, systemFonts(ui.Size{Width: 360, Height: 160}))
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Theme: testTheme(cupertino.CupertinoThemeDataOpts{Brightness: b}),
				Home: cupertino.NewCupertinoPageScaffold(cupertino.NewCupertinoTextSelectionToolbar(ui.Offset{Dx: 180, Dy: 100}, ui.Offset{Dx: 180, Dy: 130}, []widgets.Widget{
					cupertino.NewCupertinoTextSelectionToolbarButtonText("Cut", cupertino.CupertinoTextSelectionToolbarButtonOpts{OnPressed: func() {}}),
					cupertino.NewCupertinoTextSelectionToolbarButtonText("Copy", cupertino.CupertinoTextSelectionToolbarButtonOpts{OnPressed: func() {}}),
					cupertino.NewCupertinoTextSelectionToolbarButtonText("Paste", cupertino.CupertinoTextSelectionToolbarButtonOpts{OnPressed: func() {}}),
					cupertino.NewCupertinoTextSelectionToolbarButtonText("Select All"),
				}))}))
			ts.Pump()
			ts.MatchesGolden(name)
		}
	})
}

// TestCupertinoMagnifierGolden draws the magnifier of iOS over a red square, which it shows twice: the original and the
// one in the magnifier.
func TestCupertinoMagnifierGolden(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
	controller := widgets.NewMagnifierController()
	defer controller.RemoveFromOverlay()
	var bc widgets.BuildContext
	square := foundation.NewUniqueKey()
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Theme: testTheme(cupertino.CupertinoThemeDataOpts{PrimaryColor: ui.NewColor(0xFF6750A4)}),
		Home: widgets.NewColoredBox(ui.NewColor(0xFF00FFB3), widgets.NewCenter(widgets.NewBuilder(func(b widgets.BuildContext) widgets.Widget {
			bc = b
			return widgets.NewColoredBox(ui.NewColor(0xFFF44336), widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: square, Width: foundation.Some(10.0), Height: foundation.Some(10.0)}))
		})))}))
	rect := ts.TopLeft(gluttertest.ByKey(square))
	field := ui.NewRectFromLTWH(rect.Dx, rect.Dy, 10, 10)
	info := foundation.NewValueNotifier(widgets.MagnifierInfo{CurrentLineBoundaries: field, FieldBounds: field, CaretRect: field, GlobalGesturePosition: field.Center()})
	defer info.Dispose()
	controller.Show(bc, func(widgets.BuildContext) widgets.Widget {
		return cupertino.NewCupertinoTextMagnifier(controller, info)
	})
	ts.PumpAndSettle()
	ts.MatchesGolden("cupertino.magnifier")
}
