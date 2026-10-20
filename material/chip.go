// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
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

const (
	// chipHeight is the height of a chip.
	//
	// Flutter: material/chip.dart _kChipHeight
	chipHeight = 32.0
	// chipCheckmarkAlpha is the alpha of the check mark: 87%.
	//
	// Flutter: material/chip.dart _kCheckmarkAlpha
	chipCheckmarkAlpha = 0xde
	// chipDisabledAlpha is the alpha of the chip's content when it is disabled:
	// 38%.
	//
	// Flutter: material/chip.dart _kDisabledAlpha
	chipDisabledAlpha = 0x61
	// chipCheckmarkStrokeWidth is the stroke width of the check mark.
	//
	// Flutter: material/chip.dart _kCheckmarkStrokeWidth
	chipCheckmarkStrokeWidth = 2.0

	chipSelectDuration                  = 195 * time.Millisecond
	chipCheckmarkDuration               = 150 * time.Millisecond
	chipCheckmarkReverseDuration        = 50 * time.Millisecond
	chipDrawerDuration                  = 150 * time.Millisecond
	chipReverseDrawerDuration           = 100 * time.Millisecond
	chipDisableDuration                 = 75 * time.Millisecond
	chipPressedAnimationDuration        = 75 * time.Millisecond
	chipSelectScrimColor         uint32 = 0x60191919
)

// ChipAnimationStyle overrides the default animations of a chip: how it appears
// when it is enabled or disabled, selected or not, and when its avatar or its
// delete icon appears or goes away. A style that gives no duration keeps the
// default one.
//
// Flutter: material/chip.dart ChipAnimationStyle
type ChipAnimationStyle struct {
	// EnableAnimation overrides the animation of the chip when it is enabled or
	// disabled.
	//
	// Flutter: material/chip.dart ChipAnimationStyle.enableAnimation
	EnableAnimation *animation.AnimationStyle
	// SelectAnimation overrides the animation of the chip when it is selected or
	// unselected.
	//
	// Flutter: material/chip.dart ChipAnimationStyle.selectAnimation
	SelectAnimation *animation.AnimationStyle
	// AvatarDrawerAnimation overrides the animation of the avatar when it is
	// shown or hidden.
	//
	// Flutter: material/chip.dart ChipAnimationStyle.avatarDrawerAnimation
	AvatarDrawerAnimation *animation.AnimationStyle
	// DeleteDrawerAnimation overrides the animation of the delete icon when it
	// is shown or hidden.
	//
	// Flutter: material/chip.dart ChipAnimationStyle.deleteDrawerAnimation
	DeleteDrawerAnimation *animation.AnimationStyle
}

// chipStyleDurations are the duration and the reverse duration of an animation
// style that may be nil: the duration defaults to def.
func chipStyleDurations(s *animation.AnimationStyle, def time.Duration) (time.Duration, foundation.Opt[time.Duration]) {
	if s == nil {
		return def, foundation.None[time.Duration]()
	}
	return s.Duration.OrElse(def), s.ReverseDuration
}

