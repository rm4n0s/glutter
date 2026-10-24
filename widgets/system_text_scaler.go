package widgets

import (
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// SystemTextScaler is a TextScaler that reflects the user's font scale preferences from the
// platform's accessibility settings. Its Scale asks the platform dispatcher, which converts the
// size the way the platform does (not linearly on Android 14 and later). A MediaQueryData made
// from a view has one.
//
// Flutter: widgets/media_query.dart SystemTextScaler
type SystemTextScaler struct {
	dispatcher *ui.PlatformDispatcher
	factor     float64
}

func newSystemTextScaler(d *ui.PlatformDispatcher) SystemTextScaler {
	return SystemTextScaler{dispatcher: d, factor: d.TextScaleFactor()}
}

// Scale returns the font size scaled by the platform.
//
// Flutter: widgets/media_query.dart SystemTextScaler.scale
func (s SystemTextScaler) Scale(fontSize float64) float64 {
	return s.dispatcher.ScaleFontSize(fontSize)
}

// TextScaleFactor is a value that represents the current user preference for the scaling factor
// for fonts. It is used to compare SystemTextScalers: two with the same factor are equal as
// their Scale produces the same output for the same input. It should not be used in arithmetic.
//
// Flutter: widgets/media_query.dart SystemTextScaler.textScaleFactor
func (s SystemTextScaler) TextScaleFactor() float64 { return s.factor }

// Clamp returns a scaler that limits the factor of this one.
//
// Flutter: painting/text_scaler.dart TextScaler.clamp
func (s SystemTextScaler) Clamp(opts ...painting.TextScalerClampOpts) painting.TextScaler {
	return painting.TextScalerClamp(s, opts...)
}

// EqualTextScaler is the == of Flutter's SystemTextScaler, which painting.TextScalerEqual calls.
// The system's text scale factor is used for the check because the output of Scale increases
// monotonically with it. When the factor is 1, the scaler is equal to no scaling, since the two
// are extensionally equivalent.
//
// Flutter: widgets/media_query.dart SystemTextScaler.==
func (s SystemTextScaler) EqualTextScaler(other painting.TextScaler) bool {
	if o, ok := other.(SystemTextScaler); ok {
		return s.factor == o.factor
	}
	return s.factor == 1.0 && painting.TextScalerEqual(painting.TextScalerNoScaling, other)
}

// String describes the scaler.
//
// Flutter: widgets/media_query.dart SystemTextScaler.toString
func (s SystemTextScaler) String() string {
	if s.factor == 1.0 {
		return "SystemTextScaler (no scaling)"
	}
	return "SystemTextScaler (" + dartfmt.Double(s.factor) + "x)"
}
