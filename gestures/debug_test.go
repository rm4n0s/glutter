//go:build !glutter_release && !glutter_profile

package gestures

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/fakeclock"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// The tests here check debug-only behavior: the panics of failed debug checks,
// the diagnostics output and the debug sampling clock.

func TestArenaErrors(t *testing.T) {
	tests := []struct {
		name  string
		do    func(a *arenaTester)
		route string
	}{
		{"joining a closed arena", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.arena.Close(primaryKey)
			a.arena.Add(primaryKey, &testGestureArenaMember{t: a.t, key: primaryKey})
		}, "Add.GestureArenaClosed"},
		{"sweeping an open arena", func(a *arenaTester) {
			a.addFirst()
			a.arena.Sweep(primaryKey)
		}, "Sweep.GestureArenaOpenOnSweep"},
		{"resolving for a member that is not in the arena", func(a *arenaTester) {
			a.addFirst()
			stranger := &arenaEntry{a.arena, primaryKey, &testGestureArenaMember{t: a.t, key: primaryKey}}
			stranger.Resolve(GestureAccepted)
		}, "resolve.GestureArenaMemberUnknown"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := newArenaTester(t)
			assertRoute(t, recoverError(t, func() { tc.do(a) }), tc.route)
		})
	}
}

func TestArenaDiagnostics(t *testing.T) {
	var lines []string
	a := newArenaTester(t)
	a.arena = NewGestureArenaManager(a.micro.schedule, GestureArenaManagerOpts{Debug: &DebugOptions{
		PrintGestureArenaDiagnostics: true,
		Print:                        func(line string) { lines = append(lines, line) },
	}})
	a.addFirst()
	a.arena.Close(primaryKey)
	a.micro.flush()
	if len(lines) < 4 {
		t.Fatalf("got %d lines: %q", len(lines), lines)
	}
	if want := "Gesture arena 4    ❙ ★ Opening new gesture arena."; lines[0] != want {
		t.Errorf("lines[0] = %q, want %q", lines[0], want)
	}
	if want := "Gesture arena 4    ❙ Closing with 1 member."; lines[2] != want {
		t.Errorf("lines[2] = %q, want %q", lines[2], want)
	}
}

func TestBindingHitTestDebugOutput(t *testing.T) {
	var lines []string
	host := &fakeHost{}
	d := ui.NewPlatformDispatcher(host)
	d.AddView(ui.ImplicitViewID, ui.ViewConfiguration{DevicePixelRatio: 1})
	b := NewGestureBinding(d, scheduler.NewSchedulerBinding(d), GestureBindingOpts{Debug: DebugOptions{
		PrintHitTestResults: true, Print: func(l string) { lines = append(lines, l) },
	}})
	b.HandlePointerEvent(NewPointerDownEvent(PointerEventOpts{Pointer: 1}))
	if len(lines) != 1 {
		t.Fatalf("got %d lines: %q", len(lines), lines)
	}
	if want := "HitTestResult(HitTestEntryBase#"; len(lines[0]) < len(want) {
		t.Errorf("line = %q", lines[0])
	}
}

func TestBindingErrors(t *testing.T) {
	bt := newBindingTester(t, 1)
	tests := []struct {
		name  string
		do    func()
		route string
	}{
		{"a down event for a pointer that is down", func() {
			bt.binding.HandlePointerEvent(NewPointerDownEvent(PointerEventOpts{Pointer: 1}))
			bt.binding.HandlePointerEvent(NewPointerDownEvent(PointerEventOpts{Pointer: 1}))
		}, "GestureBinding.handlePointerEventImmediately.PointerHitTestExists"},
		{"an event without a hit test result", func() {
			bt.binding.DispatchEvent(NewPointerMoveEvent(), nil)
		}, "DispatchEvent.PointerEventNotHitTested"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertRoute(t, recoverError(t, tc.do), tc.route)
		})
	}
	unlock := bt.sched.LockEvents()
	defer unlock()
	assertRoute(t, recoverError(t, func() { bt.binding.HandlePointerEvent(NewPointerAddedEvent()) }),
		"HandlePointerEvent.GestureBindingLocked")
}