// RawChipOpts are the optional parameters of NewRawChip.
//
// Flutter: material/chip.dart RawChip.new
type RawChipOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// DefaultProperties are the defaults of the chip, used instead of those of
	// the Material version (M2 or M3) when they are given.
	DefaultProperties foundation.Opt[ChipThemeData]
	// Avatar is the widget before the label, typically a CircleAvatar.
	Avatar widgets.Widget
	// LabelStyle is the style of the label, merged over that of the theme.
	LabelStyle *painting.TextStyle
	// Padding is the padding between the chip's border and its content.
	Padding painting.EdgeInsetsGeometry
	// VisualDensity is how compact the chip is.
	VisualDensity foundation.Opt[VisualDensity]
	// LabelPadding is the padding around the label.
	LabelPadding painting.EdgeInsetsGeometry
	// DeleteIcon is the icon that is shown when OnDeleted is set; the default is
	// Icons.cancel.
	DeleteIcon widgets.Widget
	// OnDeleted is called when the delete icon is pressed; the chip shows the
	// icon only when it is set.
	OnDeleted func()
	// DeleteIconColor is the color of the delete icon.
	DeleteIconColor foundation.Opt[ui.Color]
	// DeleteButtonTooltipMessage is the message of the tooltip of the delete
	// icon; the default is that of the localizations.
	DeleteButtonTooltipMessage foundation.Opt[string]
	// OnPressed is called when the chip is tapped; it cannot be given with
	// OnSelected.
	OnPressed func()
	// OnSelected is called when the chip is tapped, with the value that it asks
	// for; it cannot be given with OnPressed.
	OnSelected func(selected bool)
	// PressElevation is the elevation while the chip is pressed.
	PressElevation foundation.Opt[float64]
	// TapEnabled says whether the body of the chip reacts to taps; the default is
	// true.
	TapEnabled foundation.Opt[bool]
	// Selected says whether the chip is selected.
	Selected bool
	// IsEnabled says whether the chip is enabled; the default is true.
	IsEnabled foundation.Opt[bool]
	// DisabledColor is the color of the chip's background when it is disabled.
	DisabledColor foundation.Opt[ui.Color]
	// SelectedColor is the color of the chip's background when it is selected.
	SelectedColor foundation.Opt[ui.Color]
	// Tooltip is the message of the tooltip of the chip.
	Tooltip foundation.Opt[string]
	// Side is the border of the chip, as one border for every state.
	Side foundation.Opt[painting.BorderSide]
	// StateSide is the border of the chip in each state (Flutter's
	// WidgetStateBorderSide); it replaces Side when it is set.
	StateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// Shape is the shape of the chip's border and of its ink splash.
	//
	// Deviation: a shape that depends on the widget states (Flutter's
	// WidgetStateOutlinedBorder) is not supported; the shape is a plain border.
	Shape painting.OutlinedBorder
	// ClipBehavior is how the content is clipped; the default is ui.ClipNone.
	ClipBehavior ui.Clip
	// FocusNode is the focus node of the chip.
	FocusNode *widgets.FocusNode
	// Autofocus says whether the chip takes the focus when it is first built.
	Autofocus bool
	// Color is the color of the chip, in each state; it takes precedence over the
	// other colors.
	Color widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// BackgroundColor is the color of the chip's background.
	BackgroundColor foundation.Opt[ui.Color]
	// MaterialTapTargetSize is the size of the tap target; the default is that of
	// the theme.
	MaterialTapTargetSize MaterialTapTargetSize
	// Elevation is the elevation of the chip.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the chip's shadow.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the color of the surface tint overlay.
	SurfaceTintColor foundation.Opt[ui.Color]
	// IconTheme is the theme of the icons of the chip.
	IconTheme foundation.Opt[widgets.IconThemeData]
	// SelectedShadowColor is the color of the shadow when the chip is selected.
	SelectedShadowColor foundation.Opt[ui.Color]
	// ShowCheckmark says whether a selected chip shows a check mark.
	ShowCheckmark foundation.Opt[bool]
	// CheckmarkColor is the color of the check mark.
	CheckmarkColor foundation.Opt[ui.Color]
	// AvatarBorder is the shape of the avatar, which the selection scrim follows;
	// the default is a circle.
	AvatarBorder painting.ShapeBorder
	// AvatarBoxConstraints are the constraints of the avatar.
	AvatarBoxConstraints foundation.Opt[rendering.BoxConstraints]
	// DeleteIconBoxConstraints are the constraints of the delete icon.
	DeleteIconBoxConstraints foundation.Opt[rendering.BoxConstraints]
	// ChipAnimationStyle overrides the durations of the animations of the chip.
	ChipAnimationStyle *ChipAnimationStyle
	// MouseCursor is the cursor over the chip.
	MouseCursor services.MouseCursor
}

// RawChip is a Material Design chip that the other chips (Input, Filter, Choice
// and Action) are made of: a compact element that represents an attribute, text,
// entity or action.
//
// Flutter: material/chip.dart RawChip
type RawChip struct {
	widgets.StatefulWidget
	label widgets.Widget
	o     RawChipOpts
}

// NewRawChip creates a chip with the label. In debug builds it panics with the
// tag RawChipPressElevationNegative or RawChipElevationNegative when the
// elevations are negative.
//
// Flutter: material/chip.dart RawChip.new
func NewRawChip(label widgets.Widget, opts ...RawChipOpts) *RawChip {
	o := firstOpts(opts)
	debugCheckChipElevations("RawChip", o.PressElevation, o.Elevation)
	return &RawChip{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), label: label, o: o}
}

// debugCheckChipElevations panics when an elevation of a chip is negative.
func debugCheckChipElevations(name string, pressElevation, elevation foundation.Opt[float64]) {
	if !foundation.DebugMode {
		return
	}
	if v, ok := pressElevation.Get(); ok && !(v >= 0.0) {
		panic(errors.New(name+"PressElevationNegative", "the press elevation of a chip must not be negative", "pressElevation", v))
	}
	if v, ok := elevation.Get(); ok && !(v >= 0.0) {
		panic(errors.New(name+"ElevationNegative", "the elevation of a chip must not be negative", "elevation", v))
	}
}

// Label is the primary content of the chip.
//
// Flutter: material/chip.dart RawChip.label
func (w *RawChip) Label() widgets.Widget { return w.label }

func (w *RawChip) isEnabled() bool  { return w.o.IsEnabled.OrElse(true) }
func (w *RawChip) tapEnabled() bool { return w.o.TapEnabled.OrElse(true) }
func (w *RawChip) deleteIcon() widgets.Widget {
	if w.o.DeleteIcon != nil {
		return w.o.DeleteIcon
	}
	return widgets.NewIcon(Icons.Cancel())
}

// CreateState creates the state.
//
// Flutter: material/chip.dart RawChip.createState
func (w *RawChip) CreateState() widgets.State { return &rawChipState{} }

