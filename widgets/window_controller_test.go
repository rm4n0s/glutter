package widgets_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// text is a Text in a left-to-right Directionality.
func text(s string) Widget { return NewDirectionality(ui.TextDirectionLTR, NewText(s)) }

// windowApp pumps an app with the implicit view and returns the tester and a
// BuildContext in it.
func windowApp(t *testing.T) (*gluttertest.Tester, BuildContext) {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(text("main"))
	return ts, ts.Element(gluttertest.ByText("main"))
}

// withWindows replaces the root with a collection of the implicit view and the
// windows, and pumps.
func withWindows(ts *gluttertest.Tester, main Widget, windows ...Widget) {
	views := append([]Widget{NewView(ts.View, main)}, windows...)
	ts.PumpWidget(NewViewCollection(views), gluttertest.PumpWidgetOpts{NoView: true})
}

func routeOf(t *testing.T, err error, want string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(want) {
		t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), want)
	}
}

func TestRegularWindowShowsItsChildInItsOwnView(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{Title: "Second"})
	if err != nil {
		t.Fatal(err)
	}
	withWindows(ts, text("main"), NewRegularWindow(c, text("second")))

	ws := ts.Windows()
	if len(ws) != 1 || ws[0].Spec.Kind != ui.WindowKindRegular || ws[0].Title != "Second" || ws[0].Size != (ui.Size{Width: 400, Height: 300}) {
		t.Fatalf("windows = %+v", ws)
	}
	if ws[0].Spec.HasParent {
		t.Error("a regular window has no parent")
	}
	if c.RootView().ViewID() == ui.ImplicitViewID {
		t.Error("the window shares the implicit view")
	}
	if !ts.Exists(gluttertest.ByText("second")) || !ts.Exists(gluttertest.ByText("main")) {
		t.Fatal("both windows show their content")
	}
	// The content of the second window sees the second view.
	second := ts.Element(gluttertest.ByText("second"))
	if got := MediaQuerySizeOf(second); got != (ui.Size{Width: 400, Height: 300}) {
		t.Errorf("the media of the second window = %v", got)
	}
	if got := ViewOf(second); got != c.RootView() {
		t.Error("ViewOf is not the view of the controller")
	}
	if got := MediaQuerySizeOf(ts.Element(gluttertest.ByText("main"))); got != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("the media of the main window = %v", got)
	}
	if n := len(ts.ViewScenes(c.RootView().ViewID())); n == 0 {
		t.Error("the second view rendered no scene")
	}
	if n := len(ts.ViewScenes(ui.ImplicitViewID)); n == 0 {
		t.Error("the implicit view rendered no scene")
	}
}

func TestWindowControllerConstraints(t *testing.T) {
	tests := []struct {
		name     string
		c        rendering.BoxConstraints
		min, max ui.Size
	}{
		{"bounded", rendering.BoxConstraints{MinWidth: 200, MaxWidth: 900, MinHeight: 100, MaxHeight: 700},
			ui.Size{Width: 200, Height: 100}, ui.Size{Width: 900, Height: 700}},
		{"only a minimum", rendering.BoxConstraints{MinWidth: 320, MaxWidth: math.Inf(1), MinHeight: 240, MaxHeight: math.Inf(1)},
			ui.Size{Width: 320, Height: 240}, ui.Size{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := windowApp(t)
			c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300},
				RegularWindowControllerOpts{Constraints: foundation.Some(tt.c)})
			if err != nil {
				t.Fatal(err)
			}
			if w := ts.Windows()[0]; w.Spec.MinSize != tt.min || w.Spec.MaxSize != tt.max {
				t.Errorf("limits = %v %v, want %v %v", w.Spec.MinSize, w.Spec.MaxSize, tt.min, tt.max)
			}
			c.SetConstraints(rendering.BoxConstraints{MinWidth: 250, MaxWidth: 500, MinHeight: 150, MaxHeight: 400})
			if w := ts.Windows()[0]; w.Min != (ui.Size{Width: 250, Height: 150}) || w.Max != (ui.Size{Width: 500, Height: 400}) {
				t.Errorf("limits after SetConstraints = %v %v", w.Min, w.Max)
			}
		})
	}
}

