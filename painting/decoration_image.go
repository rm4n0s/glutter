// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"math"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// ImageRepeat is how to paint any portions of a box not covered by an image. The
// zero value is ImageRepeatUnspecified: noRepeat.
//
// Flutter: painting/decoration_image.dart ImageRepeat
type ImageRepeat int

// The values of ImageRepeat follow Flutter's declaration order.
const (
	// ImageRepeatUnspecified is the zero value: noRepeat.
	ImageRepeatUnspecified ImageRepeat = iota
	// ImageRepeatRepeat repeats the image in both the x and y directions.
	//
	// Flutter: painting/decoration_image.dart ImageRepeat.repeat
	ImageRepeatRepeat
	// ImageRepeatRepeatX repeats the image in the x direction.
	//
	// Flutter: painting/decoration_image.dart ImageRepeat.repeatX
	ImageRepeatRepeatX
	// ImageRepeatRepeatY repeats the image in the y direction.
	//
	// Flutter: painting/decoration_image.dart ImageRepeat.repeatY
	ImageRepeatRepeatY
	// ImageRepeatNoRepeat leaves uncovered portions of the box transparent.
	//
	// Flutter: painting/decoration_image.dart ImageRepeat.noRepeat
	ImageRepeatNoRepeat
)

// Name returns the Dart enum name, such as "repeatX".
func (r ImageRepeat) Name() string {
	switch r {
	case ImageRepeatRepeat:
		return "repeat"
	case ImageRepeatRepeatX:
		return "repeatX"
	case ImageRepeatRepeatY:
		return "repeatY"
	}
	return "noRepeat"
}

// String returns the name Dart prints, such as "ImageRepeat.repeatX".
func (r ImageRepeat) String() string { return "ImageRepeat." + r.Name() }

// DecorationImage is an image for a box decoration: the provider and how to
// paint it. It implements DecorationImageSource.
//
// Flutter: painting/decoration_image.dart DecorationImage
type DecorationImage struct {
	// Image is the image to be painted into the decoration.
	Image ImageProvider
	// OnError is called when the image fails to load.
	OnError ImageErrorListener
	// ColorFilter is a color filter to apply to the image before painting it.
	ColorFilter *ui.ColorFilter
	// Fit is how the image should be inscribed into the box; unspecified is
	// scaleDown, or fill with a CenterSlice.
	Fit BoxFit
	// Alignment is how to align the image within its bounds; nil is center.
	Alignment AlignmentGeometry
	// CenterSlice is the center slice of a nine-patch image.
	CenterSlice foundation.Opt[ui.Rect]
	// Repeat is how to paint any portions of the box not covered by the image;
	// unspecified is noRepeat.
	Repeat ImageRepeat
	// MatchTextDirection says whether to paint the image in the direction of the
	// TextDirection.
	MatchTextDirection bool
	// Scale is the scale factor applied to the image; the default is 1.
	Scale float64
	// Opacity is the opacity of the image, 0 to 1; the default is 1.
	Opacity float64
	// FilterQuality is the quality of sampling; the default is medium.
	FilterQuality ui.FilterQuality
	// InvertColors says whether the colors of the image are inverted.
	InvertColors bool
	// IsAntiAlias says whether to paint the image with anti-aliasing.
	IsAntiAlias bool
}

// DecorationImageOpts are the optional parameters of NewDecorationImage.
//
// Flutter: painting/decoration_image.dart DecorationImage.new
type DecorationImageOpts struct {
	// OnError is called when the image fails to load.
	OnError ImageErrorListener
	// ColorFilter is applied to the image before painting it.
	ColorFilter *ui.ColorFilter
	// Fit is how the image is inscribed into the box.
	Fit BoxFit
	// Alignment defaults to center.
	Alignment AlignmentGeometry
	// CenterSlice is the center slice of a nine-patch image.
	CenterSlice foundation.Opt[ui.Rect]
	// Repeat defaults to noRepeat.
	Repeat ImageRepeat
	// MatchTextDirection defaults to false.
	MatchTextDirection bool
	// Scale defaults to 1.
	Scale foundation.Opt[float64]
	// Opacity defaults to 1.
	Opacity foundation.Opt[float64]
	// FilterQuality defaults to medium.
	FilterQuality ui.FilterQuality
	// InvertColors defaults to false.
	InvertColors bool
	// IsAntiAlias defaults to false.
	IsAntiAlias bool
}

