package scheduler

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/scheduler/{binding,scheduler,animation,time_dilation}_test.dart.

// testStrategy is Flutter's TestStrategy.
type testStrategy struct{ allowedPriority int }

func (s *testStrategy) shouldRunTaskWithPriority(priority int, _ *SchedulerBinding) bool {
	return priority >= s.allowedPriority
}

func TestTasksAreExecutedInTheRightOrder(t *testing.T) {
	tt := newTester()
	strategy := &testStrategy{allowedPriority: 10000}
	tt.SchedulingStrategy = strategy.shouldRunTaskWithPriority
	var executed []int
	scheduleAddingTask := func(x int) {
		ScheduleTask(tt.SchedulerBinding, func() struct{} { executed = append(executed, x); return struct{}{} }, PriorityIdle.Add(x))
	}
	for _, x := range []int{2, 23, 23, 11, 0, 80, 3} {
		scheduleAddingTask(x)
	}
	handle := func(times int, want bool) {
		t.Helper()
		for i := 0; i < times; i++ {
			if got := tt.HandleEventLoopCallback(); got != want {
				t.Fatalf("HandleEventLoopCallback = %v, want %v", got, want)
			}
		}
	}
	steps := []struct {
		allowed int
		calls   int
		want    []int
	}{
		{100, 3, nil},
		{50, 3, []int{80}},
		{20, 3, []int{23, 23}},
		{20, 3, []int{99, 97}}, // after scheduling 99, 19, 5 and 97 below
		{10, 3, []int{19, 11}},
		{1, 4, []int{5, 3, 2}},
	}
	for i, s := range steps {
		if i == 3 {
			for _, x := range []int{99, 19, 5, 97} {
				scheduleAddingTask(x)
			}
		}
		strategy.allowedPriority = s.allowed
		executed = nil
		handle(s.calls, true)
		if !reflect.DeepEqual(executed, s.want) {
			t.Errorf("step %d (allowed %d): executed %v, want %v", i, s.allowed, executed, s.want)
		}
	}
	strategy.allowedPriority = 0
	executed = nil
	handle(1, false)
	if !reflect.DeepEqual(executed, []int{0}) {
		t.Errorf("last task: executed %v, want [0]", executed)
	}
}

