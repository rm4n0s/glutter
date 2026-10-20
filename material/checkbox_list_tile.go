// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CheckboxListTileOpts are the optional parameters of NewCheckboxListTile.
//
// Flutter: material/checkbox_list_tile.dart CheckboxListTile.new
type CheckboxListTileOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// MouseCursor is the cursor of a pointer over the checkbox.
	MouseCursor services.MouseCursor
	// ActiveColor is the color of the checkbox when it is selected.
	ActiveColor foundation.Opt[ui.Color]
	// FillColor is the color of the checkbox's fill, in each state.
	FillColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// CheckColor is the color of the check mark.
	CheckColor foundation.Opt[ui.Color]
	// HoverColor is the color of the tile when a pointer hovers over it.
	HoverColor foundation.Opt[ui.Color]
	// OverlayColor is the color of the checkbox's overlay, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashRadius is the radius of the checkbox's splash.
	SplashRadius foundation.Opt[float64]
	// MaterialTapTargetSize is the tap target size of the checkbox; the default
	// is shrinkWrap.
	MaterialTapTargetSize MaterialTapTargetSize
	// VisualDensity defines how compact the tile's layout is.
	VisualDensity foundation.Opt[VisualDensity]
	// FocusNode is the focus node of the tile.
	FocusNode *widgets.FocusNode
	// StatesController is the controller of the states of the tile.
	StatesController *widgets.WidgetStatesController
	// Autofocus is whether the tile focuses itself when it is first built.
	Autofocus bool
	// Shape is the shape of the tile.
	Shape painting.ShapeBorder
	// Side is the side of the checkbox's border.
	Side foundation.Opt[painting.BorderSide]
	// IsError gives the checkbox its error look.
	IsError bool
	// Enabled is whether the tile is interactive; the default is that it is when
	// onChanged is set.
	Enabled foundation.Opt[bool]
	// TileColor is the background color of the tile.
	TileColor foundation.Opt[ui.Color]
	// Title is the primary content of the tile.
	Title widgets.Widget
	// Subtitle is additional content displayed below the title.
	Subtitle widgets.Widget
	// IsThreeLine is whether the tile is intended to display three lines of text.
	IsThreeLine foundation.Opt[bool]
	// Dense is whether the tile is part of a vertically dense list.
	Dense foundation.Opt[bool]
	// Secondary is a widget to display on the side opposite to the checkbox.
	Secondary widgets.Widget
	// Selected is whether the tile looks selected.
	Selected bool
	// ControlAffinity is where to place the checkbox relative to the text; the
	// default is platform, which is trailing.
	ControlAffinity ListTileControlAffinity
	// ContentPadding is the tile's internal padding.
	ContentPadding painting.EdgeInsetsGeometry
	// Tristate makes the checkbox cycle through false, true and none.
	Tristate bool
	// CheckboxShape is the shape of the checkbox.
	CheckboxShape painting.OutlinedBorder
	// SelectedTileColor is the background color of a selected tile.
	SelectedTileColor foundation.Opt[ui.Color]
	// OnFocusChange is called when the focus of the tile changes.
	OnFocusChange func(bool)
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
	// CheckboxSemanticLabel is the semantic label of the checkbox.
	CheckboxSemanticLabel string
	// CheckboxScaleFactor scales the checkbox; the default is 1.
	CheckboxScaleFactor foundation.Opt[float64]
	// TitleAlignment defines how the leading and trailing widgets align against
	// the title.
	TitleAlignment ListTileTitleAlignment
	// InternalAddSemanticForOnTap gives the tile the semantics of a button.
	InternalAddSemanticForOnTap bool
}

// CheckboxListTile is a ListTile with a Checkbox.
//
// Flutter: material/checkbox_list_tile.dart CheckboxListTile
type CheckboxListTile struct {
	widgets.StatelessWidget
	value     foundation.Opt[bool]
	onChanged func(foundation.Opt[bool])
	o         CheckboxListTileOpts
	adaptive  bool
}

func newCheckboxListTile(value foundation.Opt[bool], onChanged func(foundation.Opt[bool]), adaptive bool, opts []CheckboxListTileOpts) *CheckboxListTile {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if !o.Tristate && value.IsNone() {
			panic(errors.New("CheckboxListTileValueNull", "the value of a CheckboxListTile that is not tristate can't be none"))
		}
		if o.IsThreeLine.OrElse(false) && o.Subtitle == nil {
			panic(errors.New("CheckboxListTileThreeLineNoSubtitle", "a three line list tile needs a subtitle"))
		}
	}
	return &CheckboxListTile{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: value, onChanged: onChanged, o: o, adaptive: adaptive}
}

