package iokit

import (
	"log/slog"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/gamepadhid"
	"github.com/rm4n0s/glutter/internal/platform"
)

// DeviceInfo is what a HID device says of itself (the properties of an IOHIDDevice).
//
// Flutter: no counterpart
type DeviceInfo struct {
	Vendor, Product, Version uint16
	VendorName, ProductName  string
	// Transport is how the device is connected, as IOKit says it: "USB", "Bluetooth" and the like.
	Transport string
	// UsagePage and Usage are the primary usage of the device.
	UsagePage, Usage uint32
	// Path says where the device is on the system (its location ID), for the app that wants to tell two pads of a model apart.
	Path string
}

// Device is a HID device that the manager found.
//
// Flutter: no counterpart
type Device interface {
	// Info says what the device is.
	Info() DeviceInfo
	// Elements lists the elements of the device, the ones in collections too.
	Elements() []RawElement
	// Value reads the current value of an element; ok is false when the system has none.
	Value(cookie uint32) (v int32, ok bool)
	// OpenHID opens the device for the reports of a HID driver. The transport belongs to the caller.
	OpenHID() (gamepadhid.Transport, error)
	// Release lets go of the device: the Service has no more use for it.
	Release()
}

// Callbacks are the calls that a Manager makes. They run on the UI thread (the manager is on the main run loop), never inside
// Manager.Start.
//
// Flutter: no counterpart
type Callbacks struct {
	// Added is a device that was found; the devices that are there when the manager starts are found first.
	Added func(d Device)
	// Removed is a device that has gone.
	Removed func(d Device)
	// Input is an element of a device that has a new value.
	Input func(d Device, cookie uint32, value int32)
}

// Manager finds the HID devices of the system that may be pads and reports what they do.
//
// Flutter: no counterpart
type Manager interface {
	// Start begins to report. It may be called again after Stop.
	Start(cb Callbacks) error
	// Stop ends the reports and lets go of what the manager holds; no callback is made after it returns.
	Stop()
}

// Options are the options of a Service.
//
// Flutter: no counterpart
type Options struct {
	// Manager finds the devices.
	Manager Manager
	// Post runs a function on the UI thread; it is safe to call from any goroutine.
	Post func(func())
	// Clock is the monotonic clock that stamps the events; the default is the time since the service was made.
	Clock func() time.Duration
	// Logger receives the notes of the backend; nil logs nothing.
	Logger *slog.Logger
	// DisableHID turns the HID drivers off: every pad is read through its elements.
	DisableHID bool
}

// Service is the gamepad backend of macOS (platform.Gamepads). A pad is read through its HID elements, or, when package
// gamepadhid has a driver for its model, through its HID reports, which gives the light, the touchpad, the motion sensors and
// the trigger motors that the elements do not. Its methods and the callbacks of the manager all run on the UI thread.
//
// The rumble of a pad read through its elements is the Force Feedback framework's (step 37.3, after SDL's iokit joystick): one
// sine effect that the two motors' strengths set; the pads that a HID driver serves rumble through it.
//
// Flutter: no counterpart
type Service struct {
	mgr   Manager
	post  func(func())
	clock func() time.Duration
	log   *slog.Logger
	noHID bool

	nextID platform.PadID
	cur    *run
}

// New makes the service. It does nothing until Start.
//
// Flutter: no counterpart
func New(o Options) *Service {
	s := &Service{mgr: o.Manager, post: o.Post, clock: o.Clock, log: o.Logger, noHID: o.DisableHID}
	if s.log == nil {
		s.log = slog.New(slog.DiscardHandler)
	}
	if s.clock == nil {
		begin := time.Now()
		s.clock = func() time.Duration { return time.Since(begin) }
	}
	return s
}

var _ platform.Gamepads = (*Service)(nil)

// MappingPlatform implements platform.Gamepads: the mappings of "Mac OS X".
func (s *Service) MappingPlatform() string { return "Mac OS X" }

// run is one Start of the service.
type run struct {
	s       *Service
	h       platform.PadHandler
	stopped bool
	pads    map[Device]*pad
	byID    map[platform.PadID]*pad
	// noDriver holds the devices whose HID driver failed: they are read through their elements.
	noDriver map[Device]bool
}

// pad is a device that the service reports.
type pad struct {
	id     platform.PadID
	dev    Device
	layout *Layout
	// hid is set for a pad that a HID driver serves.
	hid    *gamepadhid.Device
	bridge *gamepadhid.Bridge
	// announced says that the handler has been told about the pad.
	announced bool
	// ff is the rumble of a pad that is read through its elements and has force feedback; it is made at the first rumble.
	ff ForceFeedback
}

// Start implements platform.Gamepads.
func (s *Service) Start(h platform.PadHandler) error {
	if s.cur != nil {
		s.Stop()
	}
	r := &run{s: s, h: h, pads: map[Device]*pad{}, byID: map[platform.PadID]*pad{}, noDriver: map[Device]bool{}}
	s.cur = r
	err := s.mgr.Start(Callbacks{Added: r.added, Removed: r.removed, Input: r.input})
	if err != nil {
		s.cur = nil
		r.stopped = true
		return err
	}
	return nil
}

