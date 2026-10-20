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

// ElevatedButtonThemeData overrides the default style of ElevatedButtons descendant of a ElevatedButtonTheme.
// Only the style is configurable; a property that the style does not set leaves
// the choice to the default style of the button.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonThemeData
type ElevatedButtonThemeData struct {
	// Style overrides the default appearance of the ElevatedButtons.
	Style *ButtonStyle
}

// ElevatedButtonThemeDataOpts are the parameters of NewElevatedButtonThemeData: the fields of the data.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonThemeData.new
type ElevatedButtonThemeDataOpts = ElevatedButtonThemeData

// NewElevatedButtonThemeData creates the data.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonThemeData.new
func NewElevatedButtonThemeData(opts ...ElevatedButtonThemeDataOpts) ElevatedButtonThemeData {
	return firstOpts(opts)
}

// ElevatedButtonThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonThemeData.lerp
func ElevatedButtonThemeDataLerp(a, b ElevatedButtonThemeData, t float64) ElevatedButtonThemeData {
	return ElevatedButtonThemeData{Style: ButtonStyleLerp(a.Style, b.Style, t)}
}

// Equal reports whether the two themes have equal styles.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonThemeData.==
func (d ElevatedButtonThemeData) Equal(o ElevatedButtonThemeData) bool { return d.Style.Equal(o.Style) }

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (ElevatedButtonThemeData) ToStringShort() string { return "ElevatedButtonThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d ElevatedButtonThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the style.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonThemeData.debugFillProperties
func (d ElevatedButtonThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("style", nilIfZero(d.Style)))
}

// ElevatedButtonTheme is an inherited theme that overrides the ElevatedButtonThemeData of the theme
// for the buttons below it.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonTheme
type ElevatedButtonTheme struct {
	widgets.InheritedTheme
	data ElevatedButtonThemeData
}

// NewElevatedButtonTheme creates the theme.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonTheme.new
func NewElevatedButtonTheme(data ElevatedButtonThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *ElevatedButtonTheme {
	return &ElevatedButtonTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonTheme.data
func (w *ElevatedButtonTheme) Data() ElevatedButtonThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonTheme.updateShouldNotify
func (w *ElevatedButtonTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*ElevatedButtonTheme).data)
}

// Wrap makes a ElevatedButtonTheme with the same data around the child.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonTheme.wrap
func (w *ElevatedButtonTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewElevatedButtonTheme(w.data, child)
}

// ElevatedButtonThemeOf returns the data of the closest ElevatedButtonTheme above the context, or the
// elevatedButtonTheme of the closest Theme. The context depends on it.
//
// Flutter: material/elevated_button_theme.dart ElevatedButtonTheme.of
func ElevatedButtonThemeOf(bc widgets.BuildContext) ElevatedButtonThemeData {
	if t, ok := widgets.DependOn[*ElevatedButtonTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).ElevatedButtonTheme
}
