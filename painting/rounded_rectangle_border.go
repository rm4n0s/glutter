// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// RoundedRectangleBorder is a rectangular border with rounded corners.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder
type RoundedRectangleBorder struct {
	ShapeBorderDefaults
	side BorderSide
	// BorderRadius is the radii for each corner; nil is BorderRadiusZero.
	BorderRadius BorderRadiusGeometry
}

// RoundedRectangleBorderOpts are the optional parameters of
// NewRoundedRectangleBorder.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.new
type RoundedRectangleBorderOpts struct {
	// Side defaults to BorderSideNone.
	Side BorderSide
	// BorderRadius defaults to BorderRadiusZero.
	BorderRadius BorderRadiusGeometry
}

// NewRoundedRectangleBorder creates a rounded rectangle border.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.new
func NewRoundedRectangleBorder(opts ...RoundedRectangleBorderOpts) RoundedRectangleBorder {
	o := firstOpts(opts)
	if o.BorderRadius == nil {
		o.BorderRadius = BorderRadiusZero
	}
	return RoundedRectangleBorder{side: o.Side.normalize(), BorderRadius: o.BorderRadius}
}

// Side is the border outline's color and weight.
//
// Flutter: painting/borders.dart OutlinedBorder.side
func (b RoundedRectangleBorder) Side() BorderSide { return b.side.normalize() }

func (b RoundedRectangleBorder) radius() BorderRadiusGeometry {
	if b.BorderRadius == nil {
		return BorderRadiusZero
	}
	return b.BorderRadius
}

// Dimensions is the stroke inset on all sides.
//
// Flutter: painting/borders.dart OutlinedBorder.dimensions
func (b RoundedRectangleBorder) Dimensions() EdgeInsetsGeometry {
	return OutlinedBorderDimensions(b.Side())
}

// Scale scales the side and the radii by t.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.scale
func (b RoundedRectangleBorder) Scale(t float64) ShapeBorder {
	return RoundedRectangleBorder{side: b.Side().Scale(t), BorderRadius: BorderRadiusGeometryMul(b.radius(), t)}
}

// LerpFrom interpolates from a RoundedRectangleBorder or a CircleBorder.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.lerpFrom
func (b RoundedRectangleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if r, ok := a.(RoundedRectangleBorder); ok {
		return RoundedRectangleBorder{
			side:         BorderSideLerp(r.Side(), b.Side(), t),
			BorderRadius: BorderRadiusGeometryLerp(r.radius(), b.radius(), t),
		}
	}
	if c, ok := asCircleBorder(a); ok {
		return shapeToCircleBorder{
			kind: shapeRoundedRect, side: BorderSideLerp(c.Side(), b.Side(), t), borderRadius: b.radius(),
			circularity: 1.0 - t, eccentricity: c.Eccentricity,
		}
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to a RoundedRectangleBorder or a CircleBorder.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.lerpTo
func (b RoundedRectangleBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if r, ok := o.(RoundedRectangleBorder); ok {
		return RoundedRectangleBorder{
			side:         BorderSideLerp(b.Side(), r.Side(), t),
			BorderRadius: BorderRadiusGeometryLerp(b.radius(), r.radius(), t),
		}
	}
	if c, ok := asCircleBorder(o); ok {
		return shapeToCircleBorder{
			kind: shapeRoundedRect, side: BorderSideLerp(b.Side(), c.Side(), t), borderRadius: b.radius(),
			circularity: t, eccentricity: c.Eccentricity,
		}
	}
	return ShapeBorderDefaultLerpTo(b, o, t)
}

// RoundedRectangleBorderCopy holds the replacements for
// RoundedRectangleBorder.CopyWith.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.copyWith
type RoundedRectangleBorderCopy struct {
	Side         foundation.Opt[BorderSide]
	BorderRadius BorderRadiusGeometry
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.copyWith
func (b RoundedRectangleBorder) CopyWith(c RoundedRectangleBorderCopy) RoundedRectangleBorder {
	r := b.radius()
	if c.BorderRadius != nil {
		r = c.BorderRadius
	}
	return RoundedRectangleBorder{side: c.Side.OrElse(b.Side()), BorderRadius: r}
}

// CopyWithSide returns a copy with the given side.
//
// Flutter: painting/borders.dart OutlinedBorder.copyWith
func (b RoundedRectangleBorder) CopyWithSide(side BorderSide) OutlinedBorder {
	return b.CopyWith(RoundedRectangleBorderCopy{Side: foundation.Some(side)})
}

// GetInnerPath is the rounded rectangle, deflated by the stroke inset.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.getInnerPath
func (b RoundedRectangleBorder) GetInnerPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.radius().Resolve(td).ToRRect(rect).Deflate(b.Side().StrokeInset()))
	return p
}

// GetOuterPath is the rounded rectangle.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.getOuterPath
func (b RoundedRectangleBorder) GetOuterPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.radius().Resolve(td).ToRRect(rect))
	return p
}

