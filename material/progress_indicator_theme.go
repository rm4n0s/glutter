// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ProgressIndicatorThemeData defines default property values for descendant ProgressIndicator widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorThemeData
//
//glutter:copywith value
type ProgressIndicatorThemeData struct {
	// Color is the color of the active part of the indicators.
	Color foundation.Opt[ui.Color]
	// LinearTrackColor is the color of the track of a LinearProgressIndicator.
	LinearTrackColor foundation.Opt[ui.Color]
	// LinearMinHeight is the minimum height of a LinearProgressIndicator.
	LinearMinHeight foundation.Opt[float64]
	// CircularTrackColor is the color of the track of a CircularProgressIndicator.
	CircularTrackColor foundation.Opt[ui.Color]
	// RefreshBackgroundColor is the background color of a RefreshProgressIndicator.
	RefreshBackgroundColor foundation.Opt[ui.Color]
	// BorderRadius is the border radius of the LinearProgressIndicator.
	BorderRadius painting.BorderRadiusGeometry
	// StopIndicatorColor is the color of the stop indicator of a determinate LinearProgressIndicator.
	StopIndicatorColor foundation.Opt[ui.Color]
	// StopIndicatorRadius is the radius of the stop indicator.
	StopIndicatorRadius foundation.Opt[float64]
	// StrokeWidth is the width of the line of a CircularProgressIndicator.
	StrokeWidth foundation.Opt[float64]
	// StrokeAlign is where the stroke of a CircularProgressIndicator is placed relative to its boundary.
	StrokeAlign foundation.Opt[float64]
	// StrokeCap is the shape of the ends of the line of the indicators.
	StrokeCap foundation.Opt[ui.StrokeCap]
	// Constraints are the size constraints of a CircularProgressIndicator.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// TrackGap is the gap between the active part and the track.
	TrackGap foundation.Opt[float64]
	// CircularTrackPadding is the padding around the track of a CircularProgressIndicator.
	CircularTrackPadding painting.EdgeInsetsGeometry
	// Year2023 is true for the 2023 look of the indicators; Flutter's year2023, which defaults to true.
	Year2023 foundation.Opt[bool]
	// Controller is the controller of the indeterminate animation, when the theme provides it.
	Controller *animation.AnimationController
}

// ProgressIndicatorThemeDataOpts are the parameters of NewProgressIndicatorThemeData: the fields of the data.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorThemeData.new
type ProgressIndicatorThemeDataOpts = ProgressIndicatorThemeData

// NewProgressIndicatorThemeData creates the data; every field is optional.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorThemeData.new
func NewProgressIndicatorThemeData(opts ...ProgressIndicatorThemeDataOpts) ProgressIndicatorThemeData {
	return firstOpts(opts)
}

// ProgressIndicatorThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorThemeData.lerp
func ProgressIndicatorThemeDataLerp(a, b ProgressIndicatorThemeData, t float64) ProgressIndicatorThemeData {
	return ProgressIndicatorThemeData{
		Color:                  lerpOptColor(a.Color, b.Color, t),
		LinearTrackColor:       lerpOptColor(a.LinearTrackColor, b.LinearTrackColor, t),
		LinearMinHeight:        lerpOptDouble(a.LinearMinHeight, b.LinearMinHeight, t),
		CircularTrackColor:     lerpOptColor(a.CircularTrackColor, b.CircularTrackColor, t),
		RefreshBackgroundColor: lerpOptColor(a.RefreshBackgroundColor, b.RefreshBackgroundColor, t),
		BorderRadius:           painting.BorderRadiusGeometryLerp(a.BorderRadius, b.BorderRadius, t),
		StopIndicatorColor:     lerpOptColor(a.StopIndicatorColor, b.StopIndicatorColor, t),
		StopIndicatorRadius:    lerpOptDouble(a.StopIndicatorRadius, b.StopIndicatorRadius, t),
		StrokeWidth:            lerpOptDouble(a.StrokeWidth, b.StrokeWidth, t),
		StrokeAlign:            lerpOptDouble(a.StrokeAlign, b.StrokeAlign, t),
		StrokeCap:              lerpFirstOrSecond(a.StrokeCap, b.StrokeCap, t),
		Constraints:            lerpOptConstraints(a.Constraints, b.Constraints, t),
		TrackGap:               lerpOptDouble(a.TrackGap, b.TrackGap, t),
		CircularTrackPadding:   painting.EdgeInsetsGeometryLerp(a.CircularTrackPadding, b.CircularTrackPadding, t),
		Year2023:               lerpFirstOrSecond(a.Year2023, b.Year2023, t),
		Controller:             lerpFirstOrSecond(a.Controller, b.Controller, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorThemeData.==
func (d ProgressIndicatorThemeData) Equal(o ProgressIndicatorThemeData) bool {
	return d.Color == o.Color &&
		d.LinearTrackColor == o.LinearTrackColor &&
		d.LinearMinHeight == o.LinearMinHeight &&
		d.CircularTrackColor == o.CircularTrackColor &&
		d.RefreshBackgroundColor == o.RefreshBackgroundColor &&
		foundation.ValuesEqual(d.BorderRadius, o.BorderRadius) &&
		d.StopIndicatorColor == o.StopIndicatorColor &&
		d.StopIndicatorRadius == o.StopIndicatorRadius &&
		d.StrokeWidth == o.StrokeWidth &&
		d.StrokeAlign == o.StrokeAlign &&
		d.StrokeCap == o.StrokeCap &&
		d.Constraints == o.Constraints &&
		d.TrackGap == o.TrackGap &&
		foundation.ValuesEqual(d.CircularTrackPadding, o.CircularTrackPadding) &&
		d.Year2023 == o.Year2023 &&
		d.Controller == o.Controller
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (ProgressIndicatorThemeData) ToStringShort() string { return "ProgressIndicatorThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d ProgressIndicatorThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorThemeData.debugFillProperties
func (d ProgressIndicatorThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("color", d.Color))
	p.Add(colorProp("linearTrackColor", d.LinearTrackColor))
	p.Add(doubleProp("linearMinHeight", d.LinearMinHeight))
	p.Add(colorProp("circularTrackColor", d.CircularTrackColor))
	p.Add(colorProp("refreshBackgroundColor", d.RefreshBackgroundColor))
	p.Add(objectProp("borderRadius", d.BorderRadius))
	p.Add(colorProp("stopIndicatorColor", d.StopIndicatorColor))
	p.Add(doubleProp("stopIndicatorRadius", d.StopIndicatorRadius))
	p.Add(doubleProp("strokeWidth", d.StrokeWidth))
	p.Add(doubleProp("strokeAlign", d.StrokeAlign))
	p.Add(objectProp("strokeCap", optValue(d.StrokeCap)))
	p.Add(objectProp("constraints", optValue(d.Constraints)))
	p.Add(doubleProp("trackGap", d.TrackGap))
	p.Add(objectProp("circularTrackPadding", d.CircularTrackPadding))
	p.Add(objectProp("year2023", optValue(d.Year2023)))
	p.Add(objectProp("controller", d.Controller))
}

// ProgressIndicatorTheme is an inherited theme that overrides the ProgressIndicatorThemeData of the theme for the
// widgets below it.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorTheme
type ProgressIndicatorTheme struct {
	widgets.InheritedTheme
	data ProgressIndicatorThemeData
}

// NewProgressIndicatorTheme creates the theme.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorTheme.new
func NewProgressIndicatorTheme(data ProgressIndicatorThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *ProgressIndicatorTheme {
	return &ProgressIndicatorTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorTheme.data
func (w *ProgressIndicatorTheme) Data() ProgressIndicatorThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorTheme.updateShouldNotify
func (w *ProgressIndicatorTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*ProgressIndicatorTheme).data)
}

// Wrap makes a ProgressIndicatorTheme with the same data around the child.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorTheme.wrap
func (w *ProgressIndicatorTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewProgressIndicatorTheme(w.data, child)
}

// ProgressIndicatorThemeOf returns the data of the closest ProgressIndicatorTheme above the context, or the
// progressIndicatorTheme of the closest Theme. The context depends on it.
//
// Flutter: material/progress_indicator_theme.dart ProgressIndicatorTheme.of
func ProgressIndicatorThemeOf(bc widgets.BuildContext) ProgressIndicatorThemeData {
	if t, ok := widgets.DependOn[*ProgressIndicatorTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).ProgressIndicatorTheme
}
