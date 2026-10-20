// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"strconv"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// inputTransitionDuration is the duration of the transitions of the field,
	// extracted from the Android TextInputLayout.
	//
	// Flutter: material/input_decorator.dart _kTransitionDuration
	inputTransitionDuration = 167 * time.Millisecond
	// inputHintFadeTransitionDuration is the default duration of the fade of the
	// hint. Animating the hint is not mentioned in the Material specification; the
	// animation is kept for backward compatibility and a short duration is used to
	// mitigate the UX impact.
	//
	// Flutter: material/input_decorator.dart _kHintFadeTransitionDuration
	inputHintFadeTransitionDuration = 20 * time.Millisecond
	// inputBorderHoverDuration is the duration of the fade of the hover color.
	//
	// Flutter: material/input_decorator.dart _BorderContainerState._kHoverDuration
	inputBorderHoverDuration = 15 * time.Millisecond
)

// inputTransitionCurve is the curve of the transitions of the field.
//
// Flutter: material/input_decorator.dart _kTransitionCurve
var inputTransitionCurve animation.Curve = animation.Curves.FastOutSlowIn

// The semantics tags of the affixes.
//
// Flutter: material/input_decorator.dart _InputDecoratorState._kPrefixSemanticsTag
var (
	decoratorPrefixSemanticsTag     = semantics.NewSemanticsTag("_InputDecoratorState.prefix")
	decoratorPrefixIconSemanticsTag = semantics.NewSemanticsTag("_InputDecoratorState.prefixIcon")
	decoratorSuffixSemanticsTag     = semantics.NewSemanticsTag("_InputDecoratorState.suffix")
	decoratorSuffixIconSemanticsTag = semantics.NewSemanticsTag("_InputDecoratorState.suffixIcon")
)

// inputBorderTween interpolates between two input borders.
//
// Flutter: material/input_decorator.dart _InputBorderTween
type inputBorderTween struct{ begin, end InputBorder }

func (t inputBorderTween) evaluate(a animation.Animation[float64]) InputBorder {
	v := a.Value()
	if v == 0 {
		return t.begin
	}
	if v == 1 {
		return t.end
	}
	if b, ok := painting.ShapeBorderLerp(t.begin, t.end, v).(InputBorder); ok {
		return b
	}
	return t.end
}

// inputBorderPainter passes the gap parameters along to the paint method of an
// InputBorder.
//
// Flutter: material/input_decorator.dart _InputBorderPainter
type inputBorderPainter struct {
	rendering.CustomPainterBase
	borderAnimation animation.Animation[float64]
	border          inputBorderTween
	gapAnimation    animation.Animation[float64]
	gap             *inputBorderGap
	textDirection   ui.TextDirection
	fillColor       ui.Color
	hoverBegin      ui.Color
	hoverEnd        ui.Color
	hoverAnimation  animation.Animation[float64]
}

func (p *inputBorderPainter) blendedColor() ui.Color {
	hover := ui.ColorLerp(&p.hoverBegin, &p.hoverEnd, p.hoverAnimation.Value())
	return ui.ColorAlphaBlend(*hover, p.fillColor)
}

// Paint fills the container and draws the border, with the gap of the label.
//
// Flutter: material/input_decorator.dart _InputBorderPainter.paint
func (p *inputBorderPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	borderValue := p.border.evaluate(p.borderAnimation)
	canvasRect := ui.NewRectFromLTWH(0, 0, size.Width, size.Height)
	blendedFillColor := p.blendedColor()
	if blendedFillColor.Alpha() > 0 {
		paint := ui.NewPaint()
		paint.SetColor(blendedFillColor)
		paint.SetStyle(ui.PaintingStyleFill)
		if borderValue.PreferPaintInterior() {
			borderValue.PaintInterior(canvas, canvasRect, paint, p.textDirection)
		} else {
			canvas.DrawPath(borderValue.GetOuterPath(canvasRect, p.textDirection), paint)
		}
	}
	borderValue.PaintGap(canvas, canvasRect, InputBorderGap{Start: p.gap.start, Extent: p.gap.extent, Percentage: p.gapAnimation.Value()}, p.textDirection)
}

// ShouldRepaint is true when anything that is painted changed.
//
// Flutter: material/input_decorator.dart _InputBorderPainter.shouldRepaint
func (p *inputBorderPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o := old.(*inputBorderPainter)
	return p.borderAnimation != o.borderAnimation || p.hoverAnimation != o.hoverAnimation || p.gapAnimation != o.gapAnimation ||
		!foundation.ValuesEqual(p.border, o.border) || p.gap != o.gap && (p.gap.start != o.gap.start || p.gap.extent != o.gap.extent) ||
		p.textDirection != o.textDirection
}

// HitTest defers to the default.
func (p *inputBorderPainter) HitTest(ui.Offset) foundation.Opt[bool] { return foundation.None[bool]() }

// borderContainer is an analog of AnimatedContainer, which can animate its
// shaped border. This specialized animated container is needed because the gap,
// which is computed at layout time, is required by the border's paint method.
//
// Flutter: material/input_decorator.dart _BorderContainer
type borderContainer struct {
	widgets.StatefulWidget
	border       InputBorder
	gap          *inputBorderGap
	gapAnimation animation.Animation[float64]
	fillColor    ui.Color
	hoverColor   ui.Color
	isHovering   bool
}

func (w *borderContainer) CreateState() widgets.State { return &borderContainerState{} }

type borderContainerState struct {
	widgets.StateOf[*borderContainer]
	widgets.TickerProvider
	controller           *animation.AnimationController
	hoverColorController *animation.AnimationController
	borderAnimation      *animation.CurvedAnimation
	border               inputBorderTween
	hoverAnimation       *animation.CurvedAnimation
	hoverBegin           ui.Color
	hoverEnd             ui.Color
}

// InitState creates the animations of the border and of the hover color.
//
// Flutter: material/input_decorator.dart _BorderContainerState.initState
func (s *borderContainerState) InitState() {
	w := s.Widget()
	hoverValue := 0.0
	if w.isHovering {
		hoverValue = 1.0
	}
	s.hoverColorController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(inputBorderHoverDuration), Value: foundation.Some(hoverValue)})
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(inputTransitionDuration)})
	s.borderAnimation = animation.NewCurvedAnimation(s.controller, inputTransitionCurve, inputTransitionCurve.Flipped())
	s.border = inputBorderTween{begin: w.border, end: w.border}
	s.hoverAnimation = animation.NewCurvedAnimation(s.hoverColorController, animation.Curves.Linear)
	// Animate between transparent hoverColor and hoverColor.
	s.hoverBegin, s.hoverEnd = w.hoverColor.WithAlpha(0), w.hoverColor
}

