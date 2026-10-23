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

// RadioListTileOpts are the optional parameters of NewRadioListTile.
//
// Flutter: material/radio_list_tile.dart RadioListTile.new
type RadioListTileOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// GroupValue is the currently selected value for a group of radio tiles; a
	// RadioGroup above the tile is the way to manage the group.
	GroupValue foundation.Opt[T]
	// OnChanged is called when the user selects this tile; a RadioGroup above the
	// tile is the way to handle the change.
	OnChanged func(foundation.Opt[T])
	// MouseCursor is the cursor of a pointer over the radio.
	MouseCursor services.MouseCursor
	// Toggleable makes the radio deselectable by selecting it again.
	Toggleable bool
	// ActiveColor is the color of the radio when it is selected.
	ActiveColor foundation.Opt[ui.Color]
	// FillColor is the color of the radio's fill, in each state.
	FillColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// HoverColor is the color of the tile when a pointer hovers over it.
	HoverColor foundation.Opt[ui.Color]
	// OverlayColor is the color of the radio's overlay, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashRadius is the radius of the radio's splash.
	SplashRadius foundation.Opt[float64]
	// MaterialTapTargetSize is the tap target size of the radio; the default is
	// shrinkWrap.
	MaterialTapTargetSize MaterialTapTargetSize
	// Title is the primary content of the tile.
	Title widgets.Widget
	// Subtitle is additional content displayed below the title.
	Subtitle widgets.Widget
	// IsThreeLine is whether the tile is intended to display three lines of text.
	IsThreeLine foundation.Opt[bool]
	// Dense is whether the tile is part of a vertically dense list.
	Dense foundation.Opt[bool]
	// Secondary is a widget to display on the side opposite to the radio.
	Secondary widgets.Widget
	// Selected is whether the tile looks selected.
	Selected bool
	// ControlAffinity is where to place the radio relative to the text; the
	// default is platform, which is leading.
	ControlAffinity ListTileControlAffinity
	// Autofocus is whether the tile focuses itself when it is first built.
	Autofocus bool
	// ContentPadding is the tile's internal padding.
	ContentPadding painting.EdgeInsetsGeometry
	// Shape is the shape of the tile.
	Shape painting.ShapeBorder
	// TileColor is the background color of the tile.
	TileColor foundation.Opt[ui.Color]
	// SelectedTileColor is the background color of a selected tile.
	SelectedTileColor foundation.Opt[ui.Color]
	// VisualDensity defines how compact the tile's layout is.
	VisualDensity foundation.Opt[VisualDensity]
	// FocusNode is the focus node of the tile.
	FocusNode *widgets.FocusNode
	// StatesController is the controller of the states of the tile.
	StatesController *widgets.WidgetStatesController
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
	// RadioScaleFactor scales the radio; the default is 1.
	RadioScaleFactor foundation.Opt[float64]
	// TitleAlignment defines how the leading and trailing widgets align against
	// the title.
	TitleAlignment ListTileTitleAlignment
	// Enabled is whether the tile is interactive; the default is that it is when
	// onChanged is set or a RadioGroup is above it.
	Enabled foundation.Opt[bool]
	// InternalAddSemanticForOnTap gives the tile the semantics of a button.
	InternalAddSemanticForOnTap bool
	// RadioBackgroundColor is the background color of the radio, in each state.
	RadioBackgroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// RadioSide is the side of the radio's border.
	RadioSide foundation.Opt[painting.BorderSide]
	// RadioInnerRadius is the radius of the radio's inner circle, in each state.
	RadioInnerRadius widgets.WidgetStateProperty[foundation.Opt[float64]]
}

// RadioListTile is a ListTile with a Radio.
//
// Flutter: material/radio_list_tile.dart RadioListTile
type RadioListTile[T comparable] struct {
	widgets.StatefulWidget
	value    T
	o        RadioListTileOpts[T]
	adaptive bool
}

// NewRadioListTile creates a list tile with a radio for the value. The tag of
// its failure is RadioListTileThreeLineNoSubtitle.
//
// Flutter: material/radio_list_tile.dart RadioListTile.new
func NewRadioListTile[T comparable](value T, opts ...RadioListTileOpts[T]) *RadioListTile[T] {
	o := firstOpts(opts)
	if foundation.DebugMode && o.IsThreeLine.OrElse(false) && o.Subtitle == nil {
		panic(errors.New(tagRadioListTileThreeLineNoSubtitle, "a three line list tile needs a subtitle"))
	}
	return &RadioListTile[T]{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: value, o: o}
}

