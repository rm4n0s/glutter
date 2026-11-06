//go:build windows

package windows

import (
	"log/slog"
	"os"
	"runtime"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gamepadhid"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/internal/platform/windows/winpad"
)

// The gamepads of Windows (SPEC §28.2, step 31.2): XInput for the Xbox-compatible pads, Raw Input and the HID parser for the
// others, and a message-only window on a thread of its own that hears of devices arriving and leaving (WM_INPUT_DEVICE_CHANGE)
// and receives the reports of the HID pads. The service is in package winpad, which holds no call into Windows; this file is
// the real XInput, the real HID source and the window.

const padClassName = "GlutterGamepads"

var _ winpad.XInputAPI = (*win32.XInput)(nil)

// padService is platform.Gamepads: the service of winpad with the window that feeds it.
type padService struct {
	*winpad.Service
	p *Platform

	mu  sync.Mutex
	mon *padMonitor
}

var _ platform.Gamepads = (*padService)(nil)

// gamepads makes the gamepad service on the first call.
func (p *Platform) gamepads() platform.Gamepads {
	p.padsOnce.Do(func() {
		opts := winpad.Opts{Post: p.Post, Clock: p.pointerClock, Logger: p.log, HID: newRawInputHID(p.log), OpenHID: openHIDTransport,
			DisableDrivers: os.Getenv("GLUTTER_GAMEPAD_HID") == "0"}
		x, err := win32.LoadXInput()
		if err != nil {
			p.log.Info("XInput is not available, so the Xbox pads are read through HID", "err", err)
		} else {
			opts.XInput = x
		}
		p.pads = &padService{Service: winpad.New(opts), p: p}
	})
	return p.pads
}

// Start implements platform.Gamepads: the window that hears of the devices starts, then the service.
func (s *padService) Start(h platform.PadHandler) error {
	s.mu.Lock()
	if s.mon == nil {
		mon, err := startPadMonitor(s.p, s.Service, padMonitorOpts{class: padClassName, register: win32.RegisterRawInputDevices})
		if err != nil {
			// The pads are still found by the look every two seconds; only the reports of the HID pads need the window.
			s.p.log.Warn("the window that hears of the gamepads could not be made, so the HID pads are not read", "err", err)
		} else {
			s.mon = mon
		}
	}
	s.mu.Unlock()
	return s.Service.Start(h)
}

// Stop implements platform.Gamepads.
func (s *padService) Stop() {
	s.Service.Stop()
	s.mu.Lock()
	mon := s.mon
	s.mon = nil
	s.mu.Unlock()
	if mon != nil {
		mon.stop()
	}
}

// padMonitor is the message-only window of a thread of its own.
type padMonitor struct {
	p     *Platform
	svc   *winpad.Service
	hwnd  uintptr
	token uintptr
	done  chan struct{}
	opts  padMonitorOpts
}

// padMonitorOpts are the calls of the monitor that a test replaces: the window class, and the registration of the raw input.
type padMonitorOpts struct {
	class    string
	register func([]win32.RawInputDevice) bool
}

// startPadMonitor starts the thread and waits until its window is there. It fails with the tag of CreateWindowEx when the window
// cannot be made and with PadMonitorRegisterFailed when the raw input cannot be asked for.
func startPadMonitor(p *Platform, svc *winpad.Service, opts padMonitorOpts) (*padMonitor, error) {
	m := &padMonitor{p: p, svc: svc, done: make(chan struct{}), opts: opts}
	ready := make(chan error, 1)
	go m.run(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return m, nil
}

func (m *padMonitor) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(m.done)
	m.token = registerTarget(m)
	defer unregisterTarget(m.token)
	hwnd, err := win32.CreateWindowEx(0, m.opts.class, "", 0, 0, 0, 0, 0, win32.HWNDMessage, 0, m.p.instance, m.token)
	if err != nil {
		ready <- err
		return
	}
	m.hwnd = hwnd
	// The reports of the joysticks, gamepads and multi-axis controllers come to the window even when it is not in front, and so
	// does the news of devices that arrive and leave.
	var devices []win32.RawInputDevice
	for _, usage := range []uint16{win32.HIDUsageGenericJoystick, win32.HIDUsageGenericGamepad, 0x08} {
		devices = append(devices, win32.RawInputDevice{UsagePage: win32.HIDUsagePageGeneric, Usage: usage, Flags: win32.RIDEVInputSink | win32.RIDEVDevNotify, Target: hwnd})
	}
	if !m.opts.register(devices) {
		_ = win32.DestroyWindow(hwnd) // the window is not wanted without the input
		ready <- errors.New(tagPadMonitorRegisterFailed, "RegisterRawInputDevices failed")
		return
	}
	ready <- nil
	var msg win32.Msg
	for win32.GetMessage(&msg, 0, 0, 0) > 0 {
		win32.TranslateMessage(&msg)
		win32.DispatchMessage(&msg)
	}
	for i := range devices {
		devices[i].Flags, devices[i].Target = win32.RIDEVRemove, 0
	}
	m.opts.register(devices)
}

// stop closes the window and waits for the thread.
func (m *padMonitor) stop() {
	if err := win32.PostMessage(m.hwnd, win32.WMClose, 0, 0); err != nil {
		m.p.log.Debug("the window of the gamepads could not be told to close", "err", err)
	}
	<-m.done
}

