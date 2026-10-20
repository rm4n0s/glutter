// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"reflect"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// SubmenuButtonOpts are the optional parameters of NewSubmenuButton.
//
// Flutter: material/menu_anchor.dart SubmenuButton.new
type SubmenuButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnHover is called when a pointer enters or exits the button.
	OnHover func(hovering bool)
	// OnFocusChange is called when the focus of the button changes.
	OnFocusChange func(focused bool)
	// OnOpen is called when the submenu opens.
	OnOpen func()
	// OnClose is called when the submenu closes.
	OnClose func()
	// Controller is the controller of the submenu; the button makes its own when
	// there is none.
	Controller *widgets.MenuController
	// Style customizes the appearance of the button.
	Style *ButtonStyle
	// MenuStyle is the style of the submenu panel.
	MenuStyle *MenuStyle
	// AlignmentOffset is the offset of the submenu from its alignment.
	AlignmentOffset foundation.Opt[ui.Offset]
	// ClipBehavior is how the submenu panel is clipped; the default is hardEdge.
	ClipBehavior ui.Clip
	// FocusNode is the focus node of the button.
	FocusNode *widgets.FocusNode
	// StatesController is the controller of the widget states of the button; the
	// button only reads it, to know the padding of its submenu.
	StatesController *widgets.WidgetStatesController
	// LeadingIcon is the widget at the start of the button.
	LeadingIcon widgets.Widget
	// TrailingIcon is the widget at the end of the button, before the arrow of
	// the submenu.
	TrailingIcon widgets.Widget
	// SubmenuIcon is the arrow at the end of the button, in each state; it wins
	// over the one of the MenuTheme.
	SubmenuIcon widgets.WidgetStateProperty[widgets.Widget]
	// UseRootOverlay puts the submenu in the root overlay instead of the closest.
	UseRootOverlay bool
	// HoverOpenDelay is how long a pointer hovers over the button before the
	// submenu opens; zero opens it at once. A non-zero delay is not allowed in a
	// top level SubmenuButton of a MenuBar.
	HoverOpenDelay time.Duration
	// Animated animates the opening and the closing of the submenu.
	Animated bool
	// OnAnimationStatusChanged is called when the animation status of the submenu
	// changes.
	OnAnimationStatusChanged func(status animation.AnimationStatus)
}

// SubmenuButton is a menu item that has a submenu: a button that opens a menu
// of other items, and shows an arrow when it is in a menu. It is the button
// of the top level menus of a MenuBar. Its submenu opens on a press, and on
// hover or focus when a sibling menu is already open.
//
// Flutter: material/menu_anchor.dart SubmenuButton
type SubmenuButton struct {
	widgets.StatefulWidget
	menuChildren []widgets.Widget
	child        widgets.Widget
	o            SubmenuButtonOpts
}

// NewSubmenuButton creates a button with the given menu items and label.
//
// Flutter: material/menu_anchor.dart SubmenuButton.new
func NewSubmenuButton(menuChildren []widgets.Widget, child widgets.Widget, opts ...SubmenuButtonOpts) *SubmenuButton {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &SubmenuButton{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), menuChildren: menuChildren, child: child, o: o}
}

// MenuChildren are the items of the submenu.
//
// Flutter: material/menu_anchor.dart SubmenuButton.menuChildren
func (w *SubmenuButton) MenuChildren() []widgets.Widget { return w.menuChildren }

// Child is the label of the button.
//
// Flutter: material/menu_anchor.dart SubmenuButton.child
func (w *SubmenuButton) Child() widgets.Widget { return w.child }

// CreateState creates the state.
//
// Flutter: material/menu_anchor.dart SubmenuButton.createState
func (w *SubmenuButton) CreateState() widgets.State {
	s := &submenuButtonState{isOpenOnFocusEnabled: true, animationStatus: animation.AnimationStatusDismissed}
	s.actions = widgets.ActionMap{reflect.TypeFor[widgets.DirectionalFocusIntent](): newSubmenuDirectionalFocusAction(s)}
	return s
}

