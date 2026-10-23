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
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// SwitchOpts are the optional parameters of NewSwitch.
//
// Flutter: material/switch.dart Switch.new
type SwitchOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ActiveColor is the color of the thumb when the switch is on.
	//
	// Deprecated: Flutter deprecates it after v3.31; use ActiveThumbColor.
	ActiveColor foundation.Opt[ui.Color]
	// ActiveThumbColor is the color of the thumb when the switch is on.
	ActiveThumbColor foundation.Opt[ui.Color]
	// ActiveTrackColor is the color of the track when the switch is on.
	ActiveTrackColor foundation.Opt[ui.Color]
	// InactiveThumbColor is the color of the thumb when the switch is off.
	InactiveThumbColor foundation.Opt[ui.Color]
	// InactiveTrackColor is the color of the track when the switch is off.
	InactiveTrackColor foundation.Opt[ui.Color]
	// ActiveThumbImage is the image on the thumb when the switch is on.
	ActiveThumbImage painting.ImageProvider
	// OnActiveThumbImageError is called when ActiveThumbImage fails to load; it
	// needs ActiveThumbImage.
	OnActiveThumbImageError painting.ImageErrorListener
	// InactiveThumbImage is the image on the thumb when the switch is off.
	InactiveThumbImage painting.ImageProvider
	// OnInactiveThumbImageError is called when InactiveThumbImage fails to load; it
	// needs InactiveThumbImage.
	OnInactiveThumbImageError painting.ImageErrorListener
	// ThumbColor is the color of the thumb, in each state.
	ThumbColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TrackColor is the color of the track, in each state.
	TrackColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TrackOutlineColor is the color of the outline of the track, in each state.
	TrackOutlineColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TrackOutlineWidth is the width of the outline of the track, in each state.
	TrackOutlineWidth widgets.WidgetStateProperty[foundation.Opt[float64]]
	// ThumbIcon is the icon on the thumb, in each state; nil for none.
	ThumbIcon widgets.WidgetStateProperty[*widgets.Icon]
	// MaterialTapTargetSize is the size of the tap target.
	MaterialTapTargetSize MaterialTapTargetSize
	// DragStartBehavior is how the drag of the thumb starts; the default is start.
	DragStartBehavior gestures.DragStartBehavior
	// MouseCursor is the cursor over the switch; it may resolve with the states.
	MouseCursor services.MouseCursor
	// FocusColor is the color of the overlay while the switch has the focus.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the color of the overlay while the pointer is over the switch.
	HoverColor foundation.Opt[ui.Color]
	// OverlayColor is the color of the overlay, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashRadius is the radius of the splash.
	SplashRadius foundation.Opt[float64]
	// FocusNode is the focus node of the switch.
	FocusNode *widgets.FocusNode
	// OnFocusChange is called when the focus of the switch changes.
	OnFocusChange func(focused bool)
	// Autofocus asks for the focus when the switch is built.
	Autofocus bool
	// Padding is the padding around the switch.
	Padding painting.EdgeInsetsGeometry
}

// Switch is a Material Design switch: it toggles the state of a single setting
// on or off. It does not keep its own value: the widget calls OnChanged and shows
// the value that it gets.
//
// Deviation: the adaptive constructor builds the Material switch on iOS and macOS
// too, and the thumb images work with the Material look only; there is no
// Cupertino library (SPEC §12.10).
//
// Flutter: material/switch.dart Switch
type Switch struct {
	widgets.StatelessWidget
	value     bool
	onChanged func(value bool)
	o         SwitchOpts
}

// NewSwitch creates the widget. A nil onChanged disables the switch. In debug
// builds it panics with the tag SwitchThumbImageErrorWithoutImage when an error
// listener has no image.
//
// Flutter: material/switch.dart Switch.new
func NewSwitch(value bool, onChanged func(value bool), opts ...SwitchOpts) *Switch {
	o := firstOpts(opts)
	if foundation.DebugMode && (o.ActiveThumbImage == nil && o.OnActiveThumbImageError != nil ||
		o.InactiveThumbImage == nil && o.OnInactiveThumbImageError != nil) {
		panic(errors.New(tagSwitchThumbImageErrorWithoutImage, "the error listener of a thumb image needs the image"))
	}
	return &Switch{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: value, onChanged: onChanged, o: o}
}

// NewSwitchAdaptive creates a switch that adapts to the platform.
//
// Deviation: it is the same as NewSwitch (SPEC §12.10).
//
// Flutter: material/switch.dart Switch.adaptive
func NewSwitchAdaptive(value bool, onChanged func(value bool), opts ...SwitchOpts) *Switch {
	return NewSwitch(value, onChanged, opts...)
}

// Value is whether the switch is on.
//
// Flutter: material/switch.dart Switch.value
func (w *Switch) Value() bool { return w.value }

// OnChanged is called when the user toggles the switch; nil disables it.
//
// Flutter: material/switch.dart Switch.onChanged
func (w *Switch) OnChanged() func(value bool) { return w.onChanged }

func (w *Switch) switchSize(bc widgets.BuildContext) ui.Size {
	theme := ThemeOf(bc)
	switchTheme := SwitchThemeOf(bc)
	defaults := newSwitchDefaults(bc)
	config := newSwitchConfig(bc)
	tapTarget := w.o.MaterialTapTargetSize
	if tapTarget == MaterialTapTargetSizeUnspecified {
		tapTarget = switchTheme.MaterialTapTargetSize
	}
	if tapTarget == MaterialTapTargetSizeUnspecified {
		tapTarget = theme.MaterialTapTargetSize
	}
	padding := w.o.Padding
	if padding == nil {
		padding = switchTheme.Padding
	}
	if padding == nil {
		padding = defaults.padding
	}
	height := config.switchHeight
	if tapTarget == MaterialTapTargetSizeShrinkWrap {
		height = config.switchHeightCollapsed
	}
	return ui.Size{Width: config.switchWidth + padding.Horizontal(), Height: height + padding.Vertical()}
}

