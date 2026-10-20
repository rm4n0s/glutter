// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ActionChipOpts are the optional parameters of NewActionChip.
//
// Flutter: material/action_chip.dart ActionChip.new
type ActionChipOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Avatar is the widget before the label, typically a CircleAvatar.
	Avatar widgets.Widget
	// LabelStyle is the style of the label, merged over that of the theme.
	LabelStyle *painting.TextStyle
	// LabelPadding is the padding around the label.
	LabelPadding painting.EdgeInsetsGeometry
	// OnPressed is called when the chip is tapped; the chip is disabled when it is nil.
	OnPressed func()
	// PressElevation is the elevation while the chip is pressed.
	PressElevation foundation.Opt[float64]
	// Tooltip is the message of the tooltip of the chip.
	Tooltip foundation.Opt[string]
	// Side is the border of the chip, as one border for every state.
	Side foundation.Opt[painting.BorderSide]
	// StateSide is the border of the chip in each state (Flutter's WidgetStateBorderSide); it replaces Side when it is set.
	StateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// Shape is the shape of the chip's border and of its ink splash.
	Shape painting.OutlinedBorder
	// ClipBehavior is how the content is clipped; the default is ui.ClipNone.
	ClipBehavior ui.Clip
	// FocusNode is the focus node of the chip.
	FocusNode *widgets.FocusNode
	// Autofocus says whether the chip takes the focus when it is first built.
	Autofocus bool
	// Color is the color of the chip, in each state; it takes precedence over the other colors.
	Color widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// BackgroundColor is the color of the chip's background.
	BackgroundColor foundation.Opt[ui.Color]
	// DisabledColor is the color of the chip's background when it is disabled.
	DisabledColor foundation.Opt[ui.Color]
	// Padding is the padding between the chip's border and its content.
	Padding painting.EdgeInsetsGeometry
	// VisualDensity is how compact the chip is.
	VisualDensity foundation.Opt[VisualDensity]
	// MaterialTapTargetSize is the size of the tap target; the default is that of the theme.
	MaterialTapTargetSize MaterialTapTargetSize
	// Elevation is the elevation of the chip.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the chip's shadow.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the color of the surface tint overlay.
	SurfaceTintColor foundation.Opt[ui.Color]
	// IconTheme is the theme of the icons of the chip.
	IconTheme foundation.Opt[widgets.IconThemeData]
	// AvatarBoxConstraints are the constraints of the avatar.
	AvatarBoxConstraints foundation.Opt[rendering.BoxConstraints]
	// ChipAnimationStyle overrides the durations of the animations of the chip.
	ChipAnimationStyle *ChipAnimationStyle
	// MouseCursor is the cursor over the chip.
	MouseCursor services.MouseCursor
}

// ActionChip is a Material Design chip that triggers an action: a compact element that represents an action related to primary content. The chip is disabled when it has no OnPressed.
//
// Flutter: material/action_chip.dart ActionChip
type ActionChip struct {
	widgets.StatelessWidget
	label    widgets.Widget
	o        ActionChipOpts
	elevated bool
}

// NewActionChip creates a chip (flat) with the label. In debug builds it panics with the tag ActionChipPressElevationNegative or ActionChipElevationNegative when an elevation is negative.
//
// Flutter: material/action_chip.dart ActionChip.new
func NewActionChip(label widgets.Widget, opts ...ActionChipOpts) *ActionChip {
	o := firstOpts(opts)
	debugCheckChipElevations("ActionChip", o.PressElevation, o.Elevation)
	return &ActionChip{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), label: label, o: o, elevated: false}
}

// NewActionChipElevated creates a chip that is raised. In debug builds it panics with the tag ActionChipPressElevationNegative or ActionChipElevationNegative when an elevation is negative.
//
// Flutter: material/action_chip.dart ActionChip.elevated
func NewActionChipElevated(label widgets.Widget, opts ...ActionChipOpts) *ActionChip {
	o := firstOpts(opts)
	debugCheckChipElevations("ActionChip", o.PressElevation, o.Elevation)
	return &ActionChip{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), label: label, o: o, elevated: true}
}

// Label is the primary content of the chip.
//
// Flutter: material/action_chip.dart ActionChip.label
func (w *ActionChip) Label() widgets.Widget { return w.label }

// IsEnabled says whether the chip is enabled.
//
// Flutter: material/action_chip.dart ActionChip.isEnabled
func (w *ActionChip) IsEnabled() bool { return w.o.OnPressed != nil }

// Build builds a RawChip with the settings of the chip.
//
// Flutter: material/action_chip.dart ActionChip.build
func (w *ActionChip) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	o := w.o
	theme := ThemeOf(bc)
	var defaults foundation.Opt[ChipThemeData]
	if theme.UseMaterial3 {
		defaults = foundation.Some(actionChipDefaultsM3(bc, w.IsEnabled(), w.elevated))
	}
	return NewRawChip(w.label, RawChipOpts{
		DefaultProperties:     defaults,
		Avatar:                o.Avatar,
		LabelStyle:            o.LabelStyle,
		LabelPadding:          o.LabelPadding,
		OnPressed:             o.OnPressed,
		PressElevation:        o.PressElevation,
		Tooltip:               o.Tooltip,
		Side:                  o.Side,
		StateSide:             o.StateSide,
		Shape:                 o.Shape,
		ClipBehavior:          o.ClipBehavior,
		FocusNode:             o.FocusNode,
		Autofocus:             o.Autofocus,
		Color:                 o.Color,
		BackgroundColor:       o.BackgroundColor,
		DisabledColor:         o.DisabledColor,
		Padding:               o.Padding,
		VisualDensity:         o.VisualDensity,
		MaterialTapTargetSize: o.MaterialTapTargetSize,
		Elevation:             o.Elevation,
		ShadowColor:           o.ShadowColor,
		SurfaceTintColor:      o.SurfaceTintColor,
		IconTheme:             o.IconTheme,
		AvatarBoxConstraints:  o.AvatarBoxConstraints,
		ChipAnimationStyle:    o.ChipAnimationStyle,
		MouseCursor:           o.MouseCursor,
		IsEnabled:             foundation.Some(w.IsEnabled()),
	})
}
