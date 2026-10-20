//go:build glutter_display

package x11

import (
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// testRandR adds and removes monitors on the server from a second connection, which
// is what a plugged in or a removed monitor looks like to the app.
type testRandR struct {
	*second
	setMonitor    func(display uintptr, window uint64, info uintptr)
	deleteMonitor func(display uintptr, window, name uint64)
}

func newTestRandR(t *testing.T, p *Platform) *testRandR {
	t.Helper()
	lib, err := dl.Open(libXrandr, "libXrandr.so")
	if err != nil {
		t.Skipf("libXrandr is not available: %v", err)
	}
	r := &testRandR{second: openSecond(t, p)}
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&r.setMonitor, "XRRSetMonitor"},
		{&r.deleteMonitor, "XRRDeleteMonitor"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			t.Skipf("libXrandr has no %s: %v", b.symbol, err)
		}
	}
	return r
}

// add makes a monitor of the geometry, with a name and no outputs (a monitor that the
// server does not tie to an output is a logical one, which is still a monitor to
// XRRGetMonitors).
func (r *testRandR) add(name string, x, y, w, h int32) uint64 {
	atom := r.atom(name)
	mem := dl.Calloc(1, 56)
	defer dl.Free(mem)
	m := view[xrrMonitorInfo](mem)
	m.name, m.x, m.y, m.width, m.height, m.mmWidth, m.mmHeight = atom, x, y, w, h, w/4, h/4
	r.setMonitor(r.display, r.x.defaultRootWindow(r.display), mem)
	r.x.flush(r.display)
	return atom
}

func (r *testRandR) remove(name uint64) {
	r.deleteMonitor(r.display, r.x.defaultRootWindow(r.display), name)
	r.x.flush(r.display)
}

// notify plays the event that the server sends when the screen changes
// (RRScreenChangeNotify) or when an output or a CRTC does (RRNotify): Xvfb has one
// output, and Xlib cannot send these events with XSendEvent, so the test hands the
// event to the app's event handler on the UI thread. The event carries the size of
// the screen.
func notify(e *x11Env, kind int32) {
	e.onLoop(func() {
		mem := e.p.event
		*view[int32](mem) = e.p.rrEventBase + kind
		// XRRScreenChangeNotifyEvent: type, serial, send_event, display, window, root,
		// timestamp, config_timestamp, size_index, subpixel_order, rotation, width,
		// height, mwidth, mheight.
		*view[uint64](mem + 32) = e.p.root
		*view[uint64](mem + 40) = e.p.root
		*view[int32](mem + 72) = 1280
		*view[int32](mem + 76) = 800
		*view[int32](mem + 80) = 320
		*view[int32](mem + 84) = 200
		e.p.handleEvent()
	})
}

func displayNames(e *x11Env) []string {
	var names []string
	e.onLoop(func() {
		for _, d := range e.p.Displays() {
			names = append(names, d.Name)
		}
	})
	return names
}

func TestDisplayMonitorsComeAndGo(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	before := displayNames(e)
	var base int32
	e.onLoop(func() { base = e.p.rrEventBase })
	if base == 0 {
		t.Skip("the server has no RandR")
	}
	r := newTestRandR(t, e.p)

	// A monitor appears and the screen changes: the app is told, once.
	n0 := e.h.displayChanges()
	mon := r.add("GLUTTER-TEST-2", 1280, 0, 640, 480)
	notify(e, rrScreenChange)
	waitFor(t, func() bool { return e.h.displayChanges() > n0 }, "the new monitor was not noticed")
	after := displayNames(e)
	if len(after) != len(before)+1 || !slices.Contains(after, "GLUTTER-TEST-2") {
		t.Errorf("monitors = %v after plugging one in, were %v", after, before)
	}
	var d platform.Display
	e.onLoop(func() {
		for _, x := range e.p.Displays() {
			if x.Name == "GLUTTER-TEST-2" {
				d = x
			}
		}
	})
	if d.Bounds.Left != 1280 || d.Bounds.Width() != 640 || d.Bounds.Height() != 480 || d.DevicePixelRatio != 1 {
		t.Errorf("the new display = %+v", d)
	}

	// The output and the CRTC change too: the monitors are the same, so nobody is told again.
	n1 := e.h.displayChanges()
	notify(e, rrNotify)
	notify(e, rrScreenChange)
	if got := e.h.displayChanges(); got != n1 {
		t.Errorf("events that changed no monitor made %d more calls", got-n1)
	}

	// The monitor goes away; this time the event is the one for an output.
	r.remove(mon)
	r.x.sync(r.display, 0)
	notify(e, rrNotify)
	waitFor(t, func() bool { return len(displayNames(e)) == len(before) }, "the removed monitor was not noticed")
	if got := e.h.displayChanges(); got <= n1 {
		t.Errorf("the handler was not told that the monitor went away")
	}
}
