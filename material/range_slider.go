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

// PaintRangeValueIndicator paints a value indicator of a RangeSlider into the
// overlay.
//
// Flutter: material/range_slider.dart PaintRangeValueIndicator
type PaintRangeValueIndicator func(context *rendering.PaintingContext, offset ui.Offset)

// RangeSliderOpts are the optional parameters of NewRangeSlider.
//
// Flutter: material/range_slider.dart RangeSlider.new
type RangeSliderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnChangeStart is called when the user starts selecting a new range.
	OnChangeStart func(values RangeValues)
	// OnChangeEnd is called when the user is done selecting a new range.
	OnChangeEnd func(values RangeValues)
	// Min is the minimum value; the default is 0.
	Min float64
	// Max is the maximum value; the default is 1.
	Max foundation.Opt[float64]
	// Divisions is the number of discrete divisions; none for a continuous slider.
	Divisions foundation.Opt[int]
	// Labels are the labels above the thumbs while the user drags them.
	Labels foundation.Opt[RangeLabels]
	// ActiveColor is the color of the active part of the track, and of the thumbs.
	ActiveColor foundation.Opt[ui.Color]
	// InactiveColor is the color of the inactive part of the track.
	InactiveColor foundation.Opt[ui.Color]
	// OverlayColor is the color of the overlay, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// MouseCursor is the cursor over the slider, in each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// SemanticFormatterCallback formats the values for the semantics.
	SemanticFormatterCallback SemanticFormatterCallback
	// Padding is the padding around the slider.
	Padding painting.EdgeInsetsGeometry
	// Year2023 selects the look of 2023 (the default) or, when false, the one of
	// 2024.
	//
	// Deprecated: Flutter deprecates it after v3.30; set it to false to opt into
	// the 2024 look.
	Year2023 foundation.Opt[bool]
}

// RangeSlider is a Material Design slider for a range of values: it lets the user
// select a range with two thumbs. It does not keep its own values: the widget
// calls OnChanged and shows the values that it gets.
//
// Flutter: material/range_slider.dart RangeSlider
type RangeSlider struct {
	widgets.StatefulWidget
	values    RangeValues
	onChanged func(values RangeValues)
	o         RangeSliderOpts
}

// rangeSliderMinTouchTargetWidth is the minimum width of the touch target of a
// thumb.
//
// Flutter: material/range_slider.dart RangeSlider._minTouchTargetWidth
const rangeSliderMinTouchTargetWidth = MinInteractiveDimension

// NewRangeSlider creates the widget. A nil onChanged disables the slider. In debug
// builds it panics with the tag RangeSliderMinGreaterThanMax,
// RangeSliderValuesInvalid or RangeSliderDivisionsInvalid when the arguments are
// not valid.
//
// Flutter: material/range_slider.dart RangeSlider.new
func NewRangeSlider(values RangeValues, onChanged func(values RangeValues), opts ...RangeSliderOpts) *RangeSlider {
	o := firstOpts(opts)
	if foundation.DebugMode {
		min, max := o.Min, o.Max.OrElse(1.0)
		switch {
		case min > max:
			panic(errors.New("RangeSliderMinGreaterThanMax", "the minimum of a RangeSlider must not be greater than its maximum", "min", min, "max", max))
		case values.Start > values.End || values.Start < min || values.Start > max || values.End < min || values.End > max:
			panic(errors.New("RangeSliderValuesInvalid", "the values of a RangeSlider must be in order and between the minimum and the maximum",
				"values", values.String(), "min", min, "max", max))
		case o.Divisions.IsSome() && o.Divisions.Value() <= 0:
			panic(errors.New("RangeSliderDivisionsInvalid", "the divisions of a RangeSlider must be greater than zero", "divisions", o.Divisions.Value()))
		}
	}
	return &RangeSlider{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), values: values, onChanged: onChanged, o: o}
}

// Values are the currently selected values.
//
// Flutter: material/range_slider.dart RangeSlider.values
func (w *RangeSlider) Values() RangeValues { return w.values }

// OnChanged is called when the user selects a new range; nil disables the slider.
//
// Flutter: material/range_slider.dart RangeSlider.onChanged
func (w *RangeSlider) OnChanged() func(values RangeValues) { return w.onChanged }

func (w *RangeSlider) min() float64 { return w.o.Min }
func (w *RangeSlider) max() float64 { return w.o.Max.OrElse(1.0) }

// CreateState creates the state.
//
// Flutter: material/range_slider.dart RangeSlider.createState
func (w *RangeSlider) CreateState() widgets.State { return &rangeSliderState{} }

// DebugFillProperties adds the values, the range and the other parameters.
//
// Flutter: material/range_slider.dart RangeSlider.debugFillProperties
func (w *RangeSlider) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDoubleProperty("valueStart", foundation.Some(w.values.Start)))
	p.Add(foundation.NewDoubleProperty("valueEnd", foundation.Some(w.values.End)))
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
	labelStart, labelEnd := foundation.None[string](), foundation.None[string]()
	if l, ok := w.o.Labels.Get(); ok {
		labelStart, labelEnd = foundation.Some(l.Start), foundation.Some(l.End)
	}
	p.Add(foundation.NewStringProperty("labelStart", labelStart))
	p.Add(foundation.NewStringProperty("labelEnd", labelEnd))
	p.Add(colorProp("activeColor", w.o.ActiveColor))
	p.Add(colorProp("inactiveColor", w.o.InactiveColor))
	has("semanticFormatterCallback", w.o.SemanticFormatterCallback != nil)
}

type rangeSliderState struct {
	widgets.StateOf[*RangeSlider]
	widgets.TickerProvider
	startFocusNode           *widgets.FocusNode
	endFocusNode             *widgets.FocusNode
	overlayController        *animation.AnimationController
	valueIndicatorController *animation.AnimationController
	enableController         *animation.AnimationController
	startPositionController  *animation.AnimationController
	endPositionController    *animation.AnimationController
	interactionTimer         *gestures.Timer

	paintTopValueIndicator    PaintRangeValueIndicator
	paintBottomValueIndicator PaintRangeValueIndicator

	dragging           bool
	hovering           bool
	showHoverHighlight bool

	valueIndicatorOverlayPortalController *widgets.OverlayPortalController
	layerLink                             *rendering.LayerLink
}

func (s *rangeSliderState) enabled() bool { return s.Widget().onChanged != nil }

func (s *rangeSliderState) handleHoverChanged(hovering bool) {
	if hovering != s.hovering {
		s.SetState(func() {
			s.hovering = hovering
			s.showHoverHighlight = hovering && s.enabled()
		})
	}
}

func (s *rangeSliderState) InitState() {
	w := s.Widget()
	s.startFocusNode = widgets.NewFocusNode()
	s.endFocusNode = widgets.NewFocusNode()
	s.layerLink = rendering.NewLayerLink()
	s.valueIndicatorOverlayPortalController = widgets.NewOverlayPortalController(widgets.OverlayPortalControllerOpts{DebugLabel: "RangeSlider ValueIndicator"})
	s.valueIndicatorOverlayPortalController.Show()
	s.overlayController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(RadialReactionDuration)})
	s.valueIndicatorController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(sliderValueIndicatorAnimationDuration)})
	enabled := 0.0
	if s.enabled() {
		enabled = 1.0
	}
	s.enableController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(sliderEnableAnimationDuration), Value: foundation.Some(enabled)})
	s.startPositionController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(time.Duration(0)), Value: foundation.Some(s.unlerp(w.values.Start))})
	s.endPositionController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(time.Duration(0)), Value: foundation.Some(s.unlerp(w.values.End))})
}

func (s *rangeSliderState) DidUpdateWidget(old *RangeSlider) {
	if sameFunc(old.onChanged, s.Widget().onChanged) {
		return
	}
	wasEnabled := old.onChanged != nil
	isEnabled := s.enabled()
	if wasEnabled != isEnabled {
		if isEnabled {
			s.enableController.Forward()
		} else {
			s.enableController.Reverse()
		}
		s.showHoverHighlight = s.hovering && isEnabled
	}
}