// handleMessage implements messageTarget.
func (m *padMonitor) handleMessage(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case win32.WMInputDeviceChange:
		m.svc.Rescan()
		return 0
	case win32.WMInput:
		if dev, reports, ok := win32.GetRawInputHID(lparam); ok {
			m.svc.HIDReports(dev, reports, m.p.pointerClock())
		}
		// DefWindowProc frees what Windows kept for the message.
		return win32.DefWindowProc(hwnd, msg, wparam, lparam)
	case win32.WMClose:
		if err := win32.DestroyWindow(hwnd); err != nil {
			m.p.log.Debug("the window of the gamepads could not be destroyed", "err", err)
		}
		return 0
	case win32.WMDestroy:
		win32.PostQuitMessage(0)
		return 0
	}
	return win32.DefWindowProc(hwnd, msg, wparam, lparam)
}

// openHIDTransport opens a HID device as the transport of a HID driver (SPEC §28.2, step 31.3). GLUTTER_GAMEPAD_HID=0 turns the
// drivers off, so that every pad is read through Raw Input.
func openHIDTransport(path string) (gamepadhid.Transport, error) {
	f, err := win32.OpenHidFile(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// rawInputHID is winpad.HIDSource over Raw Input and the HID parser.
type rawInputHID struct {
	log *slog.Logger
	mu  sync.Mutex
	// parsed is the parsed report descriptor of each device that was described, by its handle.
	parsed map[uintptr][]byte
}

var _ winpad.HIDSource = (*rawInputHID)(nil)

func newRawInputHID(log *slog.Logger) *rawInputHID {
	return &rawInputHID{log: log, parsed: map[uintptr][]byte{}}
}

// Devices implements winpad.HIDSource.
func (h *rawInputHID) Devices() ([]winpad.HIDDevice, error) {
	list, err := win32.GetRawInputDeviceList()
	if err != nil {
		return nil, err
	}
	var out []winpad.HIDDevice
	for _, d := range list {
		if d.Type != win32.RIMTypeHID {
			continue
		}
		info, ok := win32.GetRawInputDeviceInfo(d.Device)
		if !ok {
			continue
		}
		path, _ := win32.GetRawInputDeviceName(d.Device)
		out = append(out, winpad.HIDDevice{
			Handle: d.Device, Path: path,
			Vendor: uint16(info.HID.VendorID), Product: uint16(info.HID.ProductID), Version: uint16(info.HID.VersionNumber),
			UsagePage: info.HID.UsagePage, Usage: info.HID.Usage,
		})
	}
	return out, nil
}

// Describe implements winpad.HIDSource.
func (h *rawInputHID) Describe(d winpad.HIDDevice) (winpad.HIDDescription, error) {
	pre, ok := win32.GetRawInputPreparsedData(d.Handle)
	if !ok {
		return winpad.HIDDescription{}, errors.New(tagHidDescribeFailed, "the report descriptor of the device cannot be read", "path", d.Path)
	}
	caps, err := win32.HidpGetCaps(pre)
	if err != nil {
		return winpad.HIDDescription{}, err
	}
	buttons, err := win32.HidpInputButtonCaps(pre, int(caps.NumberInputButtonCaps))
	if err != nil {
		return winpad.HIDDescription{}, err
	}
	values, err := win32.HidpInputValueCaps(pre, int(caps.NumberInputValueCaps))
	if err != nil {
		return winpad.HIDDescription{}, err
	}
	h.mu.Lock()
	h.parsed[d.Handle] = pre
	h.mu.Unlock()
	return winpad.HIDDescription{Name: h.productName(d.Path), Buttons: buttons, Values: values, MaxItems: win32.HidpMaxDataListLength(pre)}, nil
}

// productName reads the product string of a device from the device itself; a device that will not say has none.
func (h *rawInputHID) productName(path string) string {
	if path == "" {
		return ""
	}
	handle, err := win32.CreateFile(path, 0, win32.FileShareRead|win32.FileShareWrite, win32.OpenExisting)
	if err != nil {
		h.log.Debug("a HID device cannot be opened to read its name", "path", path, "err", err)
		return ""
	}
	name, _ := win32.HidProductString(handle)
	if err := win32.CloseHandle(handle); err != nil {
		h.log.Debug("a HID device handle was not closed", "err", err)
	}
	return name
}

// GetData implements winpad.HIDSource.
func (h *rawInputHID) GetData(d winpad.HIDDevice, report []byte, items []win32.HidpData) (int, error) {
	h.mu.Lock()
	pre := h.parsed[d.Handle]
	h.mu.Unlock()
	if pre == nil {
		return 0, errors.New(tagHidDescribeFailed, "the device was not described", "path", d.Path)
	}
	return win32.HidpGetData(pre, items, report)
}

// Release implements winpad.HIDSource.
func (h *rawInputHID) Release(d winpad.HIDDevice) {
	h.mu.Lock()
	delete(h.parsed, d.Handle)
	h.mu.Unlock()
}

// The tags of the errors of the gamepads of Windows (rm4n0s/errors, R5).
const (
	tagPadMonitorRegisterFailed = "PadMonitorRegisterFailed"
	tagHidDescribeFailed        = "HidDescribeFailed"
)