// DefaultStyleOf is the default style of the button in Material 3.
//
// Flutter: material/menu_anchor.dart SubmenuButton.defaultStyleOf
func (w *SubmenuButton) DefaultStyleOf(bc widgets.BuildContext) *ButtonStyle {
	return menuButtonDefaultsM3(bc)
}

// ThemeStyleOf is the style of the closest MenuButtonTheme, or of the theme.
//
// Flutter: material/menu_anchor.dart SubmenuButton.themeStyleOf
func (w *SubmenuButton) ThemeStyleOf(bc widgets.BuildContext) *ButtonStyle {
	return MenuButtonThemeOf(bc).Style
}

// SubmenuButtonStyleFrom is a convenience that makes a ButtonStyle from simple
// values, as TextButtonStyleFrom does.
//
// Flutter: material/menu_anchor.dart SubmenuButton.styleFrom
func SubmenuButtonStyleFrom(o ButtonStyleFromOpts) *ButtonStyle { return TextButtonStyleFrom(o) }

// DebugDescribeChildren is the items of the submenu.
//
// Flutter: material/menu_anchor.dart SubmenuButton.debugDescribeChildren
func (w *SubmenuButton) DebugDescribeChildren() []foundation.DiagnosticsNode {
	return widgetsToDiagnosticsNodes(w.menuChildren)
}

// DebugFillProperties adds the properties of the button.
//
// Flutter: material/menu_anchor.dart SubmenuButton.debugFillProperties
func (w *SubmenuButton) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("focusNode", nilIfZero(w.o.FocusNode)))
	p.Add(objectProp("menuStyle", nilIfZero(w.o.MenuStyle)))
	p.Add(objectProp("alignmentOffset", optValue(w.o.AlignmentOffset)))
	p.Add(foundation.NewEnumProperty("clipBehavior", foundation.Some(w.o.ClipBehavior)))
}

type submenuButtonState struct {
	widgets.StateOf[*SubmenuButton]
	actions                widgets.ActionMap
	waitingToFocusMenu     bool
	isOpenOnFocusEnabled   bool
	internalMenuController *widgets.MenuController
	internalFocusNode      *widgets.FocusNode
	anchorKey              *widgets.GlobalKey[widgets.State]
	isHovered              bool
	animationStatus        animation.AnimationStatus
	hoverOpenTimer         *gestures.Timer
	focusedNode            *widgets.FocusNode
	focusHandle            foundation.ListenerHandle
}

func (s *submenuButtonState) menuController() *widgets.MenuController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	return s.internalMenuController
}

func (s *submenuButtonState) parent() *menuAnchorState { return maybeMenuAnchorOf(s.Context()) }

func (s *submenuButtonState) anchorState() *menuAnchorState {
	st, ok := s.anchorKey.CurrentState()
	if !ok {
		return nil
	}
	a, _ := st.(*menuAnchorState)
	return a
}

func (s *submenuButtonState) buttonFocusNode() *widgets.FocusNode {
	if n := s.Widget().o.FocusNode; n != nil {
		return n
	}
	return s.internalFocusNode
}

func (s *submenuButtonState) enabled() bool { return len(s.Widget().menuChildren) > 0 }

func (s *submenuButtonState) listenToButton() {
	s.focusedNode = s.buttonFocusNode()
	s.focusHandle = s.focusedNode.AddListener(s.handleFocusChange)
}

func (s *submenuButtonState) InitState() {
	s.anchorKey = widgets.NewGlobalKey[widgets.State]()
	if s.Widget().o.FocusNode == nil {
		s.internalFocusNode = widgets.NewFocusNode()
	}
	if s.Widget().o.Controller == nil {
		s.internalMenuController = widgets.NewMenuController()
	}
	s.listenToButton()
}

func (s *submenuButtonState) DidChangeDependencies() { s.debugValidateHoverOpenDelay() }

