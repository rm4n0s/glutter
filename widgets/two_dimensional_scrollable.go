package widgets

import (
	"math"
	"reflect"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// DiagonalDragBehavior is how a two-dimensional scrollable responds to a drag
// that is not along an axis.
//
// Flutter: widgets/scroll_view.dart DiagonalDragBehavior
type DiagonalDragBehavior int

const (
	// DiagonalDragBehaviorNone allows scrolling along one axis at a time: the
	// one the drag starts along. It is the zero value.
	//
	// Flutter: widgets/scroll_view.dart DiagonalDragBehavior.none
	DiagonalDragBehaviorNone DiagonalDragBehavior = iota
	// DiagonalDragBehaviorWeightedEvent scrolls along both axes until one wins
	// the gesture, which is decided once per drag.
	//
	// Flutter: widgets/scroll_view.dart DiagonalDragBehavior.weightedEvent
	DiagonalDragBehaviorWeightedEvent
	// DiagonalDragBehaviorWeightedContinuous scrolls along both axes and weighs
	// the axes again at every update.
	//
	// Flutter: widgets/scroll_view.dart DiagonalDragBehavior.weightedContinuous
	DiagonalDragBehaviorWeightedContinuous
	// DiagonalDragBehaviorFree scrolls along both axes freely.
	//
	// Flutter: widgets/scroll_view.dart DiagonalDragBehavior.free
	DiagonalDragBehaviorFree
)

// TwoDimensionalViewportBuilder builds the viewport of a TwoDimensionalScrollable
// with the offset of each axis.
//
// Flutter: widgets/scrollable.dart TwoDimensionalViewportBuilder
type TwoDimensionalViewportBuilder func(bc BuildContext, verticalPosition, horizontalPosition rendering.ViewportOffset) Widget

// twoDimensionalDimension marks a Scrollable as the vertical outer or the
// horizontal inner scrollable of a TwoDimensionalScrollable.
//
// Flutter: widgets/scrollable.dart _VerticalOuterDimension, _HorizontalInnerDimension
type twoDimensionalDimension struct {
	// vertical is true for the outer scrollable, false for the inner one.
	vertical bool
	// otherKey is the key of the horizontal scrollable (for the vertical one) or
	// of the vertical one (for the horizontal one).
	otherKey             *GlobalKey[*ScrollableState]
	diagonalDragBehavior DiagonalDragBehavior
}

func (s *ScrollableState) horizontalScrollableOf(dim *twoDimensionalDimension) *ScrollableState {
	st, _ := dim.otherKey.CurrentState()
	return st
}

func (s *ScrollableState) evaluateLockedAxis(offset ui.Offset) {
	offsetDelta := s.lastDragOffset.Value().Sub(offset)
	axisDifferential := math.Abs(offsetDelta.Dx) - math.Abs(offsetDelta.Dy)
	if math.Abs(axisDifferential) >= gestures.TouchSlop {
		// We have a single axis winner.
		if axisDifferential > 0 {
			s.lockedAxis = foundation.Some(painting.AxisHorizontal)
		} else {
			s.lockedAxis = foundation.Some(painting.AxisVertical)
		}
	} else {
		s.lockedAxis = foundation.None[painting.Axis]()
	}
}

// outerDragDown is the drag down of the vertical scrollable.
//
// Flutter: widgets/scrollable.dart _VerticalOuterDimensionState._handleDragDown
func (s *ScrollableState) outerDragDown(dim *twoDimensionalDimension, details gestures.DragDownDetails) {
	if dim.diagonalDragBehavior != DiagonalDragBehaviorNone {
		// Initiate a hold. If one or the other wins the gesture, the opposite
		// axis is cancelled.
		s.horizontalScrollableOf(dim).handleDragDown(details)
	}
	s.baseDragDown(details)
}

// outerDragStart is the drag start of the vertical scrollable.
//
// Flutter: widgets/scrollable.dart _VerticalOuterDimensionState._handleDragStart
func (s *ScrollableState) outerDragStart(dim *twoDimensionalDimension, details gestures.DragStartDetails) {
	s.lastDragOffset = foundation.Some(details.GlobalPosition)
	switch dim.diagonalDragBehavior {
	case DiagonalDragBehaviorFree:
		// Prepare to scroll both; the vertical one is started below.
		s.horizontalScrollableOf(dim).handleDragStart(details)
	case DiagonalDragBehaviorWeightedEvent, DiagonalDragBehaviorWeightedContinuous:
		// See if one axis wins the drag.
		s.evaluateLockedAxis(details.GlobalPosition)
		axis, locked := s.lockedAxis.Get()
		switch {
		case !locked:
			// Prepare to scroll both, there is no winner yet.
			s.horizontalScrollableOf(dim).handleDragStart(details)
		case axis == painting.AxisHorizontal:
			// Prepare to scroll horizontally.
			s.horizontalScrollableOf(dim).handleDragStart(details)
			return
		}
	}
	s.baseDragStart(details)
}

// outerDragUpdate is the drag update of the vertical scrollable.
//
// Flutter: widgets/scrollable.dart _VerticalOuterDimensionState._handleDragUpdate
func (s *ScrollableState) outerDragUpdate(dim *twoDimensionalDimension, details gestures.DragUpdateDetails) {
	verticalDetails := gestures.NewDragUpdateDetails(details.GlobalPosition, gestures.DragUpdateOpts{
		SourceTimeStamp: details.SourceTimeStamp, Delta: ui.Offset{Dy: details.Delta.Dy},
		PrimaryDelta: foundation.Some(details.Delta.Dy), LocalPosition: foundation.Some(details.LocalPosition)})
	horizontalDetails := gestures.NewDragUpdateDetails(details.GlobalPosition, gestures.DragUpdateOpts{
		SourceTimeStamp: details.SourceTimeStamp, Delta: ui.Offset{Dx: details.Delta.Dx},
		PrimaryDelta: foundation.Some(details.Delta.Dx), LocalPosition: foundation.Some(details.LocalPosition)})
	switch dim.diagonalDragBehavior {
	case DiagonalDragBehaviorNone:
		// Default gesture handling.
		s.baseDragUpdate(verticalDetails)
		return
	case DiagonalDragBehaviorFree:
		// Scroll both axes.
		s.horizontalScrollableOf(dim).handleDragUpdate(horizontalDetails)
		s.baseDragUpdate(verticalDetails)
		return
	case DiagonalDragBehaviorWeightedContinuous:
		// Re-evaluate the locked axis for every update.
		s.evaluateLockedAxis(details.GlobalPosition)
		s.lastDragOffset = foundation.Some(details.GlobalPosition)
	case DiagonalDragBehaviorWeightedEvent:
		// Lock the axis only once per gesture.
		if s.lockedAxis.IsNone() && s.lastDragOffset.IsSome() {
			// A winner has not been declared yet; see if one axis has won.
			s.evaluateLockedAxis(details.GlobalPosition)
		}
	}
	axis, locked := s.lockedAxis.Get()
	switch {
	case !locked:
		// Scroll both; the vertical one is scrolled below.
		s.horizontalScrollableOf(dim).handleDragUpdate(horizontalDetails)
	case axis == painting.AxisHorizontal:
		s.horizontalScrollableOf(dim).handleDragUpdate(horizontalDetails)
		return
	}
	s.baseDragUpdate(verticalDetails)
}

// outerDragEnd is the drag end of the vertical scrollable.
//
// Flutter: widgets/scrollable.dart _VerticalOuterDimensionState._handleDragEnd
func (s *ScrollableState) outerDragEnd(dim *twoDimensionalDimension, details gestures.DragEndDetails) {
	s.lastDragOffset = foundation.None[ui.Offset]()
	s.lockedAxis = foundation.None[painting.Axis]()
	dx, dy := details.Velocity.PixelsPerSecond.Dx, details.Velocity.PixelsPerSecond.Dy
	verticalDetails := gestures.NewDragEndDetails(gestures.DragEndOpts{
		Velocity: gestures.Velocity{PixelsPerSecond: ui.Offset{Dy: dy}}, PrimaryVelocity: foundation.Some(dy)})
	horizontalDetails := gestures.NewDragEndDetails(gestures.DragEndOpts{
		Velocity: gestures.Velocity{PixelsPerSecond: ui.Offset{Dx: dx}}, PrimaryVelocity: foundation.Some(dx)})
	if dim.diagonalDragBehavior != DiagonalDragBehaviorNone {
		s.horizontalScrollableOf(dim).handleDragEnd(horizontalDetails)
	}
	s.baseDragEnd(verticalDetails)
}

// outerDragCancel is the drag cancel of the vertical scrollable.
//
// Flutter: widgets/scrollable.dart _VerticalOuterDimensionState._handleDragCancel
func (s *ScrollableState) outerDragCancel(dim *twoDimensionalDimension) {
	s.lastDragOffset = foundation.None[ui.Offset]()
	s.lockedAxis = foundation.None[painting.Axis]()
	if dim.diagonalDragBehavior != DiagonalDragBehaviorNone {
		s.horizontalScrollableOf(dim).handleDragCancel()
	}
	s.baseDragCancel()
}

// dimensionSetCanDrag is SetCanDrag of the two scrollables when the drag may be
// diagonal: the vertical one gets a pan recognizer, which routes the horizontal
// deltas to the horizontal one, and the horizontal one gets none.
//
// Flutter: widgets/scrollable.dart _VerticalOuterDimensionState.setCanDrag
func (s *ScrollableState) dimensionSetCanDrag(dim *twoDimensionalDimension, value bool) {
	if !value {
		return
	}
	if dim.vertical {
		// Replaces the typical vertical/horizontal drag gesture recognizers with a
		// pan gesture recognizer to allow bidirectional scrolling.
		binding := WidgetsBindingOf(s.Context()).Gestures()
		f := NewGestureRecognizerFactoryWithHandlers(
			func() *gestures.PanGestureRecognizer {
				return gestures.NewPanGestureRecognizer(binding, gestures.DragGestureRecognizerOpts{SupportedDevices: s.configuration.DragDevices()})
			},
			func(r *gestures.PanGestureRecognizer) {
				r.OnDown = s.handleDragDown
				r.OnStart = s.handleDragStart
				r.OnUpdate = s.handleDragUpdate
				r.OnEnd = s.handleDragEnd
				r.OnCancel = s.handleDragCancel
				if s.physics != nil {
					r.MinFlingDistance = foundation.Some(s.physics.MinFlingDistance())
					r.MinFlingVelocity = foundation.Some(s.physics.MinFlingVelocity())
					r.MaxFlingVelocity = foundation.Some(s.physics.MaxFlingVelocity())
				}
				r.VelocityTrackerBuilder = s.configuration.VelocityTrackerBuilder(s.Context())
				r.DragStartBehavior = s.Widget().dragStartBehavior
				r.GestureSettings = s.mediaQueryGestureSt
			})
		s.gestureRecognizers = map[reflect.Type]GestureRecognizerFactory{f.RecognizerType(): f}
	} else {
		// A diagonal scroll creates a pan recognizer for the vertical scrollable,
		// so the horizontal one needs none; but the outer dimension must be updated
		// in case it did not have enough content to enable dragging.
		s.gestureRecognizers = map[reflect.Type]GestureRecognizerFactory{}
		if outer, ok := dim.otherKey.CurrentState(); ok {
			outer.SetCanDrag(value)
		}
	}
	// Cancel the active hold/drag (if any) because the gesture recognizers will
	// soon be disposed by the RawGestureDetector, and no pointer up events arrive
	// to cancel the hold/drag.
	s.handleDragCancel()
	s.lastCanDrag = foundation.Some(value)
	s.lastAxis = foundation.Some(s.Widget().Axis())
	if state, ok := s.gestureDetectorKey.CurrentState(); ok {
		state.ReplaceGestureRecognizers(s.gestureRecognizers)
	}
}

// dimensionEnsureVisible reveals the object in both axes, from the horizontal
// scrollable; the vertical one does nothing.
//
// Flutter: widgets/scrollable.dart _HorizontalInnerDimensionState._performEnsureVisible
func (s *ScrollableState) dimensionEnsureVisible(dim *twoDimensionalDimension, object rendering.RenderObject, opts EnsureVisibleOpts) (*scheduler.TickerFuture, *ScrollableState) {
	if dim.vertical {
		if foundation.DebugMode {
			panic(errors.New("TwoDimensionalEnsureVisibleVertical", "the performEnsureVisible method was called for the vertical scrollable of a TwoDimensionalScrollable; this should not happen as the horizontal scrollable handles both axes"))
		}
		return scheduler.NewTickerFutureComplete(), s
	}
	futures := []*scheduler.TickerFuture{
		s.position.EnsureVisible(object, opts),
		s.verticalScrollable.position.EnsureVisible(object, opts),
	}
	return scheduler.WaitTickerFutures(futures...), s.verticalScrollable
}

// TwoDimensionalScrollableOpts are the optional parameters of
// NewTwoDimensionalScrollable.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollable.new
type TwoDimensionalScrollableOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// IncrementCalculator calculates how far to scroll for a keyboard scroll.
	IncrementCalculator ScrollIncrementCalculator
	// ExcludeFromSemantics is whether to leave the scrollables out of the
	// semantics tree.
	ExcludeFromSemantics bool
	// DiagonalDragBehavior is how the scrollable responds to a diagonal drag;
	// the default is DiagonalDragBehaviorNone.
	DiagonalDragBehavior DiagonalDragBehavior
	// DragStartBehavior is where a drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// HitTestBehavior is how the scrollables behave during hit testing; the
	// default is HitTestBehaviorOpaque.
	HitTestBehavior rendering.HitTestBehavior
}

