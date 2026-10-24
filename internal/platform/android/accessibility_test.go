package android

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// a11yHost is the engine's side of the bridge: it records what the bridge tells it.
type a11yHost struct {
	changes []bool
	actions []ui.SemanticsActionEvent
}

func (h *a11yHost) ScreenReaderChanged(enabled bool)        { h.changes = append(h.changes, enabled) }
func (h *a11yHost) PerformAction(e ui.SemanticsActionEvent) { h.actions = append(h.actions, e) }

var _ platform.AccessibilityHost = (*a11yHost)(nil)

// a11yFixture is a platform over fakes with its bridge, and the commands it sends.
type a11yFixture struct {
	t    *testing.T
	b    *bridge
	host *fakeHost
	p    *Platform
	a    platform.Accessibility
	eng  *a11yHost
}

func newA11yFixture(t *testing.T) *a11yFixture {
	t.Helper()
	b := newBridge()
	b.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	host := newFakeHost()
	p := newPlatform(b, host, b.log)
	svc := p.Services()
	if svc == nil || svc.Accessibility() == nil {
		t.Fatal("the platform has no accessibility bridge")
	}
	return &a11yFixture{t: t, b: b, host: host, p: p, a: svc.Accessibility(), eng: &a11yHost{}}
}

// state is what Native.a11yState does, with no UI thread running: the state is applied at once.
func (f *a11yFixture) state(enabled, touch, nav bool) {
	f.b.a11yChanged(a11yState{enabled: enabled, touch: touch, navigation: nav})
	f.p.a11y.stateChanged()
}

func (f *a11yFixture) ops() []int {
	var ops []int
	for _, c := range f.host.sentCommands() {
		ops = append(ops, c.Op)
	}
	return ops
}

func sampleUpdate() *ui.SemanticsUpdate {
	b := ui.NewSemanticsUpdateBuilder()
	b.UpdateNode(update(func(n *ui.SemanticsNodeUpdate) { n.ID = 0; n.ChildrenInTraversalOrder = []int{1} }))
	b.UpdateNode(update(func(n *ui.SemanticsNodeUpdate) { n.ID = 1; n.Label = "OK"; n.Actions = ui.SemanticsActionTap }))
	return b.Build()
}

func TestAccessibilityStartStoresTheHostAndTellsTheStateAtOnce(t *testing.T) {
	tests := []struct {
		name  string
		state a11yState
		want  []bool
	}{
		{"off", a11yState{}, nil},
		{"a service runs", a11yState{enabled: true}, []bool{true}},
		{"touch exploration", a11yState{touch: true}, []bool{true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newA11yFixture(t)
			f.b.a11yChanged(tt.state) // before Start: applied, nobody to tell
			f.a.Start(context.Background(), f.eng)
			if !reflect.DeepEqual(f.eng.changes, tt.want) {
				t.Errorf("ScreenReaderChanged calls = %v, want %v", f.eng.changes, tt.want)
			}
			f.a.Start(context.Background(), f.eng) // a second Start does nothing
			if !reflect.DeepEqual(f.eng.changes, tt.want) {
				t.Errorf("after a second Start: %v", f.eng.changes)
			}
		})
	}
}

func TestAccessibilityFollowsTheScreenReaderState(t *testing.T) {
	f := newA11yFixture(t)
	f.a.Start(context.Background(), f.eng)
	f.state(true, false, false)
	f.state(true, true, false) // touch exploration joins: still on, no second call
	f.state(false, true, true) // the service goes, touch exploration stays: on
	f.state(false, false, false)
	f.state(false, false, false)
	f.state(true, false, true)
	if want := []bool{true, false, true}; !reflect.DeepEqual(f.eng.changes, want) {
		t.Errorf("ScreenReaderChanged calls = %v, want %v", f.eng.changes, want)
	}
	// Going off drops the tree of the Java side.
	if got, want := f.ops(), []int{opA11yReset}; !reflect.DeepEqual(got, want) {
		t.Errorf("commands = %v, want %v", got, want)
	}
}

