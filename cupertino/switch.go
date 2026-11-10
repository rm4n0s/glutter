package cupertino

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The measures of the switch (the constants of cupertino/switch.dart).
const (
	switchDisabledOpacity      = 0.5
	switchThumbRadius          = 14.0
	switchTrackHeight          = 31.0
	switchTrackWidth           = 51.0
	switchThumbExtensionFactor = 7.0
	// switchDragCommitThreshold is the part of the track that a drag must cover to turn the switch.
	switchDragCommitThreshold = 0.7
	// switchDragReverseThreshold is the part that a drag must cover back to undo the turn it did.
	switchDragReverseThreshold = 0.2

	switchOnLabelWidth           = 1.0
	switchOnLabelHeight          = 10.0
	switchOnLabelPaddingHorizont = 11.0
	switchOffLabelWidth          = 1.0
	switchOffLabelPaddingHoriz   = 12.0
	switchOffLabelRadius         = 5.0
)

// switchSize is the size of the switch with the space around its track.
var switchSize = ui.Size{Width: 59.0, Height: 39.0}

// switchOffLabelColor is the default color of the label of a switch that is off.
var switchOffLabelColor = NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFB3B3B3), ui.NewColor(0xFFB3B3B3), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFFFFFFFF),
	CupertinoDynamicColorOpts{DebugLabel: "offSwitchLabel"})

// CupertinoSwitchOpts are the optional parameters of NewCupertinoSwitch.
//
// Flutter: cupertino/switch.dart CupertinoSwitch.new
type CupertinoSwitchOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ActiveTrackColor is the color of the track when the switch is on; the default is the primary color of the
	// theme when it applies to all the widgets, and the system green otherwise.
	ActiveTrackColor ui.ColorValue
	// InactiveTrackColor is the color of the track when the switch is off.
	InactiveTrackColor ui.ColorValue
	// ThumbColor is the color of the thumb when the switch is on; the default is white.
	ThumbColor ui.ColorValue
	// InactiveThumbColor is the color of the thumb when the switch is off; the default is ThumbColor.
	InactiveThumbColor ui.ColorValue
	// ApplyTheme says whether the primary color of the theme is the color of the track when it is on; the
	// default is the theme's ApplyThemeToAll.
	ApplyTheme foundation.Opt[bool]
	// FocusColor is the color of the outline of a switch that has the focus.
	FocusColor ui.ColorValue
	// OnLabelColor is the color of the "on" label (a bar) that the switch shows when the user asked for labels.
	OnLabelColor ui.ColorValue
	// OffLabelColor is the color of the "off" label (a ring).
	OffLabelColor ui.ColorValue
	// ActiveThumbImage is an image in the thumb when the switch is on.
	ActiveThumbImage painting.ImageProvider
	// OnActiveThumbImageError is called when ActiveThumbImage fails to load; it needs ActiveThumbImage.
	OnActiveThumbImageError painting.ImageErrorListener
	// InactiveThumbImage is an image in the thumb when the switch is off.
	InactiveThumbImage painting.ImageProvider
	// OnInactiveThumbImageError is called when InactiveThumbImage fails to load; it needs InactiveThumbImage.
	OnInactiveThumbImageError painting.ImageErrorListener
	// TrackOutlineColor is the color of the outline of the track, by state.
	TrackOutlineColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// TrackOutlineWidth is the width of the outline of the track, by state.
	TrackOutlineWidth widgets.WidgetStateProperty[foundation.Opt[float64]]
	// ThumbIcon is the icon in the thumb, by state.
	ThumbIcon widgets.WidgetStateProperty[*widgets.Icon]
	// MouseCursor is the cursor over the switch, by state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// FocusNode is the focus node of the switch.
	FocusNode *widgets.FocusNode
	// OnFocusChange is called when the focus of the switch changes.
	OnFocusChange func(focused bool)
	// Autofocus asks for the focus when the switch is built.
	Autofocus bool
	// DragStartBehavior is how the drag starts; the default is start.
	DragStartBehavior gestures.DragStartBehavior
	// Deviation: no activeColor and trackColor (deprecated in Flutter, replaced by ActiveTrackColor and
	// InactiveTrackColor), and a color that depends on the state is given with the Material form of a switch.
}

// CupertinoSwitch is an iOS styled switch: a track with a round thumb that slides, turns when it is tapped or
// dragged, and gives a light haptic feedback on iOS. It does not keep its value: the widget that makes it
// listens to OnChanged and rebuilds it with the new value; a switch with no callback is disabled.
//
// Flutter: cupertino/switch.dart CupertinoSwitch
type CupertinoSwitch struct {
	widgets.StatefulWidget
	value     bool
	onChanged func(bool)
	o         CupertinoSwitchOpts
}

