// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The sizes and the timing of the popup menus.
const (
	popupMenuDuration         = 300 * time.Millisecond
	popupMenuCloseIntervalEnd = 2.0 / 3.0
	popupMenuDividerHeight    = 16.0
	popupMenuWidthStep        = 56.0
	popupMenuMaxWidth         = 5.0 * popupMenuWidthStep
	popupMenuMinWidth         = 2.0 * popupMenuWidthStep
	popupMenuScreenPadding    = 8.0
)

// PopupMenuEntry is an item of a popup menu: a PopupMenuItem, a
// CheckedPopupMenuItem or a PopupMenuDivider, or any widget that has a height
// and can say whether it stands for a value.
//
// Deviation: Flutter's PopupMenuEntry<T> has the type of the value; here the
// value is an any, so that the PopupMenuDivider (a PopupMenuEntry<Never> there)
// fits in every menu. The PopupMenuButton[T] gives the type back.
//
// Flutter: material/popup_menu.dart PopupMenuEntry
type PopupMenuEntry interface {
	widgets.Widget
	// Height is the amount of vertical space occupied by this entry.
	//
	// Flutter: material/popup_menu.dart PopupMenuEntry.height
	Height() float64
	// Represents is whether the entry stands for the value: the menu highlights
	// the entry of its initial value.
	//
	// Flutter: material/popup_menu.dart PopupMenuEntry.represents
	Represents(value any) bool
}

// PopupMenuDividerOpts are the optional parameters of NewPopupMenuDivider.
//
// Flutter: material/popup_menu.dart PopupMenuDivider.new
type PopupMenuDividerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Height is the height of the divider entry; the default is 16.
	Height foundation.Opt[float64]
	// Thickness is the thickness of the line that is drawn within the divider.
	Thickness foundation.Opt[float64]
	// Indent is the amount of empty space at the start of the divider.
	Indent foundation.Opt[float64]
	// EndIndent is the amount of empty space at the end of the divider.
	EndIndent foundation.Opt[float64]
	// Radius is the amount of radius for the border of the divider.
	Radius painting.BorderRadiusGeometry
	// Color is the color of the divider line.
	Color foundation.Opt[ui.Color]
}

// PopupMenuDivider is a horizontal divider in a Material Design popup menu. It
// is a PopupMenuEntry of a menu that shows a Divider.
//
// Flutter: material/popup_menu.dart PopupMenuDivider
type PopupMenuDivider struct {
	widgets.StatelessWidget
	o PopupMenuDividerOpts
}

// NewPopupMenuDivider creates the divider.
//
// Flutter: material/popup_menu.dart PopupMenuDivider.new
func NewPopupMenuDivider(opts ...PopupMenuDividerOpts) *PopupMenuDivider {
	o := firstOpts(opts)
	return &PopupMenuDivider{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Height is the height of the divider.
//
// Flutter: material/popup_menu.dart PopupMenuDivider.height
func (w *PopupMenuDivider) Height() float64 { return w.o.Height.OrElse(popupMenuDividerHeight) }

// Represents is false: a divider stands for no value.
//
// Flutter: material/popup_menu.dart PopupMenuDivider.represents
func (w *PopupMenuDivider) Represents(any) bool { return false }

// Build builds a Divider.
//
// Flutter: material/popup_menu.dart _PopupMenuDividerState.build
func (w *PopupMenuDivider) Build(bc widgets.BuildContext) widgets.Widget {
	return NewDivider(DividerOpts{
		Height: foundation.Some(w.Height()), Thickness: w.o.Thickness, Indent: w.o.Indent, EndIndent: w.o.EndIndent, Radius: w.o.Radius, Color: w.o.Color})
}

// popupMenuItemLayout reports the size of each menu item: the sizes are used by
// the layout of the route to compute the y coordinate of the origin of the menu
// so that the center of the selected menu item lines up with the center of its
// PopupMenuButton.
//
// Flutter: material/popup_menu.dart _MenuItem
type popupMenuItemLayout struct {
	widgets.SingleChildRenderObjectWidget
	onLayout func(size ui.Size)
}

func newPopupMenuItemLayout(onLayout func(size ui.Size), child widgets.Widget) *popupMenuItemLayout {
	return &popupMenuItemLayout{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child), onLayout: onLayout}
}

func (w *popupMenuItemLayout) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderPopupMenuItem{onLayout: w.onLayout}
	rendering.InitRenderShiftedBox(&r.RenderShiftedBox, r, nil)
	return r
}

func (w *popupMenuItemLayout) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*renderPopupMenuItem).onLayout = w.onLayout
}

// renderPopupMenuItem is the render object of popupMenuItemLayout.
//
// Flutter: material/popup_menu.dart _RenderMenuItem
type renderPopupMenuItem struct {
	rendering.RenderShiftedBox
	onLayout func(size ui.Size)
}

func (r *renderPopupMenuItem) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	if child := r.Child(); child != nil {
		return child.GetDryLayout(constraints)
	}
	return ui.Size{}
}

func (r *renderPopupMenuItem) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if child := r.Child(); child != nil {
		return child.GetDryBaseline(constraints, baseline)
	}
	return foundation.None[float64]()
}

func (r *renderPopupMenuItem) PerformLayout() {
	if child := r.Child(); child == nil {
		r.SetSize(ui.Size{})
	} else {
		child.Layout(r.BoxConstraints(), rendering.LayoutOpts{ParentUsesSize: true})
		r.SetSize(r.BoxConstraints().Constrain(child.Size()))
		rendering.BoxParentDataOf(child).Offset = ui.Offset{}
	}
	r.onLayout(r.Size())
}

// PopupMenuItemOpts are the optional parameters of NewPopupMenuItem.
//
// Flutter: material/popup_menu.dart PopupMenuItem.new
type PopupMenuItemOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Value is the value that is passed to the select callback of the menu when
	// the item is tapped, and what it is compared with to highlight the item that
	// is the initial value of the menu.
	Value foundation.Opt[T]
	// OnTap is called when the item is tapped, after the menu is popped.
	OnTap func()
	// Disabled makes the item unable to be tapped (Flutter's enabled: false).
	Disabled bool
	// Height is the minimum height of the item; the default is 48.
	Height foundation.Opt[float64]
	// Padding is the padding of the item; the default is 12 on each side
	// (Material 3) or 16 (Material 2).
	Padding painting.EdgeInsetsGeometry
	// TextStyle is the style of the text of the item, in Material 2.
	TextStyle *painting.TextStyle
	// LabelTextStyle is the style of the text of the item, in each state, in
	// Material 3.
	LabelTextStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// MouseCursor is the cursor over the item.
	MouseCursor services.MouseCursor
}