func (s *rangeSliderState) Dispose() {
	if s.interactionTimer != nil {
		s.interactionTimer.Cancel()
	}
	s.overlayController.Dispose()
	s.valueIndicatorController.Dispose()
	s.enableController.Dispose()
	s.startPositionController.Dispose()
	s.endPositionController.Dispose()
	s.startFocusNode.Dispose()
	s.endFocusNode.Dispose()
}

func (s *rangeSliderState) handleChanged(values RangeValues) {
	lerpValues := s.lerpRangeValues(values)
	if lerpValues != s.Widget().values {
		s.Widget().onChanged(lerpValues)
	}
}

func (s *rangeSliderState) handleDragStart(values RangeValues) {
	s.SetState(func() { s.dragging = true })
	if f := s.Widget().o.OnChangeStart; f != nil {
		f(s.lerpRangeValues(values))
	}
}

func (s *rangeSliderState) handleDragEnd(values RangeValues) {
	s.SetState(func() { s.dragging = false })
	if f := s.Widget().o.OnChangeEnd; f != nil {
		f(s.lerpRangeValues(values))
	}
}

func (s *rangeSliderState) lerp(value float64) float64 {
	w := s.Widget()
	return ui.LerpDouble(w.min(), w.max(), value)
}

func (s *rangeSliderState) lerpRangeValues(values RangeValues) RangeValues {
	return RangeValues{Start: s.lerp(values.Start), End: s.lerp(values.End)}
}

func (s *rangeSliderState) unlerp(value float64) float64 {
	w := s.Widget()
	if w.max() > w.min() {
		return (value - w.min()) / (w.max() - w.min())
	}
	return 0.0
}

func (s *rangeSliderState) unlerpRangeValues(values RangeValues) RangeValues {
	return RangeValues{Start: s.unlerp(values.Start), End: s.unlerp(values.End)}
}

// defaultRangeThumbSelector picks the thumb that is closer to the press.
//
// Flutter: material/range_slider.dart _RangeSliderState._defaultRangeThumbSelector
func defaultRangeThumbSelector(textDirection ui.TextDirection, values RangeValues, tapValue float64, thumbSize, trackSize ui.Size, dx float64) foundation.Opt[Thumb] {
	touchRadius := math.Max(thumbSize.Width, rangeSliderMinTouchTargetWidth) / 2
	inStartTouchTarget := math.Abs(tapValue-values.Start)*trackSize.Width < touchRadius
	inEndTouchTarget := math.Abs(tapValue-values.End)*trackSize.Width < touchRadius

	if inStartTouchTarget && inEndTouchTarget {
		towardsStart, towardsEnd := dx < 0, dx > 0
		if textDirection == ui.TextDirectionRTL {
			towardsStart, towardsEnd = dx > 0, dx < 0
		}
		if towardsStart {
			return foundation.Some(ThumbStart)
		}
		if towardsEnd {
			return foundation.Some(ThumbEnd)
		}
	} else if tapValue*2 < values.Start+values.End {
		return foundation.Some(ThumbStart)
	} else {
		return foundation.Some(ThumbEnd)
	}
	return foundation.None[Thumb]()
}

func (s *rangeSliderState) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	widgets.DebugCheckHasMediaQuery(bc)
	w := s.Widget()
	o := w.o
	theme := ThemeOf(bc)
	sliderTheme := SliderThemeOf(bc)
	year2023 := o.Year2023.OrElse(sliderTheme.Year2023.OrElse(true))
	defaults := rangeSliderDefaults(bc, theme.UseMaterial3, year2023)

	var states widgets.WidgetStates
	if !s.enabled() {
		states = states.With(widgets.WidgetStateDisabled)
	}
	if s.hovering {
		states = states.With(widgets.WidgetStateHovered)
	}
	if s.dragging {
		states = states.With(widgets.WidgetStateDragged)
	}

	valueIndicatorShape := sliderTheme.RangeValueIndicatorShape
	if valueIndicatorShape == nil {
		valueIndicatorShape = defaults.RangeValueIndicatorShape
	}
	var valueIndicatorColor ui.Color
	if _, ok := valueIndicatorShape.(RectangularRangeSliderValueIndicatorShape); ok {
		if c, ok := sliderTheme.ValueIndicatorColor.Get(); ok {
			valueIndicatorColor = c
		} else {
			valueIndicatorColor = ui.ColorAlphaBlend(theme.ColorScheme.OnSurface().WithOpacity(0.60), theme.ColorScheme.Surface().WithOpacity(0.90))
		}
	} else {
		valueIndicatorColor = firstOptColor(o.ActiveColor, sliderTheme.ValueIndicatorColor, defaults.ValueIndicatorColor).Value()
	}

	var fromWidget, fromActive foundation.Opt[ui.Color]
	if o.OverlayColor != nil {
		fromWidget = o.OverlayColor.Resolve(states)
	}
	if c, ok := o.ActiveColor.Get(); ok {
		fromActive = foundation.Some(c.WithOpacity(0.12))
	}
	effectiveOverlayColor := firstOptColor(fromWidget, fromActive, sliderTheme.overlayColorFor(states), defaults.OverlayColor)

	st := sliderTheme
	st.TrackHeight = firstOptDouble(sliderTheme.TrackHeight, defaults.TrackHeight)
	st.ActiveTrackColor = firstOptColor(o.ActiveColor, sliderTheme.ActiveTrackColor, defaults.ActiveTrackColor)
	st.InactiveTrackColor = firstOptColor(o.InactiveColor, sliderTheme.InactiveTrackColor, defaults.InactiveTrackColor)
	st.DisabledActiveTrackColor = firstOptColor(sliderTheme.DisabledActiveTrackColor, defaults.DisabledActiveTrackColor)
	st.DisabledInactiveTrackColor = firstOptColor(sliderTheme.DisabledInactiveTrackColor, defaults.DisabledInactiveTrackColor)
	st.ActiveTickMarkColor = firstOptColor(o.InactiveColor, sliderTheme.ActiveTickMarkColor, defaults.ActiveTickMarkColor)
	st.InactiveTickMarkColor = firstOptColor(o.ActiveColor, sliderTheme.InactiveTickMarkColor, defaults.InactiveTickMarkColor)
	st.DisabledActiveTickMarkColor = firstOptColor(sliderTheme.DisabledActiveTickMarkColor, defaults.DisabledActiveTickMarkColor)
	st.DisabledInactiveTickMarkColor = firstOptColor(sliderTheme.DisabledInactiveTickMarkColor, defaults.DisabledInactiveTickMarkColor)
	st.ThumbColor = firstOptColor(o.ActiveColor, sliderTheme.ThumbColor, defaults.ThumbColor)
	st.OverlappingShapeStrokeColor = firstOptColor(sliderTheme.OverlappingShapeStrokeColor, defaults.OverlappingShapeStrokeColor)
	st.DisabledThumbColor = firstOptColor(sliderTheme.DisabledThumbColor, defaults.DisabledThumbColor)
	st.OverlayColor = effectiveOverlayColor
	st.StateOverlayColor = nil
	st.ValueIndicatorColor = foundation.Some(valueIndicatorColor)
	if st.RangeTrackShape == nil {
		st.RangeTrackShape = defaults.RangeTrackShape
	}
	if st.RangeTickMarkShape == nil {
		st.RangeTickMarkShape = defaults.RangeTickMarkShape
	}
	if st.RangeThumbShape == nil {
		st.RangeThumbShape = defaults.RangeThumbShape
	}
	if st.OverlayShape == nil {
		st.OverlayShape = defaults.OverlayShape
	}
	st.RangeValueIndicatorShape = valueIndicatorShape
	if st.ShowValueIndicator == ShowValueIndicatorUnspecified {
		st.ShowValueIndicator = defaults.ShowValueIndicator
	}
	if st.ValueIndicatorTextStyle == nil {
		st.ValueIndicatorTextStyle = defaults.ValueIndicatorTextStyle
	}
	st.MinThumbSeparation = firstOptDouble(sliderTheme.MinThumbSeparation, defaults.MinThumbSeparation)
	if st.ThumbSelector == nil {
		st.ThumbSelector = defaultRangeThumbSelector
	}
	if o.Padding != nil {
		st.Padding = o.Padding
	}
	if st.ThumbSize == nil {
		st.ThumbSize = defaults.ThumbSize
	}
	st.TrackGap = firstOptDouble(sliderTheme.TrackGap, defaults.TrackGap)

	var effectiveMouseCursor services.MouseCursor
	if o.MouseCursor != nil {
		effectiveMouseCursor = o.MouseCursor.Resolve(states)
	}
	if effectiveMouseCursor == nil && sliderTheme.MouseCursor != nil {
		effectiveMouseCursor = sliderTheme.MouseCursor.Resolve(states)
	}
	if effectiveMouseCursor == nil {
		effectiveMouseCursor = widgets.WidgetStateMouseCursorClickable.Resolve(states)
	}

	fontSize := st.ValueIndicatorTextStyle.FontSize.OrElse(painting.DefaultFontSize)
	fontSizeToScale := fontSize
	if fontSize == 0.0 {
		fontSizeToScale = painting.DefaultFontSize
	}
	effectiveTextScale := widgets.MediaQueryTextScalerOf(bc).Scale(fontSizeToScale) / fontSizeToScale

	var onChanged func(RangeValues)
	if s.enabled() && w.max() > w.min() {
		onChanged = s.handleChanged
	}
	showValueIndicator := st.ShowValueIndicator
	var result widgets.Widget = widgets.NewCompositedTransformTarget(s.layerLink,
		widgets.NewOverlayPortal(s.valueIndicatorOverlayPortalController, func(widgets.BuildContext) widgets.Widget {
			return s.buildValueIndicator(showValueIndicator)
		}, &rangeSliderRenderObjectWidget{
			LeafRenderObjectWidget: widgets.NewLeafRenderObjectWidgetBase(),
			values:                 s.unlerpRangeValues(w.values), divisions: o.Divisions, labels: o.Labels, sliderTheme: st,
			textScaleFactor: effectiveTextScale, screenSize: widgets.MediaQuerySizeOf(bc), onChanged: onChanged,
			onChangeStart: s.handleDragStart, onChangeEnd: s.handleDragEnd, state: s,
			semanticFormatterCallback: o.SemanticFormatterCallback, hovering: s.showHoverHighlight,
			textEngine: textEngineOf(bc), gestures: widgets.WidgetsBindingOf(bc).Gestures(),
			gestureSettings: widgets.MediaQueryGestureSettingsOf(bc),
		}))
	if st.Padding != nil {
		result = widgets.NewPadding(st.Padding, result)
	}

	return widgets.NewStack([]widgets.Widget{
		widgets.NewRow([]widgets.Widget{
			widgets.NewFocus(widgets.NewSizedBoxShrink(nil), widgets.FocusOpts{FocusNode: s.startFocusNode, ExcludeSemantics: true}),
			widgets.NewFocus(widgets.NewSizedBoxShrink(nil), widgets.FocusOpts{FocusNode: s.endFocusNode, ExcludeSemantics: true}),
		}),
		widgets.NewMouseRegion(result, widgets.MouseRegionOpts{
			OnEnter: func(*gestures.PointerEnterEvent) { s.handleHoverChanged(true) },
			OnExit:  func(*gestures.PointerExitEvent) { s.handleHoverChanged(false) },
			Cursor:  effectiveMouseCursor,
		}),
	})
}