// NewCupertinoSwitch creates a switch with the value; OnChanged nil makes it disabled. In debug builds it panics with
// the tags CupertinoSwitchActiveThumbImageErrorWithoutImage and CupertinoSwitchInactiveThumbImageErrorWithoutImage.
//
// Flutter: cupertino/switch.dart CupertinoSwitch.new
func NewCupertinoSwitch(value bool, onChanged func(bool), opts ...CupertinoSwitchOpts) *CupertinoSwitch {
	o := firstOf(opts)
	if foundation.DebugMode {
		if o.ActiveThumbImage == nil && o.OnActiveThumbImageError != nil {
			panic(errors.New(tagCupertinoSwitchActiveThumbImageErrorWithoutImage, "onActiveThumbImageError needs an activeThumbImage"))
		}
		if o.InactiveThumbImage == nil && o.OnInactiveThumbImageError != nil {
			panic(errors.New(tagCupertinoSwitchInactiveThumbImageErrorWithoutImage, "onInactiveThumbImageError needs an inactiveThumbImage"))
		}
	}
	return &CupertinoSwitch{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), value, onChanged, o}
}

// Value is whether the switch is on.
//
// Flutter: cupertino/switch.dart CupertinoSwitch.value
func (w *CupertinoSwitch) Value() bool { return w.value }

// OnChanged is the callback for a change, or nil for a disabled switch.
//
// Flutter: cupertino/switch.dart CupertinoSwitch.onChanged
func (w *CupertinoSwitch) OnChanged() func(bool) { return w.onChanged }

// ActiveTrackColor is the color of the track when the switch is on, or nil.
//
// Flutter: cupertino/switch.dart CupertinoSwitch.activeTrackColor
func (w *CupertinoSwitch) ActiveTrackColor() ui.ColorValue { return w.o.ActiveTrackColor }

// InactiveTrackColor is the color of the track when the switch is off, or nil.
//
// Flutter: cupertino/switch.dart CupertinoSwitch.inactiveTrackColor
func (w *CupertinoSwitch) InactiveTrackColor() ui.ColorValue { return w.o.InactiveTrackColor }

// ThumbColor is the color of the thumb when the switch is on, or nil.
//
// Flutter: cupertino/switch.dart CupertinoSwitch.thumbColor
func (w *CupertinoSwitch) ThumbColor() ui.ColorValue { return w.o.ThumbColor }

// FocusColor is the color of the focus outline, or nil.
//
// Flutter: cupertino/switch.dart CupertinoSwitch.focusColor
func (w *CupertinoSwitch) FocusColor() ui.ColorValue { return w.o.FocusColor }

// CreateState creates the state.
//
// Flutter: cupertino/switch.dart CupertinoSwitch.createState
func (w *CupertinoSwitch) CreateState() widgets.State { return &cupertinoSwitchState{} }

type cupertinoSwitchState struct {
	widgets.StateOf[*CupertinoSwitch]
	widgets.TickerProvider
	widgets.ToggleableStateMixin
	painter           *cupertinoSwitchPainter
	dragStartPosition foundation.Opt[ui.Offset]
	dragDelta         float64
	dragValue         foundation.Opt[bool]
	needsPositionAnim bool
}

func (s *cupertinoSwitchState) InitState() {
	s.painter = &cupertinoSwitchPainter{}
	s.Init(s, s)
	s.PositionController().Duration = foundation.Some(200 * millisecond)
	s.ReactionController().Duration = foundation.Some(300 * millisecond)
	position := s.Position()
	position.Curve = animation.Curves.Ease
	position.ReverseCurve = animation.Curves.Ease.Flipped()
}

func (s *cupertinoSwitchState) DidUpdateWidget(old *CupertinoSwitch) {
	if old.value != s.Widget().value {
		s.AnimateToValue()
	}
}

func (s *cupertinoSwitchState) Dispose() {
	s.painter.dispose()
	s.DisposeToggleable()
}

func (s *cupertinoSwitchState) ToggleableOnChanged() func(foundation.Opt[bool]) {
	if s.Widget().onChanged == nil {
		return nil
	}
	return s.handleChanged
}
func (s *cupertinoSwitchState) ToggleableValue() foundation.Opt[bool] {
	return foundation.Some(s.Widget().value)
}
func (s *cupertinoSwitchState) ToggleableTristate() bool { return false }

// switchTrackInnerLength is the length of the track that the center of the thumb moves along.
func switchTrackInnerLength() float64 {
	const trackInnerStart = switchTrackHeight / 2.0
	const trackInnerEnd = switchTrackWidth - trackInnerStart
	return trackInnerEnd - trackInnerStart
}

