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

// lerpStateOutlinedBorder is WidgetStateProperty.lerp with OutlinedBorder.lerp.
//
// Flutter: widgets/widget_state.dart WidgetStateProperty.lerp
func lerpStateOutlinedBorder(a, b widgets.WidgetStateProperty[painting.OutlinedBorder], t float64) widgets.WidgetStateProperty[painting.OutlinedBorder] {
	if a == nil && b == nil {
		return nil
	}
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) painting.OutlinedBorder {
		var sa, sb painting.OutlinedBorder
		if a != nil {
			sa = a.Resolve(states)
		}
		if b != nil {
			sb = b.Resolve(states)
		}
		return painting.OutlinedBorderLerp(sa, sb, t)
	})
}

// lerpDatePickerBorderSide fades the border in from, and out to, a border of no
// width when only one of the two exists.
//
// Flutter: material/date_picker_theme.dart DatePickerThemeData._lerpBorderSide
func lerpDatePickerBorderSide(a, b foundation.Opt[painting.BorderSide], t float64) foundation.Opt[painting.BorderSide] {
	sa, okA := a.Get()
	sb, okB := b.Get()
	switch {
	case !okA && !okB:
		return foundation.None[painting.BorderSide]()
	case a == b:
		return a
	case !okA:
		return foundation.Some(painting.BorderSideLerp(painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(0.0), Color: foundation.Some(sb.Color.WithAlpha(0))}), sb, t))
	default:
		return foundation.Some(painting.BorderSideLerp(sa, painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(0.0), Color: foundation.Some(sa.Color.WithAlpha(0))}), t))
	}
}

// inputDecorationEqual reports whether the two optional decoration themes are
// equal.
func inputDecorationEqual(a, b *InputDecorationThemeData) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

// datePickerOverlay makes the overlay color of the days, the years and the range:
// the selected state has the overlay of the color on primary, the others the
// overlay of the color on surface.
func datePickerOverlay(onSelected, onOther ui.Color, pressed, hovered, focused float64) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		base := onOther
		if states.Has(widgets.WidgetStateSelected) {
			base = onSelected
		}
		switch {
		case states.Has(widgets.WidgetStatePressed):
			return foundation.Some(base.WithOpacity(pressed))
		case states.Has(widgets.WidgetStateHovered):
			return foundation.Some(base.WithOpacity(hovered))
		case states.Has(widgets.WidgetStateFocused):
			return foundation.Some(base.WithOpacity(focused))
		}
		return foundation.None[ui.Color]()
	})
}

// datePickerSelectedColor makes the color property that is a color for the
// selected state and nothing otherwise.
func datePickerSelectedColor(selected ui.Color) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		if states.Has(widgets.WidgetStateSelected) {
			return foundation.Some(selected)
		}
		return foundation.None[ui.Color]()
	})
}

// datePickerForeground makes the color property of a foreground: the selected
// color, the disabled color, or the normal color.
func datePickerForeground(selected, disabled, normal ui.Color) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		switch {
		case states.Has(widgets.WidgetStateSelected):
			return foundation.Some(selected)
		case states.Has(widgets.WidgetStateDisabled):
			return foundation.Some(disabled)
		}
		return foundation.Some(normal)
	})
}

// DatePickerThemeDefaults returns the defaults of the date picker theme of the
// theme of the context: the Material 3 ones, or the Material 2 ones.
//
// Flutter: material/date_picker_theme.dart DatePickerTheme.defaults
func DatePickerThemeDefaults(bc widgets.BuildContext) DatePickerThemeData {
	theme := ThemeOf(bc)
	if theme.UseMaterial3 {
		return datePickerDefaultsM3(theme)
	}
	return datePickerDefaultsM2(theme)
}