// TwoDimensionalScrollable is a widget with two Scrollables, one nested in the
// other, that scrolls along both axes: it builds a viewport with the offset of
// each of them.
//
// Deviation: it has no restorationId (SPEC §12.10).
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollable
type TwoDimensionalScrollable struct {
	StatefulWidget
	horizontalDetails ScrollableDetails
	verticalDetails   ScrollableDetails
	viewportBuilder   TwoDimensionalViewportBuilder
	opts              TwoDimensionalScrollableOpts
}

// NewTwoDimensionalScrollable creates the widget.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollable.new
func NewTwoDimensionalScrollable(horizontalDetails, verticalDetails ScrollableDetails, viewportBuilder TwoDimensionalViewportBuilder, opts ...TwoDimensionalScrollableOpts) *TwoDimensionalScrollable {
	o := firstOpts(opts)
	if o.HitTestBehavior == rendering.HitTestBehaviorUnspecified {
		o.HitTestBehavior = rendering.HitTestBehaviorOpaque
	}
	if o.DragStartBehavior == gestures.DragStartBehaviorUnspecified {
		o.DragStartBehavior = gestures.DragStartBehaviorStart
	}
	return &TwoDimensionalScrollable{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), horizontalDetails, verticalDetails, viewportBuilder, o}
}