func (s *cupertinoSwitchState) handleOnTapDown(details gestures.TapDownDetails) {
	if s.IsInteractive() {
		s.dragStartPosition = foundation.Some(details.GlobalPosition)
	}
}

func (s *cupertinoSwitchState) directionSign() float64 {
	if widgets.DirectionalityOf(s.Context()) == ui.TextDirectionRTL {
		return -1
	}
	return 1
}

func (s *cupertinoSwitchState) handleDragStart(details gestures.DragStartDetails) {
	if !s.IsInteractive() {
		return
	}
	s.ReactionController().Forward()
	if start, ok := s.dragStartPosition.Get(); ok && start != (ui.Offset{}) {
		delta := (details.GlobalPosition.Dx - start.Dx) / switchTrackWidth
		s.dragDelta = s.directionSign() * delta
	}
	s.dragValue = foundation.Some(s.Widget().value)
}

func (s *cupertinoSwitchState) handleDragUpdate(details gestures.DragUpdateDetails) {
	if !s.IsInteractive() {
		return
	}
	w := s.Widget()
	delta := details.PrimaryDelta.OrElse(0) / switchTrackWidth
	s.dragDelta += s.directionSign() * delta

	valueChangedWhileDragging := foundation.Some(w.value) != s.dragValue
	threshold := switchDragCommitThreshold
	if valueChangedWhileDragging {
		threshold = switchDragReverseThreshold
	}
	effectiveThreshold := threshold
	if w.value {
		effectiveThreshold = -threshold
	}
	newDragValue := s.dragDelta >= effectiveThreshold
	if s.dragValue != foundation.Some(newDragValue) {
		s.emitVibration()
		if newDragValue {
			s.PositionController().Forward()
		} else {
			s.PositionController().Reverse()
		}
		s.dragValue = foundation.Some(newDragValue)
	}
}

func (s *cupertinoSwitchState) handleDragEnd(gestures.DragEndDetails) {
	w := s.Widget()
	if s.dragValue != foundation.Some(w.value) {
		if w.onChanged != nil {
			w.onChanged(!w.value)
		}
		s.SetState(func() { s.needsPositionAnim = true })
	}
	s.dragStartPosition = foundation.None[ui.Offset]()
	s.dragDelta = 0
	s.dragValue = foundation.None[bool]()
	s.ReactionController().Reverse()
}

func (s *cupertinoSwitchState) handleChanged(value foundation.Opt[bool]) {
	if f := s.Widget().onChanged; f != nil {
		f(value.Value())
	}
	s.emitVibration()
}

func (s *cupertinoSwitchState) emitVibration() {
	if foundation.DefaultTargetPlatform() == foundation.TargetPlatformIOS {
		widgets.WidgetsBindingOf(s.Context()).HapticFeedback().LightImpact()
	}
}

