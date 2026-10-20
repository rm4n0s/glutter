// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// SearchBarThemeData defines default property values for descendant SearchBar widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/search_bar_theme.dart SearchBarThemeData
//
//glutter:copywith value
type SearchBarThemeData struct {
	// Elevation is the elevation of the search bar, in each state.
	Elevation widgets.WidgetStateProperty[foundation.Opt[float64]]
	// BackgroundColor is the color of the search bar's background, in each state.
	BackgroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// ShadowColor is the color of the search bar's shadow, in each state.
	ShadowColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SurfaceTintColor is the color of the search bar's surface tint overlay, in each state.
	SurfaceTintColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// OverlayColor is the color of the overlay of the search bar (the hover, focus and press highlights), in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// Side is the border of the search bar, in each state.
	Side widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// Shape is the shape of the search bar, in each state.
	Shape widgets.WidgetStateProperty[painting.OutlinedBorder]
	// Padding is the padding between the search bar's border and its contents, in each state.
	Padding widgets.WidgetStateProperty[painting.EdgeInsetsGeometry]
	// TextStyle is the style of the text that the user types, in each state.
	TextStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// HintStyle is the style of the hint text, in each state.
	HintStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// Constraints are the size constraints of the search bar.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// TextCapitalization is how the input method capitalizes the text.
	TextCapitalization services.TextCapitalization
}

// SearchBarThemeDataOpts are the parameters of NewSearchBarThemeData: the fields of the data.
//
// Flutter: material/search_bar_theme.dart SearchBarThemeData.new
type SearchBarThemeDataOpts = SearchBarThemeData

// NewSearchBarThemeData creates the data; every field is optional.
//
// Flutter: material/search_bar_theme.dart SearchBarThemeData.new
func NewSearchBarThemeData(opts ...SearchBarThemeDataOpts) SearchBarThemeData {
	return firstOpts(opts)
}

// SearchBarThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/search_bar_theme.dart SearchBarThemeData.lerp
func SearchBarThemeDataLerp(a, b SearchBarThemeData, t float64) SearchBarThemeData {
	return SearchBarThemeData{
		Elevation:          lerpStateOptDouble(a.Elevation, b.Elevation, t),
		BackgroundColor:    lerpStateOptColor(a.BackgroundColor, b.BackgroundColor, t),
		ShadowColor:        lerpStateOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor:   lerpStateOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		OverlayColor:       lerpStateOptColor(a.OverlayColor, b.OverlayColor, t),
		Side:               lerpStateOptSide(a.Side, b.Side, t),
		Shape:              lerpStateOutlined(a.Shape, b.Shape, t),
		Padding:            lerpStateWith(a.Padding, b.Padding, t, lerpOptInsets),
		TextStyle:          lerpStateWith(a.TextStyle, b.TextStyle, t, lerpOptTextStyle),
		HintStyle:          lerpStateWith(a.HintStyle, b.HintStyle, t, lerpOptTextStyle),
		Constraints:        lerpOptConstraints(a.Constraints, b.Constraints, t),
		TextCapitalization: lerpFirstOrSecond(a.TextCapitalization, b.TextCapitalization, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/search_bar_theme.dart SearchBarThemeData.==
func (d SearchBarThemeData) Equal(o SearchBarThemeData) bool {
	return foundation.ValuesEqual(d.Elevation, o.Elevation) &&
		foundation.ValuesEqual(d.BackgroundColor, o.BackgroundColor) &&
		foundation.ValuesEqual(d.ShadowColor, o.ShadowColor) &&
		foundation.ValuesEqual(d.SurfaceTintColor, o.SurfaceTintColor) &&
		foundation.ValuesEqual(d.OverlayColor, o.OverlayColor) &&
		foundation.ValuesEqual(d.Side, o.Side) &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		foundation.ValuesEqual(d.Padding, o.Padding) &&
		foundation.ValuesEqual(d.TextStyle, o.TextStyle) &&
		foundation.ValuesEqual(d.HintStyle, o.HintStyle) &&
		d.Constraints == o.Constraints &&
		d.TextCapitalization == o.TextCapitalization
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (SearchBarThemeData) ToStringShort() string { return "SearchBarThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d SearchBarThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/search_bar_theme.dart SearchBarThemeData.debugFillProperties
func (d SearchBarThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("elevation", propOrNil(d.Elevation)))
	p.Add(objectProp("backgroundColor", propOrNil(d.BackgroundColor)))
	p.Add(objectProp("shadowColor", propOrNil(d.ShadowColor)))
	p.Add(objectProp("surfaceTintColor", propOrNil(d.SurfaceTintColor)))
	p.Add(objectProp("overlayColor", propOrNil(d.OverlayColor)))
	p.Add(objectProp("side", propOrNil(d.Side)))
	p.Add(objectProp("shape", propOrNil(d.Shape)))
	p.Add(objectProp("padding", propOrNil(d.Padding)))
	p.Add(objectProp("textStyle", propOrNil(d.TextStyle)))
	p.Add(objectProp("hintStyle", propOrNil(d.HintStyle)))
	p.Add(objectProp("constraints", optValue(d.Constraints)))
	p.Add(objectProp("textCapitalization", nilIfZero(d.TextCapitalization)))
}

// SearchBarTheme is an inherited theme that overrides the SearchBarThemeData of the theme for the
// widgets below it.
//
// Flutter: material/search_bar_theme.dart SearchBarTheme
type SearchBarTheme struct {
	widgets.InheritedTheme
	data SearchBarThemeData
}

// NewSearchBarTheme creates the theme.
//
// Flutter: material/search_bar_theme.dart SearchBarTheme.new
func NewSearchBarTheme(data SearchBarThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *SearchBarTheme {
	return &SearchBarTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/search_bar_theme.dart SearchBarTheme.data
func (w *SearchBarTheme) Data() SearchBarThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/search_bar_theme.dart SearchBarTheme.updateShouldNotify
func (w *SearchBarTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*SearchBarTheme).data)
}

// Wrap makes a SearchBarTheme with the same data around the child.
//
// Flutter: material/search_bar_theme.dart SearchBarTheme.wrap
func (w *SearchBarTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewSearchBarTheme(w.data, child)
}

// SearchBarThemeOf returns the data of the closest SearchBarTheme above the context, or the
// searchBarTheme of the closest Theme. The context depends on it.
//
// Flutter: material/search_bar_theme.dart SearchBarTheme.of
func SearchBarThemeOf(bc widgets.BuildContext) SearchBarThemeData {
	if t, ok := widgets.DependOn[*SearchBarTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).SearchBarTheme
}
