package mcu

import (
	"fmt"
	"testing"
)

func toneOf(c *DynamicColor, s *DynamicScheme) float64 { return c.GetHct(s).Tone() }

func schemeSeeds() []*Hct {
	return []*Hct{HctFromInt(0xFFFF0000), HctFromInt(0xFFFFFF00), HctFromInt(0xFF00FF00), HctFromInt(0xFF0000FF)}
}

// Ported from scheme_monochrome_test.dart.
func TestMonochromeSpec(t *testing.T) {
	tests := []struct {
		name   string
		isDark bool
		want   map[*DynamicColor]float64
	}{
		{"dark", true, map[*DynamicColor]float64{
			Primary: 100, OnPrimary: 10, PrimaryContainer: 85, OnPrimaryContainer: 0,
			Secondary: 80, OnSecondary: 10, SecondaryContainer: 30, OnSecondaryContainer: 90,
			Tertiary: 90, OnTertiary: 10, TertiaryContainer: 60, OnTertiaryContainer: 0,
		}},
		{"light", false, map[*DynamicColor]float64{
			Primary: 0, OnPrimary: 90, PrimaryContainer: 25, OnPrimaryContainer: 100,
			Secondary: 40, OnSecondary: 100, SecondaryContainer: 85, OnSecondaryContainer: 10,
			Tertiary: 25, OnTertiary: 90, TertiaryContainer: 49, OnTertiaryContainer: 100,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := NewSchemeMonochrome(HctFromInt(0xff0000ff), tt.isDark, 0.0)
			for c, want := range tt.want {
				closeTo(t, c.Name, toneOf(c, scheme), want, 1)
			}
		})
	}
}

// Ported from dynamic_color_test.dart.
func TestDynamicColorValuesAreCorrect(t *testing.T) {
	tests := []struct {
		name   string
		color  *DynamicColor
		scheme *DynamicScheme
		want   uint32
	}{
		{"on primary container of fidelity", OnPrimaryContainer, NewSchemeFidelity(HctFromInt(0xFFFF0000), false, 0.5), 0xFFFFFFFF},
		{"on secondary container of content", OnSecondaryContainer, NewSchemeContent(HctFromInt(0xFF0000FF), false, 0.5), 0xFFFFFFFF},
		{"on tertiary container of content", OnTertiaryContainer, NewSchemeContent(HctFromInt(0xFFFFFF00), true, -0.5), 0xff959b1a},
		{"inverse surface of content", InverseSurface, NewSchemeContent(HctFromInt(0xFF0000FF), false, 0.0), 0xFF2F2F3B},
		{"inverse primary of content", InversePrimary, NewSchemeContent(HctFromInt(0xFFFF0000), false, -0.5), 0xffff422f},
		{"outline variant of content", OutlineVariant, NewSchemeContent(HctFromInt(0xFFFFFF00), true, 0.0), 0xFF484831},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireColor(t, tt.name, tt.color.GetArgb(tt.scheme), tt.want)
		})
	}
}

// TestTextSurfacePairsRespectContrast checks, for every scheme, that each text and
// surface pair has a contrast of at least 4.5 (3.0 when the contrast is reduced).
func TestTextSurfacePairsRespectContrast(t *testing.T) {
	colors := map[string]*DynamicColor{
		"background": Background, "on_background": OnBackground, "surface": Surface, "surface_dim": SurfaceDim,
		"surface_bright": SurfaceBright, "surface_container_lowest": SurfaceContainerLowest,
		"surface_container_low": SurfaceContainerLow, "surface_container": SurfaceContainer,
		"surface_container_high": SurfaceContainerHigh, "surface_container_highest": SurfaceContainerHighest,
		"on_surface": OnSurface, "surface_variant": SurfaceVariant, "on_surface_variant": OnSurfaceVariant,
		"inverse_surface": InverseSurface, "inverse_on_surface": InverseOnSurface, "outline": Outline,
		"outline_variant": OutlineVariant, "shadow": Shadow, "scrim": Scrim, "surface_tint": SurfaceTint,
		"primary": Primary, "on_primary": OnPrimary, "primary_container": PrimaryContainer,
		"on_primary_container": OnPrimaryContainer, "inverse_primary": InversePrimary, "secondary": Secondary,
		"on_secondary": OnSecondary, "secondary_container": SecondaryContainer, "on_secondary_container": OnSecondaryContainer,
		"tertiary": Tertiary, "on_tertiary": OnTertiary, "tertiary_container": TertiaryContainer,
		"on_tertiary_container": OnTertiaryContainer, "error": Error, "on_error": OnError,
		"error_container": ErrorContainer, "on_error_container": OnErrorContainer,
	}
	pairs := [][2]string{
		{"on_primary", "primary"}, {"on_primary_container", "primary_container"}, {"on_secondary", "secondary"},
		{"on_secondary_container", "secondary_container"}, {"on_tertiary", "tertiary"},
		{"on_tertiary_container", "tertiary_container"}, {"on_error", "error"}, {"on_error_container", "error_container"},
		{"on_background", "background"}, {"on_surface_variant", "surface_bright"}, {"on_surface_variant", "surface_dim"},
		{"inverse_on_surface", "inverse_surface"},
	}
	for name, color := range colors {
		if color.Name != name {
			t.Errorf("the color %q is named %q", name, color.Name)
		}
	}
	makers := map[string]func(*Hct, bool, float64) *DynamicScheme{
		"content": NewSchemeContent, "monochrome": NewSchemeMonochrome, "tonal spot": NewSchemeTonalSpot, "fidelity": NewSchemeFidelity,
	}
	for _, seed := range schemeSeeds() {
		for _, contrastLevel := range []float64{-1.0, -0.5, 0.0, 0.5, 1.0} {
			for _, isDark := range []bool{false, true} {
				for kind, make := range makers {
					scheme := make(seed, isDark, contrastLevel)
					name := fmt.Sprintf("%s; seed %s; contrast %v; dark %v", kind, HexFromArgb(seed.ToInt(), true), contrastLevel, isDark)
					t.Run(name, func(t *testing.T) {
						for _, pair := range pairs {
							fgTone, bgTone := toneOf(colors[pair[0]], scheme), toneOf(colors[pair[1]], scheme)
							contrast := RatioOfTones(fgTone, bgTone)
							minimum := 4.5
							if contrastLevel < 0.0 {
								minimum = 3.0
							}
							if contrast < minimum {
								t.Errorf("contrast %v is too low between foreground (%s; %v) and background (%s; %v)",
									contrast, pair[0], fgTone, pair[1], bgTone)
							}
						}
					})
				}
			}
		}
	}
}