func TestWindowSizeMustSatisfyTheConstraints(t *testing.T) {
	skipUnlessDebug(t)
	_, bc := windowApp(t)
	defer func() {
		r := recover()
		err, _ := r.(error)
		if err == nil {
			t.Fatalf("recovered %v, want an error", r)
		}
		routeOf(t, err, "WindowSizeViolatesConstraints")
	}()
	_, _ = NewRegularWindowController(bc, ui.Size{Width: 100, Height: 100}, RegularWindowControllerOpts{
		Constraints: foundation.Some(rendering.BoxConstraints{MinWidth: 200, MaxWidth: 400, MinHeight: 200, MaxHeight: 400}),
	})
}

func TestCreatingAWindowFailsWhenThePlatformCannot(t *testing.T) {
	ts, bc := windowApp(t)
	ts.SetWindowCreateError(errors.New(tagWindowingUnsupported, "no windows here"))
	_, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	routeOf(t, err, "WindowingUnsupported")
	_, err = NewDialogWindowController(bc, ui.Size{Width: 400, Height: 300})
	routeOf(t, err, "WindowingUnsupported")
	if n := len(ts.Windows()); n != 0 {
		t.Errorf("%d windows exist", n)
	}
}

// recordingDelegate counts the calls of a regular window's delegate.
type recordingDelegate struct {
	RegularWindowControllerDelegateBase
	closeRequests, destroyed int
	allowClose               bool
}

func (d *recordingDelegate) OnWindowCloseRequested(c *RegularWindowController) {
	d.closeRequests++
	if d.allowClose {
		c.Destroy()
	}
}
func (d *recordingDelegate) OnWindowDestroyed() { d.destroyed++ }

func TestDestroyingAWindow(t *testing.T) {
	ts, bc := windowApp(t)
	d := &recordingDelegate{}
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{Delegate: d})
	if err != nil {
		t.Fatal(err)
	}
	withWindows(ts, text("main"), NewRegularWindow(c, text("second")))
	notified := 0
	c.AddListener(func() { notified++ })
	before := len(ts.ViewScenes(c.RootView().ViewID()))

	c.Destroy()
	c.Destroy() // it is allowed to destroy twice

	if !ts.Windows()[0].Destroyed {
		t.Error("the platform window was not destroyed")
	}
	if d.destroyed != 1 {
		t.Errorf("OnWindowDestroyed was called %d times, want once", d.destroyed)
	}
	if notified == 0 {
		t.Error("the listeners were not told")
	}
	if !c.RootView().Destroyed() {
		t.Error("the view is not marked destroyed")
	}
	// The widget stays in the tree until the app removes it, and renders nothing.
	ts.Dispatcher().ScheduleFrame()
	ts.Pump()
	if after := len(ts.ViewScenes(c.RootView().ViewID())); after != before {
		t.Errorf("the destroyed view rendered %d more scenes", after-before)
	}
	// Asking a destroyed window for anything does nothing.
	c.SetTitle("late")
	c.SetSize(ui.Size{Width: 1, Height: 1})
	c.SetMaximized(true)
	if w := ts.Windows()[0]; w.Title == "late" || len(ts.ViewActions(w.View)) != 0 {
		t.Errorf("a destroyed window was changed: %+v", w)
	}
}

func TestClosingAWindowGoesThroughItsDelegate(t *testing.T) {
	tests := []struct {
		name       string
		allowClose bool
		wantGone   bool
	}{
		{"the delegate destroys the window", true, true},
		{"the delegate keeps the window", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := windowApp(t)
			d := &recordingDelegate{allowClose: tt.allowClose}
			c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{Delegate: d})
			if err != nil {
				t.Fatal(err)
			}
			view := c.RootView().ViewID()
			// The engine leaves the window open: the delegate decides.
			if ts.RequestWindowClose(view) {
				t.Error("the platform would close the window by itself")
			}
			if d.closeRequests != 1 {
				t.Errorf("the delegate got %d requests, want 1", d.closeRequests)
			}
			if got := ts.Windows()[0].Destroyed; got != tt.wantGone {
				t.Errorf("destroyed = %v, want %v", got, tt.wantGone)
			}
		})
	}
}

func TestTheDefaultDelegateDestroysTheWindow(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	ts.RequestWindowClose(c.RootView().ViewID())
	if !ts.Windows()[0].Destroyed {
		t.Error("the window is still there")
	}
}

type panickingDelegate struct {
	RegularWindowControllerDelegateBase
}

func (panickingDelegate) OnWindowCloseRequested(*RegularWindowController) { panic("delegate failed") }

