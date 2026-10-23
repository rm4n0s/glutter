// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ButtonSegment describes a segment of a SegmentedButton: the value that it
// stands for, and its icon and label.
//
// Flutter: material/segmented_button.dart ButtonSegment
type ButtonSegment[T comparable] struct {
	// Value is the value that identifies the segment; it is unique in the button.
	Value T
	// Icon is the icon of the segment; with a label it is shown before it, and
	// replaced by the selected icon when the segment is selected.
	Icon widgets.Widget
	// Label is the label of the segment.
	Label widgets.Widget
	// Tooltip is the tooltip of the segment.
	Tooltip string
	// Disabled says that the segment cannot be pressed (Flutter's enabled: false).
	Disabled bool
}

// SegmentedButtonOpts are the optional parameters of NewSegmentedButton.
//
// Flutter: material/segmented_button.dart SegmentedButton.new
type SegmentedButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// MultiSelectionEnabled says that more than one segment can be selected.
	MultiSelectionEnabled bool
	// EmptySelectionAllowed says that no segment can be selected.
	EmptySelectionAllowed bool
	// ExpandedInsets makes the button as wide as the space it has, less the insets.
	ExpandedInsets foundation.Opt[painting.EdgeInsets]
	// Style customizes the appearance of the button; the segments are TextButtons.
	Style *ButtonStyle
	// NoShowSelectedIcon says that the selected segments have no selected icon
	// (Flutter's showSelectedIcon: false).
	NoShowSelectedIcon bool
	// SelectedIcon is the icon of the selected segments; the default is a check mark.
	SelectedIcon widgets.Widget
	// Direction is the axis along which the segments are laid out; the default
	// is horizontal.
	Direction painting.Axis
}

// SegmentedButton is a Material Design segmented button: a row of segments, of
// which one (or more with multi-selection) is selected, for choosing between
// two to five options.
//
// Flutter: material/segmented_button.dart SegmentedButton
type SegmentedButton[T comparable] struct {
	widgets.StatefulWidget
	segments           []ButtonSegment[T]
	selected           []T
	onSelectionChanged func(selected []T)
	o                  SegmentedButtonOpts
}

// NewSegmentedButton creates a segmented button of the segments, with the
// selected values. A nil onSelectionChanged disables the button. In debug builds
// it panics with the tags SegmentedButtonNoSegments, SegmentedButtonNoSelection
// and SegmentedButtonMultipleSelected when there are no segments, nothing is
// selected without EmptySelectionAllowed, or several are selected without
// MultiSelectionEnabled.
//
// Flutter: material/segmented_button.dart SegmentedButton.new
func NewSegmentedButton[T comparable](segments []ButtonSegment[T], selected []T, onSelectionChanged func(selected []T), opts ...SegmentedButtonOpts) *SegmentedButton[T] {
	o := firstOpts(opts)
	if foundation.DebugMode {
		switch {
		case len(segments) == 0:
			panic(errors.New(tagSegmentedButtonNoSegments, "a SegmentedButton needs at least one segment"))
		case len(selected) == 0 && !o.EmptySelectionAllowed:
			panic(errors.New(tagSegmentedButtonNoSelection, "a SegmentedButton needs a selected segment unless EmptySelectionAllowed is set"))
		case len(selected) >= 2 && !o.MultiSelectionEnabled:
			panic(errors.New(tagSegmentedButtonMultipleSelected, "a SegmentedButton can have several selected segments only with MultiSelectionEnabled"))
		}
		for _, s := range segments {
			if s.Icon == nil && s.Label == nil {
				panic(errors.New(tagButtonSegmentEmpty, "a ButtonSegment needs an icon or a label", "value", s.Value))
			}
		}
	}
	if o.Direction == painting.AxisUnspecified {
		o.Direction = painting.AxisHorizontal
	}
	return &SegmentedButton[T]{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), segments: segments, selected: selected,
		onSelectionChanged: onSelectionChanged, o: o}
}

// Segments are the segments of the button.
//
// Flutter: material/segmented_button.dart SegmentedButton.segments
func (w *SegmentedButton[T]) Segments() []ButtonSegment[T] { return w.segments }

// Selected are the values of the selected segments.
//
// Flutter: material/segmented_button.dart SegmentedButton.selected
func (w *SegmentedButton[T]) Selected() []T { return w.selected }

// OnSelectionChanged is called when the selection changes; nil disables the button.
//
// Flutter: material/segmented_button.dart SegmentedButton.onSelectionChanged
func (w *SegmentedButton[T]) OnSelectionChanged() func([]T) { return w.onSelectionChanged }

// SegmentedButtonStyleFromOpts are the parameters of SegmentedButtonStyleFrom: those of
// TextButtonStyleFrom, plus the colors of the selected segments.
//
// Flutter: material/segmented_button.dart SegmentedButton.styleFrom
type SegmentedButtonStyleFromOpts struct {
	ButtonStyleFromOpts
	// SelectedForegroundColor is the foreground color of the selected segments.
	SelectedForegroundColor foundation.Opt[ui.Color]
	// SelectedBackgroundColor is the background color of the selected segments.
	SelectedBackgroundColor foundation.Opt[ui.Color]
}

