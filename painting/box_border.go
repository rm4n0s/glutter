// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// BoxShape is the shape to use when painting a Border or BoxDecoration. The zero
// value is BoxShapeUnspecified: the default (rectangle).
//
// Flutter: painting/box_border.dart BoxShape
type BoxShape int

// The values of BoxShape follow Flutter's declaration order.
const (
	// BoxShapeUnspecified is the zero value: a rectangle.
	//
	// Flutter: no counterpart
	BoxShapeUnspecified BoxShape = iota
	// BoxShapeRectangle is an axis-aligned, 2-dimensional rectangle.
	//
	// Flutter: painting/box_border.dart BoxShape.rectangle
	BoxShapeRectangle
	// BoxShapeCircle is an axis-aligned, 2-dimensional circle.
	//
	// Flutter: painting/box_border.dart BoxShape.circle
	BoxShapeCircle
)

// Name returns the name of the value, such as "circle".
//
// Flutter: painting/box_border.dart BoxShape.name
func (s BoxShape) Name() string {
	switch s {
	case BoxShapeRectangle:
		return "rectangle"
	case BoxShapeCircle:
		return "circle"
	}
	return "unspecified"
}

// String returns the name Dart prints, such as "BoxShape.circle".
//
// Flutter: painting/box_border.dart BoxShape.toString
func (s BoxShape) String() string { return "BoxShape." + s.Name() }

func (s BoxShape) orDefault() BoxShape {
	if s == BoxShapeUnspecified {
		return BoxShapeRectangle
	}
	return s
}

// BoxPaintOpts are the optional parameters of BoxBorder.PaintBox.
//
// Flutter: painting/box_border.dart BoxBorder.paint
type BoxPaintOpts struct {
	// TextDirection is the direction that a BorderDirectional needs when its
	// sides differ.
	TextDirection ui.TextDirection
	// Shape is the shape of the border; the default is a rectangle.
	Shape BoxShape
	// BorderRadius rounds the corners of a rectangle; None for none.
	BorderRadius foundation.Opt[BorderRadius]
}

// BoxBorder is a border of a box, comprised of four sides: the top, right,
// bottom, and left. The sides are named TopSide and BottomSide (the properties
// Top and Bottom are the fields of the concrete types).
//
// Flutter: painting/box_border.dart BoxBorder
type BoxBorder interface {
	ShapeBorder
	// TopSide is the top side of this border.
	TopSide() BorderSide
	// BottomSide is the bottom side of this border.
	BottomSide() BorderSide
	// IsUniform reports whether all four sides are the same in color, width,
	// style and stroke alignment.
	IsUniform() bool
	// PaintBox paints the border within the given rect on the given canvas.
	// Paint is PaintBox with the default shape and no radius.
	PaintBox(canvas *ui.Canvas, rect ui.Rect, opts BoxPaintOpts)
}

