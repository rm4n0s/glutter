package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// settingsIcon is the rounded colored square of the leading widget of a settings row.
func settingsIcon(color cupertino.CupertinoDynamicColor, icon widgets.IconData) widgets.Widget {
	return widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(color.Color),
		BorderRadius: painting.NewBorderRadiusCircular(6)}),
		widgets.NewCenter(widgets.NewIcon(icon, widgets.IconOpts{Color: foundation.Some(cupertino.CupertinoColors.White), Size: foundation.Some(18.0)})))
}

// listsPage is the page of the proof of step 34.8: an inset grouped section with notched tiles (a switch, additional info
// and chevrons), an edge-to-edge section, and a form section with rows.
func listsPage() widgets.Widget {
	return widgets.NewColumn([]widgets.Widget{
		cupertino.NewCupertinoListSectionInsetGrouped(cupertino.CupertinoListSectionOpts{
			Header: widgets.NewText("Settings"), Footer: widgets.NewText("Connections are remembered."),
			Children: []widgets.Widget{
				cupertino.NewCupertinoListTileNotched(widgets.NewText("Airplane Mode"), cupertino.CupertinoListTileOpts{
					Leading: settingsIcon(cupertino.CupertinoColors.SystemOrange, cupertino.CupertinoIcons.Airplane()), Trailing: cupertino.NewCupertinoSwitch(true, func(bool) {})}),
				cupertino.NewCupertinoListTileNotched(widgets.NewText("Wi-Fi"), cupertino.CupertinoListTileOpts{
					Leading: settingsIcon(cupertino.CupertinoColors.SystemBlue, cupertino.CupertinoIcons.Wifi()), AdditionalInfo: widgets.NewText("Not Connected"),
					Trailing: cupertino.NewCupertinoListTileChevron(), OnTap: func() {}}),
				cupertino.NewCupertinoListTileNotched(widgets.NewText("Bluetooth"), cupertino.CupertinoListTileOpts{
					Leading: settingsIcon(cupertino.CupertinoColors.SystemBlue, cupertino.CupertinoIcons.Bluetooth()), AdditionalInfo: widgets.NewText("On"),
					Trailing: cupertino.NewCupertinoListTileChevron(), OnTap: func() {}}),
			}}),
		cupertino.NewCupertinoListSection(cupertino.CupertinoListSectionOpts{
			Header: widgets.NewText("EDGE TO EDGE"), NotHasLeading: true,
			Children: []widgets.Widget{
				cupertino.NewCupertinoListTile(widgets.NewText("Title"), cupertino.CupertinoListTileOpts{Subtitle: widgets.NewText("Subtitle"), Trailing: cupertino.NewCupertinoListTileChevron()}),
				cupertino.NewCupertinoListTile(widgets.NewText("Another"), cupertino.CupertinoListTileOpts{AdditionalInfo: widgets.NewText("Info"), Trailing: cupertino.NewCupertinoListTileChevron()}),
			}}),
		cupertino.NewCupertinoFormSectionInsetGrouped([]widgets.Widget{
			cupertino.NewCupertinoFormRow(cupertino.NewCupertinoSwitch(false, func(bool) {}), cupertino.CupertinoFormRowOpts{Prefix: widgets.NewText("Remember"),
				Helper: widgets.NewText("Keeps the choice."), Error: widgets.NewText("Not allowed here.", widgets.TextOpts{Style: painting.NewTextStyle(painting.TextStyleOpts{FontFamily: cupertino.CupertinoSystemTextFamily})})}),
		}, cupertino.CupertinoFormSectionOpts{Header: widgets.NewText("FORM", widgets.TextOpts{Style: painting.NewTextStyle(painting.TextStyleOpts{FontFamily: cupertino.CupertinoSystemTextFamily})})}),
	}, widgets.ColumnOpts{MainAxisSize: 1})
}

// TestCupertinoListsGoldens draws the lists page of step 34.8 in light and dark.
func TestCupertinoListsGoldens(t *testing.T) {
	for _, tt := range []struct {
		name       string
		brightness ui.Brightness
	}{{"cupertino.lists.light", ui.BrightnessLight}, {"cupertino.lists.dark", ui.BrightnessDark}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 400, Height: 700})
			ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
				Theme: cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: tt.brightness}),
				Home:  cupertino.NewCupertinoPageScaffold(widgets.NewSingleChildScrollView(listsPage()))}))
			ts.PumpAndSettle()
			ts.MatchesGolden(tt.name)
		})
	}
}
