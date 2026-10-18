package ui

import (
	"math"

	"github.com/rm4n0s/errors"
)

// Shader is the base of the objects that supply a color for each pixel of a
// shape when it is painted: gradients and image shaders. Set one on a Paint
// with Paint.SetShader.
//
// Flutter: ui/painting.dart Shader
type Shader interface {
	shader()
}

// GradientKind says what shape of gradient a Gradient is.
//
// Flutter: ui/painting.dart Gradient
type GradientKind int

// The kinds of Gradient.
const (
	// GradientKindUnspecified is the zero value; no gradient has it.
	//
	// Flutter: no counterpart
	GradientKindUnspecified GradientKind = iota
	// GradientKindLinear changes along the line between two points.
	//
	// Flutter: ui/painting.dart Gradient.linear
	GradientKindLinear
	// GradientKindRadial changes with the distance from a center.
	//
	// Flutter: ui/painting.dart Gradient.radial
	GradientKindRadial
	// GradientKindConical is a two-point conical gradient: a radial one with a
	// focal point and radius.
	//
	// Flutter: ui/painting.dart Gradient.radial
	GradientKindConical
	// GradientKindSweep changes with the angle around a center.
	//
	// Flutter: ui/painting.dart Gradient.sweep
	GradientKindSweep
)

// Gradient is a shader that renders a color gradient: linear, radial (with an
// optional focal point, which makes a two-point conical gradient) or sweep. It
// is immutable.
//
// Flutter: ui/painting.dart Gradient
type Gradient struct {
	kind        GradientKind
	from, to    Offset  // linear
	center      Offset  // radial, conical and sweep
	radius      float64 // radial and conical
	focal       Offset  // conical
	focalRadius float64 // conical
	startAngle  float64 // sweep
	endAngle    float64 // sweep
	colors      []Color
	colorStops  []float64
	tileMode    TileMode
	matrix      [16]float32
	hasMatrix   bool
}

func (*Gradient) shader() {}

// GradientLinearOpts are the optional arguments of NewGradientLinear.
//
// Flutter: ui/painting.dart Gradient.linear
type GradientLinearOpts struct {
	// ColorStops are the positions, from 0 to 1, of the colors along the line.
	// When nil there must be exactly two colors, at 0 and 1.
	ColorStops []float64
	// TileMode is what happens outside the line; the zero value is
	// TileModeClamp.
	TileMode TileMode
	// Matrix4 transforms the gradient; nil means none.
	Matrix4 *Matrix4
}

// GradientRadialOpts are the optional arguments of NewGradientRadial.
//
// Flutter: ui/painting.dart Gradient.radial
type GradientRadialOpts struct {
	// ColorStops are the positions, from 0 to 1, of the colors along the radius.
	// When nil there must be exactly two colors, at 0 and 1.
	ColorStops []float64
	// TileMode is what happens outside the radius; the zero value is
	// TileModeClamp.
	TileMode TileMode
	// Matrix4 transforms the gradient; nil means none.
	Matrix4 *Matrix4
	// Focal is the focal point of a two-point conical gradient; nil means a
	// plain radial gradient.
	Focal *Offset
	// FocalRadius is the radius of the focal circle.
	FocalRadius float64
}

// GradientSweepOpts are the optional arguments of NewGradientSweep.
//
// Flutter: ui/painting.dart Gradient.sweep
type GradientSweepOpts struct {
	// ColorStops are the positions, from 0 to 1, of the colors along the sweep.
	// When nil there must be exactly two colors, at 0 and 1.
	ColorStops []float64
	// TileMode is what happens outside the angles; the zero value is
	// TileModeClamp.
	TileMode TileMode
	// StartAngle is where the sweep starts, in radians clockwise from the
	// positive x axis.
	StartAngle float64
	// EndAngle is where the sweep ends. Flutter's default is 2*pi, which is also
	// used here when EndAngle is 0 and StartAngle is not negative (an end of 0
	// is only meaningful after a negative start).
	EndAngle float64
	// Matrix4 transforms the gradient; nil means none.
	Matrix4 *Matrix4
}