// Build makes the switch.
//
// Flutter: material/switch.dart Switch.build
func (w *Switch) Build(bc widgets.BuildContext) widgets.Widget {
	return &materialSwitch{
		StatefulWidget: widgets.NewStatefulWidgetBase(),
		value:          w.value, onChanged: w.onChanged, size: w.switchSize(bc),
		o:                w.o,
		activeThumbColor: firstOptColor(w.o.ActiveThumbColor, w.o.ActiveColor),
	}
}

// DebugFillProperties adds the value and whether the switch is disabled.
//
// Flutter: material/switch.dart Switch.debugFillProperties
func (w *Switch) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewFlagProperty("value", foundation.Some(w.value), foundation.FlagPropertyOpts{IfTrue: "on", IfFalse: "off", ShowName: true}))
	var onChanged any
	if w.onChanged != nil {
		onChanged = true
	}
	p.Add(foundation.NewObjectFlagProperty("onChanged", onChanged, foundation.ObjectFlagPropertyOpts{IfNull: "disabled"}))
}

// materialSwitch is the stateful part of a Switch.
//
// Flutter: material/switch.dart _MaterialSwitch
type materialSwitch struct {
	widgets.StatefulWidget
	value            bool
	onChanged        func(value bool)
	size             ui.Size
	o                SwitchOpts
	activeThumbColor foundation.Opt[ui.Color]
}

func (w *materialSwitch) CreateState() widgets.State { return &materialSwitchState{} }

type materialSwitchState struct {
	widgets.StateOf[*materialSwitch]
	widgets.TickerProvider
	widgets.ToggleableStateMixin
	painter                *switchPainter
	needsPositionAnimation bool
}

func (s *materialSwitchState) InitState() {
	s.painter = &switchPainter{}
	s.Init(s, s, widgets.ToggleableStateMixinOpts{ReactionAnimationDuration: RadialReactionDuration})
}

func (s *materialSwitchState) DidUpdateWidget(old *materialSwitch) {
	if old.value != s.Widget().value {
		position := s.Position()
		if position.Value() == 0.0 || position.Value() == 1.0 {
			s.updateCurve()
		}
		s.AnimateToValue()
	}
}

func (s *materialSwitchState) Dispose() {
	s.painter.dispose()
	s.DisposeToggleable()
}

func (s *materialSwitchState) ToggleableOnChanged() func(foundation.Opt[bool]) {
	if s.Widget().onChanged == nil {
		return nil
	}
	return s.handleChanged
}
func (s *materialSwitchState) ToggleableValue() foundation.Opt[bool] {
	return foundation.Some(s.Widget().value)
}
func (s *materialSwitchState) ToggleableTristate() bool { return false }

func (s *materialSwitchState) updateCurve() {
	position := s.Position()
	if ThemeOf(s.Context()).UseMaterial3 {
		position.Curve = animation.Curves.EaseOutBack
		position.ReverseCurve = animation.Curves.EaseOutBack.Flipped()
	} else {
		position.Curve = animation.Curves.EaseIn
		position.ReverseCurve = animation.Curves.EaseOut
	}
}

func (s *materialSwitchState) widgetThumbColor() widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		w := s.Widget()
		if states.Has(widgets.WidgetStateDisabled) {
			return w.o.InactiveThumbColor
		}
		if states.Has(widgets.WidgetStateSelected) {
			return w.activeThumbColor
		}
		return w.o.InactiveThumbColor
	})
}

func (s *materialSwitchState) widgetTrackColor() widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		w := s.Widget()
		if states.Has(widgets.WidgetStateSelected) {
			return w.o.ActiveTrackColor
		}
		return w.o.InactiveTrackColor
	})
}

func (s *materialSwitchState) trackInnerLength() float64 {
	config := newSwitchConfig(s.Context())
	trackInnerStart := config.trackHeight / 2.0
	trackInnerEnd := config.trackWidth - trackInnerStart
	return trackInnerEnd - trackInnerStart
}

func (s *materialSwitchState) handleDragStart(gestures.DragStartDetails) {
	if s.IsInteractive() {
		s.ReactionController().Forward()
	}
}

func (s *materialSwitchState) handleDragUpdate(details gestures.DragUpdateDetails) {
	if !s.IsInteractive() {
		return
	}
	position := s.Position()
	position.Curve = animation.Curves.Linear
	position.ReverseCurve = nil
	delta := details.PrimaryDelta.Value() / s.trackInnerLength()
	if widgets.DirectionalityOf(s.Context()) == ui.TextDirectionRTL {
		delta = -delta
	}
	controller := s.PositionController()
	controller.SetValue(controller.Value() + delta)
}

func (s *materialSwitchState) handleDragEnd(gestures.DragEndDetails) {
	w := s.Widget()
	if (s.Position().Value() >= 0.5) != w.value {
		if w.onChanged != nil {
			w.onChanged(!w.value)
		}
		s.SetState(func() { s.needsPositionAnimation = true })
	} else {
		s.AnimateToValue()
	}
	s.ReactionController().Reverse()
}

func (s *materialSwitchState) handleChanged(value foundation.Opt[bool]) {
	if f := s.Widget().onChanged; f != nil {
		f(value.Value())
	}
}

