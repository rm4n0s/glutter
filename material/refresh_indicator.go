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
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	dragContainerExtentPercentage = 0.25
	dragSizeFactorLimit           = 1.5
	indicatorSnapDuration         = 150 * time.Millisecond
	indicatorScaleDuration        = 200 * time.Millisecond
)

// RefreshCallback is called when the user has dragged far enough to refresh. It
// starts the refresh and returns a channel that receives a value, or is closed,
// when the refresh is done; nil means that it is done at once.
//
// Flutter: material/refresh_indicator.dart RefreshCallback
type RefreshCallback func() <-chan struct{}

// RefreshIndicatorStatus is the state of a RefreshIndicator.
//
// Flutter: material/refresh_indicator.dart RefreshIndicatorStatus
type RefreshIndicatorStatus int

const (
	// RefreshIndicatorStatusNone is the state of an indicator that is not showing,
	// which Flutter has as null.
	RefreshIndicatorStatusNone RefreshIndicatorStatus = iota
	// RefreshIndicatorStatusDrag is the state while the user drags the content.
	//
	// Flutter: material/refresh_indicator.dart RefreshIndicatorStatus.drag
	RefreshIndicatorStatusDrag
	// RefreshIndicatorStatusArmed is the state when the drag has gone far enough to
	// start the refresh.
	//
	// Flutter: material/refresh_indicator.dart RefreshIndicatorStatus.armed
	RefreshIndicatorStatusArmed
	// RefreshIndicatorStatusSnap is the state while the indicator moves to the place
	// where it waits for the refresh.
	//
	// Flutter: material/refresh_indicator.dart RefreshIndicatorStatus.snap
	RefreshIndicatorStatusSnap
	// RefreshIndicatorStatusRefresh is the state while the refresh runs.
	//
	// Flutter: material/refresh_indicator.dart RefreshIndicatorStatus.refresh
	RefreshIndicatorStatusRefresh
	// RefreshIndicatorStatusDone is the state while the indicator goes away after the
	// refresh.
	//
	// Flutter: material/refresh_indicator.dart RefreshIndicatorStatus.done
	RefreshIndicatorStatusDone
	// RefreshIndicatorStatusCanceled is the state while the indicator goes away
	// after the drag was released too early.
	//
	// Flutter: material/refresh_indicator.dart RefreshIndicatorStatus.canceled
	RefreshIndicatorStatusCanceled
)

// Name is the name of the status; the empty status is "null", as in Flutter.
func (s RefreshIndicatorStatus) Name() string {
	return enumName(int(s), "null", "drag", "armed", "snap", "refresh", "done", "canceled")
}

// String is the name of the type and of the status.
func (s RefreshIndicatorStatus) String() string { return "RefreshIndicatorStatus." + s.Name() }

// RefreshIndicatorTriggerMode is when a drag can start the indicator.
//
// Flutter: material/refresh_indicator.dart RefreshIndicatorTriggerMode
type RefreshIndicatorTriggerMode int

const (
	// RefreshIndicatorTriggerModeUnspecified is the default, onEdge.
	RefreshIndicatorTriggerModeUnspecified RefreshIndicatorTriggerMode = iota
	// RefreshIndicatorTriggerModeAnywhere shows the indicator when the user drags
	// from the edge or from the middle of the content, once it is at its edge.
	//
	// Flutter: material/refresh_indicator.dart RefreshIndicatorTriggerMode.anywhere
	RefreshIndicatorTriggerModeAnywhere
	// RefreshIndicatorTriggerModeOnEdge shows the indicator only when the drag
	// starts while the content is at its edge.
	//
	// Flutter: material/refresh_indicator.dart RefreshIndicatorTriggerMode.onEdge
	RefreshIndicatorTriggerModeOnEdge
)

// Name is the name of the mode.
func (m RefreshIndicatorTriggerMode) Name() string {
	return enumName(int(m), "unspecified", "anywhere", "onEdge")
}

// String is the name of the type and of the mode.
func (m RefreshIndicatorTriggerMode) String() string {
	return "RefreshIndicatorTriggerMode." + m.Name()
}

