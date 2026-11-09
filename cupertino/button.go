package cupertino

import (
	"reflect"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoButtonSize is the size of a CupertinoButton: its padding, its minimum size and its corners.
//
// Flutter: cupertino/button.dart CupertinoButtonSize
type CupertinoButtonSize int

const (
	// CupertinoButtonSizeUnspecified is the size that the button has by default: large.
	//
	// Flutter: no counterpart
	CupertinoButtonSizeUnspecified CupertinoButtonSize = iota
	// CupertinoButtonSizeSmall is a small button: 28 pixels high at least.
	//
	// Flutter: cupertino/button.dart CupertinoButtonSize.small
	CupertinoButtonSizeSmall
	// CupertinoButtonSizeMedium is a medium button: 32 pixels high at least.
	//
	// Flutter: cupertino/button.dart CupertinoButtonSize.medium
	CupertinoButtonSizeMedium
	// CupertinoButtonSizeLarge is a large button: 44 pixels high at least.
	//
	// Flutter: cupertino/button.dart CupertinoButtonSize.large
	CupertinoButtonSizeLarge
)

var cupertinoButtonSizeNames = [...]string{"large", "small", "medium", "large"}

// Name is the name of the size in Flutter's enum.
//
// Flutter: cupertino/button.dart CupertinoButtonSize.name
func (s CupertinoButtonSize) Name() string {
	if s < 0 || int(s) >= len(cupertinoButtonSizeNames) {
		return "CupertinoButtonSize(" + string(rune('0'+int(s)%10)) + ")"
	}
	return cupertinoButtonSizeNames[s]
}

type cupertinoButtonStyle int

const (
	cupertinoButtonStylePlain cupertinoButtonStyle = iota
	cupertinoButtonStyleTinted
	cupertinoButtonStyleFilled
)

// CupertinoButtonOpts are the optional parameters of NewCupertinoButton, NewCupertinoButtonTinted and
// NewCupertinoButtonFilled.
//
// Not part of them: Flutter's deprecated MinSize (use MinimumSize).
//
// Flutter: cupertino/button.dart CupertinoButton.new
type CupertinoButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// SizeStyle is the size of the button; the default is large.
	SizeStyle CupertinoButtonSize
	// Padding is the padding inside the button; the default is the one of the size.
	Padding painting.EdgeInsetsGeometry
	// Color is the background color: of a filled or tinted button, the primary color of the theme by default;
	// a plain button has none.
	Color ui.ColorValue
	// ForegroundColor is the color of the text and of the icons.
	ForegroundColor ui.ColorValue
	// DisabledColor is the background color when the button is disabled; the default is the quaternary system
	// fill of a plain button and the tertiary one of the others.
	DisabledColor ui.ColorValue
	// MinimumSize is the minimum size of the button; the default is the one of the size.
	MinimumSize foundation.Opt[ui.Size]
	// PressedOpacity is the opacity of the button while it is pressed, from 0 to 1; the default is 0.4.
	PressedOpacity foundation.Opt[float64]
	// BorderRadius is the corners of the button; the default is the one of the size.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// Alignment aligns the child in the button; the default is the center.
	Alignment painting.AlignmentGeometry
	// FocusColor is the color of the outline of the focused button.
	FocusColor ui.ColorValue
	// FocusNode is the focus node of the button.
	FocusNode *widgets.FocusNode
	// OnFocusChange is called when the focus of the button changes.
	OnFocusChange func(focused bool)
	// Autofocus is whether the button gets the focus first when there is none.
	Autofocus bool
	// MouseCursor is the cursor over the button, or a widgets.WidgetStateMouseCursor; the default is the click
	// cursor of the web when it is enabled and none of the other platforms.
	MouseCursor services.MouseCursor
	// OnLongPress is called when the button is long pressed.
	OnLongPress func()
}