type rawChipState struct {
	widgets.StateOf[*RawChip]
	widgets.TickerProvider

	selectController       *animation.AnimationController
	avatarDrawerController *animation.AnimationController
	deleteDrawerController *animation.AnimationController
	enableController       *animation.AnimationController
	checkmarkAnimation     *animation.CurvedAnimation
	avatarDrawerAnimation  *animation.CurvedAnimation
	deleteDrawerAnimation  *animation.CurvedAnimation
	enableAnimation        *animation.CurvedAnimation
	selectionFade          *animation.CurvedAnimation

	statesController *widgets.WidgetStatesController
	statesHandle     foundation.ListenerHandle
	isTappingFlag    bool
}

func (s *rawChipState) hasDeleteButton() bool { return s.Widget().o.OnDeleted != nil }
func (s *rawChipState) hasAvatar() bool       { return s.Widget().o.Avatar != nil }

func (s *rawChipState) canTap() bool {
	w := s.Widget()
	return w.isEnabled() && w.tapEnabled() && (w.o.OnPressed != nil || w.o.OnSelected != nil)
}

func (s *rawChipState) isTapping() bool { return s.canTap() && s.isTappingFlag }

func (s *rawChipState) InitState() {
	w := s.Widget()
	if foundation.DebugMode && w.o.OnSelected != nil && w.o.OnPressed != nil {
		panic(errors.New("RawChipSelectedAndPressed", "a RawChip cannot have both onSelected and onPressed"))
	}
	s.statesController = widgets.NewWidgetStatesController()
	s.statesController.Update(widgets.WidgetStateDisabled, !w.isEnabled())
	s.statesController.Update(widgets.WidgetStateSelected, w.o.Selected)
	s.statesHandle = s.statesController.AddListener(func() { s.SetState(func() {}) })

	style := w.o.ChipAnimationStyle
	var enableS, selectS, avatarS, deleteS *animation.AnimationStyle
	if style != nil {
		enableS, selectS, avatarS, deleteS = style.EnableAnimation, style.SelectAnimation, style.AvatarDrawerAnimation, style.DeleteDrawerAnimation
	}
	val := func(b bool) foundation.Opt[float64] {
		if b {
			return foundation.Some(1.0)
		}
		return foundation.Some(0.0)
	}
	controller := func(st *animation.AnimationStyle, def time.Duration, value bool) *animation.AnimationController {
		d, rd := chipStyleDurations(st, def)
		return animation.NewAnimationController(s, animation.AnimationControllerOpts{
			Duration: foundation.Some(d), ReverseDuration: rd, Value: val(value)})
	}
	s.selectController = controller(selectS, chipSelectDuration, w.o.Selected)
	s.selectionFade = animation.NewCurvedAnimation(s.selectController, animation.Curves.FastOutSlowIn)
	s.avatarDrawerController = controller(avatarS, chipDrawerDuration, s.hasAvatar() || w.o.Selected)
	s.deleteDrawerController = controller(deleteS, chipDrawerDuration, s.hasDeleteButton())
	s.enableController = controller(enableS, chipDisableDuration, w.isEnabled())

	ms := func(d time.Duration) float64 { return float64(d.Milliseconds()) }
	checkmarkPercentage := ms(chipCheckmarkDuration) / ms(chipSelectDuration)
	checkmarkReversePercentage := ms(chipCheckmarkReverseDuration) / ms(chipSelectDuration)
	avatarDrawerReversePercentage := ms(chipReverseDrawerDuration) / ms(chipSelectDuration)
	s.checkmarkAnimation = animation.NewCurvedAnimation(s.selectController,
		animation.NewInterval(1.0-checkmarkPercentage, 1.0, animation.Curves.FastOutSlowIn),
		animation.NewInterval(1.0-checkmarkReversePercentage, 1.0, animation.Curves.FastOutSlowIn))
	s.deleteDrawerAnimation = animation.NewCurvedAnimation(s.deleteDrawerController, animation.Curves.FastOutSlowIn)
	s.avatarDrawerAnimation = animation.NewCurvedAnimation(s.avatarDrawerController, animation.Curves.FastOutSlowIn,
		animation.NewInterval(1.0-avatarDrawerReversePercentage, 1.0, animation.Curves.FastOutSlowIn))
	s.enableAnimation = animation.NewCurvedAnimation(s.enableController, animation.Curves.FastOutSlowIn)
}

func (s *rawChipState) Dispose() {
	s.selectController.Dispose()
	s.avatarDrawerController.Dispose()
	s.deleteDrawerController.Dispose()
	s.enableController.Dispose()
	s.checkmarkAnimation.Dispose()
	s.avatarDrawerAnimation.Dispose()
	s.deleteDrawerAnimation.Dispose()
	s.enableAnimation.Dispose()
	s.selectionFade.Dispose()
	s.statesController.RemoveListener(s.statesHandle)
	s.statesController.Dispose()
}

func (s *rawChipState) handleTapDown(gestures.TapDownDetails) {
	if !s.canTap() {
		return
	}
	s.statesController.Update(widgets.WidgetStatePressed, true)
	s.SetState(func() { s.isTappingFlag = true })
}

func (s *rawChipState) handleTapCancel() {
	if !s.canTap() {
		return
	}
	s.statesController.Update(widgets.WidgetStatePressed, false)
	s.SetState(func() { s.isTappingFlag = false })
}