func TestAPanicInTheDelegateIsReported(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{Delegate: panickingDelegate{}})
	if err != nil {
		t.Fatal(err)
	}
	ts.RequestWindowClose(c.RootView().ViewID())
	routeOf(t, ts.TakeError(), "CallbackPanicked")
}

func TestTheFirstWindowClosesThroughTheExitRequest(t *testing.T) {
	ts, bc := windowApp(t)
	c, ok := WidgetsBindingOf(bc).WindowingOwner().ImplicitWindowController()
	if !ok {
		t.Fatal("the implicit view has no controller")
	}
	if c.RootView() != ts.View {
		t.Error("the controller is not the one of the implicit view")
	}
	if again, _ := WidgetsBindingOf(bc).WindowingOwner().ImplicitWindowController(); again != c {
		t.Error("the implicit controller is made twice")
	}
	// Nobody objects to the exit: the engine closes the window by itself.
	if !ts.RequestWindowClose(ui.ImplicitViewID) {
		t.Error("the first window was not allowed to close")
	}
	if len(ts.Windows()) != 0 {
		t.Error("the implicit window is not one that the app asked for")
	}
}

func TestADialogIsModalToItsParent(t *testing.T) {
	ts, bc := windowApp(t)
	parent, ok := WidgetsBindingOf(bc).WindowingOwner().ImplicitWindowController()
	if !ok {
		t.Fatal("no implicit controller")
	}
	d, err := NewDialogWindowController(bc, ui.Size{Width: 300, Height: 200},
		DialogWindowControllerOpts{Parent: parent, Title: "Dialog"})
	if err != nil {
		t.Fatal(err)
	}
	w := ts.Windows()[0]
	if w.Spec.Kind != ui.WindowKindDialog || !w.Spec.HasParent || w.Spec.Parent != ui.ImplicitViewID || w.Title != "Dialog" {
		t.Errorf("window = %+v", w)
	}
	if d.Parent() != BaseWindowController(parent) {
		t.Error("the parent is not the controller that was given")
	}
	withWindows(ts, text("main"), NewDialogWindow(d, text("dialog")))
	if !ts.Exists(gluttertest.ByText("dialog")) {
		t.Error("the dialog shows no content")
	}
}

func TestAModelessDialogHasNoParent(t *testing.T) {
	ts, bc := windowApp(t)
	if _, err := NewDialogWindowController(bc, ui.Size{Width: 300, Height: 200}); err != nil {
		t.Fatal(err)
	}
	if w := ts.Windows()[0]; w.Spec.HasParent || w.Spec.Kind != ui.WindowKindDialog {
		t.Errorf("window = %+v", w)
	}
}

func TestADialogClosesWithItsParent(t *testing.T) {
	ts, bc := windowApp(t)
	parent, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	d1, err := NewDialogWindowController(bc, ui.Size{Width: 300, Height: 200}, DialogWindowControllerOpts{Parent: parent})
	if err != nil {
		t.Fatal(err)
	}
	d2, err := NewDialogWindowController(bc, ui.Size{Width: 300, Height: 200}, DialogWindowControllerOpts{Parent: d1})
	if err != nil {
		t.Fatal(err)
	}
	modeless, err := NewDialogWindowController(bc, ui.Size{Width: 300, Height: 200})
	if err != nil {
		t.Fatal(err)
	}

	parent.Destroy()

	got := map[ui.ViewID]bool{}
	for _, w := range ts.Windows() {
		got[w.View] = w.Destroyed
	}
	for name, c := range map[string]BaseWindowController{"parent": parent, "child": d1, "grandchild": d2} {
		if !got[c.RootView().ViewID()] {
			t.Errorf("the %s was not destroyed", name)
		}
	}
	if got[modeless.RootView().ViewID()] {
		t.Error("a dialog with no parent closed with another window")
	}
}

func TestADialogCannotHaveADestroyedParent(t *testing.T) {
	_, bc := windowApp(t)
	parent, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	parent.Destroy()
	_, err = NewDialogWindowController(bc, ui.Size{Width: 300, Height: 200}, DialogWindowControllerOpts{Parent: parent})
	routeOf(t, err, "WindowParentDestroyed")
}

