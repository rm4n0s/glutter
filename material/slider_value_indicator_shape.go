// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// rectangularValueIndicatorPathPainter paints the rectangular value indicator
// with a notch at the bottom.
//
// Flutter: material/slider_value_indicator_shape.dart _RectangularSliderValueIndicatorPathPainter
type rectangularValueIndicatorPathPainter struct{}

const (
	rectangularTriangleHeight   = 8.0
	rectangularLabelPadding     = 16.0
	rectangularPreferredHeight  = 32.0
	rectangularMinLabelWidth    = 16.0
	rectangularBottomTipYOffset = 14.0
	rectangularUpperRectRadius  = 4.0
)

func (p rectangularValueIndicatorPathPainter) getPreferredSize(labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	return ui.Size{Width: p.upperRectangleWidth(labelPainter, 1, textScaleFactor), Height: labelPainter.Height() + rectangularLabelPadding}
}

func (rectangularValueIndicatorPathPainter) upperRectangleWidth(labelPainter *painting.TextPainter, scale, textScaleFactor float64) float64 {
	return (math.Max(rectangularMinLabelWidth*textScaleFactor, labelPainter.Width()) + rectangularLabelPadding*2) * scale
}

func (p rectangularValueIndicatorPathPainter) horizontalShift(a valueIndicatorPaintArgs) float64 {
	return horizontalShiftOf(a.parentBox, a.center, p.upperRectangleWidth(a.labelPainter, a.scale, a.textScaleFactor), a.textScaleFactor, a.sizeWithOverflow)
}

func (p rectangularValueIndicatorPathPainter) paint(a valueIndicatorPaintArgs) {
	if a.scale == 0.0 {
		return
	}
	rectangleWidth := p.upperRectangleWidth(a.labelPainter, a.scale, a.textScaleFactor)
	horizontalShift := p.horizontalShift(a)
	rectHeight := a.labelPainter.Height() + rectangularLabelPadding
	upperRect := ui.NewRectFromLTWH(-rectangleWidth/2+horizontalShift, -rectangularTriangleHeight-rectHeight, rectangleWidth, rectHeight)

	trianglePath := ui.NewPath()
	trianglePath.LineTo(-rectangularTriangleHeight, -rectangularTriangleHeight)
	trianglePath.LineTo(rectangularTriangleHeight, -rectangularTriangleHeight)
	trianglePath.Close()
	upperRRect := ui.NewRRectFromRectAndRadius(upperRect, ui.NewRadiusCircular(rectangularUpperRectRadius))
	trianglePath.AddRRect(upperRRect)

	canvas := a.canvas
	canvas.Save()
	canvas.Translate(a.center.Dx, a.center.Dy-rectangularBottomTipYOffset)
	canvas.Scale(a.scale, a.scale)
	if c, ok := a.strokePaintColor.Get(); ok {
		canvas.DrawPath(trianglePath, strokePaintOf(c))
	}
	canvas.DrawPath(trianglePath, paintWithColor(a.backgroundPaintColor))

	bottomTipToUpperRectTranslateY := -(rectangularPreferredHeight/2)/2 - upperRect.Height()
	canvas.Translate(0, bottomTipToUpperRectTranslateY)
	boxCenter := ui.Offset{Dx: horizontalShift, Dy: upperRect.Height() / 2}
	halfLabelPainterOffset := ui.Offset{Dx: a.labelPainter.Width() / 2, Dy: a.labelPainter.Height() / 2}
	a.labelPainter.Paint(canvas, boxCenter.Sub(halfLabelPainterOffset))
	canvas.Restore()
}

// RectangularSliderValueIndicatorShape is the value indicator that is a rectangle
// with a notch.
//
// Flutter: material/slider_value_indicator_shape.dart RectangularSliderValueIndicatorShape
type RectangularSliderValueIndicatorShape struct{}

// GetPreferredSize is the size for an empty label.
//
// Flutter: material/slider_value_indicator_shape.dart RectangularSliderValueIndicatorShape.getPreferredSize
func (RectangularSliderValueIndicatorShape) GetPreferredSize(bool, bool) ui.Size { return ui.Size{} }

