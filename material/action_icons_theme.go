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

// ActionIconThemeData defines default property values for descendant BackButton, CloseButton, DrawerButton and EndDrawerButton widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/action_icons_theme.dart ActionIconThemeData
//
//glutter:copywith value
type ActionIconThemeData struct {
	// BackButtonIconBuilder builds the icon of the BackButtons.
	BackButtonIconBuilder widgets.WidgetBuilder
	// CloseButtonIconBuilder builds the icon of the CloseButtons.
	CloseButtonIconBuilder widgets.WidgetBuilder
	// DrawerButtonIconBuilder builds the icon of the DrawerButtons.
	DrawerButtonIconBuilder widgets.WidgetBuilder
	// EndDrawerButtonIconBuilder builds the icon of the EndDrawerButtons.
	EndDrawerButtonIconBuilder widgets.WidgetBuilder
}

// ActionIconThemeDataOpts are the parameters of NewActionIconThemeData: the fields of the data.
//
// Flutter: material/action_icons_theme.dart ActionIconThemeData.new
type ActionIconThemeDataOpts = ActionIconThemeData

// NewActionIconThemeData creates the data; every field is optional.
//
// Flutter: material/action_icons_theme.dart ActionIconThemeData.new
func NewActionIconThemeData(opts ...ActionIconThemeDataOpts) ActionIconThemeData {
	return firstOpts(opts)
}

// ActionIconThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/action_icons_theme.dart ActionIconThemeData.lerp
func ActionIconThemeDataLerp(a, b ActionIconThemeData, t float64) ActionIconThemeData {
	return ActionIconThemeData{
		BackButtonIconBuilder:      lerpFirstOrSecond(a.BackButtonIconBuilder, b.BackButtonIconBuilder, t),
		CloseButtonIconBuilder:     lerpFirstOrSecond(a.CloseButtonIconBuilder, b.CloseButtonIconBuilder, t),
		DrawerButtonIconBuilder:    lerpFirstOrSecond(a.DrawerButtonIconBuilder, b.DrawerButtonIconBuilder, t),
		EndDrawerButtonIconBuilder: lerpFirstOrSecond(a.EndDrawerButtonIconBuilder, b.EndDrawerButtonIconBuilder, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/action_icons_theme.dart ActionIconThemeData.==
func (d ActionIconThemeData) Equal(o ActionIconThemeData) bool {
	return foundation.ValuesEqual(d.BackButtonIconBuilder, o.BackButtonIconBuilder) &&
		foundation.ValuesEqual(d.CloseButtonIconBuilder, o.CloseButtonIconBuilder) &&
		foundation.ValuesEqual(d.DrawerButtonIconBuilder, o.DrawerButtonIconBuilder) &&
		foundation.ValuesEqual(d.EndDrawerButtonIconBuilder, o.EndDrawerButtonIconBuilder)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (ActionIconThemeData) ToStringShort() string { return "ActionIconThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d ActionIconThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/action_icons_theme.dart ActionIconThemeData.debugFillProperties
func (d ActionIconThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("backButtonIconBuilder", nilFuncValue(d.BackButtonIconBuilder)))
	p.Add(objectProp("closeButtonIconBuilder", nilFuncValue(d.CloseButtonIconBuilder)))
	p.Add(objectProp("drawerButtonIconBuilder", nilFuncValue(d.DrawerButtonIconBuilder)))
	p.Add(objectProp("endDrawerButtonIconBuilder", nilFuncValue(d.EndDrawerButtonIconBuilder)))
}

// ActionIconTheme is an inherited theme that overrides the ActionIconThemeData of the theme for the
// widgets below it.
//
// Flutter: material/action_icons_theme.dart ActionIconTheme
type ActionIconTheme struct {
	widgets.InheritedTheme
	data ActionIconThemeData
}

// NewActionIconTheme creates the theme.
//
// Flutter: material/action_icons_theme.dart ActionIconTheme.new
func NewActionIconTheme(data ActionIconThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *ActionIconTheme {
	return &ActionIconTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/action_icons_theme.dart ActionIconTheme.data
func (w *ActionIconTheme) Data() ActionIconThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/action_icons_theme.dart ActionIconTheme.updateShouldNotify
func (w *ActionIconTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*ActionIconTheme).data)
}

// Wrap makes a ActionIconTheme with the same data around the child.
//
// Flutter: material/action_icons_theme.dart ActionIconTheme.wrap
func (w *ActionIconTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewActionIconTheme(w.data, child)
}

// ActionIconThemeOf returns the data of the closest ActionIconTheme above the context, or the
// actionIconTheme of the closest Theme. The context depends on it.
//
// Flutter: material/action_icons_theme.dart ActionIconTheme.of
func ActionIconThemeOf(bc widgets.BuildContext) ActionIconThemeData {
	if t, ok := widgets.DependOn[*ActionIconTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).ActionIconTheme
}

// nilFuncValue is the value that the diagnostics of a widget builder show: nil
// when there is none.
func nilFuncValue(f widgets.WidgetBuilder) any {
	if f == nil {
		return nil
	}
	return "WidgetBuilder"
}
