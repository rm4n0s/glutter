// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// menuDefaultBorder is the shape of the menu panels in Material 3.
var menuDefaultBorder painting.OutlinedBorder = painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{
	BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(4.0))})

// menuBarDefaultsM3 is the default style of the panel of a MenuBar in Material 3.
//
// Flutter: material/menu_anchor.dart _MenuBarDefaultsM3
func menuBarDefaultsM3(bc widgets.BuildContext) *MenuStyle {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	return &MenuStyle{
		Elevation:        allOpt(3.0),
		Shape:            widgets.NewWidgetStatePropertyAll(menuDefaultBorder),
		Alignment:        painting.AlignmentDirectionalBottomStart,
		BackgroundColor:  allOpt(colors.SurfaceContainer()),
		ShadowColor:      allOpt(colors.Shadow()),
		SurfaceTintColor: allOpt(Colors.Transparent),
		Padding: widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](
			painting.NewEdgeInsetsDirectionalSymmetric(menuTopLevelHorizontalMinPadding, 0)),
		VisualDensity: foundation.Some(theme.VisualDensity),
	}
}

// menuDefaultsM3 is the default style of the panel of a menu in Material 3.
//
// Flutter: material/menu_anchor.dart _MenuDefaultsM3
func menuDefaultsM3(bc widgets.BuildContext) *MenuStyle {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	return &MenuStyle{
		Elevation:        allOpt(3.0),
		Shape:            widgets.NewWidgetStatePropertyAll(menuDefaultBorder),
		Alignment:        painting.AlignmentDirectionalTopEnd,
		BackgroundColor:  allOpt(colors.SurfaceContainer()),
		SurfaceTintColor: allOpt(Colors.Transparent),
		ShadowColor:      allOpt(colors.Shadow()),
		Padding: widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](
			painting.NewEdgeInsetsDirectionalSymmetric(0, menuVerticalMinPadding)),
		VisualDensity: foundation.Some(theme.VisualDensity),
	}
}

// menuButtonDefaultsM3 is the default style of the buttons of a menu in
// Material 3.
//
// Flutter: material/menu_anchor.dart _MenuButtonDefaultsM3
func menuButtonDefaultsM3(bc widgets.BuildContext) *ButtonStyle {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	return &ButtonStyle{
		AnimationDuration: foundation.Some(ThemeChangeDuration),
		EnableFeedback:    foundation.Some(true),
		Alignment:         painting.AlignmentDirectionalCenterStart,
		BackgroundColor:   allOpt(Colors.Transparent),
		Elevation:         allOpt(0.0),
		ForegroundColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if states.Has(widgets.WidgetStateDisabled) {
				return foundation.Some(colors.OnSurface().WithOpacity(0.38))
			}
			return foundation.Some(colors.OnSurface())
		}),
		IconColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if states.Has(widgets.WidgetStateDisabled) {
				return foundation.Some(colors.OnSurface().WithOpacity(0.38))
			}
			return foundation.Some(colors.OnSurfaceVariant())
		}),
		IconSize:    allOpt(24.0),
		MaximumSize: allOpt(ui.SizeInfinite),
		MinimumSize: allOpt(ui.Size{Width: 64.0, Height: 48.0}),
		MouseCursor: widgets.WidgetStateMouseCursorAdaptiveClickable,
		OverlayColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			switch {
			case states.Has(widgets.WidgetStatePressed):
				return foundation.Some(colors.OnSurface().WithOpacity(0.1))
			case states.Has(widgets.WidgetStateHovered):
				return foundation.Some(colors.OnSurface().WithOpacity(0.08))
			case states.Has(widgets.WidgetStateFocused):
				return foundation.Some(colors.OnSurface().WithOpacity(0.1))
			}
			return foundation.Some(Colors.Transparent)
		}),
		Padding:       allNilable[painting.EdgeInsetsGeometry](menuButtonScaledPadding(bc)),
		Shape:         widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](painting.NewRoundedRectangleBorder()),
		SplashFactory: theme.SplashFactory,
		TapTargetSize: theme.MaterialTapTargetSize,
		// This is taken from https://m3.material.io/components/menus/specs.
		TextStyle:     widgets.NewWidgetStatePropertyAll(theme.TextTheme.LabelLarge),
		VisualDensity: foundation.Some(theme.VisualDensity),
	}
}

// menuButtonScaledPadding is the padding of a menu button, which scales with the
// text. The horizontal padding comes from the spec.
//
// Flutter: material/menu_anchor.dart _MenuButtonDefaultsM3._scaledPadding
func menuButtonScaledPadding(bc widgets.BuildContext) painting.EdgeInsetsGeometry {
	theme := ThemeOf(bc)
	visualDensity := theme.VisualDensity
	// When the horizontal VisualDensity is greater than zero, set it to zero
	// because the ButtonStyleButton has already handled the padding based on the
	// density. However, the ButtonStyleButton doesn't allow the VisualDensity
	// adjustment to reduce the width of the left/right padding, so it is handled
	// here if the density is less than zero, such as on desktop platforms.
	if visualDensity.Horizontal > 0 {
		visualDensity = NewVisualDensity(VisualDensityOpts{Vertical: foundation.Some(visualDensity.Vertical)})
	}
	// Since the threshold paddings used below are empirical values determined at
	// a font size of 14.0, 14.0 is used as the base value for scaling the padding.
	fontSize := 14.0
	if ls := theme.TextTheme.LabelLarge; ls != nil {
		fontSize = ls.FontSize.OrElse(14.0)
	}
	fontSizeRatio := widgets.MediaQueryTextScalerOf(bc).Scale(fontSize) / 14.0
	adjustment := visualDensity.BaseSizeAdjustment()
	return ButtonStyleButtonScaledPadding(
		painting.NewEdgeInsetsSymmetric(0, math.Max(menuViewPadding, menuLabelItemDefaultSpacing+adjustment.Dx)),
		painting.NewEdgeInsetsSymmetric(0, math.Max(menuViewPadding, 8+adjustment.Dx)),
		painting.NewEdgeInsetsSymmetric(0, menuViewPadding),
		fontSizeRatio)
}
