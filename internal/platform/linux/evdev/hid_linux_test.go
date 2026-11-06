//go:build linux && !android

package evdev

import (
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	rerrors "github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/gamepadhid"
	"github.com/rm4n0s/glutter/internal/platform"
)

// The recorder takes the extras of the pads too.
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

// fakeHID is the transport of a HID driver under test.
type fakeHID struct {
	in       chan []byte
	failRead chan error
	closed   chan struct{}
	once     sync.Once

	mu       sync.Mutex
	written  [][]byte
	features map[byte][]byte
}

func newFakeHID() *fakeHID {
	return &fakeHID{in: make(chan []byte, 64), failRead: make(chan error, 1), closed: make(chan struct{}), features: map[byte][]byte{}}
}

func (f *fakeHID) Read(buf []byte, timeout time.Duration) (int, error) {
	select {
	case r := <-f.in:
		return copy(buf, r), nil
	case err := <-f.failRead:
		return 0, err
	case <-f.closed:
		return 0, rerrors.New(tagFakeHIDClosed, "closed")
	case <-time.After(timeout):
		return 0, nil
	}
}

func (f *fakeHID) Write(data []byte) (int, error) {
	f.mu.Lock()
	f.written = append(f.written, append([]byte(nil), data...))
	f.mu.Unlock()
	return len(data), nil
}

func (f *fakeHID) SendFeature(data []byte) (int, error) { return len(data), nil }

func (f *fakeHID) GetFeature(data []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.features[data[0]]
	if !ok {
		return -1, rerrors.New(tagFakeHIDFeatureMissing, "no such feature report")
	}
	return copy(data, v), nil
}

func (f *fakeHID) Close() error { f.once.Do(func() { close(f.closed) }); return nil }

func (f *fakeHID) writes() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte(nil), f.written...)
}

func (f *fakeHID) isClosed() bool {
	select {
	case <-f.closed:
		return true
	default:
		return false
	}
}

// hidFixture adds to a fixture what the HID drivers need: a hidraw node below an event device, and a transport for it.
type hidFixture struct {
	*fixture
	hidDir string
	opened map[string]*fakeHID
	// openErr, when set, is what opening a hidraw node gives.
	openErr error
	mu      sync.Mutex
}

func newHIDFixture(t *testing.T, pollEvery time.Duration) *hidFixture {
	t.Helper()
	f := &hidFixture{fixture: newFixture(t, pollEvery), opened: map[string]*fakeHID{}}
	f.hidDir = filepath.Join(filepath.Dir(f.dev), "hid")
	if err := os.Mkdir(f.hidDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f.s.hidDir = f.hidDir
	f.s.openHID = func(path string) (gamepadhid.Transport, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.openErr != nil {
			return nil, f.openErr
		}
		tr := newFakeHID()
		f.opened[filepath.Base(path)] = tr
		return tr, nil
	}
	return f
}

// withHidraw makes the hidraw node of an event device in the fake sysfs.
func (f *hidFixture) withHidraw(node, hidraw string) {
	f.t.Helper()
	dir := filepath.Join(f.sys, node, "device", "device", "hidraw", hidraw)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		f.t.Fatal(err)
	}
}

func (f *hidFixture) transport(hidraw string) *fakeHID {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.opened[hidraw]
}

// dualShock4Usb is a DualShock 4 on USB, as the kernel's driver reports it.
func dualShock4Usb() *caps {
	c := dualShock4()
	c.ID.Bus = 3
	return c
}

// ps4Report is the input report of a DualShock 4 on USB with the buttons byte given: the cross is 0x28.
func ps4Report(buttons byte) []byte {
	r := make([]byte, 64)
	r[0] = 1
	r[1], r[2], r[3], r[4] = 128, 128, 128, 128
	r[5] = buttons
	return r
}

func waitWrites(t *testing.T, f *fakeHID, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for len(f.writes()) < n {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d writes, have %d", n, len(f.writes()))
		}
		time.Sleep(time.Millisecond)
	}
}

