// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	radioOuterRadius = 8.0
	radioInnerRadius = 4.5
)

// RadioOpts are the optional parameters of NewRadio.
//
// Flutter: material/radio.dart Radio.new
type RadioOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// GroupValue is the currently selected value for a group of radio buttons.
	//
	// Deprecated: Flutter deprecates it after v3.32; use a RadioGroup above the
	// radios to manage the value of the group.
	GroupValue foundation.Opt[T]
	// OnChanged is called when the user selects this radio.
	//
	// Deprecated: Flutter deprecates it after v3.32; use a RadioGroup above the
	// radios to handle changes.
	OnChanged func(value foundation.Opt[T])
	// MouseCursor is the cursor over the radio; it may resolve with the states of
	// the radio.
	MouseCursor services.MouseCursor
	// Toggleable makes the radio deselect when it is selected again.
	Toggleable bool
	// ActiveColor is the color of the radio when it is selected.
	ActiveColor foundation.Opt[ui.Color]
	// FillColor is the color of the radio, in each state.
	FillColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// FocusColor is the color of the overlay while the radio has the focus.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the color of the overlay while the pointer is over the radio.
	HoverColor foundation.Opt[ui.Color]
	// OverlayColor is the color of the overlay, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashRadius is the radius of the splash.
	SplashRadius foundation.Opt[float64]
	// MaterialTapTargetSize is the size of the tap target.
	MaterialTapTargetSize MaterialTapTargetSize
	// VisualDensity is how compact the layout is.
	VisualDensity foundation.Opt[VisualDensity]
	// FocusNode is the focus node of the radio.
	FocusNode *widgets.FocusNode
	// Autofocus asks for the focus when the radio is built.
	Autofocus bool
	// Enabled says whether the radio responds to input; the default is whether it
	// has an OnChanged, a GroupRegistry or a RadioGroup above.
	Enabled foundation.Opt[bool]
	// GroupRegistry is the registry of the group of the radio; the default is the
	// one of the RadioGroup above.
	GroupRegistry widgets.RadioGroupRegistry[T]
	// BackgroundColor is the color of the background of the radio, in each state.
	BackgroundColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// Side is the border of the radio when it is not selected.
	Side foundation.Opt[painting.BorderSide]
	// StateSide is the border of the radio in each state (Flutter's
	// WidgetStateBorderSide); when it is set, Side is ignored.
	StateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// InnerRadius is the radius of the inner circle, in each state.
	InnerRadius widgets.WidgetStateProperty[foundation.Opt[float64]]
}

// Radio is a Material Design radio button: it lets the user select one option of
// a set. Put the radios below a RadioGroup, which keeps the selected value.
//
// Deviation: the adaptive constructor builds the Material radio on iOS and macOS
// too, as there is no Cupertino library (SPEC §12.10).
//
// Flutter: material/radio.dart Radio
type Radio[T comparable] struct {
	widgets.StatefulWidget
	value T
	o     RadioOpts[T]
}

// NewRadio creates the widget.
//
// Flutter: material/radio.dart Radio.new
func NewRadio[T comparable](value T, opts ...RadioOpts[T]) *Radio[T] {
	o := firstOpts(opts)
	return &Radio[T]{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: value, o: o}
}

// NewRadioAdaptive creates a radio that adapts to the platform.
//
// Deviation: it is the same as NewRadio (SPEC §12.10).
//
// Flutter: material/radio.dart Radio.adaptive
func NewRadioAdaptive[T comparable](value T, opts ...RadioOpts[T]) *Radio[T] {
	return NewRadio(value, opts...)
}

// Value is the value that this radio stands for.
//
// Flutter: material/radio.dart Radio.value
func (w *Radio[T]) Value() T { return w.value }

// CreateState creates the state.
//
// Flutter: material/radio.dart Radio.createState
func (w *Radio[T]) CreateState() widgets.State { return &radioState[T]{} }