// NewDecorationImage creates an image for a decoration.
//
// Flutter: painting/decoration_image.dart DecorationImage.new
func NewDecorationImage(image ImageProvider, opts ...DecorationImageOpts) *DecorationImage {
	o := firstOpts(opts)
	d := &DecorationImage{
		Image: image, OnError: o.OnError, ColorFilter: o.ColorFilter, Fit: o.Fit, Alignment: o.Alignment,
		CenterSlice: o.CenterSlice, Repeat: o.Repeat, MatchTextDirection: o.MatchTextDirection,
		Scale: o.Scale.OrElse(1.0), Opacity: o.Opacity.OrElse(1.0), FilterQuality: o.FilterQuality,
		InvertColors: o.InvertColors, IsAntiAlias: o.IsAntiAlias,
	}
	if d.Alignment == nil {
		d.Alignment = AlignmentCenter
	}
	if d.Repeat == ImageRepeatUnspecified {
		d.Repeat = ImageRepeatNoRepeat
	}
	if d.FilterQuality == ui.FilterQualityUnspecified {
		d.FilterQuality = ui.FilterQualityMedium
	}
	return d
}

// CreatePainter creates a painter for this image, which calls onChanged when it
// must be repainted.
//
// Flutter: painting/decoration_image.dart DecorationImage.createPainter
func (d *DecorationImage) CreatePainter(onChanged func()) DecorationImagePainter {
	return &decorationImagePainter{details: d, onChanged: onChanged}
}

// Equal is Dart's == on decoration images.
//
// Flutter: painting/decoration_image.dart DecorationImage.operator ==
func (d *DecorationImage) Equal(other DecorationImageSource) bool {
	o, ok := other.(*DecorationImage)
	if !ok || o == nil || d == nil {
		return false
	}
	if d == o {
		return true
	}
	return sameValue(o.Image, d.Image) && colorFilterEqual(o.ColorFilter, d.ColorFilter) && o.Fit == d.Fit &&
		alignmentGeometryEqual(o.Alignment, d.Alignment) && o.CenterSlice == d.CenterSlice && o.Repeat == d.Repeat &&
		o.MatchTextDirection == d.MatchTextDirection && o.Scale == d.Scale && o.Opacity == d.Opacity &&
		o.FilterQuality == d.FilterQuality && o.InvertColors == d.InvertColors && o.IsAntiAlias == d.IsAntiAlias
}

func colorFilterEqual(a, b *ui.ColorFilter) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

func alignmentGeometryEqual(a, b AlignmentGeometry) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

// Blended returns the image that paints a and b on top of each other, blended by
// t.
//
// Flutter: painting/decoration_image.dart DecorationImage.lerp
func (d *DecorationImage) Blended(a, b DecorationImageSource, t float64) DecorationImageSource {
	return &blendedDecorationImage{a: a, b: b, t: t}
}

// String describes the image.
//
// Flutter: painting/decoration_image.dart DecorationImage.toString
func (d *DecorationImage) String() string {
	props := []string{providerString(d.Image)}
	if d.ColorFilter != nil {
		props = append(props, d.ColorFilter.String())
	}
	if d.Fit != BoxFitUnspecified && (d.Fit != BoxFitFill || d.CenterSlice.IsNone()) && (d.Fit != BoxFitScaleDown || d.CenterSlice.IsSome()) {
		props = append(props, d.Fit.String())
	}
	props = append(props, d.Alignment.String())
	if cs, ok := d.CenterSlice.Get(); ok {
		props = append(props, "centerSlice: "+cs.String())
	}
	if d.Repeat != ImageRepeatNoRepeat {
		props = append(props, d.Repeat.String())
	}
	if d.MatchTextDirection {
		props = append(props, "match text direction")
	}
	props = append(props, "scale "+dartfmt.ToStringAsFixed(d.Scale, 1), "opacity "+dartfmt.ToStringAsFixed(d.Opacity, 1), d.FilterQuality.String())
	if d.InvertColors {
		props = append(props, "invert colors")
	}
	if d.IsAntiAlias {
		props = append(props, "use anti-aliasing")
	}
	return "DecorationImage(" + strings.Join(props, ", ") + ")"
}

