package mcu

import (
	"fmt"
	"math"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// CommonTones are the tones that a palette lists: the ones of the Material 3
// tonal palettes.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.commonTones
var CommonTones = [...]int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 99, 100}

// CommonSize is the number of the common tones.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.commonSize
const CommonSize = len(CommonTones)

// TonalPalette is a convenience type for retrieving colors that are constant in
// hue and chroma, but vary in tone. A palette is not safe for concurrent use: it
// caches the colors it makes.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette
type TonalPalette struct {
	hue, chroma float64
	keyColor    *Hct
	cache       map[int]uint32
	isFromCache bool
}

// TonalPaletteOf creates a palette from a hue and a chroma, with the key color
// that the closest tone allows.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.of
func TonalPaletteOf(hue, chroma float64) *TonalPalette {
	return &TonalPalette{hue: hue, chroma: chroma, keyColor: NewKeyColor(hue, chroma).Create(), cache: map[int]uint32{}}
}

// TonalPaletteFromHct creates a palette from a color: the hue and the chroma of
// the color, and the color as the key color.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.fromHct
func TonalPaletteFromHct(hct *Hct) *TonalPalette {
	return &TonalPalette{hue: hct.Hue(), chroma: hct.Chroma(), keyColor: hct, cache: map[int]uint32{}}
}

// TonalPaletteFromList creates a palette from the colors of the common tones. The
// hue and the chroma are deduced from the color of the highest chroma (the
// colors that are too close to white are ignored; see
// https://github.com/material-foundation/material-color-utilities/issues/140).
// In debug builds it panics with the tag TonalPaletteListSize when the list does
// not have CommonSize colors.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.fromList
func TonalPaletteFromList(colors []uint32) *TonalPalette {
	if foundation.DebugMode && len(colors) != CommonSize {
		panic(errors.New("TonalPaletteListSize", "a palette needs one color for each common tone", "colors", len(colors), "want", CommonSize))
	}
	cache := make(map[int]uint32, CommonSize)
	for i, tone := range CommonTones {
		cache[tone] = colors[i]
	}
	var bestHue, bestChroma float64
	for _, argb := range colors {
		hct := HctFromInt(argb)
		if hct.Tone() > 98.0 {
			continue
		}
		if hct.Chroma() > bestChroma {
			bestHue, bestChroma = hct.Hue(), hct.Chroma()
		}
	}
	return &TonalPalette{hue: bestHue, chroma: bestChroma, keyColor: NewKeyColor(bestHue, bestChroma).Create(), cache: cache, isFromCache: true}
}

// Hue is the hue of the palette.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.hue
func (p *TonalPalette) Hue() float64 { return p.hue }

// Chroma is the chroma of the palette.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.chroma
func (p *TonalPalette) Chroma() float64 { return p.chroma }

// KeyColor is the key color of the palette: the color of the palette's hue with
// the tone that gives the requested chroma, closest to tone 50.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.keyColor
func (p *TonalPalette) KeyColor() *Hct { return p.keyColor }

// AsList is the colors of the common tones.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.asList
func (p *TonalPalette) AsList() []uint32 {
	out := make([]uint32, len(CommonTones))
	for i, tone := range CommonTones {
		out[i] = p.Get(tone)
	}
	return out
}

// Get is the ARGB color of the palette at the tone.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.get
func (p *TonalPalette) Get(tone int) uint32 {
	if argb, ok := p.cache[tone]; ok {
		return argb
	}
	argb := HctFrom(p.hue, p.chroma, float64(tone)).ToInt()
	p.cache[tone] = argb
	return argb
}

// GetHct is the color of the palette at the tone. A tone that is a whole number
// and was asked before comes from the cache.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.getHct
func (p *TonalPalette) GetHct(tone float64) *Hct {
	if tone == math.Trunc(tone) {
		if argb, ok := p.cache[int(tone)]; ok {
			return HctFromInt(argb)
		}
	}
	return HctFrom(p.hue, p.chroma, tone)
}

// Equal reports whether two palettes are the same: two palettes made from a hue
// and a chroma (or a color) by the same hue and chroma, and any palettes with the
// same colors at the common tones.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.==
func (p *TonalPalette) Equal(other *TonalPalette) bool {
	if other == nil {
		return false
	}
	if !p.isFromCache && !other.isFromCache {
		return p.hue == other.hue && p.chroma == other.chroma
	}
	return slices.Equal(p.AsList(), other.AsList())
}

// String is the palette as TonalPalette.of(hue, chroma) or
// TonalPalette.fromList(colors).
//
// Flutter: material-color-utilities palettes/tonal_palette.dart TonalPalette.toString
func (p *TonalPalette) String() string {
	if !p.isFromCache {
		return fmt.Sprintf("TonalPalette.of(%v, %v)", p.hue, p.chroma)
	}
	return fmt.Sprintf("TonalPalette.fromList(%v)", p.AsList())
}

// KeyColor finds the key color of a hue and a chroma: the color of that hue
// that has the requested chroma at the tone nearest to 50, or the most chromatic
// one when the hue can't reach it.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart KeyColor
type KeyColor struct {
	hue, requestedChroma float64
	chromaCache          map[int]float64
}

const maxChromaValue = 200.0

// NewKeyColor creates a key color finder.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart KeyColor.new
func NewKeyColor(hue, requestedChroma float64) *KeyColor {
	return &KeyColor{hue: hue, requestedChroma: requestedChroma, chromaCache: map[int]float64{}}
}

// Create finds the key color.
//
// Flutter: material-color-utilities palettes/tonal_palette.dart KeyColor.create
func (k *KeyColor) Create() *Hct {
	// Pivot around T50 because T50 has the most chroma available, on average, so
	// it is the most likely to have a direct answer.
	const pivotTone = 50
	const toneStepSize = 1
	// Epsilon to accept values slightly higher than the requested chroma.
	const epsilon = 0.01

	// Binary search for the tone that can provide a chroma closest to the
	// requested one.
	lowerTone, upperTone := 0, 100
	for lowerTone < upperTone {
		midTone := (lowerTone + upperTone) / 2
		isAscending := k.maxChroma(midTone) < k.maxChroma(midTone+toneStepSize)
		sufficientChroma := k.maxChroma(midTone) >= k.requestedChroma-epsilon

		if sufficientChroma {
			// Either [lowerTone, midTone] or [midTone, upperTone] has the answer:
			// search the range that is closer to the pivot tone.
			if abs(lowerTone-pivotTone) < abs(upperTone-pivotTone) {
				upperTone = midTone
			} else {
				if lowerTone == midTone {
					return HctFrom(k.hue, k.requestedChroma, float64(lowerTone))
				}
				lowerTone = midTone
			}
		} else if isAscending {
			// There is no sufficient chroma at midTone: follow the direction to the
			// chroma peak.
			lowerTone = midTone + toneStepSize
		} else {
			// Keep midTone for a potential chroma peak.
			upperTone = midTone
		}
	}
	return HctFrom(k.hue, k.requestedChroma, float64(lowerTone))
}

// maxChroma is the maximum chroma that the hue has at the tone.
func (k *KeyColor) maxChroma(tone int) float64 {
	if c, ok := k.chromaCache[tone]; ok {
		return c
	}
	c := HctFrom(k.hue, maxChromaValue, float64(tone)).Chroma()
	k.chromaCache[tone] = c
	return c
}
