//go:build linux && !android

package wayland

import (
	"fmt"
	"github.com/rm4n0s/glutter/foundation"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

// The scale, the suspended state and the order of a resize, against the fake
// compositor (SPEC §8.11).

func scaleGlobals() []fakecomp.Global {
	return append(fakecomp.DefaultGlobals(),
		fakecomp.Global{Name: 20, Interface: "wp_viewporter", Version: 1},
		fakecomp.Global{Name: 21, Interface: "wp_fractional_scale_manager_v1", Version: 1})
}

const suspendedState = 9 // xdg_toplevel.state.suspended

// lastMetrics returns the newest metrics report.
func (e *eglEnv) lastMetrics() platform.ViewMetrics {
	m := e.h.metricsSeen()
	if len(m) == 0 {
		return platform.ViewMetrics{}
	}
	last := m[len(m)-1]
	last.Window = ui.WindowInfo{} // these tests are about the size and the scale
	last.Display = foundation.None[int]()
	return last
}

func (e *eglEnv) waitMetrics(want platform.ViewMetrics) {
	e.t.Helper()
	waitUntil(e.t, func() bool { return e.lastMetrics() == want }, fmt.Sprintf("the metrics never became %+v (last: %+v)", want, e.lastMetrics()))
}

func (e *eglEnv) resizeCalls() []resizeCall {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]resizeCall(nil), e.resizes...)
}

func TestFractionalScaleSizesTheBufferAndMapsItWithAViewport(t *testing.T) {
	e := startEGLWindowWith(t, fakecomp.Config{Globals: scaleGlobals()})
	waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "the first frame was never requested")
	if m := e.lastMetrics(); m.PhysicalSize != (ui.Size{Width: 640, Height: 400}) || m.DevicePixelRatio != 1 {
		t.Fatalf("metrics without a preferred scale = %+v", m)
	}
	if lines := e.srv.RequestsWith("wp_"); len(lines) != 4 { // the manager, its object, the viewporter, the viewport, in some order
		if !strings.Contains(strings.Join(lines, "\n"), "get_fractional_scale") || !strings.Contains(strings.Join(lines, "\n"), "get_viewport") {
			t.Errorf("requests = %v", lines)
		}
	}

	// 150%: the buffer is 960x600, and the viewport maps it to the window's 640x400.
	e.srv.SendPreferredScale(180)
	e.waitMetrics(platform.ViewMetrics{PhysicalSize: ui.Size{Width: 960, Height: 600}, DevicePixelRatio: 1.5})
	waitUntil(t, func() bool { return e.h.frameCount() == 2 }, "a new scale did not render a frame")
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wp_viewport.set_destination")) >= 1 }, "no viewport destination")
	if got := e.srv.RequestsWith("wp_viewport.set_destination"); len(got) != 1 || got[0] != "wp_viewport.set_destination 640x400" {
		t.Errorf("viewport = %v", got)
	}
	if got := e.srv.RequestsWith("set_buffer_scale"); len(got) != 0 {
		t.Errorf("a fractional scale set the buffer scale: %v", got)
	}
	resizes := e.resizeCalls()
	if last := resizes[len(resizes)-1]; last.width != 960 || last.height != 600 {
		t.Errorf("wl_egl_window_resize = %+v", last)
	}
	// The viewport is set before the commit of the frame that has the new buffer.
	waitUntil(t, func() bool {
		lines := e.srv.Requests()
		dest := index(lines, 0, "wp_viewport.set_destination 640x400")
		return dest >= 0 && index(lines, dest, "wl_surface.commit") >= 0
	}, "the viewport must come before a commit")

	// A configure at 150%: sizes round up (ceil(333 * 1.5) = 500, ceil(201 * 1.5) = 302).
	e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{Width: 333, Height: 201})
	e.waitMetrics(platform.ViewMetrics{PhysicalSize: ui.Size{Width: 500, Height: 302}, DevicePixelRatio: 1.5})
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wp_viewport.set_destination 333x201")) == 1 }, "the viewport did not follow the configure")

	// 125% and 175%, as GNOME and KDE offer them.
	e.srv.SendPreferredScale(150)
	e.waitMetrics(platform.ViewMetrics{PhysicalSize: ui.Size{Width: 417, Height: 252}, DevicePixelRatio: 1.25})
	e.srv.SendPreferredScale(210)
	e.waitMetrics(platform.ViewMetrics{PhysicalSize: ui.Size{Width: 583, Height: 352}, DevicePixelRatio: 1.75})
	// The same scale again changes nothing. (The frame of the last change is drawn just
	// after its metrics are reported.)
	time.Sleep(50 * time.Millisecond)
	n := e.h.frameCount()
	e.srv.SendPreferredScale(210)
	time.Sleep(50 * time.Millisecond)
	if e.h.frameCount() != n {
		t.Error("an unchanged scale rendered a frame")
	}
	// Back to 100%: the viewport stays in use, with a buffer of the window's size.
	e.srv.SendPreferredScale(120)
	e.waitMetrics(platform.ViewMetrics{PhysicalSize: ui.Size{Width: 333, Height: 201}, DevicePixelRatio: 1})
}

