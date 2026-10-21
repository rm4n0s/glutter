//go:build linux

package fakecomp

import "fmt"

// handleInput answers the requests of the pointer, touch, cursor-shape and
// pointer-gestures objects. It reports whether it knew the request.
func (s *Server) handleInput(iface string, op uint16, id uint32, r *reader) bool {
	switch iface + "." + fmt.Sprint(op) {
	case "wl_pointer.0": // set_cursor(serial, surface, hotspot_x, hotspot_y)
		serial, surface, x, y := r.u32(), r.u32(), r.i32(), r.i32()
		s.record("wl_pointer.set_cursor serial=%d surface=%v hotspot=%d,%d", serial, surface != 0, x, y)
	case "wl_pointer.1":
		s.record("wl_pointer.release")
	case "wl_touch.0":
		s.record("wl_touch.release")
	case "wp_cursor_shape_manager_v1.0":
		s.record("wp_cursor_shape_manager_v1.destroy")
	case "wp_cursor_shape_manager_v1.1": // get_pointer(id, pointer)
		dev := r.u32()
		s.addObject(dev, "wp_cursor_shape_device_v1")
		s.record("wp_cursor_shape_manager_v1.get_pointer")
	case "wp_cursor_shape_manager_v1.2": // get_tablet_tool_v2(id, tablet_tool)
		dev := r.u32()
		s.addObject(dev, "wp_cursor_shape_device_v1")
		s.record("wp_cursor_shape_manager_v1.get_tablet_tool_v2")
	case "wp_cursor_shape_device_v1.0":
		s.record("wp_cursor_shape_device_v1.destroy")
	case "wp_cursor_shape_device_v1.1": // set_shape(serial, shape)
		serial, shape := r.u32(), r.u32()
		s.record("wp_cursor_shape_device_v1.set_shape serial=%d shape=%d", serial, shape)
	case "zwp_pointer_gestures_v1.0": // get_swipe_gesture(id, pointer)
		g := r.u32()
		s.addObject(g, "zwp_pointer_gesture_swipe_v1")
		s.record("zwp_pointer_gestures_v1.get_swipe_gesture")
	case "zwp_pointer_gestures_v1.1": // get_pinch_gesture(id, pointer)
		g := r.u32()
		s.addObject(g, "zwp_pointer_gesture_pinch_v1")
		s.record("zwp_pointer_gestures_v1.get_pinch_gesture")
	case "zwp_pointer_gestures_v1.2":
		s.record("zwp_pointer_gestures_v1.release")
	case "zwp_pointer_gesture_swipe_v1.0":
		s.record("zwp_pointer_gesture_swipe_v1.destroy")
	case "zwp_pointer_gesture_pinch_v1.0":
		s.record("zwp_pointer_gesture_pinch_v1.destroy")
	default:
		return false
	}
	return true
}

// PointerEnter sends wl_pointer.enter.
//
// Flutter: no counterpart
func (s *Server) PointerEnter(surface uint32, x, y float64) {
	s.Send(s.ID("wl_pointer"), 0, s.nextSerial(), surface, Fixed(x), Fixed(y))
}

// PointerLeave sends wl_pointer.leave.
//
// Flutter: no counterpart
func (s *Server) PointerLeave(surface uint32) {
	s.Send(s.ID("wl_pointer"), 1, s.nextSerial(), surface)
}

// PointerMotion sends wl_pointer.motion.
//
// Flutter: no counterpart
func (s *Server) PointerMotion(x, y float64) {
	s.Send(s.ID("wl_pointer"), 2, uint32(0), Fixed(x), Fixed(y))
}

// PointerButton sends wl_pointer.button for the Linux button code.
//
// Flutter: no counterpart
func (s *Server) PointerButton(button uint32, pressed bool) {
	state := uint32(0)
	if pressed {
		state = 1
	}
	s.Send(s.ID("wl_pointer"), 3, s.nextSerial(), uint32(0), button, state)
}

// PointerAxis sends wl_pointer.axis (0 vertical, 1 horizontal).
//
// Flutter: no counterpart
func (s *Server) PointerAxis(axis uint32, value float64) {
	s.Send(s.ID("wl_pointer"), 4, uint32(0), axis, Fixed(value))
}

// PointerFrame sends wl_pointer.frame.
//
// Flutter: no counterpart
func (s *Server) PointerFrame() { s.Send(s.ID("wl_pointer"), 5) }

// PointerAxisSource sends wl_pointer.axis_source (0 wheel, 1 finger, 2
// continuous, 3 wheel tilt).
//
// Flutter: no counterpart
func (s *Server) PointerAxisSource(source uint32) { s.Send(s.ID("wl_pointer"), 6, source) }

