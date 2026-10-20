// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// MenuItemButtonOpts are the optional parameters of NewMenuItemButton.
//
// Flutter: material/menu_anchor.dart MenuItemButton.new
type MenuItemButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnPressed is called when the item is activated; nil disables the item.
	OnPressed func()
	// OnHover is called when a pointer enters or exits the item.
	OnHover func(hovering bool)
	// NoRequestFocusOnHover stops the item from taking the focus when a pointer
	// hovers over it (Flutter's requestFocusOnHover: false).
	NoRequestFocusOnHover bool
	// OnFocusChange is called when the focus of the item changes.
	OnFocusChange func(focused bool)
	// FocusNode is the focus node of the item.
	FocusNode *widgets.FocusNode
	// Autofocus says that the item focuses itself when it is built.
	Autofocus bool
	// Shortcut is the shortcut that is shown at the end of the item. It does not
	// activate the item: the app has to bind it.
	Shortcut widgets.MenuSerializableShortcut
	// SemanticsLabel is the label that the semantics of the item has instead of
	// its contents; empty is none.
	SemanticsLabel string
	// Style customizes the appearance of the button; it wins over the
	// MenuButtonTheme and over the default.
	Style *ButtonStyle
	// StatesController is the controller of the widget states of the item.
	StatesController *widgets.WidgetStatesController
	// ClipBehavior is how the content is clipped; the default is no clip.
	ClipBehavior ui.Clip
	// LeadingIcon is the widget at the start of the item.
	LeadingIcon widgets.Widget
	// TrailingIcon is the widget at the end of the item, before the shortcut.
	TrailingIcon widgets.Widget
	// NoCloseOnActivate keeps the menu open when the item is activated
	// (Flutter's closeOnActivate: false).
	NoCloseOnActivate bool
	// OverflowAxis is the axis along which the label of the item can overflow,
	// when the item is not in a menu; the default is horizontal.
	OverflowAxis painting.Axis
	// Child is the label of the item.
	Child widgets.Widget
}

// MenuItemButton is a button for use in a MenuAnchor's submenu, a MenuBar, or
// a SubmenuButton's menu: a TextButton with the look of a menu item, which
// closes its menu when it is activated.
//
// Flutter: material/menu_anchor.dart MenuItemButton
type MenuItemButton struct {
	widgets.StatefulWidget
	o MenuItemButtonOpts
}

