//go:build windows && glutter_wine

package windows

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/internal/platform/windows/winpad"
)

// These tests run the gamepad service of Windows under Wine (SPEC §28.8): the real XInput and Raw Input calls, the window that
// hears of devices and the platform's services. Wine has no pad here (its winebus sees /dev/uinput, which the sandbox lacks), so
// what is checked is that every call works and answers as Windows does for an empty machine, and the plumbing of the window is
// driven with messages and a scripted XInput.

func TestWineXInputLoadsAndSlotsAreEmpty(t *testing.T) {
	x, err := win32.LoadXInput()
	if err != nil {
		t.Fatalf("LoadXInput: %v", err)
	}
	t.Logf("XInput is %s", x.Name())
	for user := uint32(0); user < win32.XUserMaxCount; user++ {
		var st win32.XInputState
		code := x.GetState(user, &st)
		if code != win32.ErrorSuccessCode && code != win32.ErrorDeviceNotConnected {
			t.Errorf("GetState(%d) = %d, want success or not connected", user, code)
		}
		var caps win32.XInputCapabilities
		if got := x.GetCapabilities(user, win32.XInputFlagGamepad, &caps); got != code {
			t.Errorf("GetCapabilities(%d) = %d but GetState says %d", user, got, code)
		}
		var v win32.XInputVibration
		if got := x.SetState(user, &v); got != code {
			t.Errorf("SetState(%d) = %d but GetState says %d", user, got, code)
		}
	}
}

func TestWineRawInputListsDevicesWithoutFailing(t *testing.T) {
	list, err := win32.GetRawInputDeviceList()
	if err != nil {
		t.Fatalf("GetRawInputDeviceList: %v", err)
	}
	hid := 0
	for _, d := range list {
		if d.Type != win32.RIMTypeHID {
			continue
		}
		hid++
		info, ok := win32.GetRawInputDeviceInfo(d.Device)
		if !ok {
			t.Errorf("no info for the HID device %#x", d.Device)
			continue
		}
		name, _ := win32.GetRawInputDeviceName(d.Device)
		t.Logf("HID device %q vendor %04x product %04x usage %02x:%02x", name, info.HID.VendorID, info.HID.ProductID, info.HID.UsagePage, info.HID.Usage)
	}
	t.Logf("%d devices of Raw Input, %d of them HID", len(list), hid)

	src := newRawInputHID(slog.Default())
	devices, err := src.Devices()
	if err != nil {
		t.Fatalf("Devices: %v", err)
	}
	for _, d := range devices {
		// Each HID device that the machine has is described or fails with an error, never a panic.
		if desc, err := src.Describe(d); err == nil {
			t.Logf("%s: %d button caps, %d value caps", d.Path, len(desc.Buttons), len(desc.Values))
			src.Release(d)
		}
	}
}

func TestWineGamepadServiceOfThePlatformStartsAndStops(t *testing.T) {
	p, _ := newTestPlatform(t)
	svc := p.Services()
	if svc == nil {
		t.Fatal("the platform has no services")
	}
	pads := svc.Gamepads()
	if pads == nil {
		t.Fatal("the platform has no gamepad service")
	}
	if pads.MappingPlatform() != "Windows" {
		t.Errorf("MappingPlatform() = %q", pads.MappingPlatform())
	}
	h := &padRecorder{}
	for round := 0; round < 2; round++ {
		if err := pads.Start(h); err != nil {
			t.Fatalf("Start (round %d): %v", round, err)
		}
		time.Sleep(150 * time.Millisecond)
		pads.Stop()
	}
	if err := pads.Rumble(1, 1, 1, time.Second); err == nil {
		t.Error("a pad that is not there rumbled")
	}
}

func TestWineWindowHearsOfDevicesAndFeedsTheService(t *testing.T) {
	p, _ := newTestPlatform(t)
	x := &scriptedXInput{}
	posts := make(chan func(), 64)
	svc := winpad.New(winpad.Opts{
		Post: func(fn func()) { posts <- fn }, XInput: x,
		// The slots are looked at when the window says so, not by the clock.
		PollEvery: time.Hour, ScanEvery: time.Hour,
	})
	h := &padRecorder{}
	if err := svc.Start(h); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(svc.Stop)
	mon, err := startPadMonitor(p, svc, padMonitorOpts{class: padClassName, register: win32.RegisterRawInputDevices})
	if err != nil {
		t.Fatalf("startPadMonitor: %v", err)
	}
	stopped := false
	stop := func() {
		if !stopped {
			stopped = true
			mon.stop()
		}
	}
	t.Cleanup(stop)

	x.plug()
	// Windows tells the window that a device arrived (SendMessage runs the window procedure on the monitor's thread).
	win32.SendMessage(mon.hwnd, win32.WMInputDeviceChange, win32.GIDCArrival, 0)
	deadline := time.After(5 * time.Second)
	for h.addedCount() == 0 {
		select {
		case fn := <-posts:
			fn()
		case <-deadline:
			t.Fatal("the pad of the scripted XInput did not appear after the message")
		}
	}
	if h.firstAdded().Path != "XInput#0" {
		t.Errorf("the pad = %+v", h.firstAdded())
	}
	stop()
	// After the window is gone a message to it does nothing and nothing hangs.
	win32.SendMessage(mon.hwnd, win32.WMInputDeviceChange, win32.GIDCArrival, 0)
}

