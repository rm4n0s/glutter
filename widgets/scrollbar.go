// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"math"
	"reflect"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

const (
	minThumbExtent      = 18.0
	minInteractiveSize  = 48.0
	scrollbarThickness  = 6.0
	scrollbarFade       = 300 * time.Millisecond
	scrollbarTimeToFade = 600 * time.Millisecond
)

// ScrollbarOrientation is where a scrollbar sits relative to its scrollable.
//
// Flutter: widgets/scrollbar.dart ScrollbarOrientation
type ScrollbarOrientation int

const (
	// ScrollbarOrientationUnspecified is the zero value: the default placement,
	// the right (or left, in right-to-left text) side of a vertical scrollable
	// and the bottom of a horizontal one.
	//
	// Flutter: no counterpart
	ScrollbarOrientationUnspecified ScrollbarOrientation = iota
	// ScrollbarOrientationLeft places a vertical scrollbar on the left side.
	//
	// Flutter: widgets/scrollbar.dart ScrollbarOrientation.left
	ScrollbarOrientationLeft
	// ScrollbarOrientationRight places a vertical scrollbar on the right side.
	//
	// Flutter: widgets/scrollbar.dart ScrollbarOrientation.right
	ScrollbarOrientationRight
	// ScrollbarOrientationTop places a horizontal scrollbar at the top.
	//
	// Flutter: widgets/scrollbar.dart ScrollbarOrientation.top
	ScrollbarOrientationTop
	// ScrollbarOrientationBottom places a horizontal scrollbar at the bottom.
	//
	// Flutter: widgets/scrollbar.dart ScrollbarOrientation.bottom
	ScrollbarOrientationBottom
)

// String is the name with the enum, such as "ScrollbarOrientation.left".
//
// Flutter: widgets/scrollbar.dart ScrollbarOrientation
func (o ScrollbarOrientation) String() string {
	switch o {
	case ScrollbarOrientationLeft:
		return "ScrollbarOrientation.left"
	case ScrollbarOrientationRight:
		return "ScrollbarOrientation.right"
	case ScrollbarOrientationTop:
		return "ScrollbarOrientation.top"
	case ScrollbarOrientationBottom:
		return "ScrollbarOrientation.bottom"
	}
	return "ScrollbarOrientation.unspecified"
}

func (o ScrollbarOrientation) isVertical() bool {
	return o == ScrollbarOrientationLeft || o == ScrollbarOrientationRight
}

// ScrollbarPainterOpts are the optional parameters of NewScrollbarPainter.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.new
type ScrollbarPainterOpts struct {
	// TrackColor is the color of the track; the default is transparent.
	TrackColor ui.Color
	// TrackBorderColor is the color of the border of the track; the default is
	// transparent.
	TrackBorderColor ui.Color
	// TextDirection is the direction of the text, to place the scrollbar and
	// resolve the padding.
	TextDirection ui.TextDirection
	// Thickness is the thickness of the thumb; the default is 6.
	Thickness foundation.Opt[float64]
	// Padding is the padding around the scrollbar; the default is zero.
	Padding painting.EdgeInsetsGeometry
	// MainAxisMargin is the margin at the ends of the track; the default is 0.
	MainAxisMargin float64
	// CrossAxisMargin is the margin between the thumb and the edge; the default
	// is 0.
	CrossAxisMargin float64
	// Radius is the corner radius of the thumb.
	Radius foundation.Opt[ui.Radius]
	// TrackRadius is the corner radius of the track.
	TrackRadius foundation.Opt[ui.Radius]
	// Shape is the shape of the thumb, instead of a radius.
	Shape painting.OutlinedBorder
	// MinLength is the shortest the thumb gets; the default is 18.
	MinLength foundation.Opt[float64]
	// MinOverscrollLength is the shortest the thumb gets while it overscrolls;
	// the default is MinLength.
	MinOverscrollLength foundation.Opt[float64]
	// ScrollbarOrientation is the side of the scrollable the scrollbar sits on.
	ScrollbarOrientation ScrollbarOrientation
	// IgnorePointer makes the scrollbar ignore the pointer.
	IgnorePointer bool
}

// ScrollbarPainter paints a scrollbar: the thumb, and optionally the track and its
// border. It fades with the opacity animation, and updates from the scroll
// metrics. A RawScrollbar uses it.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter
type ScrollbarPainter struct {
	foundation.ChangeNotifier

	color                   ui.Color
	trackColor              ui.Color
	trackBorderColor        ui.Color
	trackRadius             foundation.Opt[ui.Radius]
	textDirection           ui.TextDirection
	thickness               float64
	fadeoutOpacityAnimation animation.Animation[float64]
	fadeoutHandle           foundation.ListenerHandle
	mainAxisMargin          float64
	crossAxisMargin         float64
	radius                  foundation.Opt[ui.Radius]
	shape                   painting.OutlinedBorder
	padding                 painting.EdgeInsetsGeometry
	resolvedPadding         painting.EdgeInsets
	minLength               float64
	minOverscrollLength     float64
	scrollbarOrientation    ScrollbarOrientation
	ignorePointer           bool

	trackRect, thumbRect foundation.Opt[ui.Rect]
	thumbOffset          float64
	thumbExtent          float64
	lastMetrics          ScrollMetrics
	lastAxisDirection    painting.AxisDirection
	hasLastAxisDirection bool
}

// NewScrollbarPainter creates the painter; the color is the color of the thumb
// and fadeoutOpacityAnimation fades the whole scrollbar. In debug builds it
// panics with the tag ScrollbarPainterInvalid when the arguments are not valid
// (a radius and a shape, a negative length, a negative padding, or a directional
// padding without text direction).
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.new
func NewScrollbarPainter(color ui.Color, fadeoutOpacityAnimation animation.Animation[float64], opts ...ScrollbarPainterOpts) *ScrollbarPainter {
	o := firstOpts(opts)
	minLength := o.MinLength.OrElse(minThumbExtent)
	padding := o.Padding
	if padding == nil {
		padding = painting.EdgeInsetsZero
	}
	if foundation.DebugMode {
		invalid := func(why string) {
			panic(errors.New("ScrollbarPainterInvalid", why))
		}
		switch {
		case o.Radius.IsSome() && !isNil(o.Shape):
			invalid("a scrollbar painter can have a radius or a shape, not both")
		case minLength < 0:
			invalid("the minimum length of a scrollbar must not be negative")
		case o.MinOverscrollLength.IsSome() && o.MinOverscrollLength.Value() > minLength:
			invalid("the minimum overscroll length of a scrollbar must not be more than its minimum length")
		case o.MinOverscrollLength.IsSome() && o.MinOverscrollLength.Value() < 0:
			invalid("the minimum overscroll length of a scrollbar must not be negative")
		case !padding.IsNonNegative():
			invalid("the padding of a scrollbar must not be negative")
		}
		if _, directional := padding.(painting.EdgeInsetsDirectional); directional && o.TextDirection == ui.TextDirectionUnspecified {
			invalid("a text direction must be provided when using EdgeInsetsDirectional for padding")
		}
	}
	p := &ScrollbarPainter{
		color: color, trackColor: o.TrackColor, trackBorderColor: o.TrackBorderColor, textDirection: o.TextDirection,
		thickness: o.Thickness.OrElse(scrollbarThickness), fadeoutOpacityAnimation: fadeoutOpacityAnimation,
		mainAxisMargin: o.MainAxisMargin, crossAxisMargin: o.CrossAxisMargin, radius: o.Radius, trackRadius: o.TrackRadius,
		shape: o.Shape, padding: padding, minLength: minLength, minOverscrollLength: o.MinOverscrollLength.OrElse(minLength),
		scrollbarOrientation: o.ScrollbarOrientation, ignorePointer: o.IgnorePointer,
	}
	p.resolvedPadding = p.resolvePadding()
	p.fadeoutHandle = fadeoutOpacityAnimation.AddListener(p.NotifyListeners)
	return p
}

func (p *ScrollbarPainter) resolvePadding() painting.EdgeInsets {
	if d, ok := p.padding.(painting.EdgeInsetsDirectional); ok && p.textDirection == ui.TextDirectionUnspecified {
		return d.Resolve(ui.TextDirectionLTR)
	}
	return p.padding.Resolve(p.textDirection)
}

// Color is the color of the thumb.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.color
func (p *ScrollbarPainter) Color() ui.Color { return p.color }

// SetColor sets the color of the thumb and repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.color
func (p *ScrollbarPainter) SetColor(value ui.Color) {
	if p.color == value {
		return
	}
	p.color = value
	p.NotifyListeners()
}

// TrackColor is the color of the track.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.trackColor
func (p *ScrollbarPainter) TrackColor() ui.Color { return p.trackColor }

// SetTrackColor sets the color of the track and repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.trackColor
func (p *ScrollbarPainter) SetTrackColor(value ui.Color) {
	if p.trackColor == value {
		return
	}
	p.trackColor = value
	p.NotifyListeners()
}

// TrackBorderColor is the color of the border of the track.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.trackBorderColor
func (p *ScrollbarPainter) TrackBorderColor() ui.Color { return p.trackBorderColor }

// SetTrackBorderColor sets the color of the border of the track and repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.trackBorderColor
func (p *ScrollbarPainter) SetTrackBorderColor(value ui.Color) {
	if p.trackBorderColor == value {
		return
	}
	p.trackBorderColor = value
	p.NotifyListeners()
}

