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

// SegmentedButtonThemeData defines default property values for descendant SegmentedButton widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonThemeData
//
//glutter:copywith value
type SegmentedButtonThemeData struct {
	// Style customizes the appearance of the buttons, the segments being TextButtons.
	Style *ButtonStyle
	// SelectedIcon is the icon that is shown in the selected segments.
	SelectedIcon widgets.Widget
}

// SegmentedButtonThemeDataOpts are the parameters of NewSegmentedButtonThemeData: the fields of the data.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonThemeData.new
type SegmentedButtonThemeDataOpts = SegmentedButtonThemeData

// NewSegmentedButtonThemeData creates the data; every field is optional.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonThemeData.new
func NewSegmentedButtonThemeData(opts ...SegmentedButtonThemeDataOpts) SegmentedButtonThemeData {
	return firstOpts(opts)
}

// SegmentedButtonThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonThemeData.lerp
func SegmentedButtonThemeDataLerp(a, b SegmentedButtonThemeData, t float64) SegmentedButtonThemeData {
	return SegmentedButtonThemeData{
		Style:        ButtonStyleLerp(a.Style, b.Style, t),
		SelectedIcon: lerpFirstOrSecond(a.SelectedIcon, b.SelectedIcon, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonThemeData.==
func (d SegmentedButtonThemeData) Equal(o SegmentedButtonThemeData) bool {
	return d.Style.Equal(o.Style) &&
		d.SelectedIcon == o.SelectedIcon
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (SegmentedButtonThemeData) ToStringShort() string { return "SegmentedButtonThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d SegmentedButtonThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonThemeData.debugFillProperties
func (d SegmentedButtonThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("style", nilIfZero(d.Style)))
	p.Add(objectProp("selectedIcon", d.SelectedIcon))
}

// SegmentedButtonTheme is an inherited theme that overrides the SegmentedButtonThemeData of the theme for the
// widgets below it.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonTheme
type SegmentedButtonTheme struct {
	widgets.InheritedTheme
	data SegmentedButtonThemeData
}

// NewSegmentedButtonTheme creates the theme.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonTheme.new
func NewSegmentedButtonTheme(data SegmentedButtonThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *SegmentedButtonTheme {
	return &SegmentedButtonTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonTheme.data
func (w *SegmentedButtonTheme) Data() SegmentedButtonThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonTheme.updateShouldNotify
func (w *SegmentedButtonTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*SegmentedButtonTheme).data)
}

// Wrap makes a SegmentedButtonTheme with the same data around the child.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonTheme.wrap
func (w *SegmentedButtonTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewSegmentedButtonTheme(w.data, child)
}

// SegmentedButtonThemeOf returns the data of the closest SegmentedButtonTheme above the context, or the
// segmentedButtonTheme of the closest Theme. The context depends on it.
//
// Flutter: material/segmented_button_theme.dart SegmentedButtonTheme.of
func SegmentedButtonThemeOf(bc widgets.BuildContext) SegmentedButtonThemeData {
	if t, ok := widgets.DependOn[*SegmentedButtonTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).SegmentedButtonTheme
}
