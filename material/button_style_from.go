// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ButtonStyleFromOpts are the parameters of TextButtonStyleFrom,
// ElevatedButtonStyleFrom, FilledButtonStyleFrom and OutlinedButtonStyleFrom:
// the colors and values that become the properties of a ButtonStyle, most of
// them the same in every state. The colors of the states of a button come from
// the foreground, background and icon colors and their disabled variants, and
// the overlay colors from the foreground or overlay color.
//
// Flutter: material/text_button.dart TextButton.styleFrom
type ButtonStyleFromOpts struct {
	// ForegroundColor is the color of the text and the icon of the enabled button.
	ForegroundColor foundation.Opt[ui.Color]
	// BackgroundColor is the color of the enabled button.
	BackgroundColor foundation.Opt[ui.Color]
	// DisabledForegroundColor is the foreground color of the disabled button.
	DisabledForegroundColor foundation.Opt[ui.Color]
	// DisabledBackgroundColor is the background color of the disabled button.
	DisabledBackgroundColor foundation.Opt[ui.Color]
	// ShadowColor is the color of the shadow below the button.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the button.
	SurfaceTintColor foundation.Opt[ui.Color]
	// IconColor is the color of the icon of the enabled button.
	IconColor foundation.Opt[ui.Color]
	// IconSize is the size of the icon of the button.
	IconSize foundation.Opt[float64]
	// IconAlignment is the side of the label that the icon is on.
	IconAlignment IconAlignment
	// DisabledIconColor is the color of the icon of the disabled button.
	DisabledIconColor foundation.Opt[ui.Color]
	// OverlayColor is the highlight color of the button; the pressed, hovered
	// and focused colors are this color with opacities, or those of the
	// foreground color when there is none.
	OverlayColor foundation.Opt[ui.Color]
	// Elevation is the elevation of the button at rest.
	Elevation foundation.Opt[float64]
	// TextStyle is the style of the text of the button, merged onto the default
	// label style (see ButtonStyle.TextStyle).
	TextStyle *painting.TextStyle
	// Padding is the padding between the border of the button and its child.
	Padding painting.EdgeInsetsGeometry
	// MinimumSize is the minimum size of the button.
	MinimumSize foundation.Opt[ui.Size]
	// FixedSize is the size of the button.
	FixedSize foundation.Opt[ui.Size]
	// MaximumSize is the maximum size of the button.
	MaximumSize foundation.Opt[ui.Size]
	// Side is the outline of the button.
	Side foundation.Opt[painting.BorderSide]
	// Shape is the shape of the button.
	Shape painting.OutlinedBorder
	// EnabledMouseCursor is the cursor over the enabled button.
	EnabledMouseCursor services.MouseCursor
	// DisabledMouseCursor is the cursor over the disabled button.
	DisabledMouseCursor services.MouseCursor
	// VisualDensity defines how compact the layout of the button is.
	VisualDensity foundation.Opt[VisualDensity]
	// TapTargetSize configures the minimum size of the tap target.
	TapTargetSize MaterialTapTargetSize
	// AnimationDuration is the duration of the changes of the shape and elevation.
	AnimationDuration foundation.Opt[time.Duration]
	// EnableFeedback says whether detected gestures give acoustic and haptic feedback.
	EnableFeedback foundation.Opt[bool]
	// Alignment is the alignment of the child within the button.
	Alignment painting.AlignmentGeometry
	// SplashFactory creates the splashes of the button.
	SplashFactory InteractiveInkFeatureFactory
	// BackgroundBuilder builds a layer between the Material of the button and its child.
	BackgroundBuilder ButtonLayerBuilder
	// ForegroundBuilder builds a layer above the child of the button.
	ForegroundBuilder ButtonLayerBuilder
}

// buttonStyleFromKind says which of the styleFrom functions of Flutter builds
// the style: they differ in few properties.
type buttonStyleFromKind int

const (
	styleFromText buttonStyleFromKind = iota
	styleFromElevated
	styleFromFilled
	styleFromOutlined
)

// allOpt is a property that has the value in every state.
func allOpt[T any](v T) widgets.WidgetStateProperty[foundation.Opt[T]] {
	return widgets.NewWidgetStatePropertyAll(foundation.Some(v))
}

