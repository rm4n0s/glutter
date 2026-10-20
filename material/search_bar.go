// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"reflect"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// searchBarDisabledOpacity is the opacity of a disabled search bar.
//
// Flutter: material/search_anchor.dart _kDisableSearchBarOpacity
const searchBarDisabledOpacity = 0.38

// SearchBarOpts are the optional parameters of NewSearchBar.
//
// Flutter: material/search_anchor.dart SearchBar.new
type SearchBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller controls the text that the user types; the default is the one of
	// the text field.
	Controller *widgets.TextEditingController
	// FocusNode is the focus node of the text field.
	FocusNode *widgets.FocusNode
	// HintText is the text that suggests what to search.
	HintText foundation.Opt[string]
	// Leading is the widget before the text field, typically an icon.
	Leading widgets.Widget
	// Trailing are the widgets after the text field, typically icon buttons.
	Trailing []widgets.Widget
	// OnTap is called when the user taps the search bar.
	OnTap func()
	// OnTapOutside is called when the user taps outside of the search bar.
	OnTapOutside widgets.TapRegionCallback
	// OnChanged is called when the text changes.
	OnChanged func(text string)
	// OnSubmitted is called when the user submits the text.
	OnSubmitted func(text string)
	// Constraints are the size constraints of the search bar.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Elevation is the elevation of the search bar, in each state.
	Elevation widgets.WidgetStateProperty[foundation.Opt[float64]]
	// BackgroundColor is the color of the background, in each state.
	BackgroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// ShadowColor is the color of the shadow, in each state.
	ShadowColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SurfaceTintColor is the color of the surface tint, in each state.
	SurfaceTintColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// OverlayColor is the color of the overlay, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// Side is the border of the search bar, in each state.
	Side widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// Shape is the shape of the search bar, in each state; the default is a stadium.
	Shape widgets.WidgetStateProperty[painting.OutlinedBorder]
	// Padding is the padding around the contents, in each state.
	Padding widgets.WidgetStateProperty[painting.EdgeInsetsGeometry]
	// TextStyle is the style of the text that the user types, in each state.
	TextStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// HintStyle is the style of the hint text, in each state.
	HintStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// TextCapitalization is how the input method capitalizes the text.
	TextCapitalization services.TextCapitalization
	// Disabled makes the search bar not react to the user, and fade.
	Disabled bool
	// AutoFocus takes the focus when the search bar is first built.
	AutoFocus bool
	// TextInputAction is the action key of the keyboard.
	TextInputAction services.TextInputAction
	// KeyboardType is the type of the keyboard.
	KeyboardType foundation.Opt[services.TextInputType]
	// ScrollPadding is the space kept around the caret when the field scrolls; the
	// default is 20 on every side.
	ScrollPadding foundation.Opt[painting.EdgeInsets]
	// ContextMenuBuilder builds the context menu of the text field; the default is
	// the toolbar of the platform.
	ContextMenuBuilder widgets.EditableTextContextMenuBuilder
	// ReadOnly makes the text field not editable.
	ReadOnly bool
	// SmartDashesType says whether the dashes are replaced.
	SmartDashesType services.SmartDashesType
	// SmartQuotesType says whether the quotes are replaced.
	SmartQuotesType services.SmartQuotesType
}

// SearchBar is a Material Design search bar: a text field with a leading and
// trailing widgets in a rounded container.
//
// Flutter: material/search_anchor.dart SearchBar
type SearchBar struct {
	widgets.StatefulWidget
	o SearchBarOpts
}

// NewSearchBar creates a search bar.
//
// Flutter: material/search_anchor.dart SearchBar.new
func NewSearchBar(opts ...SearchBarOpts) *SearchBar {
	o := firstOpts(opts)
	return &SearchBar{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// CreateState creates the state.
//
// Flutter: material/search_anchor.dart SearchBar.createState
func (w *SearchBar) CreateState() widgets.State { return &searchBarState{} }

type searchBarState struct {
	widgets.StateOf[*SearchBar]
	statesController *widgets.WidgetStatesController
	statesHandle     foundation.ListenerHandle
	internalFocus    *widgets.FocusNode
}

func (s *searchBarState) focusNode() *widgets.FocusNode {
	if s.Widget().o.FocusNode != nil {
		return s.Widget().o.FocusNode
	}
	if s.internalFocus == nil {
		s.internalFocus = widgets.NewFocusNode()
	}
	return s.internalFocus
}

func (s *searchBarState) InitState() {
	s.statesController = widgets.NewWidgetStatesController()
	s.statesHandle = s.statesController.AddListener(func() { s.SetState(func() {}) })
}

func (s *searchBarState) Dispose() {
	s.statesController.RemoveListener(s.statesHandle)
	s.statesController.Dispose()
	if s.internalFocus != nil {
		s.internalFocus.Dispose()
	}
}

// isNilValue says whether the value is a nil pointer, interface, function, slice
// or map.
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Slice, reflect.Map:
		return rv.IsNil()
	}
	return false
}

