package fakecomp

import "fmt"

// tabletToolEvents maps the names that ToolSend takes to the opcodes of the events
// of zwp_tablet_tool_v2.
var tabletToolEvents = map[string]uint16{
	"type": 0, "hardware_serial": 1, "hardware_id_wacom": 2, "capability": 3, "done": 4, "removed": 5,
	"proximity_in": 6, "proximity_out": 7, "down": 8, "up": 9, "motion": 10, "pressure": 11, "distance": 12,
	"tilt": 13, "rotation": 14, "slider": 15, "wheel": 16, "button": 17, "frame": 18,
}

// The types of a tablet tool (zwp_tablet_tool_v2.type), for AddTabletTool.
const (
	// ToolPen is a pen.
	//
	// Flutter: no counterpart
	ToolPen uint32 = 0x140
	// ToolEraser is the eraser end of a pen.
	//
	// Flutter: no counterpart
	ToolEraser uint32 = 0x141
	// ToolFinger is a finger on the tablet.
	//
	// Flutter: no counterpart
	ToolFinger uint32 = 0x145
	// ToolMouse is a mouse on the tablet.
	//
	// Flutter: no counterpart
	ToolMouse uint32 = 0x146
)

// extras is the state of the protocols of step 14.7.
type extras struct {
	tokens int    // the activation tokens given so far
	tablet uint32 // the zwp_tablet_v2 object, 0 until the first tool
}

// bindExtras sends what a compositor sends when a client binds a global of step
// 14.7.
func (s *Server) bindExtras(iface string, id uint32) {
	if iface == "xdg_toplevel_icon_manager_v1" {
		s.Send(id, 0, int32(48)) // icon_size
		s.Send(id, 0, int32(64))
		s.Send(id, 1) // done
	}
}

// handleExtras answers the requests of the activation, idle inhibit, content type,
// toplevel icon and tablet protocols.
func (s *Server) handleExtras(iface string, op uint16, id uint32, r *reader) bool {
	switch iface + "." + fmt.Sprint(op) {
	case "xdg_activation_v1.0":
		s.record("xdg_activation_v1.destroy")
	case "xdg_activation_v1.1": // get_activation_token(id)
		tok := r.u32()
		s.addObject(tok, "xdg_activation_token_v1")
		s.record("xdg_activation_v1.get_activation_token")
	case "xdg_activation_v1.2": // activate(token, surface)
		token, surface := r.str(), r.u32()
		s.record("xdg_activation_v1.activate token=%s surface=%v", token, surface == s.WindowSurface())
	case "xdg_activation_token_v1.0": // set_serial(serial, seat)
		serial, seat := r.u32(), r.u32()
		s.record("xdg_activation_token_v1.set_serial serial=%d seat=%v", serial, seat == s.ID("wl_seat"))
	case "xdg_activation_token_v1.1": // set_app_id(app_id)
		s.record("xdg_activation_token_v1.set_app_id %s", r.str())
	case "xdg_activation_token_v1.2": // set_surface(surface)
		s.record("xdg_activation_token_v1.set_surface surface=%v", r.u32() == s.WindowSurface())
	case "xdg_activation_token_v1.3": // commit
		s.mu.Lock()
		s.extras.tokens++
		token := fmt.Sprintf("fake-token-%d", s.extras.tokens)
		s.mu.Unlock()
		s.record("xdg_activation_token_v1.commit")
		s.Send(id, 0, token) // done(token)
	case "xdg_activation_token_v1.4":
		s.record("xdg_activation_token_v1.destroy")
	case "zwp_idle_inhibit_manager_v1.0":
		s.record("zwp_idle_inhibit_manager_v1.destroy")
	case "zwp_idle_inhibit_manager_v1.1": // create_inhibitor(id, surface)
		inh, surface := r.u32(), r.u32()
		s.addObject(inh, "zwp_idle_inhibitor_v1")
		s.record("zwp_idle_inhibit_manager_v1.create_inhibitor surface=%v", surface == s.WindowSurface())
	case "zwp_idle_inhibitor_v1.0":
		s.record("zwp_idle_inhibitor_v1.destroy")
	case "wp_content_type_manager_v1.0":
		s.record("wp_content_type_manager_v1.destroy")
	case "wp_content_type_manager_v1.1": // get_surface_content_type(id, surface)
		ct, surface := r.u32(), r.u32()
		s.addObject(ct, "wp_content_type_v1")
		s.record("wp_content_type_manager_v1.get_surface_content_type surface=%v", surface == s.WindowSurface())
	case "wp_content_type_v1.0":
		s.record("wp_content_type_v1.destroy")
	case "wp_content_type_v1.1": // set_content_type(content_type)
		s.record("wp_content_type_v1.set_content_type %d", r.u32())
	case "xdg_toplevel_icon_manager_v1.0":
		s.record("xdg_toplevel_icon_manager_v1.destroy")
	case "xdg_toplevel_icon_manager_v1.1": // create_icon(id)
		icon := r.u32()
		s.addObject(icon, "xdg_toplevel_icon_v1")
		s.record("xdg_toplevel_icon_manager_v1.create_icon")
	case "xdg_toplevel_icon_manager_v1.2": // set_icon(toplevel, icon)
		toplevel, icon := r.u32(), r.u32()
		s.record("xdg_toplevel_icon_manager_v1.set_icon toplevel=%v icon=%v", toplevel != 0, icon != 0)
	case "xdg_toplevel_icon_v1.0":
		s.record("xdg_toplevel_icon_v1.destroy")
	case "xdg_toplevel_icon_v1.1": // set_name(icon_name)
		s.record("xdg_toplevel_icon_v1.set_name %s", r.str())
	case "xdg_toplevel_icon_v1.2": // add_buffer(buffer, scale)
		buffer, scale := r.u32(), r.i32()
		s.mu.Lock()
		size := s.win.bufferSize[buffer]
		s.mu.Unlock()
		s.record("xdg_toplevel_icon_v1.add_buffer %dx%d scale=%d", size[0], size[1], scale)
	case "zwp_tablet_manager_v2.0": // get_tablet_seat(tablet_seat, seat)
		seat := r.u32()
		s.addObject(seat, "zwp_tablet_seat_v2")
		s.record("zwp_tablet_manager_v2.get_tablet_seat")
	case "zwp_tablet_manager_v2.1":
		s.record("zwp_tablet_manager_v2.destroy")
	case "zwp_tablet_seat_v2.0":
		s.record("zwp_tablet_seat_v2.destroy")
	case "zwp_tablet_tool_v2.0": // set_cursor(serial, surface, hotspot_x, hotspot_y)
		serial, surface, x, y := r.u32(), r.u32(), r.i32(), r.i32()
		s.record("zwp_tablet_tool_v2.set_cursor serial=%d surface=%v hotspot=%d,%d", serial, surface != 0, x, y)
	case "zwp_tablet_tool_v2.1":
		s.record("zwp_tablet_tool_v2.destroy")
	case "zwp_tablet_v2.0":
		s.record("zwp_tablet_v2.destroy")
	default:
		return false
	}
	return true
}