func (s *cupertinoSwitchState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	if s.needsPositionAnim {
		s.needsPositionAnim = false
		s.AnimateToValue()
	}
	theme := CupertinoThemeOf(bc)

	var activeColor ui.Color
	switch {
	case w.o.ActiveTrackColor != nil:
		activeColor = CupertinoDynamicColorResolve(w.o.ActiveTrackColor, bc)
	case w.o.ApplyTheme.OrElse(theme.ApplyThemeToAll()):
		activeColor = CupertinoDynamicColorResolve(theme.PrimaryColor(), bc)
	default:
		activeColor = CupertinoDynamicColorResolve(CupertinoColors.SystemGreen, bc)
	}

	var onOffLabelColors foundation.Opt[[2]ui.Color]
	if widgets.MediaQueryOf(bc).OnOffSwitchLabels {
		on := ui.ColorValue(CupertinoColors.White)
		if w.o.OnLabelColor != nil {
			on = w.o.OnLabelColor
		}
		off := ui.ColorValue(switchOffLabelColor)
		if w.o.OffLabelColor != nil {
			off = w.o.OffLabelColor
		}
		onOffLabelColors = foundation.Some([2]ui.Color{CupertinoDynamicColorResolve(on, bc), CupertinoDynamicColorResolve(off, bc)})
	}

	states := s.States()
	activeStates := states.With(widgets.WidgetStateSelected)
	inactiveStates := states.Without(widgets.WidgetStateSelected)

	resolveColor := func(c ui.ColorValue) foundation.Opt[ui.Color] {
		if c == nil {
			return foundation.None[ui.Color]()
		}
		return foundation.Some(CupertinoDynamicColorResolve(c, bc))
	}
	thumbColor := func(selected bool) foundation.Opt[ui.Color] {
		if selected {
			return resolveColor(w.o.ThumbColor)
		}
		return resolveColor(w.o.InactiveThumbColor)
	}
	effectiveActiveThumbColor := thumbColor(true).OrElse(CupertinoColors.White)
	effectiveInactiveThumbColor := thumbColor(false).OrElse(effectiveActiveThumbColor)
	effectiveActiveTrackColor := activeColor
	if w.o.ActiveTrackColor != nil {
		effectiveActiveTrackColor = CupertinoDynamicColorResolve(w.o.ActiveTrackColor, bc)
	}
	effectiveInactiveTrackColor := resolveColor(w.o.InactiveTrackColor).OrElse(CupertinoDynamicColorResolve(CupertinoColors.SecondarySystemFill, bc))

	outlineColor := func(st widgets.WidgetStates) foundation.Opt[ui.Color] {
		if w.o.TrackOutlineColor == nil {
			return foundation.None[ui.Color]()
		}
		return w.o.TrackOutlineColor.Resolve(st)
	}
	outlineWidth := func(st widgets.WidgetStates) foundation.Opt[float64] {
		if w.o.TrackOutlineWidth == nil {
			return foundation.None[float64]()
		}
		return w.o.TrackOutlineWidth.Resolve(st)
	}
	icon := func(st widgets.WidgetStates) *widgets.Icon {
		if w.o.ThumbIcon == nil {
			return nil
		}
		return w.o.ThumbIcon.Resolve(st)
	}
	effectiveActiveIcon, effectiveInactiveIcon := icon(activeStates), icon(inactiveStates)
	// The painter draws the glyph as text, so the font of the icons is loaded here (an Icon widget would do it).
	for _, i := range []*widgets.Icon{effectiveActiveIcon, effectiveInactiveIcon} {
		if i != nil {
			widgets.IconFontEnsure(bc, i.Icon())
		}
	}
	iconColor := func(i *widgets.Icon) ui.Color {
		if i != nil {
			if c, ok := i.Color().Get(); ok {
				return c
			}
		}
		return CupertinoColors.Black
	}

	// The thumb that is pressed has the color that is given for the on state, then the one for the off state.
	effectiveActivePressedThumbColor := thumbColor(true).OrElse(CupertinoColors.White)
	effectiveInactivePressedThumbColor := thumbColor(true).OrElse(thumbColor(false).OrElse(CupertinoColors.White))

	mouseCursor := w.o.MouseCursor
	if mouseCursor == nil {
		mouseCursor = widgets.WidgetStatePropertyResolveWith(func(st widgets.WidgetStates) services.MouseCursor {
			if st.Has(widgets.WidgetStateDisabled) || !foundation.IsWeb {
				return services.MouseCursorDefer
			}
			return services.SystemMouseCursors.Click
		})
	}

	focusColor := activeColor.WithOpacity(CupertinoFocusColorOpacity)
	focus := painting.NewHSLColorFromColor(focusColor).WithLightness(CupertinoFocusColorBrightness).WithSaturation(CupertinoFocusColorSaturation).ToColor()
	if w.o.FocusColor != nil {
		focus = CupertinoDynamicColorResolve(w.o.FocusColor, bc)
	}

	var engine ui.ParagraphFactory
	if b := widgets.MaybeWidgetsBindingOf(bc); b != nil {
		engine = b.PlatformDispatcher().TextEngine()
	}

	p := s.painter
	p.engine = engine
	p.SetPosition(s.Position())
	p.SetReaction(s.Reaction())
	p.SetReactionFocusFade(s.ReactionFocusFade())
	p.SetReactionHoverFade(s.ReactionHoverFade())
	p.SetFocusColor(focus)
	p.SetDownPosition(s.DownPosition())
	p.SetIsFocused(states.Has(widgets.WidgetStateFocused))
	p.SetIsHovered(states.Has(widgets.WidgetStateHovered))
	p.SetActiveColor(effectiveActiveThumbColor)
	p.SetInactiveColor(effectiveInactiveThumbColor)
	p.set(cupertinoSwitchPainterValues{
		activePressedColor: effectiveActivePressedThumbColor, inactivePressedColor: effectiveInactivePressedThumbColor,
		onOffLabelColors: onOffLabelColors,
		activeThumbImage: w.o.ActiveThumbImage, onActiveThumbImageError: w.o.OnActiveThumbImageError,
		inactiveThumbImage: w.o.InactiveThumbImage, onInactiveThumbImageError: w.o.OnInactiveThumbImageError,
		activeTrackColor: effectiveActiveTrackColor, activeTrackOutlineColor: outlineColor(activeStates), activeTrackOutlineWidth: outlineWidth(activeStates),
		inactiveTrackColor: effectiveInactiveTrackColor, inactiveTrackOutlineColor: outlineColor(inactiveStates), inactiveTrackOutlineWidth: outlineWidth(inactiveStates),
		configuration: widgets.CreateLocalImageConfiguration(bc, foundation.None[ui.Size]()), isInteractive: s.IsInteractive(),
		trackInnerLength: switchTrackInnerLength(), textDirection: widgets.DirectionalityOf(bc),
		activeIconColor: iconColor(effectiveActiveIcon), inactiveIconColor: iconColor(effectiveInactiveIcon),
		activeIcon: effectiveActiveIcon, inactiveIcon: effectiveInactiveIcon, iconTheme: widgets.IconThemeOf(bc),
		surfaceColor: theme.ScaffoldBackgroundColor().EffectiveColor(),
	})
	p.setPositionController(s.PositionController())

	opacity := 1.0
	if s.ToggleableOnChanged() == nil {
		opacity = switchDisabledOpacity
	}
	return widgets.NewSemantics(
		widgets.NewGestureDetector(widgets.NewOpacity(opacity, s.BuildToggleable(switchSize, p, widgets.ToggleableOpts{
			MouseCursor: mouseCursor, FocusNode: w.o.FocusNode, OnFocusChange: w.o.OnFocusChange, Autofocus: w.o.Autofocus})),
			widgets.GestureDetectorOpts{ExcludeFromSemantics: true, OnTapDown: s.handleOnTapDown, OnHorizontalDragStart: s.handleDragStart,
				OnHorizontalDragUpdate: s.handleDragUpdate, OnHorizontalDragEnd: s.handleDragEnd, DragStartBehavior: w.o.DragStartBehavior}),
		widgets.SemanticsOpts{Toggled: foundation.Some(w.value)})
}