type radioState[T comparable] struct {
	widgets.StateOf[*Radio[T]]
	internalFocusNode *widgets.FocusNode
	internalRegistry  *radioRegistry[T]
}

func (s *radioState[T]) focusNode() *widgets.FocusNode {
	if n := s.Widget().o.FocusNode; n != nil {
		return n
	}
	if s.internalFocusNode == nil {
		s.internalFocusNode = widgets.NewFocusNode()
	}
	return s.internalFocusNode
}

func (s *radioState[T]) enabled() bool {
	o := s.Widget().o
	if e, ok := o.Enabled.Get(); ok {
		return e
	}
	if o.OnChanged != nil || o.GroupRegistry != nil {
		return true
	}
	_, ok := widgets.RadioGroupMaybeOf[T](s.Context())
	return ok
}

func (s *radioState[T]) effectiveRegistry() widgets.RadioGroupRegistry[T] {
	if r := s.Widget().o.GroupRegistry; r != nil {
		return r
	}
	if r, ok := widgets.RadioGroupMaybeOf[T](s.Context()); ok {
		return r
	}
	if s.internalRegistry == nil {
		s.internalRegistry = &radioRegistry[T]{state: s}
	}
	return s.internalRegistry
}

func (s *radioState[T]) Dispose() {
	if s.internalFocusNode != nil {
		s.internalFocusNode.Dispose()
	}
}

func (s *radioState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	if foundation.DebugMode && w.o.Enabled == foundation.Some(true) && w.o.OnChanged == nil && w.o.GroupRegistry == nil {
		if _, ok := widgets.RadioGroupMaybeOf[T](bc); !ok {
			panic(errors.New("RadioNoRegistry", "Radio is enabled but has no Radio.onChange or registry above"))
		}
	}
	DebugCheckHasMaterial(bc)
	radioTheme := RadioThemeOf(bc)
	cursor := toggleableMouseCursor(w.o.MouseCursor, radioTheme.MouseCursor)
	return widgets.NewRawRadio(w.value, cursor, w.o.Toggleable, s.focusNode(), w.o.Autofocus,
		s.effectiveRegistry(), s.enabled(),
		func(_ widgets.BuildContext, state *widgets.ToggleableStateMixin) widgets.Widget {
			return &radioPaint[T]{StatefulWidget: widgets.NewStatefulWidgetBase(), toggleableState: state, o: w.o}
		})
}

// radioRegistry is the registry of a radio that has no RadioGroup above: it is
// made of the GroupValue and the OnChanged of the radio.
//
// Flutter: material/radio.dart _RadioRegistry
type radioRegistry[T comparable] struct{ state *radioState[T] }

func (r *radioRegistry[T]) GroupValue() foundation.Opt[T]           { return r.state.Widget().o.GroupValue }
func (r *radioRegistry[T]) RegisterClient(widgets.RadioClient[T])   {}
func (r *radioRegistry[T]) UnregisterClient(widgets.RadioClient[T]) {}
func (r *radioRegistry[T]) OnChanged(value foundation.Opt[T]) {
	if f := r.state.Widget().o.OnChanged; f != nil {
		f(value)
	}
}

// radioPaint paints a radio from the state of its toggle.
//
// Flutter: material/radio.dart _RadioPaint
type radioPaint[T comparable] struct {
	widgets.StatefulWidget
	toggleableState *widgets.ToggleableStateMixin
	o               RadioOpts[T]
}

func (w *radioPaint[T]) CreateState() widgets.State { return &radioPaintState[T]{} }

type radioPaintState[T comparable] struct {
	widgets.StateOf[*radioPaint[T]]
	painter *radioPainter
}

func (s *radioPaintState[T]) InitState() { s.painter = &radioPainter{} }
func (s *radioPaintState[T]) Dispose()   { s.painter.Dispose() }

func (s *radioPaintState[T]) widgetFillColor() widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		if states.Has(widgets.WidgetStateDisabled) {
			return foundation.None[ui.Color]()
		}
		if states.Has(widgets.WidgetStateSelected) {
			return s.Widget().o.ActiveColor
		}
		return foundation.None[ui.Color]()
	})
}

