package wl

import (
	"io"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
)

// These tests run the real libwayland-client against the fake compositor over a
// socket pair, so they need libwayland but no display. They skip when the
// library is not installed.

func skipWithoutLibwayland(t *testing.T) {
	t.Helper()
	if _, err := loadClient(); err != nil {
		t.Skipf("libwayland-client is not available: %v", err)
	}
}

// session is a connection to a fake compositor with the registry listed.
type session struct {
	t       *testing.T
	conn    *Conn
	srv     *fakecomp.Server
	reg     *WlRegistry
	globals map[string]global
}

type global struct{ name, version uint32 }

func newSession(t *testing.T) *session {
	t.Helper()
	skipWithoutLibwayland(t)
	srv, clientFD := fakecomp.New(t, fakecomp.Config{})
	conn, err := ConnectFD(clientFD)
	if err != nil {
		t.Fatalf("ConnectFD: %v", err)
	}
	s := &session{t: t, conn: conn, srv: srv, globals: map[string]global{}}
	t.Cleanup(func() {
		conn.Close()
		srv.Wait()
	})
	s.reg = conn.Display().GetRegistry()
	s.reg.SetEvents(&WlRegistryEvents{OnGlobal: func(name uint32, iface string, version uint32) {
		s.globals[iface] = global{name, version}
	}})
	s.roundtrip()
	return s
}

func (s *session) roundtrip() {
	s.t.Helper()
	if err := s.conn.Roundtrip(); err != nil {
		s.t.Fatalf("Roundtrip: %v", err)
	}
}

// g returns the advertised global of an interface.
func (s *session) g(iface string) global {
	s.t.Helper()
	g, ok := s.globals[iface]
	if !ok {
		s.t.Fatalf("the compositor does not advertise %s", iface)
	}
	return g
}

func (s *session) compositor() *WlCompositor {
	g := s.g("wl_compositor")
	return BindWlCompositor(s.reg, g.name, g.version)
}

func (s *session) output() *WlOutput {
	g := s.g("wl_output")
	return BindWlOutput(s.reg, g.name, g.version)
}

func (s *session) seat() *WlSeat {
	g := s.g("wl_seat")
	return BindWlSeat(s.reg, g.name, g.version)
}

func TestConnectFailures(t *testing.T) {
	skipWithoutLibwayland(t)
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv("WAYLAND_DISPLAY", "")

	_, err := Connect("glutter-no-such-display")
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != "WaylandConnectFailed" || !appErr.HasRoute("Connect.WaylandConnectFailed") {
		t.Errorf("Tag = %q, route = %s, want WaylandConnectFailed via Connect", appErr.Tag, appErr.Route())
	}
	if appErr.Metadata["display"] != "glutter-no-such-display" {
		t.Errorf("Metadata[display] = %v", appErr.Metadata["display"])
	}
}

func TestRegistryAndBind(t *testing.T) {
	s := newSession(t)

	want := map[string]global{
		"wl_compositor": {1, 6}, "wl_output": {2, 4}, "wl_seat": {3, 7},
		"xdg_wm_base": {4, 6}, "wl_data_device_manager": {5, 3}, "wl_shm": {6, 1},
		"wl_subcompositor": {7, 1},
	}
	if !reflect.DeepEqual(s.globals, want) {
		t.Errorf("globals = %v, want %v", s.globals, want)
	}

	compositor := s.compositor()
	if compositor == nil {
		t.Fatal("BindWlCompositor returned nil")
	}
	s.roundtrip()
	if got := s.srv.Joined(); !strings.Contains(got, "wl_registry.bind name=1 interface=wl_compositor version=6") {
		t.Errorf("the compositor did not receive a bind with name, interface and version:\n%s", got)
	}
	if compositor.ProxyVersion() != 6 || compositor.ProxyID() == 0 {
		t.Errorf("version = %d, id = %d, want 6 and non-zero", compositor.ProxyVersion(), compositor.ProxyID())
	}
	if compositor.Conn() != s.conn {
		t.Error("Conn() is not the connection of the object")
	}
}