func TestResizeIsRenderedBetweenTheBufferResizeAndTheCommit(t *testing.T) {
	e := startEGLWindowWith(t, fakecomp.Config{Globals: scaleGlobals(), HoldFrames: true})
	waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "the first frame was never requested")
	e.srv.SendPreferredScale(180)
	waitUntil(t, func() bool { return e.h.frameCount() == 2 }, "a new scale did not render a frame")

	// Several configures while the compositor holds the frame callback: each is
	// rendered at its size, with the buffer resized before the ack and the ack before
	// the commit, so no commit after an ack has the old size.
	sizes := [][2]int32{{500, 300}, {501, 301}, {320, 240}, {800, 600}}
	for i, sz := range sizes {
		before := len(e.srv.RequestsWith("xdg_surface.ack_configure"))
		e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{Width: sz[0], Height: sz[1]})
		waitUntil(t, func() bool { return e.h.frameCount() == 3+i }, "a configure did not render a frame")
		pw, ph := (sz[0]*3+1)/2, (sz[1]*3+1)/2
		e.waitMetrics(platform.ViewMetrics{PhysicalSize: ui.Size{Width: float64(pw), Height: float64(ph)}, DevicePixelRatio: 1.5})

		resizes := e.resizeCalls()
		rc := resizes[len(resizes)-1]
		e.h.mu.Lock()
		framed := e.h.configures[len(e.h.configures)-1]
		e.h.mu.Unlock()
		// The buffer was resized before the configure was acknowledged, and the frame
		// came after the acknowledgement.
		if rc.width != pw || rc.height != ph || framed != rc.configures+1 {
			t.Errorf("configure %v: resize %+v, frame after %d acknowledged configures", sz, rc, framed)
		}
		wantDest := "wp_viewport.set_destination " + itoa(sz[0]) + "x" + itoa(sz[1])
		waitUntil(t, func() bool {
			lines := e.srv.Requests()
			return len(e.srv.RequestsWith("xdg_surface.ack_configure")) == before+1 && index(lines, index(lines, 0, wantDest), "wl_surface.commit") >= 0
		}, "the ack, the viewport and the commit never all arrived")
		lines := e.srv.Requests()
		dest := index(lines, 0, wantDest)
		var ack int
		for j, l := range lines {
			if strings.HasPrefix(l, "xdg_surface.ack_configure") {
				ack = j
			}
		}
		commit := index(lines, ack, "wl_surface.commit")
		if dest < 0 || dest > ack || commit < ack {
			t.Errorf("configure %v: viewport %d / ack %d / commit %d in\n%s", sz, dest, ack, commit, strings.Join(lines[max(ack-4, 0):], "\n"))
		}
	}
	// One frame callback in flight the whole time.
	if n := len(e.srv.RequestsWith("wl_surface.frame")); n != 1 {
		t.Errorf("%d frame callbacks, want 1", n)
	}
}

func itoa(v int32) string { return strconv.Itoa(int(v)) }

