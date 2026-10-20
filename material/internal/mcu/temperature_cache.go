package mcu

import (
	"math"
	"slices"
)

// TemperatureCache is a design system for color temperature: it ranks the colors
// of a color's chroma and tone around the hue wheel from cool to warm, and finds
// the complement and the analogous colors of the input by temperature. The
// results are cached; the type is not safe for concurrent use.
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache
type TemperatureCache struct {
	// Input is the color the cache was made for.
	Input *Hct

	hctsByTemp                 []*Hct
	hctsByHue                  []*Hct
	tempsByHct                 map[uint32]float64
	inputRelativeTemperature   float64
	inputRelativeTemperatureOK bool
	complement                 *Hct
}

// NewTemperatureCache creates the cache of a color.
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.new
func NewTemperatureCache(input *Hct) *TemperatureCache {
	return &TemperatureCache{Input: input}
}

// Warmest is the warmest color of the input's chroma and tone, in any hue.
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.warmest
func (c *TemperatureCache) Warmest() *Hct {
	h := c.HctsByTemp()
	return h[len(h)-1]
}

// Coldest is the coldest color of the input's chroma and tone, in any hue.
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.coldest
func (c *TemperatureCache) Coldest() *Hct { return c.HctsByTemp()[0] }

// Analogous is count colors that are the input and its neighbours on the hue
// wheel, found by dividing the wheel into divisions steps of equal temperature
// difference: the colors with a lower hue come first, the input is in the
// middle (when count is odd). The defaults of Flutter are 5 and 12.
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.analogous
func (c *TemperatureCache) Analogous(count, divisions int) []*Hct {
	startHue := int(math.Round(c.Input.Hue()))
	hctsByHue := c.HctsByHue()
	startHct := hctsByHue[startHue]
	lastTemp := c.RelativeTemperature(startHct)
	allColors := []*Hct{startHct}

	absoluteTotalTempDelta := 0.0
	for i := range 360 {
		hue := SanitizeDegreesInt(startHue + i)
		temp := c.RelativeTemperature(hctsByHue[hue])
		absoluteTotalTempDelta += math.Abs(temp - lastTemp)
		lastTemp = temp
	}
	hueAddend := 1
	tempStep := absoluteTotalTempDelta / float64(divisions)
	totalTempDelta := 0.0
	lastTemp = c.RelativeTemperature(startHct)
	for len(allColors) < divisions {
		hue := SanitizeDegreesInt(startHue + hueAddend)
		hct := hctsByHue[hue]
		temp := c.RelativeTemperature(hct)
		totalTempDelta += math.Abs(temp - lastTemp)

		desiredTotalTempDeltaForIndex := float64(len(allColors)) * tempStep
		indexSatisfied := totalTempDelta >= desiredTotalTempDeltaForIndex
		indexAddend := 1
		// Keep adding this hue to the answers until its temperature is
		// insufficient. This keeps the behavior consistent when there aren't
		// `divisions` discrete steps between 0 and 360 in hue with `tempStep`
		// delta in temperature between them: white and black have no analogues, so
		// they are just added to the array as answers.
		for indexSatisfied && len(allColors) < divisions {
			allColors = append(allColors, hct)
			desired := float64(len(allColors)+indexAddend) * tempStep
			indexSatisfied = totalTempDelta >= desired
			indexAddend++
		}
		lastTemp = temp
		hueAddend++
		if hueAddend > 360 {
			for len(allColors) < divisions {
				allColors = append(allColors, hct)
			}
			break
		}
	}

	answers := []*Hct{c.Input}

	// First, the analogues from rotating counter-clockwise.
	increaseHueCount := int(math.Floor(float64(count-1) / 2.0))
	for i := 1; i < increaseHueCount+1; i++ {
		index := 0 - i
		for index < 0 {
			index = len(allColors) + index
		}
		if index >= len(allColors) {
			index %= len(allColors)
		}
		answers = slices.Insert(answers, 0, allColors[index])
	}

	// Second, the analogues from rotating clockwise.
	decreaseHueCount := count - increaseHueCount - 1
	for i := 1; i < decreaseHueCount+1; i++ {
		index := i
		for index < 0 {
			index = len(allColors) + index
		}
		if index >= len(allColors) {
			index %= len(allColors)
		}
		answers = append(answers, allColors[index])
	}
	return answers
}

