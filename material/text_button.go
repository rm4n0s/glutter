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

// TextButtonOpts are the optional parameters of NewTextButton.
//
// Flutter: material/text_button.dart TextButton.new
type TextButtonOpts = ButtonStyleButtonOpts

// TextButtonIconOpts are the optional parameters of NewTextButtonIcon.
//
// Flutter: material/text_button.dart TextButton.icon
type TextButtonIconOpts struct {
	ButtonStyleButtonOpts
	// Icon is the widget before (or, with IconAlignment, after) the label.
	Icon widgets.Widget
	// IconAlignment is the side of the label that the icon is on.
	IconAlignment IconAlignment
}

// TextButton is a Material Design text button: a label on a Material that has
// no elevation and no outline, so that it is used for low emphasis actions, in
// toolbars, dialogs and inline with other content.
//
// Flutter: material/text_button.dart TextButton
type TextButton struct {
	ButtonStyleButton
	addPadding bool
}

// NewTextButton creates a text button. A nil onPressed (without OnLongPress)
// disables the button.
//
// Flutter: material/text_button.dart TextButton.new
func NewTextButton(onPressed func(), child widgets.Widget, opts ...TextButtonOpts) *TextButton {
	return &TextButton{ButtonStyleButton: NewButtonStyleButtonBase(onPressed, child, firstOpts(opts))}
}

// NewTextButtonIcon creates a text button with an icon and a label. The icon
// may be nil, and then the button has only the label.
//
// Flutter: material/text_button.dart TextButton.icon
func NewTextButtonIcon(onPressed func(), label widgets.Widget, opts ...TextButtonIconOpts) *TextButton {
	o := firstOpts(opts)
	child := label
	if o.Icon != nil {
		child = &buttonWithIconChild{StatelessWidget: widgets.NewStatelessWidgetBase(), label: label, icon: o.Icon, buttonStyle: o.Style,
			iconAlignment: o.IconAlignment, themeStyle: func(bc widgets.BuildContext) *ButtonStyle { return TextButtonThemeOf(bc).Style }, gap1: 8, gap2: 4}
	}
	return &TextButton{ButtonStyleButton: NewButtonStyleButtonBase(onPressed, child, o.ButtonStyleButtonOpts), addPadding: o.Icon != nil}
}

// TextButtonStyleFrom is a static convenience that makes a ButtonStyle from
// simple values: the colors of the states and the other properties that are the
// same in all states.
//
// Flutter: material/text_button.dart TextButton.styleFrom
func TextButtonStyleFrom(o ButtonStyleFromOpts) *ButtonStyle {
	return buttonStyleFrom(o, styleFromText)
}

// DefaultStyleOf is the default style of the button for the theme: Material 3 or
// Material 2.
//
// Flutter: material/text_button.dart TextButton.defaultStyleOf
func (w *TextButton) DefaultStyleOf(bc widgets.BuildContext) *ButtonStyle {
	theme := ThemeOf(bc)
	colorScheme := theme.ColorScheme
	var buttonStyle *ButtonStyle
	if theme.UseMaterial3 {
		buttonStyle = textButtonDefaultsM3(bc)
	} else {
		buttonStyle = TextButtonStyleFrom(ButtonStyleFromOpts{
			ForegroundColor:         foundation.Some(colorScheme.Primary()),
			DisabledForegroundColor: foundation.Some(colorScheme.OnSurface().WithOpacity(0.38)),
			BackgroundColor:         foundation.Some(Colors.Transparent),
			DisabledBackgroundColor: foundation.Some(Colors.Transparent),
			ShadowColor:             foundation.Some(theme.ShadowColor),
			Elevation:               foundation.Some(0.0),
			TextStyle:               theme.TextTheme.LabelLarge,
			Padding:                 textButtonScaledPadding(bc),
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
		defaultFontSize := 14.0
		if buttonStyle.TextStyle != nil {
			if ts := buttonStyle.TextStyle.Resolve(0); ts != nil {
				defaultFontSize = ts.FontSize.OrElse(14.0)
			}
		}
		effectiveTextScale := widgets.MediaQueryTextScalerOf(bc).Scale(defaultFontSize) / 14.0
		var one painting.EdgeInsetsGeometry = painting.NewEdgeInsetsAll(8)
		if theme.UseMaterial3 {
			one = painting.NewEdgeInsetsDirectionalFromSTEB(12, 8, 16, 8)
		}
		scaled := ButtonStyleButtonScaledPadding(one, painting.NewEdgeInsetsSymmetric(0, 4), painting.NewEdgeInsetsSymmetric(0, 4), effectiveTextScale)
		return buttonStyle.CopyWith(ButtonStyleOpts{Padding: widgets.NewWidgetStatePropertyAll(scaled)})
	}
	return buttonStyle
}

// ThemeStyleOf is the style of the TextButtonTheme of the context.
//
// Flutter: material/text_button.dart TextButton.themeStyleOf
func (w *TextButton) ThemeStyleOf(bc widgets.BuildContext) *ButtonStyle {
	return TextButtonThemeOf(bc).Style
}

// textButtonScaledPadding is the padding of a text button for the text scale.
//
// Flutter: material/text_button.dart _scaledPadding
func textButtonScaledPadding(bc widgets.BuildContext) painting.EdgeInsetsGeometry {
	return buttonScaledPadding(bc, painting.NewEdgeInsetsSymmetric(8, 12), painting.NewEdgeInsetsAll(8),
		painting.NewEdgeInsetsSymmetric(0, 8), painting.NewEdgeInsetsSymmetric(0, 4))
}

// textButtonDefaultsM3 is the default style of a text button in Material 3.
//
// Flutter: material/text_button.dart _TextButtonDefaultsM3
func textButtonDefaultsM3(bc widgets.BuildContext) *ButtonStyle {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	return &ButtonStyle{
		AnimationDuration: foundation.Some(ThemeChangeDuration),
		EnableFeedback:    foundation.Some(true),
		Alignment:         painting.AlignmentCenter,
		TextStyle:         widgets.NewWidgetStatePropertyAll(theme.TextTheme.LabelLarge),
		BackgroundColor:   allOpt(Colors.Transparent),
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
		ShadowColor:      allOpt(Colors.Transparent),
		SurfaceTintColor: allOpt(Colors.Transparent),
		Elevation:        allOpt(0.0),
		Padding:          widgets.NewWidgetStatePropertyAll(textButtonScaledPadding(bc)),
		MinimumSize:      allOpt(ui.Size{Width: 64, Height: 40}),
		IconSize:         allOpt(18.0),
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
