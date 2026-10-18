package soft

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// rgba is a premultiplied color with each channel from 0 to 1.
type rgba = [4]float32

// blend combines the source color s with the destination color d, both
// premultiplied, as the blend mode says. The formulas are Skia's
// (SkRasterPipeline), which Flutter's modes are defined by.
func blend(mode ui.BlendMode, s, d rgba) rgba {
	sa, da := s[3], d[3]
	var o rgba
	switch mode {
	case ui.BlendModeClear:
	case ui.BlendModeSrc:
		o = s
	case ui.BlendModeDst:
		o = d
	case ui.BlendModeDstOver:
		for k := range o {
			o[k] = d[k] + s[k]*(1-da)
		}
	case ui.BlendModeSrcIn:
		for k := range o {
			o[k] = s[k] * da
		}
	case ui.BlendModeDstIn:
		for k := range o {
			o[k] = d[k] * sa
		}
	case ui.BlendModeSrcOut:
		for k := range o {
			o[k] = s[k] * (1 - da)
		}
	case ui.BlendModeDstOut:
		for k := range o {
			o[k] = d[k] * (1 - sa)
		}
	case ui.BlendModeSrcATop:
		for k := range o {
			o[k] = s[k]*da + d[k]*(1-sa)
		}
	case ui.BlendModeDstATop:
		for k := range o {
			o[k] = d[k]*sa + s[k]*(1-da)
		}
	case ui.BlendModeXor:
		for k := range o {
			o[k] = s[k]*(1-da) + d[k]*(1-sa)
		}
	case ui.BlendModePlus:
		for k := range o {
			o[k] = min(s[k]+d[k], 1)
		}
	case ui.BlendModeModulate:
		for k := range o {
			o[k] = s[k] * d[k]
		}
	case ui.BlendModeScreen:
		for k := range o {
			o[k] = s[k] + d[k] - s[k]*d[k]
		}
	case ui.BlendModeHue, ui.BlendModeSaturation, ui.BlendModeColor, ui.BlendModeLuminosity:
		return blendNonSeparable(mode, s, d)
	case ui.BlendModeSrcOver, ui.BlendModeUnspecified:
		for k := range o {
			o[k] = s[k] + d[k]*(1-sa)
		}
	default:
		// The separable modes that are defined per channel, whose alpha is that
		// of source-over.
		for k := 0; k < 3; k++ {
			o[k] = blendChannel(mode, s[k], d[k], sa, da)
		}
		o[3] = sa + da - sa*da
	}
	return o
}

// blendChannel is one channel of the separable modes with their own formula.
func blendChannel(mode ui.BlendMode, s, d, sa, da float32) float32 {
	switch mode {
	case ui.BlendModeOverlay:
		return hardLight(d, s, da, sa)
	case ui.BlendModeHardLight:
		return hardLight(s, d, sa, da)
	case ui.BlendModeDarken:
		return s + d - max(s*da, d*sa)
	case ui.BlendModeLighten:
		return s + d - min(s*da, d*sa)
	case ui.BlendModeColorDodge:
		switch {
		case d == 0:
			return s * (1 - da)
		case s == sa:
			return s + d*(1-sa)
		}
		return sa*min(da, d*sa/(sa-s)) + s*(1-da) + d*(1-sa)
	case ui.BlendModeColorBurn:
		switch {
		case d == da:
			return d + s*(1-da)
		case s == 0:
			return d * (1 - sa)
		}
		return sa*(da-min(da, (da-d)*sa/s)) + s*(1-da) + d*(1-sa)
	case ui.BlendModeSoftLight:
		return softLight(s, d, sa, da)
	case ui.BlendModeDifference:
		return s + d - 2*min(s*da, d*sa)
	case ui.BlendModeExclusion:
		return s + d - 2*s*d
	case ui.BlendModeMultiply:
		return s*(1-da) + d*(1-sa) + s*d
	}
	return s + d*(1-sa)
}

// hardLight is the hard-light formula; overlay is it with the colors swapped, so
// the roles of the source and the destination are the arguments' order.
func hardLight(s, d, sa, da float32) float32 {
	if 2*s <= sa {
		return s*(1-da) + d*(1-sa) + 2*s*d
	}
	return s*(1-da) + d*(1-sa) + sa*da - 2*(da-d)*(sa-s)
}

func softLight(s, d, sa, da float32) float32 {
	var m float32
	if da > 0 {
		m = d / da
	}
	s2 := 2 * s
	m4 := 4 * m
	darkSrc := d * (sa + (s2-sa)*(1-m))
	darkDst := (m4*m4+m4)*(m-1) + 7*m
	liteDst := float32(math.Sqrt(float64(m))) - m
	tail := liteDst
	if 4*d <= da {
		tail = darkDst
	}
	liteSrc := d*sa + da*(s2-sa)*tail
	body := liteSrc
	if s2 <= sa {
		body = darkSrc
	}
	return s*(1-da) + d*(1-sa) + body
}

func lum(r, g, b float32) float32 { return 0.30*r + 0.59*g + 0.11*b }

func sat(r, g, b float32) float32 { return max(r, g, b) - min(r, g, b) }

func setSat(r, g, b *float32, s float32) {
	mn, mx := min(*r, *g, *b), max(*r, *g, *b)
	sat := mx - mn
	scale := func(c float32) float32 {
		if sat == 0 {
			return 0
		}
		return (c - mn) * s / sat
	}
	*r, *g, *b = scale(*r), scale(*g), scale(*b)
}

