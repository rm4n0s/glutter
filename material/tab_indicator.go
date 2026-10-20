// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// UnderlineTabIndicatorOpts are the optional parameters of
// NewUnderlineTabIndicator.
//
// Flutter: material/tab_indicator.dart UnderlineTabIndicator.new
type UnderlineTabIndicatorOpts struct {
	// BorderRadius is the radius of the corners of the indicator, if any.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// BorderSide is the color and weight of the horizontal line drawn below the
	// selected tab; the default is a white line of 2 pixels.
	BorderSide foundation.Opt[painting.BorderSide]
	// Insets are the horizontal insets of the line.
	Insets painting.EdgeInsetsGeometry
}

// UnderlineTabIndicator is an indicator for the selected tab of a TabBar: a line
// below the tab.
//
// Flutter: material/tab_indicator.dart UnderlineTabIndicator
type UnderlineTabIndicator struct {
	painting.DecorationDefaults
	borderRadius foundation.Opt[painting.BorderRadius]
	borderSide   painting.BorderSide
	insets       painting.EdgeInsetsGeometry
}

// NewUnderlineTabIndicator creates the indicator.
//
// Flutter: material/tab_indicator.dart UnderlineTabIndicator.new
func NewUnderlineTabIndicator(opts ...UnderlineTabIndicatorOpts) *UnderlineTabIndicator {
	o := firstOpts(opts)
	side := o.BorderSide.OrElse(painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(2.0), Color: foundation.Some(Colors.White)}))
	insets := o.Insets
	if insets == nil {
		insets = painting.EdgeInsetsZero
	}
	return &UnderlineTabIndicator{borderRadius: o.BorderRadius, borderSide: side, insets: insets}
}

// BorderRadius is the radius of the corners of the indicator, if any.
//
// Flutter: material/tab_indicator.dart UnderlineTabIndicator.borderRadius
func (d *UnderlineTabIndicator) BorderRadius() foundation.Opt[painting.BorderRadius] {
	return d.borderRadius
}

// BorderSide is the color and weight of the line.
//
// Flutter: material/tab_indicator.dart UnderlineTabIndicator.borderSide
func (d *UnderlineTabIndicator) BorderSide() painting.BorderSide { return d.borderSide }

// Insets are the horizontal insets of the line.
//
// Flutter: material/tab_indicator.dart UnderlineTabIndicator.insets
func (d *UnderlineTabIndicator) Insets() painting.EdgeInsetsGeometry { return d.insets }

// LerpFrom interpolates from another underline indicator.
//
// Flutter: material/tab_indicator.dart UnderlineTabIndicator.lerpFrom
func (d *UnderlineTabIndicator) LerpFrom(a painting.Decoration, t float64) painting.Decoration {
	if a, ok := a.(*UnderlineTabIndicator); ok {
		return &UnderlineTabIndicator{
			borderSide: painting.BorderSideLerp(a.borderSide, d.borderSide, t),
			insets:     painting.EdgeInsetsGeometryLerp(a.insets, d.insets, t),
		}
	}
	return d.DecorationDefaults.LerpFrom(a, t)
}

// LerpTo interpolates to another underline indicator.
//
// Flutter: material/tab_indicator.dart UnderlineTabIndicator.lerpTo
func (d *UnderlineTabIndicator) LerpTo(b painting.Decoration, t float64) painting.Decoration {
	if b, ok := b.(*UnderlineTabIndicator); ok {
		return &UnderlineTabIndicator{
			borderSide: painting.BorderSideLerp(d.borderSide, b.borderSide, t),
			insets:     painting.EdgeInsetsGeometryLerp(d.insets, b.insets, t),
		}
	}
	return d.DecorationDefaults.LerpTo(b, t)
}

// CreateBoxPainter creates the painter of the line.
//
// Flutter: material/tab_indicator.dart UnderlineTabIndicator.createBoxPainter
func (d *UnderlineTabIndicator) CreateBoxPainter(func()) painting.BoxPainter {
	return &underlinePainter{decoration: d}
}

func (d *UnderlineTabIndicator) indicatorRectFor(rect ui.Rect, textDirection ui.TextDirection) ui.Rect {
	indicator := d.insets.Resolve(textDirection).DeflateRect(rect)
	return ui.NewRectFromLTWH(indicator.Left, indicator.Bottom-d.borderSide.Width, indicator.Width(), d.borderSide.Width)
}

// GetClipPath is the outline of the line.
//
// Flutter: material/tab_indicator.dart UnderlineTabIndicator.getClipPath
func (d *UnderlineTabIndicator) GetClipPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	if r, ok := d.borderRadius.Get(); ok {
		p.AddRRect(r.ToRRect(d.indicatorRectFor(rect, textDirection)))
	} else {
		p.AddRect(d.indicatorRectFor(rect, textDirection))
	}
	return p
}

// ToStringShort names the decoration.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (d *UnderlineTabIndicator) ToStringShort() string { return "UnderlineTabIndicator" }

// String describes the decoration.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d *UnderlineTabIndicator) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the indicator.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.debugFillProperties
func (d *UnderlineTabIndicator) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("borderRadius", optValue(d.borderRadius)))
	p.Add(objectProp("borderSide", d.borderSide))
	p.Add(objectProp("insets", d.insets))
}

// Equal reports whether the indicators are equal.
//
// Flutter: material/tab_indicator.dart UnderlineTabIndicator.==
func (d *UnderlineTabIndicator) Equal(o *UnderlineTabIndicator) bool {
	return d.borderRadius == o.borderRadius && d.borderSide == o.borderSide && foundation.ValuesEqual(d.insets, o.insets)
}

// underlinePainter paints the line of an UnderlineTabIndicator.
//
// Flutter: material/tab_indicator.dart _UnderlinePainter
type underlinePainter struct{ decoration *UnderlineTabIndicator }

func (p *underlinePainter) Dispose() {}

func (p *underlinePainter) Paint(canvas *ui.Canvas, offset ui.Offset, configuration painting.ImageConfiguration) {
	if foundation.DebugMode && configuration.Size.IsNone() {
		panic(errors.New("UnderlineTabIndicatorSize", "the configuration to paint an UnderlineTabIndicator needs a size"))
	}
	rect := offset.And(configuration.Size.Value())
	d := p.decoration
	textDirection := configuration.TextDirection
	if r, ok := d.borderRadius.Get(); ok {
		paint := ui.NewPaint()
		paint.SetColor(d.borderSide.Color)
		indicator := d.indicatorRectFor(rect, textDirection)
		rrect := ui.NewRRectFromRectAndCorners(indicator, ui.Corners{TopLeft: r.TopLeft, TopRight: r.TopRight, BottomRight: r.BottomRight, BottomLeft: r.BottomLeft})
		canvas.DrawRRect(rrect, paint)
		return
	}
	paint := d.borderSide.ToPaint()
	paint.SetStrokeCap(ui.StrokeCapSquare)
	indicator := d.indicatorRectFor(rect, textDirection).Deflate(d.borderSide.Width / 2.0)
	canvas.DrawLine(indicator.BottomLeft(), indicator.BottomRight(), paint)
}
