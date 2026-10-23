//go:build linux && !android && glutter_display

package x11

import (
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
)

// Xvfb has no window manager, so the tests that need the states of the window, the
// moves and the focus bring their own: testWM is a small EWMH window manager on a
// connection of its own. It redirects the map and configure requests of the root,
// keeps _NET_WM_STATE and WM_STATE, resizes for maximized and fullscreen windows,
// iconifies, focuses, and records the moves, the resizes and the menus it is asked
// for. All of its X calls are made by one goroutine.

const (
	mapRequest       = 20
	configureRequest = 23
	cwWidth          = 1 << 2
	cwHeight         = 1 << 3
)

// xMapRequestEvent is XMapRequestEvent; only the window is read.
type xMapRequestEvent struct {
	_      [40]byte // type, serial, send_event, display, parent
	window uint64
}

// xConfigureRequestEvent is XConfigureRequestEvent; only the window, the size and
// the value mask are read.
type xConfigureRequestEvent struct {
	_         [40]byte // type, serial, send_event, display, parent
	window    uint64
	_         [8]byte // x, y
	width     int32
	height    int32
	_         [24]byte // border_width, above, detail
	valueMask uint64
}

// The sizes the test window manager gives maximized and fullscreen windows, and the
// size of a window that was never resized.
const (
	wmMaximizedW, wmMaximizedH   = 1200, 760
	wmFullscreenW, wmFullscreenH = 1280, 800
)

type testWM struct {
	t       *testing.T
	x       *xlib
	display uintptr
	root    uint64
	cmds    chan func()
	stop    chan struct{}
	done    chan struct{}

	unmapWindow func(display uintptr, window uint64) int32
	setFocus    func(display uintptr, window uint64, revert int32, time uint64) int32

	// Used by the goroutine only.
	states map[uint64][]uint64  // _NET_WM_STATE of each window
	normal map[uint64][2]uint32 // the size of a window that is neither maximized nor fullscreen
	// allowed are the _NET_WM_ACTION_ names the windows get; nil means all of them.
	allowed []string
	// withdrawnSet are the windows that their client withdrew: their next map reads
	// the _NET_WM_STATE the client set, as a new window's.
	withdrawnSet map[uint64]bool

	mu          sync.Mutex
	moveResizes [][5]int64
	menus       [][5]int64
	activations int
	withdrawals int
}

func startTestWM(t *testing.T, p *Platform) *testWM {
	t.Helper()
	d := p.x.openDisplay(0)
	if d == 0 {
		t.Fatal("could not open a connection for the window manager")
	}
	lib, err := dl.Open(libX11, "libX11.so")
	if err != nil {
		t.Fatal(err)
	}
	wm := &testWM{
		t: t, x: p.x, display: d, root: p.x.defaultRootWindow(d),
		cmds: make(chan func()), stop: make(chan struct{}), done: make(chan struct{}),
		states: map[uint64][]uint64{}, normal: map[uint64][2]uint32{}, withdrawnSet: map[uint64]bool{},
	}
	for _, b := range []struct {
		fptr   any
		symbol string
	}{{&wm.unmapWindow, "XUnmapWindow"}, {&wm.setFocus, "XSetInputFocus"}} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			t.Fatal(err)
		}
	}
	wm.x.selectInput(d, wm.root, substructureRedirMask|substructureNotifyMask)
	wm.x.sync(d, 0) // another window manager would make this an error
	go wm.run()
	t.Cleanup(func() {
		close(wm.stop)
		<-wm.done
		wm.x.closeDisplay(d)
	})
	return wm
}

// do runs fn on the goroutine of the window manager and waits for it.
func (wm *testWM) do(fn func()) {
	finished := make(chan struct{})
	select {
	case wm.cmds <- func() { fn(); close(finished) }:
		<-finished
	case <-wm.done:
	}
}

func (wm *testWM) atom(name string) uint64 {
	c := dl.CString(name)
	defer dl.Free(c)
	return wm.x.internAtom(wm.display, c, 0)
}

func (wm *testWM) setLongs(window, property, typ uint64, values ...uint64) {
	mem := dl.Calloc(uintptr(max(len(values), 1)), 8)
	defer dl.Free(mem)
	for i, v := range values {
		*view[uint64](mem + uintptr(i)*8) = v
	}
	wm.x.changeProperty(wm.display, window, property, typ, 32, propModeReplace, mem, int32(len(values)))
}

