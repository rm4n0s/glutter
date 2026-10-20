package mcu

// ContrastCurve is a value that depends on the contrast level of a scheme: the
// contrast ratio that a color needs, at the lowest (-1), the standard (0), the
// medium (0.5) and the highest (1) contrast level.
//
// Flutter: material-color-utilities dynamiccolor/src/contrast_curve.dart ContrastCurve
type ContrastCurve struct {
	// Low is the value at contrast level -1.0.
	Low float64
	// Normal is the value at contrast level 0.0.
	Normal float64
	// Medium is the value at contrast level 0.5.
	Medium float64
	// High is the value at contrast level 1.0.
	High float64
}

// NewContrastCurve creates a curve from its four values.
//
// Flutter: material-color-utilities dynamiccolor/src/contrast_curve.dart ContrastCurve.new
func NewContrastCurve(low, normal, medium, high float64) *ContrastCurve {
	return &ContrastCurve{low, normal, medium, high}
}

// Get is the value at the contrast level, from -1.0 (the lowest contrast) to 1.0
// (the highest): the values are interpolated between the levels.
//
// Flutter: material-color-utilities dynamiccolor/src/contrast_curve.dart ContrastCurve.get
func (c *ContrastCurve) Get(contrastLevel float64) float64 {
	switch {
	case contrastLevel <= -1.0:
		return c.Low
	case contrastLevel < 0.0:
		return Lerp(c.Low, c.Normal, (contrastLevel-(-1))/1)
	case contrastLevel < 0.5:
		return Lerp(c.Normal, c.Medium, (contrastLevel-0)/0.5)
	case contrastLevel < 1.0:
		return Lerp(c.Medium, c.High, (contrastLevel-0.5)/0.5)
	}
	return c.High
}

// TonePolarity is the relationship of the tones of the two colors of a
// ToneDeltaPair.
//
// Flutter: material-color-utilities dynamiccolor/src/tone_delta_pair.dart TonePolarity
type TonePolarity int

// The tone polarities, in the order of Flutter's enum. TonePolarityUnspecified is
// the zero value and is not a polarity.
const (
	// TonePolarityUnspecified is the zero value.
	//
	// Flutter: no counterpart
	TonePolarityUnspecified TonePolarity = iota
	// TonePolarityDarker: roleA must be darker than roleB.
	//
	// Flutter: material-color-utilities dynamiccolor/src/tone_delta_pair.dart TonePolarity.darker
	TonePolarityDarker
	// TonePolarityLighter: roleA must be lighter than roleB.
	//
	// Flutter: material-color-utilities dynamiccolor/src/tone_delta_pair.dart TonePolarity.lighter
	TonePolarityLighter
	// TonePolarityNearer: the one closer to the background is roleA: it is lighter
	// in a light scheme and darker in a dark one, and roleB is farther.
	//
	// Flutter: material-color-utilities dynamiccolor/src/tone_delta_pair.dart TonePolarity.nearer
	TonePolarityNearer
	// TonePolarityFarther: the one farther from the background is roleA.
	//
	// Flutter: material-color-utilities dynamiccolor/src/tone_delta_pair.dart TonePolarity.farther
	TonePolarityFarther
)

// ToneDeltaPair documents a constraint between two DynamicColors, in which their
// tones must have a certain distance from each other.
//
// Flutter: material-color-utilities dynamiccolor/src/tone_delta_pair.dart ToneDeltaPair
type ToneDeltaPair struct {
	// RoleA and RoleB are the colors of the pair.
	RoleA, RoleB *DynamicColor
	// Delta is the difference in tone that the colors must have, at least.
	Delta float64
	// Polarity says which of the colors is lighter, darker, nearer or farther.
	Polarity TonePolarity
	// StayTogether says that the colors must be moved together out of the
	// awkward zone of the tones 50 to 59.
	StayTogether bool
}

// NewToneDeltaPair creates a tone delta pair.
//
// Flutter: material-color-utilities dynamiccolor/src/tone_delta_pair.dart ToneDeltaPair.new
func NewToneDeltaPair(roleA, roleB *DynamicColor, delta float64, polarity TonePolarity, stayTogether bool) *ToneDeltaPair {
	return &ToneDeltaPair{roleA, roleB, delta, polarity, stayTogether}
}