// GetPreferredSizeForLabel is the size for the label.
//
// Flutter: material/slider_value_indicator_shape.dart RectangularSliderValueIndicatorShape.getPreferredSize
func (RectangularSliderValueIndicatorShape) GetPreferredSizeForLabel(_, _ bool, labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	return rectangularValueIndicatorPathPainter{}.getPreferredSize(labelPainter, textScaleFactor)
}

// Paint paints the rectangle with the label.
//
// Flutter: material/slider_value_indicator_shape.dart RectangularSliderValueIndicatorShape.paint
func (RectangularSliderValueIndicatorShape) Paint(context *rendering.PaintingContext, center ui.Offset, o SliderComponentPaintOpts) {
	rectangularValueIndicatorPathPainter{}.paint(valueIndicatorPaintArgs{
		parentBox: o.ParentBox, canvas: context.Canvas(), center: center, scale: o.ActivationAnimation.Value(),
		labelPainter: o.LabelPainter, textScaleFactor: o.TextScaleFactor, sizeWithOverflow: o.SizeWithOverflow,
		backgroundPaintColor: o.SliderTheme.ValueIndicatorColor.Value(), strokePaintColor: o.SliderTheme.ValueIndicatorStrokeColor})
}

// paddleValueIndicatorPathPainter paints the paddle-shaped value indicator.
//
// Flutter: material/slider_value_indicator_shape.dart _PaddleSliderValueIndicatorPathPainter
type paddleValueIndicatorPathPainter struct{}

const (
	paddleTopLobeRadius                  = 16.0
	paddleMinLabelWidth                  = 16.0
	paddleBottomLobeRadius               = 10.0
	paddleLabelPadding                   = 8.0
	paddleDistanceBetweenTopBottomCenter = 40.0
	paddleMiddleNeckWidth                = 3.0
	paddleBottomNeckRadius               = 4.5
	paddleTopNeckRadius                  = 13.0
	paddleNeckTriangleBase               = paddleTopNeckRadius + paddleMiddleNeckWidth/2
	paddleRightBottomNeckCenterX         = paddleMiddleNeckWidth/2 + paddleBottomNeckRadius
	paddleRightBottomNeckAngleStart      = math.Pi
	paddleNeckTriangleHypotenuse         = paddleTopLobeRadius + paddleTopNeckRadius
	paddleTwoSeventyDegrees              = 3.0 * math.Pi / 2.0
	paddleNinetyDegrees                  = math.Pi / 2.0
	paddleThirtyDegrees                  = math.Pi / 6.0
	paddlePreferredHeight                = paddleDistanceBetweenTopBottomCenter + paddleTopLobeRadius + paddleBottomLobeRadius
)

var paddleTopLobeCenter = ui.Offset{Dx: 0.0, Dy: -paddleDistanceBetweenTopBottomCenter}

func (paddleValueIndicatorPathPainter) getPreferredSize(labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	width := math.Max(paddleMinLabelWidth*textScaleFactor, labelPainter.Width()) + paddleLabelPadding*2*textScaleFactor
	return ui.Size{Width: width, Height: paddlePreferredHeight * textScaleFactor}
}

func paddleAddArc(path *ui.Path, center ui.Offset, radius, startAngle, endAngle float64) {
	arcRect := ui.NewRectFromCircle(center, radius)
	path.ArcTo(arcRect, startAngle, endAngle-startAngle, false)
}

func (p paddleValueIndicatorPathPainter) horizontalShift(center ui.Offset, labelPainter *painting.TextPainter, scale, textScaleFactor float64, sizeWithOverflow ui.Size) float64 {
	inverseTextScale := 0.0
	if textScaleFactor != 0 {
		inverseTextScale = 1.0 / textScaleFactor
	}
	labelHalfWidth := labelPainter.Width() / 2.0
	halfWidthNeeded := math.Max(0.0, inverseTextScale*labelHalfWidth-(paddleTopLobeRadius-paddleLabelPadding))
	shift := p.idealOffset(halfWidthNeeded, textScaleFactor*scale, center, sizeWithOverflow.Width)
	return shift * textScaleFactor
}

