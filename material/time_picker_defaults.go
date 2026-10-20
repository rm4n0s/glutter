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

// timePickerDefaults are the defaults of the time picker: the values that its
// theme does not give.
//
// Deviation: Flutter's defaults are a TimePickerThemeData whose fields are
// overridden with WidgetStateColors; here the colors are properties of the
// states, and a text style is a plain style (SPEC §12.10).
//
// Flutter: material/time_picker.dart _TimePickerDefaults
type timePickerDefaults struct {
	backgroundColor        ui.Color
	cancelButtonStyle      *ButtonStyle
	confirmButtonStyle     *ButtonStyle
	dayPeriodBorderSide    painting.BorderSide
	dayPeriodColor         widgets.WidgetStateProperty[ui.Color]
	dayPeriodShape         painting.OutlinedBorder
	dayPeriodPortraitSize  ui.Size
	dayPeriodLandscapeSize ui.Size
	dayPeriodInputSize     ui.Size
	dayPeriodTextColor     widgets.WidgetStateProperty[ui.Color]
	dayPeriodTextStyle     *painting.TextStyle
	dialBackgroundColor    ui.Color
	dialHandColor          ui.Color
	dialSize               ui.Size
	handWidth              float64
	dotRadius              float64
	centerRadius           float64
	dialTextColor          widgets.WidgetStateProperty[ui.Color]
	dialTextStyle          *painting.TextStyle
	elevation              float64
	entryModeIconColor     ui.Color
	helpTextStyle          *painting.TextStyle
	hourMinuteColor        widgets.WidgetStateProperty[ui.Color]
	hourMinuteShape        painting.ShapeBorder
	hourMinuteSize         ui.Size
	hourMinuteSize24Hour   ui.Size
	hourMinuteInputSize    ui.Size
	hourMinuteInputSize24  ui.Size
	hourMinuteTextColor    widgets.WidgetStateProperty[ui.Color]
	hourMinuteTextStyle    *painting.TextStyle
	inputDecorationTheme   InputDecorationThemeData
	padding                painting.EdgeInsetsGeometry
	shape                  painting.ShapeBorder
	separatorColor         widgets.WidgetStateProperty[ui.Color]
	separatorTextStyle     *painting.TextStyle
	// source is the theme that the defaults come from.
	source *ThemeData
}

// timePickerDefaultsOf are the defaults of the theme of the context.
func timePickerDefaultsOf(bc widgets.BuildContext, entryMode TimePickerEntryMode) *timePickerDefaults {
	theme := ThemeOf(bc)
	if theme.UseMaterial3 {
		d := timePickerDefaultsM3(theme, entryMode)
		d.source = theme
		return d
	}
	d := timePickerDefaultsM2(theme)
	d.source = theme
	return d
}

// stateColor makes a color property from the function of the states.
func stateColor(f func(states widgets.WidgetStates) ui.Color) widgets.WidgetStateProperty[ui.Color] {
	return widgets.WidgetStatePropertyResolveWith(f)
}

