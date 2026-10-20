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

// MenuButtonThemeData overrides the default style of MenuButtons descendant of a MenuButtonTheme.
// Only the style is configurable; a property that the style does not set leaves
// the choice to the default style of the button.
//
// Flutter: material/menu_button_theme.dart MenuButtonThemeData
type MenuButtonThemeData struct {
	// Style overrides the default appearance of the MenuButtons.
	Style *ButtonStyle
}

// MenuButtonThemeDataOpts are the parameters of NewMenuButtonThemeData: the fields of the data.
//
// Flutter: material/menu_button_theme.dart MenuButtonThemeData.new
type MenuButtonThemeDataOpts = MenuButtonThemeData

// NewMenuButtonThemeData creates the data.
//
// Flutter: material/menu_button_theme.dart MenuButtonThemeData.new
func NewMenuButtonThemeData(opts ...MenuButtonThemeDataOpts) MenuButtonThemeData {
	return firstOpts(opts)
}

// MenuButtonThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/menu_button_theme.dart MenuButtonThemeData.lerp
func MenuButtonThemeDataLerp(a, b MenuButtonThemeData, t float64) MenuButtonThemeData {
	return MenuButtonThemeData{Style: ButtonStyleLerp(a.Style, b.Style, t)}
}

// Equal reports whether the two themes have equal styles.
//
// Flutter: material/menu_button_theme.dart MenuButtonThemeData.==
func (d MenuButtonThemeData) Equal(o MenuButtonThemeData) bool { return d.Style.Equal(o.Style) }

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (MenuButtonThemeData) ToStringShort() string { return "MenuButtonThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d MenuButtonThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the style.
//
// Flutter: material/menu_button_theme.dart MenuButtonThemeData.debugFillProperties
func (d MenuButtonThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("style", nilIfZero(d.Style)))
}

// MenuButtonTheme is an inherited theme that overrides the MenuButtonThemeData of the theme
// for the buttons below it.
//
// Flutter: material/menu_button_theme.dart MenuButtonTheme
type MenuButtonTheme struct {
	widgets.InheritedTheme
	data MenuButtonThemeData
}

// NewMenuButtonTheme creates the theme.
//
// Flutter: material/menu_button_theme.dart MenuButtonTheme.new
func NewMenuButtonTheme(data MenuButtonThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *MenuButtonTheme {
	return &MenuButtonTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/menu_button_theme.dart MenuButtonTheme.data
func (w *MenuButtonTheme) Data() MenuButtonThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/menu_button_theme.dart MenuButtonTheme.updateShouldNotify
func (w *MenuButtonTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*MenuButtonTheme).data)
}

// Wrap makes a MenuButtonTheme with the same data around the child.
//
// Flutter: material/menu_button_theme.dart MenuButtonTheme.wrap
func (w *MenuButtonTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewMenuButtonTheme(w.data, child)
}

// MenuButtonThemeOf returns the data of the closest MenuButtonTheme above the context, or the
// menuButtonTheme of the closest Theme. The context depends on it.
//
// Flutter: material/menu_button_theme.dart MenuButtonTheme.of
func MenuButtonThemeOf(bc widgets.BuildContext) MenuButtonThemeData {
	if t, ok := widgets.DependOn[*MenuButtonTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).MenuButtonTheme
}
