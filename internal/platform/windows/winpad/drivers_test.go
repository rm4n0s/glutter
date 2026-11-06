package winpad

import (
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/gamepadhid"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func (r *recorder) PadTouch(t platform.PadTouch) {
	r.mu.Lock()
	r.touches = append(r.touches, t)
	r.mu.Unlock()
}

func (r *recorder) PadSensor(s platform.PadSensor) {
	r.mu.Lock()
	r.sensors = append(r.sensors, s)
	r.mu.Unlock()
}

func (r *recorder) PadBattery(b platform.PadBattery) {
	r.mu.Lock()
	r.batteries = append(r.batteries, b)
	r.mu.Unlock()
}

// fakeTransport is the transport of a HID driver under test.
type fakeTransport struct {
	in       chan []byte
	failRead chan error
	closed   chan struct{}
	once     sync.Once

	mu      sync.Mutex
	written [][]byte
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{in: make(chan []byte, 64), failRead: make(chan error, 1), closed: make(chan struct{})}
}

func (f *fakeTransport) Read(buf []byte, timeout time.Duration) (int, error) {
	select {
	case r := <-f.in:
		return copy(buf, r), nil
	case err := <-f.failRead:
		return 0, err
	case <-f.closed:
		return 0, errors.New(tagFakeTransportClosed, "closed")
	case <-time.After(timeout):
		return 0, nil
	}
}

func (f *fakeTransport) Write(data []byte) (int, error) {
	f.mu.Lock()
	f.written = append(f.written, append([]byte(nil), data...))
	f.mu.Unlock()
	return len(data), nil
}

func (f *fakeTransport) SendFeature(data []byte) (int, error) { return len(data), nil }
func (f *fakeTransport) GetFeature(data []byte) (int, error) {
	return -1, errors.New(tagFakeFeatureMissing, "no such feature report")
}
func (f *fakeTransport) Close() error { f.once.Do(func() { close(f.closed) }); return nil }

func (f *fakeTransport) writes() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte(nil), f.written...)
}

func (f *fakeTransport) isClosed() bool {
	select {
	case <-f.closed:
		return true
	default:
		return false
	}
}

func waitWrites(t *testing.T, f *fakeTransport, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for len(f.writes()) < n {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d writes, have %d", n, len(f.writes()))
		}
		time.Sleep(time.Millisecond)
	}
}

// opener is Opts.OpenHID with transports that tests can reach by the path.
type opener struct {
	mu     sync.Mutex
	opened map[string]*fakeTransport
	err    error
}

func (o *opener) open(path string) (gamepadhid.Transport, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.err != nil {
		return nil, o.err
	}
	if o.opened == nil {
		o.opened = map[string]*fakeTransport{}
	}
	tr := newFakeTransport()
	o.opened[path] = tr
	return tr, nil
}

func (o *opener) transport(path string) *fakeTransport {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.opened[path]
}

const (
	ds4USBPath = `\\?\hid#vid_054c&pid_09cc#7&2f3a1b&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`
	ds4BTPath  = `\\?\hid#{00001124-0000-1000-8000-00805f9b34fb}_vid&0002054c_pid&09cc#8&1&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`
	proUSBPath = `\\?\hid#vid_057e&pid_2009#7&1&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`
	xboxBTPath = `\\?\hid#{00001124-0000-1000-8000-00805f9b34fb}_vid&0002045e_pid&0b13#8&2&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`
)

func ds4Device(handle uintptr, path string) HIDDevice {
	return HIDDevice{Handle: handle, Path: path, Vendor: 0x054c, Product: 0x09cc, Version: 0x0100, UsagePage: win32.HIDUsagePageGeneric, Usage: win32.HIDUsageGenericGamepad}
}

// ps4Report is the input report of a DualShock 4 on USB with the buttons byte given: the cross is 0x28.
func ps4Report(buttons byte) []byte {
	r := make([]byte, 64)
	r[0] = 1
	r[1], r[2], r[3], r[4] = 128, 128, 128, 128
	r[5] = buttons
	return r
}

func newDriverService(r *recorder, x XInputAPI, h HIDSource, o *opener) *Service {
	var open func(string) (gamepadhid.Transport, error)
	if o != nil {
		open = o.open
	}
	return New(Opts{Post: r.post, XInput: x, HID: h, OpenHID: open, PollEvery: 2 * time.Millisecond, ScanEvery: 20 * time.Millisecond})
}

