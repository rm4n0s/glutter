// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// PaintValueIndicator paints the value indicator of a Slider into the overlay.
//
// Flutter: material/slider.dart PaintValueIndicator
type PaintValueIndicator func(context *rendering.PaintingContext, offset ui.Offset)

// SliderOpts are the optional parameters of NewSlider.
//
// Flutter: material/slider.dart Slider.new
type SliderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// SecondaryTrackValue is the secondary value, such as the buffered part of a
	// video; none for no secondary track.
	SecondaryTrackValue foundation.Opt[float64]
	// OnChangeStart is called when the user starts selecting a new value.
	OnChangeStart func(value float64)
	// OnChangeEnd is called when the user is done selecting a new value.
	OnChangeEnd func(value float64)
	// Min is the minimum value; the default is 0.
	Min float64
	// Max is the maximum value; the default is 1.
	Max foundation.Opt[float64]
	// Divisions is the number of discrete divisions; none for a continuous slider.
	Divisions foundation.Opt[int]
	// Label is the label above the thumb while the user drags it.
	Label foundation.Opt[string]
	// ActiveColor is the color of the active part of the track, and of the thumb.
	ActiveColor foundation.Opt[ui.Color]
	// InactiveColor is the color of the inactive part of the track.
	InactiveColor foundation.Opt[ui.Color]
	// SecondaryActiveColor is the color of the secondary track.
	SecondaryActiveColor foundation.Opt[ui.Color]
	// ThumbColor is the color of the thumb.
	ThumbColor foundation.Opt[ui.Color]
	// OverlayColor is the color of the overlay, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// MouseCursor is the cursor over the slider; it may resolve with the states.
	MouseCursor services.MouseCursor
	// SemanticFormatterCallback formats the value for the semantics.
	SemanticFormatterCallback SemanticFormatterCallback
	// FocusNode is the focus node of the slider.
	FocusNode *widgets.FocusNode
	// Autofocus asks for the focus when the slider is built.
	Autofocus bool
	// AllowedInteraction is the way the user may change the value; the default is
	// tapAndSlide.
	AllowedInteraction SliderInteraction
	// Padding is the padding around the slider.
	Padding painting.EdgeInsetsGeometry
	// ShowValueIndicator is when the value indicator shows; the default is
	// onlyForDiscrete.
	ShowValueIndicator ShowValueIndicator
	// Year2023 selects the look of 2023 (the default) or, when false, the one of
	// 2024.
	//
	// Deprecated: Flutter deprecates it after v3.27; set it to false to opt into
	// the 2024 look.
	Year2023 foundation.Opt[bool]
}

// Slider is a Material Design slider: it lets the user select from a range of
// values by moving a thumb along a track. It does not keep its own value: the
// widget calls OnChanged and shows the value that it gets.
//
// Deviation: the adaptive constructor builds the Material slider on iOS and macOS
// too, as there is no Cupertino library (SPEC §12.10).
//
// Flutter: material/slider.dart Slider
type Slider struct {
	widgets.StatefulWidget
	value     float64
	onChanged func(value float64)
	o         SliderOpts
}

// NewSlider creates the widget. A nil onChanged disables the slider. In debug
// builds it panics with the tag SliderMinGreaterThanMax, SliderValueOutOfRange,
// SliderSecondaryValueOutOfRange or SliderDivisionsInvalid when the arguments
// are not valid.
//
// Flutter: material/slider.dart Slider.new
func NewSlider(value float64, onChanged func(value float64), opts ...SliderOpts) *Slider {
	o := firstOpts(opts)
	if foundation.DebugMode {
		min, max := o.Min, o.Max.OrElse(1.0)
		switch {
		case min > max:
			panic(errors.New("SliderMinGreaterThanMax", "the minimum of a Slider must not be greater than its maximum", "min", min, "max", max))
		case value < min || value > max:
			panic(errors.NewErrf("SliderValueOutOfRange", "Value %v is not between minimum %v and maximum %v", value, min, max))
		case o.SecondaryTrackValue.IsSome() && (o.SecondaryTrackValue.Value() < min || o.SecondaryTrackValue.Value() > max):
			panic(errors.NewErrf("SliderSecondaryValueOutOfRange", "SecondaryValue %v is not between %v and %v", o.SecondaryTrackValue.Value(), min, max))
		case o.Divisions.IsSome() && o.Divisions.Value() <= 0:
			panic(errors.New("SliderDivisionsInvalid", "the divisions of a Slider must be greater than zero", "divisions", o.Divisions.Value()))
		}
	}
	return &Slider{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), value: value, onChanged: onChanged, o: o}
}

// NewSliderAdaptive creates a slider that adapts to the platform.
//
// Deviation: it is the same as NewSlider (SPEC §12.10).
//
// Flutter: material/slider.dart Slider.adaptive
func NewSliderAdaptive(value float64, onChanged func(value float64), opts ...SliderOpts) *Slider {
	o := firstOpts(opts)
	o.Padding = nil
	return NewSlider(value, onChanged, o)
}

// Value is the currently selected value.
//
// Flutter: material/slider.dart Slider.value
func (w *Slider) Value() float64 { return w.value }

// OnChanged is called when the user selects a new value; nil disables the slider.
//
// Flutter: material/slider.dart Slider.onChanged
func (w *Slider) OnChanged() func(value float64) { return w.onChanged }

func (w *Slider) min() float64 { return w.o.Min }
func (w *Slider) max() float64 { return w.o.Max.OrElse(1.0) }

// CreateState creates the state.
//
// Flutter: material/slider.dart Slider.createState
func (w *Slider) CreateState() widgets.State { return &sliderState{} }

// DebugFillProperties adds the value, the range and the other parameters.
//
// Flutter: material/slider.dart Slider.debugFillProperties
func (w *Slider) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDoubleProperty("value", foundation.Some(w.value)))
	p.Add(foundation.NewDoubleProperty("secondaryTrackValue", w.o.SecondaryTrackValue))
	var changed any
	if w.onChanged != nil {
		changed = true
	}
	p.Add(foundation.NewObjectFlagProperty("onChanged", changed, foundation.ObjectFlagPropertyOpts{IfNull: "disabled"}))
	has := func(name string, set bool) {
		var v any
		if set {
			v = true
		}
		p.Add(foundation.NewObjectFlagPropertyHas(name, v))
	}
	has("onChangeStart", w.o.OnChangeStart != nil)
	has("onChangeEnd", w.o.OnChangeEnd != nil)
	p.Add(foundation.NewDoubleProperty("min", foundation.Some(w.min())))
	p.Add(foundation.NewDoubleProperty("max", foundation.Some(w.max())))
	p.Add(foundation.NewIntProperty("divisions", w.o.Divisions))
	p.Add(foundation.NewStringProperty("label", w.o.Label))
	p.Add(colorProp("activeColor", w.o.ActiveColor))
	p.Add(colorProp("inactiveColor", w.o.InactiveColor))
	p.Add(colorProp("secondaryActiveColor", w.o.SecondaryActiveColor))
	has("semanticFormatterCallback", w.o.SemanticFormatterCallback != nil)
	has("focusNode", w.o.FocusNode != nil)
	p.Add(foundation.NewFlagProperty("autofocus", foundation.Some(w.o.Autofocus), foundation.FlagPropertyOpts{IfTrue: "autofocus"}))
}

type sliderAdjustmentType int

const (
	sliderAdjustRight sliderAdjustmentType = iota
	sliderAdjustLeft
	sliderAdjustUp
	sliderAdjustDown
)

// adjustSliderIntent moves the thumb of a slider.
//
// Flutter: material/slider.dart _AdjustSliderIntent
type adjustSliderIntent struct {
	widgets.IntentBase
	typ sliderAdjustmentType
}