// PopupMenuItem is an item in a Material Design popup menu. To show a checkmark
// next to a popup menu item, use a CheckedPopupMenuItem.
//
// Flutter: material/popup_menu.dart PopupMenuItem
type PopupMenuItem[T comparable] struct {
	widgets.StatefulWidget
	child widgets.Widget
	o     PopupMenuItemOpts[T]
	// checked is none for a plain item, and the state of a CheckedPopupMenuItem.
	checked foundation.Opt[bool]
}

// NewPopupMenuItem creates an item; the child, which may be nil, is its label.
//
// Flutter: material/popup_menu.dart PopupMenuItem.new
func NewPopupMenuItem[T comparable](child widgets.Widget, opts ...PopupMenuItemOpts[T]) *PopupMenuItem[T] {
	o := firstOpts(opts)
	return &PopupMenuItem[T]{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// Enabled is whether the item can be tapped.
//
// Flutter: material/popup_menu.dart PopupMenuItem.enabled
func (w *PopupMenuItem[T]) Enabled() bool { return !w.o.Disabled }

// Value is the value of the item.
//
// Flutter: material/popup_menu.dart PopupMenuItem.value
func (w *PopupMenuItem[T]) Value() foundation.Opt[T] { return w.o.Value }

// Height is the minimum height of the item.
//
// Flutter: material/popup_menu.dart PopupMenuItem.height
func (w *PopupMenuItem[T]) Height() float64 { return w.o.Height.OrElse(MinInteractiveDimension) }

// Child is the label of the item.
//
// Flutter: material/popup_menu.dart PopupMenuItem.child
func (w *PopupMenuItem[T]) Child() widgets.Widget { return w.child }

// Represents is whether the value is the one of the item.
//
// Flutter: material/popup_menu.dart PopupMenuItem.represents
func (w *PopupMenuItem[T]) Represents(value any) bool {
	v, ok := value.(T)
	return ok && w.o.Value == foundation.Some(v)
}

// CreateState creates the state.
//
// Flutter: material/popup_menu.dart PopupMenuItem.createState
func (w *PopupMenuItem[T]) CreateState() widgets.State { return &popupMenuItemState[T]{} }

// CheckedPopupMenuItemOpts are the optional parameters of
// NewCheckedPopupMenuItem.
//
// Flutter: material/popup_menu.dart CheckedPopupMenuItem.new
type CheckedPopupMenuItemOpts[T comparable] struct {
	PopupMenuItemOpts[T]
	// Checked is whether to display a checkmark next to the label of the item.
	Checked bool
}

// CheckedPopupMenuItem is a menu item with a checkmark. It is a PopupMenuItem
// whose label has a checkmark before it that fades in or out when the item is
// tapped.
//
// Flutter: material/popup_menu.dart CheckedPopupMenuItem
type CheckedPopupMenuItem[T comparable] struct {
	*PopupMenuItem[T]
}

// NewCheckedPopupMenuItem creates an item with a checkmark.
//
// Flutter: material/popup_menu.dart CheckedPopupMenuItem.new
func NewCheckedPopupMenuItem[T comparable](child widgets.Widget, opts ...CheckedPopupMenuItemOpts[T]) *CheckedPopupMenuItem[T] {
	o := firstOpts(opts)
	item := NewPopupMenuItem(child, o.PopupMenuItemOpts)
	item.checked = foundation.Some(o.Checked)
	return &CheckedPopupMenuItem[T]{PopupMenuItem: item}
}

// Checked is whether the item shows a checkmark.
//
// Flutter: material/popup_menu.dart CheckedPopupMenuItem.checked
func (w *CheckedPopupMenuItem[T]) Checked() bool { return w.checked.OrElse(false) }

// popupMenuItemState is the state of a PopupMenuItem, and of a
// CheckedPopupMenuItem, which has a controller for the fade of its checkmark.
//
// Flutter: material/popup_menu.dart PopupMenuItemState
type popupMenuItemState[T comparable] struct {
	widgets.StateOf[*PopupMenuItem[T]]
	widgets.SingleTickerProvider
	controller *animation.AnimationController
}

const popupMenuCheckFadeDuration = 150 * time.Millisecond

func (s *popupMenuItemState[T]) InitState() {
	if s.Widget().checked.IsSome() {
		value := 0.0
		if s.Widget().checked.Value() {
			value = 1.0
		}
		s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{
			Duration: foundation.Some(popupMenuCheckFadeDuration), Value: foundation.Some(value)})
		s.controller.AddListener(func() { s.SetState(func() {}) })
	}
}

func (s *popupMenuItemState[T]) Dispose() {
	if s.controller != nil {
		s.controller.Dispose()
	}
}

// handleTap pops the menu with the value of the item, and calls OnTap.
func (s *popupMenuItemState[T]) handleTap() {
	w := s.Widget()
	if s.controller != nil {
		// This fades the checkmark in or out when tapped.
		if w.checked.Value() {
			s.controller.Reverse()
		} else {
			s.controller.Forward()
		}
	}
	// Need to pop the navigator first in case onTap may push a new route onto the
	// navigator.
	var result any
	if v, ok := w.o.Value.Get(); ok {
		result = v
	}
	widgets.NavigatorPop(s.Context(), result)
	if w.o.OnTap != nil {
		w.o.OnTap()
	}
}

// Build builds the item with the text style and padding of Material 3 or 2.
//
// Flutter: material/popup_menu.dart PopupMenuItemState.build
func (s *popupMenuItemState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	theme := ThemeOf(bc)
	popupMenuTheme := PopupMenuThemeOf(bc)
	defaults := popupMenuDefaults(bc)
	states := widgets.NewWidgetStates()
	if !w.Enabled() {
		states = states.With(widgets.WidgetStateDisabled)
	}
	var style *painting.TextStyle
	if theme.UseMaterial3 {
		switch {
		case w.o.LabelTextStyle != nil:
			style = w.o.LabelTextStyle.Resolve(states)
		case popupMenuTheme.LabelTextStyle != nil:
			style = popupMenuTheme.LabelTextStyle.Resolve(states)
		default:
			style = defaults.LabelTextStyle.Resolve(states)
		}
	} else {
		switch {
		case w.o.TextStyle != nil:
			style = w.o.TextStyle
		case popupMenuTheme.TextStyle != nil:
			style = popupMenuTheme.TextStyle
		default:
			style = defaults.TextStyle
		}
	}
	if !w.Enabled() && !theme.UseMaterial3 {
		style = style.CopyWith(painting.TextStyleCopy{Color: foundation.Some(theme.DisabledColor)})
	}
	padding := w.o.Padding
	if padding == nil {
		padding = popupMenuItemPadding(theme.UseMaterial3)
	}
	var label widgets.Widget
	if s.controller != nil {
		label = s.buildCheckedChild(bc)
	} else {
		label = w.child
	}
	var item widgets.Widget = widgets.NewAnimatedDefaultTextStyle(style, ThemeChangeDuration,
		widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinHeight: foundation.Some(w.Height())}),
			widgets.NewPadding(padding, widgets.NewAlign(label, widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart}))))
	if !w.Enabled() {
		opacity := 0.38
		if theme.Brightness() == ui.BrightnessDark {
			opacity = 0.5
		}
		item = widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Opacity: foundation.Some(opacity)}), item)
	}
	var onTap func()
	if w.Enabled() {
		onTap = s.handleTap
	}
	// The ListTiles of the label get no content padding and the style of the label.
	return widgets.NewMergeSemantics(s.buildSemantics(NewInkWell(ListTileThemeMerge(ListTileThemeData{ContentPadding: painting.EdgeInsetsZero, TitleTextStyle: style}, item), InkWellOpts{
		OnTap:           onTap,
		CanRequestFocus: foundation.Some(w.Enabled()),
		MouseCursor:     popupEffectiveMouseCursor(w.o.MouseCursor, popupMenuTheme.MouseCursor),
	})))
}

