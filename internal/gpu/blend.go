package gpu

import "github.com/rm4n0s/glutter/ui"

// BlendFactor is a factor of a fixed-function blend equation.
//
// Flutter: no counterpart
type BlendFactor int

// The factors that the fixed-function blend modes use.
const (
	// BlendFactorZero is 0.
	//
	// Flutter: no counterpart
	BlendFactorZero BlendFactor = iota
	// BlendFactorOne is 1.
	//
	// Flutter: no counterpart
	BlendFactorOne
	// BlendFactorOneMinusSrcAlpha is 1 minus the alpha of the source.
	//
	// Flutter: no counterpart
	BlendFactorOneMinusSrcAlpha
	// BlendFactorOneMinusDstAlpha is 1 minus the alpha of the destination.
	//
	// Flutter: no counterpart
	BlendFactorOneMinusDstAlpha
)

// FixedBlend is the equation of a blend mode that the blend unit makes exactly when the
// shape's coverage scales the source: source * Src + destination * Dst, for color and
// alpha alike.
//
// Flutter: no counterpart
type FixedBlend struct{ Src, Dst BlendFactor }

// FixedBlendOf returns the fixed-function equation of a blend mode; false for the modes
// that need a shader reading a copy of the destination (SPEC §9.1 "Blend modes"). The
// zero mode is srcOver.
//
// Flutter: no counterpart
func FixedBlendOf(mode ui.BlendMode) (FixedBlend, bool) {
	switch mode {
	case ui.BlendModeUnspecified, ui.BlendModeSrcOver:
		return FixedBlend{BlendFactorOne, BlendFactorOneMinusSrcAlpha}, true
	case ui.BlendModeDstOver:
		return FixedBlend{BlendFactorOneMinusDstAlpha, BlendFactorOne}, true
	case ui.BlendModePlus:
		return FixedBlend{BlendFactorOne, BlendFactorOne}, true
	case ui.BlendModeDstOut:
		// What a shape does not cover keeps its pixels, and what it covers loses its
		// source alpha's share: the coverage scales the source, as for the others.
		return FixedBlend{BlendFactorZero, BlendFactorOneMinusSrcAlpha}, true
	}
	return FixedBlend{}, false
}
