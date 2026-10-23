// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// SwitchListTileOpts are the optional parameters of NewSwitchListTile.
//
// Flutter: material/switch_list_tile.dart SwitchListTile.new
type SwitchListTileOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ActiveColor is the color of the thumb when the switch is on.
	ActiveColor foundation.Opt[ui.Color]
	// ActiveThumbColor is the color of the thumb when the switch is on.
	ActiveThumbColor foundation.Opt[ui.Color]
	// ActiveTrackColor is the color of the track when the switch is on.
	ActiveTrackColor foundation.Opt[ui.Color]
	// InactiveThumbColor is the color of the thumb when the switch is off.
	InactiveThumbColor foundation.Opt[ui.Color]
	// InactiveTrackColor is the color of the track when the switch is off.
	InactiveTrackColor foundation.Opt[ui.Color]
	// ActiveThumbImage is the image of the thumb when the switch is on.
	ActiveThumbImage painting.ImageProvider
	// OnActiveThumbImageError is called when the image of the active thumb fails.
	OnActiveThumbImageError painting.ImageErrorListener
	// InactiveThumbImage is the image of the thumb when the switch is off.
	InactiveThumbImage painting.ImageProvider
	// OnInactiveThumbImageError is called when the image of the inactive thumb fails.
	OnInactiveThumbImageError painting.ImageErrorListener
	// ThumbColor is the color of the thumb, in each state.
	ThumbColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TrackColor is the color of the track, in each state.
	TrackColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TrackOutlineColor is the color of the track's outline, in each state.
	TrackOutlineColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// ThumbIcon is the icon of the thumb, in each state.
	ThumbIcon widgets.WidgetStateProperty[*widgets.Icon]
	// MaterialTapTargetSize is the tap target size of the switch; the default is
	// shrinkWrap.
	MaterialTapTargetSize MaterialTapTargetSize
	// DragStartBehavior is how the drag of the thumb starts.
	DragStartBehavior gestures.DragStartBehavior
	// MouseCursor is the cursor of a pointer over the switch.
	MouseCursor services.MouseCursor
	// OverlayColor is the color of the switch's overlay, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashRadius is the radius of the switch's splash.
	SplashRadius foundation.Opt[float64]
	// FocusNode is the focus node of the tile.
	FocusNode *widgets.FocusNode
	// StatesController is the controller of the states of the tile.
	StatesController *widgets.WidgetStatesController
	// OnFocusChange is called when the focus of the tile changes.
	OnFocusChange func(bool)
	// Autofocus is whether the tile focuses itself when it is first built.
	Autofocus bool
	// TileColor is the background color of the tile.
	TileColor foundation.Opt[ui.Color]
	// Title is the primary content of the tile.
	Title widgets.Widget
	// Subtitle is additional content displayed below the title.
	Subtitle widgets.Widget
	// Secondary is a widget to display on the side opposite to the switch.
	Secondary widgets.Widget
	// IsThreeLine is whether the tile is intended to display three lines of text.
	IsThreeLine foundation.Opt[bool]
	// Dense is whether the tile is part of a vertically dense list.
	Dense foundation.Opt[bool]
	// ContentPadding is the tile's internal padding.
	ContentPadding painting.EdgeInsetsGeometry
	// Selected is whether the tile looks selected.
	Selected bool
	// ControlAffinity is where to place the switch relative to the text; the
	// default is platform, which is trailing.
	ControlAffinity ListTileControlAffinity
	// Shape is the shape of the tile.
	Shape painting.ShapeBorder
	// SelectedTileColor is the background color of a selected tile.
	SelectedTileColor foundation.Opt[ui.Color]
	// VisualDensity defines how compact the tile's layout is.
	VisualDensity foundation.Opt[VisualDensity]
	// EnableFeedback is whether a tap gives feedback.
	EnableFeedback foundation.Opt[bool]
	// HorizontalTitleGap is the horizontal gap between the title and the leading
	// and trailing widgets.
	HorizontalTitleGap foundation.Opt[float64]
	// MinVerticalPadding is the minimum padding above and below the content.
	MinVerticalPadding foundation.Opt[float64]
	// MinLeadingWidth is the minimum width of the leading widget.
	MinLeadingWidth foundation.Opt[float64]
	// MinTileHeight is the minimum height of the tile.
	MinTileHeight foundation.Opt[float64]
	// HoverColor is the color of the tile when a pointer hovers over it.
	HoverColor foundation.Opt[ui.Color]
	// InternalAddSemanticForOnTap gives the tile the semantics of a button.
	InternalAddSemanticForOnTap bool
}

// SwitchListTile is a ListTile with a Switch.
//
// Flutter: material/switch_list_tile.dart SwitchListTile
type SwitchListTile struct {
	widgets.StatelessWidget
	value     bool
	onChanged func(bool)
	o         SwitchListTileOpts
}

