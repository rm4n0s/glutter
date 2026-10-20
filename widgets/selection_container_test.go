package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/selection_container_test.dart.

// testContainerDelegate is TestContainerDelegate: it can not handle events.
type testContainerDelegate struct {
	MultiSelectableSelectionContainerDelegate
}

func newTestContainerDelegate() *testContainerDelegate {
	d := &testContainerDelegate{}
	InitMultiSelectableSelectionContainerDelegate(&d.MultiSelectableSelectionContainerDelegate, d)
	return d
}

func (d *testContainerDelegate) DispatchSelectionEventToChild(rendering.Selectable, rendering.SelectionEvent) rendering.SelectionResult {
	panic("unimplemented")
}

func (d *testContainerDelegate) EnsureChildUpdated(rendering.Selectable) { panic("unimplemented") }

// testSelectionRegistrar is TestSelectionRegistrar.
type testSelectionRegistrar struct {
	selectables map[rendering.Selectable]struct{}
}

func newTestSelectionRegistrar() *testSelectionRegistrar {
	return &testSelectionRegistrar{selectables: map[rendering.Selectable]struct{}{}}
}

func (r *testSelectionRegistrar) Add(s rendering.Selectable)    { r.selectables[s] = struct{}{} }
func (r *testSelectionRegistrar) Remove(s rendering.Selectable) { delete(r.selectables, s) }

func ltrText(text string) Widget {
	return NewText(text, TextOpts{TextDirection: ui.TextDirectionLTR})
}

func columnOf(children ...Widget) Widget { return NewColumn(children) }

func pumpContainer(ts *gluttertest.Tester, child Widget) {
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: child}))
}

func TestSelectionContainerUpdatesItsRegistrarAndDelegateBasedOnTheNumberOfSelectables(t *testing.T) {
	ts := gluttertest.New(t)
	registrar, delegate := newTestSelectionRegistrar(), newTestContainerDelegate()
	t.Cleanup(delegate.Dispose)
	pumpContainer(ts, NewSelectionContainer(delegate, columnOf(ltrText("column1"), ltrText("column2"), ltrText("column3")),
		SelectionContainerOpts{Registrar: registrar}))
	ts.PumpAndSettle()
	if len(registrar.selectables) != 1 {
		t.Errorf("registrar selectables = %d, want 1", len(registrar.selectables))
	}
	if len(delegate.Selectables) != 3 {
		t.Errorf("delegate selectables = %d, want 3", len(delegate.Selectables))
	}
}

func TestSelectionContainerDisabledContainer(t *testing.T) {
	ts := gluttertest.New(t)
	registrar, delegate := newTestSelectionRegistrar(), newTestContainerDelegate()
	t.Cleanup(delegate.Dispose)
	pumpContainer(ts, NewSelectionContainer(delegate,
		NewSelectionContainerDisabled(columnOf(ltrText("column1"), ltrText("column2"), ltrText("column3"))),
		SelectionContainerOpts{Registrar: registrar}))
	if len(registrar.selectables) != 0 || len(delegate.Selectables) != 0 {
		t.Errorf("selectables = %d, %d, want 0, 0", len(registrar.selectables), len(delegate.Selectables))
	}
}

func TestSelectionContainerSwappingOutContainerDelegateDoesNotCrash(t *testing.T) {
	ts := gluttertest.New(t)
	registrar, delegate := newTestSelectionRegistrar(), newTestContainerDelegate()
	t.Cleanup(delegate.Dispose)
	childDelegate := newTestContainerDelegate()
	t.Cleanup(childDelegate.Dispose)
	build := func(d SelectionContainerDelegate) Widget {
		return NewSelectionContainer(delegate, NewBuilder(func(bc BuildContext) Widget {
			return NewSelectionContainer(d, ltrTextPlain("dummy"), SelectionContainerOpts{Registrar: SelectionContainerMaybeOf(bc)})
		}), SelectionContainerOpts{Registrar: registrar})
	}
	pumpContainer(ts, build(childDelegate))
	ts.PumpAndSettle()
	if len(registrar.selectables) != 1 || !delegate.Value().HasContent {
		t.Fatalf("selectables = %d, hasContent = %v", len(registrar.selectables), delegate.Value().HasContent)
	}
	newDelegate := newTestContainerDelegate()
	t.Cleanup(newDelegate.Dispose)
	pumpContainer(ts, build(newDelegate))
	ts.PumpAndSettle()
	if len(registrar.selectables) != 1 || !delegate.Value().HasContent {
		t.Errorf("after swapping: selectables = %d, hasContent = %v", len(registrar.selectables), delegate.Value().HasContent)
	}
}

func ltrTextPlain(text string) Widget { return NewText(text) }

func TestSelectionContainerCanUpdateWithinOneFrame(t *testing.T) {
	ts := gluttertest.New(t)
	registrar, delegate := newTestSelectionRegistrar(), newTestContainerDelegate()
	t.Cleanup(delegate.Dispose)
	childDelegate := newTestContainerDelegate()
	t.Cleanup(childDelegate.Dispose)
	// The Text widget has an internal SelectionContainer.
	pumpContainer(ts, NewSelectionContainer(delegate, NewBuilder(func(bc BuildContext) Widget {
		return NewSelectionContainer(childDelegate, ltrTextPlain("dummy"), SelectionContainerOpts{Registrar: SelectionContainerMaybeOf(bc)})
	}), SelectionContainerOpts{Registrar: registrar}))
	ts.Pump()
	// It should finish the update after flushing the microtasks.
	ts.FlushPosted()
	if len(registrar.selectables) != 1 || !delegate.Value().HasContent {
		t.Errorf("selectables = %d, hasContent = %v", len(registrar.selectables), delegate.Value().HasContent)
	}
}

func TestSelectionContainerRegistersItselfIfThereIsASelectableChild(t *testing.T) {
	ts := gluttertest.New(t)
	registrar, delegate := newTestSelectionRegistrar(), newTestContainerDelegate()
	t.Cleanup(delegate.Dispose)
	pumpContainer(ts, NewSelectionContainer(delegate, columnOf(), SelectionContainerOpts{Registrar: registrar}))
	if len(registrar.selectables) != 0 {
		t.Fatalf("selectables = %d before the child", len(registrar.selectables))
	}
	pumpContainer(ts, NewSelectionContainer(delegate, columnOf(ltrText("column1")), SelectionContainerOpts{Registrar: registrar}))
	ts.PumpAndSettle()
	if len(registrar.selectables) != 1 {
		t.Fatalf("selectables = %d with a child, want 1", len(registrar.selectables))
	}
	pumpContainer(ts, NewSelectionContainer(delegate, columnOf(), SelectionContainerOpts{Registrar: registrar}))
	ts.PumpAndSettle()
	if len(registrar.selectables) != 0 {
		t.Errorf("selectables = %d without the child, want 0", len(registrar.selectables))
	}
}

func TestSelectionContainerGetsRegistrarFromContextIfNotProvided(t *testing.T) {
	ts := gluttertest.New(t)
	registrar, delegate := newTestSelectionRegistrar(), newTestContainerDelegate()
	t.Cleanup(delegate.Dispose)
	pumpContainer(ts, NewSelectionRegistrarScope(registrar, NewSelectionContainer(delegate, columnOf(ltrText("column1")))))
	ts.PumpAndSettle()
	if len(registrar.selectables) != 1 {
		t.Errorf("selectables = %d, want 1", len(registrar.selectables))
	}
}