// Dispose releases the animations.
//
// Flutter: material/input_decorator.dart _BorderContainerState.dispose
func (s *borderContainerState) Dispose() {
	s.controller.Dispose()
	s.hoverColorController.Dispose()
	s.borderAnimation.Dispose()
	s.hoverAnimation.Dispose()
}

// DidUpdateWidget animates to the new border and hover color.
//
// Flutter: material/input_decorator.dart _BorderContainerState.didUpdateWidget
func (s *borderContainerState) DidUpdateWidget(old *borderContainer) {
	w := s.Widget()
	if !foundation.ValuesEqual(w.border, old.border) {
		s.border = inputBorderTween{begin: old.border, end: w.border}
		s.controller.SetValue(0.0)
		s.controller.Forward()
	}
	if w.hoverColor != old.hoverColor {
		s.hoverBegin, s.hoverEnd = w.hoverColor.WithAlpha(0), w.hoverColor
	}
	if w.isHovering != old.isHovering {
		if w.isHovering {
			s.hoverColorController.Forward()
		} else {
			s.hoverColorController.Reverse()
		}
	}
}

// Build paints the border in front of the (empty) child.
//
// Flutter: material/input_decorator.dart _BorderContainerState.build
func (s *borderContainerState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	painter := &inputBorderPainter{
		CustomPainterBase: rendering.NewCustomPainterBase(foundation.MergeListenables(s.borderAnimation, w.gap, s.hoverColorController)),
		borderAnimation:   s.borderAnimation,
		border:            s.border,
		gapAnimation:      w.gapAnimation,
		gap:               w.gap,
		textDirection:     widgets.DirectionalityOf(bc),
		fillColor:         w.fillColor,
		hoverBegin:        s.hoverBegin,
		hoverEnd:          s.hoverEnd,
		hoverAnimation:    s.hoverAnimation,
	}
	return widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{ForegroundPainter: painter})
}

// helperError displays the helper and error text. When the error text appears it
// fades and the helper text fades out. The error text also slides upwards a
// little when it first appears.
//
// Flutter: material/input_decorator.dart _HelperError
type helperError struct {
	widgets.StatefulWidget
	textAlign         ui.TextAlign
	helper            widgets.Widget
	helperText        foundation.Opt[string]
	helperStyle       *painting.TextStyle
	helperMaxLines    foundation.Opt[int]
	err               widgets.Widget
	errorText         foundation.Opt[string]
	errorStyle        *painting.TextStyle
	errorMaxLines     foundation.Opt[int]
	semanticsAnnounce bool
}

func (w *helperError) CreateState() widgets.State { return &helperErrorState{} }

type helperErrorState struct {
	widgets.StateOf[*helperError]
	widgets.SingleTickerProvider
	controller *animation.AnimationController
	helper     widgets.Widget
	err        widgets.Widget
	handle     foundation.ListenerHandle
}

func (s *helperErrorState) hasHelper() bool {
	w := s.Widget()
	return w.helperText.IsSome() || w.helper != nil
}

func (s *helperErrorState) hasError() bool {
	w := s.Widget()
	return w.errorText.IsSome() || w.err != nil
}

// InitState creates the controller of the fade between the helper and the error.
//
// Flutter: material/input_decorator.dart _HelperErrorState.initState
func (s *helperErrorState) InitState() {
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(inputTransitionDuration)})
	if s.hasError() {
		s.err = s.buildError()
		s.controller.SetValue(1.0)
	} else if s.hasHelper() {
		s.helper = s.buildHelper()
	}
	s.handle = s.controller.AddListener(func() { s.SetState(func() {}) })
}

// Dispose releases the controller.
//
// Flutter: material/input_decorator.dart _HelperErrorState.dispose
func (s *helperErrorState) Dispose() { s.controller.Dispose() }

// DidUpdateWidget fades to the error, or back to the helper, when one of them
// appears or goes.
//
// Flutter: material/input_decorator.dart _HelperErrorState.didUpdateWidget
func (s *helperErrorState) DidUpdateWidget(old *helperError) {
	w := s.Widget()
	errorStateChanged := (w.err != nil) != (old.err != nil)
	errorTextStateChanged := w.errorText.IsSome() != old.errorText.IsSome()
	helperStateChanged := (w.helper != nil) != (old.helper != nil)
	helperTextStateChanged := w.errorText.IsNone() && w.helperText.IsSome() != old.helperText.IsSome()
	if errorStateChanged || errorTextStateChanged || helperStateChanged || helperTextStateChanged {
		switch {
		case w.err != nil || w.errorText.IsSome():
			s.err = s.buildError()
			s.controller.Forward()
		case w.helper != nil || w.helperText.IsSome():
			s.helper = s.buildHelper()
			s.controller.Reverse()
		default:
			s.controller.Reverse()
		}
	}
}

func (s *helperErrorState) buildHelper() widgets.Widget {
	w := s.Widget()
	child := w.helper
	if child == nil {
		child = widgets.NewText(w.helperText.Value(), widgets.TextOpts{
			Style: w.helperStyle, TextAlign: w.textAlign, Overflow: painting.TextOverflowEllipsis, MaxLines: w.helperMaxLines})
	}
	return widgets.NewSemantics(widgets.NewFadeTransition(
		animation.NewTween(1.0, 0.0).Animate(s.controller), child), widgets.SemanticsOpts{Container: true})
}

func (s *helperErrorState) buildError() widgets.Widget {
	w := s.Widget()
	capturedErrorText := w.errorText
	capturedError := w.err
	if capturedError != nil && w.errorStyle != nil {
		capturedError = widgets.NewDefaultTextStyle(w.errorStyle, capturedError)
	}
	controller := s.controller
	return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		child := capturedError
		if child == nil {
			child = widgets.NewText(capturedErrorText.Value(), widgets.TextOpts{
				Style: w.errorStyle, TextAlign: w.textAlign, Overflow: painting.TextOverflowEllipsis, MaxLines: w.errorMaxLines})
		}
		translation := animation.NewTween(ui.Offset{Dx: 0.0, Dy: -0.25}, ui.Offset{}).Evaluate(controller.View())
		return widgets.NewSemantics(widgets.NewFadeTransition(controller,
			widgets.NewFractionalTranslation(translation, child)),
			widgets.SemanticsOpts{Container: true, LiveRegion: foundation.Some(w.semanticsAnnounce)})
	})
}

