package widgets

import (
	"testing"

	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// The tests of the checks that need unexported identifiers of the package, so
// that they cannot run in the external test package that uses gluttertest.

func TestMaybeWidgetsBindingOfWithoutBinding(t *testing.T) {
	tr := newTree(t)
	var missing *WidgetsBinding
	tr.pump(builderOf(func(bc BuildContext) Widget { missing = MaybeWidgetsBindingOf(bc); return newLeaf() }))
	if missing != nil {
		t.Error("MaybeWidgetsBindingOf found a binding in a tree that has none")
	}
}

func TestViewInternalChecks(t *testing.T) {
	skipUnlessDebug(t)
	bt := newBindingTree(t)
	bt.pump(newBox(10, 10))
	raw := bt.find(func(w Widget) bool { _, ok := w.(*rawViewInternal); return ok })
	if len(raw) != 1 {
		t.Fatalf("%d raw views, want 1", len(raw))
	}
	requireTag(t, "RawViewChildMoved", func() { raw[0].(*rawViewElement).MoveRenderObjectChild(nil, nil, nil) })

	// A binding whose dispatcher has no implicit view cannot wrap the app.
	noView := newBindingTree(t)
	noView.d.RemoveView(ui.ImplicitViewID)
	requireTag(t, "ImplicitViewMissing", func() { noView.b.WrapWithDefaultView(newLeaf()) })
}

// TestMediaQueryFromViewStopsListeningWhenRemoved checks that the state removes
// its observer in Dispose.
func TestMediaQueryFromViewStopsListeningWhenRemoved(t *testing.T) {
	bt := newBindingTree(t)
	// The focus manager of the binding is an observer too.
	base := len(bt.b.observers)
	bt.attachUnwrapped(NewMediaQueryFromView(bt.view, newLeaf()))
	bt.frame()
	if n := len(bt.b.observers) - base; n != 1 {
		t.Fatalf("%d observers, want 1", n)
	}
	bt.attachUnwrapped(newLeaf())
	bt.frame()
	if n := len(bt.b.observers) - base; n != 0 {
		t.Errorf("%d observers after the MediaQuery left the tree, want 0", n)
	}
}

func TestWidgetsBindingOfWithoutBinding(t *testing.T) {
	tr := newTree(t)
	tr.collectPanics = true
	tr.pump(builderOf(func(bc BuildContext) Widget { WidgetsBindingOf(bc); return newLeaf() }))
	_ = requireErrorTag(t, tr.errs[0], "WidgetsBindingMissing") // the ErrorWidget has no render tree to attach to, which is a second error
}

func TestFocusTraversalInternalChecks(t *testing.T) {
	skipUnlessDebug(t)
	bt := newBindingTree(t)
	node := NewFocusNode()
	defer node.Dispose()
	bt.pump(NewFocus(newBox(10, 10), FocusOpts{FocusNode: node}))
	nodes := []*FocusNode{node}
	requireTag(t, "TraversalDirectionInvalid", func() { sortAndFilterHorizontally(TraversalDirectionUp, ui.Rect{}, nodes, true) })
	requireTag(t, "TraversalDirectionInvalid", func() { sortAndFilterVertically(TraversalDirectionLeft, ui.Rect{}, nodes, true) })
}

func TestShortcutRegistryForeignEntry(t *testing.T) {
	skipUnlessDebug(t)
	bt := newBindingTree(t)
	registry := NewShortcutRegistry(bt.b.Scheduler())
	other := NewShortcutRegistry(bt.b.Scheduler())
	defer registry.Dispose()
	defer other.Dispose()
	token := other.AddAll(ShortcutMap{{NewSingleActivator(services.LogicalKeyboardKeyKeyA), DoNothingIntent{}}})
	defer token.Dispose()
	requireTag(t, "ShortcutRegistryEntryForeign", func() { registry.replaceAll(token, ShortcutMap{}) })
}
