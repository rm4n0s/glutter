package winpad

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/gamepadhid"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/padstd"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// HIDDevice is a HID device that Raw Input lists, with what it says about itself.
//
// Flutter: no counterpart
type HIDDevice struct {
	// Handle is the handle of Raw Input; the reports of the device carry it.
	Handle uintptr
	// Path is the path of the device.
	Path string
	// Vendor, Product and Version are the numbers the device reports.
	Vendor, Product, Version uint16
	// UsagePage and Usage say what the device is (the generic desktop page and the joystick, gamepad or multi-axis usage for a
	// pad).
	UsagePage, Usage uint16
}

// HIDDescription is what the parsed report descriptor of a device says.
//
// Flutter: no counterpart
type HIDDescription struct {
	// Name is the product name of the device, empty when it has none.
	Name string
	// Buttons and Values are the capabilities of the input reports, and MaxItems the most items one report has.
	Buttons  []win32.HidpButtonCaps
	Values   []win32.HidpValueCaps
	MaxItems int
}

// HIDSource is what the service asks of Raw Input and the HID parser; package windows has the real one.
//
// Flutter: no counterpart
type HIDSource interface {
	// Devices lists the HID devices that are there.
	Devices() ([]HIDDevice, error)
	// Describe reads the report descriptor of a device and keeps it for GetData.
	Describe(d HIDDevice) (HIDDescription, error)
	// GetData reads the items of a report of the device into items and says how many there are.
	GetData(d HIDDevice, report []byte, items []win32.HidpData) (int, error)
	// Release lets go of what Describe kept for the device.
	Release(d HIDDevice)
}

// Opts are the options of a Service.
//
// Flutter: no counterpart
type Opts struct {
	// Post runs a function on the UI thread; it is safe to call from any goroutine.
	Post func(func())
	// Clock gives the time of an event.
	Clock func() time.Duration
	// Logger receives the notes of the service; nil logs nothing.
	Logger *slog.Logger
	// XInput serves the pads of XInput; nil when the library is not there.
	XInput XInputAPI
	// HID lists and reads the other pads; nil for none.
	HID HIDSource
	// OpenHID opens the HID device at a path as the transport of a HID driver (SPEC §28.2, step 31.3); nil gives no drivers, and
	// every pad is read through Raw Input. DisableDrivers does the same when the transport is there.
	OpenHID        func(path string) (gamepadhid.Transport, error)
	DisableDrivers bool
	// Asleep starts the service with its drivers asleep (EnableExtras wakes them): every pad is read through XInput or Raw
	// Input, and nothing is written to a pad, until the app uses package gamepad (SPEC §28.2, "Decided in step 36.1").
	Asleep bool
	// PollEvery is how often the XInput pads are read while there is one (8 ms when zero); ScanEvery how often the empty slots
	// are looked at and the HID devices listed when Windows has told of nothing (2 s when zero).
	PollEvery, ScanEvery time.Duration
}

// Service is the gamepad service of Windows (platform.Gamepads). XInput pads are read by polling, the HID pads from the reports
// that Raw Input delivers (HIDReports), and Rescan says that a device came or went.
//
// Flutter: no counterpart
type Service struct {
	opts   Opts
	log    *slog.Logger
	nextID atomic.Int64

	mu  sync.Mutex
	cur *run

	// motors guards slots and timers: the XInput slot of each pad that rumbles, and the timers that stop its motors.
	motors sync.Mutex
	slots  map[platform.PadID]uint32
	timers map[platform.PadID]*time.Timer

	// drivers are the pads that HID drivers serve (drivers.go).
	drivers driverDevices
	// asleep says that the drivers are asleep (Opts.Asleep, EnableExtras).
	asleep atomic.Bool
}

var (
	_ platform.Gamepads         = (*Service)(nil)
	_ platform.PadSensorControl = (*Service)(nil)
)