func TestDisplaySyncCallback(t *testing.T) {
	s := newSession(t)
	var data []uint32
	cb := s.conn.Display().Sync()
	cb.SetEvents(&WlCallbackEvents{OnDone: func(callbackData uint32) { data = append(data, callbackData) }})
	s.roundtrip()
	if !reflect.DeepEqual(data, []uint32{7}) {
		t.Errorf("callback data = %v, want [7]", data)
	}
}

func TestSurfaceRequestsAndEvents(t *testing.T) {
	s := newSession(t)
	output := s.output()
	type geometry struct {
		x, y, w, h int32
		subpixel   WlOutputSubpixel
		make, mdl  string
		transform  WlOutputTransform
	}
	var geo []geometry
	var scale []int32
	output.SetEvents(&WlOutputEvents{
		OnGeometry: func(x, y, physicalWidth, physicalHeight int32, subpixel WlOutputSubpixel, make, model string, transform WlOutputTransform) {
			geo = append(geo, geometry{x, y, physicalWidth, physicalHeight, subpixel, make, model, transform})
		},
		OnScale: func(factor int32) { scale = append(scale, factor) },
	})
	s.roundtrip()
	if want := []geometry{{-10, 20, 300, 200, WlOutputSubpixelNone, "Glutter", "Test Panel", WlOutputTransformNormal}}; !reflect.DeepEqual(geo, want) {
		t.Errorf("geometry = %+v, want %+v", geo, want)
	}
	if !reflect.DeepEqual(scale, []int32{2}) {
		t.Errorf("scale = %v, want [2]", scale)
	}

	compositor := s.compositor()
	var entered []*WlOutput
	var preferred []int32
	surface := compositor.CreateSurface()
	surface.SetEvents(&WlSurfaceEvents{
		OnEnter:                func(o *WlOutput) { entered = append(entered, o) },
		OnPreferredBufferScale: func(f int32) { preferred = append(preferred, f) },
	})
	surface.Attach(nil, 10, 20)
	frame := surface.Frame()
	surface.Commit()
	s.roundtrip()

	if len(entered) != 1 || entered[0] != output {
		t.Errorf("wl_surface.enter delivered %v, want the very output object that was bound (%p)", entered, output)
	}
	if !reflect.DeepEqual(preferred, []int32{2}) {
		t.Errorf("preferred_buffer_scale = %v, want [2]", preferred)
	}
	if frame == nil {
		t.Error("Frame returned nil")
	}
	want := []string{
		"wl_surface.attach buffer=0 x=10 y=20", // a nil buffer is NULL on the wire
		"wl_surface.frame",
		"wl_surface.commit",
	}
	got := s.srv.Requests()
	if len(got) < 3 || !reflect.DeepEqual(got[len(got)-3:], want) {
		t.Errorf("requests = %q, want to end with %q", got, want)
	}
	if surface.Ptr() == 0 {
		t.Error("Ptr() is 0 for a live object")
	}
	if (*WlSurface)(nil).Ptr() != 0 {
		t.Error("Ptr() of a nil object must be 0")
	}
}

func TestPointerEventsWithFixedAndObjects(t *testing.T) {
	s := newSession(t)
	surface := s.compositor().CreateSurface()
	pointer := s.seat().GetPointer()

	type enter struct {
		serial  uint32
		surface *WlSurface
		x, y    float64
	}
	type motion struct {
		time uint32
		x, y float64
	}
	var enters []enter
	var motions []motion
	pointer.SetEvents(&WlPointerEvents{
		OnEnter:  func(serial uint32, surf *WlSurface, x, y float64) { enters = append(enters, enter{serial, surf, x, y}) },
		OnMotion: func(time uint32, x, y float64) { motions = append(motions, motion{time, x, y}) },
	})
	s.roundtrip()

	if want := []enter{{11, surface, 10.5, -3.25}}; !reflect.DeepEqual(enters, want) {
		t.Errorf("enter = %+v, want %+v", enters, want)
	}
	if want := []motion{{1234, 0.125, 99}}; !reflect.DeepEqual(motions, want) {
		t.Errorf("motion = %+v, want %+v", motions, want)
	}
}