var (
	traditionalNavSliderShortcutMap = widgets.ShortcutMap{
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowUp), Intent: adjustSliderIntent{typ: sliderAdjustUp}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowDown), Intent: adjustSliderIntent{typ: sliderAdjustDown}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowLeft), Intent: adjustSliderIntent{typ: sliderAdjustLeft}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowRight), Intent: adjustSliderIntent{typ: sliderAdjustRight}},
	}
	directionalNavSliderShortcutMap = widgets.ShortcutMap{
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowLeft), Intent: adjustSliderIntent{typ: sliderAdjustLeft}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowRight), Intent: adjustSliderIntent{typ: sliderAdjustRight}},
	}
)

const (
	sliderEnableAnimationDuration         = 75 * time.Millisecond
	sliderValueIndicatorAnimationDuration = 100 * time.Millisecond
)

type sliderState struct {
	widgets.StateOf[*Slider]
	widgets.TickerProvider
	overlayController        *animation.AnimationController
	valueIndicatorController *animation.AnimationController
	enableController         *animation.AnimationController
	positionController       *animation.AnimationController
	interactionTimer         *gestures.Timer

	renderObjectKey *widgets.GlobalKey[widgets.State]
	actionMap       widgets.ActionMap

	paintValueIndicator PaintValueIndicator

	dragging            bool
	currentChangedValue foundation.Opt[float64]
	internalFocusNode   *widgets.FocusNode

	valueIndicatorOverlayPortalController *widgets.OverlayPortalController
	layerLink                             *rendering.LayerLink
	focused                               bool
	hovering                              bool
}

func (s *sliderState) enabled() bool { return s.Widget().onChanged != nil }

func (s *sliderState) focusNode() *widgets.FocusNode {
	if n := s.Widget().o.FocusNode; n != nil {
		return n
	}
	return s.internalFocusNode
}

func (s *sliderState) InitState() {
	w := s.Widget()
	s.renderObjectKey = widgets.NewGlobalKey[widgets.State]()
	s.layerLink = rendering.NewLayerLink()
	s.valueIndicatorOverlayPortalController = widgets.NewOverlayPortalController(widgets.OverlayPortalControllerOpts{DebugLabel: "Slider ValueIndicator"})
	s.valueIndicatorOverlayPortalController.Show()
	s.overlayController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(RadialReactionDuration)})
	s.valueIndicatorController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(sliderValueIndicatorAnimationDuration)})
	s.enableController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(sliderEnableAnimationDuration)})
	s.positionController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(time.Duration(0))})
	if w.onChanged != nil {
		s.enableController.SetValue(1.0)
	} else {
		s.enableController.SetValue(0.0)
	}
	s.positionController.SetValue(s.convert(w.value))
	s.actionMap = widgets.ActionMap{
		reflect.TypeFor[adjustSliderIntent](): widgets.NewCallbackAction(func(i adjustSliderIntent) any { s.actionHandler(i); return nil }),
	}
	if w.o.FocusNode == nil {
		s.internalFocusNode = widgets.NewFocusNode()
	}
}

func (s *sliderState) DidUpdateWidget(old *Slider) {
	w := s.Widget()
	if old.o.FocusNode != w.o.FocusNode {
		if w.o.FocusNode == nil && s.internalFocusNode == nil {
			s.internalFocusNode = widgets.NewFocusNode()
		} else if w.o.FocusNode != nil && s.internalFocusNode != nil {
			s.internalFocusNode.Dispose()
			s.internalFocusNode = nil
		}
	}
}

func (s *sliderState) Dispose() {
	if s.interactionTimer != nil {
		s.interactionTimer.Cancel()
	}
	s.overlayController.Dispose()
	s.valueIndicatorController.Dispose()
	s.enableController.Dispose()
	s.positionController.Dispose()
	if s.internalFocusNode != nil {
		s.internalFocusNode.Dispose()
	}
}

func (s *sliderState) handleChanged(value float64) {
	lerpValue := s.lerp(value)
	if s.currentChangedValue != foundation.Some(lerpValue) {
		s.currentChangedValue = foundation.Some(lerpValue)
		if lerpValue != s.Widget().value {
			s.Widget().onChanged(lerpValue)
		}
	}
}

func (s *sliderState) handleDragStart(value float64) {
	s.SetState(func() { s.dragging = true })
	if f := s.Widget().o.OnChangeStart; f != nil {
		f(s.lerp(value))
	}
}

func (s *sliderState) handleDragEnd(value float64) {
	s.SetState(func() { s.dragging = false })
	s.currentChangedValue = foundation.None[float64]()
	if f := s.Widget().o.OnChangeEnd; f != nil {
		f(s.lerp(value))
	}
}

func (s *sliderState) sliderRender() *renderSlider {
	ctx := s.renderObjectKey.CurrentContext()
	if ctx == nil {
		return nil
	}
	r, _ := ctx.FindRenderObject().(*renderSlider)
	return r
}

func (s *sliderState) actionHandler(intent adjustSliderIntent) {
	ctx := s.renderObjectKey.CurrentContext()
	if ctx == nil {
		return
	}
	directionality := widgets.DirectionalityOf(ctx)
	var shouldIncrease bool
	switch intent.typ {
	case sliderAdjustUp:
		shouldIncrease = true
	case sliderAdjustDown:
		shouldIncrease = false
	case sliderAdjustLeft:
		shouldIncrease = directionality == ui.TextDirectionRTL
	case sliderAdjustRight:
		shouldIncrease = directionality == ui.TextDirectionLTR
	}
	slider := s.sliderRender()
	if slider == nil {
		return
	}
	if shouldIncrease {
		slider.increaseAction()
	} else {
		slider.decreaseAction()
	}
}

func (s *sliderState) handleFocusHighlightChanged(focused bool) {
	if focused != s.focused {
		s.SetState(func() { s.focused = focused })
	}
}

func (s *sliderState) handleHoverChanged(hovering bool) {
	if hovering != s.hovering {
		s.SetState(func() { s.hovering = hovering })
	}
}

// lerp converts a value from 0 to 1 to one from min to max.
func (s *sliderState) lerp(value float64) float64 {
	w := s.Widget()
	return value*(w.max()-w.min()) + w.min()
}

func (s *sliderState) discretize(value float64) float64 {
	divisions := float64(s.Widget().o.Divisions.Value())
	return math.Round(value*divisions) / divisions
}

func (s *sliderState) convert(value float64) float64 {
	ret := s.unlerp(value)
	if s.Widget().o.Divisions.IsSome() {
		ret = s.discretize(ret)
	}
	return ret
}

func (s *sliderState) unlerp(value float64) float64 {
	w := s.Widget()
	if w.max() > w.min() {
		return (value - w.min()) / (w.max() - w.min())
	}
	return 0.0
}

func (s *sliderState) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	widgets.DebugCheckHasMediaQuery(bc)
	return s.buildMaterialSlider(bc)
}