func (s *materialSwitchState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	DebugCheckHasMaterial(bc)
	if s.needsPositionAnimation {
		s.needsPositionAnimation = false
		s.AnimateToValue()
	}

	theme := ThemeOf(bc)
	switchTheme := SwitchThemeOf(bc)
	switchConfig := newSwitchConfig(bc)
	defaults := newSwitchDefaults(bc)
	s.PositionController().Duration = foundation.Some(time.Duration(switchConfig.toggleDuration) * time.Millisecond)

	states := s.States()
	activeStates := states.With(widgets.WidgetStateSelected)
	inactiveStates := states.Without(widgets.WidgetStateSelected)

	thumbColorOf := func(st widgets.WidgetStates) foundation.Opt[ui.Color] {
		return firstOptColor(optColorOf(w.o.ThumbColor, st), s.widgetThumbColor().Resolve(st), optColorOf(switchTheme.ThumbColor, st))
	}
	activeThumbColor := thumbColorOf(activeStates)
	effectiveActiveThumbColor := activeThumbColor.OrElse(defaults.thumbColor(activeStates))
	inactiveThumbColor := thumbColorOf(inactiveStates)
	effectiveInactiveThumbColor := inactiveThumbColor.OrElse(defaults.thumbColor(inactiveStates))

	var effectiveActiveTrackColor ui.Color
	if c, ok := firstOptColor(optColorOf(w.o.TrackColor, activeStates), s.widgetTrackColor().Resolve(activeStates),
		optColorOf(switchTheme.TrackColor, activeStates)).Get(); ok {
		effectiveActiveTrackColor = c
	} else if c, ok := s.widgetThumbColor().Resolve(activeStates).Get(); ok {
		effectiveActiveTrackColor = c.WithAlpha(0x80)
	} else {
		effectiveActiveTrackColor = defaults.trackColor(activeStates)
	}
	trackOutlineColor := func(st widgets.WidgetStates) foundation.Opt[ui.Color] {
		return firstOptColor(optColorOf(w.o.TrackOutlineColor, st), optColorOf(switchTheme.TrackOutlineColor, st), defaults.trackOutlineColor(st))
	}
	trackOutlineWidth := func(st widgets.WidgetStates) foundation.Opt[float64] {
		return firstOptDouble(optDoubleOf(w.o.TrackOutlineWidth, st), optDoubleOf(switchTheme.TrackOutlineWidth, st), defaults.trackOutlineWidth(st))
	}
	effectiveActiveTrackOutlineColor := trackOutlineColor(activeStates)
	effectiveActiveTrackOutlineWidth := trackOutlineWidth(activeStates)
	effectiveInactiveTrackColor := firstOptColor(optColorOf(w.o.TrackColor, inactiveStates), s.widgetTrackColor().Resolve(inactiveStates),
		optColorOf(switchTheme.TrackColor, inactiveStates)).OrElse(defaults.trackColor(inactiveStates))
	effectiveInactiveTrackOutlineColor := trackOutlineColor(inactiveStates)
	effectiveInactiveTrackOutlineWidth := trackOutlineWidth(inactiveStates)

	iconOf := func(p widgets.WidgetStateProperty[*widgets.Icon], st widgets.WidgetStates) *widgets.Icon {
		if p == nil {
			return nil
		}
		return p.Resolve(st)
	}
	effectiveIcon := func(st widgets.WidgetStates) *widgets.Icon {
		if i := iconOf(w.o.ThumbIcon, st); i != nil {
			return i
		}
		return iconOf(switchTheme.ThumbIcon, st)
	}
	effectiveActiveIcon := effectiveIcon(activeStates)
	effectiveInactiveIcon := effectiveIcon(inactiveStates)
	effectiveActiveIconColor := switchConfig.iconColor(activeStates)
	if effectiveActiveIcon != nil && effectiveActiveIcon.Color().IsSome() {
		effectiveActiveIconColor = effectiveActiveIcon.Color().Value()
	}
	effectiveInactiveIconColor := switchConfig.iconColor(inactiveStates)
	if effectiveInactiveIcon != nil && effectiveInactiveIcon.Color().IsSome() {
		effectiveInactiveIconColor = effectiveInactiveIcon.Color().Value()
	}

	focusedStates := states.With(widgets.WidgetStateFocused)
	effectiveFocusOverlayColor := firstOptColor(optColorOf(w.o.OverlayColor, focusedStates), w.o.FocusColor,
		optColorOf(switchTheme.OverlayColor, focusedStates), defaults.overlayColor(focusedStates))
	hoveredStates := states.With(widgets.WidgetStateHovered)
	effectiveHoverOverlayColor := firstOptColor(optColorOf(w.o.OverlayColor, hoveredStates), w.o.HoverColor,
		optColorOf(switchTheme.OverlayColor, hoveredStates), defaults.overlayColor(hoveredStates))

	withAlpha := func(c foundation.Opt[ui.Color]) foundation.Opt[ui.Color] {
		if v, ok := c.Get(); ok {
			return foundation.Some(v.WithAlpha(RadialReactionAlpha))
		}
		return c
	}
	thumbPressed := func(st widgets.WidgetStates) ui.Color {
		return firstOptColor(optColorOf(w.o.ThumbColor, st), s.widgetThumbColor().Resolve(st), optColorOf(switchTheme.ThumbColor, st)).
			OrElse(defaults.thumbColor(st))
	}
	activePressedStates := activeStates.With(widgets.WidgetStatePressed)
	effectiveActivePressedThumbColor := thumbPressed(activePressedStates)
	effectiveActivePressedOverlayColor := firstOptColor(optColorOf(w.o.OverlayColor, activePressedStates),
		optColorOf(switchTheme.OverlayColor, activePressedStates), withAlpha(activeThumbColor), defaults.overlayColor(activePressedStates))
	inactivePressedStates := inactiveStates.With(widgets.WidgetStatePressed)
	effectiveInactivePressedThumbColor := thumbPressed(inactivePressedStates)
	effectiveInactivePressedOverlayColor := firstOptColor(optColorOf(w.o.OverlayColor, inactivePressedStates),
		optColorOf(switchTheme.OverlayColor, inactivePressedStates), withAlpha(inactiveThumbColor), defaults.overlayColor(inactivePressedStates))

	mouseCursor := widgets.WidgetStatePropertyResolveWith(func(st widgets.WidgetStates) services.MouseCursor {
		if w.o.MouseCursor != nil {
			if p, ok := w.o.MouseCursor.(widgets.WidgetStateProperty[services.MouseCursor]); ok {
				if c := p.Resolve(st); c != nil {
					return c
				}
			} else {
				return w.o.MouseCursor
			}
		}
		if switchTheme.MouseCursor != nil {
			if c := switchTheme.MouseCursor.Resolve(st); c != nil {
				return c
			}
		}
		return widgets.WidgetStateMouseCursorClickable.Resolve(st)
	})

	effectiveActiveThumbRadius := switchConfig.activeThumbRadius
	if effectiveActiveIcon != nil {
		effectiveActiveThumbRadius = switchConfig.thumbRadiusWithIcon
	}
	effectiveInactiveThumbRadius := switchConfig.inactiveThumbRadius
	if effectiveInactiveIcon != nil || w.o.InactiveThumbImage != nil {
		effectiveInactiveThumbRadius = switchConfig.thumbRadiusWithIcon
	}
	effectiveSplashRadius := w.o.SplashRadius.OrElse(switchTheme.SplashRadius.OrElse(defaults.splashRadius))

	var engine ui.ParagraphFactory
	if b := widgets.MaybeWidgetsBindingOf(bc); b != nil {
		engine = b.PlatformDispatcher().TextEngine()
	}
	iconTheme := widgets.IconThemeOf(bc)

	p := s.painter
	p.engine = engine
	p.SetPosition(s.Position())
	p.SetReaction(s.Reaction())
	p.SetReactionFocusFade(s.ReactionFocusFade())
	p.SetReactionHoverFade(s.ReactionHoverFade())
	p.SetInactiveReactionColor(effectiveInactivePressedOverlayColor.OrElse(ui.Color{}))
	p.SetReactionColor(effectiveActivePressedOverlayColor.OrElse(ui.Color{}))
	p.SetHoverColor(effectiveHoverOverlayColor.OrElse(ui.Color{}))
	p.SetFocusColor(effectiveFocusOverlayColor.OrElse(ui.Color{}))
	p.SetSplashRadius(effectiveSplashRadius)
	p.SetDownPosition(s.DownPosition())
	p.SetIsFocused(states.Has(widgets.WidgetStateFocused))
	p.SetIsHovered(states.Has(widgets.WidgetStateHovered))
	p.SetActiveColor(effectiveActiveThumbColor)
	p.SetInactiveColor(effectiveInactiveThumbColor)
	p.set(switchPainterValues{
		activePressedColor:        effectiveActivePressedThumbColor,
		inactivePressedColor:      effectiveInactivePressedThumbColor,
		activeThumbImage:          w.o.ActiveThumbImage,
		onActiveThumbImageError:   w.o.OnActiveThumbImageError,
		inactiveThumbImage:        w.o.InactiveThumbImage,
		onInactiveThumbImageError: w.o.OnInactiveThumbImageError,
		activeTrackColor:          effectiveActiveTrackColor,
		activeTrackOutlineColor:   effectiveActiveTrackOutlineColor,
		activeTrackOutlineWidth:   effectiveActiveTrackOutlineWidth,
		inactiveTrackColor:        effectiveInactiveTrackColor,
		inactiveTrackOutlineColor: effectiveInactiveTrackOutlineColor,
		inactiveTrackOutlineWidth: effectiveInactiveTrackOutlineWidth,
		configuration:             widgets.CreateLocalImageConfiguration(bc, foundation.None[ui.Size]()),
		isInteractive:             s.IsInteractive(),
		trackInnerLength:          s.trackInnerLength(),
		textDirection:             widgets.DirectionalityOf(bc),
		surfaceColor:              theme.ColorScheme.Surface(),
		inactiveThumbRadius:       effectiveInactiveThumbRadius,
		activeThumbRadius:         effectiveActiveThumbRadius,
		pressedThumbRadius:        switchConfig.pressedThumbRadius,
		thumbOffset:               switchConfig.thumbOffset,
		trackHeight:               switchConfig.trackHeight,
		trackWidth:                switchConfig.trackWidth,
		activeIconColor:           effectiveActiveIconColor,
		inactiveIconColor:         effectiveInactiveIconColor,
		activeIcon:                effectiveActiveIcon,
		inactiveIcon:              effectiveInactiveIcon,
		iconTheme:                 iconTheme,
		thumbShadow:               switchConfig.thumbShadow,
		transitionalThumbSize:     switchConfig.transitionalThumbSize,
	})
	p.setPositionController(s.PositionController())

	disabledOpacity := 1.0
	var opacity float64 = 1
	if w.onChanged == nil {
		opacity = disabledOpacity
	}
	return widgets.NewSemantics(
		widgets.NewGestureDetector(
			widgets.NewOpacity(opacity,
				s.BuildToggleable(w.size, p, widgets.ToggleableOpts{
					MouseCursor: mouseCursor, FocusNode: w.o.FocusNode, OnFocusChange: w.o.OnFocusChange, Autofocus: w.o.Autofocus,
				})),
			widgets.GestureDetectorOpts{
				ExcludeFromSemantics:   true,
				OnHorizontalDragStart:  s.handleDragStart,
				OnHorizontalDragUpdate: s.handleDragUpdate,
				OnHorizontalDragEnd:    s.handleDragEnd,
				DragStartBehavior:      w.o.DragStartBehavior,
			}),
		widgets.SemanticsOpts{Toggled: foundation.Some(w.value)})
}

