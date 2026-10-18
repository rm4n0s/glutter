// Package fakecomp is a tiny fake Wayland compositor for tests: it speaks the
// wire protocol over a socket pair, answers the requests that the tests send,
// and logs them. It is written by hand from the protocol description and is
// independent of the generated bindings, so it can check them. It needs no
// display.
//
// Flutter: no counterpart.
package fakecomp

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
)

// Server is a fake compositor on one end of a socket pair.
//
// It handles only the requests the tests send. Every request it receives is
// appended to the log as text, so tests can assert on what the client sent.
//
// Flutter: no counterpart
type Server struct {
	t       testing.TB
	fd      int
	globals []Global

	mu      sync.Mutex // guards objects, log, ids and writes
	objects map[uint32]string
	log     []string
	ids     map[string]uint32 // the last object ID of each interface

	buf []byte // received, not yet parsed
	fds []int  // received file descriptors, in order

	serverIDs uint32 // next ID for objects the server creates (0xff000000...)
	done      chan struct{}
}

// Fixed, FD and Array mark event arguments for Send that are not plain integers
// or strings. Send also takes uint32, int32 and string.
type (
	// Fixed is a wl_fixed_t argument.
	//
	// Flutter: no counterpart
	Fixed float64
	// FD is a file descriptor, sent as ancillary data.
	//
	// Flutter: no counterpart
	FD uintptr
	// Array is a wl_array argument.
	//
	// Flutter: no counterpart
	Array []byte
)

// Global is one global that the registry announces.
//
// Flutter: no counterpart
type Global struct {
	// Name is the numeric name that identifies the global in bind.
	Name uint32
	// Interface is the interface name, such as "wl_compositor".
	Interface string
	// Version is the advertised version.
	Version uint32
}

// DefaultGlobals returns the globals the registry announces unless Config says
// otherwise.
//
// Flutter: no counterpart
func DefaultGlobals() []Global {
	return []Global{
		{1, "wl_compositor", 6}, {2, "wl_output", 4}, {3, "wl_seat", 7},
		{4, "xdg_wm_base", 6}, {5, "wl_data_device_manager", 3}, {6, "wl_shm", 1},
		{7, "wl_subcompositor", 1},
	}
}

// Config configures a Server.
//
// Flutter: no counterpart
type Config struct {
	// Globals is what the registry announces. Nil means DefaultGlobals; use an
	// empty non-nil slice for none.
	Globals []Global
}

// New starts a server on one end of a new socket pair and returns the other
// end for the client, which takes ownership of it.
//
// Flutter: no counterpart
func New(t testing.TB, cfg Config) (s *Server, clientFD int) {
	t.Helper()
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}
	globals := cfg.Globals
	if globals == nil {
		globals = DefaultGlobals()
	}
	s = &Server{
		t: t, fd: fds[0], globals: globals,
		objects:   map[uint32]string{1: "wl_display"},
		ids:       map[string]uint32{},
		serverIDs: 0xff000000,
		done:      make(chan struct{}),
	}
	go s.serve()
	return s, fds[1]
}

// Wait blocks until the client has closed its end and the server has stopped.
//
// Flutter: no counterpart
func (s *Server) Wait() {
	<-s.done
	_ = syscall.Close(s.fd)
}

// Requests returns a copy of the log of requests, one text line each.
//
// Flutter: no counterpart
func (s *Server) Requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.log...)
}

// ID returns the ID of the object of the interface that the client created or
// bound last, or 0.
//
// Flutter: no counterpart
func (s *Server) ID(iface string) uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ids[iface]
}

func (s *Server) record(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log = append(s.log, fmt.Sprintf(format, args...))
}

func (s *Server) addObject(id uint32, iface string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[id] = iface
	s.ids[iface] = id
}

func (s *Server) newServerID() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.serverIDs
	s.serverIDs++ // server-side IDs must be handed out one after the other, from 0xff000000
	return id
}

// fill reads more bytes, and any file descriptors that came with them. It
// returns false when the client has hung up.
func (s *Server) fill() bool {
	data := make([]byte, 4096)
	oob := make([]byte, syscall.CmsgSpace(8*4))
	n, oobn, _, _, err := syscall.Recvmsg(s.fd, data, oob, 0)
	if err != nil || (n == 0 && oobn == 0) {
		return false
	}
	s.buf = append(s.buf, data[:n]...)
	if oobn > 0 {
		msgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			s.t.Errorf("fake server: bad control message: %v", err)
			return false
		}
		for _, m := range msgs {
			rights, err := syscall.ParseUnixRights(&m)
			if err != nil {
				s.t.Errorf("fake server: bad rights: %v", err)
				return false
			}
			s.fds = append(s.fds, rights...)
		}
	}
	return true
}