func (s *rangeSliderState) buildValueIndicator(showValueIndicator ShowValueIndicator) widgets.Widget {
	valueIndicator := widgets.NewCompositedTransformFollower(s.layerLink, &rangeValueIndicatorRenderObjectWidget{
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

// rangeSliderRenderObjectWidget makes the render object of a range slider.
//
// Flutter: material/range_slider.dart _RangeSliderRenderObjectWidget
type rangeSliderRenderObjectWidget struct {
	widgets.LeafRenderObjectWidget
	values                    RangeValues
	divisions                 foundation.Opt[int]
	labels                    foundation.Opt[RangeLabels]
	sliderTheme               SliderThemeData
	textScaleFactor           float64
	screenSize                ui.Size
	onChanged                 func(RangeValues)
	onChangeStart             func(RangeValues)
	onChangeEnd               func(RangeValues)
	semanticFormatterCallback SemanticFormatterCallback
	state                     *rangeSliderState
	hovering                  bool
	textEngine                ui.ParagraphFactory
	gestures                  *gestures.GestureBinding
	gestureSettings           gestures.DeviceGestureSettings
}

func (w *rangeSliderRenderObjectWidget) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	r := &renderRangeSlider{
		values: w.values, divisions: w.divisions, labels: w.labels, sliderTheme: w.sliderTheme,
		textScaleFactor: w.textScaleFactor, screenSize: w.screenSize, platform: ThemeOf(bc).Platform, onChanged: w.onChanged,
		semanticFormatterCallback: w.semanticFormatterCallback, onChangeStart: w.onChangeStart, onChangeEnd: w.onChangeEnd,
		state: w.state, textDirection: widgets.DirectionalityOf(bc), hovering: w.hovering,
		gestures: w.gestures, gestureSettings: w.gestureSettings,
	}
	rendering.InitRenderBox(&r.RenderBox, r)
	r.startLabelPainter = painting.NewTextPainter(painting.TextPainterOpts{TextEngine: w.textEngine})
	r.endLabelPainter = painting.NewTextPainter(painting.TextPainterOpts{TextEngine: w.textEngine})
	r.updateLabelPainters()
	r.overlayAnimation = animation.NewCurvedAnimation(w.state.overlayController, animation.Curves.FastOutSlowIn)
	r.valueIndicatorAnimation = animation.NewCurvedAnimation(w.state.valueIndicatorController, animation.Curves.FastOutSlowIn)
	r.enableAnimation = animation.NewCurvedAnimation(w.state.enableController, animation.Curves.EaseInOut)
	return r
}

func (w *rangeSliderRenderObjectWidget) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderRangeSlider)
	r.setDivisions(w.divisions)
	r.setValues(w.values)
	r.setLabels(w.labels)
	r.setSliderTheme(w.sliderTheme)
	r.setTextScaleFactor(w.textScaleFactor)
	r.setScreenSize(w.screenSize)
	r.setOnChanged(w.onChanged)
	r.onChangeStart = w.onChangeStart
	r.onChangeEnd = w.onChangeEnd
	r.setTextDirection(widgets.DirectionalityOf(bc))
	r.setSemanticFormatterCallback(w.semanticFormatterCallback)
	r.setPlatform(ThemeOf(bc).Platform)
	r.setHovering(w.hovering)
	r.setGestureSettings(w.gestureSettings)
}

func (w *rangeSliderRenderObjectWidget) DidUnmountRenderObject(rendering.RenderObject) {}

