// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"fmt"
	"math"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// StepState is the state of a Step: how its header looks.
//
// Flutter: material/stepper.dart StepState
type StepState int

const (
	// StepStateUnspecified is the default, indexed.
	StepStateUnspecified StepState = iota
	// StepStateIndexed shows the number of the step.
	//
	// Flutter: material/stepper.dart StepState.indexed
	StepStateIndexed
	// StepStateEditing shows a pencil.
	//
	// Flutter: material/stepper.dart StepState.editing
	StepStateEditing
	// StepStateComplete shows a check mark.
	//
	// Flutter: material/stepper.dart StepState.complete
	StepStateComplete
	// StepStateDisabled shows the number of the step, dimmed, and the step cannot be
	// tapped.
	//
	// Flutter: material/stepper.dart StepState.disabled
	StepStateDisabled
	// StepStateError shows an exclamation mark in a triangle, with the title in the
	// error color.
	//
	// Flutter: material/stepper.dart StepState.error
	StepStateError
)

// Name is the name of the state.
func (s StepState) Name() string {
	return enumName(int(s), "unspecified", "indexed", "editing", "complete", "disabled", "error")
}

// String is the name of the type and of the state.
func (s StepState) String() string { return "StepState." + s.Name() }

func (s StepState) effective() StepState {
	if s == StepStateUnspecified {
		return StepStateIndexed
	}
	return s
}

// StepperType is how the steps of a Stepper are laid out.
//
// Flutter: material/stepper.dart StepperType
type StepperType int

const (
	// StepperTypeUnspecified is the default, vertical.
	StepperTypeUnspecified StepperType = iota
	// StepperTypeVertical shows the steps in a column, each with its content below.
	//
	// Flutter: material/stepper.dart StepperType.vertical
	StepperTypeVertical
	// StepperTypeHorizontal shows the headers of the steps in a row, with the
	// content of the current step below them.
	//
	// Flutter: material/stepper.dart StepperType.horizontal
	StepperTypeHorizontal
)

// Name is the name of the type.
func (t StepperType) Name() string { return enumName(int(t), "unspecified", "vertical", "horizontal") }

// String is the name of the type and of the value.
func (t StepperType) String() string { return "StepperType." + t.Name() }

// ControlsDetails are the details that the controls builder of a Stepper gets.
//
// Flutter: material/stepper.dart ControlsDetails
type ControlsDetails struct {
	// CurrentStep is the index of the step that is current in the stepper.
	CurrentStep int
	// StepIndex is the index of the step that the controls are for.
	StepIndex int
	// OnStepContinue is called to go to the next step.
	OnStepContinue func()
	// OnStepCancel is called to cancel the current step.
	OnStepCancel func()
}

// IsActive is whether the controls are for the current step.
//
// Flutter: material/stepper.dart ControlsDetails.isActive
func (d ControlsDetails) IsActive() bool { return d.CurrentStep == d.StepIndex }

// ControlsWidgetBuilder builds the controls of a step.
//
// Flutter: material/stepper.dart ControlsWidgetBuilder
type ControlsWidgetBuilder func(bc widgets.BuildContext, details ControlsDetails) widgets.Widget

// StepIconBuilder builds the icon of the header of a step; it returns nil for the
// default one.
//
// Flutter: material/stepper.dart StepIconBuilder
type StepIconBuilder func(stepIndex int, stepState StepState) widgets.Widget

// StepStyle is the style of a Step: its colors and its border.
//
// Flutter: material/stepper.dart StepStyle
type StepStyle struct {
	// Color is the color of the circle of the step.
	Color foundation.Opt[ui.Color]
	// ErrorColor is the color of the triangle of the step in the error state.
	ErrorColor foundation.Opt[ui.Color]
	// ConnectorColor is the color of the line that connects the step to the next.
	ConnectorColor foundation.Opt[ui.Color]
	// ConnectorThickness is the thickness of that line.
	ConnectorThickness foundation.Opt[float64]
	// Border is the border of the circle.
	Border painting.BoxBorder
	// BoxShadow is the shadow of the circle.
	BoxShadow foundation.Opt[painting.BoxShadow]
	// Gradient is the gradient of the circle.
	Gradient painting.Gradient
	// IndexStyle is the style of the number in the circle.
	IndexStyle *painting.TextStyle
}

// StepStyleCopy are the parameters of StepStyle.CopyWith.
//
// Flutter: material/stepper.dart StepStyle.copyWith
type StepStyleCopy = StepStyle