// cupertinoSwitchPainterValues are the values of a cupertinoSwitchPainter that the state sets in every build.
type cupertinoSwitchPainterValues struct {
	activePressedColor, inactivePressedColor           ui.Color
	onOffLabelColors                                   foundation.Opt[[2]ui.Color]
	activeThumbImage, inactiveThumbImage               painting.ImageProvider
	onActiveThumbImageError, onInactiveThumbImageError painting.ImageErrorListener
	activeTrackColor, inactiveTrackColor               ui.Color
	activeTrackOutlineColor, inactiveTrackOutlineColor foundation.Opt[ui.Color]
	activeTrackOutlineWidth, inactiveTrackOutlineWidth foundation.Opt[float64]
	configuration                                      painting.ImageConfiguration
	isInteractive                                      bool
	trackInnerLength                                   float64
	textDirection                                      ui.TextDirection
	activeIconColor, inactiveIconColor                 ui.Color
	activeIcon, inactiveIcon                           *widgets.Icon
	iconTheme                                          widgets.IconThemeData
	surfaceColor                                       ui.Color
}

func (a cupertinoSwitchPainterValues) equal(b cupertinoSwitchPainterValues) bool {
	return a.activePressedColor == b.activePressedColor && a.inactivePressedColor == b.inactivePressedColor && a.onOffLabelColors == b.onOffLabelColors &&
		a.activeThumbImage == b.activeThumbImage && a.inactiveThumbImage == b.inactiveThumbImage &&
		(a.onActiveThumbImageError == nil) == (b.onActiveThumbImageError == nil) && (a.onInactiveThumbImageError == nil) == (b.onInactiveThumbImageError == nil) &&
		a.activeTrackColor == b.activeTrackColor && a.inactiveTrackColor == b.inactiveTrackColor &&
		a.activeTrackOutlineColor == b.activeTrackOutlineColor && a.inactiveTrackOutlineColor == b.inactiveTrackOutlineColor &&
		a.activeTrackOutlineWidth == b.activeTrackOutlineWidth && a.inactiveTrackOutlineWidth == b.inactiveTrackOutlineWidth &&
		a.configuration == b.configuration && a.isInteractive == b.isInteractive && a.trackInnerLength == b.trackInnerLength &&
		a.textDirection == b.textDirection && a.activeIconColor == b.activeIconColor && a.inactiveIconColor == b.inactiveIconColor &&
		a.activeIcon == b.activeIcon && a.inactiveIcon == b.inactiveIcon && a.iconTheme.Equal(b.iconTheme) && a.surfaceColor == b.surfaceColor
}

// cupertinoSwitchPainter paints the track, the labels and the thumb of a CupertinoSwitch.
//
// Flutter: cupertino/switch.dart _SwitchPainter
type cupertinoSwitchPainter struct {
	widgets.ToggleablePainter
	engine ui.ParagraphFactory
	v      cupertinoSwitchPainterValues

	positionController *animation.AnimationController
	colorAnimation     *animation.CurvedAnimation

	cachedThumbColor         ui.Color
	cachedThumbImage         painting.ImageProvider
	cachedThumbErrorListener bool
	cachedThumbPainter       painting.BoxPainter
	isPainting               bool

	stopPressAnimation    bool
	pressedThumbExtension float64
}

