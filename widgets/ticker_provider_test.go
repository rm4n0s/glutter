package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/ticker_provider_test.dart; the
// toString tests are replaced by checks of what the mixin keeps (SPEC §12.10).

func transientCount(ts *gluttertest.Tester) int {
	return ts.Binding.Scheduler().TransientCallbackCount()
}

func dur(d time.Duration) animation.AnimationControllerOpts {
	return animation.AnimationControllerOpts{Duration: foundation.Some(d)}
}

// tickerWidget is _TickerWidget: a repeating controller.
type tickerWidget struct{ StatefulWidget }

func newTickerWidget() *tickerWidget { return &tickerWidget{NewStatefulWidgetBase()} }

func (w *tickerWidget) CreateState() State { return &tickerWidgetState{} }

type tickerWidgetState struct {
	StateOf[*tickerWidget]
	SingleTickerProvider
	controller *animation.AnimationController
}

func (s *tickerWidgetState) InitState() {
	s.controller = animation.NewAnimationController(s, dur(time.Second))
	s.controller.Repeat()
}
func (s *tickerWidgetState) Dispose()                  { s.controller.Dispose() }
func (s *tickerWidgetState) Build(BuildContext) Widget { return newLeaf() }

func TestNavigationWithTickerMode(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(navTestApp(navTestAppOpts{
		home:   newTickerWidget(),
		routes: map[string]WidgetBuilder{"/test": func(BuildContext) Widget { return NewText("hello") }},
	}))
	expect := func(want int) {
		t.Helper()
		if n := transientCount(ts); n != want {
			t.Fatalf("transient callbacks = %d, want %d", n, want)
		}
	}
	expect(1)
	state := gluttertest.StateOf[*NavigatorState](ts, gluttertest.ByType[*Navigator]())
	state.PushNamed("/test")
	expect(2)
	ts.Pump()
	expect(2)
	ts.Pump(5 * time.Second)
	expect(0)
	state.Pop()
	expect(1)
	ts.Pump()
	expect(2)
	ts.Pump(5 * time.Second)
	expect(1)
}

func TestTickerMode(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewTickerMode(false, newTickerWidget()))
	if n := transientCount(ts); n != 0 {
		t.Errorf("disabled: transient callbacks = %d, want 0", n)
	}
	ts.PumpWidget(NewTickerMode(true, newTickerWidget()))
	if n := transientCount(ts); n != 1 {
		t.Errorf("enabled: transient callbacks = %d, want 1", n)
	}
	ts.PumpWidget(NewTickerMode(false, newTickerWidget()))
	if n := transientCount(ts); n != 0 {
		t.Errorf("disabled again: transient callbacks = %d, want 0", n)
	}
}

func TestTickerModeMergesWithTheOnesAbove(t *testing.T) {
	ts := gluttertest.New(t)
	var seen []TickerModeData
	probe := builderOf(func(bc BuildContext) Widget {
		seen = append(seen, TickerModeValuesOf(bc))
		return newLeaf()
	})
	ts.PumpWidget(NewTickerMode(false, NewTickerModeMerge(probe, TickerModeMergeOpts{ForceFrames: foundation.Some(true)})))
	if len(seen) != 1 || seen[0] != (TickerModeData{Enabled: false, ForceFrames: true}) {
		t.Errorf("mode = %v, want disabled with forced frames", seen)
	}
	seen = nil
	ts.PumpWidget(NewTickerMode(true, NewTickerMode(true, probe, TickerModeOpts{ForceFrames: true})))
	if len(seen) != 1 || seen[0] != (TickerModeData{Enabled: true, ForceFrames: true}) {
		t.Errorf("mode = %v, want enabled with forced frames", seen)
	}
	seen = nil
	ts.PumpWidget(probe)
	if len(seen) != 1 || seen[0] != TickerModeDataFallback {
		t.Errorf("mode = %v, want the fallback", seen)
	}
}