func (wm *testWM) run() {
	defer close(wm.done)
	ev := dl.Calloc(1, xEventSize)
	defer dl.Free(ev)
	for {
		select {
		case <-wm.stop:
			return
		case fn := <-wm.cmds:
			fn()
		default:
		}
		for wm.x.pending(wm.display) > 0 {
			wm.x.nextEvent(wm.display, ev)
			wm.handle(ev)
		}
		wm.x.flush(wm.display)
		time.Sleep(time.Millisecond)
	}
}

func (wm *testWM) handle(ev uintptr) {
	switch view[xAnyEvent](ev).typ {
	case mapRequest:
		r := view[xMapRequestEvent](ev)
		if wm.withdrawnSet[r.window] {
			// A withdrawn window comes back as a new one: its state is what the client
			// set before it asked for the map (EWMH).
			delete(wm.withdrawnSet, r.window)
			wm.states[r.window] = wm.readLongs(r.window, wm.atom("_NET_WM_STATE"))
			if wm.bigState(r.window) {
				wm.x.resizeWindow(wm.display, r.window, wmMaximizedW, wmMaximizedH)
			}
		}
		wm.setLongs(r.window, wm.atom("WM_STATE"), wm.atom("WM_STATE"), 1, 0)
		wm.states[r.window] = removeAtom(wm.states[r.window], wm.atom("_NET_WM_STATE_HIDDEN"))
		wm.publishState(r.window)
		wm.publishActions(r.window)
		wm.x.mapWindow(wm.display, r.window)
	case configureRequest:
		r := view[xConfigureRequestEvent](ev)
		if r.valueMask&(cwWidth|cwHeight) == cwWidth|cwHeight {
			wm.x.resizeWindow(wm.display, r.window, uint32(r.width), uint32(r.height))
			if !wm.bigState(r.window) {
				wm.normal[r.window] = [2]uint32{uint32(r.width), uint32(r.height)}
			}
		}
	case clientMessage:
		wm.onMessage(view[xClientMessageEvent](ev))
	case unmapNotify:
		// The synthetic UnmapNotify to the root is how a client withdraws a window
		// (ICCCM 4.1.4): the manager forgets its state.
		if u := view[xMapEvent](ev); u.sendEvent != 0 && u.event == wm.root {
			wm.withdrawnSet[u.window] = true
			delete(wm.states, u.window)
			wm.x.deleteProperty(wm.display, u.window, wm.atom("WM_STATE"))
			wm.x.deleteProperty(wm.display, u.window, wm.atom("_NET_WM_STATE"))
			wm.mu.Lock()
			wm.withdrawals++
			wm.mu.Unlock()
		}
	}
}

// readLongs reads a property of format 32 of a window on the connection of the
// manager.
func (wm *testWM) readLongs(window, property uint64) []uint64 {
	mem := dl.Calloc(1, 8+4+8+8+8)
	defer dl.Free(mem)
	actualType, actualFormat, nitems, bytesAfter, prop := mem, mem+8, mem+16, mem+24, mem+32
	status := wm.x.getWindowProperty(wm.display, window, property, 0, 64, 0, anyPropertyType, actualType, actualFormat, nitems, bytesAfter, prop)
	data := *view[uintptr](prop)
	if data != 0 {
		defer wm.x.free(data)
	}
	if status != 0 || data == 0 || *view[int32](actualFormat) != 32 {
		return nil
	}
	out := make([]uint64, *view[uint64](nitems))
	for i := range out {
		out[i] = *view[uint64](data + uintptr(i)*8)
	}
	return out
}

// bigState is whether the window is maximized or fullscreen.
func (wm *testWM) bigState(window uint64) bool {
	s := wm.states[window]
	return containsAtom(s, wm.atom("_NET_WM_STATE_FULLSCREEN")) ||
		(containsAtom(s, wm.atom("_NET_WM_STATE_MAXIMIZED_VERT")) && containsAtom(s, wm.atom("_NET_WM_STATE_MAXIMIZED_HORZ")))
}