// NewMenuItemButton creates a menu item.
//
// Flutter: material/menu_anchor.dart MenuItemButton.new
func NewMenuItemButton(opts ...MenuItemButtonOpts) *MenuItemButton {
	o := firstOpts(opts)
	return &MenuItemButton{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Enabled is whether the item can be activated.
//
// Flutter: material/menu_anchor.dart MenuItemButton.enabled
func (w *MenuItemButton) Enabled() bool { return w.o.OnPressed != nil }

// OnPressed is called when the item is activated.
//
// Flutter: material/menu_anchor.dart MenuItemButton.onPressed
func (w *MenuItemButton) OnPressed() func() { return w.o.OnPressed }

// Child is the label of the item.
//
// Flutter: material/menu_anchor.dart MenuItemButton.child
func (w *MenuItemButton) Child() widgets.Widget { return w.o.Child }

// Shortcut is the shortcut that the item shows.
//
// Flutter: material/menu_anchor.dart MenuItemButton.shortcut
func (w *MenuItemButton) Shortcut() widgets.MenuSerializableShortcut { return w.o.Shortcut }

// CreateState creates the state.
//
// Flutter: material/menu_anchor.dart MenuItemButton.createState
func (w *MenuItemButton) CreateState() widgets.State { return &menuItemButtonState{} }

// DefaultStyleOf is the default style of the item in Material 3.
//
// Flutter: material/menu_anchor.dart MenuItemButton.defaultStyleOf
func (w *MenuItemButton) DefaultStyleOf(bc widgets.BuildContext) *ButtonStyle {
	return menuButtonDefaultsM3(bc)
}

// ThemeStyleOf is the style of the closest MenuButtonTheme, or of the theme.
//
// Flutter: material/menu_anchor.dart MenuItemButton.themeStyleOf
func (w *MenuItemButton) ThemeStyleOf(bc widgets.BuildContext) *ButtonStyle {
	return MenuButtonThemeOf(bc).Style
}

// MenuItemButtonStyleFrom is a convenience that makes a ButtonStyle from simple
// values, as TextButtonStyleFrom does.
//
// Flutter: material/menu_anchor.dart MenuItemButton.styleFrom
func MenuItemButtonStyleFrom(o ButtonStyleFromOpts) *ButtonStyle { return TextButtonStyleFrom(o) }

// DebugFillProperties adds the properties of the item.
//
// Flutter: material/menu_anchor.dart MenuItemButton.debugFillProperties
func (w *MenuItemButton) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewFlagProperty("enabled", foundation.Some(w.Enabled()), foundation.FlagPropertyOpts{IfFalse: "DISABLED"}))
	p.Add(objectProp("style", nilIfZero(w.o.Style)))
	p.Add(objectProp("shortcut", nilIfZero(w.o.Shortcut)))
	p.Add(objectProp("focusNode", nilIfZero(w.o.FocusNode)))
	p.Add(foundation.NewEnumProperty("clipBehavior", foundation.Some(w.o.ClipBehavior), foundation.EnumPropertyOpts{DefaultValue: ui.ClipNone}))
	p.Add(objectProp("statesController", nilIfZero(w.o.StatesController)))
}

type menuItemButtonState struct {
	widgets.StateOf[*MenuItemButton]
	internalFocusNode *widgets.FocusNode
	isHovered         bool
	focusHandle       foundation.ListenerHandle
	focusedNode       *widgets.FocusNode
}

func (s *menuItemButtonState) focusNode() *widgets.FocusNode {
	if n := s.Widget().o.FocusNode; n != nil {
		return n
	}
	return s.internalFocusNode
}

func (s *menuItemButtonState) anchor() *menuAnchorState { return maybeMenuAnchorOf(s.Context()) }

func (s *menuItemButtonState) createInternalFocusNodeIfNeeded() {
	if s.Widget().o.FocusNode == nil {
		s.internalFocusNode = widgets.NewFocusNode()
	}
}

func (s *menuItemButtonState) listen() {
	s.focusedNode = s.focusNode()
	s.focusHandle = s.focusedNode.AddListener(s.handleFocusChange)
}

func (s *menuItemButtonState) InitState() {
	s.createInternalFocusNodeIfNeeded()
	s.listen()
}

func (s *menuItemButtonState) Dispose() {
	s.focusedNode.RemoveListener(s.focusHandle)
	if s.internalFocusNode != nil {
		s.internalFocusNode.Dispose()
		s.internalFocusNode = nil
	}
}

func (s *menuItemButtonState) DidUpdateWidget(old *MenuItemButton) {
	if s.Widget().o.FocusNode != old.o.FocusNode {
		s.focusedNode.RemoveListener(s.focusHandle)
		if s.Widget().o.FocusNode != nil && s.internalFocusNode != nil {
			s.internalFocusNode.Dispose()
			s.internalFocusNode = nil
		}
		s.createInternalFocusNodeIfNeeded()
		s.listen()
	}
}

