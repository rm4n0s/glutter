package fakecomp

import (
	"fmt"
	"strings"
)

// ToplevelConfigure is the content of an xdg_toplevel.configure event.
//
// Flutter: no counterpart
type ToplevelConfigure struct {
	// Width and Height are the suggested size; 0 lets the client choose.
	Width, Height int32
	// States are the xdg_toplevel.state values (1 maximized, 4 activated, ...).
	States []uint32
	// Capabilities are the xdg_toplevel.wm_capabilities values sent with the
	// initial configure; nil sends none.
	Capabilities []uint32
}

// windowing is the state of the windowing requests: which surface has which
// xdg role, which buffer is attached, and whether the first configure went out.
type windowing struct {
	xdgSurface map[uint32]uint32 // xdg_surface id -> wl_surface id
	toplevel   map[uint32]uint32 // xdg_toplevel id -> xdg_surface id
	surfaceTop map[uint32]uint32 // wl_surface id -> xdg_toplevel id
	configured map[uint32]bool   // xdg_toplevel id -> initial configure sent
	lastTop    uint32            // the newest xdg_toplevel
	pending    map[uint32]uint32 // wl_surface id -> buffer attached since the last commit
	current    map[uint32]uint32 // wl_surface id -> buffer on display
	bufferSize map[uint32][2]int32
	serial     uint32
	decoration map[uint32]uint32 // zxdg_toplevel_decoration_v1 id -> xdg_toplevel id

	frames  map[uint32][]uint32 // wl_surface id -> frame callbacks requested since its last commit
	held    []uint32            // committed frame callbacks that wait for FrameDone
	clockMs uint32              // the time of the next frame callback
}

func (w *windowing) init() {
	if w.xdgSurface != nil {
		return
	}
	w.xdgSurface = map[uint32]uint32{}
	w.toplevel = map[uint32]uint32{}
	w.surfaceTop = map[uint32]uint32{}
	w.configured = map[uint32]bool{}
	w.pending = map[uint32]uint32{}
	w.current = map[uint32]uint32{}
	w.bufferSize = map[uint32][2]int32{}
	w.decoration = map[uint32]uint32{}
	w.frames = map[uint32][]uint32{}
	w.serial = 100
}

// nextSerial returns a new configure serial. s.mu must be held.
func (w *windowing) nextSerial() uint32 {
	w.serial++
	return w.serial
}

// toplevelStates encodes states as the wl_array of an xdg_toplevel.configure.
func uint32Array(v []uint32) Array {
	out := make([]byte, 0, len(v)*4)
	for _, x := range v {
		out = append(out, byte(x), byte(x>>8), byte(x>>16), byte(x>>24))
	}
	return Array(out)
}

// ConfigureToplevel sends a configure for the newest toplevel, as a compositor
// does when the window is resized, maximized or focused: the toplevel event,
// then the xdg_surface event that closes the batch.
//
// Flutter: no counterpart
func (s *Server) ConfigureToplevel(c ToplevelConfigure) {
	s.mu.Lock()
	s.win.init()
	top := s.win.lastTop
	xs := s.win.toplevel[top]
	serial := s.win.nextSerial()
	s.mu.Unlock()
	if top == 0 {
		s.t.Errorf("fake server: ConfigureToplevel without a toplevel")
		return
	}
	s.Send(top, 0, c.Width, c.Height, uint32Array(c.States)) // xdg_toplevel.configure
	s.Send(xs, 0, serial)                                    // xdg_surface.configure
}

// CloseToplevel sends xdg_toplevel.close for the newest toplevel, as a
// compositor does when the user clicks the close button.
//
// Flutter: no counterpart
func (s *Server) CloseToplevel() {
	s.mu.Lock()
	s.win.init()
	top := s.win.lastTop
	s.mu.Unlock()
	if top == 0 {
		s.t.Errorf("fake server: CloseToplevel without a toplevel")
		return
	}
	s.Send(top, 1) // xdg_toplevel.close
}

