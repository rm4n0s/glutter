//go:build linux && !android

package wayland

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

// hideGlobals are the default globals with the server-side decorations, the idle
// inhibitor and the activation, which a hidden window has to keep or make again.
func hideGlobals() []fakecomp.Global {
	return append(fakecomp.DefaultGlobals(),
		fakecomp.Global{Name: 50, Interface: "zxdg_decoration_manager_v1", Version: 1},
		fakecomp.Global{Name: 51, Interface: "zwp_idle_inhibit_manager_v1", Version: 1},
		fakecomp.Global{Name: 52, Interface: "xdg_activation_v1", Version: 1},
	)
}

// inOrder fails the test unless every line of want is in lines, in that order, from
// the line at.
func inOrder(t *testing.T, lines []string, at int, want []string) {
	t.Helper()
	for _, w := range want {
		i := index(lines, at, w)
		if i < 0 {
			t.Fatalf("%q is missing, or comes too early, in:\n%s", w, strings.Join(lines[min(at, len(lines)):], "\n"))
		}
		at = i + 1
	}
}

func TestHidingAWindowUnmapsItAndShowingMapsItAgain(t *testing.T) {
	tests := []struct {
		name    string
		globals []fakecomp.Global
		// hidden are the requests that hide the window, in order.
		hidden []string
		// shown are the requests that show it again, in order.
		shown []string
	}{
		{"with server-side decorations", hideGlobals(),
			[]string{"wl_surface.attach buffer=0", "wl_surface.commit buffer=null",
				"zxdg_toplevel_decoration_v1.destroy", "xdg_toplevel.destroy", "xdg_surface.destroy"},
			[]string{"xdg_wm_base.get_xdg_surface", "xdg_surface.get_toplevel", `xdg_toplevel.set_title "Renamed"`,
				`xdg_toplevel.set_app_id "dev.glutter.test"`, "xdg_toplevel.set_maximized",
				"zxdg_decoration_manager_v1.get_toplevel_decoration", "zxdg_toplevel_decoration_v1.set_mode 2",
				"wl_surface.commit", "xdg_surface.ack_configure", "wl_surface.attach", "wl_surface.commit buffer=",
				"xdg_activation_v1.get_activation_token", "xdg_activation_v1.activate"}},
		{"without the decoration protocol", nil,
			[]string{"wl_surface.attach buffer=0", "wl_surface.commit buffer=null", "xdg_toplevel.destroy", "xdg_surface.destroy"},
			[]string{"xdg_wm_base.get_xdg_surface", "xdg_surface.get_toplevel", `xdg_toplevel.set_title "Renamed"`,
				"wl_surface.commit", "xdg_surface.ack_configure", "wl_surface.attach", "wl_surface.commit buffer="}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := startWindows(t, fakecomp.Config{Globals: tt.globals}, config640())
			e.waitMapped()
			var w *window
			e.onLoop(func() {
				w = e.p.windows[0]
				w.states.Maximized = true // as the compositor said
				if e.p.sess.idleInhibit != nil {
					if err := w.KeepScreenOn(true); err != nil {
						t.Error(err)
					}
				}
			})

			before := len(e.srv.Requests())
			e.onLoop(func() {
				if !w.SetHidden(true) {
					t.Error("SetHidden says the platform cannot hide windows")
				}
			})
			waitUntil(t, func() bool { return len(e.srv.RequestsWith("xdg_surface.destroy")) > 0 }, "the role was never destroyed")
			lines := e.srv.Requests()
			inOrder(t, lines, before, tt.hidden)
			if _, _, ok := e.srv.Mapped(); ok {
				t.Error("the hidden window still has a buffer on the screen")
			}
			for _, gone := range []string{"wl_surface.destroy", "zwp_idle_inhibitor_v1.destroy"} {
				if len(e.srv.RequestsWith(gone)) != 0 {
					t.Errorf("%s was sent: the surface and what hangs on it stay", gone)
				}
			}
			select {
			case err := <-e.done:
				e.stopped = true
				t.Fatalf("Run returned (%v) while a hidden window exists", err)
			case <-time.After(30 * time.Millisecond):
			}

			before = len(e.srv.Requests())
			e.onLoop(func() {
				w.SetTitle("Renamed")                  // goes to the next toplevel
				w.SetState(platform.WindowStateNormal) // remembered, nothing sent
				w.SetState(platform.WindowStateMaximized)
				w.SetHidden(false)
			})
			if width, height := e.waitMapped(); width != 640 || height != 400 {
				t.Errorf("shown again at %dx%d, want 640x400", width, height)
			}
			if len(tt.shown) > 0 && strings.HasPrefix(tt.shown[len(tt.shown)-1], "xdg_activation") {
				waitUntil(t, func() bool { return len(e.srv.RequestsWith("xdg_activation_v1.activate")) > 0 }, "the shown window did not ask for the focus")
			}
			inOrder(t, e.srv.Requests(), before, tt.shown)

			e.onLoop(func() { w.Close() })
			if err := wait(t, e.done); err != nil {
				t.Fatalf("Run returned %v after the last window closed", err)
			}
			e.stopped = true
		})
	}
}