// Build builds a TextButton with the merged styles of the widget, the theme and
// the defaults, around the label of the item.
//
// Flutter: material/menu_anchor.dart _MenuItemButtonState.build
func (s *menuItemButtonState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	// Since the theme style or default style of the TextButton is not wanted, the
	// styles are merged, in the right order when each type of style exists. Each
	// "StyleOf" function is only called once.
	merged := w.DefaultStyleOf(bc)
	if themeStyle := w.ThemeStyleOf(bc); themeStyle != nil {
		merged = themeStyle.Merge(merged)
	}
	if w.o.Style != nil {
		merged = w.o.Style.Merge(merged)
	}
	overflowAxis := w.o.OverflowAxis
	if a := s.anchor(); a != nil {
		overflowAxis = a.orientation()
	}
	enabled := w.Enabled()
	opts := TextButtonOpts{
		FocusNode:        s.focusNode(),
		Style:            merged,
		Autofocus:        enabled && w.o.Autofocus,
		StatesController: w.o.StatesController,
		ClipBehavior:     w.o.ClipBehavior,
		// Flutter passes a null isSemanticButton, so the item is not a button of
		// the semantics tree.
		IsSemanticButton: foundation.Some(false),
	}
	var onPressed func()
	if enabled {
		onPressed = s.handleSelect
		opts.OnFocusChange = w.o.OnFocusChange
	}
	var child widgets.Widget = NewTextButton(onPressed, newMenuItemLabel(menuItemLabelOpts{
		leadingIcon:    w.o.LeadingIcon,
		shortcut:       w.o.Shortcut,
		semanticsLabel: w.o.SemanticsLabel,
		trailingIcon:   w.o.TrailingIcon,
		hasSubmenu:     false,
		showDecoration: true,
		overflowAxis:   overflowAxis,
		child:          w.o.Child,
	}), opts)
	if platformSupportsAccelerators() && enabled {
		child = NewMenuAcceleratorCallbackBinding(child, MenuAcceleratorCallbackBindingOpts{OnInvoke: s.handleSelect})
	}
	if w.o.OnHover != nil || !w.o.NoRequestFocusOnHover {
		child = widgets.NewMouseRegion(child, widgets.MouseRegionOpts{OnHover: s.handlePointerHover, OnExit: s.handlePointerExit})
	}
	return widgets.NewMergeSemantics(child)
}

func (s *menuItemButtonState) handleFocusChange() {
	if !s.focusNode().HasPrimaryFocus() {
		// Close any child menus of the menu of this item.
		if c := widgets.MenuControllerMaybeOf(s.Context()); c != nil {
			c.CloseChildren()
		}
	}
}

func (s *menuItemButtonState) handlePointerExit(*gestures.PointerExitEvent) {
	if s.isHovered {
		if f := s.Widget().o.OnHover; f != nil {
			f(false)
		}
		s.isHovered = false
	}
}

// handlePointerHover is used instead of TextButton.OnHover and the enter event
// of the MouseRegion, which can't be used without triggering the focus on
// scroll.
func (s *menuItemButtonState) handlePointerHover(*gestures.PointerHoverEvent) {
	if s.isHovered {
		return
	}
	s.isHovered = true
	w := s.Widget()
	if w.o.OnHover != nil {
		w.o.OnHover(true)
	}
	if !w.o.NoRequestFocusOnHover {
		s.focusNode().RequestFocus()
		// Without invalidating the focus policy, switching to directional focus may
		// not originate at this node.
		widgets.FocusTraversalGroupOf(s.Context()).InvalidateScopeData(widgets.FocusScopeOf(s.Context()))
	}
}

func (s *menuItemButtonState) handleSelect() {
	w := s.Widget()
	if !w.o.NoCloseOnActivate {
		if a := s.anchor(); a != nil {
			a.root().menuController().Close()
		}
	}
	// Delay the call to OnPressed until post-frame so that the focus is restored
	// to what it was before the menu was opened before the action is executed.
	binding := widgets.WidgetsBindingOf(s.Context())
	binding.Scheduler().AddPostFrameCallback(func(time.Duration) {
		binding.FocusManager().ApplyFocusChangesIfNeeded()
		if f := s.Widget().o.OnPressed; f != nil {
			f()
		}
	})
}

