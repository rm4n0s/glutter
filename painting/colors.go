package painting

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

func hueOf(red, green, blue, max, delta float64) float64 {
	var hue float64
	switch max {
	case 0.0:
		hue = 0.0
	case red:
		hue = 60.0 * dartMod((green-blue)/delta, 6)
	case green:
		hue = 60.0 * ((blue-red)/delta + 2)
	case blue:
		hue = 60.0 * ((red-green)/delta + 4)
	}
	if math.IsNaN(hue) {
		return 0.0
	}
	return hue
}

func colorFromHue(alpha, hue, chroma, secondary, match float64) ui.Color {
	var red, green, blue float64
	switch {
	case hue < 60.0:
		red, green, blue = chroma, secondary, 0.0
	case hue < 120.0:
		red, green, blue = secondary, chroma, 0.0
	case hue < 180.0:
		red, green, blue = 0.0, chroma, secondary
	case hue < 240.0:
		red, green, blue = 0.0, secondary, chroma
	case hue < 300.0:
		red, green, blue = secondary, 0.0, chroma
	default:
		red, green, blue = chroma, 0.0, secondary
	}
	return ui.NewColorFromARGB(dartRound(alpha*0xFF), dartRound((red+match)*0xFF), dartRound((green+match)*0xFF), dartRound((blue+match)*0xFF))
}

// dartRound rounds half away from zero, as Dart's num.round does.
func dartRound(v float64) int { return int(math.Round(v)) }

func checkUnit(name string, v float64, hi float64, owner string) {
	if foundation.DebugMode && !(v >= 0.0 && v <= hi) {
		panic(errors.New("ColorComponentOutOfRange", name+" is out of range", "owner", owner, "component", name, "value", v, "max", hi))
	}
}

func colorChannels(c ui.Color) (red, green, blue, max, min, delta, alpha float64) {
	red = float64(c.Red()) / 0xFF
	green = float64(c.Green()) / 0xFF
	blue = float64(c.Blue()) / 0xFF
	max = math.Max(red, math.Max(green, blue))
	min = math.Min(red, math.Min(green, blue))
	delta = max - min
	alpha = float64(c.Alpha()) / 0xFF
	return
}

// HSVColor is a color represented using alpha, hue, saturation, and value.
//
// Flutter: painting/colors.dart HSVColor
type HSVColor struct {
	// Alpha is the alpha channel, from 0.0 (transparent) to 1.0 (opaque).
	Alpha float64
	// Hue is the hue, from 0.0 to 360.0.
	Hue float64
	// Saturation is the saturation, from 0.0 to 1.0.
	Saturation float64
	// Value is the value (brightness), from 0.0 to 1.0.
	Value float64
}

// NewHSVColorFromAHSV creates a color from alpha, hue, saturation and value.
// Components out of range panic in debug builds with the tag
// ColorComponentOutOfRange.
//
// Flutter: painting/colors.dart HSVColor.fromAHSV
func NewHSVColorFromAHSV(alpha, hue, saturation, value float64) HSVColor {
	checkUnit("alpha", alpha, 1.0, "HSVColor")
	checkUnit("hue", hue, 360.0, "HSVColor")
	checkUnit("saturation", saturation, 1.0, "HSVColor")
	checkUnit("value", value, 1.0, "HSVColor")
	return HSVColor{alpha, hue, saturation, value}
}

// NewHSVColorFromColor creates an HSVColor from a Color.
//
// Flutter: painting/colors.dart HSVColor.fromColor
func NewHSVColorFromColor(color ui.Color) HSVColor {
	red, green, blue, max, _, delta, alpha := colorChannels(color)
	hue := hueOf(red, green, blue, max, delta)
	saturation := 0.0
	if max != 0.0 {
		saturation = delta / max
	}
	return NewHSVColorFromAHSV(alpha, hue, saturation, max)
}

