package ui

import (
	"fmt"
	"strings"
)

// MaskFilter is a mask filter to apply to shapes as they are painted. A mask
// filter is a function that takes a bitmap of color pixels, and returns another
// bitmap of color pixels. Currently only a blur is supported.
//
// Flutter: ui/painting.dart MaskFilter
type MaskFilter struct {
	style BlurStyle
	sigma float64
}

// NewMaskFilterBlur creates a mask filter that applies a Gaussian blur of the
// given standard deviation (sigma). A zero style is BlurStyleNormal.
//
// Flutter: ui/painting.dart MaskFilter.blur
func NewMaskFilterBlur(style BlurStyle, sigma float64) *MaskFilter {
	if style == BlurStyleUnspecified {
		style = BlurStyleNormal
	}
	return &MaskFilter{style, sigma}
}

// Style is the blur style.
//
// Flutter: ui/painting.dart MaskFilter._style
func (m *MaskFilter) Style() BlurStyle { return m.style }

// Sigma is the standard deviation of the Gaussian blur, in logical pixels.
//
// Flutter: ui/painting.dart MaskFilter._sigma
func (m *MaskFilter) Sigma() float64 { return m.sigma }

// Equal reports whether the two mask filters are the same blur. A nil filter
// equals only nil.
//
// Flutter: ui/painting.dart MaskFilter.operator ==
func (m *MaskFilter) Equal(other *MaskFilter) bool {
	if m == nil || other == nil {
		return m == other
	}
	return *m == *other
}

// String describes the filter, as Dart's toString does.
//
// Flutter: ui/painting.dart MaskFilter.toString
func (m *MaskFilter) String() string {
	return fmt.Sprintf("MaskFilter.blur(%s, %s)", m.style, toStringAsFixed1(m.sigma))
}

// ColorFilterKind says which kind of ColorFilter a filter is.
//
// Flutter: ui/painting.dart ColorFilter._type
type ColorFilterKind int

// The kinds of ColorFilter.
const (
	// ColorFilterKindUnspecified is the zero value; no filter has it.
	//
	// Flutter: no counterpart
	ColorFilterKindUnspecified ColorFilterKind = iota
	// ColorFilterKindMode blends a color over the source with a blend mode.
	//
	// Flutter: ui/painting.dart ColorFilter._kTypeMode
	ColorFilterKindMode
	// ColorFilterKindMatrix multiplies the color by a 5x4 matrix.
	//
	// Flutter: ui/painting.dart ColorFilter._kTypeMatrix
	ColorFilterKindMatrix
	// ColorFilterKindLinearToSrgbGamma converts linear colors to sRGB.
	//
	// Flutter: ui/painting.dart ColorFilter._kTypeLinearToSrgbGamma
	ColorFilterKindLinearToSrgbGamma
	// ColorFilterKindSrgbToLinearGamma converts sRGB colors to linear.
	//
	// Flutter: ui/painting.dart ColorFilter._kTypeSrgbToLinearGamma
	ColorFilterKindSrgbToLinearGamma
)

// ColorFilter is a description of a color filter to apply when drawing a shape or
// compositing a layer with a particular Paint. A color filter is a function that
// takes two colors, and outputs one color. When applied during compositing, it is
// independently applied to each pixel of the layer being drawn before the entire
// layer is merged with the destination.
//
// A ColorFilter is also an ImageFilter.
//
// Flutter: ui/painting.dart ColorFilter
type ColorFilter struct {
	kind      ColorFilterKind
	color     Color
	blendMode BlendMode
	matrix    [20]float64
}

// NewColorFilterMode creates a color filter that applies the blend mode given as
// the second argument. The source color is the one given as the first argument,
// and the destination color is the one from the layer being composited.
//
// Flutter: ui/painting.dart ColorFilter.mode
func NewColorFilterMode(color Color, blendMode BlendMode) *ColorFilter {
	if blendMode == BlendModeUnspecified {
		blendMode = BlendModeSrcOver
	}
	return &ColorFilter{kind: ColorFilterKindMode, color: color, blendMode: blendMode}
}

