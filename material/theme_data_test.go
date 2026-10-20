package material_test

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestThemeDataControl(t *testing.T) {
	dark := NewThemeDataDark()
	if !dark.Equal(dark.CopyWith(ThemeDataCopy{})) {
		t.Error("copyWith with no arguments changed the theme")
	}
	if got := dark.String(); !strings.HasPrefix(got, "ThemeData") {
		t.Errorf("description = %q", got)
	}
	light := NewThemeData()
	dawn := ThemeDataLerp(dark, light, 0.25)
	if got := dawn.Brightness(); got != ui.BrightnessDark {
		t.Errorf("brightness = %v, want dark", got)
	}
	if got, want := dawn.PrimaryColor, lerpColor(dark.PrimaryColor, light.PrimaryColor, 0.25); got != want {
		t.Errorf("primary color = %v, want %v", got, want)
	}
	if ThemeDataLerp(dark, dark, 0.5) != dark {
		t.Error("lerp of identical themes is not the theme itself")
	}
}

func lerpColor(a, b ui.Color, t float64) ui.Color { return *ui.ColorLerp(&a, &b, t) }

func TestThemeDataDefaultsToTheTypographyOfThePlatform(t *testing.T) {
	for _, platform := range allPlatforms {
		t.Run(platform.Name(), func(t *testing.T) {
			theme := NewThemeData(ThemeDataOpts{Platform: foundation.Some(platform), UseMaterial3: foundation.Some(false)})
			typography := NewTypographyMaterial2018(TypographyOpts{Platform: foundation.Some(platform)})
			want := typography.Black.Apply(TextThemeApply{Decoration: foundation.Some(ui.TextDecorationNone)})
			if !theme.TextTheme.Equal(want) {
				t.Error("the text theme is not the default typography of the platform")
			}
		})
	}
}