func TestAccessibleNavigationReachesTheSettings(t *testing.T) {
	f := newA11yFixture(t)
	rec := &recorder{}
	f.p.handler = rec
	if f.p.Settings().AccessibleNavigation {
		t.Fatal("accessible navigation is on before a service reports")
	}
	f.a.Start(context.Background(), f.eng)
	f.state(true, true, true)
	if !f.p.Settings().AccessibleNavigation || rec.count("settings") != 1 || !rec.lastSettings().AccessibleNavigation {
		t.Errorf("settings after the report: %+v, %d OnSettings calls", f.p.Settings(), rec.count("settings"))
	}
	f.state(true, true, true)
	if rec.count("settings") != 1 {
		t.Errorf("the same state again told the settings: %d calls", rec.count("settings"))
	}
	f.state(false, false, false)
	if f.p.Settings().AccessibleNavigation || rec.count("settings") != 2 {
		t.Errorf("settings after the service went: %+v, %d calls", f.p.Settings(), rec.count("settings"))
	}
}

func TestAccessibilityUpdatesAreSentOnlyWhileOn(t *testing.T) {
	f := newA11yFixture(t)
	f.a.Start(context.Background(), f.eng)
	f.a.UpdateSemantics(ui.ImplicitViewID, sampleUpdate())
	if len(f.ops()) != 0 {
		t.Fatalf("an update was sent while the bridge is off: %v", f.ops())
	}
	f.state(true, false, false)
	f.a.UpdateSemantics(ui.ImplicitViewID, sampleUpdate())
	f.a.UpdateSemantics(ui.ImplicitViewID, nil)
	f.a.UpdateSemantics(ui.ImplicitViewID, &ui.SemanticsUpdate{})
	f.a.UpdateSemantics(7, sampleUpdate()) // there is one view
	cmds := f.host.sentCommands()
	if len(cmds) != 1 || cmds[0].Op != opSemantics {
		t.Fatalf("commands = %+v", cmds)
	}
	nodes, err := decodeSemantics([]byte(cmds[0].S))
	if err != nil || len(nodes) != 2 || nodes[1].Label != "OK" || nodes[1].Props&propClickable == 0 {
		t.Errorf("the payload decodes to %+v, %v", nodes, err)
	}
}

func TestAccessibilityKeepsTheCustomActionsAcrossUpdates(t *testing.T) {
	f := newA11yFixture(t)
	f.a.Start(context.Background(), f.eng)
	f.state(true, false, false)
	first := ui.NewSemanticsUpdateBuilder()
	first.UpdateCustomAction(ui.SemanticsCustomActionUpdate{ID: 9, Label: "Archive", OverrideID: -1})
	f.a.UpdateSemantics(ui.ImplicitViewID, first.Build()) // definitions only: nothing to send
	second := ui.NewSemanticsUpdateBuilder()
	second.UpdateNode(update(func(n *ui.SemanticsNodeUpdate) { n.ID = 2; n.AdditionalActions = []int{9} }))
	f.a.UpdateSemantics(ui.ImplicitViewID, second.Build())
	cmds := f.host.sentCommands()
	if len(cmds) != 1 {
		t.Fatalf("commands = %+v", cmds)
	}
	nodes, err := decodeSemantics([]byte(cmds[0].S))
	if err != nil || len(nodes) != 1 || !reflect.DeepEqual(nodes[0].AndroidActions, []androidAction{{ID: 9 + firstResourceID, Label: "Archive"}}) {
		t.Errorf("nodes %+v, %v", nodes, err)
	}
}

func TestAccessibilityRemoveViewAndSetWindow(t *testing.T) {
	f := newA11yFixture(t)
	f.a.Start(context.Background(), f.eng)
	f.a.SetWindow(ui.ImplicitViewID, "Gallery", true)
	if f.p.a11y.title != "Gallery" || !f.p.a11y.active {
		t.Errorf("SetWindow kept %q, %v", f.p.a11y.title, f.p.a11y.active)
	}
	f.a.RemoveView(3)
	if len(f.ops()) != 0 {
		t.Errorf("removing another view sent %v", f.ops())
	}
	f.a.RemoveView(ui.ImplicitViewID)
	if got, want := f.ops(), []int{opA11yReset}; !reflect.DeepEqual(got, want) {
		t.Errorf("commands = %v, want %v", got, want)
	}
}