// PaintInterior fills the rounded rectangle.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.paintInterior
func (b RoundedRectangleBorder) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, td ui.TextDirection) {
	if b.radius().Equal(BorderRadiusZero) {
		canvas.DrawRect(rect, paint)
		return
	}
	canvas.DrawRRect(b.radius().Resolve(td).ToRRect(rect), paint)
}

// PreferPaintInterior is true.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.preferPaintInterior
func (b RoundedRectangleBorder) PreferPaintInterior() bool { return true }

// Paint strokes the border.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.paint
func (b RoundedRectangleBorder) Paint(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	side := b.Side()
	if side.Style != BorderStyleSolid {
		return
	}
	if side.Width == 0.0 {
		canvas.DrawRRect(b.radius().Resolve(td).ToRRect(rect), side.ToPaint())
		return
	}
	paint := ui.NewPaint()
	paint.SetColor(side.Color)
	borderRect := b.radius().Resolve(td).ToRRect(rect)
	canvas.DrawDRRect(borderRect.Inflate(side.StrokeOutset()), borderRect.Deflate(side.StrokeInset()), paint)
}

// String describes the border.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedRectangleBorder.toString
func (b RoundedRectangleBorder) String() string {
	return "RoundedRectangleBorder(" + b.Side().String() + ", " + b.radius().String() + ")"
}

// RoundedSuperellipseBorder is a rectangular border with continuous
// (superellipse) rounded corners.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder
type RoundedSuperellipseBorder struct {
	ShapeBorderDefaults
	side BorderSide
	// BorderRadius is the radii for each corner; nil is BorderRadiusZero.
	BorderRadius BorderRadiusGeometry
}

// RoundedSuperellipseBorderOpts are the optional parameters of
// NewRoundedSuperellipseBorder.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.new
type RoundedSuperellipseBorderOpts = RoundedRectangleBorderOpts

// NewRoundedSuperellipseBorder creates a rounded superellipse border.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.new
func NewRoundedSuperellipseBorder(opts ...RoundedSuperellipseBorderOpts) RoundedSuperellipseBorder {
	o := firstOpts(opts)
	if o.BorderRadius == nil {
		o.BorderRadius = BorderRadiusZero
	}
	return RoundedSuperellipseBorder{side: o.Side.normalize(), BorderRadius: o.BorderRadius}
}

// Side is the border outline's color and weight.
//
// Flutter: painting/borders.dart OutlinedBorder.side
func (b RoundedSuperellipseBorder) Side() BorderSide { return b.side.normalize() }

func (b RoundedSuperellipseBorder) radius() BorderRadiusGeometry {
	if b.BorderRadius == nil {
		return BorderRadiusZero
	}
	return b.BorderRadius
}

// Dimensions is the stroke inset on all sides.
//
// Flutter: painting/borders.dart OutlinedBorder.dimensions
func (b RoundedSuperellipseBorder) Dimensions() EdgeInsetsGeometry {
	return OutlinedBorderDimensions(b.Side())
}

// Scale scales the side and the radii by t.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.scale
func (b RoundedSuperellipseBorder) Scale(t float64) ShapeBorder {
	return RoundedSuperellipseBorder{side: b.Side().Scale(t), BorderRadius: BorderRadiusGeometryMul(b.radius(), t)}
}

