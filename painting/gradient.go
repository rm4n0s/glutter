package painting

import (
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

type colorsAndStops struct {
	colors []ui.Color
	stops  []float64
}

// sampleColors returns the color at t, interpolating between the stops.
func sampleColors(colors []ui.Color, stops []float64, t float64) ui.Color {
	if t <= stops[0] {
		return colors[0]
	}
	if t >= stops[len(stops)-1] {
		return colors[len(colors)-1]
	}
	index := -1
	for i := len(stops) - 1; i >= 0; i-- {
		if stops[i] <= t {
			index = i
			break
		}
	}
	return colors[index].Lerp(colors[index+1], (t-stops[index])/(stops[index+1]-stops[index]))
}

func interpolateColorsAndStops(aColors []ui.Color, aStops []float64, bColors []ui.Color, bStops []float64, t float64) colorsAndStops {
	if foundation.DebugMode {
		if len(aColors) < 2 || len(bColors) < 2 || len(aStops) != len(aColors) || len(bStops) != len(bColors) {
			panic(errors.New(tagGradientInterpolationInvalid,
				"both gradients need at least two colors, and as many stops as colors",
				"aColors", len(aColors), "aStops", len(aStops), "bColors", len(bColors), "bStops", len(bStops)))
		}
	}
	stops := append(append([]float64(nil), aStops...), bStops...)
	sort.Float64s(stops)
	stops = slices.Compact(stops)
	colors := make([]ui.Color, len(stops))
	for i, stop := range stops {
		colors[i] = sampleColors(aColors, aStops, stop).Lerp(sampleColors(bColors, bStops, stop), t)
	}
	return colorsAndStops{colors, stops}
}

// GradientTransform is a transform applied to a gradient's shader, such as a
// rotation.
//
// Flutter: painting/gradient.dart GradientTransform
type GradientTransform interface {
	// Transform returns the matrix to apply to the gradient in the bounds, or
	// nil for none.
	Transform(bounds ui.Rect, textDirection ui.TextDirection) *ui.Matrix4
}

// GradientRotation transforms the gradient by rotating it around its center.
//
// Flutter: painting/gradient.dart GradientRotation
type GradientRotation struct {
	// Radians is the angle of rotation, in radians.
	Radians float64
}

// Transform returns the rotation around the center of the bounds.
//
// Flutter: painting/gradient.dart GradientRotation.transform
func (g GradientRotation) Transform(bounds ui.Rect, _ ui.TextDirection) *ui.Matrix4 {
	sin := math.Sin(g.Radians)
	oneMinusCos := 1 - math.Cos(g.Radians)
	center := bounds.Center()
	originX := sin*center.Dy + oneMinusCos*center.Dx
	originY := -sin*center.Dx + oneMinusCos*center.Dy
	m := ui.NewMatrix4Identity()
	m.Translate(float32(originX), float32(originY), 0)
	m.RotateZ(g.Radians)
	return m
}

// String describes the rotation, such as "GradientRotation(radians: 1.6)".
//
// Flutter: painting/gradient.dart GradientRotation.toString
func (g GradientRotation) String() string {
	return "GradientRotation(radians: " + foundation.DebugFormatDouble(foundation.Some(g.Radians)) + ")"
}

// Gradient is a 2D gradient. A nil Gradient is Flutter's null. The methods
// LerpFrom and LerpTo are protected in Flutter; they are exported so that other
// packages can implement a Gradient.
//
// Flutter: painting/gradient.dart Gradient
type Gradient interface {
	// GetColors is the colors the gradient should obtain at each of the stops.
	GetColors() []ui.Color
	// GetStops is the stops that say where each color is obtained; nil spaces
	// the colors evenly.
	GetStops() []float64
	// GetTransform is the transform of the gradient; nil for none.
	GetTransform() GradientTransform
	// CreateShader creates a Shader for the rect.
	CreateShader(rect ui.Rect, textDirection ui.TextDirection) ui.Shader
	// Scale returns a new gradient with its colors scaled by the factor (the
	// alpha channel).
	Scale(factor float64) Gradient
	// WithOpacity returns a new gradient with opacity applied to all its colors.
	WithOpacity(opacity float64) Gradient
	// FromColor returns a gradient of the same kind and geometry with all its
	// colors replaced by color.
	FromColor(color ui.Color) Gradient
	// LerpFrom interpolates from another gradient (possibly nil) to this one;
	// it returns nil if it can't.
	LerpFrom(a Gradient, t float64) Gradient
	// LerpTo interpolates from this gradient to another (possibly nil); it
	// returns nil if it can't.
	LerpTo(b Gradient, t float64) Gradient
	// Equal reports whether the gradients are of the same kind and equal.
	Equal(other Gradient) bool
	String() string
}

// impliedStops are the stops of the gradient, spaced evenly if they are nil.
func impliedStops(colors []ui.Color, stops []float64) []float64 {
	if stops != nil {
		return stops
	}
	if foundation.DebugMode && len(colors) < 2 {
		panic(errors.New(tagGradientColorsTooFew, "colors list must have at least two colors", "colors", len(colors)))
	}
	separation := 1.0 / float64(len(colors)-1)
	out := make([]float64, len(colors))
	for i := range out {
		out[i] = float64(i) * separation
	}
	return out
}

func resolveGradientTransform(g GradientTransform, bounds ui.Rect, td ui.TextDirection) *ui.Matrix4 {
	if g == nil {
		return nil
	}
	return g.Transform(bounds, td)
}

func scaleColors(colors []ui.Color, factor float64) []ui.Color {
	out := make([]ui.Color, len(colors))
	for i, c := range colors {
		out[i] = *ui.ColorLerp(nil, &c, factor)
	}
	return out
}

func opacityColors(colors []ui.Color, opacity float64) []ui.Color {
	out := make([]ui.Color, len(colors))
	for i, c := range colors {
		out[i] = c.WithOpacity(opacity)
	}
	return out
}

func filledColors(n int, c ui.Color) []ui.Color {
	out := make([]ui.Color, n)
	for i := range out {
		out[i] = c
	}
	return out
}

func describeColors(colors []ui.Color) string {
	parts := make([]string, len(colors))
	for i, c := range colors {
		parts[i] = c.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func describeStops(stops []float64) string {
	parts := make([]string, len(stops))
	for i, s := range stops {
		parts[i] = dartfmt.Double(s)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func gradientTransformEqual(a, b GradientTransform) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	x, ok1 := a.(GradientRotation)
	y, ok2 := b.(GradientRotation)
	if ok1 && ok2 {
		return x == y
	}
	return sameValue(a, b)
}

func sameValue(a, b any) (same bool) {
	defer func() {
		if recover() != nil {
			same = false
		}
	}()
	return a == b
}

// GradientLerp linearly interpolates between two gradients. Either may be nil.
// When they are of different kinds, the result fades the first out and the
// second in.
//
// Flutter: painting/gradient.dart Gradient.lerp
func GradientLerp(a, b Gradient, t float64) Gradient {
	if a == nil && b == nil {
		return nil
	}
	if a != nil && b != nil && a.Equal(b) {
		return a
	}
	var result Gradient
	if b != nil {
		result = b.LerpFrom(a, t)
	}
	if result == nil && a != nil {
		result = a.LerpTo(b, t)
	}
	if result != nil {
		return result
	}
	if t < 0.5 {
		return a.Scale(1.0 - t*2.0)
	}
	return b.Scale((t - 0.5) * 2.0)
}

// LinearGradient is a 2D linear gradient.
//
// Flutter: painting/gradient.dart LinearGradient
type LinearGradient struct {
	// Begin is the offset at which stop 0.0 of the gradient is placed.
	Begin AlignmentGeometry
	// End is the offset at which stop 1.0 of the gradient is placed.
	End AlignmentGeometry
	// Colors are the colors the gradient should obtain at each of the stops.
	Colors []ui.Color
	// Stops say where each color is obtained; nil spaces the colors evenly.
	Stops []float64
	// TileMode is how this gradient should tile the plane beyond the region
	// before Begin and after End.
	TileMode ui.TileMode
	// Transform is the transform of the gradient; nil for none.
	Transform GradientTransform
}

// LinearGradientOpts are the optional parameters of NewLinearGradient.
//
// Flutter: painting/gradient.dart LinearGradient.new
type LinearGradientOpts struct {
	// Begin defaults to AlignmentCenterLeft.
	Begin AlignmentGeometry
	// End defaults to AlignmentCenterRight.
	End AlignmentGeometry
	// Stops say where each color is obtained; nil spaces them evenly.
	Stops []float64
	// TileMode defaults to ui.TileModeClamp.
	TileMode ui.TileMode
	// Transform is the transform of the gradient.
	Transform GradientTransform
}

// NewLinearGradient creates a linear gradient.
//
// Flutter: painting/gradient.dart LinearGradient.new
func NewLinearGradient(colors []ui.Color, opts ...LinearGradientOpts) *LinearGradient {
	o := firstOpts(opts)
	g := &LinearGradient{Begin: o.Begin, End: o.End, Colors: colors, Stops: o.Stops, TileMode: o.TileMode, Transform: o.Transform}
	if g.Begin == nil {
		g.Begin = AlignmentCenterLeft
	}
	if g.End == nil {
		g.End = AlignmentCenterRight
	}
	if g.TileMode == ui.TileModeUnspecified {
		g.TileMode = ui.TileModeClamp
	}
	return g
}

// GetColors returns the colors.
//
// Flutter: painting/gradient.dart Gradient.colors
func (g *LinearGradient) GetColors() []ui.Color { return g.Colors }

// GetStops returns the stops.
//
// Flutter: painting/gradient.dart Gradient.stops
func (g *LinearGradient) GetStops() []float64 { return g.Stops }

// GetTransform returns the transform.
//
// Flutter: painting/gradient.dart Gradient.transform
func (g *LinearGradient) GetTransform() GradientTransform { return g.Transform }

// CreateShader creates the shader for the rect.
//
// Flutter: painting/gradient.dart LinearGradient.createShader
func (g *LinearGradient) CreateShader(rect ui.Rect, td ui.TextDirection) ui.Shader {
	return ui.NewGradientLinear(g.Begin.Resolve(td).WithinRect(rect), g.End.Resolve(td).WithinRect(rect), g.Colors,
		ui.GradientLinearOpts{ColorStops: impliedStops(g.Colors, g.Stops), TileMode: g.TileMode,
			Matrix4: resolveGradientTransform(g.Transform, rect, td)})
}

func (g *LinearGradient) with(colors []ui.Color) *LinearGradient {
	c := *g
	c.Colors = colors
	return &c
}

// Scale returns a gradient with its colors scaled by the factor.
//
// Flutter: painting/gradient.dart LinearGradient.scale
func (g *LinearGradient) Scale(factor float64) Gradient { return g.with(scaleColors(g.Colors, factor)) }

// FromColor returns a gradient with all its colors replaced by color.
//
// Flutter: painting/gradient.dart LinearGradient.fromColor
func (g *LinearGradient) FromColor(c ui.Color) Gradient {
	return g.with(filledColors(len(g.Colors), c))
}

// WithOpacity returns a gradient with the opacity applied to its colors.
//
// Flutter: painting/gradient.dart LinearGradient.withOpacity
func (g *LinearGradient) WithOpacity(o float64) Gradient { return g.with(opacityColors(g.Colors, o)) }

// LerpFrom interpolates from a LinearGradient or nil.
//
// Flutter: painting/gradient.dart LinearGradient.lerpFrom
func (g *LinearGradient) LerpFrom(a Gradient, t float64) Gradient {
	switch x := a.(type) {
	case nil:
		return LinearGradientLerp(nil, g, t)
	case *LinearGradient:
		return LinearGradientLerp(x, g, t)
	}
	return gradientDefaultLerpFrom(g, a, t)
}

// LerpTo interpolates to a LinearGradient or nil.
//
// Flutter: painting/gradient.dart LinearGradient.lerpTo
func (g *LinearGradient) LerpTo(b Gradient, t float64) Gradient {
	switch x := b.(type) {
	case nil:
		return LinearGradientLerp(g, nil, t)
	case *LinearGradient:
		return LinearGradientLerp(g, x, t)
	}
	return gradientDefaultLerpTo(g, b, t)
}

func gradientDefaultLerpFrom(self, a Gradient, t float64) Gradient {
	if a == nil {
		return self.Scale(t)
	}
	return nil
}

func gradientDefaultLerpTo(self, b Gradient, t float64) Gradient {
	if b == nil {
		return self.Scale(1.0 - t)
	}
	return nil
}

// LinearGradientLerp linearly interpolates between two LinearGradients. Either
// may be nil. It returns a nil *LinearGradient for two nils.
//
// Flutter: painting/gradient.dart LinearGradient.lerp
func LinearGradientLerp(a, b *LinearGradient, t float64) *LinearGradient {
	if a == nil && b == nil {
		return nil
	}
	if a == b {
		return a
	}
	if a == nil {
		return b.Scale(t).(*LinearGradient)
	}
	if b == nil {
		return a.Scale(1.0 - t).(*LinearGradient)
	}
	in := interpolateColorsAndStops(a.Colors, impliedStops(a.Colors, a.Stops), b.Colors, impliedStops(b.Colors, b.Stops), t)
	r := &LinearGradient{
		Begin: AlignmentGeometryLerp(a.Begin, b.Begin, t), End: AlignmentGeometryLerp(a.End, b.End, t),
		Colors: in.colors, Stops: in.stops, TileMode: a.TileMode, Transform: a.Transform,
	}
	if t >= 0.5 {
		r.TileMode, r.Transform = b.TileMode, b.Transform
	}
	return r
}

// Equal reports whether other is an equal LinearGradient.
//
// Flutter: painting/gradient.dart LinearGradient.operator ==
func (g *LinearGradient) Equal(other Gradient) bool {
	o, ok := other.(*LinearGradient)
	if !ok || o == nil || g == nil {
		return ok && o == g
	}
	return o.Begin.Equal(g.Begin) && o.End.Equal(g.End) && o.TileMode == g.TileMode &&
		gradientTransformEqual(o.Transform, g.Transform) && slices.Equal(o.Colors, g.Colors) && stopsEqual(o.Stops, g.Stops)
}

func stopsEqual(a, b []float64) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return slices.Equal(a, b)
}

// String describes the gradient.
//
// Flutter: painting/gradient.dart LinearGradient.toString
func (g *LinearGradient) String() string {
	d := []string{"begin: " + g.Begin.String(), "end: " + g.End.String(), "colors: " + describeColors(g.Colors)}
	if g.Stops != nil {
		d = append(d, "stops: "+describeStops(g.Stops))
	}
	d = append(d, "tileMode: "+g.TileMode.String())
	if g.Transform != nil {
		d = append(d, "transform: "+describeTransform(g.Transform))
	}
	return "LinearGradient(" + strings.Join(d, ", ") + ")"
}

func describeTransform(t GradientTransform) string {
	if s, ok := t.(interface{ String() string }); ok {
		return s.String()
	}
	return foundation.RuntimeTypeName(t)
}

// RadialGradient is a 2D radial gradient.
//
// Flutter: painting/gradient.dart RadialGradient
type RadialGradient struct {
	// Center is the center of the gradient, as an offset into the unit square.
	Center AlignmentGeometry
	// Radius is the radius of the gradient, as a fraction of the shortest side
	// of the paint box.
	Radius float64
	// Colors are the colors the gradient should obtain at each of the stops.
	Colors []ui.Color
	// Stops say where each color is obtained; nil spaces the colors evenly.
	Stops []float64
	// TileMode is how this gradient should tile the plane beyond the outer
	// ring at Radius.
	TileMode ui.TileMode
	// Focal is the focal point of the gradient; nil for none.
	Focal AlignmentGeometry
	// FocalRadius is the radius of the focal point, as a fraction of the
	// shortest side of the paint box.
	FocalRadius float64
	// Transform is the transform of the gradient; nil for none.
	Transform GradientTransform
}

// RadialGradientOpts are the optional parameters of NewRadialGradient.
//
// Flutter: painting/gradient.dart RadialGradient.new
type RadialGradientOpts struct {
	// Center defaults to AlignmentCenter.
	Center AlignmentGeometry
	// Radius defaults to 0.5.
	Radius foundation.Opt[float64]
	// Stops say where each color is obtained; nil spaces them evenly.
	Stops []float64
	// TileMode defaults to ui.TileModeClamp.
	TileMode ui.TileMode
	// Focal is the focal point; nil for none.
	Focal AlignmentGeometry
	// FocalRadius defaults to 0.
	FocalRadius float64
	// Transform is the transform of the gradient.
	Transform GradientTransform
}

// NewRadialGradient creates a radial gradient.
//
// Flutter: painting/gradient.dart RadialGradient.new
func NewRadialGradient(colors []ui.Color, opts ...RadialGradientOpts) *RadialGradient {
	o := firstOpts(opts)
	g := &RadialGradient{Center: o.Center, Radius: o.Radius.OrElse(0.5), Colors: colors, Stops: o.Stops, TileMode: o.TileMode,
		Focal: o.Focal, FocalRadius: o.FocalRadius, Transform: o.Transform}
	if g.Center == nil {
		g.Center = AlignmentCenter
	}
	if g.TileMode == ui.TileModeUnspecified {
		g.TileMode = ui.TileModeClamp
	}
	return g
}

// GetColors returns the colors.
//
// Flutter: painting/gradient.dart Gradient.colors
func (g *RadialGradient) GetColors() []ui.Color { return g.Colors }

// GetStops returns the stops.
//
// Flutter: painting/gradient.dart Gradient.stops
func (g *RadialGradient) GetStops() []float64 { return g.Stops }

// GetTransform returns the transform.
//
// Flutter: painting/gradient.dart Gradient.transform
func (g *RadialGradient) GetTransform() GradientTransform { return g.Transform }

// CreateShader creates the shader for the rect.
//
// Flutter: painting/gradient.dart RadialGradient.createShader
func (g *RadialGradient) CreateShader(rect ui.Rect, td ui.TextDirection) ui.Shader {
	o := ui.GradientRadialOpts{ColorStops: impliedStops(g.Colors, g.Stops), TileMode: g.TileMode,
		Matrix4: resolveGradientTransform(g.Transform, rect, td), FocalRadius: g.FocalRadius * rect.ShortestSide()}
	if g.Focal != nil {
		f := g.Focal.Resolve(td).WithinRect(rect)
		o.Focal = &f
	}
	return ui.NewGradientRadial(g.Center.Resolve(td).WithinRect(rect), g.Radius*rect.ShortestSide(), g.Colors, o)
}

func (g *RadialGradient) with(colors []ui.Color) *RadialGradient {
	c := *g
	c.Colors = colors
	return &c
}

// Scale returns a gradient with its colors scaled by the factor.
//
// Flutter: painting/gradient.dart RadialGradient.scale
func (g *RadialGradient) Scale(factor float64) Gradient { return g.with(scaleColors(g.Colors, factor)) }

// FromColor returns a gradient with all its colors replaced by color.
//
// Flutter: painting/gradient.dart RadialGradient.fromColor
func (g *RadialGradient) FromColor(c ui.Color) Gradient {
	return g.with(filledColors(len(g.Colors), c))
}

// WithOpacity returns a gradient with the opacity applied to its colors.
//
// Flutter: painting/gradient.dart RadialGradient.withOpacity
func (g *RadialGradient) WithOpacity(o float64) Gradient { return g.with(opacityColors(g.Colors, o)) }

// LerpFrom interpolates from a RadialGradient or nil.
//
// Flutter: painting/gradient.dart RadialGradient.lerpFrom
func (g *RadialGradient) LerpFrom(a Gradient, t float64) Gradient {
	switch x := a.(type) {
	case nil:
		return RadialGradientLerp(nil, g, t)
	case *RadialGradient:
		return RadialGradientLerp(x, g, t)
	}
	return gradientDefaultLerpFrom(g, a, t)
}

// LerpTo interpolates to a RadialGradient or nil.
//
// Flutter: painting/gradient.dart RadialGradient.lerpTo
func (g *RadialGradient) LerpTo(b Gradient, t float64) Gradient {
	switch x := b.(type) {
	case nil:
		return RadialGradientLerp(g, nil, t)
	case *RadialGradient:
		return RadialGradientLerp(g, x, t)
	}
	return gradientDefaultLerpTo(g, b, t)
}

// RadialGradientLerp linearly interpolates between two RadialGradients. Either
// may be nil.
//
// Flutter: painting/gradient.dart RadialGradient.lerp
func RadialGradientLerp(a, b *RadialGradient, t float64) *RadialGradient {
	if a == nil && b == nil {
		return nil
	}
	if a == b {
		return a
	}
	if a == nil {
		return b.Scale(t).(*RadialGradient)
	}
	if b == nil {
		return a.Scale(1.0 - t).(*RadialGradient)
	}
	in := interpolateColorsAndStops(a.Colors, impliedStops(a.Colors, a.Stops), b.Colors, impliedStops(b.Colors, b.Stops), t)
	r := &RadialGradient{
		Center: AlignmentGeometryLerp(a.Center, b.Center, t), Radius: math.Max(0, ui.LerpDouble(a.Radius, b.Radius, t)),
		Colors: in.colors, Stops: in.stops, TileMode: a.TileMode,
		Focal: AlignmentGeometryLerp(a.Focal, b.Focal, t), FocalRadius: math.Max(0, ui.LerpDouble(a.FocalRadius, b.FocalRadius, t)),
		Transform: a.Transform,
	}
	if t >= 0.5 {
		r.TileMode, r.Transform = b.TileMode, b.Transform
	}
	return r
}

// Equal reports whether other is an equal RadialGradient.
//
// Flutter: painting/gradient.dart RadialGradient.operator ==
func (g *RadialGradient) Equal(other Gradient) bool {
	o, ok := other.(*RadialGradient)
	if !ok || o == nil || g == nil {
		return ok && o == g
	}
	return o.Center.Equal(g.Center) && o.Radius == g.Radius && o.TileMode == g.TileMode &&
		gradientTransformEqual(o.Transform, g.Transform) && slices.Equal(o.Colors, g.Colors) && stopsEqual(o.Stops, g.Stops) &&
		alignmentEqualOrNil(o.Focal, g.Focal) && o.FocalRadius == g.FocalRadius
}

func alignmentEqualOrNil(a, b AlignmentGeometry) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

// String describes the gradient.
//
// Flutter: painting/gradient.dart RadialGradient.toString
func (g *RadialGradient) String() string {
	d := []string{"center: " + g.Center.String(), "radius: " + foundation.DebugFormatDouble(foundation.Some(g.Radius)),
		"colors: " + describeColors(g.Colors)}
	if g.Stops != nil {
		d = append(d, "stops: "+describeStops(g.Stops))
	}
	d = append(d, "tileMode: "+g.TileMode.String())
	if g.Focal != nil {
		d = append(d, "focal: "+g.Focal.String())
	}
	d = append(d, "focalRadius: "+foundation.DebugFormatDouble(foundation.Some(g.FocalRadius)))
	if g.Transform != nil {
		d = append(d, "transform: "+describeTransform(g.Transform))
	}
	return "RadialGradient(" + strings.Join(d, ", ") + ")"
}

// SweepGradient is a 2D sweep gradient.
//
// Flutter: painting/gradient.dart SweepGradient
type SweepGradient struct {
	// Center is the center of the gradient, as an offset into the unit square.
	Center AlignmentGeometry
	// StartAngle is the angle in radians at which stop 0.0 of the gradient is
	// placed.
	StartAngle float64
	// EndAngle is the angle in radians at which stop 1.0 of the gradient is
	// placed.
	EndAngle float64
	// Colors are the colors the gradient should obtain at each of the stops.
	Colors []ui.Color
	// Stops say where each color is obtained; nil spaces the colors evenly.
	Stops []float64
	// TileMode is how this gradient should tile the plane beyond the angles.
	TileMode ui.TileMode
	// Transform is the transform of the gradient; nil for none.
	Transform GradientTransform
}

// SweepGradientOpts are the optional parameters of NewSweepGradient.
//
// Flutter: painting/gradient.dart SweepGradient.new
type SweepGradientOpts struct {
	// Center defaults to AlignmentCenter.
	Center AlignmentGeometry
	// StartAngle defaults to 0.
	StartAngle float64
	// EndAngle defaults to 2 pi.
	EndAngle foundation.Opt[float64]
	// Stops say where each color is obtained; nil spaces them evenly.
	Stops []float64
	// TileMode defaults to ui.TileModeClamp.
	TileMode ui.TileMode
	// Transform is the transform of the gradient.
	Transform GradientTransform
}

// NewSweepGradient creates a sweep gradient.
//
// Flutter: painting/gradient.dart SweepGradient.new
func NewSweepGradient(colors []ui.Color, opts ...SweepGradientOpts) *SweepGradient {
	o := firstOpts(opts)
	g := &SweepGradient{Center: o.Center, StartAngle: o.StartAngle, EndAngle: o.EndAngle.OrElse(math.Pi * 2), Colors: colors,
		Stops: o.Stops, TileMode: o.TileMode, Transform: o.Transform}
	if g.Center == nil {
		g.Center = AlignmentCenter
	}
	if g.TileMode == ui.TileModeUnspecified {
		g.TileMode = ui.TileModeClamp
	}
	return g
}

// GetColors returns the colors.
//
// Flutter: painting/gradient.dart Gradient.colors
func (g *SweepGradient) GetColors() []ui.Color { return g.Colors }

// GetStops returns the stops.
//
// Flutter: painting/gradient.dart Gradient.stops
func (g *SweepGradient) GetStops() []float64 { return g.Stops }

// GetTransform returns the transform.
//
// Flutter: painting/gradient.dart Gradient.transform
func (g *SweepGradient) GetTransform() GradientTransform { return g.Transform }

// CreateShader creates the shader for the rect.
//
// Flutter: painting/gradient.dart SweepGradient.createShader
func (g *SweepGradient) CreateShader(rect ui.Rect, td ui.TextDirection) ui.Shader {
	return ui.NewGradientSweep(g.Center.Resolve(td).WithinRect(rect), g.Colors, ui.GradientSweepOpts{
		ColorStops: impliedStops(g.Colors, g.Stops), TileMode: g.TileMode, StartAngle: g.StartAngle, EndAngle: g.EndAngle,
		Matrix4: resolveGradientTransform(g.Transform, rect, td)})
}

func (g *SweepGradient) with(colors []ui.Color) *SweepGradient {
	c := *g
	c.Colors = colors
	return &c
}

// Scale returns a gradient with its colors scaled by the factor.
//
// Flutter: painting/gradient.dart SweepGradient.scale
func (g *SweepGradient) Scale(factor float64) Gradient { return g.with(scaleColors(g.Colors, factor)) }

// FromColor returns a gradient with all its colors replaced by color.
//
// Flutter: painting/gradient.dart SweepGradient.fromColor
func (g *SweepGradient) FromColor(c ui.Color) Gradient { return g.with(filledColors(len(g.Colors), c)) }

// WithOpacity returns a gradient with the opacity applied to its colors.
//
// Flutter: painting/gradient.dart SweepGradient.withOpacity
func (g *SweepGradient) WithOpacity(o float64) Gradient { return g.with(opacityColors(g.Colors, o)) }

// LerpFrom interpolates from a SweepGradient or nil.
//
// Flutter: painting/gradient.dart SweepGradient.lerpFrom
func (g *SweepGradient) LerpFrom(a Gradient, t float64) Gradient {
	switch x := a.(type) {
	case nil:
		return SweepGradientLerp(nil, g, t)
	case *SweepGradient:
		return SweepGradientLerp(x, g, t)
	}
	return gradientDefaultLerpFrom(g, a, t)
}

// LerpTo interpolates to a SweepGradient or nil.
//
// Flutter: painting/gradient.dart SweepGradient.lerpTo
func (g *SweepGradient) LerpTo(b Gradient, t float64) Gradient {
	switch x := b.(type) {
	case nil:
		return SweepGradientLerp(g, nil, t)
	case *SweepGradient:
		return SweepGradientLerp(g, x, t)
	}
	return gradientDefaultLerpTo(g, b, t)
}

// SweepGradientLerp linearly interpolates between two SweepGradients. Either may
// be nil.
//
// Flutter: painting/gradient.dart SweepGradient.lerp
func SweepGradientLerp(a, b *SweepGradient, t float64) *SweepGradient {
	if a == nil && b == nil {
		return nil
	}
	if a == b {
		return a
	}
	if a == nil {
		return b.Scale(t).(*SweepGradient)
	}
	if b == nil {
		return a.Scale(1.0 - t).(*SweepGradient)
	}
	in := interpolateColorsAndStops(a.Colors, impliedStops(a.Colors, a.Stops), b.Colors, impliedStops(b.Colors, b.Stops), t)
	r := &SweepGradient{
		Center:     AlignmentGeometryLerp(a.Center, b.Center, t),
		StartAngle: math.Max(0, ui.LerpDouble(a.StartAngle, b.StartAngle, t)), EndAngle: math.Max(0, ui.LerpDouble(a.EndAngle, b.EndAngle, t)),
		Colors: in.colors, Stops: in.stops, TileMode: a.TileMode, Transform: a.Transform,
	}
	if t >= 0.5 {
		r.TileMode, r.Transform = b.TileMode, b.Transform
	}
	return r
}

// Equal reports whether other is an equal SweepGradient.
//
// Flutter: painting/gradient.dart SweepGradient.operator ==
func (g *SweepGradient) Equal(other Gradient) bool {
	o, ok := other.(*SweepGradient)
	if !ok || o == nil || g == nil {
		return ok && o == g
	}
	return o.Center.Equal(g.Center) && o.StartAngle == g.StartAngle && o.EndAngle == g.EndAngle && o.TileMode == g.TileMode &&
		gradientTransformEqual(o.Transform, g.Transform) && slices.Equal(o.Colors, g.Colors) && stopsEqual(o.Stops, g.Stops)
}

// String describes the gradient.
//
// Flutter: painting/gradient.dart SweepGradient.toString
func (g *SweepGradient) String() string {
	d := []string{"center: " + g.Center.String(), "startAngle: " + foundation.DebugFormatDouble(foundation.Some(g.StartAngle)),
		"endAngle: " + foundation.DebugFormatDouble(foundation.Some(g.EndAngle)), "colors: " + describeColors(g.Colors)}
	if g.Stops != nil {
		d = append(d, "stops: "+describeStops(g.Stops))
	}
	d = append(d, "tileMode: "+g.TileMode.String())
	if g.Transform != nil {
		d = append(d, "transform: "+describeTransform(g.Transform))
	}
	return "SweepGradient(" + strings.Join(d, ", ") + ")"
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGradientColorsTooFew         = "GradientColorsTooFew"
	tagGradientInterpolationInvalid = "GradientInterpolationInvalid"
)
