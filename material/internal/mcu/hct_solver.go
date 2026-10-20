package mcu

import "math"

var (
	scaledDiscountFromLinrgb = [3][3]float64{
		{0.001200833568784504, 0.002389694492170889, 0.0002795742885861124},
		{0.0005891086651375999, 0.0029785502573438758, 0.0003270666104008398},
		{0.00010146692491640572, 0.0005364214359186694, 0.0032979401770712076},
	}
	linrgbFromScaledDiscount = [3][3]float64{
		{1373.2198709594231, -1100.4251190754821, -7.278681089101213},
		{-271.815969077903, 559.6580465940733, -32.46047482791194},
		{1.9622899599665666, -57.173814538844006, 308.7233197812385},
	}
	yFromLinrgb = [3]float64{0.2126, 0.7152, 0.0722}
)

// dartMod is the remainder of Dart's % on doubles: it has the sign of the divisor.
func dartMod(a, b float64) float64 {
	m := math.Mod(a, b)
	if m < 0 {
		m += b
	}
	return m
}

func sanitizeRadians(angle float64) float64 {
	return dartMod(angle+math.Pi*8, math.Pi*2)
}

func trueDelinearized(rgbComponent float64) float64 {
	normalized := rgbComponent / 100.0
	var d float64
	if normalized <= 0.0031308 {
		d = normalized * 12.92
	} else {
		d = 1.055*math.Pow(normalized, 1.0/2.4) - 0.055
	}
	return d * 255.0
}

func chromaticAdaptation(component float64) float64 {
	af := math.Pow(math.Abs(component), 0.42)
	return float64(Signum(component)) * 400.0 * af / (af + 27.13)
}

// hueOf is the hue of a color given in linear RGB, in radians.
func hueOf(linrgb [3]float64) float64 {
	scaledDiscount := MatrixMultiply(linrgb, scaledDiscountFromLinrgb)
	rA := chromaticAdaptation(scaledDiscount[0])
	gA := chromaticAdaptation(scaledDiscount[1])
	bA := chromaticAdaptation(scaledDiscount[2])
	// Redness-greenness and yellowness-blueness.
	a := (11.0*rA + -12.0*gA + bA) / 11.0
	b := (rA + gA - 2.0*bA) / 9.0
	return math.Atan2(b, a)
}

func areInCyclicOrder(a, b, c float64) bool {
	return sanitizeRadians(b-a) < sanitizeRadians(c-a)
}

// intercept is how far along the way from the source to the target the mid is.
func intercept(source, mid, target float64) float64 {
	return (mid - source) / (target - source)
}

func lerpPoint(source [3]float64, t float64, target [3]float64) [3]float64 {
	return [3]float64{
		source[0] + (target[0]-source[0])*t,
		source[1] + (target[1]-source[1])*t,
		source[2] + (target[2]-source[2])*t,
	}
}

// setCoordinate is the point on the line from the source to the target whose
// coordinate on the axis is the given one.
func setCoordinate(source [3]float64, coordinate float64, target [3]float64, axis int) [3]float64 {
	return lerpPoint(source, intercept(source[axis], coordinate, target[axis]), target)
}

func isBounded(x float64) bool { return 0.0 <= x && x <= 100.0 }

// nthVertex is the nth intersection of the plane of the given Y with an edge of
// the RGB cube, or (-1, -1, -1) when the plane does not cross that edge.
func nthVertex(y float64, n int) [3]float64 {
	kR, kG, kB := yFromLinrgb[0], yFromLinrgb[1], yFromLinrgb[2]
	coordA := 100.0
	if n%4 <= 1 {
		coordA = 0.0
	}
	coordB := 100.0
	if n%2 == 0 {
		coordB = 0.0
	}
	none := [3]float64{-1.0, -1.0, -1.0}
	switch {
	case n < 4:
		g, b := coordA, coordB
		r := (y - g*kG - b*kB) / kR
		if isBounded(r) {
			return [3]float64{r, g, b}
		}
	case n < 8:
		b, r := coordA, coordB
		g := (y - r*kR - b*kB) / kG
		if isBounded(g) {
			return [3]float64{r, g, b}
		}
	default:
		r, g := coordA, coordB
		b := (y - r*kR - g*kG) / kB
		if isBounded(b) {
			return [3]float64{r, g, b}
		}
	}
	return none
}

// bisectToSegment finds the segment of the polygon of the plane of the given Y
// that holds the target hue.
func bisectToSegment(y, targetHue float64) (left, right [3]float64) {
	left = [3]float64{-1.0, -1.0, -1.0}
	right = left
	var leftHue, rightHue float64
	initialized, uncut := false, true
	for n := range 12 {
		mid := nthVertex(y, n)
		if mid[0] < 0 {
			continue
		}
		midHue := hueOf(mid)
		if !initialized {
			left, right = mid, mid
			leftHue, rightHue = midHue, midHue
			initialized = true
			continue
		}
		if uncut || areInCyclicOrder(leftHue, midHue, rightHue) {
			uncut = false
			if areInCyclicOrder(leftHue, targetHue, midHue) {
				right, rightHue = mid, midHue
			} else {
				left, leftHue = mid, midHue
			}
		}
	}
	return left, right
}

func midpoint(a, b [3]float64) [3]float64 {
	return [3]float64{(a[0] + b[0]) / 2, (a[1] + b[1]) / 2, (a[2] + b[2]) / 2}
}

func criticalPlaneBelow(x float64) int { return int(math.Floor(x - 0.5)) }
func criticalPlaneAbove(x float64) int { return int(math.Ceil(x - 0.5)) }