// CupertinoButton is an iOS style button: text or an icon that fades while it is pressed, in a plain, a tinted
// or a filled style, in three sizes. It is disabled when it has neither OnPressed nor OnLongPress.
//
// Flutter: cupertino/button.dart CupertinoButton
type CupertinoButton struct {
	widgets.StatefulWidget
	onPressed func()
	child     widgets.Widget
	o         CupertinoButtonOpts
	style     cupertinoButtonStyle
}

func newCupertinoButton(onPressed func(), child widgets.Widget, o CupertinoButtonOpts, style cupertinoButtonStyle) *CupertinoButton {
	if foundation.DebugMode && o.PressedOpacity.IsSome() && (o.PressedOpacity.Value() < 0 || o.PressedOpacity.Value() > 1) {
		panic(errors.New(tagCupertinoButtonPressedOpacityInvalid, "pressedOpacity must be from 0.0 to 1.0", "pressedOpacity", o.PressedOpacity.Value()))
	}
	if o.DisabledColor == nil {
		if style == cupertinoButtonStylePlain {
			o.DisabledColor = CupertinoColors.QuaternarySystemFill
		} else {
			o.DisabledColor = CupertinoColors.TertiarySystemFill
		}
	}
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	return &CupertinoButton{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed, child, o, style}
}

// NewCupertinoButton creates a plain button: onPressed is nil for a button that is disabled (if it has no
// OnLongPress either). In debug builds it panics with the tag CupertinoButtonPressedOpacityInvalid when the
// pressed opacity is not from 0 to 1.
//
// Flutter: cupertino/button.dart CupertinoButton.new
func NewCupertinoButton(onPressed func(), child widgets.Widget, opts ...CupertinoButtonOpts) *CupertinoButton {
	return newCupertinoButton(onPressed, child, firstOf(opts), cupertinoButtonStylePlain)
}

// NewCupertinoButtonTinted creates a button with a tinted background: the color of the button, or the primary
// color of the theme, at a low opacity.
//
// Flutter: cupertino/button.dart CupertinoButton.tinted
func NewCupertinoButtonTinted(onPressed func(), child widgets.Widget, opts ...CupertinoButtonOpts) *CupertinoButton {
	return newCupertinoButton(onPressed, child, firstOpts(opts), cupertinoButtonStyleTinted)
}

// NewCupertinoButtonFilled creates a button with a background of the color of the button, or the primary color
// of the theme.
//
// Flutter: cupertino/button.dart CupertinoButton.filled
func NewCupertinoButtonFilled(onPressed func(), child widgets.Widget, opts ...CupertinoButtonOpts) *CupertinoButton {
	return newCupertinoButton(onPressed, child, firstOpts(opts), cupertinoButtonStyleFilled)
}

func firstOpts[T any](opts []T) T { return firstOf(opts) }

// Child is the widget in the button.
//
// Flutter: cupertino/button.dart CupertinoButton.child
func (w *CupertinoButton) Child() widgets.Widget { return w.child }

// Enabled is whether the button reacts: it has an OnPressed or an OnLongPress.
//
// Flutter: cupertino/button.dart CupertinoButton.enabled
func (w *CupertinoButton) Enabled() bool { return w.onPressed != nil || w.o.OnLongPress != nil }

// CupertinoButtonTapMoveSlopOf is how far outside of the button a tap can move and still count: 70 pixels on the
// platforms with a touch screen and none on the desktop.
//
// Flutter: cupertino/button.dart CupertinoButton.tapMoveSlop
func CupertinoButtonTapMoveSlopOf() float64 {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
		return CupertinoButtonTapMoveSlop
	}
	return 0
}

// DebugFillProperties adds whether the button is enabled.
//
// Flutter: cupertino/button.dart CupertinoButton.debugFillProperties
func (w *CupertinoButton) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewFlagProperty("enabled", foundation.Some(w.Enabled()), foundation.FlagPropertyOpts{IfFalse: "disabled"}))
}

// CreateState creates the state.
//
// Flutter: cupertino/button.dart CupertinoButton.createState
func (w *CupertinoButton) CreateState() widgets.State { return &cupertinoButtonState{} }