// WithAlpha returns a copy of this color with the alpha channel replaced.
//
// Flutter: painting/colors.dart HSVColor.withAlpha
func (c HSVColor) WithAlpha(alpha float64) HSVColor {
	return NewHSVColorFromAHSV(alpha, c.Hue, c.Saturation, c.Value)
}

// WithHue returns a copy of this color with the hue replaced.
//
// Flutter: painting/colors.dart HSVColor.withHue
func (c HSVColor) WithHue(hue float64) HSVColor {
	return NewHSVColorFromAHSV(c.Alpha, hue, c.Saturation, c.Value)
}

// WithSaturation returns a copy of this color with the saturation replaced.
//
// Flutter: painting/colors.dart HSVColor.withSaturation
func (c HSVColor) WithSaturation(saturation float64) HSVColor {
	return NewHSVColorFromAHSV(c.Alpha, c.Hue, saturation, c.Value)
}

// WithValue returns a copy of this color with the value replaced.
//
// Flutter: painting/colors.dart HSVColor.withValue
func (c HSVColor) WithValue(value float64) HSVColor {
	return NewHSVColorFromAHSV(c.Alpha, c.Hue, c.Saturation, value)
}

// ToColor returns the color as an RGB ui.Color.
//
// Flutter: painting/colors.dart HSVColor.toColor
func (c HSVColor) ToColor() ui.Color {
	chroma := c.Saturation * c.Value
	secondary := chroma * (1.0 - math.Abs(dartMod(c.Hue/60.0, 2.0)-1.0))
	match := c.Value - chroma
	return colorFromHue(c.Alpha, c.Hue, chroma, secondary, match)
}

// HSVColorLerp linearly interpolates between two HSVColors. Either may be nil.
//
// Flutter: painting/colors.dart HSVColor.lerp
func HSVColorLerp(a, b *HSVColor, t float64) *HSVColor {
	if a == b {
		return a
	}
	if a == nil {
		r := b.WithAlpha(b.Alpha * t)
		return &r
	}
	if b == nil {
		r := a.WithAlpha(a.Alpha * (1.0 - t))
		return &r
	}
	r := NewHSVColorFromAHSV(
		ui.ClampDouble(ui.LerpDouble(a.Alpha, b.Alpha, t), 0.0, 1.0),
		dartMod(ui.LerpDouble(a.Hue, b.Hue, t), 360.0),
		ui.ClampDouble(ui.LerpDouble(a.Saturation, b.Saturation, t), 0.0, 1.0),
		ui.ClampDouble(ui.LerpDouble(a.Value, b.Value, t), 0.0, 1.0),
	)
	return &r
}

// String describes the color.
//
// Flutter: painting/colors.dart HSVColor.toString
func (c HSVColor) String() string {
	return "HSVColor(" + dartDouble(c.Alpha) + ", " + dartDouble(c.Hue) + ", " + dartDouble(c.Saturation) + ", " + dartDouble(c.Value) + ")"
}

func dartDouble(v float64) string { return dartfmt.Double(v) }

// HSLColor is a color represented using alpha, hue, saturation, and lightness.
//
// Flutter: painting/colors.dart HSLColor
type HSLColor struct {
	// Alpha is the alpha channel, from 0.0 (transparent) to 1.0 (opaque).
	Alpha float64
	// Hue is the hue, from 0.0 to 360.0.
	Hue float64
	// Saturation is the saturation, from 0.0 to 1.0.
	Saturation float64
	// Lightness is the lightness, from 0.0 to 1.0.
	Lightness float64
}

// NewHSLColorFromAHSL creates a color from alpha, hue, saturation and
// lightness. Components out of range panic in debug builds with the tag
// ColorComponentOutOfRange.
//
// Flutter: painting/colors.dart HSLColor.fromAHSL
func NewHSLColorFromAHSL(alpha, hue, saturation, lightness float64) HSLColor {
	checkUnit("alpha", alpha, 1.0, "HSLColor")
	checkUnit("hue", hue, 360.0, "HSLColor")
	checkUnit("saturation", saturation, 1.0, "HSLColor")
	checkUnit("lightness", lightness, 1.0, "HSLColor")
	return HSLColor{alpha, hue, saturation, lightness}
}

