package webgpu

import (
	"sync"

	"github.com/rm4n0s/errors"
)

// lossState is whether a device was lost or closed, and why. The browser tells about a loss in a
// callback, which may run while the UI goroutine is between two calls, so it is guarded.
type lossState struct {
	mu      sync.Mutex
	closed  bool
	lost    bool
	reason  string
	message string
}

// markClosed records that the owner destroyed the device: the loss that destroy() reports is not
// one. It returns false when the device was closed already.
func (s *lossState) markClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.closed = true
	return true
}

// markLost records a loss the browser reported. It returns false when the owner closed the device
// (a closed device always reports a loss with the reason "destroyed") or the loss was recorded
// already, so that the caller starts a new device only for a loss that is news.
func (s *lossState) markLost(reason, message string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.lost {
		return false
	}
	s.lost, s.reason, s.message = true, reason, message
	return true
}

// err is the error to return from a call on the device: the loss, or nil while it is usable.
func (s *lossState) err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lost {
		return errors.New(TagDeviceLost, "the WebGPU device was lost", "reason", s.reason, "message", s.message)
	}
	return nil
}

// isLost says whether the device was lost.
func (s *lossState) isLost() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lost
}