// TrackRadius is the corner radius of the track.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.trackRadius
func (p *ScrollbarPainter) TrackRadius() foundation.Opt[ui.Radius] { return p.trackRadius }

// SetTrackRadius sets the corner radius of the track and repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.trackRadius
func (p *ScrollbarPainter) SetTrackRadius(value foundation.Opt[ui.Radius]) {
	if p.trackRadius == value {
		return
	}
	p.trackRadius = value
	p.NotifyListeners()
}

// TextDirection is the direction of the text.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.textDirection
func (p *ScrollbarPainter) TextDirection() ui.TextDirection { return p.textDirection }

// SetTextDirection sets the direction of the text and repaints. In debug builds
// it panics with the tag ScrollbarPainterInvalid when it is unspecified.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.textDirection
func (p *ScrollbarPainter) SetTextDirection(value ui.TextDirection) {
	if foundation.DebugMode && value == ui.TextDirectionUnspecified {
		panic(errors.New("ScrollbarPainterInvalid", "the text direction of a scrollbar painter must be set"))
	}
	if p.textDirection == value {
		return
	}
	p.textDirection = value
	p.resolvedPadding = p.resolvePadding()
	p.NotifyListeners()
}

// Thickness is the thickness of the thumb.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.thickness
func (p *ScrollbarPainter) Thickness() float64 { return p.thickness }

// SetThickness sets the thickness of the thumb and repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.thickness
func (p *ScrollbarPainter) SetThickness(value float64) {
	if p.thickness == value {
		return
	}
	p.thickness = value
	p.NotifyListeners()
}

// FadeoutOpacityAnimation is the animation of the opacity of the scrollbar.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.fadeoutOpacityAnimation
func (p *ScrollbarPainter) FadeoutOpacityAnimation() animation.Animation[float64] {
	return p.fadeoutOpacityAnimation
}

// MainAxisMargin is the margin at the ends of the track.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.mainAxisMargin
func (p *ScrollbarPainter) MainAxisMargin() float64 { return p.mainAxisMargin }

// SetMainAxisMargin sets the margin at the ends of the track and repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.mainAxisMargin
func (p *ScrollbarPainter) SetMainAxisMargin(value float64) {
	if p.mainAxisMargin == value {
		return
	}
	p.mainAxisMargin = value
	p.NotifyListeners()
}

// CrossAxisMargin is the margin between the thumb and the edge.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.crossAxisMargin
func (p *ScrollbarPainter) CrossAxisMargin() float64 { return p.crossAxisMargin }

// SetCrossAxisMargin sets the margin between the thumb and the edge and
// repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.crossAxisMargin
func (p *ScrollbarPainter) SetCrossAxisMargin(value float64) {
	if p.crossAxisMargin == value {
		return
	}
	p.crossAxisMargin = value
	p.NotifyListeners()
}

// Radius is the corner radius of the thumb.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.radius
func (p *ScrollbarPainter) Radius() foundation.Opt[ui.Radius] { return p.radius }

// SetRadius sets the corner radius of the thumb and repaints. In debug builds it
// panics with the tag ScrollbarPainterInvalid when the painter has a shape.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.radius
func (p *ScrollbarPainter) SetRadius(value foundation.Opt[ui.Radius]) {
	if foundation.DebugMode && !isNil(p.shape) && value.IsSome() {
		panic(errors.New("ScrollbarPainterInvalid", "a scrollbar painter can have a radius or a shape, not both"))
	}
	if p.radius == value {
		return
	}
	p.radius = value
	p.NotifyListeners()
}

// Shape is the shape of the thumb.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.shape
func (p *ScrollbarPainter) Shape() painting.OutlinedBorder { return p.shape }

// SetShape sets the shape of the thumb and repaints. In debug builds it panics
// with the tag ScrollbarPainterInvalid when the painter has a radius.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.shape
func (p *ScrollbarPainter) SetShape(value painting.OutlinedBorder) {
	if foundation.DebugMode && p.radius.IsSome() && !isNil(value) {
		panic(errors.New("ScrollbarPainterInvalid", "a scrollbar painter can have a radius or a shape, not both"))
	}
	if p.shape == value {
		return
	}
	p.shape = value
	p.NotifyListeners()
}

// Padding is the padding around the scrollbar.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.padding
func (p *ScrollbarPainter) Padding() painting.EdgeInsetsGeometry { return p.padding }

// SetPadding sets the padding around the scrollbar and repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.padding
func (p *ScrollbarPainter) SetPadding(value painting.EdgeInsetsGeometry) {
	if p.padding.Equal(value) {
		return
	}
	p.padding = value
	p.resolvedPadding = p.resolvePadding()
	p.NotifyListeners()
}

// MinLength is the shortest the thumb gets.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.minLength
func (p *ScrollbarPainter) MinLength() float64 { return p.minLength }

// SetMinLength sets the shortest the thumb gets and repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.minLength
func (p *ScrollbarPainter) SetMinLength(value float64) {
	if p.minLength == value {
		return
	}
	p.minLength = value
	p.NotifyListeners()
}

// MinOverscrollLength is the shortest the thumb gets while it overscrolls.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.minOverscrollLength
func (p *ScrollbarPainter) MinOverscrollLength() float64 { return p.minOverscrollLength }

// SetMinOverscrollLength sets the shortest the thumb gets while it overscrolls
// and repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.minOverscrollLength
func (p *ScrollbarPainter) SetMinOverscrollLength(value float64) {
	if p.minOverscrollLength == value {
		return
	}
	p.minOverscrollLength = value
	p.NotifyListeners()
}

// ScrollbarOrientation is the side the scrollbar sits on.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.scrollbarOrientation
func (p *ScrollbarPainter) ScrollbarOrientation() ScrollbarOrientation { return p.scrollbarOrientation }

// SetScrollbarOrientation sets the side the scrollbar sits on and repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.scrollbarOrientation
func (p *ScrollbarPainter) SetScrollbarOrientation(value ScrollbarOrientation) {
	if p.scrollbarOrientation == value {
		return
	}
	p.scrollbarOrientation = value
	p.NotifyListeners()
}

// IgnorePointer is whether the scrollbar ignores the pointer.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.ignorePointer
func (p *ScrollbarPainter) IgnorePointer() bool { return p.ignorePointer }

// SetIgnorePointer sets whether the scrollbar ignores the pointer and repaints.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.ignorePointer
func (p *ScrollbarPainter) SetIgnorePointer(value bool) {
	if p.ignorePointer == value {
		return
	}
	p.ignorePointer = value
	p.NotifyListeners()
}

// ThumbOffset is the position of the thumb along the track, from the start of the
// scrollbar's box, as of the last paint.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter._thumbOffset
func (p *ScrollbarPainter) ThumbOffset() float64 { return p.thumbOffset }

func (p *ScrollbarPainter) isVertical() bool {
	return p.lastAxisDirection == painting.AxisDirectionDown || p.lastAxisDirection == painting.AxisDirectionUp
}

func (p *ScrollbarPainter) isReversed() bool {
	return p.lastAxisDirection == painting.AxisDirectionUp || p.lastAxisDirection == painting.AxisDirectionLeft
}

func (p *ScrollbarPainter) totalTrackMainAxisOffsets() float64 {
	if p.isVertical() {
		return p.resolvedPadding.Vertical()
	}
	return p.resolvedPadding.Horizontal()
}

func (p *ScrollbarPainter) trackExtent() float64 {
	return p.lastMetrics.ViewportDimension() - p.totalTrackMainAxisOffsets()
}

func (p *ScrollbarPainter) traversableTrackExtent() float64 {
	return p.trackExtent() - 2*p.mainAxisMargin
}

func (p *ScrollbarPainter) leadingTrackMainAxisOffset() float64 {
	switch p.resolvedOrientation() {
	case ScrollbarOrientationLeft, ScrollbarOrientationRight:
		return p.resolvedPadding.Top
	}
	return p.resolvedPadding.Left
}

func (p *ScrollbarPainter) leadingThumbMainAxisOffset() float64 {
	return p.leadingTrackMainAxisOffset() + p.mainAxisMargin
}

func (p *ScrollbarPainter) beforeExtent() float64 {
	if p.isReversed() {
		return p.lastMetrics.ExtentAfter()
	}
	return p.lastMetrics.ExtentBefore()
}

func (p *ScrollbarPainter) afterExtent() float64 {
	if p.isReversed() {
		return p.lastMetrics.ExtentBefore()
	}
	return p.lastMetrics.ExtentAfter()
}

func (p *ScrollbarPainter) totalContentExtent() float64 {
	return p.lastMetrics.MaxScrollExtent() - p.lastMetrics.MinScrollExtent() + p.lastMetrics.ViewportDimension()
}

func (p *ScrollbarPainter) lastMetricsAreScrollable() bool {
	return p.lastMetrics.MinScrollExtent() != p.lastMetrics.MaxScrollExtent()
}

func (p *ScrollbarPainter) resolvedOrientation() ScrollbarOrientation {
	if p.scrollbarOrientation == ScrollbarOrientationUnspecified {
		if p.isVertical() {
			if p.textDirection == ui.TextDirectionLTR {
				return ScrollbarOrientationRight
			}
			return ScrollbarOrientationLeft
		}
		return ScrollbarOrientationBottom
	}
	return p.scrollbarOrientation
}