func TestThemeDataThemesContrastWithBrightness(t *testing.T) {
	m2 := foundation.Some(false)
	light := NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessLight, UseMaterial3: m2})
	dark := NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessDark, UseMaterial3: m2})
	typography := NewTypographyMaterial2018(TypographyOpts{Platform: foundation.Some(light.Platform)})
	lightPrimary := NewThemeData(ThemeDataOpts{PrimaryColor: foundation.Some(Colors.White), UseMaterial3: m2})
	darkPrimary := NewThemeData(ThemeDataOpts{PrimaryColor: foundation.Some(Colors.Black), UseMaterial3: m2})
	tests := []struct {
		name      string
		got, want ui.Color
	}{
		{"text on light", light.TextTheme.TitleLarge.Color.Value(), typography.Black.TitleLarge.Color.Value()},
		{"text on dark", dark.TextTheme.TitleLarge.Color.Value(), typography.White.TitleLarge.Color.Value()},
		{"primary text on a light primary", lightPrimary.PrimaryTextTheme.TitleLarge.Color.Value(), typography.Black.TitleLarge.Color.Value()},
		{"primary text on a dark primary", darkPrimary.PrimaryTextTheme.TitleLarge.Color.Value(), typography.White.TitleLarge.Color.Value()},
		{"icons on light", light.IconTheme.Color().Value(), DefaultIconDarkColor},
		{"icons on dark", dark.IconTheme.Color().Value(), DefaultIconLightColor},
		{"primary icons on a light primary", lightPrimary.PrimaryIconTheme.Color().Value(), Colors.Black},
		{"primary icons on a dark primary", darkPrimary.PrimaryIconTheme.Color().Value(), Colors.White},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("color = %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestThemeDataConstructorsSupportMaterial3(t *testing.T) {
	for name, theme := range map[string]*ThemeData{"default": NewThemeData(), "dark": NewThemeDataDark(), "fallback": NewThemeDataFallback()} {
		t.Run(name, func(t *testing.T) {
			if !theme.UseMaterial3 {
				t.Error("useMaterial3 is false")
			}
			want := NewTypographyMaterial2021(Typography2021Opts{
				TypographyOpts: TypographyOpts{Platform: foundation.Some(theme.Platform)}, ColorScheme: theme.ColorScheme})
			if !theme.Typography.Equal(want) {
				t.Error("the typography is not Material 2021 with the color scheme of the theme")
			}
		})
	}
	if NewThemeDataLight(false).UseMaterial3 || NewThemeDataDark(false).UseMaterial3 || NewThemeDataFallback(false).UseMaterial3 {
		t.Error("useMaterial3 false was ignored")
	}
}

func TestThemeDataMaterialTapTargetSizeDefault(t *testing.T) {
	for _, platform := range allPlatforms {
		t.Run(platform.Name(), func(t *testing.T) {
			theme := NewThemeData(ThemeDataOpts{Platform: foundation.Some(platform)})
			want := MaterialTapTargetSizeShrinkWrap
			switch platform {
			case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformIOS:
				want = MaterialTapTargetSizePadded
			}
			if theme.MaterialTapTargetSize != want {
				t.Errorf("materialTapTargetSize = %v, want %v", theme.MaterialTapTargetSize, want)
			}
		})
	}
}

func TestThemeDataFontFamilyDefault(t *testing.T) {
	theme := NewThemeData(ThemeDataOpts{
		FontFamily: "FlutterTest",
		TextTheme:  NewTextTheme(TextThemeOpts{TitleLarge: painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto"})}),
	})
	if got := theme.TextTheme.BodyLarge.FontFamily; got != "FlutterTest" {
		t.Errorf("bodyLarge font family = %q", got)
	}
	if got := theme.PrimaryTextTheme.DisplaySmall.FontFamily; got != "FlutterTest" {
		t.Errorf("primary displaySmall font family = %q", got)
	}
	if got := theme.TextTheme.TitleLarge.FontFamily; got != "Roboto" {
		t.Errorf("titleLarge font family = %q, want the one that was given", got)
	}
}

func TestEstimateBrightnessForColor(t *testing.T) {
	tests := []struct {
		name  string
		color ui.Color
		want  ui.Brightness
	}{
		{"white", Colors.White, ui.BrightnessLight},
		{"black", Colors.Black, ui.BrightnessDark},
		{"blue", Colors.Blue.Primary, ui.BrightnessDark},
		{"yellow", Colors.Yellow.Primary, ui.BrightnessLight},
		{"deepOrange", Colors.DeepOrange.Primary, ui.BrightnessDark},
		{"orange", Colors.Orange.Primary, ui.BrightnessLight},
		{"lime", Colors.Lime.Primary, ui.BrightnessLight},
		{"grey", Colors.Grey.Primary, ui.BrightnessLight},
		{"teal", Colors.Teal.Primary, ui.BrightnessDark},
		{"indigo", Colors.Indigo.Primary, ui.BrightnessDark},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EstimateBrightnessForColor(tt.color); got != tt.want {
				t.Errorf("brightness = %v, want %v", got, tt.want)
			}
		})
	}
}

// recoverRoute runs fn, which must panic with an *errors.Error, and returns the route of the error.
func recoverRoute(t *testing.T, fn func()) (route string) {
	t.Helper()
	defer func() {
		appErr, ok := errors.FromError(asError(recover()))
		if !ok {
			t.Fatal("expected a panic with *errors.Error")
		}
		route = appErr.Route()
	}()
	fn()
	return ""
}

func TestThemeDataInvalidCombinations(t *testing.T) {
	skipUnlessDebug(t)
	seed := foundation.Some(Colors.Blue.Primary)
	light := NewColorSchemeLight()
	green := Colors.Green
	tests := []struct {
		name string
		opts ThemeDataOpts
		tag  string
	}{
		{"seed with colorScheme", ThemeDataOpts{ColorSchemeSeed: seed, ColorScheme: light}, "ThemeDataSeedConflict"},
		{"seed with primaryColor", ThemeDataOpts{ColorSchemeSeed: seed, PrimaryColor: foundation.Some(Colors.Green.Primary)}, "ThemeDataSeedConflict"},
		{"seed with primarySwatch", ThemeDataOpts{ColorSchemeSeed: seed, PrimarySwatch: &green}, "ThemeDataSeedConflict"},
		{"brightness not matching the color scheme", ThemeDataOpts{ColorScheme: light, Brightness: ui.BrightnessDark}, "ThemeDataBrightnessMismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if route := recoverRoute(t, func() { NewThemeData(tt.opts) }); !strings.HasSuffix(route, tt.tag) {
				t.Errorf("failure path = %q, want it to end in %s", route, tt.tag)
			}
		})
	}
}