type refreshIndicatorType int

const (
	refreshIndicatorMaterial refreshIndicatorType = iota
	refreshIndicatorAdaptive
	refreshIndicatorNoSpinner
)

// RefreshIndicatorOpts are the optional parameters of NewRefreshIndicator and of
// NewRefreshIndicatorAdaptive.
//
// Flutter: material/refresh_indicator.dart RefreshIndicator
type RefreshIndicatorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Displacement is the distance from the top of the child to the indicator;
	// the default is 40.
	Displacement foundation.Opt[float64]
	// EdgeOffset is the offset of the indicator from the edge of the scrollable;
	// the default is 0.
	EdgeOffset foundation.Opt[float64]
	// Color is the color of the arc; the primary color by default.
	Color foundation.Opt[ui.Color]
	// BackgroundColor is the color of the disc.
	BackgroundColor foundation.Opt[ui.Color]
	// NotificationPredicate says which scroll notifications the indicator reacts
	// to; the default is DefaultScrollNotificationPredicate.
	NotificationPredicate widgets.ScrollNotificationPredicate
	// SemanticsLabel is the label of the semantics of the indicator.
	SemanticsLabel string
	// SemanticsValue is the value of the semantics of the indicator.
	SemanticsValue string
	// StrokeWidth is the width of the arc; the default is
	// RefreshProgressIndicatorDefaultStrokeWidth.
	StrokeWidth foundation.Opt[float64]
	// TriggerMode is when a drag starts the indicator; the default is onEdge.
	TriggerMode RefreshIndicatorTriggerMode
	// Elevation is the elevation of the disc; the default is 2.
	Elevation foundation.Opt[float64]
}

// RefreshIndicatorNoSpinnerOpts are the optional parameters of
// NewRefreshIndicatorNoSpinner.
//
// Flutter: material/refresh_indicator.dart RefreshIndicator.noSpinner
type RefreshIndicatorNoSpinnerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnStatusChange is called when the status of the indicator changes.
	OnStatusChange func(status RefreshIndicatorStatus)
	// NotificationPredicate says which scroll notifications the indicator reacts
	// to; the default is DefaultScrollNotificationPredicate.
	NotificationPredicate widgets.ScrollNotificationPredicate
	// SemanticsLabel is the label of the semantics.
	SemanticsLabel string
	// SemanticsValue is the value of the semantics.
	SemanticsValue string
	// TriggerMode is when a drag starts the indicator; the default is onEdge.
	TriggerMode RefreshIndicatorTriggerMode
	// Elevation is the elevation (it must not be negative).
	Elevation foundation.Opt[float64]
}

// RefreshIndicator is a widget that supports the Material "swipe to refresh"
// idiom: it shows a circular progress indicator when the user drags the
// scrollable below it past its edge, and calls a callback to refresh the content.
//
// Flutter: material/refresh_indicator.dart RefreshIndicator
type RefreshIndicator struct {
	widgets.StatefulWidget
	child          widgets.Widget
	onRefresh      RefreshCallback
	onStatusChange func(RefreshIndicatorStatus)
	kind           refreshIndicatorType
	displacement   float64
	edgeOffset     float64
	o              RefreshIndicatorOpts
}

func newRefreshIndicator(onRefresh RefreshCallback, child widgets.Widget, o RefreshIndicatorOpts, kind refreshIndicatorType) *RefreshIndicator {
	if foundation.DebugMode && o.Elevation.OrElse(2.0) < 0 {
		panic(errors.New(tagRefreshIndicatorElevationNegative, "the elevation of a RefreshIndicator must not be negative", "elevation", o.Elevation.Value()))
	}
	if o.NotificationPredicate == nil {
		o.NotificationPredicate = widgets.DefaultScrollNotificationPredicate
	}
	return &RefreshIndicator{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, onRefresh: onRefresh, kind: kind,
		displacement: o.Displacement.OrElse(40.0), edgeOffset: o.EdgeOffset.OrElse(0.0), o: o}
}