// renderRangeSlider is the render object of a range slider.
//
// Flutter: material/range_slider.dart _RenderRangeSlider
type renderRangeSlider struct {
	rendering.RenderBox

	state                   *rangeSliderState
	overlayAnimation        *animation.CurvedAnimation
	valueIndicatorAnimation *animation.CurvedAnimation
	enableAnimation         *animation.CurvedAnimation
	startLabelPainter       *painting.TextPainter
	endLabelPainter         *painting.TextPainter
	gestures                *gestures.GestureBinding
	gestureSettings         gestures.DeviceGestureSettings
	drag                    *gestures.HorizontalDragGestureRecognizer
	tap                     *gestures.TapGestureRecognizer
	active                  bool
	newValues               RangeValues
	startThumbCenter        ui.Offset
	endThumbCenter          ui.Offset
	overlayStartRect        foundation.Opt[ui.Rect]
	overlayEndRect          foundation.Opt[ui.Rect]
	lastThumbSelection      foundation.Opt[Thumb]

	values                    RangeValues
	platform                  foundation.TargetPlatform
	semanticFormatterCallback SemanticFormatterCallback
	divisions                 foundation.Opt[int]
	labels                    foundation.Opt[RangeLabels]
	sliderTheme               SliderThemeData
	textScaleFactor           float64
	screenSize                ui.Size
	onChanged                 func(RangeValues)
	onChangeStart             func(RangeValues)
	onChangeEnd               func(RangeValues)
	textDirection             ui.TextDirection
	hovering                  bool
	hoveringStartThumb        bool
	hoveringEndThumb          bool

	startSemanticsNode *semantics.SemanticsNode
	endSemanticsNode   *semantics.SemanticsNode

	handles [9]foundation.ListenerHandle
}

func (r *renderRangeSlider) isEnabled() bool  { return r.onChanged != nil }
func (r *renderRangeSlider) isDiscrete() bool { return r.divisions.IsSome() && r.divisions.Value() > 0 }

func (r *renderRangeSlider) thumbSizeHeight() float64 {
	return r.sliderTheme.RangeThumbShape.GetPreferredSize(r.isEnabled(), r.isDiscrete()).Height
}

func (r *renderRangeSlider) overlayHeight() float64 {
	return r.sliderTheme.OverlayShape.GetPreferredSize(r.isEnabled(), r.isDiscrete()).Height
}

func (r *renderRangeSlider) sliderPartSizes() []ui.Size {
	partHeight := r.overlayHeight()
	if r.sliderTheme.Padding != nil {
		partHeight = r.thumbSizeHeight()
	}
	return []ui.Size{
		{Width: r.sliderTheme.OverlayShape.GetPreferredSize(r.isEnabled(), r.isDiscrete()).Width, Height: partHeight},
		r.sliderTheme.RangeThumbShape.GetPreferredSize(r.isEnabled(), r.isDiscrete()),
		r.sliderTheme.RangeTickMarkShape.GetPreferredSize(r.sliderTheme, r.isEnabled()),
	}
}

func (r *renderRangeSlider) maxSliderPartWidth() float64 {
	w := 0.0
	for i, s := range r.sliderPartSizes() {
		if i == 0 || s.Width > w {
			w = s.Width
		}
	}
	return w
}

func (r *renderRangeSlider) maxSliderPartHeight() float64 {
	h := 0.0
	for i, s := range r.sliderPartSizes() {
		if i == 0 || s.Height > h {
			h = s.Height
		}
	}
	return h
}

func (r *renderRangeSlider) minPreferredTrackHeight() float64 {
	return r.sliderTheme.TrackHeight.Value()
}

func (r *renderRangeSlider) trackRect() ui.Rect {
	return r.sliderTheme.RangeTrackShape.GetPreferredRect(SliderTrackRectOpts{ParentBox: r, SliderTheme: r.sliderTheme})
}

func (r *renderRangeSlider) minThumbSeparationValue() float64 {
	if r.isDiscrete() {
		return 0
	}
	return r.sliderTheme.MinThumbSeparation.Value() / r.trackRect().Width()
}

func (r *renderRangeSlider) thumbSize() ui.Size {
	return r.sliderTheme.RangeThumbShape.GetPreferredSize(r.isEnabled(), r.isDiscrete())
}

func (r *renderRangeSlider) setValues(newValues RangeValues) {
	converted := newValues
	if r.isDiscrete() {
		converted = r.discretizeRangeValues(newValues)
	}
	if converted == r.values {
		return
	}
	r.values = converted
	animate := func(c *animation.AnimationController, target float64) {
		distance := math.Abs(target - c.Value())
		if r.isDiscrete() {
			if distance != 0.0 {
				c.Duration = foundation.Some(time.Duration(float64(sliderPositionAnimationDuration) * (1.0 / distance)))
			} else {
				c.Duration = foundation.Some(time.Duration(0))
			}
			c.AnimateTo(target, animation.AnimateOpts{Curve: animation.Curves.EaseInOut})
		} else {
			c.SetValue(target)
		}
	}
	animate(r.state.startPositionController, r.values.Start)
	animate(r.state.endPositionController, r.values.End)
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderRangeSlider) setPlatform(v foundation.TargetPlatform) {
	if r.platform == v {
		return
	}
	r.platform = v
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderRangeSlider) setGestureSettings(v gestures.DeviceGestureSettings) {
	r.gestureSettings = v
	if r.drag != nil {
		r.drag.GestureSettings = foundation.Some(v)
		r.tap.GestureSettings = foundation.Some(v)
	}
}

func (r *renderRangeSlider) setSemanticFormatterCallback(v SemanticFormatterCallback) {
	if sameFunc(r.semanticFormatterCallback, v) {
		return
	}
	r.semanticFormatterCallback = v
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderRangeSlider) setDivisions(v foundation.Opt[int]) {
	if v == r.divisions {
		return
	}
	r.divisions = v
	r.MarkNeedsPaint()
}

func (r *renderRangeSlider) setLabels(v foundation.Opt[RangeLabels]) {
	if v == r.labels {
		return
	}
	r.labels = v
	r.updateLabelPainters()
}

func (r *renderRangeSlider) setSliderTheme(v SliderThemeData) {
	if v.Equal(r.sliderTheme) {
		return
	}
	r.sliderTheme = v
	r.MarkNeedsPaint()
}

func (r *renderRangeSlider) setTextScaleFactor(v float64) {
	if v == r.textScaleFactor {
		return
	}
	r.textScaleFactor = v
	r.updateLabelPainters()
}

func (r *renderRangeSlider) setScreenSize(v ui.Size) {
	if v == r.screenSize {
		return
	}
	r.screenSize = v
	r.MarkNeedsPaint()
}

func (r *renderRangeSlider) setOnChanged(v func(RangeValues)) {
	if (v == nil) == (r.onChanged == nil) {
		r.onChanged = v
		return
	}
	r.onChanged = v
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderRangeSlider) setTextDirection(v ui.TextDirection) {
	if v == r.textDirection {
		return
	}
	r.textDirection = v
	r.updateLabelPainters()
}

func (r *renderRangeSlider) setHovering(v bool) {
	if v == r.hovering {
		return
	}
	r.hovering = v
	r.updateForHover(v)
}

func (r *renderRangeSlider) setHoveringStartThumb(v bool) {
	if v == r.hoveringStartThumb {
		return
	}
	r.hoveringStartThumb = v
	r.updateForHover(r.hovering)
}

func (r *renderRangeSlider) setHoveringEndThumb(v bool) {
	if v == r.hoveringEndThumb {
		return
	}
	r.hoveringEndThumb = v
	r.updateForHover(r.hovering)
}

func (r *renderRangeSlider) updateForHover(hovered bool) {
	if hovered && (r.hoveringStartThumb || r.hoveringEndThumb) {
		r.state.overlayController.Forward()
	} else {
		r.state.overlayController.Reverse()
	}
}

func (r *renderRangeSlider) shouldAlwaysShowValueIndicator() bool {
	return r.sliderTheme.ShowValueIndicator == ShowValueIndicatorAlwaysVisible
}

func (r *renderRangeSlider) shouldShowValueIndicatorWhenDragged() bool {
	switch r.sliderTheme.ShowValueIndicator {
	case ShowValueIndicatorOnlyForDiscrete:
		return r.isDiscrete()
	case ShowValueIndicatorOnlyForContinuous:
		return !r.isDiscrete()
	case ShowValueIndicatorAlwaysVisible, ShowValueIndicatorAlways, ShowValueIndicatorOnDrag:
		return true
	}
	return false
}

func (r *renderRangeSlider) adjustmentUnit() float64 {
	if r.platform == foundation.TargetPlatformIOS {
		return 0.1
	}
	return 0.05
}

