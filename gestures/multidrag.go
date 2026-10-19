package gestures

import (
	"math"
	"slices"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Drag represents a drag that a MultiDragGestureRecognizer reports to the app. It
// is the object that GestureMultiDragStartCallback returns.
//
// Flutter: gestures/drag.dart Drag
type Drag interface {
	// Update is called when the pointer moves.
	//
	// Flutter: gestures/drag.dart Drag.update
	Update(details DragUpdateDetails)
	// End is called when the pointer is lifted.
	//
	// Flutter: gestures/drag.dart Drag.end
	End(details DragEndDetails)
	// Cancel is called when the drag is cancelled.
	//
	// Flutter: gestures/drag.dart Drag.cancel
	Cancel()
}

// GestureMultiDragStartCallback is called when a drag starts at a position. It
// returns the Drag to report the drag to, or nil to ignore it.
//
// Flutter: gestures/multidrag.dart GestureMultiDragStartCallback
type GestureMultiDragStartCallback func(position ui.Offset) Drag

// MultiDragPointerStateHooks are the methods of a MultiDragPointerState that the
// subclass supplies.
//
// Flutter: gestures/multidrag.dart MultiDragPointerState
type MultiDragPointerStateHooks interface {
	// Base returns the embedded state.
	Base() *MultiDragPointerState
	// CheckForResolutionAfterMove is called after the pointer moved before the
	// drag is accepted; it resolves the arena when the state is decided.
	//
	// Flutter: gestures/multidrag.dart MultiDragPointerState.checkForResolutionAfterMove
	CheckForResolutionAfterMove()
	// Accepted is called when the recognizer wins the arena; it calls the
	// starter when the drag may start.
	//
	// Flutter: gestures/multidrag.dart MultiDragPointerState.accepted
	Accepted(starter GestureMultiDragStartCallback)
	// Rejected is called when the recognizer loses the arena.
	//
	// Flutter: gestures/multidrag.dart MultiDragPointerState.rejected
	Rejected()
	// Dispose releases the state.
	//
	// Flutter: gestures/multidrag.dart MultiDragPointerState.dispose
	Dispose()
}

// MultiDragPointerState is the state of one pointer of a
// MultiDragGestureRecognizer.
//
// Flutter: gestures/multidrag.dart MultiDragPointerState
type MultiDragPointerState struct {
	self            MultiDragPointerStateHooks
	gestureSettings foundation.Opt[DeviceGestureSettings]
	// InitialPosition is where the pointer went down.
	//
	// Flutter: gestures/multidrag.dart MultiDragPointerState.initialPosition
	InitialPosition ui.Offset
	// Kind is the kind of device of the pointer.
	//
	// Flutter: gestures/multidrag.dart MultiDragPointerState.kind
	Kind            ui.PointerDeviceKind
	velocityTracker *VelocityTracker

	client                   Drag
	pendingDelta             foundation.Opt[ui.Offset]
	lastPendingEventTimeStmp foundation.Opt[time.Duration]
	arenaEntry               GestureArenaEntry
}

// InitMultiDragPointerState sets up the state; self is the state that embeds it.
//
// Flutter: gestures/multidrag.dart MultiDragPointerState.new
func (s *MultiDragPointerState) InitMultiDragPointerState(clock SamplingClock, self MultiDragPointerStateHooks, initialPosition ui.Offset,
	kind ui.PointerDeviceKind, settings foundation.Opt[DeviceGestureSettings]) {
	s.self = self
	s.InitialPosition = initialPosition
	s.Kind = kind
	s.gestureSettings = settings
	s.velocityTracker = NewVelocityTrackerWithKind(clock, kind)
	s.pendingDelta = foundation.Some(ui.Offset{})
}

// Base returns the state.
//
// Flutter: gestures/multidrag.dart MultiDragPointerState.this
func (s *MultiDragPointerState) Base() *MultiDragPointerState { return s }

// GestureSettings are the gesture thresholds of the device.
//
// Flutter: gestures/multidrag.dart MultiDragPointerState.gestureSettings
func (s *MultiDragPointerState) GestureSettings() foundation.Opt[DeviceGestureSettings] {
	return s.gestureSettings
}

// PendingDelta is how far the pointer moved before the drag started; None once
// the state ended.
//
// Flutter: gestures/multidrag.dart MultiDragPointerState.pendingDelta
func (s *MultiDragPointerState) PendingDelta() foundation.Opt[ui.Offset] { return s.pendingDelta }

func (s *MultiDragPointerState) setArenaEntry(entry GestureArenaEntry) { s.arenaEntry = entry }

// Resolve accepts or rejects the pointer in the arena.
//
// Flutter: gestures/multidrag.dart MultiDragPointerState.resolve
func (s *MultiDragPointerState) Resolve(disposition GestureDisposition) {
	s.arenaEntry.Resolve(disposition)
}

func (s *MultiDragPointerState) move(event *PointerMoveEvent) {
	if !event.Synthesized() {
		s.velocityTracker.AddPosition(event.TimeStamp(), event.Position())
	}
	if s.client != nil {
		s.client.Update(NewDragUpdateDetails(event.Position(), DragUpdateOpts{
			SourceTimeStamp: foundation.Some(event.TimeStamp()), Delta: event.Delta(),
		}))
	} else {
		s.pendingDelta = foundation.Some(s.pendingDelta.Value().Add(event.Delta()))
		s.lastPendingEventTimeStmp = foundation.Some(event.TimeStamp())
		s.self.CheckForResolutionAfterMove()
	}
}

// CheckForResolutionAfterMove does nothing.
//
// Flutter: gestures/multidrag.dart MultiDragPointerState.checkForResolutionAfterMove
func (s *MultiDragPointerState) CheckForResolutionAfterMove() {}

// Rejected forgets the pending delta.
//
// Flutter: gestures/multidrag.dart MultiDragPointerState.rejected
func (s *MultiDragPointerState) Rejected() {
	s.pendingDelta = foundation.None[ui.Offset]()
	s.lastPendingEventTimeStmp = foundation.None[time.Duration]()
	s.arenaEntry = nil
}

func (s *MultiDragPointerState) startDrag(client Drag) {
	s.client = client
	details := NewDragUpdateDetails(s.InitialPosition, DragUpdateOpts{
		SourceTimeStamp: s.lastPendingEventTimeStmp, Delta: s.pendingDelta.Value(),
	})
	s.pendingDelta = foundation.None[ui.Offset]()
	s.lastPendingEventTimeStmp = foundation.None[time.Duration]()
	s.client.Update(details)
}

func (s *MultiDragPointerState) up() {
	if s.client != nil {
		details := NewDragEndDetails(DragEndOpts{Velocity: s.velocityTracker.GetVelocity()})
		client := s.client
		s.client = nil
		client.End(details)
	} else {
		s.pendingDelta = foundation.None[ui.Offset]()
		s.lastPendingEventTimeStmp = foundation.None[time.Duration]()
	}
}

func (s *MultiDragPointerState) cancel() {
	if s.client != nil {
		client := s.client
		s.client = nil
		client.Cancel()
	} else {
		s.pendingDelta = foundation.None[ui.Offset]()
		s.lastPendingEventTimeStmp = foundation.None[time.Duration]()
	}
}

// Dispose rejects the pointer if it is still in the arena.
//
// Flutter: gestures/multidrag.dart MultiDragPointerState.dispose
func (s *MultiDragPointerState) Dispose() {
	if s.arenaEntry != nil {
		s.arenaEntry.Resolve(GestureRejected)
		s.arenaEntry = nil
	}
	s.pendingDelta = foundation.None[ui.Offset]()
}

// MultiDragHooks are the methods of a MultiDragGestureRecognizer that the
// subclass supplies.
//
// Flutter: gestures/multidrag.dart MultiDragGestureRecognizer
type MultiDragHooks interface {
	GestureRecognizerHooks
	// CreateNewPointerState makes the state of a pointer that went down.
	//
	// Flutter: gestures/multidrag.dart MultiDragGestureRecognizer.createNewPointerState
	CreateNewPointerState(event *PointerDownEvent) MultiDragPointerStateHooks
}

// MultiDragGestureRecognizerOpts are the optional parameters of the multi-drag
// recognizers.
//
// Flutter: gestures/multidrag.dart MultiDragGestureRecognizer.new
type MultiDragGestureRecognizerOpts struct {
	// DebugOwner is the object that owns the recognizer.
	DebugOwner any
	// SupportedDevices are the kinds of pointer the recognizer accepts; nil for
	// all.
	SupportedDevices []ui.PointerDeviceKind
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed;
	// the default allows the primary button only.
	AllowedButtonsFilter AllowedButtonsFilter
}

// MultiDragGestureRecognizer is the base of the recognizers that track several
// pointers, each as its own drag.
//
// Flutter: gestures/multidrag.dart MultiDragGestureRecognizer
type MultiDragGestureRecognizer struct {
	GestureRecognizerBase
	multi MultiDragHooks

	// OnStart is called when a drag starts. It returns the Drag to report the
	// drag to, or nil to ignore it.
	//
	// Flutter: gestures/multidrag.dart MultiDragGestureRecognizer.onStart
	OnStart GestureMultiDragStartCallback

	pointers      map[int]MultiDragPointerStateHooks
	pointerRoutes map[int]PointerRouteHandle
	disposed      bool
}

// InitMultiDrag sets up the recognizer; self is the recognizer that embeds it.
//
// Flutter: gestures/multidrag.dart MultiDragGestureRecognizer.new
func (r *MultiDragGestureRecognizer) InitMultiDrag(binding *GestureBinding, self MultiDragHooks, opts MultiDragGestureRecognizerOpts) {
	filter := opts.AllowedButtonsFilter
	if filter == nil {
		filter = func(b int) bool { return b == PrimaryButton }
	}
	r.Init(binding, self, GestureRecognizerOpts{DebugOwner: opts.DebugOwner, SupportedDevices: opts.SupportedDevices, AllowedButtonsFilter: filter})
	r.multi = self
	r.pointers = map[int]MultiDragPointerStateHooks{}
	r.pointerRoutes = map[int]PointerRouteHandle{}
}

// AddAllowedPointer creates the state of the pointer and adds the recognizer to
// its arena.
//
// Flutter: gestures/multidrag.dart MultiDragGestureRecognizer.addAllowedPointer
func (r *MultiDragGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	state := r.multi.CreateNewPointerState(event)
	r.pointers[event.Pointer()] = state
	r.pointerRoutes[event.Pointer()] = r.binding.PointerRouter.AddRoute(event.Pointer(), r.handleEvent, nil)
	state.Base().setArenaEntry(r.binding.GestureArena.Add(event.Pointer(), r.multi))
}

func (r *MultiDragGestureRecognizer) handleEvent(event PointerEvent) {
	state := r.pointers[event.Pointer()]
	switch e := event.(type) {
	case *PointerMoveEvent:
		state.Base().move(e)
	case *PointerUpEvent:
		state.Base().up()
		r.removeState(event.Pointer())
	case *PointerCancelEvent:
		state.Base().cancel()
		r.removeState(event.Pointer())
	}
}

// AcceptGesture starts the drag of the pointer.
//
// Flutter: gestures/multidrag.dart MultiDragGestureRecognizer.acceptGesture
func (r *MultiDragGestureRecognizer) AcceptGesture(pointer int) {
	state := r.pointers[pointer]
	if state == nil {
		return // We might already have canceled this drag if the up comes before the accept.
	}
	state.Accepted(func(initialPosition ui.Offset) Drag { return r.startDrag(initialPosition, pointer) })
}

func (r *MultiDragGestureRecognizer) startDrag(initialPosition ui.Offset, pointer int) Drag {
	state := r.pointers[pointer]
	var drag Drag
	if r.OnStart != nil {
		drag = InvokeCallbackResult(&r.GestureRecognizerBase, "onStart", func() Drag { return r.OnStart(initialPosition) })
	}
	if drag != nil {
		state.Base().startDrag(drag)
	} else {
		r.removeState(pointer)
	}
	return drag
}

// RejectGesture ends the state of the pointer.
//
// Flutter: gestures/multidrag.dart MultiDragGestureRecognizer.rejectGesture
func (r *MultiDragGestureRecognizer) RejectGesture(pointer int) {
	if state, ok := r.pointers[pointer]; ok {
		state.Rejected()
		r.removeState(pointer)
	} // else we already preemptively forgot about it (e.g. we got an up event)
}

func (r *MultiDragGestureRecognizer) removeState(pointer int) {
	if r.disposed {
		return
	}
	if foundation.DebugMode && r.pointers[pointer] == nil {
		panic(errors.New("MultiDragStateUnknown", "the pointer has no state", "pointer", pointer))
	}
	r.binding.PointerRouter.RemoveRoute(pointer, r.pointerRoutes[pointer])
	delete(r.pointerRoutes, pointer)
	state := r.pointers[pointer]
	delete(r.pointers, pointer)
	state.Dispose()
}

// Dispose ends the state of every pointer.
//
// Flutter: gestures/multidrag.dart MultiDragGestureRecognizer.dispose
func (r *MultiDragGestureRecognizer) Dispose() {
	keys := make([]int, 0, len(r.pointers))
	for p := range r.pointers {
		keys = append(keys, p)
	}
	slices.Sort(keys)
	for _, p := range keys {
		r.removeState(p)
	}
	r.disposed = true
	r.GestureRecognizerBase.Dispose()
}

// axisPointerState is the state of a pointer that is accepted when it moved
// past the slop along an axis (or in any direction).
type axisPointerState struct {
	MultiDragPointerState
	measure func(delta ui.Offset) float64
}

func (s *axisPointerState) CheckForResolutionAfterMove() {
	if s.measure(s.pendingDelta.Value()) > ComputeHitSlop(s.Kind, s.gestureSettings) {
		s.Resolve(GestureAccepted)
	}
}

func (s *axisPointerState) Accepted(starter GestureMultiDragStartCallback) {
	starter(s.InitialPosition)
}

func newAxisPointerState(binding *GestureBinding, event *PointerDownEvent, settings foundation.Opt[DeviceGestureSettings],
	measure func(ui.Offset) float64) *axisPointerState {
	s := &axisPointerState{measure: measure}
	s.InitMultiDragPointerState(binding.Clock(), s, event.Position(), event.Kind(), settings)
	return s
}

// ImmediateMultiDragGestureRecognizer starts a drag as soon as a pointer moves
// past the slop in any direction.
//
// Flutter: gestures/multidrag.dart ImmediateMultiDragGestureRecognizer
type ImmediateMultiDragGestureRecognizer struct{ MultiDragGestureRecognizer }

// NewImmediateMultiDragGestureRecognizer creates a recognizer that works with the
// binding.
//
// Flutter: gestures/multidrag.dart ImmediateMultiDragGestureRecognizer.new
func NewImmediateMultiDragGestureRecognizer(binding *GestureBinding, opts ...MultiDragGestureRecognizerOpts) *ImmediateMultiDragGestureRecognizer {
	r := &ImmediateMultiDragGestureRecognizer{}
	r.InitMultiDrag(binding, r, oneOpts(opts))
	return r
}

// CreateNewPointerState makes the state of the pointer.
//
// Flutter: gestures/multidrag.dart ImmediateMultiDragGestureRecognizer.createNewPointerState
func (r *ImmediateMultiDragGestureRecognizer) CreateNewPointerState(event *PointerDownEvent) MultiDragPointerStateHooks {
	return newAxisPointerState(r.binding, event, r.GestureSettings, ui.Offset.Distance)
}

// DebugDescription is "multidrag".
//
// Flutter: gestures/multidrag.dart ImmediateMultiDragGestureRecognizer.debugDescription
func (r *ImmediateMultiDragGestureRecognizer) DebugDescription() string { return "multidrag" }

// HorizontalMultiDragGestureRecognizer starts a drag as soon as a pointer moves
// past the slop horizontally.
//
// Flutter: gestures/multidrag.dart HorizontalMultiDragGestureRecognizer
type HorizontalMultiDragGestureRecognizer struct{ MultiDragGestureRecognizer }

// NewHorizontalMultiDragGestureRecognizer creates a recognizer that works with the
// binding.
//
// Flutter: gestures/multidrag.dart HorizontalMultiDragGestureRecognizer.new
func NewHorizontalMultiDragGestureRecognizer(binding *GestureBinding, opts ...MultiDragGestureRecognizerOpts) *HorizontalMultiDragGestureRecognizer {
	r := &HorizontalMultiDragGestureRecognizer{}
	r.InitMultiDrag(binding, r, oneOpts(opts))
	return r
}

// CreateNewPointerState makes the state of the pointer.
//
// Flutter: gestures/multidrag.dart HorizontalMultiDragGestureRecognizer.createNewPointerState
func (r *HorizontalMultiDragGestureRecognizer) CreateNewPointerState(event *PointerDownEvent) MultiDragPointerStateHooks {
	return newAxisPointerState(r.binding, event, r.GestureSettings, func(d ui.Offset) float64 { return math.Abs(d.Dx) })
}

// DebugDescription is "horizontal multidrag".
//
// Flutter: gestures/multidrag.dart HorizontalMultiDragGestureRecognizer.debugDescription
func (r *HorizontalMultiDragGestureRecognizer) DebugDescription() string {
	return "horizontal multidrag"
}

// VerticalMultiDragGestureRecognizer starts a drag as soon as a pointer moves
// past the slop vertically.
//
// Flutter: gestures/multidrag.dart VerticalMultiDragGestureRecognizer
type VerticalMultiDragGestureRecognizer struct{ MultiDragGestureRecognizer }

// NewVerticalMultiDragGestureRecognizer creates a recognizer that works with the
// binding.
//
// Flutter: gestures/multidrag.dart VerticalMultiDragGestureRecognizer.new
func NewVerticalMultiDragGestureRecognizer(binding *GestureBinding, opts ...MultiDragGestureRecognizerOpts) *VerticalMultiDragGestureRecognizer {
	r := &VerticalMultiDragGestureRecognizer{}
	r.InitMultiDrag(binding, r, oneOpts(opts))
	return r
}

// CreateNewPointerState makes the state of the pointer.
//
// Flutter: gestures/multidrag.dart VerticalMultiDragGestureRecognizer.createNewPointerState
func (r *VerticalMultiDragGestureRecognizer) CreateNewPointerState(event *PointerDownEvent) MultiDragPointerStateHooks {
	return newAxisPointerState(r.binding, event, r.GestureSettings, func(d ui.Offset) float64 { return math.Abs(d.Dy) })
}

// DebugDescription is "vertical multidrag".
//
// Flutter: gestures/multidrag.dart VerticalMultiDragGestureRecognizer.debugDescription
func (r *VerticalMultiDragGestureRecognizer) DebugDescription() string { return "vertical multidrag" }

type delayedPointerState struct {
	MultiDragPointerState
	timer   *Timer
	starter GestureMultiDragStartCallback
}

func (s *delayedPointerState) delayPassed() {
	s.timer = nil
	if s.starter != nil {
		s.starter(s.InitialPosition)
		s.starter = nil
	} else {
		s.Resolve(GestureAccepted)
	}
}

func (s *delayedPointerState) ensureTimerStopped() {
	if s.timer != nil {
		s.timer.Cancel()
		s.timer = nil
	}
}

func (s *delayedPointerState) Accepted(starter GestureMultiDragStartCallback) {
	if s.timer == nil {
		starter(s.InitialPosition)
	} else {
		s.starter = starter
	}
}

func (s *delayedPointerState) CheckForResolutionAfterMove() {
	if s.timer == nil {
		// If we've been accepted by the gesture arena but the pointer moves too
		// much before the delay passes, ignore: the starter is called when the
		// timer expires.
		return
	}
	if s.pendingDelta.Value().Distance() > ComputeHitSlop(s.Kind, s.gestureSettings) {
		s.Resolve(GestureRejected)
		s.ensureTimerStopped()
	}
}

func (s *delayedPointerState) Dispose() {
	s.ensureTimerStopped()
	s.MultiDragPointerState.Dispose()
}

// DelayedMultiDragGestureRecognizerOpts are the optional parameters of
// NewDelayedMultiDragGestureRecognizer.
//
// Flutter: gestures/multidrag.dart DelayedMultiDragGestureRecognizer.new
type DelayedMultiDragGestureRecognizerOpts struct {
	MultiDragGestureRecognizerOpts
	// Delay is how long a pointer must stay down before a drag; the default is
	// LongPressTimeout.
	Delay foundation.Opt[time.Duration]
}

// DelayedMultiDragGestureRecognizer starts a drag after a pointer has stayed down
// for a delay without moving past the slop.
//
// Flutter: gestures/multidrag.dart DelayedMultiDragGestureRecognizer
type DelayedMultiDragGestureRecognizer struct {
	MultiDragGestureRecognizer
	delay time.Duration
}

// NewDelayedMultiDragGestureRecognizer creates a recognizer that works with the
// binding.
//
// Flutter: gestures/multidrag.dart DelayedMultiDragGestureRecognizer.new
func NewDelayedMultiDragGestureRecognizer(binding *GestureBinding, opts ...DelayedMultiDragGestureRecognizerOpts) *DelayedMultiDragGestureRecognizer {
	o := oneOpts(opts)
	r := &DelayedMultiDragGestureRecognizer{delay: o.Delay.OrElse(LongPressTimeout)}
	r.InitMultiDrag(binding, r, o.MultiDragGestureRecognizerOpts)
	return r
}

// Delay is how long a pointer must stay down before a drag.
//
// Flutter: gestures/multidrag.dart DelayedMultiDragGestureRecognizer.delay
func (r *DelayedMultiDragGestureRecognizer) Delay() time.Duration { return r.delay }

// CreateNewPointerState makes the state of the pointer, with its timer.
//
// Flutter: gestures/multidrag.dart DelayedMultiDragGestureRecognizer.createNewPointerState
func (r *DelayedMultiDragGestureRecognizer) CreateNewPointerState(event *PointerDownEvent) MultiDragPointerStateHooks {
	s := &delayedPointerState{}
	s.InitMultiDragPointerState(r.binding.Clock(), s, event.Position(), event.Kind(), r.GestureSettings)
	s.timer = r.binding.NewTimer(r.delay, s.delayPassed)
	return s
}

// DebugDescription is "long multidrag".
//
// Flutter: gestures/multidrag.dart DelayedMultiDragGestureRecognizer.debugDescription
func (r *DelayedMultiDragGestureRecognizer) DebugDescription() string { return "long multidrag" }