func TestWindowRequests(t *testing.T) {
	tests := []struct {
		name string
		do   func(c *RegularWindowController)
		want ui.WindowActionKind
	}{
		{"maximize", func(c *RegularWindowController) { c.SetMaximized(true) }, ui.WindowActionMaximize},
		{"restore", func(c *RegularWindowController) { c.SetMaximized(false) }, ui.WindowActionRestore},
		{"fullscreen", func(c *RegularWindowController) { c.SetFullscreen(true) }, ui.WindowActionFullscreen},
		{"leave fullscreen", func(c *RegularWindowController) { c.SetFullscreen(false) }, ui.WindowActionUnfullscreen},
		{"minimize", func(c *RegularWindowController) { c.SetMinimized(true) }, ui.WindowActionMinimize},
		{"unminimize activates", func(c *RegularWindowController) { c.SetMinimized(false) }, ui.WindowActionActivate},
		{"activate", func(c *RegularWindowController) { c.Activate() }, ui.WindowActionActivate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := windowApp(t)
			c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
			if err != nil {
				t.Fatal(err)
			}
			tt.do(c)
			got := ts.ViewActions(c.RootView().ViewID())
			if len(got) != 1 || got[0].Kind != tt.want {
				t.Errorf("actions = %+v, want one %v", got, tt.want)
			}
			if len(ts.ViewActions(ui.ImplicitViewID)) != 0 {
				t.Error("the main window got the action")
			}
		})
	}
}

func TestTitleAndSizeRequests(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{Title: "One"})
	if err != nil {
		t.Fatal(err)
	}
	c.SetTitle("Two")
	c.SetSize(ui.Size{Width: 500, Height: 350})
	w := ts.Windows()[0]
	if w.Title != "Two" || w.Size != (ui.Size{Width: 500, Height: 350}) || c.Title() != "Two" {
		t.Errorf("window = %+v, title = %q", w, c.Title())
	}
}

func TestTheControllerFollowsThePlatform(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{Title: "One"})
	if err != nil {
		t.Fatal(err)
	}
	view := c.RootView().ViewID()
	notified := 0
	c.AddListener(func() { notified++ })

	if c.ContentSize() != (ui.Size{Width: 400, Height: 300}) || c.IsMaximized() || c.IsActivated() || c.IsFullscreen() || c.IsMinimized() {
		t.Fatalf("the start state is wrong: %v", c.ContentSize())
	}
	ts.SetWindowSize(view, ui.Size{Width: 640, Height: 480})
	if c.ContentSize() != (ui.Size{Width: 640, Height: 480}) || notified != 1 {
		t.Errorf("size = %v, notified %d times", c.ContentSize(), notified)
	}
	ts.SetWindowInfo(view, ui.WindowInfo{Maximized: true, Activated: true, Title: "From the platform"})
	if !c.IsMaximized() || !c.IsActivated() || c.Title() != "From the platform" || notified != 2 {
		t.Errorf("maximized %v activated %v title %q notified %d", c.IsMaximized(), c.IsActivated(), c.Title(), notified)
	}
	ts.SetWindowInfo(view, ui.WindowInfo{Fullscreen: true, Activated: true, Title: "From the platform"})
	if !c.IsFullscreen() || c.IsMaximized() {
		t.Errorf("fullscreen %v maximized %v", c.IsFullscreen(), c.IsMaximized())
	}
	// Reporting the same state again tells nobody.
	n := notified
	ts.SetWindowInfo(view, ui.WindowInfo{Fullscreen: true, Activated: true, Title: "From the platform"})
	if notified != n {
		t.Error("an unchanged state notified the listeners")
	}
}

func TestMinimizedEndsWhenTheWindowIsActivated(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	c.SetMinimized(true)
	if !c.IsMinimized() {
		t.Fatal("the window is not minimized")
	}
	ts.SetWindowInfo(c.RootView().ViewID(), ui.WindowInfo{Activated: true})
	if c.IsMinimized() {
		t.Error("an activated window is still minimized")
	}
}

func TestAPlatformThatSaysWhenAWindowIsMinimizedIsBelieved(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	view := c.RootView().ViewID()
	// The user minimizes the window with the task bar, not the app.
	ts.SetWindowInfo(view, ui.WindowInfo{Minimized: true})
	if !c.IsMinimized() {
		t.Error("the platform says the window is minimized")
	}
	// The user brings it back without it being active (the manager restored it).
	ts.SetWindowInfo(view, ui.WindowInfo{})
	if c.IsMinimized() {
		t.Error("the platform says the window is not minimized any more")
	}
	// A platform that never says leaves the answer to what the app asked for.
	c.SetMinimized(true)
	ts.SetWindowInfo(view, ui.WindowInfo{})
	if !c.IsMinimized() {
		t.Error("a platform that does not say changed the answer")
	}
}