// NewRefreshIndicator creates an indicator around the scrollable that calls
// onRefresh. In debug builds it panics with the tag
// RefreshIndicatorElevationNegative when the elevation is negative.
//
// Flutter: material/refresh_indicator.dart RefreshIndicator.new
func NewRefreshIndicator(onRefresh RefreshCallback, child widgets.Widget, opts ...RefreshIndicatorOpts) *RefreshIndicator {
	return newRefreshIndicator(onRefresh, child, firstOpts(opts), refreshIndicatorMaterial)
}

// NewRefreshIndicatorAdaptive creates an indicator that is a Cupertino one on
// iOS and macOS in Flutter.
//
// Deviation: Glutter has no Cupertino library, so the indicator is the Material
// one on every platform (SPEC §12.10).
//
// Flutter: material/refresh_indicator.dart RefreshIndicator.adaptive
func NewRefreshIndicatorAdaptive(onRefresh RefreshCallback, child widgets.Widget, opts ...RefreshIndicatorOpts) *RefreshIndicator {
	return newRefreshIndicator(onRefresh, child, firstOpts(opts), refreshIndicatorAdaptive)
}

// NewRefreshIndicatorNoSpinner creates an indicator that shows nothing, and that
// reports the status changes: the app draws the indicator.
//
// Flutter: material/refresh_indicator.dart RefreshIndicator.noSpinner
func NewRefreshIndicatorNoSpinner(onRefresh RefreshCallback, child widgets.Widget, opts ...RefreshIndicatorNoSpinnerOpts) *RefreshIndicator {
	o := firstOpts(opts)
	w := newRefreshIndicator(onRefresh, child, RefreshIndicatorOpts{Key: o.Key, NotificationPredicate: o.NotificationPredicate, SemanticsLabel: o.SemanticsLabel,
		SemanticsValue: o.SemanticsValue, TriggerMode: o.TriggerMode, Elevation: o.Elevation, StrokeWidth: foundation.Some(0.0)}, refreshIndicatorNoSpinner)
	w.displacement, w.edgeOffset = 0, 0
	w.onStatusChange = o.OnStatusChange
	return w
}

// CreateState creates the state.
//
// Flutter: material/refresh_indicator.dart RefreshIndicator.createState
func (w *RefreshIndicator) CreateState() widgets.State { return &RefreshIndicatorState{} }

// RefreshIndicatorState is the state of a RefreshIndicator; its Show method
// shows the indicator and starts the refresh from code.
//
// Flutter: material/refresh_indicator.dart RefreshIndicatorState
type RefreshIndicatorState struct {
	widgets.StateOf[*RefreshIndicator]
	widgets.TickerProvider
	positionController  *animation.AnimationController
	scaleController     *animation.AnimationController
	positionFactor      animation.Animation[float64]
	scaleFactor         animation.Animation[float64]
	value               animation.Animation[float64]
	valueColor          animation.Animation[foundation.Opt[ui.Color]]
	status              RefreshIndicatorStatus
	pendingRefresh      chan struct{}
	isIndicatorAtTop    foundation.Opt[bool]
	dragOffset          foundation.Opt[float64]
	effectiveValueColor ui.Color
}

var (
	threeQuarterTween      = animation.NewTween(0.0, 0.75)
	dragSizeFactorLimitTwn = animation.NewTween(0.0, dragSizeFactorLimit)
	oneToZeroTween         = animation.NewTween(1.0, 0.0)
)

// InitState creates the controllers of the position and of the scale.
//
// Flutter: material/refresh_indicator.dart RefreshIndicatorState.initState
func (s *RefreshIndicatorState) InitState() {
	s.positionController = animation.NewAnimationController(s, animation.AnimationControllerOpts{})
	s.positionFactor = animation.Drive[float64](s.positionController, dragSizeFactorLimitTwn)
	s.value = animation.Drive[float64](s.positionController, threeQuarterTween)
	s.scaleController = animation.NewAnimationController(s, animation.AnimationControllerOpts{})
	s.scaleFactor = animation.Drive[float64](s.scaleController, oneToZeroTween)
}

