package engine

import (
	"context"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/ui"
)

func TestAccessibilityFollowsTheWindowsAndTheTree(t *testing.T) {
	e := newWindowsEngine(t)
	acc := e.p.Accessibility()
	e.h.first.title = "Main"
	e.h.startAccessibility(context.Background())
	if !acc.Started() {
		t.Fatal("the bridge was not started")
	}
	first := e.h.first.id
	if got := acc.Windows(); !reflect.DeepEqual(got, []fake.WindowRecord{{View: first, Title: "Main"}}) {
		t.Errorf("windows = %+v", got)
	}

	// The screen reader asks for the tree, and the framework hears it.
	var told int
	e.d.OnSemanticsEnabledChanged = func() { told++ }
	e.h.ScreenReaderChanged(true)
	if !e.d.SemanticsEnabled() || told != 1 {
		t.Errorf("semantics enabled %v, told %d", e.d.SemanticsEnabled(), told)
	}
	e.h.ScreenReaderChanged(false)
	if e.d.SemanticsEnabled() || told != 2 {
		t.Errorf("semantics enabled %v, told %d", e.d.SemanticsEnabled(), told)
	}

	// The tree that the view sends reaches the bridge.
	b := ui.NewSemanticsUpdateBuilder()
	b.UpdateNode(ui.SemanticsNodeUpdate{ID: 0, Label: "root"})
	update := b.Build()
	e.d.View(first).UpdateSemantics(update)
	if got := acc.Updates(); len(got) != 1 || got[0].View != first || got[0].Update != update {
		t.Errorf("updates = %+v", got)
	}
	e.d.SetSemanticsTreeEnabled(true) // nothing to do for the engine

	// What the screen reader asks for reaches the framework.
	var got []ui.SemanticsActionEvent
	e.d.OnSemanticsActionEvent = func(ev ui.SemanticsActionEvent) { got = append(got, ev) }
	ev := ui.SemanticsActionEvent{ViewID: first, NodeID: 3, Type: ui.SemanticsActionTap}
	e.h.PerformAction(ev)
	if !reflect.DeepEqual(got, []ui.SemanticsActionEvent{ev}) {
		t.Errorf("actions = %+v", got)
	}

	// Titles, the focus, new windows and closed windows.
	e.h.SetWindowTitle(first, "Renamed")
	e.h.OnViewFocus(first, true)
	id, _, err := e.h.CreateWindow(ui.WindowSpec{Title: "Second", Size: ui.Size{Width: 100, Height: 100}})
	if err != nil {
		t.Fatal(err)
	}
	e.h.DestroyWindow(id)
	want := []fake.WindowRecord{
		{View: first, Title: "Main"}, {View: first, Title: "Renamed"}, {View: first, Title: "Renamed", Active: true},
		{View: id, Title: "Second"},
	}
	if got := acc.Windows(); !reflect.DeepEqual(got, want) {
		t.Errorf("windows = %+v\nwant      %+v", got, want)
	}
	if got := acc.Removed(); !reflect.DeepEqual(got, []ui.ViewID{id}) {
		t.Errorf("removed = %v", got)
	}

	e.h.release()
	if !acc.Closed() {
		t.Error("the bridge was not closed")
	}
	e.h.release() // twice is fine
	// Without a bridge the engine ignores the tree.
	e.h.UpdateSemantics(first, update)
}

func TestTheScreenReaderOfThePlatformReachesTheHandler(t *testing.T) {
	e := newWindowsEngine(t)
	acc := e.p.Accessibility()
	e.h.startAccessibility(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	enabled := make(chan bool, 1)
	e.d.OnSemanticsEnabledChanged = func() { enabled <- e.d.SemanticsEnabled() }
	done := make(chan error, 1)
	go func() { done <- e.p.Run(ctx, e.h) }()
	acc.ScreenReader(true)
	if got := <-enabled; !got {
		t.Error("the framework was not asked for semantics")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