func TestWindowScope(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{Title: "One"})
	if err != nil {
		t.Fatal(err)
	}
	var (
		controller         BaseWindowController
		size               ui.Size
		title              string
		activated, maximiz bool
		sizeBuilds         int
		titleBuilds        int
		activatedBuilds    int
	)
	probe := func(build func(bc BuildContext)) Widget {
		return NewBuilder(func(bc BuildContext) Widget {
			build(bc)
			return NewSizedBox(nil)
		})
	}
	content := NewColumn([]Widget{
		probe(func(bc BuildContext) { sizeBuilds++; size = WindowScopeContentSizeOf(bc) }),
		probe(func(bc BuildContext) { titleBuilds++; title = WindowScopeTitleOf(bc) }),
		probe(func(bc BuildContext) {
			activatedBuilds++
			activated = WindowScopeIsActivatedOf(bc)
			maximiz = WindowScopeIsMaximizedOf(bc)
		}),
		// WindowScopeOf depends on the whole scope, so it is not counted.
		probe(func(bc BuildContext) { controller = WindowScopeOf(bc) }),
	})
	withWindows(ts, text("main"), NewRegularWindow(c, content))
	if controller != BaseWindowController(c) || size != (ui.Size{Width: 400, Height: 300}) || title != "One" || activated || maximiz {
		t.Fatalf("controller %v size %v title %q activated %v maximized %v", controller, size, title, activated, maximiz)
	}

	// Each dependent is rebuilt only when what it depends on changes.
	sizeBuilds, titleBuilds, activatedBuilds = 0, 0, 0
	c.SetTitle("Two")
	ts.Pump()
	if title != "Two" || titleBuilds != 1 || sizeBuilds != 0 || activatedBuilds != 0 {
		t.Errorf("after the title: %q, builds %d %d %d", title, sizeBuilds, titleBuilds, activatedBuilds)
	}
	ts.SetWindowSize(c.RootView().ViewID(), ui.Size{Width: 500, Height: 380})
	ts.Pump()
	if size != (ui.Size{Width: 500, Height: 380}) || sizeBuilds != 1 || titleBuilds != 1 || activatedBuilds != 0 {
		t.Errorf("after the size: %v, builds %d %d %d", size, sizeBuilds, titleBuilds, activatedBuilds)
	}
	ts.SetWindowInfo(c.RootView().ViewID(), ui.WindowInfo{Activated: true, Maximized: true, Title: "Two"})
	ts.Pump()
	if !activated || !maximiz || activatedBuilds != 1 || titleBuilds != 1 {
		t.Errorf("after the state: activated %v maximized %v, builds %d %d %d", activated, maximiz, sizeBuilds, titleBuilds, activatedBuilds)
	}
}

func TestWindowScopeOutsideAWindow(t *testing.T) {
	ts, _ := windowApp(t)
	bc := ts.Element(gluttertest.ByText("main"))
	if got := WindowScopeMaybeOf(bc); got != nil {
		t.Errorf("WindowScopeMaybeOf = %v, want nil", got)
	}
	if _, ok := WindowScopeMaybeTitleOf(bc); ok {
		t.Error("WindowScopeMaybeTitleOf found a title")
	}
	if _, ok := WindowScopeMaybeContentSizeOf(bc); ok {
		t.Error("WindowScopeMaybeContentSizeOf found a size")
	}
	defer func() {
		err, _ := recover().(error)
		if err == nil {
			t.Fatal("WindowScopeOf did not panic")
		}
		routeOf(t, err, "WindowScopeMissing")
	}()
	WindowScopeOf(bc)
}

func TestTheFirstWindowHasAScope(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(ts.Binding.WrapWithWindowScope(text("main")))
	controller, _ := ts.Binding.WindowingOwner().ImplicitWindowController()
	if got := WindowScopeOf(ts.Element(gluttertest.ByText("main"))); got != BaseWindowController(controller) {
		t.Errorf("scope = %v", got)
	}
	// The scope follows the window.
	ts.SetWindowInfo(ui.ImplicitViewID, ui.WindowInfo{Title: "Renamed", Activated: true})
	ts.Pump()
	if got := WindowScopeTitleOf(ts.Element(gluttertest.ByText("main"))); got != "Renamed" {
		t.Errorf("title = %q", got)
	}
}