// SegmentedButtonStyleFrom is a static convenience that makes a ButtonStyle for
// a segmented button from simple values.
//
// Flutter: material/segmented_button.dart SegmentedButton.styleFrom
func SegmentedButtonStyleFrom(o SegmentedButtonStyleFromOpts) *ButtonStyle {
	var overlay widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	if o.ForegroundColor.IsSome() || o.SelectedForegroundColor.IsSome() || o.OverlayColor.IsSome() {
		if oc, ok := o.OverlayColor.Get(); ok && oc.ToARGB32() == 0 {
			overlay = allOpt(Colors.Transparent)
		} else {
			overlay = segmentedButtonResolveStateColor(o.ForegroundColor, o.SelectedForegroundColor, o.OverlayColor)
		}
	}
	base := o.ButtonStyleFromOpts
	foreground, background, disabledForeground, disabledBackground := base.ForegroundColor, base.BackgroundColor, base.DisabledForegroundColor, base.DisabledBackgroundColor
	base.ForegroundColor, base.BackgroundColor, base.DisabledForegroundColor, base.DisabledBackgroundColor, base.OverlayColor = foundation.None[ui.Color](), foundation.None[ui.Color](),
		foundation.None[ui.Color](), foundation.None[ui.Color](), foundation.None[ui.Color]()
	base.IconAlignment = IconAlignmentUnspecified
	s := TextButtonStyleFrom(base)
	return s.CopyWith(ButtonStyleOpts{
		ForegroundColor: segmentedButtonDefaultColor(foreground, disabledForeground, o.SelectedForegroundColor),
		BackgroundColor: segmentedButtonDefaultColor(background, disabledBackground, o.SelectedBackgroundColor),
		OverlayColor:    overlay,
	})
}

// Flutter: material/segmented_button.dart SegmentedButton._defaultColor
func segmentedButtonDefaultColor(enabled, disabled, selected foundation.Opt[ui.Color]) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	if firstOpt(selected, enabled, disabled).IsNone() {
		return nil
	}
	return widgets.NewWidgetStateMapper(widgets.WidgetStateMap[foundation.Opt[ui.Color]]{
		{Key: widgets.WidgetStateDisabled, Value: disabled},
		{Key: widgets.WidgetStateSelected, Value: selected},
		{Key: widgets.WidgetStateAny, Value: enabled},
	})
}

// Flutter: material/segmented_button.dart _SegmentedButtonDefaultsM3.resolveStateColor
func segmentedButtonResolveStateColor(unselectedColor, selectedColor, overlayColor foundation.Opt[ui.Color]) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	selected := firstOpt(overlayColor, selectedColor)
	unselected := firstOpt(overlayColor, unselectedColor)
	withOpacity := func(c foundation.Opt[ui.Color], opacity float64) foundation.Opt[ui.Color] {
		if v, ok := c.Get(); ok {
			return foundation.Some(v.WithOpacity(opacity))
		}
		return c
	}
	return widgets.NewWidgetStateMapper(widgets.WidgetStateMap[foundation.Opt[ui.Color]]{
		{Key: widgets.WidgetStatesAnd(widgets.WidgetStateSelected, widgets.WidgetStatePressed), Value: withOpacity(selected, 0.1)},
		{Key: widgets.WidgetStatesAnd(widgets.WidgetStateSelected, widgets.WidgetStateHovered), Value: withOpacity(selected, 0.08)},
		{Key: widgets.WidgetStatesAnd(widgets.WidgetStateSelected, widgets.WidgetStateFocused), Value: withOpacity(selected, 0.1)},
		{Key: widgets.WidgetStatePressed, Value: withOpacity(unselected, 0.1)},
		{Key: widgets.WidgetStateHovered, Value: withOpacity(unselected, 0.08)},
		{Key: widgets.WidgetStateFocused, Value: withOpacity(unselected, 0.1)},
		{Key: widgets.WidgetStateAny, Value: foundation.Some(Colors.Transparent)},
	})
}

// CreateState creates the state.
//
// Flutter: material/segmented_button.dart SegmentedButton.createState
func (w *SegmentedButton[T]) CreateState() widgets.State { return &SegmentedButtonState[T]{} }

// SegmentedButtonState is the state of a SegmentedButton.
//
// Flutter: material/segmented_button.dart SegmentedButtonState
type SegmentedButtonState[T comparable] struct {
	widgets.StateOf[*SegmentedButton[T]]
	hovering bool
	focused  bool
	// statesControllers are the states controllers of the segments, by value.
	statesControllers map[T]*widgets.WidgetStatesController
}

func (s *SegmentedButtonState[T]) enabled() bool { return s.Widget().onSelectionChanged != nil }