// NewRadioListTileAdaptive creates a list tile with a radio that adapts to the
// platform.
//
// Deviation: it is the same as NewRadioListTile (SPEC §12.10).
//
// Flutter: material/radio_list_tile.dart RadioListTile.adaptive
func NewRadioListTileAdaptive[T comparable](value T, opts ...RadioListTileOpts[T]) *RadioListTile[T] {
	w := NewRadioListTile(value, opts...)
	w.adaptive = true
	return w
}

// Checked is whether the tile's value is the group value.
//
// Flutter: material/radio_list_tile.dart RadioListTile.checked
func (w *RadioListTile[T]) Checked() bool { return w.o.GroupValue == foundation.Some(w.value) }

// CreateState creates the state.
//
// Flutter: material/radio_list_tile.dart RadioListTile.createState
func (w *RadioListTile[T]) CreateState() widgets.State { return &radioListTileState[T]{} }

type radioListTileState[T comparable] struct {
	widgets.StateOf[*RadioListTile[T]]
	widgets.RadioClientRegistration[T]
	internalFocusNode *widgets.FocusNode
	registry          *radioListTileRegistry[T]
}

// FocusNode is the focus node of the tile.
//
// Flutter: material/radio_list_tile.dart _RadioListTileState.focusNode
func (s *radioListTileState[T]) FocusNode() *widgets.FocusNode {
	if n := s.Widget().o.FocusNode; n != nil {
		return n
	}
	if s.internalFocusNode == nil {
		s.internalFocusNode = widgets.NewFocusNode()
	}
	return s.internalFocusNode
}

// RadioValue is the value of the tile.
//
// Flutter: material/radio_list_tile.dart _RadioListTileState.radioValue
func (s *radioListTileState[T]) RadioValue() T { return s.Widget().value }

// Tristate is whether the radio can be deselected.
//
// Flutter: material/radio_list_tile.dart _RadioListTileState.tristate
func (s *radioListTileState[T]) Tristate() bool { return s.Widget().o.Toggleable }

// Enabled is whether the tile is interactive.
//
// Flutter: material/radio_list_tile.dart _RadioListTileState.enabled
func (s *radioListTileState[T]) Enabled() bool {
	o := s.Widget().o
	return o.Enabled.OrElse(o.OnChanged != nil || s.Registry() != nil)
}

func (s *radioListTileState[T]) effectiveGroupValue() foundation.Opt[T] {
	if r := s.Registry(); r != nil {
		return firstOpt(r.GroupValue(), s.Widget().o.GroupValue)
	}
	return s.Widget().o.GroupValue
}

func (s *radioListTileState[T]) checked() bool {
	return s.effectiveGroupValue() == foundation.Some(s.Widget().value)
}

func (s *radioListTileState[T]) handleListTileTap() {
	if !s.Widget().o.Toggleable && s.checked() {
		return
	}
	if s.checked() {
		s.handleChange(foundation.None[T]())
		return
	}
	s.handleChange(foundation.Some(s.Widget().value))
}

func (s *radioListTileState[T]) handleChange(value foundation.Opt[T]) {
	if r := s.Registry(); r != nil {
		r.OnChanged(value)
	}
	if f := s.Widget().o.OnChanged; f != nil {
		f(value)
	}
}

// InitState creates the registry of the tile's radio.
//
// Flutter: material/radio_list_tile.dart _RadioListTileState
func (s *radioListTileState[T]) InitState() { s.registry = &radioListTileRegistry[T]{state: s} }

// DidChangeDependencies registers the tile with the RadioGroup above it.
//
// Flutter: material/radio_list_tile.dart _RadioListTileState.didChangeDependencies
func (s *radioListTileState[T]) DidChangeDependencies() {
	r, _ := widgets.RadioGroupMaybeOf[T](s.Context())
	s.SetRegistry(s, r)
}

// Dispose leaves the group and releases the focus node.
//
// Flutter: material/radio_list_tile.dart _RadioListTileState.dispose
func (s *radioListTileState[T]) Dispose() {
	s.SetRegistry(s, nil)
	if s.internalFocusNode != nil {
		s.internalFocusNode.Dispose()
	}
}

