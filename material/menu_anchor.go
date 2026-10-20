// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The sizes and durations of the menus.
const (
	// menuDefaultSubmenuIconSize is the default size of the arrow of a menu that
	// has a submenu.
	menuDefaultSubmenuIconSize = 24.0
	// menuLabelItemDefaultSpacing is the default spacing between the leading
	// icon, the label, the trailing icon and the shortcut label of a menu item.
	menuLabelItemDefaultSpacing = 12.0
	// menuLabelItemMinSpacing is the minimum spacing between the leading icon,
	// the label, the trailing icon and the shortcut label of a menu item.
	menuLabelItemMinSpacing = 4.0
	// menuVerticalMinPadding is the minimum vertical spacing on the outside of
	// menus.
	menuVerticalMinPadding = 8.0
	// menuViewPadding is how close to the edge of the safe area the menu will be
	// placed.
	menuViewPadding = 8.0
	// menuTopLevelHorizontalMinPadding is the minimum horizontal spacing on the
	// outside of the top level menu.
	menuTopLevelHorizontalMinPadding = 4.0
	// menuOpeningDuration is the default opening animation duration for menus.
	menuOpeningDuration = 500 * time.Millisecond
	// menuClosingDuration is the default closing animation duration for menus.
	menuClosingDuration = 150 * time.Millisecond
	// menuItemRelativeFadeInDuration is the default fade-in duration of each
	// menu item as a fraction of the opening duration.
	menuItemRelativeFadeInDuration = 1.0 / 2
	// menuItemRelativeFadeOutDuration is the default fade-out duration of each
	// menu item as a fraction of the closing duration.
	menuItemRelativeFadeOutDuration = 1.0 / 3
	// menuItemRelativeFadeOutDelay is the default delay between the start of each
	// menu item's fade-out as a fraction of the closing duration.
	menuItemRelativeFadeOutDelay = 1.0 / 3
)

// The curves of the animations of the menu panel. The height curve follows the
// material web implementation of Easing.emphasized instead of the curve of the
// Material 3 specification, as Flutter does.
var (
	menuPanelHeightForwardCurve  animation.Curve = animation.NewCubic(.3, 0, 0, 1)
	menuPanelHeightReverseCurve  animation.Curve = newTweenCurve(0.35, 1, animation.NewFlippedCurve(Easing.EmphasizedAccelerate))
	menuPanelOpacityForwardCurve animation.Curve = animation.NewInterval(0, 50.0/500)
	menuPanelOpacityReverseCurve animation.Curve = animation.NewFlippedCurve(animation.NewInterval(100.0/150, 150.0/150))
)

// newTweenCurve is a curve that linearly interpolates between two values over a
// given curve: the curve is applied first, and then the result is interpolated
// between begin and end. It differs from an Interval, which changes the
// duration over which the curve is applied.
//
// Flutter: material/menu_anchor.dart _TweenCurve
func newTweenCurve(begin, end float64, curve animation.Curve) animation.Curve {
	return animation.CurveFunc(func(t float64) float64 {
		t = curve.Transform(t)
		return begin + (end-begin)*t
	})
}

// menuTraversalShortcuts are the navigation shortcuts that need to be active
// when menus are open.
//
// Flutter: material/menu_anchor.dart _kMenuTraversalShortcuts
func menuTraversalShortcuts() widgets.ShortcutMap {
	return widgets.ShortcutMap{
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyGameButtonA), Intent: widgets.ActivateIntent{}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyEscape), Intent: widgets.DismissIntent{}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyTab), Intent: widgets.NextFocusIntent{}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyTab, widgets.SingleActivatorOpts{Shift: true}), Intent: widgets.PreviousFocusIntent{}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowDown), Intent: widgets.DirectionalFocusIntent{Direction: widgets.TraversalDirectionDown}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowUp), Intent: widgets.DirectionalFocusIntent{Direction: widgets.TraversalDirectionUp}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowLeft), Intent: widgets.DirectionalFocusIntent{Direction: widgets.TraversalDirectionLeft}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowRight), Intent: widgets.DirectionalFocusIntent{Direction: widgets.TraversalDirectionRight}},
	}
}