func TestTasksOfTheSamePriorityRunInOrder(t *testing.T) {
	tt := newTester()
	var order []string
	for _, name := range []string{"a", "b", "c", "d"} {
		ScheduleTask(tt.SchedulerBinding, func() string { order = append(order, name); return name }, PriorityAnimation)
	}
	for tt.HandleEventLoopCallback() {
	}
	if want := []string{"a", "b", "c", "d"}; !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestScheduleTaskRunsThroughTheEventLoopAndReturnsTheResult(t *testing.T) {
	tt := newTester()
	ch := ScheduleTask(tt.SchedulerBinding, func() int { return 7 }, PriorityIdle)
	other := ScheduleTask(tt.SchedulerBinding, func() string { return "x" }, PriorityTouch)
	if len(tt.host.posted) != 1 {
		t.Fatalf("posted = %d, want one request for the event loop (the second task adds none)", len(tt.host.posted))
	}
	select {
	case <-ch:
		t.Fatal("the task ran before the event loop")
	default:
	}
	tt.host.runPosted() // runs every task, one event loop turn each
	if got := <-ch; got != 7 {
		t.Errorf("result = %d", got)
	}
	if got := <-other; got != "x" {
		t.Errorf("result = %q", got)
	}
}

func TestSchedulingStrategyHoldsTasksBackDuringAnimations(t *testing.T) {
	tt := newTester()
	ran := false
	tt.ScheduleFrameCallback(func(time.Duration) {}) // an animation is pending
	ScheduleTask(tt.SchedulerBinding, func() bool { ran = true; return true }, PriorityIdle)
	if !tt.HandleEventLoopCallback() || ran {
		t.Fatalf("an idle task must wait for the animation: ran %v", ran)
	}
	tick := func() { tt.tick(16 * time.Millisecond) }
	tick() // the animation callback runs
	if tt.HandleEventLoopCallback() || !ran {
		t.Errorf("the task runs once there is no animation: ran %v", ran)
	}
	// A touch task passes the default strategy even during an animation.
	tt.ScheduleFrameCallback(func(time.Duration) {})
	ranTouch := false
	ScheduleTask(tt.SchedulerBinding, func() bool { ranTouch = true; return true }, PriorityTouch)
	tt.HandleEventLoopCallback()
	if !ranTouch {
		t.Error("a touch task must run during an animation")
	}
}

func TestLockEventsHoldsTasks(t *testing.T) {
	tt := newTester()
	unlock := tt.LockEvents()
	unlock2 := tt.LockEvents()
	ran := false
	ScheduleTask(tt.SchedulerBinding, func() bool { ran = true; return true }, PriorityTouch)
	if len(tt.host.posted) != 0 || !tt.Locked() {
		t.Fatalf("a locked binding asks for no event loop turn: posted %d, locked %v", len(tt.host.posted), tt.Locked())
	}
	if tt.HandleEventLoopCallback() || ran {
		t.Error("a locked binding runs no task")
	}
	unlocked := 0
	tt.AddUnlockedCallback(func() { unlocked++ })
	unlock()
	unlock() // twice does nothing
	if !tt.Locked() || unlocked != 0 {
		t.Errorf("one lock is left: locked %v, unlocked callbacks %d", tt.Locked(), unlocked)
	}
	unlock2()
	if tt.Locked() || unlocked != 1 {
		t.Errorf("locked %v, unlocked callbacks %d", tt.Locked(), unlocked)
	}
	tt.host.runPosted()
	if !ran {
		t.Error("the task runs after the unlock")
	}
}

func TestFrameCallbacksRunInOrderOnceAndCanBeCanceled(t *testing.T) {
	// "Can cancel queued callback" of animation_test.dart.
	tt := newTester()
	var secondID int
	firstRan, secondRan := false, false
	tt.ScheduleFrameCallback(func(ts time.Duration) {
		if firstRan || secondRan || ts.Milliseconds() != 0 {
			t.Errorf("first callback: firstRan %v secondRan %v ts %v", firstRan, secondRan, ts)
		}
		firstRan = true
		tt.CancelFrameCallbackWithID(secondID)
	})
	secondID = tt.ScheduleFrameCallback(func(time.Duration) { secondRan = true })

	tt.tick(16 * time.Millisecond)
	if !firstRan || secondRan {
		t.Errorf("firstRan %v secondRan %v, want true false", firstRan, secondRan)
	}
	firstRan, secondRan = false, false
	tt.tick(32 * time.Millisecond)
	if firstRan || secondRan {
		t.Errorf("callbacks run once: firstRan %v secondRan %v", firstRan, secondRan)
	}
	if tt.TransientCallbackCount() != 0 {
		t.Errorf("transient callbacks left: %d", tt.TransientCallbackCount())
	}
}

func TestCanceledIDsAreForgottenAfterTheFrame(t *testing.T) {
	tt := newTester()
	id := tt.ScheduleFrameCallback(func(time.Duration) { t.Error("canceled callback ran") })
	tt.CancelFrameCallbackWithID(id)
	if tt.TransientCallbackCount() != 0 {
		t.Error("a canceled callback leaves the list")
	}
	tt.tick(0)
	ran := false
	tt.ScheduleFrameCallback(func(time.Duration) { ran = true })
	tt.tick(16 * time.Millisecond)
	if !ran {
		t.Error("a later callback runs")
	}
	assertRoute(t, recoverError(t, func() { tt.CancelFrameCallbackWithID(0) }), "SchedulerBinding.CancelFrameCallbackWithID.FrameCallbackIDInvalid")
}

func TestFrameCallbackScheduledDuringAFrameRunsInTheNextOne(t *testing.T) {
	tt := newTester()
	var log []string
	tt.ScheduleFrameCallback(func(time.Duration) {
		log = append(log, "outer")
		tt.ScheduleFrameCallback(func(time.Duration) { log = append(log, "inner") }, ScheduleFrameCallbackOpts{Rescheduling: true})
	})
	tt.tick(1 * time.Millisecond)
	if !reflect.DeepEqual(log, []string{"outer"}) || !tt.HasScheduledFrame() {
		t.Errorf("log %v, scheduled %v", log, tt.HasScheduledFrame())
	}
	tt.tick(2 * time.Millisecond)
	if !reflect.DeepEqual(log, []string{"outer", "inner"}) {
		t.Errorf("log %v", log)
	}
}

func TestScheduleFrameCallbackWithAndWithoutANewFrame(t *testing.T) {
	tt := newTester()
	tt.HandleBeginFrame(foundation.None[time.Duration]())
	tt.HandleDrawFrame()
	called := false
	if tt.HasScheduledFrame() {
		t.Fatal("no frame is scheduled at the start")
	}
	tt.ScheduleFrameCallback(func(time.Duration) { called = true }, ScheduleFrameCallbackOpts{NoNewFrame: true})
	if tt.HasScheduledFrame() {
		t.Error("NoNewFrame must not schedule a frame")
	}
	tt.HandleBeginFrame(foundation.None[time.Duration]())
	tt.HandleDrawFrame()
	if !called {
		t.Error("the callback must run")
	}

	called = false
	tt.ScheduleFrameCallback(func(time.Duration) { called = true })
	if !tt.HasScheduledFrame() {
		t.Error("a frame must be scheduled")
	}
	tt.HandleBeginFrame(foundation.None[time.Duration]())
	tt.HandleDrawFrame()
	if !called || tt.HasScheduledFrame() {
		t.Errorf("called %v, scheduled %v", called, tt.HasScheduledFrame())
	}
}

func TestPersistentAndPostFrameCallbacks(t *testing.T) {
	tt := newTester()
	var log []string
	tt.ScheduleFrameCallback(func(time.Duration) { log = append(log, "transient") })
	tt.AddPersistentFrameCallback(func(time.Duration) {
		log = append(log, "persistent "+tt.SchedulerPhase().Name())
	})
	tt.AddPostFrameCallback(func(time.Duration) {
		log = append(log, "post "+tt.SchedulerPhase().Name())
		tt.AddPostFrameCallback(func(time.Duration) { log = append(log, "post 2") })
	})
	tt.tick(0)
	want := []string{"transient", "persistent persistentCallbacks", "post postFrameCallbacks"}
	if !reflect.DeepEqual(log, want) {
		t.Errorf("first frame: %v, want %v", log, want)
	}
	if tt.SchedulerPhase() != SchedulerPhaseIdle {
		t.Errorf("phase after the frame = %v", tt.SchedulerPhase())
	}
	tt.tick(16 * time.Millisecond)
	want = append(want, "persistent persistentCallbacks", "post 2")
	if !reflect.DeepEqual(log, want) {
		t.Errorf("second frame: %v, want %v", log, want)
	}
}

func TestAddingAPersistentFrameCallbackDuringAPersistentFrameCallback(t *testing.T) {
	tt := newTester()
	calledBack := false
	tt.AddPersistentFrameCallback(func(time.Duration) {
		if !calledBack {
			tt.AddPersistentFrameCallback(func(time.Duration) { calledBack = true })
		}
	})
	tt.HandleBeginFrame(foundation.None[time.Duration]())
	tt.HandleDrawFrame()
	if calledBack {
		t.Error("a callback added during a frame must wait for the next frame")
	}
	tt.HandleBeginFrame(foundation.None[time.Duration]())
	tt.HandleDrawFrame()
	if !calledBack {
		t.Error("the added callback must run in the next frame")
	}
}

func TestPhasesDuringAFrame(t *testing.T) {
	tt := newTester()
	var phases []SchedulerPhase
	note := func(time.Duration) { phases = append(phases, tt.SchedulerPhase()) }
	tt.ScheduleFrameCallback(note)
	tt.AddPersistentFrameCallback(note)
	tt.AddPostFrameCallback(note)
	phases = append(phases, tt.SchedulerPhase())
	tt.HandleBeginFrame(foundation.Some(time.Millisecond))
	phases = append(phases, tt.SchedulerPhase())
	tt.HandleDrawFrame()
	phases = append(phases, tt.SchedulerPhase())
	want := []SchedulerPhase{
		SchedulerPhaseIdle, SchedulerPhaseTransientCallbacks, SchedulerPhaseMidFrameMicrotasks,
		SchedulerPhasePersistentCallbacks, SchedulerPhasePostFrameCallbacks, SchedulerPhaseIdle,
	}
	if !reflect.DeepEqual(phases, want) {
		t.Errorf("phases = %v, want %v", phases, want)
	}
}

func TestFrameOrderErrors(t *testing.T) {
	tt := newTester()
	assertRoute(t, recoverError(t, func() { tt.HandleDrawFrame() }), "SchedulerBinding.HandleDrawFrame.FrameNotBegun")
	tt.HandleBeginFrame(foundation.None[time.Duration]())
	assertRoute(t, recoverError(t, func() { tt.HandleBeginFrame(foundation.None[time.Duration]()) }), "SchedulerBinding.HandleBeginFrame.FrameAlreadyBegun")
	tt.HandleDrawFrame()
	assertRoute(t, recoverError(t, func() { _ = tt.CurrentFrameTimeStamp() }), "SchedulerBinding.CurrentFrameTimeStamp.NoFrameInProgress")
}

func TestCurrentSystemFrameTimeStampIsTheRawTimeStamp(t *testing.T) {
	tt := newTester()
	var lastTimeStamp, lastSystemTimeStamp time.Duration
	frameCallback := func(ts time.Duration) {
		if ts != tt.CurrentFrameTimeStamp() {
			t.Errorf("callback got %v, CurrentFrameTimeStamp is %v", ts, tt.CurrentFrameTimeStamp())
		}
		lastTimeStamp = tt.CurrentFrameTimeStamp()
		lastSystemTimeStamp = tt.CurrentSystemFrameTimeStamp()
	}
	steps := []struct {
		dilation   float64
		raw        time.Duration
		wantStamp  time.Duration
		wantSystem time.Duration
	}{
		{1, 2 * time.Second, 0, 2 * time.Second},
		{1, 4 * time.Second, 2 * time.Second, 4 * time.Second},
		// Setting the time dilation resets the epoch.
		{2, 6 * time.Second, 2 * time.Second, 6 * time.Second},
		{2, 8 * time.Second, 3 * time.Second, 8 * time.Second}, // 2s + (8 - 6)s / 2
	}
	for _, s := range steps {
		tt.SetTimeDilation(s.dilation)
		tt.ScheduleFrameCallback(frameCallback)
		tt.tick(s.raw)
		if lastTimeStamp != s.wantStamp || lastSystemTimeStamp != s.wantSystem {
			t.Errorf("raw %v: stamp %v (want %v), system %v (want %v)", s.raw, lastTimeStamp, s.wantStamp, lastSystemTimeStamp, s.wantSystem)
		}
	}
}

func TestTimeDilation(t *testing.T) {
	tt := newTester()
	if tt.TimeDilation() != 1 {
		t.Errorf("time dilation starts at %v, want 1", tt.TimeDilation())
	}
	tt.SetTimeDilation(2)
	if tt.TimeDilation() != 2 {
		t.Errorf("time dilation = %v", tt.TimeDilation())
	}
	tt.SetTimeDilation(2) // the same: nothing
	for _, bad := range []float64{0, -1} {
		assertRoute(t, recoverError(t, func() { tt.SetTimeDilation(bad) }), "SchedulerBinding.SetTimeDilation.TimeDilationInvalid")
	}
}

func TestDebugAssertNoTimeDilation(t *testing.T) {
	tt := newTester()
	tt.SetTimeDilation(2)
	tt.SetTimeDilation(1)
	if !tt.DebugAssertNoTimeDilation("reason") {
		t.Error("no error once the dilation is reset")
	}
	tt.SetTimeDilation(3)
	assertRoute(t, recoverError(t, func() { tt.DebugAssertNoTimeDilation("reason") }), "SchedulerBinding.DebugAssertNoTimeDilation.TimeDilationActive")
}

func TestDebugAssertNoTransientCallbacks(t *testing.T) {
	tt := newTester()
	if !tt.DebugAssertNoTransientCallbacks("reason") {
		t.Error("no callbacks, no error")
	}
	tt.ScheduleFrameCallback(func(time.Duration) {})
	assertRoute(t, recoverError(t, func() { tt.DebugAssertNoTransientCallbacks("reason") }), "SchedulerBinding.DebugAssertNoTransientCallbacks.TransientCallbacksPending")
}

func TestScheduleFrameAsksTheDispatcherOnce(t *testing.T) {
	tt := newTester()
	if tt.d.OnBeginFrame != nil || tt.d.OnDrawFrame != nil {
		t.Fatal("a new binding registers no callbacks until a frame is scheduled")
	}
	tt.ScheduleFrame()
	tt.ScheduleFrame()
	tt.ScheduleForcedFrame()
	if tt.host.scheduled != 1 || !tt.HasScheduledFrame() {
		t.Errorf("host asked %d times, scheduled %v", tt.host.scheduled, tt.HasScheduledFrame())
	}
	if tt.d.OnBeginFrame == nil || tt.d.OnDrawFrame == nil {
		t.Fatal("scheduling a frame registers the callbacks")
	}
	// The frame the platform gives goes through the dispatcher.
	ran := false
	tt.ScheduleFrameCallback(func(ts time.Duration) { ran = ts == 0 }, ScheduleFrameCallbackOpts{NoNewFrame: true})
	tt.d.BeginFrame(5 * time.Second)
	tt.d.DrawFrame()
	if !ran || tt.HasScheduledFrame() || tt.SchedulerPhase() != SchedulerPhaseIdle {
		t.Errorf("ran %v, scheduled %v, phase %v", ran, tt.HasScheduledFrame(), tt.SchedulerPhase())
	}
}

func TestEnsureFrameCallbacksRegisteredKeepsAnotherListener(t *testing.T) {
	tt := newTester()
	called := 0
	tt.d.OnBeginFrame = func(time.Duration) { called++ }
	tt.EnsureFrameCallbacksRegistered()
	tt.d.BeginFrame(0)
	if called != 1 {
		t.Errorf("the other listener was replaced: called %d", called)
	}
	if tt.d.OnDrawFrame == nil {
		t.Error("OnDrawFrame is still registered")
	}
}

func TestScheduleForcedFrameSetsUpFrameCallbacks(t *testing.T) {
	tt := newTester()
	tt.ScheduleForcedFrame()
	if tt.d.OnBeginFrame == nil || !tt.HasScheduledFrame() {
		t.Error("scheduleForcedFrame sets up frame callbacks")
	}
}

func TestEnsureVisualUpdate(t *testing.T) {
	tests := []struct {
		name  string
		phase func(tt *tester) // gets the binding into a phase, with no frame scheduled
		want  int
	}{
		{"idle", func(*tester) {}, 1},
		{"during transient callbacks", func(tt *tester) {
			tt.ScheduleFrameCallback(func(time.Duration) { tt.EnsureVisualUpdate() }, ScheduleFrameCallbackOpts{NoNewFrame: true})
			tt.HandleBeginFrame(foundation.Some(time.Millisecond))
		}, 0},
		{"during persistent callbacks", func(tt *tester) {
			tt.AddPersistentFrameCallback(func(time.Duration) { tt.EnsureVisualUpdate() })
			tt.HandleBeginFrame(foundation.Some(time.Millisecond))
			tt.HandleDrawFrame()
		}, 0},
		{"during post-frame callbacks", func(tt *tester) {
			tt.AddPostFrameCallback(func(time.Duration) { tt.EnsureVisualUpdate() })
			tt.HandleBeginFrame(foundation.Some(time.Millisecond))
			tt.HandleDrawFrame()
		}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tt := newTester()
			if tc.name == "idle" {
				tt.EnsureVisualUpdate()
			} else {
				tc.phase(tt)
			}
			if tt.host.scheduled != tc.want {
				t.Errorf("frames asked = %d, want %d", tt.host.scheduled, tc.want)
			}
		})
	}
}

