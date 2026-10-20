// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// IconAlignment is the side of the label that the icon of a button is on.
//
// Flutter: material/button_style_button.dart IconAlignment
type IconAlignment int

const (
	// IconAlignmentUnspecified is the zero value: the icon is at the start.
	IconAlignmentUnspecified IconAlignment = iota
	// IconAlignmentStart puts the icon before the label.
	//
	// Flutter: material/button_style_button.dart IconAlignment.start
	IconAlignmentStart
	// IconAlignmentEnd puts the icon after the label.
	//
	// Flutter: material/button_style_button.dart IconAlignment.end
	IconAlignmentEnd
)

// Name is the name of the value, as Dart's Enum.name.
func (a IconAlignment) Name() string { return enumName(int(a), "start", "end") }

// String is "IconAlignment." and the name.
func (a IconAlignment) String() string { return "IconAlignment." + a.Name() }

// ButtonStyleButtonOpts are the optional parameters that the buttons made of a
// ButtonStyle share: those of NewTextButton, NewElevatedButton, NewFilledButton
// and NewOutlinedButton.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.new
type ButtonStyleButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnLongPress is called when the button is long-pressed.
	OnLongPress func()
	// OnHover is called when a pointer enters or exits the button.
	OnHover func(value bool)
	// OnFocusChange is called when the focus of the button changes.
	OnFocusChange func(value bool)
	// Style customizes this button's appearance; it wins over the style of the
	// theme and over the default style.
	Style *ButtonStyle
	// FocusNode is the focus node of the button.
	FocusNode *widgets.FocusNode
	// Autofocus says that the button focuses itself when it is built, if nothing
	// else has the focus.
	Autofocus bool
	// ClipBehavior is how the content is clipped; the default is no clip, or
	// antiAlias when the style has a background or a foreground builder.
	ClipBehavior ui.Clip
	// StatesController is the controller of the widget states of the button.
	StatesController *widgets.WidgetStatesController
	// IsSemanticButton says whether the button is a semantics button; the
	// default is true.
	IsSemanticButton foundation.Opt[bool]
	// Tooltip is the text of the tooltip of the button.
	Tooltip string
}

// ButtonStyleButtonSpec is what a button made of a ButtonStyle (TextButton,
// ElevatedButton, FilledButton, OutlinedButton) has to give to the shared state
// of ButtonStyleButton: its base and the default style and the theme style.
//
// Flutter: material/button_style_button.dart ButtonStyleButton
type ButtonStyleButtonSpec interface {
	widgets.Widget
	// ButtonStyleButtonBase is the shared base of the button.
	ButtonStyleButtonBase() *ButtonStyleButton
	// DefaultStyleOf is the style that the button has when neither the button nor
	// the theme gave a property.
	DefaultStyleOf(bc widgets.BuildContext) *ButtonStyle
	// ThemeStyleOf is the style of the theme of the button, or nil.
	ThemeStyleOf(bc widgets.BuildContext) *ButtonStyle
}

// ButtonStyleButton is the base of the buttons that are made from a ButtonStyle:
// it resolves the properties of the style, from the widget, the theme and the
// defaults, and builds the Material, the ink and the tap target of the button.
//
// Flutter: material/button_style_button.dart ButtonStyleButton
type ButtonStyleButton struct {
	widgets.StatefulWidget
	onPressed func()
	child     widgets.Widget
	o         ButtonStyleButtonOpts
}

// NewButtonStyleButtonBase makes the base of a button: a nil onPressed (and no
// OnLongPress) disables the button. The child may be nil.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.new
func NewButtonStyleButtonBase(onPressed func(), child widgets.Widget, opts ...ButtonStyleButtonOpts) ButtonStyleButton {
	o := firstOpts(opts)
	return ButtonStyleButton{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed: onPressed, child: child, o: o}
}

// ButtonStyleButtonBase is the base itself.
//
// Flutter: no counterpart (the spec interface reaches the shared fields)
func (w *ButtonStyleButton) ButtonStyleButtonBase() *ButtonStyleButton { return w }