// NewSwitchListTile creates a list tile with a switch. A nil onChanged disables
// the tile. The tags of its failures are
// SwitchListTileThumbImageErrorWithoutImage and SwitchListTileThreeLineNoSubtitle.
//
// Flutter: material/switch_list_tile.dart SwitchListTile.new
func NewSwitchListTile(value bool, onChanged func(bool), opts ...SwitchListTileOpts) *SwitchListTile {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.ActiveThumbImage == nil && o.OnActiveThumbImageError != nil || o.InactiveThumbImage == nil && o.OnInactiveThumbImageError != nil {
			panic(errors.New(tagSwitchListTileThumbImageErrorWithoutImage, "the error listener of a thumb image needs the image"))
		}
		if o.IsThreeLine.OrElse(false) && o.Subtitle == nil {
			panic(errors.New(tagSwitchListTileThreeLineNoSubtitle, "a three line list tile needs a subtitle"))
		}
	}
	return &SwitchListTile{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: value, onChanged: onChanged, o: o}
}

// NewSwitchListTileAdaptive creates a list tile with a switch that adapts to the
// platform.
//
// Deviation: it is the same as NewSwitchListTile (SPEC §12.10).
//
// Flutter: material/switch_list_tile.dart SwitchListTile.adaptive
func NewSwitchListTileAdaptive(value bool, onChanged func(bool), opts ...SwitchListTileOpts) *SwitchListTile {
	return NewSwitchListTile(value, onChanged, opts...)
}

// Build builds the tile.
//
// Flutter: material/switch_list_tile.dart SwitchListTile.build
func (w *SwitchListTile) Build(bc widgets.BuildContext) widgets.Widget {
	o := w.o
	control := widgets.NewExcludeFocus(NewSwitch(w.value, w.onChanged, SwitchOpts{
		ActiveColor: o.ActiveColor, ActiveThumbColor: o.ActiveThumbColor, ActiveThumbImage: o.ActiveThumbImage,
		InactiveThumbImage: o.InactiveThumbImage, MaterialTapTargetSize: firstNonZero(o.MaterialTapTargetSize, MaterialTapTargetSizeShrinkWrap),
		ActiveTrackColor: o.ActiveTrackColor, InactiveTrackColor: o.InactiveTrackColor, InactiveThumbColor: o.InactiveThumbColor,
		Autofocus: o.Autofocus, OnFocusChange: o.OnFocusChange, OnActiveThumbImageError: o.OnActiveThumbImageError,
		OnInactiveThumbImageError: o.OnInactiveThumbImageError, ThumbColor: o.ThumbColor, TrackColor: o.TrackColor,
		TrackOutlineColor: o.TrackOutlineColor, ThumbIcon: o.ThumbIcon, DragStartBehavior: o.DragStartBehavior,
		MouseCursor: o.MouseCursor, SplashRadius: o.SplashRadius, OverlayColor: o.OverlayColor,
	}))
	listTileTheme := ListTileThemeOf(bc)
	affinity := firstNonZero(o.ControlAffinity, listTileTheme.ControlAffinity, ListTileControlAffinityPlatform)
	leading, trailing := o.Secondary, widgets.Widget(control)
	if affinity == ListTileControlAffinityLeading {
		leading, trailing = control, o.Secondary
	}
	theme := ThemeOf(bc)
	switchTheme := SwitchThemeOf(bc)
	var states widgets.WidgetStates
	if o.Selected {
		states = states.With(widgets.WidgetStateSelected)
	}
	var fromTheme foundation.Opt[ui.Color]
	if switchTheme.ThumbColor != nil {
		fromTheme = switchTheme.ThumbColor.Resolve(states)
	}
	effectiveActiveColor := firstOpt(o.ActiveThumbColor, o.ActiveColor, fromTheme, foundation.Some(theme.ColorScheme.Secondary()))
	var onTap func()
	if w.onChanged != nil {
		onTap = func() { w.onChanged(!w.value) }
	}
	return widgets.NewMergeSemantics(NewListTile(ListTileOpts{
		SelectedColor: effectiveActiveColor, Leading: leading, Title: o.Title, Subtitle: o.Subtitle, Trailing: trailing,
		IsThreeLine: o.IsThreeLine, Dense: o.Dense, ContentPadding: o.ContentPadding, Disabled: w.onChanged == nil, OnTap: onTap,
		Selected: o.Selected, SelectedTileColor: o.SelectedTileColor, Autofocus: o.Autofocus, Shape: o.Shape, TileColor: o.TileColor,
		VisualDensity: o.VisualDensity, FocusNode: o.FocusNode, StatesController: o.StatesController, OnFocusChange: o.OnFocusChange,
		EnableFeedback: o.EnableFeedback, HorizontalTitleGap: o.HorizontalTitleGap, MinVerticalPadding: o.MinVerticalPadding,
		MinLeadingWidth: o.MinLeadingWidth, MinTileHeight: o.MinTileHeight, HoverColor: o.HoverColor,
		internalNoSemanticForOnTap: !o.InternalAddSemanticForOnTap,
	}))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSwitchListTileThreeLineNoSubtitle         = "SwitchListTileThreeLineNoSubtitle"
	tagSwitchListTileThumbImageErrorWithoutImage = "SwitchListTileThumbImageErrorWithoutImage"
)
