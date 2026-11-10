package cupertino

import (
	"fmt"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The measures of the sliding segmented control (the constants of cupertino/sliding_segmented_control.dart).
const (
	slidingMinHeight                 = 28.0
	slidingSeparatorWidth            = 1.0
	slidingMinThumbScale             = 0.95
	slidingMaxThumbScaleForMomentary = 1.05
	slidingSegmentMinPadding         = 10.0
	slidingTouchYDistanceThreshold   = 50.0 * 50.0
	slidingContentPressedMinOpacity  = 0.2
	slidingFontSize                  = 13.0
	slidingFontWeight                = ui.FontWeightW500
	slidingHighlightedFontWeight     = ui.FontWeightW600

	slidingSpringAnimationDuration    = 412 * millisecond
	slidingOpacityAnimationDuration   = 470 * millisecond
	slidingHighlightAnimationDuration = 200 * millisecond
)

var (
	slidingCornerRadius   = ui.NewRadiusCircular(9)
	slidingThumbRadius    = ui.NewRadiusCircular(7)
	slidingThumbInsets    = painting.NewEdgeInsetsSymmetric(0, 1)
	slidingSeparatorInset = painting.NewEdgeInsetsSymmetric(5, 0)
	slidingSeparatorColor = ui.NewColor(0x4D8E8E93)
	slidingDisabledColor  = ui.NewColor(0x737A7A7A)
	slidingThumbColor     = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF636366))
)

// slidingThumbSpring is the simulation of the springs of the thumb: it is always started from the beginning, so
// every animation that uses it replaces the previous one.
//
// Flutter: cupertino/sliding_segmented_control.dart _kThumbSpringAnimationSimulation
func slidingThumbSpring() *physics.SpringSimulation {
	return physics.NewSpringSimulation(physics.NewSpringDescription(1, 503.551, 44.8799), 0, 1, 0)
}

// slidingSegmentLocation says where a segment is in the control.
//
// Flutter: cupertino/sliding_segmented_control.dart _SegmentLocation
type slidingSegmentLocation int

const (
	slidingSegmentLeftmost slidingSegmentLocation = iota
	slidingSegmentRightmost
	slidingSegmentInbetween
)

// slidingSegment is the widget of a segment of the control: its child with the effects of a press and of the
// highlight.
//
// Flutter: cupertino/sliding_segmented_control.dart _Segment
type slidingSegment[T comparable] struct {
	widgets.StatefulWidget
	child                                      widgets.Widget
	pressed, highlighted, enabled, isMomentary bool
	isDragging                                 bool
	segmentLocation                            slidingSegmentLocation
}

func (w *slidingSegment[T]) shouldFadeoutContent() bool {
	return w.pressed && !w.highlighted && w.enabled && !w.isMomentary
}

func (w *slidingSegment[T]) shouldScaleContent() bool {
	return w.pressed && w.enabled && (w.highlighted && w.isDragging || w.isMomentary)
}

func (w *slidingSegment[T]) shouldHighlightContent() bool { return w.highlighted && !w.isMomentary }

func (w *slidingSegment[T]) CreateState() widgets.State { return &slidingSegmentState[T]{} }

type slidingSegmentState[T comparable] struct {
	widgets.StateOf[*slidingSegment[T]]
	widgets.TickerProvider
	highlightPressScaleController *animation.AnimationController
	highlightPressScaleAnimation  animation.Animation[float64]
}

func (s *slidingSegmentState[T]) InitState() {
	value := 0.0
	if s.Widget().shouldScaleContent() {
		value = 1
	}
	s.highlightPressScaleController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(slidingOpacityAnimationDuration), Value: foundation.Some(value)})
	s.highlightPressScaleAnimation = animation.Drive[float64](s.highlightPressScaleController, animation.NewTween(1.0, slidingMinThumbScale))
}

func (s *slidingSegmentState[T]) DidUpdateWidget(old *slidingSegment[T]) {
	w := s.Widget()
	if old.shouldScaleContent() == w.shouldScaleContent() {
		return
	}
	var scaleAnimation animation.Animatable[float64]
	switch {
	case w.isMomentary && w.shouldScaleContent():
		scaleAnimation = animation.NewTweenSequence([]animation.TweenSequenceItem[float64]{
			{Tween: animation.NewTween(s.highlightPressScaleAnimation.Value(), slidingMaxThumbScaleForMomentary), Weight: 50},
			{Tween: animation.NewTween(slidingMaxThumbScaleForMomentary, 1.0), Weight: 50},
		})
	case w.shouldScaleContent():
		scaleAnimation = animation.NewTween(s.highlightPressScaleAnimation.Value(), slidingMinThumbScale)
	default:
		scaleAnimation = animation.NewTween(s.highlightPressScaleAnimation.Value(), 1.0)
	}
	s.highlightPressScaleAnimation = animation.Drive(s.highlightPressScaleController, scaleAnimation)
	s.highlightPressScaleController.AnimateWith(slidingThumbSpring())
}

func (s *slidingSegmentState[T]) Dispose() { s.highlightPressScaleController.Dispose() }