func TestThemeDataColorSchemes(t *testing.T) {
	tests := []struct {
		name  string
		theme *ThemeData
		want  []colorWant
	}{
		{"light from the seed", NewThemeData(ThemeDataOpts{ColorSchemeSeed: foundation.Some(Colors.Blue.Primary)}), seedLightWant},
		{"dark from the seed", NewThemeData(ThemeDataOpts{ColorSchemeSeed: foundation.Some(Colors.Blue.Primary), Brightness: ui.BrightnessDark}), seedDarkWant},
		{"default M3 light", NewThemeData(), m3LightWant},
		{"M3 light", NewThemeDataLight(), m3LightWant},
		{"M3 dark", NewThemeDataDark(), m3DarkWant},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { checkColors(t, tt.theme.ColorScheme, tt.want) })
	}
	theme := NewThemeData()
	scheme := theme.ColorScheme
	m3 := []struct {
		name      string
		got, want ui.Color
	}{
		{"primaryColor", theme.PrimaryColor, scheme.Primary()},
		{"canvasColor", theme.CanvasColor, scheme.Surface()},
		{"scaffoldBackgroundColor", theme.ScaffoldBackgroundColor, scheme.Surface()},
		{"cardColor", theme.CardColor, scheme.Surface()},
		{"dividerColor", theme.DividerColor, scheme.Outline()},
		{"dialogBackgroundColor", theme.DialogBackgroundColor, scheme.Surface()},
		{"indicatorColor", theme.IndicatorColor, scheme.OnPrimary()},
	}
	for _, c := range m3 {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if theme.ApplyElevationOverlayColor {
		t.Error("applyElevationOverlayColor is true in a light theme")
	}
}

func TestThemeDataFrom(t *testing.T) {
	light, dark := NewColorSchemeLight(), NewColorSchemeDark()
	tests := []struct {
		name        string
		scheme      *ColorScheme
		brightness  ui.Brightness
		primary     ui.Color
		overlayDark bool
	}{
		{"light", light, ui.BrightnessLight, light.Primary(), false},
		// In a dark theme the color used for main components is surface, not primary.
		{"dark", dark, ui.BrightnessDark, dark.Surface(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := NewThemeDataFrom(tt.scheme, nil, foundation.None[bool]())
			if theme.Brightness() != tt.brightness || theme.PrimaryColor != tt.primary {
				t.Errorf("brightness %v, primary %v", theme.Brightness(), theme.PrimaryColor)
			}
			for name, got := range map[string]ui.Color{"card": theme.CardColor, "canvas": theme.CanvasColor,
				"scaffold": theme.ScaffoldBackgroundColor, "dialog": theme.DialogBackgroundColor} {
				if got != tt.scheme.Surface() {
					t.Errorf("%s color = %v, want the surface", name, got)
				}
			}
			if theme.ApplyElevationOverlayColor != tt.overlayDark {
				t.Errorf("applyElevationOverlayColor = %v", theme.ApplyElevationOverlayColor)
			}
		})
	}
}