// BoxBorderLerp linearly interpolates between two BoxBorders. Both must be a
// Border or a BorderDirectional (or nil), or it panics with the tag
// BoxBorderLerpInvalid.
//
// Flutter: painting/box_border.dart BoxBorder.lerp
func BoxBorderLerp(a, b BoxBorder, t float64) BoxBorder {
	if a == nil && b == nil {
		return nil
	}
	if a != nil && b != nil && sameBorder(a, b) {
		return a
	}
	an, aIsBorder := asBorder(a)
	bn, bIsBorder := asBorder(b)
	if (a == nil || aIsBorder) && (b == nil || bIsBorder) {
		return borderLerpBox(an, bn, t, a == nil, b == nil)
	}
	ad, aIsDir := asBorderDirectional(a)
	bd, bIsDir := asBorderDirectional(b)
	if (a == nil || aIsDir) && (b == nil || bIsDir) {
		return borderDirectionalLerpBox(ad, bd, t, a == nil, b == nil)
	}
	if bIsBorder && aIsDir {
		an, bd = bn, ad
		aIsBorder, bIsDir = true, true
		t = 1.0 - t
	}
	if aIsBorder && bIsDir {
		if bd.Start.normalize() == BorderSideNone && bd.End.normalize() == BorderSideNone {
			return Border{
				Top:    BorderSideLerp(an.Top.normalize(), bd.Top.normalize(), t),
				Right:  BorderSideLerp(an.Right.normalize(), BorderSideNone, t),
				Bottom: BorderSideLerp(an.Bottom.normalize(), bd.Bottom.normalize(), t),
				Left:   BorderSideLerp(an.Left.normalize(), BorderSideNone, t),
			}
		}
		if an.Left.normalize() == BorderSideNone && an.Right.normalize() == BorderSideNone {
			return BorderDirectional{
				Top:    BorderSideLerp(an.Top.normalize(), bd.Top.normalize(), t),
				Start:  BorderSideLerp(BorderSideNone, bd.Start.normalize(), t),
				End:    BorderSideLerp(BorderSideNone, bd.End.normalize(), t),
				Bottom: BorderSideLerp(an.Bottom.normalize(), bd.Bottom.normalize(), t),
			}
		}
		if t < 0.5 {
			return Border{
				Top:    BorderSideLerp(an.Top.normalize(), bd.Top.normalize(), t),
				Right:  BorderSideLerp(an.Right.normalize(), BorderSideNone, t*2.0),
				Bottom: BorderSideLerp(an.Bottom.normalize(), bd.Bottom.normalize(), t),
				Left:   BorderSideLerp(an.Left.normalize(), BorderSideNone, t*2.0),
			}
		}
		return BorderDirectional{
			Top:    BorderSideLerp(an.Top.normalize(), bd.Top.normalize(), t),
			Start:  BorderSideLerp(BorderSideNone, bd.Start.normalize(), (t-0.5)*2.0),
			End:    BorderSideLerp(BorderSideNone, bd.End.normalize(), (t-0.5)*2.0),
			Bottom: BorderSideLerp(an.Bottom.normalize(), bd.Bottom.normalize(), t),
		}
	}
	panic(errors.New("BoxBorderLerpInvalid",
		"BoxBorder.lerp can only interpolate Border and BorderDirectional classes; for a more general interpolation method, consider using ShapeBorderLerp instead",
		"a", describeBorder(a), "b", describeBorder(b)))
}

func describeBorder(b BoxBorder) string {
	if b == nil {
		return "null"
	}
	return foundation.RuntimeTypeName(b) + ": " + b.String()
}

func asBorder(b BoxBorder) (Border, bool) {
	if v, ok := b.(Border); ok {
		return v, true
	}
	return Border{}, false
}

func asBorderDirectional(b BoxBorder) (BorderDirectional, bool) {
	if v, ok := b.(BorderDirectional); ok {
		return v, true
	}
	return BorderDirectional{}, false
}

func borderLerpBox(a, b Border, t float64, aNil, bNil bool) BoxBorder {
	var pa, pb *Border
	if !aNil {
		pa = &a
	}
	if !bNil {
		pb = &b
	}
	if r := BorderLerp(pa, pb, t); r != nil {
		return *r
	}
	return nil
}

func borderDirectionalLerpBox(a, b BorderDirectional, t float64, aNil, bNil bool) BoxBorder {
	var pa, pb *BorderDirectional
	if !aNil {
		pa = &a
	}
	if !bNil {
		pb = &b
	}
	if r := BorderDirectionalLerp(pa, pb, t); r != nil {
		return *r
	}
	return nil
}

// boxBorderGetInnerPath is BoxBorder.getInnerPath: the rect inset by the
// dimensions.
func boxBorderGetInnerPath(b BoxBorder, rect ui.Rect, td ui.TextDirection) *ui.Path {
	if foundation.DebugMode && td == ui.TextDirectionUnspecified {
		panic(errors.New("TextDirectionRequired",
			"the textDirection argument to "+foundation.RuntimeTypeName(b)+".GetInnerPath must not be unspecified"))
	}
	p := ui.NewPath()
	p.AddRect(b.Dimensions().Resolve(td).DeflateRect(rect))
	return p
}

func boxBorderGetOuterPath(b BoxBorder, rect ui.Rect, td ui.TextDirection) *ui.Path {
	if foundation.DebugMode && td == ui.TextDirectionUnspecified {
		panic(errors.New("TextDirectionRequired",
			"the textDirection argument to "+foundation.RuntimeTypeName(b)+".GetOuterPath must not be unspecified"))
	}
	p := ui.NewPath()
	p.AddRect(rect)
	return p
}

