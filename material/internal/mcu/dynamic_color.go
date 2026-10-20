package mcu

import "math"

// DynamicColor is a color that adjusts itself to a DynamicScheme: it has a role in
// the scheme (a palette and a tone), and, when it is meant to be read on another
// color, it moves its tone until the contrast with that color is what the
// scheme's contrast level asks for.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_color.dart DynamicColor
type DynamicColor struct {
	// Name is the name of the color, such as "primary_container".
	Name string
	// Palette is the palette of the scheme that the color is from.
	Palette func(*DynamicScheme) *TonalPalette
	// Tone is the tone of the color in the scheme, before it is adjusted for
	// contrast.
	Tone func(*DynamicScheme) float64
	// IsBackground says that the color is a background: it is kept out of the
	// awkward zone of the tones 50 to 59, where neither a light nor a dark
	// foreground has enough contrast.
	IsBackground bool
	// Background is the color that this color is drawn over, or nil.
	Background func(*DynamicScheme) *DynamicColor
	// SecondBackground is a second color that this color is drawn over, or nil.
	SecondBackground func(*DynamicScheme) *DynamicColor
	// ContrastCurve is the contrast ratio that the color needs against its
	// background; it is set when Background is.
	ContrastCurve *ContrastCurve
	// ToneDeltaPair is the constraint between this color and another, or nil.
	ToneDeltaPair func(*DynamicScheme) *ToneDeltaPair
}

// DynamicColorOpts are the parameters of NewDynamicColorFromPalette.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_color.dart DynamicColor.fromPalette
type DynamicColorOpts struct {
	// Name is the name of the color.
	Name string
	// IsBackground says that the color is a background.
	IsBackground bool
	// Background is the color that this color is drawn over.
	Background func(*DynamicScheme) *DynamicColor
	// SecondBackground is a second color that this color is drawn over.
	SecondBackground func(*DynamicScheme) *DynamicColor
	// ContrastCurve is the contrast ratio against the background.
	ContrastCurve *ContrastCurve
	// ToneDeltaPair is the constraint between this color and another.
	ToneDeltaPair func(*DynamicScheme) *ToneDeltaPair
}

// NewDynamicColorFromPalette creates a color from the palette and the tone of its
// role in a scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_color.dart DynamicColor.fromPalette
func NewDynamicColorFromPalette(palette func(*DynamicScheme) *TonalPalette, tone func(*DynamicScheme) float64, o DynamicColorOpts) *DynamicColor {
	return &DynamicColor{
		Name: o.Name, Palette: palette, Tone: tone, IsBackground: o.IsBackground, Background: o.Background,
		SecondBackground: o.SecondBackground, ContrastCurve: o.ContrastCurve, ToneDeltaPair: o.ToneDeltaPair,
	}
}

// GetArgb is the ARGB color in the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_color.dart DynamicColor.getArgb
func (d *DynamicColor) GetArgb(scheme *DynamicScheme) uint32 { return d.GetHct(scheme).ToInt() }

// GetHct is the color in the scheme. The scheme remembers the answer.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_color.dart DynamicColor.getHct
func (d *DynamicColor) GetHct(scheme *DynamicScheme) *Hct {
	if cached, ok := scheme.hctCache[d]; ok {
		return cached
	}
	answer := d.Palette(scheme).GetHct(d.GetTone(scheme))
	if scheme.hctCache == nil {
		scheme.hctCache = map[*DynamicColor]*Hct{}
	}
	scheme.hctCache[d] = answer
	return answer
}

