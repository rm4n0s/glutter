//go:build linux && !android

package fakecomp

import "fmt"

// TextInputGlobals returns the globals of DefaultGlobals plus
// zwp_text_input_manager_v3, for a test of the input method.
//
// Flutter: no counterpart
func TextInputGlobals() []Global {
	return append(DefaultGlobals(), Global{Name: 20, Interface: "zwp_text_input_manager_v3", Version: 1})
}

// handleTextInput answers the requests of zwp_text_input_manager_v3 and
// zwp_text_input_v3. It reports whether it knew the request.
func (s *Server) handleTextInput(iface string, op uint16, _ uint32, r *reader) bool {
	switch iface + "." + fmt.Sprint(op) {
	case "zwp_text_input_manager_v3.0":
		s.record("zwp_text_input_manager_v3.destroy")
	case "zwp_text_input_manager_v3.1": // get_text_input(id, seat)
		id := r.u32()
		s.addObject(id, "zwp_text_input_v3")
		s.record("zwp_text_input_manager_v3.get_text_input")
	case "zwp_text_input_v3.0":
		s.record("zwp_text_input_v3.destroy")
	case "zwp_text_input_v3.1":
		s.record("zwp_text_input_v3.enable")
	case "zwp_text_input_v3.2":
		s.record("zwp_text_input_v3.disable")
	case "zwp_text_input_v3.3": // set_surrounding_text(text, cursor, anchor)
		text, cursor, anchor := r.str(), r.i32(), r.i32()
		s.record("zwp_text_input_v3.set_surrounding_text text=%q cursor=%d anchor=%d", text, cursor, anchor)
	case "zwp_text_input_v3.4": // set_text_change_cause(cause)
		s.record("zwp_text_input_v3.set_text_change_cause cause=%d", r.u32())
	case "zwp_text_input_v3.5": // set_content_type(hint, purpose)
		hint, purpose := r.u32(), r.u32()
		s.record("zwp_text_input_v3.set_content_type hint=%#x purpose=%d", hint, purpose)
	case "zwp_text_input_v3.6": // set_cursor_rectangle(x, y, width, height)
		x, y, w, h := r.i32(), r.i32(), r.i32(), r.i32()
		s.record("zwp_text_input_v3.set_cursor_rectangle x=%d y=%d width=%d height=%d", x, y, w, h)
	case "zwp_text_input_v3.7":
		s.mu.Lock()
		s.textCommits++
		n := s.textCommits
		s.mu.Unlock()
		s.record("zwp_text_input_v3.commit serial=%d", n)
	default:
		return false
	}
	return true
}

// TextInputEnter sends zwp_text_input_v3.enter for the surface.
//
// Flutter: no counterpart
func (s *Server) TextInputEnter(surface uint32) { s.Send(s.ID("zwp_text_input_v3"), 0, surface) }

// TextInputLeave sends zwp_text_input_v3.leave for the surface.
//
// Flutter: no counterpart
func (s *Server) TextInputLeave(surface uint32) { s.Send(s.ID("zwp_text_input_v3"), 1, surface) }

// TextInputPreedit sends zwp_text_input_v3.preedit_string.
//
// Flutter: no counterpart
func (s *Server) TextInputPreedit(text string, cursorBegin, cursorEnd int32) {
	s.Send(s.ID("zwp_text_input_v3"), 2, text, cursorBegin, cursorEnd)
}

// TextInputCommit sends zwp_text_input_v3.commit_string.
//
// Flutter: no counterpart
func (s *Server) TextInputCommit(text string) { s.Send(s.ID("zwp_text_input_v3"), 3, text) }

// TextInputDelete sends zwp_text_input_v3.delete_surrounding_text.
//
// Flutter: no counterpart
func (s *Server) TextInputDelete(before, after uint32) {
	s.Send(s.ID("zwp_text_input_v3"), 4, before, after)
}

// TextInputDone sends zwp_text_input_v3.done with the number of commit requests
// received so far as the serial.
//
// Flutter: no counterpart
func (s *Server) TextInputDone() {
	s.mu.Lock()
	n := s.textCommits
	s.mu.Unlock()
	s.Send(s.ID("zwp_text_input_v3"), 5, n)
}

// TextInputDoneSerial sends zwp_text_input_v3.done with the serial given.
//
// Flutter: no counterpart
func (s *Server) TextInputDoneSerial(serial uint32) { s.Send(s.ID("zwp_text_input_v3"), 5, serial) }