func (p *cupertinoSwitchPainter) set(v cupertinoSwitchPainterValues) {
	if !p.v.equal(v) {
		p.v = v
		p.NotifyListeners()
	}
}

func (p *cupertinoSwitchPainter) setPositionController(c *animation.AnimationController) {
	if p.positionController == c {
		return
	}
	p.positionController = c
	if p.colorAnimation != nil {
		p.colorAnimation.Dispose()
	}
	p.colorAnimation = animation.NewCurvedAnimation(c, animation.Curves.EaseOut, animation.Curves.EaseIn)
	p.NotifyListeners()
}

func (p *cupertinoSwitchPainter) handleDecorationChanged() {
	if !p.isPainting {
		p.NotifyListeners()
	}
}

func (p *cupertinoSwitchPainter) createDefaultThumbDecoration(color ui.Color, image painting.ImageProvider, onError painting.ImageErrorListener) *painting.ShapeDecoration {
	var source painting.DecorationImageSource
	if image != nil {
		source = painting.NewDecorationImage(image, painting.DecorationImageOpts{OnError: onError})
	}
	return painting.NewShapeDecoration(painting.NewStadiumBorder(), painting.ShapeDecorationOpts{Color: foundation.Some(color), Image: source})
}

// Paint paints the track, the labels and the thumb.
//
// Flutter: cupertino/switch.dart _SwitchPainter.paint
func (p *cupertinoSwitchPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	v := &p.v
	position, reaction := p.Position(), p.Reaction()
	currentValue := position.Value()
	visualPosition := currentValue
	if v.textDirection == ui.TextDirectionRTL {
		visualPosition = 1.0 - currentValue
	}
	p.stopPressAnimation = reaction.Status() == animation.AnimationStatusReverse && !p.stopPressAnimation

	p.pressedThumbExtension = reaction.Value() * switchThumbExtensionFactor
	thumbSize := ui.Size{Width: switchThumbRadius*2 + p.pressedThumbExtension, Height: switchThumbRadius * 2}

	colorValue := p.colorAnimation.Value()
	trackColor := lerpColor(v.inactiveTrackColor, v.activeTrackColor, position.Value())
	var trackOutlineColor foundation.Opt[ui.Color]
	if in, ok := v.inactiveTrackOutlineColor.Get(); ok {
		if ac, ok := v.activeTrackOutlineColor.Get(); ok {
			trackOutlineColor = foundation.Some(lerpColor(in, ac, colorValue))
		}
	}
	trackOutlineWidth := lerpOptDouble(v.inactiveTrackOutlineWidth, v.activeTrackOutlineWidth, colorValue)

	var lerpedThumbColor ui.Color
	switch {
	case reaction.Status() != animation.AnimationStatusDismissed:
		lerpedThumbColor = lerpColor(v.inactivePressedColor, v.activePressedColor, colorValue)
	case p.positionController.Status() == animation.AnimationStatusForward:
		lerpedThumbColor = lerpColor(v.inactivePressedColor, p.ActiveColor(), colorValue)
	case p.positionController.Status() == animation.AnimationStatusReverse:
		lerpedThumbColor = lerpColor(p.InactiveColor(), v.activePressedColor, colorValue)
	default:
		lerpedThumbColor = lerpColor(p.InactiveColor(), p.ActiveColor(), colorValue)
	}
	thumbColor := ui.ColorAlphaBlend(lerpedThumbColor, v.surfaceColor)

	thumbIcon, thumbImage, thumbErrorListener := v.activeIcon, v.activeThumbImage, v.onActiveThumbImageError
	if currentValue < 0.5 {
		thumbIcon, thumbImage, thumbErrorListener = v.inactiveIcon, v.inactiveThumbImage, v.onInactiveThumbImageError
	}

	paint := ui.NewPaint()
	paint.SetColor(trackColor)

	trackPaintOffset := ui.Offset{Dx: (size.Width - switchTrackWidth) / 2.0, Dy: (size.Height - switchTrackHeight) / 2.0}
	thumbPaintOffset := p.computeThumbPaintOffset(trackPaintOffset, thumbSize, visualPosition)
	trackRect := ui.NewRectFromLTWH(trackPaintOffset.Dx, trackPaintOffset.Dy, switchTrackWidth, switchTrackHeight)

	p.paintTrackWith(canvas, paint, trackPaintOffset, trackOutlineColor, trackOutlineWidth, trackRect)

	currentReactionValue := reaction.Value()
	if labels, ok := v.onOffLabelColors.Get(); ok {
		onLabelColor, offLabelColor := labels[0], labels[1]
		leftLabelOpacity := visualPosition * (1.0 - currentReactionValue)
		rightLabelOpacity := (1.0 - visualPosition) * (1.0 - currentReactionValue)
		var onLabelOpacity, offLabelOpacity float64
		var onLabelOffset, offLabelOffset ui.Offset
		if v.textDirection == ui.TextDirectionRTL {
			onLabelOpacity, offLabelOpacity = rightLabelOpacity, leftLabelOpacity
			onLabelOffset = trackRect.CenterRight().Translate(-switchOnLabelPaddingHorizont, 0)
			offLabelOffset = trackRect.CenterLeft().Translate(switchOffLabelPaddingHoriz, 0)
		} else {
			onLabelOpacity, offLabelOpacity = leftLabelOpacity, rightLabelOpacity
			onLabelOffset = trackRect.CenterLeft().Translate(switchOnLabelPaddingHorizont, 0)
			offLabelOffset = trackRect.CenterRight().Translate(-switchOffLabelPaddingHoriz, 0)
		}
		onLabelRect := ui.NewRectFromCenter(onLabelOffset, switchOnLabelWidth, switchOnLabelHeight)
		onLabelPaint := ui.NewPaint()
		onLabelPaint.SetColor(onLabelColor.WithOpacity(onLabelOpacity))
		onLabelPaint.SetStyle(ui.PaintingStyleFill)
		canvas.DrawRect(onLabelRect, onLabelPaint)

		offLabelPaint := ui.NewPaint()
		offLabelPaint.SetColor(offLabelColor.WithOpacity(offLabelOpacity))
		offLabelPaint.SetStyle(ui.PaintingStyleStroke)
		offLabelPaint.SetStrokeWidth(switchOffLabelWidth)
		canvas.DrawCircle(offLabelOffset, switchOffLabelRadius, offLabelPaint)
	}
	p.paintThumbWith(thumbPaintOffset, canvas, colorValue, thumbColor, thumbImage, thumbErrorListener, thumbIcon, thumbSize)
}

