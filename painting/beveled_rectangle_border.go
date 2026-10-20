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

// BeveledRectangleBorder is a rectangular border with flattened or "beveled"
// corners.
//
// Flutter: painting/beveled_rectangle_border.dart BeveledRectangleBorder
type BeveledRectangleBorder struct {
	ShapeBorderDefaults
	side BorderSide
	// BorderRadius is the radii for each corner; nil is BorderRadiusZero.
	BorderRadius BorderRadiusGeometry
}

// BeveledRectangleBorderOpts are the optional parameters of
// NewBeveledRectangleBorder.
//
// Flutter: painting/beveled_rectangle_border.dart BeveledRectangleBorder.new
type BeveledRectangleBorderOpts = RoundedRectangleBorderOpts

// NewBeveledRectangleBorder creates a beveled rectangle border.
//
// Flutter: painting/beveled_rectangle_border.dart BeveledRectangleBorder.new
func NewBeveledRectangleBorder(opts ...BeveledRectangleBorderOpts) BeveledRectangleBorder {
	o := firstOpts(opts)
	if o.BorderRadius == nil {
		o.BorderRadius = BorderRadiusZero
	}
	return BeveledRectangleBorder{side: o.Side.normalize(), BorderRadius: o.BorderRadius}
}

// Side is the border outline's color and weight.
//
// Flutter: painting/borders.dart OutlinedBorder.side
func (b BeveledRectangleBorder) Side() BorderSide { return b.side.normalize() }

func (b BeveledRectangleBorder) radius() BorderRadiusGeometry {
	if b.BorderRadius == nil {
		return BorderRadiusZero
	}
	return b.BorderRadius
}

// Dimensions is the stroke inset on all sides.
//
// Flutter: painting/borders.dart OutlinedBorder.dimensions
func (b BeveledRectangleBorder) Dimensions() EdgeInsetsGeometry {
	return OutlinedBorderDimensions(b.Side())
}

// Scale scales the side and the radii by t.
//
// Flutter: painting/beveled_rectangle_border.dart BeveledRectangleBorder.scale
func (b BeveledRectangleBorder) Scale(t float64) ShapeBorder {
	return BeveledRectangleBorder{side: b.Side().Scale(t), BorderRadius: BorderRadiusGeometryMul(b.radius(), t)}
}

// LerpFrom interpolates from a BeveledRectangleBorder.
//
// Flutter: painting/beveled_rectangle_border.dart BeveledRectangleBorder.lerpFrom
func (b BeveledRectangleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if r, ok := a.(BeveledRectangleBorder); ok {
		return BeveledRectangleBorder{
			side:         BorderSideLerp(r.Side(), b.Side(), t),
			BorderRadius: BorderRadiusGeometryLerp(r.radius(), b.radius(), t),
		}
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to a BeveledRectangleBorder.
//
// Flutter: painting/beveled_rectangle_border.dart BeveledRectangleBorder.lerpTo
func (b BeveledRectangleBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if r, ok := o.(BeveledRectangleBorder); ok {
		return BeveledRectangleBorder{
			side:         BorderSideLerp(b.Side(), r.Side(), t),
			BorderRadius: BorderRadiusGeometryLerp(b.radius(), r.radius(), t),
		}
	}
	return ShapeBorderDefaultLerpTo(b, o, t)
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: painting/beveled_rectangle_border.dart BeveledRectangleBorder.copyWith
func (b BeveledRectangleBorder) CopyWith(c RoundedRectangleBorderCopy) BeveledRectangleBorder {
	r := b.radius()
	if c.BorderRadius != nil {
		r = c.BorderRadius
	}
	return BeveledRectangleBorder{side: c.Side.OrElse(b.Side()), BorderRadius: r}
}

// CopyWithSide returns a copy with the given side.
//
// Flutter: painting/borders.dart OutlinedBorder.copyWith
func (b BeveledRectangleBorder) CopyWithSide(side BorderSide) OutlinedBorder {
	return b.CopyWith(RoundedRectangleBorderCopy{Side: foundation.Some(side)})
}

func beveledPath(rrect ui.RRect) *ui.Path {
	centerLeft := ui.Offset{Dx: rrect.Left, Dy: rrect.Center().Dy}
	centerRight := ui.Offset{Dx: rrect.Right, Dy: rrect.Center().Dy}
	centerTop := ui.Offset{Dx: rrect.Center().Dx, Dy: rrect.Top}
	centerBottom := ui.Offset{Dx: rrect.Center().Dx, Dy: rrect.Bottom}
	tlX, tlY := math.Max(0.0, rrect.TlRadiusX), math.Max(0.0, rrect.TlRadiusY)
	trX, trY := math.Max(0.0, rrect.TrRadiusX), math.Max(0.0, rrect.TrRadiusY)
	blX, blY := math.Max(0.0, rrect.BlRadiusX), math.Max(0.0, rrect.BlRadiusY)
	brX, brY := math.Max(0.0, rrect.BrRadiusX), math.Max(0.0, rrect.BrRadiusY)
	vertices := []ui.Offset{
		{Dx: rrect.Left, Dy: math.Min(centerLeft.Dy, rrect.Top+tlY)},
		{Dx: math.Min(centerTop.Dx, rrect.Left+tlX), Dy: rrect.Top},
		{Dx: math.Max(centerTop.Dx, rrect.Right-trX), Dy: rrect.Top},
		{Dx: rrect.Right, Dy: math.Min(centerRight.Dy, rrect.Top+trY)},
		{Dx: rrect.Right, Dy: math.Max(centerRight.Dy, rrect.Bottom-brY)},
		{Dx: math.Max(centerBottom.Dx, rrect.Right-brX), Dy: rrect.Bottom},
		{Dx: math.Min(centerBottom.Dx, rrect.Left+blX), Dy: rrect.Bottom},
		{Dx: rrect.Left, Dy: math.Max(centerLeft.Dy, rrect.Bottom-blY)},
	}
	p := ui.NewPath()
	p.AddPolygon(vertices, true)
	return p
}

// GetInnerPath is the beveled rectangle, deflated by the stroke inset.
//
// Flutter: painting/beveled_rectangle_border.dart BeveledRectangleBorder.getInnerPath
func (b BeveledRectangleBorder) GetInnerPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	return beveledPath(b.radius().Resolve(td).ToRRect(rect).Deflate(b.Side().StrokeInset()))
}

// GetOuterPath is the beveled rectangle.
//
// Flutter: painting/beveled_rectangle_border.dart BeveledRectangleBorder.getOuterPath
func (b BeveledRectangleBorder) GetOuterPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	return beveledPath(b.radius().Resolve(td).ToRRect(rect))
}

// Paint strokes the border.
//
// Flutter: painting/beveled_rectangle_border.dart BeveledRectangleBorder.paint
func (b BeveledRectangleBorder) Paint(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	if rect.IsEmpty() {
		return
	}
	side := b.Side()
	if side.Style != BorderStyleSolid {
		return
	}
	adjusted := b.radius().Resolve(td).ToRRect(rect).Inflate(side.StrokeOutset())
	path := beveledPath(adjusted)
	path.AddPath(b.GetInnerPath(rect, td), ui.Offset{})
	canvas.DrawPath(path, side.ToPaint())
}

// String describes the border.
//
// Flutter: painting/beveled_rectangle_border.dart BeveledRectangleBorder.toString
func (b BeveledRectangleBorder) String() string {
	return "BeveledRectangleBorder(" + b.Side().String() + ", " + b.radius().String() + ")"
}

var _ OutlinedBorder = BeveledRectangleBorder{}
