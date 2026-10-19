package rendering

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// RenderImage paints an image into its box.
//
// Flutter: rendering/image.dart RenderImage
type RenderImage struct {
	RenderBox
	image              *ui.Image
	debugImageLabel    string
	width, height      foundation.Opt[float64]
	scale              float64
	color              foundation.Opt[ui.Color]
	opacity            foundation.ValueListenable[float64]
	opacityHandle      foundation.ListenerHandle
	colorBlendMode     ui.BlendMode
	colorFilter        *ui.ColorFilter
	fit                painting.BoxFit
	alignment          painting.AlignmentGeometry
	repeat             painting.ImageRepeat
	centerSlice        foundation.Opt[ui.Rect]
	matchTextDirection bool
	textDirection      ui.TextDirection
	invertColors       bool
	isAntiAlias        bool
	filterQuality      ui.FilterQuality
	resolvedAlignment  foundation.Opt[painting.Alignment]
	flipHorizontally   bool
}

// RenderImageOpts are the optional parameters of NewRenderImage.
//
// Flutter: rendering/image.dart RenderImage.new
type RenderImageOpts struct {
	// Image is the image to display; nil for none.
	Image *ui.Image
	// DebugImageLabel is a description of the image, for debugging.
	DebugImageLabel string
	// Width, if set, requires the box to have this width, if the constraints
	// allow it.
	Width foundation.Opt[float64]
	// Height, if set, requires the box to have this height, if the constraints
	// allow it.
	Height foundation.Opt[float64]
	// Scale is the number of image pixels per logical pixel; the default is 1.
	Scale foundation.Opt[float64]
	// Color, if set, is blended with the image with ColorBlendMode.
	Color foundation.Opt[ui.Color]
	// Opacity, if set, gives the opacity of the image, between 0 and 1.
	// Deviation: Flutter takes an Animation<double>, which implements
	// ValueListenable<double>; the animation library arrives in step 8.1.
	Opacity foundation.ValueListenable[float64]
	// ColorBlendMode is how Color is blended; the default is srcIn.
	ColorBlendMode ui.BlendMode
	// Fit is how the image is inscribed into the box; unspecified is the null
	// of Flutter.
	Fit painting.BoxFit
	// Alignment is how the image is aligned in the box; the default is center.
	Alignment painting.AlignmentGeometry
	// Repeat is how the image repeats in the area it does not cover.
	Repeat painting.ImageRepeat
	// CenterSlice, if set, makes the image a nine-patch.
	CenterSlice foundation.Opt[ui.Rect]
	// MatchTextDirection mirrors the image in a right-to-left context.
	MatchTextDirection bool
	// TextDirection resolves the alignment and the mirroring; unspecified is
	// null.
	TextDirection ui.TextDirection
	// InvertColors inverts the colors of the image.
	InvertColors bool
	// IsAntiAlias paints the image with anti-aliasing.
	IsAntiAlias bool
	// FilterQuality defaults to ui.FilterQualityMedium.
	FilterQuality ui.FilterQuality
}

// NewRenderImage creates the box.
//
// Flutter: rendering/image.dart RenderImage.new
func NewRenderImage(opts ...RenderImageOpts) *RenderImage {
	o := firstOpts(opts)
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	if o.FilterQuality == ui.FilterQualityUnspecified {
		o.FilterQuality = ui.FilterQualityMedium
	}
	if o.Repeat == painting.ImageRepeatUnspecified {
		o.Repeat = painting.ImageRepeatNoRepeat
	}
	r := &RenderImage{image: o.Image, debugImageLabel: o.DebugImageLabel, width: o.Width, height: o.Height,
		scale: o.Scale.OrElse(1), color: o.Color, opacity: o.Opacity, colorBlendMode: o.ColorBlendMode, fit: o.Fit,
		alignment: o.Alignment, repeat: o.Repeat, centerSlice: o.CenterSlice, matchTextDirection: o.MatchTextDirection,
		textDirection: o.TextDirection, invertColors: o.InvertColors, isAntiAlias: o.IsAntiAlias, filterQuality: o.FilterQuality}
	InitRenderBox(&r.RenderBox, r)
	r.updateColorFilter()
	return r
}