// DidChangeDependencies sets up the color of the arc.
//
// Flutter: material/refresh_indicator.dart RefreshIndicatorState.didChangeDependencies
func (s *RefreshIndicatorState) DidChangeDependencies() { s.setupColorTween() }

// DidUpdateWidget sets up the color of the arc again when it changed.
//
// Flutter: material/refresh_indicator.dart RefreshIndicatorState.didUpdateWidget
func (s *RefreshIndicatorState) DidUpdateWidget(old *RefreshIndicator) {
	if old.o.Color != s.Widget().o.Color {
		s.setupColorTween()
	}
}

// Dispose disposes the controllers.
//
// Flutter: material/refresh_indicator.dart RefreshIndicatorState.dispose
func (s *RefreshIndicatorState) Dispose() {
	s.positionController.Dispose()
	s.scaleController.Dispose()
}

func (s *RefreshIndicatorState) setupColorTween() {
	s.effectiveValueColor = s.Widget().o.Color.OrElse(ThemeOf(s.Context()).ColorScheme.Primary())
	color := s.effectiveValueColor
	if color.Alpha() == 0x00 {
		s.valueColor = animation.NewAlwaysStoppedAnimation(foundation.Some(color))
		return
	}
	tween := animation.NewColorTween(foundation.Some(color.WithAlpha(0)), foundation.Some(color)).
		Chain(animation.NewCurveTween(animation.NewInterval(0.0, 1.0/dragSizeFactorLimit)))
	s.valueColor = animation.Drive[foundation.Opt[ui.Color]](s.positionController, tween)
}

func (s *RefreshIndicatorState) changeStatus(status RefreshIndicatorStatus) {
	s.status = status
	if f := s.Widget().onStatusChange; f != nil {
		f(status)
	}
}

func (s *RefreshIndicatorState) shouldStart(notification widgets.ScrollNotification) bool {
	var dragged bool
	switch n := notification.(type) {
	case *widgets.ScrollStartNotification:
		dragged = n.DragDetails.IsSome()
	case *widgets.ScrollUpdateNotification:
		dragged = n.DragDetails.IsSome() && s.Widget().o.TriggerMode == RefreshIndicatorTriggerModeAnywhere
	}
	m := notification.Metrics()
	atEdge := (m.AxisDirection() == painting.AxisDirectionUp && m.ExtentAfter() == 0.0) ||
		(m.AxisDirection() == painting.AxisDirectionDown && m.ExtentBefore() == 0.0)
	return dragged && atEdge && s.status == RefreshIndicatorStatusNone && s.start(m.AxisDirection())
}

func (s *RefreshIndicatorState) handleScrollNotification(notification widgets.ScrollNotification) bool {
	w := s.Widget()
	if !w.o.NotificationPredicate(notification) {
		return false
	}
	if s.shouldStart(notification) {
		s.SetState(func() { s.changeStatus(RefreshIndicatorStatusDrag) })
		return false
	}
	var indicatorAtTopNow foundation.Opt[bool]
	switch notification.Metrics().AxisDirection() {
	case painting.AxisDirectionDown, painting.AxisDirectionUp:
		indicatorAtTopNow = foundation.Some(true)
	}
	dragging := s.status == RefreshIndicatorStatusDrag || s.status == RefreshIndicatorStatusArmed
	if indicatorAtTopNow != s.isIndicatorAtTop {
		if dragging {
			s.dismiss(RefreshIndicatorStatusCanceled)
		}
		return false
	}
	switch n := notification.(type) {
	case *widgets.ScrollUpdateNotification:
		if dragging {
			switch n.Metrics().AxisDirection() {
			case painting.AxisDirectionDown:
				s.dragOffset = foundation.Some(s.dragOffset.Value() - n.ScrollDelta.Value())
			case painting.AxisDirectionUp:
				s.dragOffset = foundation.Some(s.dragOffset.Value() + n.ScrollDelta.Value())
			}
			s.checkDragOffset(n.Metrics().ViewportDimension())
		}
		if s.status == RefreshIndicatorStatusArmed && n.DragDetails.IsNone() {
			s.show()
		}
	case *widgets.OverscrollNotification:
		if dragging {
			switch n.Metrics().AxisDirection() {
			case painting.AxisDirectionDown:
				s.dragOffset = foundation.Some(s.dragOffset.Value() - n.Overscroll)
			case painting.AxisDirectionUp:
				s.dragOffset = foundation.Some(s.dragOffset.Value() + n.Overscroll)
			}
			s.checkDragOffset(n.Metrics().ViewportDimension())
		}
	case *widgets.ScrollEndNotification:
		switch s.status {
		case RefreshIndicatorStatusArmed:
			if s.positionController.Value() < 1.0 {
				s.dismiss(RefreshIndicatorStatusCanceled)
			} else {
				s.show()
			}
		case RefreshIndicatorStatusDrag:
			s.dismiss(RefreshIndicatorStatusCanceled)
		}
	}
	return false
}