// MenuAnchorChildBuilder is the type of the function that builds the widget
// that a MenuAnchor surrounds (MenuAnchorOpts.Builder). The controller can open
// and close the menu; child is the optional MenuAnchorOpts.Child, which the
// function is meant to incorporate in its result.
//
// Flutter: material/menu_anchor.dart MenuAnchorChildBuilder
type MenuAnchorChildBuilder func(bc widgets.BuildContext, controller *widgets.MenuController, child widgets.Widget) widgets.Widget

// menuAnchorScope gives the descendants of an anchor its state, and notifies
// them of the changes of its animation status.
//
// Flutter: material/menu_anchor.dart _MenuAnchorScope
type menuAnchorScope struct {
	widgets.InheritedWidget
	state           *menuAnchorState
	animationStatus animation.AnimationStatus
}

func (w *menuAnchorScope) UpdateShouldNotify(old widgets.Widget) bool {
	// The state of a MenuAnchor does not change.
	return old.(*menuAnchorScope).animationStatus != w.animationStatus
}

// MenuAnchorOpts are the optional parameters of NewMenuAnchor.
//
// Flutter: material/menu_anchor.dart MenuAnchor.new
type MenuAnchorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller is the controller that opens and closes the menu; the anchor
	// makes its own when there is none.
	Controller *widgets.MenuController
	// ChildFocusNode is the focus node of the widget that the anchor surrounds:
	// it gets the focus back when the menu is closed with the keyboard.
	ChildFocusNode *widgets.FocusNode
	// Style is the style of the menu panel; its properties win over the theme.
	Style *MenuStyle
	// AlignmentOffset is the offset of the menu from the alignment of its style
	// in the anchor rectangle.
	AlignmentOffset foundation.Opt[ui.Offset]
	// ReservedPadding is how close to the edge of the safe area the menu will be
	// placed; the default is 8 on every side.
	ReservedPadding painting.EdgeInsetsGeometry
	// LayerLink positions the menu relative to the leader of this link, below
	// its start edge, instead of the anchor rectangle.
	LayerLink *rendering.LayerLink
	// ClipBehavior is how the menu panel is clipped; the default is hardEdge.
	ClipBehavior ui.Clip
	// ConsumeOutsideTap stops a tap outside the menu from doing anything else
	// than closing the menu.
	ConsumeOutsideTap bool
	// OnOpen is called when the menu opens.
	OnOpen func()
	// OnClose is called when the menu closes.
	OnClose func()
	// NoCrossAxisUnconstrained makes the menu panel keep the constraints of the
	// cross axis of the overlay (Flutter's crossAxisUnconstrained: false).
	NoCrossAxisUnconstrained bool
	// UseRootOverlay puts the menu in the root overlay instead of the closest.
	UseRootOverlay bool
	// Animated animates the opening and the closing of the menu.
	Animated bool
	// OnAnimationStatusChanged is called when the animation status of the menu
	// changes; the status is reverse while the menu is closing.
	OnAnimationStatusChanged func(status animation.AnimationStatus)
	// Builder builds the widget that the anchor surrounds; the default is Child.
	Builder MenuAnchorChildBuilder
	// Child is the child that the Builder is given, or the anchor child itself.
	Child widgets.Widget
}

// MenuAnchor is a widget used to mark the "anchor" for a set of submenus,
// defining the rectangle used to position the menu, which can be done either
// with an explicit location, or with an alignment. When creating a menu with
// MenuBar or a SubmenuButton, a MenuAnchor is not needed, since they provide
// their own internally.
//
// The MenuStyle.VisualDensity only affects horizontal padding, and it will
// never make it negative. Vertical padding is not affected at all.
//
// Flutter: material/menu_anchor.dart MenuAnchor
type MenuAnchor struct {
	widgets.StatefulWidget
	menuChildren []widgets.Widget
	o            MenuAnchorOpts
	// isBar makes the anchor the one of a MenuBar.
	isBar bool
}

