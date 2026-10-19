package gestures

import (
	"math"
	"slices"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

type dragState int

const (
	dragReady dragState = iota
	dragPossible
	dragAccepted
)

type dragDirection int

const (
	dragDirectionNone dragDirection = iota
	dragDirectionHorizontal
	dragDirectionVertical
)

// tryInvert returns the inverse of m, or nil if it has none.
func tryInvert(m *ui.Matrix4) *ui.Matrix4 {
	c := m.Clone()
	if c.Invert() == 0 {
		return nil
	}
	return c
}

// GestureVelocityTrackerBuilder makes the velocity tracker for a pointer.
//
// Flutter: gestures/monodrag.dart GestureVelocityTrackerBuilder
type GestureVelocityTrackerBuilder func(event PointerEvent) *VelocityTracker

// dragAxis is what differs between the vertical, horizontal and free drag.
type dragAxis interface {
	isFlingGesture(r *DragGestureRecognizer, estimate VelocityEstimate, kind ui.PointerDeviceKind) bool
	considerFling(r *DragGestureRecognizer, estimate VelocityEstimate, kind ui.PointerDeviceKind) foundation.Opt[DragEndDetails]
	hasSufficientGlobalDistanceToAccept(r *DragGestureRecognizer, kind ui.PointerDeviceKind) bool
	getDeltaForDetails(delta ui.Offset) ui.Offset
	getPrimaryValueFromOffset(value ui.Offset) foundation.Opt[float64]
	primaryDragAxis() dragDirection
	debugDescription() string
}

// DragGestureRecognizerOpts are the optional parameters of the drag
// recognizers.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.new
type DragGestureRecognizerOpts struct {
	// DebugOwner is the object that owns the recognizer.
	DebugOwner any
	// SupportedDevices are the kinds of pointer the recognizer accepts; nil for
	// all.
	SupportedDevices []ui.PointerDeviceKind
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed;
	// the default allows the primary button only.
	AllowedButtonsFilter AllowedButtonsFilter
	// DragStartBehavior is where the drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior DragStartBehavior
	// MultitouchDragStrategy is how the drag follows several fingers; the
	// default is MultitouchDragStrategyLatestPointer.
	MultitouchDragStrategy MultitouchDragStrategy
	// VelocityTrackerBuilder makes the velocity tracker of a pointer; the
	// default is NewVelocityTrackerWithKind of the binding's clock.
	VelocityTrackerBuilder GestureVelocityTrackerBuilder
	// OnlyAcceptDragOnThreshold makes the recognizer report a drag only after
	// the pointer has moved past the slop.
	OnlyAcceptDragOnThreshold bool
}

// DragGestureRecognizer is the base of the recognizers of drags: it recognizes
// the pointer moving past the slop with its button down. Use
// VerticalDragGestureRecognizer, HorizontalDragGestureRecognizer or
// PanGestureRecognizer.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer
type DragGestureRecognizer struct {
	OneSequenceGestureRecognizer
	axis dragAxis

	// DragStartBehavior is where the drag starts.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.dragStartBehavior
	DragStartBehavior DragStartBehavior
	// MultitouchDragStrategy is how the drag follows several fingers.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.multitouchDragStrategy
	MultitouchDragStrategy MultitouchDragStrategy
	// OnDown is called when a pointer that might drag touches the screen.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.onDown
	OnDown GestureDragDownCallback
	// OnStart is called when the drag starts.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.onStart
	OnStart GestureDragStartCallback
	// OnUpdate is called when the pointer moves during the drag.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.onUpdate
	OnUpdate GestureDragUpdateCallback
	// OnEnd is called when the pointer that dragged is lifted.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.onEnd
	OnEnd GestureDragEndCallback
	// OnCancel is called when the pointer that sent OnDown will not drag.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.onCancel
	OnCancel GestureDragCancelCallback
	// MinFlingDistance is the shortest distance of a fling; the default is the
	// hit slop of the device.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.minFlingDistance
	MinFlingDistance foundation.Opt[float64]
	// MinFlingVelocity is the slowest fling; the default is MinFlingVelocity.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.minFlingVelocity
	MinFlingVelocity foundation.Opt[float64]
	// MaxFlingVelocity is the fastest fling; the default is MaxFlingVelocity.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.maxFlingVelocity
	MaxFlingVelocity foundation.Opt[float64]
	// OnlyAcceptDragOnThreshold makes the recognizer report a drag only after
	// the pointer has moved past the slop.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.onlyAcceptDragOnThreshold
	OnlyAcceptDragOnThreshold bool
	// VelocityTrackerBuilder makes the velocity tracker of a pointer.
	//
	// Flutter: gestures/monodrag.dart DragGestureRecognizer.velocityTrackerBuilder
	VelocityTrackerBuilder GestureVelocityTrackerBuilder

	state                    dragState
	initialPosition          OffsetPair
	pendingDragOffset        OffsetPair
	lastPosition             OffsetPair
	lastPendingEventTimeStmp foundation.Opt[time.Duration]
	initialButtons           foundation.Opt[int]
	lastTransform            *ui.Matrix4
	globalDistanceMoved      float64
	hasDragThresholdBeenMet  bool
	velocityTrackers         map[int]*VelocityTracker
	// moveDeltaBeforeFrame is the sum of the local deltas of each pointer in the
	// current frame, in the order the pointers moved first.
	moveDeltaBeforeFrame   map[int]ui.Offset
	moveDeltaOrder         []int
	frameTimeStamp         foundation.Opt[time.Duration]
	lastUpdatedDeltaForPan ui.Offset
	acceptedActivePointers []int
	activePointer          foundation.Opt[int]
}

func (r *DragGestureRecognizer) initDrag(binding *GestureBinding, self OneSequenceHooks, axis dragAxis, opts DragGestureRecognizerOpts) {
	filter := opts.AllowedButtonsFilter
	if filter == nil {
		filter = func(buttons int) bool { return buttons == PrimaryButton }
	}
	r.InitOneSequence(binding, self, GestureRecognizerOpts{
		DebugOwner: opts.DebugOwner, SupportedDevices: opts.SupportedDevices, AllowedButtonsFilter: filter,
	})
	r.axis = axis
	r.DragStartBehavior = opts.DragStartBehavior.orDefault()
	r.MultitouchDragStrategy = opts.MultitouchDragStrategy
	if r.MultitouchDragStrategy == MultitouchDragStrategyUnspecified {
		r.MultitouchDragStrategy = MultitouchDragStrategyLatestPointer
	}
	r.OnlyAcceptDragOnThreshold = opts.OnlyAcceptDragOnThreshold
	r.VelocityTrackerBuilder = opts.VelocityTrackerBuilder
	if r.VelocityTrackerBuilder == nil {
		r.VelocityTrackerBuilder = func(e PointerEvent) *VelocityTracker {
			return NewVelocityTrackerWithKind(binding.Clock(), e.Kind())
		}
	}
	r.velocityTrackers = map[int]*VelocityTracker{}
	r.moveDeltaBeforeFrame = map[int]ui.Offset{}
}

// LastPosition is the position of the pointer at its last move.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.lastPosition
func (r *DragGestureRecognizer) LastPosition() OffsetPair { return r.lastPosition }

// GlobalDistanceMoved is how far the pointer moved along the axis of the
// drag, in global coordinates, with the sign of the movement.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.globalDistanceMoved
func (r *DragGestureRecognizer) GlobalDistanceMoved() float64 { return r.globalDistanceMoved }

// DebugLastPendingEventTimestamp is the time of the last event before the drag
// was accepted. It is meant for tests.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.debugLastPendingEventTimestamp
func (r *DragGestureRecognizer) DebugLastPendingEventTimestamp() foundation.Opt[time.Duration] {
	if !foundation.DebugMode {
		return foundation.None[time.Duration]()
	}
	return r.lastPendingEventTimeStmp
}

// IsFlingGesture reports whether the velocity estimate is a fling.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.isFlingGesture
func (r *DragGestureRecognizer) IsFlingGesture(estimate VelocityEstimate, kind ui.PointerDeviceKind) bool {
	return r.axis.isFlingGesture(r, estimate, kind)
}

// ConsiderFling returns the details of a fling, or None if the estimate is not
// one.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.considerFling
func (r *DragGestureRecognizer) ConsiderFling(estimate VelocityEstimate, kind ui.PointerDeviceKind) foundation.Opt[DragEndDetails] {
	return r.axis.considerFling(r, estimate, kind)
}

// IsPointerAllowed allows a pointer if a callback is set and, once a drag is
// possible, its buttons are those that started it.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.isPointerAllowed
func (r *DragGestureRecognizer) IsPointerAllowed(event *PointerDownEvent) bool {
	if ib, ok := r.initialButtons.Get(); !ok {
		if r.OnDown == nil && r.OnStart == nil && r.OnUpdate == nil && r.OnEnd == nil && r.OnCancel == nil {
			return false
		}
	} else if event.Buttons() != ib {
		return false
	}
	return r.OneSequenceGestureRecognizer.IsPointerAllowed(event)
}

func (r *DragGestureRecognizer) addPointer(event PointerEvent) {
	r.velocityTrackers[event.Pointer()] = r.VelocityTrackerBuilder(event)
	switch r.state {
	case dragReady:
		r.state = dragPossible
		r.initialPosition = OffsetPair{Global: event.Position(), Local: event.LocalPosition()}
		r.lastPosition = r.initialPosition
		r.pendingDragOffset = OffsetPair{}
		r.globalDistanceMoved = 0
		r.lastPendingEventTimeStmp = foundation.Some(event.TimeStamp())
		r.lastTransform = event.Transform()
		r.checkDown()
	case dragPossible:
	case dragAccepted:
		r.oneSeq.Resolve(GestureAccepted)
	}
}

// AddAllowedPointer tracks the pointer.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.addAllowedPointer
func (r *DragGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	r.OneSequenceGestureRecognizer.AddAllowedPointer(event)
	if r.state == dragReady {
		r.initialButtons = foundation.Some(event.Buttons())
	}
	r.addPointer(event)
}

// AddAllowedPointerPanZoom tracks the trackpad gesture as a pointer with the
// primary button.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.addAllowedPointerPanZoom
func (r *DragGestureRecognizer) AddAllowedPointerPanZoom(event *PointerPanZoomStartEvent) {
	r.OneSequenceGestureRecognizer.AddAllowedPointerPanZoom(event)
	r.oneSeq.StartTrackingPointer(event.Pointer(), event.Transform())
	if r.state == dragReady {
		r.initialButtons = foundation.Some(PrimaryButton)
	}
	r.addPointer(event)
}

func (r *DragGestureRecognizer) shouldTrackMoveEvent(pointer int) bool {
	switch r.MultitouchDragStrategy {
	case MultitouchDragStrategySumAllPointers, MultitouchDragStrategyAverageBoundaryPointers:
		return true
	}
	ap, ok := r.activePointer.Get()
	return !ok || pointer == ap
}

func (r *DragGestureRecognizer) recordMoveDeltaForMultitouch(pointer int, localDelta ui.Offset) {
	if r.MultitouchDragStrategy != MultitouchDragStrategyAverageBoundaryPointers {
		return
	}
	if r.state != dragAccepted || localDelta == (ui.Offset{}) {
		return
	}
	if offset, ok := r.moveDeltaBeforeFrame[pointer]; ok {
		r.moveDeltaBeforeFrame[pointer] = offset.Add(localDelta)
	} else {
		r.moveDeltaBeforeFrame[pointer] = localDelta
		r.moveDeltaOrder = append(r.moveDeltaOrder, pointer)
	}
}

func (r *DragGestureRecognizer) clearMoveDeltas() {
	r.moveDeltaBeforeFrame = map[int]ui.Offset{}
	r.moveDeltaOrder = nil
}

func (r *DragGestureRecognizer) getSumDelta(pointer int, positive bool, axis dragDirection) float64 {
	offset, ok := r.moveDeltaBeforeFrame[pointer]
	if !ok {
		return 0
	}
	v := offset.Dx
	if axis == dragDirectionVertical {
		v = offset.Dy
	}
	if positive {
		return math.Max(v, 0)
	}
	return math.Min(v, 0)
}

func (r *DragGestureRecognizer) getMaxSumDeltaPointer(positive bool, axis dragDirection) foundation.Opt[int] {
	if len(r.moveDeltaOrder) == 0 {
		return foundation.None[int]()
	}
	ret := r.moveDeltaOrder[0]
	max := r.getSumDelta(ret, positive, axis)
	for _, pointer := range r.moveDeltaOrder[1:] {
		sum := r.getSumDelta(pointer, positive, axis)
		if (positive && sum > max) || (!positive && sum < max) {
			ret, max = pointer, sum
		}
	}
	return foundation.Some(ret)
}

func (r *DragGestureRecognizer) resolveLocalDeltaForMultitouch(pointer int, localDelta ui.Offset) ui.Offset {
	if r.MultitouchDragStrategy != MultitouchDragStrategyAverageBoundaryPointers {
		if r.frameTimeStamp.IsSome() {
			r.clearMoveDeltas()
			r.frameTimeStamp = foundation.None[time.Duration]()
			r.lastUpdatedDeltaForPan = ui.Offset{}
		}
		return localDelta
	}
	current := r.binding.Scheduler().CurrentSystemFrameTimeStamp()
	if ft, ok := r.frameTimeStamp.Get(); !ok || ft != current {
		r.clearMoveDeltas()
		r.lastUpdatedDeltaForPan = ui.Offset{}
		r.frameTimeStamp = foundation.Some(current)
	}
	axis := r.axis.primaryDragAxis()
	if r.state != dragAccepted || localDelta == (ui.Offset{}) || (len(r.moveDeltaOrder) == 0 && axis != dragDirectionNone) {
		return localDelta
	}
	var dx, dy float64
	switch axis {
	case dragDirectionHorizontal:
		dx = r.resolveDelta(pointer, dragDirectionHorizontal, localDelta)
	case dragDirectionVertical:
		dy = r.resolveDelta(pointer, dragDirectionVertical, localDelta)
	default:
		averageX := r.resolveDeltaForPanGesture(dragDirectionHorizontal, localDelta)
		averageY := r.resolveDeltaForPanGesture(dragDirectionVertical, localDelta)
		updated := ui.Offset{Dx: averageX, Dy: averageY}.Sub(r.lastUpdatedDeltaForPan)
		r.lastUpdatedDeltaForPan = ui.Offset{Dx: averageX, Dy: averageY}
		dx, dy = updated.Dx, updated.Dy
	}
	return ui.Offset{Dx: dx, Dy: dy}
}

func (r *DragGestureRecognizer) resolveDelta(pointer int, axis dragDirection, localDelta ui.Offset) float64 {
	delta := localDelta.Dx
	if axis == dragDirectionVertical {
		delta = localDelta.Dy
	}
	positive := delta > 0
	maxPtr, ok := r.getMaxSumDeltaPointer(positive, axis).Get()
	if foundation.DebugMode && !ok {
		panic(errors.New("DragNoMaxDeltaPointer", "there is no pointer with a delta in this frame"))
	}
	if maxPtr == pointer {
		return delta
	}
	maxSumDelta := r.getSumDelta(maxPtr, positive, axis)
	cur := r.getSumDelta(pointer, positive, axis)
	if positive {
		if cur+delta > maxSumDelta {
			return cur + delta - maxSumDelta
		}
		return 0
	}
	if cur+delta < maxSumDelta {
		return cur + delta - maxSumDelta
	}
	return 0
}

func (r *DragGestureRecognizer) resolveDeltaForPanGesture(axis dragDirection, localDelta ui.Offset) float64 {
	delta := localDelta.Dx
	if axis == dragDirectionVertical {
		delta = localDelta.Dy
	}
	pointerCount := len(r.acceptedActivePointers)
	sum := delta
	for _, p := range r.moveDeltaOrder {
		if axis == dragDirectionHorizontal {
			sum += r.moveDeltaBeforeFrame[p].Dx
		} else {
			sum += r.moveDeltaBeforeFrame[p].Dy
		}
	}
	return sum / float64(pointerCount)
}

// HandleEvent follows the pointer.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.handleEvent
func (r *DragGestureRecognizer) HandleEvent(event PointerEvent) {
	if foundation.DebugMode && r.state == dragReady {
		panic(errors.New("DragEventWhileReady", "an event arrived while the drag recognizer is ready"))
	}
	if !event.Synthesized() {
		switch e := event.(type) {
		case *PointerDownEvent, *PointerMoveEvent, *PointerPanZoomStartEvent, *PointerPanZoomUpdateEvent:
			position := event.LocalPosition()
			switch pe := e.(type) {
			case *PointerPanZoomStartEvent:
				position = ui.Offset{}
			case *PointerPanZoomUpdateEvent:
				position = pe.Pan()
			}
			r.velocityTrackers[event.Pointer()].AddPosition(event.TimeStamp(), position)
		}
	}
	if move, ok := event.(*PointerMoveEvent); ok && (!r.initialButtons.IsSome() || move.Buttons() != r.initialButtons.Value()) {
		r.giveUpPointer(event.Pointer())
		return
	}
	var panZoom *PointerPanZoomUpdateEvent
	move, isMove := event.(*PointerMoveEvent)
	panZoom, isPanZoom := event.(*PointerPanZoomUpdateEvent)
	if (isMove || isPanZoom) && r.shouldTrackMoveEvent(event.Pointer()) {
		var delta, localDelta, position, localPosition ui.Offset
		if isMove {
			delta, localDelta, position, localPosition = move.Delta(), move.LocalDelta(), move.Position(), move.LocalPosition()
		} else {
			delta, localDelta = panZoom.PanDelta(), panZoom.LocalPanDelta()
			position, localPosition = panZoom.Position().Add(panZoom.Pan()), panZoom.LocalPosition().Add(panZoom.LocalPan())
		}
		r.lastPosition = OffsetPair{Local: localPosition, Global: position}
		resolvedDelta := r.resolveLocalDeltaForMultitouch(event.Pointer(), localDelta)
		switch r.state {
		case dragReady, dragPossible:
			r.pendingDragOffset = r.pendingDragOffset.Add(OffsetPair{Local: localDelta, Global: delta})
			r.lastPendingEventTimeStmp = foundation.Some(event.TimeStamp())
			r.lastTransform = event.Transform()
			movedLocally := r.axis.getDeltaForDetails(localDelta)
			var localToGlobal *ui.Matrix4
			if t := event.Transform(); t != nil {
				localToGlobal = tryInvert(t)
			}
			sign := 1.0
			if v, ok := r.axis.getPrimaryValueFromOffset(movedLocally).Get(); ok {
				sign = dartSign(v)
			}
			r.globalDistanceMoved += TransformDeltaViaPositions(localPosition, movedLocally, localToGlobal).Distance() * sign
			if r.axis.hasSufficientGlobalDistanceToAccept(r, event.Kind()) {
				r.hasDragThresholdBeenMet = true
				if slices.Contains(r.acceptedActivePointers, event.Pointer()) {
					r.checkDrag(event.Pointer())
				} else {
					r.oneSeq.Resolve(GestureAccepted)
				}
			}
		case dragAccepted:
			r.checkUpdate(foundation.Some(event.TimeStamp()), r.axis.getDeltaForDetails(resolvedDelta),
				r.axis.getPrimaryValueFromOffset(resolvedDelta), position, foundation.Some(localPosition), event.Pointer())
		}
		r.recordMoveDeltaForMultitouch(event.Pointer(), localDelta)
	}
	switch event.(type) {
	case *PointerUpEvent, *PointerCancelEvent, *PointerPanZoomEndEvent:
		r.giveUpPointer(event.Pointer())
	}
}

// dartSign is num.sign: -1, 0 or 1 (NaN stays NaN).
func dartSign(v float64) float64 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return v
}