func (s *popupMenuItemState[T]) buildSemantics(child widgets.Widget) widgets.Widget {
	w := s.Widget()
	if s.controller != nil {
		return widgets.NewSemantics(child, widgets.SemanticsOpts{
			Role: foundation.Some(ui.SemanticsRoleMenuItemCheckbox), Enabled: foundation.Some(w.Enabled()),
			Checked: w.checked, Button: foundation.Some(true)})
	}
	return widgets.NewSemantics(child, widgets.SemanticsOpts{
		Role: foundation.Some(ui.SemanticsRoleMenuItem), Enabled: foundation.Some(w.Enabled()), Button: foundation.Some(true)})
}

// buildCheckedChild is the label of a checked item: the checkmark that fades,
// then the label.
//
// Flutter: material/popup_menu.dart _CheckedPopupMenuItemState.buildChild
func (s *popupMenuItemState[T]) buildCheckedChild(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	popupMenuTheme := PopupMenuThemeOf(bc)
	defaults := popupMenuDefaults(bc)
	states := widgets.NewWidgetStates()
	if w.checked.Value() {
		states = states.With(widgets.WidgetStateSelected)
	}
	effective := w.o.LabelTextStyle
	if effective == nil {
		effective = popupMenuTheme.LabelTextStyle
	}
	if effective == nil {
		effective = defaults.LabelTextStyle
	}
	var titleStyle *painting.TextStyle
	if effective != nil {
		titleStyle = effective.Resolve(states)
	}
	var check widgets.Widget
	if s.controller.Status() == animation.AnimationStatusDismissed {
		check = widgets.NewIcon(widgets.IconData{})
	} else {
		check = widgets.NewIcon(Icons.Done())
	}
	return widgets.NewIgnorePointer(NewListTile(ListTileOpts{
		Disabled: !w.Enabled(), TitleTextStyle: titleStyle, Title: w.child,
		Leading: widgets.NewFadeTransition(s.controller, check),
	}))
}

// popupEffectiveMouseCursor is the cursor over an item: the cursor of the item,
// then of the theme, then the clickable one, resolved against the states that
// the ink well gives.
//
// Flutter: material/popup_menu.dart _EffectiveMouseCursor
func popupEffectiveMouseCursor(widgetCursor services.MouseCursor, themeCursor widgets.WidgetStateProperty[services.MouseCursor]) services.MouseCursor {
	return widgets.WidgetStateMouseCursorResolveWith(func(states widgets.WidgetStates) services.MouseCursor {
		if widgetCursor != nil {
			if p, ok := widgetCursor.(widgets.WidgetStateProperty[services.MouseCursor]); ok {
				if c := p.Resolve(states); c != nil {
					return c
				}
			} else {
				return widgetCursor
			}
		}
		if themeCursor != nil {
			if c := themeCursor.Resolve(states); c != nil {
				return c
			}
		}
		return widgets.WidgetStateMouseCursorAdaptiveClickable.Resolve(states)
	}, "WidgetStateMouseCursor(PopupMenuItemState)")
}

// popupMenuItemPadding is the padding of an item of a popup menu.
//
// Flutter: material/popup_menu.dart _PopupMenuDefaultsM3.menuItemPadding
func popupMenuItemPadding(material3 bool) painting.EdgeInsetsGeometry {
	if material3 {
		return painting.NewEdgeInsetsSymmetric(0, 12.0)
	}
	return painting.NewEdgeInsetsSymmetric(0, 16.0)
}

// popupMenuDefaults is the default theme of the popup menus in Material 3 or 2.
//
// Flutter: material/popup_menu.dart _PopupMenuDefaultsM3
func popupMenuDefaults(bc widgets.BuildContext) PopupMenuThemeData {
	theme := ThemeOf(bc)
	if !theme.UseMaterial3 {
		// Flutter: material/popup_menu.dart _PopupMenuDefaultsM2
		return PopupMenuThemeData{
			Elevation:   foundation.Some(8.0),
			TextStyle:   theme.TextTheme.TitleMedium,
			MenuPadding: painting.NewEdgeInsetsSymmetric(8.0, 0),
		}
	}
	colors := theme.ColorScheme
	return PopupMenuThemeData{
		Elevation: foundation.Some(3.0),
		LabelTextStyle: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) *painting.TextStyle {
			// TODO(quncheng): Update this hard-coded value to use the latest tokens.
			style := theme.TextTheme.LabelLarge
			if states.Has(widgets.WidgetStateDisabled) {
				return style.Apply(painting.TextStyleApply{Color: foundation.Some(colors.OnSurface().WithOpacity(0.38))})
			}
			return style.Apply(painting.TextStyleApply{Color: foundation.Some(colors.OnSurface())})
		}),
		Color:            foundation.Some(colors.SurfaceContainer()),
		ShadowColor:      foundation.Some(colors.Shadow()),
		SurfaceTintColor: foundation.Some(Colors.Transparent),
		Shape:            painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(4.0))}),
		// This is taken from https://m3.material.io/components/menus/specs.
		MenuPadding: painting.NewEdgeInsetsSymmetric(8.0, 0),
	}
}

