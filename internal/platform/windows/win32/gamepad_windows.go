//go:build windows

package win32

import (
	"unsafe"

	"github.com/rm4n0s/errors"
)

// The bindings that the gamepad service of Windows needs (step 31.2, SPEC §28.2): XInput, the Raw Input calls that list and read
// the HID devices, the HID parser of hid.dll and the few kernel32 and user32 calls around them.

var hidDLL = &dllTable{name: "hid.dll"}

// XInput is the XInput library of the machine. The newest that is there is used: xinput1_4.dll (Windows 8 and later), else
// xinput1_3.dll or xinput9_1_0.dll.
//
// Flutter: no counterpart
type XInput struct {
	lib                                     *Library
	getState, getStateEx, setState, getCaps *Proc
	getCapsEx, getBattery                   *Proc
}

// LoadXInput loads the library. It fails with LibraryLoadFailed when none of the XInput DLLs is there.
//
// Flutter: no counterpart
func LoadXInput() (*XInput, error) {
	var lib *Library
	var err error
	for _, name := range []string{"xinput1_4.dll", "xinput1_3.dll", "xinput9_1_0.dll"} {
		if lib, err = LoadSystemDLL(name); err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	x := &XInput{lib: lib}
	if x.getState, err = lib.Proc("XInputGetState"); err != nil {
		return nil, err
	}
	if x.setState, err = lib.Proc("XInputSetState"); err != nil {
		return nil, err
	}
	if x.getCaps, err = lib.Proc("XInputGetCapabilities"); err != nil {
		return nil, err
	}
	// The undocumented functions are used when they are there: the guide button, the vendor and the product.
	x.getStateEx, _ = lib.OrdinalProc(100)
	x.getCapsEx, _ = lib.OrdinalProc(108)
	x.getBattery, _ = lib.OptionalProc("XInputGetBatteryInformation")
	return x, nil
}

// Name is the file name of the library that was loaded.
//
// Flutter: no counterpart
func (x *XInput) Name() string { return x.lib.Name() }

// GetState reads the state of the pad of a slot (0 to 3), with the guide button where the library can tell. It returns the
// Win32 error code: ErrorSuccessCode for a pad that is there and ErrorDeviceNotConnected for an empty slot.
//
// Flutter: no counterpart
func (x *XInput) GetState(user uint32, st *XInputState) uint32 {
	p := x.getState
	if x.getStateEx != nil {
		p = x.getStateEx
	}
	r, _ := p.Call(uintptr(user), uintptr(unsafe.Pointer(st)))
	return uint32(r)
}

// SetState sets the motors of a pad (0 stops them) and returns the Win32 error code.
//
// Flutter: no counterpart
func (x *XInput) SetState(user uint32, v *XInputVibration) uint32 {
	r, _ := x.setState.Call(uintptr(user), uintptr(unsafe.Pointer(v)))
	return uint32(r)
}

// GetCapabilities reads what the pad of a slot is, for gamepads only (XInputFlagGamepad), and returns the Win32 error code.
//
// Flutter: no counterpart
func (x *XInput) GetCapabilities(user, flags uint32, c *XInputCapabilities) uint32 {
	r, _ := x.getCaps.Call(uintptr(user), uintptr(flags), uintptr(unsafe.Pointer(c)))
	return uint32(r)
}

// GetCapabilitiesEx reads the vendor, product and version of the pad of a slot. ok is false when the library has no such call
// or the slot is empty.
//
// Flutter: no counterpart
func (x *XInput) GetCapabilitiesEx(user uint32) (c XInputCapabilitiesEx, ok bool) {
	if x.getCapsEx == nil {
		return c, false
	}
	r, _ := x.getCapsEx.Call(1, uintptr(user), 0, uintptr(unsafe.Pointer(&c)))
	return c, uint32(r) == ErrorSuccessCode
}

// GetBatteryInformation reads the battery of the pad of a slot; ok is false when the library cannot tell.
//
// Flutter: no counterpart
func (x *XInput) GetBatteryInformation(user uint32) (b XInputBatteryInformation, ok bool) {
	if x.getBattery == nil {
		return b, false
	}
	r, _ := x.getBattery.Call(uintptr(user), XInputDevTypeGamepad, uintptr(unsafe.Pointer(&b)))
	return b, uint32(r) == ErrorSuccessCode
}

// GetRawInputDeviceList lists the devices of Raw Input.
//
// Flutter: no counterpart
func GetRawInputDeviceList() ([]RawInputDeviceList, error) {
	size := unsafe.Sizeof(RawInputDeviceList{})
	for tries := 0; tries < 3; tries++ {
		var n uint32
		r, errno, err := user32.call("GetRawInputDeviceList", 0, uintptr(unsafe.Pointer(&n)), size)
		if err != nil {
			return nil, err
		}
		if r == ^uintptr(0) {
			return nil, Win32Error("GetRawInputDeviceList", uint32(errno))
		}
		if n == 0 {
			return nil, nil
		}
		list := make([]RawInputDeviceList, n)
		r, _, err = user32.call("GetRawInputDeviceList", uintptr(unsafe.Pointer(&list[0])), uintptr(unsafe.Pointer(&n)), size)
		if err != nil {
			return nil, err
		}
		if r == ^uintptr(0) {
			// ERROR_INSUFFICIENT_BUFFER: a device came between the two calls; n is the new count.
			continue
		}
		return list[:r], nil
	}
	return nil, Win32Error("GetRawInputDeviceList", 122)
}

// rawInputDeviceInfo is GetRawInputDeviceInfoW into a buffer of the size it asks for.
func rawInputDeviceInfo(device uintptr, command uint32, charUnit uintptr) ([]byte, bool) {
	var n uint32
	r, _, err := user32.call("GetRawInputDeviceInfoW", device, uintptr(command), 0, uintptr(unsafe.Pointer(&n)))
	if err != nil || r == ^uintptr(0) || n == 0 {
		return nil, false
	}
	buf := make([]byte, uintptr(n)*charUnit)
	r, _, err = user32.call("GetRawInputDeviceInfoW", device, uintptr(command), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if err != nil || r == ^uintptr(0) {
		return nil, false
	}
	return buf, true
}

// GetRawInputDeviceName is the path of a device of Raw Input ("\\?\HID#VID_045E&PID_028E&IG_00#...").
//
// Flutter: no counterpart
func GetRawInputDeviceName(device uintptr) (string, bool) {
	buf, ok := rawInputDeviceInfo(device, RIDIDeviceName, 2)
	if !ok {
		return "", false
	}
	return UTF16PtrToString((*uint16)(unsafe.Pointer(&buf[0]))), true
}

// GetRawInputDeviceInfo is the description of a device of Raw Input.
//
// Flutter: no counterpart
func GetRawInputDeviceInfo(device uintptr) (RidDeviceInfo, bool) {
	var info RidDeviceInfo
	info.Size = uint32(unsafe.Sizeof(info))
	n := info.Size
	r, _, err := user32.call("GetRawInputDeviceInfoW", device, RIDIDeviceInfo, uintptr(unsafe.Pointer(&info)), uintptr(unsafe.Pointer(&n)))
	if err != nil || r == ^uintptr(0) {
		return RidDeviceInfo{}, false
	}
	return info, true
}

// GetRawInputPreparsedData is the parsed report descriptor of a HID device, the opaque block that the HidP functions take.
//
// Flutter: no counterpart
func GetRawInputPreparsedData(device uintptr) ([]byte, bool) {
	return rawInputDeviceInfo(device, RIDIPreparsedData, 1)
}

// GetRawInputHID reads the HID reports of a WM_INPUT message (the handle is its lParam): the device they come from and each
// report. It answers false for input that is not a HID report or cannot be read.
//
// Flutter: no counterpart
func GetRawInputHID(handle uintptr) (device uintptr, reports [][]byte, ok bool) {
	var n uint32
	hs := unsafe.Sizeof(RawInputHeader{})
	r, _, err := user32.call("GetRawInputData", handle, RIDInput, 0, uintptr(unsafe.Pointer(&n)), hs)
	if err != nil || r == ^uintptr(0) || n < uint32(hs)+8 {
		return 0, nil, false
	}
	buf := make([]byte, n)
	r, _, err = user32.call("GetRawInputData", handle, RIDInput, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)), hs)
	if err != nil || r == ^uintptr(0) {
		return 0, nil, false
	}
	return ParseRawInputHID(buf[:n])
}

