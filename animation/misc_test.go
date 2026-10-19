package animation

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/animation/animation_style_test.dart,
// animation_from_listener_test.dart, listener_helpers_test.dart,
// iteration_patterns_test.dart and futures_test.dart. animation_sheet_test.dart
// and live_binding_test.dart test flutter_test's AnimationSheetBuilder and live
// binding, which gluttertest does not have.

func TestAnimationStyleCopyWithEqualsBasics(t *testing.T) {
	if !NewAnimationStyle().Equals(NewAnimationStyle().CopyWith(AnimationStyle{})) {
		t.Error("a copy with nothing changed should be equal")
	}
}

func TestAnimationStyleCopyWithOverridesAllProperties(t *testing.T) {
	original := NewAnimationStyle(AnimationStyle{Curve: Curves.Ease, Duration: foundation.Some(time.Second), ReverseCurve: Curves.Ease, ReverseDuration: foundation.Some(time.Second)})
	cp := original.CopyWith(AnimationStyle{Curve: Curves.Linear, Duration: foundation.Some(2 * time.Second), ReverseCurve: Curves.Linear, ReverseDuration: foundation.Some(2 * time.Second)})
	if cp.Curve != Curves.Linear || cp.Duration.Value() != 2*time.Second || cp.ReverseCurve != Curves.Linear || cp.ReverseDuration.Value() != 2*time.Second {
		t.Errorf("copy = %v", cp)
	}
}

func TestAnimationStyleMergeFillsInNullProperties(t *testing.T) {
	original := NewAnimationStyle(AnimationStyle{Curve: Curves.Ease, Duration: foundation.Some(time.Second), ReverseCurve: Curves.Ease, ReverseDuration: foundation.Some(time.Second)})
	other := AnimationStyle{Curve: Curves.Linear, Duration: foundation.Some(2 * time.Second)}
	merged := original.Merge(&other)
	if merged.Curve != Curves.Linear || merged.Duration.Value() != 2*time.Second || merged.ReverseCurve != Curves.Ease || merged.ReverseDuration.Value() != time.Second {
		t.Errorf("merged = %v", merged)
	}
	if got := original.Merge(nil); !got.Equals(original) {
		t.Errorf("merging nil changed the style: %v", got)
	}
}

func TestAnimationStyleLerpIdentical(t *testing.T) {
	if LerpAnimationStyle(nil, nil, 0) != nil {
		t.Error("lerp of nil and nil should be nil")
	}
	data := &AnimationStyle{}
	if LerpAnimationStyle(data, data, 0.5) != data {
		t.Error("lerp of identical styles should be the same style")
	}
}

func TestAnimationStyleLerpSmoothlyTransitionsAllValues(t *testing.T) {
	a := &AnimationStyle{Curve: Curves.Ease, Duration: foundation.Some(time.Second), ReverseCurve: Curves.Ease, ReverseDuration: foundation.Some(time.Second)}
	b := &AnimationStyle{Curve: Curves.Linear, Duration: foundation.Some(2 * time.Second), ReverseCurve: Curves.Linear, ReverseDuration: foundation.Some(2 * time.Second)}
	if got := LerpAnimationStyle(a, b, 0); !got.Equals(*a) {
		t.Errorf("lerp at 0 = %v, want a", got)
	}
	if got := LerpAnimationStyle(a, b, 1); !got.Equals(*b) {
		t.Errorf("lerp at 1 = %v, want b", got)
	}
	const steps = 5
	for s := 0; s < steps; s++ {
		st := float64(s+1) / (steps + 1)
		lerped := LerpAnimationStyle(a, b, st)
		want := time.Duration(ui.LerpDouble(float64(time.Second.Microseconds()), float64((2*time.Second).Microseconds()), st)+0.5) * time.Microsecond
		if got := lerped.Duration.Value(); got != want {
			t.Errorf("duration at %v = %v, want %v", st, got, want)
		}
		if got := lerped.ReverseDuration.Value(); got != want {
			t.Errorf("reverse duration at %v = %v, want %v", st, got, want)
		}
		for c := 0; c < steps; c++ {
			x := float64(c+1) / (steps + 1)
			wantCurve := ui.LerpDouble(Curves.Ease.Transform(x), Curves.Linear.Transform(x), st)
			moreOrLess(t, "lerped curve", lerped.Curve.Transform(x), wantCurve)
		}
	}
}

func TestAnimationStyleLerpWithNullProperty(t *testing.T) {
	a := &AnimationStyle{Duration: foundation.Some(2 * time.Second)}
	b := &AnimationStyle{}
	if got := LerpAnimationStyle(a, b, 1.0/2).Duration.Value(); got != time.Second {
		t.Errorf("duration = %v, want 1s", got)
	}
	if got := LerpAnimationStyle(a, b, 1.0/4).Duration.Value(); got != 1500*time.Millisecond {
		t.Errorf("duration = %v, want 1.5s", got)
	}
}