// NewColorFilterMatrix creates a color filter from a 5x4 color matrix, in
// row-major order. The matrix is applied to the color as a column vector
// (r, g, b, a, 1) with the components from 0 to 255:
//
//	| R' |   | a00 a01 a02 a03 a04 |   | R |
//	| G' |   | a10 a11 a12 a13 a14 |   | G |
//	| B' | = | a20 a21 a22 a23 a24 | * | B |
//	| A' |   | a30 a31 a32 a33 a34 |   | A |
//	                                    | 1 |
//
// Flutter: ui/painting.dart ColorFilter.matrix
func NewColorFilterMatrix(matrix [20]float64) *ColorFilter {
	return &ColorFilter{kind: ColorFilterKindMatrix, matrix: matrix}
}

// NewColorFilterLinearToSrgbGamma creates a color filter that applies the
// inverse of the sRGB gamma curve to the RGB channels.
//
// Flutter: ui/painting.dart ColorFilter.linearToSrgbGamma
func NewColorFilterLinearToSrgbGamma() *ColorFilter {
	return &ColorFilter{kind: ColorFilterKindLinearToSrgbGamma}
}

// NewColorFilterSrgbToLinearGamma creates a color filter that applies the sRGB
// gamma curve to the RGB channels.
//
// Flutter: ui/painting.dart ColorFilter.srgbToLinearGamma
func NewColorFilterSrgbToLinearGamma() *ColorFilter {
	return &ColorFilter{kind: ColorFilterKindSrgbToLinearGamma}
}

// NewColorFilterSaturation creates a color filter that scales the saturation:
// 0 gives grayscale, 1 leaves the colors alone, and more than 1 oversaturates.
//
// Flutter: ui/painting.dart ColorFilter.saturation
func NewColorFilterSaturation(saturation float64) *ColorFilter {
	const rLuminance = 0.2126
	const gLuminance = 0.7152
	const bLuminance = 0.0722
	invSat := 1 - saturation
	return NewColorFilterMatrix([20]float64{
		invSat*rLuminance + saturation, invSat * gLuminance, invSat * bLuminance, 0, 0,
		invSat * rLuminance, invSat*gLuminance + saturation, invSat * bLuminance, 0, 0,
		invSat * rLuminance, invSat * gLuminance, invSat*bLuminance + saturation, 0, 0,
		0, 0, 0, 1, 0,
	})
}

// Kind says which kind of filter this is.
//
// Flutter: ui/painting.dart ColorFilter._type
func (f *ColorFilter) Kind() ColorFilterKind { return f.kind }

// Color is the source color of a mode filter.
//
// Flutter: ui/painting.dart ColorFilter._color
func (f *ColorFilter) Color() Color { return f.color }

// BlendMode is the blend mode of a mode filter.
//
// Flutter: ui/painting.dart ColorFilter._blendMode
func (f *ColorFilter) BlendMode() BlendMode { return f.blendMode }

// Matrix is the 5x4 row-major matrix of a matrix filter.
//
// Flutter: ui/painting.dart ColorFilter._matrix
func (f *ColorFilter) Matrix() [20]float64 { return f.matrix }

func (*ColorFilter) imageFilter() {}

// Equal reports whether other is a color filter of the same kind and values.
//
// Flutter: ui/painting.dart ColorFilter.operator ==
func (f *ColorFilter) Equal(other ImageFilter) bool {
	o, ok := other.(*ColorFilter)
	if !ok || f == nil || o == nil {
		return ok && f == o
	}
	return *f == *o
}

// DebugShortDescription describes the filter in a few words.
//
// Flutter: ui/painting.dart ColorFilter.debugShortDescription
func (f *ColorFilter) DebugShortDescription() string { return f.String() }

// String describes the filter, as Dart's toString does.
//
// Flutter: ui/painting.dart ColorFilter.toString
func (f *ColorFilter) String() string {
	switch f.kind {
	case ColorFilterKindMode:
		return fmt.Sprintf("ColorFilter.mode(%s, %s)", f.color, f.blendMode)
	case ColorFilterKindMatrix:
		parts := make([]string, len(f.matrix))
		for i, v := range f.matrix {
			parts[i] = dartDouble(v)
		}
		return "ColorFilter.matrix([" + strings.Join(parts, ", ") + "])"
	case ColorFilterKindLinearToSrgbGamma:
		return "ColorFilter.linearToSrgbGamma()"
	case ColorFilterKindSrgbToLinearGamma:
		return "ColorFilter.srgbToLinearGamma()"
	}
	return "Unknown ColorFilter type. This is an error."
}