func (s *radioPaintState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	ts := w.toggleableState
	radioTheme := RadioThemeOf(bc)
	defaults := newRadioDefaults(bc)

	states := ts.States()
	activeStates := states.With(widgets.WidgetStateSelected)
	inactiveStates := states.Without(widgets.WidgetStateSelected)
	fill := func(st widgets.WidgetStates) foundation.Opt[ui.Color] {
		return firstOptColor(optColorOf(o.FillColor, st), s.widgetFillColor().Resolve(st), optColorOf(radioTheme.FillColor, st))
	}
	activeColor := fill(activeStates)
	effectiveActiveColor := activeColor.OrElse(defaults.fillColor(activeStates))
	inactiveColor := fill(inactiveStates)
	effectiveInactiveColor := inactiveColor.OrElse(defaults.fillColor(inactiveStates))
	background := func(st widgets.WidgetStates) ui.Color {
		return firstOptColor(optColorOf(o.BackgroundColor, st), optColorOf(radioTheme.BackgroundColor, st)).OrElse(defaults.backgroundColor)
	}
	activeBackgroundColor := background(activeStates)
	inactiveBackgroundColor := background(inactiveStates)

	focusedStates := states.With(widgets.WidgetStateFocused)
	effectiveFocusOverlayColor := firstOptColor(optColorOf(o.OverlayColor, focusedStates), o.FocusColor,
		optColorOf(radioTheme.OverlayColor, focusedStates)).OrElse(defaults.overlayColor(focusedStates))
	hoveredStates := states.With(widgets.WidgetStateHovered)
	effectiveHoverOverlayColor := firstOptColor(optColorOf(o.OverlayColor, hoveredStates), o.HoverColor,
		optColorOf(radioTheme.OverlayColor, hoveredStates)).OrElse(defaults.overlayColor(hoveredStates))

	withAlpha := func(c foundation.Opt[ui.Color]) foundation.Opt[ui.Color] {
		if v, ok := c.Get(); ok {
			return foundation.Some(v.WithAlpha(RadialReactionAlpha))
		}
		return c
	}
	activePressedStates := activeStates.With(widgets.WidgetStatePressed)
	effectiveActivePressedOverlayColor := firstOptColor(optColorOf(o.OverlayColor, activePressedStates),
		optColorOf(radioTheme.OverlayColor, activePressedStates), withAlpha(activeColor)).OrElse(defaults.overlayColor(activePressedStates))
	inactivePressedStates := inactiveStates.With(widgets.WidgetStatePressed)
	effectiveInactivePressedOverlayColor := firstOptColor(optColorOf(o.OverlayColor, inactivePressedStates),
		optColorOf(radioTheme.OverlayColor, inactivePressedStates), withAlpha(inactiveColor)).OrElse(defaults.overlayColor(inactivePressedStates))

	if ts.DownPosition().IsSome() {
		if states.Has(widgets.WidgetStateSelected) {
			effectiveHoverOverlayColor = effectiveActivePressedOverlayColor
			effectiveFocusOverlayColor = effectiveActivePressedOverlayColor
		} else {
			effectiveHoverOverlayColor = effectiveInactivePressedOverlayColor
			effectiveFocusOverlayColor = effectiveInactivePressedOverlayColor
		}
	}

	tapTarget := MaterialTapTargetSizeUnspecified
	for _, t := range []MaterialTapTargetSize{o.MaterialTapTargetSize, radioTheme.MaterialTapTargetSize, defaults.materialTapTargetSize} {
		if t != MaterialTapTargetSizeUnspecified {
			tapTarget = t
			break
		}
	}
	density := o.VisualDensity.OrElse(radioTheme.VisualDensity.OrElse(defaults.visualDensity))
	size := toggleableSize(tapTarget, density)
	side := func(st widgets.WidgetStates, c ui.Color) painting.BorderSide {
		if v, ok := firstSide(resolveToggleSide(o.Side, o.StateSide, st), resolveToggleSide(radioTheme.Side, radioTheme.StateSide, st)).Get(); ok {
			return v
		}
		return painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(c), Width: foundation.Some(2.0), StrokeAlign: foundation.Some(painting.StrokeAlignCenter)})
	}
	activeSide := side(activeStates, effectiveActiveColor)
	inactiveSide := side(inactiveStates, effectiveInactiveColor)
	innerRadius := firstOptDouble(optDoubleOf(o.InnerRadius, activeStates), optDoubleOf(radioTheme.InnerRadius, activeStates)).OrElse(radioInnerRadius)

	p := s.painter
	p.SetPosition(ts.Position())
	p.SetReaction(ts.Reaction())
	p.SetReactionFocusFade(ts.ReactionFocusFade())
	p.SetReactionHoverFade(ts.ReactionHoverFade())
	p.SetInactiveReactionColor(effectiveInactivePressedOverlayColor)
	p.SetReactionColor(effectiveActivePressedOverlayColor)
	p.SetHoverColor(effectiveHoverOverlayColor)
	p.SetFocusColor(effectiveFocusOverlayColor)
	p.SetSplashRadius(o.SplashRadius.OrElse(radioTheme.SplashRadius.OrElse(RadialReactionRadius)))
	p.SetDownPosition(ts.DownPosition())
	p.SetIsFocused(states.Has(widgets.WidgetStateFocused))
	p.SetIsHovered(states.Has(widgets.WidgetStateHovered))
	p.SetActiveColor(effectiveActiveColor)
	p.SetInactiveColor(effectiveInactiveColor)
	p.setBackgroundColors(inactiveBackgroundColor, activeBackgroundColor)
	p.setSides(inactiveSide, activeSide)
	p.setInnerRadius(innerRadius)
	return widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Size: size, Painter: p})
}