func TestVisualDensityDefaults(t *testing.T) {
	for _, platform := range allPlatforms {
		t.Run(platform.Name(), func(t *testing.T) {
			want := VisualDensityCompact()
			switch platform {
			case foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia:
				want = VisualDensityStandard()
			}
			if got := VisualDensityDefaultForPlatform(platform); got != want {
				t.Errorf("density for the platform = %v, want %v", got, want)
			}
			if got := NewThemeData(ThemeDataOpts{Platform: foundation.Some(platform)}).VisualDensity; got != want {
				t.Errorf("density of the theme = %v, want %v", got, want)
			}
		})
	}
	// The platform of the target is linux in tests.
	if got := VisualDensityAdaptivePlatformDensity(); got != VisualDensityCompact() {
		t.Errorf("adaptive density = %v", got)
	}
	if got := NewThemeData().VisualDensity; got != VisualDensityCompact() {
		t.Errorf("default density of the theme = %v", got)
	}
}

func TestVisualDensityEffectiveConstraints(t *testing.T) {
	density := func(v float64) VisualDensity {
		return NewVisualDensity(VisualDensityOpts{Horizontal: foundation.Some(v), Vertical: foundation.Some(v)})
	}
	tests := []struct {
		name                   string
		density                VisualDensity
		constraints            rendering.BoxConstraints
		minW, minH, maxW, maxH float64
	}{
		{"clamped expand", density(4), rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(35.0), Height: foundation.Some(35.0)}), 35, 35, 35, 35},
		{"clamped contract", density(-4), rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(35.0), Height: foundation.Some(35.0)}), 19, 19, 35, 35},
		{"small expand", density(4), rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(4.0), Height: foundation.Some(4.0)}), 4, 4, 4, 4},
		{"small contract", density(-4), rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(4.0), Height: foundation.Some(4.0)}), 0, 0, 4, 4},
		{"loose expand", density(4), rendering.NewBoxConstraints(), 16, 16, math.Inf(1), math.Inf(1)},
		{"loose contract", density(-4), rendering.NewBoxConstraints(), 0, 0, math.Inf(1), math.Inf(1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.density.EffectiveConstraints(tt.constraints)
			if got.MinWidth != tt.minW || got.MinHeight != tt.minH || got.MaxWidth != tt.maxW || got.MaxHeight != tt.maxH {
				t.Errorf("constraints = %v", got)
			}
		})
	}
}

func TestVisualDensityOutOfRange(t *testing.T) {
	skipUnlessDebug(t)
	route := recoverRoute(t, func() { NewVisualDensity(VisualDensityOpts{Horizontal: foundation.Some(4.5)}) })
	if !strings.HasSuffix(route, "VisualDensityOutOfRange") {
		t.Errorf("failure path = %q", route)
	}
}

type extensionA struct{ color1, color2 foundation.Opt[ui.Color] }

func (e *extensionA) CopyWith() ThemeExtension { c := *e; return &c }
func (e *extensionA) Lerp(other ThemeExtension, t float64) ThemeExtension {
	o, ok := other.(*extensionA)
	if !ok {
		return e
	}
	return &extensionA{lerpOpt(e.color1, o.color1, t), lerpOpt(e.color2, o.color2, t)}
}

func lerpOpt(a, b foundation.Opt[ui.Color], t float64) foundation.Opt[ui.Color] {
	return foundation.Some(lerpColor(a.Value(), b.Value(), t))
}

type extensionB struct{ textStyle *painting.TextStyle }

func (e *extensionB) CopyWith() ThemeExtension { c := *e; return &c }
func (e *extensionB) Lerp(other ThemeExtension, t float64) ThemeExtension {
	o, ok := other.(*extensionB)
	if !ok {
		return e
	}
	return &extensionB{painting.TextStyleLerp(e.textStyle, o.textStyle, t)}
}

func fontSize(size float64) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(size)})
}