// VerticalDetails are the details of the vertical axis.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollable.verticalDetails
func (w *TwoDimensionalScrollable) VerticalDetails() ScrollableDetails { return w.verticalDetails }

// HorizontalDetails are the details of the horizontal axis.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollable.horizontalDetails
func (w *TwoDimensionalScrollable) HorizontalDetails() ScrollableDetails { return w.horizontalDetails }

// DiagonalDragBehavior is how the scrollable responds to a diagonal drag.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollable.diagonalDragBehavior
func (w *TwoDimensionalScrollable) DiagonalDragBehavior() DiagonalDragBehavior {
	return w.opts.DiagonalDragBehavior
}

// DragStartBehavior is where a drag starts.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollable.dragStartBehavior
func (w *TwoDimensionalScrollable) DragStartBehavior() gestures.DragStartBehavior {
	return w.opts.DragStartBehavior
}

// CreateState creates the state.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollable.createState
func (w *TwoDimensionalScrollable) CreateState() State {
	return &TwoDimensionalScrollableState{
		verticalOuterKey:   NewGlobalKey[*ScrollableState](),
		horizontalInnerKey: NewGlobalKey[*ScrollableState](),
	}
}

// TwoDimensionalScrollableMaybeOf is the state of the closest
// TwoDimensionalScrollable above the context, and false if there is none.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollable.maybeOf
func TwoDimensionalScrollableMaybeOf(bc BuildContext) (*TwoDimensionalScrollableState, bool) {
	if w, ok := DependOn[*twoDimensionalScrollableScope](bc); ok {
		return w.twoDimensionalScrollable, true
	}
	return nil, false
}