func firstOptDouble(values ...foundation.Opt[float64]) foundation.Opt[float64] {
	for _, v := range values {
		if v.IsSome() {
			return v
		}
	}
	return foundation.None[float64]()
}

// radioPainter paints the circle and the dot of a Radio.
//
// Flutter: material/radio.dart _RadioPainter
type radioPainter struct {
	widgets.ToggleablePainter
	inactiveBackgroundColor ui.Color
	activeBackgroundColor   ui.Color
	inactiveSide            painting.BorderSide
	activeSide              painting.BorderSide
	innerRadius             float64
}

func (p *radioPainter) setBackgroundColors(inactive, active ui.Color) {
	if p.inactiveBackgroundColor != inactive || p.activeBackgroundColor != active {
		p.inactiveBackgroundColor, p.activeBackgroundColor = inactive, active
		p.NotifyListeners()
	}
}

func (p *radioPainter) setSides(inactive, active painting.BorderSide) {
	if p.inactiveSide != inactive || p.activeSide != active {
		p.inactiveSide, p.activeSide = inactive, active
		p.NotifyListeners()
	}
}

func (p *radioPainter) setInnerRadius(v float64) {
	if p.innerRadius != v {
		p.innerRadius = v
		p.NotifyListeners()
	}
}

// Paint paints the radial reaction, the circle and the dot.
//
// Flutter: material/radio.dart _RadioPainter.paint
func (p *radioPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	p.PaintRadialReaction(canvas, ui.Offset{}, size.Center(ui.Offset{}))

	rect := ui.NewRectFromLTWH(0, 0, size.Width, size.Height)
	center := rect.Center()
	effectiveRect := ui.NewRectFromLTWH(center.Dx-radioOuterRadius, center.Dy-radioOuterRadius, radioOuterRadius*2, radioOuterRadius*2)

	position := p.Position()
	backgroundPaint := ui.NewPaint()
	backgroundPaint.SetColor(lerpColor(p.inactiveBackgroundColor, p.activeBackgroundColor, position.Value()))
	backgroundPaint.SetStyle(ui.PaintingStyleFill)
	canvas.DrawCircle(center, radioOuterRadius, backgroundPaint)

	side := painting.BorderSideLerp(p.inactiveSide, p.activeSide, position.Value())
	painting.NewCircleBorder(painting.CircleBorderOpts{Side: side}).Paint(canvas, effectiveRect, ui.TextDirectionLTR)

	// Inner circle.
	if position.Status() != animation.AnimationStatusDismissed {
		innerPaint := ui.NewPaint()
		innerPaint.SetStyle(ui.PaintingStyleFill)
		innerPaint.SetColor(lerpColor(p.InactiveColor(), p.ActiveColor(), position.Value()))
		canvas.DrawCircle(center, p.innerRadius*position.Value(), innerPaint)
	}
}