func TestThemeExtensions(t *testing.T) {
	a1 := &extensionA{foundation.Some(Colors.Black), foundation.Some(Colors.Amber.Primary)}
	a2 := &extensionA{foundation.Some(Colors.White), foundation.Some(Colors.Blue.Primary)}
	b1, b2 := &extensionB{fontSize(50)}, &extensionB{fontSize(100)}

	t.Run("can be obtained", func(t *testing.T) {
		theme := NewThemeData(ThemeDataOpts{Extensions: []ThemeExtension{a1, b1}})
		gotA, ok := ThemeDataExtension[*extensionA](theme)
		if !ok || gotA.color1.Value() != Colors.Black || gotA.color2.Value() != Colors.Amber.Primary {
			t.Errorf("extension A = %+v, %v", gotA, ok)
		}
		gotB, ok := ThemeDataExtension[*extensionB](theme)
		if !ok || !gotB.textStyle.Equal(fontSize(50)) {
			t.Errorf("extension B = %+v, %v", gotB, ok)
		}
	})
	t.Run("can use copyWith", func(t *testing.T) {
		copied := a1.CopyWith().(*extensionA)
		copied.color1 = foundation.Some(Colors.Blue.Primary)
		theme := NewThemeData(ThemeDataOpts{Extensions: []ThemeExtension{copied}})
		got, _ := ThemeDataExtension[*extensionA](theme)
		if got.color1.Value() != Colors.Blue.Primary || got.color2.Value() != Colors.Amber.Primary {
			t.Errorf("extension A = %+v", got)
		}
		if a1.color1.Value() != Colors.Black {
			t.Error("copyWith changed the original")
		}
	})
	t.Run("can lerp", func(t *testing.T) {
		lerp := func(a, b []ThemeExtension) *ThemeData {
			return ThemeDataLerp(NewThemeData(ThemeDataOpts{Extensions: a}), NewThemeData(ThemeDataOpts{Extensions: b}), 0.5)
		}
		both := lerp([]ThemeExtension{a1, b1}, []ThemeExtension{a2, b2})
		gotA, _ := ThemeDataExtension[*extensionA](both)
		gotB, _ := ThemeDataExtension[*extensionB](both)
		if gotA.color1.Value().ToARGB32() != 0xff7f7f7f && gotA.color1.Value().ToARGB32() != 0xff808080 {
			t.Errorf("lerped color1 = %v", gotA.color1.Value())
		}
		if got := gotA.color2.Value().ToARGB32(); got != 0xff90ab7d && got != 0xff90ac7d {
			t.Errorf("lerped color2 = %08x", got)
		}
		if !gotB.textStyle.Equal(fontSize(75)) {
			t.Errorf("lerped text style = %v", gotB.textStyle)
		}

		missingSecond := lerp([]ThemeExtension{a1, b1}, []ThemeExtension{b2})
		gotA, _ = ThemeDataExtension[*extensionA](missingSecond)
		gotB, _ = ThemeDataExtension[*extensionB](missingSecond)
		if gotA.color1.Value() != Colors.Black || gotA.color2.Value() != Colors.Amber.Primary {
			t.Errorf("extension missing from the second theme was lerped: %+v", gotA)
		}
		if !gotB.textStyle.Equal(fontSize(75)) {
			t.Errorf("lerped text style = %v", gotB.textStyle)
		}

		missingFirst := lerp([]ThemeExtension{a1}, []ThemeExtension{a2, b2})
		gotB, _ = ThemeDataExtension[*extensionB](missingFirst)
		if !gotB.textStyle.Equal(fontSize(100)) {
			t.Errorf("extension missing from the first theme was lerped: %v", gotB.textStyle)
		}
	})
	t.Run("not found", func(t *testing.T) {
		if got, ok := ThemeDataExtension[*extensionA](NewThemeData(ThemeDataOpts{Extensions: []ThemeExtension{}})); ok || got != nil {
			t.Errorf("found %v", got)
		}
	})
}