func providerString(p ImageProvider) string {
	if s, ok := p.(interface{ String() string }); ok {
		return s.String()
	}
	return foundation.RuntimeTypeName(p)
}

// DecorationImagePaintOpts are the optional parameters of
// DecorationImagePainter.Paint.
//
// Flutter: painting/decoration_image.dart DecorationImagePainter.paint
type DecorationImagePaintOpts struct {
	// Blend multiplies the opacity of the image; the default is 1.
	Blend foundation.Opt[float64]
	// BlendMode is how the image combines with what is below; the default is
	// srcOver.
	BlendMode ui.BlendMode
}

type decorationImagePainter struct {
	details   *DecorationImage
	onChanged func()

	imageStream *ImageStream
	listener    *ImageStreamListener
	image       *ImageInfo
}

// Paint draws the image, resolving it first with the configuration's
// PaintingBinding. In debug builds MatchTextDirection without a text direction
// in the configuration panics with the tag DecorationImageNoTextDirection, and a
// configuration without a binding with ImageConfigurationNoBinding.
//
// Flutter: painting/decoration_image.dart DecorationImagePainter.paint
func (p *decorationImagePainter) Paint(canvas *ui.Canvas, rect ui.Rect, clipPath *ui.Path, configuration ImageConfiguration, opts ...DecorationImagePaintOpts) {
	o := firstOpts(opts)
	blend := o.Blend.OrElse(1.0)
	blendMode := o.BlendMode
	if blendMode == ui.BlendModeUnspecified {
		blendMode = ui.BlendModeSrcOver
	}
	flipHorizontally := false
	if p.details.MatchTextDirection {
		if foundation.DebugMode && configuration.TextDirection == ui.TextDirectionUnspecified {
			panic(errors.New("DecorationImageNoTextDirection",
				"DecorationImage.matchTextDirection can only be used when a TextDirection is available. "+
					"When DecorationImagePainter.paint() was called, there was no text direction provided in the ImageConfiguration object to match.",
				"decorationImage", p.details.String(), "configuration", configuration.String()))
		}
		if configuration.TextDirection == ui.TextDirectionRTL {
			flipHorizontally = true
		}
	}
	if configuration.Binding == nil {
		panic(errors.New("ImageConfigurationNoBinding", "the ImageConfiguration given to a DecorationImage painter has no PaintingBinding"))
	}
	newStream := ImageProviderResolve(configuration.Binding, p.details.Image, configuration)
	if p.imageStream == nil || newStream.Key() != p.imageStream.Key() {
		if p.imageStream != nil && p.listener != nil {
			p.imageStream.RemoveListener(p.listener)
		}
		p.listener = NewImageStreamListener(p.handleImage, ImageStreamListenerOpts{OnError: p.details.OnError})
		p.imageStream = newStream
		p.imageStream.AddListener(p.listener)
	}
	if p.image == nil {
		return
	}
	if clipPath != nil {
		canvas.Save()
		canvas.ClipPath(clipPath)
	}
	PaintImage(canvas, rect, p.image.Image, PaintImageOpts{
		DebugImageLabel:  p.image.DebugLabel,
		Scale:            foundation.Some(p.details.Scale * p.image.Scale),
		ColorFilter:      p.details.ColorFilter,
		Fit:              p.details.Fit,
		Alignment:        foundation.Some(p.details.Alignment.Resolve(configuration.TextDirection)),
		CenterSlice:      p.details.CenterSlice,
		Repeat:           p.details.Repeat,
		FlipHorizontally: flipHorizontally,
		Opacity:          foundation.Some(p.details.Opacity * blend),
		FilterQuality:    p.details.FilterQuality,
		InvertColors:     p.details.InvertColors,
		IsAntiAlias:      p.details.IsAntiAlias,
		BlendMode:        blendMode,
	})
	if clipPath != nil {
		canvas.Restore()
	}
}