// LerpFrom interpolates from a RoundedSuperellipseBorder or a CircleBorder.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.lerpFrom
func (b RoundedSuperellipseBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if r, ok := a.(RoundedSuperellipseBorder); ok {
		return RoundedSuperellipseBorder{
			side:         BorderSideLerp(r.Side(), b.Side(), t),
			BorderRadius: BorderRadiusGeometryLerp(r.radius(), b.radius(), t),
		}
	}
	if c, ok := asCircleBorder(a); ok {
		return shapeToCircleBorder{
			kind: shapeSuperellipse, side: BorderSideLerp(c.Side(), b.Side(), t), borderRadius: b.radius(),
			circularity: 1.0 - t, eccentricity: c.Eccentricity,
		}
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to a RoundedSuperellipseBorder or a CircleBorder.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.lerpTo
func (b RoundedSuperellipseBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if r, ok := o.(RoundedSuperellipseBorder); ok {
		return RoundedSuperellipseBorder{
			side:         BorderSideLerp(b.Side(), r.Side(), t),
			BorderRadius: BorderRadiusGeometryLerp(b.radius(), r.radius(), t),
		}
	}
	if c, ok := asCircleBorder(o); ok {
		return shapeToCircleBorder{
			kind: shapeSuperellipse, side: BorderSideLerp(b.Side(), c.Side(), t), borderRadius: b.radius(),
			circularity: t, eccentricity: c.Eccentricity,
		}
	}
	return ShapeBorderDefaultLerpTo(b, o, t)
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.copyWith
func (b RoundedSuperellipseBorder) CopyWith(c RoundedRectangleBorderCopy) RoundedSuperellipseBorder {
	r := b.radius()
	if c.BorderRadius != nil {
		r = c.BorderRadius
	}
	return RoundedSuperellipseBorder{side: c.Side.OrElse(b.Side()), BorderRadius: r}
}

// CopyWithSide returns a copy with the given side.
//
// Flutter: painting/borders.dart OutlinedBorder.copyWith
func (b RoundedSuperellipseBorder) CopyWithSide(side BorderSide) OutlinedBorder {
	return b.CopyWith(RoundedRectangleBorderCopy{Side: foundation.Some(side)})
}

// GetInnerPath is the shape, deflated by the stroke inset.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.getInnerPath
func (b RoundedSuperellipseBorder) GetInnerPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	if b.radius().Equal(BorderRadiusZero) {
		p.AddRect(rect.Deflate(b.Side().StrokeInset()))
	} else {
		p.AddRSuperellipse(b.radius().Resolve(td).ToRSuperellipse(rect).Deflate(b.Side().StrokeInset()))
	}
	return p
}

// GetOuterPath is the shape.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.getOuterPath
func (b RoundedSuperellipseBorder) GetOuterPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	if b.radius().Equal(BorderRadiusZero) {
		p.AddRect(rect)
	} else {
		p.AddRSuperellipse(b.radius().Resolve(td).ToRSuperellipse(rect))
	}
	return p
}

// PaintInterior fills the shape.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.paintInterior
func (b RoundedSuperellipseBorder) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, td ui.TextDirection) {
	if b.radius().Equal(BorderRadiusZero) {
		canvas.DrawRect(rect, paint)
		return
	}
	canvas.DrawRSuperellipse(b.radius().Resolve(td).ToRSuperellipse(rect), paint)
}

// PreferPaintInterior is true.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.preferPaintInterior
func (b RoundedSuperellipseBorder) PreferPaintInterior() bool { return true }

// Paint strokes the border.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.paint
func (b RoundedSuperellipseBorder) Paint(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	side := b.Side()
	if side.Style != BorderStyleSolid {
		return
	}
	strokeOffset := (side.StrokeOutset() - side.StrokeInset()) / 2
	if b.radius().Equal(BorderRadiusZero) {
		canvas.DrawRect(rect.Inflate(strokeOffset), side.ToPaint())
		return
	}
	canvas.DrawRSuperellipse(b.radius().Resolve(td).ToRSuperellipse(rect).Inflate(strokeOffset), side.ToPaint())
}

// String describes the border.
//
// Flutter: painting/rounded_rectangle_border.dart RoundedSuperellipseBorder.toString
func (b RoundedSuperellipseBorder) String() string {
	return "RoundedSuperellipseBorder(" + b.Side().String() + ", " + b.radius().String() + ")"
}

type shapeKind int

const (
	shapeRoundedRect shapeKind = iota
	shapeSuperellipse
)

func (k shapeKind) name() string {
	if k == shapeSuperellipse {
		return "RoundedSuperellipseBorder"
	}
	return "RoundedRectangleBorder"
}

// shapeToCircleBorder is Flutter's _ShapeToCircleBorder: the interpolation of a
// RoundedRectangleBorder or RoundedSuperellipseBorder (by kind) to a
// CircleBorder.
type shapeToCircleBorder struct {
	ShapeBorderDefaults
	kind         shapeKind
	side         BorderSide
	borderRadius BorderRadiusGeometry
	circularity  float64
	eccentricity float64
}