func TestClosingAHiddenWindowEndsRun(t *testing.T) {
	e := startWindows(t, fakecomp.Config{Globals: hideGlobals()}, config640())
	e.waitMapped()
	e.onLoop(func() { e.p.windows[0].SetHidden(true) })
	e.onLoop(func() { e.p.windows[0].Close() })
	if err := wait(t, e.done); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
	e.stopped = true
	// The role was gone already: it is not destroyed twice.
	if n := len(e.srv.RequestsWith("xdg_toplevel.destroy")); n != 1 {
		t.Errorf("xdg_toplevel.destroy sent %d times, want once", n)
	}
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_surface.destroy")) == 1 }, "the surface was never destroyed")
}

func TestAHiddenWindowIgnoresTheRequestsThatNeedAToplevel(t *testing.T) {
	e := startWindows(t, fakecomp.Config{Globals: hideGlobals()}, config640())
	e.waitMapped()
	var handle string
	e.onLoop(func() {
		w := e.p.windows[0]
		w.SetHidden(true)
		for _, kind := range []ui.WindowActionKind{ui.WindowActionMinimize, ui.WindowActionMaximize, ui.WindowActionRestore,
			ui.WindowActionFullscreen, ui.WindowActionUnfullscreen, ui.WindowActionActivate, ui.WindowActionMove,
			ui.WindowActionResize, ui.WindowActionShowMenu} {
			w.PerformWindowAction(ui.WindowAction{Kind: kind})
		}
		w.SetSizeConstraints(ui.Size{Width: 100, Height: 100}, ui.Size{})
		w.SetSize(ui.Size{Width: 300, Height: 200})
		w.PortalHandle(func(h string) { handle = h })
	})
	if handle != "" {
		t.Errorf("a hidden window was exported as %q", handle)
	}
	if n := len(e.srv.RequestsWith("xdg_activation_v1.activate")); n != 0 {
		t.Error("a hidden window asked for the focus")
	}
	// It comes back at the size asked for meanwhile.
	e.onLoop(func() { e.p.windows[0].SetHidden(false) })
	waitUntil(t, func() bool { w, h, ok := e.srv.Mapped(); return ok && w == 300 && h == 200 }, "the window did not come back at 300x200")
}

func TestEGLWindowDrawsNothingWhileHiddenOrBeforeItsNewConfigure(t *testing.T) {
	e := startEGLWindow(t, true, fakecomp.ToplevelConfigure{})
	waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "the first frame was never requested")
	// The callback of the first frame is held: hiding forgets it.
	e.onLoop(func() {
		w := e.win.(*window)
		w.SetHidden(true)
		w.RequestFrame()
	})
	time.Sleep(30 * time.Millisecond)
	e.srv.FrameDone() // the stale callback comes late
	time.Sleep(30 * time.Millisecond)
	if n := e.h.frameCount(); n != 1 {
		t.Fatalf("%d frames while hidden, want none after the first", n-1)
	}

	var configures int
	e.onLoop(func() {
		w := e.win.(*window)
		configures = w.configures
		w.SetHidden(false)
		w.RequestFrame()
		if e.h.frameCount() != 1 {
			t.Error("a frame was drawn before the configure of the new toplevel")
		}
	})
	waitUntil(t, func() bool { return e.h.frameCount() == 2 }, "the shown window was never drawn")
	e.h.mu.Lock()
	at := e.h.configures[1]
	e.h.mu.Unlock()
	if at != configures+1 {
		t.Errorf("the frame after the show was drawn with %d configures acknowledged, want %d", at, configures+1)
	}
	time.Sleep(30 * time.Millisecond)
	if n := e.h.frameCount(); n != 2 {
		t.Errorf("%d frames after the show, want one until its callback comes", n-1)
	}
}

func TestHidingAgainBeforeTheConfigureKeepsTheState(t *testing.T) {
	e := startWindows(t, fakecomp.Config{Toplevel: fakecomp.ToplevelConfigure{States: []uint32{1}}}, config640()) // maximized
	e.waitMapped()
	e.onLoop(func() {
		w := e.p.windows[0]
		if !w.states.Maximized {
			t.Fatal("the compositor's configure did not maximize the window")
		}
		// No configure comes between these: the second hide must not take the
		// state of the toplevel that was never configured.
		w.SetHidden(true)
		w.SetHidden(false)
		w.SetHidden(true)
		w.SetHidden(false)
	})
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("xdg_toplevel.set_maximized")) == 2 },
		"each new toplevel did not ask to be maximized again")
}