func (r *RenderImage) resolve() {
	if r.resolvedAlignment.IsSome() {
		return
	}
	r.resolvedAlignment = foundation.Some(r.alignment.Resolve(r.textDirection))
	r.flipHorizontally = r.matchTextDirection && r.textDirection == ui.TextDirectionRTL
}

func (r *RenderImage) markNeedResolution() {
	r.resolvedAlignment = foundation.None[painting.Alignment]()
	r.flipHorizontally = false
	r.MarkNeedsPaint()
}

// Image is the image to display.
//
// Flutter: rendering/image.dart RenderImage.image
func (r *RenderImage) Image() *ui.Image { return r.image }

// SetImage changes the image. The box takes the ownership of the image and
// disposes the previous one, unless the new one is a clone of it, which is
// disposed instead.
//
// Flutter: rendering/image.dart RenderImage.image
func (r *RenderImage) SetImage(value *ui.Image) {
	if value == r.image {
		return
	}
	if value != nil && r.image != nil && value.IsCloneOf(r.image) {
		value.Dispose()
		return
	}
	sizeChanged := (r.image == nil) != (value == nil) ||
		(r.image != nil && value != nil && (r.image.Width() != value.Width() || r.image.Height() != value.Height()))
	if r.image != nil {
		r.image.Dispose()
	}
	r.image = value
	r.MarkNeedsPaint()
	if sizeChanged && (!r.width.IsSome() || !r.height.IsSome()) {
		r.MarkNeedsLayout()
	}
}

// DebugImageLabel is a description of the image, for debugging.
//
// Flutter: rendering/image.dart RenderImage.debugImageLabel
func (r *RenderImage) DebugImageLabel() string { return r.debugImageLabel }

// SetDebugImageLabel changes the label.
//
// Flutter: rendering/image.dart RenderImage.debugImageLabel
func (r *RenderImage) SetDebugImageLabel(value string) { r.debugImageLabel = value }

// Width is the width the box is required to have, if the constraints allow it.
//
// Flutter: rendering/image.dart RenderImage.width
func (r *RenderImage) Width() foundation.Opt[float64] { return r.width }

// SetWidth marks the box as needing layout when the value changes.
//
// Flutter: rendering/image.dart RenderImage.width
func (r *RenderImage) SetWidth(value foundation.Opt[float64]) {
	if value == r.width {
		return
	}
	r.width = value
	r.MarkNeedsLayout()
}

// Height is the height the box is required to have, if the constraints allow it.
//
// Flutter: rendering/image.dart RenderImage.height
func (r *RenderImage) Height() foundation.Opt[float64] { return r.height }

// SetHeight marks the box as needing layout when the value changes.
//
// Flutter: rendering/image.dart RenderImage.height
func (r *RenderImage) SetHeight(value foundation.Opt[float64]) {
	if value == r.height {
		return
	}
	r.height = value
	r.MarkNeedsLayout()
}

// Scale is the number of image pixels per logical pixel.
//
// Flutter: rendering/image.dart RenderImage.scale
func (r *RenderImage) Scale() float64 { return r.scale }

// SetScale marks the box as needing layout when the value changes.
//
// Flutter: rendering/image.dart RenderImage.scale
func (r *RenderImage) SetScale(value float64) {
	if value == r.scale {
		return
	}
	r.scale = value
	r.MarkNeedsLayout()
}

func (r *RenderImage) updateColorFilter() {
	c, ok := r.color.Get()
	if !ok {
		r.colorFilter = nil
		return
	}
	mode := r.colorBlendMode
	if mode == ui.BlendModeUnspecified {
		mode = ui.BlendModeSrcIn
	}
	r.colorFilter = ui.NewColorFilterMode(c, mode)
}

// Color is the color that is blended with the image.
//
// Flutter: rendering/image.dart RenderImage.color
func (r *RenderImage) Color() foundation.Opt[ui.Color] { return r.color }

// SetColor marks the box as needing paint when the value changes.
//
// Flutter: rendering/image.dart RenderImage.color
func (r *RenderImage) SetColor(value foundation.Opt[ui.Color]) {
	if value == r.color {
		return
	}
	r.color = value
	r.updateColorFilter()
	r.MarkNeedsPaint()
}

// Opacity is the opacity of the image, or nil.
//
// Flutter: rendering/image.dart RenderImage.opacity
func (r *RenderImage) Opacity() foundation.ValueListenable[float64] { return r.opacity }

