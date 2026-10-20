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

// WidgetStateInputBorder is an InputBorder that is also a property that resolves
// to an InputBorder for a set of states. As a border, it is the default
// OutlineInputBorder.
//
// Flutter: material/material_state.dart WidgetStateInputBorder
type WidgetStateInputBorder interface {
	InputBorder
	widgets.WidgetStateProperty[InputBorder]
}

type widgetStateInputBorder struct {
	OutlineInputBorder
	resolve widgets.WidgetPropertyResolver[InputBorder]
}

func (b *widgetStateInputBorder) Resolve(states widgets.WidgetStates) InputBorder {
	return b.resolve(states)
}

// WidgetStateInputBorderResolveWith is an input border that resolves with the
// callback.
//
// Flutter: material/material_state.dart WidgetStateInputBorder.resolveWith
func WidgetStateInputBorderResolveWith(callback widgets.WidgetPropertyResolver[InputBorder]) WidgetStateInputBorder {
	return &widgetStateInputBorder{OutlineInputBorder: NewOutlineInputBorder(), resolve: callback}
}

// WidgetStateInputBorderFromMap is an input border that resolves with a map.
//
// Flutter: material/material_state.dart WidgetStateInputBorder.fromMap
func WidgetStateInputBorderFromMap(m widgets.WidgetStateMap[InputBorder]) WidgetStateInputBorder {
	mapper := widgets.NewWidgetStateMapper(m)
	return &widgetStateInputBorder{OutlineInputBorder: NewOutlineInputBorder(), resolve: mapper.Resolve}
}

// inputDecoratorDefaultsM2 is the default theme of the decoration of a field in
// Material 2.
//
// Flutter: material/input_decorator.dart _InputDecoratorDefaultsM2
func inputDecoratorDefaultsM2(bc widgets.BuildContext) InputDecorationThemeData {
	// The theme is read when a property is resolved, as Flutter's getters do.
	themeOf := func() *ThemeData { return ThemeOf(bc) }
	textStyle := func(c ui.Color) *painting.TextStyle {
		return painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(c)})
	}
	hintOrDisabled := func(states widgets.WidgetStates) *painting.TextStyle {
		if states.Has(widgets.WidgetStateDisabled) {
			return textStyle(themeOf().DisabledColor)
		}
		return textStyle(themeOf().HintColor)
	}
	iconColor := func(states widgets.WidgetStates, withError bool) ui.Color {
		theme := themeOf()
		if states.Has(widgets.WidgetStateDisabled) && !states.Has(widgets.WidgetStateFocused) {
			return theme.DisabledColor
		}
		if withError && states.Has(widgets.WidgetStateError) {
			return theme.ColorScheme.Error()
		}
		if states.Has(widgets.WidgetStateFocused) {
			return theme.ColorScheme.Primary()
		}
		if ThemeBrightnessOf(bc) == ui.BrightnessDark {
			return Colors.White70
		}
		return Colors.Black45
	}
	return InputDecorationThemeData{
		HintStyle:  widgets.WidgetStatePropertyResolveWith(hintOrDisabled),
		LabelStyle: widgets.WidgetStatePropertyResolveWith(hintOrDisabled),
		FloatingLabelStyle: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) *painting.TextStyle {
			theme := themeOf()
			switch {
			case states.Has(widgets.WidgetStateDisabled):
				return textStyle(theme.DisabledColor)
			case states.Has(widgets.WidgetStateError):
				return textStyle(theme.ColorScheme.Error())
			case states.Has(widgets.WidgetStateFocused):
				return textStyle(theme.ColorScheme.Primary())
			}
			return textStyle(theme.HintColor)
		}),
		HelperStyle: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) *painting.TextStyle {
			theme := themeOf()
			if states.Has(widgets.WidgetStateDisabled) {
				return theme.TextTheme.BodySmall.CopyWith(painting.TextStyleCopy{Color: foundation.Some(Colors.Transparent)})
			}
			return theme.TextTheme.BodySmall.CopyWith(painting.TextStyleCopy{Color: foundation.Some(theme.HintColor)})
		}),
		ErrorStyle: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) *painting.TextStyle {
			theme := themeOf()
			if states.Has(widgets.WidgetStateDisabled) {
				return theme.TextTheme.BodySmall.CopyWith(painting.TextStyleCopy{Color: foundation.Some(Colors.Transparent)})
			}
			return theme.TextTheme.BodySmall.CopyWith(painting.TextStyleCopy{Color: foundation.Some(theme.ColorScheme.Error())})
		}),
		FillColor: widgets.WidgetStateColorResolveWith(func(states widgets.WidgetStates) ui.Color {
			disabled := states.Has(widgets.WidgetStateDisabled)
			if ThemeBrightnessOf(bc) == ui.BrightnessDark {
				if disabled {
					return ui.NewColor(0x0DFFFFFF) // 5% white
				}
				return ui.NewColor(0x1AFFFFFF) // 10% white
			}
			if disabled {
				return ui.NewColor(0x05000000) // 2% black
			}
			return ui.NewColor(0x0A000000) // 4% black
		}),
		IconColor:       widgets.WidgetStateColorResolveWith(func(s widgets.WidgetStates) ui.Color { return iconColor(s, false) }),
		PrefixIconColor: widgets.WidgetStateColorResolveWith(func(s widgets.WidgetStates) ui.Color { return iconColor(s, false) }),
		SuffixIconColor: widgets.WidgetStateColorResolveWith(func(s widgets.WidgetStates) ui.Color { return iconColor(s, true) }),
	}
}

