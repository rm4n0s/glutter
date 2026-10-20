package widgets_test

import (
	"image/color"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

func roboto(size float64, c uint32, weight ui.FontWeight) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto", FontSize: foundation.Some(size), Color: foundation.Some(ui.NewColor(c)), FontWeight: weight})
}

// TestBasicWidgetsGolden2 draws the text, icon, image and gesture widgets of
// step 7.7: styled and aligned text, a default text style, a rich text with an
// inline widget, an ellipsis, icons in a theme, and images of a memory provider.
func TestBasicWidgetsGolden2(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 300}})
	swatch := func(w, h int, c color.RGBA) Widget {
		return NewImageMemory(pngBytes(t, w, h, c), ImageMemoryOpts{ImageOpts: ImageOpts{Width: foundation.Some(float64(w)), Height: foundation.Some(float64(h)), Fit: painting.BoxFitFill}})
	}
	addIcon := NewIconData(0xE047, IconDataOpts{FontFamily: "MaterialIcons"})
	scene := NewDirectionality(ui.TextDirectionLTR, NewStack([]Widget{
		NewPositionedFill(NewColoredBox(ui.NewColor(0xFFFFFBFE), nil)),
		at(10, 10, NewText("Glutter text", TextOpts{Style: roboto(28, 0xFF6750A4, ui.FontWeightW700)})),
		at(10, 50, NewDefaultTextStyle(roboto(16, 0xFF1C1B1F, ui.FontWeightW400), NewColumn([]Widget{
			NewText("Inherited style"),
			NewText("Own style wins", TextOpts{Style: roboto(12, 0xFFB3261E, ui.FontWeightW400)}),
		}, ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart}))),
		at(10, 110, NewSizedBox(NewText("A long line of text that does not fit in its box and ends with an ellipsis", TextOpts{
			Style: roboto(14, 0xFF1C1B1F, ui.FontWeightW400), MaxLines: foundation.Some(2), Overflow: painting.TextOverflowEllipsis}),
			SizedBoxOpts{Width: foundation.Some(140.0)})),
		at(200, 10, NewSizedBox(NewText("Centered text in a box", TextOpts{Style: roboto(14, 0xFF1C1B1F, ui.FontWeightW400), TextAlign: ui.TextAlignCenter}),
			SizedBoxOpts{Width: foundation.Some(180.0)})),
		at(200, 60, NewRichText(painting.NewTextSpan(painting.TextSpanOpts{Style: roboto(16, 0xFF1C1B1F, ui.FontWeightW400), Text: foundation.Some("Rich "),
			Children: []painting.InlineSpan{
				painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some("red "), Style: roboto(16, 0xFFB3261E, ui.FontWeightW700)}),
				NewWidgetSpan(swatch(16, 16, color.RGBA{R: 0x38, G: 0x6A, B: 0x20, A: 255}), WidgetSpanOpts{Alignment: ui.PlaceholderAlignmentMiddle}),
				painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(" inline")}),
			}}))),
		at(10, 170, NewIconTheme(NewIconThemeData(IconThemeDataOpts{Size: foundation.Some(36.0), Color: foundation.Some(ui.NewColor(0xFF6750A4))}), NewRow([]Widget{
			NewIcon(addIcon), NewIcon(addIcon, IconOpts{Color: foundation.Some(ui.NewColor(0xFFB3261E)), Size: foundation.Some(24.0)}),
			NewIcon(addIcon, IconOpts{Size: foundation.Some(48.0)}),
		}))),
		at(10, 230, swatch(40, 40, color.RGBA{R: 0x67, G: 0x50, B: 0xA4, A: 255})),
		at(60, 230, NewImageMemory(pngBytes(t, 4, 4, color.RGBA{R: 0xB3, G: 0x26, B: 0x1E, A: 255}), ImageMemoryOpts{ImageOpts: ImageOpts{
			Width: foundation.Some(40.0), Height: foundation.Some(40.0), Fit: painting.BoxFitFill, Color: foundation.Some(ui.NewColor(0xFF0061A4)), ColorBlendMode: ui.BlendModeModulate}})),
		at(110, 230, NewImageIcon(painting.NewMemoryImage(pngBytes(t, 64, 64, color.RGBA{A: 255})), ImageIconOpts{Size: foundation.Some(40.0), Color: foundation.Some(ui.NewColor(0xFF386A20))})),
	}))
	ts.PumpWidget(scene)
	waitFor(t, ts, func() bool {
		ts.Pump()
		for _, e := range ts.Elements(gluttertest.ByType[*RawImage]()) {
			if e.Widget().(*RawImage).Image() == nil {
				return false
			}
		}
		return true
	})
	ts.MatchesGolden("basic_widgets_part2")
}
