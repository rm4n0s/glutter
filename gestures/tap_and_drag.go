package gestures

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func getGlobalDistance(event PointerEvent, origin OffsetPair) float64 {
	return event.Position().Sub(origin.Global).Distance()
}

// TapDragDownDetails is the data of a pointer that touched the screen and might
// start a tap or a drag.
//
// Flutter: gestures/tap_and_drag.dart TapDragDownDetails
type TapDragDownDetails struct {
	// GlobalPosition is where the pointer touched the screen.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// Kind is the kind of device.
	Kind foundation.Opt[ui.PointerDeviceKind]
	// ConsecutiveTapCount is the number of taps of the series, including this
	// one.
	ConsecutiveTapCount int
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/tap_and_drag.dart TapDragDownDetails.globalPosition
func (d TapDragDownDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/tap_and_drag.dart TapDragDownDetails.localPosition
func (d TapDragDownDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/tap_and_drag.dart TapDragDownDetails.debugFillProperties
func (d TapDragDownDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewEnumProperty("kind", d.Kind))
	p.Add(foundation.NewIntProperty("consecutiveTapCount", foundation.Some(d.ConsecutiveTapCount)))
}

// GestureTapDragDownCallback is called when a pointer that might tap or drag
// touches the screen.
//
// Flutter: gestures/tap_and_drag.dart GestureTapDragDownCallback
type GestureTapDragDownCallback func(details TapDragDownDetails)

// TapDragUpDetails is the data of a tap that ended.
//
// Flutter: gestures/tap_and_drag.dart TapDragUpDetails
type TapDragUpDetails struct {
	// GlobalPosition is where the pointer was lifted.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// Kind is the kind of device.
	Kind ui.PointerDeviceKind
	// ConsecutiveTapCount is the number of taps of the series, including this
	// one.
	ConsecutiveTapCount int
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/tap_and_drag.dart TapDragUpDetails.globalPosition
func (d TapDragUpDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/tap_and_drag.dart TapDragUpDetails.localPosition
func (d TapDragUpDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/tap_and_drag.dart TapDragUpDetails.debugFillProperties
func (d TapDragUpDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewEnumProperty("kind", foundation.Some(d.Kind)))
	p.Add(foundation.NewIntProperty("consecutiveTapCount", foundation.Some(d.ConsecutiveTapCount)))
}

// GestureTapDragUpCallback is called when a tap ends.
//
// Flutter: gestures/tap_and_drag.dart GestureTapDragUpCallback
type GestureTapDragUpCallback func(details TapDragUpDetails)

// TapDragStartDetails is the data of a drag that started after a tap.
//
// Flutter: gestures/tap_and_drag.dart TapDragStartDetails
type TapDragStartDetails struct {
	// GlobalPosition is where the drag started.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// SourceTimeStamp is the time of the event that started the drag.
	SourceTimeStamp foundation.Opt[time.Duration]
	// Kind is the kind of device.
	Kind foundation.Opt[ui.PointerDeviceKind]
	// ConsecutiveTapCount is the number of taps of the series.
	ConsecutiveTapCount int
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/tap_and_drag.dart TapDragStartDetails.globalPosition
func (d TapDragStartDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/tap_and_drag.dart TapDragStartDetails.localPosition
func (d TapDragStartDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/tap_and_drag.dart TapDragStartDetails.debugFillProperties
func (d TapDragStartDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewDiagnosticsPropertyOpt("sourceTimeStamp", d.SourceTimeStamp))
	p.Add(foundation.NewEnumProperty("kind", d.Kind))
	p.Add(foundation.NewIntProperty("consecutiveTapCount", foundation.Some(d.ConsecutiveTapCount)))
}

// GestureTapDragStartCallback is called when a drag starts after a tap.
//
// Flutter: gestures/tap_and_drag.dart GestureTapDragStartCallback
type GestureTapDragStartCallback func(details TapDragStartDetails)

// TapDragUpdateDetails is the data of a pointer that moved during a drag after a
// tap.
//
// Flutter: gestures/tap_and_drag.dart TapDragUpdateDetails
type TapDragUpdateDetails struct {
	// GlobalPosition is the global position of the pointer.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// SourceTimeStamp is the time of the event.
	SourceTimeStamp foundation.Opt[time.Duration]
	// Delta is how far the pointer moved since the last update.
	Delta ui.Offset
	// PrimaryDelta is the delta along the primary axis, if the drag has one.
	PrimaryDelta foundation.Opt[float64]
	// Kind is the kind of device.
	Kind foundation.Opt[ui.PointerDeviceKind]
	// OffsetFromOrigin is how far the pointer is from where the drag started,
	// in global coordinates.
	OffsetFromOrigin ui.Offset
	// LocalOffsetFromOrigin is OffsetFromOrigin in the coordinates of the
	// target.
	LocalOffsetFromOrigin ui.Offset
	// ConsecutiveTapCount is the number of taps of the series.
	ConsecutiveTapCount int
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/tap_and_drag.dart TapDragUpdateDetails.globalPosition
func (d TapDragUpdateDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/tap_and_drag.dart TapDragUpdateDetails.localPosition
func (d TapDragUpdateDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/tap_and_drag.dart TapDragUpdateDetails.debugFillProperties
func (d TapDragUpdateDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewDiagnosticsPropertyOpt("sourceTimeStamp", d.SourceTimeStamp))
	p.Add(foundation.NewDiagnosticsProperty("delta", d.Delta))
	p.Add(foundation.NewDoubleProperty("primaryDelta", d.PrimaryDelta))
	p.Add(foundation.NewEnumProperty("kind", d.Kind))
	p.Add(foundation.NewDiagnosticsProperty("offsetFromOrigin", d.OffsetFromOrigin))
	p.Add(foundation.NewDiagnosticsProperty("localOffsetFromOrigin", d.LocalOffsetFromOrigin))
	p.Add(foundation.NewIntProperty("consecutiveTapCount", foundation.Some(d.ConsecutiveTapCount)))
}

// GestureTapDragUpdateCallback is called when the pointer moves during a drag
// after a tap.
//
// Flutter: gestures/tap_and_drag.dart GestureTapDragUpdateCallback
type GestureTapDragUpdateCallback func(details TapDragUpdateDetails)

// TapDragEndDetails is the data of a drag after a tap that ended.
//
// Flutter: gestures/tap_and_drag.dart TapDragEndDetails
type TapDragEndDetails struct {
	// GlobalPosition is where the pointer was lifted.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// Velocity is the velocity of the pointer.
	Velocity Velocity
	// PrimaryVelocity is the velocity along the primary axis, if there is one.
	PrimaryVelocity foundation.Opt[float64]
	// ConsecutiveTapCount is the number of taps of the series.
	ConsecutiveTapCount int
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/tap_and_drag.dart TapDragEndDetails.globalPosition
func (d TapDragEndDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/tap_and_drag.dart TapDragEndDetails.localPosition
func (d TapDragEndDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/tap_and_drag.dart TapDragEndDetails.debugFillProperties
func (d TapDragEndDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewDiagnosticsProperty("velocity", d.Velocity))
	p.Add(foundation.NewDoubleProperty("primaryVelocity", d.PrimaryVelocity))
	p.Add(foundation.NewIntProperty("consecutiveTapCount", foundation.Some(d.ConsecutiveTapCount)))
}

// GestureTapDragEndCallback is called when a drag after a tap ends.
//
// Flutter: gestures/tap_and_drag.dart GestureTapDragEndCallback
type GestureTapDragEndCallback func(endDetails TapDragEndDetails)

// GestureCancelCallback is called when the gesture is cancelled.
//
// Flutter: gestures/tap_and_drag.dart GestureCancelCallback
type GestureCancelCallback func()

// tapDragAxis is what differs between the horizontal and the free drag.
type tapDragAxis interface {
	getDeltaForDetails(delta ui.Offset) ui.Offset
	getPrimaryValueFromOffset(value ui.Offset) foundation.Opt[float64]
	hasSufficientGlobalDistanceToAccept(r *BaseTapAndDragGestureRecognizer, kind ui.PointerDeviceKind) bool
	debugDescription() string
}

// TapAndDragGestureRecognizerOpts are the optional parameters of the tap and
// drag recognizers.
//
// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.new
type TapAndDragGestureRecognizerOpts struct {
	// DebugOwner is the object that owns the recognizer.
	DebugOwner any
	// SupportedDevices are the kinds of pointer the recognizer accepts; nil for
	// all.
	SupportedDevices []ui.PointerDeviceKind
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed.
	AllowedButtonsFilter AllowedButtonsFilter
	// NoEagerVictoryOnDrag makes the recognizer wait to win the arena by the
	// usual rules after a drag started, instead of claiming the win. Flutter's
	// eagerVictoryOnDrag is true by default.
	NoEagerVictoryOnDrag bool
}

// BaseTapAndDragGestureRecognizer recognizes a tap, or a drag that starts with a
// tap, and counts the consecutive taps. Use TapAndPanGestureRecognizer or
// TapAndHorizontalDragGestureRecognizer.
//
// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer
type BaseTapAndDragGestureRecognizer struct {
	OneSequenceGestureRecognizer
	axis tapDragAxis

	// DragStartBehavior is where the drag starts; the default is
	// DragStartBehaviorStart.
	//
	// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.dragStartBehavior
	DragStartBehavior DragStartBehavior
	// DragUpdateThrottleFrequency, if set, limits the drag updates to one in
	// that time.
	//
	// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.dragUpdateThrottleFrequency
	DragUpdateThrottleFrequency foundation.Opt[time.Duration]
	// MaxConsecutiveTap is the number of taps after which the series starts
	// over.
	//
	// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.maxConsecutiveTap
	MaxConsecutiveTap foundation.Opt[int]
	// EagerVictoryOnDrag makes the recognizer win the arena when a drag starts.
	//
	// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.eagerVictoryOnDrag
	EagerVictoryOnDrag bool
	// OnTapDown is called when a pointer that might tap or drag touches the
	// screen.
	//
	// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.onTapDown
	OnTapDown GestureTapDragDownCallback
	// OnTapUp is called when a tap ends.
	//
	// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.onTapUp
	OnTapUp GestureTapDragUpCallback
	// OnDragStart is called when a drag starts.
	//
	// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.onDragStart
	OnDragStart GestureTapDragStartCallback
	// OnDragUpdate is called when the pointer moves during the drag.
	//
	// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.onDragUpdate
	OnDragUpdate GestureTapDragUpdateCallback
	// OnDragEnd is called when the drag ends.
	//
	// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.onDragEnd
	OnDragEnd GestureTapDragEndCallback
	// OnCancel is called when the pointer that sent OnTapDown will not tap or
	// drag.
	//
	// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.onCancel
	OnCancel GestureCancelCallback
	// OnTapTrackStart, if set, is called when a tap starts to be tracked.
	//
	// Flutter: gestures/tap_and_drag.dart _TapStatusTrackerMixin.onTapTrackStart
	OnTapTrackStart func()
	// OnTapTrackReset, if set, is called when the tap tracking is reset.
	//
	// Flutter: gestures/tap_and_drag.dart _TapStatusTrackerMixin.onTapTrackReset
	OnTapTrackReset func()

	// State of the tap tracker.
	down                *PointerDownEvent
	up                  *PointerUpEvent
	consecutiveTapCount int
	originPosition      foundation.Opt[OffsetPair]
	previousButtons     foundation.Opt[int]
	consecutiveTapTimer *Timer
	lastTapOffset       foundation.Opt[ui.Offset]

	pastSlopTolerance           bool
	sentTapDown                 bool
	wonArenaForPrimaryPointer   bool
	primaryPointer              foundation.Opt[int]
	deadlineTimer               *Timer
	dragState                   dragState
	start                       PointerEvent
	initialPosition             OffsetPair
	currentPosition             OffsetPair
	globalDistanceMoved         float64
	globalDistanceMovedAllAxes  float64
	lastDragUpdateDetails       foundation.Opt[TapDragUpdateDetails]
	dragUpdateThrottleTimer     *Timer
	acceptedActivePointers      map[int]bool
	acceptedActivePointersOrder []int
}

func (r *BaseTapAndDragGestureRecognizer) initTapAndDrag(binding *GestureBinding, self OneSequenceHooks, axis tapDragAxis, opts TapAndDragGestureRecognizerOpts) {
	r.InitOneSequence(binding, self, GestureRecognizerOpts{
		DebugOwner: opts.DebugOwner, SupportedDevices: opts.SupportedDevices, AllowedButtonsFilter: opts.AllowedButtonsFilter,
	})
	r.axis = axis
	r.EagerVictoryOnDrag = !opts.NoEagerVictoryOnDrag
	r.DragStartBehavior = DragStartBehaviorStart
	r.acceptedActivePointers = map[int]bool{}
}

// CurrentDown is the down event of the tap being tracked.
//
// Flutter: gestures/tap_and_drag.dart _TapStatusTrackerMixin.currentDown
func (r *BaseTapAndDragGestureRecognizer) CurrentDown() *PointerDownEvent { return r.down }

// CurrentUp is the up event of the tap being tracked.
//
// Flutter: gestures/tap_and_drag.dart _TapStatusTrackerMixin.currentUp
func (r *BaseTapAndDragGestureRecognizer) CurrentUp() *PointerUpEvent { return r.up }

// ConsecutiveTapCount is the number of taps of the series so far.
//
// Flutter: gestures/tap_and_drag.dart _TapStatusTrackerMixin.consecutiveTapCount
func (r *BaseTapAndDragGestureRecognizer) ConsecutiveTapCount() int { return r.consecutiveTapCount }

func (r *BaseTapAndDragGestureRecognizer) handleDragUpdateThrottled() {
	if d, ok := r.lastDragUpdateDetails.Get(); ok && r.OnDragUpdate != nil {
		r.InvokeCallback("onDragUpdate", func() { r.OnDragUpdate(d) })
	}
	r.dragUpdateThrottleTimer = nil
	r.lastDragUpdateDetails = foundation.None[TapDragUpdateDetails]()
}

// IsPointerAllowed allows a primary-button pointer when a callback is set; once a
// pointer is primary, no other is allowed.
//
// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.isPointerAllowed
func (r *BaseTapAndDragGestureRecognizer) IsPointerAllowed(event *PointerDownEvent) bool {
	if primary, ok := r.primaryPointer.Get(); !ok {
		if event.Buttons() != PrimaryButton {
			return false
		}
		if r.OnTapDown == nil && r.OnDragStart == nil && r.OnDragUpdate == nil && r.OnDragEnd == nil && r.OnTapUp == nil && r.OnCancel == nil {
			return false
		}
	} else if event.Pointer() != primary {
		return false
	}
	return r.OneSequenceGestureRecognizer.IsPointerAllowed(event)
}

// AddAllowedPointer tracks the pointer as a tap of the series, and starts the
// deadline.
//
// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.addAllowedPointer
func (r *BaseTapAndDragGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	if r.dragState != dragReady {
		return
	}
	// The tap status tracker.
	r.OneSequenceGestureRecognizer.AddAllowedPointer(event)
	if r.consecutiveTapTimer != nil && !r.consecutiveTapTimer.IsActive() {
		r.tapTrackerReset()
	}
	if mct, ok := r.MaxConsecutiveTap.Get(); ok && mct == r.consecutiveTapCount {
		r.tapTrackerReset()
	}
	r.up = nil
	if r.down != nil && !r.representsSameSeries(event) {
		r.consecutiveTapCount = 1
	} else {
		r.consecutiveTapCount++
	}
	r.consecutiveTapTimerStop()
	r.trackTap(event)

	r.primaryPointer = foundation.Some(event.Pointer())
	r.globalDistanceMoved = 0
	r.globalDistanceMovedAllAxes = 0
	r.dragState = dragPossible
	r.initialPosition = OffsetPair{Global: event.Position(), Local: event.LocalPosition()}
	r.currentPosition = r.initialPosition
	r.deadlineTimer = r.binding.NewTimer(PressTimeout, r.didExceedDeadline)
}

// HandleNonAllowedPointer rejects the recognizer for a pointer that does not use
// the primary button, unless it already won.
//
// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.handleNonAllowedPointer
func (r *BaseTapAndDragGestureRecognizer) HandleNonAllowedPointer(event *PointerDownEvent) {
	if event.Buttons() != PrimaryButton && !r.wonArenaForPrimaryPointer {
		r.OneSequenceGestureRecognizer.HandleNonAllowedPointer(event)
	}
}

// AcceptGesture reports the tap down and the start of a drag that was waiting
// for the win.
//
// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.acceptGesture
func (r *BaseTapAndDragGestureRecognizer) AcceptGesture(pointer int) {
	if p, _ := r.primaryPointer.Get(); pointer != p {
		return
	}
	r.stopDeadlineTimer()
	if foundation.DebugMode && r.acceptedActivePointers[pointer] {
		panic(errors.New("TapDragPointerAcceptedTwice", "the pointer was accepted twice", "pointer", pointer))
	}
	r.acceptPointer(pointer)
	if r.down != nil {
		r.checkTapDown(r.down)
	}
	r.wonArenaForPrimaryPointer = true
	if r.start != nil && r.EagerVictoryOnDrag {
		r.acceptDrag(r.start)
	}
	if r.start != nil && !r.EagerVictoryOnDrag {
		r.dragState = dragAccepted
		r.acceptDrag(r.start)
	}
	if r.up != nil {
		r.checkTapUp(r.up)
	}
}

func (r *BaseTapAndDragGestureRecognizer) acceptPointer(pointer int) {
	if !r.acceptedActivePointers[pointer] {
		r.acceptedActivePointers[pointer] = true
		r.acceptedActivePointersOrder = append(r.acceptedActivePointersOrder, pointer)
	}
}

// removeAccepted removes the pointer from the accepted ones; it reports whether
// it was there.
func (r *BaseTapAndDragGestureRecognizer) removeAccepted(pointer int) bool {
	if !r.acceptedActivePointers[pointer] {
		return false
	}
	delete(r.acceptedActivePointers, pointer)
	for i, p := range r.acceptedActivePointersOrder {
		if p == pointer {
			r.acceptedActivePointersOrder = append(r.acceptedActivePointersOrder[:i], r.acceptedActivePointersOrder[i+1:]...)
			break
		}
	}
	return true
}

// DidStopTrackingLastPointer ends the tap or drag.
//
// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.didStopTrackingLastPointer
func (r *BaseTapAndDragGestureRecognizer) DidStopTrackingLastPointer(pointer int) {
	switch r.dragState {
	case dragReady:
		r.checkCancel()
		r.oneSeq.Resolve(GestureRejected)
	case dragPossible:
		if r.pastSlopTolerance {
			if r.wonArenaForPrimaryPointer {
				if r.down != nil {
					if !r.removeAccepted(pointer) {
						r.oneSeq.ResolvePointer(pointer, GestureRejected)
					}
					r.dragState = dragAccepted
					r.acceptDrag(r.down)
					r.checkDragEnd()
				}
			} else {
				r.checkCancel()
				r.oneSeq.Resolve(GestureRejected)
			}
		} else if r.up != nil {
			r.checkTapUp(r.up)
		}
	case dragAccepted:
		r.checkDragEnd()
	}
	r.stopDeadlineTimer()
	r.start = nil
	r.dragState = dragReady
	r.pastSlopTolerance = false
}

// HandleEvent follows the primary pointer.
//
// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.handleEvent
func (r *BaseTapAndDragGestureRecognizer) HandleEvent(event PointerEvent) {
	if p, _ := r.primaryPointer.Get(); event.Pointer() != p {
		return
	}
	// The tap status tracker.
	switch e := event.(type) {
	case *PointerMoveEvent:
		computedSlop := ComputeHitSlop(e.Kind(), r.GestureSettings)
		if getGlobalDistance(e, r.originPosition.Value()) > computedSlop {
			r.consecutiveTapTimerStop()
			r.previousButtons = foundation.None[int]()
			r.lastTapOffset = foundation.None[ui.Offset]()
		}
	case *PointerUpEvent:
		r.up = e
		if r.down != nil {
			r.consecutiveTapTimerStop()
			r.consecutiveTapTimerStart()
		}
	case *PointerCancelEvent:
		r.tapTrackerReset()
	}

	switch e := event.(type) {
	case *PointerMoveEvent:
		// Drag-related state is only used when the pointer moves past the slop
		// tolerance.
		computedSlop := ComputeHitSlop(e.Kind(), r.GestureSettings)
		r.pastSlopTolerance = r.pastSlopTolerance || getGlobalDistance(e, r.initialPosition) > computedSlop
		switch r.dragState {
		case dragAccepted:
			r.currentPosition = NewOffsetPairFromEventPosition(e)
			r.checkDragUpdate(e, foundation.None[OffsetPair]())
		case dragPossible:
			if r.start == nil {
				// Only check for a drag if the start of a drag was not already
				// identified.
				r.checkDrag(e)
			}
			// This can occur when the gesture has been accepted: the arena
			// was won before the drag was identified.
			if r.start != nil && r.wonArenaForPrimaryPointer {
				r.dragState = dragAccepted
				r.acceptDrag(r.start)
			}
		}
	case *PointerUpEvent:
		switch r.dragState {
		case dragPossible:
			// The drag has not been accepted before the pointer is lifted: it is
			// a tap.
			r.StopTrackingIfPointerNoLongerDown(e)
		case dragAccepted:
			// For the case when the drag has been accepted, the up event ends the
			// drag.
			r.giveUpPointer(e.Pointer())
		}
	case *PointerCancelEvent:
		r.dragState = dragReady
		r.giveUpPointer(e.Pointer())
	}
}

// RejectGesture gives up the pointer and resets the state.
//
// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.rejectGesture
func (r *BaseTapAndDragGestureRecognizer) RejectGesture(pointer int) {
	if p, _ := r.primaryPointer.Get(); pointer != p {
		return
	}
	r.tapTrackerReset() // the tap status tracker
	r.stopDeadlineTimer()
	r.giveUpPointer(pointer)
	r.resetTaps()
	r.resetDragUpdateThrottle()
}

// Dispose stops the timers.
//
// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.dispose
func (r *BaseTapAndDragGestureRecognizer) Dispose() {
	r.stopDeadlineTimer()
	r.resetDragUpdateThrottle()
	r.tapTrackerReset() // the tap status tracker
	r.OneSequenceGestureRecognizer.Dispose()
}

// DebugDescription names the recognizer.
//
// Flutter: gestures/tap_and_drag.dart BaseTapAndDragGestureRecognizer.debugDescription
func (r *BaseTapAndDragGestureRecognizer) DebugDescription() string { return r.axis.debugDescription() }

func (r *BaseTapAndDragGestureRecognizer) acceptDrag(event PointerEvent) {
	if !r.wonArenaForPrimaryPointer {
		return
	}
	if r.DragStartBehavior == DragStartBehaviorStart {
		r.initialPosition = r.initialPosition.Add(NewOffsetPairFromEventDelta(event))
		r.currentPosition = r.initialPosition
	}
	r.checkDragStart(event)
	if localDelta := event.LocalDelta(); localDelta != (ui.Offset{}) {
		r.currentPosition = NewOffsetPairFromEventPosition(event)
		correctedLocalPosition := r.initialPosition.Local.Add(localDelta)
		var localToGlobal *ui.Matrix4
		if t := event.Transform(); t != nil {
			localToGlobal = tryInvert(t)
		}
		globalUpdateDelta := TransformDeltaViaPositions(correctedLocalPosition, localDelta, localToGlobal)
		updateDelta := OffsetPair{Local: localDelta, Global: globalUpdateDelta}
		r.checkDragUpdate(event, foundation.Some(r.initialPosition.Add(updateDelta)))
	}
}

func (r *BaseTapAndDragGestureRecognizer) checkDrag(event *PointerMoveEvent) {
	var localToGlobal *ui.Matrix4
	if t := event.Transform(); t != nil {
		localToGlobal = tryInvert(t)
	}
	movedLocally := r.axis.getDeltaForDetails(event.LocalDelta())
	sign := 1.0
	if v, ok := r.axis.getPrimaryValueFromOffset(movedLocally).Get(); ok {
		sign = dartSign(v)
	}
	r.globalDistanceMoved += TransformDeltaViaPositions(event.LocalPosition(), movedLocally, localToGlobal).Distance() * sign
	r.globalDistanceMovedAllAxes += TransformDeltaViaPositions(event.LocalPosition(), event.LocalDelta(), localToGlobal).Distance()
	if r.axis.hasSufficientGlobalDistanceToAccept(r, event.Kind()) ||
		(r.wonArenaForPrimaryPointer && math.Abs(r.globalDistanceMovedAllAxes) > ComputePanSlop(event.Kind(), r.GestureSettings)) {
		r.start = event
		if r.EagerVictoryOnDrag {
			r.dragState = dragAccepted
			if !r.wonArenaForPrimaryPointer {
				r.oneSeq.Resolve(GestureAccepted)
			}
		}
	}
}

func (r *BaseTapAndDragGestureRecognizer) checkTapDown(event *PointerDownEvent) {
	if r.sentTapDown {
		return
	}
	details := TapDragDownDetails{
		GlobalPosition: event.Position(), LocalPosition: event.LocalPosition(),
		Kind: foundation.Some(r.GetKindForPointer(event.Pointer())), ConsecutiveTapCount: r.consecutiveTapCount,
	}
	if r.OnTapDown != nil {
		r.InvokeCallback("onTapDown", func() { r.OnTapDown(details) })
	}
	r.sentTapDown = true
}

func (r *BaseTapAndDragGestureRecognizer) checkTapUp(event *PointerUpEvent) {
	if !r.wonArenaForPrimaryPointer {
		return
	}
	details := TapDragUpDetails{
		Kind: event.Kind(), GlobalPosition: event.Position(), LocalPosition: event.LocalPosition(),
		ConsecutiveTapCount: r.consecutiveTapCount,
	}
	if r.OnTapUp != nil {
		r.InvokeCallback("onTapUp", func() { r.OnTapUp(details) })
	}
	r.resetTaps()
	if !r.removeAccepted(event.Pointer()) {
		r.oneSeq.ResolvePointer(event.Pointer(), GestureRejected)
	}
}

func (r *BaseTapAndDragGestureRecognizer) checkDragStart(event PointerEvent) {
	if r.OnDragStart != nil {
		details := TapDragStartDetails{
			SourceTimeStamp: foundation.Some(event.TimeStamp()), GlobalPosition: r.initialPosition.Global,
			LocalPosition: r.initialPosition.Local, Kind: foundation.Some(r.GetKindForPointer(event.Pointer())),
			ConsecutiveTapCount: r.consecutiveTapCount,
		}
		r.InvokeCallback("onDragStart", func() { r.OnDragStart(details) })
	}
	r.start = nil
}

func (r *BaseTapAndDragGestureRecognizer) checkDragUpdate(event PointerEvent, corrected foundation.Opt[OffsetPair]) {
	globalPosition, localPosition := event.Position(), event.LocalPosition()
	if c, ok := corrected.Get(); ok {
		globalPosition, localPosition = c.Global, c.Local
	}
	details := TapDragUpdateDetails{
		SourceTimeStamp: foundation.Some(event.TimeStamp()), Delta: event.LocalDelta(),
		GlobalPosition: globalPosition, Kind: foundation.Some(r.GetKindForPointer(event.Pointer())),
		LocalPosition:    localPosition,
		OffsetFromOrigin: globalPosition.Sub(r.initialPosition.Global), LocalOffsetFromOrigin: localPosition.Sub(r.initialPosition.Local),
		ConsecutiveTapCount: r.consecutiveTapCount,
	}
	if freq, ok := r.DragUpdateThrottleFrequency.Get(); ok {
		// If the drag updates are throttled, remember the last details and
		// report them when the timer fires.
		r.lastDragUpdateDetails = foundation.Some(details)
		if r.dragUpdateThrottleTimer == nil {
			r.dragUpdateThrottleTimer = r.binding.NewTimer(freq, r.handleDragUpdateThrottled)
		}
	} else if r.OnDragUpdate != nil {
		r.InvokeCallback("onDragUpdate", func() { r.OnDragUpdate(details) })
	}
}

func (r *BaseTapAndDragGestureRecognizer) checkDragEnd() {
	if r.dragUpdateThrottleTimer != nil {
		// If there's already an update scheduled, trigger it immediately and
		// cancel the timer.
		r.dragUpdateThrottleTimer.Cancel()
		r.handleDragUpdateThrottled()
	}
	endDetails := TapDragEndDetails{
		GlobalPosition: r.currentPosition.Global, LocalPosition: r.currentPosition.Local,
		PrimaryVelocity: foundation.Some(0.0), ConsecutiveTapCount: r.consecutiveTapCount,
	}
	if r.OnDragEnd != nil {
		r.InvokeCallback("onDragEnd", func() { r.OnDragEnd(endDetails) })
	}
	r.resetTaps()
	r.resetDragUpdateThrottle()
}

func (r *BaseTapAndDragGestureRecognizer) checkCancel() {
	if !r.sentTapDown {
		// Do not fire cancel if the recognizer never sent a tap down.
		return
	}
	if r.OnCancel != nil {
		r.InvokeCallback("onCancel", r.OnCancel)
	}
	r.resetDragUpdateThrottle()
	r.resetTaps()
}

func (r *BaseTapAndDragGestureRecognizer) didExceedDeadline() {
	if r.down != nil {
		r.checkTapDown(r.down)
		if r.consecutiveTapCount > 1 {
			// If our consecutive tap count is greater than 1, i.e. is a double
			// tap or greater, then this recognizer declares victory to prevent
			// the gesture arena from being swept.
			r.oneSeq.Resolve(GestureAccepted)
		}
	}
}

func (r *BaseTapAndDragGestureRecognizer) giveUpPointer(pointer int) {
	r.StopTrackingPointer(pointer)
	if !r.removeAccepted(pointer) {
		r.oneSeq.ResolvePointer(pointer, GestureRejected)
	}
}

func (r *BaseTapAndDragGestureRecognizer) resetTaps() {
	r.sentTapDown = false
	r.wonArenaForPrimaryPointer = false
	r.primaryPointer = foundation.None[int]()
}

func (r *BaseTapAndDragGestureRecognizer) resetDragUpdateThrottle() {
	if !r.DragUpdateThrottleFrequency.IsSome() {
		return
	}
	r.lastDragUpdateDetails = foundation.None[TapDragUpdateDetails]()
	if r.dragUpdateThrottleTimer != nil {
		r.dragUpdateThrottleTimer.Cancel()
		r.dragUpdateThrottleTimer = nil
	}
}

func (r *BaseTapAndDragGestureRecognizer) stopDeadlineTimer() {
	if r.deadlineTimer != nil {
		r.deadlineTimer.Cancel()
		r.deadlineTimer = nil
	}
}

// The tap status tracker.

func (r *BaseTapAndDragGestureRecognizer) trackTap(event *PointerDownEvent) {
	r.down = event
	r.previousButtons = foundation.Some(event.Buttons())
	r.lastTapOffset = foundation.Some(event.Position())
	r.originPosition = foundation.Some(OffsetPair{Local: event.LocalPosition(), Global: event.Position()})
	if r.OnTapTrackStart != nil {
		r.OnTapTrackStart()
	}
}

func (r *BaseTapAndDragGestureRecognizer) hasSameButton(buttons int) bool {
	return buttons == r.previousButtons.Value()
}

func (r *BaseTapAndDragGestureRecognizer) isWithinConsecutiveTapTolerance(secondTapOffset ui.Offset) bool {
	last, ok := r.lastTapOffset.Get()
	if !ok {
		return false
	}
	return secondTapOffset.Sub(last).Distance() <= DoubleTapSlop
}

func (r *BaseTapAndDragGestureRecognizer) representsSameSeries(event *PointerDownEvent) bool {
	return r.consecutiveTapTimer != nil && r.isWithinConsecutiveTapTolerance(event.Position()) && r.hasSameButton(event.Buttons())
}

func (r *BaseTapAndDragGestureRecognizer) consecutiveTapTimerStart() {
	if r.consecutiveTapTimer == nil {
		r.consecutiveTapTimer = r.binding.NewTimer(DoubleTapTimeout, func() {})
	}
}

func (r *BaseTapAndDragGestureRecognizer) consecutiveTapTimerStop() {
	if r.consecutiveTapTimer != nil {
		r.consecutiveTapTimer.Cancel()
		r.consecutiveTapTimer = nil
	}
}

func (r *BaseTapAndDragGestureRecognizer) tapTrackerReset() {
	r.consecutiveTapTimerStop()
	r.previousButtons = foundation.None[int]()
	r.originPosition = foundation.None[OffsetPair]()
	r.lastTapOffset = foundation.None[ui.Offset]()
	r.consecutiveTapCount = 0
	r.down = nil
	r.up = nil
	if r.OnTapTrackReset != nil {
		r.OnTapTrackReset()
	}
}

type tapAndHorizontalDragAxis struct{}

func (tapAndHorizontalDragAxis) getDeltaForDetails(d ui.Offset) ui.Offset { return ui.Offset{Dx: d.Dx} }
func (tapAndHorizontalDragAxis) getPrimaryValueFromOffset(v ui.Offset) foundation.Opt[float64] {
	return foundation.Some(v.Dx)
}
func (tapAndHorizontalDragAxis) hasSufficientGlobalDistanceToAccept(r *BaseTapAndDragGestureRecognizer, kind ui.PointerDeviceKind) bool {
	return math.Abs(r.globalDistanceMoved) > ComputeHitSlop(kind, r.GestureSettings)
}
func (tapAndHorizontalDragAxis) debugDescription() string { return "tap and horizontal drag" }

type tapAndPanAxis struct{}

func (tapAndPanAxis) getDeltaForDetails(d ui.Offset) ui.Offset { return d }
func (tapAndPanAxis) getPrimaryValueFromOffset(ui.Offset) foundation.Opt[float64] {
	return foundation.None[float64]()
}
func (tapAndPanAxis) hasSufficientGlobalDistanceToAccept(r *BaseTapAndDragGestureRecognizer, kind ui.PointerDeviceKind) bool {
	return math.Abs(r.globalDistanceMoved) > ComputePanSlop(kind, r.GestureSettings)
}
func (tapAndPanAxis) debugDescription() string { return "tap and pan" }

// TapAndHorizontalDragGestureRecognizer recognizes a tap, or a horizontal drag
// that starts with a tap.
//
// Flutter: gestures/tap_and_drag.dart TapAndHorizontalDragGestureRecognizer
type TapAndHorizontalDragGestureRecognizer struct {
	BaseTapAndDragGestureRecognizer
}

// NewTapAndHorizontalDragGestureRecognizer creates a recognizer that works with
// the binding.
//
// Flutter: gestures/tap_and_drag.dart TapAndHorizontalDragGestureRecognizer.new
func NewTapAndHorizontalDragGestureRecognizer(binding *GestureBinding, opts ...TapAndDragGestureRecognizerOpts) *TapAndHorizontalDragGestureRecognizer {
	r := &TapAndHorizontalDragGestureRecognizer{}
	r.initTapAndDrag(binding, r, tapAndHorizontalDragAxis{}, oneOpts(opts))
	return r
}

// TapAndPanGestureRecognizer recognizes a tap, or a drag in any direction that
// starts with a tap.
//
// Flutter: gestures/tap_and_drag.dart TapAndPanGestureRecognizer
type TapAndPanGestureRecognizer struct {
	BaseTapAndDragGestureRecognizer
}

// NewTapAndPanGestureRecognizer creates a recognizer that works with the
// binding.
//
// Flutter: gestures/tap_and_drag.dart TapAndPanGestureRecognizer.new
func NewTapAndPanGestureRecognizer(binding *GestureBinding, opts ...TapAndDragGestureRecognizerOpts) *TapAndPanGestureRecognizer {
	r := &TapAndPanGestureRecognizer{}
	r.initTapAndDrag(binding, r, tapAndPanAxis{}, oneOpts(opts))
	return r
}