func (r *renderRangeSlider) updateLabelPainters() {
	r.updateLabelPainter(ThumbStart)
	r.updateLabelPainter(ThumbEnd)
}

func (r *renderRangeSlider) updateLabelPainter(thumb Thumb) {
	labels, ok := r.labels.Get()
	if !ok {
		return
	}
	text, painter := labels.Start, r.startLabelPainter
	if thumb == ThumbEnd {
		text, painter = labels.End, r.endLabelPainter
	}
	painter.SetText(painting.NewTextSpan(painting.TextSpanOpts{Style: r.sliderTheme.ValueIndicatorTextStyle, Text: foundation.Some(text)}))
	painter.SetTextDirection(r.textDirection)
	painter.SetTextScaler(painting.NewTextScalerLinear(r.textScaleFactor))
	painter.Layout(0, math.Inf(1))
	r.MarkNeedsLayout()
}

// SystemFontsDidChange lays the labels out again.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.systemFontsDidChange
func (r *renderRangeSlider) SystemFontsDidChange() {
	r.startLabelPainter.MarkNeedsLayout()
	r.endLabelPainter.MarkNeedsLayout()
	r.updateLabelPainters()
}

// Attach creates the recognizers and listens to the animations and the focus.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.attach
func (r *renderRangeSlider) Attach(owner *rendering.PipelineOwner) {
	r.RenderBox.Attach(owner)
	team := gestures.NewGestureArenaTeam(r.gestures.GestureArena)
	r.drag = gestures.NewHorizontalDragGestureRecognizer(r.gestures, gestures.DragGestureRecognizerOpts{DebugOwner: r})
	r.drag.SetTeam(team)
	r.drag.OnStart = r.handleDragStart
	r.drag.OnUpdate = r.handleDragUpdate
	r.drag.OnEnd = r.handleDragEnd
	r.drag.OnCancel = r.handleDragCancel
	r.drag.GestureSettings = foundation.Some(r.gestureSettings)
	r.tap = gestures.NewTapGestureRecognizer(r.gestures, gestures.TapGestureRecognizerOpts{DebugOwner: r})
	r.tap.SetTeam(team)
	r.tap.OnTapDown = r.handleTapDown
	r.tap.OnTapUp = r.handleTapUp
	r.tap.GestureSettings = foundation.Some(r.gestureSettings)
	r.handles[0] = r.overlayAnimation.AddListener(r.MarkNeedsPaint)
	r.handles[1] = r.valueIndicatorAnimation.AddListener(r.MarkNeedsPaint)
	r.handles[2] = r.enableAnimation.AddListener(r.MarkNeedsPaint)
	r.handles[3] = r.state.startPositionController.AddListener(r.MarkNeedsPaint)
	r.handles[4] = r.state.endPositionController.AddListener(r.MarkNeedsPaint)
	r.handles[5] = r.state.startFocusNode.AddListener(r.focusChanged)
	r.handles[6] = r.state.endFocusNode.AddListener(r.focusChanged)
}