// NewCheckboxListTile creates a list tile with a checkbox. A nil onChanged
// disables the tile. The tags of its failures are CheckboxListTileValueNull and
// CheckboxListTileThreeLineNoSubtitle.
//
// Flutter: material/checkbox_list_tile.dart CheckboxListTile.new
func NewCheckboxListTile(value foundation.Opt[bool], onChanged func(foundation.Opt[bool]), opts ...CheckboxListTileOpts) *CheckboxListTile {
	return newCheckboxListTile(value, onChanged, false, opts)
}

// NewCheckboxListTileAdaptive creates a list tile with a checkbox that adapts to
// the platform.
//
// Deviation: it is the same as NewCheckboxListTile (SPEC §12.10).
//
// Flutter: material/checkbox_list_tile.dart CheckboxListTile.adaptive
func NewCheckboxListTileAdaptive(value foundation.Opt[bool], onChanged func(foundation.Opt[bool]), opts ...CheckboxListTileOpts) *CheckboxListTile {
	return newCheckboxListTile(value, onChanged, true, opts)
}

func (w *CheckboxListTile) handleValueChange() {
	switch v, ok := w.value.Get(); {
	case ok && !v:
		w.onChanged(foundation.Some(true))
	case ok && v:
		if w.o.Tristate {
			w.onChanged(foundation.None[bool]())
		} else {
			w.onChanged(foundation.Some(false))
		}
	default:
		w.onChanged(foundation.Some(false))
	}
}

// Build builds the tile.
//
// Flutter: material/checkbox_list_tile.dart CheckboxListTile.build
func (w *CheckboxListTile) Build(bc widgets.BuildContext) widgets.Widget {
	o := w.o
	onChanged := w.onChanged
	if !o.Enabled.OrElse(true) {
		onChanged = nil
	}
	checkboxOpts := CheckboxOpts{
		MouseCursor: o.MouseCursor, ActiveColor: o.ActiveColor, FillColor: o.FillColor, CheckColor: o.CheckColor, HoverColor: o.HoverColor,
		OverlayColor: o.OverlayColor, SplashRadius: o.SplashRadius, Autofocus: o.Autofocus, Tristate: o.Tristate, Shape: o.CheckboxShape,
		Side: o.Side, IsError: o.IsError, SemanticLabel: o.CheckboxSemanticLabel,
		MaterialTapTargetSize: firstNonZero(o.MaterialTapTargetSize, MaterialTapTargetSizeShrinkWrap),
	}
	var checkbox widgets.Widget
	if w.adaptive {
		checkbox = NewCheckboxAdaptive(w.value, onChanged, checkboxOpts)
	} else {
		checkbox = NewCheckbox(w.value, onChanged, checkboxOpts)
	}
	control := widgets.Widget(widgets.NewExcludeFocus(checkbox))
	if f := o.CheckboxScaleFactor.OrElse(1); f != 1 {
		control = widgets.NewTransformScale(control, widgets.TransformScaleOpts{Scale: foundation.Some(f)})
	}
	listTileTheme := ListTileThemeOf(bc)
	affinity := firstNonZero(o.ControlAffinity, listTileTheme.ControlAffinity, ListTileControlAffinityPlatform)
	leading, trailing := o.Secondary, control
	if affinity == ListTileControlAffinityLeading {
		leading, trailing = control, o.Secondary
	}
	theme := ThemeOf(bc)
	checkboxTheme := CheckboxThemeOf(bc)
	var states widgets.WidgetStates
	if o.Selected {
		states = states.With(widgets.WidgetStateSelected)
	}
	var fromTheme foundation.Opt[ui.Color]
	if checkboxTheme.FillColor != nil {
		fromTheme = checkboxTheme.FillColor.Resolve(states)
	}
	effectiveActiveColor := firstOpt(o.ActiveColor, fromTheme, foundation.Some(theme.ColorScheme.Secondary()))
	var onTap func()
	if w.onChanged != nil {
		onTap = w.handleValueChange
	}
	return widgets.NewMergeSemantics(NewListTile(ListTileOpts{
		SelectedColor: effectiveActiveColor, Leading: leading, Title: o.Title, Subtitle: o.Subtitle, Trailing: trailing,
		IsThreeLine: o.IsThreeLine, Dense: o.Dense, Disabled: !o.Enabled.OrElse(w.onChanged != nil), OnTap: onTap, Selected: o.Selected,
		Autofocus: o.Autofocus, ContentPadding: o.ContentPadding, Shape: o.Shape, SelectedTileColor: o.SelectedTileColor,
		TileColor: o.TileColor, VisualDensity: o.VisualDensity, FocusNode: o.FocusNode, StatesController: o.StatesController,
		OnFocusChange: o.OnFocusChange, EnableFeedback: o.EnableFeedback, HorizontalTitleGap: o.HorizontalTitleGap,
		MinVerticalPadding: o.MinVerticalPadding, MinLeadingWidth: o.MinLeadingWidth, MinTileHeight: o.MinTileHeight,
		TitleAlignment: o.TitleAlignment, internalNoSemanticForOnTap: !o.InternalAddSemanticForOnTap,
	}))
}