func TestBindingResamplesEvents(t *testing.T) {
	// Pointer event resampling, from gesture_binding_resample_event_test.dart.
	d := ui.NewPlatformDispatcher(nil) // posts run at once, as timers fire inside the fake clock
	d.AddView(ui.ImplicitViewID, ui.ViewConfiguration{PhysicalSize: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 1})
	clock := fakeclock.New(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))
	sched := &fakeScheduler{}
	b := NewGestureBinding(d, sched, GestureBindingOpts{Clock: sampleClock{clock}})
	var events []PointerEvent
	b.OnHitTestInView = func(result *HitTestResult, _ ui.Offset, _ ui.ViewID) {
		result.Add(NewHitTestEntry(recordingTarget{events: &events}))
	}

	currentTime := func() time.Duration { return time.Duration(clock.Now().UnixMilli()) * time.Millisecond }
	epoch := currentTime()
	ms := func(n int) time.Duration { return epoch + time.Duration(n)*time.Millisecond }
	packet := ui.PointerDataPacket{Data: []ui.PointerData{
		{Change: ui.PointerChangeAdd, TimeStamp: ms(0)},
		{Change: ui.PointerChangeDown, TimeStamp: ms(10)},
		{Change: ui.PointerChangeMove, PhysicalX: 10, TimeStamp: ms(20)},
		{Change: ui.PointerChangeMove, PhysicalX: 20, TimeStamp: ms(30)},
		{Change: ui.PointerChangeMove, PhysicalX: 30, TimeStamp: ms(40)},
		{Change: ui.PointerChangeMove, PhysicalX: 40, TimeStamp: ms(50)},
		{Change: ui.PointerChangeMove, PhysicalX: 50, TimeStamp: ms(60)},
		{Change: ui.PointerChangeUp, PhysicalX: 50, TimeStamp: ms(70)},
		{Change: ui.PointerChangeRemove, PhysicalX: 50, TimeStamp: ms(70)},
	}}
	const samplingOffset = -5 * time.Millisecond
	const frameInterval = 16667 * time.Microsecond
	b.ResamplingEnabled = true
	b.SamplingOffset = samplingOffset

	d.DispatchPointerDataPacket(packet)
	if len(events) != 0 {
		t.Fatalf("%d events were dispatched before the first frame", len(events))
	}
	// A frame callback should have been requested.
	callback := sched.postFrameCallback
	sched.postFrameCallback = nil
	if callback == nil {
		t.Fatal("no post-frame callback was added")
	}
	sched.frameTime = ms(15)
	callback(0)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1: %v", len(events), typeNames(events))
	}
	if _, ok := events[0].(*PointerDownEvent); !ok || events[0].TimeStamp() != sched.frameTime+samplingOffset ||
		events[0].Position() != (ui.Offset{}) {
		t.Errorf("event 0 = %v at %v, want a down event at %v", events[0], events[0].TimeStamp(), sched.frameTime+samplingOffset)
	}
	// The second frame callback should have been requested.
	callback = sched.postFrameCallback
	sched.postFrameCallback = nil
	if callback == nil {
		t.Fatal("no second post-frame callback was added")
	}
	frameTime := ms(25)
	sched.frameTime = frameTime
	callback(0)
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if _, ok := events[1].(*PointerMoveEvent); !ok || events[1].TimeStamp() != frameTime+samplingOffset ||
		events[1].Position() != (ui.Offset{Dx: 10}) || events[1].Delta() != (ui.Offset{Dx: 10}) {
		t.Errorf("event 1 = %v at %v, delta %v", events[1], events[1].TimeStamp(), events[1].Delta())
	}
	// Resampling continues without a frame callback.
	clock.Advance(frameInterval * 3 / 2)
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3: %v", len(events), typeNames(events))
	}
	if _, ok := events[2].(*PointerMoveEvent); !ok || events[2].TimeStamp() != frameTime+frameInterval+samplingOffset {
		t.Errorf("event 2 = %v at %v, want a move at %v", events[2], events[2].TimeStamp(), frameTime+frameInterval+samplingOffset)
	}
	clock.Advance(frameInterval)
	if len(events) != 5 {
		t.Fatalf("got %d events, want 5: %v", len(events), typeNames(events))
	}
	if _, ok := events[3].(*PointerMoveEvent); !ok || events[3].TimeStamp() != frameTime+2*frameInterval+samplingOffset {
		t.Errorf("event 3 = %v at %v", events[3], events[3].TimeStamp())
	}
	if _, ok := events[4].(*PointerUpEvent); !ok || events[4].TimeStamp() != frameTime+2*frameInterval+samplingOffset {
		t.Errorf("event 4 = %v at %v", events[4], events[4].TimeStamp())
	}
	clock.Advance(frameInterval)
	if len(events) != 5 {
		t.Errorf("got %d events, want 5: no more events", len(events))
	}
	b.ResamplingEnabled = false
}

