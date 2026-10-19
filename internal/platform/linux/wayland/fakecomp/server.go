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

	"github.com/rm4n0s/errors"
)

// Server is a fake compositor on one end of a socket pair.
//
// It handles only the requests the tests send. Every request it receives is
// appended to the log as text, so tests can assert on what the client sent.
//
// Flutter: no counterpart
type Server struct {
	t          testing.TB
	fd         int
	globals    []Global
	toplevel   ToplevelConfigure
	holdFrames bool
	keymap     string
	caps       uint32
	serial     uint32 // the last serial sent with a keyboard event

	selection string // the text of the selection, guarded by mu
	stall     bool
	stalled   []int // the pipes that a stalled owner holds

	outputScale, bufferScale int32

	win windowing // see windowing.go; guarded by mu

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
	// HoldFrames keeps the frame callbacks (wl_surface.frame) until the test calls
	// FrameDone, instead of answering each one on the commit that follows it. A
	// real compositor answers when the frame is shown, some time after the
	// commit.
	HoldFrames bool
	// Keymap is the text of the keyboard's keymap. Empty means KeymapText, which is
	// not a real keymap.
	Keymap string
	// Capabilities is what wl_seat.capabilities announces (1 pointer, 2 keyboard, 4
	// touch). Zero means a keyboard only.
	Capabilities uint32
	// Toplevel is what the initial configure of a toplevel window says. The zero
	// value lets the client choose its size.
	Toplevel ToplevelConfigure
	// OutputScale is the wl_output.scale that a wl_output announces when it is bound;
	// zero means 2.
	OutputScale int32
	// PreferredBufferScale is the wl_surface.preferred_buffer_scale that a new
	// wl_surface announces; zero means 2, and a negative number sends none, as a
	// compositor with wl_compositor older than version 6 does.
	PreferredBufferScale int32
	// Selection, when it is not empty, is the text of the clipboard that another
	// application owns: the data device announces it as the selection, in the
	// formats "text/plain;charset=utf-8" and "image/png", and a receive request
	// for a text format gets it.
	Selection string
	// StallSelection makes the owner of the selection never write the data: the
	// pipe of a receive request stays open and empty until the server stops.
	StallSelection bool
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
	keymap, caps := cfg.Keymap, cfg.Capabilities
	if keymap == "" {
		keymap = KeymapText
	}
	if caps == 0 {
		caps = 2
	}
	outputScale, bufferScale := cfg.OutputScale, cfg.PreferredBufferScale
	if outputScale == 0 {
		outputScale = 2
	}
	if bufferScale == 0 {
		bufferScale = 2
	}
	s = &Server{
		outputScale: outputScale, bufferScale: bufferScale,
		t: t, fd: fds[0], globals: globals, toplevel: cfg.Toplevel, holdFrames: cfg.HoldFrames,
		keymap: keymap, caps: caps, selection: cfg.Selection, stall: cfg.StallSelection,
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
	defer func() {
		for _, fd := range s.stalled {
			_ = syscall.Close(fd)
		}
	}()
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
			s.Send(newID, 3, s.outputScale) // scale(factor)
		}
		if ifaceName == "wl_seat" {
			s.Send(newID, 0, s.caps) // capabilities
		}
	case "wl_compositor.0": // create_surface(id)
		surface := r.u32()
		s.addObject(surface, "wl_surface")
		s.record("wl_compositor.create_surface")
		if output := s.ID("wl_output"); output != 0 {
			s.Send(surface, 0, output) // wl_surface.enter(output)
		}
		if s.bufferScale > 0 {
			s.Send(surface, 2, s.bufferScale) // preferred_buffer_scale(factor)
		}
	case "wl_seat.0": // get_pointer(id)
		ptr := r.u32()
		s.addObject(ptr, "wl_pointer")
		s.record("wl_seat.get_pointer")
		if surface := s.ID("wl_surface"); surface != 0 {
			s.Send(ptr, 0, uint32(11), surface, Fixed(10.5), Fixed(-3.25)) // enter
			s.Send(ptr, 2, uint32(1234), Fixed(0.125), Fixed(99))          // motion
		}
	case "wl_seat.2": // get_touch(id)
		t := r.u32()
		s.addObject(t, "wl_touch")
		s.record("wl_seat.get_touch")
	case "wl_seat.1": // get_keyboard(id)
		kb := r.u32()
		s.addObject(kb, "wl_keyboard")
		s.record("wl_seat.get_keyboard")
		keymap, err := os.CreateTemp(s.t.TempDir(), "keymap")
		if err != nil {
			s.t.Errorf("fake server: %v", err)
			return
		}
		if _, err := keymap.WriteString(s.keymap); err != nil {
			s.t.Errorf("fake server: %v", err)
		}
		if _, err := keymap.Seek(0, 0); err != nil {
			s.t.Errorf("fake server: %v", err)
		}
		s.Send(kb, 0, uint32(1), FD(keymap.Fd()), uint32(len(s.keymap))) // keymap(format, fd, size)
		if surface := s.ID("wl_surface"); surface != 0 {
			s.Send(kb, 1, uint32(5), surface, Array([]byte{30, 0, 0, 0, 31, 0, 0, 0})) // enter(serial, surface, keys)
		}
		if err := keymap.Close(); err != nil {
			s.t.Errorf("fake server: %v", err)
		}
	case "wl_keyboard.0": // release
		s.record("wl_keyboard.release")
	case "wl_data_device_manager.1": // get_data_device(id, seat)
		dd, seat := r.u32(), r.u32()
		s.addObject(dd, "wl_data_device")
		s.record("wl_data_device_manager.get_data_device seat=%d", seat)
		offer := s.newServerID()
		s.addObject(offer, "wl_data_offer")
		s.Send(dd, 0, offer)                         // data_offer(id): the server creates the object
		s.Send(offer, 0, "text/plain;charset=utf-8") // offer(mime_type)
		s.Send(offer, 0, "image/png")
		s.mu.Lock()
		hasSelection := s.selection != ""
		s.mu.Unlock()
		if hasSelection {
			s.Send(dd, 5, offer) // selection(id)
		}
	case "wl_data_device_manager.0": // create_data_source(id)
		src := r.u32()
		s.addObject(src, "wl_data_source")
		s.record("wl_data_device_manager.create_data_source")
	case "wl_data_source.0": // offer(mime_type)
		s.record("wl_data_source.offer mime=%s", r.str())
	case "wl_data_source.1": // destroy
		s.record("wl_data_source.destroy")
	case "wl_data_device.1": // set_selection(source, serial)
		src, serial := r.u32(), r.u32()
		s.record("wl_data_device.set_selection source=%d serial=%d", src, serial)
	case "wl_data_device.2": // release
		s.record("wl_data_device.release")
	case "wl_data_offer.1": // receive(mime_type, fd)
		mime, fd := r.str(), r.fd()
		s.record("wl_data_offer.receive mime=%s fd=%v", mime, fd >= 0)
		s.mu.Lock()
		text := s.selection
		s.mu.Unlock()
		if fd >= 0 && s.stall {
			s.stalled = append(s.stalled, fd) // closed when the server stops
		} else if fd >= 0 && text != "" && (strings.HasPrefix(mime, "text/") || mime == "UTF8_STRING") {
			// The other application writes the text from its own goroutine.
			go func(fd int) {
				f := os.NewFile(uintptr(fd), "selection")
				_, _ = f.WriteString(text)
				_ = f.Close()
			}(fd)
		} else {
			_ = syscall.Close(fd)
		}
	case "wl_data_offer.2": // destroy
		s.record("wl_data_offer.destroy")
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
	case "xdg_wm_base.3": // pong(serial)
		s.record("xdg_wm_base.pong serial=%d", r.u32())
	default:
		if !s.handleInput(iface, op, id, r) && !s.handleWindowing(iface, op, id, r) {
			s.t.Errorf("fake server: unexpected request %s opcode %d on object %d", iface, op, id)
		}
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
	if err := syscall.Sendmsg(s.fd, msg, oob, nil, 0); err != nil && !errors.Is(err, syscall.EPIPE) && !errors.Is(err, syscall.ECONNRESET) {
		// A client that has already gone away is not a failure of the test.
		s.t.Errorf("fake server: sendmsg: %v", err)
	}
}