// ImageFilter is a filter operation to apply to a raster image: a blur, a
// dilation or erosion, a matrix transform, a composition of two filters, or a
// ColorFilter. The concrete types are *BlurImageFilter, *DilateImageFilter,
// *ErodeImageFilter, *MatrixImageFilter, *ComposeImageFilter and *ColorFilter.
//
// Flutter: ui/painting.dart ImageFilter
type ImageFilter interface {
	// DebugShortDescription describes the filter in a few words.
	DebugShortDescription() string
	// Equal reports whether other is the same filter with the same values.
	Equal(other ImageFilter) bool
	// String describes the filter, as Dart's toString does.
	String() string

	imageFilter()
}

// BlurImageFilter is the Gaussian blur image filter.
//
// Flutter: ui/painting.dart _GaussianBlurImageFilter
type BlurImageFilter struct {
	sigmaX, sigmaY float64
	tileMode       TileMode
	bounds         Rect
	hasBounds      bool
}

// ImageFilterBlurOpts are the optional arguments of NewImageFilterBlur.
//
// Flutter: ui/painting.dart ImageFilter.blur
type ImageFilterBlurOpts struct {
	// SigmaX is the standard deviation of the blur on the x axis.
	SigmaX float64
	// SigmaY is the standard deviation of the blur on the y axis.
	SigmaY float64
	// TileMode is how the edges are treated; the zero value leaves the choice to
	// the renderer (Flutter's null).
	TileMode TileMode
	// Bounds limits where the filter reads and writes; nil means unbounded.
	Bounds *Rect
}

// NewImageFilterBlur creates an image filter that applies a Gaussian blur.
//
// Flutter: ui/painting.dart ImageFilter.blur
func NewImageFilterBlur(opts ...ImageFilterBlurOpts) ImageFilter {
	o := oneOpts(opts)
	f := &BlurImageFilter{sigmaX: o.SigmaX, sigmaY: o.SigmaY, tileMode: o.TileMode}
	if o.Bounds != nil {
		f.bounds, f.hasBounds = *o.Bounds, true
	}
	return f
}

// SigmaX is the standard deviation of the blur on the x axis.
//
// Flutter: ui/painting.dart _GaussianBlurImageFilter.sigmaX
func (f *BlurImageFilter) SigmaX() float64 { return f.sigmaX }

// SigmaY is the standard deviation of the blur on the y axis.
//
// Flutter: ui/painting.dart _GaussianBlurImageFilter.sigmaY
func (f *BlurImageFilter) SigmaY() float64 { return f.sigmaY }

// TileMode is how the edges are treated; TileModeUnspecified when not given.
//
// Flutter: ui/painting.dart _GaussianBlurImageFilter.tileMode
func (f *BlurImageFilter) TileMode() TileMode { return f.tileMode }

// Bounds is the rectangle the filter is limited to, and whether there is one.
//
// Flutter: ui/painting.dart _GaussianBlurImageFilter.bounds
func (f *BlurImageFilter) Bounds() (Rect, bool) { return f.bounds, f.hasBounds }

func (*BlurImageFilter) imageFilter() {}

// Equal reports whether other is a blur with the same values.
//
// Flutter: ui/painting.dart _GaussianBlurImageFilter.operator ==
func (f *BlurImageFilter) Equal(other ImageFilter) bool {
	o, ok := other.(*BlurImageFilter)
	return ok && *f == *o
}

func (f *BlurImageFilter) modeString() string {
	switch f.tileMode {
	case TileModeClamp:
		return "clamp"
	case TileModeMirror:
		return "mirror"
	case TileModeRepeated:
		return "repeated"
	case TileModeDecal:
		return "decal"
	}
	return "unspecified"
}

func (f *BlurImageFilter) boundsString() string {
	if !f.hasBounds {
		return ""
	}
	return ", bounds: " + f.bounds.String()
}

// DebugShortDescription describes the filter in a few words.
//
// Flutter: ui/painting.dart _GaussianBlurImageFilter.debugShortDescription
func (f *BlurImageFilter) DebugShortDescription() string {
	return fmt.Sprintf("blur(%s, %s, %s%s)", dartDouble(f.sigmaX), dartDouble(f.sigmaY), f.modeString(), f.boundsString())
}

// String describes the filter, as Dart's toString does.
//
// Flutter: ui/painting.dart _GaussianBlurImageFilter.toString
func (f *BlurImageFilter) String() string {
	return fmt.Sprintf("ImageFilter.blur(%s, %s, %s%s)", dartDouble(f.sigmaX), dartDouble(f.sigmaY), f.modeString(), f.boundsString())
}