func (s *slidingSegmentState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var scaleAlignment painting.AlignmentGeometry
	switch w.segmentLocation {
	case slidingSegmentLeftmost:
		scaleAlignment = painting.AlignmentCenterLeft
	case slidingSegmentRightmost:
		scaleAlignment = painting.AlignmentCenterRight
	default:
		scaleAlignment = painting.AlignmentCenter
	}
	opacity := 1.0
	if w.shouldFadeoutContent() {
		opacity = slidingContentPressedMinOpacity
	}
	weight := slidingFontWeight
	if w.shouldHighlightContent() {
		weight = slidingHighlightedFontWeight
	}
	style := painting.TextStyleOpts{FontWeight: weight, FontSize: foundation.Some(slidingFontSize)}
	if !w.enabled {
		style.Color = foundation.Some(slidingDisabledColor)
	}
	return widgets.NewMetaData(widgets.NewIndexedStack([]widgets.Widget{
		widgets.NewAnimatedOpacity(opacity, slidingOpacityAnimationDuration,
			widgets.NewAnimatedDefaultTextStyle(widgets.DefaultTextStyleOf(bc).Style().Merge(painting.NewTextStyle(style)), slidingHighlightAnimationDuration,
				widgets.NewScaleTransition(s.highlightPressScaleAnimation, w.child, widgets.ScaleTransitionOpts{Alignment: scaleAlignment}),
				widgets.AnimatedDefaultTextStyleOpts{Curve: animation.Curves.Ease}),
			widgets.AnimatedOpacityOpts{Curve: animation.Curves.Ease}),
		widgets.NewDefaultTextStyleMerge(w.child, widgets.DefaultTextStyleMergeOpts{
			Style: painting.NewTextStyle(painting.TextStyleOpts{FontWeight: slidingHighlightedFontWeight, FontSize: foundation.Some(slidingFontSize)})}),
	}, widgets.IndexedStackOpts{Alignment: painting.AlignmentCenter}), widgets.MetaDataOpts{Behavior: rendering.HitTestBehaviorOpaque})
}

// slidingSegmentSeparator is the line between two segments; it fades out while a segment next to it is highlighted.
//
// Flutter: cupertino/sliding_segmented_control.dart _SegmentSeparator
type slidingSegmentSeparator struct {
	widgets.StatefulWidget
	highlighted bool
}

func (w *slidingSegmentSeparator) CreateState() widgets.State { return &slidingSegmentSeparatorState{} }

type slidingSegmentSeparatorState struct {
	widgets.StateOf[*slidingSegmentSeparator]
	widgets.TickerProvider
	separatorOpacityController *animation.AnimationController
}

func (s *slidingSegmentSeparatorState) InitState() {
	value := 1.0
	if s.Widget().highlighted {
		value = 0
	}
	s.separatorOpacityController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(slidingSpringAnimationDuration), Value: foundation.Some(value)})
}

func (s *slidingSegmentSeparatorState) DidUpdateWidget(old *slidingSegmentSeparator) {
	if old.highlighted != s.Widget().highlighted {
		target := 1.0
		if s.Widget().highlighted {
			target = 0
		}
		s.separatorOpacityController.AnimateTo(target, animation.AnimateOpts{Duration: foundation.Some(slidingSpringAnimationDuration), Curve: animation.Curves.Ease})
	}
}

func (s *slidingSegmentSeparatorState) Dispose() { s.separatorOpacityController.Dispose() }

func (s *slidingSegmentSeparatorState) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewAnimatedBuilder(s.separatorOpacityController, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
		return widgets.NewPadding(slidingSeparatorInset, widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color:        foundation.Some(slidingSeparatorColor.WithOpacity(slidingSeparatorColor.Opacity() * s.separatorOpacityController.Value())),
			BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(slidingSeparatorWidth / 2)),
		}), child))
	}, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(slidingSeparatorWidth)}))
}

// CupertinoSlidingSegmentedControlOpts are the optional parameters of NewCupertinoSlidingSegmentedControl.
//
// Flutter: cupertino/sliding_segmented_control.dart CupertinoSlidingSegmentedControl.new
type CupertinoSlidingSegmentedControlOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// DisabledChildren are the values of the segments that do not respond to a tap.
	DisabledChildren []T
	// GroupValue is the value of the selected segment; none selects nothing.
	GroupValue foundation.Opt[T]
	// ThumbColor is the color of the thumb; the default is white in light mode and a gray in dark mode.
	ThumbColor ui.ColorValue
	// Padding is the padding around the segments; the default is 2 vertically and 3 horizontally.
	Padding painting.EdgeInsetsGeometry
	// BackgroundColor is the color of the control; the default is CupertinoColors.TertiarySystemFill.
	BackgroundColor ui.ColorValue
	// ProportionalWidth makes each segment as wide as its content needs, in proportion, instead of all the same.
	ProportionalWidth bool
	// IsMomentary makes the control a row of buttons: no segment stays selected.
	IsMomentary bool
}

// CupertinoSlidingSegmentedControl is an iOS 13 styled segmented control: a thumb that slides between the
// segments, and that the user can drag.
//
// Flutter: cupertino/sliding_segmented_control.dart CupertinoSlidingSegmentedControl
type CupertinoSlidingSegmentedControl[T comparable] struct {
	widgets.StatefulWidget
	children       []CupertinoSegment[T]
	onValueChanged func(value T)
	o              CupertinoSlidingSegmentedControlOpts[T]
}

// NewCupertinoSlidingSegmentedControl creates a segmented control from at least two segments. In debug builds it
// panics with the tag CupertinoSlidingSegmentedControlTooFewChildren, CupertinoSlidingSegmentedControlDuplicateValue
// or CupertinoSlidingSegmentedControlGroupValueNotAChild when the arguments are not valid.
//
// Flutter: cupertino/sliding_segmented_control.dart CupertinoSlidingSegmentedControl.new
func NewCupertinoSlidingSegmentedControl[T comparable](onValueChanged func(value T), children []CupertinoSegment[T], opts ...CupertinoSlidingSegmentedControlOpts[T]) *CupertinoSlidingSegmentedControl[T] {
	o := firstOf(opts)
	if foundation.DebugMode {
		seen := map[T]bool{}
		for _, c := range children {
			if seen[c.Value] {
				panic(errors.New(tagCupertinoSlidingSegmentedControlDuplicateValue, "two segments of a CupertinoSlidingSegmentedControl have the same value", "value", fmt.Sprint(c.Value)))
			}
			seen[c.Value] = true
		}
		switch {
		case len(children) < 2:
			panic(errors.New(tagCupertinoSlidingSegmentedControlTooFewChildren, "a CupertinoSlidingSegmentedControl needs at least two segments", "count", len(children)))
		case o.GroupValue.IsSome() && !seen[o.GroupValue.Value()]:
			panic(errors.New(tagCupertinoSlidingSegmentedControlGroupValueNotAChild, "The groupValue must be either none or one of the values of the segments.", "groupValue", fmt.Sprint(o.GroupValue.Value())))
		}
	}
	return &CupertinoSlidingSegmentedControl[T]{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), children, onValueChanged, o}
}

