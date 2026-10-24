package widgets_test

import (
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
	"slices"
	"strings"
	"testing"

	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/widgets/binding_test.dart: the observer
// tests (the lifecycle, the reentrant removal, 'WidgetsBindingObserver callbacks
// handle exceptions gracefully', the pop route, the memory pressure) and
// 'Application lifecycle affects frame scheduling', and the back gesture tests
// (handleStartBackGesture, handleUpdateBackGestureProgress, handleCommitBackGesture and
// handleCancelBackGesture, with the dispatcher in place of the flutter/backgesture channel).
// The route push tests, the view focus tests and the stack filter test are not ported:
// Glutter has no navigation channel and no stack filters.

// recordingObserver is LoggingObserver and ThrowingObserver of binding_test.dart.
type recordingObserver struct {
	WidgetsBindingObserverBase
	log    *[]string
	name   string
	throws bool
	pops   bool
	// takesBackGesture is what HandleStartBackGesture returns.
	takesBackGesture bool
	// throwsAfterStart makes the observer panic in the callbacks of a back gesture after its
	// start (BackGestureThrowingObserver); the start itself returns takesBackGesture.
	throwsAfterStart bool
	response         ui.AppExitResponse
	states           []ui.AppLifecycleState
	locales          [][]ui.Locale
	// onCall runs at each callback, for the reentrant test.
	onCall func()
}

func (o *recordingObserver) record(what string) {
	*o.log = append(*o.log, o.name+":"+what)
	if o.onCall != nil {
		o.onCall()
	}
	if o.throws || (o.throwsAfterStart && what != "handleStartBackGesture") {
		panic(errors.New(tagObserverFailed, o.name+" failed in "+what))
	}
}

func (o *recordingObserver) DidPopRoute() bool { o.record("didPopRoute"); return o.pops }
func (o *recordingObserver) HandleStartBackGesture(services.PredictiveBackEvent) bool {
	o.record("handleStartBackGesture")
	return o.takesBackGesture
}
func (o *recordingObserver) HandleUpdateBackGestureProgress(services.PredictiveBackEvent) {
	o.record("handleUpdateBackGestureProgress")
}
func (o *recordingObserver) HandleCommitBackGesture() { o.record("handleCommitBackGesture") }
func (o *recordingObserver) HandleCancelBackGesture() { o.record("handleCancelBackGesture") }
func (o *recordingObserver) DidChangeMetrics()        { o.record("didChangeMetrics") }
func (o *recordingObserver) DidChangeTextScaleFactor() {
	o.record("didChangeTextScaleFactor")
}
func (o *recordingObserver) DidChangePlatformBrightness() {
	o.record("didChangePlatformBrightness")
}
func (o *recordingObserver) DidChangeLocales(l []ui.Locale) {
	o.locales = append(o.locales, l)
	o.record("didChangeLocales")
}
func (o *recordingObserver) DidChangeAppLifecycleState(s ui.AppLifecycleState) {
	o.states = append(o.states, s)
	o.record("didChangeAppLifecycleState")
}
func (o *recordingObserver) DidRequestAppExit() ui.AppExitResponse {
	o.record("didRequestAppExit")
	return o.response
}
func (o *recordingObserver) DidHaveMemoryPressure() { o.record("didHaveMemoryPressure") }
func (o *recordingObserver) DidChangeAccessibilityFeatures() {
	o.record("didChangeAccessibilityFeatures")
}

// startBackGestureEvent is the event of the start of the Dart tests: touchOffset [5.0, 300.0],
// progress 0.0, swipeEdge 0 (left).
var startBackEvent = ui.BackGestureEvent{HasTouchOffset: true, TouchX: 5, TouchY: 300, SwipeEdge: ui.BackSwipeEdgeLeft}

func TestObserversHandleExceptionsGracefully(t *testing.T) {
	tests := []struct {
		name string
		// trigger makes the call of the platform that the observers hear of.
		trigger func(ts *gluttertest.Tester)
	}{
		{"didChangeAppLifecycleState", func(ts *gluttertest.Tester) { ts.Dispatcher().SetAppLifecycleState(ui.AppLifecycleStatePaused) }},
		{"didPopRoute", func(ts *gluttertest.Tester) { ts.Binding.HandlePopRoute() }},
		{"didChangeMetrics", func(ts *gluttertest.Tester) {
			ts.SetViewConfiguration(func(c *ui.ViewConfiguration) { c.PhysicalSize = ui.Size{Width: 400, Height: 300} })
		}},
		{"didRequestAppExit", func(ts *gluttertest.Tester) { ts.Binding.HandleRequestAppExit() }},
		{"didChangeTextScaleFactor", func(ts *gluttertest.Tester) {
			ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.TextScaleFactor = 1.5 })
		}},
		{"didChangePlatformBrightness", func(ts *gluttertest.Tester) {
			ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.PlatformBrightness = ui.BrightnessDark })
		}},
		{"didChangeAccessibilityFeatures", func(ts *gluttertest.Tester) {
			ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.AccessibilityFeatures.BoldText = true })
		}},
		{"didChangeLocales", func(ts *gluttertest.Tester) {
			ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.Locales = []ui.Locale{{LanguageCode: "es"}} })
		}},
		{"didHaveMemoryPressure", func(ts *gluttertest.Tester) { ts.Binding.HandleMemoryPressure() }},
		{"handleStartBackGesture", func(ts *gluttertest.Tester) { ts.Dispatcher().StartBackGesture(startBackEvent) }},
		{"handleUpdateBackGestureProgress", func(ts *gluttertest.Tester) {
			ts.Dispatcher().StartBackGesture(startBackEvent)
			ts.Dispatcher().UpdateBackGestureProgress(ui.BackGestureEvent{HasTouchOffset: true, TouchX: 100, TouchY: 300, Progress: 0.35, SwipeEdge: ui.BackSwipeEdgeLeft})
		}},
		{"handleCommitBackGesture", func(ts *gluttertest.Tester) {
			ts.Dispatcher().StartBackGesture(startBackEvent)
			ts.Dispatcher().CommitBackGesture()
		}},
		{"handleCancelBackGesture", func(ts *gluttertest.Tester) {
			ts.Dispatcher().StartBackGesture(startBackEvent)
			ts.Dispatcher().CancelBackGesture()
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var log []string
			// The back gesture cases use BackGestureThrowingObserver, which takes the start and
			// throws in the rest; the others use ThrowingObserver.
			throwing := &recordingObserver{log: &log, name: "throwing", throws: true}
			if strings.HasPrefix(tc.name, "handle") && tc.name != "handleStartBackGesture" {
				throwing = &recordingObserver{log: &log, name: "throwing", throwsAfterStart: true, takesBackGesture: true}
			}
			logging := &recordingObserver{log: &log, name: "logging", takesBackGesture: true}
			ts.Binding.AddObserver(throwing)
			ts.Binding.AddObserver(logging)
			tc.trigger(ts)
			if !slices.Contains(log, "logging:"+tc.name) {
				t.Errorf("log = %v, want the second observer to be called after the first one failed", log)
			}
			errs := ts.TakeErrors()
			if len(errs) != 1 {
				t.Fatalf("%d errors reported, want 1: %v", len(errs), errs)
			}
			appErr, ok := errors.FromError(errs[0])
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", errs[0])
			}
			if !appErr.HasRoute("ObserverFailed") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if want := "while dispatching notifications for WidgetsBindingObserver." + tc.name; appErr.Metadata["context"] != want {
				t.Errorf("context = %v, want %q", appErr.Metadata["context"], want)
			}
		})
	}
}

