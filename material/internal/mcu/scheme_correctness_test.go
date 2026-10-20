package mcu

import (
	"fmt"
	"testing"
)

// Ported from scheme_correctness_test.dart: every variant, at every contrast
// level, for four seed colors in light and dark, must satisfy the contrast,
// tone delta and background constraints of the Material 3 color system.

type contrastConstraint struct {
	foreground, background *DynamicColor
	curve                  *ContrastCurve
}

type deltaConstraint struct {
	roleA, roleB *DynamicColor
	delta        float64
	polarity     TonePolarity
}

func describeScheme(s *DynamicScheme) string {
	brightness := "light"
	if s.IsDark {
		brightness = "dark"
	}
	return fmt.Sprintf("variant %s, source color %s, brightness %s, contrast level %v",
		s.Variant.Label(), s.SourceColorHct, brightness, s.ContrastLevel)
}

func (c contrastConstraint) testAgainst(t *testing.T, scheme *DynamicScheme) {
	t.Helper()
	const tolerance = 0.05
	fg, bg := c.foreground.GetHct(scheme), c.background.GetHct(scheme)
	actual := RatioOfTones(fg.Tone(), bg.Tone())
	desired := c.curve.Get(scheme.ContrastLevel)
	if desired <= 4.5 {
		// A requirement of 4.5 or less must be met, with a tolerance.
		if actual < desired-tolerance {
			t.Errorf("%s: %s should have contrast at least %v against %s, but has %v", describeScheme(scheme),
				c.foreground.Name, desired, c.background.Name, actual)
		}
		return
	}
	if actual < 4.5-tolerance {
		t.Errorf("%s: %s should have contrast at least 4.5 against %s, but has %v", describeScheme(scheme),
			c.foreground.Name, c.background.Name, actual)
	} else if actual < desired-tolerance && fg.Tone() != 100.0 && fg.Tone() != 0.0 {
		t.Errorf("%s: %s should have contrast at least %v against %s, but has %v, and no color has a tone of 0 or 100",
			describeScheme(scheme), c.foreground.Name, desired, c.background.Name, actual)
	}
}

func (c deltaConstraint) testAgainst(t *testing.T, scheme *DynamicScheme) {
	t.Helper()
	const tolerance = 0.5
	a, b := c.roleA.GetHct(scheme), c.roleB.GetHct(scheme)
	aShouldBeLighter := c.polarity == TonePolarityLighter ||
		(c.polarity == TonePolarityNearer && !scheme.IsDark) ||
		(c.polarity == TonePolarityFarther && scheme.IsDark)
	actual := b.Tone() - a.Tone()
	if aShouldBeLighter {
		actual = a.Tone() - b.Tone()
	}
	if actual < c.delta-tolerance {
		t.Errorf("%s: %s should be %v apart from %s, but they have tones %v and %v", describeScheme(scheme),
			c.roleA.Name, c.delta, c.roleB.Name, a.Tone(), b.Tone())
	}
}

func backgroundConstraint(t *testing.T, scheme *DynamicScheme, background *DynamicColor) {
	t.Helper()
	tone := background.GetHct(scheme).Tone()
	if tone >= 50.5 && tone < 59.5 {
		t.Errorf("%s: %s has tone %v which is in the forbidden zone 50.5 <= tone < 59.5", describeScheme(scheme), background.Name, tone)
	}
}

