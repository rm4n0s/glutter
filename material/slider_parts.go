// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// SliderComponentPaintOpts are the named parameters of SliderComponentShape.Paint.
//
// Flutter: material/slider_value_indicator_shape.dart SliderComponentShape.paint
type SliderComponentPaintOpts struct {
	// ActivationAnimation is the animation that gives the opening of the shape
	// (the overlay and the value indicator) or the pressing of the thumb.
	ActivationAnimation animation.Animation[float64]
	// EnableAnimation is the animation that gives the transition between enabled
	// and disabled.
	EnableAnimation animation.Animation[float64]
	// IsDiscrete is whether the slider has divisions.
	IsDiscrete bool
	// LabelPainter paints the label of the value indicator.
	LabelPainter *painting.TextPainter
	// ParentBox is the box of the slider.
	ParentBox rendering.Box
	// SliderTheme is the theme of the slider, with the defaults filled in.
	SliderTheme SliderThemeData
	// TextDirection is the directionality of the slider.
	TextDirection ui.TextDirection
	// Value is the value of the slider, from 0 to 1.
	Value float64
	// TextScaleFactor is the scale of the label.
	TextScaleFactor float64
	// SizeWithOverflow is the size of the area that the shape may paint in, to
	// shift the value indicator back into the view.
	SizeWithOverflow ui.Size
}

// SliderComponentShape is the base of the shapes of the thumb, the overlay and the
// value indicator of a Slider.
//
// Flutter: material/slider_value_indicator_shape.dart SliderComponentShape
type SliderComponentShape interface {
	// GetPreferredSize is the preferred size of the shape.
	//
	// Flutter: material/slider_value_indicator_shape.dart SliderComponentShape.getPreferredSize
	GetPreferredSize(isEnabled, isDiscrete bool) ui.Size
	// Paint paints the shape around the center.
	//
	// Flutter: material/slider_value_indicator_shape.dart SliderComponentShape.paint
	Paint(context *rendering.PaintingContext, center ui.Offset, opts SliderComponentPaintOpts)
}

// SliderValueIndicatorShape is a SliderComponentShape whose size depends on the
// label.
//
// Deviation: Dart's getPreferredSize takes the label painter and the text scale as
// optional named parameters; here they have a method of their own, and
// GetPreferredSize of these shapes is the size of an empty label (SPEC §12.10).
//
// Flutter: material/slider_value_indicator_shape.dart SliderComponentShape.getPreferredSize
type SliderValueIndicatorShape interface {
	SliderComponentShape
	// GetPreferredSizeForLabel is the preferred size of the shape for the label.
	//
	// Flutter: material/slider_value_indicator_shape.dart SliderComponentShape.getPreferredSize
	GetPreferredSizeForLabel(isEnabled, isDiscrete bool, labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size
}

type emptySliderComponentShape struct{}

func (emptySliderComponentShape) GetPreferredSize(bool, bool) ui.Size { return ui.Size{} }
func (emptySliderComponentShape) Paint(*rendering.PaintingContext, ui.Offset, SliderComponentPaintOpts) {
}

// SliderComponentShapeNoThumb is a shape that paints nothing and has no size, to
// leave the thumb out. It is read-only.
//
// Flutter: material/slider_value_indicator_shape.dart SliderComponentShape.noThumb
var SliderComponentShapeNoThumb SliderComponentShape = emptySliderComponentShape{}

// SliderComponentShapeNoOverlay is a shape that paints nothing and has no size, to
// leave the overlay out. It is read-only.
//
// Flutter: material/slider_value_indicator_shape.dart SliderComponentShape.noOverlay
var SliderComponentShapeNoOverlay SliderComponentShape = emptySliderComponentShape{}

// colorTweenEvaluate is ColorTween(begin, end).evaluate(animation)!.
func colorTweenEvaluate(begin, end foundation.Opt[ui.Color], a animation.Animation[float64]) ui.Color {
	return animation.NewColorTween(begin, end).Evaluate(a).Value()
}

func paintWithColor(c ui.Color) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(c)
	return p
}

// RoundSliderOverlayShape is the overlay that is a circle around the thumb.
//
// Flutter: material/slider_value_indicator_shape.dart RoundSliderOverlayShape
type RoundSliderOverlayShape struct {
	// OverlayRadius is the radius of the overlay; the default is 24.
	OverlayRadius foundation.Opt[float64]
}

// NewRoundSliderOverlayShape creates the shape.
//
// Flutter: material/slider_value_indicator_shape.dart RoundSliderOverlayShape.new
func NewRoundSliderOverlayShape(opts ...RoundSliderOverlayShape) RoundSliderOverlayShape {
	return firstOpts(opts)
}

func (s RoundSliderOverlayShape) radius() float64 { return s.OverlayRadius.OrElse(24.0) }

// GetPreferredSize is the size of the circle.
//
// Flutter: material/slider_value_indicator_shape.dart RoundSliderOverlayShape.getPreferredSize
func (s RoundSliderOverlayShape) GetPreferredSize(bool, bool) ui.Size {
	return sizeFromRadius(s.radius())
}