// NewHSLColorFromColor creates an HSLColor from a Color.
//
// Flutter: painting/colors.dart HSLColor.fromColor
func NewHSLColorFromColor(color ui.Color) HSLColor {
	red, green, blue, max, min, delta, alpha := colorChannels(color)
	hue := hueOf(red, green, blue, max, delta)
	lightness := (max + min) / 2.0
	saturation := 0.0
	if min != max {
		saturation = ui.ClampDouble(delta/(1.0-math.Abs(2.0*lightness-1.0)), 0.0, 1.0)
	}
	return NewHSLColorFromAHSL(alpha, hue, saturation, lightness)
}

// WithAlpha returns a copy of this color with the alpha channel replaced.
//
// Flutter: painting/colors.dart HSLColor.withAlpha
func (c HSLColor) WithAlpha(alpha float64) HSLColor {
	return NewHSLColorFromAHSL(alpha, c.Hue, c.Saturation, c.Lightness)
}

// WithHue returns a copy of this color with the hue replaced.
//
// Flutter: painting/colors.dart HSLColor.withHue
func (c HSLColor) WithHue(hue float64) HSLColor {
	return NewHSLColorFromAHSL(c.Alpha, hue, c.Saturation, c.Lightness)
}

// WithSaturation returns a copy of this color with the saturation replaced.
//
// Flutter: painting/colors.dart HSLColor.withSaturation
func (c HSLColor) WithSaturation(saturation float64) HSLColor {
	return NewHSLColorFromAHSL(c.Alpha, c.Hue, saturation, c.Lightness)
}

// WithLightness returns a copy of this color with the lightness replaced.
//
// Flutter: painting/colors.dart HSLColor.withLightness
func (c HSLColor) WithLightness(lightness float64) HSLColor {
	return NewHSLColorFromAHSL(c.Alpha, c.Hue, c.Saturation, lightness)
}

// ToColor returns the color as an RGB ui.Color.
//
// Flutter: painting/colors.dart HSLColor.toColor
func (c HSLColor) ToColor() ui.Color {
	chroma := (1.0 - math.Abs(2.0*c.Lightness-1.0)) * c.Saturation
	secondary := chroma * (1.0 - math.Abs(dartMod(c.Hue/60.0, 2.0)-1.0))
	match := c.Lightness - chroma/2.0
	return colorFromHue(c.Alpha, c.Hue, chroma, secondary, match)
}

// HSLColorLerp linearly interpolates between two HSLColors. Either may be nil.
//
// Flutter: painting/colors.dart HSLColor.lerp
func HSLColorLerp(a, b *HSLColor, t float64) *HSLColor {
	if a == b {
		return a
	}
	if a == nil {
		r := b.WithAlpha(b.Alpha * t)
		return &r
	}
	if b == nil {
		r := a.WithAlpha(a.Alpha * (1.0 - t))
		return &r
	}
	r := NewHSLColorFromAHSL(
		ui.ClampDouble(ui.LerpDouble(a.Alpha, b.Alpha, t), 0.0, 1.0),
		dartMod(ui.LerpDouble(a.Hue, b.Hue, t), 360.0),
		ui.ClampDouble(ui.LerpDouble(a.Saturation, b.Saturation, t), 0.0, 1.0),
		ui.ClampDouble(ui.LerpDouble(a.Lightness, b.Lightness, t), 0.0, 1.0),
	)
	return &r
}

// String describes the color.
//
// Flutter: painting/colors.dart HSLColor.toString
func (c HSLColor) String() string {
	return "HSLColor(" + dartDouble(c.Alpha) + ", " + dartDouble(c.Hue) + ", " + dartDouble(c.Saturation) + ", " + dartDouble(c.Lightness) + ")"
}

