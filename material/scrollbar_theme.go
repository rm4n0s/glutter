// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ScrollbarThemeData defines default property values for descendant Scrollbar widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/scrollbar_theme.dart ScrollbarThemeData
//
//glutter:copywith value
type ScrollbarThemeData struct {
	// ThumbVisibility is whether the thumb is visible, for each state.
	ThumbVisibility widgets.WidgetStateProperty[foundation.Opt[bool]]
	// Thickness is the thickness of the thumb, for each state.
	Thickness widgets.WidgetStateProperty[foundation.Opt[float64]]
	// TrackVisibility is whether the track is visible, for each state.
	TrackVisibility widgets.WidgetStateProperty[foundation.Opt[bool]]
	// Interactive is whether the thumb can be dragged.
	Interactive foundation.Opt[bool]
	// Radius is the radius of the corners of the thumb.
	Radius foundation.Opt[ui.Radius]
	// ThumbColor is the color of the thumb, for each state.
	ThumbColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TrackColor is the color of the track, for each state.
	TrackColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TrackBorderColor is the color of the border of the track, for each state.
	TrackBorderColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// CrossAxisMargin is the margin between the scrollbar and the edge of the view across the scroll direction.
	CrossAxisMargin foundation.Opt[float64]
	// MainAxisMargin is the margin at the two ends of the scrollbar along the scroll direction.
	MainAxisMargin foundation.Opt[float64]
	// MinThumbLength is the minimum length of the thumb.
	MinThumbLength foundation.Opt[float64]
}

// ScrollbarThemeDataOpts are the parameters of NewScrollbarThemeData: the fields of the data.
//
// Flutter: material/scrollbar_theme.dart ScrollbarThemeData.new
type ScrollbarThemeDataOpts = ScrollbarThemeData

// NewScrollbarThemeData creates the data; every field is optional.
//
// Flutter: material/scrollbar_theme.dart ScrollbarThemeData.new
func NewScrollbarThemeData(opts ...ScrollbarThemeDataOpts) ScrollbarThemeData {
	return firstOpts(opts)
}

// ScrollbarThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/scrollbar_theme.dart ScrollbarThemeData.lerp
func ScrollbarThemeDataLerp(a, b ScrollbarThemeData, t float64) ScrollbarThemeData {
	return ScrollbarThemeData{
		ThumbVisibility:  lerpStateOpt(a.ThumbVisibility, b.ThumbVisibility, t, lerpFirstOrSecond[foundation.Opt[bool]]),
		Thickness:        lerpStateOptDouble(a.Thickness, b.Thickness, t),
		TrackVisibility:  lerpStateOpt(a.TrackVisibility, b.TrackVisibility, t, lerpFirstOrSecond[foundation.Opt[bool]]),
		Interactive:      lerpFirstOrSecond(a.Interactive, b.Interactive, t),
		Radius:           lerpOptRadius(a.Radius, b.Radius, t),
		ThumbColor:       lerpStateOptColor(a.ThumbColor, b.ThumbColor, t),
		TrackColor:       lerpStateOptColor(a.TrackColor, b.TrackColor, t),
		TrackBorderColor: lerpStateOptColor(a.TrackBorderColor, b.TrackBorderColor, t),
		CrossAxisMargin:  lerpOptDouble(a.CrossAxisMargin, b.CrossAxisMargin, t),
		MainAxisMargin:   lerpOptDouble(a.MainAxisMargin, b.MainAxisMargin, t),
		MinThumbLength:   lerpOptDouble(a.MinThumbLength, b.MinThumbLength, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/scrollbar_theme.dart ScrollbarThemeData.==
func (d ScrollbarThemeData) Equal(o ScrollbarThemeData) bool {
	return foundation.ValuesEqual(d.ThumbVisibility, o.ThumbVisibility) &&
		foundation.ValuesEqual(d.Thickness, o.Thickness) &&
		foundation.ValuesEqual(d.TrackVisibility, o.TrackVisibility) &&
		d.Interactive == o.Interactive &&
		d.Radius == o.Radius &&
		foundation.ValuesEqual(d.ThumbColor, o.ThumbColor) &&
		foundation.ValuesEqual(d.TrackColor, o.TrackColor) &&
		foundation.ValuesEqual(d.TrackBorderColor, o.TrackBorderColor) &&
		d.CrossAxisMargin == o.CrossAxisMargin &&
		d.MainAxisMargin == o.MainAxisMargin &&
		d.MinThumbLength == o.MinThumbLength
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (ScrollbarThemeData) ToStringShort() string { return "ScrollbarThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d ScrollbarThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/scrollbar_theme.dart ScrollbarThemeData.debugFillProperties
func (d ScrollbarThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("thumbVisibility", propOrNil(d.ThumbVisibility)))
	p.Add(objectProp("thickness", propOrNil(d.Thickness)))
	p.Add(objectProp("trackVisibility", propOrNil(d.TrackVisibility)))
	p.Add(objectProp("interactive", optValue(d.Interactive)))
	p.Add(objectProp("radius", optValue(d.Radius)))
	p.Add(objectProp("thumbColor", propOrNil(d.ThumbColor)))
	p.Add(objectProp("trackColor", propOrNil(d.TrackColor)))
	p.Add(objectProp("trackBorderColor", propOrNil(d.TrackBorderColor)))
	p.Add(doubleProp("crossAxisMargin", d.CrossAxisMargin))
	p.Add(doubleProp("mainAxisMargin", d.MainAxisMargin))
	p.Add(doubleProp("minThumbLength", d.MinThumbLength))
}

// ScrollbarTheme is an inherited theme that overrides the ScrollbarThemeData of the theme for the
// widgets below it.
//
// Flutter: material/scrollbar_theme.dart ScrollbarTheme
type ScrollbarTheme struct {
	widgets.InheritedTheme
	data ScrollbarThemeData
}

// NewScrollbarTheme creates the theme.
//
// Flutter: material/scrollbar_theme.dart ScrollbarTheme.new
func NewScrollbarTheme(data ScrollbarThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *ScrollbarTheme {
	return &ScrollbarTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/scrollbar_theme.dart ScrollbarTheme.data
func (w *ScrollbarTheme) Data() ScrollbarThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/scrollbar_theme.dart ScrollbarTheme.updateShouldNotify
func (w *ScrollbarTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*ScrollbarTheme).data)
}

// Wrap makes a ScrollbarTheme with the same data around the child.
//
// Flutter: material/scrollbar_theme.dart ScrollbarTheme.wrap
func (w *ScrollbarTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewScrollbarTheme(w.data, child)
}

// ScrollbarThemeOf returns the data of the closest ScrollbarTheme above the context, or the
// scrollbarTheme of the closest Theme. The context depends on it.
//
// Flutter: material/scrollbar_theme.dart ScrollbarTheme.of
func ScrollbarThemeOf(bc widgets.BuildContext) ScrollbarThemeData {
	if t, ok := widgets.DependOn[*ScrollbarTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).ScrollbarTheme
}
