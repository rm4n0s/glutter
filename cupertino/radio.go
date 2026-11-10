package cupertino

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The measures of the radio (the constants of cupertino/radio.dart).
const (
	radioOuterRadius           = 7.0
	radioInnerRadius           = 2.975
	radioPressedOverlayOpacity = 0.15
	radioCheckmarkStrokeWidth  = 2.0
	radioFocusOutlineWidth     = 3.0
	radioBorderOutlineWidth    = 0.3
)

// radioSize is the size of the radio.
var radioSize = ui.Size{Width: 18.0, Height: 18.0}

var (
	radioDisabledOuterColor  = CupertinoColors.White.WithOpacity(0.50)
	radioDisabledInnerColor  = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x40000000), ui.NewColor(0x40FFFFFF))
	radioDisabledBorderColor = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x40000000), ui.NewColor(0x40000000))
	radioDefaultBorderColor  = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFD1D1D6), ui.NewColor(0x40000000))
	radioDefaultInnerColor   = NewCupertinoDynamicColorWithBrightness(CupertinoColors.White, ui.NewColor(0xFFDEE8F8))
	radioDefaultOuterColor   = NewCupertinoDynamicColorWithBrightness(CupertinoColors.ActiveBlue.Color, ui.NewColor(0xFF3264D7))
)

// CupertinoRadioOpts are the optional parameters of NewCupertinoRadio.
//
// Flutter: cupertino/radio.dart CupertinoRadio.new
type CupertinoRadioOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// GroupValue is the selected value of the group of radios.
	//
	// Deprecated: Flutter deprecates it after v3.32; use a RadioGroup above the radios.
	GroupValue foundation.Opt[T]
	// OnChanged is called when the user selects this radio.
	//
	// Deprecated: Flutter deprecates it after v3.32; use a RadioGroup above the radios.
	OnChanged func(value foundation.Opt[T])
	// MouseCursor is the cursor over the radio, by state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// Toggleable makes the radio deselect when it is selected again.
	Toggleable bool
	// ActiveColor is the color of the outer circle of a radio that is selected.
	ActiveColor ui.ColorValue
	// InactiveColor is the color of the outer circle of a radio that is not selected.
	InactiveColor ui.ColorValue
	// FillColor is the color of the inner circle or the check mark of a radio that is selected.
	FillColor ui.ColorValue
	// FocusColor is the color of the outline of a radio that has the focus.
	FocusColor ui.ColorValue
	// FocusNode is the focus node of the radio.
	FocusNode *widgets.FocusNode
	// Autofocus asks for the focus when the radio is built.
	Autofocus bool
	// UseCheckmarkStyle shows a check mark in place of the inner circle when the radio is selected.
	UseCheckmarkStyle bool
	// Enabled says whether the radio answers input; the default is whether it has an OnChanged, a GroupRegistry or
	// a RadioGroup above.
	Enabled foundation.Opt[bool]
	// GroupRegistry is the registry of the group of the radio; the default is the one of the RadioGroup above.
	GroupRegistry widgets.RadioGroupRegistry[T]
}

// CupertinoRadio is an iOS styled radio button: it lets the user select one option of a set. Put the radios below a
// widgets.RadioGroup, which keeps the selected value.
//
// Flutter: cupertino/radio.dart CupertinoRadio
type CupertinoRadio[T comparable] struct {
	widgets.StatefulWidget
	value T
	o     CupertinoRadioOpts[T]
}

// NewCupertinoRadio creates a radio that stands for the value.
//
// Flutter: cupertino/radio.dart CupertinoRadio.new
func NewCupertinoRadio[T comparable](value T, opts ...CupertinoRadioOpts[T]) *CupertinoRadio[T] {
	o := firstOf(opts)
	return &CupertinoRadio[T]{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), value, o}
}

// Value is the value that this radio stands for.
//
// Flutter: cupertino/radio.dart CupertinoRadio.value
func (w *CupertinoRadio[T]) Value() T { return w.value }

// Toggleable is whether a selected radio is deselected by a tap.
//
// Flutter: cupertino/radio.dart CupertinoRadio.toggleable
func (w *CupertinoRadio[T]) Toggleable() bool { return w.o.Toggleable }

// UseCheckmarkStyle is whether the selected radio shows a check mark.
//
// Flutter: cupertino/radio.dart CupertinoRadio.useCheckmarkStyle
func (w *CupertinoRadio[T]) UseCheckmarkStyle() bool { return w.o.UseCheckmarkStyle }

// CreateState creates the state.
//
// Flutter: cupertino/radio.dart CupertinoRadio.createState
func (w *CupertinoRadio[T]) CreateState() widgets.State { return &cupertinoRadioState[T]{} }

