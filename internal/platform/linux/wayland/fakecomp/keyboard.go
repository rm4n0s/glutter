package fakecomp

// KeyboardEnter sends wl_keyboard.enter: the surface gets keyboard focus, and
// keys are the evdev scancodes that are already held down.
//
// Flutter: no counterpart
func (s *Server) KeyboardEnter(surface uint32, keys ...uint32) {
	var array Array
	for _, k := range keys {
		array = append(array, byte(k), byte(k>>8), byte(k>>16), byte(k>>24))
	}
	s.Send(s.ID("wl_keyboard"), 1, s.nextSerial(), surface, array)
}

// KeyboardLeave sends wl_keyboard.leave for the surface.
//
// Flutter: no counterpart
func (s *Server) KeyboardLeave(surface uint32) {
	s.Send(s.ID("wl_keyboard"), 2, s.nextSerial(), surface)
}

// KeyboardKey sends wl_keyboard.key for the evdev scancode.
//
// Flutter: no counterpart
func (s *Server) KeyboardKey(key uint32, pressed bool) {
	state := uint32(0)
	if pressed {
		state = 1
	}
	s.Send(s.ID("wl_keyboard"), 3, s.nextSerial(), uint32(0), key, state)
}

// KeyboardModifiers sends wl_keyboard.modifiers.
//
// Flutter: no counterpart
func (s *Server) KeyboardModifiers(depressed, latched, locked, group uint32) {
	s.Send(s.ID("wl_keyboard"), 4, s.nextSerial(), depressed, latched, locked, group)
}

// KeyboardRepeatInfo sends wl_keyboard.repeat_info: rate keys per second after
// delay milliseconds.
//
// Flutter: no counterpart
func (s *Server) KeyboardRepeatInfo(rate, delay int32) {
	s.Send(s.ID("wl_keyboard"), 5, rate, delay)
}

// SeatCapabilities sends wl_seat.capabilities.
//
// Flutter: no counterpart
func (s *Server) SeatCapabilities(caps uint32) {
	s.Send(s.ID("wl_seat"), 0, caps)
}

func (s *Server) nextSerial() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serial++
	return s.serial
}
