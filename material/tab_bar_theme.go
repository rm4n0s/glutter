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

// TabBarThemeData defines default property values for descendant TabBar widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/tab_bar_theme.dart TabBarThemeData
//
//glutter:copywith value
type TabBarThemeData struct {
	// Indicator is the decoration of the indicator of the selected tab.
	Indicator painting.Decoration
	// IndicatorColor is the color of the indicator of the selected tab.
	IndicatorColor foundation.Opt[ui.Color]
	// IndicatorSize is how the bounds of the indicator are computed.
	IndicatorSize TabBarIndicatorSize
	// DividerColor is the color of the divider below the bar.
	DividerColor foundation.Opt[ui.Color]
	// DividerHeight is the height of the divider below the bar.
	DividerHeight foundation.Opt[float64]
	// LabelColor is the color of the label of the selected tab.
	LabelColor foundation.Opt[ui.Color]
	// LabelPadding is the padding around the label of each tab.
	LabelPadding painting.EdgeInsetsGeometry
	// LabelStyle is the text style of the label of the selected tab.
	LabelStyle *painting.TextStyle
	// UnselectedLabelColor is the color of the labels of the tabs that are not selected.
	UnselectedLabelColor foundation.Opt[ui.Color]
	// UnselectedLabelStyle is the text style of the labels of the tabs that are not selected.
	UnselectedLabelStyle *painting.TextStyle
	// OverlayColor is the color of the highlight and the splash of a tab, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashFactory creates the splashes of the tabs.
	SplashFactory InteractiveInkFeatureFactory
	// MouseCursor is the cursor over a tab, in each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// TabAlignment is how the tabs are aligned horizontally.
	TabAlignment TabAlignment
	// TextScaler scales the text of the labels.
	TextScaler painting.TextScaler
	// IndicatorAnimation is how the indicator moves when the selected tab changes.
	IndicatorAnimation TabIndicatorAnimation
	// SplashBorderRadius is the radius of the splash and the highlight of a tab.
	SplashBorderRadius foundation.Opt[painting.BorderRadius]
}

// TabBarThemeDataOpts are the parameters of NewTabBarThemeData: the fields of the data.
//
// Flutter: material/tab_bar_theme.dart TabBarThemeData.new
type TabBarThemeDataOpts = TabBarThemeData

// NewTabBarThemeData creates the data; every field is optional.
//
// Flutter: material/tab_bar_theme.dart TabBarThemeData.new
func NewTabBarThemeData(opts ...TabBarThemeDataOpts) TabBarThemeData {
	return firstOpts(opts)
}

// TabBarThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/tab_bar_theme.dart TabBarThemeData.lerp
func TabBarThemeDataLerp(a, b TabBarThemeData, t float64) TabBarThemeData {
	return TabBarThemeData{
		Indicator:            painting.DecorationLerp(a.Indicator, b.Indicator, t),
		IndicatorColor:       lerpOptColor(a.IndicatorColor, b.IndicatorColor, t),
		IndicatorSize:        lerpFirstOrSecond(a.IndicatorSize, b.IndicatorSize, t),
		DividerColor:         lerpOptColor(a.DividerColor, b.DividerColor, t),
		DividerHeight:        lerpOptDouble(a.DividerHeight, b.DividerHeight, t),
		LabelColor:           lerpOptColor(a.LabelColor, b.LabelColor, t),
		LabelPadding:         painting.EdgeInsetsGeometryLerp(a.LabelPadding, b.LabelPadding, t),
		LabelStyle:           painting.TextStyleLerp(a.LabelStyle, b.LabelStyle, t),
		UnselectedLabelColor: lerpOptColor(a.UnselectedLabelColor, b.UnselectedLabelColor, t),
		UnselectedLabelStyle: painting.TextStyleLerp(a.UnselectedLabelStyle, b.UnselectedLabelStyle, t),
		OverlayColor:         lerpStateOptColor(a.OverlayColor, b.OverlayColor, t),
		SplashFactory:        lerpFirstOrSecond(a.SplashFactory, b.SplashFactory, t),
		MouseCursor:          lerpFirstOrSecond(a.MouseCursor, b.MouseCursor, t),
		TabAlignment:         lerpFirstOrSecond(a.TabAlignment, b.TabAlignment, t),
		TextScaler:           lerpFirstOrSecond(a.TextScaler, b.TextScaler, t),
		IndicatorAnimation:   lerpFirstOrSecond(a.IndicatorAnimation, b.IndicatorAnimation, t),
		SplashBorderRadius:   lerpOptBorderRadius(a.SplashBorderRadius, b.SplashBorderRadius, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/tab_bar_theme.dart TabBarThemeData.==
func (d TabBarThemeData) Equal(o TabBarThemeData) bool {
	return foundation.ValuesEqual(d.Indicator, o.Indicator) &&
		d.IndicatorColor == o.IndicatorColor &&
		d.IndicatorSize == o.IndicatorSize &&
		d.DividerColor == o.DividerColor &&
		d.DividerHeight == o.DividerHeight &&
		d.LabelColor == o.LabelColor &&
		foundation.ValuesEqual(d.LabelPadding, o.LabelPadding) &&
		textStyleEqual(d.LabelStyle, o.LabelStyle) &&
		d.UnselectedLabelColor == o.UnselectedLabelColor &&
		textStyleEqual(d.UnselectedLabelStyle, o.UnselectedLabelStyle) &&
		foundation.ValuesEqual(d.OverlayColor, o.OverlayColor) &&
		d.SplashFactory == o.SplashFactory &&
		foundation.ValuesEqual(d.MouseCursor, o.MouseCursor) &&
		d.TabAlignment == o.TabAlignment &&
		foundation.ValuesEqual(d.TextScaler, o.TextScaler) &&
		d.IndicatorAnimation == o.IndicatorAnimation &&
		d.SplashBorderRadius == o.SplashBorderRadius
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (TabBarThemeData) ToStringShort() string { return "TabBarThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d TabBarThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/tab_bar_theme.dart TabBarThemeData.debugFillProperties
func (d TabBarThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("indicator", d.Indicator))
	p.Add(colorProp("indicatorColor", d.IndicatorColor))
	p.Add(objectProp("indicatorSize", nilIfZero(d.IndicatorSize)))
	p.Add(colorProp("dividerColor", d.DividerColor))
	p.Add(doubleProp("dividerHeight", d.DividerHeight))
	p.Add(colorProp("labelColor", d.LabelColor))
	p.Add(objectProp("labelPadding", d.LabelPadding))
	p.Add(objectProp("labelStyle", d.LabelStyle))
	p.Add(colorProp("unselectedLabelColor", d.UnselectedLabelColor))
	p.Add(objectProp("unselectedLabelStyle", d.UnselectedLabelStyle))
	p.Add(objectProp("overlayColor", propOrNil(d.OverlayColor)))
	p.Add(objectProp("splashFactory", d.SplashFactory))
	p.Add(objectProp("mouseCursor", propOrNil(d.MouseCursor)))
	p.Add(objectProp("tabAlignment", nilIfZero(d.TabAlignment)))
	p.Add(objectProp("textScaler", d.TextScaler))
	p.Add(objectProp("indicatorAnimation", nilIfZero(d.IndicatorAnimation)))
	p.Add(objectProp("splashBorderRadius", optValue(d.SplashBorderRadius)))
}

// TabBarTheme is an inherited theme that overrides the TabBarThemeData of the theme for the
// widgets below it.
//
// Flutter: material/tab_bar_theme.dart TabBarTheme
type TabBarTheme struct {
	widgets.InheritedTheme
	data TabBarThemeData
}

// NewTabBarTheme creates the theme.
//
// Flutter: material/tab_bar_theme.dart TabBarTheme.new
func NewTabBarTheme(data TabBarThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *TabBarTheme {
	return &TabBarTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/tab_bar_theme.dart TabBarTheme.data
func (w *TabBarTheme) Data() TabBarThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/tab_bar_theme.dart TabBarTheme.updateShouldNotify
func (w *TabBarTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*TabBarTheme).data)
}

// Wrap makes a TabBarTheme with the same data around the child.
//
// Flutter: material/tab_bar_theme.dart TabBarTheme.wrap
func (w *TabBarTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewTabBarTheme(w.data, child)
}

// TabBarThemeOf returns the data of the closest TabBarTheme above the context, or the
// tabBarTheme of the closest Theme. The context depends on it.
//
// Flutter: material/tab_bar_theme.dart TabBarTheme.of
func TabBarThemeOf(bc widgets.BuildContext) TabBarThemeData {
	if t, ok := widgets.DependOn[*TabBarTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).TabBarTheme
}
