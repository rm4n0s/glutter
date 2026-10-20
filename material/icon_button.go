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
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// iconButtonMinButtonSize is the minimum size of the icon buttons of Material 2.
//
// Flutter: material/icon_button.dart _kMinButtonSize
const iconButtonMinButtonSize = MinInteractiveDimension

// iconButtonVariant is the kind of an IconButton.
//
// Flutter: material/icon_button.dart _IconButtonVariant
type iconButtonVariant int

const (
	iconButtonStandard iconButtonVariant = iota
	iconButtonFilled
	iconButtonFilledTonal
	iconButtonOutlined
)

// IconButtonOpts are the optional parameters of NewIconButton, NewIconButtonFilled,
// NewIconButtonFilledTonal and NewIconButtonOutlined.
//
// Flutter: material/icon_button.dart IconButton.new
type IconButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// IconSize is the size of the icon inside the button; the default is the
	// icon theme size, or 24.
	IconSize foundation.Opt[float64]
	// VisualDensity defines how compact the layout of the button is.
	VisualDensity foundation.Opt[VisualDensity]
	// Padding is the padding around the icon; the default is 8 on all sides.
	Padding painting.EdgeInsetsGeometry
	// Alignment defines how the icon is positioned within the button.
	Alignment painting.AlignmentGeometry
	// SplashRadius is the splash radius of the button (Material 2).
	SplashRadius foundation.Opt[float64]
	// Color is the color of the icon; the default is the one of the style.
	Color foundation.Opt[ui.Color]
	// FocusColor is the color of the button when it has the input focus.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the color of the button when a pointer is hovering over it.
	HoverColor foundation.Opt[ui.Color]
	// HighlightColor is the highlight color of the button while it is pressed.
	HighlightColor foundation.Opt[ui.Color]
	// SplashColor is the splash color of the button (Material 2).
	SplashColor foundation.Opt[ui.Color]
	// DisabledColor is the color of the icon when the button is disabled.
	DisabledColor foundation.Opt[ui.Color]
	// OnHover is called when a pointer enters or exits the button.
	OnHover func(value bool)
	// OnLongPress is called when the button is long-pressed.
	OnLongPress func()
	// MouseCursor is the cursor over the button.
	MouseCursor services.MouseCursor
	// FocusNode is the focus node of the button.
	FocusNode *widgets.FocusNode
	// Autofocus says that the button focuses itself when it is built.
	Autofocus bool
	// Tooltip is the text of the tooltip of the button.
	Tooltip string
	// EnableFeedback says whether detected gestures give acoustic and haptic
	// feedback; the default is true.
	EnableFeedback foundation.Opt[bool]
	// Constraints are the size constraints of the button.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Style customizes the button; it wins over the arguments that map to the
	// style, over the theme and over the default style.
	Style *ButtonStyle
	// IsSelected says whether the button is selected (a toggle button); nil
	// means it is not a toggle button.
	IsSelected foundation.Opt[bool]
	// SelectedIcon is the icon of the selected button.
	SelectedIcon widgets.Widget
	// StatesController is the controller of the widget states of the button.
	StatesController *widgets.WidgetStatesController
}

// IconButton is a Material Design icon button: an icon that reacts to touches
// by filling with color (ink). The filled, filled tonal and outlined variants
// are for the actions with more emphasis, and with a selected state they toggle.
//
// Flutter: material/icon_button.dart IconButton
type IconButton struct {
	widgets.StatelessWidget
	onPressed func()
	icon      widgets.Widget
	o         IconButtonOpts
	variant   iconButtonVariant
}

func newIconButton(onPressed func(), icon widgets.Widget, variant iconButtonVariant, opts []IconButtonOpts) *IconButton {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if r, ok := o.SplashRadius.Get(); ok && !(r > 0) {
			panic(errors.New("IconButtonSplashRadius", "the splash radius of an IconButton must be greater than zero", "splashRadius", r))
		}
	}
	return &IconButton{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed: onPressed, icon: icon, o: o, variant: variant}
}

// NewIconButton creates a standard icon button. A nil onPressed disables the
// button.
//
// Flutter: material/icon_button.dart IconButton.new
func NewIconButton(onPressed func(), icon widgets.Widget, opts ...IconButtonOpts) *IconButton {
	return newIconButton(onPressed, icon, iconButtonStandard, opts)
}

