package winpad

import (
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func buttonRange(min, max uint16, dataMin, dataMax uint16) win32.HidpButtonCaps {
	c := win32.HidpButtonCaps{UsagePage: win32.HIDUsagePageButton, IsRange: 1}
	c.Union[0], c.Union[1] = byte(min), byte(min>>8)
	c.Union[2], c.Union[3] = byte(max), byte(max>>8)
	c.Union[12], c.Union[13] = byte(dataMin), byte(dataMin>>8)
	c.Union[14], c.Union[15] = byte(dataMax), byte(dataMax>>8)
	return c
}

func button(usage, data uint16) win32.HidpButtonCaps {
	c := win32.HidpButtonCaps{UsagePage: win32.HIDUsagePageButton}
	c.Union[0], c.Union[1] = byte(usage), byte(usage>>8)
	c.Union[12], c.Union[13] = byte(data), byte(data>>8)
	return c
}

func value(usage, data uint16, bits uint16, min, max int32) win32.HidpValueCaps {
	c := win32.HidpValueCaps{UsagePage: win32.HIDUsagePageGeneric, BitSize: bits, LogicalMin: min, LogicalMax: max}
	c.Union[0], c.Union[1] = byte(usage), byte(usage>>8)
	c.Union[12], c.Union[13] = byte(data), byte(data>>8)
	return c
}

// flightPad is a pad as its report descriptor says: 2 button ranges (the second after a gap in the data indices), X, Y, Z axes
// (the Y and X given out of order), a hat of 0 to 7 and a vendor-defined value.
func flightPad() (buttons []win32.HidpButtonCaps, values []win32.HidpValueCaps) {
	buttons = []win32.HidpButtonCaps{buttonRange(1, 4, 0, 3), button(5, 4), {UsagePage: 0xFF00}}
	values = []win32.HidpValueCaps{
		value(0x31, 6, 8, 0, 255),    // Y
		value(0x30, 5, 8, 0, 255),    // X
		value(0x32, 7, 8, 0, 255),    // Z
		value(0x39, 8, 4, 0, 7),      // hat
		value(0x30, 9, 16, 0, 65535), // a range would be skipped, so make it one
	}
	values[4].IsRange = 1
	return buttons, values
}

func TestNewLayoutNumbersTheInputsAsSDLDoes(t *testing.T) {
	buttons, values := flightPad()
	l := NewLayout(buttons, values, 12)
	b, a, h := l.Counts()
	if b != 5 || a != 3 || h != 1 {
		t.Fatalf("counts = %d buttons, %d axes, %d hats, want 5, 3, 1", b, a, h)
	}
	// The axes are sorted by usage: X (data index 5), Y (6), Z (7).
	if l.axes[0].index != 5 || l.axes[1].index != 6 || l.axes[2].index != 7 || l.hats[0].index != 8 {
		t.Errorf("axes %+v hats %+v", l.axes, l.hats)
	}
	if want := []uint16{0, 1, 2, 3, 4}; len(l.buttons) != 5 || l.buttons[0] != want[0] || l.buttons[3] != want[3] || l.buttons[4] != want[4] {
		t.Errorf("buttons = %v, want %v", l.buttons, want)
	}
	if !l.IsPad() {
		t.Error("a device with buttons and axes is not a pad")
	}
	if NewLayout(buttons[:1], nil, 1).IsPad() || NewLayout(nil, values, 1).IsPad() {
		t.Error("buttons alone or axes alone are a pad")
	}
}

func TestAxisValue(t *testing.T) {
	tests := []struct {
		name string
		raw  uint32
		a    hidAxis
		want int16
	}{
		{"16 bit low", 0, hidAxis{bits: 16, min: 0, max: 65535}, -32768},
		{"16 bit middle", 32768, hidAxis{bits: 16, min: 0, max: 65535}, 0},
		{"16 bit high", 65535, hidAxis{bits: 16, min: 0, max: 65535}, 32767},
		{"8 bit low", 0, hidAxis{bits: 8, min: 0, max: 255}, -32768},
		{"8 bit high", 255, hidAxis{bits: 8, min: 0, max: 255}, 32767},
		{"8 bit centre", 128, hidAxis{bits: 8, min: 0, max: 255}, 128},
		{"signed 8 bit negative", 0xff, hidAxis{bits: 8, min: -128, max: 127}, -129},
		{"signed 8 bit min", 0x80, hidAxis{bits: 8, min: -128, max: 127}, -32768},
		{"signed 8 bit max", 0x7f, hidAxis{bits: 8, min: -128, max: 127}, 32767},
		{"out of range is clamped", 300, hidAxis{bits: 16, min: 0, max: 255}, 32767},
		{"empty range", 5, hidAxis{bits: 8, min: 3, max: 3}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := axisValue(tt.raw, tt.a); got != tt.want {
				t.Errorf("axisValue(%d) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestHatMask(t *testing.T) {
	const up, right, down, left = gamepaddb.HatUp, gamepaddb.HatRight, gamepaddb.HatDown, gamepaddb.HatLeft
	zero := hidHat{min: 0, max: 7}
	one := hidHat{min: 1, max: 8}
	four := hidHat{min: 0, max: 3}
	tests := []struct {
		name string
		raw  uint32
		h    hidHat
		want int
	}{
		{"0 to 7 up", 0, zero, up}, {"0 to 7 right", 2, zero, right}, {"0 to 7 down left", 5, zero, down | left}, {"0 to 7 up left", 7, zero, up | left},
		{"0 to 7 null 8", 8, zero, 0}, {"0 to 7 null 15", 15, zero, 0},
		{"1 to 8 up", 1, one, up}, {"1 to 8 up right", 2, one, up | right}, {"1 to 8 up left", 8, one, up | left}, {"1 to 8 null 0", 0, one, 0},
		{"4 positions right", 1, four, right}, {"4 positions left", 3, four, left},
		{"odd count", 1, hidHat{min: 0, max: 5}, 0},
		{"negative is the null state", 0xffffffff, zero, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hatMask(tt.raw, tt.h); got != tt.want {
				t.Errorf("hatMask(%d) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestHIDPathsAndBuses(t *testing.T) {
	for _, tt := range []struct {
		path    string
		xinput  bool
		bus     uint16
		comment string
	}{
		{`\\?\HID#VID_045E&PID_028E&IG_00#8&1a2b3c&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`, true, gamepaddb.BusUSB, "an Xbox pad"},
		{`\\?\HID#VID_054C&PID_09CC#7&2c&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`, false, gamepaddb.BusUSB, "a DualShock on USB"},
		{`\\?\HID#{00001124-0000-1000-8000-00805f9b34fb}_VID&0002054c_PID&09cc#8&1&0&0000`, false, gamepaddb.BusBluetooth, "a DualShock on Bluetooth"},
		{`\\?\HID#BTHENUM#pad`, false, gamepaddb.BusBluetooth, "BTHENUM"},
	} {
		if got := isXInputPath(tt.path); got != tt.xinput {
			t.Errorf("%s: isXInputPath = %v", tt.comment, got)
		}
		if got := busOfPath(tt.path); got != tt.bus {
			t.Errorf("%s: busOfPath = %d, want %d", tt.comment, got, tt.bus)
		}
	}
}

// fakeHID is a HIDSource with scripted devices; a report of the fake is the list of items, one byte of data index and one of
// value for each, and a data index with the high bit set is a button that is on.
type fakeHID struct {
	mu        sync.Mutex
	devices   []HIDDevice
	desc      map[uintptr]HIDDescription
	listErr   error
	described []uintptr
	released  []uintptr
}

func (f *fakeHID) set(devices ...HIDDevice) {
	f.mu.Lock()
	f.devices = devices
	f.mu.Unlock()
}

func (f *fakeHID) Devices() ([]HIDDevice, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]HIDDevice(nil), f.devices...), f.listErr
}

func (f *fakeHID) Describe(d HIDDevice) (HIDDescription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.described = append(f.described, d.Handle)
	desc, ok := f.desc[d.Handle]
	if !ok {
		return HIDDescription{}, errors.New(tagFakeHIDNoDescriptor, "no descriptor")
	}
	return desc, nil
}

func (f *fakeHID) GetData(d HIDDevice, report []byte, items []win32.HidpData) (int, error) {
	n := 0
	for i := 0; i+1 < len(report) && n < len(items); i += 2 {
		idx, v := report[i], report[i+1]
		if idx&0x80 != 0 {
			items[n] = win32.HidpData{DataIndex: uint16(idx & 0x7f), RawValue: 1}
		} else {
			items[n] = win32.HidpData{DataIndex: uint16(idx), RawValue: uint32(v)}
		}
		n++
	}
	return n, nil
}

func (f *fakeHID) Release(d HIDDevice) {
	f.mu.Lock()
	f.released = append(f.released, d.Handle)
	f.mu.Unlock()
}

func padDevice(handle uintptr, path string) HIDDevice {
	return HIDDevice{Handle: handle, Path: path, Vendor: 0x1234, Product: 0x5678, Version: 0x0100, UsagePage: win32.HIDUsagePageGeneric, Usage: win32.HIDUsageGenericGamepad}
}

func TestHIDPadsAreListedAndRead(t *testing.T) {
	buttons, values := flightPad()
	h := &fakeHID{desc: map[uintptr]HIDDescription{
		1: {Name: "Flight Pad", Buttons: buttons, Values: values, MaxItems: 12},
		4: {Name: "Switch", Buttons: buttons[:1], MaxItems: 4},
	}}
	mouse := padDevice(2, `\\?\HID#VID_046D&PID_C52B#mouse`)
	mouse.Usage = 2
	xbox := padDevice(3, `\\?\HID#VID_045E&PID_028E&IG_00#xbox`)
	button := padDevice(4, `\\?\HID#VID_1111&PID_2222#switch`)
	h.set(padDevice(1, `\\?\HID#VID_1234&PID_5678#pad`), mouse, xbox, button)

	r := newRecorder()
	x := &fakeXInput{}
	s := newService(r, x, h)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
	r.drain()
	info := r.added[0]
	if info.Name != "Flight Pad" || info.Buttons != 5 || info.Axes != 3 || info.Hats != 1 || info.Standard || info.Rumble || info.Path != `\\?\HID#VID_1234&PID_5678#pad` {
		t.Errorf("info = %+v", info)
	}
	if g := gamepaddb.GUID(info.GUID).Info(); g.Vendor != 0x1234 || g.Product != 0x5678 || g.Version != 0x0100 {
		t.Errorf("GUID says %+v", g)
	}
	// Neither the mouse (its usage), the XInput interface (its path), nor the device with one button is listed.
	h.mu.Lock()
	for _, handle := range h.described {
		if handle == 2 || handle == 3 {
			t.Errorf("the device %d was described", handle)
		}
	}
	h.mu.Unlock()

	// The first report gives the whole state: the pressed buttons, every axis and the hat.
	s.HIDReports(1, [][]byte{{0x80, 0, 4 | 0x80, 0, 5, 0xff, 6, 0, 7, 0, 8, 2}}, 7*time.Second)
	r.wait(t, "the first report", func() bool { return len(r.events) >= 6 })
	ev := r.takeEvents()
	want := []platform.PadEvent{
		{Pad: info.ID, Time: 7 * time.Second, Kind: platform.PadButton, Index: 0, Value: 1},
		{Pad: info.ID, Time: 7 * time.Second, Kind: platform.PadButton, Index: 4, Value: 1},
		{Pad: info.ID, Time: 7 * time.Second, Kind: platform.PadAxis, Index: 0, Value: 32767},
		{Pad: info.ID, Time: 7 * time.Second, Kind: platform.PadAxis, Index: 1, Value: -32768},
		{Pad: info.ID, Time: 7 * time.Second, Kind: platform.PadAxis, Index: 2, Value: -32768},
		{Pad: info.ID, Time: 7 * time.Second, Kind: platform.PadHat, Index: 0, Value: gamepaddb.HatRight},
	}
	if len(ev) != len(want) {
		t.Fatalf("events = %+v, want %+v", ev, want)
	}
	for i := range want {
		if ev[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, ev[i], want[i])
		}
	}
	// A later report gives what changed: the button let go, Y moved, the hat back to the centre.
	s.HIDReports(1, [][]byte{{5, 0xff, 6, 255, 7, 0, 8, 15}}, 8*time.Second)
	r.wait(t, "the changes", func() bool { return len(r.events) >= 4 })
	ev = r.takeEvents()
	wantLater := []platform.PadEvent{
		{Pad: info.ID, Time: 8 * time.Second, Kind: platform.PadButton, Index: 0, Value: 0},
		{Pad: info.ID, Time: 8 * time.Second, Kind: platform.PadButton, Index: 4, Value: 0},
		{Pad: info.ID, Time: 8 * time.Second, Kind: platform.PadAxis, Index: 1, Value: 32767},
		{Pad: info.ID, Time: 8 * time.Second, Kind: platform.PadHat, Index: 0, Value: 0},
	}
	if len(ev) != len(wantLater) {
		t.Fatalf("events = %+v, want %+v", ev, wantLater)
	}
	for i := range wantLater {
		if ev[i] != wantLater[i] {
			t.Errorf("event %d = %+v, want %+v", i, ev[i], wantLater[i])
		}
	}

	// A report of a device that is not there is ignored.
	s.HIDReports(99, [][]byte{{0x80, 0}}, 9*time.Second)
	// The pad is unplugged.
	h.set()
	s.Rescan()
	r.wait(t, "the removal", func() bool { return len(r.gone) == 1 })
	if r.gone[0] != info.ID {
		t.Errorf("removed %d, added %d", r.gone[0], info.ID)
	}
	h.mu.Lock()
	released := append([]uintptr(nil), h.released...)
	h.mu.Unlock()
	found := false
	for _, handle := range released {
		found = found || handle == 1
	}
	if !found {
		t.Errorf("the description of the pad was not released: %v", released)
	}
}

func TestHIDPadOfAnXInputInterfaceIsListedWithoutXInput(t *testing.T) {
	buttons, values := flightPad()
	h := &fakeHID{desc: map[uintptr]HIDDescription{3: {Name: "Pad", Buttons: buttons, Values: values, MaxItems: 12}}}
	h.set(padDevice(3, `\\?\HID#VID_045E&PID_028E&IG_00#xbox`))
	r := newRecorder()
	s := newService(r, nil, h) // XInput is not there
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	r.wait(t, "the pad", func() bool { return len(r.added) == 1 })
}

func TestHIDListFailureIsLoggedOnce(t *testing.T) {
	h := &fakeHID{listErr: errors.New(tagFakeHIDListFailed, "no list")}
	r := newRecorder()
	s := newService(r, nil, h)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond) // a few scans
	s.Stop()
	if got := len(r.added); got != 0 {
		t.Errorf("%d pads from a failing list", got)
	}
}

func TestHIDDeviceThatCannotBeDescribedIsLeftOut(t *testing.T) {
	h := &fakeHID{desc: map[uintptr]HIDDescription{}}
	h.set(padDevice(1, `\\?\HID#VID_1234&PID_5678#pad`))
	r := newRecorder()
	s := newService(r, nil, h)
	if err := s.Start(r); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	s.Stop()
	if len(r.added) != 0 {
		t.Errorf("a device with no descriptor became %+v", r.added)
	}
}

// The tags of the errors of the fake HID source.
const (
	tagFakeHIDNoDescriptor = "FakeHIDNoDescriptor"
	tagFakeHIDListFailed   = "FakeHIDListFailed"
)