func TestEventsToStringControl(t *testing.T) {
	e := NewPointerDownEvent()
	for name, s := range map[string]string{"String": e.String(), "ToStringFull": ToStringFull(e)} {
		if s == "" || strings.Contains(s, "\n") || strings.TrimSpace(s) != s {
			t.Errorf("%s = %q, want a one-line description", name, s)
		}
		if !strings.HasPrefix(s, "PointerDownEvent#") {
			t.Errorf("%s = %q, want it to start with PointerDownEvent#", name, s)
		}
	}
	if full, short := ToStringFull(e), e.String(); len(full) <= len(short) {
		t.Errorf("the full description %q is not longer than %q", full, short)
	}
}

func TestEventsDescription(t *testing.T) {
	e := NewPointerDownEvent(PointerEventOpts{Position: ui.Offset{Dx: 20, Dy: 30}})
	re := `^PointerDownEvent#[0-9a-f]{5}\(position: Offset\(20\.0, 30\.0\)\)$`
	if s := e.String(); !strings.HasPrefix(s, "PointerDownEvent#") || !strings.HasSuffix(s, "(position: Offset(20.0, 30.0))") {
		t.Errorf("String() = %q, want to match %s", s, re)
	}
	scroll := NewPointerScrollEvent(PointerEventOpts{ScrollDelta: ui.Offset{Dx: 1, Dy: 2}})
	if s := scroll.String(); !strings.Contains(s, "scrollDelta: Offset(1.0, 2.0)") {
		t.Errorf("String() = %q, want the scroll delta", s)
	}
	// The debug level shows the local values and the time stamp.
	full := ToStringFull(NewPointerMoveEvent(PointerEventOpts{TimeStamp: 1500 * time.Millisecond, Pointer: 3}))
	for _, want := range []string{"timeStamp: 0:00:01.500000", "pointer: 3", "kind: touch", "down: true", "buttons: 1"} {
		if !strings.Contains(full, want) {
			t.Errorf("ToStringFull = %q, want it to contain %q", full, want)
		}
	}
}

