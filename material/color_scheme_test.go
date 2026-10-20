package material_test

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/material/internal/mcu"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// colorGetters reads every color of a scheme by the name of its getter.
var colorGetters = map[string]func(*ColorScheme) ui.Color{
	"Primary": (*ColorScheme).Primary, "OnPrimary": (*ColorScheme).OnPrimary,
	"PrimaryContainer": (*ColorScheme).PrimaryContainer, "OnPrimaryContainer": (*ColorScheme).OnPrimaryContainer,
	"PrimaryFixed": (*ColorScheme).PrimaryFixed, "PrimaryFixedDim": (*ColorScheme).PrimaryFixedDim,
	"OnPrimaryFixed": (*ColorScheme).OnPrimaryFixed, "OnPrimaryFixedVariant": (*ColorScheme).OnPrimaryFixedVariant,
	"Secondary": (*ColorScheme).Secondary, "OnSecondary": (*ColorScheme).OnSecondary,
	"SecondaryContainer": (*ColorScheme).SecondaryContainer, "OnSecondaryContainer": (*ColorScheme).OnSecondaryContainer,
	"SecondaryFixed": (*ColorScheme).SecondaryFixed, "SecondaryFixedDim": (*ColorScheme).SecondaryFixedDim,
	"OnSecondaryFixed": (*ColorScheme).OnSecondaryFixed, "OnSecondaryFixedVariant": (*ColorScheme).OnSecondaryFixedVariant,
	"Tertiary": (*ColorScheme).Tertiary, "OnTertiary": (*ColorScheme).OnTertiary,
	"TertiaryContainer": (*ColorScheme).TertiaryContainer, "OnTertiaryContainer": (*ColorScheme).OnTertiaryContainer,
	"TertiaryFixed": (*ColorScheme).TertiaryFixed, "TertiaryFixedDim": (*ColorScheme).TertiaryFixedDim,
	"OnTertiaryFixed": (*ColorScheme).OnTertiaryFixed, "OnTertiaryFixedVariant": (*ColorScheme).OnTertiaryFixedVariant,
	"Error": (*ColorScheme).Error, "OnError": (*ColorScheme).OnError,
	"ErrorContainer": (*ColorScheme).ErrorContainer, "OnErrorContainer": (*ColorScheme).OnErrorContainer,
	"Background": (*ColorScheme).Background, "OnBackground": (*ColorScheme).OnBackground,
	"Surface": (*ColorScheme).Surface, "SurfaceDim": (*ColorScheme).SurfaceDim, "SurfaceBright": (*ColorScheme).SurfaceBright,
	"SurfaceContainerLowest": (*ColorScheme).SurfaceContainerLowest, "SurfaceContainerLow": (*ColorScheme).SurfaceContainerLow,
	"SurfaceContainer": (*ColorScheme).SurfaceContainer, "SurfaceContainerHigh": (*ColorScheme).SurfaceContainerHigh,
	"SurfaceContainerHighest": (*ColorScheme).SurfaceContainerHighest,
	"OnSurface":               (*ColorScheme).OnSurface, "SurfaceVariant": (*ColorScheme).SurfaceVariant,
	"OnSurfaceVariant": (*ColorScheme).OnSurfaceVariant, "Outline": (*ColorScheme).Outline,
	"OutlineVariant": (*ColorScheme).OutlineVariant, "Shadow": (*ColorScheme).Shadow, "Scrim": (*ColorScheme).Scrim,
	"InverseSurface": (*ColorScheme).InverseSurface, "OnInverseSurface": (*ColorScheme).OnInverseSurface,
	"InversePrimary": (*ColorScheme).InversePrimary, "SurfaceTint": (*ColorScheme).SurfaceTint,
}

func checkColors(t *testing.T, scheme *ColorScheme, want []colorWant) {
	t.Helper()
	for _, w := range want {
		get, ok := colorGetters[w.name]
		if !ok {
			t.Fatalf("no getter named %s", w.name)
		}
		expected := ui.NewColor(w.value)
		if w.sameAs != "" {
			expected = colorGetters[w.sameAs](scheme)
		}
		if got := get(scheme); got != expected {
			t.Errorf("%s = %v, want %v", w.name, got, expected)
		}
	}
}

func TestColorSchemeLerpSpecialCases(t *testing.T) {
	scheme := NewColorSchemeLight()
	if ColorSchemeLerp(scheme, scheme, 0.5) != scheme {
		t.Error("lerp of identical schemes is not the scheme itself")
	}
}

func TestColorSchemeMatchesTheSpec(t *testing.T) {
	tests := []struct {
		name       string
		scheme     *ColorScheme
		brightness ui.Brightness
		want       []colorWant
	}{
		{"light", NewColorSchemeLight(), ui.BrightnessLight, lightSchemeWant},
		{"dark", NewColorSchemeDark(), ui.BrightnessDark, darkSchemeWant},
		{"high contrast light", NewColorSchemeHighContrastLight(), ui.BrightnessLight, highContrastLightWant},
		{"high contrast dark", NewColorSchemeHighContrastDark(), ui.BrightnessDark, highContrastDarkWant},
		{"light from seed", ColorSchemeFromSeed(Colors.Blue.Primary), ui.BrightnessLight, seedLightWant},
		{"dark from seed", ColorSchemeFromSeed(Colors.Blue.Primary, ColorSchemeFromSeedOpts{Brightness: ui.BrightnessDark}), ui.BrightnessDark, seedDarkWant},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scheme.Brightness(); got != tt.brightness {
				t.Errorf("brightness = %v, want %v", got, tt.brightness)
			}
			checkColors(t, tt.scheme, tt.want)
		})
	}
}

