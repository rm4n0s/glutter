// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"fmt"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// alignParts are the three components of an AlignmentGeometry: Flutter's
// protected getters _x, _start and _y.
type alignParts struct{ x, start, y float64 }

func (p alignParts) String() string {
	if p.start == 0 {
		return stringifyAlignment(p.x, p.y)
	}
	if p.x == 0 {
		return stringifyAlignmentDirectional(p.start, p.y)
	}
	return stringifyAlignment(p.x, p.y) + " + " + stringifyAlignmentDirectional(p.start, 0)
}

var alignmentNames = map[[2]float64]string{
	{-1, -1}: "topLeft", {0, -1}: "topCenter", {1, -1}: "topRight",
	{-1, 0}: "centerLeft", {0, 0}: "center", {1, 0}: "centerRight",
	{-1, 1}: "bottomLeft", {0, 1}: "bottomCenter", {1, 1}: "bottomRight",
}

var alignmentDirectionalNames = map[[2]float64]string{
	{-1, -1}: "topStart", {0, -1}: "topCenter", {1, -1}: "topEnd",
	{-1, 0}: "centerStart", {0, 0}: "center", {1, 0}: "centerEnd",
	{-1, 1}: "bottomStart", {0, 1}: "bottomCenter", {1, 1}: "bottomEnd",
}

func stringifyAlignment(x, y float64) string {
	if n, ok := alignmentNames[[2]float64{x, y}]; ok {
		return "Alignment." + n
	}
	return fmt.Sprintf("Alignment(%s, %s)", dartfmt.ToStringAsFixed(x, 1), dartfmt.ToStringAsFixed(y, 1))
}

func stringifyAlignmentDirectional(start, y float64) string {
	if n, ok := alignmentDirectionalNames[[2]float64{start, y}]; ok {
		return "AlignmentDirectional." + n
	}
	return fmt.Sprintf("AlignmentDirectional(%s, %s)", dartfmt.ToStringAsFixed(start, 1), dartfmt.ToStringAsFixed(y, 1))
}

// AlignmentGeometry is a base class for Alignment and AlignmentDirectional,
// representing a point in a rectangle. Its covariant operations (Neg, Mul, Div,
// TruncDiv and Mod) are the functions AlignmentGeometryNeg, … and the methods of
// the concrete types.
//
// Flutter: painting/alignment.dart AlignmentGeometry
type AlignmentGeometry interface {
	// Add returns the sum of the components of this and other.
	Add(other AlignmentGeometry) AlignmentGeometry
	// Resolve returns an Alignment for the text direction. A directional value
	// panics with the tag TextDirectionRequired in debug builds if the
	// direction is unspecified.
	Resolve(direction ui.TextDirection) Alignment
	// Equal reports whether the three components are equal.
	Equal(other AlignmentGeometry) bool
	String() string
	alignParts() alignParts
}

type mixedAlignment alignParts

func (m mixedAlignment) alignParts() alignParts { return alignParts(m) }
func (m mixedAlignment) Add(o AlignmentGeometry) AlignmentGeometry {
	return addAlignments(m, o)
}
func (m mixedAlignment) Equal(o AlignmentGeometry) bool {
	return o != nil && alignParts(m) == o.alignParts()
}
func (m mixedAlignment) String() string { return alignParts(m).String() }

// Resolve implements AlignmentGeometry.
//
// Flutter: painting/alignment.dart _MixedAlignment.resolve
func (m mixedAlignment) Resolve(direction ui.TextDirection) Alignment {
	debugCheckCanResolveTextDirection(direction, "_MixedAlignment")
	if direction == ui.TextDirectionRTL {
		return Alignment{m.x - m.start, m.y}
	}
	return Alignment{m.x + m.start, m.y}
}

func addAlignments(a, b AlignmentGeometry) AlignmentGeometry {
	x, y := a.alignParts(), b.alignParts()
	return mixedAlignment{x.x + y.x, x.start + y.start, x.y + y.y}
}

func mapAlignment(g AlignmentGeometry, f func(float64) float64) AlignmentGeometry {
	switch v := g.(type) {
	case Alignment:
		return Alignment{f(v.X), f(v.Y)}
	case AlignmentDirectional:
		return AlignmentDirectional{f(v.Start), f(v.Y)}
	}
	p := g.alignParts()
	return mixedAlignment{f(p.x), f(p.start), f(p.y)}
}

// AlignmentGeometryNeg negates every component.
//
// Flutter: painting/alignment.dart AlignmentGeometry.operator -
func AlignmentGeometryNeg(g AlignmentGeometry) AlignmentGeometry {
	return mapAlignment(g, func(v float64) float64 { return -v })
}

