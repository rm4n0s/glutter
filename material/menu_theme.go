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

// MenuThemeData defines default property values for descendant MenuAnchor and MenuBar widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/menu_theme.dart MenuThemeData
//
//glutter:copywith value
type MenuThemeData struct {
	// Style is the style of the menu panels.
	Style *MenuStyle
	// SubmenuIcon is the icon that a SubmenuButton shows at its end, in each state.
	SubmenuIcon widgets.WidgetStateProperty[widgets.Widget]
}

// MenuThemeDataOpts are the parameters of NewMenuThemeData: the fields of the data.
//
// Flutter: material/menu_theme.dart MenuThemeData.new
type MenuThemeDataOpts = MenuThemeData

// NewMenuThemeData creates the data; every field is optional.
//
// Flutter: material/menu_theme.dart MenuThemeData.new
func NewMenuThemeData(opts ...MenuThemeDataOpts) MenuThemeData {
	return firstOpts(opts)
}

// MenuThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/menu_theme.dart MenuThemeData.lerp
func MenuThemeDataLerp(a, b MenuThemeData, t float64) MenuThemeData {
	return MenuThemeData{
		Style:       MenuStyleLerp(a.Style, b.Style, t),
		SubmenuIcon: lerpFirstOrSecond(a.SubmenuIcon, b.SubmenuIcon, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/menu_theme.dart MenuThemeData.==
func (d MenuThemeData) Equal(o MenuThemeData) bool {
	return d.Style.Equal(o.Style) &&
		foundation.ValuesEqual(d.SubmenuIcon, o.SubmenuIcon)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (MenuThemeData) ToStringShort() string { return "MenuThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d MenuThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/menu_theme.dart MenuThemeData.debugFillProperties
func (d MenuThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("style", nilIfZero(d.Style)))
	p.Add(objectProp("submenuIcon", propOrNil(d.SubmenuIcon)))
}

// MenuTheme is an inherited theme that overrides the MenuThemeData of the theme for the
// widgets below it.
//
// Flutter: material/menu_theme.dart MenuTheme
type MenuTheme struct {
	widgets.InheritedTheme
	data MenuThemeData
}

// NewMenuTheme creates the theme.
//
// Flutter: material/menu_theme.dart MenuTheme.new
func NewMenuTheme(data MenuThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *MenuTheme {
	return &MenuTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/menu_theme.dart MenuTheme.data
func (w *MenuTheme) Data() MenuThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/menu_theme.dart MenuTheme.updateShouldNotify
func (w *MenuTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*MenuTheme).data)
}

// Wrap makes a MenuTheme with the same data around the child.
//
// Flutter: material/menu_theme.dart MenuTheme.wrap
func (w *MenuTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewMenuTheme(w.data, child)
}

// MenuThemeOf returns the data of the closest MenuTheme above the context, or the
// menuTheme of the closest Theme. The context depends on it.
//
// Flutter: material/menu_theme.dart MenuTheme.of
func MenuThemeOf(bc widgets.BuildContext) MenuThemeData {
	if t, ok := widgets.DependOn[*MenuTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).MenuTheme
}