func (s *rawChipState) handleTap() {
	if !s.canTap() {
		return
	}
	s.statesController.Update(widgets.WidgetStatePressed, false)
	s.SetState(func() { s.isTappingFlag = false })
	w := s.Widget()
	if w.o.OnSelected != nil {
		w.o.OnSelected(!w.o.Selected)
	}
	if w.o.OnPressed != nil {
		w.o.OnPressed()
	}
}

// resolveChipSide resolves a border of a chip, given as a plain side or as one
// that depends on the states.
func resolveChipSide(side foundation.Opt[painting.BorderSide], stateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]], states widgets.WidgetStates) foundation.Opt[painting.BorderSide] {
	if stateSide != nil {
		return stateSide.Resolve(states)
	}
	return side
}

// Flutter: material/chip.dart _RawChipState._getShape
func (s *rawChipState) getShape(chipTheme, chipDefaults ChipThemeData) painting.OutlinedBorder {
	w := s.Widget()
	states := s.statesController.Value()
	resolvedSide := firstSide(resolveChipSide(w.o.Side, w.o.StateSide, states), resolveChipSide(chipTheme.Side, chipTheme.StateSide, states))
	var resolvedShape painting.OutlinedBorder
	for _, shape := range []painting.OutlinedBorder{w.o.Shape, chipTheme.Shape, chipDefaults.Shape} {
		if shape != nil {
			resolvedShape = shape
			break
		}
	}
	if resolvedShape == nil {
		resolvedShape = painting.NewStadiumBorder()
	}
	if side, ok := resolvedSide.Get(); ok {
		return resolvedShape.CopyWithSide(side)
	}
	if resolvedShape.Side() != painting.BorderSideNone {
		return resolvedShape
	}
	if side, ok := resolveChipSide(chipDefaults.Side, chipDefaults.StateSide, states).Get(); ok {
		return resolvedShape.CopyWithSide(side)
	}
	return resolvedShape
}

// chipColorOverrides resolves the colors given to a chip one by one, in the way
// of Flutter's _IndividualOverrides.
//
// Flutter: material/chip.dart _IndividualOverrides
func chipColorOverrides(states widgets.WidgetStates, color widgets.WidgetStateProperty[foundation.Opt[ui.Color]], selectedColor, backgroundColor, disabledColor foundation.Opt[ui.Color]) foundation.Opt[ui.Color] {
	if color != nil {
		return color.Resolve(states)
	}
	selected, disabled := states.Has(widgets.WidgetStateSelected), states.Has(widgets.WidgetStateDisabled)
	switch {
	case selected && disabled:
		return selectedColor
	case disabled:
		return disabledColor
	case selected:
		return selectedColor
	}
	return backgroundColor
}

func (s *rawChipState) resolveColor(color widgets.WidgetStateProperty[foundation.Opt[ui.Color]], selectedColor, backgroundColor, disabledColor foundation.Opt[ui.Color], defaultColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]) foundation.Opt[ui.Color] {
	states := s.statesController.Value()
	if c := chipColorOverrides(states, color, selectedColor, backgroundColor, disabledColor); c.IsSome() {
		return c
	}
	if defaultColor != nil {
		return defaultColor.Resolve(states)
	}
	return foundation.None[ui.Color]()
}

func firstStateColor(values ...widgets.WidgetStateProperty[foundation.Opt[ui.Color]]) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

// Flutter: material/chip.dart _RawChipState._getBackgroundColor
func (s *rawChipState) getBackgroundColor(theme *ThemeData, chipTheme, chipDefaults ChipThemeData) foundation.Opt[ui.Color] {
	w := s.Widget()
	o := w.o
	if theme.UseMaterial3 {
		color := firstStateColor(o.Color, chipTheme.Color)
		disabledColor := s.resolveColor(color, foundation.None[ui.Color](), foundation.None[ui.Color](), firstOptColor(o.DisabledColor, chipTheme.DisabledColor), chipDefaults.Color)
		backgroundColor := s.resolveColor(color, foundation.None[ui.Color](), firstOptColor(o.BackgroundColor, chipTheme.BackgroundColor), foundation.None[ui.Color](), chipDefaults.Color)
		selectedColor := s.resolveColor(color, firstOptColor(o.SelectedColor, chipTheme.SelectedColor), foundation.None[ui.Color](), foundation.None[ui.Color](), chipDefaults.Color)
		backgroundTween := animation.NewColorTween(disabledColor, backgroundColor)
		selectTween := animation.NewColorTween(backgroundTween.Evaluate(s.enableController), selectedColor)
		return selectTween.Evaluate(s.selectionFade)
	}
	backgroundTween := animation.NewColorTween(
		firstOptColor(o.DisabledColor, chipTheme.DisabledColor, foundation.Some(theme.DisabledColor)),
		firstOptColor(o.BackgroundColor, chipTheme.BackgroundColor, theme.ChipTheme.BackgroundColor, chipDefaults.BackgroundColor))
	selectTween := animation.NewColorTween(backgroundTween.Evaluate(s.enableController),
		firstOptColor(o.SelectedColor, chipTheme.SelectedColor, theme.ChipTheme.SelectedColor, chipDefaults.SelectedColor))
	return selectTween.Evaluate(s.selectionFade)
}

