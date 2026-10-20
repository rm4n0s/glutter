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
	"github.com/rm4n0s/glutter/widgets"
)

// dividerDefaults are the defaults of the dividers: the values that the theme
// does not give.
//
// Flutter: material/divider.dart _DividerDefaultsM2, _DividerDefaultsM3
func dividerDefaults(bc widgets.BuildContext) (space, thickness, indent, endIndent float64, color ui.Color) {
	theme := ThemeOf(bc)
	if theme.UseMaterial3 {
		return 16, 1, 0, 0, theme.ColorScheme.OutlineVariant()
	}
	return 16, 0, 0, 0, theme.DividerColor
}

// checkDividerSizes panics with the tag when a size is negative.
func checkDividerSizes(prefix string, size, thickness, indent, endIndent foundation.Opt[float64]) {
	if !foundation.DebugMode {
		return
	}
	for _, c := range []struct {
		name string
		v    foundation.Opt[float64]
	}{{"Size", size}, {"Thickness", thickness}, {"Indent", indent}, {"EndIndent", endIndent}} {
		if v, ok := c.v.Get(); ok && v < 0 {
			panic(errors.New(prefix+c.name+"Negative", "a divider's sizes must not be negative", "value", v))
		}
	}
}

// DividerOpts are the optional parameters of NewDivider.
//
// Flutter: material/divider.dart Divider.new
type DividerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Height is the divider's height extent: the space the divider takes, with
	// the line in the middle of it; the default is 16.
	Height foundation.Opt[float64]
	// Thickness is the thickness of the line; the default is 1 in Material 3 and
	// 0 (a hairline) in Material 2.
	Thickness foundation.Opt[float64]
	// Indent is the amount of empty space before the line.
	Indent foundation.Opt[float64]
	// EndIndent is the amount of empty space after the line.
	EndIndent foundation.Opt[float64]
	// Color is the color of the line.
	Color foundation.Opt[ui.Color]
	// Radius is the border radius of the line.
	Radius painting.BorderRadiusGeometry
}

// Divider is a thin horizontal line, with padding on either side.
//
// Flutter: material/divider.dart Divider
type Divider struct {
	widgets.StatelessWidget
	o DividerOpts
}

// NewDivider creates a divider. The tags of its failures are
// DividerSizeNegative, DividerThicknessNegative, DividerIndentNegative and
// DividerEndIndentNegative (Flutter's assertions).
//
// Flutter: material/divider.dart Divider.new
func NewDivider(opts ...DividerOpts) *Divider {
	o := firstOpts(opts)
	checkDividerSizes("Divider", o.Height, o.Thickness, o.Indent, o.EndIndent)
	return &Divider{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// DividerBorderSideOpts are the optional parameters of DividerCreateBorderSide.
//
// Flutter: material/divider.dart Divider.createBorderSide
type DividerBorderSideOpts struct {
	// Color overrides the color of the theme.
	Color foundation.Opt[ui.Color]
	// Width overrides the thickness of the theme.
	Width foundation.Opt[float64]
}

// DividerCreateBorderSide is a border side that has the color and the width of
// the divider of the context's theme (a hairline with no color when the context
// is nil), for the widgets that draw a divider as a border.
//
// Flutter: material/divider.dart Divider.createBorderSide
func DividerCreateBorderSide(bc widgets.BuildContext, opts ...DividerBorderSideOpts) painting.BorderSide {
	o := firstOpts(opts)
	var color foundation.Opt[ui.Color]
	width := o.Width
	if bc != nil {
		theme := DividerThemeOf(bc)
		_, defThickness, _, _, defColor := dividerDefaults(bc)
		color = firstOpt(o.Color, theme.Color, foundation.Some(defColor))
		width = firstOpt(o.Width, theme.Thickness, foundation.Some(defThickness))
	} else {
		color = o.Color
	}
	if c, ok := color.Get(); ok {
		return painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(c), Width: foundation.Some(width.OrElse(0))})
	}
	return painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(width.OrElse(0))})
}