// CheckboxMenuButtonOpts are the optional parameters of NewCheckboxMenuButton.
//
// Flutter: material/menu_anchor.dart CheckboxMenuButton.new
type CheckboxMenuButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Tristate makes the value of the checkbox cycle between true, false and none.
	Tristate bool
	// IsError makes the checkbox look as if it had an error.
	IsError bool
	// OnHover is called when a pointer enters or exits the item.
	OnHover func(hovering bool)
	// OnFocusChange is called when the focus of the item changes.
	OnFocusChange func(focused bool)
	// FocusNode is the focus node of the item.
	FocusNode *widgets.FocusNode
	// Shortcut is the shortcut that is shown at the end of the item.
	Shortcut widgets.MenuSerializableShortcut
	// Style customizes the appearance of the button.
	Style *ButtonStyle
	// StatesController is the controller of the widget states of the item.
	StatesController *widgets.WidgetStatesController
	// ClipBehavior is how the content is clipped; the default is no clip.
	ClipBehavior ui.Clip
	// TrailingIcon is the widget at the end of the item, before the shortcut.
	TrailingIcon widgets.Widget
	// NoCloseOnActivate keeps the menu open when the item is activated.
	NoCloseOnActivate bool
}

// CheckboxMenuButton is a menu item that combines a checkbox with a
// MenuItemButton. The value of the checkbox is none for the third state of a
// tristate checkbox.
//
// Flutter: material/menu_anchor.dart CheckboxMenuButton
type CheckboxMenuButton struct {
	widgets.StatelessWidget
	value     foundation.Opt[bool]
	onChanged func(value foundation.Opt[bool])
	child     widgets.Widget
	o         CheckboxMenuButtonOpts
}

// NewCheckboxMenuButton creates the item; a nil onChanged disables it.
//
// Flutter: material/menu_anchor.dart CheckboxMenuButton.new
func NewCheckboxMenuButton(value foundation.Opt[bool], onChanged func(value foundation.Opt[bool]), child widgets.Widget, opts ...CheckboxMenuButtonOpts) *CheckboxMenuButton {
	o := firstOpts(opts)
	return &CheckboxMenuButton{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: value, onChanged: onChanged, child: child, o: o}
}

// Enabled is whether the checkbox can be changed.
//
// Flutter: material/menu_anchor.dart CheckboxMenuButton.enabled
func (w *CheckboxMenuButton) Enabled() bool { return w.onChanged != nil }

// Build builds a MenuItemButton with the checkbox as its leading icon.
//
// Flutter: material/menu_anchor.dart CheckboxMenuButton.build
func (w *CheckboxMenuButton) Build(bc widgets.BuildContext) widgets.Widget {
	var onPressed func()
	if w.onChanged != nil {
		onPressed = func() {
			switch {
			case w.value == foundation.Some(false):
				w.onChanged(foundation.Some(true))
			case w.value == foundation.Some(true):
				if w.o.Tristate {
					w.onChanged(foundation.None[bool]())
				} else {
					w.onChanged(foundation.Some(false))
				}
			default:
				w.onChanged(foundation.Some(false))
			}
		}
	}
	return NewMenuItemButton(MenuItemButtonOpts{
		Key:       w.Key(),
		OnPressed: onPressed,
		OnHover:   w.o.OnHover, OnFocusChange: w.o.OnFocusChange, FocusNode: w.o.FocusNode, Style: w.o.Style, Shortcut: w.o.Shortcut,
		StatesController: w.o.StatesController, ClipBehavior: w.o.ClipBehavior, TrailingIcon: w.o.TrailingIcon, NoCloseOnActivate: w.o.NoCloseOnActivate,
		LeadingIcon: widgets.NewExcludeFocus(widgets.NewIgnorePointer(widgets.NewConstrainedBox(
			rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(CheckboxWidth), MaxHeight: foundation.Some(CheckboxWidth)}),
			NewCheckbox(w.value, w.onChanged, CheckboxOpts{Tristate: w.o.Tristate, IsError: w.o.IsError})))),
		Child: w.child,
	})
}

