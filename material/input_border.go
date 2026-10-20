// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// InputBorderGap is the gap that the floating label of an outlined text field
// leaves in the top edge of the border.
//
// Flutter: material/input_border.dart InputBorder.paint
type InputBorderGap struct {
	// Start is where the gap starts, in the coordinates of the rectangle; none
	// for no gap.
	Start foundation.Opt[float64]
	// Extent is the length of the gap, without the padding that the border adds
	// on each side.
	Extent float64
	// Percentage is how much of the gap is open, from 0 to 1.
	Percentage float64
}

// InputBorder is the border drawn around a text field: a line under it
// (UnderlineInputBorder), a box around it (OutlineInputBorder), or none
// (InputBorderNone). An outlined border can leave a gap in its top edge for the
// floating label.
//
// Flutter: material/input_border.dart InputBorder
type InputBorder interface {
	painting.ShapeBorder
	// BorderSide is the side of the border.
	BorderSide() painting.BorderSide
	// IsOutline says whether the border is an outline around the field.
	IsOutline() bool
	// CopyWithBorderSide returns the border with another side.
	CopyWithBorderSide(side painting.BorderSide) InputBorder
	// PaintGap paints the border within the rectangle, with a gap in the top
	// edge for the label.
	PaintGap(canvas *ui.Canvas, rect ui.Rect, gap InputBorderGap, textDirection ui.TextDirection)
}

// noInputBorder is the type of InputBorderNone.
type noInputBorder struct{ painting.ShapeBorderDefaults }

// InputBorderNone is the border that does not draw anything.
//
// Flutter: material/input_border.dart InputBorder.none
var InputBorderNone InputBorder = noInputBorder{}

func (noInputBorder) BorderSide() painting.BorderSide { return painting.BorderSideNone }
func (noInputBorder) IsOutline() bool                 { return false }
func (b noInputBorder) CopyWithBorderSide(painting.BorderSide) InputBorder {
	return b
}
func (noInputBorder) Dimensions() painting.EdgeInsetsGeometry { return painting.EdgeInsetsZero }
func (b noInputBorder) Scale(float64) painting.ShapeBorder    { return b }
func (b noInputBorder) LerpFrom(a painting.ShapeBorder, t float64) painting.ShapeBorder {
	return painting.ShapeBorderDefaultLerpFrom(b, a, t)
}
func (b noInputBorder) LerpTo(o painting.ShapeBorder, t float64) painting.ShapeBorder {
	return painting.ShapeBorderDefaultLerpTo(b, o, t)
}
func (noInputBorder) GetInnerPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRect(rect)
	return p
}
func (noInputBorder) GetOuterPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRect(rect)
	return p
}
func (noInputBorder) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, _ ui.TextDirection) {
	canvas.DrawRect(rect, paint)
}
func (noInputBorder) PreferPaintInterior() bool                                      { return true }
func (noInputBorder) Paint(*ui.Canvas, ui.Rect, ui.TextDirection)                    {}
func (noInputBorder) PaintGap(*ui.Canvas, ui.Rect, InputBorderGap, ui.TextDirection) {}
func (noInputBorder) String() string                                                 { return "InputBorder.none" }

// inputBorderSide is the side of a border that was given a zero side: the default
// BorderSide().
func inputBorderSide(side painting.BorderSide) painting.BorderSide {
	if side == (painting.BorderSide{}) {
		return painting.NewBorderSide()
	}
	return side
}

// UnderlineInputBorder draws a horizontal line at the bottom of an
// InputDecorator's container and defines the container's shape.
//
// Flutter: material/input_border.dart UnderlineInputBorder
type UnderlineInputBorder struct {
	painting.ShapeBorderDefaults
	borderSide   painting.BorderSide
	borderRadius painting.BorderRadius
}

// UnderlineInputBorderOpts are the optional parameters of
// NewUnderlineInputBorder.
//
// Flutter: material/input_border.dart UnderlineInputBorder.new
type UnderlineInputBorderOpts struct {
	// BorderSide defaults to a one pixel black side.
	BorderSide painting.BorderSide
	// BorderRadius defaults to a radius of 4 on the top corners.
	BorderRadius foundation.Opt[painting.BorderRadius]
}