// AcceptGesture starts the drag.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.acceptGesture
func (r *DragGestureRecognizer) AcceptGesture(pointer int) {
	if foundation.DebugMode && slices.Contains(r.acceptedActivePointers, pointer) {
		panic(errors.New("DragPointerAcceptedTwice", "the pointer was accepted twice", "pointer", pointer))
	}
	r.acceptedActivePointers = append(r.acceptedActivePointers, pointer)
	r.activePointer = foundation.Some(pointer)
	if !r.OnlyAcceptDragOnThreshold || r.hasDragThresholdBeenMet {
		r.checkDrag(pointer)
	}
}

// RejectGesture gives up the pointer.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.rejectGesture
func (r *DragGestureRecognizer) RejectGesture(pointer int) { r.giveUpPointer(pointer) }

// DidStopTrackingLastPointer ends or cancels the drag.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.didStopTrackingLastPointer
func (r *DragGestureRecognizer) DidStopTrackingLastPointer(pointer int) {
	if foundation.DebugMode && r.state == dragReady {
		panic(errors.New("DragStoppedWhileReady", "the last pointer stopped while the drag recognizer is ready"))
	}
	switch r.state {
	case dragPossible:
		r.oneSeq.Resolve(GestureRejected)
		r.checkCancel()
	case dragAccepted:
		r.checkEnd(pointer)
	}
	r.hasDragThresholdBeenMet = false
	r.velocityTrackers = map[int]*VelocityTracker{}
	r.initialButtons = foundation.None[int]()
	r.state = dragReady
}

