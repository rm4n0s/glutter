// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"slices"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// TooltipComponentBuilder is the type of the function that builds the tooltip
// component of a RawTooltip; the animation goes from 0 to 1 as the tooltip
// shows.
//
// Flutter: widgets/raw_tooltip.dart TooltipComponentBuilder
type TooltipComponentBuilder func(bc BuildContext, animation animation.Animation[float64]) Widget

// TooltipPositionDelegate is the type of the function that gives the position
// of the tooltip, in the coordinates of the overlay.
//
// Flutter: widgets/raw_tooltip.dart TooltipPositionDelegate
type TooltipPositionDelegate func(context TooltipPositionContext) ui.Offset

// TooltipPositionContext is the context for a TooltipPositionDelegate.
//
// Flutter: widgets/raw_tooltip.dart TooltipPositionContext
type TooltipPositionContext struct {
	// Target is the center of the widget that the tooltip is for, in the
	// coordinates of the overlay.
	//
	// Flutter: widgets/raw_tooltip.dart TooltipPositionContext.target
	Target ui.Offset
	// TargetSize is the size of the widget that the tooltip is for.
	//
	// Flutter: widgets/raw_tooltip.dart TooltipPositionContext.targetSize
	TargetSize ui.Size
	// TooltipSize is the size of the tooltip.
	//
	// Flutter: widgets/raw_tooltip.dart TooltipPositionContext.tooltipSize
	TooltipSize ui.Size
	// VerticalOffset is the vertical distance between the target and the tooltip.
	//
	// Flutter: widgets/raw_tooltip.dart TooltipPositionContext.verticalOffset
	VerticalOffset float64
	// NoPreferBelow makes the tooltip prefer to be above the target (Flutter's
	// preferBelow is true by default).
	NoPreferBelow bool
	// OverlaySize is the size of the overlay; the default (zero) is unbounded.
	//
	// Flutter: widgets/raw_tooltip.dart TooltipPositionContext.overlaySize
	OverlaySize ui.Size
}

// PreferBelow is whether the tooltip prefers to be below the target.
//
// Flutter: widgets/raw_tooltip.dart TooltipPositionContext.preferBelow
func (c TooltipPositionContext) PreferBelow() bool { return !c.NoPreferBelow }

// TooltipTriggerMode is the mode of the gesture that triggers a tooltip.
//
// Flutter: widgets/raw_tooltip.dart TooltipTriggerMode
type TooltipTriggerMode int

const (
	// TooltipTriggerModeUnspecified is the zero value: the default, long press.
	TooltipTriggerModeUnspecified TooltipTriggerMode = iota
	// TooltipTriggerModeManual does not trigger the tooltip with a gesture: it
	// shows on hover, or when ensureTooltipVisible is called.
	//
	// Flutter: widgets/raw_tooltip.dart TooltipTriggerMode.manual
	TooltipTriggerModeManual
	// TooltipTriggerModeLongPress triggers the tooltip with a long press.
	//
	// Flutter: widgets/raw_tooltip.dart TooltipTriggerMode.longPress
	TooltipTriggerModeLongPress
	// TooltipTriggerModeTap triggers the tooltip with a tap.
	//
	// Flutter: widgets/raw_tooltip.dart TooltipTriggerMode.tap
	TooltipTriggerModeTap
)

// Name is the name of the mode.
//
// Flutter: widgets/raw_tooltip.dart TooltipTriggerMode.name
func (m TooltipTriggerMode) Name() string {
	switch m {
	case TooltipTriggerModeManual:
		return "manual"
	case TooltipTriggerModeLongPress:
		return "longPress"
	case TooltipTriggerModeTap:
		return "tap"
	}
	return "unspecified"
}

// String is "TooltipTriggerMode." and the name.
//
// Flutter: widgets/raw_tooltip.dart TooltipTriggerMode.toString
func (m TooltipTriggerMode) String() string { return "TooltipTriggerMode." + m.Name() }

// rawTooltipRegistry has the state that the raw tooltips of an app share.
//
// Deviation: Flutter keeps the open tooltips in a static list of RawTooltip
// (SPEC §12.10).
//
// Flutter: widgets/raw_tooltip.dart RawTooltip._openedTooltips
type rawTooltipRegistry struct {
	opened []*RawTooltipState
	mouse  rendering.ExclusiveMouseRegionState
}