// OnPressed is called when the button is tapped or otherwise activated; nil
// (with no OnLongPress) disables the button.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.onPressed
func (w *ButtonStyleButton) OnPressed() func() { return w.onPressed }

// OnLongPress is called when the button is long-pressed.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.onLongPress
func (w *ButtonStyleButton) OnLongPress() func() { return w.o.OnLongPress }

// OnHover is called when a pointer enters or exits the button.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.onHover
func (w *ButtonStyleButton) OnHover() func(bool) { return w.o.OnHover }

// OnFocusChange is called when the focus of the button changes.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.onFocusChange
func (w *ButtonStyleButton) OnFocusChange() func(bool) { return w.o.OnFocusChange }

// Style customizes this button's appearance.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.style
func (w *ButtonStyleButton) Style() *ButtonStyle { return w.o.Style }

// ClipBehavior is how the content of the button is clipped.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.clipBehavior
func (w *ButtonStyleButton) ClipBehavior() ui.Clip { return w.o.ClipBehavior }

// FocusNode is the focus node of the button.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.focusNode
func (w *ButtonStyleButton) FocusNode() *widgets.FocusNode { return w.o.FocusNode }

// Autofocus says that the button focuses itself when it is built.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.autofocus
func (w *ButtonStyleButton) Autofocus() bool { return w.o.Autofocus }

// StatesController is the controller of the widget states of the button.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.statesController
func (w *ButtonStyleButton) StatesController() *widgets.WidgetStatesController {
	return w.o.StatesController
}

// IsSemanticButton says whether the button is a semantics button.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.isSemanticButton
func (w *ButtonStyleButton) IsSemanticButton() bool { return w.o.IsSemanticButton.OrElse(true) }

// Tooltip is the text of the tooltip of the button.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.tooltip
func (w *ButtonStyleButton) Tooltip() string { return w.o.Tooltip }

// Child is the widget in the button.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.child
func (w *ButtonStyleButton) Child() widgets.Widget { return w.child }

// Enabled says whether the button can be pressed.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.enabled
func (w *ButtonStyleButton) Enabled() bool { return w.onPressed != nil || w.o.OnLongPress != nil }

// CreateState creates the state.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.createState
func (w *ButtonStyleButton) CreateState() widgets.State { return &buttonStyleState{} }

// DebugFillProperties adds the flag of the enabled button and the style and the
// focus node.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.debugFillProperties
func (w *ButtonStyleButton) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	properties.Add(foundation.NewFlagProperty("enabled", foundation.Some(w.Enabled()), foundation.FlagPropertyOpts{IfFalse: "disabled"}))
	properties.Add(objectProp("style", nilIfZero(w.o.Style)))
	properties.Add(objectProp("focusNode", nilIfZero(w.o.FocusNode)))
}

// ButtonStyleButtonAllOrNull returns a property with the value in every state,
// or nil when there is no value.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.allOrNull
func ButtonStyleButtonAllOrNull[T any](value foundation.Opt[T]) widgets.WidgetStateProperty[foundation.Opt[T]] {
	if value.IsNone() {
		return nil
	}
	return widgets.NewWidgetStatePropertyAll(value)
}

// ButtonStyleButtonDefaultColor returns a color property that is the disabled
// color when the state is disabled and the enabled color otherwise, or nil when
// both colors are missing.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.defaultColor
func ButtonStyleButtonDefaultColor(enabled, disabled foundation.Opt[ui.Color]) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	if firstOpt(enabled, disabled).IsNone() {
		return nil
	}
	return widgets.NewWidgetStateMapper(widgets.WidgetStateMap[foundation.Opt[ui.Color]]{
		{Key: widgets.WidgetStateDisabled, Value: disabled},
		{Key: widgets.WidgetStateAny, Value: enabled},
	})
}

