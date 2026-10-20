// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"reflect"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

const (
	toggleDuration       = 200 * time.Millisecond
	reactionFadeDuration = 50 * time.Millisecond
)

// ToggleableOwner is what a state that embeds a ToggleableStateMixin provides: the
// abstract members of Flutter's ToggleableStateMixin. The state also is the
// vsync of the animations, so it must embed a TickerProvider.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin
type ToggleableOwner interface {
	// SetState schedules a rebuild after running the function.
	SetState(fn func())
	// Context is the build context of the state.
	Context() BuildContext
	// ToggleableOnChanged is the callback that the widget calls when the value
	// changes; nil when the widget is not interactive.
	//
	// Flutter: widgets/toggleable.dart ToggleableStateMixin.onChanged
	ToggleableOnChanged() func(value foundation.Opt[bool])
	// ToggleableValue is the value of the widget: none is the mixed (null) value.
	//
	// Flutter: widgets/toggleable.dart ToggleableStateMixin.value
	ToggleableValue() foundation.Opt[bool]
	// ToggleableTristate is whether the widget cycles through none too.
	//
	// Flutter: widgets/toggleable.dart ToggleableStateMixin.tristate
	ToggleableTristate() bool
}

// ToggleableStateMixinOpts are the optional parameters of ToggleableStateMixin.Init.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin
type ToggleableStateMixinOpts struct {
	// ReactionAnimationDuration is the duration of the radial reaction; the
	// default is 100 ms.
	//
	// Flutter: widgets/toggleable.dart ToggleableStateMixin.reactionAnimationDuration
	ReactionAnimationDuration time.Duration
	// IsInteractive overrides the answer of whether the widget responds to input;
	// the default is whether ToggleableOnChanged is not nil.
	//
	// Flutter: widgets/toggleable.dart ToggleableStateMixin.isInteractive
	IsInteractive func() bool
}

// ToggleableStateMixin has the animations and the gestures that the state of a
// toggleable widget (a checkbox, a radio button, a switch) shares: the position
// of the toggle, the radial reaction, the hover and focus fades, the activation
// by tap or key, and the states for the properties. Embed it in the state, next
// to a TickerProvider, and call Init from InitState and DisposeToggleable from
// Dispose.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin
type ToggleableStateMixin struct {
	owner                   ToggleableOwner
	opts                    ToggleableStateMixinOpts
	positionController      *animation.AnimationController
	position                *animation.CurvedAnimation
	reactionController      *animation.AnimationController
	reaction                *animation.CurvedAnimation
	reactionHoverFade       *animation.CurvedAnimation
	hoverFadeController     *animation.AnimationController
	reactionFocusFade       *animation.CurvedAnimation
	focusFadeController     *animation.AnimationController
	downPosition            foundation.Opt[ui.Offset]
	focused                 bool
	hovering                bool
	reactionAnimationLength time.Duration
	actionMap               ActionMap
	vsync                   scheduler.TickerProvider
}

// Init creates the animations. Call it from InitState; the owner is the state
// itself and the vsync is its TickerProvider (usually the same value).
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.initState
func (m *ToggleableStateMixin) Init(owner ToggleableOwner, vsync scheduler.TickerProvider, opts ...ToggleableStateMixinOpts) {
	m.owner = owner
	m.vsync = vsync
	m.opts = firstOpts(opts)
	m.reactionAnimationLength = m.opts.ReactionAnimationDuration
	if m.reactionAnimationLength == 0 {
		m.reactionAnimationLength = 100 * time.Millisecond
	}
	initial := 1.0
	if v, ok := owner.ToggleableValue().Get(); ok && !v {
		initial = 0.0
	}
	m.positionController = animation.NewAnimationController(vsync, animation.AnimationControllerOpts{
		Duration: foundation.Some(toggleDuration), Value: foundation.Some(initial)})
	m.position = animation.NewCurvedAnimation(m.positionController, animation.Curves.EaseIn, animation.Curves.EaseOut)
	m.reactionController = animation.NewAnimationController(vsync, animation.AnimationControllerOpts{
		Duration: foundation.Some(m.reactionAnimationLength)})
	m.reaction = animation.NewCurvedAnimation(m.reactionController, animation.Curves.FastOutSlowIn)
	fade := 0.0
	if m.hovering || m.focused {
		fade = 1.0
	}
	m.hoverFadeController = animation.NewAnimationController(vsync, animation.AnimationControllerOpts{
		Duration: foundation.Some(reactionFadeDuration), Value: foundation.Some(fade)})
	m.reactionHoverFade = animation.NewCurvedAnimation(m.hoverFadeController, animation.Curves.FastOutSlowIn)
	m.focusFadeController = animation.NewAnimationController(vsync, animation.AnimationControllerOpts{
		Duration: foundation.Some(reactionFadeDuration), Value: foundation.Some(fade)})
	m.reactionFocusFade = animation.NewCurvedAnimation(m.focusFadeController, animation.Curves.FastOutSlowIn)
	m.actionMap = ActionMap{
		reflect.TypeFor[ActivateIntent](): NewCallbackAction(func(ActivateIntent) any { m.handleTap(); return nil }),
	}
}