func (p *ScrollbarPainter) setThumbExtent() {
	// Thumb extent reflects fraction of content visible, as long as this
	// isn't less than the absolute minimum size.
	// _totalContentExtent >= viewportDimension, so (_totalContentExtent -
	// _mainAxisPadding) > 0
	fractionVisible := ui.ClampDouble(
		(p.lastMetrics.ExtentInside()-p.totalTrackMainAxisOffsets())/(p.totalContentExtent()-p.totalTrackMainAxisOffsets()), 0, 1)
	thumbExtent := math.Max(math.Min(p.traversableTrackExtent(), p.minOverscrollLength), p.traversableTrackExtent()*fractionVisible)

	fractionOverscrolled := 1.0 - p.lastMetrics.ExtentInside()/p.lastMetrics.ViewportDimension()
	safeMinLength := math.Min(p.minLength, p.traversableTrackExtent())
	var newMinLength float64
	if p.beforeExtent() > 0 && p.afterExtent() > 0 {
		// Thumb extent is no smaller than minLength if scrolling normally.
		newMinLength = safeMinLength
	} else {
		// The thumb is overscrolling, so its length may shrink to
		// minOverscrollLength from minLength.
		newMinLength = safeMinLength * (1.0 - ui.ClampDouble(fractionOverscrolled, 0, 0.2)/0.2)
	}
	// The `thumbExtent` should be no greater than `trackSize`, otherwise the
	// scrollbar may scroll towards the wrong direction.
	p.thumbExtent = ui.ClampDouble(thumbExtent, newMinLength, p.traversableTrackExtent())
}

func needPaint(metrics ScrollMetrics) bool {
	return !isNil(metrics) && metrics.MaxScrollExtent()-metrics.MinScrollExtent() > foundation.PrecisionErrorTolerance
}

// Update updates the scrollbar from the metrics of the scrollable and repaints if
// the extents or the axis changed and the scrollbar is, or was, visible.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.update
func (p *ScrollbarPainter) Update(metrics ScrollMetrics, axisDirection painting.AxisDirection) {
	if !isNil(p.lastMetrics) &&
		p.lastMetrics.ExtentBefore() == metrics.ExtentBefore() &&
		p.lastMetrics.ExtentInside() == metrics.ExtentInside() &&
		p.lastMetrics.ExtentAfter() == metrics.ExtentAfter() &&
		p.hasLastAxisDirection && p.lastAxisDirection == axisDirection {
		return
	}
	oldMetrics := p.lastMetrics
	p.lastMetrics = metrics
	p.lastAxisDirection, p.hasLastAxisDirection = axisDirection, true
	if !needPaint(oldMetrics) && !needPaint(metrics) {
		return
	}
	p.NotifyListeners()
}

// UpdateThickness changes the thickness and the radius together, as the Material
// scrollbar does when the pointer hovers it.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.updateThickness
func (p *ScrollbarPainter) UpdateThickness(nextThickness float64, nextRadius ui.Radius) {
	p.SetThickness(nextThickness)
	p.SetRadius(foundation.Some(nextRadius))
}

func (p *ScrollbarPainter) paintThumb() *ui.Paint {
	paint := ui.NewPaint()
	paint.SetColor(p.color.WithOpacity(p.color.Opacity() * p.fadeoutOpacityAnimation.Value()))
	return paint
}

func (p *ScrollbarPainter) paintTrack(isBorder bool) *ui.Paint {
	paint := ui.NewPaint()
	if isBorder {
		paint.SetColor(p.trackBorderColor.WithOpacity(p.trackBorderColor.Opacity() * p.fadeoutOpacityAnimation.Value()))
		paint.SetStyle(ui.PaintingStyleStroke)
		paint.SetStrokeWidth(1.0)
		return paint
	}
	paint.SetColor(p.trackColor.WithOpacity(p.trackColor.Opacity() * p.fadeoutOpacityAnimation.Value()))
	return paint
}

func (p *ScrollbarPainter) paintScrollbar(canvas *ui.Canvas, size ui.Size) {
	if foundation.DebugMode && p.textDirection == ui.TextDirectionUnspecified {
		panic(errors.New("ScrollbarPainterInvalid", "a TextDirection must be provided before a Scrollbar can be painted"))
	}
	var x, y float64
	var thumbSize, trackSize ui.Size
	var trackOffset, borderStart, borderEnd ui.Offset
	orientation := p.resolvedOrientation()
	if foundation.DebugMode && orientation.isVertical() != p.isVertical() {
		panic(errors.New("ScrollbarOrientationInvalid", "the given ScrollbarOrientation is incompatible with the current AxisDirection",
			"orientation", orientation.String(), "axisDirection", p.lastAxisDirection.String()))
	}
	pad := p.resolvedPadding
	switch orientation {
	case ScrollbarOrientationLeft:
		thumbSize = ui.Size{Width: p.thickness, Height: p.thumbExtent}
		trackSize = ui.Size{Width: p.thickness + 2*p.crossAxisMargin, Height: p.trackExtent()}
		x = p.crossAxisMargin + pad.Left
		y = p.thumbOffset
		trackOffset = ui.Offset{Dx: x - p.crossAxisMargin, Dy: p.leadingTrackMainAxisOffset()}
		borderStart = trackOffset.Add(ui.Offset{Dx: trackSize.Width})
		borderEnd = ui.Offset{Dx: trackOffset.Dx + trackSize.Width, Dy: trackOffset.Dy + p.trackExtent()}
	case ScrollbarOrientationRight:
		thumbSize = ui.Size{Width: p.thickness, Height: p.thumbExtent}
		trackSize = ui.Size{Width: p.thickness + 2*p.crossAxisMargin, Height: p.trackExtent()}
		x = size.Width - p.thickness - p.crossAxisMargin - pad.Right
		y = p.thumbOffset
		trackOffset = ui.Offset{Dx: x - p.crossAxisMargin, Dy: p.leadingTrackMainAxisOffset()}
		borderStart = trackOffset
		borderEnd = ui.Offset{Dx: trackOffset.Dx, Dy: trackOffset.Dy + p.trackExtent()}
	case ScrollbarOrientationTop:
		thumbSize = ui.Size{Width: p.thumbExtent, Height: p.thickness}
		trackSize = ui.Size{Width: p.trackExtent(), Height: p.thickness + 2*p.crossAxisMargin}
		x = p.thumbOffset
		y = p.crossAxisMargin + pad.Top
		trackOffset = ui.Offset{Dx: p.leadingTrackMainAxisOffset(), Dy: y - p.crossAxisMargin}
		borderStart = trackOffset.Add(ui.Offset{Dy: trackSize.Height})
		borderEnd = ui.Offset{Dx: trackOffset.Dx + p.trackExtent(), Dy: trackOffset.Dy + trackSize.Height}
	default: // bottom
		thumbSize = ui.Size{Width: p.thumbExtent, Height: p.thickness}
		trackSize = ui.Size{Width: p.trackExtent(), Height: p.thickness + 2*p.crossAxisMargin}
		x = p.thumbOffset
		y = size.Height - p.thickness - p.crossAxisMargin - pad.Bottom
		trackOffset = ui.Offset{Dx: p.leadingTrackMainAxisOffset(), Dy: y - p.crossAxisMargin}
		borderStart = trackOffset
		borderEnd = ui.Offset{Dx: trackOffset.Dx + p.trackExtent(), Dy: trackOffset.Dy}
	}

	// Whether we paint or not, calculating these rects allows us to hit test
	// when the scrollbar is transparent.
	p.trackRect = foundation.Some(trackOffset.And(trackSize))
	p.thumbRect = foundation.Some(ui.Offset{Dx: x, Dy: y}.And(thumbSize))

	// Paint if the opacity dictates visibility.
	if p.fadeoutOpacityAnimation.Value() != 0.0 {
		// Track.
		if r, ok := p.trackRadius.Get(); !ok {
			canvas.DrawRect(p.trackRect.Value(), p.paintTrack(false))
		} else {
			canvas.DrawRRect(ui.NewRRectFromRectAndRadius(p.trackRect.Value(), r), p.paintTrack(false))
		}
		// Track border.
		canvas.DrawLine(borderStart, borderEnd, p.paintTrack(true))
		// Thumb.
		if r, ok := p.radius.Get(); ok {
			canvas.DrawRRect(ui.NewRRectFromRectAndRadius(p.thumbRect.Value(), r), p.paintThumb())
			return
		}
		if isNil(p.shape) {
			canvas.DrawRect(p.thumbRect.Value(), p.paintThumb())
			return
		}
		// If the scrollbar shape can paint its interior, it should be used.
		if p.shape.PreferPaintInterior() {
			p.shape.PaintInterior(canvas, p.thumbRect.Value(), p.paintThumb(), p.textDirection)
		} else {
			outerPath := p.shape.GetOuterPath(p.thumbRect.Value(), p.textDirection)
			canvas.DrawPath(outerPath, p.paintThumb())
		}
		p.shape.Paint(canvas, p.thumbRect.Value(), p.textDirection)
	}
}

// Paint paints the track and the thumb when the content scrolls.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.paint
func (p *ScrollbarPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	if !p.hasLastAxisDirection || !needPaint(p.lastMetrics) {
		return
	}
	// Skip painting if there's not enough space.
	if p.traversableTrackExtent() <= 0 {
		return
	}
	// Do not paint a scrollbar if the scroll view is infinitely long.
	if math.IsInf(p.lastMetrics.MaxScrollExtent(), 0) {
		return
	}
	p.setThumbExtent()
	thumbPositionOffset := p.getScrollToTrack(p.lastMetrics, p.thumbExtent)
	p.thumbOffset = thumbPositionOffset + p.leadingThumbMainAxisOffset()
	p.paintScrollbar(canvas, size)
}

