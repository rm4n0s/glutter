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

// RadioThemeData defines default property values for descendant Radio widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/radio_theme.dart RadioThemeData
//
//glutter:copywith value
type RadioThemeData struct {
	// MouseCursor is the cursor over the radio, in each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// FillColor is the color of the radio, in each state.
	FillColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// OverlayColor is the color of the overlay, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashRadius is the radius of the splash of the radio.
	SplashRadius foundation.Opt[float64]
	// MaterialTapTargetSize is the size of the tap target of the radio.
	MaterialTapTargetSize MaterialTapTargetSize
	// VisualDensity is how compact the layout of the radio is.
	VisualDensity foundation.Opt[VisualDensity]
	// BackgroundColor is the color of the background of the radio, in each state.
	BackgroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// Side is the border of the radio when it is not selected, as one border for every state.
	Side foundation.Opt[painting.BorderSide]
	// StateSide is the border of the radio in each state (Flutter's WidgetStateBorderSide); it replaces Side when it is set.
	StateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// InnerRadius is the radius of the inner circle of the radio, in each state.
	InnerRadius widgets.WidgetStateProperty[foundation.Opt[float64]]
}

// RadioThemeDataOpts are the parameters of NewRadioThemeData: the fields of the data.
//
// Flutter: material/radio_theme.dart RadioThemeData.new
type RadioThemeDataOpts = RadioThemeData

// NewRadioThemeData creates the data; every field is optional.
//
// Flutter: material/radio_theme.dart RadioThemeData.new
func NewRadioThemeData(opts ...RadioThemeDataOpts) RadioThemeData {
	return firstOpts(opts)
}

// RadioThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/radio_theme.dart RadioThemeData.lerp
func RadioThemeDataLerp(a, b RadioThemeData, t float64) RadioThemeData {
	return RadioThemeData{
		MouseCursor:           lerpFirstOrSecond(a.MouseCursor, b.MouseCursor, t),
		FillColor:             lerpStateOptColor(a.FillColor, b.FillColor, t),
		OverlayColor:          lerpStateOptColor(a.OverlayColor, b.OverlayColor, t),
		SplashRadius:          lerpOptDouble(a.SplashRadius, b.SplashRadius, t),
		MaterialTapTargetSize: lerpFirstOrSecond(a.MaterialTapTargetSize, b.MaterialTapTargetSize, t),
		VisualDensity:         lerpFirstOrSecond(a.VisualDensity, b.VisualDensity, t),
		BackgroundColor:       lerpStateOptColor(a.BackgroundColor, b.BackgroundColor, t),
		Side:                  lerpWidgetSide(a.Side, a.StateSide, b.Side, b.StateSide, t),
		StateSide:             nil,
		InnerRadius:           lerpStateOpt(a.InnerRadius, b.InnerRadius, t, lerpOptDouble),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/radio_theme.dart RadioThemeData.==
func (d RadioThemeData) Equal(o RadioThemeData) bool {
	return foundation.ValuesEqual(d.MouseCursor, o.MouseCursor) &&
		foundation.ValuesEqual(d.FillColor, o.FillColor) &&
		foundation.ValuesEqual(d.OverlayColor, o.OverlayColor) &&
		d.SplashRadius == o.SplashRadius &&
		d.MaterialTapTargetSize == o.MaterialTapTargetSize &&
		d.VisualDensity == o.VisualDensity &&
		foundation.ValuesEqual(d.BackgroundColor, o.BackgroundColor) &&
		d.Side == o.Side &&
		foundation.ValuesEqual(d.StateSide, o.StateSide) &&
		foundation.ValuesEqual(d.InnerRadius, o.InnerRadius)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (RadioThemeData) ToStringShort() string { return "RadioThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d RadioThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/radio_theme.dart RadioThemeData.debugFillProperties
func (d RadioThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("mouseCursor", propOrNil(d.MouseCursor)))
	p.Add(objectProp("fillColor", propOrNil(d.FillColor)))
	p.Add(objectProp("overlayColor", propOrNil(d.OverlayColor)))
	p.Add(doubleProp("splashRadius", d.SplashRadius))
	p.Add(objectProp("materialTapTargetSize", nilIfZero(d.MaterialTapTargetSize)))
	p.Add(objectProp("visualDensity", optValue(d.VisualDensity)))
	p.Add(objectProp("backgroundColor", propOrNil(d.BackgroundColor)))
	p.Add(objectProp("side", optValue(d.Side)))
	p.Add(objectProp("stateSide", propOrNil(d.StateSide)))
	p.Add(objectProp("innerRadius", propOrNil(d.InnerRadius)))
}

// RadioTheme is an inherited theme that overrides the RadioThemeData of the theme for the
// widgets below it.
//
// Flutter: material/radio_theme.dart RadioTheme
type RadioTheme struct {
	widgets.InheritedTheme
	data RadioThemeData
}

// NewRadioTheme creates the theme.
//
// Flutter: material/radio_theme.dart RadioTheme.new
func NewRadioTheme(data RadioThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *RadioTheme {
	return &RadioTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/radio_theme.dart RadioTheme.data
func (w *RadioTheme) Data() RadioThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/radio_theme.dart RadioTheme.updateShouldNotify
func (w *RadioTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*RadioTheme).data)
}

// Wrap makes a RadioTheme with the same data around the child.
//
// Flutter: material/radio_theme.dart RadioTheme.wrap
func (w *RadioTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewRadioTheme(w.data, child)
}

// RadioThemeOf returns the data of the closest RadioTheme above the context, or the
// radioTheme of the closest Theme. The context depends on it.
//
// Flutter: material/radio_theme.dart RadioTheme.of
func RadioThemeOf(bc widgets.BuildContext) RadioThemeData {
	if t, ok := widgets.DependOn[*RadioTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).RadioTheme
}
