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

// SwitchThemeData defines default property values for descendant Switch widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/switch_theme.dart SwitchThemeData
//
//glutter:copywith value
type SwitchThemeData struct {
	// ThumbColor is the color of the thumb, in each state.
	ThumbColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TrackColor is the color of the track, in each state.
	TrackColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TrackOutlineColor is the color of the outline of the track, in each state.
	TrackOutlineColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TrackOutlineWidth is the width of the outline of the track, in each state.
	TrackOutlineWidth widgets.WidgetStateProperty[foundation.Opt[float64]]
	// MaterialTapTargetSize is the size of the tap target of the switch.
	MaterialTapTargetSize MaterialTapTargetSize
	// MouseCursor is the cursor over the switch, in each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// OverlayColor is the color of the overlay, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashRadius is the radius of the splash of the switch.
	SplashRadius foundation.Opt[float64]
	// ThumbIcon is the icon on the thumb, in each state.
	ThumbIcon widgets.WidgetStateProperty[*widgets.Icon]
	// Padding is the padding around the switch.
	Padding painting.EdgeInsetsGeometry
}

// SwitchThemeDataOpts are the parameters of NewSwitchThemeData: the fields of the data.
//
// Flutter: material/switch_theme.dart SwitchThemeData.new
type SwitchThemeDataOpts = SwitchThemeData

// NewSwitchThemeData creates the data; every field is optional.
//
// Flutter: material/switch_theme.dart SwitchThemeData.new
func NewSwitchThemeData(opts ...SwitchThemeDataOpts) SwitchThemeData {
	return firstOpts(opts)
}

// SwitchThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/switch_theme.dart SwitchThemeData.lerp
func SwitchThemeDataLerp(a, b SwitchThemeData, t float64) SwitchThemeData {
	return SwitchThemeData{
		ThumbColor:            lerpStateOptColor(a.ThumbColor, b.ThumbColor, t),
		TrackColor:            lerpStateOptColor(a.TrackColor, b.TrackColor, t),
		TrackOutlineColor:     lerpStateOptColor(a.TrackOutlineColor, b.TrackOutlineColor, t),
		TrackOutlineWidth:     lerpStateOpt(a.TrackOutlineWidth, b.TrackOutlineWidth, t, lerpOptDouble),
		MaterialTapTargetSize: lerpFirstOrSecond(a.MaterialTapTargetSize, b.MaterialTapTargetSize, t),
		MouseCursor:           lerpFirstOrSecond(a.MouseCursor, b.MouseCursor, t),
		OverlayColor:          lerpStateOptColor(a.OverlayColor, b.OverlayColor, t),
		SplashRadius:          lerpOptDouble(a.SplashRadius, b.SplashRadius, t),
		ThumbIcon:             lerpFirstOrSecond(a.ThumbIcon, b.ThumbIcon, t),
		Padding:               painting.EdgeInsetsGeometryLerp(a.Padding, b.Padding, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/switch_theme.dart SwitchThemeData.==
func (d SwitchThemeData) Equal(o SwitchThemeData) bool {
	return foundation.ValuesEqual(d.ThumbColor, o.ThumbColor) &&
		foundation.ValuesEqual(d.TrackColor, o.TrackColor) &&
		foundation.ValuesEqual(d.TrackOutlineColor, o.TrackOutlineColor) &&
		foundation.ValuesEqual(d.TrackOutlineWidth, o.TrackOutlineWidth) &&
		d.MaterialTapTargetSize == o.MaterialTapTargetSize &&
		foundation.ValuesEqual(d.MouseCursor, o.MouseCursor) &&
		foundation.ValuesEqual(d.OverlayColor, o.OverlayColor) &&
		d.SplashRadius == o.SplashRadius &&
		foundation.ValuesEqual(d.ThumbIcon, o.ThumbIcon) &&
		foundation.ValuesEqual(d.Padding, o.Padding)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (SwitchThemeData) ToStringShort() string { return "SwitchThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d SwitchThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/switch_theme.dart SwitchThemeData.debugFillProperties
func (d SwitchThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("thumbColor", propOrNil(d.ThumbColor)))
	p.Add(objectProp("trackColor", propOrNil(d.TrackColor)))
	p.Add(objectProp("trackOutlineColor", propOrNil(d.TrackOutlineColor)))
	p.Add(objectProp("trackOutlineWidth", propOrNil(d.TrackOutlineWidth)))
	p.Add(objectProp("materialTapTargetSize", nilIfZero(d.MaterialTapTargetSize)))
	p.Add(objectProp("mouseCursor", propOrNil(d.MouseCursor)))
	p.Add(objectProp("overlayColor", propOrNil(d.OverlayColor)))
	p.Add(doubleProp("splashRadius", d.SplashRadius))
	p.Add(objectProp("thumbIcon", propOrNil(d.ThumbIcon)))
	p.Add(objectProp("padding", d.Padding))
}

// SwitchTheme is an inherited theme that overrides the SwitchThemeData of the theme for the
// widgets below it.
//
// Flutter: material/switch_theme.dart SwitchTheme
type SwitchTheme struct {
	widgets.InheritedTheme
	data SwitchThemeData
}

// NewSwitchTheme creates the theme.
//
// Flutter: material/switch_theme.dart SwitchTheme.new
func NewSwitchTheme(data SwitchThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *SwitchTheme {
	return &SwitchTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/switch_theme.dart SwitchTheme.data
func (w *SwitchTheme) Data() SwitchThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/switch_theme.dart SwitchTheme.updateShouldNotify
func (w *SwitchTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*SwitchTheme).data)
}

// Wrap makes a SwitchTheme with the same data around the child.
//
// Flutter: material/switch_theme.dart SwitchTheme.wrap
func (w *SwitchTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewSwitchTheme(w.data, child)
}

// SwitchThemeOf returns the data of the closest SwitchTheme above the context, or the
// switchTheme of the closest Theme. The context depends on it.
//
// Flutter: material/switch_theme.dart SwitchTheme.of
func SwitchThemeOf(bc widgets.BuildContext) SwitchThemeData {
	if t, ok := widgets.DependOn[*SwitchTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).SwitchTheme
}
