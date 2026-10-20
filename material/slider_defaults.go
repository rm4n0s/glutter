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

// sliderDefaults are the default values of a Slider or a RangeSlider: the
// defaults of Material 2, of Material 3 as of 2023, or of Material 3 as of 2024.
//
// Flutter: material/slider.dart _SliderDefaultsM2, _SliderDefaultsM3Year2023, _SliderDefaultsM3
func sliderDefaults(bc widgets.BuildContext, material3, year2023 bool) SliderThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	some := foundation.Some[ui.Color]
	if !material3 {
		sliderTheme := SliderThemeOf(bc)
		valueIndicatorColor := colors.Primary()
		if _, ok := sliderTheme.ValueIndicatorShape.(RoundedRectSliderValueIndicatorShape); ok {
			valueIndicatorColor = colors.InverseSurface()
		}
		return SliderThemeData{
			TrackHeight:                       foundation.Some(4.0),
			ActiveTrackColor:                  some(colors.Primary()),
			InactiveTrackColor:                some(colors.Primary().WithOpacity(0.24)),
			SecondaryActiveTrackColor:         some(colors.Primary().WithOpacity(0.54)),
			DisabledActiveTrackColor:          some(colors.OnSurface().WithOpacity(0.32)),
			DisabledInactiveTrackColor:        some(colors.OnSurface().WithOpacity(0.12)),
			DisabledSecondaryActiveTrackColor: some(colors.OnSurface().WithOpacity(0.12)),
			ActiveTickMarkColor:               some(colors.OnPrimary().WithOpacity(0.54)),
			InactiveTickMarkColor:             some(colors.Primary().WithOpacity(0.54)),
			DisabledActiveTickMarkColor:       some(colors.OnPrimary().WithOpacity(0.12)),
			DisabledInactiveTickMarkColor:     some(colors.OnSurface().WithOpacity(0.12)),
			ThumbColor:                        some(colors.Primary()),
			DisabledThumbColor:                some(ui.ColorAlphaBlend(colors.OnSurface().WithOpacity(.38), colors.Surface())),
			OverlayColor:                      some(colors.Primary().WithOpacity(0.12)),
			ValueIndicatorTextStyle:           theme.TextTheme.BodyLarge.CopyWith(painting.TextStyleCopy{Color: some(colors.OnPrimary())}),
			ValueIndicatorColor:               some(valueIndicatorColor),
			ValueIndicatorShape:               RectangularSliderValueIndicatorShape{},
			ThumbShape:                        RoundSliderThumbShape{},
			TrackShape:                        RoundedRectSliderTrackShape{},
			OverlayShape:                      RoundSliderOverlayShape{},
			TickMarkShape:                     RoundSliderTickMarkShape{},
		}
	}
	overlay := widgets.WidgetStatePropertyResolveWith(func(s widgets.WidgetStates) foundation.Opt[ui.Color] {
		switch {
		case s.Has(widgets.WidgetStateDragged):
			return some(colors.Primary().WithOpacity(0.1))
		case s.Has(widgets.WidgetStateHovered):
			return some(colors.Primary().WithOpacity(0.08))
		case s.Has(widgets.WidgetStateFocused):
			return some(colors.Primary().WithOpacity(0.1))
		}
		return some(Colors.Transparent)
	})
	if year2023 {
		return SliderThemeData{
			TrackHeight:                       foundation.Some(4.0),
			ActiveTrackColor:                  some(colors.Primary()),
			InactiveTrackColor:                some(colors.SurfaceContainerHighest()),
			SecondaryActiveTrackColor:         some(colors.Primary().WithOpacity(0.54)),
			DisabledActiveTrackColor:          some(colors.OnSurface().WithOpacity(0.38)),
			DisabledInactiveTrackColor:        some(colors.OnSurface().WithOpacity(0.12)),
			DisabledSecondaryActiveTrackColor: some(colors.OnSurface().WithOpacity(0.12)),
			ActiveTickMarkColor:               some(colors.OnPrimary().WithOpacity(0.38)),
			InactiveTickMarkColor:             some(colors.OnSurfaceVariant().WithOpacity(0.38)),
			DisabledActiveTickMarkColor:       some(colors.OnSurface().WithOpacity(0.38)),
			DisabledInactiveTickMarkColor:     some(colors.OnSurface().WithOpacity(0.38)),
			ThumbColor:                        some(colors.Primary()),
			DisabledThumbColor:                some(ui.ColorAlphaBlend(colors.OnSurface().WithOpacity(0.38), colors.Surface())),
			StateOverlayColor:                 overlay,
			ValueIndicatorTextStyle:           theme.TextTheme.LabelMedium.CopyWith(painting.TextStyleCopy{Color: some(colors.OnPrimary())}),
			ValueIndicatorColor:               some(colors.Primary()),
			ValueIndicatorShape:               DropSliderValueIndicatorShape{},
			ThumbShape:                        RoundSliderThumbShape{},
			TrackShape:                        RoundedRectSliderTrackShape{},
			OverlayShape:                      RoundSliderOverlayShape{},
			TickMarkShape:                     RoundSliderTickMarkShape{},
		}
	}
	return SliderThemeData{
		TrackHeight:                       foundation.Some(16.0),
		ActiveTrackColor:                  some(colors.Primary()),
		InactiveTrackColor:                some(colors.SecondaryContainer()),
		SecondaryActiveTrackColor:         some(colors.Primary().WithOpacity(0.54)),
		DisabledActiveTrackColor:          some(colors.OnSurface().WithOpacity(0.38)),
		DisabledInactiveTrackColor:        some(colors.OnSurface().WithOpacity(0.12)),
		DisabledSecondaryActiveTrackColor: some(colors.OnSurface().WithOpacity(0.38)),
		ActiveTickMarkColor:               some(colors.OnPrimary().WithOpacity(1.0)),
		InactiveTickMarkColor:             some(colors.OnSecondaryContainer().WithOpacity(1.0)),
		DisabledActiveTickMarkColor:       some(colors.OnInverseSurface()),
		DisabledInactiveTickMarkColor:     some(colors.OnSurface()),
		ThumbColor:                        some(colors.Primary()),
		DisabledThumbColor:                some(colors.OnSurface().WithOpacity(0.38)),
		StateOverlayColor:                 overlay,
		ValueIndicatorTextStyle:           theme.TextTheme.LabelLarge.CopyWith(painting.TextStyleCopy{Color: some(colors.OnInverseSurface())}),
		ValueIndicatorColor:               some(colors.InverseSurface()),
		ValueIndicatorShape:               RoundedRectSliderValueIndicatorShape{},
		ThumbShape:                        HandleThumbShape{},
		TrackShape:                        GappedSliderTrackShape{},
		OverlayShape:                      RoundSliderOverlayShape{},
		TickMarkShape:                     RoundSliderTickMarkShape{TickMarkRadius: foundation.Some(4.0 / 2)},
		ThumbSize: widgets.WidgetStatePropertyResolveWith(func(s widgets.WidgetStates) foundation.Opt[ui.Size] {
			switch {
			case s.Has(widgets.WidgetStateDisabled), s.Has(widgets.WidgetStateHovered):
				return foundation.Some(ui.Size{Width: 4.0, Height: 44.0})
			case s.Has(widgets.WidgetStateFocused), s.Has(widgets.WidgetStatePressed):
				return foundation.Some(ui.Size{Width: 2.0, Height: 44.0})
			}
			return foundation.Some(ui.Size{Width: 4.0, Height: 44.0})
		}),
		TrackGap: foundation.Some(6.0),
	}
}