// ButtonStyleButtonScaledPadding returns the padding for the font size
// multiplier: the 1x padding up to a multiplier of 1, the interpolation to the 2x
// padding and then to the 3x padding up to a multiplier of 3, and the 3x padding
// after.
//
// Flutter: material/button_style_button.dart ButtonStyleButton.scaledPadding
func ButtonStyleButtonScaledPadding(geometry1x, geometry2x, geometry3x painting.EdgeInsetsGeometry, fontSizeMultiplier float64) painting.EdgeInsetsGeometry {
	switch {
	case fontSizeMultiplier <= 1:
		return geometry1x
	case fontSizeMultiplier < 2:
		return painting.EdgeInsetsGeometryLerp(geometry1x, geometry2x, fontSizeMultiplier-1)
	case fontSizeMultiplier < 3:
		return painting.EdgeInsetsGeometryLerp(geometry2x, geometry3x, fontSizeMultiplier-2)
	}
	return geometry3x
}

// buttonStyleOpt is the first value of the property that the styles give, where a
// style or a property that is missing, or a value that is none, gives way to the
// next style.
func buttonStyleOpt[T any](styles [3]*ButtonStyle, states widgets.WidgetStates, get func(*ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[T]]) foundation.Opt[T] {
	for _, st := range styles {
		if st == nil {
			continue
		}
		p := get(st)
		if p == nil {
			continue
		}
		if v := p.Resolve(states); v.IsSome() {
			return v
		}
	}
	return foundation.None[T]()
}

// buttonStyleNil is buttonStyleOpt for the properties whose value is nilable: the first
// value that is not nil.
func buttonStyleNil[T comparable](styles [3]*ButtonStyle, states widgets.WidgetStates, get func(*ButtonStyle) widgets.WidgetStateProperty[T]) T {
	var zero T
	for _, st := range styles {
		if st == nil {
			continue
		}
		p := get(st)
		if p == nil {
			continue
		}
		if v := p.Resolve(states); v != zero {
			return v
		}
	}
	return zero
}

// buttonStyleValue is the first value that the styles have of a property that is not
// a widget state property.
func buttonStyleValue[T any](styles [3]*ButtonStyle, get func(*ButtonStyle) foundation.Opt[T]) foundation.Opt[T] {
	for _, st := range styles {
		if st == nil {
			continue
		}
		if v := get(st); v.IsSome() {
			return v
		}
	}
	return foundation.None[T]()
}

type buttonStyleState struct {
	widgets.StateOf[ButtonStyleButtonSpec]
	widgets.TickerProvider
	controller               *animation.AnimationController
	elevation                foundation.Opt[float64]
	backgroundColor          foundation.Opt[ui.Color]
	internalStatesController *widgets.WidgetStatesController
	statesListener           foundation.ListenerHandle
}

func (s *buttonStyleState) base() *ButtonStyleButton { return s.Widget().ButtonStyleButtonBase() }

func (s *buttonStyleState) handleStatesControllerChange() {
	// Force a rebuild to resolve MaterialStateProperty properties.
	s.SetState(func() {})
}

func (s *buttonStyleState) statesController() *widgets.WidgetStatesController {
	if c := s.base().o.StatesController; c != nil {
		return c
	}
	return s.internalStatesController
}

func (s *buttonStyleState) initStatesController() {
	if s.base().o.StatesController == nil {
		s.internalStatesController = widgets.NewWidgetStatesController()
	}
	s.statesController().Update(widgets.WidgetStateDisabled, !s.base().Enabled())
	s.statesListener = s.statesController().AddListener(s.handleStatesControllerChange)
}

func (s *buttonStyleState) InitState() { s.initStatesController() }

func (s *buttonStyleState) DidUpdateWidget(old ButtonStyleButtonSpec) {
	w, o := s.base(), old.ButtonStyleButtonBase()
	if w.o.StatesController != o.o.StatesController {
		if o.o.StatesController != nil {
			o.o.StatesController.RemoveListener(s.statesListener)
		} else if s.internalStatesController != nil {
			s.internalStatesController.RemoveListener(s.statesListener)
		}
		if w.o.StatesController != nil && s.internalStatesController != nil {
			s.internalStatesController.Dispose()
			s.internalStatesController = nil
		}
		s.initStatesController()
	}
	if w.Enabled() != o.Enabled() {
		s.statesController().Update(widgets.WidgetStateDisabled, !w.Enabled())
		if !w.Enabled() {
			// The button may have been disabled while a press was in progress.
			s.statesController().Update(widgets.WidgetStatePressed, false)
		}
	}
}