// timePickerDefaultsM2 are the Material 2 defaults of the time picker.
//
// Flutter: material/time_picker.dart _TimePickerDefaultsM2
func timePickerDefaultsM2(theme *ThemeData) *timePickerDefaults {
	colors := theme.ColorScheme
	textTheme := theme.TextTheme
	dark := colors.Brightness() == ui.BrightnessDark
	pick := func(darkValue, lightValue float64) float64 {
		if dark {
			return darkValue
		}
		return lightValue
	}
	defaultShape := painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(4))})
	hourMinuteStyle := textTheme.DisplayMedium
	hourMinuteInputColor := stateColor(func(states widgets.WidgetStates) ui.Color {
		if states.Has(widgets.WidgetStateSelected) {
			return Colors.Transparent
		}
		return colors.OnSurface().WithOpacity(0.12)
	})
	return &timePickerDefaults{
		backgroundColor:     colors.Surface(),
		cancelButtonStyle:   TextButtonStyleFrom(ButtonStyleFromOpts{}),
		confirmButtonStyle:  TextButtonStyleFrom(ButtonStyleFromOpts{}),
		dayPeriodBorderSide: painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(ui.ColorAlphaBlend(colors.OnSurface().WithOpacity(0.38), colors.Surface()))}),
		dayPeriodColor: stateColor(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateSelected) {
				return colors.Primary().WithOpacity(pick(0.24, 0.12))
			}
			return Colors.Transparent
		}),
		dayPeriodShape:         defaultShape,
		dayPeriodPortraitSize:  ui.Size{Width: 52, Height: 80},
		dayPeriodLandscapeSize: ui.Size{Width: 0, Height: 40},
		dayPeriodInputSize:     ui.Size{Width: 52, Height: 70},
		dayPeriodTextColor: stateColor(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateSelected) {
				return colors.Primary()
			}
			return colors.OnSurface().WithOpacity(0.60)
		}),
		dayPeriodTextStyle:  textTheme.TitleMedium,
		dialBackgroundColor: colors.OnSurface().WithOpacity(pick(0.12, 0.08)),
		dialHandColor:       colors.Primary(),
		dialSize:            ui.Size{Width: 280, Height: 280},
		handWidth:           2,
		dotRadius:           22,
		centerRadius:        4,
		dialTextColor: stateColor(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateSelected) {
				return colors.Surface()
			}
			return colors.OnSurface()
		}),
		dialTextStyle:      textTheme.BodyLarge,
		elevation:          6,
		entryModeIconColor: colors.OnSurface().WithOpacity(pick(1.0, 0.6)),
		helpTextStyle:      textTheme.LabelSmall,
		hourMinuteColor: stateColor(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateSelected) {
				return colors.Primary().WithOpacity(pick(0.24, 0.12))
			}
			return colors.OnSurface().WithOpacity(0.12)
		}),
		hourMinuteShape:       defaultShape,
		hourMinuteSize:        ui.Size{Width: 96, Height: 80},
		hourMinuteSize24Hour:  ui.Size{Width: 114, Height: 80},
		hourMinuteInputSize:   ui.Size{Width: 96, Height: 70},
		hourMinuteInputSize24: ui.Size{Width: 114, Height: 70},
		hourMinuteTextColor: stateColor(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateSelected) {
				return colors.Primary()
			}
			return colors.OnSurface()
		}),
		hourMinuteTextStyle: hourMinuteStyle,
		inputDecorationTheme: InputDecorationThemeData{
			ContentPadding: painting.EdgeInsetsZero,
			Filled:         true,
			FillColor:      hourMinuteInputColor,
			FocusColor:     foundation.Some(Colors.Transparent),
			EnabledBorder: NewOutlineInputBorder(OutlineInputBorderOpts{
				BorderSide: painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(Colors.Transparent)})}),
			ErrorBorder: NewOutlineInputBorder(OutlineInputBorderOpts{
				BorderSide: painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.Error()), Width: foundation.Some(2.0)})}),
			FocusedBorder: NewOutlineInputBorder(OutlineInputBorderOpts{
				BorderSide: painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.Primary()), Width: foundation.Some(2.0)})}),
			FocusedErrorBorder: NewOutlineInputBorder(OutlineInputBorderOpts{
				BorderSide: painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.Error()), Width: foundation.Some(2.0)})}),
			HintStyle:  widgets.NewWidgetStatePropertyAll(hourMinuteStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurface().WithOpacity(0.36))})),
			ErrorStyle: widgets.NewWidgetStatePropertyAll(painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(0.0), Height: foundation.Some(1.0)})),
		},
		padding: painting.NewEdgeInsetsFromLTRB(8, 18, 8, 8),
		shape:   defaultShape,
	}
}