// TwoDimensionalScrollableOf is the state of the closest TwoDimensionalScrollable
// above the context. In debug builds it panics with the tag
// TwoDimensionalScrollableMissing when there is none.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollable.of
func TwoDimensionalScrollableOf(bc BuildContext) *TwoDimensionalScrollableState {
	s, ok := TwoDimensionalScrollableMaybeOf(bc)
	if !ok && foundation.DebugMode {
		panic(errors.New("TwoDimensionalScrollableMissing",
			"TwoDimensionalScrollableOf() was called with a context that does not contain a TwoDimensionalScrollable widget; this can happen because you are using a widget that looks for a TwoDimensionalScrollable ancestor, but no such ancestor exists",
			"context", bc.(Element).String()))
	}
	return s
}

// TwoDimensionalScrollableState is the state of a TwoDimensionalScrollable.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollableState
type TwoDimensionalScrollableState struct {
	StateOf[*TwoDimensionalScrollable]
	verticalFallbackController   ScrollController
	horizontalFallbackController ScrollController
	verticalOuterKey             *GlobalKey[*ScrollableState]
	horizontalInnerKey           *GlobalKey[*ScrollableState]
}

// VerticalScrollable is the state of the vertical scrollable.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollableState.verticalScrollable
func (s *TwoDimensionalScrollableState) VerticalScrollable() *ScrollableState {
	st, ok := s.verticalOuterKey.CurrentState()
	if !ok && foundation.DebugMode {
		panic(errors.New("TwoDimensionalScrollableNotBuilt", "the vertical scrollable has not been built yet"))
	}
	return st
}

