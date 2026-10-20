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

// FilledButtonThemeData overrides the default style of FilledButtons descendant of a FilledButtonTheme.
// Only the style is configurable; a property that the style does not set leaves
// the choice to the default style of the button.
//
// Flutter: material/filled_button_theme.dart FilledButtonThemeData
type FilledButtonThemeData struct {
	// Style overrides the default appearance of the FilledButtons.
	Style *ButtonStyle
}

// FilledButtonThemeDataOpts are the parameters of NewFilledButtonThemeData: the fields of the data.
//
// Flutter: material/filled_button_theme.dart FilledButtonThemeData.new
type FilledButtonThemeDataOpts = FilledButtonThemeData

// NewFilledButtonThemeData creates the data.
//
// Flutter: material/filled_button_theme.dart FilledButtonThemeData.new
func NewFilledButtonThemeData(opts ...FilledButtonThemeDataOpts) FilledButtonThemeData {
	return firstOpts(opts)
}

// FilledButtonThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/filled_button_theme.dart FilledButtonThemeData.lerp
func FilledButtonThemeDataLerp(a, b FilledButtonThemeData, t float64) FilledButtonThemeData {
	return FilledButtonThemeData{Style: ButtonStyleLerp(a.Style, b.Style, t)}
}

// Equal reports whether the two themes have equal styles.
//
// Flutter: material/filled_button_theme.dart FilledButtonThemeData.==
func (d FilledButtonThemeData) Equal(o FilledButtonThemeData) bool { return d.Style.Equal(o.Style) }

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (FilledButtonThemeData) ToStringShort() string { return "FilledButtonThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d FilledButtonThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the style.
//
// Flutter: material/filled_button_theme.dart FilledButtonThemeData.debugFillProperties
func (d FilledButtonThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("style", nilIfZero(d.Style)))
}

// FilledButtonTheme is an inherited theme that overrides the FilledButtonThemeData of the theme
// for the buttons below it.
//
// Flutter: material/filled_button_theme.dart FilledButtonTheme
type FilledButtonTheme struct {
	widgets.InheritedTheme
	data FilledButtonThemeData
}

// NewFilledButtonTheme creates the theme.
//
// Flutter: material/filled_button_theme.dart FilledButtonTheme.new
func NewFilledButtonTheme(data FilledButtonThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *FilledButtonTheme {
	return &FilledButtonTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/filled_button_theme.dart FilledButtonTheme.data
func (w *FilledButtonTheme) Data() FilledButtonThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/filled_button_theme.dart FilledButtonTheme.updateShouldNotify
func (w *FilledButtonTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*FilledButtonTheme).data)
}

// Wrap makes a FilledButtonTheme with the same data around the child.
//
// Flutter: material/filled_button_theme.dart FilledButtonTheme.wrap
func (w *FilledButtonTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewFilledButtonTheme(w.data, child)
}

// FilledButtonThemeOf returns the data of the closest FilledButtonTheme above the context, or the
// filledButtonTheme of the closest Theme. The context depends on it.
//
// Flutter: material/filled_button_theme.dart FilledButtonTheme.of
func FilledButtonThemeOf(bc widgets.BuildContext) FilledButtonThemeData {
	if t, ok := widgets.DependOn[*FilledButtonTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).FilledButtonTheme
}