func TestWineGamepadErrorsHaveTheirRoutes(t *testing.T) {
	p, _ := newTestPlatform(t)
	svc := winpad.New(winpad.Opts{Post: func(func()) {}})
	checkRoute := func(t *testing.T, err error, tag, route string) {
		t.Helper()
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
		}
		if appErr.Tag != tag || !appErr.HasRoute(route) {
			t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
		}
	}
	t.Run("a window of a class that is not there", func(t *testing.T) {
		_, err := startPadMonitor(p, svc, padMonitorOpts{class: "NoSuchGamepadClass", register: win32.RegisterRawInputDevices})
		checkRoute(t, err, win32.TagWin32WindowCreateFailed, "padMonitor.run->CreateWindowEx.Win32WindowCreateFailed")
	})
	t.Run("raw input that cannot be asked for", func(t *testing.T) {
		_, err := startPadMonitor(p, svc, padMonitorOpts{class: padClassName, register: func([]win32.RawInputDevice) bool { return false }})
		checkRoute(t, err, tagPadMonitorRegisterFailed, "padMonitor.run.PadMonitorRegisterFailed")
	})
	src := newRawInputHID(slog.Default())
	missing := winpad.HIDDevice{Handle: 0xdead, Path: "nothing"}
	t.Run("a device that is not there is not described", func(t *testing.T) {
		_, err := src.Describe(missing)
		checkRoute(t, err, tagHidDescribeFailed, "rawInputHID.Describe.HidDescribeFailed")
	})
	t.Run("a device that was not described is not read", func(t *testing.T) {
		_, err := src.GetData(missing, []byte{1}, make([]win32.HidpData, 1))
		checkRoute(t, err, tagHidDescribeFailed, "rawInputHID.GetData.HidDescribeFailed")
	})
}

// padRecorder is a platform.PadHandler that keeps what it is told.
type padRecorder struct {
	mu    sync.Mutex
	added []platform.PadInfo
}

func (r *padRecorder) PadAdded(info platform.PadInfo) {
	r.mu.Lock()
	r.added = append(r.added, info)
	r.mu.Unlock()
}
func (r *padRecorder) PadRemoved(platform.PadID)     {}
func (r *padRecorder) PadEvents([]platform.PadEvent) {}

func (r *padRecorder) addedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.added)
}

func (r *padRecorder) firstAdded() platform.PadInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.added[0]
}

// scriptedXInput has one pad in slot 0 once plug is called.
type scriptedXInput struct {
	mu      sync.Mutex
	present bool
}

func (x *scriptedXInput) plug() {
	x.mu.Lock()
	x.present = true
	x.mu.Unlock()
}

func (x *scriptedXInput) code() uint32 {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.present {
		return win32.ErrorSuccessCode
	}
	return win32.ErrorDeviceNotConnected
}

func (x *scriptedXInput) slot(user uint32) uint32 {
	if user != 0 {
		return win32.ErrorDeviceNotConnected
	}
	return x.code()
}

func (x *scriptedXInput) GetState(user uint32, st *win32.XInputState) uint32 { return x.slot(user) }
func (x *scriptedXInput) SetState(user uint32, v *win32.XInputVibration) uint32 {
	return x.slot(user)
}
func (x *scriptedXInput) GetCapabilities(user, flags uint32, c *win32.XInputCapabilities) uint32 {
	*c = win32.XInputCapabilities{Type: win32.XInputDevTypeGamepad, SubType: win32.XInputDevSubTypeGamepad}
	return x.slot(user)
}
func (x *scriptedXInput) GetCapabilitiesEx(uint32) (win32.XInputCapabilitiesEx, bool) {
	return win32.XInputCapabilitiesEx{}, false
}
func (x *scriptedXInput) GetBatteryInformation(uint32) (win32.XInputBatteryInformation, bool) {
	return win32.XInputBatteryInformation{}, false
}
