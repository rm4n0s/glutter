// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"fmt"
	"math"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// RangeValues are the two values of a RangeSlider: the start and the end of a
// range.
//
// Flutter: material/range_slider_parts.dart RangeValues
type RangeValues struct {
	// Start is the lower bound of the range.
	//
	// Flutter: material/range_slider_parts.dart RangeValues.start
	Start float64
	// End is the upper bound of the range.
	//
	// Flutter: material/range_slider_parts.dart RangeValues.end
	End float64
}

// NewRangeValues creates the values.
//
// Flutter: material/range_slider_parts.dart RangeValues.new
func NewRangeValues(start, end float64) RangeValues { return RangeValues{Start: start, End: end} }

// String describes the values.
//
// Flutter: material/range_slider_parts.dart RangeValues.toString
func (v RangeValues) String() string { return fmt.Sprintf("RangeValues(%v, %v)", v.Start, v.End) }

// RangeLabels are the labels of the two thumbs of a RangeSlider.
//
// Flutter: material/range_slider_parts.dart RangeLabels
type RangeLabels struct {
	// Start is the label of the start thumb.
	//
	// Flutter: material/range_slider_parts.dart RangeLabels.start
	Start string
	// End is the label of the end thumb.
	//
	// Flutter: material/range_slider_parts.dart RangeLabels.end
	End string
}

// NewRangeLabels creates the labels.
//
// Flutter: material/range_slider_parts.dart RangeLabels.new
func NewRangeLabels(start, end string) RangeLabels { return RangeLabels{Start: start, End: end} }

// String describes the labels.
//
// Flutter: material/range_slider_parts.dart RangeLabels.toString
func (l RangeLabels) String() string { return fmt.Sprintf("RangeLabels(%s, %s)", l.Start, l.End) }

// RangeThumbSelector decides which thumb of a RangeSlider a press moves: none
// when the press is on neither.
//
// Flutter: material/range_slider_parts.dart RangeThumbSelector
type RangeThumbSelector func(textDirection ui.TextDirection, values RangeValues, tapValue float64, thumbSize, trackSize ui.Size, dx float64) foundation.Opt[Thumb]

// RangeSliderThumbPaintOpts are the named parameters of RangeSliderThumbShape.Paint.
//
// Flutter: material/range_slider_parts.dart RangeSliderThumbShape.paint
type RangeSliderThumbPaintOpts struct {
	// ActivationAnimation is the animation of the press of the thumb.
	ActivationAnimation animation.Animation[float64]
	// EnableAnimation is the animation of the transition between enabled and
	// disabled.
	EnableAnimation animation.Animation[float64]
	// IsDiscrete is whether the slider has divisions.
	IsDiscrete bool
	// IsEnabled is whether the slider is enabled.
	IsEnabled bool
	// IsOnTop is whether the thumb is above the other one.
	IsOnTop bool
	// TextDirection is the directionality of the slider.
	TextDirection ui.TextDirection
	// SliderTheme is the theme of the slider, with the defaults filled in.
	SliderTheme SliderThemeData
	// Thumb is which thumb this is.
	Thumb Thumb
	// IsPressed is whether the thumb is pressed.
	IsPressed bool
}

// RangeSliderThumbShape is the base of the shapes of the thumbs of a RangeSlider.
//
// Flutter: material/range_slider_parts.dart RangeSliderThumbShape
type RangeSliderThumbShape interface {
	// GetPreferredSize is the preferred size of a thumb.
	//
	// Flutter: material/range_slider_parts.dart RangeSliderThumbShape.getPreferredSize
	GetPreferredSize(isEnabled, isDiscrete bool) ui.Size
	// Paint paints a thumb around the center.
	//
	// Flutter: material/range_slider_parts.dart RangeSliderThumbShape.paint
	Paint(context *rendering.PaintingContext, center ui.Offset, opts RangeSliderThumbPaintOpts)
}

// RangeSliderValueIndicatorPaintOpts are the named parameters of
// RangeSliderValueIndicatorShape.Paint.
//
// Flutter: material/range_slider_parts.dart RangeSliderValueIndicatorShape.paint
type RangeSliderValueIndicatorPaintOpts struct {
	// ActivationAnimation is the animation of the opening of the indicator.
	ActivationAnimation animation.Animation[float64]
	// EnableAnimation is the animation of the transition between enabled and
	// disabled.
	EnableAnimation animation.Animation[float64]
	// IsDiscrete is whether the slider has divisions.
	IsDiscrete bool
	// IsOnTop is whether the indicator is above the other one.
	IsOnTop bool
	// LabelPainter paints the label.
	LabelPainter *painting.TextPainter
	// TextScaleFactor is the scale of the label.
	TextScaleFactor float64
	// SizeWithOverflow is the size of the area that the indicator may paint in.
	SizeWithOverflow ui.Size
	// ParentBox is the box of the slider.
	ParentBox rendering.Box
	// SliderTheme is the theme of the slider, with the defaults filled in.
	SliderTheme SliderThemeData
	// TextDirection is the directionality of the slider.
	TextDirection ui.TextDirection
	// Value is the value of the thumb, from 0 to 1.
	Value float64
	// Thumb is which thumb the indicator is for.
	Thumb Thumb
}