// Stop implements platform.Gamepads.
func (s *Service) Stop() {
	r := s.cur
	if r == nil {
		return
	}
	s.cur = nil
	r.stopped = true
	s.mgr.Stop()
	for dev, p := range r.pads {
		if p.hid != nil {
			p.hid.Close()
		}
		if p.ff != nil {
			p.ff.Close()
		}
		dev.Release()
	}
	r.pads, r.byID = nil, nil
}

// busOf is the bus of a device as the HID drivers name it.
func busOf(transport string) gamepadhid.Bus {
	switch {
	case strings.EqualFold(transport, "USB"):
		return gamepadhid.BusUSB
	case strings.EqualFold(transport, "Bluetooth"), strings.EqualFold(transport, "BluetoothLowEnergy"):
		return gamepadhid.BusBluetooth
	}
	return gamepadhid.BusUnknown
}

// added handles a device that was found.
func (r *run) added(dev Device) {
	if r.stopped {
		dev.Release()
		return
	}
	info := dev.Info()
	if !IsPad(info.UsagePage, info.Usage) || r.pads[dev] != nil {
		dev.Release()
		return
	}
	p := &pad{dev: dev}
	r.s.nextID++
	p.id = r.s.nextID
	if r.tryHID(p, info) {
		r.pads[dev], r.byID[p.id] = p, p
		return
	}
	r.addRaw(p, info)
}

// tryHID starts the HID driver of a pad if there is one, and says whether the pad is served so for now: the driver starts on a
// goroutine, and tells the service when it fails, which sends the pad back to its elements.
func (r *run) tryHID(p *pad, info DeviceInfo) bool {
	if r.s.noHID || r.noDriver[p.dev] {
		return false
	}
	hi := gamepadhid.DeviceInfo{Vendor: info.Vendor, Product: info.Product, Version: info.Version, Bus: busOf(info.Transport), Name: info.ProductName, Path: info.Path}
	if gamepadhid.ModelOf(hi) == gamepadhid.ModelNone {
		return false
	}
	tr, err := p.dev.OpenHID()
	if err != nil {
		r.s.log.Debug("a gamepad's HID reports cannot be opened", "device", info.ProductName, "err", err)
		r.noDriver[p.dev] = true
		return false
	}
	p.bridge = &gamepadhid.Bridge{
		ID: p.id, Info: hi,
		Deliver:  r.deliver,
		OnFailed: func(err error) { r.s.post(func() { r.hidFailed(p, err) }) },
		OnLost:   func() { r.s.post(func() { r.hidLost(p) }) },
	}
	dev, ok := gamepadhid.Start(hi, tr, p.bridge, gamepadhid.Options{Clock: r.s.clock, Logger: r.s.log})
	if !ok {
		if err := tr.Close(); err != nil {
			r.s.log.Debug("the HID reports of a gamepad did not close cleanly", "device", info.ProductName, "err", err)
		}
		return false
	}
	p.hid = dev
	return true
}

// deliver runs a function with the handler on the UI thread, unless the service has been stopped.
func (r *run) deliver(fn func(h platform.PadHandler)) {
	r.s.post(func() {
		if !r.stopped {
			fn(r.h)
		}
	})
}

// hidFailed handles a driver that could not start: the pad is read through its elements.
func (r *run) hidFailed(p *pad, err error) {
	if r.stopped || r.pads[p.dev] != p || p.hid == nil {
		return
	}
	p.hid.Close()
	p.hid, p.bridge = nil, nil
	r.noDriver[p.dev] = true
	r.s.log.Info("the HID driver of a gamepad could not start, so it works through its elements", "device", p.dev.Info().ProductName, "err", err)
	r.addRaw(p, p.dev.Info())
}

// hidLost handles a driver whose pad is gone (the removal has been delivered): the device is ended; the manager will say that
// the device has gone.
func (r *run) hidLost(p *pad) {
	if r.stopped || r.pads[p.dev] != p || p.hid == nil {
		return
	}
	p.hid.Close()
}

// addRaw reports a pad that is read through its elements, with the values that they have.
func (r *run) addRaw(p *pad, info DeviceInfo) {
	p.layout = BuildLayout(p.dev.Elements())
	if p.layout.Count() == 0 {
		delete(r.pads, p.dev)
		delete(r.byID, p.id)
		p.dev.Release()
		return
	}
	r.pads[p.dev], r.byID[p.id] = p, p
	bus := gamepaddb.BusUSB
	pi := platform.PadInfo{
		ID:      p.id,
		Name:    gamepaddb.DeviceName(info.Vendor, info.Product, info.VendorName, info.ProductName),
		GUID:    gamepaddb.NewGUID(bus, info.Vendor, info.Product, info.Version, info.VendorName, info.ProductName, 0, 0),
		Path:    info.Path,
		Axes:    len(p.layout.Axes),
		Buttons: len(p.layout.Buttons),
		Hats:    len(p.layout.Hats),
		Battery: -1,
	}
	if ff, ok := p.dev.(ForceFeedbacker); ok && ff.HasForceFeedback() {
		pi.Rumble = true
	}
	p.announced = true
	r.h.PadAdded(pi)
	var events []platform.PadEvent
	now := r.s.clock()
	for _, k := range []Kind{KindAxis, KindButton, KindHat} {
		for i, e := range p.layout.list(k) {
			v, ok := p.dev.Value(e.Cookie)
			if !ok {
				continue
			}
			if ev, ok := e.event(p.id, now, k, i, v); ok && (k == KindAxis || ev.Value != 0) {
				events = append(events, ev)
			}
		}
	}
	if len(events) > 0 {
		r.h.PadEvents(events)
	}
}