func (s *sliderState) buildMaterialSlider(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	theme := ThemeOf(bc)
	sliderTheme := SliderThemeOf(bc)
	year2023 := o.Year2023.OrElse(sliderTheme.Year2023.OrElse(true))
	defaults := sliderDefaults(bc, theme.UseMaterial3, year2023)

	var states widgets.WidgetStates
	if !s.enabled() {
		states = states.With(widgets.WidgetStateDisabled)
	}
	if s.hovering {
		states = states.With(widgets.WidgetStateHovered)
	}
	if s.focused {
		states = states.With(widgets.WidgetStateFocused)
	}
	if s.dragging {
		states = states.With(widgets.WidgetStateDragged)
	}

	valueIndicatorShape := sliderTheme.ValueIndicatorShape
	if valueIndicatorShape == nil {
		valueIndicatorShape = defaults.ValueIndicatorShape
	}
	var valueIndicatorColor ui.Color
	if _, ok := valueIndicatorShape.(RectangularSliderValueIndicatorShape); ok {
		if c, ok := sliderTheme.ValueIndicatorColor.Get(); ok {
			valueIndicatorColor = c
		} else {
			valueIndicatorColor = ui.ColorAlphaBlend(theme.ColorScheme.OnSurface().WithOpacity(0.60), theme.ColorScheme.Surface().WithOpacity(0.90))
		}
	} else {
		valueIndicatorColor = firstOptColor(o.ActiveColor, sliderTheme.ValueIndicatorColor, defaults.ValueIndicatorColor).Value()
	}

	effectiveOverlayColor := func() foundation.Opt[ui.Color] {
		var fromWidget, fromActive foundation.Opt[ui.Color]
		if o.OverlayColor != nil {
			fromWidget = o.OverlayColor.Resolve(states)
		}
		if c, ok := o.ActiveColor.Get(); ok {
			fromActive = foundation.Some(c.WithOpacity(0.12))
		}
		return firstOptColor(fromWidget, fromActive, sliderTheme.overlayColorFor(states), defaults.overlayColorFor(states))
	}

	valueIndicatorTextStyle := sliderTheme.ValueIndicatorTextStyle
	if valueIndicatorTextStyle == nil {
		valueIndicatorTextStyle = defaults.ValueIndicatorTextStyle
	}
	if widgets.MediaQueryBoldTextOf(bc) {
		valueIndicatorTextStyle = valueIndicatorTextStyle.Merge(painting.NewTextStyle(painting.TextStyleOpts{FontWeight: ui.FontWeightBold}))
	}

	st := sliderTheme
	st.TrackHeight = firstOptDouble(sliderTheme.TrackHeight, defaults.TrackHeight)
	st.ActiveTrackColor = firstOptColor(o.ActiveColor, sliderTheme.ActiveTrackColor, defaults.ActiveTrackColor)
	st.InactiveTrackColor = firstOptColor(o.InactiveColor, sliderTheme.InactiveTrackColor, defaults.InactiveTrackColor)
	st.SecondaryActiveTrackColor = firstOptColor(o.SecondaryActiveColor, sliderTheme.SecondaryActiveTrackColor, defaults.SecondaryActiveTrackColor)
	st.DisabledActiveTrackColor = firstOptColor(sliderTheme.DisabledActiveTrackColor, defaults.DisabledActiveTrackColor)
	st.DisabledInactiveTrackColor = firstOptColor(sliderTheme.DisabledInactiveTrackColor, defaults.DisabledInactiveTrackColor)
	st.DisabledSecondaryActiveTrackColor = firstOptColor(sliderTheme.DisabledSecondaryActiveTrackColor, defaults.DisabledSecondaryActiveTrackColor)
	st.ActiveTickMarkColor = firstOptColor(o.InactiveColor, sliderTheme.ActiveTickMarkColor, defaults.ActiveTickMarkColor)
	st.InactiveTickMarkColor = firstOptColor(o.ActiveColor, sliderTheme.InactiveTickMarkColor, defaults.InactiveTickMarkColor)
	st.DisabledActiveTickMarkColor = firstOptColor(sliderTheme.DisabledActiveTickMarkColor, defaults.DisabledActiveTickMarkColor)
	st.DisabledInactiveTickMarkColor = firstOptColor(sliderTheme.DisabledInactiveTickMarkColor, defaults.DisabledInactiveTickMarkColor)
	st.ThumbColor = firstOptColor(o.ThumbColor, o.ActiveColor, sliderTheme.ThumbColor, defaults.ThumbColor)
	st.DisabledThumbColor = firstOptColor(sliderTheme.DisabledThumbColor, defaults.DisabledThumbColor)
	st.OverlayColor = effectiveOverlayColor()
	st.StateOverlayColor = nil
	st.ValueIndicatorColor = foundation.Some(valueIndicatorColor)
	if st.TrackShape == nil {
		st.TrackShape = defaults.TrackShape
	}
	if st.TickMarkShape == nil {
		st.TickMarkShape = defaults.TickMarkShape
	}
	if st.ThumbShape == nil {
		st.ThumbShape = defaults.ThumbShape
	}
	if st.OverlayShape == nil {
		st.OverlayShape = defaults.OverlayShape
	}
	st.ValueIndicatorShape = valueIndicatorShape
	if o.ShowValueIndicator != ShowValueIndicatorUnspecified {
		st.ShowValueIndicator = o.ShowValueIndicator
	} else if st.ShowValueIndicator == ShowValueIndicatorUnspecified {
		st.ShowValueIndicator = ShowValueIndicatorOnlyForDiscrete
	}
	st.ValueIndicatorTextStyle = valueIndicatorTextStyle
	if o.Padding != nil {
		st.Padding = o.Padding
	}
	if st.ThumbSize == nil {
		st.ThumbSize = defaults.ThumbSize
	}
	st.TrackGap = firstOptDouble(sliderTheme.TrackGap, defaults.TrackGap)

	effectiveMouseCursor := func() services.MouseCursor {
		if o.MouseCursor != nil {
			if p, ok := o.MouseCursor.(widgets.WidgetStateProperty[services.MouseCursor]); ok {
				if c := p.Resolve(states); c != nil {
					return c
				}
			} else {
				return o.MouseCursor
			}
		}
		if sliderTheme.MouseCursor != nil {
			if c := sliderTheme.MouseCursor.Resolve(states); c != nil {
				return c
			}
		}
		return widgets.WidgetStateMouseCursorClickable.Resolve(states)
	}()
	effectiveAllowedInteraction := o.AllowedInteraction
	if effectiveAllowedInteraction == SliderInteractionUnspecified {
		effectiveAllowedInteraction = sliderTheme.AllowedInteraction
	}
	if effectiveAllowedInteraction == SliderInteractionUnspecified {
		effectiveAllowedInteraction = SliderInteractionTapAndSlide
	}

	var handleDidGainAccessibilityFocus func()
	if theme.Platform == foundation.TargetPlatformWindows {
		handleDidGainAccessibilityFocus = func() {
			if n := s.focusNode(); !n.HasFocus() && n.CanRequestFocus() {
				n.RequestFocus()
			}
		}
	}

	shortcutMap := traditionalNavSliderShortcutMap
	if widgets.MediaQueryNavigationModeOf(bc) == widgets.NavigationModeDirectional {
		shortcutMap = directionalNavSliderShortcutMap
	}

	fontSize := st.ValueIndicatorTextStyle.FontSize.OrElse(painting.DefaultFontSize)
	fontSizeToScale := fontSize
	if fontSize == 0.0 {
		fontSizeToScale = painting.DefaultFontSize
	}
	textScaler := widgets.MediaQueryTextScalerOf(bc)
	if theme.UseMaterial3 {
		textScaler = textScaler.Clamp(painting.TextScalerClampOpts{MaxScaleFactor: foundation.Some(1.3)})
	}
	effectiveTextScale := textScaler.Scale(fontSizeToScale) / fontSizeToScale

	var onChanged func(float64)
	if w.onChanged != nil && w.max() > w.min() {
		onChanged = s.handleChanged
	}
	var secondary foundation.Opt[float64]
	if sv, ok := o.SecondaryTrackValue.Get(); ok {
		secondary = foundation.Some(s.convert(sv))
	}
	var result widgets.Widget = widgets.NewCompositedTransformTarget(s.layerLink, &sliderRenderObjectWidget{
		LeafRenderObjectWidget: widgets.NewLeafRenderObjectWidgetBase(widgets.WidgetOpts{Key: s.renderObjectKey}),
		value:                  s.convert(w.value), secondaryTrackValue: secondary, divisions: o.Divisions, label: o.Label,
		sliderTheme: st, textScaleFactor: effectiveTextScale, screenSize: widgets.MediaQuerySizeOf(bc),
		onChanged: onChanged, onChangeStart: s.handleDragStart, onChangeEnd: s.handleDragEnd, state: s,
		semanticFormatterCallback: o.SemanticFormatterCallback, onDidGainAccessibilityFocus: handleDidGainAccessibilityFocus,
		hasFocus: s.focused, hovering: s.hovering, allowedInteraction: effectiveAllowedInteraction,
		textEngine: textEngineOf(bc), gestures: widgets.WidgetsBindingOf(bc).Gestures(),
		gestureSettings: widgets.MediaQueryGestureSettingsOf(bc),
	})

	if st.Padding != nil {
		result = widgets.NewPadding(st.Padding, result)
	}
	showValueIndicator := st.ShowValueIndicator
	result = widgets.NewOverlayPortal(s.valueIndicatorOverlayPortalController, func(widgets.BuildContext) widgets.Widget {
		return s.buildValueIndicator(showValueIndicator)
	}, result)

	return widgets.NewFocusableActionDetector(result, widgets.FocusableActionDetectorOpts{
		Actions: s.actionMap, Shortcuts: shortcutMap, FocusNode: s.focusNode(), Autofocus: o.Autofocus, Disabled: !s.enabled(),
		OnShowFocusHighlight: s.handleFocusHighlightChanged, OnShowHoverHighlight: s.handleHoverChanged,
		MouseCursor: effectiveMouseCursor, ExcludeFocusSemantics: true,
	})
}