func (b shapeToCircleBorder) Side() BorderSide { return b.side.normalize() }

func (b shapeToCircleBorder) Dimensions() EdgeInsetsGeometry {
	return OutlinedBorderDimensions(b.Side())
}

func (b shapeToCircleBorder) radius() BorderRadiusGeometry {
	if b.borderRadius == nil {
		return BorderRadiusZero
	}
	return b.borderRadius
}

func (b shapeToCircleBorder) copyWith(side BorderSide, borderRadius BorderRadiusGeometry, circularity, eccentricity float64) shapeToCircleBorder {
	return shapeToCircleBorder{kind: b.kind, side: side, borderRadius: borderRadius, circularity: circularity, eccentricity: eccentricity}
}

func (b shapeToCircleBorder) CopyWithSide(side BorderSide) OutlinedBorder {
	return b.copyWith(side, b.radius(), b.circularity, b.eccentricity)
}

func (b shapeToCircleBorder) drawShape(canvas *ui.Canvas, rect ui.Rect, radius BorderRadius, paint *ui.Paint, inflation *float64) {
	if b.kind == shapeSuperellipse {
		rs := radius.ToRSuperellipse(rect)
		if inflation != nil {
			rs = rs.Inflate(*inflation)
		}
		canvas.DrawRSuperellipse(rs, paint)
		return
	}
	rr := radius.ToRRect(rect)
	if inflation != nil {
		rr = rr.Inflate(*inflation)
	}
	canvas.DrawRRect(rr, paint)
}

func (b shapeToCircleBorder) buildPath(rect ui.Rect, radius BorderRadius, inflation *float64) *ui.Path {
	p := ui.NewPath()
	if b.kind == shapeSuperellipse {
		rs := radius.ToRSuperellipse(rect)
		if inflation != nil {
			rs = rs.Inflate(*inflation)
		}
		p.AddRSuperellipse(rs)
		return p
	}
	rr := radius.ToRRect(rect)
	if inflation != nil {
		rr = rr.Inflate(*inflation)
	}
	p.AddRRect(rr)
	return p
}

func (b shapeToCircleBorder) Scale(t float64) ShapeBorder {
	return b.copyWith(b.Side().Scale(t), BorderRadiusGeometryMul(b.radius(), t), t, b.eccentricity)
}

// likeKind returns the side and radius of a border of this shape's rounded kind.
func (b shapeToCircleBorder) likeKind(x ShapeBorder) (BorderSide, BorderRadiusGeometry, bool) {
	switch r := x.(type) {
	case RoundedRectangleBorder:
		if b.kind == shapeRoundedRect {
			return r.Side(), r.radius(), true
		}
	case RoundedSuperellipseBorder:
		if b.kind == shapeSuperellipse {
			return r.Side(), r.radius(), true
		}
	}
	return BorderSide{}, nil, false
}