// GetTrackToScroll is the scroll offset that corresponds to the offset of the
// thumb along the track.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.getTrackToScroll
func (p *ScrollbarPainter) GetTrackToScroll(thumbOffsetLocal float64) float64 {
	scrollableExtent := p.lastMetrics.MaxScrollExtent() - p.lastMetrics.MinScrollExtent()
	thumbMovableExtent := p.traversableTrackExtent() - p.thumbExtent
	return scrollableExtent * thumbOffsetLocal / thumbMovableExtent
}

// GetThumbScrollOffset is the offset of the thumb along the track that
// corresponds to the scroll position. In debug builds it panics with the tag
// ScrollbarExtentsInfinite when the extents are not finite.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.getThumbScrollOffset
func (p *ScrollbarPainter) GetThumbScrollOffset() float64 {
	if foundation.DebugMode && !(isFinite(p.lastMetrics.MaxScrollExtent()) && isFinite(p.lastMetrics.MinScrollExtent())) {
		panic(errors.New("ScrollbarExtentsInfinite", "the scroll extents must be finite to get the scroll offset of the thumb"))
	}
	scrollableExtent := p.lastMetrics.MaxScrollExtent() - p.lastMetrics.MinScrollExtent()
	maxFraction := p.lastMetrics.MaxScrollExtent() / scrollableExtent
	minFraction := p.lastMetrics.MinScrollExtent() / scrollableExtent
	fractionPast := 0.0
	if scrollableExtent > 0 {
		fractionPast = ui.ClampDouble(p.lastMetrics.Pixels()/scrollableExtent, minFraction, maxFraction)
	}
	return fractionPast * (p.traversableTrackExtent() - p.thumbExtent)
}

// getScrollToTrack is the offset of the thumb along the track for the metrics.
func (p *ScrollbarPainter) getScrollToTrack(metrics ScrollMetrics, thumbExtent float64) float64 {
	scrollableExtent := metrics.MaxScrollExtent() - metrics.MinScrollExtent()
	fractionPast := 0.0
	if scrollableExtent > 0 {
		fractionPast = ui.ClampDouble((metrics.Pixels()-metrics.MinScrollExtent())/scrollableExtent, 0, 1)
	}
	if p.isReversed() {
		fractionPast = 1 - fractionPast
	}
	return fractionPast * (p.traversableTrackExtent() - thumbExtent)
}

// HitTest is whether the position is on the track; none before the first paint,
// and false while the scrollbar is transparent, ignores the pointer or cannot
// scroll. The scrollbar absorbs pointer signals over it with this.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.hitTest
func (p *ScrollbarPainter) HitTest(position ui.Offset) foundation.Opt[bool] {
	// There is nothing painted to hit.
	if !p.thumbRect.IsSome() {
		return foundation.Opt[bool]{}
	}
	// Interaction with the scrollbar is not allowed.
	if p.ignorePointer ||
		// There is nothing to interact with.
		p.fadeoutOpacityAnimation.Value() == 0.0 ||
		// The scrollbar has nothing to scroll.
		!p.lastMetricsAreScrollable() {
		return foundation.Some(false)
	}
	return foundation.Some(p.trackRect.Value().Contains(position))
}

// HitTestInteractive is whether the position is on the part of the scrollbar that
// the kind of pointer can interact with: the track (with a larger area around the
// thumb for a touch), including while it is transparent for a hover.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.hitTestInteractive
func (p *ScrollbarPainter) HitTestInteractive(position ui.Offset, kind ui.PointerDeviceKind, forHover ...bool) bool {
	if !p.trackRect.IsSome() {
		// We have not computed the scrollbar position yet.
		return false
	}
	if p.ignorePointer {
		return false
	}
	if !p.lastMetricsAreScrollable() {
		return false
	}
	interactiveRect := p.trackRect.Value()
	paddedRect := interactiveRect.ExpandToInclude(ui.NewRectFromCircle(p.thumbRect.Value().Center(), minInteractiveSize/2))

	// The scrollbar is not visible.
	if p.fadeoutOpacityAnimation.Value() == 0.0 {
		// Hovering is allowed for the mouse, regardless of visibility.
		if len(forHover) > 0 && forHover[0] && kind == ui.PointerDeviceKindMouse {
			return paddedRect.Contains(position)
		}
		return false
	}
	// The scrollbar is visible. Check for a touch or a mouse.
	switch kind {
	case ui.PointerDeviceKindTouch, ui.PointerDeviceKindTrackpad:
		return paddedRect.Contains(position)
	}
	return interactiveRect.Contains(position)
}

// HitTestOnlyThumbInteractive is whether the position is on the thumb (with a
// larger area around it for a touch).
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.hitTestOnlyThumbInteractive
func (p *ScrollbarPainter) HitTestOnlyThumbInteractive(position ui.Offset, kind ui.PointerDeviceKind) bool {
	if !p.thumbRect.IsSome() {
		return false
	}
	if p.ignorePointer {
		return false
	}
	// The thumb is not able to be hit when transparent.
	if p.fadeoutOpacityAnimation.Value() == 0.0 {
		return false
	}
	if !p.lastMetricsAreScrollable() {
		return false
	}
	switch kind {
	case ui.PointerDeviceKindTouch, ui.PointerDeviceKindTrackpad:
		touchThumbRect := p.thumbRect.Value().ExpandToInclude(ui.NewRectFromCircle(p.thumbRect.Value().Center(), minInteractiveSize/2))
		return touchThumbRect.Contains(position)
	}
	return p.thumbRect.Value().Contains(position)
}

// ShouldRepaint is whether any property differs from the old painter's.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.shouldRepaint
func (p *ScrollbarPainter) ShouldRepaint(oldDelegate rendering.CustomPainter) bool {
	// Should repaint if any properties changed.
	old, ok := oldDelegate.(*ScrollbarPainter)
	if !ok {
		return true
	}
	return p.color != old.color ||
		p.trackColor != old.trackColor ||
		p.trackBorderColor != old.trackBorderColor ||
		p.textDirection != old.textDirection ||
		p.thickness != old.thickness ||
		p.fadeoutOpacityAnimation != old.fadeoutOpacityAnimation ||
		p.mainAxisMargin != old.mainAxisMargin ||
		p.crossAxisMargin != old.crossAxisMargin ||
		p.radius != old.radius ||
		p.trackRadius != old.trackRadius ||
		p.shape != old.shape ||
		!p.padding.Equal(old.padding) ||
		p.minLength != old.minLength ||
		p.minOverscrollLength != old.minOverscrollLength ||
		p.scrollbarOrientation != old.scrollbarOrientation ||
		p.ignorePointer != old.ignorePointer
}

// Dispose stops listening to the opacity animation.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.dispose
func (p *ScrollbarPainter) Dispose() {
	p.fadeoutOpacityAnimation.RemoveListener(p.fadeoutHandle)
	p.ChangeNotifier.Dispose()
}

// String is the identity of the painter.
//
// Flutter: widgets/scrollbar.dart ScrollbarPainter.toString
func (p *ScrollbarPainter) String() string { return foundation.DescribeIdentity(p) }

// RawScrollbarOpts are the optional parameters of NewRawScrollbar.
//
// Flutter: widgets/scrollbar.dart RawScrollbar.new
type RawScrollbarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller is the controller of the scrollable the scrollbar belongs to;
	// the PrimaryScrollController's by default.
	Controller ScrollController
	// ThumbVisibility makes the thumb always visible when true, and never when
	// false; the default (none) is visible while scrolling.
	ThumbVisibility foundation.Opt[bool]
	// Shape is the shape of the thumb, instead of a radius.
	Shape painting.OutlinedBorder
	// Radius is the corner radius of the thumb.
	Radius foundation.Opt[ui.Radius]
	// Thickness is the thickness of the thumb; the default is 6.
	Thickness foundation.Opt[float64]
	// ThumbColor is the color of the thumb; the default is 0x66BCBCBC.
	ThumbColor foundation.Opt[ui.Color]
	// MinThumbLength is the shortest the thumb gets; the default is 18.
	MinThumbLength foundation.Opt[float64]
	// MinOverscrollLength is the shortest the thumb gets while it overscrolls;
	// the default is MinThumbLength.
	MinOverscrollLength foundation.Opt[float64]
	// TrackVisibility makes the track visible when true and the thumb is.
	TrackVisibility foundation.Opt[bool]
	// TrackRadius is the corner radius of the track.
	TrackRadius foundation.Opt[ui.Radius]
	// TrackColor is the color of the track; the default is 0x08000000.
	TrackColor foundation.Opt[ui.Color]
	// TrackBorderColor is the color of the border of the track; the default is
	// 0x1A000000.
	TrackBorderColor foundation.Opt[ui.Color]
	// FadeDuration is the duration of the fade of the scrollbar; the default is
	// 300 ms.
	FadeDuration foundation.Opt[time.Duration]
	// TimeToFade is the time after scrolling before the scrollbar fades; the
	// default is 600 ms.
	TimeToFade foundation.Opt[time.Duration]
	// PressDuration is how long the thumb must be pressed to start the drag; the
	// default is zero.
	PressDuration time.Duration
	// NotificationPredicate decides which scroll notifications the scrollbar
	// handles; the default is DefaultScrollNotificationPredicate.
	NotificationPredicate ScrollNotificationPredicate
	// Interactive is whether the scrollbar responds to the pointer; the default
	// is true.
	Interactive foundation.Opt[bool]
	// ScrollbarOrientation is the side of the scrollable the scrollbar sits on.
	ScrollbarOrientation ScrollbarOrientation
	// MainAxisMargin is the margin at the ends of the track.
	MainAxisMargin float64
	// CrossAxisMargin is the margin between the thumb and the edge.
	CrossAxisMargin float64
	// Padding is the padding around the scrollbar; the MediaQuery's padding by
	// default.
	Padding painting.EdgeInsetsGeometry
}

