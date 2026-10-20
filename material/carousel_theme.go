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

// CarouselViewThemeData defines default property values for descendant CarouselView widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/carousel_theme.dart CarouselViewThemeData
//
//glutter:copywith value
type CarouselViewThemeData struct {
	// Padding is the padding around each item.
	Padding painting.EdgeInsetsGeometry
	// BackgroundColor is the background color of the items.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the elevation of the items.
	Elevation foundation.Opt[float64]
	// Shape is the shape of the items.
	Shape painting.ShapeBorder
	// ItemClipBehavior is how the items are clipped.
	ItemClipBehavior ui.Clip
	// OverlayColor is the color of the overlay of an item, for each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
}

// CarouselViewThemeDataOpts are the parameters of NewCarouselViewThemeData: the fields of the data.
//
// Flutter: material/carousel_theme.dart CarouselViewThemeData.new
type CarouselViewThemeDataOpts = CarouselViewThemeData

// NewCarouselViewThemeData creates the data; every field is optional.
//
// Flutter: material/carousel_theme.dart CarouselViewThemeData.new
func NewCarouselViewThemeData(opts ...CarouselViewThemeDataOpts) CarouselViewThemeData {
	return firstOpts(opts)
}

// CarouselViewThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/carousel_theme.dart CarouselViewThemeData.lerp
func CarouselViewThemeDataLerp(a, b CarouselViewThemeData, t float64) CarouselViewThemeData {
	return CarouselViewThemeData{
		Padding:          painting.EdgeInsetsGeometryLerp(a.Padding, b.Padding, t),
		BackgroundColor:  lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		Elevation:        lerpOptDouble(a.Elevation, b.Elevation, t),
		Shape:            painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		ItemClipBehavior: lerpFirstOrSecond(a.ItemClipBehavior, b.ItemClipBehavior, t),
		OverlayColor:     lerpStateOptColor(a.OverlayColor, b.OverlayColor, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/carousel_theme.dart CarouselViewThemeData.==
func (d CarouselViewThemeData) Equal(o CarouselViewThemeData) bool {
	return foundation.ValuesEqual(d.Padding, o.Padding) &&
		d.BackgroundColor == o.BackgroundColor &&
		d.Elevation == o.Elevation &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		d.ItemClipBehavior == o.ItemClipBehavior &&
		foundation.ValuesEqual(d.OverlayColor, o.OverlayColor)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (CarouselViewThemeData) ToStringShort() string { return "CarouselViewThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d CarouselViewThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/carousel_theme.dart CarouselViewThemeData.debugFillProperties
func (d CarouselViewThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("padding", d.Padding))
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("itemClipBehavior", nilIfZero(d.ItemClipBehavior)))
	p.Add(objectProp("overlayColor", propOrNil(d.OverlayColor)))
}

// CarouselViewTheme is an inherited theme that overrides the CarouselViewThemeData of the theme for the
// widgets below it.
//
// Flutter: material/carousel_theme.dart CarouselViewTheme
type CarouselViewTheme struct {
	widgets.InheritedTheme
	data CarouselViewThemeData
}

// NewCarouselViewTheme creates the theme.
//
// Flutter: material/carousel_theme.dart CarouselViewTheme.new
func NewCarouselViewTheme(data CarouselViewThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *CarouselViewTheme {
	return &CarouselViewTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/carousel_theme.dart CarouselViewTheme.data
func (w *CarouselViewTheme) Data() CarouselViewThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/carousel_theme.dart CarouselViewTheme.updateShouldNotify
func (w *CarouselViewTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*CarouselViewTheme).data)
}

// Wrap makes a CarouselViewTheme with the same data around the child.
//
// Flutter: material/carousel_theme.dart CarouselViewTheme.wrap
func (w *CarouselViewTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewCarouselViewTheme(w.data, child)
}

// CarouselViewThemeOf returns the data of the closest CarouselViewTheme above the context, or the
// carouselViewTheme of the closest Theme. The context depends on it.
//
// Flutter: material/carousel_theme.dart CarouselViewTheme.of
func CarouselViewThemeOf(bc widgets.BuildContext) CarouselViewThemeData {
	if t, ok := widgets.DependOn[*CarouselViewTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).CarouselViewTheme
}
