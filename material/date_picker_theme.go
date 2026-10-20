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

// DatePickerThemeData defines default property values for descendant DatePickerDialog widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/date_picker_theme.dart DatePickerThemeData
//
//glutter:copywith value
type DatePickerThemeData struct {
	// BackgroundColor is the background color of the date picker dialog.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the elevation of the date picker dialog.
	Elevation foundation.Opt[float64]
	// ShadowColor is the shadow color of the date picker dialog.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the date picker dialog.
	SurfaceTintColor foundation.Opt[ui.Color]
	// Shape is the shape of the date picker dialog.
	Shape painting.ShapeBorder
	// HeaderBackgroundColor is the background color of the header of the dialog.
	HeaderBackgroundColor foundation.Opt[ui.Color]
	// HeaderForegroundColor is the color of the texts and icons of the header.
	HeaderForegroundColor foundation.Opt[ui.Color]
	// HeaderHeadlineStyle is the style of the selected date in the header.
	HeaderHeadlineStyle *painting.TextStyle
	// HeaderHelpStyle is the style of the help text in the header.
	HeaderHelpStyle *painting.TextStyle
	// WeekdayStyle is the style of the weekday labels above the days.
	WeekdayStyle *painting.TextStyle
	// DayStyle is the style of the day numbers.
	DayStyle *painting.TextStyle
	// DayForegroundColor is the color of the day numbers.
	DayForegroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// DayBackgroundColor is the background color of the days.
	DayBackgroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// DayOverlayColor is the overlay color of the days.
	DayOverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// DayShape is the shape of the days.
	DayShape widgets.WidgetStateProperty[painting.OutlinedBorder]
	// TodayForegroundColor is the color of today's number.
	TodayForegroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TodayBackgroundColor is the background color of today.
	TodayBackgroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TodayBorder is the border around today.
	TodayBorder foundation.Opt[painting.BorderSide]
	// YearStyle is the style of the years in the year picker.
	YearStyle *painting.TextStyle
	// YearForegroundColor is the color of the years.
	YearForegroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// YearBackgroundColor is the background color of the years.
	YearBackgroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// YearOverlayColor is the overlay color of the years.
	YearOverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// YearShape is the shape of the years.
	YearShape widgets.WidgetStateProperty[painting.OutlinedBorder]
	// RangePickerBackgroundColor is the background color of the date range picker.
	RangePickerBackgroundColor foundation.Opt[ui.Color]
	// RangePickerElevation is the elevation of the date range picker.
	RangePickerElevation foundation.Opt[float64]
	// RangePickerShadowColor is the shadow color of the date range picker.
	RangePickerShadowColor foundation.Opt[ui.Color]
	// RangePickerSurfaceTintColor is the surface tint of the date range picker.
	RangePickerSurfaceTintColor foundation.Opt[ui.Color]
	// RangePickerShape is the shape of the date range picker.
	RangePickerShape painting.ShapeBorder
	// RangePickerHeaderBackgroundColor is the background color of the header of the date range picker.
	RangePickerHeaderBackgroundColor foundation.Opt[ui.Color]
	// RangePickerHeaderForegroundColor is the color of the texts and icons of the header of the date range picker.
	RangePickerHeaderForegroundColor foundation.Opt[ui.Color]
	// RangePickerHeaderHeadlineStyle is the style of the headline of the date range picker.
	RangePickerHeaderHeadlineStyle *painting.TextStyle
	// RangePickerHeaderHelpStyle is the style of the help text of the date range picker.
	RangePickerHeaderHelpStyle *painting.TextStyle
	// RangeSelectionBackgroundColor is the background color of the days between the ends of a selected range.
	RangeSelectionBackgroundColor foundation.Opt[ui.Color]
	// RangeSelectionOverlayColor is the overlay color of the days in a selected range.
	RangeSelectionOverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// DividerColor is the color of the dividers.
	DividerColor foundation.Opt[ui.Color]
	// InputDecorationTheme is the decoration of the text fields of the input mode.
	InputDecorationTheme *InputDecorationThemeData
	// CancelButtonStyle is the style of the cancel button.
	CancelButtonStyle *ButtonStyle
	// ConfirmButtonStyle is the style of the confirm button.
	ConfirmButtonStyle *ButtonStyle
	// Locale is the locale of the date picker, used to format the dates.
	Locale foundation.Opt[ui.Locale]
	// ToggleButtonTextStyle is the style of the button that toggles the year picker.
	ToggleButtonTextStyle *painting.TextStyle
	// SubHeaderForegroundColor is the color of the sub header: the toggle button's icon and the month navigation.
	SubHeaderForegroundColor foundation.Opt[ui.Color]
}