// PositionController is the controller of the position animation.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.positionController
func (m *ToggleableStateMixin) PositionController() *animation.AnimationController {
	return m.positionController
}

// Position is the animation of the toggle from 0 (off) to 1 (on).
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.position
func (m *ToggleableStateMixin) Position() *animation.CurvedAnimation { return m.position }

// ReactionController is the controller of the radial reaction animation.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.reactionController
func (m *ToggleableStateMixin) ReactionController() *animation.AnimationController {
	return m.reactionController
}

// Reaction is the animation of the radial reaction while the widget is pressed.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.reaction
func (m *ToggleableStateMixin) Reaction() *animation.CurvedAnimation { return m.reaction }

// ReactionHoverFade is the animation of the hover highlight.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.reactionHoverFade
func (m *ToggleableStateMixin) ReactionHoverFade() *animation.CurvedAnimation {
	return m.reactionHoverFade
}

// ReactionFocusFade is the animation of the focus highlight.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.reactionFocusFade
func (m *ToggleableStateMixin) ReactionFocusFade() *animation.CurvedAnimation {
	return m.reactionFocusFade
}

// ReactionAnimationDuration is the duration of the radial reaction.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.reactionAnimationDuration
func (m *ToggleableStateMixin) ReactionAnimationDuration() time.Duration {
	return m.reactionAnimationLength
}

// IsInteractive is whether the widget responds to input.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.isInteractive
func (m *ToggleableStateMixin) IsInteractive() bool {
	if m.opts.IsInteractive != nil {
		return m.opts.IsInteractive()
	}
	return m.owner.ToggleableOnChanged() != nil
}

// AnimateToValue runs the position animation to the value of the widget.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.animateToValue
func (m *ToggleableStateMixin) AnimateToValue() {
	value := m.owner.ToggleableValue()
	if m.owner.ToggleableTristate() {
		if value.IsNone() {
			m.positionController.SetValue(0.0)
		}
		if v, ok := value.Get(); !ok || v {
			m.positionController.Forward()
		} else {
			m.positionController.Reverse()
		}
		return
	}
	if v, _ := value.Get(); v {
		m.positionController.Forward()
	} else {
		m.positionController.Reverse()
	}
}

// DisposeToggleable disposes the animations. Call it from Dispose.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.dispose
func (m *ToggleableStateMixin) DisposeToggleable() {
	m.positionController.Dispose()
	m.position.Dispose()
	m.reactionController.Dispose()
	m.reaction.Dispose()
	m.hoverFadeController.Dispose()
	m.reactionHoverFade.Dispose()
	m.focusFadeController.Dispose()
	m.reactionFocusFade.Dispose()
}

// DownPosition is where the pointer is pressed on the widget, if it is.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.downPosition
func (m *ToggleableStateMixin) DownPosition() foundation.Opt[ui.Offset] { return m.downPosition }

func (m *ToggleableStateMixin) handleTapDown(details gestures.TapDownDetails) {
	if m.IsInteractive() {
		m.owner.SetState(func() { m.downPosition = foundation.Some(details.LocalPosition) })
		m.reactionController.Forward()
	}
}

func (m *ToggleableStateMixin) handleTap() {
	if !m.IsInteractive() {
		return
	}
	onChanged := m.owner.ToggleableOnChanged()
	if onChanged != nil {
		value := m.owner.ToggleableValue()
		switch {
		case value == foundation.Some(false):
			onChanged(foundation.Some(true))
		case value == foundation.Some(true):
			if m.owner.ToggleableTristate() {
				onChanged(foundation.None[bool]())
			} else {
				onChanged(foundation.Some(false))
			}
		default:
			onChanged(foundation.Some(false))
		}
	}
	if ro := m.owner.Context().FindRenderObject(); ro != nil {
		ro.SendSemanticsEvent(semantics.NewTapSemanticEvent())
	}
}