// DilateImageFilter is the morphological dilation image filter.
//
// Flutter: ui/painting.dart _DilateImageFilter
type DilateImageFilter struct{ radiusX, radiusY float64 }

// ErodeImageFilter is the morphological erosion image filter.
//
// Flutter: ui/painting.dart _ErodeImageFilter
type ErodeImageFilter struct{ radiusX, radiusY float64 }

// ImageFilterMorphologyOpts are the optional arguments of NewImageFilterDilate
// and NewImageFilterErode.
//
// Flutter: ui/painting.dart ImageFilter.dilate
type ImageFilterMorphologyOpts struct {
	// RadiusX is the radius of the filter on the x axis, in logical pixels.
	RadiusX float64
	// RadiusY is the radius of the filter on the y axis, in logical pixels.
	RadiusY float64
}

// NewImageFilterDilate creates an image filter that dilates each pixel's color
// from its neighboring pixels, within the radii.
//
// Flutter: ui/painting.dart ImageFilter.dilate
func NewImageFilterDilate(opts ...ImageFilterMorphologyOpts) ImageFilter {
	o := oneOpts(opts)
	return &DilateImageFilter{o.RadiusX, o.RadiusY}
}

// NewImageFilterErode creates an image filter that erodes each pixel's color
// from its neighboring pixels, within the radii.
//
// Flutter: ui/painting.dart ImageFilter.erode
func NewImageFilterErode(opts ...ImageFilterMorphologyOpts) ImageFilter {
	o := oneOpts(opts)
	return &ErodeImageFilter{o.RadiusX, o.RadiusY}
}

// RadiusX is the radius on the x axis.
//
// Flutter: ui/painting.dart _DilateImageFilter.radiusX
func (f *DilateImageFilter) RadiusX() float64 { return f.radiusX }

// RadiusY is the radius on the y axis.
//
// Flutter: ui/painting.dart _DilateImageFilter.radiusY
func (f *DilateImageFilter) RadiusY() float64 { return f.radiusY }

// RadiusX is the radius on the x axis.
//
// Flutter: ui/painting.dart _ErodeImageFilter.radiusX
func (f *ErodeImageFilter) RadiusX() float64 { return f.radiusX }

// RadiusY is the radius on the y axis.
//
// Flutter: ui/painting.dart _ErodeImageFilter.radiusY
func (f *ErodeImageFilter) RadiusY() float64 { return f.radiusY }

func (*DilateImageFilter) imageFilter() {}
func (*ErodeImageFilter) imageFilter()  {}

// Equal reports whether other is a dilation with the same radii.
//
// Flutter: ui/painting.dart _DilateImageFilter.operator ==
func (f *DilateImageFilter) Equal(other ImageFilter) bool {
	o, ok := other.(*DilateImageFilter)
	return ok && *f == *o
}

// Equal reports whether other is an erosion with the same radii.
//
// Flutter: ui/painting.dart _ErodeImageFilter.operator ==
func (f *ErodeImageFilter) Equal(other ImageFilter) bool {
	o, ok := other.(*ErodeImageFilter)
	return ok && *f == *o
}

// DebugShortDescription describes the filter in a few words.
//
// Flutter: ui/painting.dart _DilateImageFilter.debugShortDescription
func (f *DilateImageFilter) DebugShortDescription() string {
	return fmt.Sprintf("dilate(%s, %s)", dartDouble(f.radiusX), dartDouble(f.radiusY))
}

// DebugShortDescription describes the filter in a few words.
//
// Flutter: ui/painting.dart _ErodeImageFilter.debugShortDescription
func (f *ErodeImageFilter) DebugShortDescription() string {
	return fmt.Sprintf("erode(%s, %s)", dartDouble(f.radiusX), dartDouble(f.radiusY))
}

// String describes the filter, as Dart's toString does.
//
// Flutter: ui/painting.dart _DilateImageFilter.toString
func (f *DilateImageFilter) String() string {
	return fmt.Sprintf("ImageFilter.dilate(%s, %s)", dartDouble(f.radiusX), dartDouble(f.radiusY))
}

// String describes the filter, as Dart's toString does.
//
// Flutter: ui/painting.dart _ErodeImageFilter.toString
func (f *ErodeImageFilter) String() string {
	return fmt.Sprintf("ImageFilter.erode(%s, %s)", dartDouble(f.radiusX), dartDouble(f.radiusY))
}

// MatrixImageFilter is the image filter that transforms the image with a matrix.
//
// Flutter: ui/painting.dart _MatrixImageFilter
type MatrixImageFilter struct {
	matrix        [16]float32
	filterQuality FilterQuality
}