func TestObserverRemovingItselfDuringCallbacks(t *testing.T) {
	ts := gluttertest.New(t)
	var log []string
	obs := &recordingObserver{log: &log, name: "self"}
	obs.onCall = func() { ts.Binding.RemoveObserver(obs) }
	ts.Binding.AddObserver(obs)
	ts.Dispatcher().SetAppLifecycleState(ui.AppLifecycleStateResumed)
	ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.TextScaleFactor = 2 })
	if len(log) != 1 {
		t.Errorf("log = %v, want the observer called once, before it removed itself", log)
	}
	if ts.Binding.RemoveObserver(obs) {
		t.Error("RemoveObserver found an observer that had removed itself")
	}
}

func TestObserversAreCalledInOrderAndEachRegistrationCounts(t *testing.T) {
	ts := gluttertest.New(t)
	var log []string
	a := &recordingObserver{log: &log, name: "a"}
	b := &recordingObserver{log: &log, name: "b"}
	ts.Binding.AddObserver(a)
	ts.Binding.AddObserver(b)
	ts.Binding.AddObserver(a)
	ts.Binding.HandleMemoryPressure()
	want := []string{"a:didHaveMemoryPressure", "b:didHaveMemoryPressure", "a:didHaveMemoryPressure"}
	if !slices.Equal(log, want) {
		t.Errorf("log = %v, want %v", log, want)
	}
	for i, want := range []bool{true, true, false} {
		if got := ts.Binding.RemoveObserver(a); got != want {
			t.Errorf("RemoveObserver #%d = %v, want %v: it removes one registration each time", i+1, got, want)
		}
	}
}