func (s *submenuButtonState) Dispose() {
	s.clearHoverOpenTimer()
	s.focusedNode.RemoveListener(s.focusHandle)
	if s.internalFocusNode != nil {
		s.internalFocusNode.Dispose()
		s.internalFocusNode = nil
	}
}

func (s *submenuButtonState) DidUpdateWidget(old *SubmenuButton) {
	s.debugValidateHoverOpenDelay()
	if s.Widget().o.FocusNode != old.o.FocusNode {
		s.focusedNode.RemoveListener(s.focusHandle)
		if old.o.FocusNode == nil {
			s.internalFocusNode.Dispose()
			s.internalFocusNode = nil
		}
		if s.Widget().o.FocusNode == nil && s.internalFocusNode == nil {
			s.internalFocusNode = widgets.NewFocusNode()
		}
		s.listenToButton()
	}
	if s.Widget().o.Controller != old.o.Controller {
		s.internalMenuController = nil
		if old.o.Controller != nil {
			s.internalMenuController = widgets.NewMenuController()
		}
	}
}

// Build builds a MenuAnchor whose anchor is a TextButton with the look of a menu
// item.
//
// Flutter: material/menu_anchor.dart _SubmenuButtonState.build
func (s *submenuButtonState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	menuPaddingOffset := w.o.AlignmentOffset.OrElse(ui.Offset{})
	menuPadding := s.computeMenuPadding(bc)
	orientation := painting.AxisVertical
	if p := s.parent(); p != nil {
		orientation = p.orientation()
	}
	// Move the submenu over by the size of the menu padding, so that the first
	// menu item aligns with the submenu button that opens it.
	switch {
	case orientation == painting.AxisHorizontal && widgets.DirectionalityOf(bc) == ui.TextDirectionRTL:
		menuPaddingOffset = menuPaddingOffset.Add(ui.Offset{Dx: menuPadding.Right})
	case orientation == painting.AxisHorizontal:
		menuPaddingOffset = menuPaddingOffset.Add(ui.Offset{Dx: -menuPadding.Left})
	default:
		menuPaddingOffset = menuPaddingOffset.Add(ui.Offset{Dy: -menuPadding.Top})
	}

	states := widgets.NewWidgetStates()
	if !s.enabled() {
		states = states.With(widgets.WidgetStateDisabled)
	}
	if s.isHovered {
		states = states.With(widgets.WidgetStateHovered)
	}
	if s.buttonFocusNode().HasFocus() {
		states = states.With(widgets.WidgetStateFocused)
	}
	var submenuIcon widgets.Widget
	if w.o.SubmenuIcon != nil {
		submenuIcon = w.o.SubmenuIcon.Resolve(states)
	}
	if submenuIcon == nil {
		if themed := MenuThemeOf(bc).SubmenuIcon; themed != nil {
			submenuIcon = themed.Resolve(states)
		}
	}
	if submenuIcon == nil {
		// The arrow switches with the text direction.
		submenuIcon = widgets.NewIcon(Icons.ArrowRight(), widgets.IconOpts{Size: foundation.Some(menuDefaultSubmenuIconSize)})
	}

	anchor := NewMenuAnchor(w.menuChildren, MenuAnchorOpts{
		Key:                      s.anchorKey,
		OnAnimationStatusChanged: s.handleAnimationStatusChanged,
		Controller:               s.menuController(),
		ChildFocusNode:           s.buttonFocusNode(),
		AlignmentOffset:          foundation.Some(menuPaddingOffset),
		ClipBehavior:             w.o.ClipBehavior,
		OnClose:                  s.handleClose,
		OnOpen:                   s.handleOpen,
		Style:                    w.o.MenuStyle,
		UseRootOverlay:           w.o.UseRootOverlay,
		Animated:                 w.o.Animated,
		Child:                    w.child,
		Builder: func(bc widgets.BuildContext, controller *widgets.MenuController, child widgets.Widget) widgets.Widget {
			return s.buildButton(bc, controller, child, submenuIcon)
		},
	})
	return widgets.NewActions(s.actions, anchor)
}