func (r *renderRangeSlider) focusChanged() {
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

// Detach disposes the recognizers and stops listening.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.detach
func (r *renderRangeSlider) Detach() {
	r.overlayAnimation.RemoveListener(r.handles[0])
	r.valueIndicatorAnimation.RemoveListener(r.handles[1])
	r.enableAnimation.RemoveListener(r.handles[2])
	r.state.startPositionController.RemoveListener(r.handles[3])
	r.state.endPositionController.RemoveListener(r.handles[4])
	r.state.startFocusNode.RemoveListener(r.handles[5])
	r.state.endFocusNode.RemoveListener(r.handles[6])
	if r.drag != nil {
		r.drag.Dispose()
		r.tap.Dispose()
		r.drag, r.tap = nil, nil
	}
	r.RenderBox.Detach()
}

// Dispose releases the animations and the label painters.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.dispose
func (r *renderRangeSlider) Dispose() {
	r.startLabelPainter.Dispose()
	r.endLabelPainter.Dispose()
	r.enableAnimation.Dispose()
	r.valueIndicatorAnimation.Dispose()
	r.overlayAnimation.Dispose()
	r.RenderBox.Dispose()
}

func (r *renderRangeSlider) getValueFromVisualPosition(visualPosition float64) float64 {
	if r.textDirection == ui.TextDirectionRTL {
		return 1.0 - visualPosition
	}
	return visualPosition
}

func (r *renderRangeSlider) getValueFromGlobalPosition(globalPosition ui.Offset) float64 {
	trackRect := r.trackRect()
	visualPosition := (r.GlobalToLocal(globalPosition, nil).Dx - trackRect.Left) / trackRect.Width()
	return r.getValueFromVisualPosition(visualPosition)
}

func (r *renderRangeSlider) discretize(value float64) float64 {
	result := ui.ClampDouble(value, 0.0, 1.0)
	if r.isDiscrete() {
		d := float64(r.divisions.Value())
		result = math.Round(result*d) / d
	}
	return result
}

func (r *renderRangeSlider) discretizeRangeValues(values RangeValues) RangeValues {
	return RangeValues{Start: r.discretize(values.Start), End: r.discretize(values.End)}
}

func (r *renderRangeSlider) startInteraction(globalPosition ui.Offset) {
	if r.active {
		return
	}
	tapValue := ui.ClampDouble(r.getValueFromGlobalPosition(globalPosition), 0.0, 1.0)
	r.lastThumbSelection = r.sliderTheme.ThumbSelector(r.textDirection, r.values, tapValue, r.thumbSize(), r.Size(), 0)
	thumb, ok := r.lastThumbSelection.Get()
	if !ok {
		return
	}
	switch thumb {
	case ThumbStart:
		r.state.startFocusNode.RequestFocus()
	case ThumbEnd:
		r.state.endFocusNode.RequestFocus()
	}
	r.active = true
	currentValues := r.discretizeRangeValues(r.values)
	if thumb == ThumbStart {
		r.newValues = RangeValues{Start: tapValue, End: currentValues.End}
	} else {
		r.newValues = RangeValues{Start: currentValues.Start, End: tapValue}
	}
	r.updateLabelPainter(thumb)
	if r.onChangeStart != nil {
		r.onChangeStart(currentValues)
	}
	r.onChanged(r.discretizeRangeValues(r.newValues))
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

func (r *renderRangeSlider) handleDragUpdate(details gestures.DragUpdateDetails) {
	if !r.state.Mounted() {
		return
	}
	dragValue := r.getValueFromGlobalPosition(details.GlobalPosition)
	shouldCallOnChangeStart := false
	if r.lastThumbSelection.IsNone() {
		r.lastThumbSelection = r.sliderTheme.ThumbSelector(r.textDirection, r.values, dragValue, r.thumbSize(), r.Size(), details.Delta.Dx)
		if r.lastThumbSelection.IsSome() {
			shouldCallOnChangeStart = true
			r.active = true
			r.state.overlayController.Forward()
			if r.shouldShowValueIndicatorWhenDragged() {
				r.state.valueIndicatorController.Forward()
			}
		}
	}
	if thumb, ok := r.lastThumbSelection.Get(); ok && r.isEnabled() {
		currentValues := r.discretizeRangeValues(r.values)
		if r.onChangeStart != nil && shouldCallOnChangeStart {
			r.onChangeStart(currentValues)
		}
		currentDragValue := r.discretize(dragValue)
		if thumb == ThumbStart {
			r.newValues = RangeValues{Start: math.Min(currentDragValue, currentValues.End-r.minThumbSeparationValue()), End: currentValues.End}
		} else {
			r.newValues = RangeValues{Start: currentValues.Start, End: math.Max(currentDragValue, currentValues.Start+r.minThumbSeparationValue())}
		}
		r.onChanged(r.discretizeRangeValues(r.newValues))
	}
}

func (r *renderRangeSlider) endInteraction() {
	if !r.state.Mounted() {
		return
	}
	if r.shouldShowValueIndicatorWhenDragged() && r.state.interactionTimer == nil {
		r.state.valueIndicatorController.Reverse()
	}
	if r.active && r.state.Mounted() && r.lastThumbSelection.IsSome() {
		discreteValues := r.discretizeRangeValues(r.newValues)
		if r.onChangeEnd != nil {
			r.onChangeEnd(discreteValues)
		}
		r.active = false
	}
	r.state.overlayController.Reverse()
}

func (r *renderRangeSlider) handleDragStart(details gestures.DragStartDetails) {
	r.startInteraction(details.GlobalPosition)
}
func (r *renderRangeSlider) handleDragEnd(gestures.DragEndDetails) { r.endInteraction() }
func (r *renderRangeSlider) handleDragCancel()                     { r.endInteraction() }
func (r *renderRangeSlider) handleTapDown(details gestures.TapDownDetails) {
	r.startInteraction(details.GlobalPosition)
}
func (r *renderRangeSlider) handleTapUp(gestures.TapUpDetails) { r.endInteraction() }

// HitTestSelf is true: the whole box takes the presses.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.hitTestSelf
func (r *renderRangeSlider) HitTestSelf(ui.Offset) bool { return true }

// HandleEvent hands a pointer-down to the recognizers, and tracks the hover over
// the thumbs.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.handleEvent
func (r *renderRangeSlider) HandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) {
	if down, ok := event.(*gestures.PointerDownEvent); ok && r.isEnabled() && r.drag != nil {
		r.drag.AddPointer(down)
		r.tap.AddPointer(down)
	}
	if r.isEnabled() {
		if rect, ok := r.overlayStartRect.Get(); ok {
			r.setHoveringStartThumb(rect.Contains(event.LocalPosition()))
		}
		if rect, ok := r.overlayEndRect.Get(); ok {
			r.setHoveringEndThumb(rect.Contains(event.LocalPosition()))
		}
	}
}

// ComputeMinIntrinsicWidth is the preferred track width and the widest part.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.computeMinIntrinsicWidth
func (r *renderRangeSlider) ComputeMinIntrinsicWidth(float64) float64 {
	return sliderMinPreferredTrackWidth + r.maxSliderPartWidth()
}

// ComputeMaxIntrinsicWidth is the preferred track width and the widest part.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.computeMaxIntrinsicWidth
func (r *renderRangeSlider) ComputeMaxIntrinsicWidth(float64) float64 {
	return sliderMinPreferredTrackWidth + r.maxSliderPartWidth()
}

// ComputeMinIntrinsicHeight is the taller of the track and the parts.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.computeMinIntrinsicHeight
func (r *renderRangeSlider) ComputeMinIntrinsicHeight(float64) float64 {
	return math.Max(r.minPreferredTrackHeight(), r.maxSliderPartHeight())
}

// ComputeMaxIntrinsicHeight is the taller of the track and the parts.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.computeMaxIntrinsicHeight
func (r *renderRangeSlider) ComputeMaxIntrinsicHeight(float64) float64 {
	return math.Max(r.minPreferredTrackHeight(), r.maxSliderPartHeight())
}

// SizedByParent is true.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.sizedByParent
func (r *renderRangeSlider) SizedByParent() bool { return true }

// ComputeDryLayout is the biggest size, or the preferred one when unbounded.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.computeDryLayout
func (r *renderRangeSlider) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
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
// Flutter: material/range_slider.dart _RenderRangeSlider (sizedByParent)
func (r *renderRangeSlider) PerformLayout() {}

func (r *renderRangeSlider) overlayPaintOpts(activation animation.Animation[float64], painter *painting.TextPainter, value float64, sizeWithOverflow ui.Size) SliderComponentPaintOpts {
	return SliderComponentPaintOpts{
		ActivationAnimation: activation, EnableAnimation: r.enableAnimation, IsDiscrete: r.isDiscrete(), LabelPainter: painter,
		ParentBox: r, SliderTheme: r.sliderTheme, TextDirection: r.textDirection, Value: value, TextScaleFactor: r.textScaleFactor,
		SizeWithOverflow: sizeWithOverflow}
}

// Paint paints the track, the overlays, the tick marks and the thumbs, and records
// the painting of the value indicators.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.paint
func (r *renderRangeSlider) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	startValue := r.state.startPositionController.Value()
	endValue := r.state.endPositionController.Value()
	startVisualPosition, endVisualPosition := startValue, endValue
	if r.textDirection == ui.TextDirectionRTL {
		startVisualPosition, endVisualPosition = 1.0-startValue, 1.0-endValue
	}
	st := r.sliderTheme
	trackRect := st.RangeTrackShape.GetPreferredRect(SliderTrackRectOpts{ParentBox: r, Offset: offset, SliderTheme: st, IsDiscrete: r.isDiscrete()})
	padding := 0.0
	if st.RangeTrackShape.IsRounded() {
		padding = trackRect.Height()
	}
	thumbYOffset := trackRect.Center().Dy
	position := func(visual float64) float64 {
		if r.isDiscrete() {
			return trackRect.Left + visual*(trackRect.Width()-padding) + padding/2
		}
		return trackRect.Left + visual*trackRect.Width()
	}
	thumbPreferredSize := st.RangeThumbShape.GetPreferredSize(r.isEnabled(), r.isDiscrete())
	thumbPadding := 0.0
	if padding > thumbPreferredSize.Width/2 {
		thumbPadding = padding / 2
	}
	r.startThumbCenter = ui.Offset{Dx: ui.ClampDouble(position(startVisualPosition), trackRect.Left+thumbPadding, trackRect.Right-thumbPadding), Dy: thumbYOffset}
	r.endThumbCenter = ui.Offset{Dx: ui.ClampDouble(position(endVisualPosition), trackRect.Left+thumbPadding, trackRect.Right-thumbPadding), Dy: thumbYOffset}
	if r.isEnabled() {
		overlaySize := st.OverlayShape.GetPreferredSize(r.isEnabled(), false)
		r.overlayStartRect = foundation.Some(ui.NewRectFromCircle(r.startThumbCenter, overlaySize.Width/2.0))
		r.overlayEndRect = foundation.Some(ui.NewRectFromCircle(r.endThumbCenter, overlaySize.Width/2.0))
	}

	thumbSizeAt := func(states widgets.WidgetStates) foundation.Opt[ui.Size] {
		if st.ThumbSize == nil {
			return foundation.None[ui.Size]()
		}
		return st.ThumbSize.Resolve(states)
	}
	var thumbWidth, thumbHeight foundation.Opt[float64]
	if sz, ok := thumbSizeAt(0).Get(); ok {
		thumbWidth, thumbHeight = foundation.Some(sz.Width), foundation.Some(sz.Height)
	}
	trackGap := st.TrackGap
	var pressedThumbWidth foundation.Opt[float64]
	if sz, ok := thumbSizeAt(widgets.WidgetStates(0).With(widgets.WidgetStatePressed)).Get(); ok {
		pressedThumbWidth = foundation.Some(sz.Width)
	}
	if r.active && thumbWidth.IsSome() && pressedThumbWidth.IsSome() && trackGap.IsSome() {
		delta := thumbWidth.Value() - pressedThumbWidth.Value()
		thumbWidth = pressedThumbWidth
		if trackGap.Value() > 0.0 {
			trackGap = foundation.Some(trackGap.Value() - delta/2)
		}
	}

	trackTheme := st
	trackTheme.TrackGap = trackGap
	st.RangeTrackShape.Paint(context, offset, RangeSliderTrackPaintOpts{
		ParentBox: r, SliderTheme: trackTheme, EnableAnimation: r.enableAnimation, StartThumbCenter: r.startThumbCenter,
		EndThumbCenter: r.endThumbCenter, IsEnabled: r.isEnabled(), IsDiscrete: r.isDiscrete(), TextDirection: r.textDirection})

	startThumbSelected := r.lastThumbSelection == foundation.Some(ThumbStart) && !r.hoveringEndThumb
	endThumbSelected := r.lastThumbSelection == foundation.Some(ThumbEnd) && !r.hoveringStartThumb
	resolvedScreenSize := r.screenSize
	if resolvedScreenSize.IsEmpty() {
		resolvedScreenSize = r.Size()
	}

	always := animation.NewAlwaysStoppedAnimation(1.0)
	if r.state.startFocusNode.HasFocus() {
		st.OverlayShape.Paint(context, r.startThumbCenter, r.overlayPaintOpts(always, r.startLabelPainter, startValue, resolvedScreenSize))
	}
	if r.state.endFocusNode.HasFocus() {
		st.OverlayShape.Paint(context, r.endThumbCenter, r.overlayPaintOpts(always, r.endLabelPainter, endValue, resolvedScreenSize))
	}
	if r.overlayAnimation.Status() != animation.AnimationStatusDismissed {
		if startThumbSelected || r.hoveringStartThumb {
			st.OverlayShape.Paint(context, r.startThumbCenter, r.overlayPaintOpts(r.overlayAnimation, r.startLabelPainter, startValue, resolvedScreenSize))
		}
		if endThumbSelected || r.hoveringEndThumb {
			st.OverlayShape.Paint(context, r.endThumbCenter, r.overlayPaintOpts(r.overlayAnimation, r.endLabelPainter, endValue, resolvedScreenSize))
		}
	}

	if r.isDiscrete() {
		divisions := float64(r.divisions.Value())
		tickMarkWidth := st.RangeTickMarkShape.GetPreferredSize(st, r.isEnabled()).Width
		discreteTrackPadding := trackRect.Height()
		adjustedTrackWidth := trackRect.Width() - discreteTrackPadding
		if adjustedTrackWidth/divisions >= 3.0*tickMarkWidth {
			dy := trackRect.Center().Dy
			for i := 0; i <= r.divisions.Value(); i++ {
				value := float64(i) / divisions
				dx := trackRect.Left + value*adjustedTrackWidth + discreteTrackPadding/2
				st.RangeTickMarkShape.Paint(context, ui.Offset{Dx: dx, Dy: dy}, RangeSliderTickMarkPaintOpts{
					ParentBox: r, SliderTheme: st, EnableAnimation: r.enableAnimation, TextDirection: r.textDirection,
					StartThumbCenter: r.startThumbCenter, EndThumbCenter: r.endThumbCenter, IsEnabled: r.isEnabled()})
			}
		}
	}

	thumbDelta := math.Abs(r.endThumbCenter.Dx - r.startThumbCenter.Dx)
	isLastThumbStart := r.lastThumbSelection == foundation.Some(ThumbStart)
	bottomThumb, topThumb := ThumbStart, ThumbEnd
	bottomThumbCenter, topThumbCenter := r.startThumbCenter, r.endThumbCenter
	bottomLabelPainter, topLabelPainter := r.startLabelPainter, r.endLabelPainter
	bottomValue, topValue := startValue, endValue
	if isLastThumbStart {
		bottomThumb, topThumb = ThumbEnd, ThumbStart
		bottomThumbCenter, topThumbCenter = r.endThumbCenter, r.startThumbCenter
		bottomLabelPainter, topLabelPainter = r.endLabelPainter, r.startLabelPainter
		bottomValue, topValue = endValue, startValue
	}
	shouldPaintValueIndicators := r.isEnabled() && r.labels.IsSome() &&
		((r.shouldShowValueIndicatorWhenDragged() && r.valueIndicatorAnimation.Status() != animation.AnimationStatusDismissed) || r.shouldAlwaysShowValueIndicator())

	indicatorOpts := func(isOnTop bool, painter *painting.TextPainter, thumb Thumb, value float64) RangeSliderValueIndicatorPaintOpts {
		var activation, enable animation.Animation[float64] = r.valueIndicatorAnimation, r.enableAnimation
		if r.shouldAlwaysShowValueIndicator() {
			activation, enable = animation.NewAlwaysStoppedAnimation(1.0), animation.NewAlwaysStoppedAnimation(1.0)
		}
		return RangeSliderValueIndicatorPaintOpts{
			ActivationAnimation: activation, EnableAnimation: enable, IsDiscrete: r.isDiscrete(), IsOnTop: isOnTop,
			LabelPainter: painter, ParentBox: r, SliderTheme: st, TextDirection: r.textDirection, Thumb: thumb, Value: value,
			TextScaleFactor: r.textScaleFactor, SizeWithOverflow: resolvedScreenSize}
	}
	if shouldPaintValueIndicators {
		r.state.paintBottomValueIndicator = func(context *rendering.PaintingContext, _ ui.Offset) {
			if r.Attached() {
				st.RangeValueIndicatorShape.Paint(context, bottomThumbCenter, indicatorOpts(false, bottomLabelPainter, bottomThumb, bottomValue))
			}
		}
	}

	thumbTheme := st
	if tw, ok := thumbWidth.Get(); ok && thumbHeight.IsSome() {
		thumbTheme.ThumbSize = widgets.NewWidgetStatePropertyAll(foundation.Some(ui.Size{Width: tw, Height: thumbHeight.Value()}))
	}
	isPressed := func(thumb Thumb) bool {
		if thumb == ThumbStart {
			return startThumbSelected
		}
		return endThumbSelected
	}
	st.RangeThumbShape.Paint(context, bottomThumbCenter, RangeSliderThumbPaintOpts{
		ActivationAnimation: r.valueIndicatorAnimation, EnableAnimation: r.enableAnimation, IsDiscrete: r.isDiscrete(),
		IsEnabled: r.isEnabled(), IsOnTop: false, TextDirection: r.textDirection, SliderTheme: thumbTheme, Thumb: bottomThumb,
		IsPressed: isPressed(bottomThumb)})

	if shouldPaintValueIndicators {
		shape := st.RangeValueIndicatorShape
		shiftOpts := func(center ui.Offset, painter *painting.TextPainter) RangeSliderHorizontalShiftOpts {
			return RangeSliderHorizontalShiftOpts{ParentBox: r, Center: center, LabelPainter: painter,
				ActivationAnimation: r.valueIndicatorAnimation, TextScaleFactor: r.textScaleFactor, SizeWithOverflow: resolvedScreenSize}
		}
		startOffset := shape.GetHorizontalShift(shiftOpts(r.startThumbCenter, r.startLabelPainter))
		endOffset := shape.GetHorizontalShift(shiftOpts(r.endThumbCenter, r.endLabelPainter))
		startHalfWidth := shape.GetPreferredSize(r.isEnabled(), r.isDiscrete(), r.startLabelPainter, r.textScaleFactor).Width / 2
		endHalfWidth := shape.GetPreferredSize(r.isEnabled(), r.isDiscrete(), r.endLabelPainter, r.textScaleFactor).Width / 2
		innerOverflow := startHalfWidth + endHalfWidth
		if r.textDirection == ui.TextDirectionRTL {
			innerOverflow += endOffset - startOffset
		} else {
			innerOverflow += startOffset - endOffset
		}
		r.state.paintTopValueIndicator = func(context *rendering.PaintingContext, _ ui.Offset) {
			if r.Attached() {
				shape.Paint(context, topThumbCenter, indicatorOpts(thumbDelta < innerOverflow, topLabelPainter, topThumb, topValue))
			}
		}
	}

	st.RangeThumbShape.Paint(context, topThumbCenter, RangeSliderThumbPaintOpts{
		ActivationAnimation: r.overlayAnimation, EnableAnimation: r.enableAnimation, IsDiscrete: r.isDiscrete(),
		IsEnabled: r.isEnabled(), IsOnTop: thumbDelta < st.RangeThumbShape.GetPreferredSize(r.isEnabled(), r.isDiscrete()).Width,
		TextDirection: r.textDirection, SliderTheme: thumbTheme, Thumb: topThumb, IsPressed: isPressed(topThumb)})
}