func TestThemeDataEquality(t *testing.T) {
	if !NewThemeData().Equal(NewThemeData().CopyWith(ThemeDataCopy{})) {
		t.Error("copyWith with no arguments changed the theme")
	}
	// Regression test for https://github.com/flutter/flutter/issues/91587.
	tests := []struct {
		name string
		a, b ThemeDataOpts
	}{
		{"focusColor", ThemeDataOpts{FocusColor: foundation.Some(Colors.Black)}, ThemeDataOpts{FocusColor: foundation.Some(Colors.White)}},
		{"hoverColor", ThemeDataOpts{HoverColor: foundation.Some(Colors.Black)}, ThemeDataOpts{HoverColor: foundation.Some(Colors.White)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if NewThemeData(tt.a).Equal(NewThemeData(tt.b)) {
				t.Error("themes with different colors are equal")
			}
		})
	}
}

func TestThemeDataCopyWithCopiesAllArguments(t *testing.T) {
	other := NewThemeData(ThemeDataOpts{
		ApplyElevationOverlayColor: foundation.Some(true), MaterialTapTargetSize: MaterialTapTargetSizePadded,
		Platform: foundation.Some(foundation.TargetPlatformAndroid), UseMaterial3: foundation.Some(false),
		VisualDensity: foundation.Some(VisualDensityComfortable()),
		ColorScheme:   NewColorSchemeDark(),
		CanvasColor:   foundation.Some(ui.NewColor(1)), CardColor: foundation.Some(ui.NewColor(2)),
		DisabledColor: foundation.Some(ui.NewColor(3)), DividerColor: foundation.Some(ui.NewColor(4)),
		FocusColor: foundation.Some(ui.NewColor(5)), HighlightColor: foundation.Some(ui.NewColor(6)),
		HintColor: foundation.Some(ui.NewColor(7)), HoverColor: foundation.Some(ui.NewColor(8)),
		PrimaryColor: foundation.Some(ui.NewColor(9)), PrimaryColorDark: foundation.Some(ui.NewColor(10)),
		PrimaryColorLight: foundation.Some(ui.NewColor(11)), ScaffoldBackgroundColor: foundation.Some(ui.NewColor(12)),
		SecondaryHeaderColor: foundation.Some(ui.NewColor(13)), ShadowColor: foundation.Some(ui.NewColor(14)),
		SplashColor: foundation.Some(ui.NewColor(15)), UnselectedWidgetColor: foundation.Some(ui.NewColor(16)),
		DialogBackgroundColor: foundation.Some(ui.NewColor(17)), IndicatorColor: foundation.Some(ui.NewColor(18)),
		IconTheme:        foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(7.0)})),
		PrimaryIconTheme: foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(8.0)})),
		Extensions:       []ThemeExtension{&extensionB{fontSize(3)}},
	})
	copied := NewThemeData().CopyWith(ThemeDataCopy{
		ApplyElevationOverlayColor: foundation.Some(other.ApplyElevationOverlayColor),
		Extensions:                 foundation.Some(other.Extensions),
		MaterialTapTargetSize:      foundation.Some(other.MaterialTapTargetSize),
		Platform:                   foundation.Some(other.Platform),
		UseMaterial3:               foundation.Some(other.UseMaterial3),
		VisualDensity:              foundation.Some(other.VisualDensity),
		CanvasColor:                foundation.Some(other.CanvasColor), CardColor: foundation.Some(other.CardColor),
		ColorScheme:   foundation.Some(other.ColorScheme),
		DisabledColor: foundation.Some(other.DisabledColor), DividerColor: foundation.Some(other.DividerColor),
		FocusColor: foundation.Some(other.FocusColor), HighlightColor: foundation.Some(other.HighlightColor),
		HintColor: foundation.Some(other.HintColor), HoverColor: foundation.Some(other.HoverColor),
		PrimaryColor: foundation.Some(other.PrimaryColor), PrimaryColorDark: foundation.Some(other.PrimaryColorDark),
		PrimaryColorLight: foundation.Some(other.PrimaryColorLight), ScaffoldBackgroundColor: foundation.Some(other.ScaffoldBackgroundColor),
		SecondaryHeaderColor: foundation.Some(other.SecondaryHeaderColor), ShadowColor: foundation.Some(other.ShadowColor),
		SplashColor: foundation.Some(other.SplashColor), UnselectedWidgetColor: foundation.Some(other.UnselectedWidgetColor),
		SplashFactory: foundation.Some(other.SplashFactory),
		IconTheme:     foundation.Some(other.IconTheme), PrimaryIconTheme: foundation.Some(other.PrimaryIconTheme),
		PrimaryTextTheme: foundation.Some(other.PrimaryTextTheme), TextTheme: foundation.Some(other.TextTheme),
		Typography:            foundation.Some(other.Typography),
		DialogBackgroundColor: foundation.Some(other.DialogBackgroundColor), IndicatorColor: foundation.Some(other.IndicatorColor),
	})
	if !copied.Equal(other) {
		t.Error("a theme copied with every field of another is not equal to it")
	}
}