func removeAtom(list []uint64, atom uint64) []uint64 {
	var out []uint64
	for _, a := range list {
		if a != atom {
			out = append(out, a)
		}
	}
	return out
}

func (wm *testWM) publishState(window uint64) {
	wm.setLongs(window, wm.atom("_NET_WM_STATE"), xaAtom, wm.states[window]...)
}

func (wm *testWM) publishActions(window uint64) {
	names := wm.allowed
	if names == nil {
		names = []string{"MAXIMIZE_HORZ", "MAXIMIZE_VERT", "FULLSCREEN", "MINIMIZE", "MOVE", "RESIZE", "CLOSE"}
	}
	var atoms []uint64
	for _, n := range names {
		atoms = append(atoms, wm.atom("_NET_WM_ACTION_"+n))
	}
	wm.setLongs(window, wm.atom("_NET_WM_ALLOWED_ACTIONS"), xaAtom, atoms...)
}

func (wm *testWM) onMessage(m *xClientMessageEvent) {
	switch m.messageType {
	case wm.atom("_NET_WM_STATE"):
		before := wm.bigState(m.window)
		for _, a := range []uint64{uint64(m.data[1]), uint64(m.data[2])} {
			if a == 0 {
				continue
			}
			has := containsAtom(wm.states[m.window], a)
			switch {
			case m.data[0] == 1 && !has, m.data[0] == 2 && !has:
				wm.states[m.window] = append(wm.states[m.window], a)
			case m.data[0] == 0 && has, m.data[0] == 2 && has:
				wm.states[m.window] = removeAtom(wm.states[m.window], a)
			}
		}
		wm.publishState(m.window)
		after := wm.bigState(m.window)
		switch {
		case after && containsAtom(wm.states[m.window], wm.atom("_NET_WM_STATE_FULLSCREEN")):
			wm.x.resizeWindow(wm.display, m.window, wmFullscreenW, wmFullscreenH)
		case after:
			wm.x.resizeWindow(wm.display, m.window, wmMaximizedW, wmMaximizedH)
		case before:
			n, ok := wm.normal[m.window]
			if !ok {
				n = [2]uint32{400, 300} // the size of windowConfig, which the manager has not seen change
			}
			wm.x.resizeWindow(wm.display, m.window, n[0], n[1])
		}
	case wm.atom("WM_CHANGE_STATE"):
		if m.data[0] == iconicState {
			wm.setLongs(m.window, wm.atom("WM_STATE"), wm.atom("WM_STATE"), iconicState, 0)
			wm.unmapWindow(wm.display, m.window)
		}
	case wm.atom("_NET_ACTIVE_WINDOW"):
		wm.mu.Lock()
		wm.activations++
		wm.mu.Unlock()
		wm.setFocus(wm.display, m.window, 2 /* RevertToParent */, 0)
	case wm.atom("_NET_WM_MOVERESIZE"):
		wm.mu.Lock()
		wm.moveResizes = append(wm.moveResizes, m.data)
		wm.mu.Unlock()
	case wm.atom("_GTK_SHOW_WINDOW_MENU"):
		wm.mu.Lock()
		wm.menus = append(wm.menus, m.data)
		wm.mu.Unlock()
	}
}

// focus gives the window the keyboard focus, as a window manager does when the user
// clicks it.
func (wm *testWM) focus(window uint64) {
	wm.do(func() { wm.setFocus(wm.display, window, 2, 0) })
}

// unfocus takes the focus away from every window.
func (wm *testWM) unfocus() {
	wm.do(func() { wm.setFocus(wm.display, wm.root, 2, 0) })
}

func (wm *testWM) moveResizeLog() [][5]int64 {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	return append([][5]int64(nil), wm.moveResizes...)
}

func (wm *testWM) menuLog() [][5]int64 {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	return append([][5]int64(nil), wm.menus...)
}

// startWithWM starts a window with the window manager already running.
func startWithWM(t *testing.T, cfg platform.WindowConfig, setup ...func(*testWM)) (*x11Env, *testWM) {
	t.Helper()
	var wm *testWM
	e := startDisplayWith(t, cfg, func(p *Platform) {
		wm = startTestWM(t, p)
		wm.normal = map[uint64][2]uint32{}
		for _, f := range setup {
			f(wm)
		}
	})
	e.waitMapped()
	return e, wm
}