func TestKeyboardFileDescriptorAndArrayEvents(t *testing.T) {
	s := newSession(t)
	surface := s.compositor().CreateSurface()
	keyboard := s.seat().GetKeyboard()

	var keymap string
	var size uint32
	var format WlKeyboardKeymapFormat
	var keys []uint32
	var enterSurface *WlSurface
	keyboard.SetEvents(&WlKeyboardEvents{
		OnKeymap: func(f WlKeyboardKeymapFormat, fd int, sz uint32) {
			file := os.NewFile(uintptr(fd), "keymap")
			data, err := io.ReadAll(file)
			if err != nil {
				t.Errorf("reading the keymap fd: %v", err)
			}
			if err := file.Close(); err != nil {
				t.Errorf("closing the keymap fd: %v", err)
			}
			keymap, size, format = string(data), sz, f
		},
		OnEnter: func(serial uint32, surf *WlSurface, k []uint32) { enterSurface, keys = surf, k },
	})
	s.roundtrip()

	if keymap != fakecomp.KeymapText || size != uint32(len(fakecomp.KeymapText)) || format != WlKeyboardKeymapFormatXkbV1 {
		t.Errorf("keymap = %q (format %d, size %d), want %q", keymap, format, size, fakecomp.KeymapText)
	}
	if !reflect.DeepEqual(keys, []uint32{30, 31}) {
		t.Errorf("keys = %v, want [30 31]", keys)
	}
	if enterSurface != surface {
		t.Errorf("enter surface = %p, want %p", enterSurface, surface)
	}
}

func TestServerCreatedObjectsInEvents(t *testing.T) {
	s := newSession(t)
	g := s.g("wl_data_device_manager")
	manager := BindWlDataDeviceManager(s.reg, g.name, g.version)
	seat := s.seat()
	device := manager.GetDataDevice(seat)

	var offers []*WlDataOffer
	var mimes []string
	device.SetEvents(&WlDataDeviceEvents{OnDataOffer: func(offer *WlDataOffer) {
		offers = append(offers, offer)
		// The offer's own events follow in the same batch, so the callbacks
		// must be set here, inside the event that created the object.
		offer.SetEvents(&WlDataOfferEvents{OnOffer: func(mime string) { mimes = append(mimes, mime) }})
	}})
	s.roundtrip()

	if len(offers) != 1 {
		t.Fatalf("got %d data offers, want 1", len(offers))
	}
	if want := []string{"text/plain;charset=utf-8", "image/png"}; !reflect.DeepEqual(mimes, want) {
		t.Errorf("offered mime types = %q, want %q", mimes, want)
	}

	// Requests on the server-created object, including a file descriptor and a
	// nullable string.
	var pipe [2]int
	if err := syscall.Pipe2(pipe[:], syscall.O_CLOEXEC); err != nil {
		t.Fatal(err)
	}
	for _, fd := range pipe {
		t.Cleanup(func() {
			if err := syscall.Close(fd); err != nil {
				t.Errorf("closing a pipe end: %v", err)
			}
		})
	}
	offers[0].Receive("image/png", pipe[1])
	offers[0].Accept(5, "")
	s.roundtrip()
	got := s.srv.Requests()
	if n := len(got); n < 2 || got[n-2] != "wl_data_offer.receive mime=image/png fd=true" || got[n-1] != "wl_data_offer.accept serial=5 mime=<null>" {
		t.Errorf("requests = %q, want receive with an fd, then accept with a NULL mime type", got)
	}
}

func TestRequestWithFileDescriptor(t *testing.T) {
	s := newSession(t)
	g := s.g("wl_shm")
	shm := BindWlShm(s.reg, g.name, g.version)

	file, err := os.CreateTemp(t.TempDir(), "pool")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Errorf("closing the pool file: %v", err)
		}
	})
	if err := file.Truncate(4096); err != nil {
		t.Fatal(err)
	}
	pool := shm.CreatePool(int(file.Fd()), 4096)
	if pool == nil {
		t.Fatal("CreatePool returned nil")
	}
	s.roundtrip()
	if got := s.srv.Joined(); !strings.Contains(got, "wl_shm.create_pool size=4096 filesize=4096") {
		t.Errorf("the compositor did not receive the file descriptor of the pool:\n%s", got)
	}
}

