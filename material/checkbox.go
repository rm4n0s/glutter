// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The helpers that Checkbox, Radio and Switch share.

// optColorOf resolves a property of nullable colors; a nil property resolves to
// none, as Dart's null property does.
func optColorOf(p widgets.WidgetStateProperty[foundation.Opt[ui.Color]], states widgets.WidgetStates) foundation.Opt[ui.Color] {
	if p == nil {
		return foundation.None[ui.Color]()
	}
	return p.Resolve(states)
}

// optDoubleOf resolves a property of nullable numbers.
func optDoubleOf(p widgets.WidgetStateProperty[foundation.Opt[float64]], states widgets.WidgetStates) foundation.Opt[float64] {
	if p == nil {
		return foundation.None[float64]()
	}
	return p.Resolve(states)
}

// firstOptColor is the first of the colors that is set (Dart's ??).
func firstOptColor(colors ...foundation.Opt[ui.Color]) foundation.Opt[ui.Color] {
	for _, c := range colors {
		if c.IsSome() {
			return c
		}
	}
	return foundation.None[ui.Color]()
}

// toggleableMouseCursor is the cursor of a Checkbox, Radio or Switch: the cursor of
// the widget, then of the theme, then the adaptive clickable one.
func toggleableMouseCursor(widgetCursor services.MouseCursor, themeCursor widgets.WidgetStateProperty[services.MouseCursor]) widgets.WidgetStateProperty[services.MouseCursor] {
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) services.MouseCursor {
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
	})
}

// toggleableSize is the size of the box of a Checkbox or Radio for the tap target
// size and the density.
func toggleableSize(tapTarget MaterialTapTargetSize, density VisualDensity) ui.Size {
	side := MinInteractiveDimension
	if tapTarget == MaterialTapTargetSizeShrinkWrap {
		side -= 8.0
	}
	adj := density.BaseSizeAdjustment()
	return ui.Size{Width: side + adj.Dx, Height: side + adj.Dy}
}

// resolveToggleSide is _resolveSide of Checkbox and Radio: a side that depends on
// the states resolves with them; a plain side only applies when the widget is not
// selected.
//
// Flutter: material/checkbox.dart _CheckboxState._resolveSide
func resolveToggleSide(side foundation.Opt[painting.BorderSide], stateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]], states widgets.WidgetStates) foundation.Opt[painting.BorderSide] {
	if stateSide != nil {
		return stateSide.Resolve(states)
	}
	if !states.Has(widgets.WidgetStateSelected) {
		return side
	}
	return foundation.None[painting.BorderSide]()
}

// firstSide is the first of the sides that is set.
func firstSide(sides ...foundation.Opt[painting.BorderSide]) foundation.Opt[painting.BorderSide] {
	for _, s := range sides {
		if s.IsSome() {
			return s
		}
	}
	return foundation.None[painting.BorderSide]()
}

// CheckboxWidth is the width of the Checkbox; its height is the same.
//
// Flutter: material/checkbox.dart Checkbox.width
const CheckboxWidth = 18.0

