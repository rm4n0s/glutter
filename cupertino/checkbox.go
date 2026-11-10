package cupertino

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

var (
	checkboxDisabledCheckColor  = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x40000000), ui.NewColor(0x40FFFFFF))
	checkboxDisabledBorderColor = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x0D000000), ui.NewColor(0x0D000000))
	checkboxDefaultBorderColor  = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFD1D1D6), ui.NewColor(0x32808080))
	checkboxDefaultFillColor    = NewCupertinoDynamicColorWithBrightness(CupertinoColors.ActiveBlue.Color, ui.NewColor(0xFF3264D7))
	checkboxDefaultCheckColor   = NewCupertinoDynamicColorWithBrightness(CupertinoColors.White, ui.NewColor(0xFFDEE8F8))
)

const checkboxPressedOverlayOpacity = 0.15

var (
	checkboxDarkGradientOpacities         = [2]float64{0.14, 0.29}
	checkboxDisabledDarkGradientOpacities = [2]float64{0.08, 0.14}
)

// CupertinoCheckboxWidth is the width of the box of a checkbox.
//
// Flutter: cupertino/checkbox.dart CupertinoCheckbox.width
const CupertinoCheckboxWidth = 14.0

// CupertinoCheckboxOpts are the optional parameters of NewCupertinoCheckbox.
//
// Flutter: cupertino/checkbox.dart CupertinoCheckbox.new
type CupertinoCheckboxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Tristate says that the value can also be none (mixed) and that a tap cycles through the three.
	Tristate bool
	// MouseCursor is the cursor over the checkbox, by state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// ActiveColor is the fill of a checkbox that is checked; the default is a dynamic blue.
	ActiveColor ui.ColorValue
	// FillColor is the fill of the checkbox in each state; it takes the place of the default fills.
	FillColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// CheckColor is the color of the check mark.
	CheckColor ui.ColorValue
	// FocusColor is the color of the outline of a checkbox that has the focus.
	FocusColor ui.ColorValue
	// FocusNode is the focus node of the checkbox.
	FocusNode *widgets.FocusNode
	// Autofocus asks for the focus when the checkbox is built.
	Autofocus bool
	// Side is the border of the checkbox when it is not checked.
	Side foundation.Opt[painting.BorderSide]
	// StateSide is the border of the checkbox in each state; when it is set, Side is ignored.
	StateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// Shape is the shape of the box; the default is a rounded rectangle with a radius of 4.
	Shape painting.OutlinedBorder
	// TapTargetSize is the size of the area that takes taps; the default is 44 on the phones and 14 on the desktop.
	TapTargetSize foundation.Opt[ui.Size]
	// SemanticLabel is the semantics label of the checkbox.
	SemanticLabel string
	// Deviation: no inactiveColor (deprecated in Flutter, replaced by FillColor).
}

// CupertinoCheckbox is an iOS styled checkbox: it lets the user select an option of a set, and with Tristate it has a
// third, mixed value. It does not keep its value: the widget that makes it listens to OnChanged and rebuilds it.
//
// Flutter: cupertino/checkbox.dart CupertinoCheckbox
type CupertinoCheckbox struct {
	widgets.StatefulWidget
	value     foundation.Opt[bool]
	onChanged func(foundation.Opt[bool])
	o         CupertinoCheckboxOpts
}

// NewCupertinoCheckbox creates a checkbox; a nil onChanged disables it. In debug builds it panics with the tag
// CupertinoCheckboxValueNull when the value is none and the checkbox is not tristate.
//
// Flutter: cupertino/checkbox.dart CupertinoCheckbox.new
func NewCupertinoCheckbox(value foundation.Opt[bool], onChanged func(foundation.Opt[bool]), opts ...CupertinoCheckboxOpts) *CupertinoCheckbox {
	o := firstOf(opts)
	if foundation.DebugMode && !o.Tristate && value.IsNone() {
		panic(errors.New(tagCupertinoCheckboxValueNull, "the value of a CupertinoCheckbox that is not tristate can't be none"))
	}
	return &CupertinoCheckbox{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), value, onChanged, o}
}

