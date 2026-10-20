// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

import (
	"fmt"
	"math"
)

// Ptr returns a pointer to v. It is for the options whose Flutter parameter is
// nullable, such as ColorWithValuesOpts.
//
// Flutter: no counterpart
func Ptr[T any](v T) *T { return &v }

// Color is an immutable color value, with float32 alpha, red, green and blue
// components in a color space. In the sRGB space the components are from 0 to
// 1; in the extended sRGB space they can be outside that range.
//
// The zero value is transparent black in sRGB. It can be written as a literal:
// Color{A: 1, R: 1}.
//
// Flutter: ui/painting.dart Color
type Color struct {
	// A is the alpha channel: 0 is transparent and 1 is opaque.
	A float32
	// R is the red channel.
	R float32
	// G is the green channel.
	G float32
	// B is the blue channel.
	B float32
	// ColorSpace is the color space of the components. Its zero value is sRGB.
	ColorSpace ColorSpace
}

// NewColor constructs a color from the 32-bit value 0xAARRGGBB, in sRGB.
//
// Flutter: ui/painting.dart Color
func NewColor(value uint32) Color {
	return NewColorFromARGB(int(value>>24), int(value>>16), int(value>>8), int(value))
}

// ColorFromOpts are the optional arguments of NewColorFrom.
//
// Flutter: ui/painting.dart Color.from
type ColorFromOpts struct {
	// ColorSpace is the color space of the components; the zero value is sRGB.
	ColorSpace ColorSpace
}

// NewColorFrom constructs a color from floating point components.
//
// Flutter: ui/painting.dart Color.from
func NewColorFrom(alpha, red, green, blue float64, opts ...ColorFromOpts) Color {
	o := oneOpts(opts)
	return Color{float32(alpha), float32(red), float32(green), float32(blue), o.ColorSpace}
}

// NewColorFromARGB constructs a color from the lower 8 bits of four integers:
// alpha, red, green and blue, in sRGB.
//
// Flutter: ui/painting.dart Color.fromARGB
func NewColorFromARGB(a, r, g, b int) Color {
	return NewColorFromRGBO(r, g, b, float64(a&0xff)/255)
}

// NewColorFromRGBO constructs a color from the lower 8 bits of three integers
// (red, green and blue) and an opacity from 0 to 1, in sRGB.
//
// Flutter: ui/painting.dart Color.fromRGBO
func NewColorFromRGBO(r, g, b int, opacity float64) Color {
	return Color{
		A: float32(opacity),
		R: float32(float64(r&0xff) / 255),
		G: float32(float64(g&0xff) / 255),
		B: float32(float64(b&0xff) / 255),
	}
}

// floatToInt8 converts a component from 0 to 1 to an integer from 0 to 255.
func floatToInt8(x float32) uint32 {
	v := math.Round(float64(x) * 255.0)
	switch {
	case math.IsNaN(v) || v < 0:
		return 0
	case v > 255:
		return 255
	}
	return uint32(v)
}

// ToARGB32 converts the color to the 32-bit value 0xAARRGGBB, clamping each
// component to 8 bits.
//
// Flutter: ui/painting.dart Color.toARGB32
func (c Color) ToARGB32() uint32 {
	return floatToInt8(c.A)<<24 | floatToInt8(c.R)<<16 | floatToInt8(c.G)<<8 | floatToInt8(c.B)
}

// Alpha is the alpha channel as an integer from 0 to 255.
//
// Flutter: ui/painting.dart Color.alpha
func (c Color) Alpha() int { return int(floatToInt8(c.A)) }

// Red is the red channel as an integer from 0 to 255.
//
// Flutter: ui/painting.dart Color.red
func (c Color) Red() int { return int(floatToInt8(c.R)) }

// Green is the green channel as an integer from 0 to 255.
//
// Flutter: ui/painting.dart Color.green
func (c Color) Green() int { return int(floatToInt8(c.G)) }

// Blue is the blue channel as an integer from 0 to 255.
//
// Flutter: ui/painting.dart Color.blue
func (c Color) Blue() int { return int(floatToInt8(c.B)) }