// CheckboxOpts are the optional parameters of NewCheckbox.
//
// Flutter: material/checkbox.dart Checkbox.new
type CheckboxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Tristate makes the checkbox cycle through true, false and none; with it
	// the value may be none.
	Tristate bool
	// MouseCursor is the cursor over the checkbox; it may resolve with the
	// states of the checkbox. The default is the cursor of the theme, then the
	// adaptive clickable one.
	MouseCursor services.MouseCursor
	// ActiveColor is the color of the checkbox when it is selected.
	ActiveColor foundation.Opt[ui.Color]
	// FillColor is the color that fills the checkbox, in each state.
	FillColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// CheckColor is the color of the check mark.
	CheckColor foundation.Opt[ui.Color]
	// FocusColor is the color of the overlay while the checkbox has the focus.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the color of the overlay while the pointer is over the
	// checkbox.
	HoverColor foundation.Opt[ui.Color]
	// OverlayColor is the color of the overlay, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// SplashRadius is the radius of the splash.
	SplashRadius foundation.Opt[float64]
	// MaterialTapTargetSize is the size of the tap target.
	MaterialTapTargetSize MaterialTapTargetSize
	// VisualDensity is how compact the layout is.
	VisualDensity foundation.Opt[VisualDensity]
	// FocusNode is the focus node of the checkbox.
	FocusNode *widgets.FocusNode
	// Autofocus asks for the focus when the checkbox is built.
	Autofocus bool
	// Shape is the shape of the checkbox; the default is a rounded rectangle.
	Shape painting.OutlinedBorder
	// Side is the border of the checkbox when it is not selected.
	Side foundation.Opt[painting.BorderSide]
	// StateSide is the border of the checkbox in each state (Flutter's
	// WidgetStateBorderSide); when it is set, Side is ignored.
	StateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// IsError draws the checkbox in the colors of an error.
	IsError bool
	// SemanticLabel is the semantic label of the checkbox.
	SemanticLabel string
}

// Checkbox is a Material Design checkbox: it lets the user select more than one
// option of a set, and with Tristate it has a third, mixed value. It does not
// keep its own value: the widget calls OnChanged and shows the Value that it
// gets.
//
// Deviation: the adaptive constructor builds the Material checkbox on iOS and
// macOS too, as there is no Cupertino library (SPEC §12.10).
//
// Flutter: material/checkbox.dart Checkbox
type Checkbox struct {
	widgets.StatefulWidget
	value     foundation.Opt[bool]
	onChanged func(value foundation.Opt[bool])
	o         CheckboxOpts
}

// NewCheckbox creates the widget. A nil onChanged disables the checkbox. In
// debug builds it panics with the tag CheckboxValueNull when the value is none
// and the checkbox is not tristate.
//
// Flutter: material/checkbox.dart Checkbox.new
func NewCheckbox(value foundation.Opt[bool], onChanged func(value foundation.Opt[bool]), opts ...CheckboxOpts) *Checkbox {
	o := firstOpts(opts)
	if foundation.DebugMode && !o.Tristate && value.IsNone() {
		panic(errors.New("CheckboxValueNull", "the value of a Checkbox that is not tristate can't be none"))
	}
	return &Checkbox{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: value, onChanged: onChanged, o: o}
}

// NewCheckboxAdaptive creates a checkbox that adapts to the platform.
//
// Deviation: it is the same as NewCheckbox (SPEC §12.10).
//
// Flutter: material/checkbox.dart Checkbox.adaptive
func NewCheckboxAdaptive(value foundation.Opt[bool], onChanged func(value foundation.Opt[bool]), opts ...CheckboxOpts) *Checkbox {
	return NewCheckbox(value, onChanged, opts...)
}

// Value is whether the checkbox is checked: none is the mixed value.
//
// Flutter: material/checkbox.dart Checkbox.value
func (w *Checkbox) Value() foundation.Opt[bool] { return w.value }

// OnChanged is called when the user toggles the checkbox; nil disables it.
//
// Flutter: material/checkbox.dart Checkbox.onChanged
func (w *Checkbox) OnChanged() func(value foundation.Opt[bool]) { return w.onChanged }

// Tristate is whether the checkbox cycles through none too.
//
// Flutter: material/checkbox.dart Checkbox.tristate
func (w *Checkbox) Tristate() bool { return w.o.Tristate }

// CreateState creates the state.
//
// Flutter: material/checkbox.dart Checkbox.createState
func (w *Checkbox) CreateState() widgets.State { return &checkboxState{} }

type checkboxState struct {
	widgets.StateOf[*Checkbox]
	widgets.TickerProvider
	widgets.ToggleableStateMixin
	painter       *checkboxPainter
	previousValue foundation.Opt[bool]
}

func (s *checkboxState) InitState() {
	s.painter = &checkboxPainter{}
	s.Init(s, s, widgets.ToggleableStateMixinOpts{ReactionAnimationDuration: RadialReactionDuration})
	s.previousValue = s.Widget().value
}