// NewUnderlineInputBorder creates a border that is a line under the field.
//
// Flutter: material/input_border.dart UnderlineInputBorder.new
func NewUnderlineInputBorder(opts ...UnderlineInputBorderOpts) UnderlineInputBorder {
	o := firstOpts(opts)
	return UnderlineInputBorder{
		borderSide: inputBorderSide(o.BorderSide),
		borderRadius: o.BorderRadius.OrElse(painting.BorderRadius{
			TopLeft: ui.NewRadiusCircular(4.0), TopRight: ui.NewRadiusCircular(4.0),
		}),
	}
}

// BorderSide is the side of the border.
//
// Flutter: material/input_border.dart InputBorder.borderSide
func (b UnderlineInputBorder) BorderSide() painting.BorderSide { return inputBorderSide(b.borderSide) }

// BorderRadius is the radii of the corners of the container.
//
// Flutter: material/input_border.dart UnderlineInputBorder.borderRadius
func (b UnderlineInputBorder) BorderRadius() painting.BorderRadius { return b.borderRadius }

// IsOutline is false.
//
// Flutter: material/input_border.dart UnderlineInputBorder.isOutline
func (UnderlineInputBorder) IsOutline() bool { return false }

// UnderlineInputBorderCopy holds the replacements for
// UnderlineInputBorder.CopyWith.
//
// Flutter: material/input_border.dart UnderlineInputBorder.copyWith
type UnderlineInputBorderCopy struct {
	BorderSide   foundation.Opt[painting.BorderSide]
	BorderRadius foundation.Opt[painting.BorderRadius]
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: material/input_border.dart UnderlineInputBorder.copyWith
func (b UnderlineInputBorder) CopyWith(c UnderlineInputBorderCopy) UnderlineInputBorder {
	return UnderlineInputBorder{
		borderSide:   c.BorderSide.OrElse(b.BorderSide()),
		borderRadius: c.BorderRadius.OrElse(b.borderRadius),
	}
}

// CopyWithBorderSide returns the border with another side.
//
// Flutter: material/input_border.dart InputBorder.copyWith
func (b UnderlineInputBorder) CopyWithBorderSide(side painting.BorderSide) InputBorder {
	return UnderlineInputBorder{borderSide: inputBorderSide(side), borderRadius: b.borderRadius}
}

// Dimensions is the width of the side, at the bottom.
//
// Flutter: material/input_border.dart UnderlineInputBorder.dimensions
func (b UnderlineInputBorder) Dimensions() painting.EdgeInsetsGeometry {
	return painting.EdgeInsets{Bottom: b.BorderSide().Width}
}

// Scale scales the side by t.
//
// Flutter: material/input_border.dart UnderlineInputBorder.scale
func (b UnderlineInputBorder) Scale(t float64) painting.ShapeBorder {
	return NewUnderlineInputBorder(UnderlineInputBorderOpts{BorderSide: b.BorderSide().Scale(t)})
}

// GetInnerPath is the rectangle without the width of the side.
//
// Flutter: material/input_border.dart UnderlineInputBorder.getInnerPath
func (b UnderlineInputBorder) GetInnerPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRect(ui.NewRectFromLTWH(rect.Left, rect.Top, rect.Width(), math.Max(0.0, rect.Height()-b.BorderSide().Width)))
	return p
}

// GetOuterPath is the rounded rectangle of the container.
//
// Flutter: material/input_border.dart UnderlineInputBorder.getOuterPath
func (b UnderlineInputBorder) GetOuterPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.borderRadius.Resolve(textDirection).ToRRect(rect))
	return p
}

// PaintInterior fills the rounded rectangle of the container.
//
// Flutter: material/input_border.dart UnderlineInputBorder.paintInterior
func (b UnderlineInputBorder) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection ui.TextDirection) {
	canvas.DrawRRect(b.borderRadius.Resolve(textDirection).ToRRect(rect), paint)
}

// PreferPaintInterior is true.
//
// Flutter: material/input_border.dart UnderlineInputBorder.preferPaintInterior
func (UnderlineInputBorder) PreferPaintInterior() bool { return true }