// switchPainterValues are the values of a switchPainter that the state sets in
// every build.
type switchPainterValues struct {
	activePressedColor        ui.Color
	inactivePressedColor      ui.Color
	activeThumbImage          painting.ImageProvider
	onActiveThumbImageError   painting.ImageErrorListener
	inactiveThumbImage        painting.ImageProvider
	onInactiveThumbImageError painting.ImageErrorListener
	activeTrackColor          ui.Color
	activeTrackOutlineColor   foundation.Opt[ui.Color]
	activeTrackOutlineWidth   foundation.Opt[float64]
	inactiveTrackColor        ui.Color
	inactiveTrackOutlineColor foundation.Opt[ui.Color]
	inactiveTrackOutlineWidth foundation.Opt[float64]
	configuration             painting.ImageConfiguration
	isInteractive             bool
	trackInnerLength          float64
	textDirection             ui.TextDirection
	surfaceColor              ui.Color
	inactiveThumbRadius       float64
	activeThumbRadius         float64
	pressedThumbRadius        float64
	thumbOffset               foundation.Opt[float64]
	trackHeight               float64
	trackWidth                float64
	activeIconColor           ui.Color
	inactiveIconColor         ui.Color
	activeIcon                *widgets.Icon
	inactiveIcon              *widgets.Icon
	iconTheme                 widgets.IconThemeData
	thumbShadow               []painting.BoxShadow
	transitionalThumbSize     ui.Size
}

