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

// BottomAppBarThemeData defines default property values for descendant
// BottomAppBar widgets. Every field that is not set leaves the choice to the
// BottomAppBar's own defaults.
//
// Deviation: Flutter's legacy BottomAppBarTheme constructor that took each
// field separately is not ported; the data is the one parameter (SPEC §12.10).
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarThemeData
//
//glutter:copywith value
type BottomAppBarThemeData struct {
	// Color is the background color of the bar.
	Color foundation.Opt[ui.Color]
	// Elevation is the elevation of the bar.
	Elevation foundation.Opt[float64]
	// Shape is the notch shape of the bar.
	Shape painting.NotchedShape
	// Height is the height of the bar.
	Height foundation.Opt[float64]
	// SurfaceTintColor is the surface tint color.
	SurfaceTintColor foundation.Opt[ui.Color]
	// ShadowColor is the color of the shadow.
	ShadowColor foundation.Opt[ui.Color]
	// Padding is the padding around the child.
	Padding painting.EdgeInsetsGeometry
}

// BottomAppBarThemeDataOpts are the parameters of NewBottomAppBarThemeData: the
// fields of the data.
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarThemeData.new
type BottomAppBarThemeDataOpts = BottomAppBarThemeData

// NewBottomAppBarThemeData creates the data; every field is optional. In debug
// builds it panics with the tag BottomAppBarThemeElevationNegative when the
// elevation is negative.
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarThemeData.new
func NewBottomAppBarThemeData(opts ...BottomAppBarThemeDataOpts) BottomAppBarThemeData {
	d := firstOpts(opts)
	if foundation.DebugMode && d.Elevation.OrElse(0) < 0 {
		panic(errors.New("BottomAppBarThemeElevationNegative", "the elevation of a BottomAppBarThemeData must not be negative", "elevation", d.Elevation.Value()))
	}
	return d
}

// BottomAppBarThemeDataLerp linearly interpolates between two themes. The shape
// switches at t = 0.5.
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarThemeData.lerp
func BottomAppBarThemeDataLerp(a, b BottomAppBarThemeData, t float64) BottomAppBarThemeData {
	return BottomAppBarThemeData{
		Color:            lerpOptColor(a.Color, b.Color, t),
		Elevation:        lerpOptDouble(a.Elevation, b.Elevation, t),
		Shape:            lerpFirstOrSecond(a.Shape, b.Shape, t),
		Height:           lerpOptDouble(a.Height, b.Height, t),
		SurfaceTintColor: lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		ShadowColor:      lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		Padding:          painting.EdgeInsetsGeometryLerp(a.Padding, b.Padding, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarThemeData.==
func (d BottomAppBarThemeData) Equal(o BottomAppBarThemeData) bool {
	return d.Color == o.Color && d.Elevation == o.Elevation && foundation.ValuesEqual(d.Shape, o.Shape) && d.Height == o.Height &&
		d.SurfaceTintColor == o.SurfaceTintColor && d.ShadowColor == o.ShadowColor && foundation.ValuesEqual(d.Padding, o.Padding)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (BottomAppBarThemeData) ToStringShort() string { return "BottomAppBarThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d BottomAppBarThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarThemeData.debugFillProperties
func (d BottomAppBarThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("color", d.Color))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(objectProp("shape", d.Shape))
	p.Add(doubleProp("height", d.Height))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(objectProp("padding", d.Padding))
}

// BottomAppBarTheme is an inherited theme that overrides the
// BottomAppBarThemeData of the theme for the BottomAppBar widgets below it.
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarTheme
type BottomAppBarTheme struct {
	widgets.InheritedTheme
	data BottomAppBarThemeData
}

// NewBottomAppBarTheme creates the theme. A nil child is an empty SizedBox, as
// in Flutter.
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarTheme.new
func NewBottomAppBarTheme(data BottomAppBarThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *BottomAppBarTheme {
	if child == nil {
		child = widgets.NewSizedBoxShrink(nil)
	}
	return &BottomAppBarTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarTheme.data
func (w *BottomAppBarTheme) Data() BottomAppBarThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarTheme.updateShouldNotify
func (w *BottomAppBarTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*BottomAppBarTheme).data)
}

// Wrap makes a BottomAppBarTheme with the same data around the child.
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarTheme.wrap
func (w *BottomAppBarTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewBottomAppBarTheme(w.data, child)
}

// BottomAppBarThemeOf returns the data of the closest BottomAppBarTheme above the
// context, or the bottomAppBarTheme of the closest Theme. The context depends on
// it.
//
// Flutter: material/bottom_app_bar_theme.dart BottomAppBarTheme.of
func BottomAppBarThemeOf(bc widgets.BuildContext) BottomAppBarThemeData {
	if t, ok := widgets.DependOn[*BottomAppBarTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).BottomAppBarTheme
}
