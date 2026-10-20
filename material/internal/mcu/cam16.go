package mcu

import "math"

// Cam16 is a color in the CAM16 color appearance model: hue, chroma, lightness
// (J), brightness (Q), colorfulness (M), saturation (S), and the coordinates of
// the CAM16-UCS color space (Jstar, Astar, Bstar).
//
// Flutter: material-color-utilities hct/cam16.dart Cam16
type Cam16 struct {
	// Hue is the hue, in degrees.
	Hue float64
	// Chroma is the chroma of the color.
	Chroma float64
	// J is the lightness.
	J float64
	// Q is the brightness.
	Q float64
	// M is the colorfulness.
	M float64
	// S is the saturation.
	S float64
	// Jstar is the lightness in CAM16-UCS.
	Jstar float64
	// Astar is the a* coordinate in CAM16-UCS.
	Astar float64
	// Bstar is the b* coordinate in CAM16-UCS.
	Bstar float64
}

// Distance is the CAM16-UCS distance of two colors.
//
// Flutter: material-color-utilities hct/cam16.dart Cam16.distance
func (c *Cam16) Distance(other *Cam16) float64 {
	dJ := c.Jstar - other.Jstar
	dA := c.Astar - other.Astar
	dB := c.Bstar - other.Bstar
	dEPrime := math.Sqrt(dJ*dJ + dA*dA + dB*dB)
	return 1.41 * math.Pow(dEPrime, 0.63)
}

// Cam16FromInt is the CAM16 color of an ARGB color, in the sRGB viewing
// conditions.
//
// Flutter: material-color-utilities hct/cam16.dart Cam16.fromInt
func Cam16FromInt(argb uint32) *Cam16 {
	return Cam16FromIntInViewingConditions(argb, ViewingConditionsSRGB)
}

// Cam16FromIntInViewingConditions is the CAM16 color of an ARGB color in the
// viewing conditions.
//
// Flutter: material-color-utilities hct/cam16.dart Cam16.fromIntInViewingConditions
func Cam16FromIntInViewingConditions(argb uint32, vc *ViewingConditions) *Cam16 {
	xyz := XyzFromArgb(argb)
	return Cam16FromXyzInViewingConditions(xyz[0], xyz[1], xyz[2], vc)
}

