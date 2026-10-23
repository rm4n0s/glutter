package painting

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// TextScaler defines a policy for scaling font sizes by the user's text size
// setting. Implement Scale and TextScaleFactor, and Clamp with TextScalerClamp
// (the default of Flutter's abstract class).
//
// Flutter: painting/text_scaler.dart TextScaler
type TextScaler interface {
	// Scale returns the font size scaled by the policy.
	Scale(fontSize float64) float64
	// TextScaleFactor is the approximate scale factor of the policy, for what
	// uses a factor instead of Scale.
	TextScaleFactor() float64
	// Clamp returns a scaler that limits the factor of this one.
	Clamp(opts ...TextScalerClampOpts) TextScaler
	String() string
}

// TextScalerClampOpts are the optional parameters of TextScaler.Clamp.
//
// Flutter: painting/text_scaler.dart TextScaler.clamp
type TextScalerClampOpts struct {
	// MinScaleFactor defaults to 0; it must be finite and not negative.
	MinScaleFactor float64
	// MaxScaleFactor defaults to infinity; it must not be less than the minimum.
	MaxScaleFactor foundation.Opt[float64]
}

func (o TextScalerClampOpts) resolved() (minScale, maxScale float64) {
	minScale, maxScale = o.MinScaleFactor, o.MaxScaleFactor.OrElse(math.Inf(1))
	if foundation.DebugMode && !(maxScale >= minScale && !math.IsNaN(maxScale) && !math.IsInf(minScale, 0) && !math.IsNaN(minScale) && minScale >= 0) {
		panic(errors.New(tagTextScalerClampInvalid, "the scale factors must satisfy 0 <= minScaleFactor <= maxScaleFactor",
			"minScaleFactor", minScale, "maxScaleFactor", maxScale))
	}
	return minScale, maxScale
}

// TextScalerNoScaling is a TextScaler that doesn't scale.
//
// Flutter: painting/text_scaler.dart TextScaler.noScaling
var TextScalerNoScaling TextScaler = linearTextScaler{1.0}

// NewTextScalerLinear creates a scaler that scales font sizes by the factor. In
// debug builds a negative factor panics with the tag TextScalerInvalid.
//
// Flutter: painting/text_scaler.dart TextScaler.linear
func NewTextScalerLinear(textScaleFactor float64) TextScaler {
	if foundation.DebugMode && !(textScaleFactor >= 0) {
		panic(errors.New(tagTextScalerInvalid, "the text scale factor must not be negative", "textScaleFactor", textScaleFactor))
	}
	return linearTextScaler{textScaleFactor}
}

// TextScalerClamp is the default Clamp of a TextScaler: this scaler, limited to
// the factors.
//
// Flutter: painting/text_scaler.dart TextScaler.clamp
func TextScalerClamp(s TextScaler, opts ...TextScalerClampOpts) TextScaler {
	minScale, maxScale := firstOpts(opts).resolved()
	if minScale == 0 && math.IsInf(maxScale, 1) {
		return s
	}
	if minScale == maxScale {
		return NewTextScalerLinear(minScale)
	}
	return clampedTextScaler{s, minScale, maxScale}
}

// linearTextScaler is Flutter's _LinearTextScaler.
type linearTextScaler struct{ factor float64 }

func (s linearTextScaler) TextScaleFactor() float64 { return s.factor }

func (s linearTextScaler) Scale(fontSize float64) float64 {
	if foundation.DebugMode && !(fontSize >= 0 && !math.IsInf(fontSize, 0) && !math.IsNaN(fontSize)) {
		panic(errors.New(tagTextScalerFontSizeInvalid, "the font size to scale must be finite and not negative", "fontSize", fontSize))
	}
	return fontSize * s.factor
}

func (s linearTextScaler) Clamp(opts ...TextScalerClampOpts) TextScaler {
	minScale, maxScale := firstOpts(opts).resolved()
	f := ui.ClampDouble(s.factor, minScale, maxScale)
	if f == s.factor {
		return s
	}
	return linearTextScaler{f}
}

func (s linearTextScaler) String() string {
	if s.factor == 1.0 {
		return "no scaling"
	}
	return "linear (" + dartfmt.Double(s.factor) + "x)"
}

// clampedTextScaler is Flutter's _ClampedTextScaler.
type clampedTextScaler struct {
	scaler             TextScaler
	minScale, maxScale float64
}

func (s clampedTextScaler) TextScaleFactor() float64 {
	return ui.ClampDouble(s.scaler.TextScaleFactor(), s.minScale, s.maxScale)
}

func (s clampedTextScaler) Scale(fontSize float64) float64 {
	if foundation.DebugMode && !(fontSize >= 0 && !math.IsInf(fontSize, 0) && !math.IsNaN(fontSize)) {
		panic(errors.New(tagTextScalerFontSizeInvalid, "the font size to scale must be finite and not negative", "fontSize", fontSize))
	}
	return ui.ClampDouble(s.scaler.Scale(fontSize), s.minScale*fontSize, s.maxScale*fontSize)
}

func (s clampedTextScaler) Clamp(opts ...TextScalerClampOpts) TextScaler {
	minScale, maxScale := firstOpts(opts).resolved()
	newMin, newMax := math.Max(s.minScale, minScale), math.Min(s.maxScale, maxScale)
	if newMax <= newMin {
		return NewTextScalerLinear(ui.ClampDouble(s.minScale, minScale, maxScale))
	}
	return clampedTextScaler{s.scaler, newMin, newMax}
}

func (s clampedTextScaler) String() string {
	return s.scaler.String() + " clamped [" + dartfmt.Double(s.minScale) + ", " + dartfmt.Double(s.maxScale) + "]"
}

// TextScalerEqual is Dart's == on text scalers: linear scalers are equal when
// their factors are, clamped ones when their limits and scalers are.
//
// Flutter: painting/text_scaler.dart TextScaler.operator ==
func TextScalerEqual(a, b TextScaler) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch x := a.(type) {
	case linearTextScaler:
		y, ok := b.(linearTextScaler)
		return ok && x == y
	case clampedTextScaler:
		y, ok := b.(clampedTextScaler)
		return ok && x.minScale == y.minScale && x.maxScale == y.maxScale && TextScalerEqual(x.scaler, y.scaler)
	}
	return sameValue(a, b)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTextScalerClampInvalid    = "TextScalerClampInvalid"
	tagTextScalerFontSizeInvalid = "TextScalerFontSizeInvalid"
	tagTextScalerInvalid         = "TextScalerInvalid"
)