func (s *buttonStyleState) Dispose() {
	s.statesController().RemoveListener(s.statesListener)
	if s.internalStatesController != nil {
		s.internalStatesController.Dispose()
	}
	if s.controller != nil {
		s.controller.Dispose()
	}
}

func (s *buttonStyleState) Build(bc widgets.BuildContext) widgets.Widget {
	spec := s.Widget()
	w := spec.ButtonStyleButtonBase()
	theme := ThemeOf(bc)
	iconTheme := widgets.IconThemeOf(bc)
	styles := [3]*ButtonStyle{w.o.Style, spec.ThemeStyleOf(bc), spec.DefaultStyleOf(bc)}
	states := s.statesController().Value()

	effectiveIconColor := func() foundation.Opt[ui.Color] {
		for _, st := range [...]*ButtonStyle{styles[0], styles[1]} {
			if st != nil && st.IconColor != nil {
				if c := st.IconColor.Resolve(states); c.IsSome() {
					return c
				}
			}
		}
		for _, st := range [...]*ButtonStyle{styles[0], styles[1]} {
			if st != nil && st.ForegroundColor != nil {
				if c := st.ForegroundColor.Resolve(states); c.IsSome() {
					return c
				}
			}
		}
		if st := styles[2]; st != nil {
			if st.IconColor != nil {
				if c := st.IconColor.Resolve(states); c.IsSome() {
					return c
				}
			}
			if st.ForegroundColor != nil {
				return st.ForegroundColor.Resolve(states)
			}
		}
		return foundation.None[ui.Color]()
	}

	resolvedElevation := buttonStyleOpt(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[float64]] { return st.Elevation })
	resolvedTextStyle := buttonStyleNil(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[*painting.TextStyle] { return st.TextStyle })
	resolvedBackgroundColor := buttonStyleOpt(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] { return st.BackgroundColor })
	resolvedForegroundColor := buttonStyleOpt(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] { return st.ForegroundColor })
	resolvedShadowColor := buttonStyleOpt(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] { return st.ShadowColor })
	resolvedSurfaceTintColor := buttonStyleOpt(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
		return st.SurfaceTintColor
	})
	resolvedPadding := buttonStyleNil(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[painting.EdgeInsetsGeometry] { return st.Padding })
	resolvedMinimumSize := buttonStyleOpt(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Size]] { return st.MinimumSize })
	resolvedFixedSize := buttonStyleOpt(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Size]] { return st.FixedSize })
	resolvedMaximumSize := buttonStyleOpt(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Size]] { return st.MaximumSize })
	resolvedIconColor := effectiveIconColor()
	resolvedIconSize := buttonStyleOpt(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[float64]] { return st.IconSize })
	resolvedSide := buttonStyleOpt(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]] { return st.Side })
	resolvedShape := buttonStyleNil(styles, states, func(st *ButtonStyle) widgets.WidgetStateProperty[painting.OutlinedBorder] { return st.Shape })

	mouseCursor := widgets.WidgetStateMouseCursorResolveWith(func(cursorStates widgets.WidgetStates) services.MouseCursor {
		if c := buttonStyleNil(styles, cursorStates, func(st *ButtonStyle) widgets.WidgetStateProperty[services.MouseCursor] { return st.MouseCursor }); c != nil {
			return c
		}
		return widgets.WidgetStateMouseCursorClickable.Resolve(cursorStates)
	}, "ButtonStyleButton_MouseCursor")

	overlayColor := widgets.WidgetStatePropertyResolveWith(func(overlayStates widgets.WidgetStates) foundation.Opt[ui.Color] {
		return buttonStyleOpt(styles, overlayStates, func(st *ButtonStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] { return st.OverlayColor })
	})

	resolvedVisualDensity := buttonStyleValue(styles, func(st *ButtonStyle) foundation.Opt[VisualDensity] { return st.VisualDensity }).OrElse(VisualDensity{})
	resolvedTapTargetSize := MaterialTapTargetSizeUnspecified
	for _, st := range styles {
		if st != nil && st.TapTargetSize != MaterialTapTargetSizeUnspecified {
			resolvedTapTargetSize = st.TapTargetSize
			break
		}
	}
	resolvedAnimationDuration := buttonStyleValue(styles, func(st *ButtonStyle) foundation.Opt[time.Duration] { return st.AnimationDuration }).OrElse(0)
	resolvedEnableFeedback := buttonStyleValue(styles, func(st *ButtonStyle) foundation.Opt[bool] { return st.EnableFeedback }).OrElse(true)
	var resolvedAlignment painting.AlignmentGeometry
	var resolvedSplashFactory InteractiveInkFeatureFactory
	var resolvedBackgroundBuilder, resolvedForegroundBuilder ButtonLayerBuilder
	for _, st := range styles {
		if st == nil {
			continue
		}
		if resolvedAlignment == nil {
			resolvedAlignment = st.Alignment
		}
		if resolvedSplashFactory == nil {
			resolvedSplashFactory = st.SplashFactory
		}
		if resolvedBackgroundBuilder == nil {
			resolvedBackgroundBuilder = st.BackgroundBuilder
		}
		if resolvedForegroundBuilder == nil {
			resolvedForegroundBuilder = st.ForegroundBuilder
		}
	}
	if resolvedAlignment == nil {
		resolvedAlignment = painting.AlignmentCenter
	}
	densityAdjustment := resolvedVisualDensity.BaseSizeAdjustment()

	effectiveClipBehavior := w.o.ClipBehavior
	if effectiveClipBehavior == ui.ClipUnspecified {
		effectiveClipBehavior = ui.ClipNone
		if resolvedBackgroundBuilder != nil || resolvedForegroundBuilder != nil {
			effectiveClipBehavior = ui.ClipAntiAlias
		}
	}

	minimumSize := resolvedMinimumSize.OrElse(ui.Size{})
	maximumSize := resolvedMaximumSize.OrElse(ui.Size{Width: math.Inf(1), Height: math.Inf(1)})
	effectiveConstraints := resolvedVisualDensity.EffectiveConstraints(rendering.BoxConstraints{
		MinWidth: minimumSize.Width, MinHeight: minimumSize.Height, MaxWidth: maximumSize.Width, MaxHeight: maximumSize.Height})
	if fixed, ok := resolvedFixedSize.Get(); ok {
		size := effectiveConstraints.Constrain(fixed)
		if !math.IsInf(size.Width, 0) && !math.IsNaN(size.Width) {
			effectiveConstraints = effectiveConstraints.CopyWith(rendering.BoxConstraintsCopy{MinWidth: foundation.Some(size.Width), MaxWidth: foundation.Some(size.Width)})
		}
		if !math.IsInf(size.Height, 0) && !math.IsNaN(size.Height) {
			effectiveConstraints = effectiveConstraints.CopyWith(rendering.BoxConstraintsCopy{MinHeight: foundation.Some(size.Height), MaxHeight: foundation.Some(size.Height)})
		}
	}

	dy := densityAdjustment.Dy
	dx := math.Max(0, densityAdjustment.Dx)
	if resolvedPadding == nil {
		resolvedPadding = painting.EdgeInsetsZero
	}
	padding := resolvedPadding.Add(painting.NewEdgeInsetsFromLTRB(dx, dy, dx, dy)).Clamp(painting.EdgeInsetsZero, painting.EdgeInsetsGeometryInfinity)

	// A button that loses its elevation and gets a translucent background keeps
	// the opaque background until its elevation has gone, so that the shadow does
	// not show through.
	if prev, ok := s.backgroundColor.Get(); ok && resolvedAnimationDuration > 0 && s.elevation.IsSome() {
		if bg, ok := resolvedBackgroundColor.Get(); ok && s.elevation != resolvedElevation && prev.ToARGB32() != bg.ToARGB32() &&
			prev.Opacity() == 1 && bg.Opacity() < 1 && resolvedElevation == foundation.Some(0.0) {
			if s.controller == nil || s.controller.Duration.OrElse(0) != resolvedAnimationDuration {
				if s.controller != nil {
					s.controller.Dispose()
				}
				s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(resolvedAnimationDuration)})
				s.controller.AddStatusListener(func(status animation.AnimationStatus) {
					if status == animation.AnimationStatusCompleted {
						s.SetState(func() {}) // Rebuild with the final background color.
					}
				})
			}
			resolvedBackgroundColor = s.backgroundColor // Defer changing the background color.
			s.controller.SetValue(0)
			s.controller.Forward()
		}
	}
	s.elevation = resolvedElevation
	s.backgroundColor = resolvedBackgroundColor

	content := w.child
	if resolvedForegroundBuilder != nil {
		content = resolvedForegroundBuilder(bc, states, w.child)
	}
	var result widgets.Widget = widgets.NewPadding(padding,
		widgets.NewAlign(content, widgets.AlignOpts{Alignment: resolvedAlignment, WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)}))
	if resolvedBackgroundBuilder != nil {
		result = resolvedBackgroundBuilder(bc, states, result)
	}

	// The shape of the button gets the side of the style.
	shape := resolvedShape
	if side, ok := resolvedSide.Get(); ok && shape != nil {
		shape = shape.CopyWithSide(side)
	}

	iconData := widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: resolvedIconSize, Color: resolvedIconColor})
	merged := iconTheme.Merge(&iconData)
	var inkShape painting.ShapeBorder
	if shape != nil {
		inkShape = shape
	}
	result = NewAnimatedTheme(theme.CopyWith(ThemeDataCopy{IconTheme: foundation.Some(merged)}),
		NewInkWell(result, InkWellOpts{
			OnTap:            w.onPressed,
			OnLongPress:      w.o.OnLongPress,
			OnHover:          w.o.OnHover,
			MouseCursor:      mouseCursor,
			EnableFeedback:   foundation.Some(resolvedEnableFeedback),
			FocusNode:        w.o.FocusNode,
			CanRequestFocus:  foundation.Some(w.Enabled()),
			OnFocusChange:    w.o.OnFocusChange,
			Autofocus:        w.o.Autofocus,
			SplashFactory:    resolvedSplashFactory,
			OverlayColor:     overlayColor,
			HighlightColor:   foundation.Some(Colors.Transparent),
			CustomBorder:     inkShape,
			StatesController: s.statesController(),
		}),
		AnimatedThemeOpts{Duration: foundation.Some(animationDurationOrTick(resolvedAnimationDuration))})

	if w.o.Tooltip != "" {
		result = NewTooltip(result, TooltipOpts{Message: w.o.Tooltip})
	}

	minSize := ui.Size{}
	if resolvedTapTargetSize == MaterialTapTargetSizePadded {
		minSize = ui.Size{Width: MinInteractiveDimension + densityAdjustment.Dx, Height: MinInteractiveDimension + densityAdjustment.Dy}
	}

	materialType := MaterialTypeButton
	if resolvedBackgroundColor.IsNone() {
		materialType = MaterialTypeTransparency
	}
	var materialShape painting.ShapeBorder
	if shape != nil {
		materialShape = shape
	}
	var textStyle *painting.TextStyle
	if resolvedTextStyle != nil {
		textStyle = resolvedTextStyle.CopyWith(painting.TextStyleCopy{Color: resolvedForegroundColor})
	}
	return widgets.NewSemantics(
		newInputPadding(minSize,
			widgets.NewConstrainedBox(effectiveConstraints,
				NewMaterial(result, MaterialOpts{
					Elevation:            resolvedElevation.OrElse(0),
					TextStyle:            textStyle,
					Shape:                materialShape,
					Color:                resolvedBackgroundColor,
					ShadowColor:          resolvedShadowColor,
					SurfaceTintColor:     resolvedSurfaceTintColor,
					Type:                 materialType,
					AnimationDuration:    animationDurationOrTick(resolvedAnimationDuration),
					ClipBehavior:         effectiveClipBehavior,
					NoBorderOnForeground: true,
				}))),
		widgets.SemanticsOpts{Container: true, Button: foundation.Some(w.IsSemanticButton()), Enabled: foundation.Some(w.Enabled())},
	)
}