// SetOpacity changes the opacity; the box repaints when it notifies.
//
// Flutter: rendering/image.dart RenderImage.opacity
func (r *RenderImage) SetOpacity(value foundation.ValueListenable[float64]) {
	if value == r.opacity {
		return
	}
	if r.Attached() && r.opacity != nil {
		r.opacity.RemoveListener(r.opacityHandle)
	}
	r.opacity = value
	if r.Attached() && value != nil {
		r.opacityHandle = value.AddListener(r.MarkNeedsPaint)
	}
}

// FilterQuality is the quality of sampling when the image is scaled.
//
// Flutter: rendering/image.dart RenderImage.filterQuality
func (r *RenderImage) FilterQuality() ui.FilterQuality { return r.filterQuality }

// SetFilterQuality marks the box as needing paint when the value changes.
//
// Flutter: rendering/image.dart RenderImage.filterQuality
func (r *RenderImage) SetFilterQuality(value ui.FilterQuality) {
	if value == ui.FilterQualityUnspecified {
		value = ui.FilterQualityMedium
	}
	if value == r.filterQuality {
		return
	}
	r.filterQuality = value
	r.MarkNeedsPaint()
}

// ColorBlendMode is how the color is blended with the image.
//
// Flutter: rendering/image.dart RenderImage.colorBlendMode
func (r *RenderImage) ColorBlendMode() ui.BlendMode { return r.colorBlendMode }

// SetColorBlendMode marks the box as needing paint when the value changes.
//
// Flutter: rendering/image.dart RenderImage.colorBlendMode
func (r *RenderImage) SetColorBlendMode(value ui.BlendMode) {
	if value == r.colorBlendMode {
		return
	}
	r.colorBlendMode = value
	r.updateColorFilter()
	r.MarkNeedsPaint()
}

// Fit is how the image is inscribed into the box.
//
// Flutter: rendering/image.dart RenderImage.fit
func (r *RenderImage) Fit() painting.BoxFit { return r.fit }

// SetFit marks the box as needing paint when the value changes.
//
// Flutter: rendering/image.dart RenderImage.fit
func (r *RenderImage) SetFit(value painting.BoxFit) {
	if value == r.fit {
		return
	}
	r.fit = value
	r.MarkNeedsPaint()
}

// Alignment is how the image is aligned in the box.
//
// Flutter: rendering/image.dart RenderImage.alignment
func (r *RenderImage) Alignment() painting.AlignmentGeometry { return r.alignment }

// SetAlignment marks the box as needing paint when the value changes.
//
// Flutter: rendering/image.dart RenderImage.alignment
func (r *RenderImage) SetAlignment(value painting.AlignmentGeometry) {
	if value == nil {
		value = painting.AlignmentCenter
	}
	if value == r.alignment {
		return
	}
	r.alignment = value
	r.markNeedResolution()
}

// Repeat is how the image repeats in the area it does not cover.
//
// Flutter: rendering/image.dart RenderImage.repeat
func (r *RenderImage) Repeat() painting.ImageRepeat { return r.repeat }

// SetRepeat marks the box as needing paint when the value changes.
//
// Flutter: rendering/image.dart RenderImage.repeat
func (r *RenderImage) SetRepeat(value painting.ImageRepeat) {
	if value == painting.ImageRepeatUnspecified {
		value = painting.ImageRepeatNoRepeat
	}
	if value == r.repeat {
		return
	}
	r.repeat = value
	r.MarkNeedsPaint()
}

// CenterSlice is the center slice of a nine-patch image.
//
// Flutter: rendering/image.dart RenderImage.centerSlice
func (r *RenderImage) CenterSlice() foundation.Opt[ui.Rect] { return r.centerSlice }

// SetCenterSlice marks the box as needing paint when the value changes.
//
// Flutter: rendering/image.dart RenderImage.centerSlice
func (r *RenderImage) SetCenterSlice(value foundation.Opt[ui.Rect]) {
	if value == r.centerSlice {
		return
	}
	r.centerSlice = value
	r.MarkNeedsPaint()
}

// InvertColors says whether the colors of the image are inverted.
//
// Flutter: rendering/image.dart RenderImage.invertColors
func (r *RenderImage) InvertColors() bool { return r.invertColors }

