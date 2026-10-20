// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// CircleBorder is a border that fits a circle within the available space.
//
// Flutter: painting/circle_border.dart CircleBorder
type CircleBorder struct {
	ShapeBorderDefaults
	side BorderSide
	// Eccentricity defines the ovalness of the shape: 0.0 is a circle and 1.0
	// is a stadium-like oval.
	Eccentricity float64
}

// CircleBorderOpts are the optional parameters of NewCircleBorder.
//
// Flutter: painting/circle_border.dart CircleBorder.new
type CircleBorderOpts struct {
	// Side defaults to BorderSideNone.
	Side BorderSide
	// Eccentricity defaults to 0.0; it must be from 0.0 to 1.0.
	Eccentricity float64
}

func checkEccentricity(e float64) {
	if foundation.DebugMode && !(e >= 0.0 && e <= 1.0) {
		panic(errors.New("BorderEccentricityOutOfRange", "the eccentricity argument must be from 0.0 to 1.0", "eccentricity", e))
	}
}

// NewCircleBorder creates a border that fits a circle. In debug builds an
// eccentricity outside 0 to 1 panics with the tag BorderEccentricityOutOfRange.
//
// Flutter: painting/circle_border.dart CircleBorder.new
func NewCircleBorder(opts ...CircleBorderOpts) CircleBorder {
	o := firstOpts(opts)
	checkEccentricity(o.Eccentricity)
	return CircleBorder{side: o.Side.normalize(), Eccentricity: o.Eccentricity}
}

// Side is the border outline's color and weight.
//
// Flutter: painting/borders.dart OutlinedBorder.side
func (b CircleBorder) Side() BorderSide { return b.side.normalize() }

// Dimensions is the stroke inset on all sides.
//
// Flutter: painting/borders.dart OutlinedBorder.dimensions
func (b CircleBorder) Dimensions() EdgeInsetsGeometry { return OutlinedBorderDimensions(b.Side()) }

// Scale scales the side by t.
//
// Flutter: painting/circle_border.dart CircleBorder.scale
func (b CircleBorder) Scale(t float64) ShapeBorder {
	return CircleBorder{side: b.Side().Scale(t), Eccentricity: b.Eccentricity}
}

// asCircleBorder reports whether the border is a CircleBorder (an OvalBorder
// is one too) and returns it.
func asCircleBorder(b ShapeBorder) (CircleBorder, bool) {
	switch c := b.(type) {
	case CircleBorder:
		return c, true
	case OvalBorder:
		return c.CircleBorder, true
	}
	return CircleBorder{}, false
}

// LerpFrom interpolates from a CircleBorder, or scales from nil.
//
// Flutter: painting/circle_border.dart CircleBorder.lerpFrom
func (b CircleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if c, ok := asCircleBorder(a); ok {
		return CircleBorder{
			side:         BorderSideLerp(c.Side(), b.Side(), t),
			Eccentricity: ui.ClampDouble(ui.LerpDouble(c.Eccentricity, b.Eccentricity, t), 0.0, 1.0),
		}
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to a CircleBorder, or scales to nil.
//
// Flutter: painting/circle_border.dart CircleBorder.lerpTo
func (b CircleBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if c, ok := asCircleBorder(o); ok {
		return CircleBorder{
			side:         BorderSideLerp(b.Side(), c.Side(), t),
			Eccentricity: ui.ClampDouble(ui.LerpDouble(b.Eccentricity, c.Eccentricity, t), 0.0, 1.0),
		}
	}
	return ShapeBorderDefaultLerpTo(b, o, t)
}

// GetInnerPath is the circle, deflated by the stroke inset.
//
// Flutter: painting/circle_border.dart CircleBorder.getInnerPath
func (b CircleBorder) GetInnerPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddOval(b.adjustRect(rect).Deflate(b.Side().StrokeInset()))
	return p
}

// GetOuterPath is the circle.
//
// Flutter: painting/circle_border.dart CircleBorder.getOuterPath
func (b CircleBorder) GetOuterPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddOval(b.adjustRect(rect))
	return p
}

// PaintInterior fills the circle.
//
// Flutter: painting/circle_border.dart CircleBorder.paintInterior
func (b CircleBorder) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, _ ui.TextDirection) {
	if b.Eccentricity == 0.0 {
		canvas.DrawCircle(rect.Center(), rect.ShortestSide()/2.0, paint)
	} else {
		canvas.DrawOval(b.adjustRect(rect), paint)
	}
}

// PreferPaintInterior is true.
//
// Flutter: painting/circle_border.dart CircleBorder.preferPaintInterior
func (b CircleBorder) PreferPaintInterior() bool { return true }

