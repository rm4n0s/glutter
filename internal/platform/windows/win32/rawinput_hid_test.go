package win32

import (
	"encoding/binary"
	"testing"
	"unsafe"
)

// rawHID builds the RAWINPUT of a HID report as GetRawInputData gives it: the header, then the size of a report, their number and
// the reports.
func rawHID(device uintptr, size, count uint32, data []byte) []byte {
	hs := int(unsafe.Sizeof(RawInputHeader{}))
	buf := make([]byte, hs+8+len(data))
	binary.NativeEndian.PutUint32(buf[0:], RIMTypeHID)
	binary.NativeEndian.PutUint32(buf[4:], uint32(len(buf)))
	if hs == 24 {
		binary.NativeEndian.PutUint64(buf[8:], uint64(device))
	} else {
		binary.NativeEndian.PutUint32(buf[8:], uint32(device))
	}
	binary.NativeEndian.PutUint32(buf[hs:], size)
	binary.NativeEndian.PutUint32(buf[hs+4:], count)
	copy(buf[hs+8:], data)
	return buf
}

func TestParseRawInputHID(t *testing.T) {
	tests := []struct {
		name    string
		buf     []byte
		ok      bool
		device  uintptr
		reports [][]byte
	}{
		{"one report", rawHID(0x1234, 3, 1, []byte{1, 2, 3}), true, 0x1234, [][]byte{{1, 2, 3}}},
		{"two reports", rawHID(7, 2, 2, []byte{1, 2, 3, 4}), true, 7, [][]byte{{1, 2}, {3, 4}}},
		{"fewer bytes than the reports say", rawHID(7, 4, 2, []byte{1, 2, 3, 4}), false, 0, nil},
		{"no reports", rawHID(7, 4, 0, nil), false, 0, nil},
		{"no size", rawHID(7, 0, 1, []byte{1}), false, 0, nil},
		{"too short", []byte{2, 0, 0, 0}, false, 0, nil},
		{"a mouse report", func() []byte { b := rawHID(7, 1, 1, []byte{1}); b[0] = 0; return b }(), false, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device, reports, ok := ParseRawInputHID(tt.buf)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if device != tt.device || len(reports) != len(tt.reports) {
				t.Fatalf("device %#x reports %v, want %#x %v", device, reports, tt.device, tt.reports)
			}
			for i := range reports {
				if string(reports[i]) != string(tt.reports[i]) {
					t.Errorf("report %d = %v, want %v", i, reports[i], tt.reports[i])
				}
			}
		})
	}
}

// TestGamepadStructSizes checks the sizes that the SDK gives the structures of XInput, Raw Input and the HID parser on a 64-bit
// target, worked out from the headers; the layout test checks every offset against the headers when the mingw-w64 compiler is
// there.
func TestGamepadStructSizes(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("the sizes are those of a 64-bit target")
	}
	tests := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"XINPUT_GAMEPAD", unsafe.Sizeof(XInputGamepad{}), 12},
		{"XINPUT_STATE", unsafe.Sizeof(XInputState{}), 16},
		{"XINPUT_VIBRATION", unsafe.Sizeof(XInputVibration{}), 4},
		{"XINPUT_CAPABILITIES", unsafe.Sizeof(XInputCapabilities{}), 20},
		{"XINPUT_CAPABILITIES_EX", unsafe.Sizeof(XInputCapabilitiesEx{}), 32},
		{"XINPUT_BATTERY_INFORMATION", unsafe.Sizeof(XInputBatteryInformation{}), 2},
		{"RAWINPUTDEVICELIST", unsafe.Sizeof(RawInputDeviceList{}), 16},
		{"RID_DEVICE_INFO", unsafe.Sizeof(RidDeviceInfo{}), 32},
		{"HIDP_CAPS", unsafe.Sizeof(HidpCaps{}), 64},
		{"HIDP_BUTTON_CAPS", unsafe.Sizeof(HidpButtonCaps{}), 72},
		{"HIDP_VALUE_CAPS", unsafe.Sizeof(HidpValueCaps{}), 72},
		{"HIDP_DATA", unsafe.Sizeof(HidpData{}), 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("sizeof(%s) = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestHidCapsUnions(t *testing.T) {
	var b HidpButtonCaps
	b.Union = [16]byte{1, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7, 0, 9, 1}
	if b.UsageMin() != 1 || b.UsageMax() != 4 || b.DataIndexMin() != 7 || b.DataIndexMax() != 0x109 {
		t.Errorf("button caps: %d %d %d %d", b.UsageMin(), b.UsageMax(), b.DataIndexMin(), b.DataIndexMax())
	}
	var v HidpValueCaps
	v.Union = b.Union
	if v.UsageMin() != 1 || v.UsageMax() != 4 || v.DataIndexMin() != 7 || v.DataIndexMax() != 0x109 {
		t.Errorf("value caps: %d %d %d %d", v.UsageMin(), v.UsageMax(), v.DataIndexMin(), v.DataIndexMax())
	}
	if (HidpData{RawValue: 0x100}).On() || !(HidpData{RawValue: 1}).On() {
		t.Error("On reads more than the low byte")
	}
}
