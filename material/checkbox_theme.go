// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CheckboxThemeData defines default property values for descendant Checkbox widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/checkbox_theme.dart CheckboxThemeData
//
//glutter:copywith value
type CheckboxThemeData struct {
	// MouseCursor is the cursor over the checkbox, in each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// FillColor is the color that fills the checkbox, in each state.
	FillColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// CheckColor is the color of the check mark, in each state.
	CheckColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// OverlayColor is the color of the overlay (the hover, focus and press highlights), in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashRadius is the radius of the splash of the checkbox.
	SplashRadius foundation.Opt[float64]
	// MaterialTapTargetSize is the size of the tap target of the checkbox.
	MaterialTapTargetSize MaterialTapTargetSize
	// VisualDensity is how compact the layout of the checkbox is.
	VisualDensity foundation.Opt[VisualDensity]
	// Shape is the shape of the checkbox.
	Shape painting.OutlinedBorder
	// Side is the border of the checkbox when it is not selected, as one border for every state.
	Side foundation.Opt[painting.BorderSide]
	// StateSide is the border of the checkbox in each state (Flutter's WidgetStateBorderSide); it replaces Side when it is set.
	StateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
}

// CheckboxThemeDataOpts are the parameters of NewCheckboxThemeData: the fields of the data.
//
// Flutter: material/checkbox_theme.dart CheckboxThemeData.new
type CheckboxThemeDataOpts = CheckboxThemeData

// NewCheckboxThemeData creates the data; every field is optional.
//
// Flutter: material/checkbox_theme.dart CheckboxThemeData.new
func NewCheckboxThemeData(opts ...CheckboxThemeDataOpts) CheckboxThemeData {
	return firstOpts(opts)
}

// CheckboxThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/checkbox_theme.dart CheckboxThemeData.lerp
func CheckboxThemeDataLerp(a, b CheckboxThemeData, t float64) CheckboxThemeData {
	return CheckboxThemeData{
		MouseCursor:           lerpFirstOrSecond(a.MouseCursor, b.MouseCursor, t),
		FillColor:             lerpStateOptColor(a.FillColor, b.FillColor, t),
		CheckColor:            lerpStateOptColor(a.CheckColor, b.CheckColor, t),
		OverlayColor:          lerpStateOptColor(a.OverlayColor, b.OverlayColor, t),
		SplashRadius:          lerpOptDouble(a.SplashRadius, b.SplashRadius, t),
		MaterialTapTargetSize: lerpFirstOrSecond(a.MaterialTapTargetSize, b.MaterialTapTargetSize, t),
		VisualDensity:         lerpFirstOrSecond(a.VisualDensity, b.VisualDensity, t),
		Shape:                 painting.OutlinedBorderLerp(a.Shape, b.Shape, t),
		Side:                  lerpWidgetSide(a.Side, a.StateSide, b.Side, b.StateSide, t),
		StateSide:             nil,
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/checkbox_theme.dart CheckboxThemeData.==
func (d CheckboxThemeData) Equal(o CheckboxThemeData) bool {
	return foundation.ValuesEqual(d.MouseCursor, o.MouseCursor) &&
		foundation.ValuesEqual(d.FillColor, o.FillColor) &&
		foundation.ValuesEqual(d.CheckColor, o.CheckColor) &&
		foundation.ValuesEqual(d.OverlayColor, o.OverlayColor) &&
		d.SplashRadius == o.SplashRadius &&
		d.MaterialTapTargetSize == o.MaterialTapTargetSize &&
		d.VisualDensity == o.VisualDensity &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		d.Side == o.Side &&
		foundation.ValuesEqual(d.StateSide, o.StateSide)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (CheckboxThemeData) ToStringShort() string { return "CheckboxThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d CheckboxThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/checkbox_theme.dart CheckboxThemeData.debugFillProperties
func (d CheckboxThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("mouseCursor", propOrNil(d.MouseCursor)))
	p.Add(objectProp("fillColor", propOrNil(d.FillColor)))
	p.Add(objectProp("checkColor", propOrNil(d.CheckColor)))
	p.Add(objectProp("overlayColor", propOrNil(d.OverlayColor)))
	p.Add(doubleProp("splashRadius", d.SplashRadius))
	p.Add(objectProp("materialTapTargetSize", nilIfZero(d.MaterialTapTargetSize)))
	p.Add(objectProp("visualDensity", optValue(d.VisualDensity)))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("side", optValue(d.Side)))
	p.Add(objectProp("stateSide", propOrNil(d.StateSide)))
}

// CheckboxTheme is an inherited theme that overrides the CheckboxThemeData of the theme for the
// widgets below it.
//
// Flutter: material/checkbox_theme.dart CheckboxTheme
type CheckboxTheme struct {
	widgets.InheritedTheme
	data CheckboxThemeData
}

// NewCheckboxTheme creates the theme.
//
// Flutter: material/checkbox_theme.dart CheckboxTheme.new
func NewCheckboxTheme(data CheckboxThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *CheckboxTheme {
	return &CheckboxTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/checkbox_theme.dart CheckboxTheme.data
func (w *CheckboxTheme) Data() CheckboxThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/checkbox_theme.dart CheckboxTheme.updateShouldNotify
func (w *CheckboxTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*CheckboxTheme).data)
}

// Wrap makes a CheckboxTheme with the same data around the child.
//
// Flutter: material/checkbox_theme.dart CheckboxTheme.wrap
func (w *CheckboxTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewCheckboxTheme(w.data, child)
}

// CheckboxThemeOf returns the data of the closest CheckboxTheme above the context, or the
// checkboxTheme of the closest Theme. The context depends on it.
//
// Flutter: material/checkbox_theme.dart CheckboxTheme.of
func CheckboxThemeOf(bc widgets.BuildContext) CheckboxThemeData {
	if t, ok := widgets.DependOn[*CheckboxTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).CheckboxTheme
}