const (
	cupertinoButtonFadeOutDuration = 120 * time.Millisecond
	cupertinoButtonFadeInDuration  = 180 * time.Millisecond
)

type cupertinoButtonState struct {
	widgets.StateOf[*CupertinoButton]
	widgets.SingleTickerProvider
	opacityTween        *animation.Tween[float64]
	animationController *animation.AnimationController
	opacityAnimation    animation.Animation[float64]
	isFocused           bool
	buttonHeldDown      bool
	tapInProgress       bool
}

func (s *cupertinoButtonState) InitState() {
	s.opacityTween = animation.NewTween(1.0, 1.0)
	s.animationController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(200 * time.Millisecond), Value: foundation.Some(0.0)})
	s.opacityAnimation = animation.Drive[float64](animation.Drive[float64](s.animationController, animation.NewCurveTween(animation.Curves.Decelerate)), s.opacityTween)
	s.setTween()
}

func (s *cupertinoButtonState) DidUpdateWidget(*CupertinoButton) { s.setTween() }

func (s *cupertinoButtonState) setTween() {
	s.opacityTween.End = s.Widget().o.PressedOpacity.OrElse(0.4)
}

func (s *cupertinoButtonState) Dispose() { s.animationController.Dispose() }

func (s *cupertinoButtonState) handleTapDown(gestures.TapDownDetails) {
	s.SetState(func() { s.tapInProgress = true })
	if !s.buttonHeldDown {
		s.buttonHeldDown = true
		s.animate()
	}
}

func (s *cupertinoButtonState) handleTapUp(details gestures.TapUpDetails) {
	s.SetState(func() { s.tapInProgress = false })
	if s.buttonHeldDown {
		s.buttonHeldDown = false
		s.animate()
	}
	if s.insideBounds(details.GlobalPosition) {
		s.handleTap()
	}
}

func (s *cupertinoButtonState) handleTapCancel() {
	s.SetState(func() { s.tapInProgress = false })
	if s.buttonHeldDown {
		s.buttonHeldDown = false
		s.animate()
	}
}

func (s *cupertinoButtonState) handleTapMove(details gestures.TapMoveDetails) {
	shouldHold := s.insideBounds(details.GlobalPosition)
	if s.tapInProgress && shouldHold != s.buttonHeldDown {
		s.buttonHeldDown = shouldHold
		s.animate()
	}
}

// insideBounds is whether the global position is in the button, inflated by the tap slop.
func (s *cupertinoButtonState) insideBounds(global ui.Offset) bool {
	box, ok := s.Context().(widgets.Element).FindRenderObject().(rendering.Box)
	if !ok {
		return false
	}
	local := box.GlobalToLocal(global, nil)
	return box.AsRenderBox().PaintBounds().Inflate(CupertinoButtonTapMoveSlopOf()).Contains(local)
}

func (s *cupertinoButtonState) handleTap() {
	if onPressed := s.Widget().onPressed; onPressed != nil {
		onPressed()
		s.Context().(widgets.Element).FindRenderObject().SendSemanticsEvent(semantics.NewTapSemanticEvent())
	}
}

func (s *cupertinoButtonState) animate() {
	if s.animationController.IsAnimating() {
		return
	}
	wasHeldDown := s.buttonHeldDown
	var ticker *scheduler.TickerFuture
	if s.buttonHeldDown {
		ticker = s.animationController.AnimateTo(1.0, animation.AnimateOpts{Duration: foundation.Some(cupertinoButtonFadeOutDuration), Curve: animation.Curves.EaseInOutCubicEmphasized})
	} else {
		ticker = s.animationController.AnimateTo(0.0, animation.AnimateOpts{Duration: foundation.Some(cupertinoButtonFadeInDuration), Curve: animation.Curves.EaseOutCubic})
	}
	ticker.Then(func() {
		if s.Mounted() && wasHeldDown != s.buttonHeldDown {
			s.animate()
		}
	})
}