func (s *checkboxState) DidUpdateWidget(old *Checkbox) {
	if old.value != s.Widget().value {
		s.previousValue = old.value
		s.AnimateToValue()
	}
}

func (s *checkboxState) Dispose() {
	s.painter.Dispose()
	s.DisposeToggleable()
}

func (s *checkboxState) ToggleableOnChanged() func(foundation.Opt[bool]) { return s.Widget().onChanged }
func (s *checkboxState) ToggleableValue() foundation.Opt[bool]           { return s.Widget().value }
func (s *checkboxState) ToggleableTristate() bool                        { return s.Widget().o.Tristate }

func (s *checkboxState) widgetFillColor() widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
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

func (s *checkboxState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	DebugCheckHasMaterial(bc)
	checkboxTheme := CheckboxThemeOf(bc)
	defaults := newCheckboxDefaults(bc)
	tapTarget := MaterialTapTargetSizeUnspecified
	for _, t := range []MaterialTapTargetSize{w.o.MaterialTapTargetSize, checkboxTheme.MaterialTapTargetSize, defaults.materialTapTargetSize} {
		if t != MaterialTapTargetSizeUnspecified {
			tapTarget = t
			break
		}
	}
	density := w.o.VisualDensity.OrElse(checkboxTheme.VisualDensity.OrElse(defaults.visualDensity))
	size := toggleableSize(tapTarget, density)
	mouseCursor := toggleableMouseCursor(w.o.MouseCursor, checkboxTheme.MouseCursor)

	states := s.States()
	isError := w.o.IsError
	with := func(st widgets.WidgetStates, add ...widgets.WidgetState) widgets.WidgetStates {
		for _, a := range add {
			st = st.With(a)
		}
		return st
	}
	activeStates := states.With(widgets.WidgetStateSelected)
	inactiveStates := states.Without(widgets.WidgetStateSelected)
	if isError {
		activeStates = activeStates.With(widgets.WidgetStateError)
		inactiveStates = inactiveStates.With(widgets.WidgetStateError)
	}
	fill := func(st widgets.WidgetStates) foundation.Opt[ui.Color] {
		return firstOptColor(optColorOf(w.o.FillColor, st), s.widgetFillColor().Resolve(st), optColorOf(checkboxTheme.FillColor, st))
	}
	activeColor := fill(activeStates)
	effectiveActiveColor := activeColor.OrElse(defaults.fillColor(activeStates))
	inactiveColor := fill(inactiveStates)
	effectiveInactiveColor := inactiveColor.OrElse(defaults.fillColor(inactiveStates))

	side := func(st widgets.WidgetStates) painting.BorderSide {
		if v, ok := firstSide(resolveToggleSide(w.o.Side, w.o.StateSide, st), resolveToggleSide(checkboxTheme.Side, checkboxTheme.StateSide, st)).Get(); ok {
			return v
		}
		return defaults.side(st)
	}
	activeSide, inactiveSide := side(activeStates), side(inactiveStates)

	focusedStates := with(states, widgets.WidgetStateFocused)
	if isError {
		focusedStates = focusedStates.With(widgets.WidgetStateError)
	}
	effectiveFocusOverlayColor := firstOptColor(optColorOf(w.o.OverlayColor, focusedStates), w.o.FocusColor,
		optColorOf(checkboxTheme.OverlayColor, focusedStates)).OrElse(defaults.overlayColor(focusedStates))
	hoveredStates := with(states, widgets.WidgetStateHovered)
	if isError {
		hoveredStates = hoveredStates.With(widgets.WidgetStateError)
	}
	effectiveHoverOverlayColor := firstOptColor(optColorOf(w.o.OverlayColor, hoveredStates), w.o.HoverColor,
		optColorOf(checkboxTheme.OverlayColor, hoveredStates)).OrElse(defaults.overlayColor(hoveredStates))

	withAlpha := func(c foundation.Opt[ui.Color]) foundation.Opt[ui.Color] {
		if v, ok := c.Get(); ok {
			return foundation.Some(v.WithAlpha(RadialReactionAlpha))
		}
		return c
	}
	activePressedStates := activeStates.With(widgets.WidgetStatePressed)
	effectiveActivePressedOverlayColor := firstOptColor(optColorOf(w.o.OverlayColor, activePressedStates),
		optColorOf(checkboxTheme.OverlayColor, activePressedStates), withAlpha(activeColor)).OrElse(defaults.overlayColor(activePressedStates))
	inactivePressedStates := inactiveStates.With(widgets.WidgetStatePressed)
	effectiveInactivePressedOverlayColor := firstOptColor(optColorOf(w.o.OverlayColor, inactivePressedStates),
		optColorOf(checkboxTheme.OverlayColor, inactivePressedStates), withAlpha(inactiveColor)).OrElse(defaults.overlayColor(inactivePressedStates))

	if s.DownPosition().IsSome() {
		if states.Has(widgets.WidgetStateSelected) {
			effectiveHoverOverlayColor = effectiveActivePressedOverlayColor
			effectiveFocusOverlayColor = effectiveActivePressedOverlayColor
		} else {
			effectiveHoverOverlayColor = effectiveInactivePressedOverlayColor
			effectiveFocusOverlayColor = effectiveInactivePressedOverlayColor
		}
	}

	checkStates := states
	if isError {
		checkStates = checkStates.With(widgets.WidgetStateError)
	}
	effectiveCheckColor := firstOptColor(w.o.CheckColor, optColorOf(checkboxTheme.CheckColor, checkStates)).OrElse(defaults.checkColor(checkStates))
	effectiveSplashRadius := w.o.SplashRadius.OrElse(checkboxTheme.SplashRadius.OrElse(defaults.splashRadius))

	shape := w.o.Shape
	if shape == nil {
		shape = checkboxTheme.Shape
	}
	if shape == nil {
		shape = defaults.shape
	}

	p := s.painter
	p.SetPosition(s.Position())
	p.SetReaction(s.Reaction())
	p.SetReactionFocusFade(s.ReactionFocusFade())
	p.SetReactionHoverFade(s.ReactionHoverFade())
	p.SetInactiveReactionColor(effectiveInactivePressedOverlayColor)
	p.SetReactionColor(effectiveActivePressedOverlayColor)
	p.SetHoverColor(effectiveHoverOverlayColor)
	p.SetFocusColor(effectiveFocusOverlayColor)
	p.SetSplashRadius(effectiveSplashRadius)
	p.SetDownPosition(s.DownPosition())
	p.SetIsFocused(states.Has(widgets.WidgetStateFocused))
	p.SetIsHovered(states.Has(widgets.WidgetStateHovered))
	p.SetActiveColor(effectiveActiveColor)
	p.SetInactiveColor(effectiveInactiveColor)
	p.setCheckColor(effectiveCheckColor)
	p.setValue(w.value)
	p.setPreviousValue(s.previousValue)
	p.setShape(shape)
	p.setActiveSide(activeSide)
	p.setInactiveSide(inactiveSide)

	label := foundation.None[string]()
	if w.o.SemanticLabel != "" {
		label = foundation.Some(w.o.SemanticLabel)
	}
	mixed := foundation.None[bool]()
	if w.o.Tristate {
		mixed = foundation.Some(w.value.IsNone())
	}
	return widgets.NewSemantics(
		s.BuildToggleable(size, p, widgets.ToggleableOpts{MouseCursor: mouseCursor, FocusNode: w.o.FocusNode, Autofocus: w.o.Autofocus}),
		widgets.SemanticsOpts{Label: label, Checked: foundation.Some(w.value.OrElse(false)), Mixed: mixed})
}