// LerpFrom interpolates from another UnderlineInputBorder.
//
// Flutter: material/input_border.dart UnderlineInputBorder.lerpFrom
func (b UnderlineInputBorder) LerpFrom(a painting.ShapeBorder, t float64) painting.ShapeBorder {
	if u, ok := a.(UnderlineInputBorder); ok {
		return UnderlineInputBorder{
			borderSide:   painting.BorderSideLerp(u.BorderSide(), b.BorderSide(), t),
			borderRadius: *painting.BorderRadiusLerp(&u.borderRadius, &b.borderRadius, t),
		}
	}
	return painting.ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to another UnderlineInputBorder.
//
// Flutter: material/input_border.dart UnderlineInputBorder.lerpTo
func (b UnderlineInputBorder) LerpTo(o painting.ShapeBorder, t float64) painting.ShapeBorder {
	if u, ok := o.(UnderlineInputBorder); ok {
		return UnderlineInputBorder{
			borderSide:   painting.BorderSideLerp(b.BorderSide(), u.BorderSide(), t),
			borderRadius: *painting.BorderRadiusLerp(&b.borderRadius, &u.borderRadius, t),
		}
	}
	return painting.ShapeBorderDefaultLerpTo(b, o, t)
}

// Paint paints the line under the field.
//
// Flutter: material/input_border.dart UnderlineInputBorder.paint
func (b UnderlineInputBorder) Paint(canvas *ui.Canvas, rect ui.Rect, textDirection ui.TextDirection) {
	b.PaintGap(canvas, rect, InputBorderGap{}, textDirection)
}

// PaintGap paints the line under the field; the underline has no gap.
//
// Flutter: material/input_border.dart UnderlineInputBorder.paint
func (b UnderlineInputBorder) PaintGap(canvas *ui.Canvas, rect ui.Rect, _ InputBorderGap, textDirection ui.TextDirection) {
	side := b.BorderSide()
	if side.Style == painting.BorderStyleNone {
		return
	}
	if b.borderRadius.BottomLeft != ui.RadiusZero || b.borderRadius.BottomRight != ui.RadiusZero {
		// This prevents the border from leaking the color due to anti-aliasing
		// rounding errors.
		maximum := ui.NewRadiusCircular(rect.Height() / 2)
		updated := painting.BorderRadius{
			BottomLeft:  b.borderRadius.BottomLeft.Clamp(ui.RadiusClampOpts{Maximum: &maximum}),
			BottomRight: b.borderRadius.BottomRight.Clamp(ui.RadiusClampOpts{Maximum: &maximum}),
		}
		painting.PaintNonUniformBorder(canvas, rect, painting.NonUniformBorderOpts{
			BorderRadius:  foundation.Some[painting.BorderRadiusGeometry](updated),
			TextDirection: textDirection,
			Bottom:        side.CopyWith(painting.BorderSideCopy{StrokeAlign: foundation.Some(painting.StrokeAlignInside)}),
			Color:         side.Color,
		})
		return
	}
	alignInsideOffset := ui.Offset{Dx: 0, Dy: side.Width / 2}
	canvas.DrawLine(rect.BottomLeft().Sub(alignInsideOffset), rect.BottomRight().Sub(alignInsideOffset), side.ToPaint())
}

// String describes the border.
func (b UnderlineInputBorder) String() string {
	return "UnderlineInputBorder(" + b.BorderSide().String() + ", " + b.borderRadius.String() + ")"
}

// OutlineInputBorder draws a rounded rectangle around an InputDecorator's
// container, with a gap in the top edge for the floating label.
//
// Flutter: material/input_border.dart OutlineInputBorder
type OutlineInputBorder struct {
	painting.ShapeBorderDefaults
	borderSide   painting.BorderSide
	borderRadius painting.BorderRadius
	gapPadding   float64
}

// OutlineInputBorderOpts are the optional parameters of NewOutlineInputBorder.
//
// Flutter: material/input_border.dart OutlineInputBorder.new
type OutlineInputBorderOpts struct {
	// BorderSide defaults to a one pixel black side.
	BorderSide painting.BorderSide
	// BorderRadius defaults to a radius of 4 on all the corners; the corners must
	// be circular.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// GapPadding is the horizontal padding on both sides of the gap; the default
	// is 4.
	GapPadding foundation.Opt[float64]
}

// NewOutlineInputBorder creates a border that is a rounded rectangle around the
// field. A negative gap padding panics with the tag
// OutlineInputBorderGapPadding in debug builds.
//
// Flutter: material/input_border.dart OutlineInputBorder.new
func NewOutlineInputBorder(opts ...OutlineInputBorderOpts) OutlineInputBorder {
	o := firstOpts(opts)
	gap := o.GapPadding.OrElse(4.0)
	if foundation.DebugMode && !(gap >= 0.0) {
		panic(errors.New("OutlineInputBorderGapPadding", "the gap padding of an OutlineInputBorder must not be negative", "gapPadding", gap))
	}
	return OutlineInputBorder{
		borderSide:   inputBorderSide(o.BorderSide),
		borderRadius: o.BorderRadius.OrElse(painting.NewBorderRadiusAll(ui.NewRadiusCircular(4.0))),
		gapPadding:   gap,
	}
}

// outlineCornersAreCircular says whether every corner has the same x and y.
func outlineCornersAreCircular(b painting.BorderRadius) bool {
	return b.TopLeft.X == b.TopLeft.Y && b.BottomLeft.X == b.BottomLeft.Y &&
		b.TopRight.X == b.TopRight.Y && b.BottomRight.X == b.BottomRight.Y
}

// BorderSide is the side of the border.
//
// Flutter: material/input_border.dart InputBorder.borderSide
func (b OutlineInputBorder) BorderSide() painting.BorderSide { return inputBorderSide(b.borderSide) }

// BorderRadius is the radii of the corners of the outline.
//
// Flutter: material/input_border.dart OutlineInputBorder.borderRadius
func (b OutlineInputBorder) BorderRadius() painting.BorderRadius { return b.borderRadius }

// GapPadding is the horizontal padding on each side of the gap.
//
// Flutter: material/input_border.dart OutlineInputBorder.gapPadding
func (b OutlineInputBorder) GapPadding() float64 { return b.gapPadding }

// IsOutline is true.
//
// Flutter: material/input_border.dart OutlineInputBorder.isOutline
func (OutlineInputBorder) IsOutline() bool { return true }

// OutlineInputBorderCopy holds the replacements for OutlineInputBorder.CopyWith.
//
// Flutter: material/input_border.dart OutlineInputBorder.copyWith
type OutlineInputBorderCopy struct {
	BorderSide   foundation.Opt[painting.BorderSide]
	BorderRadius foundation.Opt[painting.BorderRadius]
	GapPadding   foundation.Opt[float64]
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: material/input_border.dart OutlineInputBorder.copyWith
func (b OutlineInputBorder) CopyWith(c OutlineInputBorderCopy) OutlineInputBorder {
	return NewOutlineInputBorder(OutlineInputBorderOpts{
		BorderSide:   c.BorderSide.OrElse(b.BorderSide()),
		BorderRadius: foundation.Some(c.BorderRadius.OrElse(b.borderRadius)),
		GapPadding:   foundation.Some(c.GapPadding.OrElse(b.gapPadding)),
	})
}

// CopyWithBorderSide returns the border with another side.
//
// Flutter: material/input_border.dart InputBorder.copyWith
func (b OutlineInputBorder) CopyWithBorderSide(side painting.BorderSide) InputBorder {
	return OutlineInputBorder{borderSide: inputBorderSide(side), borderRadius: b.borderRadius, gapPadding: b.gapPadding}
}

// Dimensions is the stroke inset on every side.
//
// Flutter: material/input_border.dart OutlineInputBorder.dimensions
func (b OutlineInputBorder) Dimensions() painting.EdgeInsetsGeometry {
	return painting.NewEdgeInsetsAll(b.BorderSide().StrokeInset())
}

// Scale scales the side, the radii and the gap padding by t.
//
// Flutter: material/input_border.dart OutlineInputBorder.scale
func (b OutlineInputBorder) Scale(t float64) painting.ShapeBorder {
	return OutlineInputBorder{borderSide: b.BorderSide().Scale(t), borderRadius: b.borderRadius.Mul(t), gapPadding: b.gapPadding * t}
}

// LerpFrom interpolates from another OutlineInputBorder.
//
// Flutter: material/input_border.dart OutlineInputBorder.lerpFrom
func (b OutlineInputBorder) LerpFrom(a painting.ShapeBorder, t float64) painting.ShapeBorder {
	if o, ok := a.(OutlineInputBorder); ok {
		return OutlineInputBorder{
			borderRadius: *painting.BorderRadiusLerp(&o.borderRadius, &b.borderRadius, t),
			borderSide:   painting.BorderSideLerp(o.BorderSide(), b.BorderSide(), t),
			gapPadding:   o.gapPadding,
		}
	}
	return painting.ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to another OutlineInputBorder.
//
// Flutter: material/input_border.dart OutlineInputBorder.lerpTo
func (b OutlineInputBorder) LerpTo(other painting.ShapeBorder, t float64) painting.ShapeBorder {
	if o, ok := other.(OutlineInputBorder); ok {
		return OutlineInputBorder{
			borderRadius: *painting.BorderRadiusLerp(&b.borderRadius, &o.borderRadius, t),
			borderSide:   painting.BorderSideLerp(b.BorderSide(), o.BorderSide(), t),
			gapPadding:   o.gapPadding,
		}
	}
	return painting.ShapeBorderDefaultLerpTo(b, other, t)
}

// GetInnerPath is the rounded rectangle without the stroke inset.
//
// Flutter: material/input_border.dart OutlineInputBorder.getInnerPath
func (b OutlineInputBorder) GetInnerPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.borderRadius.Resolve(textDirection).ToRRect(rect).Deflate(b.BorderSide().StrokeInset()))
	return p
}

// GetOuterPath is the rounded rectangle.
//
// Flutter: material/input_border.dart OutlineInputBorder.getOuterPath
func (b OutlineInputBorder) GetOuterPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.borderRadius.Resolve(textDirection).ToRRect(rect))
	return p
}