func (r *DragGestureRecognizer) giveUpPointer(pointer int) {
	r.StopTrackingPointer(pointer)
	if i := slices.Index(r.acceptedActivePointers, pointer); i >= 0 {
		r.acceptedActivePointers = slices.Delete(r.acceptedActivePointers, i, i+1)
	} else {
		r.oneSeq.ResolvePointer(pointer, GestureRejected)
	}
	if _, ok := r.moveDeltaBeforeFrame[pointer]; ok {
		delete(r.moveDeltaBeforeFrame, pointer)
		r.moveDeltaOrder = slices.DeleteFunc(r.moveDeltaOrder, func(p int) bool { return p == pointer })
	}
	if ap, ok := r.activePointer.Get(); ok && ap == pointer {
		if len(r.acceptedActivePointers) > 0 {
			r.activePointer = foundation.Some(r.acceptedActivePointers[0])
		} else {
			r.activePointer = foundation.None[int]()
		}
	}
}

func (r *DragGestureRecognizer) checkDown() {
	if r.OnDown != nil {
		details := NewDragDownDetails(DragDownOpts{
			GlobalPosition: r.initialPosition.Global, LocalPosition: foundation.Some(r.initialPosition.Local),
		})
		r.InvokeCallback("onDown", func() { r.OnDown(details) })
	}
}