func (s *submenuButtonState) buildButton(bc widgets.BuildContext, controller *widgets.MenuController, child, submenuIcon widgets.Widget) widgets.Widget {
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
	toggleShowMenu := func() {
		if !s.Mounted() {
			return
		}
		if s.animationStatus.IsForwardOrCompleted() {
			controller.Close()
		} else {
			controller.Open()
		}
	}
	handlePointerExit := func(*gestures.PointerExitEvent) {
		if s.isHovered {
			if w.o.OnHover != nil {
				w.o.OnHover(false)
			}
			s.isHovered = false
			s.clearHoverOpenTimer()
		}
	}
	// MouseRegion.OnEnter and TextButton.OnHover are called if a button is
	// hovered after scrolling. This interferes with focus traversal and scroll
	// position. MouseRegion.OnHover avoids this issue.
	handlePointerHover := func(*gestures.PointerHoverEvent) {
		if s.isHovered {
			return
		}
		s.isHovered = true
		if w.o.OnHover != nil {
			w.o.OnHover(true)
		}
		root := maybeMenuAnchorOf(bc).root()
		// Don't open the root menu bar menus on hover unless a sibling menu is
		// already open. This means that the user has to first click to open a menu
		// on the menu bar before hovering allows them to traverse it.
		if p := s.parent(); p != nil && p.orientation() == painting.AxisHorizontal && !root.menuController().IsOpen() {
			return
		}
		if s.buttonFocusNode().HasPrimaryFocus() {
			s.clearHoverOpenTimer()
			s.maybeOpenMenuOnHoverOrFocus()
		} else {
			s.buttonFocusNode().RequestFocus()
		}
	}

	enabled := s.enabled()
	showDecoration := true
	if p := s.parent(); p != nil {
		showDecoration = p.orientation() == painting.AxisVertical
	}
	opts := TextButtonOpts{Style: merged, FocusNode: s.buttonFocusNode(), IsSemanticButton: foundation.Some(false)}
	var onPressed func()
	if enabled {
		opts.OnFocusChange = w.o.OnFocusChange
		onPressed = toggleShowMenu
	}
	var result widgets.Widget = widgets.NewMergeSemantics(widgets.NewSemantics(
		NewTextButton(onPressed, newMenuItemLabel(menuItemLabelOpts{
			leadingIcon:    w.o.LeadingIcon,
			trailingIcon:   w.o.TrailingIcon,
			hasSubmenu:     true,
			showDecoration: showDecoration,
			submenuIcon:    submenuIcon,
			child:          child,
		}), opts),
		widgets.SemanticsOpts{Expanded: foundation.Some(enabled && s.animationStatus.IsForwardOrCompleted())}))
	if !enabled {
		return result
	}
	result = widgets.NewMouseRegion(result, widgets.MouseRegionOpts{OnHover: handlePointerHover, OnExit: handlePointerExit})
	if platformSupportsAccelerators() {
		return NewMenuAcceleratorCallbackBinding(result, MenuAcceleratorCallbackBindingOpts{OnInvoke: toggleShowMenu, HasSubmenu: true})
	}
	return result
}

func (s *submenuButtonState) handleAnimationStatusChanged(status animation.AnimationStatus) {
	s.SetState(func() { s.animationStatus = status })
	if f := s.Widget().o.OnAnimationStatusChanged; f != nil {
		f(status)
	}
}

func (s *submenuButtonState) handleClose() {
	// After closing the children of this submenu, this submenu button will regain
	// focus. Because submenu buttons open on focus, this submenu will immediately
	// reopen. To prevent this from happening, focus on SubmenuButtons that do not
	// already have focus is prevented using the isOpenOnFocusEnabled flag. This
	// flag is reset after one frame.
	if !s.buttonFocusNode().HasFocus() {
		s.isOpenOnFocusEnabled = false
		binding := widgets.WidgetsBindingOf(s.Context())
		binding.Scheduler().AddPostFrameCallback(func(time.Duration) {
			binding.FocusManager().ApplyFocusChangesIfNeeded()
			s.isOpenOnFocusEnabled = true
		})
	}
	if f := s.Widget().o.OnClose; f != nil {
		f()
	}
}

