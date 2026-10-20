package mcu

import "math"

// Harmonize shifts the hue of the design color towards the source color, by at
// most 15 degrees, keeping its chroma and tone: the custom colors of an app look
// like they belong to its theme.
//
// Flutter: material-color-utilities blend/blend.dart Blend.harmonize
func Harmonize(designColor, sourceColor uint32) uint32 {
	from, to := HctFromInt(designColor), HctFromInt(sourceColor)
	difference := DifferenceDegrees(from.Hue(), to.Hue())
	rotation := math.Min(difference*0.5, 15.0)
	outputHue := SanitizeDegreesDouble(from.Hue() + rotation*RotationDirection(from.Hue(), to.Hue()))
	return HctFrom(outputHue, from.Chroma(), from.Tone()).ToInt()
}

// BlendHctHue blends the hue of a color towards another, in CAM16-UCS, keeping the
// chroma and the tone of the first one. The amount is from 0 (the first color) to
// 1 (the hue of the second).
//
// Flutter: material-color-utilities blend/blend.dart Blend.hctHue
func BlendHctHue(from, to uint32, amount float64) uint32 {
	ucs := BlendCam16Ucs(from, to, amount)
	ucsCam := Cam16FromInt(ucs)
	fromCam := Cam16FromInt(from)
	return HctFrom(ucsCam.Hue, fromCam.Chroma, LstarFromArgb(from)).ToInt()
}

// BlendCam16Ucs blends two colors in CAM16-UCS.
//
// Flutter: material-color-utilities blend/blend.dart Blend.cam16Ucs
func BlendCam16Ucs(from, to uint32, amount float64) uint32 {
	fromCam, toCam := Cam16FromInt(from), Cam16FromInt(to)
	jstar := fromCam.Jstar + (toCam.Jstar-fromCam.Jstar)*amount
	astar := fromCam.Astar + (toCam.Astar-fromCam.Astar)*amount
	bstar := fromCam.Bstar + (toCam.Bstar-fromCam.Bstar)*amount
	return Cam16FromUcs(jstar, astar, bstar).ToInt()
}