// switchPainter paints the track, the thumb and the icon of a Switch.
//
// Flutter: material/switch.dart _SwitchPainter
type switchPainter struct {
	widgets.ToggleablePainter
	engine ui.ParagraphFactory
	v      switchPainterValues

	positionController *animation.AnimationController
	colorAnimation     *animation.CurvedAnimation

	cachedThumbColor         ui.Color
	cachedThumbImage         painting.ImageProvider
	cachedThumbErrorListener bool
	cachedThumbPainter       painting.BoxPainter
	isPainting               bool

	stopPressAnimation         bool
	pressedInactiveThumbRadius foundation.Opt[float64]
	pressedActiveThumbRadius   foundation.Opt[float64]
}

// iconEqual says whether two icons are the same widget for the painter.
func (p *switchPainter) set(v switchPainterValues) {
	if !switchPainterValuesEqual(p.v, v) {
		p.v = v
		p.NotifyListeners()
	}
}

func switchPainterValuesEqual(a, b switchPainterValues) bool {
	return a.activePressedColor == b.activePressedColor && a.inactivePressedColor == b.inactivePressedColor &&
		a.activeThumbImage == b.activeThumbImage && a.inactiveThumbImage == b.inactiveThumbImage &&
		a.activeTrackColor == b.activeTrackColor && a.activeTrackOutlineColor == b.activeTrackOutlineColor &&
		a.activeTrackOutlineWidth == b.activeTrackOutlineWidth && a.inactiveTrackColor == b.inactiveTrackColor &&
		a.inactiveTrackOutlineColor == b.inactiveTrackOutlineColor && a.inactiveTrackOutlineWidth == b.inactiveTrackOutlineWidth &&
		a.configuration == b.configuration && a.isInteractive == b.isInteractive && a.trackInnerLength == b.trackInnerLength &&
		a.textDirection == b.textDirection && a.surfaceColor == b.surfaceColor &&
		a.inactiveThumbRadius == b.inactiveThumbRadius && a.activeThumbRadius == b.activeThumbRadius &&
		a.pressedThumbRadius == b.pressedThumbRadius && a.thumbOffset == b.thumbOffset &&
		a.trackHeight == b.trackHeight && a.trackWidth == b.trackWidth &&
		a.activeIconColor == b.activeIconColor && a.inactiveIconColor == b.inactiveIconColor &&
		a.activeIcon == b.activeIcon && a.inactiveIcon == b.inactiveIcon && a.iconTheme.Equal(b.iconTheme) &&
		foundation.ValuesEqual(a.thumbShadow, b.thumbShadow) && a.transitionalThumbSize == b.transitionalThumbSize
}

func (p *switchPainter) setPositionController(c *animation.AnimationController) {
	if p.positionController == c {
		return
	}
	p.positionController = c
	if p.colorAnimation != nil {
		p.colorAnimation.Dispose()
	}
	p.colorAnimation = animation.NewCurvedAnimation(c, animation.Curves.EaseOut, animation.Curves.EaseIn)
	p.NotifyListeners()
}

func (p *switchPainter) createDefaultThumbDecoration(color ui.Color, image painting.ImageProvider, onError painting.ImageErrorListener) *painting.ShapeDecoration {
	var source painting.DecorationImageSource
	if image != nil {
		source = painting.NewDecorationImage(image, painting.DecorationImageOpts{OnError: onError})
	}
	return painting.NewShapeDecoration(painting.NewStadiumBorder(), painting.ShapeDecorationOpts{
		Color: foundation.Some(color), Image: source, Shadows: p.v.thumbShadow})
}

func (p *switchPainter) handleDecorationChanged() {
	if !p.isPainting {
		p.NotifyListeners()
	}
}

