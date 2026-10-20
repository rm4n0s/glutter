package fakecomp

import (
	"fmt"
	"io"
	"os"
	"sort"
	"syscall"
	"time"
)

// clientSource is a data source (wl_data_source or
// zwp_primary_selection_source_v1) that the client created.
type clientSource struct {
	id      uint32
	primary bool
	mimes   []string
}

// sendOp and cancelOp are the opcodes of the events send and cancelled: a
// wl_data_source has target before them.
func (c *clientSource) sendOp() uint16 {
	if c.primary {
		return 0
	}
	return 1
}

func (c *clientSource) cancelOp() uint16 {
	if c.primary {
		return 1
	}
	return 2
}

// handleSelection handles the requests of the data device and of the primary
// selection device. It reports whether it did.
func (s *Server) handleSelection(id uint32, iface string, op uint16, r *reader) bool {
	switch iface + "." + fmt.Sprint(op) {
	case "wl_data_device_manager.0": // create_data_source(id)
		src := r.u32()
		s.addObject(src, "wl_data_source")
		s.newClientSource(src, false)
		s.record("wl_data_device_manager.create_data_source")
	case "wl_data_source.0": // offer(mime_type)
		mime := r.str()
		s.addMime(id, mime)
		s.record("wl_data_source.offer mime=%s", mime)
	case "wl_data_source.1": // destroy
		s.dropSource(id)
		s.record("wl_data_source.destroy")
	case "wl_data_device.1": // set_selection(source, serial)
		src, serial := r.u32(), r.u32()
		s.setClientSelection(src, false)
		s.record("wl_data_device.set_selection source=%d serial=%d", src, serial)
	case "zwp_primary_selection_device_manager_v1.0": // create_source(id)
		src := r.u32()
		s.addObject(src, "zwp_primary_selection_source_v1")
		s.newClientSource(src, true)
		s.record("zwp_primary_selection_device_manager_v1.create_source")
	case "zwp_primary_selection_device_manager_v1.1": // get_device(id, seat)
		dev, seat := r.u32(), r.u32()
		s.addObject(dev, "zwp_primary_selection_device_v1")
		s.record("zwp_primary_selection_device_manager_v1.get_device seat=%d", seat)
		s.mu.Lock()
		data := s.primaryData
		s.mu.Unlock()
		if len(data) > 0 {
			s.announcePrimary(dev, data)
		}
	case "zwp_primary_selection_device_manager_v1.2": // destroy
		s.record("zwp_primary_selection_device_manager_v1.destroy")
	case "zwp_primary_selection_source_v1.0": // offer(mime_type)
		mime := r.str()
		s.addMime(id, mime)
		s.record("zwp_primary_selection_source_v1.offer mime=%s", mime)
	case "zwp_primary_selection_source_v1.1": // destroy
		s.dropSource(id)
		s.record("zwp_primary_selection_source_v1.destroy")
	case "zwp_primary_selection_device_v1.0": // set_selection(source, serial)
		src, serial := r.u32(), r.u32()
		s.setClientSelection(src, true)
		s.record("zwp_primary_selection_device_v1.set_selection source=%d serial=%d", src, serial)
	case "zwp_primary_selection_device_v1.1": // destroy
		s.record("zwp_primary_selection_device_v1.destroy")
	case "zwp_primary_selection_offer_v1.0": // receive(mime_type, fd)
		mime, fd := r.str(), r.fd()
		s.record("zwp_primary_selection_offer_v1.receive mime=%s fd=%v", mime, fd >= 0)
		s.mu.Lock()
		data, ok := s.primaryData[mime]
		s.mu.Unlock()
		s.writeSelection(fd, data, ok)
	case "zwp_primary_selection_offer_v1.1": // destroy
		s.record("zwp_primary_selection_offer_v1.destroy")
	default:
		return false
	}
	return true
}

func (s *Server) newClientSource(id uint32, primary bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sources == nil {
		s.sources = map[uint32]*clientSource{}
	}
	s.sources[id] = &clientSource{id: id, primary: primary}
}

func (s *Server) addMime(id uint32, mime string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if src := s.sources[id]; src != nil {
		src.mimes = append(src.mimes, mime)
	}
}

func (s *Server) dropSource(id uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.sources[id]
	delete(s.sources, id)
	if s.clientSel == src {
		s.clientSel = nil
	}
	if s.clientPrimary == src {
		s.clientPrimary = nil
	}
}

// setClientSelection records the source that the client made the selection; the
// source it replaces is cancelled, as a compositor does.
func (s *Server) setClientSelection(id uint32, primary bool) {
	s.mu.Lock()
	src := s.sources[id]
	slot := &s.clientSel
	if primary {
		slot = &s.clientPrimary
	}
	old := *slot
	*slot = src
	s.mu.Unlock()
	if old != nil && old != src {
		s.Send(old.id, old.cancelOp()) // cancelled
	}
}