// HorizontalScrollable is the state of the horizontal scrollable.
//
// Flutter: widgets/scrollable.dart TwoDimensionalScrollableState.horizontalScrollable
func (s *TwoDimensionalScrollableState) HorizontalScrollable() *ScrollableState {
	st, ok := s.horizontalInnerKey.CurrentState()
	if !ok && foundation.DebugMode {
		panic(errors.New("TwoDimensionalScrollableNotBuilt", "the horizontal scrollable has not been built yet"))
	}
	return st
}

// InitState makes the fallback controllers for the details without a controller.
func (s *TwoDimensionalScrollableState) InitState() {
	w := s.Widget()
	if isNil(w.verticalDetails.Controller) {
		s.verticalFallbackController = NewScrollController()
	}
	if isNil(w.horizontalDetails.Controller) {
		s.horizontalFallbackController = NewScrollController()
	}
}

// DidUpdateWidget handles the changes in the provided and fallback controllers.
func (s *TwoDimensionalScrollableState) DidUpdateWidget(oldWidget *TwoDimensionalScrollable) {
	w := s.Widget()
	update := func(old, current ScrollController, fallback *ScrollController) {
		if old == current {
			return
		}
		if isNil(old) {
			// The old controller was null, meaning the fallback is not.
			(*fallback).Dispose()
			*fallback = nil
		} else if isNil(current) {
			// The new controller is null, so the fallback is set up.
			*fallback = NewScrollController()
		}
	}
	update(oldWidget.verticalDetails.Controller, w.verticalDetails.Controller, &s.verticalFallbackController)
	update(oldWidget.horizontalDetails.Controller, w.horizontalDetails.Controller, &s.horizontalFallbackController)
}