func TestEventKinds(t *testing.T) {
	// Regression tests for https://github.com/flutter/flutter/issues/107962 and 108176.
	trackpad := PointerEventOpts{Kind: ui.PointerDeviceKindTrackpad}
	for name, make := range map[string]func(){
		"hover":                 func() { NewPointerHoverEvent(trackpad) },
		"scroll inertia cancel": func() { NewPointerScrollInertiaCancelEvent(trackpad) },
		"scroll":                func() { NewPointerScrollEvent(trackpad) },
	} {
		t.Run("allowed "+name, func(t *testing.T) { make() })
	}
	for name, make := range map[string]func(){
		"down":   func() { NewPointerDownEvent(trackpad) },
		"move":   func() { NewPointerMoveEvent(trackpad) },
		"up":     func() { NewPointerUpEvent(trackpad) },
		"cancel": func() { NewPointerCancelEvent(trackpad) },
		"enter":  func() { NewPointerEnterEvent(trackpad) },
		"exit":   func() { NewPointerExitEvent(trackpad) },
	} {
		t.Run("not allowed "+name, func(t *testing.T) {
			assertRoute(t, recoverError(t, make), "newEventData.PointerEventKindTrackpad")
		})
	}
}

func TestHitTestResultErrors(t *testing.T) {
	tests := []struct {
		name  string
		do    func()
		route string
	}{
		{"an entry added twice", func() {
			r := NewHitTestResult()
			e := newDummyEntry()
			r.Add(e)
			r.Add(e)
		}, "Add.HitTestEntryAddedTwice"},
		{"a transform with perspective", func() {
			m := ui.NewMatrix4Identity()
			m.SetEntry(2, 3, 0.5)
			NewHitTestResult().PushTransform(m)
		}, "PushTransform.HitTestTransformPerspective"},
		{"popping more than pushed", func() {
			r := NewHitTestResult()
			r.PopTransform()
		}, "PopTransform.HitTestTransformStackEmpty"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertRoute(t, recoverError(t, tc.do), tc.route)
		})
	}
}

func TestRouterRemoveErrors(t *testing.T) {
	router := NewPointerRouter()
	handle := router.AddRoute(1, func(PointerEvent) {}, nil)
	tests := []struct {
		name  string
		do    func()
		route string
	}{
		{"unknown pointer", func() { router.RemoveRoute(2, handle) }, "RemoveRoute.PointerRouteNotFound"},
		{"unknown route", func() { router.RemoveRoute(1, PointerRouteHandle{}) }, "RemoveRoute.PointerRouteNotFound"},
		{"unknown global route", func() { router.RemoveGlobalRoute(handle) }, "RemoveGlobalRoute.PointerRouteNotFound"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertRoute(t, recoverError(t, tc.do), tc.route)
		})
	}
	router.RemoveRoute(1, handle)
	if n := router.DebugGlobalRouteCount(); n != 0 {
		t.Errorf("DebugGlobalRouteCount() = %d, want 0", n)
	}
	router.AddGlobalRoute(func(PointerEvent) {}, nil)
	if n := router.DebugGlobalRouteCount(); n != 1 {
		t.Errorf("DebugGlobalRouteCount() = %d, want 1", n)
	}
}

func TestPointerSignalResolverErrors(t *testing.T) {
	t.Run("registering another event while one is pending is an error", func(t *testing.T) {
		resolver := NewPointerSignalResolver()
		resolver.Register(NewPointerScrollEvent(), func(PointerSignalEvent) {})
		assertRoute(t, recoverError(t, func() {
			resolver.Register(NewPointerScrollEvent(), func(PointerSignalEvent) {})
		}), "Register.PointerSignalEventMismatch")
	})
	t.Run("resolving another event than the registered one is an error", func(t *testing.T) {
		resolver := NewPointerSignalResolver()
		resolver.Register(NewPointerScrollEvent(), func(PointerSignalEvent) {})
		assertRoute(t, recoverError(t, func() { resolver.Resolve(NewPointerScrollEvent()) }),
			"Resolve.PointerSignalEventMismatch")
	})
}

func TestLeastSquaresSolverLengthMismatch(t *testing.T) {
	err := recoverError(t, func() { NewLeastSquaresSolver([]float64{1, 2}, []float64{1}, []float64{1, 1}) })
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "LeastSquaresLengthMismatch" {
		t.Errorf("unexpected error %v", err)
	}
}