// Children are the segments, in the order they are shown.
//
// Flutter: cupertino/sliding_segmented_control.dart CupertinoSlidingSegmentedControl.children
func (w *CupertinoSlidingSegmentedControl[T]) Children() []CupertinoSegment[T] { return w.children }

// GroupValue is the value of the selected segment.
//
// Flutter: cupertino/sliding_segmented_control.dart CupertinoSlidingSegmentedControl.groupValue
func (w *CupertinoSlidingSegmentedControl[T]) GroupValue() foundation.Opt[T] { return w.o.GroupValue }

// OnValueChanged is called when the user selects a segment.
//
// Flutter: cupertino/sliding_segmented_control.dart CupertinoSlidingSegmentedControl.onValueChanged
func (w *CupertinoSlidingSegmentedControl[T]) OnValueChanged() func(value T) { return w.onValueChanged }

// IsMomentary is whether the control is a row of buttons.
//
// Flutter: cupertino/sliding_segmented_control.dart CupertinoSlidingSegmentedControl.isMomentary
func (w *CupertinoSlidingSegmentedControl[T]) IsMomentary() bool { return w.o.IsMomentary }

// ProportionalWidth is whether the segments have the width of their content.
//
// Flutter: cupertino/sliding_segmented_control.dart CupertinoSlidingSegmentedControl.proportionalWidth
func (w *CupertinoSlidingSegmentedControl[T]) ProportionalWidth() bool { return w.o.ProportionalWidth }

// CreateState creates the state.
//
// Flutter: cupertino/sliding_segmented_control.dart CupertinoSlidingSegmentedControl.createState
func (w *CupertinoSlidingSegmentedControl[T]) CreateState() widgets.State {
	return &slidingSegmentedControlState[T]{segmentKeys: map[T]*widgets.GlobalKey[*slidingSegmentButtonState[T]]{}}
}

func (w *CupertinoSlidingSegmentedControl[T]) disabled(value T) bool {
	for _, d := range w.o.DisabledChildren {
		if d == value {
			return true
		}
	}
	return false
}

// slidingSegmentButton is a segment as a radio client with a focus node.
//
// Flutter: cupertino/sliding_segmented_control.dart _SlidingSegmentButton
type slidingSegmentButton[T comparable] struct {
	widgets.StatefulWidget
	value   T
	child   widgets.Widget
	enabled bool
}

func (w *slidingSegmentButton[T]) CreateState() widgets.State { return &slidingSegmentButtonState[T]{} }

type slidingSegmentButtonState[T comparable] struct {
	widgets.StateOf[*slidingSegmentButton[T]]
	widgets.RadioClientRegistration[T]
	focusNode *widgets.FocusNode
}

func (s *slidingSegmentButtonState[T]) InitState() {
	s.focusNode = widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: fmt.Sprintf("CupertinoSlidingSegmentedControl<%T>[%v]", s.Widget().value, s.Widget().value)})
}

func (s *slidingSegmentButtonState[T]) updateRegistry() {
	if !s.Widget().enabled {
		s.SetRegistry(s, nil)
		return
	}
	r, _ := widgets.RadioGroupMaybeOf[T](s.Context())
	s.SetRegistry(s, r)
}

func (s *slidingSegmentButtonState[T]) DidChangeDependencies() { s.updateRegistry() }

func (s *slidingSegmentButtonState[T]) DidUpdateWidget(old *slidingSegmentButton[T]) {
	if old.enabled != s.Widget().enabled {
		s.updateRegistry()
	}
}

func (s *slidingSegmentButtonState[T]) Dispose() {
	s.SetRegistry(s, nil)
	s.focusNode.Dispose()
}

func (s *slidingSegmentButtonState[T]) RadioValue() T                 { return s.Widget().value }
func (s *slidingSegmentButtonState[T]) FocusNode() *widgets.FocusNode { return s.focusNode }
func (s *slidingSegmentButtonState[T]) Tristate() bool                { return false }
func (s *slidingSegmentButtonState[T]) Enabled() bool                 { return s.Widget().enabled }

func (s *slidingSegmentButtonState[T]) requestFocus() {
	if s.Widget().enabled {
		s.focusNode.RequestFocus()
	}
}

func (s *slidingSegmentButtonState[T]) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewFocus(s.Widget().child, widgets.FocusOpts{
		FocusNode:       s.focusNode,
		CanRequestFocus: foundation.Some(s.Widget().enabled),
		OnKeyEvent: func(*widgets.FocusNode, services.KeyEvent) widgets.KeyEventResult {
			return widgets.KeyEventResultIgnored
		},
	})
}

type slidingSegmentedControlState[T comparable] struct {
	widgets.StateOf[*CupertinoSlidingSegmentedControl[T]]
	widgets.TickerProvider
	thumbController      *animation.AnimationController
	thumbAnimatable      animation.Animatable[foundation.Opt[ui.Rect]]
	thumbScaleController *animation.AnimationController
	thumbScaleAnimation  animation.Animation[float64]

	tap       *gestures.TapGestureRecognizer
	drag      *gestures.HorizontalDragGestureRecognizer
	longPress *gestures.LongPressGestureRecognizer

	renderWidgetKey *widgets.GlobalKey[widgets.State]
	segmentKeys     map[T]*widgets.GlobalKey[*slidingSegmentButtonState[T]]

	startedOnSelectedSegment foundation.Opt[bool]
	startedOnDisabledSegment bool
	highlighted, pressed     foundation.Opt[T]
}

