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

// TextButtonThemeData overrides the default style of TextButtons descendant of a TextButtonTheme.
// Only the style is configurable; a property that the style does not set leaves
// the choice to the default style of the button.
//
// Flutter: material/text_button_theme.dart TextButtonThemeData
type TextButtonThemeData struct {
	// Style overrides the default appearance of the TextButtons.
	Style *ButtonStyle
}

// TextButtonThemeDataOpts are the parameters of NewTextButtonThemeData: the fields of the data.
//
// Flutter: material/text_button_theme.dart TextButtonThemeData.new
type TextButtonThemeDataOpts = TextButtonThemeData

// NewTextButtonThemeData creates the data.
//
// Flutter: material/text_button_theme.dart TextButtonThemeData.new
func NewTextButtonThemeData(opts ...TextButtonThemeDataOpts) TextButtonThemeData {
	return firstOpts(opts)
}

// TextButtonThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/text_button_theme.dart TextButtonThemeData.lerp
func TextButtonThemeDataLerp(a, b TextButtonThemeData, t float64) TextButtonThemeData {
	return TextButtonThemeData{Style: ButtonStyleLerp(a.Style, b.Style, t)}
}

// Equal reports whether the two themes have equal styles.
//
// Flutter: material/text_button_theme.dart TextButtonThemeData.==
func (d TextButtonThemeData) Equal(o TextButtonThemeData) bool { return d.Style.Equal(o.Style) }

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (TextButtonThemeData) ToStringShort() string { return "TextButtonThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d TextButtonThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the style.
//
// Flutter: material/text_button_theme.dart TextButtonThemeData.debugFillProperties
func (d TextButtonThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("style", nilIfZero(d.Style)))
}

// TextButtonTheme is an inherited theme that overrides the TextButtonThemeData of the theme
// for the buttons below it.
//
// Flutter: material/text_button_theme.dart TextButtonTheme
type TextButtonTheme struct {
	widgets.InheritedTheme
	data TextButtonThemeData
}

// NewTextButtonTheme creates the theme.
//
// Flutter: material/text_button_theme.dart TextButtonTheme.new
func NewTextButtonTheme(data TextButtonThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *TextButtonTheme {
	return &TextButtonTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/text_button_theme.dart TextButtonTheme.data
func (w *TextButtonTheme) Data() TextButtonThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/text_button_theme.dart TextButtonTheme.updateShouldNotify
func (w *TextButtonTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*TextButtonTheme).data)
}

// Wrap makes a TextButtonTheme with the same data around the child.
//
// Flutter: material/text_button_theme.dart TextButtonTheme.wrap
func (w *TextButtonTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewTextButtonTheme(w.data, child)
}

// TextButtonThemeOf returns the data of the closest TextButtonTheme above the context, or the
// textButtonTheme of the closest Theme. The context depends on it.
//
// Flutter: material/text_button_theme.dart TextButtonTheme.of
func TextButtonThemeOf(bc widgets.BuildContext) TextButtonThemeData {
	if t, ok := widgets.DependOn[*TextButtonTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).TextButtonTheme
}