// serve reads requests until the client hangs up.
func (s *Server) serve() {
	defer close(s.done)
	for {
		for len(s.buf) < 8 {
			if !s.fill() {
				return
			}
		}
		id := binary.LittleEndian.Uint32(s.buf[0:])
		sizeOp := binary.LittleEndian.Uint32(s.buf[4:])
		size, op := int(sizeOp>>16), uint16(sizeOp&0xffff)
		for len(s.buf) < size {
			if !s.fill() {
				return
			}
		}
		body := append([]byte(nil), s.buf[8:size]...)
		s.buf = s.buf[size:]
		s.handle(id, op, &reader{b: body, s: s})
	}
}

// reader decodes the arguments of a request.
type reader struct {
	b []byte
	s *Server
}

func (r *reader) u32() uint32 {
	v := binary.LittleEndian.Uint32(r.b)
	r.b = r.b[4:]
	return v
}
func (r *reader) i32() int32 { return int32(r.u32()) }
func (r *reader) str() string {
	n := int(r.u32())
	if n == 0 {
		return "<null>"
	}
	s := string(r.b[:n-1])
	r.b = r.b[(n+3)&^3:]
	return s
}
func (r *reader) fd() int {
	if len(r.s.fds) == 0 {
		r.s.t.Errorf("fake server: a request needs a file descriptor, but none arrived")
		return -1
	}
	fd := r.s.fds[0]
	r.s.fds = r.s.fds[1:]
	return fd
}

// handle answers one request. The opcodes are those of the protocol XML.
func (s *Server) handle(id uint32, op uint16, r *reader) {
	s.mu.Lock()
	iface := s.objects[id]
	s.mu.Unlock()
	switch iface + "." + fmt.Sprint(op) {
	case "wl_display.0": // sync(callback)
		cb := r.u32()
		s.addObject(cb, "wl_callback")
		s.Send(cb, 0, uint32(7)) // wl_callback.done(callback_data)
		s.Send(1, 1, cb)         // wl_display.delete_id(id)
	case "wl_display.1": // get_registry(registry)
		reg := r.u32()
		s.addObject(reg, "wl_registry")
		s.record("wl_display.get_registry")
		for _, g := range s.globals {
			s.Send(reg, 0, g.Name, g.Interface, g.Version) // wl_registry.global
		}
	case "wl_registry.0": // bind(name, interface, version, id)
		name, ifaceName, version, newID := r.u32(), r.str(), r.u32(), r.u32()
		s.addObject(newID, ifaceName)
		s.record("wl_registry.bind name=%d interface=%s version=%d", name, ifaceName, version)
		if ifaceName == "wl_output" {
			// geometry(x, y, physical_width, physical_height, subpixel, make, model, transform)
			s.Send(newID, 0, int32(-10), int32(20), int32(300), int32(200), int32(1), "Glutter", "Test Panel", int32(0))
			s.Send(newID, 3, int32(2)) // scale(factor)
		}
	case "wl_compositor.0": // create_surface(id)
		surface := r.u32()
		s.addObject(surface, "wl_surface")
		s.record("wl_compositor.create_surface")
		if output := s.ID("wl_output"); output != 0 {
			s.Send(surface, 0, output) // wl_surface.enter(output)
		}
		s.Send(surface, 2, int32(2)) // preferred_buffer_scale(factor)
	case "wl_surface.1": // attach(buffer, x, y)
		s.record("wl_surface.attach buffer=%d x=%d y=%d", r.u32(), r.i32(), r.i32())
	case "wl_surface.6":
		s.record("wl_surface.commit")
	case "wl_surface.3": // frame(callback)
		cb := r.u32()
		s.addObject(cb, "wl_callback")
		s.record("wl_surface.frame")
	case "wl_seat.0": // get_pointer(id)
		ptr := r.u32()
		s.addObject(ptr, "wl_pointer")
		s.record("wl_seat.get_pointer")
		surface := s.ID("wl_surface")
		s.Send(ptr, 0, uint32(11), surface, Fixed(10.5), Fixed(-3.25)) // enter
		s.Send(ptr, 2, uint32(1234), Fixed(0.125), Fixed(99))          // motion
	case "wl_seat.1": // get_keyboard(id)
		kb := r.u32()
		s.addObject(kb, "wl_keyboard")
		s.record("wl_seat.get_keyboard")
		keymap, err := os.CreateTemp(s.t.TempDir(), "keymap")
		if err != nil {
			s.t.Errorf("fake server: %v", err)
			return
		}
		if _, err := keymap.WriteString(KeymapText); err != nil {
			s.t.Errorf("fake server: %v", err)
		}
		if _, err := keymap.Seek(0, 0); err != nil {
			s.t.Errorf("fake server: %v", err)
		}
		s.Send(kb, 0, uint32(1), FD(keymap.Fd()), uint32(len(KeymapText)))                    // keymap(format, fd, size)
		s.Send(kb, 1, uint32(5), s.ID("wl_surface"), Array([]byte{30, 0, 0, 0, 31, 0, 0, 0})) // enter(serial, surface, keys)
		if err := keymap.Close(); err != nil {
			s.t.Errorf("fake server: %v", err)
		}
	case "wl_data_device_manager.1": // get_data_device(id, seat)
		dd, seat := r.u32(), r.u32()
		s.addObject(dd, "wl_data_device")
		s.record("wl_data_device_manager.get_data_device seat=%d", seat)
		offer := s.newServerID()
		s.addObject(offer, "wl_data_offer")
		s.Send(dd, 0, offer)                         // data_offer(id): the server creates the object
		s.Send(offer, 0, "text/plain;charset=utf-8") // offer(mime_type)
		s.Send(offer, 0, "image/png")
	case "wl_data_offer.1": // receive(mime_type, fd)
		mime, fd := r.str(), r.fd()
		s.record("wl_data_offer.receive mime=%s fd=%v", mime, fd >= 0)
		_ = syscall.Close(fd)
	case "wl_data_offer.0": // accept(serial, mime_type)
		s.record("wl_data_offer.accept serial=%d mime=%s", r.u32(), r.str())
	case "wl_shm.0": // create_pool(id, fd, size)
		pool, fd, size := r.u32(), r.fd(), r.i32()
		s.addObject(pool, "wl_shm_pool")
		var st syscall.Stat_t
		if err := syscall.Fstat(fd, &st); err != nil {
			s.t.Errorf("fake server: fstat of the received fd: %v", err)
		}
		s.record("wl_shm.create_pool size=%d filesize=%d", size, st.Size)
		_ = syscall.Close(fd)
	case "xdg_wm_base.2": // get_xdg_surface(id, surface)
		xs, surface := r.u32(), r.u32()
		s.addObject(xs, "xdg_surface")
		s.record("xdg_wm_base.get_xdg_surface surface=%v", surface == s.ID("wl_surface"))
	case "xdg_surface.1": // get_toplevel(id)
		tl := r.u32()
		s.addObject(tl, "xdg_toplevel")
		s.record("xdg_surface.get_toplevel")
		// configure(width, height, states) with states maximized (1) and activated (4)
		s.Send(tl, 0, int32(800), int32(600), Array([]byte{1, 0, 0, 0, 4, 0, 0, 0}))
		s.Send(tl, 3, Array([]byte{1, 0, 0, 0, 2, 0, 0, 0})) // wm_capabilities
		s.Send(id, 0, uint32(77))                            // xdg_surface.configure(serial)
	case "xdg_wm_base.3": // pong(serial)
		s.record("xdg_wm_base.pong serial=%d", r.u32())
	case "xdg_surface.4":
		s.record("xdg_surface.ack_configure serial=%d", r.u32())
	case "xdg_toplevel.2":
		s.record("xdg_toplevel.set_title %q", r.str())
	case "xdg_toplevel.3":
		s.record("xdg_toplevel.set_app_id %q", r.str())
	case "xdg_toplevel.8":
		s.record("xdg_toplevel.set_min_size %dx%d", r.i32(), r.i32())
	case "xdg_toplevel.0":
		s.record("xdg_toplevel.destroy")
	default:
		s.t.Errorf("fake server: unexpected request %s opcode %d on object %d", iface, op, id)
	}
}

