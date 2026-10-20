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

// NavigationBarThemeData defines default property values for descendant NavigationBar widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarThemeData
//
//glutter:copywith value
type NavigationBarThemeData struct {
	// Height is the height of the bar, not counting the system padding below it.
	Height foundation.Opt[float64]
	// BackgroundColor is the color of the bar.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the bar.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow below the bar.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the bar (Material 3).
	SurfaceTintColor foundation.Opt[ui.Color]
	// IndicatorColor is the color of the indicator of the selected destination.
	IndicatorColor foundation.Opt[ui.Color]
	// IndicatorShape is the shape of the indicator.
	IndicatorShape painting.ShapeBorder
	// LabelTextStyle is the style of the labels, in each state.
	LabelTextStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// IconTheme is the theme of the icons, in each state.
	IconTheme widgets.WidgetStateProperty[widgets.IconThemeData]
	// LabelBehavior is how the labels of the destinations are shown.
	LabelBehavior NavigationDestinationLabelBehavior
	// OverlayColor is the color of the highlight and the splash of a destination, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// LabelPadding is the padding around the labels.
	LabelPadding painting.EdgeInsetsGeometry
}

// NavigationBarThemeDataOpts are the parameters of NewNavigationBarThemeData: the fields of the data.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarThemeData.new
type NavigationBarThemeDataOpts = NavigationBarThemeData

// NewNavigationBarThemeData creates the data; every field is optional.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarThemeData.new
func NewNavigationBarThemeData(opts ...NavigationBarThemeDataOpts) NavigationBarThemeData {
	return firstOpts(opts)
}

// NavigationBarThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarThemeData.lerp
func NavigationBarThemeDataLerp(a, b NavigationBarThemeData, t float64) NavigationBarThemeData {
	return NavigationBarThemeData{
		Height:           lerpOptDouble(a.Height, b.Height, t),
		BackgroundColor:  lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		Elevation:        lerpOptDouble(a.Elevation, b.Elevation, t),
		ShadowColor:      lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor: lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		IndicatorColor:   lerpOptColor(a.IndicatorColor, b.IndicatorColor, t),
		IndicatorShape:   painting.ShapeBorderLerp(a.IndicatorShape, b.IndicatorShape, t),
		LabelTextStyle:   lerpStateWith(a.LabelTextStyle, b.LabelTextStyle, t, lerpOptTextStyle),
		IconTheme:        lerpStateWith(a.IconTheme, b.IconTheme, t, lerpOptIconTheme),
		LabelBehavior:    lerpFirstOrSecond(a.LabelBehavior, b.LabelBehavior, t),
		OverlayColor:     lerpStateOptColor(a.OverlayColor, b.OverlayColor, t),
		LabelPadding:     painting.EdgeInsetsGeometryLerp(a.LabelPadding, b.LabelPadding, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarThemeData.==
func (d NavigationBarThemeData) Equal(o NavigationBarThemeData) bool {
	return d.Height == o.Height &&
		d.BackgroundColor == o.BackgroundColor &&
		d.Elevation == o.Elevation &&
		d.ShadowColor == o.ShadowColor &&
		d.SurfaceTintColor == o.SurfaceTintColor &&
		d.IndicatorColor == o.IndicatorColor &&
		foundation.ValuesEqual(d.IndicatorShape, o.IndicatorShape) &&
		foundation.ValuesEqual(d.LabelTextStyle, o.LabelTextStyle) &&
		foundation.ValuesEqual(d.IconTheme, o.IconTheme) &&
		d.LabelBehavior == o.LabelBehavior &&
		foundation.ValuesEqual(d.OverlayColor, o.OverlayColor) &&
		foundation.ValuesEqual(d.LabelPadding, o.LabelPadding)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (NavigationBarThemeData) ToStringShort() string { return "NavigationBarThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d NavigationBarThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarThemeData.debugFillProperties
func (d NavigationBarThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(doubleProp("height", d.Height))
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(colorProp("indicatorColor", d.IndicatorColor))
	p.Add(objectProp("indicatorShape", d.IndicatorShape))
	p.Add(objectProp("labelTextStyle", propOrNil(d.LabelTextStyle)))
	p.Add(objectProp("iconTheme", propOrNil(d.IconTheme)))
	p.Add(objectProp("labelBehavior", nilIfZero(d.LabelBehavior)))
	p.Add(objectProp("overlayColor", propOrNil(d.OverlayColor)))
	p.Add(objectProp("labelPadding", d.LabelPadding))
}

// NavigationBarTheme is an inherited theme that overrides the NavigationBarThemeData of the theme for the
// widgets below it.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarTheme
type NavigationBarTheme struct {
	widgets.InheritedTheme
	data NavigationBarThemeData
}

// NewNavigationBarTheme creates the theme.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarTheme.new
func NewNavigationBarTheme(data NavigationBarThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *NavigationBarTheme {
	return &NavigationBarTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarTheme.data
func (w *NavigationBarTheme) Data() NavigationBarThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarTheme.updateShouldNotify
func (w *NavigationBarTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*NavigationBarTheme).data)
}

// Wrap makes a NavigationBarTheme with the same data around the child.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarTheme.wrap
func (w *NavigationBarTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewNavigationBarTheme(w.data, child)
}

// NavigationBarThemeOf returns the data of the closest NavigationBarTheme above the context, or the
// navigationBarTheme of the closest Theme. The context depends on it.
//
// Flutter: material/navigation_bar_theme.dart NavigationBarTheme.of
func NavigationBarThemeOf(bc widgets.BuildContext) NavigationBarThemeData {
	if t, ok := widgets.DependOn[*NavigationBarTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).NavigationBarTheme
}
