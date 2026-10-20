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

// PopupMenuThemeData defines default property values for descendant PopupMenuButton widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/popup_menu_theme.dart PopupMenuThemeData
//
//glutter:copywith value
type PopupMenuThemeData struct {
	// Color is the background color of the popup menus.
	Color foundation.Opt[ui.Color]
	// Shape is the shape of the popup menus.
	Shape painting.ShapeBorder
	// MenuPadding is the padding of the menu panel.
	MenuPadding painting.EdgeInsetsGeometry
	// Elevation is the z-coordinate of the popup menus.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow of the popup menus.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the popup menus.
	SurfaceTintColor foundation.Opt[ui.Color]
	// TextStyle is the text style of the items of the popup menus, in Material 2.
	TextStyle *painting.TextStyle
	// LabelTextStyle is the text style of the items of the popup menus, in each state, in Material 3.
	LabelTextStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// EnableFeedback says whether detected gestures give acoustic and haptic feedback.
	EnableFeedback foundation.Opt[bool]
	// MouseCursor is the cursor over the items of the popup menus, in each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// Position is where the menu of a PopupMenuButton opens relative to the button.
	Position PopupMenuPosition
	// IconColor is the color of the icon of the PopupMenuButton.
	IconColor foundation.Opt[ui.Color]
	// IconSize is the size of the icon of the PopupMenuButton.
	IconSize foundation.Opt[float64]
}

// PopupMenuThemeDataOpts are the parameters of NewPopupMenuThemeData: the fields of the data.
//
// Flutter: material/popup_menu_theme.dart PopupMenuThemeData.new
type PopupMenuThemeDataOpts = PopupMenuThemeData

// NewPopupMenuThemeData creates the data; every field is optional.
//
// Flutter: material/popup_menu_theme.dart PopupMenuThemeData.new
func NewPopupMenuThemeData(opts ...PopupMenuThemeDataOpts) PopupMenuThemeData {
	return firstOpts(opts)
}

// PopupMenuThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/popup_menu_theme.dart PopupMenuThemeData.lerp
func PopupMenuThemeDataLerp(a, b PopupMenuThemeData, t float64) PopupMenuThemeData {
	return PopupMenuThemeData{
		Color:            lerpOptColor(a.Color, b.Color, t),
		Shape:            painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		MenuPadding:      painting.EdgeInsetsGeometryLerp(a.MenuPadding, b.MenuPadding, t),
		Elevation:        lerpOptDouble(a.Elevation, b.Elevation, t),
		ShadowColor:      lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor: lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		TextStyle:        painting.TextStyleLerp(a.TextStyle, b.TextStyle, t),
		LabelTextStyle:   lerpStateWith(a.LabelTextStyle, b.LabelTextStyle, t, lerpOptTextStyle),
		EnableFeedback:   lerpFirstOrSecond(a.EnableFeedback, b.EnableFeedback, t),
		MouseCursor:      lerpFirstOrSecond(a.MouseCursor, b.MouseCursor, t),
		Position:         lerpFirstOrSecond(a.Position, b.Position, t),
		IconColor:        lerpOptColor(a.IconColor, b.IconColor, t),
		IconSize:         lerpOptDouble(a.IconSize, b.IconSize, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/popup_menu_theme.dart PopupMenuThemeData.==
func (d PopupMenuThemeData) Equal(o PopupMenuThemeData) bool {
	return d.Color == o.Color &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		foundation.ValuesEqual(d.MenuPadding, o.MenuPadding) &&
		d.Elevation == o.Elevation &&
		d.ShadowColor == o.ShadowColor &&
		d.SurfaceTintColor == o.SurfaceTintColor &&
		textStyleEqual(d.TextStyle, o.TextStyle) &&
		foundation.ValuesEqual(d.LabelTextStyle, o.LabelTextStyle) &&
		d.EnableFeedback == o.EnableFeedback &&
		foundation.ValuesEqual(d.MouseCursor, o.MouseCursor) &&
		d.Position == o.Position &&
		d.IconColor == o.IconColor &&
		d.IconSize == o.IconSize
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (PopupMenuThemeData) ToStringShort() string { return "PopupMenuThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d PopupMenuThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/popup_menu_theme.dart PopupMenuThemeData.debugFillProperties
func (d PopupMenuThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("color", d.Color))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("menuPadding", d.MenuPadding))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(objectProp("textStyle", d.TextStyle))
	p.Add(objectProp("labelTextStyle", propOrNil(d.LabelTextStyle)))
	p.Add(objectProp("enableFeedback", optValue(d.EnableFeedback)))
	p.Add(objectProp("mouseCursor", propOrNil(d.MouseCursor)))
	p.Add(objectProp("position", nilIfZero(d.Position)))
	p.Add(colorProp("iconColor", d.IconColor))
	p.Add(doubleProp("iconSize", d.IconSize))
}

// PopupMenuTheme is an inherited theme that overrides the PopupMenuThemeData of the theme for the
// widgets below it.
//
// Flutter: material/popup_menu_theme.dart PopupMenuTheme
type PopupMenuTheme struct {
	widgets.InheritedTheme
	data PopupMenuThemeData
}

// NewPopupMenuTheme creates the theme.
//
// Flutter: material/popup_menu_theme.dart PopupMenuTheme.new
func NewPopupMenuTheme(data PopupMenuThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *PopupMenuTheme {
	return &PopupMenuTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/popup_menu_theme.dart PopupMenuTheme.data
func (w *PopupMenuTheme) Data() PopupMenuThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/popup_menu_theme.dart PopupMenuTheme.updateShouldNotify
func (w *PopupMenuTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*PopupMenuTheme).data)
}

// Wrap makes a PopupMenuTheme with the same data around the child.
//
// Flutter: material/popup_menu_theme.dart PopupMenuTheme.wrap
func (w *PopupMenuTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewPopupMenuTheme(w.data, child)
}

// PopupMenuThemeOf returns the data of the closest PopupMenuTheme above the context, or the
// popupMenuTheme of the closest Theme. The context depends on it.
//
// Flutter: material/popup_menu_theme.dart PopupMenuTheme.of
func PopupMenuThemeOf(bc widgets.BuildContext) PopupMenuThemeData {
	if t, ok := widgets.DependOn[*PopupMenuTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).PopupMenuTheme
}

// PopupMenuPosition is where the menu of a PopupMenuButton opens relative to
// the button.
//
// Flutter: material/popup_menu_theme.dart PopupMenuPosition
type PopupMenuPosition int

const (
	// PopupMenuPositionUnspecified is the zero value: over.
	PopupMenuPositionUnspecified PopupMenuPosition = iota
	// PopupMenuPositionOver makes the menu appear over the button that created it.
	//
	// Flutter: material/popup_menu_theme.dart PopupMenuPosition.over
	PopupMenuPositionOver
	// PopupMenuPositionUnder makes the menu appear under the button that created
	// it.
	//
	// Flutter: material/popup_menu_theme.dart PopupMenuPosition.under
	PopupMenuPositionUnder
)

// Name is the name of the value, as Dart's Enum.name.
func (p PopupMenuPosition) Name() string { return enumName(int(p), "over", "under") }

// String is "PopupMenuPosition." and the name.
func (p PopupMenuPosition) String() string { return "PopupMenuPosition." + p.Name() }