// lastInfo is the window info of the last metrics the handler heard.
func (h *displayHandler) lastInfo() ui.WindowInfo {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.metrics) == 0 {
		return ui.WindowInfo{}
	}
	return h.metrics[len(h.metrics)-1].Window
}

func (h *displayHandler) lastSize() ui.Size {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.metrics) == 0 {
		return ui.Size{}
	}
	return h.metrics[len(h.metrics)-1].PhysicalSize
}

func TestDisplayWindowStatesFollowTheWindowManager(t *testing.T) {
	e, _ := startWithWM(t, windowConfig())
	if info := e.h.lastInfo(); info.Maximized || info.Fullscreen || info.Minimized || info.Decoration != ui.DecorationServerSide {
		t.Fatalf("a new window: %+v", info)
	}
	steps := []struct {
		name  string
		state platform.WindowState
		check func(i ui.WindowInfo, s ui.Size) bool
	}{
		{"maximize", platform.WindowStateMaximized, func(i ui.WindowInfo, s ui.Size) bool {
			return i.Maximized && !i.Fullscreen && s == ui.Size{Width: wmMaximizedW, Height: wmMaximizedH}
		}},
		{"fullscreen", platform.WindowStateFullscreen, func(i ui.WindowInfo, s ui.Size) bool {
			return i.Maximized && i.Fullscreen && s == ui.Size{Width: wmFullscreenW, Height: wmFullscreenH}
		}},
		{"back to normal", platform.WindowStateNormal, func(i ui.WindowInfo, s ui.Size) bool {
			return !i.Maximized && !i.Fullscreen && s == ui.Size{Width: 400, Height: 300}
		}},
		{"minimize", platform.WindowStateMinimized, func(i ui.WindowInfo, _ ui.Size) bool { return i.Minimized }},
		{"restore", platform.WindowStateNormal, func(i ui.WindowInfo, _ ui.Size) bool { return !i.Minimized }},
	}
	for _, step := range steps {
		e.onLoop(func() { e.win.SetState(step.state) })
		waitFor(t, func() bool { return step.check(e.h.lastInfo(), e.h.lastSize()) }, "after "+step.name+": "+
			"the window is not as it should be")
	}
	// The actions of a title bar that the app draws do the same.
	for _, kind := range []ui.WindowActionKind{ui.WindowActionMaximize, ui.WindowActionRestore, ui.WindowActionFullscreen, ui.WindowActionUnfullscreen} {
		e.onLoop(func() { e.win.PerformWindowAction(ui.WindowAction{Kind: kind}) })
		want := kind == ui.WindowActionMaximize || kind == ui.WindowActionFullscreen
		waitFor(t, func() bool {
			i := e.h.lastInfo()
			if kind == ui.WindowActionMaximize || kind == ui.WindowActionRestore {
				return i.Maximized == want
			}
			return i.Fullscreen == want
		}, "the action did not change the state")
	}
	if e.p.xErrors != 0 {
		t.Errorf("the X server reported %d errors", e.p.xErrors)
	}
}

func TestDisplayAllowedActionsLimitTheCapabilities(t *testing.T) {
	e, _ := startWithWM(t, windowConfig(), func(wm *testWM) { wm.allowed = []string{"MINIMIZE", "MOVE", "RESIZE", "CLOSE"} })
	waitFor(t, func() bool { return e.h.lastInfo().Capabilities.Minimize }, "the allowed actions were not read")
	if c := e.h.lastInfo().Capabilities; c.Maximize || c.Fullscreen || !c.Minimize || !c.WindowMenu {
		t.Errorf("capabilities = %+v, want only the minimize and the menu", c)
	}
}

func TestDisplayWithoutAListOfAllowedActionsEveryOneIsAllowed(t *testing.T) {
	e := startDisplay(t, windowConfig()) // no window manager
	e.waitMapped()
	if c := e.h.lastInfo().Capabilities; c != ui.AllWindowCapabilities {
		t.Errorf("capabilities without a list = %+v", c)
	}
}