// AddTabletTool announces a tablet tool of the type (ToolPen and the others), as a
// compositor does when a tool first touches a tablet: the first one also announces
// the tablet. It returns the ID of the tool, for ToolSend.
//
// Flutter: no counterpart
func (s *Server) AddTabletTool(toolType uint32) uint32 {
	seat := s.ID("zwp_tablet_seat_v2")
	s.mu.Lock()
	tablet := s.extras.tablet
	s.mu.Unlock()
	if tablet == 0 {
		tablet = s.newServerID()
		s.addObject(tablet, "zwp_tablet_v2")
		s.mu.Lock()
		s.extras.tablet = tablet
		s.mu.Unlock()
		s.Send(seat, 0, tablet) // tablet_added(id)
		s.Send(tablet, 0, "Test Tablet")
		s.Send(tablet, 3) // done
	}
	tool := s.newServerID()
	s.addObject(tool, "zwp_tablet_tool_v2")
	s.Send(seat, 1, tool) // tool_added(id)
	s.Send(tool, tabletToolEvents["type"], toolType)
	s.Send(tool, tabletToolEvents["capability"], uint32(2)) // pressure
	s.Send(tool, tabletToolEvents["done"])
	return tool
}

// ToolSend sends an event of a tablet tool: "proximity_in" (with the serial, the
// tablet and the surface; ToolProximityIn fills them in), "proximity_out", "down"
// (serial), "up", "motion" (x, y as Fixed), "pressure", "distance", "tilt",
// "rotation", "button" (serial, button, state), "frame" (time) and "removed".
//
// Flutter: no counterpart
func (s *Server) ToolSend(tool uint32, event string, args ...any) {
	op, ok := tabletToolEvents[event]
	if !ok {
		s.t.Errorf("fake server: unknown tablet tool event %q", event)
		return
	}
	s.Send(tool, op, args...)
}

// ToolProximityIn sends proximity_in for the tool over the newest window.
//
// Flutter: no counterpart
func (s *Server) ToolProximityIn(tool uint32) {
	s.mu.Lock()
	tablet := s.extras.tablet
	s.mu.Unlock()
	s.ToolSend(tool, "proximity_in", s.nextSerial(), tablet, s.WindowSurface())
}
