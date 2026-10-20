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

// ElevatedButtonOpts are the optional parameters of NewElevatedButton.
//
// Flutter: material/elevated_button.dart ElevatedButton.new
type ElevatedButtonOpts = ButtonStyleButtonOpts

// ElevatedButtonIconOpts are the optional parameters of NewElevatedButtonIcon.
//
// Flutter: material/elevated_button.dart ElevatedButton.icon
type ElevatedButtonIconOpts = TextButtonIconOpts

// ElevatedButton is a Material Design elevated button: a filled button whose
// Material elevates when pressed. It gives dimension to mostly flat layouts.
//
// Flutter: material/elevated_button.dart ElevatedButton
type ElevatedButton struct {
	ButtonStyleButton
	addPadding bool
}

// NewElevatedButton creates an elevated button. A nil onPressed (without
// OnLongPress) disables the button.
//
// Flutter: material/elevated_button.dart ElevatedButton.new
func NewElevatedButton(onPressed func(), child widgets.Widget, opts ...ElevatedButtonOpts) *ElevatedButton {
	return &ElevatedButton{ButtonStyleButton: NewButtonStyleButtonBase(onPressed, child, firstOpts(opts))}
}

// NewElevatedButtonIcon creates an elevated button with an icon and a label. The
// icon may be nil, and then the button has only the label.
//
// Flutter: material/elevated_button.dart ElevatedButton.icon
func NewElevatedButtonIcon(onPressed func(), label widgets.Widget, opts ...ElevatedButtonIconOpts) *ElevatedButton {
	o := firstOpts(opts)
	child := label
	if o.Icon != nil {
		child = &buttonWithIconChild{StatelessWidget: widgets.NewStatelessWidgetBase(), label: label, icon: o.Icon, buttonStyle: o.Style,
			iconAlignment: o.IconAlignment, themeStyle: func(bc widgets.BuildContext) *ButtonStyle { return ElevatedButtonThemeOf(bc).Style }, gap1: 8, gap2: 4}
	}
	return &ElevatedButton{ButtonStyleButton: NewButtonStyleButtonBase(onPressed, child, o.ButtonStyleButtonOpts), addPadding: o.Icon != nil}
}

// ElevatedButtonStyleFrom is a static convenience that makes a ButtonStyle from
// simple values. The elevation is the one of the button at rest; the disabled,
// pressed, hovered and focused elevations come from it.
//
// Flutter: material/elevated_button.dart ElevatedButton.styleFrom
func ElevatedButtonStyleFrom(o ButtonStyleFromOpts) *ButtonStyle {
	return buttonStyleFrom(o, styleFromElevated)
}

// DefaultStyleOf is the default style of the button for the theme: Material 3 or
// Material 2.
//
// Flutter: material/elevated_button.dart ElevatedButton.defaultStyleOf
func (w *ElevatedButton) DefaultStyleOf(bc widgets.BuildContext) *ButtonStyle {
	theme := ThemeOf(bc)
	colorScheme := theme.ColorScheme
	var buttonStyle *ButtonStyle
	if theme.UseMaterial3 {
		buttonStyle = elevatedButtonDefaultsM3(bc)
	} else {
		buttonStyle = ElevatedButtonStyleFrom(ButtonStyleFromOpts{
			BackgroundColor:         foundation.Some(colorScheme.Primary()),
			ForegroundColor:         foundation.Some(colorScheme.OnPrimary()),
			DisabledBackgroundColor: foundation.Some(colorScheme.OnSurface().WithOpacity(0.12)),
			DisabledForegroundColor: foundation.Some(colorScheme.OnSurface().WithOpacity(0.38)),
			ShadowColor:             foundation.Some(theme.ShadowColor),
			Elevation:               foundation.Some(2.0),
			TextStyle:               theme.TextTheme.LabelLarge,
			Padding:                 elevatedButtonScaledPadding(bc),
			MinimumSize:             foundation.Some(ui.Size{Width: 64, Height: 36}),
			MaximumSize:             foundation.Some(ui.SizeInfinite),
			Shape:                   painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(4))}),
			EnabledMouseCursor:      services.SystemMouseCursors.Basic,
			DisabledMouseCursor:     services.SystemMouseCursors.Basic,
			VisualDensity:           foundation.Some(theme.VisualDensity),
			TapTargetSize:           theme.MaterialTapTargetSize,
			AnimationDuration:       foundation.Some(ThemeChangeDuration),
			EnableFeedback:          foundation.Some(true),
			Alignment:               painting.AlignmentCenter,
			SplashFactory:           InkRippleFactory,
		})
	}
	if w.addPadding {
		effectiveTextScale := buttonEffectiveTextScale(bc, buttonStyle)
		var scaled painting.EdgeInsetsGeometry
		if theme.UseMaterial3 {
			scaled = ButtonStyleButtonScaledPadding(
				painting.NewEdgeInsetsDirectionalFromSTEB(16, 0, 24, 0),
				painting.NewEdgeInsetsDirectionalFromSTEB(8, 0, 12, 0),
				painting.NewEdgeInsetsDirectionalFromSTEB(4, 0, 6, 0), effectiveTextScale)
		} else {
			scaled = ButtonStyleButtonScaledPadding(
				painting.NewEdgeInsetsDirectionalFromSTEB(12, 0, 16, 0),
				painting.NewEdgeInsetsSymmetric(0, 8),
				painting.NewEdgeInsetsDirectionalFromSTEB(8, 0, 4, 0), effectiveTextScale)
		}
		return buttonStyle.CopyWith(ButtonStyleOpts{Padding: widgets.NewWidgetStatePropertyAll(scaled)})
	}
	return buttonStyle
}

