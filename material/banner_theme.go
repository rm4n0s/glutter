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

// MaterialBannerThemeData defines the default properties of MaterialBanner
// widgets. Every field that is not set leaves the choice to the MaterialBanner's
// own defaults.
//
// Flutter: material/banner_theme.dart MaterialBannerThemeData
//
//glutter:copywith value
type MaterialBannerThemeData struct {
	// BackgroundColor is the background color of the banner.
	BackgroundColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint color of the banner.
	SurfaceTintColor foundation.Opt[ui.Color]
	// ShadowColor is the color of the shadow below the banner.
	ShadowColor foundation.Opt[ui.Color]
	// DividerColor is the color of the divider below the banner.
	DividerColor foundation.Opt[ui.Color]
	// ContentTextStyle is the style of the text of the content.
	ContentTextStyle *painting.TextStyle
	// Elevation is the elevation of the banner; the default is 0.
	Elevation foundation.Opt[float64]
	// Padding is the padding around the content.
	Padding painting.EdgeInsetsGeometry
	// LeadingPadding is the padding around the leading widget.
	LeadingPadding painting.EdgeInsetsGeometry
}

// MaterialBannerThemeDataOpts are the parameters of NewMaterialBannerThemeData:
// the fields of the data.
//
// Flutter: material/banner_theme.dart MaterialBannerThemeData.new
type MaterialBannerThemeDataOpts = MaterialBannerThemeData

// NewMaterialBannerThemeData creates the data; every field is optional.
//
// Flutter: material/banner_theme.dart MaterialBannerThemeData.new
func NewMaterialBannerThemeData(opts ...MaterialBannerThemeDataOpts) MaterialBannerThemeData {
	return firstOpts(opts)
}

// MaterialBannerThemeDataLerp linearly interpolates between two banner themes.
//
// Flutter: material/banner_theme.dart MaterialBannerThemeData.lerp
func MaterialBannerThemeDataLerp(a, b MaterialBannerThemeData, t float64) MaterialBannerThemeData {
	return MaterialBannerThemeData{
		BackgroundColor:  lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		SurfaceTintColor: lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		ShadowColor:      lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		DividerColor:     lerpOptColor(a.DividerColor, b.DividerColor, t),
		ContentTextStyle: painting.TextStyleLerp(a.ContentTextStyle, b.ContentTextStyle, t),
		Elevation:        lerpOptDouble(a.Elevation, b.Elevation, t),
		Padding:          painting.EdgeInsetsGeometryLerp(a.Padding, b.Padding, t),
		LeadingPadding:   painting.EdgeInsetsGeometryLerp(a.LeadingPadding, b.LeadingPadding, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/banner_theme.dart MaterialBannerThemeData.==
func (d MaterialBannerThemeData) Equal(o MaterialBannerThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor && d.SurfaceTintColor == o.SurfaceTintColor && d.ShadowColor == o.ShadowColor &&
		d.DividerColor == o.DividerColor && textStyleEqual(d.ContentTextStyle, o.ContentTextStyle) && d.Elevation == o.Elevation &&
		foundation.ValuesEqual(d.Padding, o.Padding) && foundation.ValuesEqual(d.LeadingPadding, o.LeadingPadding)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (MaterialBannerThemeData) ToStringShort() string { return "MaterialBannerThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d MaterialBannerThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/banner_theme.dart MaterialBannerThemeData.debugFillProperties
func (d MaterialBannerThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(colorProp("dividerColor", d.DividerColor))
	p.Add(objectProp("contentTextStyle", d.ContentTextStyle))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(objectProp("padding", d.Padding))
	p.Add(objectProp("leadingPadding", d.LeadingPadding))
}

// MaterialBannerTheme is an inherited theme that overrides the
// MaterialBannerThemeData of the theme for the MaterialBanner widgets below it.
//
// Flutter: material/banner_theme.dart MaterialBannerTheme
type MaterialBannerTheme struct {
	widgets.InheritedTheme
	data MaterialBannerThemeData
}

// NewMaterialBannerTheme creates the theme.
//
// Flutter: material/banner_theme.dart MaterialBannerTheme.new
func NewMaterialBannerTheme(data MaterialBannerThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *MaterialBannerTheme {
	return &MaterialBannerTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/banner_theme.dart MaterialBannerTheme.data
func (w *MaterialBannerTheme) Data() MaterialBannerThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/banner_theme.dart MaterialBannerTheme.updateShouldNotify
func (w *MaterialBannerTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*MaterialBannerTheme).data)
}

// Wrap makes a MaterialBannerTheme with the same data around the child.
//
// Flutter: material/banner_theme.dart MaterialBannerTheme.wrap
func (w *MaterialBannerTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewMaterialBannerTheme(w.data, child)
}

// MaterialBannerThemeOf returns the data of the closest MaterialBannerTheme above
// the context, or the bannerTheme of the closest Theme. The context depends on
// it.
//
// Flutter: material/banner_theme.dart MaterialBannerTheme.of
func MaterialBannerThemeOf(bc widgets.BuildContext) MaterialBannerThemeData {
	if t, ok := widgets.DependOn[*MaterialBannerTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).BannerTheme
}
