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

// InputChipOpts are the optional parameters of NewInputChip.
//
// Flutter: material/input_chip.dart InputChip.new
type InputChipOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Selected says whether the chip is selected.
	Selected bool
	// Avatar is the widget before the label, typically a CircleAvatar.
	Avatar widgets.Widget
	// LabelStyle is the style of the label, merged over that of the theme.
	LabelStyle *painting.TextStyle
	// LabelPadding is the padding around the label.
	LabelPadding painting.EdgeInsetsGeometry
	// IsEnabled says whether the chip is enabled; the default is true. The chip is disabled anyway when it has no callback.
	IsEnabled foundation.Opt[bool]
	// OnSelected is called when the chip is tapped, with the value that it asks for; the chip is disabled when it is nil.
	OnSelected func(selected bool)
	// DeleteIcon is the icon that is shown when OnDeleted is set.
	DeleteIcon widgets.Widget
	// OnDeleted is called when the delete icon is pressed; the chip shows the icon only when it is set.
	OnDeleted func()
	// DeleteIconColor is the color of the delete icon.
	DeleteIconColor foundation.Opt[ui.Color]
	// DeleteButtonTooltipMessage is the message of the tooltip of the delete icon; the default is that of the localizations.
	DeleteButtonTooltipMessage foundation.Opt[string]
	// OnPressed is called when the chip is tapped; the chip is disabled when it is nil.
	OnPressed func()
	// PressElevation is the elevation while the chip is pressed.
	PressElevation foundation.Opt[float64]
	// DisabledColor is the color of the chip's background when it is disabled.
	DisabledColor foundation.Opt[ui.Color]
	// SelectedColor is the color of the chip's background when it is selected.
	SelectedColor foundation.Opt[ui.Color]
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
	// SelectedShadowColor is the color of the shadow when the chip is selected.
	SelectedShadowColor foundation.Opt[ui.Color]
	// ShowCheckmark says whether a selected chip shows a check mark.
	ShowCheckmark foundation.Opt[bool]
	// CheckmarkColor is the color of the check mark.
	CheckmarkColor foundation.Opt[ui.Color]
	// AvatarBorder is the shape of the avatar, which the selection scrim follows; the default is a circle.
	AvatarBorder painting.ShapeBorder
	// AvatarBoxConstraints are the constraints of the avatar.
	AvatarBoxConstraints foundation.Opt[rendering.BoxConstraints]
	// DeleteIconBoxConstraints are the constraints of the delete icon.
	DeleteIconBoxConstraints foundation.Opt[rendering.BoxConstraints]
	// ChipAnimationStyle overrides the durations of the animations of the chip.
	ChipAnimationStyle *ChipAnimationStyle
	// MouseCursor is the cursor over the chip.
	MouseCursor services.MouseCursor
}

// InputChip is a Material Design input chip: a compact element that represents a piece of information, such as an entity (person, place or thing) or text, in a compact form. The chip is disabled when it has no OnSelected, OnDeleted or OnPressed.
//
// Flutter: material/input_chip.dart InputChip
type InputChip struct {
	widgets.StatelessWidget
	label widgets.Widget
	o     InputChipOpts
}

// NewInputChip creates a chip with the label. In debug builds it panics with the tag InputChipPressElevationNegative or InputChipElevationNegative when an elevation is negative.
//
// Flutter: material/input_chip.dart InputChip.new
func NewInputChip(label widgets.Widget, opts ...InputChipOpts) *InputChip {
	o := firstOpts(opts)
	debugCheckChipElevations("InputChip", o.PressElevation, o.Elevation)
	return &InputChip{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), label: label, o: o}
}

// Label is the primary content of the chip.
//
// Flutter: material/input_chip.dart InputChip.label
func (w *InputChip) Label() widgets.Widget { return w.label }

// IsEnabled says whether the chip is enabled.
//
// Flutter: material/input_chip.dart InputChip.isEnabled
func (w *InputChip) IsEnabled() bool {
	return w.o.IsEnabled.OrElse(true) && (w.o.OnSelected != nil || w.o.OnDeleted != nil || w.o.OnPressed != nil)
}

// Build builds a RawChip with the settings of the chip.
//
// Flutter: material/input_chip.dart InputChip.build
func (w *InputChip) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	o := w.o
	theme := ThemeOf(bc)
	var defaults foundation.Opt[ChipThemeData]
	if theme.UseMaterial3 {
		defaults = foundation.Some(inputChipDefaultsM3(bc, w.IsEnabled(), w.o.Selected))
	}
	deleteIcon := o.DeleteIcon
	if deleteIcon == nil && theme.UseMaterial3 {
		deleteIcon = widgets.NewIcon(Icons.Clear(), widgets.IconOpts{Size: foundation.Some(18.0)})
	}
	return NewRawChip(w.label, RawChipOpts{
		DefaultProperties:          defaults,
		Avatar:                     o.Avatar,
		LabelStyle:                 o.LabelStyle,
		LabelPadding:               o.LabelPadding,
		OnSelected:                 o.OnSelected,
		OnDeleted:                  o.OnDeleted,
		DeleteIconColor:            o.DeleteIconColor,
		DeleteButtonTooltipMessage: o.DeleteButtonTooltipMessage,
		OnPressed:                  o.OnPressed,
		PressElevation:             o.PressElevation,
		DisabledColor:              o.DisabledColor,
		SelectedColor:              o.SelectedColor,
		Tooltip:                    o.Tooltip,
		Side:                       o.Side,
		StateSide:                  o.StateSide,
		Shape:                      o.Shape,
		ClipBehavior:               o.ClipBehavior,
		FocusNode:                  o.FocusNode,
		Autofocus:                  o.Autofocus,
		Color:                      o.Color,
		BackgroundColor:            o.BackgroundColor,
		Padding:                    o.Padding,
		VisualDensity:              o.VisualDensity,
		MaterialTapTargetSize:      o.MaterialTapTargetSize,
		Elevation:                  o.Elevation,
		ShadowColor:                o.ShadowColor,
		SurfaceTintColor:           o.SurfaceTintColor,
		IconTheme:                  o.IconTheme,
		SelectedShadowColor:        o.SelectedShadowColor,
		ShowCheckmark:              o.ShowCheckmark,
		CheckmarkColor:             o.CheckmarkColor,
		AvatarBorder:               o.AvatarBorder,
		AvatarBoxConstraints:       o.AvatarBoxConstraints,
		DeleteIconBoxConstraints:   o.DeleteIconBoxConstraints,
		ChipAnimationStyle:         o.ChipAnimationStyle,
		MouseCursor:                o.MouseCursor,
		DeleteIcon:                 deleteIcon,
		Selected:                   w.o.Selected,
		IsEnabled:                  foundation.Some(w.IsEnabled()),
	})
}
