package semantics_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

func TestSemanticsBindingFollowsThePlatformAndHandles(t *testing.T) {
	s := renderhost.New()
	b := s.Binding.SemanticsBinding()
	changes := 0
	b.AddSemanticsEnabledListener(func() { changes++ })
	if b.SemanticsEnabled() || s.Host.SemanticsTreeEnabled {
		t.Fatal("semantics are on at the start")
	}

	s.Dispatcher.SetSemanticsEnabled(true)
	if !b.SemanticsEnabled() || changes != 1 || !s.Host.SemanticsTreeEnabled {
		t.Errorf("after the platform asked: enabled %v, changes %d, tree %v", b.SemanticsEnabled(), changes, s.Host.SemanticsTreeEnabled)
	}
	s.Dispatcher.SetSemanticsEnabled(false)
	if b.SemanticsEnabled() || s.Host.SemanticsTreeEnabled {
		t.Error("semantics stay on after the platform stopped asking")
	}

	h1, h2 := b.EnsureSemantics(), b.EnsureSemantics()
	if !b.SemanticsEnabled() || b.DebugOutstandingSemanticsHandles() != 2 {
		t.Errorf("with handles: enabled %v, outstanding %d", b.SemanticsEnabled(), b.DebugOutstandingSemanticsHandles())
	}
	h1.Dispose()
	if !b.SemanticsEnabled() {
		t.Error("semantics stop while a handle is open")
	}
	h2.Dispose()
	if b.SemanticsEnabled() {
		t.Error("semantics stay on without handles")
	}
}

func TestSemanticsActionEventsReachListenersAndTheNode(t *testing.T) {
	skipUnlessDebug(t)
	s := renderhost.New()
	b := s.Binding.SemanticsBinding()
	handle := b.EnsureSemantics()
	defer handle.Dispose()

	taps := 0
	props := semantics.NewSemanticsProperties(semantics.SemanticsProperties{OnTap: func() { taps++ }})
	box := rendering.NewRenderSemanticsAnnotations(props,
		rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: 10, Height: 10}), nil), rendering.SemanticsAnnotationsOpts{Container: true})
	s.SetRoot(box)
	s.Pump()

	var events []ui.SemanticsActionEvent
	remove := b.AddSemanticsActionListener(func(e ui.SemanticsActionEvent) { events = append(events, e) })
	node := box.DebugSemantics()
	if node == nil {
		t.Fatal("no node")
	}
	event := ui.SemanticsActionEvent{ViewID: s.View.ViewID(), NodeID: node.ID(), Type: ui.SemanticsActionTap}
	s.Dispatcher.DispatchSemanticsActionEvent(event)
	if taps != 1 || len(events) != 1 {
		t.Errorf("taps %d, events %v", taps, events)
	}
	remove()
	s.Dispatcher.DispatchSemanticsActionEvent(event)
	if taps != 2 || len(events) != 1 {
		t.Errorf("after removing the listener: taps %d, events %d", taps, len(events))
	}
	// An action for a view or node that does not exist is ignored.
	s.Dispatcher.DispatchSemanticsActionEvent(ui.SemanticsActionEvent{ViewID: 99, NodeID: 0, Type: ui.SemanticsActionTap})
	s.Dispatcher.DispatchSemanticsActionEvent(ui.SemanticsActionEvent{ViewID: s.View.ViewID(), NodeID: 99, Type: ui.SemanticsActionTap})
	if taps != 2 {
		t.Errorf("taps = %d, want 2", taps)
	}
}

func TestSemanticsUpdateReachesTheView(t *testing.T) {
	skipUnlessDebug(t)
	s := renderhost.New()
	box := rendering.NewRenderSemanticsAnnotations(semantics.NewSemanticsProperties(semantics.SemanticsProperties{Label: foundation.Some("hello")}),
		rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: 10, Height: 10}), nil),
		rendering.SemanticsAnnotationsOpts{Container: true, TextDirection: ui.TextDirectionLTR})
	s.SetRoot(box)
	s.Pump()
	if len(s.Host.SemanticsUpdates) != 0 {
		t.Fatalf("an update was sent without semantics: %d", len(s.Host.SemanticsUpdates))
	}
	handle := s.Binding.SemanticsBinding().EnsureSemantics()
	s.Pump()
	if len(s.Host.SemanticsUpdates) != 1 {
		t.Fatalf("updates = %d, want 1", len(s.Host.SemanticsUpdates))
	}
	found := false
	for _, n := range s.Host.SemanticsUpdates[0].Nodes {
		if n.Label == "hello" {
			found = true
		}
	}
	if !found {
		t.Errorf("no node with the label in %+v", s.Host.SemanticsUpdates[0].Nodes)
	}
	// Disabling semantics clears the nodes.
	handle.Dispose()
	if box.DebugSemantics() != nil {
		t.Error("the node stays after semantics were turned off")
	}
}

func TestDisableAnimations(t *testing.T) {
	s := renderhost.New()
	b := s.Binding.SemanticsBinding()
	if b.DisableAnimations() {
		t.Error("animations are disabled by default")
	}
	s.Dispatcher.UpdatePlatformConfiguration(ui.PlatformConfiguration{AccessibilityFeatures: ui.AccessibilityFeatures{DisableAnimations: true}})
	s.Dispatcher.OnAccessibilityFeaturesChanged()
	if !b.DisableAnimations() {
		t.Error("animations stay enabled after the platform asked to disable them")
	}
	if foundation.DebugMode {
		b.DebugSemanticsDisableAnimations = foundation.Some(false)
		if b.DisableAnimations() {
			t.Error("the debug override is ignored")
		}
	}
}