// datePickerDefaultsM2 are the Material 2 defaults of the date picker.
//
// Flutter: material/date_picker_theme.dart _DatePickerDefaultsM2
func datePickerDefaultsM2(theme *ThemeData) DatePickerThemeData {
	colors := theme.ColorScheme
	textTheme := theme.TextTheme
	isDark := colors.Brightness() == ui.BrightnessDark
	subHeader := colors.OnSurface().WithOpacity(0.60)
	headerBackground, headerForeground := colors.Primary(), colors.OnPrimary()
	if isDark {
		headerBackground, headerForeground = colors.Surface(), colors.OnSurface()
	}
	dayBackground := datePickerSelectedColor(colors.Primary())
	overlay := datePickerOverlay(colors.OnPrimary(), colors.OnSurfaceVariant(), 0.38, 0.08, 0.12)
	// The pressed overlay of an unselected day is 0.12 in Material 2.
	dayOverlay := widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		if !states.Has(widgets.WidgetStateSelected) && states.Has(widgets.WidgetStatePressed) {
			return foundation.Some(colors.OnSurfaceVariant().WithOpacity(0.12))
		}
		return overlay.Resolve(states)
	})
	return DatePickerThemeData{
		Elevation:                        foundation.Some(24.0),
		Shape:                            painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(4))}),
		DayShape:                         widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](painting.NewCircleBorder()),
		YearShape:                        widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](painting.NewStadiumBorder()),
		RangePickerElevation:             foundation.Some(0.0),
		RangePickerShape:                 painting.NewRoundedRectangleBorder(),
		HeaderBackgroundColor:            foundation.Some(headerBackground),
		SubHeaderForegroundColor:         foundation.Some(subHeader),
		ToggleButtonTextStyle:            textTheme.TitleSmall.Apply(painting.TextStyleApply{Color: foundation.Some(subHeader)}),
		CancelButtonStyle:                TextButtonStyleFrom(ButtonStyleFromOpts{}),
		ConfirmButtonStyle:               TextButtonStyleFrom(ButtonStyleFromOpts{}),
		HeaderForegroundColor:            foundation.Some(headerForeground),
		HeaderHeadlineStyle:              textTheme.HeadlineSmall,
		HeaderHelpStyle:                  textTheme.LabelSmall,
		WeekdayStyle:                     textTheme.BodySmall.Apply(painting.TextStyleApply{Color: foundation.Some(colors.OnSurface().WithOpacity(0.60))}),
		DayStyle:                         textTheme.BodySmall,
		DayForegroundColor:               datePickerForeground(colors.OnPrimary(), colors.OnSurface().WithOpacity(0.38), colors.OnSurface()),
		DayBackgroundColor:               dayBackground,
		DayOverlayColor:                  dayOverlay,
		TodayForegroundColor:             datePickerForeground(colors.OnPrimary(), colors.OnSurface().WithOpacity(0.38), colors.Primary()),
		TodayBackgroundColor:             dayBackground,
		TodayBorder:                      foundation.Some(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.Primary())})),
		YearStyle:                        textTheme.BodyLarge,
		RangePickerBackgroundColor:       foundation.Some(colors.Surface()),
		RangePickerShadowColor:           foundation.Some(Colors.Transparent),
		RangePickerSurfaceTintColor:      foundation.Some(Colors.Transparent),
		RangePickerHeaderBackgroundColor: foundation.Some(headerBackground),
		RangePickerHeaderForegroundColor: foundation.Some(headerForeground),
		RangePickerHeaderHeadlineStyle:   textTheme.HeadlineSmall,
		RangePickerHeaderHelpStyle:       textTheme.LabelSmall,
		RangeSelectionBackgroundColor:    foundation.Some(colors.Primary().WithOpacity(0.12)),
		RangeSelectionOverlayColor:       dayOverlay,
	}
}