// NewMenuAnchor creates an anchor whose menu has the given children.
//
// Flutter: material/menu_anchor.dart MenuAnchor.new
func NewMenuAnchor(menuChildren []widgets.Widget, opts ...MenuAnchorOpts) *MenuAnchor {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return &MenuAnchor{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), menuChildren: menuChildren, o: o}
}

// Controller is the controller that was given to the anchor, or nil.
//
// Flutter: material/menu_anchor.dart MenuAnchor.controller
func (w *MenuAnchor) Controller() *widgets.MenuController { return w.o.Controller }

// ChildFocusNode is the focus node of the widget that the anchor surrounds.
//
// Flutter: material/menu_anchor.dart MenuAnchor.childFocusNode
func (w *MenuAnchor) ChildFocusNode() *widgets.FocusNode { return w.o.ChildFocusNode }

// Style is the style of the menu panel, or nil.
//
// Flutter: material/menu_anchor.dart MenuAnchor.style
func (w *MenuAnchor) Style() *MenuStyle { return w.o.Style }

// MenuChildren are the children of the menu.
//
// Flutter: material/menu_anchor.dart MenuAnchor.menuChildren
func (w *MenuAnchor) MenuChildren() []widgets.Widget { return w.menuChildren }

// ConsumeOutsideTap is whether a tap outside the menu is consumed.
//
// Flutter: material/menu_anchor.dart MenuAnchor.consumeOutsideTap
func (w *MenuAnchor) ConsumeOutsideTap() bool { return w.o.ConsumeOutsideTap }

// Animated is whether the opening and the closing of the menu is animated.
//
// Flutter: material/menu_anchor.dart MenuAnchor.animated
func (w *MenuAnchor) Animated() bool { return w.o.Animated }

// CreateState creates the state.
//
// Flutter: material/menu_anchor.dart MenuAnchor.createState
func (w *MenuAnchor) CreateState() widgets.State { return &menuAnchorState{} }

// DebugDescribeChildren is the menu children.
//
// Flutter: material/menu_anchor.dart MenuAnchor.debugDescribeChildren
func (w *MenuAnchor) DebugDescribeChildren() []foundation.DiagnosticsNode {
	return widgetsToDiagnosticsNodes(w.menuChildren)
}

// DebugFillProperties adds the properties of the anchor.
//
// Flutter: material/menu_anchor.dart MenuAnchor.debugFillProperties
func (w *MenuAnchor) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("focusNode", nilIfZero(w.o.ChildFocusNode)))
	p.Add(objectProp("style", nilIfZero(w.o.Style)))
	p.Add(foundation.NewEnumProperty("clipBehavior", foundation.Some(w.o.ClipBehavior)))
	p.Add(objectProp("alignmentOffset", optValue(w.o.AlignmentOffset)))
}

// widgetsToDiagnosticsNodes describes the widgets as the children of another.
func widgetsToDiagnosticsNodes(children []widgets.Widget) []foundation.DiagnosticsNode {
	nodes := make([]foundation.DiagnosticsNode, 0, len(children))
	for _, c := range children {
		nodes = append(nodes, widgets.DescribeWidget(c))
	}
	return nodes
}

// menuAnchorState is the state of a MenuAnchor, and of the anchor of a MenuBar.
//
// Flutter: material/menu_anchor.dart _MenuAnchorState
type menuAnchorState struct {
	widgets.StateOf[*MenuAnchor]
	widgets.SingleTickerProvider

	internalMenuController *widgets.MenuController
	menuScopeNode          *widgets.FocusScopeNode
	animationController    *animation.AnimationController
	heightAnimation        *animation.CurvedAnimation
	opacityAnimation       *animation.CurvedAnimation
	menuChildren           []widgets.Widget
	cachedAnimations       []*animation.CurvedAnimation
	statusHandle           animation.StatusListenerHandle
	barActions             widgets.ActionMap
}

// orientation is the axis along which the items of the menu are laid out.
func (s *menuAnchorState) orientation() painting.Axis {
	if s.Widget().isBar {
		return painting.AxisHorizontal
	}
	return painting.AxisVertical
}