func (r *DragGestureRecognizer) checkDrag(pointer int) {
	if r.state == dragAccepted {
		return
	}
	r.state = dragAccepted
	delta := r.pendingDragOffset
	timestamp := r.lastPendingEventTimeStmp
	transform := r.lastTransform
	var localUpdateDelta ui.Offset
	switch r.DragStartBehavior {
	case DragStartBehaviorStart:
		r.initialPosition = r.initialPosition.Add(delta)
	case DragStartBehaviorDown:
		localUpdateDelta = r.axis.getDeltaForDetails(delta.Local)
	}
	r.pendingDragOffset = OffsetPair{}
	r.lastPendingEventTimeStmp = foundation.None[time.Duration]()
	r.lastTransform = nil
	r.checkStart(timestamp, pointer)
	if localUpdateDelta != (ui.Offset{}) && r.OnUpdate != nil {
		var localToGlobal *ui.Matrix4
		if transform != nil {
			localToGlobal = tryInvert(transform)
		}
		correctedLocalPosition := r.initialPosition.Local.Add(localUpdateDelta)
		globalUpdateDelta := TransformDeltaViaPositions(correctedLocalPosition, localUpdateDelta, localToGlobal)
		updateDelta := OffsetPair{Local: localUpdateDelta, Global: globalUpdateDelta}
		correctedPosition := r.initialPosition.Add(updateDelta) // Only adds delta for down behaviour
		r.checkUpdate(timestamp, localUpdateDelta, r.axis.getPrimaryValueFromOffset(localUpdateDelta),
			correctedPosition.Global, foundation.Some(correctedPosition.Local), pointer)
	}
	r.oneSeq.Resolve(GestureAccepted)
}