func (s *rawChipState) DidUpdateWidget(old *RawChip) {
	w := s.Widget()
	if old.isEnabled() != w.isEnabled() {
		s.SetState(func() {
			s.statesController.Update(widgets.WidgetStateDisabled, !w.isEnabled())
			if w.isEnabled() {
				s.enableController.Forward()
			} else {
				s.enableController.Reverse()
			}
		})
	}
	if old.o.Avatar != w.o.Avatar || old.o.Selected != w.o.Selected {
		s.SetState(func() {
			if s.hasAvatar() || w.o.Selected {
				s.avatarDrawerController.Forward()
			} else {
				s.avatarDrawerController.Reverse()
			}
		})
	}
	if old.o.Selected != w.o.Selected {
		s.SetState(func() {
			s.statesController.Update(widgets.WidgetStateSelected, w.o.Selected)
			if w.o.Selected {
				s.selectController.Forward()
			} else {
				s.selectController.Reverse()
			}
		})
	}
	if (old.o.OnDeleted == nil) != (w.o.OnDeleted == nil) {
		s.SetState(func() {
			if s.hasDeleteButton() {
				s.deleteDrawerController.Forward()
			} else {
				s.deleteDrawerController.Reverse()
			}
		})
	}
}

func wrapWithChipTooltip(tooltip foundation.Opt[string], enabled bool, child widgets.Widget) widgets.Widget {
	if child == nil || !enabled || tooltip.IsNone() {
		return child
	}
	// TODO(step 13.9): Tooltip.
	return tempTooltip(tooltip.Value(), child)
}

// Flutter: material/chip.dart _RawChipState._buildDeleteIcon
func (s *rawChipState) buildDeleteIcon(bc widgets.BuildContext, theme *ThemeData, chipTheme, chipDefaults ChipThemeData) widgets.Widget {
	if !s.hasDeleteButton() {
		return nil
	}
	w := s.Widget()
	o := w.o
	m3 := chipDefaultsM3(bc, w.isEnabled())
	iconTheme := firstOpt(o.IconTheme, chipTheme.IconTheme, theme.ChipTheme.IconTheme, m3.IconTheme).Value()
	effectiveDeleteIconColor := firstOptColor(o.DeleteIconColor, chipTheme.DeleteIconColor, theme.ChipTheme.DeleteIconColor,
		iconThemeColor(o.IconTheme), iconThemeColor(chipTheme.IconTheme), chipDefaults.DeleteIconColor)
	effectiveIconSize := firstOptDouble(iconThemeSize(o.IconTheme), iconThemeSize(chipTheme.IconTheme),
		iconThemeSize(theme.ChipTheme.IconTheme), iconThemeSize(m3.IconTheme)).Value()

	effectiveTapTarget := o.MaterialTapTargetSize
	if effectiveTapTarget == MaterialTapTargetSizeUnspecified {
		effectiveTapTarget = theme.MaterialTapTargetSize
	}
	semanticSide := MinInteractiveDimension
	if effectiveTapTarget == MaterialTapTargetSizeShrinkWrap {
		semanticSide = MinInteractiveDimension - 8.0
	}
	density := o.VisualDensity.OrElse(theme.VisualDensity)
	adjust := density.BaseSizeAdjustment()
	var onTap func()
	if w.isEnabled() {
		onTap = o.OnDeleted
	}
	paddingVertical := 0.0
	if o.Padding != nil {
		paddingVertical = o.Padding.Vertical()
	}
	message := o.DeleteButtonTooltipMessage
	if message.IsNone() {
		message = foundation.Some(MaterialLocalizationsOf(bc).DeleteButtonTooltip())
	}
	return newEnsureMinSemanticsSize(ui.Size{Width: semanticSide + adjust.Dx, Height: semanticSide + adjust.Dy},
		wrapWithChipTooltip(message, w.isEnabled() && o.OnDeleted != nil,
			NewInkWell(widgets.NewIconTheme(iconTheme.CopyWith(widgets.IconThemeDataOpts{Color: effectiveDeleteIconColor, Size: foundation.Some(effectiveIconSize)}), w.deleteIcon()),
				InkWellOpts{
					Radius:        foundation.Some((chipHeight + paddingVertical) * .45),
					SplashFactory: unconstrainedInkSplashFactory{parent: theme.SplashFactory},
					CustomBorder:  painting.NewCircleBorder(),
					OnTap:         onTap,
				})))
}

func iconThemeColor(t foundation.Opt[widgets.IconThemeData]) foundation.Opt[ui.Color] {
	if v, ok := t.Get(); ok {
		return v.Color()
	}
	return foundation.None[ui.Color]()
}

func iconThemeSize(t foundation.Opt[widgets.IconThemeData]) foundation.Opt[float64] {
	if v, ok := t.Get(); ok {
		return v.Size()
	}
	return foundation.None[float64]()
}

