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

// DrawerThemeData defines default property values for descendant Drawer widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/drawer_theme.dart DrawerThemeData
//
//glutter:copywith value
type DrawerThemeData struct {
	// BackgroundColor is the color of the drawer.
	BackgroundColor foundation.Opt[ui.Color]
	// ScrimColor is the color of the scrim behind the drawer.
	ScrimColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the drawer.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow below the drawer.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the drawer (Material 3).
	SurfaceTintColor foundation.Opt[ui.Color]
	// Shape is the shape of the drawer.
	Shape painting.ShapeBorder
	// EndShape is the shape of the end drawer.
	EndShape painting.ShapeBorder
	// Width is the width of the drawer.
	Width foundation.Opt[float64]
	// ClipBehavior is the clip of the drawer's Material.
	ClipBehavior ui.Clip
}

// DrawerThemeDataOpts are the parameters of NewDrawerThemeData: the fields of the data.
//
// Flutter: material/drawer_theme.dart DrawerThemeData.new
type DrawerThemeDataOpts = DrawerThemeData

// NewDrawerThemeData creates the data; every field is optional.
//
// Flutter: material/drawer_theme.dart DrawerThemeData.new
func NewDrawerThemeData(opts ...DrawerThemeDataOpts) DrawerThemeData {
	return firstOpts(opts)
}

// DrawerThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/drawer_theme.dart DrawerThemeData.lerp
func DrawerThemeDataLerp(a, b DrawerThemeData, t float64) DrawerThemeData {
	return DrawerThemeData{
		BackgroundColor:  lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		ScrimColor:       lerpOptColor(a.ScrimColor, b.ScrimColor, t),
		Elevation:        lerpOptDouble(a.Elevation, b.Elevation, t),
		ShadowColor:      lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor: lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		Shape:            painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		EndShape:         painting.ShapeBorderLerp(a.EndShape, b.EndShape, t),
		Width:            lerpOptDouble(a.Width, b.Width, t),
		ClipBehavior:     lerpFirstOrSecond(a.ClipBehavior, b.ClipBehavior, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/drawer_theme.dart DrawerThemeData.==
func (d DrawerThemeData) Equal(o DrawerThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor &&
		d.ScrimColor == o.ScrimColor &&
		d.Elevation == o.Elevation &&
		d.ShadowColor == o.ShadowColor &&
		d.SurfaceTintColor == o.SurfaceTintColor &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		foundation.ValuesEqual(d.EndShape, o.EndShape) &&
		d.Width == o.Width &&
		d.ClipBehavior == o.ClipBehavior
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (DrawerThemeData) ToStringShort() string { return "DrawerThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d DrawerThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/drawer_theme.dart DrawerThemeData.debugFillProperties
func (d DrawerThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(colorProp("scrimColor", d.ScrimColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("endShape", d.EndShape))
	p.Add(doubleProp("width", d.Width))
	p.Add(objectProp("clipBehavior", nilIfZero(d.ClipBehavior)))
}

// DrawerTheme is an inherited theme that overrides the DrawerThemeData of the theme for the
// widgets below it.
//
// Flutter: material/drawer_theme.dart DrawerTheme
type DrawerTheme struct {
	widgets.InheritedTheme
	data DrawerThemeData
}

// NewDrawerTheme creates the theme.
//
// Flutter: material/drawer_theme.dart DrawerTheme.new
func NewDrawerTheme(data DrawerThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *DrawerTheme {
	return &DrawerTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/drawer_theme.dart DrawerTheme.data
func (w *DrawerTheme) Data() DrawerThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/drawer_theme.dart DrawerTheme.updateShouldNotify
func (w *DrawerTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*DrawerTheme).data)
}

// Wrap makes a DrawerTheme with the same data around the child.
//
// Flutter: material/drawer_theme.dart DrawerTheme.wrap
func (w *DrawerTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewDrawerTheme(w.data, child)
}

// DrawerThemeOf returns the data of the closest DrawerTheme above the context, or the
// drawerTheme of the closest Theme. The context depends on it.
//
// Flutter: material/drawer_theme.dart DrawerTheme.of
func DrawerThemeOf(bc widgets.BuildContext) DrawerThemeData {
	if t, ok := widgets.DependOn[*DrawerTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).DrawerTheme
}