// SetInvertColors marks the box as needing paint when the value changes.
//
// Flutter: rendering/image.dart RenderImage.invertColors
func (r *RenderImage) SetInvertColors(value bool) {
	if value == r.invertColors {
		return
	}
	r.invertColors = value
	r.MarkNeedsPaint()
}

// MatchTextDirection says whether the image is mirrored in a right-to-left
// context.
//
// Flutter: rendering/image.dart RenderImage.matchTextDirection
func (r *RenderImage) MatchTextDirection() bool { return r.matchTextDirection }

// SetMatchTextDirection marks the box as needing paint when the value changes.
//
// Flutter: rendering/image.dart RenderImage.matchTextDirection
func (r *RenderImage) SetMatchTextDirection(value bool) {
	if value == r.matchTextDirection {
		return
	}
	r.matchTextDirection = value
	r.markNeedResolution()
}

// TextDirection resolves the alignment and the mirroring. Unspecified is null.
//
// Flutter: rendering/image.dart RenderImage.textDirection
func (r *RenderImage) TextDirection() ui.TextDirection { return r.textDirection }

// SetTextDirection marks the box as needing paint when the value changes.
//
// Flutter: rendering/image.dart RenderImage.textDirection
func (r *RenderImage) SetTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.markNeedResolution()
}

// IsAntiAlias says whether the image is painted with anti-aliasing.
//
// Flutter: rendering/image.dart RenderImage.isAntiAlias
func (r *RenderImage) IsAntiAlias() bool { return r.isAntiAlias }

// SetIsAntiAlias marks the box as needing paint when the value changes.
//
// Flutter: rendering/image.dart RenderImage.isAntiAlias
func (r *RenderImage) SetIsAntiAlias(value bool) {
	if r.isAntiAlias == value {
		return
	}
	r.isAntiAlias = value
	r.MarkNeedsPaint()
}

func (r *RenderImage) sizeForConstraints(constraints BoxConstraints) ui.Size {
	constraints = NewBoxConstraintsTightFor(TightForOpts{Width: r.width, Height: r.height}).Enforce(constraints)
	if r.image == nil {
		return constraints.Smallest()
	}
	return constraints.ConstrainSizeAndAttemptToPreserveAspectRatio(
		ui.Size{Width: float64(r.image.Width()) / r.scale, Height: float64(r.image.Height()) / r.scale})
}

// ComputeMinIntrinsicWidth is 0 if the box has no size of its own, else the
// width it would have for the height.
//
// Flutter: rendering/image.dart RenderImage.computeMinIntrinsicWidth
func (r *RenderImage) ComputeMinIntrinsicWidth(height float64) float64 {
	if !r.width.IsSome() && !r.height.IsSome() {
		return 0
	}
	return r.sizeForConstraints(NewBoxConstraintsTightForFinite(TightForFiniteOpts{Height: foundation.Some(height)})).Width
}

// ComputeMaxIntrinsicWidth is the width the box would have for the height.
//
// Flutter: rendering/image.dart RenderImage.computeMaxIntrinsicWidth
func (r *RenderImage) ComputeMaxIntrinsicWidth(height float64) float64 {
	return r.sizeForConstraints(NewBoxConstraintsTightForFinite(TightForFiniteOpts{Height: foundation.Some(height)})).Width
}

// ComputeMinIntrinsicHeight is 0 if the box has no size of its own, else the
// height it would have for the width.
//
// Flutter: rendering/image.dart RenderImage.computeMinIntrinsicHeight
func (r *RenderImage) ComputeMinIntrinsicHeight(width float64) float64 {
	if !r.width.IsSome() && !r.height.IsSome() {
		return 0
	}
	return r.sizeForConstraints(NewBoxConstraintsTightForFinite(TightForFiniteOpts{Width: foundation.Some(width)})).Height
}

// ComputeMaxIntrinsicHeight is the height the box would have for the width.
//
// Flutter: rendering/image.dart RenderImage.computeMaxIntrinsicHeight
func (r *RenderImage) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.sizeForConstraints(NewBoxConstraintsTightForFinite(TightForFiniteOpts{Width: foundation.Some(width)})).Height
}

// HitTestSelf is true: the box is hit anywhere.
//
// Flutter: rendering/image.dart RenderImage.hitTestSelf
func (r *RenderImage) HitTestSelf(ui.Offset) bool { return true }

