package mcu

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Signum is the signum function: 1 if num > 0, -1 if num < 0, and 0 if num = 0.
//
// Flutter: material-color-utilities utils/math_utils.dart MathUtils.signum
func Signum(num float64) int {
	switch {
	case num < 0:
		return -1
	case num == 0:
		return 0
	}
	return 1
}

// Lerp is the linear interpolation function: start if amount = 0 and stop if
// amount = 1.
//
// Flutter: material-color-utilities utils/math_utils.dart MathUtils.lerp
func Lerp(start, stop, amount float64) float64 {
	return (1.0-amount)*start + amount*stop
}

// ClampInt clamps an integer between two integers.
//
// Flutter: material-color-utilities utils/math_utils.dart MathUtils.clampInt
func ClampInt(lo, hi, input int) int {
	if input < lo {
		return lo
	}
	if input > hi {
		return hi
	}
	return input
}

// ClampDouble clamps a number between two numbers.
//
// Flutter: material-color-utilities utils/math_utils.dart MathUtils.clampDouble
func ClampDouble(lo, hi, input float64) float64 {
	if input < lo {
		return lo
	}
	if input > hi {
		return hi
	}
	return input
}

// SanitizeDegreesInt sanitizes a degree measure as an integer: a degree measure
// between 0 (inclusive) and 360 (exclusive).
//
// Flutter: material-color-utilities utils/math_utils.dart MathUtils.sanitizeDegreesInt
func SanitizeDegreesInt(degrees int) int {
	degrees %= 360
	if degrees < 0 {
		degrees += 360
	}
	return degrees
}

// SanitizeDegreesDouble sanitizes a degree measure as a floating-point number: a
// degree measure between 0.0 (inclusive) and 360.0 (exclusive).
//
// Flutter: material-color-utilities utils/math_utils.dart MathUtils.sanitizeDegreesDouble
func SanitizeDegreesDouble(degrees float64) float64 {
	// Dart's % takes the sign of the divisor, like this.
	degrees = math.Mod(degrees, 360.0)
	if degrees < 0 {
		degrees += 360.0
	}
	return degrees
}

// RotationDirection is the sign of the direction change needed to travel from one
// angle to another: -1 if decreasing from leads to the shortest travel distance,
// 1 if increasing does. For angles 180 degrees apart, 1 is returned.
//
// Flutter: material-color-utilities utils/math_utils.dart MathUtils.rotationDirection
func RotationDirection(from, to float64) float64 {
	if SanitizeDegreesDouble(to-from) <= 180.0 {
		return 1.0
	}
	return -1.0
}

// DifferenceDegrees is the distance of two points on a circle, in degrees.
//
// Flutter: material-color-utilities utils/math_utils.dart MathUtils.differenceDegrees
func DifferenceDegrees(a, b float64) float64 {
	return 180.0 - math.Abs(math.Abs(a-b)-180.0)
}

// MatrixMultiply multiplies a 1x3 row vector with a 3x3 matrix.
//
// Flutter: material-color-utilities utils/math_utils.dart MathUtils.matrixMultiply
func MatrixMultiply(row [3]float64, matrix [3][3]float64) [3]float64 {
	return [3]float64{
		row[0]*matrix[0][0] + row[1]*matrix[0][1] + row[2]*matrix[0][2],
		row[0]*matrix[1][0] + row[1]*matrix[1][1] + row[2]*matrix[1][2],
		row[0]*matrix[2][0] + row[1]*matrix[2][1] + row[2]*matrix[2][2],
	}
}

var (
	srgbToXYZ = [3][3]float64{
		{0.41233895, 0.35762064, 0.18051042},
		{0.2126, 0.7152, 0.0722},
		{0.01932141, 0.11916382, 0.95034478},
	}
	xyzToSRGB = [3][3]float64{
		{3.2413774792388685, -1.5376652402851851, -0.49885366846268053},
		{-0.9691452513005321, 1.8758853451067872, 0.04156585616912061},
		{0.05562093689691305, -0.20395524564742123, 1.0571799111220335},
	}
	whitePointD65 = [3]float64{95.047, 100.0, 108.883}
)

// ArgbFromRgb converts a color from RGB components to ARGB format.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.argbFromRgb
func ArgbFromRgb(red, green, blue int) uint32 {
	return 255<<24 | uint32(red&255)<<16 | uint32(green&255)<<8 | uint32(blue&255)
}

// ArgbFromLinrgb converts a color from linear RGB components to ARGB format.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.argbFromLinrgb
func ArgbFromLinrgb(linrgb [3]float64) uint32 {
	return ArgbFromRgb(Delinearized(linrgb[0]), Delinearized(linrgb[1]), Delinearized(linrgb[2]))
}

// AlphaFromArgb is the alpha component of a color in ARGB format.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.alphaFromArgb
func AlphaFromArgb(argb uint32) int { return int(argb >> 24 & 255) }

// RedFromArgb is the red component of a color in ARGB format.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.redFromArgb
func RedFromArgb(argb uint32) int { return int(argb >> 16 & 255) }

// GreenFromArgb is the green component of a color in ARGB format.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.greenFromArgb
func GreenFromArgb(argb uint32) int { return int(argb >> 8 & 255) }

// BlueFromArgb is the blue component of a color in ARGB format.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.blueFromArgb
func BlueFromArgb(argb uint32) int { return int(argb & 255) }

// IsOpaque reports whether a color in ARGB format is opaque.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.isOpaque
func IsOpaque(argb uint32) bool { return AlphaFromArgb(argb) >= 255 }