func (r *DragGestureRecognizer) checkStart(timestamp foundation.Opt[time.Duration], pointer int) {
	if r.OnStart != nil {
		details := NewDragStartDetails(DragStartOpts{
			SourceTimeStamp: timestamp, GlobalPosition: r.initialPosition.Global,
			LocalPosition: foundation.Some(r.initialPosition.Local),
			Kind:          foundation.Some(r.GetKindForPointer(pointer)),
		})
		r.InvokeCallback("onStart", func() { r.OnStart(details) })
	}
}

func (r *DragGestureRecognizer) checkUpdate(sourceTimeStamp foundation.Opt[time.Duration], delta ui.Offset,
	primaryDelta foundation.Opt[float64], globalPosition ui.Offset, localPosition foundation.Opt[ui.Offset], pointer int) {
	if r.OnUpdate != nil {
		details := NewDragUpdateDetails(globalPosition, DragUpdateOpts{
			SourceTimeStamp: sourceTimeStamp, Delta: delta, PrimaryDelta: primaryDelta,
			LocalPosition: localPosition, Kind: foundation.Some(r.GetKindForPointer(pointer)),
		})
		r.InvokeCallback("onUpdate", func() { r.OnUpdate(details) })
	}
}

func (r *DragGestureRecognizer) checkEnd(pointer int) {
	if r.OnEnd == nil {
		return
	}
	tracker := r.velocityTrackers[pointer]
	estimate, ok := tracker.GetVelocityEstimate().Get()
	var details foundation.Opt[DragEndDetails]
	var debugReport func() string
	if !ok {
		debugReport = func() string { return "Could not estimate velocity." }
	} else {
		details = r.axis.considerFling(r, estimate, tracker.Kind())
		if d, fling := details.Get(); fling {
			debugReport = func() string { return estimate.String() + "; fling at " + d.Velocity.String() + "." }
		} else {
			debugReport = func() string { return estimate.String() + "; judged to not be a fling." }
		}
	}
	end, ok := details.Get()
	if !ok {
		end = NewDragEndDetails(DragEndOpts{
			PrimaryVelocity: foundation.Some(0.0), GlobalPosition: r.lastPosition.Global,
			LocalPosition: foundation.Some(r.lastPosition.Local),
		})
	}
	r.InvokeCallback("onEnd", func() { r.OnEnd(end) }, debugReport)
}