// AlignmentGeometryMul multiplies every component by k.
//
// Flutter: painting/alignment.dart AlignmentGeometry.operator *
func AlignmentGeometryMul(g AlignmentGeometry, k float64) AlignmentGeometry {
	return mapAlignment(g, func(v float64) float64 { return v * k })
}

// AlignmentGeometryDiv divides every component by k.
//
// Flutter: painting/alignment.dart AlignmentGeometry.operator /
func AlignmentGeometryDiv(g AlignmentGeometry, k float64) AlignmentGeometry {
	return mapAlignment(g, func(v float64) float64 { return v / k })
}

// AlignmentGeometryTruncDiv integer-divides every component by k.
//
// Flutter: painting/alignment.dart AlignmentGeometry.operator ~/
func AlignmentGeometryTruncDiv(g AlignmentGeometry, k float64) AlignmentGeometry {
	return mapAlignment(g, func(v float64) float64 { return truncDiv(v, k) })
}

// AlignmentGeometryMod takes the modulo k of every component.
//
// Flutter: painting/alignment.dart AlignmentGeometry.operator %
func AlignmentGeometryMod(g AlignmentGeometry, k float64) AlignmentGeometry {
	return mapAlignment(g, func(v float64) float64 { return dartMod(v, k) })
}

// AlignmentGeometryLerp linearly interpolates between two AlignmentGeometry
// objects. Either may be nil, which counts as zero.
//
// Flutter: painting/alignment.dart AlignmentGeometry.lerp
func AlignmentGeometryLerp(a, b AlignmentGeometry, t float64) AlignmentGeometry {
	if a == nil && b == nil {
		return nil
	}
	if a != nil && b != nil && a.Equal(b) && fmt.Sprintf("%T", a) == fmt.Sprintf("%T", b) {
		return a
	}
	if a == nil {
		return AlignmentGeometryMul(b, t)
	}
	if b == nil {
		return AlignmentGeometryMul(a, 1.0-t)
	}
	if x, ok := a.(Alignment); ok {
		if y, ok := b.(Alignment); ok {
			return AlignmentLerp(&x, &y, t)
		}
	}
	if x, ok := a.(AlignmentDirectional); ok {
		if y, ok := b.(AlignmentDirectional); ok {
			return AlignmentDirectionalLerp(&x, &y, t)
		}
	}
	x, y := a.alignParts(), b.alignParts()
	return mixedAlignment{ui.LerpDouble(x.x, y.x, t), ui.LerpDouble(x.start, y.start, t), ui.LerpDouble(x.y, y.y, t)}
}

// Alignment is a point within a rectangle: (-1.0, -1.0) represents the top left
// of the rectangle, (1.0, 1.0) the bottom right.
//
// Flutter: painting/alignment.dart Alignment
type Alignment struct {
	// X is the distance fraction in the horizontal direction.
	X float64
	// Y is the distance fraction in the vertical direction.
	Y float64
}

// The nine alignments of a rectangle.
var (
	// AlignmentTopLeft is the top left corner.
	//
	// Flutter: painting/alignment.dart Alignment.topLeft
	AlignmentTopLeft = Alignment{-1, -1}
	// AlignmentTopCenter is the center point along the top edge.
	//
	// Flutter: painting/alignment.dart Alignment.topCenter
	AlignmentTopCenter = Alignment{0, -1}
	// AlignmentTopRight is the top right corner.
	//
	// Flutter: painting/alignment.dart Alignment.topRight
	AlignmentTopRight = Alignment{1, -1}
	// AlignmentCenterLeft is the center point along the left edge.
	//
	// Flutter: painting/alignment.dart Alignment.centerLeft
	AlignmentCenterLeft = Alignment{-1, 0}
	// AlignmentCenter is the center point, both horizontally and vertically.
	//
	// Flutter: painting/alignment.dart Alignment.center
	AlignmentCenter = Alignment{0, 0}
	// AlignmentCenterRight is the center point along the right edge.
	//
	// Flutter: painting/alignment.dart Alignment.centerRight
	AlignmentCenterRight = Alignment{1, 0}
	// AlignmentBottomLeft is the bottom left corner.
	//
	// Flutter: painting/alignment.dart Alignment.bottomLeft
	AlignmentBottomLeft = Alignment{-1, 1}
	// AlignmentBottomCenter is the center point along the bottom edge.
	//
	// Flutter: painting/alignment.dart Alignment.bottomCenter
	AlignmentBottomCenter = Alignment{0, 1}
	// AlignmentBottomRight is the bottom right corner.
	//
	// Flutter: painting/alignment.dart Alignment.bottomRight
	AlignmentBottomRight = Alignment{1, 1}
)