// RadioMenuButtonOpts are the optional parameters of NewRadioMenuButton.
//
// Flutter: material/menu_anchor.dart RadioMenuButton.new
type RadioMenuButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Toggleable lets the radio be unselected by pressing it when it is selected.
	Toggleable bool
	// OnHover is called when a pointer enters or exits the item.
	OnHover func(hovering bool)
	// OnFocusChange is called when the focus of the item changes.
	OnFocusChange func(focused bool)
	// FocusNode is the focus node of the item.
	FocusNode *widgets.FocusNode
	// Shortcut is the shortcut that is shown at the end of the item.
	Shortcut widgets.MenuSerializableShortcut
	// Style customizes the appearance of the button.
	Style *ButtonStyle
	// StatesController is the controller of the widget states of the item.
	StatesController *widgets.WidgetStatesController
	// ClipBehavior is how the content is clipped; the default is no clip.
	ClipBehavior ui.Clip
	// TrailingIcon is the widget at the end of the item, before the shortcut.
	TrailingIcon widgets.Widget
	// NoCloseOnActivate keeps the menu open when the item is activated.
	NoCloseOnActivate bool
}

// RadioMenuButton is a menu item that combines a radio with a MenuItemButton.
//
// Flutter: material/menu_anchor.dart RadioMenuButton
type RadioMenuButton[T comparable] struct {
	widgets.StatelessWidget
	value      T
	groupValue foundation.Opt[T]
	onChanged  func(value foundation.Opt[T])
	child      widgets.Widget
	o          RadioMenuButtonOpts
}

// NewRadioMenuButton creates the item; a nil onChanged disables it. The value
// passed to onChanged is none when a toggleable selected radio is pressed.
//
// Flutter: material/menu_anchor.dart RadioMenuButton.new
func NewRadioMenuButton[T comparable](value T, groupValue foundation.Opt[T], onChanged func(value foundation.Opt[T]), child widgets.Widget, opts ...RadioMenuButtonOpts) *RadioMenuButton[T] {
	o := firstOpts(opts)
	return &RadioMenuButton[T]{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: value, groupValue: groupValue, onChanged: onChanged, child: child, o: o}
}

// Enabled is whether the radio can be changed.
//
// Flutter: material/menu_anchor.dart RadioMenuButton.enabled
func (w *RadioMenuButton[T]) Enabled() bool { return w.onChanged != nil }

// Build builds a MenuItemButton with the radio as its leading icon.
//
// Flutter: material/menu_anchor.dart RadioMenuButton.build
func (w *RadioMenuButton[T]) Build(bc widgets.BuildContext) widgets.Widget {
	var onPressed func()
	if w.onChanged != nil {
		onPressed = func() {
			if w.o.Toggleable && w.groupValue == foundation.Some(w.value) {
				w.onChanged(foundation.None[T]())
				return
			}
			w.onChanged(foundation.Some(w.value))
		}
	}
	return NewMenuItemButton(MenuItemButtonOpts{
		Key:       w.Key(),
		OnPressed: onPressed,
		OnHover:   w.o.OnHover, OnFocusChange: w.o.OnFocusChange, FocusNode: w.o.FocusNode, Style: w.o.Style, Shortcut: w.o.Shortcut,
		StatesController: w.o.StatesController, ClipBehavior: w.o.ClipBehavior, TrailingIcon: w.o.TrailingIcon, NoCloseOnActivate: w.o.NoCloseOnActivate,
		LeadingIcon: widgets.NewExcludeFocus(widgets.NewIgnorePointer(widgets.NewConstrainedBox(
			rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(CheckboxWidth), MaxHeight: foundation.Some(CheckboxWidth)}),
			NewRadio(w.value, RadioOpts[T]{GroupValue: w.groupValue, OnChanged: w.onChanged, Toggleable: w.o.Toggleable})))),
		Child: w.child,
	})
}