func (p *decorationImagePainter) handleImage(value *ImageInfo, synchronousCall bool) {
	if p.image == value {
		return
	}
	if p.image != nil && p.image.IsCloneOf(value) {
		value.Dispose()
		return
	}
	if p.image != nil {
		p.image.Dispose()
	}
	p.image = value
	if !synchronousCall {
		p.onChanged()
	}
}

// Dispose removes the listener and releases the image.
//
// Flutter: painting/decoration_image.dart DecorationImagePainter.dispose
func (p *decorationImagePainter) Dispose() {
	if p.imageStream != nil && p.listener != nil {
		p.imageStream.RemoveListener(p.listener)
	}
	if p.image != nil {
		p.image.Dispose()
	}
	p.image = nil
}

// PaintImageOpts are the optional parameters of PaintImage.
//
// Flutter: painting/decoration_image.dart paintImage
type PaintImageOpts struct {
	// DebugImageLabel is a description of the image, for debugging.
	DebugImageLabel string
	// Scale is the number of image pixels per logical pixel; the default is 1.
	Scale foundation.Opt[float64]
	// Opacity is the opacity of the image; the default is 1.
	Opacity foundation.Opt[float64]
	// ColorFilter is applied to the image before painting it.
	ColorFilter *ui.ColorFilter
	// Fit is how the image is inscribed into the rect; unspecified is scaleDown,
	// or fill with a CenterSlice.
	Fit BoxFit
	// Alignment is how to align the image within the rect; the default is center.
	Alignment foundation.Opt[Alignment]
	// CenterSlice is the center slice of a nine-patch image.
	CenterSlice foundation.Opt[ui.Rect]
	// Repeat is how to paint the portions of the rect the image doesn't cover.
	Repeat ImageRepeat
	// FlipHorizontally paints the image mirrored.
	FlipHorizontally bool
	// InvertColors inverts the colors of the image.
	InvertColors bool
	// FilterQuality is the quality of sampling; the default is medium.
	FilterQuality ui.FilterQuality
	// IsAntiAlias says whether to paint with anti-aliasing.
	IsAntiAlias bool
	// BlendMode is how the image combines with what is below; the default is
	// srcOver.
	BlendMode ui.BlendMode
}