// Build shows the helper, the error, or the two fading into each other.
//
// Flutter: material/input_decorator.dart _HelperErrorState.build
func (s *helperErrorState) Build(widgets.BuildContext) widgets.Widget {
	empty := widgets.Widget(widgets.NewSizedBox(nil))
	if s.controller.Status() == animation.AnimationStatusDismissed {
		s.err = nil
		if s.hasHelper() {
			s.helper = s.buildHelper()
			return s.helper
		}
		s.helper = nil
		return empty
	}
	if s.controller.Status() == animation.AnimationStatusCompleted {
		s.helper = nil
		if s.hasError() {
			s.err = s.buildError()
			return s.err
		}
		s.err = nil
		return empty
	}
	if s.helper == nil && s.hasError() {
		return s.buildError()
	}
	if s.err == nil && s.hasHelper() {
		return s.buildHelper()
	}
	if s.hasError() {
		return widgets.NewStack([]widgets.Widget{
			widgets.NewFadeTransition(animation.NewTween(1.0, 0.0).Animate(s.controller), s.helper),
			s.buildError(),
		})
	}
	if s.hasHelper() {
		return widgets.NewStack([]widgets.Widget{
			s.buildHelper(),
			widgets.NewFadeTransition(s.controller, s.err),
		})
	}
	return empty
}

// decorator is the slotted widget that creates the render object of a decorator
// from its parts.
//
// Flutter: material/input_decorator.dart _Decorator
type decorator struct {
	widgets.SlottedMultiChildRenderObjectWidget
	textAlignVertical foundation.Opt[painting.TextAlignVertical]
	decoration        decoration
	textDirection     ui.TextDirection
	textBaseline      ui.TextBaseline
	isFocused         bool
	expands           bool
}

// Slots are the slots of the decorator.
func (w *decorator) Slots() []any { return allDecorationSlots }

// ChildForSlot is the part of the decoration in the slot.
//
// Flutter: material/input_decorator.dart _Decorator.childForSlot
func (w *decorator) ChildForSlot(slot any) widgets.Widget {
	d := w.decoration
	switch slot.(decorationSlot) {
	case slotIcon:
		return d.icon
	case slotInput:
		return d.input
	case slotLabel:
		return d.label
	case slotHint:
		return d.hint
	case slotPrefix:
		return d.prefix
	case slotSuffix:
		return d.suffix
	case slotPrefixIcon:
		return d.prefixIcon
	case slotSuffixIcon:
		return d.suffixIcon
	case slotHelperError:
		return d.helperError
	case slotCounter:
		return d.counter
	case slotContainer:
		return d.container
	}
	return nil
}

// CreateRenderObject creates the render object.
//
// Flutter: material/input_decorator.dart _Decorator.createRenderObject
func (w *decorator) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	return newRenderDecoration(w.decoration, w.textDirection, w.textBaseline, w.isFocused, w.expands, ThemeOf(bc).UseMaterial3, w.textAlignVertical)
}

// UpdateRenderObject copies the properties to the render object.
//
// Flutter: material/input_decorator.dart _Decorator.updateRenderObject
func (w *decorator) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderDecoration)
	r.setDecoration(w.decoration)
	r.setExpands(w.expands)
	r.setIsFocused(w.isFocused)
	r.setTextAlignVertical(w.textAlignVertical)
	r.setTextBaseline(w.textBaseline)
	r.setTextDirection(w.textDirection)
}

// affixText is the text or widget of a prefix or a suffix: it is only seen when
// the label floats.
//
// Flutter: material/input_decorator.dart _AffixText
type affixText struct {
	widgets.StatelessWidget
	labelIsFloating  bool
	text             foundation.Opt[string]
	style            *painting.TextStyle
	child            widgets.Widget
	semanticsSortKey semantics.SemanticsSortKey
	semanticsTag     semantics.SemanticsTag
}

// Build builds the faded text.
//
// Flutter: material/input_decorator.dart _AffixText.build
func (w *affixText) Build(widgets.BuildContext) widgets.Widget {
	child := w.child
	if child == nil {
		if text, ok := w.text.Get(); ok {
			child = widgets.NewText(text, widgets.TextOpts{Style: w.style})
		}
	}
	opacity := 0.0
	if w.labelIsFloating {
		opacity = 1.0
	}
	return widgets.NewDefaultTextStyleMerge(
		widgets.NewIgnorePointer(
			widgets.NewAnimatedOpacity(opacity, inputTransitionDuration,
				widgets.NewSemantics(child, widgets.SemanticsOpts{SortKey: w.semanticsSortKey, TagForChildren: foundation.Some(w.semanticsTag)}),
				widgets.AnimatedOpacityOpts{Curve: inputTransitionCurve}),
			widgets.IgnorePointerOpts{Ignoring: foundation.Some(!w.labelIsFloating)}),
		widgets.DefaultTextStyleMergeOpts{Style: w.style})
}

// InputDecorator is the widget that draws the decoration of a text field: its
// border, label, icons and the text around the input, laid out in the container
// around the child (the input).
//
// Flutter: material/input_decorator.dart InputDecorator
type InputDecorator struct {
	widgets.StatefulWidget
	decoration InputDecoration
	o          InputDecoratorOpts
}

// InputDecoratorOpts are the optional parameters of NewInputDecorator.
//
// Flutter: material/input_decorator.dart InputDecorator.new
type InputDecoratorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BaseStyle is the text style of the input: the label, the hint and the
	// affixes are based on it.
	BaseStyle *painting.TextStyle
	// TextAlign is how the text of the decoration is aligned horizontally.
	TextAlign ui.TextAlign
	// TextAlignVertical is how the text is aligned vertically in the container.
	TextAlignVertical foundation.Opt[painting.TextAlignVertical]
	// IsFocused says whether the input has the focus.
	IsFocused bool
	// IsHovering says whether the pointer is over the input.
	IsHovering bool
	// Expands says whether the input fills the height it is given.
	Expands bool
	// IsEmpty says whether the input has no text.
	IsEmpty bool
	// Child is the input.
	Child widgets.Widget
}

// NewInputDecorator creates the decorator of an input.
//
// Flutter: material/input_decorator.dart InputDecorator.new
func NewInputDecorator(decoration InputDecoration, opts ...InputDecoratorOpts) *InputDecorator {
	o := firstOpts(opts)
	return &InputDecorator{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), decoration: decoration, o: o}
}

// Decoration is the decoration of the input.
//
// Flutter: material/input_decorator.dart InputDecorator.decoration
func (w *InputDecorator) Decoration() InputDecoration { return w.decoration }