func TestHIDPadIsServedByItsDriver(t *testing.T) {
	h := &fakeHID{}
	h.set(ds4Device(7, ds4USBPath))
	o := &opener{}
	r := newRecorder()
	s := newDriverService(r, nil, h, o)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
	info := r.added[0]
	if !info.Standard || info.Name != "PS4 Controller" || info.Path != ds4USBPath || !info.Rumble || !info.LED || !info.Sensors || info.Touchpads != 1 {
		t.Errorf("info = %+v", info)
	}
	if g := gamepaddb.GUID(info.GUID); g[14] != 'h' {
		t.Errorf("the GUID does not have the signature of the HID drivers: %s", g)
	}
	if len(h.described) != 0 {
		t.Error("a pad served by a driver was read through Raw Input")
	}
	tr := o.transport(ds4USBPath)
	waitWrites(t, tr, 1)

	tr.in <- ps4Report(0x28)
	r.wait(t, "the cross", func() bool {
		for _, e := range r.events {
			if e.Kind == platform.PadButton && e.Index == int(gamepaddb.South) && e.Value == 1 && e.Pad == info.ID {
				return true
			}
		}
		return false
	})

	// Rumble, the light and the sensors go to the driver.
	if err := s.Rumble(info.ID, 0xFFFF, 0, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLED(info.ID, 1, 2, 3); err != nil {
		t.Fatal(err)
	}
	if err := s.RumbleTriggers(info.ID, 1, 1, time.Second); err == nil {
		t.Error("a PlayStation 4 pad rumbled its triggers")
	}
	if err := s.SetPadSensors(info.ID, true); err != nil {
		t.Errorf("the sensors cannot be turned on: %v", err)
	}
	waitWrites(t, tr, 3)
	w := tr.writes()
	if w[1][1] != 0x01 || w[1][5] != 0xFF || w[2][1] != 0x02 || w[2][6] != 1 || w[2][7] != 2 || w[2][8] != 3 {
		t.Errorf("writes = %x, %x", w[1][:10], w[2][:10])
	}

	// The battery and the touchpad come with the reports.
	rep := ps4Report(0x08)
	rep[1+29] = 0x15
	rep[1+34] = 0x00
	rep[1+35], rep[1+36], rep[1+37] = 0xC0, 0x03, 0x1C
	tr.in <- rep
	r.wait(t, "the battery and the finger", func() bool {
		return len(r.batteries) > 0 && r.batteries[len(r.batteries)-1].State == platform.PadCharging && len(r.touches) >= 1
	})
	if b := r.batteries[len(r.batteries)-1]; b.Pad != info.ID || b.Percent != 55 {
		t.Errorf("battery = %+v", b)
	}
	if tc := r.touches[0]; tc.Pad != info.ID || !tc.Down {
		t.Errorf("touch = %+v", tc)
	}

	// Raw Input delivers the reports of the device too; they are not read twice.
	s.HIDReports(7, [][]byte{{0x80, 1}}, time.Second)
	time.Sleep(30 * time.Millisecond)
	r.drain()

	// The device goes with its pad, and the driver ends.
	h.set()
	r.wait(t, "the removal", func() bool { return len(r.gone) == 1 })
	if r.gone[0] != info.ID {
		t.Errorf("removed %d, want %d", r.gone[0], info.ID)
	}
	if !tr.isClosed() {
		t.Error("the device of a pad that went was not closed")
	}
	if err := s.Rumble(info.ID, 1, 1, time.Second); err == nil {
		t.Error("a pad that went was told to rumble")
	}
}

func TestDriverThatCannotStartFallsBackToRawInput(t *testing.T) {
	// A Switch Pro controller on USB that does not answer the handshake: the driver gives it up and Raw Input serves the pad.
	buttons, values := flightPad()
	h := &fakeHID{desc: map[uintptr]HIDDescription{3: {Name: "Pro Controller", Buttons: buttons, Values: values, MaxItems: 12}}}
	dev := padDevice(3, proUSBPath)
	dev.Vendor, dev.Product = 0x057e, 0x2009
	h.set(dev)
	o := &opener{}
	r := newRecorder()
	s := newDriverService(r, nil, h, o)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
	if r.added[0].Standard {
		t.Errorf("the pad is served by a driver that gave it up: %+v", r.added[0])
	}
	if !o.transport(proUSBPath).isClosed() {
		t.Error("the transport of a driver that failed was not closed")
	}
	if err := s.SetPadSensors(r.added[0].ID, true); err == nil {
		t.Error("the sensors of a pad read through Raw Input were turned on")
	}
	if err := s.SetLED(r.added[0].ID, 1, 2, 3); err == nil {
		t.Error("the light of a pad read through Raw Input was set")
	}
}

func TestDriverThatIsLostFallsBackToRawInput(t *testing.T) {
	buttons, values := flightPad()
	h := &fakeHID{desc: map[uintptr]HIDDescription{7: {Name: "Wireless Controller", Buttons: buttons, Values: values, MaxItems: 12}}}
	h.set(ds4Device(7, ds4USBPath))
	o := &opener{}
	r := newRecorder()
	s := newDriverService(r, nil, h, o)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
	o.transport(ds4USBPath).failRead <- errors.New(tagFakeDeviceGone, "the device is gone")
	r.wait(t, "the second pad", func() bool { return len(r.added) == 2 && len(r.gone) == 1 })
	if r.added[1].Standard {
		t.Errorf("the pad is not read through Raw Input: %+v", r.added[1])
	}
	if r.gone[0] != r.added[0].ID {
		t.Errorf("removed %d, want %d", r.gone[0], r.added[0].ID)
	}
}

func TestDevicesThatDriversDoNotServe(t *testing.T) {
	buttons, values := flightPad()
	xboxDev := HIDDevice{Handle: 9, Path: xboxBTPath, Vendor: 0x045e, Product: 0x0b13, UsagePage: win32.HIDUsagePageGeneric, Usage: win32.HIDUsageGenericGamepad}
	tests := []struct {
		name string
		dev  HIDDevice
		opts func(o *opener) (XInputAPI, *opener)
	}{
		{"a driver that is off", ds4Device(7, ds4USBPath), func(o *opener) (XInputAPI, *opener) { return nil, nil }},
		{"an Xbox pad over Bluetooth while XInput is there", xboxDev, func(o *opener) (XInputAPI, *opener) { return &fakeXInput{}, o }},
		{"a pad with no driver", padDevice(5, `\\?\hid#vid_1234&pid_5678#1`), func(o *opener) (XInputAPI, *opener) { return nil, o }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &fakeHID{desc: map[uintptr]HIDDescription{tt.dev.Handle: {Name: "Pad", Buttons: buttons, Values: values, MaxItems: 12}}}
			h.set(tt.dev)
			o := &opener{}
			x, op := tt.opts(o)
			r := newRecorder()
			s := newDriverService(r, x, h, op)
			if err := s.Start(r); err != nil {
				t.Fatal(err)
			}
			defer s.Stop()
			r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
			if r.added[0].Standard {
				t.Errorf("a driver served the pad: %+v", r.added[0])
			}
			if o.transport(tt.dev.Path) != nil {
				t.Error("the device was opened for a driver")
			}
		})
	}
}

