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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// SearchViewThemeData defines default property values for descendant SearchAnchor widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/search_view_theme.dart SearchViewThemeData
//
//glutter:copywith value
type SearchViewThemeData struct {
	// BackgroundColor is the color of the search view's background.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the elevation of the search view's Material.
	Elevation foundation.Opt[float64]
	// SurfaceTintColor is the color of the search view's surface tint overlay.
	SurfaceTintColor foundation.Opt[ui.Color]
	// Side is the border of the search view; it is not used by a full screen view.
	Side foundation.Opt[painting.BorderSide]
	// Shape is the shape of the search view; it is not used by a full screen view.
	Shape painting.OutlinedBorder
	// HeaderHeight is the height of the search field on the search view.
	HeaderHeight foundation.Opt[float64]
	// HeaderTextStyle is the style of the text that the user types in the search view.
	HeaderTextStyle *painting.TextStyle
	// HeaderHintStyle is the style of the hint text of the search view. Its lerp interpolates the header text styles, as Flutter's does.
	HeaderHintStyle *painting.TextStyle
	// Constraints are the size constraints of the search view.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Padding is the padding around the search view's contents; the default is none.
	Padding painting.EdgeInsetsGeometry
	// BarPadding is the padding around the search field of the search view.
	BarPadding painting.EdgeInsetsGeometry
	// ShrinkWrap says whether the search view shrinks to the height of its suggestions.
	ShrinkWrap foundation.Opt[bool]
	// DividerColor is the color of the divider under the search field.
	DividerColor foundation.Opt[ui.Color]
}

// SearchViewThemeDataOpts are the parameters of NewSearchViewThemeData: the fields of the data.
//
// Flutter: material/search_view_theme.dart SearchViewThemeData.new
type SearchViewThemeDataOpts = SearchViewThemeData

// NewSearchViewThemeData creates the data; every field is optional.
//
// Flutter: material/search_view_theme.dart SearchViewThemeData.new
func NewSearchViewThemeData(opts ...SearchViewThemeDataOpts) SearchViewThemeData {
	return firstOpts(opts)
}

// SearchViewThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/search_view_theme.dart SearchViewThemeData.lerp
func SearchViewThemeDataLerp(a, b SearchViewThemeData, t float64) SearchViewThemeData {
	return SearchViewThemeData{
		BackgroundColor:  lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		Elevation:        lerpOptDouble(a.Elevation, b.Elevation, t),
		SurfaceTintColor: lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		Side:             lerpOptBorderSide(a.Side, b.Side, t),
		Shape:            painting.OutlinedBorderLerp(a.Shape, b.Shape, t),
		HeaderHeight:     lerpOptDouble(a.HeaderHeight, b.HeaderHeight, t),
		HeaderTextStyle:  painting.TextStyleLerp(a.HeaderTextStyle, b.HeaderTextStyle, t),
		HeaderHintStyle:  painting.TextStyleLerp(a.HeaderTextStyle, b.HeaderTextStyle, t),
		Constraints:      lerpOptConstraints(a.Constraints, b.Constraints, t),
		Padding:          painting.EdgeInsetsGeometryLerp(a.Padding, b.Padding, t),
		BarPadding:       painting.EdgeInsetsGeometryLerp(a.BarPadding, b.BarPadding, t),
		ShrinkWrap:       lerpFirstOrSecond(a.ShrinkWrap, b.ShrinkWrap, t),
		DividerColor:     lerpOptColor(a.DividerColor, b.DividerColor, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/search_view_theme.dart SearchViewThemeData.==
func (d SearchViewThemeData) Equal(o SearchViewThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor &&
		d.Elevation == o.Elevation &&
		d.SurfaceTintColor == o.SurfaceTintColor &&
		d.Side == o.Side &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		d.HeaderHeight == o.HeaderHeight &&
		textStyleEqual(d.HeaderTextStyle, o.HeaderTextStyle) &&
		textStyleEqual(d.HeaderHintStyle, o.HeaderHintStyle) &&
		d.Constraints == o.Constraints &&
		foundation.ValuesEqual(d.Padding, o.Padding) &&
		foundation.ValuesEqual(d.BarPadding, o.BarPadding) &&
		d.ShrinkWrap == o.ShrinkWrap &&
		d.DividerColor == o.DividerColor
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (SearchViewThemeData) ToStringShort() string { return "SearchViewThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d SearchViewThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/search_view_theme.dart SearchViewThemeData.debugFillProperties
func (d SearchViewThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(objectProp("side", optValue(d.Side)))
	p.Add(objectProp("shape", d.Shape))
	p.Add(doubleProp("headerHeight", d.HeaderHeight))
	p.Add(objectProp("headerTextStyle", d.HeaderTextStyle))
	p.Add(objectProp("headerHintStyle", d.HeaderHintStyle))
	p.Add(objectProp("constraints", optValue(d.Constraints)))
	p.Add(objectProp("padding", d.Padding))
	p.Add(objectProp("barPadding", d.BarPadding))
	p.Add(objectProp("shrinkWrap", optValue(d.ShrinkWrap)))
	p.Add(colorProp("dividerColor", d.DividerColor))
}

// SearchViewTheme is an inherited theme that overrides the SearchViewThemeData of the theme for the
// widgets below it.
//
// Flutter: material/search_view_theme.dart SearchViewTheme
type SearchViewTheme struct {
	widgets.InheritedTheme
	data SearchViewThemeData
}

// NewSearchViewTheme creates the theme.
//
// Flutter: material/search_view_theme.dart SearchViewTheme.new
func NewSearchViewTheme(data SearchViewThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *SearchViewTheme {
	return &SearchViewTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/search_view_theme.dart SearchViewTheme.data
func (w *SearchViewTheme) Data() SearchViewThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/search_view_theme.dart SearchViewTheme.updateShouldNotify
func (w *SearchViewTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*SearchViewTheme).data)
}

// Wrap makes a SearchViewTheme with the same data around the child.
//
// Flutter: material/search_view_theme.dart SearchViewTheme.wrap
func (w *SearchViewTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewSearchViewTheme(w.data, child)
}

// SearchViewThemeOf returns the data of the closest SearchViewTheme above the context, or the
// searchViewTheme of the closest Theme. The context depends on it.
//
// Flutter: material/search_view_theme.dart SearchViewTheme.of
func SearchViewThemeOf(bc widgets.BuildContext) SearchViewThemeData {
	if t, ok := widgets.DependOn[*SearchViewTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).SearchViewTheme
}