func TestObserverLifecycleStates(t *testing.T) {
	ts := gluttertest.New(t)
	var log []string
	obs := &recordingObserver{log: &log, name: "o"}
	ts.Binding.AddObserver(obs)
	for _, s := range []ui.AppLifecycleState{ui.AppLifecycleStateResumed, ui.AppLifecycleStateInactive,
		ui.AppLifecycleStateInactive, ui.AppLifecycleStateHidden} {
		ts.Dispatcher().SetAppLifecycleState(s)
	}
	want := []ui.AppLifecycleState{ui.AppLifecycleStateResumed, ui.AppLifecycleStateInactive, ui.AppLifecycleStateHidden}
	if !slices.Equal(obs.states, want) {
		t.Errorf("states = %v, want %v (the same state twice is not a change)", obs.states, want)
	}
	if got := ts.Binding.Scheduler().LifecycleState(); got != ui.AppLifecycleStateHidden {
		t.Errorf("the scheduler's state = %v, want hidden", got)
	}
}

func TestObserverLocales(t *testing.T) {
	ts := gluttertest.New(t)
	var log []string
	obs := &recordingObserver{log: &log, name: "o"}
	ts.Binding.AddObserver(obs)
	ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.Locales = []ui.Locale{{LanguageCode: "fr"}, {LanguageCode: "en"}} })
	if len(obs.locales) != 1 || len(obs.locales[0]) != 2 || obs.locales[0][0].LanguageCode != "fr" {
		t.Errorf("locales = %v, want one call with fr, en", obs.locales)
	}
}

// TestFramesAreEnabledWhenTheRootIsAttached checks that no frame is scheduled
// before runApp attaches the root widget ('readyToProduceFrames' of
// binding.dart), and that attaching it schedules the first one.
func TestFramesAreEnabledWhenTheRootIsAttached(t *testing.T) {
	ts := gluttertest.New(t)
	sched := ts.Binding.Scheduler()
	sched.ScheduleFrame()
	if sched.HasScheduledFrame() || sched.FramesEnabled() {
		t.Fatal("a frame was scheduled before the root widget was attached")
	}
	if ts.Binding.IsRootWidgetAttached() || ts.Binding.RootElement() != nil {
		t.Fatal("the root is attached at the start")
	}
	ts.Binding.AttachRootWidget(ts.Binding.WrapWithDefaultView(newLeaf()))
	if !ts.Binding.IsRootWidgetAttached() || !sched.FramesEnabled() || !sched.HasScheduledFrame() {
		t.Errorf("attached = %v, frames enabled = %v, scheduled = %v; want all true",
			ts.Binding.IsRootWidgetAttached(), sched.FramesEnabled(), sched.HasScheduledFrame())
	}
	ts.Pump()
	if sched.HasScheduledFrame() {
		t.Error("a frame is still scheduled after the first one")
	}
}

// TestApplicationLifecycleAffectsFrameScheduling is the test of the same name
// of binding_test.dart, without the warm up frame (which the scheduler does not
// have).
func TestApplicationLifecycleAffectsFrameScheduling(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(newLeaf())
	sched := ts.Binding.Scheduler()
	steps := []struct {
		name      string
		do        func()
		scheduled bool
	}{
		{"at the start", func() {}, false},
		{"paused", func() { ts.Dispatcher().SetAppLifecycleState(ui.AppLifecycleStatePaused) }, false},
		{"resumed", func() { ts.Dispatcher().SetAppLifecycleState(ui.AppLifecycleStateResumed) }, true},
		{"after a frame", func() { ts.Pump() }, false},
		{"inactive", func() { ts.Dispatcher().SetAppLifecycleState(ui.AppLifecycleStateInactive) }, false},
		{"paused again", func() { ts.Dispatcher().SetAppLifecycleState(ui.AppLifecycleStatePaused) }, false},
		{"detached", func() { ts.Dispatcher().SetAppLifecycleState(ui.AppLifecycleStateDetached) }, false},
		{"inactive again", func() { ts.Dispatcher().SetAppLifecycleState(ui.AppLifecycleStateInactive) }, true},
		{"after another frame", func() { ts.Pump() }, false},
		{"paused for the last time", func() { ts.Dispatcher().SetAppLifecycleState(ui.AppLifecycleStatePaused) }, false},
		{"scheduleFrame while paused", sched.ScheduleFrame, false},
		{"scheduleForcedFrame while paused", sched.ScheduleForcedFrame, true},
	}
	for _, step := range steps {
		step.do()
		if got := sched.HasScheduledFrame(); got != step.scheduled {
			t.Fatalf("%s: HasScheduledFrame = %v, want %v", step.name, got, step.scheduled)
		}
	}
}