// OfferSelection makes another application the owner of the clipboard, with the
// text: the data device gets a new offer in the formats and a selection event.
// A receive request for a text format gets the text.
//
// Flutter: no counterpart
func (s *Server) OfferSelection(text string, mimes ...string) {
	s.mu.Lock()
	s.selection = text
	s.mu.Unlock()
	offer := s.newServerID()
	s.addObject(offer, "wl_data_offer")
	dd := s.ID("wl_data_device")
	s.Send(dd, 0, offer) // data_offer(id)
	for _, m := range mimes {
		s.Send(offer, 0, m) // offer(mime_type)
	}
	s.Send(dd, 5, offer) // selection(id)
}

// ClearSelection empties the clipboard: the data device announces no selection.
//
// Flutter: no counterpart
func (s *Server) ClearSelection() {
	s.mu.Lock()
	s.selection = ""
	s.mu.Unlock()
	s.Send(s.ID("wl_data_device"), 5, uint32(0)) // selection(null)
}

// Joined returns the log of requests as one string.
//
// Flutter: no counterpart
func (s *Server) Joined() string { return strings.Join(s.Requests(), "\n") }

// Hangup shuts the connection down, as a crashing compositor would.
//
// Flutter: no counterpart
func (s *Server) Hangup() { _ = syscall.Shutdown(s.fd, syscall.SHUT_RDWR) }

// KeymapText is the content the fake keyboard sends as its "keymap" file, unless
// Config.Keymap says otherwise: the smallest keymap that libxkbcommon accepts,
// with one key.
//
// Flutter: no counterpart
const KeymapText = `xkb_keymap {
	xkb_keycodes { minimum = 8; maximum = 255; <AC01> = 38; };
	xkb_types { type "ONE_LEVEL" { modifiers = none; map[none] = Level1; }; };
	xkb_compat { };
	xkb_symbols { key <AC01> { [ a ] }; };
};
`