// Build builds the tile.
//
// Flutter: material/radio_list_tile.dart _RadioListTileState.build
func (s *radioListTileState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	if foundation.DebugMode && o.Enabled.OrElse(false) && o.OnChanged == nil {
		if _, ok := widgets.RadioGroupMaybeOf[T](bc); !ok {
			panic(errors.New(tagRadioListTileNoRegistry, "Radio is enabled but has no RadioListTile.onChange or registry above"))
		}
	}
	enabled := s.Enabled()
	radioOpts := RadioOpts[T]{
		GroupValue: s.registry.GroupValue(), Toggleable: o.Toggleable, ActiveColor: o.ActiveColor, FillColor: o.FillColor,
		MouseCursor: o.MouseCursor, HoverColor: o.HoverColor, OverlayColor: o.OverlayColor, SplashRadius: o.SplashRadius,
		MaterialTapTargetSize: firstNonZero(o.MaterialTapTargetSize, MaterialTapTargetSizeShrinkWrap), Autofocus: o.Autofocus,
		Enabled: foundation.Some(enabled), GroupRegistry: s.registry, BackgroundColor: o.RadioBackgroundColor, Side: o.RadioSide,
		InnerRadius: o.RadioInnerRadius,
	}
	var radio widgets.Widget
	if w.adaptive {
		radio = NewRadioAdaptive(w.value, radioOpts)
	} else {
		radio = NewRadio(w.value, radioOpts)
	}
	control := widgets.Widget(widgets.NewExcludeFocus(radio))
	if f := o.RadioScaleFactor.OrElse(1); f != 1 {
		control = widgets.NewTransformScale(control, widgets.TransformScaleOpts{Scale: foundation.Some(f)})
	}
	listTileTheme := ListTileThemeOf(bc)
	affinity := firstNonZero(o.ControlAffinity, listTileTheme.ControlAffinity, ListTileControlAffinityPlatform)
	leading, trailing := control, o.Secondary
	if affinity == ListTileControlAffinityTrailing {
		leading, trailing = o.Secondary, control
	}
	theme := ThemeOf(bc)
	radioTheme := RadioThemeOf(bc)
	var states widgets.WidgetStates
	if o.Selected {
		states = states.With(widgets.WidgetStateSelected)
	}
	var fromTheme foundation.Opt[ui.Color]
	if radioTheme.FillColor != nil {
		fromTheme = radioTheme.FillColor.Resolve(states)
	}
	effectiveActiveColor := firstOpt(o.ActiveColor, fromTheme, foundation.Some(theme.ColorScheme.Secondary()))
	var onTap func()
	if enabled {
		onTap = s.handleListTileTap
	}
	return widgets.NewMergeSemantics(NewListTile(ListTileOpts{
		SelectedColor: effectiveActiveColor, Leading: leading, Title: o.Title, Subtitle: o.Subtitle, Trailing: trailing,
		IsThreeLine: o.IsThreeLine, Dense: o.Dense, Disabled: !enabled, Shape: o.Shape, TileColor: o.TileColor,
		SelectedTileColor: o.SelectedTileColor, OnTap: onTap, Selected: o.Selected, Autofocus: o.Autofocus,
		ContentPadding: o.ContentPadding, VisualDensity: o.VisualDensity, FocusNode: s.FocusNode(), StatesController: o.StatesController,
		OnFocusChange: o.OnFocusChange, EnableFeedback: o.EnableFeedback, HorizontalTitleGap: o.HorizontalTitleGap,
		MinVerticalPadding: o.MinVerticalPadding, MinLeadingWidth: o.MinLeadingWidth, MinTileHeight: o.MinTileHeight,
		TitleAlignment: o.TitleAlignment, internalNoSemanticForOnTap: !o.InternalAddSemanticForOnTap,
	}))
}

// radioListTileRegistry is the registry of the radio of a tile: it gives the
// group value of the tile and sends the changes to it.
//
// Flutter: material/radio_list_tile.dart _RadioRegistry
type radioListTileRegistry[T comparable] struct{ state *radioListTileState[T] }

func (r *radioListTileRegistry[T]) GroupValue() foundation.Opt[T] {
	return r.state.effectiveGroupValue()
}
func (r *radioListTileRegistry[T]) RegisterClient(widgets.RadioClient[T])   {}
func (r *radioListTileRegistry[T]) UnregisterClient(widgets.RadioClient[T]) {}
func (r *radioListTileRegistry[T]) OnChanged(value foundation.Opt[T])       { r.state.handleChange(value) }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagRadioListTileNoRegistry          = "RadioListTileNoRegistry"
	tagRadioListTileThreeLineNoSubtitle = "RadioListTileThreeLineNoSubtitle"
)