// NewIconButtonFilled creates a filled icon button: the high emphasis variant.
//
// Flutter: material/icon_button.dart IconButton.filled
func NewIconButtonFilled(onPressed func(), icon widgets.Widget, opts ...IconButtonOpts) *IconButton {
	return newIconButton(onPressed, icon, iconButtonFilled, opts)
}

// NewIconButtonFilledTonal creates a filled tonal icon button: the variant
// between the filled and the outlined in emphasis.
//
// Flutter: material/icon_button.dart IconButton.filledTonal
func NewIconButtonFilledTonal(onPressed func(), icon widgets.Widget, opts ...IconButtonOpts) *IconButton {
	return newIconButton(onPressed, icon, iconButtonFilledTonal, opts)
}

// NewIconButtonOutlined creates an outlined icon button: the medium emphasis
// variant.
//
// Flutter: material/icon_button.dart IconButton.outlined
func NewIconButtonOutlined(onPressed func(), icon widgets.Widget, opts ...IconButtonOpts) *IconButton {
	return newIconButton(onPressed, icon, iconButtonOutlined, opts)
}

// OnPressed is called when the button is tapped; nil disables the button.
//
// Flutter: material/icon_button.dart IconButton.onPressed
func (w *IconButton) OnPressed() func() { return w.onPressed }

// Icon is the icon of the button.
//
// Flutter: material/icon_button.dart IconButton.icon
func (w *IconButton) Icon() widgets.Widget { return w.icon }

// Tooltip is the text of the tooltip of the button.
//
// Flutter: material/icon_button.dart IconButton.tooltip
func (w *IconButton) Tooltip() string { return w.o.Tooltip }

// IconButtonStyleFromOpts are the parameters of IconButtonStyleFrom.
//
// Flutter: material/icon_button.dart IconButton.styleFrom
type IconButtonStyleFromOpts struct {
	// ForegroundColor is the color of the icon of the enabled button.
	ForegroundColor foundation.Opt[ui.Color]
	// BackgroundColor is the color of the enabled button.
	BackgroundColor foundation.Opt[ui.Color]
	// DisabledForegroundColor is the color of the icon of the disabled button.
	DisabledForegroundColor foundation.Opt[ui.Color]
	// DisabledBackgroundColor is the color of the disabled button.
	DisabledBackgroundColor foundation.Opt[ui.Color]
	// FocusColor is the overlay color of the focused button.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the overlay color of the hovered button.
	HoverColor foundation.Opt[ui.Color]
	// HighlightColor is the overlay color of the pressed button.
	HighlightColor foundation.Opt[ui.Color]
	// ShadowColor is the color of the shadow below the button.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the button.
	SurfaceTintColor foundation.Opt[ui.Color]
	// OverlayColor is the overlay color for the states without their own color;
	// the default is the foreground color.
	OverlayColor foundation.Opt[ui.Color]
	// Elevation is the elevation of the button.
	Elevation foundation.Opt[float64]
	// MinimumSize is the minimum size of the button.
	MinimumSize foundation.Opt[ui.Size]
	// FixedSize is the size of the button.
	FixedSize foundation.Opt[ui.Size]
	// MaximumSize is the maximum size of the button.
	MaximumSize foundation.Opt[ui.Size]
	// IconSize is the size of the icon.
	IconSize foundation.Opt[float64]
	// Side is the outline of the button.
	Side foundation.Opt[painting.BorderSide]
	// Shape is the shape of the button.
	Shape painting.OutlinedBorder
	// Padding is the padding around the icon.
	Padding painting.EdgeInsetsGeometry
	// EnabledMouseCursor is the cursor over the enabled button.
	EnabledMouseCursor services.MouseCursor
	// DisabledMouseCursor is the cursor over the disabled button.
	DisabledMouseCursor services.MouseCursor
	// VisualDensity defines how compact the layout of the button is.
	VisualDensity foundation.Opt[VisualDensity]
	// TapTargetSize configures the minimum size of the tap target.
	TapTargetSize MaterialTapTargetSize
	// AnimationDuration is the duration of the changes of the shape and elevation.
	AnimationDuration foundation.Opt[time.Duration]
	// EnableFeedback says whether detected gestures give acoustic and haptic feedback.
	EnableFeedback foundation.Opt[bool]
	// Alignment is the alignment of the icon within the button.
	Alignment painting.AlignmentGeometry
	// SplashFactory creates the splashes of the button.
	SplashFactory InteractiveInkFeatureFactory
}

