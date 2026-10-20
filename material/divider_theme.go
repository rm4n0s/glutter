// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// DividerThemeData defines default property values for descendant Divider and VerticalDivider widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/divider_theme.dart DividerThemeData
//
//glutter:copywith value
type DividerThemeData struct {
	// Color is the color of the dividers.
	Color foundation.Opt[ui.Color]
	// Space is the height of a Divider and the width of a VerticalDivider, which is the space the divider takes.
	Space foundation.Opt[float64]
	// Thickness is the thickness of the line that the divider draws.
	Thickness foundation.Opt[float64]
	// Indent is the amount of empty space before the line.
	Indent foundation.Opt[float64]
	// EndIndent is the amount of empty space after the line.
	EndIndent foundation.Opt[float64]
	// Radius is the border radius of the line.
	Radius painting.BorderRadiusGeometry
}

// DividerThemeDataOpts are the parameters of NewDividerThemeData: the fields of the data.
//
// Flutter: material/divider_theme.dart DividerThemeData.new
type DividerThemeDataOpts = DividerThemeData

// NewDividerThemeData creates the data; every field is optional.
//
// Flutter: material/divider_theme.dart DividerThemeData.new
func NewDividerThemeData(opts ...DividerThemeDataOpts) DividerThemeData {
	return firstOpts(opts)
}

// DividerThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/divider_theme.dart DividerThemeData.lerp
func DividerThemeDataLerp(a, b DividerThemeData, t float64) DividerThemeData {
	return DividerThemeData{
		Color:     lerpOptColor(a.Color, b.Color, t),
		Space:     lerpOptDouble(a.Space, b.Space, t),
		Thickness: lerpOptDouble(a.Thickness, b.Thickness, t),
		Indent:    lerpOptDouble(a.Indent, b.Indent, t),
		EndIndent: lerpOptDouble(a.EndIndent, b.EndIndent, t),
		Radius:    painting.BorderRadiusGeometryLerp(a.Radius, b.Radius, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/divider_theme.dart DividerThemeData.==
func (d DividerThemeData) Equal(o DividerThemeData) bool {
	return d.Color == o.Color &&
		d.Space == o.Space &&
		d.Thickness == o.Thickness &&
		d.Indent == o.Indent &&
		d.EndIndent == o.EndIndent &&
		foundation.ValuesEqual(d.Radius, o.Radius)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (DividerThemeData) ToStringShort() string { return "DividerThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d DividerThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/divider_theme.dart DividerThemeData.debugFillProperties
func (d DividerThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("color", d.Color))
	p.Add(doubleProp("space", d.Space))
	p.Add(doubleProp("thickness", d.Thickness))
	p.Add(doubleProp("indent", d.Indent))
	p.Add(doubleProp("endIndent", d.EndIndent))
	p.Add(objectProp("radius", d.Radius))
}

// DividerTheme is an inherited theme that overrides the DividerThemeData of the theme for the
// widgets below it.
//
// Flutter: material/divider_theme.dart DividerTheme
type DividerTheme struct {
	widgets.InheritedTheme
	data DividerThemeData
}

// NewDividerTheme creates the theme.
//
// Flutter: material/divider_theme.dart DividerTheme.new
func NewDividerTheme(data DividerThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *DividerTheme {
	return &DividerTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/divider_theme.dart DividerTheme.data
func (w *DividerTheme) Data() DividerThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/divider_theme.dart DividerTheme.updateShouldNotify
func (w *DividerTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*DividerTheme).data)
}

// Wrap makes a DividerTheme with the same data around the child.
//
// Flutter: material/divider_theme.dart DividerTheme.wrap
func (w *DividerTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewDividerTheme(w.data, child)
}

// DividerThemeOf returns the data of the closest DividerTheme above the context, or the
// dividerTheme of the closest Theme. The context depends on it.
//
// Flutter: material/divider_theme.dart DividerTheme.of
func DividerThemeOf(bc widgets.BuildContext) DividerThemeData {
	if t, ok := widgets.DependOn[*DividerTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).DividerTheme
}