func (r *DragGestureRecognizer) checkCancel() {
	if r.OnCancel != nil {
		r.InvokeCallback("onCancel", r.OnCancel)
	}
}

// Dispose releases the velocity trackers.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.dispose
func (r *DragGestureRecognizer) Dispose() {
	r.velocityTrackers = map[int]*VelocityTracker{}
	r.OneSequenceGestureRecognizer.Dispose()
}

// DebugDescription names the recognizer by its axis.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.debugDescription
func (r *DragGestureRecognizer) DebugDescription() string { return r.axis.debugDescription() }

// DebugFillProperties adds the start behavior.
//
// Flutter: gestures/monodrag.dart DragGestureRecognizer.debugFillProperties
func (r *DragGestureRecognizer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	r.OneSequenceGestureRecognizer.DebugFillProperties(p)
	p.Add(foundation.NewEnumProperty("start behavior", foundation.Some(r.DragStartBehavior)))
}

// flingBounds returns the limits a fling is judged by.
func (r *DragGestureRecognizer) flingBounds(kind ui.PointerDeviceKind) (minVelocity, minDistance, maxVelocity float64) {
	return r.MinFlingVelocity.OrElse(MinFlingVelocity),
		r.MinFlingDistance.OrElse(ComputeHitSlop(kind, r.GestureSettings)),
		r.MaxFlingVelocity.OrElse(MaxFlingVelocity)
}