// RawScrollbar is a scrollbar without a visual style: it paints a thumb (and
// optionally a track) over the scrollable it wraps, fades it in and out, and lets
// the user drag the thumb and click the track. The Material Scrollbar builds on
// it. The ScrollBehavior builds one for each scrollable on the desktop platforms.
//
// Flutter: widgets/scrollbar.dart RawScrollbar
type RawScrollbar struct {
	StatefulWidget
	child Widget
	o     RawScrollbarOpts
}

// NewRawScrollbar creates the scrollbar around the child. In debug builds it
// panics with the tag RawScrollbarInvalid when its options are inconsistent: a
// track without a thumb, a negative length, a radius and a shape.
//
// Flutter: widgets/scrollbar.dart RawScrollbar.new
func NewRawScrollbar(child Widget, opts ...RawScrollbarOpts) *RawScrollbar {
	o := firstOpts(opts)
	if o.NotificationPredicate == nil {
		o.NotificationPredicate = DefaultScrollNotificationPredicate
	}
	if foundation.DebugMode {
		minLength := o.MinThumbLength.OrElse(minThumbExtent)
		invalid := func(why string) { panic(errors.New("RawScrollbarInvalid", why)) }
		switch {
		case o.ThumbVisibility.IsSome() && !o.ThumbVisibility.Value() && o.TrackVisibility.OrElse(false):
			invalid("a scrollbar track cannot be drawn without a scrollbar thumb")
		case minLength < 0:
			invalid("the minimum length of the thumb must not be negative")
		case o.MinOverscrollLength.IsSome() && o.MinOverscrollLength.Value() > minLength:
			invalid("the minimum overscroll length must not be more than the minimum thumb length")
		case o.MinOverscrollLength.IsSome() && o.MinOverscrollLength.Value() < 0:
			invalid("the minimum overscroll length must not be negative")
		case o.Radius.IsSome() && !isNil(o.Shape):
			invalid("a scrollbar can have a radius or a shape, not both")
		}
	}
	return &RawScrollbar{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child, o: o}
}

// Child is the scrollable the scrollbar is for.
//
// Flutter: widgets/scrollbar.dart RawScrollbar.child
func (w *RawScrollbar) Child() Widget { return w.child }

// Options are the options the widget was made with.
func (w *RawScrollbar) Options() RawScrollbarOpts { return w.o }

// CreateState creates the state.
//
// Flutter: widgets/scrollbar.dart RawScrollbar.createState
func (w *RawScrollbar) CreateState() State {
	return &RawScrollbarState{
		scrollbarPainterKey: NewGlobalKey[State](),
		gestureDetectorKey:  NewGlobalKey[*RawGestureDetectorState](),
	}
}

// RawScrollbarState is the state of a RawScrollbar.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState
type RawScrollbarState struct {
	StateOf[*RawScrollbar]
	TickerProvider

	startDragScrollbarAxisOffset foundation.Opt[ui.Offset]
	lastDragUpdateOffset         foundation.Opt[ui.Offset]
	startDragThumbOffset         foundation.Opt[float64]
	cachedController             ScrollController
	fadeoutTimer                 *gestures.Timer
	fadeoutAnimationController   *animation.AnimationController
	fadeoutOpacityAnimation      *animation.CurvedAnimation
	scrollbarPainterKey          *GlobalKey[State]
	hoverIsActive                bool
	thumbDrag                    gestures.Drag
	maxScrollExtentPermitsScroll bool
	thumbHold                    ScrollHoldController
	axis                         foundation.Opt[painting.Axis]
	gestureDetectorKey           *GlobalKey[*RawGestureDetectorState]

	scrollbarPainter *ScrollbarPainter
}

// ScrollbarPainter is the painter of the scrollbar.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.scrollbarPainter
func (s *RawScrollbarState) ScrollbarPainter() *ScrollbarPainter { return s.scrollbarPainter }

func (s *RawScrollbarState) effectiveScrollController() ScrollController {
	if c := s.Widget().o.Controller; !isNil(c) {
		return c
	}
	return PrimaryScrollControllerMaybeOf(s.Context())
}

// ShowScrollbar is whether the thumb is always visible.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.showScrollbar
func (s *RawScrollbarState) ShowScrollbar() bool { return s.Widget().o.ThumbVisibility.OrElse(false) }

func (s *RawScrollbarState) showTrack() bool {
	return s.ShowScrollbar() && s.Widget().o.TrackVisibility.OrElse(false)
}

// EnableGestures is whether the scrollbar responds to the pointer.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.enableGestures
func (s *RawScrollbarState) EnableGestures() bool { return s.Widget().o.Interactive.OrElse(true) }

func (s *RawScrollbarState) thumbColor() ui.Color {
	return s.Widget().o.ThumbColor.OrElse(ui.NewColor(0x66BCBCBC))
}

func (s *RawScrollbarState) thickness() float64 {
	return s.Widget().o.Thickness.OrElse(scrollbarThickness)
}

func (s *RawScrollbarState) minThumbLength() float64 {
	return s.Widget().o.MinThumbLength.OrElse(minThumbExtent)
}

// InitState creates the animation of the fade and the painter.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.initState
func (s *RawScrollbarState) InitState() {
	o := s.Widget().o
	s.fadeoutAnimationController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(o.FadeDuration.OrElse(scrollbarFade))})
	s.fadeoutAnimationController.AddStatusListener(s.validateInteractions)
	s.fadeoutOpacityAnimation = animation.NewCurvedAnimation(s.fadeoutAnimationController, animation.Curves.FastOutSlowIn)
	minOverscroll := o.MinOverscrollLength.OrElse(s.minThumbLength())
	s.scrollbarPainter = NewScrollbarPainter(s.thumbColor(), s.fadeoutOpacityAnimation, ScrollbarPainterOpts{
		Thickness: foundation.Some(s.thickness()), Radius: o.Radius, TrackRadius: o.TrackRadius,
		ScrollbarOrientation: o.ScrollbarOrientation, MainAxisMargin: o.MainAxisMargin, Shape: o.Shape,
		CrossAxisMargin: o.CrossAxisMargin, MinLength: foundation.Some(s.minThumbLength()), MinOverscrollLength: foundation.Some(minOverscroll),
	})
}

// DidChangeDependencies checks, after the frame, that the controller has a
// position when the thumb is always visible.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.didChangeDependencies
func (s *RawScrollbarState) DidChangeDependencies() {
	if foundation.DebugMode {
		s.debugScheduleCheckHasValidScrollPosition()
	}
}

func (s *RawScrollbarState) debugScheduleCheckHasValidScrollPosition() {
	if !s.ShowScrollbar() {
		return
	}
	WidgetsBindingOf(s.Context()).RunPostFrame(func(time.Duration) { s.debugCheckHasValidScrollPosition() })
}

func (s *RawScrollbarState) validateInteractions(status animation.AnimationStatus) {
	if status.IsDismissed() {
		if foundation.DebugMode && s.fadeoutOpacityAnimation.Value() != 0.0 {
			panic(errors.New("RawScrollbarFadeInvalid", "a dismissed scrollbar fade must have an opacity of zero"))
		}
	} else if !isNil(s.effectiveScrollController()) && s.EnableGestures() {
		// Interactive scrollbars should be properly mounted, otherwise
		// interaction (hover/drag) would fail; they are mounted if the fade
		// goes forward or completes.
		if s.fadeoutAnimationController.Status() == animation.AnimationStatusForward && s.Widget().o.ThumbVisibility.OrElse(false) {
			// When thumbVisibility is true, we do not check hasClients because
			// it is checked after the first frame.
			return
		}
		if foundation.DebugMode {
			s.debugCheckHasValidScrollPosition()
		}
	}
}

// debugCheckHasValidScrollPosition panics when the scrollbar needs a scroll
// position and there is none, or there are several.
func (s *RawScrollbarState) debugCheckHasValidScrollPosition() {
	if !s.Mounted() {
		return
	}
	controller := s.effectiveScrollController()
	tryPrimary := isNil(s.Widget().o.Controller)
	controllerForError := "provided ScrollController"
	if tryPrimary {
		controllerForError = "PrimaryScrollController"
	}
	when := "using the Scrollbar"
	if s.Widget().o.ThumbVisibility.OrElse(false) {
		when = "Scrollbar.thumbVisibility is true"
	} else if s.EnableGestures() {
		when = "the scrollbar is interactive"
	}
	if isNil(controller) {
		panic(errors.New("ScrollbarControllerMissing", "a ScrollController is required when "+when+
			"; the Scrollbar was not provided a ScrollController, and attempted to use the PrimaryScrollController, but none was found",
			"controller", controllerForError))
	}
	if !controller.HasClients() {
		panic(errors.New("ScrollbarNoScrollPosition",
			"the Scrollbar's ScrollController has no ScrollPosition attached; a Scrollbar cannot be painted without a ScrollPosition; the Scrollbar attempted to use the "+
				controllerForError+"; this ScrollController should be associated with the ScrollView that the Scrollbar is being applied to",
			"controller", controllerForError))
	}
	if len(controller.Positions()) > 1 {
		panic(errors.New("ScrollbarMultipleScrollPositions",
			"the "+controllerForError+" is attached to more than one ScrollPosition; the Scrollbar requires a single ScrollPosition in order to be painted; when "+when+
				", the associated ScrollController must only have one ScrollPosition attached",
			"controller", controllerForError, "positions", len(controller.Positions())))
	}
}