func TestDisplayActivatedFollowsTheFocus(t *testing.T) {
	e, wm := startWithWM(t, windowConfig())
	xid := uint64(e.win.NativeSurface().Window)
	wm.focus(xid)
	waitFor(t, func() bool { return e.h.lastInfo().Activated }, "the window did not become active")
	wm.unfocus()
	waitFor(t, func() bool { return !e.h.lastInfo().Activated }, "the window did not become inactive")
	// Activating the window asks the window manager, which focuses it.
	e.onLoop(func() { e.win.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionActivate}) })
	waitFor(t, func() bool { return e.h.lastInfo().Activated }, "the activation request did not give the focus")
	wm.mu.Lock()
	n := wm.activations
	wm.mu.Unlock()
	if n != 1 {
		t.Errorf("the window manager got %d activation requests, want 1", n)
	}
}

func TestDisplayWindowIconIsNetWMIcon(t *testing.T) {
	pixels := func(size int, r, g, b, a byte) ui.WindowIconImage {
		px := make([]byte, size*size*4)
		for i := 0; i < len(px); i += 4 {
			px[i], px[i+1], px[i+2], px[i+3] = r, g, b, a
		}
		return ui.WindowIconImage{Size: size, RGBA: px}
	}
	cfg := windowConfig()
	cfg.Icon = ui.WindowIcon{Name: "ignored-on-x11", Images: []ui.WindowIconImage{pixels(2, 0x10, 0x20, 0x30, 0xff)}}
	e := startDisplay(t, cfg)
	e.waitMapped()
	xid := uint64(e.win.NativeSurface().Window)
	var got []uint64
	read := func() {
		e.onLoop(func() { got = e.p.readLongs(xid, e.p.atoms.netWMIcon, xaCardinal, 1<<20) })
	}
	read()
	want := []uint64{2, 2, 0xff102030, 0xff102030, 0xff102030, 0xff102030}
	// Xlib sign-extends the 32-bit items it hands out as longs.
	for i := range got {
		got[i] &= 0xffffffff
	}
	if len(got) != len(want) {
		t.Fatalf("_NET_WM_ICON = %x, want %x", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("_NET_WM_ICON = %x, want %x", got, want)
		}
	}

	// A new icon with two pictures, one of them translucent, replaces it.
	var err error
	e.onLoop(func() {
		err = e.win.SetIcon(ui.WindowIcon{Images: []ui.WindowIconImage{pixels(1, 1, 2, 3, 0x80), pixels(2, 0, 0, 0, 0)}})
	})
	if err != nil {
		t.Fatal(err)
	}
	read()
	for i := range got {
		got[i] &= 0xffffffff
	}
	if len(got) != 3+6 || got[0] != 1 || got[1] != 1 || got[2] != 0x80010203 || got[3] != 2 || got[4] != 2 {
		t.Errorf("_NET_WM_ICON after the second icon = %x", got)
	}
	// A wrong picture changes nothing and fails.
	e.onLoop(func() {
		err = e.win.SetIcon(ui.WindowIcon{Images: []ui.WindowIconImage{{Size: 3, RGBA: make([]byte, 5)}}})
	})
	if err == nil {
		t.Error("a wrong picture was accepted")
	}
	// An empty icon removes the property.
	e.onLoop(func() { err = e.win.SetIcon(ui.WindowIcon{}) })
	if err != nil {
		t.Fatal(err)
	}
	read()
	if got != nil {
		t.Errorf("_NET_WM_ICON after the empty icon = %x, want none", got)
	}
	// A window cannot be made with a wrong icon.
	bad := windowConfig()
	bad.Icon = ui.WindowIcon{Images: []ui.WindowIconImage{{Size: 2}}}
	e.onLoop(func() { _, err = e.p.NewWindow(bad) })
	if err == nil {
		t.Error("a window was made with a wrong icon")
	}
}