// DatePickerThemeDataOpts are the parameters of NewDatePickerThemeData: the fields of the data.
//
// Flutter: material/date_picker_theme.dart DatePickerThemeData.new
type DatePickerThemeDataOpts = DatePickerThemeData

// NewDatePickerThemeData creates the data; every field is optional.
//
// Flutter: material/date_picker_theme.dart DatePickerThemeData.new
func NewDatePickerThemeData(opts ...DatePickerThemeDataOpts) DatePickerThemeData {
	return firstOpts(opts)
}

// DatePickerThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/date_picker_theme.dart DatePickerThemeData.lerp
func DatePickerThemeDataLerp(a, b DatePickerThemeData, t float64) DatePickerThemeData {
	return DatePickerThemeData{
		BackgroundColor:                  lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		Elevation:                        lerpOptDouble(a.Elevation, b.Elevation, t),
		ShadowColor:                      lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		SurfaceTintColor:                 lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		Shape:                            painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		HeaderBackgroundColor:            lerpOptColor(a.HeaderBackgroundColor, b.HeaderBackgroundColor, t),
		HeaderForegroundColor:            lerpOptColor(a.HeaderForegroundColor, b.HeaderForegroundColor, t),
		HeaderHeadlineStyle:              painting.TextStyleLerp(a.HeaderHeadlineStyle, b.HeaderHeadlineStyle, t),
		HeaderHelpStyle:                  painting.TextStyleLerp(a.HeaderHelpStyle, b.HeaderHelpStyle, t),
		WeekdayStyle:                     painting.TextStyleLerp(a.WeekdayStyle, b.WeekdayStyle, t),
		DayStyle:                         painting.TextStyleLerp(a.DayStyle, b.DayStyle, t),
		DayForegroundColor:               lerpStateOptColor(a.DayForegroundColor, b.DayForegroundColor, t),
		DayBackgroundColor:               lerpStateOptColor(a.DayBackgroundColor, b.DayBackgroundColor, t),
		DayOverlayColor:                  lerpStateOptColor(a.DayOverlayColor, b.DayOverlayColor, t),
		DayShape:                         lerpStateOutlinedBorder(a.DayShape, b.DayShape, t),
		TodayForegroundColor:             lerpStateOptColor(a.TodayForegroundColor, b.TodayForegroundColor, t),
		TodayBackgroundColor:             lerpStateOptColor(a.TodayBackgroundColor, b.TodayBackgroundColor, t),
		TodayBorder:                      lerpDatePickerBorderSide(a.TodayBorder, b.TodayBorder, t),
		YearStyle:                        painting.TextStyleLerp(a.YearStyle, b.YearStyle, t),
		YearForegroundColor:              lerpStateOptColor(a.YearForegroundColor, b.YearForegroundColor, t),
		YearBackgroundColor:              lerpStateOptColor(a.YearBackgroundColor, b.YearBackgroundColor, t),
		YearOverlayColor:                 lerpStateOptColor(a.YearOverlayColor, b.YearOverlayColor, t),
		YearShape:                        lerpStateOutlinedBorder(a.YearShape, b.YearShape, t),
		RangePickerBackgroundColor:       lerpOptColor(a.RangePickerBackgroundColor, b.RangePickerBackgroundColor, t),
		RangePickerElevation:             lerpOptDouble(a.RangePickerElevation, b.RangePickerElevation, t),
		RangePickerShadowColor:           lerpOptColor(a.RangePickerShadowColor, b.RangePickerShadowColor, t),
		RangePickerSurfaceTintColor:      lerpOptColor(a.RangePickerSurfaceTintColor, b.RangePickerSurfaceTintColor, t),
		RangePickerShape:                 painting.ShapeBorderLerp(a.RangePickerShape, b.RangePickerShape, t),
		RangePickerHeaderBackgroundColor: lerpOptColor(a.RangePickerHeaderBackgroundColor, b.RangePickerHeaderBackgroundColor, t),
		RangePickerHeaderForegroundColor: lerpOptColor(a.RangePickerHeaderForegroundColor, b.RangePickerHeaderForegroundColor, t),
		RangePickerHeaderHeadlineStyle:   painting.TextStyleLerp(a.RangePickerHeaderHeadlineStyle, b.RangePickerHeaderHeadlineStyle, t),
		RangePickerHeaderHelpStyle:       painting.TextStyleLerp(a.RangePickerHeaderHelpStyle, b.RangePickerHeaderHelpStyle, t),
		RangeSelectionBackgroundColor:    lerpOptColor(a.RangeSelectionBackgroundColor, b.RangeSelectionBackgroundColor, t),
		RangeSelectionOverlayColor:       lerpStateOptColor(a.RangeSelectionOverlayColor, b.RangeSelectionOverlayColor, t),
		DividerColor:                     lerpOptColor(a.DividerColor, b.DividerColor, t),
		InputDecorationTheme:             lerpFirstOrSecond(a.InputDecorationTheme, b.InputDecorationTheme, t),
		CancelButtonStyle:                ButtonStyleLerp(a.CancelButtonStyle, b.CancelButtonStyle, t),
		ConfirmButtonStyle:               ButtonStyleLerp(a.ConfirmButtonStyle, b.ConfirmButtonStyle, t),
		Locale:                           lerpFirstOrSecond(a.Locale, b.Locale, t),
		ToggleButtonTextStyle:            painting.TextStyleLerp(a.ToggleButtonTextStyle, b.ToggleButtonTextStyle, t),
		SubHeaderForegroundColor:         lerpOptColor(a.SubHeaderForegroundColor, b.SubHeaderForegroundColor, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/date_picker_theme.dart DatePickerThemeData.==
func (d DatePickerThemeData) Equal(o DatePickerThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor &&
		d.Elevation == o.Elevation &&
		d.ShadowColor == o.ShadowColor &&
		d.SurfaceTintColor == o.SurfaceTintColor &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		d.HeaderBackgroundColor == o.HeaderBackgroundColor &&
		d.HeaderForegroundColor == o.HeaderForegroundColor &&
		textStyleEqual(d.HeaderHeadlineStyle, o.HeaderHeadlineStyle) &&
		textStyleEqual(d.HeaderHelpStyle, o.HeaderHelpStyle) &&
		textStyleEqual(d.WeekdayStyle, o.WeekdayStyle) &&
		textStyleEqual(d.DayStyle, o.DayStyle) &&
		foundation.ValuesEqual(d.DayForegroundColor, o.DayForegroundColor) &&
		foundation.ValuesEqual(d.DayBackgroundColor, o.DayBackgroundColor) &&
		foundation.ValuesEqual(d.DayOverlayColor, o.DayOverlayColor) &&
		foundation.ValuesEqual(d.DayShape, o.DayShape) &&
		foundation.ValuesEqual(d.TodayForegroundColor, o.TodayForegroundColor) &&
		foundation.ValuesEqual(d.TodayBackgroundColor, o.TodayBackgroundColor) &&
		d.TodayBorder == o.TodayBorder &&
		textStyleEqual(d.YearStyle, o.YearStyle) &&
		foundation.ValuesEqual(d.YearForegroundColor, o.YearForegroundColor) &&
		foundation.ValuesEqual(d.YearBackgroundColor, o.YearBackgroundColor) &&
		foundation.ValuesEqual(d.YearOverlayColor, o.YearOverlayColor) &&
		foundation.ValuesEqual(d.YearShape, o.YearShape) &&
		d.RangePickerBackgroundColor == o.RangePickerBackgroundColor &&
		d.RangePickerElevation == o.RangePickerElevation &&
		d.RangePickerShadowColor == o.RangePickerShadowColor &&
		d.RangePickerSurfaceTintColor == o.RangePickerSurfaceTintColor &&
		foundation.ValuesEqual(d.RangePickerShape, o.RangePickerShape) &&
		d.RangePickerHeaderBackgroundColor == o.RangePickerHeaderBackgroundColor &&
		d.RangePickerHeaderForegroundColor == o.RangePickerHeaderForegroundColor &&
		textStyleEqual(d.RangePickerHeaderHeadlineStyle, o.RangePickerHeaderHeadlineStyle) &&
		textStyleEqual(d.RangePickerHeaderHelpStyle, o.RangePickerHeaderHelpStyle) &&
		d.RangeSelectionBackgroundColor == o.RangeSelectionBackgroundColor &&
		foundation.ValuesEqual(d.RangeSelectionOverlayColor, o.RangeSelectionOverlayColor) &&
		d.DividerColor == o.DividerColor &&
		inputDecorationEqual(d.InputDecorationTheme, o.InputDecorationTheme) &&
		d.CancelButtonStyle.Equal(o.CancelButtonStyle) &&
		d.ConfirmButtonStyle.Equal(o.ConfirmButtonStyle) &&
		foundation.ValuesEqual(d.Locale, o.Locale) &&
		textStyleEqual(d.ToggleButtonTextStyle, o.ToggleButtonTextStyle) &&
		d.SubHeaderForegroundColor == o.SubHeaderForegroundColor
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (DatePickerThemeData) ToStringShort() string { return "DatePickerThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d DatePickerThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/date_picker_theme.dart DatePickerThemeData.debugFillProperties
func (d DatePickerThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(objectProp("shape", d.Shape))
	p.Add(colorProp("headerBackgroundColor", d.HeaderBackgroundColor))
	p.Add(colorProp("headerForegroundColor", d.HeaderForegroundColor))
	p.Add(objectProp("headerHeadlineStyle", d.HeaderHeadlineStyle))
	p.Add(objectProp("headerHelpStyle", d.HeaderHelpStyle))
	p.Add(objectProp("weekdayStyle", d.WeekdayStyle))
	p.Add(objectProp("dayStyle", d.DayStyle))
	p.Add(objectProp("dayForegroundColor", propOrNil(d.DayForegroundColor)))
	p.Add(objectProp("dayBackgroundColor", propOrNil(d.DayBackgroundColor)))
	p.Add(objectProp("dayOverlayColor", propOrNil(d.DayOverlayColor)))
	p.Add(objectProp("dayShape", propOrNil(d.DayShape)))
	p.Add(objectProp("todayForegroundColor", propOrNil(d.TodayForegroundColor)))
	p.Add(objectProp("todayBackgroundColor", propOrNil(d.TodayBackgroundColor)))
	p.Add(objectProp("todayBorder", optValue(d.TodayBorder)))
	p.Add(objectProp("yearStyle", d.YearStyle))
	p.Add(objectProp("yearForegroundColor", propOrNil(d.YearForegroundColor)))
	p.Add(objectProp("yearBackgroundColor", propOrNil(d.YearBackgroundColor)))
	p.Add(objectProp("yearOverlayColor", propOrNil(d.YearOverlayColor)))
	p.Add(objectProp("yearShape", propOrNil(d.YearShape)))
	p.Add(colorProp("rangePickerBackgroundColor", d.RangePickerBackgroundColor))
	p.Add(doubleProp("rangePickerElevation", d.RangePickerElevation))
	p.Add(colorProp("rangePickerShadowColor", d.RangePickerShadowColor))
	p.Add(colorProp("rangePickerSurfaceTintColor", d.RangePickerSurfaceTintColor))
	p.Add(objectProp("rangePickerShape", d.RangePickerShape))
	p.Add(colorProp("rangePickerHeaderBackgroundColor", d.RangePickerHeaderBackgroundColor))
	p.Add(colorProp("rangePickerHeaderForegroundColor", d.RangePickerHeaderForegroundColor))
	p.Add(objectProp("rangePickerHeaderHeadlineStyle", d.RangePickerHeaderHeadlineStyle))
	p.Add(objectProp("rangePickerHeaderHelpStyle", d.RangePickerHeaderHelpStyle))
	p.Add(colorProp("rangeSelectionBackgroundColor", d.RangeSelectionBackgroundColor))
	p.Add(objectProp("rangeSelectionOverlayColor", propOrNil(d.RangeSelectionOverlayColor)))
	p.Add(colorProp("dividerColor", d.DividerColor))
	p.Add(objectProp("inputDecorationTheme", d.InputDecorationTheme))
	p.Add(objectProp("cancelButtonStyle", d.CancelButtonStyle))
	p.Add(objectProp("confirmButtonStyle", d.ConfirmButtonStyle))
	p.Add(objectProp("locale", optValue(d.Locale)))
	p.Add(objectProp("toggleButtonTextStyle", d.ToggleButtonTextStyle))
	p.Add(colorProp("subHeaderForegroundColor", d.SubHeaderForegroundColor))
}

// DatePickerTheme is an inherited theme that overrides the DatePickerThemeData of the theme for the
// widgets below it.
//
// Flutter: material/date_picker_theme.dart DatePickerTheme
type DatePickerTheme struct {
	widgets.InheritedTheme
	data DatePickerThemeData
}

// NewDatePickerTheme creates the theme.
//
// Flutter: material/date_picker_theme.dart DatePickerTheme.new
func NewDatePickerTheme(data DatePickerThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *DatePickerTheme {
	return &DatePickerTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/date_picker_theme.dart DatePickerTheme.data
func (w *DatePickerTheme) Data() DatePickerThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/date_picker_theme.dart DatePickerTheme.updateShouldNotify
func (w *DatePickerTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*DatePickerTheme).data)
}

// Wrap makes a DatePickerTheme with the same data around the child.
//
// Flutter: material/date_picker_theme.dart DatePickerTheme.wrap
func (w *DatePickerTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewDatePickerTheme(w.data, child)
}

// DatePickerThemeOf returns the data of the closest DatePickerTheme above the context, or the
// datePickerTheme of the closest Theme. The context depends on it.
//
// Flutter: material/date_picker_theme.dart DatePickerTheme.of
func DatePickerThemeOf(bc widgets.BuildContext) DatePickerThemeData {
	if t, ok := widgets.DependOn[*DatePickerTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).DatePickerTheme
}