// UpdateScrollbarPainter copies the options of the widget and the text direction
// and the padding of the context to the painter.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.updateScrollbarPainter
func (s *RawScrollbarState) UpdateScrollbarPainter() {
	o := s.Widget().o
	textDirection := DirectionalityOf(s.Context())
	trackColor, trackBorderColor := ui.NewColor(0x00000000), ui.NewColor(0x00000000)
	if s.showTrack() {
		trackColor = o.TrackColor.OrElse(ui.NewColor(0x08000000))
		trackBorderColor = o.TrackBorderColor.OrElse(ui.NewColor(0x1a000000))
	}
	padding := o.Padding
	if padding == nil {
		padding = MediaQueryPaddingOf(s.Context())
	}
	p := s.scrollbarPainter
	p.SetColor(s.thumbColor())
	p.SetTrackRadius(o.TrackRadius)
	p.SetTrackColor(trackColor)
	p.SetTrackBorderColor(trackBorderColor)
	p.SetTextDirection(textDirection)
	p.SetThickness(s.thickness())
	p.radius = o.Radius // the shape setter below checks the pair
	p.SetPadding(padding.Resolve(textDirection))
	p.SetScrollbarOrientation(o.ScrollbarOrientation)
	p.SetMainAxisMargin(o.MainAxisMargin)
	p.shape = o.Shape
	p.SetCrossAxisMargin(o.CrossAxisMargin)
	p.SetMinLength(s.minThumbLength())
	p.SetMinOverscrollLength(o.MinOverscrollLength.OrElse(s.minThumbLength()))
	p.SetIgnorePointer(!s.EnableGestures())
}

// DidUpdateWidget starts or reverses the fade when the visibility of the thumb
// changed.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.didUpdateWidget
func (s *RawScrollbarState) DidUpdateWidget(oldWidget *RawScrollbar) {
	if s.Widget().o.ThumbVisibility != oldWidget.o.ThumbVisibility {
		if s.Widget().o.ThumbVisibility.OrElse(false) {
			if foundation.DebugMode {
				s.debugScheduleCheckHasValidScrollPosition()
			}
			if s.fadeoutTimer != nil {
				s.fadeoutTimer.Cancel()
			}
			s.fadeoutAnimationController.AnimateTo(1.0)
		} else {
			s.fadeoutAnimationController.Reverse()
		}
	}
}

func (s *RawScrollbarState) maybeStartFadeoutTimer() {
	if !s.ShowScrollbar() {
		if s.fadeoutTimer != nil {
			s.fadeoutTimer.Cancel()
		}
		s.fadeoutTimer = WidgetsBindingOf(s.Context()).Gestures().NewTimer(s.Widget().o.TimeToFade.OrElse(scrollbarTimeToFade), func() {
			s.fadeoutAnimationController.Reverse()
			s.fadeoutTimer = nil
		})
	}
}

func (s *RawScrollbarState) cancelFadeoutTimer() {
	if s.fadeoutTimer != nil {
		s.fadeoutTimer.Cancel()
	}
}

// GetScrollbarDirection is the axis of the scrollable the scrollbar listens to;
// none until the first scroll metrics notification.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.getScrollbarDirection
func (s *RawScrollbarState) GetScrollbarDirection() foundation.Opt[painting.Axis] { return s.axis }

func (s *RawScrollbarState) disposeThumbDrag() { s.thumbDrag = nil }

func (s *RawScrollbarState) disposeThumbHold() { s.thumbHold = nil }

// getPrimaryDelta is the change of the scroll offset that the drag of the thumb
// to localPosition makes; none if the position does not change.
func (s *RawScrollbarState) getPrimaryDelta(localPosition ui.Offset) foundation.Opt[float64] {
	position := s.cachedController.Position()
	start, last := s.startDragScrollbarAxisOffset.Value(), s.lastDragUpdateOffset.Value()
	var primaryDeltaFromDragStart, primaryDeltaFromLastDragUpdate float64
	switch position.AxisDirection() {
	case painting.AxisDirectionUp:
		primaryDeltaFromDragStart = start.Dy - localPosition.Dy
		primaryDeltaFromLastDragUpdate = last.Dy - localPosition.Dy
	case painting.AxisDirectionRight:
		primaryDeltaFromDragStart = localPosition.Dx - start.Dx
		primaryDeltaFromLastDragUpdate = localPosition.Dx - last.Dx
	case painting.AxisDirectionDown:
		primaryDeltaFromDragStart = localPosition.Dy - start.Dy
		primaryDeltaFromLastDragUpdate = localPosition.Dy - last.Dy
	case painting.AxisDirectionLeft:
		primaryDeltaFromDragStart = start.Dx - localPosition.Dx
		primaryDeltaFromLastDragUpdate = last.Dx - localPosition.Dx
	}
	// Convert primaryDelta, the amount that the scrollbar moved since the last
	// time when drag started or last updated, into the coordinate space of the
	// scroll position, and to account for a possible difference in the ratio of
	// scrollbar height and scrollable height, which can be different from 1.0.
	scrollOffsetGlobal := s.scrollbarPainter.GetTrackToScroll(s.startDragThumbOffset.Value() + primaryDeltaFromDragStart)
	if (primaryDeltaFromDragStart > 0 && scrollOffsetGlobal < position.Pixels()) ||
		(primaryDeltaFromDragStart < 0 && scrollOffsetGlobal > position.Pixels()) {
		// Adjust the position value if the scrolling direction conflicts with
		// the dragging direction due to scroll metrics shrinking.
		scrollOffsetGlobal = position.Pixels() + s.scrollbarPainter.GetTrackToScroll(primaryDeltaFromLastDragUpdate)
	}
	if scrollOffsetGlobal != position.Pixels() {
		// Ensure we don't drag into overscroll if the physics do not allow it.
		physicsAdjustment := position.Physics().ApplyBoundaryConditions(position, scrollOffsetGlobal)
		newPosition := scrollOffsetGlobal - physicsAdjustment
		// The physics may allow overscroll when actually *scrolling*, but the
		// scrollbar should not allow the position to go out of bounds when
		// dragging, so we are just clamping the value here.
		switch ScrollConfigurationOf(s.Context()).GetPlatform(s.Context()) {
		case foundation.TargetPlatformFuchsia, foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows:
			newPosition = ui.ClampDouble(newPosition, position.MinScrollExtent(), position.MaxScrollExtent())
		}
		if painting.AxisDirectionIsReversed(position.AxisDirection()) {
			return foundation.Some(newPosition - position.Pixels())
		}
		return foundation.Some(position.Pixels() - newPosition)
	}
	return foundation.Opt[float64]{}
}

// HandleThumbPress is called when a pointer touches the thumb: it holds the
// scrollable.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.handleThumbPress
func (s *RawScrollbarState) HandleThumbPress() {
	if foundation.DebugMode {
		s.debugCheckHasValidScrollPosition()
	}
	s.cachedController = s.effectiveScrollController()
	if !s.GetScrollbarDirection().IsSome() {
		return
	}
	s.cancelFadeoutTimer()
	s.thumbHold = s.cachedController.Position().Hold(s.disposeThumbHold)
}

func (s *RawScrollbarState) scrollbarBox() rendering.Box {
	return s.scrollbarPainterKey.CurrentContext().FindRenderObject().(rendering.Box)
}

// HandleThumbPressStart is called when the thumb starts to be dragged: it starts
// a drag of the position.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.handleThumbPressStart
func (s *RawScrollbarState) HandleThumbPressStart(localPosition ui.Offset) {
	if foundation.DebugMode {
		s.debugCheckHasValidScrollPosition()
	}
	if !s.GetScrollbarDirection().IsSome() {
		return
	}
	s.cancelFadeoutTimer()
	s.fadeoutAnimationController.Forward()
	if foundation.DebugMode && s.thumbDrag != nil {
		panic(errors.New("RawScrollbarDragInvalid", "a thumb drag started while the scrollbar has one"))
	}
	position := s.cachedController.Position()
	details := gestures.NewDragStartDetails(gestures.DragStartOpts{
		LocalPosition: foundation.Some(localPosition), GlobalPosition: s.scrollbarBox().LocalToGlobal(localPosition, nil)})
	s.thumbDrag = position.Drag(details, s.disposeThumbDrag)
	s.startDragScrollbarAxisOffset = foundation.Some(localPosition)
	s.lastDragUpdateOffset = foundation.Some(localPosition)
	s.startDragThumbOffset = foundation.Some(s.scrollbarPainter.GetThumbScrollOffset())
}