func (s *SegmentedButtonState[T]) states() widgets.WidgetStates {
	var st widgets.WidgetStates
	if !s.enabled() {
		st = st.With(widgets.WidgetStateDisabled)
	}
	if s.hovering {
		st = st.With(widgets.WidgetStateHovered)
	}
	if s.focused {
		st = st.With(widgets.WidgetStateFocused)
	}
	if len(s.Widget().selected) > 0 {
		st = st.With(widgets.WidgetStateSelected)
	}
	return st
}

// StatesControllers are the controllers of the widget states of the segments,
// by the value of the segment.
//
// Flutter: material/segmented_button.dart SegmentedButtonState.statesControllers
func (s *SegmentedButtonState[T]) StatesControllers() map[T]*widgets.WidgetStatesController {
	return s.statesControllers
}

// InitState makes the map of the states controllers.
//
// Flutter: material/segmented_button.dart SegmentedButtonState.initState
func (s *SegmentedButtonState[T]) InitState() {
	s.statesControllers = map[T]*widgets.WidgetStatesController{}
}

// DidUpdateWidget disposes the states controllers of the segments that went.
//
// Flutter: material/segmented_button.dart SegmentedButtonState.didUpdateWidget
func (s *SegmentedButtonState[T]) DidUpdateWidget(old *SegmentedButton[T]) {
	w := s.Widget()
	for value, controller := range s.statesControllers {
		if !slices.ContainsFunc(w.segments, func(seg ButtonSegment[T]) bool { return seg.Value == value }) {
			controller.Dispose()
			delete(s.statesControllers, value)
		}
	}
}

func (s *SegmentedButtonState[T]) handleOnPressed(segmentValue T) {
	if !s.enabled() {
		return
	}
	w := s.Widget()
	contains := slices.Contains(w.selected, segmentValue)
	onlySelectedSegment := len(w.selected) == 1 && contains
	validChange := w.o.EmptySelectionAllowed || !onlySelectedSegment
	if !validChange {
		return
	}
	toggle := w.o.MultiSelectionEnabled || (w.o.EmptySelectionAllowed && onlySelectedSegment)
	var updatedSelection []T
	switch {
	case toggle && contains:
		for _, v := range w.selected {
			if v != segmentValue {
				updatedSelection = append(updatedSelection, v)
			}
		}
	case toggle:
		updatedSelection = append(append(updatedSelection, w.selected...), segmentValue)
	default:
		updatedSelection = []T{segmentValue}
	}
	if !sameSet(updatedSelection, w.selected) {
		w.onSelectionChanged(updatedSelection)
	}
}

func sameSet[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for _, v := range a {
		if !slices.Contains(b, v) {
			return false
		}
	}
	return true
}

// segmentedButtonStyleFor is the part of a style that the segments take.
//
// Flutter: material/segmented_button.dart SegmentedButtonState.build segmentStyleFor
func segmentedButtonStyleFor(style *ButtonStyle) *ButtonStyle {
	out := &ButtonStyle{Shape: widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](painting.NewRoundedRectangleBorder())}
	if style != nil {
		out.TextStyle, out.BackgroundColor, out.ForegroundColor, out.OverlayColor = style.TextStyle, style.BackgroundColor, style.ForegroundColor, style.OverlayColor
		out.SurfaceTintColor, out.Elevation, out.Padding, out.IconColor, out.IconSize = style.SurfaceTintColor, style.Elevation, style.Padding, style.IconColor, style.IconSize
		out.MouseCursor, out.VisualDensity, out.TapTargetSize, out.AnimationDuration = style.MouseCursor, style.VisualDensity, style.TapTargetSize, style.AnimationDuration
		out.EnableFeedback, out.Alignment, out.SplashFactory = style.EnableFeedback, style.Alignment, style.SplashFactory
	}
	return out
}