func paintUniformBorderWithRadius(canvas *ui.Canvas, rect ui.Rect, side BorderSide, br BorderRadius) {
	paint := ui.NewPaint()
	paint.SetColor(side.Color)
	width := side.Width
	if width == 0 {
		paint.SetStyle(ui.PaintingStyleStroke)
		paint.SetStrokeWidth(0)
		canvas.DrawRRect(br.ToRRect(rect), paint)
		return
	}
	borderRect := br.ToRRect(rect)
	inner := borderRect.Deflate(side.StrokeInset())
	outer := borderRect.Inflate(side.StrokeOutset())
	canvas.DrawDRRect(outer, inner, paint)
}

// NonUniformBorderOpts are the parameters of PaintNonUniformBorder.
//
// Flutter: painting/box_border.dart BoxBorder.paintNonUniformBorder
type NonUniformBorderOpts struct {
	// BorderRadius rounds the corners of a rectangle; None for none.
	BorderRadius foundation.Opt[BorderRadiusGeometry]
	// TextDirection resolves a directional border radius.
	TextDirection ui.TextDirection
	// Shape is the shape of the border; the default is a rectangle.
	Shape BoxShape
	// Top, Right, Bottom and Left are the sides; BorderSideNone for the ones
	// that are left out.
	Top, Right, Bottom, Left BorderSide
	// Color is the color of the border.
	Color ui.Color
}

// PaintNonUniformBorder paints a border of different widths around a rectangle
// or a circle, with a single color.
//
// Flutter: painting/box_border.dart BoxBorder.paintNonUniformBorder
func PaintNonUniformBorder(canvas *ui.Canvas, rect ui.Rect, o NonUniformBorderOpts) {
	var borderRect ui.RRect
	if o.Shape.orDefault() == BoxShapeCircle {
		if foundation.DebugMode && o.BorderRadius.IsSome() {
			panic(errors.New("BoxBorderCircleRadius",
				"a circle cannot have a border radius; remove either the shape or the borderRadius argument"))
		}
		borderRect = ui.NewRRectFromRectAndRadius(ui.NewRectFromCircle(rect.Center(), rect.ShortestSide()/2.0), ui.NewRadiusCircular(rect.Width()))
	} else {
		var br BorderRadiusGeometry = BorderRadiusZero
		if v, ok := o.BorderRadius.Get(); ok {
			br = v
		}
		borderRect = br.Resolve(o.TextDirection).ToRRect(rect)
	}
	paint := ui.NewPaint()
	paint.SetColor(o.Color)
	inner := NewEdgeInsetsFromLTRB(o.Left.StrokeInset(), o.Top.StrokeInset(), o.Right.StrokeInset(), o.Bottom.StrokeInset()).DeflateRRect(borderRect)
	outer := NewEdgeInsetsFromLTRB(o.Left.StrokeOutset(), o.Top.StrokeOutset(), o.Right.StrokeOutset(), o.Bottom.StrokeOutset()).InflateRRect(borderRect)
	canvas.DrawDRRect(outer, inner, paint)
}

func paintUniformBorderWithCircle(canvas *ui.Canvas, rect ui.Rect, side BorderSide) {
	radius := (rect.ShortestSide() + side.StrokeOffset()) / 2
	canvas.DrawCircle(rect.Center(), radius, side.ToPaint())
}

func paintUniformBorderWithRectangle(canvas *ui.Canvas, rect ui.Rect, side BorderSide) {
	canvas.DrawRect(rect.Inflate(side.StrokeOffset()/2), side.ToPaint())
}

// boxSides is the sides of a border in the order the colors are collected
// (top, right or end, bottom, left or start), already resolved to the visual
// left and right.
type boxSides struct{ top, right, bottom, left BorderSide }

func (s boxSides) colorIsUniform() bool {
	c := s.top.Color
	return s.left.Color == c && s.bottom.Color == c && s.right.Color == c
}

func (s boxSides) widthIsUniform() bool {
	w := s.top.Width
	return s.left.Width == w && s.bottom.Width == w && s.right.Width == w
}

func (s boxSides) styleIsUniform() bool {
	st := s.top.Style
	return s.left.Style == st && s.bottom.Style == st && s.right.Style == st
}