// popupMenu is the panel of a popup menu: the items in a scrollable Material
// that grows as the route animates, and the items fade in one after the other.
//
// Flutter: material/popup_menu.dart _PopupMenu
type popupMenu struct {
	widgets.StatefulWidget
	itemKeys      []*widgets.GlobalKey[widgets.State]
	route         *popupMenuRoute
	semanticLabel string
	constraints   foundation.Opt[rendering.BoxConstraints]
	clipBehavior  ui.Clip
}

func (w *popupMenu) CreateState() widgets.State { return &popupMenuState{} }

type popupMenuState struct {
	widgets.StateOf[*popupMenu]
	opacities []*animation.CurvedAnimation
}

func (s *popupMenuState) InitState() { s.setOpacities() }

func (s *popupMenuState) DidUpdateWidget(old *popupMenu) {
	if len(old.route.items) != len(s.Widget().route.items) || old.route.Animation() != s.Widget().route.Animation() {
		s.setOpacities()
	}
}

func (s *popupMenuState) setOpacities() {
	for _, o := range s.opacities {
		o.Dispose()
	}
	route := s.Widget().route
	// 1.0 for the width and 0.5 for the last item's fade.
	unit := 1.0 / (float64(len(route.items)) + 1.5)
	opacities := make([]*animation.CurvedAnimation, 0, len(route.items))
	for i := range route.items {
		start := float64(i+1) * unit
		end := ui.ClampDouble(start+1.5*unit, 0.0, 1.0)
		opacities = append(opacities, animation.NewCurvedAnimation(route.Animation(), animation.NewInterval(start, end)))
	}
	s.opacities = opacities
}

func (s *popupMenuState) Dispose() {
	for _, o := range s.opacities {
		o.Dispose()
	}
}

// Build builds the menu.
//
// Flutter: material/popup_menu.dart _PopupMenuState.build
func (s *popupMenuState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	route := w.route
	unit := 1.0 / (float64(len(route.items)) + 1.5) // 1.0 for the width and 0.5 for the last item's fade.
	theme := ThemeOf(bc)
	popupMenuTheme := PopupMenuThemeOf(bc)
	defaults := popupMenuDefaults(bc)
	children := make([]widgets.Widget, 0, len(route.items))
	for i, entry := range route.items {
		var item widgets.Widget = entry
		if route.initialValue != nil && entry.Represents(route.initialValue) {
			item = widgets.NewColoredBox(theme.HighlightColor, item)
		}
		i := i
		children = append(children, newPopupMenuItemLayout(func(size ui.Size) { route.itemSizes[i] = foundation.Some(size) },
			widgets.NewFadeTransition(s.opacities[i], item, widgets.FadeTransitionOpts{Key: w.itemKeys[i]})))
	}
	opacity := animation.NewCurveTween(animation.NewInterval(0.0, 1.0/3.0))
	width := animation.NewCurveTween(animation.NewInterval(0.0, unit))
	height := animation.NewCurveTween(animation.NewInterval(0.0, unit*float64(len(route.items))))

	constraints := rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinWidth: foundation.Some(popupMenuMinWidth), MaxWidth: foundation.Some(popupMenuMaxWidth)})
	if c, ok := w.constraints.Get(); ok {
		constraints = c
	}
	menuPadding := route.menuPadding
	if menuPadding == nil {
		menuPadding = popupMenuTheme.MenuPadding
	}
	if menuPadding == nil {
		menuPadding = defaults.MenuPadding
	}
	child := widgets.NewConstrainedBox(constraints,
		widgets.NewIntrinsicWidth(
			widgets.NewSemantics(
				widgets.NewSingleChildScrollView(widgets.NewListBody(children), widgets.SingleChildScrollViewOpts{Padding: menuPadding}),
				widgets.SemanticsOpts{
					Role: foundation.Some(ui.SemanticsRoleMenu), ScopesRoute: foundation.Some(true), NamesRoute: foundation.Some(true),
					ExplicitChildNodes: true, Label: optString(w.semanticLabel)}),
			widgets.IntrinsicWidthOpts{StepWidth: foundation.Some(popupMenuWidthStep)}))

	return widgets.NewAnimatedBuilder(route.Animation(), func(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
		anim := route.Animation()
		shape := route.shape
		if shape == nil {
			shape = popupMenuTheme.Shape
		}
		if shape == nil {
			shape = defaults.Shape
		}
		return widgets.NewFadeTransition(animation.Drive[float64](anim, opacity),
			NewMaterial(
				widgets.NewAlign(child, widgets.AlignOpts{
					Alignment:    painting.AlignmentDirectionalTopEnd,
					WidthFactor:  foundation.Some(width.Evaluate(anim)),
					HeightFactor: foundation.Some(height.Evaluate(anim)),
				}),
				MaterialOpts{
					Shape:            shape,
					Color:            firstOpt(route.color, popupMenuTheme.Color, defaults.Color),
					ClipBehavior:     w.clipBehavior,
					Type:             MaterialTypeCard,
					Elevation:        firstOpt(route.elevation, popupMenuTheme.Elevation, defaults.Elevation).OrElse(0),
					ShadowColor:      firstOpt(route.shadowColor, popupMenuTheme.ShadowColor, defaults.ShadowColor),
					SurfaceTintColor: firstOpt(route.surfaceTintColor, popupMenuTheme.SurfaceTintColor, defaults.SurfaceTintColor),
				}))
	}, child)
}

// optString is a string, or none when it is empty.
func optString(s string) foundation.Opt[string] {
	if s == "" {
		return foundation.None[string]()
	}
	return foundation.Some(s)
}

// popupMenuRouteLayout positions the menu on the screen.
//
// Flutter: material/popup_menu.dart _PopupMenuRouteLayout
type popupMenuRouteLayout struct {
	rendering.SingleChildLayoutDelegateBase
	// position is the rectangle of the underlying button, relative to the
	// overlay's dimensions.
	position rendering.RelativeRect
	// itemSizes are the sizes of each item, which are computed when the menu is
	// laid out, and before the route is laid out.
	itemSizes []foundation.Opt[ui.Size]
	// selectedItemIndex is the index of the selected item, or none if the initial
	// value was not specified.
	selectedItemIndex foundation.Opt[int]
	// textDirection is whether to prefer going to the left or to the right.
	textDirection ui.TextDirection
	// padding is the padding of the unsafe area.
	padding painting.EdgeInsets
}