// animationDurationOrTick is the duration for the widgets that treat a zero
// duration as their default: the shortest duration instead of zero.
func animationDurationOrTick(d time.Duration) time.Duration {
	if d <= 0 {
		return time.Nanosecond
	}
	return d
}

// inputPadding pads the child to the minimum size of the tap target, and
// forwards the taps in that padding to the child.
//
// Flutter: material/button_style_button.dart _InputPadding
type inputPadding struct {
	widgets.SingleChildRenderObjectWidget
	minSize ui.Size
}

func newInputPadding(minSize ui.Size, child widgets.Widget) *inputPadding {
	return &inputPadding{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child), minSize: minSize}
}

func (w *inputPadding) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderInputPadding{minSize: w.minSize}
	rendering.InitRenderShiftedBox(&r.RenderShiftedBox, r, nil)
	return r
}

func (w *inputPadding) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*renderInputPadding).setMinSize(w.minSize)
}

// Flutter: material/button_style_button.dart _RenderInputPadding
type renderInputPadding struct {
	rendering.RenderShiftedBox
	minSize ui.Size
}

func (r *renderInputPadding) setMinSize(value ui.Size) {
	if r.minSize == value {
		return
	}
	r.minSize = value
	r.MarkNeedsLayout()
}

func (r *renderInputPadding) ComputeMinIntrinsicWidth(height float64) float64 {
	if child := r.Child(); child != nil {
		return math.Max(child.GetMinIntrinsicWidth(height), r.minSize.Width)
	}
	return 0.0
}