func (s *sliderState) buildValueIndicator(showValueIndicator ShowValueIndicator) widgets.Widget {
	valueIndicator := widgets.NewCompositedTransformFollower(s.layerLink, &valueIndicatorRenderObjectWidget{
		LeafRenderObjectWidget: widgets.NewLeafRenderObjectWidgetBase(), state: s})
	hidden := widgets.NewSizedBoxShrink(nil)
	divisions := s.Widget().o.Divisions
	switch showValueIndicator {
	case ShowValueIndicatorNever:
		return hidden
	case ShowValueIndicatorOnlyForDiscrete:
		if divisions.IsSome() {
			return valueIndicator
		}
		return hidden
	case ShowValueIndicatorOnlyForContinuous:
		if divisions.IsNone() {
			return valueIndicator
		}
		return hidden
	}
	return valueIndicator
}

// textEngineOf is the text engine of the app of the context.
func textEngineOf(bc widgets.BuildContext) ui.ParagraphFactory {
	if b := widgets.MaybeWidgetsBindingOf(bc); b != nil {
		return b.PlatformDispatcher().TextEngine()
	}
	return nil
}

// sliderRenderObjectWidget makes the render object of a slider.
//
// Flutter: material/slider.dart _SliderRenderObjectWidget
type sliderRenderObjectWidget struct {
	widgets.LeafRenderObjectWidget
	value                       float64
	secondaryTrackValue         foundation.Opt[float64]
	divisions                   foundation.Opt[int]
	label                       foundation.Opt[string]
	sliderTheme                 SliderThemeData
	textScaleFactor             float64
	screenSize                  ui.Size
	onChanged                   func(float64)
	onChangeStart               func(float64)
	onChangeEnd                 func(float64)
	state                       *sliderState
	semanticFormatterCallback   SemanticFormatterCallback
	onDidGainAccessibilityFocus func()
	hasFocus                    bool
	hovering                    bool
	allowedInteraction          SliderInteraction
	textEngine                  ui.ParagraphFactory
	gestures                    *gestures.GestureBinding
	gestureSettings             gestures.DeviceGestureSettings
}

func (w *sliderRenderObjectWidget) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	r := &renderSlider{
		value: w.value, secondaryTrackValue: w.secondaryTrackValue, divisions: w.divisions, label: w.label,
		sliderTheme: w.sliderTheme, textScaleFactor: w.textScaleFactor, screenSize: w.screenSize, onChanged: w.onChanged,
		onChangeStart: w.onChangeStart, onChangeEnd: w.onChangeEnd, state: w.state, textDirection: widgets.DirectionalityOf(bc),
		semanticFormatterCallback: w.semanticFormatterCallback, onDidGainAccessibilityFocus: w.onDidGainAccessibilityFocus,
		platform: ThemeOf(bc).Platform, hasFocus: w.hasFocus, hovering: w.hovering, allowedInteraction: w.allowedInteraction,
		gestures: w.gestures, gestureSettings: w.gestureSettings,
	}
	rendering.InitRenderBox(&r.RenderBox, r)
	r.labelPainter = painting.NewTextPainter(painting.TextPainterOpts{TextEngine: w.textEngine})
	r.updateLabelPainter()
	r.overlayAnimation = animation.NewCurvedAnimation(w.state.overlayController, animation.Curves.FastOutSlowIn)
	r.valueIndicatorAnimation = animation.NewCurvedAnimation(w.state.valueIndicatorController, animation.Curves.FastOutSlowIn)
	r.enableAnimation = animation.NewCurvedAnimation(w.state.enableController, animation.Curves.EaseInOut)
	return r
}

func (w *sliderRenderObjectWidget) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderSlider)
	r.setDivisions(w.divisions)
	r.setValue(w.value)
	r.setSecondaryTrackValue(w.secondaryTrackValue)
	r.setLabel(w.label)
	r.setSliderTheme(w.sliderTheme)
	r.setTextScaleFactor(w.textScaleFactor)
	r.setScreenSize(w.screenSize)
	r.setOnChanged(w.onChanged)
	r.onChangeStart = w.onChangeStart
	r.onChangeEnd = w.onChangeEnd
	r.setTextDirection(widgets.DirectionalityOf(bc))
	r.setSemanticFormatterCallback(w.semanticFormatterCallback)
	r.onDidGainAccessibilityFocus = w.onDidGainAccessibilityFocus
	r.setPlatform(ThemeOf(bc).Platform)
	r.setHasFocus(w.hasFocus)
	r.setHovering(w.hovering)
	r.setGestureSettings(w.gestureSettings)
	r.setAllowedInteraction(w.allowedInteraction)
}

func (w *sliderRenderObjectWidget) DidUnmountRenderObject(rendering.RenderObject) {}

const (
	sliderPositionAnimationDuration = 75 * time.Millisecond
	sliderMinimumInteractionTime    = 500 * time.Millisecond
	sliderMinPreferredTrackWidth    = 144.0
)

// renderSlider is the render object of a slider: it lays out the track, the thumb
// and the overlay, handles the taps and the drags, and paints.
//
// Flutter: material/slider.dart _RenderSlider
type renderSlider struct {
	rendering.RenderBox

	state                       *sliderState
	overlayAnimation            *animation.CurvedAnimation
	valueIndicatorAnimation     *animation.CurvedAnimation
	enableAnimation             *animation.CurvedAnimation
	labelPainter                *painting.TextPainter
	gestures                    *gestures.GestureBinding
	drag                        *gestures.HorizontalDragGestureRecognizer
	tap                         *gestures.TapGestureRecognizer
	gestureSettings             gestures.DeviceGestureSettings
	active                      bool
	onDidGainAccessibilityFocus func()
	currentDragValue            float64
	overlayRect                 foundation.Opt[ui.Rect]

	value                     float64
	secondaryTrackValue       foundation.Opt[float64]
	platform                  foundation.TargetPlatform
	semanticFormatterCallback SemanticFormatterCallback
	divisions                 foundation.Opt[int]
	label                     foundation.Opt[string]
	sliderTheme               SliderThemeData
	textScaleFactor           float64
	screenSize                ui.Size
	onChanged                 func(float64)
	onChangeStart             func(float64)
	onChangeEnd               func(float64)
	textDirection             ui.TextDirection
	hasFocus                  bool
	hovering                  bool
	hoveringThumb             bool
	allowedInteraction        SliderInteraction

	listenerHandles [4]foundation.ListenerHandle
}