// RangeSliderHorizontalShiftOpts are the named parameters of
// RangeSliderValueIndicatorShape.GetHorizontalShift.
//
// Flutter: material/range_slider_parts.dart RangeSliderValueIndicatorShape.getHorizontalShift
type RangeSliderHorizontalShiftOpts struct {
	// ParentBox is the box of the slider.
	ParentBox rendering.Box
	// Center is the center of the thumb.
	Center ui.Offset
	// LabelPainter paints the label.
	LabelPainter *painting.TextPainter
	// ActivationAnimation is the animation of the opening of the indicator.
	ActivationAnimation animation.Animation[float64]
	// TextScaleFactor is the scale of the label.
	TextScaleFactor float64
	// SizeWithOverflow is the size of the area that the indicator may paint in.
	SizeWithOverflow ui.Size
}

// RangeSliderValueIndicatorShape is the base of the shapes of the value indicators
// of a RangeSlider.
//
// Flutter: material/range_slider_parts.dart RangeSliderValueIndicatorShape
type RangeSliderValueIndicatorShape interface {
	// GetPreferredSize is the preferred size of an indicator for the label.
	//
	// Flutter: material/range_slider_parts.dart RangeSliderValueIndicatorShape.getPreferredSize
	GetPreferredSize(isEnabled, isDiscrete bool, labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size
	// GetHorizontalShift is how far the indicator shifts to stay in the view; the
	// default is zero.
	//
	// Flutter: material/range_slider_parts.dart RangeSliderValueIndicatorShape.getHorizontalShift
	GetHorizontalShift(opts RangeSliderHorizontalShiftOpts) float64
	// Paint paints an indicator above the center of the thumb.
	//
	// Flutter: material/range_slider_parts.dart RangeSliderValueIndicatorShape.paint
	Paint(context *rendering.PaintingContext, center ui.Offset, opts RangeSliderValueIndicatorPaintOpts)
}

// RangeSliderValueIndicatorShapeBase has the default GetHorizontalShift of a
// RangeSliderValueIndicatorShape: embed it.
//
// Flutter: material/range_slider_parts.dart RangeSliderValueIndicatorShape.getHorizontalShift
type RangeSliderValueIndicatorShapeBase struct{}

// GetHorizontalShift is zero.
//
// Flutter: material/range_slider_parts.dart RangeSliderValueIndicatorShape.getHorizontalShift
func (RangeSliderValueIndicatorShapeBase) GetHorizontalShift(RangeSliderHorizontalShiftOpts) float64 {
	return 0
}

// RangeSliderTickMarkPaintOpts are the named parameters of
// RangeSliderTickMarkShape.Paint.
//
// Flutter: material/range_slider_parts.dart RangeSliderTickMarkShape.paint
type RangeSliderTickMarkPaintOpts struct {
	// ParentBox is the box of the slider.
	ParentBox rendering.Box
	// SliderTheme is the theme of the slider, with the defaults filled in.
	SliderTheme SliderThemeData
	// EnableAnimation is the animation of the transition between enabled and
	// disabled.
	EnableAnimation animation.Animation[float64]
	// StartThumbCenter is the center of the start thumb.
	StartThumbCenter ui.Offset
	// EndThumbCenter is the center of the end thumb.
	EndThumbCenter ui.Offset
	// IsEnabled is whether the slider is enabled.
	IsEnabled bool
	// TextDirection is the directionality of the slider.
	TextDirection ui.TextDirection
}

// RangeSliderTickMarkShape is the base of the shapes of the tick marks of a
// discrete RangeSlider.
//
// Flutter: material/range_slider_parts.dart RangeSliderTickMarkShape
type RangeSliderTickMarkShape interface {
	// GetPreferredSize is the preferred size of a tick mark.
	//
	// Flutter: material/range_slider_parts.dart RangeSliderTickMarkShape.getPreferredSize
	GetPreferredSize(sliderTheme SliderThemeData, isEnabled bool) ui.Size
	// Paint paints a tick mark at the center.
	//
	// Flutter: material/range_slider_parts.dart RangeSliderTickMarkShape.paint
	Paint(context *rendering.PaintingContext, center ui.Offset, opts RangeSliderTickMarkPaintOpts)
}

// RangeSliderTrackPaintOpts are the named parameters of
// RangeSliderTrackShape.Paint.
//
// Flutter: material/range_slider_parts.dart RangeSliderTrackShape.paint
type RangeSliderTrackPaintOpts struct {
	// ParentBox is the box of the slider.
	ParentBox rendering.Box
	// SliderTheme is the theme of the slider, with the defaults filled in.
	SliderTheme SliderThemeData
	// EnableAnimation is the animation of the transition between enabled and
	// disabled.
	EnableAnimation animation.Animation[float64]
	// StartThumbCenter is the center of the start thumb.
	StartThumbCenter ui.Offset
	// EndThumbCenter is the center of the end thumb.
	EndThumbCenter ui.Offset
	// IsEnabled is whether the slider is enabled.
	IsEnabled bool
	// IsDiscrete is whether the slider has divisions.
	IsDiscrete bool
	// TextDirection is the directionality of the slider.
	TextDirection ui.TextDirection
	// AdditionalActiveTrackHeight is how much taller the active track is than the
	// inactive one; the default is 2.
	AdditionalActiveTrackHeight foundation.Opt[float64]
}

// RangeSliderTrackShape is the base of the shapes of the track of a RangeSlider.
//
// Flutter: material/range_slider_parts.dart RangeSliderTrackShape
type RangeSliderTrackShape interface {
	// GetPreferredRect is the bounds of the track.
	//
	// Flutter: material/range_slider_parts.dart RangeSliderTrackShape.getPreferredRect
	GetPreferredRect(opts SliderTrackRectOpts) ui.Rect
	// Paint paints the track.
	//
	// Flutter: material/range_slider_parts.dart RangeSliderTrackShape.paint
	Paint(context *rendering.PaintingContext, offset ui.Offset, opts RangeSliderTrackPaintOpts)
	// IsRounded is whether the track has rounded ends.
	//
	// Flutter: material/range_slider_parts.dart RangeSliderTrackShape.isRounded
	IsRounded() bool
}

// baseRangeSliderTrackRect is BaseRangeSliderTrackShape.getPreferredRect.
//
// Flutter: material/range_slider_parts.dart BaseRangeSliderTrackShape.getPreferredRect
func baseRangeSliderTrackRect(o SliderTrackRectOpts) ui.Rect {
	t := o.SliderTheme
	thumbSize := t.RangeThumbShape.GetPreferredSize(o.IsEnabled, o.IsDiscrete)
	overlayWidth := t.OverlayShape.GetPreferredSize(o.IsEnabled, o.IsDiscrete).Width
	trackHeight := t.TrackHeight.Value()
	if t.ActiveTrackColor == foundation.Some(Colors.Transparent) && t.InactiveTrackColor == foundation.Some(Colors.Transparent) {
		trackHeight = 0
	}
	left, sub := thumbSize.Width/2, thumbSize.Width
	if t.Padding == nil {
		left, sub = math.Max(overlayWidth/2, thumbSize.Width/2), math.Max(thumbSize.Width, overlayWidth)
	}
	trackLeft := o.Offset.Dx + left
	trackTop := o.Offset.Dy + (o.ParentBox.Size().Height-trackHeight)/2
	trackRight := trackLeft + o.ParentBox.Size().Width - sub
	trackBottom := trackTop + trackHeight
	return ui.NewRectFromLTRB(math.Min(trackLeft, trackRight), trackTop, math.Max(trackLeft, trackRight), trackBottom)
}

func rangeTrackPaints(o RangeSliderTrackPaintOpts) (active, inactive *ui.Paint) {
	t := o.SliderTheme
	return paintWithColor(colorTweenEvaluate(t.DisabledActiveTrackColor, t.ActiveTrackColor, o.EnableAnimation)),
		paintWithColor(colorTweenEvaluate(t.DisabledInactiveTrackColor, t.InactiveTrackColor, o.EnableAnimation))
}

func leftRightThumbs(o RangeSliderTrackPaintOpts) (left, right ui.Offset) {
	if o.TextDirection == ui.TextDirectionRTL {
		return o.EndThumbCenter, o.StartThumbCenter
	}
	return o.StartThumbCenter, o.EndThumbCenter
}

// RectangularRangeSliderTrackShape is the track that is a rectangle.
//
// Flutter: material/range_slider_parts.dart RectangularRangeSliderTrackShape
type RectangularRangeSliderTrackShape struct{}

// GetPreferredRect is the bounds of the track.
//
// Flutter: material/range_slider_parts.dart BaseRangeSliderTrackShape.getPreferredRect
func (RectangularRangeSliderTrackShape) GetPreferredRect(o SliderTrackRectOpts) ui.Rect {
	return baseRangeSliderTrackRect(o)
}

// IsRounded is false.
//
// Flutter: material/range_slider_parts.dart RangeSliderTrackShape.isRounded
func (RectangularRangeSliderTrackShape) IsRounded() bool { return false }

// Paint paints the track.
//
// Flutter: material/range_slider_parts.dart RectangularRangeSliderTrackShape.paint
func (s RectangularRangeSliderTrackShape) Paint(context *rendering.PaintingContext, offset ui.Offset, o RangeSliderTrackPaintOpts) {
	activePaint, inactivePaint := rangeTrackPaints(o)
	leftThumb, rightThumb := leftRightThumbs(o)
	trackRect := s.GetPreferredRect(SliderTrackRectOpts{ParentBox: o.ParentBox, Offset: offset, SliderTheme: o.SliderTheme, IsEnabled: o.IsEnabled, IsDiscrete: o.IsDiscrete})
	canvas := context.Canvas()
	if seg := ui.NewRectFromLTRB(trackRect.Left, trackRect.Top, leftThumb.Dx, trackRect.Bottom); !seg.IsEmpty() {
		canvas.DrawRect(seg, inactivePaint)
	}
	if seg := ui.NewRectFromLTRB(leftThumb.Dx, trackRect.Top, rightThumb.Dx, trackRect.Bottom); !seg.IsEmpty() {
		canvas.DrawRect(seg, activePaint)
	}
	if seg := ui.NewRectFromLTRB(rightThumb.Dx, trackRect.Top, trackRect.Right, trackRect.Bottom); !seg.IsEmpty() {
		canvas.DrawRect(seg, inactivePaint)
	}
}

// RoundedRectRangeSliderTrackShape is the track that is a rounded rectangle, with
// the active part taller.
//
// Flutter: material/range_slider_parts.dart RoundedRectRangeSliderTrackShape
type RoundedRectRangeSliderTrackShape struct{}

// GetPreferredRect is the bounds of the track.
//
// Flutter: material/range_slider_parts.dart BaseRangeSliderTrackShape.getPreferredRect
func (RoundedRectRangeSliderTrackShape) GetPreferredRect(o SliderTrackRectOpts) ui.Rect {
	return baseRangeSliderTrackRect(o)
}

// IsRounded is true.
//
// Flutter: material/range_slider_parts.dart RoundedRectRangeSliderTrackShape.isRounded
func (RoundedRectRangeSliderTrackShape) IsRounded() bool { return true }

// Paint paints the track.
//
// Flutter: material/range_slider_parts.dart RoundedRectRangeSliderTrackShape.paint
func (s RoundedRectRangeSliderTrackShape) Paint(context *rendering.PaintingContext, offset ui.Offset, o RangeSliderTrackPaintOpts) {
	t := o.SliderTheme
	if t.TrackHeight.IsNone() || t.TrackHeight.Value() <= 0 {
		return
	}
	additionalActiveTrackHeight := o.AdditionalActiveTrackHeight.OrElse(2)
	activePaint, inactivePaint := rangeTrackPaints(o)
	leftThumb, rightThumb := leftRightThumbs(o)
	trackRect := s.GetPreferredRect(SliderTrackRectOpts{ParentBox: o.ParentBox, Offset: offset, SliderTheme: t, IsEnabled: o.IsEnabled, IsDiscrete: o.IsDiscrete})
	trackRadius := ui.NewRadiusCircular(trackRect.Height() / 2)
	canvas := context.Canvas()
	canvas.DrawRRect(ui.NewRRectFromLTRBAndCorners(trackRect.Left, trackRect.Top, leftThumb.Dx, trackRect.Bottom,
		ui.Corners{TopLeft: trackRadius, BottomLeft: trackRadius}), inactivePaint)
	canvas.DrawRRect(ui.NewRRectFromLTRBAndCorners(rightThumb.Dx, trackRect.Top, trackRect.Right, trackRect.Bottom,
		ui.Corners{TopRight: trackRadius, BottomRight: trackRadius}), inactivePaint)
	canvas.DrawRRect(ui.NewRRectFromLTRBR(leftThumb.Dx-t.TrackHeight.Value()/2, trackRect.Top-additionalActiveTrackHeight/2,
		rightThumb.Dx+t.TrackHeight.Value()/2, trackRect.Bottom+additionalActiveTrackHeight/2, trackRadius), activePaint)
}

// GappedRangeSliderTrackShape is the track of Material 3: segments with a gap
// around the thumbs, and stop indicators at the ends.
//
// Flutter: material/range_slider_parts.dart GappedRangeSliderTrackShape
type GappedRangeSliderTrackShape struct{}

// GetPreferredRect is the bounds of the track.
//
// Flutter: material/range_slider_parts.dart BaseRangeSliderTrackShape.getPreferredRect
func (GappedRangeSliderTrackShape) GetPreferredRect(o SliderTrackRectOpts) ui.Rect {
	return baseRangeSliderTrackRect(o)
}

// IsRounded is true.
//
// Flutter: material/range_slider_parts.dart GappedRangeSliderTrackShape.isRounded
func (GappedRangeSliderTrackShape) IsRounded() bool { return true }

// Paint paints the track.
//
// Flutter: material/range_slider_parts.dart GappedRangeSliderTrackShape.paint
func (s GappedRangeSliderTrackShape) Paint(context *rendering.PaintingContext, offset ui.Offset, o RangeSliderTrackPaintOpts) {
	t := o.SliderTheme
	if t.TrackHeight.IsNone() || t.TrackHeight.Value() <= 0 {
		return
	}
	activePaint, inactivePaint := rangeTrackPaints(o)
	trackRect := s.GetPreferredRect(SliderTrackRectOpts{ParentBox: o.ParentBox, Offset: offset, SliderTheme: t, IsEnabled: o.IsEnabled, IsDiscrete: o.IsDiscrete})
	trackCornerRadius := ui.NewRadiusCircular(trackRect.ShortestSide() / 2)
	trackInsideCornerRadius := ui.NewRadiusCircular(2.0)
	leftThumb, rightThumb := leftRightThumbs(o)
	trackGap := t.TrackGap.Value()
	trackHeight := t.TrackHeight.Value()
	canvas := context.Canvas()

	trackRRect := ui.NewRRectFromRectAndCorners(trackRect, ui.Corners{TopLeft: trackCornerRadius, BottomLeft: trackCornerRadius, TopRight: trackCornerRadius, BottomRight: trackCornerRadius})
	leftRRect := ui.NewRRectFromLTRBAndCorners(trackRect.Left, trackRect.Top, leftThumb.Dx-trackGap, trackRect.Bottom,
		ui.Corners{TopLeft: trackCornerRadius, BottomLeft: trackCornerRadius, TopRight: trackInsideCornerRadius, BottomRight: trackInsideCornerRadius})
	rightRRect := ui.NewRRectFromLTRBAndCorners(rightThumb.Dx+trackGap, trackRect.Top, trackRect.Right, trackRect.Bottom,
		ui.Corners{TopLeft: trackInsideCornerRadius, BottomLeft: trackInsideCornerRadius, TopRight: trackCornerRadius, BottomRight: trackCornerRadius})

	canvas.Save()
	canvas.ClipRRect(trackRRect)
	drawLeftTrack := o.StartThumbCenter.Dx > (leftRRect.Left + trackHeight/2)
	drawRightTrack := o.EndThumbCenter.Dx < (rightRRect.Right - trackHeight/2)
	if drawLeftTrack {
		canvas.DrawRRect(leftRRect, inactivePaint)
	}
	if drawRightTrack {
		canvas.DrawRRect(rightRRect, inactivePaint)
	}
	if leftThumb.Dx+trackGap < rightThumb.Dx-trackGap {
		canvas.DrawRRect(ui.NewRRectFromLTRBR(leftThumb.Dx+trackGap, trackRect.Top, rightThumb.Dx-trackGap, trackRect.Bottom, trackInsideCornerRadius), activePaint)
	}
	canvas.Restore()

	const stopIndicatorRadius = 2.0
	stopIndicatorTrailingSpace := trackHeight / 2
	startStopIndicatorOffset := ui.Offset{Dx: trackRect.CenterLeft().Dx + stopIndicatorTrailingSpace, Dy: trackRect.Center().Dy}
	endStopIndicatorOffset := ui.Offset{Dx: trackRect.CenterRight().Dx - stopIndicatorTrailingSpace, Dy: trackRect.Center().Dy}
	if o.StartThumbCenter.Dx > startStopIndicatorOffset.Dx && !o.IsDiscrete {
		canvas.DrawCircle(startStopIndicatorOffset, stopIndicatorRadius, activePaint)
	}
	if o.EndThumbCenter.Dx < endStopIndicatorOffset.Dx && !o.IsDiscrete {
		canvas.DrawCircle(endStopIndicatorOffset, stopIndicatorRadius, activePaint)
	}
}

// RoundRangeSliderTickMarkShape is the tick mark that is a circle.
//
// Flutter: material/range_slider_parts.dart RoundRangeSliderTickMarkShape
type RoundRangeSliderTickMarkShape struct {
	// TickMarkRadius is the radius of the tick mark; the default is a quarter of
	// the height of the track.
	TickMarkRadius foundation.Opt[float64]
}

// NewRoundRangeSliderTickMarkShape creates the shape.
//
// Flutter: material/range_slider_parts.dart RoundRangeSliderTickMarkShape.new
func NewRoundRangeSliderTickMarkShape(opts ...RoundRangeSliderTickMarkShape) RoundRangeSliderTickMarkShape {
	return firstOpts(opts)
}

// GetPreferredSize is the size of the circle.
//
// Flutter: material/range_slider_parts.dart RoundRangeSliderTickMarkShape.getPreferredSize
func (s RoundRangeSliderTickMarkShape) GetPreferredSize(sliderTheme SliderThemeData, _ bool) ui.Size {
	return sizeFromRadius(s.TickMarkRadius.OrElse(sliderTheme.TrackHeight.Value() / 4))
}

// Paint paints the circle.
//
// Flutter: material/range_slider_parts.dart RoundRangeSliderTickMarkShape.paint
func (s RoundRangeSliderTickMarkShape) Paint(context *rendering.PaintingContext, center ui.Offset, o RangeSliderTickMarkPaintOpts) {
	t := o.SliderTheme
	hasGap := t.TrackGap.IsSome() && t.TrackGap.Value() > 0
	underThumb := o.StartThumbCenter.Dx == center.Dx || o.EndThumbCenter.Dx == center.Dx
	if hasGap && underThumb {
		return
	}
	var isBetweenThumbs bool
	if o.TextDirection == ui.TextDirectionRTL {
		isBetweenThumbs = o.EndThumbCenter.Dx < center.Dx && center.Dx < o.StartThumbCenter.Dx
	} else {
		isBetweenThumbs = o.StartThumbCenter.Dx < center.Dx && center.Dx < o.EndThumbCenter.Dx
	}
	begin, end := t.DisabledInactiveTickMarkColor, t.InactiveTickMarkColor
	if isBetweenThumbs {
		begin, end = t.DisabledActiveTickMarkColor, t.ActiveTickMarkColor
	}
	paint := paintWithColor(colorTweenEvaluate(begin, end, o.EnableAnimation))
	radius := s.GetPreferredSize(t, o.IsEnabled).Width / 2
	if radius > 0 {
		context.Canvas().DrawCircle(center, radius, paint)
	}
}

// RoundRangeSliderThumbShape is the thumb that is a circle with a shadow.
//
// Flutter: material/range_slider_parts.dart RoundRangeSliderThumbShape
type RoundRangeSliderThumbShape struct {
	// EnabledThumbRadius is the radius of the thumb when the slider is enabled;
	// the default is 10.
	EnabledThumbRadius foundation.Opt[float64]
	// DisabledThumbRadius is the radius of the thumb when the slider is disabled;
	// the default is EnabledThumbRadius.
	DisabledThumbRadius foundation.Opt[float64]
	// Elevation is the elevation of the thumb; the default is 1.
	Elevation foundation.Opt[float64]
	// PressedElevation is the elevation of the thumb while it is pressed; the
	// default is 6.
	PressedElevation foundation.Opt[float64]
}

// NewRoundRangeSliderThumbShape creates the shape.
//
// Flutter: material/range_slider_parts.dart RoundRangeSliderThumbShape.new
func NewRoundRangeSliderThumbShape(opts ...RoundRangeSliderThumbShape) RoundRangeSliderThumbShape {
	return firstOpts(opts)
}

func (s RoundRangeSliderThumbShape) enabledRadius() float64 { return s.EnabledThumbRadius.OrElse(10.0) }
func (s RoundRangeSliderThumbShape) disabledRadius() float64 {
	return s.DisabledThumbRadius.OrElse(s.enabledRadius())
}

// GetPreferredSize is the size of the circle.
//
// Flutter: material/range_slider_parts.dart RoundRangeSliderThumbShape.getPreferredSize
func (s RoundRangeSliderThumbShape) GetPreferredSize(isEnabled, _ bool) ui.Size {
	if isEnabled {
		return sizeFromRadius(s.enabledRadius())
	}
	return sizeFromRadius(s.disabledRadius())
}

// Paint paints the circle and its shadow.
//
// Flutter: material/range_slider_parts.dart RoundRangeSliderThumbShape.paint
func (s RoundRangeSliderThumbShape) Paint(context *rendering.PaintingContext, center ui.Offset, o RangeSliderThumbPaintOpts) {
	canvas := context.Canvas()
	t := o.SliderTheme
	radius := animation.NewTween(s.disabledRadius(), s.enabledRadius()).Evaluate(o.EnableAnimation)
	elevation := s.Elevation.OrElse(1.0)
	if o.IsOnTop {
		canvas.DrawCircle(center, radius, strokePaintOf(t.OverlappingShapeStrokeColor.Value()))
	}
	color := colorTweenEvaluate(t.DisabledThumbColor, t.ThumbColor, o.EnableAnimation)
	evaluatedElevation := elevation
	if o.IsPressed {
		evaluatedElevation = animation.NewTween(elevation, s.PressedElevation.OrElse(6.0)).Evaluate(o.ActivationAnimation)
	}
	shadowPath := ui.NewPath()
	shadowPath.AddArc(ui.NewRectFromCenter(center, 2*radius, 2*radius), 0, math.Pi*2)
	canvas.DrawShadow(shadowPath, Colors.Black, evaluatedElevation, true)
	canvas.DrawCircle(center, radius, paintWithColor(color))
}

// HandleRangeSliderThumbShape is the thumb of Material 3: a vertical bar with
// rounded ends.
//
// Flutter: material/range_slider_parts.dart HandleRangeSliderThumbShape
type HandleRangeSliderThumbShape struct{}

// GetPreferredSize is 4 by 44.
//
// Flutter: material/range_slider_parts.dart HandleRangeSliderThumbShape.getPreferredSize
func (HandleRangeSliderThumbShape) GetPreferredSize(bool, bool) ui.Size {
	return ui.Size{Width: 4.0, Height: 44.0}
}

// Paint paints the bar, with the size that the theme gives.
//
// Flutter: material/range_slider_parts.dart HandleRangeSliderThumbShape.paint
func (HandleRangeSliderThumbShape) Paint(context *rendering.PaintingContext, center ui.Offset, o RangeSliderThumbPaintOpts) {
	color := colorTweenEvaluate(o.SliderTheme.DisabledThumbColor, o.SliderTheme.ThumbColor, o.EnableAnimation)
	thumbSize := o.SliderTheme.ThumbSize.Resolve(0).Value()
	rrect := ui.NewRRectFromRectAndRadius(ui.NewRectFromCenter(center, thumbSize.Width, thumbSize.Height),
		ui.NewRadiusCircular(math.Min(math.Abs(thumbSize.Width), math.Abs(thumbSize.Height))/2))
	context.Canvas().DrawRRect(rrect, paintWithColor(color))
}

func rangeIndicatorArgs(context *rendering.PaintingContext, center ui.Offset, o RangeSliderValueIndicatorPaintOpts) valueIndicatorPaintArgs {
	stroke := o.SliderTheme.ValueIndicatorStrokeColor
	if o.IsOnTop {
		stroke = o.SliderTheme.OverlappingShapeStrokeColor
	}
	return valueIndicatorPaintArgs{
		parentBox: o.ParentBox, canvas: context.Canvas(), center: center, scale: o.ActivationAnimation.Value(),
		labelPainter: o.LabelPainter, textScaleFactor: o.TextScaleFactor, sizeWithOverflow: o.SizeWithOverflow,
		backgroundPaintColor: o.SliderTheme.ValueIndicatorColor.Value(), strokePaintColor: stroke}
}

// RoundedRectRangeSliderValueIndicatorShape is the value indicator that is a pill.
//
// Flutter: material/range_slider_parts.dart RoundedRectRangeSliderValueIndicatorShape
type RoundedRectRangeSliderValueIndicatorShape struct {
	RangeSliderValueIndicatorShapeBase
}

// GetPreferredSize is the size for the label.
//
// Flutter: material/range_slider_parts.dart RoundedRectRangeSliderValueIndicatorShape.getPreferredSize
func (RoundedRectRangeSliderValueIndicatorShape) GetPreferredSize(_, _ bool, labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	return roundedRectValueIndicatorPathPainter{}.getPreferredSize(labelPainter, textScaleFactor)
}

// Paint paints the pill with the label.
//
// Flutter: material/range_slider_parts.dart RoundedRectRangeSliderValueIndicatorShape.paint
func (RoundedRectRangeSliderValueIndicatorShape) Paint(context *rendering.PaintingContext, center ui.Offset, o RangeSliderValueIndicatorPaintOpts) {
	roundedRectValueIndicatorPathPainter{}.paint(rangeIndicatorArgs(context, center, o))
}

// DropRangeSliderValueIndicatorShape is the value indicator that is a drop.
//
// Flutter: material/range_slider_parts.dart DropRangeSliderValueIndicatorShape
type DropRangeSliderValueIndicatorShape struct {
	RangeSliderValueIndicatorShapeBase
}

// GetPreferredSize is the size for the label.
//
// Flutter: material/range_slider_parts.dart DropRangeSliderValueIndicatorShape.getPreferredSize
func (DropRangeSliderValueIndicatorShape) GetPreferredSize(_, _ bool, labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	return dropValueIndicatorPathPainter{}.getPreferredSize(labelPainter, textScaleFactor)
}

// Paint paints the drop with the label.
//
// Flutter: material/range_slider_parts.dart DropRangeSliderValueIndicatorShape.paint
func (DropRangeSliderValueIndicatorShape) Paint(context *rendering.PaintingContext, center ui.Offset, o RangeSliderValueIndicatorPaintOpts) {
	dropValueIndicatorPathPainter{}.paint(rangeIndicatorArgs(context, center, o))
}

// RectangularRangeSliderValueIndicatorShape is the value indicator that is a
// rectangle with a notch.
//
// Flutter: material/slider_value_indicator_shape.dart RectangularRangeSliderValueIndicatorShape
type RectangularRangeSliderValueIndicatorShape struct{}

// GetPreferredSize is the size for the label.
//
// Flutter: material/slider_value_indicator_shape.dart RectangularRangeSliderValueIndicatorShape.getPreferredSize
func (RectangularRangeSliderValueIndicatorShape) GetPreferredSize(_, _ bool, labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	return rectangularValueIndicatorPathPainter{}.getPreferredSize(labelPainter, textScaleFactor)
}

// GetHorizontalShift is the shift that keeps the indicator in the view.
//
// Flutter: material/slider_value_indicator_shape.dart RectangularRangeSliderValueIndicatorShape.getHorizontalShift
func (RectangularRangeSliderValueIndicatorShape) GetHorizontalShift(o RangeSliderHorizontalShiftOpts) float64 {
	return rectangularValueIndicatorPathPainter{}.horizontalShift(valueIndicatorPaintArgs{
		parentBox: o.ParentBox, center: o.Center, labelPainter: o.LabelPainter, textScaleFactor: o.TextScaleFactor,
		sizeWithOverflow: o.SizeWithOverflow, scale: o.ActivationAnimation.Value()})
}

// Paint paints the rectangle with the label.
//
// Flutter: material/slider_value_indicator_shape.dart RectangularRangeSliderValueIndicatorShape.paint
func (RectangularRangeSliderValueIndicatorShape) Paint(context *rendering.PaintingContext, center ui.Offset, o RangeSliderValueIndicatorPaintOpts) {
	rectangularValueIndicatorPathPainter{}.paint(rangeIndicatorArgs(context, center, o))
}

// PaddleRangeSliderValueIndicatorShape is the value indicator that is a paddle.
//
// Flutter: material/slider_value_indicator_shape.dart PaddleRangeSliderValueIndicatorShape
type PaddleRangeSliderValueIndicatorShape struct{}

// GetPreferredSize is the size for the label.
//
// Flutter: material/slider_value_indicator_shape.dart PaddleRangeSliderValueIndicatorShape.getPreferredSize
func (PaddleRangeSliderValueIndicatorShape) GetPreferredSize(_, _ bool, labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	return paddleValueIndicatorPathPainter{}.getPreferredSize(labelPainter, textScaleFactor)
}

// GetHorizontalShift is the shift that keeps the indicator in the view.
//
// Flutter: material/slider_value_indicator_shape.dart PaddleRangeSliderValueIndicatorShape.getHorizontalShift
func (PaddleRangeSliderValueIndicatorShape) GetHorizontalShift(o RangeSliderHorizontalShiftOpts) float64 {
	return paddleValueIndicatorPathPainter{}.horizontalShift(o.Center, o.LabelPainter, o.ActivationAnimation.Value(), o.TextScaleFactor, o.SizeWithOverflow)
}

// Paint paints the paddle with the label.
//
// Flutter: material/slider_value_indicator_shape.dart PaddleRangeSliderValueIndicatorShape.paint
func (PaddleRangeSliderValueIndicatorShape) Paint(context *rendering.PaintingContext, center ui.Offset, o RangeSliderValueIndicatorPaintOpts) {
	enableColor := colorTweenEvaluate(o.SliderTheme.DisabledThumbColor, o.SliderTheme.ValueIndicatorColor, o.EnableAnimation)
	stroke := o.SliderTheme.ValueIndicatorStrokeColor
	if o.IsOnTop {
		stroke = o.SliderTheme.OverlappingShapeStrokeColor
	}
	paddleValueIndicatorPathPainter{}.paint(context.Canvas(), center, paintWithColor(enableColor), o.ActivationAnimation.Value(),
		o.LabelPainter, o.TextScaleFactor, o.SizeWithOverflow, stroke)
}