func (s *submenuButtonState) handleOpen() {
	if !s.waitingToFocusMenu {
		widgets.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
			if s.Mounted() {
				s.buttonFocusNode().RequestFocus()
				s.waitingToFocusMenu = false
			}
		})
		s.waitingToFocusMenu = true
	}
	// Rebuild with the updated controller.IsOpen value.
	s.SetState(func() {})
	if f := s.Widget().o.OnOpen; f != nil {
		f()
	}
}

func (s *submenuButtonState) computeMenuPadding(bc widgets.BuildContext) painting.EdgeInsets {
	w := s.Widget()
	var insets widgets.WidgetStateProperty[painting.EdgeInsetsGeometry]
	if w.o.MenuStyle != nil {
		insets = w.o.MenuStyle.Padding
	}
	if themed := MenuThemeOf(bc).Style; insets == nil && themed != nil {
		insets = themed.Padding
	}
	if insets == nil {
		insets = menuDefaultsM3(bc).Padding
	}
	states := widgets.NewWidgetStates()
	if w.o.StatesController != nil {
		states = w.o.StatesController.Value()
	}
	return insets.Resolve(states).Resolve(widgets.DirectionalityOf(bc))
}

func (s *submenuButtonState) handleFocusChange() {
	s.clearHoverOpenTimer()
	if !s.buttonFocusNode().HasPrimaryFocus() {
		if a := s.anchorState(); a != nil && !a.menuScopeNode.HasFocus() && s.animationStatus.IsForwardOrCompleted() {
			s.menuController().Close()
		}
		return
	}
	s.maybeOpenMenuOnHoverOrFocus()
}

func (s *submenuButtonState) maybeOpenMenuOnHoverOrFocus() {
	if !s.isOpenOnFocusEnabled {
		return
	}
	if s.menuController().IsOpen() {
		if s.animationStatus != animation.AnimationStatusReverse {
			// If the menu isn't closing, there's no reason to reopen it.
			return
		}
		if s.isHovered {
			// If the button is hovered, avoid reopening the menu that the user
			// deliberately closed.
			return
		}
		if p := s.parent(); p != nil && p.orientation() == painting.AxisHorizontal {
			// Top-level (horizontal) buttons in a menubar will try to reopen when
			// closed, since focus returns to their button.
			return
		}
	}
	if s.Widget().o.HoverOpenDelay == 0 {
		s.menuController().Open()
		return
	}
	s.hoverOpenTimer = widgets.WidgetsBindingOf(s.Context()).Gestures().NewTimer(s.Widget().o.HoverOpenDelay, func() {
		// The menu controller could change, so a method value is not used.
		s.menuController().Open()
	})
}

func (s *submenuButtonState) clearHoverOpenTimer() {
	if s.hoverOpenTimer != nil {
		s.hoverOpenTimer.Cancel()
		s.hoverOpenTimer = nil
	}
}

func (s *submenuButtonState) debugValidateHoverOpenDelay() {
	if !foundation.DebugMode {
		return
	}
	if p := s.parent(); p != nil && p.orientation() == painting.AxisHorizontal && s.Widget().o.HoverOpenDelay > 0 {
		panic(errors.New("SubmenuButtonHoverOpenDelayInMenuBar",
			"a non-zero hoverOpenDelay was used in a top-level SubmenuButton situated in a MenuBar: MenuBar children can only be opened by hover if a sibling SubmenuButton is already open; when the hoverOpenDelay for a SubmenuButton is longer than the closing animation of a sibling SubmenuButton, that sibling will close before this SubmenuButton begins opening, leading to this SubmenuButton never opening",
			"hoverOpenDelay", s.Widget().o.HoverOpenDelay.String()))
	}
}

