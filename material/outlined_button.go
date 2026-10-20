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

// OutlinedButtonOpts are the optional parameters of NewOutlinedButton.
//
// Flutter: material/outlined_button.dart OutlinedButton.new
type OutlinedButtonOpts = ButtonStyleButtonOpts

// OutlinedButtonIconOpts are the optional parameters of NewOutlinedButtonIcon.
//
// Flutter: material/outlined_button.dart OutlinedButton.icon
type OutlinedButtonIconOpts = TextButtonIconOpts

// OutlinedButton is a Material Design outlined button: a text button with an
// outline, for the medium emphasis actions that are important but are not the
// primary action of the app.
//
// Flutter: material/outlined_button.dart OutlinedButton
type OutlinedButton struct {
	ButtonStyleButton
	addPadding bool
}

// NewOutlinedButton creates an outlined button. A nil onPressed (without
// OnLongPress) disables the button.
//
// Flutter: material/outlined_button.dart OutlinedButton.new
func NewOutlinedButton(onPressed func(), child widgets.Widget, opts ...OutlinedButtonOpts) *OutlinedButton {
	return &OutlinedButton{ButtonStyleButton: NewButtonStyleButtonBase(onPressed, child, firstOpts(opts))}
}

// NewOutlinedButtonIcon creates an outlined button with an icon and a label. The
// icon may be nil, and then the button has only the label.
//
// Flutter: material/outlined_button.dart OutlinedButton.icon
func NewOutlinedButtonIcon(onPressed func(), label widgets.Widget, opts ...OutlinedButtonIconOpts) *OutlinedButton {
	o := firstOpts(opts)
	child := label
	if o.Icon != nil {
		child = &buttonWithIconChild{StatelessWidget: widgets.NewStatelessWidgetBase(), label: label, icon: o.Icon, buttonStyle: o.Style,
			iconAlignment: o.IconAlignment, themeStyle: func(bc widgets.BuildContext) *ButtonStyle { return OutlinedButtonThemeOf(bc).Style }, gap1: 8, gap2: 4}
	}
	return &OutlinedButton{ButtonStyleButton: NewButtonStyleButtonBase(onPressed, child, o.ButtonStyleButtonOpts), addPadding: o.Icon != nil}
}

// OutlinedButtonStyleFrom is a static convenience that makes a ButtonStyle from
// simple values.
//
// Flutter: material/outlined_button.dart OutlinedButton.styleFrom
func OutlinedButtonStyleFrom(o ButtonStyleFromOpts) *ButtonStyle {
	return buttonStyleFrom(o, styleFromOutlined)
}

// DefaultStyleOf is the default style of the button for the theme: Material 3 or
// Material 2.
//
// Flutter: material/outlined_button.dart OutlinedButton.defaultStyleOf
func (w *OutlinedButton) DefaultStyleOf(bc widgets.BuildContext) *ButtonStyle {
	theme := ThemeOf(bc)
	colorScheme := theme.ColorScheme
	var buttonStyle *ButtonStyle
	if theme.UseMaterial3 {
		buttonStyle = outlinedButtonDefaultsM3(bc)
	} else {
		buttonStyle = OutlinedButtonStyleFrom(ButtonStyleFromOpts{
			ForegroundColor:         foundation.Some(colorScheme.Primary()),
			DisabledForegroundColor: foundation.Some(colorScheme.OnSurface().WithOpacity(0.38)),
			BackgroundColor:         foundation.Some(Colors.Transparent),
			DisabledBackgroundColor: foundation.Some(Colors.Transparent),
			ShadowColor:             foundation.Some(theme.ShadowColor),
			Elevation:               foundation.Some(0.0),
			TextStyle:               theme.TextTheme.LabelLarge,
			Padding:                 elevatedButtonScaledPadding(bc),
			MinimumSize:             foundation.Some(ui.Size{Width: 64, Height: 36}),
			MaximumSize:             foundation.Some(ui.SizeInfinite),
			Side:                    foundation.Some(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colorScheme.OnSurface().WithOpacity(0.12))})),
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
	if w.addPadding && theme.UseMaterial3 {
		effectiveTextScale := buttonEffectiveTextScale(bc, buttonStyle)
		scaled := ButtonStyleButtonScaledPadding(
			painting.NewEdgeInsetsDirectionalFromSTEB(16, 0, 24, 0),
			painting.NewEdgeInsetsDirectionalFromSTEB(8, 0, 12, 0),
			painting.NewEdgeInsetsDirectionalFromSTEB(4, 0, 6, 0), effectiveTextScale)
		return buttonStyle.CopyWith(ButtonStyleOpts{Padding: widgets.NewWidgetStatePropertyAll(scaled)})
	}
	return buttonStyle
}

// ThemeStyleOf is the style of the OutlinedButtonTheme of the context.
//
// Flutter: material/outlined_button.dart OutlinedButton.themeStyleOf
func (w *OutlinedButton) ThemeStyleOf(bc widgets.BuildContext) *ButtonStyle {
	return OutlinedButtonThemeOf(bc).Style
}

// outlinedButtonDefaultsM3 is the default style of an outlined button in
// Material 3.
//
// Flutter: material/outlined_button.dart _OutlinedButtonDefaultsM3
func outlinedButtonDefaultsM3(bc widgets.BuildContext) *ButtonStyle {
	s := textButtonDefaultsM3(bc)
	colors := ThemeOf(bc).ColorScheme
	s.Padding = widgets.NewWidgetStatePropertyAll(elevatedButtonScaledPadding(bc))
	s.Side = widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[painting.BorderSide] {
		c := colors.Outline()
		switch {
		case states.Has(widgets.WidgetStateDisabled):
			c = colors.OnSurface().WithOpacity(0.12)
		case states.Has(widgets.WidgetStateFocused):
			c = colors.Primary()
		}
		return foundation.Some(painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(c)}))
	})
	return s
}