// Opacity is the alpha channel from 0 to 1, rounded to 8 bits.
//
// Flutter: ui/painting.dart Color.opacity
func (c Color) Opacity() float64 { return float64(c.Alpha()) / 0xFF }

// ColorWithValuesOpts are the arguments of Color.WithValues. A nil field leaves
// that component as it is.
//
// Flutter: ui/painting.dart Color.withValues
type ColorWithValuesOpts struct {
	// Alpha replaces the alpha channel.
	Alpha *float64
	// Red replaces the red channel.
	Red *float64
	// Green replaces the green channel.
	Green *float64
	// Blue replaces the blue channel.
	Blue *float64
	// ColorSpace converts the color to this color space after the components are
	// replaced.
	ColorSpace *ColorSpace
}

// WithValues returns a new color that matches this color with the components
// replaced, and converted to the color space when one is given. Passing more
// than one options value panics.
//
// Flutter: ui/painting.dart Color.withValues
func (c Color) WithValues(opts ...ColorWithValuesOpts) Color {
	o := oneOpts(opts)
	updated := c
	pick := func(v *float64, old float32) float32 {
		if v == nil {
			return old
		}
		return float32(*v)
	}
	updated.A = pick(o.Alpha, c.A)
	updated.R = pick(o.Red, c.R)
	updated.G = pick(o.Green, c.G)
	updated.B = pick(o.Blue, c.B)
	if o.ColorSpace != nil && *o.ColorSpace != c.ColorSpace {
		return colorTransform(c.ColorSpace, *o.ColorSpace)(updated, *o.ColorSpace)
	}
	return updated
}

// WithAlpha returns a new color that matches this color with the alpha channel
// replaced with a (which is from 0 to 255). The other channels are rounded to 8
// bits.
//
// Flutter: ui/painting.dart Color.withAlpha
func (c Color) WithAlpha(a int) Color { return NewColorFromARGB(a, c.Red(), c.Green(), c.Blue()) }

// WithOpacity returns a new color that matches this color with the alpha channel
// replaced with the given opacity (from 0 to 1). It loses precision: prefer
// WithValues.
//
// Flutter: ui/painting.dart Color.withOpacity
func (c Color) WithOpacity(opacity float64) Color {
	return c.WithAlpha(int(math.Round(255.0 * opacity)))
}

// WithRed returns a new color that matches this color with the red channel
// replaced with r (from 0 to 255).
//
// Flutter: ui/painting.dart Color.withRed
func (c Color) WithRed(r int) Color { return NewColorFromARGB(c.Alpha(), r, c.Green(), c.Blue()) }

// WithGreen returns a new color that matches this color with the green channel
// replaced with g (from 0 to 255).
//
// Flutter: ui/painting.dart Color.withGreen
func (c Color) WithGreen(g int) Color { return NewColorFromARGB(c.Alpha(), c.Red(), g, c.Blue()) }

// WithBlue returns a new color that matches this color with the blue channel
// replaced with b (from 0 to 255).
//
// Flutter: ui/painting.dart Color.withBlue
func (c Color) WithBlue(b int) Color { return NewColorFromARGB(c.Alpha(), c.Red(), c.Green(), b) }

// linearizeColorComponent is the inverse of the sRGB transfer function, see
// https://www.w3.org/TR/WCAG20/#relativeluminancedef.
func linearizeColorComponent(component float64) float64 {
	if component <= 0.03928 {
		return component / 12.92
	}
	return math.Pow((component+0.055)/1.055, 2.4)
}