// writeSelection answers a receive request of the client with the data, or
// closes the pipe when there is none.
func (s *Server) writeSelection(fd int, data []byte, ok bool) {
	switch {
	case fd < 0:
	case s.stall:
		s.stalled = append(s.stalled, fd) // closed when the server stops
	case ok:
		// The other application writes from its own goroutine.
		go func(fd int) {
			f := os.NewFile(uintptr(fd), "selection")
			_, _ = f.Write(data)
			_ = f.Close()
		}(fd)
	default:
		_ = syscall.Close(fd)
	}
}

func sortedMimes(data map[string][]byte) []string {
	mimes := make([]string, 0, len(data))
	for m := range data {
		mimes = append(mimes, m)
	}
	sort.Strings(mimes)
	return mimes
}

func (s *Server) announcePrimary(dev uint32, data map[string][]byte) {
	offer := s.newServerID()
	s.addObject(offer, "zwp_primary_selection_offer_v1")
	s.Send(dev, 0, offer) // data_offer(id)
	for _, m := range sortedMimes(data) {
		s.Send(offer, 0, m) // offer(mime_type)
	}
	s.Send(dev, 1, offer) // selection(id)
}

// OfferSelectionData makes another application the owner of the clipboard, with
// the data in each of the formats: the data device gets a new offer in the
// formats and a selection event, and a receive request for a format gets its data.
//
// Flutter: no counterpart
func (s *Server) OfferSelectionData(data map[string][]byte) {
	s.mu.Lock()
	s.data = data
	s.selection = "x" // so that the first bound data device sees a selection
	s.mu.Unlock()
	offer := s.newServerID()
	s.addObject(offer, "wl_data_offer")
	dd := s.ID("wl_data_device")
	s.Send(dd, 0, offer) // data_offer(id)
	for _, m := range sortedMimes(data) {
		s.Send(offer, 0, m) // offer(mime_type)
	}
	s.Send(dd, 5, offer) // selection(id)
}

// OfferPrimary makes another application the owner of the primary selection,
// with the data in each of the formats.
//
// Flutter: no counterpart
func (s *Server) OfferPrimary(data map[string][]byte) {
	s.mu.Lock()
	s.primaryData = data
	s.mu.Unlock()
	s.announcePrimary(s.ID("zwp_primary_selection_device_v1"), data)
}

// ClearPrimary empties the primary selection: the device announces none.
//
// Flutter: no counterpart
func (s *Server) ClearPrimary() {
	s.mu.Lock()
	s.primaryData = nil
	s.mu.Unlock()
	s.Send(s.ID("zwp_primary_selection_device_v1"), 1, uint32(0)) // selection(null)
}

// ClientSelectionMimes are the formats of the selection that the client owns
// (the primary selection if primary), or nil when it owns none.
//
// Flutter: no counterpart
func (s *Server) ClientSelectionMimes(primary bool) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.clientSel
	if primary {
		src = s.clientPrimary
	}
	if src == nil {
		return nil
	}
	return append([]string(nil), src.mimes...)
}

// ReadClientSelection reads the selection that the client owns (the primary
// selection if primary) in the format, as another application would: it sends the
// source a send event with a pipe and reads what the client writes.
//
// Flutter: no counterpart
func (s *Server) ReadClientSelection(primary bool, mime string) ([]byte, error) {
	s.mu.Lock()
	src := s.clientSel
	if primary {
		src = s.clientPrimary
	}
	s.mu.Unlock()
	if src == nil {
		return nil, fmt.Errorf("the client owns no selection")
	}
	return s.readSource(src, mime)
}

// readSource sends the source a send event with a pipe and reads what the client
// writes.
func (s *Server) readSource(src *clientSource, mime string) ([]byte, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	s.Send(src.id, src.sendOp(), mime, FD(w.Fd())) // send(mime_type, fd)
	if err := w.Close(); err != nil {
		return nil, err
	}
	_ = r.SetReadDeadline(time.Now().Add(5 * time.Second))
	return io.ReadAll(r)
}

// CancelClientSelection tells the client that another application took the
// selection it owns (the primary selection if primary).
//
// Flutter: no counterpart
func (s *Server) CancelClientSelection(primary bool) {
	s.mu.Lock()
	src := s.clientSel
	if primary {
		src = s.clientPrimary
	}
	if primary {
		s.clientPrimary = nil
	} else {
		s.clientSel = nil
	}
	s.mu.Unlock()
	if src != nil {
		s.Send(src.id, src.cancelOp()) // cancelled
	}
}
