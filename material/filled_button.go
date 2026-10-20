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

// FilledButtonOpts are the optional parameters of NewFilledButton and
// NewFilledButtonTonal.
//
// Flutter: material/filled_button.dart FilledButton.new
type FilledButtonOpts = ButtonStyleButtonOpts

// FilledButtonIconOpts are the optional parameters of NewFilledButtonIcon and
// NewFilledButtonTonalIcon.
//
// Flutter: material/filled_button.dart FilledButton.icon
type FilledButtonIconOpts = TextButtonIconOpts

// filledButtonVariant is the kind of a FilledButton.
//
// Flutter: material/filled_button.dart _FilledButtonVariant
type filledButtonVariant int

const (
	filledButtonFilled filledButtonVariant = iota
	filledButtonTonal
)

// FilledButton is a Material Design filled button: a button with the strong
// background of its color, for the important actions that end a flow, such as
// Save, Join now or Confirm. The tonal variant has a softer background.
//
// Flutter: material/filled_button.dart FilledButton
type FilledButton struct {
	ButtonStyleButton
	variant    filledButtonVariant
	addPadding bool
}

// NewFilledButton creates a filled button. A nil onPressed (without OnLongPress)
// disables the button.
//
// Flutter: material/filled_button.dart FilledButton.new
func NewFilledButton(onPressed func(), child widgets.Widget, opts ...FilledButtonOpts) *FilledButton {
	return &FilledButton{ButtonStyleButton: NewButtonStyleButtonBase(onPressed, child, firstOpts(opts)), variant: filledButtonFilled}
}

// NewFilledButtonIcon creates a filled button with an icon and a label. The icon
// may be nil, and then the button has only the label.
//
// Flutter: material/filled_button.dart FilledButton.icon
func NewFilledButtonIcon(onPressed func(), label widgets.Widget, opts ...FilledButtonIconOpts) *FilledButton {
	return newFilledButtonIcon(onPressed, label, filledButtonFilled, firstOpts(opts))
}

// NewFilledButtonTonal creates a filled tonal button: a button between the
// filled and the outlined buttons in emphasis, for the secondary actions.
//
// Flutter: material/filled_button.dart FilledButton.tonal
func NewFilledButtonTonal(onPressed func(), child widgets.Widget, opts ...FilledButtonOpts) *FilledButton {
	return &FilledButton{ButtonStyleButton: NewButtonStyleButtonBase(onPressed, child, firstOpts(opts)), variant: filledButtonTonal}
}

// NewFilledButtonTonalIcon creates a filled tonal button with an icon and a
// label. The icon may be nil, and then the button has only the label.
//
// Flutter: material/filled_button.dart FilledButton.tonalIcon
func NewFilledButtonTonalIcon(onPressed func(), label widgets.Widget, opts ...FilledButtonIconOpts) *FilledButton {
	return newFilledButtonIcon(onPressed, label, filledButtonTonal, firstOpts(opts))
}

func newFilledButtonIcon(onPressed func(), label widgets.Widget, variant filledButtonVariant, o FilledButtonIconOpts) *FilledButton {
	child := label
	if o.Icon != nil {
		child = &buttonWithIconChild{StatelessWidget: widgets.NewStatelessWidgetBase(), label: label, icon: o.Icon, buttonStyle: o.Style,
			iconAlignment: o.IconAlignment, themeStyle: func(bc widgets.BuildContext) *ButtonStyle { return FilledButtonThemeOf(bc).Style }, gap1: 8, gap2: 4}
	}
	return &FilledButton{ButtonStyleButton: NewButtonStyleButtonBase(onPressed, child, o.ButtonStyleButtonOpts), variant: variant, addPadding: o.Icon != nil}
}

// FilledButtonStyleFrom is a static convenience that makes a ButtonStyle from
// simple values.
//
// Flutter: material/filled_button.dart FilledButton.styleFrom
func FilledButtonStyleFrom(o ButtonStyleFromOpts) *ButtonStyle {
	return buttonStyleFrom(o, styleFromFilled)
}

