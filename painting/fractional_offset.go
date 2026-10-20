// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// FractionalOffset is an offset that's expressed as a fraction of a Size: (0, 0)
// is the top left, (1, 1) the bottom right. It is an Alignment, whose x and y
// run from -1 to 1, so it embeds one.
//
// Flutter: painting/fractional_offset.dart FractionalOffset
type FractionalOffset struct{ Alignment }

// NewFractionalOffset creates a fractional offset from the fractions of the
// width and the height.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.new
func NewFractionalOffset(dx, dy float64) FractionalOffset {
	return FractionalOffset{Alignment{dx*2.0 - 1.0, dy*2.0 - 1.0}}
}

// NewFractionalOffsetFromOffsetAndSize returns the fractional offset of the
// offset in a rectangle of the size, with its top left at the origin.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.fromOffsetAndSize
func NewFractionalOffsetFromOffsetAndSize(offset ui.Offset, size ui.Size) FractionalOffset {
	return NewFractionalOffset(offset.Dx/size.Width, offset.Dy/size.Height)
}

// NewFractionalOffsetFromOffsetAndRect returns the fractional offset of the
// offset in the rectangle.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.fromOffsetAndRect
func NewFractionalOffsetFromOffsetAndRect(offset ui.Offset, rect ui.Rect) FractionalOffset {
	return NewFractionalOffsetFromOffsetAndSize(offset.Sub(rect.TopLeft()), rect.Size())
}

// DX is the distance fraction in the horizontal direction.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.dx
func (f FractionalOffset) DX() float64 { return (f.X + 1.0) / 2.0 }

// DY is the distance fraction in the vertical direction.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.dy
func (f FractionalOffset) DY() float64 { return (f.Y + 1.0) / 2.0 }

// The named fractional offsets.
var (
	// FractionalOffsetTopLeft is the top left corner.
	//
	// Flutter: painting/fractional_offset.dart FractionalOffset.topLeft
	FractionalOffsetTopLeft = NewFractionalOffset(0.0, 0.0)
	// FractionalOffsetTopCenter is the center point along the top edge.
	//
	// Flutter: painting/fractional_offset.dart FractionalOffset.topCenter
	FractionalOffsetTopCenter = NewFractionalOffset(0.5, 0.0)
	// FractionalOffsetTopRight is the top right corner.
	//
	// Flutter: painting/fractional_offset.dart FractionalOffset.topRight
	FractionalOffsetTopRight = NewFractionalOffset(1.0, 0.0)
	// FractionalOffsetCenterLeft is the center point along the left edge.
	//
	// Flutter: painting/fractional_offset.dart FractionalOffset.centerLeft
	FractionalOffsetCenterLeft = NewFractionalOffset(0.0, 0.5)
	// FractionalOffsetCenter is the center point, both horizontally and
	// vertically.
	//
	// Flutter: painting/fractional_offset.dart FractionalOffset.center
	FractionalOffsetCenter = NewFractionalOffset(0.5, 0.5)
	// FractionalOffsetCenterRight is the center point along the right edge.
	//
	// Flutter: painting/fractional_offset.dart FractionalOffset.centerRight
	FractionalOffsetCenterRight = NewFractionalOffset(1.0, 0.5)
	// FractionalOffsetBottomLeft is the bottom left corner.
	//
	// Flutter: painting/fractional_offset.dart FractionalOffset.bottomLeft
	FractionalOffsetBottomLeft = NewFractionalOffset(0.0, 1.0)
	// FractionalOffsetBottomCenter is the center point along the bottom edge.
	//
	// Flutter: painting/fractional_offset.dart FractionalOffset.bottomCenter
	FractionalOffsetBottomCenter = NewFractionalOffset(0.5, 1.0)
	// FractionalOffsetBottomRight is the bottom right corner.
	//
	// Flutter: painting/fractional_offset.dart FractionalOffset.bottomRight
	FractionalOffsetBottomRight = NewFractionalOffset(1.0, 1.0)
)

// Add returns the sum of the components of this and other, as a FractionalOffset
// if other is one.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.operator +
func (f FractionalOffset) Add(other AlignmentGeometry) AlignmentGeometry {
	if o, ok := other.(FractionalOffset); ok {
		return f.Plus(o)
	}
	return f.Alignment.Add(other)
}

// Plus is the sum of two fractional offsets.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.operator +
func (f FractionalOffset) Plus(o FractionalOffset) FractionalOffset {
	return NewFractionalOffset(f.DX()+o.DX(), f.DY()+o.DY())
}

// Minus is the difference of two fractional offsets.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.operator -
func (f FractionalOffset) Minus(o FractionalOffset) FractionalOffset {
	return NewFractionalOffset(f.DX()-o.DX(), f.DY()-o.DY())
}

// Neg negates both fractions.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.operator unary-
func (f FractionalOffset) Neg() FractionalOffset { return NewFractionalOffset(-f.DX(), -f.DY()) }

// Mul scales both fractions by k.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.operator *
func (f FractionalOffset) Mul(k float64) FractionalOffset {
	return NewFractionalOffset(f.DX()*k, f.DY()*k)
}

// Div divides both fractions by k.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.operator /
func (f FractionalOffset) Div(k float64) FractionalOffset {
	return NewFractionalOffset(f.DX()/k, f.DY()/k)
}

// TruncDiv integer-divides both fractions by k.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.operator ~/
func (f FractionalOffset) TruncDiv(k float64) FractionalOffset {
	return NewFractionalOffset(truncDiv(f.DX(), k), truncDiv(f.DY(), k))
}

// Mod takes the modulo k of both fractions.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.operator %
func (f FractionalOffset) Mod(k float64) FractionalOffset {
	return NewFractionalOffset(dartMod(f.DX(), k), dartMod(f.DY(), k))
}

// FractionalOffsetLerp linearly interpolates between two fractional offsets,
// either of which may be nil, which stands for the center.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.lerp
func FractionalOffsetLerp(a, b *FractionalOffset, t float64) *FractionalOffset {
	if a == b {
		return a
	}
	var r FractionalOffset
	switch {
	case a == nil:
		r = NewFractionalOffset(ui.LerpDouble(0.5, b.DX(), t), ui.LerpDouble(0.5, b.DY(), t))
	case b == nil:
		r = NewFractionalOffset(ui.LerpDouble(a.DX(), 0.5, t), ui.LerpDouble(a.DY(), 0.5, t))
	default:
		r = NewFractionalOffset(ui.LerpDouble(a.DX(), b.DX(), t), ui.LerpDouble(a.DY(), b.DY(), t))
	}
	return &r
}

// String describes the offset.
//
// Flutter: painting/fractional_offset.dart FractionalOffset.toString
func (f FractionalOffset) String() string {
	return "FractionalOffset(" + dartfmt.ToStringAsFixed(f.DX(), 1) + ", " + dartfmt.ToStringAsFixed(f.DY(), 1) + ")"
}