// ArgbFromXyz converts a color from XYZ to ARGB.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.argbFromXyz
func ArgbFromXyz(x, y, z float64) uint32 {
	m := xyzToSRGB
	linearR := m[0][0]*x + m[0][1]*y + m[0][2]*z
	linearG := m[1][0]*x + m[1][1]*y + m[1][2]*z
	linearB := m[2][0]*x + m[2][1]*y + m[2][2]*z
	return ArgbFromRgb(Delinearized(linearR), Delinearized(linearG), Delinearized(linearB))
}

// XyzFromArgb converts a color from ARGB to XYZ.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.xyzFromArgb
func XyzFromArgb(argb uint32) [3]float64 {
	r := Linearized(RedFromArgb(argb))
	g := Linearized(GreenFromArgb(argb))
	b := Linearized(BlueFromArgb(argb))
	return MatrixMultiply([3]float64{r, g, b}, srgbToXYZ)
}

// ArgbFromLab converts a color represented in Lab color space into an ARGB value.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.argbFromLab
func ArgbFromLab(l, a, b float64) uint32 {
	fy := (l + 16.0) / 116.0
	fx := a/500.0 + fy
	fz := fy - b/200.0
	return ArgbFromXyz(labInvf(fx)*whitePointD65[0], labInvf(fy)*whitePointD65[1], labInvf(fz)*whitePointD65[2])
}

// LabFromArgb converts a color from ARGB representation to L*a*b*.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.labFromArgb
func LabFromArgb(argb uint32) [3]float64 {
	linearR := Linearized(RedFromArgb(argb))
	linearG := Linearized(GreenFromArgb(argb))
	linearB := Linearized(BlueFromArgb(argb))
	m := srgbToXYZ
	x := m[0][0]*linearR + m[0][1]*linearG + m[0][2]*linearB
	y := m[1][0]*linearR + m[1][1]*linearG + m[1][2]*linearB
	z := m[2][0]*linearR + m[2][1]*linearG + m[2][2]*linearB
	fx := labF(x / whitePointD65[0])
	fy := labF(y / whitePointD65[1])
	fz := labF(z / whitePointD65[2])
	return [3]float64{116.0*fy - 16, 500.0 * (fx - fy), 200.0 * (fy - fz)}
}

// ArgbFromLstar converts an L* value to the ARGB representation of the grayscale
// color with that lightness.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.argbFromLstar
func ArgbFromLstar(lstar float64) uint32 {
	component := Delinearized(YFromLstar(lstar))
	return ArgbFromRgb(component, component, component)
}

// LstarFromArgb computes the L* value of a color in ARGB representation.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.lstarFromArgb
func LstarFromArgb(argb uint32) float64 {
	y := XyzFromArgb(argb)[1]
	return 116.0*labF(y/100.0) - 16.0
}

// YFromLstar converts an L* value to a Y value: both measure luminance, L*
// perceptually and Y relatively.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.yFromLstar
func YFromLstar(lstar float64) float64 {
	return 100.0 * labInvf((lstar+16.0)/116.0)
}

// LstarFromY converts a Y value to an L* value.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.lstarFromY
func LstarFromY(y float64) float64 {
	return labF(y/100.0)*116.0 - 16.0
}

// Linearized linearizes an RGB component: 0 <= rgbComponent <= 255 becomes
// 0.0 <= output <= 100.0.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.linearized
func Linearized(rgbComponent int) float64 {
	normalized := float64(rgbComponent) / 255.0
	if normalized <= 0.040449936 {
		return normalized / 12.92 * 100.0
	}
	return math.Pow((normalized+0.055)/1.055, 2.4) * 100.0
}

// Delinearized delinearizes an RGB component: 0.0 <= rgbComponent <= 100.0
// becomes 0 <= output <= 255.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.delinearized
func Delinearized(rgbComponent float64) int {
	normalized := rgbComponent / 100.0
	var d float64
	if normalized <= 0.0031308 {
		d = normalized * 12.92
	} else {
		d = 1.055*math.Pow(normalized, 1.0/2.4) - 0.055
	}
	return ClampInt(0, 255, int(math.Round(d*255.0)))
}

// WhitePointD65 returns the standard white point; white on a sunny day.
//
// Flutter: material-color-utilities utils/color_utils.dart ColorUtils.whitePointD65
func WhitePointD65() [3]float64 { return whitePointD65 }

func labF(t float64) float64 {
	const e = 216.0 / 24389.0
	const kappa = 24389.0 / 27.0
	if t > e {
		return math.Pow(t, 1.0/3.0)
	}
	return (kappa*t + 16) / 116
}

func labInvf(ft float64) float64 {
	const e = 216.0 / 24389.0
	const kappa = 24389.0 / 27.0
	ft3 := ft * ft * ft
	if ft3 > e {
		return ft3
	}
	return (116*ft - 16) / kappa
}

// HexFromArgb is the hex string of a color, "#RRGGBB" with the hash sign, or
// "RRGGBB" without it.
//
// Flutter: material-color-utilities utils/string_utils.dart StringUtils.hexFromArgb
func HexFromArgb(argb uint32, leadingHashSign bool) string {
	s := fmt.Sprintf("%02X%02X%02X", RedFromArgb(argb), GreenFromArgb(argb), BlueFromArgb(argb))
	if leadingHashSign {
		return "#" + s
	}
	return s
}

// ArgbFromHex parses a hex string, with or without the hash sign; ok is false
// when it is not hexadecimal.
//
// Flutter: material-color-utilities utils/string_utils.dart StringUtils.argbFromHex
func ArgbFromHex(hex string) (argb uint32, ok bool) {
	n, err := strconv.ParseInt(strings.ReplaceAll(hex, "#", ""), 16, 64)
	if err != nil {
		return 0, false
	}
	return uint32(n), true
}