type cupertinoRadioState[T comparable] struct {
	widgets.StateOf[*CupertinoRadio[T]]
	internalFocusNode *widgets.FocusNode
	internalRegistry  *cupertinoRadioRegistry[T]
}

func (s *cupertinoRadioState[T]) effectiveFocusNode() *widgets.FocusNode {
	if n := s.Widget().o.FocusNode; n != nil {
		return n
	}
	if s.internalFocusNode == nil {
		s.internalFocusNode = widgets.NewFocusNode()
	}
	return s.internalFocusNode
}

func (s *cupertinoRadioState[T]) enabled() bool {
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

func (s *cupertinoRadioState[T]) effectiveRegistry() widgets.RadioGroupRegistry[T] {
	if r := s.Widget().o.GroupRegistry; r != nil {
		return r
	}
	if r, ok := widgets.RadioGroupMaybeOf[T](s.Context()); ok {
		return r
	}
	if s.internalRegistry == nil {
		s.internalRegistry = &cupertinoRadioRegistry[T]{state: s}
	}
	return s.internalRegistry
}

func (s *cupertinoRadioState[T]) Dispose() {
	if s.internalFocusNode != nil {
		s.internalFocusNode.Dispose()
	}
}

func (s *cupertinoRadioState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	if foundation.DebugMode && w.o.Enabled == foundation.Some(true) && w.o.OnChanged == nil && w.o.GroupRegistry == nil {
		if _, ok := widgets.RadioGroupMaybeOf[T](bc); !ok {
			panic(errors.New(tagCupertinoRadioNoRegistry, "CupertinoRadio is enabled but has no OnChanged, GroupRegistry, or RadioGroup above"))
		}
	}
	cursor := widgets.WidgetStatePropertyResolveWith(func(st widgets.WidgetStates) services.MouseCursor {
		if w.o.MouseCursor != nil {
			if c := w.o.MouseCursor.Resolve(st); c != nil {
				return c
			}
		}
		if !st.Has(widgets.WidgetStateDisabled) && foundation.IsWeb {
			return services.SystemMouseCursors.Click
		}
		return services.SystemMouseCursors.Basic
	})
	enabled := s.enabled()
	node := s.effectiveFocusNode()
	return widgets.NewRawRadio(w.value, cursor, w.o.Toggleable, node, w.o.Autofocus, s.effectiveRegistry(), enabled,
		func(_ widgets.BuildContext, state *widgets.ToggleableStateMixin) widgets.Widget {
			return &cupertinoRadioPaint{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}), state, w.o.ActiveColor, w.o.InactiveColor,
				w.o.FillColor, w.o.FocusColor, w.o.UseCheckmarkStyle, enabled, node.HasFocus()}
		})
}

// cupertinoRadioRegistry is the registry of a radio that has no RadioGroup above: its GroupValue and its OnChanged.
//
// Flutter: cupertino/radio.dart _RadioRegistry
type cupertinoRadioRegistry[T comparable] struct{ state *cupertinoRadioState[T] }

func (r *cupertinoRadioRegistry[T]) GroupValue() foundation.Opt[T] {
	return r.state.Widget().o.GroupValue
}
func (r *cupertinoRadioRegistry[T]) RegisterClient(widgets.RadioClient[T])   {}
func (r *cupertinoRadioRegistry[T]) UnregisterClient(widgets.RadioClient[T]) {}
func (r *cupertinoRadioRegistry[T]) OnChanged(value foundation.Opt[T]) {
	if f := r.state.Widget().o.OnChanged; f != nil {
		f(value)
	}
}

// cupertinoRadioPaint paints a radio from the state of its toggle.
//
// Flutter: cupertino/radio.dart _RadioPaint
type cupertinoRadioPaint struct {
	widgets.StatefulWidget
	toggleableState                                   *widgets.ToggleableStateMixin
	activeColor, inactiveColor, fillColor, focusColor ui.ColorValue
	useCheckmarkStyle, isActive, focused              bool
}

func (w *cupertinoRadioPaint) CreateState() widgets.State { return &cupertinoRadioPaintState{} }

type cupertinoRadioPaintState struct {
	widgets.StateOf[*cupertinoRadioPaint]
	painter *cupertinoRadioPainter
}

func (s *cupertinoRadioPaintState) InitState() { s.painter = &cupertinoRadioPainter{} }
func (s *cupertinoRadioPaintState) Dispose()   { s.painter.Dispose() }