// Build builds the divider.
//
// Flutter: material/divider.dart Divider.build
func (w *Divider) Build(bc widgets.BuildContext) widgets.Widget {
	theme := DividerThemeOf(bc)
	space, thickness, indent, endIndent, _ := dividerDefaults(bc)
	height := firstOpt(w.o.Height, theme.Space, foundation.Some(space)).Value()
	thick := firstOpt(w.o.Thickness, theme.Thickness, foundation.Some(thickness)).Value()
	ind := firstOpt(w.o.Indent, theme.Indent, foundation.Some(indent)).Value()
	endInd := firstOpt(w.o.EndIndent, theme.EndIndent, foundation.Some(endIndent)).Value()
	radius := w.o.Radius
	if radius == nil {
		radius = theme.Radius
	}
	return widgets.NewSizedBox(
		widgets.NewCenter(widgets.NewContainer(nil, widgets.ContainerOpts{
			Height: foundation.Some(thick),
			Margin: painting.NewEdgeInsetsDirectionalFromSTEB(ind, 0, endInd, 0),
			Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
				BorderRadius: radius,
				Border:       painting.Border{Bottom: DividerCreateBorderSide(bc, DividerBorderSideOpts{Color: w.o.Color, Width: foundation.Some(thick)})},
			}),
		})),
		widgets.SizedBoxOpts{Height: foundation.Some(height)})
}

// VerticalDividerOpts are the optional parameters of NewVerticalDivider.
//
// Flutter: material/divider.dart VerticalDivider.new
type VerticalDividerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Width is the divider's width: the space the divider takes, with the line
	// in the middle of it; the default is 16.
	Width foundation.Opt[float64]
	// Thickness is the thickness of the line; the default is 1 in Material 3 and
	// 0 (a hairline) in Material 2.
	Thickness foundation.Opt[float64]
	// Indent is the amount of empty space above the line.
	Indent foundation.Opt[float64]
	// EndIndent is the amount of empty space below the line.
	EndIndent foundation.Opt[float64]
	// Color is the color of the line.
	Color foundation.Opt[ui.Color]
	// Radius is the border radius of the line.
	Radius painting.BorderRadiusGeometry
}

// VerticalDivider is a thin vertical line, with padding on either side.
//
// Flutter: material/divider.dart VerticalDivider
type VerticalDivider struct {
	widgets.StatelessWidget
	o VerticalDividerOpts
}

// NewVerticalDivider creates a vertical divider. The tags of its failures are
// VerticalDividerSizeNegative, VerticalDividerThicknessNegative,
// VerticalDividerIndentNegative and VerticalDividerEndIndentNegative.
//
// Flutter: material/divider.dart VerticalDivider.new
func NewVerticalDivider(opts ...VerticalDividerOpts) *VerticalDivider {
	o := firstOpts(opts)
	checkDividerSizes("VerticalDivider", o.Width, o.Thickness, o.Indent, o.EndIndent)
	return &VerticalDivider{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Build builds the divider.
//
// Flutter: material/divider.dart VerticalDivider.build
func (w *VerticalDivider) Build(bc widgets.BuildContext) widgets.Widget {
	theme := DividerThemeOf(bc)
	space, thickness, indent, endIndent, _ := dividerDefaults(bc)
	width := firstOpt(w.o.Width, theme.Space, foundation.Some(space)).Value()
	thick := firstOpt(w.o.Thickness, theme.Thickness, foundation.Some(thickness)).Value()
	ind := firstOpt(w.o.Indent, theme.Indent, foundation.Some(indent)).Value()
	endInd := firstOpt(w.o.EndIndent, theme.EndIndent, foundation.Some(endIndent)).Value()
	radius := w.o.Radius
	if radius == nil {
		radius = theme.Radius
	}
	return widgets.NewSizedBox(
		widgets.NewCenter(widgets.NewContainer(nil, widgets.ContainerOpts{
			Width:  foundation.Some(thick),
			Margin: painting.NewEdgeInsetsFromLTRB(0, ind, 0, endInd),
			Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
				BorderRadius: radius,
				Border:       painting.Border{Left: DividerCreateBorderSide(bc, DividerBorderSideOpts{Color: w.o.Color, Width: foundation.Some(thick)})},
			}),
		})),
		widgets.SizedBoxOpts{Width: foundation.Some(width)})
}
