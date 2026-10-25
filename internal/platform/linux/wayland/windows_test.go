//go:build linux && !android

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
				// The server reads the requests in its own goroutine.
				waitUntil(t, func() bool {
					lines := e.srv.Requests()
					return index(lines, 0, "xdg_dialog_v1.destroy") >= 0 && index(lines, 0, "xdg_toplevel.destroy") >= 0
				}, "the dialog and its toplevel were not both destroyed")
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
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("xdg_toplevel.set_max_size 900x700")) == 1 }, "the maximum size was not sent")
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

// The xdg_toplevel.state values the tests send, and the seat capabilities.
const (
	stateMaximized uint32 = 1
	stateActivated uint32 = 4

	seatPointerOnly uint32 = 1 // a seat without a keyboard, as the headless sway of the tests has
)

// waitFocus waits until the handler was told the view focus events, and fails
// when more come.
func waitFocus(t *testing.T, e *windowEnv, want []string) {
	t.Helper()
	waitUntil(t, func() bool { return len(e.h.focusSeen()) >= len(want) }, "the focus events did not come: "+strings.Join(want, ", "))
	e.onLoop(func() {}) // everything the compositor sent so far was handled
	if got := e.h.focusSeen(); !reflect.DeepEqual(got, want) {
		t.Errorf("view focus events = %q, want %q", got, want)
	}
}