func (s *menuAnchorState) menuController() *widgets.MenuController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	return s.internalMenuController
}

// parent is the state of the anchor that this one is a submenu of, or nil.
func (s *menuAnchorState) parent() *menuAnchorState { return maybeMenuAnchorOf(s.Context()) }

func (s *menuAnchorState) isClosingOrClosed() bool {
	switch s.animationController.Status() {
	case animation.AnimationStatusDismissed, animation.AnimationStatusReverse:
		return true
	}
	return false
}

func (s *menuAnchorState) isClosing() bool {
	return s.animationController.Status() == animation.AnimationStatusReverse
}

// InitState creates the controller of the animation, and the controller of the
// menu if there is none.
//
// Flutter: material/menu_anchor.dart _MenuAnchorState.initState
func (s *menuAnchorState) InitState() {
	s.menuScopeNode = widgets.NewFocusScopeNode()
	s.animationController = animation.NewAnimationController(s)
	s.heightAnimation = animation.NewCurvedAnimation(s.animationController, menuPanelHeightForwardCurve, menuPanelHeightReverseCurve)
	s.opacityAnimation = animation.NewCurvedAnimation(s.animationController, menuPanelOpacityForwardCurve, menuPanelOpacityReverseCurve)
	s.resolveAnimationController()
	s.resolveMenuItems()
	s.statusHandle = s.animationController.AddStatusListener(s.handleAnimationStatusChanged)
	if s.Widget().o.Controller == nil {
		s.internalMenuController = widgets.NewMenuController()
	}
	if s.Widget().isBar {
		s.barActions = widgets.ActionMap{}
		a := widgets.NewDismissMenuAction(s.menuController())
		s.barActions[a.IntentType()] = a
	}
}

// DidUpdateWidget keeps the controller, the items and the animation durations
// up to date.
//
// Flutter: material/menu_anchor.dart _MenuAnchorState.didUpdateWidget
func (s *menuAnchorState) DidUpdateWidget(old *MenuAnchor) {
	if old.o.Controller != s.Widget().o.Controller {
		if s.Widget().o.Controller == nil {
			s.internalMenuController = widgets.NewMenuController()
		} else {
			s.internalMenuController = nil
		}
	}
	if old.o.Animated != s.Widget().o.Animated || !sameWidgets(s.Widget().menuChildren, old.menuChildren) {
		s.resolveMenuItems()
	}
	if old.o.Animated != s.Widget().o.Animated {
		s.resolveAnimationController()
	}
}

// Dispose releases the animations and the focus scope.
//
// Flutter: material/menu_anchor.dart _MenuAnchorState.dispose
func (s *menuAnchorState) Dispose() {
	s.menuChildren = nil
	for _, a := range s.cachedAnimations {
		a.Dispose()
	}
	s.internalMenuController = nil
	s.menuScopeNode.Dispose()
	s.heightAnimation.Dispose()
	s.opacityAnimation.Dispose()
	s.animationController.Stop()
	s.animationController.Dispose()
}

func (s *menuAnchorState) resolveAnimationController() {
	if s.Widget().o.Animated {
		s.animationController.Duration = foundation.Some(menuOpeningDuration)
		s.animationController.ReverseDuration = foundation.Some(menuClosingDuration)
	} else {
		s.animationController.Duration = foundation.Some(time.Duration(0))
		s.animationController.ReverseDuration = foundation.Some(time.Duration(0))
	}
}

