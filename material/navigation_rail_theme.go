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

// NavigationRailThemeData defines default property values for descendant NavigationRail widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailThemeData
//
//glutter:copywith value
type NavigationRailThemeData struct {
	// BackgroundColor is the color of the rail.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the rail.
	Elevation foundation.Opt[float64]
	// UnselectedLabelTextStyle is the style of the labels of the destinations that are not selected.
	UnselectedLabelTextStyle *painting.TextStyle
	// SelectedLabelTextStyle is the style of the label of the selected destination.
	SelectedLabelTextStyle *painting.TextStyle
	// UnselectedIconTheme is the theme of the icons of the destinations that are not selected.
	UnselectedIconTheme foundation.Opt[widgets.IconThemeData]
	// SelectedIconTheme is the theme of the icon of the selected destination.
	SelectedIconTheme foundation.Opt[widgets.IconThemeData]
	// GroupAlignment is the vertical alignment of the destinations, from -1 (top) to 1 (bottom).
	GroupAlignment foundation.Opt[float64]
	// LabelType is how the labels are shown.
	LabelType NavigationRailLabelType
	// UseIndicator says whether the selected destination shows an indicator.
	UseIndicator foundation.Opt[bool]
	// IndicatorColor is the color of the indicator.
	IndicatorColor foundation.Opt[ui.Color]
	// IndicatorShape is the shape of the indicator.
	IndicatorShape painting.ShapeBorder
	// MinWidth is the width of the rail when it is not extended.
	MinWidth foundation.Opt[float64]
	// MinExtendedWidth is the width of the rail when it is extended.
	MinExtendedWidth foundation.Opt[float64]
}

// NavigationRailThemeDataOpts are the parameters of NewNavigationRailThemeData: the fields of the data.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailThemeData.new
type NavigationRailThemeDataOpts = NavigationRailThemeData

// NewNavigationRailThemeData creates the data; every field is optional.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailThemeData.new
func NewNavigationRailThemeData(opts ...NavigationRailThemeDataOpts) NavigationRailThemeData {
	return firstOpts(opts)
}

// NavigationRailThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailThemeData.lerp
func NavigationRailThemeDataLerp(a, b NavigationRailThemeData, t float64) NavigationRailThemeData {
	return NavigationRailThemeData{
		BackgroundColor:          lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		Elevation:                lerpOptDouble(a.Elevation, b.Elevation, t),
		UnselectedLabelTextStyle: painting.TextStyleLerp(a.UnselectedLabelTextStyle, b.UnselectedLabelTextStyle, t),
		SelectedLabelTextStyle:   painting.TextStyleLerp(a.SelectedLabelTextStyle, b.SelectedLabelTextStyle, t),
		UnselectedIconTheme:      lerpOptIconTheme(a.UnselectedIconTheme, b.UnselectedIconTheme, t),
		SelectedIconTheme:        lerpOptIconTheme(a.SelectedIconTheme, b.SelectedIconTheme, t),
		GroupAlignment:           lerpOptDouble(a.GroupAlignment, b.GroupAlignment, t),
		LabelType:                lerpFirstOrSecond(a.LabelType, b.LabelType, t),
		UseIndicator:             lerpFirstOrSecond(a.UseIndicator, b.UseIndicator, t),
		IndicatorColor:           lerpOptColor(a.IndicatorColor, b.IndicatorColor, t),
		IndicatorShape:           painting.ShapeBorderLerp(a.IndicatorShape, b.IndicatorShape, t),
		MinWidth:                 lerpOptDouble(a.MinWidth, b.MinWidth, t),
		MinExtendedWidth:         lerpOptDouble(a.MinExtendedWidth, b.MinExtendedWidth, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailThemeData.==
func (d NavigationRailThemeData) Equal(o NavigationRailThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor &&
		d.Elevation == o.Elevation &&
		textStyleEqual(d.UnselectedLabelTextStyle, o.UnselectedLabelTextStyle) &&
		textStyleEqual(d.SelectedLabelTextStyle, o.SelectedLabelTextStyle) &&
		iconThemeOptEqual(d.UnselectedIconTheme, o.UnselectedIconTheme) &&
		iconThemeOptEqual(d.SelectedIconTheme, o.SelectedIconTheme) &&
		d.GroupAlignment == o.GroupAlignment &&
		d.LabelType == o.LabelType &&
		d.UseIndicator == o.UseIndicator &&
		d.IndicatorColor == o.IndicatorColor &&
		foundation.ValuesEqual(d.IndicatorShape, o.IndicatorShape) &&
		d.MinWidth == o.MinWidth &&
		d.MinExtendedWidth == o.MinExtendedWidth
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (NavigationRailThemeData) ToStringShort() string { return "NavigationRailThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d NavigationRailThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailThemeData.debugFillProperties
func (d NavigationRailThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(objectProp("unselectedLabelTextStyle", d.UnselectedLabelTextStyle))
	p.Add(objectProp("selectedLabelTextStyle", d.SelectedLabelTextStyle))
	p.Add(objectProp("unselectedIconTheme", optValue(d.UnselectedIconTheme)))
	p.Add(objectProp("selectedIconTheme", optValue(d.SelectedIconTheme)))
	p.Add(doubleProp("groupAlignment", d.GroupAlignment))
	p.Add(objectProp("labelType", nilIfZero(d.LabelType)))
	p.Add(objectProp("useIndicator", optValue(d.UseIndicator)))
	p.Add(colorProp("indicatorColor", d.IndicatorColor))
	p.Add(objectProp("indicatorShape", d.IndicatorShape))
	p.Add(doubleProp("minWidth", d.MinWidth))
	p.Add(doubleProp("minExtendedWidth", d.MinExtendedWidth))
}

// NavigationRailTheme is an inherited theme that overrides the NavigationRailThemeData of the theme for the
// widgets below it.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailTheme
type NavigationRailTheme struct {
	widgets.InheritedTheme
	data NavigationRailThemeData
}

// NewNavigationRailTheme creates the theme.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailTheme.new
func NewNavigationRailTheme(data NavigationRailThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *NavigationRailTheme {
	return &NavigationRailTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailTheme.data
func (w *NavigationRailTheme) Data() NavigationRailThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailTheme.updateShouldNotify
func (w *NavigationRailTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*NavigationRailTheme).data)
}

// Wrap makes a NavigationRailTheme with the same data around the child.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailTheme.wrap
func (w *NavigationRailTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewNavigationRailTheme(w.data, child)
}

// NavigationRailThemeOf returns the data of the closest NavigationRailTheme above the context, or the
// navigationRailTheme of the closest Theme. The context depends on it.
//
// Flutter: material/navigation_rail_theme.dart NavigationRailTheme.of
func NavigationRailThemeOf(bc widgets.BuildContext) NavigationRailThemeData {
	if t, ok := widgets.DependOn[*NavigationRailTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).NavigationRailTheme
}