func (r *renderSlider) isInteractive() bool { return r.onChanged != nil }
func (r *renderSlider) isDiscrete() bool    { return r.divisions.IsSome() && r.divisions.Value() > 0 }

func (r *renderSlider) thumbSizeHeight() float64 {
	return r.sliderTheme.ThumbShape.GetPreferredSize(r.isInteractive(), r.isDiscrete()).Height
}

func (r *renderSlider) overlayHeight() float64 {
	return r.sliderTheme.OverlayShape.GetPreferredSize(r.isInteractive(), r.isDiscrete()).Height
}

func (r *renderSlider) sliderPartSizes() []ui.Size {
	partHeight := r.overlayHeight()
	if r.sliderTheme.Padding != nil {
		partHeight = r.thumbSizeHeight()
	}
	return []ui.Size{
		{Width: r.sliderTheme.OverlayShape.GetPreferredSize(r.isInteractive(), r.isDiscrete()).Width, Height: partHeight},
		r.sliderTheme.ThumbShape.GetPreferredSize(r.isInteractive(), r.isDiscrete()),
		r.sliderTheme.TickMarkShape.GetPreferredSize(r.sliderTheme, r.isInteractive()),
	}
}

func (r *renderSlider) maxSliderPartWidth() float64 {
	w := 0.0
	for i, s := range r.sliderPartSizes() {
		if i == 0 || s.Width > w {
			w = s.Width
		}
	}
	return w
}

func (r *renderSlider) maxSliderPartHeight() float64 {
	h := 0.0
	for i, s := range r.sliderPartSizes() {
		if i == 0 || s.Height > h {
			h = s.Height
		}
	}
	return h
}

func (r *renderSlider) minPreferredTrackHeight() float64 { return r.sliderTheme.TrackHeight.Value() }

func (r *renderSlider) trackRect() ui.Rect {
	return r.sliderTheme.TrackShape.GetPreferredRect(SliderTrackRectOpts{ParentBox: r, SliderTheme: r.sliderTheme})
}

func (r *renderSlider) setValue(newValue float64) {
	convertedValue := newValue
	if r.isDiscrete() {
		convertedValue = r.discretize(newValue)
	}
	if convertedValue == r.value {
		return
	}
	r.value = convertedValue
	pc := r.state.positionController
	if r.isDiscrete() {
		distance := math.Abs(r.value - pc.Value())
		if distance != 0.0 {
			pc.Duration = foundation.Some(time.Duration(float64(sliderPositionAnimationDuration) * (1.0 / distance)))
		} else {
			pc.Duration = foundation.Some(time.Duration(0))
		}
		pc.AnimateTo(convertedValue, animation.AnimateOpts{Curve: animation.Curves.EaseInOut})
	} else {
		pc.SetValue(convertedValue)
	}
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderSlider) setSecondaryTrackValue(v foundation.Opt[float64]) {
	if v == r.secondaryTrackValue {
		return
	}
	r.secondaryTrackValue = v
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderSlider) setGestureSettings(v gestures.DeviceGestureSettings) {
	r.gestureSettings = v
	if r.drag != nil {
		r.drag.GestureSettings = foundation.Some(v)
		r.tap.GestureSettings = foundation.Some(v)
	}
}

func (r *renderSlider) setPlatform(v foundation.TargetPlatform) {
	if r.platform == v {
		return
	}
	r.platform = v
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderSlider) setSemanticFormatterCallback(v SemanticFormatterCallback) {
	if sameFunc(r.semanticFormatterCallback, v) {
		return
	}
	r.semanticFormatterCallback = v
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderSlider) setDivisions(v foundation.Opt[int]) {
	if v == r.divisions {
		return
	}
	r.divisions = v
	r.MarkNeedsPaint()
}

func (r *renderSlider) setLabel(v foundation.Opt[string]) {
	if v == r.label {
		return
	}
	r.label = v
	r.updateLabelPainter()
}

func (r *renderSlider) setSliderTheme(v SliderThemeData) {
	if v.Equal(r.sliderTheme) {
		return
	}
	r.sliderTheme = v
	r.updateLabelPainter()
}

func (r *renderSlider) setTextScaleFactor(v float64) {
	if v == r.textScaleFactor {
		return
	}
	r.textScaleFactor = v
	r.updateLabelPainter()
}

func (r *renderSlider) setScreenSize(v ui.Size) {
	if v == r.screenSize {
		return
	}
	r.screenSize = v
	r.MarkNeedsPaint()
}

func (r *renderSlider) setOnChanged(v func(float64)) {
	if (v == nil) == (r.onChanged == nil) {
		r.onChanged = v
		return
	}
	r.onChanged = v
	if r.isInteractive() {
		r.state.enableController.Forward()
	} else {
		r.state.enableController.Reverse()
	}
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderSlider) setTextDirection(v ui.TextDirection) {
	if v == r.textDirection {
		return
	}
	r.textDirection = v
	r.updateLabelPainter()
}

func (r *renderSlider) setHasFocus(v bool) {
	if v == r.hasFocus {
		return
	}
	r.hasFocus = v
	r.updateForFocus(v)
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderSlider) setHovering(v bool) {
	if v == r.hovering {
		return
	}
	r.hovering = v
	r.updateForHover(v)
}

func (r *renderSlider) setHoveringThumb(v bool) {
	if v == r.hoveringThumb {
		return
	}
	r.hoveringThumb = v
	r.updateForHover(r.hovering)
}

func (r *renderSlider) setAllowedInteraction(v SliderInteraction) {
	if v == r.allowedInteraction {
		return
	}
	r.allowedInteraction = v
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderSlider) updateForFocus(focused bool) {
	if focused {
		r.state.overlayController.Forward()
		if r.shouldShowValueIndicatorWhenDragged() {
			r.state.valueIndicatorController.Forward()
		}
	} else {
		r.state.overlayController.Reverse()
		if r.shouldShowValueIndicatorWhenDragged() {
			r.state.valueIndicatorController.Reverse()
		}
	}
}

func (r *renderSlider) updateForHover(hovered bool) {
	if hovered && r.hoveringThumb {
		r.state.overlayController.Forward()
	} else if !r.active && !r.hasFocus {
		r.state.overlayController.Reverse()
	}
}

func (r *renderSlider) shouldAlwaysShowValueIndicator() bool {
	return r.sliderTheme.ShowValueIndicator == ShowValueIndicatorAlwaysVisible
}

func (r *renderSlider) shouldShowValueIndicatorWhenDragged() bool {
	switch r.sliderTheme.ShowValueIndicator {
	case ShowValueIndicatorOnlyForDiscrete:
		return r.isDiscrete()
	case ShowValueIndicatorOnlyForContinuous:
		return !r.isDiscrete()
	case ShowValueIndicatorAlways, ShowValueIndicatorOnDrag:
		return true
	}
	return false
}

func (r *renderSlider) adjustmentUnit() float64 {
	switch r.platform {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		return 0.1
	}
	return 0.05
}

func (r *renderSlider) updateLabelPainter() {
	if label, ok := r.label.Get(); ok {
		r.labelPainter.SetText(painting.NewTextSpan(painting.TextSpanOpts{Style: r.sliderTheme.ValueIndicatorTextStyle, Text: foundation.Some(label)}))
		r.labelPainter.SetTextDirection(r.textDirection)
		r.labelPainter.SetTextScaler(painting.NewTextScalerLinear(r.textScaleFactor))
		r.labelPainter.Layout(0, math.Inf(1))
	} else {
		r.labelPainter.SetText(nil)
	}
	r.MarkNeedsLayout()
}

// SystemFontsDidChange lays the label out again.
//
// Flutter: material/slider.dart _RenderSlider.systemFontsDidChange
func (r *renderSlider) SystemFontsDidChange() {
	r.labelPainter.MarkNeedsLayout()
	r.updateLabelPainter()
}