// validateColorStops panics with GradientColorCountInvalid or
// GradientStopCountMismatch when the lists do not fit together.
func validateColorStops(colors []Color, colorStops []float64) {
	if colorStops == nil {
		if len(colors) != 2 {
			panic(errors.New("GradientColorCountInvalid", `"colors" must have length 2 if "colorStops" is omitted.`,
				"colors", len(colors)))
		}
		return
	}
	if len(colors) != len(colorStops) {
		panic(errors.New("GradientStopCountMismatch", `"colors" and "colorStops" arguments must have equal length.`,
			"colors", len(colors), "colorStops", len(colorStops)))
	}
}

// wideColors returns the colors in extended sRGB, which is what the renderers
// work in, as a new list.
func wideColors(colors []Color) []Color {
	space := ColorSpaceExtendedSRGB
	out := make([]Color, len(colors))
	for i, c := range colors {
		out[i] = c.WithValues(ColorWithValuesOpts{ColorSpace: &space})
	}
	return out
}

func newGradient(kind GradientKind, colors []Color, colorStops []float64, tileMode TileMode, matrix *Matrix4) *Gradient {
	validateColorStops(colors, colorStops)
	if tileMode == TileModeUnspecified {
		tileMode = TileModeClamp
	}
	g := &Gradient{kind: kind, colors: wideColors(colors), tileMode: tileMode}
	if colorStops != nil {
		g.colorStops = append([]float64(nil), colorStops...)
	}
	if matrix != nil {
		g.matrix, g.hasMatrix = matrix.storage, true
	}
	return g
}

// NewGradientLinear creates a linear gradient from one point to another. The
// colors are given in a list, and the optional color stops say where each is.
// Passing a list of colors whose length does not fit the stops panics with
// GradientColorCountInvalid or GradientStopCountMismatch.
//
// Flutter: ui/painting.dart Gradient.linear
func NewGradientLinear(from, to Offset, colors []Color, opts ...GradientLinearOpts) *Gradient {
	o := oneOpts(opts)
	g := newGradient(GradientKindLinear, colors, o.ColorStops, o.TileMode, o.Matrix4)
	g.from, g.to = from, to
	return g
}

// NewGradientRadial creates a radial gradient centered at center that ends at
// radius distance from the center. With a Focal point (other than the center, or
// with a focal radius) it is a two-point conical gradient. Bad color lists panic
// as in NewGradientLinear.
//
// Flutter: ui/painting.dart Gradient.radial
func NewGradientRadial(center Offset, radius float64, colors []Color, opts ...GradientRadialOpts) *Gradient {
	o := oneOpts(opts)
	// If focal is nil, this is a regular radial gradient, and so it is when the
	// focal point is the center and the focal radius is 0.
	if o.Focal == nil || (*o.Focal == center && o.FocalRadius == 0.0) {
		g := newGradient(GradientKindRadial, colors, o.ColorStops, o.TileMode, o.Matrix4)
		g.center, g.radius = center, radius
		return g
	}
	g := newGradient(GradientKindConical, colors, o.ColorStops, o.TileMode, o.Matrix4)
	g.center, g.radius = center, radius
	g.focal, g.focalRadius = *o.Focal, o.FocalRadius
	return g
}

// NewGradientSweep creates a sweep gradient centered at center that starts at
// StartAngle and ends at EndAngle. Bad color lists panic as in
// NewGradientLinear.
//
// Flutter: ui/painting.dart Gradient.sweep
func NewGradientSweep(center Offset, colors []Color, opts ...GradientSweepOpts) *Gradient {
	o := oneOpts(opts)
	g := newGradient(GradientKindSweep, colors, o.ColorStops, o.TileMode, o.Matrix4)
	g.center = center
	g.startAngle, g.endAngle = o.StartAngle, o.EndAngle
	if o.EndAngle == 0 && o.StartAngle >= 0 {
		g.endAngle = math.Pi * 2
	}
	return g
}

// Kind says which shape of gradient this is.
//
// Flutter: ui/painting.dart Gradient
func (g *Gradient) Kind() GradientKind { return g.kind }

// From is the start point of a linear gradient.
//
// Flutter: ui/painting.dart Gradient.linear
func (g *Gradient) From() Offset { return g.from }

// To is the end point of a linear gradient.
//
// Flutter: ui/painting.dart Gradient.linear
func (g *Gradient) To() Offset { return g.to }

// Center is the center of a radial, conical or sweep gradient.
//
// Flutter: ui/painting.dart Gradient.radial
func (g *Gradient) Center() Offset { return g.center }