// Build builds the segments as TextButtons in a render object that paints the
// border over them.
//
// Flutter: material/segmented_button.dart SegmentedButtonState.build
func (s *SegmentedButtonState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	theme := SegmentedButtonThemeOf(bc)
	defaults := segmentedButtonDefaultsM3(bc)
	textDirection := widgets.DirectionalityOf(bc)
	themeData := ThemeOf(bc)
	styles := [3]*ButtonStyle{w.o.Style, theme.Style, defaults.Style}
	states := s.states()
	disabledState := widgets.NewWidgetStates(widgets.WidgetStateDisabled)

	segmentStyle := segmentedButtonStyleFor(w.o.Style)
	segmentThemeStyle := segmentedButtonStyleFor(theme.Style).Merge(segmentedButtonStyleFor(defaults.Style))
	var selectedIcon widgets.Widget
	if !w.o.NoShowSelectedIcon {
		for _, icon := range []widgets.Widget{w.o.SelectedIcon, theme.SelectedIcon, defaults.SelectedIcon} {
			if icon != nil {
				selectedIcon = icon
				break
			}
		}
	}

	buttonFor := func(segment ButtonSegment[T]) widgets.Widget {
		label := segment.Label
		if label == nil {
			label = segment.Icon
		}
		if label == nil {
			label = widgets.NewSizedBoxShrink(nil)
		}
		segmentSelected := slices.Contains(w.selected, segment.Value)
		var icon widgets.Widget
		switch {
		case segmentSelected && !w.o.NoShowSelectedIcon:
			icon = selectedIcon
		case segment.Label != nil:
			icon = segment.Icon
		}
		controller, ok := s.statesControllers[segment.Value]
		if !ok {
			controller = widgets.NewWidgetStatesController()
			s.statesControllers[segment.Value] = controller
		}
		controller.Update(widgets.WidgetStateSelected, segmentSelected)

		content := label
		effectiveSegmentStyle := segmentStyle
		if icon != nil {
			effectiveTextScale := buttonEffectiveTextScale(bc, segmentStyle)
			var one painting.EdgeInsetsGeometry = painting.NewEdgeInsetsAll(8)
			if themeData.UseMaterial3 {
				one = painting.NewEdgeInsetsDirectionalFromSTEB(12, 8, 16, 8)
			}
			scaledPadding := ButtonStyleButtonScaledPadding(one, painting.NewEdgeInsetsSymmetric(0, 4), painting.NewEdgeInsetsSymmetric(0, 4), effectiveTextScale)
			effectiveSegmentStyle = segmentStyle.CopyWith(ButtonStyleOpts{Padding: widgets.NewWidgetStatePropertyAll(scaledPadding)})
			scale := ui.ClampDouble(effectiveTextScale, 1.0, 2.0) - 1.0
			effectiveIconAlignment := IconAlignmentUnspecified
			if ts := TextButtonThemeOf(bc).Style; ts != nil {
				effectiveIconAlignment = ts.IconAlignment
			}
			if effectiveIconAlignment == IconAlignmentUnspecified {
				effectiveIconAlignment = segmentStyle.IconAlignment
			}
			children := []widgets.Widget{icon, widgets.NewFlexible(label)}
			if effectiveIconAlignment == IconAlignmentEnd {
				children = []widgets.Widget{widgets.NewFlexible(label), icon}
			}
			content = widgets.NewRow(children, widgets.RowOpts{MainAxisSize: rendering.MainAxisSizeMin, Spacing: ui.LerpDouble(8, 4, scale)})
		}

		var onPressed func()
		if s.enabled() && !segment.Disabled {
			value := segment.Value
			onPressed = func() { s.handleOnPressed(value) }
		}
		var button widgets.Widget = NewTextButton(onPressed, content, TextButtonOpts{
			Style:            effectiveSegmentStyle,
			StatesController: controller,
			OnHover:          func(hovering bool) { s.SetState(func() { s.hovering = hovering }) },
			OnFocusChange:    func(focused bool) { s.SetState(func() { s.focused = focused }) },
		})
		if segment.Tooltip != "" {
			button = NewTooltip(button, TooltipOpts{Message: segment.Tooltip})
		}
		semantics := widgets.SemanticsOpts{Selected: foundation.Some(segmentSelected)}
		if !w.o.MultiSelectionEnabled {
			semantics.InMutuallyExclusiveGroup = foundation.Some(true)
		}
		return widgets.NewMergeSemantics(widgets.NewSemantics(button, semantics))
	}

	resolveShape := func(st widgets.WidgetStates) painting.OutlinedBorder {
		if b := buttonStyleNil(styles, st, func(x *ButtonStyle) widgets.WidgetStateProperty[painting.OutlinedBorder] { return x.Shape }); b != nil {
			return b
		}
		return painting.NewRoundedRectangleBorder()
	}
	resolveSide := func(st widgets.WidgetStates) painting.BorderSide {
		return buttonStyleOpt(styles, st, func(x *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]] { return x.Side }).OrElse(painting.BorderSideNone)
	}
	enabledBorder := resolveShape(states).CopyWithSide(resolveSide(states))
	disabledBorder := resolveShape(disabledState).CopyWithSide(resolveSide(disabledState))
	resolvedVisualDensity := firstOpt(segmentStyle.VisualDensity, segmentThemeStyle.VisualDensity, foundation.Some(themeData.VisualDensity)).Value()
	var resolvedPadding painting.EdgeInsetsGeometry = painting.EdgeInsetsZero
	if p := buttonStyleNil(styles, states, func(x *ButtonStyle) widgets.WidgetStateProperty[painting.EdgeInsetsGeometry] { return x.Padding }); p != nil {
		resolvedPadding = p
	}
	resolvedTapTargetSize := segmentStyle.TapTargetSize
	if resolvedTapTargetSize == MaterialTapTargetSizeUnspecified {
		resolvedTapTargetSize = segmentThemeStyle.TapTargetSize
	}
	if resolvedTapTargetSize == MaterialTapTargetSizeUnspecified {
		resolvedTapTargetSize = themeData.MaterialTapTargetSize
	}
	fontSize := 20.0
	if ts := buttonStyleNil(styles, states, func(x *ButtonStyle) widgets.WidgetStateProperty[*painting.TextStyle] { return x.TextStyle }); ts != nil {
		fontSize = ts.FontSize.OrElse(20.0)
	}

	buttons := make([]widgets.Widget, len(w.segments))
	for i, seg := range w.segments {
		buttons[i] = buttonFor(seg)
	}

	densityAdjustment := resolvedVisualDensity.BaseSizeAdjustment()
	const textButtonMinHeight = 40.0
	adjustButtonMinHeight := textButtonMinHeight + densityAdjustment.Dy
	effectiveVerticalPadding := resolvedPadding.Vertical() + densityAdjustment.Dy*2
	effectedButtonHeight := math.Max(fontSize+effectiveVerticalPadding, adjustButtonMinHeight)
	tapTargetVerticalPadding := 0.0
	if resolvedTapTargetSize == MaterialTapTargetSizePadded {
		tapTargetVerticalPadding = math.Max(0, MinInteractiveDimension+densityAdjustment.Dy-effectedButtonHeight)
	}

	border := enabledBorder
	if !s.enabled() {
		border = disabledBorder
	}
	var padding painting.EdgeInsetsGeometry = painting.EdgeInsetsZero
	if in, ok := w.o.ExpandedInsets.Get(); ok {
		padding = in
	}
	return NewMaterial(
		NewTextButtonTheme(TextButtonThemeData{Style: segmentThemeStyle},
			widgets.NewPadding(padding, &segmentedButtonRenderWidget[T]{
				MultiChildRenderObjectWidget: widgets.NewMultiChildRenderObjectWidgetBase(buttons),
				segments:                     w.segments, enabledBorder: border, disabledBorder: disabledBorder, direction: w.o.Direction,
				textDirection: textDirection, tapTargetVerticalPadding: tapTargetVerticalPadding, isExpanded: w.o.ExpandedInsets.IsSome()})),
		MaterialOpts{
			Type:             MaterialTypeTransparency,
			Elevation:        buttonStyleOpt(styles, states, func(x *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[float64]] { return x.Elevation }).OrElse(0),
			ShadowColor:      buttonStyleOpt(styles, states, func(x *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] { return x.ShadowColor }),
			SurfaceTintColor: buttonStyleOpt(styles, states, func(x *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] { return x.SurfaceTintColor }),
		})
}