// GetSize is the biggest size.
func (d *popupMenuRouteLayout) GetSize(constraints rendering.BoxConstraints) ui.Size {
	return constraints.Biggest()
}

// GetConstraintsForChild: the menu can be at most the size of the overlay minus
// 8 pixels in each direction. The child is put wherever position specifies, so
// long as it fits within the specified parent size padded (inset) by 8; if
// necessary, the position is adjusted so that it fits.
func (d *popupMenuRouteLayout) GetConstraintsForChild(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	return rendering.NewBoxConstraintsLoose(constraints.Biggest()).Deflate(painting.NewEdgeInsetsAll(popupMenuScreenPadding).Add(d.padding))
}

func (d *popupMenuRouteLayout) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	y := d.position.Top
	// Find the ideal horizontal position. size: the size of the overlay.
	// childSize: the size of the menu, when fully open, as determined by
	// GetConstraintsForChild.
	var x float64
	switch {
	case d.position.Left > d.position.Right:
		// The menu button is closer to the right edge, so grow to the left,
		// aligned to the right edge.
		x = size.Width - d.position.Right - childSize.Width
	case d.position.Left < d.position.Right:
		// The menu button is closer to the left edge, so grow to the right,
		// aligned to the left edge.
		x = d.position.Left
	default:
		// The menu button is equidistant from both edges, so grow in the reading
		// direction.
		if d.textDirection == ui.TextDirectionRTL {
			x = size.Width - d.position.Right - childSize.Width
		} else {
			x = d.position.Left
		}
	}
	return d.fitInsideScreen(ui.Rect{Right: size.Width, Bottom: size.Height}, childSize, ui.Offset{Dx: x, Dy: y})
}

func (d *popupMenuRouteLayout) fitInsideScreen(screen ui.Rect, childSize ui.Size, wanted ui.Offset) ui.Offset {
	x, y := wanted.Dx, wanted.Dy
	p := d.padding
	// Avoid going outside an area defined as the rectangle 8 pixels from the edge
	// of the screen in every direction.
	if x < screen.Left+popupMenuScreenPadding+p.Left {
		x = screen.Left + popupMenuScreenPadding + p.Left
	} else if x+childSize.Width > screen.Right-popupMenuScreenPadding-p.Right {
		x = screen.Right - childSize.Width - popupMenuScreenPadding - p.Right
	}
	if y < screen.Top+popupMenuScreenPadding+p.Top {
		y = popupMenuScreenPadding + p.Top
	} else if y+childSize.Height > screen.Bottom-popupMenuScreenPadding-p.Bottom {
		y = screen.Bottom - childSize.Height - popupMenuScreenPadding - p.Bottom
	}
	return ui.Offset{Dx: x, Dy: y}
}

func (d *popupMenuRouteLayout) ShouldRelayout(old rendering.SingleChildLayoutDelegate) bool {
	o := old.(*popupMenuRouteLayout)
	// The old and new itemSizes have the same length: there's no practical way to
	// change the length of the items list once the menu has been shown.
	if d.position != o.position || d.selectedItemIndex != o.selectedItemIndex || d.textDirection != o.textDirection || d.padding != o.padding {
		return true
	}
	for i := range d.itemSizes {
		if d.itemSizes[i] != o.itemSizes[i] {
			return true
		}
	}
	return false
}

// PopupMenuPositionBuilder is the type of the function that gives the position
// of a popup menu in the overlay, for the constraints of the overlay.
//
// Flutter: material/popup_menu.dart PopupMenuPositionBuilder
type PopupMenuPositionBuilder func(bc widgets.BuildContext, constraints rendering.BoxConstraints) rendering.RelativeRect

// popupMenuRoute is the route of a popup menu.
//
// Flutter: material/popup_menu.dart _PopupMenuRoute
type popupMenuRoute struct {
	widgets.PopupRouteBase
	position                             foundation.Opt[rendering.RelativeRect]
	positionBuilder                      PopupMenuPositionBuilder
	items                                []PopupMenuEntry
	itemKeys                             []*widgets.GlobalKey[widgets.State]
	itemSizes                            []foundation.Opt[ui.Size]
	initialValue                         any
	elevation                            foundation.Opt[float64]
	surfaceTintColor, shadowColor, color foundation.Opt[ui.Color]
	semanticLabel                        string
	barrierLabel                         string
	shape                                painting.ShapeBorder
	menuPadding                          painting.EdgeInsetsGeometry
	capturedThemes                       *widgets.CapturedThemes
	constraints                          foundation.Opt[rendering.BoxConstraints]
	clipBehavior                         ui.Clip
	animationStyle                       *animation.AnimationStyle
	curved                               *animation.CurvedAnimation
}

// TransitionDuration is the duration of the animation style, or 300 ms.
func (r *popupMenuRoute) TransitionDuration() time.Duration {
	if r.animationStyle != nil && r.animationStyle.Duration.IsSome() {
		return r.animationStyle.Duration.Value()
	}
	return popupMenuDuration
}

// ReverseTransitionDuration is the duration of the transition.
func (r *popupMenuRoute) ReverseTransitionDuration() time.Duration { return r.TransitionDuration() }

// BarrierDismissible is true: tapping outside closes the menu.
func (r *popupMenuRoute) BarrierDismissible() bool { return true }

// BarrierColor is none: the barrier is transparent.
func (r *popupMenuRoute) BarrierColor() foundation.Opt[ui.Color] { return foundation.None[ui.Color]() }

// BarrierLabel is the label of the barrier.
func (r *popupMenuRoute) BarrierLabel() string { return r.barrierLabel }

// CreateAnimation is the animation of the controller, curved: linear going
// forward and the first two thirds going in reverse.
func (r *popupMenuRoute) CreateAnimation() animation.Animation[float64] {
	if r.animationStyle != nil && *r.animationStyle == animation.AnimationStyleNoAnimation {
		return r.PopupRouteBase.CreateAnimation()
	}
	if r.curved == nil {
		curve := animation.Curve(animation.Curves.Linear)
		reverse := animation.Curve(animation.NewInterval(0.0, popupMenuCloseIntervalEnd))
		if r.animationStyle != nil {
			if r.animationStyle.Curve != nil {
				curve = r.animationStyle.Curve
			}
			if r.animationStyle.ReverseCurve != nil {
				reverse = r.animationStyle.ReverseCurve
			}
		}
		r.curved = animation.NewCurvedAnimation(r.PopupRouteBase.CreateAnimation(), curve, reverse)
	}
	return r.curved
}

