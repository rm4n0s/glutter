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

// MenuBarThemeData defines default property values for descendant MenuBar widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/menu_bar_theme.dart MenuBarThemeData
//
//glutter:copywith value
type MenuBarThemeData struct {
	// Style is the style of the menu bar.
	Style *MenuStyle
}

// MenuBarThemeDataOpts are the parameters of NewMenuBarThemeData: the fields of the data.
//
// Flutter: material/menu_bar_theme.dart MenuBarThemeData.new
type MenuBarThemeDataOpts = MenuBarThemeData

// NewMenuBarThemeData creates the data; every field is optional.
//
// Flutter: material/menu_bar_theme.dart MenuBarThemeData.new
func NewMenuBarThemeData(opts ...MenuBarThemeDataOpts) MenuBarThemeData {
	return firstOpts(opts)
}

// MenuBarThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/menu_bar_theme.dart MenuBarThemeData.lerp
func MenuBarThemeDataLerp(a, b MenuBarThemeData, t float64) MenuBarThemeData {
	return MenuBarThemeData{
		Style: MenuStyleLerp(a.Style, b.Style, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/menu_bar_theme.dart MenuBarThemeData.==
func (d MenuBarThemeData) Equal(o MenuBarThemeData) bool {
	return d.Style.Equal(o.Style)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (MenuBarThemeData) ToStringShort() string { return "MenuBarThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d MenuBarThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/menu_bar_theme.dart MenuBarThemeData.debugFillProperties
func (d MenuBarThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("style", nilIfZero(d.Style)))
}

// MenuBarTheme is an inherited theme that overrides the MenuBarThemeData of the theme for the
// widgets below it.
//
// Flutter: material/menu_bar_theme.dart MenuBarTheme
type MenuBarTheme struct {
	widgets.InheritedTheme
	data MenuBarThemeData
}

// NewMenuBarTheme creates the theme.
//
// Flutter: material/menu_bar_theme.dart MenuBarTheme.new
func NewMenuBarTheme(data MenuBarThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *MenuBarTheme {
	return &MenuBarTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/menu_bar_theme.dart MenuBarTheme.data
func (w *MenuBarTheme) Data() MenuBarThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/menu_bar_theme.dart MenuBarTheme.updateShouldNotify
func (w *MenuBarTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*MenuBarTheme).data)
}

// Wrap makes a MenuBarTheme with the same data around the child.
//
// Flutter: material/menu_bar_theme.dart MenuBarTheme.wrap
func (w *MenuBarTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewMenuBarTheme(w.data, child)
}

// MenuBarThemeOf returns the data of the closest MenuBarTheme above the context, or the
// menuBarTheme of the closest Theme. The context depends on it.
//
// Flutter: material/menu_bar_theme.dart MenuBarTheme.of
func MenuBarThemeOf(bc widgets.BuildContext) MenuBarThemeData {
	if t, ok := widgets.DependOn[*MenuBarTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).MenuBarTheme
}
