package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

var testIcon = widgets.NewIconData(0xE047, widgets.IconDataOpts{FontFamily: "MaterialIcons"})

func TestThemeDataTween(t *testing.T) {
	light, dark := NewThemeData(), NewThemeDataDark()
	tween := NewThemeDataTween(light, dark)
	if !tween.Lerp(0.25).Equal(ThemeDataLerp(light, dark, 0.25)) {
		t.Error("the tween at 0.25 is not the lerp of the themes")
	}
}

func TestThemeBrightnessOf(t *testing.T) {
	dark := widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{PlatformBrightness: ui.BrightnessDark})
	tests := []struct {
		name      string
		w         func(child widgets.Widget) widgets.Widget
		noView    bool
		want      ui.Brightness
		wantFound bool
	}{
		{"just a media query", func(c widgets.Widget) widgets.Widget { return widgets.NewMediaQuery(dark, c) }, false, ui.BrightnessDark, true},
		{"the theme wins over the media query", func(c widgets.Widget) widgets.Widget {
			return widgets.NewMediaQuery(dark, NewTheme(NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessLight}), c))
		}, false, ui.BrightnessLight, true},
		{"neither", func(c widgets.Widget) widgets.Widget { return c }, true, ui.BrightnessLight, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var got, gotMaybe ui.Brightness
			var found bool
			w := tt.w(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				got = ThemeBrightnessOf(bc)
				gotMaybe, found = ThemeMaybeBrightnessOf(bc)
				return widgets.NewSizedBox(nil)
			}))
			if tt.noView {
				// A tree without the implicit View has no MediaQuery either.
				ts.PumpWidget(widgets.NewRawView(ts.View, w), gluttertest.PumpWidgetOpts{NoView: true})
			} else {
				ts.PumpWidget(w)
			}
			if got != tt.want {
				t.Errorf("brightnessOf = %v, want %v", got, tt.want)
			}
			if found != tt.wantFound || (found && gotMaybe != tt.want) {
				t.Errorf("maybeBrightnessOf = %v, %v; want %v, %v", gotMaybe, found, tt.want, tt.wantFound)
			}
		})
	}
}

func capture(ts *gluttertest.Tester, data *ThemeData) *ThemeData {
	var theme *ThemeData
	ts.PumpWidget(NewTheme(data, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		theme = ThemeOf(bc)
		return widgets.NewSizedBox(nil)
	})))
	return theme
}

func TestThemeFallback(t *testing.T) {
	tests := []struct {
		name     string
		data     *ThemeData
		fallback *ThemeData
		geometry *TextTheme
	}{
		{"Material 2", NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false)}), NewThemeDataFallback(false), TypographyEnglishLike2014},
		{"Material 3", NewThemeData(), NewThemeDataFallback(), TypographyEnglishLike2021},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			if got, want := capture(ts, tt.data), ThemeDataLocalize(tt.fallback, tt.geometry); !got.Equal(want) {
				t.Error("Theme.of is not the localized fallback theme")
			}
		})
	}
}

func TestThemeOfWithoutThemeIsTheFallback(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var theme *ThemeData
	ts.PumpWidget(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		theme = ThemeOf(bc)
		return widgets.NewSizedBox(nil)
	}))
	if want := ThemeDataLocalize(NewThemeDataFallback(), TypographyEnglishLike2021); theme == nil || !theme.TextTheme.Equal(want.TextTheme) {
		t.Error("the theme without a Theme has not the text theme of the fallback")
	}
}

func TestThemeDataTypographyDefaults(t *testing.T) {
	if got := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false)}).Typography; !got.Equal(NewTypographyMaterial2014(TypographyOpts{Platform: foundation.Some(foundation.DefaultTargetPlatform())})) {
		t.Error("Material 2 theme has not the typography of 2014")
	}
	m3 := NewThemeData()
	if !m3.Typography.Equal(NewTypographyMaterial2021(Typography2021Opts{
		TypographyOpts: TypographyOpts{Platform: foundation.Some(foundation.DefaultTargetPlatform())}, ColorScheme: m3.ColorScheme})) {
		t.Error("Material 3 theme has not the typography of 2021")
	}
}

func TestIconThemesAreApplied(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	iconStyle := func() *painting.TextStyle {
		span := ts.RenderObject(gluttertest.ByType[*widgets.RichText]()).(*rendering.RenderParagraph).Text().(*painting.TextSpan)
		return span.Style()
	}
	themed := func(color ui.Color, size float64) widgets.Widget {
		return widgets.NewDirectionality(ui.TextDirectionLTR, NewAnimatedTheme(NewThemeData(ThemeDataOpts{IconTheme: foundation.Some(
			widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(color), Size: foundation.Some(size)}))}),
			widgets.NewIcon(testIcon)))
	}
	green, orange := Colors.Green.Primary, Colors.Orange.Primary
	ts.PumpWidget(themed(green, 10))
	if s := iconStyle(); s.Color.Value() != green || s.FontSize.Value() != 10 {
		t.Errorf("first style: color %v, size %v", s.Color, s.FontSize)
	}
	ts.PumpWidget(themed(orange, 20))
	ts.Pump(100 * time.Millisecond) // halfway through the theme transition
	if s := iconStyle(); s.Color.Value() != lerpColor(green, orange, 0.5) || s.FontSize.Value() != 15 {
		t.Errorf("halfway style: color %v, size %v", s.Color, s.FontSize)
	}
	ts.Pump(100 * time.Millisecond) // finish the transition
	if s := iconStyle(); s.Color.Value() != orange || s.FontSize.Value() != 20 {
		t.Errorf("final style: color %v, size %v", s.Color, s.FontSize)
	}
}