// CircleBorderCopy holds the replacements for CircleBorder.CopyWith.
//
// Flutter: painting/circle_border.dart CircleBorder.copyWith
type CircleBorderCopy struct {
	Side         foundation.Opt[BorderSide]
	Eccentricity foundation.Opt[float64]
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: painting/circle_border.dart CircleBorder.copyWith
func (b CircleBorder) CopyWith(c CircleBorderCopy) CircleBorder {
	return CircleBorder{side: c.Side.OrElse(b.Side()), Eccentricity: c.Eccentricity.OrElse(b.Eccentricity)}
}

// CopyWithSide returns a copy with the given side.
//
// Flutter: painting/borders.dart OutlinedBorder.copyWith
func (b CircleBorder) CopyWithSide(side BorderSide) OutlinedBorder {
	return b.CopyWith(CircleBorderCopy{Side: foundation.Some(side)})
}

// Paint strokes the circle.
//
// Flutter: painting/circle_border.dart CircleBorder.paint
func (b CircleBorder) Paint(canvas *ui.Canvas, rect ui.Rect, _ ui.TextDirection) {
	side := b.Side()
	if side.Style != BorderStyleSolid {
		return
	}
	if b.Eccentricity == 0.0 {
		canvas.DrawCircle(rect.Center(), (rect.ShortestSide()+side.StrokeOffset())/2, side.ToPaint())
		return
	}
	canvas.DrawOval(b.adjustRect(rect).Inflate(side.StrokeOffset()/2), side.ToPaint())
}

func (b CircleBorder) adjustRect(rect ui.Rect) ui.Rect {
	if b.Eccentricity == 0.0 || rect.Width() == rect.Height() {
		return ui.NewRectFromCircle(rect.Center(), rect.ShortestSide()/2.0)
	}
	if rect.Width() < rect.Height() {
		delta := (1.0 - b.Eccentricity) * (rect.Height() - rect.Width()) / 2.0
		return ui.NewRectFromLTRB(rect.Left, rect.Top+delta, rect.Right, rect.Bottom-delta)
	}
	delta := (1.0 - b.Eccentricity) * (rect.Width() - rect.Height()) / 2.0
	return ui.NewRectFromLTRB(rect.Left+delta, rect.Top, rect.Right-delta, rect.Bottom)
}

// String describes the border.
//
// Flutter: painting/circle_border.dart CircleBorder.toString
func (b CircleBorder) String() string {
	if b.Eccentricity != 0.0 {
		return "CircleBorder(" + b.Side().String() + ", eccentricity: " + dartDouble(b.Eccentricity) + ")"
	}
	return "CircleBorder(" + b.Side().String() + ")"
}

// OvalBorder is a border that fits an oval within the available space: a
// CircleBorder with an eccentricity of 1.0 by default.
//
// Flutter: painting/oval_border.dart OvalBorder
type OvalBorder struct {
	CircleBorder
}

// NewOvalBorder creates an oval border. The eccentricity defaults to 1.0.
//
// Flutter: painting/oval_border.dart OvalBorder.new
func NewOvalBorder(opts ...OvalBorderOpts) OvalBorder {
	o := firstOpts(opts)
	e := o.Eccentricity.OrElse(1.0)
	checkEccentricity(e)
	return OvalBorder{CircleBorder{side: o.Side.normalize(), Eccentricity: e}}
}

// OvalBorderOpts are the optional parameters of NewOvalBorder.
//
// Flutter: painting/oval_border.dart OvalBorder.new
type OvalBorderOpts struct {
	// Side defaults to BorderSideNone.
	Side BorderSide
	// Eccentricity defaults to 1.0; it must be from 0.0 to 1.0.
	Eccentricity foundation.Opt[float64]
}

// Scale scales the side by t.
//
// Flutter: painting/oval_border.dart OvalBorder.scale
func (b OvalBorder) Scale(t float64) ShapeBorder {
	return OvalBorder{CircleBorder{side: b.Side().Scale(t), Eccentricity: b.Eccentricity}}
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: painting/oval_border.dart OvalBorder.copyWith
func (b OvalBorder) CopyWith(c CircleBorderCopy) OvalBorder {
	return OvalBorder{b.CircleBorder.CopyWith(c)}
}

// CopyWithSide returns a copy with the given side.
//
// Flutter: painting/borders.dart OutlinedBorder.copyWith
func (b OvalBorder) CopyWithSide(side BorderSide) OutlinedBorder {
	return b.CopyWith(CircleBorderCopy{Side: foundation.Some(side)})
}

// LerpFrom interpolates from an OvalBorder, or as a CircleBorder.
//
// Flutter: painting/oval_border.dart OvalBorder.lerpFrom
func (b OvalBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if o, ok := a.(OvalBorder); ok {
		return OvalBorder{CircleBorder{
			side:         BorderSideLerp(o.Side(), b.Side(), t),
			Eccentricity: ui.ClampDouble(ui.LerpDouble(o.Eccentricity, b.Eccentricity, t), 0.0, 1.0),
		}}
	}
	return b.CircleBorder.LerpFrom(a, t)
}

// LerpTo interpolates to an OvalBorder, or as a CircleBorder.
//
// Flutter: painting/oval_border.dart OvalBorder.lerpTo
func (b OvalBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if x, ok := o.(OvalBorder); ok {
		return OvalBorder{CircleBorder{
			side:         BorderSideLerp(b.Side(), x.Side(), t),
			Eccentricity: ui.ClampDouble(ui.LerpDouble(b.Eccentricity, x.Eccentricity, t), 0.0, 1.0),
		}}
	}
	return b.CircleBorder.LerpTo(o, t)
}

// String describes the border.
//
// Flutter: painting/oval_border.dart OvalBorder.toString
func (b OvalBorder) String() string {
	if b.Eccentricity != 1.0 {
		return "OvalBorder(" + b.Side().String() + ", eccentricity: " + dartDouble(b.Eccentricity) + ")"
	}
	return "OvalBorder(" + b.Side().String() + ")"
}

var (
	_ OutlinedBorder = CircleBorder{}
	_ OutlinedBorder = OvalBorder{}
)