// IconButtonStyleFrom is a static convenience that makes a ButtonStyle for an
// icon button from simple values.
//
// Flutter: material/icon_button.dart IconButton.styleFrom
func IconButtonStyleFrom(o IconButtonStyleFromOpts) *ButtonStyle {
	overlayFallback := firstOpt(o.OverlayColor, o.ForegroundColor)
	var overlay widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	if firstOpt(o.HoverColor, o.FocusColor, o.HighlightColor, overlayFallback).IsSome() {
		if oc, ok := o.OverlayColor.Get(); ok && oc.A == 0 {
			overlay = widgets.NewWidgetStatePropertyAll(o.OverlayColor)
		} else {
			withOpacity := func(opacity float64) foundation.Opt[ui.Color] {
				if c, ok := overlayFallback.Get(); ok {
					return foundation.Some(c.WithOpacity(opacity))
				}
				return foundation.None[ui.Color]()
			}
			overlay = widgets.NewWidgetStateMapper(widgets.WidgetStateMap[foundation.Opt[ui.Color]]{
				{Key: widgets.WidgetStatePressed, Value: firstOpt(o.HighlightColor, withOpacity(0.1))},
				{Key: widgets.WidgetStateHovered, Value: firstOpt(o.HoverColor, withOpacity(0.08))},
				{Key: widgets.WidgetStateFocused, Value: firstOpt(o.FocusColor, withOpacity(0.1))},
			})
		}
	}
	s := &ButtonStyle{
		BackgroundColor:   ButtonStyleButtonDefaultColor(o.BackgroundColor, o.DisabledBackgroundColor),
		ForegroundColor:   ButtonStyleButtonDefaultColor(o.ForegroundColor, o.DisabledForegroundColor),
		OverlayColor:      overlay,
		ShadowColor:       allOptOr(o.ShadowColor),
		SurfaceTintColor:  allOptOr(o.SurfaceTintColor),
		Elevation:         allOptOr(o.Elevation),
		Padding:           allNilable(o.Padding),
		MinimumSize:       allOptOr(o.MinimumSize),
		FixedSize:         allOptOr(o.FixedSize),
		MaximumSize:       allOptOr(o.MaximumSize),
		IconSize:          allOptOr(o.IconSize),
		Side:              allOptOr(o.Side),
		Shape:             allNilable(o.Shape),
		VisualDensity:     o.VisualDensity,
		TapTargetSize:     o.TapTargetSize,
		AnimationDuration: o.AnimationDuration,
		EnableFeedback:    o.EnableFeedback,
		Alignment:         o.Alignment,
		SplashFactory:     o.SplashFactory,
	}
	if o.EnabledMouseCursor != nil || o.DisabledMouseCursor != nil {
		s.MouseCursor = widgets.NewWidgetStateMapper(widgets.WidgetStateMap[services.MouseCursor]{
			{Key: widgets.WidgetStateDisabled, Value: o.DisabledMouseCursor},
			{Key: widgets.WidgetStateAny, Value: o.EnabledMouseCursor},
		})
	}
	return s
}