func TestVelocityTrackerErrors(t *testing.T) {
	clock := sampleClock{fakeclock.New(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))}
	t.Run("no points", func(t *testing.T) {
		tracker := NewIOSScrollViewFlingVelocityTracker(clock, ui.PointerDeviceKindTouch)
		assertRoute(t, recoverError(t, func() { tracker.GetVelocity() }), "VelocityNoSamples")
	})
	t.Run("a point that precedes the previous one", func(t *testing.T) {
		tracker := NewIOSScrollViewFlingVelocityTracker(clock, ui.PointerDeviceKindTouch)
		tracker.AddPosition(time.Hour, ui.Offset{})
		assertRoute(t, recoverError(t, func() { tracker.AddPosition(time.Second, ui.Offset{}) }), "VelocityPositionPrecedesPredecessor")
	})
}

func TestDragAcceptGestureToleratesNoLastPendingEventTimestamp(t *testing.T) {
	g := newGestureTester(t)
	recognizer := NewVerticalDragGestureRecognizer(g.binding)
	event := NewPointerDownEvent(PointerEventOpts{TimeStamp: 240 * time.Hour})
	if recognizer.DebugLastPendingEventTimestamp().IsSome() {
		t.Fatal("the timestamp is set before any pointer")
	}
	recognizer.AddAllowedPointer(event)
	if got := recognizer.DebugLastPendingEventTimestamp(); got != foundation.Some(event.TimeStamp()) {
		t.Fatalf("timestamp = %v, want %v", got, event.TimeStamp())
	}
	recognizer.AcceptGesture(event.Pointer())
	if recognizer.DebugLastPendingEventTimestamp().IsSome() {
		t.Error("the timestamp is kept after accepting")
	}
	recognizer.RejectGesture(event.Pointer())
	recognizer.AcceptGesture(event.Pointer())
	if recognizer.DebugLastPendingEventTimestamp().IsSome() {
		t.Error("the timestamp is set after rejecting")
	}
}

func TestGestureRecognizerSmoke(t *testing.T) {
	g := newGestureTester(t)
	r := &testGestureRecognizer{}
	r.Init(g.binding, r, GestureRecognizerOpts{DebugOwner: 0})
	if s := foundation.ToDiagnosticsNode(r).ToStringDeep(); s == "" {
		t.Error("the recognizer has no description")
	}
}

