package mcu

import "math"

// RatioOfTones is the contrast ratio of two tones, from 1 to 21. The tones are
// clamped to 0..100.
//
// Flutter: material-color-utilities contrast/contrast.dart Contrast.ratioOfTones
func RatioOfTones(toneA, toneB float64) float64 {
	toneA = ClampDouble(0.0, 100.0, toneA)
	toneB = ClampDouble(0.0, 100.0, toneB)
	return ratioOfYs(YFromLstar(toneA), YFromLstar(toneB))
}

func ratioOfYs(y1, y2 float64) float64 {
	lighter := y2
	if y1 > y2 {
		lighter = y1
	}
	darker := y2
	if lighter == y2 {
		darker = y1
	}
	return (lighter + 5.0) / (darker + 5.0)
}

// Lighter is the tone that is lighter than the tone and has the contrast ratio
// with it, or -1 when there is none (the ratio can't be reached, or the tone is
// not in 0..100). The result is 0.4 higher than the exact tone, so that mapping
// to the gamut keeps the ratio.
//
// Flutter: material-color-utilities contrast/contrast.dart Contrast.lighter
func Lighter(tone, ratio float64) float64 {
	if tone < 0.0 || tone > 100.0 {
		return -1.0
	}
	darkY := YFromLstar(tone)
	lightY := ratio*(darkY+5.0) - 5.0
	realContrast := ratioOfYs(lightY, darkY)
	delta := math.Abs(realContrast - ratio)
	if realContrast < ratio && delta > 0.04 {
		return -1
	}
	// Gamut mapping needs a range on the tone: lighten slightly to keep the ratio.
	v := LstarFromY(lightY) + 0.4
	if v < 0 || v > 100 {
		return -1
	}
	return v
}

// Darker is the tone that is darker than the tone and has the contrast ratio
// with it, or -1 when there is none.
//
// Flutter: material-color-utilities contrast/contrast.dart Contrast.darker
func Darker(tone, ratio float64) float64 {
	if tone < 0.0 || tone > 100.0 {
		return -1.0
	}
	lightY := YFromLstar(tone)
	darkY := ((lightY + 5.0) / ratio) - 5.0
	realContrast := ratioOfYs(lightY, darkY)
	delta := math.Abs(realContrast - ratio)
	if realContrast < ratio && delta > 0.04 {
		return -1
	}
	// Darken slightly to keep the ratio after gamut mapping.
	v := LstarFromY(darkY) - 0.4
	if v < 0 || v > 100 {
		return -1
	}
	return v
}

// LighterUnsafe is like Lighter, but 100 when there is no such tone. The ratio
// is then not reached.
//
// Flutter: material-color-utilities contrast/contrast.dart Contrast.lighterUnsafe
func LighterUnsafe(tone, ratio float64) float64 {
	if l := Lighter(tone, ratio); l >= 0.0 {
		return l
	}
	return 100.0
}

// DarkerUnsafe is like Darker, but 0 when there is no such tone. The ratio is
// then not reached.
//
// Flutter: material-color-utilities contrast/contrast.dart Contrast.darkerUnsafe
func DarkerUnsafe(tone, ratio float64) float64 {
	if d := Darker(tone, ratio); d >= 0.0 {
		return d
	}
	return 0.0
}