func (r *popupMenuRoute) scrollTo(selectedItemIndex int) {
	widgets.WidgetsBindingOf(r.Navigator().Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		if ctx := r.itemKeys[selectedItemIndex].CurrentContext(); ctx != nil {
			widgets.ScrollableEnsureVisible(ctx)
		}
	})
}

// BuildPage builds the menu in a layout that positions it.
//
// Flutter: material/popup_menu.dart _PopupMenuRoute.buildPage
func (r *popupMenuRoute) BuildPage(bc widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
	selected := foundation.None[int]()
	if r.initialValue != nil {
		for i, entry := range r.items {
			if entry.Represents(r.initialValue) {
				selected = foundation.Some(i)
				break
			}
		}
	}
	if i, ok := selected.Get(); ok {
		r.scrollTo(i)
	}
	menu := &popupMenu{StatefulWidget: widgets.NewStatefulWidgetBase(), route: r, itemKeys: r.itemKeys,
		semanticLabel: r.semanticLabel, constraints: r.constraints, clipBehavior: r.clipBehavior}
	mediaQuery := widgets.MediaQueryOf(bc)
	return widgets.NewMediaQueryRemovePadding(bc, widgets.RemoveSides{Top: true, Bottom: true, Left: true, Right: true},
		widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
			var position rendering.RelativeRect
			if r.positionBuilder != nil {
				position = r.positionBuilder(bc, constraints)
			} else {
				position = r.position.Value()
			}
			return widgets.NewCustomSingleChildLayout(&popupMenuRouteLayout{
				SingleChildLayoutDelegateBase: rendering.NewSingleChildLayoutDelegateBase(nil),
				position:                      position, itemSizes: r.itemSizes, selectedItemIndex: selected,
				textDirection: widgets.DirectionalityOf(bc), padding: mediaQuery.Padding,
			}, r.capturedThemes.Wrap(menu))
		}))
}

// Dispose releases the curved animation.
func (r *popupMenuRoute) Dispose() {
	if r.curved != nil {
		r.curved.Dispose()
	}
	r.PopupRouteBase.Dispose()
}

// ShowMenuOpts are the optional parameters of ShowMenu.
//
// Flutter: material/popup_menu.dart showMenu
type ShowMenuOpts struct {
	// Position is the position of the menu relative to the overlay; give this or
	// PositionBuilder.
	Position foundation.Opt[rendering.RelativeRect]
	// PositionBuilder gives the position of the menu for the constraints of the
	// overlay, when the menu is built; give this or Position.
	PositionBuilder PopupMenuPositionBuilder
	// InitialValue is the value of the item that is highlighted, and scrolled to,
	// when the menu opens.
	InitialValue any
	// Elevation is the z-coordinate of the menu.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow of the menu.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the menu.
	SurfaceTintColor foundation.Opt[ui.Color]
	// SemanticLabel is the label that the semantics of the menu has; the default
	// is the popupMenuLabel of the localizations, except on iOS and macOS.
	SemanticLabel string
	// Shape is the shape of the menu.
	Shape painting.ShapeBorder
	// MenuPadding is the padding of the items.
	MenuPadding painting.EdgeInsetsGeometry
	// Color is the background color of the menu.
	Color foundation.Opt[ui.Color]
	// UseRootNavigator pushes the menu on the root Navigator.
	UseRootNavigator bool
	// Constraints are the size constraints of the menu.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// ClipBehavior is how the menu is clipped; the default is no clip.
	ClipBehavior ui.Clip
	// RouteSettings are the settings of the route.
	RouteSettings widgets.RouteSettings
	// PopUpAnimationStyle is the curve and the durations of the animation of the
	// menu.
	PopUpAnimationStyle *animation.AnimationStyle
	// RequestFocus says whether the menu takes the focus when it opens.
	RequestFocus foundation.Opt[bool]
}

// showMenuRoute pushes a popup menu and returns its route.
func showMenuRoute(bc widgets.BuildContext, items []PopupMenuEntry, o ShowMenuOpts) *popupMenuRoute {
	if foundation.DebugMode {
		if len(items) == 0 {
			panic(errors.New("ShowMenuNoItems", "a popup menu needs at least one item"))
		}
		DebugCheckHasMaterialLocalizations(bc)
		if o.Position.IsSome() == (o.PositionBuilder != nil) {
			panic(errors.New("ShowMenuPosition", "either position or positionBuilder must be provided"))
		}
	}
	semanticLabel := o.SemanticLabel
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
	default:
		if semanticLabel == "" {
			semanticLabel = MaterialLocalizationsOf(bc).PopupMenuLabel()
		}
	}
	itemKeys := make([]*widgets.GlobalKey[widgets.State], len(items))
	for i := range itemKeys {
		itemKeys[i] = widgets.NewGlobalKey[widgets.State]()
	}
	navigator := widgets.NavigatorOf(bc, widgets.NavigatorOfOpts{RootNavigator: o.UseRootNavigator})
	r := &popupMenuRoute{
		position: o.Position, positionBuilder: o.PositionBuilder, items: items, itemKeys: itemKeys,
		itemSizes:    make([]foundation.Opt[ui.Size], len(items)),
		initialValue: o.InitialValue, elevation: o.Elevation, shadowColor: o.ShadowColor, surfaceTintColor: o.SurfaceTintColor,
		semanticLabel: semanticLabel, barrierLabel: MaterialLocalizationsOf(bc).MenuDismissLabel(), shape: o.Shape,
		menuPadding: o.MenuPadding, color: o.Color, capturedThemes: widgets.CaptureThemes(bc, navigator.Context()),
		constraints: o.Constraints, clipBehavior: o.ClipBehavior, animationStyle: o.PopUpAnimationStyle,
	}
	// Menus always cycle focus through their items irrespective of the focus
	// traversal edge behavior set in the Navigator.
	widgets.InitPopupRouteBase(&r.PopupRouteBase, r, widgets.ModalRouteOpts{
		RouteOpts:             widgets.RouteOpts{Settings: o.RouteSettings, RequestFocus: o.RequestFocus},
		TraversalEdgeBehavior: widgets.TraversalEdgeBehaviorClosedLoop,
	})
	navigator.Push(r)
	return r
}