// New makes the service. It does nothing until Start.
//
// Flutter: no counterpart
func New(opts Opts) *Service {
	if opts.PollEvery <= 0 {
		opts.PollEvery = 8 * time.Millisecond
	}
	if opts.ScanEvery <= 0 {
		opts.ScanEvery = 2 * time.Second
	}
	log := opts.Logger
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	s := &Service{opts: opts, log: log, slots: map[platform.PadID]uint32{}, timers: map[platform.PadID]*time.Timer{}}
	s.asleep.Store(opts.Asleep)
	return s
}

var _ platform.PadExtras = (*Service)(nil)

// EnableExtras implements platform.PadExtras: the HID drivers wake. The pads that Raw Input serves and that have a driver are let
// go (the handler is told they are removed) and the next look at the devices starts their drivers and adds them. Called before
// Start, it makes the service start awake.
func (s *Service) EnableExtras() {
	if !s.asleep.Swap(false) {
		return
	}
	s.mu.Lock()
	r := s.cur
	s.mu.Unlock()
	if r != nil {
		select {
		case r.upgrade <- struct{}{}:
		default:
		}
	}
}

// MappingPlatform implements platform.Gamepads.
func (s *Service) MappingPlatform() string { return "Windows" }

// Start implements platform.Gamepads.
func (s *Service) Start(h platform.PadHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cur != nil {
		return nil
	}
	r := &run{
		s: s, h: h, stop: make(chan struct{}), done: make(chan struct{}),
		wake: make(chan struct{}, 1), upgrade: make(chan struct{}, 1), reports: make(chan hidReports, 256),
		xi: map[uint32]*xinputPad{}, hid: map[uintptr]*hidPad{},
		drv: map[uintptr]*drvPad{}, drvFailed: map[uintptr]bool{}, drvEvents: make(chan drvEvent, 16),
	}
	s.cur = r
	go r.loop()
	return nil
}

// Stop implements platform.Gamepads.
func (s *Service) Stop() {
	s.mu.Lock()
	r := s.cur
	s.cur = nil
	s.mu.Unlock()
	if r == nil {
		return
	}
	r.stopped.Store(true)
	close(r.stop)
	<-r.done
	s.motors.Lock()
	for id, t := range s.timers {
		t.Stop()
		delete(s.timers, id)
	}
	s.motors.Unlock()
}