// datePickerDefaultsM3 are the Material 3 defaults of the date picker (generated
// from the Material token database in Flutter).
//
// Flutter: material/date_picker_theme.dart _DatePickerDefaultsM3
func datePickerDefaultsM3(theme *ThemeData) DatePickerThemeData {
	colors := theme.ColorScheme
	textTheme := theme.TextTheme
	subHeader := colors.OnSurface().WithOpacity(0.60)
	dayBackground := datePickerSelectedColor(colors.Primary())
	dayOverlay := datePickerOverlay(colors.OnPrimary(), colors.OnSurfaceVariant(), 0.1, 0.08, 0.1)
	return DatePickerThemeData{
		Elevation:                        foundation.Some(6.0),
		Shape:                            painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(28))}),
		DayShape:                         widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](painting.NewCircleBorder()),
		YearShape:                        widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](painting.NewStadiumBorder()),
		RangePickerElevation:             foundation.Some(0.0),
		RangePickerShape:                 painting.NewRoundedRectangleBorder(),
		BackgroundColor:                  foundation.Some(colors.SurfaceContainerHigh()),
		SubHeaderForegroundColor:         foundation.Some(subHeader),
		ToggleButtonTextStyle:            textTheme.TitleSmall.Apply(painting.TextStyleApply{Color: foundation.Some(subHeader)}),
		CancelButtonStyle:                TextButtonStyleFrom(ButtonStyleFromOpts{}),
		ConfirmButtonStyle:               TextButtonStyleFrom(ButtonStyleFromOpts{}),
		ShadowColor:                      foundation.Some(Colors.Transparent),
		SurfaceTintColor:                 foundation.Some(Colors.Transparent),
		HeaderBackgroundColor:            foundation.Some(Colors.Transparent),
		HeaderForegroundColor:            foundation.Some(colors.OnSurfaceVariant()),
		HeaderHeadlineStyle:              textTheme.HeadlineLarge,
		HeaderHelpStyle:                  textTheme.LabelLarge,
		WeekdayStyle:                     textTheme.BodyLarge.Apply(painting.TextStyleApply{Color: foundation.Some(colors.OnSurface())}),
		DayStyle:                         textTheme.BodyLarge,
		DayForegroundColor:               datePickerForeground(colors.OnPrimary(), colors.OnSurface().WithOpacity(0.38), colors.OnSurface()),
		DayBackgroundColor:               dayBackground,
		DayOverlayColor:                  dayOverlay,
		TodayForegroundColor:             datePickerForeground(colors.OnPrimary(), colors.Primary().WithOpacity(0.38), colors.Primary()),
		TodayBackgroundColor:             dayBackground,
		TodayBorder:                      foundation.Some(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.Primary())})),
		YearStyle:                        textTheme.BodyLarge,
		YearForegroundColor:              datePickerForeground(colors.OnPrimary(), colors.OnSurfaceVariant().WithOpacity(0.38), colors.OnSurfaceVariant()),
		YearBackgroundColor:              dayBackground,
		YearOverlayColor:                 dayOverlay,
		RangePickerShadowColor:           foundation.Some(Colors.Transparent),
		RangePickerSurfaceTintColor:      foundation.Some(Colors.Transparent),
		RangeSelectionBackgroundColor:    foundation.Some(colors.SecondaryContainer()),
		RangeSelectionOverlayColor:       datePickerOverlay(colors.OnPrimaryContainer(), colors.OnPrimaryContainer(), 0.1, 0.08, 0.1),
		RangePickerHeaderBackgroundColor: foundation.Some(Colors.Transparent),
		RangePickerHeaderForegroundColor: foundation.Some(colors.OnSurfaceVariant()),
		RangePickerHeaderHeadlineStyle:   textTheme.TitleLarge,
		RangePickerHeaderHelpStyle:       textTheme.TitleSmall,
	}
}

// lerpTimePickerBorderSide interpolates the border of the AM/PM toggle: the one
// that exists when only one does.
//
// Flutter: material/time_picker_theme.dart TimePickerThemeData.lerp
func lerpTimePickerBorderSide(a, b foundation.Opt[painting.BorderSide], t float64) foundation.Opt[painting.BorderSide] {
	sa, okA := a.Get()
	sb, okB := b.Get()
	switch {
	case !okA && !okB:
		return foundation.None[painting.BorderSide]()
	case !okA:
		return b
	case !okB:
		return a
	}
	return foundation.Some(painting.BorderSideLerp(sa, sb, t))
}
