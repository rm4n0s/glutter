package cupertino

import (
	"math"
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The measures of the slider (the constants of cupertino/slider.dart).
const (
	sliderVelocityThreshold = 1.0
	sliderPadding           = 8.0
	sliderHeight            = 2.0 * (CupertinoThumbPainterRadius + sliderPadding)
	sliderWidth             = 176.0 // Matches Material Design slider.
	sliderAdjustmentUnit    = 0.1   // Matches iOS implementation of material slider.

	sliderDiscreteTransitionDuration = 500 * time.Millisecond
)

// CupertinoSliderOpts are the optional parameters of NewCupertinoSlider.
//
// Flutter: cupertino/slider.dart CupertinoSlider.new
type CupertinoSliderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnChangeStart is called when the user starts to select a new value, with the value at that moment.
	OnChangeStart func(value float64)
	// OnChangeEnd is called when the user is done selecting a new value, with the value that was selected.
	OnChangeEnd func(value float64)
	// Min is the minimum value; the default is 0.
	Min float64
	// Max is the maximum value; the default is 1.
	Max foundation.Opt[float64]
	// Divisions is the number of discrete divisions; when none the slider is continuous.
	Divisions foundation.Opt[int]
	// ActiveColor is the color of the track on the side of the minimum; the default is the primary color of the
	// theme.
	ActiveColor ui.ColorValue
	// ThumbColor is the color of the thumb; the default is CupertinoColors.White.
	ThumbColor ui.ColorValue
}

// CupertinoSlider is an iOS styled slider: it lets the user select a value from a range by dragging its thumb.
//
// Flutter: cupertino/slider.dart CupertinoSlider
type CupertinoSlider struct {
	widgets.StatefulWidget
	value     float64
	onChanged func(value float64)
	o         CupertinoSliderOpts
}

// NewCupertinoSlider creates a slider; a nil onChanged disables it. In debug builds it panics with the tag
// CupertinoSliderValueOutOfRange or CupertinoSliderDivisionsInvalid when the arguments are not valid.
//
// Flutter: cupertino/slider.dart CupertinoSlider.new
func NewCupertinoSlider(value float64, onChanged func(value float64), opts ...CupertinoSliderOpts) *CupertinoSlider {
	o := firstOf(opts)
	if foundation.DebugMode {
		switch {
		case value < o.Min || value > o.Max.OrElse(1.0):
			panic(errors.NewErrf(tagCupertinoSliderValueOutOfRange, "Value %v is not between minimum %v and maximum %v", value, o.Min, o.Max.OrElse(1.0)))
		case o.Divisions.IsSome() && o.Divisions.Value() <= 0:
			panic(errors.New(tagCupertinoSliderDivisionsInvalid, "the divisions of a CupertinoSlider must be greater than zero", "divisions", o.Divisions.Value()))
		}
	}
	return &CupertinoSlider{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), value, onChanged, o}
}

// Value is the currently selected value.
//
// Flutter: cupertino/slider.dart CupertinoSlider.value
func (w *CupertinoSlider) Value() float64 { return w.value }

// OnChanged is called when the user selects a new value; nil disables the slider.
//
// Flutter: cupertino/slider.dart CupertinoSlider.onChanged
func (w *CupertinoSlider) OnChanged() func(value float64) { return w.onChanged }

// Min is the minimum value.
//
// Flutter: cupertino/slider.dart CupertinoSlider.min
func (w *CupertinoSlider) Min() float64 { return w.o.Min }

// Max is the maximum value.
//
// Flutter: cupertino/slider.dart CupertinoSlider.max
func (w *CupertinoSlider) Max() float64 { return w.o.Max.OrElse(1.0) }

// Divisions is the number of discrete divisions, when there are some.
//
// Flutter: cupertino/slider.dart CupertinoSlider.divisions
func (w *CupertinoSlider) Divisions() foundation.Opt[int] { return w.o.Divisions }

// CreateState creates the state.
//
// Flutter: cupertino/slider.dart CupertinoSlider.createState
func (w *CupertinoSlider) CreateState() widgets.State { return &cupertinoSliderState{} }

type cupertinoSliderState struct {
	widgets.StateOf[*CupertinoSlider]
	widgets.TickerProvider
}