const (
	checkboxEdgeSize    = CheckboxWidth
	checkboxStrokeWidth = 2.0
)

// checkboxPainter paints the box and the check mark of a Checkbox.
//
// Flutter: material/checkbox.dart _CheckboxPainter
type checkboxPainter struct {
	widgets.ToggleablePainter
	checkColor    ui.Color
	value         foundation.Opt[bool]
	previousValue foundation.Opt[bool]
	shape         painting.OutlinedBorder
	activeSide    painting.BorderSide
	inactiveSide  painting.BorderSide
}

func (p *checkboxPainter) setCheckColor(v ui.Color) {
	if p.checkColor != v {
		p.checkColor = v
		p.NotifyListeners()
	}
}

func (p *checkboxPainter) setValue(v foundation.Opt[bool]) {
	if p.value != v {
		p.value = v
		p.NotifyListeners()
	}
}

func (p *checkboxPainter) setPreviousValue(v foundation.Opt[bool]) {
	if p.previousValue != v {
		p.previousValue = v
		p.NotifyListeners()
	}
}

func (p *checkboxPainter) setShape(v painting.OutlinedBorder) {
	if !foundation.ValuesEqual(p.shape, v) {
		p.shape = v
		p.NotifyListeners()
	}
}

func (p *checkboxPainter) setActiveSide(v painting.BorderSide) {
	if p.activeSide != v {
		p.activeSide = v
		p.NotifyListeners()
	}
}

