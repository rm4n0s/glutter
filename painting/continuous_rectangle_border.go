// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// ContinuousRectangleBorder is a rectangular border with smooth continuous
// transitions between the straight sides and the rounded corners.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder
type ContinuousRectangleBorder struct {
	ShapeBorderDefaults
	side BorderSide
	// BorderRadius is the radii for each corner; nil is BorderRadiusZero.
	BorderRadius BorderRadiusGeometry
}

// ContinuousRectangleBorderOpts are the optional parameters of
// NewContinuousRectangleBorder.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder.new
type ContinuousRectangleBorderOpts = RoundedRectangleBorderOpts

// NewContinuousRectangleBorder creates a continuous rectangle border.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder.new
func NewContinuousRectangleBorder(opts ...ContinuousRectangleBorderOpts) ContinuousRectangleBorder {
	o := firstOpts(opts)
	if o.BorderRadius == nil {
		o.BorderRadius = BorderRadiusZero
	}
	return ContinuousRectangleBorder{side: o.Side.normalize(), BorderRadius: o.BorderRadius}
}

// Side is the border outline's color and weight.
//
// Flutter: painting/borders.dart OutlinedBorder.side
func (b ContinuousRectangleBorder) Side() BorderSide { return b.side.normalize() }

func (b ContinuousRectangleBorder) radius() BorderRadiusGeometry {
	if b.BorderRadius == nil {
		return BorderRadiusZero
	}
	return b.BorderRadius
}

// Dimensions is the width of the side on all sides.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder.dimensions
func (b ContinuousRectangleBorder) Dimensions() EdgeInsetsGeometry {
	return NewEdgeInsetsAll(b.Side().Width)
}

// Scale scales the side and the radii by t.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder.scale
func (b ContinuousRectangleBorder) Scale(t float64) ShapeBorder {
	return ContinuousRectangleBorder{side: b.Side().Scale(t), BorderRadius: BorderRadiusGeometryMul(b.radius(), t)}
}

// LerpFrom interpolates from a ContinuousRectangleBorder.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder.lerpFrom
func (b ContinuousRectangleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if r, ok := a.(ContinuousRectangleBorder); ok {
		return ContinuousRectangleBorder{
			side:         BorderSideLerp(r.Side(), b.Side(), t),
			BorderRadius: BorderRadiusGeometryLerp(r.radius(), b.radius(), t),
		}
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to a ContinuousRectangleBorder.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder.lerpTo
func (b ContinuousRectangleBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if r, ok := o.(ContinuousRectangleBorder); ok {
		return ContinuousRectangleBorder{
			side:         BorderSideLerp(b.Side(), r.Side(), t),
			BorderRadius: BorderRadiusGeometryLerp(b.radius(), r.radius(), t),
		}
	}
	return ShapeBorderDefaultLerpTo(b, o, t)
}

func continuousPath(rrect ui.RRect) *ui.Path {
	clamp := func(v float64) float64 {
		if v > rrect.ShortestSide() {
			v = rrect.ShortestSide()
		}
		return math.Max(0.0, v)
	}
	left, right, top, bottom := rrect.Left, rrect.Right, rrect.Top, rrect.Bottom
	tlX, tlY := clamp(rrect.TlRadiusX), clamp(rrect.TlRadiusY)
	trX, trY := clamp(rrect.TrRadiusX), clamp(rrect.TrRadiusY)
	blX, blY := clamp(rrect.BlRadiusX), clamp(rrect.BlRadiusY)
	brX, brY := clamp(rrect.BrRadiusX), clamp(rrect.BrRadiusY)
	p := ui.NewPath()
	p.MoveTo(left, top+tlX)
	p.CubicTo(left, top, left, top, left+tlY, top)
	p.LineTo(right-trX, top)
	p.CubicTo(right, top, right, top, right, top+trY)
	p.LineTo(right, bottom-brX)
	p.CubicTo(right, bottom, right, bottom, right-brY, bottom)
	p.LineTo(left+blX, bottom)
	p.CubicTo(left, bottom, left, bottom, left, bottom-blY)
	p.Close()
	return p
}

// GetInnerPath is the shape, deflated by the width of the side.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder.getInnerPath
func (b ContinuousRectangleBorder) GetInnerPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	return continuousPath(b.radius().Resolve(td).ToRRect(rect).Deflate(b.Side().Width))
}

// GetOuterPath is the shape.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder.getOuterPath
func (b ContinuousRectangleBorder) GetOuterPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	return continuousPath(b.radius().Resolve(td).ToRRect(rect))
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder.copyWith
func (b ContinuousRectangleBorder) CopyWith(c RoundedRectangleBorderCopy) ContinuousRectangleBorder {
	r := b.radius()
	if c.BorderRadius != nil {
		r = c.BorderRadius
	}
	return ContinuousRectangleBorder{side: c.Side.OrElse(b.Side()), BorderRadius: r}
}

// CopyWithSide returns a copy with the given side.
//
// Flutter: painting/borders.dart OutlinedBorder.copyWith
func (b ContinuousRectangleBorder) CopyWithSide(side BorderSide) OutlinedBorder {
	return b.CopyWith(RoundedRectangleBorderCopy{Side: foundation.Some(side)})
}

// Paint strokes the border.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder.paint
func (b ContinuousRectangleBorder) Paint(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	if rect.IsEmpty() {
		return
	}
	if side := b.Side(); side.Style == BorderStyleSolid {
		canvas.DrawPath(b.GetOuterPath(rect, td), side.ToPaint())
	}
}

// String describes the border.
//
// Flutter: painting/continuous_rectangle_border.dart ContinuousRectangleBorder.toString
func (b ContinuousRectangleBorder) String() string {
	return "ContinuousRectangleBorder(" + b.Side().String() + ", " + b.radius().String() + ")"
}

var _ OutlinedBorder = ContinuousRectangleBorder{}