func (s *slidingSegmentedControlState[T]) InitState() {
	s.thumbController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(slidingSpringAnimationDuration), Value: foundation.Some(0.0)})
	s.thumbScaleController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(slidingSpringAnimationDuration), Value: foundation.Some(0.0)})
	s.thumbScaleAnimation = animation.Drive[float64](s.thumbScaleController, animation.NewTween(1.0, slidingMinThumbScale))
	s.renderWidgetKey = widgets.NewGlobalKey[widgets.State]()

	binding := widgets.WidgetsBindingOf(s.Context()).Gestures()
	s.tap = gestures.NewTapGestureRecognizer(binding, gestures.TapGestureRecognizerOpts{DebugOwner: s})
	s.drag = gestures.NewHorizontalDragGestureRecognizer(binding, gestures.DragGestureRecognizerOpts{DebugOwner: s})
	s.longPress = gestures.NewLongPressGestureRecognizer(binding, gestures.LongPressGestureRecognizerOpts{DebugOwner: s})
	team := gestures.NewGestureArenaTeam(binding.GestureArena)
	s.longPress.SetTeam(team)
	s.drag.SetTeam(team)
	team.Captain = s.drag
	s.drag.OnDown = s.onDown
	s.drag.OnUpdate = s.onUpdate
	s.drag.OnEnd = s.onEnd
	s.drag.OnCancel = s.onCancel
	s.tap.OnTapUp = s.onTapUp
	s.longPress.OnLongPress = func() {}
	s.highlighted = s.Widget().o.GroupValue
}

func (s *slidingSegmentedControlState[T]) DidUpdateWidget(*CupertinoSlidingSegmentedControl[T]) {
	w := s.Widget()
	if !s.isThumbDragging() && s.highlighted != w.o.GroupValue {
		s.thumbController.AnimateWith(slidingThumbSpring())
		s.thumbAnimatable = nil
		s.highlighted = w.o.GroupValue
	}
}

func (s *slidingSegmentedControlState[T]) Dispose() {
	s.thumbScaleController.Dispose()
	s.thumbController.Dispose()
	s.drag.Dispose()
	s.tap.Dispose()
	s.longPress.Dispose()
}

func (s *slidingSegmentedControlState[T]) isThumbDragging() bool {
	return s.startedOnSelectedSegment.OrElse(false) && !s.startedOnDisabledSegment
}

func (s *slidingSegmentedControlState[T]) segmentForXPosition(dx float64) T {
	w := s.Widget()
	element := s.renderWidgetKey.CurrentElement()
	renderBox := element.(widgets.BuildContext).FindRenderObject().(*renderSlidingSegmentedControl[T])
	numOfChildren := len(w.children)
	segmentIndex := renderBox.getClosestSegmentIndex(dx)
	if widgets.DirectionalityOf(s.Context()) == ui.TextDirectionRTL {
		segmentIndex = numOfChildren - 1 - segmentIndex
	}
	return w.children[segmentIndex].Value
}

func (s *slidingSegmentedControlState[T]) hasDraggedTooFar(details gestures.DragUpdateDetails) bool {
	size := s.Context().FindRenderObject().(rendering.Box).Size()
	offCenter := details.LocalPosition.Sub(ui.Offset{Dx: size.Width / 2, Dy: size.Height / 2})
	l2 := math.Pow(math.Max(0.0, math.Abs(offCenter.Dx)-size.Width/2), 2) + math.Pow(math.Max(0.0, math.Abs(offCenter.Dy)-size.Height/2), 2)
	return l2 > slidingTouchYDistanceThreshold
}

func (s *slidingSegmentedControlState[T]) playThumbScaleAnimation(isExpanding bool) {
	end := slidingMinThumbScale
	if isExpanding {
		end = 1
	}
	s.thumbScaleAnimation = animation.Drive[float64](s.thumbScaleController, animation.NewTween(s.thumbScaleAnimation.Value(), end))
	s.thumbScaleController.AnimateWith(slidingThumbSpring())
}

func (s *slidingSegmentedControlState[T]) onHighlightChangedByGesture(newValue T) {
	if s.highlighted == foundation.Some(newValue) {
		return
	}
	s.SetState(func() { s.highlighted = foundation.Some(newValue) })
	s.thumbController.AnimateWith(slidingThumbSpring())
	s.thumbAnimatable = nil
}

func (s *slidingSegmentedControlState[T]) onPressedChangedByGesture(newValue foundation.Opt[T]) {
	if s.pressed != newValue {
		s.SetState(func() { s.pressed = newValue })
	}
}

func (s *slidingSegmentedControlState[T]) requestSegmentFocus(value T) {
	if k := s.segmentKeys[value]; k != nil {
		if st, ok := k.CurrentState(); ok {
			st.requestFocus()
		}
	}
}

func (s *slidingSegmentedControlState[T]) onTapUp(details gestures.TapUpDetails) {
	if s.isThumbDragging() {
		return
	}
	w := s.Widget()
	segment := s.segmentForXPosition(details.LocalPosition.Dx)
	s.onPressedChangedByGesture(foundation.None[T]())
	if !w.disabled(segment) {
		s.requestSegmentFocus(segment)
		if w.o.GroupValue != foundation.Some(segment) {
			w.onValueChanged(segment)
		}
	}
}

func (s *slidingSegmentedControlState[T]) onDown(details gestures.DragDownDetails) {
	w := s.Widget()
	touchDownSegment := s.segmentForXPosition(details.LocalPosition.Dx)
	s.startedOnSelectedSegment = foundation.Some(s.highlighted == foundation.Some(touchDownSegment))
	s.startedOnDisabledSegment = w.disabled(touchDownSegment)
	if w.disabled(touchDownSegment) {
		return
	}
	s.onPressedChangedByGesture(foundation.Some(touchDownSegment))
	if s.isThumbDragging() {
		s.playThumbScaleAnimation(false)
	}
}