// Build builds the chip.
//
// Flutter: material/chip.dart _RawChipState.build
func (s *rawChipState) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	widgets.DebugCheckHasMediaQuery(bc)
	widgets.DebugCheckHasDirectionality(bc)
	DebugCheckHasMaterialLocalizations(bc)

	w := s.Widget()
	o := w.o
	theme := ThemeOf(bc)
	chipTheme := ChipThemeOf(bc)
	brightness := chipTheme.Brightness.OrElse(theme.Brightness())
	var chipDefaults ChipThemeData
	if d, ok := o.DefaultProperties.Get(); ok {
		chipDefaults = d
	} else if theme.UseMaterial3 {
		chipDefaults = chipDefaultsM3(bc, w.isEnabled())
	} else {
		secondary := theme.PrimaryColor
		if brightness == ui.BrightnessDark {
			secondary = Colors.TealAccent.Shade200()
		}
		chipDefaults = ChipThemeDataFromDefaults(secondary, theme.TextTheme.BodyLarge, ChipThemeDataFromDefaultsOpts{Brightness: brightness})
	}
	textDirection := widgets.DirectionalityOf(bc)
	resolvedShape := s.getShape(chipTheme, chipDefaults)

	elevation := firstOptDouble(o.Elevation, chipTheme.Elevation, chipDefaults.Elevation).OrElse(0)
	pressElevation := firstOptDouble(o.PressElevation, chipTheme.PressElevation, chipDefaults.PressElevation).OrElse(0)
	shadowColor := firstOptColor(o.ShadowColor, chipTheme.ShadowColor, chipDefaults.ShadowColor)
	surfaceTintColor := firstOptColor(o.SurfaceTintColor, chipTheme.SurfaceTintColor, chipDefaults.SurfaceTintColor)
	selectedShadowColor := firstOptColor(o.SelectedShadowColor, chipTheme.SelectedShadowColor, chipDefaults.SelectedShadowColor)
	checkmarkColor := firstOptColor(o.CheckmarkColor, chipTheme.CheckmarkColor, chipDefaults.CheckmarkColor)
	showCheckmark := firstOpt(o.ShowCheckmark, chipTheme.ShowCheckmark, chipDefaults.ShowCheckmark).Value()
	padding := firstInsets(o.Padding, chipTheme.Padding, chipDefaults.Padding)
	labelStyle := chipTheme.LabelStyle
	if labelStyle == nil {
		labelStyle = chipDefaults.LabelStyle
	}
	iconTheme := firstOpt(o.IconTheme, chipTheme.IconTheme, chipDefaults.IconTheme)
	avatarBoxConstraints := firstOpt(o.AvatarBoxConstraints, chipTheme.AvatarBoxConstraints)
	deleteIconBoxConstraints := firstOpt(o.DeleteIconBoxConstraints, chipTheme.DeleteIconBoxConstraints)

	effectiveLabelStyle := labelStyle.Merge(o.LabelStyle)
	resolvedLabelStyle := effectiveLabelStyle
	avatar := o.Avatar
	if it, ok := iconTheme.Get(); ok && s.hasAvatar() {
		avatar = widgets.NewIconThemeMerge(chipDefaults.IconTheme.Value().Merge(&it), o.Avatar)
	}

	defaultFontSize := effectiveLabelStyle.FontSize.OrElse(14.0)
	effectiveTextScale := widgets.MediaQueryTextScalerOf(bc).Scale(defaultFontSize) / 14.0
	defaultLabelPadding := chipLabelPaddingForScale(effectiveTextScale)
	labelPadding := firstInsets(o.LabelPadding, chipTheme.LabelPadding, chipDefaults.LabelPadding)
	if labelPadding == nil {
		labelPadding = defaultLabelPadding
	}

	var tapColor foundation.Opt[ui.Color]
	if firstStateColor(o.Color, chipTheme.Color) != nil {
		tapColor = foundation.Some(Colors.Transparent)
	}
	canTap := s.canTap()
	shadow := shadowColor
	if o.Selected {
		shadow = selectedShadowColor
	}
	pressElevationOrElevation := elevation
	if s.isTapping() {
		pressElevationOrElevation = pressElevation
	}
	var onTap func()
	var onTapDown gestures.GestureTapDownCallback
	var onTapCancel func()
	var onHover func(bool)
	if canTap {
		onTap, onTapDown, onTapCancel = s.handleTap, s.handleTapDown, s.handleTapCancel
		onHover = func(value bool) { s.statesController.Update(widgets.WidgetStateHovered, value) }
	}

	renderTheme := chipRenderTheme{
		label: widgets.NewDefaultTextStyle(resolvedLabelStyle, w.label, widgets.DefaultTextStyleOpts{
			Overflow: painting.TextOverflowFade, TextAlign: ui.TextAlignStart, MaxLines: foundation.Some(1), SoftWrap: foundation.Some(false)}),
		avatar: widgets.NewAnimatedSwitcher(chipDrawerDuration, avatar, widgets.AnimatedSwitcherOpts{SwitchInCurve: animation.Curves.FastOutSlowIn}),
		deleteIcon: widgets.NewAnimatedSwitcher(chipDrawerDuration, s.buildDeleteIcon(bc, theme, chipTheme, chipDefaults),
			widgets.AnimatedSwitcherOpts{SwitchInCurve: animation.Curves.FastOutSlowIn}),
		brightness: brightness, padding: padding.Resolve(textDirection), visualDensity: o.VisualDensity.OrElse(theme.VisualDensity),
		labelPadding: labelPadding.Resolve(textDirection), showAvatar: s.hasAvatar(), showCheckmark: showCheckmark,
		checkmarkColor: checkmarkColor, canTapBody: canTap,
	}
	var result widgets.Widget = NewMaterial(
		NewInkWell(
			widgets.NewAnimatedBuilder(foundation.MergeListenables(s.selectController, s.enableController),
				func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
					return NewInk(child, InkOpts{Decoration: painting.NewShapeDecoration(resolvedShape, painting.ShapeDecorationOpts{
						Color: s.getBackgroundColor(theme, chipTheme, chipDefaults)})})
				},
				wrapWithChipTooltip(o.Tooltip, o.OnPressed != nil || o.OnSelected != nil, &chipRenderWidget{
					SlottedMultiChildRenderObjectWidget: widgets.NewSlottedMultiChildRenderObjectWidgetBase(),
					theme:                               renderTheme, value: o.Selected, isEnabled: w.isEnabled(),
					checkmarkAnimation: s.checkmarkAnimation, avatarDrawerAnimation: s.avatarDrawerAnimation,
					deleteDrawerAnimation: s.deleteDrawerAnimation, enableAnimation: s.enableAnimation,
					avatarBorder: o.avatarBorderOrCircle(), avatarBoxConstraints: avatarBoxConstraints,
					deleteIconBoxConstraints: deleteIconBoxConstraints,
				})),
			InkWellOpts{
				OnFocusChange: func(value bool) { s.statesController.Update(widgets.WidgetStateFocused, value) },
				FocusNode:     o.FocusNode, Autofocus: o.Autofocus, CanRequestFocus: foundation.Some(w.isEnabled()),
				OnTap: onTap, OnTapDown: onTapDown, OnTapCancel: onTapCancel, OnHover: onHover,
				MouseCursor: o.MouseCursor, HoverColor: tapColor, CustomBorder: resolvedShape,
			}),
		MaterialOpts{
			Elevation: pressElevationOrElevation, ShadowColor: shadow, SurfaceTintColor: surfaceTintColor,
			AnimationDuration: chipPressedAnimationDuration, Shape: resolvedShape, ClipBehavior: o.ClipBehavior,
		})

	density := o.VisualDensity.OrElse(theme.VisualDensity).BaseSizeAdjustment()
	constraints := rendering.NewBoxConstraints()
	tapTarget := o.MaterialTapTargetSize
	if tapTarget == MaterialTapTargetSizeUnspecified {
		tapTarget = theme.MaterialTapTargetSize
	}
	if tapTarget == MaterialTapTargetSizePadded {
		constraints = rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{
			MinWidth: foundation.Some(MinInteractiveDimension + density.Dx), MinHeight: foundation.Some(MinInteractiveDimension + density.Dy)})
	}
	result = newChipRedirectingHitDetection(constraints, widgets.NewCenter(result, widgets.CenterOpts{WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)}))
	semanticsOpts := widgets.SemanticsOpts{Button: foundation.Some(w.tapEnabled()), Container: true, Selected: foundation.Some(o.Selected)}
	if w.tapEnabled() {
		semanticsOpts.Enabled = foundation.Some(canTap)
	}
	return widgets.NewSemantics(result, semanticsOpts)
}