// allOptOr is allOpt for a value that may be none: it is a nil property then.
func allOptOr[T any](v foundation.Opt[T]) widgets.WidgetStateProperty[foundation.Opt[T]] {
	return ButtonStyleButtonAllOrNull(v)
}

// allNilable is a property that has the value in every state, or nil when the
// value is nil.
func allNilable[T comparable](v T) widgets.WidgetStateProperty[T] {
	var zero T
	if v == zero {
		return nil
	}
	return widgets.NewWidgetStatePropertyAll(v)
}

// colorsByStates is a color property from a function of the states.
func colorsByStates(f func(states widgets.WidgetStates) foundation.Opt[ui.Color]) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	return widgets.WidgetStatePropertyResolveWith(f)
}

// opacities of the overlay of a button over its foreground color.
const (
	buttonPressedOverlayOpacity = 0.1
	buttonHoveredOverlayOpacity = 0.08
	buttonFocusedOverlayOpacity = 0.1
)

// overlayColorsOf is the overlay color of the pressed, hovered and focused
// states, in that order of precedence, of the color.
func overlayColorsOf(c ui.Color) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	return widgets.NewWidgetStateMapper(widgets.WidgetStateMap[foundation.Opt[ui.Color]]{
		{Key: widgets.WidgetStatePressed, Value: foundation.Some(c.WithOpacity(buttonPressedOverlayOpacity))},
		{Key: widgets.WidgetStateHovered, Value: foundation.Some(c.WithOpacity(buttonHoveredOverlayOpacity))},
		{Key: widgets.WidgetStateFocused, Value: foundation.Some(c.WithOpacity(buttonFocusedOverlayOpacity))},
	})
}

func buttonStyleFrom(o ButtonStyleFromOpts, kind buttonStyleFromKind) *ButtonStyle {
	// specialDefault is the color property of the styles that give the color of
	// the enabled button to the disabled one when no disabled color is set.
	specialDefault := func(enabled, disabled foundation.Opt[ui.Color]) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
		if enabled.IsSome() && disabled.IsNone() {
			return widgets.NewWidgetStatePropertyAll(enabled)
		}
		return ButtonStyleButtonDefaultColor(enabled, disabled)
	}
	var overlay widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	switch ov, hasOverlay := o.OverlayColor.Get(); {
	case o.ForegroundColor.IsNone() && !hasOverlay:
	case hasOverlay && ov.A == 0:
		overlay = widgets.NewWidgetStatePropertyAll(o.OverlayColor)
	case hasOverlay:
		overlay = overlayColorsOf(ov)
	default:
		overlay = overlayColorsOf(o.ForegroundColor.OrElse(ui.Color{}))
	}

	s := &ButtonStyle{
		ForegroundColor:   ButtonStyleButtonDefaultColor(o.ForegroundColor, o.DisabledForegroundColor),
		OverlayColor:      overlay,
		ShadowColor:       allOptOr(o.ShadowColor),
		SurfaceTintColor:  allOptOr(o.SurfaceTintColor),
		IconSize:          allOptOr(o.IconSize),
		IconAlignment:     o.IconAlignment,
		Padding:           allNilable(o.Padding),
		MinimumSize:       allOptOr(o.MinimumSize),
		FixedSize:         allOptOr(o.FixedSize),
		MaximumSize:       allOptOr(o.MaximumSize),
		Side:              allOptOr(o.Side),
		Shape:             allNilable(o.Shape),
		VisualDensity:     o.VisualDensity,
		TapTargetSize:     o.TapTargetSize,
		AnimationDuration: o.AnimationDuration,
		EnableFeedback:    o.EnableFeedback,
		Alignment:         o.Alignment,
		SplashFactory:     o.SplashFactory,
		BackgroundBuilder: o.BackgroundBuilder,
		ForegroundBuilder: o.ForegroundBuilder,
		TextStyle:         allNilable(o.TextStyle),
		Elevation:         allOptOr(o.Elevation),
	}
	if o.EnabledMouseCursor != nil || o.DisabledMouseCursor != nil {
		s.MouseCursor = widgets.NewWidgetStateMapper(widgets.WidgetStateMap[services.MouseCursor]{
			{Key: widgets.WidgetStateDisabled, Value: o.DisabledMouseCursor},
			{Key: widgets.WidgetStateAny, Value: o.EnabledMouseCursor},
		})
	}
	switch kind {
	case styleFromText, styleFromOutlined:
		s.BackgroundColor = specialDefault(o.BackgroundColor, o.DisabledBackgroundColor)
	default:
		s.BackgroundColor = ButtonStyleButtonDefaultColor(o.BackgroundColor, o.DisabledBackgroundColor)
	}
	if kind == styleFromText {
		s.IconColor = specialDefault(o.IconColor, o.DisabledIconColor)
	} else {
		s.IconColor = ButtonStyleButtonDefaultColor(o.IconColor, o.DisabledIconColor)
	}
	if kind == styleFromElevated {
		s.Elevation = nil
		if e, ok := o.Elevation.Get(); ok {
			s.Elevation = widgets.NewWidgetStateMapper(widgets.WidgetStateMap[foundation.Opt[float64]]{
				{Key: widgets.WidgetStateDisabled, Value: foundation.Some(0.0)},
				{Key: widgets.WidgetStatePressed, Value: foundation.Some(e + 6)},
				{Key: widgets.WidgetStateHovered, Value: foundation.Some(e + 2)},
				{Key: widgets.WidgetStateFocused, Value: foundation.Some(e + 2)},
				{Key: widgets.WidgetStateAny, Value: foundation.Some(e)},
			})
		}
	}
	return s
}