// IsFocused says whether the input has the focus.
//
// Flutter: material/input_decorator.dart InputDecorator.isFocused
func (w *InputDecorator) IsFocused() bool { return w.o.IsFocused }

// IsEmpty says whether the input is empty.
//
// Flutter: material/input_decorator.dart InputDecorator.isEmpty
func (w *InputDecorator) IsEmpty() bool { return w.o.IsEmpty }

// Child is the input.
//
// Flutter: material/input_decorator.dart InputDecorator.child
func (w *InputDecorator) Child() widgets.Widget { return w.o.Child }

func (w *InputDecorator) labelShouldWithdraw() bool {
	return !w.o.IsEmpty || (w.o.IsFocused && !w.decoration.Disabled)
}

// CreateState creates the state.
//
// Flutter: material/input_decorator.dart InputDecorator.createState
func (w *InputDecorator) CreateState() widgets.State { return &inputDecoratorState{} }

// InputDecoratorContainerOf is the box of the container of the closest decorator
// above the context, or nil.
//
// Flutter: material/input_decorator.dart InputDecorator.containerOf
func InputDecoratorContainerOf(bc widgets.BuildContext) rendering.Box {
	r, ok := widgets.FindAncestorRenderObjectOfType[*renderDecoration](bc)
	if !ok {
		return nil
	}
	return r.container()
}

// DebugFillProperties adds the decoration and the state of the input.
//
// Flutter: material/input_decorator.dart InputDecorator.debugFillProperties
func (w *InputDecorator) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("decoration", w.decoration))
	p.Add(objectProp("baseStyle", propOrNilStyle(w.o.BaseStyle)))
	p.Add(foundation.NewDiagnosticsProperty("isFocused", w.o.IsFocused))
	p.Add(foundation.NewDiagnosticsProperty("expands", w.o.Expands, foundation.DiagnosticsPropertyOpts{DefaultValue: false}))
	p.Add(foundation.NewDiagnosticsProperty("isEmpty", w.o.IsEmpty))
}

func propOrNilStyle(s *painting.TextStyle) any {
	if s == nil {
		return nil
	}
	return s
}

type inputDecoratorState struct {
	widgets.StateOf[*InputDecorator]
	widgets.TickerProvider
	floatingLabelController *animation.AnimationController
	floatingLabelAnimation  *animation.CurvedAnimation
	shakingLabelController  *animation.AnimationController
	borderGap               *inputBorderGap
	// A unique name avoids mixing up the sort order with sibling decorators.
	sortName        string
	effective       *InputDecoration
	curvedAnimation *animation.CurvedAnimation
	curvedParent    animation.Animation[float64]
	handle          foundation.ListenerHandle
}

var decoratorSerial int

// InitState creates the animations of the floating label.
//
// Flutter: material/input_decorator.dart _InputDecoratorState.initState
func (s *inputDecoratorState) InitState() {
	s.borderGap = newInputBorderGap()
	decoratorSerial++
	s.sortName = strconv.Itoa(decoratorSerial)
	s.floatingLabelController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(inputTransitionDuration)})
	s.handle = s.floatingLabelController.AddListener(func() { s.SetState(func() {}) })
	s.floatingLabelAnimation = animation.NewCurvedAnimation(s.floatingLabelController, inputTransitionCurve, inputTransitionCurve.Flipped())
	s.shakingLabelController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(inputTransitionDuration)})
}

// DidChangeDependencies resets the decoration, and the floating label to where
// it should be.
//
// Flutter: material/input_decorator.dart _InputDecoratorState.didChangeDependencies
func (s *inputDecoratorState) DidChangeDependencies() {
	s.effective = nil
	dec := s.decoration()
	labelIsInitiallyFloating := dec.FloatingLabelBehavior.orAuto() != FloatingLabelBehaviorNever && s.labelShouldWithdraw()
	if labelIsInitiallyFloating {
		s.floatingLabelController.SetValue(1.0)
	} else {
		s.floatingLabelController.SetValue(0.0)
	}
}

// Dispose releases the animations.
//
// Flutter: material/input_decorator.dart _InputDecoratorState.dispose
func (s *inputDecoratorState) Dispose() {
	s.floatingLabelController.Dispose()
	s.floatingLabelAnimation.Dispose()
	s.shakingLabelController.Dispose()
	s.borderGap.Dispose()
	if s.curvedAnimation != nil {
		s.curvedAnimation.Dispose()
	}
}

// decoration is the decoration of the widget with the defaults of the theme.
//
// Flutter: material/input_decorator.dart _InputDecoratorState.decoration
func (s *inputDecoratorState) decoration() InputDecoration {
	if s.effective == nil {
		d := s.Widget().decoration.ApplyDefaults(InputDecorationThemeOf(s.Context()))
		s.effective = &d
	}
	return *s.effective
}

func (s *inputDecoratorState) hasError() bool {
	d := s.decoration()
	return d.ErrorText.IsSome() || d.Error != nil
}

func (s *inputDecoratorState) isFocused() bool { return s.Widget().o.IsFocused }

func (s *inputDecoratorState) isHovering() bool {
	return s.Widget().o.IsHovering && !s.decoration().Disabled
}

func (s *inputDecoratorState) isEmpty() bool { return s.Widget().o.IsEmpty }

func (s *inputDecoratorState) floatingLabelEnabled() bool {
	return s.decoration().FloatingLabelBehavior.orAuto() != FloatingLabelBehaviorNever
}

func (s *inputDecoratorState) labelShouldWithdraw() bool {
	return s.Widget().labelShouldWithdraw() || s.decoration().FloatingLabelBehavior.orAuto() == FloatingLabelBehaviorAlways
}

// DidUpdateWidget moves the label when it has to, and shakes it when an error
// appears.
//
// Flutter: material/input_decorator.dart _InputDecoratorState.didUpdateWidget
func (s *inputDecoratorState) DidUpdateWidget(old *InputDecorator) {
	w := s.Widget()
	if !w.decoration.Equal(old.decoration) {
		s.effective = nil
	}
	floatBehaviorChanged := w.decoration.FloatingLabelBehavior != old.decoration.FloatingLabelBehavior
	if w.labelShouldWithdraw() != old.labelShouldWithdraw() || floatBehaviorChanged {
		if s.floatingLabelEnabled() && s.labelShouldWithdraw() {
			s.floatingLabelController.Forward()
		} else {
			s.floatingLabelController.Reverse()
		}
	}
	errorText, hasError := s.decoration().ErrorText.Get()
	oldErrorText, hadError := old.decoration.ErrorText.Get()
	if s.floatingLabelController.Status() == animation.AnimationStatusCompleted && hasError && (!hadError || errorText != oldErrorText) {
		s.shakingLabelController.SetValue(0.0)
		s.shakingLabelController.Forward()
	}
}