// Value is whether the checkbox is checked; none is the mixed value.
//
// Flutter: cupertino/checkbox.dart CupertinoCheckbox.value
func (w *CupertinoCheckbox) Value() foundation.Opt[bool] { return w.value }

// OnChanged is the callback for a change, or nil for a disabled checkbox.
//
// Flutter: cupertino/checkbox.dart CupertinoCheckbox.onChanged
func (w *CupertinoCheckbox) OnChanged() func(foundation.Opt[bool]) { return w.onChanged }

// Tristate is whether the value can be mixed.
//
// Flutter: cupertino/checkbox.dart CupertinoCheckbox.tristate
func (w *CupertinoCheckbox) Tristate() bool { return w.o.Tristate }

// CreateState creates the state.
//
// Flutter: cupertino/checkbox.dart CupertinoCheckbox.createState
func (w *CupertinoCheckbox) CreateState() widgets.State { return &cupertinoCheckboxState{} }

type cupertinoCheckboxState struct {
	widgets.StateOf[*CupertinoCheckbox]
	widgets.TickerProvider
	widgets.ToggleableStateMixin
	painter       *cupertinoCheckboxPainter
	previousValue foundation.Opt[bool]
}

func (s *cupertinoCheckboxState) InitState() {
	s.painter = &cupertinoCheckboxPainter{}
	s.previousValue = s.Widget().value
	s.Init(s, s)
}

func (s *cupertinoCheckboxState) DidUpdateWidget(old *CupertinoCheckbox) {
	if old.value != s.Widget().value {
		s.previousValue = old.value
	}
}

func (s *cupertinoCheckboxState) Dispose() {
	s.painter.Dispose()
	s.DisposeToggleable()
}

func (s *cupertinoCheckboxState) ToggleableOnChanged() func(foundation.Opt[bool]) {
	return s.Widget().onChanged
}
func (s *cupertinoCheckboxState) ToggleableValue() foundation.Opt[bool] { return s.Widget().value }
func (s *cupertinoCheckboxState) ToggleableTristate() bool              { return s.Widget().o.Tristate }

// defaultFillColor is the fill of the checkbox in the states, when the widget gives none.
//
// Flutter: cupertino/checkbox.dart _CupertinoCheckboxState._defaultFillColor
func (s *cupertinoCheckboxState) defaultFillColor(states widgets.WidgetStates) ui.Color {
	switch {
	case states.Has(widgets.WidgetStateDisabled):
		return CupertinoColors.White.WithOpacity(0.5)
	case states.Has(widgets.WidgetStateSelected):
		if s.Widget().o.ActiveColor != nil {
			return s.Widget().o.ActiveColor.EffectiveColor()
		}
		return CupertinoDynamicColorResolve(checkboxDefaultFillColor, s.Context())
	}
	return CupertinoColors.White
}

// defaultCheckColor is the color of the check mark in the states.
//
// Flutter: cupertino/checkbox.dart _CupertinoCheckboxState._defaultCheckColor
func (s *cupertinoCheckboxState) defaultCheckColor(states widgets.WidgetStates) ui.Color {
	given := s.Widget().o.CheckColor
	switch {
	case states.Has(widgets.WidgetStateDisabled) && states.Has(widgets.WidgetStateSelected):
		if given != nil {
			return given.EffectiveColor()
		}
		return CupertinoDynamicColorResolve(checkboxDisabledCheckColor, s.Context())
	case states.Has(widgets.WidgetStateSelected):
		if given != nil {
			return given.EffectiveColor()
		}
		return CupertinoDynamicColorResolve(checkboxDefaultCheckColor, s.Context())
	}
	return CupertinoColors.White
}