// DefaultStyleOf is the default style of the button for the theme and the
// variant.
//
// Flutter: material/filled_button.dart FilledButton.defaultStyleOf
func (w *FilledButton) DefaultStyleOf(bc widgets.BuildContext) *ButtonStyle {
	var buttonStyle *ButtonStyle
	if w.variant == filledButtonTonal {
		buttonStyle = filledTonalButtonDefaultsM3(bc)
	} else {
		buttonStyle = filledButtonDefaultsM3(bc)
	}
	if w.addPadding {
		effectiveTextScale := buttonEffectiveTextScale(bc, buttonStyle)
		var scaled painting.EdgeInsetsGeometry
		if ThemeOf(bc).UseMaterial3 {
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

// ThemeStyleOf is the style of the FilledButtonTheme of the context.
//
// Flutter: material/filled_button.dart FilledButton.themeStyleOf
func (w *FilledButton) ThemeStyleOf(bc widgets.BuildContext) *ButtonStyle {
	return FilledButtonThemeOf(bc).Style
}

// Flutter: material/filled_button.dart _scaledPadding
func filledButtonScaledPadding(bc widgets.BuildContext) painting.EdgeInsetsGeometry {
	return elevatedButtonScaledPadding(bc)
}

// filledButtonDefaults is the default style of the filled buttons in Material 3:
// the background, the foreground (which is also the color of the overlay and of
// the icon) and the elevation when hovered of the variant.
func filledButtonDefaults(bc widgets.BuildContext, background, foreground ui.Color) *ButtonStyle {
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
			return foundation.Some(background)
		}),
		ForegroundColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if states.Has(widgets.WidgetStateDisabled) {
				return foundation.Some(colors.OnSurface().WithOpacity(0.38))
			}
			return foundation.Some(foreground)
		}),
		OverlayColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			switch {
			case states.Has(widgets.WidgetStatePressed):
				return foundation.Some(foreground.WithOpacity(0.1))
			case states.Has(widgets.WidgetStateHovered):
				return foundation.Some(foreground.WithOpacity(0.08))
			case states.Has(widgets.WidgetStateFocused):
				return foundation.Some(foreground.WithOpacity(0.1))
			}
			return foundation.None[ui.Color]()
		}),
		ShadowColor:      allOpt(colors.Shadow()),
		SurfaceTintColor: allOpt(Colors.Transparent),
		Elevation: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[float64] {
			if states.Has(widgets.WidgetStateHovered) && !states.Has(widgets.WidgetStateDisabled) && !states.Has(widgets.WidgetStatePressed) {
				return foundation.Some(1.0)
			}
			return foundation.Some(0.0)
		}),
		Padding:     widgets.NewWidgetStatePropertyAll(filledButtonScaledPadding(bc)),
		MinimumSize: allOpt(ui.Size{Width: 64, Height: 40}),
		IconSize:    allOpt(18.0),
		IconColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if states.Has(widgets.WidgetStateDisabled) {
				return foundation.Some(colors.OnSurface().WithOpacity(0.38))
			}
			return foundation.Some(foreground)
		}),
		MaximumSize:   allOpt(ui.SizeInfinite),
		Shape:         widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](painting.NewStadiumBorder()),
		MouseCursor:   widgets.WidgetStateMouseCursorAdaptiveClickable,
		VisualDensity: foundation.Some(theme.VisualDensity),
		TapTargetSize: theme.MaterialTapTargetSize,
		SplashFactory: theme.SplashFactory,
	}
}

// Flutter: material/filled_button.dart _FilledButtonDefaultsM3
func filledButtonDefaultsM3(bc widgets.BuildContext) *ButtonStyle {
	colors := ThemeOf(bc).ColorScheme
	return filledButtonDefaults(bc, colors.Primary(), colors.OnPrimary())
}

// Flutter: material/filled_button.dart _FilledTonalButtonDefaultsM3
func filledTonalButtonDefaultsM3(bc widgets.BuildContext) *ButtonStyle {
	colors := ThemeOf(bc).ColorScheme
	return filledButtonDefaults(bc, colors.SecondaryContainer(), colors.OnSecondaryContainer())
}