// Cam16FromXyzInViewingConditions is the CAM16 color of XYZ coordinates in the
// viewing conditions.
//
// Flutter: material-color-utilities hct/cam16.dart Cam16.fromXyzInViewingConditions
func Cam16FromXyzInViewingConditions(x, y, z float64, vc *ViewingConditions) *Cam16 {
	// Transform XYZ to 'cone'/'rgb' responses.
	rC := 0.401288*x + 0.650173*y - 0.051461*z
	gC := -0.250268*x + 1.204414*y + 0.045854*z
	bC := -0.002079*x + 0.048952*y + 0.953127*z

	// Discount the illuminant.
	rD := vc.RgbD[0] * rC
	gD := vc.RgbD[1] * gC
	bD := vc.RgbD[2] * bC

	// Chromatic adaptation.
	rAF := math.Pow(vc.Fl*math.Abs(rD)/100.0, 0.42)
	gAF := math.Pow(vc.Fl*math.Abs(gD)/100.0, 0.42)
	bAF := math.Pow(vc.Fl*math.Abs(bD)/100.0, 0.42)
	rA := float64(Signum(rD)) * 400.0 * rAF / (rAF + 27.13)
	gA := float64(Signum(gD)) * 400.0 * gAF / (gAF + 27.13)
	bA := float64(Signum(bD)) * 400.0 * bAF / (bAF + 27.13)

	// Redness-greenness and yellowness-blueness.
	a := (11.0*rA + -12.0*gA + bA) / 11.0
	b := (rA + gA - 2.0*bA) / 9.0

	// Auxiliary components.
	u := (20.0*rA + 20.0*gA + 21.0*bA) / 20.0
	p2 := (40.0*rA + 20.0*gA + bA) / 20.0

	// Hue.
	atanDegrees := math.Atan2(b, a) * 180.0 / math.Pi
	hue := SanitizeDegreesDouble(atanDegrees)
	hueRadians := hue * math.Pi / 180.0

	// Achromatic response to color.
	ac := p2 * vc.Nbb

	// CAM16 lightness and brightness.
	J := 100.0 * math.Pow(ac/vc.Aw, vc.C*vc.Z)
	Q := (4.0 / vc.C) * math.Sqrt(J/100.0) * (vc.Aw + 4.0) * vc.FLRoot

	huePrime := hue
	if hue < 20.14 {
		huePrime = hue + 360
	}
	eHue := (1.0 / 4.0) * (math.Cos(huePrime*math.Pi/180.0+2.0) + 3.8)
	p1 := 50000.0 / 13.0 * eHue * vc.NC * vc.Ncb
	t := p1 * math.Sqrt(a*a+b*b) / (u + 0.305)
	alpha := math.Pow(t, 0.9) * math.Pow(1.64-math.Pow(0.29, vc.BackgroundYToWhitePointY), 0.73)
	// CAM16 chroma, colorfulness and saturation.
	C := alpha * math.Sqrt(J/100.0)
	M := C * vc.FLRoot
	s := 50.0 * math.Sqrt((alpha*vc.C)/(vc.Aw+4.0))

	// CAM16-UCS components.
	jstar := (1.0 + 100.0*0.007) * J / (1.0 + 0.007*J)
	mstar := math.Log(1.0+0.0228*M) / 0.0228
	return &Cam16{
		Hue: hue, Chroma: C, J: J, Q: Q, M: M, S: s,
		Jstar: jstar, Astar: mstar * math.Cos(hueRadians), Bstar: mstar * math.Sin(hueRadians),
	}
}

// Cam16FromJch creates a CAM16 color from lightness, chroma and hue, in the sRGB
// viewing conditions.
//
// Flutter: material-color-utilities hct/cam16.dart Cam16.fromJch
func Cam16FromJch(j, c, h float64) *Cam16 {
	return Cam16FromJchInViewingConditions(j, c, h, ViewingConditionsSRGB)
}

// Cam16FromJchInViewingConditions creates a CAM16 color from lightness, chroma
// and hue in the viewing conditions.
//
// Flutter: material-color-utilities hct/cam16.dart Cam16.fromJchInViewingConditions
func Cam16FromJchInViewingConditions(J, C, h float64, vc *ViewingConditions) *Cam16 {
	Q := (4.0 / vc.C) * math.Sqrt(J/100.0) * (vc.Aw + 4.0) * vc.FLRoot
	M := C * vc.FLRoot
	alpha := C / math.Sqrt(J/100.0)
	s := 50.0 * math.Sqrt((alpha*vc.C)/(vc.Aw+4.0))

	hueRadians := h * math.Pi / 180.0
	jstar := (1.0 + 100.0*0.007) * J / (1.0 + 0.007*J)
	mstar := 1.0 / 0.0228 * math.Log(1.0+0.0228*M)
	return &Cam16{
		Hue: h, Chroma: C, J: J, Q: Q, M: M, S: s,
		Jstar: jstar, Astar: mstar * math.Cos(hueRadians), Bstar: mstar * math.Sin(hueRadians),
	}
}

// Cam16FromUcs creates a CAM16 color from CAM16-UCS coordinates, in the standard
// viewing conditions.
//
// Flutter: material-color-utilities hct/cam16.dart Cam16.fromUcs
func Cam16FromUcs(jstar, astar, bstar float64) *Cam16 {
	return Cam16FromUcsInViewingConditions(jstar, astar, bstar, ViewingConditionsStandard)
}