// RawTooltipDismissAll dismisses all the open tooltips of the app, and says
// whether there were any.
//
// Flutter: widgets/raw_tooltip.dart RawTooltip.dismissAllToolTips
func RawTooltipDismissAll(bc BuildContext) bool {
	r := &WidgetsBindingOf(bc).rawTooltips
	if len(r.opened) == 0 {
		return false
	}
	// Avoid concurrent modification.
	for _, s := range slices.Clone(r.opened) {
		s.scheduleDismissTooltip(0)
	}
	return true
}

// defaultTooltipAnimationStyle is the animation style of a tooltip.
//
// Flutter: widgets/raw_tooltip.dart _kDefaultAnimationStyle
func defaultTooltipAnimationStyle() animation.AnimationStyle {
	return animation.AnimationStyle{
		Curve: animation.Curves.FastOutSlowIn, Duration: foundation.Some(150 * time.Millisecond), ReverseDuration: foundation.Some(75 * time.Millisecond),
	}
}

// RawTooltipOpts are the optional parameters of NewRawTooltip.
//
// Flutter: widgets/raw_tooltip.dart RawTooltip.new
type RawTooltipOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// HoverDelay is the delay before the tooltip shows when a mouse hovers the
	// child; the default is zero.
	HoverDelay time.Duration
	// TouchDelay is how long the tooltip stays after a touch trigger; the
	// default is 1.5 seconds.
	TouchDelay foundation.Opt[time.Duration]
	// DismissDelay is the delay before the tooltip is dismissed when a mouse
	// leaves; the default is 100 milliseconds.
	DismissDelay foundation.Opt[time.Duration]
	// DisableTapToDismiss stops a tap elsewhere from dismissing the tooltip
	// (Flutter's enableTapToDismiss is true by default).
	DisableTapToDismiss bool
	// TriggerMode is the gesture that triggers the tooltip; the default is long press.
	TriggerMode TooltipTriggerMode
	// DisableFeedback turns the feedback of the trigger off (Flutter's
	// enableFeedback is true by default).
	DisableFeedback bool
	// OnTriggered is called when the tooltip is triggered by a gesture.
	OnTriggered foundation.VoidCallback
	// AnimationStyle overrides the curve and the durations of the animation of
	// the tooltip.
	AnimationStyle animation.AnimationStyle
	// PositionDelegate gives the position of the tooltip; the default puts it
	// below the target, or above if it doesn't fit.
	PositionDelegate TooltipPositionDelegate
	// IgnorePointer makes the tooltip ignore pointer events.
	IgnorePointer bool
}

// RawTooltip is a widget that shows a tooltip component when a gesture
// triggers it, or a mouse hovers the child, and that has no style of its own.
//
// Deviation: the announcement of the tooltip to the semantics service is not
// ported (SPEC §12.10).
//
// Flutter: widgets/raw_tooltip.dart RawTooltip
type RawTooltip struct {
	StatefulWidget
	semanticsTooltip foundation.Opt[string]
	tooltipBuilder   TooltipComponentBuilder
	child            Widget
	o                RawTooltipOpts
}

// NewRawTooltip creates the tooltip. The semanticsTooltip is the text for
// accessibility; when it is the empty string, the tooltip is not created.
//
// Flutter: widgets/raw_tooltip.dart RawTooltip.new
func NewRawTooltip(semanticsTooltip foundation.Opt[string], tooltipBuilder TooltipComponentBuilder, child Widget, opts ...RawTooltipOpts) *RawTooltip {
	o := firstOpts(opts)
	if o.TriggerMode == TooltipTriggerModeUnspecified {
		o.TriggerMode = TooltipTriggerModeLongPress
	}
	def := defaultTooltipAnimationStyle()
	if o.AnimationStyle.Curve == nil {
		o.AnimationStyle.Curve = def.Curve
	}
	if o.AnimationStyle.Duration.IsNone() {
		o.AnimationStyle.Duration = def.Duration
	}
	if o.AnimationStyle.ReverseDuration.IsNone() {
		o.AnimationStyle.ReverseDuration = def.ReverseDuration
	}
	return &RawTooltip{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), semanticsTooltip: semanticsTooltip, tooltipBuilder: tooltipBuilder, child: child, o: o}
}

func (w *RawTooltip) touchDelay() time.Duration {
	return w.o.TouchDelay.OrElse(1500 * time.Millisecond)
}

func (w *RawTooltip) dismissDelay() time.Duration {
	return w.o.DismissDelay.OrElse(100 * time.Millisecond)
}

