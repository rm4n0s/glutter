package mcu

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// ViewingConditions are the environment in which a color is viewed, as CAM16
// takes it: the white point, the luminance that the eye is adapted to, the
// background and the surround, and the factors derived from them.
//
// Flutter: material-color-utilities hct/viewing_conditions.dart ViewingConditions
type ViewingConditions struct {
	WhitePoint            [3]float64
	AdaptingLuminance     float64
	BackgroundLstar       float64
	Surround              float64
	DiscountingIlluminant bool

	BackgroundYToWhitePointY float64
	Aw, Nbb, Ncb, C, NC      float64
	RgbD                     [3]float64
	Fl, FLRoot, Z            float64
}

// ViewingConditionsOpts are the optional parameters of NewViewingConditions.
//
// Flutter: material-color-utilities hct/viewing_conditions.dart ViewingConditions.make
type ViewingConditionsOpts struct {
	// WhitePoint is the white point in XYZ; the default is D65.
	WhitePoint foundation.Opt[[3]float64]
	// AdaptingLuminance is the luminance of the adapting field in cd/m2; the
	// default (unset, or zero or less) is 11.72, a 50% gray in an environment of
	// 200 lux.
	AdaptingLuminance foundation.Opt[float64]
	// BackgroundLstar is the L* of the background; the default is 50.
	BackgroundLstar foundation.Opt[float64]
	// Surround is the surround, from 0 (dark) to 2 (average); the default is 2.
	Surround foundation.Opt[float64]
	// DiscountingIlluminant is whether the eye discounts the illuminant.
	DiscountingIlluminant bool
}

// ViewingConditionsSRGB are the conditions of sRGB: D65, an adapting luminance of
// 11.72 cd/m2 and a 50% gray background. It is read-only.
//
// Flutter: material-color-utilities hct/viewing_conditions.dart ViewingConditions.sRgb
var ViewingConditionsSRGB = NewViewingConditions(ViewingConditionsOpts{})

// ViewingConditionsStandard is ViewingConditionsSRGB.
//
// Flutter: material-color-utilities hct/viewing_conditions.dart ViewingConditions.standard
var ViewingConditionsStandard = ViewingConditionsSRGB

// NewViewingConditions creates viewing conditions from the options, with the
// defaults of Flutter's ViewingConditions.make.
//
// Flutter: material-color-utilities hct/viewing_conditions.dart ViewingConditions.make
func NewViewingConditions(o ViewingConditionsOpts) *ViewingConditions {
	whitePoint := o.WhitePoint.OrElse(WhitePointD65())
	adaptingLuminance := o.AdaptingLuminance.OrElse(-1.0)
	if adaptingLuminance <= 0.0 {
		adaptingLuminance = 200.0 / math.Pi * YFromLstar(50.0) / 100.0
	}
	backgroundLstar := o.BackgroundLstar.OrElse(50.0)
	// A background of pure black is non-physical and leads to infinities.
	backgroundLstar = math.Max(0.1, backgroundLstar)
	surround := o.Surround.OrElse(2.0)
	if foundation.DebugMode && (surround < 0.0 || surround > 2.0) {
		panic(errors.New("ViewingConditionsSurroundInvalid", "the surround must be between 0 and 2", "surround", surround))
	}

	// Transform the test illuminant white in XYZ to 'cone'/'rgb' responses.
	xyz := whitePoint
	rW := xyz[0]*0.401288 + xyz[1]*0.650173 + xyz[2]*-0.051461
	gW := xyz[0]*-0.250268 + xyz[1]*1.204414 + xyz[2]*0.045854
	bW := xyz[0]*-0.002079 + xyz[1]*0.048952 + xyz[2]*0.953127

	// Scale the input surround, domain (0, 2), to the CAM16 surround, (0.8, 1.0).
	f := 0.8 + (surround / 10.0)
	// "Exponential non-linearity"
	var c float64
	if f >= 0.9 {
		c = Lerp(0.59, 0.69, (f-0.9)*10.0)
	} else {
		c = Lerp(0.525, 0.59, (f-0.8)*10.0)
	}
	// The degree of adaptation to the illuminant.
	d := 1.0
	if !o.DiscountingIlluminant {
		d = f * (1.0 - ((1.0 / 3.6) * math.Exp((-adaptingLuminance-42.0)/92.0)))
	}
	// Per Li et al, if D is greater than 1 or less than 0, set it to 1 or 0.
	switch {
	case d > 1.0:
		d = 1.0
	case d < 0.0:
		d = 0.0
	}
	// Chromatic induction factor.
	nc := f

	// Cone responses to the white point, adjusted for discounting. 100.0 is used,
	// not the relative luminance of the white point (see the Dart source).
	rgbD := [3]float64{
		d*(100.0/rW) + 1.0 - d,
		d*(100.0/gW) + 1.0 - d,
		d*(100.0/bW) + 1.0 - d,
	}

	// Factors used in calculating meaningful factors.
	k := 1.0 / (5.0*adaptingLuminance + 1.0)
	k4 := k * k * k * k
	k4F := 1.0 - k4

	// Luminance-level adaptation factor.
	fl := (k4 * adaptingLuminance) + (0.1 * k4F * k4F * math.Pow(5.0*adaptingLuminance, 1.0/3.0))
	// The ratio of the background relative luminance to the white relative luminance.
	n := YFromLstar(backgroundLstar) / whitePoint[1]

	// Base exponential nonlinearity. Schlomer 2018 has a typo and uses 1.58; the
	// correct factor is 1.48.
	z := 1.48 + math.Sqrt(n)

	// Luminance-level induction factors.
	nbb := 0.725 / math.Pow(n, 0.2)
	ncb := nbb

	// Discounted cone responses to the white point, adjusted for post-adaptation
	// perceptual nonlinearities.
	rgbAFactors := [3]float64{
		math.Pow(fl*rgbD[0]*rW/100.0, 0.42),
		math.Pow(fl*rgbD[1]*gW/100.0, 0.42),
		math.Pow(fl*rgbD[2]*bW/100.0, 0.42),
	}
	rgbA := [3]float64{
		(400.0 * rgbAFactors[0]) / (rgbAFactors[0] + 27.13),
		(400.0 * rgbAFactors[1]) / (rgbAFactors[1] + 27.13),
		(400.0 * rgbAFactors[2]) / (rgbAFactors[2] + 27.13),
	}
	aw := (40.0*rgbA[0] + 20.0*rgbA[1] + rgbA[2]) / 20.0 * nbb

	return &ViewingConditions{
		WhitePoint: whitePoint, AdaptingLuminance: adaptingLuminance, BackgroundLstar: backgroundLstar,
		Surround: surround, DiscountingIlluminant: o.DiscountingIlluminant,
		BackgroundYToWhitePointY: n, Aw: aw, Nbb: nbb, Ncb: ncb, C: c, NC: nc,
		RgbD: rgbD, Fl: fl, FLRoot: math.Pow(fl, 0.25), Z: z,
	}
}