// defaultSide is the border of the checkbox in the states.
//
// Flutter: cupertino/checkbox.dart _CupertinoCheckboxState._defaultSide
func (s *cupertinoCheckboxState) defaultSide(states widgets.WidgetStates) painting.BorderSide {
	switch {
	case (states.Has(widgets.WidgetStateSelected) || states.Has(widgets.WidgetStateFocused)) && !states.Has(widgets.WidgetStateDisabled):
		return painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(0.0), Color: foundation.Some(CupertinoColors.Transparent)})
	case states.Has(widgets.WidgetStateDisabled):
		return painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(CupertinoDynamicColorResolve(checkboxDisabledBorderColor, s.Context()))})
	}
	return painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(CupertinoDynamicColorResolve(checkboxDefaultBorderColor, s.Context()))})
}

func (s *cupertinoCheckboxState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	states := s.States()
	activeStates := states.With(widgets.WidgetStateSelected)
	inactiveStates := states.Without(widgets.WidgetStateSelected)

	fill := func(st widgets.WidgetStates) ui.Color {
		if w.o.FillColor != nil {
			if c, ok := w.o.FillColor.Resolve(st).Get(); ok {
				return c
			}
		}
		return s.defaultFillColor(st)
	}
	effectiveActiveColor, effectiveInactiveColor := fill(activeStates), fill(inactiveStates)

	effectiveBorderSide := s.defaultSide(states)
	if side, ok := resolveCheckboxSide(w.o.Side, w.o.StateSide, states).Get(); ok {
		effectiveBorderSide = side
	}

	effectiveFocusOverlayColor := painting.NewHSLColorFromColor(effectiveActiveColor.WithOpacity(CupertinoFocusColorOpacity)).
		WithLightness(CupertinoFocusColorBrightness).WithSaturation(CupertinoFocusColorSaturation).ToColor()
	if w.o.FocusColor != nil {
		effectiveFocusOverlayColor = w.o.FocusColor.EffectiveColor()
	}

	mouseCursor := widgets.WidgetStatePropertyResolveWith(func(st widgets.WidgetStates) services.MouseCursor {
		if w.o.MouseCursor != nil {
			if c := w.o.MouseCursor.Resolve(st); c != nil {
				return c
			}
		}
		if foundation.IsWeb && !st.Has(widgets.WidgetStateDisabled) {
			return services.SystemMouseCursors.Click
		}
		return services.SystemMouseCursors.Basic
	})

	effectiveSize := w.o.TapTargetSize.OrElse(ui.Size{Width: CupertinoCheckboxWidth, Height: CupertinoCheckboxWidth})
	if w.o.TapTargetSize.IsNone() {
		switch foundation.DefaultTargetPlatform() {
		case foundation.TargetPlatformIOS, foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
			effectiveSize = ui.Size{Width: MinInteractiveDimensionCupertino, Height: MinInteractiveDimensionCupertino}
		}
	}

	shape := w.o.Shape
	if shape == nil {
		shape = painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(4.0))})
	}

	p := s.painter
	p.SetPosition(s.Position())
	p.SetReaction(s.Reaction())
	p.SetFocusColor(effectiveFocusOverlayColor)
	p.SetDownPosition(s.DownPosition())
	p.SetIsFocused(states.Has(widgets.WidgetStateFocused))
	p.SetIsHovered(states.Has(widgets.WidgetStateHovered))
	p.SetActiveColor(effectiveActiveColor)
	p.SetInactiveColor(effectiveInactiveColor)
	p.set(cupertinoCheckboxPainterValues{
		checkColor: s.defaultCheckColor(states), value: w.value, previousValue: s.previousValue, isActive: w.onChanged != nil,
		shape: shape, side: effectiveBorderSide, brightness: CupertinoThemeOf(bc).Brightness(),
	})

	var mixed foundation.Opt[bool]
	if w.o.Tristate {
		mixed = foundation.Some(w.value.IsNone())
	}
	opts := widgets.SemanticsOpts{Checked: foundation.Some(w.value.OrElse(false)), Mixed: mixed}
	if w.o.SemanticLabel != "" {
		opts.Label = foundation.Some(w.o.SemanticLabel)
	}
	return widgets.NewSemantics(s.BuildToggleable(effectiveSize, p, widgets.ToggleableOpts{
		MouseCursor: mouseCursor, FocusNode: w.o.FocusNode, Autofocus: w.o.Autofocus}), opts)
}