// CreateState creates the state.
//
// Flutter: widgets/raw_tooltip.dart RawTooltip.createState
func (w *RawTooltip) CreateState() State {
	return &RawTooltipState{overlayController: NewOverlayPortalController(), activeHoveringPointerDevices: map[int]struct{}{}, animationStatus: animation.AnimationStatusDismissed}
}

// RawTooltipState is the state of a RawTooltip.
//
// Flutter: widgets/raw_tooltip.dart RawTooltipState
type RawTooltipState struct {
	StateOf[*RawTooltip]
	SingleTickerProvider
	overlayController *OverlayPortalController

	timer             *gestures.Timer
	backingController *animation.AnimationController
	overlayAnimation  *animation.CurvedAnimation

	longPressRecognizer *gestures.LongPressGestureRecognizer
	tapRecognizer       *gestures.TapGestureRecognizer
	routeHandle         gestures.PointerRouteHandle

	// activeHoveringPointerDevices are the ids of mouse devices that are keeping
	// the tooltip from being dismissed. Device ids are added to this set in
	// handleMouseEnter, and removed in handleMouseExit. The set is cleared in
	// handleTapToDismiss, typically when a PointerDown event interacts with some
	// other UI component.
	activeHoveringPointerDevices map[int]struct{}
	animationStatus              animation.AnimationStatus
}

func (s *RawTooltipState) registry() *rawTooltipRegistry {
	return &WidgetsBindingOf(s.Context()).rawTooltips
}

func (s *RawTooltipState) controller() *animation.AnimationController {
	if s.backingController == nil {
		w := s.Widget()
		s.backingController = animation.NewAnimationController(&s.SingleTickerProvider, animation.AnimationControllerOpts{
			Duration: w.o.AnimationStyle.Duration, ReverseDuration: w.o.AnimationStyle.ReverseDuration,
		})
		s.backingController.AddStatusListener(s.handleStatusChanged)
	}
	return s.backingController
}

func (s *RawTooltipState) getOverlayAnimation() *animation.CurvedAnimation {
	if s.overlayAnimation == nil {
		curve := s.Widget().o.AnimationStyle.Curve
		s.overlayAnimation = animation.NewCurvedAnimation(s.controller(), curve)
	}
	return s.overlayAnimation
}

func (s *RawTooltipState) handleStatusChanged(status animation.AnimationStatus) {
	wasDismissed, isDismissed := s.animationStatus.IsDismissed(), status.IsDismissed()
	switch {
	case !wasDismissed && isDismissed:
		r := s.registry()
		if i := slices.Index(r.opened, s); i >= 0 {
			r.opened = slices.Delete(r.opened, i, i+1)
		}
		s.overlayController.Hide()
	case wasDismissed && !isDismissed:
		s.overlayController.Show()
		s.registry().opened = append(s.registry().opened, s)
	}
	s.animationStatus = status
}

func (s *RawTooltipState) gestureBinding() *gestures.GestureBinding {
	return WidgetsBindingOf(s.Context()).Gestures()
}

func (s *RawTooltipState) cancelTimer() {
	if s.timer != nil {
		s.timer.Cancel()
		s.timer = nil
	}
}

func (s *RawTooltipState) scheduleShowTooltip(withDelay time.Duration, touchDelay foundation.Opt[time.Duration]) {
	show := func() {
		s.controller().Forward()
		s.cancelTimer()
		if d, ok := touchDelay.Get(); ok {
			s.timer = s.gestureBinding().NewTimer(d, func() { s.controller().Reverse() })
		}
	}
	if s.controller().Status().IsDismissed() && withDelay > 0 {
		s.cancelTimer()
		s.timer = s.gestureBinding().NewTimer(withDelay, show)
	} else {
		// If the tooltip is already animating in or fully visible, skip the
		// animation and show the tooltip immediately.
		show()
	}
}

func (s *RawTooltipState) scheduleDismissTooltip(withDelay time.Duration) {
	s.cancelTimer()
	// Use the backing controller instead of the getter to prevent the lazy getter
	// from instantiating an AnimationController unnecessarily.
	if s.backingController != nil && s.backingController.Status().IsForwardOrCompleted() {
		// Dismiss when the tooltip is animating in: if there's a dismiss delay
		// we'll allow the animation to continue until the delay timer fires.
		if withDelay > 0 {
			s.timer = s.gestureBinding().NewTimer(withDelay, func() { s.controller().Reverse() })
		} else {
			s.controller().Reverse()
		}
	}
}