// list returns the elements of a kind.
func (l *Layout) list(k Kind) []*Element {
	switch k {
	case KindAxis:
		return l.Axes
	case KindButton:
		return l.Buttons
	case KindHat:
		return l.Hats
	}
	return nil
}

// event makes the event of a new value of the element, unless it is the value that it had.
func (e *Element) event(id platform.PadID, now time.Duration, k Kind, index int, v int32) (platform.PadEvent, bool) {
	ev := platform.PadEvent{Pad: id, Time: now, Index: index}
	switch k {
	case KindAxis:
		ev.Kind, ev.Value = platform.PadAxis, int(e.AxisValue(v))
	case KindButton:
		ev.Kind, ev.Value = platform.PadButton, e.ButtonValue(v)
	case KindHat:
		ev.Kind, ev.Value = platform.PadHat, e.HatValue(v)
	default:
		return ev, false
	}
	if e.has && e.last == ev.Value {
		return ev, false
	}
	e.last, e.has = ev.Value, true
	return ev, true
}

// input handles a new value of an element.
func (r *run) input(dev Device, cookie uint32, value int32) {
	p := r.pads[dev]
	if r.stopped || p == nil || p.hid != nil || p.layout == nil {
		return
	}
	pos, ok := p.layout.byCookie[cookie]
	if !ok {
		return
	}
	e := p.layout.list(pos.kind)[pos.index]
	if ev, ok := e.event(p.id, r.s.clock(), pos.kind, pos.index, value); ok {
		r.h.PadEvents([]platform.PadEvent{ev})
	}
}

// removed handles a device that has gone.
func (r *run) removed(dev Device) {
	p := r.pads[dev]
	if r.stopped || p == nil {
		return
	}
	delete(r.pads, dev)
	delete(r.byID, p.id)
	delete(r.noDriver, dev)
	if p.ff != nil {
		p.ff.Close()
		p.ff = nil
	}
	switch {
	case p.hid != nil:
		p.hid.Close()
		p.bridge.Remove()
	case p.announced:
		r.h.PadRemoved(p.id)
	}
	dev.Release()
}

// hidOf returns the HID device of a pad that a driver serves, or nil.
func (s *Service) hidOf(id platform.PadID) *gamepadhid.Device {
	if r := s.cur; r != nil {
		if p := r.byID[id]; p != nil {
			return p.hid
		}
	}
	return nil
}

// Rumble implements platform.Gamepads.
func (s *Service) Rumble(id platform.PadID, low, high uint16, d time.Duration) error {
	if dev := s.hidOf(id); dev != nil {
		return dev.Rumble(low, high, d)
	}
	if r := s.cur; r != nil {
		if p := r.byID[id]; p != nil && p.layout != nil {
			if ff, ok := p.dev.(ForceFeedbacker); ok && ff.HasForceFeedback() {
				if p.ff == nil {
					rumble, err := ff.OpenForceFeedback()
					if err != nil {
						return err
					}
					p.ff = rumble
				}
				return p.ff.Play(low, high, d)
			}
		}
	}
	return errors.New(platform.TagGamepadRumbleUnavailable, "the pad has no HID driver and no force feedback", "pad", int64(id))
}

// RumbleTriggers implements platform.Gamepads.
func (s *Service) RumbleTriggers(id platform.PadID, left, right uint16, d time.Duration) error {
	if dev := s.hidOf(id); dev != nil {
		return dev.RumbleTriggers(left, right, d)
	}
	return errors.New(platform.TagGamepadRumbleUnavailable, "only the pads that have a HID driver rumble their triggers on macOS", "pad", int64(id))
}

// SetLED implements platform.Gamepads.
func (s *Service) SetLED(id platform.PadID, r, g, b uint8) error {
	if dev := s.hidOf(id); dev != nil {
		return dev.SetLED(r, g, b)
	}
	return errors.New(platform.TagGamepadLEDUnavailable, "the light of a pad can be set only through its HID driver", "pad", int64(id))
}

var _ platform.PadSensorControl = (*Service)(nil)

// SetPadSensors implements platform.PadSensorControl: the sensors of a pad are on its HID driver.
func (s *Service) SetPadSensors(id platform.PadID, on bool) error {
	if dev := s.hidOf(id); dev != nil {
		return dev.SetSensors(on)
	}
	return errors.New(platform.TagGamepadSensorsUnavailable, "the sensors of a pad can be read only through its HID driver", "pad", int64(id))
}