// ComputeLuminance returns the relative luminance of this color, from 0 for
// black to 1 for white. Alpha is ignored.
//
// Flutter: ui/painting.dart Color.computeLuminance
func (c Color) ComputeLuminance() float64 {
	r := linearizeColorComponent(float64(c.R))
	g := linearizeColorComponent(float64(c.G))
	b := linearizeColorComponent(float64(c.B))
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// scaleAlpha multiplies the alpha of x by factor, clamped to 0 to 1.
func scaleAlpha(x Color, factor float64) Color {
	x.A = float32(ClampDouble(float64(x.A)*factor, 0, 1))
	return x
}

// widerColorSpace is Display P3 when either space is, else a.
func widerColorSpace(a, b ColorSpace) ColorSpace {
	if a == ColorSpaceDisplayP3 || b == ColorSpaceDisplayP3 {
		return ColorSpaceDisplayP3
	}
	return a
}

// Lerp linearly interpolates between c and y: c when t is 0, y when t is 1. When
// the colors are in different color spaces, both are converted to the wider
// one. The result's components are clamped to 0 to 1.
//
// Flutter: ui/painting.dart Color.lerp
func (c Color) Lerp(y Color, t float64) Color {
	a, b := c, y
	space := c.ColorSpace
	if c.ColorSpace != y.ColorSpace {
		space = widerColorSpace(c.ColorSpace, y.ColorSpace)
		a = c.WithValues(ColorWithValuesOpts{ColorSpace: &space})
		b = y.WithValues(ColorWithValuesOpts{ColorSpace: &space})
	}
	lerp := func(p, q float32) float32 {
		return float32(ClampDouble(lerpDouble(float64(p), float64(q), t), 0, 1))
	}
	return Color{lerp(a.A, b.A), lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), space}
}

// ColorLerp linearly interpolates between two colors, either of which may be
// nil. A nil color counts as transparent: the other color's alpha is scaled.
// Two nils give nil.
//
// Flutter: ui/painting.dart Color.lerp
func ColorLerp(x, y *Color, t float64) *Color {
	switch {
	case x == nil && y == nil:
		return nil
	case y == nil:
		r := scaleAlpha(*x, 1.0-t)
		return &r
	case x == nil:
		r := scaleAlpha(*y, t)
		return &r
	}
	r := x.Lerp(*y, t)
	return &r
}

// ColorAlphaBlend combines the foreground color as a transparent color over a
// background color, with the same result as drawing the foreground over the
// background. Both colors must be in the same color space.
//
// Flutter: ui/painting.dart Color.alphaBlend
func ColorAlphaBlend(foreground, background Color) Color {
	alpha := float64(foreground.A)
	if alpha == 0 {
		// Foreground completely transparent.
		return background
	}
	invAlpha := 1 - alpha
	backAlpha := float64(background.A)
	fr, fg, fb := float64(foreground.R), float64(foreground.G), float64(foreground.B)
	br, bg, bb := float64(background.R), float64(background.G), float64(background.B)
	if backAlpha == 1 {
		// Opaque background case
		return NewColorFrom(1,
			alpha*fr+invAlpha*br,
			alpha*fg+invAlpha*bg,
			alpha*fb+invAlpha*bb,
			ColorFromOpts{ColorSpace: foreground.ColorSpace})
	}
	// General case
	backAlpha *= invAlpha
	outAlpha := alpha + backAlpha
	return NewColorFrom(outAlpha,
		(fr*alpha+br*backAlpha)/outAlpha,
		(fg*alpha+bg*backAlpha)/outAlpha,
		(fb*alpha+bb*backAlpha)/outAlpha,
		ColorFromOpts{ColorSpace: foreground.ColorSpace})
}

// ColorGetAlphaFromOpacity returns an alpha value from 0 to 255 for an opacity
// from 0 to 1 (clamped).
//
// Flutter: ui/painting.dart Color.getAlphaFromOpacity
func ColorGetAlphaFromOpacity(opacity float64) int {
	return int(math.Round(ClampDouble(opacity, 0.0, 1.0) * 255))
}

// String describes the color, as Dart's toString does.
//
// Flutter: ui/painting.dart Color.toString
func (c Color) String() string {
	return fmt.Sprintf("Color(alpha: %s, red: %s, green: %s, blue: %s, colorSpace: %s)",
		toStringAsFixed(float64(c.A), 4), toStringAsFixed(float64(c.R), 4),
		toStringAsFixed(float64(c.G), 4), toStringAsFixed(float64(c.B), 4), c.ColorSpace)
}

// colorTransformFunc converts a color to another color space.
type colorTransformFunc func(c Color, result ColorSpace) Color

// sRGB standard constants for transfer functions, see
// https://en.wikipedia.org/wiki/SRGB.
const (
	srgbGamma                = 2.4
	srgbLinearThreshold      = 0.04045
	srgbLinearSlope          = 12.92
	srgbEncodedOffset        = 0.055
	srgbEncodedDivisor       = 1.055
	srgbLinearToEncodedLimit = 0.0031308
)