// triggerModeDeviceKinds are the PointerDeviceKinds that don't support hovering.
var triggerModeDeviceKinds = []ui.PointerDeviceKind{
	ui.PointerDeviceKindInvertedStylus, ui.PointerDeviceKindStylus, ui.PointerDeviceKindTouch, ui.PointerDeviceKindUnknown,
	// MouseRegion only tracks PointerDeviceKind == mouse.
	ui.PointerDeviceKindTrackpad,
}

func (s *RawTooltipState) handlePointerDown(event *gestures.PointerDownEvent) {
	w := s.Widget()
	switch w.o.TriggerMode {
	case TooltipTriggerModeLongPress:
		if s.longPressRecognizer == nil {
			s.longPressRecognizer = gestures.NewLongPressGestureRecognizer(s.gestureBinding(),
				gestures.LongPressGestureRecognizerOpts{DebugOwner: s, SupportedDevices: triggerModeDeviceKinds})
		}
		r := s.longPressRecognizer
		r.OnLongPressCancel = s.handleTapToDismiss
		r.OnLongPress = s.handleLongPress
		r.OnLongPressUp = s.handlePressUp
		r.AddPointer(event)
	case TooltipTriggerModeTap:
		if s.tapRecognizer == nil {
			s.tapRecognizer = gestures.NewTapGestureRecognizer(s.gestureBinding(),
				gestures.TapGestureRecognizerOpts{DebugOwner: s, SupportedDevices: triggerModeDeviceKinds})
		}
		r := s.tapRecognizer
		r.OnTapCancel = s.handleTapToDismiss
		r.OnTap = s.handleTap
		r.AddPointer(event)
	}
}

// handleGlobalPointerEvent is called for PointerDownEvents after handlePointerDown.
func (s *RawTooltipState) handleGlobalPointerEvent(event gestures.PointerEvent) {
	pointer := event.Pointer()
	if s.tapRecognizer != nil && s.tapRecognizer.PrimaryPointer() == foundation.Some(pointer) ||
		s.longPressRecognizer != nil && s.longPressRecognizer.PrimaryPointer() == foundation.Some(pointer) {
		// This is a pointer of interest specified by the trigger mode, since it's
		// picked up by the recognizer. The recognizer will later determine if this
		// is indeed a "trigger" gesture and dismiss the tooltip if that's not the
		// case. However there's still a chance that the event was cancelled before
		// the recognizer gets to emit a tap/longPress down, in which case the
		// cancel callback (handleTapToDismiss) will not be called.
		return
	}
	_, isDown := event.(*gestures.PointerDownEvent)
	if (s.timer == nil && s.controller().Status().IsDismissed()) || !isDown {
		return
	}
	s.handleTapToDismiss()
}

// handleTapToDismiss: the primary pointer is not part of a "trigger" gesture so
// the tooltip should be dismissed.
func (s *RawTooltipState) handleTapToDismiss() {
	if s.Widget().o.DisableTapToDismiss {
		return
	}
	s.scheduleDismissTooltip(0)
	clear(s.activeHoveringPointerDevices)
}

func (s *RawTooltipState) handleTap() {
	w := s.Widget()
	if s.controller().Status().IsDismissed() && !w.o.DisableFeedback {
		FeedbackForTap(s.Context())
	}
	if w.o.OnTriggered != nil {
		w.o.OnTriggered()
	}
	touchDelay := foundation.Some(w.touchDelay())
	// The hovering devices keep the tooltip visible.
	if len(s.activeHoveringPointerDevices) > 0 {
		touchDelay = foundation.None[time.Duration]()
	}
	s.scheduleShowTooltip(0, touchDelay)
}

// handleLongPress: a "trigger" gesture is recognized and the pointer down event
// is a part of it.
func (s *RawTooltipState) handleLongPress() {
	w := s.Widget()
	if s.controller().Status().IsDismissed() && !w.o.DisableFeedback {
		FeedbackForLongPress(s.Context())
	}
	if w.o.OnTriggered != nil {
		w.o.OnTriggered()
	}
	s.scheduleShowTooltip(0, foundation.None[time.Duration]())
}

func (s *RawTooltipState) handlePressUp() {
	if len(s.activeHoveringPointerDevices) > 0 {
		return
	}
	s.scheduleDismissTooltip(s.Widget().touchDelay())
}

