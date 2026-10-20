// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// AppBarThemeData defines default property values for descendant AppBar
// widgets. Every field that is not set (None, nil) leaves the choice to the
// AppBar's own defaults.
//
// Deviation: Flutter's deprecated color parameter (the same as BackgroundColor)
// is not ported (SPEC §12.10).
//
// Flutter: material/app_bar_theme.dart AppBarThemeData
//
//glutter:copywith value
type AppBarThemeData struct {
	// BackgroundColor is the color of the app bar's Material.
	BackgroundColor foundation.Opt[ui.Color]
	// ForegroundColor is the default color of the title and the icons.
	ForegroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the app bar.
	Elevation foundation.Opt[float64]
	// ScrolledUnderElevation is the elevation when content is scrolled under the
	// app bar.
	ScrolledUnderElevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow below the app bar.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the color of the surface tint overlay.
	SurfaceTintColor foundation.Opt[ui.Color]
	// Shape is the shape of the app bar's Material.
	Shape painting.ShapeBorder
	// IconTheme is the color, opacity and size of the icons in the app bar.
	IconTheme foundation.Opt[widgets.IconThemeData]
	// ActionsIconTheme is the color, opacity and size of the icons of the
	// actions.
	ActionsIconTheme foundation.Opt[widgets.IconThemeData]
	// CenterTitle says whether the title is centered.
	CenterTitle foundation.Opt[bool]
	// TitleSpacing is the spacing around the title.
	TitleSpacing foundation.Opt[float64]
	// LeadingWidth is the width of the leading widget.
	LeadingWidth foundation.Opt[float64]
	// ToolbarHeight is the height of the toolbar.
	ToolbarHeight foundation.Opt[float64]
	// ToolbarTextStyle is the default text style of the toolbar.
	ToolbarTextStyle *painting.TextStyle
	// TitleTextStyle is the default text style of the title.
	TitleTextStyle *painting.TextStyle
	// SystemOverlayStyle is the style of the system overlays.
	SystemOverlayStyle foundation.Opt[services.SystemUiOverlayStyle]
	// ActionsPadding is the padding around the actions.
	ActionsPadding painting.EdgeInsetsGeometry
}

// AppBarThemeDataOpts are the parameters of NewAppBarThemeData: the fields of
// the data.
//
// Flutter: material/app_bar_theme.dart AppBarThemeData.new
type AppBarThemeDataOpts = AppBarThemeData

// NewAppBarThemeData creates the data; every field is optional.
//
// Flutter: material/app_bar_theme.dart AppBarThemeData.new
func NewAppBarThemeData(opts ...AppBarThemeDataOpts) AppBarThemeData { return firstOpts(opts) }