func TestViewFocusIsKnownAtTheFirstConfigure(t *testing.T) {
	for _, seat := range []struct {
		name string
		caps uint32
	}{{"with a keyboard", 0}, {"without a keyboard", seatPointerOnly}} {
		tests := []struct {
			name   string
			states []uint32
			want   []string
		}{
			{"activated", []uint32{stateActivated}, []string{"got 0"}},
			{"activated and maximized", []uint32{stateMaximized, stateActivated}, []string{"got 0"}},
			{"not activated", nil, []string{"lost 0"}},
			{"maximized only", []uint32{stateMaximized}, []string{"lost 0"}},
		}
		for _, tt := range tests {
			t.Run(seat.name+"/"+tt.name, func(t *testing.T) {
				e := startWindows(t, fakecomp.Config{Capabilities: seat.caps, Toplevel: fakecomp.ToplevelConfigure{States: tt.states}}, config640())
				e.waitMapped()
				waitFocus(t, e, tt.want)
				if err := e.stop(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestViewFocusFollowsTheActivatedState(t *testing.T) {
	for _, seat := range []struct {
		name string
		caps uint32
	}{{"with a keyboard", 0}, {"without a keyboard", seatPointerOnly}} {
		t.Run(seat.name, func(t *testing.T) {
			e := startWindows(t, fakecomp.Config{Capabilities: seat.caps}, config640())
			e.waitMapped()
			want := []string{"lost 0"}
			waitFocus(t, e, want)
			steps := []struct {
				name   string
				states []uint32
				event  string // the event it makes, if any
			}{
				{"activated", []uint32{stateActivated}, "got 0"},
				{"configured again, still activated", []uint32{stateMaximized, stateActivated}, ""},
				{"deactivated", []uint32{stateMaximized}, "lost 0"},
				{"configured again, still not", nil, ""},
				{"activated again", []uint32{stateActivated}, "got 0"},
			}
			for _, st := range steps {
				e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{States: st.states})
				if st.event != "" {
					want = append(want, st.event)
				}
				waitFocus(t, e, want)
			}
			if err := e.stop(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// The keys follow wl_keyboard.enter and leave and the view focus follows the
// activated state: neither makes the other.
func TestKeyboardFocusDoesNotMakeViewFocus(t *testing.T) {
	e := startWindows(t, fakecomp.Config{Keymap: keymapFor(t, "us")}, config640())
	e.waitMapped()
	waitFocus(t, e, []string{"lost 0"})
	surface := e.srv.ID("wl_surface")

	e.srv.KeyboardEnter(surface)
	waitUntil(t, func() bool { return len(e.h.keyboardFocusSeen()) == 1 }, "the keyboard focus was not reported")
	e.srv.KeyboardLeave(surface)
	waitUntil(t, func() bool { return len(e.h.keyboardFocusSeen()) == 2 }, "the loss of the keyboard focus was not reported")
	if got, want := e.h.keyboardFocusSeen(), []string{"got 0", "lost 0"}; !reflect.DeepEqual(got, want) {
		t.Errorf("keyboard focus events = %q, want %q", got, want)
	}
	if got := e.h.focusSeen(); !reflect.DeepEqual(got, []string{"lost 0"}) {
		t.Errorf("view focus events = %q after the keyboard moved, want only the first", got)
	}

	// The window is activated while the keyboard is elsewhere.
	e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{States: []uint32{stateActivated}})
	waitFocus(t, e, []string{"lost 0", "got 0"})
	if got := e.h.keyboardFocusSeen(); len(got) != 2 {
		t.Errorf("keyboard focus events = %q after the activation, want the two from before", got)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestAHiddenOrClosedWindowHasNoFocus(t *testing.T) {
	tests := []struct {
		name string
		end  func(e *windowEnv)
		want []string
	}{
		{"hidden", func(e *windowEnv) {
			e.onLoop(func() { e.p.windows[0].SetHidden(true) })
		}, []string{"got 0", "lost 0"}},
		{"hidden and shown again", func(e *windowEnv) {
			e.onLoop(func() { e.p.windows[0].SetHidden(true) })
			e.onLoop(func() { e.p.windows[0].SetHidden(false) })
		}, []string{"got 0", "lost 0", "got 0"}},
		{"closed", func(e *windowEnv) {
			e.onLoop(func() { e.p.windows[0].Close() })
		}, []string{"got 0", "lost 0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := startWindows(t, fakecomp.Config{Toplevel: fakecomp.ToplevelConfigure{States: []uint32{stateActivated}}}, config640(), config640())
			e.waitMapped()
			waitFocus(t, e, []string{"got 0", "got 1"})
			tt.end(e)
			// The other window is left as it was; the first one reports its loss.
			waitUntil(t, func() bool { return len(e.h.focusSeen()) >= 2+len(tt.want)-1 }, "the focus events did not come")
			var first []string
			for _, ev := range e.h.focusSeen() {
				if strings.HasSuffix(ev, " 0") {
					first = append(first, ev)
				}
			}
			if !reflect.DeepEqual(first, tt.want) {
				t.Errorf("view focus events of the first window = %q, want %q", first, tt.want)
			}
		})
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

// A window created maximized, minimized or fullscreen asks for that state before its
// first commit, so the compositor configures it in that state and it never shows at its
// size first (step 27.4).
func TestAWindowIsCreatedInItsState(t *testing.T) {
	tests := []struct {
		name  string
		state platform.WindowState
		want  string // the request before the first commit
	}{
		{"maximized", platform.WindowStateMaximized, "xdg_toplevel.set_maximized"},
		{"minimized", platform.WindowStateMinimized, "xdg_toplevel.set_minimized"},
		{"fullscreen", platform.WindowStateFullscreen, "xdg_toplevel.set_fullscreen"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config640()
			cfg.State = tt.state
			e := startWindows(t, fakecomp.Config{}, cfg)
			e.waitMapped()
			lines := e.srv.Requests()
			state, commit := index(lines, 0, tt.want), index(lines, 0, "wl_surface.commit")
			if state < 0 || commit < 0 || state > commit {
				t.Errorf("%s is at %d and the first commit at %d in:\n%s", tt.want, state, commit, strings.Join(lines, "\n"))
			}
			// The other states are not asked for.
			for _, other := range []string{"xdg_toplevel.set_maximized", "xdg_toplevel.set_minimized", "xdg_toplevel.set_fullscreen"} {
				if other != tt.want && index(lines, 0, other) >= 0 {
					t.Errorf("%s was sent for a window created in the state %v", other, tt.state)
				}
			}
			if err := e.stop(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