func TestEndOfFrame(t *testing.T) {
	tt := newTester()
	a := tt.EndOfFrame()
	b := tt.EndOfFrame()
	if tt.host.scheduled != 1 {
		t.Errorf("frames asked = %d, want 1", tt.host.scheduled)
	}
	for _, ch := range []<-chan struct{}{a, b} {
		select {
		case <-ch:
			t.Fatal("closed before the frame")
		default:
		}
	}
	tt.tick(0)
	for i, ch := range []<-chan struct{}{a, b} {
		select {
		case <-ch:
		default:
			t.Errorf("channel %d not closed after the frame", i)
		}
	}
	c := tt.EndOfFrame() // a new frame is awaited
	select {
	case <-c:
		t.Error("the next channel waits for the next frame")
	default:
	}
}

func TestFramesStopWhileTheAppIsNotVisible(t *testing.T) {
	tests := []struct {
		state   ui.AppLifecycleState
		enabled bool
	}{
		{ui.AppLifecycleStateResumed, true},
		{ui.AppLifecycleStateInactive, true},
		{ui.AppLifecycleStateHidden, false},
		{ui.AppLifecycleStatePaused, false},
		{ui.AppLifecycleStateDetached, false},
	}
	for _, tc := range tests {
		t.Run(tc.state.String(), func(t *testing.T) {
			tt := newTester()
			tt.HandleAppLifecycleStateChanged(ui.AppLifecycleStateDetached) // start from frames off
			tt.HandleAppLifecycleStateChanged(tc.state)
			if tt.FramesEnabled() != tc.enabled || tt.LifecycleState() != tc.state {
				t.Errorf("frames enabled %v (want %v), state %v", tt.FramesEnabled(), tc.enabled, tt.LifecycleState())
			}
			tt.ScheduleFrame()
			if scheduled := tt.host.scheduled > 0; scheduled != tc.enabled {
				t.Errorf("a frame was asked for: %v, want %v", scheduled, tc.enabled)
			}
		})
	}
}