func TestHandlePopRoute(t *testing.T) {
	tests := []struct {
		name      string
		observers []bool // whether each observer handles the pop
		want      bool
		wantExit  int
		wantCalls int
	}{
		{"no observers asks to exit", nil, false, 1, 0},
		{"unhandled asks to exit", []bool{false, false}, false, 1, 2},
		{"the first that handles it ends the walk", []bool{false, true, true}, true, 0, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var log []string
			for i, handles := range tc.observers {
				ts.Binding.AddObserver(&recordingObserver{log: &log, name: string(rune('a' + i)), pops: handles})
			}
			if got := ts.Binding.HandlePopRoute(); got != tc.want {
				t.Errorf("HandlePopRoute = %v, want %v", got, tc.want)
			}
			if ts.ExitRequests() != tc.wantExit || len(log) != tc.wantCalls {
				t.Errorf("exit requests = %d, calls = %d; want %d, %d", ts.ExitRequests(), len(log), tc.wantExit, tc.wantCalls)
			}
		})
	}
}

func TestHandleRequestAppExitAndExitApplication(t *testing.T) {
	tests := []struct {
		name      string
		responses []ui.AppExitResponse
		exitType  ui.AppExitType
		want      ui.AppExitResponse
		wantExit  int
	}{
		{"nobody objects", nil, ui.AppExitTypeCancelable, ui.AppExitResponseExit, 1},
		{"unspecified counts as exit", []ui.AppExitResponse{ui.AppExitResponseUnspecified}, ui.AppExitTypeCancelable, ui.AppExitResponseExit, 1},
		{"one cancels", []ui.AppExitResponse{ui.AppExitResponseExit, ui.AppExitResponseCancel}, ui.AppExitTypeCancelable, ui.AppExitResponseCancel, 0},
		{"a required exit cannot be canceled", []ui.AppExitResponse{ui.AppExitResponseCancel}, ui.AppExitTypeRequired, ui.AppExitResponseExit, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var log []string
			for i, r := range tc.responses {
				ts.Binding.AddObserver(&recordingObserver{log: &log, name: string(rune('a' + i)), response: r})
			}
			if got := ts.Binding.ExitApplication(tc.exitType); got != tc.want {
				t.Errorf("ExitApplication = %v, want %v", got, tc.want)
			}
			if ts.ExitRequests() != tc.wantExit {
				t.Errorf("exit requests = %d, want %d", ts.ExitRequests(), tc.wantExit)
			}
			wantCalls := len(tc.responses)
			if tc.exitType == ui.AppExitTypeRequired {
				wantCalls = 0
			}
			if len(log) != wantCalls {
				t.Errorf("observers were asked %d times, want %d: a required exit asks nobody, a cancelable one asks everybody", len(log), wantCalls)
			}
		})
	}
}

// TestClosingTheWindowAsksTheObservers checks SPEC §8.7 "Closing": the close
// request of the window goes through the dispatcher to the observers.
func TestClosingTheWindowAsksTheObservers(t *testing.T) {
	ts := gluttertest.New(t)
	var log []string
	obs := &recordingObserver{log: &log, name: "o", response: ui.AppExitResponseCancel}
	ts.Binding.AddObserver(obs)
	if ts.Dispatcher().RequestClose(ui.ImplicitViewID) {
		t.Error("the window may close although the observer canceled")
	}
	obs.response = ui.AppExitResponseExit
	if !ts.Dispatcher().RequestClose(ui.ImplicitViewID) {
		t.Error("the window may not close although the observer agreed")
	}
}