func (s *RefreshIndicatorState) start(direction painting.AxisDirection) bool {
	switch direction {
	case painting.AxisDirectionDown, painting.AxisDirectionUp:
		s.isIndicatorAtTop = foundation.Some(true)
	default:
		s.isIndicatorAtTop = foundation.None[bool]()
		return false
	}
	s.dragOffset = foundation.Some(0.0)
	s.scaleController.SetValue(0.0)
	s.positionController.SetValue(0.0)
	return true
}

func (s *RefreshIndicatorState) checkDragOffset(containerExtent float64) {
	newValue := s.dragOffset.Value() / (containerExtent * dragContainerExtentPercentage)
	if s.status == RefreshIndicatorStatusArmed {
		newValue = math.Max(newValue, 1.0/dragSizeFactorLimit)
	}
	s.positionController.SetValue(ui.ClampDouble(newValue, 0.0, 1.0)) // This triggers various rebuilds.
	if s.status == RefreshIndicatorStatusDrag && s.valueColor.Value().Value().Alpha() == s.effectiveValueColor.Alpha() {
		s.status = RefreshIndicatorStatusArmed
		s.changeStatus(s.status)
	}
}

// dismiss hides the indicator after the current microtask, as Flutter's async
// _dismiss does.
func (s *RefreshIndicatorState) dismiss(newMode RefreshIndicatorStatus) {
	widgets.WidgetsBindingOf(s.Context()).Post(func() {
		if !s.Mounted() {
			return
		}
		s.SetState(func() { s.changeStatus(newMode) })
		var future *scheduler.TickerFuture
		switch newMode {
		case RefreshIndicatorStatusDone:
			future = s.scaleController.AnimateTo(1.0, animation.AnimateOpts{Duration: foundation.Some(indicatorScaleDuration)})
		case RefreshIndicatorStatusCanceled:
			future = s.positionController.AnimateTo(0.0, animation.AnimateOpts{Duration: foundation.Some(indicatorScaleDuration)})
		default:
			return
		}
		future.Then(func() {
			if s.Mounted() && s.status == newMode {
				s.dragOffset = foundation.None[float64]()
				s.isIndicatorAtTop = foundation.None[bool]()
				s.SetState(func() { s.status = RefreshIndicatorStatusNone })
			}
		})
	})
}

func (s *RefreshIndicatorState) show() {
	s.pendingRefresh = make(chan struct{})
	pending := s.pendingRefresh
	s.changeStatus(RefreshIndicatorStatusSnap)
	binding := widgets.WidgetsBindingOf(s.Context())
	s.positionController.AnimateTo(1.0/dragSizeFactorLimit, animation.AnimateOpts{Duration: foundation.Some(indicatorSnapDuration)}).Then(func() {
		if !s.Mounted() || s.status != RefreshIndicatorStatusSnap {
			return
		}
		s.SetState(func() { s.status = RefreshIndicatorStatusRefresh })
		finish := func() {
			if s.Mounted() && s.status == RefreshIndicatorStatusRefresh {
				close(pending)
				s.dismiss(RefreshIndicatorStatusDone)
			}
		}
		result := s.Widget().onRefresh()
		if result == nil {
			finish()
			return
		}
		// The refresh may end on any goroutine: the indicator goes away on the UI
		// thread.
		go func() {
			<-result
			binding.Post(finish)
		}()
	})
}

