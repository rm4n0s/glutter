// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// TableBorder is the border of a table: the sides of the table and the lines
// between its rows and columns.
//
// Flutter: rendering/table_border.dart TableBorder
type TableBorder struct {
	// Top is the top side of this border.
	//
	// Flutter: rendering/table_border.dart TableBorder.top
	Top painting.BorderSide
	// Right is the right side of this border.
	//
	// Flutter: rendering/table_border.dart TableBorder.right
	Right painting.BorderSide
	// Bottom is the bottom side of this border.
	//
	// Flutter: rendering/table_border.dart TableBorder.bottom
	Bottom painting.BorderSide
	// Left is the left side of this border.
	//
	// Flutter: rendering/table_border.dart TableBorder.left
	Left painting.BorderSide
	// HorizontalInside is the horizontal interior lines of this border.
	//
	// Flutter: rendering/table_border.dart TableBorder.horizontalInside
	HorizontalInside painting.BorderSide
	// VerticalInside is the vertical interior lines of this border.
	//
	// Flutter: rendering/table_border.dart TableBorder.verticalInside
	VerticalInside painting.BorderSide
	// BorderRadius is the radius of the corners of the table's outer border.
	//
	// Flutter: rendering/table_border.dart TableBorder.borderRadius
	BorderRadius painting.BorderRadius
}

// normSide is BorderSideNone for the zero BorderSide, which a struct literal
// leaves in the sides it does not set.
func normSide(s painting.BorderSide) painting.BorderSide {
	if s == (painting.BorderSide{}) {
		return painting.BorderSideNone
	}
	return s
}

// NewTableBorder creates a border; the zero sides of the literal are
// BorderSideNone.
//
// Flutter: rendering/table_border.dart TableBorder.new
func NewTableBorder(b TableBorder) TableBorder { return b.normalized() }

func (b TableBorder) normalized() TableBorder {
	b.Top, b.Right, b.Bottom, b.Left = normSide(b.Top), normSide(b.Right), normSide(b.Bottom), normSide(b.Left)
	b.HorizontalInside, b.VerticalInside = normSide(b.HorizontalInside), normSide(b.VerticalInside)
	return b
}

// TableBorderAllOpts are the optional parameters of NewTableBorderAll.
//
// Flutter: rendering/table_border.dart TableBorder.all
type TableBorderAllOpts struct {
	// Color defaults to black.
	Color foundation.Opt[ui.Color]
	// Width defaults to 1.
	Width foundation.Opt[float64]
	// Style defaults to painting.BorderStyleSolid.
	Style painting.BorderStyle
	// BorderRadius defaults to painting.BorderRadiusZero.
	BorderRadius painting.BorderRadius
}

// NewTableBorderAll creates a border that consists of a solid line of the same
// color and width on all sides.
//
// Flutter: rendering/table_border.dart TableBorder.all
func NewTableBorderAll(opts ...TableBorderAllOpts) TableBorder {
	o := firstOpts(opts)
	side := painting.NewBorderSide(painting.BorderSideOpts{
		Color: foundation.Some(o.Color.OrElse(ui.NewColor(0xFF000000))), Width: foundation.Some(o.Width.OrElse(1)), Style: o.Style,
	})
	return TableBorder{Top: side, Right: side, Bottom: side, Left: side, HorizontalInside: side, VerticalInside: side,
		BorderRadius: o.BorderRadius}
}

// NewTableBorderSymmetric creates a border with the same outside sides and the
// same inside lines.
//
// Flutter: rendering/table_border.dart TableBorder.symmetric
func NewTableBorderSymmetric(inside, outside painting.BorderSide, borderRadius painting.BorderRadius) TableBorder {
	return TableBorder{Top: outside, Right: outside, Bottom: outside, Left: outside, HorizontalInside: inside,
		VerticalInside: inside, BorderRadius: borderRadius}.normalized()
}

// Dimensions is the widths of the outer sides as insets.
//
// Flutter: rendering/table_border.dart TableBorder.dimensions
func (b TableBorder) Dimensions() painting.EdgeInsets {
	b = b.normalized()
	return painting.NewEdgeInsetsFromLTRB(b.Left.Width, b.Top.Width, b.Right.Width, b.Bottom.Width)
}

// IsUniform reports whether all the sides, inside and outside, have the same
// color, width and style.
//
// Flutter: rendering/table_border.dart TableBorder.isUniform
func (b TableBorder) IsUniform() bool {
	b = b.normalized()
	sides := []painting.BorderSide{b.Right, b.Bottom, b.Left, b.HorizontalInside, b.VerticalInside}
	for _, s := range sides {
		if s.Color != b.Top.Color || s.Width != b.Top.Width || s.Style != b.Top.Style {
			return false
		}
	}
	return true
}

func (b TableBorder) outerBorderIsUniform() bool {
	for _, s := range []painting.BorderSide{b.Right, b.Bottom, b.Left} {
		if s.Color != b.Top.Color || s.Width != b.Top.Width || s.Style != b.Top.Style {
			return false
		}
	}
	return true
}

func (b TableBorder) distinctVisibleOuterColors() []ui.Color {
	var colors []ui.Color
	for _, s := range []painting.BorderSide{b.Top, b.Right, b.Bottom, b.Left} {
		if s.Style == painting.BorderStyleNone {
			continue
		}
		seen := false
		for _, c := range colors {
			seen = seen || c == s.Color
		}
		if !seen {
			colors = append(colors, s.Color)
		}
	}
	return colors
}