func (s *cupertinoSliderState) lerp(value float64) float64 {
	w := s.Widget()
	return ui.LerpDouble(w.o.Min, w.Max(), value)
}

func (s *cupertinoSliderState) handleChanged(value float64, isFastDrag bool) {
	w := s.Widget()
	lerpValue := s.lerp(value)
	isAtEdge := lerpValue == w.Max() || lerpValue == w.o.Min
	if lerpValue != w.value {
		if isAtEdge {
			s.emitHapticFeedback(isFastDrag)
		}
		w.onChanged(lerpValue)
	}
}

func (s *cupertinoSliderState) handleDragStart(value float64) {
	s.Widget().o.OnChangeStart(s.lerp(value))
}
func (s *cupertinoSliderState) handleDragEnd(value float64) { s.Widget().o.OnChangeEnd(s.lerp(value)) }

func (s *cupertinoSliderState) emitHapticFeedback(isFastDrag bool) {
	if foundation.DefaultTargetPlatform() != foundation.TargetPlatformIOS {
		return
	}
	haptic := widgets.WidgetsBindingOf(s.Context()).HapticFeedback()
	if isFastDrag {
		haptic.MediumImpact()
	} else {
		haptic.SelectionClick()
	}
}

func (s *cupertinoSliderState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	active := CupertinoThemeOf(bc).PrimaryColor()
	if w.o.ActiveColor != nil {
		active = w.o.ActiveColor
	}
	thumb := ui.ColorValue(CupertinoColors.White)
	if w.o.ThumbColor != nil {
		thumb = w.o.ThumbColor
	}
	r := &cupertinoSliderRenderWidget{
		LeafRenderObjectWidget: widgets.NewLeafRenderObjectWidgetBase(),
		value:                  (w.value - w.o.Min) / (w.Max() - w.o.Min),
		divisions:              w.o.Divisions,
		activeColor:            CupertinoDynamicColorResolve(active, bc),
		thumbColor:             CupertinoDynamicColorResolve(thumb, bc),
		vsync:                  s,
		gestures:               widgets.WidgetsBindingOf(bc).Gestures(),
		gestureSettings:        widgets.MediaQueryGestureSettingsOf(bc),
	}
	if w.onChanged != nil {
		r.onChanged = s.handleChanged
	}
	if w.o.OnChangeStart != nil {
		r.onChangeStart = s.handleDragStart
	}
	if w.o.OnChangeEnd != nil {
		r.onChangeEnd = s.handleDragEnd
	}
	return r
}

// cupertinoSliderRenderWidget makes the render object of a slider.
//
// Flutter: cupertino/slider.dart _CupertinoSliderRenderObjectWidget
type cupertinoSliderRenderWidget struct {
	widgets.LeafRenderObjectWidget
	value                   float64
	divisions               foundation.Opt[int]
	activeColor, thumbColor ui.Color
	onChanged               func(value float64, isFastDrag bool)
	onChangeStart           func(value float64)
	onChangeEnd             func(value float64)
	vsync                   scheduler.TickerProvider
	gestures                *gestures.GestureBinding
	gestureSettings         gestures.DeviceGestureSettings
}

func (w *cupertinoSliderRenderWidget) trackColor(bc widgets.BuildContext) ui.Color {
	return CupertinoDynamicColorResolve(CupertinoColors.SystemFill, bc)
}

func (w *cupertinoSliderRenderWidget) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	cursor := services.MouseCursorDefer
	if foundation.IsWeb {
		cursor = services.SystemMouseCursors.Click
	}
	r := &renderCupertinoSlider{
		value: w.value, divisions: w.divisions, activeColor: w.activeColor, thumbColor: w.thumbColor, trackColor: w.trackColor(bc),
		onChanged: w.onChanged, onChangeStart: w.onChangeStart, onChangeEnd: w.onChangeEnd, textDirection: widgets.DirectionalityOf(bc),
		cursor: cursor, gestures: w.gestures, gestureSettings: w.gestureSettings,
	}
	rendering.InitRenderConstrainedBox(&r.RenderConstrainedBox, r, rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{
		Width: foundation.Some(sliderWidth), Height: foundation.Some(sliderHeight)}), nil)
	r.position = animation.NewAnimationController(w.vsync, animation.AnimationControllerOpts{
		Value: foundation.Some(w.value), Duration: foundation.Some(sliderDiscreteTransitionDuration)})
	r.positionListener = r.position.AddListener(r.MarkNeedsPaint)
	return r
}