// Dispose disposes the states controllers.
//
// Flutter: material/segmented_button.dart SegmentedButtonState.dispose
func (s *SegmentedButtonState[T]) Dispose() {
	for _, c := range s.statesControllers {
		c.Dispose()
	}
}

// segmentedButtonRenderWidget is the widget of the render object that lays the
// segments out and paints the border over them.
//
// Flutter: material/segmented_button.dart _SegmentedButtonRenderWidget
type segmentedButtonRenderWidget[T comparable] struct {
	widgets.MultiChildRenderObjectWidget
	segments                 []ButtonSegment[T]
	enabledBorder            painting.OutlinedBorder
	disabledBorder           painting.OutlinedBorder
	direction                painting.Axis
	textDirection            ui.TextDirection
	tapTargetVerticalPadding float64
	isExpanded               bool
}

func (w *segmentedButtonRenderWidget[T]) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderSegmentedButton{enabledBorder: w.enabledBorder, disabledBorder: w.disabledBorder, textDirection: w.textDirection, direction: w.direction,
		tapTargetVerticalPadding: w.tapTargetVerticalPadding, isExpanded: w.isExpanded}
	r.segmentEnabled = segmentsEnabled(w.segments)
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitContainerRenderObject(&r.ContainerRenderObject, r)
	return r
}

func segmentsEnabled[T comparable](segments []ButtonSegment[T]) []bool {
	out := make([]bool, len(segments))
	for i, s := range segments {
		out[i] = !s.Disabled
	}
	return out
}

func (w *segmentedButtonRenderWidget[T]) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderSegmentedButton)
	r.setSegments(segmentsEnabled(w.segments))
	r.setEnabledBorder(w.enabledBorder)
	r.setDisabledBorder(w.disabledBorder)
	r.setDirection(w.direction)
	r.setTextDirection(w.textDirection)
}

// segmentedButtonParentData is the parent data of a segment: the rectangle that
// surrounds it.
//
// Flutter: material/segmented_button.dart _SegmentedButtonContainerBoxParentData
type segmentedButtonParentData struct {
	rendering.ContainerBoxParentData[rendering.Box]
	surroundingRect *ui.RRect
}

func segmentParentData(child rendering.Box) *segmentedButtonParentData {
	return child.ParentData().(*segmentedButtonParentData)
}

