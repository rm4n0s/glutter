package gamepadhid

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// Bridge is the Handler that a platform backend gives a Device: it makes the pad the app sees of what the driver finds. When the
// driver is ready the pad is announced (platform.PadHandler.PadAdded) as a Standard pad, the snapshots of the buttons and axes
// become events, and the touchpad, the sensors and the battery are passed to the handler if it takes them
// (platform.PadExtrasHandler). Everything reaches the handler through Deliver, which runs a function on the UI thread; the
// backend decides what to do when the service has stopped by then.
//
// Flutter: no counterpart
type Bridge struct {
	// ID is the pad's number, which the backend made.
	ID platform.PadID
	// Info is the device that the driver serves.
	Info DeviceInfo
	// Deliver runs fn on the UI thread with the handler of the service, unless the service has been stopped.
	Deliver func(fn func(h platform.PadHandler))
	// OnReady is called, on the goroutine of the device, once the pad has been announced; OnFailed when the driver could not
	// start (the pad was not announced) and OnLost when the pad was announced and is gone (the removal has been delivered). They
	// are for the backend's own bookkeeping and may be nil.
	OnReady  func(caps Caps)
	OnFailed func(err error)
	OnLost   func()

	mu      sync.Mutex
	tracker *padstd.Tracker
	caps    Caps
	ready   bool
	removed atomic.Bool
}

var _ Handler = (*Bridge)(nil)

// Caps returns what the driver said when it was ready, and whether it has been.
func (b *Bridge) Caps() (Caps, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.caps, b.ready
}

// PadInfo makes the description of the pad of what the driver said.
func (b *Bridge) PadInfo(caps Caps, path string) platform.PadInfo {
	name := caps.Name
	if name == "" {
		name = gamepaddb.DeviceName(b.Info.Vendor, b.Info.Product, "", b.Info.Name)
	}
	return platform.PadInfo{
		ID:             b.ID,
		Name:           name,
		GUID:           GUID(b.Info, caps),
		Path:           path,
		Standard:       true,
		Rumble:         caps.Rumble,
		RumbleTriggers: caps.RumbleTriggers,
		LED:            caps.LED,
		Battery:        -1,
		ButtonMask:     caps.Buttons,
		Touchpads:      caps.Touchpads,
		Sensors:        caps.Sensors,
		SensorRate:     caps.Rate,
	}
}

// Ready implements Handler.
func (b *Bridge) Ready(caps Caps) {
	b.mu.Lock()
	b.caps, b.ready = caps, true
	b.tracker = padstd.NewTracker(b.ID)
	b.mu.Unlock()
	info := b.PadInfo(caps, b.Info.Path)
	b.Deliver(func(h platform.PadHandler) { h.PadAdded(info) })
	if b.OnReady != nil {
		b.OnReady(caps)
	}
}

// Failed implements Handler.
func (b *Bridge) Failed(err error) {
	if b.OnFailed != nil {
		b.OnFailed(err)
	}
}

// State implements Handler.
func (b *Bridge) State(now time.Duration, s *padstd.Snapshot) {
	b.mu.Lock()
	events := b.tracker.Update(now, s, nil)
	b.mu.Unlock()
	if len(events) > 0 {
		b.Deliver(func(h platform.PadHandler) { h.PadEvents(events) })
	}
}

// Touch implements Handler.
func (b *Bridge) Touch(t platform.PadTouch) {
	t.Pad = b.ID
	b.Deliver(func(h platform.PadHandler) {
		if x, ok := h.(platform.PadExtrasHandler); ok {
			x.PadTouch(t)
		}
	})
}

// Sensor implements Handler.
func (b *Bridge) Sensor(s platform.PadSensor) {
	s.Pad = b.ID
	b.Deliver(func(h platform.PadHandler) {
		if x, ok := h.(platform.PadExtrasHandler); ok {
			x.PadSensor(s)
		}
	})
}

// Battery implements Handler.
func (b *Bridge) Battery(p platform.PadBattery) {
	p.Pad = b.ID
	b.Deliver(func(h platform.PadHandler) {
		if x, ok := h.(platform.PadExtrasHandler); ok {
			x.PadBattery(p)
		}
	})
}

// Lost implements Handler: the pad is removed.
func (b *Bridge) Lost() {
	b.remove()
	if b.OnLost != nil {
		b.OnLost()
	}
}

// Remove delivers the removal of the pad when the backend ends the device itself (the node of the pad went away): call it after
// Device.Close has returned. The removal is delivered once, whether Lost or Remove comes first, and only for a pad that was
// announced.
func (b *Bridge) Remove() { b.remove() }

func (b *Bridge) remove() {
	b.mu.Lock()
	ready := b.ready
	b.mu.Unlock()
	if !ready || !b.removed.CompareAndSwap(false, true) {
		return
	}
	id := b.ID
	b.Deliver(func(h platform.PadHandler) { h.PadRemoved(id) })
}
