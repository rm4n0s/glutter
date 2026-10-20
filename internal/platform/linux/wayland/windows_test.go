package wayland

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

// withDialogManager is a fake compositor that offers xdg_wm_dialog_v1.
func withDialogManager() fakecomp.Config {
	return fakecomp.Config{Globals: append(fakecomp.DefaultGlobals(), fakecomp.Global{Name: 8, Interface: "xdg_wm_dialog_v1", Version: 1})}
}

func TestDialogs(t *testing.T) {
	tests := []struct {
		name         string
		compositor   fakecomp.Config
		modal        bool
		hasParent    bool
		wantDialog   bool
		wantSetModal bool
	}{
		{"a modal dialog is an xdg_dialog_v1", withDialogManager(), true, true, true, true},
		{"a dialog that is not modal needs no xdg_dialog_v1", withDialogManager(), false, true, false, false},
		{"a dialog with no parent is not modal", withDialogManager(), true, false, false, false},
		{"without the protocol the dialog is only transient", fakecomp.Config{}, true, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialog := config640()
			dialog.Kind = platform.WindowKindDialog
			dialog.HasParent, dialog.Parent, dialog.Modal = tt.hasParent, 0, tt.modal
			e := startWindows(t, tt.compositor, config640(), dialog)
			waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_surface.commit buffer=")) >= 2 }, "both windows are not mapped")

			if got := len(e.srv.RequestsWith("xdg_wm_dialog_v1.get_xdg_dialog")) == 1; got != tt.wantDialog {
				t.Errorf("get_xdg_dialog = %v, want %v", got, tt.wantDialog)
			}
			if got := len(e.srv.RequestsWith("xdg_dialog_v1.set_modal")) == 1; got != tt.wantSetModal {
				t.Errorf("set_modal = %v, want %v", got, tt.wantSetModal)
			}
			if got := len(e.srv.RequestsWith("xdg_toplevel.set_parent true")) == 1; got != tt.hasParent {
				t.Errorf("set_parent = %v, want %v", got, tt.hasParent)
			}
			// Closing the dialog destroys its xdg_dialog_v1 before the toplevel.
			e.onLoop(func() { e.p.windows[1].Close() })
			if tt.wantDialog {
				lines := e.srv.Requests()
				if index(lines, 0, "xdg_dialog_v1.destroy") > index(lines, 0, "xdg_toplevel.destroy") {
					t.Error("the dialog object was destroyed after its toplevel")
				}
			}
			if err := e.stop(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSizeConstraints(t *testing.T) {
	e := startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()
	e.onLoop(func() {
		e.p.windows[0].SetSizeConstraints(ui.Size{Width: 300, Height: 200}, ui.Size{Width: 900, Height: 700})
	})
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("xdg_toplevel.set_min_size 300x200")) == 1 }, "the minimum size was not sent")
	if got := e.srv.RequestsWith("xdg_toplevel.set_max_size 900x700"); len(got) != 1 {
		t.Errorf("set_max_size requests = %q", got)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestAWindowThatCannotBeResizedHasOneSize(t *testing.T) {
	cfg := config640()
	cfg.Resizable = foundation.Some(false)
	e := startWindows(t, fakecomp.Config{}, cfg)
	e.waitMapped()
	lines := e.srv.Requests()
	var got []string
	for _, l := range lines {
		if strings.HasPrefix(l, "xdg_toplevel.set_m") && !strings.Contains(l, " 0x0") {
			got = append(got, l)
		}
	}
	if want := []string{"xdg_toplevel.set_min_size 640x400", "xdg_toplevel.set_max_size 640x400"}; !reflect.DeepEqual(got, want) {
		// The order of the two requests is the order of the code; the sets are what counts.
		if len(got) != 2 || !contains(got, want[0]) || !contains(got, want[1]) {
			t.Errorf("size requests = %q, want %q", got, want)
		}
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func TestKeyboardFocusIsReportedAsViewFocus(t *testing.T) {
	e := startKeyboardNoFocus(t, fakecomp.Config{Keymap: keymapFor(t, "us")})
	e.srv.KeyboardEnter(e.surface)
	waitUntil(t, func() bool { return len(e.h.focusEvents()) == 1 }, "the focus was not reported")
	e.srv.KeyboardLeave(e.surface)
	waitUntil(t, func() bool { return len(e.h.focusEvents()) == 2 }, "the loss of the focus was not reported")
	if got, want := e.h.focusEvents(), []string{"got 0", "lost 0"}; !reflect.DeepEqual(got, want) {
		t.Errorf("focus events = %q, want %q", got, want)
	}
}

func TestWindowActionsForTheFrameworkWindows(t *testing.T) {
	e := startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()
	e.onLoop(func() {
		w := e.p.windows[0]
		w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionFullscreen})
		w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionUnfullscreen})
		w.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionActivate}) // xdg-shell cannot do it
	})
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("xdg_toplevel.unset_fullscreen")) >= 1 }, "the actions did not arrive")
	if len(e.srv.RequestsWith("xdg_toplevel.set_fullscreen")) != 1 {
		t.Error("the window was not made fullscreen")
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}