func TestTickerModeMutesAnActiveTickerAndForcesFrames(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewTickerMode(true, newTickerWidget(), TickerModeOpts{ForceFrames: true}))
	state := gluttertest.StateOf[*tickerWidgetState](ts, gluttertest.ByType[*tickerWidget]())
	if state.controller.IsAnimating() != true {
		t.Fatal("the controller should be animating")
	}
	ts.PumpWidget(NewTickerMode(false, newTickerWidget(), TickerModeOpts{ForceFrames: true}))
	if !state.controller.IsAnimating() {
		t.Error("a muted ticker is still active")
	}
	if n := transientCount(ts); n != 0 {
		t.Errorf("muted: transient callbacks = %d, want 0", n)
	}
}

// boringTickerTest is BoringTickerTest: a provider that is never used.
type boringTickerTest struct{ StatefulWidget }

func (w *boringTickerTest) CreateState() State { return &boringTickerTestState{} }

type boringTickerTestState struct {
	StateOf[*boringTickerTest]
	SingleTickerProvider
}

func (s *boringTickerTestState) Build(BuildContext) Widget { return newLeaf() }

func TestSingleTickerProviderCanHandleNotBeingUsed(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(&boringTickerTest{NewStatefulWidgetBase()})
	ts.PumpWidget(newLeaf())
}

// singleTickerTest is _SingleTickerTest.
type singleTickerTest struct{ StatefulWidget }

func newSingleTickerTest(opts ...WidgetOpts) *singleTickerTest {
	return &singleTickerTest{NewStatefulWidgetBase(opts...)}
}

func (w *singleTickerTest) CreateState() State { return &singleTickerTestState{} }

type singleTickerTestState struct {
	StateOf[*singleTickerTest]
	SingleTickerProvider
	controller *animation.AnimationController
}

func (s *singleTickerTestState) InitState() {
	s.controller = animation.NewAnimationController(s, dur(100*time.Second))
}
func (s *singleTickerTestState) Build(BuildContext) Widget { return newLeaf() }

// multipleTickerTest is _MultipleTickerTest.
type multipleTickerTest struct{ StatefulWidget }

func newMultipleTickerTest(opts ...WidgetOpts) *multipleTickerTest {
	return &multipleTickerTest{NewStatefulWidgetBase(opts...)}
}

func (w *multipleTickerTest) CreateState() State { return &multipleTickerTestState{} }

type multipleTickerTestState struct {
	StateOf[*multipleTickerTest]
	TickerProvider
	controllers []*animation.AnimationController
}

func (s *multipleTickerTestState) InitState() {
	s.controllers = append(s.controllers, animation.NewAnimationController(s, dur(100*time.Second)),
		animation.NewAnimationController(s, dur(100*time.Second)))
}
func (s *multipleTickerTestState) Build(BuildContext) Widget { return newLeaf() }

// singleTickerCreateMultipleTicker is _SingleTickerCreateMultipleTicker.
type singleTickerCreateMultipleTicker struct{ StatefulWidget }

func (w *singleTickerCreateMultipleTicker) CreateState() State {
	return &singleTickerCreateMultipleTickerState{}
}

type singleTickerCreateMultipleTickerState struct {
	StateOf[*singleTickerCreateMultipleTicker]
	SingleTickerProvider
}

func (s *singleTickerCreateMultipleTickerState) InitState() {
	animation.NewAnimationController(s, dur(5*time.Second))
	animation.NewAnimationController(s, dur(6*time.Second))
}
func (s *singleTickerCreateMultipleTickerState) Build(BuildContext) Widget { return newLeaf() }

func TestSingleTickerProviderCreatingMultipleTickersPanics(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(&singleTickerCreateMultipleTicker{NewStatefulWidgetBase()})
	_ = requireErrorTag(t, ts.TakeError(), "SingleTickerProviderMultipleTickers")
}

