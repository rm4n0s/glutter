package gamepadhid

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
)

// Handler is told by a Device what its driver finds. Its methods run on the goroutine of the device, so they must not wait for
// the UI thread and must not call the Device back; the platform's handlers post what they get to the UI thread.
//
// Flutter: no counterpart
type Handler interface {
	// Ready says that the driver has started and the pad can be announced.
	Ready(caps Caps)
	// Failed says that the driver could not start. The transport is closed; the platform may read the pad another way.
	Failed(err error)
	// State gives the buttons and axes of the layout after a report.
	State(now time.Duration, s *padstd.Snapshot)
	// Touch, Sensor and Battery give what the pad reports besides.
	Touch(t platform.PadTouch)
	Sensor(s platform.PadSensor)
	Battery(b platform.PadBattery)
	// Lost says that the pad is gone, which ends the device (the transport is closed). It is never called after Close.
	Lost()
}

// Options are the options of a Device.
//
// Flutter: no counterpart
type Options struct {
	// Clock is the monotonic clock that stamps the reports; the default is the time since the device was started.
	Clock func() time.Duration
	// Logger receives the notes of the device; nil logs nothing.
	Logger *slog.Logger
	// ReadTimeout is how long a read waits before the driver's Tick runs; 8 ms when 0.
	ReadTimeout time.Duration
	// WriteGap is the pause after each queued output report; 10 ms when 0 (the rumble thread of SDL waits that long so that
	// the writes do not starve the reads).
	WriteGap time.Duration
}

// Device runs a driver on a goroutine of its own over a transport.
//
// Flutter: no counterpart
type Device struct {
	info  DeviceInfo
	t     Transport
	h     Handler
	drv   Driver
	log   *slog.Logger
	clock func() time.Duration
	out   *outbox
	read  time.Duration

	stop     chan struct{}
	stopOnce sync.Once
	done     chan struct{}
	closing  atomic.Bool
	// lost is set by the driver, through its sink, when the pad went and the link did not.
	lost atomic.Bool

	// mu guards the driver and the rest below: the goroutine of the device holds it while it uses the driver, and so do the
	// calls that come from the UI thread.
	mu     sync.Mutex
	caps   Caps
	ready  bool
	dead   bool
	rumble rumbleState
}

// rumbleState is the rumble that the app asked for, which the device stops when its time is over and sends again now and then
// (some pads stop on their own after a while).
type rumbleState struct {
	low, high, left, right uint16
	motorsUntil            time.Duration
	triggersUntil          time.Duration
	resendAt               time.Duration
}

// rumbleResend is how often a long rumble is sent again (SDL does so every two seconds).
const rumbleResend = 2 * time.Second

// Start finds the driver of the device and starts it on a goroutine. It returns nil and false, and keeps the transport open,
// when no driver serves the device. Otherwise the transport belongs to the Device, and h is told Ready (or Failed) and then
// what the pad does.
//
// Flutter: no counterpart
func Start(info DeviceInfo, t Transport, h Handler, opts Options) (*Device, bool) {
	m := ModelOf(info)
	if m == ModelNone {
		return nil, false
	}
	return startWith(newDriver(m, info), info, t, h, opts), true
}

// startWith starts a device with the given driver.
func startWith(drv Driver, info DeviceInfo, t Transport, h Handler, opts Options) *Device {
	d := &Device{info: info, t: t, h: h, drv: drv, log: opts.Logger, clock: opts.Clock, read: opts.ReadTimeout, stop: make(chan struct{}), done: make(chan struct{})}
	if d.log == nil {
		d.log = slog.New(slog.DiscardHandler)
	}
	if d.clock == nil {
		begin := time.Now()
		d.clock = func() time.Duration { return time.Since(begin) }
	}
	if d.read <= 0 {
		d.read = 8 * time.Millisecond
	}
	gap := opts.WriteGap
	if gap <= 0 {
		gap = 10 * time.Millisecond
	}
	d.out = newOutbox(t, gap)
	go d.run()
	return d
}

// Info is the device that the driver serves.
func (d *Device) Info() DeviceInfo { return d.info }

// Close stops the device, closes the transport and waits for the goroutine. A driver that changed the pad's mode puts it back.
// Handler.Lost is not called after Close is called.
func (d *Device) Close() {
	d.closing.Store(true)
	d.stopOnce.Do(func() { close(d.stop) })
	<-d.done
}

// run is the goroutine of the device.
func (d *Device) run() {
	defer close(d.done)
	defer d.out.close()
	d.mu.Lock()
	caps, err := d.drv.Start(devIO{d})
	if err == nil {
		d.caps, d.ready = caps, true
	}
	d.mu.Unlock()
	if err != nil {
		d.mu.Lock()
		d.dead = true
		d.mu.Unlock()
		d.out.close()
		d.closeTransport()
		if !d.closing.Load() {
			d.h.Failed(err)
		}
		return
	}
	if d.closing.Load() {
		d.shutdown()
		return
	}
	d.h.Ready(caps)
	buf := make([]byte, 512)
	for {
		select {
		case <-d.stop:
			d.shutdown()
			return
		default:
		}
		n, err := d.t.Read(buf, d.read)
		now := d.clock()
		d.mu.Lock()
		if err == nil {
			if n > 0 {
				d.drv.Report(now, buf[:n], sinkOf{d})
			}
			d.drv.Tick(now, sinkOf{d})
			d.rumbleTick(now)
		}
		d.mu.Unlock()
		if err != nil {
			d.shutdown()
			if !d.closing.Load() {
				d.h.Lost()
			}
			return
		}
		if d.lost.Load() {
			d.shutdown()
			if !d.closing.Load() {
				d.h.Lost()
			}
			return
		}
	}
}