// rangeSliderDefaults are the default values of a RangeSlider: the ones of Material
// 3 as of 2024, or the ones of Material 2 for every other look.
//
// Flutter: material/range_slider.dart _RangeSliderDefaultsM2, _RangeSliderDefaultsM3
func rangeSliderDefaults(bc widgets.BuildContext, material3, year2023 bool) SliderThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	some := foundation.Some[ui.Color]
	if material3 && !year2023 {
		return SliderThemeData{
			TrackHeight:                   foundation.Some(16.0),
			ActiveTrackColor:              some(colors.Primary()),
			InactiveTrackColor:            some(colors.SecondaryContainer()),
			DisabledActiveTrackColor:      some(colors.OnSurface().WithOpacity(0.38)),
			DisabledInactiveTrackColor:    some(colors.OnSurface().WithOpacity(0.12)),
			ActiveTickMarkColor:           some(colors.OnPrimary().WithOpacity(1.0)),
			InactiveTickMarkColor:         some(colors.OnSecondaryContainer().WithOpacity(1.0)),
			DisabledActiveTickMarkColor:   some(colors.OnInverseSurface()),
			DisabledInactiveTickMarkColor: some(colors.OnSurface()),
			ThumbColor:                    some(colors.Primary()),
			OverlappingShapeStrokeColor:   some(colors.Surface()),
			DisabledThumbColor:            some(colors.OnSurface().WithOpacity(0.38)),
			OverlayColor:                  some(colors.Primary().WithOpacity(0.12)),
			ValueIndicatorTextStyle:       theme.TextTheme.LabelLarge.CopyWith(painting.TextStyleCopy{Color: some(colors.OnInverseSurface())}),
			ValueIndicatorColor:           some(colors.InverseSurface()),
			RangeTrackShape:               GappedRangeSliderTrackShape{},
			RangeTickMarkShape:            RoundRangeSliderTickMarkShape{TickMarkRadius: foundation.Some(4.0 / 2)},
			RangeThumbShape:               HandleRangeSliderThumbShape{},
			OverlayShape:                  RoundSliderOverlayShape{},
			RangeValueIndicatorShape:      RoundedRectRangeSliderValueIndicatorShape{},
			ShowValueIndicator:            ShowValueIndicatorOnlyForDiscrete,
			MinThumbSeparation:            foundation.Some(0.0),
			ThumbSize: widgets.WidgetStatePropertyResolveWith(func(s widgets.WidgetStates) foundation.Opt[ui.Size] {
				switch {
				case s.Has(widgets.WidgetStateDisabled), s.Has(widgets.WidgetStateHovered):
					return foundation.Some(ui.Size{Width: 4.0, Height: 44.0})
				case s.Has(widgets.WidgetStateFocused), s.Has(widgets.WidgetStatePressed):
					return foundation.Some(ui.Size{Width: 2.0, Height: 44.0})
				}
				return foundation.Some(ui.Size{Width: 4.0, Height: 44.0})
			}),
			TrackGap: foundation.Some(6.0),
		}
	}
	return SliderThemeData{
		TrackHeight:                   foundation.Some(4.0),
		ActiveTrackColor:              some(colors.Primary()),
		InactiveTrackColor:            some(colors.Primary().WithOpacity(0.24)),
		DisabledActiveTrackColor:      some(colors.OnSurface().WithOpacity(0.32)),
		DisabledInactiveTrackColor:    some(colors.OnSurface().WithOpacity(0.12)),
		ActiveTickMarkColor:           some(colors.OnPrimary().WithOpacity(0.54)),
		InactiveTickMarkColor:         some(colors.Primary().WithOpacity(0.54)),
		DisabledActiveTickMarkColor:   some(colors.OnPrimary().WithOpacity(0.12)),
		DisabledInactiveTickMarkColor: some(colors.OnSurface().WithOpacity(0.12)),
		ThumbColor:                    some(colors.Primary()),
		OverlappingShapeStrokeColor:   some(colors.Surface()),
		DisabledThumbColor:            some(ui.ColorAlphaBlend(colors.OnSurface().WithOpacity(.38), colors.Surface())),
		OverlayColor:                  some(colors.Primary().WithOpacity(0.12)),
		ValueIndicatorTextStyle:       theme.TextTheme.BodyLarge.CopyWith(painting.TextStyleCopy{Color: some(colors.OnPrimary())}),
		ValueIndicatorColor:           some(colors.Primary()),
		RangeTrackShape:               RoundedRectRangeSliderTrackShape{},
		RangeTickMarkShape:            RoundRangeSliderTickMarkShape{},
		RangeThumbShape:               RoundRangeSliderThumbShape{},
		OverlayShape:                  RoundSliderOverlayShape{},
		RangeValueIndicatorShape:      RectangularRangeSliderValueIndicatorShape{},
		ShowValueIndicator:            ShowValueIndicatorOnlyForDiscrete,
		MinThumbSeparation:            foundation.Some(8.0),
	}
}