// Attach creates the recognizers and listens to the animations.
//
// Flutter: material/slider.dart _RenderSlider.attach
func (r *renderSlider) Attach(owner *rendering.PipelineOwner) {
	r.RenderBox.Attach(owner)
	team := gestures.NewGestureArenaTeam(r.gestures.GestureArena)
	r.drag = gestures.NewHorizontalDragGestureRecognizer(r.gestures, gestures.DragGestureRecognizerOpts{DebugOwner: r})
	r.drag.SetTeam(team)
	r.drag.OnStart = r.handleDragStart
	r.drag.OnUpdate = r.handleDragUpdate
	r.drag.OnEnd = r.handleDragEnd
	r.drag.OnCancel = r.endInteraction
	r.drag.GestureSettings = foundation.Some(r.gestureSettings)
	r.tap = gestures.NewTapGestureRecognizer(r.gestures, gestures.TapGestureRecognizerOpts{DebugOwner: r})
	r.tap.SetTeam(team)
	r.tap.OnTapDown = r.handleTapDown
	r.tap.OnTapUp = r.handleTapUp
	r.tap.GestureSettings = foundation.Some(r.gestureSettings)
	r.listenerHandles[0] = r.overlayAnimation.AddListener(r.MarkNeedsPaint)
	r.listenerHandles[1] = r.valueIndicatorAnimation.AddListener(r.MarkNeedsPaint)
	r.listenerHandles[2] = r.enableAnimation.AddListener(r.MarkNeedsPaint)
	r.listenerHandles[3] = r.state.positionController.AddListener(r.MarkNeedsPaint)
}

// Detach disposes the recognizers and stops listening.
//
// Flutter: material/slider.dart _RenderSlider.detach
func (r *renderSlider) Detach() {
	r.overlayAnimation.RemoveListener(r.listenerHandles[0])
	r.valueIndicatorAnimation.RemoveListener(r.listenerHandles[1])
	r.enableAnimation.RemoveListener(r.listenerHandles[2])
	r.state.positionController.RemoveListener(r.listenerHandles[3])
	if r.drag != nil {
		r.drag.Dispose()
		r.tap.Dispose()
		r.drag, r.tap = nil, nil
	}
	r.RenderBox.Detach()
}

// Dispose releases the animations and the label painter.
//
// Flutter: material/slider.dart _RenderSlider.dispose
func (r *renderSlider) Dispose() {
	r.labelPainter.Dispose()
	r.enableAnimation.Dispose()
	r.valueIndicatorAnimation.Dispose()
	r.overlayAnimation.Dispose()
	r.RenderBox.Dispose()
}

func (r *renderSlider) getValueFromVisualPosition(visualPosition float64) float64 {
	if r.textDirection == ui.TextDirectionRTL {
		return 1.0 - visualPosition
	}
	return visualPosition
}

func (r *renderSlider) getValueFromGlobalPosition(globalPosition ui.Offset) float64 {
	trackRect := r.trackRect()
	visualPosition := (r.GlobalToLocal(globalPosition, nil).Dx - trackRect.Left) / trackRect.Width()
	return r.getValueFromVisualPosition(visualPosition)
}

func (r *renderSlider) discretize(value float64) float64 {
	result := ui.ClampDouble(value, 0.0, 1.0)
	if r.isDiscrete() {
		d := float64(r.divisions.Value())
		result = math.Round(result*d) / d
	}
	return result
}

func (r *renderSlider) startInteraction(globalPosition ui.Offset) {
	if !r.state.Mounted() {
		return
	}
	if r.active || !r.isInteractive() {
		return
	}
	switch r.allowedInteraction {
	case SliderInteractionTapAndSlide, SliderInteractionTapOnly:
		r.active = true
		r.currentDragValue = r.getValueFromGlobalPosition(globalPosition)
	case SliderInteractionSlideThumb:
		if r.isPointerOnOverlay(globalPosition) {
			r.active = true
			r.currentDragValue = r.value
		}
	case SliderInteractionSlideOnly:
		r.active = true
		r.currentDragValue = r.value
	}
	if !r.active {
		return
	}
	if r.onChangeStart != nil {
		r.onChangeStart(r.discretize(r.value))
	}
	r.onChanged(r.discretize(r.currentDragValue))
	r.state.overlayController.Forward()
	if r.shouldShowValueIndicatorWhenDragged() {
		r.state.valueIndicatorController.Forward()
		if r.state.interactionTimer != nil {
			r.state.interactionTimer.Cancel()
		}
		dilation := widgets.WidgetsBindingOf(r.state.Context()).Scheduler().TimeDilation()
		r.state.interactionTimer = r.gestures.NewTimer(time.Duration(float64(sliderMinimumInteractionTime)*dilation), func() {
			r.state.interactionTimer = nil
			if !r.active && r.state.valueIndicatorController.Status() == animation.AnimationStatusCompleted {
				r.state.valueIndicatorController.Reverse()
			}
		})
	}
}

func (r *renderSlider) endInteraction() {
	if !r.state.Mounted() {
		return
	}
	if r.active {
		if r.onChangeEnd != nil {
			r.onChangeEnd(r.discretize(r.currentDragValue))
		}
		r.active = false
		r.currentDragValue = 0.0
		r.state.overlayController.Reverse()
		if r.shouldShowValueIndicatorWhenDragged() && r.state.interactionTimer == nil {
			r.state.valueIndicatorController.Reverse()
		}
	}
}

func (r *renderSlider) handleDragStart(details gestures.DragStartDetails) {
	r.startInteraction(details.GlobalPosition)
}

func (r *renderSlider) handleDragUpdate(details gestures.DragUpdateDetails) {
	if !r.state.Mounted() {
		return
	}
	switch r.allowedInteraction {
	case SliderInteractionTapAndSlide, SliderInteractionSlideOnly, SliderInteractionSlideThumb:
		if r.active && r.isInteractive() {
			valueDelta := details.PrimaryDelta.Value() / r.trackRect().Width()
			if r.textDirection == ui.TextDirectionRTL {
				r.currentDragValue -= valueDelta
			} else {
				r.currentDragValue += valueDelta
			}
			r.onChanged(r.discretize(r.currentDragValue))
		}
	}
}

func (r *renderSlider) handleDragEnd(gestures.DragEndDetails) { r.endInteraction() }

func (r *renderSlider) handleTapDown(details gestures.TapDownDetails) {
	r.startInteraction(details.GlobalPosition)
}

func (r *renderSlider) handleTapUp(gestures.TapUpDetails) { r.endInteraction() }

func (r *renderSlider) isPointerOnOverlay(globalPosition ui.Offset) bool {
	rect, ok := r.overlayRect.Get()
	return ok && rect.Contains(r.GlobalToLocal(globalPosition, nil))
}

// HitTestSelf is true: the whole box takes the presses.
//
// Flutter: material/slider.dart _RenderSlider.hitTestSelf
func (r *renderSlider) HitTestSelf(ui.Offset) bool { return true }

// HandleEvent hands a pointer-down to the recognizers, and tracks the hover over
// the thumb.
//
// Flutter: material/slider.dart _RenderSlider.handleEvent
func (r *renderSlider) HandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) {
	if !r.state.Mounted() {
		return
	}
	if down, ok := event.(*gestures.PointerDownEvent); ok && r.isInteractive() && r.drag != nil {
		r.drag.AddPointer(down)
		r.tap.AddPointer(down)
	}
	if rect, ok := r.overlayRect.Get(); ok && r.isInteractive() {
		r.setHoveringThumb(rect.Contains(event.LocalPosition()))
	}
}

// ComputeMinIntrinsicWidth is the preferred track width and the widest part.
//
// Flutter: material/slider.dart _RenderSlider.computeMinIntrinsicWidth
func (r *renderSlider) ComputeMinIntrinsicWidth(float64) float64 {
	return sliderMinPreferredTrackWidth + r.maxSliderPartWidth()
}