func (m *ToggleableStateMixin) handleTapEnd() {
	if m.downPosition.IsSome() {
		m.owner.SetState(func() { m.downPosition = foundation.None[ui.Offset]() })
	}
	m.reactionController.Reverse()
}

func (m *ToggleableStateMixin) handleFocusHighlightChanged(focused bool) {
	if focused == m.focused {
		return
	}
	m.owner.SetState(func() { m.focused = focused })
	if focused {
		m.focusFadeController.Forward()
	} else {
		m.focusFadeController.Reverse()
	}
}

func (m *ToggleableStateMixin) handleHoverChanged(hovering bool) {
	if hovering == m.hovering {
		return
	}
	m.owner.SetState(func() { m.hovering = hovering })
	if hovering {
		m.hoverFadeController.Forward()
	} else {
		m.hoverFadeController.Reverse()
	}
}

// States are the states of the widget, for resolving its properties.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.states
func (m *ToggleableStateMixin) States() WidgetStates {
	var s WidgetStates
	if !m.IsInteractive() {
		s = s.With(WidgetStateDisabled)
	}
	if m.hovering {
		s = s.With(WidgetStateHovered)
	}
	if m.focused {
		s = s.With(WidgetStateFocused)
	}
	if v, ok := m.owner.ToggleableValue().Get(); !ok || v {
		s = s.With(WidgetStateSelected)
	}
	return s
}

// ToggleableOpts are the optional parameters of BuildToggleable and
// BuildToggleableWithChild.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.buildToggleable
type ToggleableOpts struct {
	// FocusNode is the focus node of the widget; nil for the detector's own.
	FocusNode *FocusNode
	// OnFocusChange is called when the focus of the widget changes.
	OnFocusChange func(focused bool)
	// Autofocus asks for the focus when the widget is built.
	Autofocus bool
	// MouseCursor is the cursor over the widget, resolved with the states; nil
	// for the basic cursor.
	MouseCursor WidgetStateProperty[services.MouseCursor]
}

// BuildToggleable builds the detector of focus, hover, keys and taps around a box
// of the given size that the painter paints.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.buildToggleable
func (m *ToggleableStateMixin) BuildToggleable(size ui.Size, painter rendering.CustomPainter, opts ...ToggleableOpts) Widget {
	return m.BuildToggleableWithChild(NewCustomPaint(nil, CustomPaintOpts{Size: size, Painter: painter}), opts...)
}

// BuildToggleableWithChild builds the detector of focus, hover, keys and taps
// around the child.
//
// Flutter: widgets/toggleable.dart ToggleableStateMixin.buildToggleableWithChild
func (m *ToggleableStateMixin) BuildToggleableWithChild(child Widget, opts ...ToggleableOpts) Widget {
	o := firstOpts(opts)
	interactive := m.IsInteractive()
	var cursor services.MouseCursor = services.SystemMouseCursors.Basic
	if o.MouseCursor != nil {
		cursor = o.MouseCursor.Resolve(m.States())
	}
	var onTapDown gestures.GestureTapDownCallback
	var onTap gestures.GestureTapCallback
	var onTapEnd gestures.GestureTapUpCallback
	var onTapCancel gestures.GestureTapCancelCallback
	if interactive {
		onTapDown = m.handleTapDown
		onTap = m.handleTap
		onTapEnd = func(gestures.TapUpDetails) { m.handleTapEnd() }
		onTapCancel = m.handleTapEnd
	}
	return NewFocusableActionDetector(
		NewGestureDetector(NewSemantics(child, SemanticsOpts{Enabled: foundation.Some(interactive)}), GestureDetectorOpts{
			ExcludeFromSemantics: !interactive,
			OnTapDown:            onTapDown,
			OnTap:                onTap,
			OnTapUp:              onTapEnd,
			OnTapCancel:          onTapCancel,
		}),
		FocusableActionDetectorOpts{
			Actions:              m.actionMap,
			FocusNode:            o.FocusNode,
			Autofocus:            o.Autofocus,
			OnFocusChange:        o.OnFocusChange,
			Disabled:             !interactive,
			OnShowFocusHighlight: m.handleFocusHighlightChanged,
			OnShowHoverHighlight: m.handleHoverChanged,
			MouseCursor:          cursor,
		})
}