// CopyWith makes a copy of the style with the fields that the parameter sets
// replaced.
//
// Flutter: material/stepper.dart StepStyle.copyWith
func (s StepStyle) CopyWith(c StepStyleCopy) StepStyle {
	return StepStyle{
		Color: firstOpt(c.Color, s.Color), ErrorColor: firstOpt(c.ErrorColor, s.ErrorColor), ConnectorColor: firstOpt(c.ConnectorColor, s.ConnectorColor),
		ConnectorThickness: firstOpt(c.ConnectorThickness, s.ConnectorThickness), Border: firstNonNil(c.Border, s.Border),
		BoxShadow: firstOpt(c.BoxShadow, s.BoxShadow), Gradient: firstNonNil(c.Gradient, s.Gradient), IndexStyle: firstNonNil(c.IndexStyle, s.IndexStyle)}
}

// Merge is the style with the fields of the other style set over its own.
//
// Flutter: material/stepper.dart StepStyle.merge
func (s StepStyle) Merge(o *StepStyle) StepStyle {
	if o == nil {
		return s
	}
	return s.CopyWith(*o)
}

// Equal is whether the two styles are the same.
//
// Flutter: material/stepper.dart StepStyle.==
func (s StepStyle) Equal(o StepStyle) bool {
	return s.Color == o.Color && s.ErrorColor == o.ErrorColor && s.ConnectorColor == o.ConnectorColor && s.ConnectorThickness == o.ConnectorThickness &&
		foundation.ValuesEqual(s.Border, o.Border) && s.BoxShadow == o.BoxShadow && foundation.ValuesEqual(s.Gradient, o.Gradient) && textStyleEqual(s.IndexStyle, o.IndexStyle)
}

// StepOpts are the optional parameters of NewStep.
//
// Flutter: material/stepper.dart Step
type StepOpts struct {
	// Subtitle is the widget under the title.
	Subtitle widgets.Widget
	// State is the state of the step; the default is indexed.
	State StepState
	// IsActive is whether the step is active: its circle has the primary color.
	IsActive bool
	// Label is the widget that is shown under the icon in a horizontal stepper.
	Label widgets.Widget
	// StepStyle is the style of the step.
	StepStyle *StepStyle
}

// Step is a step of a Stepper.
//
// Flutter: material/stepper.dart Step
type Step struct {
	// Title is the title of the step.
	Title widgets.Widget
	// Content is the content of the step.
	Content widgets.Widget
	o       StepOpts
}

// NewStep creates a step.
//
// Flutter: material/stepper.dart Step.new
func NewStep(title, content widgets.Widget, opts ...StepOpts) Step {
	return Step{Title: title, Content: content, o: firstOpts(opts)}
}

// State is the state of the step.
//
// Flutter: material/stepper.dart Step.state
func (s Step) State() StepState { return s.o.State.effective() }

// IsActive is whether the step is active.
//
// Flutter: material/stepper.dart Step.isActive
func (s Step) IsActive() bool { return s.o.IsActive }

// StepperOpts are the optional parameters of NewStepper.
//
// Flutter: material/stepper.dart Stepper
type StepperOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller is the controller of the scroll of the steps.
	Controller widgets.ScrollController
	// Physics is the physics of the scroll of the steps.
	Physics widgets.ScrollPhysics
	// Type is how the steps are laid out; the default is vertical.
	Type StepperType
	// CurrentStep is the index of the step that is open; the default is 0.
	CurrentStep int
	// OnStepTapped is called with the index of the step that the user taps.
	OnStepTapped func(index int)
	// OnStepContinue is called when the user taps "Continue".
	OnStepContinue func()
	// OnStepCancel is called when the user taps "Cancel".
	OnStepCancel func()
	// ControlsBuilder builds the controls that replace "Continue" and "Cancel".
	ControlsBuilder ControlsWidgetBuilder
	// Elevation is the elevation of the header of a horizontal stepper; the
	// default is 2.
	Elevation foundation.Opt[float64]
	// Margin is not used by Flutter's Stepper either; it is kept for the API.
	Margin painting.EdgeInsetsGeometry
	// ConnectorColor is the color of the connector lines and of the circles, for
	// the selected (active) and the disabled steps.
	ConnectorColor widgets.WidgetStateProperty[ui.Color]
	// ConnectorThickness is the thickness of the connector lines.
	ConnectorThickness foundation.Opt[float64]
	// StepIconBuilder builds the icons of the headers.
	StepIconBuilder StepIconBuilder
	// StepIconHeight is the height of the icon, between 24 and 80.
	StepIconHeight foundation.Opt[float64]
	// StepIconWidth is the width of the icon, between 24 and 80; it must be equal
	// to the height.
	StepIconWidth foundation.Opt[float64]
	// StepIconMargin is the margin around the icon.
	StepIconMargin foundation.Opt[painting.EdgeInsets]
	// ClipBehavior is how the content of the steps is clipped.
	ClipBehavior ui.Clip
	// HeaderPadding is the padding of the headers; the default is 24 on the sides.
	HeaderPadding painting.EdgeInsetsGeometry
	// ContentPadding is the padding of the content of the steps.
	ContentPadding painting.EdgeInsetsGeometry
}