func TestAnimationStyleNoAnimationAndString(t *testing.T) {
	if AnimationStyleNoAnimation.Duration.Value() != 0 || AnimationStyleNoAnimation.ReverseDuration.Value() != 0 || !AnimationStyleNoAnimation.Duration.IsSome() {
		t.Errorf("noAnimation = %v", AnimationStyleNoAnimation)
	}
	if got := NewAnimationStyle().String(); got != "AnimationStyle()" {
		t.Errorf("empty style = %q", got)
	}
	s := AnimationStyle{Curve: Curves.EaseInOut, Duration: foundation.Some(time.Second), ReverseCurve: Curves.BounceInOut, ReverseDuration: foundation.Some(2 * time.Second)}
	for _, want := range []string{"curve: Cubic(0.42, 0.00, 0.58, 1.00)", "duration: 1s", "reverseCurve: BounceInOut", "reverseDuration: 2s"} {
		if !strings.Contains(s.String(), want) {
			t.Errorf("String = %q, want it to contain %q", s.String(), want)
		}
	}
}

func TestAnimationStyleWithAFunctionCurveDoesNotPanicOnEquals(t *testing.T) {
	f := CurveFunc(func(t float64) float64 { return t })
	a := AnimationStyle{Curve: f}
	if a.Equals(AnimationStyle{Curve: f}) {
		t.Error("curves that can't be compared are not equal")
	}
	if !a.Equals(AnimationStyle{}) == false {
		t.Error("a style with a curve is not equal to one without")
	}
}

func TestAnimationCreatedFromValueListenable(t *testing.T) {
	listenable := foundation.NewValueNotifier(0.0)
	animation := NewAnimationFromValueListenable[float64](listenable, nil)
	if animation.Status() != forward || animation.Value() != 0 {
		t.Fatalf("status, value = %v, %v", animation.Status(), animation.Value())
	}
	listenable.SetValue(1.0)
	if animation.Value() != 1 {
		t.Errorf("value = %v", animation.Value())
	}
	called := false
	h := animation.AddListener(func() { called = true })
	listenable.SetValue(0.5)
	if !called {
		t.Error("the listener was not called")
	}
	called = false
	animation.RemoveListener(h)
	listenable.SetValue(0.2)
	if called {
		t.Error("a removed listener was called")
	}
}

func TestAnimationCreatedFromValueListenableCanTransformValue(t *testing.T) {
	listenable := foundation.NewValueNotifier(0.0)
	animation := NewAnimationFromValueListenable[float64](listenable, func(v float64) float64 { return v / 10 })
	if animation.Status() != forward || animation.Value() != 0 {
		t.Fatalf("status, value = %v, %v", animation.Status(), animation.Value())
	}
	listenable.SetValue(10)
	if animation.Value() != 1 {
		t.Errorf("value = %v, want 1", animation.Value())
	}
}

func TestAnimationCreatedFromValueListenableCanBeDriven(t *testing.T) {
	listenable := foundation.NewValueNotifier(0.0)
	animation := NewAnimationFromValueListenable[float64](listenable, nil)
	offset := Drive(animation, NewAnimatableFromCallback(func(v float64) ui.Offset { return ui.Offset{Dy: v} }))
	if offset.Value() != (ui.Offset{}) || offset.Status() != forward {
		t.Fatalf("value, status = %v, %v", offset.Value(), offset.Status())
	}
	listenable.SetValue(10)
	if offset.Value() != (ui.Offset{Dy: 10}) {
		t.Errorf("value = %v", offset.Value())
	}
	called := false
	h := offset.AddListener(func() { called = true })
	listenable.SetValue(0.5)
	if !called {
		t.Error("the listener was not called")
	}
	called = false
	offset.RemoveListener(h)
	listenable.SetValue(0.2)
	if called {
		t.Error("a removed listener was called")
	}
}