// Build builds the Material 3 button as a ButtonStyleButton, and the Material 2
// one as an InkResponse around the padded icon.
//
// Flutter: material/icon_button.dart IconButton.build
func (w *IconButton) Build(bc widgets.BuildContext) widgets.Widget {
	theme := ThemeOf(bc)
	o := w.o

	if theme.UseMaterial3 {
		var minSize, maxSize foundation.Opt[ui.Size]
		if c, ok := o.Constraints.Get(); ok {
			minSize = foundation.Some(ui.Size{Width: c.MinWidth, Height: c.MinHeight})
			maxSize = foundation.Some(ui.Size{Width: c.MaxWidth, Height: c.MaxHeight})
		}
		adjustedStyle := IconButtonStyleFrom(IconButtonStyleFromOpts{
			VisualDensity:           o.VisualDensity,
			ForegroundColor:         o.Color,
			DisabledForegroundColor: o.DisabledColor,
			FocusColor:              o.FocusColor,
			HoverColor:              o.HoverColor,
			HighlightColor:          o.HighlightColor,
			Padding:                 o.Padding,
			MinimumSize:             minSize,
			MaximumSize:             maxSize,
			IconSize:                o.IconSize,
			Alignment:               o.Alignment,
			EnabledMouseCursor:      o.MouseCursor,
			DisabledMouseCursor:     o.MouseCursor,
			EnableFeedback:          o.EnableFeedback,
		})
		if o.Style != nil {
			adjustedStyle = o.Style.Merge(adjustedStyle)
		}
		if adjustedStyle.IconColor == nil {
			adjustedStyle = adjustedStyle.CopyWith(ButtonStyleOpts{IconColor: adjustedStyle.ForegroundColor})
		}
		effectiveIcon := w.icon
		if o.IsSelected.OrElse(false) && o.SelectedIcon != nil {
			effectiveIcon = o.SelectedIcon
		}
		var onLongPress func()
		if w.onPressed != nil {
			onLongPress = o.OnLongPress
		}
		return &selectableIconButton{StatefulWidget: widgets.NewStatefulWidgetBase(), style: adjustedStyle, onPressed: w.onPressed, onHover: o.OnHover,
			onLongPress: onLongPress, autofocus: o.Autofocus, focusNode: o.FocusNode, isSelected: o.IsSelected, variant: w.variant,
			tooltip: o.Tooltip, statesController: o.StatesController, child: effectiveIcon}
	}

	DebugCheckHasMaterial(bc)

	currentColor := o.Color
	if w.onPressed == nil {
		currentColor = firstOpt(o.DisabledColor, foundation.Some(theme.DisabledColor))
	}
	effectiveVisualDensity := o.VisualDensity.OrElse(theme.VisualDensity)
	unadjustedConstraints := o.Constraints.OrElse(rendering.BoxConstraints{MinWidth: iconButtonMinButtonSize, MinHeight: iconButtonMinButtonSize,
		MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)})
	adjustedConstraints := effectiveVisualDensity.EffectiveConstraints(unadjustedConstraints)
	effectiveIconSize := o.IconSize.OrElse(widgets.IconThemeOf(bc).Size().OrElse(24.0))
	var effectivePadding painting.EdgeInsetsGeometry = painting.NewEdgeInsetsAll(8.0)
	if o.Padding != nil {
		effectivePadding = o.Padding
	}
	var effectiveAlignment painting.AlignmentGeometry = painting.AlignmentCenter
	if o.Alignment != nil {
		effectiveAlignment = o.Alignment
	}

	var result widgets.Widget = widgets.NewConstrainedBox(adjustedConstraints,
		widgets.NewPadding(effectivePadding,
			widgets.NewSizedBoxSquare(foundation.Some(effectiveIconSize),
				widgets.NewAlign(widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(effectiveIconSize), Color: currentColor}), w.icon),
					widgets.AlignOpts{Alignment: effectiveAlignment}))))

	var mouseCursor services.MouseCursor = services.SystemMouseCursors.Basic
	if w.onPressed != nil {
		mouseCursor = widgets.WidgetStateMouseCursorAdaptiveClickable
	}
	if o.MouseCursor != nil {
		mouseCursor = o.MouseCursor
	}
	var onLongPress func()
	if w.onPressed != nil {
		onLongPress = o.OnLongPress
	}
	result = NewInkResponse(result, InkResponseOpts{
		FocusNode:       o.FocusNode,
		Autofocus:       o.Autofocus,
		CanRequestFocus: foundation.Some(w.onPressed != nil),
		OnTap:           w.onPressed,
		OnHover:         o.OnHover,
		OnLongPress:     onLongPress,
		MouseCursor:     mouseCursor,
		EnableFeedback:  foundation.Some(o.EnableFeedback.OrElse(true)),
		FocusColor:      firstOpt(o.FocusColor, foundation.Some(theme.FocusColor)),
		HoverColor:      firstOpt(o.HoverColor, foundation.Some(theme.HoverColor)),
		HighlightColor:  firstOpt(o.HighlightColor, foundation.Some(theme.HighlightColor)),
		SplashColor:     firstOpt(o.SplashColor, foundation.Some(theme.SplashColor)),
		Radius: firstOpt(o.SplashRadius, foundation.Some(math.Max(MaterialDefaultSplashRadius,
			(effectiveIconSize+math.Min(effectivePadding.Horizontal(), effectivePadding.Vertical()))*0.7))),
	})
	if o.Tooltip != "" {
		result = NewTooltip(result, TooltipOpts{Message: o.Tooltip})
	}
	return widgets.NewSemantics(result, widgets.SemanticsOpts{Button: foundation.Some(true), Enabled: foundation.Some(w.onPressed != nil)})
}