const (
	stepSize         = 24.0
	maxStepSize      = 80.0
	triangleSqrt     = 0.866025 // sqrt(3.0) / 2.0
	stepTriangleHigh = stepSize * triangleSqrt
)

// Stepper is a Material Design stepper: a series of steps that the user goes
// through, each with a header and a content that opens when it is current.
//
// Flutter: material/stepper.dart Stepper
type Stepper struct {
	widgets.StatefulWidget
	steps []Step
	o     StepperOpts
}

// NewStepper creates a stepper. In debug builds it panics with the tags
// StepperCurrentStepRange, StepperIconSizeRange and StepperIconSizeMismatch for
// the assertions of Flutter.
//
// Flutter: material/stepper.dart Stepper.new
func NewStepper(steps []Step, opts ...StepperOpts) *Stepper {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.CurrentStep < 0 || o.CurrentStep >= len(steps) {
			panic(errors.New(tagStepperCurrentStepRange, "the currentStep of a Stepper must be one of its steps", "currentStep", o.CurrentStep, "steps", len(steps)))
		}
		for _, v := range []foundation.Opt[float64]{o.StepIconHeight, o.StepIconWidth} {
			if x, ok := v.Get(); ok && (x < stepSize || x > maxStepSize) {
				panic(errors.New(tagStepperIconSizeRange, fmt.Sprintf("stepIconHeight and stepIconWidth must be greater than %v and less or equal to %v", stepSize, maxStepSize), "size", x))
			}
		}
		if o.StepIconHeight.IsSome() && o.StepIconWidth.IsSome() && o.StepIconHeight != o.StepIconWidth {
			panic(errors.New(tagStepperIconSizeMismatch, "If either stepIconHeight or stepIconWidth is specified, both must be specified and the values must be equal."))
		}
	}
	return &Stepper{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), steps: steps, o: o}
}

// CreateState creates the state.
//
// Flutter: material/stepper.dart Stepper.createState
func (w *Stepper) CreateState() widgets.State { return &stepperState{} }

var (
	stepTextStyle    = painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(12.0), Color: foundation.Some(ui.NewColor(0xFFFFFFFF))})
	stepErrorLight   = Colors.Red.Primary
	stepCircleLight  = ui.NewColor(0xFFFFFFFF)
	stepCircleDark   = Colors.Black87
	stepDisabledLite = Colors.Black38
	stepDisabledDark = Colors.White38
)

type stepperState struct {
	widgets.StateOf[*Stepper]
	widgets.TickerProvider
	keys      []*widgets.GlobalKey[widgets.State]
	oldStates map[int]StepState
}

func (s *stepperState) InitState() {
	w := s.Widget()
	s.oldStates = map[int]StepState{}
	s.keys = make([]*widgets.GlobalKey[widgets.State], len(w.steps))
	for i := range w.steps {
		s.keys[i] = widgets.NewGlobalKey[widgets.State]()
		s.oldStates[i] = w.steps[i].State()
	}
}

func (s *stepperState) DidUpdateWidget(old *Stepper) {
	if foundation.DebugMode && len(s.Widget().steps) != len(old.steps) {
		panic(errors.New(tagStepperStepCountChanged, "the number of steps of a Stepper cannot change", "old", len(old.steps), "new", len(s.Widget().steps)))
	}
	for i := range old.steps {
		s.oldStates[i] = old.steps[i].State()
	}
}

func (s *stepperState) errorColorDark() ui.Color { return Colors.Red.Shade400() }

func (s *stepperState) isDark() bool       { return ThemeBrightnessOf(s.Context()) == ui.BrightnessDark }
func (s *stepperState) isFirst(i int) bool { return i == 0 }
func (s *stepperState) isLast(i int) bool  { return len(s.Widget().steps)-1 == i }
func (s *stepperState) isCurrent(i int) bool {
	return s.Widget().o.CurrentStep == i
}
func (s *stepperState) isLabel() bool {
	for _, st := range s.Widget().steps {
		if st.o.Label != nil {
			return true
		}
	}
	return false
}
func (s *stepperState) stepStyle(i int) *StepStyle { return s.Widget().steps[i].o.StepStyle }
func (s *stepperState) headerPadding() painting.EdgeInsetsGeometry {
	return firstNonNil(s.Widget().o.HeaderPadding, painting.EdgeInsetsGeometry(painting.NewEdgeInsetsSymmetric(0, 24.0)))
}