func (s *cupertinoButtonState) onShowFocusHighlight(show bool) {
	s.SetState(func() { s.isFocused = show })
}

func (s *cupertinoButtonState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	enabled := w.Enabled()
	theme := CupertinoThemeOf(bc)
	primaryColor := CupertinoDynamicColorResolve(theme.PrimaryColor(), bc)

	var background foundation.Opt[ui.Color]
	switch {
	case o.Color == nil && w.style != cupertinoButtonStylePlain:
		background = foundation.Some(primaryColor)
	case o.Color != nil:
		background = CupertinoDynamicColorMaybeResolve(o.Color, bc)
	}
	if background.IsSome() {
		var opacity float64
		switch {
		case w.style == cupertinoButtonStyleTinted && CupertinoThemeBrightnessOf(bc) == ui.BrightnessLight:
			opacity = CupertinoButtonTintedOpacityLight
		case w.style == cupertinoButtonStyleTinted:
			opacity = CupertinoButtonTintedOpacityDark
		case o.Color != nil:
			opacity = o.Color.EffectiveColor().Opacity()
		default:
			opacity = 1.0
		}
		background = foundation.Some(background.Value().WithOpacity(opacity))
	}

	var foreground ui.Color
	switch {
	case o.ForegroundColor != nil:
		foreground = CupertinoDynamicColorResolve(o.ForegroundColor, bc)
	case w.style == cupertinoButtonStyleFilled:
		foreground = CupertinoDynamicColorResolve(theme.PrimaryContrastingColor(), bc)
	case enabled:
		foreground = primaryColor
	default:
		foreground = CupertinoDynamicColorResolve(CupertinoColors.TertiaryLabel, bc)
	}

	var focusOutline ui.Color
	if o.FocusColor != nil {
		focusOutline = CupertinoDynamicColorResolve(o.FocusColor, bc)
	} else {
		base := CupertinoColors.ActiveBlue.Color
		if background.IsSome() {
			base = background.Value()
		}
		focusOutline = cupertinoFocusOutlineColor(base)
	}

	textTheme := theme.TextTheme()
	base := textTheme.ActionTextStyle()
	if o.SizeStyle == CupertinoButtonSizeSmall {
		base = textTheme.ActionSmallTextStyle()
	}
	textStyle := base.CopyWith(painting.TextStyleCopy{Color: foundation.Some(foreground)})
	iconSize := CupertinoButtonDefaultIconSize
	if textStyle.FontSize.IsSome() {
		iconSize = textStyle.FontSize.Value() * 1.2
	}
	iconTheme := widgets.IconThemeOf(bc).CopyWith(widgets.IconThemeDataOpts{Color: foundation.Some(foreground), Size: foundation.Some(iconSize)})

	var settings foundation.Opt[gestures.DeviceGestureSettings]
	if gs, ok := widgets.MediaQueryMaybeGestureSettingsOf(bc); ok {
		settings = foundation.Some(gs)
	}

	states := widgets.NewWidgetStates()
	if !enabled {
		states = states.With(widgets.WidgetStateDisabled)
	}
	if s.tapInProgress {
		states = states.With(widgets.WidgetStatePressed)
	}
	if s.isFocused {
		states = states.With(widgets.WidgetStateFocused)
	}
	var cursor services.MouseCursor = services.MouseCursorDefer
	if o.MouseCursor != nil {
		cursor = widgets.WidgetStateResolveAs[services.MouseCursor](o.MouseCursor, states)
	} else if !states.Has(widgets.WidgetStateDisabled) && foundation.IsWeb {
		cursor = services.SystemMouseCursors.Click
	}

	side := painting.BorderSideNone
	if enabled && s.isFocused {
		side = painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(focusOutline), Width: foundation.Some(3.5), StrokeAlign: foundation.Some(painting.StrokeAlignOutside)})
	}
	var radius painting.BorderRadiusGeometry = CupertinoButtonSizeBorderRadius(o.SizeStyle)
	if o.BorderRadius.IsSome() {
		radius = o.BorderRadius.Value()
	}
	var fill foundation.Opt[ui.Color] = background
	if background.IsSome() && !enabled {
		fill = foundation.Some(CupertinoDynamicColorResolve(o.DisabledColor, bc))
	}
	decoration := painting.NewShapeDecoration(painting.NewRoundedSuperellipseBorder(painting.RoundedSuperellipseBorderOpts{Side: side, BorderRadius: radius}),
		painting.ShapeDecorationOpts{Color: fill})

	minWidth, minHeight := CupertinoButtonMinSize(o.SizeStyle), CupertinoButtonMinSize(o.SizeStyle)
	if o.MinimumSize.IsSome() {
		minWidth, minHeight = o.MinimumSize.Value().Width, o.MinimumSize.Value().Height
	}
	var padding painting.EdgeInsetsGeometry = CupertinoButtonPadding(o.SizeStyle)
	if o.Padding != nil {
		padding = o.Padding
	}

	gesturesMap := map[reflect.Type]widgets.GestureRecognizerFactory{}
	tap := widgets.NewGestureRecognizerFactoryWithHandlers(
		func() *gestures.TapGestureRecognizer {
			return gestures.NewTapGestureRecognizer(widgets.WidgetsBindingOf(bc).Gestures(), gestures.TapGestureRecognizerOpts{
				DebugOwner: w, PostAcceptSlopTolerance: gestures.SlopToleranceNone})
		},
		func(r *gestures.TapGestureRecognizer) {
			r.OnTapDown, r.OnTapUp, r.OnTapCancel, r.OnTapMove = nil, nil, nil, nil
			if enabled {
				r.OnTapDown, r.OnTapUp, r.OnTapCancel, r.OnTapMove = s.handleTapDown, s.handleTapUp, s.handleTapCancel, s.handleTapMove
			}
			r.Base().GestureSettings = settings
		})
	gesturesMap[tap.RecognizerType()] = tap
	if o.OnLongPress != nil {
		longPress := widgets.NewGestureRecognizerFactoryWithHandlers(
			func() *gestures.LongPressGestureRecognizer {
				return gestures.NewLongPressGestureRecognizer(widgets.WidgetsBindingOf(bc).Gestures(), gestures.LongPressGestureRecognizerOpts{DebugOwner: w})
			},
			func(r *gestures.LongPressGestureRecognizer) {
				r.OnLongPress = o.OnLongPress
				r.Base().GestureSettings = settings
			})
		gesturesMap[longPress.RecognizerType()] = longPress
	}

	return widgets.NewMouseRegion(widgets.NewFocusableActionDetector(
		widgets.NewRawGestureDetector(
			widgets.NewSemantics(widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinWidth: foundation.Some(minWidth), MinHeight: foundation.Some(minHeight)}),
				widgets.NewFadeTransition(s.opacityAnimation, widgets.NewDecoratedBox(decoration,
					widgets.NewPadding(padding, widgets.NewAlign(
						widgets.NewDefaultTextStyle(textStyle, widgets.NewIconTheme(iconTheme, w.child)),
						widgets.AlignOpts{Alignment: o.Alignment, WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)}))))),
				widgets.SemanticsOpts{Button: foundation.Some(true)}),
			widgets.RawGestureDetectorOpts{Behavior: rendering.HitTestBehaviorOpaque, Gestures: gesturesMap}),
		widgets.FocusableActionDetectorOpts{
			Actions:   widgets.ActionMap{reflect.TypeFor[widgets.ActivateIntent](): widgets.NewCallbackAction(func(widgets.ActivateIntent) any { s.handleTap(); return nil })},
			FocusNode: o.FocusNode, Autofocus: o.Autofocus, OnFocusChange: o.OnFocusChange, OnShowFocusHighlight: s.onShowFocusHighlight, Disabled: !enabled,
		}), widgets.MouseRegionOpts{Cursor: cursor})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoButtonPressedOpacityInvalid = "CupertinoButtonPressedOpacityInvalid"
)