// ShowMenu shows a popup menu that contains the items at the position. The
// channel gets the value of the item that was selected, or nil if the menu was
// dismissed without a selection.
//
// Flutter: material/popup_menu.dart showMenu
func ShowMenu(bc widgets.BuildContext, items []PopupMenuEntry, opts ...ShowMenuOpts) <-chan any {
	return showMenuRoute(bc, items, firstOpts(opts)).Popped()
}

// PopupMenuButtonOpts are the optional parameters of NewPopupMenuButton.
//
// Flutter: material/popup_menu.dart PopupMenuButton.new
type PopupMenuButtonOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// InitialValue is the value of the item that is highlighted when the menu
	// opens.
	InitialValue foundation.Opt[T]
	// OnOpened is called when the menu is opened.
	OnOpened func()
	// OnSelected is called with the value of the item that was selected.
	OnSelected func(value T)
	// OnCanceled is called when the menu is dismissed without a selection.
	OnCanceled func()
	// Tooltip is the text of the tooltip of the button; the default is the
	// showMenuTooltip of the localizations.
	Tooltip string
	// Elevation is the z-coordinate of the menu.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow of the menu.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the menu.
	SurfaceTintColor foundation.Opt[ui.Color]
	// Padding is the padding of the icon button, when there is no child; the
	// default is 8 on every side.
	Padding painting.EdgeInsetsGeometry
	// MenuPadding is the padding of the items of the menu.
	MenuPadding painting.EdgeInsetsGeometry
	// Child is the widget that the button is made of, instead of an icon.
	Child widgets.Widget
	// BorderRadius is the border radius of the ink of the child.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// SplashRadius is the splash radius.
	SplashRadius foundation.Opt[float64]
	// Icon is the icon of the button; the default is "more" (three dots).
	Icon widgets.Widget
	// IconSize is the size of the icon.
	IconSize foundation.Opt[float64]
	// Offset is the offset applied to the position of the menu.
	Offset ui.Offset
	// Disabled makes the button unable to be pressed (Flutter's enabled: false).
	Disabled bool
	// Shape is the shape of the menu.
	Shape painting.ShapeBorder
	// Color is the background color of the menu.
	Color foundation.Opt[ui.Color]
	// IconColor is the color of the icon.
	IconColor foundation.Opt[ui.Color]
	// EnableFeedback says whether detected gestures give acoustic and haptic
	// feedback; the default is true.
	EnableFeedback foundation.Opt[bool]
	// Constraints are the size constraints of the menu.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Position is where the menu opens relative to the button; the default is
	// the one of the theme, or over.
	Position PopupMenuPosition
	// ClipBehavior is how the menu is clipped; the default is no clip.
	ClipBehavior ui.Clip
	// UseRootNavigator pushes the menu on the root Navigator.
	UseRootNavigator bool
	// PopUpAnimationStyle is the curve and the durations of the animation of the
	// menu.
	PopUpAnimationStyle *animation.AnimationStyle
	// RouteSettings are the settings of the route of the menu.
	RouteSettings widgets.RouteSettings
	// Style is the style of the icon button.
	Style *ButtonStyle
	// RequestFocus says whether the menu takes the focus when it opens.
	RequestFocus foundation.Opt[bool]
}

// PopupMenuButton displays a menu when pressed and calls OnSelected when the
// menu is dismissed because an item was selected: the value passed is the value
// of the selected menu item. If child is nil, the button is an icon button with
// the "more" icon.
//
// Flutter: material/popup_menu.dart PopupMenuButton
type PopupMenuButton[T comparable] struct {
	widgets.StatefulWidget
	itemBuilder func(bc widgets.BuildContext) []PopupMenuEntry
	o           PopupMenuButtonOpts[T]
}

// NewPopupMenuButton creates the button; itemBuilder builds the entries of the
// menu each time it opens, and no menu opens when it returns none. In debug
// builds it panics with the tag PopupMenuButtonChildAndIcon when both a child
// and an icon are given.
//
// Flutter: material/popup_menu.dart PopupMenuButton.new
func NewPopupMenuButton[T comparable](itemBuilder func(bc widgets.BuildContext) []PopupMenuEntry, opts ...PopupMenuButtonOpts[T]) *PopupMenuButton[T] {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Child != nil && o.Icon != nil {
		panic(errors.New("PopupMenuButtonChildAndIcon", "you can only pass a child or an icon to a PopupMenuButton, not both"))
	}
	if o.Padding == nil {
		o.Padding = painting.NewEdgeInsetsAll(8.0)
	}
	return &PopupMenuButton[T]{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), itemBuilder: itemBuilder, o: o}
}

// CreateState creates the state.
//
// Flutter: material/popup_menu.dart PopupMenuButton.createState
func (w *PopupMenuButton[T]) CreateState() widgets.State { return &popupMenuButtonState[T]{} }

// popupMenuButtonState is the state of a PopupMenuButton: Show opens the menu.
//
// Flutter: material/popup_menu.dart PopupMenuButtonState
type popupMenuButtonState[T comparable] struct {
	widgets.StateOf[*PopupMenuButton[T]]
	isMenuExpanded   bool
	lastPosition     foundation.Opt[rendering.RelativeRect]
	popupMenuTheme   PopupMenuThemeData
	cachedButton     rendering.Box
	cachedOverlayBox rendering.Box
}

func (s *popupMenuButtonState[T]) DidChangeDependencies() { s.updateCachedObjects() }

func (s *popupMenuButtonState[T]) updateCachedObjects() {
	if !s.Mounted() {
		return
	}
	s.popupMenuTheme = PopupMenuThemeOf(s.Context())
	if box, ok := s.Context().FindRenderObject().(rendering.Box); ok {
		s.cachedButton = box
	}
	navigator := widgets.NavigatorMaybeOf(s.Context(), widgets.NavigatorOfOpts{RootNavigator: s.Widget().o.UseRootNavigator})
	if navigator == nil || navigator.Overlay() == nil {
		s.cachedButton, s.cachedOverlayBox = nil, nil
		return
	}
	if box, ok := navigator.Overlay().Context().FindRenderObject().(rendering.Box); ok {
		s.cachedOverlayBox = box
	}
}

func (s *popupMenuButtonState[T]) defaultPosition(constraints rendering.BoxConstraints) rendering.RelativeRect {
	if p, ok := s.lastPosition.Get(); ok {
		return p
	}
	return rendering.NewRelativeRectFromSize(ui.Rect{}, constraints.Biggest())
}