// Paint paints the circle, growing with the activation animation.
//
// Flutter: material/slider_value_indicator_shape.dart RoundSliderOverlayShape.paint
func (s RoundSliderOverlayShape) Paint(context *rendering.PaintingContext, center ui.Offset, o SliderComponentPaintOpts) {
	radius := animation.NewTween(0.0, s.radius()).Evaluate(o.ActivationAnimation)
	context.Canvas().DrawCircle(center, radius, paintWithColor(o.SliderTheme.OverlayColor.Value()))
}

func sizeFromRadius(r float64) ui.Size { return ui.Size{Width: r * 2, Height: r * 2} }

// SliderTickMarkPaintOpts are the named parameters of SliderTickMarkShape.Paint.
//
// Flutter: material/slider_parts.dart SliderTickMarkShape.paint
type SliderTickMarkPaintOpts struct {
	// ParentBox is the box of the slider.
	ParentBox rendering.Box
	// SliderTheme is the theme of the slider, with the defaults filled in.
	SliderTheme SliderThemeData
	// EnableAnimation is the animation of the transition between enabled and
	// disabled.
	EnableAnimation animation.Animation[float64]
	// ThumbCenter is the center of the thumb.
	ThumbCenter ui.Offset
	// IsEnabled is whether the slider is enabled.
	IsEnabled bool
	// TextDirection is the directionality of the slider.
	TextDirection ui.TextDirection
}

// SliderTickMarkShape is the base of the shapes of the tick marks of a discrete
// Slider.
//
// Flutter: material/slider_parts.dart SliderTickMarkShape
type SliderTickMarkShape interface {
	// GetPreferredSize is the preferred size of a tick mark.
	//
	// Flutter: material/slider_parts.dart SliderTickMarkShape.getPreferredSize
	GetPreferredSize(sliderTheme SliderThemeData, isEnabled bool) ui.Size
	// Paint paints a tick mark at the center.
	//
	// Flutter: material/slider_parts.dart SliderTickMarkShape.paint
	Paint(context *rendering.PaintingContext, center ui.Offset, opts SliderTickMarkPaintOpts)
}

type emptySliderTickMarkShape struct{}

func (emptySliderTickMarkShape) GetPreferredSize(SliderThemeData, bool) ui.Size { return ui.Size{} }
func (emptySliderTickMarkShape) Paint(*rendering.PaintingContext, ui.Offset, SliderTickMarkPaintOpts) {
}

// SliderTickMarkShapeNoTickMark is a shape that paints nothing and has no size,
// to leave the tick marks out. It is read-only.
//
// Flutter: material/slider_parts.dart SliderTickMarkShape.noTickMark
var SliderTickMarkShapeNoTickMark SliderTickMarkShape = emptySliderTickMarkShape{}

// RoundSliderTickMarkShape is the tick mark that is a circle.
//
// Flutter: material/slider_parts.dart RoundSliderTickMarkShape
type RoundSliderTickMarkShape struct {
	// TickMarkRadius is the radius of the tick mark; the default is a quarter of
	// the height of the track.
	TickMarkRadius foundation.Opt[float64]
}

// NewRoundSliderTickMarkShape creates the shape.
//
// Flutter: material/slider_parts.dart RoundSliderTickMarkShape.new
func NewRoundSliderTickMarkShape(opts ...RoundSliderTickMarkShape) RoundSliderTickMarkShape {
	return firstOpts(opts)
}

// GetPreferredSize is the size of the circle.
//
// Flutter: material/slider_parts.dart RoundSliderTickMarkShape.getPreferredSize
func (s RoundSliderTickMarkShape) GetPreferredSize(sliderTheme SliderThemeData, _ bool) ui.Size {
	return sizeFromRadius(s.TickMarkRadius.OrElse(sliderTheme.TrackHeight.Value() / 4))
}

// Paint paints the circle.
//
// Flutter: material/slider_parts.dart RoundSliderTickMarkShape.paint
func (s RoundSliderTickMarkShape) Paint(context *rendering.PaintingContext, center ui.Offset, o SliderTickMarkPaintOpts) {
	t := o.SliderTheme
	xOffset := center.Dx - o.ThumbCenter.Dx
	var begin, end foundation.Opt[ui.Color]
	if (o.TextDirection == ui.TextDirectionLTR && xOffset > 0) || (o.TextDirection == ui.TextDirectionRTL && xOffset < 0) {
		begin, end = t.DisabledInactiveTickMarkColor, t.InactiveTickMarkColor
	} else {
		begin, end = t.DisabledActiveTickMarkColor, t.ActiveTickMarkColor
	}
	paint := paintWithColor(colorTweenEvaluate(begin, end, o.EnableAnimation))
	radius := s.GetPreferredSize(t, o.IsEnabled).Width / 2
	if radius > 0 {
		context.Canvas().DrawCircle(center, radius, paint)
	}
}

// SliderTrackRectOpts are the named parameters of
// SliderTrackShape.GetPreferredRect.
//
// Flutter: material/slider_parts.dart SliderTrackShape.getPreferredRect
type SliderTrackRectOpts struct {
	// ParentBox is the box of the slider.
	ParentBox rendering.Box
	// Offset is where the box is painted; the default is zero.
	Offset ui.Offset
	// SliderTheme is the theme of the slider, with the defaults filled in.
	SliderTheme SliderThemeData
	// IsEnabled is whether the slider is enabled.
	IsEnabled bool
	// IsDiscrete is whether the slider has divisions.
	IsDiscrete bool
}