func (p *checkboxPainter) setInactiveSide(v painting.BorderSide) {
	if p.inactiveSide != v {
		p.inactiveSide = v
		p.NotifyListeners()
	}
}

func (p *checkboxPainter) outerRectAt(origin ui.Offset, t float64) ui.Rect {
	// First 25% of the animation, only the outer rect shrinks.
	inset := 1.0 - math.Abs(t-0.5)*2.0
	size := checkboxEdgeSize - inset*checkboxStrokeWidth
	return ui.NewRectFromLTWH(origin.Dx+inset, origin.Dy+inset, size, size)
}

func (p *checkboxPainter) colorAt(t float64) ui.Color {
	// As t goes from 0.0 to 0.25, animate from the inactiveColor to activeColor.
	if t >= 0.25 {
		return p.ActiveColor()
	}
	return lerpColor(p.InactiveColor(), p.ActiveColor(), t*4.0)
}

func (p *checkboxPainter) createStrokePaint() *ui.Paint {
	paint := ui.NewPaint()
	paint.SetColor(p.checkColor)
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(checkboxStrokeWidth)
	return paint
}

func (p *checkboxPainter) drawBox(canvas *ui.Canvas, outer ui.Rect, paint *ui.Paint, side *painting.BorderSide) {
	if p.shape.PreferPaintInterior() {
		p.shape.PaintInterior(canvas, outer, paint, ui.TextDirectionLTR)
	} else {
		canvas.DrawPath(p.shape.GetOuterPath(outer, ui.TextDirectionLTR), paint)
	}
	if side != nil {
		p.shape.CopyWithSide(*side).Paint(canvas, outer, ui.TextDirectionLTR)
	}
}

func (p *checkboxPainter) drawCheck(canvas *ui.Canvas, origin ui.Offset, t float64, paint *ui.Paint) {
	path := ui.NewPath()
	start := ui.Offset{Dx: checkboxEdgeSize * 0.15, Dy: checkboxEdgeSize * 0.45}
	mid := ui.Offset{Dx: checkboxEdgeSize * 0.4, Dy: checkboxEdgeSize * 0.7}
	end := ui.Offset{Dx: checkboxEdgeSize * 0.85, Dy: checkboxEdgeSize * 0.25}
	if t < 0.5 {
		strokeT := t * 2.0
		drawMid := start.Lerp(mid, strokeT)
		path.MoveTo(origin.Dx+start.Dx, origin.Dy+start.Dy)
		path.LineTo(origin.Dx+drawMid.Dx, origin.Dy+drawMid.Dy)
	} else {
		strokeT := (t - 0.5) * 2.0
		drawEnd := mid.Lerp(end, strokeT)
		path.MoveTo(origin.Dx+start.Dx, origin.Dy+start.Dy)
		path.LineTo(origin.Dx+mid.Dx, origin.Dy+mid.Dy)
		path.LineTo(origin.Dx+drawEnd.Dx, origin.Dy+drawEnd.Dy)
	}
	canvas.DrawPath(path, paint)
}