func TestDisplayDecorationsOfTheApp(t *testing.T) {
	tests := []struct {
		name       string
		cfg        func(platform.WindowConfig) platform.WindowConfig
		wantMotif  bool
		wantDecoro ui.DecorationMode
	}{
		{"decorated by the window manager", func(c platform.WindowConfig) platform.WindowConfig { return c }, false, ui.DecorationServerSide},
		{"drawn by the app", func(c platform.WindowConfig) platform.WindowConfig {
			c.Decorations = platform.DecorationsApp
			return c
		}, true, ui.DecorationApp},
		{"not decorated", func(c platform.WindowConfig) platform.WindowConfig {
			c.Decorated = foundation.Some(false)
			return c
		}, true, ui.DecorationNone},
		{"borderless", func(c platform.WindowConfig) platform.WindowConfig {
			c.Kind = platform.WindowKindBorderless
			return c
		}, true, ui.DecorationNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := startDisplay(t, tt.cfg(windowConfig()))
			e.waitMapped()
			xid := uint64(e.win.NativeSurface().Window)
			var hints []uint64
			e.onLoop(func() { hints = e.p.readLongs(xid, e.p.atoms.motifWMHints, e.p.atoms.motifWMHints, 5) })
			if tt.wantMotif {
				// MWM_HINTS_DECORATIONS set, no decorations.
				if len(hints) != 5 || hints[0] != 2 || hints[2] != 0 {
					t.Errorf("_MOTIF_WM_HINTS = %v, want the decorations flag and none", hints)
				}
			} else if hints != nil {
				t.Errorf("_MOTIF_WM_HINTS = %v on a window the manager decorates", hints)
			}
			if got := e.h.lastInfo().Decoration; got != tt.wantDecoro {
				t.Errorf("decoration = %v, want %v", got, tt.wantDecoro)
			}
		})
	}
}

// pressButton plays the press of a pointer button over the window, as the XInput2
// event does; the position is on the screen.
func pressButton(e *x11Env, button int32, rootX, rootY float64) {
	e.onLoop(func() {
		in := e.p.input
		w := e.p.windows[uint64(e.win.NativeSurface().Window)]
		in.pointerWindow = w
		in.buttonsDown |= 1 << button // what the dispatcher of the XInput2 events keeps
		in.onButton(w, &xiDeviceEvent{detail: button, deviceID: 2, rootX: rootX, rootY: rootY, eventX: 10, eventY: 10}, true)
	})
}

func TestDisplayMoveResizeAndMenuGoToTheWindowManager(t *testing.T) {
	cfg := windowConfig()
	cfg.Decorations = platform.DecorationsApp
	e, wm := startWithWM(t, cfg)

	// Without a button that is down there is nothing to move with.
	e.onLoop(func() { e.win.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionMove}) })
	e.onLoop(func() {
		e.win.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionShowMenu, Position: ui.Offset{Dx: 5, Dy: 5}})
	})
	wm.do(func() {})
	if n := len(wm.moveResizeLog()) + len(wm.menuLog()); n != 0 {
		t.Fatalf("%d requests went to the window manager without a button", n)
	}

	pressButton(e, 1, 120, 140)
	tests := []struct {
		edge ui.ResizeEdge
		dir  int64
	}{
		{ui.ResizeEdgeTopLeft, 0}, {ui.ResizeEdgeTop, 1}, {ui.ResizeEdgeTopRight, 2}, {ui.ResizeEdgeRight, 3},
		{ui.ResizeEdgeBottomRight, 4}, {ui.ResizeEdgeBottom, 5}, {ui.ResizeEdgeBottomLeft, 6}, {ui.ResizeEdgeLeft, 7},
	}
	var want [][5]int64
	e.onLoop(func() { e.win.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionMove}) })
	want = append(want, [5]int64{120, 140, 8, 1, 1})
	for _, tt := range tests {
		e.onLoop(func() { e.win.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionResize, Edge: tt.edge}) })
		want = append(want, [5]int64{120, 140, tt.dir, 1, 1})
	}
	waitFor(t, func() bool { return len(wm.moveResizeLog()) == len(want) }, "the moves and resizes did not reach the window manager")
	for i, got := range wm.moveResizeLog() {
		if got != want[i] {
			t.Errorf("request %d = %v, want %v", i, got, want[i])
		}
	}

	// The menu opens where the title bar says, in the coordinates of the screen.
	e.onLoop(func() {
		e.win.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionShowMenu, Position: ui.Offset{Dx: 30, Dy: 12}})
	})
	waitFor(t, func() bool { return len(wm.menuLog()) == 1 }, "the menu request did not reach the window manager")
	var ox, oy int
	e.onLoop(func() { ox, oy = e.p.windows[uint64(e.win.NativeSurface().Window)].rootOrigin() })
	if got := wm.menuLog()[0]; got[0] != 2 || got[1] != int64(ox)+30 || got[2] != int64(oy)+12 {
		t.Errorf("menu request = %v, want device 2 at %d,%d", got, ox+30, oy+12)
	}

	// Once the button is released there is nothing to move with.
	e.onLoop(func() {
		in := e.p.input
		in.buttonsDown &^= 1 << 1
		in.onButton(e.p.windows[uint64(e.win.NativeSurface().Window)], &xiDeviceEvent{detail: 1, deviceID: 2}, false)
		e.win.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionMove})
	})
	wm.do(func() {})
	time.Sleep(50 * time.Millisecond)
	if n := len(wm.moveResizeLog()); n != len(want) {
		t.Errorf("a move after the release reached the window manager (%d requests)", n)
	}
	if e.p.xErrors != 0 {
		t.Errorf("the X server reported %d errors", e.p.xErrors)
	}
}