func TestXdgToplevel(t *testing.T) {
	s := newSession(t)
	g := s.g("xdg_wm_base")
	wm := BindXdgWmBase(s.reg, g.name, g.version)
	surface := s.compositor().CreateSurface()
	xdgSurface := wm.GetXdgSurface(surface)
	toplevel := xdgSurface.GetToplevel()

	var width, height int32
	var states, capabilities []uint32
	var serial uint32
	toplevel.SetEvents(&XdgToplevelEvents{
		OnConfigure:      func(w, h int32, st []uint32) { width, height, states = w, h, st },
		OnWmCapabilities: func(c []uint32) { capabilities = c },
	})
	xdgSurface.SetEvents(&XdgSurfaceEvents{OnConfigure: func(sr uint32) { serial = sr }})
	s.roundtrip()

	if width != 800 || height != 600 || !reflect.DeepEqual(states, []uint32{uint32(XdgToplevelStateMaximized), uint32(XdgToplevelStateActivated)}) {
		t.Errorf("configure = %dx%d %v, want 800x600 [maximized activated]", width, height, states)
	}
	if !reflect.DeepEqual(capabilities, []uint32{1, 2}) {
		t.Errorf("wm_capabilities = %v, want [1 2]", capabilities)
	}
	if serial != 77 {
		t.Errorf("xdg_surface.configure serial = %d, want 77", serial)
	}

	xdgSurface.AckConfigure(serial)
	toplevel.SetTitle("Glutter dev window")
	toplevel.SetAppId("dev.glutter.test")
	toplevel.SetMinSize(320, 200)
	s.roundtrip()
	want := []string{
		"xdg_wm_base.get_xdg_surface surface=true",
		"xdg_surface.get_toplevel",
		"xdg_surface.ack_configure serial=77",
		`xdg_toplevel.set_title "Glutter dev window"`,
		`xdg_toplevel.set_app_id "dev.glutter.test"`,
		"xdg_toplevel.set_min_size 320x200",
	}
	got := s.srv.Requests()
	if len(got) < len(want) || !reflect.DeepEqual(got[len(got)-len(want):], want) {
		t.Errorf("requests = %q, want to end with %q", got, want)
	}

	before := handleCount()
	toplevel.Destroy()
	s.roundtrip()
	if !strings.HasSuffix(s.srv.Joined(), "xdg_toplevel.destroy") {
		t.Errorf("the destructor request was not sent:\n%s", s.srv.Joined())
	}
	if handleCount() != before-1 {
		t.Errorf("handle count = %d, want %d: a destroyed object must be forgotten", handleCount(), before-1)
	}

	t.Run("a request on a destroyed object panics", func(t *testing.T) {
		defer func() {
			appErr, ok := errors.FromError(asError(recover()))
			if !ok || appErr.Tag != "WaylandObjectDestroyed" {
				t.Errorf("recovered %v, want a WaylandObjectDestroyed error", appErr)
			}
		}()
		toplevel.SetTitle("too late")
		t.Error("no panic")
	})
}

// asError returns v if it is an error, else nil.
func asError(v any) error {
	err, _ := v.(error)
	return err
}

func TestEventHandlerPanicIsRaisedAgainOutsideC(t *testing.T) {
	s := newSession(t)
	output := s.output()
	output.SetEvents(&WlOutputEvents{OnScale: func(int32) { panic("boom") }})

	func() {
		defer func() {
			if got := recover(); got != "boom" {
				t.Errorf("recovered %v, want boom", got)
			}
		}()
		_ = s.conn.Roundtrip()
		t.Error("Roundtrip returned normally after a handler panicked")
	}()

	// The connection still works afterwards.
	output.SetEvents(nil)
	s.roundtrip()
}