// Paint paints the track, the radial reaction and the thumb.
//
// Flutter: material/switch.dart _SwitchPainter.paint
func (p *switchPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	v := &p.v
	position := p.Position()
	reaction := p.Reaction()
	currentValue := position.Value()

	visualPosition := currentValue
	if v.textDirection == ui.TextDirectionRTL {
		visualPosition = 1.0 - currentValue
	}
	p.stopPressAnimation = reaction.Status() == animation.AnimationStatusReverse && !p.stopPressAnimation

	if !p.stopPressAnimation {
		if reaction.Status() == animation.AnimationStatusCompleted {
			p.pressedInactiveThumbRadius = foundation.Some(ui.LerpDouble(v.inactiveThumbRadius, v.pressedThumbRadius, reaction.Value()))
			p.pressedActiveThumbRadius = foundation.Some(ui.LerpDouble(v.activeThumbRadius, v.pressedThumbRadius, reaction.Value()))
		}
		if currentValue == 0 {
			p.pressedInactiveThumbRadius = foundation.Some(ui.LerpDouble(v.inactiveThumbRadius, v.pressedThumbRadius, reaction.Value()))
			p.pressedActiveThumbRadius = foundation.Some(v.activeThumbRadius)
		}
		if currentValue == 1 {
			p.pressedActiveThumbRadius = foundation.Some(ui.LerpDouble(v.activeThumbRadius, v.pressedThumbRadius, reaction.Value()))
			p.pressedInactiveThumbRadius = foundation.Some(v.inactiveThumbRadius)
		}
	}
	radius := func(r foundation.Opt[float64], def float64) ui.Size {
		x := r.OrElse(def)
		return ui.Size{Width: x * 2, Height: x * 2}
	}
	inactiveThumbSize := radius(p.pressedInactiveThumbRadius, v.inactiveThumbRadius)
	activeThumbSize := radius(p.pressedActiveThumbRadius, v.activeThumbRadius)

	thumbSizeAnimation := func(isForward bool) ui.Size {
		var items []animation.TweenSequenceItem[ui.Size]
		if isForward {
			items = []animation.TweenSequenceItem[ui.Size]{
				{Tween: animation.NewTween(inactiveThumbSize, v.transitionalThumbSize).Chain(animation.NewCurveTween(animation.NewCubic(0.31, 0.00, 0.56, 1.00))), Weight: 11},
				{Tween: animation.NewTween(v.transitionalThumbSize, activeThumbSize).Chain(animation.NewCurveTween(animation.NewCubic(0.20, 0.00, 0.00, 1.00))), Weight: 72},
				{Tween: animation.NewConstantTween(activeThumbSize), Weight: 17},
			}
		} else {
			items = []animation.TweenSequenceItem[ui.Size]{
				{Tween: animation.NewConstantTween(inactiveThumbSize), Weight: 17},
				{Tween: animation.NewTween(inactiveThumbSize, v.transitionalThumbSize).Chain(animation.NewCurveTween(animation.NewCubic(0.20, 0.00, 0.00, 1.00).Flipped())), Weight: 72},
				{Tween: animation.NewTween(v.transitionalThumbSize, activeThumbSize).Chain(animation.NewCurveTween(animation.NewCubic(0.31, 0.00, 0.56, 1.00).Flipped())), Weight: 11},
			}
		}
		return animation.NewTweenSequence(items).Evaluate(p.positionController)
	}

	var thumbSize ui.Size
	if reaction.Status() == animation.AnimationStatusCompleted {
		thumbSize = ui.Size{Width: v.pressedThumbRadius * 2, Height: v.pressedThumbRadius * 2}
	} else if position.Status() == animation.AnimationStatusDismissed || position.Status() == animation.AnimationStatusForward {
		thumbSize = thumbSizeAnimation(true)
	} else {
		thumbSize = thumbSizeAnimation(false)
	}

	inset := 0.0
	if to, ok := v.thumbOffset.Get(); ok {
		inset = 1.0 - math.Abs(currentValue-to)*2.0
	}
	thumbSize = ui.Size{Width: thumbSize.Width - inset, Height: thumbSize.Height - inset}

	colorValue := p.colorAnimation.Value()
	trackColor := lerpColor(v.inactiveTrackColor, v.activeTrackColor, colorValue)
	trackOutlineColor := foundation.None[ui.Color]()
	if ic, ok := v.inactiveTrackOutlineColor.Get(); ok {
		if ac, ok := v.activeTrackOutlineColor.Get(); ok {
			trackOutlineColor = foundation.Some(lerpColor(ic, ac, colorValue))
		}
	}
	trackOutlineWidth := lerpOptDouble(v.inactiveTrackOutlineWidth, v.activeTrackOutlineWidth, colorValue)
	var lerpedThumbColor ui.Color
	switch {
	case reaction.Status() != animation.AnimationStatusDismissed:
		lerpedThumbColor = lerpColor(v.inactivePressedColor, v.activePressedColor, colorValue)
	case p.positionController.Status() == animation.AnimationStatusForward:
		lerpedThumbColor = lerpColor(v.inactivePressedColor, p.ActiveColor(), colorValue)
	case p.positionController.Status() == animation.AnimationStatusReverse:
		lerpedThumbColor = lerpColor(p.InactiveColor(), v.activePressedColor, colorValue)
	default:
		lerpedThumbColor = lerpColor(p.InactiveColor(), p.ActiveColor(), colorValue)
	}
	thumbColor := ui.ColorAlphaBlend(lerpedThumbColor, v.surfaceColor)

	thumbIcon, thumbImage, thumbErrorListener := v.activeIcon, v.activeThumbImage, v.onActiveThumbImageError
	if currentValue < 0.5 {
		thumbIcon, thumbImage, thumbErrorListener = v.inactiveIcon, v.inactiveThumbImage, v.onInactiveThumbImageError
	}

	paint := ui.NewPaint()
	paint.SetColor(trackColor)

	trackPaintOffset := ui.Offset{Dx: (size.Width - v.trackWidth) / 2.0, Dy: (size.Height - v.trackHeight) / 2.0}
	thumbPaintOffset := p.computeThumbPaintOffset(trackPaintOffset, thumbSize, visualPosition)
	radialReactionOrigin := ui.Offset{Dx: thumbPaintOffset.Dx + thumbSize.Height/2, Dy: size.Height / 2}

	p.paintTrackWith(canvas, paint, trackPaintOffset, trackOutlineColor, trackOutlineWidth)
	p.PaintRadialReaction(canvas, ui.Offset{}, radialReactionOrigin)
	p.paintThumbWith(thumbPaintOffset, canvas, colorValue, thumbColor, thumbImage, thumbErrorListener, thumbIcon, thumbSize)
}

func (p *switchPainter) computeThumbPaintOffset(trackPaintOffset ui.Offset, thumbSize ui.Size, visualPosition float64) ui.Offset {
	trackRadius := p.v.trackHeight / 2
	additionalThumbRadius := thumbSize.Height/2 - trackRadius
	horizontalProgress := visualPosition * p.v.trackInnerLength
	thumbHorizontalOffset := trackPaintOffset.Dx + trackRadius - thumbSize.Width/2 + horizontalProgress
	thumbVerticalOffset := trackPaintOffset.Dy - additionalThumbRadius
	return ui.Offset{Dx: thumbHorizontalOffset, Dy: thumbVerticalOffset}
}

func (p *switchPainter) paintTrackWith(canvas *ui.Canvas, paint *ui.Paint, trackPaintOffset ui.Offset, trackOutlineColor foundation.Opt[ui.Color], trackOutlineWidth foundation.Opt[float64]) {
	v := &p.v
	trackRect := ui.NewRectFromLTWH(trackPaintOffset.Dx, trackPaintOffset.Dy, v.trackWidth, v.trackHeight)
	trackRadius := v.trackHeight / 2
	canvas.DrawRRect(ui.NewRRectFromRectAndRadius(trackRect, ui.NewRadiusCircular(trackRadius)), paint)

	if c, ok := trackOutlineColor.Get(); ok {
		outlineRect := ui.NewRectFromLTWH(trackPaintOffset.Dx+1, trackPaintOffset.Dy+1, v.trackWidth-2, v.trackHeight-2)
		outlinePaint := ui.NewPaint()
		outlinePaint.SetStyle(ui.PaintingStyleStroke)
		outlinePaint.SetStrokeWidth(trackOutlineWidth.OrElse(2.0))
		outlinePaint.SetColor(c)
		canvas.DrawRRect(ui.NewRRectFromRectAndRadius(outlineRect, ui.NewRadiusCircular(trackRadius)), outlinePaint)
	}
}