// timePickerDefaultsM3 are the Material 3 defaults of the time picker (generated
// from the Material token database in Flutter).
//
// Flutter: material/time_picker.dart _TimePickerDefaultsM3
func timePickerDefaultsM3(theme *ThemeData, entryMode TimePickerEntryMode) *timePickerDefaults {
	colors := theme.ColorScheme
	textTheme := theme.TextTheme
	selectorRadius := painting.NewBorderRadiusAll(ui.NewRadiusCircular(8))
	dayPeriodBorderSide := painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.Outline())})
	hourMinuteTextColor := stateColor(func(states widgets.WidgetStates) ui.Color {
		if states.Has(widgets.WidgetStateSelected) {
			return colors.OnPrimaryContainer()
		}
		return colors.OnSurface()
	})
	hourMinuteColor := stateColor(func(states widgets.WidgetStates) ui.Color {
		if states.Has(widgets.WidgetStateSelected) {
			overlay := colors.PrimaryContainer()
			switch {
			case states.Has(widgets.WidgetStatePressed):
				overlay = colors.OnPrimaryContainer()
			case states.Has(widgets.WidgetStateHovered):
				overlay = colors.OnPrimaryContainer().WithOpacity(0.08)
			case states.Has(widgets.WidgetStateFocused):
				overlay = colors.OnPrimaryContainer().WithOpacity(0.1)
			}
			return ui.ColorAlphaBlend(overlay, colors.PrimaryContainer())
		}
		overlay := colors.SurfaceContainerHighest()
		switch {
		case states.Has(widgets.WidgetStatePressed):
			overlay = colors.OnSurface()
		case states.Has(widgets.WidgetStateHovered):
			overlay = colors.OnSurface().WithOpacity(0.08)
		case states.Has(widgets.WidgetStateFocused):
			overlay = colors.OnSurface().WithOpacity(0.1)
		}
		return ui.ColorAlphaBlend(overlay, colors.SurfaceContainerHighest())
	})
	hourMinuteStyle := textTheme.DisplayMedium
	if entryMode == TimePickerEntryModeDial || entryMode == TimePickerEntryModeDialOnly || entryMode == TimePickerEntryModeUnspecified {
		hourMinuteStyle = textTheme.DisplayLarge
	}
	hourMinuteStyle = hourMinuteStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(hourMinuteTextColor.Resolve(0))})
	border := func(side painting.BorderSide) InputBorder {
		return NewOutlineInputBorder(OutlineInputBorderOpts{BorderRadius: foundation.Some(selectorRadius), BorderSide: side})
	}
	hourMinuteSize := ui.Size{Width: 96, Height: 80}
	hourMinuteSize24 := ui.Size{Width: 114, Height: hourMinuteSize.Height}
	dayPeriodPortraitSize := ui.Size{Width: 52, Height: 80}
	return &timePickerDefaults{
		backgroundColor:     colors.SurfaceContainerHigh(),
		cancelButtonStyle:   TextButtonStyleFrom(ButtonStyleFromOpts{}),
		confirmButtonStyle:  TextButtonStyleFrom(ButtonStyleFromOpts{}),
		dayPeriodBorderSide: dayPeriodBorderSide,
		dayPeriodColor: stateColor(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateSelected) {
				return colors.TertiaryContainer()
			}
			return Colors.Transparent
		}),
		dayPeriodShape:         painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: selectorRadius, Side: dayPeriodBorderSide}),
		dayPeriodPortraitSize:  dayPeriodPortraitSize,
		dayPeriodLandscapeSize: ui.Size{Width: 216, Height: 38},
		dayPeriodInputSize:     ui.Size{Width: dayPeriodPortraitSize.Width, Height: dayPeriodPortraitSize.Height - 8},
		dayPeriodTextColor: stateColor(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateSelected) {
				return colors.OnTertiaryContainer()
			}
			return colors.OnSurfaceVariant()
		}),
		dayPeriodTextStyle:  textTheme.TitleMedium,
		dialBackgroundColor: colors.SurfaceContainerHighest(),
		dialHandColor:       colors.Primary(),
		dialSize:            ui.Size{Width: 256, Height: 256},
		handWidth:           2,
		dotRadius:           24,
		centerRadius:        4,
		dialTextColor: stateColor(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateSelected) {
				return colors.OnPrimary()
			}
			return colors.OnSurface()
		}),
		dialTextStyle:      textTheme.BodyLarge,
		elevation:          6,
		entryModeIconColor: colors.OnSurface(),
		// Deviation: Flutter's help text style is a WidgetStateTextStyle, which
		// behaves like an empty style when a Text uses it as a plain style; the
		// style it resolves to is used here (SPEC §12.10).
		helpTextStyle:         textTheme.LabelMedium.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurfaceVariant())}),
		padding:               painting.NewEdgeInsetsAll(24),
		hourMinuteColor:       hourMinuteColor,
		hourMinuteShape:       painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: selectorRadius}),
		hourMinuteSize:        hourMinuteSize,
		hourMinuteSize24Hour:  hourMinuteSize24,
		hourMinuteInputSize:   ui.Size{Width: hourMinuteSize.Width, Height: hourMinuteSize.Height - 8},
		hourMinuteInputSize24: ui.Size{Width: hourMinuteSize24.Width, Height: hourMinuteSize24.Height - 8},
		hourMinuteTextColor:   hourMinuteTextColor,
		hourMinuteTextStyle:   hourMinuteStyle,
		inputDecorationTheme: InputDecorationThemeData{
			ContentPadding:     painting.EdgeInsetsZero,
			Filled:             true,
			FillColor:          hourMinuteColor,
			FocusColor:         foundation.Some(colors.PrimaryContainer()),
			EnabledBorder:      border(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(Colors.Transparent)})),
			ErrorBorder:        border(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.Error()), Width: foundation.Some(2.0)})),
			FocusedBorder:      border(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.Primary()), Width: foundation.Some(2.0)})),
			FocusedErrorBorder: border(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.Error()), Width: foundation.Some(2.0)})),
			HintStyle:          widgets.NewWidgetStatePropertyAll(hourMinuteStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurface().WithOpacity(0.36))})),
			ErrorStyle:         widgets.NewWidgetStatePropertyAll(painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(0.0)})),
		},
		shape:              painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(28))}),
		separatorColor:     widgets.NewWidgetStatePropertyAll(colors.OnSurface()),
		separatorTextStyle: textTheme.DisplayLarge,
	}
}

// timePickerThemeColor resolves a color of the time picker theme for the states:
// the property of the theme, then the plain color of the theme, then the default.
func timePickerThemeColor(plain foundation.Opt[ui.Color], property widgets.WidgetStateProperty[foundation.Opt[ui.Color]],
	def widgets.WidgetStateProperty[ui.Color], states widgets.WidgetStates) ui.Color {
	if property != nil {
		if c, ok := property.Resolve(states).Get(); ok {
			return c
		}
	}
	if c, ok := plain.Get(); ok {
		return c
	}
	return def.Resolve(states)
}