func (s *cupertinoRadioPaintState) defaultOuterColor(states widgets.WidgetStates) ui.Color {
	w := s.Widget()
	switch {
	case states.Has(widgets.WidgetStateDisabled):
		return radioDisabledOuterColor
	case states.Has(widgets.WidgetStateSelected):
		if w.activeColor != nil {
			return w.activeColor.EffectiveColor()
		}
		return CupertinoDynamicColorResolve(radioDefaultOuterColor, s.Context())
	}
	if w.inactiveColor != nil {
		return w.inactiveColor.EffectiveColor()
	}
	return CupertinoColors.White
}

func (s *cupertinoRadioPaintState) defaultInnerColor(states widgets.WidgetStates) ui.Color {
	w := s.Widget()
	switch {
	case states.Has(widgets.WidgetStateDisabled) && states.Has(widgets.WidgetStateSelected):
		if w.fillColor != nil {
			return w.fillColor.EffectiveColor()
		}
		return CupertinoDynamicColorResolve(radioDisabledInnerColor, s.Context())
	case states.Has(widgets.WidgetStateSelected):
		if w.fillColor != nil {
			return w.fillColor.EffectiveColor()
		}
		return CupertinoDynamicColorResolve(radioDefaultInnerColor, s.Context())
	}
	return CupertinoColors.White
}

func (s *cupertinoRadioPaintState) defaultBorderColor(states widgets.WidgetStates) ui.Color {
	switch {
	case (states.Has(widgets.WidgetStateSelected) || states.Has(widgets.WidgetStateFocused)) && !states.Has(widgets.WidgetStateDisabled):
		return CupertinoColors.Transparent
	case states.Has(widgets.WidgetStateDisabled):
		return CupertinoDynamicColorResolve(radioDisabledBorderColor, s.Context())
	}
	return CupertinoDynamicColorResolve(radioDefaultBorderColor, s.Context())
}

func (s *cupertinoRadioPaintState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	ts := w.toggleableState
	states := ts.States()
	activeStates := states.With(widgets.WidgetStateSelected)
	inactiveStates := states.Without(widgets.WidgetStateSelected)

	effectiveActiveColor := s.defaultOuterColor(activeStates)
	effectiveInactiveColor := s.defaultOuterColor(inactiveStates)
	effectiveFocusOverlayColor := painting.NewHSLColorFromColor(effectiveActiveColor.WithOpacity(CupertinoFocusColorOpacity)).
		WithLightness(CupertinoFocusColorBrightness).WithSaturation(CupertinoFocusColorSaturation).ToColor()
	if w.focusColor != nil {
		effectiveFocusOverlayColor = w.focusColor.EffectiveColor()
	}

	p := s.painter
	p.SetPosition(ts.Position())
	p.SetReaction(ts.Reaction())
	p.SetFocusColor(effectiveFocusOverlayColor)
	p.SetDownPosition(ts.DownPosition())
	p.SetIsFocused(w.focused)
	p.SetActiveColor(effectiveActiveColor)
	p.SetInactiveColor(effectiveInactiveColor)
	p.set(cupertinoRadioPainterValues{
		fillColor: s.defaultInnerColor(states), value: ts.Value(), checkmarkStyle: w.useCheckmarkStyle, isActive: w.isActive,
		borderColor: s.defaultBorderColor(states), brightness: CupertinoThemeOf(bc).Brightness(),
	})
	return widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Size: radioSize, Painter: p})
}

// cupertinoRadioPainterValues are the values of a cupertinoRadioPainter that the state sets in every build.
type cupertinoRadioPainterValues struct {
	fillColor, borderColor ui.Color
	value                  foundation.Opt[bool]
	checkmarkStyle         bool
	isActive               bool
	brightness             ui.Brightness
}

// cupertinoRadioPainter paints the circles or the check mark of a CupertinoRadio.
//
// Flutter: cupertino/radio.dart _RadioPainter
type cupertinoRadioPainter struct {
	widgets.ToggleablePainter
	v cupertinoRadioPainterValues
}

func (p *cupertinoRadioPainter) set(v cupertinoRadioPainterValues) {
	if p.v != v {
		p.v = v
		p.NotifyListeners()
	}
}

func (p *cupertinoRadioPainter) drawPressedOverlay(canvas *ui.Canvas, center ui.Offset, radius float64) {
	pressed := ui.NewPaint()
	if p.v.brightness == ui.BrightnessLight {
		pressed.SetColor(CupertinoColors.Black.WithOpacity(radioPressedOverlayOpacity))
	} else {
		pressed.SetColor(CupertinoColors.White.WithOpacity(radioPressedOverlayOpacity))
	}
	canvas.DrawCircle(center, radius, pressed)
}