// handleWindowing answers the surface, xdg-shell, shm-buffer and decoration
// requests. It reports whether it knew the request.
func (s *Server) handleWindowing(iface string, op uint16, id uint32, r *reader) bool {
	s.mu.Lock()
	s.win.init()
	s.mu.Unlock()

	switch iface + "." + fmt.Sprint(op) {
	case "wl_surface.0": // destroy
		s.record("wl_surface.destroy")
	case "wl_surface.1": // attach(buffer, x, y)
		buffer, x, y := r.u32(), r.i32(), r.i32()
		s.mu.Lock()
		s.win.pending[id] = buffer
		s.mu.Unlock()
		s.record("wl_surface.attach buffer=%d x=%d y=%d", buffer, x, y)
	case "wl_surface.2": // damage
		s.record("wl_surface.damage %d,%d %dx%d", r.i32(), r.i32(), r.i32(), r.i32())
	case "wl_surface.9": // damage_buffer
		s.record("wl_surface.damage_buffer %d,%d %dx%d", r.i32(), r.i32(), r.i32(), r.i32())
	case "wl_surface.3": // frame(callback)
		cb := r.u32()
		s.addObject(cb, "wl_callback")
		s.mu.Lock()
		s.win.frames[id] = append(s.win.frames[id], cb)
		s.mu.Unlock()
		s.record("wl_surface.frame")
	case "wl_surface.6": // commit
		s.commit(id)
	case "wl_shm_pool.0": // create_buffer(id, offset, width, height, stride, format)
		buffer, offset, w, h, stride, format := r.u32(), r.i32(), r.i32(), r.i32(), r.i32(), r.u32()
		s.addObject(buffer, "wl_buffer")
		s.mu.Lock()
		s.win.bufferSize[buffer] = [2]int32{w, h}
		s.mu.Unlock()
		s.record("wl_shm_pool.create_buffer %dx%d offset=%d stride=%d format=%d", w, h, offset, stride, format)
	case "wl_shm_pool.1":
		s.record("wl_shm_pool.destroy")
	case "wl_buffer.0":
		s.record("wl_buffer.destroy")
	case "xdg_wm_base.2": // get_xdg_surface(id, surface)
		xs, surface := r.u32(), r.u32()
		s.addObject(xs, "xdg_surface")
		s.mu.Lock()
		s.win.xdgSurface[xs] = surface
		s.mu.Unlock()
		s.record("xdg_wm_base.get_xdg_surface surface=%v", surface == s.ID("wl_surface"))
	case "xdg_surface.0":
		s.record("xdg_surface.destroy")
	case "xdg_surface.1": // get_toplevel(id)
		top := r.u32()
		s.addObject(top, "xdg_toplevel")
		s.mu.Lock()
		s.win.toplevel[top] = id
		s.win.surfaceTop[s.win.xdgSurface[id]] = top
		s.win.lastTop = top
		s.mu.Unlock()
		s.record("xdg_surface.get_toplevel")
	case "xdg_surface.3": // set_window_geometry
		s.record("xdg_surface.set_window_geometry %d,%d %dx%d", r.i32(), r.i32(), r.i32(), r.i32())
	case "xdg_surface.4":
		s.record("xdg_surface.ack_configure serial=%d", r.u32())
	case "xdg_toplevel.0":
		s.record("xdg_toplevel.destroy")
	case "xdg_toplevel.1": // set_parent(parent)
		parent := r.u32()
		s.record("xdg_toplevel.set_parent %v", parent != 0)
	case "xdg_toplevel.2":
		s.record("xdg_toplevel.set_title %q", r.str())
	case "xdg_toplevel.3":
		s.record("xdg_toplevel.set_app_id %q", r.str())
	case "xdg_toplevel.7":
		s.record("xdg_toplevel.set_max_size %dx%d", r.i32(), r.i32())
	case "xdg_toplevel.8":
		s.record("xdg_toplevel.set_min_size %dx%d", r.i32(), r.i32())
	case "xdg_toplevel.9":
		s.record("xdg_toplevel.set_maximized")
	case "xdg_toplevel.10":
		s.record("xdg_toplevel.unset_maximized")
	case "xdg_toplevel.11":
		s.record("xdg_toplevel.set_fullscreen")
	case "xdg_toplevel.12":
		s.record("xdg_toplevel.unset_fullscreen")
	case "xdg_toplevel.13":
		s.record("xdg_toplevel.set_minimized")
	case "zxdg_decoration_manager_v1.0":
		s.record("zxdg_decoration_manager_v1.destroy")
	case "zxdg_decoration_manager_v1.1": // get_toplevel_decoration(id, toplevel)
		dec, top := r.u32(), r.u32()
		s.addObject(dec, "zxdg_toplevel_decoration_v1")
		s.mu.Lock()
		s.win.decoration[dec] = top
		s.mu.Unlock()
		s.record("zxdg_decoration_manager_v1.get_toplevel_decoration")
	case "zxdg_toplevel_decoration_v1.0":
		s.record("zxdg_toplevel_decoration_v1.destroy")
	case "zxdg_toplevel_decoration_v1.1": // set_mode(mode)
		mode := r.u32()
		s.record("zxdg_toplevel_decoration_v1.set_mode %d", mode)
		s.Send(id, 0, mode) // configure(mode): the compositor agrees
	case "zxdg_toplevel_decoration_v1.2":
		s.record("zxdg_toplevel_decoration_v1.unset_mode")
	default:
		return false
	}
	return true
}