func TestLazyListenersRemovingAnUnregisteredListenerIsANoOp(t *testing.T) {
	var starts, stops int
	var local LocalListeners
	var status LocalStatusListeners
	var lazy LazyListeners
	lazy.OnStartListening = func() { starts++ }
	lazy.OnStopListening = func() { stops++ }
	local.OnRegister, local.OnUnregister = lazy.DidRegisterListener, lazy.DidUnregisterListener
	status.OnRegister, status.OnUnregister = lazy.DidRegisterListener, lazy.DidUnregisterListener

	local.RemoveListener(foundation.ListenerHandle{})
	status.RemoveStatusListener(StatusListenerHandle{})
	if starts != 0 || stops != 0 {
		t.Fatalf("starts, stops = %d, %d, want 0, 0", starts, stops)
	}
	h := local.AddListener(func() {})
	sh := status.AddStatusListener(func(AnimationStatus) {})
	if starts != 1 || !lazy.IsListening() {
		t.Errorf("starts = %d, listening = %v", starts, lazy.IsListening())
	}
	local.RemoveListener(h)
	local.RemoveListener(h)
	if stops != 0 {
		t.Errorf("stops = %d after removing only one of two listeners", stops)
	}
	status.RemoveStatusListener(sh)
	status.RemoveStatusListener(sh)
	if stops != 1 || lazy.IsListening() {
		t.Errorf("stops = %d, listening = %v", stops, lazy.IsListening())
	}
}

func TestAnimationControllerWithMutatingListener(t *testing.T) {
	controller := NewAnimationController(newTester(), dur(ms(100)))
	var log []string
	var h1, h3 foundation.ListenerHandle
	listener1 := func() { log = append(log, "listener1") }
	listener3 := func() { log = append(log, "listener3") }
	listener4 := func() { log = append(log, "listener4") }
	listener2 := func() {
		log = append(log, "listener2")
		controller.RemoveListener(h1)
		controller.RemoveListener(h3)
		controller.AddListener(listener4)
	}
	h1 = controller.AddListener(listener1)
	controller.AddListener(listener2)
	h3 = controller.AddListener(listener3)
	for _, tc := range []struct {
		value float64
		want  []string
	}{{0.2, []string{"listener1", "listener2"}}, {0.3, []string{"listener2", "listener4"}},
		// Deviation: a listener is a handle, not a function identity, so the
		// listener4 that listener2 adds again on each call is a second listener
		// and is called twice, as in Flutter's status listeners.
		{0.4, []string{"listener2", "listener4", "listener4"}}} {
		controller.SetValue(tc.value)
		if strings.Join(log, ",") != strings.Join(tc.want, ",") {
			t.Errorf("SetValue(%v): log = %v, want %v", tc.value, log, tc.want)
		}
		log = nil
	}
}

func TestAnimationControllerWithMutatingStatusListener(t *testing.T) {
	controller := NewAnimationController(newTester(), dur(ms(100)))
	var log []string
	var h1, h3 StatusListenerHandle
	listener1 := func(AnimationStatus) { log = append(log, "listener1") }
	listener3 := func(AnimationStatus) { log = append(log, "listener3") }
	listener4 := func(AnimationStatus) { log = append(log, "listener4") }
	listener2 := func(AnimationStatus) {
		log = append(log, "listener2")
		controller.RemoveStatusListener(h1)
		controller.RemoveStatusListener(h3)
		controller.AddStatusListener(listener4)
	}
	h1 = controller.AddStatusListener(listener1)
	controller.AddStatusListener(listener2)
	h3 = controller.AddStatusListener(listener3)
	for i, tc := range []struct {
		run  func()
		want []string
	}{
		{func() { controller.Forward() }, []string{"listener1", "listener2"}},
		{func() { controller.Reverse() }, []string{"listener2", "listener4"}},
		{func() { controller.Forward() }, []string{"listener2", "listener4", "listener4"}},
	} {
		tc.run()
		if strings.Join(log, ",") != strings.Join(tc.want, ",") {
			t.Errorf("step %d: log = %v, want %v", i, log, tc.want)
		}
		log = nil
	}
	controller.Dispose()
}

func TestAnimationControllerWithThrowingListener(t *testing.T) {
	tt := newTester()
	var reported []error
	tt.reporter = func(err error) { reported = append(reported, err) }
	controller := NewAnimationController(envTester{tt}, dur(ms(100)))
	defer controller.Dispose()
	var log []string
	controller.AddListener(func() { log = append(log, "listener1") })
	controller.AddListener(func() { log = append(log, "badListener"); panic("bad") })
	controller.AddListener(func() { log = append(log, "listener2") })
	controller.SetValue(0.2)
	if strings.Join(log, ",") != "listener1,badListener,listener2" {
		t.Errorf("log = %v", log)
	}
	if len(reported) != 1 {
		t.Errorf("reported %d errors, want 1", len(reported))
	}
}

func TestAnimationControllerWithThrowingStatusListener(t *testing.T) {
	tt := newTester()
	var reported []error
	tt.reporter = func(err error) { reported = append(reported, err) }
	controller := NewAnimationController(envTester{tt}, dur(ms(100)))
	var log []string
	controller.AddStatusListener(func(AnimationStatus) { log = append(log, "listener1") })
	controller.AddStatusListener(func(AnimationStatus) { log = append(log, "badListener"); panic("bad") })
	controller.AddStatusListener(func(AnimationStatus) { log = append(log, "listener2") })
	controller.Forward()
	if strings.Join(log, ",") != "listener1,badListener,listener2" {
		t.Errorf("log = %v", log)
	}
	if len(reported) != 1 {
		t.Errorf("reported %d errors, want 1", len(reported))
	}
	controller.Dispose()
}