func (o RawChipOpts) avatarBorderOrCircle() painting.ShapeBorder {
	if o.AvatarBorder != nil {
		return o.AvatarBorder
	}
	return painting.NewCircleBorder()
}

// firstInsets is the first of the insets that is set.
func firstInsets(values ...painting.EdgeInsetsGeometry) painting.EdgeInsetsGeometry {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

// chipLabelPaddingForScale is EdgeInsets.lerp(symmetric(horizontal: 8), symmetric(horizontal: 4),
// clampDouble(scale - 1.0, 0.0, 1.0)).
func chipLabelPaddingForScale(scale float64) painting.EdgeInsetsGeometry {
	t := ui.ClampDouble(scale-1.0, 0.0, 1.0)
	h := 8.0 + (4.0-8.0)*t
	return painting.NewEdgeInsetsSymmetric(0, h)
}

// ChipOpts are the optional parameters of NewChip.
//
// Flutter: material/chip.dart Chip.new
type ChipOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Avatar is the widget before the label, typically a CircleAvatar.
	Avatar widgets.Widget
	// LabelStyle is the style of the label.
	LabelStyle *painting.TextStyle
	// LabelPadding is the padding around the label.
	LabelPadding painting.EdgeInsetsGeometry
	// DeleteIcon is the icon that is shown when OnDeleted is set.
	DeleteIcon widgets.Widget
	// OnDeleted is called when the delete icon is pressed; the chip shows the
	// icon only when it is set.
	OnDeleted func()
	// DeleteIconColor is the color of the delete icon.
	DeleteIconColor foundation.Opt[ui.Color]
	// DeleteButtonTooltipMessage is the message of the tooltip of the delete
	// icon.
	DeleteButtonTooltipMessage foundation.Opt[string]
	// Side is the border of the chip, as one border for every state.
	Side foundation.Opt[painting.BorderSide]
	// StateSide is the border of the chip in each state; it replaces Side when it
	// is set.
	StateSide widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	// Shape is the shape of the chip's border.
	Shape painting.OutlinedBorder
	// ClipBehavior is how the content is clipped.
	ClipBehavior ui.Clip
	// FocusNode is the focus node of the chip.
	FocusNode *widgets.FocusNode
	// Autofocus says whether the chip takes the focus when it is first built.
	Autofocus bool
	// Color is the color of the chip, in each state.
	Color widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// BackgroundColor is the color of the chip's background.
	BackgroundColor foundation.Opt[ui.Color]
	// Padding is the padding between the chip's border and its content.
	Padding painting.EdgeInsetsGeometry
	// VisualDensity is how compact the chip is.
	VisualDensity foundation.Opt[VisualDensity]
	// MaterialTapTargetSize is the size of the tap target.
	MaterialTapTargetSize MaterialTapTargetSize
	// Elevation is the elevation of the chip.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the chip's shadow.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the color of the surface tint overlay.
	SurfaceTintColor foundation.Opt[ui.Color]
	// IconTheme is the theme of the icons of the chip.
	IconTheme foundation.Opt[widgets.IconThemeData]
	// AvatarBoxConstraints are the constraints of the avatar.
	AvatarBoxConstraints foundation.Opt[rendering.BoxConstraints]
	// DeleteIconBoxConstraints are the constraints of the delete icon.
	DeleteIconBoxConstraints foundation.Opt[rendering.BoxConstraints]
	// ChipAnimationStyle overrides the durations of the animations of the chip.
	ChipAnimationStyle *ChipAnimationStyle
	// MouseCursor is the cursor over the chip.
	MouseCursor services.MouseCursor
}