// ImageFilterMatrixOpts are the optional arguments of NewImageFilterMatrix.
//
// Flutter: ui/painting.dart ImageFilter.matrix
type ImageFilterMatrixOpts struct {
	// FilterQuality is the quality of the resampling; the zero value is
	// FilterQualityMedium, Flutter's default here.
	FilterQuality FilterQuality
}

// NewImageFilterMatrix creates an image filter that applies a matrix
// transformation. The matrix is copied.
//
// Flutter: ui/painting.dart ImageFilter.matrix
func NewImageFilterMatrix(matrix *Matrix4, opts ...ImageFilterMatrixOpts) ImageFilter {
	o := oneOpts(opts)
	if o.FilterQuality == FilterQualityUnspecified {
		o.FilterQuality = FilterQualityMedium
	}
	return &MatrixImageFilter{matrix.storage, o.FilterQuality}
}

// Matrix is a copy of the transformation matrix.
//
// Flutter: ui/painting.dart _MatrixImageFilter.data
func (f *MatrixImageFilter) Matrix() *Matrix4 { return &Matrix4{f.matrix} }

// FilterQuality is the quality of the resampling.
//
// Flutter: ui/painting.dart _MatrixImageFilter.filterQuality
func (f *MatrixImageFilter) FilterQuality() FilterQuality { return f.filterQuality }

func (*MatrixImageFilter) imageFilter() {}

// Equal reports whether other is a matrix filter with the same matrix and
// quality.
//
// Flutter: ui/painting.dart _MatrixImageFilter.operator ==
func (f *MatrixImageFilter) Equal(other ImageFilter) bool {
	o, ok := other.(*MatrixImageFilter)
	return ok && *f == *o
}

func (f *MatrixImageFilter) data() string {
	parts := make([]string, len(f.matrix))
	for i, v := range f.matrix {
		parts[i] = dartDouble(float64(v))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// DebugShortDescription describes the filter in a few words.
//
// Flutter: ui/painting.dart _MatrixImageFilter.debugShortDescription
func (f *MatrixImageFilter) DebugShortDescription() string {
	return fmt.Sprintf("matrix(%s, %s)", f.data(), f.filterQuality)
}

// String describes the filter, as Dart's toString does.
//
// Flutter: ui/painting.dart _MatrixImageFilter.toString
func (f *MatrixImageFilter) String() string {
	return fmt.Sprintf("ImageFilter.matrix(%s, %s)", f.data(), f.filterQuality)
}

// ComposeImageFilter is the image filter that applies one filter after another.
//
// Flutter: ui/painting.dart _ComposeImageFilter
type ComposeImageFilter struct{ outer, inner ImageFilter }

// NewImageFilterCompose creates an image filter that applies the inner filter
// first and then the outer filter to its result.
//
// Flutter: ui/painting.dart ImageFilter.compose
func NewImageFilterCompose(outer, inner ImageFilter) ImageFilter {
	return &ComposeImageFilter{outer, inner}
}

// Outer is the filter applied last.
//
// Flutter: ui/painting.dart _ComposeImageFilter.outerFilter
func (f *ComposeImageFilter) Outer() ImageFilter { return f.outer }

// Inner is the filter applied first.
//
// Flutter: ui/painting.dart _ComposeImageFilter.innerFilter
func (f *ComposeImageFilter) Inner() ImageFilter { return f.inner }

func (*ComposeImageFilter) imageFilter() {}

// Equal reports whether other composes equal filters in the same order.
//
// Flutter: ui/painting.dart _ComposeImageFilter.operator ==
func (f *ComposeImageFilter) Equal(other ImageFilter) bool {
	o, ok := other.(*ComposeImageFilter)
	return ok && f.inner.Equal(o.inner) && f.outer.Equal(o.outer)
}

// DebugShortDescription describes the filter in a few words.
//
// Flutter: ui/painting.dart _ComposeImageFilter.debugShortDescription
func (f *ComposeImageFilter) DebugShortDescription() string {
	return f.inner.DebugShortDescription() + " -> " + f.outer.DebugShortDescription()
}

// String describes the filter, as Dart's toString does.
//
// Flutter: ui/painting.dart _ComposeImageFilter.toString
func (f *ComposeImageFilter) String() string {
	return "ImageFilter.compose(source -> " + f.DebugShortDescription() + " -> result)"
}