func TestFixedColors(t *testing.T) {
	order := []*DynamicColor{
		PrimaryFixed, PrimaryFixedDim, OnPrimaryFixed, OnPrimaryFixedVariant,
		SecondaryFixed, SecondaryFixedDim, OnSecondaryFixed, OnSecondaryFixedVariant,
		TertiaryFixed, TertiaryFixedDim, OnTertiaryFixed, OnTertiaryFixedVariant,
	}
	tests := []struct {
		name   string
		scheme *DynamicScheme
		tones  [12]float64
	}{
		{"non-monochrome", NewSchemeTonalSpot(HctFromInt(0xFFFF0000), true, 0.0), [12]float64{90, 80, 10, 30, 90, 80, 10, 30, 90, 80, 10, 30}},
		{"light monochrome", NewSchemeMonochrome(HctFromInt(0xFFFF0000), false, 0.0), [12]float64{40, 30, 100, 90, 80, 70, 10, 25, 40, 30, 100, 90}},
		{"dark monochrome", NewSchemeMonochrome(HctFromInt(0xFFFF0000), true, 0.0), [12]float64{40, 30, 100, 90, 80, 70, 10, 25, 40, 30, 100, 90}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, c := range order {
				closeTo(t, c.Name, toneOf(c, tt.scheme), tt.tones[i], 1.0)
			}
		})
	}
}

func TestSchemeGettersAreTheDynamicColors(t *testing.T) {
	s := NewSchemeTonalSpot(HctFromInt(0xff6750a4), false, 0)
	for name, tc := range map[string][2]uint32{
		"primary": {s.Primary(), Primary.GetArgb(s)}, "surface": {s.Surface(), Surface.GetArgb(s)},
		"onTertiaryFixedVariant": {s.OnTertiaryFixedVariant(), OnTertiaryFixedVariant.GetArgb(s)},
		"error":                  {s.Error(), Error.GetArgb(s)}, "key color": {s.PrimaryPaletteKeyColor(), PrimaryPaletteKeyColor.GetArgb(s)},
	} {
		requireColor(t, name, tc[0], tc[1])
	}
	if s.GetHct(Primary) != Primary.GetHct(s) {
		t.Error("the scheme does not remember its colors")
	}
}

func TestContrastCurveInterpolates(t *testing.T) {
	c := NewContrastCurve(1, 2, 4, 8)
	for level, want := range map[float64]float64{-2: 1, -1: 1, -0.5: 1.5, 0: 2, 0.25: 3, 0.5: 4, 0.75: 6, 1: 8, 2: 8} {
		closeTo(t, fmt.Sprint("level ", level), c.Get(level), want, 1e-9)
	}
}

func TestVariantLabelsAndDescriptions(t *testing.T) {
	if VariantFruitSalad.Label() != "fruit salad" || VariantTonalSpot.Label() != "tonal spot" || VariantMonochrome.Label() != "monochrome" {
		t.Error("labels")
	}
	for v := VariantMonochrome; v <= VariantFruitSalad; v++ {
		if v.Description() == "" || v.Label() == "" {
			t.Errorf("variant %d has no label or description", v)
		}
	}
	if NewSchemeFromVariant(VariantFidelity, HctFromInt(red), false, 0).Variant != VariantFidelity ||
		NewSchemeFromVariant(VariantUnspecified, HctFromInt(red), false, 0).Variant != VariantTonalSpot {
		t.Error("NewSchemeFromVariant")
	}
}

func TestToneHelpers(t *testing.T) {
	if !TonePrefersLightForeground(59.4) || TonePrefersLightForeground(59.6) || !ToneAllowsLightForeground(49.4) || ToneAllowsLightForeground(49.6) {
		t.Error("tone preferences")
	}
	if EnableLightForeground(55) != 49 || EnableLightForeground(40) != 40 || EnableLightForeground(70) != 70 {
		t.Error("enableLightForeground")
	}
}