func (paddleValueIndicatorPathPainter) idealOffset(halfWidthNeeded, scale float64, center ui.Offset, widthWithOverflow float64) float64 {
	const edgeMargin = 8.0
	topLobeRect := ui.NewRectFromLTWH(-paddleTopLobeRadius-halfWidthNeeded, -paddleTopLobeRadius-paddleDistanceBetweenTopBottomCenter,
		2.0*(paddleTopLobeRadius+halfWidthNeeded), 2.0*paddleTopLobeRadius)
	topLeft := ui.Offset{Dx: topLobeRect.Left * scale, Dy: topLobeRect.Top * scale}.Add(center)
	bottomRight := ui.Offset{Dx: topLobeRect.Right * scale, Dy: topLobeRect.Bottom * scale}.Add(center)
	shift := 0.0
	if topLeft.Dx < edgeMargin {
		shift = edgeMargin - topLeft.Dx
	}
	endGlobal := widthWithOverflow
	if bottomRight.Dx > endGlobal-edgeMargin {
		shift = endGlobal - edgeMargin - bottomRight.Dx
	}
	if scale == 0.0 {
		shift = 0.0
	} else {
		shift /= scale
	}
	if shift < 0.0 {
		return math.Max(shift, -halfWidthNeeded)
	}
	return math.Min(shift, halfWidthNeeded)
}

func (p paddleValueIndicatorPathPainter) paint(canvas *ui.Canvas, center ui.Offset, paint *ui.Paint, scale float64,
	labelPainter *painting.TextPainter, textScaleFactor float64, sizeWithOverflow ui.Size, strokePaintColor foundation.Opt[ui.Color]) {
	if scale == 0.0 {
		return
	}
	overallScale := scale * textScaleFactor
	inverseTextScale := 0.0
	if textScaleFactor != 0 {
		inverseTextScale = 1.0 / textScaleFactor
	}
	labelHalfWidth := labelPainter.Width() / 2.0

	canvas.Save()
	canvas.Translate(center.Dx, center.Dy)
	canvas.Scale(overallScale, overallScale)

	bottomNeckTriangleHypotenuse := paddleBottomNeckRadius + paddleBottomLobeRadius/overallScale
	rightBottomNeckCenterY := -math.Sqrt(bottomNeckTriangleHypotenuse*bottomNeckTriangleHypotenuse - paddleRightBottomNeckCenterX*paddleRightBottomNeckCenterX)
	rightBottomNeckAngleEnd := math.Pi + math.Atan(rightBottomNeckCenterY/paddleRightBottomNeckCenterX)
	path := ui.NewPath()
	path.MoveTo(paddleMiddleNeckWidth/2, rightBottomNeckCenterY)
	paddleAddArc(path, ui.Offset{Dx: paddleRightBottomNeckCenterX, Dy: rightBottomNeckCenterY}, paddleBottomNeckRadius,
		paddleRightBottomNeckAngleStart, rightBottomNeckAngleEnd)
	paddleAddArc(path, ui.Offset{}, paddleBottomLobeRadius/overallScale, rightBottomNeckAngleEnd-math.Pi, 2*math.Pi-rightBottomNeckAngleEnd)
	paddleAddArc(path, ui.Offset{Dx: -paddleRightBottomNeckCenterX, Dy: rightBottomNeckCenterY}, paddleBottomNeckRadius,
		math.Pi-rightBottomNeckAngleEnd, 0)

	halfWidthNeeded := math.Max(0.0, inverseTextScale*labelHalfWidth-(paddleTopLobeRadius-paddleLabelPadding))
	shift := p.idealOffset(halfWidthNeeded, overallScale, center, sizeWithOverflow.Width)
	leftWidthNeeded := halfWidthNeeded - shift
	rightWidthNeeded := halfWidthNeeded + shift

	leftAmount := math.Max(0.0, math.Min(1.0, leftWidthNeeded/paddleNeckTriangleBase))
	rightAmount := math.Max(0.0, math.Min(1.0, rightWidthNeeded/paddleNeckTriangleBase))
	leftTheta := (1.0 - leftAmount) * paddleThirtyDegrees
	rightTheta := (1.0 - rightAmount) * paddleThirtyDegrees
	leftTopNeckCenter := ui.Offset{Dx: -paddleNeckTriangleBase, Dy: paddleTopLobeCenter.Dy + math.Cos(leftTheta)*paddleNeckTriangleHypotenuse}
	neckRightCenter := ui.Offset{Dx: paddleNeckTriangleBase, Dy: paddleTopLobeCenter.Dy + math.Cos(rightTheta)*paddleNeckTriangleHypotenuse}
	leftNeckArcAngle := paddleNinetyDegrees - leftTheta
	rightNeckArcAngle := math.Pi + paddleNinetyDegrees - rightTheta
	neckStretchBaseline := math.Max(0.0, rightBottomNeckCenterY-math.Max(leftTopNeckCenter.Dy, neckRightCenter.Dy))
	t := math.Pow(inverseTextScale, 3.0)
	stretch := ui.ClampDouble(neckStretchBaseline*t, 0.0, 10.0*neckStretchBaseline)
	neckStretch := ui.Offset{Dx: 0.0, Dy: neckStretchBaseline - stretch}

	paddleAddArc(path, leftTopNeckCenter.Add(neckStretch), paddleTopNeckRadius, 0.0, -leftNeckArcAngle)
	paddleAddArc(path, paddleTopLobeCenter.Sub(ui.Offset{Dx: leftWidthNeeded}).Add(neckStretch), paddleTopLobeRadius,
		paddleNinetyDegrees+leftTheta, paddleTwoSeventyDegrees)
	paddleAddArc(path, paddleTopLobeCenter.Add(ui.Offset{Dx: rightWidthNeeded}).Add(neckStretch), paddleTopLobeRadius,
		paddleTwoSeventyDegrees, paddleTwoSeventyDegrees+math.Pi-rightTheta)
	paddleAddArc(path, neckRightCenter.Add(neckStretch), paddleTopNeckRadius, rightNeckArcAngle, math.Pi)

	if c, ok := strokePaintColor.Get(); ok {
		canvas.DrawPath(path, strokePaintOf(c))
	}
	canvas.DrawPath(path, paint)

	canvas.Save()
	canvas.Translate(shift, -paddleDistanceBetweenTopBottomCenter+neckStretch.Dy)
	canvas.Scale(inverseTextScale, inverseTextScale)
	labelPainter.Paint(canvas, ui.Offset{Dx: -labelHalfWidth, Dy: -labelPainter.Height() / 2.0})
	canvas.Restore()
	canvas.Restore()
}