// ThemeStyleOf is the style of the ElevatedButtonTheme of the context.
//
// Flutter: material/elevated_button.dart ElevatedButton.themeStyleOf
func (w *ElevatedButton) ThemeStyleOf(bc widgets.BuildContext) *ButtonStyle {
	return ElevatedButtonThemeOf(bc).Style
}

// buttonEffectiveTextScale is the scale of the text of the default style of a
// button, relative to the 14 pixels of the label.
func buttonEffectiveTextScale(bc widgets.BuildContext, style *ButtonStyle) float64 {
	defaultFontSize := 14.0
	if style.TextStyle != nil {
		if ts := style.TextStyle.Resolve(0); ts != nil {
			defaultFontSize = ts.FontSize.OrElse(14.0)
		}
	}
	return widgets.MediaQueryTextScalerOf(bc).Scale(defaultFontSize) / 14.0
}

// Flutter: material/elevated_button.dart _scaledPadding
func elevatedButtonScaledPadding(bc widgets.BuildContext) painting.EdgeInsetsGeometry {
	padding1x := 16.0
	if ThemeOf(bc).UseMaterial3 {
		padding1x = 24.0
	}
	return buttonScaledPadding(bc, painting.NewEdgeInsetsSymmetric(0, padding1x), painting.NewEdgeInsetsSymmetric(0, padding1x),
		painting.NewEdgeInsetsSymmetric(0, padding1x/2), painting.NewEdgeInsetsSymmetric(0, padding1x/2/2))
}

// elevatedButtonDefaultsM3 is the default style of an elevated button in
// Material 3.
//
// Flutter: material/elevated_button.dart _ElevatedButtonDefaultsM3
func elevatedButtonDefaultsM3(bc widgets.BuildContext) *ButtonStyle {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	return &ButtonStyle{
		AnimationDuration: foundation.Some(ThemeChangeDuration),
		EnableFeedback:    foundation.Some(true),
		Alignment:         painting.AlignmentCenter,
		TextStyle:         widgets.NewWidgetStatePropertyAll(theme.TextTheme.LabelLarge),
		BackgroundColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if states.Has(widgets.WidgetStateDisabled) {
				return foundation.Some(colors.OnSurface().WithOpacity(0.12))
			}
			return foundation.Some(colors.SurfaceContainerLow())
		}),
		ForegroundColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if states.Has(widgets.WidgetStateDisabled) {
				return foundation.Some(colors.OnSurface().WithOpacity(0.38))
			}
			return foundation.Some(colors.Primary())
		}),
		OverlayColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			switch {
			case states.Has(widgets.WidgetStatePressed):
				return foundation.Some(colors.Primary().WithOpacity(0.1))
			case states.Has(widgets.WidgetStateHovered):
				return foundation.Some(colors.Primary().WithOpacity(0.08))
			case states.Has(widgets.WidgetStateFocused):
				return foundation.Some(colors.Primary().WithOpacity(0.1))
			}
			return foundation.None[ui.Color]()
		}),
		ShadowColor:      allOpt(colors.Shadow()),
		SurfaceTintColor: allOpt(Colors.Transparent),
		Elevation: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[float64] {
			switch {
			case states.Has(widgets.WidgetStateDisabled):
				return foundation.Some(0.0)
			case states.Has(widgets.WidgetStatePressed):
				return foundation.Some(1.0)
			case states.Has(widgets.WidgetStateHovered):
				return foundation.Some(3.0)
			}
			return foundation.Some(1.0)
		}),
		Padding:     widgets.NewWidgetStatePropertyAll(elevatedButtonScaledPadding(bc)),
		MinimumSize: allOpt(ui.Size{Width: 64, Height: 40}),
		IconSize:    allOpt(18.0),
		IconColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if states.Has(widgets.WidgetStateDisabled) {
				return foundation.Some(colors.OnSurface().WithOpacity(0.38))
			}
			return foundation.Some(colors.Primary())
		}),
		MaximumSize:   allOpt(ui.SizeInfinite),
		Shape:         widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](painting.NewStadiumBorder()),
		MouseCursor:   widgets.WidgetStateMouseCursorAdaptiveClickable,
		VisualDensity: foundation.Some(theme.VisualDensity),
		TapTargetSize: theme.MaterialTapTargetSize,
		SplashFactory: theme.SplashFactory,
	}
}