// firstResolvedOpt resolves the properties for the states, and returns the first
// value that is set.
func firstResolvedOpt[T any](states widgets.WidgetStates, props ...widgets.WidgetStateProperty[foundation.Opt[T]]) foundation.Opt[T] {
	for _, p := range props {
		if p == nil {
			continue
		}
		if v := p.Resolve(states); v.IsSome() {
			return v
		}
	}
	return foundation.None[T]()
}

// firstResolvedValue resolves the properties for the states, and returns the first
// value that is not nil (Dart's `a?.resolve(states) ?? b?.resolve(states)`).
func firstResolvedValue[T any](states widgets.WidgetStates, props ...widgets.WidgetStateProperty[T]) T {
	var zero T
	for _, p := range props {
		if p == nil {
			continue
		}
		if v := p.Resolve(states); !isNilValue(v) {
			return v
		}
	}
	return zero
}

// outlinedWithSide is `shape?.copyWith(side: side)`: a nil shape stays nil and a
// side that is not set keeps the side of the shape.
func outlinedWithSide(shape painting.OutlinedBorder, side foundation.Opt[painting.BorderSide]) painting.OutlinedBorder {
	if isNilValue(shape) {
		return nil
	}
	if v, ok := side.Get(); ok {
		return shape.CopyWithSide(v)
	}
	return shape
}

