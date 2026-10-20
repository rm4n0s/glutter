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

// SliderThemeData defines default property values for descendant Slider widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/slider_theme.dart SliderThemeData
//
//glutter:copywith value
type SliderThemeData struct {
	// TrackHeight is the height of the track.
	TrackHeight foundation.Opt[float64]
	// ActiveTrackColor is the color of the active part of the track, between the start of the slider and the thumb.
	ActiveTrackColor foundation.Opt[ui.Color]
	// InactiveTrackColor is the color of the inactive part of the track.
	InactiveTrackColor foundation.Opt[ui.Color]
	// SecondaryActiveTrackColor is the color of the secondary active part of the track, which shows a second value such as the buffered part of a video.
	SecondaryActiveTrackColor foundation.Opt[ui.Color]
	// DisabledActiveTrackColor is the color of the active part of the track when the slider is disabled.
	DisabledActiveTrackColor foundation.Opt[ui.Color]
	// DisabledInactiveTrackColor is the color of the inactive part of the track when the slider is disabled.
	DisabledInactiveTrackColor foundation.Opt[ui.Color]
	// DisabledSecondaryActiveTrackColor is the color of the secondary active part of the track when the slider is disabled.
	DisabledSecondaryActiveTrackColor foundation.Opt[ui.Color]
	// ActiveTickMarkColor is the color of the tick marks in the active part of the track.
	ActiveTickMarkColor foundation.Opt[ui.Color]
	// InactiveTickMarkColor is the color of the tick marks in the inactive part of the track.
	InactiveTickMarkColor foundation.Opt[ui.Color]
	// DisabledActiveTickMarkColor is the color of the tick marks in the active part of the track when the slider is disabled.
	DisabledActiveTickMarkColor foundation.Opt[ui.Color]
	// DisabledInactiveTickMarkColor is the color of the tick marks in the inactive part of the track when the slider is disabled.
	DisabledInactiveTickMarkColor foundation.Opt[ui.Color]
	// ThumbColor is the color of the thumb.
	ThumbColor foundation.Opt[ui.Color]
	// OverlappingShapeStrokeColor is the color of the stroke of the thumb or the value indicator that is above the other one in a RangeSlider.
	OverlappingShapeStrokeColor foundation.Opt[ui.Color]
	// DisabledThumbColor is the color of the thumb when the slider is disabled.
	DisabledThumbColor foundation.Opt[ui.Color]
	// OverlayColor is the color of the overlay around the thumb while it is pressed, focused or hovered.
	OverlayColor foundation.Opt[ui.Color]
	// StateOverlayColor is the color of the overlay in each state (Flutter's WidgetStateColor as overlayColor); when it is set, OverlayColor is ignored.
	StateOverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// ValueIndicatorColor is the color of the value indicator.
	ValueIndicatorColor foundation.Opt[ui.Color]
	// ValueIndicatorStrokeColor is the color of the stroke of the value indicator.
	ValueIndicatorStrokeColor foundation.Opt[ui.Color]
	// OverlayShape is the shape of the overlay.
	OverlayShape SliderComponentShape
	// TickMarkShape is the shape of the tick marks.
	TickMarkShape SliderTickMarkShape
	// ThumbShape is the shape of the thumb.
	ThumbShape SliderComponentShape
	// TrackShape is the shape of the track.
	TrackShape SliderTrackShape
	// ValueIndicatorShape is the shape of the value indicator.
	ValueIndicatorShape SliderComponentShape
	// RangeTickMarkShape is the shape of the tick marks of a RangeSlider.
	RangeTickMarkShape RangeSliderTickMarkShape
	// RangeThumbShape is the shape of the thumbs of a RangeSlider.
	RangeThumbShape RangeSliderThumbShape
	// RangeTrackShape is the shape of the track of a RangeSlider.
	RangeTrackShape RangeSliderTrackShape
	// RangeValueIndicatorShape is the shape of the value indicators of a RangeSlider.
	RangeValueIndicatorShape RangeSliderValueIndicatorShape
	// ShowValueIndicator is when the value indicator shows.
	ShowValueIndicator ShowValueIndicator
	// ValueIndicatorTextStyle is the text style of the label of the value indicator.
	ValueIndicatorTextStyle *painting.TextStyle
	// MinThumbSeparation is the minimum distance between the thumbs of a RangeSlider.
	MinThumbSeparation foundation.Opt[float64]
	// ThumbSelector chooses the thumb of a RangeSlider that a press moves.
	ThumbSelector RangeThumbSelector
	// MouseCursor is the cursor over the slider, in each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// AllowedInteraction is the way the user may change the value.
	AllowedInteraction SliderInteraction
	// Padding is the padding around the slider.
	Padding painting.EdgeInsetsGeometry
	// ThumbSize is the size of the thumb of Material 3, in each state.
	ThumbSize widgets.WidgetStateProperty[foundation.Opt[ui.Size]]
	// TrackGap is the gap between the thumb and the track of Material 3.
	TrackGap foundation.Opt[float64]
	// Year2023 says whether the slider has the look of 2023 (the default) or the one of 2024.
	//
	// Deprecated: Flutter deprecates it after v3.27; set it to false to opt into the 2024 look.
	Year2023 foundation.Opt[bool]
}

