package mcu

import "math"

// IsDisliked reports whether the color is a dark yellow-green that most people
// dislike: a hue from 90 to 111, a chroma above 16 and a tone below 65, each
// rounded.
//
// Flutter: material-color-utilities dislike/dislike_analyzer.dart DislikeAnalyzer.isDisliked
func IsDisliked(hct *Hct) bool {
	hue := math.Round(hct.Hue())
	huePasses := hue >= 90.0 && hue <= 111.0
	chromaPasses := math.Round(hct.Chroma()) > 16.0
	tonePasses := math.Round(hct.Tone()) < 65.0
	return huePasses && chromaPasses && tonePasses
}

// FixIfDisliked is the color, or the same color at tone 70 when the color is
// disliked.
//
// Flutter: material-color-utilities dislike/dislike_analyzer.dart DislikeAnalyzer.fixIfDisliked
func FixIfDisliked(hct *Hct) *Hct {
	if IsDisliked(hct) {
		return HctFrom(hct.Hue(), hct.Chroma(), 70.0)
	}
	return hct
}
