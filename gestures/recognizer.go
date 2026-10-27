package gestures

import (
	"fmt"
	"strings"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// DragStartBehavior is when a drag starts, for the details it reports.
//
// Flutter: gestures/recognizer.dart DragStartBehavior
type DragStartBehavior int

const (
	// DragStartBehaviorUnspecified is the zero value: the widget's default,
	// which is DragStartBehaviorStart.
	DragStartBehaviorUnspecified DragStartBehavior = iota
	// DragStartBehaviorDown starts the drag where the pointer touched down.
	//
	// Flutter: gestures/recognizer.dart DragStartBehavior.down
	DragStartBehaviorDown
	// DragStartBehaviorStart starts the drag where the pointer was when the
	// gesture won the arena.
	//
	// Flutter: gestures/recognizer.dart DragStartBehavior.start
	DragStartBehaviorStart
)

// Name returns the Dart name of the behavior.
//
// Flutter: gestures/recognizer.dart DragStartBehavior
func (b DragStartBehavior) Name() string {
	switch b {
	case DragStartBehaviorDown:
		return "down"
	case DragStartBehaviorStart:
		return "start"
	}
	return "DragStartBehavior(" + fmt.Sprint(int(b)) + ")"
}

// orDefault returns the behavior, or DragStartBehaviorStart if it is not set.
func (b DragStartBehavior) orDefault() DragStartBehavior {
	if b == DragStartBehaviorUnspecified {
		return DragStartBehaviorStart
	}
	return b
}

// MultitouchDragStrategy is how a drag follows several fingers.
//
// Flutter: gestures/recognizer.dart MultitouchDragStrategy
type MultitouchDragStrategy int

const (
	// MultitouchDragStrategyUnspecified is the zero value: the default of the
	// platform (see DragGestureRecognizer).
	MultitouchDragStrategyUnspecified MultitouchDragStrategy = iota
	// MultitouchDragStrategyLatestPointer follows the most recent pointer.
	//
	// Flutter: gestures/recognizer.dart MultitouchDragStrategy.latestPointer
	MultitouchDragStrategyLatestPointer
	// MultitouchDragStrategyAverageBoundaryPointers follows the average of the
	// pointers that are farthest left, right, up and down.
	//
	// Flutter: gestures/recognizer.dart MultitouchDragStrategy.averageBoundaryPointers
	MultitouchDragStrategyAverageBoundaryPointers
	// MultitouchDragStrategySumAllPointers follows the sum of all the pointers.
	//
	// Flutter: gestures/recognizer.dart MultitouchDragStrategy.sumAllPointers
	MultitouchDragStrategySumAllPointers
)

// Name returns the Dart name of the strategy.
//
// Flutter: gestures/recognizer.dart MultitouchDragStrategy
func (s MultitouchDragStrategy) Name() string {
	switch s {
	case MultitouchDragStrategyLatestPointer:
		return "latestPointer"
	case MultitouchDragStrategyAverageBoundaryPointers:
		return "averageBoundaryPointers"
	case MultitouchDragStrategySumAllPointers:
		return "sumAllPointers"
	}
	return "MultitouchDragStrategy(" + fmt.Sprint(int(s)) + ")"
}

// AllowedButtonsFilter says whether a gesture may start with the buttons
// pressed.
//
// Flutter: gestures/recognizer.dart AllowedButtonsFilter
type AllowedButtonsFilter func(buttons int) bool

// GestureRecognizer is the interface of all recognizers: it accepts pointers
// and takes part in the gesture arena.
//
// A recognizer is built from the embedded GestureRecognizerBase, then
// OneSequenceGestureRecognizer and PrimaryPointerGestureRecognizer, as
// Flutter's classes extend one another. The base calls the methods the
// subclass overrides (Dart's virtual methods) through the "self" the subclass
// gives to its Init.
//
// Flutter: gestures/recognizer.dart GestureRecognizer
type GestureRecognizer interface {
	GestureArenaMember
	foundation.Diagnosticable
	// AddPointer starts tracking the pointer of the down event, if the
	// recognizer allows it.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.addPointer
	AddPointer(event *PointerDownEvent)
	// AddPointerPanZoom starts tracking the trackpad gesture, if the recognizer
	// allows it.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.addPointerPanZoom
	AddPointerPanZoom(event *PointerPanZoomStartEvent)
	// Dispose releases the resources of the recognizer. It must not be used
	// afterwards.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.dispose
	Dispose()
	// DebugDescription is a short name for debug output.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.debugDescription
	DebugDescription() string
	// Base returns the embedded base, whose fields (the supported devices, the
	// gesture settings and the button filter) the owner of the recognizer sets.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer
	Base() *GestureRecognizerBase
}

// GestureRecognizerHooks are the methods of a recognizer that GestureRecognizerBase
// calls and a subclass may override. The base has a default for each.
//
// Flutter: gestures/recognizer.dart GestureRecognizer
type GestureRecognizerHooks interface {
	GestureRecognizer
	// AddAllowedPointer is called by AddPointer for a pointer that
	// IsPointerAllowed.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.addAllowedPointer
	AddAllowedPointer(event *PointerDownEvent)
	// HandleNonAllowedPointer is called by AddPointer for a pointer that is not
	// allowed.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.handleNonAllowedPointer
	HandleNonAllowedPointer(event *PointerDownEvent)
	// IsPointerAllowed says whether the recognizer takes the pointer: its
	// device is supported and its buttons pass the filter.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.isPointerAllowed
	IsPointerAllowed(event *PointerDownEvent) bool
	// AddAllowedPointerPanZoom is called by AddPointerPanZoom for a gesture
	// that IsPointerPanZoomAllowed.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.addAllowedPointerPanZoom
	AddAllowedPointerPanZoom(event *PointerPanZoomStartEvent)
	// HandleNonAllowedPointerPanZoom is called by AddPointerPanZoom for a
	// gesture that is not allowed.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.handleNonAllowedPointerPanZoom
	HandleNonAllowedPointerPanZoom(event *PointerPanZoomStartEvent)
	// IsPointerPanZoomAllowed says whether the recognizer takes the trackpad
	// gesture: its device is supported.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.isPointerPanZoomAllowed
	IsPointerPanZoomAllowed(event *PointerPanZoomStartEvent) bool
}

// GestureRecognizerOpts are the optional parameters that every recognizer
// takes. The options structs of the recognizers repeat these fields.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.new
type GestureRecognizerOpts struct {
	// DebugOwner is the object that owns the recognizer, for debug output.
	DebugOwner any
	// SupportedDevices are the kinds of pointer the recognizer accepts. If it
	// is nil, all kinds are accepted; if it is empty, none is.
	SupportedDevices []ui.PointerDeviceKind
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed;
	// the default allows any buttons.
	AllowedButtonsFilter AllowedButtonsFilter
}

type recognizerEventData struct {
	kind    ui.PointerDeviceKind
	buttons int
}

// GestureRecognizerBase is the state and behavior that all recognizers share.
// Embed it through OneSequenceGestureRecognizer or one of the recognizers that
// extend it.
//
// Flutter: gestures/recognizer.dart GestureRecognizer
type GestureRecognizerBase struct {
	binding *GestureBinding
	self    GestureRecognizerHooks

	// DebugOwner is the object that owns the recognizer, for debug output.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.debugOwner
	DebugOwner any
	// GestureSettings are the gesture thresholds of the device; the owner sets
	// them from the media query.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.gestureSettings
	GestureSettings foundation.Opt[DeviceGestureSettings]
	// SupportedDevices are the kinds of pointer the recognizer accepts. If it
	// is nil, all kinds are accepted; if it is empty, none is.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.supportedDevices
	SupportedDevices []ui.PointerDeviceKind
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizer.allowedButtonsFilter
	AllowedButtonsFilter AllowedButtonsFilter

	pointerToEventData map[int]recognizerEventData
	// pointerRing holds the pointers of pointerToEventData in the order they were added, to
	// forget the oldest (see rememberPointer).
	pointerRing []int
	ringNext    int
}

// maxRememberedPointers is how many pointers a recognizer remembers the kind and buttons of.
// Flutter keeps them for as long as the recognizer lives, and with a new pointer id for every
// press that is a record for each press of a recognizer that lives as long as the app. A pointer
// that is still down is one of the last few, so the oldest are forgotten.
const maxRememberedPointers = 256

// rememberPointer keeps the data of the pointer, and forgets the data of the pointer that was
// added maxRememberedPointers pointers ago.
func (r *GestureRecognizerBase) rememberPointer(pointer int, data recognizerEventData) {
	if _, ok := r.pointerToEventData[pointer]; !ok {
		if len(r.pointerRing) < maxRememberedPointers {
			r.pointerRing = append(r.pointerRing, pointer)
		} else {
			delete(r.pointerToEventData, r.pointerRing[r.ringNext])
			r.pointerRing[r.ringNext] = pointer
			r.ringNext = (r.ringNext + 1) % maxRememberedPointers
		}
	}
	r.pointerToEventData[pointer] = data
}

// Init sets up the base. self is the recognizer that embeds it (usually its
// pointer), which the base calls for the methods the recognizer overrides.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.new
func (r *GestureRecognizerBase) Init(binding *GestureBinding, self GestureRecognizerHooks, opts GestureRecognizerOpts) {
	r.binding = binding
	r.self = self
	r.DebugOwner = opts.DebugOwner
	r.SupportedDevices = opts.SupportedDevices
	r.AllowedButtonsFilter = opts.AllowedButtonsFilter
	if r.AllowedButtonsFilter == nil {
		r.AllowedButtonsFilter = func(int) bool { return true }
	}
	r.pointerToEventData = map[int]recognizerEventData{}
}

// Base returns the base, for the GestureRecognizer interface.
//
// Flutter: gestures/recognizer.dart GestureRecognizer
func (r *GestureRecognizerBase) Base() *GestureRecognizerBase { return r }

// Binding is the gesture binding the recognizer works with.
//
// Deviation: Flutter's recognizers reach GestureBinding.instance (SPEC §12.10).
//
// Flutter: gestures/recognizer.dart GestureRecognizerBase.GestureBinding.instance
func (r *GestureRecognizerBase) Binding() *GestureBinding { return r.binding }

func (r *GestureRecognizerBase) supports(kind ui.PointerDeviceKind) bool {
	if r.SupportedDevices == nil {
		return true
	}
	for _, k := range r.SupportedDevices {
		if k == kind {
			return true
		}
	}
	return false
}

// AddPointerPanZoom remembers the kind and buttons of the pointer, and hands the
// trackpad gesture to AddAllowedPointerPanZoom, or to
// HandleNonAllowedPointerPanZoom if it is not allowed.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.addPointerPanZoom
func (r *GestureRecognizerBase) AddPointerPanZoom(event *PointerPanZoomStartEvent) {
	r.rememberPointer(event.Pointer(), recognizerEventData{kind: event.Kind(), buttons: event.Buttons()})
	if r.self.IsPointerPanZoomAllowed(event) {
		r.self.AddAllowedPointerPanZoom(event)
	} else {
		r.self.HandleNonAllowedPointerPanZoom(event)
	}
}

// AddAllowedPointerPanZoom is called for a trackpad gesture that the recognizer
// allows. The default does nothing.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.addAllowedPointerPanZoom
func (r *GestureRecognizerBase) AddAllowedPointerPanZoom(*PointerPanZoomStartEvent) {}

// AddPointer remembers the kind and buttons of the pointer, and hands it to
// AddAllowedPointer, or to HandleNonAllowedPointer if it is not allowed.
//
// Deviation: it remembers the kind and buttons of the last 256 pointers, not of all of them
// (SPEC §12.10).
//
// Flutter: gestures/recognizer.dart GestureRecognizer.addPointer
func (r *GestureRecognizerBase) AddPointer(event *PointerDownEvent) {
	r.rememberPointer(event.Pointer(), recognizerEventData{kind: event.Kind(), buttons: event.Buttons()})
	if r.self.IsPointerAllowed(event) {
		r.self.AddAllowedPointer(event)
	} else {
		r.self.HandleNonAllowedPointer(event)
	}
}

// AddAllowedPointer is called for a pointer that the recognizer allows. The
// default does nothing.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.addAllowedPointer
func (r *GestureRecognizerBase) AddAllowedPointer(*PointerDownEvent) {}

// HandleNonAllowedPointer is called for a pointer that the recognizer does not
// allow. The default does nothing.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.handleNonAllowedPointer
func (r *GestureRecognizerBase) HandleNonAllowedPointer(*PointerDownEvent) {}

// IsPointerAllowed reports whether the device of the pointer is supported and
// its buttons pass the filter.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.isPointerAllowed
func (r *GestureRecognizerBase) IsPointerAllowed(event *PointerDownEvent) bool {
	return r.supports(event.Kind()) && r.AllowedButtonsFilter(event.Buttons())
}

// HandleNonAllowedPointerPanZoom is called for a trackpad gesture that the
// recognizer does not allow. The default does nothing.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.handleNonAllowedPointerPanZoom
func (r *GestureRecognizerBase) HandleNonAllowedPointerPanZoom(*PointerPanZoomStartEvent) {}

// IsPointerPanZoomAllowed reports whether the device of the trackpad gesture is
// supported.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.isPointerPanZoomAllowed
func (r *GestureRecognizerBase) IsPointerPanZoomAllowed(event *PointerPanZoomStartEvent) bool {
	return r.supports(event.Kind())
}

// GetKindForPointer is the kind of device of the pointer, which must have been
// added with AddPointer or AddPointerPanZoom.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.getKindForPointer
func (r *GestureRecognizerBase) GetKindForPointer(pointer int) ui.PointerDeviceKind {
	data, ok := r.pointerToEventData[pointer]
	if foundation.DebugMode && !ok {
		panic(errors.New(tagRecognizerPointerUnknown, "the pointer was not added to the recognizer", "pointer", pointer))
	}
	return data.kind
}

// GetButtonsForPointer is the buttons that were pressed when the pointer was
// added.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.getButtonsForPointer
func (r *GestureRecognizerBase) GetButtonsForPointer(pointer int) int {
	data, ok := r.pointerToEventData[pointer]
	if foundation.DebugMode && !ok {
		panic(errors.New(tagRecognizerPointerUnknown, "the pointer was not added to the recognizer", "pointer", pointer))
	}
	return data.buttons
}

// Dispose releases the recognizer.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.dispose
func (r *GestureRecognizerBase) Dispose() {}

// InvokeCallback calls a callback of the recognizer. The call is a recovery
// point (SPEC §14.2), and it prints a trace if the debug flag is on; debugReport
// adds text to the trace.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.invokeCallback
func (r *GestureRecognizerBase) InvokeCallback(name string, callback func(), debugReport ...func() string) {
	r.traceCallback(name, debugReport)
	// A callback that panics is reported (CallbackPanicked) and the recognizer
	// goes on, as Flutter's does.
	foundation.Guard(r.binding.errorReporter, foundation.CallbackPanicked, "while handling a gesture",
		func() []any { return []any{"recognizer", foundation.DescribeIdentity(r.self), "callback", name} }, callback)
}

// InvokeCallbackResult is InvokeCallback for a callback that returns a value.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.invokeCallback
func InvokeCallbackResult[T any](r *GestureRecognizerBase, name string, callback func() T, debugReport ...func() string) T {
	r.traceCallback(name, debugReport)
	var result T
	foundation.Guard(r.binding.errorReporter, foundation.CallbackPanicked, "while handling a gesture",
		func() []any { return []any{"recognizer", foundation.DescribeIdentity(r.self), "callback", name} },
		func() { result = callback() })
	return result
}

func (r *GestureRecognizerBase) traceCallback(name string, debugReport []func() string) {
	if !foundation.DebugMode || !r.binding.Debug.PrintRecognizerCallbacksTrace {
		return
	}
	report := ""
	if len(debugReport) > 0 && debugReport[0] != nil {
		report = debugReport[0]()
	}
	// The 19 is the width of the prefix of the arena's diagnostics.
	prefix := ""
	if r.binding.Debug.PrintGestureArenaDiagnostics {
		prefix = strings.Repeat(" ", 19) + "❙ "
	}
	if report != "" {
		report = " " + report
	}
	r.binding.Debug.print(fmt.Sprintf("%s%s calling %s callback.%s", prefix, foundation.DiagnosticableString(r.self), name, report))
}

// DebugFillProperties adds the debug owner.
//
// Flutter: gestures/recognizer.dart GestureRecognizer.debugFillProperties
func (r *GestureRecognizerBase) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("debugOwner", r.DebugOwner,
		foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
}

// OneSequenceHooks are the methods of a OneSequenceGestureRecognizer that the
// subclass supplies or may override.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer
type OneSequenceHooks interface {
	GestureRecognizerHooks
	// HandleEvent is called with the events of the pointers the recognizer
	// tracks.
	//
	// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.handleEvent
	HandleEvent(event PointerEvent)
	// DidStopTrackingLastPointer is called when the recognizer stops tracking
	// the last pointer.
	//
	// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.didStopTrackingLastPointer
	DidStopTrackingLastPointer(pointer int)
	// StartTrackingPointer routes the events of the pointer to HandleEvent and
	// adds the recognizer to the arena of the pointer.
	//
	// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.startTrackingPointer
	StartTrackingPointer(pointer int, transform *ui.Matrix4)
	// Resolve accepts or rejects the recognizer in the arenas it is in.
	//
	// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.resolve
	Resolve(disposition GestureDisposition)
	// ResolvePointer accepts or rejects the recognizer in the arena of the
	// pointer.
	//
	// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.resolvePointer
	ResolvePointer(pointer int, disposition GestureDisposition)
}

// OneSequenceGestureRecognizer is a recognizer that tracks pointers from their
// down event to their up event, and so takes part in the arena of each pointer.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer
type OneSequenceGestureRecognizer struct {
	GestureRecognizerBase
	oneSeq OneSequenceHooks

	entries         map[int]GestureArenaEntry
	trackedPointers map[int]PointerRouteHandle
	team            *GestureArenaTeam
}

// InitOneSequence sets up the recognizer; self is the recognizer that embeds
// it.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.new
func (r *OneSequenceGestureRecognizer) InitOneSequence(binding *GestureBinding, self OneSequenceHooks, opts GestureRecognizerOpts) {
	r.Init(binding, self, opts)
	r.oneSeq = self
	r.entries = map[int]GestureArenaEntry{}
	r.trackedPointers = map[int]PointerRouteHandle{}
}

// AddAllowedPointer starts tracking the pointer.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.addAllowedPointer
func (r *OneSequenceGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	r.oneSeq.StartTrackingPointer(event.Pointer(), event.Transform())
}

// HandleNonAllowedPointer rejects the recognizer, so that it does not take part
// in the arena for the pointer.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.handleNonAllowedPointer
func (r *OneSequenceGestureRecognizer) HandleNonAllowedPointer(*PointerDownEvent) {
	r.oneSeq.Resolve(GestureRejected)
}

// AcceptGesture does nothing; a recognizer overrides it.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.acceptGesture
func (r *OneSequenceGestureRecognizer) AcceptGesture(int) {}

// RejectGesture does nothing; a recognizer overrides it.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.rejectGesture
func (r *OneSequenceGestureRecognizer) RejectGesture(int) {}

// Resolve accepts or rejects the recognizer in all the arenas it is in.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.resolve
func (r *OneSequenceGestureRecognizer) Resolve(disposition GestureDisposition) {
	local := make([]GestureArenaEntry, 0, len(r.entries))
	for _, e := range r.entries {
		local = append(local, e)
	}
	r.entries = map[int]GestureArenaEntry{}
	for _, e := range local {
		e.Resolve(disposition)
	}
}

// ResolvePointer accepts or rejects the recognizer in the arena of the pointer.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.resolvePointer
func (r *OneSequenceGestureRecognizer) ResolvePointer(pointer int, disposition GestureDisposition) {
	if e, ok := r.entries[pointer]; ok {
		delete(r.entries, pointer)
		e.Resolve(disposition)
	}
}

// Dispose rejects the recognizer in its arenas and stops tracking its pointers.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.dispose
func (r *OneSequenceGestureRecognizer) Dispose() {
	r.oneSeq.Resolve(GestureRejected)
	for pointer, handle := range r.trackedPointers {
		r.binding.PointerRouter.RemoveRoute(pointer, handle)
	}
	r.trackedPointers = map[int]PointerRouteHandle{}
	if foundation.DebugMode && len(r.entries) != 0 {
		panic(errors.New(tagRecognizerEntriesLeft, "the recognizer still has arena entries after dispose"))
	}
	r.GestureRecognizerBase.Dispose()
}

// Team is the team the recognizer is in, if any.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.team
func (r *OneSequenceGestureRecognizer) Team() *GestureArenaTeam { return r.team }

// SetTeam puts the recognizer in the team, which makes it join the team's
// combined member in each arena. It can be set once, before the recognizer
// tracks a pointer; otherwise a debug build panics with the tag
// RecognizerTeamSetTwice.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.team
func (r *OneSequenceGestureRecognizer) SetTeam(team *GestureArenaTeam) {
	if foundation.DebugMode && (team == nil || len(r.entries) != 0 || len(r.trackedPointers) != 0 || r.team != nil) {
		panic(errors.New(tagRecognizerTeamSetTwice,
			"the team must be set once, to a team, before any pointer is tracked"))
	}
	r.team = team
}

func (r *OneSequenceGestureRecognizer) addPointerToArena(pointer int) GestureArenaEntry {
	if r.team != nil {
		return r.team.Add(pointer, r.oneSeq)
	}
	return r.binding.GestureArena.Add(pointer, r.oneSeq)
}

// StartTrackingPointer routes the events of the pointer to HandleEvent and adds
// the recognizer to the arena of the pointer. The transform is applied to the
// events.
//
// Deviation: it first forgets the entries of the arenas that have resolved. Flutter keeps them
// until the recognizer resolves or is disposed (flutter/flutter#117356), and with a new pointer
// id for every press that is an entry, an arena and its members for each press of a recognizer
// that lives as long as the app (SPEC §12.10).
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.startTrackingPointer
func (r *OneSequenceGestureRecognizer) StartTrackingPointer(pointer int, transform *ui.Matrix4) {
	for p, e := range r.entries {
		if d, ok := e.(defunctEntry); ok && d.defunct() {
			delete(r.entries, p)
		}
	}
	r.trackedPointers[pointer] = r.binding.PointerRouter.AddRoute(pointer, r.oneSeq.HandleEvent, transform)
	r.entries[pointer] = r.addPointerToArena(pointer)
}

// StopTrackingPointer stops the routing of the pointer's events. When it was
// the last tracked pointer, DidStopTrackingLastPointer is called.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.stopTrackingPointer
func (r *OneSequenceGestureRecognizer) StopTrackingPointer(pointer int) {
	handle, ok := r.trackedPointers[pointer]
	if !ok {
		return
	}
	r.binding.PointerRouter.RemoveRoute(pointer, handle)
	delete(r.trackedPointers, pointer)
	if len(r.trackedPointers) == 0 {
		r.oneSeq.DidStopTrackingLastPointer(pointer)
	}
}

// IsTrackingPointer reports whether the recognizer tracks the pointer.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.isTrackingPointer
func (r *OneSequenceGestureRecognizer) IsTrackingPointer(pointer int) bool {
	_, ok := r.trackedPointers[pointer]
	return ok
}

// TrackedPointerCount is the number of pointers the recognizer tracks.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer._trackedPointers
func (r *OneSequenceGestureRecognizer) TrackedPointerCount() int { return len(r.trackedPointers) }

// StopTrackingIfPointerNoLongerDown stops tracking the pointer if the event
// ends it: an up, cancel or pan/zoom end event.
//
// Flutter: gestures/recognizer.dart OneSequenceGestureRecognizer.stopTrackingIfPointerNoLongerDown
func (r *OneSequenceGestureRecognizer) StopTrackingIfPointerNoLongerDown(event PointerEvent) {
	switch event.(type) {
	case *PointerUpEvent, *PointerCancelEvent, *PointerPanZoomEndEvent:
		r.StopTrackingPointer(event.Pointer())
	}
}

// GestureRecognizerState is the state of a PrimaryPointerGestureRecognizer.
//
// Flutter: gestures/recognizer.dart GestureRecognizerState
type GestureRecognizerState int

const (
	// GestureRecognizerStateUnspecified is the zero value; it is never a
	// state.
	GestureRecognizerStateUnspecified GestureRecognizerState = iota
	// GestureRecognizerStateReady means the recognizer is ready to start
	// recognizing a gesture.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizerState.ready
	GestureRecognizerStateReady
	// GestureRecognizerStatePossible means a pointer is down and the recognizer
	// is trying to decide whether it is a gesture.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizerState.possible
	GestureRecognizerStatePossible
	// GestureRecognizerStateDefunct means the pointer is no longer being
	// considered, as the recognizer lost or gave up.
	//
	// Flutter: gestures/recognizer.dart GestureRecognizerState.defunct
	GestureRecognizerStateDefunct
)

// Name returns the Dart name of the state.
//
// Flutter: gestures/recognizer.dart GestureRecognizerState
func (s GestureRecognizerState) Name() string {
	switch s {
	case GestureRecognizerStateReady:
		return "ready"
	case GestureRecognizerStatePossible:
		return "possible"
	case GestureRecognizerStateDefunct:
		return "defunct"
	}
	return "GestureRecognizerState(" + fmt.Sprint(int(s)) + ")"
}

type slopKind int

const (
	slopDefault slopKind = iota
	slopNone
	slopValue
)

// SlopTolerance is how far a pointer may move before a recognizer gives up: the
// default (the touch slop of the device settings), no limit, or a distance in
// logical pixels. The zero value is the default.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.preAcceptSlopTolerance
type SlopTolerance struct {
	kind  slopKind
	value float64
}

// SlopToleranceNone is no limit: Flutter's null.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.preAcceptSlopTolerance
var SlopToleranceNone = SlopTolerance{kind: slopNone}

// NewSlopTolerance is a limit of the distance, which must not be negative.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.new
func NewSlopTolerance(distance float64) SlopTolerance {
	return SlopTolerance{kind: slopValue, value: distance}
}

// PrimaryPointerHooks are the methods of a PrimaryPointerGestureRecognizer that
// the subclass supplies or may override.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer
type PrimaryPointerHooks interface {
	OneSequenceHooks
	// HandlePrimaryPointer is called with the events of the primary pointer.
	//
	// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.handlePrimaryPointer
	HandlePrimaryPointer(event PointerEvent)
	// DidExceedDeadline is called when the deadline passes with the pointer
	// still down.
	//
	// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.didExceedDeadline
	DidExceedDeadline()
	// DidExceedDeadlineWithEvent is called with the down event when the
	// deadline passes. The default calls DidExceedDeadline.
	//
	// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.didExceedDeadlineWithEvent
	DidExceedDeadlineWithEvent(event *PointerDownEvent)
}

// PrimaryPointerOpts are the parameters of InitPrimaryPointer.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.new
type PrimaryPointerOpts struct {
	GestureRecognizerOpts
	// Deadline is how long after the down event DidExceedDeadline is called;
	// None for no deadline.
	Deadline foundation.Opt[time.Duration]
	// PreAcceptSlopTolerance is how far the pointer may move before the
	// recognizer wins the arena, without it giving up. The default is the touch
	// slop.
	PreAcceptSlopTolerance SlopTolerance
	// PostAcceptSlopTolerance is how far the pointer may move after the
	// recognizer won. The default is the touch slop.
	PostAcceptSlopTolerance SlopTolerance
}

// PrimaryPointerGestureRecognizer is a recognizer that tracks one primary
// pointer (the first that goes down) and ignores the others while it does.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer
type PrimaryPointerGestureRecognizer struct {
	OneSequenceGestureRecognizer
	primary PrimaryPointerHooks

	deadline                foundation.Opt[time.Duration]
	preAcceptSlopTolerance  SlopTolerance
	postAcceptSlopTolerance SlopTolerance

	state           GestureRecognizerState
	primaryPointer  foundation.Opt[int]
	initialPosition foundation.Opt[OffsetPair]
	// gestureAccepted is whether this pointer is accepted by winning the arena
	// or as defined by a subclass calling AcceptGesture.
	gestureAccepted bool
	timer           *Timer
}

// InitPrimaryPointer sets up the recognizer; self is the recognizer that embeds
// it. A negative slop tolerance panics with the tag SlopToleranceNegative in
// debug builds.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.new
func (r *PrimaryPointerGestureRecognizer) InitPrimaryPointer(binding *GestureBinding, self PrimaryPointerHooks, opts PrimaryPointerOpts) {
	if foundation.DebugMode {
		for _, s := range []SlopTolerance{opts.PreAcceptSlopTolerance, opts.PostAcceptSlopTolerance} {
			if s.kind == slopValue && s.value < 0 {
				panic(errors.New(tagSlopToleranceNegative, "the slop tolerance must be unspecified, positive, or none",
					"value", s.value))
			}
		}
	}
	r.InitOneSequence(binding, self, opts.GestureRecognizerOpts)
	r.primary = self
	r.deadline = opts.Deadline
	r.preAcceptSlopTolerance = opts.PreAcceptSlopTolerance
	r.postAcceptSlopTolerance = opts.PostAcceptSlopTolerance
	r.state = GestureRecognizerStateReady
}

func (r *PrimaryPointerGestureRecognizer) defaultTouchSlop() float64 {
	if s, ok := r.GestureSettings.Get(); ok {
		return s.TouchSlop.OrElse(TouchSlop)
	}
	return TouchSlop
}

func (r *PrimaryPointerGestureRecognizer) resolveSlop(s SlopTolerance) foundation.Opt[float64] {
	switch s.kind {
	case slopNone:
		return foundation.None[float64]()
	case slopValue:
		return foundation.Some(s.value)
	}
	return foundation.Some(r.defaultTouchSlop())
}

// PreAcceptSlopTolerance is how far the pointer may move before the recognizer
// wins the arena without it giving up; None for any distance.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.preAcceptSlopTolerance
func (r *PrimaryPointerGestureRecognizer) PreAcceptSlopTolerance() foundation.Opt[float64] {
	return r.resolveSlop(r.preAcceptSlopTolerance)
}

// PostAcceptSlopTolerance is how far the pointer may move after the recognizer
// won; None for any distance.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.postAcceptSlopTolerance
func (r *PrimaryPointerGestureRecognizer) PostAcceptSlopTolerance() foundation.Opt[float64] {
	return r.resolveSlop(r.postAcceptSlopTolerance)
}

// Deadline is how long after the down event DidExceedDeadline is called.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.deadline
func (r *PrimaryPointerGestureRecognizer) Deadline() foundation.Opt[time.Duration] { return r.deadline }

// State is the state of the recognizer.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.state
func (r *PrimaryPointerGestureRecognizer) State() GestureRecognizerState { return r.state }

// PrimaryPointer is the pointer the recognizer follows; None while it is ready.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.primaryPointer
func (r *PrimaryPointerGestureRecognizer) PrimaryPointer() foundation.Opt[int] {
	return r.primaryPointer
}

// InitialPosition is where the primary pointer went down.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.initialPosition
func (r *PrimaryPointerGestureRecognizer) InitialPosition() foundation.Opt[OffsetPair] {
	return r.initialPosition
}

// AddAllowedPointer tracks the pointer, and makes it the primary pointer if the
// recognizer is ready.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.addAllowedPointer
func (r *PrimaryPointerGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	r.OneSequenceGestureRecognizer.AddAllowedPointer(event)
	if r.state == GestureRecognizerStateReady {
		r.state = GestureRecognizerStatePossible
		r.primaryPointer = foundation.Some(event.Pointer())
		r.initialPosition = foundation.Some(OffsetPair{Local: event.LocalPosition(), Global: event.Position()})
		if d, ok := r.deadline.Get(); ok {
			r.timer = r.binding.NewTimer(d, func() { r.primary.DidExceedDeadlineWithEvent(event) })
		}
	}
}

// HandleNonAllowedPointer rejects the recognizer unless it has already won.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.handleNonAllowedPointer
func (r *PrimaryPointerGestureRecognizer) HandleNonAllowedPointer(event *PointerDownEvent) {
	if !r.gestureAccepted {
		r.OneSequenceGestureRecognizer.HandleNonAllowedPointer(event)
	}
}

// HandleEvent gives the events of the primary pointer to HandlePrimaryPointer,
// and rejects the recognizer when the pointer moves farther than the slop
// tolerance.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.handleEvent
func (r *PrimaryPointerGestureRecognizer) HandleEvent(event PointerEvent) {
	if foundation.DebugMode && r.state == GestureRecognizerStateReady {
		panic(errors.New(tagRecognizerEventWhileReady, "an event arrived while the recognizer is ready"))
	}
	if primary, _ := r.primaryPointer.Get(); r.state == GestureRecognizerStatePossible && event.Pointer() == primary {
		pre, hasPre := r.PreAcceptSlopTolerance().Get()
		post, hasPost := r.PostAcceptSlopTolerance().Get()
		isPreAcceptSlopPastTolerance := !r.gestureAccepted && hasPre && r.getGlobalDistance(event) > pre
		isPostAcceptSlopPastTolerance := r.gestureAccepted && hasPost && r.getGlobalDistance(event) > post
		if _, isMove := event.(*PointerMoveEvent); isMove && (isPreAcceptSlopPastTolerance || isPostAcceptSlopPastTolerance) {
			r.primary.Resolve(GestureRejected)
			r.StopTrackingPointer(primary)
		} else {
			r.primary.HandlePrimaryPointer(event)
		}
	}
	r.StopTrackingIfPointerNoLongerDown(event)
}

// DidExceedDeadline is called when the deadline passes. A recognizer with a
// deadline overrides it; the default panics with the tag
// RecognizerNoDeadline in debug builds, if there is none.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.didExceedDeadline
func (r *PrimaryPointerGestureRecognizer) DidExceedDeadline() {
	if foundation.DebugMode && r.deadline.IsSome() {
		panic(errors.New(tagRecognizerNoDeadline, "a recognizer with a deadline must override DidExceedDeadline"))
	}
}

// DidExceedDeadlineWithEvent calls DidExceedDeadline.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.didExceedDeadlineWithEvent
func (r *PrimaryPointerGestureRecognizer) DidExceedDeadlineWithEvent(*PointerDownEvent) {
	r.primary.DidExceedDeadline()
}

// AcceptGesture marks the gesture as accepted and stops the deadline timer.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.acceptGesture
func (r *PrimaryPointerGestureRecognizer) AcceptGesture(pointer int) {
	if p, _ := r.primaryPointer.Get(); pointer == p {
		r.stopTimer()
		r.gestureAccepted = true
	}
}

// RejectGesture ends the gesture if it was possible.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.rejectGesture
func (r *PrimaryPointerGestureRecognizer) RejectGesture(pointer int) {
	if p, _ := r.primaryPointer.Get(); pointer == p && r.state == GestureRecognizerStatePossible {
		r.stopTimer()
		r.state = GestureRecognizerStateDefunct
	}
}

// DidStopTrackingLastPointer makes the recognizer ready again.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.didStopTrackingLastPointer
func (r *PrimaryPointerGestureRecognizer) DidStopTrackingLastPointer(int) {
	if foundation.DebugMode && r.state == GestureRecognizerStateReady {
		panic(errors.New(tagRecognizerStoppedWhileReady, "the last pointer stopped while the recognizer is ready"))
	}
	r.stopTimer()
	r.state = GestureRecognizerStateReady
	r.initialPosition = foundation.None[OffsetPair]()
	r.gestureAccepted = false
}

// Dispose stops the timer.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.dispose
func (r *PrimaryPointerGestureRecognizer) Dispose() {
	r.stopTimer()
	r.OneSequenceGestureRecognizer.Dispose()
}

func (r *PrimaryPointerGestureRecognizer) stopTimer() {
	if r.timer != nil {
		r.timer.Cancel()
		r.timer = nil
	}
}

func (r *PrimaryPointerGestureRecognizer) getGlobalDistance(event PointerEvent) float64 {
	return event.Position().Sub(r.initialPosition.Value().Global).Distance()
}

// DebugFillProperties adds the debug owner and the state.
//
// Flutter: gestures/recognizer.dart PrimaryPointerGestureRecognizer.debugFillProperties
func (r *PrimaryPointerGestureRecognizer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	r.OneSequenceGestureRecognizer.DebugFillProperties(p)
	p.Add(foundation.NewEnumProperty("state", foundation.Some(r.state)))
}

// OffsetPair is a pair of offsets, the same point in the local and the global
// coordinates.
//
// Flutter: gestures/recognizer.dart OffsetPair
type OffsetPair struct {
	// Local is the offset in local coordinates.
	Local ui.Offset
	// Global is the offset in global coordinates.
	Global ui.Offset
}

// OffsetPairZero is the pair of zero offsets.
//
// Flutter: gestures/recognizer.dart OffsetPair.zero
var OffsetPairZero = OffsetPair{}

// NewOffsetPairFromEventPosition is the position of the event in both
// coordinates.
//
// Flutter: gestures/recognizer.dart OffsetPair.fromEventPosition
func NewOffsetPairFromEventPosition(event PointerEvent) OffsetPair {
	return OffsetPair{Local: event.LocalPosition(), Global: event.Position()}
}

// NewOffsetPairFromEventDelta is the delta of the event in both coordinates.
//
// Flutter: gestures/recognizer.dart OffsetPair.fromEventDelta
func NewOffsetPairFromEventDelta(event PointerEvent) OffsetPair {
	return OffsetPair{Local: event.LocalDelta(), Global: event.Delta()}
}

// Add returns the sum of the pairs.
//
// Flutter: gestures/recognizer.dart OffsetPair.operator+
func (p OffsetPair) Add(other OffsetPair) OffsetPair {
	return OffsetPair{Local: p.Local.Add(other.Local), Global: p.Global.Add(other.Global)}
}

// Sub returns the difference of the pairs.
//
// Flutter: gestures/recognizer.dart OffsetPair.operator-
func (p OffsetPair) Sub(other OffsetPair) OffsetPair {
	return OffsetPair{Local: p.Local.Sub(other.Local), Global: p.Global.Sub(other.Global)}
}

// String describes the pair.
//
// Flutter: gestures/recognizer.dart OffsetPair.toString
func (p OffsetPair) String() string {
	return fmt.Sprintf("OffsetPair(local: %s, global: %s)", p.Local, p.Global)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagRecognizerEntriesLeft       = "RecognizerEntriesLeft"
	tagRecognizerEventWhileReady   = "RecognizerEventWhileReady"
	tagRecognizerNoDeadline        = "RecognizerNoDeadline"
	tagRecognizerPointerUnknown    = "RecognizerPointerUnknown"
	tagRecognizerStoppedWhileReady = "RecognizerStoppedWhileReady"
	tagRecognizerTeamSetTwice      = "RecognizerTeamSetTwice"
	tagSlopToleranceNegative       = "SlopToleranceNegative"
)