// resolveCheckboxSide is the border that the widget gives for the states: the property when there is one, and the
// plain side when the checkbox is not selected.
//
// Flutter: cupertino/checkbox.dart _CupertinoCheckboxState._resolveSide
func resolveCheckboxSide(side foundation.Opt[painting.BorderSide], stateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]], states widgets.WidgetStates) foundation.Opt[painting.BorderSide] {
	if stateSide != nil {
		return stateSide.Resolve(states)
	}
	if !states.Has(widgets.WidgetStateSelected) {
		return side
	}
	return foundation.None[painting.BorderSide]()
}

// cupertinoCheckboxPainterValues are the values of a cupertinoCheckboxPainter that the state sets in every build.
type cupertinoCheckboxPainterValues struct {
	checkColor    ui.Color
	value         foundation.Opt[bool]
	previousValue foundation.Opt[bool]
	isActive      bool
	shape         painting.OutlinedBorder
	side          painting.BorderSide
	brightness    ui.Brightness
}

// cupertinoCheckboxPainter paints the box and the mark of a CupertinoCheckbox.
//
// Flutter: cupertino/checkbox.dart _CheckboxPainter
type cupertinoCheckboxPainter struct {
	widgets.ToggleablePainter
	v cupertinoCheckboxPainterValues
}

func (p *cupertinoCheckboxPainter) set(v cupertinoCheckboxPainterValues) {
	if p.v.checkColor != v.checkColor || p.v.value != v.value || p.v.previousValue != v.previousValue || p.v.isActive != v.isActive ||
		!foundation.ValuesEqual(p.v.shape, v.shape) || p.v.side != v.side || p.v.brightness != v.brightness {
		p.v = v
		p.NotifyListeners()
	}
}

func (p *cupertinoCheckboxPainter) colorAt(value bool) ui.Color {
	if value && p.v.isActive {
		return p.ActiveColor()
	}
	return p.InactiveColor()
}

func (p *cupertinoCheckboxPainter) createStrokePaint() *ui.Paint {
	paint := ui.NewPaint()
	paint.SetColor(p.v.checkColor)
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(2.0)
	paint.SetStrokeCap(ui.StrokeCapRound)
	return paint
}

func (p *cupertinoCheckboxPainter) fillInterior(canvas *ui.Canvas, outer ui.Rect, paint *ui.Paint) {
	if p.v.shape.PreferPaintInterior() {
		p.v.shape.PaintInterior(canvas, outer, paint, ui.TextDirectionLTR)
	} else {
		canvas.DrawPath(p.v.shape.GetOuterPath(outer, ui.TextDirectionLTR), paint)
	}
}

func (p *cupertinoCheckboxPainter) drawFillGradient(canvas *ui.Canvas, outer ui.Rect, topColor, bottomColor ui.Color) {
	fillGradient := painting.NewLinearGradient([]ui.Color{topColor, bottomColor}, painting.LinearGradientOpts{
		Begin: painting.AlignmentTopCenter, End: painting.AlignmentBottomCenter})
	gradientPaint := ui.NewPaint()
	gradientPaint.SetShader(fillGradient.CreateShader(outer, ui.TextDirectionLTR))
	p.fillInterior(canvas, outer, gradientPaint)
}