func TestHIDPadIsServedByItsDriver(t *testing.T) {
	modes(t, func(t *testing.T, base *fixture) {
		f := &hidFixture{fixture: base, opened: map[string]*fakeHID{}}
		f.hidDir = filepath.Join(filepath.Dir(f.dev), "hid")
		f.s.hidDir = f.hidDir
		f.s.openHID = func(path string) (gamepadhid.Transport, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			tr := newFakeHID()
			f.opened[filepath.Base(path)] = tr
			return tr, nil
		}
		f.writeSysfs("event0", dualShock4Usb())
		f.writeNode("event0")
		f.withHidraw("event0", "hidraw3")
		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
		info := f.h.added[0]
		if !info.Standard || info.Name != "PS4 Controller" || filepath.Base(info.Path) != "hidraw3" || !info.Rumble || !info.LED || !info.Sensors ||
			info.Touchpads != 1 || info.RumbleTriggers {
			t.Errorf("info = %+v", info)
		}
		if g := gamepaddb.GUID(info.GUID); g[14] != 'h' {
			t.Errorf("the GUID does not have the signature of the HID drivers: %s", g)
		}
		tr := f.transport("hidraw3")
		if tr == nil {
			t.Fatal("the hidraw node was not opened")
		}
		// The driver switched the pad to the enhanced mode with an effects report.
		waitWrites(t, tr, 1)
		if w := tr.writes()[0]; len(w) != 32 || w[0] != 0x05 {
			t.Errorf("first write = %x", w)
		}

		// A report with the cross down becomes an event of the layout.
		tr.in <- ps4Report(0x28)
		f.h.wait(t, "the cross", func() bool {
			for _, e := range f.h.events {
				if e.Kind == platform.PadButton && e.Index == int(gamepaddb.South) && e.Value == 1 && e.Pad == info.ID {
					return true
				}
			}
			return false
		})

		// Rumble and the light go to the driver.
		if err := f.s.Rumble(info.ID, 0xFFFF, 0, time.Minute); err != nil {
			t.Fatal(err)
		}
		if err := f.s.SetLED(info.ID, 1, 2, 3); err != nil {
			t.Fatal(err)
		}
		waitWrites(t, tr, 3)
		w := tr.writes()
		if w[1][1] != 0x01 || w[1][5] != 0xFF || w[2][1] != 0x02 || w[2][6] != 1 || w[2][7] != 2 || w[2][8] != 3 {
			t.Errorf("writes = %x, %x", w[1][:10], w[2][:10])
		}
		if err := f.s.RumbleTriggers(info.ID, 1, 1, time.Second); err == nil {
			t.Error("a PlayStation 4 pad rumbled its triggers")
		}
		if err := f.s.SetPadSensors(info.ID, true); err != nil {
			t.Errorf("the sensors cannot be turned on: %v", err)
		}
		// The battery and the touchpad come with the reports.
		r := ps4Report(0x08)
		r[1+29] = 0x15 // charging, level 5
		r[1+34] = 0x00 // a finger down on the touchpad
		r[1+35], r[1+36], r[1+37] = 0xC0, 0x03, 0x1C
		tr.in <- r
		f.h.wait(t, "the battery and the finger", func() bool {
			return len(f.h.batteries) > 0 && f.h.batteries[len(f.h.batteries)-1].State == platform.PadCharging && len(f.h.touches) >= 1
		})
		if b := f.h.batteries[len(f.h.batteries)-1]; b.Pad != info.ID || b.Percent != 55 {
			t.Errorf("battery = %+v", b)
		}
		if tc := f.h.touches[0]; tc.Pad != info.ID || !tc.Down || tc.Finger != 0 {
			t.Errorf("touch = %+v", tc)
		}
	})
}

func TestHIDPadGoesWithItsNode(t *testing.T) {
	modes(t, func(t *testing.T, base *fixture) {
		f := &hidFixture{fixture: base, opened: map[string]*fakeHID{}}
		f.s.hidDir = filepath.Join(filepath.Dir(f.dev), "hid")
		f.s.openHID = func(path string) (gamepadhid.Transport, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			tr := newFakeHID()
			f.opened[filepath.Base(path)] = tr
			return tr, nil
		}
		f.writeSysfs("event0", dualShock4Usb())
		f.writeNode("event0")
		f.withHidraw("event0", "hidraw3")
		if err := f.s.Start(f.h); err != nil {
			t.Fatal(err)
		}
		f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
		id := f.h.added[0].ID
		if err := os.Remove(filepath.Join(f.dev, "event0")); err != nil {
			t.Fatal(err)
		}
		f.h.wait(t, "the removal", func() bool { return len(f.h.gone) == 1 })
		if f.h.gone[0] != id {
			t.Errorf("removed %d, want %d", f.h.gone[0], id)
		}
		if !f.transport("hidraw3").isClosed() {
			t.Error("the hidraw node of a pad that went was not closed")
		}
		// The pad can no longer be told to rumble.
		if err := f.s.Rumble(id, 1, 1, time.Second); err == nil {
			t.Error("a pad that went was told to rumble")
		}
	})
}