// radioDefaults are the default values of a Radio.
//
// Flutter: material/radio.dart _RadioDefaultsM2, _RadioDefaultsM3
type radioDefaults struct {
	fillColor             func(states widgets.WidgetStates) ui.Color
	overlayColor          func(states widgets.WidgetStates) ui.Color
	materialTapTargetSize MaterialTapTargetSize
	visualDensity         VisualDensity
	backgroundColor       ui.Color
}

func newRadioDefaults(bc widgets.BuildContext) radioDefaults {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	transparent := ui.NewColor(0x00000000)
	d := radioDefaults{materialTapTargetSize: theme.MaterialTapTargetSize, visualDensity: theme.VisualDensity, backgroundColor: transparent}
	if theme.UseMaterial3 {
		d.fillColor = func(s widgets.WidgetStates) ui.Color {
			if s.Has(widgets.WidgetStateSelected) {
				if s.Has(widgets.WidgetStateDisabled) {
					return colors.OnSurface().WithOpacity(0.38)
				}
				return colors.Primary()
			}
			switch {
			case s.Has(widgets.WidgetStateDisabled):
				return colors.OnSurface().WithOpacity(0.38)
			case s.Has(widgets.WidgetStatePressed), s.Has(widgets.WidgetStateHovered), s.Has(widgets.WidgetStateFocused):
				return colors.OnSurface()
			}
			return colors.OnSurfaceVariant()
		}
		d.overlayColor = func(s widgets.WidgetStates) ui.Color {
			if s.Has(widgets.WidgetStateSelected) {
				switch {
				case s.Has(widgets.WidgetStatePressed):
					return colors.OnSurface().WithOpacity(0.1)
				case s.Has(widgets.WidgetStateHovered):
					return colors.Primary().WithOpacity(0.08)
				case s.Has(widgets.WidgetStateFocused):
					return colors.Primary().WithOpacity(0.1)
				}
				return transparent
			}
			switch {
			case s.Has(widgets.WidgetStatePressed):
				return colors.Primary().WithOpacity(0.1)
			case s.Has(widgets.WidgetStateHovered):
				return colors.OnSurface().WithOpacity(0.08)
			case s.Has(widgets.WidgetStateFocused):
				return colors.OnSurface().WithOpacity(0.1)
			}
			return transparent
		}
		return d
	}
	d.fillColor = func(s widgets.WidgetStates) ui.Color {
		switch {
		case s.Has(widgets.WidgetStateDisabled):
			return theme.DisabledColor
		case s.Has(widgets.WidgetStateSelected):
			return colors.Secondary()
		}
		return theme.UnselectedWidgetColor
	}
	d.overlayColor = func(s widgets.WidgetStates) ui.Color {
		switch {
		case s.Has(widgets.WidgetStatePressed):
			return d.fillColor(s).WithAlpha(RadialReactionAlpha)
		case s.Has(widgets.WidgetStateHovered):
			return theme.HoverColor
		case s.Has(widgets.WidgetStateFocused):
			return theme.FocusColor
		}
		return transparent
	}
	return d
}
