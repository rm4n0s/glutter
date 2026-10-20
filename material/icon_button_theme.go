// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/widgets"
)

// IconButtonThemeData overrides the default style of IconButtons descendant of a IconButtonTheme.
// Only the style is configurable; a property that the style does not set leaves
// the choice to the default style of the button.
//
// Flutter: material/icon_button_theme.dart IconButtonThemeData
type IconButtonThemeData struct {
	// Style overrides the default appearance of the IconButtons.
	Style *ButtonStyle
}

// IconButtonThemeDataOpts are the parameters of NewIconButtonThemeData: the fields of the data.
//
// Flutter: material/icon_button_theme.dart IconButtonThemeData.new
type IconButtonThemeDataOpts = IconButtonThemeData

// NewIconButtonThemeData creates the data.
//
// Flutter: material/icon_button_theme.dart IconButtonThemeData.new
func NewIconButtonThemeData(opts ...IconButtonThemeDataOpts) IconButtonThemeData {
	return firstOpts(opts)
}

// IconButtonThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/icon_button_theme.dart IconButtonThemeData.lerp
func IconButtonThemeDataLerp(a, b IconButtonThemeData, t float64) IconButtonThemeData {
	return IconButtonThemeData{Style: ButtonStyleLerp(a.Style, b.Style, t)}
}

// Equal reports whether the two themes have equal styles.
//
// Flutter: material/icon_button_theme.dart IconButtonThemeData.==
func (d IconButtonThemeData) Equal(o IconButtonThemeData) bool { return d.Style.Equal(o.Style) }

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (IconButtonThemeData) ToStringShort() string { return "IconButtonThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d IconButtonThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the style.
//
// Flutter: material/icon_button_theme.dart IconButtonThemeData.debugFillProperties
func (d IconButtonThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("style", nilIfZero(d.Style)))
}

// IconButtonTheme is an inherited theme that overrides the IconButtonThemeData of the theme
// for the buttons below it.
//
// Flutter: material/icon_button_theme.dart IconButtonTheme
type IconButtonTheme struct {
	widgets.InheritedTheme
	data IconButtonThemeData
}

// NewIconButtonTheme creates the theme.
//
// Flutter: material/icon_button_theme.dart IconButtonTheme.new
func NewIconButtonTheme(data IconButtonThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *IconButtonTheme {
	return &IconButtonTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/icon_button_theme.dart IconButtonTheme.data
func (w *IconButtonTheme) Data() IconButtonThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/icon_button_theme.dart IconButtonTheme.updateShouldNotify
func (w *IconButtonTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*IconButtonTheme).data)
}

// Wrap makes a IconButtonTheme with the same data around the child.
//
// Flutter: material/icon_button_theme.dart IconButtonTheme.wrap
func (w *IconButtonTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewIconButtonTheme(w.data, child)
}

// IconButtonThemeOf returns the data of the closest IconButtonTheme above the context, or the
// iconButtonTheme of the closest Theme. The context depends on it.
//
// Flutter: material/icon_button_theme.dart IconButtonTheme.of
func IconButtonThemeOf(bc widgets.BuildContext) IconButtonThemeData {
	if t, ok := widgets.DependOn[*IconButtonTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).IconButtonTheme
}