func TestColorSchemeCopyWithOverridesGivenColors(t *testing.T) {
	scheme := NewColorSchemeLight().CopyWith(copyWithArgs)
	if got := scheme.Brightness(); got != ui.BrightnessDark {
		t.Errorf("brightness = %v, want dark", got)
	}
	checkColors(t, scheme, copyWithWant)
	if !NewColorSchemeLight().Equal(NewColorSchemeLight().CopyWith(ColorSchemeCopy{})) {
		t.Error("copyWith with no arguments changed the scheme")
	}
}

func TestColorSchemeFromSeedOverridesSpecificColors(t *testing.T) {
	base := ColorSchemeFromSeed(Colors.Blue.Primary)
	override := ui.NewColor(0xffabcdef)
	scheme := ColorSchemeFromSeed(Colors.Blue.Primary, ColorSchemeFromSeedOpts{
		Overrides: ColorSchemeVariantOpts{Primary: foundation.Some(override)},
	})
	if got := scheme.Primary(); got != override {
		t.Errorf("primary = %v, want %v", got, override)
	}
	for name, get := range colorGetters {
		if name == "Primary" {
			continue
		}
		if got, want := get(scheme), get(base); got != want {
			t.Errorf("%s = %v, want the one of the base scheme %v", name, got, want)
		}
	}
}

func TestColorSchemeFromSeedAssertsOnInvalidContrastLevels(t *testing.T) {
	skipUnlessDebug(t)
	for _, level := range []float64{-1.5, 1.5} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			defer func() {
				appErr, ok := errors.FromError(asError(recover()))
				if !ok {
					t.Fatalf("expected *errors.Error")
				}
				if !appErr.HasRoute("ColorSchemeContrastLevel") {
					t.Errorf("unexpected failure path: %s", appErr.Route())
				}
			}()
			ColorSchemeFromSeed(Colors.Blue.Primary, ColorSchemeFromSeedOpts{ContrastLevel: level})
		})
	}
}

func asError(v any) error {
	if err, ok := v.(error); ok {
		return err
	}
	return nil
}

// TestColorSchemeFromSeedSpotChecks is the spot checks of "ColorScheme.fromSeed
// with different variants" in Flutter, which read the background of a filled
// button; that is the primary color of the scheme.
func TestColorSchemeFromSeedSpotChecks(t *testing.T) {
	tests := []struct {
		seed    uint32
		variant DynamicSchemeVariant
		want    uint32
	}{
		{0xFF000000, DynamicSchemeVariantUnspecified, 0xFF8C4A60},
		{0xFF00FF00, DynamicSchemeVariantUnspecified, 0xFF406836},
		{0xFF6559F5, DynamicSchemeVariantUnspecified, 0xFF5B5891},
		{0xFFFFFFFF, DynamicSchemeVariantUnspecified, 0xFF006874},
		{0xFF000000, DynamicSchemeVariantFidelity, 0xFF000000},
		{0xFF00FF00, DynamicSchemeVariantFidelity, 0xFF026E00},
		{0xFF6559F5, DynamicSchemeVariantFidelity, 0xFF4C3CDB},
		{0xFFFFFFFF, DynamicSchemeVariantFidelity, 0xFF5D5F5F},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%08X %v", tt.seed, tt.variant), func(t *testing.T) {
			scheme := ColorSchemeFromSeed(ui.NewColor(tt.seed), ColorSchemeFromSeedOpts{DynamicSchemeVariant: tt.variant})
			if got, want := scheme.Primary(), ui.NewColor(tt.want); got != want {
				t.Errorf("primary = %v, want %v", got, want)
			}
		})
	}
}

// dynamicColorsOf lists the colors of MaterialDynamicColors that a ColorScheme
// has, with the getter that reads each.
var dynamicColorsOf = map[string]*mcu.DynamicColor{}