// DebugFillProperties adds the properties of the button.
//
// Flutter: material/icon_button.dart IconButton.debugFillProperties
func (w *IconButton) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	properties.Add(foundation.NewStringProperty("tooltip", foundation.Some(w.o.Tooltip), foundation.StringPropertyOpts{Unquoted: true}))
	callback := func(set bool) any {
		if set {
			return true
		}
		return nil
	}
	properties.Add(foundation.NewObjectFlagProperty("onPressed", callback(w.onPressed != nil), foundation.ObjectFlagPropertyOpts{IfNull: "disabled"}))
	properties.Add(foundation.NewObjectFlagProperty("onHover", callback(w.o.OnHover != nil), foundation.ObjectFlagPropertyOpts{IfNull: "disabled"}))
	properties.Add(foundation.NewObjectFlagProperty("onLongPress", callback(w.o.OnLongPress != nil), foundation.ObjectFlagPropertyOpts{IfNull: "disabled"}))
	properties.Add(colorProp("color", w.o.Color))
	properties.Add(colorProp("disabledColor", w.o.DisabledColor))
	properties.Add(colorProp("focusColor", w.o.FocusColor))
	properties.Add(colorProp("hoverColor", w.o.HoverColor))
	properties.Add(colorProp("highlightColor", w.o.HighlightColor))
	properties.Add(colorProp("splashColor", w.o.SplashColor))
	properties.Add(objectProp("padding", w.o.Padding))
	properties.Add(objectProp("focusNode", nilIfZero(w.o.FocusNode)))
}

// selectableIconButton keeps the selected state of an icon button.
//
// Flutter: material/icon_button.dart _SelectableIconButton
type selectableIconButton struct {
	widgets.StatefulWidget
	isSelected       foundation.Opt[bool]
	style            *ButtonStyle
	focusNode        *widgets.FocusNode
	variant          iconButtonVariant
	autofocus        bool
	onPressed        func()
	tooltip          string
	child            widgets.Widget
	onLongPress      func()
	onHover          func(bool)
	statesController *widgets.WidgetStatesController
}

func (w *selectableIconButton) CreateState() widgets.State { return &selectableIconButtonState{} }

type selectableIconButtonState struct {
	widgets.StateOf[*selectableIconButton]
	internalStatesController *widgets.WidgetStatesController
}

func (s *selectableIconButtonState) statesController() *widgets.WidgetStatesController {
	if c := s.Widget().statesController; c != nil {
		return c
	}
	return s.internalStatesController
}

func (s *selectableIconButtonState) isSelected() bool { return s.Widget().isSelected.OrElse(false) }

func (s *selectableIconButtonState) initStatesController() {
	if s.Widget().statesController == nil {
		s.internalStatesController = widgets.NewWidgetStatesController()
	}
	s.statesController().Update(widgets.WidgetStateSelected, s.isSelected())
}

func (s *selectableIconButtonState) InitState() { s.initStatesController() }

func (s *selectableIconButtonState) DidUpdateWidget(old *selectableIconButton) {
	w := s.Widget()
	if w.statesController != old.statesController {
		if w.statesController != nil && s.internalStatesController != nil {
			s.internalStatesController.Dispose()
			s.internalStatesController = nil
		}
		s.initStatesController()
	}
	if w.isSelected != old.isSelected {
		s.statesController().Update(widgets.WidgetStateSelected, s.isSelected())
	}
}

func (s *selectableIconButtonState) Dispose() {
	if s.internalStatesController != nil {
		s.internalStatesController.Dispose()
	}
}