// ColorSwatch is a color that has a small table of related colors called a
// swatch. Dart's ColorSwatch extends Color; ui.Color is a plain value, so this
// holds the primary color in Primary instead.
//
// Flutter: painting/colors.dart ColorSwatch
type ColorSwatch[T comparable] struct {
	// Primary is the color of the swatch itself.
	Primary ui.Color
	swatch  map[T]ui.Color
	keys    []T
}

// NewColorSwatch creates a swatch. The keys are kept in the given order.
//
// Flutter: painting/colors.dart ColorSwatch.new
func NewColorSwatch[T comparable](primary ui.Color, keys []T, colors map[T]ui.Color) *ColorSwatch[T] {
	m := make(map[T]ui.Color, len(colors))
	for k, c := range colors {
		m[k] = c
	}
	return &ColorSwatch[T]{Primary: primary, swatch: m, keys: append([]T(nil), keys...)}
}

// At returns the color of the key, and whether the swatch has it.
//
// Flutter: painting/colors.dart ColorSwatch.[]
func (s *ColorSwatch[T]) At(key T) (ui.Color, bool) {
	c, ok := s.swatch[key]
	return c, ok
}

// Keys are the keys of the swatch, in order.
//
// Flutter: painting/colors.dart ColorSwatch.keys
func (s *ColorSwatch[T]) Keys() []T { return append([]T(nil), s.keys...) }

// Equal reports whether the other swatch has the same primary color and the
// same table.
//
// Flutter: painting/colors.dart ColorSwatch.operator ==
func (s *ColorSwatch[T]) Equal(o *ColorSwatch[T]) bool {
	if s == o {
		return true
	}
	if s == nil || o == nil || s.Primary != o.Primary || len(s.swatch) != len(o.swatch) {
		return false
	}
	for k, c := range s.swatch {
		if oc, ok := o.swatch[k]; !ok || oc != c {
			return false
		}
	}
	return true
}

// String describes the swatch.
//
// Flutter: painting/colors.dart ColorSwatch.toString
func (s *ColorSwatch[T]) String() string {
	return "ColorSwatch(primary value: " + s.Primary.String() + ")"
}

// ColorSwatchLerp linearly interpolates between two swatches. Either may be nil.
//
// Flutter: painting/colors.dart ColorSwatch.lerp
func ColorSwatchLerp[T comparable](a, b *ColorSwatch[T], t float64) *ColorSwatch[T] {
	if a == b {
		return a
	}
	var keys []T
	m := map[T]ui.Color{}
	switch {
	case b == nil:
		keys = a.keys
		for _, k := range keys {
			m[k] = *ui.ColorLerp(ptr(a.swatch[k]), nil, t)
		}
	case a == nil:
		keys = b.keys
		for _, k := range keys {
			m[k] = *ui.ColorLerp(nil, ptr(b.swatch[k]), t)
		}
	default:
		keys = a.keys
		for _, k := range keys {
			bc, ok := b.swatch[k]
			var bp *ui.Color
			if ok {
				bp = &bc
			}
			m[k] = *ui.ColorLerp(ptr(a.swatch[k]), bp, t)
		}
	}
	var ap, bp *ui.Color
	if a != nil {
		ap = &a.Primary
	}
	if b != nil {
		bp = &b.Primary
	}
	return &ColorSwatch[T]{Primary: *ui.ColorLerp(ap, bp, t), swatch: m, keys: append([]T(nil), keys...)}
}

func ptr[T any](v T) *T { return &v }

// NewColorProperty creates a diagnostics property for a color; None is Dart's
// null.
//
// Flutter: painting/colors.dart ColorProperty
func NewColorProperty(name string, value foundation.Opt[ui.Color], opts ...foundation.DiagnosticsPropertyOpts) *foundation.DiagnosticsProperty {
	return foundation.NewDiagnosticsPropertyOpt(name, value, opts...)
}
