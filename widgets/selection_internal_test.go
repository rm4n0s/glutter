package widgets

import (
	"testing"

	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// These tests check the failures of the selection, menus and autocomplete code
// that need unexported identifiers.

// fakeSelectable is a Selectable that is only compared; calling it panics.
type fakeSelectable struct{ rendering.Selectable }

func TestMenuAnchorChildrenGuards(t *testing.T) {
	skipUnlessDebug(t)
	base := &rawMenuAnchorBase{}
	child := &rawMenuAnchorGroupState{}
	base.addChild(child)
	requireTag(t, "MenuAnchorChildAlreadyAdded", func() { base.addChild(child) })
	base.removeChild(child)
	requireTag(t, "MenuAnchorChildNotFound", func() { base.removeChild(child) })
}

func TestRawAutocompleteHighlightNeedsAShowableOptionsView(t *testing.T) {
	skipUnlessDebug(t)
	focus := NewFocusNode()
	defer focus.Dispose()
	s := &rawAutocompleteStateOf[string]{}
	s.widget = &RawAutocomplete[string]{o: RawAutocompleteOpts[string]{FocusNode: focus}}
	requireTag(t, "RawAutocompleteOptionsNotShowable", func() { s.highlightOption(0) })
}

func TestMultiSelectableDelegateRegistrarGuards(t *testing.T) {
	skipUnlessDebug(t)
	m := &MultiSelectableSelectionContainerDelegate{}
	s := &fakeSelectable{}
	m.Selectables = append(m.Selectables, s)
	requireTag(t, "SelectableAlreadyAdded", func() { m.Add(s) })
	m.Selectables = nil
	requireTag(t, "SelectableNotInRegistrar", func() { m.removeSelectable(s) })
}

func TestSelectableRegionSelectionStatusFinalizesOnlyAfterChanging(t *testing.T) {
	skipUnlessDebug(t)
	n := &selectionStatusNotifier{}
	n.set(SelectableRegionSelectionStatusChanging)
	n.set(SelectableRegionSelectionStatusFinalized)
	requireTag(t, "SelectableRegionAlreadyFinalized", func() { n.set(SelectableRegionSelectionStatusFinalized) })
}

func TestSelectionContainerDelegateBeforeItIsBuilt(t *testing.T) {
	skipUnlessDebug(t)
	b := &SelectionContainerDelegateBase{}
	requireTag(t, "SelectionContainerNotLaidOut", func() { b.HasSize() })
	requireTag(t, "SelectionContainerNotLaidOut", func() { b.ContainerSize() })
	requireTag(t, "SelectionContainerNotLaidOut", func() { b.GetTransformTo(nil) })
}

func TestSelectionContainerDisabledGuards(t *testing.T) {
	skipUnlessDebug(t)
	bt := newBindingTree(t)
	bt.pump(internalTestApp(NewSelectionContainerDisabled(NewSizedBox(nil))))
	found := bt.find(func(w Widget) bool { sc, ok := w.(*SelectionContainer); return ok && sc.disabled() })
	if len(found) != 1 {
		t.Fatalf("found %d disabled containers", len(found))
	}
	state := found[0].(*StatefulElement).State().(*selectionContainerState)
	requireTag(t, "SelectionContainerDisabled", func() { state.AddListener(func() {}) })
}

func TestTextSelectionOverlayShowToolbarDuringTheBuild(t *testing.T) {
	skipUnlessDebug(t)
	bt := newBindingTree(t, bindingTreeOpts{collectPanics: true})
	controller, focus, app := internalEditable("hello")
	defer focus.Dispose()
	defer controller.Dispose()
	bt.pump(app)
	state := bt.editableState()
	controller.SetSelection(services.NewTextSelection(0, 5))
	bt.frame()
	overlay := state.createSelectionOverlay()
	defer overlay.Dispose()
	bt.errs = nil
	bt.pump(NewBuilder(func(BuildContext) Widget {
		overlay.ShowToolbar()
		return NewSizedBox(nil)
	}))
	requireReported(t, bt, "ShowToolbarDuringBuild")
}

func TestSelectableRegionReportsTheFailuresOfTheClipboard(t *testing.T) {
	bt := newBindingTree(t)
	bt.b.clipboard = services.NewClipboard(failingClipboard{post: bt.d.Post})
	bt.pump(internalTestApp(NewSelectableRegion(EmptyTextSelectionControls{}, NewCenter(NewText("hello", TextOpts{
		TextDirection: ui.TextDirectionLTR,
	})))))
	found := bt.find(func(w Widget) bool { _, ok := w.(*SelectableRegion); return ok })
	if len(found) != 1 {
		t.Fatalf("found %d selectable regions", len(found))
	}
	state := found[0].(*StatefulElement).State().(*SelectableRegionState)
	state.SelectAll()
	bt.frame()
	bt.errs = nil
	state.CopySelection()
	bt.frame()
	requireReported(t, bt, "SelectableRegionCopyFailed")
}