func TestListenerPanicsAreThrownAgainWithoutAReporter(t *testing.T) {
	controller := NewAnimationController(newTester(), dur(ms(100)))
	var log []string
	controller.AddListener(func() { log = append(log, "a"); panic("first") })
	controller.AddListener(func() { log = append(log, "b") })
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("the panic of the listener was swallowed")
			}
		}()
		controller.SetValue(0.2)
	}()
	if strings.Join(log, ",") != "a,b" {
		t.Errorf("log = %v, want every listener called", log)
	}
	statusLog := []string{}
	controller.AddStatusListener(func(AnimationStatus) { statusLog = append(statusLog, "a"); panic("status") })
	controller.AddStatusListener(func(AnimationStatus) { statusLog = append(statusLog, "b") })
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("the panic of the status listener was swallowed")
			}
		}()
		controller.NotifyStatusListeners(completed)
	}()
	if strings.Join(statusLog, ",") != "a,b" {
		t.Errorf("status log = %v", statusLog)
	}
}

// futureLog is the chain of "await" calls of futures_test.dart, as callbacks.
func awaitAll(tt *tester, log *[]string, controllers []*AnimationController, names []string, useOrCancel bool) {
	var step func(i int)
	step = func(i int) {
		*log = append(*log, names[i])
		if i == len(controllers) {
			*log = append(*log, "end")
			return
		}
		f := controllers[i].Forward()
		if useOrCancel {
			// await f.orCancel: continue when it completes, stop when canceled.
			f.WhenCompleteOrCancel(func() {
				if completed, _ := f.Completed(); completed {
					step(i + 1)
				}
			})
			return
		}
		f.Then(func() { step(i + 1) })
	}
	step(0)
}

func TestAwaitingAnimationControllers(t *testing.T) {
	for _, orCancel := range []bool{false, true} {
		tt := newTester()
		c1 := NewAnimationController(tt, dur(ms(100)))
		c2 := NewAnimationController(tt, dur(ms(600)))
		c3 := NewAnimationController(tt, dur(ms(300)))
		var log []string
		log = append(log, "start")
		awaitAll(tt, &log, []*AnimationController{c1, c2, c3}, []string{"a", "b", "c", "d"}, orCancel)
		type pump struct {
			d    int
			want string
		}
		for i, p := range []pump{
			{0, "start,a"}, {0, "start,a"}, {50, "start,a"}, {100, "start,a,b"}, {50, "start,a,b"}, {400, "start,a,b"},
			{199, "start,a,b,c"}, {51, "start,a,b,c"}, {400, "start,a,b,c,d,end"}, {400, "start,a,b,c,d,end"},
		} {
			tt.pump(ms(p.d))
			if got := strings.Join(log, ","); got != p.want {
				t.Errorf("orCancel=%v pump %d: log = %s, want %s", orCancel, i, got, p.want)
			}
		}
		c1.Dispose()
		c2.Dispose()
		c3.Dispose()
	}
}

func TestAwaitingAnimationControllersAndFailing(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(ms(100)))
	var log []string
	log = append(log, "start")
	ch := controller.Forward().OrCancel()
	tt.pump(0)
	tt.pump(ms(50))
	select {
	case err := <-ch:
		t.Fatalf("the future ended early with %v", err)
	default:
	}
	controller.Dispose()
	select {
	case err := <-ch:
		if err == nil {
			t.Fatal("the future completed, want canceled")
		}
		log = append(log, "caught")
	default:
		t.Fatal("the future did not end when the controller was disposed")
	}
	if strings.Join(log, ",") != "start,caught" {
		t.Errorf("log = %v", log)
	}
}

func TestCreatingOrCancelFutureLater(t *testing.T) {
	tt := newTester()
	controller := NewAnimationController(tt, dur(ms(100)))
	defer controller.Dispose()
	f := controller.Forward()
	tt.pump(0)
	tt.pump(ms(200))
	select {
	case <-f.Done():
	default:
		t.Fatal("the future should have completed")
	}
	if err := <-f.OrCancel(); err != nil {
		t.Errorf("OrCancel after completion = %v, want nil", err)
	}

	f2 := controller.Forward(0)
	tt.pump(0)
	controller.Stop()
	if err := <-f2.OrCancel(); err == nil {
		t.Error("OrCancel after stop should be a canceled error")
	}
}