// renderSegmentedButton lays the segments out in a line, with the same size, and
// paints the border around them and the dividers between them.
//
// Flutter: material/segmented_button.dart _RenderSegmentedButton
type renderSegmentedButton struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
	segmentEnabled           []bool
	enabledBorder            painting.OutlinedBorder
	disabledBorder           painting.OutlinedBorder
	textDirection            ui.TextDirection
	direction                painting.Axis
	tapTargetVerticalPadding float64
	isExpanded               bool
}

func (r *renderSegmentedButton) setSegments(value []bool) {
	if slices.Equal(r.segmentEnabled, value) {
		return
	}
	r.segmentEnabled = value
	r.MarkNeedsLayout()
}

func (r *renderSegmentedButton) setEnabledBorder(value painting.OutlinedBorder) {
	if foundation.ValuesEqual(r.enabledBorder, value) {
		return
	}
	r.enabledBorder = value
	r.MarkNeedsLayout()
}

func (r *renderSegmentedButton) setDisabledBorder(value painting.OutlinedBorder) {
	if foundation.ValuesEqual(r.disabledBorder, value) {
		return
	}
	r.disabledBorder = value
	r.MarkNeedsLayout()
}

func (r *renderSegmentedButton) setTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.MarkNeedsLayout()
}

func (r *renderSegmentedButton) setDirection(value painting.Axis) {
	if r.direction == value {
		return
	}
	r.direction = value
	r.MarkNeedsLayout()
}

func (r *renderSegmentedButton) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*segmentedButtonParentData); !ok {
		child.AsRenderObject().SetParentData(&segmentedButtonParentData{})
	}
}

func (r *renderSegmentedButton) eachChild(f func(child rendering.Box)) {
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		f(child)
	}
}

func (r *renderSegmentedButton) ComputeMinIntrinsicWidth(height float64) float64 {
	minWidth := 0.0
	r.eachChild(func(c rendering.Box) { minWidth = math.Max(minWidth, c.GetMinIntrinsicWidth(height)) })
	return minWidth * float64(r.ChildCount())
}

func (r *renderSegmentedButton) ComputeMaxIntrinsicWidth(height float64) float64 {
	maxWidth := 0.0
	r.eachChild(func(c rendering.Box) { maxWidth = math.Max(maxWidth, c.GetMaxIntrinsicWidth(height)) })
	return maxWidth * float64(r.ChildCount())
}

func (r *renderSegmentedButton) ComputeMinIntrinsicHeight(width float64) float64 {
	minHeight := 0.0
	r.eachChild(func(c rendering.Box) { minHeight = math.Max(minHeight, c.GetMinIntrinsicHeight(width)) })
	return minHeight
}

func (r *renderSegmentedButton) ComputeMaxIntrinsicHeight(width float64) float64 {
	maxHeight := 0.0
	r.eachChild(func(c rendering.Box) { maxHeight = math.Max(maxHeight, c.GetMaxIntrinsicHeight(width)) })
	return maxHeight
}

func (r *renderSegmentedButton) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	return rendering.DefaultComputeDistanceToHighestActualBaseline(&r.ContainerRenderObject, baseline)
}

// layoutRects positions the children from the left child on, with next giving the next child.
func (r *renderSegmentedButton) layoutRects(next func(rendering.Box) rendering.Box, leftChild rendering.Box) {
	start := 0.0
	for child := leftChild; child != nil; child = next(child) {
		pd := segmentParentData(child)
		size := child.Size()
		var childRect ui.Rect
		if r.direction == painting.AxisVertical {
			pd.Offset = ui.Offset{Dx: 0.0, Dy: start}
			childRect = ui.NewRectFromLTWH(0.0, pd.Offset.Dy, size.Width, size.Height)
			start += size.Height
		} else {
			pd.Offset = ui.Offset{Dx: start, Dy: 0.0}
			childRect = ui.NewRectFromLTWH(start, 0.0, size.Width, size.Height)
			start += size.Width
		}
		rr := ui.NewRRectFromRectAndCorners(childRect)
		pd.surroundingRect = &rr
	}
}

func (r *renderSegmentedButton) calculateChildSize(constraints rendering.BoxConstraints) ui.Size {
	if r.direction == painting.AxisHorizontal {
		return r.calculateHorizontalChildSize(constraints)
	}
	return r.calculateVerticalChildSize(constraints)
}

func (r *renderSegmentedButton) calculateHorizontalChildSize(constraints rendering.BoxConstraints) ui.Size {
	maxHeight := 0.0
	count := float64(r.ChildCount())
	var childWidth float64
	if r.isExpanded {
		childWidth = constraints.MaxWidth / count
	} else {
		childWidth = constraints.MinWidth / count
		r.eachChild(func(c rendering.Box) { childWidth = math.Max(childWidth, c.GetMaxIntrinsicWidth(math.Inf(1))) })
		childWidth = math.Min(childWidth, constraints.MaxWidth/count)
	}
	r.eachChild(func(c rendering.Box) { maxHeight = math.Max(maxHeight, c.GetMaxIntrinsicHeight(childWidth)) })
	return ui.Size{Width: childWidth, Height: maxHeight}
}