// Complement is the color of the opposite temperature: its relative temperature
// is the inverse of the input's, on the other side of the wheel.
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.complement
func (c *TemperatureCache) Complement() *Hct {
	if c.complement != nil {
		return c.complement
	}
	coldest, warmest := c.Coldest(), c.Warmest()
	temps := c.TempsByHct()
	coldestHue, coldestTemp := coldest.Hue(), temps[coldest.ToInt()]
	warmestHue, warmestTemp := warmest.Hue(), temps[warmest.ToInt()]
	rng := warmestTemp - coldestTemp
	startHueIsColdestToWarmest := IsBetween(c.Input.Hue(), coldestHue, warmestHue)
	startHue, endHue := coldestHue, warmestHue
	if startHueIsColdestToWarmest {
		startHue, endHue = warmestHue, coldestHue
	}
	const directionOfRotation = 1.0
	smallestError := 1000.0
	answer := c.HctsByHue()[int(math.Round(c.Input.Hue()))]

	complementRelativeTemp := 1.0 - c.InputRelativeTemperature()
	// Find the color in the other section, closest to the inverse percentile of
	// the input color. This is the complement.
	for hueAddend := 0.0; hueAddend <= 360.0; hueAddend += 1.0 {
		hue := SanitizeDegreesDouble(startHue + directionOfRotation*hueAddend)
		if !IsBetween(hue, startHue, endHue) {
			continue
		}
		possible := c.HctsByHue()[int(math.Round(hue))]
		relativeTemp := (temps[possible.ToInt()] - coldestTemp) / rng
		if err := math.Abs(complementRelativeTemp - relativeTemp); err < smallestError {
			smallestError = err
			answer = possible
		}
	}
	c.complement = answer
	return answer
}

// RelativeTemperature is the temperature of the color relative to the coldest
// (0) and the warmest (1) of the cache. It is 0.5 when there is no difference,
// as at tone 100, where only white is available.
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.relativeTemperature
func (c *TemperatureCache) RelativeTemperature(hct *Hct) float64 {
	temps := c.TempsByHct()
	coldest := temps[c.Coldest().ToInt()]
	rng := temps[c.Warmest().ToInt()] - coldest
	if rng == 0.0 {
		return 0.5
	}
	return (temps[hct.ToInt()] - coldest) / rng
}

// InputRelativeTemperature is the relative temperature of the input.
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.inputRelativeTemperature
func (c *TemperatureCache) InputRelativeTemperature() float64 {
	if c.inputRelativeTemperatureOK {
		return c.inputRelativeTemperature
	}
	temps := c.TempsByHct()
	coldestTemp := temps[c.Coldest().ToInt()]
	rng := temps[c.Warmest().ToInt()] - coldestTemp
	rel := 0.5
	if rng != 0.0 {
		rel = (temps[c.Input.ToInt()] - coldestTemp) / rng
	}
	c.inputRelativeTemperature, c.inputRelativeTemperatureOK = rel, true
	return rel
}

// HctsByTemp is the colors of every whole hue and the input, from the coldest to
// the warmest.
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.hctsByTemp
func (c *TemperatureCache) HctsByTemp() []*Hct {
	if len(c.hctsByTemp) > 0 {
		return c.hctsByTemp
	}
	hcts := append(slices.Clone(c.HctsByHue()), c.Input)
	temps := c.TempsByHct()
	slices.SortStableFunc(hcts, func(a, b *Hct) int {
		ta, tb := temps[a.ToInt()], temps[b.ToInt()]
		switch {
		case ta < tb:
			return -1
		case ta > tb:
			return 1
		}
		return 0
	})
	c.hctsByTemp = hcts
	return hcts
}

// TempsByHct is the raw temperature of every color of HctsByHue and the input,
// by ARGB value (colors are equal when their ARGB values are).
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.tempsByHct
func (c *TemperatureCache) TempsByHct() map[uint32]float64 {
	if len(c.tempsByHct) > 0 {
		return c.tempsByHct
	}
	all := append(slices.Clone(c.HctsByHue()), c.Input)
	temps := make(map[uint32]float64, len(all))
	for _, h := range all {
		temps[h.ToInt()] = RawTemperature(h)
	}
	c.tempsByHct = temps
	return temps
}

// HctsByHue is the colors of the input's chroma and tone at the hues 0 to 360.
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.hctsByHue
func (c *TemperatureCache) HctsByHue() []*Hct {
	if len(c.hctsByHue) > 0 {
		return c.hctsByHue
	}
	hcts := make([]*Hct, 0, 361)
	for hue := 0.0; hue <= 360.0; hue += 1.0 {
		hcts = append(hcts, HctFrom(hue, c.Input.Chroma(), c.Input.Tone()))
	}
	c.hctsByHue = hcts
	return hcts
}

// IsBetween reports whether the angle is between a and b, going from a to b in
// the increasing direction.
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.isBetween
func IsBetween(angle, a, b float64) bool {
	if a < b {
		return a <= angle && angle <= b
	}
	return a <= angle || angle <= b
}

// RawTemperature is the temperature of a color, from a model of Ou, Woodcock and
// Wright: it follows the Lab hue and chroma, and is highest for the red-orange
// hues. Values are around -0.5 (cool) to 1 (warm).
//
// Flutter: material-color-utilities temperature/temperature_cache.dart TemperatureCache.rawTemperature
func RawTemperature(color *Hct) float64 {
	lab := LabFromArgb(color.ToInt())
	hue := SanitizeDegreesDouble(math.Atan2(lab[2], lab[1]) * 180.0 / math.Pi)
	chroma := math.Sqrt((lab[1] * lab[1]) + (lab[2] * lab[2]))
	return -0.5 + 0.02*math.Pow(chroma, 1.07)*math.Cos(SanitizeDegreesDouble(hue-50.0)*math.Pi/180.0)
}