func (a Alignment) alignParts() alignParts { return alignParts{x: a.X, y: a.Y} }

// Add returns the sum of the components of this and other.
//
// Flutter: painting/alignment.dart Alignment.add
func (a Alignment) Add(other AlignmentGeometry) AlignmentGeometry {
	if o, ok := other.(Alignment); ok {
		return a.Plus(o)
	}
	return addAlignments(a, other)
}

// Minus returns the difference of the two alignments.
//
// Flutter: painting/alignment.dart Alignment.operator -
func (a Alignment) Minus(o Alignment) Alignment { return Alignment{a.X - o.X, a.Y - o.Y} }

// Plus returns the sum of the two alignments.
//
// Flutter: painting/alignment.dart Alignment.operator +
func (a Alignment) Plus(o Alignment) Alignment { return Alignment{a.X + o.X, a.Y + o.Y} }

// Neg returns the alignment with both components negated.
//
// Flutter: painting/alignment.dart Alignment.unary-
func (a Alignment) Neg() Alignment { return Alignment{-a.X, -a.Y} }

// Mul scales both components by k.
//
// Flutter: painting/alignment.dart Alignment.operator *
func (a Alignment) Mul(k float64) Alignment { return Alignment{a.X * k, a.Y * k} }

// Div divides both components by k.
//
// Flutter: painting/alignment.dart Alignment.operator /
func (a Alignment) Div(k float64) Alignment { return Alignment{a.X / k, a.Y / k} }

// TruncDiv integer-divides both components by k.
//
// Flutter: painting/alignment.dart Alignment.operator ~/
func (a Alignment) TruncDiv(k float64) Alignment {
	return Alignment{truncDiv(a.X, k), truncDiv(a.Y, k)}
}

// Mod takes the modulo k of both components.
//
// Flutter: painting/alignment.dart Alignment.operator %
func (a Alignment) Mod(k float64) Alignment { return Alignment{dartMod(a.X, k), dartMod(a.Y, k)} }

// AlongOffset returns the offset that is this fraction in the distance from
// the top left up to the bottom right of the offset.
//
// Flutter: painting/alignment.dart Alignment.alongOffset
func (a Alignment) AlongOffset(other ui.Offset) ui.Offset {
	cx, cy := other.Dx/2.0, other.Dy/2.0
	return ui.Offset{Dx: cx + a.X*cx, Dy: cy + a.Y*cy}
}

// AlongSize returns the offset that is this fraction in the distance from the
// top left up to the bottom right of the size.
//
// Flutter: painting/alignment.dart Alignment.alongSize
func (a Alignment) AlongSize(other ui.Size) ui.Offset {
	cx, cy := other.Width/2.0, other.Height/2.0
	return ui.Offset{Dx: cx + a.X*cx, Dy: cy + a.Y*cy}
}

// WithinRect returns the point that is this fraction in the distance from the
// top left up to the bottom right of the rect.
//
// Flutter: painting/alignment.dart Alignment.withinRect
func (a Alignment) WithinRect(rect ui.Rect) ui.Offset {
	hw, hh := rect.Width()/2.0, rect.Height()/2.0
	return ui.Offset{Dx: rect.Left + hw + a.X*hw, Dy: rect.Top + hh + a.Y*hh}
}

// Inscribe returns a rect of the given size, aligned within the rect.
//
// Flutter: painting/alignment.dart Alignment.inscribe
func (a Alignment) Inscribe(size ui.Size, rect ui.Rect) ui.Rect {
	hwd := (rect.Width() - size.Width) / 2.0
	hhd := (rect.Height() - size.Height) / 2.0
	return ui.NewRectFromLTWH(rect.Left+hwd+a.X*hwd, rect.Top+hhd+a.Y*hhd, size.Width, size.Height)
}

// Resolve returns the alignment itself: it does not depend on the direction.
//
// Flutter: painting/alignment.dart Alignment.resolve
func (a Alignment) Resolve(ui.TextDirection) Alignment { return a }

// Equal reports whether the three components are equal.
//
// Flutter: painting/alignment.dart Alignment.operator ==
func (a Alignment) Equal(o AlignmentGeometry) bool {
	return o != nil && a.alignParts() == o.alignParts()
}

// String describes the alignment, such as "Alignment.topLeft".
//
// Flutter: painting/alignment.dart Alignment.toString
func (a Alignment) String() string { return stringifyAlignment(a.X, a.Y) }