func (s boxSides) strokeAlignIsUniform() bool {
	a := s.top.StrokeAlign
	return s.left.StrokeAlign == a && s.bottom.StrokeAlign == a && s.right.StrokeAlign == a
}

func (s boxSides) isUniform() bool {
	return s.colorIsUniform() && s.widthIsUniform() && s.styleIsUniform() && s.strokeAlignIsUniform()
}

// distinctVisibleColors lists the distinct colors of the sides that have a
// style, in the order top, right, bottom, left.
func (s boxSides) distinctVisibleColors() []ui.Color {
	var out []ui.Color
	add := func(side BorderSide) {
		if side.Style == BorderStyleNone {
			return
		}
		for _, c := range out {
			if c == side.Color {
				return
			}
		}
		out = append(out, side.Color)
	}
	add(s.top)
	add(s.right)
	add(s.bottom)
	add(s.left)
	return out
}

func (s boxSides) hasHairlineBorder() bool {
	hair := func(b BorderSide) bool { return b.Style == BorderStyleSolid && b.Width == 0 }
	return hair(s.top) || hair(s.right) || hair(s.bottom) || hair(s.left)
}

func borderRadiusIsRound(br foundation.Opt[BorderRadius]) bool {
	v, ok := br.Get()
	return ok && v != BorderRadiusZero
}

// paintBoxSides is the paint method both Border and BorderDirectional share;
// the directional one resolves left and right first.
func paintBoxSides(canvas *ui.Canvas, rect ui.Rect, s boxSides, opts BoxPaintOpts, directional bool) {
	shape := opts.Shape.orDefault()
	if s.isUniform() {
		if s.top.Style == BorderStyleNone {
			return
		}
		if s.top.Style == BorderStyleSolid {
			if shape == BoxShapeCircle {
				if foundation.DebugMode && opts.BorderRadius.IsSome() {
					panic(errors.New("BoxBorderCircleRadius",
						"a circle cannot have a border radius; remove either the shape or the borderRadius argument"))
				}
				paintUniformBorderWithCircle(canvas, rect, s.top)
			} else {
				if borderRadiusIsRound(opts.BorderRadius) {
					paintUniformBorderWithRadius(canvas, rect, s.top, opts.BorderRadius.Value())
					return
				}
				paintUniformBorderWithRectangle(canvas, rect, s.top)
			}
			return
		}
	}
	if s.styleIsUniform() && s.top.Style == BorderStyleNone {
		return
	}
	colors := s.distinctVisibleColors()
	hairline := s.hasHairlineBorder()
	if len(colors) == 1 && !hairline && (shape == BoxShapeCircle || borderRadiusIsRound(opts.BorderRadius)) {
		side := func(b BorderSide) BorderSide {
			if b.Style == BorderStyleNone {
				return BorderSideNone
			}
			return b
		}
		var br foundation.Opt[BorderRadiusGeometry]
		if v, ok := opts.BorderRadius.Get(); ok {
			br = foundation.Some[BorderRadiusGeometry](v)
		}
		PaintNonUniformBorder(canvas, rect, NonUniformBorderOpts{
			Shape: shape, BorderRadius: br, TextDirection: opts.TextDirection,
			Top: side(s.top), Right: side(s.right), Bottom: side(s.bottom), Left: side(s.left), Color: colors[0],
		})
		return
	}
	if foundation.DebugMode {
		if hairline && borderRadiusIsRound(opts.BorderRadius) {
			panic(errors.New("BoxBorderHairlineRadius",
				"a hairline border like BorderSide(width: 0.0, style: solid) can only be drawn when BorderRadius is zero or nil"))
		}
		if opts.BorderRadius.IsSome() && (!directional && borderRadiusIsRound(opts.BorderRadius) || directional) {
			panic(errors.New("BoxBorderNonUniformRadius", "a borderRadius can only be given on borders with uniform colors"))
		}
		if shape != BoxShapeRectangle {
			panic(errors.New("BoxBorderNonUniformShape", "a Border can only be drawn as a circle on borders with uniform colors"))
		}
		if !s.strokeAlignIsUniform() || s.top.StrokeAlign != StrokeAlignInside {
			panic(errors.New("BoxBorderStrokeAlignNonUniform",
				"a Border can only draw strokeAlign different than StrokeAlignInside on borders with uniform colors"))
		}
	}
	PaintBorder(canvas, rect, BorderPaintOpts{Top: s.top, Right: s.right, Bottom: s.bottom, Left: s.left})
}