func (s *popupMenuButtonState[T]) positionBuilder(_ widgets.BuildContext, constraints rendering.BoxConstraints) rendering.RelativeRect {
	if !s.Mounted() {
		// When the route is displayed, the position builder is stored. Even after
		// the button has been unmounted and the context becomes invalid, the route
		// might keep displaying, and the builder must continue to work in that case.
		return s.defaultPosition(constraints)
	}
	button, overlay := s.cachedButton, s.cachedOverlayBox
	// Check if the cached render objects are available and still attached.
	if button == nil || overlay == nil || !button.Attached() || !overlay.Attached() {
		return s.defaultPosition(constraints)
	}
	w := s.Widget()
	position := w.o.Position
	if position == PopupMenuPositionUnspecified {
		position = s.popupMenuTheme.Position
	}
	offset := w.o.Offset
	if position == PopupMenuPositionUnder {
		offset = ui.Offset{Dy: button.Size().Height}.Add(w.o.Offset)
		if w.o.Child == nil {
			// Remove the padding of the icon button.
			offset = offset.Sub(ui.Offset{Dy: (w.o.Padding.Resolve(ui.TextDirectionLTR).Top + w.o.Padding.Resolve(ui.TextDirectionLTR).Bottom) / 2})
		}
	}
	a := button.LocalToGlobal(offset, overlay)
	b := button.LocalToGlobal(ui.Offset{Dx: button.Size().Width, Dy: button.Size().Height}.Add(offset), overlay)
	rect := ui.Rect{Left: math.Min(a.Dx, b.Dx), Top: math.Min(a.Dy, b.Dy), Right: math.Max(a.Dx, b.Dx), Bottom: math.Max(a.Dy, b.Dy)}
	p := rendering.NewRelativeRectFromRect(rect, ui.Rect{Right: overlay.Size().Width, Bottom: overlay.Size().Height})
	s.lastPosition = foundation.Some(p)
	return p
}

// ShowButtonMenu programmatically opens the menu given by the item builder, if
// it has items.
//
// Flutter: material/popup_menu.dart PopupMenuButtonState.showButtonMenu
func (s *popupMenuButtonState[T]) showButtonMenu() {
	// Ensure the cached render objects are initialized.
	s.updateCachedObjects()
	w := s.Widget()
	items := w.itemBuilder(s.Context())
	// Only show the menu if there is something to show.
	if len(items) == 0 {
		return
	}
	if w.o.OnOpened != nil {
		w.o.OnOpened()
	}
	s.SetState(func() { s.isMenuExpanded = true })
	var initial any
	if v, ok := w.o.InitialValue.Get(); ok {
		initial = v
	}
	route := showMenuRoute(s.Context(), items, ShowMenuOpts{
		Elevation: w.o.Elevation, ShadowColor: w.o.ShadowColor, SurfaceTintColor: w.o.SurfaceTintColor,
		InitialValue: initial, PositionBuilder: s.positionBuilder, Shape: w.o.Shape, MenuPadding: w.o.MenuPadding, Color: w.o.Color,
		Constraints: w.o.Constraints, ClipBehavior: w.o.ClipBehavior, UseRootNavigator: w.o.UseRootNavigator,
		PopUpAnimationStyle: w.o.PopUpAnimationStyle, RouteSettings: w.o.RouteSettings, RequestFocus: w.o.RequestFocus,
	})
	route.PoppedThen(func(result any) {
		if !s.Mounted() {
			return
		}
		s.SetState(func() { s.isMenuExpanded = false })
		if result == nil {
			if f := w.o.OnCanceled; f != nil {
				f()
			}
			return
		}
		if v, ok := result.(T); ok && w.o.OnSelected != nil {
			w.o.OnSelected(v)
		}
	})
}

// adaptiveMoreIcon is the "more" icon of the platform: three dots in a column,
// or in a row on iOS and macOS.
//
// Flutter: material/icons.dart Icons.adaptive.more
func adaptiveMoreIcon() widgets.IconData {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		return Icons.MoreHoriz()
	}
	return Icons.MoreVert()
}

// Build builds an InkWell around the child, or an IconButton.
//
// Flutter: material/popup_menu.dart PopupMenuButtonState.build
func (s *popupMenuButtonState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	iconTheme := widgets.IconThemeOf(bc)
	popupMenuTheme := PopupMenuThemeOf(bc)
	enableFeedback := w.o.EnableFeedback.OrElse(popupMenuTheme.EnableFeedback.OrElse(true))
	DebugCheckHasMaterialLocalizations(bc)
	tooltip := w.o.Tooltip
	if tooltip == "" {
		tooltip = MaterialLocalizationsOf(bc).ShowMenuTooltip()
	}
	var onPressed func()
	if !w.o.Disabled {
		onPressed = s.showButtonMenu
	}
	if w.o.Child != nil {
		child := NewTooltip(NewInkWell(w.o.Child, InkWellOpts{
			BorderRadius:    w.o.BorderRadius,
			OnTap:           onPressed,
			CanRequestFocus: foundation.Some(!w.o.Disabled),
			Radius:          w.o.SplashRadius,
			EnableFeedback:  foundation.Some(enableFeedback),
		}), TooltipOpts{Message: tooltip})
		tapTargetSize := MaterialTapTargetSizeShrinkWrap
		if w.o.Style != nil && w.o.Style.TapTargetSize != MaterialTapTargetSizeUnspecified {
			tapTargetSize = w.o.Style.TapTargetSize
		}
		if tapTargetSize == MaterialTapTargetSizePadded {
			return widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{
				MinWidth: foundation.Some(MinInteractiveDimension), MinHeight: foundation.Some(MinInteractiveDimension)}), child)
		}
		return widgets.NewSemantics(child, widgets.SemanticsOpts{Expanded: foundation.Some(s.isMenuExpanded)})
	}
	icon := w.o.Icon
	if icon == nil {
		icon = widgets.NewIcon(adaptiveMoreIcon())
	}
	return widgets.NewSemantics(NewIconButton(onPressed,
		widgets.NewSemantics(icon, widgets.SemanticsOpts{Expanded: foundation.Some(s.isMenuExpanded)}),
		IconButtonOpts{
			Padding:        w.o.Padding,
			SplashRadius:   w.o.SplashRadius,
			IconSize:       firstOpt(w.o.IconSize, popupMenuTheme.IconSize, iconTheme.Size()),
			Color:          firstOpt(w.o.IconColor, popupMenuTheme.IconColor, iconTheme.Color()),
			Tooltip:        tooltip,
			EnableFeedback: foundation.Some(enableFeedback),
			Style:          w.o.Style,
		}), widgets.SemanticsOpts{})
}