func (w *cupertinoSliderRenderWidget) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderCupertinoSlider)
	r.setValue(w.value)
	r.setDivisions(w.divisions)
	r.setActiveColor(w.activeColor)
	r.setThumbColor(w.thumbColor)
	r.setTrackColor(w.trackColor(bc))
	r.setOnChanged(w.onChanged)
	r.onChangeStart = w.onChangeStart
	r.onChangeEnd = w.onChangeEnd
	r.setTextDirection(widgets.DirectionalityOf(bc))
}

func (w *cupertinoSliderRenderWidget) DidUnmountRenderObject(rendering.RenderObject) {}

// renderCupertinoSlider is the render object of a slider: a track and a thumb, with a drag that selects the value.
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider
type renderCupertinoSlider struct {
	rendering.RenderConstrainedBox
	value                   float64
	divisions               foundation.Opt[int]
	activeColor, thumbColor ui.Color
	trackColor              ui.Color
	onChanged               func(value float64, isFastDrag bool)
	onChangeStart           func(value float64)
	onChangeEnd             func(value float64)
	textDirection           ui.TextDirection
	cursor                  services.MouseCursor
	gestures                *gestures.GestureBinding
	gestureSettings         gestures.DeviceGestureSettings

	position         *animation.AnimationController
	positionListener foundation.ListenerHandle
	drag             *gestures.HorizontalDragGestureRecognizer

	currentDragValue    float64
	lastUpdateTimestamp foundation.Opt[time.Duration]
}

func (r *renderCupertinoSlider) isInteractive() bool { return r.onChanged != nil }

func (r *renderCupertinoSlider) setValue(newValue float64) {
	if newValue == r.value {
		return
	}
	r.value = newValue
	if r.divisions.IsSome() {
		r.position.AnimateTo(newValue, animation.AnimateOpts{Curve: animation.Curves.FastOutSlowIn})
	} else {
		r.position.SetValue(newValue)
	}
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderCupertinoSlider) setDivisions(v foundation.Opt[int]) {
	if v == r.divisions {
		return
	}
	r.divisions = v
	r.MarkNeedsPaint()
}

func (r *renderCupertinoSlider) setActiveColor(v ui.Color) {
	if v == r.activeColor {
		return
	}
	r.activeColor = v
	r.MarkNeedsPaint()
}

func (r *renderCupertinoSlider) setThumbColor(v ui.Color) {
	if v == r.thumbColor {
		return
	}
	r.thumbColor = v
	r.MarkNeedsPaint()
}

func (r *renderCupertinoSlider) setTrackColor(v ui.Color) {
	if v == r.trackColor {
		return
	}
	r.trackColor = v
	r.MarkNeedsPaint()
}

func (r *renderCupertinoSlider) setOnChanged(v func(float64, bool)) {
	wasInteractive := r.isInteractive()
	r.onChanged = v
	if wasInteractive != r.isInteractive() {
		r.MarkNeedsSemanticsUpdate()
	}
}

func (r *renderCupertinoSlider) setTextDirection(v ui.TextDirection) {
	if r.textDirection == v {
		return
	}
	r.textDirection = v
	r.MarkNeedsPaint()
}

func (r *renderCupertinoSlider) discretizedCurrentDragValue() float64 {
	dragValue := ui.ClampDouble(r.currentDragValue, 0.0, 1.0)
	if d, ok := r.divisions.Get(); ok {
		dragValue = math.Round(dragValue*float64(d)) / float64(d)
	}
	return dragValue
}

func (r *renderCupertinoSlider) trackLeft() float64  { return sliderPadding }
func (r *renderCupertinoSlider) trackRight() float64 { return r.Size().Width - sliderPadding }
func (r *renderCupertinoSlider) thumbCenter() float64 {
	visualPosition := r.value
	if r.textDirection == ui.TextDirectionRTL {
		visualPosition = 1.0 - r.value
	}
	return ui.LerpDouble(r.trackLeft()+CupertinoThumbPainterRadius, r.trackRight()-CupertinoThumbPainterRadius, visualPosition)
}