func setLum(r, g, b *float32, l float32) {
	diff := l - lum(*r, *g, *b)
	*r, *g, *b = *r+diff, *g+diff, *b+diff
}

func clipColor(r, g, b *float32, a float32) {
	mn, mx := min(*r, *g, *b), max(*r, *g, *b)
	l := lum(*r, *g, *b)
	clip := func(c float32) float32 {
		if mn < 0 {
			c = l + (c-l)*l/(l-mn)
		}
		if mx > a {
			c = l + (c-l)*(a-l)/(mx-l)
		}
		return max(c, 0)
	}
	*r, *g, *b = clip(*r), clip(*g), clip(*b)
}

// blendNonSeparable is hue, saturation, color and luminosity: the modes that mix
// the channels.
func blendNonSeparable(mode ui.BlendMode, s, d rgba) rgba {
	sa, da := s[3], d[3]
	var r, g, b float32
	switch mode {
	case ui.BlendModeHue:
		r, g, b = s[0]*sa, s[1]*sa, s[2]*sa
		setSat(&r, &g, &b, sat(d[0], d[1], d[2])*sa)
		setLum(&r, &g, &b, lum(d[0], d[1], d[2])*sa)
	case ui.BlendModeSaturation:
		r, g, b = d[0]*sa, d[1]*sa, d[2]*sa
		setSat(&r, &g, &b, sat(s[0], s[1], s[2])*da)
		setLum(&r, &g, &b, lum(d[0], d[1], d[2])*sa)
	case ui.BlendModeColor:
		r, g, b = s[0]*da, s[1]*da, s[2]*da
		setLum(&r, &g, &b, lum(d[0], d[1], d[2])*sa)
	default: // luminosity
		r, g, b = d[0]*sa, d[1]*sa, d[2]*sa
		setLum(&r, &g, &b, lum(s[0], s[1], s[2])*da)
	}
	clipColor(&r, &g, &b, sa*da)
	return rgba{
		s[0]*(1-da) + d[0]*(1-sa) + r,
		s[1]*(1-da) + d[1]*(1-sa) + g,
		s[2]*(1-da) + d[2]*(1-sa) + b,
		sa + da - sa*da,
	}
}

// filterFunc changes the color a paint makes before it is blended.
type filterFunc func(rgba) rgba

// colorFilterFunc returns the function that applies the color filter, or nil for
// no filter. Matrix and gamma filters work on unpremultiplied colors.
func colorFilterFunc(f *ui.ColorFilter) filterFunc {
	if f == nil {
		return nil
	}
	switch f.Kind() {
	case ui.ColorFilterKindMode:
		c := f.Color()
		a := float32(c.Alpha()) / 255
		fc := rgba{float32(c.Red()) / 255 * a, float32(c.Green()) / 255 * a, float32(c.Blue()) / 255 * a, a}
		mode := f.BlendMode()
		return func(px rgba) rgba { return blend(mode, fc, px) }
	case ui.ColorFilterKindMatrix:
		m := f.Matrix()
		return func(px rgba) rgba {
			c := unpremul(px)
			in := [5]float64{float64(c[0]) * 255, float64(c[1]) * 255, float64(c[2]) * 255, float64(c[3]) * 255, 1}
			var out rgba
			for row := 0; row < 4; row++ {
				v := 0.0
				for col := 0; col < 5; col++ {
					v += m[row*5+col] * in[col]
				}
				out[row] = clamp01(float32(v / 255))
			}
			return premul(out)
		}
	case ui.ColorFilterKindLinearToSrgbGamma:
		return gammaFilter(linearToSrgb)
	case ui.ColorFilterKindSrgbToLinearGamma:
		return gammaFilter(srgbToLinear)
	}
	return nil
}

func gammaFilter(curve func(float64) float64) filterFunc {
	return func(px rgba) rgba {
		c := unpremul(px)
		for k := 0; k < 3; k++ {
			c[k] = float32(curve(float64(c[k])))
		}
		return premul(c)
	}
}

func linearToSrgb(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// invertFunc is the filter of Paint.invertColors: it inverts the red, green and
// blue of the unpremultiplied color.
func invertFunc(px rgba) rgba {
	c := unpremul(px)
	return premul(rgba{1 - c[0], 1 - c[1], 1 - c[2], c[3]})
}

func unpremul(c rgba) rgba {
	if c[3] <= 0 {
		return rgba{}
	}
	return rgba{clamp01(c[0] / c[3]), clamp01(c[1] / c[3]), clamp01(c[2] / c[3]), c[3]}
}

func premul(c rgba) rgba { return rgba{c[0] * c[3], c[1] * c[3], c[2] * c[3], c[3]} }

func clamp01(v float32) float32 { return min(max(v, 0), 1) }

// chainFilters returns the function that applies each of the filters in order,
// skipping the nil ones, or nil if there are none.
func chainFilters(fs ...filterFunc) filterFunc {
	var list []filterFunc
	for _, f := range fs {
		if f != nil {
			list = append(list, f)
		}
	}
	switch len(list) {
	case 0:
		return nil
	case 1:
		return list[0]
	}
	return func(px rgba) rgba {
		for _, f := range list {
			px = f(px)
		}
		return px
	}
}