func (p *cupertinoSwitchPainter) computeThumbPaintOffset(trackPaintOffset ui.Offset, thumbSize ui.Size, visualPosition float64) ui.Offset {
	const trackRadius = switchTrackHeight / 2
	additionalThumbRadius := thumbSize.Height/2 - trackRadius
	horizontalProgress := visualPosition * (p.v.trackInnerLength - p.pressedThumbExtension)
	thumbHorizontalOffset := trackPaintOffset.Dx + trackRadius + (p.pressedThumbExtension / 2) - thumbSize.Width/2 + horizontalProgress
	thumbVerticalOffset := trackPaintOffset.Dy - additionalThumbRadius
	return ui.Offset{Dx: thumbHorizontalOffset, Dy: thumbVerticalOffset}
}

func (p *cupertinoSwitchPainter) paintTrackWith(canvas *ui.Canvas, paint *ui.Paint, trackPaintOffset ui.Offset, trackOutlineColor foundation.Opt[ui.Color],
	trackOutlineWidth foundation.Opt[float64], trackRect ui.Rect) {
	const trackRadius = switchTrackHeight / 2
	trackRRect := ui.NewRRectFromRectAndRadius(trackRect, ui.NewRadiusCircular(trackRadius))
	canvas.DrawRRect(trackRRect, paint)

	if c, ok := trackOutlineColor.Get(); ok {
		outlineTrackRect := ui.NewRectFromLTWH(trackPaintOffset.Dx+1, trackPaintOffset.Dy+1, switchTrackWidth-2, switchTrackHeight-2)
		outlinePaint := ui.NewPaint()
		outlinePaint.SetStyle(ui.PaintingStyleStroke)
		outlinePaint.SetStrokeWidth(trackOutlineWidth.OrElse(2.0))
		outlinePaint.SetColor(c)
		canvas.DrawRRect(ui.NewRRectFromRectAndRadius(outlineTrackRect, ui.NewRadiusCircular(trackRadius)), outlinePaint)
	}
	if p.IsFocused() {
		focusedPaint := ui.NewPaint()
		focusedPaint.SetStyle(ui.PaintingStyleStroke)
		focusedPaint.SetColor(p.FocusColor())
		focusedPaint.SetStrokeWidth(3.5)
		canvas.DrawRRect(trackRRect.Inflate(1.75), focusedPaint)
	}
	canvas.ClipRRect(trackRRect)
}