func TestWidgetsBindingOf(t *testing.T) {
	ts := gluttertest.New(t)
	var got *WidgetsBinding
	ts.PumpWidget(builderOf(func(bc BuildContext) Widget { got = WidgetsBindingOf(bc); return newLeaf() }))
	if got != ts.Binding {
		t.Errorf("WidgetsBindingOf = %p, want the binding %p", got, ts.Binding)
	}
}

// TestDrawFrameBuildsThenLaysOut checks the order of WidgetsBinding.drawFrame:
// the dirty widgets are built, then the render tree is laid out and painted, and
// the first frame reaches the engine.
func TestDrawFrameBuildsThenLaysOut(t *testing.T) {
	ts := gluttertest.New(t)
	var order []string
	var setState func(func())
	size := 10.0
	ts.PumpWidget(newBuilder(func(_ BuildContext, ss func(func())) Widget {
		setState = ss
		order = append(order, "build")
		return newBox(size, size)
	}))
	if len(ts.Scenes()) != 1 {
		t.Fatalf("%d scenes after the first frame, want 1", len(ts.Scenes()))
	}
	if !ts.Binding.FirstFrameRasterized() {
		t.Error("the first frame was not counted as rasterized")
	}
	select {
	case <-ts.Binding.WaitUntilFirstFrameRasterized():
	default:
		t.Error("WaitUntilFirstFrameRasterized is not closed after the first frame")
	}
	setState(func() { size = 20 })
	if !ts.Binding.Scheduler().HasScheduledFrame() {
		t.Fatal("SetState did not schedule a frame")
	}
	ts.Pump()
	boxes := ts.Elements(gluttertest.ByPredicate("test widget", func(w Widget) bool { _, ok := w.(*box); return ok }))
	if len(boxes) != 1 {
		t.Fatalf("%d boxes, want 1", len(boxes))
	}
	if !slices.Equal(order, []string{"build", "build"}) {
		t.Errorf("order = %v, want two builds", order)
	}
	if len(ts.Scenes()) != 2 {
		t.Errorf("%d scenes, want one per frame", len(ts.Scenes()))
	}
}

// probeBox is a render box whose layout runs a function.
type probeBox struct {
	rendering.RenderBox
	onLayout func()
}

func (b *probeBox) PerformLayout() {
	b.onLayout()
	b.SetSize(b.BoxConstraints().Smallest())
}

// layoutProbe is a leaf render object widget with a probeBox.
type layoutProbe struct {
	LeafRenderObjectWidget
	onLayout func()
}