func TestEnablingFramesSchedulesOne(t *testing.T) {
	tt := newTester()
	tt.HandleAppLifecycleStateChanged(ui.AppLifecycleStatePaused)
	tt.HandleAppLifecycleStateChanged(ui.AppLifecycleStatePaused) // same state: nothing
	if tt.host.scheduled != 0 {
		t.Fatal("a paused app asks for no frame")
	}
	tt.HandleAppLifecycleStateChanged(ui.AppLifecycleStateResumed)
	if tt.host.scheduled != 1 {
		t.Errorf("resuming asks for a frame: %d", tt.host.scheduled)
	}
	tt.ResetInternalState()
	if tt.LifecycleState() != ui.AppLifecycleStateUnspecified || !tt.FramesEnabled() {
		t.Error("ResetInternalState")
	}
}

func TestNewSchedulerBindingReadsTheLifecycleState(t *testing.T) {
	host := &fakeHost{}
	d := ui.NewPlatformDispatcher(host)
	d.SetAppLifecycleState(ui.AppLifecycleStateHidden)
	b := NewSchedulerBinding(d)
	if b.FramesEnabled() || b.LifecycleState() != ui.AppLifecycleStateHidden {
		t.Errorf("frames enabled %v, state %v", b.FramesEnabled(), b.LifecycleState())
	}
	if b.PlatformDispatcher() != d {
		t.Error("PlatformDispatcher")
	}
}

func TestScheduleForcedFrameWorksWhileFramesAreDisabled(t *testing.T) {
	tt := newTester()
	tt.HandleAppLifecycleStateChanged(ui.AppLifecycleStatePaused)
	tt.ScheduleFrame()
	if tt.HasScheduledFrame() {
		t.Error("a normal frame is not scheduled while frames are disabled")
	}
	tt.ScheduleForcedFrame()
	if !tt.HasScheduledFrame() || tt.host.scheduled != 1 {
		t.Error("a forced frame is")
	}
}

func TestSchedulerPhaseNames(t *testing.T) {
	tests := []struct {
		p    SchedulerPhase
		name string
		str  string
	}{
		{SchedulerPhaseIdle, "idle", "SchedulerPhase.idle"},
		{SchedulerPhasePostFrameCallbacks, "postFrameCallbacks", "SchedulerPhase.postFrameCallbacks"},
		{SchedulerPhase(99), "SchedulerPhase(99)", "SchedulerPhase(99)"},
	}
	for _, tc := range tests {
		if tc.p.Name() != tc.name || tc.p.String() != tc.str {
			t.Errorf("%d: %q %q", int(tc.p), tc.p.Name(), tc.p.String())
		}
	}
}