func TestHIDPadThatIsLostFallsBackToItsEventDevice(t *testing.T) {
	f := newHIDFixture(t, 0)
	f.writeSysfs("event0", dualShock4Usb())
	f.writeNode("event0")
	f.withHidraw("event0", "hidraw3")
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
	f.transport("hidraw3").failRead <- rerrors.New(tagFakeHIDNoDevice, "no such device")
	// The pad is removed, and since its event node is still there it is read through that, as a raw pad.
	f.h.wait(t, "the second pad", func() bool { return len(f.h.added) == 2 && len(f.h.gone) == 1 })
	if f.h.added[1].Standard || f.h.added[1].Buttons == 0 {
		t.Errorf("the pad is not read through its event device: %+v", f.h.added[1])
	}
	if f.h.gone[0] != f.h.added[0].ID {
		t.Errorf("removed %d, want %d", f.h.gone[0], f.h.added[0].ID)
	}
}

func TestHIDDriverThatCannotStartFallsBackToItsEventDevice(t *testing.T) {
	// A Switch Pro controller on USB that does not answer the handshake: the driver gives it up.
	c := padCaps(0x057e, 0x2009, "Pro Controller",
		[]int{btnSouth, btnEast, btnNorth, btnWest, btnTL, btnTR, btnSelect, btnStart, btnMode, btnThumbL, btnThumbR},
		[]int{absX, absY, absRX, absRY, absHat0X, absHat0Y},
		map[int]absInfo{absX: stick, absY: stick, absRX: stick, absRY: stick, absHat0X: digitalHat, absHat0Y: digitalHat})
	f := newHIDFixture(t, 0)
	f.writeSysfs("event0", c)
	f.writeNode("event0")
	f.withHidraw("event0", "hidraw4")
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
	if f.h.added[0].Standard {
		t.Errorf("the pad is served by a driver that gave it up: %+v", f.h.added[0])
	}
	if !f.transport("hidraw4").isClosed() {
		t.Error("the transport of a driver that failed was not closed")
	}
	found := false
	for _, m := range f.log.messages() {
		if m == "the HID driver of a gamepad could not start, so it works through its event device" {
			found = true
		}
	}
	if !found {
		t.Errorf("the failure was not logged: %v", f.log.messages())
	}
}

func TestHIDNodeThatCannotBeOpenedIsReadThroughTheEventDevice(t *testing.T) {
	f := newHIDFixture(t, 0)
	f.openErr = rerrors.NewErr(tagHidrawPermission, syscall.EACCES)
	for _, n := range []string{"event0", "event1"} {
		f.writeSysfs(n, dualShock4Usb())
		f.writeNode(n)
		f.withHidraw(n, "hidraw"+n[len(n)-1:])
	}
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "two pads", func() bool { return len(f.h.added) == 2 })
	for _, info := range f.h.added {
		if info.Standard || info.Buttons == 0 {
			t.Errorf("a pad whose hidraw node is closed to the user is not read through its event device: %+v", info)
		}
	}
	// The user is told once, with the way out.
	n := 0
	for _, m := range f.log.messages() {
		if len(m) > 20 && m[:20] == "a gamepad has a HID " {
			n++
		}
	}
	if n != 1 {
		t.Errorf("the user was told %d times: %v", n, f.log.messages())
	}
}

func TestHIDDriversCanBeTurnedOff(t *testing.T) {
	f := newHIDFixture(t, 0)
	f.s.noHID = true
	f.writeSysfs("event0", dualShock4Usb())
	f.writeNode("event0")
	f.withHidraw("event0", "hidraw3")
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
	if f.h.added[0].Standard || f.transport("hidraw3") != nil {
		t.Error("a driver served a pad while they were off")
	}
}

func TestPadWithoutAHidrawNodeIsReadThroughTheEventDevice(t *testing.T) {
	f := newHIDFixture(t, 0)
	f.writeSysfs("event0", dualShock4Usb())
	f.writeNode("event0")
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
	if f.h.added[0].Standard {
		t.Error("a pad with no hidraw node was served by a driver")
	}
	if err := f.s.SetPadSensors(f.h.added[0].ID, true); err == nil {
		t.Error("the sensors of a pad read through evdev were turned on")
	}
	if err := f.s.SetLED(f.h.added[0].ID, 1, 2, 3); err == nil {
		t.Error("the light of a pad read through evdev was set")
	}
}

func TestPadOfAnotherKindIsNotOfferedToTheDrivers(t *testing.T) {
	f := newHIDFixture(t, 0)
	f.writeSysfs("event0", xbox360())
	f.writeNode("event0")
	f.withHidraw("event0", "hidraw3")
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "the pad", func() bool { return len(f.h.added) == 1 })
	if f.transport("hidraw3") != nil {
		t.Error("the hidraw node of an Xbox 360 pad over USB was opened")
	}
}

// The tags of the errors of the fake transport.
const (
	tagFakeHIDClosed         = "FakeHIDClosed"
	tagFakeHIDFeatureMissing = "FakeHIDFeatureMissing"
	tagFakeHIDNoDevice       = "FakeHIDNoDevice"
)