// HandleThumbPressUpdate is called when the thumb is dragged: it updates the
// drag of the position.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.handleThumbPressUpdate
func (s *RawScrollbarState) HandleThumbPressUpdate(localPosition ui.Offset) {
	if foundation.DebugMode {
		s.debugCheckHasValidScrollPosition()
	}
	if s.lastDragUpdateOffset.IsSome() && s.lastDragUpdateOffset.Value() == localPosition {
		// In some cases, the drag update is called with the same position as
		// the last update.
		return
	}
	position := s.cachedController.Position()
	if !position.Physics().ShouldAcceptUserOffset(position) {
		return
	}
	direction := s.GetScrollbarDirection()
	if !direction.IsSome() {
		return
	}
	// _thumbDrag might be null if the drag activity ended and called
	// _disposeThumbDrag.
	if s.thumbDrag == nil {
		return
	}
	primaryDelta := s.getPrimaryDelta(localPosition)
	if !primaryDelta.IsSome() {
		return
	}
	delta := ui.Offset{Dy: primaryDelta.Value()}
	if direction.Value() == painting.AxisHorizontal {
		delta = ui.Offset{Dx: primaryDelta.Value()}
	}
	scrollDetails := gestures.NewDragUpdateDetails(s.scrollbarBox().LocalToGlobal(localPosition, nil), gestures.DragUpdateOpts{
		Delta: delta, PrimaryDelta: primaryDelta, LocalPosition: foundation.Some(localPosition)})
	// Triggers updates to the ScrollPosition and ScrollbarPainter.
	s.thumbDrag.Update(scrollDetails)
	s.lastDragUpdateOffset = foundation.Some(localPosition)
}

// HandleThumbPressEnd is called when the thumb stops being dragged: it ends the
// drag of the position, with the velocity of the drag on mobile platforms.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.handleThumbPressEnd
func (s *RawScrollbarState) HandleThumbPressEnd(localPosition ui.Offset, velocity gestures.Velocity) {
	if foundation.DebugMode {
		s.debugCheckHasValidScrollPosition()
	}
	direction := s.GetScrollbarDirection()
	if !direction.IsSome() {
		return
	}
	s.maybeStartFadeoutTimer()
	s.cachedController = nil
	s.lastDragUpdateOffset = foundation.Opt[ui.Offset]{}
	if s.thumbDrag == nil {
		return
	}
	adjustedVelocity := gestures.Velocity{}
	switch ScrollConfigurationOf(s.Context()).GetPlatform(s.Context()) {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformAndroid:
		adjustedVelocity = velocity.Neg()
	}
	primaryVelocity := adjustedVelocity.PixelsPerSecond.Dy
	if direction.Value() == painting.AxisHorizontal {
		primaryVelocity = adjustedVelocity.PixelsPerSecond.Dx
	}
	details := gestures.NewDragEndDetails(gestures.DragEndOpts{
		LocalPosition: foundation.Some(localPosition), GlobalPosition: s.scrollbarBox().LocalToGlobal(localPosition, nil),
		Velocity: adjustedVelocity, PrimaryVelocity: foundation.Some(primaryVelocity)})
	s.thumbDrag.End(details)
	s.startDragScrollbarAxisOffset = foundation.Opt[ui.Offset]{}
	s.lastDragUpdateOffset = foundation.Opt[ui.Offset]{}
	s.startDragThumbOffset = foundation.Opt[float64]{}
	s.cachedController = nil
}

// HandleTrackTapDown is called when the track is tapped: it scrolls a page toward
// the tap.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.handleTrackTapDown
func (s *RawScrollbarState) HandleTrackTapDown(details gestures.TapDownDetails) {
	if foundation.DebugMode {
		s.debugCheckHasValidScrollPosition()
	}
	s.cachedController = s.effectiveScrollController()
	position := s.cachedController.Position()
	if !position.Physics().ShouldAcceptUserOffset(position) {
		return
	}
	// Determines the scroll direction.
	var scrollDirection painting.AxisDirection
	switch painting.AxisDirectionToAxis(position.AxisDirection()) {
	case painting.AxisVertical:
		if details.LocalPosition.Dy > s.scrollbarPainter.thumbOffset {
			scrollDirection = painting.AxisDirectionDown
		} else {
			scrollDirection = painting.AxisDirectionUp
		}
	default:
		if details.LocalPosition.Dx > s.scrollbarPainter.thumbOffset {
			scrollDirection = painting.AxisDirectionRight
		} else {
			scrollDirection = painting.AxisDirectionLeft
		}
	}
	state := ScrollableMaybeOf(position.Context().NotificationContext())
	if foundation.DebugMode && state == nil {
		panic(errors.New("ScrollableNotFound", "the scroll position of the scrollbar has no Scrollable above its notification context"))
	}
	intent := ScrollIntent{Direction: scrollDirection, Type: ScrollIncrementTypePage}
	scrollIncrement := ScrollActionGetDirectionalIncrement(state, intent)
	position.MoveTo(position.Pixels()+scrollIncrement, rendering.ViewportOffsetMoveToOpts{
		Duration: foundation.Some(100 * time.Millisecond), Curve: animation.Curves.EaseInOut})
}

func (s *RawScrollbarState) shouldUpdatePainter(notificationAxis painting.Axis) bool {
	scrollController := s.effectiveScrollController()
	// Only update the painter of this scrollbar if the notification
	// metrics do not conflict with the information we have from the scroll
	// controller.
	// We do not have a scroll controller dependency yet, so we cannot compare.
	if isNil(scrollController) {
		return true
	}
	// The scroll controller is attached to multiple positions.
	if len(scrollController.Positions()) > 1 {
		return false
	}
	// The scroll controller is not attached to a position, or its position axis
	// is the same as the notification's.
	return !scrollController.HasClients() || scrollController.Position().Axis() == notificationAxis
}

func (s *RawScrollbarState) handleScrollMetricsNotification(notification *ScrollMetricsNotification) bool {
	if !s.Widget().o.NotificationPredicate(notification.AsScrollUpdate()) {
		return false
	}
	if s.ShowScrollbar() && !s.fadeoutAnimationController.Status().IsForwardOrCompleted() {
		s.fadeoutAnimationController.Forward()
	}
	metrics := notification.Metrics()
	if s.shouldUpdatePainter(metrics.Axis()) {
		s.scrollbarPainter.Update(metrics, metrics.AxisDirection())
	}
	if !s.axis.IsSome() || s.axis.Value() != metrics.Axis() {
		s.SetState(func() { s.axis = foundation.Some(metrics.Axis()) })
	}
	if s.maxScrollExtentPermitsScroll != (notification.Metrics().MaxScrollExtent() > 0.0) {
		s.SetState(func() { s.maxScrollExtentPermitsScroll = !s.maxScrollExtentPermitsScroll })
	}
	return false
}

func (s *RawScrollbarState) handleScrollNotification(notification ScrollNotification) bool {
	if !s.Widget().o.NotificationPredicate(notification) {
		return false
	}
	metrics := notification.Metrics()
	if metrics.MaxScrollExtent() <= metrics.MinScrollExtent() {
		// Hide the bar when the Scrollable widget has no space to scroll.
		if s.fadeoutAnimationController.Status().IsForwardOrCompleted() {
			s.fadeoutAnimationController.Reverse()
		}
		if s.shouldUpdatePainter(metrics.Axis()) {
			s.scrollbarPainter.Update(metrics, metrics.AxisDirection())
		}
		return false
	}
	switch notification.(type) {
	case *ScrollUpdateNotification, *OverscrollNotification:
		// Any movement beyond the scrollable's extents or in its bounds shows
		// the scrollbar.
		if !s.fadeoutAnimationController.Status().IsForwardOrCompleted() {
			s.fadeoutAnimationController.Forward()
		}
		s.cancelFadeoutTimer()
		if s.shouldUpdatePainter(metrics.Axis()) {
			s.scrollbarPainter.Update(metrics, metrics.AxisDirection())
		}
	case *ScrollEndNotification:
		if s.thumbDrag == nil {
			s.maybeStartFadeoutTimer()
		}
	}
	return false
}

func (s *RawScrollbarState) handleThumbDragDown(gestures.DragDownDetails) { s.HandleThumbPress() }

func (s *RawScrollbarState) globalToScrollbar(offset ui.Offset) ui.Offset {
	return s.scrollbarBox().GlobalToLocal(offset, nil)
}

func (s *RawScrollbarState) handleThumbDragStart(details gestures.DragStartDetails) {
	s.HandleThumbPressStart(s.globalToScrollbar(details.GlobalPosition))
}

func (s *RawScrollbarState) handleThumbDragUpdate(details gestures.DragUpdateDetails) {
	s.HandleThumbPressUpdate(s.globalToScrollbar(details.GlobalPosition))
}

func (s *RawScrollbarState) handleThumbDragEnd(details gestures.DragEndDetails) {
	s.HandleThumbPressEnd(s.globalToScrollbar(details.GlobalPosition), details.Velocity)
}

func (s *RawScrollbarState) handleThumbDragCancel() {
	if s.gestureDetectorKey.CurrentContext() == nil {
		// The cancel was caused by the GestureDetector getting disposed, which
		// means we will get disposed momentarily as well and shouldn't do
		// any work.
		return
	}
	// _thumbHold might be null if the drag started.
	// _thumbDrag might be null if the drag activity ended and called
	// _disposeThumbDrag.
	if s.thumbHold != nil {
		s.thumbHold.Cancel()
	}
	if s.thumbDrag != nil {
		s.thumbDrag.Cancel()
	}
}