func (b TableBorder) paintTableBorder(canvas *ui.Canvas, rect ui.Rect) {
	hasRadius := !b.BorderRadius.Equal(painting.BorderRadiusZero)
	if b.outerBorderIsUniform() && hasRadius {
		outer := b.BorderRadius.ToRRect(rect)
		inner := outer.Deflate(b.Top.Width)
		paint := ui.NewPaint()
		paint.SetColor(b.Top.Color)
		canvas.DrawDRRect(outer, inner, paint)
		return
	}
	if colors := b.distinctVisibleOuterColors(); len(colors) == 1 && hasRadius {
		visible := func(s painting.BorderSide) painting.BorderSide {
			if s.Style == painting.BorderStyleNone {
				return painting.BorderSideNone
			}
			return s
		}
		top, right, bottom, left := visible(b.Top), visible(b.Right), visible(b.Bottom), visible(b.Left)
		borderRect := b.BorderRadius.ToRRect(rect)
		paint := ui.NewPaint()
		paint.SetColor(colors[0])
		inner := painting.NewEdgeInsetsFromLTRB(left.StrokeInset(), top.StrokeInset(), right.StrokeInset(), bottom.StrokeInset()).DeflateRRect(borderRect)
		outer := painting.NewEdgeInsetsFromLTRB(left.StrokeOutset(), top.StrokeOutset(), right.StrokeOutset(), bottom.StrokeOutset()).InflateRRect(borderRect)
		canvas.DrawDRRect(outer, inner, paint)
		return
	}
	painting.PaintBorder(canvas, rect, painting.BorderPaintOpts{Top: b.Top, Right: b.Right, Bottom: b.Bottom, Left: b.Left})
}

// Scale scales the widths of the sides by t. The border radius is not kept.
//
// Flutter: rendering/table_border.dart TableBorder.scale
func (b TableBorder) Scale(t float64) TableBorder {
	b = b.normalized()
	return TableBorder{Top: b.Top.Scale(t), Right: b.Right.Scale(t), Bottom: b.Bottom.Scale(t), Left: b.Left.Scale(t),
		HorizontalInside: b.HorizontalInside.Scale(t), VerticalInside: b.VerticalInside.Scale(t)}
}

// TableBorderLerp linearly interpolates between two table borders. A nil border
// is Flutter's null.
//
// Flutter: rendering/table_border.dart TableBorder.lerp
func TableBorderLerp(a, b *TableBorder, t float64) *TableBorder {
	switch {
	case a == b:
		return a
	case a == nil:
		s := b.Scale(t)
		return &s
	case b == nil:
		s := a.Scale(1 - t)
		return &s
	}
	an, bn := a.normalized(), b.normalized()
	return &TableBorder{
		Top: painting.BorderSideLerp(an.Top, bn.Top, t), Right: painting.BorderSideLerp(an.Right, bn.Right, t),
		Bottom: painting.BorderSideLerp(an.Bottom, bn.Bottom, t), Left: painting.BorderSideLerp(an.Left, bn.Left, t),
		HorizontalInside: painting.BorderSideLerp(an.HorizontalInside, bn.HorizontalInside, t),
		VerticalInside:   painting.BorderSideLerp(an.VerticalInside, bn.VerticalInside, t),
	}
}

// Paint draws the border around the rectangle, and the inside lines at the given
// offsets from the top (rows) and from the left (columns) of the rectangle. In
// debug builds it panics with TableBorderOutOfRange if an offset is outside the
// rectangle.
//
// Flutter: rendering/table_border.dart TableBorder.paint
func (b TableBorder) Paint(canvas *ui.Canvas, rect ui.Rect, rows, columns []float64) {
	b = b.normalized()
	if foundation.DebugMode {
		if len(rows) > 0 && (rows[0] < 0 || rows[len(rows)-1] > rect.Height()) {
			panic(errors.New("TableBorderOutOfRange", "the rows of a table border must be inside the rectangle",
				"rows", rows, "height", rect.Height()))
		}
		if len(columns) > 0 && (columns[0] < 0 || columns[len(columns)-1] > rect.Width()) {
			panic(errors.New("TableBorderOutOfRange", "the columns of a table border must be inside the rectangle",
				"columns", columns, "width", rect.Width()))
		}
	}
	if len(columns) > 0 || len(rows) > 0 {
		paint := ui.NewPaint()
		path := ui.NewPath()
		if len(columns) > 0 && b.VerticalInside.Style == painting.BorderStyleSolid {
			paint.SetColor(b.VerticalInside.Color)
			paint.SetStrokeWidth(b.VerticalInside.Width)
			paint.SetStyle(ui.PaintingStyleStroke)
			path.Reset()
			for _, x := range columns {
				path.MoveTo(rect.Left+x, rect.Top)
				path.LineTo(rect.Left+x, rect.Bottom)
			}
			canvas.DrawPath(path, paint)
		}
		if len(rows) > 0 && b.HorizontalInside.Style == painting.BorderStyleSolid {
			paint.SetColor(b.HorizontalInside.Color)
			paint.SetStrokeWidth(b.HorizontalInside.Width)
			paint.SetStyle(ui.PaintingStyleStroke)
			path.Reset()
			for _, y := range rows {
				path.MoveTo(rect.Left, rect.Top+y)
				path.LineTo(rect.Right, rect.Top+y)
			}
			canvas.DrawPath(path, paint)
		}
	}
	b.paintTableBorder(canvas, rect)
}

// String lists the sides and the border radius.
//
// Flutter: rendering/table_border.dart TableBorder.toString
func (b TableBorder) String() string {
	b = b.normalized()
	return "TableBorder(" + b.Top.String() + ", " + b.Right.String() + ", " + b.Bottom.String() + ", " + b.Left.String() + ", " +
		b.HorizontalInside.String() + ", " + b.VerticalInside.String() + ", " + b.BorderRadius.String() + ")"
}
