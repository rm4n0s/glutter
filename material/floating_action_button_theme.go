// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// FloatingActionButtonThemeData defines default property values for descendant FloatingActionButton widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonThemeData
//
//glutter:copywith value
type FloatingActionButtonThemeData struct {
	// ForegroundColor is the default icon and text color of the floating action buttons.
	ForegroundColor foundation.Opt[ui.Color]
	// BackgroundColor is the color of the floating action buttons.
	BackgroundColor foundation.Opt[ui.Color]
	// FocusColor is the color of the ink well of a button that has the focus.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the color of the ink well of a button that a pointer hovers over.
	HoverColor foundation.Opt[ui.Color]
	// SplashColor is the splash color of the ink well.
	SplashColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the buttons at rest.
	Elevation foundation.Opt[float64]
	// FocusElevation is the z-coordinate of a button that has the focus.
	FocusElevation foundation.Opt[float64]
	// HoverElevation is the z-coordinate of a button that a pointer hovers over.
	HoverElevation foundation.Opt[float64]
	// DisabledElevation is the z-coordinate of a disabled button.
	DisabledElevation foundation.Opt[float64]
	// HighlightElevation is the z-coordinate of a button that is pressed.
	HighlightElevation foundation.Opt[float64]
	// Shape is the shape of the buttons.
	Shape painting.ShapeBorder
	// EnableFeedback says whether detected gestures give acoustic and haptic feedback.
	EnableFeedback foundation.Opt[bool]
	// IconSize is the size of the icon in the buttons.
	IconSize foundation.Opt[float64]
	// SizeConstraints are the size constraints of the regular buttons.
	SizeConstraints foundation.Opt[rendering.BoxConstraints]
	// SmallSizeConstraints are the size constraints of the small buttons.
	SmallSizeConstraints foundation.Opt[rendering.BoxConstraints]
	// LargeSizeConstraints are the size constraints of the large buttons.
	LargeSizeConstraints foundation.Opt[rendering.BoxConstraints]
	// ExtendedSizeConstraints are the size constraints of the extended buttons.
	ExtendedSizeConstraints foundation.Opt[rendering.BoxConstraints]
	// ExtendedIconLabelSpacing is the spacing between the icon and the label of an extended button.
	ExtendedIconLabelSpacing foundation.Opt[float64]
	// ExtendedPadding is the padding of the content of an extended button.
	ExtendedPadding painting.EdgeInsetsGeometry
	// ExtendedTextStyle is the text style of the label of an extended button.
	ExtendedTextStyle *painting.TextStyle
	// MouseCursor is the cursor over the buttons, in each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
}

// FloatingActionButtonThemeDataOpts are the parameters of NewFloatingActionButtonThemeData: the fields of the data.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonThemeData.new
type FloatingActionButtonThemeDataOpts = FloatingActionButtonThemeData

// NewFloatingActionButtonThemeData creates the data; every field is optional.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonThemeData.new
func NewFloatingActionButtonThemeData(opts ...FloatingActionButtonThemeDataOpts) FloatingActionButtonThemeData {
	return firstOpts(opts)
}

// FloatingActionButtonThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonThemeData.lerp
func FloatingActionButtonThemeDataLerp(a, b FloatingActionButtonThemeData, t float64) FloatingActionButtonThemeData {
	return FloatingActionButtonThemeData{
		ForegroundColor:          lerpOptColor(a.ForegroundColor, b.ForegroundColor, t),
		BackgroundColor:          lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		FocusColor:               lerpOptColor(a.FocusColor, b.FocusColor, t),
		HoverColor:               lerpOptColor(a.HoverColor, b.HoverColor, t),
		SplashColor:              lerpOptColor(a.SplashColor, b.SplashColor, t),
		Elevation:                lerpOptDouble(a.Elevation, b.Elevation, t),
		FocusElevation:           lerpOptDouble(a.FocusElevation, b.FocusElevation, t),
		HoverElevation:           lerpOptDouble(a.HoverElevation, b.HoverElevation, t),
		DisabledElevation:        lerpOptDouble(a.DisabledElevation, b.DisabledElevation, t),
		HighlightElevation:       lerpOptDouble(a.HighlightElevation, b.HighlightElevation, t),
		Shape:                    painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		EnableFeedback:           lerpFirstOrSecond(a.EnableFeedback, b.EnableFeedback, t),
		IconSize:                 lerpOptDouble(a.IconSize, b.IconSize, t),
		SizeConstraints:          lerpOptConstraints(a.SizeConstraints, b.SizeConstraints, t),
		SmallSizeConstraints:     lerpOptConstraints(a.SmallSizeConstraints, b.SmallSizeConstraints, t),
		LargeSizeConstraints:     lerpOptConstraints(a.LargeSizeConstraints, b.LargeSizeConstraints, t),
		ExtendedSizeConstraints:  lerpOptConstraints(a.ExtendedSizeConstraints, b.ExtendedSizeConstraints, t),
		ExtendedIconLabelSpacing: lerpOptDouble(a.ExtendedIconLabelSpacing, b.ExtendedIconLabelSpacing, t),
		ExtendedPadding:          painting.EdgeInsetsGeometryLerp(a.ExtendedPadding, b.ExtendedPadding, t),
		ExtendedTextStyle:        painting.TextStyleLerp(a.ExtendedTextStyle, b.ExtendedTextStyle, t),
		MouseCursor:              lerpFirstOrSecond(a.MouseCursor, b.MouseCursor, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonThemeData.==
func (d FloatingActionButtonThemeData) Equal(o FloatingActionButtonThemeData) bool {
	return d.ForegroundColor == o.ForegroundColor &&
		d.BackgroundColor == o.BackgroundColor &&
		d.FocusColor == o.FocusColor &&
		d.HoverColor == o.HoverColor &&
		d.SplashColor == o.SplashColor &&
		d.Elevation == o.Elevation &&
		d.FocusElevation == o.FocusElevation &&
		d.HoverElevation == o.HoverElevation &&
		d.DisabledElevation == o.DisabledElevation &&
		d.HighlightElevation == o.HighlightElevation &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		d.EnableFeedback == o.EnableFeedback &&
		d.IconSize == o.IconSize &&
		d.SizeConstraints == o.SizeConstraints &&
		d.SmallSizeConstraints == o.SmallSizeConstraints &&
		d.LargeSizeConstraints == o.LargeSizeConstraints &&
		d.ExtendedSizeConstraints == o.ExtendedSizeConstraints &&
		d.ExtendedIconLabelSpacing == o.ExtendedIconLabelSpacing &&
		foundation.ValuesEqual(d.ExtendedPadding, o.ExtendedPadding) &&
		textStyleEqual(d.ExtendedTextStyle, o.ExtendedTextStyle) &&
		foundation.ValuesEqual(d.MouseCursor, o.MouseCursor)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (FloatingActionButtonThemeData) ToStringShort() string { return "FloatingActionButtonThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d FloatingActionButtonThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonThemeData.debugFillProperties
func (d FloatingActionButtonThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("foregroundColor", d.ForegroundColor))
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(colorProp("focusColor", d.FocusColor))
	p.Add(colorProp("hoverColor", d.HoverColor))
	p.Add(colorProp("splashColor", d.SplashColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(doubleProp("focusElevation", d.FocusElevation))
	p.Add(doubleProp("hoverElevation", d.HoverElevation))
	p.Add(doubleProp("disabledElevation", d.DisabledElevation))
	p.Add(doubleProp("highlightElevation", d.HighlightElevation))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("enableFeedback", optValue(d.EnableFeedback)))
	p.Add(doubleProp("iconSize", d.IconSize))
	p.Add(objectProp("sizeConstraints", optValue(d.SizeConstraints)))
	p.Add(objectProp("smallSizeConstraints", optValue(d.SmallSizeConstraints)))
	p.Add(objectProp("largeSizeConstraints", optValue(d.LargeSizeConstraints)))
	p.Add(objectProp("extendedSizeConstraints", optValue(d.ExtendedSizeConstraints)))
	p.Add(doubleProp("extendedIconLabelSpacing", d.ExtendedIconLabelSpacing))
	p.Add(objectProp("extendedPadding", d.ExtendedPadding))
	p.Add(objectProp("extendedTextStyle", d.ExtendedTextStyle))
	p.Add(objectProp("mouseCursor", propOrNil(d.MouseCursor)))
}

// FloatingActionButtonTheme is an inherited theme that overrides the FloatingActionButtonThemeData of the theme for the
// widgets below it.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonTheme
type FloatingActionButtonTheme struct {
	widgets.InheritedTheme
	data FloatingActionButtonThemeData
}

// NewFloatingActionButtonTheme creates the theme.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonTheme.new
func NewFloatingActionButtonTheme(data FloatingActionButtonThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *FloatingActionButtonTheme {
	return &FloatingActionButtonTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonTheme.data
func (w *FloatingActionButtonTheme) Data() FloatingActionButtonThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonTheme.updateShouldNotify
func (w *FloatingActionButtonTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*FloatingActionButtonTheme).data)
}

// Wrap makes a FloatingActionButtonTheme with the same data around the child.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonTheme.wrap
func (w *FloatingActionButtonTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewFloatingActionButtonTheme(w.data, child)
}

// FloatingActionButtonThemeOf returns the data of the closest FloatingActionButtonTheme above the context, or the
// floatingActionButtonTheme of the closest Theme. The context depends on it.
//
// Flutter: material/floating_action_button_theme.dart FloatingActionButtonTheme.of
func FloatingActionButtonThemeOf(bc widgets.BuildContext) FloatingActionButtonThemeData {
	if t, ok := widgets.DependOn[*FloatingActionButtonTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).FloatingActionButtonTheme
}