// resolveMenuItems wraps the menu children in a fade transition, one after the
// other, when the menu is animated.
func (s *menuAnchorState) resolveMenuItems() {
	s.menuChildren = nil
	for _, a := range s.cachedAnimations {
		a.Dispose()
	}
	s.cachedAnimations = nil
	w := s.Widget()
	itemCount := len(w.menuChildren)
	if itemCount == 0 {
		return
	}
	if !w.o.Animated {
		s.menuChildren = append(s.menuChildren, w.menuChildren...)
		return
	}
	const forwardFinalItemOffset = 1 - menuItemRelativeFadeInDuration
	const reverseFinalItemOffset = 1 - menuItemRelativeFadeOutDuration - menuItemRelativeFadeOutDelay
	forwardProgress, reverseProgress := 0.0, 0.0
	itemFadeInGap, itemFadeOutGap := 0.0, 0.0
	if itemCount > 1 {
		// Spread every item evenly across the remaining time after accounting for
		// the fade in/out durations.
		itemFadeInGap = forwardFinalItemOffset / float64(itemCount-1)
		itemFadeOutGap = reverseFinalItemOffset / float64(itemCount-1)
	}
	for _, child := range w.menuChildren {
		forwardCurve := animation.NewInterval(forwardProgress, forwardProgress+menuItemRelativeFadeInDuration)
		reverseCurve := animation.NewInterval(reverseProgress, reverseProgress+menuItemRelativeFadeOutDuration)
		a := animation.NewCurvedAnimation(s.animationController, forwardCurve, reverseCurve)
		s.cachedAnimations = append(s.cachedAnimations, a)
		s.menuChildren = append(s.menuChildren, widgets.NewFadeTransition(a, child, widgets.FadeTransitionOpts{AlwaysIncludeSemantics: true}))
		forwardProgress += itemFadeInGap
		reverseProgress += itemFadeOutGap
	}
}

func (s *menuAnchorState) handleAnimationStatusChanged(status animation.AnimationStatus) {
	// Rebuild to update isClosedOrClosing and notify dependents of the changes
	// of the animation status.
	s.SetState(func() {})
	if f := s.Widget().o.OnAnimationStatusChanged; f != nil {
		f(status)
	}
}

func (s *menuAnchorState) handleMenuOpenRequest(_ foundation.Opt[ui.Offset], showOverlay foundation.VoidCallback) {
	// If the parent of this menu is closing, submenus should not open. This
	// prevents a submenu calling MenuController.Open after a parent menu has
	// started closing.
	if p := s.parent(); p != nil && p.isClosing() {
		return
	}
	showOverlay()
	if s.animationController.Status().IsForwardOrCompleted() {
		return
	}
	s.animationController.Forward()
}

func (s *menuAnchorState) handleMenuCloseRequest(hideOverlay foundation.VoidCallback) {
	if !s.animationController.Status().IsForwardOrCompleted() {
		return
	}
	s.animationController.Reverse().Then(hideOverlay)
}

// Build builds the raw anchor, in the scope that tells the descendants about
// the state and the animation status of the menu.
//
// Flutter: material/menu_anchor.dart _MenuAnchorState.build
func (s *menuAnchorState) Build(bc widgets.BuildContext) widgets.Widget {
	if s.Widget().isBar {
		return s.buildBar(bc)
	}
	w := s.Widget()
	var builder widgets.RawMenuAnchorChildBuilder
	if w.o.Builder != nil {
		builder = widgets.RawMenuAnchorChildBuilder(w.o.Builder)
	}
	child := widgets.Widget(&menuAnchorScope{
		InheritedWidget: widgets.NewInheritedWidgetBase(widgets.NewRawMenuAnchor(s.menuController(), s.buildOverlay, w.o.Child, widgets.RawMenuAnchorOpts{
			OnOpenRequested:    s.handleMenuOpenRequest,
			OnCloseRequested:   s.handleMenuCloseRequest,
			UseRootOverlay:     w.o.UseRootOverlay,
			OnOpen:             w.o.OnOpen,
			OnClose:            w.o.OnClose,
			ConsumeOutsideTaps: w.o.ConsumeOutsideTap,
			ChildFocusNode:     w.o.ChildFocusNode,
			Builder:            builder,
		})),
		state:           s,
		animationStatus: s.animationController.Status(),
	})
	if w.o.LayerLink == nil {
		return child
	}
	return widgets.NewCompositedTransformTarget(w.o.LayerLink, child)
}