// ComputeMaxIntrinsicWidth is the preferred track width and the widest part.
//
// Flutter: material/slider.dart _RenderSlider.computeMaxIntrinsicWidth
func (r *renderSlider) ComputeMaxIntrinsicWidth(float64) float64 {
	return sliderMinPreferredTrackWidth + r.maxSliderPartWidth()
}

// ComputeMinIntrinsicHeight is the taller of the track and the parts.
//
// Flutter: material/slider.dart _RenderSlider.computeMinIntrinsicHeight
func (r *renderSlider) ComputeMinIntrinsicHeight(float64) float64 {
	return math.Max(r.minPreferredTrackHeight(), r.maxSliderPartHeight())
}

// ComputeMaxIntrinsicHeight is the taller of the track and the parts.
//
// Flutter: material/slider.dart _RenderSlider.computeMaxIntrinsicHeight
func (r *renderSlider) ComputeMaxIntrinsicHeight(float64) float64 {
	return math.Max(r.minPreferredTrackHeight(), r.maxSliderPartHeight())
}

// SizedByParent is true.
//
// Flutter: material/slider.dart _RenderSlider.sizedByParent
func (r *renderSlider) SizedByParent() bool { return true }

// ComputeDryLayout is the biggest size, or the preferred one when unbounded.
//
// Flutter: material/slider.dart _RenderSlider.computeDryLayout
func (r *renderSlider) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	width := sliderMinPreferredTrackWidth + r.maxSliderPartWidth()
	if c.HasBoundedWidth() {
		width = c.MaxWidth
	}
	height := math.Max(r.minPreferredTrackHeight(), r.maxSliderPartHeight())
	if c.HasBoundedHeight() {
		height = c.MaxHeight
	}
	return ui.Size{Width: width, Height: height}
}

// PerformLayout does nothing: the box is sized by its parent.
//
// Flutter: material/slider.dart _RenderSlider (sizedByParent)
func (r *renderSlider) PerformLayout() {}

// Paint paints the track, the overlay, the tick marks and the thumb, and records
// the painting of the value indicator.
//
// Flutter: material/slider.dart _RenderSlider.paint
func (r *renderSlider) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	controllerValue := r.state.positionController.Value()
	var visualPosition float64
	secondaryVisualPosition := foundation.None[float64]()
	if r.textDirection == ui.TextDirectionRTL {
		visualPosition = 1.0 - controllerValue
		if sv, ok := r.secondaryTrackValue.Get(); ok {
			secondaryVisualPosition = foundation.Some(1.0 - sv)
		}
	} else {
		visualPosition = controllerValue
		secondaryVisualPosition = r.secondaryTrackValue
	}
	st := r.sliderTheme
	trackRect := st.TrackShape.GetPreferredRect(SliderTrackRectOpts{ParentBox: r, Offset: offset, SliderTheme: st, IsDiscrete: r.isDiscrete()})
	thumbCenter := r.calcThumbCenter(trackRect, visualPosition)

	if r.isInteractive() {
		overlaySize := st.OverlayShape.GetPreferredSize(r.isInteractive(), false)
		r.overlayRect = foundation.Some(ui.NewRectFromCircle(thumbCenter, overlaySize.Width/2.0))
	}
	secondaryOffset := foundation.None[ui.Offset]()
	if sp, ok := secondaryVisualPosition.Get(); ok {
		secondaryOffset = foundation.Some(ui.Offset{Dx: trackRect.Left + sp*trackRect.Width(), Dy: trackRect.Center().Dy})
	}

	thumbSizeAt := func(states widgets.WidgetStates) foundation.Opt[ui.Size] {
		if st.ThumbSize == nil {
			return foundation.None[ui.Size]()
		}
		return st.ThumbSize.Resolve(states)
	}
	var thumbWidth, thumbHeight, trackGap foundation.Opt[float64]
	if sz, ok := thumbSizeAt(0).Get(); ok {
		thumbWidth, thumbHeight = foundation.Some(sz.Width), foundation.Some(sz.Height)
	}
	trackGap = st.TrackGap
	var pressedThumbWidth foundation.Opt[float64]
	if sz, ok := thumbSizeAt(widgets.WidgetStates(0).With(widgets.WidgetStatePressed)).Get(); ok {
		pressedThumbWidth = foundation.Some(sz.Width)
	}
	if r.active && thumbWidth.IsSome() && pressedThumbWidth.IsSome() && trackGap.IsSome() {
		delta := thumbWidth.Value() - pressedThumbWidth.Value()
		if thumbWidth.Value() > 0.0 {
			thumbWidth = pressedThumbWidth
		}
		if trackGap.Value() > 0.0 {
			trackGap = foundation.Some(trackGap.Value() - delta/2)
		}
	}

	trackTheme := st
	trackTheme.TrackGap = trackGap
	st.TrackShape.Paint(context, offset, SliderTrackPaintOpts{
		ParentBox: r, SliderTheme: trackTheme, EnableAnimation: r.enableAnimation, TextDirection: r.textDirection,
		ThumbCenter: thumbCenter, SecondaryOffset: secondaryOffset, IsDiscrete: r.isDiscrete(), IsEnabled: r.isInteractive()})

	if r.overlayAnimation.Status() != animation.AnimationStatusDismissed {
		st.OverlayShape.Paint(context, thumbCenter, r.componentPaintOpts(r.overlayAnimation, r.enableAnimation, st))
	}

	if r.isDiscrete() {
		divisions := float64(r.divisions.Value())
		tickMarkWidth := st.TickMarkShape.GetPreferredSize(st, r.isInteractive()).Width
		discreteTrackPadding := trackRect.Height()
		adjustedTrackWidth := trackRect.Width() - discreteTrackPadding
		if adjustedTrackWidth/divisions >= 3.0*tickMarkWidth {
			dy := trackRect.Center().Dy
			for i := 0; i <= r.divisions.Value(); i++ {
				value := float64(i) / divisions
				dx := trackRect.Left + value*adjustedTrackWidth + discreteTrackPadding/2
				st.TickMarkShape.Paint(context, ui.Offset{Dx: dx, Dy: dy}, SliderTickMarkPaintOpts{
					ParentBox: r, SliderTheme: st, EnableAnimation: r.enableAnimation, TextDirection: r.textDirection,
					ThumbCenter: thumbCenter, IsEnabled: r.isInteractive()})
			}
		}
	}

	if r.isInteractive() && r.label.IsSome() &&
		((r.shouldShowValueIndicatorWhenDragged() && r.valueIndicatorAnimation.Status() != animation.AnimationStatusDismissed) || r.shouldAlwaysShowValueIndicator()) {
		r.state.paintValueIndicator = func(context *rendering.PaintingContext, offset ui.Offset) {
			if r.Attached() && r.labelPainter.Text() != nil {
				var activation, enable animation.Animation[float64] = r.valueIndicatorAnimation, r.enableAnimation
				if r.shouldAlwaysShowValueIndicator() {
					activation, enable = animation.NewAlwaysStoppedAnimation(1.0), animation.NewAlwaysStoppedAnimation(1.0)
				}
				if st.ValueIndicatorShape != nil {
					st.ValueIndicatorShape.Paint(context, offset.Add(thumbCenter), r.componentPaintOpts(activation, enable, st))
				}
			}
		}
	} else {
		r.state.paintValueIndicator = nil
	}

	thumbTheme := st
	if tw, ok := thumbWidth.Get(); ok && thumbHeight.IsSome() {
		size := ui.Size{Width: tw, Height: thumbHeight.Value()}
		thumbTheme.ThumbSize = widgets.NewWidgetStatePropertyAll(foundation.Some(size))
	}
	st.ThumbShape.Paint(context, thumbCenter, r.componentPaintOpts(r.overlayAnimation, r.enableAnimation, thumbTheme))
}