// handleMouseEnter: current hovering behavior:
//  1. Hovered tooltips don't show more than one at a time, for each mouse
//     device. For example, a chip with a delete icon typically shouldn't show
//     both the delete icon tooltip and the chip tooltip at the same time.
//  2. Hovered tooltips are dismissed when: DismissAll is called, even these
//     tooltips are still hovered; an unrecognized PointerDownEvent occurred
//     within the application (even these tooltips are still hovered); the last
//     hovering device leaves the tooltip.
func (s *RawTooltipState) handleMouseEnter(event *gestures.PointerEnterEvent) {
	// handleMouseEnter is only called when the mouse starts to hover over this
	// tooltip (including the actual tooltip it shows on the overlay), and this
	// tooltip is the first to be hit in the widget tree's hit testing order.
	s.activeHoveringPointerDevices[event.Device()] = struct{}{}
	// Dismiss other open tooltips unless they're kept visible by other mice. The
	// mouse tracker implementation always dispatches all exit events before
	// dispatching any enter events, so the device must have already been removed
	// from the set of the tooltips that are no longer being hovered over.
	var toDismiss []*RawTooltipState
	for _, t := range s.registry().opened {
		if len(t.activeHoveringPointerDevices) == 0 {
			toDismiss = append(toDismiss, t)
		}
	}
	for _, t := range toDismiss {
		t.scheduleDismissTooltip(0)
	}
	delay := s.Widget().o.HoverDelay
	if len(toDismiss) > 0 {
		delay = 0
	}
	s.scheduleShowTooltip(delay, foundation.None[time.Duration]())
}

func (s *RawTooltipState) handleMouseExit(event *gestures.PointerExitEvent) {
	if len(s.activeHoveringPointerDevices) == 0 {
		return
	}
	delete(s.activeHoveringPointerDevices, event.Device())
	if len(s.activeHoveringPointerDevices) == 0 {
		s.scheduleDismissTooltip(s.Widget().dismissDelay())
	}
}

// EnsureTooltipVisible shows the tooltip if it is not already visible, and
// says whether it did.
//
// Flutter: widgets/raw_tooltip.dart RawTooltipState.ensureTooltipVisible
func (s *RawTooltipState) EnsureTooltipVisible() bool {
	s.cancelTimer()
	if s.controller().Status().IsForwardOrCompleted() {
		return false
	}
	s.scheduleShowTooltip(0, foundation.None[time.Duration]())
	return true
}

// InitState listens to the global pointer events, so that a tooltip is hidden
// immediately if some other control is clicked on. Pointer events are
// dispatched to global routes after other routes.
//
// Flutter: widgets/raw_tooltip.dart RawTooltipState.initState
func (s *RawTooltipState) InitState() {
	s.routeHandle = s.gestureBinding().PointerRouter.AddGlobalRoute(s.handleGlobalPointerEvent, nil)
}

func (s *RawTooltipState) buildTooltipOverlay(bc BuildContext, layoutInfo OverlayChildLayoutInfo) Widget {
	if layoutInfo.ChildPaintTransform.Determinant() == 0.0 {
		// The child is not visible.
		return NewSizedBoxShrink(nil)
	}
	w := s.Widget()
	target := layoutInfo.ChildPaintTransform.TransformPoint(ui.Offset{Dx: layoutInfo.ChildSize.Width / 2, Dy: layoutInfo.ChildSize.Height / 2})
	// Keep the tooltip visible while the overlay child is hovered.
	tooltip := NewIgnorePointer(
		s.exclusiveMouseRegion(w.tooltipBuilder(bc, s.getOverlayAnimation())),
		IgnorePointerOpts{Ignoring: foundation.Some(w.o.IgnorePointer)})
	bottom := 0.0
	if insets, ok := MediaQueryMaybeViewInsetsOf(bc); ok {
		bottom = insets.Bottom
	}
	overlayChild := Widget(NewPositionedFill(
		NewCustomSingleChildLayout(newTooltipPositionDelegate(target, layoutInfo.ChildSize, w.o.PositionDelegate), tooltip),
		PositionedFillOpts{Bottom: bottom}))
	if SelectionContainerMaybeOf(bc) == nil {
		return overlayChild
	}
	return NewSelectionContainerDisabled(overlayChild)
}

// exclusiveMouseRegion wraps the child in a mouse region that only the
// innermost tooltip region receives events for.
func (s *RawTooltipState) exclusiveMouseRegion(child Widget) Widget {
	return &exclusiveMouseRegion{
		SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child),
		state:                         &s.registry().mouse, onEnter: s.handleMouseEnter, onExit: s.handleMouseExit,
	}
}

