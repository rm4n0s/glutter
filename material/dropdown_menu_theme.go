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

// DropdownMenuThemeData defines the default appearance of the DropdownMenu
// widgets below it. Every field that is not set leaves the choice to the
// widget's own defaults.
//
// Deviation: Dart's inputDecorationTheme can be an InputDecorationTheme widget
// or its data; here it is the data (SPEC §12.10).
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuThemeData
//
//glutter:copywith value
type DropdownMenuThemeData struct {
	// TextStyle is the text style of the text field and of the entries.
	TextStyle *painting.TextStyle
	// InputDecorationTheme is the decoration theme of the text field.
	InputDecorationTheme *InputDecorationThemeData
	// MenuStyle is the style of the menu.
	MenuStyle *MenuStyle
	// DisabledColor is the color of the text and the icons when the menu is
	// disabled.
	DisabledColor foundation.Opt[ui.Color]
}

// DropdownMenuThemeDataOpts are the parameters of NewDropdownMenuThemeData: the
// fields of the data.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuThemeData.new
type DropdownMenuThemeDataOpts = DropdownMenuThemeData

// NewDropdownMenuThemeData creates the data; every field is optional.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuThemeData.new
func NewDropdownMenuThemeData(opts ...DropdownMenuThemeDataOpts) DropdownMenuThemeData {
	return firstOpts(opts)
}

// DropdownMenuThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuThemeData.lerp
func DropdownMenuThemeDataLerp(a, b DropdownMenuThemeData, t float64) DropdownMenuThemeData {
	return DropdownMenuThemeData{
		TextStyle:            painting.TextStyleLerp(a.TextStyle, b.TextStyle, t),
		InputDecorationTheme: lerpFirstOrSecond(a.InputDecorationTheme, b.InputDecorationTheme, t),
		MenuStyle:            MenuStyleLerp(a.MenuStyle, b.MenuStyle, t),
		DisabledColor:        lerpOptColor(a.DisabledColor, b.DisabledColor, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuThemeData.==
func (d DropdownMenuThemeData) Equal(o DropdownMenuThemeData) bool {
	sameDecoration := d.InputDecorationTheme == o.InputDecorationTheme ||
		d.InputDecorationTheme != nil && o.InputDecorationTheme != nil && d.InputDecorationTheme.Equal(*o.InputDecorationTheme)
	return foundation.ValuesEqual(d.TextStyle, o.TextStyle) && sameDecoration &&
		d.MenuStyle.Equal(o.MenuStyle) && d.DisabledColor == o.DisabledColor
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (DropdownMenuThemeData) ToStringShort() string { return "DropdownMenuThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d DropdownMenuThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuThemeData.debugFillProperties
func (d DropdownMenuThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("textStyle", ptrOrNil(d.TextStyle)))
	p.Add(objectProp("inputDecorationThemeData", ptrOrNil(d.InputDecorationTheme)))
	p.Add(objectProp("menuStyle", nilIfZero(d.MenuStyle)))
	p.Add(painting.NewColorProperty("disabledColor", d.DisabledColor, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
}

// DropdownMenuTheme is an inherited theme that overrides the
// DropdownMenuThemeData of the theme for the DropdownMenu widgets below it.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuTheme
type DropdownMenuTheme struct {
	widgets.InheritedTheme
	data DropdownMenuThemeData
}

// NewDropdownMenuTheme creates the theme.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuTheme.new
func NewDropdownMenuTheme(data DropdownMenuThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *DropdownMenuTheme {
	return &DropdownMenuTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuTheme.data
func (w *DropdownMenuTheme) Data() DropdownMenuThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuTheme.updateShouldNotify
func (w *DropdownMenuTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*DropdownMenuTheme).data)
}

// Wrap makes a DropdownMenuTheme with the same data around the child.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuTheme.wrap
func (w *DropdownMenuTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewDropdownMenuTheme(w.data, child)
}

// DropdownMenuThemeOf returns the data of the closest DropdownMenuTheme above
// the context, or the dropdownMenuTheme of the closest Theme. The context
// depends on it.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuTheme.of
func DropdownMenuThemeOf(bc widgets.BuildContext) DropdownMenuThemeData {
	if d, ok := DropdownMenuThemeMaybeOf(bc); ok {
		return d
	}
	return ThemeOf(bc).DropdownMenuTheme
}

// DropdownMenuThemeMaybeOf returns the data of the closest DropdownMenuTheme
// above the context, if there is one. The context depends on it.
//
// Flutter: material/dropdown_menu_theme.dart DropdownMenuTheme.maybeOf
func DropdownMenuThemeMaybeOf(bc widgets.BuildContext) (DropdownMenuThemeData, bool) {
	if t, ok := widgets.DependOn[*DropdownMenuTheme](bc); ok {
		return t.data, true
	}
	return DropdownMenuThemeData{}, false
}