// Border is a border of a box, comprised of four sides: the top, right, bottom,
// and left. A side left out of a literal is BorderSideNone.
//
// Flutter: painting/box_border.dart Border
type Border struct {
	// Top is the top side of this border.
	Top BorderSide
	// Right is the right side of this border.
	Right BorderSide
	// Bottom is the bottom side of this border.
	Bottom BorderSide
	// Left is the left side of this border.
	Left BorderSide
}

// NewBorderFromBorderSide creates a border whose sides are all the same.
//
// Flutter: painting/box_border.dart Border.fromBorderSide
func NewBorderFromBorderSide(side BorderSide) Border { return Border{side, side, side, side} }

// NewBorderSymmetric creates a border with symmetrical vertical and horizontal
// sides: vertical is the left and right side, horizontal the top and bottom.
//
// Flutter: painting/box_border.dart Border.symmetric
func NewBorderSymmetric(vertical, horizontal BorderSide) Border {
	return Border{Top: horizontal, Right: vertical, Bottom: horizontal, Left: vertical}
}

// NewBorderAll creates a border whose sides are all the same.
//
// Flutter: painting/box_border.dart Border.all
func NewBorderAll(opts ...BorderSideOpts) Border {
	return NewBorderFromBorderSide(NewBorderSide(opts...))
}

// BorderMerge creates a border that represents the addition of the two given
// borders.
//
// Flutter: painting/box_border.dart Border.merge
func BorderMerge(a, b Border) Border {
	if foundation.DebugMode {
		for _, p := range [][2]BorderSide{{a.Top, b.Top}, {a.Right, b.Right}, {a.Bottom, b.Bottom}, {a.Left, b.Left}} {
			if !BorderSideCanMerge(p[0].normalize(), p[1].normalize()) {
				panic(errors.New("BorderSideCannotMerge", "the border sides have different styles or colors"))
			}
		}
	}
	return Border{
		Top:    BorderSideMerge(a.Top.normalize(), b.Top.normalize()),
		Right:  BorderSideMerge(a.Right.normalize(), b.Right.normalize()),
		Bottom: BorderSideMerge(a.Bottom.normalize(), b.Bottom.normalize()),
		Left:   BorderSideMerge(a.Left.normalize(), b.Left.normalize()),
	}
}

func (b Border) sides() boxSides {
	return boxSides{b.Top.normalize(), b.Right.normalize(), b.Bottom.normalize(), b.Left.normalize()}
}

// TopSide is the top side of this border.
//
// Flutter: painting/box_border.dart Border.top
func (b Border) TopSide() BorderSide { return b.Top.normalize() }

// BottomSide is the bottom side of this border.
//
// Flutter: painting/box_border.dart Border.bottom
func (b Border) BottomSide() BorderSide { return b.Bottom.normalize() }

// Dimensions is the widths of the sides: the stroke insets.
//
// Flutter: painting/box_border.dart Border.dimensions
func (b Border) Dimensions() EdgeInsetsGeometry {
	s := b.sides()
	return NewEdgeInsetsFromLTRB(s.left.StrokeInset(), s.top.StrokeInset(), s.right.StrokeInset(), s.bottom.StrokeInset())
}

// IsUniform reports whether all four sides are the same in color, width, style
// and stroke alignment.
//
// Flutter: painting/box_border.dart Border.isUniform
func (b Border) IsUniform() bool { return b.sides().isUniform() }

// Add merges this border with another Border when their sides can be merged;
// otherwise it returns nil.
//
// Flutter: painting/box_border.dart Border.add
func (b Border) Add(other ShapeBorder, reversed bool) ShapeBorder {
	o, ok := other.(Border)
	if ok {
		x, y := b.sides(), o.sides()
		if BorderSideCanMerge(x.top, y.top) && BorderSideCanMerge(x.right, y.right) &&
			BorderSideCanMerge(x.bottom, y.bottom) && BorderSideCanMerge(x.left, y.left) {
			return BorderMerge(b, o)
		}
	}
	return nil
}