func TestThemeDataSplashFactory(t *testing.T) {
	tests := []struct {
		name string
		o    ThemeDataOpts
		want InteractiveInkFeatureFactory
	}{
		{"Material 3 ripples", ThemeDataOpts{}, InkRippleFactory},
		{"Material 2 splashes", ThemeDataOpts{UseMaterial3: foundation.Some(false)}, InkSplashFactory},
		{"explicit", ThemeDataOpts{SplashFactory: InkSplashFactory}, InkSplashFactory},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewThemeData(tt.o).SplashFactory; got != tt.want {
				t.Errorf("SplashFactory = %T, want %T", got, tt.want)
			}
		})
	}
	if NewThemeData().Equal(NewThemeData(ThemeDataOpts{SplashFactory: InkSplashFactory})) {
		t.Error("themes with different splash factories are equal")
	}
}

func TestThemeDataToStringIsShort(t *testing.T) {
	for _, scheme := range []*ColorScheme{NewColorSchemeDark(), NewColorSchemeLight()} {
		if got := NewThemeDataFrom(scheme, nil, foundation.None[bool]()).String(); len(got) >= 200 {
			t.Errorf("description has %d characters: %s", len(got), got)
		}
	}
}

func TestThemeDataCopyWithBrightness(t *testing.T) {
	light := NewColorSchemeLight()
	theme := NewThemeDataFrom(light, nil, foundation.None[bool]()).CopyWith(ThemeDataCopy{Brightness: ui.BrightnessDark})
	// The brightness only overrides the brightness of the color scheme.
	if theme.Brightness() != ui.BrightnessDark || theme.ColorScheme.Brightness() != ui.BrightnessDark {
		t.Errorf("brightness = %v", theme.Brightness())
	}
	for name, got := range map[string]ui.Color{"primary": theme.PrimaryColor} {
		if got != light.Primary() {
			t.Errorf("%s color = %v, want the one of the light scheme", name, got)
		}
	}
	for name, got := range map[string]ui.Color{"card": theme.CardColor, "canvas": theme.CanvasColor,
		"scaffold": theme.ScaffoldBackgroundColor, "dialog": theme.DialogBackgroundColor} {
		if got != light.Surface() {
			t.Errorf("%s color = %v, want the surface of the light scheme", name, got)
		}
	}
	if theme.ApplyElevationOverlayColor {
		t.Error("applyElevationOverlayColor is true")
	}
}