func TestDisplayTheServerTakesTouchAndTheWindowSelectsIt(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	var has bool
	var errs int
	e.onLoop(func() {
		has = e.p.input != nil && e.p.input.hasTouch
		e.p.x.sync(e.p.display, 0)
		errs = e.p.xErrors
	})
	if !has {
		t.Skip("the X server has no XInput 2.2: no touch")
	}
	if errs != 0 {
		t.Errorf("selecting touch events made %d X errors", errs)
	}
}

func TestDisplayHidingWithdrawsTheWindowAndShowingMapsItAgain(t *testing.T) {
	e, wm := startWithWM(t, windowConfig())
	w := e.win.(*window)
	e.onLoop(func() { e.win.SetState(platform.WindowStateMaximized) })
	waitFor(t, func() bool { return e.h.lastInfo().Maximized }, "the window was not maximized")

	e.onLoop(func() {
		if !e.win.SetHidden(true) {
			t.Error("SetHidden says X11 cannot hide windows")
		}
	})
	waitFor(t, func() bool { wm.mu.Lock(); defer wm.mu.Unlock(); return wm.withdrawals == 1 }, "the window manager was not told of the withdrawal")
	_, framesBefore, _, _, _ := e.h.snapshot()
	e.onLoop(func() {
		e.win.RequestFrame()                          // waits for the map
		e.win.SetState(platform.WindowStateMinimized) // would map it iconic
		e.win.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionActivate})
	})
	time.Sleep(100 * time.Millisecond)
	var mapped bool
	var minimized bool
	e.onLoop(func() { mapped, minimized = w.mapped, w.info().Minimized })
	if _, frames, _, _, _ := e.h.snapshot(); frames != framesBefore {
		t.Errorf("%d frames while hidden", frames-framesBefore)
	}
	if mapped || minimized {
		t.Errorf("mapped %v, minimized %v: a withdrawn window is neither", mapped, minimized)
	}
	select {
	case err := <-e.done:
		t.Fatalf("Run returned (%v) while a hidden window exists", err)
	default:
	}

	e.onLoop(func() { e.win.SetHidden(false) })
	waitFor(t, func() bool { _, frames, _, _, _ := e.h.snapshot(); return frames > framesBefore }, "the shown window was not drawn")
	waitFor(t, func() bool { return e.h.lastInfo().Maximized && !e.h.lastInfo().Minimized }, "the window did not come back maximized")
	waitFor(t, func() bool { return e.h.lastSize() == ui.Size{Width: wmMaximizedW, Height: wmMaximizedH} }, "the window did not come back at its size")
	wm.mu.Lock()
	activations := wm.activations
	wm.mu.Unlock()
	if activations != 1 {
		t.Errorf("%d activation requests, want the one of the show", activations)
	}
	if e.p.xErrors != 0 {
		t.Errorf("the X server reported %d errors", e.p.xErrors)
	}

	e.onLoop(func() { e.win.Close() })
	select {
	case err := <-e.done:
		if err != nil {
			t.Fatalf("Run returned %v after the last window closed", err)
		}
		e.done <- nil // for the cleanup
	case <-time.After(displayTimeout):
		t.Fatal("Run did not return after the hidden window closed")
	}
}