// RefreshIndicatorShowOpts are the optional parameters of
// RefreshIndicatorState.Show.
//
// Flutter: material/refresh_indicator.dart RefreshIndicatorState.show
type RefreshIndicatorShowOpts struct {
	// AtBottom shows the indicator at the bottom of the scrollable; Flutter's
	// atTop: false.
	AtBottom bool
}

// Show shows the indicator at the top (or the bottom) and calls onRefresh, as if
// the user had dragged. The channel is closed when the refresh is done.
//
// Flutter: material/refresh_indicator.dart RefreshIndicatorState.show
func (s *RefreshIndicatorState) Show(opts ...RefreshIndicatorShowOpts) <-chan struct{} {
	o := firstOpts(opts)
	if s.status != RefreshIndicatorStatusRefresh && s.status != RefreshIndicatorStatusSnap {
		if s.status == RefreshIndicatorStatusNone {
			direction := painting.AxisDirectionDown
			if o.AtBottom {
				direction = painting.AxisDirectionUp
			}
			s.start(direction)
		}
		s.show()
	}
	return s.pendingRefresh
}

// Build builds the child and the indicator over it.
//
// Flutter: material/refresh_indicator.dart RefreshIndicatorState.build
func (s *RefreshIndicatorState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	child := widgets.NewNotificationListener[widgets.ScrollNotification](s.handleScrollNotification, w.child)
	showIndeterminate := s.status == RefreshIndicatorStatusRefresh || s.status == RefreshIndicatorStatusDone
	children := []widgets.Widget{child}
	if s.status != RefreshIndicatorStatusNone {
		atTop := s.isIndicatorAtTop.Value()
		positioned := widgets.PositionedOpts{Left: foundation.Some(0.0), Right: foundation.Some(0.0)}
		alignY, padding, align := -1.0, painting.NewEdgeInsetsFromLTRB(0, 0, 0, w.displacement), painting.AlignmentBottomCenter
		if atTop {
			positioned.Top = foundation.Some(w.edgeOffset)
			alignY, padding, align = 1.0, painting.NewEdgeInsetsFromLTRB(0, w.displacement, 0, 0), painting.AlignmentTopCenter
		} else {
			positioned.Bottom = foundation.Some(w.edgeOffset)
		}
		children = append(children, widgets.NewPositioned(
			widgets.NewSizeTransition(s.positionFactor,
				widgets.NewPadding(padding,
					widgets.NewAlign(
						widgets.NewScaleTransition(s.scaleFactor,
							widgets.NewAnimatedBuilder(s.positionController, func(bc widgets.BuildContext, _ widgets.Widget) widgets.Widget {
								label := w.o.SemanticsLabel
								if label == "" {
									label = MaterialLocalizationsOf(bc).RefreshIndicatorSemanticLabel()
								}
								value := foundation.Some(s.value.Value())
								if showIndeterminate {
									value = foundation.None[float64]()
								}
								if w.kind == refreshIndicatorNoSpinner {
									return widgets.NewContainer(nil)
								}
								return NewRefreshProgressIndicator(RefreshProgressIndicatorOpts{SemanticsLabel: label, SemanticsValue: w.o.SemanticsValue, Value: value,
									ValueColor: s.valueColor, BackgroundColor: w.o.BackgroundColor, StrokeWidth: foundation.Some(w.o.StrokeWidth.OrElse(RefreshProgressIndicatorDefaultStrokeWidth)),
									Elevation: w.o.Elevation})
							}, nil)),
						widgets.AlignOpts{Alignment: align})),
				widgets.SizeTransitionOpts{Alignment: painting.AlignmentDirectional{Start: -1.0, Y: alignY}}),
			positioned))
	}
	return widgets.NewStack(children)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagRefreshIndicatorElevationNegative = "RefreshIndicatorElevationNegative"
)