// SliderTrackPaintOpts are the named parameters of SliderTrackShape.Paint.
//
// Flutter: material/slider_parts.dart SliderTrackShape.paint
type SliderTrackPaintOpts struct {
	// ParentBox is the box of the slider.
	ParentBox rendering.Box
	// SliderTheme is the theme of the slider, with the defaults filled in.
	SliderTheme SliderThemeData
	// EnableAnimation is the animation of the transition between enabled and
	// disabled.
	EnableAnimation animation.Animation[float64]
	// ThumbCenter is the center of the thumb.
	ThumbCenter ui.Offset
	// SecondaryOffset is the end of the secondary track; none for no secondary
	// track.
	SecondaryOffset foundation.Opt[ui.Offset]
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

// SliderTrackShape is the base of the shapes of the track of a Slider.
//
// Flutter: material/slider_parts.dart SliderTrackShape
type SliderTrackShape interface {
	// GetPreferredRect is the bounds of the track.
	//
	// Flutter: material/slider_parts.dart SliderTrackShape.getPreferredRect
	GetPreferredRect(opts SliderTrackRectOpts) ui.Rect
	// Paint paints the track.
	//
	// Flutter: material/slider_parts.dart SliderTrackShape.paint
	Paint(context *rendering.PaintingContext, offset ui.Offset, opts SliderTrackPaintOpts)
	// IsRounded is whether the track has rounded ends.
	//
	// Flutter: material/slider_parts.dart SliderTrackShape.isRounded
	IsRounded() bool
}

// baseSliderTrackRect is BaseSliderTrackShape.getPreferredRect.
//
// Flutter: material/slider_parts.dart BaseSliderTrackShape.getPreferredRect
func baseSliderTrackRect(o SliderTrackRectOpts) ui.Rect {
	t := o.SliderTheme
	thumbWidth := t.ThumbShape.GetPreferredSize(o.IsEnabled, o.IsDiscrete).Width
	overlayWidth := t.OverlayShape.GetPreferredSize(o.IsEnabled, o.IsDiscrete).Width
	trackHeight := t.TrackHeight.Value()
	if t.ActiveTrackColor == foundation.Some(Colors.Transparent) && t.InactiveTrackColor == foundation.Some(Colors.Transparent) {
		trackHeight = 0
	}
	extra := 0.0
	if t.Padding == nil {
		extra = math.Max(overlayWidth/2, thumbWidth/2)
	}
	trackLeft := o.Offset.Dx + extra
	trackTop := o.Offset.Dy + (o.ParentBox.Size().Height-trackHeight)/2
	sub := 0.0
	if t.Padding == nil {
		sub = math.Max(thumbWidth, overlayWidth)
	}
	trackRight := trackLeft + o.ParentBox.Size().Width - sub
	trackBottom := trackTop + trackHeight
	return ui.NewRectFromLTRB(math.Min(trackLeft, trackRight), trackTop, math.Max(trackLeft, trackRight), trackBottom)
}

// trackPaints are the paints of the active and the inactive track.
func trackPaints(o SliderTrackPaintOpts) (active, inactive *ui.Paint) {
	t := o.SliderTheme
	return paintWithColor(colorTweenEvaluate(t.DisabledActiveTrackColor, t.ActiveTrackColor, o.EnableAnimation)),
		paintWithColor(colorTweenEvaluate(t.DisabledInactiveTrackColor, t.InactiveTrackColor, o.EnableAnimation))
}

// RectangularSliderTrackShape is the track that is a rectangle.
//
// Flutter: material/slider_parts.dart RectangularSliderTrackShape
type RectangularSliderTrackShape struct{}

// GetPreferredRect is the bounds of the track.
//
// Flutter: material/slider_parts.dart BaseSliderTrackShape.getPreferredRect
func (RectangularSliderTrackShape) GetPreferredRect(o SliderTrackRectOpts) ui.Rect {
	return baseSliderTrackRect(o)
}

// IsRounded is false.
//
// Flutter: material/slider_parts.dart BaseSliderTrackShape.isRounded
func (RectangularSliderTrackShape) IsRounded() bool { return false }

// Paint paints the track.
//
// Flutter: material/slider_parts.dart RectangularSliderTrackShape.paint
func (s RectangularSliderTrackShape) Paint(context *rendering.PaintingContext, offset ui.Offset, o SliderTrackPaintOpts) {
	t := o.SliderTheme
	if t.TrackHeight.Value() <= 0 {
		return
	}
	activePaint, inactivePaint := trackPaints(o)
	leftTrackPaint, rightTrackPaint := activePaint, inactivePaint
	if o.TextDirection == ui.TextDirectionRTL {
		leftTrackPaint, rightTrackPaint = inactivePaint, activePaint
	}
	trackRect := s.GetPreferredRect(SliderTrackRectOpts{ParentBox: o.ParentBox, Offset: offset, SliderTheme: t, IsEnabled: o.IsEnabled, IsDiscrete: o.IsDiscrete})
	canvas := context.Canvas()
	left := ui.NewRectFromLTRB(trackRect.Left, trackRect.Top, o.ThumbCenter.Dx, trackRect.Bottom)
	if !left.IsEmpty() {
		canvas.DrawRect(left, leftTrackPaint)
	}
	right := ui.NewRectFromLTRB(o.ThumbCenter.Dx, trackRect.Top, trackRect.Right, trackRect.Bottom)
	if !right.IsEmpty() {
		canvas.DrawRect(right, rightTrackPaint)
	}
	sec, hasSecondary := o.SecondaryOffset.Get()
	showSecondaryTrack := hasSecondary && ((o.TextDirection == ui.TextDirectionRTL && sec.Dx < o.ThumbCenter.Dx) ||
		(o.TextDirection == ui.TextDirectionLTR && sec.Dx > o.ThumbCenter.Dx))
	if showSecondaryTrack {
		secondaryPaint := paintWithColor(colorTweenEvaluate(t.DisabledSecondaryActiveTrackColor, t.SecondaryActiveTrackColor, o.EnableAnimation))
		var segment ui.Rect
		if o.TextDirection == ui.TextDirectionRTL {
			segment = ui.NewRectFromLTRB(sec.Dx, trackRect.Top, o.ThumbCenter.Dx, trackRect.Bottom)
		} else {
			segment = ui.NewRectFromLTRB(o.ThumbCenter.Dx, trackRect.Top, sec.Dx, trackRect.Bottom)
		}
		if !segment.IsEmpty() {
			canvas.DrawRect(segment, secondaryPaint)
		}
	}
}

// RoundedRectSliderTrackShape is the track that is a rounded rectangle, with the
// active part taller.
//
// Flutter: material/slider_parts.dart RoundedRectSliderTrackShape
type RoundedRectSliderTrackShape struct{}

// GetPreferredRect is the bounds of the track.
//
// Flutter: material/slider_parts.dart BaseSliderTrackShape.getPreferredRect
func (RoundedRectSliderTrackShape) GetPreferredRect(o SliderTrackRectOpts) ui.Rect {
	return baseSliderTrackRect(o)
}

// IsRounded is true.
//
// Flutter: material/slider_parts.dart RoundedRectSliderTrackShape.isRounded
func (RoundedRectSliderTrackShape) IsRounded() bool { return true }

// Paint paints the track.
//
// Flutter: material/slider_parts.dart RoundedRectSliderTrackShape.paint
func (s RoundedRectSliderTrackShape) Paint(context *rendering.PaintingContext, offset ui.Offset, o SliderTrackPaintOpts) {
	t := o.SliderTheme
	if t.TrackHeight.IsNone() || t.TrackHeight.Value() <= 0 {
		return
	}
	additionalActiveTrackHeight := o.AdditionalActiveTrackHeight.OrElse(2)
	activePaint, inactivePaint := trackPaints(o)
	leftTrackPaint, rightTrackPaint := activePaint, inactivePaint
	if o.TextDirection == ui.TextDirectionRTL {
		leftTrackPaint, rightTrackPaint = inactivePaint, activePaint
	}
	trackRect := s.GetPreferredRect(SliderTrackRectOpts{ParentBox: o.ParentBox, Offset: offset, SliderTheme: t, IsEnabled: o.IsEnabled, IsDiscrete: o.IsDiscrete})
	trackRadius := ui.NewRadiusCircular(trackRect.Height() / 2)
	activeTrackRadius := ui.NewRadiusCircular((trackRect.Height() + additionalActiveTrackHeight) / 2)
	isLTR := o.TextDirection == ui.TextDirectionLTR
	isRTL := o.TextDirection == ui.TextDirectionRTL
	canvas := context.Canvas()
	trackHeight := t.TrackHeight.Value()

	if o.ThumbCenter.Dx < (trackRect.Right - trackHeight/2) {
		top, bottom := trackRect.Top, trackRect.Bottom
		if isRTL {
			top -= additionalActiveTrackHeight / 2
			bottom += additionalActiveTrackHeight / 2
		}
		radius := activeTrackRadius
		if isLTR {
			radius = trackRadius
		}
		canvas.DrawRRect(ui.NewRRectFromLTRBR(o.ThumbCenter.Dx-trackHeight/2, top, trackRect.Right, bottom, radius), rightTrackPaint)
	}
	if o.ThumbCenter.Dx > (trackRect.Left + trackHeight/2) {
		top, bottom := trackRect.Top, trackRect.Bottom
		if isLTR {
			top -= additionalActiveTrackHeight / 2
			bottom += additionalActiveTrackHeight / 2
		}
		radius := trackRadius
		if isLTR {
			radius = activeTrackRadius
		}
		canvas.DrawRRect(ui.NewRRectFromLTRBR(trackRect.Left, top, o.ThumbCenter.Dx+trackHeight/2, bottom, radius), leftTrackPaint)
	}

	sec, hasSecondary := o.SecondaryOffset.Get()
	showSecondaryTrack := hasSecondary && ((isLTR && sec.Dx > o.ThumbCenter.Dx) || (!isLTR && sec.Dx < o.ThumbCenter.Dx))
	if showSecondaryTrack {
		secondaryPaint := paintWithColor(colorTweenEvaluate(t.DisabledSecondaryActiveTrackColor, t.SecondaryActiveTrackColor, o.EnableAnimation))
		if isLTR {
			canvas.DrawRRect(ui.NewRRectFromLTRBAndCorners(o.ThumbCenter.Dx, trackRect.Top, sec.Dx, trackRect.Bottom,
				ui.Corners{TopRight: trackRadius, BottomRight: trackRadius}), secondaryPaint)
		} else {
			canvas.DrawRRect(ui.NewRRectFromLTRBAndCorners(sec.Dx, trackRect.Top, o.ThumbCenter.Dx, trackRect.Bottom,
				ui.Corners{TopLeft: trackRadius, BottomLeft: trackRadius}), secondaryPaint)
		}
	}
}

// GappedSliderTrackShape is the track of Material 3: two segments with a gap
// around the thumb, and a stop indicator at the end.
//
// Flutter: material/slider_parts.dart GappedSliderTrackShape
type GappedSliderTrackShape struct{}

// GetPreferredRect is the bounds of the track.
//
// Flutter: material/slider_parts.dart BaseSliderTrackShape.getPreferredRect
func (GappedSliderTrackShape) GetPreferredRect(o SliderTrackRectOpts) ui.Rect {
	return baseSliderTrackRect(o)
}

// IsRounded is true.
//
// Flutter: material/slider_parts.dart GappedSliderTrackShape.isRounded
func (GappedSliderTrackShape) IsRounded() bool { return true }

// Paint paints the track.
//
// Flutter: material/slider_parts.dart GappedSliderTrackShape.paint
func (s GappedSliderTrackShape) Paint(context *rendering.PaintingContext, offset ui.Offset, o SliderTrackPaintOpts) {
	t := o.SliderTheme
	if t.TrackHeight.IsNone() || t.TrackHeight.Value() <= 0 {
		return
	}
	activePaint, inactivePaint := trackPaints(o)
	leftTrackPaint, rightTrackPaint := activePaint, inactivePaint
	if o.TextDirection == ui.TextDirectionRTL {
		leftTrackPaint, rightTrackPaint = inactivePaint, activePaint
	}
	trackGap := t.TrackGap.Value()
	trackRect := s.GetPreferredRect(SliderTrackRectOpts{ParentBox: o.ParentBox, Offset: offset, SliderTheme: t, IsEnabled: o.IsEnabled, IsDiscrete: o.IsDiscrete})
	trackCornerRadius := ui.NewRadiusCircular(trackRect.ShortestSide() / 2)
	trackInsideCornerRadius := ui.NewRadiusCircular(2.0)
	trackHeight := t.TrackHeight.Value()
	canvas := context.Canvas()

	trackRRect := ui.NewRRectFromRectAndCorners(trackRect, ui.Corners{TopLeft: trackCornerRadius, BottomLeft: trackCornerRadius, TopRight: trackCornerRadius, BottomRight: trackCornerRadius})
	leftRRect := ui.NewRRectFromLTRBAndCorners(trackRect.Left, trackRect.Top, math.Max(trackRect.Left, o.ThumbCenter.Dx-trackGap), trackRect.Bottom,
		ui.Corners{TopLeft: trackCornerRadius, BottomLeft: trackCornerRadius, TopRight: trackInsideCornerRadius, BottomRight: trackInsideCornerRadius})
	rightRRect := ui.NewRRectFromLTRBAndCorners(o.ThumbCenter.Dx+trackGap, trackRect.Top, trackRect.Right, trackRect.Bottom,
		ui.Corners{TopRight: trackCornerRadius, BottomRight: trackCornerRadius, TopLeft: trackInsideCornerRadius, BottomLeft: trackInsideCornerRadius})

	canvas.Save()
	canvas.ClipRRect(trackRRect)
	drawLeftTrack := o.ThumbCenter.Dx > (leftRRect.Left + trackHeight/2)
	drawRightTrack := o.ThumbCenter.Dx < (rightRRect.Right - trackHeight/2)
	if drawLeftTrack {
		canvas.DrawRRect(leftRRect, leftTrackPaint)
	}
	if drawRightTrack {
		canvas.DrawRRect(rightRRect, rightTrackPaint)
	}

	isLTR := o.TextDirection == ui.TextDirectionLTR
	sec, hasSecondary := o.SecondaryOffset.Get()
	showSecondaryTrack := hasSecondary && ((isLTR && sec.Dx > o.ThumbCenter.Dx+trackGap) || (!isLTR && sec.Dx < o.ThumbCenter.Dx-trackGap))
	if showSecondaryTrack {
		secondaryPaint := paintWithColor(colorTweenEvaluate(t.DisabledSecondaryActiveTrackColor, t.SecondaryActiveTrackColor, o.EnableAnimation))
		corners := ui.Corners{TopLeft: trackInsideCornerRadius, BottomLeft: trackInsideCornerRadius, TopRight: trackCornerRadius, BottomRight: trackCornerRadius}
		if isLTR {
			canvas.DrawRRect(ui.NewRRectFromLTRBAndCorners(o.ThumbCenter.Dx+trackGap, trackRect.Top, sec.Dx, trackRect.Bottom, corners), secondaryPaint)
		} else {
			canvas.DrawRRect(ui.NewRRectFromLTRBAndCorners(sec.Dx-trackGap, trackRect.Top, o.ThumbCenter.Dx, trackRect.Bottom, corners), secondaryPaint)
		}
	}
	canvas.Restore()

	const stopIndicatorRadius = 2.0
	stopIndicatorTrailingSpace := trackHeight / 2
	var stopIndicatorOffset ui.Offset
	if isLTR {
		stopIndicatorOffset = ui.Offset{Dx: trackRect.CenterRight().Dx - stopIndicatorTrailingSpace, Dy: trackRect.Center().Dy}
	} else {
		stopIndicatorOffset = ui.Offset{Dx: trackRect.CenterLeft().Dx + stopIndicatorTrailingSpace, Dy: trackRect.Center().Dy}
	}
	var showStopIndicator bool
	if isLTR {
		showStopIndicator = o.ThumbCenter.Dx < stopIndicatorOffset.Dx
	} else {
		showStopIndicator = o.ThumbCenter.Dx > stopIndicatorOffset.Dx
	}
	if showStopIndicator && !o.IsDiscrete {
		canvas.DrawCircle(stopIndicatorOffset, stopIndicatorRadius, activePaint)
	}
}

// RoundSliderThumbShape is the thumb that is a circle with a shadow.
//
// Flutter: material/slider_parts.dart RoundSliderThumbShape
type RoundSliderThumbShape struct {
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

// NewRoundSliderThumbShape creates the shape.
//
// Flutter: material/slider_parts.dart RoundSliderThumbShape.new
func NewRoundSliderThumbShape(opts ...RoundSliderThumbShape) RoundSliderThumbShape {
	return firstOpts(opts)
}

func (s RoundSliderThumbShape) enabledRadius() float64 { return s.EnabledThumbRadius.OrElse(10.0) }
func (s RoundSliderThumbShape) disabledRadius() float64 {
	return s.DisabledThumbRadius.OrElse(s.enabledRadius())
}

// GetPreferredSize is the size of the circle.
//
// Flutter: material/slider_parts.dart RoundSliderThumbShape.getPreferredSize
func (s RoundSliderThumbShape) GetPreferredSize(isEnabled, _ bool) ui.Size {
	if isEnabled {
		return sizeFromRadius(s.enabledRadius())
	}
	return sizeFromRadius(s.disabledRadius())
}

// Paint paints the circle and its shadow.
//
// Flutter: material/slider_parts.dart RoundSliderThumbShape.paint
func (s RoundSliderThumbShape) Paint(context *rendering.PaintingContext, center ui.Offset, o SliderComponentPaintOpts) {
	canvas := context.Canvas()
	color := colorTweenEvaluate(o.SliderTheme.DisabledThumbColor, o.SliderTheme.ThumbColor, o.EnableAnimation)
	radius := animation.NewTween(s.disabledRadius(), s.enabledRadius()).Evaluate(o.EnableAnimation)
	evaluatedElevation := animation.NewTween(s.Elevation.OrElse(1.0), s.PressedElevation.OrElse(6.0)).Evaluate(o.ActivationAnimation)
	path := ui.NewPath()
	path.AddArc(ui.NewRectFromCenter(center, 2*radius, 2*radius), 0, math.Pi*2)
	canvas.DrawShadow(path, Colors.Black, evaluatedElevation, true)
	canvas.DrawCircle(center, radius, paintWithColor(color))
}

// HandleThumbShape is the thumb of Material 3: a vertical bar with rounded ends.
//
// Flutter: material/slider_parts.dart HandleThumbShape
type HandleThumbShape struct{}

// GetPreferredSize is 4 by 44.
//
// Flutter: material/slider_parts.dart HandleThumbShape.getPreferredSize
func (HandleThumbShape) GetPreferredSize(bool, bool) ui.Size {
	return ui.Size{Width: 4.0, Height: 44.0}
}

// Paint paints the bar, with the size that the theme gives.
//
// Flutter: material/slider_parts.dart HandleThumbShape.paint
func (HandleThumbShape) Paint(context *rendering.PaintingContext, center ui.Offset, o SliderComponentPaintOpts) {
	color := colorTweenEvaluate(o.SliderTheme.DisabledThumbColor, o.SliderTheme.ThumbColor, o.EnableAnimation)
	thumbSize := o.SliderTheme.ThumbSize.Resolve(0).Value()
	rrect := ui.NewRRectFromRectAndRadius(ui.NewRectFromCenter(center, thumbSize.Width, thumbSize.Height),
		ui.NewRadiusCircular(math.Min(math.Abs(thumbSize.Width), math.Abs(thumbSize.Height))/2))
	context.Canvas().DrawRRect(rrect, paintWithColor(color))
}

// horizontalShiftOf is the shift that keeps a value indicator with the rectangle
// width inside the view; the Rounded rect, the Drop and the Rectangular value
// indicators use it.
func horizontalShiftOf(parentBox rendering.Box, center ui.Offset, rectangleWidth, textScaleFactor float64, sizeWithOverflow ui.Size) float64 {
	const edgePadding = 8.0
	globalCenter := parentBox.LocalToGlobal(center, nil)
	overflowLeft := math.Max(0, rectangleWidth/2-globalCenter.Dx+edgePadding)
	overflowRight := math.Max(0, rectangleWidth/2-(sizeWithOverflow.Width-globalCenter.Dx-edgePadding))
	switch {
	case rectangleWidth < sizeWithOverflow.Width:
		return overflowLeft - overflowRight
	case overflowLeft-overflowRight > 0:
		return overflowLeft - (edgePadding * textScaleFactor)
	default:
		return -overflowRight + (edgePadding * textScaleFactor)
	}
}

// valueIndicatorPaintArgs are the arguments that the path painters of the value
// indicators share.
type valueIndicatorPaintArgs struct {
	parentBox            rendering.Box
	canvas               *ui.Canvas
	center               ui.Offset
	scale                float64
	labelPainter         *painting.TextPainter
	textScaleFactor      float64
	sizeWithOverflow     ui.Size
	backgroundPaintColor ui.Color
	strokePaintColor     foundation.Opt[ui.Color]
}

func strokePaintOf(c ui.Color) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(c)
	p.SetStrokeWidth(1.0)
	p.SetStyle(ui.PaintingStyleStroke)
	return p
}

// dropValueIndicatorPathPainter paints the drop-shaped value indicator.
//
// Flutter: material/slider_parts.dart _DropSliderValueIndicatorPathPainter
type dropValueIndicatorPathPainter struct{}

const (
	dropTriangleHeight   = 10.0
	dropLabelPadding     = 8.0
	dropPreferredHeight  = 32.0
	dropMinLabelWidth    = 20.0
	dropMinRectHeight    = 28.0
	dropRectYOffset      = 6.0
	dropBottomTipYOffset = 16.0
	dropUpperRectRadius  = 4.0
)

func (dropValueIndicatorPathPainter) getPreferredSize(labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	width := math.Max(dropMinLabelWidth, labelPainter.Width()) + dropLabelPadding*2*textScaleFactor
	return ui.Size{Width: width, Height: dropPreferredHeight * textScaleFactor}
}

func (dropValueIndicatorPathPainter) upperRectangleWidth(labelPainter *painting.TextPainter, scale float64) float64 {
	return (math.Max(dropMinLabelWidth, labelPainter.Width()) + dropLabelPadding) * scale
}

func (p dropValueIndicatorPathPainter) horizontalShift(a valueIndicatorPaintArgs) float64 {
	return horizontalShiftOf(a.parentBox, a.center, p.upperRectangleWidth(a.labelPainter, a.scale), a.textScaleFactor, a.sizeWithOverflow)
}

func (p dropValueIndicatorPathPainter) paint(a valueIndicatorPaintArgs) {
	if a.scale == 0.0 {
		return
	}
	rectangleWidth := p.upperRectangleWidth(a.labelPainter, a.scale)
	horizontalShift := p.horizontalShift(a)
	upperRect := ui.NewRectFromLTWH(-rectangleWidth/2+horizontalShift, -dropRectYOffset-dropMinRectHeight, rectangleWidth, dropMinRectHeight)
	canvas := a.canvas
	canvas.Save()
	canvas.Translate(a.center.Dx, a.center.Dy-dropBottomTipYOffset)
	canvas.Scale(a.scale, a.scale)

	// _adjustBorderRadius with rectness 0: the radius of the half of the shortest side.
	borderRect := painting.NewBorderRadiusAll(ui.NewRadiusCircular(upperRect.ShortestSide() / 2.0)).Resolve(a.labelPainter.TextDirection()).ToRRect(upperRect)
	trianglePath := ui.NewPath()
	trianglePath.LineTo(-dropTriangleHeight, -dropTriangleHeight)
	trianglePath.LineTo(dropTriangleHeight, -dropTriangleHeight)
	trianglePath.Close()
	trianglePath.AddRRect(borderRect)

	if c, ok := a.strokePaintColor.Get(); ok {
		canvas.DrawPath(trianglePath, strokePaintOf(c))
	}
	canvas.DrawPath(trianglePath, paintWithColor(a.backgroundPaintColor))

	bottomTipToUpperRectTranslateY := -(dropPreferredHeight/2)/2 - upperRect.Height()
	canvas.Translate(0, bottomTipToUpperRectTranslateY)
	boxCenter := ui.Offset{Dx: horizontalShift, Dy: upperRect.Height() / 1.75}
	halfLabelPainterOffset := ui.Offset{Dx: a.labelPainter.Width() / 2, Dy: a.labelPainter.Height() / 2}
	a.labelPainter.Paint(canvas, boxCenter.Sub(halfLabelPainterOffset))
	canvas.Restore()
}

// roundedRectValueIndicatorPathPainter paints the pill-shaped value indicator.
//
// Flutter: material/slider_parts.dart _RoundedRectSliderValueIndicatorPathPainter
type roundedRectValueIndicatorPathPainter struct{}

const (
	roundedLabelPadding     = 10.0
	roundedPreferredHeight  = 32.0
	roundedMinLabelWidth    = 16.0
	roundedRectYOffset      = 10.0
	roundedBottomTipYOffset = 16.0
)

func (roundedRectValueIndicatorPathPainter) getPreferredSize(labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	width := math.Max(roundedMinLabelWidth, labelPainter.Width()) + (roundedLabelPadding*2)*textScaleFactor
	return ui.Size{Width: width, Height: roundedPreferredHeight * textScaleFactor}
}

func (roundedRectValueIndicatorPathPainter) upperRectangleWidth(labelPainter *painting.TextPainter, scale float64) float64 {
	return (math.Max(roundedMinLabelWidth, labelPainter.Width()) + roundedLabelPadding*2) * scale
}

func (p roundedRectValueIndicatorPathPainter) horizontalShift(a valueIndicatorPaintArgs) float64 {
	return horizontalShiftOf(a.parentBox, a.center, p.upperRectangleWidth(a.labelPainter, a.scale), a.textScaleFactor, a.sizeWithOverflow)
}

func (p roundedRectValueIndicatorPathPainter) paint(a valueIndicatorPaintArgs) {
	if a.scale == 0.0 {
		return
	}
	rectangleWidth := p.upperRectangleWidth(a.labelPainter, a.scale)
	horizontalShift := p.horizontalShift(a)
	upperRect := ui.NewRectFromLTWH(-rectangleWidth/2+horizontalShift, -roundedRectYOffset-roundedPreferredHeight, rectangleWidth, roundedPreferredHeight)
	canvas := a.canvas
	canvas.Save()
	canvas.Translate(a.center.Dx, a.center.Dy-roundedBottomTipYOffset)
	canvas.Scale(a.scale, a.scale)
	rrect := ui.NewRRectFromRectAndRadius(upperRect, ui.NewRadiusCircular(upperRect.Height()/2))
	if c, ok := a.strokePaintColor.Get(); ok {
		canvas.DrawRRect(rrect, strokePaintOf(c))
	}
	canvas.DrawRRect(rrect, paintWithColor(a.backgroundPaintColor))
	bottomTipToUpperRectTranslateY := -(roundedPreferredHeight/2)/2 - upperRect.Height()
	canvas.Translate(0, bottomTipToUpperRectTranslateY)
	boxCenter := ui.Offset{Dx: horizontalShift, Dy: upperRect.Height() / 2.3}
	halfLabelPainterOffset := ui.Offset{Dx: a.labelPainter.Width() / 2, Dy: a.labelPainter.Height() / 2}
	a.labelPainter.Paint(canvas, boxCenter.Sub(halfLabelPainterOffset))
	canvas.Restore()
}

// DropSliderValueIndicatorShape is the value indicator that is a drop.
//
// Flutter: material/slider_parts.dart DropSliderValueIndicatorShape
type DropSliderValueIndicatorShape struct{}

// GetPreferredSize is the size for an empty label.
//
// Flutter: material/slider_parts.dart DropSliderValueIndicatorShape.getPreferredSize
func (DropSliderValueIndicatorShape) GetPreferredSize(bool, bool) ui.Size { return ui.Size{} }

// GetPreferredSizeForLabel is the size for the label.
//
// Flutter: material/slider_parts.dart DropSliderValueIndicatorShape.getPreferredSize
func (DropSliderValueIndicatorShape) GetPreferredSizeForLabel(_, _ bool, labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	return dropValueIndicatorPathPainter{}.getPreferredSize(labelPainter, textScaleFactor)
}

// Paint paints the drop with the label.
//
// Flutter: material/slider_parts.dart DropSliderValueIndicatorShape.paint
func (DropSliderValueIndicatorShape) Paint(context *rendering.PaintingContext, center ui.Offset, o SliderComponentPaintOpts) {
	dropValueIndicatorPathPainter{}.paint(valueIndicatorPaintArgs{
		parentBox: o.ParentBox, canvas: context.Canvas(), center: center, scale: o.ActivationAnimation.Value(),
		labelPainter: o.LabelPainter, textScaleFactor: o.TextScaleFactor, sizeWithOverflow: o.SizeWithOverflow,
		backgroundPaintColor: o.SliderTheme.ValueIndicatorColor.Value(), strokePaintColor: o.SliderTheme.ValueIndicatorStrokeColor})
}

// RoundedRectSliderValueIndicatorShape is the value indicator that is a pill.
//
// Flutter: material/slider_parts.dart RoundedRectSliderValueIndicatorShape
type RoundedRectSliderValueIndicatorShape struct{}

// GetPreferredSize is the size for an empty label.
//
// Flutter: material/slider_parts.dart RoundedRectSliderValueIndicatorShape.getPreferredSize
func (RoundedRectSliderValueIndicatorShape) GetPreferredSize(bool, bool) ui.Size { return ui.Size{} }

// GetPreferredSizeForLabel is the size for the label.
//
// Flutter: material/slider_parts.dart RoundedRectSliderValueIndicatorShape.getPreferredSize
func (RoundedRectSliderValueIndicatorShape) GetPreferredSizeForLabel(_, _ bool, labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	return roundedRectValueIndicatorPathPainter{}.getPreferredSize(labelPainter, textScaleFactor)
}

// Paint paints the pill with the label.
//
// Flutter: material/slider_parts.dart RoundedRectSliderValueIndicatorShape.paint
func (RoundedRectSliderValueIndicatorShape) Paint(context *rendering.PaintingContext, center ui.Offset, o SliderComponentPaintOpts) {
	roundedRectValueIndicatorPathPainter{}.paint(valueIndicatorPaintArgs{
		parentBox: o.ParentBox, canvas: context.Canvas(), center: center, scale: o.ActivationAnimation.Value(),
		labelPainter: o.LabelPainter, textScaleFactor: o.TextScaleFactor, sizeWithOverflow: o.SizeWithOverflow,
		backgroundPaintColor: o.SliderTheme.ValueIndicatorColor.Value(), strokePaintColor: o.SliderTheme.ValueIndicatorStrokeColor})
}