func TestProtocolError(t *testing.T) {
	s := newSession(t)
	surface := s.compositor().CreateSurface()
	s.roundtrip()

	// wl_display.error(object_id, code, message)
	s.srv.Send(1, 0, surface.ProxyID(), uint32(2), "bad scale")
	err := s.conn.Roundtrip()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != "WaylandProtocolError" || !appErr.HasRoute("Conn.Roundtrip->Conn.lastErrorOr->Conn.lastError.WaylandProtocolError") {
		t.Errorf("Tag = %q, route = %s, want WaylandProtocolError via Roundtrip", appErr.Tag, appErr.Route())
	}
	if appErr.Metadata["interface"] != "wl_surface" || appErr.Metadata["id"] != surface.ProxyID() || appErr.Metadata["code"] != uint32(2) {
		t.Errorf("Metadata = %v, want interface wl_surface, id %d, code 2", appErr.Metadata, surface.ProxyID())
	}
}

func TestConnectionLost(t *testing.T) {
	s := newSession(t)
	// The compositor hangs up.
	s.srv.Hangup()
	s.srv.Wait()

	err := s.conn.Roundtrip()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != "WaylandConnectionLost" || !appErr.HasRoute("Conn.Roundtrip->Conn.lastErrorOr.WaylandConnectionLost") && !appErr.HasRoute("Conn.Roundtrip->Conn.lastErrorOr->Conn.lastError.WaylandConnectionLost") {
		t.Errorf("Tag = %q, route = %s, want WaylandConnectionLost via Roundtrip", appErr.Tag, appErr.Route())
	}
}

// TestReadLoop uses the sequence of the event loop: flush, prepare_read, wait
// for the socket, read_events, dispatch_pending.
func TestReadLoop(t *testing.T) {
	s := newSession(t)
	output := s.output()
	s.roundtrip() // the bind reaches the compositor
	var scales []int32
	output.SetEvents(&WlOutputEvents{OnScale: func(f int32) { scales = append(scales, f) }})

	// An event the client did not ask for.
	s.srv.Send(s.srv.ID("wl_output"), 3, int32(5))

	deadline := time.Now().Add(5 * time.Second)
	for len(scales) == 0 && time.Now().Before(deadline) {
		for !s.conn.PrepareRead() {
			if _, err := s.conn.DispatchPending(); err != nil {
				t.Fatal(err)
			}
		}
		if blocked, err := s.conn.Flush(); err != nil || blocked {
			s.conn.CancelRead()
			t.Fatalf("Flush = %v, %v", blocked, err)
		}
		var readable syscall.FdSet
		fd := s.conn.FD()
		readable.Bits[fd/64] |= 1 << (uint(fd) % 64)
		timeout := syscall.Timeval{Usec: 200_000}
		n, err := syscall.Select(fd+1, &readable, nil, nil, &timeout)
		if err != nil || n == 0 {
			s.conn.CancelRead()
			continue
		}
		if err := s.conn.ReadEvents(); err != nil {
			t.Fatal(err)
		}
		if _, err := s.conn.DispatchPending(); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(scales, []int32{5}) {
		t.Errorf("scale events = %v, want [5]", scales)
	}
}

func TestCloseForgetsObjects(t *testing.T) {
	skipWithoutLibwayland(t)
	before := handleCount()
	s := newSession(t)
	s.compositor()
	s.output()
	if handleCount() <= before {
		t.Fatalf("handle count = %d, want more than %d while the connection is open", handleCount(), before)
	}
	s.conn.Close()
	if handleCount() != before {
		t.Errorf("handle count = %d after Close, want %d", handleCount(), before)
	}
	s.conn.Close() // closing twice is harmless
}

func TestSingleDispatcherIsShared(t *testing.T) {
	skipWithoutLibwayland(t)
	first := dispatcherPointer()
	count := dl.CallbackCount()
	s := newSession(t)
	s.compositor()
	s.output()
	s.seat().GetPointer()
	if dispatcherPointer() != first {
		t.Error("the dispatcher pointer changed")
	}
	if got := dl.CallbackCount(); got != count {
		t.Errorf("callbacks created = %d more, want none: all proxies share one dispatcher", got-count)
	}
}