func (p *switchPainter) paintThumbWith(thumbPaintOffset ui.Offset, canvas *ui.Canvas, currentValue float64, thumbColor ui.Color,
	thumbImage painting.ImageProvider, thumbErrorListener painting.ImageErrorListener, thumbIcon *widgets.Icon, thumbSize ui.Size) {
	v := &p.v
	p.isPainting = true
	defer func() { p.isPainting = false }()
	if p.cachedThumbPainter == nil || thumbColor != p.cachedThumbColor || thumbImage != p.cachedThumbImage || (thumbErrorListener != nil) != p.cachedThumbErrorListener {
		p.cachedThumbColor = thumbColor
		p.cachedThumbImage = thumbImage
		p.cachedThumbErrorListener = thumbErrorListener != nil
		if p.cachedThumbPainter != nil {
			p.cachedThumbPainter.Dispose()
		}
		p.cachedThumbPainter = p.createDefaultThumbDecoration(thumbColor, thumbImage, thumbErrorListener).CreateBoxPainter(p.handleDecorationChanged)
	}
	p.cachedThumbPainter.Paint(canvas, thumbPaintOffset, v.configuration.CopyWith(painting.ImageConfigurationCopy{Size: foundation.Some(thumbSize)}))

	if thumbIcon != nil && !thumbIcon.Icon().IsZero() {
		iconColor := lerpColor(v.inactiveIconColor, v.activeIconColor, currentValue)
		iconSize := thumbIcon.Size().OrElse(switchIconSize)
		iconData := thumbIcon.Icon()
		orTheme := func(a, b foundation.Opt[float64]) foundation.Opt[float64] {
			if a.IsSome() {
				return a
			}
			return b
		}
		weight := orTheme(thumbIcon.Weight(), v.iconTheme.Weight())
		fill := orTheme(thumbIcon.Fill(), v.iconTheme.Fill())
		grade := orTheme(thumbIcon.Grade(), v.iconTheme.Grade())
		opticalSize := orTheme(thumbIcon.OpticalSize(), v.iconTheme.OpticalSize())
		shadows := thumbIcon.Shadows()
		if shadows == nil {
			shadows = v.iconTheme.Shadows()
		}
		var variations []ui.FontVariation
		for _, fv := range []struct {
			axis string
			v    foundation.Opt[float64]
		}{{"FILL", fill}, {"wght", weight}, {"GRAD", grade}, {"opsz", opticalSize}} {
			if x, ok := fv.v.Get(); ok {
				variations = append(variations, ui.NewFontVariation(fv.axis, x))
			}
		}
		style := painting.NewTextStyle(painting.TextStyleOpts{
			FontVariations: variations, Color: foundation.Some(iconColor), FontSize: foundation.Some(iconSize),
			Inherit: foundation.Some(false), FontFamily: iconData.FontFamily, Package: iconData.FontPackage,
			FontFamilyFallback: iconData.FontFamilyFallback(), Shadows: shadows,
		})
		tp := painting.NewTextPainter(painting.TextPainterOpts{
			TextEngine: p.engine, TextDirection: v.textDirection,
			Text: painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(string(rune(iconData.CodePoint))), Style: style}),
		})
		tp.Layout(0, math.Inf(1))
		additionalHorizontalOffset := (thumbSize.Width - iconSize) / 2
		additionalVerticalOffset := (thumbSize.Height - iconSize) / 2
		tp.Paint(canvas, thumbPaintOffset.Add(ui.Offset{Dx: additionalHorizontalOffset, Dy: additionalVerticalOffset}))
	}
}

func (p *switchPainter) dispose() {
	if p.cachedThumbPainter != nil {
		p.cachedThumbPainter.Dispose()
		p.cachedThumbPainter = nil
	}
	if p.colorAnimation != nil {
		p.colorAnimation.Dispose()
	}
	p.Dispose()
}

// switchIconSize is the size of the icon of the thumb in Material 3.
//
// Flutter: material/switch.dart _SwitchConfigM3.iconSize
const switchIconSize = 16.0

// switchConfig is the measures and the animation of a Switch.
//
// Flutter: material/switch.dart _SwitchConfig
type switchConfig struct {
	trackHeight, trackWidth                          float64
	switchWidth, switchHeight, switchHeightCollapsed float64
	activeThumbRadius, inactiveThumbRadius           float64
	pressedThumbRadius, thumbRadiusWithIcon          float64
	thumbShadow                                      []painting.BoxShadow
	iconColor                                        func(states widgets.WidgetStates) ui.Color
	thumbOffset                                      foundation.Opt[float64]
	transitionalThumbSize                            ui.Size
	toggleDuration                                   int
	switchMinSize                                    ui.Size
}

func newSwitchConfig(bc widgets.BuildContext) switchConfig {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	if theme.UseMaterial3 {
		c := switchConfig{
			activeThumbRadius: 24.0 / 2, inactiveThumbRadius: 16.0 / 2, pressedThumbRadius: 28.0 / 2, thumbRadiusWithIcon: 24.0 / 2,
			thumbShadow: ElevationToShadow[0], trackHeight: 32.0, trackWidth: 52.0, switchWidth: 52.0,
			transitionalThumbSize: ui.Size{Width: 34, Height: 22}, toggleDuration: 300,
			switchMinSize: ui.Size{Width: MinInteractiveDimension, Height: MinInteractiveDimension - 8.0},
		}
		c.switchHeight = c.switchMinSize.Height + 8.0
		c.switchHeightCollapsed = c.switchMinSize.Height
		c.iconColor = func(s widgets.WidgetStates) ui.Color {
			if s.Has(widgets.WidgetStateDisabled) {
				if s.Has(widgets.WidgetStateSelected) {
					return colors.OnSurface().WithOpacity(0.38)
				}
				return colors.SurfaceContainerHighest().WithOpacity(0.38)
			}
			if s.Has(widgets.WidgetStateSelected) {
				return colors.OnPrimaryContainer()
			}
			return colors.SurfaceContainerHighest()
		}
		return c
	}
	c := switchConfig{
		activeThumbRadius: 10.0, inactiveThumbRadius: 10.0, pressedThumbRadius: 10.0, thumbRadiusWithIcon: 10.0,
		thumbShadow: ElevationToShadow[1], trackHeight: 14.0, trackWidth: 33.0, thumbOffset: foundation.Some(0.5),
		transitionalThumbSize: ui.Size{Width: 20, Height: 20}, toggleDuration: 200,
		switchMinSize: ui.Size{Width: MinInteractiveDimension - 8.0, Height: MinInteractiveDimension - 8.0},
		iconColor:     func(widgets.WidgetStates) ui.Color { return ui.NewColor(0x00000000) },
	}
	c.switchHeight = c.switchMinSize.Height + 8.0
	c.switchHeightCollapsed = c.switchMinSize.Height
	c.switchWidth = c.trackWidth - 2*(c.trackHeight/2.0) + c.switchMinSize.Width
	return c
}

