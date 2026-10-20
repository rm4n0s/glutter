package wayland

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

// lastWindowInfo is the window info of the newest metrics report.
func (h *recHandler) lastWindowInfo() (ui.WindowInfo, bool) {
	m := h.metricsSeen()
	if len(m) == 0 {
		return ui.WindowInfo{}, false
	}
	return m[len(m)-1].Window, true
}

// TestDecorationMode checks who draws the title bar, by what the compositor has.
// The platform has no libdecor here (Options.NoLibdecor): it needs a real
// compositor, and the display test of this package covers it.
func TestDecorationMode(t *testing.T) {
	withManager := append(fakecomp.DefaultGlobals(), fakecomp.Global{Name: 50, Interface: "zxdg_decoration_manager_v1", Version: 1})
	tests := []struct {
		name    string
		globals []fakecomp.Global
		mutate  func(*platform.WindowConfig)
		want    ui.DecorationMode
	}{
		{"the compositor draws it", withManager, func(*platform.WindowConfig) {}, ui.DecorationServerSide},
		{"no protocol and no libdecor: the app draws it", nil, func(*platform.WindowConfig) {}, ui.DecorationApp},
		{"the app asked to draw it", withManager, func(c *platform.WindowConfig) { c.Decorations = platform.DecorationsApp }, ui.DecorationApp},
		{"auto is the default", withManager, func(c *platform.WindowConfig) { c.Decorations = platform.DecorationsAuto }, ui.DecorationServerSide},
		{"not decorated", withManager, func(c *platform.WindowConfig) { c.Decorated = foundation.Some(false) }, ui.DecorationNone},
		{"a borderless window", withManager, func(c *platform.WindowConfig) { c.Kind = platform.WindowKindBorderless }, ui.DecorationNone},
		{"libdecor asked for but missing: auto", withManager, func(c *platform.WindowConfig) { c.Decorations = platform.DecorationsLibdecor }, ui.DecorationServerSide},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config640()
			tt.mutate(&cfg)
			e := startWindows(t, fakecomp.Config{Globals: tt.globals}, cfg)
			e.waitMapped()
			waitUntil(t, func() bool { _, ok := e.h.lastWindowInfo(); return ok }, "no metrics")
			info, _ := e.h.lastWindowInfo()
			if info.Decoration != tt.want {
				t.Errorf("decoration = %v, want %v", info.Decoration, tt.want)
			}
			if info.Title != cfg.Title {
				t.Errorf("title = %q, want %q", info.Title, cfg.Title)
			}
			if err := e.stop(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestWindowInfoFollowsTheCompositor checks the state and the capabilities that
// the compositor reports, and that a change in them is reported even when the size
// stays.
func TestWindowInfoFollowsTheCompositor(t *testing.T) {
	e := startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()
	e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{Width: 640, Height: 400, States: []uint32{4}, Capabilities: []uint32{1, 2}})
	waitUntil(t, func() bool { i, ok := e.h.lastWindowInfo(); return ok && i.Activated }, "the window is never reported active")
	info, _ := e.h.lastWindowInfo()
	if info.Maximized || !info.Activated {
		t.Errorf("state = maximized %v, activated %v", info.Maximized, info.Activated)
	}
	if want := (ui.WindowCapabilities{WindowMenu: true, Maximize: true}); info.Capabilities != want {
		t.Errorf("capabilities = %+v, want %+v", info.Capabilities, want)
	}
	// Maximized, with the same size: a new report.
	e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{Width: 640, Height: 400, States: []uint32{1, 4}})
	waitUntil(t, func() bool { i, ok := e.h.lastWindowInfo(); return ok && i.Maximized }, "maximizing was not reported")
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

// TestPerformWindowAction checks the requests of the actions of a title bar that
// the app draws.
func TestPerformWindowAction(t *testing.T) {
	tests := []struct {
		name   string
		action ui.WindowAction
		want   string
	}{
		{"move", ui.WindowAction{Kind: ui.WindowActionMove}, "xdg_toplevel.move"},
		{"resize from the bottom right", ui.WindowAction{Kind: ui.WindowActionResize, Edge: ui.ResizeEdgeBottomRight}, "xdg_toplevel.resize serial=0 edges=10"},
		{"resize from the top", ui.WindowAction{Kind: ui.WindowActionResize, Edge: ui.ResizeEdgeTop}, "xdg_toplevel.resize serial=0 edges=1"},
		{"the window menu", ui.WindowAction{Kind: ui.WindowActionShowMenu, Position: ui.Offset{Dx: 10, Dy: 20}}, "xdg_toplevel.show_window_menu serial=0 at 10,20"},
		{"minimize", ui.WindowAction{Kind: ui.WindowActionMinimize}, "xdg_toplevel.set_minimized"},
		{"maximize", ui.WindowAction{Kind: ui.WindowActionMaximize}, "xdg_toplevel.set_maximized"},
		{"restore", ui.WindowAction{Kind: ui.WindowActionRestore}, "xdg_toplevel.unset_maximized"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := startWindows(t, fakecomp.Config{}, config640())
			e.waitMapped()
			// The window menu needs the capability; the compositor says it has it.
			e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{Width: 640, Height: 400, Capabilities: []uint32{1, 2, 3, 4}})
			waitUntil(t, func() bool { i, ok := e.h.lastWindowInfo(); return ok && i.Capabilities.WindowMenu }, "no capabilities")
			e.onLoop(func() { e.p.windows[0].PerformWindowAction(tt.action) })
			waitUntil(t, func() bool {
				for _, line := range e.srv.Requests() {
					if strings.HasPrefix(line, tt.want) {
						return true
					}
				}
				return false
			}, "the request "+tt.want+" was not sent; got "+strings.Join(e.srv.Requests(), "\n"))
			if err := e.stop(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPerformWindowActionClose(t *testing.T) {
	e := startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()
	e.onLoop(func() { e.p.windows[0].PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionClose}) })
	if got := e.h.closeRequests(); len(got) != 1 {
		t.Errorf("close requests = %v, want one", got)
	}
	if err := wait(t, e.done); err != nil {
		t.Fatalf("Run returned %v after the window closed", err)
	}
	e.stopped = true
}