func (p *cupertinoCheckboxPainter) drawBox(canvas *ui.Canvas, outer ui.Rect, paint *ui.Paint, side painting.BorderSide, value bool) {
	if p.v.brightness == ui.BrightnessDark && !(p.v.isActive && value) {
		opacities := checkboxDisabledDarkGradientOpacities
		if p.v.isActive {
			opacities = checkboxDarkGradientOpacities
		}
		p.drawFillGradient(canvas, outer, paint.Color().WithOpacity(opacities[0]), paint.Color().WithOpacity(opacities[1]))
	} else {
		p.fillInterior(canvas, outer, paint)
	}
	p.v.shape.CopyWithSide(side).Paint(canvas, outer, ui.TextDirectionLTR)
}

func (p *cupertinoCheckboxPainter) drawCheck(canvas *ui.Canvas, origin ui.Offset, paint *ui.Paint) {
	path := ui.NewPath()
	const w = CupertinoCheckboxWidth
	start := ui.Offset{Dx: w * 0.22, Dy: w * 0.54}
	mid := ui.Offset{Dx: w * 0.40, Dy: w * 0.75}
	end := ui.Offset{Dx: w * 0.78, Dy: w * 0.25}
	path.MoveTo(origin.Dx+start.Dx, origin.Dy+start.Dy)
	path.LineTo(origin.Dx+mid.Dx, origin.Dy+mid.Dy)
	path.MoveTo(origin.Dx+mid.Dx, origin.Dy+mid.Dy)
	path.LineTo(origin.Dx+end.Dx, origin.Dy+end.Dy)
	canvas.DrawPath(path, paint)
}

func (p *cupertinoCheckboxPainter) drawDash(canvas *ui.Canvas, origin ui.Offset, paint *ui.Paint) {
	const w = CupertinoCheckboxWidth
	canvas.DrawLine(origin.Add(ui.Offset{Dx: w * 0.25, Dy: w * 0.5}), origin.Add(ui.Offset{Dx: w * 0.75, Dy: w * 0.5}), paint)
}

// Paint paints the box, the mark, the pressed overlay and the focus outline.
//
// Flutter: cupertino/checkbox.dart _CheckboxPainter.paint
func (p *cupertinoCheckboxPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	strokePaint := p.createStrokePaint()
	origin := ui.Offset{Dx: size.Width/2.0 - CupertinoCheckboxWidth/2.0, Dy: size.Height/2.0 - CupertinoCheckboxWidth/2.0}
	outer := ui.NewRectFromLTWH(origin.Dx, origin.Dy, CupertinoCheckboxWidth, CupertinoCheckboxWidth)
	value, hasValue := p.v.value.Get()
	paint := ui.NewPaint()
	paint.SetColor(p.colorAt(!hasValue || value))

	p.drawBox(canvas, outer, paint, p.v.side, !hasValue || value)
	switch {
	case !hasValue:
		p.drawDash(canvas, origin, strokePaint)
	case value:
		p.drawCheck(canvas, origin, strokePaint)
	}
	if p.DownPosition().IsSome() {
		pressed := ui.NewPaint()
		if p.v.brightness == ui.BrightnessLight {
			pressed.SetColor(CupertinoColors.Black.WithOpacity(checkboxPressedOverlayOpacity))
		} else {
			pressed.SetColor(CupertinoColors.White.WithOpacity(checkboxPressedOverlayOpacity))
		}
		p.fillInterior(canvas, outer, pressed)
	}
	if p.IsFocused() {
		focusOuter := outer.Inflate(1)
		borderPaint := ui.NewPaint()
		borderPaint.SetColor(p.FocusColor())
		borderPaint.SetStyle(ui.PaintingStyleStroke)
		borderPaint.SetStrokeWidth(3.5)
		p.drawBox(canvas, focusOuter, borderPaint, p.v.side, !hasValue || value)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagCupertinoCheckboxValueNull = "CupertinoCheckboxValueNull"