func (s *RawScrollbarState) canHandleScrollGestures() bool {
	c := s.effectiveScrollController()
	return s.EnableGestures() && !isNil(c) && len(c.Positions()) == 1 && c.Position().HasContentDimensions() &&
		c.Position().MaxScrollExtent()-c.Position().MinScrollExtent() > foundation.PrecisionErrorTolerance
}

// thumbAndTrackGestures are the recognizers of the thumb drag and the track tap, for the axis of
// the scrollable; none when the scrollbar cannot handle gestures.
func (s *RawScrollbarState) thumbAndTrackGestures() map[reflect.Type]GestureRecognizerFactory {
	factories := map[reflect.Type]GestureRecognizerFactory{}
	if !s.canHandleScrollGestures() {
		return factories
	}
	binding := WidgetsBindingOf(s.Context()).Gestures()
	initDrag := func(r *gestures.DragGestureRecognizer) {
		r.OnDown = s.handleThumbDragDown
		r.OnStart = s.handleThumbDragStart
		r.OnUpdate = s.handleThumbDragUpdate
		r.OnEnd = s.handleThumbDragEnd
		r.OnCancel = s.handleThumbDragCancel
		r.GestureSettings = foundation.Some(gestures.DeviceGestureSettings{TouchSlop: foundation.Some(0.0)})
		r.DragStartBehavior = gestures.DragStartBehaviorDown
	}
	var f GestureRecognizerFactory
	if s.effectiveScrollController().Position().Axis() == painting.AxisHorizontal {
		f = NewGestureRecognizerFactoryWithHandlers(
			func() *horizontalThumbDragRecognizer { return newHorizontalThumbDragRecognizer(binding, s) },
			func(r *horizontalThumbDragRecognizer) { initDrag(&r.DragGestureRecognizer) })
	} else {
		f = NewGestureRecognizerFactoryWithHandlers(
			func() *verticalThumbDragRecognizer { return newVerticalThumbDragRecognizer(binding, s) },
			func(r *verticalThumbDragRecognizer) { initDrag(&r.DragGestureRecognizer) })
	}
	factories[f.RecognizerType()] = f
	tap := NewGestureRecognizerFactoryWithHandlers(
		func() *trackTapRecognizer { return newTrackTapRecognizer(binding, s) },
		func(r *trackTapRecognizer) { r.OnTapDown = s.HandleTrackTapDown })
	factories[tap.RecognizerType()] = tap
	return factories
}

func (s *RawScrollbarState) painterAt(position ui.Offset) (ui.Offset, bool) {
	c := s.scrollbarPainterKey.CurrentContext()
	if c == nil {
		return ui.Offset{}, false
	}
	return c.FindRenderObject().(rendering.Box).GlobalToLocal(position, nil), true
}

// IsPointerOverTrack is whether the global position is over the track but not the
// thumb.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.isPointerOverTrack
func (s *RawScrollbarState) IsPointerOverTrack(position ui.Offset, kind ui.PointerDeviceKind) bool {
	local, ok := s.painterAt(position)
	if !ok {
		return false
	}
	return s.scrollbarPainter.HitTestInteractive(local, kind) && !s.scrollbarPainter.HitTestOnlyThumbInteractive(local, kind)
}

// IsPointerOverThumb is whether the global position is over the thumb.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.isPointerOverThumb
func (s *RawScrollbarState) IsPointerOverThumb(position ui.Offset, kind ui.PointerDeviceKind) bool {
	local, ok := s.painterAt(position)
	if !ok {
		return false
	}
	return s.scrollbarPainter.HitTestOnlyThumbInteractive(local, kind)
}

// IsPointerOverScrollbar is whether the global position is over the scrollbar;
// forHover accepts a transparent scrollbar for a mouse.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.isPointerOverScrollbar
func (s *RawScrollbarState) IsPointerOverScrollbar(position ui.Offset, kind ui.PointerDeviceKind) bool {
	local, ok := s.painterAt(position)
	if !ok {
		return false
	}
	return s.scrollbarPainter.HitTestInteractive(local, kind, true)
}

// HandleHover is called when a mouse moves over the scrollbar's box: it shows the
// scrollbar while the mouse is over it.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.handleHover
func (s *RawScrollbarState) HandleHover(event *gestures.PointerHoverEvent) {
	// Check if the position of the pointer falls over the painted scrollbar.
	if s.IsPointerOverScrollbar(event.Position(), event.Kind()) {
		// Pointer is hovering over the scrollbar.
		s.hoverIsActive = true
		s.fadeoutAnimationController.Forward()
		s.cancelFadeoutTimer()
	} else if s.hoverIsActive {
		// Pointer was, but is no longer over painted scrollbar.
		s.hoverIsActive = false
		s.maybeStartFadeoutTimer()
	}
}

// HandleHoverExit is called when the mouse leaves the scrollbar's box.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.handleHoverExit
func (s *RawScrollbarState) HandleHoverExit(*gestures.PointerExitEvent) {
	s.hoverIsActive = false
	s.maybeStartFadeoutTimer()
}

func (s *RawScrollbarState) pointerSignalEventDelta(event *gestures.PointerScrollEvent) float64 {
	position := s.cachedController.Position()
	delta := event.ScrollDelta().Dy
	if position.Axis() == painting.AxisHorizontal {
		delta = event.ScrollDelta().Dx
	}
	if painting.AxisDirectionIsReversed(position.AxisDirection()) {
		delta *= -1
	}
	return delta
}

func (s *RawScrollbarState) targetScrollOffsetForPointerScroll(delta float64) float64 {
	position := s.cachedController.Position()
	return math.Min(math.Max(position.Pixels()+delta, position.MinScrollExtent()), position.MaxScrollExtent())
}

func (s *RawScrollbarState) handlePointerScroll(event gestures.PointerSignalEvent) {
	s.cachedController = s.effectiveScrollController()
	delta := s.pointerSignalEventDelta(event.(*gestures.PointerScrollEvent))
	targetScrollOffset := s.targetScrollOffsetForPointerScroll(delta)
	if delta != 0 && targetScrollOffset != s.cachedController.Position().Pixels() {
		s.cachedController.Position().PointerScroll(delta)
	}
}

func (s *RawScrollbarState) receivedPointerSignal(event gestures.PointerSignalEvent) {
	s.cachedController = s.effectiveScrollController()
	// Only try to scroll if the bar absorb the hit test.
	if hit := s.scrollbarPainter.HitTest(event.LocalPosition()); hit.OrElse(false) &&
		!isNil(s.cachedController) && s.cachedController.HasClients() && s.thumbDrag == nil {
		position := s.cachedController.Position()
		switch e := event.(type) {
		case *gestures.PointerScrollEvent:
			if !position.Physics().ShouldAcceptUserOffset(position) {
				return
			}
			delta := s.pointerSignalEventDelta(e)
			targetScrollOffset := s.targetScrollOffsetForPointerScroll(delta)
			if delta != 0 && targetScrollOffset != position.Pixels() {
				WidgetsBindingOf(s.Context()).Gestures().PointerSignalResolver.Register(e, s.handlePointerScroll)
			}
		case *gestures.PointerScrollInertiaCancelEvent:
			position.JumpTo(position.Pixels())
			// Don't use the pointer signal resolver, all hit-tested scrollables
			// should stop.
		}
	}
}

// Dispose disposes the fade, the timer and the painter.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.dispose
func (s *RawScrollbarState) Dispose() {
	s.fadeoutAnimationController.Dispose()
	s.cancelFadeoutTimer()
	s.scrollbarPainter.Dispose()
	s.fadeoutOpacityAnimation.Dispose()
}

// Build updates the painter and builds the notification listeners, the gesture
// detector, the mouse region and the painted child.
//
// Flutter: widgets/scrollbar.dart RawScrollbarState.build
func (s *RawScrollbarState) Build(bc BuildContext) Widget {
	s.UpdateScrollbarPainter()
	child := NewRepaintBoundary(
		NewListener(
			NewRawGestureDetector(
				NewMouseRegion(
					NewCustomPaint(NewRepaintBoundary(s.Widget().child), CustomPaintOpts{Key: s.scrollbarPainterKey, ForegroundPainter: s.scrollbarPainter}),
					MouseRegionOpts{
						OnExit: func(event *gestures.PointerExitEvent) {
							switch event.Kind() {
							case ui.PointerDeviceKindMouse, ui.PointerDeviceKindTrackpad:
								if s.EnableGestures() {
									s.HandleHoverExit(event)
								}
							}
						},
						OnHover: func(event *gestures.PointerHoverEvent) {
							switch event.Kind() {
							case ui.PointerDeviceKindMouse, ui.PointerDeviceKindTrackpad:
								if s.EnableGestures() {
									s.HandleHover(event)
								}
							}
						},
					}),
				RawGestureDetectorOpts{Key: s.gestureDetectorKey, Gestures: s.thumbAndTrackGestures()}),
			ListenerOpts{OnPointerSignal: s.receivedPointerSignal}))
	return NewNotificationListener[*ScrollMetricsNotification](
		NewNotificationListener[ScrollNotification](child, s.handleScrollNotification),
		s.handleScrollMetricsNotification)
}

func scrollbarControllerMissingError() error {
	return errors.New("ScrollbarControllerMissing", "a ScrollController is required to build a scrollbar for a Scrollable; the details of the scrollable must have a controller")
}