func (s *slidingSegmentedControlState[T]) onUpdate(details gestures.DragUpdateDetails) {
	if s.startedOnDisabledSegment {
		return
	}
	w := s.Widget()
	touchDownSegment := s.segmentForXPosition(details.LocalPosition.Dx)
	if w.disabled(touchDownSegment) {
		return
	}
	if s.isThumbDragging() {
		s.onPressedChangedByGesture(foundation.Some(touchDownSegment))
		s.onHighlightChangedByGesture(touchDownSegment)
		return
	}
	segment := foundation.None[T]()
	if !s.hasDraggedTooFar(details) {
		segment = foundation.Some(s.segmentForXPosition(details.LocalPosition.Dx))
	}
	s.onPressedChangedByGesture(segment)
}

func (s *slidingSegmentedControlState[T]) onEnd(gestures.DragEndDetails) {
	w := s.Widget()
	pressed := s.pressed
	if s.isThumbDragging() {
		s.playThumbScaleAnimation(true)
		if s.highlighted != w.o.GroupValue {
			s.requestSegmentFocus(s.highlighted.Value())
			w.onValueChanged(s.highlighted.Value())
		}
	} else if p, ok := pressed.Get(); ok {
		s.onHighlightChangedByGesture(p)
		if s.highlighted != w.o.GroupValue {
			s.requestSegmentFocus(s.highlighted.Value())
			w.onValueChanged(s.highlighted.Value())
		}
	}
	s.onPressedChangedByGesture(foundation.None[T]())
	s.startedOnSelectedSegment = foundation.None[bool]()
}

func (s *slidingSegmentedControlState[T]) onCancel() {
	if s.isThumbDragging() {
		s.playThumbScaleAnimation(true)
	}
	s.onPressedChangedByGesture(foundation.None[T]())
	s.startedOnSelectedSegment = foundation.None[bool]()
}

func (s *slidingSegmentedControlState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	textDirection := widgets.DirectionalityOf(bc)
	var children []widgets.Widget
	isPreviousSegmentHighlighted := false
	highlightedIndex := foundation.None[int]()
	for index, entry := range w.children {
		isHighlighted := s.highlighted == foundation.Some(entry.Value)
		if isHighlighted {
			highlightedIndex = foundation.Some(index)
		}
		if index != 0 {
			children = append(children, &slidingSegmentSeparator{
				widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: foundation.NewValueKey(index)}), isPreviousSegmentHighlighted || isHighlighted})
		}
		last := len(w.children) - 1
		location := slidingSegmentInbetween
		switch {
		case textDirection == ui.TextDirectionLTR && index == 0, textDirection == ui.TextDirectionRTL && index == last:
			location = slidingSegmentLeftmost
		case textDirection == ui.TextDirectionLTR && index == last, textDirection == ui.TextDirectionRTL && index == 0:
			location = slidingSegmentRightmost
		}
		segmentKey := s.segmentKeys[entry.Value]
		if segmentKey == nil {
			segmentKey = widgets.NewGlobalKey[*slidingSegmentButtonState[T]]()
			s.segmentKeys[entry.Value] = segmentKey
		}
		value := entry.Value
		enabled := !w.disabled(value)
		var cursor services.MouseCursor = services.MouseCursorDefer
		if foundation.IsWeb {
			cursor = services.SystemMouseCursors.Click
		}
		children = append(children, &slidingSegmentButton[T]{
			widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: segmentKey}), value,
			widgets.NewSemantics(widgets.NewMouseRegion(&slidingSegment[T]{
				StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: foundation.NewValueKey(value)}),
				child:          entry.Child, pressed: s.pressed == foundation.Some(value), highlighted: isHighlighted, isDragging: s.isThumbDragging(),
				enabled: enabled, segmentLocation: location, isMomentary: w.o.IsMomentary,
			}, widgets.MouseRegionOpts{Cursor: cursor}), widgets.SemanticsOpts{
				Button: foundation.Some(true), InMutuallyExclusiveGroup: foundation.Some(true),
				Selected: foundation.Some(w.o.GroupValue == foundation.Some(value)),
				OnTap: func() {
					if w.disabled(value) {
						return
					}
					s.requestSegmentFocus(value)
					w.onValueChanged(value)
				},
			}),
			enabled,
		})
		isPreviousSegmentHighlighted = isHighlighted
	}
	if textDirection == ui.TextDirectionRTL {
		for i, j := 0, len(children)-1; i < j; i, j = i+1, j-1 {
			children[i], children[j] = children[j], children[i]
		}
		if i, ok := highlightedIndex.Get(); ok {
			highlightedIndex = foundation.Some(len(w.children) - 1 - i)
		}
	}
	var padding painting.EdgeInsetsGeometry = painting.NewEdgeInsetsSymmetric(2, 3)
	if w.o.Padding != nil {
		padding = w.o.Padding
	}
	var background ui.ColorValue = CupertinoColors.TertiarySystemFill
	if w.o.BackgroundColor != nil {
		background = w.o.BackgroundColor
	}
	var thumb ui.ColorValue = slidingThumbColor
	if w.o.ThumbColor != nil {
		thumb = w.o.ThumbColor
	}
	return widgets.NewActions(widgets.ActionMap{widgets.IntentTypeOf[widgets.VoidCallbackIntent](): widgets.NewVoidCallbackAction()},
		widgets.NewRadioGroup(func(value foundation.Opt[T]) {
			if v, ok := value.Get(); ok && !w.disabled(v) {
				w.onValueChanged(v)
			}
		}, widgets.NewUnconstrainedBox(widgets.NewContainer(
			widgets.NewAnimatedBuilder(s.thumbScaleAnimation, func(widgets.BuildContext, widgets.Widget) widgets.Widget {
				hi := highlightedIndex
				if w.o.IsMomentary {
					hi = foundation.None[int]()
				}
				return &slidingSegmentedControlRenderWidget[T]{
					MultiChildRenderObjectWidget: widgets.NewMultiChildRenderObjectWidgetBase(children, widgets.WidgetOpts{Key: s.renderWidgetKey}),
					highlightedIndex:             hi, thumbColor: CupertinoDynamicColorResolve(thumb, bc), thumbScale: s.thumbScaleAnimation.Value(),
					proportionalWidth: w.o.ProportionalWidth, state: s,
				}
			}, nil),
			widgets.ContainerOpts{
				ClipBehavior: ui.ClipAntiAlias,
				Padding:      padding.Resolve(textDirection),
				Decoration: painting.NewShapeDecoration(painting.NewRoundedSuperellipseBorder(painting.RoundedSuperellipseBorderOpts{
					BorderRadius: painting.NewBorderRadiusAll(slidingCornerRadius)}),
					painting.ShapeDecorationOpts{Color: foundation.Some(CupertinoDynamicColorResolve(background, bc))}),
			}), widgets.UnconstrainedBoxOpts{ConstrainedAxis: painting.AxisHorizontal}),
			widgets.RadioGroupOpts[T]{GroupValue: w.o.GroupValue}))
}

