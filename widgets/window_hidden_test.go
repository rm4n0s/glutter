package widgets_test

import (
	"strconv"
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// hiddenCounter shows a number that only its state knows.
type hiddenCounter struct{ StatefulWidget }

func newHiddenCounter() *hiddenCounter {
	return &hiddenCounter{StatefulWidget: NewStatefulWidgetBase()}
}

func (c *hiddenCounter) CreateState() State { return &hiddenCounterState{} }

type hiddenCounterState struct {
	StateOf[*hiddenCounter]
	n int
}

func (s *hiddenCounterState) Build(BuildContext) Widget { return text("count " + strconv.Itoa(s.n)) }

func TestHidingAWindow(t *testing.T) {
	tests := []struct {
		name       string
		do         func(c *RegularWindowController)
		wantHidden bool
		// wantNotified is how many times the listeners heard of it.
		wantNotified int
		// wantAction is the last window action, or -1 for none.
		wantAction ui.WindowActionKind
	}{
		{"hide", func(c *RegularWindowController) { c.SetHidden(true) }, true, 1, -1},
		{"hide twice", func(c *RegularWindowController) { c.SetHidden(true); c.SetHidden(true) }, true, 1, -1},
		{"show again", func(c *RegularWindowController) { c.SetHidden(true); c.SetHidden(false) }, false, 2, -1},
		{"showing a window that is not hidden does nothing", func(c *RegularWindowController) { c.SetHidden(false) }, false, 0, -1},
		{"activate shows a hidden window first", func(c *RegularWindowController) { c.SetHidden(true); c.Activate() }, false, 2, ui.WindowActionActivate},
		{"unminimizing shows a hidden window", func(c *RegularWindowController) { c.SetHidden(true); c.SetMinimized(false) }, false, 2, ui.WindowActionActivate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := windowApp(t)
			c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
			if err != nil {
				t.Fatal(err)
			}
			withWindows(ts, text("main"), NewRegularWindow(c, text("second")))
			notified := 0
			c.AddListener(func() { notified++ })

			tt.do(c)

			w := ts.Windows()[0]
			if c.IsHidden() != tt.wantHidden || w.Hidden != tt.wantHidden || ts.WindowHidden(w.View) != tt.wantHidden {
				t.Errorf("IsHidden %v, platform hidden %v, want %v", c.IsHidden(), w.Hidden, tt.wantHidden)
			}
			if w.Destroyed || c.RootView().Destroyed() {
				t.Error("hiding destroyed the window")
			}
			if notified != tt.wantNotified {
				t.Errorf("the listeners were told %d times, want %d", notified, tt.wantNotified)
			}
			actions := ts.ViewActions(w.View)
			last := ui.WindowActionKind(-1)
			if len(actions) > 0 {
				last = actions[len(actions)-1].Kind
			}
			if last != tt.wantAction {
				t.Errorf("last action %v, want %v", last, tt.wantAction)
			}
		})
	}
}

func TestADestroyedWindowIsNotHidden(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	c.Destroy()
	c.SetHidden(true)
	if c.IsHidden() || ts.Windows()[0].Hidden {
		t.Error("a destroyed window was hidden")
	}
}

func TestAHiddenWindowKeepsItsWidgetsAndTheirState(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	withWindows(ts, text("main"), NewRegularWindow(c, newHiddenCounter()))
	state := ts.Element(gluttertest.ByType[*hiddenCounter]()).(*StatefulElement).State().(*hiddenCounterState)
	state.SetState(func() { state.n = 7 })
	ts.Pump()

	c.SetHidden(true)
	ts.Pump()
	c.SetHidden(false)
	ts.Pump()

	again := ts.Element(gluttertest.ByType[*hiddenCounter]()).(*StatefulElement).State().(*hiddenCounterState)
	if again != state || again.n != 7 {
		t.Errorf("the state was rebuilt: same %v, n = %d", again == state, again.n)
	}
	if !ts.Exists(gluttertest.ByText("count 7")) {
		t.Error("the window does not show what it showed before it was hidden")
	}
}

func TestHidingEveryWindowHidesTheApp(t *testing.T) {
	ts, bc := windowApp(t)
	ts.SetLifecycleState(ui.AppLifecycleStateResumed)
	owner := WidgetsBindingOf(bc).WindowingOwner()
	main, ok := owner.ImplicitWindowController()
	if !ok {
		t.Fatal("no controller for the first window")
	}
	second, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	withWindows(ts, text("main"), NewRegularWindow(second, text("second")))
	scheduler := ts.Binding.Scheduler()
	steps := []struct {
		name       string
		do         func()
		want       ui.AppLifecycleState
		wantFrames bool
	}{
		{"one of two windows hidden", func() { second.SetHidden(true) }, ui.AppLifecycleStateResumed, true},
		{"both hidden", func() { main.SetHidden(true) }, ui.AppLifecycleStateHidden, false},
		{"one shown again", func() { second.Activate() }, ui.AppLifecycleStateResumed, true},
	}
	for _, s := range steps {
		s.do()
		if got := scheduler.LifecycleState(); got != s.want {
			t.Errorf("%s: lifecycle %v, want %v", s.name, got, s.want)
		}
		if got := scheduler.FramesEnabled(); got != s.wantFrames {
			t.Errorf("%s: frames enabled %v, want %v", s.name, got, s.wantFrames)
		}
	}
	// While every window is hidden a change asks for no frame; showing one does.
	main.SetHidden(true)
	second.SetHidden(true)
	ts.Pump()
	scheduler.ScheduleFrame()
	if scheduler.HasScheduledFrame() {
		t.Error("a frame was scheduled while every window is hidden")
	}
	main.SetHidden(false)
	if !scheduler.HasScheduledFrame() {
		t.Error("showing a window did not schedule a frame")
	}
}

func TestExitingWithTheOnlyWindowHidden(t *testing.T) {
	tests := []struct {
		name string
		exit func(ts *gluttertest.Tester, b *WidgetsBinding) bool
	}{
		{"ExitApplication", func(ts *gluttertest.Tester, b *WidgetsBinding) bool {
			b.ExitApplication(ui.AppExitTypeRequired)
			return ts.ExitRequests() == 1
		}},
		{"a cancelable exit nobody objects to", func(ts *gluttertest.Tester, b *WidgetsBinding) bool {
			return b.ExitApplication(ui.AppExitTypeCancelable) == ui.AppExitResponseExit && ts.ExitRequests() == 1
		}},
		{"the platform closes the window", func(ts *gluttertest.Tester, _ *WidgetsBinding) bool {
			return ts.RequestWindowClose(ui.ImplicitViewID)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := windowApp(t)
			b := WidgetsBindingOf(bc)
			c, _ := b.WindowingOwner().ImplicitWindowController()
			c.SetHidden(true)
			if !ts.WindowHidden(ui.ImplicitViewID) {
				t.Fatal("the first window is not hidden")
			}
			if !tt.exit(ts, b) {
				t.Error("the app did not exit")
			}
		})
	}
}