func (r *renderRangeSlider) createSemanticsConfiguration(value, increasedValue, decreasedValue float64, increaseAction, decreaseAction func(), focused bool) *semantics.SemanticsConfiguration {
	config := semantics.NewSemanticsConfiguration()
	config.SetIsEnabled(foundation.Some(r.isEnabled()))
	config.SetTextDirection(r.textDirection)
	config.SetIsSlider(true)
	config.SetIsFocusable(true)
	config.SetIsFocused(foundation.Some(focused))
	if r.isEnabled() {
		config.SetOnIncrease(increaseAction)
		config.SetOnDecrease(decreaseAction)
	}
	if r.semanticFormatterCallback != nil {
		config.SetValue(r.semanticFormatterCallback(r.state.lerp(value)))
		config.SetIncreasedValue(r.semanticFormatterCallback(r.state.lerp(increasedValue)))
		config.SetDecreasedValue(r.semanticFormatterCallback(r.state.lerp(decreasedValue)))
	} else {
		pct := func(v float64) string { return strconv.Itoa(int(math.Round(v*100))) + "%" }
		config.SetValue(pct(value))
		config.SetIncreasedValue(pct(increasedValue))
		config.SetDecreasedValue(pct(decreasedValue))
	}
	return config
}

// AssembleSemanticsNode makes a node for each thumb.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.assembleSemanticsNode
func (r *renderRangeSlider) AssembleSemanticsNode(node *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, _ []*semantics.SemanticsNode) {
	startConfig := r.createSemanticsConfiguration(r.values.Start, r.increasedStartValue(), r.decreasedStartValue(),
		r.increaseStartAction, r.decreaseStartAction, r.state.startFocusNode.HasFocus())
	endConfig := r.createSemanticsConfiguration(r.values.End, r.increasedEndValue(), r.decreasedEndValue(),
		r.increaseEndAction, r.decreaseEndAction, r.state.endFocusNode.HasFocus())
	leftRect := ui.NewRectFromCenter(r.startThumbCenter, MinInteractiveDimension, MinInteractiveDimension)
	rightRect := ui.NewRectFromCenter(r.endThumbCenter, MinInteractiveDimension, MinInteractiveDimension)
	if r.startSemanticsNode == nil {
		r.startSemanticsNode = semantics.NewSemanticsNode()
	}
	if r.endSemanticsNode == nil {
		r.endSemanticsNode = semantics.NewSemanticsNode()
	}
	if r.textDirection == ui.TextDirectionRTL {
		r.startSemanticsNode.SetRect(rightRect)
		r.endSemanticsNode.SetRect(leftRect)
	} else {
		r.startSemanticsNode.SetRect(leftRect)
		r.endSemanticsNode.SetRect(rightRect)
	}
	r.startSemanticsNode.UpdateWith(startConfig, nil)
	r.endSemanticsNode.UpdateWith(endConfig, nil)
	node.UpdateWith(config, []*semantics.SemanticsNode{r.startSemanticsNode, r.endSemanticsNode})
}