// slidingSegmentedControlRenderWidget makes the render object that lays the segments out and paints the thumb.
//
// Flutter: cupertino/sliding_segmented_control.dart _SegmentedControlRenderWidget
type slidingSegmentedControlRenderWidget[T comparable] struct {
	widgets.MultiChildRenderObjectWidget
	highlightedIndex  foundation.Opt[int]
	thumbColor        ui.Color
	thumbScale        float64
	proportionalWidth bool
	state             *slidingSegmentedControlState[T]
}

func (w *slidingSegmentedControlRenderWidget[T]) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderSlidingSegmentedControl[T]{highlightedIndex: w.highlightedIndex, thumbColor: w.thumbColor, thumbScale: w.thumbScale,
		proportionalWidth: w.proportionalWidth, state: w.state}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitContainerRenderObject(&r.ContainerRenderObject, r)
	return r
}

func (w *slidingSegmentedControlRenderWidget[T]) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderSlidingSegmentedControl[T])
	r.setThumbColor(w.thumbColor)
	r.setThumbScale(w.thumbScale)
	r.setHighlightedIndex(w.highlightedIndex)
	r.setProportionalWidth(w.proportionalWidth)
}

// slidingParentData is the parent data of a segment or of a separator.
//
// Flutter: cupertino/sliding_segmented_control.dart _SegmentedControlContainerBoxParentData
type slidingParentData struct {
	rendering.ContainerBoxParentData[rendering.Box]
}

func slidingParent(child rendering.Box) *slidingParentData {
	return child.ParentData().(*slidingParentData)
}

// renderSlidingSegmentedControl lays the segments and the separators out in a line, and paints the thumb under
// the highlighted segment. The children are the segments and, between them, the separators.
//
// Flutter: cupertino/sliding_segmented_control.dart _RenderSegmentedControl
type renderSlidingSegmentedControl[T comparable] struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
	state             *slidingSegmentedControlState[T]
	currentThumbRect  foundation.Opt[ui.Rect]
	highlightedIndex  foundation.Opt[int]
	thumbColor        ui.Color
	thumbScale        float64
	proportionalWidth bool
	thumbListener     foundation.ListenerHandle
}

func (r *renderSlidingSegmentedControl[T]) Attach(owner *rendering.PipelineOwner) {
	r.RenderBox.Attach(owner)
	r.thumbListener = r.state.thumbController.AddListener(r.MarkNeedsPaint)
}

func (r *renderSlidingSegmentedControl[T]) Detach() {
	r.state.thumbController.RemoveListener(r.thumbListener)
	r.RenderBox.Detach()
}

func (r *renderSlidingSegmentedControl[T]) setThumbScale(v float64) {
	if r.thumbScale == v {
		return
	}
	r.thumbScale = v
	if r.state.highlighted.IsSome() {
		r.MarkNeedsPaint()
	}
}

func (r *renderSlidingSegmentedControl[T]) setHighlightedIndex(v foundation.Opt[int]) {
	if r.highlightedIndex == v {
		return
	}
	r.highlightedIndex = v
	r.MarkNeedsPaint()
}

func (r *renderSlidingSegmentedControl[T]) setThumbColor(v ui.Color) {
	if r.thumbColor == v {
		return
	}
	r.thumbColor = v
	r.MarkNeedsPaint()
}

func (r *renderSlidingSegmentedControl[T]) setProportionalWidth(v bool) {
	if r.proportionalWidth == v {
		return
	}
	r.proportionalWidth = v
	r.MarkNeedsLayout()
}

func (r *renderSlidingSegmentedControl[T]) HandleEvent(event gestures.PointerEvent, _ gestures.HitTestEntry) {
	if down, ok := event.(*gestures.PointerDownEvent); ok && !r.state.isThumbDragging() {
		r.state.tap.AddPointer(down)
		r.state.longPress.AddPointer(down)
		r.state.drag.AddPointer(down)
	}
}

func (r *renderSlidingSegmentedControl[T]) separatorWidth() float64 {
	return slidingSeparatorInset.Horizontal() + slidingSeparatorWidth
}

func (r *renderSlidingSegmentedControl[T]) totalSeparatorWidth() float64 {
	return r.separatorWidth() * float64(r.ChildCount()/2)
}

func (r *renderSlidingSegmentedControl[T]) getClosestSegmentIndex(dx float64) int {
	index := 0
	for child := r.FirstChild(); child != nil; child = r.nonSeparatorChildAfter(child) {
		pd := slidingParent(child)
		clampX := ui.ClampDouble(dx, pd.Offset.Dx, child.Size().Width+pd.Offset.Dx)
		if dx <= clampX {
			break
		}
		index++
	}
	segmentCount := r.ChildCount()/2 + 1
	return min(index, segmentCount-1)
}