// Cam16FromUcsInViewingConditions creates a CAM16 color from CAM16-UCS
// coordinates in the viewing conditions.
//
// Flutter: material-color-utilities hct/cam16.dart Cam16.fromUcsInViewingConditions
func Cam16FromUcsInViewingConditions(jstar, astar, bstar float64, vc *ViewingConditions) *Cam16 {
	a, b := astar, bstar
	m := math.Sqrt(a*a + b*b)
	M := (math.Exp(m*0.0228) - 1.0) / 0.0228
	c := M / vc.FLRoot
	h := math.Atan2(b, a) * (180.0 / math.Pi)
	if h < 0.0 {
		h += 360.0
	}
	j := jstar / (1 - (jstar-100)*0.007)
	return Cam16FromJchInViewingConditions(j, c, h, vc)
}

// ToInt is the ARGB color of the CAM16 color in the sRGB viewing conditions.
//
// Flutter: material-color-utilities hct/cam16.dart Cam16.toInt
func (c *Cam16) ToInt() uint32 { return c.Viewed(ViewingConditionsSRGB) }

// Viewed is the ARGB color of the CAM16 color in the viewing conditions.
//
// Flutter: material-color-utilities hct/cam16.dart Cam16.viewed
func (c *Cam16) Viewed(vc *ViewingConditions) uint32 {
	xyz := c.XyzInViewingConditions(vc)
	return ArgbFromXyz(xyz[0], xyz[1], xyz[2])
}

// XyzInViewingConditions is the XYZ color of the CAM16 color in the viewing
// conditions.
//
// Flutter: material-color-utilities hct/cam16.dart Cam16.xyzInViewingConditions
func (c *Cam16) XyzInViewingConditions(vc *ViewingConditions) [3]float64 {
	alpha := 0.0
	if c.Chroma != 0.0 && c.J != 0.0 {
		alpha = c.Chroma / math.Sqrt(c.J/100.0)
	}
	t := math.Pow(alpha/math.Pow(1.64-math.Pow(0.29, vc.BackgroundYToWhitePointY), 0.73), 1.0/0.9)
	hRad := c.Hue * math.Pi / 180.0

	eHue := 0.25 * (math.Cos(hRad+2.0) + 3.8)
	ac := vc.Aw * math.Pow(c.J/100.0, 1.0/vc.C/vc.Z)
	p1 := eHue * (50000.0 / 13.0) * vc.NC * vc.Ncb
	p2 := ac / vc.Nbb

	hSin, hCos := math.Sin(hRad), math.Cos(hRad)

	gamma := 23.0 * (p2 + 0.305) * t / (23.0*p1 + 11*t*hCos + 108.0*t*hSin)
	a := gamma * hCos
	b := gamma * hSin
	rA := (460.0*p2 + 451.0*a + 288.0*b) / 1403.0
	gA := (460.0*p2 - 891.0*a - 261.0*b) / 1403.0
	bA := (460.0*p2 - 220.0*a - 6300.0*b) / 1403.0

	rC := inverseAdaptation(rA, vc.Fl)
	gC := inverseAdaptation(gA, vc.Fl)
	bC := inverseAdaptation(bA, vc.Fl)
	rF := rC / vc.RgbD[0]
	gF := gC / vc.RgbD[1]
	bF := bC / vc.RgbD[2]

	return [3]float64{
		1.86206786*rF - 1.01125463*gF + 0.14918677*bF,
		0.38752654*rF + 0.62144744*gF - 0.00897398*bF,
		-0.01584150*rF - 0.03412294*gF + 1.04996444*bF,
	}
}

// inverseAdaptation undoes the post-adaptation nonlinearity of a cone response.
func inverseAdaptation(adapted, fl float64) float64 {
	base := math.Max(0, (27.13*math.Abs(adapted))/(400.0-math.Abs(adapted)))
	return float64(Signum(adapted)) * (100.0 / fl) * math.Pow(base, 1.0/0.42)
}