func (s *selectableIconButtonState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	toggleable := w.isSelected.IsSome()
	var onLongPress func()
	if w.onPressed != nil {
		onLongPress = w.onLongPress
	}
	return &iconButtonM3{
		ButtonStyleButton: NewButtonStyleButtonBase(w.onPressed, widgets.NewSemantics(w.child, widgets.SemanticsOpts{Selected: w.isSelected}), ButtonStyleButtonOpts{
			Style: w.style, FocusNode: w.focusNode, OnHover: w.onHover, OnLongPress: onLongPress, Autofocus: w.autofocus,
			StatesController: s.statesController(), Tooltip: w.tooltip, ClipBehavior: ui.ClipNone,
		}),
		variant: w.variant, toggleable: toggleable,
	}
}

// iconButtonM3 is the ButtonStyleButton of a Material 3 icon button.
//
// Flutter: material/icon_button.dart _IconButtonM3
type iconButtonM3 struct {
	ButtonStyleButton
	variant    iconButtonVariant
	toggleable bool
}

func (w *iconButtonM3) DefaultStyleOf(bc widgets.BuildContext) *ButtonStyle {
	return iconButtonDefaultsM3(bc, w.variant, w.toggleable)
}

func (w *iconButtonM3) ThemeStyleOf(bc widgets.BuildContext) *ButtonStyle {
	iconTheme := widgets.IconThemeOf(bc)
	isDefaultSize := iconTheme.Size() == widgets.NewIconThemeDataFallback().Size()
	defaultColor := DefaultIconDarkColor
	if ThemeBrightnessOf(bc) == ui.BrightnessDark {
		defaultColor = DefaultIconLightColor
	}
	// Deviation: Dart's identical compares the instances of the color; the
	// comparison here is by value.
	isDefaultColor := iconTheme.Color() == foundation.Some(defaultColor)

	iconThemeStyle := IconButtonStyleFrom(IconButtonStyleFromOpts{})
	if !isDefaultColor {
		iconThemeStyle = IconButtonStyleFrom(IconButtonStyleFromOpts{ForegroundColor: iconTheme.Color()})
	}
	if !isDefaultSize {
		iconThemeStyle = iconThemeStyle.CopyWith(ButtonStyleOpts{IconSize: ButtonStyleButtonAllOrNull(iconTheme.Size())})
	}
	if style := IconButtonThemeOf(bc).Style; style != nil {
		return style.Merge(iconThemeStyle)
	}
	return iconThemeStyle
}