// Send writes one event. Arguments are uint32, int32, string, Fixed, Array or
// FD (sent as ancillary data). It is safe to call from any goroutine.
//
// Flutter: no counterpart
func (s *Server) Send(id uint32, op uint16, args ...any) {
	var body []byte
	var fds []int
	put32 := func(v uint32) { body = binary.LittleEndian.AppendUint32(body, v) }
	pad := func() {
		for len(body)%4 != 0 {
			body = append(body, 0)
		}
	}
	for _, a := range args {
		switch v := a.(type) {
		case uint32:
			put32(v)
		case int32:
			put32(uint32(v))
		case Fixed:
			put32(uint32(int32(math.Round(float64(v) * 256))))
		case string:
			put32(uint32(len(v) + 1))
			body = append(body, v...)
			body = append(body, 0)
			pad()
		case Array:
			put32(uint32(len(v)))
			body = append(body, v...)
			pad()
		case FD:
			fds = append(fds, int(v))
		default:
			s.t.Errorf("fake server: cannot send %T", a)
		}
	}
	msg := binary.LittleEndian.AppendUint32(nil, id)
	msg = binary.LittleEndian.AppendUint32(msg, uint32(8+len(body))<<16|uint32(op))
	msg = append(msg, body...)

	s.mu.Lock()
	defer s.mu.Unlock()
	var oob []byte
	if len(fds) > 0 {
		oob = syscall.UnixRights(fds...)
	}
	if err := syscall.Sendmsg(s.fd, msg, oob, nil, 0); err != nil {
		s.t.Errorf("fake server: sendmsg: %v", err)
	}
}

// Joined returns the log of requests as one string.
//
// Flutter: no counterpart
func (s *Server) Joined() string { return strings.Join(s.Requests(), "\n") }

// Hangup shuts the connection down, as a crashing compositor would.
//
// Flutter: no counterpart
func (s *Server) Hangup() { _ = syscall.Shutdown(s.fd, syscall.SHUT_RDWR) }

// KeymapText is the content the fake keyboard sends as its "keymap" file.
//
// Flutter: no counterpart
const KeymapText = "xkb_keymap { fake };"