func (w *layoutProbe) CreateRenderObject(BuildContext) rendering.RenderObject {
	b := &probeBox{onLayout: w.onLayout}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

// TestBuildScheduledDuringFrame checks the error of binding.dart
// _handleBuildScheduled: a SetState from a layout callback would show one frame
// late, so debug builds report it.
func TestBuildScheduledDuringFrame(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	var setState func(func())
	armed := false
	ts.PumpWidget(newMulti(
		newBuilder(func(_ BuildContext, ss func(func())) Widget { setState = ss; return newLeaf() }),
		&layoutProbe{LeafRenderObjectWidget: NewLeafRenderObjectWidgetBase(), onLayout: func() {
			if armed {
				setState(func() {})
			}
		}},
	))
	if errs := ts.Errors(); len(errs) != 0 {
		t.Fatalf("errors while attaching: %v", errs)
	}
	armed = true
	probes := ts.Elements(gluttertest.ByPredicate("test widget", func(w Widget) bool { _, ok := w.(*layoutProbe); return ok }))
	probes[0].AsElement().RenderObject().MarkNeedsLayout()
	ts.Pump()
	errs := ts.TakeErrors()
	if len(errs) != 1 {
		t.Fatalf("%d errors, want 1: %v", len(errs), errs)
	}
	_ = requireErrorTag(t, errs[0], "BuildScheduledDuringFrame")
}

func TestBackGestureObservers(t *testing.T) {
	update := ui.BackGestureEvent{HasTouchOffset: true, TouchX: 100, TouchY: 300, Progress: 0.35, SwipeEdge: ui.BackSwipeEdgeLeft}
	tests := []struct {
		name string
		// takes is whether each observer takes the start, in order of registration.
		takes []bool
		// run is what the engine does.
		run func(d *ui.PlatformDispatcher) (tookStart bool)
		// wantLog lists the calls, wantExit the exit requests that a pop with no handler makes.
		wantLog   []string
		wantTook  bool
		wantExits int
	}{
		{
			name:  "start asks every observer, and only those that took it hear the rest",
			takes: []bool{false, true, true},
			run: func(d *ui.PlatformDispatcher) bool {
				took := d.StartBackGesture(startBackEvent)
				d.UpdateBackGestureProgress(update)
				d.CommitBackGesture()
				return took
			},
			wantLog: []string{
				"a:handleStartBackGesture", "b:handleStartBackGesture", "c:handleStartBackGesture",
				"b:handleUpdateBackGestureProgress", "c:handleUpdateBackGestureProgress",
				"b:handleCommitBackGesture", "c:handleCommitBackGesture",
			},
			wantTook: true,
		},
		{
			name:  "cancel goes to the observers that took the start",
			takes: []bool{true, false},
			run: func(d *ui.PlatformDispatcher) bool {
				took := d.StartBackGesture(startBackEvent)
				d.CancelBackGesture()
				return took
			},
			wantLog:  []string{"a:handleStartBackGesture", "b:handleStartBackGesture", "a:handleCancelBackGesture"},
			wantTook: true,
		},
		{
			name:  "commit with no observer that took the start pops the route",
			takes: []bool{false, false},
			run: func(d *ui.PlatformDispatcher) bool {
				took := d.StartBackGesture(startBackEvent)
				d.UpdateBackGestureProgress(update)
				d.CommitBackGesture()
				return took
			},
			// The pop asks the observers (DidPopRoute logs nothing here) and then exits.
			wantLog:   []string{"a:handleStartBackGesture", "b:handleStartBackGesture", "a:didPopRoute", "b:didPopRoute"},
			wantTook:  false,
			wantExits: 1,
		},
		{
			name:      "commit without a start pops the route",
			takes:     nil,
			run:       func(d *ui.PlatformDispatcher) bool { d.CommitBackGesture(); return false },
			wantExits: 1,
		},
		{
			name:  "cancel without a start does nothing",
			takes: []bool{true},
			run:   func(d *ui.PlatformDispatcher) bool { d.CancelBackGesture(); return false },
		},
		{
			name:  "update without a start does nothing",
			takes: []bool{true},
			run:   func(d *ui.PlatformDispatcher) bool { d.UpdateBackGestureProgress(update); return false },
		},
		{
			name:  "a new start forgets the observers of the gesture before",
			takes: []bool{true},
			run: func(d *ui.PlatformDispatcher) bool {
				d.StartBackGesture(startBackEvent)
				// The second start clears the list of the first, so the observer is in it once.
				took := d.StartBackGesture(startBackEvent)
				d.UpdateBackGestureProgress(update)
				return took
			},
			wantLog: []string{
				"a:handleStartBackGesture", "a:handleStartBackGesture", "a:handleUpdateBackGestureProgress",
			},
			wantTook: true,
		},
		{
			name:  "a cancel does not forget the observers (Flutter clears them only at the next start)",
			takes: []bool{true},
			run: func(d *ui.PlatformDispatcher) bool {
				d.StartBackGesture(startBackEvent)
				d.CancelBackGesture()
				d.CommitBackGesture()
				return true
			},
			wantLog:  []string{"a:handleStartBackGesture", "a:handleCancelBackGesture", "a:handleCommitBackGesture"},
			wantTook: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var log []string
			for i, takes := range tc.takes {
				ts.Binding.AddObserver(&recordingObserver{log: &log, name: string(rune('a' + i)), takesBackGesture: takes})
			}
			if got := tc.run(ts.Dispatcher()); got != tc.wantTook {
				t.Errorf("StartBackGesture = %v, want %v", got, tc.wantTook)
			}
			if !slices.Equal(log, tc.wantLog) {
				t.Errorf("log = %v, want %v", log, tc.wantLog)
			}
			if ts.ExitRequests() != tc.wantExits {
				t.Errorf("exit requests = %d, want %d", ts.ExitRequests(), tc.wantExits)
			}
		})
	}
}