// CreateFile opens a file or a device (CreateFileW) with the share mode and the way to open it, and returns its handle.
//
// Flutter: no counterpart
func CreateFile(path string, access, share, disposition uint32) (uintptr, error) {
	p, err := UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	r, errno, err := kern32.call("CreateFileW", uintptr(unsafe.Pointer(p)), uintptr(access), uintptr(share), 0, uintptr(disposition), 0, 0)
	if err != nil {
		return 0, err
	}
	if r == ^uintptr(0) {
		return 0, Win32Error("CreateFileW", uint32(errno))
	}
	return r, nil
}

// HidProductString reads the product name of a HID device from its open handle (HidD_GetProductString).
//
// Flutter: no counterpart
func HidProductString(handle uintptr) (string, bool) {
	buf := make([]uint16, 256)
	if !boolCall(hidDLL, "HidD_GetProductString", handle, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)*2)) {
		return "", false
	}
	return UTF16ToString(buf), true
}

// HidpGetCaps reads what a HID device is and how many buttons and values its input reports have (HidP_GetCaps).
//
// Flutter: no counterpart
func HidpGetCaps(preparsed []byte) (HidpCaps, error) {
	var caps HidpCaps
	if len(preparsed) == 0 {
		return caps, errors.New(TagHidParserFailed, "no parsed report descriptor", "call", "HidP_GetCaps")
	}
	r, _, err := hidDLL.call("HidP_GetCaps", uintptr(unsafe.Pointer(&preparsed[0])), uintptr(unsafe.Pointer(&caps)))
	if err != nil {
		return caps, err
	}
	if uint32(r) != HidPStatusSuccess {
		return caps, errors.New(TagHidParserFailed, "HidP_GetCaps failed", "call", "HidP_GetCaps", "status", uint32(r))
	}
	return caps, nil
}

