package widgets

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// These tests check the failures of the editable text that need unexported
// identifiers, or a host without a system clipboard (the host of bindingTree).

func internalTestApp(home Widget) Widget {
	return NewWidgetsApp(ui.NewColor(0xFFFFFFFF), WidgetsAppOpts{
		Home: home,
		PageRouteBuilder: func(settings RouteSettings, builder WidgetBuilder) PageRoute {
			return NewPageRouteBuilder(
				func(bc BuildContext, _, _ animation.Animation[float64]) Widget { return builder(bc) },
				PageRouteBuilderOpts{
					PageRouteOpts: PageRouteOpts{RouteOpts: RouteOpts{Settings: settings}},
					TransitionsBuilder: func(_ BuildContext, _, _ animation.Animation[float64], child Widget) Widget {
						return child
					},
				})
		},
	})
}

func internalEditable(text string) (*TextEditingController, *FocusNode, Widget) {
	controller := NewTextEditingController(text)
	focus := NewFocusNode(FocusNodeOpts{DebugLabel: "editable"})
	field := NewEditableText(controller, focus,
		painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto", FontSize: foundation.Some(16.0), Color: foundation.Some(ui.NewColor(0xFF000000))}),
		ui.NewColor(0xFF000000), ui.NewColor(0xFF888888))
	return controller, focus, internalTestApp(NewAlign(NewSizedBox(field, SizedBoxOpts{Width: foundation.Some(200.0)})))
}

func (bt *bindingTree) editableState() *EditableTextState {
	bt.t.Helper()
	found := bt.find(func(w Widget) bool { _, ok := w.(*EditableText); return ok })
	if len(found) != 1 {
		bt.t.Fatalf("found %d editable texts", len(found))
	}
	return found[0].(*StatefulElement).State().(*EditableTextState)
}

func tagsOf(errs []error) []string {
	var tags []string
	for _, err := range errs {
		if appErr, ok := errors.FromError(err); ok {
			tags = append(tags, appErr.Route())
		} else {
			tags = append(tags, err.Error())
		}
	}
	return tags
}

func requireReported(t *testing.T, bt *bindingTree, tag string) {
	t.Helper()
	for _, err := range bt.errs {
		if appErr, ok := errors.FromError(err); ok && appErr.HasRoute(tag) {
			return
		}
	}
	t.Errorf("no error with the tag %s was reported; reported: %v", tag, tagsOf(bt.errs))
}

func TestEditableTextStateBeforeItIsBuilt(t *testing.T) {
	skipUnlessDebug(t)
	s := newEditableTextState()
	if s.RenderEditable() != nil {
		t.Error("RenderEditable is not nil before the build")
	}
	requireTag(t, "EditableTextNotBuilt", func() { s.renderEditable() })
	requireTag(t, "EditableTextNotMounted", func() { s.textEditingValueForTextLayoutMetrics() })
}

// failingClipboard is a clipboard backend whose every call fails, later, as the
// calls of the engine do.
type failingClipboard struct{ post func(func()) }

func (c failingClipboard) SetClipboardText(_ context.Context, _ string, done func(error)) {
	if done != nil {
		c.post(func() { done(errors.New("ClipboardTestFailure", "the clipboard is broken")) })
	}
}

func (c failingClipboard) GetClipboardText(_ context.Context, done func(string, bool, error)) {
	c.post(func() { done("", false, errors.New("ClipboardTestFailure", "the clipboard is broken")) })
}

func TestEditableTextReportsTheFailuresOfTheClipboard(t *testing.T) {
	bt := newBindingTree(t)
	bt.b.clipboard = services.NewClipboard(failingClipboard{post: bt.d.Post})
	controller, focus, app := internalEditable("hello")
	defer focus.Dispose()
	defer controller.Dispose()
	bt.pump(app)
	state := bt.editableState()
	controller.SetSelection(services.NewTextSelection(0, 5))
	bt.frame()

	t.Run("copy", func(t *testing.T) {
		bt.b.UIThread().Adopt() // the subtests run on other goroutines
		bt.errs = nil
		state.CopySelection(services.SelectionChangedCauseKeyboard)
		bt.frame()
		requireReported(t, bt, "EditableTextClipboardFailed")
	})
	t.Run("paste", func(t *testing.T) {
		bt.b.UIThread().Adopt()
		bt.errs = nil
		state.PasteText(services.SelectionChangedCauseKeyboard)
		bt.frame()
		requireReported(t, bt, "EditableTextPasteFailed")
	})
	t.Run("the status of the clipboard stays unknown", func(t *testing.T) {
		bt.b.UIThread().Adopt()
		bt.errs = nil
		n := NewClipboardStatusNotifier(bt.b)
		defer n.Dispose()
		n.Update()
		bt.frame()
		requireReported(t, bt, "ClipboardStatusCheckFailed")
		if n.Value() != ClipboardStatusUnknown {
			t.Errorf("status = %v, want unknown", n.Value())
		}
	})
}

func TestEditableTextActionsFailures(t *testing.T) {
	bt := newBindingTree(t)
	controller, focus, app := internalEditable("hello")
	defer focus.Dispose()
	defer controller.Dispose()
	bt.pump(app)
	state := bt.editableState()
	t.Run("UpdateSelectionOutOfRange", func(t *testing.T) {
		skipUnlessDebug(t)
		value := state.TextEditingValue()
		requireTag(t, "UpdateSelectionOutOfRange", func() {
			state.updateSelection(NewUpdateSelectionIntent(value, services.NewTextSelectionCollapsed(len(value.Text)+1), services.SelectionChangedCauseKeyboard))
		})
	})
	t.Run("UpdateSelection in range", func(t *testing.T) {
		value := state.TextEditingValue()
		state.updateSelection(NewUpdateSelectionIntent(value, services.NewTextSelectionCollapsed(2), services.SelectionChangedCauseKeyboard))
		if got := controller.Selection(); got.BaseOffset != 2 || got.ExtentOffset != 2 {
			t.Errorf("selection = %v", got)
		}
	})
}

func TestTapOutsideUnfocuses(t *testing.T) {
	mobile := []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia}
	desktop := []foundation.TargetPlatform{foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows}
	for _, p := range mobile {
		for kind, want := range map[ui.PointerDeviceKind]bool{ui.PointerDeviceKindTouch: false, ui.PointerDeviceKindMouse: true, ui.PointerDeviceKindStylus: true} {
			if got := tapOutsideUnfocuses(p, kind); got != want {
				t.Errorf("%v, %v: unfocuses = %v, want %v", p, kind, got, want)
			}
		}
		requireTag(t, "EditableTextTapOutsideTrackpad", func() { tapOutsideUnfocuses(p, ui.PointerDeviceKindTrackpad) })
	}
	for _, p := range desktop {
		for _, kind := range []ui.PointerDeviceKind{ui.PointerDeviceKindTouch, ui.PointerDeviceKindMouse, ui.PointerDeviceKindTrackpad} {
			if !tapOutsideUnfocuses(p, kind) {
				t.Errorf("%v, %v: does not unfocus", p, kind)
			}
		}
	}
}