func (r *renderSlidingSegmentedControl[T]) nonSeparatorChildAfter(child rendering.Box) rendering.Box {
	next := r.ChildAfter(child)
	if next == nil {
		return nil
	}
	return r.ChildAfter(next)
}

func (r *renderSlidingSegmentedControl[T]) eachSegment(f func(child rendering.Box)) {
	for child := r.FirstChild(); child != nil; child = r.nonSeparatorChildAfter(child) {
		f(child)
	}
}

func (r *renderSlidingSegmentedControl[T]) ComputeMinIntrinsicWidth(height float64) float64 {
	count := float64(r.ChildCount()/2 + 1)
	maxMin := 0.0
	r.eachSegment(func(c rendering.Box) { maxMin = math.Max(maxMin, c.GetMinIntrinsicWidth(height)) })
	return (maxMin+2*slidingSegmentMinPadding)*count + r.totalSeparatorWidth()
}

func (r *renderSlidingSegmentedControl[T]) ComputeMaxIntrinsicWidth(height float64) float64 {
	count := float64(r.ChildCount()/2 + 1)
	maxMax := 0.0
	r.eachSegment(func(c rendering.Box) { maxMax = math.Max(maxMax, c.GetMaxIntrinsicWidth(height)) })
	return (maxMax+2*slidingSegmentMinPadding)*count + r.totalSeparatorWidth()
}

func (r *renderSlidingSegmentedControl[T]) ComputeMinIntrinsicHeight(width float64) float64 {
	h := slidingMinHeight
	r.eachSegment(func(c rendering.Box) { h = math.Max(h, c.GetMinIntrinsicHeight(width)) })
	return h
}

func (r *renderSlidingSegmentedControl[T]) ComputeMaxIntrinsicHeight(width float64) float64 {
	h := slidingMinHeight
	r.eachSegment(func(c rendering.Box) { h = math.Max(h, c.GetMaxIntrinsicHeight(width)) })
	return h
}

func (r *renderSlidingSegmentedControl[T]) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	return rendering.DefaultComputeDistanceToHighestActualBaseline(&r.ContainerRenderObject, baseline)
}

func (r *renderSlidingSegmentedControl[T]) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*slidingParentData); !ok {
		child.AsRenderObject().SetParentData(&slidingParentData{})
	}
}

func (r *renderSlidingSegmentedControl[T]) getMaxChildWidth(c rendering.BoxConstraints) float64 {
	count := float64(r.ChildCount()/2 + 1)
	childWidth := (c.MinWidth - r.totalSeparatorWidth()) / count
	r.eachSegment(func(ch rendering.Box) {
		childWidth = math.Max(childWidth, ch.GetMaxIntrinsicWidth(math.Inf(1))+2*slidingSegmentMinPadding)
	})
	return math.Min(childWidth, (c.MaxWidth-r.totalSeparatorWidth())/count)
}

func (r *renderSlidingSegmentedControl[T]) getMaxChildHeight(childWidth float64) float64 {
	maxHeight := slidingMinHeight
	r.eachSegment(func(c rendering.Box) { maxHeight = math.Max(maxHeight, c.GetMaxIntrinsicHeight(childWidth)) })
	return maxHeight
}

func (r *renderSlidingSegmentedControl[T]) getChildWidths(c rendering.BoxConstraints) []float64 {
	segmentCount := r.ChildCount()/2 + 1
	if !r.proportionalWidth {
		maxChildWidth := r.getMaxChildWidth(c)
		widths := make([]float64, segmentCount)
		for i := range widths {
			widths[i] = maxChildWidth
		}
		return widths
	}
	var widths []float64
	total := 0.0
	r.eachSegment(func(ch rendering.Box) {
		w := ch.GetMaxIntrinsicWidth(math.Inf(1)) + 2*slidingSegmentMinPadding
		widths = append(widths, w)
		total += w
	})
	allowedMax := c.MaxWidth - r.totalSeparatorWidth()
	allowedMin := c.MinWidth - r.totalSeparatorWidth()
	scale := ui.ClampDouble(total, allowedMin, allowedMax) / total
	if scale != 1 {
		for i := range widths {
			widths[i] *= scale
		}
	}
	return widths
}

func sumFloats(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s
}

func (r *renderSlidingSegmentedControl[T]) computeOverallSize(c rendering.BoxConstraints) ui.Size {
	maxChildHeight := r.getMaxChildHeight(c.MaxWidth)
	return c.Constrain(ui.Size{Width: sumFloats(r.getChildWidths(c)) + r.totalSeparatorWidth(), Height: maxChildHeight})
}

func (r *renderSlidingSegmentedControl[T]) ComputeDryBaseline(c rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	segmentWidths := r.getChildWidths(c)
	childHeight := r.getMaxChildHeight(c.MaxWidth)
	result := foundation.None[float64]()
	index := 0
	r.eachSegment(func(ch rendering.Box) {
		cc := rendering.NewBoxConstraintsTight(ui.Size{Width: segmentWidths[index], Height: childHeight})
		if v, ok := ch.GetDryBaseline(cc, baseline).Get(); ok {
			if cur, has := result.Get(); !has || v < cur {
				result = foundation.Some(v)
			}
		}
		index++
	})
	return result
}

func (r *renderSlidingSegmentedControl[T]) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return r.computeOverallSize(c)
}

func (r *renderSlidingSegmentedControl[T]) PerformLayout() {
	constraints := r.BoxConstraints()
	segmentWidths := r.getChildWidths(constraints)
	childHeight := r.getMaxChildHeight(math.Inf(1))
	separatorConstraints := rendering.BoxConstraints{MinWidth: 0, MaxWidth: math.Inf(1), MinHeight: childHeight, MaxHeight: childHeight}
	start := 0.0
	index := 0
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		cc := rendering.NewBoxConstraintsTight(ui.Size{Width: segmentWidths[index/2], Height: childHeight})
		if index%2 == 1 {
			cc = separatorConstraints
		}
		child.Layout(cc, rendering.LayoutOpts{ParentUsesSize: true})
		slidingParent(child).Offset = ui.Offset{Dx: start, Dy: 0}
		start += child.Size().Width
		index++
	}
	r.SetSize(r.computeOverallSize(constraints))
}

