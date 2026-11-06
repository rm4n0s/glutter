package gamepadhid

import (
	"sync"
	"time"
)

// outbox is the queue of output reports of a device, which SDL's rumble thread is (src/joystick/hidapi/SDL_hidapi_rumble.c,
// translated to Go and changed): the reports are written on a goroutine of their own so that a write that takes its time does not
// hold up the app, a report that has not been written yet is replaced by a newer one of the same kind (a rumble that changes
// ten times in a frame is written once), and a pause after every write keeps the reads from starving.
type outbox struct {
	t   Transport
	gap time.Duration

	mu      sync.Mutex
	pending [][]byte
	wake    chan struct{}
	closed  bool
	done    chan struct{}
	started bool
}

func newOutbox(t Transport, gap time.Duration) *outbox {
	return &outbox{t: t, gap: gap, wake: make(chan struct{}, 1), done: make(chan struct{})}
}

// send queues a report. merge says whether a queued report is replaced by this one; nil means the same length and report ID.
func (o *outbox) send(data []byte, merge func(pending []byte) bool) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return nil
	}
	for _, p := range o.pending {
		if len(p) != len(data) {
			continue
		}
		same := false
		if merge != nil {
			same = merge(p)
		} else {
			same = len(data) > 0 && p[0] == data[0]
		}
		if same {
			copy(p, data)
			return nil
		}
	}
	o.pending = append(o.pending, append([]byte(nil), data...))
	if !o.started {
		o.started = true
		go o.run()
	}
	select {
	case o.wake <- struct{}{}:
	default:
	}
	return nil
}

// run writes the queued reports, oldest first.
func (o *outbox) run() {
	defer close(o.done)
	for {
		o.mu.Lock()
		if o.closed {
			o.mu.Unlock()
			return
		}
		var next []byte
		if len(o.pending) > 0 {
			next = o.pending[0]
			o.pending = o.pending[1:]
		}
		o.mu.Unlock()
		if next == nil {
			<-o.wake
			continue
		}
		// A report that cannot be written is lost; the next read says whether the device is gone.
		_, _ = o.t.Write(next)
		time.Sleep(o.gap)
	}
}

// close stops the goroutine after the write that is under way and drops what is queued. It may be called more than once.
func (o *outbox) close() {
	o.mu.Lock()
	if o.closed {
		started := o.started
		o.mu.Unlock()
		if started {
			<-o.done
		}
		return
	}
	o.closed = true
	o.pending = nil
	started := o.started
	o.mu.Unlock()
	if started {
		select {
		case o.wake <- struct{}{}:
		default:
		}
		<-o.done
	}
}
