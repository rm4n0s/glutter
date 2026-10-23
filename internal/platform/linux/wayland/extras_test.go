//go:build linux && !android

package wayland

import (
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

var (
	activationGlobal  = fakecomp.Global{Name: 30, Interface: "xdg_activation_v1", Version: 1}
	idleInhibitGlobal = fakecomp.Global{Name: 31, Interface: "zwp_idle_inhibit_manager_v1", Version: 1}
	contentTypeGlobal = fakecomp.Global{Name: 32, Interface: "wp_content_type_manager_v1", Version: 1}
	iconGlobal        = fakecomp.Global{Name: 33, Interface: "xdg_toplevel_icon_manager_v1", Version: 1}
	tabletGlobal      = fakecomp.Global{Name: 34, Interface: "zwp_tablet_manager_v2", Version: 2}
	exporterGlobal    = fakecomp.Global{Name: 35, Interface: "zxdg_exporter_v2", Version: 1}
)

// with is a fake compositor that offers the extra globals.
func with(globals ...fakecomp.Global) fakecomp.Config {
	return fakecomp.Config{Globals: append(fakecomp.DefaultGlobals(), globals...)}
}

func TestStartupTokenActivatesTheFirstWindowOnce(t *testing.T) {
	tests := []struct {
		name       string
		compositor fakecomp.Config
		token      string
		want       int
	}{
		{"the token of the launcher is used", with(activationGlobal), "launcher-token", 1},
		{"no token, nothing to do", with(activationGlobal), "", 0},
		{"without the protocol the token is dropped", fakecomp.Config{}, "launcher-token", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_ACTIVATION_TOKEN", tt.token)
			e := startWindows(t, tt.compositor, config640(), config640())
			waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_surface.commit buffer=")) >= 2 }, "both windows are not mapped")
			e.onLoop(func() {})
			got := e.srv.RequestsWith("xdg_activation_v1.activate")
			if len(got) != tt.want {
				t.Fatalf("activate requests = %q, want %d", got, tt.want)
			}
			if tt.want == 1 && !strings.Contains(got[0], "token=launcher-token surface=") {
				t.Errorf("request = %q", got[0])
			}
			if v := os.Getenv("XDG_ACTIVATION_TOKEN"); v != "" {
				t.Errorf("XDG_ACTIVATION_TOKEN = %q after the start, want it removed", v)
			}
			if err := e.stop(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestActivationRequests(t *testing.T) {
	e := startWindows(t, with(activationGlobal), config640())
	e.waitMapped()
	tokens := make(chan string, 1)
	e.onLoop(func() { e.p.inputSerial = 77 })
	e.onLoop(func() { e.p.RequestActivationToken(e.p.windows[0].id, func(tok string) { tokens <- tok }) })
	select {
	case token := <-tokens:
		if token != "fake-token-1" {
			t.Errorf("token = %q", token)
		}
	case <-time.After(testTimeout):
		t.Fatal("the token never arrived")
	}
	for _, want := range []string{
		"xdg_activation_token_v1.set_serial serial=77 seat=true",
		"xdg_activation_token_v1.set_app_id dev.glutter.test",
		"xdg_activation_token_v1.set_surface surface=true",
		"xdg_activation_token_v1.commit",
		"xdg_activation_token_v1.destroy",
	} {
		waitUntil(t, func() bool { return index(e.srv.Requests(), 0, want) >= 0 }, "missing "+want)
	}

	// Activating the window gets a token and uses it.
	e.onLoop(func() { e.p.windows[0].PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionActivate}) })
	waitUntil(t, func() bool {
		return len(e.srv.RequestsWith("xdg_activation_v1.activate token=fake-token-2 surface=true")) == 1
	}, "the window was not activated with the new token")
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestActivationWithoutTheProtocol(t *testing.T) {
	e := startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()
	got, called := "x", false
	e.onLoop(func() {
		e.p.RequestActivationToken(e.p.windows[0].id, func(tok string) { got, called = tok, true })
		e.p.windows[0].PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionActivate})
	})
	if !called || got != "" {
		t.Errorf("token = %q (called %v), want an empty token at once", got, called)
	}
	e.onLoop(func() {
		e.p.RequestActivationToken(99, func(tok string) { got = tok })
	})
	if got != "" {
		t.Errorf("a view that does not exist got the token %q", got)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestKeepScreenOn(t *testing.T) {
	e := startWindows(t, with(idleInhibitGlobal), config640())
	e.waitMapped()
	call := func(on bool) error {
		var err error
		e.onLoop(func() { err = e.p.windows[0].KeepScreenOn(on) })
		return err
	}
	if err := call(true); err != nil {
		t.Fatal(err)
	}
	if err := call(true); err != nil { // already inhibited
		t.Fatal(err)
	}
	waitUntil(t, func() bool {
		return len(e.srv.RequestsWith("zwp_idle_inhibit_manager_v1.create_inhibitor surface=true")) == 1
	}, "no inhibitor was made")
	if err := call(false); err != nil {
		t.Fatal(err)
	}
	if err := call(false); err != nil { // already off
		t.Fatal(err)
	}
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("zwp_idle_inhibitor_v1.destroy")) == 1 }, "the inhibitor was not destroyed")
	if n := len(e.srv.RequestsWith("zwp_idle_inhibit_manager_v1.create_inhibitor")); n != 1 {
		t.Errorf("%d inhibitors were made, want 1", n)
	}
	// An inhibitor goes away with its window.
	if err := call(true); err != nil {
		t.Fatal(err)
	}
	e.onLoop(func() { e.p.windows[0].Close() })
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("zwp_idle_inhibitor_v1.destroy")) == 2 }, "the inhibitor outlived its window")
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestKeepScreenOnWithoutTheProtocol(t *testing.T) {
	e := startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()
	var err error
	e.onLoop(func() { err = e.p.windows[0].KeepScreenOn(true) })
	assertRoute(t, err, "IdleInhibitUnavailable")
	e.onLoop(func() { err = e.p.windows[0].KeepScreenOn(false) })
	if err != nil {
		t.Errorf("letting the screen go idle failed: %v", err)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestContentType(t *testing.T) {
	cfg := config640()
	cfg.ContentType = ui.ContentTypeGame
	e := startWindows(t, with(contentTypeGlobal), cfg)
	e.waitMapped()
	lines := e.srv.Requests()
	set, commit := index(lines, 0, "wp_content_type_v1.set_content_type 3"), index(lines, 0, "wl_surface.commit")
	if set < 0 || commit < 0 || set > commit {
		t.Errorf("the hint must come before the first commit:\n%s", strings.Join(lines, "\n"))
	}
	e.onLoop(func() { e.p.windows[0].SetContentType(ui.ContentTypeVideo) })
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wp_content_type_v1.set_content_type 2")) == 1 }, "the video hint was not sent")
	e.onLoop(func() { e.p.windows[0].SetContentType(ui.ContentTypeUnspecified) })
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wp_content_type_v1.set_content_type 0")) == 1 }, "the hint was not taken back")
	if n := len(e.srv.RequestsWith("get_surface_content_type")); n != 1 {
		t.Errorf("%d content type objects, want 1", n)
	}
	e.onLoop(func() { e.p.windows[0].Close() })
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wp_content_type_v1.destroy")) == 1 }, "the content type object outlived its window")
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestContentTypeNeedsNoObjectForNoHint(t *testing.T) {
	e := startWindows(t, with(contentTypeGlobal), config640())
	e.waitMapped()
	e.onLoop(func() { e.p.windows[0].SetContentType(ui.ContentTypeUnspecified) })
	if n := len(e.srv.RequestsWith("get_surface_content_type")); n != 0 {
		t.Errorf("%d requests for no hint", n)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
	// Without the protocol the hint is ignored.
	cfg := config640()
	cfg.ContentType = ui.ContentTypeVideo
	e = startWindows(t, fakecomp.Config{}, cfg)
	e.waitMapped()
	e.onLoop(func() { e.p.windows[0].SetContentType(ui.ContentTypeGame) })
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func square(size int, r, g, b, a byte) ui.WindowIconImage {
	px := make([]byte, size*size*4)
	for i := 0; i < len(px); i += 4 {
		px[i], px[i+1], px[i+2], px[i+3] = r, g, b, a
	}
	return ui.WindowIconImage{Size: size, RGBA: px}
}

func TestWindowIcon(t *testing.T) {
	cfg := config640()
	cfg.Icon = ui.WindowIcon{Name: "text-editor", Images: []ui.WindowIconImage{square(16, 255, 0, 0, 255), square(32, 0, 255, 0, 255)}}
	e := startWindows(t, with(iconGlobal), cfg)
	e.waitMapped()
	lines := e.srv.Requests()
	want := []string{
		"xdg_toplevel_icon_manager_v1.create_icon",
		"xdg_toplevel_icon_v1.set_name text-editor",
		"xdg_toplevel_icon_v1.add_buffer 16x16 scale=1",
		"xdg_toplevel_icon_v1.add_buffer 32x32 scale=1",
		"xdg_toplevel_icon_manager_v1.set_icon toplevel=true icon=true",
		"xdg_toplevel_icon_v1.destroy",
		"wl_surface.commit",
	}
	pos := -1
	for _, w := range want {
		i := index(lines, pos+1, w)
		if i < 0 {
			t.Fatalf("missing %q (after line %d) in\n%s", w, pos, strings.Join(lines, "\n"))
		}
		pos = i
	}
	if got := e.p.sess.iconSizes; !reflect.DeepEqual(got, []int32{48, 64}) {
		t.Errorf("preferred icon sizes = %v, want [48 64]", got)
	}

	// A new icon replaces the old one; an empty one removes it.
	var err error
	e.onLoop(func() {
		err = e.p.windows[0].SetIcon(ui.WindowIcon{Images: []ui.WindowIconImage{{Size: 8, Scale: 2, RGBA: square(8, 1, 2, 3, 4).RGBA}}})
	})
	if err != nil {
		t.Fatal(err)
	}
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("xdg_toplevel_icon_v1.add_buffer 8x8 scale=2")) == 1 }, "the new icon was not sent")
	e.onLoop(func() { err = e.p.windows[0].SetIcon(ui.WindowIcon{}) })
	if err != nil {
		t.Fatal(err)
	}
	waitUntil(t, func() bool {
		return len(e.srv.RequestsWith("xdg_toplevel_icon_manager_v1.set_icon toplevel=true icon=false")) == 1
	}, "the icon was not removed")
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestWindowIconFailures(t *testing.T) {
	e := startWindows(t, with(iconGlobal), config640())
	e.waitMapped()
	bad := []ui.WindowIcon{
		{Images: []ui.WindowIconImage{{Size: 4, RGBA: make([]byte, 10)}}},
		{Images: []ui.WindowIconImage{{Size: 0}}},
		{Images: []ui.WindowIconImage{{Size: 2, Scale: -1, RGBA: make([]byte, 16)}}},
	}
	for i, icon := range bad {
		var err error
		e.onLoop(func() { err = e.p.windows[0].SetIcon(icon) })
		if err == nil {
			t.Errorf("icon %d was accepted", i)
			continue
		}
		assertRoute(t, err, "WindowIconInvalid")
	}
	if n := len(e.srv.RequestsWith("xdg_toplevel_icon_manager_v1.create_icon")); n != 0 {
		t.Errorf("%d icons were made for invalid input", n)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
	// Without the protocol the icon is ignored.
	e = startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()
	var err error
	e.onLoop(func() { err = e.p.windows[0].SetIcon(ui.WindowIcon{Name: "x"}) })
	if err != nil {
		t.Errorf("SetIcon without the protocol failed: %v", err)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestPremultiplyBGRA(t *testing.T) {
	tests := []struct {
		name string
		rgba []byte
		want []byte
	}{
		{"opaque red", []byte{255, 0, 0, 255}, []byte{0, 0, 255, 255}},
		{"half transparent white", []byte{255, 255, 255, 128}, []byte{128, 128, 128, 128}},
		{"transparent", []byte{200, 100, 50, 0}, []byte{0, 0, 0, 0}},
		{"two pixels", []byte{10, 20, 30, 255, 0, 0, 0, 255}, []byte{30, 20, 10, 255, 0, 0, 0, 255}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]byte, len(tt.rgba))
			premultiplyBGRA(dst, tt.rgba)
			if !reflect.DeepEqual(dst, tt.want) {
				t.Errorf("got %v, want %v", dst, tt.want)
			}
		})
	}
}

func TestTabletStylus(t *testing.T) {
	e := startInput(t, capPointer, 0, tabletGlobal, cursorShapeGlobal)
	if len(e.srv.RequestsWith("zwp_tablet_manager_v2.get_tablet_seat")) != 1 {
		t.Fatal("the tablet seat was not requested")
	}
	pen := e.srv.AddTabletTool(fakecomp.ToolPen)
	e.srv.ToolProximityIn(pen)
	e.srv.ToolSend(pen, "motion", fakecomp.Fixed(30), fakecomp.Fixed(40))
	e.srv.ToolSend(pen, "distance", uint32(32768))
	e.srv.ToolSend(pen, "frame", uint32(1))
	e.srv.ToolSend(pen, "down", uint32(9))
	e.srv.ToolSend(pen, "pressure", uint32(65535))
	e.srv.ToolSend(pen, "tilt", fakecomp.Fixed(0), fakecomp.Fixed(45))
	e.srv.ToolSend(pen, "button", uint32(10), uint32(0x14b), uint32(1))
	e.srv.ToolSend(pen, "frame", uint32(2))
	e.srv.ToolSend(pen, "motion", fakecomp.Fixed(31), fakecomp.Fixed(42))
	e.srv.ToolSend(pen, "frame", uint32(3))
	e.srv.ToolSend(pen, "up")
	e.srv.ToolSend(pen, "pressure", uint32(0))
	e.srv.ToolSend(pen, "frame", uint32(4))
	e.srv.ToolSend(pen, "proximity_out")
	e.srv.ToolSend(pen, "frame", uint32(5))

	var all []ui.PointerData
	for _, b := range e.batches() {
		all = append(all, b...)
	}
	var got []ui.PointerChange
	for _, d := range all {
		got = append(got, d.Change)
	}
	want := []ui.PointerChange{ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeUp, ui.PointerChangeRemove}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("changes = %v, want %v", got, want)
	}
	add, down, move := all[0], all[1], all[2]
	if add.Kind != ui.PointerDeviceKindStylus || add.PhysicalX != 30 || add.PhysicalY != 40 || add.ViewID != 0 {
		t.Errorf("add = %+v", add)
	}
	if down.Pressure != 1 || down.Buttons != 1|2 || math.Abs(down.Tilt-math.Pi/4) > 1e-6 {
		t.Errorf("down pressure %v, buttons %#x, tilt %v", down.Pressure, down.Buttons, down.Tilt)
	}
	if move.PhysicalX != 31 || move.PhysicalDeltaY != 2 {
		t.Errorf("move = %+v", move)
	}
	if n := len(e.srv.RequestsWith("wp_cursor_shape_manager_v1.get_tablet_tool_v2")); n != 1 {
		t.Errorf("%d cursor shape devices for the tool, want 1", n)
	}
	if n := len(e.srv.RequestsWith("wp_cursor_shape_device_v1.set_shape")); n != 1 {
		t.Errorf("%d cursor shapes for the tool, want 1 (on proximity in)", n)
	}

	// The eraser is another kind of device, and a tool that is removed goes away.
	eraser := e.srv.AddTabletTool(fakecomp.ToolEraser)
	e.srv.ToolProximityIn(eraser)
	e.srv.ToolSend(eraser, "frame", uint32(6))
	e.srv.ToolSend(eraser, "removed")
	var kinds []ui.PointerDeviceKind
	var changes []ui.PointerChange
	for _, b := range e.batches() {
		for _, d := range b {
			kinds = append(kinds, d.Kind)
			changes = append(changes, d.Change)
		}
	}
	if !reflect.DeepEqual(kinds, []ui.PointerDeviceKind{ui.PointerDeviceKindInvertedStylus, ui.PointerDeviceKindInvertedStylus}) ||
		!reflect.DeepEqual(changes, []ui.PointerChange{ui.PointerChangeAdd, ui.PointerChangeRemove}) {
		t.Errorf("eraser: kinds %v, changes %v", kinds, changes)
	}
	if n := len(e.srv.RequestsWith("zwp_tablet_tool_v2.destroy")); n != 1 {
		t.Errorf("%d tools destroyed, want 1", n)
	}
}

func TestTabletToolCursor(t *testing.T) {
	e := startInput(t, capPointer, 0, tabletGlobal)
	pen := e.srv.AddTabletTool(fakecomp.ToolPen)
	e.srv.ToolProximityIn(pen)
	e.srv.ToolSend(pen, "frame", uint32(1))
	e.batches()
	// Without wp_cursor_shape_v1 the compositor keeps its cursor for the tool; a
	// window that hides the cursor says so.
	e.onLoop(func() bool { e.p.windows[0].SetCursor(platform.MouseCursorNone); return true })
	e.settle()
	if n := len(e.srv.RequestsWith("zwp_tablet_tool_v2.set_cursor serial=")); n != 1 {
		t.Errorf("%d set_cursor requests, want 1", n)
	}
	if n := len(e.srv.RequestsWith("zwp_tablet_tool_v2.set_cursor serial=")); n == 1 && !strings.Contains(e.srv.RequestsWith("zwp_tablet_tool_v2.set_cursor")[0], "surface=false") {
		t.Error("the cursor was not hidden")
	}
}

func TestTabletToolLeavesWithItsWindow(t *testing.T) {
	e := startInput(t, capPointer, 0, tabletGlobal)
	pen := e.srv.AddTabletTool(fakecomp.ToolPen)
	e.srv.ToolProximityIn(pen)
	e.srv.ToolSend(pen, "frame", uint32(1))
	e.srv.ToolSend(pen, "down", uint32(9))
	e.srv.ToolSend(pen, "frame", uint32(2))
	e.batches()
	e.onLoop(func() bool {
		// The loop stops with the last window, so another one stays.
		if _, err := e.p.NewWindow(config640()); err != nil {
			t.Error(err)
		}
		e.p.windows[0].Close()
		return true
	})
	var changes []ui.PointerChange
	for _, b := range e.batches() {
		for _, d := range b {
			changes = append(changes, d.Change)
		}
	}
	if want := []ui.PointerChange{ui.PointerChangeCancel, ui.PointerChangeRemove}; !reflect.DeepEqual(changes, want) {
		t.Errorf("changes = %v, want %v", changes, want)
	}
}

func TestTabletWithoutTheProtocol(t *testing.T) {
	e := startInput(t, capPointer, 0)
	if e.p.tablet != nil {
		t.Error("a tablet seat without the protocol")
	}
}

func TestPointerPositionsAreInPhysicalPixels(t *testing.T) {
	globals := fakecomp.DefaultGlobals()
	e := startKeyboardNoFocus(t, fakecomp.Config{Capabilities: capPointer | capTouch, Globals: globals, OutputScale: 2, PreferredBufferScale: 2})
	e.srv.PointerEnter(e.surface, 10, 20)
	e.srv.PointerMotion(12, 22)
	e.srv.PointerFrame()
	e.srv.TouchDown(e.surface, 1, 5, 6)
	e.srv.TouchFrame()
	var all []ui.PointerData
	for _, b := range e.batches() {
		all = append(all, b...)
	}
	var pointer, touch []ui.PointerData
	for _, d := range all {
		if d.Kind == ui.PointerDeviceKindTouch {
			touch = append(touch, d)
		} else {
			pointer = append(pointer, d)
		}
	}
	if len(pointer) < 2 || pointer[0].PhysicalX != 20 || pointer[0].PhysicalY != 40 || pointer[1].PhysicalX != 24 || pointer[1].PhysicalDeltaY != 4 {
		t.Errorf("pointer at scale 2 = %+v", pointer)
	}
	if len(touch) < 2 || touch[0].PhysicalX != 10 || touch[0].PhysicalY != 12 {
		t.Errorf("touch at scale 2 = %+v", touch)
	}
}

func TestPortalHandleIsExportedOnceAndKept(t *testing.T) {
	e := startWindows(t, with(exporterGlobal), config640())
	e.waitMapped()
	handles := make(chan string, 4)
	e.onLoop(func() {
		w := e.p.windows[0]
		// Two callers before the handle arrives get the same export.
		w.PortalHandle(func(h string) { handles <- h })
		w.PortalHandle(func(h string) { handles <- h })
	})
	for range 2 {
		select {
		case h := <-handles:
			if h != "wayland:fake-handle-1" {
				t.Errorf("handle = %q", h)
			}
		case <-time.After(testTimeout):
			t.Fatal("the handle never arrived")
		}
	}
	// A later caller gets the kept handle without another export.
	e.onLoop(func() { e.p.windows[0].PortalHandle(func(h string) { handles <- h }) })
	if h := <-handles; h != "wayland:fake-handle-1" {
		t.Errorf("kept handle = %q", h)
	}
	if n := len(e.srv.RequestsWith("zxdg_exporter_v2.export_toplevel surface=true")); n != 1 {
		t.Errorf("%d exports, want 1", n)
	}
	// The export goes away with the window.
	e.onLoop(func() { e.p.windows[0].Close() })
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("zxdg_exported_v2.destroy")) == 1 }, "the export outlived its window")
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}

func TestPortalHandleWithoutTheExporter(t *testing.T) {
	e := startWindows(t, fakecomp.Config{}, config640())
	e.waitMapped()
	got, called := "x", false
	e.onLoop(func() { e.p.windows[0].PortalHandle(func(h string) { got, called = h, true }) })
	if !called || got != "" {
		t.Errorf("handle = %q (called %v), want an empty handle at once", got, called)
	}
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
}