func (r *renderInputPadding) ComputeMinIntrinsicHeight(width float64) float64 {
	if child := r.Child(); child != nil {
		return math.Max(child.GetMinIntrinsicHeight(width), r.minSize.Height)
	}
	return 0.0
}

func (r *renderInputPadding) ComputeMaxIntrinsicWidth(height float64) float64 {
	if child := r.Child(); child != nil {
		return math.Max(child.GetMaxIntrinsicWidth(height), r.minSize.Width)
	}
	return 0.0
}

func (r *renderInputPadding) ComputeMaxIntrinsicHeight(width float64) float64 {
	if child := r.Child(); child != nil {
		return math.Max(child.GetMaxIntrinsicHeight(width), r.minSize.Height)
	}
	return 0.0
}

func (r *renderInputPadding) computeSize(constraints rendering.BoxConstraints, layoutChild rendering.ChildLayouter) ui.Size {
	if child := r.Child(); child != nil {
		childSize := layoutChild(child, constraints)
		return constraints.Constrain(ui.Size{Width: math.Max(childSize.Width, r.minSize.Width), Height: math.Max(childSize.Height, r.minSize.Height)})
	}
	return ui.Size{}
}

func (r *renderInputPadding) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	return r.computeSize(constraints, rendering.ChildLayoutHelper.DryLayoutChild)
}