func TestIntegerScaleFallbacks(t *testing.T) {
	logical := ui.Size{Width: 640, Height: 400}
	scaled := func(n float64) platform.ViewMetrics {
		return platform.ViewMetrics{PhysicalSize: ui.Size{Width: logical.Width * n, Height: logical.Height * n}, DevicePixelRatio: n}
	}
	t.Run("the surface's preferred buffer scale", func(t *testing.T) {
		e := startEGLWindowWith(t, fakecomp.Config{PreferredBufferScale: 2})
		waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "no frame")
		e.waitMetrics(scaled(2))
		// The buffer scale is set before the first commit that has a buffer.
		waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_surface.set_buffer_scale 2")) == 1 }, "no set_buffer_scale")
		lines := e.srv.Requests()
		if set, commit := index(lines, 0, "wl_surface.set_buffer_scale 2"), index(lines, index(lines, 0, "xdg_surface.ack_configure"), "wl_surface.commit"); set < 0 || commit < set {
			t.Errorf("set_buffer_scale %d, commit after the ack %d in\n%s", set, commit, strings.Join(lines, "\n"))
		}
		if strings.Contains(strings.Join(lines, "\n"), "wp_viewport") {
			t.Error("an integer scale used a viewport")
		}
		e.srv.SendPreferredBufferScale(3)
		e.waitMetrics(scaled(3))
		waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_surface.set_buffer_scale 3")) == 1 }, "no set_buffer_scale 3")
		if r := e.resizeCalls(); r[len(r)-1].width != 1920 || r[len(r)-1].height != 1200 {
			t.Errorf("wl_egl_window_resize = %+v", r[len(r)-1])
		}
	})
	t.Run("the scale of the outputs the surface is on", func(t *testing.T) {
		e := startEGLWindowWith(t, fakecomp.Config{PreferredBufferScale: -1, OutputScale: 2})
		waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "no frame")
		e.waitMetrics(scaled(2))
		e.srv.SetOutputScale(3)
		e.waitMetrics(scaled(3))
		e.srv.LeaveOutput()
		e.waitMetrics(scaled(1))
		e.srv.EnterOutput()
		e.waitMetrics(scaled(3))
	})
	t.Run("a fractional scale wins over an integer one", func(t *testing.T) {
		e := startEGLWindowWith(t, fakecomp.Config{Globals: scaleGlobals(), PreferredBufferScale: 2})
		waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "no frame")
		e.waitMetrics(scaled(2))
		e.srv.SendPreferredScale(180)
		e.waitMetrics(scaled(1.5))
		// With the viewport in use the buffer scale goes back to 1.
		waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_surface.set_buffer_scale 1")) == 1 }, "the buffer scale was not reset")
		// The integer scale changing now does not matter.
		e.srv.SendPreferredBufferScale(4)
		time.Sleep(50 * time.Millisecond)
		if got := e.lastMetrics(); got != scaled(1.5) {
			t.Errorf("metrics = %+v", got)
		}
	})
}

func TestSuspendedWindowRendersNothingAndResumesAtOnce(t *testing.T) {
	e := startEGLWindowWith(t, fakecomp.Config{HoldFrames: true})
	waitUntil(t, func() bool { return e.h.frameCount() == 1 }, "no frame")
	// The first frame's callback is in flight, and the compositor hides the window:
	// that callback may never come.
	e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{States: []uint32{suspendedState}})
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("xdg_surface.ack_configure")) == 2 }, "a suspending configure is acknowledged")
	e.onLoop(func() {
		e.win.RequestFrame()
		e.win.RequestFrame()
	})
	e.srv.FrameDone() // a callback that comes while the window is suspended renders nothing either
	time.Sleep(60 * time.Millisecond)
	if n := e.h.frameCount(); n != 1 {
		t.Fatalf("a suspended window rendered %d frames, want 1 (the one before)", n)
	}
	var st WindowStates
	e.onLoop(func() { st = e.win.(*window).States() })
	if !st.Suspended {
		t.Errorf("states = %+v", st)
	}
	e.h.mu.Lock()
	before := e.h.frames[0]
	e.h.mu.Unlock()

	// Resumed: a frame is drawn at once, with the time of now, though the frame
	// callback that was in flight never came.
	e.srv.ConfigureToplevel(fakecomp.ToplevelConfigure{States: []uint32{4}})
	waitUntil(t, func() bool { return e.h.frameCount() == 2 }, "the window did not draw when it resumed")
	e.h.mu.Lock()
	resumed := e.h.frames[1]
	e.h.mu.Unlock()
	if resumed <= before {
		t.Errorf("the frame time went from %v to %v", before, resumed)
	}
	// Frames go on as usual, one callback at a time, and the late callback of the
	// frame before the suspension is not mistaken for the new one's.
	e.onLoop(func() { e.win.RequestFrame() })
	time.Sleep(50 * time.Millisecond)
	if n := e.h.frameCount(); n != 2 {
		t.Errorf("%d frames: the next waits for the compositor's callback", n)
	}
	e.srv.FrameDone()
	waitUntil(t, func() bool { return e.h.frameCount() == 3 }, "the next frame never came after the callback")
}