// AlignmentLerp linearly interpolates between two Alignments. Either may be nil,
// which counts as the center.
//
// Flutter: painting/alignment.dart Alignment.lerp
func AlignmentLerp(a, b *Alignment, t float64) *Alignment {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		return &Alignment{ui.LerpDouble(0, b.X, t), ui.LerpDouble(0, b.Y, t)}
	}
	if b == nil {
		return &Alignment{ui.LerpDouble(a.X, 0, t), ui.LerpDouble(a.Y, 0, t)}
	}
	if *a == *b {
		return a
	}
	return &Alignment{ui.LerpDouble(a.X, b.X, t), ui.LerpDouble(a.Y, b.Y, t)}
}

// AlignmentDirectional is an offset that's expressed as a fraction of a Size,
// but whose horizontal component is dependent on the writing direction.
//
// Flutter: painting/alignment.dart AlignmentDirectional
type AlignmentDirectional struct {
	// Start is the distance fraction in the horizontal direction.
	Start float64
	// Y is the distance fraction in the vertical direction.
	Y float64
}

// The nine directional alignments of a rectangle.
var (
	// AlignmentDirectionalTopStart is the top corner on the start side.
	//
	// Flutter: painting/alignment.dart AlignmentDirectional.topStart
	AlignmentDirectionalTopStart = AlignmentDirectional{-1, -1}
	// AlignmentDirectionalTopCenter is the center point along the top edge.
	//
	// Flutter: painting/alignment.dart AlignmentDirectional.topCenter
	AlignmentDirectionalTopCenter = AlignmentDirectional{0, -1}
	// AlignmentDirectionalTopEnd is the top corner on the end side.
	//
	// Flutter: painting/alignment.dart AlignmentDirectional.topEnd
	AlignmentDirectionalTopEnd = AlignmentDirectional{1, -1}
	// AlignmentDirectionalCenterStart is the center point along the start edge.
	//
	// Flutter: painting/alignment.dart AlignmentDirectional.centerStart
	AlignmentDirectionalCenterStart = AlignmentDirectional{-1, 0}
	// AlignmentDirectionalCenter is the center point.
	//
	// Flutter: painting/alignment.dart AlignmentDirectional.center
	AlignmentDirectionalCenter = AlignmentDirectional{0, 0}
	// AlignmentDirectionalCenterEnd is the center point along the end edge.
	//
	// Flutter: painting/alignment.dart AlignmentDirectional.centerEnd
	AlignmentDirectionalCenterEnd = AlignmentDirectional{1, 0}
	// AlignmentDirectionalBottomStart is the bottom corner on the start side.
	//
	// Flutter: painting/alignment.dart AlignmentDirectional.bottomStart
	AlignmentDirectionalBottomStart = AlignmentDirectional{-1, 1}
	// AlignmentDirectionalBottomCenter is the center point along the bottom edge.
	//
	// Flutter: painting/alignment.dart AlignmentDirectional.bottomCenter
	AlignmentDirectionalBottomCenter = AlignmentDirectional{0, 1}
	// AlignmentDirectionalBottomEnd is the bottom corner on the end side.
	//
	// Flutter: painting/alignment.dart AlignmentDirectional.bottomEnd
	AlignmentDirectionalBottomEnd = AlignmentDirectional{1, 1}
)

func (a AlignmentDirectional) alignParts() alignParts { return alignParts{start: a.Start, y: a.Y} }

// Add returns the sum of the components of this and other.
//
// Flutter: painting/alignment.dart AlignmentDirectional.add
func (a AlignmentDirectional) Add(other AlignmentGeometry) AlignmentGeometry {
	if o, ok := other.(AlignmentDirectional); ok {
		return a.Plus(o)
	}
	return addAlignments(a, other)
}

// Minus returns the difference of the two alignments.
//
// Flutter: painting/alignment.dart AlignmentDirectional.operator -
func (a AlignmentDirectional) Minus(o AlignmentDirectional) AlignmentDirectional {
	return AlignmentDirectional{a.Start - o.Start, a.Y - o.Y}
}

// Plus returns the sum of the two alignments.
//
// Flutter: painting/alignment.dart AlignmentDirectional.operator +
func (a AlignmentDirectional) Plus(o AlignmentDirectional) AlignmentDirectional {
	return AlignmentDirectional{a.Start + o.Start, a.Y + o.Y}
}

// Neg returns the alignment with both components negated.
//
// Flutter: painting/alignment.dart AlignmentDirectional.unary-
func (a AlignmentDirectional) Neg() AlignmentDirectional { return AlignmentDirectional{-a.Start, -a.Y} }