func (r *renderSegmentedButton) calculateVerticalChildSize(constraints rendering.BoxConstraints) ui.Size {
	maxWidth := 0.0
	count := float64(r.ChildCount())
	var childHeight float64
	if r.isExpanded {
		childHeight = constraints.MaxHeight / count
	} else {
		childHeight = constraints.MinHeight / count
		r.eachChild(func(c rendering.Box) { childHeight = math.Max(childHeight, c.GetMaxIntrinsicHeight(math.Inf(1))) })
		childHeight = math.Min(childHeight, constraints.MaxHeight/count)
	}
	r.eachChild(func(c rendering.Box) { maxWidth = math.Max(maxWidth, c.GetMaxIntrinsicWidth(maxWidth)) })
	childSize := ui.Size{Width: maxWidth, Height: childHeight}
	if constraints.HasTightWidth() && childSize.Width < constraints.MaxWidth {
		childSize = ui.Size{Width: constraints.MaxWidth, Height: childSize.Height}
	}
	return childSize
}

func (r *renderSegmentedButton) computeOverallSizeFromChildSize(constraints rendering.BoxConstraints, childSize ui.Size) ui.Size {
	count := float64(r.ChildCount())
	if r.direction == painting.AxisVertical {
		return constraints.Constrain(ui.Size{Width: childSize.Width, Height: childSize.Height * count})
	}
	return constraints.Constrain(ui.Size{Width: childSize.Width * count, Height: childSize.Height})
}

func (r *renderSegmentedButton) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	return r.computeOverallSizeFromChildSize(constraints, r.calculateChildSize(constraints))
}

func (r *renderSegmentedButton) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	childConstraints := rendering.NewBoxConstraintsTight(r.calculateChildSize(constraints))
	result := foundation.None[float64]()
	r.eachChild(func(c rendering.Box) {
		if v, ok := c.GetDryBaseline(childConstraints, baseline).Get(); ok {
			if cur, has := result.Get(); !has || v < cur {
				result = foundation.Some(v)
			}
		}
	})
	return result
}

func (r *renderSegmentedButton) PerformLayout() {
	constraints := r.BoxConstraints()
	childSize := r.calculateChildSize(constraints)
	childConstraints := rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(childSize.Width), Height: foundation.Some(childSize.Height)})
	r.eachChild(func(c rendering.Box) { c.Layout(childConstraints, rendering.LayoutOpts{ParentUsesSize: true}) })
	if r.textDirection == ui.TextDirectionRTL {
		r.layoutRects(r.ChildBefore, r.LastChild())
	} else {
		r.layoutRects(r.ChildAfter, r.FirstChild())
	}
	r.SetSize(r.computeOverallSizeFromChildSize(constraints, childSize))
}

func (r *renderSegmentedButton) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	size := r.Size()
	borderRect := ui.NewRectFromLTWH(offset.Dx, offset.Dy+r.tapTargetVerticalPadding/2, size.Width, size.Height-r.tapTargetVerticalPadding)
	borderClipPath := r.enabledBorder.GetInnerPath(borderRect, r.textDirection)
	canvas := context.Canvas()
	var previousChild rendering.Box
	index := 0
	var enabledClipPath, disabledClipPath *ui.Path
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		pd := segmentParentData(child)
		childRect := pd.surroundingRect.OuterRect().Shift(offset)

		canvas.Save()
		canvas.ClipPath(borderClipPath)
		context.PaintChild(child, pd.Offset.Add(offset))
		canvas.Restore()

		borderOutset := math.Max(r.enabledBorder.Side().StrokeOutset(), r.disabledBorder.Side().StrokeOutset())
		var segmentLeft, segmentRight, dividerPos float64
		if r.textDirection == ui.TextDirectionRTL {
			segmentLeft = childRect.Left
			if child == r.LastChild() {
				segmentLeft = borderRect.Left - borderOutset
			}
			segmentRight = childRect.Right
			if child == r.FirstChild() {
				segmentRight = borderRect.Right + borderOutset
			}
			dividerPos = segmentRight
		} else {
			segmentLeft = childRect.Left
			if child == r.FirstChild() {
				segmentLeft = borderRect.Left - borderOutset
			}
			segmentRight = childRect.Right
			if child == r.LastChild() {
				segmentRight = borderRect.Right + borderOutset
			}
			dividerPos = segmentLeft
		}
		segmentClipRect := ui.Rect{Left: segmentLeft, Top: borderRect.Top - borderOutset, Right: segmentRight, Bottom: borderRect.Bottom + borderOutset}
		if r.segmentEnabled[index] {
			if enabledClipPath == nil {
				enabledClipPath = ui.NewPath()
			}
			enabledClipPath.AddRect(segmentClipRect)
		} else {
			if disabledClipPath == nil {
				disabledClipPath = ui.NewPath()
			}
			disabledClipPath.AddRect(segmentClipRect)
		}

		if previousChild != nil {
			side := r.disabledBorder.Side()
			if r.segmentEnabled[index-1] || r.segmentEnabled[index] {
				side = r.enabledBorder.Side()
			}
			divider := side.CopyWith(painting.BorderSideCopy{StrokeAlign: foundation.Some(0.0)})
			switch r.direction {
			case painting.AxisHorizontal:
				canvas.DrawLine(ui.Offset{Dx: dividerPos, Dy: borderRect.Top}, ui.Offset{Dx: dividerPos, Dy: borderRect.Bottom}, divider.ToPaint())
			case painting.AxisVertical:
				canvas.Save()
				canvas.ClipPath(borderClipPath)
				canvas.DrawLine(ui.Offset{Dx: borderRect.Left, Dy: childRect.Top}, ui.Offset{Dx: borderRect.Right, Dy: childRect.Top}, divider.ToPaint())
				canvas.Restore()
			}
		}
		previousChild = child
		index++
	}

	switch {
	case disabledClipPath == nil:
		r.enabledBorder.Paint(canvas, borderRect, r.textDirection)
	case enabledClipPath == nil:
		r.disabledBorder.Paint(canvas, borderRect, r.textDirection)
	default:
		canvas.Save()
		canvas.ClipPath(enabledClipPath)
		r.enabledBorder.Paint(canvas, borderRect, r.textDirection)
		canvas.Restore()
		canvas.Save()
		canvas.ClipPath(disabledClipPath)
		r.disabledBorder.Paint(canvas, borderRect, r.textDirection)
		canvas.Restore()
	}
}