// Build builds the search bar.
//
// Flutter: material/search_anchor.dart _SearchBarState.build
func (s *searchBarState) Build(bc widgets.BuildContext) widgets.Widget {
	o := s.Widget().o
	textDirection := widgets.DirectionalityOf(bc)
	colorScheme := ThemeOf(bc).ColorScheme
	searchBarTheme := SearchBarThemeOf(bc)
	defaults := searchBarDefaultsM3(bc)
	states := s.statesController.Value()

	effectiveTextStyle := firstResolvedValue(states, o.TextStyle, searchBarTheme.TextStyle, defaults.TextStyle)
	effectiveElevation := firstResolvedOpt(states, o.Elevation, searchBarTheme.Elevation, defaults.Elevation).Value()
	effectiveShadowColor := firstResolvedOpt(states, o.ShadowColor, searchBarTheme.ShadowColor, defaults.ShadowColor)
	effectiveBackgroundColor := firstResolvedOpt(states, o.BackgroundColor, searchBarTheme.BackgroundColor, defaults.BackgroundColor)
	effectiveSurfaceTintColor := firstResolvedOpt(states, o.SurfaceTintColor, searchBarTheme.SurfaceTintColor, defaults.SurfaceTintColor)
	effectiveShape := firstResolvedValue(states, o.Shape, searchBarTheme.Shape, defaults.Shape)
	effectiveSide := firstResolvedOpt(states, o.Side, searchBarTheme.Side, defaults.Side)
	effectivePadding := firstResolvedValue(states, o.Padding, searchBarTheme.Padding, defaults.Padding)
	effectiveOverlayColor := firstStateColor(o.OverlayColor, searchBarTheme.OverlayColor, defaults.OverlayColor)
	effectiveTextCapitalization := o.TextCapitalization
	if effectiveTextCapitalization == services.TextCapitalizationUnspecified {
		effectiveTextCapitalization = searchBarTheme.TextCapitalization
	}
	effectiveHintStyle := firstResolvedValue(states, o.HintStyle, searchBarTheme.HintStyle, o.TextStyle, searchBarTheme.TextStyle, defaults.HintStyle)

	defaultColor := DefaultIconDarkColor
	if colorScheme.Brightness() == ui.BrightnessDark {
		defaultColor = DefaultIconLightColor
	}
	var customTheme foundation.Opt[widgets.IconThemeData]
	if it := widgets.IconThemeOf(bc); it.Color() != foundation.Some(defaultColor) {
		customTheme = foundation.Some(it)
	}
	iconData := func(color ui.Color) widgets.IconThemeData {
		return customTheme.OrElse(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(color)}))
	}

	var leading widgets.Widget
	if o.Leading != nil {
		leading = widgets.NewIconThemeMerge(iconData(colorScheme.OnSurface()), o.Leading)
	}
	trailing := make([]widgets.Widget, len(o.Trailing))
	for i, t := range o.Trailing {
		trailing[i] = widgets.NewIconThemeMerge(iconData(colorScheme.OnSurfaceVariant()), t)
	}

	enabled := !o.Disabled
	constraints := firstOpt(o.Constraints, searchBarTheme.Constraints, defaults.Constraints).Value()
	opacity := 1.0
	if !enabled {
		opacity = searchBarDisabledOpacity
	}
	shape := outlinedWithSide(effectiveShape, effectiveSide)
	var customBorder painting.ShapeBorder
	if !isNilValue(shape) {
		customBorder = shape
	}

	row := make([]widgets.Widget, 0, 2+len(trailing))
	if leading != nil {
		row = append(row, leading)
	}
	scrollPadding := o.ScrollPadding
	if scrollPadding.IsNone() {
		scrollPadding = foundation.Some(painting.NewEdgeInsetsAll(20.0))
	}
	contextMenu := o.ContextMenuBuilder
	if contextMenu == nil {
		contextMenu = func(_ widgets.BuildContext, state *widgets.EditableTextState) widgets.Widget {
			return NewAdaptiveTextSelectionToolbarEditableText(state)
		}
	}
	decoration := NewInputDecoration(InputDecorationOpts{HintText: o.HintText}).ApplyDefaults(NewInputDecorationThemeData(InputDecorationThemeDataOpts{
		HintStyle: widgets.NewWidgetStatePropertyAll(effectiveHintStyle), EnabledBorder: InputBorderNone, Border: InputBorderNone,
		FocusedBorder: InputBorderNone, ContentPadding: painting.EdgeInsetsZero, IsDense: true}))
	row = append(row, widgets.NewExpanded(widgets.NewPadding(effectivePadding,
		widgets.NewSemantics(NewTextField(TextFieldOpts{
			ReadOnly: o.ReadOnly, Autofocus: o.AutoFocus, OnTap: o.OnTap, OnTapAlwaysCalled: true, OnTapOutside: o.OnTapOutside,
			FocusNode: s.focusNode(), OnChanged: o.OnChanged, OnSubmitted: o.OnSubmitted, Controller: o.Controller,
			Style: effectiveTextStyle, Enabled: foundation.Some(enabled), Decoration: &decoration,
			TextCapitalization: effectiveTextCapitalization, TextInputAction: o.TextInputAction, KeyboardType: o.KeyboardType,
			ScrollPadding: scrollPadding, ContextMenuBuilder: contextMenu, SmartDashesType: o.SmartDashesType,
			SmartQuotesType: o.SmartQuotesType,
		}), widgets.SemanticsOpts{InputType: foundation.Some(ui.SemanticsInputTypeSearch)}))))
	row = append(row, trailing...)

	onTap := func() {
		if o.OnTap != nil {
			o.OnTap()
		}
		if !s.focusNode().HasFocus() {
			s.focusNode().RequestFocus()
		}
	}
	return widgets.NewConstrainedBox(constraints, widgets.NewOpacity(opacity, NewMaterial(
		widgets.NewIgnorePointer(NewInkWell(
			widgets.NewPadding(effectivePadding, widgets.NewRow(row, widgets.RowOpts{TextDirection: textDirection})),
			InkWellOpts{OnTap: onTap, OverlayColor: effectiveOverlayColor, CustomBorder: customBorder, StatesController: s.statesController}),
			widgets.IgnorePointerOpts{Ignoring: foundation.Some(!enabled)}),
		MaterialOpts{Elevation: effectiveElevation, ShadowColor: effectiveShadowColor, Color: effectiveBackgroundColor,
			SurfaceTintColor: effectiveSurfaceTintColor, Shape: customBorder})))
}
