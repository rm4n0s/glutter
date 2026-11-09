package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// swatches is the colors of the golden, in the order of CupertinoColors: the accent colors,
// the grays, the labels, the fills and the backgrounds.
func swatches() []ui.ColorValue {
	c := cupertino.CupertinoColors
	return []ui.ColorValue{
		c.SystemBlue, c.SystemGreen, c.SystemMint, c.SystemIndigo, c.SystemOrange, c.SystemPink, c.SystemBrown, c.SystemPurple,
		c.SystemRed, c.SystemTeal, c.SystemCyan, c.SystemYellow, c.SystemGrey, c.SystemGrey2, c.SystemGrey3, c.SystemGrey4,
		c.SystemGrey5, c.SystemGrey6, c.Label, c.SecondaryLabel, c.TertiaryLabel, c.QuaternaryLabel, c.SystemFill, c.SecondarySystemFill,
		c.TertiarySystemFill, c.QuaternarySystemFill, c.SystemBackground, c.SecondarySystemBackground, c.TertiarySystemBackground,
		c.SystemGroupedBackground, c.SecondarySystemGroupedBackground, c.TertiarySystemGroupedBackground, c.Separator, c.OpaqueSeparator,
		c.Link, c.InactiveGray,
	}
}

// colorsPanel is the swatches in one brightness, on the scaffold background of the theme,
// resolved in the context of the panel, under a title in the label color.
func colorsPanel(brightness ui.Brightness, highContrast bool, title string) cupertino.Widget {
	return cupertino.NewMediaQuery(cupertino.NewMediaQueryData(cupertino.MediaQueryDataOpts{HighContrast: highContrast}), cupertino.NewCupertinoTheme(cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: brightness}),
		cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
			theme := cupertino.CupertinoThemeOf(bc)
			label := cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.Label, bc)
			var boxes []cupertino.Widget
			for _, c := range swatches() {
				boxes = append(boxes, cupertino.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
					Color:  foundation.Some(cupertino.CupertinoDynamicColorResolve(c, bc)),
					Border: painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.OpaqueSeparator, bc)), Width: foundation.Some(1.0)}),
				}), cupertino.NewSizedBox(nil, cupertino.SizedBoxOpts{Width: foundation.Some(48.0), Height: foundation.Some(36.0)})))
			}
			return cupertino.NewColoredBox(theme.ScaffoldBackgroundColor().EffectiveColor(), cupertino.NewPadding(painting.NewEdgeInsetsAll(16),
				cupertino.NewColumn([]cupertino.Widget{
					cupertino.NewText(title, cupertino.TextOpts{Style: painting.NewTextStyle(painting.TextStyleOpts{
						FontFamily: "Roboto", FontSize: foundation.Some(20.0), Color: foundation.Some(label), FontWeight: ui.FontWeightW600})}),
					cupertino.NewSizedBox(nil, cupertino.SizedBoxOpts{Height: foundation.Some(12.0)}),
					cupertino.NewWrap(boxes, cupertino.WrapOpts{Spacing: 8, RunSpacing: 8}),
					cupertino.NewSizedBox(nil, cupertino.SizedBoxOpts{Height: foundation.Some(16.0)}),
					iconRow(),
				}, cupertino.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})))
		})))
}

// iconRow is a row of Cupertino icons in the accent colors of the theme: the icon theme of the
// theme gives the first one its primary color.
func iconRow() cupertino.Widget {
	c := cupertino.CupertinoColors
	icons := []struct {
		icon  cupertino.IconData
		color ui.ColorValue
	}{
		{cupertino.CupertinoIcons.HeartFill(), c.SystemPink}, {cupertino.CupertinoIcons.BellFill(), c.SystemGreen},
		{cupertino.CupertinoIcons.UmbrellaFill(), c.SystemBlue}, {cupertino.CupertinoIcons.Gear(), nil}, {cupertino.CupertinoIcons.Search(), nil},
	}
	var children []cupertino.Widget
	for _, i := range icons {
		opts := cupertino.IconOpts{Size: foundation.Some(30.0)}
		if i.color != nil {
			opts.Color = foundation.Some(i.color.EffectiveColor())
		}
		children = append(children, cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
			o := opts
			if i.color != nil {
				o.Color = foundation.Some(cupertino.CupertinoDynamicColorResolve(i.color, bc))
			}
			return cupertino.NewIcon(i.icon, o)
		}))
	}
	return cupertino.NewRow(children, cupertino.RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween})
}

// TestCupertinoColorsGolden draws the system colors in the light and in the dark brightness
// and in the light brightness with high contrast, with a row of icons under each (proof 34.1-cupertino-colors.png).
func TestCupertinoColorsGolden(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(cupertino.NewDirectionality(ui.TextDirectionLTR, cupertino.NewRow([]cupertino.Widget{
		cupertino.NewExpanded(colorsPanel(ui.BrightnessLight, false, "light")),
		cupertino.NewExpanded(colorsPanel(ui.BrightnessDark, false, "dark")),
		cupertino.NewExpanded(colorsPanel(ui.BrightnessLight, true, "light, high contrast")),
	}, cupertino.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})))
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("reported %v", errs)
	}
	ts.MatchesGolden("cupertino.colors")
}