func TestWindowManagerShowsTheRegisteredWindows(t *testing.T) {
	ts, _ := windowApp(t)
	var registry *WindowRegistry
	ts.PumpWidget(NewWindowManager(NewBuilder(func(bc BuildContext) Widget {
		registry = WindowRegistryOf(bc)
		return text("main")
	})))
	if registry == nil || len(registry.Windows()) != 0 {
		t.Fatal("the manager gave no registry")
	}
	bc := ts.Element(gluttertest.ByText("main"))
	if WindowRegistryMaybeOf(bc) != registry {
		t.Error("the registry is not the one of the manager")
	}

	c1, _ := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	c2, _ := NewDialogWindowController(bc, ui.Size{Width: 300, Height: 200}, DialogWindowControllerOpts{Parent: c1})
	e1 := NewWindowEntry(c1, func(BuildContext) Widget { return text("first") })
	e2 := NewWindowEntry(c2, func(BuildContext) Widget { return text("dialog") })
	registry.Register(e1)
	registry.Register(e2)
	ts.Pump()
	for _, want := range []string{"main", "first", "dialog"} {
		if !ts.Exists(gluttertest.ByText(want)) {
			t.Errorf("%q is not shown", want)
		}
	}
	if len(ts.ViewScenes(c1.RootView().ViewID())) == 0 || len(ts.ViewScenes(c2.RootView().ViewID())) == 0 {
		t.Error("a registered window rendered nothing")
	}

	// Removing the first leaves the second as it is.
	dialogElement := ts.Element(gluttertest.ByText("dialog"))
	registry.Unregister(e1)
	ts.Pump()
	if ts.Exists(gluttertest.ByText("first")) {
		t.Error("the unregistered window is still shown")
	}
	if !ts.Exists(gluttertest.ByText("dialog")) || ts.Element(gluttertest.ByText("dialog")) != dialogElement {
		t.Error("the other window was rebuilt")
	}
}

func TestWindowRegistryOutsideAManager(t *testing.T) {
	ts, _ := windowApp(t)
	bc := ts.Element(gluttertest.ByText("main"))
	if WindowRegistryMaybeOf(bc) != nil {
		t.Error("there is a registry")
	}
	defer func() {
		err, _ := recover().(error)
		if err == nil {
			t.Fatal("WindowRegistryOf did not panic")
		}
		routeOf(t, err, "WindowRegistryMissing")
	}()
	WindowRegistryOf(bc)
}

func TestWindowManagerAtTheRootKeepsItsChild(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	var registry *WindowRegistry
	ts.PumpWidget(NewWindowManager(NewBuilder(func(bc BuildContext) Widget {
		registry = WindowRegistryOf(bc)
		return NewView(ts.View, text("main"))
	})), gluttertest.PumpWidgetOpts{NoView: true})
	registry.Register(NewWindowEntry(c, func(BuildContext) Widget { return text("second") }))
	ts.Pump()
	if !ts.Exists(gluttertest.ByText("main")) || !ts.Exists(gluttertest.ByText("second")) {
		t.Error("the manager at the root does not show both")
	}
}

func TestViewAnchorRendersASideView(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 200, Height: 100})
	if err != nil {
		t.Fatal(err)
	}
	ts.PumpWidget(NewViewAnchor(text("anchored"), ViewAnchorOpts{
		View: NewRegularWindow(c, text("side")),
	}))
	if !ts.Exists(gluttertest.ByText("anchored")) || !ts.Exists(gluttertest.ByText("side")) {
		t.Fatal("the anchor does not show both")
	}
	// The side view reaches the inherited widgets above the anchor but not the
	// widgets of the anchored child.
	anchored := ts.Element(gluttertest.ByText("anchored"))
	side := ts.Element(gluttertest.ByText("side"))
	if ViewOf(anchored) == ViewOf(side) {
		t.Error("both are in the same view")
	}
	if len(ts.ViewScenes(c.RootView().ViewID())) == 0 {
		t.Error("the side view rendered nothing")
	}

	// Without the view slot the anchor is its child.
	ts.PumpWidget(NewViewAnchor(text("alone")))
	if !ts.Exists(gluttertest.ByText("alone")) || ts.Exists(gluttertest.ByText("side")) {
		t.Error("the anchor without a view shows the side view")
	}
}