// inputDecoratorDefaultsM3 is the default theme of the decoration of a field in
// Material 3.
//
// Flutter: material/input_decorator.dart _InputDecoratorDefaultsM3
func inputDecoratorDefaultsM3(bc widgets.BuildContext) InputDecorationThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	textTheme := theme.TextTheme
	opacity := func(c ui.Color, o float64) ui.Color { return c.WithOpacity(o) }
	// For InputDecorator, the focused state takes precedence over the hovered
	// state. For instance, the focused state increases the border width (2dp) and
	// applies bright colors (primary or error) while the hovered state has the
	// same border as the non-focused state (1dp) and uses a color a little darker
	// than the non-focused state. On the desktop, it is also very common that a
	// text field is focused and hovered because users often rely on the mouse to
	// select. For the other widgets, the hovered state takes precedence over the
	// focused state, because it is mainly used to determine the overlay color.
	side := func(c ui.Color, width ...float64) foundation.Opt[painting.BorderSide] {
		o := painting.BorderSideOpts{Color: foundation.Some(c)}
		if len(width) > 0 {
			o.Width = foundation.Some(width[0])
		}
		return foundation.Some(painting.NewBorderSide(o))
	}
	borderSide := func(disabled, normal ui.Color) widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]] {
		return widgets.WidgetStateBorderSideResolveWith(func(states widgets.WidgetStates) foundation.Opt[painting.BorderSide] {
			switch {
			case states.Has(widgets.WidgetStateDisabled):
				return side(disabled)
			case states.Has(widgets.WidgetStateError):
				switch {
				case states.Has(widgets.WidgetStateFocused):
					return side(colors.Error(), 2.0)
				case states.Has(widgets.WidgetStateHovered):
					return side(colors.OnErrorContainer())
				}
				return side(colors.Error())
			case states.Has(widgets.WidgetStateFocused):
				return side(colors.Primary(), 2.0)
			case states.Has(widgets.WidgetStateHovered):
				return side(colors.OnSurface())
			}
			return side(normal)
		})
	}
	labelStyle := func(states widgets.WidgetStates) *painting.TextStyle {
		style := textTheme.BodyLarge
		if style == nil {
			style = painting.NewTextStyle()
		}
		color := func(c ui.Color) *painting.TextStyle {
			return style.CopyWith(painting.TextStyleCopy{Color: foundation.Some(c)})
		}
		switch {
		case states.Has(widgets.WidgetStateDisabled):
			return color(opacity(colors.OnSurface(), 0.38))
		case states.Has(widgets.WidgetStateError):
			switch {
			case states.Has(widgets.WidgetStateFocused):
				return color(colors.Error())
			case states.Has(widgets.WidgetStateHovered):
				return color(colors.OnErrorContainer())
			}
			return color(colors.Error())
		case states.Has(widgets.WidgetStateFocused):
			return color(colors.Primary())
		}
		return color(colors.OnSurfaceVariant())
	}
	smallStyle := func() *painting.TextStyle {
		if textTheme.BodySmall != nil {
			return textTheme.BodySmall
		}
		return painting.NewTextStyle()
	}
	return InputDecorationThemeData{
		HintStyle: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) *painting.TextStyle {
			if states.Has(widgets.WidgetStateDisabled) {
				return painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(opacity(colors.OnSurface(), 0.38))})
			}
			return painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(colors.OnSurfaceVariant())})
		}),
		FillColor: widgets.WidgetStateColorResolveWith(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateDisabled) {
				return opacity(colors.OnSurface(), 0.04)
			}
			return colors.SurfaceContainerHighest()
		}),
		ActiveIndicatorBorder: borderSide(opacity(colors.OnSurface(), 0.38), colors.OnSurfaceVariant()),
		OutlineBorder:         borderSide(opacity(colors.OnSurface(), 0.12), colors.Outline()),
		IconColor:             widgets.NewWidgetStatePropertyAll(colors.OnSurfaceVariant()),
		PrefixIconColor: widgets.WidgetStateColorResolveWith(func(states widgets.WidgetStates) ui.Color {
			if states.Has(widgets.WidgetStateDisabled) {
				return opacity(colors.OnSurface(), 0.38)
			}
			return colors.OnSurfaceVariant()
		}),
		SuffixIconColor: widgets.WidgetStateColorResolveWith(func(states widgets.WidgetStates) ui.Color {
			switch {
			case states.Has(widgets.WidgetStateDisabled):
				return opacity(colors.OnSurface(), 0.38)
			case states.Has(widgets.WidgetStateError):
				if states.Has(widgets.WidgetStateHovered) {
					return colors.OnErrorContainer()
				}
				return colors.Error()
			}
			return colors.OnSurfaceVariant()
		}),
		LabelStyle:         widgets.WidgetStatePropertyResolveWith(labelStyle),
		FloatingLabelStyle: widgets.WidgetStatePropertyResolveWith(labelStyle),
		HelperStyle: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) *painting.TextStyle {
			style := smallStyle()
			if states.Has(widgets.WidgetStateDisabled) {
				return style.CopyWith(painting.TextStyleCopy{Color: foundation.Some(opacity(colors.OnSurface(), 0.38))})
			}
			return style.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurfaceVariant())})
		}),
		ErrorStyle: widgets.WidgetStatePropertyResolveWith(func(widgets.WidgetStates) *painting.TextStyle {
			return smallStyle().CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.Error())})
		}),
	}
}