func TestThemeDataDiagnosticsIncludeAllProperties(t *testing.T) {
	skipUnlessDebug(t)
	// The names of the properties of Flutter's ThemeData that the Go port has so far;
	// the component themes come with their components.
	flutterNames := []string{"adaptations", "applyElevationOverlayColor", "cupertinoOverrideTheme", "extensions", "inputDecorationTheme",
		"materialTapTargetSize", "pageTransitionsTheme", "platform", "scrollbarTheme", "splashFactory", "visualDensity", "useMaterial3",
		"colorScheme", "primaryColor", "primaryColorLight", "primaryColorDark", "focusColor", "hoverColor", "shadowColor", "canvasColor",
		"scaffoldBackgroundColor", "cardColor", "dividerColor", "highlightColor", "splashColor", "unselectedWidgetColor", "disabledColor",
		"secondaryHeaderColor", "hintColor", "typography", "textTheme", "primaryTextTheme", "iconTheme", "primaryIconTheme",
		"dialogBackgroundColor", "indicatorColor", "appBarTheme", "bannerTheme", "bottomAppBarTheme", "snackBarTheme", "drawerTheme", "bottomSheetTheme", "navigationBarTheme", "navigationRailTheme", "navigationDrawerTheme", "tabBarTheme", "textButtonTheme", "elevatedButtonTheme", "filledButtonTheme", "outlinedButtonTheme", "iconButtonTheme", "floatingActionButtonTheme", "segmentedButtonTheme", "actionIconTheme", "menuTheme", "menuBarTheme", "menuButtonTheme", "popupMenuTheme", "dropdownMenuTheme", "textSelectionTheme", "checkboxTheme", "radioTheme", "switchTheme", "sliderTheme", "chipTheme", "searchBarTheme", "searchViewTheme", "dialogTheme", "datePickerTheme", "timePickerTheme", "cardTheme", "dividerTheme", "tooltipTheme", "badgeTheme", "expansionTileTheme", "listTileTheme", "progressIndicatorTheme", "dataTableTheme", "carouselViewTheme"}
	builder := foundation.NewDiagnosticPropertiesBuilder()
	NewThemeData().DebugFillProperties(builder)
	var names []string
	for _, node := range builder.Properties {
		names = append(names, node.Name().Value())
	}
	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			t.Errorf("duplicate property %s", name)
		}
		seen[name] = true
		if !slices.Contains(flutterNames, name) {
			t.Errorf("property %s is not one of Flutter's", name)
		}
	}
	for _, name := range []string{"colorScheme", "textTheme", "visualDensity", "useMaterial3", "indicatorColor"} {
		if !seen[name] {
			t.Errorf("missing property %s", name)
		}
	}
}

func TestThemeDataLerpEdges(t *testing.T) {
	a := NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformAndroid)})
	b := NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformIOS), UseMaterial3: foundation.Some(false)})
	tests := []struct {
		t            float64
		wantPlatform foundation.TargetPlatform
		wantM3       bool
	}{{0.49, foundation.TargetPlatformAndroid, true}, {0.5, foundation.TargetPlatformIOS, false}, {1, foundation.TargetPlatformIOS, false}}
	for _, tt := range tests {
		got := ThemeDataLerp(a, b, tt.t)
		if got.Platform != tt.wantPlatform || got.UseMaterial3 != tt.wantM3 {
			t.Errorf("lerp at %v: platform %v, useMaterial3 %v", tt.t, got.Platform, got.UseMaterial3)
		}
	}
}

func TestThemeDataLocalize(t *testing.T) {
	base := NewThemeData()
	geometry := NewTextTheme(TextThemeOpts{BodyMedium: fontSize(99)})
	localized := ThemeDataLocalize(base, geometry)
	// The theme has colors and no geometry, so the geometry is what the localized
	// theme gets, and the color of the theme stays.
	if got := localized.TextTheme.BodyMedium.FontSize.Value(); got != 99 {
		t.Errorf("body medium size = %v, want 99", got)
	}
	if got, want := localized.TextTheme.BodyMedium.Color, base.TextTheme.BodyMedium.Color; got != want {
		t.Errorf("body medium color = %v, want %v", got, want)
	}
	if localized == base {
		t.Error("localize returned the base theme")
	}
}

func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug-only checks of the framework")
	}
}
