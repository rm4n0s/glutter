package mcu

import (
	"fmt"
	"math"
)

// Hct is a color in the HCT color space: hue and chroma of CAM16, and the tone
// (L*) of the Lab color space. HCT is what Material 3 builds its palettes with:
// the tone is perceptually even, and a color of a given hue and chroma exists at
// every tone that the sRGB gamut allows.
//
// Flutter: material-color-utilities hct/hct.dart Hct
type Hct struct {
	hue, chroma, tone float64
	argb              uint32
}

// HctFrom creates a color from the hue, chroma and tone. The color is the
// closest one in the sRGB gamut, so its chroma can be lower than asked.
//
// Flutter: material-color-utilities hct/hct.dart Hct.from
func HctFrom(hue, chroma, tone float64) *Hct {
	return newHct(SolveHctToInt(hue, chroma, tone))
}

// HctFromInt creates a color from an ARGB value.
//
// Flutter: material-color-utilities hct/hct.dart Hct.fromInt
func HctFromInt(argb uint32) *Hct { return newHct(argb) }

func newHct(argb uint32) *Hct {
	h := &Hct{}
	h.setInt(argb)
	return h
}

func (h *Hct) setInt(argb uint32) {
	h.argb = argb
	cam16 := Cam16FromInt(argb)
	h.hue = cam16.Hue
	h.chroma = cam16.Chroma
	h.tone = LstarFromArgb(argb)
}

// Equal reports whether two colors are the same ARGB value.
//
// Flutter: material-color-utilities hct/hct.dart Hct.==
func (h *Hct) Equal(other *Hct) bool { return other != nil && h.argb == other.argb }

// String is the color as "H<hue> C<chroma> T<tone>", each rounded.
//
// Flutter: material-color-utilities hct/hct.dart Hct.toString
func (h *Hct) String() string {
	return fmt.Sprintf("H%d C%d T%d", int(math.Round(h.hue)), int(math.Round(h.chroma)), int(math.Round(h.tone)))
}

// ToInt is the ARGB value of the color.
//
// Flutter: material-color-utilities hct/hct.dart Hct.toInt
func (h *Hct) ToInt() uint32 { return h.argb }

// Hue is the hue, from 0 (inclusive) to 360 (exclusive).
//
// Flutter: material-color-utilities hct/hct.dart Hct.hue
func (h *Hct) Hue() float64 { return h.hue }

// SetHue changes the hue, keeping the chroma and the tone as far as the gamut
// allows.
//
// Flutter: material-color-utilities hct/hct.dart Hct.hue
func (h *Hct) SetHue(newHue float64) { h.setInt(SolveHctToInt(newHue, h.chroma, h.tone)) }

// Chroma is the chroma.
//
// Flutter: material-color-utilities hct/hct.dart Hct.chroma
func (h *Hct) Chroma() float64 { return h.chroma }

// SetChroma changes the chroma, keeping the hue and the tone as far as the gamut
// allows.
//
// Flutter: material-color-utilities hct/hct.dart Hct.chroma
func (h *Hct) SetChroma(newChroma float64) { h.setInt(SolveHctToInt(h.hue, newChroma, h.tone)) }

// Tone is the tone, from 0 (black) to 100 (white).
//
// Flutter: material-color-utilities hct/hct.dart Hct.tone
func (h *Hct) Tone() float64 { return h.tone }

// SetTone changes the tone, keeping the hue and the chroma as far as the gamut
// allows.
//
// Flutter: material-color-utilities hct/hct.dart Hct.tone
func (h *Hct) SetTone(newTone float64) { h.setInt(SolveHctToInt(h.hue, h.chroma, newTone)) }

// InViewingConditions translates the color into the color that looks the same
// in the viewing conditions: the HCT of the color when it is viewed there.
//
// Flutter: material-color-utilities hct/hct.dart Hct.inViewingConditions
func (h *Hct) InViewingConditions(vc *ViewingConditions) *Hct {
	// 1. Use CAM16 to find the XYZ coordinates of the color in the viewing conditions.
	cam16 := Cam16FromInt(h.ToInt())
	viewed := cam16.XyzInViewingConditions(vc)
	// 2. Create the CAM16 of those XYZ coordinates in the default conditions.
	recast := Cam16FromXyzInViewingConditions(viewed[0], viewed[1], viewed[2], NewViewingConditions(ViewingConditionsOpts{}))
	// 3. Create the HCT from that CAM16 and the L* of the Y in the viewing conditions.
	return HctFrom(recast.Hue, recast.Chroma, LstarFromY(viewed[1]))
}