// Radius is the radius of a radial or conical gradient.
//
// Flutter: ui/painting.dart Gradient.radial
func (g *Gradient) Radius() float64 { return g.radius }

// Focal is the focal point of a conical gradient.
//
// Flutter: ui/painting.dart Gradient.radial
func (g *Gradient) Focal() Offset { return g.focal }

// FocalRadius is the radius of the focal circle of a conical gradient.
//
// Flutter: ui/painting.dart Gradient.radial
func (g *Gradient) FocalRadius() float64 { return g.focalRadius }

// StartAngle is the angle a sweep gradient starts at.
//
// Flutter: ui/painting.dart Gradient.sweep
func (g *Gradient) StartAngle() float64 { return g.startAngle }

// EndAngle is the angle a sweep gradient ends at.
//
// Flutter: ui/painting.dart Gradient.sweep
func (g *Gradient) EndAngle() float64 { return g.endAngle }

// Colors returns a copy of the colors, converted to extended sRGB as the engine
// takes them (their ColorSpace field may still say sRGB: the components are the
// same).
//
// Flutter: ui/painting.dart _encodeWideColorList
func (g *Gradient) Colors() []Color { return append([]Color(nil), g.colors...) }

// ColorStops returns a copy of the color stops, or nil when there are two
// colors at 0 and 1.
//
// Flutter: ui/painting.dart Gradient.linear
func (g *Gradient) ColorStops() []float64 { return append([]float64(nil), g.colorStops...) }

// TileMode is what happens outside the gradient's bounds.
//
// Flutter: ui/painting.dart Gradient.linear
func (g *Gradient) TileMode() TileMode { return g.tileMode }

// Matrix4 returns a copy of the transform of the gradient, and false when it has
// none.
//
// Flutter: ui/painting.dart Gradient.linear
func (g *Gradient) Matrix4() (*Matrix4, bool) { return &Matrix4{g.matrix}, g.hasMatrix }

// ImageShader is a shader that tiles an image.
//
// Flutter: ui/painting.dart ImageShader
type ImageShader struct {
	image         *Image
	tmx, tmy      TileMode
	matrix        [16]float32
	filterQuality FilterQuality
}

func (*ImageShader) shader() {}

// ImageShaderOpts are the optional arguments of NewImageShader.
//
// Flutter: ui/painting.dart ImageShader
type ImageShaderOpts struct {
	// FilterQuality is the quality of the resampling; the zero value leaves the
	// choice to the renderer (Flutter's null).
	FilterQuality FilterQuality
}

// NewImageShader creates an image shader that repeats the image as the tile
// modes say on each axis, transformed by the matrix. A nil matrix is the
// identity. The matrix is copied.
//
// Flutter: ui/painting.dart ImageShader
func NewImageShader(image *Image, tmx, tmy TileMode, matrix *Matrix4, opts ...ImageShaderOpts) *ImageShader {
	o := oneOpts(opts)
	if tmx == TileModeUnspecified {
		tmx = TileModeClamp
	}
	if tmy == TileModeUnspecified {
		tmy = TileModeClamp
	}
	s := &ImageShader{image: image, tmx: tmx, tmy: tmy, filterQuality: o.FilterQuality}
	if matrix == nil {
		matrix = NewMatrix4Identity()
	}
	s.matrix = matrix.storage
	return s
}

// Image is the image that is tiled.
//
// Flutter: ui/painting.dart ImageShader
func (s *ImageShader) Image() *Image { return s.image }

// TileModeX is how the image repeats horizontally.
//
// Flutter: ui/painting.dart ImageShader
func (s *ImageShader) TileModeX() TileMode { return s.tmx }

// TileModeY is how the image repeats vertically.
//
// Flutter: ui/painting.dart ImageShader
func (s *ImageShader) TileModeY() TileMode { return s.tmy }

// Matrix4 returns a copy of the transform of the image.
//
// Flutter: ui/painting.dart ImageShader
func (s *ImageShader) Matrix4() *Matrix4 { return &Matrix4{s.matrix} }

// FilterQuality is the quality of the resampling; FilterQualityUnspecified when
// the renderer chooses.
//
// Flutter: ui/painting.dart ImageShader
func (s *ImageShader) FilterQuality() FilterQuality { return s.filterQuality }