// HidpInputButtonCaps lists the buttons of the input reports (HidP_GetButtonCaps); n is HidpCaps.NumberInputButtonCaps.
//
// Flutter: no counterpart
func HidpInputButtonCaps(preparsed []byte, n int) ([]HidpButtonCaps, error) {
	if n == 0 {
		return nil, nil
	}
	caps := make([]HidpButtonCaps, n)
	count := uint16(n)
	r, _, err := hidDLL.call("HidP_GetButtonCaps", HidPInput, uintptr(unsafe.Pointer(&caps[0])), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&preparsed[0])))
	if err != nil {
		return nil, err
	}
	if uint32(r) != HidPStatusSuccess {
		return nil, errors.New(TagHidParserFailed, "HidP_GetButtonCaps failed", "call", "HidP_GetButtonCaps", "status", uint32(r))
	}
	return caps[:count], nil
}

// HidpInputValueCaps lists the values (axes, hats) of the input reports (HidP_GetValueCaps); n is
// HidpCaps.NumberInputValueCaps.
//
// Flutter: no counterpart
func HidpInputValueCaps(preparsed []byte, n int) ([]HidpValueCaps, error) {
	if n == 0 {
		return nil, nil
	}
	caps := make([]HidpValueCaps, n)
	count := uint16(n)
	r, _, err := hidDLL.call("HidP_GetValueCaps", HidPInput, uintptr(unsafe.Pointer(&caps[0])), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&preparsed[0])))
	if err != nil {
		return nil, err
	}
	if uint32(r) != HidPStatusSuccess {
		return nil, errors.New(TagHidParserFailed, "HidP_GetValueCaps failed", "call", "HidP_GetValueCaps", "status", uint32(r))
	}
	return caps[:count], nil
}

// HidpMaxDataListLength is the most items an input report can have (HidP_MaxDataListLength).
//
// Flutter: no counterpart
func HidpMaxDataListLength(preparsed []byte) int {
	r, _, err := hidDLL.call("HidP_MaxDataListLength", HidPInput, uintptr(unsafe.Pointer(&preparsed[0])))
	if err != nil {
		return 0
	}
	return int(uint32(r))
}

// HidpGetData reads the items of an input report (HidP_GetData) into data, and returns how many there are. A report that is
// not of this device's descriptor is an error.
//
// Flutter: no counterpart
func HidpGetData(preparsed []byte, data []HidpData, report []byte) (int, error) {
	if len(data) == 0 || len(report) == 0 {
		return 0, nil
	}
	n := uint32(len(data))
	r, _, err := hidDLL.call("HidP_GetData", HidPInput, uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&n)),
		uintptr(unsafe.Pointer(&preparsed[0])), uintptr(unsafe.Pointer(&report[0])), uintptr(len(report)))
	if err != nil {
		return 0, err
	}
	if uint32(r) != HidPStatusSuccess {
		return 0, errors.New(TagHidParserFailed, "HidP_GetData failed", "call", "HidP_GetData", "status", uint32(r))
	}
	return int(n), nil
}

// GetMessage waits for a message of the thread's queue and takes it; it returns 0 for WM_QUIT, -1 for an error and otherwise a
// positive number.
//
// Flutter: no counterpart
func GetMessage(m *Msg, hwnd uintptr, min, max uint32) int32 {
	r, _, err := user32.call("GetMessageW", uintptr(unsafe.Pointer(m)), hwnd, uintptr(min), uintptr(max))
	if err != nil {
		return -1
	}
	return int32(r)
}

// PostQuitMessage puts WM_QUIT in the queue of the calling thread.
//
// Flutter: no counterpart
func PostQuitMessage(code int) { _, _, _ = user32.call("PostQuitMessage", uintptr(code)) } //nolint:errcheck // nothing to do when user32 is missing

// TagHidParserFailed is the tag of the error of a HID parser call (hid.dll) that failed; the metadata are "call" and "status".
const TagHidParserFailed = "HidParserFailed"