// Scale creates a copy of the border with its sides scaled by t.
//
// Flutter: painting/box_border.dart Border.scale
func (b Border) Scale(t float64) ShapeBorder {
	s := b.sides()
	return Border{s.top.Scale(t), s.right.Scale(t), s.bottom.Scale(t), s.left.Scale(t)}
}

// LerpFrom interpolates from another Border.
//
// Flutter: painting/box_border.dart Border.lerpFrom
func (b Border) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(Border); ok {
		return *BorderLerp(&x, &b, t)
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to another Border.
//
// Flutter: painting/box_border.dart Border.lerpTo
func (b Border) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if x, ok := o.(Border); ok {
		return *BorderLerp(&b, &x, t)
	}
	return ShapeBorderDefaultLerpTo(b, o, t)
}

// BorderLerp linearly interpolates between two Borders. Either may be nil.
//
// Flutter: painting/box_border.dart Border.lerp
func BorderLerp(a, b *Border, t float64) *Border {
	if a == b {
		return a
	}
	if a == nil {
		r := b.Scale(t).(Border)
		return &r
	}
	if b == nil {
		r := a.Scale(1.0 - t).(Border)
		return &r
	}
	x, y := a.sides(), b.sides()
	return &Border{
		Top:    BorderSideLerp(x.top, y.top, t),
		Right:  BorderSideLerp(x.right, y.right, t),
		Bottom: BorderSideLerp(x.bottom, y.bottom, t),
		Left:   BorderSideLerp(x.left, y.left, t),
	}
}

// GetInnerPath is the rect inset by the dimensions.
//
// Flutter: painting/box_border.dart BoxBorder.getInnerPath
func (b Border) GetInnerPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	return boxBorderGetInnerPath(b, rect, td)
}

// GetOuterPath is the rect.
//
// Flutter: painting/box_border.dart BoxBorder.getOuterPath
func (b Border) GetOuterPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	return boxBorderGetOuterPath(b, rect, td)
}

// PaintInterior fills the rect.
//
// Flutter: painting/box_border.dart BoxBorder.paintInterior
func (b Border) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, _ ui.TextDirection) {
	canvas.DrawRect(rect, paint)
}

// PreferPaintInterior is true: filling the rect is faster than the path.
//
// Flutter: painting/box_border.dart BoxBorder.preferPaintInterior
func (b Border) PreferPaintInterior() bool { return true }

// Paint paints the border around the rect.
//
// Flutter: painting/box_border.dart Border.paint
func (b Border) Paint(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	b.PaintBox(canvas, rect, BoxPaintOpts{TextDirection: td})
}

// PaintBox paints the border around the rect, with a shape and a border radius.
//
// Flutter: painting/box_border.dart Border.paint
func (b Border) PaintBox(canvas *ui.Canvas, rect ui.Rect, opts BoxPaintOpts) {
	paintBoxSides(canvas, rect, b.sides(), opts, false)
}

// String describes the border, such as "Border.all(BorderSide(...))".
//
// Flutter: painting/box_border.dart Border.toString
func (b Border) String() string {
	s := b.sides()
	if s.isUniform() {
		return "Border.all(" + foundation.DiagnosticableString(s.top) + ")"
	}
	var args []string
	if s.top != BorderSideNone {
		args = append(args, "top: "+foundation.DiagnosticableString(s.top))
	}
	if s.right != BorderSideNone {
		args = append(args, "right: "+foundation.DiagnosticableString(s.right))
	}
	if s.bottom != BorderSideNone {
		args = append(args, "bottom: "+foundation.DiagnosticableString(s.bottom))
	}
	if s.left != BorderSideNone {
		args = append(args, "left: "+foundation.DiagnosticableString(s.left))
	}
	return "Border(" + strings.Join(args, ", ") + ")"
}

// BorderDirectional is a border of a box, comprised of four sides, where the
// horizontal sides are dependent on the writing direction.
//
// Flutter: painting/box_border.dart BorderDirectional
type BorderDirectional struct {
	// Top is the top side of this border.
	Top BorderSide
	// Start is the start side of this border.
	Start BorderSide
	// End is the end side of this border.
	End BorderSide
	// Bottom is the bottom side of this border.
	Bottom BorderSide
}