func (r *renderSegmentedButton) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	for child := r.LastChild(); child != nil; child = segmentParentData(child).PreviousSibling {
		pd := segmentParentData(child)
		if pd.surroundingRect.Contains(position) {
			offset := pd.Offset
			return result.AddWithPaintOffset(&offset, position, func(result *rendering.BoxHitTestResult, local ui.Offset) bool {
				return child.HitTest(result, local)
			})
		}
	}
	return false
}

// segmentedButtonDefaultsM3 is the default theme of the segmented buttons in
// Material 3.
//
// Flutter: material/segmented_button.dart _SegmentedButtonDefaultsM3
func segmentedButtonDefaultsM3(bc widgets.BuildContext) SegmentedButtonThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	some := foundation.Some[ui.Color]
	overlay := func(states widgets.WidgetStates, c ui.Color) foundation.Opt[ui.Color] {
		switch {
		case states.Has(widgets.WidgetStatePressed):
			return some(c.WithOpacity(0.1))
		case states.Has(widgets.WidgetStateHovered):
			return some(c.WithOpacity(0.08))
		case states.Has(widgets.WidgetStateFocused):
			return some(c.WithOpacity(0.1))
		}
		return foundation.None[ui.Color]()
	}
	return SegmentedButtonThemeData{
		Style: &ButtonStyle{
			TextStyle: widgets.NewWidgetStatePropertyAll(theme.TextTheme.LabelLarge),
			BackgroundColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
				if !states.Has(widgets.WidgetStateDisabled) && states.Has(widgets.WidgetStateSelected) {
					return some(colors.SecondaryContainer())
				}
				return foundation.None[ui.Color]()
			}),
			ForegroundColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
				switch {
				case states.Has(widgets.WidgetStateDisabled):
					return some(colors.OnSurface().WithOpacity(0.38))
				case states.Has(widgets.WidgetStateSelected):
					return some(colors.OnSecondaryContainer())
				}
				return some(colors.OnSurface())
			}),
			OverlayColor: colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
				if states.Has(widgets.WidgetStateSelected) {
					return overlay(states, colors.OnSecondaryContainer())
				}
				return overlay(states, colors.OnSurface())
			}),
			SurfaceTintColor: allOpt(Colors.Transparent),
			Elevation:        allOpt(0.0),
			IconSize:         allOpt(18.0),
			Side: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[painting.BorderSide] {
				c := colors.Outline()
				if states.Has(widgets.WidgetStateDisabled) {
					c = colors.OnSurface().WithOpacity(0.12)
				}
				return foundation.Some(painting.NewBorderSide(painting.BorderSideOpts{Color: some(c)}))
			}),
			Shape:       widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](painting.NewStadiumBorder()),
			MinimumSize: allOpt(ui.NewSizeFromHeight(40.0)),
		},
		SelectedIcon: NewIcon(Icons.Check()),
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagButtonSegmentEmpty              = "ButtonSegmentEmpty"
	tagSegmentedButtonMultipleSelected = "SegmentedButtonMultipleSelected"
	tagSegmentedButtonNoSegments       = "SegmentedButtonNoSegments"
	tagSegmentedButtonNoSelection      = "SegmentedButtonNoSelection"
)