// Attach creates the drag recognizer.
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider (the recognizer is made in the constructor)
func (r *renderCupertinoSlider) Attach(owner *rendering.PipelineOwner) {
	r.RenderConstrainedBox.Attach(owner)
	r.drag = gestures.NewHorizontalDragGestureRecognizer(r.gestures, gestures.DragGestureRecognizerOpts{DebugOwner: r})
	r.drag.OnStart = r.handleDragStart
	r.drag.OnUpdate = r.handleDragUpdate
	r.drag.OnEnd = r.handleDragEnd
	r.drag.GestureSettings = foundation.Some(r.gestureSettings)
}

// Detach disposes the drag recognizer.
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider.dispose
func (r *renderCupertinoSlider) Detach() {
	if r.drag != nil {
		r.drag.Dispose()
		r.drag = nil
	}
	r.RenderConstrainedBox.Detach()
}

// Dispose releases the position animation.
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider.dispose
func (r *renderCupertinoSlider) Dispose() {
	r.position.RemoveListener(r.positionListener)
	r.position.Dispose()
	r.RenderConstrainedBox.Dispose()
}

func (r *renderCupertinoSlider) handleDragStart(details gestures.DragStartDetails) {
	r.startInteraction(details)
}

func (r *renderCupertinoSlider) handleDragUpdate(details gestures.DragUpdateDetails) {
	if !r.isInteractive() {
		return
	}
	extent := math.Max(sliderPadding, r.Size().Width-2.0*(sliderPadding+CupertinoThumbPainterRadius))
	valueDelta := details.PrimaryDelta.Value() / extent
	if r.textDirection == ui.TextDirectionRTL {
		r.currentDragValue -= valueDelta
	} else {
		r.currentDragValue += valueDelta
	}
	isFast := false
	current := details.SourceTimeStamp
	if c, ok := current.Get(); ok && r.lastUpdateTimestamp.IsSome() {
		timeDelta := (c - r.lastUpdateTimestamp.Value()).Milliseconds()
		velocity := math.Abs(valueDelta) * 1000.0 / float64(timeDelta)
		isFast = velocity > sliderVelocityThreshold
	}
	r.lastUpdateTimestamp = current
	r.onChanged(r.discretizedCurrentDragValue(), isFast)
}

func (r *renderCupertinoSlider) handleDragEnd(gestures.DragEndDetails) { r.endInteraction() }

func (r *renderCupertinoSlider) startInteraction(details gestures.DragStartDetails) {
	if r.isInteractive() {
		if r.onChangeStart != nil {
			r.onChangeStart(r.discretizedCurrentDragValue())
		}
		r.currentDragValue = r.value
		r.lastUpdateTimestamp = details.SourceTimeStamp
		r.onChanged(r.discretizedCurrentDragValue(), false)
	}
}

func (r *renderCupertinoSlider) endInteraction() {
	if r.onChangeEnd != nil {
		r.onChangeEnd(r.discretizedCurrentDragValue())
	}
	r.currentDragValue = 0.0
	r.lastUpdateTimestamp = foundation.None[time.Duration]()
}

// HitTestSelf is true when the position is on the thumb (with the padding around it).
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider.hitTestSelf
func (r *renderCupertinoSlider) HitTestSelf(position ui.Offset) bool {
	return math.Abs(position.Dx-r.thumbCenter()) < CupertinoThumbPainterRadius+sliderPadding
}

// HandleEvent hands a pointer-down to the drag recognizer.
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider.handleEvent
func (r *renderCupertinoSlider) HandleEvent(event gestures.PointerEvent, _ gestures.HitTestEntry) {
	if down, ok := event.(*gestures.PointerDownEvent); ok && r.isInteractive() && r.drag != nil {
		r.drag.AddPointer(down)
	}
}