func TestSingleTickerProviderDisposeWhileActive(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	key := NewGlobalKey[*singleTickerTestState]()
	ts.PumpWidget(newSingleTickerTest(WidgetOpts{Key: key}))
	state, _ := key.CurrentState()
	state.controller.Repeat()
	// Unmounting the state disposes the mixin while the ticker is active.
	ts.PumpWidget(newLeaf())
	appErr := requireErrorTag(t, ts.TakeError(), "TickerProviderActiveTicker")
	if appErr.Metadata["ticker"] == nil {
		t.Errorf("the error does not name the ticker: %v", appErr.Metadata)
	}
	state.controller.Stop()
}

func TestTickerProviderDisposeWhileAnyTickerIsActive(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	key := NewGlobalKey[*multipleTickerTestState]()
	ts.PumpWidget(newMultipleTickerTest(WidgetOpts{Key: key}))
	state, _ := key.CurrentState()
	state.controllers[0].Repeat()
	ts.PumpWidget(newLeaf())
	_ = requireErrorTag(t, ts.TakeError(), "TickerProviderActiveTicker")
	state.controllers[0].Stop()
}

// lateDispose is _LateDispose and _LateDisposeSingle: the controller is made
// after the first use and disposed in Dispose, even if it was never made.
type lateDispose struct {
	StatefulWidget
	single bool
}

func (w *lateDispose) CreateState() State {
	if w.single {
		return &lateDisposeSingleState{}
	}
	return &lateDisposeState{}
}

type lateDisposeState struct {
	StateOf[*lateDispose]
	TickerProvider
	controller *animation.AnimationController
}

func (s *lateDisposeState) lazy() *animation.AnimationController {
	if s.controller == nil {
		s.controller = animation.NewAnimationController(s)
	}
	return s.controller
}
func (s *lateDisposeState) Dispose()                  { s.lazy().Dispose() }
func (s *lateDisposeState) Build(BuildContext) Widget { return newLeaf() }

type lateDisposeSingleState struct {
	StateOf[*lateDispose]
	SingleTickerProvider
	controller *animation.AnimationController
}

func (s *lateDisposeSingleState) lazy() *animation.AnimationController {
	if s.controller == nil {
		s.controller = animation.NewAnimationController(s)
	}
	return s.controller
}
func (s *lateDisposeSingleState) Dispose()                  { s.lazy().Dispose() }
func (s *lateDisposeSingleState) Build(BuildContext) Widget { return newLeaf() }

func TestTickerProvidersDoNotFailForALateControllerMember(t *testing.T) {
	for _, single := range []bool{true, false} {
		ts := gluttertest.New(t)
		ts.PumpWidget(&lateDispose{StatefulWidget: NewStatefulWidgetBase(), single: single})
		ts.PumpWidget(newLeaf())
		if err := ts.TakeError(); err != nil {
			t.Errorf("single=%v: unexpected error %v", single, err)
		}
	}
}

func TestSingleTickerProviderIsTheEnvironmentOfItsControllers(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(newSingleTickerTest())
	state := gluttertest.StateOf[*singleTickerTestState](ts, gluttertest.ByType[*singleTickerTest]())
	var env animation.Environment = &state.SingleTickerProvider
	if env.DisableAnimations() {
		t.Error("animations should be enabled by default")
	}
	ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.AccessibilityFeatures.DisableAnimations = true })
	if !env.DisableAnimations() {
		t.Error("animations should be disabled when the system asks")
	}
	if env.ErrorReporter() == nil {
		t.Error("the environment should report to the binding")
	}
	var _ scheduler.TickerProvider = &state.SingleTickerProvider
}

func TestTickerProviderRemovesDisposedTickers(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(newMultipleTickerTest())
	state := gluttertest.StateOf[*multipleTickerTestState](ts, gluttertest.ByType[*multipleTickerTest]())
	c := animation.NewAnimationController(state, dur(time.Second))
	c.Repeat()
	c.Dispose() // an active ticker that is disposed is not a leak
	ts.PumpWidget(newLeaf())
	// The two controllers made by InitState were never active.
	if err := ts.TakeError(); err != nil {
		t.Errorf("unexpected error %v", err)
	}
}
