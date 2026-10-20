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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ChipThemeData defines default property values for descendant Chip widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/chip_theme.dart ChipThemeData
//
//glutter:copywith value
type ChipThemeData struct {
	// Color is the color of the chip, in each state; it takes precedence over the other chip colors.
	Color widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// BackgroundColor is the color of the chip's background.
	BackgroundColor foundation.Opt[ui.Color]
	// DeleteIconColor is the color of the chip's delete icon.
	DeleteIconColor foundation.Opt[ui.Color]
	// DisabledColor is the color of the chip's background when it is disabled.
	DisabledColor foundation.Opt[ui.Color]
	// SelectedColor is the color of the chip's background when it is selected.
	SelectedColor foundation.Opt[ui.Color]
	// SecondarySelectedColor is the background color of a selected chip that uses the secondary label style.
	SecondarySelectedColor foundation.Opt[ui.Color]
	// ShadowColor is the color of the chip's shadow.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the color of the chip's surface tint overlay.
	SurfaceTintColor foundation.Opt[ui.Color]
	// SelectedShadowColor is the color of the shadow of a selected chip.
	SelectedShadowColor foundation.Opt[ui.Color]
	// ShowCheckmark is whether a selected chip shows a check mark; the default is true.
	ShowCheckmark foundation.Opt[bool]
	// CheckmarkColor is the color of the check mark of a selected chip.
	CheckmarkColor foundation.Opt[ui.Color]
	// LabelPadding is the padding around the label.
	LabelPadding painting.EdgeInsetsGeometry
	// Padding is the padding between the chip's border and its content.
	Padding painting.EdgeInsetsGeometry
	// Side is the border of the chip, as one border for every state.
	Side foundation.Opt[painting.BorderSide]
	// StateSide is the border of the chip in each state (Flutter's WidgetStateBorderSide); it replaces Side when it is set.
	StateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// Shape is the shape of the chip's border and of its ink splash.
	Shape painting.OutlinedBorder
	// LabelStyle is the style of the chip's label.
	LabelStyle *painting.TextStyle
	// SecondaryLabelStyle is the style of the label of a chip that is selected with the secondary colors.
	SecondaryLabelStyle *painting.TextStyle
	// Brightness is the brightness that the chip's colors are for.
	Brightness foundation.Opt[ui.Brightness]
	// Elevation is the elevation of the chip.
	Elevation foundation.Opt[float64]
	// PressElevation is the elevation of the chip while it is pressed.
	PressElevation foundation.Opt[float64]
	// IconTheme is the theme of the icons of the chip.
	IconTheme foundation.Opt[widgets.IconThemeData]
	// AvatarBoxConstraints are the constraints of the chip's avatar.
	AvatarBoxConstraints foundation.Opt[rendering.BoxConstraints]
	// DeleteIconBoxConstraints are the constraints of the chip's delete icon.
	DeleteIconBoxConstraints foundation.Opt[rendering.BoxConstraints]
}

// ChipThemeDataOpts are the parameters of NewChipThemeData: the fields of the data.
//
// Flutter: material/chip_theme.dart ChipThemeData.new
type ChipThemeDataOpts = ChipThemeData

// NewChipThemeData creates the data; every field is optional.
//
// Flutter: material/chip_theme.dart ChipThemeData.new
func NewChipThemeData(opts ...ChipThemeDataOpts) ChipThemeData {
	return firstOpts(opts)
}

// ChipThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/chip_theme.dart ChipThemeData.lerp
func ChipThemeDataLerp(a, b ChipThemeData, t float64) ChipThemeData {
	return ChipThemeData{
		Color:                    lerpStateOptColor(a.Color, b.Color, t),
		BackgroundColor:          lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		DeleteIconColor:          lerpOptColor(a.DeleteIconColor, b.DeleteIconColor, t),
		DisabledColor:            lerpOptColor(a.DisabledColor, b.DisabledColor, t),
		SelectedColor:            lerpOptColor(a.SelectedColor, b.SelectedColor, t),
		SecondarySelectedColor:   lerpOptColor(a.SecondarySelectedColor, b.SecondarySelectedColor, t),
		ShadowColor:              lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor:         lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		SelectedShadowColor:      lerpOptColor(a.SelectedShadowColor, b.SelectedShadowColor, t),
		ShowCheckmark:            lerpBoolDefault(a.ShowCheckmark, b.ShowCheckmark, t, true),
		CheckmarkColor:           lerpOptColor(a.CheckmarkColor, b.CheckmarkColor, t),
		LabelPadding:             painting.EdgeInsetsGeometryLerp(a.LabelPadding, b.LabelPadding, t),
		Padding:                  painting.EdgeInsetsGeometryLerp(a.Padding, b.Padding, t),
		Side:                     lerpWidgetSide(a.Side, a.StateSide, b.Side, b.StateSide, t),
		StateSide:                nil,
		Shape:                    painting.OutlinedBorderLerp(a.Shape, b.Shape, t),
		LabelStyle:               painting.TextStyleLerp(a.LabelStyle, b.LabelStyle, t),
		SecondaryLabelStyle:      painting.TextStyleLerp(a.SecondaryLabelStyle, b.SecondaryLabelStyle, t),
		Brightness:               lerpBrightnessDefaultLight(a.Brightness, b.Brightness, t),
		Elevation:                lerpOptDouble(a.Elevation, b.Elevation, t),
		PressElevation:           lerpOptDouble(a.PressElevation, b.PressElevation, t),
		IconTheme:                lerpOptIconTheme(a.IconTheme, b.IconTheme, t),
		AvatarBoxConstraints:     lerpOptConstraints(a.AvatarBoxConstraints, b.AvatarBoxConstraints, t),
		DeleteIconBoxConstraints: lerpOptConstraints(a.DeleteIconBoxConstraints, b.DeleteIconBoxConstraints, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/chip_theme.dart ChipThemeData.==
func (d ChipThemeData) Equal(o ChipThemeData) bool {
	return foundation.ValuesEqual(d.Color, o.Color) &&
		d.BackgroundColor == o.BackgroundColor &&
		d.DeleteIconColor == o.DeleteIconColor &&
		d.DisabledColor == o.DisabledColor &&
		d.SelectedColor == o.SelectedColor &&
		d.SecondarySelectedColor == o.SecondarySelectedColor &&
		d.ShadowColor == o.ShadowColor &&
		d.SurfaceTintColor == o.SurfaceTintColor &&
		d.SelectedShadowColor == o.SelectedShadowColor &&
		d.ShowCheckmark == o.ShowCheckmark &&
		d.CheckmarkColor == o.CheckmarkColor &&
		foundation.ValuesEqual(d.LabelPadding, o.LabelPadding) &&
		foundation.ValuesEqual(d.Padding, o.Padding) &&
		d.Side == o.Side &&
		foundation.ValuesEqual(d.StateSide, o.StateSide) &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		textStyleEqual(d.LabelStyle, o.LabelStyle) &&
		textStyleEqual(d.SecondaryLabelStyle, o.SecondaryLabelStyle) &&
		d.Brightness == o.Brightness &&
		d.Elevation == o.Elevation &&
		d.PressElevation == o.PressElevation &&
		iconThemeOptEqual(d.IconTheme, o.IconTheme) &&
		d.AvatarBoxConstraints == o.AvatarBoxConstraints &&
		d.DeleteIconBoxConstraints == o.DeleteIconBoxConstraints
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (ChipThemeData) ToStringShort() string { return "ChipThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d ChipThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/chip_theme.dart ChipThemeData.debugFillProperties
func (d ChipThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("color", propOrNil(d.Color)))
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(colorProp("deleteIconColor", d.DeleteIconColor))
	p.Add(colorProp("disabledColor", d.DisabledColor))
	p.Add(colorProp("selectedColor", d.SelectedColor))
	p.Add(colorProp("secondarySelectedColor", d.SecondarySelectedColor))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(colorProp("selectedShadowColor", d.SelectedShadowColor))
	p.Add(objectProp("showCheckmark", optValue(d.ShowCheckmark)))
	p.Add(colorProp("checkMarkColor", d.CheckmarkColor))
	p.Add(objectProp("labelPadding", d.LabelPadding))
	p.Add(objectProp("padding", d.Padding))
	p.Add(objectProp("side", optValue(d.Side)))
	p.Add(objectProp("stateSide", propOrNil(d.StateSide)))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("labelStyle", d.LabelStyle))
	p.Add(objectProp("secondaryLabelStyle", d.SecondaryLabelStyle))
	p.Add(objectProp("brightness", optValue(d.Brightness)))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(doubleProp("pressElevation", d.PressElevation))
	p.Add(objectProp("iconTheme", optValue(d.IconTheme)))
	p.Add(objectProp("avatarBoxConstraints", optValue(d.AvatarBoxConstraints)))
	p.Add(objectProp("deleteIconBoxConstraints", optValue(d.DeleteIconBoxConstraints)))
}

// ChipTheme is an inherited theme that overrides the ChipThemeData of the theme for the
// widgets below it.
//
// Flutter: material/chip_theme.dart ChipTheme
type ChipTheme struct {
	widgets.InheritedTheme
	data ChipThemeData
}

// NewChipTheme creates the theme.
//
// Flutter: material/chip_theme.dart ChipTheme.new
func NewChipTheme(data ChipThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *ChipTheme {
	return &ChipTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/chip_theme.dart ChipTheme.data
func (w *ChipTheme) Data() ChipThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/chip_theme.dart ChipTheme.updateShouldNotify
func (w *ChipTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*ChipTheme).data)
}

// Wrap makes a ChipTheme with the same data around the child.
//
// Flutter: material/chip_theme.dart ChipTheme.wrap
func (w *ChipTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewChipTheme(w.data, child)
}

// ChipThemeOf returns the data of the closest ChipTheme above the context, or the
// chipTheme of the closest Theme. The context depends on it.
//
// Flutter: material/chip_theme.dart ChipTheme.of
func ChipThemeOf(bc widgets.BuildContext) ChipThemeData {
	if t, ok := widgets.DependOn[*ChipTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).ChipTheme
}