// PaddleSliderValueIndicatorShape is the value indicator that is a paddle.
//
// Flutter: material/slider_value_indicator_shape.dart PaddleSliderValueIndicatorShape
type PaddleSliderValueIndicatorShape struct{}

// GetPreferredSize is the size for an empty label.
//
// Flutter: material/slider_value_indicator_shape.dart PaddleSliderValueIndicatorShape.getPreferredSize
func (PaddleSliderValueIndicatorShape) GetPreferredSize(bool, bool) ui.Size { return ui.Size{} }

// GetPreferredSizeForLabel is the size for the label.
//
// Flutter: material/slider_value_indicator_shape.dart PaddleSliderValueIndicatorShape.getPreferredSize
func (PaddleSliderValueIndicatorShape) GetPreferredSizeForLabel(_, _ bool, labelPainter *painting.TextPainter, textScaleFactor float64) ui.Size {
	return paddleValueIndicatorPathPainter{}.getPreferredSize(labelPainter, textScaleFactor)
}

// Paint paints the paddle with the label.
//
// Flutter: material/slider_value_indicator_shape.dart PaddleSliderValueIndicatorShape.paint
func (PaddleSliderValueIndicatorShape) Paint(context *rendering.PaintingContext, center ui.Offset, o SliderComponentPaintOpts) {
	enableColor := colorTweenEvaluate(o.SliderTheme.DisabledThumbColor, o.SliderTheme.ValueIndicatorColor, o.EnableAnimation)
	paddleValueIndicatorPathPainter{}.paint(context.Canvas(), center, paintWithColor(enableColor), o.ActivationAnimation.Value(),
		o.LabelPainter, o.TextScaleFactor, o.SizeWithOverflow, o.SliderTheme.ValueIndicatorStrokeColor)
}