func (p *checkboxPainter) drawDash(canvas *ui.Canvas, origin ui.Offset, t float64, paint *ui.Paint) {
	start := ui.Offset{Dx: checkboxEdgeSize * 0.2, Dy: checkboxEdgeSize * 0.5}
	mid := ui.Offset{Dx: checkboxEdgeSize * 0.5, Dy: checkboxEdgeSize * 0.5}
	end := ui.Offset{Dx: checkboxEdgeSize * 0.8, Dy: checkboxEdgeSize * 0.5}
	drawStart := start.Lerp(mid, 1.0-t)
	drawEnd := mid.Lerp(end, t)
	canvas.DrawLine(origin.Add(drawStart), origin.Add(drawEnd), paint)
}

// Paint paints the radial reaction, the box and the mark.
//
// Flutter: material/checkbox.dart _CheckboxPainter.paint
func (p *checkboxPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	p.PaintRadialReaction(canvas, ui.Offset{}, size.Center(ui.Offset{}))

	strokePaint := p.createStrokePaint()
	origin := ui.Offset{Dx: size.Width/2.0 - checkboxEdgeSize/2.0, Dy: size.Height/2.0 - checkboxEdgeSize/2.0}
	position := p.Position()
	var tNormalized float64
	switch position.Status() {
	case animation.AnimationStatusForward, animation.AnimationStatusCompleted:
		tNormalized = position.Value()
	default:
		tNormalized = 1.0 - position.Value()
	}

	// Four cases: false to null, false to true, null to false, true to false.
	if p.previousValue == foundation.Some(false) || p.value == foundation.Some(false) {
		t := tNormalized
		if p.value == foundation.Some(false) {
			t = 1.0 - tNormalized
		}
		outer := p.outerRectAt(origin, t)
		paint := ui.NewPaint()
		paint.SetColor(p.colorAt(t))
		if t <= 0.5 {
			border := painting.BorderSideLerp(p.inactiveSide, p.activeSide, t)
			p.drawBox(canvas, outer, paint, &border)
		} else {
			p.drawBox(canvas, outer, paint, &p.activeSide)
			tShrink := (t - 0.5) * 2.0
			if p.previousValue.IsNone() || p.value.IsNone() {
				p.drawDash(canvas, origin, tShrink, strokePaint)
			} else {
				p.drawCheck(canvas, origin, tShrink, strokePaint)
			}
		}
		return
	}
	// Two cases: null to true, true to null.
	outer := p.outerRectAt(origin, 1.0)
	paint := ui.NewPaint()
	paint.SetColor(p.colorAt(1.0))
	p.drawBox(canvas, outer, paint, &p.activeSide)
	if tNormalized <= 0.5 {
		tShrink := 1.0 - tNormalized*2.0
		if p.previousValue.OrElse(false) {
			p.drawCheck(canvas, origin, tShrink, strokePaint)
		} else {
			p.drawDash(canvas, origin, tShrink, strokePaint)
		}
	} else {
		tExpand := (tNormalized - 0.5) * 2.0
		if p.value.OrElse(false) {
			p.drawCheck(canvas, origin, tExpand, strokePaint)
		} else {
			p.drawDash(canvas, origin, tExpand, strokePaint)
		}
	}
}

// checkboxDefaults are the default values of a Checkbox.
//
// Flutter: material/checkbox.dart _CheckboxDefaultsM2, _CheckboxDefaultsM3
type checkboxDefaults struct {
	side                  func(states widgets.WidgetStates) painting.BorderSide
	fillColor             func(states widgets.WidgetStates) ui.Color
	checkColor            func(states widgets.WidgetStates) ui.Color
	overlayColor          func(states widgets.WidgetStates) ui.Color
	splashRadius          float64
	materialTapTargetSize MaterialTapTargetSize
	visualDensity         VisualDensity
	shape                 painting.OutlinedBorder
}