func TestAViewCollectionCannotBeInARenderTree(t *testing.T) {
	skipUnlessDebug(t)
	ts, _ := windowApp(t)
	ts.PumpWidget(NewPadding(painting.NewEdgeInsetsAll(1), NewViewCollection(nil)))
	routeOf(t, ts.TakeError(), "ViewCollectionInRenderTree")
}

func TestViewFocusMovesTheFocusBetweenWindows(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	mainNode := NewFocusNode()
	secondNode := NewFocusNode()
	t.Cleanup(func() { mainNode.Dispose(); secondNode.Dispose() })
	withWindows(ts,
		NewFocus(text("main"), FocusOpts{FocusNode: mainNode}),
		NewRegularWindow(c, NewFocus(text("second"), FocusOpts{FocusNode: secondNode})))

	// Each window remembers the widget that had the focus in it.
	secondNode.RequestFocus()
	mainNode.RequestFocus()
	ts.Pump()
	ts.SetViewFocus(ui.ImplicitViewID, true)
	ts.FlushPosted()
	if !mainNode.HasFocus() {
		t.Error("the main window has the platform focus but its widget does not")
	}
	// The platform moves the focus from one window to the other: the old one is
	// told first.
	ts.SetViewFocus(ui.ImplicitViewID, false)
	ts.FlushPosted()
	if mainNode.HasFocus() || secondNode.HasFocus() {
		t.Errorf("a window that lost the focus parks it: main %v second %v", mainNode.HasFocus(), secondNode.HasFocus())
	}
	ts.SetViewFocus(c.RootView().ViewID(), true)
	ts.FlushPosted()
	if mainNode.HasFocus() || !secondNode.HasFocus() {
		t.Errorf("after focusing the second window: main %v second %v", mainNode.HasFocus(), secondNode.HasFocus())
	}
	ts.SetViewFocus(c.RootView().ViewID(), false)
	ts.FlushPosted()
	ts.SetViewFocus(ui.ImplicitViewID, true)
	ts.FlushPosted()
	if !mainNode.HasFocus() || secondNode.HasFocus() {
		t.Errorf("back in the main window: main %v second %v", mainNode.HasFocus(), secondNode.HasFocus())
	}
}

func TestFocusingAWidgetAsksThePlatformToFocusItsWindow(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	secondNode := NewFocusNode()
	t.Cleanup(secondNode.Dispose)
	withWindows(ts, text("main"), NewRegularWindow(c, NewFocus(text("second"), FocusOpts{FocusNode: secondNode})))
	secondNode.RequestFocus()
	ts.Pump()
	got := ts.FocusRequests()
	if len(got) == 0 || got[len(got)-1] != c.RootView().ViewID() {
		t.Errorf("focus requests = %v, want the second view", got)
	}
}