// PaintImage paints an image into the given rectangle on the canvas. In debug
// builds a CenterSlice with a fit of none or cover panics with the tag
// PaintImageCenterSliceFit, and a fit that doesn't show the whole image with a
// CenterSlice with PaintImageCenterSliceCropped. A disposed image panics with the
// tag PaintImageDisposed.
//
// Flutter: painting/decoration_image.dart paintImage
func PaintImage(canvas *ui.Canvas, rect ui.Rect, image *ui.Image, opts ...PaintImageOpts) {
	o := firstOpts(opts)
	if foundation.DebugMode && image.IsDisposed() {
		panic(errors.New("PaintImageDisposed", "Cannot paint an image that is disposed. The caller of paintImage is expected to wait to dispose the image until after painting has completed."))
	}
	if rect.IsEmpty() {
		return
	}
	scale := o.Scale.OrElse(1.0)
	opacity := o.Opacity.OrElse(1.0)
	alignment := o.Alignment.OrElse(AlignmentCenter)
	filterQuality := o.FilterQuality
	if filterQuality == ui.FilterQualityUnspecified {
		filterQuality = ui.FilterQualityMedium
	}
	blendMode := o.BlendMode
	if blendMode == ui.BlendModeUnspecified {
		blendMode = ui.BlendModeSrcOver
	}
	repeat := o.Repeat
	if repeat == ImageRepeatUnspecified {
		repeat = ImageRepeatNoRepeat
	}

	outputSize := rect.Size()
	inputSize := ui.Size{Width: float64(image.Width()), Height: float64(image.Height())}
	var sliceBorder ui.Offset
	centerSlice, hasSlice := o.CenterSlice.Get()
	if hasSlice {
		sliceBorder = inputSize.Div(scale).Sub(centerSlice.Size())
		outputSize = outputSize.SubOffset(sliceBorder)
		inputSize = inputSize.SubOffset(sliceBorder.Mul(scale))
	}
	fit := o.Fit
	if fit == BoxFitUnspecified {
		if hasSlice {
			fit = BoxFitFill
		} else {
			fit = BoxFitScaleDown
		}
	}
	if foundation.DebugMode && hasSlice && (fit == BoxFitNone || fit == BoxFitCover) {
		panic(errors.New("PaintImageCenterSliceFit", "a centerSlice needs a BoxFit that shows the whole image", "fit", fit.String()))
	}
	fittedSizes := ApplyBoxFit(fit, inputSize.Div(scale), outputSize)
	sourceSize := fittedSizes.Source.Mul(scale)
	destinationSize := fittedSizes.Destination
	if hasSlice {
		outputSize = outputSize.Add(sliceBorder)
		destinationSize = destinationSize.Add(sliceBorder)
		if foundation.DebugMode && sourceSize != inputSize {
			panic(errors.New("PaintImageCenterSliceCropped", "centerSlice was used with a BoxFit that does not guarantee that the image is fully visible"))
		}
	}
	if repeat != ImageRepeatNoRepeat && destinationSize == outputSize {
		repeat = ImageRepeatNoRepeat
	}
	paint := ui.NewPaint()
	paint.SetIsAntiAlias(o.IsAntiAlias)
	if o.ColorFilter != nil {
		paint.SetColorFilter(o.ColorFilter)
	}
	paint.SetColor(ui.NewColorFromRGBO(0, 0, 0, ui.ClampDouble(opacity, 0.0, 1.0)))
	paint.SetFilterQuality(filterQuality)
	paint.SetInvertColors(o.InvertColors)
	paint.SetBlendMode(blendMode)
	halfWidthDelta := (outputSize.Width - destinationSize.Width) / 2.0
	halfHeightDelta := (outputSize.Height - destinationSize.Height) / 2.0
	ax := alignment.X
	if o.FlipHorizontally {
		ax = -alignment.X
	}
	dx := halfWidthDelta + ax*halfWidthDelta
	dy := halfHeightDelta + alignment.Y*halfHeightDelta
	destinationPosition := rect.TopLeft().Add(ui.Offset{Dx: dx, Dy: dy})
	destinationRect := destinationPosition.And(destinationSize)

	needSave := hasSlice || repeat != ImageRepeatNoRepeat || o.FlipHorizontally
	if needSave {
		canvas.Save()
	}
	if repeat != ImageRepeatNoRepeat {
		canvas.ClipRect(rect)
	}
	if o.FlipHorizontally {
		fdx := -(rect.Left + rect.Width()/2.0)
		canvas.Translate(-fdx, 0.0)
		canvas.Scale(-1.0, 1.0)
		canvas.Translate(fdx, 0.0)
	}
	if !hasSlice {
		sourceRect := alignment.Inscribe(sourceSize, (ui.Offset{}).And(inputSize))
		if repeat == ImageRepeatNoRepeat {
			canvas.DrawImageRect(image, sourceRect, destinationRect, paint)
		} else {
			for _, tile := range generateImageTileRects(rect, destinationRect, repeat) {
				canvas.DrawImageRect(image, sourceRect, tile, paint)
			}
		}
	} else {
		canvas.ScaleUniform(1 / scale)
		if repeat == ImageRepeatNoRepeat {
			canvas.DrawImageNine(image, scaleRect(centerSlice, scale), scaleRect(destinationRect, scale), paint)
		} else {
			for _, tile := range generateImageTileRects(rect, destinationRect, repeat) {
				canvas.DrawImageNine(image, scaleRect(centerSlice, scale), scaleRect(tile, scale), paint)
			}
		}
	}
	if needSave {
		canvas.Restore()
	}
}

