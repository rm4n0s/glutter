package webgpu

import "sync"

// failure is why a device could not be had: the tag says which of the errors it becomes.
type failure struct {
	tag, message string
}

// supply hands out the devices of a Provider. A device is asked for from the browser without
// waiting (start), and arrives in a callback (deliver); take gives it to the renderer that opens
// it, or says that it is still coming, or why it failed. The first device is awaited by Acquire
// before the event listeners; one that follows the loss of a device is not awaited by anyone.
type supply struct {
	mu      sync.Mutex
	request func()
	ready   any
	failed  *failure
	pending bool
}

// start asks for a device unless one is on its way or waiting.
func (s *supply) start() {
	s.mu.Lock()
	if s.pending || s.ready != nil {
		s.mu.Unlock()
		return
	}
	s.pending, s.failed = true, nil
	request := s.request
	s.mu.Unlock()
	request()
}

// deliver is the answer to start: a device, or the reason there is none.
func (s *supply) deliver(device any, f *failure) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = false
	s.ready, s.failed = device, f
}

// state is where the supply stands.
type supplyState int

const (
	supplyPending supplyState = iota // a device is on its way
	supplyReady                      // a device is waiting for take
	supplyFailed                     // the last request failed
	supplyEmpty                      // nothing was asked for
)

// peek says where the supply stands without changing it.
func (s *supply) peek() (supplyState, *failure) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.ready != nil:
		return supplyReady, nil
	case s.pending:
		return supplyPending, nil
	case s.failed != nil:
		return supplyFailed, s.failed
	}
	return supplyEmpty, nil
}

// take gives away the device that is waiting. In any other state it returns nil and the state;
// an empty supply starts a request, so the answer is then supplyPending.
func (s *supply) take() (any, supplyState, *failure) {
	s.mu.Lock()
	if s.ready != nil {
		d := s.ready
		s.ready = nil
		s.mu.Unlock()
		return d, supplyReady, nil
	}
	switch {
	case s.pending:
		s.mu.Unlock()
		return nil, supplyPending, nil
	case s.failed != nil:
		f := s.failed
		s.mu.Unlock()
		return nil, supplyFailed, f
	}
	s.mu.Unlock()
	s.start()
	return nil, supplyPending, nil
}

// release takes away the device that is waiting, which nobody opened, and returns it; nil when none
// is. The provider destroys it when it is closed.
func (s *supply) release() any {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.ready
	s.ready = nil
	return d
}