// switchDefaults are the default values of a Switch.
//
// Flutter: material/switch.dart _SwitchDefaultsM2, _SwitchDefaultsM3
type switchDefaults struct {
	thumbColor        func(states widgets.WidgetStates) ui.Color
	trackColor        func(states widgets.WidgetStates) ui.Color
	trackOutlineColor func(states widgets.WidgetStates) foundation.Opt[ui.Color]
	trackOutlineWidth func(states widgets.WidgetStates) foundation.Opt[float64]
	overlayColor      func(states widgets.WidgetStates) foundation.Opt[ui.Color]
	splashRadius      float64
	padding           painting.EdgeInsetsGeometry
}

func newSwitchDefaults(bc widgets.BuildContext) switchDefaults {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	transparent := foundation.Some(ui.NewColor(0x00000000))
	none := foundation.None[ui.Color]()
	if theme.UseMaterial3 {
		return switchDefaults{
			thumbColor: func(s widgets.WidgetStates) ui.Color {
				if s.Has(widgets.WidgetStateDisabled) {
					if s.Has(widgets.WidgetStateSelected) {
						return colors.Surface().WithOpacity(1.0)
					}
					return colors.OnSurface().WithOpacity(0.38)
				}
				if s.Has(widgets.WidgetStateSelected) {
					if s.Has(widgets.WidgetStatePressed) || s.Has(widgets.WidgetStateHovered) || s.Has(widgets.WidgetStateFocused) {
						return colors.PrimaryContainer()
					}
					return colors.OnPrimary()
				}
				if s.Has(widgets.WidgetStatePressed) || s.Has(widgets.WidgetStateHovered) || s.Has(widgets.WidgetStateFocused) {
					return colors.OnSurfaceVariant()
				}
				return colors.Outline()
			},
			trackColor: func(s widgets.WidgetStates) ui.Color {
				if s.Has(widgets.WidgetStateDisabled) {
					if s.Has(widgets.WidgetStateSelected) {
						return colors.OnSurface().WithOpacity(0.12)
					}
					return colors.SurfaceContainerHighest().WithOpacity(0.12)
				}
				if s.Has(widgets.WidgetStateSelected) {
					return colors.Primary()
				}
				return colors.SurfaceContainerHighest()
			},
			trackOutlineColor: func(s widgets.WidgetStates) foundation.Opt[ui.Color] {
				if s.Has(widgets.WidgetStateSelected) {
					return transparent
				}
				if s.Has(widgets.WidgetStateDisabled) {
					return foundation.Some(colors.OnSurface().WithOpacity(0.12))
				}
				return foundation.Some(colors.Outline())
			},
			trackOutlineWidth: func(widgets.WidgetStates) foundation.Opt[float64] { return foundation.Some(2.0) },
			overlayColor: func(s widgets.WidgetStates) foundation.Opt[ui.Color] {
				if s.Has(widgets.WidgetStateSelected) {
					switch {
					case s.Has(widgets.WidgetStatePressed):
						return foundation.Some(colors.Primary().WithOpacity(0.1))
					case s.Has(widgets.WidgetStateHovered):
						return foundation.Some(colors.Primary().WithOpacity(0.08))
					case s.Has(widgets.WidgetStateFocused):
						return foundation.Some(colors.Primary().WithOpacity(0.1))
					}
					return none
				}
				switch {
				case s.Has(widgets.WidgetStatePressed):
					return foundation.Some(colors.OnSurface().WithOpacity(0.1))
				case s.Has(widgets.WidgetStateHovered):
					return foundation.Some(colors.OnSurface().WithOpacity(0.08))
				case s.Has(widgets.WidgetStateFocused):
					return foundation.Some(colors.OnSurface().WithOpacity(0.1))
				}
				return none
			},
			splashRadius: 40.0 / 2,
			padding:      painting.NewEdgeInsetsSymmetric(0, 4),
		}
	}
	isDark := theme.Brightness() == ui.BrightnessDark
	d := switchDefaults{splashRadius: RadialReactionRadius, padding: painting.EdgeInsetsZero}
	d.thumbColor = func(s widgets.WidgetStates) ui.Color {
		if s.Has(widgets.WidgetStateDisabled) {
			if isDark {
				return Colors.Grey.Shade800()
			}
			return Colors.Grey.Shade400()
		}
		if s.Has(widgets.WidgetStateSelected) {
			return colors.Secondary()
		}
		if isDark {
			return Colors.Grey.Shade400()
		}
		return Colors.Grey.Shade50()
	}
	d.trackColor = func(s widgets.WidgetStates) ui.Color {
		if s.Has(widgets.WidgetStateDisabled) {
			if isDark {
				return Colors.White10
			}
			return Colors.Black12
		}
		if s.Has(widgets.WidgetStateSelected) {
			return colors.Secondary().WithAlpha(0x80)
		}
		if isDark {
			return Colors.White30
		}
		return ui.NewColor(0x52000000) // Black with 32% opacity
	}
	d.trackOutlineColor = func(widgets.WidgetStates) foundation.Opt[ui.Color] { return transparent }
	d.trackOutlineWidth = func(widgets.WidgetStates) foundation.Opt[float64] { return foundation.None[float64]() }
	d.overlayColor = func(s widgets.WidgetStates) foundation.Opt[ui.Color] {
		switch {
		case s.Has(widgets.WidgetStatePressed):
			return foundation.Some(d.thumbColor(s).WithAlpha(RadialReactionAlpha))
		case s.Has(widgets.WidgetStateHovered):
			return foundation.Some(theme.HoverColor)
		case s.Has(widgets.WidgetStateFocused):
			return foundation.Some(theme.FocusColor)
		}
		return none
	}
	return d
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSwitchThumbImageErrorWithoutImage = "SwitchThumbImageErrorWithoutImage"
)
