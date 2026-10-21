//go:build linux

package fakecomp

import (
	"fmt"
	"time"
)

// The drag and drop of the fake compositor: another application drags over the
// window (DragEnter, DragMotion, DragLeave, DragDrop), and the client starts drags
// of its own (the requests are recorded; DragSource* play the compositor's and the
// target's part).

// dndOffer is a drag from another application over the window.
type dndOffer struct {
	id   uint32
	data map[string][]byte
}

// DND action masks of wl_data_device_manager.
//
// Flutter: no counterpart
const (
	DndCopy = 1
	DndMove = 2
	DndAsk  = 4
)

// DragEnter starts a drag of another application over the newest toplevel's surface
// at (x, y) in logical pixels. The source offers the data in the formats and allows
// the actions (DndCopy | DndMove | DndAsk).
//
// Flutter: no counterpart
func (s *Server) DragEnter(actions uint32, data map[string][]byte, x, y float64) {
	offer := s.newServerID()
	s.addObject(offer, "wl_data_offer")
	s.mu.Lock()
	s.dnd = &dndOffer{id: offer, data: data}
	s.dndSerial++
	serial := 100 + s.dndSerial
	s.mu.Unlock()
	dd := s.ID("wl_data_device")
	s.Send(dd, 0, offer) // data_offer(id)
	for _, m := range sortedMimes(data) {
		s.Send(offer, 0, m) // offer(mime_type)
	}
	s.Send(offer, 1, actions)                                           // source_actions(actions)
	s.Send(dd, 1, serial, s.WindowSurface(), Fixed(x), Fixed(y), offer) // enter(serial, surface, x, y, id)
}

// DragMotion moves the drag.
//
// Flutter: no counterpart
func (s *Server) DragMotion(x, y float64) {
	s.Send(s.ID("wl_data_device"), 3, uint32(time.Now().UnixMilli()), Fixed(x), Fixed(y)) // motion(time, x, y)
}

// DragLeave ends the drag without a drop.
//
// Flutter: no counterpart
func (s *Server) DragLeave() {
	s.Send(s.ID("wl_data_device"), 2) // leave
}

// DragDrop drops the drag; the client reads the data and finishes.
//
// Flutter: no counterpart
func (s *Server) DragDrop() {
	s.Send(s.ID("wl_data_device"), 4) // drop
}

// handleDnd handles the requests about drags. It reports whether it did.
func (s *Server) handleDnd(id uint32, iface string, op uint16, r *reader) bool {
	switch iface + "." + fmt.Sprint(op) {
	case "wl_data_device.0": // start_drag(source, origin, icon, serial)
		src, origin, icon, serial := r.u32(), r.u32(), r.u32(), r.u32()
		s.mu.Lock()
		s.dragSource = s.sources[src]
		s.mu.Unlock()
		s.record("wl_data_device.start_drag source=%d origin=%d icon=%v serial=%d", src, origin, icon != 0, serial)
	case "wl_data_source.2": // set_actions(dnd_actions)
		s.record("wl_data_source.set_actions actions=%d", r.u32())
	case "wl_data_offer.3": // finish
		s.record("wl_data_offer.finish")
	case "wl_data_offer.4": // set_actions(dnd_actions, preferred_action)
		a, p := r.u32(), r.u32()
		s.record("wl_data_offer.set_actions actions=%d preferred=%d", a, p)
	default:
		return false
	}
	return true
}

// dndData is the data the drag over the window offers for the format, if the offer
// is the drag's.
func (s *Server) dndData(offer uint32, mime string) (data []byte, ok, isDnd bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dnd == nil || s.dnd.id != offer {
		return nil, false, false
	}
	d, ok := s.dnd.data[mime]
	return d, ok, true
}

// DragSourceMimes are the formats of the drag that the client started.
//
// Flutter: no counterpart
func (s *Server) DragSourceMimes() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dragSource == nil {
		return nil
	}
	return append([]string(nil), s.dragSource.mimes...)
}

// ReadDragSource reads the data of the drag that the client started, as the target
// would: it sends the source a send event with a pipe and reads what the client
// writes.
//
// Flutter: no counterpart
func (s *Server) ReadDragSource(mime string) ([]byte, error) {
	s.mu.Lock()
	src := s.dragSource
	s.mu.Unlock()
	if src == nil {
		return nil, fmt.Errorf("the client started no drag")
	}
	return s.readSource(src, mime)
}

// DragSourceAction tells the source the action the target chose.
//
// Flutter: no counterpart
func (s *Server) DragSourceAction(action uint32) { s.sendToDragSource(5, action) }

// DragSourceDropPerformed tells the source that the drop happened.
//
// Flutter: no counterpart
func (s *Server) DragSourceDropPerformed() { s.sendToDragSource(3) }

// DragSourceFinished tells the source that the target is done with the data.
//
// Flutter: no counterpart
func (s *Server) DragSourceFinished() { s.sendToDragSource(4) }

// DragSourceCancelled tells the source that the drag ended without a drop.
//
// Flutter: no counterpart
func (s *Server) DragSourceCancelled() { s.sendToDragSource(2) }

func (s *Server) sendToDragSource(op uint16, args ...any) {
	s.mu.Lock()
	src := s.dragSource
	s.mu.Unlock()
	if src == nil {
		s.t.Errorf("fake server: no drag source to send to")
		return
	}
	s.Send(src.id, op, args...)
}
