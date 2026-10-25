package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// kindsOfWindows pumps a window of each kind with the text "content" in it, and
// returns the context of the content and the view of the window.
var kindsOfWindows = []struct {
	name        string
	maximizable bool
	pump        func(t *testing.T, ts *gluttertest.Tester, bc BuildContext) (content BuildContext, view ui.ViewID)
}{
	{"the first window", true, func(t *testing.T, ts *gluttertest.Tester, _ BuildContext) (BuildContext, ui.ViewID) {
		ts.PumpWidget(ts.Binding.WrapWithWindowScope(text("content")))
		return ts.Element(gluttertest.ByText("content")), ui.ImplicitViewID
	}},
	{"a regular window", true, func(t *testing.T, ts *gluttertest.Tester, bc BuildContext) (BuildContext, ui.ViewID) {
		c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{Title: "Regular"})
		if err != nil {
			t.Fatal(err)
		}
		withWindows(ts, text("main"), NewRegularWindow(c, text("content")))
		return ts.Element(gluttertest.ByText("content")), c.RootView().ViewID()
	}},
	{"a dialog", false, func(t *testing.T, ts *gluttertest.Tester, bc BuildContext) (BuildContext, ui.ViewID) {
		c, err := NewDialogWindowController(bc, ui.Size{Width: 300, Height: 200}, DialogWindowControllerOpts{Title: "Dialog"})
		if err != nil {
			t.Fatal(err)
		}
		withWindows(ts, text("main"), NewDialogWindow(c, text("content")))
		return ts.Element(gluttertest.ByText("content")), c.RootView().ViewID()
	}},
}

func actionKinds(ts *gluttertest.Tester, view ui.ViewID) []ui.WindowActionKind {
	var out []ui.WindowActionKind
	for _, a := range ts.ViewActions(view) {
		out = append(out, a.Kind)
	}
	return out
}

func hasKind(kinds []ui.WindowActionKind, want ui.WindowActionKind) bool {
	for _, k := range kinds {
		if k == want {
			return true
		}
	}
	return false
}

func TestWindowControllerOfReachesWhatEveryWindowCanDo(t *testing.T) {
	for _, kind := range kindsOfWindows {
		t.Run(kind.name, func(t *testing.T) {
			ts, bc := windowApp(t)
			content, view := kind.pump(t, ts, bc)
			c := WindowControllerOf(content)
			if WindowControllerMaybeOf(content) == nil || c.RootView().ViewID() != view {
				t.Fatalf("the controller is the one of view %d, want %d", c.RootView().ViewID(), view)
			}
			if c != WindowScopeOf(content).(WindowController) {
				t.Error("WindowControllerOf is not the controller of WindowScopeOf")
			}

			c.SetTitle("New title")
			if c.Title() != "New title" || ts.ViewTitle(view) != "New title" {
				t.Errorf("title = %q, the platform has %q", c.Title(), ts.ViewTitle(view))
			}
			c.Activate()
			if !hasKind(actionKinds(ts, view), ui.WindowActionActivate) {
				t.Errorf("Activate asked the platform for %v", actionKinds(ts, view))
			}
			c.SetMinimized(true)
			if !c.IsMinimized() || !hasKind(actionKinds(ts, view), ui.WindowActionMinimize) {
				t.Errorf("minimized = %v, actions %v", c.IsMinimized(), actionKinds(ts, view))
			}
			ts.SetWindowInfo(view, ui.WindowInfo{Activated: true})
			if !c.IsActivated() {
				t.Error("IsActivated is false after the platform said the window is active")
			}

			m, ok := c.(MaximizableWindowController)
			if ok != kind.maximizable {
				t.Fatalf("is a MaximizableWindowController = %v, want %v", ok, kind.maximizable)
			}
			if !ok {
				return
			}
			m.SetMaximized(true)
			m.SetFullscreen(true)
			got := actionKinds(ts, view)
			if !hasKind(got, ui.WindowActionMaximize) || !hasKind(got, ui.WindowActionFullscreen) {
				t.Errorf("actions = %v, want a maximize and a fullscreen", got)
			}
			ts.SetWindowInfo(view, ui.WindowInfo{Maximized: true})
			if !m.IsMaximized() || m.IsFullscreen() {
				t.Errorf("maximized %v fullscreen %v", m.IsMaximized(), m.IsFullscreen())
			}
			ts.SetWindowInfo(view, ui.WindowInfo{Fullscreen: true})
			if !m.IsFullscreen() {
				t.Error("IsFullscreen is false after the platform said so")
			}
		})
	}
}

func TestWindowControllerOfWithoutAScope(t *testing.T) {
	_, bc := windowApp(t)
	if WindowControllerMaybeOf(bc) != nil {
		t.Error("a controller without a scope")
	}
	requireTag(t, "WindowScopeMissing", func() { WindowControllerOf(bc) })
}

func TestARegularWindowIsCreatedInItsState(t *testing.T) {
	tests := []struct {
		name  string
		state WindowState
		want  ui.WindowStartState
	}{
		{"default", WindowStateUnspecified, ui.WindowStartStateNormal},
		{"normal", WindowStateNormal, ui.WindowStartStateNormal},
		{"maximized", WindowStateMaximized, ui.WindowStartStateMaximized},
		{"minimized", WindowStateMinimized, ui.WindowStartStateMinimized},
		{"fullscreen", WindowStateFullscreen, ui.WindowStartStateFullscreen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := windowApp(t)
			if _, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{State: tt.state}); err != nil {
				t.Fatal(err)
			}
			if got := ts.Windows()[0].Spec.State; got != tt.want {
				t.Errorf("the window was asked for in the state %d, want %d", got, tt.want)
			}
			// A dialog is always created normal.
			if _, err := NewDialogWindowController(bc, ui.Size{Width: 300, Height: 200}); err != nil {
				t.Fatal(err)
			}
			if got := ts.Windows()[1].Spec.State; got != ui.WindowStartStateUnspecified {
				t.Errorf("a dialog was asked for in the state %d", got)
			}
		})
	}
}

func TestViewTitleOfTheImplicitView(t *testing.T) {
	ts, bc := windowApp(t)
	if got := ts.ViewTitle(ui.ImplicitViewID); got != "" {
		t.Errorf("the title of a view that never got one = %q", got)
	}
	c, ok := WidgetsBindingOf(bc).WindowingOwner().ImplicitWindowController()
	if !ok {
		t.Fatal("no implicit controller")
	}
	c.SetTitle("First")
	if got := ts.ViewTitle(ui.ImplicitViewID); got != "First" {
		t.Errorf("title of the implicit view = %q, want First", got)
	}
	// The windows that Windows lists keep it to themselves.
	if len(ts.Windows()) != 0 {
		t.Errorf("Windows lists %d windows, want only the ones the app created", len(ts.Windows()))
	}
	second, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{Title: "Second"})
	if err != nil {
		t.Fatal(err)
	}
	if got := ts.ViewTitle(second.RootView().ViewID()); got != "Second" {
		t.Errorf("title of a created window = %q, want Second", got)
	}
}