func (s *stepperState) heightFactor() float64 {
	if s.isLabel() && s.Widget().o.StepIconHeight.IsSome() {
		return 2.5
	}
	return 2.0
}

func (s *stepperState) connectorColor(isActive bool) ui.Color {
	colorScheme := ThemeOf(s.Context()).ColorScheme
	if p := s.Widget().o.ConnectorColor; p != nil {
		states := widgets.NewWidgetStates(widgets.WidgetStateDisabled)
		if isActive {
			states = widgets.NewWidgetStates(widgets.WidgetStateSelected)
		}
		return p.Resolve(states)
	}
	if isActive {
		return colorScheme.Primary()
	}
	return Colors.Grey.Shade400()
}

func (s *stepperState) buildLine(visible, isActive bool) widgets.Widget {
	width := 0.0
	if visible {
		width = s.Widget().o.ConnectorThickness.OrElse(1.0)
	}
	return widgets.NewColoredBox(s.connectorColor(isActive), widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(width), Height: foundation.Some(16.0)}))
}

func (s *stepperState) buildCircleChild(index int, oldState bool) widgets.Widget {
	w := s.Widget()
	state := w.steps[index].State()
	if oldState {
		state = s.oldStates[index]
	}
	if f := w.o.StepIconBuilder; f != nil {
		if icon := f(index, state); icon != nil {
			return icon
		}
	}
	var textStyle *painting.TextStyle
	if st := s.stepStyle(index); st != nil {
		textStyle = st.IndexStyle
	}
	isDarkActive := s.isDark() && w.steps[index].IsActive()
	iconColor := stepCircleLight
	if isDarkActive {
		iconColor = stepCircleDark
	}
	if textStyle == nil {
		textStyle = stepTextStyle
		if isDarkActive {
			textStyle = stepTextStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(Colors.Black87)})
		}
	}
	switch state {
	case StepStateEditing:
		return widgets.NewIcon(Icons.Edit(), widgets.IconOpts{Color: foundation.Some(iconColor), Size: foundation.Some(18.0)})
	case StepStateComplete:
		return widgets.NewIcon(Icons.Check(), widgets.IconOpts{Color: foundation.Some(iconColor), Size: foundation.Some(18.0)})
	case StepStateError:
		return widgets.NewCenter(widgets.NewText("!", widgets.TextOpts{Style: stepTextStyle}))
	}
	return widgets.NewText(fmt.Sprint(index+1), widgets.TextOpts{Style: textStyle})
}

func (s *stepperState) circleColor(index int) ui.Color {
	w := s.Widget()
	isActive := w.steps[index].IsActive()
	colorScheme := ThemeOf(s.Context()).ColorScheme
	if p := w.o.ConnectorColor; p != nil {
		states := widgets.NewWidgetStates(widgets.WidgetStateDisabled)
		if isActive {
			states = widgets.NewWidgetStates(widgets.WidgetStateSelected)
		}
		return p.Resolve(states)
	}
	if !s.isDark() {
		if isActive {
			return colorScheme.Primary()
		}
		return colorScheme.OnSurface().WithOpacity(0.38)
	}
	if isActive {
		return colorScheme.Secondary()
	}
	return colorScheme.Background()
}

func (s *stepperState) iconMargin() painting.EdgeInsetsGeometry {
	if m, ok := s.Widget().o.StepIconMargin.Get(); ok {
		return m
	}
	return painting.NewEdgeInsetsSymmetric(8.0, 0)
}

func (s *stepperState) buildCircle(index int, oldState bool) widgets.Widget {
	w := s.Widget()
	st := s.stepStyle(index)
	decoration := painting.BoxDecorationOpts{Shape: painting.BoxShapeCircle}
	decoration.Color = foundation.Some(s.circleColor(index))
	if st != nil {
		decoration.Color = firstOpt(st.Color, decoration.Color)
		decoration.Border = st.Border
		if b, ok := st.BoxShadow.Get(); ok {
			decoration.BoxShadow = []painting.BoxShadow{b}
		}
		decoration.Gradient = st.Gradient
	}
	return widgets.NewPadding(s.iconMargin(), widgets.NewSizedBox(
		widgets.NewAnimatedContainer(ThemeAnimationDuration, widgets.NewCenter(s.buildCircleChild(index, oldState && w.steps[index].State() == StepStateError)),
			widgets.AnimatedContainerOpts{Curve: animation.Curves.FastOutSlowIn, Decoration: painting.NewBoxDecoration(decoration)}),
		widgets.SizedBoxOpts{Width: foundation.Some(w.o.StepIconWidth.OrElse(stepSize)), Height: foundation.Some(w.o.StepIconHeight.OrElse(stepSize))}))
}