// Rescan says that a device came or went: the service looks at the slots and the HID devices at once. It does not block and
// may be called from any goroutine.
//
// Flutter: no counterpart
func (s *Service) Rescan() {
	s.mu.Lock()
	r := s.cur
	s.mu.Unlock()
	if r == nil {
		return
	}
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

// HIDReports gives the service the reports that Raw Input delivered for a device, as they were read at time at. The reports are
// kept, so the caller may reuse its buffers. It does not block: when the service is behind, reports are dropped (each carries the
// whole state of the pad, so the next one repairs it).
//
// Flutter: no counterpart
func (s *Service) HIDReports(device uintptr, reports [][]byte, at time.Duration) {
	s.mu.Lock()
	r := s.cur
	s.mu.Unlock()
	if r == nil {
		return
	}
	cp := make([][]byte, len(reports))
	for i, b := range reports {
		cp[i] = append([]byte(nil), b...)
	}
	select {
	case r.reports <- hidReports{device: device, reports: cp, at: at}:
	default:
	}
}

// Rumble implements platform.Gamepads for the XInput pads: the motors turn at the strengths for d, and a timer stops them. A
// duration of 0 stops them. The pads of the HID drivers rumble through their driver; a pad read through Raw Input cannot rumble.
func (s *Service) Rumble(id platform.PadID, low, high uint16, d time.Duration) error {
	if dev := s.drivers.get(id); dev != nil {
		return dev.Rumble(low, high, d)
	}
	s.motors.Lock()
	defer s.motors.Unlock()
	user, ok := s.slots[id]
	if !ok || s.opts.XInput == nil {
		return errors.New(platform.TagGamepadRumbleUnavailable, "the pad is not an XInput pad", "pad", int64(id))
	}
	if t := s.timers[id]; t != nil {
		t.Stop()
		delete(s.timers, id)
	}
	if d <= 0 {
		low, high = 0, 0
	}
	v := win32.XInputVibration{LeftMotorSpeed: low, RightMotorSpeed: high}
	if code := s.opts.XInput.SetState(user, &v); code != win32.ErrorSuccessCode {
		return errors.New(tagXInputRumbleFailed, "XInputSetState failed", "pad", int64(id), "slot", int(user), "code", int(code))
	}
	if d > 0 {
		s.timers[id] = time.AfterFunc(d, func() { s.stopMotors(id, user) })
	}
	return nil
}

// stopMotors turns the motors of a pad off when its time is over.
func (s *Service) stopMotors(id platform.PadID, user uint32) {
	s.motors.Lock()
	defer s.motors.Unlock()
	if slot, ok := s.slots[id]; !ok || slot != user {
		return
	}
	delete(s.timers, id)
	var v win32.XInputVibration
	if code := s.opts.XInput.SetState(user, &v); code != win32.ErrorSuccessCode {
		s.log.Debug("the motors of a pad could not be stopped", "slot", int(user), "code", int(code))
	}
}

// RumbleTriggers implements platform.Gamepads: the pads of the HID drivers that have motors in their triggers rumble them; XInput
// has none.
func (s *Service) RumbleTriggers(id platform.PadID, left, right uint16, d time.Duration) error {
	if dev := s.drivers.get(id); dev != nil {
		return dev.RumbleTriggers(left, right, d)
	}
	return errors.New(platform.TagGamepadRumbleUnavailable, "the triggers of a pad cannot rumble through XInput", "pad", int64(id))
}

// SetLED implements platform.Gamepads: the pads of the HID drivers that have a light set it; neither XInput nor Raw Input can.
func (s *Service) SetLED(id platform.PadID, r, g, b uint8) error {
	if dev := s.drivers.get(id); dev != nil {
		return dev.SetLED(r, g, b)
	}
	return errors.New(platform.TagGamepadLEDUnavailable, "the light of a pad cannot be set on Windows without its HID driver", "pad", int64(id))
}

// SetPadSensors implements platform.PadSensorControl: the sensors of a pad are on its HID driver.
func (s *Service) SetPadSensors(id platform.PadID, on bool) error {
	if dev := s.drivers.get(id); dev != nil {
		return dev.SetSensors(on)
	}
	return errors.New(platform.TagGamepadSensorsUnavailable, "the pad has no motion sensors that Windows can reach without its HID driver", "pad", int64(id))
}

// hidReports are the reports of a HID device that Raw Input delivered together.
type hidReports struct {
	device  uintptr
	reports [][]byte
	at      time.Duration
}

type xinputPad struct {
	id      platform.PadID
	user    uint32
	tracker *padstd.Tracker
	packet  uint32
}

type hidPad struct {
	dev     HIDDevice
	id      platform.PadID
	tracker *hidTracker
	items   []win32.HidpData
}

// run is one Start of the service; its goroutine owns the pads.
type run struct {
	s    *Service
	h    platform.PadHandler
	stop chan struct{}
	done chan struct{}
	wake chan struct{}
	// upgrade asks the loop to let go of the pads that a driver can serve now (EnableExtras).
	upgrade chan struct{}
	reports chan hidReports
	stopped atomic.Bool

	xi  map[uint32]*xinputPad
	hid map[uintptr]*hidPad
	// drv are the pads that HID drivers serve, by the handle of Raw Input; drvFailed the devices whose driver gave up, which are
	// read through Raw Input; drvEvents is what the drivers tell the loop.
	drv       map[uintptr]*drvPad
	drvFailed map[uintptr]bool
	drvEvents chan drvEvent
	// warned is the notes that were made once.
	warned map[string]bool
}

// deliver runs fn on the UI thread unless the service has been stopped by then.
func (r *run) deliver(fn func(platform.PadHandler)) {
	r.s.opts.Post(func() {
		if !r.stopped.Load() {
			fn(r.h)
		}
	})
}

func (r *run) loop() {
	defer close(r.done)
	r.scanXInput()
	r.scanHID()
	poll := time.NewTicker(r.s.opts.PollEvery)
	defer poll.Stop()
	scan := time.NewTicker(r.s.opts.ScanEvery)
	defer scan.Stop()
	for {
		select {
		case <-r.stop:
			r.s.motors.Lock()
			clear(r.s.slots)
			r.s.motors.Unlock()
			r.closeDrivers()
			return
		case ev := <-r.drvEvents:
			r.driverEvent(ev)
		case <-r.upgrade:
			r.upgradeDrivers()
			r.scanHID()
		case <-r.wake:
			r.scanXInput()
			r.scanHID()
		case rep := <-r.reports:
			r.readReports(rep)
		case <-scan.C:
			r.scanXInput()
			r.scanHID()
		case <-poll.C:
			// Only the pads that are there are read: XInput is slow to answer for an empty slot.
			if len(r.xi) > 0 {
				r.pollXInput()
			}
		}
	}
}

// scanXInput looks at the slots that have no pad and connects the pads that XInput has in them.
func (r *run) scanXInput() {
	x := r.s.opts.XInput
	if x == nil {
		return
	}
	for user := uint32(0); user < win32.XUserMaxCount; user++ {
		if r.xi[user] != nil {
			continue
		}
		var caps win32.XInputCapabilities
		if x.GetCapabilities(user, win32.XInputFlagGamepad, &caps) != win32.ErrorSuccessCode || caps.SubType == 0 {
			continue
		}
		var st win32.XInputState
		if x.GetState(user, &st) != win32.ErrorSuccessCode {
			continue
		}
		r.connectXInput(user, &caps, &st)
	}
}

func (r *run) connectXInput(user uint32, caps *win32.XInputCapabilities, st *win32.XInputState) {
	x := r.s.opts.XInput
	vendor, product, version := xinputIdentity(x, user, caps)
	raw := xinputName(user, caps.SubType)
	bat, ok := x.GetBatteryInformation(user)
	id := platform.PadID(r.s.nextID.Add(1))
	info := platform.PadInfo{
		ID:       id,
		Name:     gamepaddb.DeviceName(vendor, product, "", raw),
		GUID:     xinputGUID(vendor, product, version, raw, caps.SubType),
		Path:     fmt.Sprintf("XInput#%d", user),
		Axes:     int(gamepaddb.AxisCount),
		Buttons:  int(gamepaddb.ButtonCount),
		Standard: true,
		Rumble:   true,
		Battery:  xinputBattery(bat, ok),
	}
	p := &xinputPad{id: id, user: user, tracker: padstd.NewTracker(id), packet: st.PacketNumber}
	r.xi[user] = p
	r.s.motors.Lock()
	r.s.slots[id] = user
	r.s.motors.Unlock()
	snap := xinputSnapshot(st.Gamepad)
	events := p.tracker.Update(r.now(), &snap, nil)
	r.deliver(func(h platform.PadHandler) { h.PadAdded(info) })
	if len(events) > 0 {
		r.deliver(func(h platform.PadHandler) { h.PadEvents(events) })
	}
}

func (r *run) now() time.Duration {
	if r.s.opts.Clock == nil {
		return 0
	}
	return r.s.opts.Clock()
}

// pollXInput reads the pads of the slots and reports what changed; a slot that no longer answers loses its pad.
func (r *run) pollXInput() {
	x := r.s.opts.XInput
	var batch []platform.PadEvent
	for user, p := range r.xi {
		var st win32.XInputState
		if x.GetState(user, &st) != win32.ErrorSuccessCode {
			r.disconnectXInput(user, p)
			continue
		}
		if st.PacketNumber == p.packet {
			continue
		}
		p.packet = st.PacketNumber
		snap := xinputSnapshot(st.Gamepad)
		batch = p.tracker.Update(r.now(), &snap, batch)
	}
	if len(batch) > 0 {
		r.deliver(func(h platform.PadHandler) { h.PadEvents(batch) })
	}
}

func (r *run) disconnectXInput(user uint32, p *xinputPad) {
	delete(r.xi, user)
	r.s.motors.Lock()
	delete(r.s.slots, p.id)
	if t := r.s.timers[p.id]; t != nil {
		t.Stop()
		delete(r.s.timers, p.id)
	}
	r.s.motors.Unlock()
	id := p.id
	r.deliver(func(h platform.PadHandler) { h.PadRemoved(id) })
}

// scanHID lists the HID devices, connects the new pads and drops those that are gone.
func (r *run) scanHID() {
	src := r.s.opts.HID
	if src == nil {
		return
	}
	devices, err := src.Devices()
	if err != nil {
		if !r.warned["hid-list"] {
			if r.warned == nil {
				r.warned = map[string]bool{}
			}
			r.warned["hid-list"] = true
			r.s.log.Warn("the HID devices cannot be listed, so only the XInput pads are found", "err", err)
		}
		return
	}
	present := make(map[uintptr]bool, len(devices))
	for _, d := range devices {
		if d.UsagePage != win32.HIDUsagePageGeneric ||
			(d.Usage != win32.HIDUsageGenericJoystick && d.Usage != win32.HIDUsageGenericGamepad && d.Usage != win32.HIDUsageGenericMultiAxis) {
			continue
		}
		// An XInput pad has a HID interface too: XInput serves it, so it is skipped, unless XInput is not there to do so.
		if r.s.opts.XInput != nil && isXInputPath(d.Path) {
			continue
		}
		present[d.Handle] = true
		if r.drv[d.Handle] != nil || r.hid[d.Handle] != nil {
			continue
		}
		if !r.tryDriver(d) {
			r.connectHID(d)
		}
	}
	for _, p := range r.drv {
		if !present[p.handle] {
			r.dropDriver(p)
		}
	}
	for handle := range r.drvFailed {
		if !present[handle] {
			delete(r.drvFailed, handle)
		}
	}
	for handle, p := range r.hid {
		if !present[handle] {
			delete(r.hid, handle)
			src.Release(p.dev)
			id := p.id
			r.deliver(func(h platform.PadHandler) { h.PadRemoved(id) })
		}
	}
}

func (r *run) connectHID(d HIDDevice) {
	src := r.s.opts.HID
	desc, err := src.Describe(d)
	if err != nil {
		r.s.log.Debug("a HID device cannot be read as a pad", "path", d.Path, "err", err)
		return
	}
	layout := NewLayout(desc.Buttons, desc.Values, desc.MaxItems)
	if !layout.IsPad() {
		src.Release(d)
		return
	}
	buttons, axes, hats := layout.Counts()
	name := desc.Name
	id := platform.PadID(r.s.nextID.Add(1))
	info := platform.PadInfo{
		ID:      id,
		Name:    gamepaddb.DeviceName(d.Vendor, d.Product, "", name),
		GUID:    gamepaddb.NewGUID(busOfPath(d.Path), d.Vendor, d.Product, d.Version, "", name, 0, 0),
		Path:    d.Path,
		Axes:    axes,
		Buttons: buttons,
		Hats:    hats,
		Battery: -1,
	}
	r.hid[d.Handle] = &hidPad{dev: d, id: id, tracker: &hidTracker{id: id, layout: layout}, items: make([]win32.HidpData, max(layout.MaxItems, 1))}
	r.deliver(func(h platform.PadHandler) { h.PadAdded(info) })
}

// readReports decodes the reports of a HID pad.
func (r *run) readReports(rep hidReports) {
	p := r.hid[rep.device]
	if p == nil {
		return
	}
	var batch []platform.PadEvent
	for _, report := range rep.reports {
		n, err := r.s.opts.HID.GetData(p.dev, report, p.items)
		if err != nil {
			continue
		}
		batch = p.tracker.update(rep.at, p.items[:n], batch)
	}
	if len(batch) > 0 {
		r.deliver(func(h platform.PadHandler) { h.PadEvents(batch) })
	}
}

// The tag of the error of a rumble that XInput refused (rm4n0s/errors, R5).
const tagXInputRumbleFailed = "XInputRumbleFailed"
