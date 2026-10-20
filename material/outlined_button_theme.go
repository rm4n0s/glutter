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

// OutlinedButtonThemeData overrides the default style of OutlinedButtons descendant of a OutlinedButtonTheme.
// Only the style is configurable; a property that the style does not set leaves
// the choice to the default style of the button.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonThemeData
type OutlinedButtonThemeData struct {
	// Style overrides the default appearance of the OutlinedButtons.
	Style *ButtonStyle
}

// OutlinedButtonThemeDataOpts are the parameters of NewOutlinedButtonThemeData: the fields of the data.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonThemeData.new
type OutlinedButtonThemeDataOpts = OutlinedButtonThemeData

// NewOutlinedButtonThemeData creates the data.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonThemeData.new
func NewOutlinedButtonThemeData(opts ...OutlinedButtonThemeDataOpts) OutlinedButtonThemeData {
	return firstOpts(opts)
}

// OutlinedButtonThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonThemeData.lerp
func OutlinedButtonThemeDataLerp(a, b OutlinedButtonThemeData, t float64) OutlinedButtonThemeData {
	return OutlinedButtonThemeData{Style: ButtonStyleLerp(a.Style, b.Style, t)}
}

// Equal reports whether the two themes have equal styles.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonThemeData.==
func (d OutlinedButtonThemeData) Equal(o OutlinedButtonThemeData) bool { return d.Style.Equal(o.Style) }

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (OutlinedButtonThemeData) ToStringShort() string { return "OutlinedButtonThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d OutlinedButtonThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the style.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonThemeData.debugFillProperties
func (d OutlinedButtonThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("style", nilIfZero(d.Style)))
}

// OutlinedButtonTheme is an inherited theme that overrides the OutlinedButtonThemeData of the theme
// for the buttons below it.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonTheme
type OutlinedButtonTheme struct {
	widgets.InheritedTheme
	data OutlinedButtonThemeData
}

// NewOutlinedButtonTheme creates the theme.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonTheme.new
func NewOutlinedButtonTheme(data OutlinedButtonThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *OutlinedButtonTheme {
	return &OutlinedButtonTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonTheme.data
func (w *OutlinedButtonTheme) Data() OutlinedButtonThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonTheme.updateShouldNotify
func (w *OutlinedButtonTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*OutlinedButtonTheme).data)
}

// Wrap makes a OutlinedButtonTheme with the same data around the child.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonTheme.wrap
func (w *OutlinedButtonTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewOutlinedButtonTheme(w.data, child)
}

// OutlinedButtonThemeOf returns the data of the closest OutlinedButtonTheme above the context, or the
// outlinedButtonTheme of the closest Theme. The context depends on it.
//
// Flutter: material/outlined_button_theme.dart OutlinedButtonTheme.of
func OutlinedButtonThemeOf(bc widgets.BuildContext) OutlinedButtonThemeData {
	if t, ok := widgets.DependOn[*OutlinedButtonTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).OutlinedButtonTheme
}