func (p *cupertinoRadioPainter) drawFillGradient(canvas *ui.Canvas, center ui.Offset, radius float64, topColor, bottomColor ui.Color) {
	fillGradient := painting.NewLinearGradient([]ui.Color{topColor, bottomColor}, painting.LinearGradientOpts{
		Begin: painting.AlignmentTopCenter, End: painting.AlignmentBottomCenter})
	circleRect := ui.NewRectFromCircle(center, radius)
	gradientPaint := ui.NewPaint()
	gradientPaint.SetShader(fillGradient.CreateShader(circleRect, ui.TextDirectionLTR))
	path := ui.NewPath()
	path.AddOval(circleRect)
	canvas.DrawPath(path, gradientPaint)
}

func (p *cupertinoRadioPainter) drawOuterBorder(canvas *ui.Canvas, center ui.Offset) {
	borderPaint := ui.NewPaint()
	borderPaint.SetStyle(ui.PaintingStyleStroke)
	borderPaint.SetColor(p.v.borderColor)
	borderPaint.SetStrokeWidth(radioBorderOutlineWidth)
	canvas.DrawCircle(center, radioOuterRadius, borderPaint)
}

func gradientOpacities(active bool) [2]float64 {
	if active {
		return checkboxDarkGradientOpacities
	}
	return checkboxDisabledDarkGradientOpacities
}

// Paint paints the radio.
//
// Flutter: cupertino/radio.dart _RadioPainter.paint
func (p *cupertinoRadioPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	center := ui.NewRectFromLTWH(0, 0, size.Width, size.Height).Center()
	selected := p.v.value.OrElse(false)
	switch {
	case p.v.checkmarkStyle:
		if selected {
			path := ui.NewPath()
			checkPaint := ui.NewPaint()
			checkPaint.SetColor(p.ActiveColor())
			checkPaint.SetStyle(ui.PaintingStyleStroke)
			checkPaint.SetStrokeWidth(radioCheckmarkStrokeWidth)
			checkPaint.SetStrokeCap(ui.StrokeCapRound)
			width := radioSize.Width
			origin := ui.Offset{Dx: center.Dx - width/2, Dy: center.Dy - width/2}
			start := ui.Offset{Dx: width * 0.25, Dy: width * 0.52}
			mid := ui.Offset{Dx: width * 0.46, Dy: width * 0.75}
			end := ui.Offset{Dx: width * 0.85, Dy: width * 0.29}
			path.MoveTo(origin.Dx+start.Dx, origin.Dy+start.Dy)
			path.LineTo(origin.Dx+mid.Dx, origin.Dy+mid.Dy)
			canvas.DrawPath(path, checkPaint)
			path.MoveTo(origin.Dx+mid.Dx, origin.Dy+mid.Dy)
			path.LineTo(origin.Dx+end.Dx, origin.Dy+end.Dy)
			canvas.DrawPath(path, checkPaint)
		}
	case selected:
		outerPaint := ui.NewPaint()
		outerPaint.SetColor(p.ActiveColor())
		if p.v.brightness == ui.BrightnessDark && !p.v.isActive {
			o := gradientOpacities(p.v.isActive)
			p.drawFillGradient(canvas, center, radioOuterRadius, p.ActiveColor().WithOpacity(o[0]), p.ActiveColor().WithOpacity(o[1]))
		} else {
			canvas.DrawCircle(center, radioOuterRadius, outerPaint)
		}
		if p.DownPosition().IsSome() {
			p.drawPressedOverlay(canvas, center, radioOuterRadius)
		}
		innerPaint := ui.NewPaint()
		innerPaint.SetColor(p.v.fillColor)
		canvas.DrawCircle(center, radioInnerRadius, innerPaint)
		if !p.v.isActive {
			p.drawOuterBorder(canvas, center)
		}
	default:
		paint := ui.NewPaint()
		color := radioDisabledOuterColor
		if p.v.isActive {
			color = p.InactiveColor()
		}
		paint.SetColor(color)
		if p.v.brightness == ui.BrightnessDark {
			o := gradientOpacities(p.v.isActive)
			p.drawFillGradient(canvas, center, radioOuterRadius, color.WithOpacity(o[0]), color.WithOpacity(o[1]))
		} else {
			canvas.DrawCircle(center, radioOuterRadius, paint)
		}
		if p.DownPosition().IsSome() {
			p.drawPressedOverlay(canvas, center, radioOuterRadius)
		}
		p.drawOuterBorder(canvas, center)
	}
	if p.IsFocused() {
		focusPaint := ui.NewPaint()
		focusPaint.SetStyle(ui.PaintingStyleStroke)
		focusPaint.SetColor(p.FocusColor())
		focusPaint.SetStrokeWidth(radioFocusOutlineWidth)
		canvas.DrawCircle(center, radioOuterRadius+radioFocusOutlineWidth/2, focusPaint)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagCupertinoRadioNoRegistry = "CupertinoRadioNoRegistry"