func TestAccessibilityPerformsActionsOfTheJavaSideWhileOn(t *testing.T) {
	f := newA11yFixture(t)
	f.a.Start(context.Background(), f.eng)
	f.p.a11y.perform(1, int(ui.SemanticsActionTap), nil) // off: dropped
	f.state(true, false, false)
	f.p.a11y.perform(1, int(ui.SemanticsActionTap), nil)
	f.p.a11y.perform(2, int(ui.SemanticsActionSetText), []byte("hi"))
	f.p.a11y.perform(3, 0, nil) // not understood: logged and dropped
	want := []ui.SemanticsActionEvent{
		{ViewID: ui.ImplicitViewID, NodeID: 1, Type: ui.SemanticsActionTap},
		{ViewID: ui.ImplicitViewID, NodeID: 2, Type: ui.SemanticsActionSetText, Arguments: "hi"},
	}
	if !reflect.DeepEqual(f.eng.actions, want) {
		t.Errorf("actions = %+v, want %+v", f.eng.actions, want)
	}
}

func TestAccessibilityCloseStopsTheBridge(t *testing.T) {
	f := newA11yFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	f.a.Start(ctx, f.eng)
	f.state(true, false, false)
	cancel()
	waitFor(t, "the bridge to close", f.p.a11y.isClosed)
	f.a.Close() // twice is fine
	f.a.UpdateSemantics(ui.ImplicitViewID, sampleUpdate())
	f.state(false, false, false)
	f.p.a11y.perform(1, int(ui.SemanticsActionTap), nil)
	if len(f.ops()) != 0 || len(f.eng.actions) != 0 || !reflect.DeepEqual(f.eng.changes, []bool{true}) {
		t.Errorf("a closed bridge did something: commands %v, actions %v, changes %v", f.ops(), f.eng.actions, f.eng.changes)
	}
}

func TestAccessibilityForcedOnInDebugBuilds(t *testing.T) {
	f := newA11yFixture(t)
	f.p.a11y.force() // before Start: the Java side is told, the host is not yet
	if got, want := f.host.sentCommands(), []command{{Op: opA11yForce, A: 1}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %+v, want %+v", got, want)
	}
	f.a.Start(context.Background(), f.eng)
	if !reflect.DeepEqual(f.eng.changes, []bool{true}) {
		t.Errorf("a forced bridge is on at Start: %v", f.eng.changes)
	}
	f.state(false, false, false) // the real state stays off: the bridge stays on
	if !reflect.DeepEqual(f.eng.changes, []bool{true}) {
		t.Errorf("changes = %v", f.eng.changes)
	}
}

func TestAccessibilityResendForANewActivity(t *testing.T) {
	f := newA11yFixture(t)
	f.a.Start(context.Background(), f.eng)
	f.state(true, false, false)
	f.p.activityCreated()
	// The framework makes the tree again: off and on.
	if want := []bool{true, false, true}; !reflect.DeepEqual(f.eng.changes, want) {
		t.Errorf("changes = %v, want %v", f.eng.changes, want)
	}
	f.p.a11y.forced = true
	f.p.activityCreated()
	found := false
	for _, c := range f.host.sentCommands() {
		found = found || (c.Op == opA11yForce && c.A == 1)
	}
	if !found {
		t.Error("a forced bridge is forced again on the new activity")
	}
}

// The natives of the Java side go through the queue: the state reaches the host on the UI thread.
func TestAccessibilityStateAndActionsGoThroughTheQueue(t *testing.T) {
	h := newHarness(t)
	eng := &a11yHost{}
	h.p.Services().Accessibility().Start(context.Background(), eng)
	h.run()
	h.b.a11yChanged(a11yState{enabled: true, navigation: true})
	h.b.a11yAction(4, int(ui.SemanticsActionTap), nil)
	h.sync(func() {})
	h.sync(func() {
		if !reflect.DeepEqual(eng.changes, []bool{true}) || len(eng.actions) != 1 || eng.actions[0].NodeID != 4 {
			t.Errorf("changes %v, actions %+v", eng.changes, eng.actions)
		}
	})
	waitFor(t, "OnSettings", func() bool { return h.h.count("settings") > 0 })
	if !h.h.lastSettings().AccessibleNavigation {
		t.Error("the settings do not carry accessible navigation")
	}
}