func (s *stepperState) buildTriangle(index int, oldState bool) widgets.Widget {
	w := s.Widget()
	var color ui.Color
	if st := s.stepStyle(index); st != nil && st.ErrorColor.IsSome() {
		color = st.ErrorColor.Value()
	} else if s.isDark() {
		color = s.errorColorDark()
	} else {
		color = stepErrorLight
	}
	height := stepTriangleHigh
	if h, ok := w.o.StepIconHeight.Get(); ok {
		height = h * triangleSqrt
	}
	return widgets.NewPadding(s.iconMargin(), widgets.NewSizedBox(
		widgets.NewCenter(widgets.NewSizedBox(widgets.NewCustomPaint(
			widgets.NewAlign(s.buildCircleChild(index, oldState && w.steps[index].State() != StepStateError), widgets.AlignOpts{Alignment: painting.Alignment{X: 0.0, Y: 0.8}}), // 0.8 looks better than the geometrical 0.33.
			widgets.CustomPaintOpts{Painter: &stepTrianglePainter{color: color}}),
			widgets.SizedBoxOpts{Width: foundation.Some(w.o.StepIconWidth.OrElse(stepSize)), Height: foundation.Some(height)})),
		widgets.SizedBoxOpts{Width: foundation.Some(w.o.StepIconWidth.OrElse(stepSize)), Height: foundation.Some(w.o.StepIconHeight.OrElse(stepSize))}))
}

func (s *stepperState) buildIcon(index int) widgets.Widget {
	w := s.Widget()
	if w.steps[index].State() != s.oldStates[index] {
		state := widgets.CrossFadeStateShowFirst
		if w.steps[index].State() == StepStateError {
			state = widgets.CrossFadeStateShowSecond
		}
		return widgets.NewAnimatedCrossFade(s.buildCircle(index, true), s.buildTriangle(index, true), state, ThemeAnimationDuration, widgets.AnimatedCrossFadeOpts{
			FirstCurve: animation.NewInterval(0.0, 0.6, animation.Curves.FastOutSlowIn), SecondCurve: animation.NewInterval(0.4, 1.0, animation.Curves.FastOutSlowIn),
			SizeCurve: animation.Curves.FastOutSlowIn})
	}
	if w.steps[index].State() != StepStateError {
		return s.buildCircle(index, false)
	}
	return s.buildTriangle(index, false)
}

func (s *stepperState) buildVerticalControls(stepIndex int) widgets.Widget {
	w := s.Widget()
	if w.o.ControlsBuilder != nil {
		return w.o.ControlsBuilder(s.Context(), ControlsDetails{CurrentStep: w.o.CurrentStep, OnStepContinue: w.o.OnStepContinue, OnStepCancel: w.o.OnStepCancel, StepIndex: stepIndex})
	}
	cancelColor := Colors.Black54
	if ThemeBrightnessOf(s.Context()) == ui.BrightnessDark {
		cancelColor = Colors.White70
	}
	themeData := ThemeOf(s.Context())
	colorScheme := themeData.ColorScheme
	localizations := MaterialLocalizationsOf(s.Context())
	buttonShape := painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(2))})
	buttonPadding := painting.NewEdgeInsetsSymmetric(0, 16.0)
	label := func(text string) string {
		if themeData.UseMaterial3 {
			return text
		}
		return strings.ToUpper(text)
	}
	continueStyle := &ButtonStyle{
		ForegroundColor: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if states.Has(widgets.WidgetStateDisabled) {
				return foundation.None[ui.Color]()
			}
			if s.isDark() {
				return foundation.Some(colorScheme.OnSurface())
			}
			return foundation.Some(colorScheme.OnPrimary())
		}),
		BackgroundColor: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if s.isDark() || states.Has(widgets.WidgetStateDisabled) {
				return foundation.None[ui.Color]()
			}
			return foundation.Some(colorScheme.Primary())
		}),
		Padding: widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](buttonPadding),
		Shape:   widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](buttonShape),
	}
	return widgets.NewPadding(painting.NewEdgeInsetsFromLTRB(0, 16.0, 0, 0), widgets.NewSizedBox(widgets.NewRow([]widgets.Widget{
		NewTextButton(w.o.OnStepContinue, widgets.NewText(label(localizations.ContinueButtonLabel())), TextButtonOpts{Style: continueStyle}),
		widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(8.0, 0, 0, 0),
			NewTextButton(w.o.OnStepCancel, widgets.NewText(label(localizations.CancelButtonLabel())), TextButtonOpts{
				Style: TextButtonStyleFrom(ButtonStyleFromOpts{ForegroundColor: foundation.Some(cancelColor), Padding: buttonPadding, Shape: buttonShape})})),
	}), widgets.SizedBoxOpts{Height: foundation.Some(48.0)}))
}