// PaintInterior fills the rounded rectangle.
//
// Flutter: material/input_border.dart OutlineInputBorder.paintInterior
func (b OutlineInputBorder) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection ui.TextDirection) {
	canvas.DrawRRect(b.borderRadius.Resolve(textDirection).ToRRect(rect), paint)
}

// PreferPaintInterior is true.
//
// Flutter: material/input_border.dart OutlineInputBorder.preferPaintInterior
func (OutlineInputBorder) PreferPaintInterior() bool { return true }

// gapBorderPath is the border with a gap in the top edge, from the gap start to
// the gap end.
//
// Flutter: material/input_border.dart OutlineInputBorder._gapBorderPath
func (b OutlineInputBorder) gapBorderPath(center ui.RRect, outerWidth, start, extent float64) *ui.Path {
	// When the corner radii on any side add up to be greater than the given
	// height, each radius has to be scaled to not exceed the size of the
	// width/height of the RRect.
	scaled := center.ScaleRadii()
	tlCorner := ui.NewRectFromLTWH(scaled.Left, scaled.Top, scaled.TlRadiusX*2.0, scaled.TlRadiusY*2.0)
	trCorner := ui.NewRectFromLTWH(scaled.Right-scaled.TrRadiusX*2.0, scaled.Top, scaled.TrRadiusX*2.0, scaled.TrRadiusY*2.0)
	brCorner := ui.NewRectFromLTWH(scaled.Right-scaled.BrRadiusX*2.0, scaled.Bottom-scaled.BrRadiusY*2.0, scaled.BrRadiusX*2.0, scaled.BrRadiusY*2.0)
	blCorner := ui.NewRectFromLTWH(scaled.Left, scaled.Bottom-scaled.BlRadiusY*2.0, scaled.BlRadiusX*2.0, scaled.BlRadiusY*2.0)

	// This assumes that the radius is circular (x and y radius are equal).
	// Currently, BorderRadius only supports circular radii.
	const cornerArcSweep = math.Pi / 2.0
	path := ui.NewPath()

	// Top left corner.
	if scaled.TlRadius() != ui.RadiusZero {
		tlCornerArcSweep := math.Acos(ui.ClampDouble(1-start/scaled.TlRadiusX, 0.0, 1.0))
		path.AddArc(tlCorner, math.Pi, tlCornerArcSweep)
	} else {
		// Because the path is painted with Paint.strokeCap = StrokeCap.butt,
		// horizontal coordinate is moved based on strokeOffset to respect
		// strokeAlign.
		path.MoveTo(scaled.Left+b.BorderSide().StrokeOffset()/2, scaled.Top)
	}

	// Draw top border from top left corner to gap start.
	if start > scaled.TlRadiusX {
		path.LineTo(start, scaled.Top)
	}

	// Draw top border from gap end to top right corner and draw top right corner.
	const trCornerArcStart = (3 * math.Pi) / 2.0
	const trCornerArcSweep = cornerArcSweep
	if start+extent < outerWidth-scaled.TrRadiusX {
		path.MoveTo(start+extent, scaled.Top)
		path.LineTo(scaled.Right-scaled.TrRadiusX, scaled.Top)
		if scaled.TrRadius() != ui.RadiusZero {
			path.AddArc(trCorner, trCornerArcStart, trCornerArcSweep)
		}
	} else if start+extent < outerWidth {
		dx := outerWidth - (start + extent)
		sweep := math.Asin(ui.ClampDouble(1-dx/scaled.TrRadiusX, 0.0, 1.0))
		path.AddArc(trCorner, trCornerArcStart+sweep, trCornerArcSweep-sweep)
	}

	// Draw right border and bottom right corner.
	if scaled.BrRadius() != ui.RadiusZero {
		path.MoveTo(scaled.Right, scaled.Top+scaled.TrRadiusY)
	}
	path.LineTo(scaled.Right, scaled.Bottom-scaled.BrRadiusY)
	if scaled.BrRadius() != ui.RadiusZero {
		path.AddArc(brCorner, 0.0, cornerArcSweep)
	}

	// Draw bottom border and bottom left corner.
	path.LineTo(scaled.Left+scaled.BlRadiusX, scaled.Bottom)
	if scaled.BlRadius() != ui.RadiusZero {
		path.AddArc(blCorner, math.Pi/2.0, cornerArcSweep)
	}

	// Draw left border.
	path.LineTo(scaled.Left, scaled.Top+scaled.TlRadiusY)
	return path
}