// srgbEOTF is the sRGB electro-optical transfer function (gamma decode to
// linear).
func srgbEOTF(v float64) float64 {
	if v <= srgbLinearThreshold {
		return v / srgbLinearSlope
	}
	return math.Pow((v+srgbEncodedOffset)/srgbEncodedDivisor, srgbGamma)
}

// srgbOETF is the sRGB opto-electronic transfer function (gamma encode from
// linear).
func srgbOETF(v float64) float64 {
	if v <= srgbLinearToEncodedLimit {
		return v * srgbLinearSlope
	}
	return srgbEncodedDivisor*math.Pow(v, 1.0/srgbGamma) - srgbEncodedOffset
}

// srgbEOTFExtended extends srgbEOTF to negative values by symmetry.
func srgbEOTFExtended(v float64) float64 {
	if v < 0.0 {
		return -srgbEOTF(-v)
	}
	return srgbEOTF(v)
}

// srgbOETFExtended extends srgbOETF to negative values by symmetry.
func srgbOETFExtended(v float64) float64 {
	if v < 0.0 {
		return -srgbOETF(-v)
	}
	return srgbOETF(v)
}

// p3ToSrgbLinear and srgbToP3Linear are the 3x3 matrices, row-major, that
// convert linear Display P3 to linear sRGB and back.
var (
	p3ToSrgbLinear = [9]float64{
		1.2249401, -0.2249402, 0.0,
		-0.0420569, 1.0420571, 0.0,
		-0.0196376, -0.0786507, 1.0982884,
	}
	srgbToP3Linear = [9]float64{
		0.8224622, 0.1775380, 0.0,
		0.0331942, 0.9668058, 0.0,
		0.0170806, 0.0723974, 0.9105220,
	}
)

// matrixTransform returns the conversion between sRGB and Display P3 through
// the given linear matrix.
func matrixTransform(m *[9]float64) colorTransformFunc {
	return func(c Color, result ColorSpace) Color {
		rLin := srgbEOTFExtended(float64(c.R))
		gLin := srgbEOTFExtended(float64(c.G))
		bLin := srgbEOTFExtended(float64(c.B))
		rOut := m[0]*rLin + m[1]*gLin + m[2]*bLin
		gOut := m[3]*rLin + m[4]*gLin + m[5]*bLin
		bOut := m[6]*rLin + m[7]*gLin + m[8]*bLin
		return NewColorFrom(float64(c.A), srgbOETFExtended(rOut), srgbOETFExtended(gOut), srgbOETFExtended(bOut),
			ColorFromOpts{ColorSpace: result})
	}
}

// clampTransform clamps the components to 0 to 1, in the result color space.
//
// Flutter's _ClampTransform takes a child transform but does not apply it, so
// converting from Display P3 to sRGB only clamps, without converting the gamut.
// That is ported as it is.
func clampTransform(c Color, result ColorSpace) Color {
	return NewColorFrom(
		ClampDouble(float64(c.A), 0, 1),
		ClampDouble(float64(c.R), 0, 1),
		ClampDouble(float64(c.G), 0, 1),
		ClampDouble(float64(c.B), 0, 1),
		ColorFromOpts{ColorSpace: result})
}

// identityTransform returns the color as it is. Like Flutter's, it does not set
// the result color space: the components are the same in sRGB and extended sRGB.
func identityTransform(c Color, _ ColorSpace) Color { return c }

// colorTransform returns the conversion from the source color space to the
// destination.
func colorTransform(source, destination ColorSpace) colorTransformFunc {
	switch source {
	case ColorSpaceSRGB:
		switch destination {
		case ColorSpaceDisplayP3:
			return matrixTransform(&srgbToP3Linear)
		}
		return identityTransform
	case ColorSpaceExtendedSRGB:
		switch destination {
		case ColorSpaceSRGB, ColorSpaceDisplayP3:
			return clampTransform
		}
		return identityTransform
	case ColorSpaceDisplayP3:
		switch destination {
		case ColorSpaceSRGB:
			return clampTransform
		case ColorSpaceExtendedSRGB:
			return matrixTransform(&p3ToSrgbLinear)
		}
	}
	return identityTransform
}