func (r *renderSlidingSegmentedControl[T]) moveThumbRectInBound(thumbRect foundation.Opt[ui.Rect], children []rendering.Box) foundation.Opt[ui.Rect] {
	t, ok := thumbRect.Get()
	if !ok {
		return thumbRect
	}
	first := children[0]
	last := children[len(children)-1]
	firstOffset := slidingParent(first).Offset
	leftMost := firstOffset.Dx
	rightMost := slidingParent(last).Offset.Dx + last.Size().Width
	return foundation.Some(ui.NewRectFromLTRB(
		math.Max(t.Left, leftMost-slidingThumbInsets.Left),
		firstOffset.Dy-slidingThumbInsets.Top,
		math.Min(t.Right, rightMost+slidingThumbInsets.Right),
		firstOffset.Dy+first.Size().Height+slidingThumbInsets.Bottom))
}

func (r *renderSlidingSegmentedControl[T]) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	children := rendering.GetChildrenAsList(&r.ContainerRenderObject)
	for index := 1; index < r.ChildCount(); index += 2 {
		r.paintChild(context, offset, children[index])
	}
	if hi, ok := r.highlightedIndex.Get(); ok {
		selected := children[hi*2]
		pd := slidingParent(selected)
		newThumbRect := slidingThumbInsets.InflateRect(rectFromOffsetAndSize(pd.Offset, selected.Size()))
		if r.state.thumbController.IsAnimating() {
			tween := r.state.thumbAnimatable
			if tween == nil {
				starting := r.moveThumbRectInBound(r.currentThumbRect, children).OrElse(newThumbRect)
				r.state.thumbAnimatable = animation.NewRectTween(foundation.Some(starting), foundation.Some(newThumbRect))
			} else if tween.Transform(1) != foundation.Some(newThumbRect) {
				starting := r.moveThumbRectInBound(r.currentThumbRect, children).OrElse(newThumbRect)
				r.state.thumbAnimatable = animation.NewRectTween(foundation.Some(starting), foundation.Some(newThumbRect)).
					Chain(animation.NewCurveTween(animation.NewInterval(r.state.thumbController.Value(), 1)))
			}
		} else {
			r.state.thumbAnimatable = nil
		}
		unscaled := newThumbRect
		if a := r.state.thumbAnimatable; a != nil {
			unscaled = a.Evaluate(r.state.thumbController).OrElse(newThumbRect)
		}
		r.currentThumbRect = foundation.Some(unscaled)
		var location slidingSegmentLocation
		switch {
		case hi == 0:
			location = slidingSegmentLeftmost
		case hi == len(children)/2:
			location = slidingSegmentRightmost
		default:
			location = slidingSegmentInbetween
		}
		delta := 0.0
		switch location {
		case slidingSegmentLeftmost:
			delta = unscaled.Width() - unscaled.Width()*r.thumbScale
		case slidingSegmentRightmost:
			delta = unscaled.Width()*r.thumbScale - unscaled.Width()
		}
		thumbRect := ui.NewRectFromCenter(unscaled.Center().Sub(ui.Offset{Dx: delta / 2}), unscaled.Width()*r.thumbScale, unscaled.Height()*r.thumbScale)
		r.paintThumb(context, offset, thumbRect)
	} else {
		r.currentThumbRect = foundation.None[ui.Rect]()
	}
	for index := 0; index < len(children); index += 2 {
		r.paintChild(context, offset, children[index])
	}
}

func (r *renderSlidingSegmentedControl[T]) paintChild(context *rendering.PaintingContext, offset ui.Offset, child rendering.Box) {
	context.PaintChild(child, slidingParent(child).Offset.Add(offset))
}

func (r *renderSlidingSegmentedControl[T]) paintThumb(context *rendering.PaintingContext, offset ui.Offset, thumbRect ui.Rect) {
	thumbShadow := []painting.BoxShadow{boxShadow(0x1F000000, 3, 8), boxShadow(0x0A000000, 3, 1)}
	canvas := context.Canvas()
	thumbShape := ui.NewRSuperellipseFromRectAndRadius(thumbRect.Shift(offset), slidingThumbRadius)
	for _, shadow := range thumbShadow {
		canvas.DrawRSuperellipse(thumbShape.Shift(shadow.Offset), shadow.ToPaint())
	}
	outline := ui.NewPaint()
	outline.SetColor(ui.NewColor(0x0A000000))
	canvas.DrawRSuperellipse(thumbShape.Inflate(0.5), outline)
	fill := ui.NewPaint()
	fill.SetColor(r.thumbColor)
	canvas.DrawRSuperellipse(thumbShape, fill)
}

func (r *renderSlidingSegmentedControl[T]) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	for child := r.LastChild(); child != nil; child = slidingParent(child).PreviousSibling {
		pd := slidingParent(child)
		if rectFromOffsetAndSize(pd.Offset, child.Size()).Contains(position) {
			offset := pd.Offset
			return result.AddWithPaintOffset(&offset, position, func(result *rendering.BoxHitTestResult, local ui.Offset) bool {
				return child.HitTest(result, local)
			})
		}
	}
	return false
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoSlidingSegmentedControlTooFewChildren      = "CupertinoSlidingSegmentedControlTooFewChildren"
	tagCupertinoSlidingSegmentedControlDuplicateValue      = "CupertinoSlidingSegmentedControlDuplicateValue"
	tagCupertinoSlidingSegmentedControlGroupValueNotAChild = "CupertinoSlidingSegmentedControlGroupValueNotAChild"
)