// stepTextStyle is the style of the text of a step of the given kind: the title,
// the subtitle or the label. The disabled and error steps have their own color.
func (s *stepperState) stateTextStyle(index int, base *painting.TextStyle) *painting.TextStyle {
	switch s.Widget().steps[index].State() {
	case StepStateDisabled:
		c := stepDisabledLite
		if s.isDark() {
			c = stepDisabledDark
		}
		return base.CopyWith(painting.TextStyleCopy{Color: foundation.Some(c)})
	case StepStateError:
		c := stepErrorLight
		if s.isDark() {
			c = s.errorColorDark()
		}
		return base.CopyWith(painting.TextStyleCopy{Color: foundation.Some(c)})
	}
	return base
}

func (s *stepperState) buildHeaderText(index int) widgets.Widget {
	w := s.Widget()
	textTheme := ThemeOf(s.Context()).TextTheme
	children := []widgets.Widget{widgets.NewAnimatedDefaultTextStyle(s.stateTextStyle(index, textTheme.BodyLarge), ThemeAnimationDuration, w.steps[index].Title,
		widgets.AnimatedDefaultTextStyleOpts{Curve: animation.Curves.FastOutSlowIn})}
	if sub := w.steps[index].o.Subtitle; sub != nil {
		children = append(children, widgets.NewPadding(painting.NewEdgeInsetsFromLTRB(0, 2.0, 0, 0), widgets.NewAnimatedDefaultTextStyle(s.stateTextStyle(index, textTheme.BodySmall),
			ThemeAnimationDuration, sub, widgets.AnimatedDefaultTextStyleOpts{Curve: animation.Curves.FastOutSlowIn})))
	}
	return widgets.NewColumn(children, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart, MainAxisSize: rendering.MainAxisSizeMin})
}

func (s *stepperState) buildLabelText(index int) widgets.Widget {
	w := s.Widget()
	if label := w.steps[index].o.Label; label != nil {
		return widgets.NewAnimatedDefaultTextStyle(s.stateTextStyle(index, ThemeOf(s.Context()).TextTheme.BodyLarge), ThemeAnimationDuration, label)
	}
	return widgets.NewSizedBoxShrink(nil)
}

func (s *stepperState) buildVerticalHeader(index int) widgets.Widget {
	w := s.Widget()
	isActive := w.steps[index].IsActive()
	isPreviousActive := index > 0 && w.steps[index-1].IsActive()
	return widgets.NewPadding(s.headerPadding(), widgets.NewRow([]widgets.Widget{
		widgets.NewColumn([]widgets.Widget{s.buildLine(!s.isFirst(index), isPreviousActive), s.buildIcon(index), s.buildLine(!s.isLast(index), isActive)}),
		widgets.NewExpanded(widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(12.0, 0, 0, 0), s.buildHeaderText(index))),
	}))
}