// buildCounter is Flutter's Test widget: it counts its builds and depends on the theme.
type buildCounter struct {
	widgets.StatelessWidget
	builds *int
}

func (w *buildCounter) Build(bc widgets.BuildContext) widgets.Widget {
	*w.builds++
	return widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ThemeOf(bc).PrimaryColor)}), nil)
}

func TestSameThemeDataDoesNotRebuildDescendants(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	builds := 0
	counter := &buildCounter{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), &builds}
	pump := func(data *ThemeData) { ts.PumpWidget(NewTheme(data, counter)) }
	data := NewThemeData(ThemeDataOpts{PrimaryColor: foundation.Some(ui.NewColor(0xFF000000))})
	pump(data)
	if builds != 1 {
		t.Fatalf("builds after the first pump = %d, want 1", builds)
	}
	pump(data)
	if builds != 1 {
		t.Errorf("builds after the same theme data = %d, want 1", builds)
	}
	pump(NewThemeData(ThemeDataOpts{PrimaryColor: foundation.Some(ui.NewColor(0xFF000000))}))
	if builds != 1 {
		t.Errorf("builds after an equal theme data = %d, want 1", builds)
	}
	pump(NewThemeData(ThemeDataOpts{PrimaryColor: foundation.Some(ui.NewColor(0xFF222222))}))
	if builds != 2 {
		t.Errorf("builds after a different theme data = %d, want 2", builds)
	}
}

func TestThemeTextGeometryHasPrecedenceOverLocalizations(t *testing.T) {
	const magicFontSize = 4321.0
	fallback := NewThemeDataFallback()
	custom := fallback.CopyWith(ThemeDataCopy{PrimaryTextTheme: foundation.Some(fallback.PrimaryTextTheme.CopyWith(TextThemeOpts{
		BodyMedium: fallback.PrimaryTextTheme.BodyMedium.CopyWith(painting.TextStyleCopy{FontSize: foundation.Some(magicFontSize)}),
	}))})
	if got := custom.PrimaryTextTheme.BodyMedium.FontSize.Value(); got != magicFontSize {
		t.Fatalf("font size of the custom theme = %v", got)
	}
	ts := gluttertest.New(t, gluttertest.Opts{})
	if got := capture(ts, custom).PrimaryTextTheme.BodyMedium.FontSize.Value(); got != magicFontSize {
		t.Errorf("font size from Theme.of = %v, want %v", got, magicFontSize)
	}
}

func TestDefaultThemeProvidesAllBasicTextStyleProperties(t *testing.T) {
	tests := []struct {
		name       string
		data       *ThemeData
		spacing    bool
		height     bool
		decoration bool
		label      string
	}{
		{"Material 2", NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false), Platform: foundation.Some(foundation.TargetPlatformAndroid)}),
			false, false, false, "(englishLike displayLarge 2014).merge(blackMountainView displayLarge)"},
		{"Material 3", NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformAndroid)}),
			true, true, true, "(englishLike displayLarge 2021).merge((blackMountainView displayLarge).apply)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			theme := capture(ts, tt.data)
			for _, textTheme := range []*TextTheme{theme.TextTheme, theme.PrimaryTextTheme} {
				for i, s := range styles(textTheme) {
					if i == 8 || i == 14 { // Flutter's list leaves titleSmall and labelSmall out
						continue
					}
					if s.Inherit {
						t.Errorf("style %q inherits", s.DebugLabel)
					}
					if !s.Color.IsSome() || s.FontFamily == "" || !s.FontSize.IsSome() || s.FontWeight == 0 || s.TextBaseline == ui.TextBaselineUnspecified {
						t.Errorf("style %q misses a basic property: %+v", s.DebugLabel, s)
					}
					if s.FontStyle != ui.FontStyleUnspecified || s.WordSpacing.IsSome() || s.DecorationStyle != ui.TextDecorationStyleUnspecified || s.Locale.IsSome() || s.Background != nil {
						t.Errorf("style %q has a property that should be null", s.DebugLabel)
					}
					if s.LetterSpacing.IsSome() != tt.spacing || s.Height.IsSome() != tt.height || s.DecorationColor.IsSome() != tt.decoration {
						t.Errorf("style %q: letter spacing %v, height %v, decoration color %v", s.DebugLabel, s.LetterSpacing, s.Height, s.DecorationColor)
					}
					if s.Decoration.OrElse(ui.TextDecorationUnderline) != ui.TextDecorationNone {
						t.Errorf("style %q has decoration %v", s.DebugLabel, s.Decoration)
					}
					if foundation.DebugMode && s.DebugLabel == "" {
						t.Error("a style has no debug label")
					}
				}
			}
			if got := theme.TextTheme.DisplayLarge.DebugLabel; foundation.DebugMode && got != tt.label {
				t.Errorf("debug label = %q, want %q", got, tt.label)
			}
		})
	}
}
