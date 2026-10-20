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

// FilterChipOpts are the optional parameters of NewFilterChip.
//
// Flutter: material/filter_chip.dart FilterChip.new
type FilterChipOpts struct {
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
	// SelectedColor is the color of the chip's background when it is selected.
	SelectedColor foundation.Opt[ui.Color]
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
	// SelectedShadowColor is the color of the shadow when the chip is selected.
	SelectedShadowColor foundation.Opt[ui.Color]
	// ShowCheckmark says whether a selected chip shows a check mark.
	ShowCheckmark foundation.Opt[bool]
	// CheckmarkColor is the color of the check mark.
	CheckmarkColor foundation.Opt[ui.Color]
	// AvatarBorder is the shape of the avatar, which the selection scrim follows; the default is a circle.
	AvatarBorder painting.ShapeBorder
	// IconTheme is the theme of the icons of the chip.
	IconTheme foundation.Opt[widgets.IconThemeData]
	// AvatarBoxConstraints are the constraints of the avatar.
	AvatarBoxConstraints foundation.Opt[rendering.BoxConstraints]
	// DeleteIconBoxConstraints are the constraints of the delete icon.
	DeleteIconBoxConstraints foundation.Opt[rendering.BoxConstraints]
	// ChipAnimationStyle overrides the durations of the animations of the chip.
	ChipAnimationStyle *ChipAnimationStyle
	// MouseCursor is the cursor over the chip.
	MouseCursor services.MouseCursor
}

// FilterChip is a Material Design filter chip: a compact element that uses tags or descriptive words as a way to filter content. The chip is disabled when it has no OnSelected.
//
// Flutter: material/filter_chip.dart FilterChip
type FilterChip struct {
	widgets.StatelessWidget
	label    widgets.Widget
	o        FilterChipOpts
	elevated bool
}

// NewFilterChip creates a chip (flat) with the label. In debug builds it panics with the tag FilterChipPressElevationNegative or FilterChipElevationNegative when an elevation is negative.
//
// Flutter: material/filter_chip.dart FilterChip.new
func NewFilterChip(label widgets.Widget, opts ...FilterChipOpts) *FilterChip {
	o := firstOpts(opts)
	debugCheckChipElevations("FilterChip", o.PressElevation, o.Elevation)
	return &FilterChip{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), label: label, o: o, elevated: false}
}

// NewFilterChipElevated creates a chip that is raised. In debug builds it panics with the tag FilterChipPressElevationNegative or FilterChipElevationNegative when an elevation is negative.
//
// Flutter: material/filter_chip.dart FilterChip.elevated
func NewFilterChipElevated(label widgets.Widget, opts ...FilterChipOpts) *FilterChip {
	o := firstOpts(opts)
	debugCheckChipElevations("FilterChip", o.PressElevation, o.Elevation)
	return &FilterChip{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), label: label, o: o, elevated: true}
}

// Label is the primary content of the chip.
//
// Flutter: material/filter_chip.dart FilterChip.label
func (w *FilterChip) Label() widgets.Widget { return w.label }

// IsEnabled says whether the chip is enabled.
//
// Flutter: material/filter_chip.dart FilterChip.isEnabled
func (w *FilterChip) IsEnabled() bool { return w.o.OnSelected != nil }

// Build builds a RawChip with the settings of the chip.
//
// Flutter: material/filter_chip.dart FilterChip.build
func (w *FilterChip) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	o := w.o
	theme := ThemeOf(bc)
	var defaults foundation.Opt[ChipThemeData]
	if theme.UseMaterial3 {
		defaults = foundation.Some(selectableChipDefaultsM3(bc, w.IsEnabled(), w.o.Selected, w.elevated))
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
		PressElevation:             o.PressElevation,
		Tooltip:                    o.Tooltip,
		Side:                       o.Side,
		StateSide:                  o.StateSide,
		Shape:                      o.Shape,
		ClipBehavior:               o.ClipBehavior,
		FocusNode:                  o.FocusNode,
		Autofocus:                  o.Autofocus,
		Color:                      o.Color,
		BackgroundColor:            o.BackgroundColor,
		DisabledColor:              o.DisabledColor,
		SelectedColor:              o.SelectedColor,
		Padding:                    o.Padding,
		VisualDensity:              o.VisualDensity,
		MaterialTapTargetSize:      o.MaterialTapTargetSize,
		Elevation:                  o.Elevation,
		ShadowColor:                o.ShadowColor,
		SurfaceTintColor:           o.SurfaceTintColor,
		SelectedShadowColor:        o.SelectedShadowColor,
		ShowCheckmark:              o.ShowCheckmark,
		CheckmarkColor:             o.CheckmarkColor,
		AvatarBorder:               o.AvatarBorder,
		IconTheme:                  o.IconTheme,
		AvatarBoxConstraints:       o.AvatarBoxConstraints,
		DeleteIconBoxConstraints:   o.DeleteIconBoxConstraints,
		ChipAnimationStyle:         o.ChipAnimationStyle,
		MouseCursor:                o.MouseCursor,
		DeleteIcon:                 deleteIcon,
		Selected:                   w.o.Selected,
		IsEnabled:                  foundation.Some(w.IsEnabled()),
	})
}