// GetTone is the tone of the color in the scheme, adjusted for the contrast with
// its background (or backgrounds) at the contrast level of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_color.dart DynamicColor.getTone
func (d *DynamicColor) GetTone(scheme *DynamicScheme) float64 {
	decreasingContrast := scheme.ContrastLevel < 0

	// Case 1: a dual foreground, a pair of colors with a delta constraint.
	if d.ToneDeltaPair != nil {
		pair := d.ToneDeltaPair(scheme)
		roleA, roleB := pair.RoleA, pair.RoleB
		delta, polarity, stayTogether := pair.Delta, pair.Polarity, pair.StayTogether

		bg := d.Background(scheme)
		bgTone := bg.GetTone(scheme)

		aIsNearer := polarity == TonePolarityNearer ||
			(polarity == TonePolarityLighter && !scheme.IsDark) ||
			(polarity == TonePolarityDarker && scheme.IsDark)
		nearer, farther := roleB, roleA
		if aIsNearer {
			nearer, farther = roleA, roleB
		}
		amNearer := d.Name == nearer.Name
		expansionDir := -1.0
		if scheme.IsDark {
			expansionDir = 1.0
		}

		// 1st round: solve to the minimum, each.
		nContrast := nearer.ContrastCurve.Get(scheme.ContrastLevel)
		fContrast := farther.ContrastCurve.Get(scheme.ContrastLevel)

		// A color that is good enough is not adjusted.
		nInitialTone := nearer.Tone(scheme)
		nTone := ForegroundTone(bgTone, nContrast)
		if RatioOfTones(bgTone, nInitialTone) >= nContrast {
			nTone = nInitialTone
		}
		fInitialTone := farther.Tone(scheme)
		fTone := ForegroundTone(bgTone, fContrast)
		if RatioOfTones(bgTone, fInitialTone) >= fContrast {
			fTone = fInitialTone
		}

		if decreasingContrast {
			// When decreasing contrast, adjust the color to the bare minimum that
			// satisfies the contrast.
			nTone = ForegroundTone(bgTone, nContrast)
			fTone = ForegroundTone(bgTone, fContrast)
		}

		if (fTone-nTone)*expansionDir < delta {
			// 2nd round: expand the farther one to match the delta.
			fTone = ClampDouble(0, 100, nTone+delta*expansionDir)
			if (fTone-nTone)*expansionDir < delta {
				// 3rd round: contract the nearer one to match the delta.
				nTone = ClampDouble(0, 100, fTone-delta*expansionDir)
			}
		}

		// Avoid the awkward zone of the tones 50 to 59.
		switch {
		case 50 <= nTone && nTone < 60:
			// The nearer one is in the awkward zone: move it away, with the farther one.
			if expansionDir > 0 {
				nTone = 60
				fTone = math.Max(fTone, nTone+delta*expansionDir)
			} else {
				nTone = 49
				fTone = math.Min(fTone, nTone+delta*expansionDir)
			}
		case 50 <= fTone && fTone < 60:
			if stayTogether {
				// Fix both, to avoid two colors on opposite sides of the awkward zone.
				if expansionDir > 0 {
					nTone = 60
					fTone = math.Max(fTone, nTone+delta*expansionDir)
				} else {
					nTone = 49
					fTone = math.Min(fTone, nTone+delta*expansionDir)
				}
			} else if expansionDir > 0 {
				// They need not stay together: fix just one.
				fTone = 60
			} else {
				fTone = 49
			}
		}

		// The nearer tone if this color is the nearer one, otherwise the farther one.
		if amNearer {
			return nTone
		}
		return fTone
	}

	// Case 2: no contrast pair; solve for itself.
	answer := d.Tone(scheme)
	if d.Background == nil {
		return answer // No adjustment for colors with no background.
	}
	bgTone := d.Background(scheme).GetTone(scheme)
	desiredRatio := d.ContrastCurve.Get(scheme.ContrastLevel)

	if RatioOfTones(bgTone, answer) < desiredRatio {
		// A rough improvement; what is good enough is not "improved".
		answer = ForegroundTone(bgTone, desiredRatio)
	}
	if decreasingContrast {
		answer = ForegroundTone(bgTone, desiredRatio)
	}
	if d.IsBackground && 50 <= answer && answer < 60 {
		// Must adjust.
		if RatioOfTones(49, bgTone) >= desiredRatio {
			answer = 49
		} else {
			answer = 60
		}
	}

	if d.SecondBackground != nil {
		// Case 3: adjust for dual backgrounds.
		bgTone1 := d.Background(scheme).GetTone(scheme)
		bgTone2 := d.SecondBackground(scheme).GetTone(scheme)
		upper, lower := math.Max(bgTone1, bgTone2), math.Min(bgTone1, bgTone2)
		if RatioOfTones(upper, answer) >= desiredRatio && RatioOfTones(lower, answer) >= desiredRatio {
			return answer
		}
		// The darkest light tone that satisfies the desired ratio, or -1 if such a
		// ratio cannot be reached.
		lightOption := Lighter(upper, desiredRatio)
		// The lightest dark tone that satisfies the desired ratio, or -1.
		darkOption := Darker(lower, desiredRatio)

		// The tones that are suitable for the foreground.
		available := 0
		var only float64
		if lightOption != -1 {
			available++
			only = lightOption
		}
		if darkOption != -1 {
			available++
			if available == 1 {
				only = darkOption
			}
		}
		prefersLight := TonePrefersLightForeground(bgTone1) || TonePrefersLightForeground(bgTone2)
		if prefersLight {
			if lightOption < 0 {
				return 100
			}
			return lightOption
		}
		if available == 1 {
			return only
		}
		if darkOption < 0 {
			return 0
		}
		return darkOption
	}
	return answer
}