// menuItemLabelOpts are the parameters of newMenuItemLabel.
type menuItemLabelOpts struct {
	hasSubmenu     bool
	showDecoration bool
	leadingIcon    widgets.Widget
	trailingIcon   widgets.Widget
	shortcut       widgets.MenuSerializableShortcut
	semanticsLabel string
	overflowAxis   painting.Axis
	submenuIcon    widgets.Widget
	child          widgets.Widget
}

// menuItemLabel is the content of a menu item: the leading icon, the label, the
// trailing icon, the shortcut and the arrow of a submenu, spaced as the
// Material spec says.
//
// Flutter: material/menu_anchor.dart _MenuItemLabel
type menuItemLabel struct {
	widgets.StatelessWidget
	o menuItemLabelOpts
}

func newMenuItemLabel(o menuItemLabelOpts) *menuItemLabel {
	return &menuItemLabel{StatelessWidget: widgets.NewStatelessWidgetBase(), o: o}
}

func (w *menuItemLabel) Build(bc widgets.BuildContext) widgets.Widget {
	o := w.o
	density := ThemeOf(bc).VisualDensity
	horizontalPadding := math.Max(menuLabelItemMinSpacing, menuLabelItemDefaultSpacing+density.Horizontal*2)
	startPadding := func() painting.EdgeInsetsGeometry {
		if o.leadingIcon != nil {
			return painting.EdgeInsetsDirectional{Start: horizontalPadding}
		}
		return painting.EdgeInsetsZero
	}
	var leadings widgets.Widget
	if o.overflowAxis == painting.AxisVertical {
		var rowChildren []widgets.Widget
		if o.leadingIcon != nil {
			rowChildren = append(rowChildren, o.leadingIcon)
		}
		if o.child != nil {
			rowChildren = append(rowChildren, widgets.NewExpanded(widgets.NewClipRect(widgets.NewPadding(startPadding(), o.child))))
		}
		leadings = widgets.NewExpanded(widgets.NewClipRect(widgets.NewRow(rowChildren, widgets.RowOpts{MainAxisSize: rendering.MainAxisSizeMin})))
	} else {
		var rowChildren []widgets.Widget
		if o.leadingIcon != nil {
			rowChildren = append(rowChildren, o.leadingIcon)
		}
		if o.child != nil {
			rowChildren = append(rowChildren, widgets.NewPadding(startPadding(), o.child))
		}
		leadings = widgets.NewRow(rowChildren, widgets.RowOpts{MainAxisSize: rendering.MainAxisSizeMin})
	}
	start := painting.EdgeInsetsDirectional{Start: horizontalPadding}
	children := []widgets.Widget{leadings}
	if o.trailingIcon != nil {
		children = append(children, widgets.NewPadding(start, o.trailingIcon))
	}
	if o.showDecoration && o.shortcut != nil {
		children = append(children, widgets.NewPadding(start, widgets.NewText(MenuShortcutLabel(o.shortcut, MaterialLocalizationsOf(bc)))))
	}
	if o.showDecoration && o.hasSubmenu {
		children = append(children, widgets.NewPadding(start, o.submenuIcon))
	}
	var label widgets.Widget = widgets.NewRow(children, widgets.RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween})
	if o.semanticsLabel != "" {
		label = widgets.NewSemantics(label, widgets.SemanticsOpts{Label: foundation.Some(o.semanticsLabel), ExcludeSemantics: true})
	}
	return label
}

// DebugFillProperties adds the shortcut and the flags of the label.
//
// Flutter: material/menu_anchor.dart _MenuItemLabel.debugFillProperties
func (w *menuItemLabel) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("shortcut", nilIfZero(w.o.shortcut)))
	p.Add(objectProp("hasSubmenu", w.o.hasSubmenu))
	p.Add(objectProp("showDecoration", w.o.showDecoration))
}