// Mul scales both components by k.
//
// Flutter: painting/alignment.dart AlignmentDirectional.operator *
func (a AlignmentDirectional) Mul(k float64) AlignmentDirectional {
	return AlignmentDirectional{a.Start * k, a.Y * k}
}

// Div divides both components by k.
//
// Flutter: painting/alignment.dart AlignmentDirectional.operator /
func (a AlignmentDirectional) Div(k float64) AlignmentDirectional {
	return AlignmentDirectional{a.Start / k, a.Y / k}
}

// TruncDiv integer-divides both components by k.
//
// Flutter: painting/alignment.dart AlignmentDirectional.operator ~/
func (a AlignmentDirectional) TruncDiv(k float64) AlignmentDirectional {
	return AlignmentDirectional{truncDiv(a.Start, k), truncDiv(a.Y, k)}
}

// Mod takes the modulo k of both components.
//
// Flutter: painting/alignment.dart AlignmentDirectional.operator %
func (a AlignmentDirectional) Mod(k float64) AlignmentDirectional {
	return AlignmentDirectional{dartMod(a.Start, k), dartMod(a.Y, k)}
}

// Resolve returns an Alignment for the text direction.
//
// Flutter: painting/alignment.dart AlignmentDirectional.resolve
func (a AlignmentDirectional) Resolve(direction ui.TextDirection) Alignment {
	debugCheckCanResolveTextDirection(direction, "AlignmentDirectional")
	if direction == ui.TextDirectionRTL {
		return Alignment{-a.Start, a.Y}
	}
	return Alignment{a.Start, a.Y}
}

// Equal reports whether the three components are equal.
//
// Flutter: painting/alignment.dart AlignmentDirectional.operator ==
func (a AlignmentDirectional) Equal(o AlignmentGeometry) bool {
	return o != nil && a.alignParts() == o.alignParts()
}

// String describes the alignment, such as "AlignmentDirectional.topStart".
//
// Flutter: painting/alignment.dart AlignmentDirectional.toString
func (a AlignmentDirectional) String() string { return stringifyAlignmentDirectional(a.Start, a.Y) }

// AlignmentDirectionalLerp linearly interpolates between two
// AlignmentDirectionals. Either may be nil, which counts as the center.
//
// Flutter: painting/alignment.dart AlignmentDirectional.lerp
func AlignmentDirectionalLerp(a, b *AlignmentDirectional, t float64) *AlignmentDirectional {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		return &AlignmentDirectional{ui.LerpDouble(0, b.Start, t), ui.LerpDouble(0, b.Y, t)}
	}
	if b == nil {
		return &AlignmentDirectional{ui.LerpDouble(a.Start, 0, t), ui.LerpDouble(a.Y, 0, t)}
	}
	if *a == *b {
		return a
	}
	return &AlignmentDirectional{ui.LerpDouble(a.Start, b.Start, t), ui.LerpDouble(a.Y, b.Y, t)}
}

// TextAlignVertical is the vertical alignment of text within an input box: the
// y of the alignment from -1.0 (top) to 1.0 (bottom).
//
// Flutter: painting/alignment.dart TextAlignVertical
type TextAlignVertical struct {
	// Y is the vertical alignment, between -1.0 and 1.0.
	Y float64
}

// NewTextAlignVertical creates the alignment. In debug builds a y outside of
// -1.0 to 1.0 panics with the tag TextAlignVerticalOutOfRange.
//
// Flutter: painting/alignment.dart TextAlignVertical.new
func NewTextAlignVertical(y float64) TextAlignVertical {
	if foundation.DebugMode && !(y >= -1 && y <= 1) {
		panic(errors.New("TextAlignVerticalOutOfRange", "y must be between -1.0 and 1.0", "y", y))
	}
	return TextAlignVertical{y}
}

// The three standard vertical alignments.
var (
	// TextAlignVerticalTop aligns to the top.
	//
	// Flutter: painting/alignment.dart TextAlignVertical.top
	TextAlignVerticalTop = TextAlignVertical{-1}
	// TextAlignVerticalCenter aligns to the center.
	//
	// Flutter: painting/alignment.dart TextAlignVertical.center
	TextAlignVerticalCenter = TextAlignVertical{0}
	// TextAlignVerticalBottom aligns to the bottom.
	//
	// Flutter: painting/alignment.dart TextAlignVertical.bottom
	TextAlignVerticalBottom = TextAlignVertical{1}
)

// String describes the alignment, such as "TextAlignVertical(y: 0.0)".
//
// Flutter: painting/alignment.dart TextAlignVertical.toString
func (a TextAlignVertical) String() string {
	return "TextAlignVertical(y: " + dartfmt.Double(a.Y) + ")"
}