// Paint paints the outline, without a gap.
//
// Flutter: material/input_border.dart OutlineInputBorder.paint
func (b OutlineInputBorder) Paint(canvas *ui.Canvas, rect ui.Rect, textDirection ui.TextDirection) {
	b.PaintGap(canvas, rect, InputBorderGap{}, textDirection)
}

// PaintGap paints the outline with a gap in the top edge. A percentage out of
// the range 0 to 1 panics with the tag InputBorderGapPercentage, and a corner
// that is not circular with the tag OutlineInputBorderCorners, in debug builds.
//
// Flutter: material/input_border.dart OutlineInputBorder.paint
func (b OutlineInputBorder) PaintGap(canvas *ui.Canvas, rect ui.Rect, gap InputBorderGap, textDirection ui.TextDirection) {
	if foundation.DebugMode {
		if !(gap.Percentage >= 0.0 && gap.Percentage <= 1.0) {
			panic(errors.New("InputBorderGapPercentage", "the gap percentage of an input border must be between 0 and 1", "gapPercentage", gap.Percentage))
		}
		if !outlineCornersAreCircular(b.borderRadius) {
			panic(errors.New("OutlineInputBorderCorners", "the corners of an OutlineInputBorder must be circular"))
		}
	}
	side := b.BorderSide()
	paint := side.ToPaint()
	outer := b.borderRadius.ToRRect(rect)
	center := outer.Inflate(side.StrokeOffset() / 2)
	gapStart, hasGap := gap.Start.Get()
	if !hasGap || gap.Extent <= 0.0 || gap.Percentage == 0.0 {
		canvas.DrawRRect(center, paint)
		return
	}
	extent := ui.LerpDouble(0.0, gap.Extent+b.gapPadding*2.0, gap.Percentage)
	var start float64
	if textDirection == ui.TextDirectionRTL {
		start = gapStart + b.gapPadding - extent
	} else {
		start = gapStart - b.gapPadding
	}
	canvas.DrawPath(b.gapBorderPath(center, outer.Width(), math.Max(0.0, start), extent), paint)
}

// String describes the border.
func (b OutlineInputBorder) String() string {
	return "OutlineInputBorder(" + b.BorderSide().String() + ", " + b.borderRadius.String() + ")"
}