// ToggleablePainter is the base of the painters of the toggleable widgets: it has
// the animations and the colors that the radial reaction needs, and notifies
// its listeners when any of them changes. Embed it in the painter and
// implement Paint.
//
// Flutter: widgets/toggleable.dart ToggleablePainter
type ToggleablePainter struct {
	foundation.ChangeNotifier

	position          animation.Animation[float64]
	positionHandle    foundation.ListenerHandle
	reaction          animation.Animation[float64]
	reactionHandle    foundation.ListenerHandle
	reactionFocusFade animation.Animation[float64]
	focusFadeHandle   foundation.ListenerHandle
	reactionHoverFade animation.Animation[float64]
	hoverFadeHandle   foundation.ListenerHandle
	activeColor       ui.Color
	inactiveColor     ui.Color
	inactiveReaction  ui.Color
	reactionColor     ui.Color
	hoverColor        ui.Color
	focusColor        ui.Color
	splashRadius      float64
	downPosition      foundation.Opt[ui.Offset]
	isFocused         bool
	isHovered         bool
	isActive          bool
}

func (p *ToggleablePainter) setAnimation(old *animation.Animation[float64], handle *foundation.ListenerHandle, value animation.Animation[float64]) {
	if *old == value {
		return
	}
	if *old != nil {
		(*old).RemoveListener(*handle)
	}
	*handle = value.AddListener(p.NotifyListeners)
	*old = value
	p.NotifyListeners()
}

func setIfChanged[T comparable](p *ToggleablePainter, field *T, value T) {
	if *field == value {
		return
	}
	*field = value
	p.NotifyListeners()
}

// Position is the animation of the toggle.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.position
func (p *ToggleablePainter) Position() animation.Animation[float64] { return p.position }

// SetPosition sets the animation of the toggle.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.position
func (p *ToggleablePainter) SetPosition(v animation.Animation[float64]) {
	p.setAnimation(&p.position, &p.positionHandle, v)
}

// Reaction is the animation of the radial reaction.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.reaction
func (p *ToggleablePainter) Reaction() animation.Animation[float64] { return p.reaction }

// SetReaction sets the animation of the radial reaction.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.reaction
func (p *ToggleablePainter) SetReaction(v animation.Animation[float64]) {
	p.setAnimation(&p.reaction, &p.reactionHandle, v)
}

// ReactionFocusFade is the animation of the focus highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.reactionFocusFade
func (p *ToggleablePainter) ReactionFocusFade() animation.Animation[float64] {
	return p.reactionFocusFade
}

// SetReactionFocusFade sets the animation of the focus highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.reactionFocusFade
func (p *ToggleablePainter) SetReactionFocusFade(v animation.Animation[float64]) {
	p.setAnimation(&p.reactionFocusFade, &p.focusFadeHandle, v)
}

// ReactionHoverFade is the animation of the hover highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.reactionHoverFade
func (p *ToggleablePainter) ReactionHoverFade() animation.Animation[float64] {
	return p.reactionHoverFade
}

// SetReactionHoverFade sets the animation of the hover highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.reactionHoverFade
func (p *ToggleablePainter) SetReactionHoverFade(v animation.Animation[float64]) {
	p.setAnimation(&p.reactionHoverFade, &p.hoverFadeHandle, v)
}

// ActiveColor is the color of the widget when it is on.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.activeColor
func (p *ToggleablePainter) ActiveColor() ui.Color { return p.activeColor }

// SetActiveColor sets the color of the widget when it is on.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.activeColor
func (p *ToggleablePainter) SetActiveColor(v ui.Color) { setIfChanged(p, &p.activeColor, v) }

// InactiveColor is the color of the widget when it is off.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.inactiveColor
func (p *ToggleablePainter) InactiveColor() ui.Color { return p.inactiveColor }

// SetInactiveColor sets the color of the widget when it is off.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.inactiveColor
func (p *ToggleablePainter) SetInactiveColor(v ui.Color) { setIfChanged(p, &p.inactiveColor, v) }

// InactiveReactionColor is the color of the radial reaction when the widget is
// off.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.inactiveReactionColor
func (p *ToggleablePainter) InactiveReactionColor() ui.Color { return p.inactiveReaction }

// SetInactiveReactionColor sets the color of the radial reaction when the widget
// is off.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.inactiveReactionColor
func (p *ToggleablePainter) SetInactiveReactionColor(v ui.Color) {
	setIfChanged(p, &p.inactiveReaction, v)
}

// ReactionColor is the color of the radial reaction when the widget is on.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.reactionColor
func (p *ToggleablePainter) ReactionColor() ui.Color { return p.reactionColor }

// SetReactionColor sets the color of the radial reaction when the widget is on.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.reactionColor
func (p *ToggleablePainter) SetReactionColor(v ui.Color) { setIfChanged(p, &p.reactionColor, v) }

