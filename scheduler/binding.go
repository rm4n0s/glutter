package scheduler

import (
	"container/heap"
	"math"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// FrameCallback is a function that is called once per frame, with the time of
// the frame since an arbitrary point (it counts the time dilation).
//
// Flutter: scheduler/binding.dart FrameCallback
type FrameCallback func(timeStamp time.Duration)

// SchedulingStrategy decides whether a task of the given priority runs now. The
// default is DefaultSchedulingStrategy.
//
// Flutter: scheduler/binding.dart SchedulingStrategy
type SchedulingStrategy func(priority int, scheduler *SchedulerBinding) bool

// DefaultSchedulingStrategy runs a task only when it has at least the priority
// of animation while an animation callback is pending, and any task otherwise.
//
// Flutter: scheduler/binding.dart defaultSchedulingStrategy
func DefaultSchedulingStrategy(priority int, scheduler *SchedulerBinding) bool {
	if scheduler.TransientCallbackCount() > 0 {
		return priority >= PriorityAnimation.Value()
	}
	return true
}

// SchedulerPhase is where the binding is in the life of a frame.
//
// Flutter: scheduler/binding.dart SchedulerPhase
type SchedulerPhase int

// The values of SchedulerPhase follow Flutter's declaration order.
const (
	// SchedulerPhaseUnspecified is the zero value; a binding is never in it.
	//
	// Flutter: no counterpart
	SchedulerPhaseUnspecified SchedulerPhase = iota
	// SchedulerPhaseIdle means no frame is being built. Tasks and microtasks
	// run, and so do input and timers.
	//
	// Flutter: scheduler/binding.dart SchedulerPhase.idle
	SchedulerPhaseIdle
	// SchedulerPhaseTransientCallbacks is the start of a frame: the callbacks
	// that update animations run (HandleBeginFrame).
	//
	// Flutter: scheduler/binding.dart SchedulerPhase.transientCallbacks
	SchedulerPhaseTransientCallbacks
	// SchedulerPhaseMidFrameMicrotasks is the time between the transient and
	// the persistent callbacks.
	//
	// Flutter: scheduler/binding.dart SchedulerPhase.midFrameMicrotasks
	SchedulerPhaseMidFrameMicrotasks
	// SchedulerPhasePersistentCallbacks is where the frame is built: build,
	// layout and paint (HandleDrawFrame).
	//
	// Flutter: scheduler/binding.dart SchedulerPhase.persistentCallbacks
	SchedulerPhasePersistentCallbacks
	// SchedulerPhasePostFrameCallbacks is the end of a frame: the callbacks
	// that run once after it.
	//
	// Flutter: scheduler/binding.dart SchedulerPhase.postFrameCallbacks
	SchedulerPhasePostFrameCallbacks
)

var schedulerPhaseNames = [...]string{"unspecified", "idle", "transientCallbacks", "midFrameMicrotasks", "persistentCallbacks", "postFrameCallbacks"}

// Name returns the name Dart's Enum.name gives the value, such as "idle".
func (p SchedulerPhase) Name() string {
	if p >= 0 && int(p) < len(schedulerPhaseNames) {
		return schedulerPhaseNames[p]
	}
	return "SchedulerPhase(" + itoa(int(p)) + ")"
}

// String returns the name Dart prints for the value, such as
// "SchedulerPhase.idle".
func (p SchedulerPhase) String() string {
	if p >= 0 && int(p) < len(schedulerPhaseNames) {
		return "SchedulerPhase." + schedulerPhaseNames[p]
	}
	return p.Name()
}

// taskEntry is a task waiting for the event loop. sequence keeps tasks of the
// same priority in the order they were scheduled.
type taskEntry struct {
	run      func()
	priority int
	sequence uint64
}

// taskQueue is a heap of tasks, the most important first.
type taskQueue []*taskEntry

func (q taskQueue) Len() int { return len(q) }
func (q taskQueue) Less(i, j int) bool {
	if q[i].priority != q[j].priority {
		return q[i].priority > q[j].priority
	}
	return q[i].sequence < q[j].sequence
}
func (q taskQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *taskQueue) Push(x any)   { *q = append(*q, x.(*taskEntry)) }
func (q *taskQueue) Pop() any {
	old := *q
	n := len(old)
	e := old[n-1]
	old[n-1] = nil
	*q = old[:n-1]
	return e
}

// frameCallbackEntry is a transient frame callback.
type frameCallbackEntry struct {
	id       int
	callback FrameCallback
}

// SchedulerBinding schedules the work of the framework around the frames of the
// platform: tasks between frames, the callbacks of a frame in three groups, and
// the flow of a frame through its phases.
//
// It listens to the dispatcher's OnBeginFrame and OnDrawFrame (when nothing
// else does yet), and asks it for frames. The bindings of the framework embed
// it. There is one per app and it belongs to the UI thread.
//
// Flutter: scheduler/binding.dart SchedulerBinding
type SchedulerBinding struct {
	d *ui.PlatformDispatcher

	// SchedulingStrategy decides which tasks run now. The default is
	// DefaultSchedulingStrategy.
	//
	// Flutter: scheduler/binding.dart SchedulerBinding.schedulingStrategy
	SchedulingStrategy SchedulingStrategy

	// ErrorReporter receives the panics of tasks and frame callbacks, which are
	// recovered so that the others still run. Without one, a panic goes on and
	// ends the frame. The binding that owns the scheduler sets it (SPEC §14.2).
	//
	// Flutter: foundation/assertions.dart FlutterError.onError
	ErrorReporter foundation.ErrorReporter
	// FramesGate, when set, is a further condition for FramesEnabled: the
	// widgets binding keeps frames off until the root widget is attached.
	//
	// Flutter: widgets/binding.dart WidgetsBinding.framesEnabled
	FramesGate func() bool

	lifecycle     ui.AppLifecycleState
	framesEnabled bool
	timeDilation  float64

	tasks                     taskQueue
	taskSequence              uint64
	hasRequestedEventLoopCall bool
	lockCount                 int
	unlockedCallbacks         []func()

	nextFrameCallbackID int
	transientCallbacks  []frameCallbackEntry
	removedIDs          map[int]bool
	persistentCallbacks []FrameCallback
	timingsCallbacks    []timingsEntry
	nextTimingsID       int
	postFrameCallbacks  []FrameCallback
	nextFrameWaiters    []chan struct{}
	hasScheduledFrame   bool
	phase               SchedulerPhase

	firstRawTimeStampInEpoch foundation.Opt[time.Duration]
	epochStart               time.Duration
	lastRawTimeStamp         time.Duration
	currentFrameTimeStamp    foundation.Opt[time.Duration]
}

// NewSchedulerBinding creates the binding for a dispatcher. It reads the
// lifecycle state the dispatcher already has: a state that is hidden, paused or
// detached stops frames from the start.
//
// Flutter: scheduler/binding.dart SchedulerBinding.initInstances
func NewSchedulerBinding(d *ui.PlatformDispatcher) *SchedulerBinding {
	b := &SchedulerBinding{
		d:                  d,
		SchedulingStrategy: DefaultSchedulingStrategy,
		framesEnabled:      true,
		timeDilation:       1,
		removedIDs:         map[int]bool{},
		phase:              SchedulerPhaseIdle,
	}
	if state := d.AppLifecycleState(); state != ui.AppLifecycleStateUnspecified {
		b.HandleAppLifecycleStateChanged(state)
	}
	return b
}

// PlatformDispatcher is the dispatcher the binding schedules frames with.
//
// Flutter: foundation/binding.dart BindingBase.platformDispatcher
func (b *SchedulerBinding) PlatformDispatcher() *ui.PlatformDispatcher { return b.d }

// TimeDilation is how much slower the frame times run than the real time; 1 by
// default. A value of 2 slows animations to half their speed.
//
// Flutter: scheduler/binding.dart timeDilation
func (b *SchedulerBinding) TimeDilation() float64 { return b.timeDilation }

// SetTimeDilation sets the time dilation, which must be larger than zero (or it
// panics with the tag TimeDilationInvalid in debug builds). The epoch restarts
// first, so that the frame times stay continuous.
//
// Flutter: scheduler/binding.dart timeDilation=
func (b *SchedulerBinding) SetTimeDilation(value float64) {
	if foundation.DebugMode && !(value > 0) {
		panic(errors.New("TimeDilationInvalid", "the time dilation must be larger than zero", "value", value))
	}
	if b.timeDilation == value {
		return
	}
	// The epoch starts with the current time dilation.
	b.ResetEpoch()
	b.timeDilation = value
}

// LifecycleState is the last lifecycle state the binding was told about, or
// AppLifecycleStateUnspecified.
//
// Flutter: scheduler/binding.dart SchedulerBinding.lifecycleState
func (b *SchedulerBinding) LifecycleState() ui.AppLifecycleState { return b.lifecycle }

// ResetInternalState puts the binding back to its state at the start: no
// lifecycle state, and frames enabled. Tests use it.
//
// Flutter: scheduler/binding.dart SchedulerBinding.resetInternalState
func (b *SchedulerBinding) ResetInternalState() {
	b.lifecycle = ui.AppLifecycleStateUnspecified
	b.framesEnabled = true
}

// HandleAppLifecycleStateChanged tells the binding the app lifecycle changed.
// Frames stop while the app is hidden, paused or detached, and start again (a
// frame is scheduled) when it is resumed or inactive. The app's binding calls
// this; an app does not.
//
// Flutter: scheduler/binding.dart SchedulerBinding.handleAppLifecycleStateChanged
func (b *SchedulerBinding) HandleAppLifecycleStateChanged(state ui.AppLifecycleState) {
	if b.lifecycle == state {
		return
	}
	b.lifecycle = state
	switch state {
	case ui.AppLifecycleStateResumed, ui.AppLifecycleStateInactive:
		b.setFramesEnabledState(true)
	case ui.AppLifecycleStateHidden, ui.AppLifecycleStatePaused, ui.AppLifecycleStateDetached:
		b.setFramesEnabledState(false)
	}
}

// Locked reports whether input and tasks are held back (see LockEvents).
//
// Flutter: foundation/binding.dart BindingBase.locked
func (b *SchedulerBinding) Locked() bool { return b.lockCount > 0 }

// LockEvents stops tasks (and, for the bindings that use Locked, input) from
// running until the returned function is called. Calls nest. When the last lock
// is released, the callbacks added with AddUnlockedCallback run and the waiting
// tasks start again. Call the returned function once.
//
// Deviation: Flutter's lockEvents takes a callback that returns a Future and
// unlocks when it completes (SPEC §12.10).
//
// Flutter: foundation/binding.dart BindingBase.lockEvents
func (b *SchedulerBinding) LockEvents() (unlock func()) {
	b.lockCount++
	released := false
	return func() {
		if released {
			return
		}
		released = true
		b.lockCount--
		if !b.Locked() {
			b.unlocked()
		}
	}
}

// AddUnlockedCallback adds a function that runs each time the last lock is
// released. Bindings that hold events while locked use it to deliver them.
//
// Flutter: foundation/binding.dart BindingBase.unlocked
func (b *SchedulerBinding) AddUnlockedCallback(fn func()) {
	b.unlockedCallbacks = append(b.unlockedCallbacks, fn)
}

func (b *SchedulerBinding) unlocked() {
	if len(b.tasks) > 0 {
		b.ensureEventLoopCallback()
	}
	for _, fn := range slices.Clone(b.unlockedCallbacks) {
		fn()
	}
}

// ScheduleTaskOpts are the optional parameters of ScheduleTask.
//
// Flutter: scheduler/binding.dart SchedulerBinding.scheduleTask
type ScheduleTaskOpts struct {
	// DebugLabel names the task in debug output.
	DebugLabel string
}

// ScheduleTask schedules the task to run between frames, when the scheduling
// strategy allows its priority. Tasks of a higher priority run first; those of
// the same priority run in the order they were scheduled. The returned channel
// receives the task's result when it ran (and is never written if the task
// panics).
//
// Flutter: scheduler/binding.dart SchedulerBinding.scheduleTask
func ScheduleTask[T any](b *SchedulerBinding, task func() T, priority Priority, opts ...ScheduleTaskOpts) <-chan T {
	_ = firstOpts(opts)
	result := make(chan T, 1)
	isFirstTask := len(b.tasks) == 0
	b.taskSequence++
	heap.Push(&b.tasks, &taskEntry{
		run:      func() { result <- task() },
		priority: priority.Value(),
		sequence: b.taskSequence,
	})
	if isFirstTask && !b.Locked() {
		b.ensureEventLoopCallback()
	}
	return result
}

// ensureEventLoopCallback has the event loop call runTasks, unless it already
// was asked to.
func (b *SchedulerBinding) ensureEventLoopCallback() {
	if b.hasRequestedEventLoopCall {
		return
	}
	b.hasRequestedEventLoopCall = true
	b.d.Post(b.runTasks)
}

func (b *SchedulerBinding) runTasks() {
	b.hasRequestedEventLoopCall = false
	if b.HandleEventLoopCallback() {
		b.ensureEventLoopCallback() // runs the next task when there is time
	}
}

// HandleEventLoopCallback runs the most important task if the scheduling
// strategy allows it. It reports whether there is more to do: tasks waiting, or
// one that the strategy did not allow yet. The event loop calls it; tests call
// it too.
//
// A task that panics is reported to the ErrorReporter (CallbackPanicked, "during
// a task callback") and the event loop goes on.
//
// Flutter: scheduler/binding.dart SchedulerBinding.handleEventLoopCallback
func (b *SchedulerBinding) HandleEventLoopCallback() bool {
	if len(b.tasks) == 0 || b.Locked() {
		return false
	}
	entry := b.tasks[0]
	if b.SchedulingStrategy(entry.priority, b) {
		heap.Pop(&b.tasks)
		foundation.Guard(b.ErrorReporter, foundation.CallbackPanicked, "during a task callback", nil, entry.run)
		return len(b.tasks) > 0
	}
	return true
}

// TransientCallbackCount is how many transient frame callbacks wait for the
// next frame.
//
// Flutter: scheduler/binding.dart SchedulerBinding.transientCallbackCount
func (b *SchedulerBinding) TransientCallbackCount() int { return len(b.transientCallbacks) }

// ScheduleFrameCallbackOpts are the optional parameters of
// ScheduleFrameCallback.
//
// Flutter: scheduler/binding.dart SchedulerBinding.scheduleFrameCallback
type ScheduleFrameCallbackOpts struct {
	// Rescheduling must be set to true only by a callback that registers
	// itself again from inside itself. Flutter uses it to keep the stack of the
	// first registration; Go keeps no stacks, so it has no effect.
	Rescheduling bool
	// NoNewFrame stops the call from scheduling a frame; Flutter's
	// scheduleNewFrame is true by default.
	NoNewFrame bool
}

// ScheduleFrameCallback schedules the callback to run once at the start of the
// next frame (a transient callback: animations). It schedules a frame unless
// NoNewFrame is set. The returned ID cancels it with CancelFrameCallbackWithID.
//
// Callbacks scheduled during a frame's callbacks run in the next frame.
//
// Flutter: scheduler/binding.dart SchedulerBinding.scheduleFrameCallback
func (b *SchedulerBinding) ScheduleFrameCallback(callback FrameCallback, opts ...ScheduleFrameCallbackOpts) int {
	o := firstOpts(opts)
	if !o.NoNewFrame {
		b.ScheduleFrame()
	}
	b.nextFrameCallbackID++
	b.transientCallbacks = append(b.transientCallbacks, frameCallbackEntry{b.nextFrameCallbackID, callback})
	return b.nextFrameCallbackID
}

// CancelFrameCallbackWithID cancels a callback scheduled with
// ScheduleFrameCallback, also while the frame that would run it is already
// running its callbacks. The ID must be larger than zero.
//
// Flutter: scheduler/binding.dart SchedulerBinding.cancelFrameCallbackWithId
func (b *SchedulerBinding) CancelFrameCallbackWithID(id int) {
	if foundation.DebugMode && id <= 0 {
		panic(errors.New("FrameCallbackIDInvalid", "a frame callback ID is larger than zero", "id", id))
	}
	b.transientCallbacks = slices.DeleteFunc(b.transientCallbacks, func(e frameCallbackEntry) bool { return e.id == id })
	b.removedIDs[id] = true
}

// DebugAssertNoTransientCallbacks panics with the tag TransientCallbacksPending
// in debug builds when transient callbacks are still waiting. It returns true.
//
// Flutter lists the stack of each callback that is still waiting; Go keeps no such
// stacks, so the count is all there is.
//
// Flutter: scheduler/binding.dart SchedulerBinding.debugAssertNoTransientCallbacks
func (b *SchedulerBinding) DebugAssertNoTransientCallbacks(reason string) bool {
	if foundation.DebugMode && b.TransientCallbackCount() > 0 {
		panic(errors.New("TransientCallbacksPending", reason, "count", b.TransientCallbackCount()))
	}
	return true
}

// DebugAssertNoTimeDilation panics with the tag TimeDilationActive in debug
// builds when the time dilation is not 1. It returns true.
//
// Flutter: scheduler/binding.dart SchedulerBinding.debugAssertNoTimeDilation
func (b *SchedulerBinding) DebugAssertNoTimeDilation(reason string) bool {
	if foundation.DebugMode && b.timeDilation != 1 {
		panic(errors.New("TimeDilationActive", reason, "timeDilation", b.timeDilation))
	}
	return true
}

// AddPersistentFrameCallback adds a callback that runs at the end of every
// frame, for the life of the app: it builds, lays out and paints. It cannot be
// removed. A callback added during a frame first runs in the next one.
//
// Flutter: scheduler/binding.dart SchedulerBinding.addPersistentFrameCallback
func (b *SchedulerBinding) AddPersistentFrameCallback(callback FrameCallback) {
	b.persistentCallbacks = append(b.persistentCallbacks, callback)
}

// AddPostFrameCallback adds a callback that runs once after the next frame
// (it does not schedule one). A callback added by a post-frame callback runs
// after the frame after.
//
// Flutter: scheduler/binding.dart SchedulerBinding.addPostFrameCallback
func (b *SchedulerBinding) AddPostFrameCallback(callback FrameCallback) {
	b.postFrameCallbacks = append(b.postFrameCallbacks, callback)
}

// EndOfFrame returns a channel that is closed after the next frame has been
// drawn (in the post-frame phase). It schedules a frame if none is being built.
//
// Flutter: scheduler/binding.dart SchedulerBinding.endOfFrame
func (b *SchedulerBinding) EndOfFrame() <-chan struct{} {
	if len(b.nextFrameWaiters) == 0 {
		if b.phase == SchedulerPhaseIdle {
			b.ScheduleFrame()
		}
		b.AddPostFrameCallback(func(time.Duration) {
			waiters := b.nextFrameWaiters
			b.nextFrameWaiters = nil
			for _, w := range waiters {
				close(w)
			}
		})
	}
	w := make(chan struct{})
	b.nextFrameWaiters = append(b.nextFrameWaiters, w)
	return w
}

// HasScheduledFrame reports whether a frame was requested and has not begun.
//
// Flutter: scheduler/binding.dart SchedulerBinding.hasScheduledFrame
func (b *SchedulerBinding) HasScheduledFrame() bool { return b.hasScheduledFrame }

// SchedulerPhase is where the binding is in the life of a frame.
//
// Flutter: scheduler/binding.dart SchedulerBinding.schedulerPhase
func (b *SchedulerBinding) SchedulerPhase() SchedulerPhase { return b.phase }

// FramesEnabled reports whether frames are produced. They are not while the app
// is hidden, paused or detached.
//
// Flutter: scheduler/binding.dart SchedulerBinding.framesEnabled
func (b *SchedulerBinding) FramesEnabled() bool {
	return b.framesEnabled && (b.FramesGate == nil || b.FramesGate())
}

func (b *SchedulerBinding) setFramesEnabledState(enabled bool) {
	if b.framesEnabled == enabled {
		return
	}
	b.framesEnabled = enabled
	if enabled {
		b.ScheduleFrame()
	}
}

// EnsureFrameCallbacksRegistered makes the binding the listener of the
// dispatcher's OnBeginFrame and OnDrawFrame, unless something else already is.
// ScheduleFrame calls it.
//
// Flutter: scheduler/binding.dart SchedulerBinding.ensureFrameCallbacksRegistered
func (b *SchedulerBinding) EnsureFrameCallbacksRegistered() {
	if b.d.OnBeginFrame == nil {
		b.d.OnBeginFrame = func(rawTimeStamp time.Duration) { b.HandleBeginFrame(foundation.Some(rawTimeStamp)) }
	}
	if b.d.OnDrawFrame == nil {
		b.d.OnDrawFrame = b.HandleDrawFrame
	}
}

// EnsureVisualUpdate schedules a frame if the binding is idle or in the
// post-frame phase, and does nothing during the frame itself, which is about
// to draw anyway. Call it when something changes that should show.
//
// Flutter: scheduler/binding.dart SchedulerBinding.ensureVisualUpdate
func (b *SchedulerBinding) EnsureVisualUpdate() {
	switch b.phase {
	case SchedulerPhaseIdle, SchedulerPhasePostFrameCallbacks:
		b.ScheduleFrame()
	}
}

// ScheduleFrame asks the platform for a frame, once: asking again before it
// begins does nothing, and nothing happens while frames are disabled.
//
// Flutter: scheduler/binding.dart SchedulerBinding.scheduleFrame
func (b *SchedulerBinding) ScheduleFrame() {
	if b.hasScheduledFrame || !b.FramesEnabled() {
		return
	}
	b.EnsureFrameCallbacksRegistered()
	b.d.ScheduleFrame()
	b.hasScheduledFrame = true
}

// ScheduleForcedFrame is ScheduleFrame that also works while frames are
// disabled.
//
// Flutter: scheduler/binding.dart SchedulerBinding.scheduleForcedFrame
func (b *SchedulerBinding) ScheduleForcedFrame() {
	if b.hasScheduledFrame {
		return
	}
	b.EnsureFrameCallbacksRegistered()
	b.d.ScheduleFrame()
	b.hasScheduledFrame = true
}

// ResetEpoch makes the next frame time start from the last one, so that a gap
// (a change of the time dilation, a pause) does not show as a jump.
//
// Flutter: scheduler/binding.dart SchedulerBinding.resetEpoch
func (b *SchedulerBinding) ResetEpoch() {
	b.epochStart = b.adjustForEpoch(b.lastRawTimeStamp)
	b.firstRawTimeStampInEpoch = foundation.None[time.Duration]()
}

func (b *SchedulerBinding) adjustForEpoch(rawTimeStamp time.Duration) time.Duration {
	var sinceEpoch time.Duration
	if first, ok := b.firstRawTimeStampInEpoch.Get(); ok {
		sinceEpoch = rawTimeStamp - first
	}
	micros := math.Round(float64(sinceEpoch.Microseconds()) / b.timeDilation)
	return time.Duration(int64(micros))*time.Microsecond + b.epochStart
}

// CurrentFrameTimeStamp is the time of the frame being built, since the epoch
// and with the time dilation. It panics with the tag NoFrameInProgress outside
// a frame.
//
// Flutter: scheduler/binding.dart SchedulerBinding.currentFrameTimeStamp
func (b *SchedulerBinding) CurrentFrameTimeStamp() time.Duration {
	ts, ok := b.currentFrameTimeStamp.Get()
	if !ok {
		panic(errors.New("NoFrameInProgress", "the frame time stamp is only available while a frame is built"))
	}
	return ts
}

// CurrentSystemFrameTimeStamp is the raw time of the last frame, as the platform
// gave it, without epoch or time dilation.
//
// Flutter: scheduler/binding.dart SchedulerBinding.currentSystemFrameTimeStamp
func (b *SchedulerBinding) CurrentSystemFrameTimeStamp() time.Duration { return b.lastRawTimeStamp }

// HandleBeginFrame starts a frame: the transient callbacks run, with the frame
// time. The dispatcher's OnBeginFrame calls it; tests call it too. None as the
// raw time stands for a frame without a platform time (the time of the last
// one is used).
//
// The binding must be idle. Callbacks scheduled by the callbacks run in the
// next frame; one canceled by an earlier callback of the same frame does not
// run.
//
// A callback that panics is reported to the ErrorReporter (CallbackPanicked,
// "during a scheduler callback") and the others still run.
//
// Flutter: scheduler/binding.dart SchedulerBinding.handleBeginFrame
func (b *SchedulerBinding) HandleBeginFrame(rawTimeStamp foundation.Opt[time.Duration]) {
	if !b.firstRawTimeStampInEpoch.IsSome() {
		b.firstRawTimeStampInEpoch = rawTimeStamp
	}
	raw, hasRaw := rawTimeStamp.Get()
	if !hasRaw {
		raw = b.lastRawTimeStamp
	}
	b.currentFrameTimeStamp = foundation.Some(b.adjustForEpoch(raw))
	if hasRaw {
		b.lastRawTimeStamp = raw
	}
	if foundation.DebugMode && b.phase != SchedulerPhaseIdle {
		panic(errors.New("FrameAlreadyBegun", "HandleBeginFrame was called while a frame is being built", "phase", b.phase.Name()))
	}
	b.hasScheduledFrame = false
	defer foundation.StartTraceRegion("Animate").End()
	defer func() { b.phase = SchedulerPhaseMidFrameMicrotasks }()
	b.phase = SchedulerPhaseTransientCallbacks
	callbacks := b.transientCallbacks
	b.transientCallbacks = nil
	for _, entry := range callbacks {
		if !b.removedIDs[entry.id] {
			b.invokeFrameCallback(entry.callback, b.currentFrameTimeStamp.Value())
		}
	}
	clear(b.removedIDs)
}

// HandleDrawFrame ends a frame: the persistent callbacks run, then the
// post-frame callbacks, and the binding is idle again. The dispatcher's
// OnDrawFrame calls it, after HandleBeginFrame.
//
// A callback that panics is reported to the ErrorReporter (CallbackPanicked,
// "during a scheduler callback") and the others still run.
//
// Flutter: scheduler/binding.dart SchedulerBinding.handleDrawFrame
func (b *SchedulerBinding) HandleDrawFrame() {
	if foundation.DebugMode && b.phase != SchedulerPhaseMidFrameMicrotasks {
		panic(errors.New("FrameNotBegun", "HandleDrawFrame was called without HandleBeginFrame", "phase", b.phase.Name()))
	}
	defer foundation.StartTraceRegion("Frame").End()
	defer func() {
		b.phase = SchedulerPhaseIdle
		b.currentFrameTimeStamp = foundation.None[time.Duration]()
	}()
	ts := b.currentFrameTimeStamp.Value()
	b.phase = SchedulerPhasePersistentCallbacks
	for _, callback := range slices.Clone(b.persistentCallbacks) {
		b.invokeFrameCallback(callback, ts)
	}
	b.phase = SchedulerPhasePostFrameCallbacks
	postFrame := b.postFrameCallbacks
	b.postFrameCallbacks = nil
	for _, callback := range postFrame {
		b.invokeFrameCallback(callback, ts)
	}
}

// invokeFrameCallback calls a frame callback and reports its panic.
//
// Flutter: scheduler/binding.dart SchedulerBinding._invokeFrameCallback
func (b *SchedulerBinding) invokeFrameCallback(callback FrameCallback, ts time.Duration) {
	foundation.Guard(b.ErrorReporter, foundation.CallbackPanicked, "during a scheduler callback", nil, func() { callback(ts) })
}