func (s *inputDecoratorState) defaultM2BorderColor(theme *ThemeData) ui.Color {
	d := s.decoration()
	if d.Disabled && !s.isFocused() || false {
		_ = d
	}
	if d.Disabled && !s.isFocused() {
		if d.Filled.OrElse(false) && (d.Border == nil || !d.Border.IsOutline()) {
			return Colors.Transparent
		}
		return theme.DisabledColor
	}
	if s.hasError() {
		return theme.ColorScheme.Error()
	}
	if s.isFocused() {
		return theme.ColorScheme.Primary()
	}
	if d.Filled.Value() {
		return theme.HintColor
	}
	enabledColor := theme.ColorScheme.OnSurface().WithOpacity(0.38)
	if s.isHovering() {
		hoverColor := d.HoverColor.OrElse(theme.HoverColor)
		return ui.ColorAlphaBlend(hoverColor.WithOpacity(0.12), enabledColor)
	}
	return enabledColor
}

func (s *inputDecoratorState) fillColor(defaults InputDecorationThemeData) ui.Color {
	d := s.decoration()
	if !d.Filled.OrElse(false) {
		// filled == null is the same as filled == false.
		return Colors.Transparent
	}
	if d.FillColor != nil {
		return d.FillColor.Resolve(s.widgetState())
	}
	return defaults.FillColor.Resolve(s.widgetState())
}

func (s *inputDecoratorState) hoverColor(theme *ThemeData) ui.Color {
	d := s.decoration()
	if !d.Filled.OrElse(false) || d.Disabled {
		return Colors.Transparent
	}
	return d.HoverColor.OrElse(theme.HoverColor)
}

func resolveColorOr(p widgets.WidgetStateProperty[ui.Color], states widgets.WidgetStates, fallback widgets.WidgetStateProperty[ui.Color]) ui.Color {
	if p != nil {
		return p.Resolve(states)
	}
	return fallback.Resolve(states)
}

func (s *inputDecoratorState) iconColor(defaults InputDecorationThemeData) ui.Color {
	return resolveColorOr(s.decoration().IconColor, s.widgetState(), defaults.IconColor)
}

// buttonForeground is the foreground color of the icon buttons of the theme in
// the states, if there is one.
func buttonForeground(theme IconButtonThemeData, states widgets.WidgetStates) (ui.Color, bool) {
	if theme.Style == nil || theme.Style.ForegroundColor == nil {
		return ui.Color{}, false
	}
	return theme.Style.ForegroundColor.Resolve(states).Get()
}

func (s *inputDecoratorState) prefixIconColor(iconButtonTheme IconButtonThemeData, defaults InputDecorationThemeData) ui.Color {
	states := s.widgetState()
	if p := s.decoration().PrefixIconColor; p != nil {
		return p.Resolve(states)
	}
	if c, ok := buttonForeground(iconButtonTheme, states); ok {
		return c
	}
	return defaults.PrefixIconColor.Resolve(states)
}

func (s *inputDecoratorState) suffixIconColor(iconButtonTheme IconButtonThemeData, defaults InputDecorationThemeData) ui.Color {
	states := s.widgetState()
	if p := s.decoration().SuffixIconColor; p != nil {
		return p.Resolve(states)
	}
	if c, ok := buttonForeground(iconButtonTheme, states); ok {
		return c
	}
	return defaults.SuffixIconColor.Resolve(states)
}

// hasInlineLabel is true if the label will be shown and the hint will not: if
// the field is not focused, has no value, a label was provided, and the floating
// label behavior isn't always, then the label appears where the hint would.
//
// Flutter: material/input_decorator.dart _InputDecoratorState._hasInlineLabel
func (s *inputDecoratorState) hasInlineLabel() bool {
	d := s.decoration()
	return !s.labelShouldWithdraw() && (d.LabelText.IsSome() || d.Label != nil)
}

// shouldShowLabel: if the label is a floating placeholder, it's always shown.
func (s *inputDecoratorState) shouldShowLabel() bool {
	return s.hasInlineLabel() || s.floatingLabelEnabled()
}

func resolveStyle(p widgets.WidgetStateProperty[*painting.TextStyle], states widgets.WidgetStates) *painting.TextStyle {
	if p == nil {
		return nil
	}
	return p.Resolve(states)
}

// inlineLabelStyle is the base style for the label when it is displayed inline,
// i.e. when it appears in place of the empty text field.
//
// Flutter: material/input_decorator.dart _InputDecoratorState._getInlineLabelStyle
func (s *inputDecoratorState) inlineLabelStyle(theme *ThemeData, defaults InputDecorationThemeData) *painting.TextStyle {
	states := s.widgetState()
	defaultStyle := resolveStyle(defaults.LabelStyle, states)
	style := resolveStyle(s.decoration().LabelStyle, states)
	return theme.TextTheme.TitleMedium.Merge(s.Widget().o.BaseStyle).Merge(defaultStyle).Merge(style).
		CopyWith(painting.TextStyleCopy{Height: foundation.Some(1.0)})
}

// inlineHintStyle is the base style for the hint when it is displayed inline.
//
// Flutter: material/input_decorator.dart _InputDecoratorState._getInlineHintStyle
func (s *inputDecoratorState) inlineHintStyle(theme *ThemeData, defaults InputDecorationThemeData) *painting.TextStyle {
	states := s.widgetState()
	defaultStyle := resolveStyle(defaults.HintStyle, states)
	style := resolveStyle(s.decoration().HintStyle, states)
	base := theme.TextTheme.TitleMedium
	if theme.UseMaterial3 {
		base = theme.TextTheme.BodyLarge
	}
	return base.Merge(s.Widget().o.BaseStyle).Merge(defaultStyle).Merge(style)
}