func newCheckboxDefaults(bc widgets.BuildContext) checkboxDefaults {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	transparent := ui.NewColor(0x00000000)
	border := func(width float64, c ui.Color) painting.BorderSide {
		return painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(width), Color: foundation.Some(c)})
	}
	if theme.UseMaterial3 {
		return checkboxDefaults{
			side: func(s widgets.WidgetStates) painting.BorderSide {
				switch {
				case s.Has(widgets.WidgetStateDisabled):
					if s.Has(widgets.WidgetStateSelected) {
						return border(2.0, transparent)
					}
					return border(2.0, colors.OnSurface().WithOpacity(0.38))
				case s.Has(widgets.WidgetStateSelected):
					return border(0.0, transparent)
				case s.Has(widgets.WidgetStateError):
					return border(2.0, colors.Error())
				case s.Has(widgets.WidgetStatePressed), s.Has(widgets.WidgetStateHovered), s.Has(widgets.WidgetStateFocused):
					return border(2.0, colors.OnSurface())
				}
				return border(2.0, colors.OnSurfaceVariant())
			},
			fillColor: func(s widgets.WidgetStates) ui.Color {
				if s.Has(widgets.WidgetStateDisabled) {
					if s.Has(widgets.WidgetStateSelected) {
						return colors.OnSurface().WithOpacity(0.38)
					}
					return transparent
				}
				if s.Has(widgets.WidgetStateSelected) {
					if s.Has(widgets.WidgetStateError) {
						return colors.Error()
					}
					return colors.Primary()
				}
				return transparent
			},
			checkColor: func(s widgets.WidgetStates) ui.Color {
				if s.Has(widgets.WidgetStateDisabled) {
					if s.Has(widgets.WidgetStateSelected) {
						return colors.Surface()
					}
					return transparent // No icons available when the checkbox is unselected.
				}
				if s.Has(widgets.WidgetStateSelected) {
					if s.Has(widgets.WidgetStateError) {
						return colors.OnError()
					}
					return colors.OnPrimary()
				}
				return transparent // No icons available when the checkbox is unselected.
			},
			overlayColor: func(s widgets.WidgetStates) ui.Color {
				if s.Has(widgets.WidgetStateError) {
					switch {
					case s.Has(widgets.WidgetStatePressed):
						return colors.Error().WithOpacity(0.1)
					case s.Has(widgets.WidgetStateHovered):
						return colors.Error().WithOpacity(0.08)
					case s.Has(widgets.WidgetStateFocused):
						return colors.Error().WithOpacity(0.1)
					}
				}
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
			},
			splashRadius:          40.0 / 2,
			materialTapTargetSize: theme.MaterialTapTargetSize,
			visualDensity:         VisualDensityStandard(),
			shape:                 painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(2.0))}),
		}
	}
	d := checkboxDefaults{
		side: func(s widgets.WidgetStates) painting.BorderSide {
			if s.Has(widgets.WidgetStateDisabled) {
				if s.Has(widgets.WidgetStateSelected) {
					return border(2.0, transparent)
				}
				return border(2.0, theme.DisabledColor)
			}
			if s.Has(widgets.WidgetStateSelected) {
				return border(2.0, transparent)
			}
			return border(2.0, theme.UnselectedWidgetColor)
		},
		checkColor:            func(widgets.WidgetStates) ui.Color { return ui.NewColor(0xFFFFFFFF) },
		splashRadius:          RadialReactionRadius,
		materialTapTargetSize: theme.MaterialTapTargetSize,
		visualDensity:         theme.VisualDensity,
		shape:                 painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(1.0))}),
	}
	d.fillColor = func(s widgets.WidgetStates) ui.Color {
		if s.Has(widgets.WidgetStateDisabled) {
			if s.Has(widgets.WidgetStateSelected) {
				return theme.DisabledColor
			}
			return transparent
		}
		if s.Has(widgets.WidgetStateSelected) {
			return colors.Secondary()
		}
		return transparent
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