// bisectToLimit finds the color of the given Y and hue that is on the boundary
// of the gamut, by bisecting along the critical planes.
func bisectToLimit(y, targetHue float64) [3]float64 {
	left, right := bisectToSegment(y, targetHue)
	leftHue := hueOf(left)
	for axis := range 3 {
		if left[axis] == right[axis] {
			continue
		}
		var lPlane, rPlane int
		if left[axis] < right[axis] {
			lPlane = criticalPlaneBelow(trueDelinearized(left[axis]))
			rPlane = criticalPlaneAbove(trueDelinearized(right[axis]))
		} else {
			lPlane = criticalPlaneAbove(trueDelinearized(left[axis]))
			rPlane = criticalPlaneBelow(trueDelinearized(right[axis]))
		}
		for range 8 {
			if abs(rPlane-lPlane) <= 1 {
				break
			}
			mPlane := int(math.Floor(float64(lPlane+rPlane) / 2.0))
			mid := setCoordinate(left, criticalPlanes[mPlane], right, axis)
			midHue := hueOf(mid)
			if areInCyclicOrder(leftHue, targetHue, midHue) {
				right, rPlane = mid, mPlane
			} else {
				left, leftHue, lPlane = mid, midHue, mPlane
			}
		}
	}
	return midpoint(left, right)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func inverseChromaticAdaptation(adapted float64) float64 {
	adaptedAbs := math.Abs(adapted)
	base := math.Max(0, 27.13*adaptedAbs/(400.0-adaptedAbs))
	return float64(Signum(adapted)) * math.Pow(base, 1.0/0.42)
}

// findResultByJ finds the color of the hue, chroma and Y with Newton's method on
// the lightness J, or 0 when there is none in the gamut.
func findResultByJ(hueRadians, chroma, y float64) uint32 {
	// Initial estimate of j.
	j := math.Sqrt(y) * 11.0
	// The operations of Cam16 are inlined to avoid repeating the calculation.
	vc := ViewingConditionsStandard
	tInnerCoeff := 1 / math.Pow(1.64-math.Pow(0.29, vc.BackgroundYToWhitePointY), 0.73)
	eHue := 0.25 * (math.Cos(hueRadians+2.0) + 3.8)
	p1 := eHue * (50000.0 / 13.0) * vc.NC * vc.Ncb
	hSin, hCos := math.Sin(hueRadians), math.Cos(hueRadians)
	for iterationRound := range 5 {
		jNormalized := j / 100.0
		alpha := 0.0
		if chroma != 0.0 && j != 0.0 {
			alpha = chroma / math.Sqrt(jNormalized)
		}
		t := math.Pow(alpha*tInnerCoeff, 1.0/0.9)
		ac := vc.Aw * math.Pow(jNormalized, 1.0/vc.C/vc.Z)
		p2 := ac / vc.Nbb
		gamma := 23.0 * (p2 + 0.305) * t / (23.0*p1 + 11*t*hCos + 108.0*t*hSin)
		a := gamma * hCos
		b := gamma * hSin
		rA := (460.0*p2 + 451.0*a + 288.0*b) / 1403.0
		gA := (460.0*p2 - 891.0*a - 261.0*b) / 1403.0
		bA := (460.0*p2 - 220.0*a - 6300.0*b) / 1403.0
		linrgb := MatrixMultiply([3]float64{
			inverseChromaticAdaptation(rA), inverseChromaticAdaptation(gA), inverseChromaticAdaptation(bA),
		}, linrgbFromScaledDiscount)
		if linrgb[0] < 0 || linrgb[1] < 0 || linrgb[2] < 0 {
			return 0
		}
		kR, kG, kB := yFromLinrgb[0], yFromLinrgb[1], yFromLinrgb[2]
		fnj := kR*linrgb[0] + kG*linrgb[1] + kB*linrgb[2]
		if fnj <= 0 {
			return 0
		}
		if iterationRound == 4 || math.Abs(fnj-y) < 0.002 {
			if linrgb[0] > 100.01 || linrgb[1] > 100.01 || linrgb[2] > 100.01 {
				return 0
			}
			return ArgbFromLinrgb(linrgb)
		}
		// Newton's method, with 2 * fn(j) / j as the approximation of fn'(j).
		j = j - (fnj-y)*j/(2*fnj)
	}
	return 0
}

// SolveHctToInt finds an sRGB color with the hue, chroma and tone: the exact
// color when the gamut has it, otherwise the color of the hue and tone with the
// highest chroma that fits.
//
// Flutter: material-color-utilities hct/src/hct_solver.dart HctSolver.solveToInt
func SolveHctToInt(hueDegrees, chroma, lstar float64) uint32 {
	if chroma < 0.0001 || lstar < 0.0001 || lstar > 99.9999 {
		return ArgbFromLstar(lstar)
	}
	hueDegrees = SanitizeDegreesDouble(hueDegrees)
	hueRadians := hueDegrees / 180 * math.Pi
	y := YFromLstar(lstar)
	if exact := findResultByJ(hueRadians, chroma, y); exact != 0 {
		return exact
	}
	return ArgbFromLinrgb(bisectToLimit(y, hueRadians))
}

// SolveHctToCam is the CAM16 color of SolveHctToInt.
//
// Flutter: material-color-utilities hct/src/hct_solver.dart HctSolver.solveToCam
func SolveHctToCam(hueDegrees, chroma, lstar float64) *Cam16 {
	return Cam16FromInt(SolveHctToInt(hueDegrees, chroma, lstar))
}
