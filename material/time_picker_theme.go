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

// TimePickerThemeData defines default property values for descendant TimePickerDialog widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/time_picker_theme.dart TimePickerThemeData
//
//glutter:copywith value
type TimePickerThemeData struct {
	// BackgroundColor is the background color of the time picker dialog.
	BackgroundColor foundation.Opt[ui.Color]
	// CancelButtonStyle is the style of the cancel button.
	CancelButtonStyle *ButtonStyle
	// ConfirmButtonStyle is the style of the confirm button.
	ConfirmButtonStyle *ButtonStyle
	// DayPeriodBorderSide is the border of the AM/PM toggle.
	DayPeriodBorderSide foundation.Opt[painting.BorderSide]
	// DayPeriodColor is the background color of the AM/PM toggle when it is selected; the toggle is transparent otherwise. DayPeriodStateColor wins when it is set.
	DayPeriodColor foundation.Opt[ui.Color]
	// DayPeriodStateColor is the background color of the AM/PM toggle for each state; it wins over DayPeriodColor. Flutter passes a WidgetStateColor in dayPeriodColor.
	DayPeriodStateColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// DayPeriodShape is the shape of the AM/PM toggle.
	DayPeriodShape painting.OutlinedBorder
	// DayPeriodTextColor is the color of the AM/PM text. DayPeriodTextStateColor wins when it is set.
	DayPeriodTextColor foundation.Opt[ui.Color]
	// DayPeriodTextStateColor is the color of the AM/PM text for each state; it wins over DayPeriodTextColor.
	DayPeriodTextStateColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// DayPeriodTextStyle is the style of the AM/PM text.
	DayPeriodTextStyle *painting.TextStyle
	// DialBackgroundColor is the background color of the dial.
	DialBackgroundColor foundation.Opt[ui.Color]
	// DialHandColor is the color of the hand of the dial.
	DialHandColor foundation.Opt[ui.Color]
	// DialTextColor is the color of the numbers on the dial. DialTextStateColor wins when it is set.
	DialTextColor foundation.Opt[ui.Color]
	// DialTextStateColor is the color of the numbers on the dial for each state; it wins over DialTextColor.
	DialTextStateColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// DialTextStyle is the style of the numbers on the dial.
	DialTextStyle *painting.TextStyle
	// Elevation is the elevation of the time picker dialog.
	Elevation foundation.Opt[float64]
	// EntryModeIconColor is the color of the button that switches the entry mode.
	EntryModeIconColor foundation.Opt[ui.Color]
	// HelpTextStyle is the style of the help text.
	HelpTextStyle *painting.TextStyle
	// HourMinuteColor is the background color of the hour and the minute. HourMinuteStateColor wins when it is set.
	HourMinuteColor foundation.Opt[ui.Color]
	// HourMinuteStateColor is the background color of the hour and the minute for each state; it wins over HourMinuteColor.
	HourMinuteStateColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// HourMinuteShape is the shape of the hour and the minute.
	HourMinuteShape painting.ShapeBorder
	// HourMinuteTextColor is the color of the text of the hour and the minute. HourMinuteTextStateColor wins when it is set.
	HourMinuteTextColor foundation.Opt[ui.Color]
	// HourMinuteTextStateColor is the color of the text of the hour and the minute for each state; it wins over HourMinuteTextColor.
	HourMinuteTextStateColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// HourMinuteTextStyle is the style of the text of the hour and the minute.
	HourMinuteTextStyle *painting.TextStyle
	// InputDecorationTheme is the decoration of the text fields of the input mode.
	InputDecorationTheme *InputDecorationThemeData
	// Padding is the padding around the contents of the dialog.
	Padding painting.EdgeInsetsGeometry
	// Shape is the shape of the time picker dialog.
	Shape painting.ShapeBorder
	// TimeSelectorSeparatorColor is the color of the colon between the hour and the minute.
	TimeSelectorSeparatorColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TimeSelectorSeparatorTextStyle is the style of the colon between the hour and the minute.
	TimeSelectorSeparatorTextStyle widgets.WidgetStateProperty[*painting.TextStyle]
}

// TimePickerThemeDataOpts are the parameters of NewTimePickerThemeData: the fields of the data.
//
// Flutter: material/time_picker_theme.dart TimePickerThemeData.new
type TimePickerThemeDataOpts = TimePickerThemeData

// NewTimePickerThemeData creates the data; every field is optional.
//
// Flutter: material/time_picker_theme.dart TimePickerThemeData.new
func NewTimePickerThemeData(opts ...TimePickerThemeDataOpts) TimePickerThemeData {
	return firstOpts(opts)
}

// TimePickerThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/time_picker_theme.dart TimePickerThemeData.lerp
func TimePickerThemeDataLerp(a, b TimePickerThemeData, t float64) TimePickerThemeData {
	return TimePickerThemeData{
		BackgroundColor:                lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		CancelButtonStyle:              ButtonStyleLerp(a.CancelButtonStyle, b.CancelButtonStyle, t),
		ConfirmButtonStyle:             ButtonStyleLerp(a.ConfirmButtonStyle, b.ConfirmButtonStyle, t),
		DayPeriodBorderSide:            lerpTimePickerBorderSide(a.DayPeriodBorderSide, b.DayPeriodBorderSide, t),
		DayPeriodColor:                 lerpOptColor(a.DayPeriodColor, b.DayPeriodColor, t),
		DayPeriodStateColor:            lerpStateOptColor(a.DayPeriodStateColor, b.DayPeriodStateColor, t),
		DayPeriodShape:                 painting.OutlinedBorderLerp(a.DayPeriodShape, b.DayPeriodShape, t),
		DayPeriodTextColor:             lerpOptColor(a.DayPeriodTextColor, b.DayPeriodTextColor, t),
		DayPeriodTextStateColor:        lerpStateOptColor(a.DayPeriodTextStateColor, b.DayPeriodTextStateColor, t),
		DayPeriodTextStyle:             painting.TextStyleLerp(a.DayPeriodTextStyle, b.DayPeriodTextStyle, t),
		DialBackgroundColor:            lerpOptColor(a.DialBackgroundColor, b.DialBackgroundColor, t),
		DialHandColor:                  lerpOptColor(a.DialHandColor, b.DialHandColor, t),
		DialTextColor:                  lerpOptColor(a.DialTextColor, b.DialTextColor, t),
		DialTextStateColor:             lerpStateOptColor(a.DialTextStateColor, b.DialTextStateColor, t),
		DialTextStyle:                  painting.TextStyleLerp(a.DialTextStyle, b.DialTextStyle, t),
		Elevation:                      lerpOptDouble(a.Elevation, b.Elevation, t),
		EntryModeIconColor:             lerpOptColor(a.EntryModeIconColor, b.EntryModeIconColor, t),
		HelpTextStyle:                  painting.TextStyleLerp(a.HelpTextStyle, b.HelpTextStyle, t),
		HourMinuteColor:                lerpOptColor(a.HourMinuteColor, b.HourMinuteColor, t),
		HourMinuteStateColor:           lerpStateOptColor(a.HourMinuteStateColor, b.HourMinuteStateColor, t),
		HourMinuteShape:                painting.ShapeBorderLerp(a.HourMinuteShape, b.HourMinuteShape, t),
		HourMinuteTextColor:            lerpOptColor(a.HourMinuteTextColor, b.HourMinuteTextColor, t),
		HourMinuteTextStateColor:       lerpStateOptColor(a.HourMinuteTextStateColor, b.HourMinuteTextStateColor, t),
		HourMinuteTextStyle:            painting.TextStyleLerp(a.HourMinuteTextStyle, b.HourMinuteTextStyle, t),
		InputDecorationTheme:           lerpFirstOrSecond(a.InputDecorationTheme, b.InputDecorationTheme, t),
		Padding:                        painting.EdgeInsetsGeometryLerp(a.Padding, b.Padding, t),
		Shape:                          painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		TimeSelectorSeparatorColor:     lerpStateOptColor(a.TimeSelectorSeparatorColor, b.TimeSelectorSeparatorColor, t),
		TimeSelectorSeparatorTextStyle: lerpStateWith(a.TimeSelectorSeparatorTextStyle, b.TimeSelectorSeparatorTextStyle, t, lerpOptTextStyle),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/time_picker_theme.dart TimePickerThemeData.==
func (d TimePickerThemeData) Equal(o TimePickerThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor &&
		d.CancelButtonStyle.Equal(o.CancelButtonStyle) &&
		d.ConfirmButtonStyle.Equal(o.ConfirmButtonStyle) &&
		d.DayPeriodBorderSide == o.DayPeriodBorderSide &&
		d.DayPeriodColor == o.DayPeriodColor &&
		foundation.ValuesEqual(d.DayPeriodStateColor, o.DayPeriodStateColor) &&
		foundation.ValuesEqual(d.DayPeriodShape, o.DayPeriodShape) &&
		d.DayPeriodTextColor == o.DayPeriodTextColor &&
		foundation.ValuesEqual(d.DayPeriodTextStateColor, o.DayPeriodTextStateColor) &&
		textStyleEqual(d.DayPeriodTextStyle, o.DayPeriodTextStyle) &&
		d.DialBackgroundColor == o.DialBackgroundColor &&
		d.DialHandColor == o.DialHandColor &&
		d.DialTextColor == o.DialTextColor &&
		foundation.ValuesEqual(d.DialTextStateColor, o.DialTextStateColor) &&
		textStyleEqual(d.DialTextStyle, o.DialTextStyle) &&
		d.Elevation == o.Elevation &&
		d.EntryModeIconColor == o.EntryModeIconColor &&
		textStyleEqual(d.HelpTextStyle, o.HelpTextStyle) &&
		d.HourMinuteColor == o.HourMinuteColor &&
		foundation.ValuesEqual(d.HourMinuteStateColor, o.HourMinuteStateColor) &&
		foundation.ValuesEqual(d.HourMinuteShape, o.HourMinuteShape) &&
		d.HourMinuteTextColor == o.HourMinuteTextColor &&
		foundation.ValuesEqual(d.HourMinuteTextStateColor, o.HourMinuteTextStateColor) &&
		textStyleEqual(d.HourMinuteTextStyle, o.HourMinuteTextStyle) &&
		inputDecorationEqual(d.InputDecorationTheme, o.InputDecorationTheme) &&
		foundation.ValuesEqual(d.Padding, o.Padding) &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		foundation.ValuesEqual(d.TimeSelectorSeparatorColor, o.TimeSelectorSeparatorColor) &&
		foundation.ValuesEqual(d.TimeSelectorSeparatorTextStyle, o.TimeSelectorSeparatorTextStyle)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (TimePickerThemeData) ToStringShort() string { return "TimePickerThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d TimePickerThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/time_picker_theme.dart TimePickerThemeData.debugFillProperties
func (d TimePickerThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(objectProp("cancelButtonStyle", d.CancelButtonStyle))
	p.Add(objectProp("confirmButtonStyle", d.ConfirmButtonStyle))
	p.Add(objectProp("dayPeriodBorderSide", optValue(d.DayPeriodBorderSide)))
	p.Add(colorProp("dayPeriodColor", d.DayPeriodColor))
	p.Add(objectProp("dayPeriodStateColor", propOrNil(d.DayPeriodStateColor)))
	p.Add(objectProp("dayPeriodShape", d.DayPeriodShape))
	p.Add(colorProp("dayPeriodTextColor", d.DayPeriodTextColor))
	p.Add(objectProp("dayPeriodTextStateColor", propOrNil(d.DayPeriodTextStateColor)))
	p.Add(objectProp("dayPeriodTextStyle", d.DayPeriodTextStyle))
	p.Add(colorProp("dialBackgroundColor", d.DialBackgroundColor))
	p.Add(colorProp("dialHandColor", d.DialHandColor))
	p.Add(colorProp("dialTextColor", d.DialTextColor))
	p.Add(objectProp("dialTextStateColor", propOrNil(d.DialTextStateColor)))
	p.Add(objectProp("dialTextStyle", d.DialTextStyle))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(colorProp("entryModeIconColor", d.EntryModeIconColor))
	p.Add(objectProp("helpTextStyle", d.HelpTextStyle))
	p.Add(colorProp("hourMinuteColor", d.HourMinuteColor))
	p.Add(objectProp("hourMinuteStateColor", propOrNil(d.HourMinuteStateColor)))
	p.Add(objectProp("hourMinuteShape", d.HourMinuteShape))
	p.Add(colorProp("hourMinuteTextColor", d.HourMinuteTextColor))
	p.Add(objectProp("hourMinuteTextStateColor", propOrNil(d.HourMinuteTextStateColor)))
	p.Add(objectProp("hourMinuteTextStyle", d.HourMinuteTextStyle))
	p.Add(objectProp("inputDecorationTheme", d.InputDecorationTheme))
	p.Add(objectProp("padding", d.Padding))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("timeSelectorSeparatorColor", propOrNil(d.TimeSelectorSeparatorColor)))
	p.Add(objectProp("timeSelectorSeparatorTextStyle", propOrNil(d.TimeSelectorSeparatorTextStyle)))
}

// TimePickerTheme is an inherited theme that overrides the TimePickerThemeData of the theme for the
// widgets below it.
//
// Flutter: material/time_picker_theme.dart TimePickerTheme
type TimePickerTheme struct {
	widgets.InheritedTheme
	data TimePickerThemeData
}

// NewTimePickerTheme creates the theme.
//
// Flutter: material/time_picker_theme.dart TimePickerTheme.new
func NewTimePickerTheme(data TimePickerThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *TimePickerTheme {
	return &TimePickerTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/time_picker_theme.dart TimePickerTheme.data
func (w *TimePickerTheme) Data() TimePickerThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/time_picker_theme.dart TimePickerTheme.updateShouldNotify
func (w *TimePickerTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*TimePickerTheme).data)
}

// Wrap makes a TimePickerTheme with the same data around the child.
//
// Flutter: material/time_picker_theme.dart TimePickerTheme.wrap
func (w *TimePickerTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewTimePickerTheme(w.data, child)
}

// TimePickerThemeOf returns the data of the closest TimePickerTheme above the context, or the
// timePickerTheme of the closest Theme. The context depends on it.
//
// Flutter: material/time_picker_theme.dart TimePickerTheme.of
func TimePickerThemeOf(bc widgets.BuildContext) TimePickerThemeData {
	if t, ok := widgets.DependOn[*TimePickerTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).TimePickerTheme
}