// ForegroundTone is the tone of a foreground on a background of the tone bgTone
// that has the contrast ratio with it: the lighter one when the background
// prefers a light foreground, unless the darker one has the better ratio.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_color.dart DynamicColor.foregroundTone
func ForegroundTone(bgTone, ratio float64) float64 {
	lighterTone := LighterUnsafe(bgTone, ratio)
	darkerTone := DarkerUnsafe(bgTone, ratio)
	lighterRatio := RatioOfTones(lighterTone, bgTone)
	darkerRatio := RatioOfTones(darkerTone, bgTone)
	preferLighter := TonePrefersLightForeground(bgTone)

	if preferLighter {
		// This handles an edge case where the initial contrast ratio is high (for
		// example 13.0), the ratio passed is that high ratio, and both the lighter
		// and the darker ratio fail to pass it. It was observed with Tonal Spot's
		// On Primary Container turning black momentarily between high and max
		// contrast in light mode: the tone of Primary Container was T90, the one of
		// On Primary Container T10, and the contrast value was 0.6568521221032331.
		negligibleDifference := math.Abs(lighterRatio-darkerRatio) < 0.1 && lighterRatio < ratio && darkerRatio < ratio
		if lighterRatio >= ratio || lighterRatio >= darkerRatio || negligibleDifference {
			return lighterTone
		}
		return darkerTone
	}
	if darkerRatio >= ratio || darkerRatio >= lighterRatio {
		return darkerTone
	}
	return lighterTone
}

// EnableLightForeground is the tone, or 49 when the tone prefers a light
// foreground that it does not allow: with a tone from 50 to 59 a light foreground
// does not have the contrast.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_color.dart DynamicColor.enableLightForeground
func EnableLightForeground(tone float64) float64 {
	if TonePrefersLightForeground(tone) && !ToneAllowsLightForeground(tone) {
		return 49.0
	}
	return tone
}

// TonePrefersLightForeground reports whether a light foreground is preferred
// over the tone: the rounded tone is below 60.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_color.dart DynamicColor.tonePrefersLightForeground
func TonePrefersLightForeground(tone float64) bool { return math.Round(tone) < 60 }

// ToneAllowsLightForeground reports whether a light foreground has enough
// contrast over the tone: the rounded tone is at most 49.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_color.dart DynamicColor.toneAllowsLightForeground
func ToneAllowsLightForeground(tone float64) bool { return math.Round(tone) <= 49 }