// BorderDirectionalMerge creates a border that represents the addition of the
// two given borders.
//
// Flutter: painting/box_border.dart BorderDirectional.merge
func BorderDirectionalMerge(a, b BorderDirectional) BorderDirectional {
	if foundation.DebugMode {
		for _, p := range [][2]BorderSide{{a.Top, b.Top}, {a.Start, b.Start}, {a.End, b.End}, {a.Bottom, b.Bottom}} {
			if !BorderSideCanMerge(p[0].normalize(), p[1].normalize()) {
				panic(errors.New("BorderSideCannotMerge", "the border sides have different styles or colors"))
			}
		}
	}
	return BorderDirectional{
		Top:    BorderSideMerge(a.Top.normalize(), b.Top.normalize()),
		Start:  BorderSideMerge(a.Start.normalize(), b.Start.normalize()),
		End:    BorderSideMerge(a.End.normalize(), b.End.normalize()),
		Bottom: BorderSideMerge(a.Bottom.normalize(), b.Bottom.normalize()),
	}
}

func (b BorderDirectional) norm() BorderDirectional {
	return BorderDirectional{b.Top.normalize(), b.Start.normalize(), b.End.normalize(), b.Bottom.normalize()}
}

// asSides lays the sides out in the visual order, with start as the left side
// (the left-to-right layout), which suffices for the uniformity checks.
func (b BorderDirectional) asSides() boxSides {
	n := b.norm()
	return boxSides{top: n.Top, right: n.End, bottom: n.Bottom, left: n.Start}
}

// TopSide is the top side of this border.
//
// Flutter: painting/box_border.dart BorderDirectional.top
func (b BorderDirectional) TopSide() BorderSide { return b.Top.normalize() }

// BottomSide is the bottom side of this border.
//
// Flutter: painting/box_border.dart BorderDirectional.bottom
func (b BorderDirectional) BottomSide() BorderSide { return b.Bottom.normalize() }

// Dimensions is the widths of the sides: the stroke insets.
//
// Flutter: painting/box_border.dart BorderDirectional.dimensions
func (b BorderDirectional) Dimensions() EdgeInsetsGeometry {
	n := b.norm()
	return NewEdgeInsetsDirectionalFromSTEB(n.Start.StrokeInset(), n.Top.StrokeInset(), n.End.StrokeInset(), n.Bottom.StrokeInset())
}

// IsUniform reports whether all four sides are the same in color, width, style
// and stroke alignment.
//
// Flutter: painting/box_border.dart BorderDirectional.isUniform
func (b BorderDirectional) IsUniform() bool { return b.asSides().isUniform() }

// Add merges this border with another border when their sides can be merged;
// otherwise it returns nil.
//
// Flutter: painting/box_border.dart BorderDirectional.add
func (b BorderDirectional) Add(other ShapeBorder, reversed bool) ShapeBorder {
	n := b.norm()
	switch o := other.(type) {
	case BorderDirectional:
		t := o.norm()
		if BorderSideCanMerge(n.Top, t.Top) && BorderSideCanMerge(n.Start, t.Start) &&
			BorderSideCanMerge(n.End, t.End) && BorderSideCanMerge(n.Bottom, t.Bottom) {
			return BorderDirectionalMerge(b, o)
		}
		return nil
	case Border:
		t := o.sides()
		if !BorderSideCanMerge(t.top, n.Top) || !BorderSideCanMerge(t.bottom, n.Bottom) {
			return nil
		}
		if n.Start != BorderSideNone || n.End != BorderSideNone {
			if t.left != BorderSideNone || t.right != BorderSideNone {
				return nil
			}
			return BorderDirectional{Top: BorderSideMerge(t.top, n.Top), Start: n.Start, End: n.End, Bottom: BorderSideMerge(t.bottom, n.Bottom)}
		}
		return Border{Top: BorderSideMerge(t.top, n.Top), Right: t.right, Bottom: BorderSideMerge(t.bottom, n.Bottom), Left: t.left}
	}
	return nil
}

// Scale creates a copy of the border with its sides scaled by t.
//
// Flutter: painting/box_border.dart BorderDirectional.scale
func (b BorderDirectional) Scale(t float64) ShapeBorder {
	n := b.norm()
	return BorderDirectional{n.Top.Scale(t), n.Start.Scale(t), n.End.Scale(t), n.Bottom.Scale(t)}
}