func schemeConstraints() (contrast []contrastConstraint, delta []deltaConstraint, backgrounds []*DynamicColor) {
	surfaces := []*DynamicColor{SurfaceDim, SurfaceBright}
	onSurface := NewContrastCurve(4.5, 7, 11, 21)
	accent := NewContrastCurve(3, 4.5, 7, 7)
	container := NewContrastCurve(0, 0, 3, 4.5)
	onContainer := NewContrastCurve(3, 4.5, 7, 11)
	for _, s := range surfaces {
		contrast = append(contrast, contrastConstraint{OnSurface, s, onSurface})
		for _, c := range []*DynamicColor{Primary, Secondary, Tertiary, Error} {
			contrast = append(contrast, contrastConstraint{c, s, accent})
		}
		contrast = append(contrast,
			contrastConstraint{OnSurfaceVariant, s, onContainer},
			contrastConstraint{Outline, s, NewContrastCurve(1.5, 3, 4.5, 7)})
		for _, c := range []*DynamicColor{
			PrimaryContainer, PrimaryFixed, PrimaryFixedDim, SecondaryContainer, SecondaryFixed, SecondaryFixedDim,
			TertiaryContainer, TertiaryFixed, TertiaryFixedDim, ErrorContainer, OutlineVariant,
		} {
			contrast = append(contrast, contrastConstraint{c, s, container})
		}
	}
	contrast = append(contrast,
		contrastConstraint{InverseOnSurface, InverseSurface, onSurface},
		contrastConstraint{InversePrimary, InverseSurface, accent},
		contrastConstraint{OnPrimary, Primary, onSurface},
		contrastConstraint{OnSecondary, Secondary, onSurface},
		contrastConstraint{OnTertiary, Tertiary, onSurface},
		contrastConstraint{OnError, Error, onSurface},
		contrastConstraint{OnPrimaryContainer, PrimaryContainer, onContainer},
		contrastConstraint{OnSecondaryContainer, SecondaryContainer, onContainer},
		contrastConstraint{OnTertiaryContainer, TertiaryContainer, onContainer},
		contrastConstraint{OnErrorContainer, ErrorContainer, onContainer},
	)
	for _, p := range [][3]*DynamicColor{
		{OnPrimaryFixed, PrimaryFixed, PrimaryFixedDim}, {OnSecondaryFixed, SecondaryFixed, SecondaryFixedDim},
		{OnTertiaryFixed, TertiaryFixed, TertiaryFixedDim},
	} {
		contrast = append(contrast, contrastConstraint{p[0], p[1], onSurface}, contrastConstraint{p[0], p[2], onSurface})
	}
	for _, p := range [][3]*DynamicColor{
		{OnPrimaryFixedVariant, PrimaryFixed, PrimaryFixedDim}, {OnSecondaryFixedVariant, SecondaryFixed, SecondaryFixedDim},
		{OnTertiaryFixedVariant, TertiaryFixed, TertiaryFixedDim},
	} {
		contrast = append(contrast, contrastConstraint{p[0], p[1], onContainer}, contrastConstraint{p[0], p[2], onContainer})
	}
	delta = []deltaConstraint{
		{Primary, PrimaryContainer, 10, TonePolarityFarther}, {Secondary, SecondaryContainer, 10, TonePolarityFarther},
		{Tertiary, TertiaryContainer, 10, TonePolarityFarther}, {Error, ErrorContainer, 10, TonePolarityFarther},
		{PrimaryFixedDim, PrimaryFixed, 10, TonePolarityDarker}, {SecondaryFixedDim, SecondaryFixed, 10, TonePolarityDarker},
		{TertiaryFixedDim, TertiaryFixed, 10, TonePolarityDarker},
	}
	backgrounds = []*DynamicColor{
		Background, Error, ErrorContainer, Primary, PrimaryContainer, PrimaryFixed, PrimaryFixedDim, Secondary, SecondaryContainer,
		SecondaryFixed, SecondaryFixedDim, Surface, SurfaceBright, SurfaceContainer, SurfaceContainerHigh, SurfaceContainerHighest,
		SurfaceContainerLow, SurfaceContainerLowest, SurfaceDim, SurfaceTint, SurfaceVariant, Tertiary, TertiaryContainer,
		TertiaryFixed, TertiaryFixedDim,
	}
	return contrast, delta, backgrounds
}

func TestSchemeCorrectness(t *testing.T) {
	contrast, delta, backgrounds := schemeConstraints()
	for variant := VariantMonochrome; variant <= VariantFruitSalad; variant++ {
		for _, contrastLevel := range []float64{-1.0, 0.0, 0.5, 1.0} {
			t.Run(fmt.Sprintf("%s, %v", variant.Label(), contrastLevel), func(t *testing.T) {
				for _, source := range []uint32{0xFF0000FF, 0xFF00FF00, 0xFFFFFF00, 0xFFFF0000} {
					for _, isDark := range []bool{false, true} {
						s := NewSchemeFromVariant(variant, HctFromInt(source), isDark, contrastLevel)
						for _, c := range contrast {
							c.testAgainst(t, s)
						}
						for _, d := range delta {
							d.testAgainst(t, s)
						}
						for _, b := range backgrounds {
							backgroundConstraint(t, s, b)
						}
					}
				}
			})
		}
	}
}