// iconButtonDefaultsM3 is the default style of the Material 3 icon buttons of
// the variant, where toggleable says that the button has a selected state.
//
// Flutter: material/icon_button.dart _IconButtonDefaultsM3
func iconButtonDefaultsM3(bc widgets.BuildContext, variant iconButtonVariant, toggleable bool) *ButtonStyle {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	transparent := foundation.Some(Colors.Transparent)
	some := foundation.Some[ui.Color]
	// overlay is the overlay color of the states in the order of Flutter's
	// getters: the color of the state group that applies, with the opacities of
	// the pressed, hovered and focused states.
	overlay := func(states widgets.WidgetStates, c ui.Color, pressed, hovered, focused float64) (foundation.Opt[ui.Color], bool) {
		switch {
		case states.Has(widgets.WidgetStatePressed):
			return some(c.WithOpacity(pressed)), true
		case states.Has(widgets.WidgetStateHovered):
			return some(c.WithOpacity(hovered)), true
		case states.Has(widgets.WidgetStateFocused):
			return some(c.WithOpacity(focused)), true
		}
		return foundation.None[ui.Color](), false
	}

	s := &ButtonStyle{
		AnimationDuration: foundation.Some(ThemeChangeDuration),
		EnableFeedback:    foundation.Some(true),
		Alignment:         painting.AlignmentCenter,
		Elevation:         allOpt(0.0),
		ShadowColor:       allOpt(Colors.Transparent),
		SurfaceTintColor:  allOpt(Colors.Transparent),
		Padding:           widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](painting.NewEdgeInsetsAll(8.0)),
		MinimumSize:       allOpt(ui.Size{Width: 40, Height: 40}),
		MaximumSize:       allOpt(ui.SizeInfinite),
		IconSize:          allOpt(24.0),
		Shape:             widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](painting.NewStadiumBorder()),
		MouseCursor:       widgets.WidgetStateMouseCursorAdaptiveClickable,
		VisualDensity:     foundation.Some(VisualDensityStandard()),
		TapTargetSize:     theme.MaterialTapTargetSize,
		SplashFactory:     theme.SplashFactory,
	}
	disabledForeground := some(colors.OnSurface().WithOpacity(0.38))
	disabledBackground := some(colors.OnSurface().WithOpacity(0.12))
	switch variant {
	case iconButtonStandard:
		s.BackgroundColor = allOpt(Colors.Transparent)
		s.ForegroundColor = colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			switch {
			case states.Has(widgets.WidgetStateDisabled):
				return disabledForeground
			case states.Has(widgets.WidgetStateSelected):
				return some(colors.Primary())
			}
			return some(colors.OnSurfaceVariant())
		})
		s.OverlayColor = colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if states.Has(widgets.WidgetStateSelected) {
				if c, ok := overlay(states, colors.Primary(), 0.1, 0.08, 0.1); ok {
					return c
				}
			}
			if c, ok := overlay(states, colors.OnSurfaceVariant(), 0.1, 0.08, 0.1); ok {
				return c
			}
			return transparent
		})
	case iconButtonFilled, iconButtonFilledTonal:
		background, foreground, toggleBackground, toggleForeground := colors.Primary(), colors.OnPrimary(), colors.SurfaceContainerHighest(), colors.Primary()
		unselectedOverlay := colors.Primary()
		if variant == iconButtonFilledTonal {
			background, foreground = colors.SecondaryContainer(), colors.OnSecondaryContainer()
			toggleForeground, unselectedOverlay = colors.OnSurfaceVariant(), colors.OnSurfaceVariant()
		}
		s.BackgroundColor = colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			switch {
			case states.Has(widgets.WidgetStateDisabled):
				return disabledBackground
			case states.Has(widgets.WidgetStateSelected):
				return some(background)
			case toggleable:
				return some(toggleBackground)
			}
			return some(background)
		})
		s.ForegroundColor = colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			switch {
			case states.Has(widgets.WidgetStateDisabled):
				return disabledForeground
			case states.Has(widgets.WidgetStateSelected):
				return some(foreground)
			case toggleable:
				return some(toggleForeground)
			}
			return some(foreground)
		})
		s.OverlayColor = colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if states.Has(widgets.WidgetStateSelected) {
				if c, ok := overlay(states, foreground, 0.1, 0.08, 0.1); ok {
					return c
				}
			}
			if toggleable {
				if c, ok := overlay(states, unselectedOverlay, 0.1, 0.08, 0.1); ok {
					return c
				}
			}
			if c, ok := overlay(states, foreground, 0.1, 0.08, 0.1); ok {
				return c
			}
			return transparent
		})
	case iconButtonOutlined:
		s.BackgroundColor = colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			switch {
			case states.Has(widgets.WidgetStateDisabled):
				if states.Has(widgets.WidgetStateSelected) {
					return disabledBackground
				}
				return transparent
			case states.Has(widgets.WidgetStateSelected):
				return some(colors.InverseSurface())
			}
			return transparent
		})
		s.ForegroundColor = colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			switch {
			case states.Has(widgets.WidgetStateDisabled):
				return disabledForeground
			case states.Has(widgets.WidgetStateSelected):
				return some(colors.OnInverseSurface())
			}
			return some(colors.OnSurfaceVariant())
		})
		s.OverlayColor = colorsByStates(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			if states.Has(widgets.WidgetStateSelected) {
				if c, ok := overlay(states, colors.OnInverseSurface(), 0.1, 0.08, 0.08); ok {
					return c
				}
			}
			switch {
			case states.Has(widgets.WidgetStatePressed):
				return some(colors.OnSurface().WithOpacity(0.1))
			case states.Has(widgets.WidgetStateHovered):
				return some(colors.OnSurfaceVariant().WithOpacity(0.08))
			case states.Has(widgets.WidgetStateFocused):
				return some(colors.OnSurfaceVariant().WithOpacity(0.08))
			}
			return transparent
		})
		s.Side = widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[painting.BorderSide] {
			if states.Has(widgets.WidgetStateSelected) {
				return foundation.None[painting.BorderSide]()
			}
			c := colors.Outline()
			if states.Has(widgets.WidgetStateDisabled) {
				c = colors.OnSurface().WithOpacity(0.12)
			}
			return foundation.Some(painting.NewBorderSide(painting.BorderSideOpts{Color: some(c)}))
		})
	}
	return s
}