// LerpFrom interpolates from another BorderDirectional.
//
// Flutter: painting/box_border.dart BorderDirectional.lerpFrom
func (b BorderDirectional) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(BorderDirectional); ok {
		return *BorderDirectionalLerp(&x, &b, t)
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to another BorderDirectional.
//
// Flutter: painting/box_border.dart BorderDirectional.lerpTo
func (b BorderDirectional) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if x, ok := o.(BorderDirectional); ok {
		return *BorderDirectionalLerp(&b, &x, t)
	}
	return ShapeBorderDefaultLerpTo(b, o, t)
}

// BorderDirectionalLerp linearly interpolates between two BorderDirectionals.
// Either may be nil.
//
// Flutter: painting/box_border.dart BorderDirectional.lerp
func BorderDirectionalLerp(a, b *BorderDirectional, t float64) *BorderDirectional {
	if a == b {
		return a
	}
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		r := b.Scale(t).(BorderDirectional)
		return &r
	}
	if b == nil {
		r := a.Scale(1.0 - t).(BorderDirectional)
		return &r
	}
	x, y := a.norm(), b.norm()
	return &BorderDirectional{
		Top:    BorderSideLerp(x.Top, y.Top, t),
		End:    BorderSideLerp(x.End, y.End, t),
		Bottom: BorderSideLerp(x.Bottom, y.Bottom, t),
		Start:  BorderSideLerp(x.Start, y.Start, t),
	}
}

// GetInnerPath is the rect inset by the dimensions.
//
// Flutter: painting/box_border.dart BoxBorder.getInnerPath
func (b BorderDirectional) GetInnerPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	return boxBorderGetInnerPath(b, rect, td)
}

// GetOuterPath is the rect.
//
// Flutter: painting/box_border.dart BoxBorder.getOuterPath
func (b BorderDirectional) GetOuterPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	return boxBorderGetOuterPath(b, rect, td)
}

// PaintInterior fills the rect.
//
// Flutter: painting/box_border.dart BoxBorder.paintInterior
func (b BorderDirectional) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, _ ui.TextDirection) {
	canvas.DrawRect(rect, paint)
}

// PreferPaintInterior is true: filling the rect is faster than the path.
//
// Flutter: painting/box_border.dart BoxBorder.preferPaintInterior
func (b BorderDirectional) PreferPaintInterior() bool { return true }

// Paint paints the border around the rect.
//
// Flutter: painting/box_border.dart BorderDirectional.paint
func (b BorderDirectional) Paint(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	b.PaintBox(canvas, rect, BoxPaintOpts{TextDirection: td})
}

// PaintBox paints the border around the rect, with a shape and a border radius.
// A border whose sides differ needs a text direction.
//
// Flutter: painting/box_border.dart BorderDirectional.paint
func (b BorderDirectional) PaintBox(canvas *ui.Canvas, rect ui.Rect, opts BoxPaintOpts) {
	s := b.asSides()
	if !s.isUniform() && (!s.styleIsUniform() || s.top.Style != BorderStyleNone) {
		if foundation.DebugMode && opts.TextDirection == ui.TextDirectionUnspecified {
			panic(errors.New("TextDirectionRequired", "non-uniform BorderDirectional objects require a TextDirection when painting"))
		}
		if opts.TextDirection == ui.TextDirectionRTL {
			s.left, s.right = s.right, s.left
		}
	}
	paintBoxSides(canvas, rect, s, opts, true)
}

// String describes the border, such as "BorderDirectional(top: BorderSide(...))".
//
// Flutter: painting/box_border.dart BorderDirectional.toString
func (b BorderDirectional) String() string {
	n := b.norm()
	var args []string
	if n.Top != BorderSideNone {
		args = append(args, "top: "+foundation.DiagnosticableString(n.Top))
	}
	if n.Start != BorderSideNone {
		args = append(args, "start: "+foundation.DiagnosticableString(n.Start))
	}
	if n.End != BorderSideNone {
		args = append(args, "end: "+foundation.DiagnosticableString(n.End))
	}
	if n.Bottom != BorderSideNone {
		args = append(args, "bottom: "+foundation.DiagnosticableString(n.Bottom))
	}
	return "BorderDirectional(" + strings.Join(args, ", ") + ")"
}

var (
	_ BoxBorder = Border{}
	_ BoxBorder = BorderDirectional{}
)
