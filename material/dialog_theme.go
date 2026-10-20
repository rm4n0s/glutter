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

// DialogThemeData defines default property values for descendant Dialog widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/dialog_theme.dart DialogThemeData
//
//glutter:copywith value
type DialogThemeData struct {
	// BackgroundColor is the background color of the dialog.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the dialog's Material.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow below the dialog.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the dialog (Material 3).
	SurfaceTintColor foundation.Opt[ui.Color]
	// Shape is the shape of the dialog's Material.
	Shape painting.ShapeBorder
	// Alignment is where the dialog is in the space left by the inset padding.
	Alignment painting.AlignmentGeometry
	// IconColor is the color of the icon of an AlertDialog.
	IconColor foundation.Opt[ui.Color]
	// TitleTextStyle is the style of the title.
	TitleTextStyle *painting.TextStyle
	// ContentTextStyle is the style of the content.
	ContentTextStyle *painting.TextStyle
	// ActionsPadding is the padding around the actions of an AlertDialog.
	ActionsPadding painting.EdgeInsetsGeometry
	// BarrierColor is the color of the barrier behind the dialog.
	BarrierColor foundation.Opt[ui.Color]
	// InsetPadding is the distance between the edges of the screen and the dialog.
	InsetPadding *painting.EdgeInsets
	// ClipBehavior is how the dialog's Material is clipped.
	ClipBehavior ui.Clip
	// Constraints are the size constraints of the dialog.
	Constraints foundation.Opt[rendering.BoxConstraints]
}

// DialogThemeDataOpts are the parameters of NewDialogThemeData: the fields of the data.
//
// Flutter: material/dialog_theme.dart DialogThemeData.new
type DialogThemeDataOpts = DialogThemeData

// NewDialogThemeData creates the data; every field is optional.
//
// Flutter: material/dialog_theme.dart DialogThemeData.new
func NewDialogThemeData(opts ...DialogThemeDataOpts) DialogThemeData {
	return firstOpts(opts)
}

// DialogThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/dialog_theme.dart DialogThemeData.lerp
func DialogThemeDataLerp(a, b DialogThemeData, t float64) DialogThemeData {
	return DialogThemeData{
		BackgroundColor:  lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		Elevation:        lerpOptDouble(a.Elevation, b.Elevation, t),
		ShadowColor:      lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor: lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		Shape:            painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		Alignment:        painting.AlignmentGeometryLerp(a.Alignment, b.Alignment, t),
		IconColor:        lerpOptColor(a.IconColor, b.IconColor, t),
		TitleTextStyle:   painting.TextStyleLerp(a.TitleTextStyle, b.TitleTextStyle, t),
		ContentTextStyle: painting.TextStyleLerp(a.ContentTextStyle, b.ContentTextStyle, t),
		ActionsPadding:   painting.EdgeInsetsGeometryLerp(a.ActionsPadding, b.ActionsPadding, t),
		BarrierColor:     lerpOptColor(a.BarrierColor, b.BarrierColor, t),
		InsetPadding:     painting.EdgeInsetsLerp(a.InsetPadding, b.InsetPadding, t),
		ClipBehavior:     lerpFirstOrSecond(a.ClipBehavior, b.ClipBehavior, t),
		Constraints:      lerpOptConstraints(a.Constraints, b.Constraints, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/dialog_theme.dart DialogThemeData.==
func (d DialogThemeData) Equal(o DialogThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor &&
		d.Elevation == o.Elevation &&
		d.ShadowColor == o.ShadowColor &&
		d.SurfaceTintColor == o.SurfaceTintColor &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		foundation.ValuesEqual(d.Alignment, o.Alignment) &&
		d.IconColor == o.IconColor &&
		textStyleEqual(d.TitleTextStyle, o.TitleTextStyle) &&
		textStyleEqual(d.ContentTextStyle, o.ContentTextStyle) &&
		foundation.ValuesEqual(d.ActionsPadding, o.ActionsPadding) &&
		d.BarrierColor == o.BarrierColor &&
		foundation.ValuesEqual(d.InsetPadding, o.InsetPadding) &&
		d.ClipBehavior == o.ClipBehavior &&
		d.Constraints == o.Constraints
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (DialogThemeData) ToStringShort() string { return "DialogThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d DialogThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/dialog_theme.dart DialogThemeData.debugFillProperties
func (d DialogThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("alignment", d.Alignment))
	p.Add(colorProp("iconColor", d.IconColor))
	p.Add(objectProp("titleTextStyle", d.TitleTextStyle))
	p.Add(objectProp("contentTextStyle", d.ContentTextStyle))
	p.Add(objectProp("actionsPadding", d.ActionsPadding))
	p.Add(colorProp("barrierColor", d.BarrierColor))
	p.Add(objectProp("insetPadding", d.InsetPadding))
	p.Add(objectProp("clipBehavior", nilIfZero(d.ClipBehavior)))
	p.Add(objectProp("constraints", optValue(d.Constraints)))
}

// DialogTheme is an inherited theme that overrides the DialogThemeData of the theme for the
// widgets below it.
//
// Flutter: material/dialog_theme.dart DialogTheme
type DialogTheme struct {
	widgets.InheritedTheme
	data DialogThemeData
}

// NewDialogTheme creates the theme.
//
// Flutter: material/dialog_theme.dart DialogTheme.new
func NewDialogTheme(data DialogThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *DialogTheme {
	return &DialogTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/dialog_theme.dart DialogTheme.data
func (w *DialogTheme) Data() DialogThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/dialog_theme.dart DialogTheme.updateShouldNotify
func (w *DialogTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*DialogTheme).data)
}

// Wrap makes a DialogTheme with the same data around the child.
//
// Flutter: material/dialog_theme.dart DialogTheme.wrap
func (w *DialogTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewDialogTheme(w.data, child)
}

// DialogThemeOf returns the data of the closest DialogTheme above the context, or the
// dialogTheme of the closest Theme. The context depends on it.
//
// Flutter: material/dialog_theme.dart DialogTheme.of
func DialogThemeOf(bc widgets.BuildContext) DialogThemeData {
	if t, ok := widgets.DependOn[*DialogTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).DialogTheme
}