// AppBarThemeDataLerp linearly interpolates between two app bar themes. The
// center title flag, the system overlay style and the shape that cannot be
// interpolated switch at t = 0.5.
//
// Flutter: material/app_bar_theme.dart AppBarThemeData.lerp
func AppBarThemeDataLerp(a, b AppBarThemeData, t float64) AppBarThemeData {
	return AppBarThemeData{
		BackgroundColor:        lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		ForegroundColor:        lerpOptColor(a.ForegroundColor, b.ForegroundColor, t),
		Elevation:              lerpOptDouble(a.Elevation, b.Elevation, t),
		ScrolledUnderElevation: lerpOptDouble(a.ScrolledUnderElevation, b.ScrolledUnderElevation, t),
		ShadowColor:            lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor:       lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		Shape:                  painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		IconTheme:              lerpOptIconTheme(a.IconTheme, b.IconTheme, t),
		ActionsIconTheme:       lerpOptIconTheme(a.ActionsIconTheme, b.ActionsIconTheme, t),
		CenterTitle:            lerpFirstOrSecond(a.CenterTitle, b.CenterTitle, t),
		TitleSpacing:           lerpOptDouble(a.TitleSpacing, b.TitleSpacing, t),
		LeadingWidth:           lerpOptDouble(a.LeadingWidth, b.LeadingWidth, t),
		ToolbarHeight:          lerpOptDouble(a.ToolbarHeight, b.ToolbarHeight, t),
		ToolbarTextStyle:       painting.TextStyleLerp(a.ToolbarTextStyle, b.ToolbarTextStyle, t),
		TitleTextStyle:         painting.TextStyleLerp(a.TitleTextStyle, b.TitleTextStyle, t),
		SystemOverlayStyle:     lerpFirstOrSecond(a.SystemOverlayStyle, b.SystemOverlayStyle, t),
		ActionsPadding:         painting.EdgeInsetsGeometryLerp(a.ActionsPadding, b.ActionsPadding, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/app_bar_theme.dart AppBarThemeData.==
func (d AppBarThemeData) Equal(o AppBarThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor && d.ForegroundColor == o.ForegroundColor && d.Elevation == o.Elevation &&
		d.ScrolledUnderElevation == o.ScrolledUnderElevation && d.ShadowColor == o.ShadowColor && d.SurfaceTintColor == o.SurfaceTintColor &&
		foundation.ValuesEqual(d.Shape, o.Shape) && iconThemeOptEqual(d.IconTheme, o.IconTheme) && iconThemeOptEqual(d.ActionsIconTheme, o.ActionsIconTheme) &&
		d.CenterTitle == o.CenterTitle && d.TitleSpacing == o.TitleSpacing && d.LeadingWidth == o.LeadingWidth && d.ToolbarHeight == o.ToolbarHeight &&
		textStyleEqual(d.ToolbarTextStyle, o.ToolbarTextStyle) && textStyleEqual(d.TitleTextStyle, o.TitleTextStyle) &&
		d.SystemOverlayStyle == o.SystemOverlayStyle && foundation.ValuesEqual(d.ActionsPadding, o.ActionsPadding)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (AppBarThemeData) ToStringShort() string { return "AppBarThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d AppBarThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/app_bar_theme.dart AppBarThemeData.debugFillProperties
func (d AppBarThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(colorProp("foregroundColor", d.ForegroundColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(doubleProp("scrolledUnderElevation", d.ScrolledUnderElevation))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("iconTheme", optValue(d.IconTheme)))
	p.Add(objectProp("actionsIconTheme", optValue(d.ActionsIconTheme)))
	p.Add(foundation.NewDiagnosticsPropertyOpt("centerTitle", d.CenterTitle, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	p.Add(doubleProp("titleSpacing", d.TitleSpacing))
	p.Add(doubleProp("leadingWidth", d.LeadingWidth))
	p.Add(doubleProp("toolbarHeight", d.ToolbarHeight))
	p.Add(objectProp("toolbarTextStyle", d.ToolbarTextStyle))
	p.Add(objectProp("titleTextStyle", d.TitleTextStyle))
	p.Add(objectProp("systemOverlayStyle", optValue(d.SystemOverlayStyle)))
	p.Add(objectProp("actionsPadding", d.ActionsPadding))
}

// AppBarTheme is an inherited theme that overrides the AppBarThemeData of the
// theme for the AppBar widgets below it.
//
// Deviation: Flutter's constructor also takes each field of the data
// separately; here the data is the one parameter (SPEC §12.10).
//
// Flutter: material/app_bar_theme.dart AppBarTheme
type AppBarTheme struct {
	widgets.InheritedTheme
	data AppBarThemeData
}

// NewAppBarTheme creates the theme. A nil child is an empty SizedBox, as in
// Flutter.
//
// Flutter: material/app_bar_theme.dart AppBarTheme.new
func NewAppBarTheme(data AppBarThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *AppBarTheme {
	if child == nil {
		child = widgets.NewSizedBox(nil)
	}
	return &AppBarTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/app_bar_theme.dart AppBarTheme.data
func (w *AppBarTheme) Data() AppBarThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/app_bar_theme.dart AppBarTheme.updateShouldNotify
func (w *AppBarTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*AppBarTheme).data)
}

// Wrap makes an AppBarTheme with the same data around the child.
//
// Flutter: material/app_bar_theme.dart AppBarTheme.wrap
func (w *AppBarTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewAppBarTheme(w.data, child)
}

// AppBarThemeOf returns the data of the closest AppBarTheme above the context,
// or the appBarTheme of the closest Theme. The context depends on it.
//
// Flutter: material/app_bar_theme.dart AppBarTheme.of
func AppBarThemeOf(bc widgets.BuildContext) AppBarThemeData {
	if t, ok := widgets.DependOn[*AppBarTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).AppBarTheme
}

// optValue returns the value of an Opt as an any, or nil for None, which is
// what a DiagnosticsProperty of a nullable value takes.
func optValue[T any](o foundation.Opt[T]) any {
	if v, ok := o.Get(); ok {
		return v
	}
	return nil
}
