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

// chipM3Shape is the shape that the Material 3 chips have: a rounded rectangle
// with a radius of 8.
func chipM3Shape() painting.OutlinedBorder {
	return painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(8.0))})
}

func chipM3Side(c ui.Color) foundation.Opt[painting.BorderSide] {
	return foundation.Some(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(c)}))
}

func chipM3IconTheme(c ui.Color) foundation.Opt[widgets.IconThemeData] {
	return foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(c), Size: foundation.Some(18.0)}))
}

func pickColor(cond bool, a, b ui.Color) ui.Color {
	if cond {
		return a
	}
	return b
}

// actionChipDefaultsM3 are the defaults of the Material 3 action chip.
//
// Flutter: material/action_chip.dart _ActionChipDefaultsM3
func actionChipDefaultsM3(bc widgets.BuildContext, isEnabled, elevated bool) ChipThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	elevation := 0.0
	if elevated && isEnabled {
		elevation = 1.0
	}
	labelStyle := theme.TextTheme.LabelLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurface())})
	d := ChipThemeData{
		Shape:            chipM3Shape(),
		ShowCheckmark:    foundation.Some(true),
		Elevation:        foundation.Some(elevation),
		PressElevation:   foundation.Some(1.0),
		LabelStyle:       labelStyle,
		SurfaceTintColor: foundation.Some(Colors.Transparent),
		IconTheme:        chipM3IconTheme(pickColor(isEnabled, colors.Primary(), colors.OnSurface())),
		Padding:          painting.NewEdgeInsetsAll(8.0),
		LabelPadding:     chipM3LabelPadding(bc, labelStyle),
	}
	if !elevated {
		d.ShadowColor = foundation.Some(Colors.Transparent)
		d.Side = chipM3Side(pickColor(isEnabled, colors.OutlineVariant(), colors.OnSurface().WithOpacity(0.12)))
		d.Color = widgets.WidgetStatePropertyResolveWith(func(widgets.WidgetStates) foundation.Opt[ui.Color] {
			return foundation.None[ui.Color]()
		})
		return d
	}
	d.ShadowColor = foundation.Some(colors.Shadow())
	d.Side = chipM3Side(Colors.Transparent)
	d.Color = widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		if states.Has(widgets.WidgetStateDisabled) {
			return foundation.Some(colors.OnSurface().WithOpacity(0.12))
		}
		return foundation.Some(colors.SurfaceContainerLow())
	})
	return d
}

// selectableChipDefaultsM3 are the defaults of the Material 3 choice chip and
// filter chip, which are the same.
//
// Flutter: material/choice_chip.dart _ChoiceChipDefaultsM3
// Flutter: material/filter_chip.dart _FilterChipDefaultsM3
func selectableChipDefaultsM3(bc widgets.BuildContext, isEnabled, isSelected, elevated bool) ChipThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	elevation := 0.0
	if elevated && isEnabled {
		elevation = 1.0
	}
	selectedOn := pickColor(isSelected, colors.OnSecondaryContainer(), colors.OnSurfaceVariant())
	labelColor := pickColor(isEnabled, selectedOn, colors.OnSurface())
	labelStyle := theme.TextTheme.LabelLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(labelColor)})
	selectedPrimary := pickColor(isSelected, colors.OnSecondaryContainer(), colors.Primary())
	d := ChipThemeData{
		Shape:            chipM3Shape(),
		ShowCheckmark:    foundation.Some(true),
		Elevation:        foundation.Some(elevation),
		PressElevation:   foundation.Some(1.0),
		LabelStyle:       labelStyle,
		SurfaceTintColor: foundation.Some(Colors.Transparent),
		CheckmarkColor:   foundation.Some(pickColor(isEnabled, selectedPrimary, colors.OnSurface())),
		DeleteIconColor:  foundation.Some(pickColor(isEnabled, selectedOn, colors.OnSurface())),
		IconTheme:        chipM3IconTheme(pickColor(isEnabled, selectedPrimary, colors.OnSurface())),
		Padding:          painting.NewEdgeInsetsAll(8.0),
		LabelPadding:     chipM3LabelPadding(bc, labelStyle),
		ShadowColor:      foundation.Some(Colors.Transparent),
	}
	if elevated {
		d.ShadowColor = foundation.Some(colors.Shadow())
	}
	if !elevated && !isSelected {
		d.Side = chipM3Side(pickColor(isEnabled, colors.OutlineVariant(), colors.OnSurface().WithOpacity(0.12)))
	} else {
		d.Side = chipM3Side(Colors.Transparent)
	}
	d.Color = widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		selected, disabled := states.Has(widgets.WidgetStateSelected), states.Has(widgets.WidgetStateDisabled)
		switch {
		case selected && disabled:
			return foundation.Some(colors.OnSurface().WithOpacity(0.12))
		case disabled:
			if elevated {
				return foundation.Some(colors.OnSurface().WithOpacity(0.12))
			}
			return foundation.None[ui.Color]()
		case selected:
			return foundation.Some(colors.SecondaryContainer())
		}
		if elevated {
			return foundation.Some(colors.SurfaceContainerLow())
		}
		return foundation.None[ui.Color]()
	})
	return d
}

// inputChipDefaultsM3 are the defaults of the Material 3 input chip.
//
// Flutter: material/input_chip.dart _InputChipDefaultsM3
func inputChipDefaultsM3(bc widgets.BuildContext, isEnabled, isSelected bool) ChipThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	selectedOn := pickColor(isSelected, colors.OnSecondaryContainer(), colors.OnSurfaceVariant())
	labelStyle := theme.TextTheme.LabelLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(pickColor(isEnabled, selectedOn, colors.OnSurface()))})
	d := ChipThemeData{
		Elevation:        foundation.Some(0.0),
		Shape:            chipM3Shape(),
		ShowCheckmark:    foundation.Some(true),
		LabelStyle:       labelStyle,
		ShadowColor:      foundation.Some(Colors.Transparent),
		SurfaceTintColor: foundation.Some(Colors.Transparent),
		CheckmarkColor:   foundation.Some(pickColor(isEnabled, pickColor(isSelected, colors.Primary(), colors.OnSurfaceVariant()), colors.OnSurface())),
		DeleteIconColor:  foundation.Some(pickColor(isEnabled, selectedOn, colors.OnSurface())),
		IconTheme:        chipM3IconTheme(pickColor(isEnabled, pickColor(isSelected, colors.Primary(), colors.OnSurfaceVariant()), colors.OnSurface())),
		Padding:          painting.NewEdgeInsetsAll(8.0),
		LabelPadding:     chipM3LabelPadding(bc, labelStyle),
	}
	if !isSelected {
		d.Side = chipM3Side(pickColor(isEnabled, colors.OutlineVariant(), colors.OnSurface().WithOpacity(0.12)))
	} else {
		d.Side = chipM3Side(Colors.Transparent)
	}
	d.Color = widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		selected, disabled := states.Has(widgets.WidgetStateSelected), states.Has(widgets.WidgetStateDisabled)
		switch {
		case selected && disabled:
			return foundation.Some(colors.OnSurface().WithOpacity(0.12))
		case disabled:
			return foundation.None[ui.Color]()
		case selected:
			return foundation.Some(colors.SecondaryContainer())
		}
		return foundation.None[ui.Color]()
	})
	return d
}