func TestBackGestureEventReachesTheObserver(t *testing.T) {
	ts := gluttertest.New(t)
	var got []services.PredictiveBackEvent
	obs := &eventObserver{onStart: func(e services.PredictiveBackEvent) bool { got = append(got, e); return true },
		onUpdate: func(e services.PredictiveBackEvent) { got = append(got, e) }}
	ts.Binding.AddObserver(obs)
	ts.Dispatcher().StartBackGesture(startBackEvent)
	ts.Dispatcher().UpdateBackGestureProgress(ui.BackGestureEvent{HasTouchOffset: true, TouchX: 100, TouchY: 300, Progress: 0.35, SwipeEdge: ui.BackSwipeEdgeRight})
	want := []services.PredictiveBackEvent{
		services.NewPredictiveBackEvent(foundation.Some(ui.Offset{Dx: 5, Dy: 300}), 0, services.SwipeEdgeLeft),
		services.NewPredictiveBackEvent(foundation.Some(ui.Offset{Dx: 100, Dy: 300}), 0.35, services.SwipeEdgeRight),
	}
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("event %d = %v, want %v", i, got[i], want[i])
		}
	}
}

// eventObserver is an observer that hands the back gesture events to functions.
type eventObserver struct {
	WidgetsBindingObserverBase
	onStart  func(services.PredictiveBackEvent) bool
	onUpdate func(services.PredictiveBackEvent)
}

func (o *eventObserver) HandleStartBackGesture(e services.PredictiveBackEvent) bool {
	return o.onStart(e)
}
func (o *eventObserver) HandleUpdateBackGestureProgress(e services.PredictiveBackEvent) {
	o.onUpdate(e)
}

func TestBackGestureWithABadEventIsReported(t *testing.T) {
	tests := []struct {
		name  string
		event ui.BackGestureEvent
	}{
		{"progress out of range", ui.BackGestureEvent{Progress: 2, SwipeEdge: ui.BackSwipeEdgeLeft}},
		{"no swipe edge", ui.BackGestureEvent{Progress: 0.5}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var log []string
			ts.Binding.AddObserver(&recordingObserver{log: &log, name: "a", takesBackGesture: true})
			if ts.Dispatcher().StartBackGesture(tc.event) {
				t.Error("the framework took a gesture whose event is not valid")
			}
			if len(log) != 0 {
				t.Errorf("the observer heard of a bad event: %v", log)
			}
			errs := ts.TakeErrors()
			if len(errs) != 1 {
				t.Fatalf("%d errors reported, want 1: %v", len(errs), errs)
			}
			appErr, ok := errors.FromError(errs[0])
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", errs[0])
			}
			if !foundation.IsRecoveredPanic(errs[0]) {
				t.Errorf("the error is not the report of a recovered panic: %s", appErr.Route())
			}
		})
	}
}

func TestDispatcherCallbacksReachTheBinding(t *testing.T) {
	ts := gluttertest.New(t)
	var log []string
	ts.Binding.AddObserver(&recordingObserver{log: &log, name: "a", pops: true})
	ts.Dispatcher().PopRoute()
	ts.Dispatcher().NotifyMemoryPressure()
	if want := []string{"a:didPopRoute", "a:didHaveMemoryPressure"}; !slices.Equal(log, want) {
		t.Errorf("log = %v, want %v", log, want)
	}
	if ts.ExitRequests() != 0 {
		t.Errorf("exit requests = %d, want 0 (an observer handled the pop)", ts.ExitRequests())
	}
}

// backHost is a host that implements ui.BackHost.
type backHost struct {
	*renderhost.Host
	calls []bool
}

func (h *backHost) SetFrameworkHandlesBack(handles bool) { h.calls = append(h.calls, handles) }

func TestSetFrameworkHandlesBackReachesTheBackHost(t *testing.T) {
	skipUnlessDebug(t)
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformAndroid))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
	tests := []struct {
		name string
		host ui.PlatformHost
		call []bool
		want []bool
	}{
		{"a host with BackHost", &backHost{Host: &renderhost.Host{}}, []bool{true, false}, []bool{true, false}},
		{"a host without BackHost", &renderhost.Host{}, []bool{true, false}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if foundation.IsWeb {
				// system_navigator.dart: the call is for Android and not for the web.
				tc.want = nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			d := ui.NewPlatformDispatcher(tc.host)
			b := NewWidgetsBinding(ctx, d, WidgetsBindingOpts{ErrorReporter: func(err error) { t.Error(err) }})
			for _, v := range tc.call {
				b.SystemNavigator().SetFrameworkHandlesBack(v)
			}
			if bh, ok := tc.host.(*backHost); ok {
				if !slices.Equal(bh.calls, tc.want) {
					t.Errorf("host calls = %v, want %v", bh.calls, tc.want)
				}
			} else if tc.want != nil {
				t.Error("test table error")
			}
		})
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagObserverFailed = "ObserverFailed"
)