func (r *renderSlider) sizeWithOverflow() ui.Size {
	if r.screenSize.IsEmpty() {
		return r.Size()
	}
	return r.screenSize
}

func (r *renderSlider) componentPaintOpts(activation, enable animation.Animation[float64], theme SliderThemeData) SliderComponentPaintOpts {
	return SliderComponentPaintOpts{
		ActivationAnimation: activation, EnableAnimation: enable, IsDiscrete: r.isDiscrete(), LabelPainter: r.labelPainter,
		ParentBox: r, SliderTheme: theme, TextDirection: r.textDirection, Value: r.value, TextScaleFactor: r.textScaleFactor,
		SizeWithOverflow: r.sizeWithOverflow()}
}

func (r *renderSlider) calcThumbCenter(trackRect ui.Rect, visualPosition float64) ui.Offset {
	padding := 0.0
	if r.sliderTheme.TrackShape.IsRounded() {
		padding = trackRect.Height()
	}
	var thumbPosition float64
	if r.isDiscrete() {
		thumbPosition = trackRect.Left + visualPosition*(trackRect.Width()-padding) + padding/2
	} else {
		thumbPosition = trackRect.Left + visualPosition*trackRect.Width()
	}
	thumbPreferredSize := r.sliderTheme.ThumbShape.GetPreferredSize(r.isInteractive(), r.isDiscrete())
	thumbPadding := 0.0
	if padding > thumbPreferredSize.Width/2 {
		thumbPadding = padding / 2
	}
	return ui.Offset{Dx: ui.ClampDouble(thumbPosition, trackRect.Left+thumbPadding, trackRect.Right-thumbPadding), Dy: trackRect.Center().Dy}
}

func (r *renderSlider) semanticThumbCenter() ui.Offset {
	visualPosition := r.value
	if r.textDirection == ui.TextDirectionRTL {
		visualPosition = 1.0 - r.value
	}
	return r.calcThumbCenter(r.trackRect(), visualPosition)
}

// AssembleSemanticsNode sets the rectangle of the node around the thumb.
//
// Flutter: material/slider.dart _RenderSlider.assembleSemanticsNode
func (r *renderSlider) AssembleSemanticsNode(node *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, _ []*semantics.SemanticsNode) {
	node.SetRect(ui.NewRectFromCenter(r.semanticThumbCenter(), MinInteractiveDimension, MinInteractiveDimension))
	node.UpdateWith(config, nil)
}

// DescribeSemanticsConfiguration makes the slider a semantic boundary with the
// value and the actions to increase and decrease it.
//
// Flutter: material/slider.dart _RenderSlider.describeSemanticsConfiguration
func (r *renderSlider) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderBox.DescribeSemanticsConfiguration(config)
	config.SetIsSemanticBoundary(true)
	config.SetIsEnabled(foundation.Some(r.isInteractive()))
	if label, ok := r.label.Get(); ok {
		config.SetLabel(label)
	}
	config.SetIsSlider(true)
	config.SetIsFocusable(r.isInteractive())
	config.SetIsFocused(foundation.Some(r.hasFocus))
	if r.onDidGainAccessibilityFocus != nil {
		config.SetOnDidGainAccessibilityFocus(r.onDidGainAccessibilityFocus)
	}
	config.SetTextDirection(r.textDirection)
	if r.isInteractive() {
		config.SetOnIncrease(r.increaseAction)
		config.SetOnDecrease(r.decreaseAction)
		config.SetOnFocus(r.onFocusAction)
	}
	unit := r.semanticActionUnit()
	if r.semanticFormatterCallback != nil {
		config.SetValue(r.semanticFormatterCallback(r.state.lerp(r.value)))
		config.SetIncreasedValue(r.semanticFormatterCallback(r.state.lerp(ui.ClampDouble(r.value+unit, 0.0, 1.0))))
		config.SetDecreasedValue(r.semanticFormatterCallback(r.state.lerp(ui.ClampDouble(r.value-unit, 0.0, 1.0))))
	} else {
		pct := func(v float64) string { return strconv.Itoa(int(math.Round(v*100))) + "%" }
		config.SetValue(pct(r.value))
		config.SetIncreasedValue(pct(ui.ClampDouble(r.value+unit, 0.0, 1.0)))
		config.SetDecreasedValue(pct(ui.ClampDouble(r.value-unit, 0.0, 1.0)))
	}
}

func (r *renderSlider) semanticActionUnit() float64 {
	if r.divisions.IsSome() {
		return 1.0 / float64(r.divisions.Value())
	}
	return r.adjustmentUnit()
}

func (r *renderSlider) onFocusAction() {
	if r.isInteractive() {
		if !r.state.Mounted() {
			return
		}
		if !r.hasFocus {
			r.state.focusNode().RequestFocus()
		}
	}
}

func (r *renderSlider) increaseAction() {
	if r.isInteractive() {
		r.onChangeStart(r.currentValue())
		increase := r.increaseValue()
		r.onChanged(increase)
		r.onChangeEnd(increase)
	}
}

func (r *renderSlider) decreaseAction() {
	if r.isInteractive() {
		r.onChangeStart(r.currentValue())
		decrease := r.decreaseValue()
		r.onChanged(decrease)
		r.onChangeEnd(decrease)
	}
}

func (r *renderSlider) currentValue() float64 { return ui.ClampDouble(r.value, 0.0, 1.0) }
func (r *renderSlider) increaseValue() float64 {
	return ui.ClampDouble(r.value+r.semanticActionUnit(), 0.0, 1.0)
}
func (r *renderSlider) decreaseValue() float64 {
	return ui.ClampDouble(r.value-r.semanticActionUnit(), 0.0, 1.0)
}

// valueIndicatorRenderObjectWidget makes the render object that paints the value
// indicator in the overlay.
//
// Flutter: material/slider.dart _ValueIndicatorRenderObjectWidget
type valueIndicatorRenderObjectWidget struct {
	widgets.LeafRenderObjectWidget
	state *sliderState
}

func (w *valueIndicatorRenderObjectWidget) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderValueIndicator{state: w.state}
	rendering.InitRenderBox(&r.RenderBox, r)
	r.valueIndicatorAnimation = animation.NewCurvedAnimation(w.state.valueIndicatorController, animation.Curves.FastOutSlowIn)
	return r
}

func (w *valueIndicatorRenderObjectWidget) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	ro.(*renderValueIndicator).state = w.state
}

func (w *valueIndicatorRenderObjectWidget) DidUnmountRenderObject(rendering.RenderObject) {}

// renderValueIndicator paints the value indicator of a slider.
//
// Flutter: material/slider.dart _RenderValueIndicator
type renderValueIndicator struct {
	rendering.RenderBox
	state                   *sliderState
	valueIndicatorAnimation *animation.CurvedAnimation
	handles                 [2]foundation.ListenerHandle
}

func (r *renderValueIndicator) SizedByParent() bool { return true }

func (r *renderValueIndicator) Attach(owner *rendering.PipelineOwner) {
	r.RenderBox.Attach(owner)
	r.handles[0] = r.valueIndicatorAnimation.AddListener(r.MarkNeedsPaint)
	r.handles[1] = r.state.positionController.AddListener(r.MarkNeedsPaint)
}

func (r *renderValueIndicator) Detach() {
	r.valueIndicatorAnimation.RemoveListener(r.handles[0])
	r.state.positionController.RemoveListener(r.handles[1])
	r.RenderBox.Detach()
}

func (r *renderValueIndicator) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if f := r.state.paintValueIndicator; f != nil {
		f(context, offset)
	}
}

func (r *renderValueIndicator) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Smallest()
}

func (r *renderValueIndicator) PerformLayout() {}

func (r *renderValueIndicator) Dispose() {
	r.valueIndicatorAnimation.Dispose()
	r.RenderBox.Dispose()
}