// The failure cases of the recognizers: each tag has a test of its route.
func TestRecognizerErrors(t *testing.T) {
	someTime := ms(1)
	for _, tc := range []struct {
		name  string
		route string
		fn    func(g *gestureTester)
	}{
		{"drag update primary delta", "NewDragUpdateDetails.DragUpdatePrimaryDeltaInvalid", func(*gestureTester) {
			NewDragUpdateDetails(ui.Offset{}, DragUpdateOpts{Delta: off(1, 1), PrimaryDelta: foundation.Some(5.0)})
		}},
		{"drag end primary velocity", "NewDragEndDetails.DragEndPrimaryVelocityInvalid", func(*gestureTester) {
			NewDragEndDetails(DragEndOpts{Velocity: NewVelocity(off(1, 1)), PrimaryVelocity: foundation.Some(5.0)})
		}},
		{"drag event while ready", "HandleEvent.DragEventWhileReady", func(g *gestureTester) {
			NewHorizontalDragGestureRecognizer(g.binding).HandleEvent(evMove(1, off(1, 1)))
		}},
		{"drag stopped while ready", "DidStopTrackingLastPointer.DragStoppedWhileReady", func(g *gestureTester) {
			NewHorizontalDragGestureRecognizer(g.binding).DidStopTrackingLastPointer(1)
		}},
		{"drag pointer accepted twice", "AcceptGesture.DragPointerAcceptedTwice", func(g *gestureTester) {
			r := NewHorizontalDragGestureRecognizer(g.binding)
			r.AddAllowedPointer(evDown(1, off(1, 1)))
			r.AcceptGesture(1)
			r.AcceptGesture(1)
		}},
		{"drag without a max delta pointer", "resolveDelta.DragNoMaxDeltaPointer", func(g *gestureTester) {
			NewHorizontalDragGestureRecognizer(g.binding).resolveDelta(1, dragDirectionHorizontal, off(5, 0))
		}},
		{"long press unhandled button", "currentCallbacks.LongPressButtonUnhandled", func(g *gestureTester) {
			r := NewLongPressGestureRecognizer(g.binding)
			r.initialButtons = foundation.Some(12)
			r.currentCallbacks()
		}},
		{"multi drag state unknown", "removeState.MultiDragStateUnknown", func(g *gestureTester) {
			NewImmediateMultiDragGestureRecognizer(g.binding).removeState(7)
		}},
		{"multi tap pointer added twice", "AddAllowedPointer.MultiTapPointerAddedTwice", func(g *gestureTester) {
			r := NewMultiTapGestureRecognizer(g.binding)
			r.AddAllowedPointer(evDown(1, off(1, 1)))
			r.AddAllowedPointer(evDown(1, off(1, 1)))
		}},
		{"recognizer entries left", "Dispose.RecognizerEntriesLeft", func(g *gestureTester) {
			r := &stuckRecognizer{}
			r.InitOneSequence(g.binding, r, GestureRecognizerOpts{})
			r.StartTrackingPointer(1, nil)
			r.Dispose()
		}},
		{"recognizer event while ready", "HandleEvent.RecognizerEventWhileReady", func(g *gestureTester) {
			var res []string
			newTestPrimaryPointer(g.binding, GestureAccepted, isUp, &res, SlopTolerance{}, SlopTolerance{}).HandleEvent(evMove(1, off(1, 1)))
		}},
		{"recognizer without a deadline override", "DidExceedDeadline.RecognizerNoDeadline", func(g *gestureTester) {
			r := &testPrimaryPointerRecognizer{resolutions: new([]string)}
			r.InitPrimaryPointer(g.binding, r, PrimaryPointerOpts{Deadline: foundation.Some(ms(5))})
			r.DidExceedDeadline()
		}},
		{"recognizer unknown pointer (kind)", "GetKindForPointer.RecognizerPointerUnknown", func(g *gestureTester) {
			NewTapGestureRecognizer(g.binding).GetKindForPointer(99)
		}},
		{"recognizer unknown pointer (buttons)", "GetButtonsForPointer.RecognizerPointerUnknown", func(g *gestureTester) {
			NewTapGestureRecognizer(g.binding).GetButtonsForPointer(99)
		}},
		{"recognizer stopped while ready", "DidStopTrackingLastPointer.RecognizerStoppedWhileReady", func(g *gestureTester) {
			var res []string
			newTestPrimaryPointer(g.binding, GestureAccepted, isUp, &res, SlopTolerance{}, SlopTolerance{}).DidStopTrackingLastPointer(1)
		}},
		{"recognizer team set to nil", "SetTeam.RecognizerTeamSetTwice", func(g *gestureTester) {
			NewTapGestureRecognizer(g.binding).SetTeam(nil)
		}},
		{"recognizer team set twice", "SetTeam.RecognizerTeamSetTwice", func(g *gestureTester) {
			r := NewTapGestureRecognizer(g.binding)
			team := NewGestureArenaTeam(g.binding.GestureArena)
			r.SetTeam(team)
			r.SetTeam(team)
		}},
		{"negative slop tolerance", "InitPrimaryPointer.SlopToleranceNegative", func(g *gestureTester) {
			NewTapGestureRecognizer(g.binding, TapGestureRecognizerOpts{PreAcceptSlopTolerance: NewSlopTolerance(-1)})
		}},
		{"scale event while ready", "HandleEvent.ScaleEventWhileReady", func(g *gestureTester) {
			NewScaleGestureRecognizer(g.binding).HandleEvent(evMove(1, off(1, 1)))
		}},
		{"negative scale", "NewScaleUpdateDetails.ScaleNegative", func(*gestureTester) {
			NewScaleUpdateDetails(ScaleUpdateOpts{Scale: foundation.Some(-1.0)})
		}},
		{"scale pointer queue short", "updateLines.ScalePointerQueueShort", func(g *gestureTester) {
			r := NewScaleGestureRecognizer(g.binding)
			r.pointerLocations.set(1, off(0, 0))
			r.updateLines()
		}},
		{"scale stopped in an invalid state", "DidStopTrackingLastPointer.ScaleStoppedInInvalidState", func(g *gestureTester) {
			NewScaleGestureRecognizer(g.binding).DidStopTrackingLastPointer(1)
		}},
		{"serial tap count", "NewSerialTapCancelDetails->serialCount.SerialTapCountInvalid", func(*gestureTester) {
			NewSerialTapCancelDetails(-1)
		}},
		{"tap cancel without down", "Resolve.TapCancelWithoutDown", func(g *gestureTester) {
			r := NewTapGestureRecognizer(g.binding)
			r.wonArenaForPrimaryPointer = true
			r.Resolve(GestureRejected)
		}},
		{"tap tracking without down", "StartTrackingPointer.TapTrackingWithoutDown", func(g *gestureTester) {
			NewTapGestureRecognizer(g.binding).StartTrackingPointer(1, nil)
		}},
		{"tap and drag pointer accepted twice", "AcceptGesture.TapDragPointerAcceptedTwice", func(g *gestureTester) {
			r := NewTapAndPanGestureRecognizer(g.binding)
			r.OnTapDown = func(TapDragDownDetails) {}
			r.AddPointer(evDown(1, off(1, 1)))
			r.AcceptGesture(1)
			r.AcceptGesture(1)
		}},
		{"team accept for the wrong pointer", "AcceptGesture.TeamAcceptInvalid", func(g *gestureTester) {
			team := NewGestureArenaTeam(g.binding.GestureArena)
			team.Add(1, fakeArenaMember{})
			team.combiners[1].AcceptGesture(2)
		}},
		{"team add for the wrong pointer", "add.TeamAddInvalid", func(g *gestureTester) {
			team := NewGestureArenaTeam(g.binding.GestureArena)
			team.Add(1, fakeArenaMember{})
			team.combiners[1].add(2, fakeArenaMember{})
		}},
		{"team resolved twice", "close.TeamResolvedTwice", func(g *gestureTester) {
			team := NewGestureArenaTeam(g.binding.GestureArena)
			team.Add(1, fakeArenaMember{})
			c := team.combiners[1]
			c.close()
			c.close()
		}},
		{"velocity bounds", "ClampMagnitude.VelocityBoundsInvalid", func(*gestureTester) {
			VelocityZero.ClampMagnitude(-1, 5)
		}},
		{"velocity times going back", "previousVelocityAt.VelocityTimesNotIncreasing", func(g *gestureTester) {
			tracker := NewIOSScrollViewFlingVelocityTracker(sampleClock{g.clock}, ui.PointerDeviceKindTouch)
			tracker.touchSamples[0] = &pointAtTime{time: 5 * someTime}
			tracker.touchSamples[1] = &pointAtTime{time: someTime}
			tracker.index = 1
			tracker.previousVelocityAt(0)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			assertRoute(t, recoverError(t, func() { tc.fn(g) }), tc.route)
		})
	}
}

// stuckRecognizer never resolves, so its entries stay after Dispose.
type stuckRecognizer struct{ OneSequenceGestureRecognizer }

func (*stuckRecognizer) DebugDescription() string       { return "stuck" }
func (*stuckRecognizer) HandleEvent(PointerEvent)       {}
func (*stuckRecognizer) DidStopTrackingLastPointer(int) {}
func (*stuckRecognizer) Resolve(GestureDisposition)     {}