// TestWindowScopeAccessors ports the accessor tests of windowing_test.dart for the
// two kinds of windows that Glutter has (regular and dialog): each static function
// of WindowScope, with and without Maybe, reads the controller of its window. A
// dialog is never maximized or fullscreen, whatever the platform says.
func TestWindowScopeAccessors(t *testing.T) {
	size := ui.Size{Width: 400, Height: 300}
	maybe := func(v any, ok bool) any {
		if !ok {
			return "not found"
		}
		return v
	}
	accessors := []struct {
		name string
		get  func(bc BuildContext) any
		// want is what the accessor returns for a window that the platform says is
		// activated, maximized and fullscreen; max is the answer for a window that can be.
		want func(c BaseWindowController, canMaximize bool) any
	}{
		{"WindowScopeOf", func(bc BuildContext) any { return WindowScopeOf(bc) }, func(c BaseWindowController, _ bool) any { return c }},
		{"WindowScopeMaybeOf", func(bc BuildContext) any { return WindowScopeMaybeOf(bc) }, func(c BaseWindowController, _ bool) any { return c }},
		{"WindowScopeContentSizeOf", func(bc BuildContext) any { return WindowScopeContentSizeOf(bc) }, func(BaseWindowController, bool) any { return size }},
		{"WindowScopeMaybeContentSizeOf", func(bc BuildContext) any { return maybe(WindowScopeMaybeContentSizeOf(bc)) }, func(BaseWindowController, bool) any { return size }},
		{"WindowScopeTitleOf", func(bc BuildContext) any { return WindowScopeTitleOf(bc) }, func(BaseWindowController, bool) any { return "Stub Window" }},
		{"WindowScopeMaybeTitleOf", func(bc BuildContext) any { return maybe(WindowScopeMaybeTitleOf(bc)) }, func(BaseWindowController, bool) any { return "Stub Window" }},
		{"WindowScopeIsActivatedOf", func(bc BuildContext) any { return WindowScopeIsActivatedOf(bc) }, func(BaseWindowController, bool) any { return true }},
		{"WindowScopeMaybeIsActivatedOf", func(bc BuildContext) any { return maybe(WindowScopeMaybeIsActivatedOf(bc)) }, func(BaseWindowController, bool) any { return true }},
		{"WindowScopeIsMinimizedOf", func(bc BuildContext) any { return WindowScopeIsMinimizedOf(bc) }, func(BaseWindowController, bool) any { return false }},
		{"WindowScopeMaybeIsMinimizedOf", func(bc BuildContext) any { return maybe(WindowScopeMaybeIsMinimizedOf(bc)) }, func(BaseWindowController, bool) any { return false }},
		{"WindowScopeIsMaximizedOf", func(bc BuildContext) any { return WindowScopeIsMaximizedOf(bc) }, func(_ BaseWindowController, can bool) any { return can }},
		{"WindowScopeMaybeIsMaximizedOf", func(bc BuildContext) any { return maybe(WindowScopeMaybeIsMaximizedOf(bc)) }, func(_ BaseWindowController, can bool) any { return can }},
		{"WindowScopeIsFullscreenOf", func(bc BuildContext) any { return WindowScopeIsFullscreenOf(bc) }, func(_ BaseWindowController, can bool) any { return can }},
		{"WindowScopeMaybeIsFullscreenOf", func(bc BuildContext) any { return maybe(WindowScopeMaybeIsFullscreenOf(bc)) }, func(_ BaseWindowController, can bool) any { return can }},
	}
	kinds := []struct {
		name        string
		canMaximize bool
		make        func(bc BuildContext) (BaseWindowController, func(child Widget) Widget, error)
	}{
		{"regular", true, func(bc BuildContext) (BaseWindowController, func(Widget) Widget, error) {
			c, err := NewRegularWindowController(bc, size, RegularWindowControllerOpts{Title: "Stub Window"})
			if err != nil {
				return nil, nil, err
			}
			return c, func(child Widget) Widget { return NewRegularWindow(c, child) }, nil
		}},
		{"dialog", false, func(bc BuildContext) (BaseWindowController, func(Widget) Widget, error) {
			c, err := NewDialogWindowController(bc, size, DialogWindowControllerOpts{Title: "Stub Window"})
			if err != nil {
				return nil, nil, err
			}
			return c, func(child Widget) Widget { return NewDialogWindow(c, child) }, nil
		}},
	}
	for _, kind := range kinds {
		for _, acc := range accessors {
			t.Run(kind.name+"/"+acc.name, func(t *testing.T) {
				ts, bc := windowApp(t)
				c, wrap, err := kind.make(bc)
				if err != nil {
					t.Fatal(err)
				}
				var got any
				withWindows(ts, text("main"), wrap(NewBuilder(func(bc BuildContext) Widget {
					got = acc.get(bc)
					return NewSizedBox(nil)
				})))
				ts.SetWindowInfo(c.RootView().ViewID(), ui.WindowInfo{Activated: true, Maximized: true, Fullscreen: true, Title: "Stub Window"})
				ts.Pump()
				if want := acc.want(c, kind.canMaximize); got != want {
					t.Errorf("%s = %v, want %v", acc.name, got, want)
				}
			})
		}
	}
}

// TestWindowWidgetsDoNotThrow is the 'RegularWindow does not throw' and 'Dialog does
// not throw' pair of windowing_test.dart: a window widget is a root of the tree.
func TestWindowWidgetsDoNotThrow(t *testing.T) {
	ts, bc := windowApp(t)
	regular, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	dialog, err := NewDialogWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	ts.PumpWidget(NewViewCollection([]Widget{NewRegularWindow(regular, NewSizedBox(nil)), NewDialogWindow(dialog, NewSizedBox(nil))}),
		gluttertest.PumpWidgetOpts{NoView: true})
	if errs := ts.Errors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWindowingUnsupported = "WindowingUnsupported"
)