func scrollableClip(d ScrollableDetails) ui.Clip {
	return d.DecorationClipBehavior.OrElse(ui.ClipHardEdge)
}

// Build builds the two nested scrollables.
func (s *TwoDimensionalScrollableState) Build(bc BuildContext) Widget {
	w := s.Widget()
	if foundation.DebugMode {
		if painting.AxisDirectionToAxis(w.verticalDetails.Direction) != painting.AxisVertical {
			panic(errors.New("TwoDimensionalScrollableAxisInvalid", "TwoDimensionalScrollable.verticalDetails are not Axis.vertical"))
		}
		if painting.AxisDirectionToAxis(w.horizontalDetails.Direction) != painting.AxisHorizontal {
			panic(errors.New("TwoDimensionalScrollableAxisInvalid", "TwoDimensionalScrollable.horizontalDetails are not Axis.horizontal"))
		}
	}
	verticalController, horizontalController := w.verticalDetails.Controller, w.horizontalDetails.Controller
	if isNil(verticalController) {
		verticalController = s.verticalFallbackController
	}
	if isNil(horizontalController) {
		horizontalController = s.horizontalFallbackController
	}
	vertical := NewScrollable(func(bc BuildContext, verticalOffset rendering.ViewportOffset) Widget {
		horizontal := NewScrollable(func(bc BuildContext, horizontalOffset rendering.ViewportOffset) Widget {
			return w.viewportBuilder(bc, verticalOffset, horizontalOffset)
		}, ScrollableOpts{
			Key: s.horizontalInnerKey, AxisDirection: w.horizontalDetails.Direction, Controller: horizontalController, Physics: w.horizontalDetails.Physics,
			ClipBehavior: scrollableClip(w.horizontalDetails), IncrementCalculator: w.opts.IncrementCalculator,
			ExcludeFromSemantics: w.opts.ExcludeFromSemantics, DragStartBehavior: w.opts.DragStartBehavior, HitTestBehavior: w.opts.HitTestBehavior,
		})
		horizontal.dimension = &twoDimensionalDimension{vertical: false, otherKey: s.verticalOuterKey, diagonalDragBehavior: w.opts.DiagonalDragBehavior}
		return horizontal
	}, ScrollableOpts{
		Key: s.verticalOuterKey, AxisDirection: w.verticalDetails.Direction, Controller: verticalController, Physics: w.verticalDetails.Physics,
		ClipBehavior: scrollableClip(w.verticalDetails), IncrementCalculator: w.opts.IncrementCalculator,
		ExcludeFromSemantics: w.opts.ExcludeFromSemantics, DragStartBehavior: w.opts.DragStartBehavior, HitTestBehavior: w.opts.HitTestBehavior,
	})
	// For gesture forwarding.
	vertical.dimension = &twoDimensionalDimension{vertical: true, otherKey: s.horizontalInnerKey, diagonalDragBehavior: w.opts.DiagonalDragBehavior}
	return &twoDimensionalScrollableScope{NewInheritedWidgetBase(vertical), s}
}

// Dispose disposes the fallback controllers.
func (s *TwoDimensionalScrollableState) Dispose() {
	if !isNil(s.verticalFallbackController) {
		s.verticalFallbackController.Dispose()
	}
	if !isNil(s.horizontalFallbackController) {
		s.horizontalFallbackController.Dispose()
	}
}

// twoDimensionalScrollableScope lets TwoDimensionalScrollableOf work as if the
// state were an inherited widget: it never notifies.
//
// Flutter: widgets/scrollable.dart _TwoDimensionalScrollableScope
type twoDimensionalScrollableScope struct {
	InheritedWidget
	twoDimensionalScrollable *TwoDimensionalScrollableState
}

func (w *twoDimensionalScrollableScope) UpdateShouldNotify(Widget) bool { return false }