// PointerAxisStop sends wl_pointer.axis_stop.
//
// Flutter: no counterpart
func (s *Server) PointerAxisStop(axis uint32) { s.Send(s.ID("wl_pointer"), 7, uint32(0), axis) }

// PointerAxisDiscrete sends wl_pointer.axis_discrete.
//
// Flutter: no counterpart
func (s *Server) PointerAxisDiscrete(axis uint32, n int32) { s.Send(s.ID("wl_pointer"), 8, axis, n) }

// PointerAxisValue120 sends wl_pointer.axis_value120.
//
// Flutter: no counterpart
func (s *Server) PointerAxisValue120(axis uint32, v int32) { s.Send(s.ID("wl_pointer"), 9, axis, v) }

// PointerAxisDirection sends wl_pointer.axis_relative_direction (1 inverted).
//
// Flutter: no counterpart
func (s *Server) PointerAxisDirection(axis, direction uint32) {
	s.Send(s.ID("wl_pointer"), 10, axis, direction)
}

// TouchDown sends wl_touch.down.
//
// Flutter: no counterpart
func (s *Server) TouchDown(surface uint32, id int32, x, y float64) {
	s.Send(s.ID("wl_touch"), 0, s.nextSerial(), uint32(0), surface, id, Fixed(x), Fixed(y))
}

// TouchUp sends wl_touch.up.
//
// Flutter: no counterpart
func (s *Server) TouchUp(id int32) {
	s.Send(s.ID("wl_touch"), 1, s.nextSerial(), uint32(0), id)
}

// TouchMotion sends wl_touch.motion.
//
// Flutter: no counterpart
func (s *Server) TouchMotion(id int32, x, y float64) {
	s.Send(s.ID("wl_touch"), 2, uint32(0), id, Fixed(x), Fixed(y))
}

// TouchFrame sends wl_touch.frame.
//
// Flutter: no counterpart
func (s *Server) TouchFrame() { s.Send(s.ID("wl_touch"), 3) }

// TouchCancel sends wl_touch.cancel.
//
// Flutter: no counterpart
func (s *Server) TouchCancel() { s.Send(s.ID("wl_touch"), 4) }

// TouchShape sends wl_touch.shape.
//
// Flutter: no counterpart
func (s *Server) TouchShape(id int32, major, minor float64) {
	s.Send(s.ID("wl_touch"), 5, id, Fixed(major), Fixed(minor))
}

// TouchOrientation sends wl_touch.orientation.
//
// Flutter: no counterpart
func (s *Server) TouchOrientation(id int32, degrees float64) {
	s.Send(s.ID("wl_touch"), 6, id, Fixed(degrees))
}

// PinchBegin sends zwp_pointer_gesture_pinch_v1.begin.
//
// Flutter: no counterpart
func (s *Server) PinchBegin(surface, fingers uint32) {
	s.Send(s.ID("zwp_pointer_gesture_pinch_v1"), 0, s.nextSerial(), uint32(0), surface, fingers)
}

// PinchUpdate sends zwp_pointer_gesture_pinch_v1.update.
//
// Flutter: no counterpart
func (s *Server) PinchUpdate(dx, dy, scale, rotationDegrees float64) {
	s.Send(s.ID("zwp_pointer_gesture_pinch_v1"), 1, uint32(0), Fixed(dx), Fixed(dy), Fixed(scale), Fixed(rotationDegrees))
}

// PinchEnd sends zwp_pointer_gesture_pinch_v1.end.
//
// Flutter: no counterpart
func (s *Server) PinchEnd(cancelled bool) {
	c := int32(0)
	if cancelled {
		c = 1
	}
	s.Send(s.ID("zwp_pointer_gesture_pinch_v1"), 2, s.nextSerial(), uint32(0), c)
}

// SwipeBegin sends zwp_pointer_gesture_swipe_v1.begin.
//
// Flutter: no counterpart
func (s *Server) SwipeBegin(surface, fingers uint32) {
	s.Send(s.ID("zwp_pointer_gesture_swipe_v1"), 0, s.nextSerial(), uint32(0), surface, fingers)
}

// SwipeUpdate sends zwp_pointer_gesture_swipe_v1.update.
//
// Flutter: no counterpart
func (s *Server) SwipeUpdate(dx, dy float64) {
	s.Send(s.ID("zwp_pointer_gesture_swipe_v1"), 1, uint32(0), Fixed(dx), Fixed(dy))
}

// SwipeEnd sends zwp_pointer_gesture_swipe_v1.end.
//
// Flutter: no counterpart
func (s *Server) SwipeEnd(cancelled bool) {
	c := int32(0)
	if cancelled {
		c = 1
	}
	s.Send(s.ID("zwp_pointer_gesture_swipe_v1"), 2, s.nextSerial(), uint32(0), c)
}