type verticalAxis struct{}

func (verticalAxis) isFlingGesture(r *DragGestureRecognizer, e VelocityEstimate, kind ui.PointerDeviceKind) bool {
	minV, minD, _ := r.flingBounds(kind)
	return math.Abs(e.PixelsPerSecond.Dy) > minV && math.Abs(e.Offset.Dy) > minD
}

func (a verticalAxis) considerFling(r *DragGestureRecognizer, e VelocityEstimate, kind ui.PointerDeviceKind) foundation.Opt[DragEndDetails] {
	if !a.isFlingGesture(r, e, kind) {
		return foundation.None[DragEndDetails]()
	}
	_, _, maxV := r.flingBounds(kind)
	dy := ui.ClampDouble(e.PixelsPerSecond.Dy, -maxV, maxV)
	return foundation.Some(NewDragEndDetails(DragEndOpts{
		Velocity: Velocity{ui.Offset{Dy: dy}}, PrimaryVelocity: foundation.Some(dy),
		GlobalPosition: r.lastPosition.Global, LocalPosition: foundation.Some(r.lastPosition.Local),
	}))
}

func (verticalAxis) hasSufficientGlobalDistanceToAccept(r *DragGestureRecognizer, kind ui.PointerDeviceKind) bool {
	return math.Abs(r.globalDistanceMoved) > ComputeHitSlop(kind, r.GestureSettings)
}
func (verticalAxis) getDeltaForDetails(d ui.Offset) ui.Offset { return ui.Offset{Dy: d.Dy} }
func (verticalAxis) getPrimaryValueFromOffset(v ui.Offset) foundation.Opt[float64] {
	return foundation.Some(v.Dy)
}
func (verticalAxis) primaryDragAxis() dragDirection { return dragDirectionVertical }
func (verticalAxis) debugDescription() string       { return "vertical drag" }

type horizontalAxis struct{}

func (horizontalAxis) isFlingGesture(r *DragGestureRecognizer, e VelocityEstimate, kind ui.PointerDeviceKind) bool {
	minV, minD, _ := r.flingBounds(kind)
	return math.Abs(e.PixelsPerSecond.Dx) > minV && math.Abs(e.Offset.Dx) > minD
}

func (a horizontalAxis) considerFling(r *DragGestureRecognizer, e VelocityEstimate, kind ui.PointerDeviceKind) foundation.Opt[DragEndDetails] {
	if !a.isFlingGesture(r, e, kind) {
		return foundation.None[DragEndDetails]()
	}
	_, _, maxV := r.flingBounds(kind)
	dx := ui.ClampDouble(e.PixelsPerSecond.Dx, -maxV, maxV)
	return foundation.Some(NewDragEndDetails(DragEndOpts{
		Velocity: Velocity{ui.Offset{Dx: dx}}, PrimaryVelocity: foundation.Some(dx),
		GlobalPosition: r.lastPosition.Global, LocalPosition: foundation.Some(r.lastPosition.Local),
	}))
}

func (horizontalAxis) hasSufficientGlobalDistanceToAccept(r *DragGestureRecognizer, kind ui.PointerDeviceKind) bool {
	return math.Abs(r.globalDistanceMoved) > ComputeHitSlop(kind, r.GestureSettings)
}
func (horizontalAxis) getDeltaForDetails(d ui.Offset) ui.Offset { return ui.Offset{Dx: d.Dx} }
func (horizontalAxis) getPrimaryValueFromOffset(v ui.Offset) foundation.Opt[float64] {
	return foundation.Some(v.Dx)
}
func (horizontalAxis) primaryDragAxis() dragDirection { return dragDirectionHorizontal }
func (horizontalAxis) debugDescription() string       { return "horizontal drag" }

type panAxis struct{}

func (panAxis) isFlingGesture(r *DragGestureRecognizer, e VelocityEstimate, kind ui.PointerDeviceKind) bool {
	minV, minD, _ := r.flingBounds(kind)
	return e.PixelsPerSecond.DistanceSquared() > minV*minV && e.Offset.DistanceSquared() > minD*minD
}