func (r *renderInputPadding) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	child := r.Child()
	if child == nil {
		return foundation.None[float64]()
	}
	result, ok := child.GetDryBaseline(constraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	childSize := child.GetDryLayout(constraints)
	size := r.GetDryLayout(constraints)
	return foundation.Some(result + painting.AlignmentCenter.AlongOffset(ui.Offset{Dx: size.Width - childSize.Width, Dy: size.Height - childSize.Height}).Dy)
}

func (r *renderInputPadding) PerformLayout() {
	r.SetSize(r.computeSize(r.BoxConstraints(), rendering.ChildLayoutHelper.LayoutChild))
	if child := r.Child(); child != nil {
		size, childSize := r.Size(), child.Size()
		rendering.BoxParentDataOf(child).Offset = painting.AlignmentCenter.AlongOffset(ui.Offset{Dx: size.Width - childSize.Width, Dy: size.Height - childSize.Height})
	}
}

func (r *renderInputPadding) HitTest(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	if r.RenderShiftedBox.HitTest(result, position) {
		return true
	}
	child := r.Child()
	center := child.Size().Center(ui.Offset{})
	return result.AddWithRawTransform(ui.NewMatrix4ForceToPoint(center), center, func(result *rendering.BoxHitTestResult, position ui.Offset) bool {
		return child.HitTest(result, center)
	})
}