// buildBar builds the anchor of a MenuBar: the panel is always there, and the
// group of anchors handles the dismissing of the menus.
//
// Flutter: material/menu_anchor.dart _MenuBarAnchorState.build
func (s *menuAnchorState) buildBar(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	child := widgets.NewActions(s.barActions,
		widgets.NewShortcuts(menuTraversalShortcuts(),
			newMenuPanel(menuPanelOpts{menuStyle: w.o.Style, clipBehavior: w.o.ClipBehavior, orientation: painting.AxisHorizontal, crossAxisUnconstrained: true, children: w.menuChildren})))
	return &menuAnchorScope{
		InheritedWidget: widgets.NewInheritedWidgetBase(widgets.NewRawMenuAnchorGroup(s.menuController(),
			widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				isOpen, _ := widgets.MenuControllerMaybeIsOpenOf(bc)
				return widgets.NewFocusScope(
					widgets.NewExcludeFocus(child, widgets.ExcludeFocusOpts{NotExcluding: isOpen}),
					widgets.FocusScopeOpts{
						Node:                    s.menuScopeNode,
						SkipTraversal:           foundation.Some(!isOpen),
						CanRequestFocus:         foundation.Some(isOpen),
						DescendantsAreFocusable: foundation.Some(true),
					})
			}))),
		state:           s,
		animationStatus: s.animationController.Status(),
	}
}

func (s *menuAnchorState) buildOverlay(_ widgets.BuildContext, position widgets.RawMenuOverlayInfo) widgets.Widget {
	w := s.Widget()
	excluding := s.isClosingOrClosed()
	// ExcludeSemantics, ExcludeFocus, and IgnorePointer are used to effectively
	// disable all interactions with the menu while it is closing.
	//
	// An animated menu should behave the same as a menu without animations.
	// Focus should be able to move to a menu as soon as it starts to open, and a
	// menu should not be interactive as soon as it starts to close.
	reserved := w.o.ReservedPadding
	if reserved == nil {
		reserved = painting.NewEdgeInsetsAll(menuViewPadding)
	}
	return widgets.NewExcludeSemantics(
		widgets.NewIgnorePointer(
			widgets.NewExcludeFocus(
				&submenu{
					StatelessWidget:        widgets.NewStatelessWidgetBase(),
					fadeAnimation:          s.opacityAnimation,
					heightAnimation:        s.heightAnimation,
					layerLink:              w.o.LayerLink,
					consumeOutsideTaps:     w.o.ConsumeOutsideTap,
					menuScopeNode:          s.menuScopeNode,
					menuStyle:              w.o.Style,
					clipBehavior:           w.o.ClipBehavior,
					menuChildren:           s.menuChildren,
					crossAxisUnconstrained: !w.o.NoCrossAxisUnconstrained,
					menuPosition:           position,
					anchor:                 s,
					alignmentOffset:        w.o.AlignmentOffset.OrElse(ui.Offset{}),
					reservedPadding:        reserved,
				}, widgets.ExcludeFocusOpts{NotExcluding: !excluding}),
			widgets.IgnorePointerOpts{Ignoring: foundation.Some(excluding)}),
		widgets.ExcludeSemanticsOpts{Excluding: foundation.Some(excluding)})
}

// root is the state of the top anchor of the menus that this one is part of.
func (s *menuAnchorState) root() *menuAnchorState {
	anchor := s
	for {
		p := anchor.parent()
		if p == nil {
			return anchor
		}
		anchor = p
	}
}

// focusButton gives the focus to the widget that the anchor surrounds.
func (s *menuAnchorState) focusButton() {
	if s == nil || s.Widget().o.ChildFocusNode == nil {
		return
	}
	s.Widget().o.ChildFocusNode.RequestFocus()
}

func (s *menuAnchorState) scopePolicy() widgets.FocusTraversalPolicy {
	if p := widgets.FocusTraversalGroupMaybeOf(s.menuScopeNode.Context()); p != nil {
		return p
	}
	return widgets.NewReadingOrderTraversalPolicy()
}