// Paint paints the track and the thumb.
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider.paint
func (r *renderCupertinoSlider) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	var visualPosition float64
	var leftColor, rightColor ui.Color
	if r.textDirection == ui.TextDirectionRTL {
		visualPosition, leftColor, rightColor = 1.0-r.position.Value(), r.activeColor, r.trackColor
	} else {
		visualPosition, leftColor, rightColor = r.position.Value(), r.trackColor, r.activeColor
	}
	size := r.Size()
	trackCenter := offset.Dy + size.Height/2.0
	trackLeft := offset.Dx + r.trackLeft()
	trackTop := trackCenter - 1.0
	trackBottom := trackCenter + 1.0
	trackRight := offset.Dx + r.trackRight()
	trackActive := offset.Dx + r.thumbCenter()

	canvas := context.Canvas()
	if visualPosition > 0.0 {
		paint := ui.NewPaint()
		paint.SetColor(rightColor)
		canvas.DrawRRect(ui.NewRRectFromLTRBXY(trackLeft, trackTop, trackActive, trackBottom, 1.0, 1.0), paint)
	}
	if visualPosition < 1.0 {
		paint := ui.NewPaint()
		paint.SetColor(leftColor)
		canvas.DrawRRect(ui.NewRRectFromLTRBXY(trackActive, trackTop, trackRight, trackBottom, 1.0, 1.0), paint)
	}
	thumbCenter := ui.Offset{Dx: trackActive, Dy: trackCenter}
	NewCupertinoThumbPainter(CupertinoThumbPainterOpts{Color: foundation.Some(r.thumbColor)}).
		Paint(canvas, ui.NewRectFromCircle(thumbCenter, CupertinoThumbPainterRadius))
}

// DescribeSemanticsConfiguration makes the slider a semantic boundary with the value and the actions to increase
// and decrease it.
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider.describeSemanticsConfiguration
func (r *renderCupertinoSlider) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderConstrainedBox.DescribeSemanticsConfiguration(config)
	config.SetIsSemanticBoundary(r.isInteractive())
	config.SetIsSlider(true)
	if r.isInteractive() {
		config.SetTextDirection(r.textDirection)
		config.SetOnIncrease(r.increaseAction)
		config.SetOnDecrease(r.decreaseAction)
		pct := func(v float64) string { return strconv.Itoa(int(math.Round(v*100))) + "%" }
		unit := r.semanticActionUnit()
		config.SetValue(pct(r.value))
		config.SetIncreasedValue(pct(ui.ClampDouble(r.value+unit, 0.0, 1.0)))
		config.SetDecreasedValue(pct(ui.ClampDouble(r.value-unit, 0.0, 1.0)))
	}
}

func (r *renderCupertinoSlider) semanticActionUnit() float64 {
	if d, ok := r.divisions.Get(); ok {
		return 1.0 / float64(d)
	}
	return sliderAdjustmentUnit
}

func (r *renderCupertinoSlider) increaseAction() {
	if r.isInteractive() {
		r.onChanged(ui.ClampDouble(r.value+r.semanticActionUnit(), 0.0, 1.0), false)
	}
}

func (r *renderCupertinoSlider) decreaseAction() {
	if r.isInteractive() {
		r.onChanged(ui.ClampDouble(r.value-r.semanticActionUnit(), 0.0, 1.0), false)
	}
}

// Cursor is the cursor over the slider.
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider.cursor
func (r *renderCupertinoSlider) Cursor() services.MouseCursor { return r.cursor }

// OnEnter is nil: the slider does not listen to the mouse.
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider.onEnter
func (r *renderCupertinoSlider) OnEnter() services.PointerEnterEventListener { return nil }

// OnExit is nil: the slider does not listen to the mouse.
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider.onExit
func (r *renderCupertinoSlider) OnExit() services.PointerExitEventListener { return nil }

// ValidForMouseTracker is false, as in Flutter.
//
// Flutter: cupertino/slider.dart _RenderCupertinoSlider.validForMouseTracker
func (r *renderCupertinoSlider) ValidForMouseTracker() bool { return false }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoSliderValueOutOfRange  = "CupertinoSliderValueOutOfRange"
	tagCupertinoSliderDivisionsInvalid = "CupertinoSliderDivisionsInvalid"
)