// shutdown puts the pad back as it was and closes the transport.
func (d *Device) shutdown() {
	d.mu.Lock()
	d.drv.Close()
	d.dead = true
	d.mu.Unlock()
	d.out.close()
	d.closeTransport()
}

func (d *Device) closeTransport() {
	if err := d.t.Close(); err != nil {
		d.log.Debug("a HID device did not close cleanly", "device", d.info.Path, "err", err)
	}
}

// rumbleTick stops the rumble whose time is over and sends the rumble that goes on again. The caller holds mu.
func (d *Device) rumbleTick(now time.Duration) {
	r := &d.rumble
	if r.motorsUntil != 0 && now >= r.motorsUntil {
		r.low, r.high, r.motorsUntil = 0, 0, 0
		d.logErr("a rumble could not be stopped", d.drv.Rumble(0, 0))
	}
	if r.triggersUntil != 0 && now >= r.triggersUntil {
		r.left, r.right, r.triggersUntil = 0, 0, 0
		d.logErr("a trigger rumble could not be stopped", d.drv.RumbleTriggers(0, 0))
	}
	if r.resendAt != 0 && now >= r.resendAt {
		r.resendAt = now + rumbleResend
		if r.low != 0 || r.high != 0 {
			d.logErr("a rumble could not be sent again", d.drv.Rumble(r.low, r.high))
		}
		if r.left != 0 || r.right != 0 {
			d.logErr("a trigger rumble could not be sent again", d.drv.RumbleTriggers(r.left, r.right))
		}
		if r.low == 0 && r.high == 0 && r.left == 0 && r.right == 0 {
			r.resendAt = 0
		}
	}
}

func (d *Device) logErr(msg string, err error) {
	if err != nil {
		d.log.Debug(msg, "device", d.info.Path, "err", err)
	}
}

// Rumble turns the motors for d (0 stops them) at the strengths from 0 to 65535. The errors are the driver's: a pad without
// motors fails with GamepadRumbleUnavailable.
func (d *Device) Rumble(low, high uint16, dur time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.ready || d.dead {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad is not there", "device", d.info.Path)
	}
	if dur <= 0 {
		low, high = 0, 0
	}
	if err := d.drv.Rumble(low, high); err != nil {
		return err
	}
	now := d.clock()
	r := &d.rumble
	r.low, r.high = low, high
	if low == 0 && high == 0 {
		r.motorsUntil = 0
	} else {
		r.motorsUntil = now + dur
		r.resendAt = now + rumbleResend
	}
	return nil
}

// RumbleTriggers does the same for the motors of the triggers.
func (d *Device) RumbleTriggers(left, right uint16, dur time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.ready || d.dead {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad is not there", "device", d.info.Path)
	}
	if dur <= 0 {
		left, right = 0, 0
	}
	if err := d.drv.RumbleTriggers(left, right); err != nil {
		return err
	}
	now := d.clock()
	r := &d.rumble
	r.left, r.right = left, right
	if left == 0 && right == 0 {
		r.triggersUntil = 0
	} else {
		r.triggersUntil = now + dur
		r.resendAt = now + rumbleResend
	}
	return nil
}

// SetLED sets the colour of the light.
func (d *Device) SetLED(r, g, b uint8) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.ready || d.dead {
		return errors.New(platform.TagGamepadLEDUnavailable, "the pad is not there", "device", d.info.Path)
	}
	return d.drv.SetLED(r, g, b)
}

// SetSensors turns the motion sensors on or off.
func (d *Device) SetSensors(on bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.ready || d.dead {
		return errors.New(platform.TagGamepadSensorsUnavailable, "the pad is not there", "device", d.info.Path)
	}
	return d.drv.SetSensors(on)
}

// sinkOf is the Sink that a driver gets: it passes what the driver reads on to the handler.
type sinkOf struct{ d *Device }

func (s sinkOf) State(now time.Duration, snap *padstd.Snapshot) { s.d.h.State(now, snap) }
func (s sinkOf) Touch(now time.Duration, t platform.PadTouch)   { t.Time = now; s.d.h.Touch(t) }
func (s sinkOf) Sensor(now time.Duration, v platform.PadSensor) { v.Time = now; s.d.h.Sensor(v) }
func (s sinkOf) Battery(b platform.PadBattery)                  { s.d.h.Battery(b) }
func (s sinkOf) Lost()                                          { s.d.lost.Store(true) }

// devIO is the IO that a driver gets.
type devIO struct{ d *Device }

func (i devIO) Read(buf []byte, timeout time.Duration) (int, error) { return i.d.t.Read(buf, timeout) }
func (i devIO) Write(data []byte) (int, error)                      { return i.d.t.Write(data) }
func (i devIO) Send(data []byte) error                              { return i.d.out.send(data, nil) }
func (i devIO) SendMerging(data []byte, merge func([]byte) bool) error {
	return i.d.out.send(data, merge)
}
func (i devIO) Feature(id byte, buf []byte) (int, error) {
	clear(buf)
	buf[0] = id
	return i.d.t.GetFeature(buf)
}
func (i devIO) SendFeature(data []byte) (int, error) { return i.d.t.SendFeature(data) }
func (i devIO) Now() time.Duration                   { return i.d.clock() }
