package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// probe is a stateful widget that tells when its state is disposed.
type probe struct {
	StatefulWidget
	disposed *bool
}

func newProbe(disposed *bool) *probe { return &probe{NewStatefulWidgetBase(), disposed} }

func (p *probe) CreateState() State { return &probeState{} }

type probeState struct{ StateOf[*probe] }

func (s *probeState) Dispose() { *s.Widget().disposed = true }

func (s *probeState) Build(BuildContext) Widget { return text("first window") }

// closeDelegate records the close requests of a window and what it destroyed.
type closeDelegate struct {
	RegularWindowControllerDelegateBase
	requests  []*RegularWindowController
	destroyed int
	// close makes the delegate destroy the window it was asked about.
	close bool
}

func (d *closeDelegate) OnWindowCloseRequested(c *RegularWindowController) {
	d.requests = append(d.requests, c)
	if d.close {
		c.Destroy()
	}
}

func (d *closeDelegate) OnWindowDestroyed() { d.destroyed++ }

// twoWindows pumps the first window, with a probe in it, and a second window.
func twoWindows(t *testing.T, disposed *bool) (ts *gluttertest.Tester, first *RegularWindowController, second *RegularWindowController) {
	t.Helper()
	ts, bc := windowApp(t)
	first, ok := WidgetsBindingOf(bc).WindowingOwner().ImplicitWindowController()
	if !ok {
		t.Fatal("no implicit controller")
	}
	second, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	ts.PumpWidget(NewViewCollection([]Widget{NewView(ts.View, newProbe(disposed)), NewRegularWindow(second, text("second"))}),
		gluttertest.PumpWidgetOpts{NoView: true})
	return ts, first, second
}

func TestTheCloseRequestOfTheFirstWindowReachesItsDelegate(t *testing.T) {
	tests := []struct {
		name      string
		delegate  func() *closeDelegate // nil: no delegate
		wantAllow bool                  // whether the engine would close the window
		wantExit  bool                  // whether the app's exit observers were asked
	}{
		{"without a delegate it is a request to exit the app", nil, true, true},
		{"a delegate that keeps the window", func() *closeDelegate { return &closeDelegate{} }, false, false},
		{"a delegate that destroys the window", func() *closeDelegate { return &closeDelegate{close: true} }, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := windowApp(t)
			owner := WidgetsBindingOf(bc).WindowingOwner()
			first, _ := owner.ImplicitWindowController()
			var d *closeDelegate
			if tt.delegate != nil {
				d = tt.delegate()
				owner.SetImplicitWindowDelegate(d)
			}
			exits := 0
			listener := NewAppLifecycleListener(WidgetsBindingOf(bc), AppLifecycleListenerOpts{OnExitRequested: func() ui.AppExitResponse { exits++; return ui.AppExitResponseExit }})
			defer listener.Dispose()

			if got := ts.RequestWindowClose(ui.ImplicitViewID); got != tt.wantAllow {
				t.Errorf("the engine may close the window = %v, want %v", got, tt.wantAllow)
			}
			if (exits > 0) != tt.wantExit {
				t.Errorf("the exit observers were asked %d times, want asked = %v", exits, tt.wantExit)
			}
			if d != nil {
				if len(d.requests) != 1 || d.requests[0] != first || d.requests[0].RootView().ViewID() != ui.ImplicitViewID {
					t.Errorf("the delegate got %v, want one request for the first window", d.requests)
				}
				if (d.destroyed == 1) != d.close {
					t.Errorf("the delegate heard of the destruction %d times, want it once = %v", d.destroyed, d.close)
				}
			}
		})
	}
}

func TestTheDelegateOfTheFirstWindowCanBeTakenAway(t *testing.T) {
	ts, bc := windowApp(t)
	owner := WidgetsBindingOf(bc).WindowingOwner()
	d := &closeDelegate{}
	owner.SetImplicitWindowDelegate(d)
	owner.SetImplicitWindowDelegate(nil)
	if !ts.RequestWindowClose(ui.ImplicitViewID) || len(d.requests) != 0 {
		t.Errorf("a delegate that was taken away still got the request: %v", d.requests)
	}
}

func TestDestroyingTheFirstWindowKeepsTheOthers(t *testing.T) {
	var disposed bool
	ts, first, second := twoWindows(t, &disposed)
	first.Destroy()
	ts.Pump() // the View rebuilds in the next frame
	if !disposed {
		t.Fatal("the states of the first window were not disposed")
	}
	if ts.Dispatcher().View(ui.ImplicitViewID) != nil {
		t.Error("the implicit view is still in the dispatcher")
	}
	if ts.Exists(gluttertest.ByText("first window")) {
		t.Error("the content of the first window is still in the tree")
	}
	if !ts.Exists(gluttertest.ByText("second")) {
		t.Fatal("the second window lost its content")
	}
	// Frames go on for the second window.
	before := len(ts.ViewScenes(second.RootView().ViewID()))
	second.SetTitle("still alive")
	ts.PumpWidget(NewViewCollection([]Widget{NewView(ts.View, newProbe(&disposed)), NewRegularWindow(second, text("second again"))}),
		gluttertest.PumpWidgetOpts{NoView: true})
	if after := len(ts.ViewScenes(second.RootView().ViewID())); after <= before || !ts.Exists(gluttertest.ByText("second again")) {
		t.Errorf("the second window drew %d scenes after %d, and shows %v", after, before, ts.Exists(gluttertest.ByText("second again")))
	}
	if ts.ViewTitle(second.RootView().ViewID()) != "still alive" {
		t.Error("the second window cannot be changed any more")
	}
}

func TestDestroyingTheFirstWindowTellsItsDelegate(t *testing.T) {
	var disposed bool
	ts, first, _ := twoWindows(t, &disposed)
	d := &closeDelegate{}
	WidgetsBindingOf(ts.Element(gluttertest.ByText("second"))).WindowingOwner().SetImplicitWindowDelegate(d)
	first.Destroy()
	first.Destroy() // twice is once
	ts.Pump()
	if d.destroyed != 1 || !disposed {
		t.Errorf("the delegate heard %d times, states disposed = %v", d.destroyed, disposed)
	}
}

func TestTheFirstWindowClosedByThePlatformLeavesTheTree(t *testing.T) {
	var disposed bool
	ts, first, _ := twoWindows(t, &disposed)
	d := &closeDelegate{}
	WidgetsBindingOf(ts.Element(gluttertest.ByText("second"))).WindowingOwner().SetImplicitWindowDelegate(d)
	// The engine closed the window (the user chose to exit and the app allowed it).
	ts.Dispatcher().RemoveView(ui.ImplicitViewID)
	ts.Pump()
	if !disposed || ts.Exists(gluttertest.ByText("first window")) {
		t.Errorf("states disposed = %v, content still shown = %v", disposed, ts.Exists(gluttertest.ByText("first window")))
	}
	if d.destroyed != 1 {
		t.Errorf("the delegate heard of the destruction %d times, want 1", d.destroyed)
	}
	_ = first
}

func TestDestroyingTheFirstWindowAlone(t *testing.T) {
	ts, bc := windowApp(t)
	var disposed bool
	first, _ := WidgetsBindingOf(bc).WindowingOwner().ImplicitWindowController()
	ts.PumpWidget(newProbe(&disposed))
	first.Destroy()
	ts.Pump()
	if !disposed {
		t.Error("the states of the only window were not disposed")
	}
	if ts.Dispatcher().View(ui.ImplicitViewID) != nil {
		t.Error("the implicit view is still in the dispatcher")
	}
}
