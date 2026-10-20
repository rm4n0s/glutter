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

// NavigationDrawerThemeData defines default property values for descendant NavigationDrawer widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerThemeData
//
//glutter:copywith value
type NavigationDrawerThemeData struct {
	// TileHeight is the height of a destination.
	TileHeight foundation.Opt[float64]
	// BackgroundColor is the color of the drawer.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the drawer.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow below the drawer.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the drawer (Material 3).
	SurfaceTintColor foundation.Opt[ui.Color]
	// IndicatorColor is the color of the indicator of the selected destination.
	IndicatorColor foundation.Opt[ui.Color]
	// IndicatorShape is the shape of the indicator.
	IndicatorShape painting.ShapeBorder
	// IndicatorSize is the size of the indicator.
	IndicatorSize foundation.Opt[ui.Size]
	// LabelTextStyle is the style of the labels, in each state.
	LabelTextStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// IconTheme is the theme of the icons, in each state.
	IconTheme widgets.WidgetStateProperty[widgets.IconThemeData]
}

// NavigationDrawerThemeDataOpts are the parameters of NewNavigationDrawerThemeData: the fields of the data.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerThemeData.new
type NavigationDrawerThemeDataOpts = NavigationDrawerThemeData

// NewNavigationDrawerThemeData creates the data; every field is optional.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerThemeData.new
func NewNavigationDrawerThemeData(opts ...NavigationDrawerThemeDataOpts) NavigationDrawerThemeData {
	return firstOpts(opts)
}

// NavigationDrawerThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerThemeData.lerp
func NavigationDrawerThemeDataLerp(a, b NavigationDrawerThemeData, t float64) NavigationDrawerThemeData {
	return NavigationDrawerThemeData{
		TileHeight:       lerpOptDouble(a.TileHeight, b.TileHeight, t),
		BackgroundColor:  lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		Elevation:        lerpOptDouble(a.Elevation, b.Elevation, t),
		ShadowColor:      lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor: lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		IndicatorColor:   lerpOptColor(a.IndicatorColor, b.IndicatorColor, t),
		IndicatorShape:   painting.ShapeBorderLerp(a.IndicatorShape, b.IndicatorShape, t),
		IndicatorSize:    lerpOptSize(a.IndicatorSize, b.IndicatorSize, t),
		LabelTextStyle:   lerpStateWith(a.LabelTextStyle, b.LabelTextStyle, t, lerpOptTextStyle),
		IconTheme:        lerpStateWith(a.IconTheme, b.IconTheme, t, lerpOptIconTheme),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerThemeData.==
func (d NavigationDrawerThemeData) Equal(o NavigationDrawerThemeData) bool {
	return d.TileHeight == o.TileHeight &&
		d.BackgroundColor == o.BackgroundColor &&
		d.Elevation == o.Elevation &&
		d.ShadowColor == o.ShadowColor &&
		d.SurfaceTintColor == o.SurfaceTintColor &&
		d.IndicatorColor == o.IndicatorColor &&
		foundation.ValuesEqual(d.IndicatorShape, o.IndicatorShape) &&
		d.IndicatorSize == o.IndicatorSize &&
		foundation.ValuesEqual(d.LabelTextStyle, o.LabelTextStyle) &&
		foundation.ValuesEqual(d.IconTheme, o.IconTheme)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (NavigationDrawerThemeData) ToStringShort() string { return "NavigationDrawerThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d NavigationDrawerThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerThemeData.debugFillProperties
func (d NavigationDrawerThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(doubleProp("tileHeight", d.TileHeight))
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(colorProp("indicatorColor", d.IndicatorColor))
	p.Add(objectProp("indicatorShape", d.IndicatorShape))
	p.Add(objectProp("indicatorSize", optValue(d.IndicatorSize)))
	p.Add(objectProp("labelTextStyle", propOrNil(d.LabelTextStyle)))
	p.Add(objectProp("iconTheme", propOrNil(d.IconTheme)))
}

// NavigationDrawerTheme is an inherited theme that overrides the NavigationDrawerThemeData of the theme for the
// widgets below it.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerTheme
type NavigationDrawerTheme struct {
	widgets.InheritedTheme
	data NavigationDrawerThemeData
}

// NewNavigationDrawerTheme creates the theme.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerTheme.new
func NewNavigationDrawerTheme(data NavigationDrawerThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *NavigationDrawerTheme {
	return &NavigationDrawerTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerTheme.data
func (w *NavigationDrawerTheme) Data() NavigationDrawerThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerTheme.updateShouldNotify
func (w *NavigationDrawerTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*NavigationDrawerTheme).data)
}

// Wrap makes a NavigationDrawerTheme with the same data around the child.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerTheme.wrap
func (w *NavigationDrawerTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewNavigationDrawerTheme(w.data, child)
}

// NavigationDrawerThemeOf returns the data of the closest NavigationDrawerTheme above the context, or the
// navigationDrawerTheme of the closest Theme. The context depends on it.
//
// Flutter: material/navigation_drawer_theme.dart NavigationDrawerTheme.of
func NavigationDrawerThemeOf(bc widgets.BuildContext) NavigationDrawerThemeData {
	if t, ok := widgets.DependOn[*NavigationDrawerTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).NavigationDrawerTheme
}