func (a panAxis) considerFling(r *DragGestureRecognizer, e VelocityEstimate, kind ui.PointerDeviceKind) foundation.Opt[DragEndDetails] {
	if !a.isFlingGesture(r, e, kind) {
		return foundation.None[DragEndDetails]()
	}
	minV, _, maxV := r.flingBounds(kind)
	velocity := Velocity{e.PixelsPerSecond}.ClampMagnitude(minV, maxV)
	return foundation.Some(NewDragEndDetails(DragEndOpts{
		Velocity: velocity, GlobalPosition: r.lastPosition.Global, LocalPosition: foundation.Some(r.lastPosition.Local),
	}))
}

func (panAxis) hasSufficientGlobalDistanceToAccept(r *DragGestureRecognizer, kind ui.PointerDeviceKind) bool {
	return math.Abs(r.globalDistanceMoved) > ComputePanSlop(kind, r.GestureSettings)
}
func (panAxis) getDeltaForDetails(d ui.Offset) ui.Offset { return d }
func (panAxis) getPrimaryValueFromOffset(ui.Offset) foundation.Opt[float64] {
	return foundation.None[float64]()
}
func (panAxis) primaryDragAxis() dragDirection { return dragDirectionNone }
func (panAxis) debugDescription() string       { return "pan" }

// VerticalDragGestureRecognizer recognizes movement in the vertical direction.
//
// Flutter: gestures/monodrag.dart VerticalDragGestureRecognizer
type VerticalDragGestureRecognizer struct{ DragGestureRecognizer }

// NewVerticalDragGestureRecognizer creates a recognizer that works with the
// binding.
//
// Flutter: gestures/monodrag.dart VerticalDragGestureRecognizer.new
func NewVerticalDragGestureRecognizer(binding *GestureBinding, opts ...DragGestureRecognizerOpts) *VerticalDragGestureRecognizer {
	r := &VerticalDragGestureRecognizer{}
	r.initDrag(binding, r, verticalAxis{}, oneOpts(opts))
	return r
}

// InitVerticalDragGestureRecognizer sets up a recognizer that embeds a
// VerticalDragGestureRecognizer: self is the outer value, which the recognizer
// calls for the methods it overrides (IsPointerAllowed and the like). It is how a
// type of your own extends the recognizer, where Dart subclasses it.
//
// Flutter: no counterpart (Go embedding)
func InitVerticalDragGestureRecognizer(r *VerticalDragGestureRecognizer, self OneSequenceHooks, binding *GestureBinding, opts ...DragGestureRecognizerOpts) {
	r.initDrag(binding, self, verticalAxis{}, oneOpts(opts))
}

// HorizontalDragGestureRecognizer recognizes movement in the horizontal
// direction.
//
// Flutter: gestures/monodrag.dart HorizontalDragGestureRecognizer
type HorizontalDragGestureRecognizer struct{ DragGestureRecognizer }

// NewHorizontalDragGestureRecognizer creates a recognizer that works with the
// binding.
//
// Flutter: gestures/monodrag.dart HorizontalDragGestureRecognizer.new
func NewHorizontalDragGestureRecognizer(binding *GestureBinding, opts ...DragGestureRecognizerOpts) *HorizontalDragGestureRecognizer {
	r := &HorizontalDragGestureRecognizer{}
	r.initDrag(binding, r, horizontalAxis{}, oneOpts(opts))
	return r
}

// InitHorizontalDragGestureRecognizer is InitVerticalDragGestureRecognizer for a
// type that embeds a HorizontalDragGestureRecognizer.
//
// Flutter: no counterpart (Go embedding)
func InitHorizontalDragGestureRecognizer(r *HorizontalDragGestureRecognizer, self OneSequenceHooks, binding *GestureBinding, opts ...DragGestureRecognizerOpts) {
	r.initDrag(binding, self, horizontalAxis{}, oneOpts(opts))
}

// PanGestureRecognizer recognizes movement in any direction.
//
// Flutter: gestures/monodrag.dart PanGestureRecognizer
type PanGestureRecognizer struct{ DragGestureRecognizer }

// NewPanGestureRecognizer creates a recognizer that works with the binding.
//
// Flutter: gestures/monodrag.dart PanGestureRecognizer.new
func NewPanGestureRecognizer(binding *GestureBinding, opts ...DragGestureRecognizerOpts) *PanGestureRecognizer {
	r := &PanGestureRecognizer{}
	r.initDrag(binding, r, panAxis{}, oneOpts(opts))
	return r
}