func init() {
	for name, c := range map[string]*mcu.DynamicColor{
		"Primary": mcu.Primary, "OnPrimary": mcu.OnPrimary, "PrimaryContainer": mcu.PrimaryContainer,
		"OnPrimaryContainer": mcu.OnPrimaryContainer, "PrimaryFixed": mcu.PrimaryFixed, "PrimaryFixedDim": mcu.PrimaryFixedDim,
		"OnPrimaryFixed": mcu.OnPrimaryFixed, "OnPrimaryFixedVariant": mcu.OnPrimaryFixedVariant,
		"Secondary": mcu.Secondary, "OnSecondary": mcu.OnSecondary, "SecondaryContainer": mcu.SecondaryContainer,
		"OnSecondaryContainer": mcu.OnSecondaryContainer, "SecondaryFixed": mcu.SecondaryFixed,
		"SecondaryFixedDim": mcu.SecondaryFixedDim, "OnSecondaryFixed": mcu.OnSecondaryFixed,
		"OnSecondaryFixedVariant": mcu.OnSecondaryFixedVariant,
		"Tertiary":                mcu.Tertiary, "OnTertiary": mcu.OnTertiary, "TertiaryContainer": mcu.TertiaryContainer,
		"OnTertiaryContainer": mcu.OnTertiaryContainer, "TertiaryFixed": mcu.TertiaryFixed,
		"TertiaryFixedDim": mcu.TertiaryFixedDim, "OnTertiaryFixed": mcu.OnTertiaryFixed,
		"OnTertiaryFixedVariant": mcu.OnTertiaryFixedVariant,
		"Error":                  mcu.Error, "OnError": mcu.OnError, "ErrorContainer": mcu.ErrorContainer, "OnErrorContainer": mcu.OnErrorContainer,
		"Background": mcu.Background, "OnBackground": mcu.OnBackground,
		"Surface": mcu.Surface, "SurfaceDim": mcu.SurfaceDim, "SurfaceBright": mcu.SurfaceBright,
		"SurfaceContainerLowest": mcu.SurfaceContainerLowest, "SurfaceContainerLow": mcu.SurfaceContainerLow,
		"SurfaceContainer": mcu.SurfaceContainer, "SurfaceContainerHigh": mcu.SurfaceContainerHigh,
		"SurfaceContainerHighest": mcu.SurfaceContainerHighest,
		"OnSurface":               mcu.OnSurface, "SurfaceVariant": mcu.SurfaceVariant, "OnSurfaceVariant": mcu.OnSurfaceVariant,
		"Outline": mcu.Outline, "OutlineVariant": mcu.OutlineVariant, "Shadow": mcu.Shadow, "Scrim": mcu.Scrim,
		"InverseSurface": mcu.InverseSurface, "OnInverseSurface": mcu.InverseOnSurface, "InversePrimary": mcu.InversePrimary,
		"SurfaceTint": mcu.Primary,
	} {
		dynamicColorsOf[name] = c
	}
}

var schemeVariants = []struct {
	variant DynamicSchemeVariant
	make    func(*mcu.Hct, bool, float64) *mcu.DynamicScheme
}{
	{DynamicSchemeVariantTonalSpot, mcu.NewSchemeTonalSpot},
	{DynamicSchemeVariantFidelity, mcu.NewSchemeFidelity},
	{DynamicSchemeVariantContent, mcu.NewSchemeContent},
	{DynamicSchemeVariantMonochrome, mcu.NewSchemeMonochrome},
	{DynamicSchemeVariantNeutral, mcu.NewSchemeNeutral},
	{DynamicSchemeVariantVibrant, mcu.NewSchemeVibrant},
	{DynamicSchemeVariantExpressive, mcu.NewSchemeExpressive},
	{DynamicSchemeVariantRainbow, mcu.NewSchemeRainbow},
	{DynamicSchemeVariantFruitSalad, mcu.NewSchemeFruitSalad},
}

// TestColorSchemeMatchesDynamicScheme covers "Color values in
// ColorScheme.fromSeed with different variants matches values in
// DynamicScheme" and "Colors in high-contrast color scheme matches colors in
// DynamicScheme".
func TestColorSchemeMatchesDynamicScheme(t *testing.T) {
	for _, v := range schemeVariants {
		for _, brightness := range []ui.Brightness{ui.BrightnessLight, ui.BrightnessDark} {
			for _, contrast := range []float64{0, 0.5, 1} {
				name := fmt.Sprintf("variant %d %v contrast %v", v.variant, brightness, contrast)
				t.Run(name, func(t *testing.T) {
					seed := Colors.Orange.Primary
					dynamic := v.make(mcu.HctFromInt(seed.ToARGB32()), brightness == ui.BrightnessDark, contrast)
					scheme := ColorSchemeFromSeed(seed, ColorSchemeFromSeedOpts{
						Brightness: brightness, DynamicSchemeVariant: v.variant, ContrastLevel: contrast,
					})
					for name, get := range colorGetters {
						want := ui.NewColor(dynamicColorsOf[name].GetArgb(dynamic))
						if got := get(scheme); got != want {
							t.Errorf("%s = %v, want %v", name, got, want)
						}
					}
				})
			}
		}
	}
}

func TestColorSchemeOfIsTheColorSchemeOfTheTheme(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	scheme := ColorSchemeFromSeed(Colors.Red.Primary)
	var fromTheme, fromContext *ColorScheme
	ts.PumpWidget(NewTheme(NewThemeDataFrom(scheme, nil, foundation.None[bool]()), widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		fromTheme, fromContext = ThemeOf(bc).ColorScheme, ColorSchemeOf(bc)
		return widgets.NewSizedBox(nil)
	})))
	if !fromTheme.Equal(scheme) || !fromContext.Equal(scheme) {
		t.Error("the color scheme from the context is not the one of the theme")
	}
}