// commit applies a wl_surface.commit: the first commit of a toplevel brings its
// initial configure, and a new buffer releases the old one.
func (s *Server) commit(surface uint32) {
	s.mu.Lock()
	buffer := s.win.pending[surface]
	s.win.pending[surface] = 0
	size := s.win.bufferSize[buffer]
	old := s.win.current[surface]
	if buffer != 0 {
		s.win.current[surface] = buffer
	}
	callbacks := s.win.frames[surface]
	s.win.frames[surface] = nil
	if s.holdFrames {
		s.win.held = append(s.win.held, callbacks...)
		callbacks = nil
	}
	top := s.win.surfaceTop[surface]
	sendInitial := top != 0 && !s.win.configured[top]
	var xs, serial uint32
	if sendInitial {
		s.win.configured[top] = true
		xs = s.win.toplevel[top]
		serial = s.win.nextSerial()
	}
	s.mu.Unlock()

	if buffer != 0 {
		s.record("wl_surface.commit buffer=%d %dx%d", buffer, size[0], size[1])
	} else {
		s.record("wl_surface.commit")
	}
	if old != 0 && buffer != 0 && old != buffer {
		s.Send(old, 0) // wl_buffer.release
	}
	s.finishFrames(callbacks)
	if sendInitial {
		c := s.toplevel
		if c.Capabilities != nil {
			s.Send(top, 3, uint32Array(c.Capabilities)) // xdg_toplevel.wm_capabilities
		}
		s.Send(top, 0, c.Width, c.Height, uint32Array(c.States)) // xdg_toplevel.configure
		s.Send(xs, 0, serial)                                    // xdg_surface.configure
	}
}

// Mapped reports whether a buffer is on display for the newest toplevel's
// surface, and its size.
//
// Flutter: no counterpart
func (s *Server) Mapped() (width, height int32, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.win.init()
	surface := s.win.xdgSurface[s.win.toplevel[s.win.lastTop]]
	buffer := s.win.current[surface]
	if buffer == 0 {
		return 0, 0, false
	}
	size := s.win.bufferSize[buffer]
	return size[0], size[1], true
}

// RequestsWith returns the logged requests that contain substr.
//
// Flutter: no counterpart
func (s *Server) RequestsWith(substr string) []string {
	var out []string
	for _, line := range s.Requests() {
		if strings.Contains(line, substr) {
			out = append(out, line)
		}
	}
	return out
}

// finishFrames answers frame callbacks: done(time), then the display's
// delete_id for the callback.
func (s *Server) finishFrames(callbacks []uint32) {
	for _, cb := range callbacks {
		s.mu.Lock()
		s.win.clockMs += 16
		ms := s.win.clockMs
		s.mu.Unlock()
		s.Send(cb, 0, ms) // wl_callback.done(callback_data)
		s.Send(1, 1, cb)  // wl_display.delete_id(id)
	}
}

// FrameDone answers the frame callbacks that Config.HoldFrames kept back, as a
// compositor does when it has shown the frame.
//
// Flutter: no counterpart
func (s *Server) FrameDone() {
	s.mu.Lock()
	s.win.init()
	held := s.win.held
	s.win.held = nil
	s.mu.Unlock()
	s.finishFrames(held)
}

// HeldFrames returns how many frame callbacks wait for FrameDone.
//
// Flutter: no counterpart
func (s *Server) HeldFrames() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.win.held)
}