func (s *stepperState) buildVerticalBody(index int) widgets.Widget {
	w := s.Widget()
	var marginLeft, marginRight foundation.Opt[float64]
	if m, ok := w.o.StepIconMargin.Get(); ok {
		marginLeft, marginRight = foundation.Some(m.Left), foundation.Some(m.Right)
	}
	additionalLeft, additionalRight := 0.0, 0.0
	if v, ok := marginLeft.Get(); ok {
		additionalLeft = v / 2.0
	}
	if v, ok := marginRight.Get(); ok {
		additionalRight = v / 2.0
	}
	contentPadding := firstNonNil(w.o.ContentPadding, painting.EdgeInsetsGeometry(painting.NewEdgeInsetsDirectionalFromSTEB(60.0, 0, 24.0, 24.0)))
	effectivePadding := contentPadding.Add(painting.NewEdgeInsetsDirectionalFromSTEB(marginLeft.OrElse(0.0), 0, 0, 0))
	connector := 0.0
	if !s.isLast(index) {
		connector = w.o.ConnectorThickness.OrElse(1.0)
	}
	state := widgets.CrossFadeStateShowFirst
	if s.isCurrent(index) {
		state = widgets.CrossFadeStateShowSecond
	}
	return widgets.NewStack([]widgets.Widget{
		widgets.NewPositionedDirectional(widgets.NewCenter(widgets.NewSizedBox(widgets.NewColoredBox(s.connectorColor(w.steps[index].IsActive()), nil),
			widgets.SizedBoxOpts{Width: foundation.Some(connector), Height: foundation.Some(math.Inf(1))})),
			widgets.PositionedDirectionalOpts{Start: foundation.Some(24.0 + additionalLeft + additionalRight), Top: foundation.Some(0.0), Bottom: foundation.Some(0.0),
				Width: foundation.Some(w.o.StepIconWidth.OrElse(stepSize))}),
		widgets.NewAnimatedCrossFade(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(math.Inf(1)), Height: foundation.Some(0.0)}),
			widgets.NewPadding(effectivePadding, widgets.NewColumn([]widgets.Widget{
				widgets.NewClipRect(w.steps[index].Content, widgets.ClipRectOpts{ClipBehavior: w.o.ClipBehavior}),
				s.buildVerticalControls(index),
			})), state, ThemeAnimationDuration, widgets.AnimatedCrossFadeOpts{
				FirstCurve: animation.NewInterval(0.0, 0.6, animation.Curves.FastOutSlowIn), SecondCurve: animation.NewInterval(0.4, 1.0, animation.Curves.FastOutSlowIn),
				SizeCurve: animation.Curves.FastOutSlowIn}),
	})
}

func (s *stepperState) buildVertical() widgets.Widget {
	w := s.Widget()
	children := make([]widgets.Widget, len(w.steps))
	for i := range w.steps {
		i := i
		var onTap func()
		if w.steps[i].State() != StepStateDisabled {
			onTap = func() {
				if bc := s.keys[i].CurrentContext(); bc != nil {
					widgets.ScrollableEnsureVisible(bc, widgets.EnsureVisibleOpts{Curve: animation.Curves.FastOutSlowIn, Duration: ThemeAnimationDuration})
				}
				if w.o.OnStepTapped != nil {
					w.o.OnStepTapped(i)
				}
			}
		}
		children[i] = widgets.NewColumn([]widgets.Widget{
			NewInkWell(s.buildVerticalHeader(i), InkWellOpts{OnTap: onTap, CanRequestFocus: foundation.Some(w.steps[i].State() != StepStateDisabled)}),
			s.buildVerticalBody(i),
		}, widgets.ColumnOpts{Key: s.keys[i]})
	}
	return widgets.NewListView(children, widgets.ListViewOpts{Controller: w.o.Controller, ShrinkWrap: true, Physics: w.o.Physics})
}