func (s *inputDecoratorState) floatingLabelStyle(theme *ThemeData, defaults InputDecorationThemeData) *painting.TextStyle {
	states := s.widgetState()
	d := s.decoration()
	defaultTextStyle := resolveStyle(defaults.FloatingLabelStyle, states)
	if s.hasError() && d.ErrorStyle != nil {
		if es := d.ErrorStyle.Resolve(states); es != nil && es.Color.IsSome() {
			defaultTextStyle = defaultTextStyle.CopyWith(painting.TextStyleCopy{Color: es.Color})
		}
	}
	// The floating label style falls back to the label style (the one that is not
	// resolved yet, as Flutter merges the property object).
	fallback := d.FloatingLabelStyle
	if fallback == nil {
		fallback = d.LabelStyle
	}
	defaultTextStyle = defaultTextStyle.Merge(resolveStyle(fallback, states))
	style := resolveStyle(d.FloatingLabelStyle, states)
	return theme.TextTheme.TitleMedium.Merge(s.Widget().o.BaseStyle).Merge(defaultTextStyle).Merge(style).
		CopyWith(painting.TextStyleCopy{Height: foundation.Some(1.0)})
}

func (s *inputDecoratorState) helperStyle(defaults InputDecorationThemeData) *painting.TextStyle {
	states := s.widgetState()
	return resolveStyle(defaults.HelperStyle, states).Merge(resolveStyle(s.decoration().HelperStyle, states))
}

func (s *inputDecoratorState) errorStyle(defaults InputDecorationThemeData) *painting.TextStyle {
	states := s.widgetState()
	return resolveStyle(defaults.ErrorStyle, states).Merge(resolveStyle(s.decoration().ErrorStyle, states))
}

// widgetState is the set of states of the field.
//
// Flutter: material/input_decorator.dart _InputDecoratorState.widgetState
func (s *inputDecoratorState) widgetState() widgets.WidgetStates {
	var states widgets.WidgetStates
	if s.decoration().Disabled {
		states = states.With(widgets.WidgetStateDisabled)
	}
	if s.isFocused() {
		states = states.With(widgets.WidgetStateFocused)
	}
	if s.isHovering() {
		states = states.With(widgets.WidgetStateHovered)
	}
	if s.hasError() {
		states = states.With(widgets.WidgetStateError)
	}
	return states
}

func (s *inputDecoratorState) defaultBorder(theme *ThemeData, defaults InputDecorationThemeData) InputBorder {
	d := s.decoration()
	states := s.widgetState()
	var border InputBorder
	if p, ok := d.Border.(widgets.WidgetStateProperty[InputBorder]); ok {
		border = p.Resolve(states)
	} else {
		border = d.Border
	}
	if border == nil {
		border = NewUnderlineInputBorder()
	}
	if _, ok := d.Border.(widgets.WidgetStateProperty[InputBorder]); ok {
		return border
	}
	if border.BorderSide() == painting.BorderSideNone {
		return border
	}
	if theme.UseMaterial3 {
		if d.Filled.Value() {
			decorationTheme := InputDecorationThemeOf(s.Context())
			prop := decorationTheme.ActiveIndicatorBorder
			if prop == nil {
				prop = defaults.ActiveIndicatorBorder
			}
			return border.CopyWithBorderSide(prop.Resolve(states).OrElse(painting.BorderSideNone))
		}
		return border.CopyWithBorderSide(defaults.OutlineBorder.Resolve(states).OrElse(painting.BorderSideNone))
	}
	width := 1.0
	switch {
	case d.IsCollapsed.Value() || d.Border == InputBorderNone || d.Disabled:
		width = 0.0
	case s.isFocused():
		width = 2.0
	}
	return border.CopyWithBorderSide(painting.NewBorderSide(painting.BorderSideOpts{
		Color: foundation.Some(s.defaultM2BorderColor(theme)), Width: foundation.Some(width)}))
}

func (s *inputDecoratorState) buildTransition(child widgets.Widget, a animation.Animation[float64]) widgets.Widget {
	if s.curvedParent != a {
		if s.curvedAnimation != nil {
			s.curvedAnimation.Dispose()
		}
		s.curvedAnimation = animation.NewCurvedAnimation(a, inputTransitionCurve)
		s.curvedParent = a
	}
	return widgets.NewFadeTransition(s.curvedAnimation, child)
}

func topStartLayout(currentChild widgets.Widget, previousChildren []widgets.Widget) widgets.Widget {
	children := append([]widgets.Widget{}, previousChildren...)
	if currentChild != nil {
		children = append(children, currentChild)
	}
	return widgets.NewStack(children)
}

// iconSlot is the icon decoration, with its theme and a cursor that does not
// change.
func (s *inputDecoratorState) iconSlot(theme *ThemeData, defaults InputDecorationThemeData, iconSize float64) widgets.Widget {
	d := s.decoration()
	if d.Icon == nil {
		return nil
	}
	return widgets.NewMouseRegion(
		widgets.NewPadding(painting.EdgeInsetsDirectional{End: 16.0},
			widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{
				Color: foundation.Some(s.iconColor(defaults)), Size: foundation.Some(iconSize)}), d.Icon)),
		widgets.MouseRegionOpts{Cursor: services.SystemMouseCursors.Basic})
}

// affixIcon is a prefix or suffix icon: centered, with the constraints, the icon
// theme and the icon button theme of the field.
func (s *inputDecoratorState) affixIcon(icon widgets.Widget, constraints foundation.Opt[rendering.BoxConstraints], color ui.Color,
	iconSize float64, tag semantics.SemanticsTag, visualDensity VisualDensity, iconButtonTheme IconButtonThemeData) widgets.Widget {
	if icon == nil {
		return nil
	}
	style := NewButtonStyle(ButtonStyleOpts{
		ForegroundColor: widgets.NewWidgetStatePropertyAll(foundation.Some(color)),
		IconSize:        widgets.NewWidgetStatePropertyAll(foundation.Some(iconSize)),
	}).Merge(iconButtonTheme.Style)
	return widgets.NewCenter(
		widgets.NewMouseRegion(
			widgets.NewConstrainedBox(
				constraints.OrElse(visualDensity.EffectiveConstraints(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{
					MinWidth: foundation.Some(MinInteractiveDimension), MinHeight: foundation.Some(MinInteractiveDimension)}))),
				widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(color), Size: foundation.Some(iconSize)}),
					NewIconButtonTheme(NewIconButtonThemeData(IconButtonThemeDataOpts{Style: style}),
						widgets.NewSemantics(icon, widgets.SemanticsOpts{TagForChildren: foundation.Some(tag)})))),
			widgets.MouseRegionOpts{Cursor: services.SystemMouseCursors.Basic}),
		widgets.CenterOpts{WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)})
}