// submenuDirectionalFocusAction is the action that handles the arrow keys in a
// menu: it moves the focus between the buttons of a menu bar, into and out of
// submenus, before the default directional focus.
//
// Flutter: material/menu_anchor.dart _SubmenuDirectionalFocusAction
type submenuDirectionalFocusAction struct {
	widgets.ContextActionBase
	submenu *submenuButtonState
}

func newSubmenuDirectionalFocusAction(s *submenuButtonState) *submenuDirectionalFocusAction {
	a := &submenuDirectionalFocusAction{submenu: s}
	widgets.InitContextActionBase(&a.ContextActionBase, a)
	return a
}

func (a *submenuDirectionalFocusAction) IntentType() reflect.Type {
	return reflect.TypeFor[widgets.DirectionalFocusIntent]()
}

func (a *submenuDirectionalFocusAction) InvokeWithContext(intent widgets.Intent, _ widgets.BuildContext) any {
	i := intent.(widgets.DirectionalFocusIntent)
	sub := a.submenu
	parent := sub.parent()
	ltr := widgets.DirectionalityOf(sub.Context()) == ui.TextDirectionLTR
	button := sub.buttonFocusNode()
	isSubmenu := button.HasPrimaryFocus()
	anchorState := sub.anchorState()
	controller := sub.menuController()
	// toClosing and toOpening are whether the direction is the one that leaves a
	// submenu, or enters one, in the text direction.
	toClosing := (ltr && i.Direction == widgets.TraversalDirectionLeft) || (!ltr && i.Direction == widgets.TraversalDirectionRight)
	toOpening := (ltr && i.Direction == widgets.TraversalDirectionRight) || (!ltr && i.Direction == widgets.TraversalDirectionLeft)

	if parent != nil {
		switch parent.orientation() {
		case painting.AxisHorizontal:
			switch {
			case toClosing:
				// Focus this MenuBar SubmenuButton, then move focus to the
				// previous focusable MenuBar item.
				button.RequestFocus()
				button.PreviousFocus()
				return nil
			case toOpening:
				// Focus this MenuBar SubmenuButton, then move focus to the next
				// focusable MenuBar item.
				button.RequestFocus()
				button.NextFocus()
				return nil
			case i.Direction == widgets.TraversalDirectionDown:
				if isSubmenu {
					// If this is a top-level (horizontal) button in a menubar,
					// focus the first item in this button's submenu.
					anchorState.focusFirstMenuItem()
					return nil
				}
			case i.Direction == widgets.TraversalDirectionUp:
				if isSubmenu {
					// If this is a top-level (horizontal) button in a menubar,
					// focus the last item in this button's submenu. This makes
					// navigating into upward-oriented submenus more intuitive.
					anchorState.focusLastMenuItem()
					return nil
				}
			}
		case painting.AxisVertical:
			switch {
			case toClosing:
				if grand := parent.parent(); grand != nil && grand.orientation() == painting.AxisHorizontal {
					if isSubmenu {
						if node := parent.Widget().o.ChildFocusNode; node != nil {
							node.RequestFocus()
							node.PreviousFocus()
						}
					} else {
						// MenuBar SubmenuButton => SubmenuButton => child. Focus the
						// parent SubmenuButton anchor attached to this child.
						anchorState.focusButton()
					}
				} else {
					if isSubmenu {
						if parent.parent() == nil {
							// Moving in the closing direction while focused on a
							// SubmenuButton within a root MenuAnchor menu should not
							// close the menu.
							return nil
						}
						parent.focusButton()
						parent.menuController().Close()
					} else {
						// If focus is not on a submenu button, closing the anchor
						// this item presides in will close the menu and focus the
						// anchor button.
						controller.Close()
					}
				}
				return nil
			case toOpening && isSubmenu:
				if controller.IsOpen() {
					anchorState.focusFirstMenuItem()
				} else {
					controller.Open()
					widgets.WidgetsBindingOf(sub.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
						if controller.IsOpen() {
							anchorState.focusFirstMenuItem()
						}
					})
				}
				return nil
			}
		}
	}
	widgets.ActionsMaybeInvoke(sub.Context(), intent)
	return nil
}