// ClearSemantics drops the nodes of the thumbs.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.clearSemantics
func (r *renderRangeSlider) ClearSemantics() {
	r.RenderBox.ClearSemantics()
	r.startSemanticsNode = nil
	r.endSemanticsNode = nil
}

// DescribeSemanticsConfiguration makes the slider a semantic boundary.
//
// Flutter: material/range_slider.dart _RenderRangeSlider.describeSemanticsConfiguration
func (r *renderRangeSlider) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderBox.DescribeSemanticsConfiguration(config)
	config.SetIsSemanticBoundary(true)
}

func (r *renderRangeSlider) semanticActionUnit() float64 {
	if r.divisions.IsSome() {
		return 1.0 / float64(r.divisions.Value())
	}
	return r.adjustmentUnit()
}

func (r *renderRangeSlider) increaseStartAction() {
	if r.isEnabled() {
		r.onChanged(RangeValues{Start: r.increasedStartValue(), End: r.values.End})
	}
}

func (r *renderRangeSlider) decreaseStartAction() {
	if r.isEnabled() {
		r.onChanged(RangeValues{Start: r.decreasedStartValue(), End: r.values.End})
	}
}

func (r *renderRangeSlider) increaseEndAction() {
	if r.isEnabled() {
		r.onChanged(RangeValues{Start: r.values.Start, End: r.increasedEndValue()})
	}
}

func (r *renderRangeSlider) decreaseEndAction() {
	if r.isEnabled() {
		r.onChanged(RangeValues{Start: r.values.Start, End: r.decreasedEndValue()})
	}
}

func (r *renderRangeSlider) increasedStartValue() float64 {
	// Dart's double.parse(x.toStringAsFixed(2)).
	v, _ := strconv.ParseFloat(strconv.FormatFloat(r.values.Start+r.semanticActionUnit(), 'f', 2, 64), 64)
	if v <= r.values.End-r.minThumbSeparationValue() {
		return v
	}
	return r.values.Start
}

func (r *renderRangeSlider) decreasedStartValue() float64 {
	return ui.ClampDouble(r.values.Start-r.semanticActionUnit(), 0.0, 1.0)
}

func (r *renderRangeSlider) increasedEndValue() float64 {
	return ui.ClampDouble(r.values.End+r.semanticActionUnit(), 0.0, 1.0)
}

func (r *renderRangeSlider) decreasedEndValue() float64 {
	v := r.values.End - r.semanticActionUnit()
	if v >= r.values.Start+r.minThumbSeparationValue() {
		return v
	}
	return r.values.End
}

// rangeValueIndicatorRenderObjectWidget makes the render object that paints the
// value indicators in the overlay.
//
// Flutter: material/range_slider.dart _ValueIndicatorRenderObjectWidget
type rangeValueIndicatorRenderObjectWidget struct {
	widgets.LeafRenderObjectWidget
	state *rangeSliderState
}

func (w *rangeValueIndicatorRenderObjectWidget) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderRangeValueIndicator{state: w.state}
	rendering.InitRenderBox(&r.RenderBox, r)
	r.valueIndicatorAnimation = animation.NewCurvedAnimation(w.state.valueIndicatorController, animation.Curves.FastOutSlowIn)
	return r
}

func (w *rangeValueIndicatorRenderObjectWidget) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	ro.(*renderRangeValueIndicator).state = w.state
}

func (w *rangeValueIndicatorRenderObjectWidget) DidUnmountRenderObject(rendering.RenderObject) {}

// renderRangeValueIndicator paints the value indicators of a range slider.
//
// Flutter: material/range_slider.dart _RenderValueIndicator
type renderRangeValueIndicator struct {
	rendering.RenderBox
	state                   *rangeSliderState
	valueIndicatorAnimation *animation.CurvedAnimation
	handles                 [3]foundation.ListenerHandle
}

func (r *renderRangeValueIndicator) SizedByParent() bool { return true }

func (r *renderRangeValueIndicator) Attach(owner *rendering.PipelineOwner) {
	r.RenderBox.Attach(owner)
	r.handles[0] = r.valueIndicatorAnimation.AddListener(r.MarkNeedsPaint)
	r.handles[1] = r.state.startPositionController.AddListener(r.MarkNeedsPaint)
	r.handles[2] = r.state.endPositionController.AddListener(r.MarkNeedsPaint)
}

func (r *renderRangeValueIndicator) Detach() {
	r.valueIndicatorAnimation.RemoveListener(r.handles[0])
	r.state.startPositionController.RemoveListener(r.handles[1])
	r.state.endPositionController.RemoveListener(r.handles[2])
	r.RenderBox.Detach()
}

func (r *renderRangeValueIndicator) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if f := r.state.paintBottomValueIndicator; f != nil {
		f(context, offset)
	}
	if f := r.state.paintTopValueIndicator; f != nil {
		f(context, offset)
	}
}

func (r *renderRangeValueIndicator) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Smallest()
}

func (r *renderRangeValueIndicator) PerformLayout() {}

func (r *renderRangeValueIndicator) Dispose() {
	r.valueIndicatorAnimation.Dispose()
	r.RenderBox.Dispose()
}