// Dispose releases the recognizers, the timer and the animation.
//
// Flutter: widgets/raw_tooltip.dart RawTooltipState.dispose
func (s *RawTooltipState) Dispose() {
	s.gestureBinding().PointerRouter.RemoveGlobalRoute(s.routeHandle)
	r := s.registry()
	if i := slices.Index(r.opened, s); i >= 0 {
		r.opened = slices.Delete(r.opened, i, i+1)
	}
	// The recognizers may call their registered cancel callbacks if there's a
	// gesture in progress. Remove the callbacks to prevent unnecessary side
	// effects (such as animations).
	if s.longPressRecognizer != nil {
		s.longPressRecognizer.OnLongPressCancel = nil
		s.longPressRecognizer.Dispose()
	}
	if s.tapRecognizer != nil {
		s.tapRecognizer.OnTapCancel = nil
		s.tapRecognizer.Dispose()
	}
	s.cancelTimer()
	if s.backingController != nil {
		s.backingController.Dispose()
	}
	if s.overlayAnimation != nil {
		s.overlayAnimation.Dispose()
	}
}

// Build builds the child with the listeners and the overlay portal of the
// tooltip.
//
// Flutter: widgets/raw_tooltip.dart RawTooltipState.build
func (s *RawTooltipState) Build(BuildContext) Widget {
	w := s.Widget()
	// If message is empty then no need to create a tooltip overlay to show the
	// empty black container so just return the wrapped child as is.
	if t, ok := w.semanticsTooltip.Get(); ok && t == "" {
		return w.child
	}
	tooltip := w.semanticsTooltip.OrElse("")
	opts := SemanticsOpts{}
	if tooltip != "" {
		opts.Tooltip = foundation.Some(tooltip)
	}
	var result Widget = NewSemantics(w.child, opts)
	// Only check for gestures if tooltip should be visible.
	result = s.exclusiveMouseRegion(NewListener(result, ListenerOpts{OnPointerDown: s.handlePointerDown, Behavior: rendering.HitTestBehaviorOpaque}))
	return NewOverlayPortalOverlayChildLayoutBuilder(s.overlayController, s.buildTooltipOverlay, result)
}

// exclusiveMouseRegion is Flutter's _ExclusiveMouseRegion.
//
// Flutter: widgets/raw_tooltip.dart _ExclusiveMouseRegion
type exclusiveMouseRegion struct {
	SingleChildRenderObjectWidget
	state   *rendering.ExclusiveMouseRegionState
	onEnter func(*gestures.PointerEnterEvent)
	onExit  func(*gestures.PointerExitEvent)
}

func (w *exclusiveMouseRegion) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderExclusiveMouseRegion(w.state, nil, rendering.RenderMouseRegionOpts{OnEnter: w.onEnter, OnExit: w.onExit})
}

func (w *exclusiveMouseRegion) UpdateRenderObject(_ BuildContext, ro rendering.RenderObject) {
	r := ro.(*rendering.RenderExclusiveMouseRegion)
	r.SetOnEnter(w.onEnter)
	r.SetOnExit(w.onExit)
}

// tooltipPositionDelegate is Flutter's _TooltipPositionDelegate.
//
// Flutter: widgets/raw_tooltip.dart _TooltipPositionDelegate
type tooltipPositionDelegate struct {
	rendering.SingleChildLayoutDelegateBase
	target           ui.Offset
	targetSize       ui.Size
	positionDelegate TooltipPositionDelegate
}

func newTooltipPositionDelegate(target ui.Offset, targetSize ui.Size, positionDelegate TooltipPositionDelegate) *tooltipPositionDelegate {
	return &tooltipPositionDelegate{rendering.NewSingleChildLayoutDelegateBase(nil), target, targetSize, positionDelegate}
}

func (d *tooltipPositionDelegate) GetConstraintsForChild(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	return constraints.Loosen()
}

func (d *tooltipPositionDelegate) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	if d.positionDelegate != nil {
		return d.positionDelegate(TooltipPositionContext{
			Target: d.target, TargetSize: d.targetSize, TooltipSize: childSize, OverlaySize: size, VerticalOffset: 0,
		})
	}
	return painting.PositionDependentBox(painting.PositionDependentBoxOpts{Size: size, ChildSize: childSize, Target: d.target, PreferBelow: true})
}

func (d *tooltipPositionDelegate) ShouldRelayout(old rendering.SingleChildLayoutDelegate) bool {
	o := old.(*tooltipPositionDelegate)
	// Deviation: Dart compares the position delegates by identity; Go can't
	// compare functions, so any position delegate means a new layout.
	return d.target != o.target || d.targetSize != o.targetSize || d.positionDelegate != nil || o.positionDelegate != nil
}