func (p *cupertinoSwitchPainter) paintThumbWith(thumbPaintOffset ui.Offset, canvas *ui.Canvas, currentValue float64, thumbColor ui.Color,
	thumbImage painting.ImageProvider, thumbErrorListener painting.ImageErrorListener, thumbIcon *widgets.Icon, thumbSize ui.Size) {
	v := &p.v
	p.isPainting = true
	defer func() { p.isPainting = false }()
	if p.cachedThumbPainter == nil || thumbColor != p.cachedThumbColor || thumbImage != p.cachedThumbImage || (thumbErrorListener != nil) != p.cachedThumbErrorListener {
		p.cachedThumbColor = thumbColor
		p.cachedThumbImage = thumbImage
		p.cachedThumbErrorListener = thumbErrorListener != nil
		if p.cachedThumbPainter != nil {
			p.cachedThumbPainter.Dispose()
		}
		p.cachedThumbPainter = p.createDefaultThumbDecoration(thumbColor, thumbImage, thumbErrorListener).CreateBoxPainter(p.handleDecorationChanged)
	}
	p.paintShadowAndBorder(canvas, thumbPaintOffset, thumbSize)
	p.cachedThumbPainter.Paint(canvas, thumbPaintOffset, v.configuration.CopyWith(painting.ImageConfigurationCopy{Size: foundation.Some(thumbSize)}))

	if thumbIcon != nil && !thumbIcon.Icon().IsZero() {
		iconColor := lerpColor(v.inactiveIconColor, v.activeIconColor, currentValue)
		iconSize := thumbIcon.Size().OrElse(16.0)
		iconData := thumbIcon.Icon()
		orTheme := func(a, b foundation.Opt[float64]) foundation.Opt[float64] {
			if a.IsSome() {
				return a
			}
			return b
		}
		weight := orTheme(thumbIcon.Weight(), v.iconTheme.Weight())
		fill := orTheme(thumbIcon.Fill(), v.iconTheme.Fill())
		grade := orTheme(thumbIcon.Grade(), v.iconTheme.Grade())
		opticalSize := orTheme(thumbIcon.OpticalSize(), v.iconTheme.OpticalSize())
		shadows := thumbIcon.Shadows()
		if shadows == nil {
			shadows = v.iconTheme.Shadows()
		}
		var variations []ui.FontVariation
		for _, fv := range []struct {
			axis string
			v    foundation.Opt[float64]
		}{{"FILL", fill}, {"wght", weight}, {"GRAD", grade}, {"opsz", opticalSize}} {
			if x, ok := fv.v.Get(); ok {
				variations = append(variations, ui.NewFontVariation(fv.axis, x))
			}
		}
		style := painting.NewTextStyle(painting.TextStyleOpts{
			FontVariations: variations, Color: foundation.Some(iconColor), FontSize: foundation.Some(iconSize),
			Inherit: foundation.Some(false), FontFamily: iconData.FontFamily, Package: iconData.FontPackage,
			FontFamilyFallback: iconData.FontFamilyFallback(), Shadows: shadows,
		})
		tp := painting.NewTextPainter(painting.TextPainterOpts{
			TextEngine: p.engine, TextDirection: v.textDirection,
			Text: painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(string(rune(iconData.CodePoint))), Style: style}),
		})
		tp.Layout(0, math.Inf(1))
		tp.Paint(canvas, thumbPaintOffset.Add(ui.Offset{Dx: (thumbSize.Width - iconSize) / 2, Dy: (thumbSize.Height - iconSize) / 2}))
	}
}

func (p *cupertinoSwitchPainter) paintShadowAndBorder(canvas *ui.Canvas, thumbPaintOffset ui.Offset, thumbSize ui.Size) {
	thumbBounds := ui.NewRRectFromLTRBR(thumbPaintOffset.Dx, thumbPaintOffset.Dy, thumbPaintOffset.Dx+thumbSize.Width, thumbPaintOffset.Dy+thumbSize.Height,
		ui.NewRadiusCircular(thumbSize.Height/2.0))
	for _, shadow := range switchBoxShadows() {
		canvas.DrawRRect(thumbBounds.Shift(shadow.Offset), shadow.ToPaint())
	}
	border := ui.NewPaint()
	border.SetColor(thumbBorderColor)
	canvas.DrawRRect(thumbBounds.Inflate(0.5), border)
}

func (p *cupertinoSwitchPainter) dispose() {
	if p.cachedThumbPainter != nil {
		p.cachedThumbPainter.Dispose()
		p.cachedThumbPainter = nil
	}
	if p.colorAnimation != nil {
		p.colorAnimation.Dispose()
	}
	p.Dispose()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoSwitchActiveThumbImageErrorWithoutImage   = "CupertinoSwitchActiveThumbImageErrorWithoutImage"
	tagCupertinoSwitchInactiveThumbImageErrorWithoutImage = "CupertinoSwitchInactiveThumbImageErrorWithoutImage"
)