// SliderThemeDataOpts are the parameters of NewSliderThemeData: the fields of the data.
//
// Flutter: material/slider_theme.dart SliderThemeData.new
type SliderThemeDataOpts = SliderThemeData

// NewSliderThemeData creates the data; every field is optional.
//
// Flutter: material/slider_theme.dart SliderThemeData.new
func NewSliderThemeData(opts ...SliderThemeDataOpts) SliderThemeData {
	return firstOpts(opts)
}

// SliderThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/slider_theme.dart SliderThemeData.lerp
func SliderThemeDataLerp(a, b SliderThemeData, t float64) SliderThemeData {
	return SliderThemeData{
		TrackHeight:                       lerpOptDouble(a.TrackHeight, b.TrackHeight, t),
		ActiveTrackColor:                  lerpOptColor(a.ActiveTrackColor, b.ActiveTrackColor, t),
		InactiveTrackColor:                lerpOptColor(a.InactiveTrackColor, b.InactiveTrackColor, t),
		SecondaryActiveTrackColor:         lerpOptColor(a.SecondaryActiveTrackColor, b.SecondaryActiveTrackColor, t),
		DisabledActiveTrackColor:          lerpOptColor(a.DisabledActiveTrackColor, b.DisabledActiveTrackColor, t),
		DisabledInactiveTrackColor:        lerpOptColor(a.DisabledInactiveTrackColor, b.DisabledInactiveTrackColor, t),
		DisabledSecondaryActiveTrackColor: lerpOptColor(a.DisabledSecondaryActiveTrackColor, b.DisabledSecondaryActiveTrackColor, t),
		ActiveTickMarkColor:               lerpOptColor(a.ActiveTickMarkColor, b.ActiveTickMarkColor, t),
		InactiveTickMarkColor:             lerpOptColor(a.InactiveTickMarkColor, b.InactiveTickMarkColor, t),
		DisabledActiveTickMarkColor:       lerpOptColor(a.DisabledActiveTickMarkColor, b.DisabledActiveTickMarkColor, t),
		DisabledInactiveTickMarkColor:     lerpOptColor(a.DisabledInactiveTickMarkColor, b.DisabledInactiveTickMarkColor, t),
		ThumbColor:                        lerpOptColor(a.ThumbColor, b.ThumbColor, t),
		OverlappingShapeStrokeColor:       lerpOptColor(a.OverlappingShapeStrokeColor, b.OverlappingShapeStrokeColor, t),
		DisabledThumbColor:                lerpOptColor(a.DisabledThumbColor, b.DisabledThumbColor, t),
		OverlayColor:                      lerpSliderOverlayColor(a, b, t),
		StateOverlayColor:                 nil,
		ValueIndicatorColor:               lerpOptColor(a.ValueIndicatorColor, b.ValueIndicatorColor, t),
		ValueIndicatorStrokeColor:         lerpOptColor(a.ValueIndicatorStrokeColor, b.ValueIndicatorStrokeColor, t),
		OverlayShape:                      lerpFirstOrSecond(a.OverlayShape, b.OverlayShape, t),
		TickMarkShape:                     lerpFirstOrSecond(a.TickMarkShape, b.TickMarkShape, t),
		ThumbShape:                        lerpFirstOrSecond(a.ThumbShape, b.ThumbShape, t),
		TrackShape:                        lerpFirstOrSecond(a.TrackShape, b.TrackShape, t),
		ValueIndicatorShape:               lerpFirstOrSecond(a.ValueIndicatorShape, b.ValueIndicatorShape, t),
		RangeTickMarkShape:                lerpFirstOrSecond(a.RangeTickMarkShape, b.RangeTickMarkShape, t),
		RangeThumbShape:                   lerpFirstOrSecond(a.RangeThumbShape, b.RangeThumbShape, t),
		RangeTrackShape:                   lerpFirstOrSecond(a.RangeTrackShape, b.RangeTrackShape, t),
		RangeValueIndicatorShape:          lerpFirstOrSecond(a.RangeValueIndicatorShape, b.RangeValueIndicatorShape, t),
		ShowValueIndicator:                lerpFirstOrSecond(a.ShowValueIndicator, b.ShowValueIndicator, t),
		ValueIndicatorTextStyle:           painting.TextStyleLerp(a.ValueIndicatorTextStyle, b.ValueIndicatorTextStyle, t),
		MinThumbSeparation:                lerpOptDouble(a.MinThumbSeparation, b.MinThumbSeparation, t),
		ThumbSelector:                     lerpFirstOrSecond(a.ThumbSelector, b.ThumbSelector, t),
		MouseCursor:                       lerpFirstOrSecond(a.MouseCursor, b.MouseCursor, t),
		AllowedInteraction:                lerpFirstOrSecond(a.AllowedInteraction, b.AllowedInteraction, t),
		Padding:                           painting.EdgeInsetsGeometryLerp(a.Padding, b.Padding, t),
		ThumbSize:                         lerpStateOpt(a.ThumbSize, b.ThumbSize, t, lerpOptSize),
		TrackGap:                          lerpOptDouble(a.TrackGap, b.TrackGap, t),
		Year2023:                          lerpFirstOrSecond(a.Year2023, b.Year2023, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/slider_theme.dart SliderThemeData.==
func (d SliderThemeData) Equal(o SliderThemeData) bool {
	return d.TrackHeight == o.TrackHeight &&
		d.ActiveTrackColor == o.ActiveTrackColor &&
		d.InactiveTrackColor == o.InactiveTrackColor &&
		d.SecondaryActiveTrackColor == o.SecondaryActiveTrackColor &&
		d.DisabledActiveTrackColor == o.DisabledActiveTrackColor &&
		d.DisabledInactiveTrackColor == o.DisabledInactiveTrackColor &&
		d.DisabledSecondaryActiveTrackColor == o.DisabledSecondaryActiveTrackColor &&
		d.ActiveTickMarkColor == o.ActiveTickMarkColor &&
		d.InactiveTickMarkColor == o.InactiveTickMarkColor &&
		d.DisabledActiveTickMarkColor == o.DisabledActiveTickMarkColor &&
		d.DisabledInactiveTickMarkColor == o.DisabledInactiveTickMarkColor &&
		d.ThumbColor == o.ThumbColor &&
		d.OverlappingShapeStrokeColor == o.OverlappingShapeStrokeColor &&
		d.DisabledThumbColor == o.DisabledThumbColor &&
		d.OverlayColor == o.OverlayColor &&
		foundation.ValuesEqual(d.StateOverlayColor, o.StateOverlayColor) &&
		d.ValueIndicatorColor == o.ValueIndicatorColor &&
		d.ValueIndicatorStrokeColor == o.ValueIndicatorStrokeColor &&
		foundation.ValuesEqual(d.OverlayShape, o.OverlayShape) &&
		foundation.ValuesEqual(d.TickMarkShape, o.TickMarkShape) &&
		foundation.ValuesEqual(d.ThumbShape, o.ThumbShape) &&
		foundation.ValuesEqual(d.TrackShape, o.TrackShape) &&
		foundation.ValuesEqual(d.ValueIndicatorShape, o.ValueIndicatorShape) &&
		foundation.ValuesEqual(d.RangeTickMarkShape, o.RangeTickMarkShape) &&
		foundation.ValuesEqual(d.RangeThumbShape, o.RangeThumbShape) &&
		foundation.ValuesEqual(d.RangeTrackShape, o.RangeTrackShape) &&
		foundation.ValuesEqual(d.RangeValueIndicatorShape, o.RangeValueIndicatorShape) &&
		d.ShowValueIndicator == o.ShowValueIndicator &&
		textStyleEqual(d.ValueIndicatorTextStyle, o.ValueIndicatorTextStyle) &&
		d.MinThumbSeparation == o.MinThumbSeparation &&
		sameFunc(d.ThumbSelector, o.ThumbSelector) &&
		foundation.ValuesEqual(d.MouseCursor, o.MouseCursor) &&
		d.AllowedInteraction == o.AllowedInteraction &&
		foundation.ValuesEqual(d.Padding, o.Padding) &&
		foundation.ValuesEqual(d.ThumbSize, o.ThumbSize) &&
		d.TrackGap == o.TrackGap &&
		d.Year2023 == o.Year2023
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (SliderThemeData) ToStringShort() string { return "SliderThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d SliderThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/slider_theme.dart SliderThemeData.debugFillProperties
func (d SliderThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(doubleProp("trackHeight", d.TrackHeight))
	p.Add(colorProp("activeTrackColor", d.ActiveTrackColor))
	p.Add(colorProp("inactiveTrackColor", d.InactiveTrackColor))
	p.Add(colorProp("secondaryActiveTrackColor", d.SecondaryActiveTrackColor))
	p.Add(colorProp("disabledActiveTrackColor", d.DisabledActiveTrackColor))
	p.Add(colorProp("disabledInactiveTrackColor", d.DisabledInactiveTrackColor))
	p.Add(colorProp("disabledSecondaryActiveTrackColor", d.DisabledSecondaryActiveTrackColor))
	p.Add(colorProp("activeTickMarkColor", d.ActiveTickMarkColor))
	p.Add(colorProp("inactiveTickMarkColor", d.InactiveTickMarkColor))
	p.Add(colorProp("disabledActiveTickMarkColor", d.DisabledActiveTickMarkColor))
	p.Add(colorProp("disabledInactiveTickMarkColor", d.DisabledInactiveTickMarkColor))
	p.Add(colorProp("thumbColor", d.ThumbColor))
	p.Add(colorProp("overlappingShapeStrokeColor", d.OverlappingShapeStrokeColor))
	p.Add(colorProp("disabledThumbColor", d.DisabledThumbColor))
	p.Add(colorProp("overlayColor", d.OverlayColor))
	p.Add(objectProp("stateOverlayColor", propOrNil(d.StateOverlayColor)))
	p.Add(colorProp("valueIndicatorColor", d.ValueIndicatorColor))
	p.Add(colorProp("valueIndicatorStrokeColor", d.ValueIndicatorStrokeColor))
	p.Add(objectProp("overlayShape", d.OverlayShape))
	p.Add(objectProp("tickMarkShape", d.TickMarkShape))
	p.Add(objectProp("thumbShape", d.ThumbShape))
	p.Add(objectProp("trackShape", d.TrackShape))
	p.Add(objectProp("valueIndicatorShape", d.ValueIndicatorShape))
	p.Add(objectProp("rangeTickMarkShape", d.RangeTickMarkShape))
	p.Add(objectProp("rangeThumbShape", d.RangeThumbShape))
	p.Add(objectProp("rangeTrackShape", d.RangeTrackShape))
	p.Add(objectProp("rangeValueIndicatorShape", d.RangeValueIndicatorShape))
	p.Add(objectProp("showValueIndicator", nilIfZero(d.ShowValueIndicator)))
	p.Add(objectProp("valueIndicatorTextStyle", d.ValueIndicatorTextStyle))
	p.Add(doubleProp("minThumbSeparation", d.MinThumbSeparation))
	p.Add(objectProp("thumbSelector", funcOrNil(d.ThumbSelector)))
	p.Add(objectProp("mouseCursor", propOrNil(d.MouseCursor)))
	p.Add(objectProp("allowedInteraction", nilIfZero(d.AllowedInteraction)))
	p.Add(objectProp("padding", d.Padding))
	p.Add(objectProp("thumbSize", propOrNil(d.ThumbSize)))
	p.Add(doubleProp("trackGap", d.TrackGap))
	p.Add(objectProp("year2023", optValue(d.Year2023)))
}

// SliderTheme is an inherited theme that overrides the SliderThemeData of the theme for the
// widgets below it.
//
// Flutter: material/slider_theme.dart SliderTheme
type SliderTheme struct {
	widgets.InheritedTheme
	data SliderThemeData
}

// NewSliderTheme creates the theme.
//
// Flutter: material/slider_theme.dart SliderTheme.new
func NewSliderTheme(data SliderThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *SliderTheme {
	return &SliderTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/slider_theme.dart SliderTheme.data
func (w *SliderTheme) Data() SliderThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/slider_theme.dart SliderTheme.updateShouldNotify
func (w *SliderTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*SliderTheme).data)
}

// Wrap makes a SliderTheme with the same data around the child.
//
// Flutter: material/slider_theme.dart SliderTheme.wrap
func (w *SliderTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewSliderTheme(w.data, child)
}

// SliderThemeOf returns the data of the closest SliderTheme above the context, or the
// sliderTheme of the closest Theme. The context depends on it.
//
// Flutter: material/slider_theme.dart SliderTheme.of
func SliderThemeOf(bc widgets.BuildContext) SliderThemeData {
	if t, ok := widgets.DependOn[*SliderTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).SliderTheme
}