// HoverColor is the color of the hover highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.hoverColor
func (p *ToggleablePainter) HoverColor() ui.Color { return p.hoverColor }

// SetHoverColor sets the color of the hover highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.hoverColor
func (p *ToggleablePainter) SetHoverColor(v ui.Color) { setIfChanged(p, &p.hoverColor, v) }

// FocusColor is the color of the focus highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.focusColor
func (p *ToggleablePainter) FocusColor() ui.Color { return p.focusColor }

// SetFocusColor sets the color of the focus highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.focusColor
func (p *ToggleablePainter) SetFocusColor(v ui.Color) { setIfChanged(p, &p.focusColor, v) }

// SplashRadius is the radius of the radial reaction.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.splashRadius
func (p *ToggleablePainter) SplashRadius() float64 { return p.splashRadius }

// SetSplashRadius sets the radius of the radial reaction.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.splashRadius
func (p *ToggleablePainter) SetSplashRadius(v float64) { setIfChanged(p, &p.splashRadius, v) }

// DownPosition is where the pointer is pressed, if it is.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.downPosition
func (p *ToggleablePainter) DownPosition() foundation.Opt[ui.Offset] { return p.downPosition }

// SetDownPosition sets where the pointer is pressed.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.downPosition
func (p *ToggleablePainter) SetDownPosition(v foundation.Opt[ui.Offset]) {
	setIfChanged(p, &p.downPosition, v)
}

// IsFocused is whether the widget has the focus highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.isFocused
func (p *ToggleablePainter) IsFocused() bool { return p.isFocused }

// SetIsFocused sets whether the widget has the focus highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.isFocused
func (p *ToggleablePainter) SetIsFocused(v bool) { setIfChanged(p, &p.isFocused, v) }

// IsHovered is whether the widget has the hover highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.isHovered
func (p *ToggleablePainter) IsHovered() bool { return p.isHovered }

// SetIsHovered sets whether the widget has the hover highlight.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.isHovered
func (p *ToggleablePainter) SetIsHovered(v bool) { setIfChanged(p, &p.isHovered, v) }

// IsActive is whether the widget is on.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.isActive
func (p *ToggleablePainter) IsActive() bool { return p.isActive }

// SetIsActive sets whether the widget is on.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.isActive
func (p *ToggleablePainter) SetIsActive(v bool) { setIfChanged(p, &p.isActive, v) }

func colorLerp(a, b ui.Color, t float64) ui.Color { return *ui.ColorLerp(&a, &b, t) }

// PaintRadialReaction paints the radial reaction: the splash of a press, and the
// hover and focus highlights.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.paintRadialReaction
func (p *ToggleablePainter) PaintRadialReaction(canvas *ui.Canvas, offset, origin ui.Offset) {
	dismissed := func(a animation.Animation[float64]) bool { return a.Status() == animation.AnimationStatusDismissed }
	if dismissed(p.reaction) && dismissed(p.reactionFocusFade) && dismissed(p.reactionHoverFade) {
		return
	}
	paint := ui.NewPaint()
	paint.SetColor(colorLerp(
		colorLerp(
			colorLerp(p.inactiveReaction, p.reactionColor, p.position.Value()),
			p.hoverColor, p.reactionHoverFade.Value()),
		p.focusColor, p.reactionFocusFade.Value()))
	radius := p.splashRadius * p.reaction.Value()
	if p.isFocused || p.isHovered {
		radius = p.splashRadius
	}
	if radius > 0.0 {
		canvas.DrawCircle(origin.Add(offset), radius, paint)
	}
}

// Dispose removes the listeners from the animations.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.dispose
func (p *ToggleablePainter) Dispose() {
	if p.position != nil {
		p.position.RemoveListener(p.positionHandle)
	}
	if p.reaction != nil {
		p.reaction.RemoveListener(p.reactionHandle)
	}
	if p.reactionFocusFade != nil {
		p.reactionFocusFade.RemoveListener(p.focusFadeHandle)
	}
	if p.reactionHoverFade != nil {
		p.reactionHoverFade.RemoveListener(p.hoverFadeHandle)
	}
	p.ChangeNotifier.Dispose()
}

// ShouldRepaint is true: the painter notifies its own listeners when it needs to
// paint again.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.shouldRepaint
func (p *ToggleablePainter) ShouldRepaint(rendering.CustomPainter) bool { return true }

// HitTest defers to the default.
//
// Flutter: widgets/toggleable.dart ToggleablePainter.hitTest
func (p *ToggleablePainter) HitTest(ui.Offset) foundation.Opt[bool] { return foundation.None[bool]() }