// ComputeDryLayout is the size the constraints would give.
//
// Flutter: rendering/image.dart RenderImage.computeDryLayout
func (r *RenderImage) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	return r.sizeForConstraints(constraints)
}

// PerformLayout sizes the box as the image, within the constraints.
//
// Flutter: rendering/image.dart RenderImage.performLayout
func (r *RenderImage) PerformLayout() { r.SetSize(r.sizeForConstraints(r.BoxConstraints())) }

// Attach starts listening to the opacity.
//
// Flutter: rendering/image.dart RenderImage.attach
func (r *RenderImage) Attach(owner *PipelineOwner) {
	r.RenderBox.Attach(owner)
	if r.opacity != nil {
		r.opacityHandle = r.opacity.AddListener(r.MarkNeedsPaint)
	}
}

// Detach stops listening to the opacity.
//
// Flutter: rendering/image.dart RenderImage.detach
func (r *RenderImage) Detach() {
	if r.opacity != nil {
		r.opacity.RemoveListener(r.opacityHandle)
		r.opacityHandle = foundation.ListenerHandle{}
	}
	r.RenderBox.Detach()
}

// Paint paints the image.
//
// Flutter: rendering/image.dart RenderImage.paint
func (r *RenderImage) Paint(context *PaintingContext, offset ui.Offset) {
	if r.image == nil {
		return
	}
	r.resolve()
	opacity := 1.0
	if r.opacity != nil {
		opacity = r.opacity.Value()
	}
	painting.PaintImage(context.Canvas(), offset.And(r.Size()), r.image, painting.PaintImageOpts{
		DebugImageLabel: r.debugImageLabel, Scale: foundation.Some(r.scale), Opacity: foundation.Some(opacity),
		ColorFilter: r.colorFilter, Fit: r.fit, Alignment: r.resolvedAlignment, CenterSlice: r.centerSlice,
		Repeat: r.repeat, FlipHorizontally: r.flipHorizontally, InvertColors: r.invertColors,
		FilterQuality: r.filterQuality, IsAntiAlias: r.isAntiAlias,
	})
}

// Dispose disposes the image.
//
// Flutter: rendering/image.dart RenderImage.dispose
func (r *RenderImage) Dispose() {
	if r.image != nil {
		r.image.Dispose()
		r.image = nil
	}
	r.RenderBox.Dispose()
}

// DebugFillProperties adds the image and its settings.
//
// Flutter: rendering/image.dart RenderImage.debugFillProperties
func (r *RenderImage) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderBox.DebugFillProperties(b)
	null := foundation.DefaultNull
	b.Add(foundation.NewDiagnosticsProperty("image", r.image))
	b.Add(foundation.NewDoubleProperty("width", r.width, foundation.NumberPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewDoubleProperty("height", r.height, foundation.NumberPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewDoubleProperty("scale", foundation.Some(r.scale), foundation.NumberPropertyOpts{DefaultValue: 1.0}))
	b.Add(painting.NewColorProperty("color", r.color, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewDiagnosticsProperty("opacity", r.opacity, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewEnumProperty("colorBlendMode", optEnum(r.colorBlendMode, ui.BlendModeUnspecified), foundation.EnumPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewEnumProperty("fit", optEnum(r.fit, painting.BoxFitUnspecified), foundation.EnumPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewDiagnosticsProperty("alignment", r.alignment))
	b.Add(foundation.NewEnumProperty("repeat", foundation.Some(r.repeat), foundation.EnumPropertyOpts{DefaultValue: painting.ImageRepeatNoRepeat}))
	b.Add(foundation.NewDiagnosticsPropertyOpt("centerSlice", r.centerSlice, foundation.DiagnosticsPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewFlagProperty("matchTextDirection", foundation.Some(r.matchTextDirection), foundation.FlagPropertyOpts{IfTrue: "match text direction"}))
	b.Add(foundation.NewEnumProperty("textDirection", optEnum(r.textDirection, ui.TextDirectionUnspecified), foundation.EnumPropertyOpts{DefaultValue: null}))
	b.Add(foundation.NewDiagnosticsProperty("invertColors", r.invertColors))
	b.Add(foundation.NewEnumProperty("filterQuality", foundation.Some(r.filterQuality)))
}