func (b shapeToCircleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if side, br, ok := b.likeKind(a); ok {
		return b.copyWith(BorderSideLerp(side, b.Side(), t), BorderRadiusGeometryLerp(br, b.radius(), t), b.circularity*t, b.eccentricity)
	}
	if c, ok := asCircleBorder(a); ok {
		return b.copyWith(BorderSideLerp(c.Side(), b.Side(), t), b.radius(), b.circularity+(1.0-b.circularity)*(1.0-t), c.Eccentricity)
	}
	if o, ok := a.(shapeToCircleBorder); ok && o.kind == b.kind {
		return b.copyWith(BorderSideLerp(o.Side(), b.Side(), t), BorderRadiusGeometryLerp(o.radius(), b.radius(), t),
			ui.LerpDouble(o.circularity, b.circularity, t), b.eccentricity)
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

func (b shapeToCircleBorder) LerpTo(x ShapeBorder, t float64) ShapeBorder {
	if side, br, ok := b.likeKind(x); ok {
		return b.copyWith(BorderSideLerp(b.Side(), side, t), BorderRadiusGeometryLerp(b.radius(), br, t), b.circularity*(1.0-t), b.eccentricity)
	}
	if c, ok := asCircleBorder(x); ok {
		return b.copyWith(BorderSideLerp(b.Side(), c.Side(), t), b.radius(), b.circularity+(1.0-b.circularity)*t, c.Eccentricity)
	}
	if o, ok := x.(shapeToCircleBorder); ok && o.kind == b.kind {
		return b.copyWith(BorderSideLerp(b.Side(), o.Side(), t), BorderRadiusGeometryLerp(b.radius(), o.radius(), t),
			ui.LerpDouble(b.circularity, o.circularity, t), b.eccentricity)
	}
	return ShapeBorderDefaultLerpTo(b, x, t)
}

func adjustCircularRect(rect ui.Rect, circularity, eccentricity float64) ui.Rect {
	if circularity == 0.0 || rect.Width() == rect.Height() {
		return rect
	}
	if rect.Width() < rect.Height() {
		partialDelta := (rect.Height() - rect.Width()) / 2
		delta := circularity * partialDelta * (1.0 - eccentricity)
		return ui.NewRectFromLTRB(rect.Left, rect.Top+delta, rect.Right, rect.Bottom-delta)
	}
	partialDelta := (rect.Width() - rect.Height()) / 2
	delta := circularity * partialDelta * (1.0 - eccentricity)
	return ui.NewRectFromLTRB(rect.Left+delta, rect.Top, rect.Right-delta, rect.Bottom)
}

// ovalBorderRadius is the radius of the oval that the rect's corners turn into.
func ovalBorderRadius(rect ui.Rect, eccentricity float64) BorderRadius {
	if rect.Width() < rect.Height() {
		return NewBorderRadiusAll(ui.NewRadiusElliptical(rect.Width()/2, (0.5+eccentricity/2)*rect.Height()/2))
	}
	return NewBorderRadiusAll(ui.NewRadiusElliptical((0.5+eccentricity/2)*rect.Width()/2, rect.Height()/2))
}

func (b shapeToCircleBorder) adjustRect(rect ui.Rect) ui.Rect {
	return adjustCircularRect(rect, b.circularity, b.eccentricity)
}

func (b shapeToCircleBorder) adjustBorderRadius(rect ui.Rect, td ui.TextDirection) BorderRadius {
	resolved := b.radius().Resolve(td)
	if b.circularity == 0.0 {
		return resolved
	}
	target := NewBorderRadiusCircular(rect.ShortestSide() / 2)
	if b.eccentricity != 0.0 {
		target = ovalBorderRadius(rect, b.eccentricity)
	}
	return *BorderRadiusLerp(&resolved, &target, b.circularity)
}

func (b shapeToCircleBorder) GetInnerPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	side := b.Side()
	inflation := -ui.LerpDouble(side.Width, 0, side.StrokeAlign)
	return b.buildPath(b.adjustRect(rect), b.adjustBorderRadius(rect, td), &inflation)
}

func (b shapeToCircleBorder) GetOuterPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	return b.buildPath(b.adjustRect(rect), b.adjustBorderRadius(rect, td), nil)
}

func (b shapeToCircleBorder) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, td ui.TextDirection) {
	adjusted := b.adjustBorderRadius(rect, td)
	if adjusted == BorderRadiusZero {
		canvas.DrawRect(b.adjustRect(rect), paint)
		return
	}
	b.drawShape(canvas, b.adjustRect(rect), adjusted, paint, nil)
}

func (b shapeToCircleBorder) PreferPaintInterior() bool { return true }

func (b shapeToCircleBorder) Paint(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	side := b.Side()
	if side.Style != BorderStyleSolid {
		return
	}
	inflation := side.StrokeOffset() / 2
	b.drawShape(canvas, b.adjustRect(rect), b.adjustBorderRadius(rect, td), side.ToPaint(), &inflation)
}

func (b shapeToCircleBorder) String() string {
	pct := dartfmt.ToStringAsFixed(b.circularity*100, 1)
	if b.eccentricity != 0.0 {
		return b.kind.name() + "(" + b.Side().String() + ", " + b.radius().String() + ", " + pct +
			"% of the way to being a CircleBorder that is " + dartfmt.ToStringAsFixed(b.eccentricity*100, 1) + "% oval)"
	}
	return b.kind.name() + "(" + b.Side().String() + ", " + b.radius().String() + ", " + pct + "% of the way to being a CircleBorder)"
}

var (
	_ OutlinedBorder = RoundedRectangleBorder{}
	_ OutlinedBorder = RoundedSuperellipseBorder{}
	_ OutlinedBorder = shapeToCircleBorder{}
)