func (s *stepperState) buildHorizontal() widgets.Widget {
	w := s.Widget()
	contentPadding := firstNonNil(w.o.ContentPadding, painting.EdgeInsetsGeometry(painting.NewEdgeInsetsAll(24.0)))
	var children []widgets.Widget
	for i := range w.steps {
		i := i
		var onTap func()
		if w.steps[i].State() != StepStateDisabled {
			onTap = func() {
				if w.o.OnStepTapped != nil {
					w.o.OnStepTapped(i)
				}
			}
		}
		height := 72.0
		if s.isLabel() {
			height = 104.0
		}
		iconColumn := []widgets.Widget{}
		if w.steps[i].o.Label != nil {
			iconColumn = append(iconColumn, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(24.0)}))
		}
		iconColumn = append(iconColumn, widgets.NewCenter(s.buildIcon(i)))
		if w.steps[i].o.Label != nil {
			iconColumn = append(iconColumn, widgets.NewSizedBox(s.buildLabelText(i), widgets.SizedBoxOpts{Height: foundation.Some(24.0)}))
		}
		var headerPadding painting.EdgeInsetsGeometry = painting.NewEdgeInsetsDirectionalFromSTEB(12.0, 0, 0, 0)
		if m, ok := w.o.StepIconMargin.Get(); ok {
			headerPadding = m
		}
		children = append(children, NewInkResponse(widgets.NewRow([]widgets.Widget{
			widgets.NewSizedBox(widgets.NewColumn(iconColumn, widgets.ColumnOpts{MainAxisAlignment: rendering.MainAxisAlignmentCenter}), widgets.SizedBoxOpts{Height: foundation.Some(height)}),
			widgets.NewPadding(headerPadding, s.buildHeaderText(i)),
		}), InkResponseOpts{OnTap: onTap, CanRequestFocus: foundation.Some(w.steps[i].State() != StepStateDisabled)}))
		if !s.isLast(i) {
			var thickness float64
			var color ui.Color
			if st := w.steps[i].o.StepStyle; st != nil {
				thickness = firstOpt(st.ConnectorThickness, w.o.ConnectorThickness, foundation.Some(1.0)).Value()
				color = st.ConnectorColor.OrElse(s.connectorColor(w.steps[i].IsActive()))
			} else {
				thickness = w.o.ConnectorThickness.OrElse(1.0)
				color = s.connectorColor(w.steps[i].IsActive())
			}
			var margin painting.EdgeInsetsGeometry = painting.NewEdgeInsetsSymmetric(0, 8.0)
			if m, ok := w.o.StepIconMargin.Get(); ok {
				margin = m
			}
			children = append(children, widgets.NewExpanded(widgets.NewPadding(margin,
				widgets.NewSizedBox(widgets.NewColoredBox(color, nil), widgets.SizedBoxOpts{Height: foundation.Some(thickness)}))))
		}
	}
	panels := make([]widgets.Widget, len(w.steps))
	for i := range w.steps {
		panels[i] = widgets.NewVisibility(widgets.NewClipRect(w.steps[i].Content, widgets.ClipRectOpts{ClipBehavior: w.o.ClipBehavior}),
			widgets.VisibilityOpts{MaintainState: true, Visible: foundation.Some(i == w.o.CurrentStep)})
	}
	headerHeight := foundation.None[float64]()
	if h, ok := w.o.StepIconHeight.Get(); ok {
		headerHeight = foundation.Some(h * s.heightFactor())
	}
	return widgets.NewColumn([]widgets.Widget{
		NewMaterial(widgets.NewPadding(s.headerPadding(), widgets.NewSizedBox(widgets.NewRow(children), widgets.SizedBoxOpts{Height: headerHeight})),
			MaterialOpts{Elevation: w.o.Elevation.OrElse(2)}),
		widgets.NewExpanded(widgets.NewListView([]widgets.Widget{
			widgets.NewAnimatedSize(ThemeAnimationDuration, widgets.NewColumn(panels, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}),
				widgets.AnimatedSizeOpts{Curve: animation.Curves.FastOutSlowIn}),
			s.buildVerticalControls(w.o.CurrentStep),
		}, widgets.ListViewOpts{Controller: w.o.Controller, Physics: w.o.Physics, Padding: contentPadding})),
	})
}

func (s *stepperState) Build(bc widgets.BuildContext) widgets.Widget {
	if foundation.DebugMode {
		if _, ok := widgets.FindAncestorWidgetOfExactType[*Stepper](bc); ok {
			panic(errors.New(tagStepperNested, "Steppers must not be nested. The material specification advises that one should avoid embedding steppers within steppers. "+
				"https://material.io/archive/guidelines/components/steppers.html#steppers-usage"))
		}
	}
	if s.Widget().o.Type == StepperTypeHorizontal {
		return s.buildHorizontal()
	}
	return s.buildVertical()
}

// stepTrianglePainter paints the triangle of a step in the error state.
//
// Flutter: material/stepper.dart _TrianglePainter
type stepTrianglePainter struct {
	rendering.CustomPainterBase
	color ui.Color
}

func (p *stepTrianglePainter) HitTest(ui.Offset) foundation.Opt[bool] { return foundation.Some(true) }

func (p *stepTrianglePainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*stepTrianglePainter)
	return !ok || o.color != p.color
}

func (p *stepTrianglePainter) Paint(canvas *ui.Canvas, size ui.Size) {
	base, halfBase, height := size.Width, size.Width/2.0, size.Height
	path := ui.NewPath()
	path.AddPolygon([]ui.Offset{{Dx: 0.0, Dy: height}, {Dx: base, Dy: height}, {Dx: halfBase, Dy: 0.0}}, true)
	paint := ui.NewPaint()
	paint.SetColor(p.color)
	canvas.DrawPath(path, paint)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagStepperCurrentStepRange = "StepperCurrentStepRange"
	tagStepperIconSizeMismatch = "StepperIconSizeMismatch"
	tagStepperIconSizeRange    = "StepperIconSizeRange"
	tagStepperNested           = "StepperNested"
	tagStepperStepCountChanged = "StepperStepCountChanged"
)