// Build builds the decorator: the container with the border, and the parts that
// the decoration has.
//
// Flutter: material/input_decorator.dart _InputDecoratorState.build
func (s *inputDecoratorState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	d := s.decoration()
	theme := ThemeOf(bc)
	visualDensity := d.VisualDensity.OrElse(theme.VisualDensity)
	useMaterial3 := theme.UseMaterial3
	var defaults InputDecorationThemeData
	if useMaterial3 {
		defaults = inputDecoratorDefaultsM3(bc)
	} else {
		defaults = inputDecoratorDefaultsM2(bc)
	}
	iconButtonTheme := IconButtonThemeOf(bc)
	labelStyle := s.inlineLabelStyle(theme, defaults)
	textBaseline := labelStyle.TextBaseline
	if textBaseline == ui.TextBaselineUnspecified {
		textBaseline = ui.TextBaselineAlphabetic
	}
	hintStyle := s.inlineHintStyle(theme, defaults)
	maintainHintSize := !d.NoMaintainHintSize
	states := s.widgetState()

	var hint widgets.Widget
	if d.Hint != nil || d.HintText.IsSome() {
		hintWidget := d.Hint
		if hintWidget == nil {
			overflow := hintStyle.Overflow
			if overflow == painting.TextOverflowUnspecified && d.HintMaxLines.IsSome() {
				overflow = painting.TextOverflowEllipsis
			}
			textOpts := widgets.TextOpts{Style: hintStyle, Overflow: overflow, TextAlign: w.o.TextAlign, MaxLines: d.HintMaxLines}
			if dir, ok := d.HintTextDirection.Get(); ok {
				textOpts.TextDirection = dir
			}
			hintWidget = widgets.NewText(d.HintText.Value(), textOpts)
		}
		showHint := s.isEmpty() && !s.hasInlineLabel()
		fade := d.HintFadeDuration.OrElse(inputHintFadeTransitionDuration)
		if maintainHintSize {
			opacity := 0.0
			if showHint {
				opacity = 1.0
			}
			hint = widgets.NewAnimatedOpacity(opacity, fade, hintWidget, widgets.AnimatedOpacityOpts{Curve: inputTransitionCurve})
		} else {
			var switched widgets.Widget = widgets.NewSizedBox(nil)
			if showHint {
				switched = hintWidget
			}
			hint = widgets.NewAnimatedSwitcher(fade, switched, widgets.AnimatedSwitcherOpts{
				TransitionBuilder: s.buildTransition,
				LayoutBuilder:     topStartLayout,
			})
		}
	}

	var border InputBorder
	switch {
	case d.Disabled:
		if s.hasError() {
			border = d.ErrorBorder
		} else {
			border = d.DisabledBorder
		}
	case s.isFocused():
		if s.hasError() {
			border = d.FocusedErrorBorder
		} else {
			border = d.FocusedBorder
		}
	default:
		if s.hasError() {
			border = d.ErrorBorder
		} else {
			border = d.EnabledBorder
		}
	}
	if border == nil {
		border = s.defaultBorder(theme, defaults)
	}
	container := &borderContainer{
		StatefulWidget: widgets.NewStatefulWidgetBase(),
		border:         border,
		gap:            s.borderGap,
		gapAnimation:   s.floatingLabelAnimation,
		fillColor:      s.fillColor(defaults),
		hoverColor:     s.hoverColor(theme),
		isHovering:     s.isHovering(),
	}

	var label widgets.Widget
	if d.LabelText.IsSome() || d.Label != nil {
		style := labelStyle
		if s.labelShouldWithdraw() {
			style = s.floatingLabelStyle(theme, defaults)
		}
		labelChild := d.Label
		if labelChild == nil {
			labelChild = widgets.NewText(d.LabelText.Value(), widgets.TextOpts{Overflow: painting.TextOverflowEllipsis, TextAlign: w.o.TextAlign})
		}
		opacity := 0.0
		if s.shouldShowLabel() {
			opacity = 1.0
		}
		label = widgets.NewMatrixTransition(s.shakingLabelController, func(value float64) *ui.Matrix4 {
			var shakeOffset float64
			switch {
			case value <= 0.25:
				shakeOffset = -value
			case value < 0.75:
				shakeOffset = value - 0.5
			default:
				shakeOffset = (1.0 - value) * 4.0
			}
			// Shakes the floating label to the left and right when the errorText
			// first appears.
			return ui.NewMatrix4TranslationValues(float32(shakeOffset*4.0), 0.0, 0.0)
		}, widgets.NewAnimatedOpacity(opacity, inputTransitionDuration,
			widgets.NewAnimatedDefaultTextStyle(style, inputTransitionDuration, labelChild, widgets.AnimatedDefaultTextStyleOpts{Curve: inputTransitionCurve}),
			widgets.AnimatedOpacityOpts{Curve: inputTransitionCurve}))
	}

	hasPrefix := d.Prefix != nil || d.PrefixText.IsSome()
	hasSuffix := d.Suffix != nil || d.SuffixText.IsSome()
	input := w.o.Child
	// If at least two out of the three are visible, they need a semantics sort
	// order.
	needsSemanticsSortOrder := s.labelShouldWithdraw()
	if input != nil {
		needsSemanticsSortOrder = needsSemanticsSortOrder && (hasPrefix || hasSuffix)
	} else {
		needsSemanticsSortOrder = needsSemanticsSortOrder && hasPrefix && hasSuffix
	}
	affixStyle := func(p widgets.WidgetStateProperty[*painting.TextStyle]) *painting.TextStyle {
		if st := resolveStyle(p, states); st != nil {
			return st
		}
		return hintStyle
	}
	var prefix, suffix widgets.Widget
	if hasPrefix {
		a := &affixText{StatelessWidget: widgets.NewStatelessWidgetBase(), labelIsFloating: s.labelShouldWithdraw(), text: d.PrefixText,
			style: affixStyle(d.PrefixStyle), semanticsTag: decoratorPrefixSemanticsTag, child: d.Prefix}
		if needsSemanticsSortOrder {
			a.semanticsSortKey = semantics.NewOrdinalSortKey(0, s.sortName)
		}
		prefix = a
	}
	if hasSuffix {
		a := &affixText{StatelessWidget: widgets.NewStatelessWidgetBase(), labelIsFloating: s.labelShouldWithdraw(), text: d.SuffixText,
			style: affixStyle(d.SuffixStyle), semanticsTag: decoratorSuffixSemanticsTag, child: d.Suffix}
		if needsSemanticsSortOrder {
			a.semanticsSortKey = semantics.NewOrdinalSortKey(2, s.sortName)
		}
		suffix = a
	}
	if input != nil && needsSemanticsSortOrder {
		input = widgets.NewSemantics(input, widgets.SemanticsOpts{SortKey: semantics.NewOrdinalSortKey(1, s.sortName)})
	}

	decorationIsDense := d.IsDense.OrElse(false)
	iconSize := 24.0
	if decorationIsDense {
		iconSize = 18.0
	}
	icon := s.iconSlot(theme, defaults, iconSize)
	prefixIcon := s.affixIcon(d.PrefixIcon, d.PrefixIconConstraints, s.prefixIconColor(iconButtonTheme, defaults), iconSize,
		decoratorPrefixIconSemanticsTag, visualDensity, iconButtonTheme)
	suffixIcon := s.affixIcon(d.SuffixIcon, d.SuffixIconConstraints, s.suffixIconColor(iconButtonTheme, defaults), iconSize,
		decoratorSuffixIconSemanticsTag, visualDensity, iconButtonTheme)

	helperErr := &helperError{
		StatefulWidget: widgets.NewStatefulWidgetBase(),
		textAlign:      w.o.TextAlign, helper: d.Helper, helperText: d.HelperText, helperStyle: s.helperStyle(defaults), helperMaxLines: d.HelperMaxLines,
		err: d.Error, errorText: d.ErrorText, errorStyle: s.errorStyle(defaults), errorMaxLines: d.ErrorMaxLines,
	}
	var counter widgets.Widget
	if d.Counter != nil {
		counter = d.Counter
	} else if text, ok := d.CounterText.Get(); ok && text != "" {
		counter = widgets.NewSemantics(widgets.NewText(text, widgets.TextOpts{
			Style:          s.helperStyle(defaults).Merge(resolveStyle(d.CounterStyle, states)),
			Overflow:       painting.TextOverflowEllipsis,
			SemanticsLabel: d.SemanticCounterText,
		}), widgets.SemanticsOpts{Container: true, LiveRegion: foundation.Some(s.isFocused())})
	}

	// The decorator and the render object assume that the content padding has been
	// resolved to directional insets.
	textDirection := widgets.DirectionalityOf(bc)
	flipHorizontal := textDirection == ui.TextDirectionRTL
	var decorationContentPadding *painting.EdgeInsetsDirectional
	if d.ContentPadding != nil {
		rp := d.ContentPadding.Resolve(textDirection)
		left, right := rp.Left, rp.Right
		if flipHorizontal {
			left, right = right, left
		}
		p := painting.NewEdgeInsetsDirectionalFromSTEB(left, rp.Top, right, rp.Bottom)
		decorationContentPadding = &p
	}
	paddingOr := func(def painting.EdgeInsetsDirectional) painting.EdgeInsetsDirectional {
		if decorationContentPadding != nil {
			return *decorationContentPadding
		}
		return def
	}
	pick := func(m3Dense, m3, m2Dense, m2 painting.EdgeInsetsDirectional) painting.EdgeInsetsDirectional {
		switch {
		case useMaterial3 && decorationIsDense:
			return m3Dense
		case useMaterial3:
			return m3
		case decorationIsDense:
			return m2Dense
		}
		return m2
	}
	stab := painting.NewEdgeInsetsDirectionalFromSTEB
	var contentPadding painting.EdgeInsetsDirectional
	var floatingLabelHeight float64
	switch {
	case d.IsCollapsed.Value():
		floatingLabelHeight = 0.0
		contentPadding = paddingOr(painting.EdgeInsetsDirectionalZero)
	case !border.IsOutline():
		// 4.0: the vertical gap between the inline elements and the floating label.
		floatingLabelHeight = widgets.MediaQueryTextScalerOf(bc).Scale(4.0 + 0.75*labelStyle.FontSize.Value())
		if d.Filled.OrElse(false) {
			contentPadding = paddingOr(pick(stab(12, 4, 12, 4), stab(12, 8, 12, 8), stab(12, 8, 12, 8), stab(12, 12, 12, 12)))
		} else {
			// No left or right padding for underline borders that aren't filled is
			// a small concession to backwards compatibility. This eliminates the
			// most noticeable layout change introduced by flutter/flutter#13734.
			contentPadding = paddingOr(pick(stab(0, 4, 0, 4), stab(0, 8, 0, 8), stab(0, 8, 0, 8), stab(0, 12, 0, 12)))
		}
	default:
		floatingLabelHeight = 0.0
		contentPadding = paddingOr(pick(stab(12, 16, 12, 8), stab(12, 20, 12, 12), stab(12, 20, 12, 12), stab(12, 24, 12, 16)))
	}
	inputGap := 0.0
	if useMaterial3 {
		if ob, ok := border.(OutlineInputBorder); ok {
			inputGap = ob.GapPadding()
		} else if border.IsOutline() || d.Filled.OrElse(false) {
			inputGap = inputDecoratorInputExtraPadding
		}
	}
	alignment := d.FloatingLabelAlignment.OrElse(FloatingLabelAlignmentStart)
	dec := &decorator{
		SlottedMultiChildRenderObjectWidget: widgets.NewSlottedMultiChildRenderObjectWidgetBase(),
		decoration: decoration{
			contentPadding: contentPadding, isCollapsed: d.IsCollapsed.Value(), inputGap: inputGap, floatingLabelHeight: floatingLabelHeight,
			floatingLabelAlignment: alignment, floatingLabelProgress: s.floatingLabelAnimation.Value(), border: border, borderGap: s.borderGap,
			alignLabelWithHint: d.AlignLabelWithHint.OrElse(false), isDense: decorationIsDense, isEmpty: s.isEmpty(),
			visualDensity: visualDensity, maintainHintSize: maintainHintSize, maintainLabelSize: d.MaintainLabelSize,
			icon: icon, input: input, label: label, hint: hint, prefix: prefix, suffix: suffix, prefixIcon: prefixIcon, suffixIcon: suffixIcon,
			helperError: helperErr, counter: counter, container: container,
		},
		textDirection: textDirection, textBaseline: textBaseline, textAlignVertical: w.o.TextAlignVertical,
		isFocused: s.isFocused(), expands: w.o.Expands,
	}
	// Pass the error text to semantics so screen readers announce it along with the
	// input field. The hint text is already in the semantics tree, through the
	// hint Text merging up the childSemanticsConfigurationDelegate of the render
	// object, so only the error text is added here.
	semanticsHint := d.ErrorText
	var result widgets.Widget = widgets.NewSemantics(dec, widgets.SemanticsOpts{Hint: semanticsHint})
	if c, ok := d.Constraints.Get(); ok {
		return widgets.NewConstrainedBox(c, result)
	}
	return result
}

var _ = math.Inf