// focusFirstMenuItem gives the focus to the first item of the menu.
func (s *menuAnchorState) focusFirstMenuItem() {
	if s == nil || s.menuScopeNode.Context() == nil || !s.menuScopeNode.Context().Mounted() {
		return
	}
	if first := s.scopePolicy().FindFirstFocus(&s.menuScopeNode.FocusNode, true); first != nil {
		first.RequestFocus()
	}
}

// focusLastMenuItem gives the focus to the last item of the menu.
func (s *menuAnchorState) focusLastMenuItem() {
	if s == nil || s.menuScopeNode.Context() == nil || !s.menuScopeNode.Context().Mounted() {
		return
	}
	s.scopePolicy().FindLastFocus(&s.menuScopeNode.FocusNode, true).RequestFocus()
}

// maybeMenuAnchorOf is the state of the closest anchor above the context, or
// nil. It creates no dependency.
func maybeMenuAnchorOf(bc widgets.BuildContext) *menuAnchorState {
	if scope, ok := widgets.GetInherited[*menuAnchorScope](bc); ok {
		return scope.state
	}
	return nil
}

// maybeMenuAnimationStatusOf is the animation status of the closest anchor
// above the context; the context depends on it.
func maybeMenuAnimationStatusOf(bc widgets.BuildContext) (animation.AnimationStatus, bool) {
	if scope, ok := widgets.DependOn[*menuAnchorScope](bc); ok {
		return scope.animationStatus, true
	}
	return animation.AnimationStatusDismissed, false
}

// MenuBarOpts are the optional parameters of NewMenuBar.
//
// Flutter: material/menu_anchor.dart MenuBar.new
type MenuBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Style is the style of the menu bar; its properties win over the theme.
	Style *MenuStyle
	// ClipBehavior is how the menu bar is clipped; the default is no clip.
	ClipBehavior ui.Clip
	// Controller is the controller of the menus of the bar; the bar makes its
	// own when there is none.
	Controller *widgets.MenuController
}

// MenuBar is a menu bar that manages cascading child menus. It is a
// horizontal bar, usually at the top of the window, with SubmenuButtons for the
// top level menus and MenuItemButtons for the other items. A menu is opened by
// a click on its button, and then the neighbors open by hover.
//
// Flutter: material/menu_anchor.dart MenuBar
type MenuBar struct {
	widgets.StatelessWidget
	children []widgets.Widget
	o        MenuBarOpts
}

// NewMenuBar creates a menu bar with the given children.
//
// Flutter: material/menu_anchor.dart MenuBar.new
func NewMenuBar(children []widgets.Widget, opts ...MenuBarOpts) *MenuBar {
	o := firstOpts(opts)
	return &MenuBar{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), children: children, o: o}
}

// Children are the items of the bar.
//
// Flutter: material/menu_anchor.dart MenuBar.children
func (w *MenuBar) Children() []widgets.Widget { return w.children }

// Build builds the anchor of the bar. In debug builds it checks that there is an
// Overlay above.
//
// Flutter: material/menu_anchor.dart MenuBar.build
func (w *MenuBar) Build(bc widgets.BuildContext) widgets.Widget {
	widgets.DebugCheckHasOverlay(bc)
	anchor := NewMenuAnchor(w.children, MenuAnchorOpts{Controller: w.o.Controller, ClipBehavior: w.o.ClipBehavior, Style: w.o.Style})
	if w.o.ClipBehavior == ui.ClipUnspecified {
		anchor.o.ClipBehavior = ui.ClipNone
	}
	anchor.isBar = true
	return anchor
}

// DebugDescribeChildren is the items of the bar.
//
// Flutter: material/menu_anchor.dart MenuBar.debugDescribeChildren
func (w *MenuBar) DebugDescribeChildren() []foundation.DiagnosticsNode {
	return widgetsToDiagnosticsNodes(w.children)
}

// DebugFillProperties adds the style and the clip behavior.
//
// Flutter: material/menu_anchor.dart MenuBar.debugFillProperties
func (w *MenuBar) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("style", nilIfZero(w.o.Style)))
	p.Add(foundation.NewEnumProperty("clipBehavior", foundation.Some(w.o.ClipBehavior)))
}