func TestXboxPadOverBluetoothIsServedByItsDriverWithoutXInput(t *testing.T) {
	h := &fakeHID{}
	h.set(HIDDevice{Handle: 9, Path: xboxBTPath, Vendor: 0x045e, Product: 0x0b13, UsagePage: win32.HIDUsagePageGeneric, Usage: win32.HIDUsageGenericGamepad})
	o := &opener{}
	r := newRecorder()
	s := newDriverService(r, nil, h, o)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
	if !r.added[0].Standard || !r.added[0].RumbleTriggers {
		t.Errorf("info = %+v", r.added[0])
	}
}

func TestDeviceThatCannotBeOpenedIsReadThroughRawInput(t *testing.T) {
	buttons, values := flightPad()
	h := &fakeHID{desc: map[uintptr]HIDDescription{7: {Name: "Wireless Controller", Buttons: buttons, Values: values, MaxItems: 12}}}
	h.set(ds4Device(7, ds4USBPath))
	o := &opener{err: errors.New(tagFakeAccessDenied, "access denied")}
	r := newRecorder()
	s := newDriverService(r, nil, h, o)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
	if r.added[0].Standard {
		t.Errorf("info = %+v", r.added[0])
	}
}

func TestDriversEndWhenTheServiceStops(t *testing.T) {
	h := &fakeHID{}
	h.set(ds4Device(7, ds4USBPath))
	o := &opener{}
	r := newRecorder()
	s := newDriverService(r, nil, h, o)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
	id := r.added[0].ID
	s.Stop()
	if !o.transport(ds4USBPath).isClosed() {
		t.Error("the device of a driver was not closed when the service stopped")
	}
	if err := s.Rumble(id, 1, 1, time.Second); err == nil {
		t.Error("a pad of a service that stopped was told to rumble")
	}
}

func TestDriversCanBeTurnedOff(t *testing.T) {
	buttons, values := flightPad()
	h := &fakeHID{desc: map[uintptr]HIDDescription{7: {Name: "Wireless Controller", Buttons: buttons, Values: values, MaxItems: 12}}}
	h.set(ds4Device(7, ds4USBPath))
	o := &opener{}
	r := newRecorder()
	s := New(Opts{Post: r.post, HID: h, OpenHID: o.open, DisableDrivers: true, PollEvery: 2 * time.Millisecond, ScanEvery: 20 * time.Millisecond})
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
	if r.added[0].Standard || o.transport(ds4USBPath) != nil {
		t.Error("a driver served a pad while they were off")
	}
}

func TestBusOfDriverPaths(t *testing.T) {
	for _, tt := range []struct {
		path string
		want gamepadhid.Bus
	}{
		{ds4USBPath, gamepadhid.BusUSB},
		{ds4BTPath, gamepadhid.BusBluetooth},
		{`\\?\hid#{00001812-0000-1000-8000-00805f9b34fb}_dev_vid&02057e_pid&2009#9&1`, gamepadhid.BusBluetooth},
		{`\\?\HID#BTHENUM#{00001124-0000-1000-8000-00805f9b34fb}`, gamepadhid.BusBluetooth},
	} {
		if got := busOf(tt.path); got != tt.want {
			t.Errorf("busOf(%q) = %d, want %d", tt.path, got, tt.want)
		}
	}
}

// The tags of the errors of the fake transport.
const (
	tagFakeTransportClosed = "FakeTransportClosed"
	tagFakeFeatureMissing  = "FakeFeatureMissing"
	tagFakeDeviceGone      = "FakeDeviceGone"
	tagFakeAccessDenied    = "FakeAccessDenied"
)
