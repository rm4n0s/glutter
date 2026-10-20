// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CardThemeData defines default property values for descendant Card widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/card_theme.dart CardThemeData
//
//glutter:copywith value
type CardThemeData struct {
	// ClipBehavior is how the card's content is clipped.
	ClipBehavior ui.Clip
	// Color is the card's background color.
	Color foundation.Opt[ui.Color]
	// ShadowColor is the color of the card's shadow.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the card (Material 3).
	SurfaceTintColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the card's Material.
	Elevation foundation.Opt[float64]
	// Margin is the empty space around the card.
	Margin painting.EdgeInsetsGeometry
	// Shape is the shape of the card's Material.
	Shape painting.ShapeBorder
}

// CardThemeDataOpts are the parameters of NewCardThemeData: the fields of the data.
//
// Flutter: material/card_theme.dart CardThemeData.new
type CardThemeDataOpts = CardThemeData

// NewCardThemeData creates the data; every field is optional.
//
// Flutter: material/card_theme.dart CardThemeData.new
func NewCardThemeData(opts ...CardThemeDataOpts) CardThemeData {
	return firstOpts(opts)
}

// CardThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/card_theme.dart CardThemeData.lerp
func CardThemeDataLerp(a, b CardThemeData, t float64) CardThemeData {
	return CardThemeData{
		ClipBehavior:     lerpFirstOrSecond(a.ClipBehavior, b.ClipBehavior, t),
		Color:            lerpOptColor(a.Color, b.Color, t),
		ShadowColor:      lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor: lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		Elevation:        lerpOptDouble(a.Elevation, b.Elevation, t),
		Margin:           painting.EdgeInsetsGeometryLerp(a.Margin, b.Margin, t),
		Shape:            painting.ShapeBorderLerp(a.Shape, b.Shape, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/card_theme.dart CardThemeData.==
func (d CardThemeData) Equal(o CardThemeData) bool {
	return d.ClipBehavior == o.ClipBehavior &&
		d.Color == o.Color &&
		d.ShadowColor == o.ShadowColor &&
		d.SurfaceTintColor == o.SurfaceTintColor &&
		d.Elevation == o.Elevation &&
		foundation.ValuesEqual(d.Margin, o.Margin) &&
		foundation.ValuesEqual(d.Shape, o.Shape)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (CardThemeData) ToStringShort() string { return "CardThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d CardThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/card_theme.dart CardThemeData.debugFillProperties
func (d CardThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("clipBehavior", nilIfZero(d.ClipBehavior)))
	p.Add(colorProp("color", d.Color))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(objectProp("margin", d.Margin))
	p.Add(objectProp("shape", d.Shape))
}

// CardTheme is an inherited theme that overrides the CardThemeData of the theme for the
// widgets below it.
//
// Flutter: material/card_theme.dart CardTheme
type CardTheme struct {
	widgets.InheritedTheme
	data CardThemeData
}

// NewCardTheme creates the theme.
//
// Flutter: material/card_theme.dart CardTheme.new
func NewCardTheme(data CardThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *CardTheme {
	return &CardTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/card_theme.dart CardTheme.data
func (w *CardTheme) Data() CardThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/card_theme.dart CardTheme.updateShouldNotify
func (w *CardTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*CardTheme).data)
}

// Wrap makes a CardTheme with the same data around the child.
//
// Flutter: material/card_theme.dart CardTheme.wrap
func (w *CardTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewCardTheme(w.data, child)
}

// CardThemeOf returns the data of the closest CardTheme above the context, or the
// cardTheme of the closest Theme. The context depends on it.
//
// Flutter: material/card_theme.dart CardTheme.of
func CardThemeOf(bc widgets.BuildContext) CardThemeData {
	if t, ok := widgets.DependOn[*CardTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).CardTheme
}