// Chip is a Material Design chip that displays information: a compact element
// that represents an attribute, text, entity or action. It is not tappable, but
// it can be deleted.
//
// Flutter: material/chip.dart Chip
type Chip struct {
	widgets.StatelessWidget
	label widgets.Widget
	o     ChipOpts
}

// NewChip creates a chip with the label. In debug builds it panics with the tag
// ChipElevationNegative when the elevation is negative.
//
// Flutter: material/chip.dart Chip.new
func NewChip(label widgets.Widget, opts ...ChipOpts) *Chip {
	o := firstOpts(opts)
	debugCheckChipElevations("Chip", foundation.None[float64](), o.Elevation)
	return &Chip{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), label: label, o: o}
}

// Label is the primary content of the chip.
//
// Flutter: material/chip.dart Chip.label
func (w *Chip) Label() widgets.Widget { return w.label }

// Build builds a RawChip that cannot be tapped.
//
// Flutter: material/chip.dart Chip.build
func (w *Chip) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	o := w.o
	return NewRawChip(w.label, RawChipOpts{
		Avatar: o.Avatar, LabelStyle: o.LabelStyle, LabelPadding: o.LabelPadding, DeleteIcon: o.DeleteIcon, OnDeleted: o.OnDeleted,
		DeleteIconColor: o.DeleteIconColor, DeleteButtonTooltipMessage: o.DeleteButtonTooltipMessage, TapEnabled: foundation.Some(false),
		Side: o.Side, StateSide: o.StateSide, Shape: o.Shape, ClipBehavior: o.ClipBehavior, FocusNode: o.FocusNode, Autofocus: o.Autofocus,
		Color: o.Color, BackgroundColor: o.BackgroundColor, Padding: o.Padding, VisualDensity: o.VisualDensity,
		MaterialTapTargetSize: o.MaterialTapTargetSize, Elevation: o.Elevation, ShadowColor: o.ShadowColor,
		SurfaceTintColor: o.SurfaceTintColor, IconTheme: o.IconTheme, AvatarBoxConstraints: o.AvatarBoxConstraints,
		DeleteIconBoxConstraints: o.DeleteIconBoxConstraints, ChipAnimationStyle: o.ChipAnimationStyle, MouseCursor: o.MouseCursor,
	})
}

// unconstrainedInkSplashFactory makes the splashes of the parent factory without
// clipping them: the delete icon's splash may spread beyond it.
//
// Flutter: material/chip.dart _UnconstrainedInkSplashFactory
type unconstrainedInkSplashFactory struct{ parent InteractiveInkFeatureFactory }

// Create makes the feature of the parent factory, with the contained flag
// dropped.
//
// Flutter: material/chip.dart _UnconstrainedInkSplashFactory.create
func (f unconstrainedInkSplashFactory) Create(o InkFeatureCreateOpts) InteractiveInkFeature {
	o.ContainedInkWell = false
	return f.parent.Create(o)
}

var _ = semantics.NewSemanticsConfiguration