// buttonScaledPadding is the padding of the buttons for the font size of the
// label of the theme and the text scale: the padding for a scale of 1 is
// 12 by 8 in Material 3 and 8 on all sides in Material 2.
func buttonScaledPadding(bc widgets.BuildContext, material3, material2 painting.EdgeInsetsGeometry, padding2x, padding3x painting.EdgeInsetsGeometry) painting.EdgeInsetsGeometry {
	theme := ThemeOf(bc)
	defaultFontSize := 14.0
	if ls := theme.TextTheme.LabelLarge; ls != nil {
		defaultFontSize = ls.FontSize.OrElse(14.0)
	}
	effectiveTextScale := widgets.MediaQueryTextScalerOf(bc).Scale(defaultFontSize) / 14.0
	one := material2
	if theme.UseMaterial3 {
		one = material3
	}
	return ButtonStyleButtonScaledPadding(one, padding2x, padding3x, effectiveTextScale)
}

// buttonIconScale is the scale of the text, from 1 to 2, minus 1, that gives the
// gap between the icon and the label of a button with an icon.
func buttonIconScale(bc widgets.BuildContext, style *ButtonStyle) float64 {
	defaultFontSize := 14.0
	if style != nil && style.TextStyle != nil {
		if ts := style.TextStyle.Resolve(0); ts != nil {
			defaultFontSize = ts.FontSize.OrElse(14.0)
		}
	}
	return ui.ClampDouble(widgets.MediaQueryTextScalerOf(bc).Scale(defaultFontSize)/14.0, 1.0, 2.0) - 1.0
}

// buttonWithIconChild lays an icon and a label out in a row, with the icon on
// the side of the alignment, which is the given one, the one of the style of the
// theme, the one of the style of the button or the start.
//
// Flutter: material/text_button.dart _TextButtonWithIconChild
type buttonWithIconChild struct {
	widgets.StatelessWidget
	label, icon   widgets.Widget
	buttonStyle   *ButtonStyle
	iconAlignment IconAlignment
	themeStyle    func(widgets.BuildContext) *ButtonStyle
	// gaps are the gaps between the icon and the label for a scale of 1 and of 2.
	gap1, gap2 float64
}

func (w *buttonWithIconChild) Build(bc widgets.BuildContext) widgets.Widget {
	scale := buttonIconScale(bc, w.buttonStyle)
	effective := w.iconAlignment
	if effective == IconAlignmentUnspecified {
		if ts := w.themeStyle(bc); ts != nil {
			effective = ts.IconAlignment
		}
	}
	if effective == IconAlignmentUnspecified && w.buttonStyle != nil {
		effective = w.buttonStyle.IconAlignment
	}
	if effective == IconAlignmentUnspecified {
		effective = IconAlignmentStart
	}
	children := []widgets.Widget{w.icon, widgets.NewFlexible(w.label)}
	if effective == IconAlignmentEnd {
		children = []widgets.Widget{widgets.NewFlexible(w.label), w.icon}
	}
	return widgets.NewRow(children, widgets.RowOpts{MainAxisSize: rendering.MainAxisSizeMin, Spacing: ui.LerpDouble(w.gap1, w.gap2, scale)})
}