func generateImageTileRects(outputRect, fundamentalRect ui.Rect, repeat ImageRepeat) []ui.Rect {
	startX, startY, stopX, stopY := 0, 0, 0, 0
	strideX, strideY := fundamentalRect.Width(), fundamentalRect.Height()
	if repeat == ImageRepeatRepeat || repeat == ImageRepeatRepeatX {
		startX = int(math.Floor((outputRect.Left - fundamentalRect.Left) / strideX))
		stopX = int(math.Ceil((outputRect.Right - fundamentalRect.Right) / strideX))
	}
	if repeat == ImageRepeatRepeat || repeat == ImageRepeatRepeatY {
		startY = int(math.Floor((outputRect.Top - fundamentalRect.Top) / strideY))
		stopY = int(math.Ceil((outputRect.Bottom - fundamentalRect.Bottom) / strideY))
	}
	var out []ui.Rect
	for i := startX; i <= stopX; i++ {
		for j := startY; j <= stopY; j++ {
			out = append(out, fundamentalRect.Shift(ui.Offset{Dx: float64(i) * strideX, Dy: float64(j) * strideY}))
		}
	}
	return out
}

func scaleRect(r ui.Rect, scale float64) ui.Rect {
	return ui.NewRectFromLTRB(r.Left*scale, r.Top*scale, r.Right*scale, r.Bottom*scale)
}

// blendedDecorationImage is Flutter's _BlendedDecorationImage: it paints a and b,
// either of which may be nil, blended by t.
type blendedDecorationImage struct {
	a, b DecorationImageSource
	t    float64
}

func (d *blendedDecorationImage) CreatePainter(onChanged func()) DecorationImagePainter {
	p := &blendedDecorationImagePainter{t: d.t}
	if d.a != nil {
		p.a = d.a.CreatePainter(onChanged)
	}
	if d.b != nil {
		p.b = d.b.CreatePainter(onChanged)
	}
	return p
}

func (d *blendedDecorationImage) Blended(a, b DecorationImageSource, t float64) DecorationImageSource {
	return &blendedDecorationImage{a: a, b: b, t: t}
}

func (d *blendedDecorationImage) Equal(other DecorationImageSource) bool {
	o, ok := other.(*blendedDecorationImage)
	if !ok || o == nil {
		return false
	}
	return o == d || (decorationImageEqual(o.a, d.a) && decorationImageEqual(o.b, d.b) && o.t == d.t)
}

type blendedDecorationImagePainter struct {
	a, b DecorationImagePainter
	t    float64
}

func (p *blendedDecorationImagePainter) Paint(canvas *ui.Canvas, rect ui.Rect, clipPath *ui.Path, configuration ImageConfiguration, opts ...DecorationImagePaintOpts) {
	o := firstOpts(opts)
	blend := o.Blend.OrElse(1.0)
	blendMode := o.BlendMode
	if blendMode == ui.BlendModeUnspecified {
		blendMode = ui.BlendModeSrcOver
	}
	canvas.SaveLayer(nil, ui.NewPaint())
	if p.a != nil {
		p.a.Paint(canvas, rect, clipPath, configuration, DecorationImagePaintOpts{Blend: foundation.Some(blend * (1.0 - p.t)), BlendMode: blendMode})
	}
	if p.b != nil {
		mode := blendMode
		if p.a != nil {
			mode = ui.BlendModePlus
		}
		p.b.Paint(canvas, rect, clipPath, configuration, DecorationImagePaintOpts{Blend: foundation.Some(blend * p.t), BlendMode: mode})
	}
	canvas.Restore()
}

func (p *blendedDecorationImagePainter) Dispose() {
	if p.a != nil {
		p.a.Dispose()
	}
	if p.b != nil {
		p.b.Dispose()
	}
}
